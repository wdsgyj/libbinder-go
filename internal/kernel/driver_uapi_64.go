//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

func (d *DriverManager) ProtocolVersion() (int32, error) {
	if !d.opened || d.file == nil {
		return 0, ErrDriverClosed
	}

	var version BinderVersionInfo
	if err := d.Ioctl(BINDERVersion, uintptr(unsafe.Pointer(&version))); err != nil {
		return 0, err
	}
	return version.ProtocolVersion, nil
}

func (d *DriverManager) SetMaxThreads(max uint32) error {
	if !d.opened || d.file == nil {
		return ErrDriverClosed
	}

	return d.Ioctl(BINDERSetMaxThreads, uintptr(unsafe.Pointer(&max)))
}

func (d *DriverManager) WriteRead(bwr *BinderWriteRead) error {
	if bwr == nil {
		return fmt.Errorf("kernel: nil BinderWriteRead")
	}

	for {
		err := d.Ioctl(BINDERWriteRead, uintptr(unsafe.Pointer(bwr)))
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		return err
	}
}

func (d *DriverManager) WriteCommand(cmd uint32, payload []byte, readBuf []byte) (BinderWriteRead, error) {
	if !d.opened || d.file == nil {
		return BinderWriteRead{}, ErrDriverClosed
	}

	writeBuf := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(writeBuf[:4], cmd)
	copy(writeBuf[4:], payload)

	bwr := BinderWriteRead{
		WriteSize:   uint64(len(writeBuf)),
		WriteBuffer: uint64(uintptr(unsafe.Pointer(&writeBuf[0]))),
	}
	if len(readBuf) > 0 {
		bwr.ReadSize = uint64(len(readBuf))
		bwr.ReadBuffer = uint64(uintptr(unsafe.Pointer(&readBuf[0])))
	}

	err := d.WriteRead(&bwr)
	runtime.KeepAlive(writeBuf)
	runtime.KeepAlive(readBuf)
	return bwr, err
}

func (d *DriverManager) Read(readBuf []byte) (BinderWriteRead, error) {
	if !d.opened || d.file == nil {
		return BinderWriteRead{}, ErrDriverClosed
	}
	if len(readBuf) == 0 {
		return BinderWriteRead{}, fmt.Errorf("kernel: empty read buffer")
	}

	bwr := BinderWriteRead{
		ReadSize:   uint64(len(readBuf)),
		ReadBuffer: uint64(uintptr(unsafe.Pointer(&readBuf[0]))),
	}
	err := d.WriteRead(&bwr)
	runtime.KeepAlive(readBuf)
	return bwr, err
}

func (d *DriverManager) EnterLooper() error {
	bwr, err := d.WriteCommand(BCEnterLooper, nil, nil)
	if err != nil {
		return err
	}
	if want := uint64(4); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: BC_ENTER_LOOPER consumed %d bytes, want %d", bwr.WriteConsumed, want)
	}
	return nil
}

func (d *DriverManager) FreeBuffer(ptr uintptr) error {
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint64(payload, uint64(ptr))

	bwr, err := d.WriteCommand(BCFreeBuffer, payload, nil)
	if err != nil {
		return err
	}
	if want := uint64(4 + len(payload)); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: BC_FREE_BUFFER consumed %d bytes, want %d", bwr.WriteConsumed, want)
	}
	return nil
}

func (d *DriverManager) WriteHandleCommand(cmd uint32, handle uint32) error {
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, handle)

	bwr, err := d.WriteCommand(cmd, payload, nil)
	if err != nil {
		return err
	}
	if want := uint64(4 + len(payload)); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: handle command %#x consumed %d bytes, want %d", cmd, bwr.WriteConsumed, want)
	}
	return nil
}

func (d *DriverManager) AcquireHandle(handle uint32) error {
	if err := d.WriteHandleCommand(BCIncRefs, handle); err != nil {
		return err
	}
	return d.WriteHandleCommand(BCAcquire, handle)
}
