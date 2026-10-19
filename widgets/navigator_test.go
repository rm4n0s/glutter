package widgets_test

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/navigator_test.dart:
// the cases of the imperative API. Not ported here: the cases of the Pages API
// ('Page api' group and the others that use Navigator.pages), which are step
// 10.3; the restorable* methods and the restoration cases (no state restoration);
// the cases of MaterialPageRoute, Scaffold, TabBarView and the other Material
// widgets (they use navTestApp and page routes of this package instead); the
// focus cases (navigator_focus_test.go); the Android predictive back cases (they need the
// platform channel of the back gesture) and the memory events case.

type navObservation struct {
	current, previous, operation string
}

func routeName(r Route) string {
	if r == nil {
		return ""
	}
	return r.Settings().Name()
}

func TestNavigatorCanNavigateToAndFromAStatefulWidget(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewGestureDetector(navText("X"), GestureDetectorOpts{OnTap: func() { NavigatorPushNamed(bc, "/second") }})
		},
		"/second": func(bc BuildContext) Widget {
			return NewGestureDetector(navText("Y"), GestureDetectorOpts{OnTap: func() { NavigatorPop(bc) }})
		},
	}))
	if !isOnstage(ts, "X") || textsFound(ts, "Y", true) != 0 {
		t.Fatal("the first route should be there alone")
	}
	ts.Tap(gluttertest.ByText("X"))
	ts.Pump()
	if !isOnstage(ts, "X") || !isOffstage(ts, "Y") {
		t.Error("the second route should be offstage at first")
	}
	for i := 0; i < 3; i++ {
		ts.Pump(10 * time.Millisecond)
		if !isOnstage(ts, "X") || !isOnstage(ts, "Y") {
			t.Fatalf("both routes should show at +%dms", 10*(i+1))
		}
	}
	ts.Pump(time.Second)
	if textsFound(ts, "X", false) != 0 || textsFound(ts, "X", true) != 1 || !isOnstage(ts, "Y") {
		t.Error("after the transition only the second route shows")
	}
	ts.Tap(gluttertest.ByText("Y"))
	if textsFound(ts, "X", false) != 0 || !isOnstage(ts, "Y") {
		t.Error("the tap alone changes nothing")
	}
	ts.Pump()
	ts.Pump()
	if !isOnstage(ts, "X") || !isOnstage(ts, "Y") {
		t.Error("both routes show while popping")
	}
	ts.Pump(10 * time.Millisecond)
	if !isOnstage(ts, "X") || !isOnstage(ts, "Y") {
		t.Error("both routes show while popping")
	}
	ts.Pump(time.Second)
	if !isOnstage(ts, "X") || textsFound(ts, "Y", true) != 0 {
		t.Error("the first route is back alone")
	}
}

func TestNavigatorOfFailsGracefullyWhenNotFoundInContext(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	targetKey := foundation.NewValueKey("foo")
	var caught any
	ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
		return NewGestureDetector(nil, GestureDetectorOpts{
			Key:      targetKey,
			Behavior: rendering.HitTestBehaviorOpaque,
			OnTap:    func() { caught = panicOf(func() { NavigatorOf(bc) }) },
		})
	}))
	ts.Tap(gluttertest.ByKey(targetKey))
	err, ok := caught.(error)
	if !ok {
		t.Fatalf("Navigator.of did not fail: %v", caught)
	}
	appErr := requireErrorTag(t, err, "NavigatorMissing")
	if !strings.HasPrefix(appErr.Message, "Navigator operation requested with a context") {
		t.Errorf("message = %q", appErr.Message)
	}
}

func TestNavigatorCanSetClipBehavior(t *testing.T) {
	ts := gluttertest.New(t)
	page := func(RouteSettings) Route {
		return defaultNavPageRoute(&RouteSettingsBase{}, func(BuildContext) Widget { return navText("page") })
	}
	for _, tc := range []struct {
		name string
		clip ui.Clip
		want ui.Clip
	}{
		{"default is hard edge", ui.ClipUnspecified, ui.ClipHardEdge},
		{"none", ui.ClipNone, ui.ClipNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{ClipBehavior: tc.clip, OnGenerateRoute: page})))
			if got := ts.Widget(gluttertest.ByType[*Overlay]()).(*Overlay).ClipBehavior(); got != tc.want {
				t.Errorf("clip behavior = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNavigatorRootNavigatorFindsRootNavigator(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	nested := NewNavigator(NavigatorOpts{
		OnGenerateRoute: func(settings RouteSettings) Route {
			if settings.Name() != "/" {
				return nil
			}
			return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
				return NewGestureDetector(navText("Next"), GestureDetectorOpts{OnTap: func() {
					NavigatorOf(bc).Push(defaultNavPageRoute(&RouteSettingsBase{}, func(bc BuildContext) Widget {
						return NewCenter(NewGestureDetector(navText("Inner page"), GestureDetectorOpts{OnTap: func() {
							NavigatorOf(bc, NavigatorOfOpts{RootNavigator: true}).Push(defaultNavPageRoute(&RouteSettingsBase{}, func(BuildContext) Widget {
								return navText("Dialog")
							}))
						}}))
					}))
				}})
			})
		},
	})
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(BuildContext) Widget {
			return NewColumn([]Widget{
				NewSizedBox(navText("Root page"), SizedBoxOpts{Height: foundation.Some(300.0)}),
				NewSizedBox(nested, SizedBoxOpts{Height: foundation.Some(300.0)}),
			})
		},
	}))
	ts.Tap(gluttertest.ByText("Next"))
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	// Both elements are on screen.
	if got := ts.TopLeft(gluttertest.ByText("Root page")).Dy; got != 0 {
		t.Errorf("root page top = %v", got)
	}
	if got := ts.TopLeft(gluttertest.ByText("Inner page")).Dy; got <= 300 {
		t.Errorf("inner page top = %v, want > 300", got)
	}
	ts.Tap(gluttertest.ByText("Inner page"))
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	// The dialog is pushed to the whole page and is at the top of the screen, not
	// inside the inner page.
	if got := ts.TopLeft(gluttertest.ByText("Dialog")).Dy; got != 0 {
		t.Errorf("dialog top = %v", got)
	}
}

func TestNavigatorGesturesBetweenPushAndBuildAreIgnored(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewRow([]Widget{
				NewGestureDetector(navText("left"), GestureDetectorOpts{OnTap: func() {
					log = append(log, "left")
					NavigatorPushNamed(bc, "/second")
				}}),
				NewGestureDetector(navText("right"), GestureDetectorOpts{OnTap: func() { log = append(log, "right") }}),
			})
		},
		"/second": func(BuildContext) Widget { return NewSizedBox(nil) },
	}))
	ts.Tap(gluttertest.ByText("left"))
	ts.Tap(gluttertest.ByText("right"))
	if !slices.Equal(log, []string{"left"}) {
		t.Errorf("log = %v", log)
	}
}

func TestNavigatorPushNamedAcceptsAnyResultType(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":       func(BuildContext) Widget { return navText("/") },
		"/second": func(BuildContext) Widget { return navText("/second") },
	}, navTestAppOpts{navigatorKey: key}))
	if !isOnstage(ts, "/") {
		t.Fatal("the home route is missing")
	}
	navState(t, key).PushNamed("/second")
	ts.PumpAndSettle()
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "/second") {
		t.Error("the second route should replace the first on screen")
	}
}

func TestNavigatorPendingGesturesAreRejected(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewRow([]Widget{
				NewGestureDetector(navText("left"), GestureDetectorOpts{OnTap: func() {
					log = append(log, "left")
					NavigatorPushNamed(bc, "/second")
				}}),
				NewGestureDetector(navText("right"), GestureDetectorOpts{OnTap: func() { log = append(log, "right") }}),
			})
		},
		"/second": func(BuildContext) Widget { return NewSizedBox(nil) },
	}))
	gesture := ts.StartGesture(ts.Center(gluttertest.ByText("right")), gluttertest.GestureOpts{Pointer: 23})
	if len(log) != 0 {
		t.Fatalf("log = %v", log)
	}
	ts.Tap(gluttertest.ByText("left"), gluttertest.GestureOpts{Pointer: 1})
	if !slices.Equal(log, []string{"left"}) {
		t.Fatalf("log = %v", log)
	}
	gesture.Up()
	if !slices.Equal(log, []string{"left"}) {
		t.Errorf("the pending gesture was not rejected: %v", log)
	}
}

