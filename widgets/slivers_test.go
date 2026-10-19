package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/slivers_test.dart.
// Not ported: the semantics and painting cases (SliverOpacity, SliverEnsureSemantics),
// the cases that need an error widget override or a scroll offset correction
// found through the internals of the list, and the semanticIndexOffset cases.

// keptText is KeepAlive of slivers_test.dart: a text that keeps itself alive.
type keptText struct {
	StatefulWidget
	data string
}

func newKeptText(data string) *keptText { return &keptText{NewStatefulWidgetBase(), data} }

func (w *keptText) CreateState() State { return &keptTextState{} }

type keptTextState struct {
	StateOf[*keptText]
	AutomaticKeepAliveClient
}

func (s *keptTextState) WantKeepAlive() bool { return true }

func (s *keptTextState) Build(BuildContext) Widget {
	s.KeepAliveBuild()
	return NewText(s.Widget().data)
}

func viewportOfBoxes(offset, anchor float64) Widget {
	boxes := make([]Widget, 5)
	for i := range boxes {
		boxes[i] = NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(400.0)}))
	}
	return ltr(NewViewport(rendering.NewViewportOffsetFixed(offset), boxes, ViewportOpts{Anchor: anchor / 600}))
}

// verifySlivers checks the global positions of the SizedBoxes and the visibility
// of the SliverToBoxAdapters.
func verifySlivers(t *testing.T, ts *gluttertest.Tester, positions []float64, visibles []bool) {
	t.Helper()
	off := gluttertest.FindOpts{IncludeOffstage: true}
	var gotPositions []float64
	for _, e := range ts.Elements(gluttertest.ByType[*SizedBox](off)) {
		gotPositions = append(gotPositions, e.FindRenderObject().(rendering.Box).LocalToGlobal(ui.OffsetZero, nil).Dy)
	}
	var gotVisibles []bool
	for _, e := range ts.Elements(gluttertest.ByType[*SliverToBoxAdapter](off)) {
		gotVisibles = append(gotVisibles, e.FindRenderObject().(rendering.Sliver).Geometry().Visible)
	}
	if fmt.Sprint(gotPositions) != fmt.Sprint(positions) || fmt.Sprint(gotVisibles) != fmt.Sprint(visibles) {
		t.Errorf("positions %v visibles %v, want %v %v", gotPositions, gotVisibles, positions, visibles)
	}
}

func TestViewportBasic(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(viewportOfBoxes(0, 0))
	if got := ts.Size(gluttertest.ByType[*Viewport]()); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("viewport size = %v", got)
	}
	verifySlivers(t, ts, []float64{0, 400, 800, 1200, 1600}, []bool{true, true, false, false, false})
	ts.PumpWidget(viewportOfBoxes(200, 0))
	verifySlivers(t, ts, []float64{-200, 200, 600, 1000, 1400}, []bool{true, true, false, false, false})
	ts.PumpWidget(viewportOfBoxes(600, 0))
	verifySlivers(t, ts, []float64{-600, -200, 200, 600, 1000}, []bool{false, true, true, false, false})
	ts.PumpWidget(viewportOfBoxes(900, 0))
	verifySlivers(t, ts, []float64{-900, -500, -100, 300, 700}, []bool{false, false, true, true, false})
}

func TestViewportAnchor(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(viewportOfBoxes(0, 100))
	if got := ts.Size(gluttertest.ByType[*Viewport]()); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("viewport size = %v", got)
	}
	verifySlivers(t, ts, []float64{100, 500, 900, 1300, 1700}, []bool{true, true, false, false, false})
	ts.PumpWidget(viewportOfBoxes(200, 100))
	verifySlivers(t, ts, []float64{-100, 300, 700, 1100, 1500}, []bool{true, true, false, false, false})
	ts.PumpWidget(viewportOfBoxes(600, 100))
	verifySlivers(t, ts, []float64{-500, -100, 300, 700, 1100}, []bool{false, true, true, false, false})
	ts.PumpWidget(viewportOfBoxes(900, 100))
	verifySlivers(t, ts, []float64{-800, -400, 0, 400, 800}, []bool{false, false, true, true, false})
}

func TestMultipleGridsAndLists(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	box := func(child Widget) Widget { return NewSizedBox(child, SizedBoxOpts{Height: foundation.Some(22.2)}) }
	ts.PumpWidget(NewCenter(NewSizedBox(ltr(NewCustomScrollView([]Widget{
		NewSliverListList([]Widget{box(NewText("TOP")), box(nil), box(nil)}),
		NewSliverFixedExtentListList(22.2, []Widget{NewSizedBox(nil), NewText("A"), NewSizedBox(nil)}),
		NewSliverGrid(NewSliverChildListDelegate([]Widget{NewSizedBox(nil), NewText("B"), NewSizedBox(nil)}), rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2)),
		NewSliverListList([]Widget{box(nil), box(nil), box(NewText("BOTTOM"))}),
	})), SizedBoxOpts{Width: foundation.Some(44.4), Height: foundation.Some(60.0)})))
	gesture := ts.StartGesture(ui.Offset{Dx: 400, Dy: 300})
	// Only one of the texts is built at a time.
	for step, want := range []string{"TOP", "A", "B", "BOTTOM"} {
		if step > 0 {
			gesture.MoveBy(ui.Offset{Dy: -70})
			ts.Pump()
		}
		for _, text := range []string{"TOP", "A", "B", "BOTTOM"} {
			wantCount := 0
			if text == want {
				wantCount = 1
			}
			if got := ts.Count(gluttertest.ByText(text)); got != wantCount {
				t.Errorf("step %d: %q is built %d times, want %d", step, text, got, wantCount)
			}
		}
	}
	gesture.Up()
}

