package kernel

import api "github.com/wdsgyj/libbinder-go/binder"

type LocalNodeSnapshot struct {
	ID         uintptr
	Descriptor string
	Serial     bool
	Stability  api.StabilityLevel
}

type WorkerSnapshot struct {
	Started        bool
	LooperWorkers  int
	ClientWorkers  int
	BoundLoopers   int
	BoundClients   int
	LooperFailures int
	ClientFailures int
}

type ProcessSnapshot struct {
	DriverPath    string
	Started       bool
	LooperWorkers int
	ClientWorkers int
}

type DeathSnapshot struct {
	WatchCount        int
	SubscriptionCount int
}

type Snapshot struct {
	DriverPath     string
	DriverOpen     bool
	MmapSize       int
	Process        ProcessSnapshot
	Workers        WorkerSnapshot
	LocalNodeCount int
	LocalNodes     []LocalNodeSnapshot
	Deaths         DeathSnapshot
}

func (b *Backend) Snapshot() Snapshot {
	if b == nil {
		return Snapshot{}
	}

	out := Snapshot{}
	if b.Driver != nil {
		out.DriverPath = b.Driver.Path
		out.DriverOpen = b.Driver.IsOpen()
		out.MmapSize = len(b.Driver.Mmap())
	}
	if b.Process != nil {
		out.Process = ProcessSnapshot{
			DriverPath:    b.Process.DriverPath,
			Started:       b.Process.Started,
			LooperWorkers: b.Process.LooperWorkers,
			ClientWorkers: b.Process.ClientWorkers,
		}
	}
	if b.Workers != nil {
		out.Workers = b.Workers.Snapshot()
	}
	if b.locals != nil {
		out.LocalNodes = b.locals.snapshot()
		out.LocalNodeCount = len(out.LocalNodes)
	}
	if b.deaths != nil {
		out.Deaths = b.deaths.snapshot()
	}
	return out
}

func (m *WorkerManager) Snapshot() WorkerSnapshot {
	if m == nil {
		return WorkerSnapshot{}
	}

	out := WorkerSnapshot{
		Started:       m.started,
		LooperWorkers: len(m.Loopers),
		ClientWorkers: len(m.Clients),
	}
	for _, worker := range m.Loopers {
		if worker != nil && worker.State != nil && worker.State.Bound {
			out.BoundLoopers++
		}
		if worker != nil && worker.State != nil && worker.State.LastErr != nil {
			out.LooperFailures++
		}
	}
	for _, worker := range m.Clients {
		if worker != nil && worker.State != nil && worker.State.Bound {
			out.BoundClients++
		}
		if worker != nil && worker.State != nil && worker.State.LastErr != nil {
			out.ClientFailures++
		}
	}
	return out
}

func (r *localRegistry) snapshot() []LocalNodeSnapshot {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]LocalNodeSnapshot, 0, len(r.nodes))
	for _, node := range r.nodes {
		if node == nil {
			continue
		}
		out = append(out, LocalNodeSnapshot{
			ID:         node.ID,
			Descriptor: node.Handler.Descriptor(),
			Serial:     node.Serial,
			Stability:  node.Stability,
		})
	}
	return out
}

func (r *deathRegistry) snapshot() DeathSnapshot {
	if r == nil {
		return DeathSnapshot{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := DeathSnapshot{
		WatchCount: len(r.byHandle),
	}
	for _, watch := range r.byHandle {
		if watch == nil {
			continue
		}
		out.SubscriptionCount += len(watch.subs) + len(watch.closingSubs)
	}
	return out
}
