package binder

import "context"

// Flags controls Binder transaction behavior.
type Flags uint32

const (
	FlagNone Flags = 0

	// FlagOneway marks an asynchronous transaction that does not expect a reply.
	FlagOneway Flags = 1 << 0
)

// Binder is the public abstraction for a local or remote Binder object.
type Binder interface {
	Descriptor(ctx context.Context) (string, error)
	Transact(ctx context.Context, code uint32, data *Parcel, flags Flags) (*Parcel, error)
	WatchDeath(ctx context.Context) (Subscription, error)
	Close() error
}

// BinderProvider exposes the underlying low-level Binder object for generated
// typed interfaces and adapters.
type BinderProvider interface {
	AsBinder() Binder
}

// LocalHandlerRegistrar can materialize a process-local Binder node from a
// Handler so callback interfaces can be marshaled over Binder.
type LocalHandlerRegistrar interface {
	RegisterLocalHandler(handler Handler) (Binder, error)
}

// ParcelBinderMarshaler writes a Binder object into a Parcel using the
// implementation's native transport representation.
type ParcelBinderMarshaler interface {
	WriteBinderToParcel(p *Parcel) error
}

// Handler serves transactions for a local Binder object.
type Handler interface {
	Descriptor() string
	HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error)
}

// HandlerFunc adapts a function to Handler when paired with a descriptor.
type HandlerFunc func(ctx context.Context, code uint32, data *Parcel) (*Parcel, error)

// StaticHandler is a small helper for wiring a descriptor to a HandlerFunc.
type StaticHandler struct {
	DescriptorName string
	Handle         HandlerFunc
}

func (h StaticHandler) Descriptor() string {
	return h.DescriptorName
}

func (h StaticHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.Handle(ctx, code, data)
}
