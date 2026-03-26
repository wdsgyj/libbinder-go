package service

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	libbinder "github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestLookupInputService(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		want := &inputTestBinder{}
		sm := inputTestServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) {
				if ctx == nil {
					t.Fatal("ctx = nil")
				}
				if name != InputServiceName {
					t.Fatalf("name = %q, want %q", name, InputServiceName)
				}
				return want, nil
			},
		}

		got, err := LookupInputService(nil, sm)
		if err != nil {
			t.Fatalf("LookupInputService: %v", err)
		}
		if got != want {
			t.Fatalf("LookupInputService() = %#v, want %#v", got, want)
		}
	})

	t.Run("missing", func(t *testing.T) {
		sm := inputTestServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) {
				return nil, nil
			},
		}
		_, err := LookupInputService(context.Background(), sm)
		if !errors.Is(err, api.ErrNoService) {
			t.Fatalf("LookupInputService() err = %v, want ErrNoService", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		sm := inputTestServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) {
				return nil, errors.New("lookup failed")
			},
		}
		_, err := LookupInputService(context.Background(), sm)
		if err == nil || !strings.Contains(err.Error(), "lookup failed") {
			t.Fatalf("LookupInputService() err = %v, want lookup failed", err)
		}
	})

	t.Run("nil service manager", func(t *testing.T) {
		_, err := LookupInputService(context.Background(), nil)
		if !errors.Is(err, api.ErrUnsupported) {
			t.Fatalf("LookupInputService(nil) err = %v, want ErrUnsupported", err)
		}
	})
}

func TestLookupInputManagerService(t *testing.T) {
	want := &inputTestBinder{}
	sm := inputTestServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if name != InputServiceName {
				t.Fatalf("name = %q, want %q", name, InputServiceName)
			}
			return want, nil
		},
	}

	got, err := LookupInputManagerService(context.Background(), sm)
	if err != nil {
		t.Fatalf("LookupInputManagerService: %v", err)
	}
	if got == nil {
		t.Fatal("LookupInputManagerService() = nil")
	}
	if got.Binder() != want {
		t.Fatalf("Binder() = %#v, want %#v", got.Binder(), want)
	}
	if !reflect.DeepEqual(got.CommandOptions(), InputCommandOptions{}) {
		t.Fatalf("CommandOptions() = %#v, want zero value", got.CommandOptions())
	}
}

func TestWriteInputShellCommandRequest(t *testing.T) {
	req := InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:  []string{"tap", "100", "200"},
	}

	p, err := BuildInputShellCommandParcel(req)
	if err != nil {
		t.Fatalf("BuildInputShellCommandParcel: %v", err)
	}

	got, err := parseInputShellCommandRequest(p)
	if err != nil {
		t.Fatalf("parseInputShellCommandRequest: %v", err)
	}
	if got.InFD.FD() != req.InFD.FD() || got.OutFD.FD() != req.OutFD.FD() || got.ErrFD.FD() != req.ErrFD.FD() {
		t.Fatalf("fd triple = (%d, %d, %d), want (%d, %d, %d)", got.InFD.FD(), got.OutFD.FD(), got.ErrFD.FD(), req.InFD.FD(), req.OutFD.FD(), req.ErrFD.FD())
	}
	if strings.Join(got.Args, ",") != strings.Join(req.Args, ",") {
		t.Fatalf("args = %#v, want %#v", got.Args, req.Args)
	}
	if got.ShellCallback != nil {
		t.Fatalf("ShellCallback = %#v, want nil", got.ShellCallback)
	}
	if got.ResultReceiver != nil {
		t.Fatalf("ResultReceiver = %#v, want nil", got.ResultReceiver)
	}
}

func TestWriteInputShellCommandRequestNilParcel(t *testing.T) {
	err := WriteInputShellCommandRequest(nil, InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(0),
		OutFD: api.NewFileDescriptor(1),
		ErrFD: api.NewFileDescriptor(2),
		Args:  []string{"tap"},
	})
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("WriteInputShellCommandRequest(nil) err = %v, want ErrBadParcelable", err)
	}
}

