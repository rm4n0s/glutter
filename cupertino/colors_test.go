package cupertino_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/cupertino/colors_test.dart: equality, the string form,
// the constructors, resolving a null color, the dependencies of the resolution on the
// brightness, the contrast and the level, and a color in a theme. The paints matcher of
// DependentWidget is the color that the build function resolved; the tests that use a
// CupertinoApp are in app_test.go (step 34.2), and the ones for toARGB32 and value have
// nothing to test, as EffectiveColor is the only accessor.

var (
	color0 = colorOf(0)
	color1 = colorOf(1)
	color2 = colorOf(2)
	color3 = colorOf(3)
	color4 = colorOf(4)
	color5 = colorOf(5)
	color6 = colorOf(6)
	color7 = colorOf(7)

	dynamicColor = cupertino.NewCupertinoDynamicColor(color0, color1, color3, color5, color2, color4, color6, color7)
	// notSoDynamicColor1 is the same color for every combination.
	notSoDynamicColor1 = cupertino.NewCupertinoDynamicColor(color0, color0, color0, color0, color0, color0, color0, color0)
	// vibrancyDependentColor1 depends on the brightness only.
	vibrancyDependentColor1 = cupertino.NewCupertinoDynamicColor(color1, color0, color1, color0, color1, color0, color1, color0)
	// contrastDependentColor1 depends on the contrast only.
	contrastDependentColor1 = cupertino.NewCupertinoDynamicColor(color1, color1, color0, color0, color1, color1, color0, color0)
	// elevationDependentColor1 depends on the level only.
	elevationDependentColor1 = cupertino.NewCupertinoDynamicColor(color1, color1, color1, color1, color0, color0, color0, color0)
)

// resolvedColor pumps a widget tree made by wrap around a builder that resolves the color,
// and returns the color it resolved, with no error reported by the framework.
func resolvedColor(t *testing.T, c ui.ColorValue, wrap func(child cupertino.Widget) cupertino.Widget) ui.Color {
	t.Helper()
	ts := gluttertest.New(t)
	return resolvedColorIn(t, ts, c, wrap)
}

func resolvedColorIn(t *testing.T, ts *gluttertest.Tester, c ui.ColorValue, wrap func(child cupertino.Widget) cupertino.Widget) ui.Color {
	t.Helper()
	var got ui.Color
	var child cupertino.Widget = cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		got = cupertino.CupertinoDynamicColorResolve(c, bc)
		return cupertino.NewSizedBoxExpand(nil)
	})
	if wrap != nil {
		child = wrap(child)
	}
	ts.PumpWidget(child)
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("the framework reported %v", errs)
	}
	return got
}

func mediaQuery(opts cupertino.MediaQueryDataOpts) func(cupertino.Widget) cupertino.Widget {
	return func(child cupertino.Widget) cupertino.Widget {
		return cupertino.NewMediaQuery(cupertino.NewMediaQueryData(opts), child)
	}
}

func TestDynamicColorEquality(t *testing.T) {
	tests := []struct {
		name string
		a, b cupertino.CupertinoDynamicColor
		want bool
	}{
		{"the same eight colors", dynamicColor, cupertino.NewCupertinoDynamicColor(color0, color1, color3, color5, color2, color4, color6, color7), true},
		{"a color that depends on nothing against one that depends on the brightness", notSoDynamicColor1, vibrancyDependentColor1, false},
		{"a color that depends on nothing against one that depends on the contrast", notSoDynamicColor1, contrastDependentColor1, false},
		{"the brightness dependent color against one that does not depend on anything", vibrancyDependentColor1, notSoDynamicColor1, false},
		{"the debug label does not count", cupertino.NewCupertinoDynamicColor(color0, color1, color3, color5, color2, color4, color6, color7, cupertino.CupertinoDynamicColorOpts{DebugLabel: "x"}), dynamicColor, true},
		{"a resolved color has another effective color", dynamicColor.ResolveFrom(darkContext(t)), dynamicColor, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal = %v, want %v", got, tt.want)
			}
		})
	}
}

