package widgets_test

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/routes_test.dart.
// The focus cases are in route_focus_test.go. Not ported here: the Escape
// shortcut (step 10.6), semantics-tester trees of the modal route, state
// restoration, WillPopScope (removed in Flutter 3.x and not ported), the
// display-feature cases of showGeneralDialog, and the delegated-transition cases
// that use CupertinoSheetRoute and CupertinoPageRoute (no Cupertino package).

var (
	routesTestTransparent = ui.NewColor(0x00000000)
	routesTestGreen       = ui.NewColor(0xFF00FF00)
	routesTestBlack       = ui.NewColor(0xFF000000)
	routesTestWhite       = ui.NewColor(0xFFFFFFFF)
)

// routesLog is the `results` and `routes` globals of the Dart test.
type routesLog struct {
	results []string
	routes  map[*lhTestRoute]struct{}
}

func newRoutesLog() *routesLog { return &routesLog{routes: map[*lhTestRoute]struct{}{}} }

// lhTestRoute is TestRoute: a route with a local history that logs its calls.
type lhTestRoute struct {
	LocalHistoryRouteBase
	name        string
	log         *routesLog
	entries     []*OverlayEntry
	installHook func()
}

func newLHTestRoute(l *routesLog, name string) *lhTestRoute {
	r := &lhTestRoute{name: name, log: l}
	InitLocalHistoryRouteBase(&r.LocalHistoryRouteBase, r, RouteOpts{})
	return r
}

func (r *lhTestRoute) logf(format string, args ...any) {
	r.log.results = append(r.log.results, r.name+": "+fmt.Sprintf(format, args...))
}

func routesNullable(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprint(v)
}

func (r *lhTestRoute) OverlayEntries() []*OverlayEntry { return r.entries }

func (r *lhTestRoute) Install() {
	r.logf("install")
	r.entries = append(r.entries, NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) }, OverlayEntryOpts{Opaque: true}))
	r.log.routes[r] = struct{}{}
	if r.installHook != nil {
		r.installHook()
	}
	r.LocalHistoryRouteBase.Install()
}

func (r *lhTestRoute) DidPush() *scheduler.TickerFuture {
	r.logf("didPush")
	return r.LocalHistoryRouteBase.DidPush()
}

func (r *lhTestRoute) DidAdd() {
	r.logf("didAdd")
	r.LocalHistoryRouteBase.DidAdd()
}

func (r *lhTestRoute) DidReplace(oldRoute Route) {
	r.logf("didReplace %s", oldRoute.(*lhTestRoute).name)
	r.LocalHistoryRouteBase.DidReplace(oldRoute)
}

func (r *lhTestRoute) DidPop(result any) bool {
	r.logf("didPop %s", routesNullable(result))
	popped := r.LocalHistoryRouteBase.DidPop(result)
	if popped {
		r.Navigator().FinalizeRoute(r)
	}
	return popped
}

func (r *lhTestRoute) DidPopNext(nextRoute Route) {
	r.logf("didPopNext %s", nextRoute.(*lhTestRoute).name)
	r.LocalHistoryRouteBase.DidPopNext(nextRoute)
}

func (r *lhTestRoute) DidChangeNext(nextRoute Route) {
	name := "null"
	if next, ok := nextRoute.(*lhTestRoute); ok && next != nil {
		name = next.name
	}
	r.logf("didChangeNext %s", name)
	r.LocalHistoryRouteBase.DidChangeNext(nextRoute)
}

func (r *lhTestRoute) Dispose() {
	r.logf("dispose")
	for _, e := range r.entries {
		e.Dispose()
	}
	r.entries = nil
	delete(r.log.routes, r)
	r.LocalHistoryRouteBase.Dispose()
}

// runNavigatorTest runs test, then checks the log, then pumps and checks again.
func runNavigatorTest(t *testing.T, ts *gluttertest.Tester, l *routesLog, test func(), expectations []string, afterAnotherPump ...string) {
	t.Helper()
	test()
	if !slices.Equal(l.results, expectations) {
		t.Fatalf("log = %q, want %q", l.results, expectations)
	}
	l.results = nil
	ts.Pump()
	if !slices.Equal(l.results, afterAnotherPump) {
		t.Fatalf("log after a pump = %q, want %q", l.results, afterAnotherPump)
	}
	l.results = nil
}

func routesNavigatorApp(key *GlobalKey[*NavigatorState], first Route) Widget {
	return NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
		Key:             key,
		OnGenerateRoute: func(RouteSettings) Route { return first },
	}))
}

func TestRouteSettings(t *testing.T) {
	settings := NewRouteSettings(RouteSettingsOpts{Name: "A"})
	if got, want := settings.String(), "RouteSettings(\"A\", null)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if settings.Arguments() != nil {
		t.Errorf("Arguments() = %v, want nil", settings.Arguments())
	}
	arguments := &struct{}{}
	settings2 := NewRouteSettings(RouteSettingsOpts{Name: "A", Arguments: arguments})
	if settings2.Arguments() != any(arguments) {
		t.Error("Arguments() is not the object that was given")
	}
}

func TestRouteManagementPushReplacePop(t *testing.T) {
	ts := gluttertest.New(t)
	l := newRoutesLog()
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(routesNavigatorApp(key, newLHTestRoute(l, "initial")))
	host := navState(t, key)
	runNavigatorTest(t, ts, l, func() {}, []string{"initial: install", "initial: didAdd", "initial: didChangeNext null"})
	var second *lhTestRoute
	runNavigatorTest(t, ts, l, func() { second = newLHTestRoute(l, "second"); host.Push(second) }, []string{
		"second: install", "second: didPush", "second: didChangeNext null", "initial: didChangeNext second",
	})
	runNavigatorTest(t, ts, l, func() { host.Push(newLHTestRoute(l, "third")) }, []string{
		"third: install", "third: didPush", "third: didChangeNext null", "second: didChangeNext third",
	})
	runNavigatorTest(t, ts, l, func() { host.Replace(second, newLHTestRoute(l, "two")) }, []string{
		"two: install", "two: didReplace second", "two: didChangeNext third", "initial: didChangeNext two", "second: dispose",
	})
	runNavigatorTest(t, ts, l, func() { host.Pop("hello") },
		[]string{"third: didPop hello", "two: didPopNext third"}, "third: dispose")
	runNavigatorTest(t, ts, l, func() { host.Pop("good bye") },
		[]string{"two: didPop good bye", "initial: didPopNext two"}, "two: dispose")
	ts.PumpWidget(NewContainer(nil))
	if !slices.Equal(l.results, []string{"initial: dispose"}) {
		t.Errorf("log = %q", l.results)
	}
	if len(l.routes) != 0 {
		t.Errorf("%d routes were not disposed", len(l.routes))
	}
}

