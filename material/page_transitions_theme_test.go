package material_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/material/page_transitions_theme_test.dart.
// There are no Cupertino builders and no snapshots in Glutter, so the cases about
// them are not ported; the tests of the 800 ms of FadeForwardsPageTransitionsBuilder
// ('default duration is 800ms' and 'Animation duration changes accordingly when
// page transition builder changes') are not ported either, as the pinned Flutter
// source has 450 ms (kTransitionMilliseconds) and the tests disagree with it. The
// predictive back cases ('predictive back is the default on Android', 'predictive
// back falls back to FadeForwardsPageTransition', 'predictive back gestures pop the
// route on all platforms...' and the Android default of 'Default PageTransitionsTheme
// platform') are in TestPageTransitionsThemeAndroid; the ones that Flutter runs for
// TargetPlatformVariant.all() run on Android and on Linux.

type vsync struct{ s *scheduler.SchedulerBinding }

func (v vsync) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	return scheduler.NewTicker(v.s, onTick)
}

// transitionWidget finds the widget of the unexported type of the transition.
func transitionWidget(typeName string) gluttertest.Finder {
	return gluttertest.ByPredicate(typeName, func(w widgets.Widget) bool {
		return strings.HasSuffix(fmt.Sprintf("%T", w), "."+typeName)
	})
}

// buildersApp is an app with two routes, "/" with a push button and "/b" with a
// pop button and the text "page b", whose page transitions are the builders on
// the platform of the test (Linux).
func buildersApp(builder widgets.PageTransitionsBuilder, surface foundation.Opt[ui.Color]) widgets.Widget {
	theme := NewThemeData(ThemeDataOpts{
		Platform:             foundation.Some(foundation.TargetPlatformLinux),
		PageTransitionsTheme: NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{foundation.TargetPlatformLinux: builder}}),
	})
	if c, ok := surface.Get(); ok {
		theme = theme.CopyWith(ThemeDataCopy{ColorScheme: foundation.Some(theme.ColorScheme.CopyWith(ColorSchemeCopy{Surface: foundation.Some(c)}))})
	}
	return NewMaterialApp(MaterialAppOpts{
		Theme: theme,
		Routes: map[string]widgets.WidgetBuilder{
			"/": func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("push", func() { widgets.NavigatorPushNamed(bc, "/b") }))
			},
			"/b": func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(widgets.NewColumn([]widgets.Widget{
					tapText("pop", func() { widgets.NavigatorPop(bc) }),
					widgets.NewText("page b"),
				}))
			},
		},
	})
}

// boxColors are the colors of the ColoredBoxes of the tree, the lower route first.
func boxColors(ts *gluttertest.Tester) []ui.Color {
	var out []ui.Color
	for _, w := range ts.Widgets(gluttertest.ByType[*widgets.ColoredBox]()) {
		out = append(out, w.(*widgets.ColoredBox).Color())
	}
	return out
}

func hasColor(colors []ui.Color, target ui.Color) bool {
	for _, c := range colors {
		if c == target {
			return true
		}
	}
	return false
}