func TestNavigatorPopAndPushNamed(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPopAndPushNamed(bc, "/B") }) },
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", func() { NavigatorPop(bc) }) },
	}))
	if !isOnstage(ts, "/") || textsFound(ts, "A", true) != 0 || textsFound(ts, "B", true) != 0 {
		t.Fatal("only the home route should exist")
	}
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Error("A should show")
	}
	ts.Tap(gluttertest.ByText("A"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || textsFound(ts, "A", false) != 0 || !isOnstage(ts, "B") {
		t.Error("B should show")
	}
}

func TestNavigatorPushAndPopTriggerTheObservers(t *testing.T) {
	ts := gluttertest.New(t)
	isPushed, isPopped := false, false
	observer := &navTestObserver{}
	observer.onPushed = func(route, previous Route) {
		// Pushes the initial route.
		if _, ok := route.(PageRoute); !ok || route.Settings().Name() != "/" {
			t.Errorf("pushed %v", route)
		}
		if previous != nil {
			t.Errorf("previous = %v", previous)
		}
		isPushed = true
	}
	observer.onPopped = func(route, previous Route) { isPopped = true }
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPop(bc) }) },
	}, navTestAppOpts{observers: []NavigatorObserver{observer}}))
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || !isPushed || isPopped {
		t.Fatal("the initial route should be pushed")
	}

	isPushed, isPopped = false, false
	observer.onPushed = func(route, previous Route) {
		if route.Settings().Name() != "/A" || previous == nil || previous.Settings().Name() != "/" {
			t.Errorf("pushed %v after %v", routeName(route), routeName(previous))
		}
		isPushed = true
	}
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || !isPushed || isPopped {
		t.Fatal("A should be pushed")
	}

	isPushed, isPopped = false, false
	observer.onPopped = func(route, previous Route) {
		if route.Settings().Name() != "/A" || previous == nil || previous.Settings().Name() != "/" {
			t.Errorf("popped %v to %v", routeName(route), routeName(previous))
		}
		isPopped = true
	}
	ts.Tap(gluttertest.ByText("A"))
	navPumpSecond(ts)
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || isPushed || !isPopped {
		t.Error("A should be popped")
	}
}

func TestNavigatorAddAndRemoveAnObserver(t *testing.T) {
	ts := gluttertest.New(t)
	isPushed, isPopped := false, false
	observer1 := &navTestObserver{}
	observer2 := &navTestObserver{
		onPushed: func(route, previous Route) { isPushed = true },
		onPopped: func(route, previous Route) { isPopped = true },
	}
	routes := map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPop(bc) }) },
	}
	ts.PumpWidget(navRoutesApp(routes, navTestAppOpts{observers: []NavigatorObserver{observer1}}))
	if isPushed || isPopped {
		t.Fatal("observer 2 is not there yet")
	}
	ts.PumpWidget(navRoutesApp(routes, navTestAppOpts{observers: []NavigatorObserver{observer1, observer2}}))
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if !isPushed || isPopped {
		t.Fatal("observer 2 should see the push")
	}
	isPushed, isPopped = false, false
	ts.PumpWidget(navRoutesApp(routes, navTestAppOpts{observers: []NavigatorObserver{observer1}}))
	ts.Tap(gluttertest.ByText("A"))
	navPumpSecond(ts)
	if isPushed || isPopped {
		t.Error("observer 2 was removed and must not see the pop")
	}
}

func TestNavigatorInitialRouteTriggersObserverInTheRightOrder(t *testing.T) {
	ts := gluttertest.New(t)
	var observations []navObservation
	observer := &navTestObserver{onPushed: func(route, previous Route) {
		observations = append(observations, navObservation{routeName(route), routeName(previous), "push"})
	}}
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":    func(BuildContext) Widget { return navText("/") },
		"/A":   func(BuildContext) Widget { return navText("A") },
		"/A/B": func(BuildContext) Widget { return navText("B") },
	}, navTestAppOpts{initialRoute: "/A/B", observers: []NavigatorObserver{observer}}))
	want := []navObservation{{"/", "", "push"}, {"/A", "/", "push"}, {"/A/B", "/A", "push"}}
	if !slices.Equal(observations, want) {
		t.Errorf("observations = %v, want %v", observations, want)
	}
}

func TestNavigatorReplaceNamedReplaces(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() { NavigatorPushReplacementNamed(bc, "/A") })
		},
		"/A": func(bc BuildContext) Widget {
			return navOnTapPage("A", func() { NavigatorPushReplacementNamed(bc, "/B") })
		},
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", nil) },
	}))
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") {
		t.Error("A should replace the home route")
	}
	ts.Tap(gluttertest.ByText("A"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || textsFound(ts, "A", false) != 0 || !isOnstage(ts, "B") {
		t.Error("B should replace A")
	}
}

func TestNavigatorPushReplacementSetsSecondaryAnimationAfterTransition(t *testing.T) {
	ts := gluttertest.New(t)
	routes := map[string]*PageRouteBuilder{}
	builders := map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPushNamed(bc, "/B") }) },
		"/B": func(bc BuildContext) Widget {
			return navOnTapPage("B", func() { NavigatorPushReplacementNamed(bc, "/C") })
		},
		"/C": func(bc BuildContext) Widget {
			return navOnTapPage("C", func() { NavigatorRemoveRoute(bc, routes["/"]) })
		},
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route {
		ret := navSlideRoute(builders[settings.Name()], settings)
		routes[settings.Name()] = ret
		return ret
	}}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("/"))
	ts.PumpAndSettle()
	a2 := routes["/A"].SecondaryAnimation().Value()
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	if routes["/A"].SecondaryAnimation().Value() <= a2 {
		t.Error("the secondary animation of /A should have advanced")
	}
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("B"))
	ts.Pump()
	ts.Pump(200 * time.Millisecond)
	if v := routes["/A"].SecondaryAnimation().Value(); v != 1 {
		t.Errorf("secondary animation of /A = %v, want 1", v)
	}
	ts.Tap(gluttertest.ByText("C"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "C") {
		t.Error("C should be on stage")
	}
	if a, c := routes["/A"].SecondaryAnimation().Value(), routes["/C"].Animation().Value(); a != c {
		t.Errorf("secondary animation of /A = %v, animation of /C = %v", a, c)
	}
	controller := routes["/C"].Controller()
	controller.SetValue(1 - controller.Value())
	if a, c := routes["/A"].SecondaryAnimation().Value(), routes["/C"].Animation().Value(); a != c {
		t.Errorf("after the change: secondary animation of /A = %v, animation of /C = %v", a, c)
	}
}

func TestNavigatorNewRouteRemovedFromHistoryDuringPushReplacementTransition(t *testing.T) {
	ts := gluttertest.New(t)
	routes := map[string]*PageRouteBuilder{}
	builders := map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget {
			return navOnTapPage("A", func() { NavigatorPushReplacementNamed(bc, "/B") })
		},
		"/B": func(bc BuildContext) Widget {
			return navOnTapPage("B", func() { NavigatorRemoveRoute(bc, routes["/B"]) })
		},
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route {
		ret := navSlideRoute(builders[settings.Name()], settings)
		routes[settings.Name()] = ret
		return ret
	}}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("/"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(200 * time.Millisecond)
	if !isOnstage(ts, "A") || !isOnstage(ts, "B") {
		t.Fatal("A and B should show during the transition")
	}
	ts.Tap(gluttertest.ByText("B"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "/") || textsFound(ts, "B", false) != 0 || textsFound(ts, "A", false) != 0 {
		t.Error("only the home route should be left")
	}
	if v := routes["/"].SecondaryAnimation().Value(); v != 0 {
		t.Errorf("secondary animation = %v", v)
	}
	if v := routes["/"].Animation().Value(); v != 1 {
		t.Errorf("animation = %v", v)
	}
}

func TestNavigatorPushReplacementTriggersSecondaryAnimation(t *testing.T) {
	ts := gluttertest.New(t)
	builders := map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() { NavigatorPushReplacementNamed(bc, "/A") })
		},
		"/A": func(bc BuildContext) Widget {
			return navOnTapPage("A", func() { NavigatorPushReplacementNamed(bc, "/B") })
		},
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", nil) },
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route {
		return navSlideRoute(builders[settings.Name()], &RouteSettingsBase{})
	}}))
	ts.PumpAndSettle()
	rootOffsetOriginal := ts.TopLeft(gluttertest.ByText("/"))
	ts.Tap(gluttertest.ByText("/"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	if !isOnstage(ts, "/") || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Fatal("the home route and A should show")
	}
	if rootOffset := ts.TopLeft(gluttertest.ByText("/")); rootOffset.Dx >= rootOffsetOriginal.Dx {
		t.Errorf("root x = %v, original %v", rootOffset.Dx, rootOffsetOriginal.Dx)
	}
	aOffsetOriginal := ts.TopLeft(gluttertest.ByText("A"))
	ts.PumpAndSettle()
	aOffset := ts.TopLeft(gluttertest.ByText("A"))
	if aOffset.Dx >= aOffsetOriginal.Dx {
		t.Errorf("A x = %v, original %v", aOffset.Dx, aOffsetOriginal.Dx)
	}
	aOffsetOriginal = aOffset
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || !isOnstage(ts, "B") {
		t.Fatal("A and B should show")
	}
	if aOffset = ts.TopLeft(gluttertest.ByText("A")); aOffset.Dx >= aOffsetOriginal.Dx {
		t.Errorf("A x = %v, original %v", aOffset.Dx, aOffsetOriginal.Dx)
	}
}