func TestRouteManagementPushRemovePop(t *testing.T) {
	ts := gluttertest.New(t)
	l := newRoutesLog()
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(routesNavigatorApp(key, newLHTestRoute(l, "first")))
	host := navState(t, key)
	runNavigatorTest(t, ts, l, func() {}, []string{"first: install", "first: didAdd", "first: didChangeNext null"})
	var second, four *lhTestRoute
	runNavigatorTest(t, ts, l, func() { second = newLHTestRoute(l, "second"); host.Push(second) }, []string{
		"second: install", "second: didPush", "second: didChangeNext null", "first: didChangeNext second",
	})
	runNavigatorTest(t, ts, l, func() { host.Push(newLHTestRoute(l, "third")) }, []string{
		"third: install", "third: didPush", "third: didChangeNext null", "second: didChangeNext third",
	})
	runNavigatorTest(t, ts, l, func() { host.RemoveRouteBelow(second) }, []string{"first: dispose"})
	runNavigatorTest(t, ts, l, func() { host.Pop("good bye") },
		[]string{"third: didPop good bye", "second: didPopNext third"}, "third: dispose")
	runNavigatorTest(t, ts, l, func() { host.Push(newLHTestRoute(l, "three")) }, []string{
		"three: install", "three: didPush", "three: didChangeNext null", "second: didChangeNext three",
	})
	runNavigatorTest(t, ts, l, func() { four = newLHTestRoute(l, "four"); host.Push(four) }, []string{
		"four: install", "four: didPush", "four: didChangeNext null", "three: didChangeNext four",
	})
	runNavigatorTest(t, ts, l, func() { host.RemoveRouteBelow(four) }, []string{"second: didChangeNext four", "three: dispose"})
	runNavigatorTest(t, ts, l, func() { host.Pop("the end") },
		[]string{"four: didPop the end", "second: didPopNext four"}, "four: dispose")
	ts.PumpWidget(NewContainer(nil))
	if !slices.Equal(l.results, []string{"second: dispose"}) {
		t.Errorf("log = %q", l.results)
	}
	if len(l.routes) != 0 {
		t.Errorf("%d routes were not disposed", len(l.routes))
	}
}

func TestRouteManagementPushReplacePopUntil(t *testing.T) {
	ts := gluttertest.New(t)
	l := newRoutesLog()
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(routesNavigatorApp(key, newLHTestRoute(l, "A")))
	host := navState(t, key)
	runNavigatorTest(t, ts, l, func() {}, []string{"A: install", "A: didAdd", "A: didChangeNext null"})
	runNavigatorTest(t, ts, l, func() { host.Push(newLHTestRoute(l, "B")) },
		[]string{"B: install", "B: didPush", "B: didChangeNext null", "A: didChangeNext B"})
	routeC := newLHTestRoute(l, "C")
	runNavigatorTest(t, ts, l, func() { host.Push(routeC) },
		[]string{"C: install", "C: didPush", "C: didChangeNext null", "B: didChangeNext C"})
	if !routeC.IsActive() {
		t.Error("C should be active")
	}
	routeB := newLHTestRoute(l, "b")
	runNavigatorTest(t, ts, l, func() { host.ReplaceRouteBelow(routeC, routeB) }, []string{
		"b: install", "b: didReplace B", "b: didChangeNext C", "A: didChangeNext b", "B: dispose",
	})
	runNavigatorTest(t, ts, l, func() { host.PopUntil(func(r Route) bool { return r == Route(routeB) }) },
		[]string{"C: didPop null", "b: didPopNext C"}, "C: dispose")
	ts.PumpWidget(NewContainer(nil))
	if !slices.Equal(l.results, []string{"b: dispose", "A: dispose"}) {
		t.Errorf("log = %q", l.results)
	}
	if len(l.routes) != 0 {
		t.Errorf("%d routes were not disposed", len(l.routes))
	}
}

func TestRouteLocalHistoryPopUntil(t *testing.T) {
	ts := gluttertest.New(t)
	l := newRoutesLog()
	routeA := newLHTestRoute(l, "A")
	routeA.AddLocalHistoryEntry(NewLocalHistoryEntry(LocalHistoryEntryOpts{OnRemove: func() { routeA.logf("onRemove 0") }}))
	routeA.AddLocalHistoryEntry(NewLocalHistoryEntry(LocalHistoryEntryOpts{OnRemove: func() { routeA.logf("onRemove 1") }}))
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(routesNavigatorApp(key, routeA))
	host := navState(t, key)
	runNavigatorTest(t, ts, l, func() {
		host.PopUntil(func(r Route) bool { return !r.WillHandlePopInternally() })
	}, []string{
		"A: install", "A: didAdd", "A: didChangeNext null",
		"A: didPop null", "A: onRemove 1",
		"A: didPop null", "A: onRemove 0",
	})
	runNavigatorTest(t, ts, l, func() {
		host.PopUntil(func(r Route) bool { return !r.WillHandlePopInternally() })
	}, nil)
	ts.PumpWidget(NewContainer(nil))
	if len(l.routes) != 0 {
		t.Errorf("%d routes were not disposed", len(l.routes))
	}
}

// mockRouteAware is MockRouteAware: it counts the calls.
type mockRouteAware struct {
	RouteAwareBase
	didPushCount, didPopCount, didPushNextCount, didPopNextCount int
}

func (m *mockRouteAware) DidPush()     { m.didPushCount++ }
func (m *mockRouteAware) DidPop()      { m.didPopCount++ }
func (m *mockRouteAware) DidPushNext() { m.didPushNextCount++ }
func (m *mockRouteAware) DidPopNext()  { m.didPopNextCount++ }

func newMockPageRoute() PageRoute {
	return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewContainer(nil)
	})
}

