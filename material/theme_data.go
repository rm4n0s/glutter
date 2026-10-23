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
				panic(errors.New(tagVisualDensityOutOfRange, "the density must be from -4 to 4", "density", v))
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
	// CarouselViewTheme is the theme of the CarouselView widgets.
	CarouselViewTheme CarouselViewThemeData
	// DataTableTheme is the theme of the DataTable widgets.
	DataTableTheme DataTableThemeData
	// ProgressIndicatorTheme is the theme of the ProgressIndicator widgets.
	ProgressIndicatorTheme ProgressIndicatorThemeData
	// ScrollbarTheme is the theme of the Scrollbar widgets.
	ScrollbarTheme ScrollbarThemeData
	// ListTileTheme is the theme of the ListTile-related widgets.
	ListTileTheme ListTileThemeData
	// ExpansionTileTheme is the theme of the ExpansionTile-related widgets.
	ExpansionTileTheme ExpansionTileThemeData
	// BadgeTheme is the theme of the Badge-related widgets.
	BadgeTheme BadgeThemeData
	// TooltipTheme is the theme of the Tooltip-related widgets.
	TooltipTheme TooltipThemeData
	// DividerTheme is the theme of the Divider-related widgets.
	DividerTheme DividerThemeData
	// CardTheme is the theme of the Card-related widgets.
	CardTheme CardThemeData
	// TimePickerTheme is the theme of time pickers.
	TimePickerTheme TimePickerThemeData
	// DatePickerTheme is the theme of date pickers.
	DatePickerTheme DatePickerThemeData
	// DialogTheme is the theme of dialogs.
	DialogTheme DialogThemeData
	// SearchViewTheme is the default configuration of the search views.
	SearchViewTheme SearchViewThemeData
	// SearchBarTheme is the default configuration of the search bars.
	SearchBarTheme SearchBarThemeData
	// ChipTheme is the default configuration of the chips.
	ChipTheme ChipThemeData
	// SliderTheme is the theme of the Slider and RangeSlider widgets.
	SliderTheme SliderThemeData
	// SwitchTheme is the theme of the Switch widgets.
	SwitchTheme SwitchThemeData
	// RadioTheme is the theme of the Radio widgets.
	RadioTheme RadioThemeData
	// CheckboxTheme is the theme of the Checkbox widgets.
	CheckboxTheme CheckboxThemeData
	// DropdownMenuTheme defines the text style, input decoration theme, menu style and disabled color of the dropdown menus.
	DropdownMenuTheme DropdownMenuThemeData
	// TextSelectionTheme defines the appearance of the cursor and the selection of the text fields and the selectable text.
	TextSelectionTheme TextSelectionThemeData
	// InputDecorationTheme defines the default values of the properties of the decorations of the text fields.
	InputDecorationTheme InputDecorationThemeData
	// PopupMenuTheme defines the colors, shape, text styles and position of the popup menus.
	PopupMenuTheme PopupMenuThemeData
	// MenuButtonTheme defines the style of the buttons of the menus.
	MenuButtonTheme MenuButtonThemeData
	// MenuBarTheme defines the style of the menu bars.
	MenuBarTheme MenuBarThemeData
	// MenuTheme defines the style and the submenu icon of the menus.
	MenuTheme MenuThemeData
	// ActionIconTheme is the theme of the icons of the BackButtons, CloseButtons and drawer buttons.
	ActionIconTheme ActionIconThemeData
	// SegmentedButtonTheme is the theme of the SegmentedButtons.
	SegmentedButtonTheme SegmentedButtonThemeData
	// FloatingActionButtonTheme is the theme of the FloatingActionButtons.
	FloatingActionButtonTheme FloatingActionButtonThemeData
	// IconButtonTheme is the theme of the IconButtons.
	IconButtonTheme IconButtonThemeData
	// OutlinedButtonTheme is the theme of the OutlinedButtons.
	OutlinedButtonTheme OutlinedButtonThemeData
	// FilledButtonTheme is the theme of the FilledButtons.
	FilledButtonTheme FilledButtonThemeData
	// ElevatedButtonTheme is the theme of the ElevatedButtons.
	ElevatedButtonTheme ElevatedButtonThemeData
	// TextButtonTheme is the theme of the TextButtons.
	TextButtonTheme TextButtonThemeData
	// TabBarTheme is the default configuration of TabBar.
	TabBarTheme TabBarThemeData
	// NavigationDrawerTheme is the default configuration of NavigationDrawer.
	NavigationDrawerTheme NavigationDrawerThemeData
	// NavigationRailTheme is the default configuration of NavigationRail.
	NavigationRailTheme NavigationRailThemeData
	// NavigationBarTheme is the default configuration of NavigationBar.
	NavigationBarTheme NavigationBarThemeData
	// BottomSheetTheme is the default properties of BottomSheet widgets.
	BottomSheetTheme BottomSheetThemeData
	// DrawerTheme is the default properties of Drawer widgets.
	DrawerTheme DrawerThemeData
	// SnackBarTheme is the default configuration of SnackBar widgets.
	SnackBarTheme SnackBarThemeData
	// BottomAppBarTheme is the default configuration of BottomAppBar widgets.
	BottomAppBarTheme BottomAppBarThemeData
	// BannerTheme is the default configuration of MaterialBanner widgets.
	BannerTheme MaterialBannerThemeData
	// AppBarTheme is the default configuration of AppBar widgets (Material 3 and 2).
	AppBarTheme AppBarThemeData

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
	// localized memoizes ThemeOf's localized copies, one per script category. It
	// is not copied by CopyWith.
	localized [3]*ThemeData
	// iconCopies memoizes copyWithIconTheme. It is not copied by CopyWith.
	iconCopies *iconCopyCache

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
	// CarouselViewTheme is the theme of the CarouselView widgets.
	CarouselViewTheme CarouselViewThemeData
	// DataTableTheme is the theme of the DataTable widgets.
	DataTableTheme DataTableThemeData
	// ProgressIndicatorTheme is the theme of the ProgressIndicator widgets.
	ProgressIndicatorTheme ProgressIndicatorThemeData
	// ScrollbarTheme is the theme of the Scrollbar widgets.
	ScrollbarTheme ScrollbarThemeData
	// ListTileTheme is the theme of the ListTile-related widgets.
	ListTileTheme ListTileThemeData
	// ExpansionTileTheme is the theme of the ExpansionTile-related widgets.
	ExpansionTileTheme ExpansionTileThemeData
	// BadgeTheme is the theme of the Badge-related widgets.
	BadgeTheme BadgeThemeData
	// TooltipTheme is the theme of the Tooltip-related widgets.
	TooltipTheme TooltipThemeData
	// DividerTheme is the theme of the Divider-related widgets.
	DividerTheme DividerThemeData
	// CardTheme is the theme of the Card-related widgets.
	CardTheme CardThemeData
	// TimePickerTheme is the theme of time pickers.
	TimePickerTheme TimePickerThemeData
	// DatePickerTheme is the theme of date pickers.
	DatePickerTheme DatePickerThemeData
	// DialogTheme is the theme of dialogs.
	DialogTheme DialogThemeData
	// SearchViewTheme is the default configuration of the search views.
	SearchViewTheme SearchViewThemeData
	// SearchBarTheme is the default configuration of the search bars.
	SearchBarTheme SearchBarThemeData
	// ChipTheme is the default configuration of the chips.
	ChipTheme ChipThemeData
	// SliderTheme is the theme of the Slider and RangeSlider widgets.
	SliderTheme SliderThemeData
	// SwitchTheme is the theme of the Switch widgets.
	SwitchTheme SwitchThemeData
	// RadioTheme is the theme of the Radio widgets.
	RadioTheme RadioThemeData
	// CheckboxTheme is the theme of the Checkbox widgets.
	CheckboxTheme CheckboxThemeData
	// DropdownMenuTheme defines the text style, input decoration theme, menu style and disabled color of the dropdown menus.
	DropdownMenuTheme DropdownMenuThemeData
	// TextSelectionTheme defines the appearance of the cursor and the selection of the text fields and the selectable text.
	TextSelectionTheme TextSelectionThemeData
	// InputDecorationTheme defines the default values of the properties of the decorations of the text fields.
	InputDecorationTheme InputDecorationThemeData
	// PopupMenuTheme defines the colors, shape, text styles and position of the popup menus.
	PopupMenuTheme PopupMenuThemeData
	// MenuButtonTheme defines the style of the buttons of the menus.
	MenuButtonTheme MenuButtonThemeData
	// MenuBarTheme defines the style of the menu bars.
	MenuBarTheme MenuBarThemeData
	// MenuTheme defines the style and the submenu icon of the menus.
	MenuTheme MenuThemeData
	// ActionIconTheme is the theme of the icons of the BackButtons, CloseButtons and drawer buttons.
	ActionIconTheme ActionIconThemeData
	// SegmentedButtonTheme is the theme of the SegmentedButtons.
	SegmentedButtonTheme SegmentedButtonThemeData
	// FloatingActionButtonTheme is the theme of the FloatingActionButtons.
	FloatingActionButtonTheme FloatingActionButtonThemeData
	// IconButtonTheme is the theme of the IconButtons.
	IconButtonTheme IconButtonThemeData
	// OutlinedButtonTheme is the theme of the OutlinedButtons.
	OutlinedButtonTheme OutlinedButtonThemeData
	// FilledButtonTheme is the theme of the FilledButtons.
	FilledButtonTheme FilledButtonThemeData
	// ElevatedButtonTheme is the theme of the ElevatedButtons.
	ElevatedButtonTheme ElevatedButtonThemeData
	// TextButtonTheme is the theme of the TextButtons.
	TextButtonTheme TextButtonThemeData
	// TabBarTheme is the default configuration of TabBar.
	TabBarTheme TabBarThemeData
	// NavigationDrawerTheme is the default configuration of NavigationDrawer.
	NavigationDrawerTheme NavigationDrawerThemeData
	// NavigationRailTheme is the default configuration of NavigationRail.
	NavigationRailTheme NavigationRailThemeData
	// NavigationBarTheme is the default configuration of NavigationBar.
	NavigationBarTheme NavigationBarThemeData
	// BottomSheetTheme is the default properties of BottomSheet widgets.
	BottomSheetTheme BottomSheetThemeData
	// DrawerTheme is the default properties of Drawer widgets.
	DrawerTheme DrawerThemeData
	// SnackBarTheme is the default configuration of SnackBar widgets.
	SnackBarTheme SnackBarThemeData
	// BottomAppBarTheme is the default configuration of BottomAppBar widgets.
	BottomAppBarTheme BottomAppBarThemeData
	// BannerTheme is the default configuration of MaterialBanner widgets.
	BannerTheme MaterialBannerThemeData
	// AppBarTheme is the default configuration of AppBar widgets (Material 3 and 2).
	AppBarTheme AppBarThemeData

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
			panic(errors.New(tagThemeDataBrightnessMismatch, "ThemeData.brightness does not match ColorScheme.brightness",
				"brightness", o.Brightness.String(), "colorScheme", colorScheme.Brightness().String()))
		}
		if o.ColorSchemeSeed.IsSome() && (colorScheme != nil || o.PrimarySwatch != nil || o.PrimaryColor.IsSome()) {
			panic(errors.New(tagThemeDataSeedConflict, "colorSchemeSeed cannot be given with colorScheme, primarySwatch or primaryColor"))
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

		PageTransitionsTheme:      pageTransitionsTheme,
		CarouselViewTheme:         o.CarouselViewTheme,
		DataTableTheme:            o.DataTableTheme,
		ProgressIndicatorTheme:    o.ProgressIndicatorTheme,
		ScrollbarTheme:            o.ScrollbarTheme,
		ListTileTheme:             o.ListTileTheme,
		ExpansionTileTheme:        o.ExpansionTileTheme,
		BadgeTheme:                o.BadgeTheme,
		TooltipTheme:              o.TooltipTheme,
		DividerTheme:              o.DividerTheme,
		CardTheme:                 o.CardTheme,
		TimePickerTheme:           o.TimePickerTheme,
		DatePickerTheme:           o.DatePickerTheme,
		DialogTheme:               o.DialogTheme,
		SearchViewTheme:           o.SearchViewTheme,
		SearchBarTheme:            o.SearchBarTheme,
		ChipTheme:                 o.ChipTheme,
		SliderTheme:               o.SliderTheme,
		SwitchTheme:               o.SwitchTheme,
		RadioTheme:                o.RadioTheme,
		CheckboxTheme:             o.CheckboxTheme,
		DropdownMenuTheme:         o.DropdownMenuTheme,
		TextSelectionTheme:        o.TextSelectionTheme,
		InputDecorationTheme:      o.InputDecorationTheme,
		PopupMenuTheme:            o.PopupMenuTheme,
		MenuButtonTheme:           o.MenuButtonTheme,
		MenuBarTheme:              o.MenuBarTheme,
		MenuTheme:                 o.MenuTheme,
		ActionIconTheme:           o.ActionIconTheme,
		SegmentedButtonTheme:      o.SegmentedButtonTheme,
		FloatingActionButtonTheme: o.FloatingActionButtonTheme,
		IconButtonTheme:           o.IconButtonTheme,
		OutlinedButtonTheme:       o.OutlinedButtonTheme,
		FilledButtonTheme:         o.FilledButtonTheme,
		ElevatedButtonTheme:       o.ElevatedButtonTheme,
		TextButtonTheme:           o.TextButtonTheme,
		TabBarTheme:               o.TabBarTheme,
		NavigationDrawerTheme:     o.NavigationDrawerTheme,
		NavigationRailTheme:       o.NavigationRailTheme,
		NavigationBarTheme:        o.NavigationBarTheme,
		BottomSheetTheme:          o.BottomSheetTheme,
		DrawerTheme:               o.DrawerTheme,
		SnackBarTheme:             o.SnackBarTheme,
		BottomAppBarTheme:         o.BottomAppBarTheme,
		BannerTheme:               o.BannerTheme,
		AppBarTheme:               o.AppBarTheme,

		DialogBackgroundColor: dialogBackgroundColor.Value(),
		IndicatorColor:        indicatorColor.Value(),
	}
}

