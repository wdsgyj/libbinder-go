package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
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

func TestRunModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	src := `package demo; interface IFoo { void Ping(); }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "model", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"go_name": "IFoo"`) {
		t.Fatalf("stdout = %s, want model go_name", stdout.String())
	}
}

func TestRunGoStdout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	src := `package demo; interface IFoo { void Ping(); }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `type IFoo interface`) {
		t.Fatalf("stdout = %s, want generated Go interface", stdout.String())
	}
}

func TestRunGoOutDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	outDir := filepath.Join(dir, "out")
	src := `package demo; interface IFoo { void Ping(); }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", "-out", outDir, path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}

	generatedPath := filepath.Join(outDir, "demo", "demo_aidl.go")
	data, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", generatedPath, err)
	}
	if !strings.Contains(string(data), `type IFoo interface`) {
		t.Fatalf("generated file = %s, want generated Go interface", data)
	}
}

func TestRunGoWithCustomParcelableMappings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	typesPath := filepath.Join(dir, "aidl.types.json")
	src := `package demo; parcelable Foo; interface IFoo { @nullable Foo Echo(in Foo value); }`
	types := `{
  "version": 1,
  "parcelables": [
    {
      "aidl_name": "demo.Foo",
      "go_package": "example.com/generated/customcodec",
      "go_type": "FooValue",
      "write_func": "WriteFooToParcel",
      "read_func": "ReadFooFromParcel",
      "nullable": true
    }
  ]
}`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile(aidl): %v", err)
	}
	if err := os.WriteFile(typesPath, []byte(types), 0o644); err != nil {
		t.Fatalf("WriteFile(types): %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", "-types", typesPath, path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `generated_customcodec "example.com/generated/customcodec"`) {
		t.Fatalf("stdout = %s, want custom parcelable import", stdout.String())
	}
	if !strings.Contains(stdout.String(), `return generated_customcodec.WriteFooToParcel(p, v)`) {
		t.Fatalf("stdout = %s, want custom parcelable codec wrapper", stdout.String())
	}
}

func TestRunGoLoadsImportedDependencyFiles(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	callbackPath := filepath.Join(pkgDir, "ICallback.aidl")
	servicePath := filepath.Join(pkgDir, "IService.aidl")
	outDir := filepath.Join(dir, "out")

	callbackSrc := `package demo; interface ICallback { void Ping(); }`
	serviceSrc := `package demo; import demo.ICallback; interface IService { ICallback Bind(); }`

	if err := os.WriteFile(callbackPath, []byte(callbackSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(callback): %v", err)
	}
	if err := os.WriteFile(servicePath, []byte(serviceSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(service): %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", "-out", outDir, servicePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(outDir, "demo", "icallback_aidl.go")); err != nil {
		t.Fatalf("Stat(icallback_aidl.go): %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "demo", "iservice_aidl.go")); err != nil {
		t.Fatalf("Stat(iservice_aidl.go): %v", err)
	}
}

func TestRunGoAOSPBinderCorpus(t *testing.T) {
	corpus := aospBinderCorpusFiles(t)
	if len(corpus) == 0 {
		t.Skip("AOSP binder corpus not available")
	}

	outDir := t.TempDir()
	args := append([]string{"-format", "go", "-out", outDir}, corpus...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}

	found := 0
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "_aidl.go") {
			found++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk(%s): %v", outDir, err)
	}
	if found == 0 {
		t.Fatalf("generated file count = %d, want > 0", found)
	}
}

func aospBinderCorpusFiles(t *testing.T) []string {
	t.Helper()

	dirs := []string{
		filepath.Join("aosp-src", "frameworks", "native", "libs", "binder", "aidl"),
		filepath.Join("aosp-src", "frameworks", "native", "libs", "binder", "tests"),
		filepath.Join("aosp-src", "frameworks", "native", "libs", "binder", "tests", "parcel_fuzzer", "parcelables"),
	}

	var files []string
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("Stat(%s): %v", dir, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".aidl") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Walk(%s): %v", dir, err)
		}
	}
	seen := map[string]struct{}{}
	unique := files[:0]
	for _, path := range files {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	files = unique
	sort.Strings(files)
	return files
}
