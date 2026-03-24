package kernel

import (
	"context"
	"runtime"
	"sync"
)

type clientTask struct {
	run  func(*ThreadState) error
	done chan error
}

// ClientWorker represents a thread-bound Binder transaction worker.
type ClientWorker struct {
	Name string

	State  *ThreadState
	Driver *DriverManager

	tasks chan clientTask
	stop  chan struct{}
	done  chan struct{}

	closeOnce sync.Once
}

func NewClientWorker(name string, driver *DriverManager) *ClientWorker {
	return &ClientWorker{
		Name:   name,
		State:  &ThreadState{Role: "client"},
		Driver: driver,
		tasks:  make(chan clientTask),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (w *ClientWorker) Start() error {
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(w.done)

		if w.Driver != nil {
			if err := w.Driver.EnterLooper(); err != nil {
				w.State.LastErr = err
				ready <- err
				return
			}
		}

		w.State.Bound = true
		ready <- nil

		for {
			select {
			case task := <-w.tasks:
				err := task.run(w.State)
				w.State.LastErr = err
				task.done <- err
			case <-w.stop:
				return
			}
		}
	}()

	return <-ready
}

func (w *ClientWorker) Do(ctx context.Context, run func(*ThreadState) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	task := clientTask{
		run:  run,
		done: make(chan error, 1),
	}

	select {
	case <-w.done:
		return ErrNoClientWorker
	case <-ctx.Done():
		return ctx.Err()
	case w.tasks <- task:
	}

	select {
	case err := <-task.done:
		return err
	case <-w.done:
		return ErrNoClientWorker
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *ClientWorker) Close() error {
	w.closeOnce.Do(func() {
		close(w.stop)
	})
	<-w.done
	return nil
}
