package libbinder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
)

const (
	serviceManagerDescriptor    = "android.os.IServiceManager"
	serviceCallbackDescriptor   = "android.os.IServiceCallback"
	clientCallbackDescriptor    = "android.os.IClientCallback"
	getServiceTransactionID     = kernel.FirstCallTransaction + 0
	getService2TransactionID    = kernel.FirstCallTransaction + 1
	checkServiceTransactionID   = kernel.FirstCallTransaction + 2
	checkService2TransactionID  = kernel.FirstCallTransaction + 3
	addServiceTransactionID     = kernel.FirstCallTransaction + 4
	listServicesTransactionID   = kernel.FirstCallTransaction + 5
	registerNotifyTransactionID = kernel.FirstCallTransaction + 6
	unregisterNotifyTxID        = kernel.FirstCallTransaction + 7
	isDeclaredTransactionID     = kernel.FirstCallTransaction + 8
	declaredInstancesTxID       = kernel.FirstCallTransaction + 9
	updatableViaApexTxID        = kernel.FirstCallTransaction + 10
	updatableNamesTxID          = kernel.FirstCallTransaction + 11
	connectionInfoTxID          = kernel.FirstCallTransaction + 12
	registerClientCallbackTxID  = kernel.FirstCallTransaction + 13
	tryUnregisterServiceTxID    = kernel.FirstCallTransaction + 14
	debugInfoTransactionID      = kernel.FirstCallTransaction + 15
)

type waitableServiceManager interface {
	CheckService(context.Context, string) (api.Binder, error)
	WatchServiceRegistrations(context.Context, string, api.ServiceRegistrationCallback) (api.Subscription, error)
}

func waitForServiceWithNotifications(ctx context.Context, sm waitableServiceManager, name string) (api.Binder, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	service, err := sm.CheckService(ctx, name)
	if err == nil {
		return service, nil
	}
	if !errors.Is(err, api.ErrNoService) {
		return nil, err
	}

	found := make(chan api.Binder, 1)
	sub, err := sm.WatchServiceRegistrations(ctx, name, func(_ context.Context, reg api.ServiceRegistration) {
		select {
		case found <- reg.Binder:
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = sub.Close() }()

	service, err = sm.CheckService(ctx, name)
	if err == nil {
		return service, nil
	}
	if err != nil && !errors.Is(err, api.ErrNoService) {
		return nil, err
	}

	select {
	case service = <-found:
		if service == nil {
			return nil, api.ErrNoService
		}
		return service, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-sub.Done():
		if sub.Err() != nil {
			return nil, sub.Err()
		}
		return nil, api.ErrNoService
	}
}

type managedSubscription struct {
	done chan struct{}

	closeFn func() error

	mu     sync.Mutex
	err    error
	active bool
	once   sync.Once
}

func newManagedSubscription(ctx context.Context, closeFn func() error) *managedSubscription {
	sub := &managedSubscription{
		done:    make(chan struct{}),
		closeFn: closeFn,
		active:  true,
	}
	if ctx != nil {
		if done := ctx.Done(); done != nil {
			go func() {
				select {
				case <-done:
					_ = sub.Close()
				case <-sub.done:
				}
			}()
		}
	}
	return sub
}

func (s *managedSubscription) Done() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.done
}

func (s *managedSubscription) Err() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *managedSubscription) Active() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

func (s *managedSubscription) Close() error {
	if s == nil {
		return nil
	}

	var err error
	s.once.Do(func() {
		if s.closeFn != nil {
			err = s.closeFn()
		}
		s.mu.Lock()
		s.active = false
		s.err = err
		close(s.done)
		s.mu.Unlock()
	})
	return err
}

func (s *managedSubscription) finish(err error) {
	if s == nil {
		return
	}

	s.once.Do(func() {
		s.mu.Lock()
		s.active = false
		s.err = err
		close(s.done)
		s.mu.Unlock()
	})
}

type serviceRegistrationCallbackHandler struct {
	sub      *managedSubscription
	callback api.ServiceRegistrationCallback
}

func (h *serviceRegistrationCallbackHandler) Descriptor() string {
	return serviceCallbackDescriptor
}

func (h *serviceRegistrationCallbackHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	descriptor, err := data.ReadInterfaceToken()
	if err != nil {
		return nil, err
	}
	if descriptor != serviceCallbackDescriptor {
		return nil, fmt.Errorf("%w: descriptor %q, want %q", api.ErrBadParcelable, descriptor, serviceCallbackDescriptor)
	}
	if code != api.FirstCallTransaction {
		return nil, api.ErrUnknownTransaction
	}

	name, err := data.ReadString()
	if err != nil {
		return nil, err
	}
	binder, err := data.ReadStrongBinder()
	if err != nil {
		return nil, err
	}
	if h.sub != nil && h.sub.Active() && h.callback != nil {
		h.callback(ctx, api.ServiceRegistration{Name: name, Binder: binder})
	}
	return nil, nil
}

type serviceClientCallbackHandler struct {
	sub      *managedSubscription
	callback api.ServiceClientCallback
}

func (h *serviceClientCallbackHandler) Descriptor() string {
	return clientCallbackDescriptor
}

