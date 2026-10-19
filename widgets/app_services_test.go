package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from app_title_test.dart, app_navigator_key_test.dart and
// widgets_app_tester_test.dart of packages/flutter/test/widgets. Not ported: the
// ones about the restoration scope and the parameters that default to null (Go
// has no way to ask a WidgetsApp for them), and the one of the no-transition
// duration, which is the one of PageRouteBuilder.

func generatedRoute(settings RouteSettings) Route {
	return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewPlaceholder(nil)
	}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
}

func TestWidgetsAppNavigatorKey(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	color := ui.NewColor(0xFF112233)
	withKey := NewWidgetsApp(color, WidgetsAppOpts{NavigatorKey: key, OnGenerateRoute: generatedRoute})
	withoutKey := NewWidgetsApp(color, WidgetsAppOpts{OnGenerateRoute: generatedRoute})
	for i, tc := range []struct {
		app  Widget
		want bool
	}{{withKey, true}, {withoutKey, false}, {withKey, true}} {
		ts.PumpWidget(tc.app)
		if _, ok := key.CurrentState(); ok != tc.want {
			t.Errorf("step %d: key has a state = %v, want %v", i, ok, tc.want)
		}
	}
}

func TestWidgetsAppTitle(t *testing.T) {
	const titleColor, title = 0xFF333333, "Hello World"
	pump := func(ts *gluttertest.Tester, onGenerateTitle GenerateAppTitle, color ui.Color) {
		ts.PumpWidget(NewWidgetsApp(color, WidgetsAppOpts{
			SupportedLocales: []ui.Locale{ui.NewLocale("en", "US"), ui.NewLocale("en", "GB")},
			Title:            title,
			OnGenerateTitle:  onGenerateTitle,
			OnGenerateRoute: func(settings RouteSettings) Route {
				return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
					return NewContainer(nil)
				})
			},
		}))
	}
	widget := func(ts *gluttertest.Tester) *Title { return ts.Widget(gluttertest.ByType[*Title]()).(*Title) }

	t.Run("the specified title and color are used to build a Title", func(t *testing.T) {
		ts := gluttertest.New(t)
		pump(ts, nil, ui.NewColor(titleColor))
		if widget(ts).Title() != title || widget(ts).Color() != ui.NewColor(titleColor) {
			t.Errorf("title = %q, color = %v", widget(ts).Title(), widget(ts).Color())
		}
	})

	t.Run("the specified color is made opaque for Title", func(t *testing.T) {
		ts := gluttertest.New(t)
		pump(ts, nil, ui.NewColor(0xDD0000FF))
		if got := widget(ts).Color(); got != ui.NewColor(0xFF0000FF) {
			t.Errorf("color = %v, want the opaque blue", got)
		}
	})

	t.Run("onGenerateTitle handles changing locales", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{ui.NewLocale("en", "US")} })
		pump(ts, func(bc BuildContext) string { return LocalizationsLocaleOf(bc).String() }, ui.NewColor(titleColor))
		if widget(ts).Title() != "en_US" || widget(ts).Color() != ui.NewColor(titleColor) {
			t.Fatalf("title = %q, color = %v", widget(ts).Title(), widget(ts).Color())
		}
		setLocale := func(l ui.Locale) {
			ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{l} })
			ts.Pump()
		}
		setLocale(ui.NewLocale("en", "GB"))
		if widget(ts).Title() != "en_GB" {
			t.Errorf("title = %q, want en_GB", widget(ts).Title())
		}
		// Not a supported locale, so the app switches to supportedLocales[0], en_US.
		setLocale(ui.NewLocale("fr", "CA"))
		if widget(ts).Title() != "en_US" || widget(ts).Color() != ui.NewColor(titleColor) {
			t.Errorf("title = %q, color = %v; want en_US", widget(ts).Title(), widget(ts).Color())
		}
	})
}

func tappable(text string, onTap func()) Widget {
	return NewGestureDetector(navTextExt(text), GestureDetectorOpts{OnTap: onTap})
}

