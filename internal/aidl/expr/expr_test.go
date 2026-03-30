package expr

import "testing"

func TestNormalizeStripsJavaNumericSuffixes(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "1f", want: "1"},
		{in: "-1f", want: "-1"},
		{in: "1F", want: "1"},
		{in: "1d", want: "1"},
		{in: ".5f", want: ".5"},
		{in: "1e3f", want: "1e3"},
		{in: "0x10L", want: "0x10"},
		{in: "~1L", want: "^1"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
