package libbinder

import (
	"context"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func (c *RPCConn) watchDeath(ctx context.Context, handle uint32) (api.Subscription, error) {
	if c == nil || handle == 0 {
		return nil, api.ErrUnsupported
	}

	sub := newManagedSubscription(ctx, nil)
	sub.closeFn = func() error {
		c.deathMu.Lock()
		watches := c.deathWatches[handle]
		delete(watches, sub)
		if len(watches) == 0 {
			delete(c.deathWatches, handle)
		}
		c.deathMu.Unlock()
		return nil
	}

	select {
	case <-c.closed:
		sub.finish(api.ErrDeadObject)
		return sub, nil
	default:
	}

	c.deathMu.Lock()
	if c.deathWatches[handle] == nil {
		c.deathWatches[handle] = make(map[*managedSubscription]struct{})
	}
	c.deathWatches[handle][sub] = struct{}{}
	c.deathMu.Unlock()
	return sub, nil
}

func (c *RPCConn) noteRemoteDeath(handle uint32) {
	if c == nil || handle == 0 {
		return
	}

	c.exportsMu.Lock()
	remote := c.imports[handle]
	delete(c.imports, handle)
	c.exportsMu.Unlock()
	if remote != nil {
		remote.dead.Store(true)
	}

	c.deathMu.Lock()
	watches := c.deathWatches[handle]
	delete(c.deathWatches, handle)
	c.deathMu.Unlock()
	for sub := range watches {
		sub.finish(api.ErrDeadObject)
	}
}

func (c *RPCConn) noteAllRemoteDeaths() {
	if c == nil {
		return
	}

	c.exportsMu.Lock()
	imports := c.imports
	c.imports = make(map[uint32]*rpcRemoteBinder)
	c.exportsMu.Unlock()
	for _, remote := range imports {
		if remote != nil {
			remote.dead.Store(true)
		}
	}

	c.deathMu.Lock()
	watches := c.deathWatches
	c.deathWatches = make(map[uint32]map[*managedSubscription]struct{})
	c.deathMu.Unlock()
	for _, set := range watches {
		for sub := range set {
			sub.finish(api.ErrDeadObject)
		}
	}
}
