package kernel

import "fmt"

// WorkerManager tracks the thread-bound workers used by the kernel Binder backend.
type WorkerManager struct {
	Driver *DriverManager

	Loopers []*LooperWorker
	Clients []*ClientWorker

	started bool
}

func NewWorkerManager(driver *DriverManager) *WorkerManager {
	return &WorkerManager{Driver: driver}
}

func (m *WorkerManager) Start(opts StartOptions) error {
	if m.started {
		return nil
	}

	if opts.LooperWorkers <= 0 {
		opts.LooperWorkers = 1
	}
	if opts.ClientWorkers <= 0 {
		opts.ClientWorkers = 1
	}

	m.Loopers = make([]*LooperWorker, 0, opts.LooperWorkers)
	for i := 0; i < opts.LooperWorkers; i++ {
		worker := NewLooperWorker(fmt.Sprintf("binder-looper-%d", i))
		if err := worker.Start(); err != nil {
			_ = m.Close()
			return err
		}
		m.Loopers = append(m.Loopers, worker)
	}

	m.Clients = make([]*ClientWorker, 0, opts.ClientWorkers)
	for i := 0; i < opts.ClientWorkers; i++ {
		worker := NewClientWorker(fmt.Sprintf("binder-client-%d", i), m.Driver)
		if err := worker.Start(); err != nil {
			_ = m.Close()
			return err
		}
		m.Clients = append(m.Clients, worker)
	}

	m.started = true
	return nil
}

func (m *WorkerManager) Close() error {
	for _, worker := range m.Loopers {
		_ = worker.Close()
	}
	for _, worker := range m.Clients {
		_ = worker.Close()
	}

	m.Loopers = nil
	m.Clients = nil
	m.started = false
	return nil
}

func (m *WorkerManager) Client() (*ClientWorker, error) {
	if len(m.Clients) == 0 {
		return nil, ErrNoClientWorker
	}
	return m.Clients[0], nil
}