func (h *serviceClientCallbackHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	descriptor, err := data.ReadInterfaceToken()
	if err != nil {
		return nil, err
	}
	if descriptor != clientCallbackDescriptor {
		return nil, fmt.Errorf("%w: descriptor %q, want %q", api.ErrBadParcelable, descriptor, clientCallbackDescriptor)
	}
	if code != api.FirstCallTransaction {
		return nil, api.ErrUnknownTransaction
	}

	service, err := data.ReadStrongBinder()
	if err != nil {
		return nil, err
	}
	hasClients, err := data.ReadBool()
	if err != nil {
		return nil, err
	}
	if h.sub != nil && h.sub.Active() && h.callback != nil {
		h.callback(ctx, api.ServiceClientUpdate{
			Service:    service,
			HasClients: hasClients,
		})
	}
	return nil, nil
}

func writeStringSliceToParcel(p *api.Parcel, values []string) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value string) error {
		return p.WriteString(value)
	})
}

func readStringSliceFromParcel(p *api.Parcel) ([]string, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	})
}

func writeNullableConnectionInfoToParcel(p *api.Parcel, info *api.ConnectionInfo) error {
	if info == nil {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return writeSizedParcelable(p, func(p *api.Parcel) error {
		if err := p.WriteString(info.IPAddress); err != nil {
			return err
		}
		if info.Port > uint32(^uint32(0)>>1) {
			return fmt.Errorf("%w: connection info port %d out of range", api.ErrBadParcelable, info.Port)
		}
		return p.WriteInt32(int32(info.Port))
	})
}

func readNullableConnectionInfoFromParcel(p *api.Parcel) (*api.ConnectionInfo, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if present == 0 {
		return nil, nil
	}

	start := p.Position()
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	end, err := parcelableEnd(start, size, p.Len())
	if err != nil {
		return nil, err
	}

	var info api.ConnectionInfo
	if p.Position() < end {
		info.IPAddress, err = p.ReadString()
		if err != nil {
			return nil, err
		}
	}
	if p.Position() < end {
		port, err := p.ReadInt32()
		if err != nil {
			return nil, err
		}
		info.Port = uint32(port)
	}
	if err := p.SetPosition(end); err != nil {
		return nil, err
	}
	return &info, nil
}

func writeServiceDebugInfoSliceToParcel(p *api.Parcel, values []api.ServiceDebugInfo) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value api.ServiceDebugInfo) error {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		return writeSizedParcelable(p, func(p *api.Parcel) error {
			if err := p.WriteString(value.Name); err != nil {
				return err
			}
			return p.WriteInt32(value.DebugPID)
		})
	})
}

func readServiceDebugInfoSliceFromParcel(p *api.Parcel) ([]api.ServiceDebugInfo, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (api.ServiceDebugInfo, error) {
		present, err := p.ReadInt32()
		if err != nil {
			return api.ServiceDebugInfo{}, err
		}
		if present == 0 {
			return api.ServiceDebugInfo{}, fmt.Errorf("%w: unexpected null service debug info", api.ErrBadParcelable)
		}

		start := p.Position()
		size, err := p.ReadInt32()
		if err != nil {
			return api.ServiceDebugInfo{}, err
		}
		end, err := parcelableEnd(start, size, p.Len())
		if err != nil {
			return api.ServiceDebugInfo{}, err
		}

		var info api.ServiceDebugInfo
		if p.Position() < end {
			info.Name, err = p.ReadString()
			if err != nil {
				return api.ServiceDebugInfo{}, err
			}
		}
		if p.Position() < end {
			info.DebugPID, err = p.ReadInt32()
			if err != nil {
				return api.ServiceDebugInfo{}, err
			}
		}
		if err := p.SetPosition(end); err != nil {
			return api.ServiceDebugInfo{}, err
		}
		return info, nil
	})
}

func writeSizedParcelable(p *api.Parcel, writeBody func(*api.Parcel) error) error {
	start := p.Position()
	if err := p.WriteInt32(0); err != nil {
		return err
	}
	if err := writeBody(p); err != nil {
		return err
	}
	end := p.Position()
	if err := p.SetPosition(start); err != nil {
		return err
	}
	if err := p.WriteInt32(int32(end - start)); err != nil {
		return err
	}
	return p.SetPosition(end)
}

func parcelableEnd(start int, size int32, total int) (int, error) {
	if size < 4 {
		return 0, fmt.Errorf("%w: invalid parcelable size %d", api.ErrBadParcelable, size)
	}
	end := start + int(size)
	if end < start || end > total {
		return 0, io.ErrUnexpectedEOF
	}
	return end, nil
}

type binderIdentity struct {
	owner any
	kind  uint8
	id    uint64
}

func binderKey(b api.Binder) (binderIdentity, bool) {
	switch v := b.(type) {
	case *remoteBinder:
		return binderIdentity{owner: v.conn, kind: 1, id: uint64(v.handle)}, true
	case *localBinder:
		return binderIdentity{owner: v.conn, kind: 2, id: uint64(v.nodeID)}, true
	case *rpcRemoteBinder:
		return binderIdentity{owner: v.conn, kind: 3, id: uint64(v.handle)}, true
	case *rpcLocalBinder:
		return binderIdentity{owner: v.conn, kind: 4, id: uint64(v.handle)}, true
	default:
		return binderIdentity{}, false
	}
}
