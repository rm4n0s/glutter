package material

import (
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Deviation: Flutter's ThemeData also holds the component themes (appBarTheme,
// buttonTheme...), adaptations, cupertinoOverrideTheme, inputDecorationTheme,
// pageTransitionsTheme, scrollbarTheme and splashFactory. The step that ports
// each of them adds its field to ThemeData and to ThemeDataOpts (SPEC §12.10).

// ThemeExtension is an interface that defines how to copy and interpolate a
// theme object that is attached to a ThemeData with ThemeDataOpts.Extensions.
// The extension is found again by its dynamic type, with ThemeDataExtension.
//
// Deviation: Dart's CopyWith takes the fields of the subclass; the interface
// method has none, and an extension offers its own typed CopyWith.
//
// Flutter: material/theme_data.dart ThemeExtension
type ThemeExtension interface {
	// CopyWith returns a copy of the extension.
	//
	// Flutter: material/theme_data.dart ThemeExtension.copyWith
	CopyWith() ThemeExtension
	// Lerp linearly interpolates with another extension of the same type, which
	// is nil when the other theme has none.
	//
	// Flutter: material/theme_data.dart ThemeExtension.lerp
	Lerp(other ThemeExtension, t float64) ThemeExtension
}

// MaterialTapTargetSize configures the tap target and layout size of certain
// Material widgets.
//
// Flutter: material/theme_data.dart MaterialTapTargetSize
type MaterialTapTargetSize int

const (
	// MaterialTapTargetSizeUnspecified means the default of the platform.
	MaterialTapTargetSizeUnspecified MaterialTapTargetSize = iota
	// MaterialTapTargetSizePadded expands the minimum tap target size to 48px
	// by 48px.
	//
	// Flutter: material/theme_data.dart MaterialTapTargetSize.padded
	MaterialTapTargetSizePadded
	// MaterialTapTargetSizeShrinkWrap shrinks the tap target size to the size of
	// the component.
	//
	// Flutter: material/theme_data.dart MaterialTapTargetSize.shrinkWrap
	MaterialTapTargetSizeShrinkWrap
)

// Name returns the name of the value, as Dart's Enum.name.
func (s MaterialTapTargetSize) Name() string {
	switch s {
	case MaterialTapTargetSizePadded:
		return "padded"
	case MaterialTapTargetSizeShrinkWrap:
		return "shrinkWrap"
	}
	return "unspecified"
}

// String returns "MaterialTapTargetSize.padded"...
func (s MaterialTapTargetSize) String() string { return "MaterialTapTargetSize." + s.Name() }

// VisualDensity defines the compactness of the visual layout of components.
// Densities go from -4 to 4 in steps of one, which change the size of a
// component by four logical pixels.
//
// Flutter: material/theme_data.dart VisualDensity
type VisualDensity struct {
	// Horizontal is the horizontal density.
	Horizontal float64
	// Vertical is the vertical density.
	Vertical float64
}

// VisualDensityOpts are the optional parameters of NewVisualDensity and
// VisualDensity.CopyWith.
//
// Flutter: material/theme_data.dart VisualDensity.new
type VisualDensityOpts struct {
	// Horizontal is the horizontal density; the default is 0.
	Horizontal foundation.Opt[float64]
	// Vertical is the vertical density; the default is 0.
	Vertical foundation.Opt[float64]
}

const (
	// VisualDensityMinimum is the minimum density in either direction.
	//
	// Flutter: material/theme_data.dart VisualDensity.minimumDensity
	VisualDensityMinimum = -4.0
	// VisualDensityMaximum is the maximum density in either direction.
	//
	// Flutter: material/theme_data.dart VisualDensity.maximumDensity
	VisualDensityMaximum = 4.0
)

// NewVisualDensity creates a density. In debug builds a density outside -4 to
// 4 panics with the tag VisualDensityOutOfRange.
//
// Flutter: material/theme_data.dart VisualDensity.new
func NewVisualDensity(opts ...VisualDensityOpts) VisualDensity {
	o := firstOpts(opts)
	d := VisualDensity{o.Horizontal.OrElse(0), o.Vertical.OrElse(0)}
	if foundation.DebugMode {
		for _, v := range [...]float64{d.Horizontal, d.Vertical} {
			if v < VisualDensityMinimum || v > VisualDensityMaximum {
				panic(errors.New("VisualDensityOutOfRange", "the density must be from -4 to 4", "density", v))
			}
		}
	}
	return d
}

// VisualDensityStandard is the default profile for VisualDensity.
//
// Flutter: material/theme_data.dart VisualDensity.standard
func VisualDensityStandard() VisualDensity { return VisualDensity{} }

// VisualDensityComfortable is a profile that is a bit less dense than the
// standard.
//
// Flutter: material/theme_data.dart VisualDensity.comfortable
func VisualDensityComfortable() VisualDensity { return VisualDensity{-1, -1} }

// VisualDensityCompact is a profile that is denser than the standard.
//
// Flutter: material/theme_data.dart VisualDensity.compact
func VisualDensityCompact() VisualDensity { return VisualDensity{-2, -2} }

// VisualDensityAdaptivePlatformDensity is the density that suits the target
// platform.
//
// Flutter: material/theme_data.dart VisualDensity.adaptivePlatformDensity
func VisualDensityAdaptivePlatformDensity() VisualDensity {
	return VisualDensityDefaultForPlatform(foundation.DefaultTargetPlatform())
}

// VisualDensityDefaultForPlatform returns the density that suits a platform:
// standard on the mobile ones, compact on the desktop ones.
//
// Flutter: material/theme_data.dart VisualDensity.defaultDensityForPlatform
func VisualDensityDefaultForPlatform(platform foundation.TargetPlatform) VisualDensity {
	switch platform {
	case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		return VisualDensityCompact()
	}
	return VisualDensityStandard()
}

// CopyWith returns a copy with the densities of c that are set replaced.
//
// Flutter: material/theme_data.dart VisualDensity.copyWith
func (d VisualDensity) CopyWith(c VisualDensityOpts) VisualDensity {
	return NewVisualDensity(VisualDensityOpts{
		Horizontal: foundation.Some(c.Horizontal.OrElse(d.Horizontal)),
		Vertical:   foundation.Some(c.Vertical.OrElse(d.Vertical)),
	})
}

// BaseSizeAdjustment is the density as a size adjustment: four logical pixels
// per step.
//
// Flutter: material/theme_data.dart VisualDensity.baseSizeAdjustment
func (d VisualDensity) BaseSizeAdjustment() ui.Offset {
	const interval = 4.0
	return ui.Offset{Dx: d.Horizontal, Dy: d.Vertical}.Mul(interval)
}

// VisualDensityLerp linearly interpolates between two densities.
//
// Flutter: material/theme_data.dart VisualDensity.lerp
func VisualDensityLerp(a, b VisualDensity, t float64) VisualDensity {
	if a == b {
		return a
	}
	return NewVisualDensity(VisualDensityOpts{
		Horizontal: foundation.Some(ui.LerpDouble(a.Horizontal, b.Horizontal, t)),
		Vertical:   foundation.Some(ui.LerpDouble(a.Vertical, b.Vertical, t)),
	})
}

// EffectiveConstraints returns the constraints with their minimum sizes
// adjusted by the density.
//
// Flutter: material/theme_data.dart VisualDensity.effectiveConstraints
func (d VisualDensity) EffectiveConstraints(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	adj := d.BaseSizeAdjustment()
	return constraints.CopyWith(rendering.BoxConstraintsCopy{
		MinWidth:  foundation.Some(ui.ClampDouble(constraints.MinWidth+adj.Dx, 0, constraints.MaxWidth)),
		MinHeight: foundation.Some(ui.ClampDouble(constraints.MinHeight+adj.Dy, 0, constraints.MaxHeight)),
	})
}

// ToStringShort returns "VisualDensity(h: 0.0, v: 0.0)".
//
// Flutter: material/theme_data.dart VisualDensity.toStringShort
func (d VisualDensity) ToStringShort() string {
	return "VisualDensity(h: " + foundation.DebugFormatDouble(foundation.Some(d.Horizontal)) + ", v: " + foundation.DebugFormatDouble(foundation.Some(d.Vertical)) + ")"
}

// String returns the description of the density.
func (d VisualDensity) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the densities that are not zero.
//
// Flutter: material/theme_data.dart VisualDensity.debugFillProperties
func (d VisualDensity) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewDoubleProperty("horizontal", foundation.Some(d.Horizontal), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	properties.Add(foundation.NewDoubleProperty("vertical", foundation.Some(d.Vertical), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
}

// ThemeDataOpts are the optional parameters of NewThemeData. Most of them are
// defaults for the fields of ThemeData: those that are not set take the value
// that Flutter's ThemeData factory gives them.
//
// Flutter: material/theme_data.dart ThemeData.new
type ThemeDataOpts struct {
	// GENERAL CONFIGURATION

	// ApplyElevationOverlayColor says whether a surface at elevation gets a
	// lighter overlay in a dark theme.
	ApplyElevationOverlayColor foundation.Opt[bool]
	// Extensions are the arbitrary objects that are attached to the theme.
	Extensions []ThemeExtension
	// MaterialTapTargetSize is the size of the tap targets; the default is
	// padded on the mobile platforms and shrinkWrap on the desktop ones.
	MaterialTapTargetSize MaterialTapTargetSize
	// Platform is the platform that widgets adapt to; the default is
	// foundation.DefaultTargetPlatform.
	Platform foundation.Opt[foundation.TargetPlatform]
	// UseMaterial3 says whether the components use Material 3; the default is
	// true.
	UseMaterial3 foundation.Opt[bool]
	// VisualDensity is the density of the components; the default suits the
	// platform.
	VisualDensity foundation.Opt[VisualDensity]

	// COLOR

	// ColorScheme is the colors of the theme; it is the preferred way to set
	// them.
	ColorScheme *ColorScheme
	// Brightness is the overall brightness; the default is the one of the color
	// scheme, or light.
	Brightness ui.Brightness
	// ColorSchemeSeed makes the color scheme from a seed color; it must not be
	// given with ColorScheme, PrimarySwatch or PrimaryColor.
	ColorSchemeSeed foundation.Opt[ui.Color]
	// CanvasColor is the default color of Material of type canvas.
	CanvasColor foundation.Opt[ui.Color]
	// CardColor is the color of Material used by Card and similar.
	CardColor foundation.Opt[ui.Color]
	// DisabledColor is the color of disabled widgets.
	DisabledColor foundation.Opt[ui.Color]
	// DividerColor is the color of Divider and PopupMenuDivider.
	DividerColor foundation.Opt[ui.Color]
	// FocusColor is the focus color of the ink responses.
	FocusColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color of the ink responses.
	HighlightColor foundation.Opt[ui.Color]
	// HintColor is the color of hint and placeholder text.
	HintColor foundation.Opt[ui.Color]
	// HoverColor is the hover color of the ink responses.
	HoverColor foundation.Opt[ui.Color]
	// PrimaryColor is the background color of the major parts of the app.
	PrimaryColor foundation.Opt[ui.Color]
	// PrimaryColorDark is a darker version of PrimaryColor.
	PrimaryColorDark foundation.Opt[ui.Color]
	// PrimaryColorLight is a lighter version of PrimaryColor.
	PrimaryColorLight foundation.Opt[ui.Color]
	// PrimarySwatch is the swatch that the other colors are made from when the
	// scheme is not Material 3; the default is blue.
	PrimarySwatch *MaterialColor
	// ScaffoldBackgroundColor is the default color of Scaffold.
	ScaffoldBackgroundColor foundation.Opt[ui.Color]
	// SecondaryHeaderColor is the color of a data table's header when rows are
	// selected.
	SecondaryHeaderColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadows of Material.
	ShadowColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the ink responses.
	SplashColor foundation.Opt[ui.Color]
	// SplashFactory defines the appearance of ink splashes; the default is
	// InkRippleFactory (Material 3) or InkSplashFactory (Material 2).
	SplashFactory InteractiveInkFeatureFactory
	// UnselectedWidgetColor is the color of inactive toggleables.
	UnselectedWidgetColor foundation.Opt[ui.Color]

	// TYPOGRAPHY & ICONOGRAPHY

	// FontFamily replaces the font family of the text themes.
	FontFamily string
	// FontFamilyFallback replaces the font family fallback of the text themes.
	FontFamilyFallback []string
	// Package is the package that holds the font of the text themes.
	Package string
	// IconTheme is the default icon theme.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// PrimaryIconTheme is the icon theme that contrasts with PrimaryColor.
	PrimaryIconTheme foundation.Opt[widgets.IconThemeData]
	// PrimaryTextTheme is merged over the text theme that contrasts with
	// PrimaryColor.
	PrimaryTextTheme *TextTheme
	// TextTheme is merged over the default text theme.
	TextTheme *TextTheme
	// Typography is the color and geometry of the text themes; the default is
	// Material 2021 (or 2014 without Material 3).
	Typography *Typography
	// PageTransitionsTheme is the transitions of the pages that Material makes;
	// the default is the one of NewPageTransitionsTheme.
	PageTransitionsTheme *PageTransitionsTheme

	// DEPRECATED

	// DialogBackgroundColor is the background color of Dialog.
	DialogBackgroundColor foundation.Opt[ui.Color]
	// IndicatorColor is the color of the selected tab indicator.
	IndicatorColor foundation.Opt[ui.Color]
}

// ThemeData defines the configuration of the overall visual Theme for a
// MaterialApp or a widget subtree within the app. Create it with NewThemeData;
// its fields are for reading.
//
// Flutter: material/theme_data.dart ThemeData
type ThemeData struct {
	// GENERAL CONFIGURATION

	// ApplyElevationOverlayColor says whether a surface at elevation gets a
	// lighter overlay in a dark theme.
	ApplyElevationOverlayColor bool
	// Extensions are the arbitrary objects that are attached to the theme.
	Extensions []ThemeExtension
	// MaterialTapTargetSize is the size of the tap targets.
	MaterialTapTargetSize MaterialTapTargetSize
	// Platform is the platform that widgets adapt to.
	Platform foundation.TargetPlatform
	// UseMaterial3 says whether the components use Material 3.
	UseMaterial3 bool
	// VisualDensity is the density of the components.
	VisualDensity VisualDensity

	// COLOR

	// CanvasColor is the default color of Material of type canvas.
	CanvasColor ui.Color
	// CardColor is the color of Material used by Card and similar.
	CardColor ui.Color
	// ColorScheme is the colors of the theme.
	ColorScheme *ColorScheme
	// DisabledColor is the color of disabled widgets.
	DisabledColor ui.Color
	// DividerColor is the color of Divider and PopupMenuDivider.
	DividerColor ui.Color
	// FocusColor is the focus color of the ink responses.
	FocusColor ui.Color
	// HighlightColor is the highlight color of the ink responses.
	HighlightColor ui.Color
	// HintColor is the color of hint and placeholder text.
	HintColor ui.Color
	// HoverColor is the hover color of the ink responses.
	HoverColor ui.Color
	// PrimaryColor is the background color of the major parts of the app.
	PrimaryColor ui.Color
	// PrimaryColorDark is a darker version of PrimaryColor.
	PrimaryColorDark ui.Color
	// PrimaryColorLight is a lighter version of PrimaryColor.
	PrimaryColorLight ui.Color
	// ScaffoldBackgroundColor is the default color of Scaffold.
	ScaffoldBackgroundColor ui.Color
	// SecondaryHeaderColor is the color of a data table's header when rows are
	// selected.
	SecondaryHeaderColor ui.Color
	// ShadowColor is the color of the shadows of Material.
	ShadowColor ui.Color
	// SplashColor is the splash color of the ink responses.
	SplashColor ui.Color
	// SplashFactory defines the appearance of ink splashes.
	SplashFactory InteractiveInkFeatureFactory
	// UnselectedWidgetColor is the color of inactive toggleables.
	UnselectedWidgetColor ui.Color

	// TYPOGRAPHY & ICONOGRAPHY

	// IconTheme is the default icon theme.
	IconTheme widgets.IconThemeData
	// PrimaryIconTheme is the icon theme that contrasts with PrimaryColor.
	PrimaryIconTheme widgets.IconThemeData
	// PrimaryTextTheme is the text theme that contrasts with PrimaryColor.
	PrimaryTextTheme *TextTheme
	// TextTheme is the text theme that contrasts with the background.
	TextTheme *TextTheme
	// Typography is the color and geometry of the text themes.
	Typography *Typography
	// PageTransitionsTheme is the transitions of the pages that Material makes.
	PageTransitionsTheme *PageTransitionsTheme

	// DEPRECATED

	// DialogBackgroundColor is the background color of Dialog.
	DialogBackgroundColor ui.Color
	// IndicatorColor is the color of the selected tab indicator.
	IndicatorColor ui.Color
}

// Brightness is the overall brightness of this theme: the one of its color
// scheme.
//
// Flutter: material/theme_data.dart ThemeData.brightness
func (d *ThemeData) Brightness() ui.Brightness { return d.ColorScheme.Brightness() }

// NewThemeData creates a theme. Every color, text theme and so on that opts
// does not set is derived the way Flutter's ThemeData factory derives it: from
// the color scheme with Material 3, or from the primary swatch without it. In
// debug builds an inconsistent combination (a brightness different from the
// one of the color scheme, or a seed together with a color scheme, a swatch or
// a primary color) panics with the tag ThemeDataBrightnessMismatch or ThemeDataSeedConflict.
//
// Flutter: material/theme_data.dart ThemeData.new
func NewThemeData(opts ...ThemeDataOpts) *ThemeData {
	o := firstOpts(opts)
	extensions := dedupeExtensions(o.Extensions)
	platform := o.Platform.OrElse(foundation.DefaultTargetPlatform())
	tapTarget := o.MaterialTapTargetSize
	if tapTarget == MaterialTapTargetSizeUnspecified {
		switch platform {
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
			tapTarget = MaterialTapTargetSizePadded
		default:
			tapTarget = MaterialTapTargetSizeShrinkWrap
		}
	}
	visualDensity := o.VisualDensity.OrElse(VisualDensityDefaultForPlatform(platform))
	useMaterial3 := o.UseMaterial3.OrElse(true)

	// COLOR
	colorScheme := o.ColorScheme
	if foundation.DebugMode {
		if colorScheme != nil && o.Brightness != ui.BrightnessUnspecified && colorScheme.Brightness() != o.Brightness {
			panic(errors.New("ThemeDataBrightnessMismatch", "ThemeData.brightness does not match ColorScheme.brightness",
				"brightness", o.Brightness.String(), "colorScheme", colorScheme.Brightness().String()))
		}
		if o.ColorSchemeSeed.IsSome() && (colorScheme != nil || o.PrimarySwatch != nil || o.PrimaryColor.IsSome()) {
			panic(errors.New("ThemeDataSeedConflict", "colorSchemeSeed cannot be given with colorScheme, primarySwatch or primaryColor"))
		}
	}
	effectiveBrightness := o.Brightness
	if effectiveBrightness == ui.BrightnessUnspecified {
		effectiveBrightness = ui.BrightnessLight
		if colorScheme != nil {
			effectiveBrightness = colorScheme.Brightness()
		}
	}
	isDark := effectiveBrightness == ui.BrightnessDark

	primaryColor, canvasColor, scaffoldBackgroundColor, cardColor := o.PrimaryColor, o.CanvasColor, o.ScaffoldBackgroundColor, o.CardColor
	dividerColor, dialogBackgroundColor, indicatorColor := o.DividerColor, o.DialogBackgroundColor, o.IndicatorColor
	applyElevationOverlayColor := o.ApplyElevationOverlayColor
	if seed, ok := o.ColorSchemeSeed.Get(); ok || useMaterial3 {
		if ok {
			colorScheme = ColorSchemeFromSeed(seed, ColorSchemeFromSeedOpts{Brightness: effectiveBrightness})
		}
		if colorScheme == nil {
			colorScheme = colorSchemeLightM3
			if isDark {
				colorScheme = colorSchemeDarkM3
			}
		}
		// For surfaces that use primary color in light themes and surface color in dark.
		primarySurfaceColor, onPrimarySurfaceColor := colorScheme.Primary(), colorScheme.OnPrimary()
		if isDark {
			primarySurfaceColor, onPrimarySurfaceColor = colorScheme.Surface(), colorScheme.OnSurface()
		}
		primaryColor = orSome(primaryColor, primarySurfaceColor)
		canvasColor = orSome(canvasColor, colorScheme.Surface())
		scaffoldBackgroundColor = orSome(scaffoldBackgroundColor, colorScheme.Surface())
		cardColor = orSome(cardColor, colorScheme.Surface())
		dividerColor = orSome(dividerColor, colorScheme.Outline())
		dialogBackgroundColor = orSome(dialogBackgroundColor, colorScheme.Surface())
		indicatorColor = orSome(indicatorColor, onPrimarySurfaceColor)
		if !applyElevationOverlayColor.IsSome() {
			// Flutter compares the brightness parameter here, not the effective one.
			applyElevationOverlayColor = foundation.Some(o.Brightness == ui.BrightnessDark)
		}
	}
	if !applyElevationOverlayColor.IsSome() {
		applyElevationOverlayColor = foundation.Some(false)
	}
	swatch := Colors.Blue
	if o.PrimarySwatch != nil {
		swatch = *o.PrimarySwatch
	}
	grey := func(shade int) ui.Color { c, _ := Colors.Grey.At(shade); return c }
	pick := func(dark, light ui.Color) ui.Color {
		if isDark {
			return dark
		}
		return light
	}
	if !primaryColor.IsSome() {
		primaryColor = foundation.Some(pick(grey(900), swatch.Primary))
	}
	estimatedPrimaryColorBrightness := EstimateBrightnessForColor(primaryColor.Value())
	primaryColorLight := orSome(o.PrimaryColorLight, pick(grey(500), swatch.Shade100()))
	primaryColorDark := orSome(o.PrimaryColorDark, pick(Colors.Black, swatch.Shade700()))
	primaryIsDark := estimatedPrimaryColorBrightness == ui.BrightnessDark
	focusColor := orSome(o.FocusColor, pick(Colors.White.WithOpacity(0.12), Colors.Black.WithOpacity(0.12)))
	hoverColor := orSome(o.HoverColor, pick(Colors.White.WithOpacity(0.04), Colors.Black.WithOpacity(0.04)))
	shadowColor := orSome(o.ShadowColor, Colors.Black)
	canvasColor = orSome(canvasColor, pick(grey(850), grey(50)))
	scaffoldBackgroundColor = orSome(scaffoldBackgroundColor, canvasColor.Value())
	cardColor = orSome(cardColor, pick(grey(800), Colors.White))
	dividerColor = orSome(dividerColor, pick(ui.NewColor(0x1FFFFFFF), ui.NewColor(0x1F000000)))

	// Create a ColorScheme that is as backwards compatible as possible with the
	// existing default ThemeData color values.
	if colorScheme == nil {
		accent := swatch.Shade500()
		backgroundColor := swatch.Shade200()
		if isDark {
			accent, _ = Colors.TealAccent.At(200)
			backgroundColor = grey(700)
		}
		red700, _ := Colors.Red.At(700)
		colorScheme = ColorSchemeFromSwatch(ColorSchemeFromSwatchOpts{
			PrimarySwatch: &swatch, AccentColor: foundation.Some(accent), CardColor: cardColor,
			BackgroundColor: foundation.Some(backgroundColor), ErrorColor: foundation.Some(red700), Brightness: effectiveBrightness,
		})
	}
	unselectedWidgetColor := orSome(o.UnselectedWidgetColor, pick(Colors.White70, Colors.Black54))
	// The spec doesn't specify a dark theme secondaryHeaderColor, this is a guess.
	secondaryHeaderColor := orSome(o.SecondaryHeaderColor, pick(grey(700), swatch.Shade50()))
	hintColor := orSome(o.HintColor, pick(Colors.White60, Colors.Black.WithOpacity(0.6)))
	disabledColor := orSome(o.DisabledColor, pick(Colors.White38, Colors.Black38))
	highlightColor := orSome(o.HighlightColor, pick(ui.NewColor(0x40CCCCCC), ui.NewColor(0x66BCBCBC)))
	splashFactory := o.SplashFactory
	if splashFactory == nil {
		// Deviation: Flutter picks InkSparkle for Material 3 on Android; every platform here uses InkRipple.
		splashFactory = InkSplashFactory
		if useMaterial3 {
			splashFactory = InkRippleFactory
		}
	}
	splashColor := orSome(o.SplashColor, pick(ui.NewColor(0x40CCCCCC), ui.NewColor(0x66C8C8C8)))

	pageTransitionsTheme := o.PageTransitionsTheme
	if pageTransitionsTheme == nil {
		pageTransitionsTheme = NewPageTransitionsTheme()
	}

	// TYPOGRAPHY & ICONOGRAPHY
	typography := o.Typography
	if typography == nil {
		if useMaterial3 {
			typography = NewTypographyMaterial2021(Typography2021Opts{
				TypographyOpts: TypographyOpts{Platform: foundation.Some(platform)}, ColorScheme: colorScheme,
			})
		} else {
			typography = NewTypographyMaterial2014(TypographyOpts{Platform: foundation.Some(platform)})
		}
	}
	defaultTextTheme, defaultPrimaryTextTheme := typography.Black, typography.Black
	if isDark {
		defaultTextTheme = typography.White
	}
	if primaryIsDark {
		defaultPrimaryTextTheme = typography.White
	}
	if o.FontFamily != "" {
		defaultTextTheme = defaultTextTheme.Apply(TextThemeApply{FontFamily: o.FontFamily})
		defaultPrimaryTextTheme = defaultPrimaryTextTheme.Apply(TextThemeApply{FontFamily: o.FontFamily})
	}
	if o.FontFamilyFallback != nil {
		defaultTextTheme = defaultTextTheme.Apply(TextThemeApply{FontFamilyFallback: o.FontFamilyFallback})
		defaultPrimaryTextTheme = defaultPrimaryTextTheme.Apply(TextThemeApply{FontFamilyFallback: o.FontFamilyFallback})
	}
	if o.Package != "" {
		defaultTextTheme = defaultTextTheme.Apply(TextThemeApply{Package: o.Package})
		defaultPrimaryTextTheme = defaultPrimaryTextTheme.Apply(TextThemeApply{Package: o.Package})
	}
	iconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(DefaultIconDarkColor)})
	if isDark {
		iconTheme = widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(DefaultIconLightColor)})
	}
	primaryIconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(Colors.Black)})
	if primaryIsDark {
		primaryIconTheme = widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(Colors.White)})
	}

	// DEPRECATED
	dialogBackgroundColor = orSome(dialogBackgroundColor, pick(grey(800), Colors.White))
	if !indicatorColor.IsSome() {
		if colorScheme.Secondary() == primaryColor.Value() {
			indicatorColor = foundation.Some(Colors.White)
		} else {
			indicatorColor = foundation.Some(colorScheme.Secondary())
		}
	}

	return &ThemeData{
		ApplyElevationOverlayColor: applyElevationOverlayColor.Value(),
		Extensions:                 extensions,
		MaterialTapTargetSize:      tapTarget,
		Platform:                   platform,
		UseMaterial3:               useMaterial3,
		VisualDensity:              visualDensity,

		CanvasColor:             canvasColor.Value(),
		CardColor:               cardColor.Value(),
		ColorScheme:             colorScheme,
		DisabledColor:           disabledColor.Value(),
		DividerColor:            dividerColor.Value(),
		FocusColor:              focusColor.Value(),
		HighlightColor:          highlightColor.Value(),
		HintColor:               hintColor.Value(),
		HoverColor:              hoverColor.Value(),
		PrimaryColor:            primaryColor.Value(),
		PrimaryColorDark:        primaryColorDark.Value(),
		PrimaryColorLight:       primaryColorLight.Value(),
		ScaffoldBackgroundColor: scaffoldBackgroundColor.Value(),
		SecondaryHeaderColor:    secondaryHeaderColor.Value(),
		ShadowColor:             shadowColor.Value(),
		SplashColor:             splashColor.Value(),
		SplashFactory:           splashFactory,
		UnselectedWidgetColor:   unselectedWidgetColor.Value(),

		IconTheme:        o.IconTheme.OrElse(iconTheme),
		PrimaryIconTheme: o.PrimaryIconTheme.OrElse(primaryIconTheme),
		PrimaryTextTheme: defaultPrimaryTextTheme.Merge(o.PrimaryTextTheme),
		TextTheme:        defaultTextTheme.Merge(o.TextTheme),
		Typography:       typography,

		PageTransitionsTheme: pageTransitionsTheme,

		DialogBackgroundColor: dialogBackgroundColor.Value(),
		IndicatorColor:        indicatorColor.Value(),
	}
}

