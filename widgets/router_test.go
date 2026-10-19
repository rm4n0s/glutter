package widgets_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/router_test.dart.
// buildBoilerPlate (TestWidgetsApp) is a Directionality; the messages of the
// 'flutter/navigation' channel (pushRouteInformation, popRoute) are calls to the
// binding, and the log of the SystemNavigator is the OnRouteInformationUpdated
// option of the provider (SystemNavigator comes in step 10.4). Not ported: the
// memory allocation event test and the RouteInformation.location getter
// (deprecated, not ported).

type routeInfo = RouteInformation

// flushAll runs the posted functions until there are none, then pumps.
func flushAll(ts *gluttertest.Tester) {
	for range 4 {
		ts.FlushPosted()
	}
	ts.Pump()
}

func routerOf(delegate RouterDelegate[*routeInfo], opts RouterOpts[*routeInfo]) Widget {
	return ltr(NewRouter(delegate, opts))
}

func infoText(info *routeInfo) Widget { return NewText(decoded(info)) }

func shows(ts *gluttertest.Tester, text string) bool { return textsFound(ts, text, false) == 1 }

func TestRouterSimpleSynchronized(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, nil)
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider, RouteInformationParser: newSimpleParser()}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	provider.setValue(uriInfo("update"))
	ts.Pump()
	if shows(ts, "initial") || !shows(ts, "update") {
		t.Error("update should replace initial")
	}
}

func TestRouterRespectsUpdateOrder(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newMutableDelegate()
	notifier := foundation.NewValueNotifier(0)
	ts.PumpWidget(ltr(newIntInherited(notifier, NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser: newCustomParser(func(info *routeInfo, bc BuildContext) *routeInfo {
			intInheritedOf(bc) // create dependency
			return info
		}),
	}))))
	if !shows(ts, "initial") || delegate.configuration.URI().String() != "initial" {
		t.Fatal("initial should show")
	}
	delegate.updateConfiguration(uriInfo("update"))
	notifier.SetValue(1)
	// The delegate should still retain the update.
	ts.PumpAndSettle()
	if !shows(ts, "update") || delegate.configuration.URI().String() != "update" {
		t.Errorf("the delegate lost the update: %v", delegate.configuration.URI())
	}
}

func TestRouterSimpleAsynchronized(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	parser := newAsyncParser()
	delegate := newAsyncDelegate(func(_ BuildContext, i *routeInfo) Widget {
		if i == nil {
			return NewText("waiting")
		}
		return NewText(i.URI().String())
	})
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider, RouteInformationParser: parser}))
	// The future has not completed yet.
	if !shows(ts, "waiting") {
		t.Fatal("waiting should show")
	}
	parser.complete(0)
	flushAll(ts)
	delegate.setNewRoute[0].Complete(struct{}{})
	flushAll(ts)
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	provider.setValue(uriInfo("update"))
	ts.Pump()
	// The future has not completed yet.
	if !shows(ts, "initial") {
		t.Fatal("initial should still show")
	}
	parser.complete(1)
	flushAll(ts)
	delegate.setNewRoute[1].Complete(struct{}{})
	flushAll(ts)
	if !shows(ts, "update") {
		t.Error("update should show")
	}
}

func TestRouterInterruptedParsingDoesNotCrash(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	parser := newAsyncParser()
	delegate := newAsyncDelegate(func(_ BuildContext, i *routeInfo) Widget {
		if i == nil {
			return NewText("waiting")
		}
		return NewText(i.URI().String())
	})
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider, RouteInformationParser: parser}))
	if !shows(ts, "waiting") {
		t.Fatal("waiting should show")
	}
	// Starts a new parsing transaction before the previous one completes.
	provider.setValue(uriInfo("update"))
	ts.Pump()
	if !shows(ts, "waiting") {
		t.Fatal("waiting should still show")
	}
	// Completing the previous transaction does not cause an update.
	parser.complete(0)
	flushAll(ts)
	if !shows(ts, "waiting") || len(delegate.setNewRoute) != 0 {
		t.Fatal("the stale transaction should be ignored")
	}
	if err := ts.TakeError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The new transaction completes and updates.
	parser.complete(1)
	flushAll(ts)
	delegate.setNewRoute[0].Complete(struct{}{})
	flushAll(ts)
	if !shows(ts, "update") {
		t.Error("update should show")
	}
}

