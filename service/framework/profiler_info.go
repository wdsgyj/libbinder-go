package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ProfilerInfo struct {
	ProfileFile           *string
	ProfileFD             *api.ParcelFileDescriptor
	SamplingInterval      int32
	AutoStopProfiler      bool
	StreamingOutput       bool
	Agent                 *string
	AttachAgentDuringBind bool
	ClockType             int32
	ProfilerOutputVersion int32
}

func WriteProfilerInfoToParcel(p *api.Parcel, v ProfilerInfo) error {
	if err := p.WriteNullableString(v.ProfileFile); err != nil {
		return err
	}
	if v.ProfileFD != nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		if err := p.WriteParcelFileDescriptor(*v.ProfileFD); err != nil {
			return err
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	if err := p.WriteInt32(v.SamplingInterval); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.AutoStopProfiler)); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.StreamingOutput)); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.Agent); err != nil {
		return err
	}
	if err := p.WriteBool(v.AttachAgentDuringBind); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ClockType); err != nil {
		return err
	}
	return p.WriteInt32(v.ProfilerOutputVersion)
}

func ReadProfilerInfoFromParcel(p *api.Parcel) (ProfilerInfo, error) {
	profileFile, err := p.ReadNullableString()
	if err != nil {
		return ProfilerInfo{}, err
	}
	var profileFD *api.ParcelFileDescriptor
	hasFD, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	if hasFD != 0 {
		fd, err := p.ReadParcelFileDescriptor()
		if err != nil {
			return ProfilerInfo{}, err
		}
		profileFD = &fd
	}
	samplingInterval, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	autoStopProfiler, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	streamingOutput, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	agent, err := p.ReadNullableString()
	if err != nil {
		return ProfilerInfo{}, err
	}
	attachAgentDuringBind, err := p.ReadBool()
	if err != nil {
		return ProfilerInfo{}, err
	}
	clockType, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	profilerOutputVersion, err := p.ReadInt32()
	if err != nil {
		return ProfilerInfo{}, err
	}
	return ProfilerInfo{
		ProfileFile:           profileFile,
		ProfileFD:             profileFD,
		SamplingInterval:      samplingInterval,
		AutoStopProfiler:      autoStopProfiler != 0,
		StreamingOutput:       streamingOutput != 0,
		Agent:                 agent,
		AttachAgentDuringBind: attachAgentDuringBind,
		ClockType:             clockType,
		ProfilerOutputVersion: profilerOutputVersion,
	}, nil
}
