package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type PersistableBundle struct {
	RawData []byte
	Native  bool
}

func NewEmptyPersistableBundle() *PersistableBundle {
	return &PersistableBundle{}
}

func NewRawPersistableBundle(raw []byte, native bool) *PersistableBundle {
	out := &PersistableBundle{Native: native}
	if raw != nil {
		out.RawData = append([]byte(nil), raw...)
	}
	return out
}

func WritePersistableBundleToParcel(p *api.Parcel, v PersistableBundle) error {
	return WriteBundleToParcel(p, Bundle(v))
}

func ReadPersistableBundleFromParcel(p *api.Parcel) (PersistableBundle, error) {
	v, err := ReadBundleFromParcel(p)
	if err != nil {
		return PersistableBundle{}, err
	}
	return PersistableBundle(v), nil
}

func WritePersistableBundleValueToParcel(p *api.Parcel, v *PersistableBundle) error {
	if v == nil {
		return p.WriteInt32(-1)
	}
	return WritePersistableBundleToParcel(p, *v)
}

func ReadPersistableBundleValueFromParcel(p *api.Parcel) (*PersistableBundle, error) {
	v, err := ReadBundleValueFromParcel(p)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	out := PersistableBundle(*v)
	return &out, nil
}
