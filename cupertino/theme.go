package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoTheme applies a visual theme to Cupertino widgets: the colors and the text
// styles of iOS. It also gives the icons below it the primary color.
//
// Flutter: cupertino/theme.dart CupertinoTheme
type CupertinoTheme struct {
	widgets.StatelessWidget
	data  *CupertinoThemeData
	child widgets.Widget
}

// NewCupertinoTheme creates a theme that applies data to the child.
//
// Flutter: cupertino/theme.dart CupertinoTheme.new
func NewCupertinoTheme(data *CupertinoThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoTheme {
	return &CupertinoTheme{widgets.NewStatelessWidgetBase(firstOf(opts)), data, child}
}

// Data is the theme that the widget applies.
//
// Flutter: cupertino/theme.dart CupertinoTheme.data
func (w *CupertinoTheme) Data() *CupertinoThemeData { return w.data }

// Child is the widget below this widget in the tree.
//
// Flutter: cupertino/theme.dart CupertinoTheme.child
func (w *CupertinoTheme) Child() widgets.Widget { return w.child }

// CupertinoThemeOf is the data of the closest CupertinoTheme above the context, or the
// default theme when there is none, resolved in the context: its dynamic colors are the
// ones for the brightness, the contrast and the level of the context. The context
// depends on the theme.
//
// Flutter: cupertino/theme.dart CupertinoTheme.of
func CupertinoThemeOf(bc widgets.BuildContext) *CupertinoThemeData {
	if inherited, ok := widgets.DependOn[*InheritedCupertinoTheme](bc); ok {
		return inherited.theme.data.ResolveFrom(bc)
	}
	return NewCupertinoThemeData().ResolveFrom(bc)
}

// CupertinoThemeBrightnessOf is the brightness of the closest CupertinoTheme above the
// context when it sets one, and otherwise the platform brightness of the MediaQuery.
//
// Flutter: cupertino/theme.dart CupertinoTheme.brightnessOf
func CupertinoThemeBrightnessOf(bc widgets.BuildContext) ui.Brightness {
	if inherited, ok := widgets.DependOn[*InheritedCupertinoTheme](bc); ok {
		if b := inherited.theme.data.Brightness(); b != ui.BrightnessUnspecified {
			return b
		}
	}
	return widgets.MediaQueryPlatformBrightnessOf(bc)
}

// CupertinoThemeMaybeBrightnessOf is CupertinoThemeBrightnessOf that reports false when
// the theme sets no brightness and there is no MediaQuery above the context.
//
// Flutter: cupertino/theme.dart CupertinoTheme.maybeBrightnessOf
func CupertinoThemeMaybeBrightnessOf(bc widgets.BuildContext) (ui.Brightness, bool) {
	if inherited, ok := widgets.DependOn[*InheritedCupertinoTheme](bc); ok {
		if b := inherited.theme.data.Brightness(); b != ui.BrightnessUnspecified {
			return b, true
		}
	}
	return widgets.MediaQueryMaybePlatformBrightnessOf(bc)
}

// Build wraps the child in the inherited theme and in an icon theme that has the primary
// color.
//
// Flutter: cupertino/theme.dart CupertinoTheme.build
func (w *CupertinoTheme) Build(bc widgets.BuildContext) widgets.Widget {
	return NewInheritedCupertinoTheme(w, widgets.NewIconTheme(
		NewCupertinoIconThemeData(CupertinoIconThemeDataOpts{Color: w.data.PrimaryColor()}), w.child))
}

// DebugFillProperties adds the properties of the data.
//
// Flutter: cupertino/theme.dart CupertinoTheme.debugFillProperties
func (w *CupertinoTheme) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	w.data.DebugFillProperties(p)
}

// InheritedCupertinoTheme is the inherited widget that CupertinoTheme builds: it makes the
// theme available to the widgets below it.
//
// Flutter: cupertino/theme.dart InheritedCupertinoTheme
type InheritedCupertinoTheme struct {
	widgets.InheritedTheme
	theme *CupertinoTheme
}

