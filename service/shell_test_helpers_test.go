package service

import (
	"context"
	"io"
	"os"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type shellCommandRequest struct {
	InFD           api.FileDescriptor
	OutFD          api.FileDescriptor
	ErrFD          api.FileDescriptor
	Args           []string
	ShellCallback  api.Binder
	ResultReceiver api.Binder
}

type shellTestService struct {
	registry       *shellTestBinderRegistry
	onShellCommand func(context.Context, shellCommandRequest) error
}

func (f *shellTestService) Descriptor(ctx context.Context) (string, error) {
	return "fake.shell", nil
}

func (f *shellTestService) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
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

func (f *shellTestService) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f *shellTestService) Close() error {
	return nil
}

func (f *shellTestService) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	return f.registry.register(handler), nil
}

type shellTestBinderRegistry struct {
	mu     sync.Mutex
	next   uint32
	locals map[uint32]*shellTestLocalBinder
}

func newShellTestBinderRegistry() *shellTestBinderRegistry {
	return &shellTestBinderRegistry{
		next:   1,
		locals: make(map[uint32]*shellTestLocalBinder),
	}
}

func (r *shellTestBinderRegistry) register(handler api.Handler) *shellTestLocalBinder {
	r.mu.Lock()
	defer r.mu.Unlock()
	handle := r.next
	r.next++
	b := &shellTestLocalBinder{
		handle:  handle,
		handler: handler,
	}
	r.locals[handle] = b
	return b
}

func (r *shellTestBinderRegistry) resolve(handle uint32) api.Binder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locals[handle]
}

type shellTestLocalBinder struct {
	handle  uint32
	handler api.Handler
}

func (b *shellTestLocalBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b *shellTestLocalBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
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

func (b *shellTestLocalBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *shellTestLocalBinder) Close() error {
	return nil
}

func (b *shellTestLocalBinder) WriteBinderToParcel(p *api.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
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

type fileAccessCheckerFunc func(path string, seLinuxContext string, read bool, write bool) error

func (f fileAccessCheckerFunc) CheckFileAccess(path string, seLinuxContext string, read bool, write bool) error {
	return f(path, seLinuxContext, read, write)
}

func readAllAndClose(t interface {
	Helper()
	Fatalf(string, ...any)
}, r *os.File, w *os.File) string {
	t.Helper()
	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(data)
}
