package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// errorTextStyle is the style of text that has no Material above it: big, red,
// underlined with a double yellow line, so that the mistake is seen.
//
// Flutter: material/app.dart _errorTextStyle
func errorTextStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		Color:           foundation.Some(ui.NewColor(0xD0FF0000)),
		FontFamily:      "monospace",
		FontSize:        foundation.Some(48.0),
		FontWeight:      ui.FontWeightW900,
		Decoration:      foundation.Some(ui.TextDecorationUnderline),
		DecorationColor: foundation.Some(ui.NewColor(0xFFFFFF00)),
		DecorationStyle: ui.TextDecorationStyleDouble,
		DebugLabel:      "fallback style; consider putting your text in a Material",
	})
}

// ThemeMode describes which theme will be used by MaterialApp.
//
// Flutter: material/app.dart ThemeMode
type ThemeMode int

// The values of ThemeMode follow Flutter's declaration order.
const (
	// ThemeModeUnspecified is the zero value: the default, ThemeModeSystem.
	//
	// Flutter: no counterpart
	ThemeModeUnspecified ThemeMode = iota
	// ThemeModeSystem uses either the light or the dark theme based on the
	// brightness setting of the system.
	//
	// Flutter: material/app.dart ThemeMode.system
	ThemeModeSystem
	// ThemeModeLight always uses the light theme.
	//
	// Flutter: material/app.dart ThemeMode.light
	ThemeModeLight
	// ThemeModeDark always uses the dark theme.
	//
	// Flutter: material/app.dart ThemeMode.dark
	ThemeModeDark
)

// Name is the name of the mode, as Dart's Enum.name.
//
// Flutter: material/app.dart ThemeMode
func (m ThemeMode) Name() string {
	switch m {
	case ThemeModeLight:
		return "light"
	case ThemeModeDark:
		return "dark"
	}
	return "system"
}

// String is "ThemeMode.system", "ThemeMode.light" or "ThemeMode.dark".
//
// Flutter: material/app.dart ThemeMode
func (m ThemeMode) String() string { return "ThemeMode." + m.Name() }

// IsSystem is whether the mode follows the system.
//
// Flutter: material/app.dart ThemeMode.isSystem
func (m ThemeMode) IsSystem() bool { return m == ThemeModeSystem || m == ThemeModeUnspecified }

// IsLight is whether the mode is always light.
//
// Flutter: material/app.dart ThemeMode.isLight
func (m ThemeMode) IsLight() bool { return m == ThemeModeLight }

// IsDark is whether the mode is always dark.
//
// Flutter: material/app.dart ThemeMode.isDark
func (m ThemeMode) IsDark() bool { return m == ThemeModeDark }

