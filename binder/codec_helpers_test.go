package binder

import (
	"errors"
	"reflect"
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

func TestWriteReadMap(t *testing.T) {
	p := NewParcel()

	if err := WriteMap(p, map[string]int32{"a": 1, "b": 2}, func(p *Parcel, value string) error {
		return p.WriteString(value)
	}, func(p *Parcel, value int32) error {
		return p.WriteInt32(value)
	}); err != nil {
		t.Fatalf("WriteMap: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := ReadMap(p, func(p *Parcel) (string, error) {
		return p.ReadString()
	}, func(p *Parcel) (int32, error) {
		return p.ReadInt32()
	})
	if err != nil {
		t.Fatalf("ReadMap: %v", err)
	}
	if !reflect.DeepEqual(got, map[string]int32{"a": 1, "b": 2}) {
		t.Fatalf("ReadMap = %#v, want %#v", got, map[string]int32{"a": 1, "b": 2})
	}
}

func TestWriteReadDynamicValueNestedMap(t *testing.T) {
	p := NewParcel()
	want := map[any]any{
		"name": "demo",
		"ids":  []any{int32(1), int64(2), "3"},
		"meta": map[any]any{
			"enabled": true,
			"score":   float64(7.5),
		},
	}

	if err := WriteDynamicValue(p, want); err != nil {
		t.Fatalf("WriteDynamicValue: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	got, err := ReadDynamicValue(p)
	if err != nil {
		t.Fatalf("ReadDynamicValue: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDynamicValue = %#v, want %#v", got, want)
	}
}

func TestWriteDynamicValueLengthPrefixedContainers(t *testing.T) {
	p := NewParcel()
	if err := WriteDynamicValue(p, map[any]any{"name": "demo", "tags": []any{"a", int32(2)}}); err != nil {
		t.Fatalf("WriteDynamicValue(map): %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	tag, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32(tag): %v", err)
	}
	if ValueTag(tag) != ValueMap {
		t.Fatalf("tag = %d, want %d", tag, ValueMap)
	}
	length, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32(length): %v", err)
	}
	if length <= 0 {
		t.Fatalf("length = %d, want > 0", length)
	}
}
