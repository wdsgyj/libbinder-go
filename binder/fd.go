package binder

import (
	"os"
	"syscall"
)

// FileDescriptor is a small wrapper for a Binder-transferred file descriptor.
type FileDescriptor struct {
	fd    int
	owned bool
}

func NewFileDescriptor(fd int) FileDescriptor {
	return FileDescriptor{fd: fd}
}

func NewOwnedFileDescriptor(fd int) FileDescriptor {
	return FileDescriptor{fd: fd, owned: true}
}

func (f FileDescriptor) FD() int {
	return f.fd
}

func (f FileDescriptor) Owned() bool {
	return f.owned
}

func (f FileDescriptor) File(name string) *os.File {
	if f.fd < 0 {
		return nil
	}
	return os.NewFile(uintptr(f.fd), name)
}

func (f *FileDescriptor) Close() error {
	if f == nil || !f.owned || f.fd < 0 {
		return nil
	}
	err := syscall.Close(f.fd)
	f.fd = -1
	f.owned = false
	return err
}

// ParcelFileDescriptor represents an owned file descriptor transported using
// the AIDL ParcelFileDescriptor wire form.
type ParcelFileDescriptor struct {
	FileDescriptor
}

func NewParcelFileDescriptor(fd int) ParcelFileDescriptor {
	return ParcelFileDescriptor{FileDescriptor: NewOwnedFileDescriptor(fd)}
}
