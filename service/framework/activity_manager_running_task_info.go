package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerRunningTaskInfo = OpaqueParcelable

func WriteActivityManagerRunningTaskInfoToParcel(p *api.Parcel, v ActivityManagerRunningTaskInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadActivityManagerRunningTaskInfoFromParcel(p *api.Parcel) (ActivityManagerRunningTaskInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
