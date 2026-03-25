package runtime

import "sync"

// RefTracker owns process-scoped reference bookkeeping for remote Binder handles.
type RefTracker struct {
	mu      sync.Mutex
	handles map[uint32]*HandleRef
}

type HandleRef struct {
	BinderRefs int
	WatchRefs  int

	Acquired            bool
	Acquiring           bool
	ReleaseAfterAcquire bool
	Wait                chan struct{}
}

func NewRefTracker() *RefTracker {
	return &RefTracker{
		handles: make(map[uint32]*HandleRef),
	}
}

func (r *RefTracker) RetainBinder(handle uint32) {
	if handle == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ref := r.ensure(handle)
	ref.BinderRefs++
	ref.ReleaseAfterAcquire = false
}

func (r *RefTracker) RetainWatch(handle uint32) {
	if handle == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ref := r.ensure(handle)
	ref.WatchRefs++
	ref.ReleaseAfterAcquire = false
}

func (r *RefTracker) BeginAcquire(handle uint32) (bool, <-chan struct{}) {
	if handle == 0 {
		return false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ref := r.ensure(handle)
	if ref.Acquired {
		return false, nil
	}
	if ref.Acquiring {
		return false, ref.Wait
	}

	ref.Acquiring = true
	ref.Wait = make(chan struct{})
	return true, nil
}

func (r *RefTracker) FinishAcquire(handle uint32, ok bool) bool {
	if handle == 0 {
		return false
	}

	var wait chan struct{}
	shouldRelease := false

	r.mu.Lock()
	ref := r.handles[handle]
	if ref != nil {
		wait = ref.Wait
		ref.Wait = nil
		ref.Acquiring = false
		if ok {
			ref.Acquired = true
		}

		if ref.ReleaseAfterAcquire && ref.BinderRefs+ref.WatchRefs == 0 {
			shouldRelease = ok
			delete(r.handles, handle)
		} else if !ref.Acquired && ref.BinderRefs+ref.WatchRefs == 0 {
			delete(r.handles, handle)
		}
	}
	r.mu.Unlock()

	if wait != nil {
		close(wait)
	}
	return shouldRelease
}

func (r *RefTracker) MarkAcquired(handle uint32) {
	if handle == 0 {
		return
	}

	var wait chan struct{}

	r.mu.Lock()
	ref := r.ensure(handle)
	ref.Acquired = true
	if ref.Acquiring {
		ref.Acquiring = false
		wait = ref.Wait
		ref.Wait = nil
	}
	r.mu.Unlock()

	if wait != nil {
		close(wait)
	}
}

func (r *RefTracker) ReleaseBinder(handle uint32) bool {
	return r.release(handle, true)
}

func (r *RefTracker) ReleaseWatch(handle uint32) bool {
	return r.release(handle, false)
}

func (r *RefTracker) release(handle uint32, binder bool) bool {
	if handle == 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ref := r.handles[handle]
	if ref == nil {
		return false
	}

	if binder {
		if ref.BinderRefs > 0 {
			ref.BinderRefs--
		}
	} else if ref.WatchRefs > 0 {
		ref.WatchRefs--
	}

	if ref.BinderRefs+ref.WatchRefs > 0 {
		return false
	}

	if ref.Acquiring {
		ref.ReleaseAfterAcquire = true
		return false
	}

	delete(r.handles, handle)
	return ref.Acquired
}

func (r *RefTracker) ensure(handle uint32) *HandleRef {
	ref := r.handles[handle]
	if ref == nil {
		ref = &HandleRef{}
		r.handles[handle] = ref
	}
	return ref
}
