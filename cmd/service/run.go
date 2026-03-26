package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

const (
	usageExitCode       = 10
	unavailableExitCode = 20
)

type Options struct {
	ServiceManager api.ServiceManager
	Output         io.Writer
	Error          io.Writer
}

type resolvedOptions struct {
	serviceManager api.ServiceManager
	output         io.Writer
	errorLog       io.Writer
}

type intentArgs struct {
	action      *string
	data        *string
	mimeType    *string
	launchFlags int32
	component   *string
	categories  []string
}

func Main(ctx context.Context, argv []string, stdout io.Writer, stderr io.Writer) int {
	conn, err := libbinder.Open(libbinder.Config{})
	if err != nil {
		fmt.Fprintln(stderr, "service: Unable to get default service manager!")
		return unavailableExitCode
	}
	defer func() { _ = conn.Close() }()

	return Run(ctx, argv, Options{
		ServiceManager: conn.ServiceManager(),
		Output:         stdout,
		Error:          stderr,
	})
}

func Run(ctx context.Context, argv []string, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := resolveOptions(opts)
	if resolved.serviceManager == nil {
		fmt.Fprintln(resolved.errorLog, "service: Unable to get default service manager!")
		return unavailableExitCode
	}

	args, wantsUsage, usageCode := parseGlobalOptions(argv, resolved.errorLog)
	if wantsUsage || len(args) == 0 {
		printUsage(resolved.output)
		return usageCode
	}

	switch args[0] {
	case "list":
		return runList(ctx, resolved)
	case "check":
		return runCheck(ctx, args[1:], resolved)
	case "call":
		return runCall(ctx, args[1:], resolved)
	default:
		fmt.Fprintf(resolved.errorLog, "service: Unknown command %s\n", args[0])
		printUsage(resolved.output)
		return usageExitCode
	}
}

func resolveOptions(opts Options) resolvedOptions {
	return resolvedOptions{
		serviceManager: opts.ServiceManager,
		output:         writerOrDiscard(opts.Output),
		errorLog:       writerOrDiscard(opts.Error),
	}
}

func parseGlobalOptions(argv []string, errorLog io.Writer) ([]string, bool, int) {
	args := argv
	usageCode := 0
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-h", "-?":
			return args[1:], true, 0
		default:
			fmt.Fprintf(errorLog, "service: Unknown option %s\n", args[0])
			usageCode = usageExitCode
			return args[1:], true, usageCode
		}
	}
	return args, false, usageCode
}

func runList(ctx context.Context, opts resolvedOptions) int {
	services, err := opts.serviceManager.ListServices(ctx, api.DumpPriorityAll)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Failed to list services: %v\n", err)
		return unavailableExitCode
	}

	fmt.Fprintf(opts.output, "Found %d services:\n", len(services))
	for i, name := range services {
		descriptor := ""
		service, err := opts.serviceManager.CheckService(ctx, name)
		if err == nil && service != nil {
			descriptor, _ = service.Descriptor(ctx)
		}
		fmt.Fprintf(opts.output, "%d\t%s: [%s]\n", i, name, descriptor)
	}
	return 0
}

func runCheck(ctx context.Context, args []string, opts resolvedOptions) int {
	if len(args) == 0 {
		fmt.Fprintln(opts.errorLog, "service: No service specified for check")
		printUsage(opts.output)
		return usageExitCode
	}
	service, err := opts.serviceManager.CheckService(ctx, args[0])
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Failed to check %s: %v\n", args[0], err)
		return unavailableExitCode
	}
	state := "not found"
	if service != nil {
		state = "found"
	}
	fmt.Fprintf(opts.output, "Service %s: %s\n", args[0], state)
	return 0
}

func runCall(ctx context.Context, args []string, opts resolvedOptions) int {
	if len(args) == 0 {
		fmt.Fprintln(opts.errorLog, "service: No service specified for call")
		printUsage(opts.output)
		return usageExitCode
	}
	if len(args) == 1 {
		fmt.Fprintln(opts.errorLog, "service: No code specified for call")
		printUsage(opts.output)
		return usageExitCode
	}

	serviceName := args[0]
	code, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Invalid transaction code %q\n", args[1])
		printUsage(opts.output)
		return usageExitCode
	}

	service, err := opts.serviceManager.CheckService(ctx, serviceName)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Failed to resolve %s: %v\n", serviceName, err)
		return unavailableExitCode
	}
	if service == nil {
		fmt.Fprintf(opts.errorLog, "Service %s does not exist\n", serviceName)
		return usageExitCode
	}

	descriptor, err := service.Descriptor(ctx)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Failed to resolve descriptor for %s: %v\n", serviceName, err)
		return unavailableExitCode
	}
	if descriptor == "" {
		fmt.Fprintf(opts.errorLog, "Service %s does not exist\n", serviceName)
		return usageExitCode
	}

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(descriptor); err != nil {
		fmt.Fprintf(opts.errorLog, "service: Failed to write interface token: %v\n", err)
		return unavailableExitCode
	}

	cleanup, err := encodeCallArgs(data, args[2:], opts.errorLog, opts.output)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return usageExitCode
	}

	reply, err := service.Transact(ctx, uint32(code), data, api.FlagNone)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "service: Transaction failed: %v\n", err)
		return unavailableExitCode
	}
	fmt.Fprintf(opts.output, "Result: %s\n", formatParcel(reply))
	return 0
}

