package widgets_test

import (
	"fmt"
	"github.com/rm4n0s/glutter/rendering"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/list_view_test.dart.

func textItems(n int) []Widget {
	out := make([]Widget, n)
	for i := range out {
		out[i] = NewText(fmt.Sprintf("%d", i))
	}
	return out
}

func expectTexts(t *testing.T, ts *gluttertest.Tester, present []string, absent []string) {
	t.Helper()
	for _, s := range present {
		if ts.Count(gluttertest.ByText(s)) != 1 {
			t.Errorf("expected exactly one %q", s)
		}
	}
	for _, s := range absent {
		if ts.Count(gluttertest.ByText(s)) != 0 {
			t.Errorf("expected no %q", s)
		}
	}
}

func TestListViewBuilderRespectsFindChildIndexCallback(t *testing.T) {
	for _, name := range []string{"builder", "separated"} {
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			finderCalled := false
			itemCount := 7
			var stateSetter StateSetter
			find := func(foundation.Key) foundation.Opt[int] { finderCalled = true; return foundation.None[int]() }
			item := func(_ BuildContext, i int) Widget {
				return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(fmt.Sprint(i)), Height: foundation.Some(2000.0)})
			}
			ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
				stateSetter = setState
				if name == "builder" {
					return NewListViewBuilder(item, ListViewOpts{ItemCount: foundation.Some(itemCount), FindChildIndexCallback: find})
				}
				return NewListViewSeparated(item, func(BuildContext, int) Widget { return NewSizedBox(nil) }, itemCount,
					ListViewOpts{FindItemIndexCallback: find})
			})))
			if finderCalled {
				t.Fatal("the finder was called on the first build")
			}
			stateSetter(func() { itemCount = 77 })
			ts.Pump()
			if !finderCalled {
				t.Error("the finder was not called")
			}
		})
	}
}

func TestListViewDefaultControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewListView(nil, ListViewOpts{ItemExtent: foundation.Some(100.0)}))))
}

func TestListViewItemExtentControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	children := make([]Widget, 20)
	for i := range children {
		children[i] = NewColoredBox(ui.NewColor(0xFF00FF00), NewText(fmt.Sprint(i)))
	}
	ts.PumpWidget(ltr(NewListView(children, ListViewOpts{ItemExtent: foundation.Some(200.0)})))
	if h := ts.Size(gluttertest.First(gluttertest.ByType[*ColoredBox]())).Height; h != 200 {
		t.Errorf("height = %v, want 200", h)
	}
	expectTexts(t, ts, []string{"0", "1", "2"}, []string{"3", "4"})

	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: -250})
	ts.Pump()
	expectTexts(t, ts, []string{"1", "2", "3", "4"}, []string{"0", "5", "6"})

	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: 200})
	ts.Pump()
	expectTexts(t, ts, []string{"0", "1", "2", "3"}, []string{"4", "5"})
}

func TestListViewLargeScrollJump(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []int
	children := make([]Widget, 20)
	for i := range children {
		i := i
		children[i] = NewBuilder(func(BuildContext) Widget { log = append(log, i); return NewText(fmt.Sprint(i)) })
	}
	ts.PumpWidget(ltr(NewListView(children, ListViewOpts{ItemExtent: foundation.Some(200.0)})))
	want := func(expected ...int) {
		t.Helper()
		if fmt.Sprint(log) != fmt.Sprint(expected) {
			t.Errorf("log = %v, want %v", log, expected)
		}
		log = nil
	}
	want(0, 1, 2, 3, 4)
	position := scrollableOf(ts).Position()
	position.JumpTo(2025)
	want()
	ts.Pump()
	want(8, 9, 10, 11, 12, 13, 14)
	position.JumpTo(975)
	want()
	ts.Pump()
	want(7, 6, 5, 4, 3)
}

type aliveWidget struct {
	StatefulWidget
	alive bool
	index int
}

func newAlive(alive bool, index int) *aliveWidget {
	return &aliveWidget{NewStatefulWidgetBase(), alive, index}
}

func (w *aliveWidget) CreateState() State { return &aliveState{} }

type aliveState struct {
	StateOf[*aliveWidget]
	AutomaticKeepAliveClient
}

func (s *aliveState) WantKeepAlive() bool { return s.Widget().alive }

