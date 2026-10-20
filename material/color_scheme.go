package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ColorScheme is a set of 45 colors based on the Material spec that can be used
// to configure the color properties of most components. The colors that a scheme
// does not set take the value that Flutter gives them: the container colors
// follow the primary, secondary and tertiary colors, the surface colors follow
// the surface, and so on.
//
// A ColorScheme is immutable. Its properties are methods, as the getters of
// Flutter's class: Primary(), OnPrimary()...
//
// Flutter: material/color_scheme.dart ColorScheme
type ColorScheme struct {
	brightness ui.Brightness

	primary                 ui.Color
	onPrimary               ui.Color
	secondary               ui.Color
	onSecondary             ui.Color
	errorColor              ui.Color
	onError                 ui.Color
	surface                 ui.Color
	onSurface               ui.Color
	primaryContainer        foundation.Opt[ui.Color]
	onPrimaryContainer      foundation.Opt[ui.Color]
	primaryFixed            foundation.Opt[ui.Color]
	primaryFixedDim         foundation.Opt[ui.Color]
	onPrimaryFixed          foundation.Opt[ui.Color]
	onPrimaryFixedVariant   foundation.Opt[ui.Color]
	secondaryContainer      foundation.Opt[ui.Color]
	onSecondaryContainer    foundation.Opt[ui.Color]
	secondaryFixed          foundation.Opt[ui.Color]
	secondaryFixedDim       foundation.Opt[ui.Color]
	onSecondaryFixed        foundation.Opt[ui.Color]
	onSecondaryFixedVariant foundation.Opt[ui.Color]
	tertiary                foundation.Opt[ui.Color]
	onTertiary              foundation.Opt[ui.Color]
	tertiaryContainer       foundation.Opt[ui.Color]
	onTertiaryContainer     foundation.Opt[ui.Color]
	tertiaryFixed           foundation.Opt[ui.Color]
	tertiaryFixedDim        foundation.Opt[ui.Color]
	onTertiaryFixed         foundation.Opt[ui.Color]
	onTertiaryFixedVariant  foundation.Opt[ui.Color]
	errorContainer          foundation.Opt[ui.Color]
	onErrorContainer        foundation.Opt[ui.Color]
	surfaceDim              foundation.Opt[ui.Color]
	surfaceBright           foundation.Opt[ui.Color]
	surfaceContainerLowest  foundation.Opt[ui.Color]
	surfaceContainerLow     foundation.Opt[ui.Color]
	surfaceContainer        foundation.Opt[ui.Color]
	surfaceContainerHigh    foundation.Opt[ui.Color]
	surfaceContainerHighest foundation.Opt[ui.Color]
	onSurfaceVariant        foundation.Opt[ui.Color]
	outline                 foundation.Opt[ui.Color]
	outlineVariant          foundation.Opt[ui.Color]
	shadow                  foundation.Opt[ui.Color]
	scrim                   foundation.Opt[ui.Color]
	inverseSurface          foundation.Opt[ui.Color]
	onInverseSurface        foundation.Opt[ui.Color]
	inversePrimary          foundation.Opt[ui.Color]
	surfaceTint             foundation.Opt[ui.Color]
	background              foundation.Opt[ui.Color]
	onBackground            foundation.Opt[ui.Color]
	surfaceVariant          foundation.Opt[ui.Color]
}

