package libbinder

import (
	"context"
	"errors"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
	"github.com/wdsgyj/libbinder-go/internal/runtime"
)

const contextManagerHandle uint32 = 0

// Config controls public Binder connection startup.
type Config struct {
	DriverPath        string
	LooperWorkers     int
	ClientWorkers     int
	RequiredStability api.StabilityLevel
}

// Conn is the public entry point for talking to the kernel Binder driver.
type Conn struct {
	rt                *runtime.Runtime
	sm                *serviceManager
	requiredStability api.StabilityLevel
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
		rt:                rt,
		requiredStability: cfg.RequiredStability,
	}
	if conn.requiredStability == api.StabilityUndeclared {
		conn.requiredStability = api.DefaultLocalStability()
	}
	conn.rt.Kernel.SetParcelResolvers(conn.resolveBinderHandle, conn.resolveLocalBinder)
	conn.rt.Kernel.SetParcelObjectResolvers(conn.resolveBinderObject, conn.resolveLocalBinderObject)
	conn.sm = &serviceManager{
		conn:   conn,
		target: newRemoteBinder(conn, contextManagerHandle),
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
	return newRemoteBinder(c, handle)
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
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		needAcquire, wait := c.rt.Refs.BeginAcquire(handle)
		if wait != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-wait:
				continue
			}
		}
		if !needAcquire {
			return nil
		}

		err := mapRuntimeError(c.rt.AcquireHandle(ctx, handle))
		shouldRelease := c.rt.Refs.FinishAcquire(handle, err == nil)
		if err != nil {
			return err
		}
		if shouldRelease {
			return c.releaseKernelHandle(handle)
		}
		return nil
	}
}

func (c *Conn) markHandleAcquired(handle uint32) {
	if c == nil || c.rt == nil || handle == 0 {
		return
	}
	c.rt.Refs.MarkAcquired(handle)
}

func (c *Conn) retainBinder(handle uint32) {
	if c == nil || c.rt == nil || handle == 0 {
		return
	}
	c.rt.Refs.RetainBinder(handle)
}

func (c *Conn) retainWatch(handle uint32) {
	if c == nil || c.rt == nil || handle == 0 {
		return
	}
	c.rt.Refs.RetainWatch(handle)
}

func (c *Conn) releaseBinder(handle uint32) error {
	if c == nil || c.rt == nil || handle == 0 {
		return nil
	}
	if !c.rt.Refs.ReleaseBinder(handle) {
		return nil
	}
	return c.releaseKernelHandle(handle)
}

func (c *Conn) releaseWatch(handle uint32) error {
	if c == nil || c.rt == nil || handle == 0 {
		return nil
	}
	if !c.rt.Refs.ReleaseWatch(handle) {
		return nil
	}
	return c.releaseKernelHandle(handle)
}

func (c *Conn) releaseKernelHandle(handle uint32) error {
	if c == nil || c.rt == nil || handle == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := mapRuntimeError(c.rt.ReleaseHandle(ctx, handle))
	if errors.Is(err, kernel.ErrDriverClosed) {
		return nil
	}
	return err
}

func (c *Conn) registerLocalNode(handler api.Handler, serial bool) (runtime.LocalNodeRef, error) {
	if c == nil || c.rt == nil {
		return runtime.LocalNodeRef{}, api.ErrUnsupported
	}
	return c.rt.RegisterLocalNode(handler, serial)
}

func (c *Conn) unregisterLocalNode(id uintptr) error {
	if c == nil || c.rt == nil || id == 0 {
		return nil
	}
	c.rt.UnregisterLocalNode(id)
	return nil
}

func (c *Conn) registerLocalHandler(handler api.Handler) (api.Binder, error) {
	node, err := c.registerLocalNode(handler, false)
	if err != nil {
		return nil, err
	}
	return newLocalBinder(c, node.ID), nil
}

func (c *Conn) resolveBinderHandle(handle uint32) api.Binder {
	if c == nil {
		return nil
	}
	c.markHandleAcquired(handle)
	return newRemoteBinder(c, handle)
}

func (c *Conn) resolveBinderObject(obj api.ParcelObject) api.Binder {
	if c == nil {
		return nil
	}
	c.markHandleAcquired(obj.Handle)
	return newRemoteBinderWithStability(c, obj.Handle, obj.Stability)
}

func (c *Conn) resolveLocalBinder(nodeID uintptr) api.Binder {
	if c == nil || nodeID == 0 {
		return nil
	}
	return newLocalBinder(c, nodeID)
}

func (c *Conn) resolveLocalBinderObject(obj api.ParcelObject, nodeID uintptr) api.Binder {
	if c == nil || nodeID == 0 {
		return nil
	}
	level := obj.Stability
	if level == api.StabilityUndeclared && c.rt != nil && c.rt.Kernel != nil {
		if node, ok := c.rt.Kernel.LocalNode(nodeID); ok {
			level = node.Stability
		}
	}
	if level == api.StabilityUndeclared {
		level = api.DefaultLocalStability()
	}
	return newLocalBinderWithStability(c, nodeID, level)
}

func (c *Conn) watchDeath(ctx context.Context, handle uint32) (api.Subscription, error) {
	if c == nil || c.rt == nil || c.rt.Kernel == nil {
		return nil, api.ErrUnsupported
	}
	if err := c.ensureHandleAcquired(ctx, handle); err != nil {
		return nil, err
	}
	c.retainWatch(handle)
	sub, err := c.rt.Kernel.WatchDeath(ctx, handle)
	if err != nil {
		_ = c.releaseWatch(handle)
		return nil, mapRuntimeError(err)
	}
	return newTrackedSubscription(sub, func() error {
		return c.releaseWatch(handle)
	}), nil
}
