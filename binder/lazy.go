package binder

import (
	"context"
	"sync"
)

type LazyHandlerFactory func() (Handler, error)

type LazyHandlerConfig struct {
	Descriptor string
	Version    *int32
	Hash       *string
}

func NewLazyHandler(descriptor string, factory LazyHandlerFactory) Handler {
	return &lazyHandler{
		core: &lazyHandlerCore{
			descriptor: descriptor,
			factory:    factory,
		},
	}
}

func NewLazyHandlerWithMetadata(cfg LazyHandlerConfig, factory LazyHandlerFactory) Handler {
	core := &lazyHandlerCore{
		descriptor: cfg.Descriptor,
		factory:    factory,
	}

	switch {
	case cfg.Version != nil && cfg.Hash != nil:
		return &lazyVersionHashHandler{core: core, version: *cfg.Version, hash: *cfg.Hash}
	case cfg.Version != nil:
		return &lazyVersionHandler{core: core, version: *cfg.Version}
	case cfg.Hash != nil:
		return &lazyHashHandler{core: core, hash: *cfg.Hash}
	default:
		return &lazyHandler{core: core}
	}
}

type lazyHandlerCore struct {
	descriptor string
	factory    LazyHandlerFactory

	once    sync.Once
	handler Handler
	err     error
}

func (h *lazyHandlerCore) Descriptor() string {
	return h.descriptor
}

func (h *lazyHandlerCore) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	handler, err := h.instance()
	if err != nil {
		return nil, err
	}
	return handler.HandleTransact(ctx, code, data)
}

func (h *lazyHandlerCore) instance() (Handler, error) {
	if h == nil {
		return nil, ErrUnsupported
	}

	h.once.Do(func() {
		if h.factory == nil {
			h.err = ErrUnsupported
			return
		}
		h.handler, h.err = h.factory()
		if h.handler == nil && h.err == nil {
			h.err = ErrUnsupported
		}
	})
	return h.handler, h.err
}

type lazyHandler struct {
	core *lazyHandlerCore
}

func (h *lazyHandler) Descriptor() string {
	return h.core.Descriptor()
}

func (h *lazyHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.core.HandleTransact(ctx, code, data)
}

type lazyVersionHandler struct {
	core    *lazyHandlerCore
	version int32
}

func (h *lazyVersionHandler) Descriptor() string {
	return h.core.Descriptor()
}

func (h *lazyVersionHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.core.HandleTransact(ctx, code, data)
}

func (h *lazyVersionHandler) InterfaceVersion() int32 {
	return h.version
}

type lazyHashHandler struct {
	core *lazyHandlerCore
	hash string
}

func (h *lazyHashHandler) Descriptor() string {
	return h.core.Descriptor()
}

func (h *lazyHashHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.core.HandleTransact(ctx, code, data)
}

func (h *lazyHashHandler) InterfaceHash() string {
	return h.hash
}

type lazyVersionHashHandler struct {
	core    *lazyHandlerCore
	version int32
	hash    string
}

func (h *lazyVersionHashHandler) Descriptor() string {
	return h.core.Descriptor()
}

func (h *lazyVersionHashHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.core.HandleTransact(ctx, code, data)
}

func (h *lazyVersionHashHandler) InterfaceVersion() int32 {
	return h.version
}

func (h *lazyVersionHashHandler) InterfaceHash() string {
	return h.hash
}
