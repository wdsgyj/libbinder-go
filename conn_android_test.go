//go:build android

package libbindergo

import (
	"context"
	"errors"
	"testing"
	"time"

	api "libbinder-go/binder"
)

func TestContextManagerDescriptorOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	desc, err := conn.ContextManager().Descriptor(context.Background())
	if err != nil {
		t.Fatalf("ContextManager().Descriptor: %v", err)
	}
	if desc != "android.os.IServiceManager" {
		t.Fatalf("Descriptor = %q, want %q", desc, "android.os.IServiceManager")
	}
}

func TestServiceManagerCheckServiceOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	service, err := conn.ServiceManager().CheckService(context.Background(), "activity")
	if err != nil {
		t.Fatalf("CheckService(activity): %v", err)
	}
	if service == nil {
		t.Fatal("CheckService(activity) returned nil service")
	}

	desc, err := service.Descriptor(context.Background())
	if err != nil {
		t.Fatalf("service.Descriptor: %v", err)
	}
	if desc != "android.app.IActivityManager" {
		t.Fatalf("Descriptor = %q, want %q", desc, "android.app.IActivityManager")
	}
}

func TestServiceManagerCheckServiceMissingOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	_, err := conn.ServiceManager().CheckService(context.Background(), "libbinder.go.missing.service")
	if !errors.Is(err, api.ErrNoService) {
		t.Fatalf("CheckService(missing) error = %v, want ErrNoService", err)
	}
}

func TestServiceManagerWaitServiceExistingOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	service, err := conn.ServiceManager().WaitService(ctx, "activity")
	if err != nil {
		t.Fatalf("WaitService(activity): %v", err)
	}
	if service == nil {
		t.Fatal("WaitService(activity) returned nil service")
	}
}

func mustOpenConn(t *testing.T) *Conn {
	t.Helper()

	conn, err := Open(Config{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return conn
}

func closeConn(t *testing.T, conn *Conn) {
	t.Helper()

	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
