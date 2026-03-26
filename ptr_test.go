package libbinder

import "testing"

func TestPtr(t *testing.T) {
	v := 42
	got := Ptr(v)
	if got == nil || *got != v {
		t.Fatalf("Ptr(%d) = %v", v, got)
	}
}

func TestTypedPtrHelpers(t *testing.T) {
	tests := []struct {
		name string
		ok   func() bool
	}{
		{name: "bool", ok: func() bool { p := BoolPtr(true); return p != nil && *p }},
		{name: "string", ok: func() bool { p := StringPtr("x"); return p != nil && *p == "x" }},
		{name: "int", ok: func() bool { p := IntPtr(1); return p != nil && *p == 1 }},
		{name: "int8", ok: func() bool { p := Int8Ptr(2); return p != nil && *p == 2 }},
		{name: "int16", ok: func() bool { p := Int16Ptr(3); return p != nil && *p == 3 }},
		{name: "int32", ok: func() bool { p := Int32Ptr(4); return p != nil && *p == 4 }},
		{name: "int64", ok: func() bool { p := Int64Ptr(5); return p != nil && *p == 5 }},
		{name: "uint", ok: func() bool { p := UintPtr(6); return p != nil && *p == 6 }},
		{name: "uint8", ok: func() bool { p := Uint8Ptr(7); return p != nil && *p == 7 }},
		{name: "uint16", ok: func() bool { p := Uint16Ptr(8); return p != nil && *p == 8 }},
		{name: "uint32", ok: func() bool { p := Uint32Ptr(9); return p != nil && *p == 9 }},
		{name: "uint64", ok: func() bool { p := Uint64Ptr(10); return p != nil && *p == 10 }},
		{name: "uintptr", ok: func() bool { p := UintptrPtr(11); return p != nil && *p == 11 }},
		{name: "byte", ok: func() bool { p := BytePtr(12); return p != nil && *p == 12 }},
		{name: "rune", ok: func() bool { p := RunePtr('a'); return p != nil && *p == 'a' }},
		{name: "float32", ok: func() bool { p := Float32Ptr(1.25); return p != nil && *p == 1.25 }},
		{name: "float64", ok: func() bool { p := Float64Ptr(2.5); return p != nil && *p == 2.5 }},
		{name: "complex64", ok: func() bool { p := Complex64Ptr(complex(1, 2)); return p != nil && *p == complex64(complex(1, 2)) }},
		{name: "complex128", ok: func() bool { p := Complex128Ptr(complex(3, 4)); return p != nil && *p == complex(3, 4) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.ok() {
				t.Fatalf("%s helper returned unexpected pointer", tt.name)
			}
		})
	}
}