func TestTransactInputShellCommand(t *testing.T) {
	service := inputTestBinder{
		transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			if ctx == nil {
				t.Fatal("ctx = nil")
			}
			if code != api.ShellCommandTransaction {
				t.Fatalf("code = %d, want %d", code, api.ShellCommandTransaction)
			}
			if flags != api.FlagNone {
				t.Fatalf("flags = %#x, want 0", flags)
			}
			got, err := parseInputShellCommandRequest(data)
			if err != nil {
				t.Fatalf("parseInputShellCommandRequest: %v", err)
			}
			if strings.Join(got.Args, ",") != "keyevent,3" {
				t.Fatalf("args = %#v, want [keyevent 3]", got.Args)
			}
			return api.NewParcel(), nil
		},
	}

	err := TransactInputShellCommand(nil, service, InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:  []string{"keyevent", "3"},
	})
	if err != nil {
		t.Fatalf("TransactInputShellCommand: %v", err)
	}
}

func TestTransactInputShellCommandBuildError(t *testing.T) {
	err := TransactInputShellCommand(context.Background(), inputTestBinder{}, InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(-1),
		OutFD: api.NewFileDescriptor(1),
		ErrFD: api.NewFileDescriptor(2),
		Args:  []string{"tap"},
	})
	var buildErr *InputShellCommandBuildError
	if !errors.As(err, &buildErr) {
		t.Fatalf("TransactInputShellCommand() err = %T, want *InputShellCommandBuildError", err)
	}
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("TransactInputShellCommand() err = %v, want ErrBadParcelable", err)
	}
}

func TestExecuteInputShellCommand(t *testing.T) {
	service := inputTestBinder{
		transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}
	sm := inputTestServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return service, nil
		},
	}

	err := ExecuteInputShellCommand(context.Background(), sm, InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
		Args:  []string{"text", "hello"},
	})
	if err != nil {
		t.Fatalf("ExecuteInputShellCommand: %v", err)
	}
}

func TestExecuteInputShellCommandMissingService(t *testing.T) {
	sm := inputTestServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return nil, nil
		},
	}
	err := ExecuteInputShellCommand(context.Background(), sm, InputShellCommandRequest{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
	})
	if !errors.Is(err, api.ErrNoService) {
		t.Fatalf("ExecuteInputShellCommand() err = %v, want ErrNoService", err)
	}
}

func TestInputManagerServiceConfiguration(t *testing.T) {
	base := NewInputManagerService(inputTestBinder{})
	if base == nil {
		t.Fatal("NewInputManagerService() = nil")
	}

	scoped := base.WithSource(InputSourceMouse).WithDisplayID(7)
	if got, want := scoped.CommandOptions(), (InputCommandOptions{
		Source:    InputSourceMouse,
		DisplayID: libbinder.IntPtr(7),
	}); !reflect.DeepEqual(got, want) {
		t.Fatalf("CommandOptions() = %#v, want %#v", got, want)
	}

	cleared := scoped.WithoutSource().WithoutDisplayID()
	if got := cleared.CommandOptions(); !reflect.DeepEqual(got, InputCommandOptions{}) {
		t.Fatalf("cleared CommandOptions() = %#v, want zero value", got)
	}
	if got := base.CommandOptions(); !reflect.DeepEqual(got, InputCommandOptions{}) {
		t.Fatalf("base CommandOptions() mutated to %#v", got)
	}
}

func TestInputManagerServiceWithShellIO(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdin): %v", err)
	}
	defer func() {
		_ = inR.Close()
		_ = inW.Close()
	}()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stdout): %v", err)
	}
	defer func() {
		_ = outR.Close()
		_ = outW.Close()
	}()
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(stderr): %v", err)
	}
	defer func() {
		_ = errR.Close()
		_ = errW.Close()
	}()

	got := captureInputShellCommand(t, func(s *InputManagerService) error {
		return s.WithShellIO(InputShellIO{
			InFD:  api.NewFileDescriptor(int(inR.Fd())),
			OutFD: api.NewFileDescriptor(int(outW.Fd())),
			ErrFD: api.NewFileDescriptor(int(errW.Fd())),
		}).Press(context.Background())
	})

	if got.InFD.FD() != int(inR.Fd()) || got.OutFD.FD() != int(outW.Fd()) || got.ErrFD.FD() != int(errW.Fd()) {
		t.Fatalf("fd triple = (%d, %d, %d), want (%d, %d, %d)", got.InFD.FD(), got.OutFD.FD(), got.ErrFD.FD(), inR.Fd(), outW.Fd(), errW.Fd())
	}
}

