package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerRecentTaskInfo = OpaqueParcelable

func WriteActivityManagerRecentTaskInfoToParcel(p *api.Parcel, v ActivityManagerRecentTaskInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadActivityManagerRecentTaskInfoFromParcel(p *api.Parcel) (ActivityManagerRecentTaskInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
