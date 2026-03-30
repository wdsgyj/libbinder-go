package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"syscall"
	"time"

	libbinder "github.com/wdsgyj/libbinder-go"
	"github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/cases"
)

func main() {
	driverPath := flag.String("driver", "/dev/binder", "binder driver path")
	serviceName := flag.String("service", "libbinder.go.aidltest.baseline", "service manager name")
	expectPrefix := flag.String("expect-prefix", "java", "expected service output prefix")
	mode := flag.String("mode", "discovery", "verification mode: discovery|death")
	killPID := flag.Int("kill-pid", 0, "remote pid to kill after death registration")
	killDelay := flag.Duration("kill-delay", 500*time.Millisecond, "delay before killing the watched service")
	timeout := flag.Duration("timeout", 10*time.Second, "overall call timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := libbinder.Open(libbinder.Config{DriverPath: *driverPath})
	if err != nil {
		fatalf("open binder: %v", err)
	}
	defer conn.Close()

	switch *mode {
	case "discovery":
		if err := cases.VerifyLifecycleDiscovery(ctx, conn.ServiceManager(), *serviceName, *expectPrefix); err != nil {
			fatalf("verify lifecycle discovery: %v", err)
		}
	case "death":
		service, err := conn.ServiceManager().WaitService(ctx, *serviceName)
		if err != nil {
			fatalf("wait service %q: %v", *serviceName, err)
		}
		if *killPID <= 0 {
			fatalf("death mode requires -kill-pid")
		}
		if err := cases.WaitForBinderDeathAfter(ctx, service, *killDelay, func() error {
			return syscall.Kill(*killPID, syscall.SIGKILL)
		}); err != nil {
			fatalf("verify death: %v", err)
		}
	default:
		fatalf("unsupported mode %q", *mode)
	}

	fmt.Fprintln(os.Stdout, "OK")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
