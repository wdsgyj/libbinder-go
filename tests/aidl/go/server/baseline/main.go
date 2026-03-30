package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	libbinder "github.com/wdsgyj/libbinder-go"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type baselineServer struct{}

func (baselineServer) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

func (baselineServer) EchoNullable(ctx context.Context, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	reply := "go:" + *value
	return &reply, nil
}

func (baselineServer) Transform(ctx context.Context, input int32, payload shared.BaselinePayload) (int32, shared.BaselinePayload, shared.BaselinePayload, error) {
	doubled := shared.BaselinePayload{
		Code: input * 2,
		Note: strPtr("go:doubled"),
	}
	payload.Code += doubled.Code
	if payload.Note == nil {
		payload.Note = strPtr("go:default")
	} else {
		payload.Note = strPtr("go:" + *payload.Note)
	}
	return input + 1, doubled, payload, nil
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.baseline", "service manager name")
	flag.Parse()

	conn, err := libbinder.Open(libbinder.Config{
		DriverPath:    *driverPath,
		LooperWorkers: 1,
		ClientWorkers: 1,
	})
	if err != nil {
		fatalf("open binder: %v", err)
	}
	defer conn.Close()

	if err := shared.AddIBaselineServiceService(context.Background(), conn.ServiceManager(), *serviceName, baselineServer{}); err != nil {
		fatalf("add service %q: %v", *serviceName, err)
	}

	fmt.Fprintf(os.Stdout, "registered %s\n", *serviceName)
	waitForSignal()
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func strPtr(v string) *string {
	return &v
}