func (s *aliveState) Build(bc BuildContext) Widget {
	s.KeepAliveBuild()
	// The Dart test font is 14 pixels high; the box gives the item that height.
	return NewSizedBox(NewText(fmt.Sprintf("%d:%v", s.Widget().index, s.WantKeepAlive())), SizedBoxOpts{Height: foundation.Some(14.0)})
}

type statefulListViewWidget struct {
	StatefulWidget
	alive func(int) bool
}

func (w *statefulListViewWidget) CreateState() State { return &statefulListViewState{} }

type statefulListViewState struct {
	StateOf[*statefulListViewWidget]
}

func (s *statefulListViewState) Build(BuildContext) Widget {
	children := make([]Widget, 200)
	for i := range children {
		i := i
		children[i] = NewBuilder(func(BuildContext) Widget { return newAlive(s.Widget().alive(i), i) })
	}
	return NewGestureDetector(ltr(NewListView(children)), GestureDetectorOpts{OnTap: func() { s.SetState(func() {}) }})
}

func TestListViewLargeScrollJumpAndKeepAliveFirstChildNotKeepAlive(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	checkAndScroll := func(zero string) {
		t.Helper()
		expectTexts(t, ts, []string{zero, "1:false", "2:false", "3:true"}, []string{"116:false"})
		position := scrollableOf(ts).Position()
		position.JumpTo(1025)
		ts.Pump()
		expectTexts(t, ts, []string{"116:false"}, []string{zero, "1:false", "2:false"})
		if n := ts.Count(gluttertest.ByText("3:true", gluttertest.FindOpts{IncludeOffstage: true})); n != 1 {
			t.Errorf("3:true count with offstage = %d, want 1", n)
		}
		ts.TapAt(ui.Offset{Dx: 100, Dy: 100})
		position.JumpTo(0)
		ts.Pump()
		ts.Pump()
		expectTexts(t, ts, []string{zero, "1:false", "2:false", "3:true"}, nil)
	}
	ts.PumpWidget(&statefulListViewWidget{NewStatefulWidgetBase(), func(i int) bool { return i > 2 && i%3 == 0 }})
	checkAndScroll("0:false")
	ts.PumpWidget(&statefulListViewWidget{NewStatefulWidgetBase(), func(i int) bool { return i%3 == 0 }})
	checkAndScroll("0:true")
}

func TestListViewCanBuildOutOfUnderflow(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewListView(nil, ListViewOpts{ItemExtent: foundation.Some(100.0)})))
	expectTexts(t, ts, nil, []string{"0", "1", "2", "3", "4", "5"})
	ts.PumpWidget(ltr(NewListView(textItems(2), ListViewOpts{ItemExtent: foundation.Some(100.0)})))
	expectTexts(t, ts, []string{"0", "1"}, []string{"2", "3", "4", "5"})
	ts.PumpWidget(ltr(NewListView(textItems(5), ListViewOpts{ItemExtent: foundation.Some(100.0)})))
	expectTexts(t, ts, []string{"0", "1", "2", "3", "4"}, []string{"5"})
}

func TestListViewCanBuildOutOfOverflowPadding(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewListView([]Widget{NewText("padded")}, ListViewOpts{Padding: painting.NewEdgeInsetsAll(8)})))))
	if ts.Count(gluttertest.ByText("padded", gluttertest.FindOpts{IncludeOffstage: true})) != 1 {
		t.Error("padded text not found")
	}
}

func TestListViewWithItemExtentInUnboundedContext(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewSingleChildScrollView(NewListView(textItems(20), ListViewOpts{ItemExtent: foundation.Some(100.0), ShrinkWrap: true}))))
	expectTexts(t, ts, []string{"0", "19"}, nil)
}

type logDelegate struct {
	*SliverChildListDelegate
	log []string
}

func (d *logDelegate) DidFinishLayout(firstIndex, lastIndex int) {
	d.log = append(d.log, fmt.Sprintf("didFinishLayout firstIndex=%d lastIndex=%d", firstIndex, lastIndex))
}

