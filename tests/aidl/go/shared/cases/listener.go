package cases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

const listenerRawBinderDescriptor = "com.wdsgyj.libbinder.aidltest.shared.ListenerRawBinder"

type ListenerService struct {
	mu        sync.Mutex
	listeners []shared.IListenerCallback
}

func (s *ListenerService) RegisterListener(ctx context.Context, callback shared.IListenerCallback) error {
	if callback == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, callback)
	return nil
}

func (s *ListenerService) UnregisterListener(ctx context.Context, callback shared.IListenerCallback) error {
	if callback == nil {
		return nil
	}
	target, ok := any(callback).(binder.BinderProvider)
	if !ok || target.AsBinder() == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.listeners[:0]
	for _, candidate := range s.listeners {
		provider, ok := any(candidate).(binder.BinderProvider)
		if !ok || provider.AsBinder() == nil || !sameBinderIdentity(provider.AsBinder(), target.AsBinder()) {
			out = append(out, candidate)
		}
	}
	s.listeners = out
	return nil
}

func (s *ListenerService) Emit(ctx context.Context, value string) (int32, error) {
	s.mu.Lock()
	listeners := append([]shared.IListenerCallback(nil), s.listeners...)
	s.mu.Unlock()

	var count int32
	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		if err := listener.OnEvent(ctx, value); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *ListenerService) EchoBinder(ctx context.Context, input binder.Binder) (binder.Binder, error) {
	return input, nil
}

func VerifyListenerService(ctx context.Context, registrar binder.LocalHandlerRegistrar, svc shared.IListenerService) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	provider, ok := any(svc).(binder.BinderProvider)
	if !ok || provider.AsBinder() == nil {
		return fmt.Errorf("service does not expose binder provider")
	}
	if registrar == nil {
		if resolved, ok := provider.AsBinder().(binder.LocalHandlerRegistrar); ok {
			registrar = resolved
		}
	}
	if registrar == nil {
		return fmt.Errorf("no local handler registrar available")
	}

	rawBinder, err := registrar.RegisterLocalHandler(binder.StaticHandler{
		DescriptorName: listenerRawBinderDescriptor,
		Handle: func(ctx context.Context, code uint32, data *binder.Parcel) (*binder.Parcel, error) {
			return binder.NewParcel(), nil
		},
	})
	if err != nil {
		return fmt.Errorf("register raw binder: %w", err)
	}
	defer rawBinder.Close()

	echoed, err := svc.EchoBinder(ctx, rawBinder)
	if err != nil {
		return fmt.Errorf("EchoBinder(non-nil): %w", err)
	}
	if echoed == nil {
		return fmt.Errorf("EchoBinder(non-nil) = nil")
	}
	desc, err := echoed.Descriptor(ctx)
	if err != nil {
		return fmt.Errorf("EchoBinder descriptor: %w", err)
	}
	if desc != listenerRawBinderDescriptor {
		return fmt.Errorf("EchoBinder descriptor = %q, want %q", desc, listenerRawBinderDescriptor)
	}

	nilBinder, err := svc.EchoBinder(ctx, nil)
	if err != nil {
		return fmt.Errorf("EchoBinder(nil): %w", err)
	}
	if nilBinder != nil {
		return fmt.Errorf("EchoBinder(nil) = %v, want nil", nilBinder)
	}

	callbackRecorder := newListenerRecorder()
	callbackBinder, err := registrar.RegisterLocalHandler(shared.NewIListenerCallbackHandler(callbackRecorder))
	if err != nil {
		return fmt.Errorf("register listener callback: %w", err)
	}
	defer callbackBinder.Close()
	callback := shared.NewIListenerCallbackClient(callbackBinder)
	if err := svc.RegisterListener(ctx, callback); err != nil {
		return fmt.Errorf("RegisterListener: %w", err)
	}
	if err := svc.RegisterListener(ctx, nil); err != nil {
		return fmt.Errorf("RegisterListener(nil): %w", err)
	}

	count, err := svc.Emit(ctx, "one")
	if err != nil {
		return fmt.Errorf("Emit(one): %w", err)
	}
	if count != 1 {
		return fmt.Errorf("Emit(one) count = %d, want 1", count)
	}
	if got, err := callbackRecorder.wait(context.Background(), 2*time.Second); err != nil || got != "one" {
		return fmt.Errorf("callback one = (%q, %v), want (one, nil)", got, err)
	}

	count, err = svc.Emit(ctx, "two")
	if err != nil {
		return fmt.Errorf("Emit(two): %w", err)
	}
	if count != 1 {
		return fmt.Errorf("Emit(two) count = %d, want 1", count)
	}
	if got, err := callbackRecorder.wait(context.Background(), 2*time.Second); err != nil || got != "two" {
		return fmt.Errorf("callback two = (%q, %v), want (two, nil)", got, err)
	}

	if err := svc.UnregisterListener(ctx, nil); err != nil {
		return fmt.Errorf("UnregisterListener(nil): %w", err)
	}
	if err := svc.UnregisterListener(ctx, callback); err != nil {
		return fmt.Errorf("UnregisterListener: %w", err)
	}

	count, err = svc.Emit(ctx, "three")
	if err != nil {
		return fmt.Errorf("Emit(three): %w", err)
	}
	if count != 0 {
		return fmt.Errorf("Emit(three) count = %d, want 0", count)
	}
	if got, ok := callbackRecorder.tryRecv(); ok {
		return fmt.Errorf("unexpected callback after unregister: %q", got)
	}

	return nil
}