// ThemeDataCopy holds the replacements for ThemeData.CopyWith: a field that is None keeps the value of the
// copied theme.
//
// Flutter: material/theme_data.dart ThemeData.copyWith
type ThemeDataCopy struct {
	ApplyElevationOverlayColor foundation.Opt[bool]
	Extensions                 foundation.Opt[[]ThemeExtension]
	MaterialTapTargetSize      foundation.Opt[MaterialTapTargetSize]
	Platform                   foundation.Opt[foundation.TargetPlatform]
	UseMaterial3               foundation.Opt[bool]
	VisualDensity              foundation.Opt[VisualDensity]
	CanvasColor                foundation.Opt[ui.Color]
	CardColor                  foundation.Opt[ui.Color]
	ColorScheme                foundation.Opt[*ColorScheme]
	// Brightness replaces the brightness of the color scheme (the one given by ColorScheme, if any, or the one of the theme).
	Brightness              ui.Brightness
	DisabledColor           foundation.Opt[ui.Color]
	DividerColor            foundation.Opt[ui.Color]
	FocusColor              foundation.Opt[ui.Color]
	HighlightColor          foundation.Opt[ui.Color]
	HintColor               foundation.Opt[ui.Color]
	HoverColor              foundation.Opt[ui.Color]
	PrimaryColor            foundation.Opt[ui.Color]
	PrimaryColorDark        foundation.Opt[ui.Color]
	PrimaryColorLight       foundation.Opt[ui.Color]
	ScaffoldBackgroundColor foundation.Opt[ui.Color]
	SecondaryHeaderColor    foundation.Opt[ui.Color]
	ShadowColor             foundation.Opt[ui.Color]
	SplashColor             foundation.Opt[ui.Color]
	SplashFactory           foundation.Opt[InteractiveInkFeatureFactory]
	UnselectedWidgetColor   foundation.Opt[ui.Color]
	IconTheme               foundation.Opt[widgets.IconThemeData]
	PrimaryIconTheme        foundation.Opt[widgets.IconThemeData]
	PrimaryTextTheme        foundation.Opt[*TextTheme]
	TextTheme               foundation.Opt[*TextTheme]
	Typography              foundation.Opt[*Typography]
	PageTransitionsTheme    foundation.Opt[*PageTransitionsTheme]
	DialogBackgroundColor   foundation.Opt[ui.Color]
	IndicatorColor          foundation.Opt[ui.Color]
}

