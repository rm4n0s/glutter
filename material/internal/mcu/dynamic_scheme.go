package mcu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// DynamicScheme is a color scheme: the six palettes that the colors of Material 3
// are drawn from, whether it is dark, and its contrast level. The colors are the
// MaterialDynamicColors, read with their getters (Primary, Surface...) or with
// DynamicColor.GetArgb. A scheme remembers the colors it has made, so it is not
// safe for concurrent use.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme
type DynamicScheme struct {
	// SourceColorArgb is the source color, as ARGB.
	SourceColorArgb uint32
	// SourceColorHct is the source color.
	SourceColorHct *Hct
	// Variant is the type of the scheme.
	Variant Variant
	// IsDark says that the scheme is a dark one.
	IsDark bool
	// ContrastLevel is from -1 (the lowest contrast) to 1 (the highest); 0 is the
	// standard.
	ContrastLevel float64
	// PrimaryPalette is the primary palette.
	PrimaryPalette *TonalPalette
	// SecondaryPalette is the secondary palette.
	SecondaryPalette *TonalPalette
	// TertiaryPalette is the tertiary palette.
	TertiaryPalette *TonalPalette
	// NeutralPalette is the neutral palette.
	NeutralPalette *TonalPalette
	// NeutralVariantPalette is the neutral variant palette.
	NeutralVariantPalette *TonalPalette
	// ErrorPalette is the error palette.
	ErrorPalette *TonalPalette

	hctCache map[*DynamicColor]*Hct
}

// DynamicSchemeOpts are the parameters of NewDynamicScheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.new
type DynamicSchemeOpts struct {
	// SourceColorHct is the source color. It is required.
	SourceColorHct *Hct
	// Variant is the type of the scheme.
	Variant Variant
	// ContrastLevel is from -1 to 1; the default is 0.
	ContrastLevel float64
	// IsDark says that the scheme is a dark one.
	IsDark bool
	// PrimaryPalette, SecondaryPalette, TertiaryPalette, NeutralPalette and
	// NeutralVariantPalette are required.
	PrimaryPalette, SecondaryPalette, TertiaryPalette, NeutralPalette, NeutralVariantPalette *TonalPalette
	// ErrorPalette is the error palette; the default is TonalPalette.of(25, 84).
	ErrorPalette *TonalPalette
}

// NewDynamicScheme creates a scheme. In debug builds it panics with the tag
// DynamicSchemeIncomplete when the source color or a palette is missing.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.new
func NewDynamicScheme(o DynamicSchemeOpts) *DynamicScheme {
	if foundation.DebugMode && (o.SourceColorHct == nil || o.PrimaryPalette == nil || o.SecondaryPalette == nil ||
		o.TertiaryPalette == nil || o.NeutralPalette == nil || o.NeutralVariantPalette == nil) {
		panic(errors.New("DynamicSchemeIncomplete", "a scheme needs a source color and its five palettes"))
	}
	if o.ErrorPalette == nil {
		o.ErrorPalette = TonalPaletteOf(25.0, 84.0)
	}
	return &DynamicScheme{
		SourceColorArgb: o.SourceColorHct.ToInt(), SourceColorHct: o.SourceColorHct, Variant: o.Variant,
		IsDark: o.IsDark, ContrastLevel: o.ContrastLevel,
		PrimaryPalette: o.PrimaryPalette, SecondaryPalette: o.SecondaryPalette, TertiaryPalette: o.TertiaryPalette,
		NeutralPalette: o.NeutralPalette, NeutralVariantPalette: o.NeutralVariantPalette, ErrorPalette: o.ErrorPalette,
	}
}

// GetRotatedHue is the hue of the source color rotated by the rotation of the
// range of hues that it is in: hues[i] < hue < hues[i+1] is rotated by
// rotations[i]. With one rotation it is the source hue plus it. In debug builds
// it panics with the tag DynamicSchemeRotations when the lists differ in length.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.getRotatedHue
func GetRotatedHue(sourceColor *Hct, hues, rotations []float64) float64 {
	if foundation.DebugMode && len(hues) != len(rotations) {
		panic(errors.New("DynamicSchemeRotations", "hues and rotations must have the same length", "hues", len(hues), "rotations", len(rotations)))
	}
	sourceHue := sourceColor.Hue()
	if len(rotations) == 1 {
		return SanitizeDegreesDouble(sourceColor.Hue() + rotations[0])
	}
	for i := 0; i <= len(hues)-2; i++ {
		if hues[i] < sourceHue && sourceHue < hues[i+1] {
			return SanitizeDegreesDouble(sourceHue + rotations[i])
		}
	}
	// If this executes, something is wrong: a rotation should have been found.
	return sourceHue
}

// GetHct is the color of the dynamic color in the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.getHct
func (s *DynamicScheme) GetHct(c *DynamicColor) *Hct { return c.GetHct(s) }

// GetArgb is the ARGB color of the dynamic color in the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.getArgb
func (s *DynamicScheme) GetArgb(c *DynamicColor) uint32 { return c.GetArgb(s) }