func TestPageRouteObserver(t *testing.T) {
	t.Run("calls correct listeners", func(t *testing.T) {
		observer := NewRouteObserver[PageRoute]()
		aware1 := &mockRouteAware{}
		route1 := newMockPageRoute()
		observer.Subscribe(aware1, route1)
		if aware1.didPushCount != 1 {
			t.Errorf("didPushCount = %d, want 1", aware1.didPushCount)
		}
		aware2 := &mockRouteAware{}
		route2 := newMockPageRoute()
		observer.DidPush(route2, route1)
		if aware1.didPushNextCount != 1 {
			t.Errorf("didPushNextCount = %d, want 1", aware1.didPushNextCount)
		}
		observer.Subscribe(aware2, route2)
		if aware2.didPushCount != 1 {
			t.Errorf("didPushCount = %d, want 1", aware2.didPushCount)
		}
		observer.DidPop(route2, route1)
		if aware2.didPopCount != 1 || aware1.didPopNextCount != 1 {
			t.Errorf("didPopCount = %d, didPopNextCount = %d, want 1, 1", aware2.didPopCount, aware1.didPopNextCount)
		}
	})
	t.Run("does not call listeners for non-PageRoute", func(t *testing.T) {
		observer := NewRouteObserver[PageRoute]()
		aware := &mockRouteAware{}
		pageRoute := newMockPageRoute()
		route := newNotAnnouncedRoute()
		observer.Subscribe(aware, pageRoute)
		observer.DidPush(route, pageRoute)
		observer.DidPop(route, pageRoute)
		if aware.didPushCount != 1 || aware.didPopCount != 0 || aware.didPopNextCount != 0 {
			t.Errorf("counts = %+v", *aware)
		}
	})
	t.Run("does not call listeners when already subscribed", func(t *testing.T) {
		observer := NewRouteObserver[PageRoute]()
		aware := &mockRouteAware{}
		pageRoute := newMockPageRoute()
		observer.Subscribe(aware, pageRoute)
		observer.Subscribe(aware, pageRoute)
		if aware.didPushCount != 1 {
			t.Errorf("didPushCount = %d, want 1", aware.didPushCount)
		}
	})
	t.Run("does not call listeners when unsubscribed", func(t *testing.T) {
		observer := NewRouteObserver[PageRoute]()
		aware := &mockRouteAware{}
		pageRoute, nextPageRoute := newMockPageRoute(), newMockPageRoute()
		observer.Subscribe(aware, pageRoute)
		observer.Subscribe(aware, nextPageRoute)
		if aware.didPushCount != 2 {
			t.Fatalf("didPushCount = %d, want 2", aware.didPushCount)
		}
		observer.Unsubscribe(aware)
		observer.DidPush(nextPageRoute, pageRoute)
		observer.DidPop(nextPageRoute, pageRoute)
		if aware.didPushCount != 2 || aware.didPopCount != 0 {
			t.Errorf("counts = %+v", *aware)
		}
	})
	t.Run("releases reference to route when unsubscribed", func(t *testing.T) {
		skipUnlessDebug(t)
		observer := NewRouteObserver[PageRoute]()
		aware, aware2 := &mockRouteAware{}, &mockRouteAware{}
		pageRoute, nextPageRoute := newMockPageRoute(), newMockPageRoute()
		observer.Subscribe(aware, pageRoute)
		observer.Subscribe(aware, nextPageRoute)
		observer.Subscribe(aware2, pageRoute)
		observer.Subscribe(aware2, nextPageRoute)
		if aware.didPushCount != 2 || aware2.didPushCount != 2 {
			t.Fatalf("didPushCount = %d, %d, want 2, 2", aware.didPushCount, aware2.didPushCount)
		}
		if !observer.DebugObservingRoute(pageRoute) || !observer.DebugObservingRoute(nextPageRoute) {
			t.Fatal("the routes should be observed")
		}
		observer.Unsubscribe(aware)
		if !observer.DebugObservingRoute(pageRoute) || !observer.DebugObservingRoute(nextPageRoute) {
			t.Fatal("the routes should still be observed")
		}
		observer.Unsubscribe(aware2)
		if observer.DebugObservingRoute(pageRoute) || observer.DebugObservingRoute(nextPageRoute) {
			t.Error("the routes should not be observed any more")
		}
	})
}

// routesButton is TestButton: a tappable text.
func routesButton(label string, onTap func()) Widget {
	return NewGestureDetector(NewCenter(navText(label)), GestureDetectorOpts{OnTap: onTap, Behavior: rendering.HitTestBehaviorOpaque})
}

func routesPageBuilder(child func(BuildContext) Widget) RoutePageBuilder {
	return func(bc BuildContext, _, _ animation.Animation[float64]) Widget { return child(bc) }
}

// routesGenerate is onGenerateRoute of TestWidgetsApp that always creates a
// PageRouteBuilder around the builder.
func routesGenerate(builder func(settings RouteSettings) *PageRouteBuilder) navTestAppOpts {
	return navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route { return builder(settings) }}
}

func TestPageRouteBuilderReverseTransitionDuration(t *testing.T) {
	tests := []struct {
		name    string
		reverse foundation.Opt[time.Duration]
		half    time.Duration
	}{
		{"defaults to 300ms", foundation.Opt[time.Duration]{}, 150 * time.Millisecond},
		{"can be customized", foundation.Some(150 * time.Millisecond), 75 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(navTestApp(routesGenerate(func(settings RouteSettings) *PageRouteBuilder {
				return NewPageRouteBuilder(routesPageBuilder(func(bc BuildContext) Widget {
					return routesButton("Open page", func() {
						NavigatorPush(bc, NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("Page Two") }),
							PageRouteBuilderOpts{
								PageRouteOpts:             PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
								ReverseTransitionDuration: tc.reverse,
							}))
					})
				}), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
			})))
			ts.Tap(gluttertest.ByText("Open page"))
			ts.PumpAndSettle()
			if textsFound(ts, "Open page", false) != 0 || textsFound(ts, "Page Two", false) != 1 {
				t.Fatal("the new route should be showing alone")
			}
			gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
			ts.Pump()
			steps := []struct {
				d    time.Duration
				what string
			}{{0, "after the pop"}, {tc.half, "halfway"}, {tc.half, "at the very end"}}
			for _, s := range steps {
				if s.d > 0 {
					ts.Pump(s.d)
				}
				if textsFound(ts, "Page Two", false) != 1 {
					t.Fatalf("Page Two should be present %s of the reverse transition", s.what)
				}
			}
			ts.Pump(time.Millisecond)
			if textsFound(ts, "Page Two", false) != 0 || textsFound(ts, "Open page", false) != 1 {
				t.Error("Page Two should have transitioned out")
			}
		})
	}
}

// curvedPageRoute is TestPageRouteBuilder and _SimulationRoute: a page route
// whose animation or simulation can be replaced.
type curvedPageRoute struct {
	PageRouteBase
	pageBuilder        RoutePageBuilder
	transitionsBuilder RouteTransitionsBuilder
	transitionDuration time.Duration
	reverseDuration    time.Duration
	curve              animation.Curve
	simulation         func(current float64, forward bool) physics.Simulation
	nilController      bool
	nilAnimation       bool
}

func newCurvedPageRoute(pageBuilder RoutePageBuilder, configure func(*curvedPageRoute)) *curvedPageRoute {
	r := &curvedPageRoute{
		pageBuilder:        pageBuilder,
		transitionDuration: 300 * time.Millisecond,
		reverseDuration:    300 * time.Millisecond,
	}
	if configure != nil {
		configure(r)
	}
	InitPageRouteBase(&r.PageRouteBase, r, PageRouteOpts{})
	return r
}

func (r *curvedPageRoute) TransitionDuration() time.Duration        { return r.transitionDuration }
func (r *curvedPageRoute) ReverseTransitionDuration() time.Duration { return r.reverseDuration }
func (r *curvedPageRoute) BarrierColor() foundation.Opt[ui.Color]   { return foundation.Opt[ui.Color]{} }
func (r *curvedPageRoute) BarrierLabel() string                     { return "" }
func (r *curvedPageRoute) MaintainState() bool                      { return true }

func (r *curvedPageRoute) BuildPage(bc BuildContext, a, s animation.Animation[float64]) Widget {
	return r.pageBuilder(bc, a, s)
}

func (r *curvedPageRoute) BuildTransitions(bc BuildContext, a, s animation.Animation[float64], child Widget) Widget {
	if r.transitionsBuilder == nil {
		return child
	}
	return r.transitionsBuilder(bc, a, s, child)
}

