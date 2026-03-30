package customcodec

import api "github.com/wdsgyj/libbinder-go/binder"

type CustomBox struct {
	ID    int32
	Label *string
	Tags  []string
	Meta  map[string]string
}

func WriteCustomBoxToParcel(p *api.Parcel, v CustomBox) error {
	if err := p.WriteInt32(v.ID); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.Label); err != nil {
		return err
	}
	if err := api.WriteSlice(p, v.Tags, func(p *api.Parcel, item string) error {
		return p.WriteString(item)
	}); err != nil {
		return err
	}
	return api.WriteMap(p, v.Meta, func(p *api.Parcel, item string) error {
		return p.WriteString(item)
	}, func(p *api.Parcel, item string) error {
		return p.WriteString(item)
	})
}

func ReadCustomBoxFromParcel(p *api.Parcel) (CustomBox, error) {
	id, err := p.ReadInt32()
	if err != nil {
		return CustomBox{}, err
	}
	label, err := p.ReadNullableString()
	if err != nil {
		return CustomBox{}, err
	}
	tags, err := api.ReadSlice(p, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	})
	if err != nil {
		return CustomBox{}, err
	}
	meta, err := api.ReadMap(p, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	}, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	})
	if err != nil {
		return CustomBox{}, err
	}
	return CustomBox{
		ID:    id,
		Label: label,
		Tags:  tags,
		Meta:  meta,
	}, nil
}
