package mcu

import "math"

// ContentAccentToneDelta is the tone difference of the accents of a content
// scheme.
//
// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.contentAccentToneDelta
const ContentAccentToneDelta = 15.0

func isFidelity(s *DynamicScheme) bool {
	return s.Variant == VariantFidelity || s.Variant == VariantContent
}

func isMonochrome(s *DynamicScheme) bool { return s.Variant == VariantMonochrome }

// HighestSurface is the surface that is the farthest from the background in the
// scheme: the brightest surface of a dark scheme and the dimmest of a light one.
//
// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.highestSurface
func HighestSurface(s *DynamicScheme) *DynamicColor {
	if s.IsDark {
		return SurfaceBright
	}
	return SurfaceDim
}

// The dynamic colors of Material 3: the colors that a color scheme has, with the
// tone and the contrast that each one has in a scheme. They are read-only, and
// are made when the package is initialized, because they refer to each other.
var (
	// PrimaryPaletteKeyColor is the dynamic color primaryPaletteKeyColor of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.primaryPaletteKeyColor
	PrimaryPaletteKeyColor *DynamicColor
	// SecondaryPaletteKeyColor is the dynamic color secondaryPaletteKeyColor of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.secondaryPaletteKeyColor
	SecondaryPaletteKeyColor *DynamicColor
	// TertiaryPaletteKeyColor is the dynamic color tertiaryPaletteKeyColor of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.tertiaryPaletteKeyColor
	TertiaryPaletteKeyColor *DynamicColor
	// NeutralPaletteKeyColor is the dynamic color neutralPaletteKeyColor of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.neutralPaletteKeyColor
	NeutralPaletteKeyColor *DynamicColor
	// NeutralVariantPaletteKeyColor is the dynamic color neutralVariantPaletteKeyColor of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.neutralVariantPaletteKeyColor
	NeutralVariantPaletteKeyColor *DynamicColor
	// Background is the dynamic color background of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.background
	Background *DynamicColor
	// OnBackground is the dynamic color onBackground of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onBackground
	OnBackground *DynamicColor
	// Surface is the dynamic color surface of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surface
	Surface *DynamicColor
	// SurfaceDim is the dynamic color surfaceDim of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceDim
	SurfaceDim *DynamicColor
	// SurfaceBright is the dynamic color surfaceBright of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceBright
	SurfaceBright *DynamicColor
	// SurfaceContainerLowest is the dynamic color surfaceContainerLowest of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceContainerLowest
	SurfaceContainerLowest *DynamicColor
	// SurfaceContainerLow is the dynamic color surfaceContainerLow of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceContainerLow
	SurfaceContainerLow *DynamicColor
	// SurfaceContainer is the dynamic color surfaceContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceContainer
	SurfaceContainer *DynamicColor
	// SurfaceContainerHigh is the dynamic color surfaceContainerHigh of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceContainerHigh
	SurfaceContainerHigh *DynamicColor
	// SurfaceContainerHighest is the dynamic color surfaceContainerHighest of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceContainerHighest
	SurfaceContainerHighest *DynamicColor
	// OnSurface is the dynamic color onSurface of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSurface
	OnSurface *DynamicColor
	// SurfaceVariant is the dynamic color surfaceVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceVariant
	SurfaceVariant *DynamicColor
	// OnSurfaceVariant is the dynamic color onSurfaceVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSurfaceVariant
	OnSurfaceVariant *DynamicColor
	// InverseSurface is the dynamic color inverseSurface of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.inverseSurface
	InverseSurface *DynamicColor
	// InverseOnSurface is the dynamic color inverseOnSurface of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.inverseOnSurface
	InverseOnSurface *DynamicColor
	// Outline is the dynamic color outline of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.outline
	Outline *DynamicColor
	// OutlineVariant is the dynamic color outlineVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.outlineVariant
	OutlineVariant *DynamicColor
	// Shadow is the dynamic color shadow of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.shadow
	Shadow *DynamicColor
	// Scrim is the dynamic color scrim of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.scrim
	Scrim *DynamicColor
	// SurfaceTint is the dynamic color surfaceTint of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.surfaceTint
	SurfaceTint *DynamicColor
	// Primary is the dynamic color primary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.primary
	Primary *DynamicColor
	// OnPrimary is the dynamic color onPrimary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onPrimary
	OnPrimary *DynamicColor
	// PrimaryContainer is the dynamic color primaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.primaryContainer
	PrimaryContainer *DynamicColor
	// OnPrimaryContainer is the dynamic color onPrimaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onPrimaryContainer
	OnPrimaryContainer *DynamicColor
	// InversePrimary is the dynamic color inversePrimary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.inversePrimary
	InversePrimary *DynamicColor
	// Secondary is the dynamic color secondary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.secondary
	Secondary *DynamicColor
	// OnSecondary is the dynamic color onSecondary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSecondary
	OnSecondary *DynamicColor
	// SecondaryContainer is the dynamic color secondaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.secondaryContainer
	SecondaryContainer *DynamicColor
	// OnSecondaryContainer is the dynamic color onSecondaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSecondaryContainer
	OnSecondaryContainer *DynamicColor
	// Tertiary is the dynamic color tertiary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.tertiary
	Tertiary *DynamicColor
	// OnTertiary is the dynamic color onTertiary of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onTertiary
	OnTertiary *DynamicColor
	// TertiaryContainer is the dynamic color tertiaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.tertiaryContainer
	TertiaryContainer *DynamicColor
	// OnTertiaryContainer is the dynamic color onTertiaryContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onTertiaryContainer
	OnTertiaryContainer *DynamicColor
	// Error is the dynamic color error of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.error
	Error *DynamicColor
	// OnError is the dynamic color onError of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onError
	OnError *DynamicColor
	// ErrorContainer is the dynamic color errorContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.errorContainer
	ErrorContainer *DynamicColor
	// OnErrorContainer is the dynamic color onErrorContainer of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onErrorContainer
	OnErrorContainer *DynamicColor
	// PrimaryFixed is the dynamic color primaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.primaryFixed
	PrimaryFixed *DynamicColor
	// PrimaryFixedDim is the dynamic color primaryFixedDim of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.primaryFixedDim
	PrimaryFixedDim *DynamicColor
	// OnPrimaryFixed is the dynamic color onPrimaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onPrimaryFixed
	OnPrimaryFixed *DynamicColor
	// OnPrimaryFixedVariant is the dynamic color onPrimaryFixedVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onPrimaryFixedVariant
	OnPrimaryFixedVariant *DynamicColor
	// SecondaryFixed is the dynamic color secondaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.secondaryFixed
	SecondaryFixed *DynamicColor
	// SecondaryFixedDim is the dynamic color secondaryFixedDim of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.secondaryFixedDim
	SecondaryFixedDim *DynamicColor
	// OnSecondaryFixed is the dynamic color onSecondaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSecondaryFixed
	OnSecondaryFixed *DynamicColor
	// OnSecondaryFixedVariant is the dynamic color onSecondaryFixedVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onSecondaryFixedVariant
	OnSecondaryFixedVariant *DynamicColor
	// TertiaryFixed is the dynamic color tertiaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.tertiaryFixed
	TertiaryFixed *DynamicColor
	// TertiaryFixedDim is the dynamic color tertiaryFixedDim of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.tertiaryFixedDim
	TertiaryFixedDim *DynamicColor
	// OnTertiaryFixed is the dynamic color onTertiaryFixed of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onTertiaryFixed
	OnTertiaryFixed *DynamicColor
	// OnTertiaryFixedVariant is the dynamic color onTertiaryFixedVariant of Material 3.
	//
	// Flutter: material-color-utilities dynamiccolor/material_dynamic_colors.dart MaterialDynamicColors.onTertiaryFixedVariant
	OnTertiaryFixedVariant *DynamicColor
)