// MaterialAppOpts are the optional parameters of NewMaterialApp.
//
// Not part of them: as the framework has no counterpart, CheckerboardRasterCacheImages,
// CheckerboardOffscreenLayers, ShowSemanticsDebugger, DebugShowMaterialGrid and
// RestorationScopeId (see SPEC §12.10).
//
// Flutter: material/app.dart MaterialApp.new
type MaterialAppOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NavigatorKey is the key of the Navigator of the app.
	NavigatorKey widgets.AnyGlobalKey
	// ScaffoldMessengerKey is the key of the ScaffoldMessenger of the app, to show
	// snack bars and banners without a BuildContext.
	ScaffoldMessengerKey widgets.AnyGlobalKey
	// Home is the widget for the default route of the app ("/").
	Home widgets.Widget
	// Routes is the application's top-level routing table.
	Routes map[string]widgets.WidgetBuilder
	// InitialRoute is the name of the first route to show.
	InitialRoute string
	// OnGenerateRoute is the route generator callback used when the app is
	// navigated to a named route.
	OnGenerateRoute widgets.RouteFactory
	// OnGenerateInitialRoutes generates the initial routes.
	OnGenerateInitialRoutes widgets.InitialRouteListFactory
	// OnUnknownRoute is called when OnGenerateRoute fails to generate a route.
	OnUnknownRoute widgets.RouteFactory
	// OnNavigationNotification is called when a navigation notification
	// bubbles up.
	OnNavigationNotification widgets.NotificationListenerCallback[widgets.NavigationNotification]
	// NavigatorObservers are the observers of the Navigator.
	NavigatorObservers []widgets.NavigatorObserver
	// Builder is a builder for the part of the tree above the Navigator.
	Builder widgets.TransitionBuilder
	// Title is a one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title with a context that has the localizations.
	OnGenerateTitle widgets.GenerateAppTitle
	// Color is the primary color of the app for the system; the default is the
	// primary color of Theme, or Colors.Blue.
	Color foundation.Opt[ui.Color]
	// Theme is the theme of the app; the default is NewThemeData().
	Theme *ThemeData
	// DarkTheme is the theme used when the brightness is dark (ThemeMode).
	DarkTheme *ThemeData
	// HighContrastTheme is the theme used when the system asks for high contrast.
	HighContrastTheme *ThemeData
	// HighContrastDarkTheme is the theme used when the brightness is dark and the
	// system asks for high contrast.
	HighContrastDarkTheme *ThemeData
	// ThemeMode decides which of the themes is used; the default is
	// ThemeModeSystem.
	ThemeMode ThemeMode
	// ThemeAnimationDuration is the duration of the animation of a theme change;
	// the default is ThemeAnimationDuration.
	ThemeAnimationDuration foundation.Opt[time.Duration]
	// ThemeAnimationCurve is the curve of the animation of a theme change; the
	// default is Curves.Linear.
	ThemeAnimationCurve animation.Curve
	// ThemeAnimationStyle overrides the duration and the curve of the animation
	// of a theme change; AnimationStyleNoAnimation turns the animation off.
	ThemeAnimationStyle *animation.AnimationStyle
	// Locale is the locale of the app; the default is the one of the system.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates are the delegates of the localizations; the
	// delegate of the English Material localizations is always added.
	LocalizationsDelegates []widgets.LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the list of the
	// locales of the system.
	LocaleListResolutionCallback widgets.LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first locale of the
	// system.
	LocaleResolutionCallback widgets.LocaleResolutionCallback
	// SupportedLocales are the locales that the app has localizations for; the
	// default is en_US.
	SupportedLocales []ui.Locale
	// DebugShowCheckedModeBanner turns on a "DEBUG" banner in debug builds; the
	// default is true.
	DebugShowCheckedModeBanner foundation.Opt[bool]
	// ShowPerformanceOverlay turns on an overlay at the top of the app with the graphs
	// and the statistics of the time to build and to draw each frame.
	ShowPerformanceOverlay bool
	// Shortcuts are the keyboard shortcuts of the app.
	Shortcuts widgets.ShortcutMap
	// Actions are the actions of the app.
	Actions widgets.ActionMap
	// ScrollBehavior is the scroll behavior of the app; the default is
	// MaterialScrollBehavior.
	ScrollBehavior widgets.ScrollBehavior
}

