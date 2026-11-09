package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The families of the Cupertino text styles. The engine maps them to San Francisco Text
// and Display on Apple platforms (SPEC §31.2); elsewhere they fall back to the default
// family, as in Flutter.
const (
	// CupertinoSystemTextFamily is the family of the text styles for body sizes: San Francisco Text.
	//
	// Flutter: cupertino/text_theme.dart _kDefaultTextStyle
	CupertinoSystemTextFamily = "CupertinoSystemText"
	// CupertinoSystemDisplayFamily is the family of the text styles for large sizes: San Francisco Display.
	//
	// Flutter: cupertino/text_theme.dart _kDefaultLargeTitleTextStyle
	CupertinoSystemDisplayFamily = "CupertinoSystemDisplay"
)

// cupertinoTextStyle makes one of the constant text styles of text_theme.dart.
func cupertinoTextStyle(family string, size float64, weight ui.FontWeight, letterSpacing float64, color ui.ColorValue, decoration foundation.Opt[ui.TextDecoration]) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		Inherit: foundation.Some(false), FontFamily: family, FontSize: foundation.Some(size), FontWeight: weight,
		LetterSpacing: foundation.Some(letterSpacing), Color: foundation.Some(color.EffectiveColor()), Decoration: decoration,
	})
}

var cupertinoNoDecoration = foundation.Some(ui.TextDecorationNone)

func defaultTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemTextFamily, 17, 0, -0.41, CupertinoColors.Label, cupertinoNoDecoration)
}

func defaultActionTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemTextFamily, 17, 0, -0.41, CupertinoColors.ActiveBlue, cupertinoNoDecoration)
}

func defaultActionSmallTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemTextFamily, 15, 0, -0.23, CupertinoColors.ActiveBlue, cupertinoNoDecoration)
}

func defaultTabLabelTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemTextFamily, 10, ui.FontWeightW500, -0.24, CupertinoColors.InactiveGray, foundation.None[ui.TextDecoration]())
}

func defaultMiddleTitleTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemTextFamily, 17, ui.FontWeightW600, -0.41, CupertinoColors.Label, foundation.None[ui.TextDecoration]())
}

func defaultLargeTitleTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemDisplayFamily, 34, ui.FontWeightW700, 0.38, CupertinoColors.Label, foundation.None[ui.TextDecoration]())
}

func defaultPickerTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemDisplayFamily, 21, ui.FontWeightW400, -0.6, CupertinoColors.Label, foundation.None[ui.TextDecoration]())
}

func defaultDateTimePickerTextStyle() *painting.TextStyle {
	return cupertinoTextStyle(CupertinoSystemDisplayFamily, 21, ui.FontWeightW400, 0.4, CupertinoColors.Label, foundation.None[ui.TextDecoration]())
}

// CupertinoTextThemeData holds the text styles of the Cupertino widgets, as a
// CupertinoThemeData has them. A style that is not given is its default, in the label
// color (and, for the actions, in the primary color).
//
// Deviation: a TextStyle here holds a ui.Color, not a color that depends on the context, so a
// style that is given keeps the color it has, and the colors that follow the brightness
// are the defaults' (the label color, the inactive gray and the primary color, which
// ResolveFrom resolves). Flutter's resolveFrom also resolves the dynamic colors of a style
// that was given (SPEC §12.10).
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData
type CupertinoTextThemeData struct {
	defaults             textThemeDefaultsBuilder
	primaryColor         ui.ColorValue
	textStyle            *painting.TextStyle
	actionTextStyle      *painting.TextStyle
	actionSmallTextStyle *painting.TextStyle
	tabLabelTextStyle    *painting.TextStyle
	navTitleTextStyle    *painting.TextStyle
	navLargeTitleStyle   *painting.TextStyle
	navActionTextStyle   *painting.TextStyle
	pickerTextStyle      *painting.TextStyle
	dateTimePickerStyle  *painting.TextStyle
}

