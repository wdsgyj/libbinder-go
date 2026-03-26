package binder

import (
	"context"
	"testing"
)

func TestLazyHandlerInitializesOnFirstCall(t *testing.T) {
	var calls int
	handler := NewLazyHandler("lazy", func() (Handler, error) {
		calls++
		return StaticHandler{
			DescriptorName: "lazy.inner",
			Handle: func(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
				reply := NewParcel()
				if err := reply.WriteString("ok"); err != nil {
					return nil, err
				}
				if err := reply.SetPosition(0); err != nil {
					return nil, err
				}
				return reply, nil
			},
		}, nil
	})

	if handler.Descriptor() != "lazy" {
		t.Fatalf("Descriptor = %q, want lazy", handler.Descriptor())
	}
	if calls != 0 {
		t.Fatalf("factory calls = %d, want 0 before transact", calls)
	}

	reply, err := handler.HandleTransact(context.Background(), FirstCallTransaction, NewParcel())
	if err != nil {
		t.Fatalf("HandleTransact(first): %v", err)
	}
	if _, err := handler.HandleTransact(context.Background(), FirstCallTransaction, NewParcel()); err != nil {
		t.Fatalf("HandleTransact(second): %v", err)
	}
	if calls != 1 {
		t.Fatalf("factory calls = %d, want 1", calls)
	}
	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "ok" {
		t.Fatalf("reply = %q, want ok", got)
	}
}

func TestLazyHandlerWithMetadata(t *testing.T) {
	version := int32(9)
	hash := "hash-9"
	handler := NewLazyHandlerWithMetadata(LazyHandlerConfig{
		Descriptor: "lazy.meta",
		Version:    &version,
		Hash:       &hash,
	}, func() (Handler, error) {
		return StaticHandler{
			DescriptorName: "inner",
			Handle: func(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
				return NewParcel(), nil
			},
		}, nil
	})

	versionProvider, ok := handler.(InterfaceVersionProvider)
	if !ok || versionProvider.InterfaceVersion() != 9 {
		t.Fatalf("InterfaceVersionProvider = (%v, %v), want (true, 9)", ok, int32(9))
	}
	hashProvider, ok := handler.(InterfaceHashProvider)
	if !ok {
		t.Fatal("handler missing InterfaceHashProvider")
	}
	if hashProvider.InterfaceHash() != "hash-9" {
		t.Fatalf("InterfaceHash = %q, want hash-9", hashProvider.InterfaceHash())
	}
}
