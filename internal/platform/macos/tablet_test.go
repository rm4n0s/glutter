package macos

import (
	"math"
	"testing"
)

func TestTabletDetail(t *testing.T) {
	tests := []struct {
		name                            string
		pressure, tx, ty                float64
		wantPressure, wantTilt, wantOri float64
	}{
		{"upright, no pressure", 0, 0, 0, 0, 0, 0},
		{"pressure is kept", 0.25, 0, 0, 0.25, 0, 0},
		{"pressure over 1 is clamped", 3, 0, 0, 1, 0, 0},
		{"a negative pressure is 0", -1, 0, 0, 0, 0, 0},
		{"leaning right by 45 degrees", 1, 0.5, 0, 1, math.Pi / 4, 0},
		{"leaning toward the user by 45 degrees", 1, 0, 0.5, 1, math.Pi / 4, math.Pi / 2},
		{"leaning left", 1, -0.5, 0, 1, math.Pi / 4, math.Pi},
		{"leaning away", 1, 0, -0.5, 1, math.Pi / 4, 3 * math.Pi / 2},
		{"flat", 1, 1, 0, 1, math.Pi / 2, 0},
		{"out of range values are clamped", 1, 7, 0, 1, math.Pi / 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tabletDetail(tt.pressure, tt.tx, tt.ty)
			if d.pressureMax != 1 {
				t.Errorf("pressureMax = %v", d.pressureMax)
			}
			if math.Abs(d.pressure-tt.wantPressure) > 1e-9 || math.Abs(d.tilt-tt.wantTilt) > 1e-9 || math.Abs(d.orientation-tt.wantOri) > 1e-9 {
				t.Errorf("got pressure %v tilt %v orientation %v; want %v %v %v", d.pressure, d.tilt, d.orientation, tt.wantPressure, tt.wantTilt, tt.wantOri)
			}
		})
	}
}
