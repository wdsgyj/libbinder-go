package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestRunUsageWhenNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, Options{
		ServiceManager: fakeServiceManager{},
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "usage: service") {
		t.Fatalf("stdout = %q, want usage", got)
	}
}

func TestRunListServices(t *testing.T) {
	sm := fakeServiceManager{
		listServices: func(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
			if dumpFlags != api.DumpPriorityAll {
				t.Fatalf("ListServices flags = %#x, want %#x", dumpFlags, api.DumpPriorityAll)
			}
			return []string{"zeta", "alpha"}, nil
		},
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			switch name {
			case "zeta":
				return fakeBinder{descriptor: "iface.zeta"}, nil
			case "alpha":
				return fakeBinder{descriptor: "iface.alpha"}, nil
			default:
				return nil, nil
			}
		},
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"list"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	want := "Found 2 services:\n0\tzeta: [iface.zeta]\n1\talpha: [iface.alpha]\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunCheckService(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if name == "found" {
				return fakeBinder{descriptor: "iface"}, nil
			}
			return nil, nil
		},
	}

	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"check", "found"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          io.Discard,
	})
	if code != 0 {
		t.Fatalf("Run(found) code = %d, want 0", code)
	}
	if got := stdout.String(); got != "Service found: found\n" {
		t.Fatalf("stdout(found) = %q", got)
	}

	stdout.Reset()
	code = Run(context.Background(), []string{"check", "missing"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          io.Discard,
	})
	if code != 0 {
		t.Fatalf("Run(missing) code = %d, want 0", code)
	}
	if got := stdout.String(); got != "Service missing: not found\n" {
		t.Fatalf("stdout(missing) = %q", got)
	}
}

func TestRunCallEncodesAllArgumentKinds(t *testing.T) {
	plainPath := writeTempFile(t, "plain.txt", "plain-fd")
	anonPath := writeTempFile(t, "anon.txt", "anon-fd")
	nfdFile, err := os.Open(plainPath)
	if err != nil {
		t.Fatalf("os.Open(%s): %v", plainPath, err)
	}
	defer func() { _ = nfdFile.Close() }()

	var stdout, stderr bytes.Buffer
	service := fakeBinder{
		descriptor: "android.test.IFake",
		transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
			if code != 7 {
				t.Fatalf("code = %d, want 7", code)
			}
			if flags != api.FlagNone {
				t.Fatalf("flags = %#x, want 0", flags)
			}
			if err := data.SetPosition(0); err != nil {
				t.Fatalf("SetPosition: %v", err)
			}
			token, err := data.ReadInterfaceToken()
			if err != nil {
				t.Fatalf("ReadInterfaceToken: %v", err)
			}
			if token != "android.test.IFake" {
				t.Fatalf("token = %q, want android.test.IFake", token)
			}

			gotI32, err := data.ReadInt32()
			if err != nil || gotI32 != 12 {
				t.Fatalf("ReadInt32 = (%d, %v), want (12, nil)", gotI32, err)
			}
			gotI64, err := data.ReadInt64()
			if err != nil || gotI64 != 34 {
				t.Fatalf("ReadInt64 = (%d, %v), want (34, nil)", gotI64, err)
			}
			gotF32, err := data.ReadFloat32()
			if err != nil || gotF32 != 1.5 {
				t.Fatalf("ReadFloat32 = (%v, %v), want (1.5, nil)", gotF32, err)
			}
			gotF64, err := data.ReadFloat64()
			if err != nil || gotF64 != 2.25 {
				t.Fatalf("ReadFloat64 = (%v, %v), want (2.25, nil)", gotF64, err)
			}
			gotS16, err := data.ReadString()
			if err != nil || gotS16 != "hello" {
				t.Fatalf("ReadString = (%q, %v), want (hello, nil)", gotS16, err)
			}
			gotNull, err := data.ReadStrongBinderHandle()
			if err != nil || gotNull != nil {
				t.Fatalf("ReadStrongBinderHandle(null) = (%v, %v), want (nil, nil)", gotNull, err)
			}

			if got := readFDContents(t, data); got != "plain-fd" {
				t.Fatalf("fd contents = %q, want plain-fd", got)
			}
			if got := readFDContents(t, data); got != "plain-fd" {
				t.Fatalf("nfd contents = %q, want plain-fd", got)
			}
			if got := readFDContents(t, data); got != "anon-fd" {
				t.Fatalf("afd contents = %q, want anon-fd", got)
			}

			action, err := data.ReadNullableString()
			if err != nil || derefString(action) != "android.intent.action.VIEW" {
				t.Fatalf("action = (%v, %v), want VIEW", action, err)
			}
			intentData, err := data.ReadNullableString()
			if err != nil || derefString(intentData) != "content://demo/item" {
				t.Fatalf("data = (%v, %v), want content://demo/item", intentData, err)
			}
			intentType, err := data.ReadNullableString()
			if err != nil || derefString(intentType) != "text/plain" {
				t.Fatalf("type = (%v, %v), want text/plain", intentType, err)
			}
			flags32, err := data.ReadInt32()
			if err != nil || flags32 != 3 {
				t.Fatalf("launchFlags = (%d, %v), want (3, nil)", flags32, err)
			}
			component, err := data.ReadNullableString()
			if err != nil || derefString(component) != "pkg/.Cls" {
				t.Fatalf("component = (%v, %v), want pkg/.Cls", component, err)
			}
			categoryCount, err := data.ReadInt32()
			if err != nil || categoryCount != 2 {
				t.Fatalf("categoryCount = (%d, %v), want (2, nil)", categoryCount, err)
			}
			category0, _ := data.ReadString()
			category1, _ := data.ReadString()
			if category0 != "one" || category1 != "two" {
				t.Fatalf("categories = (%q, %q), want (one, two)", category0, category1)
			}
			extras, err := data.ReadInt32()
			if err != nil || extras != -1 {
				t.Fatalf("extras = (%d, %v), want (-1, nil)", extras, err)
			}

			reply := api.NewParcel()
			if err := reply.WriteInt32(99); err != nil {
				t.Fatalf("reply.WriteInt32: %v", err)
			}
			if err := reply.WriteStrongBinderHandleWithStability(11, api.StabilitySystem); err != nil {
				t.Fatalf("reply.WriteStrongBinderHandleWithStability: %v", err)
			}
			return reply, nil
		},
	}
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if name != "test.svc" {
				t.Fatalf("CheckService name = %q, want test.svc", name)
			}
			return service, nil
		},
	}

	code := Run(context.Background(), []string{
		"call", "test.svc", "7",
		"i32", "12",
		"i64", "34",
		"f", "1.5",
		"d", "2.25",
		"s16", "hello",
		"null",
		"fd", plainPath,
		"nfd", strconvItoa(int(nfdFile.Fd())),
		"afd", anonPath,
		"intent",
		"action=android.intent.action.VIEW",
		"data=content://demo/item",
		"type=text/plain",
		"launchFlags=3",
		"component=pkg/.Cls",
		"categories=one,two",
	}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0, stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "Result: Parcel{size=") || !strings.Contains(got, "kind=strong-binder") {
		t.Fatalf("stdout = %q, want formatted parcel dump", got)
	}
}

