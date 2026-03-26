package generatedfixture

import (
	"context"
	"testing"

	"github.com/wdsgyj/libbinder-go/binder"
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

type echoImpl struct{}

func (echoImpl) Echo(ctx context.Context, payload Payload) (Payload, error) {
	payload.Count++
	return payload, nil
}

func TestGeneratedFixtureRoundTrip(t *testing.T) {
	client := NewIEchoClient(fakeBinder{handler: NewIEchoHandler(echoImpl{})})

	got, err := client.Echo(context.Background(), Payload{Count: 41})
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if got.Count != 42 {
		t.Fatalf("got.Count = %d, want 42", got.Count)
	}
}