func TestPageTransitionsTheme(t *testing.T) {
	t.Run("the default theme has the builders of the desktop and of Android", func(t *testing.T) {
		theme := NewPageTransitionsTheme()
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformWindows} {
			if _, ok := theme.Builders()[platform].(ZoomPageTransitionsBuilder); !ok {
				t.Errorf("the builder of %v is %T", platform, theme.Builders()[platform])
			}
		}
		if _, ok := theme.Builders()[foundation.TargetPlatformAndroid].(PredictiveBackPageTransitionsBuilder); !ok {
			t.Errorf("the builder of android is %T", theme.Builders()[foundation.TargetPlatformAndroid])
		}
		if theme.Builders()[foundation.TargetPlatformFuchsia] != nil {
			t.Error("fuchsia should have no builder")
		}
	})

	t.Run("the app has the default theme", func(t *testing.T) {
		ts := gluttertest.New(t)
		var got *PageTransitionsTheme
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			got = ThemeOf(bc).PageTransitionsTheme
			return widgets.NewText("home")
		})}))
		if got == nil || got.Builders()[foundation.TargetPlatformLinux] == nil {
			t.Errorf("the page transitions theme is %v", got)
		}
	})

	t.Run("a platform without a builder uses the zoom transition", func(t *testing.T) {
		theme := NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{}})
		if theme.DelegatedTransition(foundation.TargetPlatformMacOS) == nil {
			t.Error("the zoom transition has a delegated transition")
		}
	})

	t.Run("equality", func(t *testing.T) {
		a := NewPageTransitionsTheme()
		b := NewPageTransitionsTheme()
		c := NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
			foundation.TargetPlatformLinux: NewFadeUpwardsPageTransitionsBuilder()}})
		d := NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
			foundation.TargetPlatformLinux:   NewZoomPageTransitionsBuilder(ZoomPageTransitionsBuilderOpts{BackgroundColor: foundation.Some(Colors.Pink.Primary)}),
			foundation.TargetPlatformWindows: NewZoomPageTransitionsBuilder()}})
		if !a.Equal(b) || a.Equal(c) || a.Equal(d) || a.Equal(nil) || !(*PageTransitionsTheme)(nil).Equal(nil) {
			t.Errorf("a==b %v, a==c %v, a==d %v", a.Equal(b), a.Equal(c), a.Equal(d))
		}
	})

	t.Run("the theme data keeps it", func(t *testing.T) {
		custom := NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
			foundation.TargetPlatformLinux: NewOpenUpwardsPageTransitionsBuilder()}})
		a := NewThemeData(ThemeDataOpts{PageTransitionsTheme: custom})
		b := NewThemeData()
		if a.PageTransitionsTheme != custom || a.Equal(b) {
			t.Error("the theme data should hold the theme and differ from the default one")
		}
		if copy := b.CopyWith(ThemeDataCopy{PageTransitionsTheme: foundation.Some(custom)}); copy.PageTransitionsTheme != custom {
			t.Error("CopyWith should set the theme")
		}
		if ThemeDataLerp(a, b, 0.4).PageTransitionsTheme != custom || ThemeDataLerp(a, b, 0.6).PageTransitionsTheme != b.PageTransitionsTheme {
			t.Error("the lerp should pick the theme of the nearer end")
		}
	})
}

