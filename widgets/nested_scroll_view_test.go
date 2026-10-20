package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/nested_scroll_view_test.dart.
// The Dart tests build their header with a pinned SliverAppBar and a TabBar and
// their body with a TabBarView; material arrives with milestone 11, so the header
// here is a pinned SliverPersistentHeader whose extents are the ones of the app
// bar (expanded: 200 down to 104, not expanded: 104) and the body is a ListView.
// The tests that need the app bar, the tabs, the drawer or other material widgets
// are not ported.

var nestedHeaderKey = foundation.NewValueKey("nested header")

type nestedHeaderDelegate struct {
	SliverPersistentHeaderDelegateBase
	max, min float64
}

func (d nestedHeaderDelegate) Build(BuildContext, float64, bool) Widget {
	return NewSizedBoxExpand(NewText("TTTTTTTT"), WidgetOpts{Key: nestedHeaderKey})
}
func (d nestedHeaderDelegate) MinExtent() float64 { return d.min }
func (d nestedHeaderDelegate) MaxExtent() float64 { return d.max }
func (d nestedHeaderDelegate) ShouldRebuild(old SliverPersistentHeaderDelegate) bool {
	o, ok := old.(nestedHeaderDelegate)
	return !ok || o != d
}

type nestedTestOpts struct {
	key        foundation.Key
	controller ScrollController
	collapsed  bool // an app bar that is not expanded
	physics    ScrollPhysics
	floating   bool
	body       Widget
}

func buildNestedTest(o nestedTestOpts) Widget {
	max := 200.0
	if o.collapsed {
		max = 104
	}
	body := o.body
	if body == nil {
		body = NewListView([]Widget{
			NewSizedBox(NewText("aaa1"), SizedBoxOpts{Height: foundation.Some(300.0)}),
			NewSizedBox(NewText("aaa2"), SizedBoxOpts{Height: foundation.Some(200.0)}),
			NewSizedBox(NewText("aaa3"), SizedBoxOpts{Height: foundation.Some(100.0)}),
			NewSizedBox(NewText("aaa4"), SizedBoxOpts{Height: foundation.Some(50.0)}),
		})
	}
	return ltr(NewMediaQuery(NewMediaQueryData(), NewNestedScrollView(func(BuildContext, bool) []Widget {
		return []Widget{NewSliverPersistentHeader(nestedHeaderDelegate{max: max, min: 104}, SliverPersistentHeaderOpts{Pinned: !o.floating, Floating: o.floating})}
	}, body, NestedScrollViewOpts{Key: o.key, Controller: o.controller, Physics: o.physics, FloatHeaderSlivers: o.floating})))
}

func headerHeight(ts *gluttertest.Tester) float64 {
	return ts.Size(gluttertest.ByKey(nestedHeaderKey)).Height
}

func TestNestedScrollViewScrollDirection(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/107101
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var received []rendering.ScrollDirection
	ts.PumpWidget(ltr(NewMediaQuery(NewMediaQueryData(), NewNotificationListener[ScrollNotification](
		func(n ScrollNotification) bool {
			if u, ok := n.(*UserScrollNotification); ok && u.Depth() == 1 {
				received = append(received, u.Direction)
			}
			return true
		},
		NewNestedScrollView(func(BuildContext, bool) []Widget {
			return []Widget{NewSliverPersistentHeader(nestedHeaderDelegate{max: 250, min: 56}, SliverPersistentHeaderOpts{Pinned: true})}
		}, NewListViewBuilder(func(_ BuildContext, i int) Widget {
			return NewSizedBox(NewCenter(NewText(fmt.Sprintf("Item %d", i))), SizedBoxOpts{Height: foundation.Some(50.0)})
		}, ListViewOpts{ItemCount: foundation.Some(30), Padding: painting.NewEdgeInsetsAll(8)}))))))
	// Fling to trigger ballistic activity.
	ts.Fling(gluttertest.ByText("Item 3"), ui.Offset{Dy: -250}, 10000)
	ts.PumpAndSettle()
	if want := []rendering.ScrollDirection{rendering.ScrollDirectionReverse, rendering.ScrollDirectionIdle}; fmt.Sprint(received) != fmt.Sprint(want) {
		t.Errorf("received %v, want %v", received, want)
	}
	received = nil
	// Drag forward, without ballistic activity.
	ts.Drag(gluttertest.ByText("Item 29"), ui.Offset{Dy: 20})
	ts.Pump()
	if want := []rendering.ScrollDirection{rendering.ScrollDirectionForward, rendering.ScrollDirectionIdle}; fmt.Sprint(received) != fmt.Sprint(want) {
		t.Errorf("received %v, want %v", received, want)
	}
}

