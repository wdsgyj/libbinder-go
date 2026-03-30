package binder

import (
	"fmt"
	"reflect"
)

// ValueTag matches android.os.Parcel writeValue/readValue discriminators.
type ValueTag int32

const (
	ValueNull        ValueTag = -1
	ValueString      ValueTag = 0
	ValueInteger     ValueTag = 1
	ValueMap         ValueTag = 2
	ValueParcelable  ValueTag = 4
	ValueShort       ValueTag = 5
	ValueLong        ValueTag = 6
	ValueFloat       ValueTag = 7
	ValueDouble      ValueTag = 8
	ValueBoolean     ValueTag = 9
	ValueList        ValueTag = 11
	ValueByteArray   ValueTag = 13
	ValueStringArray ValueTag = 14
	ValueIBinder     ValueTag = 15
	ValueIntArray    ValueTag = 18
	ValueLongArray   ValueTag = 19
	ValueByte        ValueTag = 20
	ValueDoubleArray ValueTag = 28
	ValueChar        ValueTag = 29
	ValueShortArray  ValueTag = 30
	ValueCharArray   ValueTag = 31
	ValueFloatArray  ValueTag = 32
)

// WriteMap writes a java.util.Map-compatible payload body.
func WriteMap[K comparable, V any](p *Parcel, values map[K]V, writeKey func(*Parcel, K) error, writeValue func(*Parcel, V) error) error {
	if p == nil {
		return ErrBadParcelable
	}
	if writeKey == nil || writeValue == nil {
		return fmt.Errorf("%w: nil map codec", ErrBadParcelable)
	}
	if values == nil {
		return p.WriteInt32(-1)
	}
	if len(values) > maxInt32 {
		return fmt.Errorf("%w: map too large: %d", ErrBadParcelable, len(values))
	}
	if err := p.WriteInt32(int32(len(values))); err != nil {
		return err
	}
	for key, value := range values {
		if err := writeKey(p, key); err != nil {
			return err
		}
		if err := writeValue(p, value); err != nil {
			return err
		}
	}
	return nil
}

