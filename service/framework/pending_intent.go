package framework

import (
	"fmt"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type PendingIntent struct {
	Target api.Binder
}

func NewPendingIntent(target api.Binder) *PendingIntent {
	if target == nil {
		return nil
	}
	return &PendingIntent{Target: target}
}

func (p *PendingIntent) AsBinder() api.Binder {
	if p == nil {
		return nil
	}
	return p.Target
}

func WritePendingIntentToParcel(parcel *api.Parcel, v PendingIntent) error {
	if v.Target == nil {
		return fmt.Errorf("%w: pending intent target cannot be nil", api.ErrBadParcelable)
	}
	return parcel.WriteStrongBinder(v.Target)
}

func ReadPendingIntentFromParcel(parcel *api.Parcel) (PendingIntent, error) {
	target, err := parcel.ReadStrongBinder()
	if err != nil {
		return PendingIntent{}, err
	}
	if target == nil {
		return PendingIntent{}, api.ErrBadParcelable
	}
	return PendingIntent{Target: target}, nil
}

func WriteNullablePendingIntentToParcel(parcel *api.Parcel, v *PendingIntent) error {
	if v == nil || v.Target == nil {
		return parcel.WriteStrongBinder(nil)
	}
	return parcel.WriteStrongBinder(v.Target)
}

func ReadNullablePendingIntentFromParcel(parcel *api.Parcel) (*PendingIntent, error) {
	target, err := parcel.ReadStrongBinder()
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, nil
	}
	return &PendingIntent{Target: target}, nil
}
