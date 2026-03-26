package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/binderdebug"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
)

const (
	usageExitCode       = 10
	unavailableExitCode = 20
	defaultTimeout      = 10 * time.Second
)

type dumpType uint32

const (
	typeDump dumpType = 1 << iota
	typePID
	typeStability
	typeThread
	typeClients
)

type binderDebugReader interface {
	GetPIDInfo(pid int) (binderdebug.PIDInfo, error)
	GetClientPIDs(callerPID int, servicePID int, handle uint32) ([]int, error)
}

type Options struct {
	ServiceManager api.ServiceManager
	Output         io.Writer
	Error          io.Writer
	DebugReader    binderDebugReader
}

type resolvedOptions struct {
	serviceManager api.ServiceManager
	output         io.Writer
	errorLog       io.Writer
	debugReader    binderDebugReader
}

type dumpSession struct {
	readFile *os.File
	done     chan error
}

func Main(ctx context.Context, argv []string, stdout io.Writer, stderr io.Writer) int {
	conn, err := libbinder.Open(libbinder.Config{})
	if err != nil {
		fmt.Fprintln(stderr, "dumpsys: Unable to get default service manager!")
		return unavailableExitCode
	}
	defer func() { _ = conn.Close() }()

	return Run(ctx, argv, Options{
		ServiceManager: conn.ServiceManager(),
		Output:         stdout,
		Error:          stderr,
		DebugReader:    binderdebug.NewReader(kernel.DefaultDriverPath),
	})
}

func Run(ctx context.Context, argv []string, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := resolveOptions(opts)
	if resolved.serviceManager == nil {
		fmt.Fprintln(resolved.errorLog, "dumpsys: Unable to get default service manager!")
		return unavailableExitCode
	}

	cfg, code := parseArgs(argv, resolved.errorLog)
	if code != 0 || cfg.showHelp {
		printUsage(resolved.errorLog)
		if cfg.showHelp {
			return 0
		}
		return code
	}

	services := append([]string(nil), cfg.services...)
	args := append([]string(nil), cfg.serviceArgs...)
	if len(services) == 0 || cfg.showListOnly {
		list, err := listServices(resolved.serviceManager, ctx, cfg.priorityFlags, cfg.asProto)
		if err != nil {
			fmt.Fprintf(resolved.errorLog, "dumpsys: Failed to list services: %v\n", err)
			return unavailableExitCode
		}
		services = list
		args = setServiceArgs(args, cfg.asProto, cfg.priorityFlags)
	}

	if len(services) > 1 || cfg.showListOnly {
		fmt.Fprintln(resolved.output, "Currently running services:")
		for _, name := range services {
			service, err := resolved.serviceManager.CheckService(ctx, name)
			if err != nil || service == nil {
				continue
			}
			suffix := ""
			if isSkipped(cfg.skippedServices, name) {
				suffix = " (skipped)"
			}
			fmt.Fprintf(resolved.output, "  %s%s\n", name, suffix)
		}
	}
	if cfg.showListOnly {
		return 0
	}

	for _, serviceName := range services {
		if isSkipped(cfg.skippedServices, serviceName) {
			continue
		}

		session, err := startDumpSession(ctx, resolved, cfg.dumpTypes, serviceName, args)
		if err != nil {
			continue
		}

		addSeparator := len(services) > 1
		if addSeparator {
			writeDumpHeader(resolved.output, serviceName, cfg.priorityFlags)
		}

		elapsed, _, status := writeDump(resolved.output, resolved.errorLog, session.readFile, serviceName, cfg.timeout, cfg.asProto)
		if errors.Is(status, errDumpTimeout) {
			fmt.Fprintf(resolved.output, "\n*** SERVICE '%s' DUMP TIMEOUT (%dms) EXPIRED ***\n\n", serviceName, cfg.timeout.Milliseconds())
		}

		if addSeparator {
			writeDumpFooter(resolved.output, serviceName, elapsed)
		}

		stopDumpSession(session, !errors.Is(status, errDumpTimeout))
	}

	return 0
}

