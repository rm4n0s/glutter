package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoAppOpts are the optional parameters of NewCupertinoApp.
//
// Not part of them: as the framework has no counterpart, CheckerboardRasterCacheImages,
// CheckerboardOffscreenLayers, ShowSemanticsDebugger, RestorationScopeId and the deprecated
// UseInheritedMediaQuery (see SPEC §12.10).
//
// Flutter: cupertino/app.dart CupertinoApp.new
type CupertinoAppOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NavigatorKey is the key of the Navigator of the app.
	NavigatorKey widgets.AnyGlobalKey
	// Home is the widget for the default route of the app ("/").
	Home widgets.Widget
	// Theme is the theme of the app: the colors and the text styles of the Cupertino widgets.
	Theme *CupertinoThemeData
	// Routes is the application's top-level routing table.
	Routes map[string]widgets.WidgetBuilder
	// InitialRoute is the name of the first route to show.
	InitialRoute string
	// OnGenerateRoute is the route generator callback used when the app is navigated to a named route.
	OnGenerateRoute widgets.RouteFactory
	// OnGenerateInitialRoutes generates the initial routes.
	OnGenerateInitialRoutes widgets.InitialRouteListFactory
	// OnUnknownRoute is called when OnGenerateRoute fails to generate a route.
	OnUnknownRoute widgets.RouteFactory
	// OnNavigationNotification is called when a navigation notification bubbles up.
	OnNavigationNotification widgets.NotificationListenerCallback[widgets.NavigationNotification]
	// NavigatorObservers are the observers of the Navigator.
	NavigatorObservers []widgets.NavigatorObserver
	// Builder is a builder for the part of the tree above the Navigator.
	Builder widgets.TransitionBuilder
	// Title is a one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title with a context that has the localizations.
	OnGenerateTitle widgets.GenerateAppTitle
	// Color is the primary color of the app for the system; the default is the primary color of the theme.
	Color ui.ColorValue
	// Locale is the locale of the app; the default is the one of the system.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates are the delegates of the localizations; the delegate of the English Cupertino
	// localizations is always added.
	LocalizationsDelegates []widgets.LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the list of the locales of the system.
	LocaleListResolutionCallback widgets.LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first locale of the system.
	LocaleResolutionCallback widgets.LocaleResolutionCallback
	// SupportedLocales are the locales that the app has localizations for; the default is en_US.
	SupportedLocales []ui.Locale
	// ShowPerformanceOverlay turns on an overlay at the top of the app with the graphs and the statistics of
	// the time to build and to draw each frame.
	ShowPerformanceOverlay bool
	// DebugShowCheckedModeBanner turns on a "DEBUG" banner in debug builds; the default is true.
	DebugShowCheckedModeBanner foundation.Opt[bool]
	// Shortcuts are the keyboard shortcuts of the app.
	Shortcuts widgets.ShortcutMap
	// Actions are the actions of the app.
	Actions widgets.ActionMap
	// ScrollBehavior is the scroll behavior of the app; the default is CupertinoScrollBehavior.
	ScrollBehavior widgets.ScrollBehavior
}

// CupertinoAppRouterOpts are the optional parameters of NewCupertinoAppRouter.
//
// Flutter: cupertino/app.dart CupertinoApp.router
type CupertinoAppRouterOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// RouteInformationProvider provides the route information of the platform.
	RouteInformationProvider widgets.RouteInformationProvider
	// RouteInformationParser parses the route information to a configuration.
	RouteInformationParser widgets.RouteInformationParser[T]
	// RouterDelegate builds the content of the app from the configuration.
	RouterDelegate widgets.RouterDelegate[T]
	// RouterConfig has all of the above in one value.
	RouterConfig *widgets.RouterConfig[T]
	// BackButtonDispatcher dispatches the back button.
	BackButtonDispatcher widgets.BackButtonDispatcher
	// Theme is the theme of the app.
	Theme *CupertinoThemeData
	// Builder is a builder for the part of the tree above the router.
	Builder widgets.TransitionBuilder
	// Title is a one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title with a context that has the localizations.
	OnGenerateTitle widgets.GenerateAppTitle
	// OnNavigationNotification is called when a navigation notification bubbles up.
	OnNavigationNotification widgets.NotificationListenerCallback[widgets.NavigationNotification]
	// Color is the primary color of the app for the system.
	Color ui.ColorValue
	// Locale is the locale of the app.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates are the delegates of the localizations.
	LocalizationsDelegates []widgets.LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the list of the locales of the system.
	LocaleListResolutionCallback widgets.LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first locale of the system.
	LocaleResolutionCallback widgets.LocaleResolutionCallback
	// SupportedLocales are the locales that the app has localizations for.
	SupportedLocales []ui.Locale
	// ShowPerformanceOverlay turns on an overlay with the graphs of the frame times.
	ShowPerformanceOverlay bool
	// DebugShowCheckedModeBanner turns on a "DEBUG" banner in debug builds; the default is true.
	DebugShowCheckedModeBanner foundation.Opt[bool]
	// Shortcuts are the keyboard shortcuts of the app.
	Shortcuts widgets.ShortcutMap
	// Actions are the actions of the app.
	Actions widgets.ActionMap
	// ScrollBehavior is the scroll behavior of the app.
	ScrollBehavior widgets.ScrollBehavior
}

