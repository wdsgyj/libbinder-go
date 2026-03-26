package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRunForwardsShellCommandArgs(t *testing.T) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout): %v", err)
	}
	defer func() { _ = stdoutR.Close(); _ = stdoutW.Close() }()
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr): %v", err)
	}
	defer func() { _ = stderrR.Close(); _ = stderrW.Close() }()

	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if got, want := strings.Join(req.Args, ","), "tap,100,200"; got != want {
				return errors.New("unexpected args: " + got)
			}
			if req.ShellCallback != nil {
				return errors.New("expected nil shell callback")
			}
			if req.ResultReceiver != nil {
				return errors.New("expected nil result receiver")
			}
			if _, err := syscall.Write(req.OutFD.FD(), []byte("ok\n")); err != nil {
				return err
			}
			if _, err := syscall.Write(req.ErrFD.FD(), []byte("warn\n")); err != nil {
				return err
			}
			return nil
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if name != inputServiceName {
				t.Fatalf("CheckService name = %q, want %q", name, inputServiceName)
			}
			return service, nil
		},
	}

	var outputLog, errorLog bytes.Buffer
	code := Run(context.Background(), []string{"tap", "100", "200"}, Options{
		ServiceManager: sm,
		Output:         &outputLog,
		Error:          &errorLog,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(stdoutW.Fd()),
		ErrFD:          int(stderrW.Fd()),
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if got := readAllAndClose(t, stdoutR, stdoutW); got != "ok\n" {
		t.Fatalf("stdout fd = %q, want ok", got)
	}
	if got := readAllAndClose(t, stderrR, stderrW); got != "warn\n" {
		t.Fatalf("stderr fd = %q, want warn", got)
	}
	if outputLog.Len() != 0 || errorLog.Len() != 0 {
		t.Fatalf("logs = stdout:%q stderr:%q, want empty", outputLog.String(), errorLog.String())
	}
}

func TestRunNoArgsStillInvokesShellCommand(t *testing.T) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout): %v", err)
	}
	defer func() { _ = stdoutR.Close(); _ = stdoutW.Close() }()

	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if len(req.Args) != 0 {
				return errors.New("expected no args")
			}
			if req.ShellCallback != nil {
				return errors.New("expected nil shell callback")
			}
			if req.ResultReceiver != nil {
				return errors.New("expected nil result receiver")
			}
			if _, err := syscall.Write(req.OutFD.FD(), []byte("Usage: input ...\n")); err != nil {
				return err
			}
			return nil
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}

	code := Run(context.Background(), nil, Options{
		ServiceManager: sm,
		Output:         io.Discard,
		Error:          io.Discard,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(stdoutW.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if got := readAllAndClose(t, stdoutR, stdoutW); got != "Usage: input ...\n" {
		t.Fatalf("stdout fd = %q", got)
	}
}

func TestRunMissingService(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"tap", "1", "2"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != unavailableExitCode {
		t.Fatalf("Run code = %d, want %d", code, unavailableExitCode)
	}
	if got := stderr.String(); !strings.Contains(got, "Can't find service: input") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunTransactFailureUsesStatusExitCode(t *testing.T) {
	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			return &api.StatusCodeError{Code: api.StatusPermissionDenied}
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"tap", "1", "2"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
	})
	if code != int(api.StatusPermissionDenied) {
		t.Fatalf("Run code = %d, want %d", code, api.StatusPermissionDenied)
	}
	if got := stdout.String(); !strings.Contains(got, "Failure calling service input") || !strings.Contains(got, "operation not permitted") {
		t.Fatalf("stdout = %q", got)
	}
}

type fakeServiceManager struct {
	checkService func(context.Context, string) (api.Binder, error)
}

func (f fakeServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if f.checkService == nil {
		return nil, nil
	}
	return f.checkService(ctx, name)
}

func (f fakeServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return api.ErrUnsupported
}

func (f fakeServiceManager) ListServices(ctx context.Context, flags api.DumpFlags) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, api.ErrUnsupported
}

func (f fakeServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	return api.ErrUnsupported
}

func (f fakeServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return nil, api.ErrUnsupported
}

type fakeShellService struct {
	onShellCommand func(context.Context, shellCommandRequest) error
}

func (f *fakeShellService) Descriptor(ctx context.Context) (string, error) {
	return "fake.input", nil
}

func (f *fakeShellService) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if code == api.InterfaceTransaction {
		reply := api.NewParcel()
		if err := reply.WriteString("fake.input"); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if code != api.ShellCommandTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}

	req, err := parseShellCommandRequest(data)
	if err != nil {
		return nil, err
	}
	if f.onShellCommand != nil {
		if err := f.onShellCommand(ctx, req); err != nil {
			return nil, err
		}
	}
	return api.NewParcel(), nil
}

func (f *fakeShellService) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f *fakeShellService) Close() error {
	return nil
}

type shellCommandRequest struct {
	InFD           api.FileDescriptor
	OutFD          api.FileDescriptor
	ErrFD          api.FileDescriptor
	Args           []string
	ShellCallback  api.Binder
	ResultReceiver api.Binder
}

func parseShellCommandRequest(p *api.Parcel) (shellCommandRequest, error) {
	inFD, err := p.ReadFileDescriptor()
	if err != nil {
		return shellCommandRequest{}, err
	}
	outFD, err := p.ReadFileDescriptor()
	if err != nil {
		return shellCommandRequest{}, err
	}
	errFD, err := p.ReadFileDescriptor()
	if err != nil {
		return shellCommandRequest{}, err
	}
	argc, err := p.ReadInt32()
	if err != nil {
		return shellCommandRequest{}, err
	}
	args := make([]string, 0, argc)
	for i := int32(0); i < argc; i++ {
		arg, err := p.ReadString()
		if err != nil {
			return shellCommandRequest{}, err
		}
		args = append(args, arg)
	}
	callback, err := p.ReadStrongBinder()
	if err != nil {
		return shellCommandRequest{}, err
	}
	result, err := p.ReadStrongBinder()
	if err != nil {
		return shellCommandRequest{}, err
	}
	return shellCommandRequest{
		InFD:           inFD,
		OutFD:          outFD,
		ErrFD:          errFD,
		Args:           args,
		ShellCallback:  callback,
		ResultReceiver: result,
	}, nil
}

func readAllAndClose(t *testing.T, r *os.File, w *os.File) string {
	t.Helper()
	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(data)
}
