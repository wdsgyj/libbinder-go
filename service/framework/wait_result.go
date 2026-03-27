package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type WaitResult struct {
	Result      int32
	Timeout     bool
	Who         *ComponentName
	TotalTime   int64
	LaunchState int32
}

func WriteWaitResultToParcel(p *api.Parcel, v WaitResult) error {
	if err := p.WriteInt32(v.Result); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.Timeout)); err != nil {
		return err
	}
	if err := WriteNullableComponentNameToParcel(p, v.Who); err != nil {
		return err
	}
	if err := p.WriteInt64(v.TotalTime); err != nil {
		return err
	}
	return p.WriteInt32(v.LaunchState)
}

func ReadWaitResultFromParcel(p *api.Parcel) (WaitResult, error) {
	result, err := p.ReadInt32()
	if err != nil {
		return WaitResult{}, err
	}
	timeout, err := p.ReadInt32()
	if err != nil {
		return WaitResult{}, err
	}
	who, err := ReadNullableComponentNameFromParcel(p)
	if err != nil {
		return WaitResult{}, err
	}
	totalTime, err := p.ReadInt64()
	if err != nil {
		return WaitResult{}, err
	}
	launchState, err := p.ReadInt32()
	if err != nil {
		return WaitResult{}, err
	}
	return WaitResult{
		Result:      result,
		Timeout:     timeout != 0,
		Who:         who,
		TotalTime:   totalTime,
		LaunchState: launchState,
	}, nil
}
