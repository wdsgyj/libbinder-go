package binder

import (
	"context"
	"fmt"
	"os"
)

// DumpBinder issues the reserved Binder dump transaction against b.
func DumpBinder(ctx context.Context, b Binder, fd FileDescriptor, args []string) error {
	if b == nil {
		return ErrUnsupported
	}
	if fd.FD() < 0 {
		return fmt.Errorf("%w: invalid dump fd %d", ErrBadParcelable, fd.FD())
	}

	data := NewParcel()
	if err := data.WriteFileDescriptor(fd); err != nil {
		return err
	}
	if err := data.WriteInt32(int32(len(args))); err != nil {
		return err
	}
	for _, arg := range args {
		if err := data.WriteString(arg); err != nil {
			return err
		}
	}
	if err := data.SetPosition(0); err != nil {
		return err
	}

	reply, err := b.Transact(ctx, DumpTransaction, data, FlagNone)
	if err != nil {
		return err
	}
	if reply == nil {
		return nil
	}
	return nil
}

// DumpBinderFile opens path and issues the reserved Binder dump transaction.
func DumpBinderFile(ctx context.Context, b Binder, path string, args []string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return DumpBinder(ctx, b, NewFileDescriptor(int(file.Fd())), args)
}

// GetDebugPID queries the reserved Binder debug PID transaction.
func GetDebugPID(ctx context.Context, b Binder) (int32, error) {
	if b == nil {
		return 0, ErrUnsupported
	}

	reply, err := b.Transact(ctx, DebugPIDTransaction, NewParcel(), FlagNone)
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, ErrBadParcelable
	}
	return reply.ReadInt32()
}
