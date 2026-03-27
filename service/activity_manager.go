package service

import (
	"context"
	"strconv"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const ActivityServiceName = "activity"

func LookupActivityService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	if sm == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	service, err := sm.CheckService(ctx, ActivityServiceName)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, api.ErrNoService
	}
	return service, nil
}

func LookupActivityManagerService(ctx context.Context, sm api.ServiceManager) (*ActivityManagerService, error) {
	service, err := LookupActivityService(ctx, sm)
	if err != nil {
		return nil, err
	}
	return NewActivityManagerService(service), nil
}

type ActivityManagerService struct {
	shell *ShellCommandService
}

func NewActivityManagerService(service api.Binder) *ActivityManagerService {
	return &ActivityManagerService{
		shell: NewShellCommandService(ActivityServiceName, service),
	}
}

func (s *ActivityManagerService) Binder() api.Binder {
	if s == nil || s.shell == nil {
		return nil
	}
	return s.shell.Binder()
}

func (s *ActivityManagerService) WithShellIO(io ShellCommandIO) *ActivityManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.shell = s.shell.WithShellIO(io)
	return &clone
}

func (s *ActivityManagerService) WithWorkingDir(dir string) *ActivityManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.shell = s.shell.WithWorkingDir(dir)
	return &clone
}

func (s *ActivityManagerService) WithFileAccessChecker(checker FileAccessChecker) *ActivityManagerService {
	if s == nil {
		return nil
	}
	clone := *s
	clone.shell = s.shell.WithFileAccessChecker(checker)
	return &clone
}

func (s *ActivityManagerService) Command(ctx context.Context, args ...string) (int, error) {
	if s == nil || s.shell == nil {
		return 0, api.ErrUnsupported
	}
	return s.shell.Command(ctx, args...)
}

func (s *ActivityManagerService) ExecuteCommand(ctx context.Context, argv []string) (int, error) {
	if s == nil || s.shell == nil {
		return 0, api.ErrUnsupported
	}
	return s.shell.ExecuteCommand(ctx, argv)
}

func (s *ActivityManagerService) run(ctx context.Context, prefix []string, args ...string) (int, error) {
	argv := make([]string, 0, len(prefix)+len(args))
	argv = append(argv, prefix...)
	argv = append(argv, args...)
	return s.Command(ctx, argv...)
}

func (s *ActivityManagerService) runWithIntent(ctx context.Context, prefix []string, options []string, intent Intent) (int, error) {
	intentArgs, err := intent.Args()
	if err != nil {
		return 0, err
	}
	argv := make([]string, 0, len(prefix)+len(options)+len(intentArgs))
	argv = append(argv, prefix...)
	argv = append(argv, options...)
	argv = append(argv, intentArgs...)
	return s.Command(ctx, argv...)
}

func (s *ActivityManagerService) Help(ctx context.Context) (int, error) {
	return s.Command(ctx, "help")
}

func (s *ActivityManagerService) Logging(ctx context.Context, mode string, config string) (int, error) {
	return s.Command(ctx, "logging", mode, config)
}

func (s *ActivityManagerService) AppLogging(ctx context.Context, processName string, uid int, mode string, config string) (int, error) {
	return s.Command(ctx, "app-logging", processName, strconv.Itoa(uid), mode, config)
}

func (s *ActivityManagerService) StartActivity(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-activity"}, args...)
}

func (s *ActivityManagerService) StartActivityWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"start-activity"}, options, intent)
}

func (s *ActivityManagerService) StartInVsync(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-in-vsync"}, args...)
}

func (s *ActivityManagerService) StartInVsyncWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"start-in-vsync"}, options, intent)
}

func (s *ActivityManagerService) StartService(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-service"}, args...)
}

func (s *ActivityManagerService) StartServiceWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"start-service"}, options, intent)
}

func (s *ActivityManagerService) StartForegroundService(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-foreground-service"}, args...)
}

func (s *ActivityManagerService) StartForegroundServiceWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"start-foreground-service"}, options, intent)
}

func (s *ActivityManagerService) StopService(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"stop-service"}, args...)
}

func (s *ActivityManagerService) StopServiceWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"stop-service"}, options, intent)
}

func (s *ActivityManagerService) Broadcast(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"broadcast"}, args...)
}

func (s *ActivityManagerService) BroadcastWithIntent(ctx context.Context, options []string, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"broadcast"}, options, intent)
}

func (s *ActivityManagerService) CompactSome(ctx context.Context, process string) (int, error) {
	return s.Command(ctx, "compact", "some", process)
}

