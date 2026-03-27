package service

import (
	"context"
	"errors"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestLookupActivityService(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		want := &inputTestBinder{}
		sm := inputTestServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) {
				if name != ActivityServiceName {
					t.Fatalf("name = %q, want %q", name, ActivityServiceName)
				}
				return want, nil
			},
		}
		got, err := LookupActivityService(context.Background(), sm)
		if err != nil {
			t.Fatalf("LookupActivityService: %v", err)
		}
		if got != want {
			t.Fatalf("got = %#v, want %#v", got, want)
		}
	})

	t.Run("missing", func(t *testing.T) {
		sm := inputTestServiceManager{
			checkService: func(ctx context.Context, name string) (api.Binder, error) { return nil, nil },
		}
		_, err := LookupActivityService(context.Background(), sm)
		if !errors.Is(err, api.ErrNoService) {
			t.Fatalf("err = %v, want ErrNoService", err)
		}
	})

	t.Run("nil service manager", func(t *testing.T) {
		_, err := LookupActivityService(context.Background(), nil)
		if !errors.Is(err, api.ErrUnsupported) {
			t.Fatalf("err = %v, want ErrUnsupported", err)
		}
	})
}

func TestLookupActivityManagerService(t *testing.T) {
	want := &inputTestBinder{}
	sm := inputTestServiceManager{
		checkService: func(ctx context.Context, name string) (api.Binder, error) {
			return want, nil
		},
	}
	got, err := LookupActivityManagerService(context.Background(), sm)
	if err != nil {
		t.Fatalf("LookupActivityManagerService: %v", err)
	}
	if got == nil {
		t.Fatal("LookupActivityManagerService() = nil")
	}
	if got.Binder() != want {
		t.Fatalf("Binder() = %#v, want %#v", got.Binder(), want)
	}
}

func TestActivityManagerServiceConfiguration(t *testing.T) {
	base := NewActivityManagerService(inputTestBinder{})
	if base == nil {
		t.Fatal("NewActivityManagerService() = nil")
	}
	checker := fileAccessCheckerFunc(func(path string, seLinuxContext string, read bool, write bool) error { return nil })
	scoped := base.WithWorkingDir("/tmp").WithFileAccessChecker(checker).WithShellIO(ShellCommandIO{
		InFD:  api.NewFileDescriptor(3),
		OutFD: api.NewFileDescriptor(4),
		ErrFD: api.NewFileDescriptor(5),
	})
	if scoped.Binder() == nil {
		t.Fatal("Binder() = nil")
	}
	if scoped.shell.workingDir != "/tmp" {
		t.Fatalf("workingDir = %q, want /tmp", scoped.shell.workingDir)
	}
	if scoped.shell.accessChecker == nil {
		t.Fatal("accessChecker = nil")
	}
	if scoped.shell.io.InFD.FD() != 3 || scoped.shell.io.OutFD.FD() != 4 || scoped.shell.io.ErrFD.FD() != 5 {
		t.Fatalf("io = (%d,%d,%d), want (3,4,5)", scoped.shell.io.InFD.FD(), scoped.shell.io.OutFD.FD(), scoped.shell.io.ErrFD.FD())
	}
	if base.shell.workingDir != "" || base.shell.accessChecker != nil {
		t.Fatalf("base mutated: %#v", base.shell)
	}
}

