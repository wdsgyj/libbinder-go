//go:build !linux && !android

package kernel

func (d *DriverManager) openPlatform() error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) closePlatform() error {
	d.opened = false
	d.file = nil
	d.mmap = nil
	return nil
}

func (d *DriverManager) ioctlPlatform(req uintptr, arg uintptr) error {
	return ErrUnsupportedPlatform
}
