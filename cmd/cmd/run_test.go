package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRunNoServiceSpecified(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, Options{
		ServiceManager: fakeServiceManager{},
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 20 {
		t.Fatalf("Run code = %d, want 20", code)
	}
	if got := stderr.String(); !strings.Contains(got, "No service specified") {
		t.Fatalf("stderr = %q, want usage error", got)
	}
}

func TestRunListServices(t *testing.T) {
	registry := newFakeBinderRegistry()
	alpha := registry.register(api.StaticHandler{DescriptorName: "alpha", Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
		return api.NewParcel(), nil
	}})
	zeta := registry.register(api.StaticHandler{DescriptorName: "zeta", Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
		return api.NewParcel(), nil
	}})

	sm := fakeServiceManager{
		listServices: func(ctx context.Context, flags api.DumpFlags) ([]string, error) {
			if flags != api.DumpPriorityAll {
				t.Fatalf("ListServices flags = %#x, want %#x", flags, api.DumpPriorityAll)
			}
			return []string{"zeta", "missing", "alpha"}, nil
		},
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			switch name {
			case "alpha":
				return alpha, nil
			case "zeta":
				return zeta, nil
			default:
				return nil, nil
			}
		},
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"-l"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); got != "Currently running services:\n  alpha\n  zeta\n" {
		t.Fatalf("stdout = %q, want sorted service list", got)
	}
}

func TestRunMissingService(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return nil, nil
		},
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"missing.service"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 20 {
		t.Fatalf("Run code = %d, want 20", code)
	}
	if got := stderr.String(); !strings.Contains(got, "Can't find service: missing.service") {
		t.Fatalf("stderr = %q, want missing service error", got)
	}
}

func TestRunWaitServiceAndShellCommand(t *testing.T) {
	registry := newFakeBinderRegistry()
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
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if got, want := req.Args, []string{"arg1", "arg2"}; strings.Join(got, ",") != strings.Join(want, ",") {
				return fmt.Errorf("args = %#v, want %#v", got, want)
			}
			if _, err := syscall.Write(req.OutFD.FD(), []byte("hello stdout\n")); err != nil {
				return err
			}
			if _, err := syscall.Write(req.ErrFD.FD(), []byte("hello stderr\n")); err != nil {
				return err
			}
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 7)
		},
	}
	sm := fakeServiceManager{
		waitService: func(ctx context.Context, name string) (api.Binder, error) {
			if name != "activity" {
				t.Fatalf("WaitService name = %q, want activity", name)
			}
			return service, nil
		},
	}

	var outputLog, errorLog bytes.Buffer
	code := Run(context.Background(), []string{"-w", "activity", "arg1", "arg2"}, Options{
		ServiceManager: sm,
		Output:         &outputLog,
		Error:          &errorLog,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(stdoutW.Fd()),
		ErrFD:          int(stderrW.Fd()),
		RunMode:        RunModeLibrary,
	})
	if code != 7 {
		t.Fatalf("Run code = %d, want 7", code)
	}
	if got := readAllAndClose(t, stdoutR, stdoutW); got != "hello stdout\n" {
		t.Fatalf("stdout fd = %q, want hello stdout", got)
	}
	if got := readAllAndClose(t, stderrR, stderrW); got != "hello stderr\n" {
		t.Fatalf("stderr fd = %q, want hello stderr", got)
	}
	if outputLog.Len() != 0 || errorLog.Len() != 0 {
		t.Fatalf("logs = stdout:%q stderr:%q, want empty", outputLog.String(), errorLog.String())
	}
}

func TestRunShellCallbackOpenFile(t *testing.T) {
	registry := newFakeBinderRegistry()
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "out.txt")
	var checkerCalls int

	service := &fakeShellService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			cb := NewShellCallbackProxy(req.ShellCallback)
			pfd, err := cb.OpenFile(ctx, "out.txt", "u:r:system_server:s0", "w")
			if err != nil {
				return err
			}
			if pfd.FD() < 0 {
				return fmt.Errorf("OpenFile returned invalid fd %d", pfd.FD())
			}
			if _, err := syscall.Write(pfd.FD(), []byte("from shell callback")); err != nil {
				return err
			}
			_ = syscall.Close(pfd.FD())
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 0)
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}

	code := Run(context.Background(), []string{"service"}, Options{
		ServiceManager: sm,
		Output:         io.Discard,
		Error:          io.Discard,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
		WorkingDir:     dir,
		AccessChecker: fileAccessCheckerFunc(func(path string, seLinuxContext string, read bool, write bool) error {
			checkerCalls++
			if path != targetPath {
				return fmt.Errorf("path = %q, want %q", path, targetPath)
			}
			if seLinuxContext != "u:r:system_server:s0" || read || !write {
				return fmt.Errorf("unexpected access check args: ctx=%q read=%v write=%v", seLinuxContext, read, write)
			}
			return nil
		}),
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if checkerCalls != 1 {
		t.Fatalf("checkerCalls = %d, want 1", checkerCalls)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", targetPath, err)
	}
	if got := string(content); got != "from shell callback" {
		t.Fatalf("file content = %q, want shell callback output", got)
	}
}

func TestRunShellCallbackOpenFileAbsolutePath(t *testing.T) {
	registry := newFakeBinderRegistry()
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "abs-out.txt")

	service := &fakeShellService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			cb := NewShellCallbackProxy(req.ShellCallback)
			pfd, err := cb.OpenFile(ctx, targetPath, "u:r:system_server:s0", "w")
			if err != nil {
				return err
			}
			if pfd.FD() < 0 {
				return fmt.Errorf("OpenFile returned invalid fd %d", pfd.FD())
			}
			if _, err := syscall.Write(pfd.FD(), []byte("absolute path")); err != nil {
				return err
			}
			_ = syscall.Close(pfd.FD())
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 0)
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}

	code := Run(context.Background(), []string{"service"}, Options{
		ServiceManager: sm,
		Output:         io.Discard,
		Error:          io.Discard,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
		WorkingDir:     filepath.Join(dir, "nested"),
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", targetPath, err)
	}
	if got := string(content); got != "absolute path" {
		t.Fatalf("file content = %q, want absolute path", got)
	}
}

