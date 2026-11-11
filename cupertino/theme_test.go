package cupertino_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/cupertino/theme_test.dart: the defaults, the cascade
// of the primary color, the override of the dependent attributes, the dependencies of the
// theme, copyWith, the icon theme of the primary color, the diagnostics, the equality, and
// the resolution of the colors in the light and the dark brightness.

// themeProbe pumps a CupertinoTheme with the data around a builder that reads the theme
// and the icon theme, and counts its builds.
type themeProbe struct {
	ts        *gluttertest.Tester
	builds    int
	theme     *cupertino.CupertinoThemeData
	iconTheme cupertino.IconThemeData
	subtree   cupertino.Widget
}

func newThemeProbe(t *testing.T) *themeProbe {
	p := &themeProbe{ts: gluttertest.New(t)}
	p.subtree = cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		p.builds++
		p.theme = cupertino.CupertinoThemeOf(bc)
		p.iconTheme = cupertino.IconThemeOf(bc)
		return cupertino.NewSizedBox(nil)
	})
	return p
}

func (p *themeProbe) pump(data *cupertino.CupertinoThemeData) {
	p.ts.PumpWidget(cupertino.NewCupertinoTheme(data, p.subtree))
}

func colorEquals(a, b ui.ColorValue) bool { return a.EffectiveColor() == b.EffectiveColor() }

func TestThemeDefaultsAreTheDefaults(t *testing.T) {
	p := newThemeProbe(t)
	p.pump(cupertino.NewCupertinoThemeData())
	theme := p.theme
	if theme.Brightness() != ui.BrightnessUnspecified {
		t.Errorf("brightness = %v, want none", theme.Brightness())
	}
	if !asDynamic(t, theme.PrimaryColor()).Equal(cupertino.CupertinoColors.ActiveBlue) {
		t.Errorf("primary color = %v, want activeBlue", theme.PrimaryColor())
	}
	if got, _ := theme.TextTheme().TextStyle().FontSize.Get(); got != 17 {
		t.Errorf("font size = %v, want 17", got)
	}
	if theme.ApplyThemeToAll() {
		t.Error("applyThemeToAll is true by default")
	}
}

func TestThemeAttributesCascade(t *testing.T) {
	p := newThemeProbe(t)
	p.pump(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{PrimaryColor: cupertino.CupertinoColors.SystemRed}))
	got, _ := p.theme.TextTheme().ActionTextStyle().Color.Get()
	if got != cupertino.CupertinoColors.SystemRed.Color {
		t.Errorf("action text color = %v, want %v", got, cupertino.CupertinoColors.SystemRed.Color)
	}
}

func TestDependentAttributeCanBeOverriddenFromCascadedValue(t *testing.T) {
	p := newThemeProbe(t)
	textTheme := cupertino.NewCupertinoTextThemeData(cupertino.CupertinoTextThemeDataOpts{
		TextStyle: painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(cupertino.CupertinoColors.Black)}),
	})
	p.pump(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark, TextTheme: textTheme}))
	if !colorEquals(p.theme.ScaffoldBackgroundColor(), cupertino.CupertinoColors.Black) {
		t.Errorf("scaffold background = %v, want black", p.theme.ScaffoldBackgroundColor().EffectiveColor())
	}
	if got, _ := p.theme.TextTheme().TextStyle().Color.Get(); got != cupertino.CupertinoColors.Black {
		t.Errorf("text color = %v, want black", got)
	}
}

func TestReadingThemesCreatesDependencies(t *testing.T) {
	style := func(family string) *cupertino.CupertinoTextThemeData {
		return cupertino.NewCupertinoTextThemeData(cupertino.CupertinoTextThemeDataOpts{
			TextStyle: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: family})})
	}
	bar := ui.NewColor(0x11223344)
	tests := []struct {
		name       string
		data       cupertino.CupertinoThemeDataOpts
		wantBuilds int
		wantFamily string
	}{
		{"first", cupertino.CupertinoThemeDataOpts{BarBackgroundColor: bar, TextTheme: style("Skeuomorphic")}, 1, "Skeuomorphic"},
		{"a brightness", cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight, BarBackgroundColor: bar, TextTheme: style("Skeuomorphic")}, 2, "Skeuomorphic"},
		{"a font family", cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight, BarBackgroundColor: bar, TextTheme: style("Flat")}, 3, "Flat"},
	}
	p := newThemeProbe(t)
	for _, tt := range tests {
		p.pump(cupertino.NewCupertinoThemeData(tt.data))
		if p.builds != tt.wantBuilds {
			t.Errorf("%s: %d builds, want %d", tt.name, p.builds, tt.wantBuilds)
		}
		if got := p.theme.TextTheme().TextStyle().FontFamily; got != tt.wantFamily {
			t.Errorf("%s: font family %q, want %q", tt.name, got, tt.wantFamily)
		}
	}
}

