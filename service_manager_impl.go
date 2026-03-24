package libbindergo

import (
	"context"
	"errors"
	"time"

	api "libbinder-go/binder"
	"libbinder-go/internal/kernel"
	"libbinder-go/internal/protocol"
)

const (
	serviceManagerDescriptor  = "android.os.IServiceManager"
	checkServiceTransactionID = kernel.FirstCallTransaction + 2
	waitServicePollInterval   = 200 * time.Millisecond
)

type serviceManager struct {
	conn   *Conn
	target *remoteBinder
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
	m.conn.markHandleAcquired(*handle)
	return m.conn.Handle(*handle), nil
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
	return api.ErrUnsupported
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

	status, err := protocol.ReadStatus(reply)
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	if status.TransportErr != nil {
		return nil, mapRuntimeError(status.TransportErr)
	}
	if status.Remote != nil {
		return nil, &api.RemoteException{
			Code:    status.Remote.Code,
			Message: status.Remote.Message,
		}
	}

	return reply, nil
}
