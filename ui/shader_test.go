package ui

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestGradientLinear(t *testing.T) {
	red, blue := NewColor(0xFFFF0000), NewColor(0xFF0000FF)
	m := NewMatrix4TranslationValues(3, 4, 0)
	g := NewGradientLinear(Offset{0, 0}, Offset{100, 50}, []Color{red, blue})
	if g.Kind() != GradientKindLinear || g.From() != (Offset{0, 0}) || g.To() != (Offset{100, 50}) {
		t.Errorf("kind/from/to = %v %v %v", g.Kind(), g.From(), g.To())
	}
	if g.TileMode() != TileModeClamp {
		t.Errorf("the default tile mode is clamp, got %v", g.TileMode())
	}
	if g.ColorStops() != nil && len(g.ColorStops()) != 0 {
		t.Errorf("two colors without stops have no stops: %v", g.ColorStops())
	}
	if _, ok := g.Matrix4(); ok {
		t.Error("no matrix was given")
	}
	if got := g.Colors(); len(got) != 2 || got[0].R != 1 || got[1].B != 1 {
		t.Errorf("Colors = %v", got)
	}

	g3 := NewGradientLinear(Offset{}, Offset{1, 1}, []Color{red, blue, red}, GradientLinearOpts{
		ColorStops: []float64{0, 0.25, 1}, TileMode: TileModeMirror, Matrix4: m,
	})
	if got := g3.ColorStops(); len(got) != 3 || got[1] != 0.25 {
		t.Errorf("ColorStops = %v", got)
	}
	if g3.TileMode() != TileModeMirror {
		t.Errorf("TileMode = %v", g3.TileMode())
	}
	if got, ok := g3.Matrix4(); !ok || *got != *m {
		t.Errorf("Matrix4 = %v %v", got, ok)
	}

	// The gradient does not share the caller's slices or matrix.
	stops := []float64{0, 1}
	colors := []Color{red, blue}
	g2 := NewGradientLinear(Offset{}, Offset{1, 1}, colors, GradientLinearOpts{ColorStops: stops, Matrix4: m})
	stops[1], colors[0], m.storage[12] = 0.5, blue, 99
	if g2.ColorStops()[1] != 1 || g2.Colors()[0].R != 1 {
		t.Error("the gradient must copy the colors and the stops")
	}
	if got, _ := g2.Matrix4(); got.storage[12] != 3 {
		t.Error("the gradient must copy the matrix")
	}
	// And the getters give copies.
	g2.Colors()[0] = blue
	if g2.Colors()[0].R != 1 {
		t.Error("Colors must return a copy")
	}
}

func TestGradientColorsAreExtendedSRGB(t *testing.T) {
	p3 := NewColorFrom(1, 0.5, 0.25, 0.75, ColorFromOpts{ColorSpace: ColorSpaceDisplayP3})
	g := NewGradientLinear(Offset{}, Offset{1, 0}, []Color{p3, p3})
	want := p3.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceExtendedSRGB)})
	if got := g.Colors()[0]; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if g.Colors()[0].ColorSpace != ColorSpaceExtendedSRGB {
		t.Errorf("a Display P3 color becomes extended sRGB, got %v", g.Colors()[0].ColorSpace)
	}
}

func TestGradientRadial(t *testing.T) {
	cs := []Color{NewColor(0xFFFF0000), NewColor(0xFF0000FF)}
	center := Offset{50, 50}
	tests := []struct {
		name        string
		g           *Gradient
		kind        GradientKind
		focal       Offset
		focalRadius float64
	}{
		{"plain", NewGradientRadial(center, 40, cs), GradientKindRadial, Offset{}, 0},
		{"focal equal to the center with no radius is plain", NewGradientRadial(center, 40, cs, GradientRadialOpts{Focal: &center}), GradientKindRadial, Offset{}, 0},
		{"focal equal to the center with a radius is conical", NewGradientRadial(center, 40, cs, GradientRadialOpts{Focal: &center, FocalRadius: 5}), GradientKindConical, center, 5},
		{"a different focal point is conical", NewGradientRadial(center, 40, cs, GradientRadialOpts{Focal: &Offset{10, 20}, FocalRadius: 2}), GradientKindConical, Offset{10, 20}, 2},
		{"a focal radius without a focal point is plain", NewGradientRadial(center, 40, cs, GradientRadialOpts{FocalRadius: 9}), GradientKindRadial, Offset{}, 0},
	}
	for _, tt := range tests {
		if tt.g.Kind() != tt.kind || tt.g.Center() != center || tt.g.Radius() != 40 ||
			tt.g.Focal() != tt.focal || tt.g.FocalRadius() != tt.focalRadius {
			t.Errorf("%s: kind %v center %v radius %v focal %v focalRadius %v", tt.name,
				tt.g.Kind(), tt.g.Center(), tt.g.Radius(), tt.g.Focal(), tt.g.FocalRadius())
		}
	}
}

