package mcu

// Scheme is the first version of the Material 3 color scheme: the colors of the
// light or the dark scheme of a core palette. DynamicScheme replaces it.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme
type Scheme struct {
	Primary, OnPrimary, PrimaryContainer, OnPrimaryContainer         uint32
	Secondary, OnSecondary, SecondaryContainer, OnSecondaryContainer uint32
	Tertiary, OnTertiary, TertiaryContainer, OnTertiaryContainer     uint32
	Error, OnError, ErrorContainer, OnErrorContainer                 uint32
	Background, OnBackground, Surface, OnSurface                     uint32
	SurfaceVariant, OnSurfaceVariant, Outline, OutlineVariant        uint32
	Shadow, Scrim, InverseSurface, InverseOnSurface, InversePrimary  uint32
}

// SchemeLight is the light scheme of a color.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.light
func SchemeLight(color uint32) *Scheme { return SchemeLightFromCorePalette(CorePaletteOf(color)) }

// SchemeDark is the dark scheme of a color.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.dark
func SchemeDark(color uint32) *Scheme { return SchemeDarkFromCorePalette(CorePaletteOf(color)) }

// SchemeLightContent is the light scheme of a color that keeps its chroma.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.lightContent
func SchemeLightContent(color uint32) *Scheme {
	return SchemeLightFromCorePalette(CorePaletteContentOf(color))
}

// SchemeDarkContent is the dark scheme of a color that keeps its chroma.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.darkContent
func SchemeDarkContent(color uint32) *Scheme {
	return SchemeDarkFromCorePalette(CorePaletteContentOf(color))
}

// SchemeLightFromCorePalette is the light scheme of the palettes.
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.lightFromCorePalette
func SchemeLightFromCorePalette(p *CorePalette) *Scheme {
	return &Scheme{
		Primary: p.Primary.Get(40), OnPrimary: p.Primary.Get(100), PrimaryContainer: p.Primary.Get(90), OnPrimaryContainer: p.Primary.Get(10),
		Secondary: p.Secondary.Get(40), OnSecondary: p.Secondary.Get(100), SecondaryContainer: p.Secondary.Get(90), OnSecondaryContainer: p.Secondary.Get(10),
		Tertiary: p.Tertiary.Get(40), OnTertiary: p.Tertiary.Get(100), TertiaryContainer: p.Tertiary.Get(90), OnTertiaryContainer: p.Tertiary.Get(10),
		Error: p.Error.Get(40), OnError: p.Error.Get(100), ErrorContainer: p.Error.Get(90), OnErrorContainer: p.Error.Get(10),
		Background: p.Neutral.Get(99), OnBackground: p.Neutral.Get(10), Surface: p.Neutral.Get(99), OnSurface: p.Neutral.Get(10),
		SurfaceVariant: p.NeutralVariant.Get(90), OnSurfaceVariant: p.NeutralVariant.Get(30),
		Outline: p.NeutralVariant.Get(50), OutlineVariant: p.NeutralVariant.Get(80),
		Shadow: p.Neutral.Get(0), Scrim: p.Neutral.Get(0), InverseSurface: p.Neutral.Get(20), InverseOnSurface: p.Neutral.Get(95),
		InversePrimary: p.Primary.Get(80),
	}
}

// SchemeDarkFromCorePalette is the dark scheme of the palettes. (The color
// onErrorContainer is the tone 80 of the error palette, as in the Dart source,
// where the light scheme has 10 and the others 90.)
//
// Flutter: material-color-utilities scheme/scheme.dart Scheme.darkFromCorePalette
func SchemeDarkFromCorePalette(p *CorePalette) *Scheme {
	return &Scheme{
		Primary: p.Primary.Get(80), OnPrimary: p.Primary.Get(20), PrimaryContainer: p.Primary.Get(30), OnPrimaryContainer: p.Primary.Get(90),
		Secondary: p.Secondary.Get(80), OnSecondary: p.Secondary.Get(20), SecondaryContainer: p.Secondary.Get(30), OnSecondaryContainer: p.Secondary.Get(90),
		Tertiary: p.Tertiary.Get(80), OnTertiary: p.Tertiary.Get(20), TertiaryContainer: p.Tertiary.Get(30), OnTertiaryContainer: p.Tertiary.Get(90),
		Error: p.Error.Get(80), OnError: p.Error.Get(20), ErrorContainer: p.Error.Get(30), OnErrorContainer: p.Error.Get(80),
		Background: p.Neutral.Get(10), OnBackground: p.Neutral.Get(90), Surface: p.Neutral.Get(10), OnSurface: p.Neutral.Get(90),
		SurfaceVariant: p.NeutralVariant.Get(30), OnSurfaceVariant: p.NeutralVariant.Get(80),
		Outline: p.NeutralVariant.Get(60), OutlineVariant: p.NeutralVariant.Get(30),
		Shadow: p.Neutral.Get(0), Scrim: p.Neutral.Get(0), InverseSurface: p.Neutral.Get(90), InverseOnSurface: p.Neutral.Get(20),
		InversePrimary: p.Primary.Get(40),
	}
}
