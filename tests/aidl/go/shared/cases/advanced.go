package cases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

const (
	AdvancedRawBinderDescriptor = "com.wdsgyj.libbinder.aidltest.shared.RawBinder"
	advancedCallbackPrefix      = "go-callback"
)

func InvokeAdvancedCallback(ctx context.Context, prefix string, callback shared.IAdvancedCallback, value string) (string, error) {
	if callback == nil {
		return "", fmt.Errorf("nil callback")
	}
	reply, err := callback.OnSync(ctx, value)
	if err != nil {
		return "", err
	}
	return prefix + ":" + reply, nil
}

func FireAdvancedOneway(ctx context.Context, prefix string, callback shared.IAdvancedCallback, value string) error {
	if callback == nil {
		return fmt.Errorf("nil callback")
	}
	return callback.OnOneway(ctx, prefix+":"+value)
}

func ReadAllFromFileDescriptor(fd binder.FileDescriptor) (string, error) {
	return readAllDescriptor(fd)
}

func ReadAllFromParcelFileDescriptor(fd binder.ParcelFileDescriptor) (string, error) {
	return readAllDescriptor(fd.FileDescriptor)
}

func VerifyAdvancedService(ctx context.Context, svc shared.IAdvancedService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	provider, ok := any(svc).(binder.BinderProvider)
	if !ok || provider.AsBinder() == nil {
		return fmt.Errorf("service does not expose binder provider")
	}
	registrar, ok := provider.AsBinder().(binder.LocalHandlerRegistrar)
	if !ok {
		return fmt.Errorf("service binder does not support local handler registration")
	}

	rawBinder, err := registrar.RegisterLocalHandler(binder.StaticHandler{
		DescriptorName: AdvancedRawBinderDescriptor,
		Handle: func(ctx context.Context, code uint32, data *binder.Parcel) (*binder.Parcel, error) {
			return binder.NewParcel(), nil
		},
	})
	if err != nil {
		return fmt.Errorf("register raw binder: %w", err)
	}
	defer rawBinder.Close()

	echoed, err := svc.EchoBinder(ctx, rawBinder)
	if err != nil {
		return fmt.Errorf("EchoBinder: %w", err)
	}
	if echoed == nil {
		return fmt.Errorf("EchoBinder returned nil")
	}
	descriptor, err := echoed.Descriptor(ctx)
	if err != nil {
		return fmt.Errorf("EchoBinder descriptor: %w", err)
	}
	if descriptor != AdvancedRawBinderDescriptor {
		return fmt.Errorf("EchoBinder descriptor = %q, want %q", descriptor, AdvancedRawBinderDescriptor)
	}

	callback := newAdvancedCallbackRecorder(advancedCallbackPrefix)
	reply, err := svc.InvokeCallback(ctx, callback, "sync-value")
	if err != nil {
		return fmt.Errorf("InvokeCallback: %w", err)
	}
	wantReply := prefix + ":" + advancedCallbackPrefix + ":sync-value"
	if reply != wantReply {
		return fmt.Errorf("InvokeCallback reply = %q, want %q", reply, wantReply)
	}
	if got := callback.lastSync(); got != "sync-value" {
		return fmt.Errorf("InvokeCallback sync arg = %q, want %q", got, "sync-value")
	}

	if err := svc.FireOneway(ctx, callback, "oneway-value"); err != nil {
		return fmt.Errorf("FireOneway transact: %w", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	gotOneway, err := callback.waitOneway(waitCtx)
	if err != nil {
		return fmt.Errorf("FireOneway callback: %w", err)
	}
	wantOneway := prefix + ":oneway-value"
	if gotOneway != wantOneway {
		return fmt.Errorf("FireOneway arg = %q, want %q", gotOneway, wantOneway)
	}

	err = svc.FailServiceSpecific(ctx, 27, "boom")
	var remote *binder.RemoteException
	if !errors.As(err, &remote) {
		return fmt.Errorf("FailServiceSpecific error = %T, want *binder.RemoteException", err)
	}
	if remote.Code != binder.ExceptionServiceSpecific || remote.ServiceCode != 27 || remote.Message != "boom" {
		return fmt.Errorf("FailServiceSpecific remote = %+v, want code=%d service=27 message=boom", remote, binder.ExceptionServiceSpecific)
	}

	fdPath, err := writeTempText("advanced-fd-", "fd-payload")
	if err != nil {
		return err
	}
	defer os.Remove(fdPath)
	fdFile, err := os.Open(fdPath)
	if err != nil {
		return fmt.Errorf("open fd temp: %w", err)
	}
	fdValue, err := svc.ReadFromFileDescriptor(ctx, binder.NewFileDescriptor(int(fdFile.Fd())))
	closeErr := fdFile.Close()
	if err != nil {
		return fmt.Errorf("ReadFromFileDescriptor: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close fd temp: %w", closeErr)
	}
	if fdValue != "fd-payload" {
		return fmt.Errorf("ReadFromFileDescriptor = %q, want %q", fdValue, "fd-payload")
	}

	pfdPath, err := writeTempText("advanced-pfd-", "pfd-payload")
	if err != nil {
		return err
	}
	defer os.Remove(pfdPath)
	pfdFile, err := os.Open(pfdPath)
	if err != nil {
		return fmt.Errorf("open pfd temp: %w", err)
	}
	dupFD, err := syscall.Dup(int(pfdFile.Fd()))
	closeErr = pfdFile.Close()
	if err != nil {
		return fmt.Errorf("dup pfd temp: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close pfd temp: %w", closeErr)
	}
	pfd := binder.NewParcelFileDescriptor(dupFD)
	pfdValue, err := svc.ReadFromParcelFileDescriptor(ctx, pfd)
	_ = pfd.Close()
	if err != nil {
		return fmt.Errorf("ReadFromParcelFileDescriptor: %w", err)
	}
	if pfdValue != "pfd-payload" {
		return fmt.Errorf("ReadFromParcelFileDescriptor = %q, want %q", pfdValue, "pfd-payload")
	}

	return nil
}

type advancedCallbackRecorder struct {
	prefix string
	syncCh chan string
	oneCh  chan string
}

func newAdvancedCallbackRecorder(prefix string) *advancedCallbackRecorder {
	return &advancedCallbackRecorder{
		prefix: prefix,
		syncCh: make(chan string, 1),
		oneCh:  make(chan string, 1),
	}
}

func (r *advancedCallbackRecorder) OnSync(ctx context.Context, value string) (string, error) {
	select {
	case r.syncCh <- value:
	default:
	}
	return r.prefix + ":" + value, nil
}

func (r *advancedCallbackRecorder) OnOneway(ctx context.Context, value string) error {
	select {
	case r.oneCh <- value:
	default:
	}
	return nil
}

func (r *advancedCallbackRecorder) lastSync() string {
	select {
	case value := <-r.syncCh:
		return value
	default:
		return ""
	}
}

func (r *advancedCallbackRecorder) waitOneway(ctx context.Context) (string, error) {
	select {
	case value := <-r.oneCh:
		return value, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func readAllDescriptor(fd binder.FileDescriptor) (string, error) {
	if fd.FD() < 0 {
		return "", fmt.Errorf("%w: invalid file descriptor %d", binder.ErrBadParcelable, fd.FD())
	}
	dupFD, err := syscall.Dup(fd.FD())
	if err != nil {
		return "", fmt.Errorf("dup fd %d: %w", fd.FD(), err)
	}
	file := os.NewFile(uintptr(dupFD), "binder-fd")
	defer file.Close()
	data, err := io.ReadAll(file)
	if fd.Owned() {
		_ = fd.Close()
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeTempText(prefix string, value string) (string, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(value); err != nil {
		f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close temp file: %w", err)
	}
	return path, nil
}
