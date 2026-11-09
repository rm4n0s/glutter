package cupertino_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/route_test.dart. A GestureDetector over a Text
// stands in for the CupertinoButton (step 34.4), and the navigation bar, the hero, the display feature,
// restoration and the paint-pattern cases are not ported (the golden of the transition has the shadow).

// button is a button: tapping its text calls onTap.
func button(label string, onTap func()) widgets.Widget {
	return widgets.NewGestureDetector(widgets.NewText(label), widgets.GestureDetectorOpts{OnTap: onTap})
}

// near reports whether a is within epsilon of b.
func near(a, b, epsilon float64) bool { return math.Abs(a-b) <= epsilon }

func requireNear(t *testing.T, what string, got, want, epsilon float64) {
	t.Helper()
	if !near(got, want, epsilon) {
		t.Errorf("%s = %.2f, want %.2f (±%g)", what, got, want, epsilon)
	}
}

// navigatorState is the state of the navigator that the app shows.
func navigatorState(ts *gluttertest.Tester) *widgets.NavigatorState {
	return gluttertest.StateOf[*widgets.NavigatorState](ts, gluttertest.ByType[*widgets.Navigator]())
}

// scaffoldOf is the page scaffold that has the text.
func scaffoldOf(text string) gluttertest.Finder {
	return gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())
}

// gestureObserver logs the user gestures and the pops of a navigator.
type gestureObserver struct {
	widgets.NavigatorObserverBase
	log []string
}

func (o *gestureObserver) DidPush(widgets.Route, widgets.Route) { o.log = append(o.log, "didPush") }
func (o *gestureObserver) DidPop(widgets.Route, widgets.Route)  { o.log = append(o.log, "didPop") }
func (o *gestureObserver) DidStartUserGesture(widgets.Route, widgets.Route) {
	o.log = append(o.log, "didStartUserGesture")
}
func (o *gestureObserver) DidStopUserGesture() { o.log = append(o.log, "didStopUserGesture") }
func (o *gestureObserver) removeLast() string {
	if len(o.log) == 0 {
		return ""
	}
	last := o.log[len(o.log)-1]
	o.log = o.log[:len(o.log)-1]
	return last
}

// scaffoldPage is a CupertinoPageScaffold with a centered text.
func scaffoldPage(label string) widgets.WidgetBuilder {
	return func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewText(label)))
	}
}

