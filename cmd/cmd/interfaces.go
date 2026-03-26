package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

const (
	shellCallbackDescriptor  = "com.android.internal.os.IShellCallback"
	resultReceiverDescriptor = "com.android.internal.os.IResultReceiver"

	shellCallbackOpenFileTransaction = api.FirstCallTransaction
	resultReceiverSendTransaction    = api.FirstCallTransaction
)

type ShellCallback interface {
	OpenFile(ctx context.Context, path string, seLinuxContext string, mode string) (api.ParcelFileDescriptor, error)
}

func NewShellCallbackProxy(b api.Binder) ShellCallback {
	if b == nil {
		return nil
	}
	return shellCallbackProxy{binder: b}
}

type shellCallbackProxy struct {
	binder api.Binder
}

func (p shellCallbackProxy) OpenFile(ctx context.Context, path string, seLinuxContext string, mode string) (api.ParcelFileDescriptor, error) {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(shellCallbackDescriptor); err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if err := data.WriteString(path); err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if err := data.WriteString(seLinuxContext); err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if err := data.WriteString(mode); err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if err := data.SetPosition(0); err != nil {
		return api.ParcelFileDescriptor{}, err
	}

	reply, err := p.binder.Transact(ctx, shellCallbackOpenFileTransaction, data, api.FlagNone)
	if err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if reply == nil {
		return api.ParcelFileDescriptor{}, api.ErrBadParcelable
	}
	status, err := protocol.ReadStatus(reply)
	if err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if status.TransportErr != nil {
		return api.ParcelFileDescriptor{}, status.TransportErr
	}
	if status.Remote != nil {
		return api.ParcelFileDescriptor{}, &api.RemoteException{Code: status.Remote.Code, Message: status.Remote.Message}
	}
	return readOptionalParcelFileDescriptor(reply)
}

type ShellCallbackOptions struct {
	WorkingDir    string
	AccessChecker FileAccessChecker
}

type ShellCallbackHandler struct {
	errorLog      io.Writer
	workingDir    string
	accessChecker FileAccessChecker
	active        atomic.Bool

	mu      sync.Mutex
	openFDs []int
}

func NewShellCallbackHandler(errorLog io.Writer, opts ShellCallbackOptions) *ShellCallbackHandler {
	workingDir := opts.WorkingDir
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		}
	}
	h := &ShellCallbackHandler{
		errorLog:      writerOrDiscard(errorLog),
		workingDir:    workingDir,
		accessChecker: opts.AccessChecker,
	}
	h.active.Store(true)
	return h
}

func (h *ShellCallbackHandler) Descriptor() string {
	return shellCallbackDescriptor
}

func (h *ShellCallbackHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	if code != shellCallbackOpenFileTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if err := expectInterfaceToken(data, shellCallbackDescriptor); err != nil {
		return nil, err
	}

	path, err := data.ReadString()
	if err != nil {
		return nil, err
	}
	seLinuxContext, err := data.ReadString()
	if err != nil {
		return nil, err
	}
	mode, err := data.ReadString()
	if err != nil {
		return nil, err
	}

	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		return nil, err
	}
	fd := h.OpenFile(path, seLinuxContext, mode)
	if err := writeOptionalParcelFileDescriptor(reply, fd); err != nil {
		return nil, err
	}
	return reply, nil
}

func (h *ShellCallbackHandler) OpenFile(path string, seLinuxContext string, mode string) int {
	fullPath := filepath.Join(h.workingDir, path)
	if !h.active.Load() {
		fmt.Fprintf(h.errorLog, "Open attempt after active for: %s\n", fullPath)
		return -int(syscall.EPERM)
	}

	flags, checkRead, checkWrite, ok := openFlagsForMode(mode)
	if !ok {
		fmt.Fprintf(h.errorLog, "Invalid mode requested: %s\n", mode)
		return -int(syscall.EINVAL)
	}

	fd, err := syscall.Open(fullPath, flags, 0o770)
	if err != nil {
		return -errnoValue(err)
	}
	if h.accessChecker != nil && seLinuxContext != "" {
		if err := h.accessChecker.CheckFileAccess(fullPath, seLinuxContext, checkRead, checkWrite); err != nil {
			_ = syscall.Close(fd)
			fmt.Fprintf(h.errorLog, "System server has no access to file %s (context %s): %v\n", fullPath, seLinuxContext, err)
			return -int(syscall.EPERM)
		}
	}

	h.mu.Lock()
	h.openFDs = append(h.openFDs, fd)
	h.mu.Unlock()
	return fd
}