// MaterialAppRouterOpts are the optional parameters of NewMaterialAppRouter.
//
// Flutter: material/app.dart MaterialApp.router
type MaterialAppRouterOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// RouteInformationProvider provides the route information of the platform.
	RouteInformationProvider widgets.RouteInformationProvider
	// RouteInformationParser parses the route information to a configuration.
	RouteInformationParser widgets.RouteInformationParser[T]
	// RouterDelegate builds the content of the app from the configuration.
	RouterDelegate widgets.RouterDelegate[T]
	// ScaffoldMessengerKey is the key of the ScaffoldMessenger of the app, to show
	// snack bars and banners without a BuildContext.
	ScaffoldMessengerKey widgets.AnyGlobalKey
	// RouterConfig has all of the above in one value.
	RouterConfig *widgets.RouterConfig[T]
	// BackButtonDispatcher dispatches the back button.
	BackButtonDispatcher widgets.BackButtonDispatcher
	// Builder is a builder for the part of the tree above the router.
	Builder widgets.TransitionBuilder
	// Title is a one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title with a context that has the localizations.
	OnGenerateTitle widgets.GenerateAppTitle
	// OnNavigationNotification is called when a navigation notification
	// bubbles up.
	OnNavigationNotification widgets.NotificationListenerCallback[widgets.NavigationNotification]
	// Color is the primary color of the app for the system.
	Color foundation.Opt[ui.Color]
	// Theme is the theme of the app.
	Theme *ThemeData
	// DarkTheme is the theme used when the brightness is dark.
	DarkTheme *ThemeData
	// HighContrastTheme is the theme used when the system asks for high contrast.
	HighContrastTheme *ThemeData
	// HighContrastDarkTheme is the theme used when the brightness is dark and the
	// system asks for high contrast.
	HighContrastDarkTheme *ThemeData
	// ThemeMode decides which of the themes is used.
	ThemeMode ThemeMode
	// ThemeAnimationDuration is the duration of the animation of a theme change.
	ThemeAnimationDuration foundation.Opt[time.Duration]
	// ThemeAnimationCurve is the curve of the animation of a theme change.
	ThemeAnimationCurve animation.Curve
	// ThemeAnimationStyle overrides the duration and the curve of the animation
	// of a theme change.
	ThemeAnimationStyle *animation.AnimationStyle
	// Locale is the locale of the app.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates are the delegates of the localizations.
	LocalizationsDelegates []widgets.LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the list of the
	// locales of the system.
	LocaleListResolutionCallback widgets.LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first locale of the
	// system.
	LocaleResolutionCallback widgets.LocaleResolutionCallback
	// SupportedLocales are the locales that the app has localizations for.
	SupportedLocales []ui.Locale
	// DebugShowCheckedModeBanner turns on a "DEBUG" banner in debug builds; the
	// default is true.
	DebugShowCheckedModeBanner foundation.Opt[bool]
	// ShowPerformanceOverlay turns on an overlay at the top of the app with the graphs
	// and the statistics of the time to build and to draw each frame.
	ShowPerformanceOverlay bool
	// Shortcuts are the keyboard shortcuts of the app.
	Shortcuts widgets.ShortcutMap
	// Actions are the actions of the app.
	Actions widgets.ActionMap
	// ScrollBehavior is the scroll behavior of the app.
	ScrollBehavior widgets.ScrollBehavior
}

// materialAppTheming are the parameters of MaterialApp that choose and animate
// the theme, which both of its constructors have.
type materialAppTheming struct {
	theme, darkTheme, highContrastTheme, highContrastDarkTheme *ThemeData
	themeMode                                                  ThemeMode
	themeAnimationDuration                                     foundation.Opt[time.Duration]
	themeAnimationCurve                                        animation.Curve
	themeAnimationStyle                                        *animation.AnimationStyle
	color                                                      foundation.Opt[ui.Color]
	scrollBehavior                                             widgets.ScrollBehavior
	builder                                                    widgets.TransitionBuilder
	localizationsDelegates                                     []widgets.LocalizationsDelegate
	scaffoldMessengerKey                                       widgets.AnyGlobalKey
}

// MaterialApp is a convenience widget that wraps a number of widgets that are
// commonly required for Material Design applications: a WidgetsApp with the
// Material theme, localizations, page routes and scroll behavior.
//
// Flutter: material/app.dart MaterialApp
type MaterialApp struct {
	widgets.StatefulWidget
	materialAppTheming
	// buildWidgetsApp makes the WidgetsApp of the app, with the builder that adds
	// the theme.
	buildWidgetsApp func(key foundation.Key, color ui.Color, delegates []widgets.LocalizationsDelegate, builder widgets.TransitionBuilder) *widgets.WidgetsApp
}

func materialLocalizationsDelegates(user []widgets.LocalizationsDelegate) []widgets.LocalizationsDelegate {
	out := make([]widgets.LocalizationsDelegate, 0, len(user)+1)
	out = append(out, user...)
	return append(out, DefaultMaterialLocalizationsDelegate)
}