// CopyWith creates a copy of this theme but with the fields of c that are set
// replaced. A Brightness replaces only the brightness of the color scheme; the
// other colors stay as they are.
//
// Flutter: material/theme_data.dart ThemeData.copyWith
func (d *ThemeData) CopyWith(c ThemeDataCopy) *ThemeData {
	out := *d
	out.ApplyElevationOverlayColor = c.ApplyElevationOverlayColor.OrElse(d.ApplyElevationOverlayColor)
	out.Extensions = c.Extensions.OrElse(d.Extensions)
	out.MaterialTapTargetSize = c.MaterialTapTargetSize.OrElse(d.MaterialTapTargetSize)
	out.Platform = c.Platform.OrElse(d.Platform)
	out.UseMaterial3 = c.UseMaterial3.OrElse(d.UseMaterial3)
	out.VisualDensity = c.VisualDensity.OrElse(d.VisualDensity)
	out.CanvasColor = c.CanvasColor.OrElse(d.CanvasColor)
	out.CardColor = c.CardColor.OrElse(d.CardColor)
	scheme := c.ColorScheme.OrElse(d.ColorScheme)
	if c.Brightness != ui.BrightnessUnspecified {
		scheme = scheme.CopyWith(ColorSchemeCopy{Brightness: c.Brightness})
	}
	out.ColorScheme = scheme
	out.DisabledColor = c.DisabledColor.OrElse(d.DisabledColor)
	out.DividerColor = c.DividerColor.OrElse(d.DividerColor)
	out.FocusColor = c.FocusColor.OrElse(d.FocusColor)
	out.HighlightColor = c.HighlightColor.OrElse(d.HighlightColor)
	out.HintColor = c.HintColor.OrElse(d.HintColor)
	out.HoverColor = c.HoverColor.OrElse(d.HoverColor)
	out.PrimaryColor = c.PrimaryColor.OrElse(d.PrimaryColor)
	out.PrimaryColorDark = c.PrimaryColorDark.OrElse(d.PrimaryColorDark)
	out.PrimaryColorLight = c.PrimaryColorLight.OrElse(d.PrimaryColorLight)
	out.ScaffoldBackgroundColor = c.ScaffoldBackgroundColor.OrElse(d.ScaffoldBackgroundColor)
	out.SecondaryHeaderColor = c.SecondaryHeaderColor.OrElse(d.SecondaryHeaderColor)
	out.ShadowColor = c.ShadowColor.OrElse(d.ShadowColor)
	out.SplashColor = c.SplashColor.OrElse(d.SplashColor)
	out.SplashFactory = c.SplashFactory.OrElse(d.SplashFactory)
	out.UnselectedWidgetColor = c.UnselectedWidgetColor.OrElse(d.UnselectedWidgetColor)
	out.IconTheme = c.IconTheme.OrElse(d.IconTheme)
	out.PrimaryIconTheme = c.PrimaryIconTheme.OrElse(d.PrimaryIconTheme)
	out.PrimaryTextTheme = c.PrimaryTextTheme.OrElse(d.PrimaryTextTheme)
	out.TextTheme = c.TextTheme.OrElse(d.TextTheme)
	out.Typography = c.Typography.OrElse(d.Typography)
	out.PageTransitionsTheme = c.PageTransitionsTheme.OrElse(d.PageTransitionsTheme)
	out.DialogBackgroundColor = c.DialogBackgroundColor.OrElse(d.DialogBackgroundColor)
	out.IndicatorColor = c.IndicatorColor.OrElse(d.IndicatorColor)
	return &out
}

