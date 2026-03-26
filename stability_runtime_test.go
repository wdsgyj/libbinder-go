package libbinder

import (
	"context"
	"errors"
	"net"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRPCTransactStabilityEnforcement(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server, err := ServeRPC(serverConn)
	if err != nil {
		t.Fatalf("ServeRPC: %v", err)
	}
	client, err := DialRPC(clientConn)
	if err != nil {
		t.Fatalf("DialRPC: %v", err)
	}
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()

	if err := server.ServiceManager().AddService(context.Background(), "vendor", api.WithStability(api.StaticHandler{
		DescriptorName: "rpc.vendor",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := reply.WriteString("ok"); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}, api.StabilityVendor)); err != nil {
		t.Fatalf("AddService(vendor): %v", err)
	}

	service, err := client.ServiceManager().WaitService(context.Background(), "vendor")
	if err != nil {
		t.Fatalf("WaitService(vendor): %v", err)
	}

	if _, err := service.Transact(context.Background(), api.FirstCallTransaction, api.NewParcel(), api.FlagNone); err == nil {
		t.Fatal("Transact(system context on vendor binder) = nil, want BAD_TYPE")
	} else {
		var statusErr *api.StatusCodeError
		if !errors.As(err, &statusErr) || statusErr.Code != api.StatusBadType {
			t.Fatalf("Transact(system context on vendor binder) error = %v, want StatusBadType", err)
		}
	}

	ctx := api.WithRequiredStability(context.Background(), api.StabilityVendor)
	reply, err := service.Transact(ctx, api.FirstCallTransaction, api.NewParcel(), api.FlagNone)
	if err != nil {
		t.Fatalf("Transact(vendor context): %v", err)
	}
	if reply == nil {
		t.Fatal("Transact(vendor context) returned nil reply")
	}

	if _, err := service.Transact(context.Background(), api.FirstCallTransaction, api.NewParcel(), api.FlagPrivateVendor); err != nil {
		t.Fatalf("Transact(private vendor): %v", err)
	}
}

func TestRPCPrivateVendorRejectsSystemBinder(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server, err := ServeRPC(serverConn)
	if err != nil {
		t.Fatalf("ServeRPC: %v", err)
	}
	client, err := DialRPC(clientConn)
	if err != nil {
		t.Fatalf("DialRPC: %v", err)
	}
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()

	if err := server.ServiceManager().AddService(context.Background(), "system", api.WithStability(api.StaticHandler{
		DescriptorName: "rpc.system",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}, api.StabilitySystem)); err != nil {
		t.Fatalf("AddService(system): %v", err)
	}

	service, err := client.ServiceManager().WaitService(context.Background(), "system")
	if err != nil {
		t.Fatalf("WaitService(system): %v", err)
	}

	if _, err := service.Transact(context.Background(), api.FirstCallTransaction, api.NewParcel(), api.FlagPrivateVendor); err == nil {
		t.Fatal("Transact(private vendor on system binder) = nil, want BAD_TYPE")
	} else {
		var statusErr *api.StatusCodeError
		if !errors.As(err, &statusErr) || statusErr.Code != api.StatusBadType {
			t.Fatalf("Transact(private vendor on system binder) error = %v, want StatusBadType", err)
		}
	}
}