// CupertinoTextThemeDataOpts are the optional parameters of NewCupertinoTextThemeData and
// CupertinoTextThemeData.CopyWith.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.new
type CupertinoTextThemeDataOpts struct {
	// PrimaryColor is the color of the actions: the default of the action text styles. The
	// default is CupertinoColors.SystemBlue.
	PrimaryColor ui.ColorValue
	// TextStyle is the default style of the text of the widgets.
	TextStyle *painting.TextStyle
	// ActionTextStyle is the style of the interactive text, such as the label of a button.
	ActionTextStyle *painting.TextStyle
	// ActionSmallTextStyle is the style of the small interactive text, in a button of the
	// small size.
	ActionSmallTextStyle *painting.TextStyle
	// TabLabelTextStyle is the style of the labels of a tab bar.
	TabLabelTextStyle *painting.TextStyle
	// NavTitleTextStyle is the style of the title of a navigation bar.
	NavTitleTextStyle *painting.TextStyle
	// NavLargeTitleTextStyle is the style of the large title of a navigation bar.
	NavLargeTitleTextStyle *painting.TextStyle
	// NavActionTextStyle is the style of the actions of a navigation bar.
	NavActionTextStyle *painting.TextStyle
	// PickerTextStyle is the style of the items of a picker.
	PickerTextStyle *painting.TextStyle
	// DateTimePickerTextStyle is the style of the items of a date and time picker.
	DateTimePickerTextStyle *painting.TextStyle
}

// NewCupertinoTextThemeData creates the text styles of a theme: the styles that are not
// given are the defaults.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.new
func NewCupertinoTextThemeData(opts ...CupertinoTextThemeDataOpts) *CupertinoTextThemeData {
	var o CupertinoTextThemeDataOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	primary := o.PrimaryColor
	if primary == nil {
		primary = CupertinoColors.SystemBlue
	}
	return &CupertinoTextThemeData{
		defaults: textThemeDefaultsBuilder{CupertinoColors.Label, CupertinoColors.InactiveGray}, primaryColor: primary,
		textStyle: o.TextStyle, actionTextStyle: o.ActionTextStyle, actionSmallTextStyle: o.ActionSmallTextStyle,
		tabLabelTextStyle: o.TabLabelTextStyle, navTitleTextStyle: o.NavTitleTextStyle, navLargeTitleStyle: o.NavLargeTitleTextStyle,
		navActionTextStyle: o.NavActionTextStyle, pickerTextStyle: o.PickerTextStyle, dateTimePickerStyle: o.DateTimePickerTextStyle,
	}
}

func orStyle(a, b *painting.TextStyle) *painting.TextStyle {
	if a != nil {
		return a
	}
	return b
}

// TextStyle is the default text style of the widgets.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.textStyle
func (t *CupertinoTextThemeData) TextStyle() *painting.TextStyle {
	return orStyle(t.textStyle, t.defaults.textStyle())
}

// ActionTextStyle is the style of the interactive text.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.actionTextStyle
func (t *CupertinoTextThemeData) ActionTextStyle() *painting.TextStyle {
	return orStyle(t.actionTextStyle, t.defaults.actionTextStyle(t.primaryColor))
}

// ActionSmallTextStyle is the style of the small interactive text.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.actionSmallTextStyle
func (t *CupertinoTextThemeData) ActionSmallTextStyle() *painting.TextStyle {
	return orStyle(t.actionSmallTextStyle, t.defaults.actionSmallTextStyle(t.primaryColor))
}

// TabLabelTextStyle is the style of the labels of a tab bar.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.tabLabelTextStyle
func (t *CupertinoTextThemeData) TabLabelTextStyle() *painting.TextStyle {
	return orStyle(t.tabLabelTextStyle, t.defaults.tabLabelTextStyle())
}

// NavTitleTextStyle is the style of the title of a navigation bar.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.navTitleTextStyle
func (t *CupertinoTextThemeData) NavTitleTextStyle() *painting.TextStyle {
	return orStyle(t.navTitleTextStyle, t.defaults.navTitleTextStyle())
}

// NavLargeTitleTextStyle is the style of the large title of a navigation bar.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.navLargeTitleTextStyle
func (t *CupertinoTextThemeData) NavLargeTitleTextStyle() *painting.TextStyle {
	return orStyle(t.navLargeTitleStyle, t.defaults.navLargeTitleTextStyle())
}

// NavActionTextStyle is the style of the actions of a navigation bar.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.navActionTextStyle
func (t *CupertinoTextThemeData) NavActionTextStyle() *painting.TextStyle {
	return orStyle(t.navActionTextStyle, t.defaults.navActionTextStyle(t.primaryColor))
}

// PickerTextStyle is the style of the items of a picker.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.pickerTextStyle
func (t *CupertinoTextThemeData) PickerTextStyle() *painting.TextStyle {
	return orStyle(t.pickerTextStyle, t.defaults.pickerTextStyle())
}

