package binder

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"unicode/utf16"
)

const (
	parcelAlignment = 4
	maxInt          = int(^uint(0) >> 1)
	maxInt32        = int(^uint32(0) >> 1)
	flatObjectSize  = 24
	flatTypeBinder  = uint32('s')<<24 | uint32('b')<<16 | uint32('*')<<8 | 0x85
	flatTypeWBinder = uint32('w')<<24 | uint32('b')<<16 | uint32('*')<<8 | 0x85
	flatTypeHandle  = uint32('s')<<24 | uint32('h')<<16 | uint32('*')<<8 | 0x85

	flatBinderFlagAcceptsFDs = 0x100
	localBinderSchedBits     = 19
	systemStabilityLevel     = 0b001100
)

type parcelFlags uint32

const (
	parcelFlagAllowFDs parcelFlags = 1 << iota
	parcelFlagSensitive
)

type parcelMode uint8

const (
	parcelModeKernel parcelMode = iota
	parcelModeRPC
)

type parcelObjectKind uint8

const (
	parcelObjectBinder parcelObjectKind = iota + 1
	parcelObjectFD
)

type parcelObjectRef struct {
	Kind       parcelObjectKind
	ObjectKind ObjectKind
	Offset     int
	Length     int
	Handle     uint32
}

// ObjectKind identifies a Binder wire object embedded in a Parcel.
type ObjectKind uint8

const (
	ObjectNullBinder ObjectKind = iota
	ObjectStrongBinder
	ObjectWeakBinder
	ObjectFileDescriptor
)

// ParcelObject describes a decoded Binder wire object tracked by the Parcel.
//
// This is primarily useful for low-level runtime code that needs access to the
// Binder object table.
type ParcelObject struct {
	Kind   ObjectKind
	Offset int
	Length int
	Handle uint32
}

// Parcel is the public payload container used for Binder transactions.
//
// The public API stays value-oriented and Go-native, while the internal
// representation tracks the cursor and object table required by Binder's wire
// format. This MVP implements the core scalar/string/[]byte codec and keeps the
// object table reserved for later kernel/RPC work.
type Parcel struct {
	buf     []byte
	pos     int
	objects []parcelObjectRef
	mode    parcelMode
	flags   parcelFlags
}

func NewParcel() *Parcel {
	return &Parcel{
		mode:  parcelModeKernel,
		flags: parcelFlagAllowFDs,
	}
}

func NewParcelBytes(b []byte) *Parcel {
	p := NewParcel()
	p.SetBytes(b)
	return p
}

// NewParcelWire constructs a Parcel from raw Binder payload bytes plus the
// decoded Binder object table.
func NewParcelWire(b []byte, objects []ParcelObject) *Parcel {
	p := NewParcel()
	p.SetWireData(b, objects)
	return p
}

func (p *Parcel) Reset() {
	if p == nil {
		return
	}
	p.buf = p.buf[:0]
	p.pos = 0
	p.objects = nil
}

func (p *Parcel) SetBytes(b []byte) {
	if p == nil {
		return
	}
	p.buf = append(p.buf[:0], b...)
	p.pos = 0
	p.objects = nil
}

// SetWireData replaces the Parcel contents with raw Binder payload bytes plus
// the decoded Binder object table.
func (p *Parcel) SetWireData(b []byte, objects []ParcelObject) {
	if p == nil {
		return
	}

	p.buf = append(p.buf[:0], b...)
	p.pos = 0
	p.objects = p.objects[:0]
	for _, obj := range objects {
		p.objects = append(p.objects, parcelObjectRef{
			Kind:       objectKindToRef(obj.Kind),
			ObjectKind: obj.Kind,
			Offset:     obj.Offset,
			Length:     obj.Length,
			Handle:     obj.Handle,
		})
	}
}

func (p *Parcel) Bytes() []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p.buf...)
}

