package runtime

import "context"

func (r *Runtime) AcquireHandle(ctx context.Context, handle uint32) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Kernel.AcquireHandle(ctx, handle)
}

func (r *Runtime) ReleaseHandle(ctx context.Context, handle uint32) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Kernel.ReleaseHandle(ctx, handle)
}
