package libbinder

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRPCServiceManagerGovernanceQueries(t *testing.T) {
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

	serviceName := "rpc.example.IFoo/default"
	handler := api.WithStability(api.StaticHandler{
		DescriptorName: "rpc.example.IFoo",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}, api.StabilityVINTF)
	if err := server.ServiceManager().AddService(context.Background(), serviceName, handler,
		api.WithDeclaredService(true),
		api.WithUpdatableViaApex("com.android.rpc"),
		api.WithConnectionInfo(api.ConnectionInfo{IPAddress: "127.0.0.1", Port: 7777}),
		api.WithDebugPID(4242),
		api.WithLazyService(true),
	); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	names, err := client.ServiceManager().ListServices(context.Background(), api.DumpPriorityAll)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(names) != 1 || names[0] != serviceName {
		t.Fatalf("ListServices = %#v, want [%q]", names, serviceName)
	}

	declared, err := client.ServiceManager().IsDeclared(context.Background(), serviceName)
	if err != nil {
		t.Fatalf("IsDeclared: %v", err)
	}
	if !declared {
		t.Fatal("IsDeclared = false, want true")
	}

	instances, err := client.ServiceManager().DeclaredInstances(context.Background(), "rpc.example.IFoo")
	if err != nil {
		t.Fatalf("DeclaredInstances: %v", err)
	}
	if len(instances) != 1 || instances[0] != "default" {
		t.Fatalf("DeclaredInstances = %#v, want [default]", instances)
	}

	apex, err := client.ServiceManager().UpdatableViaApex(context.Background(), serviceName)
	if err != nil {
		t.Fatalf("UpdatableViaApex: %v", err)
	}
	if apex == nil || *apex != "com.android.rpc" {
		t.Fatalf("UpdatableViaApex = %#v, want com.android.rpc", apex)
	}

	updatableNames, err := client.ServiceManager().UpdatableNames(context.Background(), "com.android.rpc")
	if err != nil {
		t.Fatalf("UpdatableNames: %v", err)
	}
	if len(updatableNames) != 1 || updatableNames[0] != serviceName {
		t.Fatalf("UpdatableNames = %#v, want [%q]", updatableNames, serviceName)
	}

	connectionInfo, err := client.ServiceManager().ConnectionInfo(context.Background(), serviceName)
	if err != nil {
		t.Fatalf("ConnectionInfo: %v", err)
	}
	if connectionInfo == nil || connectionInfo.IPAddress != "127.0.0.1" || connectionInfo.Port != 7777 {
		t.Fatalf("ConnectionInfo = %#v, want 127.0.0.1:7777", connectionInfo)
	}

	debugInfo, err := client.ServiceManager().DebugInfo(context.Background())
	if err != nil {
		t.Fatalf("DebugInfo: %v", err)
	}
	if len(debugInfo) != 1 || debugInfo[0].Name != serviceName || debugInfo[0].DebugPID != 4242 {
		t.Fatalf("DebugInfo = %#v, want one entry with pid 4242", debugInfo)
	}
}

func TestRPCServiceManagerWatchServiceRegistrations(t *testing.T) {
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

	registrations := make(chan api.ServiceRegistration, 2)
	sub, err := client.ServiceManager().WatchServiceRegistrations(context.Background(), "late", func(ctx context.Context, reg api.ServiceRegistration) {
		registrations <- reg
	})
	if err != nil {
		t.Fatalf("WatchServiceRegistrations: %v", err)
	}
	defer func() { _ = sub.Close() }()

	if err := server.ServiceManager().AddService(context.Background(), "late", api.StaticHandler{
		DescriptorName: "rpc.late",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}); err != nil {
		t.Fatalf("AddService(late): %v", err)
	}

	select {
	case reg := <-registrations:
		if reg.Name != "late" || reg.Binder == nil {
			t.Fatalf("registration = %#v, want populated event", reg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive registration callback")
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("sub.Close: %v", err)
	}

	if err := server.ServiceManager().AddService(context.Background(), "late", api.StaticHandler{
		DescriptorName: "rpc.late.v2",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}); err != nil {
		t.Fatalf("AddService(late v2): %v", err)
	}

	select {
	case reg := <-registrations:
		t.Fatalf("received callback after close: %#v", reg)
	case <-time.After(250 * time.Millisecond):
	}
}

func TestRPCServiceManagerWatchClientsAndTryUnregister(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server, err := ServeRPC(serverConn)
	if err != nil {
		t.Fatalf("ServeRPC: %v", err)
	}
	client, err := DialRPC(clientConn)
	if err != nil {
		t.Fatalf("DialRPC: %v", err)
	}
	defer func() { _ = server.Close() }()

	if err := server.ServiceManager().AddService(context.Background(), "echo", api.StaticHandler{
		DescriptorName: "rpc.echo",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			reply := api.NewParcel()
			if err := reply.WriteString("ok"); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}); err != nil {
		t.Fatalf("AddService: %v", err)
	}

	localService, err := server.ServiceManager().CheckService(context.Background(), "echo")
	if err != nil {
		t.Fatalf("server CheckService: %v", err)
	}

	updates := make(chan api.ServiceClientUpdate, 4)
	sub, err := server.ServiceManager().WatchClients(context.Background(), "echo", localService, func(ctx context.Context, update api.ServiceClientUpdate) {
		updates <- update
	})
	if err != nil {
		t.Fatalf("WatchClients: %v", err)
	}
	defer func() { _ = sub.Close() }()

	if _, err := client.ServiceManager().CheckService(context.Background(), "echo"); err != nil {
		t.Fatalf("client CheckService: %v", err)
	}

	select {
	case update := <-updates:
		if !update.HasClients || update.Name != "echo" || update.Service == nil {
			t.Fatalf("client update = %#v, want hasClients=true", update)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive hasClients=true callback")
	}

	if err := server.ServiceManager().TryUnregisterService(context.Background(), "echo", localService); err == nil {
		t.Fatal("TryUnregisterService succeeded with active clients, want failure")
	} else {
		var statusErr *api.StatusCodeError
		if !errors.As(err, &statusErr) || statusErr.Code != api.StatusInvalidOperation {
			t.Fatalf("TryUnregisterService error = %v, want StatusInvalidOperation", err)
		}
	}

	if err := client.Close(); err != nil {
		t.Fatalf("client.Close: %v", err)
	}

	select {
	case update := <-updates:
		if update.HasClients {
			t.Fatalf("client update = %#v, want hasClients=false", update)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive hasClients=false callback")
	}

	if err := server.ServiceManager().TryUnregisterService(context.Background(), "echo", localService); err != nil {
		t.Fatalf("TryUnregisterService(after close): %v", err)
	}
}