func TestSliverGridNegativeUsableCrossAxisExtent(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxSquare(foundation.Some(4.0), NewCustomScrollView([]Widget{NewSliverGrid(
		NewSliverChildListDelegate([]Widget{NewCenter(NewText("A")), NewCenter(NewText("B")), NewCenter(NewText("C")), NewCenter(NewText("D"))}),
		rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{CrossAxisSpacing: 8, MainAxisSpacing: 8}))})))))
	if err := ts.TakeError(); err != nil {
		t.Errorf("error: %v", err)
	}
}

func TestSliverFixedExtentListClearsGarbage(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	items := []string{"1", "2", "3", "4", "5", "6"}
	pump := func() {
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverFixedExtentListBuilder(func(_ BuildContext, i int) Widget {
			return NewCenter(newKeptText(items[i]), CenterOpts{Key: foundation.NewValueKey(items[i])})
		}, 900, SliverBuilderOpts{ItemCount: foundation.Some(len(items)), FindChildIndexCallback: func(key foundation.Key) foundation.Opt[int] {
			for i, item := range items {
				if key == foundation.Key(foundation.NewValueKey(item)) {
					return foundation.Some(i)
				}
			}
			return foundation.Some(-1)
		}})})))
	}
	pump()
	// Keep alive widgets need one frame to notify their parents.
	for _, dy := range []float64{-1200, -1200, -800} {
		ts.Drag(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: dy})
		ts.Pump()
	}
	present := func(want ...string) {
		t.Helper()
		for _, item := range []string{"0", "1", "2", "3", "4", "5"} {
			wantCount := 0
			for _, w := range want {
				if w == item {
					wantCount = 1
				}
			}
			if got := ts.Count(gluttertest.ByText(item)); got != wantCount {
				t.Errorf("%q is built %d times, want %d", item, got, wantCount)
			}
		}
	}
	present("4", "5")
	items = append([]string{"0"}, items...)
	pump()
	present("3", "4")
}

func TestSliverOffstage(t *testing.T) {
	for _, offstage := range []bool{true, false} {
		t.Run(fmt.Sprint("offstage ", offstage), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverOffstage(NewSliverToBoxAdapter(NewText("a")), SliverOffstageOpts{Offstage: foundation.Some(offstage)})})))
			wantTexts, wantExtent := 1, 16.0
			if offstage {
				wantTexts, wantExtent = 0, 0
			}
			if got := ts.Count(gluttertest.ByType[*Text]()); got != wantTexts {
				t.Errorf("%d texts, want %d", got, wantTexts)
			}
			viewport := ts.RenderObject(gluttertest.ByType[*Viewport]()).(*rendering.RenderViewport)
			if got := viewport.LastChild().Geometry().ScrollExtent; got != wantExtent {
				t.Errorf("scroll extent = %v, want %v", got, wantExtent)
			}
			if offstage && ts.Count(gluttertest.ByType[*SliverOffstage]()) != 0 {
				t.Error("the offstage sliver is found")
			}
		})
	}
}

func TestSliverIgnorePointer(t *testing.T) {
	tests := []struct {
		name     string
		ignoring foundation.Opt[bool]
		want     []string
	}{
		{"ignores pointer events", foundation.None[bool](), nil},
		{"ignores nothing", foundation.Some(false), []string{"tap"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var events []string
			ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverIgnorePointer(NewSliverToBoxAdapter(
				NewGestureDetector(NewSizedBox(NewText("a"), SizedBoxOpts{Height: foundation.Some(50.0)}),
					GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { events = append(events, "tap") }})),
				SliverIgnorePointerOpts{Ignoring: tc.ignoring})})))
			ts.TapAt(ui.Offset{Dx: 100, Dy: 20})
			if fmt.Sprint(events) != fmt.Sprint(tc.want) {
				t.Errorf("events = %v, want %v", events, tc.want)
			}
		})
	}
}

// tapTarget is an opaque tap target of 100 pixels, found by its key.
func tapTarget(label string, onTap func()) Widget {
	return NewGestureDetector(NewSizedBox(NewText(label), SizedBoxOpts{Key: foundation.NewValueKey(label), Height: foundation.Some(100.0)}),
		GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: onTap})
}

// twoColumnGrid places the children at the two halves of the first row.
func twoColumnGrid() rendering.SliverGridDelegate {
	return rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{ChildAspectRatio: foundation.Some(4.0)})
}

