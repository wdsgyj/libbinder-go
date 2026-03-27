package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerRunningServiceInfo struct {
	Service          *ComponentName
	PID              int32
	UID              int32
	Process          *string
	Foreground       bool
	ActiveSince      int64
	Started          bool
	ClientCount      int32
	CrashCount       int32
	LastActivityTime int64
	Restarting       int64
	Flags            int32
	ClientPackage    *string
	ClientLabel      int32
}

func WriteActivityManagerRunningServiceInfoToParcel(p *api.Parcel, v ActivityManagerRunningServiceInfo) error {
	if err := WriteNullableComponentNameToParcel(p, v.Service); err != nil {
		return err
	}
	if err := p.WriteInt32(v.PID); err != nil {
		return err
	}
	if err := p.WriteInt32(v.UID); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.Process); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.Foreground)); err != nil {
		return err
	}
	if err := p.WriteInt64(v.ActiveSince); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.Started)); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ClientCount); err != nil {
		return err
	}
	if err := p.WriteInt32(v.CrashCount); err != nil {
		return err
	}
	if err := p.WriteInt64(v.LastActivityTime); err != nil {
		return err
	}
	if err := p.WriteInt64(v.Restarting); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Flags); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ClientPackage); err != nil {
		return err
	}
	return p.WriteInt32(v.ClientLabel)
}

func ReadActivityManagerRunningServiceInfoFromParcel(p *api.Parcel) (ActivityManagerRunningServiceInfo, error) {
	service, err := ReadNullableComponentNameFromParcel(p)
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	pid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	uid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	process, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	foreground, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	activeSince, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	started, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	clientCount, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	crashCount, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	lastActivityTime, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	restarting, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	flags, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	clientPackage, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	clientLabel, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningServiceInfo{}, err
	}
	return ActivityManagerRunningServiceInfo{
		Service:          service,
		PID:              pid,
		UID:              uid,
		Process:          process,
		Foreground:       foreground != 0,
		ActiveSince:      activeSince,
		Started:          started != 0,
		ClientCount:      clientCount,
		CrashCount:       crashCount,
		LastActivityTime: lastActivityTime,
		Restarting:       restarting,
		Flags:            flags,
		ClientPackage:    clientPackage,
		ClientLabel:      clientLabel,
	}, nil
}
