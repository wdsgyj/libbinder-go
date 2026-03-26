package cmdinputproto

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/protocol"
)

func TestEncodeDecodeRequest(t *testing.T) {
	registry := newFakeBinderRegistry()
	callback := registry.register(api.StaticHandler{
		DescriptorName: "callback",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	})
	result := registry.register(api.StaticHandler{
		DescriptorName: ResultReceiverDescriptor,
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := protocol.WriteStatus(reply, protocol.Status{}); err != nil {
				return nil, err
			}
			return reply, nil
		},
	})

	p := api.NewParcel()
	if err := EncodeRequest(p, 0, 1, 2, []string{"touchscreen", "-d", "0", "tap", "10", "20"}, callback, result); err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	p.SetBinderResolvers(registry.resolve, nil)
	p.SetBinderObjectResolvers(func(obj api.ParcelObject) api.Binder {
		return registry.resolve(obj.Handle)
	}, nil)
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	req, err := DecodeRequest(p)
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	if got, want := len(req.Args), 6; got != want {
		t.Fatalf("len(req.Args) = %d, want %d", got, want)
	}
	if req.ShellCallback == nil {
		t.Fatal("req.ShellCallback = nil")
	}
	if req.ResultReceiver == nil {
		t.Fatal("req.ResultReceiver = nil")
	}
}

func TestEncodeRequestNilParcel(t *testing.T) {
	err := EncodeRequest(nil, 0, 1, 2, nil, nil, nil)
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("EncodeRequest err = %v, want ErrBadParcelable", err)
	}
}

func TestEncodeRequestInvalidFileDescriptorStages(t *testing.T) {
	tests := []struct {
		name  string
		inFD  int
		outFD int
		errFD int
	}{
		{name: "invalid out fd", inFD: 0, outFD: -1, errFD: 2},
		{name: "invalid err fd", inFD: 0, outFD: 1, errFD: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EncodeRequest(api.NewParcel(), tt.inFD, tt.outFD, tt.errFD, nil, nil, nil)
			if !errors.Is(err, api.ErrBadParcelable) {
				t.Fatalf("EncodeRequest err = %v, want ErrBadParcelable", err)
			}
		})
	}
}

func TestEncodeRequestInjectedErrors(t *testing.T) {
	oldWriteInt32 := parcelWriteInt32
	oldWriteString := parcelWriteString
	oldWriteStrongBinder := parcelWriteStrongBinder
	t.Cleanup(func() {
		parcelWriteInt32 = oldWriteInt32
		parcelWriteString = oldWriteString
		parcelWriteStrongBinder = oldWriteStrongBinder
	})

	parcelWriteInt32 = func(p *api.Parcel, v int32) error {
		return errors.New("int32 fail")
	}
	if err := EncodeRequest(api.NewParcel(), 0, 1, 2, nil, nil, nil); err == nil || !stringsContains(err.Error(), "int32 fail") {
		t.Fatalf("EncodeRequest int32 err = %v, want int32 fail", err)
	}

	parcelWriteInt32 = oldWriteInt32
	parcelWriteString = func(p *api.Parcel, v string) error {
		return errors.New("string fail")
	}
	if err := EncodeRequest(api.NewParcel(), 0, 1, 2, []string{"tap"}, nil, nil); err == nil || !stringsContains(err.Error(), "string fail") {
		t.Fatalf("EncodeRequest string err = %v, want string fail", err)
	}

	parcelWriteString = oldWriteString
	parcelWriteStrongBinder = func(p *api.Parcel, b api.Binder) error {
		return errors.New("binder fail")
	}
	if err := EncodeRequest(api.NewParcel(), 0, 1, 2, nil, nil, nil); err == nil || !stringsContains(err.Error(), "binder fail") {
		t.Fatalf("EncodeRequest binder err = %v, want binder fail", err)
	}
}

func TestDecodeRequestEmptyParcel(t *testing.T) {
	_, err := DecodeRequest(api.NewParcel())
	if err == nil {
		t.Fatal("DecodeRequest err = nil, want error")
	}
}