func orSome(o foundation.Opt[ui.Color], def ui.Color) foundation.Opt[ui.Color] {
	if o.IsSome() {
		return o
	}
	return foundation.Some(def)
}

// dedupeExtensions keeps the last extension of each dynamic type, in the order
// in which the types first appear, as Dart's map literal does.
func dedupeExtensions(in []ThemeExtension) []ThemeExtension {
	var out []ThemeExtension
	for _, e := range in {
		replaced := false
		for i, x := range out {
			if reflect.TypeOf(x) == reflect.TypeOf(e) {
				out[i], replaced = e, true
				break
			}
		}
		if !replaced {
			out = append(out, e)
		}
	}
	return out
}

// NewThemeDataFrom creates a theme from a color scheme, deriving the colors
// that are not in the scheme from it, as Flutter's ThemeData.from does.
//
// Flutter: material/theme_data.dart ThemeData.from
func NewThemeDataFrom(colorScheme *ColorScheme, textTheme *TextTheme, useMaterial3 foundation.Opt[bool]) *ThemeData {
	isDark := colorScheme.Brightness() == ui.BrightnessDark
	primarySurfaceColor, onPrimarySurfaceColor := colorScheme.Primary(), colorScheme.OnPrimary()
	if isDark {
		primarySurfaceColor, onPrimarySurfaceColor = colorScheme.Surface(), colorScheme.OnSurface()
	}
	return NewThemeData(ThemeDataOpts{
		ColorScheme:                colorScheme,
		Brightness:                 colorScheme.Brightness(),
		PrimaryColor:               foundation.Some(primarySurfaceColor),
		CanvasColor:                foundation.Some(colorScheme.Surface()),
		ScaffoldBackgroundColor:    foundation.Some(colorScheme.Surface()),
		CardColor:                  foundation.Some(colorScheme.Surface()),
		DividerColor:               foundation.Some(colorScheme.OnSurface().WithOpacity(0.12)),
		DialogBackgroundColor:      foundation.Some(colorScheme.Surface()),
		IndicatorColor:             foundation.Some(onPrimarySurfaceColor),
		TextTheme:                  textTheme,
		ApplyElevationOverlayColor: foundation.Some(isDark),
		UseMaterial3:               useMaterial3,
	})
}