func TestListViewDidFinishLayoutHasCorrectIndices(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	d := &logDelegate{SliverChildListDelegate: NewSliverChildListDelegate(textItems(20))}
	check := func(want ...string) {
		t.Helper()
		if fmt.Sprint(d.log) != fmt.Sprint(want) {
			t.Errorf("log = %v, want %v", d.log, want)
		}
		d.log = nil
	}
	ts.PumpWidget(ltr(NewListViewCustom(d, ListViewOpts{ItemExtent: foundation.Some(110.0)})))
	check("didFinishLayout firstIndex=0 lastIndex=7")
	ts.PumpWidget(ltr(NewListViewCustom(d, ListViewOpts{ItemExtent: foundation.Some(210.0)})))
	check("didFinishLayout firstIndex=0 lastIndex=4")
	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: -600})
	check()
	ts.Pump()
	check("didFinishLayout firstIndex=1 lastIndex=6")
}

func TestListViewAutomaticallyPadsMediaQueryOnAxis(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var inner painting.EdgeInsets
	ts.PumpWidget(ltr(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Padding: painting.NewEdgeInsetsAll(30)}),
		NewListView([]Widget{NewText("top"), NewBuilder(func(bc BuildContext) Widget {
			inner = MediaQueryPaddingOf(bc)
			return NewContainer(nil)
		})}))))
	// The top and bottom padding go into the sliver.
	if dy := ts.TopLeft(gluttertest.ByText("top")).Dy; dy != 30 {
		t.Errorf("top = %v, want 30", dy)
	}
	// The left and right padding is left as is for the children.
	if want := painting.NewEdgeInsetsSymmetric(0, 30); inner != want {
		t.Errorf("inner padding = %v, want %v", inner, want)
	}
}

func overscrollTapList(axis painting.Axis, itemExtent foundation.Opt[float64], controller ScrollController, tapped *bool) Widget {
	return ltr(NewListViewBuilder(func(_ BuildContext, i int) Widget {
		size := SizedBoxOpts{Height: foundation.Some(100.0)}
		if axis == painting.AxisHorizontal {
			size = SizedBoxOpts{Width: foundation.Some(100.0)}
		}
		return NewGestureDetector(NewSizedBox(NewText(fmt.Sprintf("Item %d", i)), size), GestureDetectorOpts{OnTap: func() { *tapped = true }})
	}, ListViewOpts{ItemExtent: itemExtent, Controller: controller, Physics: NewBouncingScrollPhysics(), ScrollDirection: axis, ItemCount: foundation.Some(15)}))
}

func TestListViewAllowsTouchOnChildrenWhenOverscrollingAndSettling(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	tapped := false
	controller := NewScrollController()
	defer controller.Dispose()
	frame := 16 * time.Millisecond
	tapAndCheck := func(text string) {
		t.Helper()
		ts.Tap(gluttertest.ByText(text))
		if !tapped {
			t.Errorf("tap on %q was not registered", text)
		}
		tapped = false
	}
	ts.PumpWidget(overscrollTapList(painting.AxisVertical, foundation.None[float64](), controller, &tapped))
	tapAndCheck("Item 0")
	ts.Fling(gluttertest.ByType[*ListView](), ui.Offset{Dy: 80}, 1000)
	for i := 0; i < 5; i++ {
		ts.Pump(frame)
	}
	if controller.Offset() >= 0 {
		t.Fatalf("offset = %v, want overscrolled", controller.Offset())
	}
	tapAndCheck("Item 1")
	ts.PumpAndSettle()
	if controller.Offset() != 0 {
		t.Errorf("offset = %v, want 0", controller.Offset())
	}
	tapAndCheck("Item 2")
	controller.JumpTo(101)
	if controller.Offset() != 101 {
		t.Errorf("offset = %v, want 101", controller.Offset())
	}
	tapAndCheck("Item 3")
	ts.PumpAndSettle()
	ts.Fling(gluttertest.ByType[*ListView](), ui.Offset{Dy: 500}, 5000)
	for i := 0; i < 5; i++ {
		ts.Pump(frame)
	}
	if controller.Offset() >= 0 {
		t.Fatalf("offset = %v, want overscrolled", controller.Offset())
	}
	tapAndCheck("Item 2")
	ts.Pump(frame)
	tapAndCheck("Item 2")
	ts.PumpAndSettle()
	tapAndCheck("Item 2")
}

