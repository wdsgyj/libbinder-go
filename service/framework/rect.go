package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func WriteRectToParcel(p *api.Parcel, v Rect) error {
	if err := p.WriteInt32(v.Left); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Top); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Right); err != nil {
		return err
	}
	return p.WriteInt32(v.Bottom)
}

func ReadRectFromParcel(p *api.Parcel) (Rect, error) {
	left, err := p.ReadInt32()
	if err != nil {
		return Rect{}, err
	}
	top, err := p.ReadInt32()
	if err != nil {
		return Rect{}, err
	}
	right, err := p.ReadInt32()
	if err != nil {
		return Rect{}, err
	}
	bottom, err := p.ReadInt32()
	if err != nil {
		return Rect{}, err
	}
	return Rect{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
	}, nil
}
