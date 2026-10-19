package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/grid_view_test.dart.

// kStates is the list of states of packages/flutter/test/widgets/states.dart.
var kStates = []string{
	"Alabama", "Alaska", "Arizona", "Arkansas", "California", "Colorado", "Connecticut", "Delaware", "Florida", "Georgia", "Hawaii", "Idaho",
	"Illinois", "Indiana", "Iowa", "Kansas", "Kentucky", "Louisiana", "Maine", "Maryland", "Massachusetts", "Michigan", "Minnesota",
	"Mississippi", "Missouri", "Montana", "Nebraska", "Nevada", "New Hampshire", "New Jersey", "New Mexico", "New York", "North Carolina",
	"North Dakota", "Ohio", "Oklahoma", "Oregon", "Pennsylvania", "Rhode Island", "South Carolina", "South Dakota", "Tennessee", "Texas",
	"Utah", "Vermont", "Virginia", "Washington", "West Virginia", "Wisconsin", "Wyoming",
}

// loggedTexts are n Builders that log their index and build a Text of it.
func loggedTexts(n int, log *[]int) []Widget {
	out := make([]Widget, n)
	for i := range out {
		i := i
		out[i] = NewBuilder(func(BuildContext) Widget { *log = append(*log, i); return NewText(fmt.Sprint(i)) })
	}
	return out
}

func requireLog(t *testing.T, log *[]int, want ...int) {
	t.Helper()
	if fmt.Sprint(*log) != fmt.Sprint(want) {
		t.Errorf("log = %v, want %v", *log, want)
	}
	*log = nil
}

func stateTiles(log *[]string) []Widget {
	out := make([]Widget, len(kStates))
	for i, state := range kStates {
		state := state
		out[i] = NewGestureDetector(NewColoredBox(ui.NewColor(0xFF0000FF), NewText(state)),
			GestureDetectorOpts{DragStartBehavior: gestures.DragStartBehaviorDown, OnTap: func() { *log = append(*log, state) }})
	}
	return out
}

func tapAndExpectLog(t *testing.T, ts *gluttertest.Tester, log *[]string, state string) {
	t.Helper()
	ts.Tap(gluttertest.ByText(state))
	if fmt.Sprint(*log) != fmt.Sprint([]string{state}) {
		t.Errorf("tap on %s: log = %v", state, *log)
	}
	*log = nil
}

func TestGridViewBuilderRespectsFindChildIndexCallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	finderCalled := false
	itemCount := 7
	var stateSetter StateSetter
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		stateSetter = setState
		return NewGridViewBuilder(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(4), func(_ BuildContext, i int) Widget {
			return NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(fmt.Sprint(i)), Height: foundation.Some(2000.0)})
		}, GridViewOpts{ItemCount: foundation.Some(itemCount), FindChildIndexCallback: func(foundation.Key) foundation.Opt[int] {
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

func TestEmptyGridView(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewGridViewCount(4, nil, GridViewOpts{DragStartBehavior: gestures.DragStartBehaviorDown})))
}

func TestGridViewCountAndExtentControl(t *testing.T) {
	for _, name := range []string{"count", "extent"} {
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var log []string
			if name == "count" {
				ts.PumpWidget(ltr(NewGridViewCount(4, stateTiles(&log), GridViewOpts{DragStartBehavior: gestures.DragStartBehaviorDown})))
			} else {
				ts.PumpWidget(ltr(NewGridViewExtent(200, stateTiles(&log), GridViewOpts{DragStartBehavior: gestures.DragStartBehaviorDown})))
			}
			if got := ts.Size(gluttertest.ByText("Arkansas")); got != (ui.Size{Width: 200, Height: 200}) {
				t.Errorf("Arkansas size = %v", got)
			}
			for i := 0; i < 8; i++ {
				tapAndExpectLog(t, ts, &log, kStates[i])
			}
			if name == "count" && ts.Count(gluttertest.ByText(kStates[12])) != 0 {
				t.Error("state 12 should not be built")
			}
			expectTexts(t, ts, nil, []string{"Nevada"})
			if name == "count" {
				ts.Drag(gluttertest.ByText("Arkansas"), ui.Offset{Dy: -200})
				ts.Pump()
				for i := 0; i < 4; i++ {
					expectTexts(t, ts, nil, []string{kStates[i]})
				}
				for i := 4; i < 12; i++ {
					tapAndExpectLog(t, ts, &log, kStates[i])
				}
				ts.Drag(gluttertest.ByText("Delaware"), ui.Offset{Dy: -4000})
			} else {
				ts.Drag(gluttertest.ByText("Arkansas"), ui.Offset{Dy: -4000})
			}
			ts.Pump()
			expectTexts(t, ts, nil, []string{"Alabama"})
			if name == "count" {
				expectTexts(t, ts, nil, []string{"Pennsylvania"})
			}
			if got := ts.Center(gluttertest.ByText("Tennessee")); got != (ui.Offset{Dx: 300, Dy: 100}) {
				t.Errorf("Tennessee center = %v", got)
			}
			tapAndExpectLog(t, ts, &log, "Tennessee")
			if name == "count" {
				ts.Drag(gluttertest.ByText("Tennessee"), ui.Offset{Dy: 200})
				ts.Pump()
				tapAndExpectLog(t, ts, &log, "Tennessee")
				tapAndExpectLog(t, ts, &log, "Pennsylvania")
			}
		})
	}
}