// KernelWireData returns copies of the Parcel payload bytes and Binder object
// offsets suitable for use with the kernel Binder driver.
func (p *Parcel) KernelWireData() ([]byte, []uint64) {
	if p == nil {
		return nil, nil
	}

	payload := append([]byte(nil), p.buf...)
	if len(p.objects) == 0 {
		return payload, nil
	}

	offsets := make([]uint64, 0, len(p.objects))
	for _, obj := range p.objects {
		offsets = append(offsets, uint64(obj.Offset))
	}
	return payload, offsets
}

// Objects returns a copy of the decoded Binder object table for this Parcel.
func (p *Parcel) Objects() []ParcelObject {
	if p == nil || len(p.objects) == 0 {
		return nil
	}

	objects := make([]ParcelObject, 0, len(p.objects))
	for _, obj := range p.objects {
		objects = append(objects, ParcelObject{
			Kind:   obj.ObjectKind,
			Offset: obj.Offset,
			Length: obj.Length,
			Handle: obj.Handle,
		})
	}
	return objects
}

func (p *Parcel) Len() int {
	if p == nil {
		return 0
	}
	return len(p.buf)
}

func (p *Parcel) Position() int {
	if p == nil {
		return 0
	}
	return p.pos
}

func (p *Parcel) Remaining() int {
	if p == nil || p.pos >= len(p.buf) {
		return 0
	}
	if p.pos < 0 {
		return len(p.buf)
	}
	return len(p.buf) - p.pos
}

func (p *Parcel) SetPosition(pos int) error {
	if p == nil {
		return ErrBadParcelable
	}
	if pos < 0 || pos > len(p.buf) {
		return fmt.Errorf("%w: invalid parcel position %d", ErrBadParcelable, pos)
	}
	p.pos = pos
	return nil
}

func (p *Parcel) WriteInt32(v int32) error {
	return p.writeBlock(4, func(dst []byte) {
		binary.LittleEndian.PutUint32(dst, uint32(v))
	})
}

func (p *Parcel) WriteUint32(v uint32) error {
	return p.writeBlock(4, func(dst []byte) {
		binary.LittleEndian.PutUint32(dst, v)
	})
}

func (p *Parcel) WriteInt64(v int64) error {
	return p.writeBlock(8, func(dst []byte) {
		binary.LittleEndian.PutUint64(dst, uint64(v))
	})
}

func (p *Parcel) WriteUint64(v uint64) error {
	return p.writeBlock(8, func(dst []byte) {
		binary.LittleEndian.PutUint64(dst, v)
	})
}

func (p *Parcel) WriteBool(v bool) error {
	if v {
		return p.WriteInt32(1)
	}
	return p.WriteInt32(0)
}

// WriteByte writes an AIDL byte value.
func (p *Parcel) WriteByte(v int8) error {
	return p.writeBlock(1, func(dst []byte) {
		dst[0] = byte(v)
	})
}

// WriteChar writes an AIDL char value.
func (p *Parcel) WriteChar(v uint16) error {
	return p.writeBlock(2, func(dst []byte) {
		binary.LittleEndian.PutUint16(dst, v)
	})
}

// WriteFloat32 writes an AIDL float value.
func (p *Parcel) WriteFloat32(v float32) error {
	return p.writeBlock(4, func(dst []byte) {
		binary.LittleEndian.PutUint32(dst, math.Float32bits(v))
	})
}

// WriteFloat64 writes an AIDL double value.
func (p *Parcel) WriteFloat64(v float64) error {
	return p.writeBlock(8, func(dst []byte) {
		binary.LittleEndian.PutUint64(dst, math.Float64bits(v))
	})
}

func (p *Parcel) WriteString(v string) error {
	return p.WriteNullableString(&v)
}