// darkContext is a build context in which the platform brightness is dark.
func darkContext(t *testing.T) cupertino.BuildContext {
	t.Helper()
	ts := gluttertest.New(t)
	var ctx cupertino.BuildContext
	ts.PumpWidget(mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark})(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		ctx = bc
		return cupertino.NewSizedBoxExpand(nil)
	})))
	return ctx
}

func TestDynamicColorString(t *testing.T) {
	c := func(n uint32) string { return colorOf(n).String() }
	tests := []struct {
		name  string
		color cupertino.CupertinoDynamicColor
		want  string
	}{
		{"every color", dynamicColor, "CupertinoDynamicColor(*color = " + c(0) + "*, darkColor = " + c(1) + ", highContrastColor = " + c(3) +
			", darkHighContrastColor = " + c(5) + ", elevatedColor = " + c(2) + ", darkElevatedColor = " + c(4) +
			", highContrastElevatedColor = " + c(6) + ", darkHighContrastElevatedColor = " + c(7)},
		{"nothing to depend on", notSoDynamicColor1, "CupertinoDynamicColor(*color = " + c(0) + "*"},
		{"the brightness", vibrancyDependentColor1, "CupertinoDynamicColor(*color = " + c(1) + "*, darkColor = " + c(0)},
		{"the contrast", contrastDependentColor1, "CupertinoDynamicColor(*color = " + c(1) + "*, highContrastColor = " + c(0)},
		{"the level", elevationDependentColor1, "CupertinoDynamicColor(*color = " + c(1) + "*, elevatedColor = " + c(0)},
		{"brightness and contrast", cupertino.NewCupertinoDynamicColorWithBrightnessAndContrast(color0, color1, color2, color3),
			"CupertinoDynamicColor(*color = " + c(0) + "*, darkColor = " + c(1) + ", highContrastColor = " + c(2) + ", darkHighContrastColor = " + c(3)},
		{"a debug label", cupertino.CupertinoColors.SystemBlue, "systemBlue(*color = "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.color.String(); !strings.Contains(got, tt.want) {
				t.Errorf("String = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestMaybeResolveOfNullColor(t *testing.T) {
	ts := gluttertest.New(t)
	var some bool
	ts.PumpWidget(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		_, some = cupertino.CupertinoDynamicColorMaybeResolve(nil, bc).Get()
		return cupertino.NewSizedBoxExpand(nil)
	}))
	if some {
		t.Error("a null color resolved to a color")
	}
}

func TestDynamicColorConstructors(t *testing.T) {
	tests := []struct {
		name string
		a, b cupertino.CupertinoDynamicColor
	}{
		{"with brightness", vibrancyDependentColor1, cupertino.NewCupertinoDynamicColorWithBrightness(color1, color0)},
		{"with brightness and contrast", contrastDependentColor1, cupertino.NewCupertinoDynamicColorWithBrightnessAndContrast(color1, color1, color0, color0)},
		{"with brightness and contrast, again", cupertino.NewCupertinoDynamicColor(color0, color1, color2, color3, color0, color1, color2, color3),
			cupertino.NewCupertinoDynamicColorWithBrightnessAndContrast(color0, color1, color2, color3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.a.Equal(tt.b) {
				t.Errorf("%v != %v", tt.a, tt.b)
			}
		})
	}
}

func TestDynamicColorsThatAreNotDynamicClaimNoDependencies(t *testing.T) {
	// With no MediaQuery and no theme above, a color that does not depend on anything resolves, and so do the others, to the light ones.
	ts := gluttertest.New(t)
	var ctx cupertino.BuildContext
	ts.PumpWidget(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		ctx = bc
		return cupertino.NewSizedBoxExpand(nil)
	}))
	if got := notSoDynamicColor1.ResolveFrom(ctx).EffectiveColor(); got != color0 {
		t.Errorf("resolved to %v, want %v", got, color0)
	}
}

