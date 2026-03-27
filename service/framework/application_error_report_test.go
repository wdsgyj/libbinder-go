package framework

import (
	"reflect"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestApplicationErrorReportParcelableCrashInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	handler := "handler"
	className := "java.lang.IllegalStateException"
	message := "boom"
	fileName := "MainActivity.java"
	throwClass := "com.example.MainActivity"
	throwMethod := "onCreate"
	stackTrace := "stack"
	crashTag := "tag"
	value := ApplicationErrorReportParcelableCrashInfo{
		ExceptionHandlerClassName: &handler,
		ExceptionClassName:        &className,
		ExceptionMessage:          &message,
		ThrowFileName:             &fileName,
		ThrowClassName:            &throwClass,
		ThrowMethodName:           &throwMethod,
		ThrowLineNumber:           123,
		StackTrace:                &stackTrace,
		CrashTag:                  &crashTag,
	}
	if err := WriteApplicationErrorReportParcelableCrashInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteApplicationErrorReportParcelableCrashInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadApplicationErrorReportParcelableCrashInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadApplicationErrorReportParcelableCrashInfoFromParcel: %v", err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}
