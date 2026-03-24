//go:build android

package kernel

import "testing"

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
