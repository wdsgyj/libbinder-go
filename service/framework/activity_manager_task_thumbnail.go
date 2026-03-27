package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerTaskThumbnail = OpaqueParcelable

func WriteActivityManagerTaskThumbnailToParcel(p *api.Parcel, v ActivityManagerTaskThumbnail) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadActivityManagerTaskThumbnailFromParcel(p *api.Parcel) (ActivityManagerTaskThumbnail, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
