package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/pop_scope_test.dart.
// Not ported: the checks of SystemNavigator.setFrameworkHandlesBack (Android
// only; the platform channel is step 10.4) and the TargetPlatform variants that
// only they depend on. The system back button is simulated as WidgetsApp does it:
// the binding tells its observers, and the navigator tries to pop.

func popScopeApp(key *GlobalKey[*NavigatorState], routes map[string]WidgetBuilder, home Widget) Widget {
	return navTestApp(navTestAppOpts{navigatorKey: key, routes: routes, home: home})
}

func popScopePage(text string) Widget { return NewCenter(NewText(text)) }

func TestPopScopeTogglingCanPopOnRootRouteAllowsOrPreventsBacks(t *testing.T) {
	ts := gluttertest.New(t)
	canPop := false
	var setState StateSetter
	var context BuildContext
	ts.PumpWidget(popScopeApp(nil, map[string]WidgetBuilder{
		"/": func(BuildContext) Widget {
			return NewStatefulBuilder(func(bc BuildContext, set StateSetter) Widget {
				context, setState = bc, set
				return NewPopScope(popScopePage("Home/PopScope Page"), PopScopeOpts{CanPop: foundation.Some(canPop)})
			})
		},
	}, nil))
	if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionDoNotPop {
		t.Fatalf("PopDisposition = %v, want doNotPop", got)
	}
	setState(func() { canPop = true })
	ts.Pump()
	if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionBubble {
		t.Errorf("PopDisposition = %v, want bubble", got)
	}
}

func TestPopScopeCanReceiveResult(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var received any
	popped := &struct{ n int }{1}
	ts.PumpWidget(popScopeApp(key, nil, NewPopScope(popScopePage("Home/PopScope Page"), PopScopeOpts{
		CanPop:                 foundation.Some(false),
		OnPopInvokedWithResult: func(didPop bool, result any) { received = result },
	})))
	navState(t, key).MaybePop(popped)
	ts.PumpAndSettle()
	if received != any(popped) {
		t.Errorf("received %v, want the popped result", received)
	}
}

func TestPopScopeResultOfAStricterRoute(t *testing.T) {
	// A PopScope without a result type, in a route that has one in Dart; routes
	// have no result type here, so this checks that the result reaches the scope.
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var received any
	record := func(didPop bool, result any) { received = result }
	ts.PumpWidget(popScopeApp(key, nil, NewPopScope(popScopePage("Home/PopScope Page"), PopScopeOpts{
		CanPop: foundation.Some(false), OnPopInvokedWithResult: record,
	})))
	navState(t, key).Push(NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewPopScope(popScopePage("new page"), PopScopeOpts{CanPop: foundation.Some(false), OnPopInvokedWithResult: record})
	}))
	ts.PumpAndSettle()
	if textsFound(ts, "new page", false) != 1 {
		t.Fatal("the new page should show")
	}
	navState(t, key).MaybePop(13)
	ts.PumpAndSettle()
	if received != 13 {
		t.Errorf("received %v, want 13", received)
	}
}

