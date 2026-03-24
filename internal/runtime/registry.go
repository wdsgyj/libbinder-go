package runtime

// Registry tracks local and remote Binder objects inside the runtime core.
type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}