func TestNavigatorPushReplacementReportsDidReplaceToTheObserver(t *testing.T) {
	ts := gluttertest.New(t)
	var observations []navObservation
	observer := &navTestObserver{
		onPopped: func(route, previous Route) {
			observations = append(observations, navObservation{routeName(route), routeName(previous), "didPop"})
		},
		onReplaced: func(route, previous Route) {
			observations = append(observations, navObservation{routeName(route), routeName(previous), "didReplace"})
		},
	}
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(BuildContext) Widget { return navOnTapPage("/", nil) },
		"/A": func(BuildContext) Widget { return navOnTapPage("A", nil) },
		"/A/B": func(bc BuildContext) Widget {
			return navOnTapPage("B", func() {
				NavigatorOf(bc).PopUntil(func(route Route) bool { return route.IsFirst() })
				NavigatorOf(bc).PushReplacementNamed("/C")
			})
		},
		"/C": func(BuildContext) Widget { return navOnTapPage("C", nil) },
	}, navTestAppOpts{observers: []NavigatorObserver{observer}, initialRoute: "/A/B"}))
	ts.PumpAndSettle()
	if !isOnstage(ts, "B") {
		t.Fatal("B should show")
	}
	ts.Tap(gluttertest.ByText("B"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	want := []navObservation{{"/A/B", "/A", "didPop"}, {"/A", "/", "didPop"}, {"/C", "/", "didReplace"}}
	if !slices.Equal(observations, want) {
		t.Fatalf("observations = %v, want %v", observations, want)
	}
	ts.PumpAndSettle()
	if !isOnstage(ts, "C") {
		t.Error("C should show")
	}
}

func TestNavigatorAbleToPopAllRoutes(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(BuildContext) Widget { return navOnTapPage("/", nil) },
		"/A": func(BuildContext) Widget { return navOnTapPage("A", nil) },
		"/A/B": func(bc BuildContext) Widget {
			return navOnTapPage("B", func() {
				// Pops all routes with a bad predicate.
				NavigatorOf(bc).PopUntil(func(Route) bool { return false })
			})
		},
	}, navTestAppOpts{initialRoute: "/A/B"}))
	ts.Tap(gluttertest.ByText("B"))
	ts.PumpAndSettle()
	if err := ts.TakeError(); err != nil {
		t.Errorf("popping all routes reported %v", err)
	}
}

func TestNavigatorPopUntilWithResultReturnsValueToTheLastPoppedRoute(t *testing.T) {
	ts := gluttertest.New(t)
	var first, second <-chan any
	ts.PumpWidget(navTestApp(navTestAppOpts{
		initialRoute: "/",
		onGenerateRoute: func(settings RouteSettings) Route {
			switch settings.Name() {
			case "/":
				return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
					return navOnTapPage("/", func() { first = NavigatorPushNamed(bc, "/A") })
				})
			case "/A":
				return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
					return navOnTapPage("A", func() { second = NavigatorPushNamed(bc, "/B") })
				})
			case "/B":
				return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
					return navOnTapPage("B", func() {
						NavigatorPopUntilWithResult(bc, func(route Route) bool { return route.IsFirst() }, true)
					})
				})
			}
			return nil
		},
	}))
	if !isOnstage(ts, "/") {
		t.Fatal("the home route should show")
	}
	ts.Tap(gluttertest.ByText("/"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("A"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "B") {
		t.Fatal("B should show")
	}
	ts.Tap(gluttertest.ByText("B"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "/") {
		t.Fatal("the home route should show again")
	}
	// Only the first return value is set.
	if v := recvResult(t, first); v != true {
		t.Errorf("first result = %v, want true", v)
	}
	if v := recvResult(t, second); v != nil {
		t.Errorf("second result = %v, want nil", v)
	}
}

func TestNavigatorPushAndRemoveUntilTriggersSecondaryAnimation(t *testing.T) {
	ts := gluttertest.New(t)
	builders := map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget {
			return navOnTapPage("A", func() {
				NavigatorPushNamedAndRemoveUntil(bc, "/B", func(Route) bool { return false })
			})
		},
		"/B": func(BuildContext) Widget { return navOnTapPage("B", nil) },
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route {
		return navSlideRoute(builders[settings.Name()], &RouteSettingsBase{})
	}}))
	ts.PumpAndSettle()
	rootOffsetOriginal := ts.TopLeft(gluttertest.ByText("/"))
	ts.Tap(gluttertest.ByText("/"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	if !isOnstage(ts, "/") || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Fatal("the home route and A should show")
	}
	if rootOffset := ts.TopLeft(gluttertest.ByText("/")); rootOffset.Dx >= rootOffsetOriginal.Dx {
		t.Errorf("root x = %v, original %v", rootOffset.Dx, rootOffsetOriginal.Dx)
	}
	aOffsetOriginal := ts.TopLeft(gluttertest.ByText("A"))
	ts.PumpAndSettle()
	aOffset := ts.TopLeft(gluttertest.ByText("A"))
	if aOffset.Dx >= aOffsetOriginal.Dx {
		t.Errorf("A x = %v, original %v", aOffset.Dx, aOffsetOriginal.Dx)
	}
	aOffsetOriginal = aOffset
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || !isOnstage(ts, "B") {
		t.Fatal("A and B should show")
	}
	if aOffset = ts.TopLeft(gluttertest.ByText("A")); aOffset.Dx >= aOffsetOriginal.Dx {
		t.Errorf("A x = %v, original %v", aOffset.Dx, aOffsetOriginal.Dx)
	}
	ts.PumpAndSettle()
	if textsFound(ts, "/", false) != 0 || textsFound(ts, "A", false) != 0 || !isOnstage(ts, "B") {
		t.Error("only B should be left")
	}
}

func TestNavigatorPushAndRemoveUntilDoesNotRemoveRoutesBelowTheFirstThatPassesThePredicate(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(BuildContext) Widget { return navText("home") },
		"/A": func(BuildContext) Widget { return navText("page A") },
		"/A/B": func(bc BuildContext) Widget {
			return navOnTapPage("B", func() {
				NavigatorOf(bc).PushNamedAndRemoveUntil("/D", ModalRouteWithName("/A"))
			})
		},
		"/D": func(BuildContext) Widget { return navText("page D") },
	}, navTestAppOpts{navigatorKey: key, initialRoute: "/A/B"}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("B"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "page D") {
		t.Fatal("page D should show")
	}
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if !isOnstage(ts, "page A") {
		t.Fatal("page A should show")
	}
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if !isOnstage(ts, "home") {
		t.Error("home should show")
	}
}

func TestNavigatorReplaceNamedReturnedValue(t *testing.T) {
	ts := gluttertest.New(t)
	var value <-chan any
	routes := map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget {
			return navOnTapPage("A", func() {
				value = NavigatorOf(bc).PushReplacementNamed("/B", NavigatorPushNamedOpts{Result: "B"})
			})
		},
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", func() { NavigatorPop(bc, "B") }) },
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: func(settings RouteSettings) Route {
		return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
			return routes[settings.Name()](bc)
		}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
	}}))
	if !isOnstage(ts, "/") || textsFound(ts, "A", true) != 0 || textsFound(ts, "B", true) != 0 {
		t.Fatal("only the home route should exist")
	}
	ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Fatal("A should show")
	}
	ts.Tap(gluttertest.ByText("A")) // replaceNamed("/B"), the stack becomes /, /B
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || textsFound(ts, "A", false) != 0 || !isOnstage(ts, "B") {
		t.Fatal("B should show")
	}
	ts.Tap(gluttertest.ByText("B")) // pop, the stack becomes /
	navPumpSecond(ts)
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || textsFound(ts, "B", false) != 0 {
		t.Fatal("the home route should show")
	}
	if v := recvResult(t, value); v != "B" {
		t.Errorf("result = %v, want B", v)
	}
}

// navMapRoutes builds the on-generate-route of the tests that need the routes by
// name: PageRouteBuilders that show the page of the builders.
func navMapRoutes(builders map[string]WidgetBuilder, routes map[string]*PageRouteBuilder) RouteFactory {
	return func(settings RouteSettings) Route {
		route := NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
			return builders[settings.Name()](bc)
		}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}}})
		routes[settings.Name()] = route
		return route
	}
}

func TestNavigatorRemoveRoute(t *testing.T) {
	ts := gluttertest.New(t)
	pageBuilders := map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPushNamed(bc, "/B") }) },
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", nil) },
	}
	routes := map[string]*PageRouteBuilder{}
	var removedRoute, previousRoute Route
	observer := &navTestObserver{onRemoved: func(route, previous Route) { removedRoute, previousRoute = route, previous }}
	ts.PumpWidget(navTestApp(navTestAppOpts{
		observers:       []NavigatorObserver{observer},
		onGenerateRoute: navMapRoutes(pageBuilders, routes),
	}))
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || textsFound(ts, "B", false) != 0 {
		t.Fatal("only the home route should show")
	}
	ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
	ts.PumpAndSettle()
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Fatal("A should show")
	}
	ts.Tap(gluttertest.ByText("A")) // pushNamed("/B"), the stack becomes /, /A, /B
	ts.PumpAndSettle()
	if textsFound(ts, "/", false) != 0 || textsFound(ts, "A", false) != 0 || !isOnstage(ts, "B") {
		t.Fatal("B should show")
	}
	// The stack of the navigator is ordered as expected.
	if !routes["/"].IsActive() || !routes["/A"].IsActive() || !routes["/B"].IsActive() || !routes["/"].IsFirst() || !routes["/B"].IsCurrent() {
		t.Fatal("the stack is not /, /A, /B")
	}

	navigator := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
	navigator.RemoveRoute(routes["/B"]) // the stack becomes /, /A
	ts.Pump()
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") || textsFound(ts, "B", false) != 0 {
		t.Error("A should show")
	}
	// The stack of the navigator does not include /B anymore.
	if !routes["/"].IsActive() || !routes["/A"].IsActive() || routes["/B"].IsActive() || !routes["/"].IsFirst() || !routes["/A"].IsCurrent() {
		t.Error("the stack is not /, /A")
	}
	if removedRoute != Route(routes["/B"]) || previousRoute != Route(routes["/A"]) {
		t.Errorf("removed %v after %v", routeName(removedRoute), routeName(previousRoute))
	}

	navigator.RemoveRoute(routes["/A"]) // the stack becomes just /
	ts.Pump()
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || textsFound(ts, "B", false) != 0 {
		t.Error("the home route should show")
	}
	if !routes["/"].IsActive() || routes["/A"].IsActive() || routes["/B"].IsActive() || !routes["/"].IsFirst() || !routes["/"].IsCurrent() {
		t.Error("the stack is not /")
	}
	if removedRoute != Route(routes["/A"]) || previousRoute != Route(routes["/"]) {
		t.Errorf("removed %v after %v", routeName(removedRoute), routeName(previousRoute))
	}
}

