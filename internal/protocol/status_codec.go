package protocol

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const (
	exceptionHasNotedAppOpsReplyHeader api.ExceptionCode = -127
	exceptionHasReplyHeader            api.ExceptionCode = -128
	exceptionTransactionFailed         api.ExceptionCode = -129
)

// ReadStatus decodes the Binder status header from a reply Parcel.
//
// The format matches android::binder::Status in libbinder:
// exception code, optional message, optional stack trace header, then optional
// service-specific payload.
func ReadStatus(p *api.Parcel) (Status, error) {
	if p == nil {
		return Status{TransportErr: api.ErrBadParcelable}, api.ErrBadParcelable
	}

	code, err := p.ReadInt32()
	if err != nil {
		return Status{TransportErr: err}, err
	}

	exception := api.ExceptionCode(code)
	if exception == exceptionHasNotedAppOpsReplyHeader {
		if err := skipUnusedHeader(p); err != nil {
			return Status{TransportErr: err}, err
		}
		code, err = p.ReadInt32()
		if err != nil {
			return Status{TransportErr: err}, err
		}
		exception = api.ExceptionCode(code)
	}

	if exception == exceptionHasReplyHeader {
		if err := skipUnusedHeader(p); err != nil {
			return Status{TransportErr: err}, err
		}
		return Status{}, nil
	}

	if exception == exceptionTransactionFailed {
		return Status{TransportErr: api.ErrFailedTxn}, api.ErrFailedTxn
	}

	if exception == api.ExceptionNone {
		return Status{}, nil
	}

	msg, err := p.ReadNullableString()
	if err != nil {
		return Status{TransportErr: err}, err
	}

	stackTraceStart := p.Position()
	stackTraceAvail := p.Remaining()
	stackTraceHeaderSize, err := p.ReadInt32()
	if err != nil {
		return Status{TransportErr: err}, err
	}
	if stackTraceHeaderSize < 0 || int(stackTraceHeaderSize) > stackTraceAvail {
		err := fmt.Errorf("%w: invalid remote stack trace header size %d", api.ErrBadParcelable, stackTraceHeaderSize)
		return Status{TransportErr: err}, err
	}
	if stackTraceHeaderSize != 0 {
		if err := p.SetPosition(stackTraceStart + int(stackTraceHeaderSize)); err != nil {
			return Status{TransportErr: err}, err
		}
	}

	remote := &RemoteException{Code: exception}
	if msg != nil {
		remote.Message = *msg
	}

	switch exception {
	case api.ExceptionServiceSpecific:
		serviceCode, err := p.ReadInt32()
		if err != nil {
			return Status{TransportErr: err}, err
		}
		remote.ServiceCode = serviceCode
	case api.ExceptionParcelable:
		if err := skipUnusedHeader(p); err != nil {
			return Status{TransportErr: err}, err
		}
	}

	return Status{Remote: remote}, nil
}

// WriteStatus encodes the Binder status header into a reply Parcel.
func WriteStatus(p *api.Parcel, st Status) error {
	if p == nil {
		return api.ErrBadParcelable
	}
	if st.TransportErr != nil {
		return st.TransportErr
	}
	if st.Remote == nil {
		return p.WriteInt32(int32(api.ExceptionNone))
	}
	if st.Remote.Code == exceptionTransactionFailed {
		return api.ErrFailedTxn
	}

	if err := p.WriteInt32(int32(st.Remote.Code)); err != nil {
		return err
	}
	if err := p.WriteString(st.Remote.Message); err != nil {
		return err
	}
	if err := p.WriteInt32(0); err != nil {
		return err
	}

	switch st.Remote.Code {
	case api.ExceptionServiceSpecific:
		return p.WriteInt32(st.Remote.ServiceCode)
	case api.ExceptionParcelable:
		return p.WriteInt32(0)
	default:
		return nil
	}
}

func skipUnusedHeader(p *api.Parcel) error {
	headerStart := p.Position()
	headerAvail := p.Remaining()

	headerSize, err := p.ReadInt32()
	if err != nil {
		return err
	}
	if headerSize < 0 || int(headerSize) > headerAvail {
		return fmt.Errorf("%w: invalid status header size %d", api.ErrBadParcelable, headerSize)
	}
	return p.SetPosition(headerStart + int(headerSize))
}
