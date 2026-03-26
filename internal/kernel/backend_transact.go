//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func (b *Backend) transactHandleParcel(ctx context.Context, state *ThreadState, handle uint32, code uint32, payload []byte, offsets []uint64, flags api.Flags) ([]byte, []api.ParcelObject, error) {
	var tx BinderTransactionData
	tx.SetTargetHandle(handle)
	tx.Code = code
	tx.Flags = TFAcceptFD
	if flags&api.FlagOneway != 0 {
		tx.Flags |= TFOneWay
	}
	setTransactionPayload(&tx, payload, offsets)

	readBuf := make([]byte, 256)
	bwr, err := b.Driver.WriteCommand(BCTransaction, tx.MarshalBinary(), readBuf)
	runtime.KeepAlive(payload)
	runtime.KeepAlive(offsets)
	if err != nil {
		return nil, nil, err
	}

	commands, reply, err := b.handleDriverResponses(ctx, state, readBuf[:bwr.ReadConsumed])
	if err != nil {
		return nil, nil, err
	}

	if flags&api.FlagOneway != 0 {
		return nil, nil, nil
	}

	for i := 0; i < 8 && reply == nil; i++ {
		next, err := b.Driver.Read(readBuf)
		if err != nil {
			return nil, nil, err
		}
		moreCommands, moreReply, err := b.handleDriverResponses(ctx, state, readBuf[:next.ReadConsumed])
		if err != nil {
			return nil, nil, err
		}
		commands = append(commands, moreCommands...)
		if moreReply != nil {
			reply = moreReply
		}
	}

	if reply == nil {
		return nil, nil, fmt.Errorf("kernel: did not receive BR_REPLY, commands=%v", commandNames(commands))
	}

	replyBytes, replyObjects, err := copyTransactionPayload(reply)
	if err != nil {
		return nil, nil, err
	}
	if reply.Flags&TFStatus != 0 {
		if reply.DataBuffer != 0 {
			if freeErr := b.Driver.FreeBuffer(reply.BufferPointer()); freeErr != nil {
				return nil, nil, freeErr
			}
		}
		return nil, nil, statusReplyError(replyBytes)
	}
	if err := b.Driver.acquireParcelObjects(replyObjects); err != nil {
		return nil, nil, err
	}
	if reply.DataBuffer != 0 {
		if freeErr := b.Driver.FreeBuffer(reply.BufferPointer()); freeErr != nil {
			return nil, nil, freeErr
		}
	}
	return replyBytes, replyObjects, nil
}

func (b *Backend) handleDriverResponses(ctx context.Context, state *ThreadState, buf []byte) ([]uint32, *BinderTransactionData, error) {
	var commands []uint32
	var reply *BinderTransactionData

	for pos := 0; pos < len(buf); {
		if len(buf[pos:]) < 4 {
			return commands, reply, fmt.Errorf("kernel: short command buffer tail: %d", len(buf[pos:]))
		}

		cmd := binary.LittleEndian.Uint32(buf[pos:])
		commands = append(commands, cmd)
		pos += 4

		switch cmd {
		case BRNoop, BRTransactionComplete, BRSpawnLooper, BRFinished:
		case BRError:
			if len(buf[pos:]) < 4 {
				return commands, reply, fmt.Errorf("kernel: short BR_ERROR payload")
			}
			code := int32(binary.LittleEndian.Uint32(buf[pos:]))
			return commands, reply, fmt.Errorf("kernel: BR_ERROR %d", code)
		case BRIncRefs, BRAcquire, BRRelease, BRDecRefs:
			if len(buf[pos:]) < 16 {
				return commands, reply, fmt.Errorf("kernel: short ptr/cookie payload for %#x", cmd)
			}
			ptr := uintptr(binary.LittleEndian.Uint64(buf[pos:]))
			cookie := uintptr(binary.LittleEndian.Uint64(buf[pos+8:]))
			pos += 16

			switch cmd {
			case BRIncRefs:
				if err := b.Driver.WritePtrCookieCommand(BCIncRefsDone, ptr, cookie); err != nil {
					return commands, reply, err
				}
			case BRAcquire:
				if err := b.Driver.WritePtrCookieCommand(BCAcquireDone, ptr, cookie); err != nil {
					return commands, reply, err
				}
			}
		case BRDeadBinder, BRClearDeathNotificationDone:
			if len(buf[pos:]) < 8 {
				return commands, reply, fmt.Errorf("kernel: short cookie payload for %#x", cmd)
			}
			cookie := uintptr(binary.LittleEndian.Uint64(buf[pos:]))
			pos += 8

			switch cmd {
			case BRDeadBinder:
				if b.Driver.deaths != nil {
					b.Driver.deaths.NotifyDead(cookie)
				}
				if err := b.Driver.DeadBinderDone(cookie); err != nil {
					return commands, reply, err
				}
			case BRClearDeathNotificationDone:
				if b.Driver.deaths != nil {
					b.Driver.deaths.NotifyCleared(cookie)
				}
			}
		case BRReply:
			if len(buf[pos:]) < binderTransactionDataSize {
				return commands, reply, fmt.Errorf("kernel: short BR_REPLY payload: have %d want %d", len(buf[pos:]), binderTransactionDataSize)
			}
			var tx BinderTransactionData
			if err := tx.UnmarshalBinary(buf[pos : pos+binderTransactionDataSize]); err != nil {
				return commands, reply, err
			}
			reply = &tx
			pos += binderTransactionDataSize
		case BRTransaction:
			if len(buf[pos:]) < binderTransactionDataSize {
				return commands, reply, fmt.Errorf("kernel: short BR_TRANSACTION payload: have %d want %d", len(buf[pos:]), binderTransactionDataSize)
			}
			var tx BinderTransactionData
			if err := tx.UnmarshalBinary(buf[pos : pos+binderTransactionDataSize]); err != nil {
				return commands, reply, err
			}
			pos += binderTransactionDataSize
			if err := b.handleIncomingTransaction(ctx, state, &tx); err != nil {
				return commands, reply, err
			}
		case BRDeadReply:
			return commands, reply, ErrDeadReply
		case BRFailedReply:
			return commands, reply, ErrFailedReply
		default:
			return commands, reply, fmt.Errorf("kernel: unsupported binder response command %#x", cmd)
		}
	}

	return commands, reply, nil
}

