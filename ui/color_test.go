package ui

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

func colorNear(t *testing.T, name string, got Color, a, r, g, b float64, space ColorSpace, eps float64) {
	t.Helper()
	if got.ColorSpace != space ||
		!near(float64(got.A), a, eps) || !near(float64(got.R), r, eps) ||
		!near(float64(got.G), g, eps) || !near(float64(got.B), b, eps) {
		t.Errorf("%s: got %v, want alpha %v red %v green %v blue %v in %v", name, got, a, r, g, b, space)
	}
}

func TestColorConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  Color
		a    float64
		r, g float64
		b    float64
	}{
		{"value", NewColor(0xFF6750A4), 1, 0x67 / 255.0, 0x50 / 255.0, 0xA4 / 255.0},
		{"value with alpha", NewColor(0x80FF0000), 0x80 / 255.0, 1, 0, 0},
		{"transparent", NewColor(0x00000000), 0, 0, 0, 0},
		{"ARGB", NewColorFromARGB(255, 103, 80, 164), 1, 0x67 / 255.0, 0x50 / 255.0, 0xA4 / 255.0},
		{"ARGB uses only the low 8 bits", NewColorFromARGB(0x1FF, 0x167, 0x150, 0x1A4), 1, 0x67 / 255.0, 0x50 / 255.0, 0xA4 / 255.0},
		{"RGBO", NewColorFromRGBO(255, 0, 0, 0.5), 0.5, 1, 0, 0},
		{"from", NewColorFrom(0.25, 0.5, 0.75, 1), 0.25, 0.5, 0.75, 1},
	}
	for _, tt := range tests {
		colorNear(t, tt.name, tt.got, tt.a, tt.r, tt.g, tt.b, ColorSpaceSRGB, 1e-7)
	}
	if c := NewColorFrom(1, 2, -1, 0, ColorFromOpts{ColorSpace: ColorSpaceExtendedSRGB}); c.ColorSpace != ColorSpaceExtendedSRGB || c.R != 2 || c.G != -1 {
		t.Errorf("an extended sRGB color keeps values outside 0 to 1: %v", c)
	}
	if (Color{}) != (Color{A: 0, R: 0, G: 0, B: 0, ColorSpace: ColorSpaceSRGB}) {
		t.Error("the zero Color is transparent black in sRGB")
	}
}

func TestColorToARGB32AndChannels(t *testing.T) {
	tests := []struct {
		name  string
		c     Color
		want  uint32
		alpha int
	}{
		{"round trip", NewColor(0xFF6750A4), 0xFF6750A4, 255},
		{"half alpha", NewColor(0x80FF0000), 0x80FF0000, 128},
		{"from floats rounds", NewColorFrom(0.5, 0, 0, 0), 0x80000000, 128}, // 127.5 rounds up
		{"clamps above 1", NewColorFrom(1, 2, 0, 0, ColorFromOpts{ColorSpace: ColorSpaceExtendedSRGB}), 0xFFFF0000, 255},
		{"clamps below 0", NewColorFrom(1, -1, 0, 0, ColorFromOpts{ColorSpace: ColorSpaceExtendedSRGB}), 0xFF000000, 255},
		{"NaN is 0", Color{A: float32(math.NaN()), R: 1}, 0x00FF0000, 0},
	}
	for _, tt := range tests {
		if got := tt.c.ToARGB32(); got != tt.want {
			t.Errorf("%s: ToARGB32 = %#08x, want %#08x", tt.name, got, tt.want)
		}
		if got := tt.c.Alpha(); got != tt.alpha {
			t.Errorf("%s: Alpha = %d, want %d", tt.name, got, tt.alpha)
		}
	}
	c := NewColor(0xFF6750A4)
	if c.Red() != 0x67 || c.Green() != 0x50 || c.Blue() != 0xA4 || c.Opacity() != 1 {
		t.Errorf("channels = %d %d %d %v", c.Red(), c.Green(), c.Blue(), c.Opacity())
	}
	if got := NewColor(0x80000000).Opacity(); got != 128.0/255 {
		t.Errorf("Opacity = %v", got)
	}
}