func TestFadeForwardsPageTransitionsBuilder(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build widgets.PageTransitionsBuilder
		want  ui.Color
	}{
		{"the default background color is the surface color", NewFadeForwardsPageTransitionsBuilder(), Colors.Pink.Primary},
		{"the background color can be overridden", NewFadeForwardsPageTransitionsBuilder(FadeForwardsPageTransitionsBuilderOpts{BackgroundColor: foundation.Some(Colors.LightGreen.Primary)}), Colors.LightGreen.Primary},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(buildersApp(tt.build, foundation.Some(Colors.Pink.Primary)))
			if ts.Count(transitionWidget("fadeForwardsPageTransition")) != 1 {
				t.Fatal("the page should have the fade forwards transition")
			}
			ts.Tap(gluttertest.ByText("push"))
			ts.Pump()
			ts.Pump(400 * time.Millisecond)
			if got := boxColors(ts); !hasColor(got, tt.want) {
				t.Errorf("colors = %v, want one of %v", got, tt.want)
			}
			ts.PumpAndSettle()
			if ts.Count(gluttertest.ByText("page b")) != 1 || ts.Count(transitionWidget("fadeForwardsPageTransition")) != 1 {
				t.Error("page b should be shown with the transition")
			}
		})
	}

	t.Run("the duration is 450 milliseconds", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(buildersApp(NewFadeForwardsPageTransitionsBuilder(), foundation.None[ui.Color]()))
		ts.Tap(gluttertest.ByText("push"))
		ts.Pump()
		ts.Pump(449 * time.Millisecond)
		// The page below has a scrim of the surface color while it is animating.
		surface := ThemeOf(ts.Element(gluttertest.ByText("push"))).ColorScheme.Surface()
		if !hasColor(boxColors(ts), surface) {
			t.Errorf("the page below should still be animating: %v", boxColors(ts))
		}
		ts.Pump(2 * time.Millisecond)
		ts.Pump()
		if hasColor(boxColors(ts), surface) {
			t.Errorf("the page below should be at rest: %v", boxColors(ts))
		}
		if ts.Count(gluttertest.ByText("page b")) != 1 {
			t.Error("page b should be shown")
		}
	})

	t.Run("opacity of the delegated transition follows the secondary animation", func(t *testing.T) {
		tests := []struct {
			name      string
			secondary func(c *animation.AnimationController) animation.Animation[float64]
			steps     []struct{ value, opacity float64 }
		}{
			{"it fades out during the forward secondary animation",
				func(c *animation.AnimationController) animation.Animation[float64] {
					return animation.NewTween(0.0, 1.0).Animate(c)
				}, []struct{ value, opacity float64 }{{0.0, 1.0}, {0.25, 0.0}}},
			{"it fades in during the reverse secondary animation",
				func(c *animation.AnimationController) animation.Animation[float64] {
					return animation.NewTween(1.0, 0.0).Animate(c)
				}, []struct{ value, opacity float64 }{{0.75, 0.0}, {1.0, 1.0}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				controller := animation.NewAnimationController(vsync{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{Duration: foundation.Some(100 * time.Millisecond)})
				defer controller.Dispose()
				secondary := tt.secondary(controller)
				primary := animation.NewTween(1.0, 0.0).Animate(controller)
				if tt.name[3] == 'f' && strings.Contains(tt.name, "in") {
					primary = animation.NewTween(0.0, 1.0).Animate(controller)
				}
				ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					return NewFadeForwardsPageTransitionsBuilder().DelegatedTransition()(bc, primary, secondary, false, widgets.NewSizedBox(nil))
				}))))
				for _, step := range tt.steps {
					controller.SetValue(step.value)
					ts.Pump()
					var opacity float64 = -1
					for _, w := range ts.Widgets(gluttertest.Ancestor(gluttertest.ByType[*widgets.SizedBox](), gluttertest.ByType[*widgets.FadeTransition]())) {
						opacity = w.(*widgets.FadeTransition).Opacity().Value()
						break
					}
					if opacity != step.opacity {
						t.Errorf("at %v the opacity is %v, want %v", step.value, opacity, step.opacity)
					}
				}
			})
		}
	})

	t.Run("does not use a ColoredBox for a route that is not opaque", func(t *testing.T) {
		ts := gluttertest.New(t)
		lightGreen := Colors.LightGreen.Primary
		theme := NewThemeData(ThemeDataOpts{
			Platform: foundation.Some(foundation.TargetPlatformLinux),
			PageTransitionsTheme: NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
				foundation.TargetPlatformLinux: NewFadeForwardsPageTransitionsBuilder(FadeForwardsPageTransitionsBuilderOpts{BackgroundColor: foundation.Some(lightGreen)})}}),
		})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return NewMaterial(tapText("push a", func() {
				widgets.NavigatorPush(bc, widgets.NewPageRouteBuilder(func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
					return NewMaterial(tapText("push b", func() {
						widgets.NavigatorPush(bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("page b") }))
					}))
				}, widgets.PageRouteBuilderOpts{NotOpaque: true}))
			}))
		})}))
		hasLightGreen := func() bool {
			return ts.Count(gluttertest.ByPredicate("light green", func(w widgets.Widget) bool {
				box, ok := w.(*widgets.ColoredBox)
				return ok && box.Color() == lightGreen
			})) > 0
		}
		ts.Tap(gluttertest.ByText("push a"))
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByText("push b"))
		ts.Pump()
		ts.Pump(400 * time.Millisecond)
		if hasLightGreen() {
			t.Error("a route that is not opaque should have no ColoredBox")
		}
		ts.PumpAndSettle()
		widgets.NavigatorPop(ts.Element(gluttertest.ByText("page b")))
		ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 400 * time.Millisecond})
		if hasLightGreen() {
			t.Error("a route that is not opaque should have no ColoredBox after the pop")
		}
	})
}