func TestActivityManagerServiceTopLevelCommands(t *testing.T) {
	tests := []struct {
		name string
		call func(*ActivityManagerService) (int, error)
		want []string
	}{
		{"help", func(s *ActivityManagerService) (int, error) { return s.Help(context.Background()) }, []string{"help"}},
		{"logging", func(s *ActivityManagerService) (int, error) {
			return s.Logging(context.Background(), "enable-text", "am_proc_start")
		}, []string{"logging", "enable-text", "am_proc_start"}},
		{"app logging", func(s *ActivityManagerService) (int, error) {
			return s.AppLogging(context.Background(), "proc", 1000, "disable-text", "am_proc_start")
		}, []string{"app-logging", "proc", "1000", "disable-text", "am_proc_start"}},
		{"start activity", func(s *ActivityManagerService) (int, error) {
			return s.StartActivity(context.Background(), "-W", "intent://demo")
		}, []string{"start-activity", "-W", "intent://demo"}},
		{"start in vsync", func(s *ActivityManagerService) (int, error) {
			return s.StartInVsync(context.Background(), "-W", "intent://demo")
		}, []string{"start-in-vsync", "-W", "intent://demo"}},
		{"start service", func(s *ActivityManagerService) (int, error) {
			return s.StartService(context.Background(), "--user", "current", "intent://demo")
		}, []string{"start-service", "--user", "current", "intent://demo"}},
		{"start foreground service", func(s *ActivityManagerService) (int, error) {
			return s.StartForegroundService(context.Background(), "intent://demo")
		}, []string{"start-foreground-service", "intent://demo"}},
		{"stop service", func(s *ActivityManagerService) (int, error) {
			return s.StopService(context.Background(), "intent://demo")
		}, []string{"stop-service", "intent://demo"}},
		{"broadcast", func(s *ActivityManagerService) (int, error) {
			return s.Broadcast(context.Background(), "--async", "intent://demo")
		}, []string{"broadcast", "--async", "intent://demo"}},
		{"compact some", func(s *ActivityManagerService) (int, error) { return s.CompactSome(context.Background(), "proc") }, []string{"compact", "some", "proc"}},
		{"compact full", func(s *ActivityManagerService) (int, error) { return s.CompactFull(context.Background(), "proc") }, []string{"compact", "full", "proc"}},
		{"compact system", func(s *ActivityManagerService) (int, error) { return s.CompactSystem(context.Background()) }, []string{"compact", "system"}},
		{"compact native some", func(s *ActivityManagerService) (int, error) { return s.CompactNativeSome(context.Background(), "123") }, []string{"compact", "native", "some", "123"}},
		{"compact native full", func(s *ActivityManagerService) (int, error) { return s.CompactNativeFull(context.Background(), "123") }, []string{"compact", "native", "full", "123"}},
		{"freeze", func(s *ActivityManagerService) (int, error) {
			return s.Freeze(context.Background(), "--sticky", "proc")
		}, []string{"freeze", "--sticky", "proc"}},
		{"unfreeze", func(s *ActivityManagerService) (int, error) { return s.Unfreeze(context.Background(), "proc") }, []string{"unfreeze", "proc"}},
		{"instrument", func(s *ActivityManagerService) (int, error) {
			return s.Instrument(context.Background(), "-w", "pkg/Runner")
		}, []string{"instrument", "-w", "pkg/Runner"}},
		{"trace ipc start", func(s *ActivityManagerService) (int, error) { return s.TraceIPCStart(context.Background()) }, []string{"trace-ipc", "start"}},
		{"trace ipc stop", func(s *ActivityManagerService) (int, error) {
			return s.TraceIPCStop(context.Background(), "--dump-file", "/tmp/trace.bin")
		}, []string{"trace-ipc", "stop", "--dump-file", "/tmp/trace.bin"}},
		{"profile start", func(s *ActivityManagerService) (int, error) {
			return s.ProfileStart(context.Background(), "proc", "/tmp/profile.trace")
		}, []string{"profile", "start", "proc", "/tmp/profile.trace"}},
		{"profile stop", func(s *ActivityManagerService) (int, error) { return s.ProfileStop(context.Background(), "proc") }, []string{"profile", "stop", "proc"}},
		{"dumpheap", func(s *ActivityManagerService) (int, error) {
			return s.DumpHeap(context.Background(), "proc", "/tmp/heap.hprof")
		}, []string{"dumpheap", "proc", "/tmp/heap.hprof"}},
		{"set debug app", func(s *ActivityManagerService) (int, error) { return s.SetDebugApp(context.Background(), "-w", "pkg") }, []string{"set-debug-app", "-w", "pkg"}},
		{"clear debug app", func(s *ActivityManagerService) (int, error) { return s.ClearDebugApp(context.Background()) }, []string{"clear-debug-app"}},
		{"set watch heap", func(s *ActivityManagerService) (int, error) {
			return s.SetWatchHeap(context.Background(), "proc", "64m")
		}, []string{"set-watch-heap", "proc", "64m"}},
		{"clear watch heap", func(s *ActivityManagerService) (int, error) { return s.ClearWatchHeap(context.Background()) }, []string{"clear-watch-heap"}},
		{"clear start info", func(s *ActivityManagerService) (int, error) {
			return s.ClearStartInfo(context.Background(), "--user", "all", "pkg")
		}, []string{"clear-start-info", "--user", "all", "pkg"}},
		{"start info detailed monitoring", func(s *ActivityManagerService) (int, error) {
			return s.StartInfoDetailedMonitoring(context.Background(), "pkg")
		}, []string{"start-info-detailed-monitoring", "pkg"}},
		{"clear exit info", func(s *ActivityManagerService) (int, error) { return s.ClearExitInfo(context.Background(), "pkg") }, []string{"clear-exit-info", "pkg"}},
		{"bug report", func(s *ActivityManagerService) (int, error) { return s.BugReport(context.Background(), "--progress") }, []string{"bug-report", "--progress"}},
		{"fgs notification rate limit", func(s *ActivityManagerService) (int, error) {
			return s.FGSNotificationRateLimit(context.Background(), "enable")
		}, []string{"fgs-notification-rate-limit", "enable"}},
		{"force stop", func(s *ActivityManagerService) (int, error) {
			return s.ForceStop(context.Background(), "--user", "current", "pkg")
		}, []string{"force-stop", "--user", "current", "pkg"}},
		{"stop app", func(s *ActivityManagerService) (int, error) { return s.StopApp(context.Background(), "pkg") }, []string{"stop-app", "pkg"}},
		{"crash", func(s *ActivityManagerService) (int, error) {
			return s.Crash(context.Background(), "--user", "0", "pkg")
		}, []string{"crash", "--user", "0", "pkg"}},
		{"kill", func(s *ActivityManagerService) (int, error) { return s.Kill(context.Background(), "pkg") }, []string{"kill", "pkg"}},
		{"kill all", func(s *ActivityManagerService) (int, error) { return s.KillAll(context.Background()) }, []string{"kill-all"}},
		{"make uid idle", func(s *ActivityManagerService) (int, error) { return s.MakeUIDIdle(context.Background(), "pkg") }, []string{"make-uid-idle", "pkg"}},
		{"set deterministic uid idle", func(s *ActivityManagerService) (int, error) {
			return s.SetDeterministicUIDIdle(context.Background(), "--user", "all", "true")
		}, []string{"set-deterministic-uid-idle", "--user", "all", "true"}},
		{"monitor", func(s *ActivityManagerService) (int, error) { return s.Monitor(context.Background(), "-s") }, []string{"monitor", "-s"}},
		{"watch uids", func(s *ActivityManagerService) (int, error) {
			return s.WatchUIDs(context.Background(), "--oom", "1000")
		}, []string{"watch-uids", "--oom", "1000"}},
		{"hang", func(s *ActivityManagerService) (int, error) { return s.Hang(context.Background(), "--allow-restart") }, []string{"hang", "--allow-restart"}},
		{"restart", func(s *ActivityManagerService) (int, error) { return s.Restart(context.Background()) }, []string{"restart"}},
		{"idle maintenance", func(s *ActivityManagerService) (int, error) { return s.IdleMaintenance(context.Background()) }, []string{"idle-maintenance"}},
		{"screen compat", func(s *ActivityManagerService) (int, error) { return s.ScreenCompat(context.Background(), "on", "pkg") }, []string{"screen-compat", "on", "pkg"}},
		{"package importance", func(s *ActivityManagerService) (int, error) { return s.PackageImportance(context.Background(), "pkg") }, []string{"package-importance", "pkg"}},
		{"to uri", func(s *ActivityManagerService) (int, error) { return s.ToURI(context.Background(), "intent://demo") }, []string{"to-uri", "intent://demo"}},
		{"to intent uri", func(s *ActivityManagerService) (int, error) {
			return s.ToIntentURI(context.Background(), "intent://demo")
		}, []string{"to-intent-uri", "intent://demo"}},
		{"to app uri", func(s *ActivityManagerService) (int, error) { return s.ToAppURI(context.Background(), "intent://demo") }, []string{"to-app-uri", "intent://demo"}},
		{"switch user", func(s *ActivityManagerService) (int, error) { return s.SwitchUser(context.Background(), "10") }, []string{"switch-user", "10"}},
		{"get current user", func(s *ActivityManagerService) (int, error) { return s.GetCurrentUser(context.Background()) }, []string{"get-current-user"}},
		{"start user", func(s *ActivityManagerService) (int, error) { return s.StartUser(context.Background(), "-w", "10") }, []string{"start-user", "-w", "10"}},
		{"unlock user", func(s *ActivityManagerService) (int, error) { return s.UnlockUser(context.Background(), "10") }, []string{"unlock-user", "10"}},
		{"stop user", func(s *ActivityManagerService) (int, error) { return s.StopUser(context.Background(), "-w", "10") }, []string{"stop-user", "-w", "10"}},
		{"is user stopped", func(s *ActivityManagerService) (int, error) { return s.IsUserStopped(context.Background(), "10") }, []string{"is-user-stopped", "10"}},
		{"get started user state", func(s *ActivityManagerService) (int, error) { return s.GetStartedUserState(context.Background(), "10") }, []string{"get-started-user-state", "10"}},
		{"track associations", func(s *ActivityManagerService) (int, error) { return s.TrackAssociations(context.Background()) }, []string{"track-associations"}},
		{"untrack associations", func(s *ActivityManagerService) (int, error) { return s.UntrackAssociations(context.Background()) }, []string{"untrack-associations"}},
		{"get uid state", func(s *ActivityManagerService) (int, error) { return s.GetUIDState(context.Background(), "1000") }, []string{"get-uid-state", "1000"}},
		{"attach agent", func(s *ActivityManagerService) (int, error) {
			return s.AttachAgent(context.Background(), "proc", "/tmp/agent.so")
		}, []string{"attach-agent", "proc", "/tmp/agent.so"}},
		{"get config", func(s *ActivityManagerService) (int, error) { return s.GetConfig(context.Background(), "--proto") }, []string{"get-config", "--proto"}},
		{"supports multiwindow", func(s *ActivityManagerService) (int, error) { return s.SupportsMultiwindow(context.Background()) }, []string{"supports-multiwindow"}},
		{"supports split screen multi window", func(s *ActivityManagerService) (int, error) {
			return s.SupportsSplitScreenMultiWindow(context.Background())
		}, []string{"supports-split-screen-multi-window"}},
		{"suppress resize config changes", func(s *ActivityManagerService) (int, error) {
			return s.SuppressResizeConfigChanges(context.Background(), "true")
		}, []string{"suppress-resize-config-changes", "true"}},
		{"set inactive", func(s *ActivityManagerService) (int, error) {
			return s.SetInactive(context.Background(), "--user", "0", "pkg", "true")
		}, []string{"set-inactive", "--user", "0", "pkg", "true"}},
		{"get inactive", func(s *ActivityManagerService) (int, error) {
			return s.GetInactive(context.Background(), "--user", "0", "pkg")
		}, []string{"get-inactive", "--user", "0", "pkg"}},
		{"set standby bucket", func(s *ActivityManagerService) (int, error) {
			return s.SetStandbyBucket(context.Background(), "--user", "0", "pkg", "restricted")
		}, []string{"set-standby-bucket", "--user", "0", "pkg", "restricted"}},
		{"get standby bucket", func(s *ActivityManagerService) (int, error) {
			return s.GetStandbyBucket(context.Background(), "--user", "0", "pkg")
		}, []string{"get-standby-bucket", "--user", "0", "pkg"}},
		{"send trim memory", func(s *ActivityManagerService) (int, error) {
			return s.SendTrimMemory(context.Background(), "proc", "RUNNING_LOW")
		}, []string{"send-trim-memory", "proc", "RUNNING_LOW"}},
		{"update appinfo", func(s *ActivityManagerService) (int, error) {
			return s.UpdateAppInfo(context.Background(), "0", "pkg.one", "pkg.two")
		}, []string{"update-appinfo", "0", "pkg.one", "pkg.two"}},
		{"write", func(s *ActivityManagerService) (int, error) { return s.Write(context.Background()) }, []string{"write"}},
		{"get isolated pids", func(s *ActivityManagerService) (int, error) { return s.GetIsolatedPIDs(context.Background(), "1000") }, []string{"get-isolated-pids", "1000"}},
		{"set stop user on switch", func(s *ActivityManagerService) (int, error) {
			return s.SetStopUserOnSwitch(context.Background(), "true")
		}, []string{"set-stop-user-on-switch", "true"}},
		{"set bg abusive uids", func(s *ActivityManagerService) (int, error) {
			return s.SetBGAbusiveUIDs(context.Background(), "1000=10")
		}, []string{"set-bg-abusive-uids", "1000=10"}},
		{"set bg restriction level", func(s *ActivityManagerService) (int, error) {
			return s.SetBGRestrictionLevel(context.Background(), "--user", "0", "pkg", "restricted_bucket")
		}, []string{"set-bg-restriction-level", "--user", "0", "pkg", "restricted_bucket"}},
		{"get bg restriction level", func(s *ActivityManagerService) (int, error) {
			return s.GetBGRestrictionLevel(context.Background(), "--user", "0", "pkg")
		}, []string{"get-bg-restriction-level", "--user", "0", "pkg"}},
		{"list displays for starting users", func(s *ActivityManagerService) (int, error) {
			return s.ListDisplaysForStartingUsers(context.Background())
		}, []string{"list-displays-for-starting-users"}},
		{"set foreground service delegate", func(s *ActivityManagerService) (int, error) {
			return s.SetForegroundServiceDelegate(context.Background(), "--user", "0", "pkg", "start")
		}, []string{"set-foreground-service-delegate", "--user", "0", "pkg", "start"}},
		{"set ignore delivery group policy", func(s *ActivityManagerService) (int, error) {
			return s.SetIgnoreDeliveryGroupPolicy(context.Background(), "android.intent.action.TEST")
		}, []string{"set-ignore-delivery-group-policy", "android.intent.action.TEST"}},
		{"clear ignore delivery group policy", func(s *ActivityManagerService) (int, error) {
			return s.ClearIgnoreDeliveryGroupPolicy(context.Background(), "android.intent.action.TEST")
		}, []string{"clear-ignore-delivery-group-policy", "android.intent.action.TEST"}},
		{"capabilities", func(s *ActivityManagerService) (int, error) {
			return s.Capabilities(context.Background(), "--protobuf")
		}, []string{"capabilities", "--protobuf"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureActivityShellCommand(t, tt.call)
			assertStringSlice(t, got.Args, tt.want)
		})
	}
}

