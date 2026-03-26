package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
	binderservice "github.com/wdsgyj/libbinder-go/service"
)

type openedConn interface {
	ServiceManager() api.ServiceManager
	Close() error
}

var openConn = func(cfg libbinder.Config) (openedConn, error) {
	return libbinder.Open(cfg)
}

const (
	unavailableExitCode = 20
)

type Options struct {
	ServiceManager api.ServiceManager
	Output         io.Writer
	Error          io.Writer
	InFD           int
	OutFD          int
	ErrFD          int
}

type resolvedOptions struct {
	serviceManager api.ServiceManager
	outputLog      io.Writer
	errorLog       io.Writer
	inFD           int
	outFD          int
	errFD          int
}

func Main(ctx context.Context, argv []string, stdout io.Writer, stderr io.Writer) int {
	conn, err := openConn(libbinder.Config{})
	if err != nil {
		fmt.Fprintln(stderr, "input: Unable to get default service manager!")
		return unavailableExitCode
	}
	defer func() { _ = conn.Close() }()

	return Run(ctx, argv, Options{
		ServiceManager: conn.ServiceManager(),
		Output:         stdout,
		Error:          stderr,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
	})
}

func ProcessExitCode(code int) int {
	return code & 0xff
}

func Run(ctx context.Context, argv []string, opts Options) int {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved := resolveOptions(opts)
	if resolved.serviceManager == nil {
		fmt.Fprintln(resolved.errorLog, "input: Unable to get default service manager!")
		return unavailableExitCode
	}

	service, err := binderservice.LookupInputManagerService(ctx, resolved.serviceManager)
	if err != nil {
		if errors.Is(err, api.ErrNoService) {
			fmt.Fprintf(resolved.errorLog, "input: Can't find service: %s\n", binderservice.InputServiceName)
			return unavailableExitCode
		}
		fmt.Fprintf(resolved.errorLog, "input: Failure finding service %s: %v\n", binderservice.InputServiceName, err)
		return unavailableExitCode
	}
	err = service.WithShellIO(binderservice.InputShellIO{
		InFD:  api.NewFileDescriptor(resolved.inFD),
		OutFD: api.NewFileDescriptor(resolved.outFD),
		ErrFD: api.NewFileDescriptor(resolved.errFD),
	}).ExecuteCommand(ctx, argv)
	if err != nil {
		var buildErr *binderservice.InputShellCommandBuildError
		if errors.As(err, &buildErr) {
			fmt.Fprintf(resolved.errorLog, "input: Failed to build shell command request: %v\n", err)
			return 1
		}
		if errors.Is(err, api.ErrBadParcelable) {
			fmt.Fprintf(resolved.errorLog, "input: %v\n", err)
			return 1
		}
		code := transactExitCode(err)
		fmt.Fprintf(resolved.outputLog, "input: Failure calling service %s: %s (%d)\n", binderservice.InputServiceName, transactErrorText(err), printableStatusMagnitude(code))
		return code
	}
	return 0
}

func resolveOptions(opts Options) resolvedOptions {
	return resolvedOptions{
		serviceManager: opts.ServiceManager,
		outputLog:      writerOrDiscard(opts.Output),
		errorLog:       writerOrDiscard(opts.Error),
		inFD:           opts.InFD,
		outFD:          opts.OutFD,
		errFD:          opts.ErrFD,
	}
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
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
