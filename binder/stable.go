package binder

import "context"

// InterfaceVersionProvider exposes a stable AIDL interface version.
type InterfaceVersionProvider interface {
	InterfaceVersion() int32
}

// InterfaceHashProvider exposes a stable AIDL interface hash.
type InterfaceHashProvider interface {
	InterfaceHash() string
}

// GetInterfaceVersion queries the reserved stable-AIDL version transaction.
func GetInterfaceVersion(ctx context.Context, b Binder) (int32, error) {
	if b == nil {
		return 0, ErrUnsupported
	}
	if provider, ok := b.(InterfaceVersionProvider); ok {
		return provider.InterfaceVersion(), nil
	}

	reply, err := b.Transact(ctx, GetInterfaceVersionTransaction, NewParcel(), FlagNone)
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, ErrBadParcelable
	}
	return reply.ReadInt32()
}

// GetInterfaceHash queries the reserved stable-AIDL hash transaction.
func GetInterfaceHash(ctx context.Context, b Binder) (string, error) {
	if b == nil {
		return "", ErrUnsupported
	}
	if provider, ok := b.(InterfaceHashProvider); ok {
		return provider.InterfaceHash(), nil
	}

	reply, err := b.Transact(ctx, GetInterfaceHashTransaction, NewParcel(), FlagNone)
	if err != nil {
		return "", err
	}
	if reply == nil {
		return "", ErrBadParcelable
	}
	return reply.ReadString()
}