func TestRunDeactivatesShellCallbackAfterTransact(t *testing.T) {
	registry := newFakeBinderRegistry()
	dir := t.TempDir()
	afterReturn := make(chan int, 1)

	service := &fakeShellService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			go func() {
				time.Sleep(20 * time.Millisecond)
				cb := NewShellCallbackProxy(req.ShellCallback)
				pfd, err := cb.OpenFile(context.Background(), "late.txt", "", "w")
				if err != nil {
					afterReturn <- -999
					return
				}
				afterReturn <- pfd.FD()
				_ = NewResultReceiverProxy(req.ResultReceiver).Send(context.Background(), 0)
			}()
			return nil
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}

	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"service"}, Options{
		ServiceManager: sm,
		Output:         io.Discard,
		Error:          &stderr,
		InFD:           int(os.Stdin.Fd()),
		OutFD:          int(os.Stdout.Fd()),
		ErrFD:          int(os.Stderr.Fd()),
		WorkingDir:     dir,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if got := <-afterReturn; got >= 0 {
		t.Fatalf("post-transact callback fd = %d, want negative EPERM path", got)
	}
	if got := stderr.String(); !strings.Contains(got, "Open attempt after active for") {
		t.Fatalf("stderr = %q, want inactive callback log", got)
	}
}

func TestShellCallbackHandlerDirectErrors(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	handler := NewShellCallbackHandler(&stderr, ShellCallbackOptions{WorkingDir: dir})

	if got := handler.OpenFile("bad.txt", "", "bad-mode"); got != -int(syscall.EINVAL) {
		t.Fatalf("OpenFile(invalid mode) = %d, want %d", got, -int(syscall.EINVAL))
	}
	handler.Deactivate()
	if got := handler.OpenFile("bad.txt", "", "r"); got != -int(syscall.EPERM) {
		t.Fatalf("OpenFile(inactive) = %d, want %d", got, -int(syscall.EPERM))
	}
	if log := stderr.String(); !strings.Contains(log, "Invalid mode requested: bad-mode") || !strings.Contains(log, "Open attempt after active for") {
		t.Fatalf("stderr = %q, want invalid mode and inactive messages", log)
	}
}

func TestShellCallbackHandlerModeMatrix(t *testing.T) {
	tests := []struct {
		mode      string
		wantFlags int
		wantRead  bool
		wantWrite bool
	}{
		{mode: "w", wantFlags: syscall.O_WRONLY | syscall.O_CREAT | syscall.O_TRUNC, wantWrite: true},
		{mode: "w+", wantFlags: syscall.O_RDWR | syscall.O_CREAT | syscall.O_TRUNC, wantRead: true, wantWrite: true},
		{mode: "r", wantFlags: syscall.O_RDONLY, wantRead: true},
		{mode: "r+", wantFlags: syscall.O_RDWR, wantRead: true, wantWrite: true},
	}
	for _, tt := range tests {
		flags, checkRead, checkWrite, ok := openFlagsForMode(tt.mode)
		if !ok {
			t.Fatalf("openFlagsForMode(%q) = !ok, want ok", tt.mode)
		}
		if flags != tt.wantFlags || checkRead != tt.wantRead || checkWrite != tt.wantWrite {
			t.Fatalf("openFlagsForMode(%q) = flags=%#x read=%v write=%v, want flags=%#x read=%v write=%v", tt.mode, flags, checkRead, checkWrite, tt.wantFlags, tt.wantRead, tt.wantWrite)
		}
	}
}