func (h *ShellCallbackHandler) Deactivate() {
	h.active.Store(false)
}

func (h *ShellCallbackHandler) Close() error {
	if h == nil {
		return nil
	}

	h.mu.Lock()
	fds := h.openFDs
	h.openFDs = nil
	h.mu.Unlock()

	var joined error
	for _, fd := range fds {
		if fd < 0 {
			continue
		}
		if err := syscall.Close(fd); err != nil && !errors.Is(err, syscall.EBADF) {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

type ResultReceiver interface {
	Send(ctx context.Context, resultCode int32) error
}

func NewResultReceiverProxy(b api.Binder) ResultReceiver {
	if b == nil {
		return nil
	}
	return resultReceiverProxy{binder: b}
}

type resultReceiverProxy struct {
	binder api.Binder
}

func (p resultReceiverProxy) Send(ctx context.Context, resultCode int32) error {
	data := api.NewParcel()
	if err := data.WriteInterfaceToken(resultReceiverDescriptor); err != nil {
		return err
	}
	if err := data.WriteInt32(resultCode); err != nil {
		return err
	}
	if err := data.SetPosition(0); err != nil {
		return err
	}
	_, err := p.binder.Transact(ctx, resultReceiverSendTransaction, data, api.FlagOneway)
	return err
}

type ResultReceiverHandler struct {
	mu     sync.Mutex
	done   chan struct{}
	result int32
	have   bool
}

func NewResultReceiverHandler() *ResultReceiverHandler {
	return &ResultReceiverHandler{
		done: make(chan struct{}),
	}
}

func (h *ResultReceiverHandler) Descriptor() string {
	return resultReceiverDescriptor
}

func (h *ResultReceiverHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	if code != resultReceiverSendTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if err := expectInterfaceToken(data, resultReceiverDescriptor); err != nil {
		return nil, err
	}
	resultCode, err := data.ReadInt32()
	if err != nil {
		return nil, err
	}
	h.Send(resultCode)

	reply := api.NewParcel()
	if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
		return nil, err
	}
	return reply, nil
}

func (h *ResultReceiverHandler) Send(resultCode int32) {
	h.mu.Lock()
	h.result = resultCode
	if !h.have {
		h.have = true
		close(h.done)
	}
	h.mu.Unlock()
}

func (h *ResultReceiverHandler) Wait(ctx context.Context) (int32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-h.done:
		h.mu.Lock()
		result := h.result
		h.mu.Unlock()
		return result, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func expectInterfaceToken(p *api.Parcel, want string) error {
	got, err := p.ReadInterfaceToken()
	if err != nil {
		return err
	}
	if got != want {
		return &api.StatusCodeError{Code: api.StatusPermissionDenied}
	}
	return nil
}

func writeOptionalParcelFileDescriptor(p *api.Parcel, fd int) error {
	if fd < 0 {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return p.WriteParcelFileDescriptor(api.ParcelFileDescriptor{
		FileDescriptor: api.NewFileDescriptor(fd),
	})
}

func readOptionalParcelFileDescriptor(p *api.Parcel) (api.ParcelFileDescriptor, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return api.ParcelFileDescriptor{}, err
	}
	if present == 0 {
		return api.ParcelFileDescriptor{FileDescriptor: api.NewFileDescriptor(-1)}, nil
	}
	if present != 1 {
		return api.ParcelFileDescriptor{}, fmt.Errorf("%w: invalid file descriptor presence %d", api.ErrBadParcelable, present)
	}
	return p.ReadParcelFileDescriptor()
}

func openFlagsForMode(mode string) (flags int, checkRead bool, checkWrite bool, ok bool) {
	switch mode {
	case "w":
		return syscall.O_WRONLY | syscall.O_CREAT | syscall.O_TRUNC, false, true, true
	case "w+":
		return syscall.O_RDWR | syscall.O_CREAT | syscall.O_TRUNC, true, true, true
	case "r":
		return syscall.O_RDONLY, true, false, true
	case "r+":
		return syscall.O_RDWR, true, true, true
	default:
		return 0, false, false, false
	}
}

func errnoValue(err error) int {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return int(errno)
	}
	return int(syscall.EIO)
}