func (r *curvedPageRoute) CreateAnimationController() *animation.AnimationController {
	if r.nilController {
		return nil
	}
	return r.PageRouteBase.CreateAnimationController()
}

func (r *curvedPageRoute) CreateAnimation() animation.Animation[float64] {
	if r.nilAnimation {
		return nil
	}
	parent := r.PageRouteBase.CreateAnimation()
	if r.curve == nil {
		return parent
	}
	return animation.NewCurvedAnimation(parent, r.curve)
}

func (r *curvedPageRoute) CreateSimulation(forward bool) physics.Simulation {
	if r.simulation == nil {
		return r.PageRouteBase.CreateSimulation(forward)
	}
	return r.simulation(r.Controller().Value(), forward)
}

func routesProxy(t *testing.T, a animation.Animation[float64]) *animation.ProxyAnimation {
	t.Helper()
	p, ok := a.(*animation.ProxyAnimation)
	if !ok {
		t.Fatalf("%T is not a ProxyAnimation", a)
	}
	return p
}

// secondaryAnimationRoutes pushes page one and then page two (made by makeTwo).
type pushedPage struct {
	secondaryProxy, animationProxy *animation.ProxyAnimation
}

func routesPushRecording(t *testing.T, ts *gluttertest.Tester, key *GlobalKey[*NavigatorState], text string) (*pushedPage, *PageRouteBuilder) {
	t.Helper()
	p := &pushedPage{}
	route := NewPageRouteBuilder(func(_ BuildContext, a, s animation.Animation[float64]) Widget {
		p.secondaryProxy = routesProxy(t, s)
		p.animationProxy = routesProxy(t, a)
		return navText(text)
	})
	navState(t, key).Push(route)
	ts.Pump()
	ts.PumpAndSettle()
	return p, route
}

func TestTransitionRouteSecondaryAnimationDismissedWhenNextRoutePops(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	one, _ := routesPushRecording(t, ts, key, "Page One")
	secondaryOne := routesProxy(t, one.secondaryProxy.Parent())
	if one.animationProxy.Value() != 1 || secondaryOne.Parent() != animation.AlwaysDismissed {
		t.Fatalf("page one: value %v, parent %v", one.animationProxy.Value(), secondaryOne.Parent())
	}
	two, _ := routesPushRecording(t, ts, key, "Page Two")
	secondaryTwo := routesProxy(t, two.secondaryProxy.Parent())
	if two.animationProxy.Value() != 1 || secondaryTwo.Parent() != animation.AlwaysDismissed {
		t.Fatalf("page two: value %v, parent %v", two.animationProxy.Value(), secondaryTwo.Parent())
	}
	if secondaryOne.Parent() != two.animationProxy.Parent() {
		t.Fatal("the secondary animation of page one should be the animation of page two")
	}
	navState(t, key).Pop()
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if secondaryOne.Parent() != two.animationProxy.Parent() {
		t.Error("the animations should still be chained while popping")
	}
	ts.PumpAndSettle()
	if two.animationProxy.Value() != 0 || secondaryOne.Parent() != animation.AlwaysDismissed {
		t.Errorf("after the pop: value %v, parent %v", two.animationProxy.Value(), secondaryOne.Parent())
	}
}

func TestTransitionRouteSecondaryAnimationDismissedWhenNextRouteIsRemoved(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	one, _ := routesPushRecording(t, ts, key, "Page One")
	secondaryOne := routesProxy(t, one.secondaryProxy.Parent())
	two, second := routesPushRecording(t, ts, key, "Page Two")
	if secondaryOne.Parent() != two.animationProxy.Parent() {
		t.Fatal("the secondary animation of page one should be the animation of page two")
	}
	navState(t, key).RemoveRoute(second)
	ts.Pump()
	if secondaryOne.Parent() != animation.AlwaysDismissed {
		t.Errorf("the secondary animation of page one is %v, want kAlwaysDismissedAnimation", secondaryOne.Parent())
	}
}

func TestTransitionRouteSecondaryAnimationAfterTrainHopping(t *testing.T) {
	setup := func(t *testing.T) (*gluttertest.Tester, *GlobalKey[*NavigatorState], *pushedPage, *animation.ProxyAnimation, *pushedPage) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
		one, _ := routesPushRecording(t, ts, key, "Page One")
		secondaryOne := routesProxy(t, one.secondaryProxy.Parent())
		two := &pushedPage{}
		navState(t, key).Push(NewPageRouteBuilder(func(_ BuildContext, a, s animation.Animation[float64]) Widget {
			two.animationProxy = routesProxy(t, a)
			return navText("Page Two")
		}))
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if secondaryOne.Parent() != two.animationProxy.Parent() {
			t.Fatal("the secondary animation of page one should be the animation of page two")
		}
		return ts, key, one, secondaryOne, two
	}
	newThree := func(t *testing.T, three *pushedPage) Route {
		return newCurvedPageRoute(func(_ BuildContext, a, s animation.Animation[float64]) Widget {
			if three != nil {
				three.animationProxy = routesProxy(t, a)
			}
			return navText("Page Three")
		}, func(r *curvedPageRoute) { r.curve = animation.Curves.EaseOutExpo })
	}

	t.Run("dismissed after train hopping finishes and pop", func(t *testing.T) {
		ts, key, _, secondaryOne, two := setup(t)
		three := &pushedPage{}
		navState(t, key).PushReplacement(newThree(t, three))
		ts.Pump()
		ts.Pump(time.Millisecond)
		hopper, ok := secondaryOne.Parent().(*animation.TrainHoppingAnimation)
		if !ok {
			t.Fatalf("the secondary animation is %T, want a TrainHoppingAnimation", secondaryOne.Parent())
		}
		if hopper.CurrentTrain() != two.animationProxy.Parent() {
			t.Error("the train hopper should be on the animation of page two")
		}
		ts.Pump(100 * time.Millisecond)
		if _, ok := secondaryOne.Parent().(*animation.TrainHoppingAnimation); ok {
			t.Error("the train hopper should have been replaced")
		}
		if secondaryOne.Parent() != three.animationProxy.Parent() {
			t.Error("the secondary animation of page one should be the animation of page three")
		}
		if hopper.CurrentTrain() != nil {
			t.Error("the train hopper should have been disposed")
		}
		ts.PumpAndSettle()
		if secondaryOne.Parent() != three.animationProxy.Parent() {
			t.Error("the secondary animation of page one should still be the animation of page three")
		}
		navState(t, key).Pop()
		ts.Pump()
		ts.PumpAndSettle()
		if secondaryOne.Parent() != animation.AlwaysDismissed {
			t.Errorf("the secondary animation of page one is %v, want kAlwaysDismissedAnimation", secondaryOne.Parent())
		}
	})

	t.Run("dismissed when train hopping is interrupted", func(t *testing.T) {
		ts, key, _, secondaryOne, two := setup(t)
		navState(t, key).PushReplacement(newThree(t, nil))
		ts.Pump()
		ts.Pump(10 * time.Millisecond)
		hopper, ok := secondaryOne.Parent().(*animation.TrainHoppingAnimation)
		if !ok {
			t.Fatalf("the secondary animation is %T, want a TrainHoppingAnimation", secondaryOne.Parent())
		}
		if hopper.CurrentTrain() != two.animationProxy.Parent() {
			t.Error("the train hopper should be on the animation of page two")
		}
		navState(t, key).Pop()
		ts.Pump()
		hopper2, ok := secondaryOne.Parent().(*animation.TrainHoppingAnimation)
		if !ok {
			t.Fatalf("the secondary animation is %T, want a TrainHoppingAnimation", secondaryOne.Parent())
		}
		if hopper2.CurrentTrain() != two.animationProxy.Parent() {
			t.Error("the new train hopper should be on the animation of page two")
		}
		if hopper.CurrentTrain() != nil {
			t.Error("the first train hopper should have been disposed")
		}
		ts.PumpAndSettle()
		if secondaryOne.Parent() != animation.AlwaysDismissed {
			t.Errorf("the secondary animation of page one is %v, want kAlwaysDismissedAnimation", secondaryOne.Parent())
		}
		if hopper2.CurrentTrain() != nil {
			t.Error("the second train hopper should have been disposed")
		}
	})
}

