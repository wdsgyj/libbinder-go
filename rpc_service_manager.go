package libbinder

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

type rpcServiceEntry struct {
	name       string
	service    api.Binder
	opts       api.AddServiceOptions
	hasClients bool
}

type rpcRegistrationWatch struct {
	id       uint64
	name     string
	callback api.ServiceRegistrationCallback
}

type rpcClientWatch struct {
	id         uint64
	name       string
	serviceKey binderIdentity
	callback   api.ServiceClientCallback
}

type rpcServiceRegistry struct {
	mu sync.RWMutex

	services map[string]*rpcServiceEntry
	waiters  map[string][]chan struct{}

	registrationWatchers map[string]map[uint64]*rpcRegistrationWatch
	clientWatchers       map[string]map[uint64]*rpcClientWatch
	nextWatchID          uint64
}

func newRPCServiceRegistry() *rpcServiceRegistry {
	return &rpcServiceRegistry{
		services:             make(map[string]*rpcServiceEntry),
		waiters:              make(map[string][]chan struct{}),
		registrationWatchers: make(map[string]map[uint64]*rpcRegistrationWatch),
		clientWatchers:       make(map[string]map[uint64]*rpcClientWatch),
	}
}

func (r *rpcServiceRegistry) check(name string) api.Binder {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.services[name]
	if entry == nil {
		return nil
	}
	return entry.service
}

func (r *rpcServiceRegistry) get(name string) (*rpcServiceEntry, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.services[name]
	if entry == nil {
		return nil, false
	}
	copy := *entry
	if entry.opts.Metadata.ConnectionInfo != nil {
		info := *entry.opts.Metadata.ConnectionInfo
		copy.opts.Metadata.ConnectionInfo = &info
	}
	return &copy, true
}

func (r *rpcServiceRegistry) add(name string, service api.Binder, opts api.AddServiceOptions) error {
	if r == nil || name == "" || service == nil {
		return nil
	}
	if api.BinderStability(service) == api.StabilityVINTF && !opts.Metadata.Declared {
		return fmt.Errorf("%w: vintf service %q must be declared", &api.StatusCodeError{Code: api.StatusBadType}, name)
	}
	if opts.Metadata.DebugPID == 0 {
		opts.Metadata.DebugPID = int32(os.Getpid())
	}

	entry := &rpcServiceEntry{
		name:    name,
		service: service,
		opts:    opts,
	}

	r.mu.Lock()
	r.services[name] = entry
	waiters := r.waiters[name]
	delete(r.waiters, name)
	watches := copyRegistrationWatches(r.registrationWatchers[name])
	r.mu.Unlock()

	for _, waiter := range waiters {
		close(waiter)
	}
	for _, watch := range watches {
		if watch != nil && watch.callback != nil {
			watch.callback(context.Background(), api.ServiceRegistration{Name: name, Binder: service})
		}
	}
	return nil
}

func (r *rpcServiceRegistry) wait(ctx context.Context, name string) (api.Binder, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if service := r.check(name); service != nil {
		return service, nil
	}

	waiter := make(chan struct{})
	r.mu.Lock()
	if entry := r.services[name]; entry != nil {
		r.mu.Unlock()
		return entry.service, nil
	}
	r.waiters[name] = append(r.waiters[name], waiter)
	r.mu.Unlock()

	select {
	case <-waiter:
		service := r.check(name)
		if service == nil {
			return nil, api.ErrNoService
		}
		return service, nil
	case <-ctx.Done():
		r.removeWaiter(name, waiter)
		return nil, ctx.Err()
	}
}

