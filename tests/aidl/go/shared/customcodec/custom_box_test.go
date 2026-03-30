package customcodec

import (
	"reflect"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestCustomBoxParcelRoundTrip(t *testing.T) {
	label := "seed"
	want := CustomBox{
		ID:    7,
		Label: &label,
		Tags:  []string{"red", "blue"},
		Meta:  map[string]string{"left": "west"},
	}
	p := api.NewParcel()
	if err := WriteCustomBoxToParcel(p, want); err != nil {
		t.Fatalf("WriteCustomBoxToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadCustomBoxFromParcel(p)
	if err != nil {
		t.Fatalf("ReadCustomBoxFromParcel: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}