func (b *Backend) handleIncomingTransaction(ctx context.Context, state *ThreadState, tx *BinderTransactionData) error {
	if tx == nil {
		return fmt.Errorf("kernel: nil transaction")
	}
	tracef("client-thread incoming transaction: code=%d flags=%#x cookie=%#x target=%#x pid=%d uid=%d", tx.Code, tx.Flags, tx.CookiePointer(), tx.TargetPointer(), tx.SenderPID, tx.SenderEUID)
	if tx.DataBuffer != 0 {
		defer func() {
			_ = b.Driver.FreeBuffer(tx.BufferPointer())
		}()
	}

	requestBytes, requestObjects, err := copyTransactionPayload(tx)
	if err != nil {
		if tx.Flags&TFOneWay == 0 {
			return b.sendStatusReply(statusCodeFromError(err), tx.Flags&TFClearBuf)
		}
		return err
	}
	if err := b.Driver.acquireParcelObjects(requestObjects); err != nil {
		if tx.Flags&TFOneWay == 0 {
			return b.sendStatusReply(statusCodeFromError(err), tx.Flags&TFClearBuf)
		}
		return err
	}

	request := api.NewParcelWire(requestBytes, requestObjects)
	request.SetBinderResolvers(b.binderResolver, b.localResolver)
	request.SetBinderObjectResolvers(b.binderObjectResolver, b.localObjectResolver)
	reply, err := b.dispatchLocalTransaction(ctx, tx.CookiePointer(), tx.Code, request, tx.Flags, transactionMetadata{
		CallingPID: tx.SenderPID,
		CallingUID: tx.SenderEUID,
		Code:       tx.Code,
		Flags:      tx.Flags,
	})
	if tx.Flags&TFOneWay != 0 {
		tracef("client-thread incoming oneway handled: code=%d cookie=%#x err=%v", tx.Code, tx.CookiePointer(), err)
		return nil
	}
	if err != nil {
		tracef("client-thread incoming transaction failed: code=%d cookie=%#x err=%v", tx.Code, tx.CookiePointer(), err)
		return b.sendStatusReply(statusCodeFromError(err), tx.Flags&TFClearBuf)
	}
	tracef("client-thread incoming transaction replying: code=%d cookie=%#x", tx.Code, tx.CookiePointer())
	return b.sendReply(reply, tx.Flags&TFClearBuf)
}

func (b *Backend) sendReply(reply *api.Parcel, flags uint32) error {
	var tx BinderTransactionData
	tx.SetTargetHandle(^uint32(0))
	tx.Flags = flags

	var payload []byte
	var offsets []uint64
	if reply != nil {
		payload, offsets = reply.KernelWireData()
	}
	setTransactionPayload(&tx, payload, offsets)

	bwr, err := b.Driver.WriteCommand(BCReply, tx.MarshalBinary(), nil)
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

func (b *Backend) sendStatusReply(code int32, flags uint32) error {
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, uint32(code))

	var tx BinderTransactionData
	tx.SetTargetHandle(^uint32(0))
	tx.Flags = TFStatus | (flags & TFClearBuf)
	setTransactionPayload(&tx, payload, nil)

	bwr, err := b.Driver.WriteCommand(BCReply, tx.MarshalBinary(), nil)
	runtime.KeepAlive(payload)
	if err != nil {
		return err
	}
	if want := uint64(4 + binderTransactionDataSize); bwr.WriteConsumed != want {
		return fmt.Errorf("kernel: BC_REPLY(status) consumed %d bytes, want %d", bwr.WriteConsumed, want)
	}
	return nil
}