// ColorSchemeOpts are the optional parameters of NewColorScheme: the colors that
// Flutter's constructor lets out.
//
// Flutter: material/color_scheme.dart ColorScheme.new
type ColorSchemeOpts struct {
	// PrimaryContainer is the color primaryContainer; the default is the color that the getter falls back to.
	PrimaryContainer foundation.Opt[ui.Color]
	// OnPrimaryContainer is the color onPrimaryContainer; the default is the color that the getter falls back to.
	OnPrimaryContainer foundation.Opt[ui.Color]
	// PrimaryFixed is the color primaryFixed; the default is the color that the getter falls back to.
	PrimaryFixed foundation.Opt[ui.Color]
	// PrimaryFixedDim is the color primaryFixedDim; the default is the color that the getter falls back to.
	PrimaryFixedDim foundation.Opt[ui.Color]
	// OnPrimaryFixed is the color onPrimaryFixed; the default is the color that the getter falls back to.
	OnPrimaryFixed foundation.Opt[ui.Color]
	// OnPrimaryFixedVariant is the color onPrimaryFixedVariant; the default is the color that the getter falls back to.
	OnPrimaryFixedVariant foundation.Opt[ui.Color]
	// SecondaryContainer is the color secondaryContainer; the default is the color that the getter falls back to.
	SecondaryContainer foundation.Opt[ui.Color]
	// OnSecondaryContainer is the color onSecondaryContainer; the default is the color that the getter falls back to.
	OnSecondaryContainer foundation.Opt[ui.Color]
	// SecondaryFixed is the color secondaryFixed; the default is the color that the getter falls back to.
	SecondaryFixed foundation.Opt[ui.Color]
	// SecondaryFixedDim is the color secondaryFixedDim; the default is the color that the getter falls back to.
	SecondaryFixedDim foundation.Opt[ui.Color]
	// OnSecondaryFixed is the color onSecondaryFixed; the default is the color that the getter falls back to.
	OnSecondaryFixed foundation.Opt[ui.Color]
	// OnSecondaryFixedVariant is the color onSecondaryFixedVariant; the default is the color that the getter falls back to.
	OnSecondaryFixedVariant foundation.Opt[ui.Color]
	// Tertiary is the color tertiary; the default is the color that the getter falls back to.
	Tertiary foundation.Opt[ui.Color]
	// OnTertiary is the color onTertiary; the default is the color that the getter falls back to.
	OnTertiary foundation.Opt[ui.Color]
	// TertiaryContainer is the color tertiaryContainer; the default is the color that the getter falls back to.
	TertiaryContainer foundation.Opt[ui.Color]
	// OnTertiaryContainer is the color onTertiaryContainer; the default is the color that the getter falls back to.
	OnTertiaryContainer foundation.Opt[ui.Color]
	// TertiaryFixed is the color tertiaryFixed; the default is the color that the getter falls back to.
	TertiaryFixed foundation.Opt[ui.Color]
	// TertiaryFixedDim is the color tertiaryFixedDim; the default is the color that the getter falls back to.
	TertiaryFixedDim foundation.Opt[ui.Color]
	// OnTertiaryFixed is the color onTertiaryFixed; the default is the color that the getter falls back to.
	OnTertiaryFixed foundation.Opt[ui.Color]
	// OnTertiaryFixedVariant is the color onTertiaryFixedVariant; the default is the color that the getter falls back to.
	OnTertiaryFixedVariant foundation.Opt[ui.Color]
	// ErrorContainer is the color errorContainer; the default is the color that the getter falls back to.
	ErrorContainer foundation.Opt[ui.Color]
	// OnErrorContainer is the color onErrorContainer; the default is the color that the getter falls back to.
	OnErrorContainer foundation.Opt[ui.Color]
	// SurfaceDim is the color surfaceDim; the default is the color that the getter falls back to.
	SurfaceDim foundation.Opt[ui.Color]
	// SurfaceBright is the color surfaceBright; the default is the color that the getter falls back to.
	SurfaceBright foundation.Opt[ui.Color]
	// SurfaceContainerLowest is the color surfaceContainerLowest; the default is the color that the getter falls back to.
	SurfaceContainerLowest foundation.Opt[ui.Color]
	// SurfaceContainerLow is the color surfaceContainerLow; the default is the color that the getter falls back to.
	SurfaceContainerLow foundation.Opt[ui.Color]
	// SurfaceContainer is the color surfaceContainer; the default is the color that the getter falls back to.
	SurfaceContainer foundation.Opt[ui.Color]
	// SurfaceContainerHigh is the color surfaceContainerHigh; the default is the color that the getter falls back to.
	SurfaceContainerHigh foundation.Opt[ui.Color]
	// SurfaceContainerHighest is the color surfaceContainerHighest; the default is the color that the getter falls back to.
	SurfaceContainerHighest foundation.Opt[ui.Color]
	// OnSurfaceVariant is the color onSurfaceVariant; the default is the color that the getter falls back to.
	OnSurfaceVariant foundation.Opt[ui.Color]
	// Outline is the color outline; the default is the color that the getter falls back to.
	Outline foundation.Opt[ui.Color]
	// OutlineVariant is the color outlineVariant; the default is the color that the getter falls back to.
	OutlineVariant foundation.Opt[ui.Color]
	// Shadow is the color shadow; the default is the color that the getter falls back to.
	Shadow foundation.Opt[ui.Color]
	// Scrim is the color scrim; the default is the color that the getter falls back to.
	Scrim foundation.Opt[ui.Color]
	// InverseSurface is the color inverseSurface; the default is the color that the getter falls back to.
	InverseSurface foundation.Opt[ui.Color]
	// OnInverseSurface is the color onInverseSurface; the default is the color that the getter falls back to.
	OnInverseSurface foundation.Opt[ui.Color]
	// InversePrimary is the color inversePrimary; the default is the color that the getter falls back to.
	InversePrimary foundation.Opt[ui.Color]
	// SurfaceTint is the color surfaceTint; the default is the color that the getter falls back to.
	SurfaceTint foundation.Opt[ui.Color]
	// Background is the color background; the default is the color that the getter falls back to. Deprecated in Flutter: use surface.
	Background foundation.Opt[ui.Color]
	// OnBackground is the color onBackground; the default is the color that the getter falls back to. Deprecated in Flutter: use onSurface.
	OnBackground foundation.Opt[ui.Color]
	// SurfaceVariant is the color surfaceVariant; the default is the color that the getter falls back to. Deprecated in Flutter: use surfaceContainerHighest.
	SurfaceVariant foundation.Opt[ui.Color]
}

// NewColorScheme creates a color scheme from its required colors, in Flutter's
// declaration order, and the optional ones. In debug builds nothing is checked:
// every color is a valid value.
//
// Flutter: material/color_scheme.dart ColorScheme.new
func NewColorScheme(brightness ui.Brightness, primary, onPrimary, secondary, onSecondary, err, onError, surface, onSurface ui.Color, opts ...ColorSchemeOpts) *ColorScheme {
	o := firstOpts(opts)
	if brightness == ui.BrightnessUnspecified {
		brightness = ui.BrightnessLight
	}
	return &ColorScheme{
		brightness: brightness, primary: primary, onPrimary: onPrimary, secondary: secondary, onSecondary: onSecondary,
		errorColor: err, onError: onError, surface: surface, onSurface: onSurface,
		primaryContainer:        o.PrimaryContainer,
		onPrimaryContainer:      o.OnPrimaryContainer,
		primaryFixed:            o.PrimaryFixed,
		primaryFixedDim:         o.PrimaryFixedDim,
		onPrimaryFixed:          o.OnPrimaryFixed,
		onPrimaryFixedVariant:   o.OnPrimaryFixedVariant,
		secondaryContainer:      o.SecondaryContainer,
		onSecondaryContainer:    o.OnSecondaryContainer,
		secondaryFixed:          o.SecondaryFixed,
		secondaryFixedDim:       o.SecondaryFixedDim,
		onSecondaryFixed:        o.OnSecondaryFixed,
		onSecondaryFixedVariant: o.OnSecondaryFixedVariant,
		tertiary:                o.Tertiary,
		onTertiary:              o.OnTertiary,
		tertiaryContainer:       o.TertiaryContainer,
		onTertiaryContainer:     o.OnTertiaryContainer,
		tertiaryFixed:           o.TertiaryFixed,
		tertiaryFixedDim:        o.TertiaryFixedDim,
		onTertiaryFixed:         o.OnTertiaryFixed,
		onTertiaryFixedVariant:  o.OnTertiaryFixedVariant,
		errorContainer:          o.ErrorContainer,
		onErrorContainer:        o.OnErrorContainer,
		surfaceDim:              o.SurfaceDim,
		surfaceBright:           o.SurfaceBright,
		surfaceContainerLowest:  o.SurfaceContainerLowest,
		surfaceContainerLow:     o.SurfaceContainerLow,
		surfaceContainer:        o.SurfaceContainer,
		surfaceContainerHigh:    o.SurfaceContainerHigh,
		surfaceContainerHighest: o.SurfaceContainerHighest,
		onSurfaceVariant:        o.OnSurfaceVariant,
		outline:                 o.Outline,
		outlineVariant:          o.OutlineVariant,
		shadow:                  o.Shadow,
		scrim:                   o.Scrim,
		inverseSurface:          o.InverseSurface,
		onInverseSurface:        o.OnInverseSurface,
		inversePrimary:          o.InversePrimary,
		surfaceTint:             o.SurfaceTint,
		background:              o.Background,
		onBackground:            o.OnBackground,
		surfaceVariant:          o.SurfaceVariant,
	}
}

