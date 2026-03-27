package content

import (
	"context"
	"testing"

	"github.com/wdsgyj/libbinder-go/binder"
	service_framework "github.com/wdsgyj/libbinder-go/service/framework"
)

type testBinder struct {
	handler  binder.Handler
	registry *testBinderRegistry
	handle   uint32
}

func stringPtr(v string) *string {
	return &v
}

type testBinderRegistry struct {
	next    uint32
	binders map[uint32]binder.Binder
}

func newTestBinder(handler binder.Handler) testBinder {
	registry := &testBinderRegistry{
		next:    1,
		binders: make(map[uint32]binder.Binder),
	}
	return registry.newBinder(handler)
}

func (r *testBinderRegistry) newBinder(handler binder.Handler) testBinder {
	handle := r.next
	r.next++
	b := testBinder{
		handler:  handler,
		registry: r,
		handle:   handle,
	}
	r.binders[handle] = b
	return b
}

func (b testBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b testBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if b.registry != nil {
		data.SetBinderResolvers(func(handle uint32) binder.Binder {
			return b.registry.binders[handle]
		}, nil)
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if b.registry != nil {
			reply.SetBinderResolvers(func(handle uint32) binder.Binder {
				return b.registry.binders[handle]
			}, nil)
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b testBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b testBinder) Close() error {
	return nil
}

func (b testBinder) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	if b.registry == nil {
		return newTestBinder(handler), nil
	}
	return b.registry.newBinder(handler), nil
}

func (b testBinder) WriteBinderToParcel(p *binder.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type intentReceiverRecorder struct {
	intent       *service_framework.Intent
	resultCode   int32
	data         *string
	extras       *service_framework.Bundle
	ordered      bool
	sticky       bool
	sendingUser  int32
	invokedCount int
}

func (r *intentReceiverRecorder) PerformReceive(ctx context.Context, intent *service_framework.Intent, resultCode int32, data *string, extras *service_framework.Bundle, ordered bool, sticky bool, sendingUser int32) error {
	r.intent = intent
	r.resultCode = resultCode
	r.data = data
	r.extras = extras
	r.ordered = ordered
	r.sticky = sticky
	r.sendingUser = sendingUser
	r.invokedCount++
	return nil
}

type intentSenderRecorder struct {
	code               int32
	intent             *service_framework.Intent
	resolvedType       *string
	whitelistTokenNil  bool
	requiredPermission *string
	options            *service_framework.Bundle
	callbackInvoked    bool
}

func (r *intentSenderRecorder) Send(ctx context.Context, code int32, intent *service_framework.Intent, resolvedType *string, whitelistToken binder.Binder, finishedReceiver IIntentReceiver, requiredPermission *string, options *service_framework.Bundle) error {
	r.code = code
	r.intent = intent
	r.resolvedType = resolvedType
	r.whitelistTokenNil = whitelistToken == nil
	r.requiredPermission = requiredPermission
	r.options = options
	if finishedReceiver != nil {
		r.callbackInvoked = true
		return finishedReceiver.PerformReceive(ctx, intent, code, resolvedType, options, true, false, 11)
	}
	return nil
}

func TestGeneratedIIntentReceiverRoundTrip(t *testing.T) {
	service_framework.SetIntentRedirectProtectionEnabled(false)

	impl := &intentReceiverRecorder{}
	client := NewIIntentReceiverClient(newTestBinder(NewIIntentReceiverHandler(impl)))

	action := "android.intent.action.VIEW"
	payload := "payload"
	intent := &service_framework.Intent{
		Action:     &action,
		Data:       service_framework.ParseURI("content://demo/item/1"),
		Categories: []string{"android.intent.category.DEFAULT"},
		Extras:     service_framework.NewEmptyBundle(),
	}
	extras := service_framework.NewRawBundle([]byte{1, 2, 3, 4}, false)

	if err := client.PerformReceive(context.Background(), intent, 7, &payload, extras, true, false, 10); err != nil {
		t.Fatalf("PerformReceive: %v", err)
	}
	if impl.invokedCount != 1 {
		t.Fatalf("invokedCount = %d, want 1", impl.invokedCount)
	}
	if impl.intent == nil || impl.intent.Action == nil || *impl.intent.Action != action {
		t.Fatalf("intent = %#v, want action %q", impl.intent, action)
	}
	if impl.intent.Data == nil || impl.intent.Data.Value != "content://demo/item/1" {
		t.Fatalf("intent.Data = %#v", impl.intent.Data)
	}
	if impl.data == nil || *impl.data != payload {
		t.Fatalf("data = %#v, want %q", impl.data, payload)
	}
	if impl.extras == nil || string(impl.extras.RawData) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("extras = %#v", impl.extras)
	}
	if !impl.ordered || impl.sticky || impl.sendingUser != 10 {
		t.Fatalf("flags = ordered:%v sticky:%v sendingUser:%d", impl.ordered, impl.sticky, impl.sendingUser)
	}
}

func TestGeneratedIIntentSenderRoundTripWithCallback(t *testing.T) {
	service_framework.SetIntentRedirectProtectionEnabled(false)

	senderImpl := &intentSenderRecorder{}
	callbackImpl := &intentReceiverRecorder{}
	client := NewIIntentSenderClient(newTestBinder(NewIIntentSenderHandler(senderImpl)))

	action := "android.intent.action.SEND"
	resolvedType := "text/plain"
	requiredPermission := "android.permission.DUMP"
	intent := &service_framework.Intent{
		Action:  &action,
		Package: stringPtr("demo.pkg"),
		Extras:  service_framework.NewEmptyBundle(),
	}
	options := service_framework.NewRawBundle([]byte{9, 8, 7, 6}, true)

	if err := client.Send(context.Background(), 23, intent, &resolvedType, nil, callbackImpl, &requiredPermission, options); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if senderImpl.code != 23 {
		t.Fatalf("code = %d, want 23", senderImpl.code)
	}
	if senderImpl.intent == nil || senderImpl.intent.Action == nil || *senderImpl.intent.Action != action {
		t.Fatalf("intent = %#v", senderImpl.intent)
	}
	if senderImpl.resolvedType == nil || *senderImpl.resolvedType != resolvedType {
		t.Fatalf("resolvedType = %#v", senderImpl.resolvedType)
	}
	if !senderImpl.whitelistTokenNil {
		t.Fatal("whitelistTokenNil = false, want true")
	}
	if senderImpl.requiredPermission == nil || *senderImpl.requiredPermission != requiredPermission {
		t.Fatalf("requiredPermission = %#v", senderImpl.requiredPermission)
	}
	if senderImpl.options == nil || !senderImpl.options.Native || string(senderImpl.options.RawData) != string([]byte{9, 8, 7, 6}) {
		t.Fatalf("options = %#v", senderImpl.options)
	}
	if !senderImpl.callbackInvoked {
		t.Fatal("callbackInvoked = false, want true")
	}
	if callbackImpl.invokedCount != 1 {
		t.Fatalf("callback invokedCount = %d, want 1", callbackImpl.invokedCount)
	}
	if callbackImpl.resultCode != 23 || callbackImpl.sendingUser != 11 {
		t.Fatalf("callback result = code:%d sendingUser:%d", callbackImpl.resultCode, callbackImpl.sendingUser)
	}
	if callbackImpl.data == nil || *callbackImpl.data != resolvedType {
		t.Fatalf("callback data = %#v, want %q", callbackImpl.data, resolvedType)
	}
}
