package libbinder

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRPCServiceManagerAndTransact(t *testing.T) {
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

	handler := &countingRPCHandler{
		descriptor: "rpc.echo",
		handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
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
	if err := server.ServiceManager().AddService(context.Background(), "echo", handler); err != nil {
		t.Fatalf("server AddService: %v", err)
	}

	service, err := client.ServiceManager().CheckService(context.Background(), "echo")
	if err != nil {
		t.Fatalf("client CheckService(first): %v", err)
	}
	serviceAgain, err := client.ServiceManager().CheckService(context.Background(), "echo")
	if err != nil {
		t.Fatalf("client CheckService(second): %v", err)
	}
	if serviceAgain == nil {
		t.Fatal("client CheckService(second) returned nil service")
	}
	if snapshot := client.DebugSnapshot(); snapshot.ServiceManager.CacheHits == 0 {
		t.Fatalf("CacheHits = %d, want > 0", snapshot.ServiceManager.CacheHits)
	}

	desc, err := service.Descriptor(context.Background())
	if err != nil {
		t.Fatalf("Descriptor(first): %v", err)
	}
	if desc != "rpc.echo" {
		t.Fatalf("Descriptor = %q, want rpc.echo", desc)
	}
	if _, err := service.Descriptor(context.Background()); err != nil {
		t.Fatalf("Descriptor(second): %v", err)
	}
	if got := handler.descriptorCalls(); got != 1 {
		t.Fatalf("descriptor calls = %d, want 1", got)
	}

	req := api.NewParcel()
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("req.WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("req.SetPosition: %v", err)
	}
	reply, err := service.Transact(context.Background(), api.FirstCallTransaction, req, api.FlagNone)
	if err != nil {
		t.Fatalf("Transact: %v", err)
	}
	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("reply = %q, want echo:ping", got)
	}
}

func TestRPCCallbackBinderRoundTrip(t *testing.T) {
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

	if err := server.ServiceManager().AddService(context.Background(), "callback", api.StaticHandler{
		DescriptorName: "rpc.callback",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			cb, err := data.ReadStrongBinder()
			if err != nil {
				return nil, err
			}
			msg, err := data.ReadString()
			if err != nil {
				return nil, err
			}

			req := api.NewParcel()
			if err := req.WriteString(msg); err != nil {
				return nil, err
			}
			if err := req.SetPosition(0); err != nil {
				return nil, err
			}
			reply, err := cb.Transact(ctx, api.FirstCallTransaction, req, api.FlagNone)
			if err != nil {
				return nil, err
			}
			got, err := reply.ReadString()
			if err != nil {
				return nil, err
			}

			out := api.NewParcel()
			if err := out.WriteString("server:" + got); err != nil {
				return nil, err
			}
			return out, nil
		},
	}); err != nil {
		t.Fatalf("server AddService: %v", err)
	}

	service, err := client.ServiceManager().WaitService(context.Background(), "callback")
	if err != nil {
		t.Fatalf("client WaitService: %v", err)
	}

	callback, err := client.RegisterLocalHandler(api.StaticHandler{
		DescriptorName: "rpc.client.callback",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			msg, err := data.ReadString()
			if err != nil {
				return nil, err
			}
			reply := api.NewParcel()
			if err := reply.WriteString("client:" + msg); err != nil {
				return nil, err
			}
			return reply, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterLocalHandler: %v", err)
	}

	req := api.NewParcel()
	if err := req.WriteStrongBinder(callback); err != nil {
		t.Fatalf("req.WriteStrongBinder: %v", err)
	}
	if err := req.WriteString("ping"); err != nil {
		t.Fatalf("req.WriteString: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("req.SetPosition: %v", err)
	}
	reply, err := service.Transact(context.Background(), api.FirstCallTransaction, req, api.FlagNone)
	if err != nil {
		t.Fatalf("Transact: %v", err)
	}
	got, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if got != "server:client:ping" {
		t.Fatalf("reply = %q, want server:client:ping", got)
	}
}

func TestRPCRejectsFileDescriptors(t *testing.T) {
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

	if err := server.ServiceManager().AddService(context.Background(), "fd", api.StaticHandler{
		DescriptorName: "rpc.fd",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return api.NewParcel(), nil
		},
	}); err != nil {
		t.Fatalf("server AddService: %v", err)
	}
	service, err := client.ServiceManager().WaitService(context.Background(), "fd")
	if err != nil {
		t.Fatalf("client WaitService: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() {
		_ = r.Close()
		_ = w.Close()
	}()

	req := api.NewParcel()
	if err := req.WriteFileDescriptor(api.NewFileDescriptor(int(r.Fd()))); err != nil {
		t.Fatalf("WriteFileDescriptor: %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	if _, err := service.Transact(context.Background(), api.FirstCallTransaction, req, api.FlagNone); !errors.Is(err, api.ErrUnsupported) {
		t.Fatalf("Transact(fd) error = %v, want ErrUnsupported", err)
	}
}

func TestRPCDebugSnapshot(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	service, err := client.ServiceManager().WaitService(ctx, "echo")
	if err != nil {
		t.Fatalf("WaitService: %v", err)
	}
	if _, err := service.Descriptor(context.Background()); err != nil {
		t.Fatalf("Descriptor: %v", err)
	}

	snapshot := client.DebugSnapshot()
	if snapshot.ImportedObjects == 0 {
		t.Fatalf("ImportedObjects = %d, want > 0", snapshot.ImportedObjects)
	}
	if snapshot.FramePoolGets == 0 || snapshot.FramePoolPuts == 0 {
		t.Fatalf("FramePool stats = (%d, %d), want both > 0", snapshot.FramePoolGets, snapshot.FramePoolPuts)
	}
}

type countingRPCHandler struct {
	descriptor string
	handle     api.HandlerFunc

	mu    sync.Mutex
	calls int
}

func (h *countingRPCHandler) Descriptor() string {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()
	return h.descriptor
}

func (h *countingRPCHandler) HandleTransact(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
	return h.handle(ctx, code, data)
}

func (h *countingRPCHandler) descriptorCalls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls
}