// Brightness is the overall brightness of this color scheme.
//
// Flutter: material/color_scheme.dart ColorScheme.brightness
func (s *ColorScheme) Brightness() ui.Brightness { return s.brightness }

// Primary is the color primary.
//
// Flutter: material/color_scheme.dart ColorScheme.primary
func (s *ColorScheme) Primary() ui.Color { return s.primary }

// OnPrimary is the color onPrimary.
//
// Flutter: material/color_scheme.dart ColorScheme.onPrimary
func (s *ColorScheme) OnPrimary() ui.Color { return s.onPrimary }

// Secondary is the color secondary.
//
// Flutter: material/color_scheme.dart ColorScheme.secondary
func (s *ColorScheme) Secondary() ui.Color { return s.secondary }

// OnSecondary is the color onSecondary.
//
// Flutter: material/color_scheme.dart ColorScheme.onSecondary
func (s *ColorScheme) OnSecondary() ui.Color { return s.onSecondary }

// Error is the color error.
//
// Flutter: material/color_scheme.dart ColorScheme.error
func (s *ColorScheme) Error() ui.Color { return s.errorColor }

// OnError is the color onError.
//
// Flutter: material/color_scheme.dart ColorScheme.onError
func (s *ColorScheme) OnError() ui.Color { return s.onError }

// Surface is the color surface.
//
// Flutter: material/color_scheme.dart ColorScheme.surface
func (s *ColorScheme) Surface() ui.Color { return s.surface }

// OnSurface is the color onSurface.
//
// Flutter: material/color_scheme.dart ColorScheme.onSurface
func (s *ColorScheme) OnSurface() ui.Color { return s.onSurface }

// PrimaryContainer is the color primaryContainer, or primary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.primaryContainer
func (s *ColorScheme) PrimaryContainer() ui.Color { return s.primaryContainer.OrElse(s.Primary()) }

// OnPrimaryContainer is the color onPrimaryContainer, or onPrimary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onPrimaryContainer
func (s *ColorScheme) OnPrimaryContainer() ui.Color {
	return s.onPrimaryContainer.OrElse(s.OnPrimary())
}

// PrimaryFixed is the color primaryFixed, or primary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.primaryFixed
func (s *ColorScheme) PrimaryFixed() ui.Color { return s.primaryFixed.OrElse(s.Primary()) }

// PrimaryFixedDim is the color primaryFixedDim, or primary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.primaryFixedDim
func (s *ColorScheme) PrimaryFixedDim() ui.Color { return s.primaryFixedDim.OrElse(s.Primary()) }

// OnPrimaryFixed is the color onPrimaryFixed, or onPrimary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onPrimaryFixed
func (s *ColorScheme) OnPrimaryFixed() ui.Color { return s.onPrimaryFixed.OrElse(s.OnPrimary()) }

// OnPrimaryFixedVariant is the color onPrimaryFixedVariant, or onPrimary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onPrimaryFixedVariant
func (s *ColorScheme) OnPrimaryFixedVariant() ui.Color {
	return s.onPrimaryFixedVariant.OrElse(s.OnPrimary())
}

// SecondaryContainer is the color secondaryContainer, or secondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.secondaryContainer
func (s *ColorScheme) SecondaryContainer() ui.Color {
	return s.secondaryContainer.OrElse(s.Secondary())
}

// OnSecondaryContainer is the color onSecondaryContainer, or onSecondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onSecondaryContainer
func (s *ColorScheme) OnSecondaryContainer() ui.Color {
	return s.onSecondaryContainer.OrElse(s.OnSecondary())
}

// SecondaryFixed is the color secondaryFixed, or secondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.secondaryFixed
func (s *ColorScheme) SecondaryFixed() ui.Color { return s.secondaryFixed.OrElse(s.Secondary()) }

// SecondaryFixedDim is the color secondaryFixedDim, or secondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.secondaryFixedDim
func (s *ColorScheme) SecondaryFixedDim() ui.Color { return s.secondaryFixedDim.OrElse(s.Secondary()) }

// OnSecondaryFixed is the color onSecondaryFixed, or onSecondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onSecondaryFixed
func (s *ColorScheme) OnSecondaryFixed() ui.Color { return s.onSecondaryFixed.OrElse(s.OnSecondary()) }

// OnSecondaryFixedVariant is the color onSecondaryFixedVariant, or onSecondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onSecondaryFixedVariant
func (s *ColorScheme) OnSecondaryFixedVariant() ui.Color {
	return s.onSecondaryFixedVariant.OrElse(s.OnSecondary())
}

// Tertiary is the color tertiary, or secondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.tertiary
func (s *ColorScheme) Tertiary() ui.Color { return s.tertiary.OrElse(s.Secondary()) }

// OnTertiary is the color onTertiary, or onSecondary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onTertiary
func (s *ColorScheme) OnTertiary() ui.Color { return s.onTertiary.OrElse(s.OnSecondary()) }

// TertiaryContainer is the color tertiaryContainer, or tertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.tertiaryContainer
func (s *ColorScheme) TertiaryContainer() ui.Color { return s.tertiaryContainer.OrElse(s.Tertiary()) }

// OnTertiaryContainer is the color onTertiaryContainer, or onTertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onTertiaryContainer
func (s *ColorScheme) OnTertiaryContainer() ui.Color {
	return s.onTertiaryContainer.OrElse(s.OnTertiary())
}

// TertiaryFixed is the color tertiaryFixed, or tertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.tertiaryFixed
func (s *ColorScheme) TertiaryFixed() ui.Color { return s.tertiaryFixed.OrElse(s.Tertiary()) }

