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

type matrixServer struct {
	prefix string
}

func (m matrixServer) EchoNullable(ctx context.Context, value *string) (*string, error) {
	return cases.EchoNullable(m.prefix, value), nil
}

func (m matrixServer) ReverseInts(ctx context.Context, values []int32) ([]int32, error) {
	return cases.ReverseInts(values), nil
}

func (m matrixServer) RotateTriple(ctx context.Context, triple [3]int32) ([3]int32, error) {
	return cases.RotateTriple(triple), nil
}

func (m matrixServer) DecorateTags(ctx context.Context, tags []string) ([]string, error) {
	return cases.DecorateTags(m.prefix, tags), nil
}

func (m matrixServer) DecoratePayloads(ctx context.Context, payloads []shared.BaselinePayload) ([]shared.BaselinePayload, error) {
	return cases.DecoratePayloads(m.prefix, payloads), nil
}

func (m matrixServer) DecorateLabels(ctx context.Context, labels map[string]string) (map[string]string, error) {
	return cases.DecorateLabels(m.prefix, labels), nil
}

func (m matrixServer) DecoratePayloadMap(ctx context.Context, payloadMap map[string]shared.BaselinePayload) (map[string]shared.BaselinePayload, error) {
	return cases.DecoratePayloadMap(m.prefix, payloadMap), nil
}

func (m matrixServer) FlipMode(ctx context.Context, mode shared.BasicMode) (shared.BasicMode, error) {
	return cases.FlipMode(mode), nil
}

func (m matrixServer) NormalizeUnion(ctx context.Context, value shared.BasicUnion) (shared.BasicUnion, error) {
	return cases.NormalizeUnion(m.prefix, value), nil
}

func (m matrixServer) NormalizeBundle(ctx context.Context, value shared.BasicBundle) (shared.BasicBundle, error) {
	return cases.NormalizeBundle(m.prefix, value), nil
}

func (m matrixServer) ExpandBundle(ctx context.Context, input shared.BasicBundle, payload shared.BasicBundle) (int32, shared.BasicBundle, shared.BasicBundle, error) {
	ret, doubled, payloadOut := cases.ExpandBundle(m.prefix, input, payload)
	return ret, doubled, payloadOut, nil
}

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.matrix", "service manager name")
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

	if err := shared.AddIBasicMatrixServiceService(context.Background(), conn.ServiceManager(), *serviceName, matrixServer{prefix: *prefix}); err != nil {
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
