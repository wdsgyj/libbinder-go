package kernel

// ThreadState stands in for the Go equivalent of the per-thread Binder runtime state.
type ThreadState struct {
	Role  string
	Bound bool

	InBuffer  []byte
	OutBuffer []byte

	LastErr error
}