func TestListViewAbsorbsTouchToStopScrollingWhenNotAtTheEdge(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	tapped := false
	controller := NewScrollController()
	defer controller.Dispose()
	frame := 16 * time.Millisecond
	ts.PumpWidget(overscrollTapList(painting.AxisVertical, foundation.None[float64](), controller, &tapped))
	controller.JumpTo(101)
	ts.Tap(gluttertest.ByText("Item 3"))
	if !tapped {
		t.Fatal("tap not registered")
	}
	tapped = false
	ts.Fling(gluttertest.ByType[*ListView](), ui.Offset{Dy: 100}, 1000)
	ts.Pump(frame)
	offset := controller.Offset()
	if !(offset < 101 && offset > 0) {
		t.Fatalf("offset = %v, want between 0 and 101", offset)
	}
	ts.Tap(gluttertest.ByText("Item 2"), gluttertest.GestureOpts{})
	if tapped {
		t.Error("the tap was not absorbed")
	}
	ts.Pump(frame)
	if offset != controller.Offset() {
		t.Errorf("offset moved from %v to %v", offset, controller.Offset())
	}
	ts.PumpAndSettle()
	if offset != controller.Offset() {
		t.Errorf("offset moved from %v to %v", offset, controller.Offset())
	}
	ts.Tap(gluttertest.ByText("Item 2"))
	if !tapped {
		t.Error("the tap on the idle list was not registered")
	}
}

func TestListViewHorizontalOverscrolledAtTheEndAllowsTouchesOnChildren(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	tapped := false
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(overscrollTapList(painting.AxisHorizontal, foundation.Some(100.0), controller, &tapped))
	ts.Tap(gluttertest.ByText("Item 3"))
	if !tapped {
		t.Fatal("tap not registered")
	}
	tapped = false
	ts.Fling(gluttertest.ByType[*ListView](), ui.Offset{Dx: -500}, 10000)
	for i := 0; i < 5; i++ {
		ts.Pump(16 * time.Millisecond)
	}
	if controller.Offset() <= controller.Position().MaxScrollExtent() {
		t.Fatalf("offset %v is not past the max extent %v", controller.Offset(), controller.Position().MaxScrollExtent())
	}
	ts.Tap(gluttertest.ByText("Item 14"))
	if !tapped {
		t.Fatal("tap while overscrolled not registered")
	}
	tapped = false
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("Item 14"))
	if !tapped {
		t.Error("tap after settling not registered")
	}
}

func TestListViewUpdatesViewportDimensionsWhenScrollDirectionChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	build := func(axis painting.Axis) Widget {
		return ltr(NewCenter(NewSizedBox(NewListView([]Widget{NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)})},
			ListViewOpts{Controller: controller, ScrollDirection: axis, ItemExtent: foundation.Some(50.0)}),
			SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(200.0)})))
	}
	for _, c := range []struct {
		axis painting.Axis
		want float64
	}{{painting.AxisHorizontal, 100}, {painting.AxisVertical, 200}, {painting.AxisHorizontal, 100}} {
		ts.PumpWidget(build(c.axis))
		if got := controller.Position().ViewportDimension(); got != c.want {
			t.Errorf("%v: viewport dimension = %v, want %v", c.axis, got, c.want)
		}
	}
}

func TestListViewRespectsClipBehavior(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	viewport := func() interface{ ClipBehavior() ui.Clip } {
		return ts.RenderObject(gluttertest.ByType[*Viewport]()).(interface{ ClipBehavior() ui.Clip })
	}
	ts.PumpWidget(ltr(NewListView([]Widget{NewContainer(nil, ContainerOpts{Height: foundation.Some(2000.0)})})))
	if got := viewport().ClipBehavior(); got != ui.ClipHardEdge {
		t.Errorf("clip = %v, want hardEdge", got)
	}
	ts.PumpWidget(ltr(NewListView([]Widget{NewContainer(nil, ContainerOpts{Height: foundation.Some(2000.0)})}, ListViewOpts{ClipBehavior: ui.ClipAntiAlias})))
	if got := viewport().ClipBehavior(); got != ui.ClipAntiAlias {
		t.Errorf("clip = %v, want antiAlias", got)
	}
}

