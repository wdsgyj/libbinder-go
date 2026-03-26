package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(Main(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
