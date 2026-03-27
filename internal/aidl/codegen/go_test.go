package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	"github.com/wdsgyj/libbinder-go/internal/aidl/gomodel"
	"github.com/wdsgyj/libbinder-go/internal/aidl/parser"
)

func TestRenderGoGoldenCorpus(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("golden corpus uses host-side testdata files")
	}

	cases, err := filepath.Glob(filepath.Join("testdata", "golden", "*", "*.aidl"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no golden AIDL inputs found")
	}

	for _, path := range cases {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".aidl"), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", path, err)
			}
			file, err := parser.Parse(path, string(data))
			if err != nil {
				t.Fatalf("Parse(%s): %v", path, err)
			}
			model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: filepath.Base(path)})
			if len(diags) != 0 {
				t.Fatalf("Lower diagnostics = %#v", diags)
			}
			outputs, err := RenderGo(model, GoOptions{})
			if err != nil {
				t.Fatalf("RenderGo: %v", err)
			}
			if len(outputs) != 1 {
				t.Fatalf("len(outputs) = %d, want 1", len(outputs))
			}

			wantPath := filepath.Join(filepath.Dir(path), "expected.go")
			want, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", wantPath, err)
			}
			if string(outputs[0].Content) != string(want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, outputs[0].Content, want)
			}
		})
	}
}