func TestActivityManagerServiceGroupedCommands(t *testing.T) {
	tests := []struct {
		name string
		call func(*ActivityManagerService) (int, error)
		want []string
	}{
		{"display raw", func(s *ActivityManagerService) (int, error) {
			return s.Display(context.Background(), "move-stack", "1", "2")
		}, []string{"display", "move-stack", "1", "2"}},
		{"display move stack", func(s *ActivityManagerService) (int, error) {
			return s.DisplayMoveStack(context.Background(), "1", "2")
		}, []string{"display", "move-stack", "1", "2"}},
		{"stack raw", func(s *ActivityManagerService) (int, error) { return s.Stack(context.Background(), "list") }, []string{"stack", "list"}},
		{"stack move task", func(s *ActivityManagerService) (int, error) {
			return s.StackMoveTask(context.Background(), "10", "20", true)
		}, []string{"stack", "move-task", "10", "20", "true"}},
		{"stack list", func(s *ActivityManagerService) (int, error) { return s.StackList(context.Background()) }, []string{"stack", "list"}},
		{"stack info", func(s *ActivityManagerService) (int, error) { return s.StackInfo(context.Background(), "1", "2") }, []string{"stack", "info", "1", "2"}},
		{"stack remove", func(s *ActivityManagerService) (int, error) { return s.StackRemove(context.Background(), "30") }, []string{"stack", "remove", "30"}},
		{"task raw", func(s *ActivityManagerService) (int, error) { return s.Task(context.Background(), "lock", "40") }, []string{"task", "lock", "40"}},
		{"task lock", func(s *ActivityManagerService) (int, error) { return s.TaskLock(context.Background(), "40") }, []string{"task", "lock", "40"}},
		{"task lock stop", func(s *ActivityManagerService) (int, error) { return s.TaskLockStop(context.Background()) }, []string{"task", "lock", "stop"}},
		{"task resizeable", func(s *ActivityManagerService) (int, error) { return s.TaskResizeable(context.Background(), "40", "2") }, []string{"task", "resizeable", "40", "2"}},
		{"task resize", func(s *ActivityManagerService) (int, error) {
			return s.TaskResize(context.Background(), "40", "0", "0", "100", "200")
		}, []string{"task", "resize", "40", "0", "0", "100", "200"}},
		{"compat raw", func(s *ActivityManagerService) (int, error) {
			return s.Compat(context.Background(), "enable", "--no-kill", "123", "pkg")
		}, []string{"compat", "enable", "--no-kill", "123", "pkg"}},
		{"compat enable", func(s *ActivityManagerService) (int, error) {
			return s.CompatEnable(context.Background(), "--no-kill", "123", "pkg")
		}, []string{"compat", "enable", "--no-kill", "123", "pkg"}},
		{"compat disable", func(s *ActivityManagerService) (int, error) {
			return s.CompatDisable(context.Background(), "123", "pkg")
		}, []string{"compat", "disable", "123", "pkg"}},
		{"compat reset", func(s *ActivityManagerService) (int, error) { return s.CompatReset(context.Background(), "123", "pkg") }, []string{"compat", "reset", "123", "pkg"}},
		{"compat enable all", func(s *ActivityManagerService) (int, error) {
			return s.CompatEnableAll(context.Background(), "34", "pkg")
		}, []string{"compat", "enable-all", "34", "pkg"}},
		{"compat disable all", func(s *ActivityManagerService) (int, error) {
			return s.CompatDisableAll(context.Background(), "34", "pkg")
		}, []string{"compat", "disable-all", "34", "pkg"}},
		{"compat reset all", func(s *ActivityManagerService) (int, error) {
			return s.CompatResetAll(context.Background(), "--no-kill", "pkg")
		}, []string{"compat", "reset-all", "--no-kill", "pkg"}},
		{"memory factor raw", func(s *ActivityManagerService) (int, error) { return s.MemoryFactor(context.Background(), "show") }, []string{"memory-factor", "show"}},
		{"memory factor set", func(s *ActivityManagerService) (int, error) { return s.MemoryFactorSet(context.Background(), "LOW") }, []string{"memory-factor", "set", "LOW"}},
		{"memory factor show", func(s *ActivityManagerService) (int, error) { return s.MemoryFactorShow(context.Background()) }, []string{"memory-factor", "show"}},
		{"memory factor reset", func(s *ActivityManagerService) (int, error) { return s.MemoryFactorReset(context.Background()) }, []string{"memory-factor", "reset"}},
		{"service restart backoff raw", func(s *ActivityManagerService) (int, error) {
			return s.ServiceRestartBackoff(context.Background(), "show", "pkg")
		}, []string{"service-restart-backoff", "show", "pkg"}},
		{"service restart backoff enable", func(s *ActivityManagerService) (int, error) {
			return s.ServiceRestartBackoffEnable(context.Background(), "pkg")
		}, []string{"service-restart-backoff", "enable", "pkg"}},
		{"service restart backoff disable", func(s *ActivityManagerService) (int, error) {
			return s.ServiceRestartBackoffDisable(context.Background(), "pkg")
		}, []string{"service-restart-backoff", "disable", "pkg"}},
		{"service restart backoff show", func(s *ActivityManagerService) (int, error) {
			return s.ServiceRestartBackoffShow(context.Background(), "pkg")
		}, []string{"service-restart-backoff", "show", "pkg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureActivityShellCommand(t, tt.call)
			assertStringSlice(t, got.Args, tt.want)
		})
	}
}

func TestActivityManagerServiceExecuteCommand(t *testing.T) {
	got := captureActivityShellCommand(t, func(s *ActivityManagerService) (int, error) {
		return s.ExecuteCommand(context.Background(), []string{"start-activity", "-W", "intent://demo"})
	})
	assertStringSlice(t, got.Args, []string{"start-activity", "-W", "intent://demo"})
}

func captureActivityShellCommand(t *testing.T, call func(*ActivityManagerService) (int, error)) shellCommandRequest {
	t.Helper()
	registry := newShellTestBinderRegistry()
	var (
		called bool
		got    shellCommandRequest
	)
	service := &shellTestService{
		registry: registry,
		onShellCommand: func(ctx context.Context, req shellCommandRequest) error {
			called = true
			got = req
			return NewResultReceiverProxy(req.ResultReceiver).Send(ctx, 0)
		},
	}

	code, err := call(NewActivityManagerService(service))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !called {
		t.Fatal("shell command transact was not invoked")
	}
	return got
}