func TestCupertinoRouteBackSwipe(t *testing.T) {
	// pushApp is an app with a page whose button pushes a route; the scaffold's context pushes it too.
	pushApp := func(ts *gluttertest.Tester, button1, page1, page2 string) (push func()) {
		key := widgets.NewGlobalKey[widgets.State]()
		push = func() {
			bc := key.CurrentContext()
			widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(scaffoldPage(page2)))
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(
			widgets.NewCenter(widgets.NewColumn([]widgets.Widget{widgets.NewText(page1), button(button1, push)})),
			cupertino.CupertinoPageScaffoldOpts{Key: key})}))
		return push
	}

	t.Run("a back swipe dismiss that a route push interrupts", func(t *testing.T) {
		ts := gluttertest.New(t)
		push := pushApp(ts, "push", "page 1", "route")
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("route")) || ts.Exists(gluttertest.ByText("push")) {
			t.Fatal("the route should be pushed")
		}
		gesture := ts.StartGesture(ui.Offset{Dx: 5, Dy: 300})
		gesture.MoveBy(ui.Offset{Dx: 400})
		gesture.Up()
		ts.Pump()
		// The route has been dragged halfway across the screen, and the page is sliding in from the left.
		if got := ts.TopLeft(scaffoldOf("route")); got != (ui.Offset{Dx: 400}) {
			t.Errorf("the route is at %v, want (400, 0)", got)
		}
		if dx := ts.TopLeft(scaffoldOf("push")).Dx; dx >= 0 {
			t.Errorf("the page is at dx %v, want less than 0", dx)
		}
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("push")) || ts.Exists(gluttertest.ByText("route")) {
			t.Fatal("the page should be back")
		}
		if got := ts.TopLeft(scaffoldOf("push")); got != (ui.Offset{}) {
			t.Errorf("the page is at %v, want the origin", got)
		}

		// Run the dismiss animation 60%, then push a route with the navigator.
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		gesture = ts.StartGesture(ui.Offset{Dx: 5, Dy: 300})
		gesture.MoveBy(ui.Offset{Dx: 400})
		gesture.Up()
		ts.Pump()
		if got := ts.TopLeft(scaffoldOf("route")); got != (ui.Offset{Dx: 400}) {
			t.Errorf("the route is at %v, want (400, 0)", got)
		}
		ts.Pump(210 * time.Millisecond)
		requireNear(t, "the route", ts.TopLeft(scaffoldOf("route")).Dx, 789, 1)
		// The route that is animating away ignores input while the pop is underway.
		push()
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("route")) || ts.Exists(gluttertest.ByText("push")) {
			t.Error("the pushed route should be shown")
		}
		if navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be over")
		}
	})

	for _, tt := range []struct {
		name   string
		by     float64
		pop    bool
		wantDx float64
	}{
		{"less than halfway, a route pop interrupts", 100, true, 0},
		{"more than halfway, a route pop interrupts", 500, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pushApp(ts, "Push Page 2", "Page 1", "Page 2")
			ts.Tap(gluttertest.ByText("Push Page 2"))
			ts.PumpAndSettle()
			if ts.Exists(gluttertest.ByText("Page 1")) || !ts.Exists(gluttertest.ByText("Page 2")) {
				t.Fatal("page 2 should be shown")
			}
			gesture := ts.StartGesture(ui.Offset{Dx: 5, Dy: 300})
			gesture.MoveBy(ui.Offset{Dx: tt.by})
			ts.Pump()
			if got := ts.TopLeft(scaffoldOf("Page 2")); got != (ui.Offset{Dx: tt.by}) {
				t.Errorf("page 2 is at %v, want (%v, 0)", got, tt.by)
			}
			if dx := ts.TopLeft(scaffoldOf("Page 1")).Dx; dx >= 0 {
				t.Errorf("page 1 is at dx %v, want less than 0", dx)
			}
			// Pop with the navigator: page 2 goes as if there were no back gesture.
			navigatorState(ts).Pop()
			ts.PumpAndSettle()
			if got := ts.TopLeft(scaffoldOf("Page 1")); got != (ui.Offset{}) {
				t.Errorf("page 1 is at %v, want the origin", got)
			}
			if ts.Exists(gluttertest.ByText("Page 2")) {
				t.Error("page 2 should be gone")
			}
		})
	}
}

func TestCupertinoFullscreenRouteAnimatesTransformValues(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return button("Button", func() {
			widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewColumn([]widgets.Widget{widgets.NewPlaceholder(nil),
					button("Close", func() { widgets.NavigatorOf(bc).Pop() })})
			}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}}))
		})
	})}))
	step := func(ms int, want float64) {
		t.Helper()
		ts.Pump(time.Duration(ms) * time.Millisecond)
		requireNear(t, "the top of the placeholder", ts.TopLeft(gluttertest.ByType[*widgets.Placeholder]()).Dy, want, 0.1)
	}
	ts.Tap(gluttertest.ByText("Button"))
	ts.Pump()
	for _, s := range []struct {
		ms   int
		want float64
	}{{40, 475.6}, {40, 350.0}, {40, 237.4}, {40, 149.2}, {40, 89.5}, {40, 54.4}, {40, 33.2}, {40, 20.4}, {40, 12.6}, {40, 7.4}, {50, 3}, {50, 0}} {
		step(s.ms, s.want)
	}
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Millisecond})
	ts.Tap(gluttertest.ByText("Close"))
	ts.Pump()
	for _, s := range []struct {
		ms   int
		want float64
	}{{50, 156.3}, {50, 308.1}, {40, 411.03}, {40, 484.35}, {40, 530.67}, {40, 557.61}, {40, 573.88}, {40, 583.86}, {40, 590.26}, {40, 594.58}, {40, 597.66}, {40, 600.0}} {
		step(s.ms, s.want)
	}
}

