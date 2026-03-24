//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"

	api "libbinder-go/binder"
)

const binderFlatObjectSize = 24

func (d *DriverManager) TransactHandle(handle uint32, code uint32, payload []byte, flags api.Flags) ([]byte, []uint32, error) {
	replyBytes, _, commands, err := d.TransactHandleParcel(handle, code, payload, flags)
	if err != nil {
		return nil, commands, err
	}
	return replyBytes, commands, nil
}

func (d *DriverManager) TransactHandleParcel(handle uint32, code uint32, payload []byte, flags api.Flags) ([]byte, []api.ParcelObject, []uint32, error) {
	var tx BinderTransactionData
	tx.SetTargetHandle(handle)
	tx.Code = code
	tx.Flags = TFAcceptFD
	if flags&api.FlagOneway != 0 {
		tx.Flags |= TFOneWay
	}
	if len(payload) > 0 {
		tx.DataSize = uint64(len(payload))
		tx.DataBuffer = uint64(uintptr(unsafe.Pointer(&payload[0])))
	}

	readBuf := make([]byte, 256)
	bwr, err := d.WriteCommand(BCTransaction, tx.MarshalBinary(), readBuf)
	runtime.KeepAlive(payload)
	if err != nil {
		return nil, nil, nil, err
	}

	commands, reply, err := parseDriverResponses(readBuf[:bwr.ReadConsumed])
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
		moreCommands, moreReply, err := parseDriverResponses(readBuf[:next.ReadConsumed])
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

	replyBytes, replyObjects, err := copyReplyPayload(reply)
	if err != nil {
		return nil, nil, commands, err
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

func parseDriverResponses(buf []byte) ([]uint32, *BinderTransactionData, error) {
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
		case BRNoop, BRTransactionComplete:
		case BRError:
			if len(buf[pos:]) < 4 {
				return commands, reply, fmt.Errorf("kernel: short BR_ERROR payload")
			}
			code := int32(binary.LittleEndian.Uint32(buf[pos:]))
			return commands, reply, fmt.Errorf("kernel: BR_ERROR %d", code)
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

func copyReplyPayload(reply *BinderTransactionData) ([]byte, []api.ParcelObject, error) {
	if reply == nil || reply.DataSize == 0 {
		return nil, nil, nil
	}
	if reply.DataBuffer == 0 {
		return nil, nil, fmt.Errorf("kernel: reply buffer pointer is nil for %d bytes", reply.DataSize)
	}

	size := int(reply.DataSize)
	if uint64(size) != reply.DataSize {
		return nil, nil, fmt.Errorf("kernel: reply payload too large: %d", reply.DataSize)
	}

	src := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(reply.DataBuffer))), size)
	out := make([]byte, len(src))
	copy(out, src)

	if reply.OffsetsSize == 0 {
		return out, nil, nil
	}

	objects, err := parseReplyObjects(out, reply)
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

func parseReplyObjects(payload []byte, reply *BinderTransactionData) ([]api.ParcelObject, error) {
	if reply == nil || reply.OffsetsSize == 0 {
		return nil, nil
	}
	if reply.DataOffsets == 0 {
		return nil, fmt.Errorf("kernel: reply offsets pointer is nil for %d bytes", reply.OffsetsSize)
	}
	if reply.OffsetsSize%8 != 0 {
		return nil, fmt.Errorf("kernel: invalid reply offsets size %d", reply.OffsetsSize)
	}

	offsetsSize := int(reply.OffsetsSize)
	if uint64(offsetsSize) != reply.OffsetsSize {
		return nil, fmt.Errorf("kernel: reply offsets too large: %d", reply.OffsetsSize)
	}

	offsetSrc := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(reply.DataOffsets))), offsetsSize)
	objects := make([]api.ParcelObject, 0, offsetsSize/8)
	for pos := 0; pos < len(offsetSrc); pos += 8 {
		offset := binary.LittleEndian.Uint64(offsetSrc[pos:])
		obj, err := parseReplyObject(payload, offset)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func parseReplyObject(payload []byte, offset uint64) (api.ParcelObject, error) {
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
