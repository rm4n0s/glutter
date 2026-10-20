package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestElevationToShadow(t *testing.T) {
	tests := []struct {
		elevation int
		shadows   int
		firstDy   float64
	}{
		{0, 0, 0}, {1, 3, 2}, {2, 3, 3}, {3, 3, 3}, {4, 3, 2}, {6, 3, 3}, {8, 3, 5}, {9, 3, 5}, {12, 3, 7}, {16, 3, 8}, {24, 3, 11},
	}
	if len(ElevationToShadow) != len(tests) {
		t.Errorf("%d elevations, want %d", len(ElevationToShadow), len(tests))
	}
	for _, tt := range tests {
		got := ElevationToShadow[tt.elevation]
		if len(got) != tt.shadows {
			t.Errorf("elevation %d: %d shadows, want %d", tt.elevation, len(got), tt.shadows)
			continue
		}
		if tt.shadows > 0 && got[0].Offset.Dy != tt.firstDy {
			t.Errorf("elevation %d: key umbra dy = %v, want %v", tt.elevation, got[0].Offset.Dy, tt.firstDy)
		}
	}
}

func TestElevationOverlayApplySurfaceTint(t *testing.T) {
	surface := ui.NewColor(0xFFFFFFFF)
	tint := ui.NewColor(0xFFFF0000)
	transparent := ui.NewColor(0x00000000)
	tests := []struct {
		name      string
		tint      *ui.Color
		elevation float64
		want      ui.Color
	}{
		{"no tint", nil, 3, surface},
		{"transparent tint", &transparent, 3, surface},
		{"elevation 0 is 0 opacity", &tint, 0, surface},
		{"below the table", &tint, -5, surface},
		{"elevation 1", &tint, 1, ui.ColorAlphaBlend(tint.WithOpacity(0.05), surface)},
		{"elevation 3", &tint, 3, ui.ColorAlphaBlend(tint.WithOpacity(0.08), surface)},
		{"elevation 6", &tint, 6, ui.ColorAlphaBlend(tint.WithOpacity(0.11), surface)},
		{"elevation 12", &tint, 12, ui.ColorAlphaBlend(tint.WithOpacity(0.14), surface)},
		{"above the table", &tint, 100, ui.ColorAlphaBlend(tint.WithOpacity(0.14), surface)},
		{"interpolated", &tint, 2, ui.ColorAlphaBlend(tint.WithOpacity(0.065), surface)},
		{"interpolated again", &tint, 10, ui.ColorAlphaBlend(tint.WithOpacity(0.13), surface)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ElevationOverlayApplySurfaceTint(surface, tt.tint, tt.elevation); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestElevationOverlayColorWithOverlay(t *testing.T) {
	surface := ui.NewColor(0xFF121212)
	overlay := ui.NewColor(0xFFFFFFFF)
	for _, elevation := range []float64{0, 1, 4, 24} {
		opacity := (4.5*math.Log(elevation+1) + 2) / 100
		want := ui.ColorAlphaBlend(overlay.WithOpacity(opacity), surface)
		if got := ElevationOverlayColorWithOverlay(surface, overlay, elevation); got != want {
			t.Errorf("elevation %v: got %v, want %v", elevation, got, want)
		}
	}
}

func TestElevationOverlayApplyOverlay(t *testing.T) {
	dark := NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark, ApplyElevationOverlayColor: foundation.Some(true), UseMaterial3: foundation.Some(false)})
	darkNoOverlay := dark.CopyWith(ThemeDataCopy{ApplyElevationOverlayColor: foundation.Some(false)})
	light := NewThemeData(ThemeDataOpts{ApplyElevationOverlayColor: foundation.Some(true), UseMaterial3: foundation.Some(false)})
	tests := []struct {
		name      string
		theme     *ThemeData
		elevation float64
		changes   bool
	}{
		{"dark with elevation", dark, 4, true},
		{"dark without elevation", dark, 0, false},
		{"dark that does not apply", darkNoOverlay, 4, false},
		{"light", light, 4, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			surface := tt.theme.ColorScheme.Surface()
			var got ui.Color
			var overlay ui.Color
			ts.PumpWidget(materialApp(tt.theme, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				got = ElevationOverlayApplyOverlay(bc, surface, tt.elevation)
				overlay = ElevationOverlayOverlayColor(bc, tt.elevation)
				return widgets.NewSizedBox(nil)
			})))
			if (got != surface) != tt.changes {
				t.Errorf("changed = %v, want %v", got != surface, tt.changes)
			}
			if tt.changes && got != ui.ColorAlphaBlend(overlay, surface) {
				t.Errorf("got %v, want the overlay %v over the surface", got, overlay)
			}
			// A color that is not the surface color is never changed.
			other := ui.NewColor(0xFF336699)
			ts.PumpWidget(materialApp(tt.theme, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				got = ElevationOverlayApplyOverlay(bc, other, tt.elevation)
				return widgets.NewSizedBox(nil)
			})))
			if got != other {
				t.Errorf("a non surface color changed to %v", got)
			}
		})
	}
}
