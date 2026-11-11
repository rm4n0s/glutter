package cupertino_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoSliverRefreshControl, ported from cupertino/refresh_test.dart (step 34.8). The Flutter file runs
// every test on iOS and macOS; the two differ only in the numbers of the scroll physics (the macOS ones come from
// the simulation of the mouse wheel and the trackpad), so the tests here run on iOS with its numbers, which come out the
// same as in Flutter and so prove the port of the sliver, the state machine and the physics.

// refreshInvocation is a call that the fake builder saw: the indicator builder or the refresh task.
type refreshInvocation struct {
	task                                       bool
	state                                      cupertino.RefreshIndicatorMode
	pulledExtent, triggerDistance, indicatorEx float64
}

func (i refreshInvocation) String() string {
	if i.task {
		return "refreshTask"
	}
	return fmt.Sprintf("{%v pulled %v trigger %v extent %v}", i.state, i.pulledExtent, i.triggerDistance, i.indicatorEx)
}

// fakeRefresh is the builder and the refresh task of the tests; it records what calls it.
type fakeRefresh struct {
	t             *testing.T
	invocations   []refreshInvocation
	task          chan struct{}
	indicator     widgets.Widget
	taskCompleted bool
}

func newFakeRefresh(t *testing.T) *fakeRefresh {
	return &fakeRefresh{t: t, task: make(chan struct{}), indicator: widgets.NewContainer(nil)}
}

func (f *fakeRefresh) builder(_ widgets.BuildContext, state cupertino.RefreshIndicatorMode, pulled, trigger, extent float64) widgets.Widget {
	if pulled < 0 || trigger < 0 || extent < 0 {
		f.t.Errorf("a negative extent: pulled %v trigger %v extent %v", pulled, trigger, extent)
	}
	f.invocations = append(f.invocations, refreshInvocation{state: state, pulledExtent: pulled, triggerDistance: trigger, indicatorEx: extent})
	return f.indicator
}

func (f *fakeRefresh) refreshTask() <-chan struct{} {
	f.invocations = append(f.invocations, refreshInvocation{task: true})
	return f.task
}

func (f *fakeRefresh) expectBuilder(state cupertino.RefreshIndicatorMode, pulled float64, tolerance ...float64) bool {
	return f.has(refreshInvocation{state: state, pulledExtent: pulled, triggerDistance: 100, indicatorEx: 60}, tolerance...)
}

func (f *fakeRefresh) has(want refreshInvocation, tolerance ...float64) bool {
	for _, got := range f.invocations {
		if got.matches(want, tolerance...) {
			return true
		}
	}
	return false
}

func (i refreshInvocation) matches(want refreshInvocation, tolerance ...float64) bool {
	eps := 1e-9
	if len(tolerance) > 0 {
		eps = tolerance[0]
	}
	return i.task == want.task && i.state == want.state && math.Abs(i.pulledExtent-want.pulledExtent) <= eps &&
		math.Abs(i.triggerDistance-want.triggerDistance) <= eps && math.Abs(i.indicatorEx-want.indicatorEx) <= eps
}

// complete finishes the refresh task and runs what that posts, like the end of a future.
func (f *fakeRefresh) complete(ts *gluttertest.Tester) {
	f.t.Helper()
	close(f.task)
	f.taskCompleted = true
	// The control waits for the task on a goroutine and posts the end of the refresh to the UI thread.
	for range 200 {
		time.Sleep(time.Millisecond)
		ts.FlushPosted()
		if mode := refreshMode(ts); mode != cupertino.RefreshIndicatorModeRefresh && mode != cupertino.RefreshIndicatorModeArmed {
			return
		}
	}
}

func refreshMode(ts *gluttertest.Tester) cupertino.RefreshIndicatorMode {
	return cupertino.CupertinoSliverRefreshControlState(ts.Element(gluttertest.ByType[*widgets.LayoutBuilder](gluttertest.FindOpts{IncludeOffstage: true})))
}