func TestListViewItemExtentBuilderRespectsItemCount(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	numbers := []float64{10, 20, 30, 40, 50}
	ts.PumpWidget(ltr(NewListViewBuilder(func(_ BuildContext, i int) Widget {
		return NewSizedBox(NewText(fmt.Sprintf("Item %d", i)), SizedBoxOpts{Height: foundation.Some(numbers[i])})
	}, ListViewOpts{ItemCount: foundation.Some(len(numbers)), ItemExtentBuilder: func(i int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		return foundation.Some(numbers[i])
	}})))
	expectTexts(t, ts, []string{"Item 0", "Item 4"}, []string{"Item 5"})
}

func TestListViewItemExtentBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	var buildLog []int
	var dims rendering.SliverLayoutDimensions
	ts.PumpWidget(ltr(NewListViewBuilder(func(_ BuildContext, i int) Widget {
		buildLog = append(buildLog, i)
		return NewText(fmt.Sprintf("Item %d", i))
	}, ListViewOpts{Controller: controller, ItemExtentBuilder: func(_ int, d rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		dims = d
		return foundation.Some(100.0)
	}})))
	expectTexts(t, ts, []string{"Item 0", "Item 5"}, []string{"Item 6"})
	if want := (rendering.SliverLayoutDimensions{ScrollOffset: 0, PrecedingScrollExtent: 0, ViewportMainAxisExtent: 600, CrossAxisExtent: 800}); dims != want {
		t.Errorf("dimensions = %v, want %v", dims, want)
	}
	minMax := func() (int, int, int) {
		lo, hi := buildLog[0], buildLog[0]
		for _, v := range buildLog {
			lo, hi = min(lo, v), max(hi, v)
		}
		return len(buildLog), lo, hi
	}
	// viewport (600) + cache extent after (250)
	if n, lo, hi := minMax(); n != 9 || lo != 0 || hi != 8 {
		t.Errorf("build log = %d %d..%d, want 9 0..8", n, lo, hi)
	}
	for _, c := range []struct {
		jump                      float64
		first, last               int
		notBefore, notAfter       string
		count, minBuilt, maxBuilt int
	}{
		{10000, 100, 105, "Item 99", "Item 106", 12, 97, 108}, // cache extent before + viewport + cache extent after
		{5000, 50, 55, "Item 49", "Item 56", 12, 47, 58},
		{4700, 47, 52, "Item 46", "Item 53", 3, 44, 46}, // only the newly entered cached area items are loaded
		{5300, 53, 58, "Item 52", "Item 59", 6, 56, 61},
	} {
		buildLog = nil
		controller.JumpTo(c.jump)
		ts.Pump()
		expectTexts(t, ts, []string{fmt.Sprintf("Item %d", c.first), fmt.Sprintf("Item %d", c.last)}, []string{c.notBefore, c.notAfter})
		// Deviation: Flutter asks the builder again in each layout, so it sees each new
		// ScrollOffset; Glutter keeps the extents until the widget is rebuilt or the viewport,
		// cross axis or child count change (step 29.40, SPEC §12.10), so a jump asks only for
		// the rows not asked before, with the dimensions of the layout that first needed them.
		if dims.ViewportMainAxisExtent != 600 || dims.CrossAxisExtent != 800 {
			t.Errorf("jump %v: dimensions = %v", c.jump, dims)
		}
		if n, lo, hi := minMax(); n != c.count || lo != c.minBuilt || hi != c.maxBuilt {
			t.Errorf("jump %v: build log = %d %d..%d, want %d %d..%d", c.jump, n, lo, hi, c.count, c.minBuilt, c.maxBuilt)
		}
	}
}

func TestListViewItemExtentPrototypeAndBuilderConflict(t *testing.T) {
	skipUnlessDebug(t)
	item := func(_ BuildContext, i int) Widget { return NewText(fmt.Sprintf("Item %d", i)) }
	builder := func(int, rendering.SliverLayoutDimensions) foundation.Opt[float64] { return foundation.Some(100.0) }
	for name, o := range map[string]ListViewOpts{
		"builder and extent":    {ItemExtentBuilder: builder, ItemExtent: foundation.Some(100.0)},
		"builder and prototype": {ItemExtentBuilder: builder, PrototypeItem: NewContainer(nil)},
		"extent and prototype":  {ItemExtent: foundation.Some(100.0), PrototypeItem: NewContainer(nil)},
	} {
		t.Run(name, func(t *testing.T) {
			requireTag(t, "ListViewItemExtentConflict", func() { NewListViewBuilder(item, o) })
		})
	}
}