// iconCopyCache holds the last sixteen copies that copyWithIconTheme made of a theme.
type iconCopyCache struct {
	entries [16]struct {
		icon  widgets.IconThemeData
		theme *ThemeData
	}
	next int
}

// copyWithIconTheme is CopyWith(ThemeDataCopy{IconTheme: Some(icon)}) that gives the same theme again
// for an equal icon theme: the last sixteen are kept in the theme it was made from (a screen has a handful of kinds of button, each with its own icon theme). Every button wraps its
// ink in an AnimatedTheme of such a copy, and a new copy in each build is a new theme to compare field by
// field with the last one (SPEC §12.10); the same copy is one comparison of pointers.
//
// Deviation: Flutter makes a new ThemeData in every build (material/button_style_button.dart).
func (d *ThemeData) copyWithIconTheme(icon widgets.IconThemeData) *ThemeData {
	if d.iconCopies == nil {
		d.iconCopies = &iconCopyCache{}
	}
	c := d.iconCopies
	for i := range c.entries {
		if e := &c.entries[i]; e.theme != nil && e.icon.Equal(icon) {
			return e.theme
		}
	}
	t := d.CopyWith(ThemeDataCopy{IconTheme: foundation.Some(icon)})
	c.entries[c.next].icon, c.entries[c.next].theme = icon, t
	c.next = (c.next + 1) % len(c.entries)
	return t
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
	Brightness                ui.Brightness
	DisabledColor             foundation.Opt[ui.Color]
	DividerColor              foundation.Opt[ui.Color]
	FocusColor                foundation.Opt[ui.Color]
	HighlightColor            foundation.Opt[ui.Color]
	HintColor                 foundation.Opt[ui.Color]
	HoverColor                foundation.Opt[ui.Color]
	PrimaryColor              foundation.Opt[ui.Color]
	PrimaryColorDark          foundation.Opt[ui.Color]
	PrimaryColorLight         foundation.Opt[ui.Color]
	ScaffoldBackgroundColor   foundation.Opt[ui.Color]
	SecondaryHeaderColor      foundation.Opt[ui.Color]
	ShadowColor               foundation.Opt[ui.Color]
	SplashColor               foundation.Opt[ui.Color]
	SplashFactory             foundation.Opt[InteractiveInkFeatureFactory]
	UnselectedWidgetColor     foundation.Opt[ui.Color]
	IconTheme                 foundation.Opt[widgets.IconThemeData]
	PrimaryIconTheme          foundation.Opt[widgets.IconThemeData]
	PrimaryTextTheme          foundation.Opt[*TextTheme]
	TextTheme                 foundation.Opt[*TextTheme]
	Typography                foundation.Opt[*Typography]
	PageTransitionsTheme      foundation.Opt[*PageTransitionsTheme]
	CarouselViewTheme         foundation.Opt[CarouselViewThemeData]
	DataTableTheme            foundation.Opt[DataTableThemeData]
	ProgressIndicatorTheme    foundation.Opt[ProgressIndicatorThemeData]
	ScrollbarTheme            foundation.Opt[ScrollbarThemeData]
	ListTileTheme             foundation.Opt[ListTileThemeData]
	ExpansionTileTheme        foundation.Opt[ExpansionTileThemeData]
	BadgeTheme                foundation.Opt[BadgeThemeData]
	TooltipTheme              foundation.Opt[TooltipThemeData]
	DividerTheme              foundation.Opt[DividerThemeData]
	CardTheme                 foundation.Opt[CardThemeData]
	TimePickerTheme           foundation.Opt[TimePickerThemeData]
	DatePickerTheme           foundation.Opt[DatePickerThemeData]
	DialogTheme               foundation.Opt[DialogThemeData]
	SearchViewTheme           foundation.Opt[SearchViewThemeData]
	SearchBarTheme            foundation.Opt[SearchBarThemeData]
	ChipTheme                 foundation.Opt[ChipThemeData]
	SliderTheme               foundation.Opt[SliderThemeData]
	SwitchTheme               foundation.Opt[SwitchThemeData]
	RadioTheme                foundation.Opt[RadioThemeData]
	CheckboxTheme             foundation.Opt[CheckboxThemeData]
	DropdownMenuTheme         foundation.Opt[DropdownMenuThemeData]
	TextSelectionTheme        foundation.Opt[TextSelectionThemeData]
	InputDecorationTheme      foundation.Opt[InputDecorationThemeData]
	PopupMenuTheme            foundation.Opt[PopupMenuThemeData]
	MenuButtonTheme           foundation.Opt[MenuButtonThemeData]
	MenuBarTheme              foundation.Opt[MenuBarThemeData]
	MenuTheme                 foundation.Opt[MenuThemeData]
	ActionIconTheme           foundation.Opt[ActionIconThemeData]
	SegmentedButtonTheme      foundation.Opt[SegmentedButtonThemeData]
	FloatingActionButtonTheme foundation.Opt[FloatingActionButtonThemeData]
	IconButtonTheme           foundation.Opt[IconButtonThemeData]
	OutlinedButtonTheme       foundation.Opt[OutlinedButtonThemeData]
	FilledButtonTheme         foundation.Opt[FilledButtonThemeData]
	ElevatedButtonTheme       foundation.Opt[ElevatedButtonThemeData]
	TextButtonTheme           foundation.Opt[TextButtonThemeData]
	TabBarTheme               foundation.Opt[TabBarThemeData]
	NavigationDrawerTheme     foundation.Opt[NavigationDrawerThemeData]
	NavigationRailTheme       foundation.Opt[NavigationRailThemeData]
	NavigationBarTheme        foundation.Opt[NavigationBarThemeData]
	BottomSheetTheme          foundation.Opt[BottomSheetThemeData]
	DrawerTheme               foundation.Opt[DrawerThemeData]
	SnackBarTheme             foundation.Opt[SnackBarThemeData]
	BottomAppBarTheme         foundation.Opt[BottomAppBarThemeData]
	BannerTheme               foundation.Opt[MaterialBannerThemeData]
	AppBarTheme               foundation.Opt[AppBarThemeData]
	DialogBackgroundColor     foundation.Opt[ui.Color]
	IndicatorColor            foundation.Opt[ui.Color]
}

