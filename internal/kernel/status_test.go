package kernel

import (
	"errors"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestStatusCodeFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int32
	}{
		{name: "unknown transaction", err: api.ErrUnknownTransaction, want: api.StatusUnknownTransaction},
		{name: "permission denied", err: api.ErrPermissionDenied, want: api.StatusPermissionDenied},
		{name: "dead object", err: api.ErrDeadObject, want: api.StatusDeadObject},
		{name: "bad parcelable", err: api.ErrBadParcelable, want: api.StatusBadValue},
		{name: "unsupported", err: api.ErrUnsupported, want: api.StatusInvalidOperation},
		{name: "failed transaction", err: api.ErrFailedTxn, want: api.StatusFailedTransaction},
		{name: "status error passthrough", err: &api.StatusCodeError{Code: 1234}, want: 1234},
		{name: "wrapped unknown transaction", err: errors.Join(api.ErrUnknownTransaction, errors.New("extra")), want: api.StatusUnknownTransaction},
		{name: "default", err: errors.New("boom"), want: api.StatusFailedTransaction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCodeFromError(tt.err); got != tt.want {
				t.Fatalf("statusCodeFromError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestStatusReplyErrorReturnsStatusCodeError(t *testing.T) {
	err := statusReplyError([]byte{0xb6, 0xff, 0xff, 0xff}) // -74

	var statusErr *api.StatusCodeError
	if !errors.As(err, &statusErr) {
		t.Fatalf("statusReplyError type = %T, want *binder.StatusCodeError", err)
	}
	if statusErr.Code != api.StatusUnknownTransaction {
		t.Fatalf("statusReplyError code = %d, want %d", statusErr.Code, api.StatusUnknownTransaction)
	}
	if !errors.Is(err, api.ErrUnknownTransaction) {
		t.Fatalf("errors.Is(statusReplyError, ErrUnknownTransaction) = false, want true")
	}
}
