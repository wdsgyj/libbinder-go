//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"

	api "libbinder-go/binder"
)

const (
	statusUnknownError      int32 = -2147483648
	statusFailedTransaction int32 = statusUnknownError + 2
)

// LooperWorker represents a thread-bound Binder looper worker.
type LooperWorker struct {
	Name string

	State   *ThreadState
	Backend *Backend
	Driver  *DriverManager

	stop chan struct{}
	done chan struct{}

	closeOnce sync.Once
}

func NewLooperWorker(name string, backend *Backend) *LooperWorker {
	return &LooperWorker{
		Name:    name,
		State:   &ThreadState{Role: "looper"},
		Backend: backend,
		Driver:  backend.Driver,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (w *LooperWorker) Start() error {
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(w.done)

		if err := w.Driver.EnterLooper(); err != nil {
			w.State.LastErr = err
			ready <- err
			return
		}

		w.State.Bound = true
		ready <- nil

		readBuf := make([]byte, 4096)
		for {
			select {
			case <-w.stop:
				return
			default:
			}

			bwr, err := w.Driver.Read(readBuf)
			if err != nil {
				if w.shouldExitOnReadError(err) {
					return
				}
				w.State.LastErr = err
				return
			}
			if bwr.ReadConsumed == 0 {
				continue
			}

			if err := w.handleCommands(context.Background(), readBuf[:bwr.ReadConsumed]); err != nil {
				if w.isStopping() {
					return
				}
				w.State.LastErr = err
				return
			}
		}
	}()

	return <-ready
}

func (w *LooperWorker) Close() error {
	w.closeOnce.Do(func() {
		close(w.stop)
	})
	<-w.done
	return nil
}

func (w *LooperWorker) handleCommands(ctx context.Context, buf []byte) error {
	for pos := 0; pos < len(buf); {
		if len(buf[pos:]) < 4 {
			return fmt.Errorf("kernel: short looper command buffer tail: %d", len(buf[pos:]))
		}

		cmd := binary.LittleEndian.Uint32(buf[pos:])
		pos += 4

		switch cmd {
		case BRNoop, BRTransactionComplete, BRSpawnLooper, BRFinished:
		case BRIncRefs, BRAcquire, BRRelease, BRDecRefs:
			if len(buf[pos:]) < 16 {
				return fmt.Errorf("kernel: short ptr/cookie payload for %#x", cmd)
			}
			ptr := uintptr(binary.LittleEndian.Uint64(buf[pos:]))
			cookie := uintptr(binary.LittleEndian.Uint64(buf[pos+8:]))
			pos += 16

			switch cmd {
			case BRIncRefs:
				if err := w.Driver.WritePtrCookieCommand(BCIncRefsDone, ptr, cookie); err != nil {
					return err
				}
			case BRAcquire:
				if err := w.Driver.WritePtrCookieCommand(BCAcquireDone, ptr, cookie); err != nil {
					return err
				}
			}
		case BRDeadBinder, BRClearDeathNotificationDone:
			if len(buf[pos:]) < 8 {
				return fmt.Errorf("kernel: short cookie payload for %#x", cmd)
			}
			cookie := uintptr(binary.LittleEndian.Uint64(buf[pos:]))
			pos += 8

			switch cmd {
			case BRDeadBinder:
				if w.Driver.deaths != nil {
					w.Driver.deaths.NotifyDead(cookie)
				}
				if err := w.Driver.DeadBinderDone(cookie); err != nil {
					return err
				}
			case BRClearDeathNotificationDone:
				if w.Driver.deaths != nil {
					w.Driver.deaths.NotifyCleared(cookie)
				}
			}
		case BRTransaction:
			if len(buf[pos:]) < binderTransactionDataSize {
				return fmt.Errorf("kernel: short BR_TRANSACTION payload: have %d want %d", len(buf[pos:]), binderTransactionDataSize)
			}
			var tx BinderTransactionData
			if err := tx.UnmarshalBinary(buf[pos : pos+binderTransactionDataSize]); err != nil {
				return err
			}
			pos += binderTransactionDataSize

			if err := w.handleTransaction(ctx, &tx); err != nil {
				return err
			}
		default:
			return fmt.Errorf("kernel: unsupported looper command %#x", cmd)
		}
	}

	return nil
}

func (w *LooperWorker) handleTransaction(ctx context.Context, tx *BinderTransactionData) error {
	if tx == nil {
		return fmt.Errorf("kernel: nil transaction")
	}

	if tx.DataBuffer != 0 {
		defer func() {
			_ = w.Driver.FreeBuffer(tx.BufferPointer())
		}()
	}

	requestBytes, requestObjects, err := copyTransactionPayload(tx)
	if err != nil {
		if tx.Flags&TFOneWay == 0 {
			return w.sendStatusReply(statusFailedTransaction, tx.Flags&TFClearBuf)
		}
		return err
	}
	if err := w.Driver.acquireParcelObjects(requestObjects); err != nil {
		if tx.Flags&TFOneWay == 0 {
			return w.sendStatusReply(statusFailedTransaction, tx.Flags&TFClearBuf)
		}
		return err
	}

	request := api.NewParcelWire(requestBytes, requestObjects)
	reply, err := w.Backend.DispatchLocalTransaction(ctx, tx.CookiePointer(), tx.Code, request, tx.Flags)
	if tx.Flags&TFOneWay != 0 {
		return nil
	}
	if err != nil {
		return w.sendStatusReply(statusFailedTransaction, tx.Flags&TFClearBuf)
	}
	return w.sendReply(reply, tx.Flags&TFClearBuf)
}

func (w *LooperWorker) sendReply(reply *api.Parcel, flags uint32) error {
	var tx BinderTransactionData
	tx.SetTargetHandle(^uint32(0))
	tx.Flags = flags

	var payload []byte
	var offsets []uint64
	if reply != nil {
		payload, offsets = reply.KernelWireData()
	}
	setTransactionPayload(&tx, payload, offsets)

	bwr, err := w.Driver.WriteCommand(BCReply, tx.MarshalBinary(), nil)
	runtime.KeepAlive(payload)
	runtime.KeepAlive(offsets)
	if err != nil {
		return err
	}
	if want := uint64(4 + binderTransactionDataSize); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: BC_REPLY consumed %d bytes, want %d", bwr.WriteConsumed, want)
	}
	return nil
}

func (w *LooperWorker) sendStatusReply(code int32, flags uint32) error {
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, uint32(code))

	var tx BinderTransactionData
	tx.SetTargetHandle(^uint32(0))
	tx.Flags = TFStatus | (flags & TFClearBuf)
	setTransactionPayload(&tx, payload, nil)

	bwr, err := w.Driver.WriteCommand(BCReply, tx.MarshalBinary(), nil)
	runtime.KeepAlive(payload)
	if err != nil {
		return err
	}
	if want := uint64(4 + binderTransactionDataSize); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: BC_REPLY(status) consumed %d bytes, want %d", bwr.WriteConsumed, want)
	}
	return nil
}

func (w *LooperWorker) isStopping() bool {
	select {
	case <-w.stop:
		return true
	default:
		return false
	}
}

func (w *LooperWorker) shouldExitOnReadError(err error) bool {
	if w.isStopping() {
		return true
	}
	return errors.Is(err, ErrDriverClosed) || errors.Is(err, syscall.EBADF)
}