func TestPopScopeTogglingCanPopOnSecondaryRoute(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	canPop := true
	var setState StateSetter
	var homeContext, oneContext BuildContext
	var lastPopSuccess bool
	ts.PumpWidget(popScopeApp(key, map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			homeContext = bc
			return NewCenter(NewColumn([]Widget{
				navText("Home Page"),
				NewGestureDetector(navText("Next"), GestureDetectorOpts{
					OnTap:    func() { NavigatorPushNamed(bc, "/one") },
					Behavior: rendering.HitTestBehaviorOpaque,
				}),
			}, ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}))
		},
		"/one": func(BuildContext) Widget {
			return NewStatefulBuilder(func(bc BuildContext, set StateSetter) Widget {
				oneContext, setState = bc, set
				return NewPopScope(popScopePage("PopScope Page"), PopScopeOpts{
					CanPop:                 foundation.Some(canPop),
					OnPopInvokedWithResult: func(didPop bool, _ any) { lastPopSuccess = didPop },
				})
			})
		},
	}, nil))

	disposition := func(bc BuildContext, want RoutePopDisposition) {
		t.Helper()
		if got := ModalRouteOf(bc).PopDisposition(); got != want {
			t.Errorf("PopDisposition = %v, want %v", got, want)
		}
	}
	shows := func(shown, hidden string) {
		t.Helper()
		if textsFound(ts, shown, false) != 1 || textsFound(ts, hidden, false) != 0 {
			t.Fatalf("%q should show and %q should not", shown, hidden)
		}
	}
	systemBack := func() {
		// WidgetsApp's didPopRoute.
		navState(t, key).MaybePop()
	}
	next := func() {
		ts.Tap(gluttertest.ByText("Next"))
		ts.PumpAndSettle()
		shows("PopScope Page", "Home Page")
		disposition(oneContext, RoutePopDispositionPop)
	}

	shows("Home Page", "PopScope Page")
	disposition(homeContext, RoutePopDispositionBubble)
	next()

	navState(t, key).MaybePop()
	ts.PumpAndSettle()
	if !lastPopSuccess {
		t.Error("the pop should have succeeded")
	}
	shows("Home Page", "PopScope Page")
	disposition(homeContext, RoutePopDispositionBubble)
	next()

	systemBack()
	ts.PumpAndSettle()
	if !lastPopSuccess {
		t.Error("the system back should have popped")
	}
	shows("Home Page", "PopScope Page")
	next()

	setState(func() { canPop = false })
	ts.Pump()
	navState(t, key).MaybePop()
	ts.PumpAndSettle()
	if lastPopSuccess {
		t.Error("the pop should have been blocked")
	}
	shows("PopScope Page", "Home Page")
	disposition(oneContext, RoutePopDispositionDoNotPop)

	systemBack()
	ts.PumpAndSettle()
	if lastPopSuccess {
		t.Error("the system back should have been blocked")
	}
	shows("PopScope Page", "Home Page")
	disposition(oneContext, RoutePopDispositionDoNotPop)

	setState(func() { canPop = true })
	ts.Pump()
	navState(t, key).MaybePop()
	ts.PumpAndSettle()
	if !lastPopSuccess {
		t.Error("the pop should have succeeded")
	}
	shows("Home Page", "PopScope Page")
	disposition(homeContext, RoutePopDispositionBubble)
	next()

	systemBack()
	ts.PumpAndSettle()
	if !lastPopSuccess {
		t.Error("the system back should have popped")
	}
	shows("Home Page", "PopScope Page")
	disposition(homeContext, RoutePopDispositionBubble)
}

func TestPopScopeRemovingItRemovesItsEffect(t *testing.T) {
	ts := gluttertest.New(t)
	usePopScope := true
	var setState StateSetter
	var context BuildContext
	ts.PumpWidget(popScopeApp(nil, map[string]WidgetBuilder{
		"/": func(BuildContext) Widget {
			return NewStatefulBuilder(func(bc BuildContext, set StateSetter) Widget {
				context, setState = bc, set
				child := popScopePage("Home/PopScope Page")
				if !usePopScope {
					return child
				}
				return NewPopScope(child, PopScopeOpts{CanPop: foundation.Some(false)})
			})
		},
	}, nil))
	if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionDoNotPop {
		t.Fatalf("PopDisposition = %v, want doNotPop", got)
	}
	setState(func() { usePopScope = false })
	ts.Pump()
	if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionBubble {
		t.Errorf("PopDisposition = %v, want bubble", got)
	}
}

func TestPopScopeIdenticalPopScopes(t *testing.T) {
	ts := gluttertest.New(t)
	use1, use2 := true, true
	var setState StateSetter
	var context BuildContext
	ts.PumpWidget(popScopeApp(nil, nil, NewStatefulBuilder(func(bc BuildContext, set StateSetter) Widget {
		context, setState = bc, set
		var children []Widget
		if use1 {
			children = append(children, NewPopScope(NewText("hello"), PopScopeOpts{CanPop: foundation.Some(false)}))
		}
		if use2 {
			children = append(children, NewPopScope(NewText("hello"), PopScopeOpts{CanPop: foundation.Some(false)}))
		}
		return NewColumn(children)
	})))
	check := func(want RoutePopDisposition) {
		t.Helper()
		if got := ModalRouteOf(context).PopDisposition(); got != want {
			t.Errorf("PopDisposition = %v, want %v", got, want)
		}
	}
	check(RoutePopDispositionDoNotPop)
	setState(func() { use1 = false })
	ts.Pump()
	check(RoutePopDispositionDoNotPop)
	setState(func() { use2 = false })
	ts.Pump()
	check(RoutePopDispositionBubble)
}

func TestPopScopeNestedNavigatorsKeepTheRouteBlocked(t *testing.T) {
	ts := gluttertest.New(t)
	root := NewGlobalKey[*NavigatorState]()
	nested := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(popScopeApp(root, map[string]WidgetBuilder{
		"/": func(BuildContext) Widget {
			return NewCenter(NewPopScope(NewNavigator(NavigatorOpts{
				Key: nested,
				OnGenerateRoute: func(RouteSettings) Route {
					return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
						return NewCenter(navText("Nested Page"))
					})
				},
			}), PopScopeOpts{CanPop: foundation.Some(false)}))
		},
	}, nil))
	ts.PumpAndSettle()
	if textsFound(ts, "Nested Page", false) != 1 {
		t.Fatal("the nested page should show")
	}
}
