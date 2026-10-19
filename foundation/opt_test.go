package foundation

import "testing"

func TestOpt(t *testing.T) {
	tests := []struct {
		name    string
		o       Opt[int]
		some    bool
		value   int
		orElse7 int
	}{
		{"zero value is none", Opt[int]{}, false, 0, 7},
		{"None", None[int](), false, 0, 7},
		{"Some zero", Some(0), true, 0, 0},
		{"Some", Some(5), true, 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.IsSome(); got != tt.some {
				t.Errorf("IsSome = %v", got)
			}
			if got := tt.o.IsNone(); got == tt.some {
				t.Errorf("IsNone = %v", got)
			}
			v, ok := tt.o.Get()
			if v != tt.value || ok != tt.some {
				t.Errorf("Get = %v, %v", v, ok)
			}
			if got := tt.o.Value(); got != tt.value {
				t.Errorf("Value = %v", got)
			}
			if got := tt.o.OrElse(7); got != tt.orElse7 {
				t.Errorf("OrElse(7) = %v, want %v", got, tt.orElse7)
			}
		})
	}
	if Some(0) == None[int]() {
		t.Error("Some(0) must differ from None")
	}
	a, b := Some("a"), Some("a")
	if a != b {
		t.Error("equal Some values must be equal")
	}
}