func TestInputManagerServiceCommands(t *testing.T) {
	tests := []struct {
		name string
		call func(*InputManagerService) error
		want []string
	}{
		{
			name: "text",
			call: func(s *InputManagerService) error {
				return s.Text(context.Background(), "hello world")
			},
			want: []string{"text", "hello world"},
		},
		{
			name: "text with command options",
			call: func(s *InputManagerService) error {
				return s.TextWithOptions(context.Background(), InputCommandOptions{
					Source:    InputSourceJoystick,
					DisplayID: libbinder.IntPtr(9),
				}, "hello")
			},
			want: []string{"joystick", "-d", "9", "text", "hello"},
		},
		{
			name: "keyevent",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceKeyboard).KeyEventWithOptions(context.Background(), InputKeyEventOptions{
					DoubleTap: true,
					Async:     true,
					DelayMS:   libbinder.Int64Ptr(40),
				}, "3", "4")
			},
			want: []string{"keyboard", "keyevent", "--doubletap", "--async", "--delay", "40", "3", "4"},
		},
		{
			name: "tap",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceTouchscreen).Tap(context.Background(), 100, 200)
			},
			want: []string{"touchscreen", "tap", "100", "200"},
		},
		{
			name: "swipe",
			call: func(s *InputManagerService) error {
				return s.Swipe(context.Background(), 1, 2, 3, 4, 250)
			},
			want: []string{"swipe", "1", "2", "3", "4", "250"},
		},
		{
			name: "draganddrop",
			call: func(s *InputManagerService) error {
				return s.WithDisplayID(InputDisplayDefault).DragAndDrop(context.Background(), 1, 2, 3, 4, 300)
			},
			want: []string{"-d", "DEFAULT_DISPLAY", "draganddrop", "1", "2", "3", "4", "300"},
		},
		{
			name: "press",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceTrackball).Press(context.Background())
			},
			want: []string{"trackball", "press"},
		},
		{
			name: "roll",
			call: func(s *InputManagerService) error {
				return s.Roll(context.Background(), -1.5, 2.5)
			},
			want: []string{"roll", "-1.5", "2.5"},
		},
		{
			name: "scroll",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceMouse).Scroll(context.Background(), []InputAxisValue{
					{Axis: InputAxisVScroll, Value: -1.5},
					{Axis: InputAxisHScroll, Value: 2},
				}, 10, 20)
			},
			want: []string{"mouse", "scroll", "10", "20", "--axis", "VSCROLL,-1.5", "--axis", "HSCROLL,2"},
		},
		{
			name: "motionevent",
			call: func(s *InputManagerService) error {
				return s.WithDisplayID(InputDisplayInvalid).MotionEvent(context.Background(), InputMotionActionCancel)
			},
			want: []string{"-d", "INVALID_DISPLAY", "motionevent", "CANCEL"},
		},
		{
			name: "keycombination",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceKeyboard).KeyCombination(context.Background(), []string{"29", "47"}, 120)
			},
			want: []string{"keyboard", "keycombination", "-t", "120", "29", "47"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureInputShellCommand(t, tt.call)
			assertStringSlice(t, got.Args, tt.want)
		})
	}
}

func TestInputManagerServiceValidation(t *testing.T) {
	tests := []struct {
		name string
		call func(*InputManagerService) error
		want string
	}{
		{
			name: "invalid source",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSource("bad-source")).Tap(context.Background(), 1, 2)
			},
			want: "unsupported source",
		},
		{
			name: "keyevent longpress and duration",
			call: func(s *InputManagerService) error {
				return s.KeyEventWithOptions(context.Background(), InputKeyEventOptions{
					LongPress:  true,
					DurationMS: libbinder.Int64Ptr(10),
				}, "3")
			},
			want: "either durationMs or longPress",
		},
		{
			name: "swipe too many durations",
			call: func(s *InputManagerService) error {
				return s.Swipe(context.Background(), 1, 2, 3, 4, 10, 20)
			},
			want: "accepts at most one duration",
		},
		{
			name: "scroll pointer requires coordinates",
			call: func(s *InputManagerService) error {
				return s.WithSource(InputSourceMouse).Scroll(context.Background(), []InputAxisValue{
					{Axis: InputAxisVScroll, Value: 1},
				})
			},
			want: "scroll requires x and y",
		},
		{
			name: "motionevent cancel coordinate count",
			call: func(s *InputManagerService) error {
				return s.MotionEvent(context.Background(), InputMotionActionCancel, 1)
			},
			want: "CANCEL accepts zero or two coordinates",
		},
		{
			name: "keycombination requires two keycodes",
			call: func(s *InputManagerService) error {
				return s.KeyCombination(context.Background(), []string{"3"})
			},
			want: "at least 2 keycodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(NewInputManagerService(inputTestBinder{}))
			if !errors.Is(err, api.ErrBadParcelable) {
				t.Fatalf("err = %v, want ErrBadParcelable", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestInputManagerServiceExecuteCommand(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "tap with source and display",
			argv: []string{"touchscreen", "-d", "3", "tap", "10", "20"},
			want: []string{"touchscreen", "-d", "3", "tap", "10", "20"},
		},
		{
			name: "scroll canonicalization",
			argv: []string{"mouse", "scroll", "1", "2", "--axis", "VSCROLL,-1.25"},
			want: []string{"mouse", "scroll", "1", "2", "--axis", "VSCROLL,-1.25"},
		},
		{
			name: "unknown command falls back to raw shell command",
			argv: []string{"diagnose", "raw", "args"},
			want: []string{"diagnose", "raw", "args"},
		},
		{
			name: "empty argv still invokes shell command",
			argv: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureInputShellCommand(t, func(s *InputManagerService) error {
				return s.ExecuteCommand(context.Background(), tt.argv)
			})
			assertStringSlice(t, got.Args, tt.want)
		})
	}
}

