package libbinder

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type rpcLocalBinder struct {
	conn      *RPCConn
	handle    uint32
	stability api.StabilityLevel
}

func newRPCLocalBinder(conn *RPCConn, handle uint32, stability api.StabilityLevel) *rpcLocalBinder {
	return &rpcLocalBinder{
		conn:      conn,
		handle:    handle,
		stability: stability,
	}
}

func (b *rpcLocalBinder) AsBinder() api.Binder {
	return b
}

func (b *rpcLocalBinder) Descriptor(ctx context.Context) (string, error) {
	if b == nil || b.conn == nil {
		return "", api.ErrUnsupported
	}
	export, ok := b.conn.lookupExport(b.handle)
	if !ok {
		return "", api.ErrClosed
	}
	return export.handler.Descriptor(), nil
}

func (b *rpcLocalBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.dispatchExport(ctx, b.handle, code, data, flags, true)
}

func (b *rpcLocalBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *rpcLocalBinder) Close() error {
	return nil
}

func (b *rpcLocalBinder) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.RegisterLocalHandler(handler)
}

func (b *rpcLocalBinder) WriteBinderToParcel(p *api.Parcel) error {
	return b.WriteBinderToParcelWithStability(p, b.stability)
}

func (b *rpcLocalBinder) WriteBinderToParcelWithStability(p *api.Parcel, level api.StabilityLevel) error {
	if b == nil || b.conn == nil {
		return api.ErrUnsupported
	}
	return p.WriteStrongBinderHandleWithStability(b.handle, level)
}

func (b *rpcLocalBinder) StabilityLevel() api.StabilityLevel {
	if b == nil {
		return api.StabilityUndeclared
	}
	return b.stability
}

type rpcRemoteBinder struct {
	conn      *RPCConn
	handle    uint32
	stability api.StabilityLevel

	closed atomic.Bool

	descriptorMu     sync.RWMutex
	descriptor       string
	descriptorCached bool
}

func newRPCRemoteBinder(conn *RPCConn, handle uint32, stability api.StabilityLevel) *rpcRemoteBinder {
	return &rpcRemoteBinder{
		conn:      conn,
		handle:    handle,
		stability: stability,
	}
}

func (b *rpcRemoteBinder) AsBinder() api.Binder {
	return b
}

func (b *rpcRemoteBinder) Descriptor(ctx context.Context) (string, error) {
	if err := b.checkOpen(); err != nil {
		return "", err
	}

	b.descriptorMu.RLock()
	if b.descriptorCached {
		desc := b.descriptor
		b.descriptorMu.RUnlock()
		return desc, nil
	}
	b.descriptorMu.RUnlock()

	reply, err := b.Transact(ctx, api.InterfaceTransaction, api.NewParcel(), api.FlagNone)
	if err != nil {
		return "", err
	}
	if reply == nil {
		return "", fmt.Errorf("%w: interface descriptor reply was nil", api.ErrBadParcelable)
	}
	desc, err := reply.ReadString()
	if err != nil {
		return "", err
	}

	b.descriptorMu.Lock()
	if !b.descriptorCached {
		b.descriptor = desc
		b.descriptorCached = true
	}
	desc = b.descriptor
	b.descriptorMu.Unlock()
	return desc, nil
}

func (b *rpcRemoteBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if err := b.checkOpen(); err != nil {
		return nil, err
	}
	return b.conn.transactRemote(ctx, b.handle, code, data, flags)
}

func (b *rpcRemoteBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *rpcRemoteBinder) Close() error {
	if b == nil {
		return nil
	}
	b.closed.Store(true)
	return nil
}

func (b *rpcRemoteBinder) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.RegisterLocalHandler(handler)
}

func (b *rpcRemoteBinder) WriteBinderToParcel(p *api.Parcel) error {
	return b.WriteBinderToParcelWithStability(p, b.stability)
}

func (b *rpcRemoteBinder) WriteBinderToParcelWithStability(p *api.Parcel, level api.StabilityLevel) error {
	if err := b.checkOpen(); err != nil {
		return err
	}
	return p.WriteStrongBinderHandleWithStability(b.handle, level)
}

func (b *rpcRemoteBinder) StabilityLevel() api.StabilityLevel {
	if b == nil {
		return api.StabilityUndeclared
	}
	return b.stability
}

func (b *rpcRemoteBinder) checkOpen() error {
	if b == nil || b.conn == nil {
		return api.ErrUnsupported
	}
	if b.closed.Load() {
		return api.ErrClosed
	}
	select {
	case <-b.conn.closed:
		return api.ErrClosed
	default:
		return nil
	}
}
