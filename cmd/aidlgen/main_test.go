package main

import (
	"bytes"
	"os"
	"os/exec"
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

func TestRunGoLoadsNestedImportedDependencyFile(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	optionsPath := filepath.Join(pkgDir, "Options.aidl")
	servicePath := filepath.Join(pkgDir, "IService.aidl")
	outDir := filepath.Join(dir, "out")

	optionsSrc := `package demo; parcelable Options.SceneInfo;`
	serviceSrc := `package demo; import demo.Options.SceneInfo; interface IService { void Ping(in SceneInfo info); }`

	if err := os.WriteFile(optionsPath, []byte(optionsSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(options): %v", err)
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

	if _, err := os.Stat(filepath.Join(outDir, "demo", "options_aidl.go")); err != nil {
		t.Fatalf("Stat(options_aidl.go): %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "demo", "iservice_aidl.go")); err != nil {
		t.Fatalf("Stat(iservice_aidl.go): %v", err)
	}
}

func TestRunGoRootsOnlySkipsImportedOutputs(t *testing.T) {
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
	code := run([]string{"-format", "go", "-roots-only", "-out", outDir, servicePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(outDir, "demo", "iservice_aidl.go")); err != nil {
		t.Fatalf("Stat(iservice_aidl.go): %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "demo", "icallback_aidl.go")); !os.IsNotExist(err) {
		t.Fatalf("Stat(icallback_aidl.go) err = %v, want not exist", err)
	}
}

func TestRunGoLoadsSamePackageSiblingDependencyFiles(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	parcelPath := filepath.Join(pkgDir, "Payload.aidl")
	servicePath := filepath.Join(pkgDir, "IService.aidl")
	outDir := filepath.Join(dir, "out")

	parcelSrc := `package demo; parcelable Payload { int id; }`
	serviceSrc := `package demo; interface IService { Payload Echo(in Payload value); }`

	if err := os.WriteFile(parcelPath, []byte(parcelSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(parcel): %v", err)
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

	if _, err := os.Stat(filepath.Join(outDir, "demo", "payload_aidl.go")); err != nil {
		t.Fatalf("Stat(payload_aidl.go): %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "demo", "iservice_aidl.go")); err != nil {
		t.Fatalf("Stat(iservice_aidl.go): %v", err)
	}
}

func TestRunGoFallsBackToOpaqueParcelableForForwardDeclaration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.aidl")
	src := `package demo; parcelable Payload; interface IFoo { Payload Echo(in Payload value); }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "type Payload struct") || !strings.Contains(out, "RawData []byte") {
		t.Fatalf("stdout = %s, want opaque parcelable fallback", out)
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

	writeGeneratedModuleGoMod(t, outDir)
	runGoTestGeneratedModule(t, outDir)
}

func TestRunGoNoPackageDirectoryCompiles(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "binder", "tests")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	certPath := filepath.Join(pkgDir, "ParcelableCertificateData.aidl")
	clientPath := filepath.Join(pkgDir, "BinderRpcTestClientInfo.aidl")
	outDir := filepath.Join(dir, "out")

	certSrc := `parcelable ParcelableCertificateData { byte[] data; }`
	clientSrc := `import ParcelableCertificateData; parcelable BinderRpcTestClientInfo { ParcelableCertificateData[] certs; }`

	if err := os.WriteFile(certPath, []byte(certSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(cert): %v", err)
	}
	if err := os.WriteFile(clientPath, []byte(clientSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(client): %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-format", "go", "-go-import-root", "example.com/generated", "-out", outDir, certPath, clientPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run code = %d, stderr = %s", code, stderr.String())
	}

	writeGeneratedModuleGoMod(t, outDir)
	runGoTestGeneratedModule(t, outDir)
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

func writeGeneratedModuleGoMod(t *testing.T, outDir string) {
	t.Helper()

	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot(t) + `
`
	if err := os.WriteFile(filepath.Join(outDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}
}

func runGoTestGeneratedModule(t *testing.T, outDir string) {
	t.Helper()

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = outDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... in %s: %v\n%s", outDir, err, output)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