// TertiaryFixedDim is the color tertiaryFixedDim, or tertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.tertiaryFixedDim
func (s *ColorScheme) TertiaryFixedDim() ui.Color { return s.tertiaryFixedDim.OrElse(s.Tertiary()) }

// OnTertiaryFixed is the color onTertiaryFixed, or onTertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onTertiaryFixed
func (s *ColorScheme) OnTertiaryFixed() ui.Color { return s.onTertiaryFixed.OrElse(s.OnTertiary()) }

// OnTertiaryFixedVariant is the color onTertiaryFixedVariant, or onTertiary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onTertiaryFixedVariant
func (s *ColorScheme) OnTertiaryFixedVariant() ui.Color {
	return s.onTertiaryFixedVariant.OrElse(s.OnTertiary())
}

// ErrorContainer is the color errorContainer, or error when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.errorContainer
func (s *ColorScheme) ErrorContainer() ui.Color { return s.errorContainer.OrElse(s.Error()) }

// OnErrorContainer is the color onErrorContainer, or onError when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onErrorContainer
func (s *ColorScheme) OnErrorContainer() ui.Color { return s.onErrorContainer.OrElse(s.OnError()) }

// SurfaceDim is the color surfaceDim, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceDim
func (s *ColorScheme) SurfaceDim() ui.Color { return s.surfaceDim.OrElse(s.Surface()) }

// SurfaceBright is the color surfaceBright, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceBright
func (s *ColorScheme) SurfaceBright() ui.Color { return s.surfaceBright.OrElse(s.Surface()) }

// SurfaceContainerLowest is the color surfaceContainerLowest, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceContainerLowest
func (s *ColorScheme) SurfaceContainerLowest() ui.Color {
	return s.surfaceContainerLowest.OrElse(s.Surface())
}

// SurfaceContainerLow is the color surfaceContainerLow, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceContainerLow
func (s *ColorScheme) SurfaceContainerLow() ui.Color {
	return s.surfaceContainerLow.OrElse(s.Surface())
}

// SurfaceContainer is the color surfaceContainer, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceContainer
func (s *ColorScheme) SurfaceContainer() ui.Color { return s.surfaceContainer.OrElse(s.Surface()) }

// SurfaceContainerHigh is the color surfaceContainerHigh, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceContainerHigh
func (s *ColorScheme) SurfaceContainerHigh() ui.Color {
	return s.surfaceContainerHigh.OrElse(s.Surface())
}

// SurfaceContainerHighest is the color surfaceContainerHighest, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceContainerHighest
func (s *ColorScheme) SurfaceContainerHighest() ui.Color {
	return s.surfaceContainerHighest.OrElse(s.Surface())
}

// OnSurfaceVariant is the color onSurfaceVariant, or onSurface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onSurfaceVariant
func (s *ColorScheme) OnSurfaceVariant() ui.Color { return s.onSurfaceVariant.OrElse(s.OnSurface()) }

// Outline is the color outline, or onBackground when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.outline
func (s *ColorScheme) Outline() ui.Color { return s.outline.OrElse(s.OnBackground()) }

// OutlineVariant is the color outlineVariant, or onBackground when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.outlineVariant
func (s *ColorScheme) OutlineVariant() ui.Color { return s.outlineVariant.OrElse(s.OnBackground()) }

// Shadow is the color shadow, or opaque black when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.shadow
func (s *ColorScheme) Shadow() ui.Color { return s.shadow.OrElse(ui.NewColor(0xff000000)) }

// Scrim is the color scrim, or opaque black when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.scrim
func (s *ColorScheme) Scrim() ui.Color { return s.scrim.OrElse(ui.NewColor(0xff000000)) }

// InverseSurface is the color inverseSurface, or onSurface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.inverseSurface
func (s *ColorScheme) InverseSurface() ui.Color { return s.inverseSurface.OrElse(s.OnSurface()) }

// OnInverseSurface is the color onInverseSurface, or surface when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.onInverseSurface
func (s *ColorScheme) OnInverseSurface() ui.Color { return s.onInverseSurface.OrElse(s.Surface()) }

// InversePrimary is the color inversePrimary, or onPrimary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.inversePrimary
func (s *ColorScheme) InversePrimary() ui.Color { return s.inversePrimary.OrElse(s.OnPrimary()) }

// SurfaceTint is the color surfaceTint, or primary when the scheme does not set it.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceTint
func (s *ColorScheme) SurfaceTint() ui.Color { return s.surfaceTint.OrElse(s.Primary()) }

// Background is the color background, or surface when the scheme does not set it. It is deprecated in Flutter, which says to use surface.
//
// Flutter: material/color_scheme.dart ColorScheme.background
func (s *ColorScheme) Background() ui.Color { return s.background.OrElse(s.Surface()) }

// OnBackground is the color onBackground, or onSurface when the scheme does not set it. It is deprecated in Flutter, which says to use onSurface.
//
// Flutter: material/color_scheme.dart ColorScheme.onBackground
func (s *ColorScheme) OnBackground() ui.Color { return s.onBackground.OrElse(s.OnSurface()) }

// SurfaceVariant is the color surfaceVariant, or surface when the scheme does not set it. It is deprecated in Flutter, which says to use surfaceContainerHighest.
//
// Flutter: material/color_scheme.dart ColorScheme.surfaceVariant
func (s *ColorScheme) SurfaceVariant() ui.Color { return s.surfaceVariant.OrElse(s.Surface()) }