func refreshList(length int) widgets.Widget {
	return widgets.NewSliverListBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
		return widgets.NewSizedBox(widgets.NewCenter(widgets.NewText(fmt.Sprint(i))), widgets.SizedBoxOpts{Height: foundation.Some(200.0)})
	}, widgets.SliverBuilderOpts{ItemCount: foundation.Some(length)})
}

func refreshApp(slivers ...widgets.Widget) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCustomScrollView(slivers)})
}

func useIOS(t *testing.T) {
	t.Helper()
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformIOS))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func noSlop() gluttertest.GestureOpts {
	return gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)}
}

func cellTop(ts *gluttertest.Tester, text string) float64 { return ts.TopLeft(sizedBoxOf(text)).Dy }

func nearTo(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-3 {
		t.Errorf("%s is %v, want %v", what, got, want)
	}
}

func rect(ts *gluttertest.Tester, f gluttertest.Finder) ui.Rect {
	tl, size := ts.TopLeft(f), ts.Size(f)
	return ui.Rect{Left: tl.Dx, Top: tl.Dy, Right: tl.Dx + size.Width, Bottom: tl.Dy + size.Height}
}

func centerOf(text string) gluttertest.Finder {
	return gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.Center]()))
}

func pumpRefreshApp(t *testing.T, f *fakeRefresh, onRefresh bool, extra ...cupertino.CupertinoSliverRefreshControlOpts) *gluttertest.Tester {
	t.Helper()
	o := cupertino.CupertinoSliverRefreshControlOpts{Builder: f.builder}
	if len(extra) > 0 {
		o = extra[0]
		if o.Builder == nil && !o.NoBuilder {
			o.Builder = f.builder
		}
	}
	if onRefresh {
		o.OnRefresh = f.refreshTask
	}
	ts := dialogTester(t, ui.Size{})
	ts.PumpWidget(refreshApp(cupertino.NewCupertinoSliverRefreshControl(o), refreshList(10)))
	return ts
}

