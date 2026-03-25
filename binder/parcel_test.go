package binder

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

func TestParcelScalarsRoundTrip(t *testing.T) {
	p := NewParcel()

	if err := p.WriteInt32(-7); err != nil {
		t.Fatalf("WriteInt32: %v", err)
	}
	if err := p.WriteUint32(9); err != nil {
		t.Fatalf("WriteUint32: %v", err)
	}
	if err := p.WriteInt64(-11); err != nil {
		t.Fatalf("WriteInt64: %v", err)
	}
	if err := p.WriteUint64(13); err != nil {
		t.Fatalf("WriteUint64: %v", err)
	}
	if err := p.WriteBool(true); err != nil {
		t.Fatalf("WriteBool(true): %v", err)
	}
	if err := p.WriteBool(false); err != nil {
		t.Fatalf("WriteBool(false): %v", err)
	}
	if err := p.WriteByte(-12); err != nil {
		t.Fatalf("WriteByte: %v", err)
	}
	if err := p.WriteChar('A'); err != nil {
		t.Fatalf("WriteChar: %v", err)
	}
	if err := p.WriteFloat32(3.5); err != nil {
		t.Fatalf("WriteFloat32: %v", err)
	}
	if err := p.WriteFloat64(-9.25); err != nil {
		t.Fatalf("WriteFloat64: %v", err)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	gotI32, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32: %v", err)
	}
	if gotI32 != -7 {
		t.Fatalf("ReadInt32 = %d, want -7", gotI32)
	}

	gotU32, err := p.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32: %v", err)
	}
	if gotU32 != 9 {
		t.Fatalf("ReadUint32 = %d, want 9", gotU32)
	}

	gotI64, err := p.ReadInt64()
	if err != nil {
		t.Fatalf("ReadInt64: %v", err)
	}
	if gotI64 != -11 {
		t.Fatalf("ReadInt64 = %d, want -11", gotI64)
	}

	gotU64, err := p.ReadUint64()
	if err != nil {
		t.Fatalf("ReadUint64: %v", err)
	}
	if gotU64 != 13 {
		t.Fatalf("ReadUint64 = %d, want 13", gotU64)
	}

	gotTrue, err := p.ReadBool()
	if err != nil {
		t.Fatalf("ReadBool(true): %v", err)
	}
	if !gotTrue {
		t.Fatal("ReadBool(true) = false, want true")
	}

	gotFalse, err := p.ReadBool()
	if err != nil {
		t.Fatalf("ReadBool(false): %v", err)
	}
	if gotFalse {
		t.Fatal("ReadBool(false) = true, want false")
	}
	gotByte, err := p.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if gotByte != -12 {
		t.Fatalf("ReadByte = %d, want -12", gotByte)
	}

	gotChar, err := p.ReadChar()
	if err != nil {
		t.Fatalf("ReadChar: %v", err)
	}
	if gotChar != 'A' {
		t.Fatalf("ReadChar = %d, want %d", gotChar, 'A')
	}

	gotF32, err := p.ReadFloat32()
	if err != nil {
		t.Fatalf("ReadFloat32: %v", err)
	}
	if math.Abs(float64(gotF32-3.5)) > 1e-6 {
		t.Fatalf("ReadFloat32 = %f, want 3.5", gotF32)
	}

	gotF64, err := p.ReadFloat64()
	if err != nil {
		t.Fatalf("ReadFloat64: %v", err)
	}
	if math.Abs(gotF64-(-9.25)) > 1e-9 {
		t.Fatalf("ReadFloat64 = %f, want -9.25", gotF64)
	}

	if remaining := p.Remaining(); remaining != 0 {
		t.Fatalf("Remaining = %d, want 0", remaining)
	}
}

func TestParcelStringWireFormat(t *testing.T) {
	p := NewParcel()

	if err := p.WriteString("A🙂"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}

	want := []byte{
		0x03, 0x00, 0x00, 0x00,
		0x41, 0x00,
		0x3d, 0xd8,
		0x42, 0xde,
		0x00, 0x00,
	}
	if got := p.Bytes(); string(got) != string(want) {
		t.Fatalf("Bytes = %v, want %v", got, want)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := p.ReadString()
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if got != "A🙂" {
		t.Fatalf("ReadString = %q, want %q", got, "A🙂")
	}
}

func TestParcelBytesNilEmptyAndAligned(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		p := NewParcel()
		if err := p.WriteBytes(nil); err != nil {
			t.Fatalf("WriteBytes(nil): %v", err)
		}
		if got := p.Bytes(); string(got) != string([]byte{0xff, 0xff, 0xff, 0xff}) {
			t.Fatalf("Bytes = %v, want [-1 length prefix]", got)
		}

		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := p.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes(nil): %v", err)
		}
		if got != nil {
			t.Fatalf("ReadBytes(nil) = %v, want nil", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		p := NewParcel()
		if err := p.WriteBytes([]byte{}); err != nil {
			t.Fatalf("WriteBytes(empty): %v", err)
		}
		if got := p.Bytes(); string(got) != string([]byte{0x00, 0x00, 0x00, 0x00}) {
			t.Fatalf("Bytes = %v, want [0 length prefix]", got)
		}

		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := p.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes(empty): %v", err)
		}
		if got == nil {
			t.Fatal("ReadBytes(empty) = nil, want empty slice")
		}
		if len(got) != 0 {
			t.Fatalf("ReadBytes(empty) len = %d, want 0", len(got))
		}
	})

	t.Run("aligned", func(t *testing.T) {
		p := NewParcel()
		if err := p.WriteBytes([]byte{1, 2, 3}); err != nil {
			t.Fatalf("WriteBytes(aligned): %v", err)
		}

		want := []byte{0x03, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x00}
		if got := p.Bytes(); string(got) != string(want) {
			t.Fatalf("Bytes = %v, want %v", got, want)
		}

		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := p.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes(aligned): %v", err)
		}
		if string(got) != string([]byte{1, 2, 3}) {
			t.Fatalf("ReadBytes(aligned) = %v, want [1 2 3]", got)
		}
	})
}

