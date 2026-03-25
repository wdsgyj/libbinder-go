package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	allowIsolated := flag.Bool("allow-isolated", false, "allow isolated callers during addService")
	flag.Parse()

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder connection: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fatalf("close binder connection: %v", err)
		}
	}()

	handler := api.StaticHandler{
		DescriptorName: serviceDesc,
		Handle: func(ctx context.Context, code uint32, data *api.Parcel) (*api.Parcel, error) {
			switch code {
			case echoTransaction:
				msg, err := data.ReadString()
				if err != nil {
					return nil, err
				}

				reply := api.NewParcel()
				if err := reply.WriteString("echo:" + msg); err != nil {
					return nil, err
				}
				return reply, nil
			default:
				return nil, fmt.Errorf("unexpected transaction code %d", code)
			}
		},
	}

	if err := conn.ServiceManager().AddService(
		context.Background(),
		serviceName,
		handler,
		api.WithAllowIsolated(*allowIsolated),
	); err != nil {
		var remoteErr *api.RemoteException
		if errors.As(err, &remoteErr) && remoteErr.Code == api.ExceptionSecurity {
			fatalf("addService denied by system policy: %v", err)
		}
		fatalf("addService %q: %v", serviceName, err)
	}

	fmt.Printf("echo server registered as %q\n", serviceName)
	fmt.Printf("descriptor: %s\n", serviceDesc)
	fmt.Printf("transaction code: %d\n", echoTransaction)
	fmt.Println("waiting for Ctrl+C")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
