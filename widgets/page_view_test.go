package widgets_test

import (
	"fmt"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/page_view_test.dart.

func stateTexts() []Widget {
	out := make([]Widget, len(kStates))
	for i, s := range kStates {
		out[i] = NewText(s)
	}
	return out
}

func stateIndex(name string) int {
	for i, s := range kStates {
		if s == name {
			return i
		}
	}
	return -1
}

func pageOf(t *testing.T, c *PageController) float64 {
	t.Helper()
	p, ok := c.Page().Get()
	if !ok {
		t.Fatal("the controller has no page")
	}
	return p
}

func TestPageViewBuilderRespectsFindChildIndexCallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	finderCalled := false
	itemCount := 7
	var stateSetter StateSetter
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		stateSetter = setState
		return NewPageViewBuilder(func(_ BuildContext, i int) Widget {
			return NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(fmt.Sprint(i)), Height: foundation.Some(2000.0)})
		}, PageViewOpts{ItemCount: foundation.Some(itemCount), FindChildIndexCallback: func(foundation.Key) foundation.Opt[int] {
			finderCalled = true
			return foundation.None[int]()
		}})
	})))
	if finderCalled {
		t.Fatal("the finder was called on the first build")
	}
	stateSetter(func() { itemCount = 77 })
	ts.Pump()
	if !finderCalled {
		t.Error("the finder was not called")
	}
}

func zeroSizePageView(controller *PageController, size ui.Size) Widget {
	return ltr(NewCenter(NewSizedBoxFromSize(size, NewPageView(stateTexts(), PageViewOpts{Controller: controller, OnPageChanged: func(int) {}}))))
}

func TestPageViewResizeFromZeroSizeViewportShouldNotLoseState(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/88956.
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: 1})
	defer controller.Dispose()
	offstage := gluttertest.FindOpts{IncludeOffstage: true}
	// The page view has a zero viewport, so nothing is displayed.
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{}))
	expectTexts(t, ts, nil, []string{"Alabama"})
	if ts.Count(gluttertest.ByText("Alabama", offstage)) != 1 {
		t.Error("Alabama should exist offstage")
	}
	// Resized to a non-zero viewport, the initial page 1 is displayed.
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{Width: 200, Height: 200}))
	expectTexts(t, ts, []string{"Alaska"}, nil)
	controller.JumpToPage(stateIndex("Iowa"))
	ts.Pump()
	expectTexts(t, ts, []string{"Iowa"}, nil)
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{}))
	expectTexts(t, ts, nil, []string{"Iowa"})
	// The page view did not lose its state, so Iowa shows again.
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{Width: 200, Height: 200}))
	expectTexts(t, ts, []string{"Iowa"}, nil)
}

func TestPageViewChangeThePageThroughTheControllerWhenZeroSizeViewport(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: 1})
	defer controller.Dispose()
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{}))
	if got := ts.Size(gluttertest.ByType[*PageView]()); got != (ui.Size{}) {
		t.Errorf("size = %v", got)
	}
	expectTexts(t, ts, nil, []string{"Alabama"})
	controller.AnimateToPage(stateIndex("Iowa"), 300*time.Millisecond, animation.Curves.Ease)
	if got := pageOf(t, controller); got != float64(stateIndex("Iowa")) {
		t.Errorf("page = %v, want Iowa", got)
	}
	controller.JumpToPage(stateIndex("Illinois"))
	if got := pageOf(t, controller); got != float64(stateIndex("Illinois")) {
		t.Errorf("page = %v, want Illinois", got)
	}
	ts.PumpWidget(zeroSizePageView(controller, ui.Size{Width: 200, Height: 200}))
	if got := pageOf(t, controller); got != float64(stateIndex("Illinois")) {
		t.Errorf("page = %v, want Illinois", got)
	}
	expectTexts(t, ts, []string{"Illinois"}, nil)
}

func TestPageControllerCannotReturnPageWhileUnattached(t *testing.T) {
	controller := NewPageController()
	defer controller.Dispose()
	requirePanic(t, func() { controller.Page() })
}

func TestPageViewControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []string
	children := make([]Widget, len(kStates))
	for i, state := range kStates {
		state := state
		children[i] = NewGestureDetector(NewContainer(NewText(state), ContainerOpts{Height: foundation.Some(200.0), Color: foundation.Some(ui.NewColor(0xFF0000FF))}),
			GestureDetectorOpts{DragStartBehavior: gestures.DragStartBehaviorDown, OnTap: func() { log = append(log, state) }})
	}
	ts.PumpWidget(ltr(NewPageView(children, PageViewOpts{DragStartBehavior: gestures.DragStartBehaviorDown})))
	tapAndExpectLog(t, ts, &log, "Alabama")
	expectTexts(t, ts, nil, []string{"Alaska"})
	ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: -20})
	ts.Pump()
	expectTexts(t, ts, []string{"Alabama", "Alaska"}, []string{"Arizona"})
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Alabama"}, []string{"Alaska"})
	ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: -401})
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Alaska"}, []string{"Alabama", "Arizona"})
	tapAndExpectLog(t, ts, &log, "Alaska")
	ts.Fling(gluttertest.ByType[*PageView](), ui.Offset{Dx: -200}, 1000)
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Arizona"}, []string{"Alabama", "Alaska"})
	ts.Fling(gluttertest.ByType[*PageView](), ui.Offset{Dx: 200}, 1000)
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Alaska"}, []string{"Alabama", "Arizona"})
}

func TestPageViewDoesNotSquishWhenOverscrolled(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t, gluttertest.Opts{})
			pages := make([]Widget, 10)
			for i := range pages {
				pages[i] = NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(i), Color: foundation.Some(ui.NewColor(0xFF0000FF))})
			}
			ts.PumpWidget(ltr(NewPageView(pages)))
			finder := gluttertest.ByKey(foundation.NewValueKey(0))
			full := ui.Size{Width: 800, Height: 600}
			if ts.TopLeft(finder).Dx != 0 || ts.Size(finder) != full {
				t.Errorf("page 0 at %v size %v", ts.TopLeft(finder), ts.Size(finder))
			}
			// Going into overscroll.
			ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: 100})
			ts.Pump()
			if ts.TopLeft(finder).Dx <= 0 || ts.Size(finder) != full {
				t.Errorf("page 0 at %v size %v", ts.TopLeft(finder), ts.Size(finder))
			}
			// Easing the overscroll past the overscroll limit.
			if platform == foundation.TargetPlatformMacOS {
				ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: -500})
			} else {
				ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: -200})
			}
			ts.Pump()
			if ts.TopLeft(finder).Dx >= 0 || ts.Size(finder) != full {
				t.Errorf("page 0 at %v size %v", ts.TopLeft(finder), ts.Size(finder))
			}
		})
	}
}

func sizedPageView(controller *PageController, width, height float64) Widget {
	return ltr(NewCenter(NewSizedBox(NewPageView(stateTexts(), PageViewOpts{Controller: controller}),
		SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})))
}

func TestPageControllerControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: 4})
	defer controller.Dispose()
	ts.PumpWidget(sizedPageView(controller, 600, 400))
	expectTexts(t, ts, []string{"California"}, nil)
	controller.NextPage(150*time.Millisecond, animation.Curves.Ease)
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Colorado"}, nil)
	ts.PumpWidget(sizedPageView(controller, 300, 400))
	expectTexts(t, ts, []string{"Colorado"}, nil)
	controller.PreviousPage(150*time.Millisecond, animation.Curves.Ease)
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"California"}, nil)
}

func TestPageControllerPageStability(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(sizedPageView(nil, 600, 400))
	expectTexts(t, ts, []string{"Alabama"}, nil)
	ts.Drag(gluttertest.ByType[*PageView](), ui.Offset{Dx: -1250})
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Arizona"}, nil)
	ts.PumpWidget(sizedPageView(nil, 250, 100))
	expectTexts(t, ts, []string{"Arizona"}, nil)
	ts.PumpWidget(sizedPageView(nil, 450, 400))
	expectTexts(t, ts, []string{"Arizona"}, nil)
}