func TestNestedScrollViewWithAScrollController(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 50})
	defer controller.Dispose()
	scrollOffset := 0.0
	controller.AddListener(func() { scrollOffset = controller.Offset() })
	ts.PumpWidget(buildNestedTest(nestedTestOpts{controller: controller}))
	position := controller.Position()
	if position.MinScrollExtent() != 0 || position.Pixels() != 50 || position.MaxScrollExtent() != 200 {
		t.Errorf("position: min %v pixels %v max %v", position.MinScrollExtent(), position.Pixels(), position.MaxScrollExtent())
	}
	// The expanded height - initialScrollOffset.
	if h := headerHeight(ts); h != 150 {
		t.Errorf("height = %v, want 150", h)
	}
	// Fully expand the header by scrolling (no animation) to 0.
	controller.JumpTo(0)
	ts.PumpAndSettle()
	if scrollOffset != 0 || headerHeight(ts) != 200 {
		t.Errorf("offset %v height %v", scrollOffset, headerHeight(ts))
	}
	// Scroll back to 50 animating over 100ms.
	controller.AnimateTo(50, 100*time.Millisecond, animation.Curves.Linear)
	ts.Pump()
	ts.Pump()
	if scrollOffset != 0 || headerHeight(ts) != 200 {
		t.Errorf("offset %v height %v", scrollOffset, headerHeight(ts))
	}
	ts.Pump(50 * time.Millisecond)
	if scrollOffset != 25 || headerHeight(ts) != 175 {
		t.Errorf("halfway: offset %v height %v", scrollOffset, headerHeight(ts))
	}
	ts.Pump(50 * time.Millisecond)
	if scrollOffset != 50 || headerHeight(ts) != 150 {
		t.Errorf("end: offset %v height %v", scrollOffset, headerHeight(ts))
	}
	// Scroll to the end of the outer scroll view; the first item of the body is
	// still visible.
	controller.JumpTo(controller.Position().MaxScrollExtent())
	ts.PumpAndSettle()
	if scrollOffset != 200 || ts.Count(gluttertest.ByText("aaa1")) != 1 {
		t.Errorf("max: offset %v", scrollOffset)
	}
}

func TestThreeNestedScrollViewsWithOneScrollController(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewTrackingScrollController()
	defer controller.Dispose()
	if controller.MostRecentlyUpdatedPosition() != nil || controller.InitialScrollOffset() != 0 {
		t.Fatal("a new tracking controller has a position")
	}
	page := func(title string) Widget {
		return NewMediaQuery(NewMediaQueryData(), NewNestedScrollView(func(BuildContext, bool) []Widget {
			return []Widget{
				NewSliverPersistentHeader(nestedHeaderDelegate{max: 200, min: 104}, SliverPersistentHeaderOpts{Pinned: true}),
				NewSliverToBoxAdapter(NewText(title)),
			}
		}, NewListView([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(1000.0)})}), NestedScrollViewOpts{Controller: controller}))
	}
	ts.PumpWidget(ltr(NewPageView([]Widget{page("Page0"), page("Page1"), page("Page2")})))
	if ts.Count(gluttertest.ByText("Page0")) != 1 || ts.Count(gluttertest.ByText("Page1")) != 0 || headerHeight(ts) != 200 {
		t.Fatalf("page 0: height %v", headerHeight(ts))
	}
	// A scroll collapses the header to 150.
	controller.JumpTo(50)
	ts.PumpAndSettle()
	if h := headerHeight(ts); h != 150 {
		t.Errorf("height = %v, want 150", h)
	}
	// Fling to Page1, whose header is as collapsed as the one of Page0.
	ts.Fling(gluttertest.ByText("Page0"), ui.Offset{Dx: -100}, 10000)
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("Page0")) != 0 || ts.Count(gluttertest.ByText("Page1")) != 1 || headerHeight(ts) != 150 {
		t.Errorf("page 1: height %v", headerHeight(ts))
	}
	controller.JumpTo(0)
	ts.PumpAndSettle()
	if h := headerHeight(ts); h != 200 {
		t.Errorf("height = %v, want 200", h)
	}
	ts.Fling(gluttertest.ByText("Page1"), ui.Offset{Dx: -100}, 10000)
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("Page1")) != 0 || ts.Count(gluttertest.ByText("Page2")) != 1 || headerHeight(ts) != 200 {
		t.Errorf("page 2: height %v", headerHeight(ts))
	}
}

