package material_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/material/app_test.dart. A
// GestureDetector over a Text stands in for the ElevatedButton and a Focus for
// the TextField, which come with their components (step 13); the cases of the
// Cupertino localizations, the debug grid, the restoration and the cases that
// need the Scaffold are not ported. The overscroll indicator cases of Android
// (step 26.1) are at the end of the file.

var includeOffstage = gluttertest.FindOpts{IncludeOffstage: true}

// tapText is a button: tapping its text calls onTap.
func tapText(label string, onTap func()) widgets.Widget {
	return widgets.NewGestureDetector(widgets.NewText(label), widgets.GestureDetectorOpts{OnTap: onTap})
}

func TestThemeModeGetters(t *testing.T) {
	tests := []struct {
		mode                      ThemeMode
		name                      string
		isSystem, isLight, isDark bool
	}{
		{ThemeModeSystem, "ThemeMode.system", true, false, false},
		{ThemeModeUnspecified, "ThemeMode.system", true, false, false},
		{ThemeModeLight, "ThemeMode.light", false, true, false},
		{ThemeModeDark, "ThemeMode.dark", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mode.IsSystem() != tt.isSystem || tt.mode.IsLight() != tt.isLight || tt.mode.IsDark() != tt.isDark {
				t.Errorf("%v: isSystem %v isLight %v isDark %v", tt.mode, tt.mode.IsSystem(), tt.mode.IsLight(), tt.mode.IsDark())
			}
			if tt.mode.String() != tt.name {
				t.Errorf("String() = %q, want %q", tt.mode.String(), tt.name)
			}
		})
	}
}

func TestMaterialAppStructure(t *testing.T) {
	t.Run("can nest apps", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterialApp(MaterialAppOpts{Home: widgets.NewText("Home sweet home")})}))
		if n := ts.Count(gluttertest.ByText("Home sweet home")); n != 1 {
			t.Errorf("found %d texts, want 1", n)
		}
	})

	t.Run("focus handling", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode()
		defer node.Dispose()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Home: NewMaterial(widgets.NewCenter(widgets.NewFocus(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)}),
				widgets.FocusOpts{FocusNode: node, Autofocus: true}))),
		}))
		if !node.HasFocus() {
			t.Error("the autofocus node should have the focus")
		}
	})

	t.Run("can place the app inside a FocusScope", func(t *testing.T) {
		ts := gluttertest.New(t)
		scope := widgets.NewFocusScopeNode()
		defer scope.Dispose()
		ts.PumpWidget(widgets.NewFocusScope(NewMaterialApp(MaterialAppOpts{Home: widgets.NewText("Home")}),
			widgets.FocusScopeOpts{Autofocus: true, Node: scope}))
		if n := ts.Count(gluttertest.ByText("Home")); n != 1 {
			t.Errorf("found %d texts, want 1", n)
		}
	})

	t.Run("without a Navigator the builder makes the app", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Builder: func(widgets.BuildContext, widgets.Widget) widgets.Widget {
			return widgets.NewSizedBox(nil)
		}}))
		if ts.Count(gluttertest.ByType[*widgets.Navigator]()) != 0 {
			t.Error("there should be no Navigator")
		}
		if ts.Count(gluttertest.ByType[*widgets.SizedBox]()) == 0 {
			t.Error("the builder should have built its SizedBox")
		}
	})

	t.Run("works at zero area and unconstrained", func(t *testing.T) {
		for _, size := range []ui.Size{{Width: 0, Height: 0}, {Width: 800, Height: 0}} {
			ts := gluttertest.New(t)
			ts.SetViewSize(size)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewText("X")}))
			if err := ts.TakeError(); err != nil {
				t.Errorf("size %v: %v", size, err)
			}
		}
	})
}

