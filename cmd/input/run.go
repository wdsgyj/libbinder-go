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
)

type openedConn interface {
	ServiceManager() api.ServiceManager
	Close() error
}

var openConn = func(cfg libbinder.Config) (openedConn, error) {
	return libbinder.Open(cfg)
}

var (
	parcelWriteInt32 = func(p *api.Parcel, v int32) error {
		return p.WriteInt32(v)
	}
	parcelWriteString = func(p *api.Parcel, v string) error {
		return p.WriteString(v)
	}
	parcelWriteNullStrongBinder = func(p *api.Parcel) error {
		return p.WriteStrongBinder(nil)
	}
)

const (
	inputServiceName    = "input"
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

	service, err := resolved.serviceManager.CheckService(ctx, inputServiceName)
	if err != nil {
		fmt.Fprintf(resolved.errorLog, "input: Failure finding service %s: %v\n", inputServiceName, err)
		return unavailableExitCode
	}
	if service == nil {
		fmt.Fprintf(resolved.errorLog, "input: Can't find service: %s\n", inputServiceName)
		return unavailableExitCode
	}

	data := api.NewParcel()
	if err := writeShellCommandRequest(data, resolved.inFD, resolved.outFD, resolved.errFD, argv); err != nil {
		fmt.Fprintf(resolved.errorLog, "input: Failed to build shell command request: %v\n", err)
		return 1
	}
	if _, err := service.Transact(ctx, api.ShellCommandTransaction, data, api.FlagNone); err != nil {
		code := transactExitCode(err)
		fmt.Fprintf(resolved.outputLog, "input: Failure calling service %s: %s (%d)\n", inputServiceName, transactErrorText(err), printableStatusMagnitude(code))
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

func writeShellCommandRequest(p *api.Parcel, inFD int, outFD int, errFD int, args []string) error {
	err := p.WriteFileDescriptor(api.NewFileDescriptor(inFD))
	if err == nil {
		err = p.WriteFileDescriptor(api.NewFileDescriptor(outFD))
	}
	if err == nil {
		err = p.WriteFileDescriptor(api.NewFileDescriptor(errFD))
	}
	if err != nil {
		return err
	}

	err = parcelWriteInt32(p, int32(len(args)))
	for _, arg := range args {
		if err == nil {
			err = parcelWriteString(p, arg)
		}
	}
	if err != nil {
		return err
	}

	// `input` shell commands are synchronous in practice; sending nil here keeps
	// the client aligned with device behavior without depending on callback IPC.
	err = parcelWriteNullStrongBinder(p)
	if err == nil {
		err = parcelWriteNullStrongBinder(p)
	}
	if err != nil {
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
