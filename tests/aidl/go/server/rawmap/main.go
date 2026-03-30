package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	libbinder "github.com/wdsgyj/libbinder-go"
	"github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/cases"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type rawMapServer struct {
	prefix string
}

func (s rawMapServer) Normalize(ctx context.Context, value map[any]any) (map[any]any, error) {
	return cases.NormalizeRawMap(s.prefix, value), nil
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.rawmap", "service manager name")
	prefix := flag.String("prefix", "go", "response prefix")
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

	if err := shared.AddIRawMapServiceService(context.Background(), conn.ServiceManager(), *serviceName, rawMapServer{prefix: *prefix}); err != nil {
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