func VerifyListenerChurn(ctx context.Context, registrar binder.LocalHandlerRegistrar, svc shared.IListenerService, rounds int) error {
	if rounds <= 0 {
		rounds = 1
	}
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	provider, ok := any(svc).(binder.BinderProvider)
	if !ok || provider.AsBinder() == nil {
		return fmt.Errorf("service does not expose binder provider")
	}
	if registrar == nil {
		if resolved, ok := provider.AsBinder().(binder.LocalHandlerRegistrar); ok {
			registrar = resolved
		}
	}
	if registrar == nil {
		return fmt.Errorf("no local handler registrar available")
	}

	callbackRecorder := newListenerRecorder()
	callbackBinder, err := registrar.RegisterLocalHandler(shared.NewIListenerCallbackHandler(callbackRecorder))
	if err != nil {
		return fmt.Errorf("register listener callback: %w", err)
	}
	defer callbackBinder.Close()
	callback := shared.NewIListenerCallbackClient(callbackBinder)

	for i := 0; i < rounds; i++ {
		if err := svc.RegisterListener(ctx, callback); err != nil {
			return fmt.Errorf("RegisterListener(round=%d): %w", i, err)
		}
		value := fmt.Sprintf("churn-%03d", i)
		count, err := svc.Emit(ctx, value)
		if err != nil {
			return fmt.Errorf("Emit(round=%d): %w", i, err)
		}
		if count != 1 {
			return fmt.Errorf("Emit(round=%d) count = %d, want 1", i, count)
		}
		if got, err := callbackRecorder.wait(context.Background(), 2*time.Second); err != nil || got != value {
			return fmt.Errorf("callback(round=%d) = (%q, %v), want (%q, nil)", i, got, err, value)
		}
		if err := svc.UnregisterListener(ctx, callback); err != nil {
			return fmt.Errorf("UnregisterListener(round=%d): %w", i, err)
		}
		count, err = svc.Emit(ctx, value+"-after")
		if err != nil {
			return fmt.Errorf("Emit(after round=%d): %w", i, err)
		}
		if count != 0 {
			return fmt.Errorf("Emit(after round=%d) count = %d, want 0", i, count)
		}
		if got, ok := callbackRecorder.tryRecv(); ok {
			return fmt.Errorf("unexpected callback after unregister round=%d: %q", i, got)
		}
	}
	return nil
}

type listenerRecorder struct {
	ch chan string
}

func newListenerRecorder() *listenerRecorder {
	return &listenerRecorder{ch: make(chan string, 8)}
}

func (r *listenerRecorder) OnEvent(ctx context.Context, value string) error {
	select {
	case r.ch <- value:
	default:
	}
	return nil
}

func (r *listenerRecorder) wait(ctx context.Context, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case value := <-r.ch:
		return value, nil
	case <-waitCtx.Done():
		return "", waitCtx.Err()
	}
}

func (r *listenerRecorder) tryRecv() (string, bool) {
	select {
	case value := <-r.ch:
		return value, true
	default:
		return "", false
	}
}

func sameBinderIdentity(left, right binder.Binder) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left == right {
		return true
	}
	leftHandle, leftOK := debugHandle(left)
	rightHandle, rightOK := debugHandle(right)
	return leftOK && rightOK && leftHandle == rightHandle
}

func debugHandle(value binder.Binder) (uint32, bool) {
	if provider, ok := value.(binder.DebugHandleProvider); ok {
		return provider.DebugHandle()
	}
	return 0, false
}
