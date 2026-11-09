package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoTabViewOpts are the optional parameters of NewCupertinoTabView.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView.new
type CupertinoTabViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Builder builds the widget of the "/" route, the first page of the tab.
	Builder widgets.WidgetBuilder
	// NavigatorKey is the key of the navigator of the tab; the default is a key that the state owns.
	NavigatorKey *widgets.GlobalKey[*widgets.NavigatorState]
	// DefaultTitle is the title of the "/" route (the label of the back button of the page above it).
	DefaultTitle foundation.Opt[string]
	// Routes is the table of the routes by name.
	Routes map[string]widgets.WidgetBuilder
	// OnGenerateRoute makes the route for a name that Builder and Routes do not handle.
	OnGenerateRoute widgets.RouteFactory
	// OnUnknownRoute makes the route when everything else failed.
	OnUnknownRoute widgets.RouteFactory
	// NavigatorObservers are the observers of the navigator of the tab; the hero controller of the tab is added.
	NavigatorObservers []widgets.NavigatorObserver
	// Deviation: no RestorationScopeId (state restoration is a non-goal, SPEC §2).
}

// CupertinoTabView is a tab with its own Navigator and history, for the tabs of a CupertinoTabScaffold: the
// pages that a tab pushes stay when the user goes to another tab. The route of its Builder is "/"; the other
// routes come from Routes and OnGenerateRoute, and are CupertinoPageRoutes.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView
type CupertinoTabView struct {
	widgets.StatefulWidget
	o CupertinoTabViewOpts
}

// NewCupertinoTabView creates a tab view.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView.new
func NewCupertinoTabView(opts ...CupertinoTabViewOpts) *CupertinoTabView {
	o := firstOf(opts)
	return &CupertinoTabView{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o}
}

// Builder builds the first page, or is nil.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView.builder
func (w *CupertinoTabView) Builder() widgets.WidgetBuilder { return w.o.Builder }

// DefaultTitle is the title of the first page.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView.defaultTitle
func (w *CupertinoTabView) DefaultTitle() foundation.Opt[string] { return w.o.DefaultTitle }

// CreateState creates the state.
//
// Flutter: cupertino/tab_view.dart CupertinoTabView.createState
func (w *CupertinoTabView) CreateState() widgets.State { return &cupertinoTabViewState{} }

type cupertinoTabViewState struct {
	widgets.StateOf[*CupertinoTabView]
	heroController     *widgets.HeroController
	navigatorObservers []widgets.NavigatorObserver
	ownedNavigatorKey  *widgets.GlobalKey[*widgets.NavigatorState]
}

func (s *cupertinoTabViewState) InitState() {
	s.heroController = CupertinoAppCreateCupertinoHeroController()
	s.updateObservers()
}

func (s *cupertinoTabViewState) DidUpdateWidget(old *CupertinoTabView) {
	w := s.Widget()
	if w.o.NavigatorKey != old.o.NavigatorKey || len(w.o.NavigatorObservers) != len(old.o.NavigatorObservers) {
		s.updateObservers()
	}
}

func (s *cupertinoTabViewState) Dispose() { s.heroController.Dispose() }

func (s *cupertinoTabViewState) updateObservers() {
	s.navigatorObservers = append(append([]widgets.NavigatorObserver(nil), s.Widget().o.NavigatorObservers...), s.heroController)
}

func (s *cupertinoTabViewState) navigatorKey() *widgets.GlobalKey[*widgets.NavigatorState] {
	if k := s.Widget().o.NavigatorKey; k != nil {
		return k
	}
	if s.ownedNavigatorKey == nil {
		s.ownedNavigatorKey = widgets.NewGlobalKey[*widgets.NavigatorState]()
	}
	return s.ownedNavigatorKey
}

// isActive is whether this tab is the active one: the tickers of the other tabs are off.
func (s *cupertinoTabViewState) isActive() bool {
	return widgets.TickerModeValuesOf(s.Context()).Enabled
}

func (s *cupertinoTabViewState) Build(widgets.BuildContext) widgets.Widget {
	child := widgets.NewNavigator(widgets.NavigatorOpts{
		Key:             s.navigatorKey(),
		OnGenerateRoute: s.onGenerateRoute,
		OnUnknownRoute:  s.onUnknownRoute,
		Observers:       s.navigatorObservers,
	})
	// Handle system back gestures only if the tab is currently active.
	return widgets.NewNavigatorPopHandler(child, widgets.NavigatorPopHandlerOpts{
		Disabled: !s.isActive(),
		OnPopWithResult: func(any) {
			if !s.isActive() {
				return
			}
			if nav, ok := s.navigatorKey().CurrentState(); ok {
				nav.MaybePop()
			}
		},
	})
}

func (s *cupertinoTabViewState) onGenerateRoute(settings widgets.RouteSettings) widgets.Route {
	w := s.Widget()
	var routeBuilder widgets.WidgetBuilder
	title := foundation.None[string]()
	if settings.Name() == widgets.NavigatorDefaultRouteName && w.o.Builder != nil {
		routeBuilder = w.o.Builder
		title = w.o.DefaultTitle
	} else {
		routeBuilder = w.o.Routes[settings.Name()]
	}
	if routeBuilder != nil {
		return NewCupertinoPageRoute(routeBuilder, CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}, Title: title})
	}
	if w.o.OnGenerateRoute != nil {
		return w.o.OnGenerateRoute(settings)
	}
	return nil
}

func (s *cupertinoTabViewState) onUnknownRoute(settings widgets.RouteSettings) widgets.Route {
	w := s.Widget()
	if w.o.OnUnknownRoute == nil {
		panic(errors.New(tagCupertinoTabViewUnknownRouteMissing,
			`could not find a generator for the route in the CupertinoTabView. Generators for routes are searched for in the following order: 1. For the "/" route, the "builder" property, if non-null, is used. 2. Otherwise, the "routes" table is used, if it has an entry for the route. 3. Otherwise, onGenerateRoute is called. It should return a non-null value for any valid route not handled by "builder" and "routes". 4. Finally if all else fails onUnknownRoute is called. Unfortunately, onUnknownRoute was not set`,
			"route", settings.Name()))
	}
	result := w.o.OnUnknownRoute(settings)
	if result == nil {
		panic(errors.New(tagCupertinoTabViewUnknownRouteNil, "the onUnknownRoute callback returned nil; such callbacks must never return nil", "route", settings.Name()))
	}
	return result
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTabViewUnknownRouteMissing = "CupertinoTabViewUnknownRouteMissing"
	tagCupertinoTabViewUnknownRouteNil     = "CupertinoTabViewUnknownRouteNil"
)