func TestDecodeRequestFailureStages(t *testing.T) {
	tests := []struct {
		name  string
		build func(*api.Parcel) error
	}{
		{
			name: "missing out fd",
			build: func(p *api.Parcel) error {
				return p.WriteFileDescriptor(api.NewFileDescriptor(0))
			},
		},
		{
			name: "missing err fd",
			build: func(p *api.Parcel) error {
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(0)); err != nil {
					return err
				}
				return p.WriteFileDescriptor(api.NewFileDescriptor(1))
			},
		},
		{
			name: "missing argc",
			build: func(p *api.Parcel) error {
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(0)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(1)); err != nil {
					return err
				}
				return p.WriteFileDescriptor(api.NewFileDescriptor(2))
			},
		},
		{
			name: "missing arg string",
			build: func(p *api.Parcel) error {
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(0)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(1)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(2)); err != nil {
					return err
				}
				return p.WriteInt32(1)
			},
		},
		{
			name: "missing callback binder",
			build: func(p *api.Parcel) error {
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(0)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(1)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(2)); err != nil {
					return err
				}
				if err := p.WriteInt32(0); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name: "missing result binder",
			build: func(p *api.Parcel) error {
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(0)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(1)); err != nil {
					return err
				}
				if err := p.WriteFileDescriptor(api.NewFileDescriptor(2)); err != nil {
					return err
				}
				if err := p.WriteInt32(0); err != nil {
					return err
				}
				return p.WriteStrongBinder(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := api.NewParcel()
			if err := tt.build(p); err != nil {
				t.Fatalf("build parcel: %v", err)
			}
			if err := p.SetPosition(0); err != nil {
				t.Fatalf("SetPosition: %v", err)
			}
			if _, err := DecodeRequest(p); err == nil {
				t.Fatal("DecodeRequest err = nil, want error")
			}
		})
	}
}

func TestExecuteHelpWithoutArgsSendsResult(t *testing.T) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout): %v", err)
	}
	defer func() { _ = stdoutR.Close(); _ = stdoutW.Close() }()

	handler := &resultReceiverHandler{}
	resultBinder := newFakeLocalBinder(1, handler)
	outcome, err := Execute(context.Background(), Request{
		OutFD:          api.NewFileDescriptor(int(stdoutW.Fd())),
		ErrFD:          api.NewFileDescriptor(int(os.Stderr.Fd())),
		ResultReceiver: resultBinder,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if outcome.ResultCode != -1 {
		t.Fatalf("outcome.ResultCode = %d, want -1", outcome.ResultCode)
	}
	if !outcome.SentResult {
		t.Fatal("outcome.SentResult = false")
	}
	if outcome.UsedShellCallback {
		t.Fatal("outcome.UsedShellCallback = true")
	}
	if got := readAllAndClose(t, stdoutR, stdoutW); got != UsageHeader {
		t.Fatalf("stdout = %q, want %q", got, UsageHeader)
	}
	if got := handler.Last(); got != -1 {
		t.Fatalf("resultReceiver code = %d, want -1", got)
	}
}

func TestExecuteKnownCommandParsesSourceAndDisplay(t *testing.T) {
	outcome, err := Execute(context.Background(), Request{
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:  []string{"keyboard", "-d", "DEFAULT_DISPLAY", "keyevent", "3"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := outcome.Source, "keyboard"; got != want {
		t.Fatalf("outcome.Source = %q, want %q", got, want)
	}
	if got, want := outcome.DisplayID, "DEFAULT_DISPLAY"; got != want {
		t.Fatalf("outcome.DisplayID = %q, want %q", got, want)
	}
	if got, want := outcome.Command, "keyevent"; got != want {
		t.Fatalf("outcome.Command = %q, want %q", got, want)
	}
	if outcome.ResultCode != 0 {
		t.Fatalf("outcome.ResultCode = %d, want 0", outcome.ResultCode)
	}
	if outcome.SentResult {
		t.Fatal("outcome.SentResult = true, want false")
	}
}

func TestExecuteUnknownCommandIgnoresShellCallback(t *testing.T) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout): %v", err)
	}
	defer func() { _ = stdoutR.Close(); _ = stdoutW.Close() }()

	callbackHandler := &countingHandler{}
	outcome, err := Execute(context.Background(), Request{
		OutFD:         api.NewFileDescriptor(int(stdoutW.Fd())),
		ErrFD:         api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:          []string{"not-a-command"},
		ShellCallback: newFakeLocalBinder(2, callbackHandler),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if outcome.ResultCode != -1 {
		t.Fatalf("outcome.ResultCode = %d, want -1", outcome.ResultCode)
	}
	if outcome.UsedShellCallback {
		t.Fatal("outcome.UsedShellCallback = true")
	}
	if callbackHandler.Calls() != 0 {
		t.Fatalf("callback calls = %d, want 0", callbackHandler.Calls())
	}
	if got := readAllAndClose(t, stdoutR, stdoutW); got != "Unknown command: not-a-command\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteMissingDisplayIDWritesErr(t *testing.T) {
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr): %v", err)
	}
	defer func() { _ = stderrR.Close(); _ = stderrW.Close() }()

	outcome, err := Execute(context.Background(), Request{
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(stderrW.Fd())),
		Args:  []string{"-d"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if outcome.ResultCode != -1 {
		t.Fatalf("outcome.ResultCode = %d, want -1", outcome.ResultCode)
	}
	if got := readAllAndClose(t, stderrR, stderrW); got != "Error: missing DISPLAY_ID after -d\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestExecuteHelpWriteFailure(t *testing.T) {
	_, err := Execute(context.Background(), Request{
		OutFD: api.NewFileDescriptor(-1),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
	})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("Execute err = %v, want ErrBadParcelable", err)
	}
}

func TestExecuteUnknownWriteFailure(t *testing.T) {
	_, err := Execute(context.Background(), Request{
		OutFD: api.NewFileDescriptor(-1),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:  []string{"nope"},
	})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("Execute err = %v, want ErrBadParcelable", err)
	}
}

func TestExecuteMissingDisplayIDWriteFailure(t *testing.T) {
	_, err := Execute(context.Background(), Request{
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(-1),
		Args:  []string{"-d"},
	})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("Execute err = %v, want ErrBadParcelable", err)
	}
}