func TestPageControllerNextAndPreviousPageReturnFuturesThatResolve(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(NewPageView(stateTexts(), PageViewOpts{Controller: controller})))
	for _, move := range []func(time.Duration, animation.Curve) *scheduler.TickerFuture{
		func(d time.Duration, c animation.Curve) *scheduler.TickerFuture { return controller.NextPage(d, c) },
		func(d time.Duration, c animation.Curve) *scheduler.TickerFuture { return controller.PreviousPage(d, c) },
	} {
		completed := false
		move(150*time.Millisecond, animation.Curves.Ease).Then(func() { completed = true })
		if completed {
			t.Fatal("completed at once")
		}
		ts.Pump(200 * time.Millisecond)
		if completed {
			t.Fatal("completed too early")
		}
		ts.Pump(200 * time.Millisecond)
		if !completed {
			t.Fatal("did not complete")
		}
	}
}

func TestPageViewInZeroSizeContainer(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewPageView(stateTexts())))))
	if ts.Count(gluttertest.ByText("Alabama", gluttertest.FindOpts{IncludeOffstage: true})) != 1 {
		t.Error("Alabama should exist offstage")
	}
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxSquare(foundation.Some(200.0), NewPageView(stateTexts())))))
	expectTexts(t, ts, []string{"Alabama"}, nil)
}

func TestPageViewChangesAtHalfwayPoint(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []int
	ts.PumpWidget(ltr(NewPageView(stateTexts(), PageViewOpts{OnPageChanged: func(i int) { log = append(log, i) }})))
	want := func(expected ...int) {
		t.Helper()
		if fmt.Sprint(log) != fmt.Sprint(expected) {
			t.Errorf("log = %v, want %v", log, expected)
		}
		log = nil
	}
	want()
	gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	// The page view is 800 wide, so this move is just short of halfway.
	gesture.MoveBy(ui.Offset{Dx: -380})
	want()
	// The halfway mark is crossed.
	gesture.MoveBy(ui.Offset{Dx: -40})
	want(1)
	// Moving a bit more does not generate redundant notifications.
	gesture.MoveBy(ui.Offset{Dx: -40})
	want()
	for i := 0; i < 3; i++ {
		gesture.MoveBy(ui.Offset{Dx: -40})
		ts.Pump()
	}
	want()
	gesture.Up()
	ts.PumpAndSettle()
	want()
	expectTexts(t, ts, []string{"Alaska"}, []string{"Alabama"})
}

func TestPageViewBouncingScrollPhysicsBallisticsDoesNotOvershoot(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []int
	controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(0.9)})
	defer controller.Dispose()
	build := func(size foundation.Opt[ui.Size]) Widget {
		pageView := ltr(NewPageView(stateTexts(), PageViewOpts{Controller: controller, OnPageChanged: func(i int) { log = append(log, i) }, Physics: NewBouncingScrollPhysics()}))
		if s, ok := size.Get(); ok {
			return NewOverflowBox(pageView, OverflowBoxOpts{MinWidth: foundation.Some(s.Width), MinHeight: foundation.Some(s.Height),
				MaxWidth: foundation.Some(s.Width), MaxHeight: foundation.Some(s.Height)})
		}
		return pageView
	}
	check := func() {
		t.Helper()
		if len(log) != 0 {
			t.Errorf("log = %v", log)
		}
		ts.Fling(gluttertest.ByType[*PageView](), ui.Offset{Dx: 100}, 800)
		ts.PumpAndSettle()
		if len(log) != 0 {
			t.Errorf("log = %v", log)
		}
		expectTexts(t, ts, []string{"Alabama", "Alaska"}, []string{"Arizona"})
	}
	ts.PumpWidget(build(foundation.None[ui.Size]()))
	check()
	// Again with a Cupertino "Plus" device size.
	ts.PumpWidget(build(foundation.Some(ui.Size{Width: 414, Height: 736})))
	check()
}

