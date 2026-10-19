package widgets

import (
	"net/url"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// GenerateAppTitle makes the title of an app from its context, which is below the
// Localizations of the app.
//
// Flutter: widgets/app.dart GenerateAppTitle
type GenerateAppTitle func(bc BuildContext) string

// PageRouteFactory makes the page route of a named route: the settings are the
// name and the arguments, and the builder makes the content of the page.
//
// Deviation: it is not generic over the result type; results are any values
// (SPEC §12.10).
//
// Flutter: widgets/app.dart PageRouteFactory
type PageRouteFactory func(settings RouteSettings, builder WidgetBuilder) PageRoute

// InitialRouteListFactory makes the routes to start the navigator with, from the
// name of the initial route.
//
// Flutter: widgets/app.dart InitialRouteListFactory
type InitialRouteListFactory func(initialRoute string) []Route

// WidgetsAppOpts are the optional parameters of NewWidgetsApp.
//
// Flutter: widgets/app.dart WidgetsApp.new
type WidgetsAppOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NavigatorKey is the key of the navigator of the app; the default is a key
	// the app makes.
	NavigatorKey AnyGlobalKey
	// OnGenerateRoute makes the routes that Home and Routes do not cover.
	OnGenerateRoute RouteFactory
	// OnGenerateInitialRoutes makes the routes to start with; the default is
	// NavigatorDefaultGenerateInitialRoutes. It can not be used with Home.
	OnGenerateInitialRoutes InitialRouteListFactory
	// OnUnknownRoute makes the route when nothing else covers the name.
	OnUnknownRoute RouteFactory
	// OnNavigationNotification handles the NavigationNotifications of the
	// navigators below; the default tells the system whether the framework can
	// handle a pop (SystemNavigator.SetFrameworkHandlesBack).
	OnNavigationNotification NotificationListenerCallback[NavigationNotification]
	// NavigatorObservers are the observers of the navigator.
	NavigatorObservers []NavigatorObserver
	// InitialRoute is the name of the first route; the default is "/".
	InitialRoute string
	// PageRouteBuilder makes the routes of Home and Routes. It is required when
	// either of them is set.
	PageRouteBuilder PageRouteFactory
	// Home is the widget of the "/" route.
	Home Widget
	// Routes is the table of the named routes.
	Routes map[string]WidgetBuilder
	// Builder wraps the navigator (the child is nil when the app has none).
	Builder TransitionBuilder
	// Title is the one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title from the context, and replaces Title.
	OnGenerateTitle GenerateAppTitle
	// TextStyle is the default text style of the app.
	TextStyle *painting.TextStyle
	// Locale forces the locale of the app.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates produce the localized resources of the app.
	LocalizationsDelegates []LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the preferred ones.
	LocaleListResolutionCallback LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first preferred one.
	LocaleResolutionCallback LocaleResolutionCallback
	// SupportedLocales are the locales the app supports; the default is en_US.
	SupportedLocales []ui.Locale
	// Shortcuts replaces the default shortcuts of the app (WidgetsAppDefaultShortcuts).
	Shortcuts ShortcutMap
	// Actions replaces the default actions of the app (WidgetsAppDefaultActions).
	Actions ActionMap
}

// WidgetsAppRouterOpts are the optional parameters of NewWidgetsAppRouter.
//
// Flutter: widgets/app.dart WidgetsApp.router
type WidgetsAppRouterOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// RouteInformationProvider provides the route information; with a parser and
	// without a provider the app uses the one of the platform.
	RouteInformationProvider RouteInformationProvider
	// RouteInformationParser parses the route information.
	RouteInformationParser RouteInformationParser[T]
	// RouterDelegate builds the pages of the app.
	RouterDelegate RouterDelegate[T]
	// RouterConfig holds the four router objects at once; it excludes them.
	RouterConfig *RouterConfig[T]
	// BackButtonDispatcher tells the router that the platform wants to pop; the
	// default is a RootBackButtonDispatcher.
	BackButtonDispatcher BackButtonDispatcher
	// Builder wraps the router.
	Builder TransitionBuilder
	// Title is the one-line description of the app for the system.
	Title string
	// OnGenerateTitle makes the title from the context, and replaces Title.
	OnGenerateTitle GenerateAppTitle
	// OnNavigationNotification handles the NavigationNotifications below.
	OnNavigationNotification NotificationListenerCallback[NavigationNotification]
	// TextStyle is the default text style of the app.
	TextStyle *painting.TextStyle
	// Locale forces the locale of the app.
	Locale foundation.Opt[ui.Locale]
	// LocalizationsDelegates produce the localized resources of the app.
	LocalizationsDelegates []LocalizationsDelegate
	// LocaleListResolutionCallback resolves the locale from the preferred ones.
	LocaleListResolutionCallback LocaleListResolutionCallback
	// LocaleResolutionCallback resolves the locale from the first preferred one.
	LocaleResolutionCallback LocaleResolutionCallback
	// SupportedLocales are the locales the app supports; the default is en_US.
	SupportedLocales []ui.Locale
	// Shortcuts replaces the default shortcuts of the app.
	Shortcuts ShortcutMap
	// Actions replaces the default actions of the app.
	Actions ActionMap
}

