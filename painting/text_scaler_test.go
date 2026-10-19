package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestTextScaler(t *testing.T) {
	tests := []struct {
		name       string
		scaler     TextScaler
		fontSize   float64
		want       float64
		wantFactor float64
		wantString string
	}{
		{"no scaling", TextScalerNoScaling, 14, 14, 1, "no scaling"},
		{"linear", NewTextScalerLinear(2), 14, 28, 2, "linear (2.0x)"},
		{"linear clamp above", NewTextScalerLinear(3).Clamp(TextScalerClampOpts{MaxScaleFactor: foundation.Some(2.0)}), 10, 20, 2, "linear (2.0x)"},
		{"linear clamp below", NewTextScalerLinear(0.5).Clamp(TextScalerClampOpts{MinScaleFactor: 1}), 10, 10, 1, "no scaling"},
		{"linear inside clamp", NewTextScalerLinear(1.5).Clamp(TextScalerClampOpts{MinScaleFactor: 1, MaxScaleFactor: foundation.Some(2.0)}), 10, 15, 1.5, "linear (1.5x)"},
		{"clamp of nothing", NewTextScalerLinear(1.5).Clamp(), 10, 15, 1.5, "linear (1.5x)"},
		{"fixed clamp", TextScalerClamp(testNonlinear{}, TextScalerClampOpts{MinScaleFactor: 2, MaxScaleFactor: foundation.Some(2.0)}), 10, 20, 2, "linear (2.0x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scaler.Scale(tt.fontSize); got != tt.want {
				t.Errorf("Scale(%v) = %v, want %v", tt.fontSize, got, tt.want)
			}
			if got := tt.scaler.TextScaleFactor(); got != tt.wantFactor {
				t.Errorf("TextScaleFactor = %v, want %v", got, tt.wantFactor)
			}
			if got := tt.scaler.String(); got != tt.wantString {
				t.Errorf("String = %q, want %q", got, tt.wantString)
			}
		})
	}
}

// testNonlinear is a scaler that grows small fonts more, like the ones of
// Android 14.
type testNonlinear struct{}

func (testNonlinear) Scale(fontSize float64) float64 { return fontSize * 3 / (1 + fontSize/10) }
func (testNonlinear) TextScaleFactor() float64       { return 3 }
func (s testNonlinear) Clamp(opts ...TextScalerClampOpts) TextScaler {
	return TextScalerClamp(s, opts...)
}
func (testNonlinear) String() string { return "nonlinear" }

func TestClampedNonlinearScaler(t *testing.T) {
	s := TextScalerClamp(testNonlinear{}, TextScalerClampOpts{MinScaleFactor: 1, MaxScaleFactor: foundation.Some(2.0)})
	tests := []struct {
		fontSize, want float64
	}{
		{10, 15},   // 10*3/2 = 15, inside [10, 20]
		{1, 2},     // 3/1.1 = 2.7, clamped to 2
		{100, 100}, // 300/11 = 27, clamped to the minimum 1x
	}
	for _, tt := range tests {
		if got := s.Scale(tt.fontSize); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("Scale(%v) = %v, want %v", tt.fontSize, got, tt.want)
		}
	}
	if got := s.TextScaleFactor(); got != 2 {
		t.Errorf("TextScaleFactor = %v, want 2 (the factor of 3 clamped)", got)
	}
	if got, want := s.String(), "nonlinear clamped [1.0, 2.0]"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	// Clamping a clamped scaler narrows the range.
	narrow := s.Clamp(TextScalerClampOpts{MinScaleFactor: 1.5, MaxScaleFactor: foundation.Some(1.8)})
	if got, want := narrow.String(), "nonlinear clamped [1.5, 1.8]"; got != want {
		t.Errorf("narrowed = %q, want %q", got, want)
	}
	// Clamping to a range disjoint from the current one gives a linear scaler.
	disjoint := s.Clamp(TextScalerClampOpts{MinScaleFactor: 3, MaxScaleFactor: foundation.Some(4.0)})
	if got, want := disjoint.String(), "linear (3.0x)"; got != want {
		t.Errorf("disjoint = %q, want %q", got, want)
	}
}

func TestTextScalerEqual(t *testing.T) {
	a := NewTextScalerLinear(1.5)
	tests := []struct {
		name string
		x, y TextScaler
		want bool
	}{
		{"same factor", a, NewTextScalerLinear(1.5), true},
		{"other factor", a, NewTextScalerLinear(2), false},
		{"no scaling is linear 1", TextScalerNoScaling, NewTextScalerLinear(1), true},
		{"nil and nil", nil, nil, true},
		{"nil and scaler", nil, a, false},
		{"clamped same", a.Clamp(TextScalerClampOpts{MaxScaleFactor: foundation.Some(1.2)}), NewTextScalerLinear(1.2), true},
	}
	for _, tt := range tests {
		if got := TextScalerEqual(tt.x, tt.y); got != tt.want {
			t.Errorf("%s: TextScalerEqual = %v, want %v", tt.name, got, tt.want)
		}
	}
}