// NewMaterialApp creates an app that uses a Navigator, with the home page, the
// routes table and the route generators.
//
// Flutter: material/app.dart MaterialApp.new
func NewMaterialApp(opts ...MaterialAppOpts) *MaterialApp {
	o := firstOpts(opts)
	a := &MaterialApp{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		materialAppTheming: materialAppTheming{theme: o.Theme, darkTheme: o.DarkTheme, highContrastTheme: o.HighContrastTheme,
			highContrastDarkTheme: o.HighContrastDarkTheme, themeMode: o.ThemeMode, themeAnimationDuration: o.ThemeAnimationDuration,
			themeAnimationCurve: o.ThemeAnimationCurve, themeAnimationStyle: o.ThemeAnimationStyle, color: o.Color,
			scrollBehavior: o.ScrollBehavior, builder: o.Builder, localizationsDelegates: o.LocalizationsDelegates,
			scaffoldMessengerKey: o.ScaffoldMessengerKey},
	}
	a.buildWidgetsApp = func(key foundation.Key, color ui.Color, delegates []widgets.LocalizationsDelegate, builder widgets.TransitionBuilder) *widgets.WidgetsApp {
		return widgets.NewWidgetsApp(color, widgets.WidgetsAppOpts{
			Key:          key,
			NavigatorKey: o.NavigatorKey, NavigatorObservers: o.NavigatorObservers,
			PageRouteBuilder: func(settings widgets.RouteSettings, builder widgets.WidgetBuilder) widgets.PageRoute {
				return NewMaterialPageRoute(builder, MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
			},
			Home: o.Home, Routes: o.Routes, InitialRoute: o.InitialRoute,
			OnGenerateRoute: o.OnGenerateRoute, OnGenerateInitialRoutes: o.OnGenerateInitialRoutes, OnUnknownRoute: o.OnUnknownRoute,
			OnNavigationNotification: o.OnNavigationNotification,
			Builder:                  builder, Title: o.Title, OnGenerateTitle: o.OnGenerateTitle, TextStyle: errorTextStyle(),
			Locale: o.Locale, LocalizationsDelegates: delegates,
			LocaleResolutionCallback: o.LocaleResolutionCallback, LocaleListResolutionCallback: o.LocaleListResolutionCallback,
			SupportedLocales:           o.SupportedLocales,
			DebugShowCheckedModeBanner: o.DebugShowCheckedModeBanner, ShowPerformanceOverlay: o.ShowPerformanceOverlay,
			Shortcuts: o.Shortcuts, Actions: o.Actions,
		})
	}
	return a
}

// NewMaterialAppRouter creates an app that uses the Router instead of a
// Navigator. T is the type of the configuration that the router parses.
//
// Flutter: material/app.dart MaterialApp.router
func NewMaterialAppRouter[T any](opts ...MaterialAppRouterOpts[T]) *MaterialApp {
	o := firstOpts(opts)
	if foundation.DebugMode && o.RouterDelegate == nil && o.RouterConfig == nil {
		panic(errors.New(tagMaterialAppRouterMissing, "either one of routerDelegate or routerConfig must be provided"))
	}
	a := &MaterialApp{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		materialAppTheming: materialAppTheming{theme: o.Theme, darkTheme: o.DarkTheme, highContrastTheme: o.HighContrastTheme,
			highContrastDarkTheme: o.HighContrastDarkTheme, themeMode: o.ThemeMode, themeAnimationDuration: o.ThemeAnimationDuration,
			themeAnimationCurve: o.ThemeAnimationCurve, themeAnimationStyle: o.ThemeAnimationStyle, color: o.Color,
			scrollBehavior: o.ScrollBehavior, builder: o.Builder, localizationsDelegates: o.LocalizationsDelegates,
			scaffoldMessengerKey: o.ScaffoldMessengerKey},
	}
	a.buildWidgetsApp = func(key foundation.Key, color ui.Color, delegates []widgets.LocalizationsDelegate, builder widgets.TransitionBuilder) *widgets.WidgetsApp {
		return widgets.NewWidgetsAppRouter(color, widgets.WidgetsAppRouterOpts[T]{
			Key:                      key,
			RouteInformationProvider: o.RouteInformationProvider, RouteInformationParser: o.RouteInformationParser,
			RouterDelegate: o.RouterDelegate, RouterConfig: o.RouterConfig, BackButtonDispatcher: o.BackButtonDispatcher,
			OnNavigationNotification: o.OnNavigationNotification,
			Builder:                  builder, Title: o.Title, OnGenerateTitle: o.OnGenerateTitle, TextStyle: errorTextStyle(),
			Locale: o.Locale, LocalizationsDelegates: delegates,
			LocaleResolutionCallback: o.LocaleResolutionCallback, LocaleListResolutionCallback: o.LocaleListResolutionCallback,
			SupportedLocales:           o.SupportedLocales,
			DebugShowCheckedModeBanner: o.DebugShowCheckedModeBanner, ShowPerformanceOverlay: o.ShowPerformanceOverlay,
			Shortcuts: o.Shortcuts, Actions: o.Actions,
		})
	}
	return a
}

// MaterialAppCreateMaterialHeroController creates a HeroController whose
// flights follow the arcs of the Material motion spec.
//
// Flutter: material/app.dart MaterialApp.createMaterialHeroController
func MaterialAppCreateMaterialHeroController() *widgets.HeroController {
	return widgets.NewHeroController(MaterialRectArcTweenCreate)
}

// CreateState creates the state.
//
// Flutter: material/app.dart MaterialApp.createState
func (w *MaterialApp) CreateState() widgets.State { return &materialAppState{} }

type materialAppState struct {
	widgets.StateOf[*MaterialApp]
	heroController *widgets.HeroController
}

func (s *materialAppState) InitState() { s.heroController = MaterialAppCreateMaterialHeroController() }

func (s *materialAppState) Dispose() { s.heroController.Dispose() }

// themeBuilder picks the theme from the mode, the brightness and the contrast of
// the platform.
//
// Deviation: Flutter also sets the system UI overlay style of the theme; a Linux
// desktop has no status bar.
//
// Flutter: material/app.dart _MaterialAppState._themeBuilder
func (s *materialAppState) themeBuilder(bc widgets.BuildContext) *ThemeData {
	w := s.Widget()
	mode := w.themeMode
	platformBrightness := widgets.MediaQueryPlatformBrightnessOf(bc)
	useDarkTheme := mode.IsDark() || (mode.IsSystem() && platformBrightness == ui.BrightnessDark)
	highContrast := widgets.MediaQueryHighContrastOf(bc)
	var theme *ThemeData
	switch {
	case useDarkTheme && highContrast && w.highContrastDarkTheme != nil:
		theme = w.highContrastDarkTheme
	case useDarkTheme && w.darkTheme != nil:
		theme = w.darkTheme
	case highContrast && w.highContrastTheme != nil:
		theme = w.highContrastTheme
	}
	if theme == nil {
		theme = w.theme
	}
	if theme == nil {
		theme = NewThemeData()
	}
	return theme
}

// materialBuilder adds the theme, the selection style and the builder of the app
// above the navigator.
//
// Flutter: material/app.dart _MaterialAppState._materialBuilder
func (s *materialAppState) materialBuilder(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
	w := s.Widget()
	theme := s.themeBuilder(bc)
	selectionColor := theme.TextSelectionTheme.SelectionColor.OrElse(theme.ColorScheme.Primary().WithOpacity(0.40))
	cursorColor := theme.TextSelectionTheme.CursorColor.OrElse(theme.ColorScheme.Primary())
	var result widgets.Widget = widgets.NewSizedBoxShrink(nil)
	if child != nil {
		result = child
	}
	if w.builder != nil {
		builder, inner := w.builder, child
		result = widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget { return builder(bc, inner) })
	}
	result = NewScaffoldMessenger(result, ScaffoldMessengerOpts{Key: w.scaffoldMessengerKey})
	result = widgets.NewDefaultSelectionStyle(result, widgets.DefaultSelectionStyleOpts{
		SelectionColor: foundation.Some(selectionColor), CursorColor: foundation.Some(cursorColor),
	})
	if w.themeAnimationStyle == nil || !w.themeAnimationStyle.Equals(animation.AnimationStyleNoAnimation) {
		opts := AnimatedThemeOpts{Duration: w.themeAnimationDuration, Curve: w.themeAnimationCurve}
		if w.themeAnimationStyle != nil {
			if d, ok := w.themeAnimationStyle.Duration.Get(); ok {
				opts.Duration = foundation.Some(d)
			}
			if w.themeAnimationStyle.Curve != nil {
				opts.Curve = w.themeAnimationStyle.Curve
			}
		}
		return NewAnimatedTheme(theme, result, opts)
	}
	return NewTheme(theme, result)
}

