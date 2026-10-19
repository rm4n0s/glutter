package widgets_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/app_test.dart. A
// focusable box that toggles on ActivateIntent stands in for the Material
// Checkbox, and the cases of Apple's text shortcuts (DefaultTextEditingShortcuts
// has only the Linux bindings) and of the predictive back of Android
// (SystemNavigator.SetFrameworkHandlesBack does nothing on the desktop) are not
// ported.

var appColor = ui.NewColor(0xFF123456)

// testCheckbox stands in for Checkbox: it takes the focus, and its ActivateAction
// toggles the value.
func testCheckbox(checked *bool) Widget {
	return NewFocusableActionDetector(box100(nil), FocusableActionDetectorOpts{
		Autofocus: true,
		Actions: ActionMap{IntentTypeOf[ActivateIntent](): NewCallbackAction[ActivateIntent](func(ActivateIntent) any {
			*checked = !*checked
			return nil
		})},
	})
}

func TestWidgetsApp(t *testing.T) {
	t.Run("with builder only", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := containerKey()
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{Key: key,
			Builder: func(BuildContext, Widget) Widget { return NewPlaceholder(nil) }}))
		if ts.Count(gluttertest.ByKey(key)) != 1 {
			t.Error("the app should be in the tree")
		}
	})

	t.Run("default key bindings", func(t *testing.T) {
		ts := gluttertest.New(t)
		checked := false
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
			Builder: func(BuildContext, Widget) Widget { return testCheckbox(&checked) }}))
		ts.Pump() // Wait for the focus to take effect.
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.PumpAndSettle()
		if !checked {
			t.Error("the default key mapping did not work")
		}
	})

	t.Run("can override default key bindings", func(t *testing.T) {
		ts := gluttertest.New(t)
		checked := false
		calls := 0
		action := NewCallbackAction[Intent](func(Intent) any { calls++; return nil })
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
			Actions:   ActionMap{IntentTypeOf[testIntentV](): action},
			Shortcuts: ShortcutMap{{NewSingleActivator(services.LogicalKeyboardKeySpace), testIntentV{}}},
			Builder:   func(BuildContext, Widget) Widget { return testCheckbox(&checked) },
		}))
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.PumpAndSettle()
		if checked {
			t.Error("the default key mapping was invoked")
		}
		if calls != 1 {
			t.Errorf("the overridden mapping was invoked %d times, want 1", calls)
		}
	})

	t.Run("default activation key mappings work", func(t *testing.T) {
		ts := gluttertest.New(t)
		checked := false
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
			Builder: func(BuildContext, Widget) Widget { return testCheckbox(&checked) }}))
		ts.Pump()
		for _, key := range []services.LogicalKeyboardKey{
			services.LogicalKeyboardKeySpace, services.LogicalKeyboardKeyEnter, services.LogicalKeyboardKeyNumpadEnter,
			services.LogicalKeyboardKeyGameButtonA, services.LogicalKeyboardKeySelect,
		} {
			checked = false
			ts.SendKeyEvent(key)
			ts.PumpAndSettle()
			if !checked {
				t.Errorf("%v did not activate", key)
			}
		}
	})

	t.Run("creates a Title", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{Builder: func(BuildContext, Widget) Widget { return NewContainer(nil) }}))
		if ts.Count(gluttertest.ByType[*Title]()) != 1 {
			t.Error("the app should have a Title")
		}
	})

	t.Run("generates the title from the context", func(t *testing.T) {
		ts := gluttertest.New(t)
		var got string
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
			Builder: func(BuildContext, Widget) Widget { return NewContainer(nil) },
			OnGenerateTitle: func(bc BuildContext) string {
				got = "generated"
				if LocalizationsLocaleOf(bc).LanguageCode != "en" {
					got = "no localizations"
				}
				return got
			},
		}))
		if title := ts.Widget(gluttertest.ByType[*Title]()).(*Title).Title(); title != "generated" {
			t.Errorf("title = %q, want generated", title)
		}
	})

	t.Run("can customize initial routes", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		pageRoute := func(text string) Route {
			return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
				return navTextExt(text)
			})
		}
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
			NavigatorKey: key,
			OnGenerateInitialRoutes: func(initialRoute string) []Route {
				if initialRoute != "/abc" {
					t.Errorf("initial route = %q, want /abc", initialRoute)
				}
				return []Route{pageRoute("non-regular page one"), pageRoute("non-regular page two")}
			},
			InitialRoute:    "/abc",
			OnGenerateRoute: func(RouteSettings) Route { return pageRoute("regular page") },
		}))
		for _, text := range []struct {
			s string
			n int
		}{{"non-regular page two", 1}, {"non-regular page one", 0}, {"regular page", 0}} {
			if got := ts.Count(gluttertest.ByText(text.s)); got != text.n {
				t.Fatalf("%q found %d times, want %d", text.s, got, text.n)
			}
		}
		state, _ := key.CurrentState()
		state.Pop()
		ts.PumpAndSettle()
		for _, text := range []struct {
			s string
			n int
		}{{"non-regular page two", 0}, {"non-regular page one", 1}, {"regular page", 0}} {
			if got := ts.Count(gluttertest.ByText(text.s)); got != text.n {
				t.Errorf("%q found %d times, want %d", text.s, got, text.n)
			}
		}
	})

	t.Run("has the default ScrollBehavior", func(t *testing.T) {
		ts := gluttertest.New(t)
		var captured BuildContext
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{Builder: func(bc BuildContext, _ Widget) Widget {
			captured = bc
			return NewPlaceholder(nil)
		}}))
		if got, want := reflect.TypeOf(ScrollConfigurationOf(captured)), reflect.TypeOf(NewScrollBehavior()); got != want {
			t.Errorf("scroll behavior is %v, want %v", got, want)
		}
	})

	t.Run("doesn't have a dependency on MediaQuery", func(t *testing.T) {
		ts := gluttertest.New(t)
		builds := 0
		app := NewWidgetsApp(ui.NewColor(0xFFFFFFFF), WidgetsAppOpts{OnGenerateRoute: func(RouteSettings) Route {
			return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
				builds++
				return NewPlaceholder(nil)
			})
		}})
		for i, scale := range []float64{10, 20} {
			ts.PumpWidget(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{TextScaler: painting.NewTextScalerLinear(scale)}), app))
			if builds != 1 {
				t.Fatalf("pump %d: route built %d times, want 1", i, builds)
			}
		}
	})

	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewCenter(NewSizedBoxShrink(NewWidgetsApp(appColor, WidgetsAppOpts{
			Builder: func(BuildContext, Widget) Widget { return NewText("X", TextOpts{TextDirection: ui.TextDirectionLTR}) }}))))
		if size := ts.Size(gluttertest.ByType[*WidgetsApp]()); size != (ui.Size{}) {
			t.Errorf("size = %v, want zero", size)
		}
	})
}

