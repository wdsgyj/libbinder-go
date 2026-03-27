package service

import (
	"fmt"
	"strconv"
	"strings"

	libbinder "github.com/wdsgyj/libbinder-go"
)

type Intent struct {
	Action     *string
	DataURI    *string
	MIMEType   *string
	Identifier *string
	Categories []string
	Component  *string
	Flags      *int64

	GrantReadURIPermission        bool
	GrantWriteURIPermission       bool
	GrantPersistableURIPermission bool
	GrantPrefixURIPermission      bool
	DebugLogResolution            bool
	ExcludeStoppedPackages        bool
	IncludeStoppedPackages        bool
	ActivityBroughtToFront        bool
	ActivityClearTop              bool
	ActivityClearWhenTaskReset    bool
	ActivityExcludeFromRecents    bool
	ActivityLaunchedFromHistory   bool
	ActivityMultipleTask          bool
	ActivityNoAnimation           bool
	ActivityNoHistory             bool
	ActivityNoUserAction          bool
	ActivityPreviousIsTop         bool
	ActivityReorderToFront        bool
	ActivityResetTaskIfNeeded     bool
	ActivitySingleTop             bool
	ActivityClearTask             bool
	ActivityTaskOnHome            bool
	ActivityMatchExternal         bool
	ReceiverRegisteredOnly        bool
	ReceiverReplacePending        bool
	ReceiverForeground            bool
	ReceiverNoAbort               bool
	ReceiverIncludeBackground     bool

	Selector *Intent
	Extras   []IntentExtra
	RawArgs  []string

	TargetURI       *string
	TargetPackage   *string
	TargetComponent *string
}

func (i Intent) Args() ([]string, error) {
	return i.AppendArgs(nil)
}

func (i Intent) AppendArgs(dst []string) ([]string, error) {
	args := append([]string(nil), dst...)

	if i.Action != nil {
		args = append(args, "-a", *i.Action)
	}
	if i.DataURI != nil {
		args = append(args, "-d", *i.DataURI)
	}
	if i.MIMEType != nil {
		args = append(args, "-t", *i.MIMEType)
	}
	if i.Identifier != nil {
		args = append(args, "-i", *i.Identifier)
	}
	for _, category := range i.Categories {
		args = append(args, "-c", category)
	}
	if i.Component != nil {
		args = append(args, "-n", *i.Component)
	}
	for _, extra := range i.Extras {
		args = append(args, extra.Args()...)
	}
	if i.Flags != nil {
		args = append(args, "-f", fmt.Sprintf("%#x", *i.Flags))
	}

	args = appendIntentFlag(args, i.GrantReadURIPermission, "--grant-read-uri-permission")
	args = appendIntentFlag(args, i.GrantWriteURIPermission, "--grant-write-uri-permission")
	args = appendIntentFlag(args, i.GrantPersistableURIPermission, "--grant-persistable-uri-permission")
	args = appendIntentFlag(args, i.GrantPrefixURIPermission, "--grant-prefix-uri-permission")
	args = appendIntentFlag(args, i.DebugLogResolution, "--debug-log-resolution")
	args = appendIntentFlag(args, i.ExcludeStoppedPackages, "--exclude-stopped-packages")
	args = appendIntentFlag(args, i.IncludeStoppedPackages, "--include-stopped-packages")
	args = appendIntentFlag(args, i.ActivityBroughtToFront, "--activity-brought-to-front")
	args = appendIntentFlag(args, i.ActivityClearTop, "--activity-clear-top")
	args = appendIntentFlag(args, i.ActivityClearWhenTaskReset, "--activity-clear-when-task-reset")
	args = appendIntentFlag(args, i.ActivityExcludeFromRecents, "--activity-exclude-from-recents")
	args = appendIntentFlag(args, i.ActivityLaunchedFromHistory, "--activity-launched-from-history")
	args = appendIntentFlag(args, i.ActivityMultipleTask, "--activity-multiple-task")
	args = appendIntentFlag(args, i.ActivityNoAnimation, "--activity-no-animation")
	args = appendIntentFlag(args, i.ActivityNoHistory, "--activity-no-history")
	args = appendIntentFlag(args, i.ActivityNoUserAction, "--activity-no-user-action")
	args = appendIntentFlag(args, i.ActivityPreviousIsTop, "--activity-previous-is-top")
	args = appendIntentFlag(args, i.ActivityReorderToFront, "--activity-reorder-to-front")
	args = appendIntentFlag(args, i.ActivityResetTaskIfNeeded, "--activity-reset-task-if-needed")
	args = appendIntentFlag(args, i.ActivitySingleTop, "--activity-single-top")
	args = appendIntentFlag(args, i.ActivityClearTask, "--activity-clear-task")
	args = appendIntentFlag(args, i.ActivityTaskOnHome, "--activity-task-on-home")
	args = appendIntentFlag(args, i.ActivityMatchExternal, "--activity-match-external")
	args = appendIntentFlag(args, i.ReceiverRegisteredOnly, "--receiver-registered-only")
	args = appendIntentFlag(args, i.ReceiverReplacePending, "--receiver-replace-pending")
	args = appendIntentFlag(args, i.ReceiverForeground, "--receiver-foreground")
	args = appendIntentFlag(args, i.ReceiverNoAbort, "--receiver-no-abort")
	args = appendIntentFlag(args, i.ReceiverIncludeBackground, "--receiver-include-background")

	if i.Selector != nil {
		args = append(args, "--selector")
		var err error
		args, err = i.Selector.AppendArgs(args)
		if err != nil {
			return nil, err
		}
	}
	args = append(args, i.RawArgs...)

	targetCount := 0
	if i.TargetURI != nil {
		targetCount++
	}
	if i.TargetPackage != nil {
		targetCount++
	}
	if i.TargetComponent != nil {
		targetCount++
	}
	if targetCount > 1 {
		return nil, fmt.Errorf("intent target must specify at most one of TargetURI, TargetPackage, TargetComponent")
	}
	switch {
	case i.TargetURI != nil:
		args = append(args, *i.TargetURI)
	case i.TargetPackage != nil:
		args = append(args, *i.TargetPackage)
	case i.TargetComponent != nil:
		args = append(args, *i.TargetComponent)
	}
	return args, nil
}