func (s *ActivityManagerService) CompactFull(ctx context.Context, process string) (int, error) {
	return s.Command(ctx, "compact", "full", process)
}

func (s *ActivityManagerService) CompactSystem(ctx context.Context) (int, error) {
	return s.Command(ctx, "compact", "system")
}

func (s *ActivityManagerService) CompactNativeSome(ctx context.Context, pid string) (int, error) {
	return s.Command(ctx, "compact", "native", "some", pid)
}

func (s *ActivityManagerService) CompactNativeFull(ctx context.Context, pid string) (int, error) {
	return s.Command(ctx, "compact", "native", "full", pid)
}

func (s *ActivityManagerService) Freeze(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"freeze"}, args...)
}

func (s *ActivityManagerService) Unfreeze(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"unfreeze"}, args...)
}

func (s *ActivityManagerService) Instrument(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"instrument"}, args...)
}

func (s *ActivityManagerService) TraceIPCStart(ctx context.Context) (int, error) {
	return s.Command(ctx, "trace-ipc", "start")
}

func (s *ActivityManagerService) TraceIPCStop(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"trace-ipc", "stop"}, args...)
}

func (s *ActivityManagerService) ProfileStart(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"profile", "start"}, args...)
}

func (s *ActivityManagerService) ProfileStop(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"profile", "stop"}, args...)
}

func (s *ActivityManagerService) DumpHeap(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"dumpheap"}, args...)
}

func (s *ActivityManagerService) SetDebugApp(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-debug-app"}, args...)
}

func (s *ActivityManagerService) ClearDebugApp(ctx context.Context) (int, error) {
	return s.Command(ctx, "clear-debug-app")
}

func (s *ActivityManagerService) SetWatchHeap(ctx context.Context, process string, memLimit string) (int, error) {
	return s.Command(ctx, "set-watch-heap", process, memLimit)
}

func (s *ActivityManagerService) ClearWatchHeap(ctx context.Context) (int, error) {
	return s.Command(ctx, "clear-watch-heap")
}

func (s *ActivityManagerService) ClearStartInfo(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"clear-start-info"}, args...)
}

func (s *ActivityManagerService) StartInfoDetailedMonitoring(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-info-detailed-monitoring"}, args...)
}

func (s *ActivityManagerService) ClearExitInfo(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"clear-exit-info"}, args...)
}

func (s *ActivityManagerService) BugReport(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"bug-report"}, args...)
}

func (s *ActivityManagerService) FGSNotificationRateLimit(ctx context.Context, mode string) (int, error) {
	return s.Command(ctx, "fgs-notification-rate-limit", mode)
}

func (s *ActivityManagerService) ForceStop(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"force-stop"}, args...)
}

func (s *ActivityManagerService) StopApp(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"stop-app"}, args...)
}

func (s *ActivityManagerService) Crash(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"crash"}, args...)
}

func (s *ActivityManagerService) Kill(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"kill"}, args...)
}

func (s *ActivityManagerService) KillAll(ctx context.Context) (int, error) {
	return s.Command(ctx, "kill-all")
}

func (s *ActivityManagerService) MakeUIDIdle(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"make-uid-idle"}, args...)
}

func (s *ActivityManagerService) SetDeterministicUIDIdle(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-deterministic-uid-idle"}, args...)
}

func (s *ActivityManagerService) Monitor(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"monitor"}, args...)
}

func (s *ActivityManagerService) WatchUIDs(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"watch-uids"}, args...)
}

func (s *ActivityManagerService) Hang(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"hang"}, args...)
}

func (s *ActivityManagerService) Restart(ctx context.Context) (int, error) {
	return s.Command(ctx, "restart")
}

func (s *ActivityManagerService) IdleMaintenance(ctx context.Context) (int, error) {
	return s.Command(ctx, "idle-maintenance")
}

func (s *ActivityManagerService) ScreenCompat(ctx context.Context, mode string, packageName string) (int, error) {
	return s.Command(ctx, "screen-compat", mode, packageName)
}

func (s *ActivityManagerService) PackageImportance(ctx context.Context, packageName string) (int, error) {
	return s.Command(ctx, "package-importance", packageName)
}

func (s *ActivityManagerService) ToURI(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"to-uri"}, args...)
}

func (s *ActivityManagerService) ToURIWithIntent(ctx context.Context, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"to-uri"}, nil, intent)
}

func (s *ActivityManagerService) ToIntentURI(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"to-intent-uri"}, args...)
}

func (s *ActivityManagerService) ToIntentURIWithIntent(ctx context.Context, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"to-intent-uri"}, nil, intent)
}

