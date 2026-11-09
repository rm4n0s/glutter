package cupertino

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoDynamicColor is a color that can be another color depending on the
// brightness of the theme (dark mode), the contrast setting of the system (high
// contrast) and the elevation of the interface (a sheet above the base level): the
// colors of iOS's semantic palette. It has eight colors, one for each combination, and
// it is resolved against a context (ResolveFrom, CupertinoDynamicColorResolve) to the
// one that the context asks for. A color that has not been resolved is its color:
// EffectiveColor, the light, normal contrast, base level one, which is what a widget
// that does not resolve draws.
//
// Deviation: Dart's CupertinoDynamicColor implements Color, so it can be given to any
// parameter that takes a Color. Color here is a plain struct, so the parameters of the
// Cupertino widgets take a ui.ColorValue, which this type is (SPEC §12.10); the
// parameters of the other libraries still take a ui.Color, and EffectiveColor or
// CupertinoDynamicColorResolve gives it. The debug label is not part of the equality
// of two colors, as in Flutter, only of the Go value: use Equal to compare as Flutter
// does. A resolved color does not remember the element that resolved it.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor
type CupertinoDynamicColor struct {
	// Color is the color to use when the brightness is light, the contrast normal and the
	// interface level base.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.color
	Color ui.Color
	// DarkColor is the color for dark mode, normal contrast and the base level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.darkColor
	DarkColor ui.Color
	// HighContrastColor is the color for light mode, high contrast and the base level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.highContrastColor
	HighContrastColor ui.Color
	// DarkHighContrastColor is the color for dark mode, high contrast and the base level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.darkHighContrastColor
	DarkHighContrastColor ui.Color
	// ElevatedColor is the color for light mode, normal contrast and the elevated level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.elevatedColor
	ElevatedColor ui.Color
	// DarkElevatedColor is the color for dark mode, normal contrast and the elevated level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.darkElevatedColor
	DarkElevatedColor ui.Color
	// HighContrastElevatedColor is the color for light mode, high contrast and the elevated level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.highContrastElevatedColor
	HighContrastElevatedColor ui.Color
	// DarkHighContrastElevatedColor is the color for dark mode, high contrast and the elevated level.
	//
	// Flutter: cupertino/colors.dart CupertinoDynamicColor.darkHighContrastElevatedColor
	DarkHighContrastElevatedColor ui.Color

	// effective is the color that this value draws as: Color until it is resolved.
	effective  ui.Color
	debugLabel string
}

// CupertinoDynamicColorOpts are the optional parameters of the constructors of
// CupertinoDynamicColor.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.new
type CupertinoDynamicColorOpts struct {
	// DebugLabel is the name that the string form of the color shows, the name of the
	// color in CupertinoColors for the system colors.
	DebugLabel string
}

// NewCupertinoDynamicColor creates a color with a color for every combination of
// brightness, contrast and interface level, in Flutter's declaration order.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.new
func NewCupertinoDynamicColor(color, darkColor, highContrastColor, darkHighContrastColor, elevatedColor, darkElevatedColor,
	highContrastElevatedColor, darkHighContrastElevatedColor ui.Color, opts ...CupertinoDynamicColorOpts) CupertinoDynamicColor {
	var o CupertinoDynamicColorOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return CupertinoDynamicColor{
		Color: color, DarkColor: darkColor, HighContrastColor: highContrastColor, DarkHighContrastColor: darkHighContrastColor,
		ElevatedColor: elevatedColor, DarkElevatedColor: darkElevatedColor, HighContrastElevatedColor: highContrastElevatedColor,
		DarkHighContrastElevatedColor: darkHighContrastElevatedColor, effective: color, debugLabel: o.DebugLabel,
	}
}

