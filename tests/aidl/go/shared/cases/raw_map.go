package cases

import (
	"context"
	"fmt"
	"reflect"

	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func RawMapInput() map[any]any {
	return map[any]any{
		"name":   "seed",
		"count":  int32(7),
		"active": true,
		"tags":   []any{"red", int32(2), int64(3)},
		"meta": map[any]any{
			"mode":    "alpha",
			"enabled": true,
			"ids":     []any{int32(1), "two"},
		},
	}
}

func NormalizeRawMap(prefix string, value map[any]any) map[any]any {
	return normalizeRawMapValue(prefix, value).(map[any]any)
}

func VerifyRawMapService(ctx context.Context, svc shared.IRawMapService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	input := RawMapInput()
	got, err := svc.Normalize(ctx, input)
	if err != nil {
		return fmt.Errorf("Normalize: %w", err)
	}
	want := NormalizeRawMap(prefix, input)
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("Normalize = %#v, want %#v", got, want)
	}
	return nil
}

func normalizeRawMapValue(prefix string, value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return prefix + ":" + v
	case int32:
		return v + 1
	case int64:
		return v + 1
	case bool:
		return v
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, normalizeRawMapValue(prefix, item))
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(v))
		for key, item := range v {
			out[key] = normalizeRawMapValue(prefix, item)
		}
		return out
	default:
		return v
	}
}