// ColorSchemeCopy holds the replacements for ColorScheme.CopyWith: a field that is
// None keeps the value that the copied scheme has for it, which for a color it
// did not set is the color its getter falls back to.
//
// Flutter: material/color_scheme.dart ColorScheme.copyWith
type ColorSchemeCopy struct {
	Brightness              ui.Brightness
	Primary                 foundation.Opt[ui.Color]
	OnPrimary               foundation.Opt[ui.Color]
	PrimaryContainer        foundation.Opt[ui.Color]
	OnPrimaryContainer      foundation.Opt[ui.Color]
	PrimaryFixed            foundation.Opt[ui.Color]
	PrimaryFixedDim         foundation.Opt[ui.Color]
	OnPrimaryFixed          foundation.Opt[ui.Color]
	OnPrimaryFixedVariant   foundation.Opt[ui.Color]
	Secondary               foundation.Opt[ui.Color]
	OnSecondary             foundation.Opt[ui.Color]
	SecondaryContainer      foundation.Opt[ui.Color]
	OnSecondaryContainer    foundation.Opt[ui.Color]
	SecondaryFixed          foundation.Opt[ui.Color]
	SecondaryFixedDim       foundation.Opt[ui.Color]
	OnSecondaryFixed        foundation.Opt[ui.Color]
	OnSecondaryFixedVariant foundation.Opt[ui.Color]
	Tertiary                foundation.Opt[ui.Color]
	OnTertiary              foundation.Opt[ui.Color]
	TertiaryContainer       foundation.Opt[ui.Color]
	OnTertiaryContainer     foundation.Opt[ui.Color]
	TertiaryFixed           foundation.Opt[ui.Color]
	TertiaryFixedDim        foundation.Opt[ui.Color]
	OnTertiaryFixed         foundation.Opt[ui.Color]
	OnTertiaryFixedVariant  foundation.Opt[ui.Color]
	Error                   foundation.Opt[ui.Color]
	OnError                 foundation.Opt[ui.Color]
	ErrorContainer          foundation.Opt[ui.Color]
	OnErrorContainer        foundation.Opt[ui.Color]
	Surface                 foundation.Opt[ui.Color]
	OnSurface               foundation.Opt[ui.Color]
	SurfaceDim              foundation.Opt[ui.Color]
	SurfaceBright           foundation.Opt[ui.Color]
	SurfaceContainerLowest  foundation.Opt[ui.Color]
	SurfaceContainerLow     foundation.Opt[ui.Color]
	SurfaceContainer        foundation.Opt[ui.Color]
	SurfaceContainerHigh    foundation.Opt[ui.Color]
	SurfaceContainerHighest foundation.Opt[ui.Color]
	OnSurfaceVariant        foundation.Opt[ui.Color]
	Outline                 foundation.Opt[ui.Color]
	OutlineVariant          foundation.Opt[ui.Color]
	Shadow                  foundation.Opt[ui.Color]
	Scrim                   foundation.Opt[ui.Color]
	InverseSurface          foundation.Opt[ui.Color]
	OnInverseSurface        foundation.Opt[ui.Color]
	InversePrimary          foundation.Opt[ui.Color]
	SurfaceTint             foundation.Opt[ui.Color]
	Background              foundation.Opt[ui.Color]
	OnBackground            foundation.Opt[ui.Color]
	SurfaceVariant          foundation.Opt[ui.Color]
}

// CopyWith creates a copy of the scheme with the given fields replaced. As in
// Flutter, the colors that are not replaced are the resolved ones, so a color
// that followed the primary color keeps the old primary color when only the
// primary color is replaced.
//
// Flutter: material/color_scheme.dart ColorScheme.copyWith
func (s *ColorScheme) CopyWith(c ColorSchemeCopy) *ColorScheme {
	brightness := s.brightness
	if c.Brightness != ui.BrightnessUnspecified {
		brightness = c.Brightness
	}
	return NewColorScheme(brightness,
		c.Primary.OrElse(s.Primary()),
		c.OnPrimary.OrElse(s.OnPrimary()),
		c.Secondary.OrElse(s.Secondary()),
		c.OnSecondary.OrElse(s.OnSecondary()),
		c.Error.OrElse(s.Error()),
		c.OnError.OrElse(s.OnError()),
		c.Surface.OrElse(s.Surface()),
		c.OnSurface.OrElse(s.OnSurface()),
		ColorSchemeOpts{
			PrimaryContainer:        foundation.Some(c.PrimaryContainer.OrElse(s.PrimaryContainer())),
			OnPrimaryContainer:      foundation.Some(c.OnPrimaryContainer.OrElse(s.OnPrimaryContainer())),
			PrimaryFixed:            foundation.Some(c.PrimaryFixed.OrElse(s.PrimaryFixed())),
			PrimaryFixedDim:         foundation.Some(c.PrimaryFixedDim.OrElse(s.PrimaryFixedDim())),
			OnPrimaryFixed:          foundation.Some(c.OnPrimaryFixed.OrElse(s.OnPrimaryFixed())),
			OnPrimaryFixedVariant:   foundation.Some(c.OnPrimaryFixedVariant.OrElse(s.OnPrimaryFixedVariant())),
			SecondaryContainer:      foundation.Some(c.SecondaryContainer.OrElse(s.SecondaryContainer())),
			OnSecondaryContainer:    foundation.Some(c.OnSecondaryContainer.OrElse(s.OnSecondaryContainer())),
			SecondaryFixed:          foundation.Some(c.SecondaryFixed.OrElse(s.SecondaryFixed())),
			SecondaryFixedDim:       foundation.Some(c.SecondaryFixedDim.OrElse(s.SecondaryFixedDim())),
			OnSecondaryFixed:        foundation.Some(c.OnSecondaryFixed.OrElse(s.OnSecondaryFixed())),
			OnSecondaryFixedVariant: foundation.Some(c.OnSecondaryFixedVariant.OrElse(s.OnSecondaryFixedVariant())),
			Tertiary:                foundation.Some(c.Tertiary.OrElse(s.Tertiary())),
			OnTertiary:              foundation.Some(c.OnTertiary.OrElse(s.OnTertiary())),
			TertiaryContainer:       foundation.Some(c.TertiaryContainer.OrElse(s.TertiaryContainer())),
			OnTertiaryContainer:     foundation.Some(c.OnTertiaryContainer.OrElse(s.OnTertiaryContainer())),
			TertiaryFixed:           foundation.Some(c.TertiaryFixed.OrElse(s.TertiaryFixed())),
			TertiaryFixedDim:        foundation.Some(c.TertiaryFixedDim.OrElse(s.TertiaryFixedDim())),
			OnTertiaryFixed:         foundation.Some(c.OnTertiaryFixed.OrElse(s.OnTertiaryFixed())),
			OnTertiaryFixedVariant:  foundation.Some(c.OnTertiaryFixedVariant.OrElse(s.OnTertiaryFixedVariant())),
			ErrorContainer:          foundation.Some(c.ErrorContainer.OrElse(s.ErrorContainer())),
			OnErrorContainer:        foundation.Some(c.OnErrorContainer.OrElse(s.OnErrorContainer())),
			SurfaceDim:              foundation.Some(c.SurfaceDim.OrElse(s.SurfaceDim())),
			SurfaceBright:           foundation.Some(c.SurfaceBright.OrElse(s.SurfaceBright())),
			SurfaceContainerLowest:  foundation.Some(c.SurfaceContainerLowest.OrElse(s.SurfaceContainerLowest())),
			SurfaceContainerLow:     foundation.Some(c.SurfaceContainerLow.OrElse(s.SurfaceContainerLow())),
			SurfaceContainer:        foundation.Some(c.SurfaceContainer.OrElse(s.SurfaceContainer())),
			SurfaceContainerHigh:    foundation.Some(c.SurfaceContainerHigh.OrElse(s.SurfaceContainerHigh())),
			SurfaceContainerHighest: foundation.Some(c.SurfaceContainerHighest.OrElse(s.SurfaceContainerHighest())),
			OnSurfaceVariant:        foundation.Some(c.OnSurfaceVariant.OrElse(s.OnSurfaceVariant())),
			Outline:                 foundation.Some(c.Outline.OrElse(s.Outline())),
			OutlineVariant:          foundation.Some(c.OutlineVariant.OrElse(s.OutlineVariant())),
			Shadow:                  foundation.Some(c.Shadow.OrElse(s.Shadow())),
			Scrim:                   foundation.Some(c.Scrim.OrElse(s.Scrim())),
			InverseSurface:          foundation.Some(c.InverseSurface.OrElse(s.InverseSurface())),
			OnInverseSurface:        foundation.Some(c.OnInverseSurface.OrElse(s.OnInverseSurface())),
			InversePrimary:          foundation.Some(c.InversePrimary.OrElse(s.InversePrimary())),
			SurfaceTint:             foundation.Some(c.SurfaceTint.OrElse(s.SurfaceTint())),
			Background:              foundation.Some(c.Background.OrElse(s.Background())),
			OnBackground:            foundation.Some(c.OnBackground.OrElse(s.OnBackground())),
			SurfaceVariant:          foundation.Some(c.SurfaceVariant.OrElse(s.SurfaceVariant())),
		})
}

