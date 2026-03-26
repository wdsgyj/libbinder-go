package binder

import "context"

// ServiceManager provides discovery and registration of Binder services.
type ServiceManager interface {
	CheckService(ctx context.Context, name string) (Binder, error)
	WaitService(ctx context.Context, name string) (Binder, error)
	AddService(ctx context.Context, name string, handler Handler, opts ...AddServiceOption) error
	ListServices(ctx context.Context, dumpFlags DumpFlags) ([]string, error)
	WatchServiceRegistrations(ctx context.Context, name string, callback ServiceRegistrationCallback) (Subscription, error)
	IsDeclared(ctx context.Context, name string) (bool, error)
	DeclaredInstances(ctx context.Context, iface string) ([]string, error)
	UpdatableViaApex(ctx context.Context, name string) (*string, error)
	UpdatableNames(ctx context.Context, apexName string) ([]string, error)
	ConnectionInfo(ctx context.Context, name string) (*ConnectionInfo, error)
	WatchClients(ctx context.Context, name string, service Binder, callback ServiceClientCallback) (Subscription, error)
	TryUnregisterService(ctx context.Context, name string, service Binder) error
	DebugInfo(ctx context.Context) ([]ServiceDebugInfo, error)
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

const (
	DumpPriorityAll     DumpFlags = DumpPriorityCritical | DumpPriorityHigh | DumpPriorityNormal | DumpPriorityDefault
	DumpFlagLazyService DumpFlags = 1 << 30
)

// ConnectionInfo describes service-manager advertised endpoint metadata for an instance.
type ConnectionInfo struct {
	IPAddress string
	Port      uint32
}

// ServiceDebugInfo captures debug metadata for a registered service.
type ServiceDebugInfo struct {
	Name     string
	DebugPID int32
}

// ServiceMetadata contains governance metadata tracked alongside a service registration.
type ServiceMetadata struct {
	Declared         bool
	UpdatableViaApex string
	ConnectionInfo   *ConnectionInfo
	DebugPID         int32
}

// ServiceRegistration describes a service registration callback event.
type ServiceRegistration struct {
	Name   string
	Binder Binder
}

// ServiceClientUpdate describes a client-count transition callback event.
type ServiceClientUpdate struct {
	Name       string
	Service    Binder
	HasClients bool
}

// ServiceRegistrationCallback receives registration events for a named service.
type ServiceRegistrationCallback func(context.Context, ServiceRegistration)

// ServiceClientCallback receives client-count transition events for a named service.
type ServiceClientCallback func(context.Context, ServiceClientUpdate)

// AddServiceOptions controls optional service registration behavior.
type AddServiceOptions struct {
	AllowIsolated bool
	DumpFlags     DumpFlags
	Serial        bool
	Lazy          bool
	Metadata      ServiceMetadata
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

func WithLazyService(v bool) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Lazy = v
	})
}

// WithSerialHandler requests that the runtime execute the handler in a serial mode.
func WithSerialHandler(v bool) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Serial = v
	})
}

func WithServiceMetadata(metadata ServiceMetadata) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Metadata = metadata
	})
}

func WithDeclaredService(v bool) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Metadata.Declared = v
	})
}

func WithUpdatableViaApex(name string) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Metadata.UpdatableViaApex = name
	})
}

func WithConnectionInfo(info ConnectionInfo) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		copied := info
		opts.Metadata.ConnectionInfo = &copied
	})
}

func WithDebugPID(pid int32) AddServiceOption {
	return addServiceOptionFunc(func(opts *AddServiceOptions) {
		opts.Metadata.DebugPID = pid
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

func (opts AddServiceOptions) EffectiveDumpFlags() DumpFlags {
	flags := opts.DumpFlags
	if opts.Lazy {
		flags |= DumpFlagLazyService
	}
	return flags
}