func (r *rpcServiceRegistry) removeWaiter(name string, waiter chan struct{}) {
	if r == nil || waiter == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	waiters := r.waiters[name]
	for i, current := range waiters {
		if current == waiter {
			waiters = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		delete(r.waiters, name)
		return
	}
	r.waiters[name] = waiters
}

func (r *rpcServiceRegistry) list(dumpFlags api.DumpFlags) []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	filter := dumpFlags & api.DumpPriorityAll
	for name, entry := range r.services {
		if entry == nil {
			continue
		}
		if filter != 0 {
			entryFlags := entry.opts.EffectiveDumpFlags() & api.DumpPriorityAll
			if entryFlags == 0 {
				entryFlags = api.DumpPriorityDefault
			}
			if entryFlags&filter == 0 {
				continue
			}
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (r *rpcServiceRegistry) watchRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) api.Subscription {
	r.mu.Lock()
	r.nextWatchID++
	id := r.nextWatchID
	watch := &rpcRegistrationWatch{id: id, name: name, callback: callback}
	if r.registrationWatchers[name] == nil {
		r.registrationWatchers[name] = make(map[uint64]*rpcRegistrationWatch)
	}
	r.registrationWatchers[name][id] = watch
	entry := r.services[name]
	r.mu.Unlock()

	sub := newManagedSubscription(ctx, func() error {
		r.mu.Lock()
		watches := r.registrationWatchers[name]
		delete(watches, id)
		if len(watches) == 0 {
			delete(r.registrationWatchers, name)
		}
		r.mu.Unlock()
		return nil
	})
	if entry != nil && callback != nil {
		callback(context.Background(), api.ServiceRegistration{Name: name, Binder: entry.service})
	}
	return sub
}

func (r *rpcServiceRegistry) watchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	key, ok := binderKey(service)
	if !ok {
		return nil, api.ErrUnsupported
	}

	r.mu.Lock()
	entry := r.services[name]
	if entry == nil {
		r.mu.Unlock()
		return nil, api.ErrNoService
	}
	entryKey, ok := binderKey(entry.service)
	if !ok || entryKey != key {
		r.mu.Unlock()
		return nil, api.ErrUnsupported
	}
	r.nextWatchID++
	id := r.nextWatchID
	watch := &rpcClientWatch{id: id, name: name, serviceKey: key, callback: callback}
	if r.clientWatchers[name] == nil {
		r.clientWatchers[name] = make(map[uint64]*rpcClientWatch)
	}
	r.clientWatchers[name][id] = watch
	hasClients := entry.hasClients
	r.mu.Unlock()

	sub := newManagedSubscription(ctx, func() error {
		r.mu.Lock()
		watches := r.clientWatchers[name]
		delete(watches, id)
		if len(watches) == 0 {
			delete(r.clientWatchers, name)
		}
		r.mu.Unlock()
		return nil
	})

	if hasClients && callback != nil {
		callback(context.Background(), api.ServiceClientUpdate{
			Name:       name,
			Service:    service,
			HasClients: true,
		})
	}
	return sub, nil
}

func (r *rpcServiceRegistry) isDeclared(name string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.services[name]
	if entry == nil {
		return false
	}
	return entry.opts.Metadata.Declared || entry.service != nil
}

func (r *rpcServiceRegistry) declaredInstances(iface string) []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0)
	prefix := iface + "/"
	for name := range r.services {
		switch {
		case len(name) >= len(prefix) && name[:len(prefix)] == prefix:
			out = append(out, name[len(prefix):])
		case name == iface:
			out = append(out, "")
		}
	}
	slices.Sort(out)
	return out
}

func (r *rpcServiceRegistry) updatableViaApex(name string) *string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.services[name]
	if entry == nil || entry.opts.Metadata.UpdatableViaApex == "" {
		return nil
	}
	value := entry.opts.Metadata.UpdatableViaApex
	return &value
}

func (r *rpcServiceRegistry) updatableNames(apexName string) []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for name, entry := range r.services {
		if entry != nil && entry.opts.Metadata.UpdatableViaApex == apexName {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

func (r *rpcServiceRegistry) connectionInfo(name string) *api.ConnectionInfo {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry := r.services[name]
	if entry == nil || entry.opts.Metadata.ConnectionInfo == nil {
		return nil
	}
	info := *entry.opts.Metadata.ConnectionInfo
	return &info
}

func (r *rpcServiceRegistry) tryUnregister(name string, service api.Binder) error {
	if r == nil {
		return api.ErrUnsupported
	}
	key, ok := binderKey(service)
	if !ok {
		return api.ErrUnsupported
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.services[name]
	if entry == nil {
		return api.ErrNoService
	}
	entryKey, ok := binderKey(entry.service)
	if !ok || entryKey != key {
		return api.ErrUnsupported
	}
	if entry.hasClients {
		return &api.StatusCodeError{Code: api.StatusInvalidOperation}
	}
	delete(r.services, name)
	return nil
}

func (r *rpcServiceRegistry) debugInfo() []api.ServiceDebugInfo {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]api.ServiceDebugInfo, 0, len(r.services))
	for name, entry := range r.services {
		if entry == nil {
			continue
		}
		out = append(out, api.ServiceDebugInfo{
			Name:     name,
			DebugPID: entry.opts.Metadata.DebugPID,
		})
	}
	slices.SortFunc(out, func(a, b api.ServiceDebugInfo) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})
	return out
}