func TestThemeCopyWithWorks(t *testing.T) {
	original := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark, ApplyThemeToAll: foundation.Some(true)})
	p := newThemeProbe(t)
	p.pump(original.CopyWith(cupertino.CupertinoThemeDataOpts{PrimaryColor: cupertino.CupertinoColors.SystemGreen, ApplyThemeToAll: foundation.Some(false)}))
	theme := p.theme
	if theme.Brightness() != ui.BrightnessDark {
		t.Errorf("brightness = %v", theme.Brightness())
	}
	green := cupertino.CupertinoColors.SystemGreen.DarkColor
	if theme.PrimaryColor().EffectiveColor() != green {
		t.Errorf("primary = %v, want %v", theme.PrimaryColor().EffectiveColor(), green)
	}
	if got, _ := theme.TextTheme().ActionTextStyle().Color.Get(); got != green {
		t.Errorf("action text color = %v, want %v", got, green)
	}
	if !colorEquals(theme.ScaffoldBackgroundColor(), cupertino.CupertinoColors.Black) {
		t.Errorf("scaffold background = %v", theme.ScaffoldBackgroundColor().EffectiveColor())
	}
	if theme.ApplyThemeToAll() {
		t.Error("applyThemeToAll is true")
	}
}

func TestThemeHasDefaultIconThemeDerivedFromThePrimaryColor(t *testing.T) {
	primary := cupertino.CupertinoColors.SystemRed
	data := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{PrimaryColor: primary})
	p := newThemeProbe(t)
	p.pump(data)
	if got, _ := p.iconTheme.Color().Get(); got != primary.Color {
		t.Errorf("icon color = %v, want %v", got, primary.Color)
	}
	p.pump(data.CopyWith(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark}))
	if got, _ := p.iconTheme.Color().Get(); got != primary.DarkColor {
		t.Errorf("dark icon color = %v, want %v", got, primary.DarkColor)
	}
}

func TestIconThemeOfCreatesADependencyOnTheIconTheme(t *testing.T) {
	p := newThemeProbe(t)
	for i, c := range []cupertino.CupertinoDynamicColor{cupertino.CupertinoColors.DestructiveRed, cupertino.CupertinoColors.ActiveOrange} {
		p.pump(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{PrimaryColor: c}))
		if p.builds != i+1 {
			t.Errorf("step %d: %d builds", i, p.builds)
		}
		if got, _ := p.iconTheme.Color().Get(); got != c.Color {
			t.Errorf("step %d: icon color %v, want %v", i, got, c.Color)
		}
	}
}

func TestThemeDiagnostics(t *testing.T) {
	skipUnlessDebug(t) // diagnostics are empty in a release build
	b := foundation.NewDiagnosticPropertiesBuilder()
	cupertino.NewCupertinoThemeData().DebugFillProperties(b)
	var names []string
	for _, n := range b.Properties {
		if name, ok := n.Name().Get(); ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	want := []string{"actionSmallTextStyle", "actionTextStyle", "applyThemeToAll", "barBackgroundColor", "dateTimePickerTextStyle",
		"navActionTextStyle", "navLargeTitleTextStyle", "navTitleTextStyle", "pickerTextStyle", "primaryColor", "primaryContrastingColor",
		"scaffoldBackgroundColor", "selectionHandleColor", "tabLabelTextStyle", "textStyle"}
	// Flutter's brightness property is in the set too; it is left out here when there is none.
	if !slices.Equal(names, want) {
		t.Errorf("properties = %v, want %v", names, want)
	}
}

func TestThemeToStringDeepUsesSingleLineStyle(t *testing.T) {
	theme := cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{PrimaryColor: cupertino.CupertinoColors.Transparent}), cupertino.NewSizedBox(nil))
	if got := strings.TrimRight(cupertino.DescribeWidget(theme).ToStringDeep(), "\n "); strings.Contains(got, "\n") {
		t.Errorf("the string form has several lines: %q", got)
	}
}

