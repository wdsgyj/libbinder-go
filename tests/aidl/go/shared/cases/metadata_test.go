package cases

import (
	"context"
	"testing"

	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type metadataServiceImpl struct {
	prefix string
}

func (s metadataServiceImpl) Echo(ctx context.Context, value string) (string, error) {
	return s.prefix + ":" + value, nil
}

func TestVerifyMetadataService(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewIMetadataServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIMetadataServiceHandler(metadataServiceImpl{prefix: "go"}),
		registrar: reg,
	})
	if err := VerifyMetadataService(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}