func TestColorWith(t *testing.T) {
	c := NewColor(0xFF6750A4)
	p3 := ColorSpaceDisplayP3
	tests := []struct {
		name string
		got  Color
		want Color
	}{
		{"withAlpha", c.WithAlpha(0x80), NewColor(0x806750A4)},
		{"withRed", c.WithRed(0x11), NewColor(0xFF1150A4)},
		{"withGreen", c.WithGreen(0x22), NewColor(0xFF6722A4)},
		{"withBlue", c.WithBlue(0x33), NewColor(0xFF675033)},
		{"withOpacity", c.WithOpacity(0.5), NewColor(0x806750A4)},
		{"withValues alpha", c.WithValues(ColorWithValuesOpts{Alpha: Ptr(0.5)}), Color{A: 0.5, R: c.R, G: c.G, B: c.B}},
		{"withValues all", c.WithValues(ColorWithValuesOpts{Alpha: Ptr(0.1), Red: Ptr(0.2), Green: Ptr(0.3), Blue: Ptr(0.4)}), NewColorFrom(0.1, 0.2, 0.3, 0.4)},
		{"withValues nothing", c.WithValues(), c},
		{"withValues the same space is a no-op", c.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceSRGB)}), c},
		// Flutter's identity transform returns the color as it is, space included.
		{"sRGB to extended sRGB keeps the values", c.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceExtendedSRGB)}), c},
	}
	for _, tt := range tests {
		// Colors that went through 8 bits and back differ in the last float32 bit.
		colorNear(t, tt.name, tt.got, float64(tt.want.A), float64(tt.want.R), float64(tt.want.G), float64(tt.want.B), tt.want.ColorSpace, 1e-6)
	}
	// Computed with Dart's formulas in double precision.
	got := c.WithValues(ColorWithValuesOpts{ColorSpace: &p3})
	colorNear(t, "sRGB to Display P3", got, 1, 0.38974667416555575, 0.31722501226629723, 0.6230728893043498, ColorSpaceDisplayP3, 1e-6)
	red := NewColor(0xFFFF0000).WithValues(ColorWithValuesOpts{ColorSpace: &p3})
	colorNear(t, "red to Display P3", red, 1, 0.9174876712728796, 0.20028681142289528, 0.13855100347019958, ColorSpaceDisplayP3, 1e-6)
	// Display P3 to extended sRGB goes through the inverse matrix.
	ext := ColorSpaceExtendedSRGB
	back := NewColorFrom(1, 0x67/255.0, 0x50/255.0, 0xA4/255.0, ColorFromOpts{ColorSpace: ColorSpaceDisplayP3}).WithValues(ColorWithValuesOpts{ColorSpace: &ext})
	colorNear(t, "Display P3 to extended sRGB", back, 1, 0.42104500023259067, 0.30922373783828966, 0.6642488774866745, ColorSpaceExtendedSRGB, 1e-6)
	// Extended sRGB to sRGB clamps.
	wide := NewColorFrom(1.5, 2, -1, 0.5, ColorFromOpts{ColorSpace: ColorSpaceExtendedSRGB})
	srgb := wide.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceSRGB)})
	colorNear(t, "extended sRGB to sRGB", srgb, 1, 1, 0, 0.5, ColorSpaceSRGB, 0)
	// Display P3 to sRGB only clamps in Flutter (its _ClampTransform ignores its child).
	clamped := NewColorFrom(1, 1.2, 0.5, -0.1, ColorFromOpts{ColorSpace: ColorSpaceDisplayP3}).WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceSRGB)})
	colorNear(t, "Display P3 to sRGB", clamped, 1, 1, 0.5, 0, ColorSpaceSRGB, 0)
}

func TestColorComputeLuminance(t *testing.T) {
	tests := []struct {
		name string
		c    Color
		want float64 // from Dart's formula
	}{
		{"black", NewColor(0xFF000000), 0},
		{"white", NewColor(0xFFFFFFFF), 1},
		{"red", NewColor(0xFFFF0000), 0.2126},
		{"green", NewColor(0xFF00FF00), 0.7152},
		{"blue", NewColor(0xFF0000FF), 0.0722},
		{"gray", NewColor(0xFF808080), 0.21586050011389923},
		{"material purple", NewColor(0xFF6750A4), 0.11301222190383803},
		{"yellow", NewColor(0xFFFFEB3B), 0.8099242849815864},
		{"alpha is ignored", NewColor(0x00FFFFFF), 1},
	}
	for _, tt := range tests {
		if got := tt.c.ComputeLuminance(); !near(got, tt.want, 1e-6) {
			t.Errorf("%s: luminance = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestColorLerp(t *testing.T) {
	a, b := NewColor(0xFF6750A4), NewColor(0x80FFEB3B)
	// Expected values computed with Dart's formula: x*(1-t) + y*t, clamped.
	colorNear(t, "t = 0.3", a.Lerp(b, 0.3), 0.8505882352941176, 0.5827450980392157, 0.4960784313725489, 0.5196078431372549, ColorSpaceSRGB, 1e-6)
	if got := a.Lerp(b, 0); got != a {
		t.Errorf("t = 0: got %v, want %v", got, a)
	}
	if got := a.Lerp(b, 1); got != b {
		t.Errorf("t = 1: got %v, want %v", got, b)
	}
	if got := a.Lerp(b, 3); got.A < 0 || got.A > 1 || got.R > 1 || got.G > 1 || got.B < 0 {
		t.Errorf("components are clamped: %v", got)
	}
	// Different color spaces meet in the wider one.
	p3 := a.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceDisplayP3)})
	mixed := a.Lerp(p3, 0.5)
	if mixed.ColorSpace != ColorSpaceDisplayP3 {
		t.Errorf("lerp of sRGB and Display P3 is in %v, want Display P3", mixed.ColorSpace)
	}
	colorNear(t, "same color in two spaces", mixed, float64(p3.A), float64(p3.R), float64(p3.G), float64(p3.B), ColorSpaceDisplayP3, 1e-5)
	if got := a.Lerp(a.WithValues(ColorWithValuesOpts{ColorSpace: Ptr(ColorSpaceExtendedSRGB)}), 0.5); got.ColorSpace != ColorSpaceSRGB {
		t.Errorf("sRGB and extended sRGB meet in %v, want sRGB (the first)", got.ColorSpace)
	}
}

