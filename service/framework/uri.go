package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type URIKind int32

const (
	URINull         URIKind = 0
	URIString       URIKind = 1
	URIOpaque       URIKind = 2
	URIHierarchical URIKind = 3
)

type URI struct {
	Kind  URIKind
	Value string
}

func ParseURI(value string) *URI {
	return &URI{Kind: URIString, Value: value}
}

func WriteURIToParcel(p *api.Parcel, v URI) error {
	kind := v.Kind
	if kind == URINull {
		kind = URIString
	}
	if kind != URIString && kind != URIOpaque && kind != URIHierarchical {
		return fmt.Errorf("%w: unsupported uri kind %d", api.ErrBadParcelable, kind)
	}
	if err := p.WriteInt32(int32(kind)); err != nil {
		return err
	}
	return p.WriteString8(v.Value)
}

func ReadURIFromParcel(p *api.Parcel) (URI, error) {
	v, err := ReadNullableURIFromParcel(p)
	if err != nil {
		return URI{}, err
	}
	if v == nil {
		return URI{}, api.ErrBadParcelable
	}
	return *v, nil
}

func WriteNullableURIToParcel(p *api.Parcel, v *URI) error {
	if v == nil {
		return p.WriteInt32(int32(URINull))
	}
	return WriteURIToParcel(p, *v)
}

func ReadNullableURIFromParcel(p *api.Parcel) (*URI, error) {
	kind, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	switch URIKind(kind) {
	case URINull:
		return nil, nil
	case URIString, URIOpaque, URIHierarchical:
		value, err := p.ReadString8()
		if err != nil {
			return nil, err
		}
		return &URI{Kind: URIKind(kind), Value: value}, nil
	default:
		return nil, fmt.Errorf("%w: unknown uri kind %d", api.ErrBadParcelable, kind)
	}
}
