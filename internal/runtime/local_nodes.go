package runtime

import api "github.com/wdsgyj/libbinder-go/binder"

type LocalNodeRef struct {
	ID        uintptr
	Stability api.StabilityLevel
}

func (r *Runtime) RegisterLocalNode(handler api.Handler, serial bool) (LocalNodeRef, error) {
	node, err := r.Kernel.RegisterLocalNode(handler, serial)
	if err != nil {
		return LocalNodeRef{}, err
	}
	return LocalNodeRef{
		ID:        node.ID,
		Stability: node.Stability,
	}, nil
}