type inputTestServiceManager struct {
	checkService func(context.Context, string) (api.Binder, error)
}

func (f inputTestServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if f.checkService == nil {
		return nil, nil
	}
	return f.checkService(ctx, name)
}

func (f inputTestServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return api.ErrUnsupported
}

func (f inputTestServiceManager) ListServices(ctx context.Context, flags api.DumpFlags) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, api.ErrUnsupported
}

func (f inputTestServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f inputTestServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	return api.ErrUnsupported
}

func (f inputTestServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return nil, api.ErrUnsupported
}

type inputTestBinder struct {
	transact func(context.Context, uint32, *api.Parcel, api.Flags) (*api.Parcel, error)
}

func (b inputTestBinder) Descriptor(ctx context.Context) (string, error) {
	return "fake.input", nil
}

func (b inputTestBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if b.transact == nil {
		return api.NewParcel(), nil
	}
	return b.transact(ctx, code, data, flags)
}

func (b inputTestBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b inputTestBinder) Close() error {
	return nil
}

type inputParsedShellCommandRequest struct {
	InFD           api.FileDescriptor
	OutFD          api.FileDescriptor
	ErrFD          api.FileDescriptor
	Args           []string
	ShellCallback  api.Binder
	ResultReceiver api.Binder
}

func parseInputShellCommandRequest(p *api.Parcel) (inputParsedShellCommandRequest, error) {
	if err := p.SetPosition(0); err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	inFD, err := p.ReadFileDescriptor()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	outFD, err := p.ReadFileDescriptor()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	errFD, err := p.ReadFileDescriptor()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	argc, err := p.ReadInt32()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	args := make([]string, 0, argc)
	for i := int32(0); i < argc; i++ {
		arg, err := p.ReadString()
		if err != nil {
			return inputParsedShellCommandRequest{}, err
		}
		args = append(args, arg)
	}
	callback, err := p.ReadStrongBinder()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	result, err := p.ReadStrongBinder()
	if err != nil {
		return inputParsedShellCommandRequest{}, err
	}
	return inputParsedShellCommandRequest{
		InFD:           inFD,
		OutFD:          outFD,
		ErrFD:          errFD,
		Args:           args,
		ShellCallback:  callback,
		ResultReceiver: result,
	}, nil
}

func captureInputShellCommand(t *testing.T, call func(*InputManagerService) error) inputParsedShellCommandRequest {
	t.Helper()

	var (
		called bool
		got    inputParsedShellCommandRequest
	)
	service := NewInputManagerService(inputTestBinder{
		transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			if code != api.ShellCommandTransaction {
				t.Fatalf("code = %d, want %d", code, api.ShellCommandTransaction)
			}
			if flags != api.FlagNone {
				t.Fatalf("flags = %#x, want %#x", flags, api.FlagNone)
			}
			var err error
			got, err = parseInputShellCommandRequest(data)
			if err != nil {
				t.Fatalf("parseInputShellCommandRequest: %v", err)
			}
			called = true
			return api.NewParcel(), nil
		},
	})

	if err := call(service); err != nil {
		t.Fatalf("call: %v", err)
	}
	if !called {
		t.Fatal("shell command transact was not invoked")
	}
	return got
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}
