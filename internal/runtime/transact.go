package runtime

import (
	"context"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func (r *Runtime) TransactHandle(ctx context.Context, handle uint32, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Kernel.TransactHandle(ctx, handle, code, data, flags)
}
