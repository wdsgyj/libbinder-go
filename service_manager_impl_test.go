package libbinder

import (
	"context"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
	iruntime "github.com/wdsgyj/libbinder-go/internal/runtime"
)

func TestServiceManagerCheckServiceCachesBinder(t *testing.T) {
	want := testServiceBinder{id: "cached"}
	target := &fakeServiceManagerTarget{
		transactFn: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
				t.Fatalf("WriteStatus: %v", err)
			}
			reply.SetBinderObjectResolvers(func(obj api.ParcelObject) api.Binder {
				if obj.Stability != api.StabilityVendor {
					t.Fatalf("Stability = %v, want %v", obj.Stability, api.StabilityVendor)
				}
				return want
			}, nil)
			if err := reply.WriteStrongBinderHandleWithStability(7, api.StabilityVendor); err != nil {
				t.Fatalf("WriteStrongBinderHandleWithStability: %v", err)
			}
			if err := reply.SetPosition(0); err != nil {
				t.Fatalf("SetPosition: %v", err)
			}
			return reply, nil
		},
	}

	sm := &serviceManager{target: target}
	first, err := sm.CheckService(context.Background(), "svc")
	if err != nil {
		t.Fatalf("CheckService(first): %v", err)
	}
	second, err := sm.CheckService(context.Background(), "svc")
	if err != nil {
		t.Fatalf("CheckService(second): %v", err)
	}
	if first != want || second != want {
		t.Fatalf("CheckService cached binders = (%#v, %#v), want %#v", first, second, want)
	}
	if target.transactCalls != 1 {
		t.Fatalf("transactCalls = %d, want 1", target.transactCalls)
	}
}

func TestServiceManagerAddServiceCachesLocalBinder(t *testing.T) {
	conn := &Conn{rt: iruntime.New(iruntime.Config{})}
	target := &fakeServiceManagerTarget{
		transactFn: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
				t.Fatalf("WriteStatus: %v", err)
			}
			if err := reply.SetPosition(0); err != nil {
				t.Fatalf("SetPosition: %v", err)
			}
			return reply, nil
		},
	}

	conn.sm = &serviceManager{conn: conn, target: target}
	handler := api.WithStability(api.StaticHandler{
		DescriptorName: "svc",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}, api.StabilityVINTF)

	if err := conn.sm.AddService(context.Background(), "svc", handler); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	service, err := conn.sm.CheckService(context.Background(), "svc")
	if err != nil {
		t.Fatalf("CheckService(cached local): %v", err)
	}
	local, ok := service.(*localBinder)
	if !ok {
		t.Fatalf("service = %T, want *localBinder", service)
	}
	if local.StabilityLevel() != api.StabilityVINTF {
		t.Fatalf("StabilityLevel = %v, want %v", local.StabilityLevel(), api.StabilityVINTF)
	}
	if target.transactCalls != 1 {
		t.Fatalf("transactCalls = %d, want 1", target.transactCalls)
	}
}

func TestConnDebugSnapshot(t *testing.T) {
	conn := &Conn{rt: iruntime.New(iruntime.Config{})}
	conn.sm = &serviceManager{conn: conn, cache: map[string]api.Binder{
		"svc": testServiceBinder{id: "svc"},
	}}
	conn.rt.Refs.RetainBinder(21)
	conn.rt.Refs.MarkAcquired(21)

	snapshot := conn.DebugSnapshot()
	if snapshot.ServiceManager.CacheEntries != 1 {
		t.Fatalf("CacheEntries = %d, want 1", snapshot.ServiceManager.CacheEntries)
	}
	if snapshot.Refs.HandleCount != 1 {
		t.Fatalf("HandleCount = %d, want 1", snapshot.Refs.HandleCount)
	}
	if snapshot.Kernel.DriverPath == "" {
		t.Fatal("Kernel.DriverPath is empty")
	}
}

type fakeServiceManagerTarget struct {
	transactCalls int
	transactFn    func(context.Context, uint32, *api.Parcel, api.Flags) (*api.Parcel, error)
}

func (b *fakeServiceManagerTarget) Descriptor(ctx context.Context) (string, error) {
	return serviceManagerDescriptor, nil
}

func (b *fakeServiceManagerTarget) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	b.transactCalls++
	return b.transactFn(ctx, code, data, flags)
}

func (b *fakeServiceManagerTarget) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *fakeServiceManagerTarget) Close() error {
	return nil
}

type testServiceBinder struct {
	id string
}

func (b testServiceBinder) Descriptor(ctx context.Context) (string, error) {
	return b.id, nil
}

func (b testServiceBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	return nil, api.ErrUnsupported
}

func (b testServiceBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b testServiceBinder) Close() error {
	return nil
}
