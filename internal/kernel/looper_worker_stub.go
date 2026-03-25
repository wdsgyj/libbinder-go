//go:build !((linux || android) && (amd64 || arm64))

package kernel

// LooperWorker represents a thread-bound Binder looper worker.
type LooperWorker struct {
	Name string

	State *ThreadState

	stop chan struct{}
	done chan struct{}
}

func NewLooperWorker(name string, backend *Backend) *LooperWorker {
	return &LooperWorker{
		Name:  name,
		State: &ThreadState{Role: "looper"},
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

func (w *LooperWorker) Start() error {
	close(w.done)
	return ErrUnsupportedPlatform
}

func (w *LooperWorker) Close() error {
	return nil
}
