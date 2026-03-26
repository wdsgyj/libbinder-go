package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	libbinder "github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

const InputServiceName = "input"

const (
	InputDisplayInvalid = -1
	InputDisplayDefault = 0
)

type InputSource string

const (
	InputSourceKeyboard        InputSource = "keyboard"
	InputSourceDPad            InputSource = "dpad"
	InputSourceGamepad         InputSource = "gamepad"
	InputSourceTouchscreen     InputSource = "touchscreen"
	InputSourceMouse           InputSource = "mouse"
	InputSourceStylus          InputSource = "stylus"
	InputSourceTrackball       InputSource = "trackball"
	InputSourceTouchpad        InputSource = "touchpad"
	InputSourceTouchNavigation InputSource = "touchnavigation"
	InputSourceJoystick        InputSource = "joystick"
	InputSourceRotaryEncoder   InputSource = "rotaryencoder"
)

type InputMotionAction string

const (
	InputMotionActionDown   InputMotionAction = "DOWN"
	InputMotionActionUp     InputMotionAction = "UP"
	InputMotionActionMove   InputMotionAction = "MOVE"
	InputMotionActionCancel InputMotionAction = "CANCEL"
)

type InputAxisName string

const (
	InputAxisScroll  InputAxisName = "SCROLL"
	InputAxisHScroll InputAxisName = "HSCROLL"
	InputAxisVScroll InputAxisName = "VSCROLL"
)

type InputAxisValue struct {
	Axis  InputAxisName
	Value float64
}

type InputCommandOptions struct {
	Source    InputSource
	DisplayID *int
}

type InputKeyEventOptions struct {
	LongPress  bool
	Async      bool
	DoubleTap  bool
	DelayMS    *int64
	DurationMS *int64
}

type InputShellIO struct {
	InFD  api.FileDescriptor
	OutFD api.FileDescriptor
	ErrFD api.FileDescriptor
}

// InputShellCommandRequest describes a shell-command request sent to Android's
// input service over SHELL_COMMAND_TRANSACTION.
type InputShellCommandRequest struct {
	InFD  api.FileDescriptor
	OutFD api.FileDescriptor
	ErrFD api.FileDescriptor
	Args  []string
}

// InputShellCommandBuildError reports that the request Parcel could not be
// encoded before any Binder transaction was attempted.
type InputShellCommandBuildError struct {
	Err error
}

func (e *InputShellCommandBuildError) Error() string {
	if e == nil || e.Err == nil {
		return api.ErrBadParcelable.Error()
	}
	return e.Err.Error()
}

func (e *InputShellCommandBuildError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type InputManagerService struct {
	service api.Binder
	io      InputShellIO
	opts    InputCommandOptions
}

var (
	knownInputSources = map[InputSource]struct{}{
		InputSourceKeyboard:        {},
		InputSourceDPad:            {},
		InputSourceGamepad:         {},
		InputSourceTouchscreen:     {},
		InputSourceMouse:           {},
		InputSourceStylus:          {},
		InputSourceTrackball:       {},
		InputSourceTouchpad:        {},
		InputSourceTouchNavigation: {},
		InputSourceJoystick:        {},
		InputSourceRotaryEncoder:   {},
	}
	pointerInputSources = map[InputSource]struct{}{
		InputSourceTouchscreen:     {},
		InputSourceMouse:           {},
		InputSourceStylus:          {},
		InputSourceTouchpad:        {},
		InputSourceTouchNavigation: {},
	}
	supportedScrollAxes = map[InputAxisName]struct{}{
		InputAxisScroll:  {},
		InputAxisHScroll: {},
		InputAxisVScroll: {},
	}
)

func DefaultInputShellIO() InputShellIO {
	return InputShellIO{
		InFD:  api.NewFileDescriptor(int(os.Stdin.Fd())),
		OutFD: api.NewFileDescriptor(int(os.Stdout.Fd())),
		ErrFD: api.NewFileDescriptor(int(os.Stderr.Fd())),
	}
}

func KnownInputSources() []InputSource {
	return []InputSource{
		InputSourceKeyboard,
		InputSourceDPad,
		InputSourceGamepad,
		InputSourceTouchscreen,
		InputSourceMouse,
		InputSourceStylus,
		InputSourceTrackball,
		InputSourceTouchpad,
		InputSourceTouchNavigation,
		InputSourceJoystick,
		InputSourceRotaryEncoder,
	}
}

// LookupInputService resolves the Android input service Binder from the provided
// ServiceManager.
func LookupInputService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	if sm == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}

	service, err := sm.CheckService(ctx, InputServiceName)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, api.ErrNoService
	}
	return service, nil
}