// WidgetsApp is a convenience widget that wraps a number of widgets that
// applications commonly require: a Navigator (or a Router) for the pages, the
// Localizations of the app, a Title, the default Shortcuts and Actions of the
// platform, the default text editing shortcuts and a focus traversal group in
// reading order. It does not draw anything itself; MaterialApp builds on it.
//
// Deviation: the restoration scope, SharedAppData, TapRegionSurface, the window
// manager, the performance overlay, the semantics debugger, the widget
// inspector, the checked-mode banner and the dismissal of tooltips on Escape
// are not part of it yet; each is added by the step that ports the widget it
// needs (SPEC §12.10).
//
// Flutter: widgets/app.dart WidgetsApp
type WidgetsApp struct {
	StatefulWidget
	color  ui.Color
	o      WidgetsAppOpts
	router *widgetsAppRouter
}

// widgetsAppRouter is what WidgetsApp keeps of the router parameters, whatever
// the type of the configuration of the router.
type widgetsAppRouter struct {
	hasDelegate, hasConfig, hasParser bool
	provider                          RouteInformationProvider
	dispatcher                        BackButtonDispatcher
	// build makes the Router widget with the provider and the dispatcher to use.
	build func(provider RouteInformationProvider, dispatcher BackButtonDispatcher) Widget
}

// NewWidgetsApp creates an app that navigates with a Navigator. In debug builds
// it panics when the parameters contradict each other, with the tags
// WidgetsAppHomeRedundant, WidgetsAppHomeAndDefaultRoute, WidgetsAppNoRoute,
// WidgetsAppBuilderOnly, WidgetsAppPageRouteBuilderMissing and
// WidgetsAppSupportedLocalesEmpty.
//
// Flutter: widgets/app.dart WidgetsApp.new
func NewWidgetsApp(color ui.Color, opts ...WidgetsAppOpts) *WidgetsApp {
	o := firstOpts(opts)
	if len(o.SupportedLocales) == 0 {
		o.SupportedLocales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	if foundation.DebugMode {
		_, hasDefault := o.Routes[NavigatorDefaultRouteName]
		switch {
		case o.Home != nil && o.OnGenerateInitialRoutes != nil:
			panic(errors.New("WidgetsAppHomeRedundant", "if onGenerateInitialRoutes is specified, the home argument will be redundant"))
		case o.Home != nil && hasDefault:
			panic(errors.New("WidgetsAppHomeAndDefaultRoute", `if the home property is specified, the routes table cannot include an entry for "/", since it would be redundant`))
		case o.Builder == nil && o.Home == nil && !hasDefault && o.OnGenerateRoute == nil && o.OnUnknownRoute == nil:
			panic(errors.New("WidgetsAppNoRoute", `either the home property must be specified, or the routes table must include an entry for "/", or there must be an onGenerateRoute callback specified, or there must be an onUnknownRoute callback specified, or the builder property must be specified, because otherwise there is nothing to fall back on if the app is started with an intent that specifies an unknown route`))
		case o.Home == nil && len(o.Routes) == 0 && o.OnGenerateRoute == nil && o.OnUnknownRoute == nil &&
			(o.Builder == nil || o.NavigatorKey != nil || o.InitialRoute != "" || len(o.NavigatorObservers) > 0):
			panic(errors.New("WidgetsAppBuilderOnly", "if no route is provided using home, routes, onGenerateRoute, or onUnknownRoute, a builder must be provided, and navigatorKey, initialRoute and navigatorObservers must have their initial values"))
		case o.Builder == nil && o.OnGenerateRoute == nil && o.PageRouteBuilder == nil:
			panic(errors.New("WidgetsAppPageRouteBuilderMissing", "if neither builder nor onGenerateRoute are provided, the pageRouteBuilder must be specified so that the default handler will know what kind of PageRoute transition to build"))
		}
	}
	return &WidgetsApp{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), color: color, o: o}
}

