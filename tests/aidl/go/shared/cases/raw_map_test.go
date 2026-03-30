package cases

import (
	"context"
	"testing"

	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type rawMapServiceImpl struct {
	prefix string
}

func (s rawMapServiceImpl) Normalize(ctx context.Context, value map[any]any) (map[any]any, error) {
	return NormalizeRawMap(s.prefix, value), nil
}

func TestVerifyRawMapService(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewIRawMapServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIRawMapServiceHandler(rawMapServiceImpl{prefix: "go"}),
		registrar: reg,
	})
	if err := VerifyRawMapService(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}