type config struct {
	showHelp        bool
	showListOnly    bool
	skipServices    bool
	asProto         bool
	timeout         time.Duration
	priorityFlags   api.DumpFlags
	dumpTypes       dumpType
	services        []string
	serviceArgs     []string
	skippedServices []string
}

func resolveOptions(opts Options) resolvedOptions {
	debugReader := opts.DebugReader
	if debugReader == nil {
		debugReader = binderdebug.NewReader(kernel.DefaultDriverPath)
	}
	return resolvedOptions{
		serviceManager: opts.ServiceManager,
		output:         writerOrDiscard(opts.Output),
		errorLog:       writerOrDiscard(opts.Error),
		debugReader:    debugReader,
	}
}

func parseArgs(argv []string, errorLog io.Writer) (config, int) {
	cfg := config{
		timeout:       defaultTimeout,
		priorityFlags: api.DumpPriorityAll,
	}

	i := 0
	for i < len(argv) {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		switch arg {
		case "--help", "-h", "-?":
			cfg.showHelp = true
			return cfg, 0
		case "-l":
			cfg.showListOnly = true
		case "--skip":
			cfg.skipServices = true
		case "--proto":
			cfg.asProto = true
		case "--dump":
			cfg.dumpTypes |= typeDump
		case "--pid":
			cfg.dumpTypes |= typePID
		case "--stability":
			cfg.dumpTypes |= typeStability
		case "--thread":
			cfg.dumpTypes |= typeThread
		case "--clients":
			cfg.dumpTypes |= typeClients
		case "-t":
			i++
			if i >= len(argv) {
				fmt.Fprintln(errorLog, "Error: missing timeout(seconds) value")
				return cfg, usageExitCode
			}
			seconds, err := strconv.Atoi(argv[i])
			if err != nil || seconds <= 0 {
				fmt.Fprintf(errorLog, "Error: invalid timeout(seconds) number: '%s'\n", argv[i])
				return cfg, usageExitCode
			}
			cfg.timeout = time.Duration(seconds) * time.Second
		case "-T":
			i++
			if i >= len(argv) {
				fmt.Fprintln(errorLog, "Error: missing timeout(milliseconds) value")
				return cfg, usageExitCode
			}
			ms, err := strconv.Atoi(argv[i])
			if err != nil || ms <= 0 {
				fmt.Fprintf(errorLog, "Error: invalid timeout(milliseconds) number: '%s'\n", argv[i])
				return cfg, usageExitCode
			}
			cfg.timeout = time.Duration(ms) * time.Millisecond
		case "--priority":
			i++
			if i >= len(argv) {
				fmt.Fprintln(errorLog, "Error: missing priority level")
				return cfg, usageExitCode
			}
			flags, ok := priorityFromString(argv[i])
			if !ok {
				return cfg, usageExitCode
			}
			cfg.priorityFlags = flags
		default:
			return cfg, usageExitCode
		}
		i++
	}

	if cfg.dumpTypes == 0 {
		cfg.dumpTypes = typeDump
	}

	for ; i < len(argv); i++ {
		if cfg.skipServices {
			cfg.skippedServices = append(cfg.skippedServices, argv[i])
			continue
		}
		if len(cfg.services) == 0 {
			cfg.services = append(cfg.services, argv[i])
			continue
		}
		cfg.serviceArgs = append(cfg.serviceArgs, argv[i])
		if argv[i] == "--proto" {
			cfg.asProto = true
		}
	}

	if (cfg.skipServices && len(cfg.skippedServices) == 0) || (cfg.showListOnly && (len(cfg.services) != 0 || len(cfg.skippedServices) != 0)) {
		return cfg, usageExitCode
	}
	return cfg, 0
}