// NewWidgetsAppRouter creates an app that navigates with a Router. In debug
// builds it panics with the tags WidgetsAppRouterConfigConflict,
// WidgetsAppRouterDelegateMissing, WidgetsAppRouterParserMissing or
// WidgetsAppSupportedLocalesEmpty when the parameters contradict each other.
//
// Flutter: widgets/app.dart WidgetsApp.router
func NewWidgetsAppRouter[T any](color ui.Color, opts ...WidgetsAppRouterOpts[T]) *WidgetsApp {
	o := firstOpts(opts)
	if len(o.SupportedLocales) == 0 {
		o.SupportedLocales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	if foundation.DebugMode {
		switch {
		case o.RouterConfig != nil && (o.RouteInformationProvider != nil || o.RouteInformationParser != nil || o.RouterDelegate != nil || o.BackButtonDispatcher != nil):
			panic(errors.New("WidgetsAppRouterConfigConflict", "if the routerConfig is provided, all the other router delegates must not be provided"))
		case o.RouterConfig == nil && o.RouterDelegate == nil:
			panic(errors.New("WidgetsAppRouterDelegateMissing", "either one of routerDelegate or routerConfig must be provided"))
		case o.RouterConfig == nil && o.RouteInformationProvider != nil && o.RouteInformationParser == nil:
			panic(errors.New("WidgetsAppRouterParserMissing", "if routeInformationProvider is provided, routeInformationParser must also be provided"))
		}
	}
	r := &widgetsAppRouter{
		hasDelegate: o.RouterDelegate != nil, hasConfig: o.RouterConfig != nil, hasParser: o.RouteInformationParser != nil,
		provider: o.RouteInformationProvider, dispatcher: o.BackButtonDispatcher,
	}
	if o.RouterDelegate != nil {
		r.build = func(provider RouteInformationProvider, dispatcher BackButtonDispatcher) Widget {
			return NewRouter(o.RouterDelegate, RouterOpts[T]{
				RouteInformationProvider: provider,
				RouteInformationParser:   o.RouteInformationParser,
				BackButtonDispatcher:     dispatcher,
			})
		}
	} else if o.RouterConfig != nil {
		r.build = func(RouteInformationProvider, BackButtonDispatcher) Widget {
			return NewRouterWithConfig(*o.RouterConfig)
		}
	}
	w := &WidgetsApp{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), color: color, router: r}
	w.o = WidgetsAppOpts{
		Key: o.Key, Builder: o.Builder, Title: o.Title, OnGenerateTitle: o.OnGenerateTitle,
		OnNavigationNotification: o.OnNavigationNotification, TextStyle: o.TextStyle, Locale: o.Locale,
		LocalizationsDelegates: o.LocalizationsDelegates, LocaleListResolutionCallback: o.LocaleListResolutionCallback,
		LocaleResolutionCallback: o.LocaleResolutionCallback, SupportedLocales: o.SupportedLocales,
		Shortcuts: o.Shortcuts, Actions: o.Actions,
	}
	return w
}

// Color is the primary color of the app, given to the Title.
//
// Flutter: widgets/app.dart WidgetsApp.color
func (w *WidgetsApp) Color() ui.Color { return w.color }

// SupportedLocales are the locales the app supports.
//
// Flutter: widgets/app.dart WidgetsApp.supportedLocales
func (w *WidgetsApp) SupportedLocales() []ui.Locale { return w.o.SupportedLocales }

// Shortcuts is the map that replaces the default shortcuts, or nil.
//
// Flutter: widgets/app.dart WidgetsApp.shortcuts
func (w *WidgetsApp) Shortcuts() ShortcutMap { return w.o.Shortcuts }

// Actions is the map that replaces the default actions, or nil.
//
// Flutter: widgets/app.dart WidgetsApp.actions
func (w *WidgetsApp) Actions() ActionMap { return w.o.Actions }