func TestTransitionRouteSecondaryAnimationTriggeredWhenPopInitialRoute(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var secondaryOfOne, primaryOfTwo animation.Animation[float64]
	ts.PumpWidget(navTestApp(navTestAppOpts{
		navigatorKey: key,
		initialRoute: "/a",
		onGenerateRoute: func(settings RouteSettings) Route {
			return NewPageRouteBuilder(func(_ BuildContext, a, s animation.Animation[float64]) Widget {
				if settings.Name() == "/" {
					secondaryOfOne = s
				} else {
					primaryOfTwo = a
				}
				return navText("Page")
			}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		},
	}))
	if secondaryOfOne.Value() != 1 || secondaryOfOne.Value() != primaryOfTwo.Value() {
		t.Fatalf("the animations should be chained at 1: %v, %v", secondaryOfOne.Value(), primaryOfTwo.Value())
	}
	navState(t, key).Pop()
	ts.Pump()
	ts.Pump(30 * time.Millisecond)
	if math.Abs(secondaryOfOne.Value()-0.9) > 1e-9 || secondaryOfOne.Value() != primaryOfTwo.Value() {
		t.Errorf("after 30ms: %v, %v, want 0.9, 0.9", secondaryOfOne.Value(), primaryOfTwo.Value())
	}
	ts.PumpAndSettle()
	if secondaryOfOne.Value() != 0 || secondaryOfOne.Value() != primaryOfTwo.Value() {
		t.Errorf("at the end: %v, %v, want 0, 0", secondaryOfOne.Value(), primaryOfTwo.Value())
	}
}

// dialogObserver is DialogObserver.
type dialogObserver struct {
	NavigatorObserverBase
	dialogRoutes []ModalRoute
	dialogCount  int
}

func (o *dialogObserver) DidPush(route, previousRoute Route) {
	if dialog, ok := route.(*RawDialogRoute); ok {
		o.dialogRoutes = append(o.dialogRoutes, dialog)
		o.dialogCount++
	}
}

func (o *dialogObserver) DidPop(route, previousRoute Route) {
	if _, ok := route.(*RawDialogRoute); ok {
		o.dialogRoutes = o.dialogRoutes[:len(o.dialogRoutes)-1]
		o.dialogCount--
	}
}

func routesSizedBox() RoutePageBuilder {
	return func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewSizedBoxShrink(nil)
	}
}

func TestShowGeneralDialogBarriers(t *testing.T) {
	setup := func(ts *gluttertest.Tester, show func(bc BuildContext)) {
		ts.PumpWidget(navTestApp(navTestAppOpts{home: NewBuilder(func(bc BuildContext) Widget {
			return routesButton("Show Dialog", func() { show(bc) })
		})}))
	}
	barriers := func(ts *gluttertest.Tester) int { return ts.Count(gluttertest.ByType[*ModalBarrier]()) }

	t.Run("handles transparent barrier color", func(t *testing.T) {
		ts := gluttertest.New(t)
		setup(ts, func(bc BuildContext) {
			ShowGeneralDialog(bc, routesSizedBox(), ShowGeneralDialogOpts{
				BarrierDismissible: true,
				BarrierLabel:       "barrier_label",
				BarrierColor:       foundation.Some(routesTestTransparent),
				TransitionDuration: foundation.Some(time.Duration(0)),
			})
		})
		ts.Tap(gluttertest.ByText("Show Dialog"))
		ts.Pump()
		if got := barriers(ts); got != 2 {
			t.Fatalf("%d barriers, want 2", got)
		}
		ts.TapAt(ui.Offset{})
		ts.Pump()
		if got := barriers(ts); got != 1 {
			t.Errorf("%d barriers after the tap, want 1", got)
		}
	})

	t.Run("adds a non-dismissible barrier when barrierDismissible is false", func(t *testing.T) {
		ts := gluttertest.New(t)
		setup(ts, func(bc BuildContext) {
			ShowGeneralDialog(bc, routesSizedBox(), ShowGeneralDialogOpts{TransitionDuration: foundation.Some(time.Duration(0))})
		})
		ts.Tap(gluttertest.ByText("Show Dialog"))
		ts.Pump()
		found := ts.Elements(gluttertest.ByType[*ModalBarrier]())
		if len(found) != 2 {
			t.Fatalf("%d barriers, want 2", len(found))
		}
		if found[len(found)-1].Widget().(*ModalBarrier).Dismissible() {
			t.Error("the barrier should not be dismissible")
		}
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.PumpAndSettle()
		if got := barriers(ts); got != 1 {
			t.Errorf("%d barriers after the pop, want 1", got)
		}
	})

	t.Run("uses no barrier label by default", func(t *testing.T) {
		ts := gluttertest.New(t)
		setup(ts, func(bc BuildContext) {
			ShowGeneralDialog(bc, routesSizedBox(), ShowGeneralDialogOpts{TransitionDuration: foundation.Some(time.Duration(0))})
		})
		ts.Tap(gluttertest.ByText("Show Dialog"))
		ts.Pump()
		found := ts.Elements(gluttertest.ByType[*ModalBarrier]())
		if len(found) != 2 {
			t.Fatalf("%d barriers, want 2", len(found))
		}
		if label := found[len(found)-1].Widget().(*ModalBarrier).SemanticsLabel(); label != "" {
			t.Errorf("SemanticsLabel = %q, want none", label)
		}
	})
}

