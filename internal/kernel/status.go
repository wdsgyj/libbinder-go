package kernel

import (
	"errors"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func statusCodeFromError(err error) int32 {
	if err == nil {
		return 0
	}

	var statusErr *api.StatusCodeError
	if errors.As(err, &statusErr) {
		return statusErr.Code
	}

	switch {
	case errors.Is(err, api.ErrUnknownTransaction):
		return api.StatusUnknownTransaction
	case errors.Is(err, api.ErrPermissionDenied):
		return api.StatusPermissionDenied
	case errors.Is(err, api.ErrDeadObject):
		return api.StatusDeadObject
	case errors.Is(err, api.ErrBadParcelable):
		return api.StatusBadValue
	case errors.Is(err, api.ErrUnsupported):
		return api.StatusInvalidOperation
	case errors.Is(err, api.ErrFailedTxn):
		return api.StatusFailedTransaction
	default:
		return api.StatusFailedTransaction
	}
}

func statusReplyError(payload []byte) error {
	if len(payload) < 4 {
		return ErrFailedReply
	}
	code := int32(uint32(payload[0]) | uint32(payload[1])<<8 | uint32(payload[2])<<16 | uint32(payload[3])<<24)
	return &api.StatusCodeError{Code: code}
}
