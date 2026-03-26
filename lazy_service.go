package libbinder

import (
	"context"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func AddLazyService(ctx context.Context, sm api.ServiceManager, name string, descriptor string, factory api.LazyHandlerFactory, opts ...api.AddServiceOption) error {
	if sm == nil {
		return api.ErrUnsupported
	}
	return sm.AddService(ctx, name, api.NewLazyHandler(descriptor, factory), opts...)
}

func AddLazyServiceWithMetadata(ctx context.Context, sm api.ServiceManager, name string, cfg api.LazyHandlerConfig, factory api.LazyHandlerFactory, opts ...api.AddServiceOption) error {
	if sm == nil {
		return api.ErrUnsupported
	}
	return sm.AddService(ctx, name, api.NewLazyHandlerWithMetadata(cfg, factory), opts...)
}