// CreateState creates the state.
//
// Flutter: widgets/app.dart WidgetsApp.createState
func (w *WidgetsApp) CreateState() State { return &widgetsAppState{} }

func shortcutEntries(entries ...ShortcutEntry) ShortcutMap { return entries }

func appShortcut(k services.LogicalKeyboardKey, o SingleActivatorOpts, intent Intent) ShortcutEntry {
	return ShortcutEntry{Activator: NewSingleActivator(k, o), Intent: intent}
}

// WidgetsAppDefaultShortcuts returns the shortcuts that WidgetsApp uses when it
// is not given any, for the current target platform: activation, dismissal, the
// focus traversal by Tab and the arrow keys, and scrolling.
//
// Flutter: widgets/app.dart WidgetsApp.defaultShortcuts
func WidgetsAppDefaultShortcuts() ShortcutMap {
	none := SingleActivatorOpts{}
	var apple bool
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		apple = true
	}
	scroll := func(k services.LogicalKeyboardKey, dir painting.AxisDirection) ShortcutEntry {
		o := SingleActivatorOpts{Control: true}
		if apple {
			o = SingleActivatorOpts{Meta: true}
		}
		return appShortcut(k, o, ScrollIntent{Direction: dir})
	}
	m := shortcutEntries(
		appShortcut(services.LogicalKeyboardKeyEnter, none, ActivateIntent{}),
		appShortcut(services.LogicalKeyboardKeyNumpadEnter, none, ActivateIntent{}),
		appShortcut(services.LogicalKeyboardKeySpace, none, ActivateIntent{}),
	)
	if !apple {
		m = append(m,
			appShortcut(services.LogicalKeyboardKeyGameButtonA, none, ActivateIntent{}),
			appShortcut(services.LogicalKeyboardKeySelect, none, ActivateIntent{}),
		)
	}
	m = append(m,
		appShortcut(services.LogicalKeyboardKeyEscape, none, DismissIntent{}),

		appShortcut(services.LogicalKeyboardKeyTab, none, NextFocusIntent{}),
		appShortcut(services.LogicalKeyboardKeyTab, SingleActivatorOpts{Shift: true}, PreviousFocusIntent{}),
		appShortcut(services.LogicalKeyboardKeyArrowLeft, none, DirectionalFocusIntent{Direction: TraversalDirectionLeft}),
		appShortcut(services.LogicalKeyboardKeyArrowRight, none, DirectionalFocusIntent{Direction: TraversalDirectionRight}),
		appShortcut(services.LogicalKeyboardKeyArrowDown, none, DirectionalFocusIntent{Direction: TraversalDirectionDown}),
		appShortcut(services.LogicalKeyboardKeyArrowUp, none, DirectionalFocusIntent{Direction: TraversalDirectionUp}),

		scroll(services.LogicalKeyboardKeyArrowUp, painting.AxisDirectionUp),
		scroll(services.LogicalKeyboardKeyArrowDown, painting.AxisDirectionDown),
		scroll(services.LogicalKeyboardKeyArrowLeft, painting.AxisDirectionLeft),
		scroll(services.LogicalKeyboardKeyArrowRight, painting.AxisDirectionRight),
		appShortcut(services.LogicalKeyboardKeyPageUp, none, ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}),
		appShortcut(services.LogicalKeyboardKeyPageDown, none, ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}),
	)
	return m
}

// WidgetsAppDefaultActions returns the actions that WidgetsApp uses when it is
// not given any. Every call makes new actions, so apps can change the map.
//
// Deviation: a function that returns a new map, where Flutter has a static
// mutable map (SPEC §12.10).
//
// Flutter: widgets/app.dart WidgetsApp.defaultActions
func WidgetsAppDefaultActions() ActionMap {
	return ActionMap{
		IntentTypeOf[DoNothingIntent]():                   NewDoNothingAction(),
		IntentTypeOf[DoNothingAndStopPropagationIntent](): NewDoNothingAction(DoNothingActionOpts{DoesNotConsumeKey: true}),
		IntentTypeOf[RequestFocusIntent]():                NewRequestFocusAction(),
		IntentTypeOf[NextFocusIntent]():                   NewNextFocusAction(),
		IntentTypeOf[PreviousFocusIntent]():               NewPreviousFocusAction(),
		IntentTypeOf[DirectionalFocusIntent]():            NewDirectionalFocusAction(),
		IntentTypeOf[ScrollIntent]():                      NewScrollAction(),
		IntentTypeOf[PrioritizedIntents]():                NewPrioritizedAction(),
		IntentTypeOf[VoidCallbackIntent]():                NewVoidCallbackAction(),
	}
}

