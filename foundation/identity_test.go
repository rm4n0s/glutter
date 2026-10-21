package foundation

import "testing"

type sameTypeA struct{ x int }
type sameTypeB struct{ x int }

func TestSameRuntimeType(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"same struct", sameTypeA{1}, sameTypeA{2}, true},
		{"struct and pointer", sameTypeA{1}, &sameTypeA{2}, true},
		{"two pointers", &sameTypeA{1}, &sameTypeA{2}, true},
		{"different structs", sameTypeA{1}, sameTypeB{1}, false},
		{"different kinds", 1, "1", false},
		{"nil and nil", nil, nil, true},
		{"nil and a value", nil, sameTypeA{}, false},
		{"a value and nil", sameTypeA{}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SameRuntimeType(tt.a, tt.b); got != tt.want {
				t.Errorf("SameRuntimeType(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			// It agrees with the names, which is what the callers compared before.
			if got := RuntimeTypeName(tt.a) == RuntimeTypeName(tt.b); got != tt.want {
				t.Errorf("RuntimeTypeName comparison = %v, want %v", got, tt.want)
			}
		})
	}
}
