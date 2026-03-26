package kernel

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestDispatchLocalTransactionContext(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	node, err := backend.RegisterLocalNode(api.StaticHandler{
		DescriptorName: "libbinder.go.test.IContext",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			tx, ok := api.TransactionContextFromContext(ctx)
			if !ok {
				t.Fatal("TransactionContextFromContext = false, want true")
			}
			if tx.Code != FirstCallTransaction {
				t.Fatalf("tx.Code = %d, want %d", tx.Code, FirstCallTransaction)
			}
			if tx.Flags != api.FlagOneway {
				t.Fatalf("tx.Flags = %d, want %d", tx.Flags, api.FlagOneway)
			}
			if tx.CallingPID != int32(os.Getpid()) {
				t.Fatalf("tx.CallingPID = %d, want %d", tx.CallingPID, os.Getpid())
			}
			if tx.CallingUID != uint32(os.Geteuid()) {
				t.Fatalf("tx.CallingUID = %d, want %d", tx.CallingUID, os.Geteuid())
			}
			if !tx.Local {
				t.Fatal("tx.Local = false, want true")
			}

			reply := api.NewParcel()
			if err := reply.WriteString("ok"); err != nil {
				return nil, err
			}
			return reply, nil
		},
	}, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	reply, err := backend.DispatchLocalTransaction(context.Background(), node.ID, FirstCallTransaction, api.NewParcel(), uint32(api.FlagOneway))
	if err != nil {
		t.Fatalf("DispatchLocalTransaction: %v", err)
	}
	if reply != nil {
		t.Fatalf("DispatchLocalTransaction(reply) = %#v, want nil for oneway", reply)
	}
}

func TestDispatchLocalTransactionSerialHandler(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)
	block := make(chan struct{})
	entered := make(chan struct{}, 2)

	node, err := backend.RegisterLocalNode(api.StaticHandler{
		DescriptorName: "libbinder.go.test.ISerial",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			entered <- struct{}{}
			<-block
			return api.NewParcel(), nil
		},
	}, true)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	var wg sync.WaitGroup
	start := func() {
		defer wg.Done()
		if _, err := backend.DispatchLocalTransaction(context.Background(), node.ID, FirstCallTransaction, api.NewParcel(), 0); err != nil {
			t.Errorf("DispatchLocalTransaction: %v", err)
		}
	}

	wg.Add(2)
	go start()
	<-entered
	go start()

	select {
	case <-entered:
		t.Fatal("second serial handler entered before first finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(block)
	wg.Wait()
}

func TestDispatchLocalTransactionStableBuiltins(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	handler := stableTestHandler{
		StaticHandler: api.StaticHandler{
			DescriptorName: "libbinder.go.test.IStable",
			Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
				return nil, api.ErrUnknownTransaction
			},
		},
		version: 7,
		hash:    "abc123",
	}

	node, err := backend.RegisterLocalNode(handler, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	reply, err := backend.DispatchLocalTransaction(context.Background(), node.ID, GetInterfaceVersionTransaction, api.NewParcel(), 0)
	if err != nil {
		t.Fatalf("DispatchLocalTransaction(version): %v", err)
	}
	version, err := reply.ReadInt32()
	if err != nil {
		t.Fatalf("reply.ReadInt32: %v", err)
	}
	if version != 7 {
		t.Fatalf("version = %d, want 7", version)
	}

	reply, err = backend.DispatchLocalTransaction(context.Background(), node.ID, GetInterfaceHashTransaction, api.NewParcel(), 0)
	if err != nil {
		t.Fatalf("DispatchLocalTransaction(hash): %v", err)
	}
	hash, err := reply.ReadString()
	if err != nil {
		t.Fatalf("reply.ReadString: %v", err)
	}
	if hash != "abc123" {
		t.Fatalf("hash = %q, want %q", hash, "abc123")
	}

	node, err = backend.RegisterLocalNode(api.StaticHandler{
		DescriptorName: "libbinder.go.test.IPlain",
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			return nil, nil
		},
	}, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode(plain): %v", err)
	}
	if _, err := backend.DispatchLocalTransaction(context.Background(), node.ID, GetInterfaceVersionTransaction, api.NewParcel(), 0); !errors.Is(err, api.ErrUnknownTransaction) {
		t.Fatalf("DispatchLocalTransaction(missing version) error = %v, want ErrUnknownTransaction", err)
	}
}

func TestDispatchLocalTransactionDumpAndDebugPID(t *testing.T) {
	backend := NewBackend(DefaultDriverPath)

	handler := dumpDebugHandler{
		StaticHandler: api.StaticHandler{
			DescriptorName: "libbinder.go.test.IDump",
			Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
				return nil, api.ErrUnknownTransaction
			},
		},
		pid: 555,
	}

	node, err := backend.RegisterLocalNode(handler, false)
	if err != nil {
		t.Fatalf("RegisterLocalNode: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()

	req := api.NewParcel()
	if err := req.WriteFileDescriptor(api.NewFileDescriptor(int(w.Fd()))); err != nil {
		t.Fatalf("WriteFileDescriptor: %v", err)
	}
	if err := req.WriteInt32(2); err != nil {
		t.Fatalf("WriteInt32: %v", err)
	}
	if err := req.WriteString("-a"); err != nil {
		t.Fatalf("WriteString(-a): %v", err)
	}
	if err := req.WriteString("--proto"); err != nil {
		t.Fatalf("WriteString(--proto): %v", err)
	}
	if err := req.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	if _, err := backend.DispatchLocalTransaction(context.Background(), node.ID, api.DumpTransaction, req, 0); err != nil {
		t.Fatalf("DispatchLocalTransaction(dump): %v", err)
	}
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "dump:-a,--proto\n" {
		t.Fatalf("dump output = %q, want dump:-a,--proto", string(data))
	}

	reply, err := backend.DispatchLocalTransaction(context.Background(), node.ID, api.DebugPIDTransaction, api.NewParcel(), 0)
	if err != nil {
		t.Fatalf("DispatchLocalTransaction(debug pid): %v", err)
	}
	pid, err := reply.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32: %v", err)
	}
	if pid != 555 {
		t.Fatalf("debug pid = %d, want 555", pid)
	}
}

type stableTestHandler struct {
	api.StaticHandler
	version int32
	hash    string
}

func (h stableTestHandler) InterfaceVersion() int32 {
	return h.version
}

func (h stableTestHandler) InterfaceHash() string {
	return h.hash
}

type dumpDebugHandler struct {
	api.StaticHandler
	pid int32
}

func (h dumpDebugHandler) Dump(ctx context.Context, fd int, args []string) error {
	_, err := io.WriteString(os.NewFile(uintptr(fd), "dump"), "dump:"+strings.Join(args, ",")+"\n")
	return err
}

func (h dumpDebugHandler) DebugPID() int32 {
	return h.pid
}