func TestCupertinoPageRouteParallax(t *testing.T) {
	build := func(fromFullscreenDialog, pushFullscreen bool) (*gluttertest.Tester, func(label string)) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
			return cupertino.NewCupertinoPageRoute(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewColumn([]widgets.Widget{widgets.NewPlaceholder(nil), button("Button", func() {
					widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(bc widgets.BuildContext) widgets.Widget {
						return button("Close", func() { widgets.NavigatorOf(bc).Pop() })
					}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: pushFullscreen}}))
				})})
			}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}, FullscreenDialog: fromFullscreenDialog}})
		}}))
		return ts, func(string) {}
	}
	placeholderDx := func(ts *gluttertest.Tester) float64 {
		return ts.TopLeft(gluttertest.ByType[*widgets.Placeholder]()).Dx
	}
	for _, fromFullscreen := range []bool{false, true} {
		name := "from a page"
		if fromFullscreen {
			name = "from a fullscreen dialog"
		}
		t.Run("the page below moves when a page is pushed on top, "+name, func(t *testing.T) {
			ts, _ := build(fromFullscreen, false)
			ts.Tap(gluttertest.ByText("Button"))
			requireNear(t, "before the frame", placeholderDx(ts), 0, 0.1)
			ts.Pump()
			for _, s := range []struct {
				ms   int
				want float64
			}{{40, -55}, {40, -111}, {40, -161}, {40, -200}, {40, -226}, {40, -242}, {40, -251}, {40, -257}, {40, -261}, {40, -263}, {50, -265}, {50, -266}} {
				ts.Pump(time.Duration(s.ms) * time.Millisecond)
				requireNear(t, "the placeholder", placeholderDx(ts), s.want, 1)
			}
			ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Millisecond})
			ts.Tap(gluttertest.ByText("Close"))
			ts.Pump()
			for _, s := range []struct {
				ms   int
				want float64
			}{{50, -197}, {50, -129}, {40, -83}, {360, 0}} {
				ts.Pump(time.Duration(s.ms) * time.Millisecond)
				requireNear(t, "the placeholder", placeholderDx(ts), s.want, 1)
			}
		})
		t.Run("the page below stays when a fullscreen dialog is pushed on top, "+name, func(t *testing.T) {
			ts, _ := build(fromFullscreen, true)
			ts.Tap(gluttertest.ByText("Button"))
			ts.Pump()
			for range 10 {
				ts.Pump(40 * time.Millisecond)
				if dx := placeholderDx(ts); dx != 0 {
					t.Fatalf("the placeholder is at dx %v, want 0", dx)
				}
			}
			ts.Tap(gluttertest.ByText("Close"))
			ts.Pump()
			for _, ms := range []int{40, 360} {
				ts.Pump(time.Duration(ms) * time.Millisecond)
				if dx := placeholderDx(ts); dx != 0 {
					t.Fatalf("the placeholder is at dx %v, want 0", dx)
				}
			}
		})
	}
}

func TestCupertinoPushAndPopAreNotLinear(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewText("1")}))
	navigatorState(ts).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoPageScaffold(widgets.NewText("2"))
	}))
	dx := func(label string) float64 { return ts.TopLeft(gluttertest.ByText(label)).Dx }
	// The screen is 800 wide: text 1 goes from 0 to -266.67 and text 2 from 800 to 0, and the translation slows down.
	ts.Pump()
	for _, s := range []struct{ one, two float64 }{{-69, 609}, {-136, 362}, {-191, 192}} {
		ts.Pump(50 * time.Millisecond)
		requireNear(t, "text 1", dx("1"), s.one, 1)
		requireNear(t, "text 2", dx("2"), s.two, 1)
	}
	ts.Pump(350 * time.Millisecond)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Millisecond})
	navigatorState(ts).Pop()
	ts.Pump()
	for _, s := range []struct{ one, two float64 }{{-197, 190}, {-129, 437}, {-74, 607}} {
		ts.Pump(50 * time.Millisecond)
		requireNear(t, "text 1", dx("1"), s.one, 1)
		requireNear(t, "text 2", dx("2"), s.two, 1)
	}
}

