package binder

import (
	"errors"
	"testing"
)

func TestWriteNoExceptionReadException(t *testing.T) {
	p := NewParcel()
	if err := WriteNoException(p); err != nil {
		t.Fatalf("WriteNoException: %v", err)
	}
	if err := p.WriteString("ok"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	if err := ReadException(p); err != nil {
		t.Fatalf("ReadException: %v", err)
	}
	got, err := p.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != "ok" {
		t.Fatalf("ReadString = %q, want %q", got, "ok")
	}
}

func TestReadExceptionRemoteException(t *testing.T) {
	p := NewParcel()
	if err := p.WriteInt32(int32(ExceptionServiceSpecific)); err != nil {
		t.Fatalf("WriteInt32(code): %v", err)
	}
	if err := p.WriteString("boom"); err != nil {
		t.Fatalf("WriteString(message): %v", err)
	}
	if err := p.WriteInt32(0); err != nil {
		t.Fatalf("WriteInt32(stack header): %v", err)
	}
	if err := p.WriteInt32(42); err != nil {
		t.Fatalf("WriteInt32(service code): %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	err := ReadException(p)
	var remote *RemoteException
	if !errors.As(err, &remote) {
		t.Fatalf("ReadException error = %T, want *RemoteException", err)
	}
	if remote.Code != ExceptionServiceSpecific {
		t.Fatalf("Remote.Code = %d, want %d", remote.Code, ExceptionServiceSpecific)
	}
	if remote.Message != "boom" {
		t.Fatalf("Remote.Message = %q, want %q", remote.Message, "boom")
	}
	if remote.ServiceCode != 42 {
		t.Fatalf("Remote.ServiceCode = %d, want 42", remote.ServiceCode)
	}
}

func TestTryWriteExceptionServiceSpecific(t *testing.T) {
	p := NewParcel()
	handled, err := TryWriteException(p, &ServiceSpecificError{
		Code:    7,
		Message: "boom",
	})
	if err != nil {
		t.Fatalf("TryWriteException: %v", err)
	}
	if !handled {
		t.Fatal("TryWriteException handled = false, want true")
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	readErr := ReadException(p)
	var remote *RemoteException
	if !errors.As(readErr, &remote) {
		t.Fatalf("ReadException error = %T, want *RemoteException", readErr)
	}
	if remote.Code != ExceptionServiceSpecific {
		t.Fatalf("Remote.Code = %d, want %d", remote.Code, ExceptionServiceSpecific)
	}
	if remote.Message != "boom" {
		t.Fatalf("Remote.Message = %q, want %q", remote.Message, "boom")
	}
	if remote.ServiceCode != 7 {
		t.Fatalf("Remote.ServiceCode = %d, want 7", remote.ServiceCode)
	}
}

func TestTryWriteExceptionUnsupported(t *testing.T) {
	p := NewParcel()
	handled, err := TryWriteException(p, errors.New("plain"))
	if err != nil {
		t.Fatalf("TryWriteException: %v", err)
	}
	if handled {
		t.Fatal("TryWriteException handled = true, want false")
	}
}
