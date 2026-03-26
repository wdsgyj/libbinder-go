package main

import (
	"fmt"
	"os"
	"sync"
)

var (
	traceOnce    sync.Once
	traceEnabled bool
)

func tracef(format string, args ...any) {
	traceOnce.Do(func() {
		traceEnabled = os.Getenv("LIBBINDER_GO_TRACE") == "1"
	})
	if !traceEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "libbinder-go/cmd: "+format+"\n", args...)
}