type IntentExtra struct {
	args []string
}

func (e IntentExtra) Args() []string {
	return append([]string(nil), e.args...)
}

func NullStringExtra(key string) IntentExtra {
	return intentExtra("--esn", key)
}

func StringExtra(key string, value string) IntentExtra {
	return intentExtra("--es", key, value)
}

func BoolExtra(key string, value bool) IntentExtra {
	return intentExtra("--ez", key, strconv.FormatBool(value))
}

func IntExtra(key string, value int) IntentExtra {
	return intentExtra("--ei", key, strconv.Itoa(value))
}

func LongExtra(key string, value int64) IntentExtra {
	return intentExtra("--el", key, strconv.FormatInt(value, 10))
}

func FloatExtra(key string, value float32) IntentExtra {
	return intentExtra("--ef", key, strconv.FormatFloat(float64(value), 'f', -1, 32))
}

func DoubleExtra(key string, value float64) IntentExtra {
	return intentExtra("--ed", key, strconv.FormatFloat(value, 'f', -1, 64))
}

func URIExtra(key string, value string) IntentExtra {
	return intentExtra("--eu", key, value)
}

func ComponentExtra(key string, value string) IntentExtra {
	return intentExtra("--ecn", key, value)
}

func IntArrayExtra(key string, values ...int) IntentExtra {
	return intentExtra("--eia", key, joinInts(values))
}

func IntArrayListExtra(key string, values ...int) IntentExtra {
	return intentExtra("--eial", key, joinInts(values))
}

func LongArrayExtra(key string, values ...int64) IntentExtra {
	return intentExtra("--ela", key, joinInt64s(values))
}

func LongArrayListExtra(key string, values ...int64) IntentExtra {
	return intentExtra("--elal", key, joinInt64s(values))
}

func FloatArrayExtra(key string, values ...float32) IntentExtra {
	return intentExtra("--efa", key, joinFloat32s(values))
}

func FloatArrayListExtra(key string, values ...float32) IntentExtra {
	return intentExtra("--efal", key, joinFloat32s(values))
}

func DoubleArrayExtra(key string, values ...float64) IntentExtra {
	return intentExtra("--eda", key, joinFloat64s(values))
}

func DoubleArrayListExtra(key string, values ...float64) IntentExtra {
	return intentExtra("--edal", key, joinFloat64s(values))
}

func StringArrayExtra(key string, values ...string) IntentExtra {
	return intentExtra("--esa", key, joinEscapedStrings(values))
}

func StringArrayListExtra(key string, values ...string) IntentExtra {
	return intentExtra("--esal", key, joinEscapedStrings(values))
}

func appendIntentFlag(args []string, enabled bool, flag string) []string {
	if enabled {
		return append(args, flag)
	}
	return args
}

func intentExtra(option string, args ...string) IntentExtra {
	tokens := make([]string, 0, 1+len(args))
	tokens = append(tokens, option)
	tokens = append(tokens, args...)
	return IntentExtra{args: tokens}
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.Itoa(value))
	}
	return strings.Join(out, ",")
}

func joinInt64s(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.FormatInt(value, 10))
	}
	return strings.Join(out, ",")
}

func joinFloat32s(values []float32) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.FormatFloat(float64(value), 'f', -1, 32))
	}
	return strings.Join(out, ",")
}

func joinFloat64s(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.FormatFloat(value, 'f', -1, 64))
	}
	return strings.Join(out, ",")
}

func joinEscapedStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, ",", "\\,")
		out = append(out, value)
	}
	return strings.Join(out, ",")
}

func NewIntentWithURI(uri string) Intent {
	return Intent{TargetURI: libbinder.StringPtr(uri)}
}

func NewIntentWithPackage(pkg string) Intent {
	return Intent{TargetPackage: libbinder.StringPtr(pkg)}
}

func NewIntentWithComponent(component string) Intent {
	return Intent{TargetComponent: libbinder.StringPtr(component)}
}