func TestNestedScrollViewRespectsNeverScrollableScrollPhysics(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/113753
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(buildNestedTest(nestedTestOpts{physics: NewNeverScrollableScrollPhysics(nil), floating: true, body: NewContainer(nil)}))
	point1 := ts.Center(gluttertest.ByText("TTTTTTTT"))
	ts.DragFrom(point1, ui.Offset{Dy: -200})
	ts.Pump()
	if point2 := ts.Center(gluttertest.ByText("TTTTTTTT", gluttertest.FindOpts{IncludeOffstage: true})); point1 != point2 {
		t.Errorf("the header moved from %v to %v", point1, point2)
	}
}

func TestNestedScrollViewExposesInnerAndOuterControllers(t *testing.T) {
	useAndroid(t)
	tests := []struct {
		name                 string
		collapsed            bool
		scrollExtent         func(appBarHeight float64) float64
		wantOuter, wantInner func(appBarHeight, scrollExtent float64) float64
	}{
		{"less than the outer extent", true, func(h float64) float64 { return h - 50 },
			func(h, s float64) float64 { return 54 }, func(h, s float64) float64 { return 0 }},
		{"exactly the outer extent", true, func(h float64) float64 { return h },
			func(h, s float64) float64 { return 104 }, func(h, s float64) float64 { return 0 }},
		{"more than the outer extent", true, func(h float64) float64 { return h + 50 },
			func(h, s float64) float64 { return h }, func(h, s float64) float64 { return s - h }},
		{"less than the expanded outer extent", false, func(h float64) float64 { return h - 50 },
			func(h, s float64) float64 { return s }, func(h, s float64) float64 { return 0 }},
		{"exactly the expanded outer extent", false, func(h float64) float64 { return h },
			func(h, s float64) float64 { return s }, func(h, s float64) float64 { return 0 }},
		{"more than the expanded outer extent", false, func(h float64) float64 { return h + 50 },
			func(h, s float64) float64 { return h }, func(h, s float64) float64 { return s - h }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			key := NewGlobalKey[*NestedScrollViewState]()
			ts.PumpWidget(buildNestedTest(nestedTestOpts{key: key, collapsed: tc.collapsed}))
			appBarHeight := headerHeight(ts)
			wantHeight := 200.0
			if tc.collapsed {
				wantHeight = 104
			}
			if appBarHeight != wantHeight {
				t.Fatalf("height = %v, want %v", appBarHeight, wantHeight)
			}
			scrollExtent := tc.scrollExtent(appBarHeight)
			state, _ := key.CurrentState()
			if state.OuterController().Offset() != 0 || state.InnerController().Offset() != 0 {
				t.Fatal("the controllers start scrolled")
			}
			// The gesture occurs in the body, so the whole scroll view scrolls.
			gesture := ts.StartGesture(ui.Offset{Dy: appBarHeight + 1})
			gesture.MoveBy(ui.Offset{Dy: -scrollExtent})
			ts.Pump()
			wantOuter, wantInner := tc.wantOuter(appBarHeight, scrollExtent), tc.wantInner(appBarHeight, scrollExtent)
			if got := state.OuterController().Offset(); got != wantOuter {
				t.Errorf("outer offset = %v, want %v", got, wantOuter)
			}
			if got := state.InnerController().Offset(); got != wantInner {
				t.Errorf("inner offset = %v, want %v", got, wantInner)
			}
			gesture.Up()
		})
	}
}