func TestWidgetsAppUnknownRoutes(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("push unknown route when onUnknownRoute is nil", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{NavigatorKey: key, Home: NewContainer(nil),
			PageRouteBuilder: defaultTestPageRoute, OnGenerateRoute: func(RouteSettings) Route { return nil }}))
		state, _ := key.CurrentState()
		requireTag(t, "WidgetsAppUnknownRouteMissing", func() { state.PushNamed("/path") })
	})
	t.Run("push unknown route when onUnknownRoute returns nil", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{NavigatorKey: key, Home: NewContainer(nil),
			PageRouteBuilder: defaultTestPageRoute,
			OnGenerateRoute:  func(RouteSettings) Route { return nil },
			OnUnknownRoute:   func(RouteSettings) Route { return nil }}))
		state, _ := key.CurrentState()
		requireTag(t, "WidgetsAppUnknownRouteNil", func() { state.PushNamed("/path") })
	})
}

func routerApp(delegate RouterDelegate[*RouteInformation], opts WidgetsAppRouterOpts[*RouteInformation]) Widget {
	opts.RouterDelegate = delegate
	return NewWidgetsAppRouter(appColor, opts)
}

func poppedDelegate() *navigatorDelegate {
	var d *navigatorDelegate
	d = newNavigatorDelegate(func(_ BuildContext, info *RouteInformation) Widget { return navTextExt(info.URI().String()) },
		func(route Route, result any) bool {
			d.routeInformation = uriInfo("popped")
			d.NotifyListeners()
			return route.DidPop(result)
		})
	return d
}