type widgetsAppState struct {
	StateOf[*WidgetsApp]
	WidgetsBindingObserverBase

	navigator                   AnyGlobalKey
	defaultRouteInformationProv *PlatformRouteInformationProvider
	defaultBackButtonDispatcher *RootBackButtonDispatcher
	localizations               *LocalizationsResolver
	lifecycleState              ui.AppLifecycleState
}

func (s *widgetsAppState) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

func (s *widgetsAppState) initialRouteName() string {
	if r := s.Widget().o.InitialRoute; r != "" {
		return r
	}
	return NavigatorDefaultRouteName
}

func (s *widgetsAppState) defaultOnNavigationNotification(n NavigationNotification) bool {
	switch s.lifecycleState {
	case ui.AppLifecycleStateUnspecified, ui.AppLifecycleStateDetached:
		// Avoid updating the engine when the app isn't ready.
		return true
	}
	s.binding().SystemNavigator().SetFrameworkHandlesBack(n.CanHandlePop)
	return true
}

// DidChangeAppLifecycleState remembers the state.
//
// Flutter: widgets/app.dart _WidgetsAppState.didChangeAppLifecycleState
func (s *widgetsAppState) DidChangeAppLifecycleState(state ui.AppLifecycleState) {
	s.lifecycleState = state
}

func (s *widgetsAppState) InitState() {
	b := s.binding()
	s.localizations = s.newLocalizationsResolver(b)
	s.updateRouting(nil)
	b.AddObserver(s)
	s.lifecycleState = b.Scheduler().LifecycleState()
}

func (s *widgetsAppState) newLocalizationsResolver(b *WidgetsBinding) *LocalizationsResolver {
	w := s.Widget()
	return NewLocalizationsResolver(b, w.o.SupportedLocales, LocalizationsResolverOpts{
		Locale: w.o.Locale, LocaleListResolutionCallback: w.o.LocaleListResolutionCallback,
		LocaleResolutionCallback: w.o.LocaleResolutionCallback, LocalizationsDelegates: w.o.LocalizationsDelegates,
	})
}

func (s *widgetsAppState) DidUpdateWidget(old *WidgetsApp) {
	s.updateRouting(old)
	w := s.Widget()
	if shouldUpdateLocalizations(w, old) {
		s.localizations.Update(w.o.SupportedLocales, LocalizationsResolverOpts{
			Locale: w.o.Locale, LocaleListResolutionCallback: w.o.LocaleListResolutionCallback,
			LocaleResolutionCallback: w.o.LocaleResolutionCallback, LocalizationsDelegates: w.o.LocalizationsDelegates,
		})
	}
}

func shouldUpdateLocalizations(w, old *WidgetsApp) bool {
	return w.o.Locale != old.o.Locale ||
		!slices.Equal(w.o.SupportedLocales, old.o.SupportedLocales) ||
		len(w.o.LocalizationsDelegates) != len(old.o.LocalizationsDelegates) ||
		!slices.Equal(w.o.LocalizationsDelegates, old.o.LocalizationsDelegates)
}

func (s *widgetsAppState) Dispose() {
	s.binding().RemoveObserver(s)
	if s.defaultRouteInformationProv != nil {
		s.defaultRouteInformationProv.Dispose()
	}
	s.localizations.Dispose()
}

func (s *widgetsAppState) usesRouterWithDelegates() bool {
	r := s.Widget().router
	return r != nil && r.hasDelegate
}

func (s *widgetsAppState) usesRouterWithConfig() bool {
	r := s.Widget().router
	return r != nil && r.hasConfig
}

func (s *widgetsAppState) usesNavigator() bool {
	w := s.Widget()
	return w.router == nil && (w.o.Home != nil || len(w.o.Routes) > 0 || w.o.OnGenerateRoute != nil || w.o.OnUnknownRoute != nil)
}

func (s *widgetsAppState) clearRouterResources() {
	if s.defaultRouteInformationProv != nil {
		s.defaultRouteInformationProv.Dispose()
	}
	s.defaultRouteInformationProv = nil
	s.defaultBackButtonDispatcher = nil
}