func TestSliverOverlapInjectorWithoutAbsorber(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	handle := NewSliverOverlapAbsorberHandle()
	defer handle.Dispose()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverOverlapInjector(handle, nil)})))
	_ = requireErrorTag(t, takeFirst(t, ts), "SliverOverlapInjectorNoExtent")
}

func TestNestedScrollViewDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBox(buildNestedTest(nestedTestOpts{}), SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)}))))
	if ts.Count(gluttertest.ByType[*NestedScrollView]()) != 1 {
		t.Error("no NestedScrollView")
	}
}

func TestNestedScrollViewScrollPointerSignalDoesNotOverscroll(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(buildNestedTest(nestedTestOpts{controller: controller}))
	at := ts.Center(gluttertest.ByType[*NestedScrollView]())
	ts.ScrollAt(at, ui.Offset{Dy: 20})
	if controller.Offset() != 20 {
		t.Errorf("offset = %v, want 20", controller.Offset())
	}
	ts.ScrollAt(at, ui.Offset{Dy: -40})
	if controller.Offset() != 0 {
		t.Errorf("offset = %v, want 0", controller.Offset())
	}
	ts.ScrollAt(at, ui.Offset{Dy: 1000000})
	if controller.Offset() != 200 || ts.Count(gluttertest.ByText("aaa1")) != 1 {
		t.Errorf("offset = %v", controller.Offset())
	}
}

func TestNestedScrollViewBasicScrollWithPointerSignal(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/55362
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(buildNestedTest(nestedTestOpts{}))
	if headerHeight(ts) != 200 {
		t.Fatalf("height = %v, want 200", headerHeight(ts))
	}
	// The offset is the responsibility of the inner position.
	for _, step := range []struct{ at, height float64 }{{201, 180}, {179, 160}, {179, 140}} {
		ts.ScrollAt(ui.Offset{Dy: step.at}, ui.Offset{Dy: 20})
		ts.Pump(250 * time.Millisecond)
		if h := headerHeight(ts); h != step.height {
			t.Errorf("after scrolling at %v: height = %v, want %v", step.at, h, step.height)
		}
	}
}

func TestNestedScrollViewPointerSignalUpdatesTheScrollDirection(t *testing.T) {
	// Related to https://github.com/flutter/issues/64266
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(buildNestedTest(nestedTestOpts{controller: controller}))
	last := rendering.ScrollDirectionIdle
	controller.AddListener(func() {
		if d := controller.Position().UserScrollDirection(); d != rendering.ScrollDirectionIdle {
			last = d
		}
	})
	ts.Drag(gluttertest.ByType[*NestedScrollView](), ui.Offset{Dy: -20}, gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)})
	if last != rendering.ScrollDirectionReverse {
		t.Errorf("direction = %v, want reverse", last)
	}
	at := ts.Center(gluttertest.ByType[*NestedScrollView]())
	ts.ScrollAt(at, ui.Offset{Dy: 20})
	if last != rendering.ScrollDirectionReverse {
		t.Errorf("direction = %v, want reverse", last)
	}
	ts.Drag(gluttertest.ByType[*NestedScrollView](), ui.Offset{Dy: 20}, gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)})
	if last != rendering.ScrollDirectionForward {
		t.Errorf("direction = %v, want forward", last)
	}
	ts.ScrollAt(at, ui.Offset{Dy: -20})
	if last != rendering.ScrollDirectionForward {
		t.Errorf("direction = %v, want forward", last)
	}
}