func TestCupertinoSliverRefreshControl(t *testing.T) {
	t.Run("does not call anything without user interaction", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, false)
		if len(f.invocations) != 0 {
			t.Errorf("invocations %v", f.invocations)
		}
		if got := ts.TopLeft(sizedBoxOf("0")); got != (ui.Offset{}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("calls the indicator builder when starting to overscroll", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, false)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 50}, noSlop())
		ts.Pump()
		if len(f.invocations) != 1 || !f.invocations[0].matches(refreshInvocation{state: cupertino.RefreshIndicatorModeDrag, pulledExtent: 50, triggerDistance: 100, indicatorEx: 60}) {
			t.Errorf("invocations %v", f.invocations)
		}
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), 50)
	})
	t.Run("does not call the builder if the overscroll does not move the slivers, like on Android", func(t *testing.T) {
		skipUnlessDebug(t)
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
		t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
		f := newFakeRefresh(t)
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(widgets.MediaQueryData{DevicePixelRatio: 1},
			widgets.NewCustomScrollView([]widgets.Widget{cupertino.NewCupertinoSliverRefreshControl(cupertino.CupertinoSliverRefreshControlOpts{Builder: f.builder}), refreshList(10)}))))
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 50})
		ts.Pump()
		if len(f.invocations) != 0 {
			t.Errorf("invocations %v", f.invocations)
		}
		if got := ts.TopLeft(sizedBoxOf("0")); got != (ui.Offset{}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("lets the builder update as a canceled drag scrolls away", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, false)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 50}, noSlop())
		ts.Pump()
		ts.Pump(20 * time.Millisecond)
		ts.Pump(20 * time.Millisecond)
		ts.Pump(3 * time.Second)
		if len(f.invocations) != 3 {
			t.Fatalf("invocations %v", f.invocations)
		}
		for i, pulled := range []float64{50, 48.36801747187993, 44.63031931875867} {
			if !f.invocations[i].matches(refreshInvocation{state: cupertino.RefreshIndicatorModeDrag, pulledExtent: pulled, triggerDistance: 100, indicatorEx: 60}, 1e-6) {
				t.Errorf("invocation %d is %v, want a drag of %v", i, f.invocations[i], pulled)
			}
		}
		if got := ts.TopLeft(sizedBoxOf("0")); got != (ui.Offset{}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("a drag past the threshold triggers the refresh task", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true)
		var haptics []ui.HapticFeedbackKind
		ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) { haptics = append(haptics, k) }
		g := ts.StartGesture(ui.Offset{})
		g.MoveBy(ui.Offset{Dy: 99})
		ts.Pump()
		g.MoveBy(ui.Offset{Dy: -30})
		ts.Pump()
		g.MoveBy(ui.Offset{Dy: 50})
		ts.Pump()
		want := []refreshInvocation{
			{state: cupertino.RefreshIndicatorModeDrag, pulledExtent: 99}, {state: cupertino.RefreshIndicatorModeDrag, pulledExtent: 86.78169},
			{state: cupertino.RefreshIndicatorModeArmed, pulledExtent: 105.80452021305739}}
		if len(f.invocations) != 4 || !f.invocations[3].task {
			t.Fatalf("invocations %v", f.invocations)
		}
		for i, w := range want {
			w.triggerDistance, w.indicatorEx = 100, 60
			if !f.invocations[i].matches(w, 1e-4) {
				t.Errorf("invocation %d is %v, want %v", i, f.invocations[i], w)
			}
		}
		if len(haptics) == 0 || haptics[len(haptics)-1] != ui.HapticFeedbackMediumImpact {
			t.Errorf("haptic feedback %v, want the medium impact", haptics)
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("a refreshing task keeps the sliver expanded until it is done", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150}, noSlop())
		ts.Pump()
		ts.Pump(50 * time.Millisecond)
		if !f.expectBuilder(cupertino.RefreshIndicatorModeArmed, 150) || !f.has(refreshInvocation{task: true}) ||
			!f.expectBuilder(cupertino.RefreshIndicatorModeArmed, 127.10396988577114, 1e-6) {
			t.Errorf("invocations %v", f.invocations)
		}
		ts.Pump(time.Second)
		if !f.expectBuilder(cupertino.RefreshIndicatorModeRefresh, 60) {
			t.Errorf("no refresh state: %v", f.invocations)
		}
		ts.Pump(1000 * time.Second)
		nearTo(t, "the top of the first cell while refreshing", cellTop(ts, "0"), 60)
		f.complete(ts)
		ts.Pump()
		if !f.expectBuilder(cupertino.RefreshIndicatorModeDone, 60) {
			t.Errorf("no done state: %v", f.invocations)
		}
		if len(f.invocations) != 5 {
			t.Errorf("%d invocations: %v", len(f.invocations), f.invocations)
		}
	})
	t.Run("an expanded refreshing sliver scrolls normally", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		f.indicator = widgets.NewCenter(widgets.NewText("-1"))
		ts := pumpRefreshApp(t, f, true)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150}, noSlop())
		ts.Pump()
		if !f.expectBuilder(cupertino.RefreshIndicatorModeArmed, 150) {
			t.Errorf("no armed state: %v", f.invocations)
		}
		if got := rect(ts, centerOf("-1")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 150}) {
			t.Errorf("the indicator is at %v", got)
		}
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: -300}, noSlop())
		ts.Pump()
		if !f.expectBuilder(cupertino.RefreshIndicatorModeRefresh, 60) {
			t.Errorf("no refresh state: %v", f.invocations)
		}
		off := gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("-1", gluttertest.FindOpts{IncludeOffstage: true}), gluttertest.ByType[*widgets.Center](gluttertest.FindOpts{IncludeOffstage: true})))
		nearTo(t, "the top of the indicator", ts.TopLeft(off).Dy, -175.38461538461536)
		nearTo(t, "the bottom of the indicator", ts.TopLeft(off).Dy+ts.Size(off).Height, -115.38461538461536)
		nearTo(t, "the top of the first cell", ts.TopLeft(centerOf("0")).Dy, -115.38461538461536)
		ts.Drag(gluttertest.ByText("1"), ui.Offset{Dy: 200})
		ts.Pump()
		ts.Pump(2 * time.Second)
		if got := rect(ts, centerOf("-1")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 60}) {
			t.Errorf("the indicator is at %v", got)
		}
		if got := rect(ts, centerOf("0")); got != (ui.Rect{Left: 0, Top: 60, Right: 800, Bottom: 260}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("an expanded refreshing sliver goes away when done", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		f.indicator = widgets.NewCenter(widgets.NewText("-1"))
		ts := pumpRefreshApp(t, f, true)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150}, noSlop())
		ts.Pump()
		if got := rect(ts, centerOf("-1")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 150}) {
			t.Errorf("the indicator is at %v", got)
		}
		if !f.has(refreshInvocation{task: true}) {
			t.Error("the refresh task was not called")
		}
		ts.Pump()
		ts.Pump(2 * time.Second)
		if got := rect(ts, centerOf("-1")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 60}) {
			t.Errorf("the indicator is at %v", got)
		}
		if got := rect(ts, centerOf("0")); got != (ui.Rect{Left: 0, Top: 60, Right: 800, Bottom: 260}) {
			t.Errorf("the first cell is at %v", got)
		}
		f.complete(ts)
		ts.Pump()
		if !f.expectBuilder(cupertino.RefreshIndicatorModeDone, 60) {
			t.Errorf("no done state: %v", f.invocations)
		}
		ts.Pump(5 * time.Second)
		if ts.Exists(gluttertest.ByText("-1")) {
			t.Error("the indicator is still there")
		}
		if got := rect(ts, centerOf("0")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 200}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("the builder is still called when the sliver snapped back more than 90%", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		f.indicator = widgets.NewCenter(widgets.NewText("-1"))
		ts := pumpRefreshApp(t, f, true)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150}, noSlop())
		ts.Pump()
		ts.Pump()
		ts.Pump(2 * time.Second)
		f.complete(ts)
		ts.Pump()
		ts.Pump(400 * time.Millisecond)
		if got := ts.TopLeft(centerOf("0")).Dy; math.Abs(got-3.0) > 0.4 {
			t.Errorf("the top of the first cell is %v, want about 3", got)
		}
		if got := ts.Size(centerOf("-1")).Height; math.Abs(got-3.0) > 0.4 {
			t.Errorf("the height of the indicator is %v, want about 3", got)
		}
		if !f.expectBuilder(cupertino.RefreshIndicatorModeInactive, 2.6980688300546443, 1e-3) {
			t.Errorf("no inactive state: %v", f.invocations)
		}
		if !ts.Exists(gluttertest.ByText("-1")) {
			t.Error("the indicator is gone")
		}
	})
	t.Run("without onRefresh the builder is called armed for one frame, then the sliver goes away", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		f.indicator = widgets.NewCenter(widgets.NewText("-1"))
		ts := pumpRefreshApp(t, f, false)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150}, noSlop())
		ts.Pump()
		if len(f.invocations) == 0 || !f.invocations[0].matches(refreshInvocation{state: cupertino.RefreshIndicatorModeArmed, pulledExtent: 150, triggerDistance: 100, indicatorEx: 60}) {
			t.Fatalf("invocations %v", f.invocations)
		}
		ts.Pump(10 * time.Millisecond)
		if last := f.invocations[len(f.invocations)-1]; !last.matches(refreshInvocation{state: cupertino.RefreshIndicatorModeDone, pulledExtent: 148.6463892921364, triggerDistance: 100, indicatorEx: 60}, 1e-6) {
			t.Errorf("the last invocation is %v", last)
		}
		ts.Pump(5 * time.Second)
		if ts.Exists(gluttertest.ByText("-1")) {
			t.Error("the indicator is still there")
		}
		if got := rect(ts, centerOf("0")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 200}) {
			t.Errorf("the first cell is at %v", got)
		}
	})
	t.Run("does nothing unless it can be overscrolled at the start of the list", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(refreshApp(refreshList(10), cupertino.NewCupertinoSliverRefreshControl(cupertino.CupertinoSliverRefreshControlOpts{Builder: f.builder, OnRefresh: f.refreshTask}), refreshList(10)))
		ts.Fling(gluttertest.First(gluttertest.ByType[*widgets.SizedBox]()), ui.Offset{Dy: 200}, 2000)
		ts.Fling(gluttertest.First(gluttertest.ByType[*widgets.SizedBox]()), ui.Offset{Dy: -200}, 3000)
		if len(f.invocations) != 0 {
			t.Errorf("invocations %v", f.invocations)
		}
	})
	t.Run("does not crash when dragged", func(t *testing.T) {
		useIOS(t)
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCustomScrollView([]widgets.Widget{
			cupertino.NewCupertinoSliverRefreshControl(cupertino.CupertinoSliverRefreshControlOpts{OnRefresh: func() <-chan struct{} { return make(chan struct{}) }})},
			widgets.CustomScrollViewOpts{Physics: widgets.NewBouncingScrollPhysics()})}))
		ts.DragFrom(ui.Offset{Dx: 100, Dy: 10}, ui.Offset{Dy: 50}, noSlop())
		ts.Pump()
		ts.DragFrom(ui.Offset{Dx: 100, Dy: 10}, ui.Offset{Dy: 500}, noSlop())
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("displays properly behind the large title navigation bar sliver", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(refreshApp(cupertino.NewCupertinoSliverNavigationBar(cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText("Title")}),
			cupertino.NewCupertinoSliverRefreshControl(cupertino.CupertinoSliverRefreshControlOpts{Builder: f.builder}), refreshList(10)))
		initial := cellTop(ts, "0")
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 50}, noSlop())
		ts.Pump()
		if len(f.invocations) != 1 || !f.invocations[0].matches(refreshInvocation{state: cupertino.RefreshIndicatorModeDrag, pulledExtent: 50, triggerDistance: 100, indicatorEx: 60}) {
			t.Errorf("invocations %v", f.invocations)
		}
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), initial+50)
	})
	t.Run("does not crash when the paint extent is more than the remaining one", func(t *testing.T) {
		useIOS(t)
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCustomScrollView([]widgets.Widget{
			cupertino.NewCupertinoSliverRefreshControl(),
			widgets.NewSliverListBuilder(func(widgets.BuildContext, int) widgets.Widget {
				return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(100.0)})
			}, widgets.SliverBuilderOpts{ItemCount: foundation.Some(20)})},
			widgets.CustomScrollViewOpts{Physics: widgets.NewBouncingScrollPhysics()})}))
		ts.DragFrom(ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dy: 500})
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}