// LookupInputManagerService resolves the Android input service and wraps it in
// an InputManagerService helper using the current process stdio.
func LookupInputManagerService(ctx context.Context, sm api.ServiceManager) (*InputManagerService, error) {
	service, err := LookupInputService(ctx, sm)
	if err != nil {
		return nil, err
	}
	return NewInputManagerService(service), nil
}

func NewInputManagerService(service api.Binder) *InputManagerService {
	return &InputManagerService{
		service: service,
		io:      DefaultInputShellIO(),
	}
}

func (s *InputManagerService) WithShellIO(io InputShellIO) *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.io = io
	return &clone
}

func (s *InputManagerService) WithCommandOptions(opts InputCommandOptions) *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.opts = opts
	return &clone
}

func (s *InputManagerService) WithSource(source InputSource) *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.opts.Source = source
	return &clone
}

func (s *InputManagerService) WithoutSource() *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.opts.Source = ""
	return &clone
}

func (s *InputManagerService) WithDisplayID(displayID int) *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.opts.DisplayID = libbinder.IntPtr(displayID)
	return &clone
}

func (s *InputManagerService) WithoutDisplayID() *InputManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.opts.DisplayID = nil
	return &clone
}

func (s *InputManagerService) CommandOptions() InputCommandOptions {
	if s == nil {
		return InputCommandOptions{}
	}
	return s.opts
}

func (s *InputManagerService) Binder() api.Binder {
	if s == nil {
		return nil
	}
	return s.service
}

func (s *InputManagerService) Command(ctx context.Context, args ...string) error {
	if s == nil || s.service == nil {
		return api.ErrUnsupported
	}
	return TransactInputShellCommand(ctx, s.service, InputShellCommandRequest{
		InFD:  s.io.InFD,
		OutFD: s.io.OutFD,
		ErrFD: s.io.ErrFD,
		Args:  args,
	})
}

func (s *InputManagerService) ExecuteCommand(ctx context.Context, argv []string) error {
	parsed, ok, err := parseInputCommand(argv)
	if err != nil {
		return err
	}
	if !ok {
		return s.Command(ctx, argv...)
	}
	scoped := s.WithCommandOptions(parsed.opts)
	switch parsed.command {
	case "text":
		return scoped.Text(ctx, parsed.text)
	case "keyevent":
		return scoped.KeyEventWithOptions(ctx, parsed.keyEventOpts, parsed.keyCodes...)
	case "tap":
		return scoped.Tap(ctx, parsed.xy[0], parsed.xy[1])
	case "swipe":
		return scoped.Swipe(ctx, parsed.xyxy[0], parsed.xyxy[1], parsed.xyxy[2], parsed.xyxy[3], parsed.optionalDuration...)
	case "draganddrop":
		return scoped.DragAndDrop(ctx, parsed.xyxy[0], parsed.xyxy[1], parsed.xyxy[2], parsed.xyxy[3], parsed.optionalDuration...)
	case "press":
		return scoped.Press(ctx)
	case "roll":
		return scoped.Roll(ctx, parsed.xy[0], parsed.xy[1])
	case "scroll":
		return scoped.Scroll(ctx, parsed.axisValues, parsed.optionalCoords...)
	case "motionevent":
		return scoped.MotionEvent(ctx, parsed.action, parsed.optionalCoords...)
	case "keycombination":
		return scoped.KeyCombination(ctx, parsed.keyCodes, parsed.optionalDuration...)
	default:
		return s.Command(ctx, argv...)
	}
}

