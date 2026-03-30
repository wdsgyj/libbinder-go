package cases

import (
	"context"
	"testing"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func TestVerifyListenerService(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewIListenerServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIListenerServiceHandlerWithRegistrar(reg, &ListenerService{}),
		registrar: reg,
	})
	if err := VerifyListenerService(context.Background(), reg, client); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyListenerChurn(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewIListenerServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIListenerServiceHandlerWithRegistrar(reg, &ListenerService{}),
		registrar: reg,
	})
	if err := VerifyListenerChurn(context.Background(), reg, client, 8); err != nil {
		t.Fatal(err)
	}
}

func TestListenerServiceUnregistersMatchingBinder(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	svc := &ListenerService{}

	firstBinder, err := reg.RegisterLocalHandler(shared.NewIListenerCallbackHandler(listenerNoopCallback{}))
	if err != nil {
		t.Fatalf("RegisterLocalHandler(first): %v", err)
	}
	secondBinder, err := reg.RegisterLocalHandler(shared.NewIListenerCallbackHandler(listenerNoopCallback{}))
	if err != nil {
		t.Fatalf("RegisterLocalHandler(second): %v", err)
	}

	first := shared.NewIListenerCallbackClient(firstBinder)
	second := shared.NewIListenerCallbackClient(secondBinder)

	if err := svc.RegisterListener(context.Background(), first); err != nil {
		t.Fatalf("RegisterListener(first): %v", err)
	}
	if err := svc.RegisterListener(context.Background(), second); err != nil {
		t.Fatalf("RegisterListener(second): %v", err)
	}
	if err := svc.UnregisterListener(context.Background(), first); err != nil {
		t.Fatalf("UnregisterListener(first): %v", err)
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.listeners) != 1 {
		t.Fatalf("listener count = %d, want 1", len(svc.listeners))
	}
	provider, ok := any(svc.listeners[0]).(binder.BinderProvider)
	if !ok || provider.AsBinder() != secondBinder {
		t.Fatalf("remaining listener binder = %#v, want %#v", provider, secondBinder)
	}
}

func TestSameBinderIdentityMatchesDebugHandle(t *testing.T) {
	left := listenerHandleBinder{handle: 41}
	right := listenerHandleBinder{handle: 41}
	if !sameBinderIdentity(left, right) {
		t.Fatal("sameBinderIdentity(handle=41, handle=41) = false, want true")
	}
	if sameBinderIdentity(left, listenerHandleBinder{handle: 42}) {
		t.Fatal("sameBinderIdentity(handle=41, handle=42) = true, want false")
	}
}

type listenerNoopCallback struct{}

func (listenerNoopCallback) OnEvent(ctx context.Context, value string) error {
	return nil
}

type listenerHandleBinder struct {
	handle uint32
}

func (b listenerHandleBinder) Descriptor(ctx context.Context) (string, error) {
	return "", nil
}

func (b listenerHandleBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	return nil, nil
}

func (b listenerHandleBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b listenerHandleBinder) Close() error { return nil }

func (b listenerHandleBinder) DebugHandle() (uint32, bool) {
	return b.handle, true
}