func TestRouterMaybeOfCanBeNull(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewText("dummy")))
	bc := ts.Element(gluttertest.ByText("dummy"))
	if r := RouterMaybeOf[*routeInfo](bc); r != nil {
		t.Errorf("RouterMaybeOf = %v, want nil", r)
	}
	appErr := requireErrorTagFromPanic(t, panicOf(func() { RouterOf[*routeInfo](bc) }), "RouterMissing")
	if got := fmt.Sprint(appErr.Message); len(got) < 6 || got[:6] != "Router" {
		t.Errorf("message %q should start with Router", got)
	}
}

func TestRouterTypeMismatch(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	delegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return NewText("x") }, nil)
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{}))
	bc := ts.Element(gluttertest.ByText("x"))
	if RouterMaybeOf[string](bc) != nil {
		t.Error("a router of another type should not be found")
	}
	_ = requireErrorTagFromPanic(t, panicOf(func() { RouterOf[string](bc) }), "RouterTypeMismatch")
	if RouterOf[*routeInfo](bc) == nil {
		t.Error("RouterOf with the right type should find the router")
	}
}

func TestRouterSimpleCanHandlePopRoute(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	dispatcher := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, func() bool {
		provider.setValue(uriInfo("popped"))
		return true
	})
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), BackButtonDispatcher: dispatcher,
	}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	if !dispatcher.InvokeCallback(false) {
		t.Error("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped") {
		t.Error("popped should show")
	}
}

func TestRouterThrowsWithProviderWithoutParser(t *testing.T) {
	skipUnlessDebug(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, nil)
	requireTag(t, "RouterParserMissing", func() {
		NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider})
	})
}

func TestPopNavigatorRouterDelegateWorks(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	dispatcher := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newNavigatorDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, func(route Route, result any) bool {
		provider.setValue(uriInfo("popped"))
		return route.DidPop(result)
	})
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), BackButtonDispatcher: dispatcher,
	}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	// Pushes a nameless route.
	state, _ := delegate.navigatorKey.CurrentState()
	state.Push(NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewText("dialog")
	}))
	ts.PumpAndSettle()
	if !shows(ts, "dialog") {
		t.Fatal("dialog should show")
	}
	// Pops the nameless route and makes sure the initial page is shown.
	if !dispatcher.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.PumpAndSettle()
	if !shows(ts, "initial") || textsFound(ts, "dialog", true) != 0 {
		t.Fatal("initial should show without the dialog")
	}
	// Pops one more time.
	if !dispatcher.InvokeCallback(false) {
		t.Fatal("the second pop should be handled")
	}
	ts.PumpAndSettle()
	if !shows(ts, "popped") {
		t.Error("popped should show")
	}
}

func TestRouterNestedBackButtonDispatcher(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	outerDelegate := newSimpleDelegate(func(_ BuildContext, info *routeInfo) Widget {
		inner := NewChildBackButtonDispatcher(outer)
		inner.TakePriority()
		innerDelegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return infoText(info) }, func() bool {
			provider.setValue(uriInfo("popped inner"))
			return true
		})
		return NewRouter[*routeInfo](innerDelegate, RouterOpts[*routeInfo]{BackButtonDispatcher: inner})
	}, func() bool {
		provider.setValue(uriInfo("popped outer"))
		return true
	})
	ts.PumpWidget(routerOf(outerDelegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: outer, RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	// The outer dispatcher triggers the pop on the inner router.
	if !outer.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped inner") {
		t.Error("popped inner should show")
	}
}