func TestMaterialAppRebuilds(t *testing.T) {
	t.Run("does not rebuild the page during a route transition", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("X", func() { widgets.NavigatorPushNamed(bc, "/next") }))
			}),
			Routes: map[string]widgets.WidgetBuilder{"/next": func(widgets.BuildContext) widgets.Widget {
				return widgets.NewBuilder(func(widgets.BuildContext) widgets.Widget { builds++; return widgets.NewText("Y") })
			}},
		}))
		check := func(step string, want int) {
			t.Helper()
			if builds != want {
				t.Errorf("%s: %d builds, want %d", step, builds, want)
			}
		}
		check("start", 0)
		ts.Tap(gluttertest.ByText("X"))
		check("tap", 0)
		ts.Pump()
		check("first frame", 1)
		for i := 0; i < 4; i++ {
			ts.Pump(10 * time.Millisecond)
			check("transition", 1)
		}
		ts.Pump(time.Second)
		check("end", 1)
		if n := ts.Count(gluttertest.ByText("Y")); n != 1 {
			t.Errorf("found %d texts Y, want 1", n)
		}
	})

	t.Run("rebuilds the home page if it changes", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		home := func(text string) widgets.Widget {
			return widgets.NewBuilder(func(widgets.BuildContext) widgets.Widget { builds++; return widgets.NewText(text) })
		}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: home("A")}))
		if builds != 1 || ts.Count(gluttertest.ByText("A")) != 1 {
			t.Fatalf("builds = %d, want 1 and a text A", builds)
		}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: home("B")}))
		if builds != 2 || ts.Count(gluttertest.ByText("B")) != 1 {
			t.Errorf("builds = %d, want 2 and a text B", builds)
		}
	})

	t.Run("does not rebuild the home page if it does not change", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		home := widgets.NewBuilder(func(widgets.BuildContext) widgets.Widget { builds++; return widgets.NewPlaceholder(nil) })
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: home}))
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: home}))
		if builds != 1 {
			t.Errorf("builds = %d, want 1", builds)
		}
	})

	t.Run("rebuilds the pages of the routes table if the app changes", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		routes := map[string]widgets.WidgetBuilder{"/": func(widgets.BuildContext) widgets.Widget { builds++; return widgets.NewPlaceholder(nil) }}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: routes}))
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: routes}))
		if builds != 2 {
			t.Errorf("builds = %d, want 2", builds)
		}
	})
}

