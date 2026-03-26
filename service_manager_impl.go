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
	txProfile   serviceManagerTransactions
	txKnown     bool
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

	reply, err := m.callName(ctx, m.transactions(ctx).checkService, name)
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

	reply, err := m.target.Transact(ctx, m.transactions(ctx).addService, data, api.FlagNone)
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
	tx := m.transactions(ctx)
	if tx.listServices == 0 {
		return nil, api.ErrUnsupported
	}
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return nil, err
	}
	if err := data.WriteInt32(int32(dumpFlags)); err != nil {
		return nil, err
	}

	reply, err := m.target.Transact(ctx, tx.listServices, data, api.FlagNone)
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

	tx := m.transactions(ctx)
	if tx.registerNotify == 0 {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, api.ErrUnsupported
	}
	reply, err := m.target.Transact(ctx, tx.registerNotify, data, api.FlagNone)
	if err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}
	if err := decodeStatusReply(reply); err != nil {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, err
	}

	sub.closeFn = func() error {
		err := m.callNotificationUnregister(context.Background(), tx.unregisterNotify, name, callbackBinder)
		cleanupErr := m.conn.unregisterLocalNode(node.ID)
		if err != nil {
			return err
		}
		return cleanupErr
	}
	return sub, nil
}

func (m *serviceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	reply, err := m.callName(ctx, m.transactions(ctx).isDeclared, name)
	if err != nil {
		return false, err
	}
	return reply.ReadBool()
}

func (m *serviceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	reply, err := m.callName(ctx, m.transactions(ctx).declaredInstances, iface)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *serviceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	reply, err := m.callName(ctx, m.transactions(ctx).updatableViaApex, name)
	if err != nil {
		return nil, err
	}
	return reply.ReadNullableString()
}

func (m *serviceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	reply, err := m.callName(ctx, m.transactions(ctx).updatableNames, apexName)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *serviceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	reply, err := m.callName(ctx, m.transactions(ctx).connectionInfo, name)
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

	tx := m.transactions(ctx)
	if tx.registerClientCallback == 0 {
		_ = m.conn.unregisterLocalNode(node.ID)
		return nil, api.ErrUnsupported
	}
	reply, err := m.target.Transact(ctx, tx.registerClientCallback, data, api.FlagNone)
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
	tx := m.transactions(ctx)
	if tx.tryUnregister == 0 {
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
	reply, err := m.target.Transact(ctx, tx.tryUnregister, data, api.FlagNone)
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
	tx := m.transactions(ctx)
	if tx.debugInfo == 0 {
		return nil, api.ErrUnsupported
	}
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		return nil, err
	}
	reply, err := m.target.Transact(ctx, tx.debugInfo, data, api.FlagNone)
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
	if code == 0 {
		return nil, api.ErrUnsupported
	}
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
	if code == 0 {
		return api.ErrUnsupported
	}
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

func (m *serviceManager) transactions(ctx context.Context) serviceManagerTransactions {
	if m == nil {
		return serviceManagerTransactionsCurrent
	}

	m.mu.RLock()
	if m.txKnown {
		tx := m.txProfile
		m.mu.RUnlock()
		return tx
	}
	m.mu.RUnlock()

	tx := m.detectTransactions(ctx)

	m.mu.Lock()
	if !m.txKnown {
		m.txProfile = tx
		m.txKnown = true
	}
	tx = m.txProfile
	m.mu.Unlock()
	return tx
}

func (m *serviceManager) detectTransactions(ctx context.Context) serviceManagerTransactions {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || m.target == nil {
		return serviceManagerTransactionsCurrent
	}
	if err := m.probeCheckService(ctx, serviceManagerTransactionsCurrent.checkService); err == nil {
		return serviceManagerTransactionsCurrent
	}
	if err := m.probeCheckService(ctx, serviceManagerTransactionsLegacy13.checkService); err == nil {
		if err := m.probeConnectionInfo(ctx, serviceManagerTransactionsLegacy13.connectionInfo); err == nil {
			return serviceManagerTransactionsLegacy13
		}
		return serviceManagerTransactionsLegacy12
	}
	return serviceManagerTransactionsCurrent
}

func (m *serviceManager) probeCheckService(ctx context.Context, code uint32) error {
	reply, err := m.callNameDirect(ctx, code, "github.com/wdsgyj/libbinder-go.__smprobe__")
	if err != nil {
		return err
	}
	_, err = reply.ReadStrongBinder()
	return err
}

func (m *serviceManager) probeConnectionInfo(ctx context.Context, code uint32) error {
	reply, err := m.callNameDirect(ctx, code, "github.com/wdsgyj/libbinder-go.__smprobe__")
	if err != nil {
		return err
	}
	_, err = readNullableConnectionInfoFromParcel(reply)
	return err
}

func (m *serviceManager) callNameDirect(ctx context.Context, code uint32, name string) (*api.Parcel, error) {
	if code == 0 {
		return nil, api.ErrUnsupported
	}
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
