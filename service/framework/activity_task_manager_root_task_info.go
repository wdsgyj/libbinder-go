package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityTaskManagerRootTaskInfo = OpaqueParcelable

func WriteActivityTaskManagerRootTaskInfoToParcel(p *api.Parcel, v ActivityTaskManagerRootTaskInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadActivityTaskManagerRootTaskInfoFromParcel(p *api.Parcel) (ActivityTaskManagerRootTaskInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
