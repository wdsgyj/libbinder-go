package kernel

import (
	"context"
	"fmt"
	"os"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type LocalNode struct {
	ID      uintptr
	Handler api.Handler
	Serial  bool
	mu      sync.Mutex
}

type transactionMetadata struct {
	CallingPID int32
	CallingUID uint32
	Code       uint32
	Flags      uint32
	Local      bool
}

type localRegistry struct {
	mu    sync.RWMutex
	next  uint64
	nodes map[uintptr]*LocalNode
}

func newLocalRegistry() *localRegistry {
	return &localRegistry{
		next:  1,
		nodes: make(map[uintptr]*LocalNode),
	}
}

func (b *Backend) RegisterLocalNode(handler api.Handler, serial bool) (*LocalNode, error) {
	if handler == nil {
		return nil, fmt.Errorf("%w: nil handler", api.ErrUnsupported)
	}

	b.locals.mu.Lock()
	defer b.locals.mu.Unlock()

	id := uintptr(b.locals.next)
	b.locals.next++

	node := &LocalNode{
		ID:      id,
		Handler: handler,
		Serial:  serial,
	}
	b.locals.nodes[id] = node
	return node, nil
}

func (b *Backend) localNodeByID(id uintptr) (*LocalNode, bool) {
	if b == nil || b.locals == nil || id == 0 {
		return nil, false
	}

	b.locals.mu.RLock()
	defer b.locals.mu.RUnlock()

	node, ok := b.locals.nodes[id]
	return node, ok
}

func (b *Backend) DispatchLocalTransaction(ctx context.Context, nodeID uintptr, code uint32, data *api.Parcel, flags uint32) (*api.Parcel, error) {
	return b.dispatchLocalTransaction(ctx, nodeID, code, data, flags, transactionMetadata{
		CallingPID: int32(os.Getpid()),
		CallingUID: uint32(os.Geteuid()),
		Code:       code,
		Flags:      flags,
		Local:      true,
	})
}

func (b *Backend) dispatchLocalTransaction(ctx context.Context, nodeID uintptr, code uint32, data *api.Parcel, flags uint32, meta transactionMetadata) (*api.Parcel, error) {
	node, ok := b.localNodeByID(nodeID)
	if !ok {
		return nil, fmt.Errorf("%w: unknown local binder node %d", api.ErrUnsupported, nodeID)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = api.WithTransactionContext(ctx, api.TransactionContext{
		Code:       meta.Code,
		Flags:      publicFlags(meta.Flags),
		CallingPID: meta.CallingPID,
		CallingUID: meta.CallingUID,
		Local:      meta.Local,
	})
	if node.Serial {
		node.mu.Lock()
		defer node.mu.Unlock()
	}

	switch code {
	case InterfaceTransaction:
		reply := api.NewParcel()
		if err := reply.WriteString(node.Handler.Descriptor()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case PingTransaction:
		return api.NewParcel(), nil
	case GetInterfaceVersionTransaction:
		provider, ok := node.Handler.(api.InterfaceVersionProvider)
		if !ok {
			return nil, api.ErrUnknownTransaction
		}
		reply := api.NewParcel()
		if err := reply.WriteInt32(provider.InterfaceVersion()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case GetInterfaceHashTransaction:
		provider, ok := node.Handler.(api.InterfaceHashProvider)
		if !ok {
			return nil, api.ErrUnknownTransaction
		}
		reply := api.NewParcel()
		if err := reply.WriteString(provider.InterfaceHash()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}

	reply, err := node.Handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if flags&0x01 != 0 {
		return nil, nil
	}
	if reply == nil {
		return api.NewParcel(), nil
	}
	if err := reply.SetPosition(0); err != nil {
		return nil, err
	}
	return reply, nil
}

func publicFlags(flags uint32) api.Flags {
	var out api.Flags
	if flags&uint32(api.FlagOneway) != 0 {
		out |= api.FlagOneway
	}
	return out
}
