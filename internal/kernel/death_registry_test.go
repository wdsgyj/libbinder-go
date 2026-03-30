package kernel

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestDeathRegistryReusesKernelWatchPerHandle(t *testing.T) {
	type requestCall struct {
		handle uint32
		cookie uintptr
	}

	var calls []requestCall
	registry := newDeathRegistry(func(_ context.Context, handle uint32, cookie uintptr) error {
		calls = append(calls, requestCall{handle: handle, cookie: cookie})
		return nil
	}, func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	})

	first, err := registry.Watch(context.Background(), 23)
	if err != nil {
		t.Fatalf("first Watch: %v", err)
	}
	second, err := registry.Watch(context.Background(), 23)
	if err != nil {
		t.Fatalf("second Watch: %v", err)
	}

	if got := len(calls); got != 1 {
		t.Fatalf("request calls = %d, want 1", got)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	waitDone(t, first.Done(), "first.Done")

	if err := second.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	registry.NotifyCleared(calls[0].cookie)
	waitDone(t, second.Done(), "second.Done")
}

func TestDeathRegistryNotifyDeadFinishesSubscribers(t *testing.T) {
	var cookie uintptr
	registry := newDeathRegistry(func(_ context.Context, _ uint32, got uintptr) error {
		cookie = got
		return nil
	}, func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	})

	first, err := registry.Watch(context.Background(), 7)
	if err != nil {
		t.Fatalf("first Watch: %v", err)
	}
	second, err := registry.Watch(context.Background(), 7)
	if err != nil {
		t.Fatalf("second Watch: %v", err)
	}

	registry.NotifyDead(cookie)

	waitDone(t, first.Done(), "first.Done")
	waitDone(t, second.Done(), "second.Done")

	if !errors.Is(first.Err(), api.ErrDeadObject) {
		t.Fatalf("first Err = %v, want %v", first.Err(), api.ErrDeadObject)
	}
	if !errors.Is(second.Err(), api.ErrDeadObject) {
		t.Fatalf("second Err = %v, want %v", second.Err(), api.ErrDeadObject)
	}
}

func TestDeathRegistryCloseAfterNotifyDeadIsNoop(t *testing.T) {
	var cookie uintptr
	registry := newDeathRegistry(func(_ context.Context, _ uint32, got uintptr) error {
		cookie = got
		return nil
	}, func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	})

	sub, err := registry.Watch(context.Background(), 29)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	registry.NotifyDead(cookie)
	waitDone(t, sub.Done(), "sub.Done")

	if err := sub.Close(); err != nil {
		t.Fatalf("Close after NotifyDead: %v", err)
	}
}

func TestDeathRegistryContextCancellationClosesSubscription(t *testing.T) {
	var registry *deathRegistry
	registry = newDeathRegistry(func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	}, func(_ context.Context, _ uint32, cookie uintptr) error {
		go registry.NotifyCleared(cookie)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := registry.Watch(ctx, 11)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	cancel()
	waitDone(t, sub.Done(), "sub.Done")

	if err := sub.Err(); err != nil {
		t.Fatalf("Err = %v, want nil", err)
	}
}

func TestDeathRegistryRequestErrorRollsBackHandle(t *testing.T) {
	wantErr := errors.New("boom")
	registry := newDeathRegistry(func(_ context.Context, _ uint32, _ uintptr) error {
		return wantErr
	}, func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	})

	sub, err := registry.Watch(context.Background(), 5)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Watch err = %v, want %v", err, wantErr)
	}
	if sub != nil {
		t.Fatalf("Watch returned sub on error: %#v", sub)
	}

	if got := len(registry.byHandle); got != 0 {
		t.Fatalf("len(byHandle) = %d, want 0", got)
	}
	if got := len(registry.byCookie); got != 0 {
		t.Fatalf("len(byCookie) = %d, want 0", got)
	}
}

func TestDeathRegistryCloseLastSubscriberClearsKernelWatch(t *testing.T) {
	type clearCall struct {
		handle uint32
		cookie uintptr
	}

	var cleared []clearCall
	registry := newDeathRegistry(func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	}, func(_ context.Context, handle uint32, cookie uintptr) error {
		cleared = append(cleared, clearCall{handle: handle, cookie: cookie})
		return nil
	})

	sub, err := registry.Watch(context.Background(), 17)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := len(cleared); got != 1 {
		t.Fatalf("clear calls = %d, want 1", got)
	}

	select {
	case <-sub.Done():
		t.Fatal("sub.Done closed before clear ack")
	default:
	}

	registry.NotifyCleared(cleared[0].cookie)
	waitDone(t, sub.Done(), "sub.Done")
}

func TestDeathRegistryCloseLastSubscriberClearFailureFinishesSubscription(t *testing.T) {
	wantErr := errors.New("clear failed")
	registry := newDeathRegistry(func(_ context.Context, _ uint32, _ uintptr) error {
		return nil
	}, func(_ context.Context, _ uint32, _ uintptr) error {
		return wantErr
	})

	sub, err := registry.Watch(context.Background(), 19)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	if err := sub.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("Close err = %v, want %v", err, wantErr)
	}
	waitDone(t, sub.Done(), "sub.Done")
}

func waitDone(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}
