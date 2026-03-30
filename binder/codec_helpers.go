package binder

import "fmt"

// WriteSlice writes an AIDL-style variable-length array or List payload.
func WriteSlice[T any](p *Parcel, values []T, writeElem func(*Parcel, T) error) error {
	if p == nil {
		return ErrBadParcelable
	}
	if writeElem == nil {
		return fmt.Errorf("%w: nil element writer", ErrBadParcelable)
	}
	if values == nil {
		return p.WriteInt32(-1)
	}
	if len(values) > maxInt32 {
		return fmt.Errorf("%w: slice too large: %d", ErrBadParcelable, len(values))
	}
	if err := p.WriteInt32(int32(len(values))); err != nil {
		return err
	}
	for _, v := range values {
		if err := writeElem(p, v); err != nil {
			return err
		}
	}
	return nil
}

// ReadSlice reads an AIDL-style variable-length array or List payload.
func ReadSlice[T any](p *Parcel, readElem func(*Parcel) (T, error)) ([]T, error) {
	var zero T
	if p == nil {
		return nil, ErrBadParcelable
	}
	if readElem == nil {
		return nil, fmt.Errorf("%w: nil element reader", ErrBadParcelable)
	}

	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if size < 0 {
		return nil, nil
	}

	values := make([]T, 0, int(size))
	for i := 0; i < int(size); i++ {
		v, err := readElem(p)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	if values == nil {
		return []T{zero}[0:0], nil
	}
	return values, nil
}

// WriteFixedSlice writes a fixed-size AIDL array payload.
func WriteFixedSlice[T any](p *Parcel, values []T, size int, writeElem func(*Parcel, T) error) error {
	if values == nil {
		return fmt.Errorf("%w: fixed slice is null", ErrBadParcelable)
	}
	if len(values) != size {
		return fmt.Errorf("%w: fixed slice length %d, want %d", ErrBadParcelable, len(values), size)
	}
	return WriteSlice(p, values, writeElem)
}

// ReadFixedSlice reads a fixed-size AIDL array payload.
func ReadFixedSlice[T any](p *Parcel, size int, readElem func(*Parcel) (T, error)) ([]T, error) {
	values, err := ReadSlice(p, readElem)
	if err != nil {
		return nil, err
	}
	if values == nil {
		return nil, fmt.Errorf("%w: fixed slice is null", ErrBadParcelable)
	}
	if len(values) != size {
		return nil, fmt.Errorf("%w: fixed slice length %d, want %d", ErrBadParcelable, len(values), size)
	}
	return values, nil
}