func TestWidgetsAppRouter(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ts := gluttertest.New(t)
		provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		delegate := poppedDelegate()
		defer delegate.Dispose()
		ts.PumpWidget(routerApp(delegate, WidgetsAppRouterOpts[*RouteInformation]{
			RouteInformationProvider: provider, RouteInformationParser: newSimpleParser()}))
		if ts.Count(gluttertest.ByText("initial")) != 1 {
			t.Fatal("initial route not shown")
		}
		// Simulate the android back button intent.
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("popped")) != 1 {
			t.Error("popped route not shown")
		}
	})

	t.Run("route information parser is optional", func(t *testing.T) {
		ts := gluttertest.New(t)
		delegate := poppedDelegate()
		defer delegate.Dispose()
		delegate.routeInformation = uriInfo("initial")
		ts.PumpWidget(routerApp(delegate, WidgetsAppRouterOpts[*RouteInformation]{}))
		if ts.Count(gluttertest.ByText("initial")) != 1 {
			t.Fatal("initial route not shown")
		}
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("popped")) != 1 {
			t.Error("popped route not shown")
		}
	})

	t.Run("router config works", func(t *testing.T) {
		ts := gluttertest.New(t)
		delegate := poppedDelegate()
		defer delegate.Dispose()
		provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		config := RouterConfig[*RouteInformation]{
			RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
			RouterDelegate: delegate, BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding),
		}
		ts.PumpWidget(NewWidgetsAppRouter(appColor, WidgetsAppRouterOpts[*RouteInformation]{RouterConfig: &config}))
		if ts.Count(gluttertest.ByText("initial")) != 1 {
			t.Fatal("initial route not shown")
		}
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("popped")) != 1 {
			t.Error("popped route not shown")
		}
	})

	t.Run("has the correct default", func(t *testing.T) {
		ts := gluttertest.New(t)
		delegate := newNavigatorDelegate(func(_ BuildContext, info *RouteInformation) Widget { return navTextExt(info.URI().String()) },
			func(Route, any) bool { return true })
		defer delegate.Dispose()
		ts.PumpWidget(routerApp(delegate, WidgetsAppRouterOpts[*RouteInformation]{RouteInformationParser: newSimpleParser()}))
		if ts.Count(gluttertest.ByText("/")) != 1 {
			t.Error("the default route should be /")
		}
	})

	t.Run("throws if the route information provider is provided but no parser", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		delegate := poppedDelegate()
		defer delegate.Dispose()
		provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		requireTag(t, "WidgetsAppRouterParserMissing", func() {
			routerApp(delegate, WidgetsAppRouterOpts[*RouteInformation]{RouteInformationProvider: provider})
		})
	})

	t.Run("throws if the router config is provided along with other delegates", func(t *testing.T) {
		skipUnlessDebug(t)
		delegate := poppedDelegate()
		defer delegate.Dispose()
		config := RouterConfig[*RouteInformation]{RouterDelegate: delegate}
		requireTag(t, "WidgetsAppRouterConfigConflict", func() {
			NewWidgetsAppRouter(appColor, WidgetsAppRouterOpts[*RouteInformation]{RouterDelegate: delegate, RouterConfig: &config})
		})
	})

	t.Run("throws without a delegate or a config", func(t *testing.T) {
		skipUnlessDebug(t)
		requireTag(t, "WidgetsAppRouterDelegateMissing", func() {
			NewWidgetsAppRouter(appColor, WidgetsAppRouterOpts[*RouteInformation]{})
		})
	})
}