func TestMaterialAppRoutes(t *testing.T) {
	routes := func() map[string]widgets.WidgetBuilder {
		route := func(name string) widgets.WidgetBuilder {
			return func(widgets.BuildContext) widgets.Widget { return widgets.NewText(`route "` + name + `"`) }
		}
		return map[string]widgets.WidgetBuilder{"/": route("/"), "/a": route("/a"), "/a/b": route("/a/b"), "/b": route("/b")}
	}
	count := func(ts *gluttertest.Tester, name string, opts ...gluttertest.FindOpts) int {
		return ts.Count(gluttertest.ByText(`route "`+name+`"`, opts...))
	}

	t.Run("cannot pop the initial route", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := widgets.NewGlobalKey[*widgets.NavigatorState]()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{NavigatorKey: key, Home: widgets.NewText("Home")}))
		navigator, ok := key.CurrentState()
		if !ok {
			t.Fatal("the navigator key has no state")
		}
		if navigator.MaybePop() {
			t.Error("the initial route was popped")
		}
		if ts.Count(gluttertest.ByText("Home")) != 1 {
			t.Error("the home page should still be there")
		}
	})

	t.Run("default initial route", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: map[string]widgets.WidgetBuilder{
			"/": func(widgets.BuildContext) widgets.Widget { return widgets.NewText(`route "/"`) }}}))
		if count(ts, "/") != 1 {
			t.Error(`route "/" should be shown`)
		}
	})

	t.Run("one-step initial route", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{InitialRoute: "/a", Routes: routes()}))
		for name, want := range map[string]int{"/": 1, "/a": 1, "/a/b": 0, "/b": 0} {
			opts := includeOffstage
			if name == "/a" {
				opts = gluttertest.FindOpts{}
			}
			if got := count(ts, name, opts); got != want {
				t.Errorf("route %s: %d, want %d", name, got, want)
			}
		}
	})

	t.Run("two-step initial route", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{InitialRoute: "/a/b", Routes: routes()}))
		for name, want := range map[string]int{"/": 1, "/a": 1, "/a/b": 1, "/b": 0} {
			opts := includeOffstage
			if name == "/a/b" {
				opts = gluttertest.FindOpts{}
			}
			if got := count(ts, name, opts); got != want {
				t.Errorf("route %s: %d, want %d", name, got, want)
			}
		}
	})

	t.Run("initial route with a missing step", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{InitialRoute: "/a/b/c", Routes: routes()}))
		err := ts.TakeError()
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("NavigatorInitialRouteUnknown") {
			t.Fatalf("unexpected error %v", err)
		}
		for name, want := range map[string]int{"/": 1, "/a": 0, "/a/b": 0, "/b": 0} {
			if got := count(ts, name); got != want {
				t.Errorf("route %s: %d, want %d", name, got, want)
			}
		}
	})

	t.Run("the initial route is used only the first time", func(t *testing.T) {
		ts := gluttertest.New(t)
		r := routes()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{InitialRoute: "/a", Routes: r}))
		for _, opts := range []MaterialAppOpts{{InitialRoute: "/b", Routes: r}, {Routes: r}} {
			ts.PumpWidget(NewMaterialApp(opts))
			if count(ts, "/", includeOffstage) != 1 || count(ts, "/a") != 1 || count(ts, "/b", includeOffstage) != 0 {
				t.Errorf("changing the initial route to %q had an effect", opts.InitialRoute)
			}
		}
	})

	t.Run("onGenerateRoute and onUnknownRoute", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		var log []string
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			OnGenerateRoute: func(s widgets.RouteSettings) widgets.Route {
				log = append(log, "onGenerateRoute "+s.Name())
				return nil
			},
			OnUnknownRoute: func(s widgets.RouteSettings) widgets.Route { log = append(log, "onUnknownRoute "+s.Name()); return nil },
		}))
		errs := ts.TakeErrors()
		if len(errs) == 0 {
			t.Fatal("a route that is not generated should report an error")
		}
		// The app reports that onUnknownRoute returned no route, and then the navigator
		// that it has no route to build.
		if appErr, ok := errors.FromError(errs[0]); !ok || !appErr.HasRoute("WidgetsAppUnknownRouteNil") {
			t.Errorf("first error %v", errs[0])
		}
		if len(log) != 2 || log[0] != "onGenerateRoute /" || log[1] != "onUnknownRoute /" {
			t.Errorf("log = %v", log)
		}
	})

	t.Run("the value of pop is returned", func(t *testing.T) {
		ts := gluttertest.New(t)
		var result <-chan any
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("X", func() { result = widgets.NavigatorPushNamed(bc, "/a") }))
			}),
			Routes: map[string]widgets.WidgetBuilder{"/a": func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("Y", func() { widgets.NavigatorPop(bc, "all done") }))
			}},
		}))
		ts.Tap(gluttertest.ByText("X"))
		ts.Pump()
		ts.Pump(time.Second)
		if ts.Count(gluttertest.ByText("Y")) != 1 {
			t.Fatal("the next page should be shown")
		}
		ts.Tap(gluttertest.ByText("Y"))
		ts.Pump()
		select {
		case v := <-result:
			if v != "all done" {
				t.Errorf("result = %v, want all done", v)
			}
		default:
			t.Error("pop did not give its result")
		}
	})

	t.Run("the navigator key", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := widgets.NewGlobalKey[*widgets.NavigatorState]()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{NavigatorKey: key, Home: widgets.NewText("Home"),
			Routes: map[string]widgets.WidgetBuilder{"/a": func(widgets.BuildContext) widgets.Widget { return widgets.NewText("A") }}}))
		navigator, ok := key.CurrentState()
		if !ok {
			t.Fatal("the key has no navigator")
		}
		navigator.PushNamed("/a")
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("A")) != 1 {
			t.Error("the navigator of the key should have pushed the route")
		}
		// A new key is a new navigator.
		key2 := widgets.NewGlobalKey[*widgets.NavigatorState]()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{NavigatorKey: key2, Home: widgets.NewText("Home")}))
		if _, ok := key2.CurrentState(); !ok {
			t.Error("the new key has no navigator")
		}
		if _, ok := key.CurrentState(); ok {
			t.Error("the old key still has a navigator")
		}
	})

	t.Run("the default localizations are there", func(t *testing.T) {
		ts := gluttertest.New(t)
		var material MaterialLocalizations
		var wl widgets.WidgetsLocalizations
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			material = MaterialLocalizationsOf(bc)
			wl = widgets.WidgetsLocalizationsOf(bc)
			return widgets.NewSizedBox(nil)
		})}))
		if material == nil || material.BackButtonTooltip() != "Back" {
			t.Errorf("the Material localizations are %v", material)
		}
		if wl == nil {
			t.Error("the widgets localizations are missing")
		}
	})
}

// useBrightness sets the brightness of the platform, and the accessibility
// features, for the rest of the test.
func useBrightness(ts *gluttertest.Tester, brightness ui.Brightness, highContrast bool) {
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) {
		c.PlatformBrightness = brightness
		c.AccessibilityFeatures = ui.AccessibilityFeatures{HighContrast: highContrast}
	})
}

// appliedTheme pumps the app and gives the theme that the home page sees.
func appliedTheme(ts *gluttertest.Tester, app func(home widgets.Widget) widgets.Widget) *ThemeData {
	var applied *ThemeData
	ts.PumpWidget(app(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		applied = ThemeOf(bc)
		return widgets.NewSizedBox(nil)
	})))
	return applied
}