func copyRegistrationWatches(src map[uint64]*rpcRegistrationWatch) []*rpcRegistrationWatch {
	if len(src) == 0 {
		return nil
	}
	out := make([]*rpcRegistrationWatch, 0, len(src))
	for _, watch := range src {
		out = append(out, watch)
	}
	return out
}

func copyClientWatches(src map[uint64]*rpcClientWatch) []*rpcClientWatch {
	if len(src) == 0 {
		return nil
	}
	out := make([]*rpcClientWatch, 0, len(src))
	for _, watch := range src {
		out = append(out, watch)
	}
	return out
}

func (r *rpcServiceRegistry) markClientActive(name string) {
	if r == nil || name == "" {
		return
	}

	r.mu.Lock()
	entry := r.services[name]
	if entry == nil || entry.hasClients {
		r.mu.Unlock()
		return
	}
	entry.hasClients = true
	service := entry.service
	watches := copyClientWatches(r.clientWatchers[name])
	r.mu.Unlock()

	for _, watch := range watches {
		if watch != nil && watch.callback != nil {
			watch.callback(context.Background(), api.ServiceClientUpdate{
				Name:       name,
				Service:    service,
				HasClients: true,
			})
		}
	}
}

func (r *rpcServiceRegistry) clearClientState() {
	if r == nil {
		return
	}

	type update struct {
		name    string
		service api.Binder
		watches []*rpcClientWatch
	}

	r.mu.Lock()
	var updates []update
	for name, entry := range r.services {
		if entry == nil || !entry.hasClients {
			continue
		}
		entry.hasClients = false
		updates = append(updates, update{
			name:    name,
			service: entry.service,
			watches: copyClientWatches(r.clientWatchers[name]),
		})
	}
	r.mu.Unlock()

	for _, update := range updates {
		for _, watch := range update.watches {
			if watch != nil && watch.callback != nil {
				watch.callback(context.Background(), api.ServiceClientUpdate{
					Name:       update.name,
					Service:    update.service,
					HasClients: false,
				})
			}
		}
	}
}

type rpcServiceManagerCache struct {
	mu          sync.RWMutex
	cache       map[string]api.Binder
	cacheHits   int
	cacheMisses int
}

func (c *rpcServiceManagerCache) cachedService(name string) (api.Binder, bool) {
	if c == nil || name == "" {
		return nil, false
	}

	c.mu.RLock()
	service, ok := c.cache[name]
	c.mu.RUnlock()
	if ok {
		c.mu.Lock()
		c.cacheHits++
		c.mu.Unlock()
		return service, true
	}

	c.mu.Lock()
	c.cacheMisses++
	c.mu.Unlock()
	return nil, false
}

func (c *rpcServiceManagerCache) storeService(name string, service api.Binder) {
	if c == nil || name == "" || service == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]api.Binder)
	}
	c.cache[name] = service
}

func (c *rpcServiceManagerCache) deleteService(name string) {
	if c == nil || name == "" {
		return
	}
	c.mu.Lock()
	delete(c.cache, name)
	c.mu.Unlock()
}

