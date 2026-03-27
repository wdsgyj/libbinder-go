package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type Point struct {
	X int32
	Y int32
}

func NewPoint(x int32, y int32) *Point {
	return &Point{X: x, Y: y}
}

func WritePointToParcel(p *api.Parcel, v Point) error {
	if err := p.WriteInt32(v.X); err != nil {
		return err
	}
	return p.WriteInt32(v.Y)
}

func ReadPointFromParcel(p *api.Parcel) (Point, error) {
	x, err := p.ReadInt32()
	if err != nil {
		return Point{}, err
	}
	y, err := p.ReadInt32()
	if err != nil {
		return Point{}, err
	}
	return Point{X: x, Y: y}, nil
}

func WriteNullablePointToParcel(p *api.Parcel, v *Point) error {
	if v == nil {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return WritePointToParcel(p, *v)
}

func ReadNullablePointFromParcel(p *api.Parcel) (*Point, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if present == 0 {
		return nil, nil
	}
	v, err := ReadPointFromParcel(p)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
