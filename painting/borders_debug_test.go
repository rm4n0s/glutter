//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestBorderSideDebugPanics(t *testing.T) {
	f := newSideFixtures()
	t.Run("negative width", func(t *testing.T) {
		requireTaggedPanic(t, "BorderSideWidthNegative", func() { sideW(-1) })
	})
	for name, pair := range map[string][2]BorderSide{
		"green blue": {f.green, f.blue}, "green2 blue2": {f.green2, f.blue2}, "green2 none2": {f.green2, f.none2},
		"none3 side2": {f.none3, f.side2}, "side2 none3": {f.side2, f.none3}, "solid none": {f.solid, f.none},
	} {
		t.Run("merge "+name, func(t *testing.T) {
			requireTaggedPanic(t, "BorderSideCannotMerge", func() { BorderSideMerge(pair[0], pair[1]) })
		})
	}
	t.Run("zero width ok", func(t *testing.T) {
		if got := side(BorderSideOpts{Width: foundation.Some(0.0)}).Width; got != 0 {
			t.Errorf("width = %v", got)
		}
	})
}

func TestGradientDebugPanics(t *testing.T) {
	requireTaggedPanic(t, "GradientColorsTooFew", func() { impliedStops(colorsOf(0xFF000000), nil) })
	requireTaggedPanic(t, "GradientInterpolationInvalid", func() {
		interpolateColorsAndStops(colorsOf(0xFF000000), []float64{0}, colorsOf(0xFF000000, 0xFFFFFFFF), []float64{0, 1}, 0.5)
	})
}

func TestColorComponentsDebugPanics(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{"hsv alpha", func() { NewHSVColorFromAHSV(1.5, 0, 0, 0) }},
		{"hsv hue", func() { NewHSVColorFromAHSV(1, 361, 0, 0) }},
		{"hsl saturation", func() { NewHSLColorFromAHSL(1, 0, -0.1, 0) }},
		{"hsl lightness", func() { NewHSLColorFromAHSL(1, 0, 0, 2) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, "ColorComponentOutOfRange", tt.call)
		})
	}
}
