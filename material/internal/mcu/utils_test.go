package mcu

import (
	"fmt"
	"math"
	"testing"
)

func rangeOf(start, stop float64, caseCount int) []float64 {
	step := (stop - start) / float64(caseCount-1)
	out := make([]float64, caseCount)
	for i := range out {
		out[i] = start + step*float64(i)
	}
	return out
}

func rgbRange() []int {
	var out []int
	for _, v := range rangeOf(0, 255, 8) {
		out = append(out, int(math.Round(v)))
	}
	return out
}

// Ported from color_utils_test.dart.
func TestRangeIntegrity(t *testing.T) {
	r := rangeOf(3.0, 9999.0, 1234)
	for i := range 1234 {
		closeTo(t, fmt.Sprint("range ", i), r[i], 3+8.1070559611*float64(i), 1e-5)
	}
}

func TestArgbFromRgb(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b int
		want    uint32
	}{
		{"black", 0, 0, 0, 0xff000000},
		{"white", 255, 255, 255, 0xffffffff},
		{"random color", 50, 150, 250, 0xff3296fa},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireColor(t, "argb", ArgbFromRgb(tt.r, tt.g, tt.b), tt.want)
		})
	}
	if ArgbFromRgb(0, 0, 0) != 4278190080 || ArgbFromRgb(255, 255, 255) != 4294967295 || ArgbFromRgb(50, 150, 250) != 4281505530 {
		t.Error("the decimal values differ")
	}
}

func TestYAndLstarRoundTrips(t *testing.T) {
	for _, y := range rangeOf(0, 100, 1001) {
		closeTo(t, "y to lstar to y", YFromLstar(LstarFromY(y)), y, 1e-5)
		closeTo(t, "lstar to y to lstar", LstarFromY(YFromLstar(y)), y, 1e-5)
	}
}

func TestYFromLstarTable(t *testing.T) {
	for _, tc := range [][2]float64{
		{0.0, 0.0}, {0.1, 0.0110705}, {0.2, 0.0221411}, {0.3, 0.0332116}, {0.4, 0.0442822}, {0.5, 0.0553528},
		{1.0, 0.1107056}, {2.0, 0.2214112}, {3.0, 0.3321169}, {4.0, 0.4428225}, {5.0, 0.5535282}, {8.0, 0.8856451},
		{10.0, 1.1260199}, {15.0, 1.9085832}, {20.0, 2.9890524}, {25.0, 4.4154767}, {30.0, 6.2359055},
		{40.0, 11.2509737}, {50.0, 18.4186518}, {60.0, 28.1233342}, {70.0, 40.7494157}, {80.0, 56.6812907},
		{90.0, 76.3033539}, {95.0, 87.6183294}, {99.0, 97.4360239}, {100.0, 100.0},
	} {
		closeTo(t, fmt.Sprint("yFromLstar(", tc[0], ")"), YFromLstar(tc[0]), tc[1], 1e-5)
	}
}

func TestLstarFromYTable(t *testing.T) {
	for _, tc := range [][2]float64{
		{0.0, 0.0}, {0.1, 0.9032962}, {0.2, 1.8065925}, {0.3, 2.7098888}, {0.4, 3.6131851}, {0.5, 4.5164814},
		{0.8856451, 8.0}, {1.0, 8.9914424}, {2.0, 15.4872443}, {3.0, 20.0438970}, {4.0, 23.6714419},
		{5.0, 26.7347653}, {10.0, 37.8424304}, {15.0, 45.6341970}, {20.0, 51.8372115}, {25.0, 57.0754208},
		{30.0, 61.6542222}, {40.0, 69.4695307}, {50.0, 76.0692610}, {60.0, 81.8381891}, {70.0, 86.9968642},
		{80.0, 91.6848609}, {90.0, 95.9967686}, {95.0, 98.0335184}, {99.0, 99.6120372}, {100.0, 100.0},
	} {
		closeTo(t, fmt.Sprint("lstarFromY(", tc[0], ")"), LstarFromY(tc[0]), tc[1], 1e-5)
	}
}

func TestYContinuity(t *testing.T) {
	const epsilon, delta = 1e-6, 1e-8
	closeTo(t, "left", YFromLstar(8.0-delta), YFromLstar(8.0), epsilon)
	closeTo(t, "right", YFromLstar(8.0+delta), YFromLstar(8.0), epsilon)
}

