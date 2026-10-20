package material_test

import (
	"net/url"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/widgets"
)

// The router helpers of app_test.dart: the parser and the delegate show the path
// of the route information as a text, and a pop sets the route information
// "popped".

type routeInfo = widgets.RouteInformation

func uriInfo(s string) *routeInfo {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return widgets.NewRouteInformation(widgets.RouteInformationOpts{URI: u})
}

type simpleParser struct {
	widgets.RouteInformationParserBase[*routeInfo]
}

func newSimpleParser() *simpleParser {
	p := &simpleParser{}
	widgets.InitRouteInformationParserBase[*routeInfo](&p.RouteInformationParserBase, p)
	return p
}

func (p *simpleParser) ParseRouteInformation(info *routeInfo) *widgets.Future[*routeInfo] {
	return widgets.NewSynchronousFuture(info)
}

func (p *simpleParser) RestoreRouteInformation(c *routeInfo) *routeInfo { return c }

// simpleDelegate is SimpleNavigatorRouterDelegate: a Navigator with a base page
// and the page of the route information; popping the page makes the route
// information "popped".
type simpleDelegate struct {
	widgets.RouterDelegateBase[*routeInfo]
	navigatorKey *widgets.GlobalKey[*widgets.NavigatorState]
	info         *routeInfo
}

func newSimpleDelegate(initial string) *simpleDelegate {
	d := &simpleDelegate{navigatorKey: widgets.NewGlobalKey[*widgets.NavigatorState]()}
	if initial != "" {
		d.info = uriInfo(initial)
	}
	widgets.InitRouterDelegateBase[*routeInfo](&d.RouterDelegateBase, d)
	return d
}

func (d *simpleDelegate) SetNewRoutePath(c *routeInfo) *widgets.Future[struct{}] {
	d.info = c
	return widgets.NewSynchronousFuture(struct{}{})
}

func (d *simpleDelegate) PopRoute() bool {
	return widgets.PopNavigatorRouterDelegatePopRoute(d.navigatorKey)
}

func page(name string, child widgets.Widget) widgets.Page {
	return NewMaterialPage(child, MaterialPageOpts{PageOpts: widgets.PageOpts{Name: name, Key: foundation.NewValueKey(name)}})
}

func (d *simpleDelegate) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewNavigator(widgets.NavigatorOpts{
		Key: d.navigatorKey,
		OnPopPage: func(route widgets.Route, result any) bool {
			d.info = uriInfo("popped")
			d.NotifyListeners()
			return route.DidPop(result)
		},
		Pages: []widgets.Page{
			// At least two pages are needed for the pop to propagate through.
			page("", widgets.NewText("base")),
			page(d.info.URI().String(), widgets.NewText(d.info.URI().String())),
		},
	})
}

func shows(ts *gluttertest.Tester, text string) bool {
	return ts.Count(gluttertest.ByText(text)) == 1
}

func TestMaterialAppRouter(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ts := gluttertest.New(t)
		provider := widgets.NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		delegate := newSimpleDelegate("")
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{
			RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), RouterDelegate: delegate,
		}))
		if !shows(ts, "initial") {
			t.Fatal("initial should show")
		}
		// The back button of the platform.
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if !shows(ts, "popped") {
			t.Error("popped should show")
		}
	})

	t.Run("works with onNavigationNotification", func(t *testing.T) {
		ts := gluttertest.New(t)
		provider := widgets.NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		count := 0
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{
			RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), RouterDelegate: newSimpleDelegate(""),
			OnNavigationNotification: func(widgets.NavigationNotification) bool { count++; return true },
		}))
		if !shows(ts, "initial") || count == 0 {
			t.Fatalf("initial shown %v, notifications %d", shows(ts, "initial"), count)
		}
		afterBuild := count
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if !shows(ts, "popped") || count <= afterBuild {
			t.Errorf("popped shown %v, notifications %d after %d", shows(ts, "popped"), count, afterBuild)
		}
	})

	t.Run("the route information parser is optional", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{RouterDelegate: newSimpleDelegate("initial")}))
		if !shows(ts, "initial") {
			t.Fatal("initial should show")
		}
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if !shows(ts, "popped") {
			t.Error("popped should show")
		}
	})

	t.Run("a router config works", func(t *testing.T) {
		ts := gluttertest.New(t)
		provider := widgets.NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		config := &widgets.RouterConfig[*routeInfo]{
			RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), RouterDelegate: newSimpleDelegate(""),
			BackButtonDispatcher: widgets.NewRootBackButtonDispatcher(ts.Binding),
		}
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{RouterConfig: config}))
		if !shows(ts, "initial") {
			t.Fatal("initial should show")
		}
		ts.Binding.HandlePopRoute()
		ts.PumpAndSettle()
		if !shows(ts, "popped") {
			t.Error("popped should show")
		}
	})

	t.Run("a provider without a parser is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		provider := widgets.NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
		defer provider.Dispose()
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{RouteInformationProvider: provider, RouterDelegate: newSimpleDelegate("initial")}))
		err := ts.TakeError()
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("WidgetsAppRouterParserMissing") {
			t.Errorf("unexpected error %v", err)
		}
		ts.TakeErrors()
	})

	t.Run("a router config with other delegates is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		delegate := newSimpleDelegate("initial")
		ts.PumpWidget(NewMaterialAppRouter(MaterialAppRouterOpts[*routeInfo]{RouterDelegate: delegate, RouterConfig: &widgets.RouterConfig[*routeInfo]{RouterDelegate: delegate}}))
		err := ts.TakeError()
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("WidgetsAppRouterConfigConflict") {
			t.Errorf("unexpected error %v", err)
		}
		ts.TakeErrors()
	})
}
