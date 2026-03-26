package libbinder

import (
	"context"
	"errors"
	"sync"
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
	target api.Binder

	mu          sync.RWMutex
	cache       map[string]api.Binder
	cacheHits   int
	cacheMisses int
}

func (m *serviceManager) RegisterLocalHandler(handler api.Handler) (api.Binder, error) {
	if m == nil || m.conn == nil {
		return nil, api.ErrUnsupported
	}
	return m.conn.registerLocalHandler(handler)
}

func (m *serviceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if service, ok := m.cachedService(name); ok {
		return service, nil
	}

	reply, err := m.call(ctx, checkServiceTransactionID, name)
	if err != nil {
		return nil, err
	}

	service, err := reply.ReadStrongBinder()
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, api.ErrNoService
	}
	m.storeService(name, service)
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
	if err := data.WriteStrongBinderLocalWithStability(node.ID, node.ID, node.Stability); err != nil {
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
	if err := decodeStatusReply(reply); err != nil {
		return err
	}

	m.storeService(name, newLocalBinderWithStability(m.conn, node.ID, node.Stability))
	return nil
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

func (m *serviceManager) cachedService(name string) (api.Binder, bool) {
	if m == nil || name == "" {
		return nil, false
	}

	m.mu.RLock()
	service, ok := m.cache[name]
	m.mu.RUnlock()
	if ok {
		m.mu.Lock()
		m.cacheHits++
		m.mu.Unlock()
		return service, true
	}

	m.mu.Lock()
	m.cacheMisses++
	m.mu.Unlock()
	return nil, false
}

func (m *serviceManager) storeService(name string, service api.Binder) {
	if m == nil || name == "" || service == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache == nil {
		m.cache = make(map[string]api.Binder)
	}
	m.cache[name] = service
}

type serviceManagerDebugSnapshot struct {
	CacheEntries int
	CacheHits    int
	CacheMisses  int
	Names        []string
}

func (m *serviceManager) debugSnapshot() serviceManagerDebugSnapshot {
	if m == nil {
		return serviceManagerDebugSnapshot{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	out := serviceManagerDebugSnapshot{
		CacheEntries: len(m.cache),
		CacheHits:    m.cacheHits,
		CacheMisses:  m.cacheMisses,
		Names:        make([]string, 0, len(m.cache)),
	}
	for name := range m.cache {
		out.Names = append(out.Names, name)
	}
	return out
}
