package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type LocusId struct {
	ID string
}

func NewLocusId(id string) *LocusId {
	if id == "" {
		return nil
	}
	return &LocusId{ID: id}
}

func WriteLocusIdToParcel(p *api.Parcel, v LocusId) error {
	if v.ID == "" {
		return fmt.Errorf("%w: locus id cannot be empty", api.ErrBadParcelable)
	}
	return p.WriteString(v.ID)
}

func ReadLocusIdFromParcel(p *api.Parcel) (LocusId, error) {
	id, err := p.ReadString()
	if err != nil {
		return LocusId{}, err
	}
	if id == "" {
		return LocusId{}, fmt.Errorf("%w: locus id cannot be empty", api.ErrBadParcelable)
	}
	return LocusId{ID: id}, nil
}

func WriteNullableLocusIdToParcel(p *api.Parcel, v *LocusId) error {
	if v == nil {
		return p.WriteNullableString(nil)
	}
	return WriteLocusIdToParcel(p, *v)
}

func ReadNullableLocusIdFromParcel(p *api.Parcel) (*LocusId, error) {
	id, err := p.ReadNullableString()
	if err != nil {
		return nil, err
	}
	if id == nil {
		return nil, nil
	}
	if *id == "" {
		return nil, fmt.Errorf("%w: locus id cannot be empty", api.ErrBadParcelable)
	}
	return &LocusId{ID: *id}, nil
}
