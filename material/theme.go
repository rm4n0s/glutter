package material

import (
	"github.com/rm4n0s/glutter/cupertino"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ThemeAnimationDuration is the default duration of the animation of
// AnimatedTheme.
//
// Flutter: material/theme.dart kThemeAnimationDuration
const ThemeAnimationDuration = 200 * time.Millisecond

// Theme applies a theme to descendant widgets. A theme describes the colors and
// typographic choices of an application.
//
// Like Flutter's, a Theme also provides a CupertinoTheme to its subtree: the one of the closest CupertinoTheme above
// it, or one that is derived from the Material data (MaterialBasedCupertinoThemeData).
//
// Flutter: material/theme.dart Theme
type Theme struct {
	widgets.StatelessWidget
	data  *ThemeData
	child widgets.Widget
}

// NewTheme creates a theme that applies data to the child.
//
// Flutter: material/theme.dart Theme.new
func NewTheme(data *ThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *Theme {
	return &Theme{widgets.NewStatelessWidgetBase(firstWidgetOpts(opts)), data, child}
}

func firstWidgetOpts(o []widgets.WidgetOpts) widgets.WidgetOpts { return firstOpts(o) }

// Data is the specification of the color and typography to use in this
// widget's subtree.
//
// Flutter: material/theme.dart Theme.data
func (w *Theme) Data() *ThemeData { return w.data }

// Child is the widget below this widget in the tree.
//
// Flutter: material/theme.dart Theme.child
func (w *Theme) Child() widgets.Widget { return w.child }

// ThemeOf returns the data from the closest Theme ancestor of bc, with the
// text geometry of the script of the context's locale merged in. If there is no
// Theme ancestor, it returns the fallback theme. The context depends on the
// theme.
//
// Flutter: material/theme.dart Theme.of
func ThemeOf(bc widgets.BuildContext) *ThemeData {
	var theme *ThemeData
	if inherited, ok := widgets.DependOn[*inheritedMaterialTheme](bc); ok {
		theme = inherited.theme.data
	} else if cupertinoTheme, ok := widgets.DependOn[*cupertino.InheritedCupertinoTheme](bc); ok {
		theme = cupertinoBasedMaterialTheme(cupertinoTheme.Theme().Data().ResolveFrom(bc), bc)
	} else {
		theme = fallbackTheme()
	}
	category := ScriptCategoryEnglishLike
	if localizations, ok := MaterialLocalizationsMaybeOf(bc); ok {
		category = localizations.ScriptCategory()
	}
	return theme.localizedFor(category)
}

// localizedFor is ThemeDataLocalize with the geometry of the category, memoized
// in the theme.
//
// Deviation: Flutter caches the last five results of ThemeData.localize in a
// static; the cache here lives in the ThemeData it was made from, so the package
// keeps no mutable global state (SPEC §12.10). The fields of a ThemeData must not
// change after the first ThemeOf, as in Flutter, where it is immutable.
func (d *ThemeData) localizedFor(category ScriptCategory) *ThemeData {
	if category < 0 || int(category) >= len(d.localized) {
		return ThemeDataLocalize(d, d.Typography.GeometryThemeFor(category))
	}
	if d.localized[category] == nil {
		d.localized[category] = ThemeDataLocalize(d, d.Typography.GeometryThemeFor(category))
	}
	return d.localized[category]
}

// fallbackTheme is the theme that ThemeOf returns when there is no Theme.
// Flutter holds one in a static; here every call makes one, as the theme is
// cheap to build and the package keeps no mutable state.
func fallbackTheme() *ThemeData { return NewThemeDataFallback() }

// ColorSchemeOf returns the color scheme of the closest Theme ancestor of bc: the
// colorScheme of ThemeOf.
//
// Flutter: material/color_scheme.dart ColorScheme.of
func ColorSchemeOf(bc widgets.BuildContext) *ColorScheme { return ThemeOf(bc).ColorScheme }

// ThemeBrightnessOf returns the brightness of the closest Theme ancestor of bc,
// or the platform brightness when there is none.
//
// Flutter: material/theme.dart Theme.brightnessOf
func ThemeBrightnessOf(bc widgets.BuildContext) ui.Brightness {
	if inherited, ok := widgets.DependOn[*inheritedMaterialTheme](bc); ok {
		return inherited.theme.data.Brightness()
	}
	return widgets.MediaQueryPlatformBrightnessOf(bc)
}

// ThemeMaybeBrightnessOf is like ThemeBrightnessOf but reports false when
// there is neither a Theme nor a MediaQuery ancestor.
//
// Flutter: material/theme.dart Theme.maybeBrightnessOf
func ThemeMaybeBrightnessOf(bc widgets.BuildContext) (ui.Brightness, bool) {
	if inherited, ok := widgets.DependOn[*inheritedMaterialTheme](bc); ok {
		return inherited.theme.data.Brightness(), true
	}
	return widgets.MediaQueryMaybePlatformBrightnessOf(bc)
}

// Build wraps the child in the inherited theme and the icon theme of the data.
//
// Flutter: material/theme.dart Theme.build
func (w *Theme) Build(bc widgets.BuildContext) widgets.Widget {
	// The inherited themes of the widgets library cannot infer their values from the
	// Theme of the material library: the child is wrapped in these inherited themes
	// to override their values directly.
	selectionStyle := widgets.DefaultSelectionStyleOf(bc)
	selection := w.data.TextSelectionTheme
	child := widgets.NewIconTheme(w.data.IconTheme, widgets.NewDefaultSelectionStyle(w.child, widgets.DefaultSelectionStyleOpts{
		SelectionColor: firstOpt(selection.SelectionColor, selectionStyle.SelectionColor()),
		CursorColor:    firstOpt(selection.CursorColor, selectionStyle.CursorColor()),
	}))
	return newInheritedMaterialTheme(w, cupertino.NewCupertinoTheme(w.inheritedCupertinoThemeData(bc), child))
}

// inheritedCupertinoThemeData is the CupertinoTheme data below the Theme: the one above it when there is one, and
// otherwise the one derived from the Material data, resolved in the context.
//
// Flutter: material/theme.dart Theme._inheritedCupertinoThemeData
func (w *Theme) inheritedCupertinoThemeData(bc widgets.BuildContext) *cupertino.CupertinoThemeData {
	if inherited, ok := widgets.DependOn[*cupertino.InheritedCupertinoTheme](bc); ok {
		return inherited.Theme().Data().ResolveFrom(bc)
	}
	return MaterialBasedCupertinoThemeData(w.data).ResolveFrom(bc)
}

// MaterialBasedCupertinoThemeData is the CupertinoThemeData that a Theme gives the Cupertino widgets below it when
// no CupertinoTheme is above it: its brightness, primary color, primary contrasting color, scaffold background and
// selection handle color come from the Material theme unless its CupertinoOverrideTheme sets them.
//
// Deviation: Flutter's is a subclass of CupertinoThemeData that defers those getters to the Material theme; this is a
// CupertinoThemeData whose properties are set to the values that the getters return (SPEC §12.10).
//
// Flutter: material/theme_data.dart MaterialBasedCupertinoThemeData
func MaterialBasedCupertinoThemeData(materialTheme *ThemeData) *cupertino.CupertinoThemeData {
	override := materialTheme.CupertinoOverrideTheme
	if override == nil {
		override = cupertino.NewNoDefaultCupertinoThemeData()
	}
	brightness := override.Brightness()
	if brightness == ui.BrightnessUnspecified {
		brightness = materialTheme.Brightness()
	}
	pick := func(override ui.ColorValue, material ui.Color) ui.ColorValue {
		if override != nil {
			return override
		}
		return material
	}
	handle := override.SelectionHandleColor()
	if handle == nil {
		if c, ok := materialTheme.TextSelectionTheme.SelectionHandleColor.Get(); ok {
			handle = c
		}
	}
	return cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{
		Brightness:              brightness,
		PrimaryColor:            pick(override.PrimaryColor(), materialTheme.ColorScheme.Primary()),
		PrimaryContrastingColor: pick(override.PrimaryContrastingColor(), materialTheme.ColorScheme.OnPrimary()),
		TextTheme:               override.TextTheme(),
		BarBackgroundColor:      override.BarBackgroundColor(),
		ScaffoldBackgroundColor: pick(override.ScaffoldBackgroundColor(), materialTheme.ScaffoldBackgroundColor),
		SelectionHandleColor:    handle,
		ApplyThemeToAll:         override.ApplyThemeToAll(),
	})
}

// cupertinoBasedMaterialTheme is the Material theme with a color scheme that is made from the colors of a
// CupertinoThemeData: what ThemeOf gives when there is a CupertinoTheme and no Theme.
//
// Flutter: material/theme_data.dart CupertinoBasedMaterialThemeData
func cupertinoBasedMaterialTheme(data *cupertino.CupertinoThemeData, bc widgets.BuildContext) *ThemeData {
	brightness := data.Brightness()
	if brightness == ui.BrightnessUnspecified {
		brightness = ui.BrightnessLight
	}
	primary := cupertino.CupertinoDynamicColorResolve(data.PrimaryColor(), bc)
	return NewThemeData(ThemeDataOpts{ColorScheme: ColorSchemeFromSeed(primary, ColorSchemeFromSeedOpts{
		Brightness: brightness,
		Overrides: ColorSchemeVariantOpts{Primary: foundation.Some(primary),
			OnPrimary: foundation.Some(cupertino.CupertinoDynamicColorResolve(data.PrimaryContrastingColor(), bc))}})})
}

// DebugFillProperties adds the data.
//
// Flutter: material/theme.dart Theme.debugFillProperties
func (w *Theme) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("data", w.data, foundation.DiagnosticsPropertyOpts{ShowName: foundation.Some(false)}))
}

