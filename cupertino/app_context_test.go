package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are the cases of colors_test.dart, localizations_test.dart and debug_test.dart of
// packages/flutter/test/cupertino that use a CupertinoApp.

func TestCupertinoLocalizationsAreInsideTheApp(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	outside, inside := widgets.NewGlobalKey[widgets.State](), widgets.NewGlobalKey[widgets.State]()
	ts.PumpWidget(widgets.NewContainer(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewContainer(nil, widgets.ContainerOpts{Key: inside})}),
		widgets.ContainerOpts{Key: outside}))

	t.Run("the context outside the app has none", func(t *testing.T) {
		err := recoverError(func() { cupertino.CupertinoLocalizationsOf(outside.CurrentContext()) })
		appErr := requireErrorTag(t, err, "CupertinoLocalizationsOf->DebugCheckHasCupertinoLocalizations.CupertinoLocalizationsMissing")
		if got := appErr.Message; len(got) < 30 || got[:len("No CupertinoLocalizations found")] != "No CupertinoLocalizations found" {
			t.Errorf("message = %q", got)
		}
		err = recoverError(func() { cupertino.DebugCheckHasCupertinoLocalizations(outside.CurrentContext()) })
		checkErrorTag(t, err, "DebugCheckHasCupertinoLocalizations.CupertinoLocalizationsMissing")
	})
	t.Run("the context inside the app has them", func(t *testing.T) {
		if _, ok := cupertino.CupertinoLocalizationsMaybeOf(inside.CurrentContext()); !ok {
			t.Error("the context inside the app has no localizations")
		}
		if !cupertino.DebugCheckHasCupertinoLocalizations(inside.CurrentContext()) {
			t.Error("the check failed inside the app")
		}
	})
}

func TestCupertinoDynamicColorUsedInACupertinoTheme(t *testing.T) {
	ts := gluttertest.New(t)
	var color ui.ColorValue
	capture := func(child widgets.Widget) widgets.WidgetBuilder {
		return func(bc widgets.BuildContext) widgets.Widget {
			color = cupertino.CupertinoThemeOf(bc).PrimaryColor()
			return child
		}
	}
	effective := func() ui.Color { return color.EffectiveColor() }
	app := func(brightness ui.Brightness, home widgets.Widget) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{
			Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: brightness, PrimaryColor: dynamicColor}), Home: home})
	}
	probe := func() widgets.Widget {
		return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget { return capture(widgets.NewPlaceholder(nil))(bc) })
	}

	ts.PumpWidget(app(ui.BrightnessDark, probe()))
	if got := effective(); got != dynamicColor.DarkColor {
		t.Errorf("dark: %v, want %v", got, dynamicColor.DarkColor)
	}
	// Changing dependencies works.
	ts.PumpWidget(app(ui.BrightnessLight, probe()))
	if got := effective(); got != dynamicColor.Color {
		t.Errorf("light: %v, want %v", got, dynamicColor.Color)
	}
	// Having a dependency below the CupertinoTheme widget works.
	ts.PumpWidget(app(ui.BrightnessUnspecified, widgets.NewMediaQuery(widgets.MediaQueryData{},
		cupertino.NewCupertinoUserInterfaceLevel(cupertino.CupertinoUserInterfaceLevelBase, probe()))))
	if got := effective(); got != dynamicColor.Color {
		t.Errorf("base level: %v, want %v", got, dynamicColor.Color)
	}
	// No brightness is given, so it defers to the MediaQuery; changing dependencies works.
	ts.PumpWidget(app(ui.BrightnessUnspecified, widgets.NewMediaQuery(widgets.MediaQueryData{PlatformBrightness: ui.BrightnessDark, HighContrast: true},
		cupertino.NewCupertinoUserInterfaceLevel(cupertino.CupertinoUserInterfaceLevelElevated, probe()))))
	if got := effective(); got != dynamicColor.DarkHighContrastElevatedColor {
		t.Errorf("dark, high contrast, elevated: %v, want %v", got, dynamicColor.DarkHighContrastElevatedColor)
	}
}