func TestGridViewLargeScrollJump(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []int
	ts.PumpWidget(ltr(NewGridViewExtent(200, loggedTexts(80, &log), GridViewOpts{ScrollDirection: painting.AxisHorizontal, ChildAspectRatio: foundation.Some(0.75)})))
	if got := ts.Size(gluttertest.ByText("4")); got != (ui.Size{Width: 200 / 0.75, Height: 200}) {
		t.Errorf("size = %v", got)
	}
	requireLog(t, &log, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	present := func(from, to int) {
		t.Helper()
		for i := 0; i < 80; i++ {
			want := 0
			if i >= from && i < to {
				want = 1
			}
			if got := ts.Count(gluttertest.ByText(fmt.Sprint(i))); got != want {
				t.Errorf("text %d: count %d, want %d", i, got, want)
			}
		}
	}
	present(0, 9)
	position := scrollableOf(ts).Position()
	position.JumpTo(3025)
	requireLog(t, &log)
	ts.Pump()
	requireLog(t, &log, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47)
	present(33, 45)
	position.JumpTo(975)
	requireLog(t, &log)
	ts.Pump()
	requireLog(t, &log, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23)
	present(9, 21)
}

func TestGridViewChangeDelegateRebuildsAndRelayouts(t *testing.T) {
	for _, name := range []string{"crossAxisCount", "maxCrossAxisExtent"} {
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var log []int
			delegate := func(first bool) rendering.SliverGridDelegate {
				if name == "crossAxisCount" {
					if first {
						return rendering.NewSliverGridDelegateWithFixedCrossAxisCount(4)
					}
					return rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2)
				}
				if first {
					return rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(200)
				}
				return rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(400)
			}
			firstLog := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
			ts.PumpWidget(ltr(NewGridView(delegate(true), loggedTexts(40, &log))))
			if got := ts.Size(gluttertest.ByText("4")); got != (ui.Size{Width: 200, Height: 200}) {
				t.Errorf("size = %v", got)
			}
			requireLog(t, &log, firstLog...)
			for i := 0; i < 40; i++ {
				want := 0
				if i < 12 {
					want = 1
				}
				if ts.Count(gluttertest.ByText(fmt.Sprint(i))) != want {
					t.Errorf("text %d: want %d", i, want)
				}
			}
			ts.PumpWidget(ltr(NewGridView(delegate(false), loggedTexts(40, &log))))
			requireLog(t, &log, firstLog...)
			if got := ts.Size(gluttertest.ByText("3")); got != (ui.Size{Width: 400, Height: 400}) {
				t.Errorf("size = %v", got)
			}
			expectTexts(t, ts, nil, []string{"4"})
		})
	}
}

func TestSliverGridRegularTileLayoutCloseToZeroMainAxisStride(t *testing.T) {
	d := rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(500, rendering.SliverGridDelegateWithMaxCrossAxisExtentOpts{ChildAspectRatio: foundation.Some(1e300)})
	layout := d.GetLayout(rendering.SliverConstraints{
		AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionForward,
		ScrollOffset: 100, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 0, CrossAxisExtent: 500,
		CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 0, CacheOrigin: 0,
	})
	if got := layout.GetMinChildIndexForScrollOffset(1000); got != 0 {
		t.Errorf("min child index = %v, want 0", got)
	}
}

func TestGridViewInZeroContext(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewGridViewCount(4, textItems(20))))))
	expectTexts(t, ts, nil, []string{"0", "1"})
}

func TestGridViewInUnboundedContext(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewSingleChildScrollView(NewGridViewCount(4, textItems(20), GridViewOpts{ShrinkWrap: true}))))
	expectTexts(t, ts, []string{"0", "19"}, nil)
}

func TestGridViewBuilderControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewGridViewBuilder(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(4), func(_ BuildContext, i int) Widget {
		return NewText(fmt.Sprint(i))
	}, GridViewOpts{ShrinkWrap: true, ItemCount: foundation.Some(20)})))
	expectTexts(t, ts, []string{"0", "11"}, []string{"12"})
}

