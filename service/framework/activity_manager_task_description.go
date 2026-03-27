package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ActivityManagerTaskDescription = OpaqueParcelable

func WriteActivityManagerTaskDescriptionToParcel(p *api.Parcel, v ActivityManagerTaskDescription) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadActivityManagerTaskDescriptionFromParcel(p *api.Parcel) (ActivityManagerTaskDescription, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