func listServices(sm api.ServiceManager, ctx context.Context, priorityFlags api.DumpFlags, asProto bool) ([]string, error) {
	services, err := sm.ListServices(ctx, priorityFlags)
	if err != nil {
		return nil, err
	}
	sort.Strings(services)
	if !asProto {
		return services, nil
	}

	protoServices, err := sm.ListServices(ctx, api.DumpProto)
	if err != nil {
		return nil, err
	}
	sort.Strings(protoServices)
	protoSet := make(map[string]struct{}, len(protoServices))
	for _, name := range protoServices {
		protoSet[name] = struct{}{}
	}
	filtered := make([]string, 0, len(services))
	for _, name := range services {
		if _, ok := protoSet[name]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered, nil
}

func setServiceArgs(args []string, asProto bool, priorityFlags api.DumpFlags) []string {
	out := append([]string(nil), args...)
	if asProto {
		out = append([]string{"--proto"}, out...)
	}
	if priorityFlags == api.DumpPriorityAll || priorityFlags == api.DumpPriorityNormal || priorityFlags == api.DumpPriorityDefault {
		out = append([]string{"-a"}, out...)
	}
	switch priorityFlags {
	case api.DumpPriorityCritical, api.DumpPriorityHigh, api.DumpPriorityNormal:
		out = append([]string{"--priority", priorityToString(priorityFlags)}, out...)
	}
	return out
}

func isSkipped(skipped []string, service string) bool {
	for _, candidate := range skipped {
		if candidate == service {
			return true
		}
	}
	return false
}

func priorityFromString(s string) (api.DumpFlags, bool) {
	switch s {
	case "CRITICAL":
		return api.DumpPriorityCritical, true
	case "HIGH":
		return api.DumpPriorityHigh, true
	case "NORMAL":
		return api.DumpPriorityNormal, true
	default:
		return 0, false
	}
}

func priorityToString(flags api.DumpFlags) string {
	switch flags {
	case api.DumpPriorityCritical:
		return "CRITICAL"
	case api.DumpPriorityHigh:
		return "HIGH"
	case api.DumpPriorityNormal:
		return "NORMAL"
	default:
		return ""
	}
}

func startDumpSession(ctx context.Context, opts resolvedOptions, types dumpType, serviceName string, args []string) (*dumpSession, error) {
	service, err := opts.serviceManager.CheckService(ctx, serviceName)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "Error with service '%s' while resolving: %v\n", serviceName, err)
		return nil, err
	}
	if service == nil {
		fmt.Fprintf(opts.errorLog, "Can't find service: %s\n", serviceName)
		return nil, api.ErrNoService
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(opts.errorLog, "Failed to create pipe to dump service info for %s: %v\n", serviceName, err)
		return nil, err
	}

	session := &dumpSession{
		readFile: reader,
		done:     make(chan error, 1),
	}
	go func() {
		defer func() { _ = writer.Close() }()
		dumpService(ctx, serviceName, service, writer, opts.errorLog, opts.debugReader, types, args)
		session.done <- nil
		close(session.done)
	}()
	return session, nil
}

var errDumpTimeout = errors.New("dumpsys: dump timed out")

func writeDump(output io.Writer, errorLog io.Writer, reader *os.File, serviceName string, timeout time.Duration, asProto bool) (time.Duration, int64, error) {
	start := time.Now()
	done := make(chan struct{})
	type readChunk struct {
		data []byte
		err  error
	}
	chunks := make(chan readChunk, 1)
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				chunks <- readChunk{data: data}
			}
			if err != nil {
				chunks <- readChunk{err: err}
				return
			}
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var bytesWritten int64
	for {
		select {
		case <-timer.C:
			if !asProto {
				fmt.Fprintf(output, "\n*** SERVICE '%s' DUMP TIMEOUT (%dms) EXPIRED ***\n\n", serviceName, timeout.Milliseconds())
			}
			return time.Since(start), bytesWritten, errDumpTimeout
		case chunk := <-chunks:
			if chunk.data != nil {
				n, err := output.Write(chunk.data)
				bytesWritten += int64(n)
				if err != nil {
					fmt.Fprintf(errorLog, "Failed to write while dumping service %s: %v\n", serviceName, err)
					return time.Since(start), bytesWritten, err
				}
				continue
			}
			if chunk.err != nil {
				if errors.Is(chunk.err, io.EOF) {
					return time.Since(start), bytesWritten, nil
				}
				fmt.Fprintf(errorLog, "Failed to read while dumping service %s: %v\n", serviceName, chunk.err)
				return time.Since(start), bytesWritten, chunk.err
			}
		}
	}
}

