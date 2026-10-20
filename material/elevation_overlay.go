package material

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// elevationOpacity holds the opacity at a given elevation.
//
// Flutter: material/elevation_overlay.dart _ElevationOpacity
type elevationOpacity struct{ elevation, opacity float64 }

// surfaceTintElevationOpacities are the surface tint opacities based on
// elevations according to the Material Design 3 specification:
// https://m3.material.io/styles/color/the-color-system/color-roles. They are
// ordered by increasing elevation.
//
// Flutter: material/elevation_overlay.dart _surfaceTintElevationOpacities
var surfaceTintElevationOpacities = [...]elevationOpacity{
	{0.0, 0.0},   // Elevation level 0
	{1.0, 0.05},  // Elevation level 1
	{3.0, 0.08},  // Elevation level 2
	{6.0, 0.11},  // Elevation level 3
	{8.0, 0.12},  // Elevation level 4
	{12.0, 0.14}, // Elevation level 5
}

// ElevationOverlayApplySurfaceTint applies a surface tint color to a given
// surface color, based on the surface's elevation: a nil tint, or a transparent
// one, leaves the color as it is. The tint is blended over the color with an
// opacity that is looked up in the table of the Material 3 specification and
// interpolated between its entries.
//
// Flutter: material/elevation_overlay.dart ElevationOverlay.applySurfaceTint
func ElevationOverlayApplySurfaceTint(color ui.Color, surfaceTint *ui.Color, elevation float64) ui.Color {
	if surfaceTint != nil && *surfaceTint != Colors.Transparent {
		return ui.ColorAlphaBlend(surfaceTint.WithOpacity(surfaceTintOpacityForElevation(elevation)), color)
	}
	return color
}

// surfaceTintOpacityForElevation calculates the opacity of the surface tint
// color from the elevation by looking it up in the token generated table of
// opacities, interpolating between values as needed. If the elevation is
// outside the range of values in the table it clamps to the smallest or the
// largest opacity.
//
// Flutter: material/elevation_overlay.dart ElevationOverlay._surfaceTintOpacityForElevation
func surfaceTintOpacityForElevation(elevation float64) float64 {
	table := surfaceTintElevationOpacities[:]
	if elevation < table[0].elevation {
		return table[0].opacity
	}
	// Walk the opacity list and find the closest match(es) for the elevation.
	index := 0
	for elevation >= table[index].elevation {
		// If we found it exactly or walked off the end of the list just return it.
		if elevation == table[index].elevation || index+1 == len(table) {
			return table[index].opacity
		}
		index++
	}
	// Interpolate between the two opacity values.
	lower, upper := table[index-1], table[index]
	t := (elevation - lower.elevation) / (upper.elevation - lower.elevation)
	return lower.opacity + t*(upper.opacity-lower.opacity)
}

// ElevationOverlayApplyOverlay applies an overlay color to a surface color to
// indicate the level of its elevation in a dark theme. It only does so for a
// surface at an elevation above zero, in a dark theme that applies the
// overlay, whose color is the surface color of the theme (ignoring alpha).
//
// Flutter: material/elevation_overlay.dart ElevationOverlay.applyOverlay
func ElevationOverlayApplyOverlay(bc widgets.BuildContext, color ui.Color, elevation float64) ui.Color {
	theme := ThemeOf(bc)
	if elevation > 0.0 && theme.ApplyElevationOverlayColor && theme.Brightness() == ui.BrightnessDark &&
		color.WithOpacity(1.0) == theme.ColorScheme.Surface().WithOpacity(1.0) {
		return ElevationOverlayColorWithOverlay(color, theme.ColorScheme.OnSurface(), elevation)
	}
	return color
}

// ElevationOverlayOverlayColor computes the appropriate overlay color used to
// indicate elevation in dark themes: the on-surface color of the theme with an
// opacity from the elevation.
//
// Flutter: material/elevation_overlay.dart ElevationOverlay.overlayColor
func ElevationOverlayOverlayColor(bc widgets.BuildContext, elevation float64) ui.Color {
	return overlayColor(ThemeOf(bc).ColorScheme.OnSurface(), elevation)
}

// ElevationOverlayColorWithOverlay blends an overlay color over the surface
// color with an opacity from the elevation.
//
// Flutter: material/elevation_overlay.dart ElevationOverlay.colorWithOverlay
func ElevationOverlayColorWithOverlay(surface, overlay ui.Color, elevation float64) ui.Color {
	return ui.ColorAlphaBlend(overlayColor(overlay, elevation), surface)
}

// overlayColor computes the opacity for the given elevation with the formula
// of the specification:
// https://material.io/design/color/dark-theme.html#properties
//
// Flutter: material/elevation_overlay.dart ElevationOverlay._overlayColor
func overlayColor(color ui.Color, elevation float64) ui.Color {
	opacity := (4.5*math.Log(elevation+1) + 2) / 100.0
	return color.WithOpacity(opacity)
}
