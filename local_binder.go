package libbinder

import (
	"context"
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
)

type localBinder struct {
	conn      *Conn
	nodeID    uintptr
	stability api.StabilityLevel
}

func newLocalBinder(conn *Conn, nodeID uintptr) *localBinder {
	return newLocalBinderWithStability(conn, nodeID, api.DefaultLocalStability())
}

func newLocalBinderWithStability(conn *Conn, nodeID uintptr, stability api.StabilityLevel) *localBinder {
	return &localBinder{
		conn:      conn,
		nodeID:    nodeID,
		stability: stability,
	}
}

func (b *localBinder) AsBinder() api.Binder {
	return b
}

func (b *localBinder) Descriptor(ctx context.Context) (string, error) {
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

func (b *localBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if err := b.checkOpen(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if data == nil {
		data = api.NewParcel()
	}
	data.SetBinderResolvers(b.conn.resolveBinderHandle, b.conn.resolveLocalBinder)
	data.SetBinderObjectResolvers(b.conn.resolveBinderObject, b.conn.resolveLocalBinderObject)
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}

	reply, err := b.conn.rt.Kernel.DispatchLocalTransaction(ctx, b.nodeID, code, data, uint32(flags))
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	if reply != nil {
		reply.SetBinderResolvers(b.conn.resolveBinderHandle, b.conn.resolveLocalBinder)
		reply.SetBinderObjectResolvers(b.conn.resolveBinderObject, b.conn.resolveLocalBinderObject)
	}
	return reply, nil
}

func (b *localBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *localBinder) Close() error {
	return nil
}

func (b *localBinder) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if b == nil || b.conn == nil {
		return nil, api.ErrUnsupported
	}
	return b.conn.registerLocalHandler(handler)
}

func (b *localBinder) WriteBinderToParcel(p *api.Parcel) error {
	return b.WriteBinderToParcelWithStability(p, b.stability)
}

func (b *localBinder) WriteBinderToParcelWithStability(p *api.Parcel, level api.StabilityLevel) error {
	if err := b.checkOpen(); err != nil {
		return err
	}
	return p.WriteStrongBinderLocalWithStability(b.nodeID, b.nodeID, level)
}

func (b *localBinder) StabilityLevel() api.StabilityLevel {
	if b == nil {
		return api.StabilityUndeclared
	}
	return b.stability
}

func (b *localBinder) checkOpen() error {
	if b == nil || b.conn == nil || b.conn.rt == nil || b.nodeID == 0 {
		return api.ErrUnsupported
	}
	return nil
}
