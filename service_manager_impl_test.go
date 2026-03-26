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

	sm := &serviceManager{target: target, txProfile: serviceManagerTransactionsCurrent, txKnown: true}
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

	conn.sm = &serviceManager{conn: conn, target: target, txProfile: serviceManagerTransactionsCurrent, txKnown: true}
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

func TestServiceManagerDetectsLegacyAndroid15Transactions(t *testing.T) {
	target := &fakeServiceManagerTarget{
		transactFn: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			switch code {
			case serviceManagerTransactionsCurrent.checkService:
				return statusReplyWithRemoteException(t, api.ExceptionNullPointer), nil
			case serviceManagerTransactionsLegacy13.checkService:
				return statusReplyWithNullableBinder(t, nil), nil
			case serviceManagerTransactionsLegacy13.connectionInfo:
				return statusReplyWithConnectionInfo(t, nil), nil
			case serviceManagerTransactionsLegacy13.listServices:
				return statusReplyWithStrings(t, "alpha", "beta"), nil
			default:
				t.Fatalf("unexpected transact code %d", code)
				return nil, nil
			}
		},
	}

	sm := &serviceManager{target: target}
	names, err := sm.ListServices(context.Background(), api.DumpPriorityAll)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if got, want := len(names), 2; got != want || names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("ListServices names = %#v, want [alpha beta]", names)
	}
	if got, want := target.codes, []uint32{
		serviceManagerTransactionsCurrent.checkService,
		serviceManagerTransactionsLegacy13.checkService,
		serviceManagerTransactionsLegacy13.connectionInfo,
		serviceManagerTransactionsLegacy13.listServices,
	}; len(got) != len(want) {
		t.Fatalf("transact codes = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("transact codes = %#v, want %#v", got, want)
			}
		}
	}

	target.codes = nil
	names, err = sm.ListServices(context.Background(), api.DumpPriorityAll)
	if err != nil {
		t.Fatalf("ListServices(second): %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("ListServices(second) names = %#v, want 2 entries", names)
	}
	if got, want := target.codes, []uint32{serviceManagerTransactionsLegacy13.listServices}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("cached transact codes = %#v, want %#v", got, want)
	}
}

func TestServiceManagerDetectsCurrentTransactions(t *testing.T) {
	target := &fakeServiceManagerTarget{
		transactFn: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			switch code {
			case serviceManagerTransactionsCurrent.checkService:
				return statusReplyWithNullableBinder(t, nil), nil
			case serviceManagerTransactionsCurrent.listServices:
				return statusReplyWithStrings(t, "svc"), nil
			default:
				t.Fatalf("unexpected transact code %d", code)
				return nil, nil
			}
		},
	}

	sm := &serviceManager{target: target}
	names, err := sm.ListServices(context.Background(), api.DumpPriorityAll)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(names) != 1 || names[0] != "svc" {
		t.Fatalf("ListServices names = %#v, want [svc]", names)
	}
	if got, want := target.codes, []uint32{
		serviceManagerTransactionsCurrent.checkService,
		serviceManagerTransactionsCurrent.listServices,
	}; len(got) != len(want) {
		t.Fatalf("transact codes = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("transact codes = %#v, want %#v", got, want)
			}
		}
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
	codes         []uint32
	transactFn    func(context.Context, uint32, *api.Parcel, api.Flags) (*api.Parcel, error)
}

func (b *fakeServiceManagerTarget) Descriptor(ctx context.Context) (string, error) {
	return serviceManagerDescriptor, nil
}

func (b *fakeServiceManagerTarget) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	b.transactCalls++
	b.codes = append(b.codes, code)
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

func statusReplyWithNullableBinder(t *testing.T, binder api.Binder) *api.Parcel {
	t.Helper()
	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	if err := reply.WriteStrongBinder(binder); err != nil {
		t.Fatalf("WriteStrongBinder: %v", err)
	}
	if err := reply.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	return reply
}

func statusReplyWithRemoteException(t *testing.T, code api.ExceptionCode) *api.Parcel {
	t.Helper()
	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{
		Remote: &protocol.RemoteException{Code: code},
	}); err != nil {
		t.Fatalf("WriteStatus(remote): %v", err)
	}
	if err := reply.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	return reply
}

func statusReplyWithStrings(t *testing.T, values ...string) *api.Parcel {
	t.Helper()
	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	if err := writeStringSliceToParcel(reply, values); err != nil {
		t.Fatalf("writeStringSliceToParcel: %v", err)
	}
	if err := reply.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	return reply
}

func statusReplyWithConnectionInfo(t *testing.T, info *api.ConnectionInfo) *api.Parcel {
	t.Helper()
	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	if err := writeNullableConnectionInfoToParcel(reply, info); err != nil {
		t.Fatalf("writeNullableConnectionInfoToParcel: %v", err)
	}
	if err := reply.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	return reply
}
