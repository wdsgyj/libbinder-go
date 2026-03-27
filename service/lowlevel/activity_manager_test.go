package lowlevel

import (
	"context"
	"errors"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestLookupActivityManager(t *testing.T) {
	activity := &testBinder{descriptor: "android.app.IActivityManager"}
	activityTask := &testBinder{descriptor: "android.app.IActivityTaskManager"}

	var calls []string
	sm := testServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			calls = append(calls, name)
			switch name {
			case ActivityServiceName:
				return activity, nil
			case ActivityTaskServiceName:
				return activityTask, nil
			default:
				t.Fatalf("unexpected service lookup %q", name)
				return nil, nil
			}
		},
	}

	manager, err := LookupActivityManager(context.Background(), sm)
	if err != nil {
		t.Fatalf("LookupActivityManager: %v", err)
	}
	if manager.GetService() != activity {
		t.Fatalf("GetService() = %#v, want %#v", manager.GetService(), activity)
	}
	if manager.GetTaskService() != activityTask {
		t.Fatalf("GetTaskService() = %#v, want %#v", manager.GetTaskService(), activityTask)
	}
	wantCalls := []string{ActivityServiceName, ActivityTaskServiceName}
	assertStringSlice(t, calls, wantCalls)
}

func TestLookupActivityManagerMissingService(t *testing.T) {
	sm := testServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			if name == ActivityServiceName {
				return &testBinder{}, nil
			}
			return nil, nil
		},
	}

	_, err := LookupActivityManager(context.Background(), sm)
	if !errors.Is(err, api.ErrNoService) {
		t.Fatalf("err = %v, want ErrNoService", err)
	}
}

func TestWaitActivityManager(t *testing.T) {
	activity := &testBinder{descriptor: "android.app.IActivityManager"}
	activityTask := &testBinder{descriptor: "android.app.IActivityTaskManager"}

	var calls []string
	sm := testServiceManager{
		waitService: func(ctx context.Context, name string) (api.Binder, error) {
			calls = append(calls, name)
			switch name {
			case ActivityServiceName:
				return activity, nil
			case ActivityTaskServiceName:
				return activityTask, nil
			default:
				t.Fatalf("unexpected service wait %q", name)
				return nil, nil
			}
		},
	}

	manager, err := WaitActivityManager(context.Background(), sm)
	if err != nil {
		t.Fatalf("WaitActivityManager: %v", err)
	}
	if manager.ActivityService() != activity {
		t.Fatalf("ActivityService() = %#v, want %#v", manager.ActivityService(), activity)
	}
	if manager.ActivityTaskService() != activityTask {
		t.Fatalf("ActivityTaskService() = %#v, want %#v", manager.ActivityTaskService(), activityTask)
	}
	wantCalls := []string{ActivityServiceName, ActivityTaskServiceName}
	assertStringSlice(t, calls, wantCalls)
}

func TestActivityManagerProviderCaches(t *testing.T) {
	activity := &testBinder{descriptor: "android.app.IActivityManager"}
	activityTask := &testBinder{descriptor: "android.app.IActivityTaskManager"}

	checkCount := 0
	sm := testServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			checkCount++
			switch name {
			case ActivityServiceName:
				return activity, nil
			case ActivityTaskServiceName:
				return activityTask, nil
			default:
				t.Fatalf("unexpected service lookup %q", name)
				return nil, nil
			}
		},
	}

	provider := NewActivityManagerProvider(sm)
	first, err := provider.Get(context.Background())
	if err != nil {
		t.Fatalf("Get(first): %v", err)
	}
	second, err := provider.Get(context.Background())
	if err != nil {
		t.Fatalf("Get(second): %v", err)
	}
	if first != second {
		t.Fatalf("cached manager pointers differ: %#v vs %#v", first, second)
	}
	if checkCount != 2 {
		t.Fatalf("checkCount = %d, want 2", checkCount)
	}

	provider.Invalidate()

	third, err := provider.Get(context.Background())
	if err != nil {
		t.Fatalf("Get(third): %v", err)
	}
	if third == first {
		t.Fatalf("Invalidate() did not drop cached instance")
	}
	if checkCount != 4 {
		t.Fatalf("checkCount after invalidate = %d, want 4", checkCount)
	}
}

func TestActivityManagerProviderNilServiceManager(t *testing.T) {
	_, err := NewActivityManagerProvider(nil).Get(context.Background())
	if !errors.Is(err, api.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

type testBinder struct {
	descriptor string
}

func (b *testBinder) Descriptor(ctx context.Context) (string, error) {
	return b.descriptor, nil
}

func (b *testBinder) Transact(ctx context.Context, code uint32, data *api.Parcel, flags api.Flags) (*api.Parcel, error) {
	return nil, api.ErrUnsupported
}

func (b *testBinder) WatchDeath(ctx context.Context) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (b *testBinder) Close() error {
	return nil
}

type testServiceManager struct {
	checkService func(context.Context, string) (api.Binder, error)
	waitService  func(context.Context, string) (api.Binder, error)
}

func (sm testServiceManager) CheckService(ctx context.Context, name string) (api.Binder, error) {
	if sm.checkService == nil {
		return nil, api.ErrUnsupported
	}
	return sm.checkService(ctx, name)
}

func (sm testServiceManager) WaitService(ctx context.Context, name string) (api.Binder, error) {
	if sm.waitService == nil {
		return nil, api.ErrUnsupported
	}
	return sm.waitService(ctx, name)
}

func (sm testServiceManager) AddService(ctx context.Context, name string, handler api.Handler, opts ...api.AddServiceOption) error {
	return api.ErrUnsupported
}

func (sm testServiceManager) ListServices(ctx context.Context, dumpFlags api.DumpFlags) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) WatchServiceRegistrations(ctx context.Context, name string, callback api.ServiceRegistrationCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) IsDeclared(ctx context.Context, name string) (bool, error) {
	return false, api.ErrUnsupported
}

func (sm testServiceManager) DeclaredInstances(ctx context.Context, iface string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) UpdatableViaApex(ctx context.Context, name string) (*string, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) UpdatableNames(ctx context.Context, apexName string) ([]string, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) ConnectionInfo(ctx context.Context, name string) (*api.ConnectionInfo, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) WatchClients(ctx context.Context, name string, service api.Binder, callback api.ServiceClientCallback) (api.Subscription, error) {
	return nil, api.ErrUnsupported
}

func (sm testServiceManager) TryUnregisterService(ctx context.Context, name string, service api.Binder) error {
	return api.ErrUnsupported
}

func (sm testServiceManager) DebugInfo(ctx context.Context) ([]api.ServiceDebugInfo, error) {
	return nil, api.ErrUnsupported
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}
