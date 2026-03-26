package libbinder

import (
	"context"
	"errors"
	"fmt"
	stdruntime "runtime"
	"sync"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
)

type remoteBinder struct {
	conn      *Conn
	handle    uint32
	stability api.StabilityLevel

	closed atomic.Bool

	descriptorMu     sync.RWMutex
	descriptor       string
	descriptorCached bool
}

func newRemoteBinder(conn *Conn, handle uint32) *remoteBinder {
	return newRemoteBinderWithStability(conn, handle, api.DefaultLocalStability())
}

func newRemoteBinderWithStability(conn *Conn, handle uint32, stability api.StabilityLevel) *remoteBinder {
	b := &remoteBinder{
		conn:      conn,
		handle:    handle,
		stability: stability,
	}
	if conn != nil && handle != 0 {
		conn.retainBinder(handle)
		stdruntime.SetFinalizer(b, finalizeRemoteBinder)
	}
	return b
}

func finalizeRemoteBinder(b *remoteBinder) {
	_ = b.close(false)
}

func (b *remoteBinder) AsBinder() api.Binder {
	return b
}

func (b *remoteBinder) Descriptor(ctx context.Context) (string, error) {
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

	reply, err := b.Transact(ctx, kernel.InterfaceTransaction, api.NewParcel(), api.FlagNone)
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

func (b *remoteBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if err := b.checkOpen(); err != nil {
		return nil, err
	}
	if err := b.conn.ensureHandleAcquired(ctx, b.handle); err != nil {
		return nil, err
	}
	reply, err := b.conn.rt.TransactHandle(ctx, b.handle, code, data, flags)
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	return reply, nil
}

func (b *remoteBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	if err := b.checkOpen(); err != nil {
		return nil, err
	}
	return b.conn.watchDeath(ctx, b.handle)
}

func (b *remoteBinder) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.registerLocalHandler(handler)
}

func (b *remoteBinder) WriteBinderToParcel(p *api.Parcel) error {
	return b.WriteBinderToParcelWithStability(p, b.stability)
}

func (b *remoteBinder) WriteBinderToParcelWithStability(p *api.Parcel, level api.StabilityLevel) error {
	if err := b.checkOpen(); err != nil {
		return err
	}
	return p.WriteStrongBinderHandleWithStability(b.handle, level)
}

func (b *remoteBinder) StabilityLevel() api.StabilityLevel {
	if b == nil {
		return api.StabilityUndeclared
	}
	return b.stability
}

func (b *remoteBinder) Close() error {
	return b.close(true)
}

func (b *remoteBinder) close(explicit bool) error {
	if b == nil {
		return nil
	}
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}
	if explicit {
		stdruntime.SetFinalizer(b, nil)
	}
	if b.handle == 0 || b.conn == nil {
		return nil
	}
	err := b.conn.releaseBinder(b.handle)
	if errors.Is(err, kernel.ErrDriverClosed) {
		return nil
	}
	return err
}

func (b *remoteBinder) checkOpen() error {
	if b == nil || b.conn == nil || b.conn.rt == nil {
		return api.ErrUnsupported
	}
	if b.closed.Load() {
		return api.ErrClosed
	}
	return nil
}
