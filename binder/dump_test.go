package binder

import (
	"context"
	"io"
	"os"
	"testing"
)

func TestDumpBinderAndGetDebugPID(t *testing.T) {
	var gotArgs []string
	b := dumpQueryBinder{
		dump: func(ctx context.Context, fd int, args []string) error {
			gotArgs = append([]string(nil), args...)
			_, err := io.WriteString(os.NewFile(uintptr(fd), "dump"), "hello\n")
			return err
		},
		pid: 321,
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()

	if err := DumpBinder(context.Background(), b, NewFileDescriptor(int(w.Fd())), []string{"-a", "--proto"}); err != nil {
		t.Fatalf("DumpBinder: %v", err)
	}
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("dump output = %q, want hello", string(data))
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-a" || gotArgs[1] != "--proto" {
		t.Fatalf("dump args = %#v, want [-a --proto]", gotArgs)
	}

	pid, err := GetDebugPID(context.Background(), b)
	if err != nil {
		t.Fatalf("GetDebugPID: %v", err)
	}
	if pid != 321 {
		t.Fatalf("GetDebugPID = %d, want 321", pid)
	}
}

type dumpQueryBinder struct {
	dump func(context.Context, int, []string) error
	pid  int32
}

func (b dumpQueryBinder) Descriptor(ctx context.Context) (string, error) { return "dump", nil }
func (b dumpQueryBinder) WatchDeath(ctx context.Context) (Subscription, error) {
	return nil, ErrUnsupported
}
func (b dumpQueryBinder) Close() error { return nil }

func (b dumpQueryBinder) Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error) {
	switch code {
	case DumpTransaction:
		fd, args, err := readDumpRequest(data)
		if err != nil {
			return nil, err
		}
		if b.dump != nil {
			if err := b.dump(ctx, fd.FD(), args); err != nil {
				return nil, err
			}
		}
		return NewParcel(), nil
	case DebugPIDTransaction:
		reply := NewParcel()
		if err := reply.WriteInt32(b.pid); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	default:
		return nil, ErrUnknownTransaction
	}
}
