package binder

import (
	"errors"
	"fmt"
)

var (
	ErrDeadObject       = errors.New("binder: dead object")
	ErrFailedTxn        = errors.New("binder: failed transaction")
	ErrBadParcelable    = errors.New("binder: bad parcelable")
	ErrPermissionDenied = errors.New("binder: permission denied")
	ErrUnsupported      = errors.New("binder: unsupported operation")
	ErrNoService        = errors.New("binder: service not found")
)

// ExceptionCode models high-level remote exceptions returned by Binder calls.
type ExceptionCode int32

const (
	ExceptionNone ExceptionCode = 0

	ExceptionSecurity             ExceptionCode = -1
	ExceptionBadParcelable        ExceptionCode = -2
	ExceptionIllegalArgument      ExceptionCode = -3
	ExceptionNullPointer          ExceptionCode = -4
	ExceptionIllegalState         ExceptionCode = -5
	ExceptionNetworkMainThread    ExceptionCode = -6
	ExceptionUnsupportedOperation ExceptionCode = -7
	ExceptionServiceSpecific      ExceptionCode = -8
	ExceptionParcelable           ExceptionCode = -9
)

// RemoteException represents an application-level exception returned by a remote Binder service.
type RemoteException struct {
	Code    ExceptionCode
	Message string
}

func (e *RemoteException) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message == "" {
		return fmt.Sprintf("binder: remote exception %d", e.Code)
	}
	return fmt.Sprintf("binder: remote exception %d: %s", e.Code, e.Message)
}
