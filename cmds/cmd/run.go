package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"syscall"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

type RunMode uint8

const (
	RunModeStandalone RunMode = iota
	RunModeLibrary
)

type FileAccessChecker interface {
	CheckFileAccess(path string, seLinuxContext string, read bool, write bool) error
}

type Options struct {
	ServiceManager api.ServiceManager
	Output         io.Writer
	Error          io.Writer
	InFD           int
	OutFD          int
	ErrFD          int
	RunMode        RunMode
	WorkingDir     string
	AccessChecker  FileAccessChecker
}

type localHandlerRegistrar interface {
	RegisterLocalHandler(handler api.Handler) (api.Binder, error)
}

func Main(ctx context.Context, argv []string, stdout io.Writer, stderr io.Writer) int {
	conn, err := libbinder.Open(libbinder.Config{})
	if err != nil {
		fmt.Fprintln(stderr, "cmd: Unable to get default service manager!")
		return 20
	}
	defer func() { _ = conn.Close() }()

	return Run(ctx, argv, Options{
		ServiceManager: conn.ServiceManager(),
		Output:         stdout,
		Error:          stderr,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
		RunMode:        RunModeStandalone,
	})
}

func ProcessExitCode(code int) int {
	return code & 0xff
}

func Run(ctx context.Context, argv []string, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved, err := resolveOptions(opts)
	if err != nil {
		fmt.Fprintln(writerOrDiscard(opts.Error), "cmd:", err)
		return 20
	}
	if resolved.serviceManager == nil {
		fmt.Fprintln(resolved.errorLog, "cmd: Unable to get default service manager!")
		return 20
	}

	argc := len(argv)
	if argc == 0 {
		fmt.Fprintln(resolved.errorLog, "cmd: No service specified; use -l to list all running services. Use -w to start and wait for a service.")
		return 20
	}

	if argc == 1 && argv[0] == "-l" {
		return listServices(ctx, resolved)
	}

	waitForService := argc > 1 && argv[0] == "-w"
	serviceIdx := 0
	if waitForService {
		serviceIdx = 1
	}
	name := argv[serviceIdx]
	service, err := lookupService(ctx, resolved.serviceManager, name, waitForService)
	if err != nil {
		if errors.Is(err, api.ErrNoService) {
			service = nil
		} else {
			fmt.Fprintf(resolved.errorLog, "cmd: Failure finding service %s: %v\n", name, err)
			return 20
		}
	}
	if service == nil {
		fmt.Fprintf(resolved.errorLog, "cmd: Can't find service: %s\n", name)
		return 20
	}

	registrar, ok := service.(localHandlerRegistrar)
	if !ok {
		code := int(api.StatusInvalidOperation)
		fmt.Fprintf(resolved.outputLog, "cmd: Failure calling service %s: %s (%d)\n", name, syscall.Errno(-code).Error(), -code)
		return code
	}

	callback := NewShellCallbackHandler(resolved.errorLog, ShellCallbackOptions{
		WorkingDir:    resolved.workingDir,
		AccessChecker: resolved.accessChecker,
	})
	result := NewResultReceiverHandler()

	callbackBinder, err := registrar.RegisterLocalHandler(callback)
	if err != nil {
		fmt.Fprintf(resolved.errorLog, "cmd: Failed to register shell callback: %v\n", err)
		return 1
	}
	defer func() { _ = callbackBinder.Close() }()

	resultBinder, err := registrar.RegisterLocalHandler(result)
	if err != nil {
		fmt.Fprintf(resolved.errorLog, "cmd: Failed to register result receiver: %v\n", err)
		return 1
	}
	defer func() { _ = resultBinder.Close() }()

	data := api.NewParcel()
	if err := writeShellCommandRequest(data, resolved.inFD, resolved.outFD, resolved.errFD, argv[serviceIdx+1:], callbackBinder, resultBinder); err != nil {
		fmt.Fprintf(resolved.errorLog, "cmd: Failed to build shell command request: %v\n", err)
		return 1
	}
	if _, err := service.Transact(ctx, api.ShellCommandTransaction, data, api.FlagNone); err != nil {
		callback.Deactivate()
		_ = callback.Close()
		code := transactExitCode(err)
		fmt.Fprintf(resolved.outputLog, "cmd: Failure calling service %s: %s (%d)\n", name, transactErrorText(err), printableStatusMagnitude(code))
		return code
	}

	callback.Deactivate()
	resultCode, err := result.Wait(ctx)
	_ = callback.Close()
	if err != nil {
		fmt.Fprintf(resolved.errorLog, "cmd: Failure waiting for command result: %v\n", err)
		return 1
	}
	return int(resultCode)
}

