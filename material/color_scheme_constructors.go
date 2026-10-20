package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/internal/mcu"
	"github.com/rm4n0s/glutter/ui"
)

// ColorSchemeVariantOpts are the optional parameters of the constructors that have
// defaults for the required colors (NewColorSchemeLight, NewColorSchemeDark,
// NewColorSchemeHighContrastLight and NewColorSchemeHighContrastDark), and of
// ColorSchemeFromSeed, where a color that is set replaces the one that the
// dynamic scheme makes.
//
// Flutter: material/color_scheme.dart ColorScheme.light
type ColorSchemeVariantOpts struct {
	ColorSchemeOpts
	// Brightness is the brightness; the default is the one of the constructor.
	Brightness ui.Brightness
	// Primary is the color primary; the default is the one of the constructor.
	Primary foundation.Opt[ui.Color]
	// OnPrimary is the color onPrimary; the default is the one of the constructor.
	OnPrimary foundation.Opt[ui.Color]
	// Secondary is the color secondary; the default is the one of the constructor.
	Secondary foundation.Opt[ui.Color]
	// OnSecondary is the color onSecondary; the default is the one of the constructor.
	OnSecondary foundation.Opt[ui.Color]
	// Error is the color error; the default is the one of the constructor.
	Error foundation.Opt[ui.Color]
	// OnError is the color onError; the default is the one of the constructor.
	OnError foundation.Opt[ui.Color]
	// Surface is the color surface; the default is the one of the constructor.
	Surface foundation.Opt[ui.Color]
	// OnSurface is the color onSurface; the default is the one of the constructor.
	OnSurface foundation.Opt[ui.Color]
}

// NewColorSchemeLight creates a scheme of the light Material 2 baseline colors (a purple primary and a teal secondary), the colors that Flutter's ColorScheme.light has.
//
// Flutter: material/color_scheme.dart ColorScheme.light
func NewColorSchemeLight(opts ...ColorSchemeVariantOpts) *ColorScheme {
	o := firstOpts(opts)
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessLight
	}
	return NewColorScheme(o.Brightness,
		o.Primary.OrElse(ui.NewColor(0xff6200ee)),
		o.OnPrimary.OrElse(ui.NewColor(0xffffffff)),
		o.Secondary.OrElse(ui.NewColor(0xff03dac6)),
		o.OnSecondary.OrElse(ui.NewColor(0xff000000)),
		o.Error.OrElse(ui.NewColor(0xffb00020)),
		o.OnError.OrElse(ui.NewColor(0xffffffff)),
		o.Surface.OrElse(ui.NewColor(0xffffffff)),
		o.OnSurface.OrElse(ui.NewColor(0xff000000)),
		o.ColorSchemeOpts)
}

// NewColorSchemeDark creates a scheme of the dark Material 2 baseline colors.
//
// Flutter: material/color_scheme.dart ColorScheme.dark
func NewColorSchemeDark(opts ...ColorSchemeVariantOpts) *ColorScheme {
	o := firstOpts(opts)
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessDark
	}
	return NewColorScheme(o.Brightness,
		o.Primary.OrElse(ui.NewColor(0xffbb86fc)),
		o.OnPrimary.OrElse(ui.NewColor(0xff000000)),
		o.Secondary.OrElse(ui.NewColor(0xff03dac6)),
		o.OnSecondary.OrElse(ui.NewColor(0xff000000)),
		o.Error.OrElse(ui.NewColor(0xffcf6679)),
		o.OnError.OrElse(ui.NewColor(0xff000000)),
		o.Surface.OrElse(ui.NewColor(0xff121212)),
		o.OnSurface.OrElse(ui.NewColor(0xffffffff)),
		o.ColorSchemeOpts)
}