// ReadMap reads a java.util.Map-compatible payload body.
func ReadMap[K comparable, V any](p *Parcel, readKey func(*Parcel) (K, error), readValue func(*Parcel) (V, error)) (map[K]V, error) {
	if p == nil {
		return nil, ErrBadParcelable
	}
	if readKey == nil || readValue == nil {
		return nil, fmt.Errorf("%w: nil map codec", ErrBadParcelable)
	}
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if size < 0 {
		return nil, nil
	}
	values := make(map[K]V, int(size))
	for i := 0; i < int(size); i++ {
		key, err := readKey(p)
		if err != nil {
			return nil, err
		}
		value, err := readValue(p)
		if err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, nil
}

// WriteDynamicValue writes the android.os.Parcel writeValue wire subset used by raw Map support.
func WriteDynamicValue(p *Parcel, value any) error {
	if p == nil {
		return ErrBadParcelable
	}
	if value == nil {
		return p.WriteInt32(int32(ValueNull))
	}

	switch v := value.(type) {
	case string:
		return writeDynamicTagged(p, ValueString, func() error { return p.WriteString(v) })
	case *string:
		if v == nil {
			return p.WriteInt32(int32(ValueNull))
		}
		return writeDynamicTagged(p, ValueString, func() error { return p.WriteString(*v) })
	case int:
		if v >= -1<<31 && v <= 1<<31-1 {
			return writeDynamicTagged(p, ValueInteger, func() error { return p.WriteInt32(int32(v)) })
		}
		return writeDynamicTagged(p, ValueLong, func() error { return p.WriteInt64(int64(v)) })
	case int32:
		return writeDynamicTagged(p, ValueInteger, func() error { return p.WriteInt32(v) })
	case int16:
		return writeDynamicTagged(p, ValueShort, func() error { return p.WriteInt32(int32(v)) })
	case int8:
		return writeDynamicTagged(p, ValueByte, func() error { return p.WriteInt32(int32(v)) })
	case int64:
		return writeDynamicTagged(p, ValueLong, func() error { return p.WriteInt64(v) })
	case uint16:
		return writeDynamicTagged(p, ValueChar, func() error { return p.WriteInt32(int32(v)) })
	case bool:
		return writeDynamicTagged(p, ValueBoolean, func() error { return p.WriteBool(v) })
	case float32:
		return writeDynamicTagged(p, ValueFloat, func() error { return p.WriteFloat32(v) })
	case float64:
		return writeDynamicTagged(p, ValueDouble, func() error { return p.WriteFloat64(v) })
	case []byte:
		return writeDynamicTagged(p, ValueByteArray, func() error { return p.WriteBytes(v) })
	case []string:
		return writeDynamicTagged(p, ValueStringArray, func() error { return writeStringArray(p, v) })
	case []int32:
		return writeDynamicTagged(p, ValueIntArray, func() error { return writeInt32Array(p, v) })
	case []int64:
		return writeDynamicTagged(p, ValueLongArray, func() error { return writeInt64Array(p, v) })
	case []int16:
		return writeDynamicTagged(p, ValueShortArray, func() error { return writeInt16Array(p, v) })
	case []uint16:
		return writeDynamicTagged(p, ValueCharArray, func() error { return writeUint16Array(p, v) })
	case []float32:
		return writeDynamicTagged(p, ValueFloatArray, func() error { return writeFloat32Array(p, v) })
	case []float64:
		return writeDynamicTagged(p, ValueDoubleArray, func() error { return writeFloat64Array(p, v) })
	case Binder:
		return writeDynamicTagged(p, ValueIBinder, func() error { return p.WriteStrongBinder(v) })
	case map[any]any:
		return writeDynamicTagged(p, ValueMap, func() error {
			return WriteMap(p, v, func(p *Parcel, item any) error { return WriteDynamicValue(p, item) }, func(p *Parcel, item any) error { return WriteDynamicValue(p, item) })
		})
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return p.WriteInt32(int32(ValueNull))
		}
		return writeDynamicTagged(p, ValueMap, func() error {
			size := rv.Len()
			if size > maxInt32 {
				return fmt.Errorf("%w: map too large: %d", ErrBadParcelable, size)
			}
			if err := p.WriteInt32(int32(size)); err != nil {
				return err
			}
			iter := rv.MapRange()
			for iter.Next() {
				if err := WriteDynamicValue(p, iter.Key().Interface()); err != nil {
					return err
				}
				if err := WriteDynamicValue(p, iter.Value().Interface()); err != nil {
					return err
				}
			}
			return nil
		})
	case reflect.Slice:
		if rv.IsNil() {
			return p.WriteInt32(int32(ValueNull))
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return writeDynamicTagged(p, ValueByteArray, func() error { return p.WriteBytes(rv.Bytes()) })
		}
		fallthrough
	case reflect.Array:
		return writeDynamicTagged(p, ValueList, func() error {
			size := rv.Len()
			if size > maxInt32 {
				return fmt.Errorf("%w: list too large: %d", ErrBadParcelable, size)
			}
			if err := p.WriteInt32(int32(size)); err != nil {
				return err
			}
			for i := 0; i < size; i++ {
				if err := WriteDynamicValue(p, rv.Index(i).Interface()); err != nil {
					return err
				}
			}
			return nil
		})
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			return p.WriteInt32(int32(ValueNull))
		}
		return WriteDynamicValue(p, rv.Elem().Interface())
	default:
		return fmt.Errorf("%w: unsupported dynamic value type %T", ErrUnsupported, value)
	}
}