func TestMaterialAppThemeMode(t *testing.T) {
	light := func() *ThemeData { return NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessLight}) }
	dark := func() *ThemeData { return NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark}) }
	tests := []struct {
		name       string
		platform   ui.Brightness
		mode       ThemeMode
		theme      *ThemeData
		darkTheme  *ThemeData
		wantBright ui.Brightness
	}{
		{"light mode with a light platform", ui.BrightnessLight, ThemeModeLight, light(), dark(), ui.BrightnessLight},
		{"light mode with a dark platform", ui.BrightnessDark, ThemeModeLight, light(), dark(), ui.BrightnessLight},
		{"dark mode with a light platform", ui.BrightnessLight, ThemeModeDark, light(), dark(), ui.BrightnessDark},
		{"dark mode with a dark platform", ui.BrightnessDark, ThemeModeDark, light(), dark(), ui.BrightnessDark},
		{"system mode with a light platform", ui.BrightnessLight, ThemeModeSystem, light(), dark(), ui.BrightnessLight},
		{"system mode with a dark platform", ui.BrightnessDark, ThemeModeSystem, light(), dark(), ui.BrightnessDark},
		{"the default mode is the system one", ui.BrightnessDark, ThemeModeUnspecified, light(), dark(), ui.BrightnessDark},
		{"a dark platform without a dark theme", ui.BrightnessDark, ThemeModeSystem, light(), nil, ui.BrightnessLight},
		{"a dark platform without any theme", ui.BrightnessDark, ThemeModeSystem, nil, nil, ui.BrightnessLight},
		{"a light platform with only a dark theme", ui.BrightnessLight, ThemeModeSystem, nil, dark(), ui.BrightnessLight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			useBrightness(ts, tt.platform, false)
			got := appliedTheme(ts, func(home widgets.Widget) widgets.Widget {
				return NewMaterialApp(MaterialAppOpts{Theme: tt.theme, DarkTheme: tt.darkTheme, ThemeMode: tt.mode, Home: home})
			})
			if got.Brightness() != tt.wantBright {
				t.Errorf("brightness = %v, want %v", got.Brightness(), tt.wantBright)
			}
		})
	}

	t.Run("the theme follows the brightness of the platform", func(t *testing.T) {
		ts := gluttertest.New(t)
		useBrightness(ts, ui.BrightnessLight, false)
		var before, after *ThemeData
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: light(), DarkTheme: dark(), Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			if before == nil {
				before = ThemeOf(bc)
			} else {
				after = ThemeOf(bc)
			}
			return widgets.NewSizedBox(nil)
		})}))
		useBrightness(ts, ui.BrightnessDark, false)
		ts.PumpAndSettle()
		if before.Brightness() != ui.BrightnessLight || after == nil || after.Brightness() != ui.BrightnessDark {
			t.Errorf("themes before %v, after %v", before, after)
		}
	})
}

func TestMaterialAppHighContrast(t *testing.T) {
	primary := func(c ui.Color) *ThemeData {
		return NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(c)})
	}
	lightBlue, blue, lightGreen, green := Colors.LightBlue.Primary, Colors.Blue.Primary, Colors.LightGreen.Primary, Colors.Green.Primary
	tests := []struct {
		name     string
		platform ui.Brightness
		opts     MaterialAppOpts
		want     ui.Color
	}{
		{"the high contrast theme", ui.BrightnessLight, MaterialAppOpts{Theme: primary(lightBlue), HighContrastTheme: primary(blue)}, blue},
		{"the high contrast dark theme", ui.BrightnessDark, MaterialAppOpts{
			Theme: primary(lightBlue), DarkTheme: primary(lightGreen), HighContrastTheme: primary(blue), HighContrastDarkTheme: primary(green)}, green},
		{"the dark theme without a high contrast dark theme", ui.BrightnessDark, MaterialAppOpts{
			Theme: primary(lightBlue), DarkTheme: primary(lightGreen)}, lightGreen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			useBrightness(ts, tt.platform, true)
			got := appliedTheme(ts, func(home widgets.Widget) widgets.Widget {
				o := tt.opts
				o.Home = home
				return NewMaterialApp(o)
			})
			if got.PrimaryColor != tt.want {
				t.Errorf("primary color = %v, want %v", got.PrimaryColor, tt.want)
			}
		})
	}
}

// backgroundOfPage is the color of a Material that takes the scaffold background
// color of the theme: it stands in for the Material of a Scaffold.
func backgroundOfPage() widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return NewMaterial(nil, MaterialOpts{Color: foundation.Some(ThemeOf(bc).ScaffoldBackgroundColor)})
	})
}

func materialColor(ts *gluttertest.Tester) ui.Color {
	return ts.Widget(gluttertest.ByType[*Material]()).(*Material).Color().Value()
}

