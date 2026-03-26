package libbinder

import (
	"context"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
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

	reply, err := m.callName(ctx, checkServiceTransactionID, name)
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
	m.watchCachedService(name, service)
	return service, nil
}

func (m *serviceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return waitForServiceWithNotifications(ctx, m, name)
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
	if err := data.WriteInt32(int32(resolved.EffectiveDumpFlags())); err != nil {
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

func (m *serviceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return nil, err
	}
	if err := data.WriteInt32(int32(dumpFlags)); err != nil {
		return nil, err
	}

	reply, err := m.target.Transact(ctx, listServicesTransactionID, data, api.FlagNone)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, api.ErrBadParcelable
	}
	if err := decodeStatusReply(reply); err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *serviceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	if callback == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sub := newManagedSubscription(ctx, nil)
	handler := &serviceRegistrationCallbackHandler{
		sub:      sub,
		callback: callback,
	}
	node, err := m.conn.registerLocalNode(handler, false)
	if err != nil {
		return nil, err
	}
	callbackBinder := newLocalBinderWithStability(m.conn, node.ID, api.HandlerStability(handler))

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := data.WriteString(name); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := data.WriteStrongBinder(callbackBinder); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}

	reply, err := m.target.Transact(ctx, registerNotifyTransactionID, data, api.FlagNone)
	if err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := decodeStatusReply(reply); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}

	sub.closeFn = func() error {
		err := m.callNotificationUnregister(context.Background(), unregisterNotifyTxID, name, callbackBinder)
		cleanupErr := m.conn.unregisterLocalNode(node.ID)
		if err != nil {
			return err
		}
		return cleanupErr
	}
	return sub, nil
}

func (m *serviceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	reply, err := m.callName(ctx, isDeclaredTransactionID, name)
	if err != nil {
		return false, err
	}
	return reply.ReadBool()
}

func (m *serviceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	reply, err := m.callName(ctx, declaredInstancesTxID, iface)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *serviceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	reply, err := m.callName(ctx, updatableViaApexTxID, name)
	if err != nil {
		return nil, err
	}
	return reply.ReadNullableString()
}

func (m *serviceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	reply, err := m.callName(ctx, updatableNamesTxID, apexName)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *serviceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	reply, err := m.callName(ctx, connectionInfoTxID, name)
	if err != nil {
		return nil, err
	}
	return readNullableConnectionInfoFromParcel(reply)
}

func (m *serviceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	if callback == nil || service == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sub := newManagedSubscription(ctx, nil)
	handler := &serviceClientCallbackHandler{
		sub: sub,
		callback: func(cbCtx context.Context, update api.ServiceClientUpdate) {
			update.Name = name
			callback(cbCtx, update)
		},
	}
	node, err := m.conn.registerLocalNode(handler, false)
	if err != nil {
		return nil, err
	}
	callbackBinder := newLocalBinderWithStability(m.conn, node.ID, api.HandlerStability(handler))

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := data.WriteString(name); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := data.WriteStrongBinder(service); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := data.WriteStrongBinder(callbackBinder); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}

	reply, err := m.target.Transact(ctx, registerClientCallbackTxID, data, api.FlagNone)
	if err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := decodeStatusReply(reply); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}

	sub.closeFn = func() error {
		return m.conn.unregisterLocalNode(node.ID)
	}
	return sub, nil
}

func (m *serviceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	if service == nil {
		return api.ErrUnsupported
	}
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return err
	}
	if err := data.WriteString(name); err != nil {
		return err
	}
	if err := data.WriteStrongBinder(service); err != nil {
		return err
	}
	reply, err := m.target.Transact(ctx, tryUnregisterServiceTxID, data, api.FlagNone)
	if err != nil {
		return err
	}
	if err := decodeStatusReply(reply); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.cache, name)
	m.mu.Unlock()
	return nil
}

func (m *serviceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return nil, err
	}
	reply, err := m.target.Transact(ctx, debugInfoTransactionID, data, api.FlagNone)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, api.ErrBadParcelable
	}
	if err := decodeStatusReply(reply); err != nil {
		return nil, err
	}
	return readServiceDebugInfoSliceFromParcel(reply)
}

func (m *serviceManager) callName(ctx context.Context, code uint32, name string) (*api.Parcel, error) {
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

func (m *serviceManager) callNotificationUnregister(ctx context.Context, code uint32, name string, callbackBinder api.Binder) error {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return err
	}
	if err := data.WriteString(name); err != nil {
		return err
	}
	if err := data.WriteStrongBinder(callbackBinder); err != nil {
		return err
	}
	reply, err := m.target.Transact(ctx, code, data, api.FlagNone)
	if err != nil {
		return err
	}
	return decodeStatusReply(reply)
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

func (m *serviceManager) watchCachedService(name string, service api.Binder) {
	if m == nil || service == nil {
		return
	}
	sub, err := service.WatchDeath(context.Background())
	if err != nil || sub == nil {
		return
	}
	go func() {
		<-sub.Done()
		m.mu.Lock()
		if m.cache[name] == service {
			delete(m.cache, name)
		}
		m.mu.Unlock()
	}()
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