func TestNavigatorRemoveARouteWhoseValueIsAwaited(t *testing.T) {
	ts := gluttertest.New(t)
	var pageValue <-chan any
	routes := map[string]*PageRouteBuilder{}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: navMapRoutes(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() { pageValue = NavigatorPushNamed(bc, "/A") })
		},
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPop(bc, "A") }) },
	}, routes)}))
	ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
	ts.PumpAndSettle()
	navigator := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
	navigator.RemoveRoute(routes["/A"], "B") // the stack becomes /, and the value completes with "B"
	if v := recvResult(t, pageValue); v != "B" {
		t.Errorf("result = %v, want B", v)
	}
}

func TestNavigatorRemoveRouteBelowAnotherOneWhoseValueIsAwaited(t *testing.T) {
	ts := gluttertest.New(t)
	var pageValue <-chan any
	routes := map[string]*PageRouteBuilder{}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: navMapRoutes(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() { pageValue = NavigatorPushNamed(bc, "/A") })
		},
		"/A": func(bc BuildContext) Widget { return navOnTapPage("/A", func() { NavigatorPushNamed(bc, "/B") }) },
		"/B": func(bc BuildContext) Widget { return navOnTapPage("B", func() { NavigatorPop(bc, "B") }) },
	}, routes)}))
	ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("/A")) // pushNamed("/B"), the stack becomes /, /A, /B
	navigator := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
	navigator.RemoveRouteBelow(routes["/B"], "A") // the stack becomes /, /B, and the value completes with "A"
	if v := recvResult(t, pageValue); v != "A" {
		t.Errorf("result = %v, want A", v)
	}
}

func TestNavigatorReplaceARouteByAnotherWhoseValueIsAwaited(t *testing.T) {
	ts := gluttertest.New(t)
	var pageValue <-chan any
	routes := map[string]*PageRouteBuilder{}
	ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: navMapRoutes(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() { pageValue = NavigatorPushNamed(bc, "/A") })
		},
		"/A": func(bc BuildContext) Widget { return navOnTapPage("/A", nil) },
	}, routes)}))
	ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
	ts.PumpAndSettle()
	navigator := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
	routeB := defaultNavPageRoute(&RouteSettingsBase{}, func(BuildContext) Widget { return navOnTapPage("/B", nil) })
	navigator.Replace(routes["/A"], routeB) // the stack becomes /, /B, and the value completes with nil
	if v, ok := nowOrNothing(pageValue); !ok || v != nil {
		t.Errorf("result = %v (completed %v), want nil", v, ok)
	}
}

func TestNavigatorPushNamedRouteAndRemoveUntilWhereRouteValuesAreAwaited(t *testing.T) {
	for _, named := range []bool{true, false} {
		name := "route"
		if named {
			name = "named route"
		}
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var pageAValue, pageBValue <-chan any
			routes := map[string]*PageRouteBuilder{}
			ts.PumpWidget(navTestApp(navTestAppOpts{onGenerateRoute: navMapRoutes(map[string]WidgetBuilder{
				"/": func(bc BuildContext) Widget {
					return navOnTapPage("/", func() { pageAValue = NavigatorPushNamed(bc, "/A") })
				},
				"/A": func(bc BuildContext) Widget {
					return navOnTapPage("/A", func() { pageBValue = NavigatorPushNamed(bc, "/B") })
				},
				"/B": func(bc BuildContext) Widget { return navOnTapPage("/B", nil) },
				"/C": func(bc BuildContext) Widget { return navOnTapPage("/C", nil) },
			}, routes)}))
			ts.Tap(gluttertest.ByText("/")) // pushNamed("/A"), the stack becomes /, /A
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("/A")) // pushNamed("/B"), the stack becomes /, /A, /B
			navigator := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
			// The stack becomes /, /C, and the values of A and B complete with nil.
			if named {
				navigator.PushNamedAndRemoveUntil("/C", ModalRouteWithName("/"))
			} else {
				routeC := defaultNavPageRoute(&RouteSettingsBase{}, func(BuildContext) Widget { return navOnTapPage("/C", nil) })
				navigator.PushAndRemoveUntil(routeC, ModalRouteWithName("/"))
			}
			if v, ok := nowOrNothing(pageAValue); !ok || v != nil {
				t.Errorf("A result = %v (completed %v)", v, ok)
			}
			if v, ok := nowOrNothing(pageBValue); !ok || v != nil {
				t.Errorf("B result = %v (completed %v)", v, ok)
			}
		})
	}
}

func TestNavigatorReplacingRouteCanBeObserved(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var log []string
	name := func(r Route) string {
		if r == nil {
			return "<none>"
		}
		return r.Settings().Name()
	}
	observer := &navTestObserver{
		onPushed: func(route, previous Route) {
			log = append(log, fmt.Sprintf("pushed %s (previous is %s)", name(route), name(previous)))
		},
		onPopped: func(route, previous Route) {
			log = append(log, fmt.Sprintf("popped %s (previous is %s)", name(route), name(previous)))
		},
		onRemoved: func(route, previous Route) {
			log = append(log, fmt.Sprintf("removed %s (previous is %s)", name(route), name(previous)))
		},
		onReplaced: func(newRoute, oldRoute Route) {
			log = append(log, fmt.Sprintf("replaced %s with %s", name(oldRoute), name(newRoute)))
		},
	}
	var routeB Route
	page := func(name string, body WidgetBuilder) Route {
		return defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: name}), body)
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{
		navigatorKey: key,
		observers:    []NavigatorObserver{observer},
		home: navOnTapPage("A", func() {
			routeB = page("B", func(BuildContext) Widget {
				return navOnTapPage("B", func() {
					navState(t, key).Push(page("C", func(BuildContext) Widget {
						return navOnTapPage("C", func() {
							navState(t, key).Replace(routeB, page("D", func(BuildContext) Widget { return navText("D") }))
						})
					}))
				})
			})
			navState(t, key).Push(routeB)
		}),
	}))
	want := []string{"pushed / (previous is <none>)"}
	if !slices.Equal(log, want) {
		t.Fatalf("log = %v", log)
	}
	for _, step := range []struct{ tap, entry string }{
		{"A", "pushed B (previous is /)"},
		{"B", "pushed C (previous is B)"},
		{"C", "replaced B with D"},
	} {
		ts.Tap(gluttertest.ByText(step.tap))
		navPumpSecond(ts)
		want = append(want, step.entry)
		if !slices.Equal(log, want) {
			t.Fatalf("log after %s = %v, want %v", step.tap, log, want)
		}
	}
}

func TestNavigatorDidStartUserGestureIsObservable(t *testing.T) {
	ts := gluttertest.New(t)
	var observedRoute, observedPrevious Route
	observer := &navTestObserver{onStartUserGesture: func(route, previous Route) { observedRoute, observedPrevious = route, previous }}
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return navOnTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPop(bc) }) },
	}, navTestAppOpts{observers: []NavigatorObserver{observer}}))
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if textsFound(ts, "/", false) != 0 || !isOnstage(ts, "A") {
		t.Fatal("A should show")
	}
	gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).DidStartUserGesture()
	if routeName(observedRoute) != "/A" || routeName(observedPrevious) != "/" {
		t.Errorf("observed %v after %v", routeName(observedRoute), routeName(observedPrevious))
	}
}

func TestNavigatorModalRouteOfSetsUpARouteToRebuildIfItsStateChanges(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var log []string
	var routeB Route
	named := func(name string, body WidgetBuilder) Route {
		return defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: name}), body)
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{
		navigatorKey: key,
		home: navOnTapPage("A", func() {
			routeB = named("B", func(BuildContext) Widget {
				log = append(log, "building B")
				return navOnTapPage("B", func() {
					navState(t, key).Push(named("C", func(bc BuildContext) Widget {
						log = append(log, "building C", "found "+ModalRouteSettingsOf(bc).Name())
						return navOnTapPage("C", func() {
							navState(t, key).Replace(routeB, named("D", func(BuildContext) Widget {
								log = append(log, "building D")
								return navText("D")
							}))
						})
					}))
				})
			})
			navState(t, key).Push(routeB)
		}),
	}))
	check := func(want ...string) {
		t.Helper()
		if !slices.Equal(log, want) {
			t.Fatalf("log = %v, want %v", log, want)
		}
	}
	check()
	ts.Tap(gluttertest.ByText("A"))
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	check("building B")
	ts.Tap(gluttertest.ByText("B"))
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	check("building B", "building C", "found C")
	ts.Tap(gluttertest.ByText("C"))
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	check("building B", "building C", "found C", "building D")
	navState(t, key).Pop()
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	check("building B", "building C", "found C", "building D")
}