// NewInheritedCupertinoTheme creates the inherited widget for the theme.
//
// Flutter: cupertino/theme.dart InheritedCupertinoTheme.new
func NewInheritedCupertinoTheme(theme *CupertinoTheme, child widgets.Widget, opts ...widgets.WidgetOpts) *InheritedCupertinoTheme {
	return &InheritedCupertinoTheme{widgets.NewInheritedThemeBase(child, opts...), theme}
}

// Theme is the theme that this widget makes available.
//
// Flutter: cupertino/theme.dart InheritedCupertinoTheme.theme
func (w *InheritedCupertinoTheme) Theme() *CupertinoTheme { return w.theme }

// Wrap makes a CupertinoTheme with the same data around the child.
//
// Flutter: cupertino/theme.dart InheritedCupertinoTheme.wrap
func (w *InheritedCupertinoTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewCupertinoTheme(w.theme.data, child)
}

// UpdateShouldNotify is whether the data of the theme changed.
//
// Flutter: cupertino/theme.dart InheritedCupertinoTheme.updateShouldNotify
func (w *InheritedCupertinoTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.theme.data.Equal(old.(*InheritedCupertinoTheme).theme.data)
}

// NoDefaultCupertinoThemeData is the theme data in which a property that is not set is
// null: a base for the data that a theme builds on, and the data that
// CupertinoThemeData.NoDefault gives back to be merged with others.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData
type NoDefaultCupertinoThemeData struct {
	brightness              ui.Brightness
	primaryColor            ui.ColorValue
	primaryContrastingColor ui.ColorValue
	textTheme               *CupertinoTextThemeData
	barBackgroundColor      ui.ColorValue
	scaffoldBackgroundColor ui.ColorValue
	selectionHandleColor    ui.ColorValue
	applyThemeToAll         foundation.Opt[bool]
}

// NoDefaultCupertinoThemeDataOpts are the optional parameters of
// NewNoDefaultCupertinoThemeData and NoDefaultCupertinoThemeData.CopyWith.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.new
type NoDefaultCupertinoThemeDataOpts struct {
	// Brightness is the brightness of the theme; unspecified follows the platform's.
	Brightness ui.Brightness
	// PrimaryColor is the color of the interactive elements: text, icons, the active tab.
	PrimaryColor ui.ColorValue
	// PrimaryContrastingColor is the color that has contrast against the primary color: the
	// text of a filled button.
	PrimaryContrastingColor ui.ColorValue
	// TextTheme is the text styles of the theme.
	TextTheme *CupertinoTextThemeData
	// BarBackgroundColor is the background of the navigation and tab bars.
	BarBackgroundColor ui.ColorValue
	// ScaffoldBackgroundColor is the background of a page scaffold.
	ScaffoldBackgroundColor ui.ColorValue
	// SelectionHandleColor is the color of the text selection handles.
	SelectionHandleColor ui.ColorValue
	// ApplyThemeToAll is whether the theme applies to all the widgets, whatever their
	// brightness.
	ApplyThemeToAll foundation.Opt[bool]
}

// NewNoDefaultCupertinoThemeData creates theme data in which the properties that are not
// given are null.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.new
func NewNoDefaultCupertinoThemeData(opts ...NoDefaultCupertinoThemeDataOpts) *NoDefaultCupertinoThemeData {
	o := firstOf(opts)
	return &NoDefaultCupertinoThemeData{o.Brightness, o.PrimaryColor, o.PrimaryContrastingColor, o.TextTheme, o.BarBackgroundColor,
		o.ScaffoldBackgroundColor, o.SelectionHandleColor, o.ApplyThemeToAll}
}

// Brightness is the brightness of the theme, or ui.BrightnessUnspecified for null (the
// platform's).
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.brightness
func (d *NoDefaultCupertinoThemeData) Brightness() ui.Brightness { return d.brightness }

// PrimaryColor is the primary color, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.primaryColor
func (d *NoDefaultCupertinoThemeData) PrimaryColor() ui.ColorValue { return d.primaryColor }

// PrimaryContrastingColor is the color that contrasts with the primary color, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.primaryContrastingColor
func (d *NoDefaultCupertinoThemeData) PrimaryContrastingColor() ui.ColorValue {
	return d.primaryContrastingColor
}

// TextTheme is the text theme, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.textTheme
func (d *NoDefaultCupertinoThemeData) TextTheme() *CupertinoTextThemeData { return d.textTheme }