func TestColorSpaceRoundTrips(t *testing.T) {
	for _, r := range rgbRange() {
		for _, g := range rgbRange() {
			for _, b := range rgbRange() {
				argb := ArgbFromRgb(r, g, b)
				xyz := XyzFromArgb(argb)
				viaXyz := ArgbFromXyz(xyz[0], xyz[1], xyz[2])
				lab := LabFromArgb(argb)
				viaLab := ArgbFromLab(lab[0], lab[1], lab[2])
				for name, c := range map[string]uint32{"xyz": viaXyz, "lab": viaLab} {
					closeTo(t, name+" red", float64(RedFromArgb(c)), float64(r), 1.5)
					closeTo(t, name+" green", float64(GreenFromArgb(c)), float64(g), 1.5)
					closeTo(t, name+" blue", float64(BlueFromArgb(c)), float64(b), 1.5)
				}
				// L* and Y commute.
				closeTo(t, "y", YFromLstar(LstarFromArgb(argb)), xyz[1], 1e-5)
			}
		}
	}
}

func TestLstarRoundTrips(t *testing.T) {
	for c := range 256 {
		argb := ArgbFromRgb(c, c, c)
		requireColor(t, fmt.Sprint("gray ", c), ArgbFromLstar(LstarFromArgb(argb)), argb)
		if got := Delinearized(Linearized(c)); got != c {
			t.Errorf("delinearized(linearized(%d)) = %d", c, got)
		}
	}
	for _, lstar := range rangeOf(0, 100, 1001) {
		closeTo(t, "lstar to rgb to y", XyzFromArgb(ArgbFromLstar(lstar))[1], YFromLstar(lstar), 1)
	}
}

func TestHexConversions(t *testing.T) {
	if got := HexFromArgb(0xff3296fa, true); got != "#3296FA" {
		t.Errorf("hex = %q", got)
	}
	if got := HexFromArgb(0xff00010f, false); got != "00010F" {
		t.Errorf("hex = %q", got)
	}
	if got, ok := ArgbFromHex("#ff3296fa"); !ok || got != 0xff3296fa {
		t.Errorf("argbFromHex = %x, %v", got, ok)
	}
	if _, ok := ArgbFromHex("zz"); ok {
		t.Error("zz parsed")
	}
}

func TestComponentsAndOpacity(t *testing.T) {
	argb := uint32(0x80102030)
	if AlphaFromArgb(argb) != 0x80 || RedFromArgb(argb) != 0x10 || GreenFromArgb(argb) != 0x20 || BlueFromArgb(argb) != 0x30 {
		t.Error("components")
	}
	if IsOpaque(argb) || !IsOpaque(0xff000000) {
		t.Error("opaque")
	}
}

// Ported from math_utils_test.dart.
func TestRotationDirection(t *testing.T) {
	direction := func(from, to float64) float64 {
		a, b, c := to-from, to-from+360.0, to-from-360.0
		aAbs, bAbs, cAbs := math.Abs(a), math.Abs(b), math.Abs(c)
		switch {
		case aAbs <= bAbs && aAbs <= cAbs:
			if a >= 0 {
				return 1
			}
			return -1
		case bAbs <= aAbs && bAbs <= cAbs:
			if b >= 0 {
				return 1
			}
			return -1
		}
		if c >= 0 {
			return 1
		}
		return -1
	}
	for from := 0.0; from < 360.0; from += 15.0 {
		for to := 7.5; to < 360.0; to += 15.0 {
			got := RotationDirection(from, to)
			if want := direction(from, to); got != want {
				t.Errorf("rotationDirection(%v, %v) = %v, want %v", from, to, got, want)
			}
			if math.Abs(got) != 1.0 {
				t.Errorf("rotationDirection(%v, %v) = %v, want +1 or -1", from, to, got)
			}
		}
	}
}

func TestMathHelpers(t *testing.T) {
	if Signum(-3) != -1 || Signum(0) != 0 || Signum(2) != 1 {
		t.Error("signum")
	}
	if Lerp(10, 20, 0.5) != 15 || ClampInt(0, 5, 9) != 5 || ClampInt(0, 5, -1) != 0 || ClampDouble(0, 1, 0.5) != 0.5 {
		t.Error("lerp or clamp")
	}
	if SanitizeDegreesInt(-30) != 330 || SanitizeDegreesInt(725) != 5 {
		t.Error("sanitizeDegreesInt")
	}
	if SanitizeDegreesDouble(-30.5) != 329.5 || SanitizeDegreesDouble(360) != 0 || SanitizeDegreesDouble(725.5) != 5.5 {
		t.Error("sanitizeDegreesDouble")
	}
	if DifferenceDegrees(10, 350) != 20 || DifferenceDegrees(0, 180) != 180 {
		t.Error("differenceDegrees")
	}
	id := [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	if MatrixMultiply([3]float64{1, 2, 3}, id) != [3]float64{1, 2, 3} {
		t.Error("matrixMultiply")
	}
}