func TestNavigatorRoutesDoNotRebuildJustBecauseTheirAnimationsEnded(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var log []string
	canPopOf := func(bc BuildContext) string {
		return strconv.FormatBool(ModalRouteCanPopOf(bc).Value())
	}
	pageRoute := func(n int) Route {
		return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
			log = append(log, fmt.Sprintf("building page %d - %s", n, canPopOf(bc)))
			return NewPlaceholder(nil)
		})
	}
	nextRoute := pageRoute(1)
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, onGenerateRoute: func(RouteSettings) Route {
		result := nextRoute
		nextRoute = nil
		return result
	}}))
	expected := []string{"building page 1 - false"}
	check := func() {
		t.Helper()
		if !slices.Equal(log, expected) {
			t.Fatalf("log = %v, want %v", log, expected)
		}
	}
	check()
	navState(t, key).PushReplacement(pageRoute(2))
	check()
	ts.Pump()
	expected = append(expected, "building page 2 - false")
	check()
	ts.Pump(150 * time.Millisecond)
	check()
	navState(t, key).PushReplacement(pageRoute(3))
	check()
	ts.Pump()
	expected = append(expected, "building page 3 - false")
	check()
	ts.Pump(200 * time.Millisecond)
	check()
}

func TestNavigatorModalRouteCanBePartiallyDependedOn(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var log []string
	named := func(name string, build func(bc BuildContext) string) Route {
		return NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
			log = append(log, "building "+ModalRouteSettingsOf(bc).Name()+" - "+build(bc))
			return NewPlaceholder(nil)
		}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: NewRouteSettings(RouteSettingsOpts{Name: name})}}})
	}
	canPop := func(bc BuildContext) string { return "canPop: " + strconv.FormatBool(ModalRouteCanPopOf(bc).Value()) }
	isCurrent := func(bc BuildContext) string {
		return "isCurrent: " + strconv.FormatBool(ModalRouteIsCurrentOf(bc).Value())
	}
	nextRoute := named("page 1", canPop)
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, onGenerateRoute: func(RouteSettings) Route {
		result := nextRoute
		nextRoute = nil
		return result
	}}))
	expected := []string{"building page 1 - canPop: false"}
	check := func() {
		t.Helper()
		if !slices.Equal(log, expected) {
			t.Fatalf("log = %v, want %v", log, expected)
		}
	}
	check()
	navState(t, key).Push(named("page 2", isCurrent))
	check()
	ts.Pump()
	expected = append(expected, "building page 2 - isCurrent: true")
	check()
	navState(t, key).Push(named("page 3", canPop))
	check()
	ts.Pump()
	expected = append(expected, "building page 3 - canPop: true", "building page 2 - isCurrent: false")
	check()
	navState(t, key).Pop()
	ts.Pump()
	expected = append(expected, "building page 2 - isCurrent: true")
	check()
	navState(t, key).Pop()
	ts.Pump()
	check()
}

func TestNavigatorArgumentsForNamedRoutes(t *testing.T) {
	for _, viaState := range []bool{false, true} {
		name := "on Navigator"
		if viaState {
			name = "on NavigatorState"
		}
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := NewGlobalKey[*NavigatorState]()
			var currentRouteKey *GlobalKey[State]
			var arguments []any
			ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, onGenerateRoute: func(settings RouteSettings) Route {
				arguments = append(arguments, settings.Arguments())
				return defaultNavPageRoute(settings, func(BuildContext) Widget {
					currentRouteKey = NewGlobalKey[State]()
					return NewCenter(navText(settings.Name()), CenterOpts{Key: currentRouteKey})
				})
			}}))
			single := func(want any) {
				t.Helper()
				if len(arguments) != 1 || arguments[0] != want {
					t.Fatalf("arguments = %v, want [%v]", arguments, want)
				}
				arguments = nil
			}
			shows := func(visible string, hidden ...string) {
				t.Helper()
				if !isOnstage(ts, visible) {
					t.Errorf("%s should show", visible)
				}
				for _, h := range hidden {
					if textsFound(ts, h, false) != 0 {
						t.Errorf("%s should not show", h)
					}
				}
			}
			shows("/")
			single(nil)

			pushNamed := func(route string, args any) {
				if viaState {
					navState(t, key).PushNamed(route, NavigatorPushNamedOpts{Arguments: args})
				} else {
					NavigatorPushNamed(currentRouteKey.CurrentContext(), route, NavigatorPushNamedOpts{Arguments: args})
				}
			}
			pushNamed("/A", "pushNamed")
			ts.PumpAndSettle()
			shows("/A", "/")
			single("pushNamed")

			if viaState {
				navState(t, key).PopAndPushNamed("/B", NavigatorPushNamedOpts{Arguments: "popAndPushNamed"})
			} else {
				NavigatorPopAndPushNamed(currentRouteKey.CurrentContext(), "/B", NavigatorPushNamedOpts{Arguments: "popAndPushNamed"})
			}
			ts.PumpAndSettle()
			shows("/B", "/", "/A")
			single("popAndPushNamed")

			first := func(route Route) bool { return route.IsFirst() }
			if viaState {
				navState(t, key).PushNamedAndRemoveUntil("/C", first, NavigatorPushNamedOpts{Arguments: "pushNamedAndRemoveUntil"})
			} else {
				NavigatorPushNamedAndRemoveUntil(currentRouteKey.CurrentContext(), "/C", first, NavigatorPushNamedOpts{Arguments: "pushNamedAndRemoveUntil"})
			}
			ts.PumpAndSettle()
			shows("/C", "/", "/A", "/B")
			single("pushNamedAndRemoveUntil")

			if viaState {
				navState(t, key).PushReplacementNamed("/D", NavigatorPushNamedOpts{Arguments: "pushReplacementNamed"})
			} else {
				NavigatorPushReplacementNamed(currentRouteKey.CurrentContext(), "/D", NavigatorPushNamedOpts{Arguments: "pushReplacementNamed"})
			}
			ts.PumpAndSettle()
			shows("/D", "/", "/A", "/B", "/C")
			single("pushReplacementNamed")
		})
	}
}

func TestNavigatorInitialRouteCanHaveGaps(t *testing.T) {
	ts := gluttertest.New(t)
	keyNav := NewGlobalKey[*NavigatorState]()
	keyRoot, keyA, keyABC := foundation.NewValueKey("Root"), foundation.NewValueKey("A"), foundation.NewValueKey("ABC")
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":      func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyRoot}) },
		"/A":     func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyA}) },
		"/A/B/C": func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyABC}) },
		// The route /A/B is intentionally left out.
	}, navTestAppOpts{navigatorKey: keyNav, initialRoute: "/A/B/C"}))
	count := func(key foundation.Key, offstage bool) int {
		return ts.Count(gluttertest.ByKey(key, gluttertest.FindOpts{IncludeOffstage: offstage}))
	}
	// The initial route /A/B/C was pushed successfully.
	if count(keyRoot, true) != 1 || count(keyA, true) != 1 || count(keyABC, false) != 1 {
		t.Fatal("the initial routes are not all there")
	}
	navState(t, keyNav).Pop()
	ts.PumpAndSettle()
	if count(keyRoot, true) != 1 || count(keyA, false) != 1 || count(keyABC, true) != 0 {
		t.Error("after the pop only / and /A are left")
	}
}

func TestNavigatorTheFullInitialRouteHasToBeMatched(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	keyRoot, keyA, keyAB := foundation.NewValueKey("Root"), foundation.NewValueKey("A"), foundation.NewValueKey("AB")
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":    func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyRoot}) },
		"/A":   func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyA}) },
		"/A/B": func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: keyAB}) },
		// The route /A/B/C is intentionally left out.
	}, navTestAppOpts{initialRoute: "/A/B/C"}))
	err := takeFirst(t, ts)
	appErr := requireErrorTag(t, err, "NavigatorInitialRouteUnknown")
	if !strings.HasPrefix(appErr.Message, "could not navigate to initial route") {
		t.Errorf("message = %q", appErr.Message)
	}
	// Only the root route was pushed.
	if ts.Count(gluttertest.ByKey(keyRoot)) != 1 || ts.Count(gluttertest.ByKey(keyA)) != 0 || ts.Count(gluttertest.ByKey(keyAB)) != 0 {
		t.Error("only the root route should be there")
	}
}

func TestNavigatorPoppingImmediatelyAfterPushingDoesNotCrash(t *testing.T) {
	ts := gluttertest.New(t)
	isPushed, isPopped := false, false
	observer := &navTestObserver{
		onPushed: func(route, previous Route) {
			if _, ok := route.(PageRoute); !ok || route.Settings().Name() != "/" || previous != nil {
				t.Errorf("pushed %v after %v", routeName(route), routeName(previous))
			}
			isPushed = true
		},
		onPopped: func(route, previous Route) { isPopped = true },
	}
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return navOnTapPage("/", func() {
				NavigatorPushNamed(bc, "/A")
				NavigatorOf(bc).Pop()
			})
		},
		"/A": func(bc BuildContext) Widget { return navOnTapPage("A", func() { NavigatorPop(bc) }) },
	}, navTestAppOpts{observers: []NavigatorObserver{observer}}))
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || !isPushed || isPopped {
		t.Fatal("the home route should be pushed")
	}
	isPushed, isPopped = false, false
	observer.onPushed = func(route, previous Route) {
		if route.Settings().Name() != "/A" || previous == nil || previous.Settings().Name() != "/" {
			t.Errorf("pushed %v after %v", routeName(route), routeName(previous))
		}
		isPushed = true
	}
	ts.Tap(gluttertest.ByText("/"))
	navPumpSecond(ts)
	if !isOnstage(ts, "/") || textsFound(ts, "A", false) != 0 || !isPushed || !isPopped {
		t.Error("A should be pushed and popped")
	}
}