// Build builds the WidgetsApp inside the scroll behavior and the hero controller.
//
// Flutter: material/app.dart _MaterialAppState.build
func (s *materialAppState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	materialColor := w.color.OrElse(Colors.Blue.Primary)
	if !w.color.IsSome() && w.theme != nil {
		materialColor = w.theme.PrimaryColor
	}
	app := w.buildWidgetsApp(widgets.NewGlobalObjectKey[widgets.State](s), materialColor,
		materialLocalizationsDelegates(w.localizationsDelegates), s.materialBuilder)
	behavior := w.scrollBehavior
	if behavior == nil {
		behavior = NewMaterialScrollBehavior()
	}
	return widgets.NewScrollConfiguration(behavior, widgets.NewHeroControllerScope(s.heroController, app))
}

// MaterialScrollBehavior describes how scrollables behave in a MaterialApp: the
// platform comes from the theme, and the vertical scrollables of the desktop
// platforms have a scrollbar.
//
// Deviation: Flutter builds a Material Scrollbar (scrollbar.dart, ported in step
// 13.9) and, on Android, the stretching or glowing overscroll indicators; until
// then this behavior builds the RawScrollbar of the widgets layer and no
// overscroll indicator.
//
// Flutter: material/app.dart MaterialScrollBehavior
type MaterialScrollBehavior struct {
	widgets.ScrollBehaviorBase
}