// NewColorSchemeHighContrastLight creates a scheme of light colors with a high contrast.
//
// Flutter: material/color_scheme.dart ColorScheme.highContrastLight
func NewColorSchemeHighContrastLight(opts ...ColorSchemeVariantOpts) *ColorScheme {
	o := firstOpts(opts)
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessLight
	}
	return NewColorScheme(o.Brightness,
		o.Primary.OrElse(ui.NewColor(0xff0000ba)),
		o.OnPrimary.OrElse(ui.NewColor(0xffffffff)),
		o.Secondary.OrElse(ui.NewColor(0xff66fff9)),
		o.OnSecondary.OrElse(ui.NewColor(0xff000000)),
		o.Error.OrElse(ui.NewColor(0xff790000)),
		o.OnError.OrElse(ui.NewColor(0xffffffff)),
		o.Surface.OrElse(ui.NewColor(0xffffffff)),
		o.OnSurface.OrElse(ui.NewColor(0xff000000)),
		o.ColorSchemeOpts)
}

// NewColorSchemeHighContrastDark creates a scheme of dark colors with a high contrast.
//
// Flutter: material/color_scheme.dart ColorScheme.highContrastDark
func NewColorSchemeHighContrastDark(opts ...ColorSchemeVariantOpts) *ColorScheme {
	o := firstOpts(opts)
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessDark
	}
	return NewColorScheme(o.Brightness,
		o.Primary.OrElse(ui.NewColor(0xffefb7ff)),
		o.OnPrimary.OrElse(ui.NewColor(0xff000000)),
		o.Secondary.OrElse(ui.NewColor(0xff66fff9)),
		o.OnSecondary.OrElse(ui.NewColor(0xff000000)),
		o.Error.OrElse(ui.NewColor(0xff9b374d)),
		o.OnError.OrElse(ui.NewColor(0xff000000)),
		o.Surface.OrElse(ui.NewColor(0xff121212)),
		o.OnSurface.OrElse(ui.NewColor(0xffffffff)),
		o.ColorSchemeOpts)
}

// DynamicSchemeVariant is the algorithm that makes a color scheme from a seed
// color: the variants of the Material 3 dynamic color system.
//
// Flutter: material/color_scheme.dart DynamicSchemeVariant
type DynamicSchemeVariant int

// The variants, in the order of Flutter's enum. DynamicSchemeVariantUnspecified is
// the zero value and means tonal spot.
const (
	// DynamicSchemeVariantUnspecified is the zero value: tonal spot.
	//
	// Flutter: no counterpart
	DynamicSchemeVariantUnspecified DynamicSchemeVariant = iota
	// DynamicSchemeVariantTonalSpot is the default Material You scheme of 2021:
	// pastel tokens and low chroma palettes.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.tonalSpot
	DynamicSchemeVariantTonalSpot
	// DynamicSchemeVariantFidelity keeps the colors of the seed color: the primary
	// container is the seed, and the tertiary color its complement.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.fidelity
	DynamicSchemeVariantFidelity
	// DynamicSchemeVariantMonochrome makes every color a shade of gray.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.monochrome
	DynamicSchemeVariantMonochrome
	// DynamicSchemeVariantNeutral is close to gray, with a hint of chroma.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.neutral
	DynamicSchemeVariantNeutral
	// DynamicSchemeVariantVibrant has high chroma palettes.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.vibrant
	DynamicSchemeVariantVibrant
	// DynamicSchemeVariantExpressive has medium chroma palettes and a primary hue
	// that differs from the seed's.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.expressive
	DynamicSchemeVariantExpressive
	// DynamicSchemeVariantContent is almost the same as fidelity, with an analogous
	// tertiary color.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.content
	DynamicSchemeVariantContent
	// DynamicSchemeVariantRainbow is a playful scheme in which the hue of the seed
	// does not appear.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.rainbow
	DynamicSchemeVariantRainbow
	// DynamicSchemeVariantFruitSalad is a playful scheme in which the hue of the
	// seed does not appear.
	//
	// Flutter: material/color_scheme.dart DynamicSchemeVariant.fruitSalad
	DynamicSchemeVariantFruitSalad
)

var dynamicSchemeVariantNames = [...]string{
	"DynamicSchemeVariantUnspecified", "DynamicSchemeVariantTonalSpot", "DynamicSchemeVariantFidelity",
	"DynamicSchemeVariantMonochrome", "DynamicSchemeVariantNeutral", "DynamicSchemeVariantVibrant",
	"DynamicSchemeVariantExpressive", "DynamicSchemeVariantContent", "DynamicSchemeVariantRainbow", "DynamicSchemeVariantFruitSalad",
}

