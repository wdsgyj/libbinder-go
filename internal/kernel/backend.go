package kernel

import (
	"context"

	api "libbinder-go/binder"
)

const DefaultDriverPath = "/dev/binder"

// StartOptions controls backend startup behavior.
type StartOptions struct {
	LooperWorkers int
	ClientWorkers int
}

func DefaultStartOptions() StartOptions {
	return StartOptions{
		LooperWorkers: 1,
		ClientWorkers: 1,
	}
}

// Backend groups the process-scoped kernel Binder runtime components.
type Backend struct {
	Driver  *DriverManager
	Process *ProcessState
	Workers *WorkerManager

	locals *localRegistry
	deaths *deathRegistry
}

func NewBackend(driverPath string) *Backend {
	if driverPath == "" {
		driverPath = DefaultDriverPath
	}

	driver := NewDriverManager(driverPath)
	backend := &Backend{
		Driver:  driver,
		Process: NewProcessState(driverPath),
		locals:  newLocalRegistry(),
	}
	backend.deaths = newDeathRegistry(backend.requestDeathNotification)
	driver.deaths = backend.deaths
	backend.Workers = NewWorkerManager(backend)
	return backend
}

func (b *Backend) Start(opts StartOptions) error {
	if err := b.Driver.Open(); err != nil {
		return err
	}

	if err := b.Workers.Start(opts); err != nil {
		_ = b.Driver.Close()
		return err
	}

	b.Process.MarkStarted(opts)
	return nil
}

func (b *Backend) Close() error {
	if b.deaths != nil {
		b.deaths.Close()
	}
	driverErr := b.Driver.Close()
	workerErr := b.Workers.Close()
	b.Process.MarkStopped()

	if workerErr != nil {
		return workerErr
	}
	return driverErr
}

func (b *Backend) TransactHandle(ctx context.Context, handle uint32, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := b.Workers.Client()
	if err != nil {
		return nil, err
	}

	var request []byte
	var requestOffsets []uint64
	if data != nil {
		request, requestOffsets = data.KernelWireData()
	}

	var reply *api.Parcel
	err = client.Do(ctx, func(state *ThreadState) error {
		replyBytes, replyObjects, _, callErr := b.Driver.TransactHandleParcel(handle, code, request, requestOffsets, flags)
		state.OutBuffer = append(state.OutBuffer[:0], request...)
		if callErr != nil {
			state.InBuffer = state.InBuffer[:0]
			return callErr
		}

		state.InBuffer = append(state.InBuffer[:0], replyBytes...)
		if flags&api.FlagOneway != 0 {
			reply = nil
			return nil
		}

		reply = api.NewParcelWire(replyBytes, replyObjects)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (b *Backend) AcquireHandle(ctx context.Context, handle uint32) error {
	if handle == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := b.Workers.Client()
	if err != nil {
		return err
	}

	return client.Do(ctx, func(_ *ThreadState) error {
		return b.Driver.AcquireHandle(handle)
	})
}

func (b *Backend) WatchDeath(ctx context.Context, handle uint32) (api.Subscription, error) {
	if b == nil || b.deaths == nil {
		return nil, ErrUnsupportedPlatform
	}
	return b.deaths.Watch(ctx, handle)
}

func (b *Backend) requestDeathNotification(ctx context.Context, handle uint32, cookie uintptr) error {
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := b.Workers.Client()
	if err != nil {
		return err
	}

	return client.Do(ctx, func(_ *ThreadState) error {
		return b.Driver.RequestDeathNotification(handle, cookie)
	})
}