func (s *InputManagerService) Text(ctx context.Context, text string) error {
	args, err := commandArgs(s.CommandOptions(), "text")
	if err != nil {
		return err
	}
	return s.Command(ctx, append(args, text)...)
}

func (s *InputManagerService) TextWithOptions(ctx context.Context, opts InputCommandOptions, text string) error {
	return s.WithCommandOptions(opts).Text(ctx, text)
}

func (s *InputManagerService) KeyEvent(ctx context.Context, keyCodes ...string) error {
	return s.KeyEventWithOptions(ctx, InputKeyEventOptions{}, keyCodes...)
}

func (s *InputManagerService) KeyEventWithOptions(ctx context.Context, keyOpts InputKeyEventOptions, keyCodes ...string) error {
	if len(keyCodes) == 0 {
		return fmt.Errorf("%w: keyevent requires at least 1 keycode", api.ErrBadParcelable)
	}
	if keyOpts.LongPress && keyOpts.DurationMS != nil {
		return fmt.Errorf("%w: keyevent args should only contain either durationMs or longPress", api.ErrBadParcelable)
	}
	args, err := commandArgs(s.CommandOptions(), "keyevent")
	if err != nil {
		return err
	}
	if keyOpts.LongPress {
		args = append(args, "--longpress")
	}
	if keyOpts.DurationMS != nil {
		args = append(args, "--duration", strconv.FormatInt(*keyOpts.DurationMS, 10))
	}
	if keyOpts.DoubleTap {
		args = append(args, "--doubletap")
	}
	if keyOpts.Async {
		args = append(args, "--async")
	}
	if keyOpts.DelayMS != nil {
		args = append(args, "--delay", strconv.FormatInt(*keyOpts.DelayMS, 10))
	}
	args = append(args, keyCodes...)
	return s.Command(ctx, args...)
}