// NewMaterialScrollBehavior creates the behavior.
//
// Flutter: material/app.dart MaterialScrollBehavior.new
func NewMaterialScrollBehavior() *MaterialScrollBehavior {
	b := &MaterialScrollBehavior{}
	widgets.InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

// GetPlatform is the platform of the theme.
//
// Flutter: material/app.dart MaterialScrollBehavior.getPlatform
func (b *MaterialScrollBehavior) GetPlatform(bc widgets.BuildContext) foundation.TargetPlatform {
	return ThemeOf(bc).Platform
}

// BuildScrollbar wraps the child of a vertical scrollable of a desktop platform
// in a scrollbar.
//
// Flutter: material/app.dart MaterialScrollBehavior.buildScrollbar
func (b *MaterialScrollBehavior) BuildScrollbar(bc widgets.BuildContext, child widgets.Widget, details widgets.ScrollableDetails) widgets.Widget {
	if painting.AxisDirectionToAxis(details.Direction) == painting.AxisHorizontal {
		return child
	}
	switch b.GetPlatform(bc) {
	case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		if foundation.DebugMode && details.Controller == nil {
			panic(errors.New(tagMaterialScrollbarControllerMissing, "a ScrollController is required to build a scrollbar for a Scrollable; the details of the scrollable must have a controller"))
		}
		return NewScrollbar(child, ScrollbarOpts{Controller: details.Controller})
	}
	return child
}

// String is the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (b *MaterialScrollBehavior) String() string { return "MaterialScrollBehavior" }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMaterialAppRouterMissing           = "MaterialAppRouterMissing"
	tagMaterialScrollbarControllerMissing = "MaterialScrollbarControllerMissing"
)
