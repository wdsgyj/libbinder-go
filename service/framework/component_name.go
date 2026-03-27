package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type ComponentName struct {
	Package string
	Class   string
}

func NewComponentName(pkg string, class string) *ComponentName {
	return &ComponentName{Package: pkg, Class: class}
}

func (c ComponentName) FlattenToShortString() string {
	return c.Package + "/" + c.Class
}

func WriteComponentNameToParcel(p *api.Parcel, v ComponentName) error {
	if v.Package == "" || v.Class == "" {
		return fmt.Errorf("%w: component name requires package and class", api.ErrBadParcelable)
	}
	if err := p.WriteString(v.Package); err != nil {
		return err
	}
	return p.WriteString(v.Class)
}

func ReadComponentNameFromParcel(p *api.Parcel) (ComponentName, error) {
	value, err := ReadNullableComponentNameFromParcel(p)
	if err != nil {
		return ComponentName{}, err
	}
	if value == nil {
		return ComponentName{}, api.ErrBadParcelable
	}
	return *value, nil
}

func WriteNullableComponentNameToParcel(p *api.Parcel, v *ComponentName) error {
	if v == nil {
		return p.WriteNullableString(nil)
	}
	if err := p.WriteString(v.Package); err != nil {
		return err
	}
	return p.WriteString(v.Class)
}

func ReadNullableComponentNameFromParcel(p *api.Parcel) (*ComponentName, error) {
	pkg, err := p.ReadNullableString()
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, nil
	}
	class, err := p.ReadString()
	if err != nil {
		return nil, err
	}
	return &ComponentName{Package: *pkg, Class: class}, nil
}
