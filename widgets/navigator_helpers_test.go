package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The helpers of the navigator tests: the Go counterparts of observer_tester.dart,
// route_tester.dart and widgets_app_tester.dart of Flutter's test/widgets.

// navTestObserver is TestObserver.
type navTestObserver struct {
	NavigatorObserverBase
	onPushed, onPopped, onRemoved, onReplaced, onStartUserGesture func(route, previousRoute Route)
}

func (o *navTestObserver) DidPush(route, previousRoute Route) {
	if o.onPushed != nil {
		o.onPushed(route, previousRoute)
	}
}

func (o *navTestObserver) DidPop(route, previousRoute Route) {
	if o.onPopped != nil {
		o.onPopped(route, previousRoute)
	}
}

func (o *navTestObserver) DidRemove(route, previousRoute Route) {
	if o.onRemoved != nil {
		o.onRemoved(route, previousRoute)
	}
}

func (o *navTestObserver) DidReplace(newRoute, oldRoute Route) {
	if o.onReplaced != nil {
		o.onReplaced(newRoute, oldRoute)
	}
}

func (o *navTestObserver) DidStartUserGesture(route, previousRoute Route) {
	if o.onStartUserGesture != nil {
		o.onStartUserGesture(route, previousRoute)
	}
}

// navTestRouteOpts are the options of newNavTestRoute.
type navTestRouteOpts struct {
	child                     Widget
	builder                   WidgetBuilder
	settings                  RouteSettings
	barrierColor              foundation.Opt[ui.Color]
	maintainState             bool
	transitionDuration        time.Duration
	reverseTransitionDuration time.Duration
	transitionsBuilder        RouteTransitionsBuilder
	fullscreenDialog          bool
	disallowSnapshotting      bool
	barrierDismissible        bool
	delegatedTransition       DelegatedTransitionBuilder
}

// navTestRoute is TestRoute: a page route with a child or a builder, and no
// transition unless one is given.
type navTestRoute struct {
	PageRouteBase
	o navTestRouteOpts
}

func newNavTestRoute(o navTestRouteOpts) *navTestRoute {
	r := &navTestRoute{o: o}
	InitPageRouteBase(&r.PageRouteBase, r, PageRouteOpts{
		RouteOpts:            RouteOpts{Settings: o.settings},
		FullscreenDialog:     o.fullscreenDialog,
		DisallowSnapshotting: o.disallowSnapshotting,
		BarrierDismissible:   o.barrierDismissible,
	})
	return r
}

func (r *navTestRoute) TransitionDuration() time.Duration { return r.o.transitionDuration }
func (r *navTestRoute) ReverseTransitionDuration() time.Duration {
	return r.o.reverseTransitionDuration
}
func (r *navTestRoute) BarrierColor() foundation.Opt[ui.Color] { return r.o.barrierColor }
func (r *navTestRoute) BarrierLabel() string                   { return "" }
func (r *navTestRoute) MaintainState() bool                    { return r.o.maintainState }
func (r *navTestRoute) DelegatedTransition() DelegatedTransitionBuilder {
	return r.o.delegatedTransition
}

func (r *navTestRoute) BuildPage(bc BuildContext, a, s animation.Animation[float64]) Widget {
	switch {
	case r.o.child != nil:
		return r.o.child
	case r.o.builder != nil:
		return r.o.builder(bc)
	}
	return NewSizedBoxShrink(nil)
}

func (r *navTestRoute) BuildTransitions(bc BuildContext, a, s animation.Animation[float64], child Widget) Widget {
	if r.o.transitionsBuilder == nil {
		return child
	}
	return r.o.transitionsBuilder(bc, a, s, child)
}

// navPageRouteFactory is the PageRouteFactory of WidgetsApp.
type navPageRouteFactory func(settings RouteSettings, builder WidgetBuilder) PageRoute

// defaultNavPageRoute is the page route builder of TestWidgetsApp: no
// transition, and the default duration.
func defaultNavPageRoute(settings RouteSettings, builder WidgetBuilder) PageRoute {
	return NewPageRouteBuilder(
		func(bc BuildContext, a, s animation.Animation[float64]) Widget { return builder(bc) },
		PageRouteBuilderOpts{
			PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
			TransitionsBuilder: func(bc BuildContext, a, s animation.Animation[float64], child Widget) Widget {
				return child
			},
		})
}

// navTestAppOpts are the parameters of navTestApp.
type navTestAppOpts struct {
	navigatorKey    *GlobalKey[*NavigatorState]
	home            Widget
	homeBuilder     WidgetBuilder // like home, but read again whenever the page is built
	initialRoute    string
	onGenerateRoute RouteFactory
	observers       []NavigatorObserver
	routes          map[string]WidgetBuilder
	pageRoute       navPageRouteFactory
}