// inheritedMaterialTheme is Flutter's private _InheritedTheme.
//
// Flutter: material/theme.dart _InheritedTheme
type inheritedMaterialTheme struct {
	widgets.InheritedTheme
	theme *Theme
}

func newInheritedMaterialTheme(theme *Theme, child widgets.Widget) *inheritedMaterialTheme {
	return &inheritedMaterialTheme{widgets.NewInheritedThemeBase(child), theme}
}

// Wrap makes a Theme with the same data around the child.
func (w *inheritedMaterialTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTheme(w.theme.data, child)
}

// UpdateShouldNotify is whether the data changed.
func (w *inheritedMaterialTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.theme.data.Equal(old.(*inheritedMaterialTheme).theme.data)
}

// NewThemeDataTween creates a tween between two themes.
//
// Flutter: material/theme.dart ThemeDataTween
func NewThemeDataTween(begin, end *ThemeData) *animation.Tween[*ThemeData] {
	return animation.NewTweenFunc(begin, end, ThemeDataLerp).SetName("ThemeDataTween")
}

// AnimatedThemeOpts are the optional parameters of NewAnimatedTheme.
//
// Flutter: material/theme.dart AnimatedTheme.new
type AnimatedThemeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// Duration is the duration of the animation; the default is
	// ThemeAnimationDuration, and a zero duration makes the change at once.
	Duration foundation.Opt[time.Duration]
	// OnEnd is called every time an animation completes.
	OnEnd func()
}