func TestThemeDataEquality(t *testing.T) {
	a := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark})
	b := a.CopyWith(cupertino.CupertinoThemeDataOpts{})
	c := a.CopyWith(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight})
	tests := []struct {
		name string
		x, y *cupertino.CupertinoThemeData
		want bool
	}{
		{"a, b", a, b, true}, {"b, a", b, a, true}, {"a, c", a, c, false}, {"c, a", c, a, false}, {"b, c", b, c, false}, {"c, b", c, b, false},
	}
	for _, tt := range tests {
		if got := tt.x.Equal(tt.y); got != tt.want {
			t.Errorf("%s: Equal = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNoDefaultThemeDataEquality(t *testing.T) {
	a := cupertino.NewNoDefaultCupertinoThemeData()
	b := a.CopyWith(cupertino.NoDefaultCupertinoThemeDataOpts{})
	c := a.CopyWith(cupertino.NoDefaultCupertinoThemeDataOpts{Brightness: ui.BrightnessLight})
	if !a.Equal(b) || a.Equal(c) {
		t.Errorf("a == b: %v, a == c: %v", a.Equal(b), a.Equal(c))
	}
	if cupertino.NewCupertinoThemeData().NoDefault().PrimaryColor() != nil {
		t.Error("the data without defaults has a primary color")
	}
}

func TestThemeBrightnessFallsBackToThePlatform(t *testing.T) {
	ts := gluttertest.New(t)
	var withTheme, withoutTheme ui.Brightness
	var maybeOK bool
	probe := cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		withTheme = cupertino.CupertinoThemeBrightnessOf(bc)
		return cupertino.NewSizedBox(nil)
	})
	ts.PumpWidget(mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark})(
		cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(), probe)))
	if withTheme != ui.BrightnessDark {
		t.Errorf("a theme with no brightness gave %v, want the platform's dark", withTheme)
	}
	ts.PumpWidget(mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark})(
		cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight}), probe)))
	if withTheme != ui.BrightnessLight {
		t.Errorf("the theme's light brightness gave %v", withTheme)
	}
	ts.PumpWidget(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		withoutTheme, maybeOK = cupertino.CupertinoThemeMaybeBrightnessOf(bc)
		return cupertino.NewSizedBox(nil)
	}))
	if !maybeOK || withoutTheme != ui.BrightnessLight {
		t.Errorf("with the view's media query only: %v, %v", withoutTheme, maybeOK)
	}
}

func TestThemeDynamicColorsResolve(t *testing.T) {
	for _, brightness := range []ui.Brightness{ui.BrightnessLight, ui.BrightnessDark} {
		pick := func(c ui.ColorValue) ui.Color {
			d, ok := c.(cupertino.CupertinoDynamicColor)
			if !ok {
				return c.EffectiveColor()
			}
			if brightness == ui.BrightnessDark {
				return d.DarkColor
			}
			return d.Color
		}
		t.Run(brightness.String()+" resolves colors", func(t *testing.T) {
			data := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: brightness, PrimaryColor: cupertino.CupertinoColors.SystemRed})
			p := newThemeProbe(t)
			p.pump(data)
			if data.PrimaryColor().EffectiveColor() != cupertino.CupertinoColors.SystemRed.Color {
				t.Error("the data that was given resolved itself")
			}
			if got, want := p.theme.PrimaryColor().EffectiveColor(), pick(cupertino.CupertinoColors.SystemRed); got != want {
				t.Errorf("primary = %v, want %v", got, want)
			}
		})
		t.Run(brightness.String()+" resolves the default values", func(t *testing.T) {
			primary := cupertino.CupertinoColors.SystemRed
			data := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: brightness, PrimaryColor: primary})
			bar := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xF0F9F9F9), ui.NewColor(0xF01D1D1D))
			p := newThemeProbe(t)
			p.pump(data)
			th := p.theme
			textColor := func(s *painting.TextStyle) ui.Color { c, _ := s.Color.Get(); return c }
			for _, e := range []struct {
				name string
				got  ui.Color
				want ui.Color
			}{
				{"primaryContrastingColor", th.PrimaryContrastingColor().EffectiveColor(), pick(cupertino.CupertinoColors.White)},
				{"barBackgroundColor", th.BarBackgroundColor().EffectiveColor(), pick(bar)},
				{"scaffoldBackgroundColor", th.ScaffoldBackgroundColor().EffectiveColor(), pick(cupertino.CupertinoColors.SystemBackground)},
				{"selectionHandleColor", th.SelectionHandleColor().EffectiveColor(), pick(cupertino.CupertinoColors.SystemBlue)},
				{"textStyle", textColor(th.TextTheme().TextStyle()), pick(cupertino.CupertinoColors.Label)},
				{"actionTextStyle", textColor(th.TextTheme().ActionTextStyle()), pick(primary)},
				{"tabLabelTextStyle", textColor(th.TextTheme().TabLabelTextStyle()), pick(cupertino.CupertinoColors.InactiveGray)},
				{"navTitleTextStyle", textColor(th.TextTheme().NavTitleTextStyle()), pick(cupertino.CupertinoColors.Label)},
				{"navLargeTitleTextStyle", textColor(th.TextTheme().NavLargeTitleTextStyle()), pick(cupertino.CupertinoColors.Label)},
				{"navActionTextStyle", textColor(th.TextTheme().NavActionTextStyle()), pick(primary)},
				{"pickerTextStyle", textColor(th.TextTheme().PickerTextStyle()), pick(cupertino.CupertinoColors.Label)},
				{"dateTimePickerTextStyle", textColor(th.TextTheme().DateTimePickerTextStyle()), pick(cupertino.CupertinoColors.Label)},
			} {
				if e.got != e.want {
					t.Errorf("%s = %v, want %v", e.name, e.got, e.want)
				}
			}
		})
	}
}