func TestShowGeneralDialogNavigators(t *testing.T) {
	// A nested Navigator inside the app Navigator; the dialog goes to the root
	// one unless useRootNavigator is false.
	for _, tc := range []struct {
		name                string
		useRoot             foundation.Opt[bool]
		wantRoot, wantInner int
	}{
		{"uses the root navigator by default", foundation.Opt[bool]{}, 1, 0},
		{"uses the nested navigator if useRootNavigator is false", foundation.Some(false), 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			root, nested := &dialogObserver{}, &dialogObserver{}
			ts.PumpWidget(navTestApp(navTestAppOpts{
				observers: []NavigatorObserver{root},
				home: NewNavigator(NavigatorOpts{
					Observers: []NavigatorObserver{nested},
					OnGenerateRoute: func(settings RouteSettings) Route {
						return NewPageRouteBuilder(routesPageBuilder(func(bc BuildContext) Widget {
							return routesButton("Show Dialog", func() {
								ShowGeneralDialog(bc, routesSizedBox(), ShowGeneralDialogOpts{
									UseRootNavigator:   tc.useRoot,
									TransitionDuration: foundation.Some(time.Duration(0)),
								})
							})
						}), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
					},
				}),
			}))
			ts.Tap(gluttertest.ByText("Show Dialog"))
			if root.dialogCount != tc.wantRoot || nested.dialogCount != tc.wantInner {
				t.Errorf("dialogs: root %d, nested %d, want %d, %d", root.dialogCount, nested.dialogCount, tc.wantRoot, tc.wantInner)
			}
		})
	}
}

func TestShowGeneralDialogDefaultArgumentValues(t *testing.T) {
	ts := gluttertest.New(t)
	root := &dialogObserver{}
	ts.PumpWidget(navTestApp(navTestAppOpts{
		observers: []NavigatorObserver{root},
		home: NewNavigator(NavigatorOpts{
			OnGenerateRoute: func(settings RouteSettings) Route {
				return NewPageRouteBuilder(routesPageBuilder(func(bc BuildContext) Widget {
					return routesButton("Show Dialog", func() { ShowGeneralDialog(bc, routesSizedBox()) })
				}), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
			},
		}),
	}))
	ts.Tap(gluttertest.ByText("Show Dialog"))
	if len(root.dialogRoutes) != 1 {
		t.Fatalf("%d dialog routes, want 1", len(root.dialogRoutes))
	}
	route := root.dialogRoutes[0]
	if route.BarrierDismissible() {
		t.Error("the barrier should not be dismissible by default")
	}
	if !route.BarrierColor().IsSome() {
		t.Error("there should be a default barrier color")
	}
	if route.TransitionDuration() != 200*time.Millisecond {
		t.Errorf("TransitionDuration = %v, want 200ms", route.TransitionDuration())
	}
}

func TestShowGeneralDialogBarrierLabelRequiredWhenDismissible(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(navTestApp(navTestAppOpts{home: navText("home")}))
	bc := ts.Element(gluttertest.ByText("home"))
	requireTag(t, "ShowGeneralDialogBarrierLabelMissing", func() {
		ShowGeneralDialog(bc, routesSizedBox(), ShowGeneralDialogOpts{BarrierDismissible: true})
	})
}

func TestTransitionRouteReverseDuration(t *testing.T) {
	green := func(key foundation.Key) func(BuildContext) Widget {
		return func(BuildContext) Widget {
			return NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(routesTestGreen)})
		}
	}
	openPage := func(key foundation.Key, configure PageRouteBuilderOpts) navTestAppOpts {
		return routesGenerate(func(settings RouteSettings) *PageRouteBuilder {
			return NewPageRouteBuilder(routesPageBuilder(func(bc BuildContext) Widget {
				return routesButton("Open page", func() {
					NavigatorPush(bc, NewPageRouteBuilder(routesPageBuilder(green(key)), configure))
				})
			}), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		})
	}
	present := func(ts *gluttertest.Tester, key foundation.Key) bool {
		return ts.Count(gluttertest.ByKey(key)) == 1
	}

	t.Run("reverseTransitionDuration defaults to transitionDuration", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[State]()
		observer := &transitionDurationObserver{}
		app := openPage(key, PageRouteBuilderOpts{})
		app.observers = []NavigatorObserver{observer}
		ts.PumpWidget(navTestApp(app))
		ts.Tap(gluttertest.ByText("Open page"))
		ts.PumpAndSettle()
		if textsFound(ts, "Open page", false) != 0 || !present(ts, key) {
			t.Fatal("the new route should be showing alone")
		}
		duration := observer.duration.OrElse(0)
		if duration == 0 {
			t.Fatal("the transition duration was not observed")
		}
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.Pump()
		for _, step := range []time.Duration{0, duration / 2, duration / 2} {
			if step > 0 {
				ts.Pump(step)
			}
			if !present(ts, key) {
				t.Fatal("the container should be present during the transition")
			}
		}
		ts.Pump(time.Millisecond)
		if present(ts, key) {
			t.Error("the container should have transitioned out")
		}
	})

	t.Run("reverseTransitionDuration can be customized", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[State]()
		ts.PumpWidget(navTestApp(openPage(key, PageRouteBuilderOpts{ReverseTransitionDuration: foundation.Some(150 * time.Millisecond)})))
		ts.Tap(gluttertest.ByText("Open page"))
		ts.PumpAndSettle()
		if textsFound(ts, "Open page", false) != 0 || !present(ts, key) {
			t.Fatal("the new route should be showing alone")
		}
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.Pump()
		for _, step := range []time.Duration{0, 75 * time.Millisecond, 75 * time.Millisecond} {
			if step > 0 {
				ts.Pump(step)
			}
			if !present(ts, key) {
				t.Fatal("the container should be present during the transition")
			}
		}
		ts.Pump(time.Millisecond)
		if present(ts, key) {
			t.Error("the container should have transitioned out")
		}
	})

	t.Run("custom reverseTransitionDuration does not result in interrupted animations", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[State]()
		ts.PumpWidget(navTestApp(openPage(key, PageRouteBuilderOpts{
			ReverseTransitionDuration: foundation.Some(150 * time.Millisecond),
			TransitionsBuilder: func(_ BuildContext, a, _ animation.Animation[float64], child Widget) Widget {
				return NewFadeTransition(a, child)
			},
		})))
		ts.Tap(gluttertest.ByText("Open page"))
		ts.Pump()
		ts.Pump(200 * time.Millisecond)
		if !present(ts, key) {
			t.Fatal("the container should be present")
		}
		opacity := func() float64 {
			return ts.Widget(gluttertest.Ancestor(gluttertest.ByKey(key), gluttertest.ByType[*FadeTransition]())).(*FadeTransition).Opacity().Value()
		}
		before := opacity()
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.Pump()
		if after := opacity(); after != before {
			t.Errorf("the opacity jumped from %v to %v", before, after)
		}
		// Reverse duration: 150ms * (200ms / 300ms) = 100ms remaining.
		ts.Pump(100 * time.Millisecond)
		if !present(ts, key) {
			t.Fatal("the container should be present at the very end")
		}
		ts.Pump(time.Millisecond)
		if present(ts, key) {
			t.Error("the container should have transitioned out")
		}
	})
}

// constantVelocitySimulation is _ConstantVelocitySimulation.
type constantVelocitySimulation struct {
	physics.SimulationBase
	forward bool
	speed   float64
	start   float64
}