func TestRouterNestedBackButtonDispatcherMultipleChildren(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	inner1 := NewChildBackButtonDispatcher(outer)
	inner2 := NewChildBackButtonDispatcher(outer)
	popping := func(name string) *simpleDelegate {
		return newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return NewContainer(nil) }, func() bool {
			provider.setValue(uriInfo(name))
			return true
		})
	}
	outerDelegate := newSimpleDelegate(func(_ BuildContext, info *routeInfo) Widget {
		return NewColumn([]Widget{
			infoText(info),
			NewRouter[*routeInfo](popping("popped inner1"), RouterOpts[*routeInfo]{BackButtonDispatcher: inner1}),
			NewRouter[*routeInfo](popping("popped inner2"), RouterOpts[*routeInfo]{BackButtonDispatcher: inner2}),
		})
	}, func() bool {
		provider.setValue(uriInfo("popped outer"))
		return true
	})
	ts.PumpWidget(routerOf(outerDelegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: outer, RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	step := func(want string) {
		t.Helper()
		if !outer.InvokeCallback(false) {
			t.Fatalf("the pop should be handled (%s)", want)
		}
		ts.Pump()
		if !shows(ts, want) {
			t.Fatalf("%q should show", want)
		}
	}
	// If none of the children have taken the priority, the root router handles the pop.
	step("popped outer")
	inner1.TakePriority()
	step("popped inner1")
	// The last child dispatcher that took priority handles the pop.
	inner2.TakePriority()
	step("popped inner2")
}

func TestChildBackButtonDispatcherCanBeReplacedWithoutTakePriority(t *testing.T) {
	ts := gluttertest.New(t)
	outer := NewRootBackButtonDispatcher(ts.Binding)
	app := func(inner BackButtonDispatcher) Widget {
		outerDelegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget {
			innerDelegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return NewContainer(nil) }, nil)
			return NewColumn([]Widget{NewText("initial"), NewRouter[*routeInfo](innerDelegate, RouterOpts[*routeInfo]{BackButtonDispatcher: inner})})
		}, nil)
		return routerOf(outerDelegate, RouterOpts[*routeInfo]{BackButtonDispatcher: outer})
	}
	ts.PumpWidget(app(NewChildBackButtonDispatcher(outer)))
	// A new child dispatcher replaces the old one, which is discarded.
	ts.PumpWidget(app(NewChildBackButtonDispatcher(outer)))
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChildBackButtonDispatcherTakesPriorityRecursively(t *testing.T) {
	ts := gluttertest.New(t)
	outer := NewRootBackButtonDispatcher(ts.Binding)
	inner1 := NewChildBackButtonDispatcher(outer)
	inner2 := NewChildBackButtonDispatcher(inner1)
	inner3 := NewChildBackButtonDispatcher(inner2)
	isPopped := false
	leaf := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return NewContainer(nil) }, func() bool {
		isPopped = true
		return true
	})
	level := func(d BackButtonDispatcher, build func() Widget) Widget {
		delegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return build() }, nil)
		return NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{BackButtonDispatcher: d})
	}
	ts.PumpWidget(ltr(level(outer, func() Widget {
		return level(inner1, func() Widget {
			return level(inner2, func() Widget {
				return NewRouter[*routeInfo](leaf, RouterOpts[*routeInfo]{BackButtonDispatcher: inner3})
			})
		})
	})))
	// This works without calling takePriority on inner2 and inner1.
	inner3.TakePriority()
	if !outer.InvokeCallback(false) || !isPopped {
		t.Errorf("the leaf should pop: popped %v", isPopped)
	}
}

// reports records what the router reports to a provider.
type reports struct {
	info *routeInfo
	typ  RouteInformationReportingType
	n    int
}

func (r *reports) clear() { *r = reports{} }

func (r *reports) record(t *testing.T) func(*routeInfo, RouteInformationReportingType) {
	return func(info *routeInfo, typ RouteInformationReportingType) {
		t.Helper()
		if r.n != 0 {
			t.Errorf("reported twice before the test cleaned up")
		}
		r.info, r.typ = info, typ
		r.n++
	}
}