func TestShellCallbackHandlerAccessDenied(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "blocked.txt")
	if err := os.WriteFile(target, []byte("seed"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", target, err)
	}

	var stderr bytes.Buffer
	handler := NewShellCallbackHandler(&stderr, ShellCallbackOptions{
		WorkingDir: dir,
		AccessChecker: fileAccessCheckerFunc(func(path string, seLinuxContext string, read bool, write bool) error {
			return errors.New("denied")
		}),
	})
	if got := handler.OpenFile("blocked.txt", "u:r:system_server:s0", "r"); got != -int(syscall.EPERM) {
		t.Fatalf("OpenFile(access denied) = %d, want %d", got, -int(syscall.EPERM))
	}
	if log := stderr.String(); !strings.Contains(log, "System server has no access to file") {
		t.Fatalf("stderr = %q, want access denied message", log)
	}
}

func TestResultReceiverProxyRoundTrip(t *testing.T) {
	registry := newFakeBinderRegistry()
	handler := NewResultReceiverHandler()
	b := registry.register(handler)

	if err := NewResultReceiverProxy(b).Send(context.Background(), 42); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, err := handler.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got != 42 {
		t.Fatalf("Wait result = %d, want 42", got)
	}
}

func TestTransactErrorTextMappings(t *testing.T) {
	tests := []struct {
		err  error
		want string
		code int
	}{
		{err: &api.StatusCodeError{Code: api.StatusBadType}, want: "Bad type", code: int(api.StatusBadType)},
		{err: &api.StatusCodeError{Code: api.StatusFailedTransaction}, want: "Failed transaction", code: int(api.StatusFailedTransaction)},
		{err: &api.StatusCodeError{Code: api.StatusFdsNotAllowed}, want: "File descriptors not allowed", code: int(api.StatusFdsNotAllowed)},
		{err: &api.StatusCodeError{Code: api.StatusUnexpectedNull}, want: "Unexpected null", code: int(api.StatusUnexpectedNull)},
		{err: &api.StatusCodeError{Code: api.StatusPermissionDenied}, want: syscall.EPERM.Error(), code: int(api.StatusPermissionDenied)},
	}
	for _, tt := range tests {
		if got := transactErrorText(tt.err); got != tt.want {
			t.Fatalf("transactErrorText(%v) = %q, want %q", tt.err, got, tt.want)
		}
		if got := transactExitCode(tt.err); got != tt.code {
			t.Fatalf("transactExitCode(%v) = %d, want %d", tt.err, got, tt.code)
		}
	}
}

type fakeServiceManager struct {
	checkService func(context.Context, string) (api.Binder, error)
	waitService  func(context.Context, string) (api.Binder, error)
	listServices func(context.Context, api.DumpFlags) ([]string, error)
}

func (f fakeServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if f.checkService == nil {
		return nil, nil
	}
	return f.checkService(ctx, name)
}

func (f fakeServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	if f.waitService == nil {
		return nil, nil
	}
	return f.waitService(ctx, name)
}

func (f fakeServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return api.ErrUnsupported
}

func (f fakeServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	if f.listServices == nil {
		return nil, nil
	}
	return f.listServices(ctx, dumpFlags)
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
	registry       *fakeBinderRegistry
	onShellCommand func(context.Context, shellCommandRequest) error
}

func (f *fakeShellService) Descriptor(ctx context.Context) (string, error) {
	return "fake.shell", nil
}

func (f *fakeShellService) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if code == api.InterfaceTransaction {
		reply := api.NewParcel()
		if err := reply.WriteString("fake.shell"); err != nil {
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
	data.SetBinderResolvers(f.registry.resolve, nil)
	data.SetBinderObjectResolvers(func(obj api.ParcelObject) api.Binder {
		return f.registry.resolve(obj.Handle)
	}, nil)
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

func (f *fakeShellService) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	return f.registry.register(handler), nil
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

type fakeBinderRegistry struct {
	mu     sync.Mutex
	next   uint32
	locals map[uint32]*fakeLocalBinder
}

func newFakeBinderRegistry() *fakeBinderRegistry {
	return &fakeBinderRegistry{
		next:   1,
		locals: make(map[uint32]*fakeLocalBinder),
	}
}

func (r *fakeBinderRegistry) register(handler api.Handler) *fakeLocalBinder {
	r.mu.Lock()
	defer r.mu.Unlock()
	handle := r.next
	r.next++
	b := &fakeLocalBinder{
		handle:  handle,
		handler: handler,
	}
	r.locals[handle] = b
	return b
}

func (r *fakeBinderRegistry) resolve(handle uint32) api.Binder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locals[handle]
}

type fakeLocalBinder struct {
	handle  uint32
	handler api.Handler
}

func (b *fakeLocalBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b *fakeLocalBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if data == nil {
		data = api.NewParcel()
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	return api.DispatchLocalHandler(ctx, b.handler, nil, code, data, flags, api.TransactionContext{
		Code:  code,
		Flags: flags,
		Local: true,
	})
}

func (b *fakeLocalBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *fakeLocalBinder) Close() error {
	return nil
}

func (b *fakeLocalBinder) WriteBinderToParcel(p *api.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type fileAccessCheckerFunc func(path string, seLinuxContext string, read bool, write bool) error

func (f fileAccessCheckerFunc) CheckFileAccess(path string, seLinuxContext string, read bool, write bool) error {
	return f(path, seLinuxContext, read, write)
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