func TestCupertinoDraggedPopGesture(t *testing.T) {
	pushTwo := func(ts *gluttertest.Tester, observers ...widgets.NavigatorObserver) {
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewText("1"), NavigatorObservers: observers}))
		navigatorState(ts).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoPageScaffold(widgets.NewText("2"))
		}))
		ts.PumpAndSettle()
	}
	dx := func(ts *gluttertest.Tester, label string) float64 { return ts.TopLeft(gluttertest.ByText(label)).Dx }

	t.Run("the dragged pop gesture is linear", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushTwo(ts)
		if ts.Exists(gluttertest.ByText("1")) || dx(ts, "2") != 0 {
			t.Fatal("page 2 should cover page 1")
		}
		swipe := ts.StartGesture(ui.Offset{Dx: 5, Dy: 100})
		swipe.MoveBy(ui.Offset{Dx: 100})
		ts.Pump()
		requireNear(t, "text 1", dx(ts, "1"), -233, 1)
		requireNear(t, "text 2", dx(ts, "2"), 100, 0.01)
		if !navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be in progress")
		}
		swipe.MoveBy(ui.Offset{Dx: 100})
		ts.Pump()
		requireNear(t, "text 1", dx(ts, "1"), -200, 0.01)
		requireNear(t, "text 2", dx(ts, "2"), 200, 0.01)
		// The same distance each time moves both routes the same distance.
		swipe.MoveBy(ui.Offset{Dx: 100})
		ts.Pump()
		requireNear(t, "text 1", dx(ts, "1"), -166, 1)
		requireNear(t, "text 2", dx(ts, "2"), 300, 0.01)
	})

	t.Run("the pop gesture snapping is not linear", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushTwo(ts)
		swipe := ts.StartGesture(ui.Offset{Dx: 5, Dy: 100})
		swipe.MoveBy(ui.Offset{Dx: 500})
		swipe.Up()
		ts.Pump()
		requireNear(t, "text 1", dx(ts, "1"), -100, 0.01)
		requireNear(t, "text 2", dx(ts, "2"), 500, 0.01)
		if !navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be in progress")
		}
		ts.Pump(50 * time.Millisecond)
		requireNear(t, "text 1", dx(ts, "1"), -61, 1)
		requireNear(t, "text 2", dx(ts, "2"), 614, 1)
		ts.Pump(50 * time.Millisecond)
		// The rate of change is slowing down.
		requireNear(t, "text 1", dx(ts, "1"), -26, 1)
		requireNear(t, "text 2", dx(ts, "2"), 721, 1)
		ts.PumpAndSettle()
		if navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be over")
		}
	})

	t.Run("snapped drags forwards and backwards signal didStart and didStopUserGesture", func(t *testing.T) {
		ts := gluttertest.New(t)
		observer := &gestureObserver{}
		pushTwo(ts, observer)
		if got := observer.removeLast(); got != "didPush" {
			t.Fatalf("last invocation = %q, want didPush", got)
		}
		ts.DragFrom(ui.Offset{Dx: 5, Dy: 100}, ui.Offset{Dx: 100})
		if got := observer.removeLast(); got != "didStartUserGesture" {
			t.Fatalf("last invocation = %q, want didStartUserGesture", got)
		}
		ts.Pump()
		requireNear(t, "text 2", dx(ts, "2"), 100, 0.01)
		if !navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be in progress")
		}
		// Not far enough to snap into dismissing the route: it snaps back.
		ts.Pump(351 * time.Millisecond)
		requireNear(t, "text 2", dx(ts, "2"), 0, 0.01)
		if navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be over")
		}
		if got := observer.removeLast(); got != "didStopUserGesture" {
			t.Fatalf("last invocation = %q, want didStopUserGesture", got)
		}
		if got := observer.removeLast(); got == "didPop" {
			t.Fatal("the route was popped")
		}
		ts.DragFrom(ui.Offset{Dx: 5, Dy: 100}, ui.Offset{Dx: 500})
		ts.Pump()
		requireNear(t, "text 2", dx(ts, "2"), 500, 0.01)
		if !navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be in progress")
		}
		if got := observer.removeLast(); got != "didPop" {
			t.Fatalf("last invocation = %q, want didPop", got)
		}
		// Far enough to snap out of the route.
		ts.Pump(351 * time.Millisecond)
		if ts.Exists(gluttertest.ByText("2")) {
			t.Error("page 2 should be gone")
		}
		requireNear(t, "text 1", dx(ts, "1"), 0, 0.01)
		if navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be over")
		}
	})
}

