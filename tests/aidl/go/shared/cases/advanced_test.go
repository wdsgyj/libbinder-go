package cases

import (
	"context"
	"os"
	"testing"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type fakeAdvancedEndpoint struct {
	handler   binder.Handler
	registrar *fakeAdvancedRegistrar
}

type fakeAdvancedRegistrar struct {
	next    uint32
	binders map[uint32]binder.Binder
}

func newFakeAdvancedRegistrar() *fakeAdvancedRegistrar {
	return &fakeAdvancedRegistrar{
		next:    100,
		binders: map[uint32]binder.Binder{},
	}
}

func (r *fakeAdvancedRegistrar) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	r.next++
	b := fakeAdvancedRegisteredBinder{handle: r.next, handler: handler, registrar: r}
	r.binders[b.handle] = b
	return b, nil
}

func (r *fakeAdvancedRegistrar) resolve(handle uint32) binder.Binder {
	return r.binders[handle]
}

type fakeAdvancedRegisteredBinder struct {
	handle    uint32
	handler   binder.Handler
	registrar *fakeAdvancedRegistrar
}

func (b fakeAdvancedRegisteredBinder) AsBinder() binder.Binder { return b }

func (b fakeAdvancedRegisteredBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeAdvancedRegisteredBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeAdvancedRegisteredBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeAdvancedRegisteredBinder) Close() error { return nil }

func (b fakeAdvancedRegisteredBinder) WriteBinderToParcel(p *binder.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

func (b fakeAdvancedEndpoint) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeAdvancedEndpoint) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := binder.DispatchLocalHandler(ctx, b.handler, nil, code, data, flags, binder.TransactionContext{})
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeAdvancedEndpoint) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeAdvancedEndpoint) Close() error { return nil }

func (b fakeAdvancedEndpoint) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	return b.registrar.RegisterLocalHandler(handler)
}

type advancedServiceImpl struct {
	prefix string
}

func (s advancedServiceImpl) EchoBinder(ctx context.Context, input binder.Binder) (binder.Binder, error) {
	return input, nil
}

func (s advancedServiceImpl) InvokeCallback(ctx context.Context, callback shared.IAdvancedCallback, value string) (string, error) {
	return InvokeAdvancedCallback(ctx, s.prefix, callback, value)
}

func (s advancedServiceImpl) FireOneway(ctx context.Context, callback shared.IAdvancedCallback, value string) error {
	return FireAdvancedOneway(ctx, s.prefix, callback, value)
}

func (s advancedServiceImpl) FailServiceSpecific(ctx context.Context, code int32, message string) error {
	return &binder.ServiceSpecificError{Code: code, Message: message}
}

func (s advancedServiceImpl) ReadFromFileDescriptor(ctx context.Context, fd binder.FileDescriptor) (string, error) {
	return ReadAllFromFileDescriptor(fd)
}

func (s advancedServiceImpl) ReadFromParcelFileDescriptor(ctx context.Context, fd binder.ParcelFileDescriptor) (string, error) {
	return ReadAllFromParcelFileDescriptor(fd)
}

func TestVerifyAdvancedService(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewIAdvancedServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIAdvancedServiceHandlerWithRegistrar(reg, advancedServiceImpl{prefix: "go"}),
		registrar: reg,
	})
	if err := VerifyAdvancedService(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}

func TestReadAllFromFileDescriptor(t *testing.T) {
	f, err := os.CreateTemp("", "advanced-read-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString("payload"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close(write): %v", err)
	}

	in, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer in.Close()

	got, err := ReadAllFromFileDescriptor(binder.NewFileDescriptor(int(in.Fd())))
	if err != nil {
		t.Fatalf("ReadAllFromFileDescriptor: %v", err)
	}
	if got != "payload" {
		t.Fatalf("ReadAllFromFileDescriptor = %q, want %q", got, "payload")
	}
}
