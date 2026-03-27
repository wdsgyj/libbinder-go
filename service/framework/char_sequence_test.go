package framework

import (
	"errors"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestCharSequenceRoundTrip(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		p := api.NewParcel()
		value := CharSequence{Text: "hello"}
		if err := WriteCharSequenceToParcel(p, value); err != nil {
			t.Fatalf("WriteCharSequenceToParcel: %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadCharSequenceFromParcel(p)
		if err != nil {
			t.Fatalf("ReadCharSequenceFromParcel: %v", err)
		}
		if got != value {
			t.Fatalf("got = %#v, want %#v", got, value)
		}
	})

	t.Run("spanned_without_spans", func(t *testing.T) {
		p := api.NewParcel()
		value := CharSequence{Text: "hello", Spanned: true}
		if err := WriteCharSequenceToParcel(p, value); err != nil {
			t.Fatalf("WriteCharSequenceToParcel: %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got, err := ReadCharSequenceFromParcel(p)
		if err != nil {
			t.Fatalf("ReadCharSequenceFromParcel: %v", err)
		}
		if got != value {
			t.Fatalf("got = %#v, want %#v", got, value)
		}
	})

	t.Run("styled_span_bad_parcelable", func(t *testing.T) {
		p := api.NewParcel()
		if err := p.WriteInt32(0); err != nil {
			t.Fatalf("WriteInt32(kind): %v", err)
		}
		if err := p.WriteString8("styled"); err != nil {
			t.Fatalf("WriteString8(text): %v", err)
		}
		if err := p.WriteInt32(7); err != nil {
			t.Fatalf("WriteInt32(span kind): %v", err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		_, err := ReadCharSequenceFromParcel(p)
		if !errors.Is(err, api.ErrBadParcelable) {
			t.Fatalf("ReadCharSequenceFromParcel error = %v, want ErrBadParcelable", err)
		}
	})
}
