package binder

import (
	"context"
	"errors"
	"testing"
)

func TestStatusCodeErrorIs(t *testing.T) {
	err := &StatusCodeError{Code: StatusUnknownTransaction}

	if !errors.Is(err, ErrUnknownTransaction) {
		t.Fatal("errors.Is(unknown transaction) = false, want true")
	}
	if errors.Is(err, ErrFailedTxn) {
		t.Fatal("errors.Is(unknown transaction, ErrFailedTxn) = true, want false")
	}
}

func TestGetInterfaceVersionAndHash(t *testing.T) {
	b := stableQueryBinder{
		version: 9,
		hash:    "hash-9",
	}

	version, err := GetInterfaceVersion(context.Background(), b)
	if err != nil {
		t.Fatalf("GetInterfaceVersion: %v", err)
	}
	if version != 9 {
		t.Fatalf("GetInterfaceVersion = %d, want 9", version)
	}

	hash, err := GetInterfaceHash(context.Background(), b)
	if err != nil {
		t.Fatalf("GetInterfaceHash: %v", err)
	}
	if hash != "hash-9" {
		t.Fatalf("GetInterfaceHash = %q, want %q", hash, "hash-9")
	}
}

func TestTransactionContextHelpers(t *testing.T) {
	ctx := WithTransactionContext(context.Background(), TransactionContext{
		Code:       FirstCallTransaction,
		Flags:      FlagOneway,
		CallingPID: 11,
		CallingUID: 22,
		Local:      true,
	})

	tx, ok := TransactionContextFromContext(ctx)
	if !ok {
		t.Fatal("TransactionContextFromContext = false, want true")
	}
	if tx.Code != FirstCallTransaction || tx.Flags != FlagOneway || tx.CallingPID != 11 || tx.CallingUID != 22 || !tx.Local {
		t.Fatalf("TransactionContext = %#v, want populated context", tx)
	}

	pid, ok := CallingPID(ctx)
	if !ok || pid != 11 {
		t.Fatalf("CallingPID = (%d, %v), want (11, true)", pid, ok)
	}
	uid, ok := CallingUID(ctx)
	if !ok || uid != 22 {
		t.Fatalf("CallingUID = (%d, %v), want (22, true)", uid, ok)
	}
}

type stableQueryBinder struct {
	version int32
	hash    string
}

func (b stableQueryBinder) Descriptor(ctx context.Context) (string, error) { return "stable", nil }

func (b stableQueryBinder) Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error) {
	reply := NewParcel()
	switch code {
	case GetInterfaceVersionTransaction:
		if err := reply.WriteInt32(b.version); err != nil {
			return nil, err
		}
	case GetInterfaceHashTransaction:
		if err := reply.WriteString(b.hash); err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnknownTransaction
	}
	if err := reply.SetPosition(0); err != nil {
		return nil, err
	}
	return reply, nil
}

func (b stableQueryBinder) WatchDeath(ctx context.Context) (Subscription, error) {
	return nil, ErrUnsupported
}
func (b stableQueryBinder) Close() error { return nil }
