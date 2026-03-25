package libbindergo

import (
	"context"
	"fmt"

	api "libbinder-go/binder"
	"libbinder-go/internal/kernel"
)

type remoteBinder struct {
	conn   *Conn
	handle uint32
}

func (b *remoteBinder) Descriptor(ctx context.Context) (string, error) {
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
	if b == nil || b.conn == nil || b.conn.rt == nil {
		return nil, api.ErrUnsupported
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
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.watchDeath(ctx, b.handle)
}