// ColorSchemeLerp linearly interpolates between two color schemes. The brightness
// is the one of the scheme that is closer.
//
// Flutter: material/color_scheme.dart ColorScheme.lerp
func ColorSchemeLerp(a, b *ColorScheme, t float64) *ColorScheme {
	if a == b {
		return a
	}
	lerp := func(x, y ui.Color) ui.Color { return *ui.ColorLerp(&x, &y, t) }
	brightness := b.brightness
	if t < 0.5 {
		brightness = a.brightness
	}
	return NewColorScheme(brightness,
		lerp(a.Primary(), b.Primary()),
		lerp(a.OnPrimary(), b.OnPrimary()),
		lerp(a.Secondary(), b.Secondary()),
		lerp(a.OnSecondary(), b.OnSecondary()),
		lerp(a.Error(), b.Error()),
		lerp(a.OnError(), b.OnError()),
		lerp(a.Surface(), b.Surface()),
		lerp(a.OnSurface(), b.OnSurface()),
		ColorSchemeOpts{
			PrimaryContainer:        foundation.Some(lerp(a.PrimaryContainer(), b.PrimaryContainer())),
			OnPrimaryContainer:      foundation.Some(lerp(a.OnPrimaryContainer(), b.OnPrimaryContainer())),
			PrimaryFixed:            foundation.Some(lerp(a.PrimaryFixed(), b.PrimaryFixed())),
			PrimaryFixedDim:         foundation.Some(lerp(a.PrimaryFixedDim(), b.PrimaryFixedDim())),
			OnPrimaryFixed:          foundation.Some(lerp(a.OnPrimaryFixed(), b.OnPrimaryFixed())),
			OnPrimaryFixedVariant:   foundation.Some(lerp(a.OnPrimaryFixedVariant(), b.OnPrimaryFixedVariant())),
			SecondaryContainer:      foundation.Some(lerp(a.SecondaryContainer(), b.SecondaryContainer())),
			OnSecondaryContainer:    foundation.Some(lerp(a.OnSecondaryContainer(), b.OnSecondaryContainer())),
			SecondaryFixed:          foundation.Some(lerp(a.SecondaryFixed(), b.SecondaryFixed())),
			SecondaryFixedDim:       foundation.Some(lerp(a.SecondaryFixedDim(), b.SecondaryFixedDim())),
			OnSecondaryFixed:        foundation.Some(lerp(a.OnSecondaryFixed(), b.OnSecondaryFixed())),
			OnSecondaryFixedVariant: foundation.Some(lerp(a.OnSecondaryFixedVariant(), b.OnSecondaryFixedVariant())),
			Tertiary:                foundation.Some(lerp(a.Tertiary(), b.Tertiary())),
			OnTertiary:              foundation.Some(lerp(a.OnTertiary(), b.OnTertiary())),
			TertiaryContainer:       foundation.Some(lerp(a.TertiaryContainer(), b.TertiaryContainer())),
			OnTertiaryContainer:     foundation.Some(lerp(a.OnTertiaryContainer(), b.OnTertiaryContainer())),
			TertiaryFixed:           foundation.Some(lerp(a.TertiaryFixed(), b.TertiaryFixed())),
			TertiaryFixedDim:        foundation.Some(lerp(a.TertiaryFixedDim(), b.TertiaryFixedDim())),
			OnTertiaryFixed:         foundation.Some(lerp(a.OnTertiaryFixed(), b.OnTertiaryFixed())),
			OnTertiaryFixedVariant:  foundation.Some(lerp(a.OnTertiaryFixedVariant(), b.OnTertiaryFixedVariant())),
			ErrorContainer:          foundation.Some(lerp(a.ErrorContainer(), b.ErrorContainer())),
			OnErrorContainer:        foundation.Some(lerp(a.OnErrorContainer(), b.OnErrorContainer())),
			SurfaceDim:              foundation.Some(lerp(a.SurfaceDim(), b.SurfaceDim())),
			SurfaceBright:           foundation.Some(lerp(a.SurfaceBright(), b.SurfaceBright())),
			SurfaceContainerLowest:  foundation.Some(lerp(a.SurfaceContainerLowest(), b.SurfaceContainerLowest())),
			SurfaceContainerLow:     foundation.Some(lerp(a.SurfaceContainerLow(), b.SurfaceContainerLow())),
			SurfaceContainer:        foundation.Some(lerp(a.SurfaceContainer(), b.SurfaceContainer())),
			SurfaceContainerHigh:    foundation.Some(lerp(a.SurfaceContainerHigh(), b.SurfaceContainerHigh())),
			SurfaceContainerHighest: foundation.Some(lerp(a.SurfaceContainerHighest(), b.SurfaceContainerHighest())),
			OnSurfaceVariant:        foundation.Some(lerp(a.OnSurfaceVariant(), b.OnSurfaceVariant())),
			Outline:                 foundation.Some(lerp(a.Outline(), b.Outline())),
			OutlineVariant:          foundation.Some(lerp(a.OutlineVariant(), b.OutlineVariant())),
			Shadow:                  foundation.Some(lerp(a.Shadow(), b.Shadow())),
			Scrim:                   foundation.Some(lerp(a.Scrim(), b.Scrim())),
			InverseSurface:          foundation.Some(lerp(a.InverseSurface(), b.InverseSurface())),
			OnInverseSurface:        foundation.Some(lerp(a.OnInverseSurface(), b.OnInverseSurface())),
			InversePrimary:          foundation.Some(lerp(a.InversePrimary(), b.InversePrimary())),
			SurfaceTint:             foundation.Some(lerp(a.SurfaceTint(), b.SurfaceTint())),
			Background:              foundation.Some(lerp(a.Background(), b.Background())),
			OnBackground:            foundation.Some(lerp(a.OnBackground(), b.OnBackground())),
			SurfaceVariant:          foundation.Some(lerp(a.SurfaceVariant(), b.SurfaceVariant())),
		})
}