// String is the name of the variant.
//
// Flutter: material/color_scheme.dart DynamicSchemeVariant
func (v DynamicSchemeVariant) String() string {
	if v < 0 || int(v) >= len(dynamicSchemeVariantNames) {
		return "DynamicSchemeVariant(?)"
	}
	return dynamicSchemeVariantNames[v]
}

// ColorSchemeFromSeedOpts are the optional parameters of ColorSchemeFromSeed.
//
// Flutter: material/color_scheme.dart ColorScheme.fromSeed
type ColorSchemeFromSeedOpts struct {
	// Brightness is the brightness of the scheme; the default is light.
	Brightness ui.Brightness
	// DynamicSchemeVariant is the algorithm; the default is tonal spot.
	DynamicSchemeVariant DynamicSchemeVariant
	// ContrastLevel is from -1 (the lowest contrast) to 1 (the highest); the
	// default is 0.
	ContrastLevel float64
	// Overrides are the colors that replace the ones the dynamic scheme makes.
	// Their Brightness is ignored.
	Overrides ColorSchemeVariantOpts
}

// ColorSchemeFromSeed makes a Material 3 color scheme from a seed color with the
// dynamic color algorithm of the variant: the palettes are made from the seed, and
// the colors are the ones of the dynamic scheme (MaterialDynamicColors) for the
// brightness and the contrast level. In debug builds it panics with the tag
// ColorSchemeContrastLevel when the contrast level is not between -1 and 1.
//
// Flutter: material/color_scheme.dart ColorScheme.fromSeed
func ColorSchemeFromSeed(seedColor ui.Color, opts ...ColorSchemeFromSeedOpts) *ColorScheme {
	o := firstOpts(opts)
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessLight
	}
	scheme := buildDynamicScheme(o.Brightness, seedColor, o.DynamicSchemeVariant, o.ContrastLevel)
	return colorSchemeFromDynamic(scheme, o.Brightness, o.Overrides)
}

