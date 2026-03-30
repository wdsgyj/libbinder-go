package cases

import (
	"context"
	"testing"

	customcodec "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/customcodec"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type customParcelableServiceImpl struct {
	prefix string
}

func (s customParcelableServiceImpl) Normalize(ctx context.Context, value customcodec.CustomBox) (customcodec.CustomBox, error) {
	return NormalizeCustomParcelable(s.prefix, value), nil
}

func (s customParcelableServiceImpl) NormalizeNullable(ctx context.Context, value *customcodec.CustomBox) (*customcodec.CustomBox, error) {
	if value == nil {
		return nil, nil
	}
	out := NormalizeCustomParcelable(s.prefix, *value)
	return &out, nil
}

func TestVerifyCustomParcelableService(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	client := shared.NewICustomParcelableServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewICustomParcelableServiceHandler(customParcelableServiceImpl{prefix: "go"}),
		registrar: reg,
	})
	if err := VerifyCustomParcelableService(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}
