package kernel

import (
	"runtime"
)

// LooperWorker represents a thread-bound Binder looper worker.
type LooperWorker struct {
	Name string

	State *ThreadState

	stop chan struct{}
	done chan struct{}
}

func NewLooperWorker(name string) *LooperWorker {
	return &LooperWorker{
		Name:  name,
		State: &ThreadState{Role: "looper"},
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

func (w *LooperWorker) Start() error {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(w.done)

		w.State.Bound = true
		<-w.stop
	}()
	return nil
}

func (w *LooperWorker) Close() error {
	select {
	case <-w.done:
		return nil
	default:
		close(w.stop)
		<-w.done
		return nil
	}
}
