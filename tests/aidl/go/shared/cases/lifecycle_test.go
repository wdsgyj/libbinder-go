package cases

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func TestVerifyLifecycleDiscovery(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	handler := shared.NewIBaselineServiceHandler(baselineLifecycleService{prefix: "go"})
	endpoint := fakeAdvancedEndpoint{handler: handler, registrar: reg}
	sm := lifecycleServiceManager{binder: endpoint, services: []string{"alpha", "life"}}

	if err := VerifyLifecycleDiscovery(context.Background(), sm, "life", "go"); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForBinderDeathAfter(t *testing.T) {
	b := &lifecycleDeathBinder{sub: newLifecycleSubscription()}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := WaitForBinderDeathAfter(ctx, b, 10*time.Millisecond, func() error {
		b.sub.finish(binder.ErrDeadObject)
		return nil
	})
	if err != nil {
		t.Fatalf("WaitForBinderDeathAfter: %v", err)
	}
}

type baselineLifecycleService struct {
	prefix string
}

func (s baselineLifecycleService) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

func (s baselineLifecycleService) EchoNullable(ctx context.Context, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	return lifecycleStringPtr(s.prefix + ":" + *value), nil
}

func (s baselineLifecycleService) Transform(ctx context.Context, input int32, payload shared.BaselinePayload) (int32, shared.BaselinePayload, shared.BaselinePayload, error) {
	return input + 1, shared.BaselinePayload{}, payload, nil
}

type lifecycleServiceManager struct {
	binder   binder.Binder
	services []string
}

func (sm lifecycleServiceManager) CheckService(ctx context.Context, name string) (binder.Binder, error) {
	return sm.binder, nil
}

func (sm lifecycleServiceManager) WaitService(ctx context.Context, name string) (binder.Binder, error) {
	return sm.binder, nil
}

func (sm lifecycleServiceManager) AddService(ctx context.Context, name string, handler binder.Handler, opts ...binder.AddServiceOption) error {
	return binder.ErrUnsupported
}

func (sm lifecycleServiceManager) ListServices(ctx context.Context, dumpFlags binder.DumpFlags) ([]string, error) {
	return append([]string(nil), sm.services...), nil
}

func (sm lifecycleServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback binder.ServiceRegistrationCallback) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) ConnectionInfo(ctx context.Context, name string) (*binder.ConnectionInfo, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) WatchClients(ctx context.Context, name string, service binder.Binder, callback binder.ServiceClientCallback) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (sm lifecycleServiceManager) TryUnregisterService(ctx context.Context, name string, service binder.Binder) error {
	return binder.ErrUnsupported
}

func (sm lifecycleServiceManager) DebugInfo(ctx context.Context) ([]binder.ServiceDebugInfo, error) {
	return nil, binder.ErrUnsupported
}

type lifecycleDeathBinder struct {
	sub *lifecycleSubscription
}

func (b *lifecycleDeathBinder) Descriptor(ctx context.Context) (string, error) {
	return "", nil
}

func (b *lifecycleDeathBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	return nil, binder.ErrUnsupported
}

func (b *lifecycleDeathBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return b.sub, nil
}

func (b *lifecycleDeathBinder) Close() error { return nil }

type lifecycleSubscription struct {
	done chan struct{}
	mu   sync.Mutex
	err  error
}

func newLifecycleSubscription() *lifecycleSubscription {
	return &lifecycleSubscription{done: make(chan struct{})}
}

func (s *lifecycleSubscription) Done() <-chan struct{} {
	return s.done
}

func (s *lifecycleSubscription) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *lifecycleSubscription) Close() error {
	s.finish(nil)
	return nil
}

func (s *lifecycleSubscription) finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil || isClosed(s.done) {
		return
	}
	s.err = err
	close(s.done)
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
