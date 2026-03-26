package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var signalIgnore = signal.Ignore
var processExit = os.Exit

func main() {
	signalIgnore(syscall.SIGPIPE)
	processExit(ProcessExitCode(Main(context.Background(), os.Args[1:], os.Stdout, os.Stderr)))
}