func (p *Parcel) WriteNullableString(v *string) error {
	if v == nil {
		return p.WriteInt32(-1)
	}

	units := utf16.Encode([]rune(*v))
	if len(units) > maxInt32 {
		return fmt.Errorf("%w: string too large: %d code units", ErrBadParcelable, len(units))
	}
	if len(units) > (maxInt/2)-1 {
		return fmt.Errorf("%w: string payload overflow", ErrBadParcelable)
	}
	if err := p.WriteInt32(int32(len(units))); err != nil {
		return err
	}

	return p.writeBlock((len(units)+1)*2, func(dst []byte) {
		for i, unit := range units {
			binary.LittleEndian.PutUint16(dst[i*2:], unit)
		}
		binary.LittleEndian.PutUint16(dst[len(units)*2:], 0)
	})
}

func (p *Parcel) WriteBytes(b []byte) error {
	if b == nil {
		return p.WriteInt32(-1)
	}
	if len(b) > maxInt32 {
		return fmt.Errorf("%w: byte slice too large: %d", ErrBadParcelable, len(b))
	}
	if err := p.WriteInt32(int32(len(b))); err != nil {
		return err
	}
	return p.writeBlock(len(b), func(dst []byte) {
		copy(dst, b)
	})
}

// WriteInterfaceToken writes the standard Android kernel Binder request header
// followed by the interface descriptor.
func (p *Parcel) WriteInterfaceToken(descriptor string) error {
	if err := p.WriteUint32(1 << 31); err != nil {
		return err
	}
	if err := p.WriteInt32(-1); err != nil {
		return err
	}
	if err := p.WriteUint32(packChars('S', 'Y', 'S', 'T')); err != nil {
		return err
	}
	return p.WriteString(descriptor)
}

// WriteStrongBinderLocal writes a local Binder object for kernel Binder IPC.
// This is a low-level helper used by runtime code that is registering or
// passing process-local Binder nodes.
func (p *Parcel) WriteStrongBinderLocal(ptr, cookie uintptr) error {
	if p == nil {
		return ErrBadParcelable
	}

	start := p.pos
	if err := p.writeBlock(flatObjectSize, func(dst []byte) {
		binary.LittleEndian.PutUint32(dst[0:], flatTypeBinder)
		binary.LittleEndian.PutUint32(dst[4:], flatBinderFlagAcceptsFDs|localBinderSchedBits)
		binary.LittleEndian.PutUint64(dst[8:], uint64(ptr))
		binary.LittleEndian.PutUint64(dst[16:], uint64(cookie))
	}); err != nil {
		return err
	}
	if err := p.WriteInt32(systemStabilityLevel); err != nil {
		return err
	}

	p.objects = append(p.objects, parcelObjectRef{
		Kind:       parcelObjectBinder,
		ObjectKind: ObjectStrongBinder,
		Offset:     start,
		Length:     flatObjectSize,
	})
	return nil
}

// WriteNullStrongBinder writes a nullable strong Binder object with a nil
// value using the kernel Binder wire format.
func (p *Parcel) WriteNullStrongBinder() error {
	if p == nil {
		return ErrBadParcelable
	}

	if err := p.writeBlock(flatObjectSize, func(dst []byte) {
		binary.LittleEndian.PutUint32(dst[0:], flatTypeBinder)
	}); err != nil {
		return err
	}
	return p.WriteInt32(0)
}

func (p *Parcel) ReadInt32() (int32, error) {
	block, err := p.readBlock(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(block)), nil
}

