package runtime

import "libbinder-go/internal/kernel"

// Config holds top-level runtime construction settings.
type Config struct {
	DriverPath string

	Kernel kernel.StartOptions
}

// Runtime is the internal root object that wires together runtime state and
// transport backends.
type Runtime struct {
	Kernel   *kernel.Backend
	Registry *Registry
	Router   *TransactionRouter
	Refs     *RefTracker
	Subs     *SubscriptionSet
}

func New(cfg Config) *Runtime {
	opts := cfg.Kernel
	if opts == (kernel.StartOptions{}) {
		opts = kernel.DefaultStartOptions()
	}

	return &Runtime{
		Kernel:   kernel.NewBackend(cfg.DriverPath),
		Registry: NewRegistry(),
		Router:   NewTransactionRouter(),
		Refs:     NewRefTracker(),
		Subs:     NewSubscriptionSet(),
	}
}

func (r *Runtime) Start(cfg Config) error {
	opts := cfg.Kernel
	if opts == (kernel.StartOptions{}) {
		opts = kernel.DefaultStartOptions()
	}
	return r.Kernel.Start(opts)
}

func (r *Runtime) Close() error {
	return r.Kernel.Close()
}