func TestFadeUpwardsPageTransitionsBuilder(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(buildersApp(NewFadeUpwardsPageTransitionsBuilder(), foundation.None[ui.Color]()))
	if ts.Count(transitionWidget("fadeUpwardsPageTransition")) != 1 {
		t.Fatal("the page should have the fade upwards transition")
	}
	ts.Tap(gluttertest.ByText("push"))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("page b")) != 1 || ts.Count(transitionWidget("fadeUpwardsPageTransition")) != 1 {
		t.Fatal("page b should be shown with the transition")
	}
	ts.Tap(gluttertest.ByText("pop"))
	ts.Pump()
	ts.Pump(299 * time.Millisecond)
	if ts.Count(gluttertest.ByText("page b")) != 1 {
		t.Error("page b should still be there")
	}
	if ts.Count(gluttertest.ByType[*widgets.ColoredBox]()) != 0 {
		t.Error("the fade upwards transition has no ColoredBox")
	}
	ts.Pump(2 * time.Millisecond)
	ts.Pump()
	if ts.Count(gluttertest.ByText("page b")) != 0 || ts.Count(gluttertest.ByText("push")) != 1 {
		t.Error("the first page should be back")
	}
}

func TestOpenUpwardsPageTransitionsBuilder(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(buildersApp(NewOpenUpwardsPageTransitionsBuilder(), foundation.None[ui.Color]()))
	ts.Tap(gluttertest.ByText("push"))
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	// The page opens from the bottom edge: it is clipped by a box that grows.
	clip := ts.Size(gluttertest.Last(gluttertest.ByPredicate("the box that clips the page", func(w widgets.Widget) bool {
		box, ok := w.(*widgets.SizedBox)
		if !ok {
			return false
		}
		_, overflow := box.Child().(*widgets.OverflowBox)
		return overflow
	})))
	if clip.Height <= 0 || clip.Height >= 600 {
		t.Errorf("the clip is %v high at 100 ms, it should be growing to 600", clip.Height)
	}
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("page b")) != 1 {
		t.Fatal("page b should be shown")
	}
	ts.Tap(gluttertest.ByText("pop"))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("page b")) != 0 || ts.Count(gluttertest.ByText("push")) != 1 {
		t.Error("the first page should be back")
	}
}

func TestZoomPageTransitionsBuilder(t *testing.T) {
	scrims := func(ts *gluttertest.Tester) []ui.Color {
		var out []ui.Color
		for _, w := range ts.Widgets(gluttertest.ByType[*widgets.ColoredBox]()) {
			out = append(out, w.(*widgets.ColoredBox).Color())
		}
		return out
	}
	contains := func(colors []ui.Color, target ui.Color) bool {
		for _, c := range colors {
			if c.Red() == target.Red() && c.Green() == target.Green() && c.Blue() == target.Blue() && c.Alpha() != 0 {
				return true
			}
		}
		return false
	}

	t.Run("uses the surface color of the theme for the scrim", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(buildersApp(NewZoomPageTransitionsBuilder(), foundation.Some(Colors.Pink.Primary)))
		ts.Tap(gluttertest.ByText("push"))
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if !contains(scrims(ts), Colors.Pink.Primary) {
			t.Errorf("no scrim of the surface color in %v", scrims(ts))
		}
	})

	t.Run("uses the color that the developer gives", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(buildersApp(NewZoomPageTransitionsBuilder(ZoomPageTransitionsBuilderOpts{BackgroundColor: foundation.Some(Colors.LightGreen.Primary)}), foundation.Some(Colors.Pink.Primary)))
		ts.Tap(gluttertest.ByText("push"))
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if !contains(scrims(ts), Colors.LightGreen.Primary) || contains(scrims(ts), Colors.Pink.Primary) {
			t.Errorf("the scrims are %v", scrims(ts))
		}
	})

	t.Run("the page is built once", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("push", func() { widgets.NavigatorPushNamed(bc, "/b") }))
			}),
			Routes: map[string]widgets.WidgetBuilder{"/b": func(widgets.BuildContext) widgets.Widget {
				return widgets.NewBuilder(func(widgets.BuildContext) widgets.Widget { builds++; return widgets.NewText("page b") })
			}},
		}))
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		if builds != 1 {
			t.Errorf("page b was built %d times, want 1", builds)
		}
	})

	t.Run("the page at rest is painted as it is", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(buildersApp(NewZoomPageTransitionsBuilder(), foundation.None[ui.Color]()))
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		before := ts.TopLeft(gluttertest.ByText("page b"))
		ts.Pump(time.Second)
		if after := ts.TopLeft(gluttertest.ByText("page b")); after != before {
			t.Errorf("page b moved from %v to %v at rest", before, after)
		}
	})
}