func TestRenderGoCompilesAndRuns(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	src := `
package demo;

enum Kind { ONE = 1, TWO = 2 }

union Result {
  int code;
  @nullable String text;
}

parcelable Payload {
  int count;
  @nullable String note;
  Kind kind;
  Result result;
  int[] ids;
  int[2] pair;
}

interface IEcho {
  @nullable String Echo(in String msg, out int code, inout Payload payload);
}
`

	file, err := parser.Parse("IEcho.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: "IEcho.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("len(outputs) = %d, want 1", len(outputs))
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)

	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"testing"

	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeBinder struct {
	handler binder.Handler
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if code == binder.GetInterfaceVersionTransaction {
		provider, ok := b.handler.(binder.InterfaceVersionProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteInt32(provider.InterfaceVersion()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if code == binder.GetInterfaceHashTransaction {
		provider, ok := b.handler.(binder.InterfaceHashProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteString(provider.InterfaceHash()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeBinder) Close() error { return nil }

type echoImpl struct{}

func (echoImpl) Echo(ctx context.Context, msg string, payload Payload) (*string, int32, Payload, error) {
	reply := "echo:" + msg
	payload.Count++
	payload.Kind = KindTwo
	payload.Result = Result{
		Tag:  ResultTagText,
		Text: &reply,
	}
	payload.Ids = append(payload.Ids, 9)
	payload.Pair = [2]int32{7, 8}
	return &reply, 200, payload, nil
}

func TestGeneratedClientServerRoundTrip(t *testing.T) {
	note := "hello"
	client := NewIEchoClient(fakeBinder{handler: NewIEchoHandler(echoImpl{})})

	got, code, payloadOut, err := client.Echo(context.Background(), "ping", Payload{
		Count: 1,
		Note:  &note,
		Kind:  KindOne,
		Result: Result{
			Tag:  ResultTagCode,
			Code: 7,
		},
		Ids:  []int32{1, 2},
		Pair: [2]int32{3, 4},
	})
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if got == nil || *got != "echo:ping" {
		t.Fatalf("got = %#v, want echo:ping", got)
	}
	if code != 200 {
		t.Fatalf("code = %d, want 200", code)
	}
	if payloadOut.Count != 2 {
		t.Fatalf("payloadOut.Count = %d, want 2", payloadOut.Count)
	}
	if payloadOut.Kind != KindTwo {
		t.Fatalf("payloadOut.Kind = %v, want KindTwo", payloadOut.Kind)
	}
	if payloadOut.Result.Tag != ResultTagText || payloadOut.Result.Text == nil || *payloadOut.Result.Text != "echo:ping" {
		t.Fatalf("payloadOut.Result = %#v, want text echo:ping", payloadOut.Result)
	}
	if len(payloadOut.Ids) != 3 || payloadOut.Ids[2] != 9 {
		t.Fatalf("payloadOut.Ids = %#v, want [1 2 9]", payloadOut.Ids)
	}
	if payloadOut.Pair != [2]int32{7, 8} {
		t.Fatalf("payloadOut.Pair = %#v, want [7 8]", payloadOut.Pair)
	}
}
`

	testPath := filepath.Join(tmp, "demo", "generated_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSanitizesKeywordArgumentNames(t *testing.T) {
	src := `
package demo;

interface IService {
  void Ping(in int map, in String type, in IBinder binder, in int err);
}
`

	file, err := parser.Parse("keywords.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: "keywords.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("len(outputs) = %d, want 1", len(outputs))
	}
	code := string(outputs[0].Content)
	if !strings.Contains(code, "Ping(ctx context.Context, map_ int32, type_ string, binder_ binder.Binder, err_ int32)") {
		t.Fatalf("generated signature missing sanitized names:\n%s", code)
	}
}

func TestOutputPathUsesPackageLayout(t *testing.T) {
	model := &gomodel.File{
		AIDLPackage: "android.test.demo",
		SourcePath:  "IEcho.aidl",
	}
	if got := outputPath(model); got != filepath.Join("android", "test", "demo", "iecho_aidl.go") {
		t.Fatalf("outputPath = %q, want android/test/demo/iecho_aidl.go", got)
	}
}

func TestOutputPathEscapesInternalSegment(t *testing.T) {
	model := &gomodel.File{
		AIDLPackage: "com.android.internal.os",
		SourcePath:  "IResultReceiver.aidl",
	}
	if got := outputPath(model); got != filepath.Join("com", "android", "internal_", "os", "iresultreceiver_aidl.go") {
		t.Fatalf("outputPath = %q, want com/android/internal_/os/iresultreceiver_aidl.go", got)
	}
}

func TestRenderGoSupportsInterfaceAndIBinder(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	src := `
package demo;

interface ICallback {
  String Echo(in String msg);
}

interface IService {
  ICallback Bind(in ICallback cb, in IBinder raw, out IBinder echoed);
}
`

	file, err := parser.Parse("service.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: "service.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"fmt"
	"testing"

	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeRegistrar struct {
	next    uint32
	binders map[uint32]binder.Binder
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{
		next:    100,
		binders: map[uint32]binder.Binder{},
	}
}

func (r *fakeRegistrar) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	r.next++
	b := fakeRegisteredBinder{handle: r.next, handler: handler, registrar: r}
	r.binders[b.handle] = b
	return b, nil
}

func (r *fakeRegistrar) resolve(handle uint32) binder.Binder {
	return r.binders[handle]
}

type fakeRegisteredBinder struct {
	handle    uint32
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeRegisteredBinder) AsBinder() binder.Binder { return b }
func (b fakeRegisteredBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}
func (b fakeRegisteredBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}
func (b fakeRegisteredBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}
func (b fakeRegisteredBinder) Close() error { return nil }
func (b fakeRegisteredBinder) WriteBinderToParcel(p *binder.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type fakeEndpoint struct {
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeEndpoint) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}
func (b fakeEndpoint) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}
func (b fakeEndpoint) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}
func (b fakeEndpoint) Close() error { return nil }
func (b fakeEndpoint) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	return b.registrar.RegisterLocalHandler(handler)
}

type callbackImpl struct{}

func (callbackImpl) Echo(ctx context.Context, msg string) (string, error) {
	return "cb:" + msg, nil
}

type serviceImpl struct{}

func (serviceImpl) Bind(ctx context.Context, cb ICallback, raw binder.Binder) (ICallback, binder.Binder, error) {
	if cb == nil {
		return nil, nil, fmt.Errorf("nil callback")
	}
	got, err := cb.Echo(ctx, "srv")
	if err != nil {
		return nil, nil, err
	}
	if got != "cb:srv" {
		return nil, nil, fmt.Errorf("unexpected callback result %q", got)
	}
	return cb, raw, nil
}

func TestGeneratedInterfaceBinderRoundTrip(t *testing.T) {
	reg := newFakeRegistrar()
	raw := fakeRegisteredBinder{
		handle:    7,
		handler:   NewICallbackHandler(callbackImpl{}),
		registrar: reg,
	}
	reg.binders[7] = raw

	service := fakeEndpoint{
		handler:   NewIServiceHandlerWithRegistrar(reg, serviceImpl{}),
		registrar: reg,
	}
	client := NewIServiceClient(service)

	cbOut, echoed, err := client.Bind(context.Background(), callbackImpl{}, raw)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if echoed == nil {
		t.Fatal("echoed = nil, want raw binder")
	}
	got, err := cbOut.Echo(context.Background(), "client")
	if err != nil {
		t.Fatalf("cbOut.Echo: %v", err)
	}
	if got != "cb:client" {
		t.Fatalf("cbOut.Echo = %q, want cb:client", got)
	}
	if desc, err := echoed.Descriptor(context.Background()); err != nil || desc != ICallbackDescriptor {
		t.Fatalf("echoed.Descriptor = (%q, %v), want (%q, nil)", desc, err, ICallbackDescriptor)
	}
}
`

	testPath := filepath.Join(tmp, "demo", "generated_interface_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_interface_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSupportsBinderObjectsAndFDInParcelableAndUnion(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	src := `
package demo;

interface ICallback {
  String Echo(in String msg);
}

union Result {
  ICallback cb;
  IBinder raw;
  FileDescriptor fd;
  @nullable ParcelFileDescriptor pfd;
}

parcelable Payload {
  ICallback cb;
  IBinder raw;
  FileDescriptor fd;
  @nullable ParcelFileDescriptor pfd;
  Result nested;
}

interface IService {
  Result Exchange(in Payload payload, out Payload echoed);
}
`

	file, err := parser.Parse("service_fd.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: "service_fd.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"fmt"
	"os"
	"testing"

	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeRegistrar struct {
	next    uint32
	binders map[uint32]binder.Binder
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{
		next:    100,
		binders: map[uint32]binder.Binder{},
	}
}

func (r *fakeRegistrar) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	r.next++
	b := fakeRegisteredBinder{handle: r.next, handler: handler, registrar: r}
	r.binders[b.handle] = b
	return b, nil
}

func (r *fakeRegistrar) resolve(handle uint32) binder.Binder {
	return r.binders[handle]
}

type fakeRegisteredBinder struct {
	handle    uint32
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeRegisteredBinder) AsBinder() binder.Binder { return b }
func (b fakeRegisteredBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}
func (b fakeRegisteredBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}
func (b fakeRegisteredBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}
func (b fakeRegisteredBinder) Close() error { return nil }
func (b fakeRegisteredBinder) WriteBinderToParcel(p *binder.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type fakeEndpoint struct {
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeEndpoint) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}
func (b fakeEndpoint) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}
func (b fakeEndpoint) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}
func (b fakeEndpoint) Close() error { return nil }
func (b fakeEndpoint) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	return b.registrar.RegisterLocalHandler(handler)
}

type callbackImpl struct{}

func (callbackImpl) Echo(ctx context.Context, msg string) (string, error) {
	return "cb:" + msg, nil
}

type serviceImpl struct{}

func (serviceImpl) Exchange(ctx context.Context, payload Payload) (Result, Payload, error) {
	if payload.Cb == nil {
		return Result{}, Payload{}, fmt.Errorf("nil payload cb")
	}
	if got, err := payload.Cb.Echo(ctx, "srv"); err != nil || got != "cb:srv" {
		return Result{}, Payload{}, fmt.Errorf("payload cb = (%q, %v)", got, err)
	}
	if payload.Raw == nil {
		return Result{}, Payload{}, fmt.Errorf("nil raw binder")
	}
	if desc, err := payload.Raw.Descriptor(ctx); err != nil || desc != ICallbackDescriptor {
		return Result{}, Payload{}, fmt.Errorf("raw.Descriptor = (%q, %v)", desc, err)
	}
	if payload.Fd.FD() < 0 {
		return Result{}, Payload{}, fmt.Errorf("invalid fd %d", payload.Fd.FD())
	}
	if payload.Pfd == nil || payload.Pfd.FD() < 0 {
		return Result{}, Payload{}, fmt.Errorf("invalid pfd %#v", payload.Pfd)
	}
	if payload.Nested.Tag != ResultTagCb || payload.Nested.Cb == nil {
		return Result{}, Payload{}, fmt.Errorf("unexpected nested union %#v", payload.Nested)
	}
	if got, err := payload.Nested.Cb.Echo(ctx, "nested"); err != nil || got != "cb:nested" {
		return Result{}, Payload{}, fmt.Errorf("nested cb = (%q, %v)", got, err)
	}
	payload.Nested = Result{
		Tag: ResultTagPfd,
		Pfd: payload.Pfd,
	}
	return Result{
		Tag: ResultTagFd,
		Fd:  payload.Fd,
	}, payload, nil
}

func TestGeneratedParcelableUnionBinderAndFDRoundTrip(t *testing.T) {
	reg := newFakeRegistrar()
	raw := fakeRegisteredBinder{
		handle:    7,
		handler:   NewICallbackHandler(callbackImpl{}),
		registrar: reg,
	}
	reg.binders[7] = raw

	service := fakeEndpoint{
		handler:   NewIServiceHandlerWithRegistrar(reg, serviceImpl{}),
		registrar: reg,
	}
	client := NewIServiceClient(service)

	fdReader, fdWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(fd): %v", err)
	}
	defer fdReader.Close()
	defer fdWriter.Close()

	pfdReader, pfdWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(pfd): %v", err)
	}
	defer pfdReader.Close()
	defer pfdWriter.Close()

	pfdValue := binder.NewParcelFileDescriptor(int(pfdReader.Fd()))
	defer pfdValue.Close()

	result, echoed, err := client.Exchange(context.Background(), Payload{
		Cb:  callbackImpl{},
		Raw: raw,
		Fd:  binder.NewFileDescriptor(int(fdReader.Fd())),
		Pfd: &pfdValue,
		Nested: Result{
			Tag: ResultTagCb,
			Cb:  callbackImpl{},
		},
	})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}

	if result.Tag != ResultTagFd || result.Fd.FD() != int(fdReader.Fd()) {
		t.Fatalf("result = %#v, want fd tag with %d", result, fdReader.Fd())
	}
	if echoed.Cb == nil {
		t.Fatal("echoed.Cb = nil, want callback")
	}
	if got, err := echoed.Cb.Echo(context.Background(), "client"); err != nil || got != "cb:client" {
		t.Fatalf("echoed.Cb.Echo = (%q, %v), want (cb:client, nil)", got, err)
	}
	if echoed.Raw == nil {
		t.Fatal("echoed.Raw = nil, want raw binder")
	}
	if desc, err := echoed.Raw.Descriptor(context.Background()); err != nil || desc != ICallbackDescriptor {
		t.Fatalf("echoed.Raw.Descriptor = (%q, %v), want (%q, nil)", desc, err, ICallbackDescriptor)
	}
	if echoed.Fd.FD() != int(fdReader.Fd()) {
		t.Fatalf("echoed.Fd = %d, want %d", echoed.Fd.FD(), fdReader.Fd())
	}
	if echoed.Pfd == nil || echoed.Pfd.FD() != int(pfdReader.Fd()) {
		t.Fatalf("echoed.Pfd = %#v, want fd %d", echoed.Pfd, pfdReader.Fd())
	}
	if echoed.Nested.Tag != ResultTagPfd || echoed.Nested.Pfd == nil || echoed.Nested.Pfd.FD() != int(pfdReader.Fd()) {
		t.Fatalf("echoed.Nested = %#v, want pfd tag with %d", echoed.Nested, pfdReader.Fd())
	}
}
`

	testPath := filepath.Join(tmp, "demo", "generated_parcelable_union_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_parcelable_union_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSupportsCustomParcelableSidecar(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	src := `
package demo;

parcelable Foo;

interface IService {
  @nullable Foo Echo(in Foo value);
}
`

	file, err := parser.Parse("custom.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{
		SourcePath: "custom.aidl",
		CustomParcelables: map[string]gomodel.CustomParcelableConfig{
			"demo.Foo": {
				AIDLName:  "demo.Foo",
				GoPackage: "example.com/generated/customcodec",
				GoType:    "FooValue",
				WriteFunc: "WriteFooToParcel",
				ReadFunc:  "ReadFooFromParcel",
				Nullable:  true,
			},
		},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	codecSrc := `package customcodec

import binder "github.com/wdsgyj/libbinder-go/binder"

type FooValue struct {
	Count int32
}

func WriteFooToParcel(p *binder.Parcel, v FooValue) error {
	return p.WriteInt32(v.Count)
}

func ReadFooFromParcel(p *binder.Parcel) (FooValue, error) {
	count, err := p.ReadInt32()
	if err != nil {
		return FooValue{}, err
	}
	return FooValue{Count: count}, nil
}
`
	codecPath := filepath.Join(tmp, "customcodec", "codec.go")
	if err := os.MkdirAll(filepath.Dir(codecPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(customcodec): %v", err)
	}
	if err := os.WriteFile(codecPath, []byte(codecSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(codec): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"testing"

	customcodec "example.com/generated/customcodec"
	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeBinder struct {
	handler binder.Handler
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if code == binder.GetInterfaceVersionTransaction {
		provider, ok := b.handler.(binder.InterfaceVersionProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteInt32(provider.InterfaceVersion()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if code == binder.GetInterfaceHashTransaction {
		provider, ok := b.handler.(binder.InterfaceHashProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteString(provider.InterfaceHash()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeBinder) Close() error { return nil }

type serviceImpl struct{}

func (serviceImpl) Echo(ctx context.Context, value customcodec.FooValue) (*customcodec.FooValue, error) {
	value.Count++
	return &value, nil
}

func TestGeneratedCustomParcelableRoundTrip(t *testing.T) {
	client := NewIServiceClient(fakeBinder{handler: NewIServiceHandler(serviceImpl{})})

	got, err := client.Echo(context.Background(), customcodec.FooValue{Count: 41})
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if got == nil || got.Count != 42 {
		t.Fatalf("got = %#v, want Count=42", got)
	}
}
`
	testPath := filepath.Join(tmp, "demo", "generated_custom_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_custom_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSupportsExternalCustomParcelableDependencyAndOnewayNameCollisions(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	depSrc := `
package demo;

parcelable Foo;
`
	ifaceSrc := `
package demo;

interface IService {
  oneway void Send(in Foo value, in String data, in int code);
  @nullable Foo Echo(in Foo value, in String data, in int code);
}
`

	depFile, err := parser.Parse("Foo.aidl", depSrc)
	if err != nil {
		t.Fatalf("Parse(Foo.aidl): %v", err)
	}
	ifaceFile, err := parser.Parse("IService.aidl", ifaceSrc)
	if err != nil {
		t.Fatalf("Parse(IService.aidl): %v", err)
	}

	customParcelables := map[string]gomodel.CustomParcelableConfig{
		"demo.Foo": {
			AIDLName:  "demo.Foo",
			GoPackage: "example.com/generated/customcodec",
			GoType:    "FooValue",
			WriteFunc: "WriteFooToParcel",
			ReadFunc:  "ReadFooFromParcel",
			Nullable:  true,
		},
	}
	model, diags := gomodel.Lower(ifaceFile, gomodel.LowerOptions{
		SourcePath:        "IService.aidl",
		CustomParcelables: customParcelables,
		DependencyFiles:   []*ast.File{depFile},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{CustomParcelables: customParcelables})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("len(outputs) = %d, want 1", len(outputs))
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	codecSrc := `package customcodec

import binder "github.com/wdsgyj/libbinder-go/binder"

type FooValue struct {
	Count int32
}

func WriteFooToParcel(p *binder.Parcel, v FooValue) error {
	return p.WriteInt32(v.Count)
}

func ReadFooFromParcel(p *binder.Parcel) (FooValue, error) {
	count, err := p.ReadInt32()
	if err != nil {
		return FooValue{}, err
	}
	return FooValue{Count: count}, nil
}
`
	codecPath := filepath.Join(tmp, "customcodec", "codec.go")
	if err := os.MkdirAll(filepath.Dir(codecPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(customcodec): %v", err)
	}
	if err := os.WriteFile(codecPath, []byte(codecSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(codec): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"testing"

	customcodec "example.com/generated/customcodec"
	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeBinder struct {
	handler binder.Handler
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeBinder) Close() error { return nil }

type serviceImpl struct {
	lastData string
	lastCode int32
}

func (s *serviceImpl) Send(ctx context.Context, value customcodec.FooValue, data string, code int32) error {
	s.lastData = data
	s.lastCode = code
	return nil
}

func (s *serviceImpl) Echo(ctx context.Context, value customcodec.FooValue, data string, code int32) (*customcodec.FooValue, error) {
	value.Count += int32(len(data)) + code
	return &value, nil
}

func TestGeneratedExternalCustomParcelableRoundTrip(t *testing.T) {
	impl := &serviceImpl{}
	client := NewIServiceClient(fakeBinder{handler: NewIServiceHandler(impl)})

	if err := client.Send(context.Background(), customcodec.FooValue{Count: 1}, "abc", 7); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if impl.lastData != "abc" || impl.lastCode != 7 {
		t.Fatalf("impl state = %#v, want data=abc code=7", impl)
	}

	got, err := client.Echo(context.Background(), customcodec.FooValue{Count: 10}, "abcd", 2)
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if got == nil || got.Count != 16 {
		t.Fatalf("got = %#v, want Count=16", got)
	}
}
`
	testPath := filepath.Join(tmp, "demo", "generated_external_custom_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_external_custom_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSupportsStableInterfaceMetadataAndFallback(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	src := `
package demo;

@VintfStability
interface IEcho {
  String Echo(in String msg);
}
`

	file, err := parser.Parse("stable.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{
		SourcePath: "stable.aidl",
		StableInterfaces: map[string]gomodel.StableInterfaceConfig{
			"demo.IEcho": {
				AIDLName: "demo.IEcho",
				Version:  3,
				Hash:     "abcdef",
			},
		},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	outPath := filepath.Join(tmp, outputs[0].Path)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(outPath, outputs[0].Content, 0o644); err != nil {
		t.Fatalf("WriteFile(generated): %v", err)
	}

	testSrc := `package demo

import (
	"context"
	"testing"

	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeBinder struct {
	handler binder.Handler
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if code == binder.GetInterfaceVersionTransaction {
		provider, ok := b.handler.(binder.InterfaceVersionProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteInt32(provider.InterfaceVersion()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if code == binder.GetInterfaceHashTransaction {
		provider, ok := b.handler.(binder.InterfaceHashProvider)
		if !ok {
			return nil, binder.ErrUnknownTransaction
		}
		reply := binder.NewParcel()
		if err := reply.WriteString(provider.InterfaceHash()); err != nil {
			return nil, err
		}
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
		return reply, nil
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeBinder) Close() error { return nil }

type legacyBinder struct {
	handler binder.Handler
}

func (b legacyBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b legacyBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if code == binder.GetInterfaceVersionTransaction || code == binder.GetInterfaceHashTransaction {
		return nil, binder.ErrUnknownTransaction
	}
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b legacyBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b legacyBinder) Close() error { return nil }

type echoImpl struct{}

func (echoImpl) Echo(ctx context.Context, msg string) (string, error) {
	return "echo:" + msg, nil
}

func TestGeneratedStableInterfaceHelpers(t *testing.T) {
	client := NewIEchoClient(fakeBinder{handler: NewIEchoHandler(echoImpl{})})
	version, err := GetIEchoInterfaceVersion(context.Background(), client)
	if err != nil {
		t.Fatalf("GetIEchoInterfaceVersion: %v", err)
	}
	if version != 3 {
		t.Fatalf("version = %d, want 3", version)
	}
	hash, err := GetIEchoInterfaceHash(context.Background(), client)
	if err != nil {
		t.Fatalf("GetIEchoInterfaceHash: %v", err)
	}
	if hash != "abcdef" {
		t.Fatalf("hash = %q, want abcdef", hash)
	}
}

func TestGeneratedStableInterfaceUnknownTransactionFallback(t *testing.T) {
	client := NewIEchoClient(legacyBinder{handler: NewIEchoHandler(echoImpl{})})
	version, err := GetIEchoInterfaceVersion(context.Background(), client)
	if err != nil {
		t.Fatalf("GetIEchoInterfaceVersion: %v", err)
	}
	if version != -1 {
		t.Fatalf("version = %d, want -1", version)
	}
	hash, err := GetIEchoInterfaceHash(context.Background(), client)
	if err != nil {
		t.Fatalf("GetIEchoInterfaceHash: %v", err)
	}
	if hash != "-1" {
		t.Fatalf("hash = %q, want -1", hash)
	}
	got, err := client.Echo(context.Background(), "ping")
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if got != "echo:ping" {
		t.Fatalf("Echo = %q, want echo:ping", got)
	}
}
`
	testPath := filepath.Join(tmp, "demo", "generated_stable_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_stable_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoGeneratesParcelableDefaultConstructor(t *testing.T) {
	src := `
package demo;

parcelable Holder {
  enum Kind {
    ONE,
    TWO,
  }
  Kind kind = Kind.TWO;
}
`

	file, err := parser.Parse("holder.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	model, diags := gomodel.Lower(file, gomodel.LowerOptions{SourcePath: "holder.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v", diags)
	}

	outputs, err := RenderGo(model, GoOptions{})
	if err != nil {
		t.Fatalf("RenderGo: %v", err)
	}
	content := string(outputs[0].Content)
	if !strings.Contains(content, "func NewHolder() Holder") {
		t.Fatalf("generated code missing default constructor:\n%s", content)
	}
	if !strings.Contains(content, "v.Kind = HolderKindTwo") {
		t.Fatalf("generated code missing default value assignment:\n%s", content)
	}
}

func TestRenderGoSupportsCrossPackageGeneratedDependencies(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	alphaSrc := `
package alpha;

parcelable Foo {
  int count;
}
`
	gammaSrc := `
package gamma;

import alpha.Foo;

interface ICallback {
  Foo Done(in Foo value);
}
`
	betaSrc := `
package beta;

import alpha.Foo;
import gamma.ICallback;

interface IBar {
  Foo Echo(in Foo value, in ICallback cb);
}
`

	alphaFile, err := parser.Parse("Foo.aidl", alphaSrc)
	if err != nil {
		t.Fatalf("Parse(Foo.aidl): %v", err)
	}
	gammaFile, err := parser.Parse("ICallback.aidl", gammaSrc)
	if err != nil {
		t.Fatalf("Parse(ICallback.aidl): %v", err)
	}
	betaFile, err := parser.Parse("IBar.aidl", betaSrc)
	if err != nil {
		t.Fatalf("Parse(IBar.aidl): %v", err)
	}

	alphaModel, diags := gomodel.Lower(alphaFile, gomodel.LowerOptions{SourcePath: "Foo.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower alpha diagnostics = %#v", diags)
	}
	gammaModel, diags := gomodel.Lower(gammaFile, gomodel.LowerOptions{
		SourcePath:      "ICallback.aidl",
		DependencyFiles: []*ast.File{alphaFile},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower gamma diagnostics = %#v", diags)
	}
	betaModel, diags := gomodel.Lower(betaFile, gomodel.LowerOptions{
		SourcePath:      "IBar.aidl",
		DependencyFiles: []*ast.File{alphaFile, gammaFile},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower beta diagnostics = %#v", diags)
	}

	const importRoot = "example.com/generated"
	var outputs []OutputFile
	for _, model := range []*gomodel.File{alphaModel, gammaModel, betaModel} {
		rendered, err := RenderGo(model, GoOptions{GeneratedImportRoot: importRoot})
		if err != nil {
			t.Fatalf("RenderGo(%s): %v", model.SourcePath, err)
		}
		outputs = append(outputs, rendered...)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}
	for _, output := range outputs {
		outPath := filepath.Join(tmp, output.Path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", outPath, err)
		}
		if err := os.WriteFile(outPath, output.Content, 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", outPath, err)
		}
	}

	testSrc := `package beta

import (
	"context"
	"testing"

	alpha "example.com/generated/alpha"
	gamma "example.com/generated/gamma"
	binder "github.com/wdsgyj/libbinder-go/binder"
)

type fakeRegistrar struct {
	next    uint32
	binders map[uint32]binder.Binder
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{
		next:    100,
		binders: map[uint32]binder.Binder{},
	}
}

func (r *fakeRegistrar) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	r.next++
	b := fakeRegisteredBinder{handle: r.next, handler: handler, registrar: r}
	r.binders[b.handle] = b
	return b, nil
}

func (r *fakeRegistrar) resolve(handle uint32) binder.Binder {
	return r.binders[handle]
}

type fakeRegisteredBinder struct {
	handle    uint32
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeRegisteredBinder) AsBinder() binder.Binder { return b }

func (b fakeRegisteredBinder) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeRegisteredBinder) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeRegisteredBinder) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeRegisteredBinder) Close() error { return nil }

func (b fakeRegisteredBinder) WriteBinderToParcel(p *binder.Parcel) error {
	return p.WriteStrongBinderHandle(b.handle)
}

type fakeEndpoint struct {
	handler   binder.Handler
	registrar *fakeRegistrar
}

func (b fakeEndpoint) Descriptor(ctx context.Context) (string, error) {
	return b.handler.Descriptor(), nil
}

func (b fakeEndpoint) Transact(ctx context.Context, code uint32, data *binder.Parcel, flags binder.Flags) (*binder.Parcel, error) {
	if err := data.SetPosition(0); err != nil {
		return nil, err
	}
	data.SetBinderResolvers(b.registrar.resolve, nil)
	reply, err := b.handler.HandleTransact(ctx, code, data)
	if err != nil {
		return nil, err
	}
	if reply != nil {
		reply.SetBinderResolvers(b.registrar.resolve, nil)
		if err := reply.SetPosition(0); err != nil {
			return nil, err
		}
	}
	return reply, nil
}

func (b fakeEndpoint) WatchDeath(ctx context.Context) (binder.Subscription, error) {
	return nil, binder.ErrUnsupported
}

func (b fakeEndpoint) Close() error { return nil }

func (b fakeEndpoint) RegisterLocalHandler(handler binder.Handler) (binder.Binder, error) {
	return b.registrar.RegisterLocalHandler(handler)
}

type callbackImpl struct {
	seen []int32
}

func (c *callbackImpl) Done(ctx context.Context, value alpha.Foo) (alpha.Foo, error) {
	c.seen = append(c.seen, value.Count)
	return alpha.Foo{Count: value.Count + 1}, nil
}

type serviceImpl struct{}

func (serviceImpl) Echo(ctx context.Context, value alpha.Foo, cb gamma.ICallback) (alpha.Foo, error) {
	reply, err := cb.Done(ctx, alpha.Foo{Count: value.Count + 10})
	if err != nil {
		return alpha.Foo{}, err
	}
	return alpha.Foo{Count: reply.Count + 100}, nil
}

func TestCrossPackageGeneratedRoundTrip(t *testing.T) {
	registrar := newFakeRegistrar()
	client := NewIBarClient(fakeEndpoint{
		handler:   NewIBarHandlerWithRegistrar(registrar, serviceImpl{}),
		registrar: registrar,
	})

	cb := &callbackImpl{}
	got, err := client.Echo(context.Background(), alpha.Foo{Count: 7}, cb)
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if len(cb.seen) != 1 || cb.seen[0] != 17 {
		t.Fatalf("callback seen = %#v, want [17]", cb.seen)
	}
	if got.Count != 118 {
		t.Fatalf("got.Count = %d, want 118", got.Count)
	}
}
`
	testPath := filepath.Join(tmp, "beta", "generated_crosspkg_test.go")
	if err := os.WriteFile(testPath, []byte(testSrc), 0o644); err != nil {
		t.Fatalf("WriteFile(generated_crosspkg_test.go): %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func TestRenderGoSupportsInternalNamedAIDLPackages(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip("requires host go toolchain")
	}

	internalSrc := `
package com.android.internal.app;

interface IVoiceInteractor {
  void Ping(in String msg);
}
`
	appSrc := `
package android.app;

import com.android.internal.app.IVoiceInteractor;

interface IService {
  void Call(in IVoiceInteractor interactor);
}
`

	internalFile, err := parser.Parse("IVoiceInteractor.aidl", internalSrc)
	if err != nil {
		t.Fatalf("Parse(IVoiceInteractor.aidl): %v", err)
	}
	appFile, err := parser.Parse("IService.aidl", appSrc)
	if err != nil {
		t.Fatalf("Parse(IService.aidl): %v", err)
	}

	internalModel, diags := gomodel.Lower(internalFile, gomodel.LowerOptions{SourcePath: "IVoiceInteractor.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower internal diagnostics = %#v", diags)
	}
	appModel, diags := gomodel.Lower(appFile, gomodel.LowerOptions{
		SourcePath:      "IService.aidl",
		DependencyFiles: []*ast.File{internalFile},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower app diagnostics = %#v", diags)
	}

	const importRoot = "example.com/generated"
	var outputs []OutputFile
	for _, model := range []*gomodel.File{internalModel, appModel} {
		rendered, err := RenderGo(model, GoOptions{GeneratedImportRoot: importRoot})
		if err != nil {
			t.Fatalf("RenderGo(%s): %v", model.SourcePath, err)
		}
		outputs = append(outputs, rendered...)
	}

	tmp := t.TempDir()
	repoRoot := testRepoRoot(t)
	goMod := `module example.com/generated

go 1.22

require github.com/wdsgyj/libbinder-go v0.0.0

replace github.com/wdsgyj/libbinder-go => ` + repoRoot + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}
	for _, output := range outputs {
		outPath := filepath.Join(tmp, output.Path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", outPath, err)
		}
		if err := os.WriteFile(outPath, output.Content, 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", outPath, err)
		}
	}

	if _, err := os.Stat(filepath.Join(tmp, "com", "android", "internal_", "app", "ivoiceinteractor_aidl.go")); err != nil {
		t.Fatalf("generated internal_ path missing: %v", err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, out)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	root = filepath.ToSlash(root)
	if strings.HasPrefix(root, "/") {
		return root
	}
	return filepath.Clean(root)
}