// NewCupertinoDynamicColorWithBrightnessAndContrast creates a color that depends on
// the brightness and the contrast but not on the interface level: the elevated colors
// are the base ones.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.withBrightnessAndContrast
func NewCupertinoDynamicColorWithBrightnessAndContrast(color, darkColor, highContrastColor, darkHighContrastColor ui.Color,
	opts ...CupertinoDynamicColorOpts) CupertinoDynamicColor {
	return NewCupertinoDynamicColor(color, darkColor, highContrastColor, darkHighContrastColor, color, darkColor,
		highContrastColor, darkHighContrastColor, opts...)
}

// NewCupertinoDynamicColorWithBrightness creates a color that depends on the
// brightness only: the high contrast and the elevated colors are the normal and the
// base ones.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.withBrightness
func NewCupertinoDynamicColorWithBrightness(color, darkColor ui.Color, opts ...CupertinoDynamicColorOpts) CupertinoDynamicColor {
	return NewCupertinoDynamicColor(color, darkColor, color, darkColor, color, darkColor, color, darkColor, opts...)
}

// EffectiveColor is the color that this value draws as: the one that ResolveFrom picked
// for a resolved color, and Color for one that was not resolved. It makes the type a
// ui.ColorValue.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.toARGB32
func (c CupertinoDynamicColor) EffectiveColor() ui.Color { return c.effective }

// ResolveColor is the color of the context: the effective color of ResolveFrom. It
// makes the type a widgets.ResolvableColor.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.resolveFrom
func (c CupertinoDynamicColor) ResolveColor(bc widgets.BuildContext) ui.Color {
	return c.ResolveFrom(bc).effective
}

// CupertinoDynamicColorResolve is the color that c has in the context: the color that a
// CupertinoDynamicColor picks for the brightness, contrast and interface level of bc,
// and the color itself for any other ui.ColorValue (nil is the zero Color).
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.resolve
func CupertinoDynamicColorResolve(c ui.ColorValue, bc widgets.BuildContext) ui.Color {
	return widgets.ResolveColorValue(c, bc)
}

// CupertinoDynamicColorMaybeResolve is CupertinoDynamicColorResolve for a color that can
// be null: it reports none for nil.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.maybeResolve
func CupertinoDynamicColorMaybeResolve(c ui.ColorValue, bc widgets.BuildContext) foundation.Opt[ui.Color] {
	if c == nil {
		return foundation.None[ui.Color]()
	}
	return foundation.Some(widgets.ResolveColorValue(c, bc))
}

func (c CupertinoDynamicColor) isPlatformBrightnessDependent() bool {
	return c.Color != c.DarkColor || c.ElevatedColor != c.DarkElevatedColor ||
		c.HighContrastColor != c.DarkHighContrastColor || c.HighContrastElevatedColor != c.DarkHighContrastElevatedColor
}

func (c CupertinoDynamicColor) isHighContrastDependent() bool {
	return c.Color != c.HighContrastColor || c.DarkColor != c.DarkHighContrastColor ||
		c.ElevatedColor != c.HighContrastElevatedColor || c.DarkElevatedColor != c.DarkHighContrastElevatedColor
}

func (c CupertinoDynamicColor) isInterfaceElevationDependent() bool {
	return c.Color != c.ElevatedColor || c.DarkColor != c.DarkElevatedColor ||
		c.HighContrastColor != c.HighContrastElevatedColor || c.DarkHighContrastColor != c.DarkHighContrastElevatedColor
}

