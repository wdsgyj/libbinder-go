package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerPendingIntentInfo struct {
	CreatorPackage   *string
	CreatorUID       int32
	Immutable        bool
	IntentSenderType int32
}

func WriteActivityManagerPendingIntentInfoToParcel(p *api.Parcel, v ActivityManagerPendingIntentInfo) error {
	if err := p.WriteNullableString(v.CreatorPackage); err != nil {
		return err
	}
	if err := p.WriteInt32(v.CreatorUID); err != nil {
		return err
	}
	if err := p.WriteBool(v.Immutable); err != nil {
		return err
	}
	return p.WriteInt32(v.IntentSenderType)
}

func ReadActivityManagerPendingIntentInfoFromParcel(p *api.Parcel) (ActivityManagerPendingIntentInfo, error) {
	creatorPackage, err := p.ReadNullableString()
	if err != nil {
		return ActivityManagerPendingIntentInfo{}, err
	}
	creatorUID, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerPendingIntentInfo{}, err
	}
	immutable, err := p.ReadBool()
	if err != nil {
		return ActivityManagerPendingIntentInfo{}, err
	}
	intentSenderType, err := p.ReadInt32()
	if err != nil {
		return ActivityManagerPendingIntentInfo{}, err
	}
	return ActivityManagerPendingIntentInfo{
		CreatorPackage:   creatorPackage,
		CreatorUID:       creatorUID,
		Immutable:        immutable,
		IntentSenderType: intentSenderType,
	}, nil
}