func (s *InputManagerService) Tap(ctx context.Context, x, y float64) error {
	args, err := commandArgs(s.CommandOptions(), "tap", formatFloatArg(x), formatFloatArg(y))
	if err != nil {
		return err
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) TapWithOptions(ctx context.Context, opts InputCommandOptions, x, y float64) error {
	return s.WithCommandOptions(opts).Tap(ctx, x, y)
}

func (s *InputManagerService) Swipe(ctx context.Context, x1, y1, x2, y2 float64, durationMS ...int64) error {
	args, err := gestureCommandArgs(s.CommandOptions(), "swipe", x1, y1, x2, y2, durationMS...)
	if err != nil {
		return err
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) SwipeWithOptions(ctx context.Context, opts InputCommandOptions, x1, y1, x2, y2 float64, durationMS ...int64) error {
	return s.WithCommandOptions(opts).Swipe(ctx, x1, y1, x2, y2, durationMS...)
}

func (s *InputManagerService) DragAndDrop(ctx context.Context, x1, y1, x2, y2 float64, durationMS ...int64) error {
	args, err := gestureCommandArgs(s.CommandOptions(), "draganddrop", x1, y1, x2, y2, durationMS...)
	if err != nil {
		return err
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) DragAndDropWithOptions(ctx context.Context, opts InputCommandOptions, x1, y1, x2, y2 float64, durationMS ...int64) error {
	return s.WithCommandOptions(opts).DragAndDrop(ctx, x1, y1, x2, y2, durationMS...)
}

func (s *InputManagerService) Press(ctx context.Context) error {
	args, err := commandArgs(s.CommandOptions(), "press")
	if err != nil {
		return err
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) PressWithOptions(ctx context.Context, opts InputCommandOptions) error {
	return s.WithCommandOptions(opts).Press(ctx)
}

func (s *InputManagerService) Roll(ctx context.Context, dx, dy float64) error {
	args, err := commandArgs(s.CommandOptions(), "roll", formatFloatArg(dx), formatFloatArg(dy))
	if err != nil {
		return err
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) RollWithOptions(ctx context.Context, opts InputCommandOptions, dx, dy float64) error {
	return s.WithCommandOptions(opts).Roll(ctx, dx, dy)
}

func (s *InputManagerService) Scroll(ctx context.Context, axisValues []InputAxisValue, coordinates ...float64) error {
	opts := s.CommandOptions()
	if len(coordinates) != 0 && len(coordinates) != 2 {
		return fmt.Errorf("%w: scroll coordinates must be omitted or provide x and y", api.ErrBadParcelable)
	}
	for _, axis := range axisValues {
		if _, ok := supportedScrollAxes[axis.Axis]; !ok {
			return fmt.Errorf("%w: unsupported axis: %s", api.ErrBadParcelable, axis.Axis)
		}
	}
	if len(coordinates) == 0 {
		if _, ok := pointerInputSources[opts.Source]; ok {
			return fmt.Errorf("%w: scroll requires x and y for pointer-based sources", api.ErrBadParcelable)
		}
	}
	args, err := commandArgs(opts, "scroll")
	if err != nil {
		return err
	}
	if len(coordinates) == 2 {
		args = append(args, formatFloatArg(coordinates[0]), formatFloatArg(coordinates[1]))
	}
	for _, axis := range axisValues {
		args = append(args, "--axis", string(axis.Axis)+","+formatFloatArg(axis.Value))
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) ScrollWithOptions(ctx context.Context, opts InputCommandOptions, axisValues []InputAxisValue, coordinates ...float64) error {
	return s.WithCommandOptions(opts).Scroll(ctx, axisValues, coordinates...)
}

func (s *InputManagerService) MotionEvent(ctx context.Context, action InputMotionAction, coordinates ...float64) error {
	if !isKnownMotionAction(action) {
		return fmt.Errorf("%w: unknown action: %s", api.ErrBadParcelable, action)
	}
	switch action {
	case InputMotionActionCancel:
		if len(coordinates) != 0 && len(coordinates) != 2 {
			return fmt.Errorf("%w: CANCEL accepts zero or two coordinates", api.ErrBadParcelable)
		}
	default:
		if len(coordinates) != 2 {
			return fmt.Errorf("%w: %s requires x and y", api.ErrBadParcelable, action)
		}
	}
	args, err := commandArgs(s.CommandOptions(), "motionevent", string(action))
	if err != nil {
		return err
	}
	if len(coordinates) == 2 {
		args = append(args, formatFloatArg(coordinates[0]), formatFloatArg(coordinates[1]))
	}
	return s.Command(ctx, args...)
}

func (s *InputManagerService) MotionEventWithOptions(ctx context.Context, opts InputCommandOptions, action InputMotionAction, coordinates ...float64) error {
	return s.WithCommandOptions(opts).MotionEvent(ctx, action, coordinates...)
}

func (s *InputManagerService) KeyCombination(ctx context.Context, keyCodes []string, durationMS ...int64) error {
	if len(durationMS) > 1 {
		return fmt.Errorf("%w: keycombination accepts at most one duration", api.ErrBadParcelable)
	}
	if len(keyCodes) < 2 {
		return fmt.Errorf("%w: keycombination requires at least 2 keycodes", api.ErrBadParcelable)
	}
	args, err := commandArgs(s.CommandOptions(), "keycombination")
	if err != nil {
		return err
	}
	if len(durationMS) == 1 {
		args = append(args, "-t", strconv.FormatInt(durationMS[0], 10))
	}
	args = append(args, keyCodes...)
	return s.Command(ctx, args...)
}

func (s *InputManagerService) KeyCombinationWithOptions(ctx context.Context, opts InputCommandOptions, keyCodes []string, durationMS ...int64) error {
	return s.WithCommandOptions(opts).KeyCombination(ctx, keyCodes, durationMS...)
}

// WriteInputShellCommandRequest encodes the shell-command request expected by
// the input service. The callback and result receiver slots are intentionally
// sent as nil to match the current synchronous input shell path.
func WriteInputShellCommandRequest(p *api.Parcel, req InputShellCommandRequest) error {
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
	if err := p.WriteStrongBinder(nil); err != nil {
		return err
	}
	if err := p.WriteStrongBinder(nil); err != nil {
		return err
	}
	return p.SetPosition(0)
}

// BuildInputShellCommandParcel constructs the request Parcel for an input shell
// command invocation.
func BuildInputShellCommandParcel(req InputShellCommandRequest) (*api.Parcel, error) {
	data := api.NewParcel()
	if err := WriteInputShellCommandRequest(data, req); err != nil {
		return nil, err
	}
	return data, nil
}

// TransactInputShellCommand sends a shell-command request to an already
// resolved input service Binder.
func TransactInputShellCommand(ctx context.Context, service api.Binder, req InputShellCommandRequest) error {
	if service == nil {
		return api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}

	data, err := BuildInputShellCommandParcel(req)
	if err != nil {
		return &InputShellCommandBuildError{Err: err}
	}
	_, err = service.Transact(ctx, api.ShellCommandTransaction, data, api.FlagNone)
	return err
}

// ExecuteInputShellCommand resolves the input service and sends the
// shell-command request in one step.
func ExecuteInputShellCommand(ctx context.Context, sm api.ServiceManager, req InputShellCommandRequest) error {
	service, err := LookupInputService(ctx, sm)
	if err != nil {
		return err
	}
	return TransactInputShellCommand(ctx, service, req)
}

type parsedInputCommand struct {
	command          string
	opts             InputCommandOptions
	text             string
	keyEventOpts     InputKeyEventOptions
	keyCodes         []string
	xy               [2]float64
	xyxy             [4]float64
	axisValues       []InputAxisValue
	action           InputMotionAction
	optionalDuration []int64
	optionalCoords   []float64
}

func parseInputCommand(argv []string) (parsedInputCommand, bool, error) {
	var out parsedInputCommand
	args := append([]string(nil), argv...)
	if len(args) == 0 {
		return out, false, nil
	}
	if isKnownInputSource(InputSource(args[0])) {
		out.opts.Source = InputSource(args[0])
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "-d" {
		if len(args) < 2 {
			return out, true, fmt.Errorf("%w: missing DISPLAY_ID after -d", api.ErrBadParcelable)
		}
		displayID, err := parseDisplayID(args[1])
		if err != nil {
			return out, true, err
		}
		out.opts.DisplayID = libbinder.IntPtr(displayID)
		args = args[2:]
	}
	if len(args) == 0 {
		return out, false, nil
	}
	out.command = args[0]
	args = args[1:]

	switch out.command {
	case "text":
		if len(args) != 1 {
			return out, true, fmt.Errorf("%w: text requires exactly 1 argument", api.ErrBadParcelable)
		}
		out.text = args[0]
	case "keyevent":
		for len(args) > 0 && strings.HasPrefix(args[0], "--") {
			switch args[0] {
			case "--longpress":
				out.keyEventOpts.LongPress = true
				args = args[1:]
			case "--async":
				out.keyEventOpts.Async = true
				args = args[1:]
			case "--doubletap":
				out.keyEventOpts.DoubleTap = true
				args = args[1:]
			case "--delay":
				if len(args) < 2 {
					return out, true, fmt.Errorf("%w: --delay requires a duration", api.ErrBadParcelable)
				}
				v, err := strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return out, true, err
				}
				out.keyEventOpts.DelayMS = libbinder.Int64Ptr(v)
				args = args[2:]
			case "--duration":
				if len(args) < 2 {
					return out, true, fmt.Errorf("%w: --duration requires a duration", api.ErrBadParcelable)
				}
				v, err := strconv.ParseInt(args[1], 10, 64)
				if err != nil {
					return out, true, err
				}
				out.keyEventOpts.DurationMS = libbinder.Int64Ptr(v)
				args = args[2:]
			default:
				return out, true, fmt.Errorf("%w: unsupported option: %s", api.ErrBadParcelable, args[0])
			}
		}
		if len(args) == 0 {
			return out, true, fmt.Errorf("%w: keyevent requires at least 1 keycode", api.ErrBadParcelable)
		}
		out.keyCodes = append(out.keyCodes, args...)
	case "tap":
		coords, err := parseFixedFloats(args, 2)
		if err != nil {
			return out, true, err
		}
		copy(out.xy[:], coords)
	case "swipe", "draganddrop":
		if len(args) != 4 && len(args) != 5 {
			return out, true, fmt.Errorf("%w: %s requires 4 coordinates and optional duration", api.ErrBadParcelable, out.command)
		}
		coords, err := parseFixedFloats(args[:4], 4)
		if err != nil {
			return out, true, err
		}
		copy(out.xyxy[:], coords)
		if len(args) == 5 {
			duration, err := strconv.ParseInt(args[4], 10, 64)
			if err != nil {
				return out, true, err
			}
			out.optionalDuration = []int64{duration}
		}
	case "press":
		if len(args) != 0 {
			return out, true, fmt.Errorf("%w: press takes no arguments", api.ErrBadParcelable)
		}
	case "roll":
		coords, err := parseFixedFloats(args, 2)
		if err != nil {
			return out, true, err
		}
		copy(out.xy[:], coords)
	case "scroll":
		axisValues, coordinates, err := parseScrollArgs(out.opts.Source, args)
		if err != nil {
			return out, true, err
		}
		out.axisValues = axisValues
		out.optionalCoords = coordinates
	case "motionevent":
		if len(args) == 0 {
			return out, true, fmt.Errorf("%w: motionevent requires an action", api.ErrBadParcelable)
		}
		out.action = InputMotionAction(strings.ToUpper(args[0]))
		if !isKnownMotionAction(out.action) {
			return out, true, fmt.Errorf("%w: unknown action: %s", api.ErrBadParcelable, args[0])
		}
		coords, err := parseOptionalCoordinates(out.action, args[1:])
		if err != nil {
			return out, true, err
		}
		out.optionalCoords = coords
	case "keycombination":
		if len(args) == 0 {
			return out, true, fmt.Errorf("%w: keycombination requires at least 2 keycodes", api.ErrBadParcelable)
		}
		if args[0] == "-t" {
			if len(args) < 2 {
				return out, true, fmt.Errorf("%w: -t requires a duration", api.ErrBadParcelable)
			}
			duration, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return out, true, err
			}
			out.optionalDuration = []int64{duration}
			args = args[2:]
		}
		if len(args) < 2 {
			return out, true, fmt.Errorf("%w: keycombination requires at least 2 keycodes", api.ErrBadParcelable)
		}
		out.keyCodes = append(out.keyCodes, args...)
	default:
		return out, false, nil
	}
	return out, true, nil
}

func parseDisplayID(arg string) (int, error) {
	switch {
	case strings.EqualFold(arg, "INVALID_DISPLAY"):
		return InputDisplayInvalid, nil
	case strings.EqualFold(arg, "DEFAULT_DISPLAY"):
		return InputDisplayDefault, nil
	default:
		displayID, err := strconv.Atoi(arg)
		if err != nil {
			return 0, fmt.Errorf("%w: invalid display ID %q", api.ErrBadParcelable, arg)
		}
		if displayID < 0 {
			if displayID == InputDisplayInvalid {
				return InputDisplayInvalid, nil
			}
			return 0, fmt.Errorf("%w: invalid display ID %q", api.ErrBadParcelable, arg)
		}
		return displayID, nil
	}
}

func parseFixedFloats(args []string, count int) ([]float64, error) {
	if len(args) != count {
		return nil, fmt.Errorf("%w: expected %d numeric arguments", api.ErrBadParcelable, count)
	}
	out := make([]float64, 0, count)
	for _, arg := range args {
		value, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func parseOptionalCoordinates(action InputMotionAction, args []string) ([]float64, error) {
	switch action {
	case InputMotionActionCancel:
		switch len(args) {
		case 0:
			return nil, nil
		case 2:
			return parseFixedFloats(args, 2)
		default:
			return nil, fmt.Errorf("%w: CANCEL accepts zero or two coordinates", api.ErrBadParcelable)
		}
	default:
		return parseFixedFloats(args, 2)
	}
}

func parseScrollArgs(source InputSource, args []string) ([]InputAxisValue, []float64, error) {
	var coords []float64
	if len(args) >= 2 && !strings.HasPrefix(args[0], "-") && !strings.HasPrefix(args[1], "-") {
		parsed, err := parseFixedFloats(args[:2], 2)
		if err != nil {
			return nil, nil, err
		}
		coords = parsed
		args = args[2:]
	}
	axisValues := make([]InputAxisValue, 0)
	for len(args) > 0 {
		if args[0] != "--axis" {
			return nil, nil, fmt.Errorf("%w: unsupported option: %s", api.ErrBadParcelable, args[0])
		}
		if len(args) < 2 {
			return nil, nil, fmt.Errorf("%w: --axis requires AXIS,VALUE", api.ErrBadParcelable)
		}
		parts := strings.Split(args[1], ",")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("%w: invalid --axis option value: %s", api.ErrBadParcelable, args[1])
		}
		axis := InputAxisName(strings.ToUpper(parts[0]))
		if _, ok := supportedScrollAxes[axis]; !ok {
			return nil, nil, fmt.Errorf("%w: unsupported axis: %s", api.ErrBadParcelable, axis)
		}
		value, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, nil, err
		}
		axisValues = append(axisValues, InputAxisValue{Axis: axis, Value: value})
		args = args[2:]
	}
	if len(coords) == 0 {
		if _, ok := pointerInputSources[source]; ok {
			return nil, nil, fmt.Errorf("%w: scroll requires x and y for pointer-based sources", api.ErrBadParcelable)
		}
	}
	return axisValues, coords, nil
}

func commandArgs(opts InputCommandOptions, command string, extra ...string) ([]string, error) {
	args := make([]string, 0, 4+len(extra))
	if opts.Source != "" {
		if !isKnownInputSource(opts.Source) {
			return nil, fmt.Errorf("%w: unsupported source: %s", api.ErrBadParcelable, opts.Source)
		}
		args = append(args, string(opts.Source))
	}
	if opts.DisplayID != nil {
		args = append(args, "-d", formatDisplayID(*opts.DisplayID))
	}
	args = append(args, command)
	args = append(args, extra...)
	return args, nil
}

func gestureCommandArgs(opts InputCommandOptions, command string, x1, y1, x2, y2 float64, durationMS ...int64) ([]string, error) {
	if len(durationMS) > 1 {
		return nil, fmt.Errorf("%w: %s accepts at most one duration", api.ErrBadParcelable, command)
	}
	args, err := commandArgs(opts, command,
		formatFloatArg(x1),
		formatFloatArg(y1),
		formatFloatArg(x2),
		formatFloatArg(y2),
	)
	if err != nil {
		return nil, err
	}
	if len(durationMS) == 1 {
		args = append(args, strconv.FormatInt(durationMS[0], 10))
	}
	return args, nil
}

func formatDisplayID(displayID int) string {
	switch displayID {
	case InputDisplayInvalid:
		return "INVALID_DISPLAY"
	case InputDisplayDefault:
		return "DEFAULT_DISPLAY"
	default:
		return strconv.Itoa(displayID)
	}
}

func formatFloatArg(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func isKnownInputSource(source InputSource) bool {
	_, ok := knownInputSources[source]
	return ok
}

func isKnownMotionAction(action InputMotionAction) bool {
	switch action {
	case InputMotionActionDown, InputMotionActionUp, InputMotionActionMove, InputMotionActionCancel:
		return true
	default:
		return false
	}
}
