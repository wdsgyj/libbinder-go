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

type metadataServer struct {
	prefix string
}

func (s metadataServer) Echo(ctx context.Context, value string) (string, error) {
	return s.prefix + ":" + value, nil
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.metadata", "service manager name")
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

	handler := shared.NewIMetadataServiceHandler(metadataServer{prefix: *prefix})
	if err := conn.ServiceManager().AddService(context.Background(), *serviceName, handler); err != nil {
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