// Equal reports whether the schemes have the same brightness and the same
// resolved colors.
//
// Flutter: material/color_scheme.dart ColorScheme.==
func (s *ColorScheme) Equal(o *ColorScheme) bool {
	if s == o {
		return true
	}
	if s == nil || o == nil {
		return false
	}
	return s.brightness == o.brightness &&
		s.Primary() == o.Primary() &&
		s.OnPrimary() == o.OnPrimary() &&
		s.Secondary() == o.Secondary() &&
		s.OnSecondary() == o.OnSecondary() &&
		s.Error() == o.Error() &&
		s.OnError() == o.OnError() &&
		s.Surface() == o.Surface() &&
		s.OnSurface() == o.OnSurface() &&
		s.PrimaryContainer() == o.PrimaryContainer() &&
		s.OnPrimaryContainer() == o.OnPrimaryContainer() &&
		s.PrimaryFixed() == o.PrimaryFixed() &&
		s.PrimaryFixedDim() == o.PrimaryFixedDim() &&
		s.OnPrimaryFixed() == o.OnPrimaryFixed() &&
		s.OnPrimaryFixedVariant() == o.OnPrimaryFixedVariant() &&
		s.SecondaryContainer() == o.SecondaryContainer() &&
		s.OnSecondaryContainer() == o.OnSecondaryContainer() &&
		s.SecondaryFixed() == o.SecondaryFixed() &&
		s.SecondaryFixedDim() == o.SecondaryFixedDim() &&
		s.OnSecondaryFixed() == o.OnSecondaryFixed() &&
		s.OnSecondaryFixedVariant() == o.OnSecondaryFixedVariant() &&
		s.Tertiary() == o.Tertiary() &&
		s.OnTertiary() == o.OnTertiary() &&
		s.TertiaryContainer() == o.TertiaryContainer() &&
		s.OnTertiaryContainer() == o.OnTertiaryContainer() &&
		s.TertiaryFixed() == o.TertiaryFixed() &&
		s.TertiaryFixedDim() == o.TertiaryFixedDim() &&
		s.OnTertiaryFixed() == o.OnTertiaryFixed() &&
		s.OnTertiaryFixedVariant() == o.OnTertiaryFixedVariant() &&
		s.ErrorContainer() == o.ErrorContainer() &&
		s.OnErrorContainer() == o.OnErrorContainer() &&
		s.SurfaceDim() == o.SurfaceDim() &&
		s.SurfaceBright() == o.SurfaceBright() &&
		s.SurfaceContainerLowest() == o.SurfaceContainerLowest() &&
		s.SurfaceContainerLow() == o.SurfaceContainerLow() &&
		s.SurfaceContainer() == o.SurfaceContainer() &&
		s.SurfaceContainerHigh() == o.SurfaceContainerHigh() &&
		s.SurfaceContainerHighest() == o.SurfaceContainerHighest() &&
		s.OnSurfaceVariant() == o.OnSurfaceVariant() &&
		s.Outline() == o.Outline() &&
		s.OutlineVariant() == o.OutlineVariant() &&
		s.Shadow() == o.Shadow() &&
		s.Scrim() == o.Scrim() &&
		s.InverseSurface() == o.InverseSurface() &&
		s.OnInverseSurface() == o.OnInverseSurface() &&
		s.InversePrimary() == o.InversePrimary() &&
		s.SurfaceTint() == o.SurfaceTint() &&
		s.Background() == o.Background() &&
		s.OnBackground() == o.OnBackground() &&
		s.SurfaceVariant() == o.SurfaceVariant()
}

