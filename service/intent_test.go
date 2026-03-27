package service

import (
	"context"
	"testing"

	libbinder "github.com/wdsgyj/libbinder-go"
)

func TestIntentArgs(t *testing.T) {
	intent := Intent{
		Action:     libbinder.StringPtr("android.intent.action.VIEW"),
		DataURI:    libbinder.StringPtr("content://demo/item"),
		MIMEType:   libbinder.StringPtr("text/plain"),
		Identifier: libbinder.StringPtr("demo-id"),
		Categories: []string{"android.intent.category.DEFAULT", "demo.category"},
		Component:  libbinder.StringPtr("demo/.MainActivity"),
		Flags:      libbinder.Int64Ptr(0x10000000),

		GrantReadURIPermission:    true,
		ActivityClearTop:          true,
		ReceiverForeground:        true,
		ReceiverIncludeBackground: true,
		Extras: []IntentExtra{
			StringExtra("s", "value"),
			NullStringExtra("sn"),
			BoolExtra("b", true),
			IntExtra("i", 1),
			LongExtra("l", 2),
			FloatExtra("f", 1.5),
			DoubleExtra("d", 2.5),
			URIExtra("u", "content://demo/extra"),
			ComponentExtra("c", "demo/.Receiver"),
			IntArrayExtra("ia", 1, 2),
			IntArrayListExtra("ial", 3, 4),
			LongArrayExtra("la", 5, 6),
			LongArrayListExtra("lal", 7, 8),
			FloatArrayExtra("fa", 1.25, 2.5),
			FloatArrayListExtra("fal", 3.75, 4.5),
			DoubleArrayExtra("da", 5.25, 6.5),
			DoubleArrayListExtra("dal", 7.75, 8.5),
			StringArrayExtra("sa", "a", "b,c"),
			StringArrayListExtra("sal", "d", "e,f"),
		},
		Selector: &Intent{
			Action:    libbinder.StringPtr("android.intent.action.PICK"),
			TargetURI: libbinder.StringPtr("content://selector"),
		},
		RawArgs:   []string{"--debug-log-resolution"},
		TargetURI: libbinder.StringPtr("intent://demo"),
	}

	got, err := intent.Args()
	if err != nil {
		t.Fatalf("Args: %v", err)
	}
	want := []string{
		"-a", "android.intent.action.VIEW",
		"-d", "content://demo/item",
		"-t", "text/plain",
		"-i", "demo-id",
		"-c", "android.intent.category.DEFAULT",
		"-c", "demo.category",
		"-n", "demo/.MainActivity",
		"--es", "s", "value",
		"--esn", "sn",
		"--ez", "b", "true",
		"--ei", "i", "1",
		"--el", "l", "2",
		"--ef", "f", "1.5",
		"--ed", "d", "2.5",
		"--eu", "u", "content://demo/extra",
		"--ecn", "c", "demo/.Receiver",
		"--eia", "ia", "1,2",
		"--eial", "ial", "3,4",
		"--ela", "la", "5,6",
		"--elal", "lal", "7,8",
		"--efa", "fa", "1.25,2.5",
		"--efal", "fal", "3.75,4.5",
		"--eda", "da", "5.25,6.5",
		"--edal", "dal", "7.75,8.5",
		"--esa", "sa", "a,b\\,c",
		"--esal", "sal", "d,e\\,f",
		"-f", "0x10000000",
		"--grant-read-uri-permission",
		"--activity-clear-top",
		"--receiver-foreground",
		"--receiver-include-background",
		"--selector",
		"-a", "android.intent.action.PICK",
		"content://selector",
		"--debug-log-resolution",
		"intent://demo",
	}
	assertStringSlice(t, got, want)
}

func TestIntentArgsConflictingTargets(t *testing.T) {
	_, err := (Intent{
		TargetURI:     libbinder.StringPtr("intent://demo"),
		TargetPackage: libbinder.StringPtr("pkg"),
	}).Args()
	if err == nil || err.Error() != "intent target must specify at most one of TargetURI, TargetPackage, TargetComponent" {
		t.Fatalf("err = %v", err)
	}
}

func TestActivityManagerServiceWithIntent(t *testing.T) {
	intent := Intent{
		Action:    libbinder.StringPtr("android.intent.action.VIEW"),
		TargetURI: libbinder.StringPtr("intent://demo"),
	}

	tests := []struct {
		name string
		call func(*ActivityManagerService) (int, error)
		want []string
	}{
		{
			name: "start activity",
			call: func(s *ActivityManagerService) (int, error) {
				return s.StartActivityWithIntent(context.Background(), []string{"-W"}, intent)
			},
			want: []string{"start-activity", "-W", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "start in vsync",
			call: func(s *ActivityManagerService) (int, error) {
				return s.StartInVsyncWithIntent(context.Background(), []string{"-W"}, intent)
			},
			want: []string{"start-in-vsync", "-W", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "start service",
			call: func(s *ActivityManagerService) (int, error) {
				return s.StartServiceWithIntent(context.Background(), []string{"--user", "current"}, intent)
			},
			want: []string{"start-service", "--user", "current", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "start foreground service",
			call: func(s *ActivityManagerService) (int, error) {
				return s.StartForegroundServiceWithIntent(context.Background(), nil, intent)
			},
			want: []string{"start-foreground-service", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "stop service",
			call: func(s *ActivityManagerService) (int, error) {
				return s.StopServiceWithIntent(context.Background(), nil, intent)
			},
			want: []string{"stop-service", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "broadcast",
			call: func(s *ActivityManagerService) (int, error) {
				return s.BroadcastWithIntent(context.Background(), []string{"--async"}, intent)
			},
			want: []string{"broadcast", "--async", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "to uri",
			call: func(s *ActivityManagerService) (int, error) {
				return s.ToURIWithIntent(context.Background(), intent)
			},
			want: []string{"to-uri", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "to intent uri",
			call: func(s *ActivityManagerService) (int, error) {
				return s.ToIntentURIWithIntent(context.Background(), intent)
			},
			want: []string{"to-intent-uri", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
		{
			name: "to app uri",
			call: func(s *ActivityManagerService) (int, error) {
				return s.ToAppURIWithIntent(context.Background(), intent)
			},
			want: []string{"to-app-uri", "-a", "android.intent.action.VIEW", "intent://demo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureActivityShellCommand(t, tt.call)
			assertStringSlice(t, got.Args, tt.want)
		})
	}
}