func TestPageViewViewportFraction(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	build := func(controller *PageController) Widget {
		return ltr(NewPageViewBuilder(func(_ BuildContext, i int) Widget {
			color := ui.NewColor(0xFF00FF00)
			if i%2 == 0 {
				color = ui.NewColor(0xFF0000FF)
			}
			return NewContainer(NewText(kStates[i]), ContainerOpts{Height: foundation.Some(200.0), Color: foundation.Some(color)})
		}, PageViewOpts{Controller: controller, ItemCount: foundation.Some(len(kStates))}))
	}
	at := func(state string, x float64) {
		t.Helper()
		if got := ts.TopLeft(gluttertest.ByText(state)); got != (ui.Offset{Dx: x}) {
			t.Errorf("%s at %v, want %v", state, got, x)
		}
	}
	controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(7.0 / 8)})
	defer controller.Dispose()
	ts.PumpWidget(build(controller))
	at("Alabama", 50)
	at("Alaska", 750)
	controller.JumpToPage(10)
	ts.Pump()
	at("Georgia", -650)
	at("Hawaii", 50)
	at("Idaho", 750)
	controller2 := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(39.0 / 40)})
	defer controller2.Dispose()
	ts.PumpWidget(build(controller2))
	at("Georgia", -770)
	at("Hawaii", 10)
	at("Idaho", 790)
}

func TestPageViewPageSnappingDisableAndReenable(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []int
	build := func(snapping bool) Widget {
		return ltr(NewPageView(stateTexts(), PageViewOpts{PageSnapping: foundation.Some(snapping), OnPageChanged: func(i int) { log = append(log, i) }}))
	}
	want := func(expected ...int) {
		t.Helper()
		if fmt.Sprint(log) != fmt.Sprint(expected) {
			t.Errorf("log = %v, want %v", log, expected)
		}
		log = nil
	}
	ts.PumpWidget(build(true))
	want()
	// Drag more than halfway to the next page, to confirm the default behavior.
	gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	gesture.MoveBy(ui.Offset{Dx: -420})
	want(1)
	gesture.Up()
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Alaska"}, []string{"Alabama"})
	// Page snapping disabled, and moved halfway: it does not snap.
	ts.PumpWidget(build(false))
	gesture = ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	gesture.MoveBy(ui.Offset{Dx: -420})
	want(2) // page notifications are still sent
	gesture.Up()
	ts.PumpAndSettle()
	expectTexts(t, ts, []string{"Alaska", "Arizona"}, []string{"Alabama", "Arkansas"})
	// Snapping re-enabled: the page settles.
	ts.PumpWidget(build(true))
	ts.PumpAndSettle()
	want()
	expectTexts(t, ts, []string{"Arizona"}, []string{"Alaska", "Arkansas"})
}

var _ rendering.ViewportOffset

func largeFractionPageView(controller *PageController, itemCount int, text func(int) string) Widget {
	return ltr(NewPageViewBuilder(func(_ BuildContext, i int) Widget {
		color := ui.NewColor(0xFF00FF00)
		if i%2 == 0 {
			color = ui.NewColor(0xFF0000FF)
		}
		return NewContainer(NewText(text(i)), ContainerOpts{Height: foundation.Some(200.0), Color: foundation.Some(color)})
	}, PageViewOpts{Controller: controller, ItemCount: foundation.Some(itemCount)}))
}

func TestPageViewLargeViewportFraction(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(5.0 / 4)})
	defer controller.Dispose()
	ts.PumpWidget(largeFractionPageView(controller, len(kStates), func(i int) string { return kStates[i] }))
	alabama := gluttertest.ByText("Alabama")
	if ts.TopLeft(alabama) != (ui.Offset{Dx: -100}) || ts.BottomRight(alabama) != (ui.Offset{Dx: 900, Dy: 600}) {
		t.Errorf("Alabama is at %v..%v", ts.TopLeft(alabama), ts.BottomRight(alabama))
	}
	controller.JumpToPage(10)
	ts.Pump()
	if got := ts.TopLeft(gluttertest.ByText("Hawaii")); got != (ui.Offset{Dx: -100}) {
		t.Errorf("Hawaii is at %v", got)
	}
}