// BarBackgroundColor is the background of the bars, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.barBackgroundColor
func (d *NoDefaultCupertinoThemeData) BarBackgroundColor() ui.ColorValue { return d.barBackgroundColor }

// ScaffoldBackgroundColor is the background of the scaffolds, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.scaffoldBackgroundColor
func (d *NoDefaultCupertinoThemeData) ScaffoldBackgroundColor() ui.ColorValue {
	return d.scaffoldBackgroundColor
}

// SelectionHandleColor is the color of the selection handles, or nil.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.selectionHandleColor
func (d *NoDefaultCupertinoThemeData) SelectionHandleColor() ui.ColorValue {
	return d.selectionHandleColor
}

// ApplyThemeToAll is whether the theme applies to all the widgets; none for null.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.applyThemeToAll
func (d *NoDefaultCupertinoThemeData) ApplyThemeToAll() foundation.Opt[bool] {
	return d.applyThemeToAll
}

// NoDefault is the data itself.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.noDefault
func (d *NoDefaultCupertinoThemeData) NoDefault() *NoDefaultCupertinoThemeData { return d }

// ResolveFrom is the data with the colors that depend on the context resolved in it.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.resolveFrom
func (d *NoDefaultCupertinoThemeData) ResolveFrom(bc widgets.BuildContext) *NoDefaultCupertinoThemeData {
	r := *d
	r.primaryColor = maybeResolveValue(d.primaryColor, bc)
	r.primaryContrastingColor = maybeResolveValue(d.primaryContrastingColor, bc)
	if d.textTheme != nil {
		r.textTheme = d.textTheme.ResolveFrom(bc)
	}
	r.barBackgroundColor = maybeResolveValue(d.barBackgroundColor, bc)
	r.scaffoldBackgroundColor = maybeResolveValue(d.scaffoldBackgroundColor, bc)
	r.selectionHandleColor = maybeResolveValue(d.selectionHandleColor, bc)
	return &r
}

func maybeResolveValue(c ui.ColorValue, bc widgets.BuildContext) ui.ColorValue {
	if c == nil {
		return nil
	}
	return resolveColorValue(c, bc)
}

// CopyWith is a copy of the data in which the given properties are replaced.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.copyWith
func (d *NoDefaultCupertinoThemeData) CopyWith(o NoDefaultCupertinoThemeDataOpts) *NoDefaultCupertinoThemeData {
	r := *d
	if o.Brightness != ui.BrightnessUnspecified {
		r.brightness = o.Brightness
	}
	r.primaryColor = orColor(o.PrimaryColor, d.primaryColor)
	r.primaryContrastingColor = orColor(o.PrimaryContrastingColor, d.primaryContrastingColor)
	if o.TextTheme != nil {
		r.textTheme = o.TextTheme
	}
	r.barBackgroundColor = orColor(o.BarBackgroundColor, d.barBackgroundColor)
	r.scaffoldBackgroundColor = orColor(o.ScaffoldBackgroundColor, d.scaffoldBackgroundColor)
	r.selectionHandleColor = orColor(o.SelectionHandleColor, d.selectionHandleColor)
	if o.ApplyThemeToAll.IsSome() {
		r.applyThemeToAll = o.ApplyThemeToAll
	}
	return &r
}

func orColor(a, b ui.ColorValue) ui.ColorValue {
	if a != nil {
		return a
	}
	return b
}

