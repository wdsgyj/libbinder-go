package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ConfigurationInfo struct {
	ReqTouchScreen   int32
	ReqKeyboardType  int32
	ReqNavigation    int32
	ReqInputFeatures int32
	ReqGlEsVersion   int32
}

func WriteConfigurationInfoToParcel(p *api.Parcel, v ConfigurationInfo) error {
	if err := p.WriteInt32(v.ReqTouchScreen); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ReqKeyboardType); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ReqNavigation); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ReqInputFeatures); err != nil {
		return err
	}
	return p.WriteInt32(v.ReqGlEsVersion)
}

func ReadConfigurationInfoFromParcel(p *api.Parcel) (ConfigurationInfo, error) {
	reqTouchScreen, err := p.ReadInt32()
	if err != nil {
		return ConfigurationInfo{}, err
	}
	reqKeyboardType, err := p.ReadInt32()
	if err != nil {
		return ConfigurationInfo{}, err
	}
	reqNavigation, err := p.ReadInt32()
	if err != nil {
		return ConfigurationInfo{}, err
	}
	reqInputFeatures, err := p.ReadInt32()
	if err != nil {
		return ConfigurationInfo{}, err
	}
	reqGlEsVersion, err := p.ReadInt32()
	if err != nil {
		return ConfigurationInfo{}, err
	}
	return ConfigurationInfo{
		ReqTouchScreen:   reqTouchScreen,
		ReqKeyboardType:  reqKeyboardType,
		ReqNavigation:    reqNavigation,
		ReqInputFeatures: reqInputFeatures,
		ReqGlEsVersion:   reqGlEsVersion,
	}, nil
}

func WriteNullableConfigurationInfoToParcel(p *api.Parcel, v *ConfigurationInfo) error {
	if v == nil {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return WriteConfigurationInfoToParcel(p, *v)
}

func ReadNullableConfigurationInfoFromParcel(p *api.Parcel) (*ConfigurationInfo, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if present == 0 {
		return nil, nil
	}
	v, err := ReadConfigurationInfoFromParcel(p)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
