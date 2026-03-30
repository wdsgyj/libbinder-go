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
	customcodec "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/customcodec"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

type customParcelableServer struct {
	prefix string
}

func (s customParcelableServer) Normalize(ctx context.Context, value customcodec.CustomBox) (customcodec.CustomBox, error) {
	return cases.NormalizeCustomParcelable(s.prefix, value), nil
}

func (s customParcelableServer) NormalizeNullable(ctx context.Context, value *customcodec.CustomBox) (*customcodec.CustomBox, error) {
	if value == nil {
		return nil, nil
	}
	out := cases.NormalizeCustomParcelable(s.prefix, *value)
	return &out, nil
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.customparcelable", "service manager name")
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

	if err := shared.AddICustomParcelableServiceService(context.Background(), conn.ServiceManager(), *serviceName, customParcelableServer{prefix: *prefix}); err != nil {
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