func TestIncomingRouteReceivesInputDuringFadeForwardsPop(t *testing.T) {
	ts := gluttertest.New(t)
	taps := 0
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
		Theme: NewThemeData(ThemeDataOpts{
			Platform: foundation.Some(foundation.TargetPlatformLinux),
			PageTransitionsTheme: NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
				foundation.TargetPlatformLinux: NewFadeForwardsPageTransitionsBuilder()}}),
		}),
		Routes: map[string]widgets.WidgetBuilder{
			"/": func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(widgets.NewColumn([]widgets.Widget{
					tapText("push", func() { widgets.NavigatorPushNamed(bc, "/b") }),
					tapText("count", func() { taps++ }),
				}))
			},
			"/b": func(bc widgets.BuildContext) widgets.Widget {
				return NewMaterial(tapText("pop", func() { widgets.NavigatorPop(bc) }))
			},
		},
	}))
	ts.Tap(gluttertest.ByText("push"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("pop"))
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	// The page that comes back takes the taps while the other one leaves.
	ts.Tap(gluttertest.ByText("count"))
	if taps != 1 {
		t.Errorf("the page that comes back got %d taps, want 1", taps)
	}
	ts.PumpAndSettle()
}

func TestPageTransitionsThemeAndroid(t *testing.T) {
	predictive := transitionWidget("predictiveBackSharedElementPageTransition")
	fadeForwards := transitionWidget("fadeForwardsPageTransition")
	pushB := func() map[string]widgets.WidgetBuilder { return pushBRoutes() }

	// Default PageTransitionsTheme platform. Deviation: iOS and macOS have no
	// builder, as there is no Cupertino library.
	t.Run("Default PageTransitionsTheme platform", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewText("home")}))
		theme := ThemeOf(ts.Element(gluttertest.ByText("home"))).PageTransitionsTheme
		for p := foundation.TargetPlatformAndroid; p <= foundation.TargetPlatformWindows; p++ {
			want := p == foundation.TargetPlatformAndroid || p == foundation.TargetPlatformLinux || p == foundation.TargetPlatformWindows
			if got := theme.Builders()[p] != nil; got != want {
				t.Errorf("theme builder for %v is set: %v, want %v", p, got, want)
			}
		}
	})

	t.Run("Default PageTransitionsTheme builds a _FadeForwardsPageTransition for android", func(t *testing.T) {
		withPlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: pushB()}))
		if got := ThemeOf(ts.Element(gluttertest.ByText("push"))).Platform; got != foundation.TargetPlatformAndroid {
			t.Errorf("the theme platform is %v", got)
		}
		if ts.Count(fadeForwards) != 1 {
			t.Errorf("%d fade forwards transitions at the start, want 1", ts.Count(fadeForwards))
		}
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("page b")) != 1 || ts.Count(fadeForwards) != 1 {
			t.Errorf("page b: %d, fade forwards transitions: %d, want 1 and 1", ts.Count(gluttertest.ByText("page b")), ts.Count(fadeForwards))
		}
	})

	t.Run("predictive back gestures pop the route on all platforms regardless of whether their transition handles predictive back", func(t *testing.T) {
		forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: pushB()}))
			ts.Tap(gluttertest.ByText("push"))
			ts.PumpAndSettle()
			if ts.Count(gluttertest.ByText("push")) != 0 || ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Fatal("page b should be shown alone")
			}

			// Start a system pop gesture.
			ts.StartBackGesture(0, backStart)
			ts.Pump()
			if ts.Count(gluttertest.ByText("push")) != 0 || ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Error("the start of the gesture should show page b alone")
			}

			// Drag the system back gesture far enough to commit.
			ts.UpdateBackGestureProgress(0.35)
			ts.PumpAndSettle()
			wantPush := 0
			if platform == foundation.TargetPlatformAndroid {
				// Shows both pages while doing the "peek" predictive back transition.
				wantPush = 1
			}
			if got := ts.Count(gluttertest.ByText("push")); got != wantPush {
				t.Errorf("%d 'push' texts while dragging, want %d", got, wantPush)
			}
			if ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Error("page b should still be shown")
			}

			// Commit the system back gesture.
			ts.CommitBackGesture()
			ts.PumpAndSettle()
			if ts.Count(gluttertest.ByText("push")) != 1 || ts.Count(gluttertest.ByText("page b")) != 0 {
				t.Error("the first page should be back")
			}
		})
	})

	t.Run("predictive back is the default on Android", func(t *testing.T) {
		forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: pushB()}))
			builder := ThemeOf(ts.Element(gluttertest.ByText("push"))).PageTransitionsTheme.Builders()[platform]
			_, isPredictive := builder.(PredictiveBackPageTransitionsBuilder)
			if want := platform == foundation.TargetPlatformAndroid; isPredictive != want {
				t.Errorf("the builder of %v is %T, predictive back: %v, want %v", platform, builder, isPredictive, want)
			}
		})
	})

	t.Run("predictive back falls back to FadeForwardsPageTransition", func(t *testing.T) {
		forAndroidAndLinux(t, func(t *testing.T, platform foundation.TargetPlatform) {
			android := platform == foundation.TargetPlatformAndroid
			builders := map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{}
			for p := foundation.TargetPlatformAndroid; p <= foundation.TargetPlatformWindows; p++ {
				builders[p] = NewPredictiveBackPageTransitionsBuilder()
			}
			ts := gluttertest.New(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Routes: pushB(),
				Theme: NewThemeData(ThemeDataOpts{PageTransitionsTheme: NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: builders})})}))
			if _, ok := ThemeOf(ts.Element(gluttertest.ByText("push"))).PageTransitionsTheme.Builders()[platform].(PredictiveBackPageTransitionsBuilder); !ok {
				t.Fatal("the builder should be PredictiveBackPageTransitionsBuilder")
			}
			if ts.Count(gluttertest.ByText("push")) != 1 || ts.Count(gluttertest.ByText("page b")) != 0 {
				t.Fatal("the first page should be shown")
			}

			ts.Tap(gluttertest.ByText("push"))
			ts.PumpAndSettle()
			if ts.Count(gluttertest.ByText("push")) != 0 || ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Fatal("page b should be shown alone")
			}

			// Only Android sends system back gestures.
			if android {
				ts.StartBackGesture(0, backStart)
				ts.Pump()
			}
			wantPredictive, wantFallback := 0, 1
			if android {
				wantPredictive, wantFallback = 1, 0
			}
			expect := func(step string, predictiveN, fallbackN int) {
				t.Helper()
				if got := ts.Count(predictive); got != predictiveN {
					t.Errorf("%s: %d predictive back transitions, want %d", step, got, predictiveN)
				}
				if got := ts.Count(fadeForwards); got != fallbackN {
					t.Errorf("%s: %d fade forwards transitions, want %d", step, got, fallbackN)
				}
			}
			expect("start", wantPredictive, wantFallback)
			if ts.Count(gluttertest.ByText("push")) != 0 || ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Error("page b should be shown alone")
			}

			// Drag the system back gesture far enough to commit.
			if android {
				ts.UpdateBackGestureProgress(0.35)
				ts.PumpAndSettle()
				if ts.Count(gluttertest.ByText("push")) != 1 {
					t.Error("the first page should show below page b")
				}
			} else if ts.Count(gluttertest.ByText("push")) != 0 {
				t.Error("the first page should not show")
			}
			if ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Error("page b should be shown")
			}
			if android {
				expect("update", 2, 0)
			} else {
				expect("update", 0, 1)
			}

			if android {
				// Commit the system back gesture on Android.
				ts.CommitBackGesture()
			} else {
				// On other platforms, send a one-off system pop.
				ts.PopRoute()
			}
			ts.Pump()
			if ts.Count(gluttertest.ByText("push")) != 1 || ts.Count(gluttertest.ByText("page b")) != 1 {
				t.Error("both pages should be shown while the route goes away")
			}
			if android {
				expect("commit", 2, 0)
			} else {
				expect("commit", 0, 2)
			}

			ts.PumpAndSettle()
			if ts.Count(gluttertest.ByText("push")) != 1 || ts.Count(gluttertest.ByText("page b")) != 0 {
				t.Error("the first page should be back")
			}
			expect("settled", 0, 1)
		})
	})
}