func encodeCallArgs(p *api.Parcel, args []string, errorLog io.Writer, output io.Writer) (func(), error) {
	closers := make([]io.Closer, 0)
	cleanup := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i].Close()
		}
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "i32":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for i32")
				printUsage(output)
				return cleanup, errors.New("missing i32")
			}
			value, err := strconv.ParseInt(args[i+1], 10, 32)
			if err != nil {
				fmt.Fprintf(errorLog, "service: Invalid i32 value %q\n", args[i+1])
				printUsage(output)
				return cleanup, err
			}
			if err := p.WriteInt32(int32(value)); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode i32: %v\n", err)
				return cleanup, err
			}
			i++
		case "i64":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for i64")
				printUsage(output)
				return cleanup, errors.New("missing i64")
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				fmt.Fprintf(errorLog, "service: Invalid i64 value %q\n", args[i+1])
				printUsage(output)
				return cleanup, err
			}
			if err := p.WriteInt64(value); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode i64: %v\n", err)
				return cleanup, err
			}
			i++
		case "f":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for f")
				printUsage(output)
				return cleanup, errors.New("missing float32")
			}
			value, err := strconv.ParseFloat(args[i+1], 32)
			if err != nil {
				fmt.Fprintf(errorLog, "service: Invalid f value %q\n", args[i+1])
				printUsage(output)
				return cleanup, err
			}
			if err := p.WriteFloat32(float32(value)); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode f: %v\n", err)
				return cleanup, err
			}
			i++
		case "d":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for d")
				printUsage(output)
				return cleanup, errors.New("missing float64")
			}
			value, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				fmt.Fprintf(errorLog, "service: Invalid d value %q\n", args[i+1])
				printUsage(output)
				return cleanup, err
			}
			if err := p.WriteFloat64(value); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode d: %v\n", err)
				return cleanup, err
			}
			i++
		case "s16":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for s16")
				printUsage(output)
				return cleanup, errors.New("missing string16")
			}
			if err := p.WriteString(args[i+1]); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode s16: %v\n", err)
				return cleanup, err
			}
			i++
		case "null":
			if err := p.WriteNullStrongBinder(); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode null binder: %v\n", err)
				return cleanup, err
			}
		case "fd":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for fd")
				printUsage(output)
				return cleanup, errors.New("missing fd path")
			}
			file, err := os.Open(args[i+1])
			if err != nil {
				fmt.Fprintf(errorLog, "service: Failed to open %s: %v\n", args[i+1], err)
				return cleanup, err
			}
			closers = append(closers, file)
			if err := p.WriteFileDescriptor(api.NewFileDescriptor(int(file.Fd()))); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode fd: %v\n", err)
				return cleanup, err
			}
			i++
		case "nfd":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for nfd")
				printUsage(output)
				return cleanup, errors.New("missing numeric fd")
			}
			fd, err := strconv.Atoi(args[i+1])
			if err != nil || fd < 0 {
				fmt.Fprintf(errorLog, "service: Invalid nfd value %q\n", args[i+1])
				printUsage(output)
				return cleanup, errors.New("invalid numeric fd")
			}
			if err := p.WriteFileDescriptor(api.NewFileDescriptor(fd)); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode nfd: %v\n", err)
				return cleanup, err
			}
			closers = append(closers, fileDescriptorCloser(fd))
			i++
		case "afd":
			if i+1 >= len(args) {
				fmt.Fprintln(errorLog, "service: Missing value for afd")
				printUsage(output)
				return cleanup, errors.New("missing afd path")
			}
			file, err := openAnonymousFile(args[i+1])
			if err != nil {
				fmt.Fprintf(errorLog, "service: Failed to create anonymous fd from %s: %v\n", args[i+1], err)
				return cleanup, err
			}
			closers = append(closers, file)
			if err := p.WriteFileDescriptor(api.NewFileDescriptor(int(file.Fd()))); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode afd: %v\n", err)
				return cleanup, err
			}
			i++
		case "intent":
			intent, err := parseIntentArgs(args[i+1:], errorLog, output)
			if err != nil {
				return cleanup, err
			}
			if err := writeIntentArgs(p, intent); err != nil {
				fmt.Fprintf(errorLog, "service: Failed to encode intent: %v\n", err)
				return cleanup, err
			}
			return cleanup, nil
		default:
			fmt.Fprintf(errorLog, "service: Unknown argument type %s\n", args[i])
			printUsage(output)
			return cleanup, errors.New("unknown argument type")
		}
	}
	return cleanup, nil
}

