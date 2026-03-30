package cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

const (
	MetadataVersion = shared.IMetadataServiceInterfaceVersion
	MetadataHash    = shared.IMetadataServiceInterfaceHash
)

func VerifyMetadataService(ctx context.Context, svc shared.IMetadataService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	value, err := svc.Echo(ctx, "hello")
	if err != nil {
		return fmt.Errorf("Echo: %w", err)
	}
	want := prefix + ":hello"
	if value != want {
		return fmt.Errorf("Echo = %q, want %q", value, want)
	}

	version, err := shared.GetIMetadataServiceInterfaceVersion(ctx, svc)
	if err != nil {
		return fmt.Errorf("InterfaceVersion: %w", err)
	}
	if version != MetadataVersion {
		return fmt.Errorf("InterfaceVersion = %d, want %d", version, MetadataVersion)
	}

	hash, err := shared.GetIMetadataServiceInterfaceHash(ctx, svc)
	if err != nil {
		return fmt.Errorf("InterfaceHash: %w", err)
	}
	if hash != MetadataHash {
		return fmt.Errorf("InterfaceHash = %q, want %q", hash, MetadataHash)
	}

	provider, ok := any(svc).(binder.BinderProvider)
	if !ok || provider.AsBinder() == nil {
		return fmt.Errorf("service does not expose binder provider")
	}
	rawVersion, err := binder.GetInterfaceVersion(ctx, provider.AsBinder())
	if err != nil {
		return fmt.Errorf("raw GetInterfaceVersion: %w", err)
	}
	if rawVersion != MetadataVersion {
		return fmt.Errorf("raw GetInterfaceVersion = %d, want %d", rawVersion, MetadataVersion)
	}
	rawHash, err := binder.GetInterfaceHash(ctx, provider.AsBinder())
	if err != nil {
		return fmt.Errorf("raw GetInterfaceHash: %w", err)
	}
	if rawHash != MetadataHash {
		return fmt.Errorf("raw GetInterfaceHash = %q, want %q", rawHash, MetadataHash)
	}

	req := binder.NewParcel()
	if err := req.WriteInterfaceToken(shared.IMetadataServiceDescriptor); err != nil {
		return fmt.Errorf("write interface token for unknown transaction: %w", err)
	}
	if _, err := provider.AsBinder().Transact(ctx, shared.IMetadataServiceTransactionEcho+99, req, binder.FlagNone); !errors.Is(err, binder.ErrUnknownTransaction) {
		return fmt.Errorf("unknown transaction error = %v, want %v", err, binder.ErrUnknownTransaction)
	}
	return nil
}