func primaryPalette(s *DynamicScheme) *TonalPalette        { return s.PrimaryPalette }
func secondaryPalette(s *DynamicScheme) *TonalPalette      { return s.SecondaryPalette }
func tertiaryPalette(s *DynamicScheme) *TonalPalette       { return s.TertiaryPalette }
func neutralPalette(s *DynamicScheme) *TonalPalette        { return s.NeutralPalette }
func neutralVariantPalette(s *DynamicScheme) *TonalPalette { return s.NeutralVariantPalette }
func errorPalette(s *DynamicScheme) *TonalPalette          { return s.ErrorPalette }

// darkLight is a tone that is one value in dark schemes and another in light ones.
func darkLight(dark, light float64) func(*DynamicScheme) float64 {
	return func(s *DynamicScheme) float64 {
		if s.IsDark {
			return dark
		}
		return light
	}
}

// monochromeOr is a tone that is the given one, or the monochrome one in a
// monochrome scheme, each for dark and light.
func monochromeOr(monoDark, monoLight, dark, light float64) func(*DynamicScheme) float64 {
	return func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return darkLight(monoDark, monoLight)(s)
		}
		return darkLight(dark, light)(s)
	}
}

// constant is a tone that does not depend on the scheme.
func constant(tone float64) func(*DynamicScheme) float64 {
	return func(*DynamicScheme) float64 { return tone }
}