func TestDynamicColorsFollowTheirDependencies(t *testing.T) {
	theme := func(b ui.Brightness) func(cupertino.Widget) cupertino.Widget {
		return func(child cupertino.Widget) cupertino.Widget {
			return cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: b}), child)
		}
	}
	level := func(l cupertino.CupertinoUserInterfaceLevelData) func(cupertino.Widget) cupertino.Widget {
		return func(child cupertino.Widget) cupertino.Widget {
			return cupertino.NewCupertinoUserInterfaceLevel(l, child)
		}
	}
	tests := []struct {
		name  string
		color cupertino.CupertinoDynamicColor
		wrap  func(cupertino.Widget) cupertino.Widget
		want  ui.Color
	}{
		{"brightness: the default is light", vibrancyDependentColor1, mediaQuery(cupertino.MediaQueryDataOpts{}), color1},
		{"brightness: the platform's dark", vibrancyDependentColor1, mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark}), color0},
		{"brightness: the theme's light wins over the platform's dark", vibrancyDependentColor1, func(c cupertino.Widget) cupertino.Widget {
			return theme(ui.BrightnessLight)(mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark})(c))
		}, color1},
		{"contrast: normal", contrastDependentColor1, mediaQuery(cupertino.MediaQueryDataOpts{}), color1},
		{"contrast: high", contrastDependentColor1, mediaQuery(cupertino.MediaQueryDataOpts{HighContrast: true}), color0},
		{"level: base", elevationDependentColor1, level(cupertino.CupertinoUserInterfaceLevelBase), color1},
		{"level: elevated", elevationDependentColor1, level(cupertino.CupertinoUserInterfaceLevelElevated), color0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvedColor(t, tt.color, tt.wrap); got != tt.want {
				t.Errorf("resolved to %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDynamicColorsChangeWhenTheirDependenciesChange(t *testing.T) {
	ts := gluttertest.New(t)
	for i, step := range []struct {
		opts cupertino.MediaQueryDataOpts
		want ui.Color
	}{
		{cupertino.MediaQueryDataOpts{}, color1},
		{cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark}, color0},
		{cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessLight}, color1},
	} {
		if got := resolvedColorIn(t, ts, vibrancyDependentColor1, mediaQuery(step.opts)); got != step.want {
			t.Errorf("step %d: resolved to %v, want %v", i, got, step.want)
		}
	}
}

func TestDynamicColorWithAllThreeDependencies(t *testing.T) {
	rainbow := cupertino.NewCupertinoDynamicColor(color0, color1, color2, color3, color7, color4, color5, color6)
	base, elevated := cupertino.CupertinoUserInterfaceLevelBase, cupertino.CupertinoUserInterfaceLevelElevated
	tests := []struct {
		name  string
		dark  bool
		high  bool
		level cupertino.CupertinoUserInterfaceLevelData
		want  ui.Color
	}{
		{"light, normal, base", false, false, base, color0},
		{"dark, normal, base", true, false, base, color1},
		{"light, high, base", false, true, base, color2},
		{"dark, high, base", true, true, base, color3},
		{"dark, normal, elevated", true, false, elevated, color4},
		{"light, high, elevated", false, true, elevated, color5},
		{"dark, high, elevated", true, true, elevated, color6},
		{"light, normal, elevated", false, false, elevated, color7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := ui.BrightnessLight
			if tt.dark {
				b = ui.BrightnessDark
			}
			got := resolvedColor(t, rainbow, func(c cupertino.Widget) cupertino.Widget {
				return mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: b, HighContrast: tt.high})(cupertino.NewCupertinoUserInterfaceLevel(tt.level, c))
			})
			if got != tt.want {
				t.Errorf("resolved to %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDynamicColorUsedInACupertinoTheme is the first half of the CupertinoApp test of
// Flutter: a theme with a dynamic primary color gives the color of its brightness, and the
// result can be resolved again in another context.
func TestDynamicColorUsedInACupertinoTheme(t *testing.T) {
	tests := []struct {
		name  string
		theme cupertino.CupertinoThemeDataOpts
		wrap  func(cupertino.Widget) cupertino.Widget
		want  ui.Color
	}{
		{"dark theme", cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark, PrimaryColor: dynamicColor}, nil, dynamicColor.DarkColor},
		{"light theme", cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight, PrimaryColor: dynamicColor}, nil, dynamicColor.Color},
		{"no brightness, media query and level at the base", cupertino.CupertinoThemeDataOpts{PrimaryColor: dynamicColor}, func(c cupertino.Widget) cupertino.Widget {
			return mediaQuery(cupertino.MediaQueryDataOpts{})(cupertino.NewCupertinoUserInterfaceLevel(cupertino.CupertinoUserInterfaceLevelBase, c))
		}, dynamicColor.Color},
		{"dark, high contrast and elevated", cupertino.CupertinoThemeDataOpts{PrimaryColor: dynamicColor}, func(c cupertino.Widget) cupertino.Widget {
			return mediaQuery(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark, HighContrast: true})(cupertino.NewCupertinoUserInterfaceLevel(cupertino.CupertinoUserInterfaceLevelElevated, c))
		}, dynamicColor.DarkHighContrastElevatedColor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var got cupertino.CupertinoDynamicColor
			var inner cupertino.Widget = cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
				got = asDynamic(t, cupertino.CupertinoThemeOf(bc).PrimaryColor())
				return cupertino.NewSizedBoxExpand(nil)
			})
			if tt.wrap != nil {
				inner = tt.wrap(inner)
			}
			ts.PumpWidget(cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(tt.theme), inner))
			if got.EffectiveColor() != tt.want {
				t.Errorf("the primary color resolved to %v, want %v", got.EffectiveColor(), tt.want)
			}
		})
	}
}

