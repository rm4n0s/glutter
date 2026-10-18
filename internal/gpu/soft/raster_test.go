package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func poly(pts ...float64) []ui.Offset {
	out := make([]ui.Offset, 0, len(pts)/2)
	for i := 0; i+1 < len(pts); i += 2 {
		out = append(out, ui.Offset{Dx: pts[i], Dy: pts[i+1]})
	}
	return out
}

var big = image.Rect(-1000, -1000, 1000, 1000)

func TestRasterizeArea(t *testing.T) {
	tests := []struct {
		name     string
		contours [][]ui.Offset
		rule     FillRule
		want     float64
	}{
		{"pixel aligned rect", [][]ui.Offset{poly(2, 3, 7, 3, 7, 9, 2, 9)}, FillNonZero, 30},
		{"fractional rect", [][]ui.Offset{poly(0.25, 0.5, 4.75, 0.5, 4.75, 3.25, 0.25, 3.25)}, FillNonZero, 4.5 * 2.75},
		{"right triangle", [][]ui.Offset{poly(0, 0, 10, 0, 0, 10)}, FillNonZero, 50},
		{"thin sliver", [][]ui.Offset{poly(0, 0, 100, 0.3, 0, 0.6)}, FillNonZero, 100 * 0.6 / 2},
		{"reversed winding", [][]ui.Offset{poly(7, 3, 2, 3, 2, 9, 7, 9)}, FillNonZero, 30},
		{"steep edge across many columns", [][]ui.Offset{poly(0, 0, 40.5, 1, 0, 1)}, FillNonZero, 20.25},
		{"overlapping same winding", [][]ui.Offset{poly(0, 0, 10, 0, 10, 10, 0, 10), poly(5, 5, 15, 5, 15, 15, 5, 15)}, FillNonZero, 175},
		{"opposite winding cancels", [][]ui.Offset{poly(0, 0, 10, 0, 10, 10, 0, 10), poly(2, 2, 2, 8, 8, 8, 8, 2)}, FillNonZero, 100 - 36},
		{"even-odd overlap", [][]ui.Offset{poly(0, 0, 10, 0, 10, 10, 0, 10), poly(2, 2, 8, 2, 8, 8, 2, 8)}, FillEvenOdd, 100 - 36},
		{"non-zero same winding nested", [][]ui.Offset{poly(0, 0, 10, 0, 10, 10, 0, 10), poly(2, 2, 8, 2, 8, 8, 2, 8)}, FillNonZero, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Rasterize(tt.contours, tt.rule, big)
			if got := m.Sum(); math.Abs(got-tt.want) > 1e-3 {
				t.Errorf("area = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRasterizeCoverage(t *testing.T) {
	// A rect from x=1.5 to 3.25 and y=0 to 1: the pixel coverages are exact.
	m := Rasterize([][]ui.Offset{poly(1.5, 0, 3.25, 0, 3.25, 1, 1.5, 1)}, FillNonZero, big)
	for x, want := range map[int]float32{0: 0, 1: 0.5, 2: 1, 3: 0.25, 4: 0} {
		if got := m.At(x, 0); math.Abs(float64(got-want)) > 1e-5 {
			t.Errorf("coverage at %d = %v, want %v", x, got, want)
		}
	}
	if m.At(-5, 0) != 0 || m.At(2, 7) != 0 {
		t.Error("coverage outside the mask is not 0")
	}
}

func TestRasterizeClipsToRect(t *testing.T) {
	// A rect much larger than the clip, with edges beyond both sides.
	contours := [][]ui.Offset{poly(-500, -500, 500, -500, 500, 500, -500, 500)}
	clip := image.Rect(3, 4, 8, 10)
	m := Rasterize(contours, FillNonZero, clip)
	if m.Rect != clip {
		t.Fatalf("mask rect = %v, want %v", m.Rect, clip)
	}
	if got := m.Sum(); math.Abs(got-30) > 1e-4 {
		t.Errorf("area = %v, want 30", got)
	}
	// A triangle whose hypotenuse crosses the left and right clip edges.
	tri := [][]ui.Offset{poly(-20, 0, 40, 0, -20, 60)}
	m = Rasterize(tri, FillNonZero, image.Rect(0, 0, 10, 10))
	// Inside the clip the result must equal the unclipped one.
	full := Rasterize(tri, FillNonZero, big)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if a, b := m.At(x, y), full.At(x, y); math.Abs(float64(a-b)) > 1e-4 {
				t.Fatalf("clipped coverage at %d,%d = %v, unclipped %v", x, y, a, b)
			}
		}
	}
}

func TestRasterizeDegenerate(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	tests := []struct {
		name     string
		contours [][]ui.Offset
	}{
		{"none", nil},
		{"one point", [][]ui.Offset{poly(1, 1)}},
		{"line", [][]ui.Offset{poly(0, 0, 10, 10)}},
		{"horizontal", [][]ui.Offset{poly(0, 5, 10, 5, 0, 5)}},
		{"nan", [][]ui.Offset{poly(0, 0, nan, 0, 10, 10)}},
		{"inf", [][]ui.Offset{poly(0, 0, inf, 0, 10, 10)}},
		{"outside the clip", [][]ui.Offset{poly(2000, 2000, 2010, 2000, 2010, 2010)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if m := Rasterize(tt.contours, FillNonZero, big); m != nil {
				t.Errorf("got a mask with area %v", m.Sum())
			}
		})
	}
	// A bad contour is skipped, the others still draw.
	m := Rasterize([][]ui.Offset{poly(0, 0, nan, 0, 10, 10), poly(0, 0, 4, 0, 4, 4, 0, 4)}, FillNonZero, big)
	if got := m.Sum(); math.Abs(got-16) > 1e-4 {
		t.Errorf("area = %v, want 16", got)
	}
	// Huge coordinates are clamped instead of overflowing.
	m = Rasterize([][]ui.Offset{poly(-1e30, -1e30, 1e30, -1e30, 1e30, 1e30, -1e30, 1e30)}, FillNonZero, image.Rect(0, 0, 4, 4))
	if got := m.Sum(); math.Abs(got-16) > 1e-4 {
		t.Errorf("area of a huge rect = %v, want 16", got)
	}
}

func TestFold(t *testing.T) {
	tests := []struct {
		acc  float64
		rule FillRule
		want float64
	}{
		{0.4, FillNonZero, 0.4},
		{-0.4, FillNonZero, 0.4},
		{3, FillNonZero, 1},
		{1.5, FillEvenOdd, 0.5},
		{2, FillEvenOdd, 0},
		{2.25, FillEvenOdd, 0.25},
		{-1, FillEvenOdd, 1},
	}
	for _, tt := range tests {
		if got := fold(tt.acc, tt.rule); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("fold(%v, %v) = %v, want %v", tt.acc, tt.rule, got, tt.want)
		}
	}
}

func FuzzRasterize(f *testing.F) {
	f.Add(0.0, 0.0, 10.0, 0.0, 5.0, 8.0, 3.0, -2.0)
	f.Add(-40.5, 7.25, 1e9, 3.0, 0.1, 0.2, 99.0, 99.0)
	f.Fuzz(func(t *testing.T, a, b, c, d, e, g, h, i float64) {
		clip := image.Rect(0, 0, 32, 32)
		for _, rule := range []FillRule{FillNonZero, FillEvenOdd} {
			m := Rasterize([][]ui.Offset{poly(a, b, c, d, e, g, h, i)}, rule, clip)
			if m == nil {
				continue
			}
			for _, v := range m.Cov {
				if !(v >= 0 && v <= 1) {
					t.Fatalf("coverage %v out of range", v)
				}
			}
		}
	})
}