type resolvedOptions struct {
	serviceManager api.ServiceManager
	outputLog      io.Writer
	errorLog       io.Writer
	inFD           int
	outFD          int
	errFD          int
	workingDir     string
	accessChecker  FileAccessChecker
}

func resolveOptions(opts Options) (resolvedOptions, error) {
	cwd := opts.WorkingDir
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return resolvedOptions{}, err
		}
	}
	return resolvedOptions{
		serviceManager: opts.ServiceManager,
		outputLog:      writerOrDiscard(opts.Output),
		errorLog:       writerOrDiscard(opts.Error),
		inFD:           opts.InFD,
		outFD:          opts.OutFD,
		errFD:          opts.ErrFD,
		workingDir:     cwd,
		accessChecker:  opts.AccessChecker,
	}, nil
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func listServices(ctx context.Context, opts resolvedOptions) int {
	services, err := opts.serviceManager.ListServices(ctx, api.DumpPriorityAll)
	if err != nil {
		fmt.Fprintf(opts.errorLog, "cmd: Failed to list services: %v\n", err)
		return 20
	}
	sort.Strings(services)

	fmt.Fprintln(opts.outputLog, "Currently running services:")
	for _, name := range services {
		service, err := opts.serviceManager.CheckService(ctx, name)
		if err != nil {
			continue
		}
		if service != nil {
			fmt.Fprintf(opts.outputLog, "  %s\n", name)
		}
	}
	return 0
}

func lookupService(ctx context.Context, sm api.ServiceManager, name string, wait bool) (api.Binder, error) {
	if wait {
		return sm.WaitService(ctx, name)
	}
	return sm.CheckService(ctx, name)
}

func writeShellCommandRequest(p *api.Parcel, inFD int, outFD int, errFD int, args []string, callback api.Binder, result api.Binder) error {
	if err := p.WriteFileDescriptor(api.NewFileDescriptor(inFD)); err != nil {
		return err
	}
	if err := p.WriteFileDescriptor(api.NewFileDescriptor(outFD)); err != nil {
		return err
	}
	if err := p.WriteFileDescriptor(api.NewFileDescriptor(errFD)); err != nil {
		return err
	}
	if err := p.WriteInt32(int32(len(args))); err != nil {
		return err
	}
	for _, arg := range args {
		if err := p.WriteString(arg); err != nil {
			return err
		}
	}
	if err := p.WriteStrongBinder(callback); err != nil {
		return err
	}
	if err := p.WriteStrongBinder(result); err != nil {
		return err
	}
	return p.SetPosition(0)
}

func transactErrorText(err error) string {
	var statusErr *api.StatusCodeError
	if errors.As(err, &statusErr) {
		switch statusErr.Code {
		case api.StatusBadType:
			return "Bad type"
		case api.StatusFailedTransaction:
			return "Failed transaction"
		case api.StatusFdsNotAllowed:
			return "File descriptors not allowed"
		case api.StatusUnexpectedNull:
			return "Unexpected null"
		}
		if statusErr.Code < 0 && statusErr.Code >= -4095 {
			return syscall.Errno(-statusErr.Code).Error()
		}
		return statusErr.Error()
	}

	switch {
	case errors.Is(err, api.ErrBadType):
		return "Bad type"
	case errors.Is(err, api.ErrFailedTxn):
		return "Failed transaction"
	default:
		return err.Error()
	}
}

func transactExitCode(err error) int {
	var statusErr *api.StatusCodeError
	if errors.As(err, &statusErr) {
		return int(statusErr.Code)
	}
	switch {
	case errors.Is(err, api.ErrBadType):
		return int(api.StatusBadType)
	case errors.Is(err, api.ErrFailedTxn):
		return int(api.StatusFailedTransaction)
	case errors.Is(err, api.ErrPermissionDenied):
		return int(api.StatusPermissionDenied)
	case errors.Is(err, api.ErrUnknownTransaction):
		return int(api.StatusUnknownTransaction)
	default:
		return 1
	}
}

func printableStatusMagnitude(code int) int {
	if code < 0 {
		return -code
	}
	return code
}
