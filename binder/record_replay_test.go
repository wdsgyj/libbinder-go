package binder

import (
	"context"
	"errors"
	"testing"
)

func TestRecordingBinderAndReplayBinder(t *testing.T) {
	recorder := NewTransactionRecorder()
	target := echoBinder{}
	recorded := NewRecordingBinder(target, recorder)

	req := NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("req.WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("req.SetPosition: %v", err)
	}

	reply, err := recorded.Transact(context.Background(), FirstCallTransaction, req, FlagNone)
	if err != nil {
		t.Fatalf("Transact(recorded): %v", err)
	}
	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("reply = %q, want echo:ping", got)
	}

	records := recorder.Records()
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Descriptor != "echo" {
		t.Fatalf("Descriptor = %q, want echo", records[0].Descriptor)
	}

	replay := NewReplayBinder(records)
	req = NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("replay req.WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("replay req.SetPosition: %v", err)
	}
	reply, err = replay.Transact(context.Background(), FirstCallTransaction, req, FlagNone)
	if err != nil {
		t.Fatalf("Transact(replay): %v", err)
	}
	got, err = reply.ReadString()
	if err != nil {
		t.Fatalf("replay reply.ReadString: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("replay reply = %q, want echo:ping", got)
	}
}

func TestReplayBinderMismatch(t *testing.T) {
	replay := NewReplayBinder([]TransactionRecord{{
		Descriptor:   "echo",
		Code:         FirstCallTransaction,
		Flags:        FlagNone,
		RequestBytes: []byte{1, 2, 3, 4},
	}})

	req := NewParcel()
	if err := req.WriteInt32(9); err != nil {
		t.Fatalf("req.WriteInt32: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("req.SetPosition: %v", err)
	}

	if _, err := replay.Transact(context.Background(), FirstCallTransaction, req, FlagNone); !errors.Is(err, ErrBadParcelable) {
		t.Fatalf("Transact mismatch error = %v, want ErrBadParcelable", err)
	}
}

func TestReplayBinderReplaysRemoteException(t *testing.T) {
	replay := NewReplayBinder([]TransactionRecord{{
		Descriptor: "err",
		Code:       FirstCallTransaction,
		Flags:      FlagNone,
		Error: &RecordedError{
			Kind:          "remote_exception",
			Message:       "boom",
			ExceptionCode: ExceptionIllegalState,
		},
	}})

	if _, err := replay.Transact(context.Background(), FirstCallTransaction, nil, FlagNone); err == nil {
		t.Fatal("Transact error = nil, want remote exception")
	} else {
		var remote *RemoteException
		if !errors.As(err, &remote) {
			t.Fatalf("error = %T, want *RemoteException", err)
		}
		if remote.Message != "boom" {
			t.Fatalf("RemoteException.Message = %q, want boom", remote.Message)
		}
	}
}

type echoBinder struct{}

func (b echoBinder) Descriptor(ctx context.Context) (string, error) {
	return "echo", nil
}

func (b echoBinder) Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error) {
	msg, err := data.ReadString()
	if err != nil {
		return nil, err
	}

	reply := NewParcel()
	if err := reply.WriteString("echo:" + msg); err != nil {
		return nil, err
	}
	if err := reply.SetPosition(0); err != nil {
		return nil, err
	}
	return reply, nil
}

func (b echoBinder) WatchDeath(ctx context.Context) (Subscription, error) {
	return nil, ErrUnsupported
}

func (b echoBinder) Close() error {
	return nil
}