// ReadDynamicValue reads the android.os.Parcel readValue wire subset used by raw Map support.
func ReadDynamicValue(p *Parcel) (any, error) {
	if p == nil {
		return nil, ErrBadParcelable
	}
	tag, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	switch ValueTag(tag) {
	case ValueNull:
		return nil, nil
	case ValueString:
		return p.ReadString()
	case ValueInteger:
		return p.ReadInt32()
	case ValueMap:
		return ReadMap(p, func(p *Parcel) (any, error) { return ReadDynamicValue(p) }, func(p *Parcel) (any, error) { return ReadDynamicValue(p) })
	case ValueParcelable:
		name, nameErr := p.ReadString()
		if nameErr != nil {
			return nil, nameErr
		}
		return nil, fmt.Errorf("%w: dynamic parcelable %q requires a typed codec", ErrUnsupported, name)
	case ValueShort:
		v, err := p.ReadInt32()
		return int16(v), err
	case ValueLong:
		return p.ReadInt64()
	case ValueFloat:
		return p.ReadFloat32()
	case ValueDouble:
		return p.ReadFloat64()
	case ValueBoolean:
		return p.ReadBool()
	case ValueList:
		return ReadSlice(p, func(p *Parcel) (any, error) { return ReadDynamicValue(p) })
	case ValueByteArray:
		return p.ReadBytes()
	case ValueStringArray:
		return readStringArray(p)
	case ValueIBinder:
		return p.ReadStrongBinder()
	case ValueIntArray:
		return readInt32Array(p)
	case ValueLongArray:
		return readInt64Array(p)
	case ValueByte:
		v, err := p.ReadInt32()
		return int8(v), err
	case ValueDoubleArray:
		return readFloat64Array(p)
	case ValueChar:
		v, err := p.ReadInt32()
		return uint16(v), err
	case ValueShortArray:
		return readInt16Array(p)
	case ValueCharArray:
		return readUint16Array(p)
	case ValueFloatArray:
		return readFloat32Array(p)
	default:
		return nil, fmt.Errorf("%w: unsupported dynamic value tag %d", ErrUnsupported, tag)
	}
}

func writeDynamicTagged(p *Parcel, tag ValueTag, body func() error) error {
	if err := p.WriteInt32(int32(tag)); err != nil {
		return err
	}
	return body()
}

func writeStringArray(p *Parcel, values []string) error {
	return WriteSlice(p, values, func(p *Parcel, value string) error { return p.WriteString(value) })
}

func readStringArray(p *Parcel) ([]string, error) {
	return ReadSlice(p, func(p *Parcel) (string, error) { return p.ReadString() })
}

func writeInt32Array(p *Parcel, values []int32) error {
	return WriteSlice(p, values, func(p *Parcel, value int32) error { return p.WriteInt32(value) })
}

func readInt32Array(p *Parcel) ([]int32, error) {
	return ReadSlice(p, func(p *Parcel) (int32, error) { return p.ReadInt32() })
}

func writeInt64Array(p *Parcel, values []int64) error {
	return WriteSlice(p, values, func(p *Parcel, value int64) error { return p.WriteInt64(value) })
}

func readInt64Array(p *Parcel) ([]int64, error) {
	return ReadSlice(p, func(p *Parcel) (int64, error) { return p.ReadInt64() })
}

func writeInt16Array(p *Parcel, values []int16) error {
	return WriteSlice(p, values, func(p *Parcel, value int16) error { return p.WriteInt32(int32(value)) })
}

func readInt16Array(p *Parcel) ([]int16, error) {
	return ReadSlice(p, func(p *Parcel) (int16, error) {
		value, err := p.ReadInt32()
		return int16(value), err
	})
}

func writeUint16Array(p *Parcel, values []uint16) error {
	return WriteSlice(p, values, func(p *Parcel, value uint16) error { return p.WriteInt32(int32(value)) })
}

func readUint16Array(p *Parcel) ([]uint16, error) {
	return ReadSlice(p, func(p *Parcel) (uint16, error) {
		value, err := p.ReadInt32()
		return uint16(value), err
	})
}

func writeFloat32Array(p *Parcel, values []float32) error {
	return WriteSlice(p, values, func(p *Parcel, value float32) error { return p.WriteFloat32(value) })
}

func readFloat32Array(p *Parcel) ([]float32, error) {
	return ReadSlice(p, func(p *Parcel) (float32, error) { return p.ReadFloat32() })
}

func writeFloat64Array(p *Parcel, values []float64) error {
	return WriteSlice(p, values, func(p *Parcel, value float64) error { return p.WriteFloat64(value) })
}

func readFloat64Array(p *Parcel) ([]float64, error) {
	return ReadSlice(p, func(p *Parcel) (float64, error) { return p.ReadFloat64() })
}
