package mcu

import "math"

// The dynamic scheme variants. Each one makes a DynamicScheme of its Variant from
// the source color, whether it is dark, and the contrast level (from -1 to 1).

// NewSchemeContent makes a scheme whose palettes and tokens match the source
// color, with the primary container the source color adjusted for contrast, and
// the tertiary palette an analogue of it (found by dividing the wheel by 6).
//
// Flutter: material-color-utilities scheme/scheme_content.dart SchemeContent
func NewSchemeContent(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantContent, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette: TonalPaletteOf(sourceColorHct.Hue(), sourceColorHct.Chroma()),
		SecondaryPalette: TonalPaletteOf(sourceColorHct.Hue(),
			math.Max(sourceColorHct.Chroma()-32.0, sourceColorHct.Chroma()*0.5)),
		TertiaryPalette: TonalPaletteFromHct(FixIfDisliked(
			NewTemperatureCache(sourceColorHct).Analogous(3, 6)[2])),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), sourceColorHct.Chroma()/8.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), (sourceColorHct.Chroma()/8.0)+4.0),
	})
}

var (
	expressiveHues               = []float64{0, 21, 51, 121, 151, 191, 271, 321, 360}
	expressiveSecondaryRotations = []float64{45, 95, 45, 20, 45, 90, 45, 45, 45}
	expressiveTertiaryRotations  = []float64{120, 120, 20, 45, 20, 15, 20, 120, 120}
)

// NewSchemeExpressive makes a scheme of pastel colors and medium chroma palettes,
// whose primary hue is different from the source color, for variety.
//
// Flutter: material-color-utilities scheme/scheme_expressive.dart SchemeExpressive
func NewSchemeExpressive(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantExpressive, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(SanitizeDegreesDouble(sourceColorHct.Hue()+240.0), 40.0),
		SecondaryPalette:      TonalPaletteOf(GetRotatedHue(sourceColorHct, expressiveHues, expressiveSecondaryRotations), 24.0),
		TertiaryPalette:       TonalPaletteOf(GetRotatedHue(sourceColorHct, expressiveHues, expressiveTertiaryRotations), 32.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue()+15.0, 8.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue()+15.0, 12.0),
	})
}

// NewSchemeFidelity makes a scheme whose palettes and tokens match the source
// color, with the primary container the source color adjusted for contrast, and
// the tertiary palette its complement.
//
// Flutter: material-color-utilities scheme/scheme_fidelity.dart SchemeFidelity
func NewSchemeFidelity(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantFidelity, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette: TonalPaletteOf(sourceColorHct.Hue(), sourceColorHct.Chroma()),
		SecondaryPalette: TonalPaletteOf(sourceColorHct.Hue(),
			math.Max(sourceColorHct.Chroma()-32.0, sourceColorHct.Chroma()*0.5)),
		TertiaryPalette:       TonalPaletteFromHct(FixIfDisliked(NewTemperatureCache(sourceColorHct).Complement())),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), sourceColorHct.Chroma()/8.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), (sourceColorHct.Chroma()/8.0)+4.0),
	})
}

// NewSchemeFruitSalad makes a playful scheme in which the hue of the source color
// does not appear.
//
// Flutter: material-color-utilities scheme/scheme_fruit_salad.dart SchemeFruitSalad
func NewSchemeFruitSalad(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantFruitSalad, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(SanitizeDegreesDouble(sourceColorHct.Hue()-50.0), 48.0),
		SecondaryPalette:      TonalPaletteOf(SanitizeDegreesDouble(sourceColorHct.Hue()-50.0), 36.0),
		TertiaryPalette:       TonalPaletteOf(sourceColorHct.Hue(), 36.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 10.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 16.0),
	})
}

// NewSchemeMonochrome makes a scheme in which all colors are grayscale.
//
// Flutter: material-color-utilities scheme/scheme_monochrome.dart SchemeMonochrome
func NewSchemeMonochrome(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantMonochrome, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(sourceColorHct.Hue(), 0.0),
		SecondaryPalette:      TonalPaletteOf(sourceColorHct.Hue(), 0.0),
		TertiaryPalette:       TonalPaletteOf(sourceColorHct.Hue(), 0.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 0.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 0.0),
	})
}