func TestCupertinoEdgeSwipeThenDrop(t *testing.T) {
	setup := func(t *testing.T) (*gluttertest.Tester, *gestureObserver) {
		ts := gluttertest.New(t)
		observer := &gestureObserver{}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
			NavigatorObservers: []widgets.NavigatorObserver{observer},
			OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
				return cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
					number := "2"
					if settings.Name() == "/" {
						number = "1"
					}
					return widgets.NewCenter(widgets.NewText("Page " + number))
				}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
			}}))
		navigatorState(ts).PushNamed("/next")
		ts.Pump()
		ts.Pump(time.Second)
		if ts.Exists(gluttertest.ByText("Page 1")) || !ts.Exists(gluttertest.ByText("Page 2", gluttertest.FindOpts{})) {
			t.Fatal("page 2 should be shown alone")
		}
		return ts, observer
	}
	t.Run("at the ending point", func(t *testing.T) {
		ts, observer := setup(t)
		gesture := ts.StartGesture(ui.Offset{Dx: 5, Dy: 200})
		gesture.MoveBy(ui.Offset{Dx: 800}) // The width of the page.
		if got := observer.removeLast(); got != "didStartUserGesture" {
			t.Fatalf("last invocation = %q, want didStartUserGesture", got)
		}
		gesture.Up()
		ts.Pump()
		if !ts.Exists(gluttertest.ByText("Page 1")) || ts.Exists(gluttertest.ByText("Page 2")) {
			t.Error("page 1 should be shown alone")
		}
		if got := observer.removeLast(); got != "didStopUserGesture" {
			t.Errorf("last invocation = %q, want didStopUserGesture", got)
		}
		if got := observer.removeLast(); got != "didPop" {
			t.Errorf("last invocation = %q, want didPop", got)
		}
	})
	t.Run("at the starting point", func(t *testing.T) {
		ts, observer := setup(t)
		gesture := ts.StartGesture(ui.Offset{Dx: 5, Dy: 200})
		gesture.MoveBy(ui.Offset{Dx: 300})
		if got := observer.removeLast(); got != "didStartUserGesture" {
			t.Fatalf("last invocation = %q, want didStartUserGesture", got)
		}
		if !navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be in progress")
		}
		ts.Pump()
		gesture.MoveBy(ui.Offset{Dx: -300})
		gesture.Up()
		ts.Pump()
		if ts.Exists(gluttertest.ByText("Page 1")) || !ts.Exists(gluttertest.ByText("Page 2")) {
			t.Error("page 2 should be shown alone")
		}
		if got := observer.removeLast(); got != "didStopUserGesture" {
			t.Errorf("last invocation = %q, want didStopUserGesture", got)
		}
		if got := observer.removeLast(); got == "didPop" {
			t.Error("the route was popped")
		}
		if navigatorState(ts).UserGestureInProgress() {
			t.Error("the user gesture should be over")
		}
	})
}

func TestCupertinoPageTransitionBarrier(t *testing.T) {
	for _, tt := range []struct {
		name       string
		fullscreen bool
		want       foundation.Opt[ui.Color]
	}{
		{"a route that is not a fullscreen dialog has a barrier color", false, foundation.Some(ui.NewColor(0x18000000))},
		{"a fullscreen dialog has no barrier color", true, foundation.None[ui.Color]()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBoxExpand(nil)}))
			navigatorState(ts).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) },
				cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: tt.fullscreen}}))
			ts.PumpAndSettle()
			barriers := ts.Widgets(gluttertest.ByType[*widgets.ModalBarrier]())
			got := barriers[len(barriers)-1].(*widgets.ModalBarrier).Color()
			if got != tt.want {
				t.Errorf("barrier color = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCupertinoPageRouteDurations(t *testing.T) {
	route := cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
	defer route.Dispose()
	if route.TransitionDuration() != 500*time.Millisecond || route.ReverseTransitionDuration() != 500*time.Millisecond {
		t.Errorf("durations = %v and %v, want 500ms", route.TransitionDuration(), route.ReverseTransitionDuration())
	}
	if route.DelegatedTransition() == nil {
		t.Error("a page route has a delegated transition")
	}
	if route.BarrierLabel() != "" {
		t.Errorf("barrier label = %q", route.BarrierLabel())
	}
}

var _ = animation.Curves
