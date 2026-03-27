package framework

import "fmt"

import api "github.com/wdsgyj/libbinder-go/binder"

const (
	javaBundleMagic   = 0x4C444E42
	nativeBundleMagic = 0x4C444E44
)

type Bundle struct {
	RawData []byte
	Native  bool
}

func NewEmptyBundle() *Bundle {
	return &Bundle{}
}

func NewRawBundle(raw []byte, native bool) *Bundle {
	out := &Bundle{Native: native}
	if raw != nil {
		out.RawData = append([]byte(nil), raw...)
	}
	return out
}

func WriteBundleToParcel(p *api.Parcel, v Bundle) error {
	if len(v.RawData) == 0 {
		return p.WriteInt32(0)
	}
	if len(v.RawData) > int(^uint32(0)>>1)-4 {
		return fmt.Errorf("%w: bundle payload too large", api.ErrBadParcelable)
	}
	if err := p.WriteInt32(int32(len(v.RawData) + 4)); err != nil {
		return err
	}
	magic := int32(javaBundleMagic)
	if v.Native {
		magic = nativeBundleMagic
	}
	if err := p.WriteInt32(magic); err != nil {
		return err
	}
	return p.WriteRawBytes(v.RawData)
}

func ReadBundleFromParcel(p *api.Parcel) (Bundle, error) {
	value, err := ReadBundleValueFromParcel(p)
	if err != nil {
		return Bundle{}, err
	}
	if value == nil {
		return Bundle{}, api.ErrBadParcelable
	}
	return *value, nil
}

// WriteBundleValueToParcel matches Parcel.writeBundle semantics and permits nil.
func WriteBundleValueToParcel(p *api.Parcel, v *Bundle) error {
	if v == nil {
		return p.WriteInt32(-1)
	}
	return WriteBundleToParcel(p, *v)
}

// ReadBundleValueFromParcel matches Parcel.readBundle semantics and permits nil.
func ReadBundleValueFromParcel(p *api.Parcel) (*Bundle, error) {
	length, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, nil
	}
	if length == 0 {
		return &Bundle{}, nil
	}
	if length < 4 || length%4 != 0 {
		return nil, fmt.Errorf("%w: invalid bundle length %d", api.ErrBadParcelable, length)
	}
	magic, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	native := false
	switch uint32(magic) {
	case javaBundleMagic:
	case nativeBundleMagic:
		native = true
	default:
		return nil, fmt.Errorf("%w: invalid bundle magic %#x", api.ErrBadParcelable, uint32(magic))
	}
	raw, err := p.ReadRawBytes(int(length) - 4)
	if err != nil {
		return nil, err
	}
	return &Bundle{
		RawData: raw,
		Native:  native,
	}, nil
}
