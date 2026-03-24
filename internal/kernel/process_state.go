package kernel

// ProcessState holds process-scoped kernel Binder runtime state.
type ProcessState struct {
	DriverPath string
	Started    bool

	LooperWorkers int
	ClientWorkers int
}

func NewProcessState(driverPath string) *ProcessState {
	return &ProcessState{DriverPath: driverPath}
}

func (p *ProcessState) MarkStarted(opts StartOptions) {
	p.Started = true
	p.LooperWorkers = opts.LooperWorkers
	p.ClientWorkers = opts.ClientWorkers
}

func (p *ProcessState) MarkStopped() {
	p.Started = false
	p.LooperWorkers = 0
	p.ClientWorkers = 0
}