func TestPageViewUpdatingLargeViewportFraction(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	oldController := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(5.0 / 4)})
	defer oldController.Dispose()
	text := func(i int) string { return kStates[i] }
	ts.PumpWidget(largeFractionPageView(oldController, len(kStates), text))
	alabama := gluttertest.ByText("Alabama")
	if ts.TopLeft(alabama) != (ui.Offset{Dx: -100}) || ts.BottomRight(alabama) != (ui.Offset{Dx: 900, Dy: 600}) {
		t.Errorf("Alabama is at %v..%v", ts.TopLeft(alabama), ts.BottomRight(alabama))
	}
	newController := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(4.0)})
	defer newController.Dispose()
	ts.PumpWidget(largeFractionPageView(newController, len(kStates), text))
	newController.JumpToPage(10)
	ts.Pump()
	if got := ts.TopLeft(gluttertest.ByText("Hawaii")); got != (ui.Offset{Dx: -(4 - 1) * 800 / 2}) {
		t.Errorf("Hawaii is at %v", got)
	}
}

func TestPageViewLargeViewportFractionCanScrollToTheLastPageAndSnap(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/45096.
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(5.0 / 4)})
	defer controller.Dispose()
	ts.PumpWidget(largeFractionPageView(controller, 3, func(i int) string { return fmt.Sprint(i) }))
	if got := ts.Center(gluttertest.ByText("0")); got != (ui.Offset{Dx: 400, Dy: 300}) {
		t.Errorf("page 0 center = %v", got)
	}
	controller.JumpToPage(2)
	ts.Pump()
	ts.PumpAndSettle()
	if got := ts.Center(gluttertest.ByText("2")); got != (ui.Offset{Dx: 400, Dy: 300}) {
		t.Errorf("page 2 center = %v", got)
	}
}

func TestPageViewAllVisiblePagesReceiveTouchEvents(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/23873.
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(1.0 / 4)})
	defer controller.Dispose()
	tapped := -1
	ts.PumpWidget(ltr(NewPageViewBuilder(func(_ BuildContext, i int) Widget {
		return NewGestureDetector(NewSizedBoxExpand(NewText(fmt.Sprint(i))), GestureDetectorOpts{OnTap: func() { tapped = i }})
	}, PageViewOpts{Controller: controller, ItemCount: foundation.Some(20)})))
	for _, i := range []int{0, 1, 2} {
		if ts.Count(gluttertest.ByText(fmt.Sprint(i))) != 1 {
			t.Fatalf("page %d not found", i)
		}
		// The center of page 2 is at x = 800, so the tap is moved a bit to land on the screen.
		ts.TapAt(ts.Center(gluttertest.ByText(fmt.Sprint(i))).Sub(ui.Offset{Dx: 3}))
		if tapped != i {
			t.Errorf("tapped = %d, want %d", tapped, i)
		}
	}
	controller.JumpToPage(19)
	ts.Pump()
	for _, i := range []int{17, 18, 19} {
		if ts.Count(gluttertest.ByText(fmt.Sprint(i))) != 1 {
			t.Fatalf("page %d not found", i)
		}
		ts.Tap(gluttertest.ByText(fmt.Sprint(i)))
		if tapped != i {
			t.Errorf("tapped = %d, want %d", tapped, i)
		}
	}
}

func TestPageViewCurrentItemRemainsCenteredOnConstraintChange(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/50505.
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: len(kStates) - 1, ViewportFraction: foundation.Some(0.5)})
	defer controller.Dispose()
	build := func(size ui.Size) Widget { return zeroSizePageView(controller, size) }
	verifyCentered := func() {
		t.Helper()
		got := ts.Center(gluttertest.ByText(kStates[len(kStates)-1]))
		if !moreOrLessEquals(got.Dx, 400) || !moreOrLessEquals(got.Dy, 300) {
			t.Errorf("center = %v, want (400, 300)", got)
		}
	}
	ts.PumpWidget(build(ui.Size{Width: 300, Height: 300}))
	ts.PumpAndSettle()
	verifyCentered()
	ts.PumpWidget(build(ui.Size{Width: 200, Height: 300}))
	ts.PumpAndSettle()
	verifyCentered()
}