func TestSliverBuildersCreateChildrenThatCanBeHit(t *testing.T) {
	tests := []struct {
		name  string
		build func(first, second func()) Widget
	}{
		{"SliverList.builder", func(first, second func()) Widget {
			return NewSliverListBuilder(func(_ BuildContext, i int) Widget {
				return tapTarget(fmt.Sprint("Index ", i), map[bool]func(){true: first, false: second}[i%2 == 0])
			}, SliverBuilderOpts{ItemCount: foundation.Some(2)})
		}},
		{"SliverList.separated", func(first, second func()) Widget {
			return NewSliverListSeparated(func(_ BuildContext, i int) Widget {
				return tapTarget(fmt.Sprint("Index ", i), map[bool]func(){true: first, false: second}[i%2 == 0])
			}, func(BuildContext, int) Widget { return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}) },
				SliverSeparatedOpts{ItemCount: foundation.Some(2)})
		}},
		{"SliverList.list", func(first, second func()) Widget {
			return NewSliverListList([]Widget{tapTarget("Index 0", first), tapTarget("Index 1", second)})
		}},
		{"SliverFixedExtentList.builder", func(first, second func()) Widget {
			return NewSliverFixedExtentListBuilder(func(_ BuildContext, i int) Widget {
				return tapTarget(fmt.Sprint("Index ", i), map[bool]func(){true: first, false: second}[i%2 == 0])
			}, 100, SliverBuilderOpts{ItemCount: foundation.Some(2)})
		}},
		{"SliverFixedExtentList.list", func(first, second func()) Widget {
			return NewSliverFixedExtentListList(100, []Widget{tapTarget("Index 0", first), tapTarget("Index 1", second)})
		}},
		{"SliverGrid.builder", func(first, second func()) Widget {
			return NewSliverGridBuilder(twoColumnGrid(), func(_ BuildContext, i int) Widget {
				return tapTarget(fmt.Sprint("Index ", i), map[bool]func(){true: first, false: second}[i%2 == 0])
			}, SliverBuilderOpts{ItemCount: foundation.Some(2)})
		}},
		{"SliverGrid.list", func(first, second func()) Widget {
			return NewSliverGridList(twoColumnGrid(), []Widget{tapTarget("Index 0", first), tapTarget("Index 1", second)})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			firstTapped, secondTapped := 0, 0
			ts.PumpWidget(ltr(NewSizedBoxExpand(NewCustomScrollView([]Widget{tc.build(func() { firstTapped++ }, func() { secondTapped++ })}))))
			ts.Tap(gluttertest.ByKey(foundation.NewValueKey("Index 0")))
			if firstTapped != 1 || secondTapped != 0 {
				t.Errorf("after the first tap: %d, %d", firstTapped, secondTapped)
			}
			firstTapped = 0
			ts.Tap(gluttertest.ByKey(foundation.NewValueKey("Index 1")))
			if firstTapped != 0 || secondTapped != 1 {
				t.Errorf("after the second tap: %d, %d", firstTapped, secondTapped)
			}
		})
	}
}

func TestSliverListSeparatedHasTheCorrectNumberOfChildren(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewSizedBoxExpand(NewCustomScrollView([]Widget{NewSliverListSeparated(
		func(BuildContext, int) Widget { return NewText("item") }, func(BuildContext, int) Widget { return NewText("separator") },
		SliverSeparatedOpts{ItemCount: foundation.Some(2)})}))))
	if ts.Count(gluttertest.ByText("item")) != 2 || ts.Count(gluttertest.ByText("separator")) != 1 {
		t.Errorf("items %d separators %d", ts.Count(gluttertest.ByText("item")), ts.Count(gluttertest.ByText("separator")))
	}
}

func TestSliverGridListWithAnEmptyChildrenList(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewSizedBoxExpand(NewCustomScrollView([]Widget{NewSliverGridList(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2), nil)}))))
	if ts.Count(gluttertest.ByType[*CustomScrollView]()) != 1 {
		t.Error("no CustomScrollView")
	}
	_ = gestures.PressTimeout
	_ = painting.AxisVertical
}

func TestSliverLayoutBuilderGivesTheSliverConstraints(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var seen []rendering.SliverConstraints
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{
		NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)})),
		NewSliverLayoutBuilder(func(_ BuildContext, c rendering.SliverConstraints) Widget {
			seen = append(seen, c)
			return NewSliverToBoxAdapter(NewSizedBox(NewText("built"), SizedBoxOpts{Height: foundation.Some(2000.0)}))
		}),
	}, CustomScrollViewOpts{Controller: controller})))
	if len(seen) != 1 || seen[0].ScrollOffset != 0 || seen[0].PrecedingScrollExtent != 100 || seen[0].RemainingPaintExtent != 500 {
		t.Fatalf("constraints = %+v", seen)
	}
	if ts.Count(gluttertest.ByText("built")) != 1 {
		t.Error("the sliver was not built")
	}
	controller.JumpTo(50)
	ts.Pump()
	last := seen[len(seen)-1]
	if last.ScrollOffset != 0 || last.RemainingPaintExtent != 550 || last.PrecedingScrollExtent != 100 {
		t.Errorf("constraints after the scroll = %+v", last)
	}
}
