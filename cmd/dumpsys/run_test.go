package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	api "github.com/wdsgyj/libbinder-go/binder"
	"github.com/wdsgyj/libbinder-go/internal/binderdebug"
)

func TestRunListOnly(t *testing.T) {
	sm := fakeServiceManager{
		listServices: func(ctx context.Context, flags api.DumpFlags) ([]string, error) {
			if flags != api.DumpPriorityAll {
				t.Fatalf("flags = %#x, want %#x", flags, api.DumpPriorityAll)
			}
			return []string{"zeta", "alpha"}, nil
		},
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{descriptor: name}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"-l"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		DebugReader:    fakeDebugReader{},
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); got != "Currently running services:\n  alpha\n  zeta\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunSingleServiceDumpWithArgs(t *testing.T) {
	var gotArgs []string
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{
				descriptor: "svc",
				transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
					switch code {
					case api.DumpTransaction:
						fd, args, err := apiReadDumpRequest(data)
						if err != nil {
							return nil, err
						}
						gotArgs = args
						if _, err := io.WriteString(os.NewFile(uintptr(fd.FD()), "dump"), "dump body\n"); err != nil {
							return nil, err
						}
						return api.NewParcel(), nil
					case api.DebugPIDTransaction:
						reply := api.NewParcel()
						if err := reply.WriteInt32(100); err != nil {
							return nil, err
						}
						return reply, nil
					default:
						return nil, api.ErrUnknownTransaction
					}
				},
			}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"activity", "--proto", "x"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		DebugReader:    fakeDebugReader{},
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0, stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); got != "dump body\n" {
		t.Fatalf("stdout = %q, want dump output", got)
	}
	if !reflect.DeepEqual(gotArgs, []string{"--proto", "x"}) {
		t.Fatalf("dump args = %#v, want [--proto x]", gotArgs)
	}
}

func TestRunAllServicesWithPriorityAndSkip(t *testing.T) {
	var dumped []string
	sm := fakeServiceManager{
		listServices: func(ctx context.Context, flags api.DumpFlags) ([]string, error) {
			switch flags {
			case api.DumpPriorityHigh:
				return []string{"b", "a"}, nil
			default:
				t.Fatalf("unexpected flags %#x", flags)
				return nil, nil
			}
		},
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{
				descriptor: name,
				transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
					if code != api.DumpTransaction {
						return nil, api.ErrUnknownTransaction
					}
					fd, args, err := apiReadDumpRequest(data)
					if err != nil {
						return nil, err
					}
					if !reflect.DeepEqual(args, []string{"--priority", "HIGH"}) {
						t.Fatalf("dump args = %#v, want [--priority HIGH]", args)
					}
					dumped = append(dumped, name)
					_, _ = io.WriteString(os.NewFile(uintptr(fd.FD()), "dump"), name+"\n")
					return api.NewParcel(), nil
				},
			}, nil
		},
	}

	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"--priority", "HIGH", "--skip", "b"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          io.Discard,
		DebugReader:    fakeDebugReader{},
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if !reflect.DeepEqual(dumped, []string{"a"}) {
		t.Fatalf("dumped = %#v, want [a]", dumped)
	}
	if got := stdout.String(); !strings.Contains(got, "Currently running services:") || !strings.Contains(got, "  b (skipped)") || !strings.Contains(got, "DUMP OF SERVICE HIGH a:") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunAuxiliaryDumpTypes(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{
				descriptor: "svc",
				handle:     910,
				stability:  api.StabilityVINTF,
				transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
					if code != api.DebugPIDTransaction {
						return nil, api.ErrUnknownTransaction
					}
					reply := api.NewParcel()
					if err := reply.WriteInt32(321); err != nil {
						return nil, err
					}
					return reply, nil
				},
			}, nil
		},
	}
	debugReader := fakeDebugReader{
		pidInfo:    binderdebugInfo{threadUsage: 2, threadCount: 4},
		clientPIDs: []int{os.Getpid(), 111, 222},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--pid", "--stability", "--thread", "--clients", "svc"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		DebugReader:    debugReader,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0, stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "Service host process PID: 321") || !strings.Contains(got, "Stability: vintf") || !strings.Contains(got, "Threads in use: 2/4") || !strings.Contains(got, "Client PIDs: 111, 222") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunThreadAndClientsErrorsStillDump(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{
				descriptor: "svc",
				handle:     910,
				transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
					switch code {
					case api.DebugPIDTransaction:
						reply := api.NewParcel()
						if err := reply.WriteInt32(321); err != nil {
							return nil, err
						}
						return reply, nil
					case api.DumpTransaction:
						fd, _, err := apiReadDumpRequest(data)
						if err != nil {
							return nil, err
						}
						if _, err := io.WriteString(os.NewFile(uintptr(fd.FD()), "dump"), "dump body\n"); err != nil {
							return nil, err
						}
						return api.NewParcel(), nil
					default:
						return nil, api.ErrUnknownTransaction
					}
				},
			}, nil
		},
	}
	debugReader := fakeDebugReader{
		pidErr:     os.ErrNotExist,
		clientsErr: os.ErrNotExist,
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--thread", "--clients", "--dump", "svc"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		DebugReader:    debugReader,
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0, stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "dump body\n" {
		t.Fatalf("stdout = %q, want dump output", got)
	}
	if got := stderr.String(); !strings.Contains(got, "Error with service 'svc' while dumping thread info: NAME_NOT_FOUND") || !strings.Contains(got, "Error with service 'svc' while dumping clients info: NAME_NOT_FOUND") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunTimeout(t *testing.T) {
	sm := fakeServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return fakeDumpBinder{
				descriptor: "svc",
				transact: func(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
					if code != api.DumpTransaction {
						return nil, api.ErrUnknownTransaction
					}
					fd, _, err := apiReadDumpRequest(data)
					if err != nil {
						return nil, err
					}
					file := os.NewFile(uintptr(fd.FD()), "dump")
					defer func() { _ = file.Close() }()
					_, _ = io.WriteString(file, "partial")
					time.Sleep(50 * time.Millisecond)
					_, _ = io.WriteString(file, "late")
					return api.NewParcel(), nil
				},
			}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"-T", "10", "svc"}, Options{
		ServiceManager: sm,
		Output:         &stdout,
		Error:          &stderr,
		DebugReader:    fakeDebugReader{},
	})
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if got := stdout.String(); !strings.Contains(got, "partial") || !strings.Contains(got, "DUMP TIMEOUT") {
		t.Fatalf("stdout = %q, want partial output + timeout", got)
	}
}

