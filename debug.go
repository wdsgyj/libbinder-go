package libbinder

import (
	"github.com/wdsgyj/libbinder-go/internal/kernel"
	"github.com/wdsgyj/libbinder-go/internal/runtime"
)

type ServiceManagerSnapshot struct {
	CacheEntries int
	CacheHits    int
	CacheMisses  int
	Names        []string
}

type DebugSnapshot struct {
	Kernel         kernel.Snapshot
	Refs           runtime.RefSnapshot
	ServiceManager ServiceManagerSnapshot
}

func (c *Conn) DebugSnapshot() DebugSnapshot {
	if c == nil {
		return DebugSnapshot{}
	}

	out := DebugSnapshot{}
	if c.rt != nil {
		out.Refs = c.rt.Refs.Snapshot()
		if c.rt.Kernel != nil {
			out.Kernel = c.rt.Kernel.Snapshot()
		}
	}
	if c.sm != nil {
		sm := c.sm.debugSnapshot()
		out.ServiceManager = ServiceManagerSnapshot{
			CacheEntries: sm.CacheEntries,
			CacheHits:    sm.CacheHits,
			CacheMisses:  sm.CacheMisses,
			Names:        sm.Names,
		}
	}
	return out
}
