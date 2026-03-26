package binder

import (
	"errors"
	"fmt"
)

var (
	ErrDeadObject         = errors.New("binder: dead object")
	ErrFailedTxn          = errors.New("binder: failed transaction")
	ErrBadType            = errors.New("binder: bad type")
	ErrBadParcelable      = errors.New("binder: bad parcelable")
	ErrPermissionDenied   = errors.New("binder: permission denied")
	ErrUnsupported        = errors.New("binder: unsupported operation")
	ErrNoService          = errors.New("binder: service not found")
	ErrClosed             = errors.New("binder: binder is closed")
	ErrUnknownTransaction = errors.New("binder: unknown transaction")
)

const (
	StatusUnknownError      int32 = -2147483648
	StatusBadType           int32 = StatusUnknownError + 1
	StatusFailedTransaction int32 = StatusUnknownError + 2
	StatusFdsNotAllowed     int32 = StatusUnknownError + 7
	StatusUnexpectedNull    int32 = StatusUnknownError + 8

	StatusPermissionDenied   int32 = -1
	StatusNameNotFound       int32 = -2
	StatusNoMemory           int32 = -12
	StatusBadValue           int32 = -22
	StatusDeadObject         int32 = -32
	StatusInvalidOperation   int32 = -38
	StatusUnknownTransaction int32 = -74
)

// StatusCodeError carries a raw Binder transport status code while still
// participating in the public error model via errors.Is.
type StatusCodeError struct {
	Code int32
}

func (e *StatusCodeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("binder: transport status %d", e.Code)
}

func (e *StatusCodeError) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	switch target {
	case ErrDeadObject:
		return e.Code == StatusDeadObject
	case ErrFailedTxn:
		return e.Code == StatusFailedTransaction
	case ErrBadType:
		return e.Code == StatusBadType
	case ErrPermissionDenied:
		return e.Code == StatusPermissionDenied
	case ErrUnknownTransaction:
		return e.Code == StatusUnknownTransaction
	default:
		return false
	}
}

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