func TestCupertinoSliverRefreshControlStates(t *testing.T) {
	t.Run("starts in the inactive state", func(t *testing.T) {
		useIOS(t)
		ts := pumpRefreshApp(t, newFakeRefresh(t), false)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeInactive {
			t.Errorf("the state is %v", got)
		}
	})
	t.Run("goes to drag and returns to inactive in a small drag", func(t *testing.T) {
		useIOS(t)
		ts := pumpRefreshApp(t, newFakeRefresh(t), false)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 20})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeDrag {
			t.Errorf("the state is %v, want drag", got)
		}
		ts.Pump(2 * time.Second)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeInactive {
			t.Errorf("the state is %v, want inactive", got)
		}
	})
	t.Run("goes to armed the frame it passes the threshold", func(t *testing.T) {
		useIOS(t)
		ts := pumpRefreshApp(t, newFakeRefresh(t), false, cupertino.CupertinoSliverRefreshControlOpts{RefreshTriggerPullDistance: foundation.Some(80.0)})
		g := ts.StartGesture(ui.Offset{})
		g.MoveBy(ui.Offset{Dy: 79})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeDrag {
			t.Errorf("the state is %v, want drag", got)
		}
		g.MoveBy(ui.Offset{Dy: 3})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Errorf("the state is %v, want armed", got)
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("goes to refresh the frame it crossed back the refresh threshold", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true, cupertino.CupertinoSliverRefreshControlOpts{
			RefreshTriggerPullDistance: foundation.Some(90.0), RefreshIndicatorExtent: foundation.Some(50.0)})
		g := ts.StartGesture(ui.Offset{})
		g.MoveBy(ui.Offset{Dy: 90})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Fatalf("the state is %v, want armed", got)
		}
		g.MoveBy(ui.Offset{Dy: -80})
		ts.Pump()
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), 49.775111111111116)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeRefresh {
			t.Errorf("the state is %v, want refresh", got)
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("goes to done as soon as the task finishes", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true)
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 100}, noSlop())
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Fatalf("the state is %v, want armed", got)
		}
		ts.Pump()
		ts.Pump(2 * time.Second)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeRefresh {
			t.Errorf("the state is %v, want refresh", got)
		}
		if got := rect(ts, sizedBoxOf("0")); got != (ui.Rect{Left: 0, Top: 60, Right: 800, Bottom: 260}) {
			t.Errorf("the first cell is at %v", got)
		}
		f.complete(ts)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeDone {
			t.Errorf("the state is %v, want done", got)
		}
	})
	t.Run("goes back to inactive when retracting past 10% of the arming distance", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true)
		g := ts.StartGesture(ui.Offset{})
		g.MoveBy(ui.Offset{Dy: 150})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Fatalf("the state is %v, want armed", got)
		}
		f.complete(ts)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeDone {
			t.Fatalf("the state is %v, want done", got)
		}
		ts.Pump()
		g.MoveBy(ui.Offset{Dy: -200})
		ts.Pump()
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), 27.944444444444457)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeDone {
			t.Errorf("the state is %v, want done", got)
		}
		g.MoveBy(ui.Offset{Dy: -35})
		ts.Pump()
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), 9.313890708161875)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeInactive {
			t.Errorf("the state is %v, want inactive", got)
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("goes back to inactive if it is scrolled away when the task completes", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true)
		g := ts.StartGesture(ui.Offset{})
		g.MoveBy(ui.Offset{Dy: 150})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Fatalf("the state is %v, want armed", got)
		}
		ts.Pump() // The correction of the scroll offset is applied a frame later.
		g.MoveBy(ui.Offset{Dy: -300})
		ts.Pump()
		const destination = -145.0332383665717
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), destination)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeRefresh {
			t.Errorf("the state is %v, want refresh", got)
		}
		f.complete(ts)
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeInactive {
			t.Errorf("the state is %v, want inactive", got)
		}
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), destination)
		ts.Pump(2 * time.Second)
		nearTo(t, "the top of the first cell", cellTop(ts, "0"), destination)
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("builds no indicator and takes no space during the refresh when it has no builder", func(t *testing.T) {
		useIOS(t)
		f := newFakeRefresh(t)
		ts := pumpRefreshApp(t, f, true, cupertino.CupertinoSliverRefreshControlOpts{NoBuilder: true, RefreshIndicatorExtent: foundation.Some(0.0)})
		ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150})
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeArmed {
			t.Fatalf("the state is %v, want armed", got)
		}
		ts.Pump()
		ts.Pump(5 * time.Second)
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeRefresh {
			t.Errorf("the state is %v, want refresh", got)
		}
		if got := rect(ts, centerOf("0")); got != (ui.Rect{Left: 0, Top: 0, Right: 800, Bottom: 200}) {
			t.Errorf("the first cell is at %v", got)
		}
		f.complete(ts)
		ts.Pump()
		if got := refreshMode(ts); got != cupertino.RefreshIndicatorModeInactive {
			t.Errorf("the state is %v, want inactive", got)
		}
	})
}

