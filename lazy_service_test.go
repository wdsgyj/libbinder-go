package libbinder

import (
	"context"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestAddLazyService(t *testing.T) {
	var (
		added api.Handler
		calls int
	)
	sm := fakeServiceManager{
		addFn: func(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
			if name != "svc" {
				t.Fatalf("name = %q, want svc", name)
			}
			added = handler
			return nil
		},
	}

	if err := AddLazyService(context.Background(), sm, "svc", "lazy.desc", func() (api.Handler, error) {
		calls++
		return api.StaticHandler{
			DescriptorName: "inner",
			Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
				return api.NewParcel(), nil
			},
		}, nil
	}); err != nil {
		t.Fatalf("AddLazyService: %v", err)
	}

	if added == nil {
		t.Fatal("added handler is nil")
	}
	if added.Descriptor() != "lazy.desc" {
		t.Fatalf("Descriptor = %q, want lazy.desc", added.Descriptor())
	}
	if calls != 0 {
		t.Fatalf("factory calls = %d, want 0 before transact", calls)
	}
	if _, err := added.HandleTransact(context.Background(), api.FirstCallTransaction, api.NewParcel()); err != nil {
		t.Fatalf("HandleTransact: %v", err)
	}
	if calls != 1 {
		t.Fatalf("factory calls = %d, want 1", calls)
	}
}

type fakeServiceManager struct {
	addFn func(context.Context, string, api.Handler, ...api.AddServiceOption) error
}

func (m fakeServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return m.addFn(ctx, name, handler, opts...)
}

func (m fakeServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, api.ErrUnsupported
}

func (m fakeServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (m fakeServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	return api.ErrUnsupported
}

func (m fakeServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return nil, api.ErrUnsupported
}
