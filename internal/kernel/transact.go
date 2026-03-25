//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const binderFlatObjectSize = 24

func (d *DriverManager) TransactHandle(handle uint32, code uint32, payload []byte, flags api.Flags) ([]byte, []uint32, error) {
	replyBytes, _, commands, err := d.TransactHandleParcel(handle, code, payload, nil, flags)
	if err != nil {
		return nil, commands, err
	}
	return replyBytes, commands, nil
}

func (d *DriverManager) TransactHandleParcel(handle uint32, code uint32, payload []byte, offsets []uint64, flags api.Flags) ([]byte, []api.ParcelObject, []uint32, error) {
	var tx BinderTransactionData
	tx.SetTargetHandle(handle)
	tx.Code = code
	tx.Flags = TFAcceptFD
	if flags&api.FlagOneway != 0 {
		tx.Flags |= TFOneWay
	}
	setTransactionPayload(&tx, payload, offsets)

	readBuf := make([]byte, 256)
	bwr, err := d.WriteCommand(BCTransaction, tx.MarshalBinary(), readBuf)
	runtime.KeepAlive(payload)
	runtime.KeepAlive(offsets)
	if err != nil {
		return nil, nil, nil, err
	}

	commands, reply, err := d.parseDriverResponses(readBuf[:bwr.ReadConsumed])
	if err != nil {
		return nil, nil, nil, err
	}

	if flags&api.FlagOneway != 0 {
		return nil, nil, commands, nil
	}

	for i := 0; i < 3 && reply == nil; i++ {
		next, err := d.Read(readBuf)
		if err != nil {
			return nil, nil, commands, err
		}
		moreCommands, moreReply, err := d.parseDriverResponses(readBuf[:next.ReadConsumed])
		if err != nil {
			return nil, nil, commands, err
		}
		commands = append(commands, moreCommands...)
		if moreReply != nil {
			reply = moreReply
		}
	}

	if reply == nil {
		return nil, nil, commands, fmt.Errorf("kernel: did not receive BR_REPLY, commands=%v", commandNames(commands))
	}

	replyBytes, replyObjects, err := copyTransactionPayload(reply)
	if err != nil {
		return nil, nil, commands, err
	}
	if reply.Flags&TFStatus != 0 {
		if reply.DataBuffer != 0 {
			if freeErr := d.FreeBuffer(reply.BufferPointer()); freeErr != nil {
				return nil, nil, commands, freeErr
			}
		}
		return nil, nil, commands, statusReplyError(replyBytes)
	}
	if err := d.acquireParcelObjects(replyObjects); err != nil {
		return nil, nil, commands, err
	}
	if reply.DataBuffer != 0 {
		if freeErr := d.FreeBuffer(reply.BufferPointer()); freeErr != nil {
			return nil, nil, commands, freeErr
		}
	}

	return replyBytes, replyObjects, commands, nil
}

func (d *DriverManager) parseDriverResponses(buf []byte) ([]uint32, *BinderTransactionData, error) {
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
				if err := d.WritePtrCookieCommand(BCIncRefsDone, ptr, cookie); err != nil {
					return commands, reply, err
				}
			case BRAcquire:
				if err := d.WritePtrCookieCommand(BCAcquireDone, ptr, cookie); err != nil {
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
				if d.deaths != nil {
					d.deaths.NotifyDead(cookie)
				}
				if err := d.DeadBinderDone(cookie); err != nil {
					return commands, reply, err
				}
			case BRClearDeathNotificationDone:
				if d.deaths != nil {
					d.deaths.NotifyCleared(cookie)
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

func copyTransactionPayload(tx *BinderTransactionData) ([]byte, []api.ParcelObject, error) {
	if tx == nil || tx.DataSize == 0 {
		return nil, nil, nil
	}
	if tx.DataBuffer == 0 {
		return nil, nil, fmt.Errorf("kernel: transaction buffer pointer is nil for %d bytes", tx.DataSize)
	}

	size := int(tx.DataSize)
	if uint64(size) != tx.DataSize {
		return nil, nil, fmt.Errorf("kernel: transaction payload too large: %d", tx.DataSize)
	}

	src := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(tx.DataBuffer))), size)
	out := make([]byte, len(src))
	copy(out, src)

	if tx.OffsetsSize == 0 {
		return out, nil, nil
	}

	objects, err := parseTransactionObjects(out, tx)
	if err != nil {
		return nil, nil, err
	}
	return out, objects, nil
}

func containsCommand(commands []uint32, want uint32) bool {
	for _, cmd := range commands {
		if cmd == want {
			return true
		}
	}
	return false
}

