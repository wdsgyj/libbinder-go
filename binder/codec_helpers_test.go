package binder

import (
	"errors"
	"testing"
)

func TestWriteReadSliceInt32(t *testing.T) {
	p := NewParcel()

	if err := WriteSlice(p, []int32{1, 2, 3}, func(p *Parcel, v int32) error {
		return p.WriteInt32(v)
	}); err != nil {
		t.Fatalf("WriteSlice: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := ReadSlice(p, func(p *Parcel) (int32, error) {
		return p.ReadInt32()
	})
	if err != nil {
		t.Fatalf("ReadSlice: %v", err)
	}
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("ReadSlice = %v, want [1 2 3]", got)
	}
}

func TestWriteReadSliceNil(t *testing.T) {
	p := NewParcel()

	if err := WriteSlice[string](p, nil, func(p *Parcel, v string) error {
		return p.WriteString(v)
	}); err != nil {
		t.Fatalf("WriteSlice(nil): %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := ReadSlice(p, func(p *Parcel) (string, error) {
		return p.ReadString()
	})
	if err != nil {
		t.Fatalf("ReadSlice(nil): %v", err)
	}
	if got != nil {
		t.Fatalf("ReadSlice(nil) = %v, want nil", got)
	}
}

func TestWriteReadFixedSlice(t *testing.T) {
	p := NewParcel()

	if err := WriteFixedSlice(p, []uint32{7, 8}, 2, func(p *Parcel, v uint32) error {
		return p.WriteUint32(v)
	}); err != nil {
		t.Fatalf("WriteFixedSlice: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := ReadFixedSlice(p, 2, func(p *Parcel) (uint32, error) {
		return p.ReadUint32()
	})
	if err != nil {
		t.Fatalf("ReadFixedSlice: %v", err)
	}
	if len(got) != 2 || got[0] != 7 || got[1] != 8 {
		t.Fatalf("ReadFixedSlice = %v, want [7 8]", got)
	}
}

func TestReadFixedSliceWrongLength(t *testing.T) {
	p := NewParcel()

	if err := WriteSlice(p, []uint32{1}, func(p *Parcel, v uint32) error {
		return p.WriteUint32(v)
	}); err != nil {
		t.Fatalf("WriteSlice: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	_, err := ReadFixedSlice(p, 2, func(p *Parcel) (uint32, error) {
		return p.ReadUint32()
	})
	if !errors.Is(err, ErrBadParcelable) {
		t.Fatalf("ReadFixedSlice error = %v, want ErrBadParcelable", err)
	}
}