// ResolveFrom is the color with its effective color set to the one that the context
// asks for: the brightness is the theme's (CupertinoThemeMaybeBrightnessOf, light when
// there is neither a theme nor a MediaQuery), the contrast is the MediaQuery's, and the
// level is the CupertinoUserInterfaceLevel's. A color that does not depend on one of
// them does not look at it, so it does not depend on its inherited widget either.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.resolveFrom
func (c CupertinoDynamicColor) ResolveFrom(bc widgets.BuildContext) CupertinoDynamicColor {
	brightness := ui.BrightnessLight
	if c.isPlatformBrightnessDependent() {
		if b, ok := CupertinoThemeMaybeBrightnessOf(bc); ok {
			brightness = b
		}
	}
	level := CupertinoUserInterfaceLevelBase
	if c.isInterfaceElevationDependent() {
		if l, ok := CupertinoUserInterfaceLevelMaybeOf(bc); ok {
			level = l
		}
	}
	highContrast := false
	if c.isHighContrastDependent() {
		if h, ok := widgets.MediaQueryMaybeHighContrastOf(bc); ok {
			highContrast = h
		}
	}
	dark := brightness == ui.BrightnessDark
	elevated := level == CupertinoUserInterfaceLevelElevated
	var resolved ui.Color
	switch {
	case !dark && !elevated && !highContrast:
		resolved = c.Color
	case !dark && !elevated && highContrast:
		resolved = c.HighContrastColor
	case !dark && elevated && !highContrast:
		resolved = c.ElevatedColor
	case !dark && elevated && highContrast:
		resolved = c.HighContrastElevatedColor
	case dark && !elevated && !highContrast:
		resolved = c.DarkColor
	case dark && !elevated && highContrast:
		resolved = c.DarkHighContrastColor
	case dark && elevated && !highContrast:
		resolved = c.DarkElevatedColor
	default:
		resolved = c.DarkHighContrastElevatedColor
	}
	c.effective = resolved
	return c
}

// Equal is Dart's == on dynamic colors: the effective color and the eight colors are
// the same; the debug label does not count.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.operator ==
func (c CupertinoDynamicColor) Equal(o CupertinoDynamicColor) bool {
	c.debugLabel, o.debugLabel = "", ""
	return c == o
}

// String describes the color: the colors it depends on, each one in the form
// `name = color`, with the effective one between asterisks.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.toString
func (c CupertinoDynamicColor) String() string {
	item := func(name string, color ui.Color) string {
		marker := ""
		if color == c.effective {
			marker = "*"
		}
		return fmt.Sprintf("%s%s = %v%s", marker, name, color, marker)
	}
	bright, contrast, elevation := c.isPlatformBrightnessDependent(), c.isHighContrastDependent(), c.isInterfaceElevationDependent()
	xs := []string{item("color", c.Color)}
	add := func(cond bool, name string, color ui.Color) {
		if cond {
			xs = append(xs, item(name, color))
		}
	}
	add(bright, "darkColor", c.DarkColor)
	add(contrast, "highContrastColor", c.HighContrastColor)
	add(bright && contrast, "darkHighContrastColor", c.DarkHighContrastColor)
	add(elevation, "elevatedColor", c.ElevatedColor)
	add(bright && elevation, "darkElevatedColor", c.DarkElevatedColor)
	add(contrast && elevation, "highContrastElevatedColor", c.HighContrastElevatedColor)
	add(bright && contrast && elevation, "darkHighContrastElevatedColor", c.DarkHighContrastElevatedColor)
	label := c.debugLabel
	if label == "" {
		label = "CupertinoDynamicColor"
	}
	return label + "(" + strings.Join(xs, ", ") + ")"
}

// DebugLabel is the name that String shows.
//
// Flutter: cupertino/colors.dart CupertinoDynamicColor.debugLabel
func (c CupertinoDynamicColor) DebugLabel() string { return c.debugLabel }

// CreateCupertinoColorProperty makes the diagnostics property of a color: the debug
// label of a CupertinoDynamicColor, or the color for any other value.
//
// Flutter: cupertino/colors.dart createCupertinoColorProperty
func CreateCupertinoColorProperty(name string, value ui.ColorValue, opts ...foundation.DiagnosticsPropertyOpts) *foundation.DiagnosticsProperty {
	var o foundation.DiagnosticsPropertyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if d, ok := value.(CupertinoDynamicColor); ok {
		if d.debugLabel != "" {
			o.Description = d.debugLabel
		}
		return foundation.NewDiagnosticsProperty(name, d, o)
	}
	if value == nil {
		return painting.NewColorProperty(name, foundation.None[ui.Color](), o)
	}
	return painting.NewColorProperty(name, foundation.Some(value.EffectiveColor()), o)
}