// Equal is Dart's == on the data. As in Flutter, the selection handle color is not
// compared.
//
// Flutter: cupertino/theme.dart NoDefaultCupertinoThemeData.operator ==
func (d *NoDefaultCupertinoThemeData) Equal(o *NoDefaultCupertinoThemeData) bool {
	if d == o {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	return d.brightness == o.brightness && colorValueEqual(d.primaryColor, o.primaryColor) &&
		colorValueEqual(d.primaryContrastingColor, o.primaryContrastingColor) && d.textTheme.Equal(o.textTheme) &&
		colorValueEqual(d.barBackgroundColor, o.barBackgroundColor) && colorValueEqual(d.scaffoldBackgroundColor, o.scaffoldBackgroundColor) &&
		d.applyThemeToAll == o.applyThemeToAll
}

// cupertinoThemeDefaults are the values that a CupertinoThemeData has for the properties
// that were not given.
//
// Flutter: cupertino/theme.dart _CupertinoThemeDefaults
type cupertinoThemeDefaults struct {
	primaryColor            ui.ColorValue
	primaryContrastingColor ui.ColorValue
	barBackgroundColor      ui.ColorValue
	scaffoldBackgroundColor ui.ColorValue
	selectionHandleColor    ui.ColorValue
	applyThemeToAll         bool
	textThemeDefaults       textThemeDefaultsBuilder
}

// defaultCupertinoTheme is the data of Flutter's _kDefaultTheme.
func defaultCupertinoTheme() cupertinoThemeDefaults {
	return cupertinoThemeDefaults{
		primaryColor:            CupertinoColors.SystemBlue,
		primaryContrastingColor: CupertinoColors.White,
		barBackgroundColor:      NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xF0F9F9F9), ui.NewColor(0xF01D1D1D)),
		scaffoldBackgroundColor: CupertinoColors.SystemBackground,
		selectionHandleColor:    CupertinoColors.SystemBlue,
		applyThemeToAll:         false,
		textThemeDefaults:       textThemeDefaultsBuilder{CupertinoColors.Label, CupertinoColors.InactiveGray},
	}
}

func (d cupertinoThemeDefaults) resolveFrom(bc widgets.BuildContext, resolveTextTheme bool) cupertinoThemeDefaults {
	r := d
	r.primaryColor = resolveColorValue(d.primaryColor, bc)
	r.primaryContrastingColor = resolveColorValue(d.primaryContrastingColor, bc)
	r.barBackgroundColor = resolveColorValue(d.barBackgroundColor, bc)
	r.scaffoldBackgroundColor = resolveColorValue(d.scaffoldBackgroundColor, bc)
	r.selectionHandleColor = resolveColorValue(d.selectionHandleColor, bc)
	if resolveTextTheme {
		r.textThemeDefaults = d.textThemeDefaults.resolveFrom(bc)
	}
	return r
}

// CupertinoThemeData holds the colors and the text styles of a CupertinoTheme. The
// properties that are not given have the defaults of iOS: a system blue primary color, a
// translucent bar background and the label colors of the text styles.
//
// Flutter: cupertino/theme.dart CupertinoThemeData
type CupertinoThemeData struct {
	NoDefaultCupertinoThemeData
	defaults cupertinoThemeDefaults
}

// CupertinoThemeDataOpts are the optional parameters of NewCupertinoThemeData and
// CupertinoThemeData.CopyWith.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.new
type CupertinoThemeDataOpts = NoDefaultCupertinoThemeDataOpts

// NewCupertinoThemeData creates the theme data; the properties that are not given have
// the defaults.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.new
func NewCupertinoThemeData(opts ...CupertinoThemeDataOpts) *CupertinoThemeData {
	return &CupertinoThemeData{*NewNoDefaultCupertinoThemeData(opts...), defaultCupertinoTheme()}
}

// PrimaryColor is the primary color: the one that was given, or the default.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.primaryColor
func (d *CupertinoThemeData) PrimaryColor() ui.ColorValue {
	return orColor(d.primaryColor, d.defaults.primaryColor)
}

// PrimaryContrastingColor is the color that contrasts with the primary color: the one that
// was given, or the default.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.primaryContrastingColor
func (d *CupertinoThemeData) PrimaryContrastingColor() ui.ColorValue {
	return orColor(d.primaryContrastingColor, d.defaults.primaryContrastingColor)
}

// TextTheme is the text theme: the one that was given, or the default one, whose actions
// have the primary color.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.textTheme
func (d *CupertinoThemeData) TextTheme() *CupertinoTextThemeData {
	if d.textTheme != nil {
		return d.textTheme
	}
	return &CupertinoTextThemeData{defaults: d.defaults.textThemeDefaults, primaryColor: d.PrimaryColor()}
}

// BarBackgroundColor is the background of the bars: the one that was given, or the default.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.barBackgroundColor
func (d *CupertinoThemeData) BarBackgroundColor() ui.ColorValue {
	return orColor(d.barBackgroundColor, d.defaults.barBackgroundColor)
}

