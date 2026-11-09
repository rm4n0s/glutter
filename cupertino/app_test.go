package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/app_test.dart. Not ported: 'Heroes work'
// (needs CupertinoButton, step 34.4), the cursor color case (needs the text field of step 34.5), the cases
// of CupertinoApp.router that test the Router itself (material's tests cover that code), and the restoration.

func TestCupertinoAppHasDefaultLocalizations(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		l := cupertino.CupertinoLocalizationsOf(bc)
		return widgets.NewColumn([]widgets.Widget{
			widgets.NewText(l.SelectAllButtonLabel()),
			widgets.NewText(l.DatePickerMediumDate(time.Date(2018, 10, 4, 0, 0, 0, 0, time.UTC))),
		})
	})}))
	for _, want := range []string{"Select All", "Thu Oct 4 "} {
		if n := ts.Count(gluttertest.ByText(want)); n != 1 {
			t.Errorf("%q found %d times, want 1", want, n)
		}
	}
}

func TestCupertinoAppCanUseDynamicColor(t *testing.T) {
	dynamic := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF000000), ui.NewColor(0xFF000001))
	for _, tt := range []struct {
		brightness ui.Brightness
		want       ui.Color
	}{{ui.BrightnessLight, ui.NewColor(0xFF000000)}, {ui.BrightnessDark, ui.NewColor(0xFF000001)}} {
		ts := gluttertest.New(t)
		ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{
			Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
			Color: dynamic, Home: widgets.NewPlaceholder(nil)}))
		title := ts.Widget(gluttertest.ByType[*widgets.Title]()).(*widgets.Title)
		if title.Color() != tt.want {
			t.Errorf("brightness %v: the color of the title is %v, want %v", tt.brightness, title.Color(), tt.want)
		}
	}
}

func TestCupertinoAppCanCustomizeInitialRoutes(t *testing.T) {
	ts := gluttertest.New(t)
	key := widgets.NewGlobalKey[*widgets.NavigatorState]()
	page := func(label string) widgets.Route {
		return widgets.NewPageRouteBuilder(func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
			return widgets.NewText(label)
		})
	}
	text := func(label string) widgets.WidgetBuilder {
		return func(widgets.BuildContext) widgets.Widget { return widgets.NewText(label) }
	}
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{
		NavigatorKey: key,
		OnGenerateInitialRoutes: func(initialRoute string) []widgets.Route {
			if initialRoute != "/abc" {
				t.Errorf("initial route = %q, want /abc", initialRoute)
			}
			return []widgets.Route{page("non-regular page one"), page("non-regular page two")}
		},
		InitialRoute: "/abc",
		Routes:       map[string]widgets.WidgetBuilder{"/": text("regular page one"), "/abc": text("regular page two")},
	}))
	want := func(one, two, regularOne, regularTwo int) {
		t.Helper()
		for label, n := range map[string]int{"non-regular page one": one, "non-regular page two": two, "regular page one": regularOne, "regular page two": regularTwo} {
			if got := ts.Count(gluttertest.ByText(label)); got != n {
				t.Errorf("%q found %d times, want %d", label, got, n)
			}
		}
	}
	want(0, 1, 0, 0)
	navigator, _ := key.CurrentState()
	navigator.Pop()
	ts.PumpAndSettle()
	want(1, 0, 0, 0)
}

func TestCupertinoAppNavigatorKeyCanBeUpdated(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := widgets.NewGlobalKey[*widgets.NavigatorState]()
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{NavigatorKey: key1, Home: widgets.NewPlaceholder(nil)}))
	if _, ok := key1.CurrentState(); !ok {
		t.Fatal("the first key has no navigator")
	}
	key2 := widgets.NewGlobalKey[*widgets.NavigatorState]()
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{NavigatorKey: key2, Home: widgets.NewPlaceholder(nil)}))
	if _, ok := key2.CurrentState(); !ok {
		t.Error("the second key has no navigator")
	}
	if _, ok := key1.CurrentState(); ok {
		t.Error("the first key still has a navigator")
	}
}

