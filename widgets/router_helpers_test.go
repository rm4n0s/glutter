package widgets_test

import (
	"net/url"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// The helper classes of packages/flutter/test/widgets/router_test.dart.

// uriInfo is RouteInformation(uri: Uri.parse(s)).
func uriInfo(s string) *RouteInformation {
	return stateInfo(s, nil)
}

func stateInfo(s string, state any) *RouteInformation {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return NewRouteInformation(RouteInformationOpts{URI: u, State: state})
}

// decoded is Uri.decodeComponent(information.uri.toString()).
func decoded(info *RouteInformation) string {
	s, err := url.PathUnescape(info.URI().String())
	if err != nil {
		panic(err)
	}
	return s
}

// simpleParser is SimpleRouteInformationParser.
type simpleParser struct {
	RouteInformationParserBase[*RouteInformation]
}

func newSimpleParser() *simpleParser {
	p := &simpleParser{}
	InitRouteInformationParserBase[*RouteInformation](&p.RouteInformationParserBase, p)
	return p
}

func (p *simpleParser) ParseRouteInformation(info *RouteInformation) *Future[*RouteInformation] {
	return NewSynchronousFuture(info)
}

func (p *simpleParser) RestoreRouteInformation(c *RouteInformation) *RouteInformation { return c }

// customParser is CustomRouteInformationParser.
type customParser struct {
	RouteInformationParserBase[*RouteInformation]
	callback func(info *RouteInformation, bc BuildContext) *RouteInformation
}

func newCustomParser(callback func(*RouteInformation, BuildContext) *RouteInformation) *customParser {
	p := &customParser{callback: callback}
	InitRouteInformationParserBase[*RouteInformation](&p.RouteInformationParserBase, p)
	return p
}

func (p *customParser) ParseRouteInformationWithDependencies(info *RouteInformation, bc BuildContext) *Future[*RouteInformation] {
	return NewSynchronousFuture(p.callback(info, bc))
}

func (p *customParser) RestoreRouteInformation(c *RouteInformation) *RouteInformation { return c }

// redirectingParser is RedirectingInformationParser.
type redirectingParser struct {
	RouteInformationParserBase[*RouteInformation]
	redirects map[string]*RouteInformation
}

func newRedirectingParser(redirects map[string]*RouteInformation) *redirectingParser {
	p := &redirectingParser{redirects: redirects}
	InitRouteInformationParserBase[*RouteInformation](&p.RouteInformationParserBase, p)
	return p
}

func (p *redirectingParser) ParseRouteInformation(info *RouteInformation) *Future[*RouteInformation] {
	if r, ok := p.redirects[info.URI().String()]; ok {
		return NewSynchronousFuture(r)
	}
	return NewSynchronousFuture(info)
}

func (p *redirectingParser) RestoreRouteInformation(c *RouteInformation) *RouteInformation { return c }

// asyncParser is SimpleAsyncRouteInformationParser and
// CompleterRouteInformationParser: the test completes the futures it makes.
type asyncParser struct {
	RouteInformationParserBase[*RouteInformation]
	pending []*Future[*RouteInformation]
	infos   []*RouteInformation
}

func newAsyncParser() *asyncParser {
	p := &asyncParser{}
	InitRouteInformationParserBase[*RouteInformation](&p.RouteInformationParserBase, p)
	return p
}

func (p *asyncParser) ParseRouteInformation(info *RouteInformation) *Future[*RouteInformation] {
	f := NewFuture[*RouteInformation]()
	p.pending = append(p.pending, f)
	p.infos = append(p.infos, info)
	return f
}

func (p *asyncParser) RestoreRouteInformation(c *RouteInformation) *RouteInformation { return c }

// complete completes the n-th parse with the information it was asked about.
func (p *asyncParser) complete(n int) { p.pending[n].Complete(p.infos[n]) }

// simpleDelegate is SimpleRouterDelegate.
type simpleDelegate struct {
	RouterDelegateBase[*RouteInformation]
	routeInformation    *RouteInformation
	builder             func(bc BuildContext, info *RouteInformation) Widget
	onPopRoute          func() bool
	reportConfiguration bool
	newRoutePaths       int
}

func newSimpleDelegate(builder func(BuildContext, *RouteInformation) Widget, onPopRoute func() bool) *simpleDelegate {
	d := &simpleDelegate{builder: builder, onPopRoute: onPopRoute}
	InitRouterDelegateBase[*RouteInformation](&d.RouterDelegateBase, d)
	return d
}

func (d *simpleDelegate) setRouteInformation(info *RouteInformation) {
	d.routeInformation = info
	d.NotifyListeners()
}

func (d *simpleDelegate) CurrentConfiguration() foundation.Opt[*RouteInformation] {
	if d.reportConfiguration {
		return foundation.Some(d.routeInformation)
	}
	return foundation.Opt[*RouteInformation]{}
}

func (d *simpleDelegate) SetNewRoutePath(c *RouteInformation) *Future[struct{}] {
	d.routeInformation = c
	d.newRoutePaths++
	return NewSynchronousFuture(struct{}{})
}

func (d *simpleDelegate) PopRoute() bool {
	if d.onPopRoute != nil {
		return d.onPopRoute()
	}
	return true
}

func (d *simpleDelegate) Build(bc BuildContext) Widget { return d.builder(bc, d.routeInformation) }

// asyncDelegate is SimpleAsyncRouterDelegate: SetNewRoutePath returns a future the
// test completes.
type asyncDelegate struct {
	RouterDelegateBase[*RouteInformation]
	routeInformation *RouteInformation
	builder          func(bc BuildContext, info *RouteInformation) Widget
	setNewRoute      []*Future[struct{}]
}

func newAsyncDelegate(builder func(BuildContext, *RouteInformation) Widget) *asyncDelegate {
	d := &asyncDelegate{builder: builder}
	InitRouterDelegateBase[*RouteInformation](&d.RouterDelegateBase, d)
	return d
}

func (d *asyncDelegate) SetNewRoutePath(c *RouteInformation) *Future[struct{}] {
	d.routeInformation = c
	f := NewFuture[struct{}]()
	d.setNewRoute = append(d.setNewRoute, f)
	return f
}

func (d *asyncDelegate) PopRoute() bool { return true }

func (d *asyncDelegate) Build(bc BuildContext) Widget { return d.builder(bc, d.routeInformation) }

// mutableDelegate is MutableRouterDelegate.
type mutableDelegate struct {
	RouterDelegateBase[*RouteInformation]
	configuration *RouteInformation
}

func newMutableDelegate() *mutableDelegate {
	d := &mutableDelegate{}
	InitRouterDelegateBase[*RouteInformation](&d.RouterDelegateBase, d)
	return d
}

func (d *mutableDelegate) CurrentConfiguration() foundation.Opt[*RouteInformation] {
	if d.configuration == nil {
		return foundation.Opt[*RouteInformation]{}
	}
	return foundation.Some(d.configuration)
}

func (d *mutableDelegate) SetNewRoutePath(c *RouteInformation) *Future[struct{}] {
	d.configuration = c
	return NewSynchronousFuture(struct{}{})
}

func (d *mutableDelegate) updateConfiguration(c *RouteInformation) {
	d.configuration = c
	d.NotifyListeners()
}

func (d *mutableDelegate) PopRoute() bool { panic("unimplemented") }

func (d *mutableDelegate) Build(BuildContext) Widget {
	if d.configuration == nil {
		return NewText("")
	}
	return NewText(d.configuration.URI().String())
}

// navigatorDelegate is SimpleNavigatorRouterDelegate.
type navigatorDelegate struct {
	RouterDelegateBase[*RouteInformation]
	navigatorKey     *GlobalKey[*NavigatorState]
	routeInformation *RouteInformation
	builder          func(bc BuildContext, info *RouteInformation) Widget
	onPopPage        func(route Route, result any) bool
}

func newNavigatorDelegate(builder func(BuildContext, *RouteInformation) Widget, onPopPage func(Route, any) bool) *navigatorDelegate {
	d := &navigatorDelegate{navigatorKey: NewGlobalKey[*NavigatorState](), builder: builder, onPopPage: onPopPage}
	InitRouterDelegateBase[*RouteInformation](&d.RouterDelegateBase, d)
	return d
}

func (d *navigatorDelegate) SetNewRoutePath(c *RouteInformation) *Future[struct{}] {
	d.routeInformation = c
	return NewSynchronousFuture(struct{}{})
}

func (d *navigatorDelegate) PopRoute() bool {
	return PopNavigatorRouterDelegatePopRoute(d.navigatorKey)
}

func (d *navigatorDelegate) Build(bc BuildContext) Widget {
	return NewNavigator(NavigatorOpts{
		Key:       d.navigatorKey,
		OnPopPage: d.onPopPage,
		Pages: []Page{
			// At least two pages are needed for the pop to propagate through.
			newMaterialPage("", NewText("base")),
			newMaterialPage(d.routeInformation.URI().String(), d.builder(bc, d.routeInformation)),
		},
	})
}

// simpleProvider is SimpleRouteInformationProvider.
type simpleProvider struct {
	foundation.ChangeNotifier
	RouteInformationProviderBase
	onRouterReport func(info *RouteInformation, t RouteInformationReportingType)
	value          *RouteInformation
}

func newSimpleProvider(initial string) *simpleProvider {
	return &simpleProvider{value: uriInfo(initial)}
}

func (p *simpleProvider) Value() *RouteInformation { return p.value }

func (p *simpleProvider) setValue(v *RouteInformation) {
	p.value = v
	p.NotifyListeners()
}

func (p *simpleProvider) RouterReportsNewRouteInformation(info *RouteInformation, t RouteInformationReportingType) {
	p.value = info
	if p.onRouterReport != nil {
		p.onRouterReport(info, t)
	}
}

// intInherited is IntInheritedNotifier.
type intInherited struct{ InheritedNotifier }

func newIntInherited(n *foundation.ValueNotifier[int], child Widget) *intInherited {
	return &intInherited{NewInheritedNotifierBase(n, child)}
}

func intInheritedOf(bc BuildContext) int {
	w, _ := DependOn[*intInherited](bc)
	return w.Notifier().(*foundation.ValueNotifier[int]).Value()
}

// testButton is TestButton and _StatefulTestButton: a tappable child.
func testButton(child Widget, onPressed func()) Widget {
	return NewGestureDetector(child, GestureDetectorOpts{OnTap: onPressed, Behavior: rendering.HitTestBehaviorOpaque})
}