// DateTimePickerTextStyle is the style of the items of a date and time picker.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.dateTimePickerTextStyle
func (t *CupertinoTextThemeData) DateTimePickerTextStyle() *painting.TextStyle {
	return orStyle(t.dateTimePickerStyle, t.defaults.dateTimePickerTextStyle())
}

// ResolveFrom is the text theme with the colors that depend on the context (the label
// color, the inactive gray and the primary color) resolved in it.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.resolveFrom
func (t *CupertinoTextThemeData) ResolveFrom(bc widgets.BuildContext) *CupertinoTextThemeData {
	r := *t
	r.defaults = t.defaults.resolveFrom(bc)
	if t.primaryColor != nil {
		r.primaryColor = resolveColorValue(t.primaryColor, bc)
	}
	return &r
}

// CopyWith is a copy of the text theme in which the given properties are replaced.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.copyWith
func (t *CupertinoTextThemeData) CopyWith(o CupertinoTextThemeDataOpts) *CupertinoTextThemeData {
	r := *t
	if o.PrimaryColor != nil {
		r.primaryColor = o.PrimaryColor
	}
	r.textStyle = orStyle(o.TextStyle, t.textStyle)
	r.actionTextStyle = orStyle(o.ActionTextStyle, t.actionTextStyle)
	r.actionSmallTextStyle = orStyle(o.ActionSmallTextStyle, t.actionSmallTextStyle)
	r.tabLabelTextStyle = orStyle(o.TabLabelTextStyle, t.tabLabelTextStyle)
	r.navTitleTextStyle = orStyle(o.NavTitleTextStyle, t.navTitleTextStyle)
	r.navLargeTitleStyle = orStyle(o.NavLargeTitleTextStyle, t.navLargeTitleStyle)
	r.navActionTextStyle = orStyle(o.NavActionTextStyle, t.navActionTextStyle)
	r.pickerTextStyle = orStyle(o.PickerTextStyle, t.pickerTextStyle)
	r.dateTimePickerStyle = orStyle(o.DateTimePickerTextStyle, t.dateTimePickerStyle)
	return &r
}

// Equal is Dart's == on text themes.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.operator ==
func (t *CupertinoTextThemeData) Equal(o *CupertinoTextThemeData) bool {
	if t == o {
		return true
	}
	if t == nil || o == nil {
		return false
	}
	return t.defaults.equal(o.defaults) && colorValueEqual(t.primaryColor, o.primaryColor) &&
		t.textStyle.Equal(o.textStyle) && t.actionTextStyle.Equal(o.actionTextStyle) && t.actionSmallTextStyle.Equal(o.actionSmallTextStyle) &&
		t.tabLabelTextStyle.Equal(o.tabLabelTextStyle) && t.navTitleTextStyle.Equal(o.navTitleTextStyle) &&
		t.navLargeTitleStyle.Equal(o.navLargeTitleStyle) && t.navActionTextStyle.Equal(o.navActionTextStyle) &&
		t.pickerTextStyle.Equal(o.pickerTextStyle) && t.dateTimePickerStyle.Equal(o.dateTimePickerStyle)
}

// DebugFillProperties adds the nine styles, those that are the defaults at a lower level.
//
// Flutter: cupertino/text_theme.dart CupertinoTextThemeData.debugFillProperties
func (t *CupertinoTextThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	def := NewCupertinoTextThemeData()
	for _, e := range []struct {
		name     string
		got, def *painting.TextStyle
	}{
		{"textStyle", t.TextStyle(), def.TextStyle()}, {"actionTextStyle", t.ActionTextStyle(), def.ActionTextStyle()},
		{"actionSmallTextStyle", t.ActionSmallTextStyle(), def.ActionSmallTextStyle()},
		{"tabLabelTextStyle", t.TabLabelTextStyle(), def.TabLabelTextStyle()}, {"navTitleTextStyle", t.NavTitleTextStyle(), def.NavTitleTextStyle()},
		{"navLargeTitleTextStyle", t.NavLargeTitleTextStyle(), def.NavLargeTitleTextStyle()},
		{"navActionTextStyle", t.NavActionTextStyle(), def.NavActionTextStyle()}, {"pickerTextStyle", t.PickerTextStyle(), def.PickerTextStyle()},
		{"dateTimePickerTextStyle", t.DateTimePickerTextStyle(), def.DateTimePickerTextStyle()},
	} {
		// Dart compares the styles by value, and the properties compare pointers: the default is the style itself when they are equal.
		def := e.def
		if e.got.Equal(e.def) {
			def = e.got
		}
		p.Add(foundation.NewDiagnosticsProperty(e.name, e.got, foundation.DiagnosticsPropertyOpts{DefaultValue: def}))
	}
}

