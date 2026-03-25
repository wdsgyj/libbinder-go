package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ir"
	"github.com/wdsgyj/libbinder-go/internal/aidl/parser"
	"github.com/wdsgyj/libbinder-go/internal/aidl/resolve"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("aidlgen", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "summary", "output format: summary or ast")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: aidlgen [-format summary|ast] <file.aidl>")
		return 2
	}

	path := fs.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read %s: %v\n", path, err)
		return 1
	}

	file, err := parser.Parse(path, string(data))
	if err != nil {
		fmt.Fprintf(stderr, "parse %s: %v\n", path, err)
		return 1
	}

	if diags := resolve.ValidateFile(file); len(diags) != 0 {
		enc := json.NewEncoder(stderr)
		enc.SetIndent("", "  ")
		_ = enc.Encode(diags)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	switch *format {
	case "ast":
		if err := enc.Encode(file); err != nil {
			fmt.Fprintf(stderr, "encode ast: %v\n", err)
			return 1
		}
	case "summary":
		if err := enc.Encode(ir.Lower(file)); err != nil {
			fmt.Fprintf(stderr, "encode summary: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unknown format %q\n", *format)
		return 2
	}

	return 0
}
