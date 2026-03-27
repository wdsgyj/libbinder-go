package framework

import (
	"reflect"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestIntentFilterRoundTrip(t *testing.T) {
	p := api.NewParcel()
	value := IntentFilter{
		Actions:         []string{"android.intent.action.VIEW", "android.intent.action.SEND"},
		Categories:      []string{"android.intent.category.DEFAULT"},
		DataSchemes:     []string{"content", "https"},
		StaticDataTypes: []string{"image/png"},
		DataTypes:       []string{"image/*"},
		MIMEGroups:      []string{"images"},
		DataSchemeSpecificParts: []PatternMatcher{
			{Pattern: "example", Type: 0},
		},
		DataAuthorities: []IntentFilterAuthorityEntry{
			{OriginalHost: "*.example.com", Host: ".example.com", Wild: true, Port: 443},
		},
		DataPaths: []PatternMatcher{
			{Pattern: "/items", Type: 1},
		},
		Priority:               42,
		HasStaticPartialTypes:  true,
		HasDynamicPartialTypes: true,
		AutoVerify:             true,
		InstantAppVisibility:   3,
		Order:                  9,
		Extras:                 NewEmptyPersistableBundle(),
		URIRelativeFilterGroups: []UriRelativeFilterGroup{
			{
				Action: 1,
				Filters: []UriRelativeFilter{
					{URIPart: 0, PatternType: 1, Filter: "/items"},
					{URIPart: 1, PatternType: 0, Filter: "id=42"},
				},
			},
		},
	}
	if err := WriteIntentFilterToParcel(p, value); err != nil {
		t.Fatalf("WriteIntentFilterToParcel: %v", err)
	}
	if err := p.SetPosition(0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	got, err := ReadIntentFilterFromParcel(p)
	if err != nil {
		t.Fatalf("ReadIntentFilterFromParcel: %v", err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("got = %#v, want %#v", got, value)
	}
}
