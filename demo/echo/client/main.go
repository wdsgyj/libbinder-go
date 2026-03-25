package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/wdsgyj/libbinder-go"
	api "github.com/wdsgyj/libbinder-go/binder"
)

const (
	serviceName     = "libbinder.go.demo.echo"
	serviceDesc     = "libbinder.go.demo.IEcho"
	echoTransaction = uint32(1)
)

func main() {
	driverPath := flag.String("driver", "", "binder driver path; default uses library default")
	timeout := flag.Duration("timeout", 5*time.Second, "service lookup timeout")
	flag.Parse()

	message := "ping"
	if flag.NArg() > 0 {
		message = flag.Arg(0)
	}

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fatalf("close binder connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	service, err := conn.ServiceManager().WaitService(ctx, serviceName)
	if err != nil {
		fatalf("waitService %q: %v", serviceName, err)
	}
	defer func() {
		if err := service.Close(); err != nil {
			fatalf("service.Close: %v", err)
		}
	}()

	desc, err := service.Descriptor(context.Background())
	if err != nil {
		fatalf("service.Descriptor: %v", err)
	}
	fmt.Printf("resolved descriptor: %s\n", desc)
	if desc != serviceDesc {
		fatalf("unexpected descriptor %q, want %q", desc, serviceDesc)
	}

	req := api.NewParcel()
	if err := req.WriteString(message); err != nil {
		fatalf("req.WriteString: %v", err)
	}

	reply, err := service.Transact(context.Background(), echoTransaction, req, api.FlagNone)
	if err != nil {
		fatalf("service.Transact: %v", err)
	}
	if reply == nil {
		fatalf("service.Transact returned nil reply")
	}

	got, err := reply.ReadString()
	if err != nil {
		fatalf("reply.ReadString: %v", err)
	}

	fmt.Printf("request: %s\n", message)
	fmt.Printf("reply:   %s\n", got)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