func TestRouterReportsURLChange(t *testing.T) {
	ts := gluttertest.New(t)
	var rep reports
	provider := newSimpleProvider("initial")
	provider.onRouterReport = rep.record(t)
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, nil)
	delegate.reportConfiguration = true
	delegate.onPopRoute = func() bool {
		delegate.setRouteInformation(uriInfo("popped"))
		return true
	}
	outer := NewRootBackButtonDispatcher(ts.Binding)
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: outer, RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") || rep.info.URI().String() != "initial" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("initial report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	delegate.setRouteInformation(uriInfo("update"))
	ts.Pump()
	if shows(ts, "initial") || !shows(ts, "update") || rep.info.URI().String() != "update" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("update report: %v %v", rep.info, rep.typ)
	}
	// The router reports as a non navigation event if only the state changes.
	rep.clear()
	delegate.setRouteInformation(stateInfo("update", "another state"))
	ts.Pump()
	if !shows(ts, "update") || rep.info.URI().String() != "update" || rep.info.State() != "another state" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("state report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	if !outer.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped") || rep.info.URI().String() != "popped" || rep.typ != RouteInformationReportingTypeNone {
		t.Errorf("popped report: %v %v", rep.info, rep.typ)
	}
}

func TestRouterCanBeForcedToRecognizeOrIgnoreNavigatingEvents(t *testing.T) {
	ts := gluttertest.New(t)
	var rep reports
	isNavigating := false
	var next *routeInfo
	provider := newSimpleProvider("initial")
	provider.onRouterReport = rep.record(t)
	delegate := newSimpleDelegate(nil, nil)
	delegate.reportConfiguration = true
	delegate.builder = func(_ BuildContext, i *routeInfo) Widget {
		// A button with state: tapping it rebuilds even when the route
		// information does not change, which makes the router report.
		return newBuilder(func(bc BuildContext, setState func(func())) Widget {
			return testButton(NewSizedBox(NewText(decoded(i)), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)}), func() {
				setState(func() {})
				change := func() {
					if delegate.routeInformation != next {
						delegate.setRouteInformation(next)
					}
				}
				if isNavigating {
					RouterNavigate(bc, change)
				} else {
					RouterNeglect(bc, change)
				}
			})
		})
	}
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding), RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") || rep.info.URI().String() != "initial" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("initial report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	next = uriInfo("update")
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.Pump()
	if shows(ts, "initial") || !shows(ts, "update") || rep.typ != RouteInformationReportingTypeNeglect || rep.info.URI().String() != "update" {
		t.Fatalf("neglect report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	isNavigating = true
	// No real navigating event happens because the next route information does
	// not change, but the router still reports because isNavigating is set.
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.Pump()
	if rep.typ != RouteInformationReportingTypeNavigate || rep.info.URI().String() != "update" {
		t.Errorf("navigate report: %v %v", rep.info, rep.typ)
	}
}

func TestRouterIgnoreNavigatingEventsUpdatesProvider(t *testing.T) {
	ts := gluttertest.New(t)
	var rep reports
	var next *routeInfo
	provider := newSimpleProvider("initial")
	provider.onRouterReport = rep.record(t)
	delegate := newSimpleDelegate(nil, nil)
	delegate.reportConfiguration = true
	delegate.builder = func(bc BuildContext, i *routeInfo) Widget {
		return testButton(NewSizedBox(NewText(decoded(i)), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)}), func() {
			RouterNeglect(bc, func() {
				if delegate.routeInformation != next {
					delegate.setRouteInformation(next)
				}
			})
		})
	}
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding), RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") || rep.info.URI().String() != "initial" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("initial report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	next = uriInfo("update")
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.Pump()
	if shows(ts, "initial") || !shows(ts, "update") || rep.info.URI().String() != "update" || rep.typ != RouteInformationReportingTypeNeglect {
		t.Errorf("neglect report: %v %v", rep.info, rep.typ)
	}
}

func TestRouterStateChangeWithoutLocationChangeUpdatesProvider(t *testing.T) {
	ts := gluttertest.New(t)
	var rep reports
	var next *routeInfo
	provider := newSimpleProvider("initial")
	provider.value = stateInfo("initial", "state1")
	provider.onRouterReport = rep.record(t)
	delegate := newSimpleDelegate(nil, nil)
	delegate.reportConfiguration = true
	delegate.builder = func(_ BuildContext, i *routeInfo) Widget {
		return testButton(NewSizedBox(NewText(decoded(i)), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)}), func() {
			delegate.setRouteInformation(next)
		})
	}
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding), RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") || rep.info.URI().String() != "initial" || rep.typ != RouteInformationReportingTypeNone {
		t.Fatalf("initial report: %v %v", rep.info, rep.typ)
	}
	rep.clear()
	next = stateInfo("initial", "state2")
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.Pump()
	if rep.info.URI().String() != "initial" || rep.info.State() != "state2" || rep.typ != RouteInformationReportingTypeNone {
		t.Errorf("state report: %v %v", rep.info, rep.typ)
	}
}