func TestPageViewDoesNotReportPageChangedOnOverscroll(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: len(kStates) - 1})
	defer controller.Dispose()
	changeIndex := 0
	ts.PumpWidget(ltr(NewPageView(stateTexts(), PageViewOpts{Controller: controller, OnPageChanged: func(p int) { changeIndex = p }})))
	controller.JumpToPage(len(kStates) * 2) // tries to move beyond the max range
	if changeIndex != 0 {
		t.Errorf("changeIndex = %d", changeIndex)
	}
	ts.Pump()
	if changeIndex != 0 {
		t.Errorf("changeIndex = %d", changeIndex)
	}
}

func TestPageMetrics(t *testing.T) {
	page := NewFixedPageMetrics(foundation.Some(100.0), foundation.Some(200.0), foundation.Some(150.0), foundation.Some(25.0),
		painting.AxisDirectionRight, 1, 1)
	if got := page.Page(); got != foundation.Some(6.0) {
		t.Errorf("page = %v, want 6", got)
	}
	page2 := page.CopyWith(ScrollMetricsChanges{Pixels: foundation.Some(page.Pixels() - 100)}).(PageMetrics)
	if got := page2.Page(); got != foundation.Some(4.0) {
		t.Errorf("page = %v, want 4", got)
	}
}

func TestPageControllerCanHandleRoundingIssue(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController()
	defer controller.Dispose()
	pages := make([]Widget, 3)
	for i := range pages {
		pages[i] = NewSemantics(NewText(fmt.Sprintf("Page #%d", i)), SemanticsOpts{Container: true})
	}
	ts.PumpWidget(ltr(NewPageView(pages, PageViewOpts{Controller: controller})))
	// Simulate a precision error.
	controller.Position().JumpTo(799.99999999999)
	if got := pageOf(t, controller); got != 1 {
		t.Errorf("page = %v, want 1", got)
	}
}

func TestPageViewPadEnds(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	viewport := func() *SliverFillViewport {
		return ts.Widget(gluttertest.ByType[*SliverFillViewport](gluttertest.FindOpts{IncludeOffstage: true})).(*SliverFillViewport)
	}
	ts.PumpWidget(ltr(NewPageView(nil)))
	if !viewport().PadEnds() {
		t.Error("padEnds should default to true")
	}
	ts.PumpWidget(ltr(NewPageView(nil, PageViewOpts{PadEnds: foundation.Some(false)})))
	if viewport().PadEnds() {
		t.Error("padEnds false was not propagated")
	}
}

func TestPageViewPrecisionErrorInsideRenderSliverFixedExtentBoxAdaptor(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/95101.
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewPageController(PageControllerOpts{InitialPage: 152})
	defer controller.Dispose()
	ts.PumpWidget(NewCenter(NewSizedBox(ltr(NewPageViewBuilder(func(BuildContext, int) Widget { return NewSizedBox(nil) },
		PageViewOpts{Controller: controller, ItemCount: foundation.Some(366)})), SizedBoxOpts{Width: foundation.Some(392.72727272727275)})))
	controller.JumpToPage(365)
	ts.Pump()
}

func TestPageControllerOnAttachAndOnDetach(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	attach, detach := 0, 0
	controller := NewPageController(PageControllerOpts{OnAttach: func(ScrollPosition) { attach++ }, OnDetach: func(ScrollPosition) { detach++ }})
	defer controller.Dispose()
	build := func(withPageView bool) Widget {
		if !withPageView {
			return ltr(NewSizedBox(nil))
		}
		return ltr(NewCenter(NewPageView([]Widget{NewCenter(NewText("First Page")), NewCenter(NewText("Second Page")), NewCenter(NewText("Third Page"))},
			PageViewOpts{Controller: controller, Physics: NewPageScrollPhysics(NewClampingScrollPhysics(nil))})))
	}
	ts.PumpWidget(build(true))
	ts.PumpAndSettle()
	if attach != 1 || detach != 0 {
		t.Errorf("attach %d detach %d, want 1 0", attach, detach)
	}
	controller.JumpToPage(1)
	ts.PumpAndSettle()
	if attach != 1 || detach != 0 {
		t.Errorf("after jumping: attach %d detach %d, want 1 0", attach, detach)
	}
	ts.PumpWidget(build(false))
	if attach != 1 || detach != 1 {
		t.Errorf("after removing: attach %d detach %d, want 1 1", attach, detach)
	}
}
