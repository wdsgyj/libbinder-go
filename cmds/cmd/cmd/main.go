package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	cmdtool "github.com/wdsgyj/libbinder-go/cmds/cmd"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(cmdtool.ProcessExitCode(cmdtool.Main(context.Background(), os.Args[1:], os.Stdout, os.Stderr)))
}