// cupertinoAppCommon are the parameters that both constructors of CupertinoApp have.
type cupertinoAppCommon struct {
	theme                  *CupertinoThemeData
	color                  ui.ColorValue
	scrollBehavior         widgets.ScrollBehavior
	builder                widgets.TransitionBuilder
	localizationsDelegates []widgets.LocalizationsDelegate
}

// CupertinoApp is a convenience widget that wraps a number of widgets that are commonly required for an
// iOS styled application: a WidgetsApp with the Cupertino theme, localizations, page routes and scroll
// behavior.
//
// Flutter: cupertino/app.dart CupertinoApp
type CupertinoApp struct {
	widgets.StatefulWidget
	cupertinoAppCommon
	// buildWidgetsApp makes the WidgetsApp of the app, with the text style and the color of the theme.
	buildWidgetsApp func(key foundation.Key, textStyle *painting.TextStyle, color ui.Color, delegates []widgets.LocalizationsDelegate) *widgets.WidgetsApp
}

func cupertinoLocalizationsDelegates(user []widgets.LocalizationsDelegate) []widgets.LocalizationsDelegate {
	out := make([]widgets.LocalizationsDelegate, 0, len(user)+1)
	out = append(out, user...)
	return append(out, DefaultCupertinoLocalizationsDelegate)
}