func TestMaterialAppAnimatesTheme(t *testing.T) {
	lightTheme, darkTheme := NewThemeData(), NewThemeDataDark()

	t.Run("animates the change", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: lightTheme, DarkTheme: darkTheme, ThemeMode: ThemeModeLight, Home: backgroundOfPage()}))
		if got := materialColor(ts); got != lightTheme.ScaffoldBackgroundColor {
			t.Fatalf("color = %v, want %v", got, lightTheme.ScaffoldBackgroundColor)
		}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{DarkTheme: NewThemeDataDark(), ThemeMode: ThemeModeDark, Home: backgroundOfPage()}))
		// Half of the theme animation duration, 200 ms: the default curve is linear.
		ts.Pump(100 * time.Millisecond)
		want := lightTheme.ScaffoldBackgroundColor.Lerp(darkTheme.ScaffoldBackgroundColor, 0.5)
		if got := materialColor(ts); got != want {
			t.Errorf("color = %v, want %v", got, want)
		}
	})

	t.Run("the animation can be turned off with a duration", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		page := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			builds++
			return NewMaterial(nil, MaterialOpts{Color: foundation.Some(ThemeOf(bc).ScaffoldBackgroundColor)})
		})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: lightTheme, DarkTheme: darkTheme, ThemeMode: ThemeModeLight,
			ThemeAnimationDuration: foundation.Some(time.Duration(0)), Home: page}))
		if materialColor(ts) != lightTheme.ScaffoldBackgroundColor || builds != 1 {
			t.Fatalf("color %v, builds %d", materialColor(ts), builds)
		}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{DarkTheme: NewThemeDataDark(), ThemeMode: ThemeModeDark,
			ThemeAnimationDuration: foundation.Some(time.Duration(0)), Home: page}))
		ts.PumpAndSettle()
		if materialColor(ts) != darkTheme.ScaffoldBackgroundColor || builds != 2 {
			t.Errorf("color %v, builds %d", materialColor(ts), builds)
		}
	})

	t.Run("the animation style overrides the duration and the curve", func(t *testing.T) {
		ts := gluttertest.New(t)
		style := animation.AnimationStyle{Duration: foundation.Some(time.Second), Curve: animation.Curves.EaseIn}
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: lightTheme, DarkTheme: darkTheme, ThemeMode: ThemeModeLight, ThemeAnimationStyle: &style, Home: backgroundOfPage()}))
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{DarkTheme: NewThemeDataDark(), ThemeMode: ThemeModeDark, ThemeAnimationStyle: &style, Home: backgroundOfPage()}))
		ts.Pump(500 * time.Millisecond)
		want := lightTheme.ScaffoldBackgroundColor.Lerp(darkTheme.ScaffoldBackgroundColor, animation.Curves.EaseIn.Transform(0.5))
		if got := materialColor(ts); got != want {
			t.Errorf("color = %v, want %v", got, want)
		}
		ts.PumpAndSettle()
		if got := materialColor(ts); got != darkTheme.ScaffoldBackgroundColor {
			t.Errorf("final color = %v, want %v", got, darkTheme.ScaffoldBackgroundColor)
		}
	})

	t.Run("no animation removes the AnimatedTheme from the tree", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewSizedBox(nil)}))
		if ts.Count(gluttertest.ByType[*AnimatedTheme]()) != 1 {
			t.Error("the app should have an AnimatedTheme")
		}
		none := animation.AnimationStyleNoAnimation
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{ThemeAnimationStyle: &none, Home: widgets.NewSizedBox(nil)}))
		if ts.Count(gluttertest.ByType[*AnimatedTheme]()) != 0 {
			t.Error("the AnimatedTheme should be gone")
		}
		if ts.Count(gluttertest.ByType[*Theme]()) == 0 {
			t.Error("the app should have a Theme")
		}
	})
}

// mockScrollBehavior is MockScrollBehavior: it never scrolls.
type mockScrollBehavior struct{ widgets.ScrollBehaviorBase }

