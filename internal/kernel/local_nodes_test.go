package kernel

import (
	"context"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestDispatchLocalTransaction(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	handler := api.StaticHandler{
		DescriptorName: "libbinder.go.test.ILocal",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			if code != FirstCallTransaction {
				t.Fatalf("code = %d, want %d", code, FirstCallTransaction)
			}

			msg, err := data.ReadString()
			if err != nil {
				return nil, err
			}

			reply := api.NewParcel()
			if err := reply.WriteString("local:" + msg); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}

	node, err := backend.RegisterLocalNode(handler, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	req := api.NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("req.WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("req.SetPosition: %v", err)
	}

	reply, err := backend.DispatchLocalTransaction(context.Background(), node.ID, FirstCallTransaction, req, 0)
	if err != nil {
		t.Fatalf("DispatchLocalTransaction: %v", err)
	}
	if reply == nil {
		t.Fatal("DispatchLocalTransaction returned nil reply")
	}

	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "local:ping" {
		t.Fatalf("reply.ReadString = %q, want %q", got, "local:ping")
	}
}

func TestDispatchLocalTransactionDescriptor(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	node, err := backend.RegisterLocalNode(api.StaticHandler{
		DescriptorName: "libbinder.go.test.IDescriptor",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return nil, nil
		},
	}, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	reply, err := backend.DispatchLocalTransaction(context.Background(), node.ID, InterfaceTransaction, api.NewParcel(), 0)
	if err != nil {
		t.Fatalf("DispatchLocalTransaction(interface): %v", err)
	}

	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "libbinder.go.test.IDescriptor" {
		t.Fatalf("reply.ReadString = %q, want %q", got, "libbinder.go.test.IDescriptor")
	}
}
