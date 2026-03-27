package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type IntentSender struct {
	Target api.Binder
}

func NewIntentSender(target api.Binder) *IntentSender {
	if target == nil {
		return nil
	}
	return &IntentSender{Target: target}
}

func (s *IntentSender) AsBinder() api.Binder {
	if s == nil {
		return nil
	}
	return s.Target
}

func WriteIntentSenderToParcel(p *api.Parcel, v IntentSender) error {
	if v.Target == nil {
		return fmt.Errorf("%w: intent sender target cannot be nil", api.ErrBadParcelable)
	}
	return p.WriteStrongBinder(v.Target)
}

func ReadIntentSenderFromParcel(p *api.Parcel) (IntentSender, error) {
	target, err := p.ReadStrongBinder()
	if err != nil {
		return IntentSender{}, err
	}
	if target == nil {
		return IntentSender{}, api.ErrBadParcelable
	}
	return IntentSender{Target: target}, nil
}

func WriteNullableIntentSenderToParcel(p *api.Parcel, v *IntentSender) error {
	if v == nil || v.Target == nil {
		return p.WriteStrongBinder(nil)
	}
	return p.WriteStrongBinder(v.Target)
}

func ReadNullableIntentSenderFromParcel(p *api.Parcel) (*IntentSender, error) {
	target, err := p.ReadStrongBinder()
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}
	return &IntentSender{Target: target}, nil
}