func TestWidgetsAppConstructorFailures(t *testing.T) {
	skipUnlessDebug(t)
	page := func(settings RouteSettings, b WidgetBuilder) PageRoute { return defaultTestPageRoute(settings, b) }
	for _, tc := range []struct {
		tag  string
		opts WidgetsAppOpts
	}{
		{"WidgetsAppHomeRedundant", WidgetsAppOpts{Home: NewContainer(nil), PageRouteBuilder: page,
			OnGenerateInitialRoutes: func(string) []Route { return nil }}},
		{"WidgetsAppHomeAndDefaultRoute", WidgetsAppOpts{Home: NewContainer(nil), PageRouteBuilder: page,
			Routes: map[string]WidgetBuilder{"/": func(BuildContext) Widget { return NewContainer(nil) }}}},
		{"WidgetsAppNoRoute", WidgetsAppOpts{}},
		{"WidgetsAppBuilderOnly", WidgetsAppOpts{Builder: func(BuildContext, Widget) Widget { return nil }, InitialRoute: "/x"}},
		{"WidgetsAppPageRouteBuilderMissing", WidgetsAppOpts{Home: NewContainer(nil)}},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			requireTag(t, tc.tag, func() { NewWidgetsApp(appColor, tc.opts) })
		})
	}
}

