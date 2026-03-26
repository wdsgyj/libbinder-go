package cmdinputproto

import (
	"context"
	"fmt"
	"syscall"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const (
	ResultReceiverDescriptor      = "com.android.internal.os.IResultReceiver"
	resultReceiverSendTransaction = api.FirstCallTransaction
)

var (
	parcelWriteInt32 = func(p *api.Parcel, v int32) error {
		return p.WriteInt32(v)
	}
	parcelWriteString = func(p *api.Parcel, v string) error {
		return p.WriteString(v)
	}
	parcelWriteStrongBinder = func(p *api.Parcel, b api.Binder) error {
		return p.WriteStrongBinder(b)
	}
	parcelWriteInterfaceToken = func(p *api.Parcel, descriptor string) error {
		return p.WriteInterfaceToken(descriptor)
	}
	parcelSetPosition = func(p *api.Parcel, pos int) error {
		return p.SetPosition(pos)
	}
)

var knownSources = map[string]struct{}{
	"touchnavigation": {},
	"touchscreen":     {},
	"joystick":        {},
	"stylus":          {},
	"touchpad":        {},
	"gamepad":         {},
	"dpad":            {},
	"mouse":           {},
	"keyboard":        {},
	"trackball":       {},
	"rotaryencoder":   {},
}

var knownCommands = map[string]struct{}{
	"text":           {},
	"keyevent":       {},
	"tap":            {},
	"swipe":          {},
	"draganddrop":    {},
	"press":          {},
	"roll":           {},
	"scroll":         {},
	"motionevent":    {},
	"keycombination": {},
}

const UsageHeader = "Usage: input [<source>] [-d DISPLAY_ID] <command> [<arg>...]\n"

type Request struct {
	InFD           api.FileDescriptor
	OutFD          api.FileDescriptor
	ErrFD          api.FileDescriptor
	Args           []string
	ShellCallback  api.Binder
	ResultReceiver api.Binder
}

type Outcome struct {
	Source            string
	DisplayID         string
	Command           string
	ResultCode        int32
	UsedShellCallback bool
	SentResult        bool
}

func EncodeRequest(p *api.Parcel, inFD int, outFD int, errFD int, args []string, callback api.Binder, result api.Binder) error {
	err := p.WriteFileDescriptor(api.NewFileDescriptor(inFD))
	if err == nil {
		err = p.WriteFileDescriptor(api.NewFileDescriptor(outFD))
	}
	if err == nil {
		err = p.WriteFileDescriptor(api.NewFileDescriptor(errFD))
	}
	if err != nil {
		return err
	}

	err = parcelWriteInt32(p, int32(len(args)))
	for _, arg := range args {
		if err == nil {
			err = parcelWriteString(p, arg)
		}
	}
	if err != nil {
		return err
	}

	err = parcelWriteStrongBinder(p, callback)
	if err == nil {
		err = parcelWriteStrongBinder(p, result)
	}
	if err != nil {
		return err
	}
	return parcelSetPosition(p, 0)
}

func DecodeRequest(p *api.Parcel) (Request, error) {
	inFD, err := p.ReadFileDescriptor()
	if err != nil {
		return Request{}, err
	}
	outFD, err := p.ReadFileDescriptor()
	if err != nil {
		return Request{}, err
	}
	errFD, err := p.ReadFileDescriptor()
	if err != nil {
		return Request{}, err
	}
	argc, err := p.ReadInt32()
	if err != nil {
		return Request{}, err
	}
	args := make([]string, 0, argc)
	for i := int32(0); i < argc; i++ {
		arg, err := p.ReadString()
		if err != nil {
			return Request{}, err
		}
		args = append(args, arg)
	}
	callback, err := p.ReadStrongBinder()
	if err != nil {
		return Request{}, err
	}
	result, err := p.ReadStrongBinder()
	if err != nil {
		return Request{}, err
	}
	return Request{
		InFD:           inFD,
		OutFD:          outFD,
		ErrFD:          errFD,
		Args:           args,
		ShellCallback:  callback,
		ResultReceiver: result,
	}, nil
}

func Execute(ctx context.Context, req Request) (Outcome, error) {
	outcome := Outcome{}
	argv := append([]string(nil), req.Args...)

	if len(argv) > 0 {
		if _, ok := knownSources[argv[0]]; ok {
			outcome.Source = argv[0]
			argv = argv[1:]
		}
	}
	if len(argv) > 0 && argv[0] == "-d" {
		if len(argv) < 2 {
			if err := writeFD(req.ErrFD, "Error: missing DISPLAY_ID after -d\n"); err != nil {
				return outcome, err
			}
			return finalize(ctx, req, outcome, -1)
		}
		outcome.DisplayID = argv[1]
		argv = argv[2:]
	}

	if len(argv) == 0 || argv[0] == "help" || argv[0] == "-h" {
		if err := writeFD(req.OutFD, UsageHeader); err != nil {
			return outcome, err
		}
		return finalize(ctx, req, outcome, -1)
	}

	outcome.Command = argv[0]
	if _, ok := knownCommands[argv[0]]; ok {
		return finalize(ctx, req, outcome, 0)
	}

	if err := writeFD(req.OutFD, fmt.Sprintf("Unknown command: %s\n", argv[0])); err != nil {
		return outcome, err
	}
	return finalize(ctx, req, outcome, -1)
}

func finalize(ctx context.Context, req Request, outcome Outcome, resultCode int32) (Outcome, error) {
	outcome.ResultCode = resultCode
	outcome.UsedShellCallback = false
	if req.ResultReceiver != nil {
		if err := sendResult(ctx, req.ResultReceiver, resultCode); err != nil {
			return outcome, err
		}
		outcome.SentResult = true
	}
	return outcome, nil
}

func sendResult(ctx context.Context, binder api.Binder, resultCode int32) error {
	data := api.NewParcel()
	err := parcelWriteInterfaceToken(data, ResultReceiverDescriptor)
	if err == nil {
		err = parcelWriteInt32(data, resultCode)
	}
	if err == nil {
		err = parcelSetPosition(data, 0)
	}
	if err != nil {
		return err
	}
	_, err = binder.Transact(ctx, resultReceiverSendTransaction, data, api.FlagOneway)
	return err
}

func writeFD(fd api.FileDescriptor, text string) error {
	if fd.FD() < 0 {
		return fmt.Errorf("%w: invalid file descriptor %d", api.ErrBadParcelable, fd.FD())
	}
	_, err := syscall.Write(fd.FD(), []byte(text))
	return err
}