// CopyWith creates a copy of this theme but with the fields of c that are set
// replaced. A Brightness replaces only the brightness of the color scheme; the
// other colors stay as they are.
//
// Flutter: material/theme_data.dart ThemeData.copyWith
func (d *ThemeData) CopyWith(c ThemeDataCopy) *ThemeData {
	out := *d
	out.localized = [3]*ThemeData{}
	out.iconCopies = nil
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
	out.CarouselViewTheme = c.CarouselViewTheme.OrElse(d.CarouselViewTheme)
	out.DataTableTheme = c.DataTableTheme.OrElse(d.DataTableTheme)
	out.ProgressIndicatorTheme = c.ProgressIndicatorTheme.OrElse(d.ProgressIndicatorTheme)
	out.ScrollbarTheme = c.ScrollbarTheme.OrElse(d.ScrollbarTheme)
	out.ListTileTheme = c.ListTileTheme.OrElse(d.ListTileTheme)
	out.ExpansionTileTheme = c.ExpansionTileTheme.OrElse(d.ExpansionTileTheme)
	out.BadgeTheme = c.BadgeTheme.OrElse(d.BadgeTheme)
	out.TooltipTheme = c.TooltipTheme.OrElse(d.TooltipTheme)
	out.DividerTheme = c.DividerTheme.OrElse(d.DividerTheme)
	out.CardTheme = c.CardTheme.OrElse(d.CardTheme)
	out.TimePickerTheme = c.TimePickerTheme.OrElse(d.TimePickerTheme)
	out.DatePickerTheme = c.DatePickerTheme.OrElse(d.DatePickerTheme)
	out.DialogTheme = c.DialogTheme.OrElse(d.DialogTheme)
	out.SearchViewTheme = c.SearchViewTheme.OrElse(d.SearchViewTheme)
	out.SearchBarTheme = c.SearchBarTheme.OrElse(d.SearchBarTheme)
	out.ChipTheme = c.ChipTheme.OrElse(d.ChipTheme)
	out.SliderTheme = c.SliderTheme.OrElse(d.SliderTheme)
	out.SwitchTheme = c.SwitchTheme.OrElse(d.SwitchTheme)
	out.RadioTheme = c.RadioTheme.OrElse(d.RadioTheme)
	out.CheckboxTheme = c.CheckboxTheme.OrElse(d.CheckboxTheme)
	out.DropdownMenuTheme = c.DropdownMenuTheme.OrElse(d.DropdownMenuTheme)
	out.TextSelectionTheme = c.TextSelectionTheme.OrElse(d.TextSelectionTheme)
	out.InputDecorationTheme = c.InputDecorationTheme.OrElse(d.InputDecorationTheme)
	out.PopupMenuTheme = c.PopupMenuTheme.OrElse(d.PopupMenuTheme)
	out.MenuButtonTheme = c.MenuButtonTheme.OrElse(d.MenuButtonTheme)
	out.MenuBarTheme = c.MenuBarTheme.OrElse(d.MenuBarTheme)
	out.MenuTheme = c.MenuTheme.OrElse(d.MenuTheme)
	out.ActionIconTheme = c.ActionIconTheme.OrElse(d.ActionIconTheme)
	out.SegmentedButtonTheme = c.SegmentedButtonTheme.OrElse(d.SegmentedButtonTheme)
	out.FloatingActionButtonTheme = c.FloatingActionButtonTheme.OrElse(d.FloatingActionButtonTheme)
	out.IconButtonTheme = c.IconButtonTheme.OrElse(d.IconButtonTheme)
	out.OutlinedButtonTheme = c.OutlinedButtonTheme.OrElse(d.OutlinedButtonTheme)
	out.FilledButtonTheme = c.FilledButtonTheme.OrElse(d.FilledButtonTheme)
	out.ElevatedButtonTheme = c.ElevatedButtonTheme.OrElse(d.ElevatedButtonTheme)
	out.TextButtonTheme = c.TextButtonTheme.OrElse(d.TextButtonTheme)
	out.TabBarTheme = c.TabBarTheme.OrElse(d.TabBarTheme)
	out.NavigationDrawerTheme = c.NavigationDrawerTheme.OrElse(d.NavigationDrawerTheme)
	out.NavigationRailTheme = c.NavigationRailTheme.OrElse(d.NavigationRailTheme)
	out.NavigationBarTheme = c.NavigationBarTheme.OrElse(d.NavigationBarTheme)
	out.BottomSheetTheme = c.BottomSheetTheme.OrElse(d.BottomSheetTheme)
	out.DrawerTheme = c.DrawerTheme.OrElse(d.DrawerTheme)
	out.SnackBarTheme = c.SnackBarTheme.OrElse(d.SnackBarTheme)
	out.BottomAppBarTheme = c.BottomAppBarTheme.OrElse(d.BottomAppBarTheme)
	out.BannerTheme = c.BannerTheme.OrElse(d.BannerTheme)
	out.AppBarTheme = c.AppBarTheme.OrElse(d.AppBarTheme)
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
		Extensions:                lerpThemeExtensions(a, b, t),
		VisualDensity:             VisualDensityLerp(a.VisualDensity, b.VisualDensity, t),
		ColorScheme:               ColorSchemeLerp(a.ColorScheme, b.ColorScheme, t),
		IconTheme:                 widgets.IconThemeDataLerp(&a.IconTheme, &b.IconTheme, t),
		PrimaryIconTheme:          widgets.IconThemeDataLerp(&a.PrimaryIconTheme, &b.PrimaryIconTheme, t),
		PrimaryTextTheme:          TextThemeLerp(a.PrimaryTextTheme, b.PrimaryTextTheme, t),
		TextTheme:                 TextThemeLerp(a.TextTheme, b.TextTheme, t),
		Typography:                TypographyLerp(a.Typography, b.Typography, t),
		CarouselViewTheme:         CarouselViewThemeDataLerp(a.CarouselViewTheme, b.CarouselViewTheme, t),
		DataTableTheme:            DataTableThemeDataLerp(a.DataTableTheme, b.DataTableTheme, t),
		ProgressIndicatorTheme:    ProgressIndicatorThemeDataLerp(a.ProgressIndicatorTheme, b.ProgressIndicatorTheme, t),
		ScrollbarTheme:            ScrollbarThemeDataLerp(a.ScrollbarTheme, b.ScrollbarTheme, t),
		ListTileTheme:             ListTileThemeDataLerp(a.ListTileTheme, b.ListTileTheme, t),
		ExpansionTileTheme:        ExpansionTileThemeDataLerp(a.ExpansionTileTheme, b.ExpansionTileTheme, t),
		BadgeTheme:                BadgeThemeDataLerp(a.BadgeTheme, b.BadgeTheme, t),
		TooltipTheme:              TooltipThemeDataLerp(a.TooltipTheme, b.TooltipTheme, t),
		DividerTheme:              DividerThemeDataLerp(a.DividerTheme, b.DividerTheme, t),
		CardTheme:                 CardThemeDataLerp(a.CardTheme, b.CardTheme, t),
		TimePickerTheme:           TimePickerThemeDataLerp(a.TimePickerTheme, b.TimePickerTheme, t),
		DatePickerTheme:           DatePickerThemeDataLerp(a.DatePickerTheme, b.DatePickerTheme, t),
		DialogTheme:               DialogThemeDataLerp(a.DialogTheme, b.DialogTheme, t),
		SearchViewTheme:           SearchViewThemeDataLerp(a.SearchViewTheme, b.SearchViewTheme, t),
		SearchBarTheme:            SearchBarThemeDataLerp(a.SearchBarTheme, b.SearchBarTheme, t),
		ChipTheme:                 ChipThemeDataLerp(a.ChipTheme, b.ChipTheme, t),
		SliderTheme:               SliderThemeDataLerp(a.SliderTheme, b.SliderTheme, t),
		SwitchTheme:               SwitchThemeDataLerp(a.SwitchTheme, b.SwitchTheme, t),
		RadioTheme:                RadioThemeDataLerp(a.RadioTheme, b.RadioTheme, t),
		CheckboxTheme:             CheckboxThemeDataLerp(a.CheckboxTheme, b.CheckboxTheme, t),
		DropdownMenuTheme:         DropdownMenuThemeDataLerp(a.DropdownMenuTheme, b.DropdownMenuTheme, t),
		TextSelectionTheme:        TextSelectionThemeDataLerp(a.TextSelectionTheme, b.TextSelectionTheme, t),
		InputDecorationTheme:      InputDecorationThemeDataLerp(a.InputDecorationTheme, b.InputDecorationTheme, t),
		PopupMenuTheme:            PopupMenuThemeDataLerp(a.PopupMenuTheme, b.PopupMenuTheme, t),
		MenuButtonTheme:           MenuButtonThemeDataLerp(a.MenuButtonTheme, b.MenuButtonTheme, t),
		MenuBarTheme:              MenuBarThemeDataLerp(a.MenuBarTheme, b.MenuBarTheme, t),
		MenuTheme:                 MenuThemeDataLerp(a.MenuTheme, b.MenuTheme, t),
		ActionIconTheme:           ActionIconThemeDataLerp(a.ActionIconTheme, b.ActionIconTheme, t),
		SegmentedButtonTheme:      SegmentedButtonThemeDataLerp(a.SegmentedButtonTheme, b.SegmentedButtonTheme, t),
		FloatingActionButtonTheme: FloatingActionButtonThemeDataLerp(a.FloatingActionButtonTheme, b.FloatingActionButtonTheme, t),
		IconButtonTheme:           IconButtonThemeDataLerp(a.IconButtonTheme, b.IconButtonTheme, t),
		OutlinedButtonTheme:       OutlinedButtonThemeDataLerp(a.OutlinedButtonTheme, b.OutlinedButtonTheme, t),
		FilledButtonTheme:         FilledButtonThemeDataLerp(a.FilledButtonTheme, b.FilledButtonTheme, t),
		ElevatedButtonTheme:       ElevatedButtonThemeDataLerp(a.ElevatedButtonTheme, b.ElevatedButtonTheme, t),
		TextButtonTheme:           TextButtonThemeDataLerp(a.TextButtonTheme, b.TextButtonTheme, t),
		TabBarTheme:               TabBarThemeDataLerp(a.TabBarTheme, b.TabBarTheme, t),
		NavigationDrawerTheme:     NavigationDrawerThemeDataLerp(a.NavigationDrawerTheme, b.NavigationDrawerTheme, t),
		NavigationRailTheme:       NavigationRailThemeDataLerp(a.NavigationRailTheme, b.NavigationRailTheme, t),
		NavigationBarTheme:        NavigationBarThemeDataLerp(a.NavigationBarTheme, b.NavigationBarTheme, t),
		BottomSheetTheme:          BottomSheetThemeDataLerp(a.BottomSheetTheme, b.BottomSheetTheme, t),
		DrawerTheme:               DrawerThemeDataLerp(a.DrawerTheme, b.DrawerTheme, t),
		SnackBarTheme:             SnackBarThemeDataLerp(a.SnackBarTheme, b.SnackBarTheme, t),
		BottomAppBarTheme:         BottomAppBarThemeDataLerp(a.BottomAppBarTheme, b.BottomAppBarTheme, t),
		BannerTheme:               MaterialBannerThemeDataLerp(a.BannerTheme, b.BannerTheme, t),
		AppBarTheme:               AppBarThemeDataLerp(a.AppBarTheme, b.AppBarTheme, t),
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
		!d.PrimaryTextTheme.Equal(other.PrimaryTextTheme) || !d.TextTheme.Equal(other.TextTheme) || !d.Typography.Equal(other.Typography) || !d.PageTransitionsTheme.Equal(other.PageTransitionsTheme) || !d.AppBarTheme.Equal(other.AppBarTheme) || !d.BannerTheme.Equal(other.BannerTheme) || !d.BottomAppBarTheme.Equal(other.BottomAppBarTheme) || !d.SnackBarTheme.Equal(other.SnackBarTheme) || !d.DrawerTheme.Equal(other.DrawerTheme) || !d.BottomSheetTheme.Equal(other.BottomSheetTheme) || !d.NavigationBarTheme.Equal(other.NavigationBarTheme) || !d.NavigationRailTheme.Equal(other.NavigationRailTheme) || !d.NavigationDrawerTheme.Equal(other.NavigationDrawerTheme) || !d.TabBarTheme.Equal(other.TabBarTheme) || !d.TextButtonTheme.Equal(other.TextButtonTheme) || !d.ElevatedButtonTheme.Equal(other.ElevatedButtonTheme) || !d.FilledButtonTheme.Equal(other.FilledButtonTheme) || !d.OutlinedButtonTheme.Equal(other.OutlinedButtonTheme) || !d.IconButtonTheme.Equal(other.IconButtonTheme) || !d.FloatingActionButtonTheme.Equal(other.FloatingActionButtonTheme) || !d.SegmentedButtonTheme.Equal(other.SegmentedButtonTheme) || !d.ActionIconTheme.Equal(other.ActionIconTheme) || !d.MenuTheme.Equal(other.MenuTheme) || !d.MenuBarTheme.Equal(other.MenuBarTheme) || !d.MenuButtonTheme.Equal(other.MenuButtonTheme) || !d.PopupMenuTheme.Equal(other.PopupMenuTheme) || !d.InputDecorationTheme.Equal(other.InputDecorationTheme) || !d.TextSelectionTheme.Equal(other.TextSelectionTheme) || !d.DropdownMenuTheme.Equal(other.DropdownMenuTheme) || !d.CheckboxTheme.Equal(other.CheckboxTheme) || !d.RadioTheme.Equal(other.RadioTheme) || !d.SwitchTheme.Equal(other.SwitchTheme) || !d.SliderTheme.Equal(other.SliderTheme) || !d.ChipTheme.Equal(other.ChipTheme) || !d.SearchBarTheme.Equal(other.SearchBarTheme) || !d.SearchViewTheme.Equal(other.SearchViewTheme) || !d.DialogTheme.Equal(other.DialogTheme) || !d.DatePickerTheme.Equal(other.DatePickerTheme) || !d.TimePickerTheme.Equal(other.TimePickerTheme) || !d.CardTheme.Equal(other.CardTheme) || !d.DividerTheme.Equal(other.DividerTheme) || !d.TooltipTheme.Equal(other.TooltipTheme) || !d.BadgeTheme.Equal(other.BadgeTheme) || !d.ExpansionTileTheme.Equal(other.ExpansionTileTheme) || !d.ListTileTheme.Equal(other.ListTileTheme) || !d.ScrollbarTheme.Equal(other.ScrollbarTheme) || !d.ProgressIndicatorTheme.Equal(other.ProgressIndicatorTheme) || !d.DataTableTheme.Equal(other.DataTableTheme) || !d.CarouselViewTheme.Equal(other.CarouselViewTheme) {
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
	properties.Add(foundation.NewDiagnosticsProperty("carouselViewTheme", d.CarouselViewTheme, dbg(def.CarouselViewTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("dataTableTheme", d.DataTableTheme, dbg(def.DataTableTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("progressIndicatorTheme", d.ProgressIndicatorTheme, dbg(def.ProgressIndicatorTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("scrollbarTheme", d.ScrollbarTheme, dbg(def.ScrollbarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("listTileTheme", d.ListTileTheme, dbg(def.ListTileTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("expansionTileTheme", d.ExpansionTileTheme, dbg(def.ExpansionTileTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("badgeTheme", d.BadgeTheme, dbg(def.BadgeTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("tooltipTheme", d.TooltipTheme, dbg(def.TooltipTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("dividerTheme", d.DividerTheme, dbg(def.DividerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("cardTheme", d.CardTheme, dbg(def.CardTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("timePickerTheme", d.TimePickerTheme, dbg(def.TimePickerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("datePickerTheme", d.DatePickerTheme, dbg(def.DatePickerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("dialogTheme", d.DialogTheme, dbg(def.DialogTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("searchViewTheme", d.SearchViewTheme, dbg(def.SearchViewTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("searchBarTheme", d.SearchBarTheme, dbg(def.SearchBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("chipTheme", d.ChipTheme, dbg(def.ChipTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("sliderTheme", d.SliderTheme, dbg(def.SliderTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("switchTheme", d.SwitchTheme, dbg(def.SwitchTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("radioTheme", d.RadioTheme, dbg(def.RadioTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("checkboxTheme", d.CheckboxTheme, dbg(def.CheckboxTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("dropdownMenuTheme", d.DropdownMenuTheme, dbg(def.DropdownMenuTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("textSelectionTheme", d.TextSelectionTheme, dbg(def.TextSelectionTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("inputDecorationTheme", d.InputDecorationTheme, dbg(def.InputDecorationTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("popupMenuTheme", d.PopupMenuTheme, dbg(def.PopupMenuTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("menuButtonTheme", d.MenuButtonTheme, dbg(def.MenuButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("menuBarTheme", d.MenuBarTheme, dbg(def.MenuBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("menuTheme", d.MenuTheme, dbg(def.MenuTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("actionIconTheme", d.ActionIconTheme, dbg(def.ActionIconTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("segmentedButtonTheme", d.SegmentedButtonTheme, dbg(def.SegmentedButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("floatingActionButtonTheme", d.FloatingActionButtonTheme, dbg(def.FloatingActionButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("iconButtonTheme", d.IconButtonTheme, dbg(def.IconButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("outlinedButtonTheme", d.OutlinedButtonTheme, dbg(def.OutlinedButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("filledButtonTheme", d.FilledButtonTheme, dbg(def.FilledButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("elevatedButtonTheme", d.ElevatedButtonTheme, dbg(def.ElevatedButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("textButtonTheme", d.TextButtonTheme, dbg(def.TextButtonTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("tabBarTheme", d.TabBarTheme, dbg(def.TabBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("navigationDrawerTheme", d.NavigationDrawerTheme, dbg(def.NavigationDrawerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("navigationRailTheme", d.NavigationRailTheme, dbg(def.NavigationRailTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("navigationBarTheme", d.NavigationBarTheme, dbg(def.NavigationBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("bottomSheetTheme", d.BottomSheetTheme, dbg(def.BottomSheetTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("drawerTheme", d.DrawerTheme, dbg(def.DrawerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("snackBarTheme", d.SnackBarTheme, dbg(def.SnackBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("bottomAppBarTheme", d.BottomAppBarTheme, dbg(def.BottomAppBarTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("bannerTheme", d.BannerTheme, dbg(def.BannerTheme)))
	properties.Add(foundation.NewDiagnosticsProperty("appBarTheme", d.AppBarTheme, dbg(def.AppBarTheme)))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagThemeDataBrightnessMismatch = "ThemeDataBrightnessMismatch"
	tagThemeDataSeedConflict       = "ThemeDataSeedConflict"
	tagVisualDensityOutOfRange     = "VisualDensityOutOfRange"
)
