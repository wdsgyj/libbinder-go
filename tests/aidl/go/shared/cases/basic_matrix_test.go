package cases

import (
	"context"
	"errors"
	"testing"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type fakeBinder struct {
	handler binder.Handler
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeBinder) Close() error { return nil }

type basicMatrixImpl struct {
	prefix string
}

func (m basicMatrixImpl) EchoNullable(ctx context.Context, value *string) (*string, error) {
	return EchoNullable(m.prefix, value), nil
}

func (m basicMatrixImpl) ReverseInts(ctx context.Context, values []int32) ([]int32, error) {
	return ReverseInts(values), nil
}

func (m basicMatrixImpl) RotateTriple(ctx context.Context, triple []int32) ([]int32, error) {
	return RotateTriple(triple), nil
}

func (m basicMatrixImpl) DecorateTags(ctx context.Context, tags []string) ([]string, error) {
	return DecorateTags(m.prefix, tags), nil
}

func (m basicMatrixImpl) DecorateTagGroups(ctx context.Context, groups []shared.BasicStringGroup) ([]shared.BasicStringGroup, error) {
	return DecorateTagGroups(m.prefix, groups), nil
}

func (m basicMatrixImpl) DecoratePayloads(ctx context.Context, payloads []shared.BaselinePayload) ([]shared.BaselinePayload, error) {
	return DecoratePayloads(m.prefix, payloads), nil
}

func (m basicMatrixImpl) DecorateLabels(ctx context.Context, labels map[string]string) (map[string]string, error) {
	return DecorateLabels(m.prefix, labels), nil
}

func (m basicMatrixImpl) DecoratePayloadMap(ctx context.Context, payloadMap map[string]shared.BaselinePayload) (map[string]shared.BaselinePayload, error) {
	return DecoratePayloadMap(m.prefix, payloadMap), nil
}

func (m basicMatrixImpl) DecoratePayloadBuckets(ctx context.Context, payloadBuckets map[string][]shared.BaselinePayload) (map[string][]shared.BaselinePayload, error) {
	return DecoratePayloadBuckets(m.prefix, payloadBuckets), nil
}

func (m basicMatrixImpl) FlipMode(ctx context.Context, mode shared.BasicMode) (shared.BasicMode, error) {
	return FlipMode(mode), nil
}

func (m basicMatrixImpl) NormalizeUnion(ctx context.Context, value *shared.BasicUnion) (*shared.BasicUnion, error) {
	out := NormalizeUnion(m.prefix, *value)
	return &out, nil
}

func (m basicMatrixImpl) NormalizeBundle(ctx context.Context, value *shared.BasicBundle) (*shared.BasicBundle, error) {
	out := NormalizeBundle(m.prefix, *value)
	return &out, nil
}

func (m basicMatrixImpl) NormalizeEnvelope(ctx context.Context, value *shared.BasicEnvelope) (*shared.BasicEnvelope, error) {
	out := NormalizeEnvelope(m.prefix, *value)
	return &out, nil
}

func (m basicMatrixImpl) ExpandBundle(ctx context.Context, input *shared.BasicBundle, payload *shared.BasicBundle) (int32, *shared.BasicBundle, *shared.BasicBundle, error) {
	ret, doubled, payloadOut := ExpandBundle(m.prefix, *input, *payload)
	return ret, &doubled, &payloadOut, nil
}

func TestVerifyBasicMatrixService(t *testing.T) {
	client := shared.NewIBasicMatrixServiceClient(fakeBinder{
		handler: shared.NewIBasicMatrixServiceHandler(basicMatrixImpl{prefix: "go"}),
	})
	if err := VerifyBasicMatrixService(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBasicMatrixPerformance(t *testing.T) {
	client := shared.NewIBasicMatrixServiceClient(fakeBinder{
		handler: shared.NewIBasicMatrixServiceHandler(basicMatrixImpl{prefix: "go"}),
	})
	if err := VerifyBasicMatrixPerformance(context.Background(), client, "go"); err != nil {
		t.Fatal(err)
	}
}

func TestRotateTripleRejectsWrongLength(t *testing.T) {
	client := shared.NewIBasicMatrixServiceClient(fakeBinder{
		handler: shared.NewIBasicMatrixServiceHandler(basicMatrixImpl{prefix: "go"}),
	})
	_, err := client.RotateTriple(context.Background(), []int32{1, 2})
	if !errors.Is(err, binder.ErrBadParcelable) {
		t.Fatalf("RotateTriple error = %v, want ErrBadParcelable", err)
	}
}
