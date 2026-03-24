package runtime

// RefTracker owns protocol-level reference bookkeeping for remote Binder objects.
type RefTracker struct{}

func NewRefTracker() *RefTracker {
	return &RefTracker{}
}
