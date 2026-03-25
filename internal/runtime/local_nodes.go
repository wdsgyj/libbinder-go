package runtime

import api "libbinder-go/binder"

type LocalNodeRef struct {
	ID uintptr
}

func (r *Runtime) RegisterLocalNode(handler api.Handler, serial bool) (LocalNodeRef, error) {
	node, err := r.Kernel.RegisterLocalNode(handler, serial)
	if err != nil {
		return LocalNodeRef{}, err
	}
	return LocalNodeRef{ID: node.ID}, nil
}