func (s *widgetsAppState) updateRouting(old *WidgetsApp) {
	w := s.Widget()
	switch {
	case s.usesRouterWithDelegates():
		s.navigator = nil
		if w.router.provider == nil && w.router.hasParser {
			if s.defaultRouteInformationProv == nil {
				initial, err := url.Parse(s.initialRouteName())
				if err != nil {
					initial = &url.URL{Path: NavigatorDefaultRouteName}
				}
				s.defaultRouteInformationProv = NewPlatformRouteInformationProvider(s.binding(),
					NewRouteInformation(RouteInformationOpts{URI: initial}))
			}
		} else if s.defaultRouteInformationProv != nil {
			s.defaultRouteInformationProv.Dispose()
			s.defaultRouteInformationProv = nil
		}
		if w.router.dispatcher == nil && s.defaultBackButtonDispatcher == nil {
			s.defaultBackButtonDispatcher = NewRootBackButtonDispatcher(s.binding())
		}
	case s.usesNavigator():
		s.clearRouterResources()
		if s.navigator == nil || (old != nil && w.o.NavigatorKey != old.o.NavigatorKey) {
			if w.o.NavigatorKey != nil {
				s.navigator = w.o.NavigatorKey
			} else {
				s.navigator = NewGlobalObjectKey[*NavigatorState](s)
			}
		}
	default:
		s.clearRouterResources()
		s.navigator = nil
	}
}

func (s *widgetsAppState) navigatorState() *NavigatorState {
	if s.navigator == nil {
		return nil
	}
	nav, _ := currentStateOf[*NavigatorState](s.navigator.currentElementIn(s.binding().BuildOwner()))
	return nav
}

func (s *widgetsAppState) onGenerateRoute(settings RouteSettings) Route {
	w := s.Widget()
	name := settings.Name()
	var content WidgetBuilder
	if name == NavigatorDefaultRouteName && w.o.Home != nil {
		// The page reads the home of the current widget each time it is built.
		content = func(BuildContext) Widget { return s.Widget().o.Home }
	} else {
		content = w.o.Routes[name]
	}
	if content != nil {
		if w.o.PageRouteBuilder == nil {
			panic(errors.New("WidgetsAppPageRouteBuilderMissing", "the default onGenerateRoute handler for WidgetsApp must have a pageRouteBuilder set if the home or routes properties are set", "route", name))
		}
		return w.o.PageRouteBuilder(settings, content)
	}
	if w.o.OnGenerateRoute != nil {
		return w.o.OnGenerateRoute(settings)
	}
	return nil
}

func (s *widgetsAppState) onUnknownRoute(settings RouteSettings) Route {
	w := s.Widget()
	if w.o.OnUnknownRoute == nil {
		if foundation.DebugMode {
			panic(errors.New("WidgetsAppUnknownRouteMissing",
				`could not find a generator for the route in the WidgetsApp. Generators for routes are searched for in the following order: 1. For the "/" route, the "home" property, if non-null, is used. 2. Otherwise, the "routes" table is used, if it has an entry for the route. 3. Otherwise, onGenerateRoute is called. It should return a non-null value for any valid route not handled by "home" and "routes". 4. Finally if all else fails onUnknownRoute is called. Unfortunately, onUnknownRoute was not set`,
				"route", settings.Name()))
		}
		return nil
	}
	route := w.o.OnUnknownRoute(settings)
	if foundation.DebugMode && route == nil {
		panic(errors.New("WidgetsAppUnknownRouteNil", "the onUnknownRoute callback returned nil; such callbacks must never return nil", "route", settings.Name()))
	}
	return route
}

// DidPopRoute pops the navigator, when the app uses one: the back button.
//
// Flutter: widgets/app.dart _WidgetsAppState.didPopRoute
func (s *widgetsAppState) DidPopRoute() bool {
	// The back button dispatcher handles the pop when the app uses a router.
	if s.usesRouterWithDelegates() {
		return false
	}
	nav := s.navigatorState()
	if nav == nil {
		return false
	}
	return nav.MaybePop()
}

