//go:build android

package runtime

import (
	"context"
	"testing"

	api "libbinder-go/binder"
	"libbinder-go/internal/kernel"
)

func TestRuntimeTransactHandlePingContextManagerOnAndroid(t *testing.T) {
	rt := New(Config{})
	if err := rt.Start(Config{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if err := rt.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	reply, err := rt.TransactHandle(context.Background(), 0, kernel.PingTransaction, api.NewParcel(), api.FlagNone)
	if err != nil {
		t.Fatalf("TransactHandle: %v", err)
	}
	if reply == nil {
		t.Fatal("TransactHandle returned nil reply for synchronous ping")
	}
	if reply.Len() != 0 {
		t.Fatalf("reply.Len() = %d, want 0", reply.Len())
	}
}
