package binder

import "context"

// ServiceManager provides discovery and registration of Binder services.
type ServiceManager interface {
	CheckService(ctx context.Context, name string) (Binder, error)
	WaitService(ctx context.Context, name string) (Binder, error)
	AddService(ctx context.Context, name string, handler Handler, opts ...AddServiceOption) error
}

// DumpFlags controls service dump registration metadata.
type DumpFlags uint32

const (
	DumpPriorityCritical DumpFlags = 1 << iota
	DumpPriorityHigh
	DumpPriorityNormal
	DumpPriorityDefault
	DumpProto
)

// AddServiceOptions controls optional service registration behavior.
type AddServiceOptions struct {
	AllowIsolated bool
	DumpFlags     DumpFlags
	Serial        bool
}

type AddServiceOption interface {
	applyAddService(*AddServiceOptions)
}

type addServiceOptionFunc func(*AddServiceOptions)

func (f addServiceOptionFunc) applyAddService(opts *AddServiceOptions) {
	f(opts)
}

func WithAllowIsolated(v bool) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.AllowIsolated = v
	})
}

func WithDumpFlags(flags DumpFlags) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.DumpFlags = flags
	})
}

// WithSerialHandler requests that the runtime execute the handler in a serial mode.
func WithSerialHandler(v bool) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Serial = v
	})
}

func DefaultAddServiceOptions() AddServiceOptions {
	return AddServiceOptions{
		DumpFlags: DumpPriorityDefault,
	}
}

func ResolveAddServiceOptions(opts ...AddServiceOption) AddServiceOptions {
	resolved := DefaultAddServiceOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.applyAddService(&resolved)
	}
	return resolved
}
