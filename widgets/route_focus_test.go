package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The focus cases of packages/flutter/test/widgets/routes_test.dart. A Focus
// widget stands in for the TestTextField of the Dart tests (EditableText comes
// with step 11.4), and the Escape shortcut ones wait for step 10.6.

func routePage(builder func(bc BuildContext) Widget, opts ...PageRouteBuilderOpts) *PageRouteBuilder {
	var o PageRouteBuilderOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget { return builder(bc) }, o)
}

func TestRouteCanAutofocusAFocusInARoute(t *testing.T) {
	ts := gluttertest.New(t)
	focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node"})
	defer focusNode.Dispose()
	ts.PumpWidget(navTestApp(navTestAppOpts{
		initialRoute: "/",
		onGenerateRoute: func(settings RouteSettings) Route {
			return routePage(func(BuildContext) Widget {
				return NewFocus(NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: focusNode}))
			}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		},
	}))
	ts.Pump()
	if !focusNode.HasPrimaryFocus() {
		t.Error("the node should have the primary focus")
	}
}

func TestRouteFocusTraversalIsCorrectWhenPoppingMultiplePagesSimultaneously(t *testing.T) {
	for _, withFocusedChildren := range []bool{false, true} {
		name := "plain"
		if withFocusedChildren {
			name = "with focused children"
		}
		t.Run(name, func(t *testing.T) {
			// Regression test: https://github.com/flutter/flutter/issues/48903
			ts := gluttertest.New(t)
			key := NewGlobalKey[*NavigatorState]()
			ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("dummy1")}))
			scopeOf := func(text string) *FocusScopeNode {
				return FocusScopeOf(ctxOf(ts, gluttertest.ByText(text)))
			}
			pageOne := scopeOf("dummy1")
			if !pageOne.HasFocus() {
				t.Fatal("page one should have the focus")
			}

			var pageTwo *FocusScopeNode
			if withFocusedChildren {
				navPush(key, routePage(func(BuildContext) Widget { return NewFocus(navText("dummy2")) }))
			} else {
				navPush(key, routePage(func(BuildContext) Widget { return navText("dummy2") }))
			}
			ts.PumpAndSettle()
			pageTwo = scopeOf("dummy2")
			if pageOne.HasFocus() || !pageTwo.HasFocus() {
				t.Fatalf("the focus should be on page two: %v %v", pageOne.HasFocus(), pageTwo.HasFocus())
			}
			if withFocusedChildren {
				// Moves the focus to another node.
				pageTwo.NextFocus()
				ts.PumpAndSettle()
				if !pageTwo.HasFocus() || pageTwo.HasPrimaryFocus() {
					t.Fatal("the focus should be in a child of page two")
				}
			}

			navPush(key, routePage(func(BuildContext) Widget { return navText("dummy3") }))
			ts.PumpAndSettle()
			pageThree := scopeOf("dummy3")
			if pageOne.HasFocus() || pageTwo.HasFocus() || !pageThree.HasFocus() {
				t.Fatal("the focus should be on page three")
			}

			// Pops two pages simultaneously.
			navState(t, key).PopUntil(func(route Route) bool { return route.IsFirst() })
			ts.PumpAndSettle()
			if !pageOne.HasFocus() {
				t.Error("page one should have the focus again")
			}
		})
	}
}

func TestRouteRequestFocusCanBeUpdated(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	page2 := func(opts ...PageRouteBuilderOpts) {
		navPush(key, routePage(func(BuildContext) Widget { return navText("page2") }, opts...))
		ts.PumpAndSettle()
		if textsFound(ts, "page2", false) != 1 {
			t.Fatal("page2 should be shown")
		}
	}
	page2()
	if route := ModalRouteOf(ctxOf(ts, gluttertest.ByText("page2"))); !route.RequestFocus() {
		t.Error("the route should request focus")
	}
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if textsFound(ts, "page2", false) != 0 {
		t.Fatal("page2 should be gone")
	}
	page2(PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{RequestFocus: foundation.Some(false)}}})
	if route := ModalRouteOf(ctxOf(ts, gluttertest.ByText("page2"))); route.RequestFocus() {
		t.Error("the route should not request focus")
	}
}

func TestModalRouteDefaultDirectionalTraversalEdgeBehaviorIsTheNavigators(t *testing.T) {
	for _, behavior := range []TraversalEdgeBehavior{TraversalEdgeBehaviorClosedLoop, TraversalEdgeBehaviorLeaveFlutterView,
		TraversalEdgeBehaviorParentScope, TraversalEdgeBehaviorStop} {
		t.Run(behavior.String(), func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
				Key:                                   foundation.NewUniqueKey(),
				RouteDirectionalTraversalEdgeBehavior: behavior,
				OnGenerateRoute: func(settings RouteSettings) Route {
					return routePage(func(BuildContext) Widget { return NewCenter(navText("page")) },
						PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
				},
			})))
			ts.PumpAndSettle()
			scope := FocusScopeOf(ctxOf(ts, gluttertest.ByText("page")))
			if got := scope.DirectionalTraversalEdgeBehavior(); got != behavior {
				t.Errorf("directional traversal edge behavior = %v, want %v", got, behavior)
			}
		})
	}
}

func TestRouteRequestFocusWorksCorrectlyInShowGeneralDialog(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: NewFocus(navText("field"), FocusOpts{FocusNode: focusNode})}))
	focusNode.RequestFocus()
	ts.Pump()
	if !focusNode.HasFocus() {
		t.Fatal("the field should have the focus")
	}
	dialog := func(requestFocus bool) {
		ShowGeneralDialog(key.CurrentContext(),
			func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
				return navText("dialog")
			},
			ShowGeneralDialogOpts{RequestFocus: foundation.Some(requestFocus), TransitionDuration: foundation.Some(200 * time.Millisecond)})
		ts.PumpAndSettle()
	}
	dialog(true)
	if !FocusScopeOf(ctxOf(ts, gluttertest.ByText("dialog"))).HasFocus() || focusNode.HasFocus() {
		t.Fatal("the dialog should have the focus")
	}
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if !focusNode.HasFocus() {
		t.Fatal("the field should have the focus again")
	}
	dialog(false)
	if FocusScopeOf(ctxOf(ts, gluttertest.ByText("dialog"))).HasFocus() || !focusNode.HasFocus() {
		t.Error("the field should keep the focus")
	}
}
