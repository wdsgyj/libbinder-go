//go:build !((linux || android) && (amd64 || arm64))

package kernel

import api "libbinder-go/binder"

func (d *DriverManager) ProtocolVersion() (int32, error) {
	return 0, ErrUnsupportedPlatform
}

func (d *DriverManager) SetMaxThreads(max uint32) error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) WriteRead(bwr *BinderWriteRead) error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) WriteCommand(cmd uint32, payload []byte, readBuf []byte) (BinderWriteRead, error) {
	return BinderWriteRead{}, ErrUnsupportedPlatform
}

func (d *DriverManager) Read(readBuf []byte) (BinderWriteRead, error) {
	return BinderWriteRead{}, ErrUnsupportedPlatform
}

func (d *DriverManager) EnterLooper() error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) FreeBuffer(ptr uintptr) error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) WriteHandleCommand(cmd uint32, handle uint32) error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) AcquireHandle(handle uint32) error {
	return ErrUnsupportedPlatform
}

func (d *DriverManager) TransactHandle(handle uint32, code uint32, payload []byte, flags api.Flags) ([]byte, []uint32, error) {
	return nil, nil, ErrUnsupportedPlatform
}

type BinderWriteRead struct{}