func TestParcelPositionAndOverwrite(t *testing.T) {
	p := NewParcel()

	if err := p.WriteInt32(1); err != nil {
		t.Fatalf("WriteInt32(first): %v", err)
	}
	if err := p.WriteInt32(2); err != nil {
		t.Fatalf("WriteInt32(second): %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition(rewind): %v", err)
	}
	if err := p.WriteInt32(9); err != nil {
		t.Fatalf("WriteInt32(overwrite): %v", err)
	}

	want := []byte{
		0x09, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x00,
	}
	if got := p.Bytes(); string(got) != string(want) {
		t.Fatalf("Bytes = %v, want %v", got, want)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition(reset): %v", err)
	}
	first, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32(first): %v", err)
	}
	second, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32(second): %v", err)
	}
	if first != 9 || second != 2 {
		t.Fatalf("ReadInt32 values = (%d, %d), want (9, 2)", first, second)
	}
}

func TestParcelReadErrors(t *testing.T) {
	t.Run("unexpected EOF", func(t *testing.T) {
		p := NewParcelBytes([]byte{0x01, 0x00, 0x00, 0x00})
		if _, err := p.ReadInt64(); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("ReadInt64 error = %v, want io.ErrUnexpectedEOF", err)
		}
	})

	t.Run("invalid string terminator", func(t *testing.T) {
		p := NewParcelBytes([]byte{
			0x01, 0x00, 0x00, 0x00,
			0x41, 0x00,
			0x01, 0x00,
		})
		if _, err := p.ReadString(); !errors.Is(err, ErrBadParcelable) {
			t.Fatalf("ReadString error = %v, want ErrBadParcelable", err)
		}
	})
}

func TestParcelReadStrongBinderHandle(t *testing.T) {
	t.Run("object table handle", func(t *testing.T) {
		payload := make([]byte, flatObjectSize+4)
		p := NewParcelWire(payload, []ParcelObject{{
			Kind:   ObjectStrongBinder,
			Offset: 0,
			Length: flatObjectSize,
			Handle: 42,
		}})

		handle, err := p.ReadStrongBinderHandle()
		if err != nil {
			t.Fatalf("ReadStrongBinderHandle: %v", err)
		}
		if handle == nil {
			t.Fatal("ReadStrongBinderHandle = nil, want 42")
		}
		if *handle != 42 {
			t.Fatalf("ReadStrongBinderHandle = %d, want 42", *handle)
		}
	})

	t.Run("null binder", func(t *testing.T) {
		payload := make([]byte, flatObjectSize+4)
		binary.LittleEndian.PutUint32(payload[:4], flatTypeBinder)
		p := NewParcelBytes(payload)

		handle, err := p.ReadStrongBinderHandle()
		if err != nil {
			t.Fatalf("ReadStrongBinderHandle(null): %v", err)
		}
		if handle != nil {
			t.Fatalf("ReadStrongBinderHandle(null) = %v, want nil", *handle)
		}
	})
}

func TestParcelWriteInterfaceToken(t *testing.T) {
	p := NewParcel()

	if err := p.WriteInterfaceToken("android.os.IServiceManager"); err != nil {
		t.Fatalf("WriteInterfaceToken: %v", err)
	}

	got := p.Bytes()
	if len(got) < 16 {
		t.Fatalf("len(Bytes) = %d, want at least 16", len(got))
	}
	if got[0] != 0x00 || got[1] != 0x00 || got[2] != 0x00 || got[3] != 0x80 {
		t.Fatalf("strict mode header = %v, want [0 0 0 128]", got[:4])
	}
	if got[4] != 0xff || got[5] != 0xff || got[6] != 0xff || got[7] != 0xff {
		t.Fatalf("work source header = %v, want [-1]", got[4:8])
	}
	if got[8] != 'T' || got[9] != 'S' || got[10] != 'Y' || got[11] != 'S' {
		t.Fatalf("interface token header = %v, want SYST in little-endian", got[8:12])
	}
}

func TestParcelWriteStrongBinderLocalWireData(t *testing.T) {
	p := NewParcel()

	if err := p.WriteStrongBinderLocal(0x11, 0x22); err != nil {
		t.Fatalf("WriteStrongBinderLocal: %v", err)
	}

	payload, offsets := p.KernelWireData()
	if len(payload) != flatObjectSize+4 {
		t.Fatalf("len(payload) = %d, want %d", len(payload), flatObjectSize+4)
	}
	if len(offsets) != 1 || offsets[0] != 0 {
		t.Fatalf("offsets = %v, want [0]", offsets)
	}
	if got := binary.LittleEndian.Uint32(payload[0:4]); got != flatTypeBinder {
		t.Fatalf("object type = %#x, want %#x", got, flatTypeBinder)
	}
	if got := binary.LittleEndian.Uint64(payload[8:16]); got != 0x11 {
		t.Fatalf("binder ptr = %#x, want %#x", got, uint64(0x11))
	}
	if got := binary.LittleEndian.Uint64(payload[16:24]); got != 0x22 {
		t.Fatalf("cookie = %#x, want %#x", got, uint64(0x22))
	}
	if got := int32(binary.LittleEndian.Uint32(payload[24:28])); got != systemStabilityLevel {
		t.Fatalf("stability = %d, want %d", got, systemStabilityLevel)
	}
}