// AnimatedTheme animates the theme data when it changes: the animated version
// of Theme.
//
// Flutter: material/theme.dart AnimatedTheme
type AnimatedTheme struct {
	widgets.ImplicitlyAnimatedWidget
	data  *ThemeData
	child widgets.Widget
}

// NewAnimatedTheme creates the widget.
//
// Flutter: material/theme.dart AnimatedTheme.new
func NewAnimatedTheme(data *ThemeData, child widgets.Widget, opts ...AnimatedThemeOpts) *AnimatedTheme {
	o := firstOpts(opts)
	return &AnimatedTheme{widgets.NewImplicitlyAnimatedWidgetBase(o.Duration.OrElse(ThemeAnimationDuration), widgets.ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}), data, child}
}

// Data is the specification of the color and typography to use in this
// widget's subtree.
//
// Flutter: material/theme.dart AnimatedTheme.data
func (w *AnimatedTheme) Data() *ThemeData { return w.data }

// Child is the widget below this widget in the tree.
//
// Flutter: material/theme.dart AnimatedTheme.child
func (w *AnimatedTheme) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: material/theme.dart AnimatedTheme.createState
func (w *AnimatedTheme) CreateState() widgets.State { return &animatedThemeState{} }

type animatedThemeState struct {
	widgets.AnimatedWidgetBaseState[*AnimatedTheme]
	data *animation.Tween[*ThemeData]
}

func (s *animatedThemeState) ForEachTween(v *widgets.TweenVisitor) {
	widgets.VisitTween(v, &s.data, foundation.Some(s.Widget().data), (*ThemeData).Equal, func(x *ThemeData) *animation.Tween[*ThemeData] {
		return NewThemeDataTween(x, x)
	})
}

func (s *animatedThemeState) Build(widgets.BuildContext) widgets.Widget {
	return NewTheme(s.data.Evaluate(s.Animation()), s.Widget().child)
}
