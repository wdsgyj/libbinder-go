package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestWriteShellCommandRequest(t *testing.T) {
	registry := newShellTestBinderRegistry()
	callback := registry.register(api.StaticHandler{DescriptorName: ShellCallbackDescriptor, Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
		return api.NewParcel(), nil
	}})
	result := registry.register(api.StaticHandler{DescriptorName: ResultReceiverDescriptor, Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
		return api.NewParcel(), nil
	}})

	req := ShellCommandRequest{
		InFD:           api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD:          api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD:          api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:           []string{"help"},
		ShellCallback:  callback,
		ResultReceiver: result,
	}
	p, err := BuildShellCommandParcel(req)
	if err != nil {
		t.Fatalf("BuildShellCommandParcel: %v", err)
	}
	p.SetBinderResolvers(registry.resolve, nil)
	p.SetBinderObjectResolvers(func(obj api.ParcelObject) api.Binder { return registry.resolve(obj.Handle) }, nil)

	got, err := parseShellCommandRequest(p)
	if err != nil {
		t.Fatalf("parseShellCommandRequest: %v", err)
	}
	if got.InFD.FD() != req.InFD.FD() || got.OutFD.FD() != req.OutFD.FD() || got.ErrFD.FD() != req.ErrFD.FD() {
		t.Fatalf("fd triple = (%d, %d, %d), want (%d, %d, %d)", got.InFD.FD(), got.OutFD.FD(), got.ErrFD.FD(), req.InFD.FD(), req.OutFD.FD(), req.ErrFD.FD())
	}
	if strings.Join(got.Args, ",") != strings.Join(req.Args, ",") {
		t.Fatalf("args = %#v, want %#v", got.Args, req.Args)
	}
	if got.ShellCallback == nil || got.ResultReceiver == nil {
		t.Fatalf("callback/result = (%v,%v), want both non-nil", got.ShellCallback, got.ResultReceiver)
	}
}

func TestWriteShellCommandRequestNilParcel(t *testing.T) {
	err := WriteShellCommandRequest(nil, ShellCommandRequest{
		InFD:  api.NewFileDescriptor(0),
		OutFD: api.NewFileDescriptor(1),
		ErrFD: api.NewFileDescriptor(2),
	})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("WriteShellCommandRequest(nil) err = %v, want ErrBadParcelable", err)
	}
}

func TestShellCommandServiceCommand(t *testing.T) {
	registry := newShellTestBinderRegistry()
	service := &shellTestService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			if got, want := strings.Join(req.Args, ","), "help"; got != want {
				return errors.New("unexpected args: " + got)
			}
			if req.ShellCallback == nil || req.ResultReceiver == nil {
				return errors.New("missing callback or result receiver")
			}
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 7)
		},
	}

	got, err := NewShellCommandService("activity", service).Command(context.Background(), "help")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if got != 7 {
		t.Fatalf("result code = %d, want 7", got)
	}
}

func TestShellCommandServiceCallbackOpenFile(t *testing.T) {
	registry := newShellTestBinderRegistry()
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "trace.bin")
	var checkerCalls int

	service := &shellTestService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			cb := NewShellCallbackProxy(req.ShellCallback)
			pfd, err := cb.OpenFile(ctx, "trace.bin", "u:r:system_server:s0", "w")
			if err != nil {
				return err
			}
			if pfd.FD() < 0 {
				return errors.New("received invalid fd")
			}
			if _, err := syscall.Write(pfd.FD(), []byte("trace data")); err != nil {
				return err
			}
			_ = syscall.Close(pfd.FD())
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 0)
		},
	}

	_, err := NewShellCommandService("activity", service).
		WithWorkingDir(dir).
		WithFileAccessChecker(fileAccessCheckerFunc(func(path string, seLinuxContext string, read bool, write bool) error {
			checkerCalls++
			if path != targetPath {
				t.Fatalf("path = %q, want %q", path, targetPath)
			}
			if seLinuxContext != "u:r:system_server:s0" || read || !write {
				t.Fatalf("unexpected access check args: ctx=%q read=%v write=%v", seLinuxContext, read, write)
			}
			return nil
		})).
		Command(context.Background(), "trace-ipc", "stop", "--dump-file", "trace.bin")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if checkerCalls != 1 {
		t.Fatalf("checkerCalls = %d, want 1", checkerCalls)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := string(content); got != "trace data" {
		t.Fatalf("content = %q, want %q", got, "trace data")
	}
}

func TestShellCommandServiceBuildError(t *testing.T) {
	registry := newShellTestBinderRegistry()
	service := &shellTestService{registry: registry}

	_, err := NewShellCommandService("activity", service).
		WithShellIO(ShellCommandIO{
			InFD:  api.NewFileDescriptor(-1),
			OutFD: api.NewFileDescriptor(1),
			ErrFD: api.NewFileDescriptor(2),
		}).
		Command(context.Background(), "help")
	var buildErr *ShellCommandBuildError
	if !errors.As(err, &buildErr) {
		t.Fatalf("err = %T, want *ShellCommandBuildError", err)
	}
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("err = %v, want ErrBadParcelable", err)
	}
}

func TestShellCommandServiceUnsupportedRegistrar(t *testing.T) {
	service := inputTestBinder{}
	_, err := NewShellCommandService("activity", service).Command(context.Background(), "help")
	if !errors.Is(err, api.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}
