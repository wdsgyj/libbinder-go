package cases

import (
	"context"
	"fmt"
	"reflect"

	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func BasicMatrixInputBundle() shared.BasicBundle {
	return shared.BasicBundle{
		Ints:   []int32{1, 2, 3, 4},
		Triple: [3]int32{7, 8, 9},
		Note:   stringPtr("seed"),
		Tags:   []string{"red", "blue"},
		Payloads: []shared.BaselinePayload{
			{Code: 10, Note: stringPtr("ten")},
			{Code: 20, Note: nil},
		},
		Labels: map[string]string{
			"left":  "west",
			"right": "east",
		},
		PayloadMap: map[string]shared.BaselinePayload{
			"first":  {Code: 100, Note: stringPtr("alpha")},
			"second": {Code: 200, Note: nil},
		},
		Mode: shared.BasicModeAlpha,
		Value: shared.BasicUnion{
			Tag:  shared.BasicUnionTagText,
			Text: stringPtr("seed"),
		},
	}
}

func EchoNullable(prefix string, value *string) *string {
	if value == nil {
		return nil
	}
	return stringPtr(prefix + ":" + *value)
}

func ReverseInts(values []int32) []int32 {
	if values == nil {
		return nil
	}
	out := append([]int32(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func RotateTriple(value [3]int32) [3]int32 {
	return [3]int32{value[1], value[2], value[0]}
}

func DecorateTags(prefix string, tags []string) []string {
	if tags == nil {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, prefix+":"+tag)
	}
	return out
}

func DecoratePayloads(prefix string, payloads []shared.BaselinePayload) []shared.BaselinePayload {
	if payloads == nil {
		return nil
	}
	out := make([]shared.BaselinePayload, 0, len(payloads))
	for i, payload := range payloads {
		out = append(out, decoratePayload(prefix, payload, int32(i+1)))
	}
	return out
}

func DecorateLabels(prefix string, labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = prefix + ":" + value
	}
	return out
}

func DecoratePayloadMap(prefix string, payloadMap map[string]shared.BaselinePayload) map[string]shared.BaselinePayload {
	if payloadMap == nil {
		return nil
	}
	out := make(map[string]shared.BaselinePayload, len(payloadMap))
	for key, value := range payloadMap {
		out[key] = decoratePayload(prefix, value, int32(len(key)))
	}
	return out
}

func FlipMode(mode shared.BasicMode) shared.BasicMode {
	switch mode {
	case shared.BasicModeAlpha:
		return shared.BasicModeBeta
	case shared.BasicModeBeta:
		return shared.BasicModeAlpha
	default:
		return shared.BasicModeAlpha
	}
}

func NormalizeUnion(prefix string, value shared.BasicUnion) shared.BasicUnion {
	switch value.Tag {
	case shared.BasicUnionTagNumber:
		return shared.BasicUnion{Tag: shared.BasicUnionTagNumber, Number: value.Number + 1}
	case shared.BasicUnionTagText:
		return shared.BasicUnion{Tag: shared.BasicUnionTagText, Text: prefixedOrDefault(prefix, value.Text, "default")}
	case shared.BasicUnionTagPayload:
		return shared.BasicUnion{Tag: shared.BasicUnionTagPayload, Payload: decoratePayload(prefix, value.Payload, 50)}
	default:
		return shared.BasicUnion{Tag: shared.BasicUnionTagText, Text: stringPtr(prefix + ":default")}
	}
}

func NormalizeBundle(prefix string, value shared.BasicBundle) shared.BasicBundle {
	return shared.BasicBundle{
		Ints:       ReverseInts(value.Ints),
		Triple:     RotateTriple(value.Triple),
		Note:       prefixedOrDefault(prefix, value.Note, "default"),
		Tags:       DecorateTags(prefix, value.Tags),
		Payloads:   DecoratePayloads(prefix, value.Payloads),
		Labels:     DecorateLabels(prefix, value.Labels),
		PayloadMap: DecoratePayloadMap(prefix, value.PayloadMap),
		Mode:       FlipMode(value.Mode),
		Value:      NormalizeUnion(prefix, value.Value),
	}
}

func ExpandBundle(prefix string, input shared.BasicBundle, payload shared.BasicBundle) (int32, shared.BasicBundle, shared.BasicBundle) {
	ret := int32(len(input.Ints) + len(payload.Tags))
	doubled := NormalizeBundle(prefix, input)
	doubled.Ints = append(doubled.Ints, ret)
	payloadOut := NormalizeBundle(prefix, payload)
	payloadOut.Triple = [3]int32{payloadOut.Triple[0] + ret, payloadOut.Triple[1], payloadOut.Triple[2]}
	return ret, doubled, payloadOut
}

func VerifyBasicMatrixService(ctx context.Context, svc shared.IBasicMatrixService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}

	input := BasicMatrixInputBundle()

	echo, err := svc.EchoNullable(ctx, stringPtr("hello"))
	if err != nil {
		return fmt.Errorf("EchoNullable: %w", err)
	}
	if err := equal("EchoNullable", echo, EchoNullable(prefix, stringPtr("hello"))); err != nil {
		return err
	}

	reversed, err := svc.ReverseInts(ctx, input.Ints)
	if err != nil {
		return fmt.Errorf("ReverseInts: %w", err)
	}
	if err := equal("ReverseInts", reversed, ReverseInts(input.Ints)); err != nil {
		return err
	}

	rotated, err := svc.RotateTriple(ctx, input.Triple)
	if err != nil {
		return fmt.Errorf("RotateTriple: %w", err)
	}
	if err := equal("RotateTriple", rotated, RotateTriple(input.Triple)); err != nil {
		return err
	}

	tags, err := svc.DecorateTags(ctx, input.Tags)
	if err != nil {
		return fmt.Errorf("DecorateTags: %w", err)
	}
	if err := equal("DecorateTags", tags, DecorateTags(prefix, input.Tags)); err != nil {
		return err
	}

	payloads, err := svc.DecoratePayloads(ctx, input.Payloads)
	if err != nil {
		return fmt.Errorf("DecoratePayloads: %w", err)
	}
	if err := equal("DecoratePayloads", payloads, DecoratePayloads(prefix, input.Payloads)); err != nil {
		return err
	}

	labels, err := svc.DecorateLabels(ctx, input.Labels)
	if err != nil {
		return fmt.Errorf("DecorateLabels: %w", err)
	}
	if err := equal("DecorateLabels", labels, DecorateLabels(prefix, input.Labels)); err != nil {
		return err
	}

	payloadMap, err := svc.DecoratePayloadMap(ctx, input.PayloadMap)
	if err != nil {
		return fmt.Errorf("DecoratePayloadMap: %w", err)
	}
	if err := equal("DecoratePayloadMap", payloadMap, DecoratePayloadMap(prefix, input.PayloadMap)); err != nil {
		return err
	}

	mode, err := svc.FlipMode(ctx, input.Mode)
	if err != nil {
		return fmt.Errorf("FlipMode: %w", err)
	}
	if err := equal("FlipMode", mode, FlipMode(input.Mode)); err != nil {
		return err
	}

	union, err := svc.NormalizeUnion(ctx, input.Value)
	if err != nil {
		return fmt.Errorf("NormalizeUnion: %w", err)
	}
	if err := equal("NormalizeUnion", union, NormalizeUnion(prefix, input.Value)); err != nil {
		return err
	}

	bundle, err := svc.NormalizeBundle(ctx, input)
	if err != nil {
		return fmt.Errorf("NormalizeBundle: %w", err)
	}
	if err := equal("NormalizeBundle", bundle, NormalizeBundle(prefix, input)); err != nil {
		return err
	}

	second := shared.BasicBundle{
		Ints:   []int32{9, 8},
		Triple: [3]int32{4, 5, 6},
		Note:   nil,
		Tags:   []string{"amber"},
		Payloads: []shared.BaselinePayload{
			{Code: 3, Note: stringPtr("bee")},
		},
		Labels: map[string]string{
			"up": "north",
		},
		PayloadMap: map[string]shared.BaselinePayload{
			"solo": {Code: 33, Note: stringPtr("solo")},
		},
		Mode: shared.BasicModeBeta,
		Value: shared.BasicUnion{
			Tag:    shared.BasicUnionTagNumber,
			Number: 41,
		},
	}

	ret, doubled, payloadOut, err := svc.ExpandBundle(ctx, input, second)
	if err != nil {
		return fmt.Errorf("ExpandBundle: %w", err)
	}
	wantRet, wantDoubled, wantPayloadOut := ExpandBundle(prefix, input, second)
	if err := equal("ExpandBundle.ret", ret, wantRet); err != nil {
		return err
	}
	if err := equal("ExpandBundle.doubled", doubled, wantDoubled); err != nil {
		return err
	}
	if err := equal("ExpandBundle.payload", payloadOut, wantPayloadOut); err != nil {
		return err
	}

	return nil
}

func decoratePayload(prefix string, payload shared.BaselinePayload, codeDelta int32) shared.BaselinePayload {
	payload.Code += codeDelta
	payload.Note = prefixedOrDefault(prefix, payload.Note, "default")
	return payload
}

func prefixedOrDefault(prefix string, value *string, fallback string) *string {
	if value == nil {
		return stringPtr(prefix + ":" + fallback)
	}
	return stringPtr(prefix + ":" + *value)
}

func stringPtr(value string) *string {
	return &value
}

func equal(name string, got any, want any) error {
	if reflect.DeepEqual(got, want) {
		return nil
	}
	return fmt.Errorf("%s mismatch: got=%#v want=%#v", name, got, want)
}