func TestRouterReportsAmbiguousIntention(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(nil, nil)
	delegate.builder = func(bc BuildContext, i *routeInfo) Widget {
		return testButton(NewSizedBox(NewText(decoded(i)), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)}), func() {
			RouterNeglect(bc, func() {})
			RouterNavigate(bc, func() {})
		})
	}
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider, RouteInformationParser: newSimpleParser()}))
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.Pump()
	_ = requireErrorTag(t, takeFirst(t, ts), "RouterReportAmbiguous")
}

func TestPlatformRouteInformationProviderWorks(t *testing.T) {
	ts := gluttertest.New(t)
	provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		children := []Widget{}
		if i.URI().String() != "" {
			children = append(children, NewText(i.URI().String()))
		}
		if i.State() != nil {
			children = append(children, NewText(fmt.Sprint(i.State())))
		}
		return NewColumn(children)
	}, nil)
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{RouteInformationProvider: provider, RouteInformationParser: newSimpleParser()}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	// Pushes through the platform's pushRouteInformation.
	if !ts.Binding.HandlePushRouteInformation(stateInfo("testRouteName", "state")) {
		t.Fatal("the push should be handled")
	}
	ts.Pump()
	if !shows(ts, "testRouteName") || !shows(ts, "state") {
		t.Fatal("testRouteName and its state should show")
	}
	// Pushes through the platform's pushRoute.
	ts.Binding.HandlePushRouteInformation(uriInfo("newTestRouteName"))
	ts.Pump()
	if !shows(ts, "newTestRouteName") {
		t.Error("newTestRouteName should show")
	}
}

func TestRootBackButtonDispatcherWorks(t *testing.T) {
	ts := gluttertest.New(t)
	outer := NewRootBackButtonDispatcher(ts.Binding)
	provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, nil)
	delegate.reportConfiguration = true
	delegate.onPopRoute = func() bool {
		delegate.setRouteInformation(uriInfo("popped"))
		return true
	}
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: outer, RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	}))
	if !shows(ts, "initial") {
		t.Fatal("initial should show")
	}
	// Pops the route as the platform does.
	if !ts.Binding.HandlePopRoute() {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped") {
		t.Error("popped should show")
	}
}

// listenerApp is the app of the BackButtonListener tests: a router whose delegate
// builds the information and the children.
func listenerApp(ts *gluttertest.Tester, provider *simpleProvider, outer BackButtonDispatcher, delegate *simpleDelegate) Widget {
	return routerOf(delegate, RouterOpts[*routeInfo]{
		BackButtonDispatcher: outer, RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(),
	})
}

func poppingTo(provider *simpleProvider, uri string, result bool) BackButtonCallback {
	return func() bool {
		provider.setValue(uriInfo(uri))
		return result
	}
}

func poppedOuter(provider *simpleProvider) func() bool {
	return func() bool {
		provider.setValue(uriInfo("popped outer"))
		return true
	}
}

func TestBackButtonListenerTakesPriorityOverRoot(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		return NewColumn([]Widget{infoText(i), NewBackButtonListener(NewContainer(nil), poppingTo(provider, "popped inner1", true))})
	}, poppedOuter(provider))
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	if !shows(ts, "initial") || !outer.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped inner1") {
		t.Error("popped inner1 should show")
	}
}

func TestBackButtonListenerUpdatesCallbackIfChanged(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(nil, poppedOuter(provider))
	build := func(name string) func(BuildContext, *routeInfo) Widget {
		return func(_ BuildContext, i *routeInfo) Widget {
			return NewColumn([]Widget{infoText(i), NewBackButtonListener(NewContainer(nil), poppingTo(provider, name, true))})
		}
	}
	delegate.builder = build("first callback")
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	delegate.builder = build("second callback")
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	ts.Pump()
	outer.InvokeCallback(false)
	ts.Pump()
	if !shows(ts, "second callback") {
		t.Error("second callback should show")
	}
}