// DebugFillProperties adds the colors that differ from the default light scheme.
//
// Flutter: material/color_scheme.dart ColorScheme.debugFillProperties
func (s *ColorScheme) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	def := NewColorSchemeLight()
	properties.Add(foundation.NewDiagnosticsProperty("brightness", s.brightness, foundation.DiagnosticsPropertyOpts{DefaultValue: def.brightness}))
	properties.Add(painting.NewColorProperty("primary", foundation.Some(s.Primary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Primary()}))
	properties.Add(painting.NewColorProperty("onPrimary", foundation.Some(s.OnPrimary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnPrimary()}))
	properties.Add(painting.NewColorProperty("secondary", foundation.Some(s.Secondary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Secondary()}))
	properties.Add(painting.NewColorProperty("onSecondary", foundation.Some(s.OnSecondary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSecondary()}))
	properties.Add(painting.NewColorProperty("error", foundation.Some(s.Error()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Error()}))
	properties.Add(painting.NewColorProperty("onError", foundation.Some(s.OnError()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnError()}))
	properties.Add(painting.NewColorProperty("surface", foundation.Some(s.Surface()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Surface()}))
	properties.Add(painting.NewColorProperty("onSurface", foundation.Some(s.OnSurface()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSurface()}))
	properties.Add(painting.NewColorProperty("primaryContainer", foundation.Some(s.PrimaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.PrimaryContainer()}))
	properties.Add(painting.NewColorProperty("onPrimaryContainer", foundation.Some(s.OnPrimaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnPrimaryContainer()}))
	properties.Add(painting.NewColorProperty("primaryFixed", foundation.Some(s.PrimaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.PrimaryFixed()}))
	properties.Add(painting.NewColorProperty("primaryFixedDim", foundation.Some(s.PrimaryFixedDim()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.PrimaryFixedDim()}))
	properties.Add(painting.NewColorProperty("onPrimaryFixed", foundation.Some(s.OnPrimaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnPrimaryFixed()}))
	properties.Add(painting.NewColorProperty("onPrimaryFixedVariant", foundation.Some(s.OnPrimaryFixedVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnPrimaryFixedVariant()}))
	properties.Add(painting.NewColorProperty("secondaryContainer", foundation.Some(s.SecondaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SecondaryContainer()}))
	properties.Add(painting.NewColorProperty("onSecondaryContainer", foundation.Some(s.OnSecondaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSecondaryContainer()}))
	properties.Add(painting.NewColorProperty("secondaryFixed", foundation.Some(s.SecondaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SecondaryFixed()}))
	properties.Add(painting.NewColorProperty("secondaryFixedDim", foundation.Some(s.SecondaryFixedDim()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SecondaryFixedDim()}))
	properties.Add(painting.NewColorProperty("onSecondaryFixed", foundation.Some(s.OnSecondaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSecondaryFixed()}))
	properties.Add(painting.NewColorProperty("onSecondaryFixedVariant", foundation.Some(s.OnSecondaryFixedVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSecondaryFixedVariant()}))
	properties.Add(painting.NewColorProperty("tertiary", foundation.Some(s.Tertiary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Tertiary()}))
	properties.Add(painting.NewColorProperty("onTertiary", foundation.Some(s.OnTertiary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnTertiary()}))
	properties.Add(painting.NewColorProperty("tertiaryContainer", foundation.Some(s.TertiaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.TertiaryContainer()}))
	properties.Add(painting.NewColorProperty("onTertiaryContainer", foundation.Some(s.OnTertiaryContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnTertiaryContainer()}))
	properties.Add(painting.NewColorProperty("tertiaryFixed", foundation.Some(s.TertiaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.TertiaryFixed()}))
	properties.Add(painting.NewColorProperty("tertiaryFixedDim", foundation.Some(s.TertiaryFixedDim()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.TertiaryFixedDim()}))
	properties.Add(painting.NewColorProperty("onTertiaryFixed", foundation.Some(s.OnTertiaryFixed()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnTertiaryFixed()}))
	properties.Add(painting.NewColorProperty("onTertiaryFixedVariant", foundation.Some(s.OnTertiaryFixedVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnTertiaryFixedVariant()}))
	properties.Add(painting.NewColorProperty("errorContainer", foundation.Some(s.ErrorContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.ErrorContainer()}))
	properties.Add(painting.NewColorProperty("onErrorContainer", foundation.Some(s.OnErrorContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnErrorContainer()}))
	properties.Add(painting.NewColorProperty("surfaceDim", foundation.Some(s.SurfaceDim()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceDim()}))
	properties.Add(painting.NewColorProperty("surfaceBright", foundation.Some(s.SurfaceBright()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceBright()}))
	properties.Add(painting.NewColorProperty("surfaceContainerLowest", foundation.Some(s.SurfaceContainerLowest()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceContainerLowest()}))
	properties.Add(painting.NewColorProperty("surfaceContainerLow", foundation.Some(s.SurfaceContainerLow()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceContainerLow()}))
	properties.Add(painting.NewColorProperty("surfaceContainer", foundation.Some(s.SurfaceContainer()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceContainer()}))
	properties.Add(painting.NewColorProperty("surfaceContainerHigh", foundation.Some(s.SurfaceContainerHigh()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceContainerHigh()}))
	properties.Add(painting.NewColorProperty("surfaceContainerHighest", foundation.Some(s.SurfaceContainerHighest()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceContainerHighest()}))
	properties.Add(painting.NewColorProperty("onSurfaceVariant", foundation.Some(s.OnSurfaceVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnSurfaceVariant()}))
	properties.Add(painting.NewColorProperty("outline", foundation.Some(s.Outline()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Outline()}))
	properties.Add(painting.NewColorProperty("outlineVariant", foundation.Some(s.OutlineVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OutlineVariant()}))
	properties.Add(painting.NewColorProperty("shadow", foundation.Some(s.Shadow()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Shadow()}))
	properties.Add(painting.NewColorProperty("scrim", foundation.Some(s.Scrim()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Scrim()}))
	properties.Add(painting.NewColorProperty("inverseSurface", foundation.Some(s.InverseSurface()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.InverseSurface()}))
	properties.Add(painting.NewColorProperty("onInverseSurface", foundation.Some(s.OnInverseSurface()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnInverseSurface()}))
	properties.Add(painting.NewColorProperty("inversePrimary", foundation.Some(s.InversePrimary()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.InversePrimary()}))
	properties.Add(painting.NewColorProperty("surfaceTint", foundation.Some(s.SurfaceTint()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceTint()}))
	properties.Add(painting.NewColorProperty("background", foundation.Some(s.Background()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.Background()}))
	properties.Add(painting.NewColorProperty("onBackground", foundation.Some(s.OnBackground()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.OnBackground()}))
	properties.Add(painting.NewColorProperty("surfaceVariant", foundation.Some(s.SurfaceVariant()), foundation.DiagnosticsPropertyOpts{DefaultValue: def.SurfaceVariant()}))
}