func (s *ActivityManagerService) ToAppURI(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"to-app-uri"}, args...)
}

func (s *ActivityManagerService) ToAppURIWithIntent(ctx context.Context, intent Intent) (int, error) {
	return s.runWithIntent(ctx, []string{"to-app-uri"}, nil, intent)
}

func (s *ActivityManagerService) SwitchUser(ctx context.Context, userID string) (int, error) {
	return s.Command(ctx, "switch-user", userID)
}

func (s *ActivityManagerService) GetCurrentUser(ctx context.Context) (int, error) {
	return s.Command(ctx, "get-current-user")
}

func (s *ActivityManagerService) StartUser(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"start-user"}, args...)
}

func (s *ActivityManagerService) UnlockUser(ctx context.Context, userID string) (int, error) {
	return s.Command(ctx, "unlock-user", userID)
}

func (s *ActivityManagerService) StopUser(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"stop-user"}, args...)
}

func (s *ActivityManagerService) IsUserStopped(ctx context.Context, userID string) (int, error) {
	return s.Command(ctx, "is-user-stopped", userID)
}

func (s *ActivityManagerService) GetStartedUserState(ctx context.Context, userID string) (int, error) {
	return s.Command(ctx, "get-started-user-state", userID)
}

func (s *ActivityManagerService) TrackAssociations(ctx context.Context) (int, error) {
	return s.Command(ctx, "track-associations")
}

func (s *ActivityManagerService) UntrackAssociations(ctx context.Context) (int, error) {
	return s.Command(ctx, "untrack-associations")
}

func (s *ActivityManagerService) GetUIDState(ctx context.Context, uid string) (int, error) {
	return s.Command(ctx, "get-uid-state", uid)
}

func (s *ActivityManagerService) AttachAgent(ctx context.Context, process string, file string) (int, error) {
	return s.Command(ctx, "attach-agent", process, file)
}

func (s *ActivityManagerService) GetConfig(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"get-config"}, args...)
}

func (s *ActivityManagerService) SupportsMultiwindow(ctx context.Context) (int, error) {
	return s.Command(ctx, "supports-multiwindow")
}

func (s *ActivityManagerService) SupportsSplitScreenMultiWindow(ctx context.Context) (int, error) {
	return s.Command(ctx, "supports-split-screen-multi-window")
}

func (s *ActivityManagerService) SuppressResizeConfigChanges(ctx context.Context, value string) (int, error) {
	return s.Command(ctx, "suppress-resize-config-changes", value)
}

func (s *ActivityManagerService) SetInactive(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-inactive"}, args...)
}

func (s *ActivityManagerService) GetInactive(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"get-inactive"}, args...)
}

func (s *ActivityManagerService) SetStandbyBucket(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-standby-bucket"}, args...)
}

func (s *ActivityManagerService) GetStandbyBucket(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"get-standby-bucket"}, args...)
}

func (s *ActivityManagerService) SendTrimMemory(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"send-trim-memory"}, args...)
}

func (s *ActivityManagerService) Display(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"display"}, args...)
}

func (s *ActivityManagerService) DisplayMoveStack(ctx context.Context, stackID string, displayID string) (int, error) {
	return s.Command(ctx, "display", "move-stack", stackID, displayID)
}

func (s *ActivityManagerService) Stack(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"stack"}, args...)
}

func (s *ActivityManagerService) StackMoveTask(ctx context.Context, taskID string, stackID string, toTop bool) (int, error) {
	return s.Command(ctx, "stack", "move-task", taskID, stackID, strconv.FormatBool(toTop))
}

func (s *ActivityManagerService) StackList(ctx context.Context) (int, error) {
	return s.Command(ctx, "stack", "list")
}

func (s *ActivityManagerService) StackInfo(ctx context.Context, windowingMode string, activityType string) (int, error) {
	return s.Command(ctx, "stack", "info", windowingMode, activityType)
}

func (s *ActivityManagerService) StackRemove(ctx context.Context, stackID string) (int, error) {
	return s.Command(ctx, "stack", "remove", stackID)
}

func (s *ActivityManagerService) Task(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"task"}, args...)
}

func (s *ActivityManagerService) TaskLock(ctx context.Context, taskID string) (int, error) {
	return s.Command(ctx, "task", "lock", taskID)
}

func (s *ActivityManagerService) TaskLockStop(ctx context.Context) (int, error) {
	return s.Command(ctx, "task", "lock", "stop")
}

func (s *ActivityManagerService) TaskResizeable(ctx context.Context, taskID string, mode string) (int, error) {
	return s.Command(ctx, "task", "resizeable", taskID, mode)
}

