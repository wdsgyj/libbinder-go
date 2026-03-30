package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	libbinder "github.com/wdsgyj/libbinder-go"
	"github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/cases"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type advancedServer struct {
	prefix string
}

func (s advancedServer) EchoBinder(ctx context.Context, input binder.Binder) (binder.Binder, error) {
	return input, nil
}

func (s advancedServer) InvokeCallback(ctx context.Context, callback shared.IAdvancedCallback, value string) (string, error) {
	return cases.InvokeAdvancedCallback(ctx, s.prefix, callback, value)
}

func (s advancedServer) FireOneway(ctx context.Context, callback shared.IAdvancedCallback, value string) error {
	return cases.FireAdvancedOneway(ctx, s.prefix, callback, value)
}

func (s advancedServer) FailServiceSpecific(ctx context.Context, code int32, message string) error {
	return &binder.ServiceSpecificError{Code: code, Message: message}
}

func (s advancedServer) ReadFromFileDescriptor(ctx context.Context, fd binder.FileDescriptor) (string, error) {
	return cases.ReadAllFromFileDescriptor(fd)
}

func (s advancedServer) ReadFromParcelFileDescriptor(ctx context.Context, fd binder.ParcelFileDescriptor) (string, error) {
	return cases.ReadAllFromParcelFileDescriptor(fd)
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.advanced", "service manager name")
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

	if err := shared.AddIAdvancedServiceService(context.Background(), conn.ServiceManager(), *serviceName, advancedServer{prefix: *prefix}); err != nil {
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
