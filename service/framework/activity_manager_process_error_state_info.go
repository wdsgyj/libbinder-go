package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerProcessErrorStateInfo struct {
	Condition   int32
	ProcessName *string
	PID         int32
	UID         int32
	Tag         *string
	ShortMsg    *string
	LongMsg     *string
	StackTrace  *string
}

func WriteActivityManagerProcessErrorStateInfoToParcel(p *api.Parcel, v ActivityManagerProcessErrorStateInfo) error {
	if err := p.WriteInt32(v.Condition); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ProcessName); err != nil {
		return err
	}
	if err := p.WriteInt32(v.PID); err != nil {
		return err
	}
	if err := p.WriteInt32(v.UID); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.Tag); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ShortMsg); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.LongMsg); err != nil {
		return err
	}
	return p.WriteNullableString(v.StackTrace)
}

func ReadActivityManagerProcessErrorStateInfoFromParcel(p *api.Parcel) (ActivityManagerProcessErrorStateInfo, error) {
	condition, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	processName, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	pid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	uid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	tag, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	shortMsg, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	longMsg, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	stackTrace, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerProcessErrorStateInfo{}, err
	}
	return ActivityManagerProcessErrorStateInfo{
		Condition:   condition,
		ProcessName: processName,
		PID:         pid,
		UID:         uid,
		Tag:         tag,
		ShortMsg:    shortMsg,
		LongMsg:     longMsg,
		StackTrace:  stackTrace,
	}, nil
}