func (s *ActivityManagerService) TaskResize(ctx context.Context, taskID string, left string, top string, right string, bottom string) (int, error) {
	return s.Command(ctx, "task", "resize", taskID, left, top, right, bottom)
}

func (s *ActivityManagerService) UpdateAppInfo(ctx context.Context, userID string, packageNames ...string) (int, error) {
	args := make([]string, 0, 2+len(packageNames))
	args = append(args, "update-appinfo", userID)
	args = append(args, packageNames...)
	return s.Command(ctx, args...)
}

func (s *ActivityManagerService) Write(ctx context.Context) (int, error) {
	return s.Command(ctx, "write")
}

func (s *ActivityManagerService) Compat(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"compat"}, args...)
}

func (s *ActivityManagerService) CompatEnable(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"compat", "enable"}, args...)
}

func (s *ActivityManagerService) CompatDisable(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"compat", "disable"}, args...)
}

func (s *ActivityManagerService) CompatReset(ctx context.Context, changeIDOrName string, packageName string) (int, error) {
	return s.Command(ctx, "compat", "reset", changeIDOrName, packageName)
}

func (s *ActivityManagerService) CompatEnableAll(ctx context.Context, targetSDKVersion string, packageName string) (int, error) {
	return s.Command(ctx, "compat", "enable-all", targetSDKVersion, packageName)
}

func (s *ActivityManagerService) CompatDisableAll(ctx context.Context, targetSDKVersion string, packageName string) (int, error) {
	return s.Command(ctx, "compat", "disable-all", targetSDKVersion, packageName)
}

func (s *ActivityManagerService) CompatResetAll(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"compat", "reset-all"}, args...)
}

func (s *ActivityManagerService) MemoryFactor(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"memory-factor"}, args...)
}

func (s *ActivityManagerService) MemoryFactorSet(ctx context.Context, level string) (int, error) {
	return s.Command(ctx, "memory-factor", "set", level)
}

func (s *ActivityManagerService) MemoryFactorShow(ctx context.Context) (int, error) {
	return s.Command(ctx, "memory-factor", "show")
}

func (s *ActivityManagerService) MemoryFactorReset(ctx context.Context) (int, error) {
	return s.Command(ctx, "memory-factor", "reset")
}

func (s *ActivityManagerService) ServiceRestartBackoff(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"service-restart-backoff"}, args...)
}

func (s *ActivityManagerService) ServiceRestartBackoffEnable(ctx context.Context, packageName string) (int, error) {
	return s.Command(ctx, "service-restart-backoff", "enable", packageName)
}

func (s *ActivityManagerService) ServiceRestartBackoffDisable(ctx context.Context, packageName string) (int, error) {
	return s.Command(ctx, "service-restart-backoff", "disable", packageName)
}

func (s *ActivityManagerService) ServiceRestartBackoffShow(ctx context.Context, packageName string) (int, error) {
	return s.Command(ctx, "service-restart-backoff", "show", packageName)
}

func (s *ActivityManagerService) GetIsolatedPIDs(ctx context.Context, uid string) (int, error) {
	return s.Command(ctx, "get-isolated-pids", uid)
}

func (s *ActivityManagerService) SetStopUserOnSwitch(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-stop-user-on-switch"}, args...)
}

func (s *ActivityManagerService) SetBGAbusiveUIDs(ctx context.Context, spec string) (int, error) {
	return s.Command(ctx, "set-bg-abusive-uids", spec)
}

func (s *ActivityManagerService) SetBGRestrictionLevel(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-bg-restriction-level"}, args...)
}

func (s *ActivityManagerService) GetBGRestrictionLevel(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"get-bg-restriction-level"}, args...)
}

func (s *ActivityManagerService) ListDisplaysForStartingUsers(ctx context.Context) (int, error) {
	return s.Command(ctx, "list-displays-for-starting-users")
}

func (s *ActivityManagerService) SetForegroundServiceDelegate(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"set-foreground-service-delegate"}, args...)
}

func (s *ActivityManagerService) SetIgnoreDeliveryGroupPolicy(ctx context.Context, action string) (int, error) {
	return s.Command(ctx, "set-ignore-delivery-group-policy", action)
}

func (s *ActivityManagerService) ClearIgnoreDeliveryGroupPolicy(ctx context.Context, action string) (int, error) {
	return s.Command(ctx, "clear-ignore-delivery-group-policy", action)
}

func (s *ActivityManagerService) Capabilities(ctx context.Context, args ...string) (int, error) {
	return s.run(ctx, []string{"capabilities"}, args...)
}
