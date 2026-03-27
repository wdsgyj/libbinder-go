package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const maxOpaqueParcelableSize = int(^uint32(0) >> 1)

// OpaqueParcelable carries raw parcel payload bytes for framework parcelables
// that are declared as non-structured AIDL types.
type OpaqueParcelable struct {
	RawData []byte
}

func NewOpaqueParcelable(raw []byte) OpaqueParcelable {
	out := OpaqueParcelable{}
	if raw != nil {
		out.RawData = append([]byte(nil), raw...)
	}
	return out
}

func writeOpaqueFrameworkParcelableToParcel(p *api.Parcel, v OpaqueParcelable) error {
	if len(v.RawData) > maxOpaqueParcelableSize {
		return fmt.Errorf("%w: opaque parcelable payload too large: %d", api.ErrBadParcelable, len(v.RawData))
	}
	if err := p.WriteInt32(int32(len(v.RawData))); err != nil {
		return err
	}
	return p.WriteRawBytes(v.RawData)
}

func readOpaqueFrameworkParcelableFromParcel(p *api.Parcel) (OpaqueParcelable, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return OpaqueParcelable{}, err
	}
	if size < 0 {
		return OpaqueParcelable{}, fmt.Errorf("%w: negative opaque parcelable size %d", api.ErrBadParcelable, size)
	}
	raw, err := p.ReadRawBytes(int(size))
	if err != nil {
		return OpaqueParcelable{}, err
	}
	return OpaqueParcelable{RawData: raw}, nil
}