func TestGradientSweep(t *testing.T) {
	cs := []Color{NewColor(0xFFFF0000), NewColor(0xFF0000FF)}
	tests := []struct {
		name       string
		opts       []GradientSweepOpts
		start, end float64
	}{
		{"defaults", nil, 0, 2 * math.Pi},
		{"start only", []GradientSweepOpts{{StartAngle: 1}}, 1, 2 * math.Pi},
		{"both", []GradientSweepOpts{{StartAngle: 1, EndAngle: 2}}, 1, 2},
		{"an end of zero after a negative start is kept", []GradientSweepOpts{{StartAngle: -1, EndAngle: 0}}, -1, 0},
	}
	for _, tt := range tests {
		g := NewGradientSweep(Offset{1, 2}, cs, tt.opts...)
		start, end := g.StartAngle(), g.EndAngle()
		if g.Kind() != GradientKindSweep || g.Center() != (Offset{1, 2}) || start != tt.start || end != tt.end {
			t.Errorf("%s: start %v end %v, want %v %v", tt.name, start, end, tt.start, tt.end)
		}
	}
}

func TestGradientPanics(t *testing.T) {
	c := NewColor(0xFF000000)
	tests := []struct {
		name  string
		build func()
		tag   string
		route string
	}{
		{"one color without stops", func() { NewGradientLinear(Offset{}, Offset{}, []Color{c}) },
			"GradientColorCountInvalid", "NewGradientLinear->newGradient->validateColorStops.GradientColorCountInvalid"},
		{"three colors without stops", func() { NewGradientRadial(Offset{}, 1, []Color{c, c, c}) },
			"GradientColorCountInvalid", "NewGradientRadial->newGradient->validateColorStops.GradientColorCountInvalid"},
		{"no colors", func() { NewGradientSweep(Offset{}, nil) },
			"GradientColorCountInvalid", "NewGradientSweep->newGradient->validateColorStops.GradientColorCountInvalid"},
		{"stops of another length", func() {
			NewGradientLinear(Offset{}, Offset{}, []Color{c, c}, GradientLinearOpts{ColorStops: []float64{0, 0.5, 1}})
		}, "GradientStopCountMismatch", "NewGradientLinear->newGradient->validateColorStops.GradientStopCountMismatch"},
		{"empty stops with colors", func() {
			NewGradientLinear(Offset{}, Offset{}, []Color{c, c}, GradientLinearOpts{ColorStops: []float64{}})
		}, "GradientStopCountMismatch", "NewGradientLinear->newGradient->validateColorStops.GradientStopCountMismatch"},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: no panic", tt.name)
					return
				}
				appErr, ok := errors.FromError(recoverError(t, r))
				if !ok {
					t.Errorf("%s: panic value is not an *errors.Error", tt.name)
					return
				}
				if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
					t.Errorf("%s: got %s via %s, want %s via %s", tt.name, appErr.Tag, appErr.Route(), tt.tag, tt.route)
				}
			}()
			tt.build()
		}()
	}
}

func TestGradientAcceptsStopsForAnyCount(t *testing.T) {
	c := NewColor(0xFF000000)
	g := NewGradientLinear(Offset{}, Offset{}, []Color{c, c, c, c}, GradientLinearOpts{ColorStops: []float64{0, 0.1, 0.9, 1}})
	if len(g.Colors()) != 4 || len(g.ColorStops()) != 4 {
		t.Error("four colors with four stops are fine")
	}
	// Two colors may also come with stops.
	g = NewGradientLinear(Offset{}, Offset{}, []Color{c, c}, GradientLinearOpts{ColorStops: []float64{0.2, 0.8}})
	if got := g.ColorStops(); len(got) != 2 || got[0] != 0.2 {
		t.Errorf("stops = %v", got)
	}
}

func TestImageShader(t *testing.T) {
	img := &Image{}
	m := NewMatrix4Diagonal3Values(2, 2, 1)
	s := NewImageShader(img, TileModeRepeated, TileModeMirror, m)
	if s.Image() != img || s.TileModeX() != TileModeRepeated || s.TileModeY() != TileModeMirror ||
		*s.Matrix4() != *m || s.FilterQuality() != FilterQualityUnspecified {
		t.Errorf("getters: %v %v %v %v", s.TileModeX(), s.TileModeY(), s.Matrix4(), s.FilterQuality())
	}
	m.storage[0] = 7
	if s.Matrix4().storage[0] != 2 {
		t.Error("the matrix must be copied")
	}
	// A nil matrix is the identity; Unspecified tile modes are clamp.
	d := NewImageShader(img, TileModeUnspecified, TileModeUnspecified, nil, ImageShaderOpts{FilterQuality: FilterQualityLow})
	if !d.Matrix4().IsIdentity() || d.TileModeX() != TileModeClamp || d.TileModeY() != TileModeClamp || d.FilterQuality() != FilterQualityLow {
		t.Errorf("defaults: %v %v %v %v", d.Matrix4(), d.TileModeX(), d.TileModeY(), d.FilterQuality())
	}
}

func TestShadersAreShaders(t *testing.T) {
	var _ Shader = (*Gradient)(nil)
	var _ Shader = (*ImageShader)(nil)
}
