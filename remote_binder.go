package libbinder

import (
	"context"
	"errors"
	"fmt"
	stdruntime "runtime"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
)

type remoteBinder struct {
	conn   *Conn
	handle uint32

	closed atomic.Bool
}

func newRemoteBinder(conn *Conn, handle uint32) *remoteBinder {
	b := &remoteBinder{
		conn:   conn,
		handle: handle,
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

func (b *remoteBinder) Descriptor(ctx context.Context) (string, error) {
	if err := b.checkOpen(); err != nil {
		return "", err
	}
	reply, err := b.Transact(ctx, kernel.InterfaceTransaction, api.NewParcel(), api.FlagNone)
	if err != nil {
		return "", err
	}
	if reply == nil {
		return "", fmt.Errorf("%w: interface descriptor reply was nil", api.ErrBadParcelable)
	}
	return reply.ReadString()
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
