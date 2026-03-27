package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerMemoryInfo struct {
	AdvertisedMem            int64
	AvailMem                 int64
	TotalMem                 int64
	Threshold                int64
	LowMemory                bool
	HiddenAppThreshold       int64
	SecondaryServerThreshold int64
	VisibleAppThreshold      int64
	ForegroundAppThreshold   int64
}

func WriteActivityManagerMemoryInfoToParcel(p *api.Parcel, v ActivityManagerMemoryInfo) error {
	if err := p.WriteInt64(v.AdvertisedMem); err != nil {
		return err
	}
	if err := p.WriteInt64(v.AvailMem); err != nil {
		return err
	}
	if err := p.WriteInt64(v.TotalMem); err != nil {
		return err
	}
	if err := p.WriteInt64(v.Threshold); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.LowMemory)); err != nil {
		return err
	}
	if err := p.WriteInt64(v.HiddenAppThreshold); err != nil {
		return err
	}
	if err := p.WriteInt64(v.SecondaryServerThreshold); err != nil {
		return err
	}
	if err := p.WriteInt64(v.VisibleAppThreshold); err != nil {
		return err
	}
	return p.WriteInt64(v.ForegroundAppThreshold)
}

func ReadActivityManagerMemoryInfoFromParcel(p *api.Parcel) (ActivityManagerMemoryInfo, error) {
	advertisedMem, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	availMem, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	totalMem, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	threshold, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	lowMemory, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	hiddenAppThreshold, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	secondaryServerThreshold, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	visibleAppThreshold, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	foregroundAppThreshold, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerMemoryInfo{}, err
	}
	return ActivityManagerMemoryInfo{
		AdvertisedMem:            advertisedMem,
		AvailMem:                 availMem,
		TotalMem:                 totalMem,
		Threshold:                threshold,
		LowMemory:                lowMemory != 0,
		HiddenAppThreshold:       hiddenAppThreshold,
		SecondaryServerThreshold: secondaryServerThreshold,
		VisibleAppThreshold:      visibleAppThreshold,
		ForegroundAppThreshold:   foregroundAppThreshold,
	}, nil
}