func TestNavigatorErrorControl(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("onGenerateRoute returns null", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewNavigator(NavigatorOpts{OnGenerateRoute: func(RouteSettings) Route { return nil }}))
		appErr := requireErrorTag(t, takeFirst(t, ts), "NavigatorOnGenerateRouteReturnedNull")
		if !strings.HasPrefix(appErr.Message, `Navigator.onGenerateRoute returned null when requested to build route "/"`) {
			t.Errorf("message = %q", appErr.Message)
		}
	})
	t.Run("onUnknownRoute null and onGenerateRoute returns null", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewNavigator(NavigatorOpts{
			OnGenerateRoute: func(RouteSettings) Route { return nil },
			OnUnknownRoute:  func(RouteSettings) Route { return nil },
		}))
		appErr := requireErrorTag(t, takeFirst(t, ts), "NavigatorOnUnknownRouteReturnedNull")
		if !strings.HasPrefix(appErr.Message, `Navigator.onUnknownRoute returned null when requested to build route "/"`) {
			t.Errorf("message = %q", appErr.Message)
		}
	})
	t.Run("onGenerateRoute missing", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewNavigator())
		_ = requireErrorTag(t, takeFirst(t, ts), "NavigatorOnGenerateRouteMissing")
	})
}

func TestNavigatorOpaqueOverlayEntries(t *testing.T) {
	t.Run("of the topmost initial route is marked as opaque", func(t *testing.T) {
		ts := gluttertest.New(t)
		root, intermediate, topmost := foundation.NewUniqueKey(), foundation.NewUniqueKey(), NewGlobalKey[State]()
		ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
			"/":    func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: root}) },
			"/A":   func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: intermediate}) },
			"/A/B": func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: topmost}) },
		}, navTestAppOpts{initialRoute: "/A/B"}))
		if !ModalRouteOf(topmost.CurrentContext()).OverlayEntries()[0].Opaque() {
			t.Error("the first overlay entry of the topmost route should be opaque")
		}
		if ts.Count(gluttertest.ByKey(root)) != 0 || ts.Count(gluttertest.ByKey(intermediate)) != 0 || ts.Count(gluttertest.ByKey(topmost)) != 1 {
			t.Error("the routes below should be hidden by the opaque route")
		}
	})
	t.Run("of the topmost route is set to opaque after push", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		keyOf := func(name string) foundation.Key { return foundation.NewValueKey(name) }
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, initialRoute: "/", onGenerateRoute: func(settings RouteSettings) Route {
			return navNoAnimationRoute(&RouteSettingsBase{}, func(BuildContext) Widget {
				return NewContainer(nil, ContainerOpts{Key: keyOf(settings.Name())})
			})
		}}))
		if ts.Count(gluttertest.ByKey(keyOf("/"))) != 1 {
			t.Fatal("the home route is missing")
		}
		navState(t, key).PushNamed("/A")
		ts.Pump()
		topMostContext := ts.Element(gluttertest.ByKey(keyOf("/A")))
		if !ModalRouteOf(topMostContext).OverlayEntries()[0].Opaque() {
			t.Error("the first overlay entry of /A should be opaque")
		}
		if ts.Count(gluttertest.ByKey(keyOf("/"))) != 0 || ts.Count(gluttertest.ByKey(keyOf("/A"))) != 1 {
			t.Error("/ should be hidden by /A")
		}
	})
	t.Run("of the topmost route is set to opaque after replace", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		keyOf := func(name string) foundation.Key { return foundation.NewValueKey(name) }
		count := func(name string, offstage bool) int {
			return ts.Count(gluttertest.ByKey(keyOf(name), gluttertest.FindOpts{IncludeOffstage: offstage}))
		}
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, initialRoute: "/A/B", onGenerateRoute: func(settings RouteSettings) Route {
			return navNoAnimationRoute(&RouteSettingsBase{}, func(BuildContext) Widget {
				return NewContainer(nil, ContainerOpts{Key: keyOf(settings.Name())})
			})
		}}))
		if count("/", false) != 0 || count("/A", false) != 0 || count("/A/B", false) != 1 {
			t.Fatal("only /A/B should show")
		}
		oldRoute := ModalRouteOf(ts.Element(gluttertest.ByKey(keyOf("/A"), gluttertest.FindOpts{IncludeOffstage: true})))
		newRoute := navNoAnimationRoute(&RouteSettingsBase{}, func(BuildContext) Widget {
			return NewContainer(nil, ContainerOpts{Key: keyOf("/C")})
		})
		navState(t, key).Replace(oldRoute, newRoute)
		ts.Pump()
		if !newRoute.OverlayEntries()[0].Opaque() {
			t.Error("the new route should be opaque")
		}
		if count("/", false) != 0 || count("/A", false) != 0 || count("/C", false) != 0 || count("/A/B", false) != 1 {
			t.Error("/A/B should hide the others")
		}
		navState(t, key).Pop()
		ts.PumpAndSettle()
		if count("/", false) != 0 || count("/A", false) != 0 || count("/A/B", false) != 0 || count("/C", false) != 1 {
			t.Error("/C should show")
		}
	})
}

func TestNavigatorPushingOpaqueRouteDoesNotRebuildRoutesBelow(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	bottomRoute, topRoute := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/":  func(BuildContext) Widget { return newStatefulTestWidget(bottomRoute) },
		"/a": func(BuildContext) Widget { return newStatefulTestWidget(topRoute) },
	}, navTestAppOpts{navigatorKey: key}))
	rebuilds := func(k foundation.Key, offstage bool) int {
		return gluttertest.StateOf[*statefulTestState](ts, gluttertest.ByKey(k, gluttertest.FindOpts{IncludeOffstage: offstage})).rebuildCount
	}
	if got := rebuilds(bottomRoute, false); got != 1 {
		t.Fatalf("bottom rebuilds = %d", got)
	}
	navState(t, key).PushNamed("/a")
	ts.PumpAndSettle()
	// The bottom route is offstage and did not rebuild.
	if ts.Count(gluttertest.ByKey(bottomRoute)) != 0 {
		t.Error("the bottom route should be offstage")
	}
	if got := rebuilds(bottomRoute, true); got != 1 {
		t.Errorf("bottom rebuilds = %d", got)
	}
	if got := rebuilds(topRoute, false); got != 1 {
		t.Errorf("top rebuilds = %d", got)
	}
}

func TestNavigatorInitialRoutesBelowOpaqueRouteAreOffstage(t *testing.T) {
	ts := gluttertest.New(t)
	testKey := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
		Key:          testKey,
		InitialRoute: "/a/b",
		OnGenerateRoute: func(s RouteSettings) Route {
			return defaultNavPageRoute(s, func(BuildContext) Widget { return navText("+" + s.Name() + "+") })
		},
	})))
	state := func(onstage, offstage []string) {
		t.Helper()
		for _, name := range onstage {
			if !isOnstage(ts, name) {
				t.Errorf("%s should be on stage", name)
			}
		}
		for _, name := range offstage {
			if textsFound(ts, name, false) != 0 || textsFound(ts, name, true) != 1 {
				t.Errorf("%s should be offstage", name)
			}
		}
	}
	state([]string{"+/a/b+"}, []string{"+/+", "+/a+"})
	navState(t, testKey).Pop()
	ts.PumpAndSettle()
	state([]string{"+/a+"}, []string{"+/+"})
	if textsFound(ts, "+/a/b+", true) != 0 {
		t.Error("/a/b should be gone")
	}
	navState(t, testKey).Pop()
	ts.PumpAndSettle()
	state([]string{"+/+"}, nil)
	if textsFound(ts, "+/a+", true) != 0 || textsFound(ts, "+/a/b+", true) != 0 {
		t.Error("the other routes should be gone")
	}
}

func TestNavigatorCanProvideCustomOnGenerateInitialRoutes(t *testing.T) {
	ts := gluttertest.New(t)
	called := false
	testKey := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
		Key:          testKey,
		InitialRoute: "Hello World",
		OnGenerateInitialRoutes: func(navigator *NavigatorState, initialRoute string) []Route {
			called = true
			var result []Route
			for _, route := range strings.Split(initialRoute, " ") {
				result = append(result, defaultNavPageRoute(&RouteSettingsBase{}, func(BuildContext) Widget { return navText(route) }))
			}
			return result
		},
	})))
	if !called || textsFound(ts, "Hello", false) != 0 || !isOnstage(ts, "World") {
		t.Fatal("World should show")
	}
	navState(t, testKey).Pop()
	ts.PumpAndSettle()
	if !isOnstage(ts, "Hello") || textsFound(ts, "World", false) != 0 {
		t.Error("Hello should show")
	}
}

