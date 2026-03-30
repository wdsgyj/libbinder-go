//go:build (linux || android) && (amd64 || arm64)

package kernel

import (
	"encoding/binary"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestParseTransactionObjectSupportsLocalBinderKinds(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		want api.ObjectKind
	}{
		{name: "strong binder", typ: BinderTypeBinder, want: api.ObjectStrongBinder},
		{name: "weak binder", typ: BinderTypeWeakBinder, want: api.ObjectWeakBinder},
		{name: "strong handle", typ: BinderTypeHandle, want: api.ObjectStrongBinder},
		{name: "weak handle", typ: BinderTypeWeakHandle, want: api.ObjectWeakBinder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, binderFlatObjectSize)
			binary.LittleEndian.PutUint32(payload[0:], tt.typ)
			binary.LittleEndian.PutUint32(payload[8:], 17)

			obj, err := parseTransactionObject(payload, 0)
			if err != nil {
				t.Fatalf("parseTransactionObject: %v", err)
			}
			if obj.Kind != tt.want {
				t.Fatalf("obj.Kind = %d, want %d", obj.Kind, tt.want)
			}
		})
	}
}
