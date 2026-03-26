//go:build !((linux || android) && (amd64 || arm64))

package kernel

import (
	"context"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func (b *Backend) transactHandleParcel(ctx context.Context, state *ThreadState, handle uint32, code uint32, payload []byte, offsets []uint64, flags api.Flags) ([]byte, []api.ParcelObject, error) {
	replyBytes, replyObjects, _, err := b.Driver.TransactHandleParcel(handle, code, payload, offsets, flags)
	return replyBytes, replyObjects, err
}
