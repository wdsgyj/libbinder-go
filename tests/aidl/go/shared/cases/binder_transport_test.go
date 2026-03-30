package cases

import (
	"context"
	"testing"
	"time"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func TestBinderTransportWithoutServiceManagerRawBinderRoundTrip(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	svc := shared.NewIAdvancedServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIAdvancedServiceHandlerWithRegistrar(reg, advancedServiceImpl{prefix: "go"}),
		registrar: reg,
	})

	rawBinder, err := reg.RegisterLocalHandler(binder.StaticHandler{
		DescriptorName: AdvancedRawBinderDescriptor,
		Handle: func(ctx context.Context, code uint32, data *binder.Parcel) (*binder.Parcel, error) {
			return binder.NewParcel(), nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterLocalHandler(raw): %v", err)
	}
	defer rawBinder.Close()

	got, err := svc.EchoBinder(context.Background(), rawBinder)
	if err != nil {
		t.Fatalf("EchoBinder: %v", err)
	}
	if got == nil {
		t.Fatal("EchoBinder = nil, want non-nil binder")
	}
	desc, err := got.Descriptor(context.Background())
	if err != nil {
		t.Fatalf("Descriptor: %v", err)
	}
	if desc != AdvancedRawBinderDescriptor {
		t.Fatalf("Descriptor = %q, want %q", desc, AdvancedRawBinderDescriptor)
	}
}

func TestBinderTransportWithoutServiceManagerTypedCallbackRoundTrip(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	svc := shared.NewIAdvancedServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIAdvancedServiceHandlerWithRegistrar(reg, advancedServiceImpl{prefix: "go"}),
		registrar: reg,
	})

	callback := newAdvancedCallbackRecorder(advancedCallbackPrefix)
	reply, err := svc.InvokeCallback(context.Background(), callback, "sync-value")
	if err != nil {
		t.Fatalf("InvokeCallback: %v", err)
	}
	if want := "go:" + advancedCallbackPrefix + ":sync-value"; reply != want {
		t.Fatalf("InvokeCallback reply = %q, want %q", reply, want)
	}
	if got := callback.lastSync(); got != "sync-value" {
		t.Fatalf("InvokeCallback arg = %q, want %q", got, "sync-value")
	}

	if err := svc.FireOneway(context.Background(), callback, "oneway-value"); err != nil {
		t.Fatalf("FireOneway: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := callback.waitOneway(waitCtx)
	if err != nil {
		t.Fatalf("waitOneway: %v", err)
	}
	if want := "go:oneway-value"; got != want {
		t.Fatalf("FireOneway arg = %q, want %q", got, want)
	}
}

func TestBinderTransportWithoutServiceManagerCallReturnedAIDLServerBinder(t *testing.T) {
	reg := newFakeAdvancedRegistrar()
	svc := shared.NewIAdvancedServiceClient(fakeAdvancedEndpoint{
		handler:   shared.NewIAdvancedServiceHandlerWithRegistrar(reg, advancedServiceImpl{prefix: "go"}),
		registrar: reg,
	})

	localServer := newAdvancedCallbackRecorder("returned")
	localServerBinder, err := reg.RegisterLocalHandler(shared.NewIAdvancedCallbackHandler(localServer))
	if err != nil {
		t.Fatalf("RegisterLocalHandler(callback server): %v", err)
	}
	defer localServerBinder.Close()

	returnedBinder, err := svc.EchoBinder(context.Background(), localServerBinder)
	if err != nil {
		t.Fatalf("EchoBinder(callback server): %v", err)
	}
	if returnedBinder == nil {
		t.Fatal("EchoBinder(callback server) = nil, want non-nil binder")
	}

	callbackClient := shared.NewIAdvancedCallbackClient(returnedBinder)
	if callbackClient == nil {
		t.Fatal("NewIAdvancedCallbackClient(returnedBinder) = nil")
	}

	reply, err := callbackClient.OnSync(context.Background(), "ping")
	if err != nil {
		t.Fatalf("OnSync over returned binder: %v", err)
	}
	if want := "returned:ping"; reply != want {
		t.Fatalf("OnSync reply = %q, want %q", reply, want)
	}
	if got := localServer.lastSync(); got != "ping" {
		t.Fatalf("OnSync server arg = %q, want %q", got, "ping")
	}

	if err := callbackClient.OnOneway(context.Background(), "fire"); err != nil {
		t.Fatalf("OnOneway over returned binder: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := localServer.waitOneway(waitCtx)
	if err != nil {
		t.Fatalf("waitOneway over returned binder: %v", err)
	}
	if want := "fire"; got != want {
		t.Fatalf("OnOneway server arg = %q, want %q", got, want)
	}
}
