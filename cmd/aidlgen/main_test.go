package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	src := `package demo; interface IFoo { void Ping(); }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"kind": "interface"`) {
		t.Fatalf("stdout = %s, want interface summary", stdout.String())
	}
}

func TestRunAST(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	src := `package demo; parcelable Foo;`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "ast", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "Foo"`) {
		t.Fatalf("stdout = %s, want parcelable AST", stdout.String())
	}
}