// ScaffoldBackgroundColor is the background of the scaffolds: the one that was given, or
// the default.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.scaffoldBackgroundColor
func (d *CupertinoThemeData) ScaffoldBackgroundColor() ui.ColorValue {
	return orColor(d.scaffoldBackgroundColor, d.defaults.scaffoldBackgroundColor)
}

// SelectionHandleColor is the color of the selection handles: the one that was given, or
// the default.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.selectionHandleColor
func (d *CupertinoThemeData) SelectionHandleColor() ui.ColorValue {
	return orColor(d.selectionHandleColor, d.defaults.selectionHandleColor)
}

// ApplyThemeToAll is whether the theme applies to all the widgets: the value that was
// given, or false.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.applyThemeToAll
func (d *CupertinoThemeData) ApplyThemeToAll() bool {
	return d.applyThemeToAll.OrElse(d.defaults.applyThemeToAll)
}

// NoDefault is the data with only the properties that were given, the others null.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.noDefault
func (d *CupertinoThemeData) NoDefault() *NoDefaultCupertinoThemeData {
	n := d.NoDefaultCupertinoThemeData
	return &n
}

// ResolveFrom is the theme data with the colors that depend on the context resolved in it
// (its own and its defaults').
//
// Flutter: cupertino/theme.dart CupertinoThemeData.resolveFrom
func (d *CupertinoThemeData) ResolveFrom(bc widgets.BuildContext) *CupertinoThemeData {
	return &CupertinoThemeData{*d.NoDefaultCupertinoThemeData.ResolveFrom(bc), d.defaults.resolveFrom(bc, d.textTheme == nil)}
}

// CopyWith is a copy of the theme data in which the given properties are replaced.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.copyWith
func (d *CupertinoThemeData) CopyWith(o CupertinoThemeDataOpts) *CupertinoThemeData {
	return &CupertinoThemeData{*d.NoDefaultCupertinoThemeData.CopyWith(o), d.defaults}
}

// Equal is Dart's == on the data: the brightness, the colors, the text theme and
// applyThemeToAll are the same.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.operator ==
func (d *CupertinoThemeData) Equal(o *CupertinoThemeData) bool {
	if d == o {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	return d.brightness == o.brightness && colorValueEqual(d.PrimaryColor(), o.PrimaryColor()) &&
		colorValueEqual(d.PrimaryContrastingColor(), o.PrimaryContrastingColor()) && d.TextTheme().Equal(o.TextTheme()) &&
		colorValueEqual(d.BarBackgroundColor(), o.BarBackgroundColor()) && colorValueEqual(d.ScaffoldBackgroundColor(), o.ScaffoldBackgroundColor()) &&
		colorValueEqual(d.SelectionHandleColor(), o.SelectionHandleColor()) && d.ApplyThemeToAll() == o.ApplyThemeToAll()
}

// DebugFillProperties adds the properties that differ from the default theme.
//
// Flutter: cupertino/theme.dart CupertinoThemeData.debugFillProperties
func (d *CupertinoThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	def := NewCupertinoThemeData()
	if d.brightness != ui.BrightnessUnspecified {
		p.Add(foundation.NewDiagnosticsProperty("brightness", d.brightness.String()))
	}
	for _, e := range []struct {
		name     string
		got, def ui.ColorValue
	}{
		{"primaryColor", d.PrimaryColor(), def.PrimaryColor()}, {"primaryContrastingColor", d.PrimaryContrastingColor(), def.PrimaryContrastingColor()},
		{"barBackgroundColor", d.BarBackgroundColor(), def.BarBackgroundColor()},
		{"scaffoldBackgroundColor", d.ScaffoldBackgroundColor(), def.ScaffoldBackgroundColor()},
		{"selectionHandleColor", d.SelectionHandleColor(), def.SelectionHandleColor()},
	} {
		p.Add(CreateCupertinoColorProperty(e.name, e.got, foundation.DiagnosticsPropertyOpts{DefaultValue: e.def}))
	}
	p.Add(foundation.NewDiagnosticsProperty("applyThemeToAll", d.ApplyThemeToAll(), foundation.DiagnosticsPropertyOpts{DefaultValue: def.ApplyThemeToAll()}))
	d.TextTheme().DebugFillProperties(p)
}
