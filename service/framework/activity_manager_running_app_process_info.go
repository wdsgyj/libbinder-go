package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerRunningAppProcessInfo struct {
	ProcessName                *string
	PID                        int32
	UID                        int32
	PkgList                    []string
	PkgDeps                    []string
	Flags                      int32
	LastTrimLevel              int32
	Importance                 int32
	LRU                        int32
	ImportanceReasonCode       int32
	ImportanceReasonPID        int32
	ImportanceReasonComponent  *ComponentName
	ImportanceReasonImportance int32
	ProcessState               int32
	IsFocused                  bool
	LastActivityTime           int64
}

func WriteActivityManagerRunningAppProcessInfoToParcel(p *api.Parcel, v ActivityManagerRunningAppProcessInfo) error {
	if err := p.WriteNullableString(v.ProcessName); err != nil {
		return err
	}
	if err := p.WriteInt32(v.PID); err != nil {
		return err
	}
	if err := p.WriteInt32(v.UID); err != nil {
		return err
	}
	if err := writeStringSliceToParcel(p, v.PkgList); err != nil {
		return err
	}
	if err := writeStringSliceToParcel(p, v.PkgDeps); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Flags); err != nil {
		return err
	}
	if err := p.WriteInt32(v.LastTrimLevel); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Importance); err != nil {
		return err
	}
	if err := p.WriteInt32(v.LRU); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ImportanceReasonCode); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ImportanceReasonPID); err != nil {
		return err
	}
	if err := WriteNullableComponentNameToParcel(p, v.ImportanceReasonComponent); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ImportanceReasonImportance); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ProcessState); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.IsFocused)); err != nil {
		return err
	}
	return p.WriteInt64(v.LastActivityTime)
}

func ReadActivityManagerRunningAppProcessInfoFromParcel(p *api.Parcel) (ActivityManagerRunningAppProcessInfo, error) {
	processName, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	pid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	uid, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	pkgList, err := readStringSliceFromParcel(p)
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	pkgDeps, err := readStringSliceFromParcel(p)
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	flags, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	lastTrimLevel, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	importance, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	lru, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	importanceReasonCode, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	importanceReasonPID, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	importanceReasonComponent, err := ReadNullableComponentNameFromParcel(p)
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	importanceReasonImportance, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	processState, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	isFocused, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	lastActivityTime, err := p.ReadInt64()
	if err != nil {
		return ActivityManagerRunningAppProcessInfo{}, err
	}
	return ActivityManagerRunningAppProcessInfo{
		ProcessName:                processName,
		PID:                        pid,
		UID:                        uid,
		PkgList:                    pkgList,
		PkgDeps:                    pkgDeps,
		Flags:                      flags,
		LastTrimLevel:              lastTrimLevel,
		Importance:                 importance,
		LRU:                        lru,
		ImportanceReasonCode:       importanceReasonCode,
		ImportanceReasonPID:        importanceReasonPID,
		ImportanceReasonComponent:  importanceReasonComponent,
		ImportanceReasonImportance: importanceReasonImportance,
		ProcessState:               processState,
		IsFocused:                  isFocused != 0,
		LastActivityTime:           lastActivityTime,
	}, nil
}