func TestRunCallUnknownArgumentType(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeBinder{descriptor: "iface"}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"call", "svc", "1", "bogus"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
	})
	if code != usageExitCode {
		t.Fatalf("Run code = %d, want %d", code, usageExitCode)
	}
	if got := stderr.String(); !strings.Contains(got, "Unknown argument type bogus") {
		t.Fatalf("stderr = %q, want unknown argument error", got)
	}
	if got := stdout.String(); !strings.Contains(got, "usage: service") {
		t.Fatalf("stdout = %q, want usage", got)
	}
}

func TestFormatParcel(t *testing.T) {
	p := api.NewParcel()
	if err := p.WriteInt32(42); err != nil {
		t.Fatalf("WriteInt32: %v", err)
	}
	if err := p.WriteStrongBinderHandleWithStability(9, api.StabilityVendor); err != nil {
		t.Fatalf("WriteStrongBinderHandleWithStability: %v", err)
	}
	got := formatParcel(p)
	if !strings.Contains(got, "data:") || !strings.Contains(got, "kind=strong-binder") || !strings.Contains(got, "stability=vendor") {
		t.Fatalf("formatParcel = %q, want dump with objects", got)
	}
}

type fakeServiceManager struct {
	checkService func(context.Context, string) (api.Binder, error)
	listServices func(context.Context, api.DumpFlags) ([]string, error)
}

func (f fakeServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if f.checkService == nil {
		return nil, nil
	}
	return f.checkService(ctx, name)
}

func (f fakeServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return api.ErrUnsupported
}

func (f fakeServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	if f.listServices == nil {
		return nil, nil
	}
	return f.listServices(ctx, dumpFlags)
}

func (f fakeServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, api.ErrUnsupported
}

func (f fakeServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (f fakeServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	return api.ErrUnsupported
}

func (f fakeServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return nil, api.ErrUnsupported
}

type fakeBinder struct {
	descriptor string
	transact   func(context.Context, uint32, *api.Parcel, api.Flags) (*api.Parcel, error)
}

func (b fakeBinder) Descriptor(ctx context.Context) (string, error) {
	return b.descriptor, nil
}

func (b fakeBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if b.transact == nil {
		return nil, api.ErrUnsupported
	}
	return b.transact(ctx, code, data, flags)
}

func (b fakeBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b fakeBinder) Close() error {
	return nil
}

func writeTempFile(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepathJoin(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

func readFDContents(t *testing.T, p *api.Parcel) string {
	t.Helper()
	fd, err := p.ReadFileDescriptor()
	if err != nil {
		t.Fatalf("ReadFileDescriptor: %v", err)
	}
	dup, err := duplicateFD(fd.FD())
	if err != nil {
		t.Fatalf("duplicateFD(%d): %v", fd.FD(), err)
	}
	defer func() { _ = dup.Close() }()
	if _, err := dup.Seek(0, 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	data, err := io.ReadAll(dup)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(data)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

func strconvItoa(v int) string {
	return fmt.Sprintf("%d", v)
}