func (c *rpcServiceManagerCache) debugSnapshot() serviceManagerDebugSnapshot {
	if c == nil {
		return serviceManagerDebugSnapshot{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	out := serviceManagerDebugSnapshot{
		CacheEntries: len(c.cache),
		CacheHits:    c.cacheHits,
		CacheMisses:  c.cacheMisses,
		Names:        make([]string, 0, len(c.cache)),
	}
	for name := range c.cache {
		out.Names = append(out.Names, name)
	}
	return out
}

type rpcLocalServiceManager struct {
	conn *RPCConn
	rpcServiceManagerCache
}

func (m *rpcLocalServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if service, ok := m.cachedService(name); ok {
		return service, nil
	}
	service := m.conn.serviceRegistry.check(name)
	if service == nil {
		return nil, api.ErrNoService
	}
	m.storeService(name, service)
	return service, nil
}

func (m *rpcLocalServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return waitForServiceWithNotifications(ctx, m, name)
}

func (m *rpcLocalServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	resolved := api.ResolveAddServiceOptions(opts...)
	service, err := m.conn.registerLocalHandler(handler, resolved.Serial)
	if err != nil {
		return err
	}
	if err := m.conn.serviceRegistry.add(name, service, resolved); err != nil {
		return err
	}
	m.storeService(name, service)
	return nil
}

func (m *rpcLocalServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	return m.conn.serviceRegistry.list(dumpFlags), nil
}

func (m *rpcLocalServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	if callback == nil {
		return nil, api.ErrUnsupported
	}
	return m.conn.serviceRegistry.watchRegistrations(ctx, name, callback), nil
}

func (m *rpcLocalServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return m.conn.serviceRegistry.isDeclared(name), nil
}

func (m *rpcLocalServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return m.conn.serviceRegistry.declaredInstances(iface), nil
}

func (m *rpcLocalServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return m.conn.serviceRegistry.updatableViaApex(name), nil
}

func (m *rpcLocalServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return m.conn.serviceRegistry.updatableNames(apexName), nil
}

func (m *rpcLocalServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return m.conn.serviceRegistry.connectionInfo(name), nil
}

func (m *rpcLocalServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	if callback == nil {
		return nil, api.ErrUnsupported
	}
	return m.conn.serviceRegistry.watchClients(ctx, name, service, callback)
}

func (m *rpcLocalServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	m.deleteService(name)
	return m.conn.serviceRegistry.tryUnregister(name, service)
}

func (m *rpcLocalServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return m.conn.serviceRegistry.debugInfo(), nil
}

type rpcRemoteServiceManager struct {
	conn   *RPCConn
	target api.Binder
	rpcServiceManagerCache
}

func (m *rpcRemoteServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
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
	return service, nil
}

func (m *rpcRemoteServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return waitForServiceWithNotifications(ctx, m, name)
}

func (m *rpcRemoteServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	resolved := api.ResolveAddServiceOptions(opts...)
	service, err := m.conn.registerLocalHandler(handler, resolved.Serial)
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
	if err := data.WriteStrongBinder(service); err != nil {
		return err
	}
	if err := data.WriteBool(resolved.AllowIsolated); err != nil {
		return err
	}
	if err := data.WriteInt32(int32(resolved.EffectiveDumpFlags())); err != nil {
		return err
	}
	if err := writeRPCServiceMetadataToParcel(data, resolved.Metadata); err != nil {
		return err
	}

	reply, err := m.target.Transact(ctx, addServiceTransactionID, data, api.FlagNone)
	if err != nil {
		return err
	}
	if err := decodeStatusReply(reply); err != nil {
		return err
	}
	m.storeService(name, service)
	return nil
}

func (m *rpcRemoteServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
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

func (m *rpcRemoteServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	if callback == nil {
		return nil, api.ErrUnsupported
	}
	sub := newManagedSubscription(ctx, nil)
	handler := &serviceRegistrationCallbackHandler{sub: sub, callback: callback}
	callbackBinder, err := m.conn.registerLocalHandler(handler, false)
	if err != nil {
		return nil, err
	}
	local, ok := callbackBinder.(*rpcLocalBinder)
	if !ok {
		return nil, api.ErrUnsupported
	}

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := data.WriteString(name); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := data.WriteStrongBinder(callbackBinder); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	reply, err := m.target.Transact(ctx, registerNotifyTransactionID, data, api.FlagNone)
	if err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := decodeStatusReply(reply); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}

	sub.closeFn = func() error {
		err := m.callNotificationUnregister(context.Background(), unregisterNotifyTxID, name, callbackBinder)
		cleanupErr := m.conn.unregisterLocalHandle(local.handle)
		if err != nil {
			return err
		}
		return cleanupErr
	}
	return sub, nil
}

func (m *rpcRemoteServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	reply, err := m.callName(ctx, isDeclaredTransactionID, name)
	if err != nil {
		return false, err
	}
	return reply.ReadBool()
}

func (m *rpcRemoteServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	reply, err := m.callName(ctx, declaredInstancesTxID, iface)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *rpcRemoteServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	reply, err := m.callName(ctx, updatableViaApexTxID, name)
	if err != nil {
		return nil, err
	}
	return reply.ReadNullableString()
}

func (m *rpcRemoteServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	reply, err := m.callName(ctx, updatableNamesTxID, apexName)
	if err != nil {
		return nil, err
	}
	return readStringSliceFromParcel(reply)
}

func (m *rpcRemoteServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	reply, err := m.callName(ctx, connectionInfoTxID, name)
	if err != nil {
		return nil, err
	}
	return readNullableConnectionInfoFromParcel(reply)
}

func (m *rpcRemoteServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	if callback == nil || service == nil {
		return nil, api.ErrUnsupported
	}
	sub := newManagedSubscription(ctx, nil)
	handler := &serviceClientCallbackHandler{
		sub: sub,
		callback: func(cbCtx context.Context, update api.ServiceClientUpdate) {
			update.Name = name
			callback(cbCtx, update)
		},
	}
	callbackBinder, err := m.conn.registerLocalHandler(handler, false)
	if err != nil {
		return nil, err
	}
	local, ok := callbackBinder.(*rpcLocalBinder)
	if !ok {
		return nil, api.ErrUnsupported
	}

	data := api.NewParcel()
	if err := data.WriteInterfaceToken(serviceManagerDescriptor); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := data.WriteString(name); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := data.WriteStrongBinder(service); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := data.WriteStrongBinder(callbackBinder); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	reply, err := m.target.Transact(ctx, registerClientCallbackTxID, data, api.FlagNone)
	if err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}
	if err := decodeStatusReply(reply); err != nil {
		_ = m.conn.unregisterLocalHandle(local.handle)
		return nil, err
	}

	sub.closeFn = func() error {
		return m.conn.unregisterLocalHandle(local.handle)
	}
	return sub, nil
}

func (m *rpcRemoteServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
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
	m.deleteService(name)
	return nil
}

func (m *rpcRemoteServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
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

func (m *rpcRemoteServiceManager) callName(ctx context.Context, code uint32, name string) (*api.Parcel, error) {
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

func (m *rpcRemoteServiceManager) callNotificationUnregister(ctx context.Context, code uint32, name string, callbackBinder api.Binder) error {
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

func (m *rpcRemoteServiceManager) watchCachedService(name string, service api.Binder) {
	if m == nil || service == nil {
		return
	}
	sub, err := service.WatchDeath(context.Background())
	if err != nil || sub == nil {
		return
	}
	go func() {
		<-sub.Done()
		m.deleteService(name)
	}()
}

type rpcServiceManagerHandler struct {
	conn *RPCConn

	mu sync.Mutex

	registrationSubs map[string]map[binderIdentity]api.Subscription
	clientSubs       map[binderIdentity]api.Subscription
}

func newRPCServiceManagerHandler(conn *RPCConn) *rpcServiceManagerHandler {
	return &rpcServiceManagerHandler{
		conn:             conn,
		registrationSubs: make(map[string]map[binderIdentity]api.Subscription),
		clientSubs:       make(map[binderIdentity]api.Subscription),
	}
}

func (h *rpcServiceManagerHandler) Descriptor() string {
	return serviceManagerDescriptor
}

func (h *rpcServiceManagerHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	descriptor, err := data.ReadInterfaceToken()
	if err != nil {
		return nil, err
	}
	if descriptor != serviceManagerDescriptor {
		return nil, api.ErrBadParcelable
	}

	switch code {
	case checkServiceTransactionID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		return h.replyWithService(name)
	case addServiceTransactionID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		service, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, api.ErrBadParcelable
		}
		allowIsolated, err := data.ReadBool()
		if err != nil {
			return nil, err
		}
		dumpFlags, err := data.ReadInt32()
		if err != nil {
			return nil, err
		}
		metadata, err := readRPCServiceMetadataFromParcel(data)
		if err != nil {
			return nil, err
		}
		opts := api.AddServiceOptions{
			AllowIsolated: allowIsolated,
			DumpFlags:     api.DumpFlags(uint32(dumpFlags) &^ uint32(api.DumpFlagLazyService)),
			Lazy:          api.DumpFlags(dumpFlags)&api.DumpFlagLazyService != 0,
			Metadata:      metadata,
		}
		if err := h.conn.serviceRegistry.add(name, service, opts); err != nil {
			return nil, err
		}
		return okReply()
	case listServicesTransactionID:
		dumpFlags, err := data.ReadInt32()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := writeStringSliceToParcel(reply, h.conn.serviceRegistry.list(api.DumpFlags(dumpFlags))); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case registerNotifyTransactionID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		callbackBinder, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		if callbackBinder == nil {
			return nil, api.ErrBadParcelable
		}
		if err := h.registerNotificationCallback(name, callbackBinder); err != nil {
			return nil, err
		}
		return okReply()
	case unregisterNotifyTxID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		callbackBinder, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		if callbackBinder == nil {
			return nil, api.ErrBadParcelable
		}
		h.unregisterNotificationCallback(name, callbackBinder)
		return okReply()
	case isDeclaredTransactionID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := reply.WriteBool(h.conn.serviceRegistry.isDeclared(name)); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case declaredInstancesTxID:
		iface, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := writeStringSliceToParcel(reply, h.conn.serviceRegistry.declaredInstances(iface)); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case updatableViaApexTxID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if value := h.conn.serviceRegistry.updatableViaApex(name); value != nil {
			if err := reply.WriteString(*value); err != nil {
				return nil, err
			}
		} else {
			if err := reply.WriteNullableString(nil); err != nil {
				return nil, err
			}
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case updatableNamesTxID:
		apexName, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := writeStringSliceToParcel(reply, h.conn.serviceRegistry.updatableNames(apexName)); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case connectionInfoTxID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := writeNullableConnectionInfoToParcel(reply, h.conn.serviceRegistry.connectionInfo(name)); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case registerClientCallbackTxID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		service, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		callbackBinder, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		if service == nil || callbackBinder == nil {
			return nil, api.ErrBadParcelable
		}
		if err := h.registerClientCallback(name, service, callbackBinder); err != nil {
			return nil, err
		}
		return okReply()
	case tryUnregisterServiceTxID:
		name, err := data.ReadString()
		if err != nil {
			return nil, err
		}
		service, err := data.ReadStrongBinder()
		if err != nil {
			return nil, err
		}
		if err := h.conn.serviceRegistry.tryUnregister(name, service); err != nil {
			return nil, err
		}
		return okReply()
	case debugInfoTransactionID:
		reply, err := okReply()
		if err != nil {
			return nil, err
		}
		if err := writeServiceDebugInfoSliceToParcel(reply, h.conn.serviceRegistry.debugInfo()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	default:
		return nil, api.ErrUnknownTransaction
	}
}

func (h *rpcServiceManagerHandler) replyWithService(name string) (*api.Parcel, error) {
	reply, err := okReply()
	if err != nil {
		return nil, err
	}
	service := h.conn.serviceRegistry.check(name)
	if service == nil {
		if err := reply.WriteNullStrongBinder(); err != nil {
			return nil, err
		}
	} else {
		h.conn.serviceRegistry.markClientActive(name)
		if err := reply.WriteStrongBinder(service); err != nil {
			return nil, err
		}
	}
	if err := reply.SetPosition(0); err != nil {
		return nil, err
	}
	return reply, nil
}

func (h *rpcServiceManagerHandler) registerNotificationCallback(name string, callbackBinder api.Binder) error {
	key, ok := binderKey(callbackBinder)
	if !ok {
		return api.ErrUnsupported
	}

	sub := h.conn.serviceRegistry.watchRegistrations(context.Background(), name, func(ctx context.Context, reg api.ServiceRegistration) {
		data := api.NewParcel()
		if err := data.WriteInterfaceToken(serviceCallbackDescriptor); err != nil {
			return
		}
		if err := data.WriteString(reg.Name); err != nil {
			return
		}
		if err := data.WriteStrongBinder(reg.Binder); err != nil {
			return
		}
		_, _ = callbackBinder.Transact(ctx, api.FirstCallTransaction, data, api.FlagOneway)
	})

	h.mu.Lock()
	if h.registrationSubs[name] == nil {
		h.registrationSubs[name] = make(map[binderIdentity]api.Subscription)
	}
	if old := h.registrationSubs[name][key]; old != nil {
		_ = old.Close()
	}
	h.registrationSubs[name][key] = sub
	h.mu.Unlock()
	return nil
}

func (h *rpcServiceManagerHandler) unregisterNotificationCallback(name string, callbackBinder api.Binder) {
	key, ok := binderKey(callbackBinder)
	if !ok {
		return
	}
	h.mu.Lock()
	subs := h.registrationSubs[name]
	sub := subs[key]
	delete(subs, key)
	if len(subs) == 0 {
		delete(h.registrationSubs, name)
	}
	h.mu.Unlock()
	if sub != nil {
		_ = sub.Close()
	}
}

func (h *rpcServiceManagerHandler) registerClientCallback(name string, service api.Binder, callbackBinder api.Binder) error {
	key, ok := binderKey(callbackBinder)
	if !ok {
		return api.ErrUnsupported
	}

	sub, err := h.conn.serviceRegistry.watchClients(context.Background(), name, service, func(ctx context.Context, update api.ServiceClientUpdate) {
		data := api.NewParcel()
		if err := data.WriteInterfaceToken(clientCallbackDescriptor); err != nil {
			return
		}
		target := update.Service
		if target == nil {
			target = service
		}
		if err := data.WriteStrongBinder(target); err != nil {
			return
		}
		if err := data.WriteBool(update.HasClients); err != nil {
			return
		}
		_, _ = callbackBinder.Transact(ctx, api.FirstCallTransaction, data, api.FlagOneway)
	})
	if err != nil {
		return err
	}

	h.mu.Lock()
	if old := h.clientSubs[key]; old != nil {
		_ = old.Close()
	}
	h.clientSubs[key] = sub
	h.mu.Unlock()
	return nil
}

func (h *rpcServiceManagerHandler) closeAllRemoteSubscriptions() {
	h.mu.Lock()
	registrationSubs := h.registrationSubs
	clientSubs := h.clientSubs
	h.registrationSubs = make(map[string]map[binderIdentity]api.Subscription)
	h.clientSubs = make(map[binderIdentity]api.Subscription)
	h.mu.Unlock()

	for _, subs := range registrationSubs {
		for _, sub := range subs {
			if sub != nil {
				_ = sub.Close()
			}
		}
	}
	for _, sub := range clientSubs {
		if sub != nil {
			_ = sub.Close()
		}
	}
}

func okReply() (*api.Parcel, error) {
	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		return nil, err
	}
	return reply, nil
}

func writeRPCServiceMetadataToParcel(p *api.Parcel, metadata api.ServiceMetadata) error {
	if err := p.WriteBool(metadata.Declared); err != nil {
		return err
	}
	if err := p.WriteNullableString(nullableString(metadata.UpdatableViaApex)); err != nil {
		return err
	}
	if err := writeNullableConnectionInfoToParcel(p, metadata.ConnectionInfo); err != nil {
		return err
	}
	return p.WriteInt32(metadata.DebugPID)
}

func readRPCServiceMetadataFromParcel(p *api.Parcel) (api.ServiceMetadata, error) {
	if p == nil || p.Remaining() == 0 {
		return api.ServiceMetadata{}, nil
	}

	declared, err := p.ReadBool()
	if err != nil {
		return api.ServiceMetadata{}, err
	}
	updatable, err := p.ReadNullableString()
	if err != nil {
		return api.ServiceMetadata{}, err
	}
	connectionInfo, err := readNullableConnectionInfoFromParcel(p)
	if err != nil {
		return api.ServiceMetadata{}, err
	}
	debugPID, err := p.ReadInt32()
	if err != nil {
		return api.ServiceMetadata{}, err
	}

	metadata := api.ServiceMetadata{
		Declared:       declared,
		ConnectionInfo: connectionInfo,
		DebugPID:       debugPID,
	}
	if updatable != nil {
		metadata.UpdatableViaApex = *updatable
	}
	return metadata, nil
}

func nullableString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