func TestCupertinoSliverRefreshControlIndicator(t *testing.T) {
	build := func(t *testing.T, mode cupertino.RefreshIndicatorMode, pulled float64) *cupertino.CupertinoActivityIndicator {
		t.Helper()
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return cupertino.CupertinoSliverRefreshControlBuildRefreshIndicator(bc, mode, pulled, 100, 10)
		})}))
		return ts.Widget(gluttertest.ByType[*cupertino.CupertinoActivityIndicator]()).(*cupertino.CupertinoActivityIndicator)
	}
	t.Run("the progress follows the pulled extent", func(t *testing.T) {
		for _, pulled := range []float64{10, 26, 100} {
			if got := build(t, cupertino.RefreshIndicatorModeDrag, pulled).Progress(); got != pulled/100 {
				t.Errorf("pulled %v: progress %v", pulled, got)
			}
		}
	})
	t.Run("the indicator does not become larger when overscrolled", func(t *testing.T) {
		if got := build(t, cupertino.RefreshIndicatorModeDone, 120).Radius(); got != 14 {
			t.Errorf("the radius is %v", got)
		}
	})
	t.Run("the modes have their names", func(t *testing.T) {
		for m, want := range map[cupertino.RefreshIndicatorMode]string{
			cupertino.RefreshIndicatorModeInactive: "RefreshIndicatorMode.inactive", cupertino.RefreshIndicatorModeDrag: "RefreshIndicatorMode.drag",
			cupertino.RefreshIndicatorModeArmed: "RefreshIndicatorMode.armed", cupertino.RefreshIndicatorModeRefresh: "RefreshIndicatorMode.refresh",
			cupertino.RefreshIndicatorModeDone: "RefreshIndicatorMode.done", cupertino.RefreshIndicatorModeUnspecified: "RefreshIndicatorMode.unspecified"} {
			if m.String() != want {
				t.Errorf("%d is %q, want %q", int(m), m.String(), want)
			}
		}
	})
}

