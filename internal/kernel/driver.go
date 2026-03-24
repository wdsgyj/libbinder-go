package kernel

import (
	"errors"
	"os"
)

const defaultMmapSize = 1 << 20

var (
	ErrDriverClosed        = errors.New("kernel: binder driver is not open")
	ErrUnsupportedPlatform = errors.New("kernel: binder backend is unsupported on this platform")
	ErrNoClientWorker      = errors.New("kernel: no client worker is available")
	ErrParcelObjects       = errors.New("kernel: parcel object transport is not implemented yet")
	ErrDeadReply           = errors.New("kernel: binder dead reply")
	ErrFailedReply         = errors.New("kernel: binder failed reply")
)

// DriverManager owns the binder device path and manages binder fd/mmap lifecycle.
type DriverManager struct {
	Path     string
	MmapSize int

	file   *os.File
	mmap   []byte
	opened bool
}

func NewDriverManager(path string) *DriverManager {
	return &DriverManager{
		Path:     path,
		MmapSize: defaultMmapSize,
	}
}

func (d *DriverManager) Open() error {
	if d.opened {
		return nil
	}
	return d.openPlatform()
}

func (d *DriverManager) Close() error {
	if !d.opened {
		return nil
	}
	return d.closePlatform()
}

func (d *DriverManager) IsOpen() bool {
	return d.opened
}

func (d *DriverManager) FD() uintptr {
	if d.file == nil {
		return 0
	}
	return d.file.Fd()
}

func (d *DriverManager) Mmap() []byte {
	return d.mmap
}

func (d *DriverManager) Ioctl(req uintptr, arg uintptr) error {
	if !d.opened || d.file == nil {
		return ErrDriverClosed
	}
	return d.ioctlPlatform(req, arg)
}
