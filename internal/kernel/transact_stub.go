//go:build !((linux || android) && (amd64 || arm64))

package kernel

import api "libbinder-go/binder"

func (d *DriverManager) TransactHandleParcel(handle uint32, code uint32, payload []byte, flags api.Flags) ([]byte, []api.ParcelObject, []uint32, error) {
	return nil, nil, nil, ErrUnsupportedPlatform
}