func TestCupertinoSliverRefreshControlErrors(t *testing.T) {
	skipUnlessDebug(t)
	for _, c := range []struct {
		name string
		opts cupertino.CupertinoSliverRefreshControlOpts
		tag  string
	}{
		{"a trigger distance that is not positive", cupertino.CupertinoSliverRefreshControlOpts{RefreshTriggerPullDistance: foundation.Some(0.0)}, "CupertinoSliverRefreshControlTriggerNotPositive"},
		{"a negative indicator extent", cupertino.CupertinoSliverRefreshControlOpts{RefreshIndicatorExtent: foundation.Some(-1.0)}, "CupertinoSliverRefreshControlExtentNegative"},
		{"an indicator extent above the trigger distance", cupertino.CupertinoSliverRefreshControlOpts{RefreshTriggerPullDistance: foundation.Some(50.0), RefreshIndicatorExtent: foundation.Some(60.0)}, "CupertinoSliverRefreshControlExtentAboveTrigger"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := recoverError(func() { cupertino.NewCupertinoSliverRefreshControl(c.opts) })
			checkErrorTag(t, err, "NewCupertinoSliverRefreshControl."+c.tag)
		})
	}
}

func TestCupertinoSliverRefreshControlDirection(t *testing.T) {
	skipUnlessDebug(t)
	ts := dialogTester(t, ui.Size{})
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCustomScrollView(
		[]widgets.Widget{cupertino.NewCupertinoSliverRefreshControl(), refreshList(3)}, widgets.CustomScrollViewOpts{Reverse: true})}))
	err := firstError(t, ts)
	checkErrorTag(t, err, "PerformLayout.CupertinoSliverRefreshDirection")
}
