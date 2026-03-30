package cases

import (
	"context"
	"fmt"
	"reflect"

	customcodec "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/customcodec"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func CustomParcelableInput() customcodec.CustomBox {
	label := "seed"
	return customcodec.CustomBox{
		ID:    7,
		Label: &label,
		Tags:  []string{"red", "blue"},
		Meta:  map[string]string{"left": "west", "right": "east"},
	}
}

func NormalizeCustomParcelable(prefix string, value customcodec.CustomBox) customcodec.CustomBox {
	out := customcodec.CustomBox{
		ID:   value.ID + 1,
		Tags: make([]string, 0, len(value.Tags)),
		Meta: make(map[string]string, len(value.Meta)),
	}
	if value.Label != nil {
		label := prefix + ":" + *value.Label
		out.Label = &label
	}
	for _, tag := range value.Tags {
		out.Tags = append(out.Tags, prefix+":"+tag)
	}
	for key, item := range value.Meta {
		out.Meta[key] = prefix + ":" + item
	}
	return out
}

func VerifyCustomParcelableService(ctx context.Context, svc shared.ICustomParcelableService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	input := CustomParcelableInput()
	got, err := svc.Normalize(ctx, &input)
	if err != nil {
		return fmt.Errorf("Normalize: %w", err)
	}
	want := NormalizeCustomParcelable(prefix, input)
	if got == nil || !reflect.DeepEqual(*got, want) {
		return fmt.Errorf("Normalize = %#v, want %#v", got, want)
	}
	nilValue, err := svc.NormalizeNullable(ctx, nil)
	if err != nil {
		return fmt.Errorf("NormalizeNullable(nil): %w", err)
	}
	if nilValue != nil {
		return fmt.Errorf("NormalizeNullable(nil) = %#v, want nil", nilValue)
	}
	gotNullable, err := svc.NormalizeNullable(ctx, &input)
	if err != nil {
		return fmt.Errorf("NormalizeNullable(value): %w", err)
	}
	if gotNullable == nil || !reflect.DeepEqual(*gotNullable, want) {
		return fmt.Errorf("NormalizeNullable(value) = %#v, want %#v", gotNullable, want)
	}
	return nil
}