func TestNavigatorOfAndMaybeOf(t *testing.T) {
	t.Run("able to handle input context is a navigator context", func(t *testing.T) {
		ts := gluttertest.New(t)
		testKey := NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: testKey, home: navText("home")}))
		state, _ := testKey.CurrentState()
		if got := NavigatorOf(testKey.CurrentContext()); got != state {
			t.Error("NavigatorOf did not find the navigator of the context")
		}
		if got := NavigatorMaybeOf(testKey.CurrentContext()); got == nil || got != state {
			t.Error("NavigatorMaybeOf did not find the navigator of the context")
		}
	})
	t.Run("root navigator", func(t *testing.T) {
		ts := gluttertest.New(t)
		root, sub := NewGlobalKey[*NavigatorState](), NewGlobalKey[*NavigatorState]()
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: root, home: NewNavigator(NavigatorOpts{
			Key: sub,
			OnGenerateRoute: func(settings RouteSettings) Route {
				return defaultNavPageRoute(settings, func(BuildContext) Widget { return navText("dummy") })
			},
		})}))
		rootState, _ := root.CurrentState()
		if got := NavigatorOf(sub.CurrentContext(), NavigatorOfOpts{RootNavigator: true}); got != rootState {
			t.Error("NavigatorOf with the root navigator found another one")
		}
		if got := NavigatorMaybeOf(sub.CurrentContext(), NavigatorOfOpts{RootNavigator: true}); got == nil || got != rootState {
			t.Error("NavigatorMaybeOf with the root navigator found another one")
		}
	})
	t.Run("there is no navigator", func(t *testing.T) {
		ts := gluttertest.New(t)
		testKey := NewGlobalKey[State]()
		ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget { return NewSizedBox(nil, SizedBoxOpts{Key: testKey}) }))
		if NavigatorMaybeOf(testKey.CurrentContext()) != nil {
			t.Error("NavigatorMaybeOf found a navigator")
		}
		if foundation.DebugMode {
			requireTag(t, "NavigatorMissing", func() { NavigatorOf(testKey.CurrentContext()) })
		}
	})
}

func TestNavigatorPushAndRemoveUntilAnimatesThePush(t *testing.T) {
	ts := gluttertest.New(t)
	navigator := NewGlobalKey[*NavigatorState]()
	observer := &transitionDurationObserver{}
	routes := map[string]ModalRoute{}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
		Key:          navigator,
		Observers:    []NavigatorObserver{observer},
		InitialRoute: "root",
		OnGenerateRoute: func(settings RouteSettings) Route {
			return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
				routes[settings.Name()] = ModalRouteOf(bc)
				return navText("Route: " + settings.Name())
			})
		},
	})))
	if !isOnstage(ts, "Route: root") {
		t.Fatal("root is missing")
	}
	for _, name := range []string{"1", "2", "3"} {
		navState(t, navigator).PushNamed(name)
		ts.PumpAndSettle()
		if !isOnstage(ts, "Route: "+name) {
			t.Fatalf("route %s is missing", name)
		}
	}
	for _, name := range []string{"2", "1", "root"} {
		if textsFound(ts, "Route: "+name, true) != 1 {
			t.Fatalf("route %s should be offstage", name)
		}
	}
	navState(t, navigator).PushNamedAndRemoveUntil("4", func(route Route) bool { return route.IsFirst() })
	ts.Pump()
	if !isOnstage(ts, "Route: 3") || !isOnstage(ts, "Route: 4") {
		t.Fatal("3 and 4 should show")
	}
	route4Entry := routes["4"].Animation()
	if route4Entry.Value() != 0 {
		t.Errorf("the entry animation started: %v", route4Entry.Value())
	}
	fourTenths := observer.duration.Value() * 2 / 5
	ts.Pump(fourTenths)
	if !isOnstage(ts, "Route: 3") || !isOnstage(ts, "Route: 4") {
		t.Error("3 and 4 should show")
	}
	if v := route4Entry.Value(); math.Abs(v-0.4) > 0.01 {
		t.Errorf("entry value = %v, want 0.4", v)
	}
	ts.Pump(fourTenths)
	if v := route4Entry.Value(); math.Abs(v-0.8) > 0.01 {
		t.Errorf("entry value = %v, want 0.8", v)
	}
	for _, name := range []string{"2", "1", "root"} {
		if textsFound(ts, "Route: "+name, true) != 1 {
			t.Errorf("route %s should be there", name)
		}
	}
	// When it hits 1.0 all but the root and the current route have been removed.
	ts.Pump(fourTenths)
	if textsFound(ts, "Route: 3", true) != 0 || !isOnstage(ts, "Route: 4") || route4Entry.Value() != 1 {
		t.Error("3 should be gone and 4 complete")
	}
	if textsFound(ts, "Route: 2", true) != 0 || textsFound(ts, "Route: 1", true) != 0 || textsFound(ts, "Route: root", true) != 1 {
		t.Error("only the root is left below")
	}
	navState(t, navigator).Pop()
	ts.PumpAndSettle()
	if !isOnstage(ts, "Route: root") || textsFound(ts, "Route: 4", true) != 0 {
		t.Error("the root should show")
	}
}

func TestNavigatorWrappingTickerModeCanTurnOffTickingInRoutes(t *testing.T) {
	ts := gluttertest.New(t)
	tickCount := 0
	widgetUnderTest := func(enabled bool) Widget {
		return NewTickerMode(enabled, NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			InitialRoute: "root",
			OnGenerateRoute: func(settings RouteSettings) Route {
				return defaultNavPageRoute(settings, func(BuildContext) Widget {
					return newNavTickingWidget(func() { tickCount++ })
				})
			},
		})))
	}
	ts.PumpWidget(widgetUnderTest(false))
	if tickCount != 0 {
		t.Fatal("ticked")
	}
	for i := 0; i < 4; i++ {
		ts.Pump(time.Second)
	}
	if tickCount != 0 {
		t.Fatalf("ticked %d times while disabled", tickCount)
	}
	ts.PumpWidget(widgetUnderTest(true))
	if tickCount != 0 {
		t.Fatal("ticked at once")
	}
	for i := 0; i < 4; i++ {
		ts.Pump(time.Second)
	}
	if tickCount != 4 {
		t.Errorf("ticked %d times, want 4", tickCount)
	}
}

func TestNavigatorRouteAnnouncesCorrectlyForFirstRouteAndLastRoute(t *testing.T) {
	ts := gluttertest.New(t)
	previousOfFirst, nextOfFirst, popNextOfFirst := Route(newNotAnnouncedRoute()), Route(newNotAnnouncedRoute()), Route(newNotAnnouncedRoute())
	previousOfSecond, nextOfSecond, popNextOfSecond := Route(newNotAnnouncedRoute()), Route(newNotAnnouncedRoute()), Route(newNotAnnouncedRoute())
	var firstRoute, secondRoute Route
	navigator := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigator, initialRoute: "/second", onGenerateRoute: func(settings RouteSettings) Route {
		if settings.Name() == "/" {
			firstRoute = newRouteAnnouncementSpy(settings,
				func(r Route) { previousOfFirst = r }, func(r Route) { nextOfFirst = r }, func(r Route) { popNextOfFirst = r })
			return firstRoute
		}
		secondRoute = newRouteAnnouncementSpy(settings,
			func(r Route) { previousOfSecond = r }, func(r Route) { nextOfSecond = r }, func(r Route) { popNextOfSecond = r })
		return secondRoute
	}}))
	ts.PumpAndSettle()
	if previousOfFirst != nil || nextOfFirst != secondRoute {
		t.Errorf("first: previous = %v, next = %v", previousOfFirst, nextOfFirst)
	}
	if _, ok := popNextOfFirst.(*notAnnouncedRoute); !ok {
		t.Errorf("first was told that its next route was popped: %v", popNextOfFirst)
	}
	if previousOfSecond != firstRoute || nextOfSecond != nil {
		t.Errorf("second: previous = %v, next = %v", previousOfSecond, nextOfSecond)
	}
	if _, ok := popNextOfSecond.(*notAnnouncedRoute); !ok {
		t.Errorf("second was told that its next route was popped: %v", popNextOfSecond)
	}
	navState(t, navigator).Pop()
	if popNextOfFirst != secondRoute {
		t.Errorf("pop next of the first = %v, want the second route", popNextOfFirst)
	}
}