// textThemeDefaultsBuilder makes the default styles: the ones of text_theme.dart in the
// label color and the inactive gray of the theme.
//
// Flutter: cupertino/text_theme.dart _TextThemeDefaultsBuilder
type textThemeDefaultsBuilder struct {
	labelColor, inactiveGrayColor ui.ColorValue
}

func applyLabelColor(original *painting.TextStyle, color ui.ColorValue) *painting.TextStyle {
	if c, ok := original.Color.Get(); ok && c == color.EffectiveColor() {
		return original
	}
	return original.CopyWith(painting.TextStyleCopy{Color: foundation.Some(color.EffectiveColor())})
}

func (b textThemeDefaultsBuilder) textStyle() *painting.TextStyle {
	return applyLabelColor(defaultTextStyle(), b.labelColor)
}

func (b textThemeDefaultsBuilder) tabLabelTextStyle() *painting.TextStyle {
	return applyLabelColor(defaultTabLabelTextStyle(), b.inactiveGrayColor)
}

func (b textThemeDefaultsBuilder) navTitleTextStyle() *painting.TextStyle {
	return applyLabelColor(defaultMiddleTitleTextStyle(), b.labelColor)
}

func (b textThemeDefaultsBuilder) navLargeTitleTextStyle() *painting.TextStyle {
	return applyLabelColor(defaultLargeTitleTextStyle(), b.labelColor)
}

func (b textThemeDefaultsBuilder) pickerTextStyle() *painting.TextStyle {
	return applyLabelColor(defaultPickerTextStyle(), b.labelColor)
}

func (b textThemeDefaultsBuilder) dateTimePickerTextStyle() *painting.TextStyle {
	return applyLabelColor(defaultDateTimePickerTextStyle(), b.labelColor)
}

// withPrimary is the style in the primary color, which is the style's own color when
// there is none (Dart's copyWith(color: null) keeps the color).
func withPrimary(style *painting.TextStyle, primary ui.ColorValue) *painting.TextStyle {
	if primary == nil {
		return style
	}
	return style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(primary.EffectiveColor())})
}

func (b textThemeDefaultsBuilder) actionTextStyle(primary ui.ColorValue) *painting.TextStyle {
	return withPrimary(defaultActionTextStyle(), primary)
}

func (b textThemeDefaultsBuilder) actionSmallTextStyle(primary ui.ColorValue) *painting.TextStyle {
	return withPrimary(defaultActionSmallTextStyle(), primary)
}

func (b textThemeDefaultsBuilder) navActionTextStyle(primary ui.ColorValue) *painting.TextStyle {
	return b.actionTextStyle(primary)
}

func (b textThemeDefaultsBuilder) resolveFrom(bc widgets.BuildContext) textThemeDefaultsBuilder {
	return textThemeDefaultsBuilder{resolveColorValue(b.labelColor, bc), resolveColorValue(b.inactiveGrayColor, bc)}
}

func (b textThemeDefaultsBuilder) equal(o textThemeDefaultsBuilder) bool {
	return colorValueEqual(b.labelColor, o.labelColor) && colorValueEqual(b.inactiveGrayColor, o.inactiveGrayColor)
}

// resolveColorValue resolves a dynamic color in the context and leaves it dynamic, as
// Dart's CupertinoDynamicColor.resolve does: the result can be resolved again in
// another context (a sheet above the base level). Any other value is returned as it is.
func resolveColorValue(c ui.ColorValue, bc widgets.BuildContext) ui.ColorValue {
	if d, ok := c.(CupertinoDynamicColor); ok {
		return d.ResolveFrom(bc)
	}
	return c
}

// colorValueEqual is Dart's == on two nullable colors: dynamic colors are equal by their
// colors and not by their debug labels.
func colorValueEqual(a, b ui.ColorValue) bool {
	da, aok := a.(CupertinoDynamicColor)
	db, bok := b.(CupertinoDynamicColor)
	if aok && bok {
		return da.Equal(db)
	}
	if aok != bok {
		return false
	}
	return a == b
}
