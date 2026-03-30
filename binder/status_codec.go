package binder

import "fmt"

const (
	exceptionHasNotedAppOpsReplyHeader ExceptionCode = -127
	exceptionHasReplyHeader            ExceptionCode = -128
	exceptionTransactionFailed         ExceptionCode = -129
)

// ReadException consumes the standard Binder reply status header.
//
// It mirrors Java's Parcel.readException(): nil means success, while remote
// exceptions are surfaced as *RemoteException values.
func ReadException(p *Parcel) error {
	if p == nil {
		return ErrBadParcelable
	}

	code, err := p.ReadInt32()
	if err != nil {
		return err
	}

	exception := ExceptionCode(code)
	if exception == exceptionHasNotedAppOpsReplyHeader {
		if err := skipExceptionHeader(p); err != nil {
			return err
		}
		code, err = p.ReadInt32()
		if err != nil {
			return err
		}
		exception = ExceptionCode(code)
	}

	if exception == exceptionHasReplyHeader {
		return skipExceptionHeader(p)
	}
	if exception == exceptionTransactionFailed {
		return ErrFailedTxn
	}
	if exception == ExceptionNone {
		return nil
	}

	msg, err := p.ReadNullableString()
	if err != nil {
		return err
	}

	stackTraceStart := p.Position()
	stackTraceAvail := p.Remaining()
	stackTraceHeaderSize, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if stackTraceHeaderSize < 0 || int(stackTraceHeaderSize) > stackTraceAvail {
		return fmt.Errorf("%w: invalid remote stack trace header size %d", ErrBadParcelable, stackTraceHeaderSize)
	}
	if stackTraceHeaderSize != 0 {
		if err := p.SetPosition(stackTraceStart + int(stackTraceHeaderSize)); err != nil {
			return err
		}
	}

	remote := &RemoteException{Code: exception}
	if msg != nil {
		remote.Message = *msg
	}

	switch exception {
	case ExceptionServiceSpecific:
		serviceCode, err := p.ReadInt32()
		if err != nil {
			return err
		}
		remote.ServiceCode = serviceCode
	case ExceptionParcelable:
		if err := skipExceptionHeader(p); err != nil {
			return err
		}
	}

	return remote
}

// WriteNoException writes the successful Binder reply status header.
func WriteNoException(p *Parcel) error {
	if p == nil {
		return ErrBadParcelable
	}
	return p.WriteInt32(int32(ExceptionNone))
}

func skipExceptionHeader(p *Parcel) error {
	headerStart := p.Position()
	headerAvail := p.Remaining()

	headerSize, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if headerSize < 0 || int(headerSize) > headerAvail {
		return fmt.Errorf("%w: invalid status header size %d", ErrBadParcelable, headerSize)
	}
	return p.SetPosition(headerStart + int(headerSize))
}
