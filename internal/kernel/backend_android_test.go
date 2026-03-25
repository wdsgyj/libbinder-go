//go:build android

package kernel

import (
	"context"
	"sync"
	"syscall"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestDriverManagerOpenCloseOnAndroid(t *testing.T) {
	driver := NewDriverManager(DefaultDriverPath)

	if err := driver.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !driver.IsOpen() {
		t.Fatal("driver should report open after Open")
	}
	if len(driver.Mmap()) == 0 {
		t.Fatal("driver mmap should not be empty after Open")
	}

	if err := driver.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if driver.IsOpen() {
		t.Fatal("driver should report closed after Close")
	}
}

func TestBackendStartCloseOnAndroid(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	if err := backend.Start(DefaultStartOptions()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !backend.Process.Started {
		t.Fatal("process state should be marked started")
	}
	if got := len(backend.Workers.Loopers); got != 1 {
		t.Fatalf("Looper worker count = %d, want 1", got)
	}
	if got := len(backend.Workers.Clients); got != 1 {
		t.Fatalf("Client worker count = %d, want 1", got)
	}

	if err := backend.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if backend.Process.Started {
		t.Fatal("process state should be marked stopped after Close")
	}
}

func TestClientWorkerRunsOnSingleOSThreadOnAndroid(t *testing.T) {
	driver := NewDriverManager(DefaultDriverPath)
	if err := driver.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	worker := NewClientWorker("test-client-worker", driver)
	if err := worker.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if err := worker.Close(); err != nil {
			t.Fatalf("worker.Close: %v", err)
		}
	}()

	const runs = 8
	var wg sync.WaitGroup
	tids := make(chan int, runs)
	errs := make(chan error, runs)

	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := worker.Do(context.Background(), func(_ *ThreadState) error {
				tids <- syscall.Gettid()
				return nil
			})
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(tids)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("worker.Do: %v", err)
		}
	}

	var first int
	for tid := range tids {
		if first == 0 {
			first = tid
			continue
		}
		if tid != first {
			t.Fatalf("worker task tid = %d, want %d", tid, first)
		}
	}
}

func TestBackendConcurrentPingContextManagerOnAndroid(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)
	if err := backend.Start(StartOptions{LooperWorkers: 1, ClientWorkers: 1}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	const callers = 16
	var wg sync.WaitGroup
	errs := make(chan error, callers)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reply, err := backend.TransactHandle(context.Background(), 0, PingTransaction, api.NewParcel(), api.FlagNone)
			if err != nil {
				errs <- err
				return
			}
			if reply == nil {
				errs <- context.DeadlineExceeded
				return
			}
			if reply.Len() != 0 {
				errs <- api.ErrBadParcelable
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ping: %v", err)
		}
	}
}