// NewCupertinoApp creates an app that uses a Navigator, with the home page, the routes table and the route
// generators.
//
// Flutter: cupertino/app.dart CupertinoApp.new
func NewCupertinoApp(opts ...CupertinoAppOpts) *CupertinoApp {
	o := firstOf(opts)
	a := &CupertinoApp{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		cupertinoAppCommon: cupertinoAppCommon{theme: o.Theme, color: o.Color, scrollBehavior: o.ScrollBehavior, builder: o.Builder,
			localizationsDelegates: o.LocalizationsDelegates},
	}
	a.buildWidgetsApp = func(key foundation.Key, textStyle *painting.TextStyle, color ui.Color, delegates []widgets.LocalizationsDelegate) *widgets.WidgetsApp {
		return widgets.NewWidgetsApp(color, widgets.WidgetsAppOpts{
			Key: key, NavigatorKey: o.NavigatorKey, NavigatorObservers: o.NavigatorObservers,
			PageRouteBuilder: func(settings widgets.RouteSettings, builder widgets.WidgetBuilder) widgets.PageRoute {
				return NewCupertinoPageRoute(builder, CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
			},
			Home: o.Home, Routes: o.Routes, InitialRoute: o.InitialRoute,
			OnGenerateRoute: o.OnGenerateRoute, OnGenerateInitialRoutes: o.OnGenerateInitialRoutes, OnUnknownRoute: o.OnUnknownRoute,
			OnNavigationNotification: o.OnNavigationNotification,
			Builder:                  o.Builder, Title: o.Title, OnGenerateTitle: o.OnGenerateTitle, TextStyle: textStyle,
			Locale: o.Locale, LocalizationsDelegates: delegates,
			LocaleResolutionCallback: o.LocaleResolutionCallback, LocaleListResolutionCallback: o.LocaleListResolutionCallback,
			SupportedLocales:           o.SupportedLocales,
			DebugShowCheckedModeBanner: o.DebugShowCheckedModeBanner, ShowPerformanceOverlay: o.ShowPerformanceOverlay,
			Shortcuts: o.Shortcuts, Actions: o.Actions,
		})
	}
	return a
}

// NewCupertinoAppRouter creates an app that uses the Router instead of a Navigator. T is the type of the
// configuration that the router parses. In debug builds it panics with the tag CupertinoAppRouterMissing when
// there is neither a router delegate nor a router config.
//
// Flutter: cupertino/app.dart CupertinoApp.router
func NewCupertinoAppRouter[T any](opts ...CupertinoAppRouterOpts[T]) *CupertinoApp {
	o := firstOf(opts)
	if foundation.DebugMode && o.RouterDelegate == nil && o.RouterConfig == nil {
		panic(errors.New(tagCupertinoAppRouterMissing, "either one of routerDelegate or routerConfig must be provided"))
	}
	a := &CupertinoApp{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		cupertinoAppCommon: cupertinoAppCommon{theme: o.Theme, color: o.Color, scrollBehavior: o.ScrollBehavior, builder: o.Builder,
			localizationsDelegates: o.LocalizationsDelegates},
	}
	a.buildWidgetsApp = func(key foundation.Key, textStyle *painting.TextStyle, color ui.Color, delegates []widgets.LocalizationsDelegate) *widgets.WidgetsApp {
		return widgets.NewWidgetsAppRouter(color, widgets.WidgetsAppRouterOpts[T]{
			Key:                      key,
			RouteInformationProvider: o.RouteInformationProvider, RouteInformationParser: o.RouteInformationParser,
			RouterDelegate: o.RouterDelegate, RouterConfig: o.RouterConfig, BackButtonDispatcher: o.BackButtonDispatcher,
			OnNavigationNotification: o.OnNavigationNotification,
			Builder:                  o.Builder, Title: o.Title, OnGenerateTitle: o.OnGenerateTitle, TextStyle: textStyle,
			Locale: o.Locale, LocalizationsDelegates: delegates,
			LocaleResolutionCallback: o.LocaleResolutionCallback, LocaleListResolutionCallback: o.LocaleListResolutionCallback,
			SupportedLocales:           o.SupportedLocales,
			DebugShowCheckedModeBanner: o.DebugShowCheckedModeBanner, ShowPerformanceOverlay: o.ShowPerformanceOverlay,
			Shortcuts: o.Shortcuts, Actions: o.Actions,
		})
	}
	return a
}

// CupertinoAppCreateCupertinoHeroController creates a HeroController whose flights follow a straight line:
// the linear tweening of Cupertino.
//
// Flutter: cupertino/app.dart CupertinoApp.createCupertinoHeroController
func CupertinoAppCreateCupertinoHeroController() *widgets.HeroController {
	return widgets.NewHeroController()
}

// CreateState creates the state.
//
// Flutter: cupertino/app.dart CupertinoApp.createState
func (w *CupertinoApp) CreateState() widgets.State { return &cupertinoAppState{} }

type cupertinoAppState struct {
	widgets.StateOf[*CupertinoApp]
	heroController *widgets.HeroController
}

func (s *cupertinoAppState) InitState() {
	s.heroController = CupertinoAppCreateCupertinoHeroController()
}

func (s *cupertinoAppState) Dispose() { s.heroController.Dispose() }

// buildWidgetApp builds the WidgetsApp with the text style and the color of the resolved theme.
//
// Deviation: the buttons of the widget inspector (Flutter's exitWidgetSelectionButtonBuilder and its
// siblings) are not built, as the widget inspector is not ported (SPEC §12.10).
//
// Flutter: cupertino/app.dart _CupertinoAppState._buildWidgetApp
func (s *cupertinoAppState) buildWidgetApp(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := CupertinoThemeOf(bc)
	var color ui.ColorValue = theme.PrimaryColor()
	if w.color != nil {
		color = w.color
	}
	return w.buildWidgetsApp(widgets.NewGlobalObjectKey[widgets.State](s), theme.TextTheme().TextStyle(),
		CupertinoDynamicColorResolve(color, bc), cupertinoLocalizationsDelegates(w.localizationsDelegates))
}

// Build builds the theme, the interface level, the selection style and the hero controller above the
// WidgetsApp, and sets the style of the system bars from the brightness.
//
// Flutter: cupertino/app.dart _CupertinoAppState.build
func (s *cupertinoAppState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	data := w.theme
	if data == nil {
		data = NewCupertinoThemeData()
	}
	effective := data.ResolveFrom(bc)
	brightness := effective.Brightness()
	if brightness == ui.BrightnessUnspecified {
		brightness = widgets.MediaQueryPlatformBrightnessOf(bc)
	}
	style := services.SystemUiOverlayStyleDark
	if brightness == ui.BrightnessDark {
		style = services.SystemUiOverlayStyleLight
	}
	widgets.WidgetsBindingOf(bc).SystemChrome().SetSystemUIOverlayStyle(style)
	behavior := w.scrollBehavior
	if behavior == nil {
		behavior = NewCupertinoScrollBehavior()
	}
	primary := effective.PrimaryColor().EffectiveColor()
	return widgets.NewScrollConfiguration(behavior, NewCupertinoUserInterfaceLevel(CupertinoUserInterfaceLevelBase,
		NewCupertinoTheme(effective, widgets.NewDefaultSelectionStyle(
			widgets.NewHeroControllerScope(s.heroController, widgets.NewBuilder(s.buildWidgetApp)),
			widgets.DefaultSelectionStyleOpts{SelectionColor: foundation.Some(primary.WithOpacity(0.2)), CursorColor: foundation.Some(primary)}))))
}

// CupertinoScrollBehavior describes how scrollables behave in a CupertinoApp: they bounce on every platform
// (and decelerate faster on macOS), the vertical ones of the desktop platforms have a CupertinoScrollbar,
// and there is no overscroll glow.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior
type CupertinoScrollBehavior struct {
	widgets.ScrollBehaviorBase
}

// NewCupertinoScrollBehavior creates the behavior.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior.new
func NewCupertinoScrollBehavior() *CupertinoScrollBehavior {
	b := &CupertinoScrollBehavior{}
	widgets.InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

// BuildScrollbar wraps the child of a scrollable of a desktop platform in a CupertinoScrollbar; the other
// platforms have none.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior.buildScrollbar
func (b *CupertinoScrollBehavior) BuildScrollbar(bc widgets.BuildContext, child widgets.Widget, details widgets.ScrollableDetails) widgets.Widget {
	switch b.GetPlatform(bc) {
	case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		if foundation.DebugMode && details.Controller == nil {
			panic(errors.New(tagCupertinoScrollbarControllerMissing, "a ScrollController is required to build a scrollbar for a Scrollable; the details of the scrollable must have a controller"))
		}
		return NewCupertinoScrollbar(child, CupertinoScrollbarOpts{Controller: details.Controller})
	}
	return child
}

// BuildOverscrollIndicator is the child: Cupertino has no overscroll indicator.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior.buildOverscrollIndicator
func (b *CupertinoScrollBehavior) BuildOverscrollIndicator(_ widgets.BuildContext, child widgets.Widget, _ widgets.ScrollableDetails) widgets.Widget {
	return child
}

// GetScrollPhysics is the bouncing physics, with the fast deceleration of the Mac.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior.getScrollPhysics
func (b *CupertinoScrollBehavior) GetScrollPhysics(bc widgets.BuildContext) widgets.ScrollPhysics {
	if b.GetPlatform(bc) == foundation.TargetPlatformMacOS {
		return widgets.NewBouncingScrollPhysics(widgets.BouncingScrollPhysicsOpts{DecelerationRate: widgets.ScrollDecelerationRateFast})
	}
	return widgets.NewBouncingScrollPhysics()
}

// GetMultitouchDragStrategy averages the pointers at the boundary.
//
// Flutter: cupertino/app.dart CupertinoScrollBehavior.getMultitouchDragStrategy
func (b *CupertinoScrollBehavior) GetMultitouchDragStrategy(widgets.BuildContext) gestures.MultitouchDragStrategy {
	return gestures.MultitouchDragStrategyAverageBoundaryPointers
}

// String is the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (b *CupertinoScrollBehavior) String() string { return "CupertinoScrollBehavior" }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoAppRouterMissing           = "CupertinoAppRouterMissing"
	tagCupertinoScrollbarControllerMissing = "CupertinoScrollbarControllerMissing"
)
