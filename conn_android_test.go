//go:build android

package libbinder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/kernel"
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

func TestServiceManagerAddServiceAndTransactOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	serviceName := fmt.Sprintf("libbinder.go.test.echo.%d", time.Now().UnixNano())
	handler := api.StaticHandler{
		DescriptorName: "libbinder.go.test.IEcho",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			if code != kernel.FirstCallTransaction {
				return nil, fmt.Errorf("unexpected code %d", code)
			}

			msg, err := data.ReadString()
			if err != nil {
				return nil, err
			}

			reply := api.NewParcel()
			if err := reply.WriteString("echo:" + msg); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}

	if err := conn.ServiceManager().AddService(context.Background(), serviceName, handler); err != nil {
		var remoteErr *api.RemoteException
		if errors.As(err, &remoteErr) && remoteErr.Code == api.ExceptionSecurity && strings.Contains(remoteErr.Message, "SELinux denied") {
			t.Skipf("stock emulator denies addService for test binaries: %v", err)
		}
		t.Fatalf("AddService(%s): %v", serviceName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	service, err := conn.ServiceManager().WaitService(ctx, serviceName)
	if err != nil {
		t.Fatalf("WaitService(%s): %v", serviceName, err)
	}

	desc, err := service.Descriptor(context.Background())
	if err != nil {
		t.Fatalf("service.Descriptor: %v", err)
	}
	if desc != handler.DescriptorName {
		t.Fatalf("Descriptor = %q, want %q", desc, handler.DescriptorName)
	}

	req := api.NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("req.WriteString: %v", err)
	}

	reply, err := service.Transact(context.Background(), kernel.FirstCallTransaction, req, api.FlagNone)
	if err != nil {
		t.Fatalf("service.Transact: %v", err)
	}
	if reply == nil {
		t.Fatal("service.Transact returned nil reply")
	}

	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("reply.ReadString = %q, want %q", got, "echo:ping")
	}
}

func TestServiceWatchDeathCloseOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	service, err := conn.ServiceManager().CheckService(context.Background(), "activity")
	if err != nil {
		t.Fatalf("CheckService(activity): %v", err)
	}

	sub, err := service.WatchDeath(context.Background())
	if err != nil {
		t.Fatalf("WatchDeath(activity): %v", err)
	}
	if sub == nil {
		t.Fatal("WatchDeath(activity) returned nil subscription")
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("sub.Close: %v", err)
	}
	waitSubscriptionDone(t, sub.Done(), "WatchDeath.Close")

	if err := service.Close(); err != nil {
		t.Fatalf("service.Close: %v", err)
	}
	if _, err := service.Descriptor(context.Background()); !errors.Is(err, api.ErrClosed) {
		t.Fatalf("Descriptor after Close error = %v, want ErrClosed", err)
	}
}

func TestServiceWatchDeathContextCancelOnAndroid(t *testing.T) {
	conn := mustOpenConn(t)
	defer closeConn(t, conn)

	service, err := conn.ServiceManager().CheckService(context.Background(), "activity")
	if err != nil {
		t.Fatalf("CheckService(activity): %v", err)
	}
	defer func() {
		if err := service.Close(); err != nil {
			t.Fatalf("service.Close: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := service.WatchDeath(ctx)
	if err != nil {
		t.Fatalf("WatchDeath(activity): %v", err)
	}

	cancel()
	waitSubscriptionDone(t, sub.Done(), "WatchDeath context cancel")
	if err := sub.Err(); err != nil {
		t.Fatalf("sub.Err() = %v, want nil", err)
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

func waitSubscriptionDone(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}
