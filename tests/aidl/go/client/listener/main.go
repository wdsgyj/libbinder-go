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
	serviceName := flag.String("service", "libbinder.go.aidltest.listener", "service manager name")
	_ = flag.String("expect-prefix", "", "ignored legacy flag")
	mode := flag.String("mode", "basic", "verification mode: basic or churn")
	rounds := flag.Int("rounds", 64, "listener churn rounds when -mode churn")
	timeout := flag.Duration("timeout", 10*time.Second, "overall call timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder: %v", err)
	}
	defer conn.Close()

	svc, err := shared.WaitIListenerServiceService(ctx, conn.ServiceManager(), *serviceName)
	if err != nil {
		fatalf("wait service %q: %v", *serviceName, err)
	}
	switch *mode {
	case "basic":
		if err := cases.VerifyListenerService(ctx, nil, svc); err != nil {
			fatalf("verify listener service: %v", err)
		}
	case "churn":
		if err := cases.VerifyListenerChurn(ctx, nil, svc, *rounds); err != nil {
			fatalf("verify listener churn: %v", err)
		}
	default:
		fatalf("unknown mode %q", *mode)
	}

	fmt.Fprintln(os.Stdout, "OK")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