func TestBackButtonListenerClearsCallbackIfDisposed(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(nil, poppedOuter(provider))
	delegate.builder = func(_ BuildContext, i *routeInfo) Widget {
		return NewColumn([]Widget{infoText(i), NewBackButtonListener(NewContainer(nil), poppingTo(provider, "first callback", true))})
	}
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	delegate.builder = func(_ BuildContext, i *routeInfo) Widget { return NewColumn([]Widget{infoText(i)}) }
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	ts.Pump()
	outer.InvokeCallback(false)
	ts.Pump()
	if !shows(ts, "popped outer") {
		t.Error("popped outer should show")
	}
}

func TestNestedBackButtonListenerTakesPriority(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		return NewColumn([]Widget{infoText(i), NewBackButtonListener(
			NewBackButtonListener(NewContainer(nil), poppingTo(provider, "popped inner2", true)),
			poppingTo(provider, "popped inner1", true))})
	}, poppedOuter(provider))
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	if !shows(ts, "initial") || !outer.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped inner2") {
		t.Error("popped inner2 should show")
	}
}

func TestNestedBackButtonListenerReturningFalseCallsNext(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		return NewColumn([]Widget{infoText(i), NewBackButtonListener(
			NewBackButtonListener(NewContainer(nil), poppingTo(provider, "popped inner2", false)),
			poppingTo(provider, "popped inner1", true))})
	}, poppedOuter(provider))
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	if !shows(ts, "initial") || !outer.InvokeCallback(false) {
		t.Fatal("the pop should be handled")
	}
	ts.Pump()
	if !shows(ts, "popped inner1") {
		t.Error("popped inner1 should show")
	}
}

func TestRouterDidUpdateWidgetOnlyUpdatesBackButtonListener(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	outer := NewRootBackButtonDispatcher(ts.Binding)
	location := "first callback"
	var setState func(func())
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		return NewColumn([]Widget{infoText(i), newBuilder(func(_ BuildContext, setter func(func())) Widget {
			setState = setter
			return NewBackButtonListener(NewContainer(nil), func() bool {
				provider.setValue(uriInfo(location))
				return true
			})
		})})
	}, poppedOuter(provider))
	ts.PumpWidget(listenerApp(ts, provider, outer, delegate))
	// Only the BackButtonListener widget updates.
	setState(func() { location = "second callback" })
	ts.Pump()
	outer.InvokeCallback(false)
	ts.Pump()
	if !shows(ts, "second callback") {
		t.Error("second callback should show")
	}
}

func TestRouterReportsLocationDifferentFromOS(t *testing.T) {
	ts := gluttertest.New(t)
	var reported []*routeInfo
	provider := newSimpleProvider("/home")
	provider.onRouterReport = func(i *routeInfo, _ RouteInformationReportingType) { reported = append(reported, i) }
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget {
		return NewText("Current route: " + i.URI().String())
	}, nil)
	delegate.reportConfiguration = true
	ts.PumpWidget(routerOf(delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser:   newRedirectingParser(map[string]*routeInfo{"/doesNotExist": uriInfo("/404")}),
	}))
	if !shows(ts, "Current route: /home") || len(reported) != 1 || reported[0].URI().String() != "/home" {
		t.Fatalf("home: %d reports", len(reported))
	}
	provider.setValue(uriInfo("/doesNotExist"))
	ts.Pump()
	if !shows(ts, "Current route: /404") || len(reported) < 2 || reported[1].URI().String() != "/404" {
		t.Errorf("404: %d reports", len(reported))
	}
}

func TestRouterParserCanLookUpDependenciesAndReparse(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, poppedOuter(provider))
	parserCalled := false
	router := NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser: newCustomParser(func(_ *routeInfo, bc BuildContext) *routeInfo {
			parserCalled = true
			return uriInfo(fmt.Sprint(DefaultTextStyleOf(bc).MaxLines().Value()))
		}),
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding),
	})
	app := func(maxLines int) Widget {
		return ltr(NewDefaultTextStyle(painting.NewTextStyle(), router, DefaultTextStyleOpts{MaxLines: foundation.Some(maxLines)}))
	}
	ts.PumpWidget(app(1))
	if !shows(ts, "1") || !parserCalled {
		t.Fatal("1 should show after parsing")
	}
	parserCalled = false
	ts.PumpWidget(app(2))
	ts.Pump()
	if !shows(ts, "2") || !parserCalled {
		t.Error("2 should show after parsing again")
	}
}