// navTestApp is TestWidgetsApp with a hero controller, as MaterialApp has one: a
// WidgetsApp whose page routes have the page route builder of the test (none by
// default).
func navTestApp(o navTestAppOpts) Widget {
	pageRoute := o.pageRoute
	if pageRoute == nil {
		pageRoute = defaultNavPageRoute
	}
	opts := WidgetsAppOpts{
		NavigatorObservers: o.observers,
		InitialRoute:       o.initialRoute,
		OnGenerateRoute:    o.onGenerateRoute,
		Routes:             o.routes,
		PageRouteBuilder:   func(settings RouteSettings, builder WidgetBuilder) PageRoute { return pageRoute(settings, builder) },
		Builder: func(_ BuildContext, child Widget) Widget {
			// MaterialApp has a hero controller, which also moves routes offstage
			// while they are measured.
			return NewHeroControllerScope(NewHeroController(), child)
		},
	}
	if o.navigatorKey != nil {
		opts.NavigatorKey = o.navigatorKey
	}
	switch {
	case o.homeBuilder != nil:
		opts.Routes = map[string]WidgetBuilder{NavigatorDefaultRouteName: o.homeBuilder}
		for k, v := range o.routes {
			opts.Routes[k] = v
		}
	case o.home != nil:
		opts.Home = o.home
	}
	opts.DebugShowCheckedModeBanner = foundation.Some(false)
	return NewWidgetsApp(ui.NewColor(0xFFFFFFFF), opts)
}

func navigatorKeyOf(k *GlobalKey[*NavigatorState]) foundation.Key {
	if k == nil {
		return nil
	}
	return k
}

// navText is a Text of a fixed size: the text of the tests has no width, as the
// test environment has no fonts, so it is given a box to be found and tapped.
func navText(s string) Widget {
	return NewSizedBox(NewText(s, TextOpts{TextDirection: ui.TextDirectionLTR}),
		SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)})
}

// navPush pushes the route through the state of the key.
func navPush(key *GlobalKey[*NavigatorState], route Route) <-chan any {
	state, _ := key.CurrentState()
	return state.Push(route)
}

func navState(t *testing.T, key *GlobalKey[*NavigatorState]) *NavigatorState {
	t.Helper()
	state, ok := key.CurrentState()
	if !ok {
		t.Fatal("the navigator is not in the tree")
	}
	return state
}

// textsFound is the number of Text widgets with the text, offstage ones
// included when offstage is set.
func textsFound(ts *gluttertest.Tester, text string, offstage bool) int {
	return ts.Count(gluttertest.ByText(text, gluttertest.FindOpts{IncludeOffstage: offstage}))
}

// navOnTapPage is OnTapPage without the Scaffold: a tappable page that shows its
// id.
func navOnTapPage(id string, onTap func()) Widget {
	return NewGestureDetector(NewCenter(navText(id)), GestureDetectorOpts{OnTap: onTap, Behavior: rendering.HitTestBehaviorOpaque})
}

// navRoutesApp builds an app whose routes are the given builders.
func navRoutesApp(routes map[string]WidgetBuilder, opts ...navTestAppOpts) Widget {
	var o navTestAppOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.routes = routes
	return navTestApp(o)
}

// nowOrNothing returns the value of the channel if it has one at once.
func nowOrNothing(ch <-chan any) (any, bool) {
	select {
	case v := <-ch:
		return v, true
	default:
		return nil, false
	}
}

// recvResult returns the result of the route, and fails if there is none yet.
func recvResult(t *testing.T, ch <-chan any) any {
	t.Helper()
	v, ok := nowOrNothing(ch)
	if !ok {
		t.Fatal("the route has not completed")
	}
	return v
}

// onstage and offstage report where the text is.
func isOnstage(ts *gluttertest.Tester, text string) bool {
	return textsFound(ts, text, false) == 1
}

func isOffstage(ts *gluttertest.Tester, text string) bool {
	return textsFound(ts, text, false) == 0 && textsFound(ts, text, true) == 1
}

// navPumpSecond is `await tester.pump(); await tester.pump(const Duration(seconds: 1));`.
func navPumpSecond(ts *gluttertest.Tester) {
	ts.Pump()
	ts.Pump(time.Second)
}

// navSlideRoute is SlideInOutPageRoute.
func navSlideRoute(body WidgetBuilder, settings RouteSettings) *PageRouteBuilder {
	return NewPageRouteBuilder(
		func(bc BuildContext, a, s animation.Animation[float64]) Widget { return body(bc) },
		PageRouteBuilderOpts{
			PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
			TransitionsBuilder: func(bc BuildContext, a, s animation.Animation[float64], child Widget) Widget {
				return NewSlideTransition(animation.NewTween(ui.Offset{Dx: 1}, ui.Offset{}).Animate(a),
					NewSlideTransition(animation.NewTween(ui.Offset{}, ui.Offset{Dx: -1}).Animate(s), child))
			},
		})
}

