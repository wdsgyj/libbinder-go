package binder

import (
	"context"
	"sync"
)

// DispatchLocalHandler executes Binder reserved transactions and then calls the local handler.
func DispatchLocalHandler(ctx context.Context, handler Handler, serial sync.Locker, code uint32, data *Parcel, flags Flags, tx TransactionContext) (*Parcel, error) {
	if handler == nil {
		return nil, ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if data == nil {
		data = NewParcel()
	}

	tx.Code = code
	tx.Flags = flags
	ctx = WithTransactionContext(ctx, tx)

	if serial != nil {
		serial.Lock()
		defer serial.Unlock()
	}

	switch code {
	case InterfaceTransaction:
		reply := NewParcel()
		if err := reply.WriteString(handler.Descriptor()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case PingTransaction:
		return NewParcel(), nil
	case GetInterfaceVersionTransaction:
		provider, ok := handler.(InterfaceVersionProvider)
		if !ok {
			return nil, ErrUnknownTransaction
		}
		reply := NewParcel()
		if err := reply.WriteInt32(provider.InterfaceVersion()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	case GetInterfaceHashTransaction:
		provider, ok := handler.(InterfaceHashProvider)
		if !ok {
			return nil, ErrUnknownTransaction
		}
		reply := NewParcel()
		if err := reply.WriteString(provider.InterfaceHash()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}

	reply, err := handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if flags&FlagOneway != 0 {
		return nil, nil
	}
	if reply == nil {
		return NewParcel(), nil
	}
	if err := reply.SetPosition(0); err != nil {
		return nil, err
	}
	return reply, nil
}