func TestWidgetsAppFeatures(t *testing.T) {
	t.Run("home widget is displayed", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: navTextExt("Home Widget")}))
		if ts.Count(gluttertest.ByText("Home Widget")) != 1 {
			t.Error("the home widget should be displayed")
		}
	})

	t.Run("uses the color", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewPlaceholder(nil)}))
		if got := ts.Widget(gluttertest.ByType[*WidgetsApp]()).(*WidgetsApp).Color(); got != ui.NewColor(0xFFFFFFFF) {
			t.Errorf("default color = %v, want white", got)
		}
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewPlaceholder(nil), color: ui.NewColor(0xFF123456)}))
		if got := ts.Widget(gluttertest.ByType[*WidgetsApp]()).(*WidgetsApp).Color(); got != ui.NewColor(0xFF123456) {
			t.Errorf("custom color = %v", got)
		}
	})

	t.Run("provides a working Overlay", func(t *testing.T) {
		ts := gluttertest.New(t)
		var entry *OverlayEntry
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: builderOf(func(bc BuildContext) Widget {
			return tappable("Show Overlay", func() {
				entry = NewOverlayEntry(func(BuildContext) Widget {
					return NewPositioned(navTextExt("Overlay Content"), PositionedOpts{Top: foundation.Some(100.0), Left: foundation.Some(100.0)})
				})
				OverlayOf(bc).Insert(entry)
			})
		})}))
		if ts.Count(gluttertest.ByText("Overlay Content")) != 0 {
			t.Fatal("the overlay content should not be there yet")
		}
		ts.Tap(gluttertest.ByText("Show Overlay"))
		ts.Pump()
		if ts.Count(gluttertest.ByText("Overlay Content")) != 1 {
			t.Fatal("the overlay content should be displayed")
		}
		entry.Remove()
		entry.Dispose()
		ts.Pump()
		if ts.Count(gluttertest.ByText("Overlay Content")) != 0 {
			t.Error("the overlay content should be gone")
		}
	})

	t.Run("provides a working Navigator that can pop routes", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: builderOf(func(bc BuildContext) Widget {
			return tappable("First Page", func() {
				NavigatorOf(bc).Push(NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
					return tappable("Second Page - Tap to go back", func() { NavigatorOf(bc).Pop() })
				}))
			})
		})}))
		ts.Tap(gluttertest.ByText("First Page"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Second Page - Tap to go back")) != 1 || textsFound(ts, "First Page", false) != 0 {
			t.Fatal("the second page should replace the first")
		}
		ts.Tap(gluttertest.ByText("Second Page - Tap to go back"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("First Page")) != 1 || ts.Count(gluttertest.ByText("Second Page - Tap to go back")) != 0 {
			t.Error("the first page should come back")
		}
	})

	t.Run("provides MediaQuery and Directionality", func(t *testing.T) {
		ts := gluttertest.New(t)
		var size ui.Size
		var direction ui.TextDirection
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: builderOf(func(bc BuildContext) Widget {
			size, direction = MediaQueryOf(bc).Size, DirectionalityOf(bc)
			return NewPlaceholder(nil)
		})}))
		if size == (ui.Size{}) || direction != ui.TextDirectionLTR {
			t.Errorf("size = %v, direction = %v", size, direction)
		}
	})

	t.Run("routes can be navigated with pushNamed and popped", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
			home: builderOf(func(bc BuildContext) Widget {
				return tappable("Home Page", func() { NavigatorOf(bc).PushNamed("/details") })
			}),
			routes: map[string]WidgetBuilder{"/details": func(bc BuildContext) Widget {
				return tappable("Details Page - Tap to go back", func() { NavigatorOf(bc).Pop() })
			}},
		}))
		ts.Tap(gluttertest.ByText("Home Page"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Details Page - Tap to go back")) != 1 || textsFound(ts, "Home Page", false) != 0 {
			t.Fatal("the details page should replace the home page")
		}
		ts.Tap(gluttertest.ByText("Details Page - Tap to go back"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Home Page")) != 1 || ts.Count(gluttertest.ByText("Details Page - Tap to go back")) != 0 {
			t.Error("the home page should come back")
		}
	})

	t.Run("initialRoute navigates to the specified route on launch", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
			initialRoute: "/details",
			routes: map[string]WidgetBuilder{
				"/":        func(BuildContext) Widget { return navTextExt("Home Page") },
				"/details": func(BuildContext) Widget { return navTextExt("Details Page") },
			},
		}))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Details Page")) != 1 {
			t.Error("the details page should be displayed")
		}
	})

	t.Run("builder wraps the navigator content", func(t *testing.T) {
		ts := gluttertest.New(t)
		var received Widget
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
			builder: func(_ BuildContext, child Widget) Widget {
				received = child
				return NewColumn([]Widget{navTextExt("Header from builder"), NewExpanded(child)})
			},
			home: navTextExt("Content"),
		}))
		if ts.Count(gluttertest.ByText("Header from builder")) != 1 || ts.Count(gluttertest.ByText("Content")) != 1 {
			t.Error("the header and the content should both be displayed")
		}
		if received == nil {
			t.Error("the builder should get the navigator as its child")
		}
	})

	t.Run("custom shortcuts respond to key events", func(t *testing.T) {
		ts := gluttertest.New(t)
		triggered := false
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
			shortcuts: ShortcutMap{{NewSingleActivator(services.LogicalKeyboardKeyKeyK, SingleActivatorOpts{Control: true}),
				VoidCallbackIntent{Callback: func() { triggered = true }}}},
			actions: ActionMap{IntentTypeOf[VoidCallbackIntent](): NewVoidCallbackAction()},
			home:    NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true}),
		}))
		ts.Pump()
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyControl)
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyK)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyKeyK)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyControl)
		if !triggered {
			t.Error("the custom shortcut should have triggered")
		}
	})

	t.Run("textStyle wraps the tree in DefaultTextStyle", func(t *testing.T) {
		ts := gluttertest.New(t)
		style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(42.0)})
		var got *painting.TextStyle
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: style, home: builderOf(func(bc BuildContext) Widget {
			got = DefaultTextStyleOf(bc).Style()
			return NewPlaceholder(nil)
		})}))
		if got == nil || got.FontSize.OrElse(0) != 42 {
			t.Errorf("default text style = %v, want font size 42", got)
		}
	})

	t.Run("navigatorKey provides access to the NavigatorState", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{navigatorKey: key, home: navTextExt("Home"),
			routes: map[string]WidgetBuilder{"/details": func(BuildContext) Widget { return navTextExt("Details") }}}))
		state, ok := key.CurrentState()
		if !ok {
			t.Fatal("the key should have a state")
		}
		state.PushNamed("/details")
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Details")) != 1 {
			t.Error("the details page should be displayed")
		}
	})

	t.Run("onGenerateRoute builds and pushes routes", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{navigatorKey: key, initialRoute: "/",
			onGenerateRoute: func(settings RouteSettings) Route {
				return defaultTestPageRoute(settings, func(BuildContext) Widget { return navTextExt("Page " + settings.Name()) })
			}}))
		if ts.Count(gluttertest.ByText("Page /")) != 1 {
			t.Fatal("the generated initial route should be displayed")
		}
		state, _ := key.CurrentState()
		state.PushNamed("/next")
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Page /next")) != 1 {
			t.Error("the pushed route should be displayed")
		}
	})
}
