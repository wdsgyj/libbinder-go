package libbindergo

import (
	"context"
	"errors"
	"sync"

	api "libbinder-go/binder"
	"libbinder-go/internal/kernel"
	"libbinder-go/internal/runtime"
)

const contextManagerHandle uint32 = 0

// Config controls public Binder connection startup.
type Config struct {
	DriverPath    string
	LooperWorkers int
	ClientWorkers int
}

// Conn is the public entry point for talking to the kernel Binder driver.
type Conn struct {
	rt *runtime.Runtime
	sm *serviceManager

	acquiredMu sync.Mutex
	acquired   map[uint32]struct{}
}

// Open starts a Binder connection backed by the kernel Binder driver.
func Open(cfg Config) (*Conn, error) {
	rtCfg := runtime.Config{
		DriverPath: cfg.DriverPath,
	}
	if cfg.LooperWorkers > 0 || cfg.ClientWorkers > 0 {
		rtCfg.Kernel = kernel.StartOptions{
			LooperWorkers: cfg.LooperWorkers,
			ClientWorkers: cfg.ClientWorkers,
		}
	}

	rt := runtime.New(rtCfg)
	if err := rt.Start(rtCfg); err != nil {
		return nil, mapRuntimeError(err)
	}

	conn := &Conn{
		rt:       rt,
		acquired: make(map[uint32]struct{}),
	}
	conn.sm = &serviceManager{
		conn:   conn,
		target: &remoteBinder{conn: conn, handle: contextManagerHandle},
	}
	return conn, nil
}

// Close releases the Binder runtime resources owned by the connection.
func (c *Conn) Close() error {
	if c == nil || c.rt == nil {
		return nil
	}
	return mapRuntimeError(c.rt.Close())
}

// Handle returns a remote Binder proxy for the given kernel Binder handle.
func (c *Conn) Handle(handle uint32) api.Binder {
	return &remoteBinder{conn: c, handle: handle}
}

// ContextManager returns a Binder proxy for the process's Binder context manager.
func (c *Conn) ContextManager() api.Binder {
	return c.Handle(contextManagerHandle)
}

// ServiceManager returns the public ServiceManager facade for this connection.
func (c *Conn) ServiceManager() api.ServiceManager {
	return c.sm
}

func mapRuntimeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, kernel.ErrDeadReply):
		return api.ErrDeadObject
	case errors.Is(err, kernel.ErrFailedReply):
		return api.ErrFailedTxn
	default:
		return err
	}
}

func (c *Conn) ensureHandleAcquired(ctx context.Context, handle uint32) error {
	if c == nil || c.rt == nil || handle == 0 {
		return nil
	}

	c.acquiredMu.Lock()
	_, ok := c.acquired[handle]
	c.acquiredMu.Unlock()
	if ok {
		return nil
	}

	if err := mapRuntimeError(c.rt.AcquireHandle(ctx, handle)); err != nil {
		return err
	}

	c.acquiredMu.Lock()
	c.acquired[handle] = struct{}{}
	c.acquiredMu.Unlock()
	return nil
}

func (c *Conn) markHandleAcquired(handle uint32) {
	if c == nil || handle == 0 {
		return
	}
	c.acquiredMu.Lock()
	if c.acquired != nil {
		c.acquired[handle] = struct{}{}
	}
	c.acquiredMu.Unlock()
}

func (c *Conn) registerLocalNode(handler api.Handler, serial bool) (runtime.LocalNodeRef, error) {
	if c == nil || c.rt == nil {
		return runtime.LocalNodeRef{}, api.ErrUnsupported
	}
	return c.rt.RegisterLocalNode(handler, serial)
}

func (c *Conn) watchDeath(ctx context.Context, handle uint32) (api.Subscription, error) {
	if c == nil || c.rt == nil || c.rt.Kernel == nil {
		return nil, api.ErrUnsupported
	}
	if err := c.ensureHandleAcquired(ctx, handle); err != nil {
		return nil, err
	}
	sub, err := c.rt.Kernel.WatchDeath(ctx, handle)
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	return sub, nil
}