// monoOrConstant is a constant tone, or another one in a monochrome scheme.
func monoOrConstant(mono, other float64) func(*DynamicScheme) float64 {
	return func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return mono
		}
		return other
	}
}

// curveTone is a tone that follows a contrast curve in each of dark and light.
func curveTone(dark, light *ContrastCurve) func(*DynamicScheme) float64 {
	return func(s *DynamicScheme) float64 {
		if s.IsDark {
			return dark.Get(s.ContrastLevel)
		}
		return light.Get(s.ContrastLevel)
	}
}

func dynamic(name string, palette func(*DynamicScheme) *TonalPalette, tone func(*DynamicScheme) float64, o DynamicColorOpts) *DynamicColor {
	o.Name = name
	return NewDynamicColorFromPalette(palette, tone, o)
}

func color(c **DynamicColor) func(*DynamicScheme) *DynamicColor {
	return func(*DynamicScheme) *DynamicColor { return *c }
}

func pair(a, b **DynamicColor, delta float64, polarity TonePolarity, stayTogether bool) func(*DynamicScheme) *ToneDeltaPair {
	return func(*DynamicScheme) *ToneDeltaPair { return NewToneDeltaPair(*a, *b, delta, polarity, stayTogether) }
}

func init() {
	// The key colors.
	PrimaryPaletteKeyColor = dynamic("primary_palette_key_color", primaryPalette,
		func(s *DynamicScheme) float64 { return s.PrimaryPalette.KeyColor().Tone() }, DynamicColorOpts{})
	SecondaryPaletteKeyColor = dynamic("secondary_palette_key_color", secondaryPalette,
		func(s *DynamicScheme) float64 { return s.SecondaryPalette.KeyColor().Tone() }, DynamicColorOpts{})
	TertiaryPaletteKeyColor = dynamic("tertiary_palette_key_color", tertiaryPalette,
		func(s *DynamicScheme) float64 { return s.TertiaryPalette.KeyColor().Tone() }, DynamicColorOpts{})
	NeutralPaletteKeyColor = dynamic("neutral_palette_key_color", neutralPalette,
		func(s *DynamicScheme) float64 { return s.NeutralPalette.KeyColor().Tone() }, DynamicColorOpts{})
	NeutralVariantPaletteKeyColor = dynamic("neutral_variant_palette_key_color", neutralVariantPalette,
		func(s *DynamicScheme) float64 { return s.NeutralVariantPalette.KeyColor().Tone() }, DynamicColorOpts{})

	// The surfaces.
	Background = dynamic("background", neutralPalette, darkLight(6, 98), DynamicColorOpts{IsBackground: true})
	OnBackground = dynamic("on_background", neutralPalette, darkLight(90, 10), DynamicColorOpts{
		Background: color(&Background), ContrastCurve: NewContrastCurve(3, 3, 4.5, 7),
	})
	Surface = dynamic("surface", neutralPalette, darkLight(6, 98), DynamicColorOpts{IsBackground: true})
	SurfaceDim = dynamic("surface_dim", neutralPalette, func(s *DynamicScheme) float64 {
		if s.IsDark {
			return 6
		}
		return NewContrastCurve(87, 87, 80, 75).Get(s.ContrastLevel)
	}, DynamicColorOpts{IsBackground: true})
	SurfaceBright = dynamic("surface_bright", neutralPalette, func(s *DynamicScheme) float64 {
		if s.IsDark {
			return NewContrastCurve(24, 24, 29, 34).Get(s.ContrastLevel)
		}
		return 98
	}, DynamicColorOpts{IsBackground: true})
	SurfaceContainerLowest = dynamic("surface_container_lowest", neutralPalette, func(s *DynamicScheme) float64 {
		if s.IsDark {
			return NewContrastCurve(4, 4, 2, 0).Get(s.ContrastLevel)
		}
		return 100
	}, DynamicColorOpts{IsBackground: true})
	SurfaceContainerLow = dynamic("surface_container_low", neutralPalette,
		curveTone(NewContrastCurve(10, 10, 11, 12), NewContrastCurve(96, 96, 96, 95)), DynamicColorOpts{IsBackground: true})
	SurfaceContainer = dynamic("surface_container", neutralPalette,
		curveTone(NewContrastCurve(12, 12, 16, 20), NewContrastCurve(94, 94, 92, 90)), DynamicColorOpts{IsBackground: true})
	SurfaceContainerHigh = dynamic("surface_container_high", neutralPalette,
		curveTone(NewContrastCurve(17, 17, 21, 25), NewContrastCurve(92, 92, 88, 85)), DynamicColorOpts{IsBackground: true})
	SurfaceContainerHighest = dynamic("surface_container_highest", neutralPalette,
		curveTone(NewContrastCurve(22, 22, 26, 30), NewContrastCurve(90, 90, 84, 80)), DynamicColorOpts{IsBackground: true})
	OnSurface = dynamic("on_surface", neutralPalette, darkLight(90, 10), DynamicColorOpts{
		Background: HighestSurface, ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	SurfaceVariant = dynamic("surface_variant", neutralVariantPalette, darkLight(30, 90), DynamicColorOpts{IsBackground: true})
	OnSurfaceVariant = dynamic("on_surface_variant", neutralVariantPalette, darkLight(80, 30), DynamicColorOpts{
		Background: HighestSurface, ContrastCurve: NewContrastCurve(3, 4.5, 7, 11),
	})
	InverseSurface = dynamic("inverse_surface", neutralPalette, darkLight(90, 20), DynamicColorOpts{})
	InverseOnSurface = dynamic("inverse_on_surface", neutralPalette, darkLight(20, 95), DynamicColorOpts{
		Background: color(&InverseSurface), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	Outline = dynamic("outline", neutralVariantPalette, darkLight(60, 50), DynamicColorOpts{
		Background: HighestSurface, ContrastCurve: NewContrastCurve(1.5, 3, 4.5, 7),
	})
	OutlineVariant = dynamic("outline_variant", neutralVariantPalette, darkLight(30, 80), DynamicColorOpts{
		Background: HighestSurface, ContrastCurve: NewContrastCurve(1, 1, 3, 4.5),
	})
	Shadow = dynamic("shadow", neutralPalette, constant(0), DynamicColorOpts{})
	Scrim = dynamic("scrim", neutralPalette, constant(0), DynamicColorOpts{})
	SurfaceTint = dynamic("surface_tint", primaryPalette, darkLight(80, 40), DynamicColorOpts{IsBackground: true})

	// The primary colors.
	Primary = dynamic("primary", primaryPalette, monochromeOr(100, 0, 80, 40), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(3, 4.5, 7, 7),
		ToneDeltaPair: pair(&PrimaryContainer, &Primary, 10, TonePolarityNearer, false),
	})
	OnPrimary = dynamic("on_primary", primaryPalette, monochromeOr(10, 90, 20, 100), DynamicColorOpts{
		Background: color(&Primary), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	PrimaryContainer = dynamic("primary_container", primaryPalette, func(s *DynamicScheme) float64 {
		if isFidelity(s) {
			return s.SourceColorHct.Tone()
		}
		return monochromeOr(85, 25, 30, 90)(s)
	}, DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(1, 1, 3, 4.5),
		ToneDeltaPair: pair(&PrimaryContainer, &Primary, 10, TonePolarityNearer, false),
	})
	OnPrimaryContainer = dynamic("on_primary_container", primaryPalette, func(s *DynamicScheme) float64 {
		if isFidelity(s) {
			return ForegroundTone(PrimaryContainer.Tone(s), 4.5)
		}
		return monochromeOr(0, 100, 90, 30)(s)
	}, DynamicColorOpts{Background: color(&PrimaryContainer), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11)})
	InversePrimary = dynamic("inverse_primary", primaryPalette, darkLight(40, 80), DynamicColorOpts{
		Background: color(&InverseSurface), ContrastCurve: NewContrastCurve(3, 4.5, 7, 7),
	})

	// The secondary colors.
	Secondary = dynamic("secondary", secondaryPalette, darkLight(80, 40), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(3, 4.5, 7, 7),
		ToneDeltaPair: pair(&SecondaryContainer, &Secondary, 10, TonePolarityNearer, false),
	})
	OnSecondary = dynamic("on_secondary", secondaryPalette, monochromeOr(10, 100, 20, 100), DynamicColorOpts{
		Background: color(&Secondary), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	SecondaryContainer = dynamic("secondary_container", secondaryPalette, func(s *DynamicScheme) float64 {
		initialTone := darkLight(30, 90)(s)
		if isMonochrome(s) {
			return darkLight(30, 85)(s)
		}
		if !isFidelity(s) {
			return initialTone
		}
		return findDesiredChromaByTone(s.SecondaryPalette.Hue(), s.SecondaryPalette.Chroma(), initialTone, !s.IsDark)
	}, DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(1, 1, 3, 4.5),
		ToneDeltaPair: pair(&SecondaryContainer, &Secondary, 10, TonePolarityNearer, false),
	})
	OnSecondaryContainer = dynamic("on_secondary_container", secondaryPalette, func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return darkLight(90, 10)(s)
		}
		if !isFidelity(s) {
			return darkLight(90, 30)(s)
		}
		return ForegroundTone(SecondaryContainer.Tone(s), 4.5)
	}, DynamicColorOpts{Background: color(&SecondaryContainer), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11)})

	// The tertiary colors.
	Tertiary = dynamic("tertiary", tertiaryPalette, monochromeOr(90, 25, 80, 40), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(3, 4.5, 7, 7),
		ToneDeltaPair: pair(&TertiaryContainer, &Tertiary, 10, TonePolarityNearer, false),
	})
	OnTertiary = dynamic("on_tertiary", tertiaryPalette, monochromeOr(10, 90, 20, 100), DynamicColorOpts{
		Background: color(&Tertiary), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	TertiaryContainer = dynamic("tertiary_container", tertiaryPalette, func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return darkLight(60, 49)(s)
		}
		if !isFidelity(s) {
			return darkLight(30, 90)(s)
		}
		proposed := s.TertiaryPalette.GetHct(s.SourceColorHct.Tone())
		return FixIfDisliked(proposed).Tone()
	}, DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(1, 1, 3, 4.5),
		ToneDeltaPair: pair(&TertiaryContainer, &Tertiary, 10, TonePolarityNearer, false),
	})
	OnTertiaryContainer = dynamic("on_tertiary_container", tertiaryPalette, func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return darkLight(0, 100)(s)
		}
		if !isFidelity(s) {
			return darkLight(90, 30)(s)
		}
		return ForegroundTone(TertiaryContainer.Tone(s), 4.5)
	}, DynamicColorOpts{Background: color(&TertiaryContainer), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11)})

	// The error colors.
	Error = dynamic("error", errorPalette, darkLight(80, 40), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(3, 4.5, 7, 7),
		ToneDeltaPair: pair(&ErrorContainer, &Error, 10, TonePolarityNearer, false),
	})
	OnError = dynamic("on_error", errorPalette, darkLight(20, 100), DynamicColorOpts{
		Background: color(&Error), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	ErrorContainer = dynamic("error_container", errorPalette, darkLight(30, 90), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: NewContrastCurve(1, 1, 3, 4.5),
		ToneDeltaPair: pair(&ErrorContainer, &Error, 10, TonePolarityNearer, false),
	})
	OnErrorContainer = dynamic("on_error_container", errorPalette, func(s *DynamicScheme) float64 {
		if isMonochrome(s) {
			return darkLight(90, 10)(s)
		}
		return darkLight(90, 30)(s)
	}, DynamicColorOpts{Background: color(&ErrorContainer), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11)})

	// The fixed colors.
	fixedCurve := NewContrastCurve(1, 1, 3, 4.5)
	PrimaryFixed = dynamic("primary_fixed", primaryPalette, monoOrConstant(40, 90), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&PrimaryFixed, &PrimaryFixedDim, 10, TonePolarityLighter, true),
	})
	PrimaryFixedDim = dynamic("primary_fixed_dim", primaryPalette, monoOrConstant(30, 80), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&PrimaryFixed, &PrimaryFixedDim, 10, TonePolarityLighter, true),
	})
	OnPrimaryFixed = dynamic("on_primary_fixed", primaryPalette, monoOrConstant(100, 10), DynamicColorOpts{
		Background: color(&PrimaryFixedDim), SecondBackground: color(&PrimaryFixed), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	OnPrimaryFixedVariant = dynamic("on_primary_fixed_variant", primaryPalette, monoOrConstant(90, 30), DynamicColorOpts{
		Background: color(&PrimaryFixedDim), SecondBackground: color(&PrimaryFixed), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11),
	})
	SecondaryFixed = dynamic("secondary_fixed", secondaryPalette, monoOrConstant(80, 90), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&SecondaryFixed, &SecondaryFixedDim, 10, TonePolarityLighter, true),
	})
	SecondaryFixedDim = dynamic("secondary_fixed_dim", secondaryPalette, monoOrConstant(70, 80), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&SecondaryFixed, &SecondaryFixedDim, 10, TonePolarityLighter, true),
	})
	OnSecondaryFixed = dynamic("on_secondary_fixed", secondaryPalette, constant(10), DynamicColorOpts{
		Background: color(&SecondaryFixedDim), SecondBackground: color(&SecondaryFixed), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	OnSecondaryFixedVariant = dynamic("on_secondary_fixed_variant", secondaryPalette, monoOrConstant(25, 30), DynamicColorOpts{
		Background: color(&SecondaryFixedDim), SecondBackground: color(&SecondaryFixed), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11),
	})
	TertiaryFixed = dynamic("tertiary_fixed", tertiaryPalette, monoOrConstant(40, 90), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&TertiaryFixed, &TertiaryFixedDim, 10, TonePolarityLighter, true),
	})
	TertiaryFixedDim = dynamic("tertiary_fixed_dim", tertiaryPalette, monoOrConstant(30, 80), DynamicColorOpts{
		IsBackground: true, Background: HighestSurface, ContrastCurve: fixedCurve,
		ToneDeltaPair: pair(&TertiaryFixed, &TertiaryFixedDim, 10, TonePolarityLighter, true),
	})
	OnTertiaryFixed = dynamic("on_tertiary_fixed", tertiaryPalette, monoOrConstant(100, 10), DynamicColorOpts{
		Background: color(&TertiaryFixedDim), SecondBackground: color(&TertiaryFixed), ContrastCurve: NewContrastCurve(4.5, 7, 11, 21),
	})
	OnTertiaryFixedVariant = dynamic("on_tertiary_fixed_variant", tertiaryPalette, monoOrConstant(90, 30), DynamicColorOpts{
		Background: color(&TertiaryFixedDim), SecondBackground: color(&TertiaryFixed), ContrastCurve: NewContrastCurve(3, 4.5, 7, 11),
	})
}

// findDesiredChromaByTone finds the tone, starting at the given one and moving
// down (or up) by one, that has the desired chroma, or the one closest to it.
func findDesiredChromaByTone(hue, chroma, tone float64, byDecreasingTone bool) float64 {
	answer := tone
	closestToChroma := HctFrom(hue, chroma, tone)
	if closestToChroma.Chroma() < chroma {
		chromaPeak := closestToChroma.Chroma()
		for closestToChroma.Chroma() < chroma {
			if byDecreasingTone {
				answer--
			} else {
				answer++
			}
			potential := HctFrom(hue, chroma, answer)
			if chromaPeak > potential.Chroma() {
				break
			}
			if math.Abs(potential.Chroma()-chroma) < 0.4 {
				break
			}
			potentialDelta := math.Abs(potential.Chroma() - chroma)
			currentDelta := math.Abs(closestToChroma.Chroma() - chroma)
			if potentialDelta < currentDelta {
				closestToChroma = potential
			}
			chromaPeak = math.Max(chromaPeak, potential.Chroma())
		}
	}
	return answer
}
