package painting

import (
	"fmt"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func colorsOf(v ...uint32) []ui.Color {
	out := make([]ui.Color, len(v))
	for i, c := range v {
		out[i] = ui.NewColor(c)
	}
	return out
}

func colorsMatch(x, y []ui.Color) bool {
	if len(x) != len(y) {
		return false
	}
	const limit = 1.0 / 255
	for i := range x {
		if math.Abs(float64(x[i].A-y[i].A)) >= limit || math.Abs(float64(x[i].R-y[i].R)) >= limit ||
			math.Abs(float64(x[i].G-y[i].G)) >= limit || math.Abs(float64(x[i].B-y[i].B)) >= limit {
			return false
		}
	}
	return true
}

func stopsMatch(x, y []float64) bool {
	if (x == nil) != (y == nil) || len(x) != len(y) {
		return false
	}
	for i := range x {
		if math.Abs(x[i]-y[i]) >= 0.0001 {
			return false
		}
	}
	return true
}

func linearMatches(a, b *LinearGradient) bool {
	return a != nil && a.Begin.Equal(b.Begin) && a.End.Equal(b.End) && a.TileMode == b.TileMode &&
		gradientTransformEqual(a.Transform, b.Transform) && colorsMatch(a.Colors, b.Colors) && stopsMatch(a.Stops, b.Stops)
}

func TestLinearGradientLerp(t *testing.T) {
	g1 := NewLinearGradient(colorsOf(0x33333333, 0x66666666), LinearGradientOpts{Begin: AlignmentTopLeft, End: AlignmentBottomLeft})
	g2 := NewLinearGradient(colorsOf(0x44444444, 0x88888888), LinearGradientOpts{Begin: AlignmentTopRight, End: AlignmentTopLeft})
	tests := []struct {
		name string
		a, b *LinearGradient
		t    float64
		want *LinearGradient
	}{
		{
			"scale from nil", nil, NewLinearGradient(colorsOf(0x00FFFFFF, 0x11777777, 0x44444444), LinearGradientOpts{Begin: AlignmentBottomRight, End: Alignment{0.7, 1.0}}), 0.25,
			NewLinearGradient(colorsOf(0x00FFFFFF, 0x04777777, 0x11444444), LinearGradientOpts{Begin: AlignmentBottomRight, End: Alignment{0.7, 1.0}}),
		},
		{
			"lerp", g1, g2, 0.5,
			NewLinearGradient(colorsOf(0x3B3B3B3B, 0x77777777), LinearGradientOpts{Begin: AlignmentTopCenter, End: AlignmentCenterLeft, Stops: []float64{0, 1}}),
		},
		{
			"lerp with stops",
			NewLinearGradient(colorsOf(0x33333333, 0x66666666), LinearGradientOpts{Begin: AlignmentTopLeft, End: AlignmentBottomLeft, Stops: []float64{0, 0.5}}),
			NewLinearGradient(colorsOf(0x44444444, 0x88888888), LinearGradientOpts{Begin: AlignmentTopRight, End: AlignmentTopLeft, Stops: []float64{0.5, 1}}), 0.5,
			NewLinearGradient(colorsOf(0x3B3B3B3B, 0x55555555, 0x77777777), LinearGradientOpts{Begin: AlignmentTopCenter, End: AlignmentCenterLeft, Stops: []float64{0, 0.5, 1}}),
		},
		{
			"unequal colors", NewLinearGradient(colorsOf(0x22222222, 0x66666666)), NewLinearGradient(colorsOf(0x44444444, 0x66666666, 0x88888888)), 0.5,
			NewLinearGradient(colorsOf(0x33333333, 0x55555555, 0x77777777), LinearGradientOpts{Stops: []float64{0, 0.5, 1}}),
		},
		{
			"stops and unequal colors",
			NewLinearGradient(colorsOf(0x33333333, 0x66666666), LinearGradientOpts{Stops: []float64{0, 0.5}}),
			NewLinearGradient(colorsOf(0x44444444, 0x48484848, 0x88888888), LinearGradientOpts{Stops: []float64{0.5, 0.7, 1}}), 0.5,
			NewLinearGradient(colorsOf(0x3B3B3B3B, 0x55555555, 0x57575757, 0x77777777), LinearGradientOpts{Stops: []float64{0, 0.5, 0.7, 1}}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LinearGradientLerp(tt.a, tt.b, tt.t); !linearMatches(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
	if LinearGradientLerp(nil, nil, 0) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	same := NewLinearGradient(colorsOf(0x33333333, 0x66666666))
	if LinearGradientLerp(same, same, 0.5) != same {
		t.Error("lerp of identical gradients should return the same gradient")
	}
}

func TestLinearGradientLerpWithTransforms(t *testing.T) {
	mk := func(r float64) *LinearGradient {
		return NewLinearGradient(colorsOf(0x33333333, 0x66666666), LinearGradientOpts{Transform: GradientRotation{r}, Stops: []float64{0, 1}})
	}
	g1, g2 := mk(math.Pi/4), mk(math.Pi/2)
	for _, tt := range []struct {
		t    float64
		want *LinearGradient
	}{{0, g1}, {1, g2}, {0.5, g2}} {
		if got := LinearGradientLerp(g1, g2, tt.t); !got.Equal(tt.want) {
			t.Errorf("t=%v: got %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestLinearGradientEquality(t *testing.T) {
	base := func(r float64) *LinearGradient {
		return NewLinearGradient(colorsOf(0x33333333, 0x66666666), LinearGradientOpts{Transform: GradientRotation{r}})
	}
	if !base(math.Pi / 4).Equal(base(math.Pi / 4)) {
		t.Error("equal gradients should be equal")
	}
	if base(math.Pi / 4).Equal(base(math.Pi / 2)) {
		t.Error("gradients with different transforms should differ")
	}
	if base(0).Equal(NewLinearGradient(colorsOf(0x33333333, 0x66666666))) {
		t.Error("a gradient with a transform should differ from one without")
	}
}

func TestGradientToString(t *testing.T) {
	cs := colorsOf(0x33333333, 0x66666666)
	lc := "[" + cs[0].String() + ", " + cs[1].String() + "]"
	tests := []struct {
		name string
		got  Gradient
		want string
	}{
		{
			"linear", NewLinearGradient(cs, LinearGradientOpts{Begin: AlignmentTopLeft, End: AlignmentBottomLeft, Transform: GradientRotation{1.6}}),
			"LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomLeft, colors: " + lc + ", tileMode: TileMode.clamp, transform: GradientRotation(radians: 1.6))",
		},
		{
			"radial", NewRadialGradient(cs, RadialGradientOpts{Center: AlignmentTopLeft, Radius: foundation.Some(1.5)}),
			"RadialGradient(center: Alignment.topLeft, radius: 1.5, colors: " + lc + ", tileMode: TileMode.clamp, focalRadius: 0.0)",
		},
		{
			"sweep", NewSweepGradient(cs, SweepGradientOpts{Center: AlignmentTopLeft, StartAngle: 0.5, EndAngle: foundation.Some(1.5)}),
			"SweepGradient(center: Alignment.topLeft, startAngle: 0.5, endAngle: 1.5, colors: " + lc + ", tileMode: TileMode.clamp)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got.String(); got != tt.want {
				t.Errorf("String =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestRadialGradientLerp(t *testing.T) {
	g1 := NewRadialGradient(colorsOf(0x33333333, 0x66666666), RadialGradientOpts{Center: AlignmentTopLeft, Radius: foundation.Some(1.0), FocalRadius: 0.2})
	g2 := NewRadialGradient(colorsOf(0x44444444, 0x88888888), RadialGradientOpts{Center: AlignmentBottomRight, Radius: foundation.Some(2.0), FocalRadius: 0.4})
	got := RadialGradientLerp(g1, g2, 0.5)
	want := NewRadialGradient(colorsOf(0x3B3B3B3B, 0x77777777), RadialGradientOpts{
		Center: AlignmentCenter, Radius: foundation.Some(1.5), FocalRadius: 0.3, Stops: []float64{0, 1}})
	if !got.Center.Equal(want.Center) || got.Radius != want.Radius || math.Abs(got.FocalRadius-want.FocalRadius) > 1e-9 ||
		!colorsMatch(got.Colors, want.Colors) || !stopsMatch(got.Stops, want.Stops) {
		t.Errorf("got %v, want %v", got, want)
	}
	scaled := RadialGradientLerp(nil, g2, 0.25)
	if !colorsMatch(scaled.Colors, colorsOf(0x11444444, 0x22888888)) {
		t.Errorf("scaled colors = %v", scaled.Colors)
	}
	if RadialGradientLerp(nil, nil, 0) != nil || RadialGradientLerp(g1, g1, 0.3) != g1 {
		t.Error("lerp of nils or of identical gradients is wrong")
	}
}

func TestSweepGradientLerp(t *testing.T) {
	g1 := NewSweepGradient(colorsOf(0x33333333, 0x66666666), SweepGradientOpts{Center: AlignmentTopLeft, StartAngle: 1.0, EndAngle: foundation.Some(2.0)})
	g2 := NewSweepGradient(colorsOf(0x44444444, 0x88888888), SweepGradientOpts{Center: AlignmentBottomRight, StartAngle: 3.0, EndAngle: foundation.Some(4.0)})
	got := SweepGradientLerp(g1, g2, 0.5)
	if !got.Center.Equal(AlignmentCenter) || got.StartAngle != 2.0 || got.EndAngle != 3.0 || !colorsMatch(got.Colors, colorsOf(0x3B3B3B3B, 0x77777777)) {
		t.Errorf("got %v", got)
	}
	if SweepGradientLerp(nil, nil, 0) != nil || SweepGradientLerp(g1, g1, 0.3) != g1 {
		t.Error("lerp of nils or of identical gradients is wrong")
	}
	if got := SweepGradientLerp(g1, nil, 0.25); !colorsMatch(got.Colors, colorsOf(0x26333333, 0x4D666666)) {
		t.Errorf("scaled colors = %v", got.Colors)
	}
}

func TestGradientLerpAcrossKinds(t *testing.T) {
	lin := NewLinearGradient(colorsOf(0xFF000000, 0xFFFFFFFF))
	rad := NewRadialGradient(colorsOf(0xFF000000, 0xFFFFFFFF))
	if got := GradientLerp(lin, rad, 0.25); got == nil || !got.(*LinearGradient).Equal(lin.Scale(0.5)) {
		t.Errorf("t=.25: got %v, want the linear gradient scaled by 0.5", got)
	}
	if got := GradientLerp(lin, rad, 0.75); got == nil || !got.(*RadialGradient).Equal(rad.Scale(0.5)) {
		t.Errorf("t=.75: got %v, want the radial gradient scaled by 0.5", got)
	}
	if GradientLerp(nil, nil, 0.5) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	if got := GradientLerp(lin, lin, 0.5); got != Gradient(lin) {
		t.Error("lerp of identical gradients should return the same gradient")
	}
	if got := GradientLerp(nil, rad, 0.5); !got.(*RadialGradient).Equal(rad.Scale(0.5)) {
		t.Errorf("nil to gradient = %v", got)
	}
}

func TestGradientFromColorAndOpacity(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	for _, g := range []Gradient{NewLinearGradient(colorsOf(0xFF000000, 0xFF00FF00)), NewRadialGradient(colorsOf(0xFF000000, 0xFF00FF00)), NewSweepGradient(colorsOf(0xFF000000, 0xFF00FF00))} {
		t.Run(fmt.Sprintf("%T", g), func(t *testing.T) {
			for _, c := range g.FromColor(red).GetColors() {
				if c != red {
					t.Errorf("FromColor color = %v", c)
				}
			}
			if got := g.WithOpacity(0.5).GetColors(); got[0].Alpha() != 128 || got[1].Alpha() != 128 {
				t.Errorf("WithOpacity alphas = %v %v", got[0].Alpha(), got[1].Alpha())
			}
		})
	}
}

func TestGradientCreateShader(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 20, 100, 50)
	for _, g := range []Gradient{
		NewLinearGradient(colorsOf(0xFF000000, 0xFFFFFFFF)),
		NewRadialGradient(colorsOf(0xFF000000, 0xFFFFFFFF), RadialGradientOpts{Focal: AlignmentTopLeft, FocalRadius: 0.1}),
		NewSweepGradient(colorsOf(0xFF000000, 0xFFFFFFFF), SweepGradientOpts{Transform: GradientRotation{1}}),
	} {
		if g.CreateShader(rect, ui.TextDirectionLTR) == nil {
			t.Errorf("%T.CreateShader = nil", g)
		}
	}
}

func TestGradientRotationTransform(t *testing.T) {
	// A quarter turn about the center of the bounds maps its top-left corner to its top-right.
	bounds := ui.NewRectFromLTWH(0, 0, 100, 100)
	m := GradientRotation{math.Pi / 2}.Transform(bounds, ui.TextDirectionLTR)
	got := m.TransformPoint(ui.Offset{Dx: 0, Dy: 0})
	if math.Abs(got.Dx-100) > 1e-3 || math.Abs(got.Dy) > 1e-3 {
		t.Errorf("rotated corner = %v, want (100, 0)", got)
	}
}
