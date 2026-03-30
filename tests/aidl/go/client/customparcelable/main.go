package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	libbinder "github.com/wdsgyj/libbinder-go"
	"github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/cases"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.customparcelable", "service manager name")
	expectPrefix := flag.String("expect-prefix", "java", "expected service output prefix")
	timeout := flag.Duration("timeout", 10*time.Second, "overall call timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder: %v", err)
	}
	defer conn.Close()

	svc, err := shared.WaitICustomParcelableServiceService(ctx, conn.ServiceManager(), *serviceName)
	if err != nil {
		fatalf("wait service %q: %v", *serviceName, err)
	}
	if err := cases.VerifyCustomParcelableService(ctx, svc, *expectPrefix); err != nil {
		fatalf("verify custom parcelable service: %v", err)
	}

	fmt.Fprintln(os.Stdout, "OK")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
