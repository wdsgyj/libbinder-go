package shared_test

import (
	"context"
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

type baselineImpl struct{}

func (baselineImpl) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

func (baselineImpl) EchoNullable(ctx context.Context, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	reply := "echo:" + *value
	return &reply, nil
}

func (baselineImpl) Transform(ctx context.Context, input int32, payload shared.BaselinePayload) (int32, shared.BaselinePayload, shared.BaselinePayload, error) {
	doubled := shared.BaselinePayload{
		Code: input * 2,
		Note: stringPtr("echo:doubled"),
	}
	payload.Code += doubled.Code
	if payload.Note != nil {
		reply := "echo:" + *payload.Note
		payload.Note = &reply
	}
	return input + 1, doubled, payload, nil
}

func TestGeneratedBaselineServiceRoundTrip(t *testing.T) {
	client := shared.NewIBaselineServiceClient(fakeBinder{handler: shared.NewIBaselineServiceHandler(baselineImpl{})})

	ping, err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !ping {
		t.Fatal("Ping = false, want true")
	}

	in := "hello"
	echo, err := client.EchoNullable(context.Background(), &in)
	if err != nil {
		t.Fatalf("EchoNullable: %v", err)
	}
	if echo == nil || *echo != "echo:hello" {
		t.Fatalf("EchoNullable = %#v, want echo:hello", echo)
	}

	seed := "seed"
	ret, doubled, payload, err := client.Transform(context.Background(), 11, shared.BaselinePayload{
		Code: 7,
		Note: &seed,
	})
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if ret != 12 {
		t.Fatalf("Transform ret = %d, want 12", ret)
	}
	if doubled.Code != 22 {
		t.Fatalf("Transform doubled.Code = %d, want 22", doubled.Code)
	}
	if doubled.Note == nil || *doubled.Note != "echo:doubled" {
		t.Fatalf("Transform doubled.Note = %#v, want echo:doubled", doubled.Note)
	}
	if payload.Code != 29 {
		t.Fatalf("Transform payload.Code = %d, want 29", payload.Code)
	}
	if payload.Note == nil || *payload.Note != "echo:seed" {
		t.Fatalf("Transform payload.Note = %#v, want echo:seed", payload.Note)
	}
}

func TestBaselinePayloadStructuredParcelableWireFormat(t *testing.T) {
	p := binder.NewParcel()
	note := "wire"
	want := shared.BaselinePayload{
		Code: 17,
		Note: &note,
	}

	if err := shared.WriteBaselinePayloadToParcel(p, want); err != nil {
		t.Fatalf("WriteBaselinePayloadToParcel: %v", err)
	}
	if err := p.WriteInt32(99); err != nil {
		t.Fatalf("WriteInt32(trailer): %v", err)
	}

	raw := p.Bytes()
	if len(raw) < 4 {
		t.Fatalf("parcel too short: %d", len(raw))
	}
	size := int(int32(raw[0]) | int32(raw[1])<<8 | int32(raw[2])<<16 | int32(raw[3])<<24)
	if size != len(raw)-4 {
		t.Fatalf("structured parcelable size = %d, want %d", size, len(raw)-4)
	}

	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition(reset): %v", err)
	}
	got, err := shared.ReadBaselinePayloadFromParcel(p)
	if err != nil {
		t.Fatalf("ReadBaselinePayloadFromParcel: %v", err)
	}
	if got.Code != want.Code {
		t.Fatalf("Code = %d, want %d", got.Code, want.Code)
	}
	if got.Note == nil || *got.Note != *want.Note {
		t.Fatalf("Note = %#v, want %#v", got.Note, want.Note)
	}

	trailer, err := p.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32(trailer): %v", err)
	}
	if trailer != 99 {
		t.Fatalf("trailer = %d, want 99", trailer)
	}
}

func stringPtr(v string) *string {
	return &v
}