// colorSchemeFromDynamic reads the colors of the dynamic scheme, with the
// overrides in place of the ones they set.
func colorSchemeFromDynamic(scheme *mcu.DynamicScheme, brightness ui.Brightness, ov ColorSchemeVariantOpts) *ColorScheme {
	pick := func(set foundation.Opt[ui.Color], c *mcu.DynamicColor) ui.Color {
		return set.OrElse(ui.NewColor(c.GetArgb(scheme)))
	}
	pickOpt := func(set foundation.Opt[ui.Color], c *mcu.DynamicColor) foundation.Opt[ui.Color] {
		return foundation.Some(pick(set, c))
	}
	return NewColorScheme(brightness,
		pick(ov.Primary, mcu.Primary),
		pick(ov.OnPrimary, mcu.OnPrimary),
		pick(ov.Secondary, mcu.Secondary),
		pick(ov.OnSecondary, mcu.OnSecondary),
		pick(ov.Error, mcu.Error),
		pick(ov.OnError, mcu.OnError),
		pick(ov.Surface, mcu.Surface),
		pick(ov.OnSurface, mcu.OnSurface),
		ColorSchemeOpts{
			PrimaryContainer:        pickOpt(ov.PrimaryContainer, mcu.PrimaryContainer),
			OnPrimaryContainer:      pickOpt(ov.OnPrimaryContainer, mcu.OnPrimaryContainer),
			PrimaryFixed:            pickOpt(ov.PrimaryFixed, mcu.PrimaryFixed),
			PrimaryFixedDim:         pickOpt(ov.PrimaryFixedDim, mcu.PrimaryFixedDim),
			OnPrimaryFixed:          pickOpt(ov.OnPrimaryFixed, mcu.OnPrimaryFixed),
			OnPrimaryFixedVariant:   pickOpt(ov.OnPrimaryFixedVariant, mcu.OnPrimaryFixedVariant),
			SecondaryContainer:      pickOpt(ov.SecondaryContainer, mcu.SecondaryContainer),
			OnSecondaryContainer:    pickOpt(ov.OnSecondaryContainer, mcu.OnSecondaryContainer),
			SecondaryFixed:          pickOpt(ov.SecondaryFixed, mcu.SecondaryFixed),
			SecondaryFixedDim:       pickOpt(ov.SecondaryFixedDim, mcu.SecondaryFixedDim),
			OnSecondaryFixed:        pickOpt(ov.OnSecondaryFixed, mcu.OnSecondaryFixed),
			OnSecondaryFixedVariant: pickOpt(ov.OnSecondaryFixedVariant, mcu.OnSecondaryFixedVariant),
			Tertiary:                pickOpt(ov.Tertiary, mcu.Tertiary),
			OnTertiary:              pickOpt(ov.OnTertiary, mcu.OnTertiary),
			TertiaryContainer:       pickOpt(ov.TertiaryContainer, mcu.TertiaryContainer),
			OnTertiaryContainer:     pickOpt(ov.OnTertiaryContainer, mcu.OnTertiaryContainer),
			TertiaryFixed:           pickOpt(ov.TertiaryFixed, mcu.TertiaryFixed),
			TertiaryFixedDim:        pickOpt(ov.TertiaryFixedDim, mcu.TertiaryFixedDim),
			OnTertiaryFixed:         pickOpt(ov.OnTertiaryFixed, mcu.OnTertiaryFixed),
			OnTertiaryFixedVariant:  pickOpt(ov.OnTertiaryFixedVariant, mcu.OnTertiaryFixedVariant),
			ErrorContainer:          pickOpt(ov.ErrorContainer, mcu.ErrorContainer),
			OnErrorContainer:        pickOpt(ov.OnErrorContainer, mcu.OnErrorContainer),
			SurfaceDim:              pickOpt(ov.SurfaceDim, mcu.SurfaceDim),
			SurfaceBright:           pickOpt(ov.SurfaceBright, mcu.SurfaceBright),
			SurfaceContainerLowest:  pickOpt(ov.SurfaceContainerLowest, mcu.SurfaceContainerLowest),
			SurfaceContainerLow:     pickOpt(ov.SurfaceContainerLow, mcu.SurfaceContainerLow),
			SurfaceContainer:        pickOpt(ov.SurfaceContainer, mcu.SurfaceContainer),
			SurfaceContainerHigh:    pickOpt(ov.SurfaceContainerHigh, mcu.SurfaceContainerHigh),
			SurfaceContainerHighest: pickOpt(ov.SurfaceContainerHighest, mcu.SurfaceContainerHighest),
			OnSurfaceVariant:        pickOpt(ov.OnSurfaceVariant, mcu.OnSurfaceVariant),
			Outline:                 pickOpt(ov.Outline, mcu.Outline),
			OutlineVariant:          pickOpt(ov.OutlineVariant, mcu.OutlineVariant),
			Shadow:                  pickOpt(ov.Shadow, mcu.Shadow),
			Scrim:                   pickOpt(ov.Scrim, mcu.Scrim),
			InverseSurface:          pickOpt(ov.InverseSurface, mcu.InverseSurface),
			OnInverseSurface:        pickOpt(ov.OnInverseSurface, mcu.InverseOnSurface),
			InversePrimary:          pickOpt(ov.InversePrimary, mcu.InversePrimary),
			SurfaceTint:             pickOpt(ov.SurfaceTint, mcu.Primary),
			Background:              pickOpt(ov.Background, mcu.Background),
			OnBackground:            pickOpt(ov.OnBackground, mcu.OnBackground),
			SurfaceVariant:          pickOpt(ov.SurfaceVariant, mcu.SurfaceVariant),
		})
}