func TestColorLerpWithNil(t *testing.T) {
	a := NewColor(0xFF6750A4)
	if ColorLerp(nil, nil, 0.5) != nil {
		t.Error("nil, nil must give nil")
	}
	if got := ColorLerp(&a, nil, 0.25); got == nil || !near(float64(got.A), 0.75, 1e-7) || got.R != a.R {
		t.Errorf("a, nil = %v: only the alpha is scaled", got)
	}
	if got := ColorLerp(nil, &a, 0.25); got == nil || !near(float64(got.A), 0.25, 1e-7) || got.G != a.G {
		t.Errorf("nil, a = %v: only the alpha is scaled", got)
	}
	b := NewColor(0xFF000000)
	if got := ColorLerp(&a, &b, 1); got == nil || *got != b {
		t.Errorf("a, b at 1 = %v", got)
	}
}

func TestColorAlphaBlend(t *testing.T) {
	fg, opaque, half := NewColor(0x80FF0000), NewColor(0xFF0000FF), NewColor(0x800000FF)
	tests := []struct {
		name       string
		fg, bg     Color
		a, r, g, b float64
	}{
		// Computed with Dart's formulas.
		{"opaque background", fg, opaque, 1, 0.5019607843137255, 0, 0.4980392156862745},
		{"translucent background", fg, half, 0.7519569396386006, 0.6675392670157068, 0, 0.3324607329842932},
		{"transparent foreground gives the background", NewColor(0x00FF0000), opaque, 1, 0, 0, 1},
		{"opaque foreground gives the foreground", NewColor(0xFFFF0000), opaque, 1, 1, 0, 0},
	}
	for _, tt := range tests {
		colorNear(t, tt.name, ColorAlphaBlend(tt.fg, tt.bg), tt.a, tt.r, tt.g, tt.b, ColorSpaceSRGB, 1e-6)
	}
	if got := ColorAlphaBlend(NewColor(0x00FF0000), opaque); got != opaque {
		t.Errorf("a transparent foreground must return the background untouched: %v", got)
	}
}

func TestColorGetAlphaFromOpacity(t *testing.T) {
	tests := []struct {
		opacity float64
		want    int
	}{{0, 0}, {1, 255}, {0.5, 128}, {0.2, 51}, {-1, 0}, {2, 255}, {1.0 / 255, 1}}
	for _, tt := range tests {
		if got := ColorGetAlphaFromOpacity(tt.opacity); got != tt.want {
			t.Errorf("ColorGetAlphaFromOpacity(%v) = %d, want %d", tt.opacity, got, tt.want)
		}
	}
}

func TestColorString(t *testing.T) {
	tests := []struct {
		c    Color
		want string
	}{
		{NewColor(0xFF000000), "Color(alpha: 1.0000, red: 0.0000, green: 0.0000, blue: 0.0000, colorSpace: ColorSpace.sRGB)"},
		{NewColor(0x80FF0000), "Color(alpha: 0.5020, red: 1.0000, green: 0.0000, blue: 0.0000, colorSpace: ColorSpace.sRGB)"},
		{NewColorFrom(1, 0.5, 0.25, 0.125, ColorFromOpts{ColorSpace: ColorSpaceDisplayP3}), "Color(alpha: 1.0000, red: 0.5000, green: 0.2500, blue: 0.1250, colorSpace: ColorSpace.displayP3)"},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestColorOptionsPanics(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(recoverError(t, recover()))
		if !ok || appErr.Tag != "MultipleOptions" {
			t.Errorf("expected MultipleOptions, got %v", appErr)
		}
	}()
	NewColor(0).WithValues(ColorWithValuesOpts{}, ColorWithValuesOpts{})
}

// recoverError returns the recovered panic value as an error.
func recoverError(t *testing.T, r any) error {
	t.Helper()
	err, ok := r.(error)
	if !ok {
		t.Fatalf("panic value %v (%T) is not an error", r, r)
	}
	return err
}