// DidPushRouteInformation pushes the named route of the location, when the app
// uses a navigator.
//
// Flutter: widgets/app.dart _WidgetsAppState.didPushRouteInformation
func (s *widgetsAppState) DidPushRouteInformation(info *RouteInformation) bool {
	if s.usesRouterWithDelegates() {
		return false
	}
	nav := s.navigatorState()
	if nav == nil {
		return false
	}
	uri := info.URI()
	path := uri.Path
	if path == "" {
		path = "/"
	}
	name := (&url.URL{Path: path, RawQuery: uri.RawQuery, Fragment: uri.Fragment}).String()
	if decoded, err := url.QueryUnescape(name); err == nil {
		name = decoded
	}
	nav.PushNamed(name)
	return true
}

func (s *widgetsAppState) Build(bc BuildContext) Widget {
	w := s.Widget()
	var routing Widget
	switch {
	case s.usesRouterWithDelegates():
		r := w.router
		provider := r.provider
		if provider == nil && s.defaultRouteInformationProv != nil {
			provider = s.defaultRouteInformationProv
		}
		dispatcher := r.dispatcher
		if dispatcher == nil {
			dispatcher = s.defaultBackButtonDispatcher
		}
		routing = r.build(provider, dispatcher)
	case s.usesNavigator():
		var initialRoutes RouteListFactory
		if w.o.OnGenerateInitialRoutes != nil {
			f := w.o.OnGenerateInitialRoutes
			initialRoutes = func(_ *NavigatorState, name string) []Route { return f(name) }
		}
		routing = NewFocusScope(
			NewNavigator(NavigatorOpts{
				Key:                        s.navigator,
				ClipBehavior:               ui.ClipNone,
				InitialRoute:               s.initialRouteName(),
				OnGenerateRoute:            s.onGenerateRoute,
				OnGenerateInitialRoutes:    initialRoutes,
				OnUnknownRoute:             s.onUnknownRoute,
				Observers:                  w.o.NavigatorObservers,
				RouteTraversalEdgeBehavior: TraversalEdgeBehaviorParentScope,
				ReportsRouteUpdateToEngine: true,
			}),
			FocusScopeOpts{DebugLabel: "Navigator Scope", Autofocus: true})
	case s.usesRouterWithConfig():
		routing = w.router.build(nil, nil)
	}

	var result Widget
	if w.o.Builder != nil {
		builder := w.o.Builder
		routed := routing
		result = NewBuilder(func(bc BuildContext) Widget { return builder(bc, routed) })
	} else {
		result = routing
	}
	if w.o.TextStyle != nil {
		result = NewDefaultTextStyle(w.o.TextStyle, result)
	}

	opaque := w.color.WithOpacity(1)
	if w.o.OnGenerateTitle != nil {
		inner, onGenerateTitle := result, w.o.OnGenerateTitle
		// This Builder provides a context below the Localizations widget, so that
		// the callback can refer to Localizations through it.
		result = NewBuilder(func(bc BuildContext) Widget {
			return NewTitle(opaque, inner, TitleOpts{Title: onGenerateTitle(bc)})
		})
	} else {
		result = NewTitle(opaque, result, TitleOpts{Title: w.o.Title})
	}

	shortcuts := w.o.Shortcuts
	if shortcuts == nil {
		shortcuts = WidgetsAppDefaultShortcuts()
	}
	actions := w.o.Actions
	if actions == nil {
		actions = WidgetsAppDefaultActions()
		actions[IntentTypeOf[ScrollIntent]()] = NewOverridableAction(NewScrollAction(), bc)
	}
	onNavigation := w.o.OnNavigationNotification
	if onNavigation == nil {
		onNavigation = s.defaultOnNavigationNotification
	}
	resolver, content := s.localizations, result
	return NewNotificationListener[NavigationNotification](
		// DefaultTextEditingShortcuts is nested inside Shortcuts so that it can fall
		// through to the default shortcuts.
		NewShortcuts(shortcuts,
			NewDefaultTextEditingShortcuts(
				NewActions(actions,
					NewFocusTraversalGroup(
						NewShortcutRegistrar(
							NewListenableBuilder(resolver, func(bc BuildContext, _ Widget) Widget {
								return NewLocalizations(resolver.Locale(), resolver.LocalizationsDelegates(), content,
									LocalizationsOpts{IsApplicationLevel: true})
							}, nil)),
						FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()}))),
			ShortcutsOpts{DebugLabel: "<Default WidgetsApp Shortcuts>"}),
		onNavigation)
}