func stopDumpSession(session *dumpSession, dumpComplete bool) {
	if session == nil {
		return
	}
	if session.readFile != nil {
		_ = session.readFile.Close()
	}
	if dumpComplete {
		for range session.done {
		}
	}
}

func writeDumpHeader(w io.Writer, serviceName string, priorityFlags api.DumpFlags) {
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	if priorityFlags == api.DumpPriorityAll || priorityFlags == api.DumpPriorityNormal || priorityFlags == api.DumpPriorityDefault {
		fmt.Fprintf(w, "DUMP OF SERVICE %s:\n", serviceName)
		return
	}
	fmt.Fprintf(w, "DUMP OF SERVICE %s %s:\n", priorityToString(priorityFlags), serviceName)
}

func writeDumpFooter(w io.Writer, serviceName string, elapsed time.Duration) {
	fmt.Fprintf(w, "--------- %.3fs was the duration of dumpsys %s, ending at: %s\n",
		elapsed.Seconds(), serviceName, time.Now().Format("2006-01-02 15:04:05"))
}

func dumpService(ctx context.Context, serviceName string, service api.Binder, out *os.File, errorLog io.Writer, debugReader binderDebugReader, types dumpType, args []string) {
	if types&typePID != 0 {
		if err := dumpPIDToWriter(ctx, service, out, types == typePID); err != nil {
			reportDumpError(errorLog, serviceName, "dumping PID", err)
		}
	}
	if types&typeStability != 0 {
		if _, err := fmt.Fprintf(out, "Stability: %s\n", api.BinderStability(service)); err != nil {
			reportDumpError(errorLog, serviceName, "dumping stability info", err)
		}
	}
	if types&typeThread != 0 {
		if err := dumpThreadsToWriter(ctx, service, out, debugReader); err != nil {
			reportDumpError(errorLog, serviceName, "dumping thread info", err)
		}
	}
	if types&typeClients != 0 {
		if err := dumpClientsToWriter(ctx, service, out, debugReader); err != nil {
			reportDumpError(errorLog, serviceName, "dumping clients info", err)
		}
	}
	if types&typeDump != 0 {
		if err := api.DumpBinder(ctx, service, api.NewFileDescriptor(int(out.Fd())), args); err != nil {
			reportDumpError(errorLog, serviceName, "dumping", err)
		}
	}
}

func reportDumpError(w io.Writer, serviceName string, action string, err error) {
	if err == nil || w == nil {
		return
	}
	fmt.Fprintf(w, "Error with service '%s' while %s: %s\n", serviceName, action, dumpErrorString(err))
}

func dumpErrorString(err error) string {
	if err == nil {
		return ""
	}

	var statusErr *api.StatusCodeError
	if errors.As(err, &statusErr) {
		if name := statusCodeName(statusErr.Code); name != "" {
			return name
		}
	}

	switch {
	case errors.Is(err, api.ErrNoService), errors.Is(err, os.ErrNotExist):
		return "NAME_NOT_FOUND"
	case errors.Is(err, api.ErrPermissionDenied):
		return "PERMISSION_DENIED"
	case errors.Is(err, api.ErrDeadObject):
		return "DEAD_OBJECT"
	case errors.Is(err, api.ErrFailedTxn):
		return "FAILED_TRANSACTION"
	case errors.Is(err, api.ErrBadType):
		return "BAD_TYPE"
	case errors.Is(err, api.ErrUnknownTransaction):
		return "UNKNOWN_TRANSACTION"
	default:
		return err.Error()
	}
}