func (p *Parcel) ReadUint32() (uint32, error) {
	block, err := p.readBlock(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(block), nil
}

func (p *Parcel) ReadInt64() (int64, error) {
	block, err := p.readBlock(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(block)), nil
}

func (p *Parcel) ReadUint64() (uint64, error) {
	block, err := p.readBlock(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(block), nil
}

func (p *Parcel) ReadBool() (bool, error) {
	v, err := p.ReadInt32()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// ReadByte reads an AIDL byte value.
func (p *Parcel) ReadByte() (int8, error) {
	block, err := p.readBlock(1)
	if err != nil {
		return 0, err
	}
	return int8(block[0]), nil
}

// ReadChar reads an AIDL char value.
func (p *Parcel) ReadChar() (uint16, error) {
	block, err := p.readBlock(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(block), nil
}

// ReadFloat32 reads an AIDL float value.
func (p *Parcel) ReadFloat32() (float32, error) {
	block, err := p.readBlock(4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(block)), nil
}

// ReadFloat64 reads an AIDL double value.
func (p *Parcel) ReadFloat64() (float64, error) {
	block, err := p.readBlock(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(block)), nil
}

func (p *Parcel) ReadString() (string, error) {
	s, err := p.ReadNullableString()
	if err != nil {
		return "", err
	}
	if s == nil {
		return "", fmt.Errorf("%w: unexpected null string", ErrBadParcelable)
	}
	return *s, nil
}

func (p *Parcel) ReadNullableString() (*string, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if size < 0 {
		return nil, nil
	}

	unitCount := int(size)
	if unitCount > (maxInt/2)-1 {
		return nil, fmt.Errorf("%w: string payload overflow", ErrBadParcelable)
	}

	block, err := p.readBlock((unitCount + 1) * 2)
	if err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint16(block[unitCount*2:]) != 0 {
		return nil, fmt.Errorf("%w: unterminated UTF-16 string", ErrBadParcelable)
	}

	units := make([]uint16, unitCount)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(block[i*2:])
	}
	if err := validateUTF16(units); err != nil {
		return nil, err
	}

	value := string(utf16.Decode(units))
	return &value, nil
}

func (p *Parcel) ReadBytes() ([]byte, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if size < 0 {
		return nil, nil
	}

	block, err := p.readBlock(int(size))
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(block))
	copy(out, block)
	return out, nil
}

// ReadObject reads the next Binder wire object from the current Parcel
// position. A null Binder object is returned with Kind ObjectNullBinder.
func (p *Parcel) ReadObject() (*ParcelObject, error) {
	if p == nil {
		return nil, ErrBadParcelable
	}

	start := p.pos
	if ref, ok := p.objectAt(start); ok {
		if _, err := p.readBlock(ref.Length); err != nil {
			return nil, err
		}
		obj := ParcelObject{
			Kind:   ref.ObjectKind,
			Offset: ref.Offset,
			Length: ref.Length,
			Handle: ref.Handle,
		}
		return &obj, nil
	}

	block, err := p.readBlock(flatObjectSize)
	if err != nil {
		return nil, err
	}
	if !isNullBinderBlock(block) {
		return nil, fmt.Errorf("%w: expected parcel object at offset %d", ErrBadParcelable, start)
	}

	return &ParcelObject{
		Kind:   ObjectNullBinder,
		Offset: start,
		Length: len(block),
	}, nil
}

// ReadStrongBinderHandle reads the next strong Binder object and returns its
// remote handle. A null strong Binder returns nil without error.
func (p *Parcel) ReadStrongBinderHandle() (*uint32, error) {
	obj, err := p.ReadObject()
	if err != nil {
		return nil, err
	}
	if _, err := p.ReadInt32(); err != nil {
		return nil, err
	}
	if obj == nil || obj.Kind == ObjectNullBinder {
		return nil, nil
	}
	if obj.Kind != ObjectStrongBinder {
		return nil, fmt.Errorf("%w: expected strong binder object, got %d", ErrBadParcelable, obj.Kind)
	}

	handle := obj.Handle
	return &handle, nil
}

func (p *Parcel) writeBlock(payloadLen int, fill func([]byte)) error {
	paddedLen, err := padSize(payloadLen)
	if err != nil {
		return err
	}

	block, err := p.reserve(paddedLen)
	if err != nil {
		return err
	}
	if payloadLen > 0 {
		fill(block[:payloadLen])
	}
	clear(block[payloadLen:])
	return nil
}

func (p *Parcel) readBlock(payloadLen int) ([]byte, error) {
	if p == nil {
		return nil, ErrBadParcelable
	}
	if payloadLen < 0 {
		return nil, fmt.Errorf("%w: negative read size %d", ErrBadParcelable, payloadLen)
	}
	if p.pos < 0 || p.pos > len(p.buf) {
		return nil, fmt.Errorf("%w: invalid parcel position %d", ErrBadParcelable, p.pos)
	}

	paddedLen, err := padSize(payloadLen)
	if err != nil {
		return nil, err
	}
	if p.pos > len(p.buf)-paddedLen {
		return nil, io.ErrUnexpectedEOF
	}

	start := p.pos
	p.pos += paddedLen
	return p.buf[start : start+payloadLen], nil
}

func (p *Parcel) reserve(n int) ([]byte, error) {
	if p == nil {
		return nil, ErrBadParcelable
	}
	if n < 0 {
		return nil, fmt.Errorf("%w: negative reserve size %d", ErrBadParcelable, n)
	}
	if p.pos < 0 || p.pos > len(p.buf) {
		return nil, fmt.Errorf("%w: invalid parcel position %d", ErrBadParcelable, p.pos)
	}
	if p.pos > maxInt-n {
		return nil, fmt.Errorf("%w: parcel size overflow", ErrBadParcelable)
	}

	end := p.pos + n
	if end > len(p.buf) {
		p.buf = append(p.buf, make([]byte, end-len(p.buf))...)
	}

	block := p.buf[p.pos:end]
	p.pos = end
	return block, nil
}

func padSize(n int) (int, error) {
	if n < 0 {
		return 0, fmt.Errorf("%w: negative size %d", ErrBadParcelable, n)
	}
	if n > maxInt-(parcelAlignment-1) {
		return 0, fmt.Errorf("%w: parcel size overflow", ErrBadParcelable)
	}
	return (n + parcelAlignment - 1) &^ (parcelAlignment - 1), nil
}

func validateUTF16(units []uint16) error {
	for i := 0; i < len(units); i++ {
		unit := units[i]
		switch {
		case 0xD800 <= unit && unit <= 0xDBFF:
			if i+1 >= len(units) {
				return fmt.Errorf("%w: unterminated UTF-16 surrogate pair", ErrBadParcelable)
			}
			next := units[i+1]
			if next < 0xDC00 || next > 0xDFFF {
				return fmt.Errorf("%w: invalid UTF-16 surrogate pair", ErrBadParcelable)
			}
			i++
		case 0xDC00 <= unit && unit <= 0xDFFF:
			return fmt.Errorf("%w: unexpected UTF-16 low surrogate", ErrBadParcelable)
		}
	}
	return nil
}

func (p *Parcel) objectAt(offset int) (parcelObjectRef, bool) {
	for _, obj := range p.objects {
		if obj.Offset == offset {
			return obj, true
		}
	}
	return parcelObjectRef{}, false
}

func objectKindToRef(kind ObjectKind) parcelObjectKind {
	switch kind {
	case ObjectStrongBinder, ObjectWeakBinder, ObjectNullBinder:
		return parcelObjectBinder
	case ObjectFileDescriptor:
		return parcelObjectFD
	default:
		return parcelObjectBinder
	}
}

func isNullBinderBlock(block []byte) bool {
	if len(block) != flatObjectSize {
		return false
	}

	typ := binary.LittleEndian.Uint32(block[0:4])
	if typ != flatTypeBinder && typ != flatTypeWBinder {
		return false
	}

	return binary.LittleEndian.Uint64(block[8:16]) == 0 &&
		binary.LittleEndian.Uint64(block[16:24]) == 0
}

func isStrongHandleBlock(block []byte) bool {
	return len(block) == flatObjectSize && binary.LittleEndian.Uint32(block[0:4]) == flatTypeHandle
}

func packChars(c1, c2, c3, c4 byte) uint32 {
	return uint32(c1)<<24 | uint32(c2)<<16 | uint32(c3)<<8 | uint32(c4)
}
