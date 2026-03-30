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

func BasicMatrixTagGroups() []shared.BasicStringGroup {
	return []shared.BasicStringGroup{
		{Tags: []string{"red", "blue"}},
		{Tags: []string{"amber"}},
	}
}

func BasicMatrixPayloadBuckets() map[string][]shared.BaselinePayload {
	return map[string][]shared.BaselinePayload{
		"left": {
			{Code: 1, Note: stringPtr("alpha")},
			{Code: 2, Note: nil},
		},
		"right": {
			{Code: 3, Note: stringPtr("beta")},
		},
	}
}

func BasicMatrixInputEnvelope() shared.BasicEnvelope {
	value := shared.NewBasicEnvelope()
	value.Note = nil
	value.Primary = &shared.BaselinePayload{Code: 5, Note: stringPtr("prime")}
	value.History = []shared.BaselinePayload{
		{Code: 7, Note: stringPtr("history")},
		{Code: 8, Note: nil},
	}
	value.Bundle = BasicMatrixInputBundle()
	return value
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

func DecorateTagGroups(prefix string, groups []shared.BasicStringGroup) []shared.BasicStringGroup {
	if groups == nil {
		return nil
	}
	out := make([]shared.BasicStringGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, shared.BasicStringGroup{Tags: DecorateTags(prefix, group.Tags)})
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

func DecoratePayloadBuckets(prefix string, payloadBuckets map[string][]shared.BaselinePayload) map[string][]shared.BaselinePayload {
	if payloadBuckets == nil {
		return nil
	}
	out := make(map[string][]shared.BaselinePayload, len(payloadBuckets))
	for key, values := range payloadBuckets {
		out[key] = DecoratePayloads(prefix+":"+key, values)
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

func NormalizeEnvelope(prefix string, value shared.BasicEnvelope) shared.BasicEnvelope {
	out := shared.NewBasicEnvelope()
	title := value.Title
	if title == "" {
		title = "untitled"
	}
	out.Title = prefix + ":" + title
	out.Note = prefixedOrDefault(prefix, value.Note, "default")
	if value.Primary != nil {
		primary := decoratePayload(prefix, *value.Primary, 11)
		out.Primary = &primary
	}
	out.History = DecoratePayloads(prefix, value.History)
	out.Bundle = NormalizeBundle(prefix, value.Bundle)
	return out
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

	tagGroups, err := svc.DecorateTagGroups(ctx, BasicMatrixTagGroups())
	if err != nil {
		return fmt.Errorf("DecorateTagGroups: %w", err)
	}
	if err := equal("DecorateTagGroups", tagGroups, DecorateTagGroups(prefix, BasicMatrixTagGroups())); err != nil {
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

	payloadBuckets, err := svc.DecoratePayloadBuckets(ctx, BasicMatrixPayloadBuckets())
	if err != nil {
		return fmt.Errorf("DecoratePayloadBuckets: %w", err)
	}
	if err := equal("DecoratePayloadBuckets", payloadBuckets, DecoratePayloadBuckets(prefix, BasicMatrixPayloadBuckets())); err != nil {
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

	envelope, err := svc.NormalizeEnvelope(ctx, BasicMatrixInputEnvelope())
	if err != nil {
		return fmt.Errorf("NormalizeEnvelope: %w", err)
	}
	if err := equal("NormalizeEnvelope", envelope, NormalizeEnvelope(prefix, BasicMatrixInputEnvelope())); err != nil {
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

func BasicMatrixLargeInputBundle() shared.BasicBundle {
	ints := make([]int32, 4096)
	tags := make([]string, 256)
	payloads := make([]shared.BaselinePayload, 256)
	labels := make(map[string]string, 128)
	payloadMap := make(map[string]shared.BaselinePayload, 128)
	for i := range ints {
		ints[i] = int32(i + 1)
	}
	for i := range tags {
		tags[i] = fmt.Sprintf("tag-%03d", i)
	}
	for i := range payloads {
		payloads[i] = shared.BaselinePayload{
			Code: int32(i + 10),
			Note: stringPtr(fmt.Sprintf("note-%03d", i)),
		}
	}
	for i := 0; i < 128; i++ {
		key := fmt.Sprintf("key-%03d", i)
		labels[key] = fmt.Sprintf("label-%03d", i)
		payloadMap[key] = shared.BaselinePayload{
			Code: int32(200 + i),
			Note: stringPtr(fmt.Sprintf("payload-%03d", i)),
		}
	}
	return shared.BasicBundle{
		Ints:       ints,
		Triple:     [3]int32{101, 202, 303},
		Note:       stringPtr("bulk"),
		Tags:       tags,
		Payloads:   payloads,
		Labels:     labels,
		PayloadMap: payloadMap,
		Mode:       shared.BasicModeAlpha,
		Value: shared.BasicUnion{
			Tag:     shared.BasicUnionTagPayload,
			Payload: shared.BaselinePayload{Code: 999, Note: stringPtr("union")},
		},
	}
}

func BasicMatrixLargePayloadBuckets() map[string][]shared.BaselinePayload {
	out := make(map[string][]shared.BaselinePayload, 32)
	for i := 0; i < 32; i++ {
		key := fmt.Sprintf("bucket-%02d", i)
		values := make([]shared.BaselinePayload, 32)
		for j := range values {
			values[j] = shared.BaselinePayload{
				Code: int32(i*100 + j),
				Note: stringPtr(fmt.Sprintf("%s-item-%02d", key, j)),
			}
		}
		out[key] = values
	}
	return out
}

func BasicMatrixLargeEnvelope() shared.BasicEnvelope {
	value := shared.NewBasicEnvelope()
	value.Title = "bulk"
	value.Note = stringPtr("bulk-note")
	value.Primary = &shared.BaselinePayload{Code: 77, Note: stringPtr("primary")}
	value.History = make([]shared.BaselinePayload, 128)
	for i := range value.History {
		value.History[i] = shared.BaselinePayload{
			Code: int32(300 + i),
			Note: stringPtr(fmt.Sprintf("history-%03d", i)),
		}
	}
	value.Bundle = BasicMatrixLargeInputBundle()
	return value
}

func VerifyBasicMatrixPerformance(ctx context.Context, svc shared.IBasicMatrixService, prefix string) error {
	if svc == nil {
		return fmt.Errorf("nil service")
	}
	ints := BasicMatrixLargeInputBundle().Ints
	reversed, err := svc.ReverseInts(ctx, ints)
	if err != nil {
		return fmt.Errorf("ReverseInts(large): %w", err)
	}
	if err := equal("ReverseInts.large", reversed, ReverseInts(ints)); err != nil {
		return err
	}

	envelope := BasicMatrixLargeEnvelope()
	gotEnvelope, err := svc.NormalizeEnvelope(ctx, envelope)
	if err != nil {
		return fmt.Errorf("NormalizeEnvelope(large): %w", err)
	}
	if err := equal("NormalizeEnvelope.large", gotEnvelope, NormalizeEnvelope(prefix, envelope)); err != nil {
		return err
	}

	buckets := BasicMatrixLargePayloadBuckets()
	gotBuckets, err := svc.DecoratePayloadBuckets(ctx, buckets)
	if err != nil {
		return fmt.Errorf("DecoratePayloadBuckets(large): %w", err)
	}
	if err := equal("DecoratePayloadBuckets.large", gotBuckets, DecoratePayloadBuckets(prefix, buckets)); err != nil {
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
