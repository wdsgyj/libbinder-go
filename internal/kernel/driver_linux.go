//go:build linux || android

package kernel

import (
	"os"
	"syscall"
)

func (d *DriverManager) openPlatform() error {
	f, err := os.OpenFile(d.Path, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	mmapSize := d.MmapSize
	if mmapSize <= 0 {
		mmapSize = defaultMmapSize
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, mmapSize, syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		_ = f.Close()
		return err
	}

	d.file = f
	d.mmap = data
	d.opened = true
	return nil
}

func (d *DriverManager) closePlatform() error {
	var err error

	if d.mmap != nil {
		if unmapErr := syscall.Munmap(d.mmap); unmapErr != nil && err == nil {
			err = unmapErr
		}
		d.mmap = nil
	}

	if d.file != nil {
		if closeErr := d.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		d.file = nil
	}

	d.opened = false
	return err
}

func (d *DriverManager) ioctlPlatform(req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, d.file.Fd(), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
