package libbinder

import (
	"context"
	"errors"
	"sync"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

type rpcServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]api.Binder
	waiters  map[string][]chan struct{}
}

func newRPCServiceRegistry() *rpcServiceRegistry {
	return &rpcServiceRegistry{
		services: make(map[string]api.Binder),
		waiters:  make(map[string][]chan struct{}),
	}
}

func (r *rpcServiceRegistry) check(name string) api.Binder {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[name]
}

func (r *rpcServiceRegistry) add(name string, service api.Binder) {
	if r == nil || name == "" || service == nil {
		return
	}

	r.mu.Lock()
	r.services[name] = service
	waiters := r.waiters[name]
	delete(r.waiters, name)
	r.mu.Unlock()

	for _, waiter := range waiters {
		close(waiter)
	}
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
	if service := r.services[name]; service != nil {
		r.mu.Unlock()
		return service, nil
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
	if service, ok := m.cachedService(name); ok {
		return service, nil
	}
	service, err := m.conn.serviceRegistry.wait(ctx, name)
	if err != nil {
		return nil, err
	}
	m.storeService(name, service)
	return service, nil
}

func (m *rpcLocalServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	resolved := api.ResolveAddServiceOptions(opts...)
	service, err := m.conn.registerLocalHandler(handler, resolved.Serial)
	if err != nil {
		return err
	}
	m.conn.serviceRegistry.add(name, service)
	m.storeService(name, service)
	return nil
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

func (m *rpcRemoteServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if service, ok := m.cachedService(name); ok {
		return service, nil
	}

	ticker := time.NewTicker(waitServicePollInterval)
	defer ticker.Stop()

	for {
		service, err := m.CheckService(ctx, name)
		if err == nil {
			return service, nil
		}
		if !errors.Is(err, api.ErrNoService) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
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
	m.storeService(name, service)
	return nil
}

func (m *rpcRemoteServiceManager) call(ctx context.Context, code uint32, name string) (*api.Parcel, error) {
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

type rpcServiceManagerHandler struct {
	conn *RPCConn
}

func newRPCServiceManagerHandler(conn *RPCConn) api.Handler {
	return &rpcServiceManagerHandler{conn: conn}
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
		reply := api.NewParcel()
		if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
			return nil, err
		}
		service := h.conn.serviceRegistry.check(name)
		if service == nil {
			if err := reply.WriteNullStrongBinder(); err != nil {
				return nil, err
			}
		} else {
			if err := reply.WriteStrongBinder(service); err != nil {
				return nil, err
			}
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
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
		if _, err := data.ReadBool(); err != nil {
			return nil, err
		}
		if _, err := data.ReadInt32(); err != nil {
			return nil, err
		}

		h.conn.serviceRegistry.add(name, service)
		reply := api.NewParcel()
		if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
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
