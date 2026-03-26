package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

type fakeOpenedConn struct {
	sm     api.ServiceManager
	closed bool
}

func (c *fakeOpenedConn) ServiceManager() api.ServiceManager {
	return c.sm
}

func (c *fakeOpenedConn) Close() error {
	c.closed = true
	return nil
}

func TestMainOpenFailure(t *testing.T) {
	oldOpen := openConn
	t.Cleanup(func() { openConn = oldOpen })

	openConn = func(cfg libbinder.Config) (openedConn, error) {
		return nil, errors.New("boom")
	}

	var stdout, stderr bytes.Buffer
	code := Main(context.Background(), []string{"tap", "1", "2"}, &stdout, &stderr)
	if code != unavailableExitCode {
		t.Fatalf("Main code = %d, want %d", code, unavailableExitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got != "input: Unable to get default service manager!\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestDefaultOpenConnCanBeCalled(t *testing.T) {
	defaultOpen := openConn
	conn, err := defaultOpen(libbinder.Config{})
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil && conn == nil {
		t.Fatal("defaultOpen returned nil conn and nil err")
	}
}

func TestMainUsesOpenedConn(t *testing.T) {
	oldOpen := openConn
	t.Cleanup(func() { openConn = oldOpen })

	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if got, want := req.Args[0], "tap"; got != want {
				t.Fatalf("req.Args[0] = %q, want %q", got, want)
			}
			return nil
		},
	}
	conn := &fakeOpenedConn{
		sm: fakeServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) {
				return service, nil
			},
		},
	}
	openConn = func(cfg libbinder.Config) (openedConn, error) {
		return conn, nil
	}

	code := Main(context.Background(), []string{"tap", "1", "2"}, io.Discard, io.Discard)
	if code != 0 {
		t.Fatalf("Main code = %d, want 0", code)
	}
	if !conn.closed {
		t.Fatal("Main did not close conn")
	}
}

func TestMainFunction(t *testing.T) {
	oldArgs := os.Args
	oldOpen := openConn
	oldSignalIgnore := signalIgnore
	oldProcessExit := processExit
	t.Cleanup(func() {
		os.Args = oldArgs
		openConn = oldOpen
		signalIgnore = oldSignalIgnore
		processExit = oldProcessExit
	})

	service := &fakeShellService{
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			return nil
		},
	}
	openConn = func(cfg libbinder.Config) (openedConn, error) {
		return &fakeOpenedConn{
			sm: fakeServiceManager{
				checkService: func(ctx context.Context, name string) (api.Binder, error) {
					return service, nil
				},
			},
		}, nil
	}

	var gotSignal os.Signal
	signalIgnore = func(sig ...os.Signal) {
		if len(sig) != 1 {
			t.Fatalf("signalIgnore len = %d, want 1", len(sig))
		}
		gotSignal = sig[0]
	}

	type exitPanic struct{}
	var gotExit int
	processExit = func(code int) {
		gotExit = code
		panic(exitPanic{})
	}

	os.Args = []string{"libbinder-go-input", "tap", "1", "2"}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(exitPanic); !ok {
				panic(r)
			}
		}
		if gotSignal != syscall.SIGPIPE {
			t.Fatalf("signalIgnore = %v, want %v", gotSignal, syscall.SIGPIPE)
		}
		if gotExit != 0 {
			t.Fatalf("processExit code = %d, want 0", gotExit)
		}
	}()

	main()
	t.Fatal("main did not exit")
}
