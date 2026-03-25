package libbinder

import (
	"context"
	"errors"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

const (
	serviceManagerDescriptor  = "android.os.IServiceManager"
	checkServiceTransactionID = kernel.FirstCallTransaction + 2
	addServiceTransactionID   = kernel.FirstCallTransaction + 4
	waitServicePollInterval   = 200 * time.Millisecond
)

type serviceManager struct {
	conn   *Conn
	target *remoteBinder
}

func (m *serviceManager) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if m == nil || m.conn == nil {
		return nil, api.ErrUnsupported
	}
	return m.conn.registerLocalHandler(handler)
}

func (m *serviceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	reply, err := m.call(ctx, checkServiceTransactionID, name)
	if err != nil {
		return nil, err
	}

	handle, err := reply.ReadStrongBinderHandle()
	if err != nil {
		return nil, err
	}
	if handle == nil {
		return nil, api.ErrNoService
	}
	service := m.conn.Handle(*handle)
	m.conn.markHandleAcquired(*handle)
	return service, nil
}

func (m *serviceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	service, err := m.CheckService(ctx, name)
	if err == nil {
		return service, nil
	}
	if !errors.Is(err, api.ErrNoService) {
		return nil, err
	}

	ticker := time.NewTicker(waitServicePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			service, err := m.CheckService(ctx, name)
			if err == nil {
				return service, nil
			}
			if !errors.Is(err, api.ErrNoService) {
				return nil, err
			}
		}
	}
}

func (m *serviceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	resolved := api.ResolveAddServiceOptions(opts...)

	node, err := m.conn.registerLocalNode(handler, resolved.Serial)
	if err != nil {
		return err
	}

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return err
	}
	if err := data.WriteString(name); err != nil {
		return err
	}
	if err := data.WriteStrongBinderLocal(node.ID, node.ID); err != nil {
		return err
	}
	if err := data.WriteBool(resolved.AllowIsolated); err != nil {
		return err
	}
	if err := data.WriteInt32(int32(resolved.DumpFlags)); err != nil {
		return err
	}

	reply, err := m.target.Transact(ctx, addServiceTransactionID, data, api.FlagNone)
	if err != nil {
		return err
	}
	return decodeStatusReply(reply)
}

func (m *serviceManager) call(ctx context.Context, code uint32, name string) (*api.Parcel, error) {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return nil, err
	}
	if err := data.WriteString(name); err != nil {
		return nil, err
	}

	reply, err := m.target.Transact(ctx, code, data, api.FlagNone)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, api.ErrBadParcelable
	}

	if err := decodeStatusReply(reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func decodeStatusReply(reply *api.Parcel) error {
	if reply == nil {
		return api.ErrBadParcelable
	}

	status, err := protocol.ReadStatus(reply)
	if err != nil {
		return mapRuntimeError(err)
	}
	if status.TransportErr != nil {
		return mapRuntimeError(status.TransportErr)
	}
	if status.Remote != nil {
		return &api.RemoteException{
			Code:    status.Remote.Code,
			Message: status.Remote.Message,
		}
	}

	return nil
}