func TestNavigatorHeroControllerScope(t *testing.T) {
	observations := func(into *[]navObservation) func(route, previous Route) {
		return func(route, previous Route) {
			*into = append(*into, navObservation{routeName(route), routeName(previous), "didPush"})
		}
	}
	generate := func(builder WidgetBuilder) RouteFactory {
		return func(s RouteSettings) Route { return defaultNavPageRoute(s, builder) }
	}
	t.Run("works", func(t *testing.T) {
		ts := gluttertest.New(t)
		top, sub := NewGlobalKey[*NavigatorState](), NewGlobalKey[*NavigatorState]()
		var obs []navObservation
		spy := newHeroControllerSpy()
		spy.onPushed = observations(&obs)
		defer spy.Dispose()
		ts.PumpWidget(NewHeroControllerScope(spy.HeroController, NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			Key:          top,
			InitialRoute: "top1",
			OnGenerateRoute: generate(func(BuildContext) Widget {
				return NewNavigator(NavigatorOpts{
					Key:             sub,
					InitialRoute:    "sub1",
					OnGenerateRoute: generate(func(BuildContext) Widget { return NewPlaceholder(nil) }),
				})
			}),
		}))))
		// It only observes the top navigator.
		if want := []navObservation{{"top1", "", "didPush"}}; !slices.Equal(obs, want) {
			t.Fatalf("observations = %v", obs)
		}
		navState(t, sub).Push(defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: "sub2"}), func(BuildContext) Widget { return navText("sub2") }))
		ts.PumpAndSettle()
		if !isOnstage(ts, "sub2") || len(obs) != 1 {
			t.Fatalf("the sub navigator was observed: %v", obs)
		}
		navState(t, top).Push(defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: "top2"}), func(BuildContext) Widget { return navText("top2") }))
		ts.PumpAndSettle()
		if want := []navObservation{{"top1", "", "didPush"}, {"top2", "top1", "didPush"}}; !slices.Equal(obs, want) {
			t.Errorf("observations = %v", obs)
		}
	})
	t.Run("can correctly transfer subscription - replacing navigator", func(t *testing.T) {
		ts := gluttertest.New(t)
		key1, key2 := NewGlobalKey[*NavigatorState](), NewGlobalKey[*NavigatorState]()
		var obs []navObservation
		spy := newHeroControllerSpy()
		spy.onPushed = observations(&obs)
		defer spy.Dispose()
		navigatorWith := func(key *GlobalKey[*NavigatorState], initial string) Widget {
			return NewHeroControllerScope(spy.HeroController, NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
				Key: key, InitialRoute: initial,
				OnGenerateRoute: generate(func(BuildContext) Widget { return NewPlaceholder(nil) }),
			})))
		}
		ts.PumpWidget(navigatorWith(key1, "navigator1"))
		// Transfers the subscription to another navigator.
		ts.PumpWidget(navigatorWith(key2, "navigator2"))
		obs = nil
		navState(t, key2).Push(defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: "new route"}), func(BuildContext) Widget { return navText("new route") }))
		ts.PumpAndSettle()
		if !isOnstage(ts, "new route") {
			t.Fatal("the new route is missing")
		}
		// It records from the new navigator.
		if want := []navObservation{{"new route", "navigator2", "didPush"}}; !slices.Equal(obs, want) {
			t.Errorf("observations = %v", obs)
		}
	})
	t.Run("can correctly transfer subscription - swapping navigator", func(t *testing.T) {
		ts := gluttertest.New(t)
		key1, key2 := NewGlobalKey[*NavigatorState](), NewGlobalKey[*NavigatorState]()
		var obs1, obs2 []navObservation
		spy1, spy2 := newHeroControllerSpy(), newHeroControllerSpy()
		spy1.onPushed, spy2.onPushed = observations(&obs1), observations(&obs2)
		defer spy1.Dispose()
		defer spy2.Dispose()
		app := func(first, second *heroControllerSpy) Widget {
			nav := func(spy *heroControllerSpy, key *GlobalKey[*NavigatorState], initial string) Widget {
				return NewHeroControllerScope(spy.HeroController, NewNavigator(NavigatorOpts{
					Key: key, InitialRoute: initial,
					OnGenerateRoute: generate(func(BuildContext) Widget { return NewPlaceholder(nil) }),
				}))
			}
			return NewDirectionality(ui.TextDirectionLTR, NewStack([]Widget{nav(first, key1, "navigator1"), nav(second, key2, "navigator2")}))
		}
		ts.PumpWidget(app(spy1, spy2))
		if want := []navObservation{{"navigator1", "", "didPush"}}; !slices.Equal(obs1, want) {
			t.Fatalf("observations 1 = %v", obs1)
		}
		if want := []navObservation{{"navigator2", "", "didPush"}}; !slices.Equal(obs2, want) {
			t.Fatalf("observations 2 = %v", obs2)
		}
		// Swaps the spies.
		ts.PumpWidget(app(spy2, spy1))
		navState(t, key2).Push(defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: "new route2"}), func(BuildContext) Widget { return navText("new route2") }))
		ts.PumpAndSettle()
		if !isOnstage(ts, "new route2") {
			t.Fatal("new route2 is missing")
		}
		// The first spy records the push in navigator 2; the second records nothing.
		if len(obs1) != 2 || obs1[1] != (navObservation{"new route2", "navigator2", "didPush"}) || len(obs2) != 1 {
			t.Fatalf("observations = %v, %v", obs1, obs2)
		}
		navState(t, key1).Push(defaultNavPageRoute(NewRouteSettings(RouteSettingsOpts{Name: "new route1"}), func(BuildContext) Widget { return navText("new route1") }))
		ts.PumpAndSettle()
		if !isOnstage(ts, "new route1") {
			t.Fatal("new route1 is missing")
		}
		if len(obs1) != 2 || len(obs2) != 2 || obs2[1] != (navObservation{"new route1", "navigator1", "didPush"}) {
			t.Errorf("observations = %v, %v", obs1, obs2)
		}
	})
	t.Run("subscribing to multiple navigators throws", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		spy := newHeroControllerSpy()
		defer spy.Dispose()
		nav := func(initial string) Widget {
			return NewNavigator(NavigatorOpts{InitialRoute: initial, OnGenerateRoute: generate(func(BuildContext) Widget { return NewPlaceholder(nil) })})
		}
		ts.PumpWidget(NewHeroControllerScope(spy.HeroController, NewDirectionality(ui.TextDirectionLTR,
			NewStack([]Widget{nav("navigator1"), nav("navigator2")}))))
		appErr := requireErrorTag(t, takeFirst(t, ts), "HeroControllerShared")
		if !strings.HasPrefix(appErr.Message, "a HeroController can not be shared by multiple Navigators") {
			t.Errorf("message = %q", appErr.Message)
		}
	})
}

func TestNavigatorCanReuseNavigatorObserverInRebuiltTree(t *testing.T) {
	ts := gluttertest.New(t)
	observer := &NavigatorObserverBase{}
	build := func(key foundation.Key) Widget {
		return NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			Key:       key,
			Observers: []NavigatorObserver{observer},
			OnGenerateRoute: func(settings RouteSettings) Route {
				return defaultNavPageRoute(settings, func(BuildContext) Widget { return NewContainer(nil) })
			},
		}))
	}
	// Without reinsertion.
	ts.PumpWidget(build(nil))
	ts.PumpWidget(NewContainer(build(nil)))
	if got := observer.Navigator(); got != gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()) {
		t.Error("the observer does not observe the navigator")
	}
	// Clears the tree.
	ts.PumpWidget(NewContainer(nil))
	if observer.Navigator() != nil {
		t.Error("the observer still observes a navigator")
	}
	// With reinsertion.
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(build(key))
	ts.PumpWidget(NewContainer(build(key)))
	if got := observer.Navigator(); got != gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()) {
		t.Error("the observer does not observe the navigator after the reinsertion")
	}
}

func TestRouteSettingsToString(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    *RouteSettingsBase
		want string
	}{
		{"when the name is not empty it has double quotes", NewRouteSettings(RouteSettingsOpts{Name: "/home"}), `RouteSettings("/home", null)`},
		{"when the name is empty it has no double quotes", NewRouteSettings(), "RouteSettings(none, null)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// navButton is a TextButton: a tappable label.
func navButton(label string, onTap func()) Widget {
	return NewGestureDetector(navText(label), GestureDetectorOpts{OnTap: onTap})
}

func TestNavigatorPopHandlerOnPopWithResult(t *testing.T) {
	ts := gluttertest.New(t)
	nav, nestedNav := NewGlobalKey[*NavigatorState](), NewGlobalKey[*NavigatorState]()
	const result = "i am a result"
	var results []any
	linksPage := func(title string, buttons ...Widget) Widget {
		return NewColumn(append([]Widget{navText(title)}, buttons...))
	}
	ts.PumpWidget(navRoutesApp(map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return linksPage("Home page",
				navButton("Go to one", func() { NavigatorOf(bc).PushNamed("/one") }),
				navButton("Go to nested", func() { NavigatorOf(bc).PushNamed("/nested") }))
		},
		"/one": func(bc BuildContext) Widget { return linksPage("Page one") },
		"/nested": func(rootContext BuildContext) Widget {
			return NewNavigatorPopHandler(
				NewNavigator(NavigatorOpts{
					Key:          nestedNav,
					InitialRoute: "/",
					OnGenerateRoute: func(settings RouteSettings) Route {
						switch settings.Name() {
						case "/":
							return defaultNavPageRoute(settings, func(bc BuildContext) Widget {
								return linksPage("Nested - home",
									navButton("Go to nested/one", func() { NavigatorOf(bc).PushNamed("/one") }),
									navButton("Go back out of nested nav", func() { NavigatorOf(rootContext).Pop() }))
							})
						case "/one":
							return defaultNavPageRoute(settings, func(BuildContext) Widget { return linksPage("Nested - page one") })
						}
						return nil
					},
				}),
				NavigatorPopHandlerOpts{OnPopWithResult: func(result any) {
					results = append(results, result)
					navState(t, nestedNav).Pop(result)
				}},
			)
		},
	}, navTestAppOpts{navigatorKey: nav, initialRoute: "/"}))
	if !isOnstage(ts, "Home page") {
		t.Fatal("the home page is missing")
	}
	ts.Tap(gluttertest.ByText("Go to nested"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "Nested - home") {
		t.Fatal("the nested home page is missing")
	}
	ts.Tap(gluttertest.ByText("Go to nested/one"))
	ts.PumpAndSettle()
	if !isOnstage(ts, "Nested - page one") || len(results) != 0 {
		t.Fatalf("the nested page one is missing or popped: %v", results)
	}
	// Pops the root navigator, despite being on a route in the nested navigator.
	// This triggers NavigatorPopHandler.OnPopWithResult with the given result.
	navState(t, nav).MaybePop(result)
	ts.PumpAndSettle()
	if !isOnstage(ts, "Nested - home") {
		t.Error("the nested home page should show again")
	}
	if len(results) != 1 || results[0] != result {
		t.Errorf("results = %v, want [%q]", results, result)
	}
}
