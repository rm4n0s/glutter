package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// The focus cases of packages/flutter/test/widgets/navigator_test.dart. The
// MaterialApp of the Dart tests is navTestApp, and the arrow keys are
// FocusInDirection on the primary focus (Shortcuts come with step 10.6). Not
// ported: the accessibility focus cases (the platform channel of Android and
// iOS).

func TestNavigatorRequestsFocusOnlyIfRequestFocusIsTrue(t *testing.T) {
	for _, requestFocus := range []bool{true, false} {
		name := "requests focus if requestFocus is true"
		if !requestFocus {
			name = "does not request focus if requestFocus is false"
		}
		t.Run("Navigator "+name, func(t *testing.T) {
			ts := gluttertest.New(t)
			navigatorKey := NewGlobalKey[*NavigatorState]()
			innerKey := NewGlobalKey[State]()
			focusNode := NewFocusScopeNode()
			defer focusNode.Dispose()
			routes := map[string]Widget{
				"/":       navText("A"),
				"/second": NewSizedBox(navText("B"), SizedBoxOpts{Key: innerKey}),
			}
			// The navigator is nested in the one of the app, as in Dart's
			// MaterialApp(home: Navigator(...)): the scope of the app's route is
			// the enclosing scope of its focus node.
			ts.PumpWidget(NewColumn([]Widget{
				NewFocusScope(NewContainer(nil), FocusScopeOpts{Node: focusNode}),
				NewExpanded(navTestApp(navTestAppOpts{home: NewNavigator(NavigatorOpts{
					Key: navigatorKey,
					OnGenerateRoute: func(settings RouteSettings) Route {
						return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
							return routes[settings.Name()]
						}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
					},
					RequestFocus: foundation.Some(requestFocus),
				})})),
			}))
			navigator := navState(t, navigatorKey)
			if navigator.Widget().RequestFocus() != requestFocus {
				t.Fatalf("RequestFocus() = %v", navigator.Widget().RequestFocus())
			}
			if textsFound(ts, "A", false) != 1 || textsFound(ts, "B", true) != 0 {
				t.Fatal("page A should be shown")
			}
			check := func(step string, want bool) {
				t.Helper()
				if focusNode.HasFocus() != want {
					t.Errorf("%s: has focus = %v, want %v", step, focusNode.HasFocus(), want)
				}
			}
			// With requestFocus false the routes never take the focus.
			afterPush := !requestFocus
			check("start", false)
			focusNode.RequestFocus()
			ts.PumpAndSettle()
			check("requested", true)

			navigator.PushNamed("/second")
			ts.PumpAndSettle()
			if textsFound(ts, "A", true) != 1 || textsFound(ts, "B", false) != 1 {
				t.Fatal("page B should be shown over A")
			}
			check("pushed", afterPush)

			focusNode.RequestFocus()
			ts.PumpAndSettle()
			check("requested again", true)

			navigator.Pop()
			ts.PumpAndSettle()
			if textsFound(ts, "A", false) != 1 || textsFound(ts, "B", true) != 0 {
				t.Fatal("page A should be shown again")
			}
			// Pop does not take focus.
			check("popped", true)

			navigator.PushReplacementNamed("/second")
			ts.PumpAndSettle()
			if textsFound(ts, "A", true) != 0 || textsFound(ts, "B", false) != 1 {
				t.Fatal("page B should have replaced A")
			}
			check("replaced", afterPush)

			focusNode.RequestFocus()
			ts.PumpAndSettle()
			check("requested a third time", true)

			ModalRouteOf(innerKey.CurrentContext()).AddLocalHistoryEntry(NewLocalHistoryEntry())
			ts.PumpAndSettle()
			// addLocalHistoryEntry does not take focus.
			check("local history entry", true)
		})
	}
}

func TestNavigatorFocusTraversalPolicy(t *testing.T) {
	page := func(policy *FocusTraversalPolicy) RouteFactory {
		return func(settings RouteSettings) Route {
			return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
				*policy = FocusTraversalGroupOf(bc)
				return NewSizedBox(nil)
			}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		}
	}
	t.Run("Navigator doesn't override FocusTraversalPolicy of ancestors", func(t *testing.T) {
		ts := gluttertest.New(t)
		var policy FocusTraversalPolicy
		ts.PumpWidget(ltr(NewFocusTraversalGroup(NewNavigator(NavigatorOpts{OnGenerateRoute: page(&policy)}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		if _, ok := policy.(*WidgetOrderTraversalPolicy); !ok {
			t.Errorf("policy = %T, want *WidgetOrderTraversalPolicy", policy)
		}
	})
	t.Run("Navigator inserts ReadingOrderTraversalPolicy if no ancestor has a policy", func(t *testing.T) {
		ts := gluttertest.New(t)
		var policy FocusTraversalPolicy
		ts.PumpWidget(ltr(NewNavigator(NavigatorOpts{OnGenerateRoute: page(&policy)})))
		if _, ok := policy.(*ReadingOrderTraversalPolicy); !ok {
			t.Errorf("policy = %T, want *ReadingOrderTraversalPolicy", policy)
		}
	})
}

func TestDirectionalFocusTraversalBehaviorWithNestedNavigators(t *testing.T) {
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	tr := newTrace(4)
	nodes := make([]*FocusNode, 6)
	for i := range nodes {
		nodes[i] = NewFocusNode()
		defer nodes[i].Dispose()
	}
	makeFocus := func(i int) Widget {
		var cb func(bool)
		if i < 4 {
			cb = tr.callback(i)
		}
		return NewFocus(sizedBoxKey(nil), FocusOpts{FocusNode: nodes[i], OnFocusChange: cb})
	}
	// Layout is:
	// ---------App---------
	//          [0]
	// ---------Nested Navigator---------
	//          [1]
	//          [2]
	// ---------Nested Navigator End---------
	//          [3]
	// ---------App End---------
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewColumn([]Widget{
		makeFocus(0),
		NewNavigator(NavigatorOpts{Key: navigatorKey, OnGenerateRoute: func(settings RouteSettings) Route {
			return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
				return NewCenter(navText("home"))
			})
		}}),
		makeFocus(3),
	})}))
	pushWith := func(behavior TraversalEdgeBehavior) {
		navPush(navigatorKey, NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
			return NewColumn([]Widget{makeFocus(1), makeFocus(2)})
		}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{DirectionalTraversalEdgeBehavior: behavior}}))
		ts.PumpAndSettle()
	}
	resetTo := func(i int) {
		nodes[i].RequestFocus()
		ts.Pump()
		tr.clear()
	}
	for _, tc := range []struct {
		behavior TraversalEdgeBehavior
		down, up string
	}{
		{TraversalEdgeBehaviorStop, "nnnn", "nnnn"},
		{TraversalEdgeBehaviorClosedLoop, "nTFn", "nFTn"},
		{TraversalEdgeBehaviorParentScope, "nnFT", "TFnn"},
		{TraversalEdgeBehaviorLeaveFlutterView, "nnFn", "nFnn"},
	} {
		pushWith(tc.behavior)
		resetTo(2)
		moveFocus(ts, TraversalDirectionDown)
		tr.expect(t, tc.behavior.String()+" down", tc.down)
		tr.clear()
		resetTo(1)
		moveFocus(ts, TraversalDirectionUp)
		tr.expect(t, tc.behavior.String()+" up", tc.up)
		tr.clear()
		navState(t, navigatorKey).Pop()
		ts.PumpAndSettle()
	}
}