func TestCupertinoColorsAreTheOnesOfFlutter(t *testing.T) {
	c := cupertino.CupertinoColors
	tests := []struct {
		name  string
		color cupertino.CupertinoDynamicColor
		light uint32
		dark  uint32
		hc    uint32
		dhc   uint32
	}{
		{"systemBlue", c.SystemBlue, 0xFF007AFF, 0xFF0A84FF, 0xFF0040DD, 0xFF409CFF},
		{"systemRed", c.SystemRed, 0xFFFF3B30, 0xFFFF453A, 0xFFD70015, 0xFFFF6961},
		{"label", c.Label, 0xFF000000, 0xFFFFFFFF, 0xFF000000, 0xFFFFFFFF},
		{"secondaryLabel", c.SecondaryLabel, 0x993C3C43, 0x99EBEBF5, 0xAD3C3C43, 0xADEBEBF5},
		{"systemBackground", c.SystemBackground, 0xFFFFFFFF, 0xFF000000, 0xFFFFFFFF, 0xFF000000},
		{"separator", c.Separator, 0x493C3C43, 0x99545458, 0x5E3C3C43, 0xAD545458},
		{"inactiveGray", c.InactiveGray, 0xFF999999, 0xFF757575, 0xFF999999, 0xFF757575},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, e := range []struct {
				which string
				got   ui.Color
				want  uint32
			}{{"color", tt.color.Color, tt.light}, {"darkColor", tt.color.DarkColor, tt.dark},
				{"highContrastColor", tt.color.HighContrastColor, tt.hc}, {"darkHighContrastColor", tt.color.DarkHighContrastColor, tt.dhc}} {
				if e.got != ui.NewColor(e.want) {
					t.Errorf("%s = %v, want %v", e.which, e.got, ui.NewColor(e.want))
				}
			}
		})
	}
	if c.ActiveBlue != c.SystemBlue || c.DestructiveRed != c.SystemRed || c.ActiveGreen != c.SystemGreen || c.ActiveOrange != c.SystemOrange {
		t.Error("an active color is not the system color it stands for")
	}
	if c.White != ui.NewColor(0xFFFFFFFF) || c.Black != ui.NewColor(0xFF000000) || c.Transparent != ui.NewColor(0) {
		t.Error("a plain color has another value")
	}
}

func TestDynamicColorIsAColorValue(t *testing.T) {
	var _ ui.ColorValue = cupertino.CupertinoColors.SystemBlue
	var _ ui.ColorValue = ui.NewColor(0xFF112233)
	if got := ui.ColorValue(ui.NewColor(0xFF112233)).EffectiveColor(); got != ui.NewColor(0xFF112233) {
		t.Errorf("a Color's effective color is %v", got)
	}
	if cupertino.CupertinoColors.SystemBlue.EffectiveColor() != cupertino.CupertinoColors.SystemBlue.Color {
		t.Error("an unresolved color is not its light color")
	}
	_ = foundation.DebugMode
}