// NewThemeDataLight creates a light theme, the default theme.
//
// Flutter: material/theme_data.dart ThemeData.light
func NewThemeDataLight(useMaterial3 ...bool) *ThemeData {
	return NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessLight, UseMaterial3: optBool(useMaterial3)})
}

// NewThemeDataDark creates a dark theme.
//
// Flutter: material/theme_data.dart ThemeData.dark
func NewThemeDataDark(useMaterial3 ...bool) *ThemeData {
	return NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark, UseMaterial3: optBool(useMaterial3)})
}

// NewThemeDataFallback creates the theme that is used when there is none: the
// light theme.
//
// Flutter: material/theme_data.dart ThemeData.fallback
func NewThemeDataFallback(useMaterial3 ...bool) *ThemeData { return NewThemeDataLight(useMaterial3...) }

// optBool turns the optional bool argument of NewThemeDataLight and
// NewThemeDataDark (Dart's `bool? useMaterial3`) into an Opt.
func optBool(v []bool) foundation.Opt[bool] {
	if len(v) == 0 {
		return foundation.None[bool]()
	}
	return foundation.Some(v[0])
}

// ThemeDataExtension returns the extension of type T that is attached to the
// theme, if any.
//
// Flutter: material/theme_data.dart ThemeData.extension
func ThemeDataExtension[T ThemeExtension](d *ThemeData) (T, bool) {
	for _, e := range d.Extensions {
		if v, ok := e.(T); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// ThemeDataLocalize returns a copy of baseTheme whose text themes have the
// geometry of localTextGeometry merged in. Flutter caches the last five results
// by identity; the Go port does not, as the framework keeps no global state.
//
// Flutter: material/theme_data.dart ThemeData.localize
func ThemeDataLocalize(baseTheme *ThemeData, localTextGeometry *TextTheme) *ThemeData {
	return baseTheme.CopyWith(ThemeDataCopy{
		PrimaryTextTheme: foundation.Some(localTextGeometry.Merge(baseTheme.PrimaryTextTheme)),
		TextTheme:        foundation.Some(localTextGeometry.Merge(baseTheme.TextTheme)),
	})
}

// EstimateBrightnessForColor determines whether the given color is light or
// dark, with the Material threshold on its relative luminance. It is used to
// choose between light and dark text on a background of that color.
//
// Flutter: material/theme_data.dart ThemeData.estimateBrightnessForColor
func EstimateBrightnessForColor(color ui.Color) ui.Brightness {
	relativeLuminance := color.ComputeLuminance()
	// The WCAG20 spec says to use kThreshold=0.0525, but Material Design appears
	// to bias more towards using light text; 0.15 is close to its palette.
	const kThreshold = 0.15
	if (relativeLuminance+0.05)*(relativeLuminance+0.05) > kThreshold {
		return ui.BrightnessLight
	}
	return ui.BrightnessDark
}

func lerpThemeExtensions(a, b *ThemeData, t float64) []ThemeExtension {
	var out []ThemeExtension
	for _, ea := range a.Extensions {
		var eb ThemeExtension
		for _, x := range b.Extensions {
			if reflect.TypeOf(x) == reflect.TypeOf(ea) {
				eb = x
				break
			}
		}
		out = append(out, ea.Lerp(eb, t))
	}
	for _, eb := range b.Extensions {
		found := false
		for _, ea := range a.Extensions {
			if reflect.TypeOf(ea) == reflect.TypeOf(eb) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, eb)
		}
	}
	return out
}

// themeColorFields lists the plain colors of ThemeData in Flutter's order.
var themeColorFields = [...]struct {
	name string
	get  func(*ThemeData) *ui.Color
}{
	{"canvasColor", func(d *ThemeData) *ui.Color { return &d.CanvasColor }},
	{"cardColor", func(d *ThemeData) *ui.Color { return &d.CardColor }},
	{"disabledColor", func(d *ThemeData) *ui.Color { return &d.DisabledColor }},
	{"dividerColor", func(d *ThemeData) *ui.Color { return &d.DividerColor }},
	{"focusColor", func(d *ThemeData) *ui.Color { return &d.FocusColor }},
	{"highlightColor", func(d *ThemeData) *ui.Color { return &d.HighlightColor }},
	{"hintColor", func(d *ThemeData) *ui.Color { return &d.HintColor }},
	{"hoverColor", func(d *ThemeData) *ui.Color { return &d.HoverColor }},
	{"primaryColor", func(d *ThemeData) *ui.Color { return &d.PrimaryColor }},
	{"primaryColorDark", func(d *ThemeData) *ui.Color { return &d.PrimaryColorDark }},
	{"primaryColorLight", func(d *ThemeData) *ui.Color { return &d.PrimaryColorLight }},
	{"scaffoldBackgroundColor", func(d *ThemeData) *ui.Color { return &d.ScaffoldBackgroundColor }},
	{"secondaryHeaderColor", func(d *ThemeData) *ui.Color { return &d.SecondaryHeaderColor }},
	{"shadowColor", func(d *ThemeData) *ui.Color { return &d.ShadowColor }},
	{"splashColor", func(d *ThemeData) *ui.Color { return &d.SplashColor }},
	{"unselectedWidgetColor", func(d *ThemeData) *ui.Color { return &d.UnselectedWidgetColor }},
	{"dialogBackgroundColor", func(d *ThemeData) *ui.Color { return &d.DialogBackgroundColor }},
	{"indicatorColor", func(d *ThemeData) *ui.Color { return &d.IndicatorColor }},
}

func lerpColor(a, b ui.Color, t float64) ui.Color { return *ui.ColorLerp(&a, &b, t) }

// ThemeDataLerp linearly interpolates between two themes. The fields that
// cannot be interpolated (the booleans, the platform, the tap target size) switch
// at t = 0.5.
//
// Flutter: material/theme_data.dart ThemeData.lerp
func ThemeDataLerp(a, b *ThemeData, t float64) *ThemeData {
	if a == b {
		return a
	}
	pickFirst := t < 0.5
	out := &ThemeData{
		Extensions:       lerpThemeExtensions(a, b, t),
		VisualDensity:    VisualDensityLerp(a.VisualDensity, b.VisualDensity, t),
		ColorScheme:      ColorSchemeLerp(a.ColorScheme, b.ColorScheme, t),
		IconTheme:        widgets.IconThemeDataLerp(&a.IconTheme, &b.IconTheme, t),
		PrimaryIconTheme: widgets.IconThemeDataLerp(&a.PrimaryIconTheme, &b.PrimaryIconTheme, t),
		PrimaryTextTheme: TextThemeLerp(a.PrimaryTextTheme, b.PrimaryTextTheme, t),
		TextTheme:        TextThemeLerp(a.TextTheme, b.TextTheme, t),
		Typography:       TypographyLerp(a.Typography, b.Typography, t),
	}
	src := b
	if pickFirst {
		src = a
	}
	out.ApplyElevationOverlayColor = src.ApplyElevationOverlayColor
	out.MaterialTapTargetSize = src.MaterialTapTargetSize
	out.Platform = src.Platform
	out.UseMaterial3 = src.UseMaterial3
	out.SplashFactory = src.SplashFactory
	out.PageTransitionsTheme = src.PageTransitionsTheme
	for _, f := range themeColorFields {
		*f.get(out) = lerpColor(*f.get(a), *f.get(b), t)
	}
	return out
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/theme_data.dart ThemeData.==
func (d *ThemeData) Equal(other *ThemeData) bool {
	if d == other {
		return true
	}
	if d == nil || other == nil {
		return false
	}
	if d.ApplyElevationOverlayColor != other.ApplyElevationOverlayColor || d.MaterialTapTargetSize != other.MaterialTapTargetSize ||
		d.Platform != other.Platform || d.UseMaterial3 != other.UseMaterial3 || !foundation.ValuesEqual(d.SplashFactory, other.SplashFactory) || d.VisualDensity != other.VisualDensity ||
		!d.ColorScheme.Equal(other.ColorScheme) || !d.IconTheme.Equal(other.IconTheme) || !d.PrimaryIconTheme.Equal(other.PrimaryIconTheme) ||
		!d.PrimaryTextTheme.Equal(other.PrimaryTextTheme) || !d.TextTheme.Equal(other.TextTheme) || !d.Typography.Equal(other.Typography) || !d.PageTransitionsTheme.Equal(other.PageTransitionsTheme) {
		return false
	}
	for _, f := range themeColorFields {
		if *f.get(d) != *f.get(other) {
			return false
		}
	}
	if len(d.Extensions) != len(other.Extensions) {
		return false
	}
	for _, e := range d.Extensions {
		found := false
		for _, x := range other.Extensions {
			if reflect.TypeOf(x) == reflect.TypeOf(e) && foundation.ValuesEqual(x, e) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (d *ThemeData) ToStringShort() string { return "ThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d *ThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme that differ from those
// of the fallback theme, at the debug level.
//
// Flutter: material/theme_data.dart ThemeData.debugFillProperties
func (d *ThemeData) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	def := NewThemeDataFallback()
	dbg := func(def any) foundation.DiagnosticsPropertyOpts {
		return foundation.DiagnosticsPropertyOpts{DefaultValue: def, Level: foundation.DiagnosticLevelDebug}
	}
	properties.Add(foundation.NewIterableProperty("extensions", d.Extensions, foundation.IterablePropertyOpts{Level: foundation.DiagnosticLevelDebug}))
	properties.Add(foundation.NewDiagnosticsProperty("applyElevationOverlayColor", d.ApplyElevationOverlayColor, dbg(def.ApplyElevationOverlayColor)))
	properties.Add(foundation.NewDiagnosticsProperty("materialTapTargetSize", d.MaterialTapTargetSize, dbg(nil)))
	properties.Add(foundation.NewEnumProperty("platform", foundation.Some(d.Platform), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultTargetPlatform(), Level: foundation.DiagnosticLevelDebug}))
	properties.Add(foundation.NewDiagnosticsProperty("useMaterial3", d.UseMaterial3, dbg(def.UseMaterial3)))
	properties.Add(foundation.NewDiagnosticsProperty("splashFactory", d.SplashFactory, dbg(def.SplashFactory)))
	properties.Add(foundation.NewDiagnosticsProperty("visualDensity", d.VisualDensity, dbg(def.VisualDensity)))
	properties.Add(foundation.NewDiagnosticsProperty("colorScheme", d.ColorScheme, dbg(def.ColorScheme)))
	for _, f := range themeColorFields {
		properties.Add(painting.NewColorProperty(f.name, foundation.Some(*f.get(d)), dbg(*f.get(def))))
	}
	properties.Add(foundation.NewDiagnosticsProperty("iconTheme", d.IconTheme, dbg(def.IconTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("primaryIconTheme", d.PrimaryIconTheme, dbg(def.PrimaryIconTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("primaryTextTheme", d.PrimaryTextTheme, dbg(def.PrimaryTextTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("textTheme", d.TextTheme, dbg(def.TextTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("typography", d.Typography, dbg(def.Typography)))
	properties.Add(foundation.NewDiagnosticsProperty("pageTransitionsTheme", d.PageTransitionsTheme, dbg(def.PageTransitionsTheme)))
}
