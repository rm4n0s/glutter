package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/pop_scope_test.dart.
// Each runs on Android, where the checks of SystemNavigator.setFrameworkHandlesBack
// apply, and on Linux, where they do not (Flutter's TargetPlatformVariant.all runs them on
// every platform; the others behave as Linux). The system back button is simulated as
// WidgetsApp does it: the binding tells its observers, and the navigator tries to pop.

// popScopeBack records the last SystemNavigator.setFrameworkHandlesBack call
// (lastFrameworkHandlesBack in the Dart test).
type popScopeBack struct {
	platform foundation.TargetPlatform
	last     foundation.Opt[bool]
}

// expect checks the last call, on Android only, as the Dart test does.
func (b *popScopeBack) expect(t *testing.T, want bool) {
	t.Helper()
	if b.platform != foundation.TargetPlatformAndroid {
		return
	}
	if foundation.IsWeb {
		// system_navigator.dart: the call is for Android and not for the web.
		if _, ok := b.last.Get(); ok {
			t.Errorf("setFrameworkHandlesBack was called on the web")
		}
		return
	}
	if got, ok := b.last.Get(); !ok || got != want {
		t.Errorf("lastFrameworkHandlesBack = %v (set: %v), want %v", got, ok, want)
	}
}

// popScopePlatforms runs the test for each platform with a Tester whose app is resumed
// and whose navigator hook records the calls.
func popScopePlatforms(t *testing.T, test func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack)) {
	t.Helper()
	skipUnlessDebug(t)
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux} {
		t.Run(platform.Name(), func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			ts := gluttertest.New(t)
			back := &popScopeBack{platform: platform}
			ts.Binding.SystemNavigator().OnSetFrameworkHandlesBack = func(v bool) { back.last = foundation.Some(v) }
			ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateResumed)
			test(t, ts, back)
		})
	}
}

func popScopeApp(key *GlobalKey[*NavigatorState], routes map[string]WidgetBuilder, home Widget) Widget {
	return navTestApp(navTestAppOpts{navigatorKey: key, routes: routes, home: home})
}

func popScopePage(text string) Widget { return NewCenter(NewText(text)) }

func TestPopScopeTogglingCanPopOnRootRouteAllowsOrPreventsBacks(t *testing.T) {
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
		back.expect(t, false)
		if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionBubble {
			t.Errorf("PopDisposition = %v, want bubble", got)
		}
	})
}

func TestPopScopeCanReceiveResult(t *testing.T) {
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
	})
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
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
			// The system back button: the binding tells WidgetsApp, which pops.
			ts.Binding.HandlePopRoute()
		}
		next := func() {
			ts.Tap(gluttertest.ByText("Next"))
			ts.PumpAndSettle()
			shows("PopScope Page", "Home Page")
			disposition(oneContext, RoutePopDispositionPop)
			back.expect(t, true)
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
		back.expect(t, false)
		next()

		systemBack()
		ts.PumpAndSettle()
		if !lastPopSuccess {
			t.Error("the system back should have popped")
		}
		shows("Home Page", "PopScope Page")
		disposition(homeContext, RoutePopDispositionBubble)
		back.expect(t, false)
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
		back.expect(t, false)
		next()

		systemBack()
		ts.PumpAndSettle()
		if !lastPopSuccess {
			t.Error("the system back should have popped")
		}
		shows("Home Page", "PopScope Page")
		disposition(homeContext, RoutePopDispositionBubble)
	})
}

func TestPopScopeRemovingItRemovesItsEffect(t *testing.T) {
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
		back.expect(t, true)
		if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionDoNotPop {
			t.Fatalf("PopDisposition = %v, want doNotPop", got)
		}
		setState(func() { usePopScope = false })
		ts.Pump()
		back.expect(t, false)
		if got := ModalRouteOf(context).PopDisposition(); got != RoutePopDispositionBubble {
			t.Errorf("PopDisposition = %v, want bubble", got)
		}
	})
}

func TestPopScopeIdenticalPopScopes(t *testing.T) {
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
		back.expect(t, true)
		check(RoutePopDispositionDoNotPop)
		setState(func() { use1 = false })
		ts.Pump()
		back.expect(t, true)
		check(RoutePopDispositionDoNotPop)
		setState(func() { use2 = false })
		ts.Pump()
		back.expect(t, false)
		check(RoutePopDispositionBubble)
	})
}

func TestPopScopeNestedNavigatorsKeepTheRouteBlocked(t *testing.T) {
	popScopePlatforms(t, func(t *testing.T, ts *gluttertest.Tester, back *popScopeBack) {
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
		back.expect(t, true)
	})
}
