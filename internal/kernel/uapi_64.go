//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"fmt"
	"unsafe"
)

const (
	BinderCurrentProtocolVersion int32 = 8

	TFOneWay   uint32 = 0x01
	TFRootObj  uint32 = 0x04
	TFStatus   uint32 = 0x08
	TFAcceptFD uint32 = 0x10
	TFClearBuf uint32 = 0x20
	TFUpdateTX uint32 = 0x40
)

const (
	iocNRBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14
	iocDirBits  = 2

	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits

	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

const (
	BINDERWriteRead     = uintptr((iocRead|iocWrite)<<iocDirShift) | uintptr('b')<<iocTypeShift | uintptr(1)<<iocNRShift | uintptr(unsafe.Sizeof(BinderWriteRead{}))<<iocSizeShift
	BINDERSetMaxThreads = uintptr(iocWrite<<iocDirShift) | uintptr('b')<<iocTypeShift | uintptr(5)<<iocNRShift | uintptr(unsafe.Sizeof(uint32(0)))<<iocSizeShift
	BINDERVersion       = uintptr((iocRead|iocWrite)<<iocDirShift) | uintptr('b')<<iocTypeShift | uintptr(9)<<iocNRShift | uintptr(unsafe.Sizeof(BinderVersionInfo{}))<<iocSizeShift
)

const (
	BCTransaction         = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(0)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionData{}))<<iocSizeShift)
	BCReply               = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(1)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionData{}))<<iocSizeShift)
	BCFreeBuffer          = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(3)<<iocNRShift | uintptr(unsafe.Sizeof(uint64(0)))<<iocSizeShift)
	BCIncRefs             = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(4)<<iocNRShift | uintptr(unsafe.Sizeof(uint32(0)))<<iocSizeShift)
	BCAcquire             = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(5)<<iocNRShift | uintptr(unsafe.Sizeof(uint32(0)))<<iocSizeShift)
	BCRelease             = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(6)<<iocNRShift | uintptr(unsafe.Sizeof(uint32(0)))<<iocSizeShift)
	BCDecRefs             = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(7)<<iocNRShift | uintptr(unsafe.Sizeof(uint32(0)))<<iocSizeShift)
	BCIncRefsDone         = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(8)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BCAcquireDone         = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(9)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BCRegisterLooper      = uint32(uintptr(iocNone)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(11)<<iocNRShift)
	BCEnterLooper         = uint32(uintptr(iocNone)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(12)<<iocNRShift)
	BCExitLooper          = uint32(uintptr(iocNone)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(13)<<iocNRShift)
	BCTransactionSG       = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(17)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionDataSG{}))<<iocSizeShift)
	BCReplySG             = uint32(uintptr(iocWrite)<<iocDirShift | uintptr('c')<<iocTypeShift | uintptr(18)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionDataSG{}))<<iocSizeShift)
	BRTransaction         = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(2)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionData{}))<<iocSizeShift)
	BRNoop                = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(12)<<iocNRShift)
	BRTransactionComplete = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(6)<<iocNRShift)
	BRReply               = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(3)<<iocNRShift | uintptr(unsafe.Sizeof(BinderTransactionData{}))<<iocSizeShift)
	BRIncRefs             = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(7)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BRAcquire             = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(8)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BRRelease             = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(9)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BRDecRefs             = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(10)<<iocNRShift | uintptr(unsafe.Sizeof(BinderPtrCookie{}))<<iocSizeShift)
	BRSpawnLooper         = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(13)<<iocNRShift)
	BRFinished            = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(14)<<iocNRShift)
	BRError               = uint32(uintptr(iocRead)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(0)<<iocNRShift | uintptr(unsafe.Sizeof(int32(0)))<<iocSizeShift)
	BRDeadReply           = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(5)<<iocNRShift)
	BRFailedReply         = uint32(uintptr(iocNone)<<iocDirShift | uintptr('r')<<iocTypeShift | uintptr(17)<<iocNRShift)
)

const (
	BinderTypeBinder     = uint32('s')<<24 | uint32('b')<<16 | uint32('*')<<8 | 0x85
	BinderTypeWeakBinder = uint32('w')<<24 | uint32('b')<<16 | uint32('*')<<8 | 0x85
	BinderTypeHandle     = uint32('s')<<24 | uint32('h')<<16 | uint32('*')<<8 | 0x85
	BinderTypeWeakHandle = uint32('w')<<24 | uint32('h')<<16 | uint32('*')<<8 | 0x85
	BinderTypeFD         = uint32('f')<<24 | uint32('d')<<16 | uint32('*')<<8 | 0x85
)

const (
	binderTransactionDataSize = int(unsafe.Sizeof(BinderTransactionData{}))
)

// BinderWriteRead mirrors struct binder_write_read for 64-bit linux/android Binder.
type BinderWriteRead struct {
	WriteSize     uint64
	WriteConsumed uint64
	WriteBuffer   uint64
	ReadSize      uint64
	ReadConsumed  uint64
	ReadBuffer    uint64
}

// BinderVersionInfo mirrors struct binder_version.
type BinderVersionInfo struct {
	ProtocolVersion int32
}

// BinderTransactionData mirrors struct binder_transaction_data on 64-bit linux/android.
type BinderTransactionData struct {
	Target      uint64
	Cookie      uint64
	Code        uint32
	Flags       uint32
	SenderPID   int32
	SenderEUID  uint32
	DataSize    uint64
	OffsetsSize uint64
	DataBuffer  uint64
	DataOffsets uint64
}

// BinderTransactionDataSG mirrors struct binder_transaction_data_sg.
type BinderTransactionDataSG struct {
	Transaction BinderTransactionData
	BuffersSize uint64
}

// BinderPtrCookie mirrors struct binder_ptr_cookie.
type BinderPtrCookie struct {
	Ptr    uint64
	Cookie uint64
}

func (t *BinderTransactionData) SetTargetHandle(handle uint32) {
	if t == nil {
		return
	}
	t.Target = uint64(handle)
}

func (t BinderTransactionData) TargetHandle() uint32 {
	return uint32(t.Target)
}

func (t BinderTransactionData) BufferPointer() uintptr {
	return uintptr(t.DataBuffer)
}

func (t BinderTransactionData) TargetPointer() uintptr {
	return uintptr(t.Target)
}

func (t BinderTransactionData) CookiePointer() uintptr {
	return uintptr(t.Cookie)
}

func (t *BinderTransactionData) MarshalBinary() []byte {
	if t == nil {
		return nil
	}

	out := make([]byte, binderTransactionDataSize)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(t)), binderTransactionDataSize))
	return out
}

func (t *BinderTransactionData) UnmarshalBinary(data []byte) error {
	if t == nil {
		return fmt.Errorf("kernel: nil BinderTransactionData")
	}
	if len(data) < binderTransactionDataSize {
		return fmt.Errorf("kernel: short binder transaction data: have %d want %d", len(data), binderTransactionDataSize)
	}

	copy(unsafe.Slice((*byte)(unsafe.Pointer(t)), binderTransactionDataSize), data[:binderTransactionDataSize])
	return nil
}
