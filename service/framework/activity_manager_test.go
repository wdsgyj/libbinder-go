package framework

import (
	"reflect"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestActivityManagerMemoryInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	value := ActivityManagerMemoryInfo{
		AdvertisedMem:            1,
		AvailMem:                 2,
		TotalMem:                 3,
		Threshold:                4,
		LowMemory:                true,
		HiddenAppThreshold:       5,
		SecondaryServerThreshold: 6,
		VisibleAppThreshold:      7,
		ForegroundAppThreshold:   8,
	}
	if err := WriteActivityManagerMemoryInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteActivityManagerMemoryInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadActivityManagerMemoryInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadActivityManagerMemoryInfoFromParcel: %v", err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}

func TestActivityManagerPendingIntentInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	creatorPackage := "pkg"
	value := ActivityManagerPendingIntentInfo{
		CreatorPackage:   &creatorPackage,
		CreatorUID:       1000,
		Immutable:        true,
		IntentSenderType: 3,
	}
	if err := WriteActivityManagerPendingIntentInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteActivityManagerPendingIntentInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadActivityManagerPendingIntentInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadActivityManagerPendingIntentInfoFromParcel: %v", err)
	}
	if got.CreatorPackage == nil || *got.CreatorPackage != creatorPackage {
		t.Fatalf("got.CreatorPackage = %#v, want %q", got.CreatorPackage, creatorPackage)
	}
	got.CreatorPackage = nil
	value.CreatorPackage = nil
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}

func TestActivityManagerProcessErrorStateInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	processName := "proc"
	tag := "tag"
	shortMsg := "short"
	longMsg := "long"
	stackTrace := "stack"
	value := ActivityManagerProcessErrorStateInfo{
		Condition:   2,
		ProcessName: &processName,
		PID:         123,
		UID:         456,
		Tag:         &tag,
		ShortMsg:    &shortMsg,
		LongMsg:     &longMsg,
		StackTrace:  &stackTrace,
	}
	if err := WriteActivityManagerProcessErrorStateInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteActivityManagerProcessErrorStateInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadActivityManagerProcessErrorStateInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadActivityManagerProcessErrorStateInfoFromParcel: %v", err)
	}
	if got.Condition != value.Condition || got.PID != value.PID || got.UID != value.UID {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
	if got.ProcessName == nil || *got.ProcessName != processName {
		t.Fatalf("got.ProcessName = %#v, want %q", got.ProcessName, processName)
	}
	if got.Tag == nil || *got.Tag != tag || got.ShortMsg == nil || *got.ShortMsg != shortMsg ||
		got.LongMsg == nil || *got.LongMsg != longMsg || got.StackTrace == nil ||
		*got.StackTrace != stackTrace {
		t.Fatalf("got = %#v, want all nullable strings restored", got)
	}
}

func TestActivityManagerRunningServiceInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	process := "system_server"
	clientPackage := "android"
	value := ActivityManagerRunningServiceInfo{
		Service:          &ComponentName{Package: "android", Class: ".ExampleService"},
		PID:              10,
		UID:              11,
		Process:          &process,
		Foreground:       true,
		ActiveSince:      12,
		Started:          true,
		ClientCount:      13,
		CrashCount:       14,
		LastActivityTime: 15,
		Restarting:       16,
		Flags:            17,
		ClientPackage:    &clientPackage,
		ClientLabel:      18,
	}
	if err := WriteActivityManagerRunningServiceInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteActivityManagerRunningServiceInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadActivityManagerRunningServiceInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadActivityManagerRunningServiceInfoFromParcel: %v", err)
	}
	if got.Service == nil || *got.Service != *value.Service {
		t.Fatalf("got.Service = %#v, want %#v", got.Service, value.Service)
	}
	if got.Process == nil || *got.Process != process {
		t.Fatalf("got.Process = %#v, want %q", got.Process, process)
	}
	if got.ClientPackage == nil || *got.ClientPackage != clientPackage {
		t.Fatalf("got.ClientPackage = %#v, want %q", got.ClientPackage, clientPackage)
	}
	got.Service = nil
	got.Process = nil
	got.ClientPackage = nil
	value.Service = nil
	value.Process = nil
	value.ClientPackage = nil
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}

func TestActivityManagerRunningAppProcessInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	processName := "com.example"
	value := ActivityManagerRunningAppProcessInfo{
		ProcessName:                &processName,
		PID:                        20,
		UID:                        21,
		PkgList:                    []string{"a", "b"},
		PkgDeps:                    []string{"c"},
		Flags:                      22,
		LastTrimLevel:              23,
		Importance:                 24,
		LRU:                        25,
		ImportanceReasonCode:       26,
		ImportanceReasonPID:        27,
		ImportanceReasonComponent:  &ComponentName{Package: "android", Class: ".Main"},
		ImportanceReasonImportance: 28,
		ProcessState:               29,
		IsFocused:                  true,
		LastActivityTime:           30,
	}
	if err := WriteActivityManagerRunningAppProcessInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteActivityManagerRunningAppProcessInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadActivityManagerRunningAppProcessInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadActivityManagerRunningAppProcessInfoFromParcel: %v", err)
	}
	if got.ProcessName == nil || *got.ProcessName != processName {
		t.Fatalf("got.ProcessName = %#v, want %q", got.ProcessName, processName)
	}
	if got.ImportanceReasonComponent == nil || *got.ImportanceReasonComponent != *value.ImportanceReasonComponent {
		t.Fatalf("got.ImportanceReasonComponent = %#v, want %#v", got.ImportanceReasonComponent, value.ImportanceReasonComponent)
	}
	got.ProcessName = nil
	got.ImportanceReasonComponent = nil
	value.ProcessName = nil
	value.ImportanceReasonComponent = nil
	if len(got.PkgList) != 2 || got.PkgList[0] != "a" || len(got.PkgDeps) != 1 || got.PkgDeps[0] != "c" {
		t.Fatalf("got arrays = %#v %#v, want pkg arrays restored", got.PkgList, got.PkgDeps)
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}

func TestActivityManagerTaskPlaceholderRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		write func(*api.Parcel, OpaqueParcelable) error
		read  func(*api.Parcel) (OpaqueParcelable, error)
	}{
		{
			name:  "task_description",
			write: WriteActivityManagerTaskDescriptionToParcel,
			read:  ReadActivityManagerTaskDescriptionFromParcel,
		},
		{
			name:  "recent_task_info",
			write: WriteActivityManagerRecentTaskInfoToParcel,
			read:  ReadActivityManagerRecentTaskInfoFromParcel,
		},
		{
			name:  "running_task_info",
			write: WriteActivityManagerRunningTaskInfoToParcel,
			read:  ReadActivityManagerRunningTaskInfoFromParcel,
		},
		{
			name:  "task_thumbnail",
			write: WriteActivityManagerTaskThumbnailToParcel,
			read:  ReadActivityManagerTaskThumbnailFromParcel,
		},
		{
			name:  "root_task_info",
			write: WriteActivityTaskManagerRootTaskInfoToParcel,
			read:  ReadActivityTaskManagerRootTaskInfoFromParcel,
		},
	}
	for i, tc := range cases {
		p := api.NewParcel()
		want := NewOpaqueParcelable([]byte{0x55, byte(i), 0xaa})
		if err := tc.write(p, want); err != nil {
			t.Fatalf("%s write error = %v", tc.name, err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("%s SetPosition: %v", tc.name, err)
		}
		got, err := tc.read(p)
		if err != nil {
			t.Fatalf("%s read error = %v", tc.name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s got = %#v, want %#v", tc.name, got, want)
		}
	}
}
