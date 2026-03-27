package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func writeStringSliceToParcel(p *api.Parcel, values []string) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value string) error {
		return p.WriteString(value)
	})
}

func readStringSliceFromParcel(p *api.Parcel) ([]string, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	})
}

func writeInt32SliceToParcel(p *api.Parcel, values []int32) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value int32) error {
		return p.WriteInt32(value)
	})
}

func readInt32SliceFromParcel(p *api.Parcel) ([]int32, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (int32, error) {
		return p.ReadInt32()
	})
}

func unsupportedFrameworkParcelable(name string) error {
	return fmt.Errorf("%w: framework parcelable %s is not implemented yet", api.ErrUnsupported, name)
}

func unsupportedFrameworkParcelableRead[T any](name string) (T, error) {
	var zero T
	return zero, unsupportedFrameworkParcelable(name)
}