func newMockScrollBehavior() *mockScrollBehavior {
	b := &mockScrollBehavior{}
	widgets.InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func (b *mockScrollBehavior) GetScrollPhysics(widgets.BuildContext) widgets.ScrollPhysics {
	return widgets.NewNeverScrollableScrollPhysics(nil)
}

func (b *mockScrollBehavior) String() string { return "mockScrollBehavior" }

func TestMaterialAppScrollBehavior(t *testing.T) {
	capture := func(ts *gluttertest.Tester, opts MaterialAppOpts) (behavior widgets.ScrollBehavior, bc widgets.BuildContext) {
		opts.Home = widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
			bc = c
			behavior = widgets.ScrollConfigurationOf(c)
			return widgets.NewPlaceholder(nil)
		})
		ts.PumpWidget(NewMaterialApp(opts))
		return behavior, bc
	}

	t.Run("the default is the MaterialScrollBehavior", func(t *testing.T) {
		ts := gluttertest.New(t)
		behavior, bc := capture(ts, MaterialAppOpts{})
		if _, ok := behavior.(*MaterialScrollBehavior); !ok {
			t.Fatalf("behavior is %T", behavior)
		}
		if got := behavior.GetKeyboardDismissBehavior(bc); got != widgets.ScrollViewKeyboardDismissBehaviorManual {
			t.Errorf("keyboard dismiss behavior = %v, want manual", got)
		}
	})

	t.Run("the keyboard dismiss behavior can be overridden", func(t *testing.T) {
		ts := gluttertest.New(t)
		custom := NewMaterialScrollBehavior().CopyWith(widgets.ScrollBehaviorCopy{KeyboardDismissBehavior: widgets.ScrollViewKeyboardDismissBehaviorOnDrag})
		behavior, bc := capture(ts, MaterialAppOpts{ScrollBehavior: custom})
		if got := behavior.GetKeyboardDismissBehavior(bc); got != widgets.ScrollViewKeyboardDismissBehaviorOnDrag {
			t.Errorf("keyboard dismiss behavior = %v, want onDrag", got)
		}
	})

	t.Run("a scroll behavior can be set", func(t *testing.T) {
		ts := gluttertest.New(t)
		behavior, bc := capture(ts, MaterialAppOpts{ScrollBehavior: newMockScrollBehavior()})
		if _, ok := behavior.(*mockScrollBehavior); !ok {
			t.Fatalf("behavior is %T", behavior)
		}
		if _, ok := behavior.GetScrollPhysics(bc).(*widgets.NeverScrollableScrollPhysics); !ok {
			t.Errorf("physics are %T", behavior.GetScrollPhysics(bc))
		}
	})

	t.Run("the platform is the one of the theme", func(t *testing.T) {
		tests := []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformAndroid}
		for _, platform := range tests {
			ts := gluttertest.New(t)
			behavior, bc := capture(ts, MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform)})})
			if got := behavior.GetPlatform(bc); got != platform {
				t.Errorf("platform = %v, want %v", got, platform)
			}
		}
	})

	t.Run("a horizontal scrollable has no scrollbar and a vertical one has on the desktop", func(t *testing.T) {
		ts := gluttertest.New(t)
		_, bc := capture(ts, MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)})})
		child := widgets.NewSizedBox(nil)
		behavior := NewMaterialScrollBehavior()
		controller := widgets.NewScrollController()
		defer controller.Dispose()
		horizontal := widgets.NewScrollableDetailsHorizontal(widgets.ScrollableDetailsOpts{Controller: controller})
		if behavior.BuildScrollbar(bc, child, horizontal) != child {
			t.Error("a horizontal scrollable should have no scrollbar")
		}
		vertical := widgets.NewScrollableDetailsVertical(widgets.ScrollableDetailsOpts{Controller: controller})
		if _, ok := behavior.BuildScrollbar(bc, child, vertical).(*Scrollbar); !ok {
			t.Errorf("a vertical scrollable should have a scrollbar, got %T", behavior.BuildScrollbar(bc, child, vertical))
		}
	})
}

func TestMaterialAppHeroController(t *testing.T) {
	controller := MaterialAppCreateMaterialHeroController()
	defer controller.Dispose()
	begin := ui.NewRectFromLTRB(0.0, 0.0, 10.0, 10.0)
	end := ui.NewRectFromLTRB(0.0, 0.0, 20.0, 20.0)
	tween := controller.CreateRectTween()(foundation.Some(begin), foundation.Some(end))
	// The flight follows arcs: its middle is not the linear middle.
	middle := tween.Transform(0.5).Value()
	if linear := begin.Lerp(end, 0.5); middle == linear {
		t.Errorf("the rectangle at 0.5 is the linear one, %v", middle)
	}
	if tween.Transform(0).Value() != begin || tween.Transform(1).Value() != end {
		t.Errorf("the ends are %v and %v", tween.Transform(0).Value(), tween.Transform(1).Value())
	}
}

func TestMaterialAppSelectionStyle(t *testing.T) {
	ts := gluttertest.New(t)
	theme := NewThemeData(ThemeDataOpts{ColorSchemeSeed: foundation.Some(Colors.Green.Primary)})
	var style *widgets.DefaultSelectionStyle
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		style = widgets.DefaultSelectionStyleOf(bc)
		return widgets.NewSizedBox(nil)
	})}))
	if got, want := style.SelectionColor().Value(), theme.ColorScheme.Primary().WithOpacity(0.40); got != want {
		t.Errorf("selection color = %v, want %v", got, want)
	}
	if got, want := style.CursorColor().Value(), theme.ColorScheme.Primary(); got != want {
		t.Errorf("cursor color = %v, want %v", got, want)
	}
}

