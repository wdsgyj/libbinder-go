package protocol

import (
	"errors"
	"testing"

	api "libbinder-go/binder"
)

func TestStatusCodecOK(t *testing.T) {
	p := api.NewParcel()
	if err := WriteStatus(p, Status{}); err != nil {
		t.Fatalf("WriteStatus(ok): %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadStatus(p)
	if err != nil {
		t.Fatalf("ReadStatus(ok): %v", err)
	}
	if !got.IsOK() {
		t.Fatalf("ReadStatus(ok) = %#v, want OK", got)
	}
}

func TestStatusCodecRemoteException(t *testing.T) {
	want := Status{
		Remote: &RemoteException{
			Code:    api.ExceptionIllegalState,
			Message: "boom",
		},
	}

	p := api.NewParcel()
	if err := WriteStatus(p, want); err != nil {
		t.Fatalf("WriteStatus(remote): %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadStatus(p)
	if err != nil {
		t.Fatalf("ReadStatus(remote): %v", err)
	}
	if got.Remote == nil {
		t.Fatal("ReadStatus(remote) returned nil Remote")
	}
	if got.Remote.Code != want.Remote.Code {
		t.Fatalf("Remote.Code = %d, want %d", got.Remote.Code, want.Remote.Code)
	}
	if got.Remote.Message != want.Remote.Message {
		t.Fatalf("Remote.Message = %q, want %q", got.Remote.Message, want.Remote.Message)
	}
}

func TestStatusCodecServiceSpecific(t *testing.T) {
	want := Status{
		Remote: &RemoteException{
			Code:        api.ExceptionServiceSpecific,
			Message:     "bad input",
			ServiceCode: 42,
		},
	}

	p := api.NewParcel()
	if err := WriteStatus(p, want); err != nil {
		t.Fatalf("WriteStatus(service specific): %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadStatus(p)
	if err != nil {
		t.Fatalf("ReadStatus(service specific): %v", err)
	}
	if got.Remote == nil {
		t.Fatal("ReadStatus(service specific) returned nil Remote")
	}
	if got.Remote.Code != want.Remote.Code {
		t.Fatalf("Remote.Code = %d, want %d", got.Remote.Code, want.Remote.Code)
	}
	if got.Remote.Message != want.Remote.Message {
		t.Fatalf("Remote.Message = %q, want %q", got.Remote.Message, want.Remote.Message)
	}
	if got.Remote.ServiceCode != want.Remote.ServiceCode {
		t.Fatalf("Remote.ServiceCode = %d, want %d", got.Remote.ServiceCode, want.Remote.ServiceCode)
	}
}

func TestStatusCodecReplyHeaders(t *testing.T) {
	t.Run("fat reply header becomes OK", func(t *testing.T) {
		p := api.NewParcel()
		if err := p.WriteInt32(int32(exceptionHasReplyHeader)); err != nil {
			t.Fatalf("WriteInt32(exception): %v", err)
		}
		if err := p.WriteInt32(8); err != nil {
			t.Fatalf("WriteInt32(header size): %v", err)
		}
		if err := p.WriteInt32(1234); err != nil {
			t.Fatalf("WriteInt32(filler): %v", err)
		}

		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadStatus(p)
		if err != nil {
			t.Fatalf("ReadStatus(reply header): %v", err)
		}
		if !got.IsOK() {
			t.Fatalf("ReadStatus(reply header) = %#v, want OK", got)
		}
		if p.Position() != 12 {
			t.Fatalf("Position = %d, want 12", p.Position())
		}
	})

	t.Run("noted appops header is skipped", func(t *testing.T) {
		p := api.NewParcel()
		if err := p.WriteInt32(int32(exceptionHasNotedAppOpsReplyHeader)); err != nil {
			t.Fatalf("WriteInt32(appops exception): %v", err)
		}
		if err := p.WriteInt32(8); err != nil {
			t.Fatalf("WriteInt32(appops header size): %v", err)
		}
		if err := p.WriteInt32(999); err != nil {
			t.Fatalf("WriteInt32(appops filler): %v", err)
		}
		if err := WriteStatus(p, Status{
			Remote: &RemoteException{
				Code:    api.ExceptionIllegalArgument,
				Message: "bad arg",
			},
		}); err != nil {
			t.Fatalf("WriteStatus(after appops header): %v", err)
		}

		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadStatus(p)
		if err != nil {
			t.Fatalf("ReadStatus(appops header): %v", err)
		}
		if got.Remote == nil {
			t.Fatal("ReadStatus(appops header) returned nil Remote")
		}
		if got.Remote.Code != api.ExceptionIllegalArgument {
			t.Fatalf("Remote.Code = %d, want %d", got.Remote.Code, api.ExceptionIllegalArgument)
		}
		if got.Remote.Message != "bad arg" {
			t.Fatalf("Remote.Message = %q, want %q", got.Remote.Message, "bad arg")
		}
	})
}

func TestStatusCodecInvalidHeader(t *testing.T) {
	p := api.NewParcel()
	if err := p.WriteInt32(int32(exceptionHasReplyHeader)); err != nil {
		t.Fatalf("WriteInt32(exception): %v", err)
	}
	if err := p.WriteInt32(64); err != nil {
		t.Fatalf("WriteInt32(header size): %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	_, err := ReadStatus(p)
	if !errors.Is(err, api.ErrBadParcelable) {
		t.Fatalf("ReadStatus error = %v, want ErrBadParcelable", err)
	}
}