func TestGridViewBuilderWithUndefinedItemCount(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewGridViewBuilder(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(4), func(_ BuildContext, i int) Widget {
		return NewText(fmt.Sprint(i))
	}, GridViewOpts{ShrinkWrap: true})))
	expectTexts(t, ts, []string{"0", "11"}, nil)
	ts.Drag(gluttertest.ByType[*GridView](), ui.Offset{Dy: -300})
	ts.Pump(200 * time.Millisecond)
	expectTexts(t, ts, []string{"13"}, nil)
}

func TestGridViewCrossAxisLayout(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	target := NewGlobalKey[State]()
	build := func(dir ui.TextDirection) Widget {
		return NewDirectionality(dir, NewGridViewCount(4, []Widget{NewContainer(nil, ContainerOpts{Key: target})}))
	}
	ts.PumpWidget(build(ui.TextDirectionLTR))
	check := func(topLeft, bottomRight ui.Offset) {
		t.Helper()
		if got := ts.TopLeft(gluttertest.ByKey(target)); got != topLeft {
			t.Errorf("top left = %v, want %v", got, topLeft)
		}
		if got := ts.BottomRight(gluttertest.ByKey(target)); got != bottomRight {
			t.Errorf("bottom right = %v, want %v", got, bottomRight)
		}
	}
	check(ui.Offset{}, ui.Offset{Dx: 200, Dy: 200})
	ts.PumpWidget(build(ui.TextDirectionRTL))
	check(ui.Offset{Dx: 600}, ui.Offset{Dx: 800, Dy: 200})
}

func TestGridViewCrossAxisSpacing(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	target := NewGlobalKey[State]()
	build := func(dir ui.TextDirection) Widget {
		return NewDirectionality(dir, NewGridViewCount(4, []Widget{NewContainer(nil, ContainerOpts{Key: target})}, GridViewOpts{CrossAxisSpacing: 8}))
	}
	ts.PumpWidget(build(ui.TextDirectionLTR))
	if got := ts.TopLeft(gluttertest.ByKey(target)); got != (ui.Offset{}) {
		t.Errorf("top left = %v", got)
	}
	if got := ts.BottomRight(gluttertest.ByKey(target)); got != (ui.Offset{Dx: 194, Dy: 194}) {
		t.Errorf("bottom right = %v", got)
	}
	ts.PumpWidget(build(ui.TextDirectionRTL))
	if got := ts.TopLeft(gluttertest.ByKey(target)); got != (ui.Offset{Dx: 606}) {
		t.Errorf("top left = %v", got)
	}
	if got := ts.BottomRight(gluttertest.ByKey(target)); got != (ui.Offset{Dx: 800, Dy: 194}) {
		t.Errorf("bottom right = %v", got)
	}
}

func TestGridViewDoesNotCacheItemBuilderCalls(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	counters := map[int]int{}
	ts.PumpWidget(ltr(NewGridViewBuilder(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(3), func(_ BuildContext, i int) Widget {
		counters[i]++
		return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i), Width: foundation.Some(200.0), Height: foundation.Some(200.0)})
	}, GridViewOpts{ItemCount: foundation.Some(1000)})))
	key4 := gluttertest.ByKey(foundation.NewValueKey(4))
	if ts.Count(key4) != 1 || counters[4] != 1 {
		t.Fatalf("item 4: count %d, built %d", ts.Count(key4), counters[4])
	}
	ts.Fling(gluttertest.ByType[*GridView](), ui.Offset{Dy: -300}, 5000)
	ts.PumpAndSettle()
	if ts.Count(key4) != 0 || counters[4] != 1 {
		t.Errorf("item 4 after the fling up: count %d, built %d", ts.Count(key4), counters[4])
	}
	ts.Fling(gluttertest.ByType[*GridView](), ui.Offset{Dy: 300}, 5000)
	ts.PumpAndSettle()
	if ts.Count(key4) != 1 || counters[4] != 2 {
		t.Errorf("item 4 after the fling down: count %d, built %d", ts.Count(key4), counters[4])
	}
}

func TestGridViewRespectsClipBehavior(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	viewport := func() interface{ ClipBehavior() ui.Clip } {
		return ts.RenderObject(gluttertest.ByType[*Viewport]()).(interface{ ClipBehavior() ui.Clip })
	}
	children := make([]Widget, 13)
	for i := range children {
		children[i] = NewContainer(nil, ContainerOpts{Height: foundation.Some(2000.0)})
	}
	delegate := rendering.NewSliverGridDelegateWithFixedCrossAxisCount(3)
	ts.PumpWidget(ltr(NewGridView(delegate, children)))
	if got := viewport().ClipBehavior(); got != ui.ClipHardEdge {
		t.Errorf("clip = %v, want hardEdge", got)
	}
	ts.PumpWidget(ltr(NewGridView(delegate, children, GridViewOpts{ClipBehavior: ui.ClipAntiAlias})))
	if got := viewport().ClipBehavior(); got != ui.ClipAntiAlias {
		t.Errorf("clip = %v, want antiAlias", got)
	}
}