func commandNames(commands []uint32) []string {
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		switch cmd {
		case BRNoop:
			names = append(names, "BR_NOOP")
		case BRTransactionComplete:
			names = append(names, "BR_TRANSACTION_COMPLETE")
		case BRIncRefs:
			names = append(names, "BR_INCREFS")
		case BRAcquire:
			names = append(names, "BR_ACQUIRE")
		case BRRelease:
			names = append(names, "BR_RELEASE")
		case BRDecRefs:
			names = append(names, "BR_DECREFS")
		case BRSpawnLooper:
			names = append(names, "BR_SPAWN_LOOPER")
		case BRFinished:
			names = append(names, "BR_FINISHED")
		case BRDeadBinder:
			names = append(names, "BR_DEAD_BINDER")
		case BRClearDeathNotificationDone:
			names = append(names, "BR_CLEAR_DEATH_NOTIFICATION_DONE")
		case BRReply:
			names = append(names, "BR_REPLY")
		case BRError:
			names = append(names, "BR_ERROR")
		case BRDeadReply:
			names = append(names, "BR_DEAD_REPLY")
		case BRFailedReply:
			names = append(names, "BR_FAILED_REPLY")
		default:
			names = append(names, fmt.Sprintf("%#x", cmd))
		}
	}
	return names
}

func parseTransactionObjects(payload []byte, tx *BinderTransactionData) ([]api.ParcelObject, error) {
	if tx == nil || tx.OffsetsSize == 0 {
		return nil, nil
	}
	if tx.DataOffsets == 0 {
		return nil, fmt.Errorf("kernel: transaction offsets pointer is nil for %d bytes", tx.OffsetsSize)
	}
	if tx.OffsetsSize%8 != 0 {
		return nil, fmt.Errorf("kernel: invalid transaction offsets size %d", tx.OffsetsSize)
	}

	offsetsSize := int(tx.OffsetsSize)
	if uint64(offsetsSize) != tx.OffsetsSize {
		return nil, fmt.Errorf("kernel: transaction offsets too large: %d", tx.OffsetsSize)
	}

	offsetSrc := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(tx.DataOffsets))), offsetsSize)
	objects := make([]api.ParcelObject, 0, offsetsSize/8)
	for pos := 0; pos < len(offsetSrc); pos += 8 {
		offset := binary.LittleEndian.Uint64(offsetSrc[pos:])
		obj, err := parseTransactionObject(payload, offset)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func parseTransactionObject(payload []byte, offset uint64) (api.ParcelObject, error) {
	start := int(offset)
	if uint64(start) != offset {
		return api.ParcelObject{}, fmt.Errorf("kernel: reply object offset out of range: %d", offset)
	}
	if start < 0 || start > len(payload)-binderFlatObjectSize {
		return api.ParcelObject{}, fmt.Errorf("kernel: reply object offset %d out of bounds", offset)
	}

	typ := binary.LittleEndian.Uint32(payload[start:])
	handle := binary.LittleEndian.Uint32(payload[start+8:])

	obj := api.ParcelObject{
		Offset: start,
		Length: binderFlatObjectSize,
		Handle: handle,
	}

	switch typ {
	case BinderTypeHandle:
		obj.Kind = api.ObjectStrongBinder
	case BinderTypeWeakHandle:
		obj.Kind = api.ObjectWeakBinder
	case BinderTypeFD:
		obj.Kind = api.ObjectFileDescriptor
	default:
		return api.ParcelObject{}, fmt.Errorf("%w: unsupported binder object type %#x", ErrParcelObjects, typ)
	}

	return obj, nil
}

func setTransactionPayload(tx *BinderTransactionData, payload []byte, offsets []uint64) {
	if tx == nil {
		return
	}
	if len(payload) > 0 {
		tx.DataSize = uint64(len(payload))
		tx.DataBuffer = uint64(uintptr(unsafe.Pointer(&payload[0])))
	}
	if len(offsets) > 0 {
		tx.OffsetsSize = uint64(len(offsets) * 8)
		tx.DataOffsets = uint64(uintptr(unsafe.Pointer(&offsets[0])))
	}
}

func (d *DriverManager) acquireParcelObjects(objects []api.ParcelObject) error {
	for _, obj := range objects {
		if obj.Kind != api.ObjectStrongBinder {
			continue
		}
		if err := d.AcquireHandle(obj.Handle); err != nil {
			return err
		}
	}
	return nil
}

func statusReplyError(payload []byte) error {
	if len(payload) < 4 {
		return ErrFailedReply
	}
	code := int32(binary.LittleEndian.Uint32(payload[:4]))
	return fmt.Errorf("kernel: transaction status %d", code)
}
