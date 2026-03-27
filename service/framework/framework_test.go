package framework

import (
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestComponentNameRoundTrip(t *testing.T) {
	p := api.NewParcel()
	value := ComponentName{Package: "pkg", Class: ".Main"}
	if err := WriteComponentNameToParcel(p, value); err != nil {
		t.Fatalf("WriteComponentNameToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadComponentNameFromParcel(p)
	if err != nil {
		t.Fatalf("ReadComponentNameFromParcel: %v", err)
	}
	if got != value {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}

func TestBundleValueRoundTrip(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		p := api.NewParcel()
		if err := WriteBundleValueToParcel(p, nil); err != nil {
			t.Fatalf("WriteBundleValueToParcel(nil): %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadBundleValueFromParcel(p)
		if err != nil {
			t.Fatalf("ReadBundleValueFromParcel(nil): %v", err)
		}
		if got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		p := api.NewParcel()
		if err := WriteBundleValueToParcel(p, NewEmptyBundle()); err != nil {
			t.Fatalf("WriteBundleValueToParcel(empty): %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadBundleValueFromParcel(p)
		if err != nil {
			t.Fatalf("ReadBundleValueFromParcel(empty): %v", err)
		}
		if got == nil {
			t.Fatal("got = nil, want empty bundle")
		}
		if len(got.RawData) != 0 || got.Native {
			t.Fatalf("got = %#v, want empty java bundle", got)
		}
	})

	t.Run("raw", func(t *testing.T) {
		p := api.NewParcel()
		value := NewRawBundle([]byte{1, 2, 3, 4}, true)
		if err := WriteBundleValueToParcel(p, value); err != nil {
			t.Fatalf("WriteBundleValueToParcel(raw): %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadBundleValueFromParcel(p)
		if err != nil {
			t.Fatalf("ReadBundleValueFromParcel(raw): %v", err)
		}
		if got == nil || !got.Native || string(got.RawData) != string([]byte{1, 2, 3, 4}) {
			t.Fatalf("got = %#v, want native raw bundle", got)
		}
	})
}

func TestProfilerInfoRoundTrip(t *testing.T) {
	p := api.NewParcel()
	profileFile := "trace.prof"
	agent := "agent.so"
	value := ProfilerInfo{
		ProfileFile:           &profileFile,
		SamplingInterval:      1000,
		AutoStopProfiler:      true,
		StreamingOutput:       true,
		Agent:                 &agent,
		AttachAgentDuringBind: true,
		ClockType:             0x110,
		ProfilerOutputVersion: 2,
	}
	if err := WriteProfilerInfoToParcel(p, value); err != nil {
		t.Fatalf("WriteProfilerInfoToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadProfilerInfoFromParcel(p)
	if err != nil {
		t.Fatalf("ReadProfilerInfoFromParcel: %v", err)
	}
	if got.SamplingInterval != value.SamplingInterval || got.ClockType != value.ClockType || got.ProfilerOutputVersion != value.ProfilerOutputVersion {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
	if got.ProfileFile == nil || *got.ProfileFile != profileFile {
		t.Fatalf("ProfileFile = %#v, want %q", got.ProfileFile, profileFile)
	}
	if got.Agent == nil || *got.Agent != agent {
		t.Fatalf("Agent = %#v, want %q", got.Agent, agent)
	}
	if !got.AutoStopProfiler || !got.StreamingOutput || !got.AttachAgentDuringBind {
		t.Fatalf("got booleans = %#v, want all true", got)
	}
}

func TestWaitResultRoundTrip(t *testing.T) {
	p := api.NewParcel()
	value := WaitResult{
		Result:      1,
		Timeout:     true,
		Who:         NewComponentName("pkg", ".Main"),
		TotalTime:   1234,
		LaunchState: 3,
	}
	if err := WriteWaitResultToParcel(p, value); err != nil {
		t.Fatalf("WriteWaitResultToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadWaitResultFromParcel(p)
	if err != nil {
		t.Fatalf("ReadWaitResultFromParcel: %v", err)
	}
	if got.Result != value.Result || got.TotalTime != value.TotalTime || got.LaunchState != value.LaunchState || !got.Timeout {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
	if got.Who == nil || *got.Who != *value.Who {
		t.Fatalf("Who = %#v, want %#v", got.Who, value.Who)
	}
}

func TestIntentRoundTrip(t *testing.T) {
	SetIntentRedirectProtectionEnabled(false)
	p := api.NewParcel()

	action := "android.intent.action.VIEW"
	mimeType := "text/plain"
	identifier := "demo-id"
	pkg := "demo.pkg"
	value := Intent{
		Action:        &action,
		Data:          ParseURI("content://demo/item"),
		MIMEType:      &mimeType,
		Identifier:    &identifier,
		Flags:         0x10000000,
		ExtendedFlags: 0x2,
		Package:       &pkg,
		Component:     NewComponentName("demo.pkg", ".MainActivity"),
		SourceBounds:  &Rect{Left: 1, Top: 2, Right: 3, Bottom: 4},
		Categories:    []string{"android.intent.category.DEFAULT", "demo.category"},
		Selector: &Intent{
			Action: &action,
			Data:   ParseURI("content://selector"),
		},
		ContentUserHint: 10,
		Extras:          NewEmptyBundle(),
		OriginalIntent: &Intent{
			Package: &pkg,
		},
	}
	if err := WriteIntentToParcel(p, value); err != nil {
		t.Fatalf("WriteIntentToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadIntentFromParcel(p)
	if err != nil {
		t.Fatalf("ReadIntentFromParcel: %v", err)
	}
	if got.Action == nil || *got.Action != action {
		t.Fatalf("Action = %#v, want %q", got.Action, action)
	}
	if got.Data == nil || got.Data.Value != "content://demo/item" {
		t.Fatalf("Data = %#v, want content://demo/item", got.Data)
	}
	if got.Component == nil || got.Component.FlattenToShortString() != "demo.pkg/.MainActivity" {
		t.Fatalf("Component = %#v", got.Component)
	}
	if got.Selector == nil || got.Selector.Data == nil || got.Selector.Data.Value != "content://selector" {
		t.Fatalf("Selector = %#v", got.Selector)
	}
	if got.Extras == nil || len(got.Extras.RawData) != 0 {
		t.Fatalf("Extras = %#v, want empty bundle", got.Extras)
	}
	if got.OriginalIntent == nil || got.OriginalIntent.Package == nil || *got.OriginalIntent.Package != pkg {
		t.Fatalf("OriginalIntent = %#v", got.OriginalIntent)
	}
	if len(got.Categories) != 2 || got.Categories[1] != "demo.category" {
		t.Fatalf("Categories = %#v", got.Categories)
	}
}