func TestMaterialAppColorOfTheApp(t *testing.T) {
	tests := []struct {
		name string
		opts MaterialAppOpts
		want ui.Color
	}{
		{"the default is blue", MaterialAppOpts{}, Colors.Blue.Primary},
		{"the primary color of the theme", MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(Colors.Green.Primary)})}, Colors.Green.Primary},
		{"the color of the app", MaterialAppOpts{Color: foundation.Some(Colors.Red.Primary), Theme: NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(Colors.Green.Primary)})}, Colors.Red.Primary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			tt.opts.Home = widgets.NewSizedBox(nil)
			ts.PumpWidget(NewMaterialApp(tt.opts))
			app := ts.Widget(gluttertest.ByType[*widgets.WidgetsApp]()).(*widgets.WidgetsApp)
			if app.Color() != tt.want {
				t.Errorf("color = %v, want %v", app.Color(), tt.want)
			}
		})
	}
}

func TestMaterialAppErrors(t *testing.T) {
	skipUnlessDebug(t)
	err := recoverError(func() { NewMaterialAppRouter[*routeInfo]() })
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("MaterialAppRouterMissing") {
		t.Errorf("unexpected error %v", err)
	}
}

// foregroundCircleColors are the colors of the circles that the foreground
// painter of the CustomPaint draws.
func foregroundCircleColors(t *testing.T, ts *gluttertest.Tester, f gluttertest.Finder) []ui.Color {
	t.Helper()
	renderObject, ok := ts.RenderObject(f).(*rendering.RenderCustomPaint)
	if !ok {
		t.Fatalf("%s is not a CustomPaint", f.Describe())
	}
	recorder := ui.NewPictureRecorder()
	canvas := ui.NewCanvas(recorder)
	renderObject.ForegroundPainter().Paint(canvas, ts.Size(f))
	var colors []ui.Color
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		if circle, ok := op.(ui.OpDrawCircle); ok {
			colors = append(colors, circle.Paint.Color())
		}
	}
	return colors
}

func TestMaterial2MaterialAppProvidesDefaultOverscrollColor(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	slowDrag := func(start, offset ui.Offset) {
		gesture := ts.StartGesture(start)
		for index := 0; index < 10; index++ {
			gesture.MoveBy(offset)
			ts.Pump(20 * time.Millisecond)
		}
		gesture.Up()
	}

	// The overscroll color should be a transparent version of the colorScheme's
	// secondary color.
	secondaryColor := ui.NewColor(0xff008800)
	glowSecondaryColor := secondaryColor.WithOpacity(0.05)
	theme := NewThemeDataFrom(NewColorSchemeLight().CopyWith(ColorSchemeCopy{Secondary: foundation.Some(secondaryColor)}), nil, foundation.Some(false))
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
		Theme: theme,
		Home:  widgets.NewSingleChildScrollView(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)})),
	}))

	// The Dart test paints the first CustomPaint of the tree, the one of the debug banner, with its
	// descendants; the Go test reads the painter of the indicator.
	painter := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*widgets.GlowingOverscrollIndicator](), gluttertest.ByType[*widgets.CustomPaint]()))
	slowDrag(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	if colors := foregroundCircleColors(t, ts, painter); len(colors) != 1 || colors[0] != glowSecondaryColor {
		t.Errorf("the circles are %v, want one of %v", colors, glowSecondaryColor)
	}
}

func TestMaterialScrollBehaviorBuildsTheOverscrollIndicatorOfAndroid(t *testing.T) {
	list := func() widgets.Widget {
		return widgets.NewListView([]widgets.Widget{
			widgets.NewSizedBox(widgets.NewText("Test"), widgets.SizedBoxOpts{Height: foundation.Some(1000.0), Width: foundation.Some(1000.0)})})
	}
	tests := []struct {
		name        string
		app         func() MaterialAppOpts // the theme is made after the platform is set
		wantStretch bool
		wantGlow    bool
	}{
		{"Material2 - ScrollBehavior default android overscroll indicator", func() MaterialAppOpts {
			return MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)}), ScrollBehavior: NewMaterialScrollBehavior()}
		}, false, true},
		{"Material3 - ScrollBehavior default android overscroll indicator", func() MaterialAppOpts {
			return MaterialAppOpts{ScrollBehavior: NewMaterialScrollBehavior()}
		}, true, false},
		{"MaterialScrollBehavior default stretch android overscroll indicator", func() MaterialAppOpts { return MaterialAppOpts{} }, true, false},
		{"Overscroll indicator can be set by theme", func() MaterialAppOpts { // The current default is M3 and stretch overscroll, setting via the theme should override.
			return MaterialAppOpts{Theme: NewThemeData().CopyWith(ThemeDataCopy{UseMaterial3: foundation.Some(false)})}
		}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t)
			app := tt.app()
			app.Home = list()
			ts.PumpWidget(NewMaterialApp(app))

			if got := ts.Count(gluttertest.ByType[*widgets.StretchingOverscrollIndicator]()); (got == 1) != tt.wantStretch || got > 1 {
				t.Errorf("%d StretchingOverscrollIndicators, want stretch %v", got, tt.wantStretch)
			}
			if got := ts.Count(gluttertest.ByType[*widgets.GlowingOverscrollIndicator]()); (got == 1) != tt.wantGlow || got > 1 {
				t.Errorf("%d GlowingOverscrollIndicators, want glow %v", got, tt.wantGlow)
			}
		})
	}
}