func statusCodeName(code int32) string {
	switch code {
	case api.StatusUnknownError:
		return "UNKNOWN_ERROR"
	case api.StatusBadType:
		return "BAD_TYPE"
	case api.StatusFailedTransaction:
		return "FAILED_TRANSACTION"
	case api.StatusFdsNotAllowed:
		return "FDS_NOT_ALLOWED"
	case api.StatusUnexpectedNull:
		return "UNEXPECTED_NULL"
	case api.StatusPermissionDenied:
		return "PERMISSION_DENIED"
	case api.StatusNameNotFound:
		return "NAME_NOT_FOUND"
	case api.StatusNoMemory:
		return "NO_MEMORY"
	case api.StatusBadValue:
		return "BAD_VALUE"
	case api.StatusDeadObject:
		return "DEAD_OBJECT"
	case api.StatusInvalidOperation:
		return "INVALID_OPERATION"
	case api.StatusUnknownTransaction:
		return "UNKNOWN_TRANSACTION"
	default:
		return strconv.Itoa(int(code))
	}
}

func dumpPIDToWriter(ctx context.Context, service api.Binder, out *os.File, exclusive bool) error {
	pid, err := api.GetDebugPID(ctx, service)
	if err != nil {
		return err
	}
	if !exclusive {
		if _, err := io.WriteString(out, "Service host process PID: "); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, strconv.Itoa(int(pid))+"\n")
	return err
}

func dumpThreadsToWriter(ctx context.Context, service api.Binder, out *os.File, debugReader binderDebugReader) error {
	pid, err := api.GetDebugPID(ctx, service)
	if err != nil {
		return err
	}
	info, err := debugReader.GetPIDInfo(int(pid))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Threads in use: %d/%d\n", info.ThreadUsage, info.ThreadCount)
	return err
}

func dumpClientsToWriter(ctx context.Context, service api.Binder, out *os.File, debugReader binderDebugReader) error {
	provider, ok := service.(api.DebugHandleProvider)
	if !ok {
		_, err := io.WriteString(out, "Client PIDs are not available for local binders.\n")
		return err
	}
	handle, ok := provider.DebugHandle()
	if !ok {
		_, err := io.WriteString(out, "Client PIDs are not available for local binders.\n")
		return err
	}
	servicePID, err := api.GetDebugPID(ctx, service)
	if err != nil {
		return err
	}
	pids, err := debugReader.GetClientPIDs(os.Getpid(), int(servicePID), handle)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(pids))
	myPID := os.Getpid()
	for _, pid := range pids {
		if pid == myPID {
			continue
		}
		filtered = append(filtered, strconv.Itoa(pid))
	}
	_, err = fmt.Fprintf(out, "Client PIDs: %s\n", strings.Join(filtered, ", "))
	return err
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: dumpsys")
	fmt.Fprintln(w, "         To dump all services.")
	fmt.Fprintln(w, "or:")
	fmt.Fprintln(w, "       dumpsys [-t TIMEOUT] [--priority LEVEL] [--clients] [--dump] [--pid] [--thread] [--help | -l | --skip SERVICES | SERVICE [ARGS]]")
	fmt.Fprintln(w, "         --help: shows this help")
	fmt.Fprintln(w, "         -l: only list services, do not dump them")
	fmt.Fprintln(w, "         -t TIMEOUT_SEC: TIMEOUT to use in seconds instead of default 10 seconds")
	fmt.Fprintln(w, "         -T TIMEOUT_MS: TIMEOUT to use in milliseconds instead of default 10 seconds")
	fmt.Fprintln(w, "         --clients: dump client PIDs instead of usual dump")
	fmt.Fprintln(w, "         --dump: ask the service to dump itself (this is the default)")
	fmt.Fprintln(w, "         --pid: dump PID instead of usual dump")
	fmt.Fprintln(w, "         --proto: filter services that support dumping data in proto format. Dumps")
	fmt.Fprintln(w, "               will be in proto format.")
	fmt.Fprintln(w, "         --priority LEVEL: filter services based on specified priority")
	fmt.Fprintln(w, "               LEVEL must be one of CRITICAL | HIGH | NORMAL")
	fmt.Fprintln(w, "         --skip SERVICES: dumps all services but SERVICES")
	fmt.Fprintln(w, "         --stability: dump binder stability information instead of usual dump")
	fmt.Fprintln(w, "         --thread: dump thread usage instead of usual dump")
	fmt.Fprintln(w, "         SERVICE [ARGS]: dumps only service SERVICE, optionally passing ARGS to it")
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