func TestExecuteResultReceiverError(t *testing.T) {
	outcome, err := Execute(context.Background(), Request{
		OutFD:          api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD:          api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:           []string{"tap", "1", "2"},
		ResultReceiver: errBinder{err: errors.New("send failed")},
	})
	if err == nil || !stringsContains(err.Error(), "send failed") {
		t.Fatalf("Execute err = %v, want send failed", err)
	}
	if outcome.ResultCode != 0 {
		t.Fatalf("outcome.ResultCode = %d, want 0", outcome.ResultCode)
	}
}

func TestSendResultInjectedError(t *testing.T) {
	oldWriteInterfaceToken := parcelWriteInterfaceToken
	oldWriteInt32 := parcelWriteInt32
	oldSetPosition := parcelSetPosition
	t.Cleanup(func() {
		parcelWriteInterfaceToken = oldWriteInterfaceToken
		parcelWriteInt32 = oldWriteInt32
		parcelSetPosition = oldSetPosition
	})

	parcelSetPosition = func(p *api.Parcel, pos int) error {
		return errors.New("set position fail")
	}
	if err := sendResult(context.Background(), errBinder{err: errors.New("unused")}, 7); err == nil || !stringsContains(err.Error(), "set position fail") {
		t.Fatalf("sendResult err = %v, want set position fail", err)
	}
}

func TestWriteFDInvalid(t *testing.T) {
	err := writeFD(api.NewFileDescriptor(-1), "x")
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("writeFD err = %v, want ErrBadParcelable", err)
	}
}

type resultReceiverHandler struct {
	mu   sync.Mutex
	last int32
}

func (h *resultReceiverHandler) Descriptor() string {
	return ResultReceiverDescriptor
}

func (h *resultReceiverHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	if code != resultReceiverSendTransaction {
		return nil, api.ErrUnknownTransaction
	}
	if _, err := data.ReadInterfaceToken(); err != nil {
		return nil, err
	}
	value, err := data.ReadInt32()
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.last = value
	h.mu.Unlock()
	return api.NewParcel(), nil
}

func (h *resultReceiverHandler) Last() int32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.last
}

type countingHandler struct {
	mu    sync.Mutex
	calls int
}

func (h *countingHandler) Descriptor() string {
	return "demo.callback"
}

func (h *countingHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()
	return api.NewParcel(), nil
}

func (h *countingHandler) Calls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls
}

type fakeBinderRegistry struct {
	mu     sync.Mutex
	next   uint32
	locals map[uint32]*fakeLocalBinder
}

func newFakeBinderRegistry() *fakeBinderRegistry {
	return &fakeBinderRegistry{
		next:   1,
		locals: make(map[uint32]*fakeLocalBinder),
	}
}

func (r *fakeBinderRegistry) register(handler api.Handler) *fakeLocalBinder {
	r.mu.Lock()
	defer r.mu.Unlock()
	handle := r.next
	r.next++
	b := &fakeLocalBinder{handle: handle, handler: handler}
	r.locals[handle] = b
	return b
}

func (r *fakeBinderRegistry) resolve(handle uint32) api.Binder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locals[handle]
}

type fakeLocalBinder struct {
	handle  uint32
	handler api.Handler
}

func newFakeLocalBinder(handle uint32, handler api.Handler) *fakeLocalBinder {
	return &fakeLocalBinder{handle: handle, handler: handler}
}

func (b *fakeLocalBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b *fakeLocalBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if data == nil {
		data = api.NewParcel()
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	return api.DispatchLocalHandler(ctx, b.handler, nil, code, data, flags, api.TransactionContext{
		Code:  code,
		Flags: flags,
		Local: true,
	})
}

func (b *fakeLocalBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *fakeLocalBinder) Close() error {
	return nil
}

func (b *fakeLocalBinder) WriteBinderToParcel(p *api.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type errBinder struct {
	err error
}

func (b errBinder) Descriptor(ctx context.Context) (string, error) {
	return "err.binder", nil
}

func (b errBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	return nil, b.err
}

func (b errBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b errBinder) Close() error {
	return nil
}

func readAllAndClose(t *testing.T, r *os.File, w *os.File) string {
	t.Helper()
	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(data)
}

func stringsContains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