func TestBasicLocaleListResolution(t *testing.T) {
	l := func(lang, script, country string) ui.Locale {
		return ui.Locale{LanguageCode: lang, ScriptCode: script, CountryCode: country}
	}
	for _, tc := range []struct {
		name               string
		preferred, support []ui.Locale
		want               ui.Locale
	}{
		{"matches exactly for the language code",
			[]ui.Locale{l("zh", "", ""), l("un", "", ""), l("en", "", "")}, []ui.Locale{l("en", "", "")}, l("en", "", "")},
		{"matches exactly for the language code and the country code",
			[]ui.Locale{l("en", "", ""), l("en", "", "US")}, []ui.Locale{l("en", "", "US")}, l("en", "", "US")},
		{"matches language+script over language+country",
			[]ui.Locale{l("zh", "Hant", "HK")}, []ui.Locale{l("zh", "", "HK"), l("zh", "Hant", "")}, l("zh", "Hant", "")},
		{"matches exactly for the language, script and country codes",
			[]ui.Locale{l("zh", "", ""), l("zh", "Hant", "TW")}, []ui.Locale{l("zh", "Hant", "TW")}, l("zh", "Hant", "TW")},
		{"selects for the country code if the language code is not found",
			[]ui.Locale{l("en", "", ""), l("ar", "", "tn")}, []ui.Locale{l("fr", "", "tn")}, l("fr", "", "tn")},
		{"selects the first locale when there is no match at all",
			[]ui.Locale{l("tn", "", "")}, []ui.Locale{l("zh", "", ""), l("un", "", ""), l("en", "", "")}, l("zh", "", "")},
		{"selects the first locale when there are no preferred ones",
			nil, []ui.Locale{l("zh", "", ""), l("en", "", "")}, l("zh", "", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := BasicLocaleListResolution(tc.preferred, tc.support); got != tc.want {
				t.Errorf("resolved %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWidgetsAppReportsAnExceptionIfTheSelectedLocaleIsNotSupported(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) {
		c.Locales = []ui.Locale{ui.NewLocale("en", "US"), ui.NewLocale("zh", "CN")}
	})
	var localesArg, supportedArg []ui.Locale
	ts.PumpWidget(NewWidgetsApp(appColor, WidgetsAppOpts{
		LocaleListResolutionCallback: func(locales, supported []ui.Locale) (ui.Locale, bool) {
			localesArg, supportedArg = locales, supported
			return ui.Locale{LanguageCode: "C_UTF-8"}, true
		},
		LocalizationsDelegates: []LocalizationsDelegate{&FuncLocalizationsDelegate[string]{
			Supported: func(l ui.Locale) bool { return l.LanguageCode == "en" },
			LoadFunc:  func(_ context.Context, l ui.Locale) (string, error) { return "x", nil },
		}},
		Builder: func(BuildContext, Widget) Widget { return NewPlaceholder(nil) },
	}))
	if len(localesArg) != 2 || localesArg[0] != ui.NewLocale("en", "US") || localesArg[1] != ui.NewLocale("zh", "CN") {
		t.Errorf("locales = %v", localesArg)
	}
	if len(supportedArg) != 1 || supportedArg[0] != ui.NewLocale("en", "US") {
		t.Errorf("supported locales = %v", supportedArg)
	}
	_ = requireErrorTag(t, takeFirst(t, ts), "LocaleNotSupported")
}

func TestLocalizationsResolver(t *testing.T) {
	en, gb, fr := ui.NewLocale("en", "US"), ui.NewLocale("en", "GB"), ui.NewLocale("fr", "FR")

	t.Run("resolves from the locales of the system and follows their changes", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{gb, fr} })
		r := NewLocalizationsResolver(ts.Binding, []ui.Locale{en, gb, fr})
		defer r.Dispose()
		if got := r.Locale(); got != gb {
			t.Fatalf("locale = %v, want %v", got, gb)
		}
		notified := 0
		r.AddListener(func() { notified++ })
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{fr} })
		if got := r.Locale(); got != fr || notified != 1 {
			t.Errorf("locale = %v, notified %d times; want %v once", got, notified, fr)
		}
		// The same result does not notify.
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{fr, en} })
		if notified != 1 {
			t.Errorf("notified %d times, want 1", notified)
		}
	})

	t.Run("uses the callbacks, then the default algorithm", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{fr} })
		var seen ui.Locale
		var hadFirst bool
		r := NewLocalizationsResolver(ts.Binding, []ui.Locale{en, gb}, LocalizationsResolverOpts{
			LocaleListResolutionCallback: func([]ui.Locale, []ui.Locale) (ui.Locale, bool) { return ui.Locale{}, false },
			LocaleResolutionCallback: func(first ui.Locale, has bool, _ []ui.Locale) (ui.Locale, bool) {
				seen, hadFirst = first, has
				return gb, true
			},
		})
		defer r.Dispose()
		if got := r.Locale(); got != gb || seen != fr || !hadFirst {
			t.Errorf("locale = %v, saw %v (%v); want %v from %v", got, seen, hadFirst, gb, fr)
		}
		// Without the callbacks, the locale is resolved again (the supported ones
		// changed) by the default algorithm.
		r.Update([]ui.Locale{en, fr}, LocalizationsResolverOpts{})
		if got := r.Locale(); got != fr {
			t.Errorf("locale = %v, want %v", got, fr)
		}
	})

	t.Run("a forced locale is resolved against the supported ones", func(t *testing.T) {
		ts := gluttertest.New(t)
		r := NewLocalizationsResolver(ts.Binding, []ui.Locale{en, gb}, LocalizationsResolverOpts{Locale: foundation.Some(gb)})
		defer r.Dispose()
		if got := r.Locale(); got != gb {
			t.Errorf("locale = %v, want %v", got, gb)
		}
		r.Update([]ui.Locale{en, gb}, LocalizationsResolverOpts{Locale: foundation.Some(ui.NewLocale("de", "DE"))})
		if got := r.Locale(); got != en {
			t.Errorf("an unsupported forced locale resolved to %v, want %v", got, en)
		}
	})

	t.Run("adds the default widgets delegate after the delegates of the app", func(t *testing.T) {
		ts := gluttertest.New(t)
		mine := &FuncLocalizationsDelegate[string]{LoadFunc: func(context.Context, ui.Locale) (string, error) { return "x", nil }}
		r := NewLocalizationsResolver(ts.Binding, []ui.Locale{en}, LocalizationsResolverOpts{LocalizationsDelegates: []LocalizationsDelegate{mine}})
		defer r.Dispose()
		got := r.LocalizationsDelegates()
		if len(got) != 2 || got[0] != LocalizationsDelegate(mine) || got[1] != DefaultWidgetsLocalizationsDelegate {
			t.Errorf("delegates = %v", got)
		}
	})
}