func newConstantVelocitySimulation(forward bool, speed float64) *constantVelocitySimulation {
	s := &constantVelocitySimulation{SimulationBase: physics.NewSimulationBase(foundation.Opt[physics.Tolerance]{}), forward: forward, speed: speed}
	if !forward {
		s.start = 1
	}
	return s
}

func (s *constantVelocitySimulation) X(time float64) float64 { return s.start + time*s.Dx(time) }
func (s *constantVelocitySimulation) Dx(float64) float64 {
	if s.forward {
		return s.speed
	}
	return -s.speed
}
func (s *constantVelocitySimulation) IsDone(time float64) bool {
	x := s.X(time)
	return x > 1 || x < 0
}

func TestRoutesCanUseSimulation(t *testing.T) {
	longDuration := 24 * time.Hour
	key := NewGlobalKey[State]()
	container := func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(routesTestGreen)})
	}
	app := func(configure func(*curvedPageRoute)) Widget {
		return navTestApp(routesGenerate(func(settings RouteSettings) *PageRouteBuilder {
			return NewPageRouteBuilder(routesPageBuilder(func(bc BuildContext) Widget {
				return routesButton("Open page", func() {
					NavigatorPush(bc, newCurvedPageRoute(container, func(r *curvedPageRoute) {
						r.transitionDuration, r.reverseDuration = longDuration, longDuration
						configure(r)
					}))
				})
			}), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		}))
	}
	present := func(ts *gluttertest.Tester) bool { return ts.Count(gluttertest.ByKey(key)) == 1 }

	t.Run("ignoring durations", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(app(func(r *curvedPageRoute) {
			// This simulation takes 1.0 second to transit.
			r.simulation = func(float64, bool) physics.Simulation { return physics.NewGravitySimulation(0, 0, 1, 1) }
		}))
		ts.Tap(gluttertest.ByText("Open page"))
		ts.PumpAndSettle()
		if textsFound(ts, "Open page", false) != 0 || !present(ts) {
			t.Fatal("the new route should be showing alone")
		}
		ts.Pump(500 * time.Millisecond)
		ts.Pump(500 * time.Millisecond)
		if !present(ts) {
			t.Fatal("the container should be present")
		}
		ts.PumpAndSettle()
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.Pump()
		for _, step := range []time.Duration{0, 500 * time.Millisecond, 490 * time.Millisecond} {
			if step > 0 {
				ts.Pump(step)
			}
			if !present(ts) {
				t.Fatal("the container should be present during the transition")
			}
		}
		ts.Pump(200 * time.Millisecond)
		if present(ts) {
			t.Error("the container should have transitioned out")
		}
	})

	t.Run("using the value", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(app(func(r *curvedPageRoute) {
			r.simulation = func(_ float64, forward bool) physics.Simulation { return newConstantVelocitySimulation(forward, 1) }
			r.transitionsBuilder = func(_ BuildContext, a, _ animation.Animation[float64], child Widget) Widget {
				return NewFractionalTranslation(animation.NewTween(ui.Offset{Dy: 1}, ui.Offset{}).Transform(a.Value()), child)
			}
		}))
		ts.Tap(gluttertest.ByText("Open page"))
		// The first pump starts the animation, the second makes the widget appear.
		ts.Pump()
		ts.Pump()
		check := func(want ui.Offset, what string) {
			t.Helper()
			if !present(ts) {
				t.Fatalf("%s: the container should be present", what)
			}
			if got := ts.TopLeft(gluttertest.ByKey(key)); got != want {
				t.Errorf("%s: top left = %v, want %v", what, got, want)
			}
		}
		check(ui.Offset{Dy: 600}, "start")
		ts.Pump(500 * time.Millisecond)
		check(ui.Offset{Dy: 300}, "halfway")
		ts.Pump(500 * time.Millisecond)
		check(ui.Offset{}, "end")
		ts.PumpAndSettle()
		gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).Pop()
		ts.Pump()
		ts.Pump()
		check(ui.Offset{}, "popped")
		ts.Pump(500 * time.Millisecond)
		check(ui.Offset{Dy: 300}, "pop halfway")
		ts.Pump(490 * time.Millisecond)
		check(ui.Offset{Dy: 594}, "pop end")
		ts.Pump(200 * time.Millisecond)
		if present(ts) {
			t.Error("the container should have transitioned out")
		}
	})
}

// testDialogRoute is _TestDialogRouteWithCustomBarrierCurve.
type testDialogRoute struct {
	PopupRouteBase
	child   Widget
	label   string
	color   ui.Color
	curve   animation.Curve
	hasCurv bool
}

func newTestDialogRoute(child Widget, configure func(*testDialogRoute)) *testDialogRoute {
	r := &testDialogRoute{child: child, color: routesTestBlack}
	if configure != nil {
		configure(r)
	}
	InitPopupRouteBase(&r.PopupRouteBase, r, ModalRouteOpts{})
	return r
}

func (r *testDialogRoute) BarrierDismissible() bool { return true }
func (r *testDialogRoute) BarrierLabel() string     { return r.label }
func (r *testDialogRoute) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.Some(r.color)
}
func (r *testDialogRoute) BarrierCurve() animation.Curve {
	if r.hasCurv {
		return r.curve
	}
	return r.PopupRouteBase.BarrierCurve()
}

// TransitionDuration is an easier value to test against.
func (r *testDialogRoute) TransitionDuration() time.Duration { return 100 * time.Millisecond }
func (r *testDialogRoute) BuildPage(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
	return r.child
}

func TestModalRouteBarrier(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*testDialogRoute)
		curve     animation.Curve
		start     ui.Color
		end       ui.Color
	}{
		{"default barrierCurve", nil, animation.Curves.Ease, routesTestTransparent, routesTestBlack},
		{"custom barrierCurve", func(r *testDialogRoute) { r.curve, r.hasCurv = animation.Curves.Linear, true }, animation.Curves.Linear, routesTestTransparent, routesTestBlack},
		{"white barrierColor", func(r *testDialogRoute) { r.color = routesTestWhite }, animation.Curves.Ease, routesTestWhite.WithOpacity(0), routesTestWhite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(navTestApp(navTestAppOpts{home: NewBuilder(func(bc BuildContext) Widget {
				return routesButton("X", func() {
					NavigatorPush(bc, newTestDialogRoute(navText("Hello World"), tc.configure))
				})
			})}))
			ts.Tap(gluttertest.ByText("X"))
			ts.Pump()
			barrier := func() foundation.Opt[ui.Color] {
				return ts.Widget(gluttertest.ByType[*AnimatedModalBarrier]()).(*AnimatedModalBarrier).Color().Value()
			}
			if got := barrier().OrElse(routesTestTransparent); got != tc.start {
				t.Fatalf("start color = %v, want %v", got, tc.start)
			}
			for _, fraction := range []float64{0.25, 0.5, 0.75} {
				ts.Pump(25 * time.Millisecond)
				want := ui.ColorGetAlphaFromOpacity(tc.curve.Transform(fraction))
				if got := barrier().OrElse(routesTestTransparent).Alpha(); math.Abs(float64(got-want)) > 1 {
					t.Errorf("alpha at %v = %d, want %d ± 1", fraction, got, want)
				}
			}
			ts.PumpAndSettle()
			if got := barrier().OrElse(routesTestTransparent); got != tc.end {
				t.Errorf("end color = %v, want %v", got, tc.end)
			}
		})
	}
}