func TestMaterialScrollBehaviorOverscrollIndicatorOnOtherPlatforms(t *testing.T) {
	tests := []struct {
		platform foundation.TargetPlatform
		want     string
	}{
		{foundation.TargetPlatformAndroid, "stretch"},
		{foundation.TargetPlatformFuchsia, "glow"},
		{foundation.TargetPlatformIOS, "none"},
		{foundation.TargetPlatformLinux, "none"},
		{foundation.TargetPlatformMacOS, "none"},
		{foundation.TargetPlatformWindows, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			ts := gluttertest.New(t)
			controller := widgets.NewScrollController()
			defer controller.Dispose()
			theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(tt.platform)})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
				Theme: theme,
				Home: widgets.NewListView([]widgets.Widget{
					widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(1000.0)})}, widgets.ListViewOpts{Controller: controller}),
			}))
			got := "none"
			switch {
			case ts.Exists(gluttertest.ByType[*widgets.StretchingOverscrollIndicator]()):
				got = "stretch"
			case ts.Exists(gluttertest.ByType[*widgets.GlowingOverscrollIndicator]()):
				got = "glow"
			}
			if got != tt.want {
				t.Errorf("the overscroll indicator is %q, want %q", got, tt.want)
			}
			for _, glow := range ts.Widgets(gluttertest.ByType[*widgets.GlowingOverscrollIndicator]()) {
				if got := glow.(*widgets.GlowingOverscrollIndicator).Color(); got != theme.ColorScheme.Secondary() {
					t.Errorf("the glow color is %v, want the secondary color of the theme", got)
				}
			}
		})
	}
}

func TestMaterial3ListViewClipBehaviorUpdatesOverscrollIndicatorClipBehavior(t *testing.T) {
	useAndroid(t)
	buildFrame := func(clipBehavior ui.Clip) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: widgets.NewColumn([]widgets.Widget{
			widgets.NewSizedBox(widgets.NewListViewBuilder(func(_ widgets.BuildContext, index int) widgets.Widget {
				return widgets.NewPadding(painting.NewEdgeInsetsAll(10.0), widgets.NewText("Index "+strconv.Itoa(index)))
			}, widgets.ListViewOpts{ItemCount: foundation.Some(20), ClipBehavior: clipBehavior}), widgets.SizedBoxOpts{Height: foundation.Some(300.0)}),
			widgets.NewOpacity(0.5, widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(ui.NewColor(0xD0FF0000)), Height: foundation.Some(100.0)})),
		})})
	}
	firstClip := func(ts *gluttertest.Tester) *rendering.RenderClipRect {
		for _, e := range ts.Elements(gluttertest.Everything()) {
			if r, ok := e.RenderObject().(*rendering.RenderClipRect); ok {
				return r
			}
		}
		t.Fatal("no RenderClipRect")
		return nil
	}
	tests := []struct {
		name  string
		clip  ui.Clip
		after ui.Clip
	}{
		{"the default clip behavior", ui.ClipHardEdge, ui.ClipHardEdge},
		{"a custom clip behavior", ui.ClipNone, ui.ClipNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(buildFrame(tt.clip))
			if ts.Count(gluttertest.ByType[*widgets.StretchingOverscrollIndicator]()) != 1 || ts.Exists(gluttertest.ByType[*widgets.GlowingOverscrollIndicator]()) {
				t.Fatal("expected the stretching indicator alone")
			}
			index1 := gluttertest.ByText("Index 1")
			if !ts.Exists(index1) {
				t.Fatal("Index 1 is missing")
			}
			// Currently not clipping
			if got := firstClip(ts).ClipBehavior(); got != ui.ClipNone {
				t.Errorf("clip behavior = %v, want Clip.none", got)
			}

			gesture := ts.StartGesture(ts.Center(index1))
			// Overscroll the start.
			gesture.MoveBy(ui.Offset{Dy: 200})
			ts.PumpAndSettle()
			if !ts.Exists(index1) || ts.Center(index1).Dy <= 0 {
				t.Errorf("Index 1 is at %v", ts.Center(index1))
			}
			// Now clipping
			if got := firstClip(ts).ClipBehavior(); got != tt.after {
				t.Errorf("clip behavior = %v, want %v", got, tt.after)
			}

			gesture.Up()
			ts.PumpAndSettle()
		})
	}
}