func TestRouterParserCanLookUpDependenciesWithoutReparsing(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, poppedOuter(provider))
	parserCalled := false
	router := NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser: newCustomParser(func(_ *routeInfo, bc BuildContext) *routeInfo {
			parserCalled = true
			style, _ := GetInherited[*DefaultTextStyle](bc)
			return uriInfo(fmt.Sprint(style.MaxLines().Value()))
		}),
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding),
	})
	app := func(maxLines int) Widget {
		return ltr(NewDefaultTextStyle(painting.NewTextStyle(), router, DefaultTextStyleOpts{MaxLines: foundation.Some(maxLines)}))
	}
	ts.PumpWidget(app(1))
	if !shows(ts, "1") || !parserCalled {
		t.Fatal("1 should show after parsing")
	}
	parserCalled = false
	// This rebuild does not trigger parsing again.
	ts.PumpWidget(app(2))
	ts.Pump()
	if shows(ts, "2") || !shows(ts, "1") || parserCalled {
		t.Errorf("no re-parse expected: parser called %v", parserCalled)
	}
}

func TestRouterDelegateDependenciesDoNotTriggerReparsing(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("initial")
	delegate := newSimpleDelegate(func(bc BuildContext, _ *routeInfo) Widget {
		return NewText(fmt.Sprint(DefaultTextStyleOf(bc).MaxLines().Value()))
	}, poppedOuter(provider))
	parserCalled := false
	router := NewRouter[*routeInfo](delegate, RouterOpts[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser: newCustomParser(func(i *routeInfo, _ BuildContext) *routeInfo {
			parserCalled = true
			return i
		}),
		BackButtonDispatcher: NewRootBackButtonDispatcher(ts.Binding),
	})
	app := func(maxLines int) Widget {
		return ltr(NewDefaultTextStyle(painting.NewTextStyle(), router, DefaultTextStyleOpts{MaxLines: foundation.Some(maxLines)}))
	}
	ts.PumpWidget(app(1))
	// The initial route is parsed regardless.
	if !shows(ts, "1") || !parserCalled {
		t.Fatal("1 should show after parsing")
	}
	parserCalled = false
	ts.PumpWidget(app(2))
	ts.Pump()
	if !shows(ts, "2") || parserCalled {
		t.Errorf("2 should show without parsing: parser called %v", parserCalled)
	}
}

func TestRouterCanInitializeWithRouterConfig(t *testing.T) {
	ts := gluttertest.New(t)
	provider := newSimpleProvider("/")
	delegate := newSimpleDelegate(func(BuildContext, *routeInfo) Widget { return NewText("text") }, nil)
	config := RouterConfig[*routeInfo]{
		RouteInformationProvider: provider,
		RouteInformationParser:   newSimpleParser(),
		RouterDelegate:           delegate,
		BackButtonDispatcher:     NewRootBackButtonDispatcher(ts.Binding),
	}
	router := NewRouterWithConfig(config)
	if router.RouterDelegate() != RouterDelegate[*routeInfo](delegate) ||
		router.RouteInformationParser() != config.RouteInformationParser ||
		router.RouteInformationProvider() != RouteInformationProvider(provider) ||
		router.BackButtonDispatcher() != config.BackButtonDispatcher {
		t.Error("the router should hold the objects of the config")
	}
	ts.PumpWidget(ltr(router))
	if !shows(ts, "text") {
		t.Error("text should show")
	}
}

