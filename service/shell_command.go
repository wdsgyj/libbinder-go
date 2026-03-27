package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

const (
	ShellCallbackDescriptor  = "com.android.internal.os.IShellCallback"
	ResultReceiverDescriptor = "com.android.internal.os.IResultReceiver"

	shellCallbackOpenFileTransaction = api.FirstCallTransaction
	resultReceiverSendTransaction    = api.FirstCallTransaction
)

type FileAccessChecker interface {
	CheckFileAccess(path string, seLinuxContext string, read bool, write bool) error
}

type ShellCommandIO struct {
	InFD  api.FileDescriptor
	OutFD api.FileDescriptor
	ErrFD api.FileDescriptor
}

func DefaultShellCommandIO() ShellCommandIO {
	return ShellCommandIO{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
	}
}

type ShellCommandRequest struct {
	InFD           api.FileDescriptor
	OutFD          api.FileDescriptor
	ErrFD          api.FileDescriptor
	Args           []string
	ShellCallback  api.Binder
	ResultReceiver api.Binder
}

type ShellCommandBuildError struct {
	Err error
}

func (e *ShellCommandBuildError) Error() string {
	if e == nil || e.Err == nil {
		return api.ErrBadParcelable.Error()
	}
	return e.Err.Error()
}

func (e *ShellCommandBuildError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

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
	if err := data.WriteInterfaceToken(ShellCallbackDescriptor); err != nil {
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
		return api.ParcelFileDescriptor{}, &api.RemoteException{
			Code:    status.Remote.Code,
			Message: status.Remote.Message,
		}
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

	on bool

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
	return &ShellCallbackHandler{
		errorLog:      writerOrDiscard(errorLog),
		workingDir:    workingDir,
		accessChecker: opts.AccessChecker,
		on:            true,
	}
}

func (h *ShellCallbackHandler) Descriptor() string {
	return ShellCallbackDescriptor
}

func (h *ShellCallbackHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	if code != shellCallbackOpenFileTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if err := expectInterfaceToken(data, ShellCallbackDescriptor); err != nil {
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
	if err := writeOptionalParcelFileDescriptor(reply, h.OpenFile(path, seLinuxContext, mode)); err != nil {
		return nil, err
	}
	return reply, nil
}

func (h *ShellCallbackHandler) OpenFile(path string, seLinuxContext string, mode string) int {
	fullPath := path
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(h.workingDir, path)
	}
	if !h.on {
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
	h.on = false
}

func (h *ShellCallbackHandler) Close() error {
	if h == nil {
		return nil
	}
	h.Deactivate()

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
	if err := data.WriteInterfaceToken(ResultReceiverDescriptor); err != nil {
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
	return &ResultReceiverHandler{done: make(chan struct{})}
}

func (h *ResultReceiverHandler) Descriptor() string {
	return ResultReceiverDescriptor
}

func (h *ResultReceiverHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	if code != resultReceiverSendTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if err := expectInterfaceToken(data, ResultReceiverDescriptor); err != nil {
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
	defer h.mu.Unlock()
	h.result = resultCode
	if !h.have {
		h.have = true
		close(h.done)
	}
}

func (h *ResultReceiverHandler) Wait(ctx context.Context) (int32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-h.done:
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.result, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

type ShellCommandService struct {
	name          string
	service       api.Binder
	io            ShellCommandIO
	workingDir    string
	accessChecker FileAccessChecker
}

func NewShellCommandService(name string, service api.Binder) *ShellCommandService {
	return &ShellCommandService{
		name:    name,
		service: service,
		io:      DefaultShellCommandIO(),
	}
}

func (s *ShellCommandService) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

func (s *ShellCommandService) Binder() api.Binder {
	if s == nil {
		return nil
	}
	return s.service
}

func (s *ShellCommandService) WithShellIO(io ShellCommandIO) *ShellCommandService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.io = io
	return &clone
}

func (s *ShellCommandService) WithWorkingDir(dir string) *ShellCommandService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.workingDir = dir
	return &clone
}

func (s *ShellCommandService) WithFileAccessChecker(checker FileAccessChecker) *ShellCommandService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.accessChecker = checker
	return &clone
}

func (s *ShellCommandService) Command(ctx context.Context, args ...string) (int, error) {
	if s == nil || s.service == nil {
		return 0, api.ErrUnsupported
	}
	registrar, ok := s.service.(api.LocalHandlerRegistrar)
	if !ok {
		return 0, api.ErrUnsupported
	}

	callback := NewShellCallbackHandler(io.Discard, ShellCallbackOptions{
		WorkingDir:    s.workingDir,
		AccessChecker: s.accessChecker,
	})
	result := NewResultReceiverHandler()

	callbackBinder, err := registrar.RegisterLocalHandler(callback)
	if err != nil {
		_ = callback.Close()
		return 0, err
	}
	defer func() { _ = callbackBinder.Close() }()

	resultBinder, err := registrar.RegisterLocalHandler(result)
	if err != nil {
		_ = callback.Close()
		return 0, err
	}
	defer func() { _ = resultBinder.Close() }()

	err = TransactShellCommand(ctx, s.service, ShellCommandRequest{
		InFD:           s.io.InFD,
		OutFD:          s.io.OutFD,
		ErrFD:          s.io.ErrFD,
		Args:           args,
		ShellCallback:  callbackBinder,
		ResultReceiver: resultBinder,
	})
	if err != nil {
		callback.Deactivate()
		_ = callback.Close()
		return 0, err
	}

	callback.Deactivate()
	resultCode, waitErr := result.Wait(ctx)
	closeErr := callback.Close()
	if waitErr != nil {
		return 0, waitErr
	}
	if closeErr != nil {
		return int(resultCode), closeErr
	}
	return int(resultCode), nil
}

func (s *ShellCommandService) ExecuteCommand(ctx context.Context, argv []string) (int, error) {
	if s == nil {
		return 0, api.ErrUnsupported
	}
	return s.Command(ctx, argv...)
}

func WriteShellCommandRequest(p *api.Parcel, req ShellCommandRequest) error {
	if err := p.WriteFileDescriptor(req.InFD); err != nil {
		return err
	}
	if err := p.WriteFileDescriptor(req.OutFD); err != nil {
		return err
	}
	if err := p.WriteFileDescriptor(req.ErrFD); err != nil {
		return err
	}
	if err := p.WriteInt32(int32(len(req.Args))); err != nil {
		return err
	}
	for _, arg := range req.Args {
		if err := p.WriteString(arg); err != nil {
			return err
		}
	}
	if err := p.WriteStrongBinder(req.ShellCallback); err != nil {
		return err
	}
	if err := p.WriteStrongBinder(req.ResultReceiver); err != nil {
		return err
	}
	return p.SetPosition(0)
}

func BuildShellCommandParcel(req ShellCommandRequest) (*api.Parcel, error) {
	data := api.NewParcel()
	if err := WriteShellCommandRequest(data, req); err != nil {
		return nil, err
	}
	return data, nil
}

func TransactShellCommand(ctx context.Context, service api.Binder, req ShellCommandRequest) error {
	if service == nil {
		return api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}

	data, err := BuildShellCommandParcel(req)
	if err != nil {
		return &ShellCommandBuildError{Err: err}
	}
	_, err = service.Transact(ctx, api.ShellCommandTransaction, data, api.FlagNone)
	return err
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
		return api.ParcelFileDescriptor{}, nil
	}
	return p.ReadParcelFileDescriptor()
}

func openFlagsForMode(mode string) (flags int, checkRead bool, checkWrite bool, ok bool) {
	switch mode {
	case "r":
		return syscall.O_RDONLY, true, false, true
	case "w":
		return syscall.O_WRONLY | syscall.O_CREAT | syscall.O_TRUNC, false, true, true
	case "wa":
		return syscall.O_WRONLY | syscall.O_CREAT | syscall.O_APPEND, false, true, true
	case "rw":
		return syscall.O_RDWR | syscall.O_CREAT, true, true, true
	case "rwt":
		return syscall.O_RDWR | syscall.O_CREAT | syscall.O_TRUNC, true, true, true
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

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