func buildDynamicScheme(brightness ui.Brightness, seedColor ui.Color, variant DynamicSchemeVariant, contrastLevel float64) *mcu.DynamicScheme {
	if foundation.DebugMode && (contrastLevel < -1.0 || contrastLevel > 1.0) {
		panic(errors.New("ColorSchemeContrastLevel", "contrastLevel must be between -1.0 and 1.0 inclusive.", "contrastLevel", contrastLevel))
	}
	isDark := brightness == ui.BrightnessDark
	source := mcu.HctFromInt(seedColor.ToARGB32())
	switch variant {
	case DynamicSchemeVariantFidelity:
		return mcu.NewSchemeFidelity(source, isDark, contrastLevel)
	case DynamicSchemeVariantContent:
		return mcu.NewSchemeContent(source, isDark, contrastLevel)
	case DynamicSchemeVariantMonochrome:
		return mcu.NewSchemeMonochrome(source, isDark, contrastLevel)
	case DynamicSchemeVariantNeutral:
		return mcu.NewSchemeNeutral(source, isDark, contrastLevel)
	case DynamicSchemeVariantVibrant:
		return mcu.NewSchemeVibrant(source, isDark, contrastLevel)
	case DynamicSchemeVariantExpressive:
		return mcu.NewSchemeExpressive(source, isDark, contrastLevel)
	case DynamicSchemeVariantRainbow:
		return mcu.NewSchemeRainbow(source, isDark, contrastLevel)
	case DynamicSchemeVariantFruitSalad:
		return mcu.NewSchemeFruitSalad(source, isDark, contrastLevel)
	}
	return mcu.NewSchemeTonalSpot(source, isDark, contrastLevel)
}

// ColorSchemeFromSwatchOpts are the optional parameters of ColorSchemeFromSwatch.
//
// Flutter: material/color_scheme.dart ColorScheme.fromSwatch
type ColorSchemeFromSwatchOpts struct {
	// PrimarySwatch is the primary swatch; the default is Colors.Blue.
	PrimarySwatch *MaterialColor
	// AccentColor is the secondary color; the default is the primary swatch, or
	// teal accent 200 in a dark scheme.
	AccentColor foundation.Opt[ui.Color]
	// CardColor is the surface color; the default is white, or grey 800 in a dark
	// scheme.
	CardColor foundation.Opt[ui.Color]
	// BackgroundColor is the background color; the default is the primary swatch's
	// shade 200, or grey 700 in a dark scheme.
	BackgroundColor foundation.Opt[ui.Color]
	// ErrorColor is the error color; the default is red 700.
	ErrorColor foundation.Opt[ui.Color]
	// Brightness is the brightness; the default is light.
	Brightness ui.Brightness
}

// ColorSchemeFromSwatch is a color scheme that matches the default ThemeData of
// Flutter before Material 3: it is made from a swatch, the on colors being black
// or white for their contrast.
//
// Flutter: material/color_scheme.dart ColorScheme.fromSwatch
func ColorSchemeFromSwatch(opts ...ColorSchemeFromSwatchOpts) *ColorScheme {
	o := firstOpts(opts)
	swatch := Colors.Blue
	if o.PrimarySwatch != nil {
		swatch = *o.PrimarySwatch
	}
	if o.Brightness == ui.BrightnessUnspecified {
		o.Brightness = ui.BrightnessLight
	}
	isDark := o.Brightness == ui.BrightnessDark
	primaryIsDark := EstimateBrightnessForColor(swatch.Primary) == ui.BrightnessDark
	secondary := o.AccentColor.OrElse(swatch.Primary)
	if isDark && o.AccentColor.IsNone() {
		secondary = Colors.TealAccent.Shade200()
	}
	secondaryIsDark := EstimateBrightnessForColor(secondary) == ui.BrightnessDark
	surface := o.CardColor.OrElse(Colors.White)
	if isDark && o.CardColor.IsNone() {
		surface = Colors.Grey.Shade800()
	}
	onColor := func(dark bool) ui.Color {
		if dark {
			return Colors.White
		}
		return Colors.Black
	}
	background := o.BackgroundColor.OrElse(swatch.Shade200())
	if isDark && o.BackgroundColor.IsNone() {
		background = Colors.Grey.Shade700()
	}
	onError := Colors.White
	if isDark {
		onError = Colors.Black
	}
	return NewColorScheme(o.Brightness, swatch.Primary, onColor(primaryIsDark), secondary, onColor(secondaryIsDark),
		o.ErrorColor.OrElse(Colors.Red.Shade700()), onError, surface, onColor(isDark),
		ColorSchemeOpts{
			// Deprecated in Flutter.
			Background:   foundation.Some(background),
			OnBackground: foundation.Some(onColor(primaryIsDark)),
		})
}
