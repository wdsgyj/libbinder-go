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

func TestRunNilContextUsesBackground(t *testing.T) {
	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if ctx == nil {
				t.Fatal("ctx = nil")
			}
			return nil
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if ctx == nil {
				t.Fatal("CheckService ctx = nil")
			}
			return service, nil
		},
	}

	code := Run(nil, []string{"tap", "1", "2"}, Options{
		ServiceManager: sm,
		Output:         io.Discard,
		Error:          io.Discard,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
}

func TestRunNilServiceManager(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"tap", "1", "2"}, Options{
		Error: &stderr,
	})
	if code != unavailableExitCode {
		t.Fatalf("Run code = %d, want %d", code, unavailableExitCode)
	}
	if got := stderr.String(); got != "input: Unable to get default service manager!\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunCheckServiceError(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return nil, errors.New("lookup failed")
		},
	}
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"tap"}, Options{
		ServiceManager: sm,
		Error:          &stderr,
	})
	if code != unavailableExitCode {
		t.Fatalf("Run code = %d, want %d", code, unavailableExitCode)
	}
	if got := stderr.String(); !strings.Contains(got, "Failure finding service input: lookup failed") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunBuildShellCommandRequestFailure(t *testing.T) {
	service := &fakeShellService{}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"tap"}, Options{
		ServiceManager: sm,
		Error:          &stderr,
		InFD:           -1,
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
	})
	if code != 1 {
		t.Fatalf("Run code = %d, want 1", code)
	}
	if got := stderr.String(); !strings.Contains(got, "Failed to build shell command request") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestProcessExitCode(t *testing.T) {
	if got := ProcessExitCode(0x1234); got != 0x34 {
		t.Fatalf("ProcessExitCode = %#x, want %#x", got, 0x34)
	}
}

func TestWriterOrDiscardNil(t *testing.T) {
	if got := writerOrDiscard(nil); got != io.Discard {
		t.Fatalf("writerOrDiscard(nil) = %v, want io.Discard", got)
	}
}

func TestWriteShellCommandRequestNilParcel(t *testing.T) {
	err := writeShellCommandRequest(nil, 0, 1, 2, []string{"tap"})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("writeShellCommandRequest err = %v, want ErrBadParcelable", err)
	}
}

func TestWriteShellCommandRequestInjectedErrors(t *testing.T) {
	oldWriteInt32 := parcelWriteInt32
	oldWriteString := parcelWriteString
	oldWriteBinder := parcelWriteNullStrongBinder
	t.Cleanup(func() {
		parcelWriteInt32 = oldWriteInt32
		parcelWriteString = oldWriteString
		parcelWriteNullStrongBinder = oldWriteBinder
	})

	parcelWriteInt32 = func(p *api.Parcel, v int32) error {
		return errors.New("int32 fail")
	}
	if err := writeShellCommandRequest(api.NewParcel(), 0, 1, 2, nil); err == nil || !strings.Contains(err.Error(), "int32 fail") {
		t.Fatalf("writeShellCommandRequest int32 err = %v, want int32 fail", err)
	}

	parcelWriteInt32 = oldWriteInt32
	parcelWriteString = func(p *api.Parcel, v string) error {
		return errors.New("string fail")
	}
	if err := writeShellCommandRequest(api.NewParcel(), 0, 1, 2, []string{"tap"}); err == nil || !strings.Contains(err.Error(), "string fail") {
		t.Fatalf("writeShellCommandRequest string err = %v, want string fail", err)
	}

	parcelWriteString = oldWriteString
	parcelWriteNullStrongBinder = func(p *api.Parcel) error {
		return errors.New("binder fail")
	}
	if err := writeShellCommandRequest(api.NewParcel(), 0, 1, 2, nil); err == nil || !strings.Contains(err.Error(), "binder fail") {
		t.Fatalf("writeShellCommandRequest binder err = %v, want binder fail", err)
	}
}

func TestTransactErrorText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "status bad type", err: &api.StatusCodeError{Code: api.StatusBadType}, want: "Bad type"},
		{name: "status failed txn", err: &api.StatusCodeError{Code: api.StatusFailedTransaction}, want: "Failed transaction"},
		{name: "status fds not allowed", err: &api.StatusCodeError{Code: api.StatusFdsNotAllowed}, want: "File descriptors not allowed"},
		{name: "status unexpected null", err: &api.StatusCodeError{Code: api.StatusUnexpectedNull}, want: "Unexpected null"},
		{name: "status errno", err: &api.StatusCodeError{Code: api.StatusPermissionDenied}, want: "operation not permitted"},
		{name: "status generic", err: &api.StatusCodeError{Code: -5000}, want: "binder: transport status -5000"},
		{name: "sentinel bad type", err: api.ErrBadType, want: "Bad type"},
		{name: "sentinel failed txn", err: api.ErrFailedTxn, want: "Failed transaction"},
		{name: "fallback", err: errors.New("boom"), want: "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transactErrorText(tt.err); got != tt.want {
				t.Fatalf("transactErrorText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransactExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "status explicit", err: &api.StatusCodeError{Code: api.StatusPermissionDenied}, want: int(api.StatusPermissionDenied)},
		{name: "sentinel bad type", err: api.ErrBadType, want: int(api.StatusBadType)},
		{name: "sentinel failed txn", err: api.ErrFailedTxn, want: int(api.StatusFailedTransaction)},
		{name: "sentinel permission denied", err: api.ErrPermissionDenied, want: int(api.StatusPermissionDenied)},
		{name: "sentinel unknown txn", err: api.ErrUnknownTransaction, want: int(api.StatusUnknownTransaction)},
		{name: "fallback", err: errors.New("boom"), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transactExitCode(tt.err); got != tt.want {
				t.Fatalf("transactExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPrintableStatusMagnitude(t *testing.T) {
	if got := printableStatusMagnitude(-7); got != 7 {
		t.Fatalf("printableStatusMagnitude(-7) = %d, want 7", got)
	}
	if got := printableStatusMagnitude(9); got != 9 {
		t.Fatalf("printableStatusMagnitude(9) = %d, want 9", got)
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