func TestParseArgsInvalidPriority(t *testing.T) {
	var stderr bytes.Buffer
	_, code := parseArgs([]string{"--priority", "LOW"}, &stderr)
	if code != usageExitCode {
		t.Fatalf("code = %d, want %d", code, usageExitCode)
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
func (f fakeServiceManager) ListServices(ctx context.Context, flags api.DumpFlags) ([]string, error) {
	if f.listServices == nil {
		return nil, nil
	}
	return f.listServices(ctx, flags)
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

type fakeDumpBinder struct {
	descriptor string
	handle     uint32
	stability  api.StabilityLevel
	transact   func(context.Context, uint32, *api.Parcel, api.Flags) (*api.Parcel, error)
}

func (b fakeDumpBinder) Descriptor(ctx context.Context) (string, error) { return b.descriptor, nil }
func (b fakeDumpBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	if b.transact == nil {
		return nil, api.ErrUnsupported
	}
	reply, err := b.transact(ctx, code, data, flags)
	if reply != nil {
		_ = reply.SetPosition(0)
	}
	return reply, err
}
func (b fakeDumpBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}
func (b fakeDumpBinder) Close() error { return nil }
func (b fakeDumpBinder) StabilityLevel() api.StabilityLevel {
	if b.stability == 0 {
		return api.StabilitySystem
	}
	return b.stability
}
func (b fakeDumpBinder) DebugHandle() (uint32, bool) {
	if b.handle == 0 {
		return 0, false
	}
	return b.handle, true
}

type binderdebugInfo struct {
	threadUsage uint32
	threadCount uint32
}

type fakeDebugReader struct {
	pidInfo    binderdebugInfo
	clientPIDs []int
	pidErr     error
	clientsErr error
}

func (f fakeDebugReader) GetPIDInfo(pid int) (binderdebug.PIDInfo, error) {
	if f.pidErr != nil {
		return binderdebug.PIDInfo{}, f.pidErr
	}
	return binderdebug.PIDInfo{
		ThreadUsage: f.pidInfo.threadUsage,
		ThreadCount: f.pidInfo.threadCount,
	}, nil
}
func (f fakeDebugReader) GetClientPIDs(callerPID int, servicePID int, handle uint32) ([]int, error) {
	if f.clientsErr != nil {
		return nil, f.clientsErr
	}
	return append([]int(nil), f.clientPIDs...), nil
}

func apiReadDumpRequest(p *api.Parcel) (api.FileDescriptor, []string, error) {
	if err := p.SetPosition(0); err != nil {
		return api.FileDescriptor{}, nil, err
	}
	fd, err := p.ReadFileDescriptor()
	if err != nil {
		return api.FileDescriptor{}, nil, err
	}
	count, err := p.ReadInt32()
	if err != nil {
		return api.FileDescriptor{}, nil, err
	}
	args := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		arg, err := p.ReadString()
		if err != nil {
			return api.FileDescriptor{}, nil, err
		}
		args = append(args, arg)
	}
	return fd, args, nil
}