// NewSchemeNeutral makes a scheme that is close to grayscale, with a hint of
// chroma.
//
// Flutter: material-color-utilities scheme/scheme_neutral.dart SchemeNeutral
func NewSchemeNeutral(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantNeutral, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(sourceColorHct.Hue(), 12.0),
		SecondaryPalette:      TonalPaletteOf(sourceColorHct.Hue(), 8.0),
		TertiaryPalette:       TonalPaletteOf(sourceColorHct.Hue(), 16.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 2.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 2.0),
	})
}

// NewSchemeRainbow makes a playful scheme in which the hue of the source color
// does not appear in the secondary and the neutral palettes.
//
// Flutter: material-color-utilities scheme/scheme_rainbow.dart SchemeRainbow
func NewSchemeRainbow(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantRainbow, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(sourceColorHct.Hue(), 48.0),
		SecondaryPalette:      TonalPaletteOf(sourceColorHct.Hue(), 16.0),
		TertiaryPalette:       TonalPaletteOf(SanitizeDegreesDouble(sourceColorHct.Hue()+60.0), 24.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 0.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 0.0),
	})
}

// NewSchemeTonalSpot makes the default Material You scheme of 2021: pastel tokens
// and low chroma palettes.
//
// Flutter: material-color-utilities scheme/scheme_tonal_spot.dart SchemeTonalSpot
func NewSchemeTonalSpot(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantTonalSpot, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(sourceColorHct.Hue(), 36.0),
		SecondaryPalette:      TonalPaletteOf(sourceColorHct.Hue(), 16.0),
		TertiaryPalette:       TonalPaletteOf(SanitizeDegreesDouble(sourceColorHct.Hue()+60.0), 24.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 6.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 8.0),
	})
}

var (
	vibrantHues               = []float64{0, 41, 61, 101, 131, 181, 251, 301, 360}
	vibrantSecondaryRotations = []float64{18, 15, 10, 12, 15, 18, 15, 12, 12}
	vibrantTertiaryRotations  = []float64{35, 30, 20, 25, 30, 35, 30, 25, 25}
)

// NewSchemeVibrant makes a scheme of pastel colors and high chroma palettes: the
// chroma of the primary palette is at its maximum.
//
// Flutter: material-color-utilities scheme/scheme_vibrant.dart SchemeVibrant
func NewSchemeVibrant(sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	return NewDynamicScheme(DynamicSchemeOpts{
		SourceColorHct: sourceColorHct, Variant: VariantVibrant, IsDark: isDark, ContrastLevel: contrastLevel,
		PrimaryPalette:        TonalPaletteOf(sourceColorHct.Hue(), 200.0),
		SecondaryPalette:      TonalPaletteOf(GetRotatedHue(sourceColorHct, vibrantHues, vibrantSecondaryRotations), 24.0),
		TertiaryPalette:       TonalPaletteOf(GetRotatedHue(sourceColorHct, vibrantHues, vibrantTertiaryRotations), 32.0),
		NeutralPalette:        TonalPaletteOf(sourceColorHct.Hue(), 10.0),
		NeutralVariantPalette: TonalPaletteOf(sourceColorHct.Hue(), 12.0),
	})
}

// NewSchemeFromVariant makes the scheme of the variant, as the test helper
// schemeFromVariant of the Dart package does. VariantTonalSpot, and
// VariantUnspecified, make a tonal spot scheme.
//
// Flutter: material-color-utilities test/utils/scheme_from_variant.dart schemeFromVariant
func NewSchemeFromVariant(variant Variant, sourceColorHct *Hct, isDark bool, contrastLevel float64) *DynamicScheme {
	switch variant {
	case VariantContent:
		return NewSchemeContent(sourceColorHct, isDark, contrastLevel)
	case VariantExpressive:
		return NewSchemeExpressive(sourceColorHct, isDark, contrastLevel)
	case VariantFidelity:
		return NewSchemeFidelity(sourceColorHct, isDark, contrastLevel)
	case VariantFruitSalad:
		return NewSchemeFruitSalad(sourceColorHct, isDark, contrastLevel)
	case VariantMonochrome:
		return NewSchemeMonochrome(sourceColorHct, isDark, contrastLevel)
	case VariantNeutral:
		return NewSchemeNeutral(sourceColorHct, isDark, contrastLevel)
	case VariantRainbow:
		return NewSchemeRainbow(sourceColorHct, isDark, contrastLevel)
	case VariantVibrant:
		return NewSchemeVibrant(sourceColorHct, isDark, contrastLevel)
	}
	return NewSchemeTonalSpot(sourceColorHct, isDark, contrastLevel)
}