// navNoAnimationRoute is NoAnimationPageRoute.
func navNoAnimationRoute(settings RouteSettings, page WidgetBuilder) *PageRouteBuilder {
	return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget { return page(bc) },
		PageRouteBuilderOpts{
			PageRouteOpts:             PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
			TransitionDuration:        foundation.Some(time.Duration(0)),
			ReverseTransitionDuration: foundation.Some(time.Duration(0)),
		})
}

// statefulTestWidget is StatefulTestWidget: it counts its builds.
type statefulTestWidget struct{ StatefulWidget }

func newStatefulTestWidget(key foundation.Key) *statefulTestWidget {
	return &statefulTestWidget{NewStatefulWidgetBase(WidgetOpts{Key: key})}
}

func (w *statefulTestWidget) CreateState() State { return &statefulTestState{} }

type statefulTestState struct {
	StateOf[*statefulTestWidget]
	rebuildCount int
}

func (s *statefulTestState) Build(BuildContext) Widget {
	s.rebuildCount++
	return NewContainer(nil)
}

// transitionDurationObserver is TransitionDurationObserver: it remembers the
// transition duration of the last transition route that was pushed.
type transitionDurationObserver struct {
	NavigatorObserverBase
	duration foundation.Opt[time.Duration]
}

func (o *transitionDurationObserver) DidPush(route, previous Route) {
	if r, ok := route.(TransitionRoute); ok {
		o.duration = foundation.Some(r.TransitionDuration())
	}
}

// notAnnouncedRoute is NotAnnounced: a placeholder for a route that was not
// announced.
type notAnnouncedRoute struct{ RouteBase }

func newNotAnnouncedRoute() *notAnnouncedRoute {
	r := &notAnnouncedRoute{}
	InitRouteBase(&r.RouteBase, r, RouteOpts{})
	return r
}

// routeAnnouncementSpy is RouteAnnouncementSpy.
type routeAnnouncementSpy struct {
	RouteBase
	entries                                            []*OverlayEntry
	onDidChangePrevious, onDidChangeNext, onDidPopNext func(Route)
}

func newRouteAnnouncementSpy(settings RouteSettings, previous, next, popNext func(Route)) *routeAnnouncementSpy {
	r := &routeAnnouncementSpy{onDidChangePrevious: previous, onDidChangeNext: next, onDidPopNext: popNext}
	r.entries = []*OverlayEntry{NewOverlayEntry(func(BuildContext) Widget { return NewPlaceholder(nil) })}
	InitRouteBase(&r.RouteBase, r, RouteOpts{Settings: settings})
	return r
}

func (r *routeAnnouncementSpy) OverlayEntries() []*OverlayEntry { return r.entries }

func (r *routeAnnouncementSpy) DidChangeNext(next Route) {
	r.RouteBase.DidChangeNext(next)
	if r.onDidChangeNext != nil {
		r.onDidChangeNext(next)
	}
}

func (r *routeAnnouncementSpy) DidChangePrevious(previous Route) {
	r.RouteBase.DidChangePrevious(previous)
	if r.onDidChangePrevious != nil {
		r.onDidChangePrevious(previous)
	}
}

func (r *routeAnnouncementSpy) DidPopNext(next Route) {
	r.RouteBase.DidPopNext(next)
	if r.onDidPopNext != nil {
		r.onDidPopNext(next)
	}
}

// heroControllerSpy is HeroControllerSpy.
type heroControllerSpy struct {
	*HeroController
	onPushed func(route, previous Route)
}

func newHeroControllerSpy() *heroControllerSpy {
	s := &heroControllerSpy{HeroController: NewHeroController()}
	InitHeroController(s.HeroController, s)
	return s
}

func (s *heroControllerSpy) DidPush(route, previous Route) {
	if s.onPushed != nil {
		s.onPushed(route, previous)
	}
}

// navTickingWidget is _TickingWidget: it ticks while its TickerMode is enabled.
type navTickingWidget struct {
	StatefulWidget
	onTick func()
}

func newNavTickingWidget(onTick func()) *navTickingWidget {
	return &navTickingWidget{NewStatefulWidgetBase(), onTick}
}

func (w *navTickingWidget) CreateState() State { return &navTickingState{} }

type navTickingState struct {
	StateOf[*navTickingWidget]
	SingleTickerProvider
	ticker *scheduler.Ticker
}

func (s *navTickingState) InitState() {
	s.ticker = s.CreateTicker(func(time.Duration) { s.Widget().onTick() })
	s.ticker.Start()
}

func (s *navTickingState) Build(BuildContext) Widget { return NewContainer(nil) }

func (s *navTickingState) Dispose() { s.ticker.Dispose() }