func TestCupertinoAppScrollBehavior(t *testing.T) {
	capture := func(ts *gluttertest.Tester, opts cupertino.CupertinoAppOpts) (behavior widgets.ScrollBehavior, bc widgets.BuildContext) {
		opts.Home = widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
			bc = c
			behavior = widgets.ScrollConfigurationOf(c)
			return widgets.NewPlaceholder(nil)
		})
		ts.PumpWidget(cupertino.NewCupertinoApp(opts))
		return behavior, bc
	}
	t.Run("the default is the CupertinoScrollBehavior", func(t *testing.T) {
		ts := gluttertest.New(t)
		behavior, bc := capture(ts, cupertino.CupertinoAppOpts{})
		if _, ok := behavior.(*cupertino.CupertinoScrollBehavior); !ok {
			t.Fatalf("behavior is %T", behavior)
		}
		if got := behavior.GetKeyboardDismissBehavior(bc); got != widgets.ScrollViewKeyboardDismissBehaviorManual {
			t.Errorf("keyboard dismiss behavior = %v, want manual", got)
		}
		if got := behavior.GetMultitouchDragStrategy(bc); got != gestures.MultitouchDragStrategyAverageBoundaryPointers {
			t.Errorf("multitouch drag strategy = %v", got)
		}
		if _, ok := behavior.GetScrollPhysics(bc).(*widgets.BouncingScrollPhysics); !ok {
			t.Errorf("physics are %T", behavior.GetScrollPhysics(bc))
		}
	})
	t.Run("the keyboard dismiss behavior can be overridden", func(t *testing.T) {
		ts := gluttertest.New(t)
		custom := cupertino.NewCupertinoScrollBehavior().CopyWith(widgets.ScrollBehaviorCopy{KeyboardDismissBehavior: widgets.ScrollViewKeyboardDismissBehaviorOnDrag})
		behavior, bc := capture(ts, cupertino.CupertinoAppOpts{ScrollBehavior: custom})
		if got := behavior.GetKeyboardDismissBehavior(bc); got != widgets.ScrollViewKeyboardDismissBehaviorOnDrag {
			t.Errorf("keyboard dismiss behavior = %v, want onDrag", got)
		}
	})
	t.Run("the overscroll indicator is the child", func(t *testing.T) {
		ts := gluttertest.New(t)
		behavior, bc := capture(ts, cupertino.CupertinoAppOpts{})
		child := widgets.NewSizedBox(nil)
		if behavior.BuildOverscrollIndicator(bc, child, widgets.NewScrollableDetailsVertical()) != child {
			t.Error("the overscroll indicator is not the child")
		}
	})
	t.Run("a scrollbar needs a controller on the desktop", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		_, bc := capture(ts, cupertino.CupertinoAppOpts{Theme: cupertino.NewCupertinoThemeData()})
		behavior := cupertino.NewCupertinoScrollBehavior()
		child := widgets.NewSizedBox(nil)
		err := recoverError(func() { behavior.BuildScrollbar(bc, child, widgets.NewScrollableDetailsVertical()) })
		if isDesktop(behavior.GetPlatform(bc)) {
			requireErrorTag(t, err, "CupertinoScrollbarControllerMissing")
		} else if err != nil {
			t.Errorf("unexpected error %v", err)
		}
	})
}

func TestCupertinoAppSystemUIOverlayStyle(t *testing.T) {
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
		platform   ui.Brightness
		want       services.SystemUiOverlayStyle
	}{
		{"the dark style when the background is light", ui.BrightnessLight, ui.BrightnessUnspecified, services.SystemUiOverlayStyleDark},
		{"the light style when the background is dark", ui.BrightnessDark, ui.BrightnessUnspecified, services.SystemUiOverlayStyleLight},
		{"the dark style when the system is light", ui.BrightnessUnspecified, ui.BrightnessLight, services.SystemUiOverlayStyleDark},
		{"the light style when the system is dark", ui.BrightnessUnspecified, ui.BrightnessDark, services.SystemUiOverlayStyleLight},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var chrome *services.SystemChrome
			app := cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					chrome = widgets.WidgetsBindingOf(bc).SystemChrome()
					return widgets.NewPlaceholder(nil)
				})})
			var root widgets.Widget = app
			if tt.platform != ui.BrightnessUnspecified {
				data := widgets.MediaQueryData{PlatformBrightness: tt.platform}
				root = widgets.NewMediaQuery(data, app)
			}
			ts.PumpWidget(root)
			ts.FlushPosted()
			got, ok := chrome.LatestStyle()
			if !ok || got != tt.want {
				t.Errorf("latest style = %+v (%v), want %+v", got, ok, tt.want)
			}
		})
	}
}

func TestCupertinoAppSelectionStyle(t *testing.T) {
	ts := gluttertest.New(t)
	var style *widgets.DefaultSelectionStyle
	theme := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight,
		PrimaryColor: cupertino.CupertinoColors.ActiveOrange})
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{Theme: theme, Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		style = widgets.DefaultSelectionStyleOf(bc)
		return widgets.NewSizedBox(nil)
	})}))
	if got, want := style.CursorColor().Value(), cupertino.CupertinoColors.ActiveOrange.Color; got != want {
		t.Errorf("cursor color = %v, want %v", got, want)
	}
	if got, want := style.SelectionColor().Value(), cupertino.CupertinoColors.ActiveOrange.Color.WithOpacity(0.2); got != want {
		t.Errorf("selection color = %v, want %v", got, want)
	}
}

func TestCupertinoAppDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{Home: widgets.NewText("X")}))))
	if size := ts.Size(gluttertest.ByType[*cupertino.CupertinoApp]()); size != (ui.Size{}) {
		t.Errorf("size = %v, want zero", size)
	}
}

func TestCupertinoAppRouterNeedsADelegateOrAConfig(t *testing.T) {
	skipUnlessDebug(t)
	err := recoverError(func() { cupertino.NewCupertinoAppRouter[int]() })
	requireErrorTag(t, err, "CupertinoAppRouterMissing")
}

func isDesktop(p foundation.TargetPlatform) bool {
	return p == foundation.TargetPlatformLinux || p == foundation.TargetPlatformMacOS || p == foundation.TargetPlatformWindows
}