func TestRouteInformationURIAPI(t *testing.T) {
	tests := []struct {
		name, in, scheme, host, path, fragment string
		query                                  map[string][]string
	}{
		{"relative", "/a?abc=def&abc=jkl#mno", "", "", "/a", "mno", map[string][]string{"abc": {"def", "jkl"}}},
		{"bare", "1", "", "", "1", "", map[string][]string{}},
		{"absolute", "http://mydomain.com/abc?def=ghi&def=jkl#mno", "http", "mydomain.com", "/abc", "mno", map[string][]string{"def": {"ghi", "jkl"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := uriInfo(tc.in).URI()
			if u.String() != tc.in || u.Scheme != tc.scheme || u.Hostname() != tc.host || u.Path != tc.path || u.Fragment != tc.fragment {
				t.Errorf("uri %v: scheme %q host %q path %q fragment %q", u, u.Scheme, u.Hostname(), u.Path, u.Fragment)
			}
			q := u.Query()
			if len(q) != len(tc.query) {
				t.Fatalf("query %v, want %v", q, tc.query)
			}
			for k, want := range tc.query {
				if got := q[k]; fmt.Sprint(got) != fmt.Sprint(want) {
					t.Errorf("query %s = %v, want %v", k, got, want)
				}
			}
		})
	}
}

type routeUpdate struct {
	uri     string
	state   any
	replace bool
}

func TestPlatformRouteInformationProviderUpdatesRouteInformation(t *testing.T) {
	ts := gluttertest.New(t)
	var log []routeUpdate
	provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"), PlatformRouteInformationProviderOpts{
		OnRouteInformationUpdated: func(i *routeInfo, replace bool) {
			log = append(log, routeUpdate{i.URI().String(), i.State(), replace})
		},
	})
	steps := []struct {
		info *routeInfo
		typ  RouteInformationReportingType
		want routeUpdate
	}{
		// Implicit reporting pushes a new history entry if the location changes.
		{stateInfo("a", true), RouteInformationReportingTypeNone, routeUpdate{"a", true, false}},
		// Since the location is the same, the provider replaces.
		{stateInfo("a", false), RouteInformationReportingTypeNone, routeUpdate{"a", false, true}},
		{stateInfo("b", false), RouteInformationReportingTypeNeglect, routeUpdate{"b", false, true}},
		{stateInfo("b", false), RouteInformationReportingTypeNavigate, routeUpdate{"b", false, false}},
	}
	for i, step := range steps {
		log = nil
		provider.RouterReportsNewRouteInformation(step.info, step.typ)
		if len(log) != 1 || log[0] != step.want {
			t.Errorf("step %d: log = %v, want %v", i, log, step.want)
		}
	}
}

func TestPlatformRouteInformationProviderDoesNotPushIfQueryIsSemanticallyTheSame(t *testing.T) {
	ts := gluttertest.New(t)
	var log []routeUpdate
	initial := uriInfo("initial?a=ws/abcd")
	provider := NewPlatformRouteInformationProvider(ts.Binding, initial, PlatformRouteInformationProviderOpts{
		OnRouteInformationUpdated: func(i *routeInfo, replace bool) {
			log = append(log, routeUpdate{i.URI().String(), i.State(), replace})
		},
	})
	// Makes sure the engine is updated with the initial route.
	provider.RouterReportsNewRouteInformation(initial, RouteInformationReportingTypeNone)
	log = nil
	escaped := NewRouteInformation(RouteInformationOpts{URI: &url.URL{Path: "initial", RawQuery: url.Values{"a": {"ws/abcd"}}.Encode()}})
	provider.RouterReportsNewRouteInformation(escaped, RouteInformationReportingTypeNone)
	if got := provider.Value().URI().String(); got != "initial?a=ws%2Fabcd" {
		t.Errorf("value = %q", got)
	}
	if len(log) != 1 || log[0] != (routeUpdate{"initial?a=ws%2Fabcd", nil, true}) {
		t.Errorf("escaped query: %v", log)
	}
	for _, tc := range []struct{ first, second string }{
		{"initial?a=1&b=2", "initial?b=2&a=1"}, // other order of the parameters
		{"initial?a=1&a=2", "initial?a=2&a=1"}, // other order of the values of a key
	} {
		provider.RouterReportsNewRouteInformation(uriInfo(tc.first), RouteInformationReportingTypeNone)
		log = nil
		provider.RouterReportsNewRouteInformation(uriInfo(tc.second), RouteInformationReportingTypeNone)
		if len(log) != 1 || log[0] != (routeUpdate{tc.second, nil, true}) {
			t.Errorf("%s then %s: %v", tc.first, tc.second, log)
		}
	}
}
