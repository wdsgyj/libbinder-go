package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"time"

	libbinder "github.com/wdsgyj/libbinder-go"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.baseline", "service manager name")
	expectPrefix := flag.String("expect-prefix", "java", "expected service output prefix")
	timeout := flag.Duration("timeout", 5*time.Second, "overall call timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder: %v", err)
	}
	defer conn.Close()

	svc, err := shared.WaitIBaselineServiceService(ctx, conn.ServiceManager(), *serviceName)
	if err != nil {
		fatalf("wait service %q: %v", *serviceName, err)
	}

	ping, err := svc.Ping(ctx)
	if err != nil {
		fatalf("ping: %v", err)
	}
	if !ping {
		fatalf("ping: got false")
	}

	echo, err := svc.EchoNullable(ctx, strPtr("hello"))
	if err != nil {
		fatalf("echoNullable: %v", err)
	}
	if !reflect.DeepEqual(echo, strPtr(*expectPrefix+":hello")) {
		fatalf("echoNullable: got=%#v want=%#v", echo, strPtr(*expectPrefix+":hello"))
	}

	payload := shared.BaselinePayload{
		Code: 7,
		Note: strPtr("seed"),
	}
	transform, doubled, payloadOut, err := svc.Transform(ctx, 11, payload)
	if err != nil {
		fatalf("transform: %v", err)
	}
	if transform != 12 {
		fatalf("transform: got=%d want=12", transform)
	}
	wantDoubled := shared.BaselinePayload{Code: 22, Note: strPtr(*expectPrefix + ":doubled")}
	if !reflect.DeepEqual(doubled, wantDoubled) {
		fatalf("doubled: got=%#v want=%#v", doubled, wantDoubled)
	}
	wantPayloadOut := shared.BaselinePayload{Code: 29, Note: strPtr(*expectPrefix + ":seed")}
	if !reflect.DeepEqual(payloadOut, wantPayloadOut) {
		fatalf("payloadOut: got=%#v want=%#v", payloadOut, wantPayloadOut)
	}

	fmt.Fprintln(os.Stdout, "OK")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func strPtr(v string) *string {
	return &v
}