// widgetWithLocalHistory is WidgetWithLocalHistory and WidgetWithNoLocalHistory.
type widgetWithLocalHistory struct {
	StatefulWidget
	register bool
}

func newWidgetWithLocalHistory(register bool) *widgetWithLocalHistory {
	return &widgetWithLocalHistory{NewStatefulWidgetBase(WidgetOpts{}), register}
}

func (w *widgetWithLocalHistory) CreateState() State { return &widgetWithLocalHistoryState{} }

type widgetWithLocalHistoryState struct {
	StateOf[*widgetWithLocalHistory]
	entry *LocalHistoryEntry
}

func (s *widgetWithLocalHistoryState) addLocalHistory() {
	s.entry = NewLocalHistoryEntry()
	if s.Widget().register {
		ModalRouteOf(s.Context()).AddLocalHistoryEntry(s.entry)
	}
}

func (s *widgetWithLocalHistoryState) Dispose() {

	s.entry.Remove()
}

func (s *widgetWithLocalHistoryState) Build(BuildContext) Widget { return NewText("dummy") }

func TestModalRouteChildWithLocalHistoryCanBeDisposed(t *testing.T) {
	for _, register := range []bool{true, false} {
		t.Run(fmt.Sprintf("registered=%v", register), func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(navTestApp(navTestAppOpts{home: newWidgetWithLocalHistory(register)}))
			gluttertest.StateOf[*widgetWithLocalHistoryState](ts, gluttertest.ByType[*widgetWithLocalHistory]()).addLocalHistory()
			// Waits for the modal route to update its internal state.
			ts.Pump()
			// A new widget disposes the state, which removes the entry from the
			// modal route while the tree is finalized.
			ts.PumpWidget(navTestApp(navTestAppOpts{home: navText("dummy")}))
			ts.Pump()
			if err := ts.TakeError(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestModalRouteRequestFocusCanBeUpdated(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	navState(t, key).Push(NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("page2") })))
	ts.PumpAndSettle()
	route := ModalRouteOf(ts.Element(gluttertest.ByText("page2")))
	if route == nil || !route.RequestFocus() {
		t.Fatal("the modal route should request focus")
	}
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if textsFound(ts, "page2", false) != 0 {
		t.Fatal("page2 should be gone")
	}
	navState(t, key).Push(NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("page2") }),
		PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{RequestFocus: foundation.Some(false)}}}))
	ts.PumpAndSettle()
	route = ModalRouteOf(ts.Element(gluttertest.ByText("page2")))
	if route == nil || route.RequestFocus() {
		t.Error("the modal route should not request focus")
	}
}

// routesRebuildCounter counts the builds of a Builder that reads a ModalRoute aspect.
func TestModalRouteAspectsOnlyRebuildWhenTheyChange(t *testing.T) {
	for _, tc := range []struct {
		name string
		read func(bc BuildContext) string
		want string
	}{
		{"isFirstOf", func(bc BuildContext) string { return fmt.Sprintf("isFirst: %v", ModalRouteIsFirstOf(bc).OrElse(false)) }, "isFirst: true"},
		{"isActiveOf", func(bc BuildContext) string {
			return fmt.Sprintf("isActive: %v", ModalRouteIsActiveOf(bc).OrElse(false))
		}, "isActive: true"},
		{"opaqueOf", func(bc BuildContext) string { return fmt.Sprintf("isOpaque: %v", ModalRouteOpaqueOf(bc).OrElse(false)) }, "isOpaque: true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := NewGlobalKey[*NavigatorState]()
			builds := 0
			ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: NewBuilder(func(bc BuildContext) Widget {
				builds++
				return NewText(tc.read(bc))
			})}))
			if builds != 1 || textsFound(ts, tc.want, false) != 1 {
				t.Fatalf("builds = %d, want 1 with %q", builds, tc.want)
			}
			// The first route stays first, active and opaque.
			navState(t, key).Push(NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("New Route") })))
			ts.PumpAndSettle()
			if builds != 1 {
				t.Errorf("builds = %d, want 1: the aspect did not change", builds)
			}
		})
	}
}

func TestModalRoutePopDispositionOfRebuildsWhenPopEntryChanges(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	builds := 0
	build := func(bc BuildContext) Widget {
		builds++
		d := ModalRoutePopDispositionOf(bc)
		return NewText("popDisposition: " + dispositionName(d.OrElse(RoutePopDispositionPop)))
	}
	// WidgetsApp reads its home when the page is built, so the page below
	// follows the current value.
	var current Widget = NewBuilder(build)
	home := func(BuildContext) Widget { return current }
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, homeBuilder: home}))
	if builds != 1 || textsFound(ts, "popDisposition: bubble", false) != 1 {
		t.Fatalf("builds = %d, want 1 with bubble", builds)
	}
	current = NewPopScope(NewBuilder(build), PopScopeOpts{CanPop: foundation.Some(false)})
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, homeBuilder: home}))
	ts.PumpAndSettle()
	if builds != 2 || textsFound(ts, "popDisposition: doNotPop", false) != 1 {
		t.Fatalf("builds = %d, want 2 with doNotPop", builds)
	}
	navState(t, key).Push(NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("New Route") })))
	ts.PumpAndSettle()
	if builds != 2 {
		t.Errorf("builds = %d, want 2: the disposition did not change", builds)
	}
}

func dispositionName(d RoutePopDisposition) string {
	switch d {
	case RoutePopDispositionPop:
		return "pop"
	case RoutePopDispositionDoNotPop:
		return "doNotPop"
	case RoutePopDispositionBubble:
		return "bubble"
	}
	return "?"
}

func TestModalRouteOfWorksForPlainRoutes(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	navState(t, key).Push(NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("page2") })))
	ts.PumpAndSettle()
	if textsFound(ts, "page2", false) != 1 {
		t.Fatal("page2 should show")
	}
	parent := ModalRouteOf(ts.Element(gluttertest.ByText("page2")))
	if _, ok := parent.(*PageRouteBuilder); !ok {
		t.Errorf("ModalRouteOf = %T, want a *PageRouteBuilder", parent)
	}
}

func TestNavigationNotifications(t *testing.T) {
	var notifications []NavigationNotification
	ts := gluttertest.New(t)
	ts.PumpWidget(NewNotificationListener(
		NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			InitialRoute: "/",
			OnGenerateRoute: func(settings RouteSettings) Route {
				return NewPageRouteBuilder(routesSizedBox(), PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
			},
		})),
		func(n NavigationNotification) bool {
			notifications = append(notifications, n)
			return true
		}))
	// One notification, from the initial route, where a pop can't be handled
	// because there is no other route to pop.
	if len(notifications) != 1 || notifications[0].CanHandlePop {
		t.Errorf("notifications = %v, want one with CanHandlePop false", notifications)
	}
}