func parseIntentArgs(args []string, errorLog io.Writer, output io.Writer) (intentArgs, error) {
	var out intentArgs
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			continue
		}
		switch key {
		case "action":
			out.action = stringPtr(value)
		case "data":
			out.data = stringPtr(value)
		case "type":
			out.mimeType = stringPtr(value)
		case "launchFlags":
			parsed, err := strconv.ParseInt(value, 0, 32)
			if err != nil {
				fmt.Fprintf(errorLog, "service: Invalid intent launchFlags %q\n", value)
				printUsage(output)
				return intentArgs{}, err
			}
			out.launchFlags = int32(parsed)
		case "component":
			out.component = stringPtr(value)
		case "categories":
			if value == "" {
				out.categories = nil
			} else {
				out.categories = strings.Split(value, ",")
			}
		}
	}
	return out, nil
}

func writeIntentArgs(p *api.Parcel, intent intentArgs) error {
	if err := p.WriteNullableString(intent.action); err != nil {
		return err
	}
	if err := p.WriteNullableString(intent.data); err != nil {
		return err
	}
	if err := p.WriteNullableString(intent.mimeType); err != nil {
		return err
	}
	if err := p.WriteInt32(intent.launchFlags); err != nil {
		return err
	}
	if err := p.WriteNullableString(intent.component); err != nil {
		return err
	}
	if err := p.WriteInt32(int32(len(intent.categories))); err != nil {
		return err
	}
	for _, category := range intent.categories {
		if err := p.WriteString(category); err != nil {
			return err
		}
	}
	return p.WriteInt32(-1)
}

func openAnonymousFile(path string) (*os.File, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()

	tmp, err := os.CreateTemp("", "libbinder-go-service-afd-*")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		name := tmp.Name()
		_ = tmp.Close()
		_ = os.Remove(name)
		return nil, err
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		name := tmp.Name()
		_ = tmp.Close()
		_ = os.Remove(name)
		return nil, err
	}
	if err := os.Remove(tmp.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = tmp.Close()
		return nil, err
	}
	return tmp, nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: service [-h|-?]")
	fmt.Fprintln(w, "       service list")
	fmt.Fprintln(w, "       service check SERVICE")
	fmt.Fprintln(w, "       service call SERVICE CODE [i32 N | i64 N | f N | d N | s16 STR | null | fd FILE | nfd FD | afd FILE | intent KEY=VALUE ...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "intent keys: action=..., data=..., type=..., launchFlags=..., component=..., categories=a,b,c")
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func formatParcel(p *api.Parcel) string {
	if p == nil {
		return "<nil>"
	}
	payload := p.Bytes()
	objects := p.Objects()

	var b strings.Builder
	fmt.Fprintf(&b, "Parcel{size=%d, objects=%d", len(payload), len(objects))
	if len(payload) == 0 && len(objects) == 0 {
		b.WriteString("}")
		return b.String()
	}
	if len(payload) != 0 {
		b.WriteString("\n  data:\n")
		for _, line := range strings.Split(strings.TrimRight(hex.Dump(payload), "\n"), "\n") {
			b.WriteString("    ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if len(objects) == 0 {
		b.WriteString("  objects: []\n")
	} else {
		b.WriteString("  objects:\n")
		for i, obj := range objects {
			fmt.Fprintf(&b, "    [%d] kind=%s offset=%d length=%d handle=%d stability=%s\n",
				i, objectKindString(obj.Kind), obj.Offset, obj.Length, obj.Handle, obj.Stability)
		}
	}
	b.WriteString("}")
	return b.String()
}

func objectKindString(kind api.ObjectKind) string {
	switch kind {
	case api.ObjectNullBinder:
		return "null-binder"
	case api.ObjectStrongBinder:
		return "strong-binder"
	case api.ObjectWeakBinder:
		return "weak-binder"
	case api.ObjectFileDescriptor:
		return "file-descriptor"
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

type fileDescriptorCloser int

func (f fileDescriptorCloser) Close() error {
	fd := int(f)
	if fd < 0 {
		return nil
	}
	if err := syscall.Close(fd); err != nil && !errors.Is(err, syscall.EBADF) {
		return err
	}
	return nil
}

func stringPtr(v string) *string {
	return &v
}

func duplicateFD(fd int) (*os.File, error) {
	dup, err := syscall.Dup(fd)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(dup), filepath.Base(fmt.Sprintf("fd-%d", fd))), nil
}
