package widgets_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/two_dimensional_scroll_view_test.dart
// and two_dimensional_viewport_test.dart, on the table of two_dimensional_utils.dart:
// a viewport of 200x200 squares. Not ported: the diagonal drag tests, the fling
// tests, the keep alive tests, restoration (not ported yet) and the show on
// screen tests.

// tableViewport is SimpleBuilderTableViewport of two_dimensional_utils.dart.
type tableViewport struct {
	TwoDimensionalViewport
	useCacheExtent, applyDimensions, forgetToLayoutChild, setLayoutOffset bool
}

func (w *tableViewport) CreateElement() Element { return NewTwoDimensionalViewportElement(w) }

func (w *tableViewport) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := &renderTable{useCacheExtent: w.useCacheExtent, applyDimensions: w.applyDimensions,
		forgetToLayoutChild: w.forgetToLayoutChild, setLayoutOffset: w.setLayoutOffset}
	rendering.InitRenderTwoDimensionalViewport(&r.RenderTwoDimensionalViewport, r, w.HorizontalOffset(), w.HorizontalAxisDirection(),
		w.VerticalOffset(), w.VerticalAxisDirection(), w.Delegate(), w.MainAxis(), bc.(rendering.TwoDimensionalChildManager),
		rendering.RenderTwoDimensionalViewportOpts{ScrollCacheExtent: w.ScrollCacheExtent(), ClipBehavior: w.ClipBehavior()})
	return r
}

func (w *tableViewport) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	w.UpdateViewport(ro.(*renderTable).AsRenderTwoDimensionalViewport())
}

// renderTable is RenderSimpleBuilderTableViewport.
type renderTable struct {
	rendering.RenderTwoDimensionalViewport
	useCacheExtent, applyDimensions, forgetToLayoutChild, setLayoutOffset bool
}

func (r *renderTable) LayoutChildSequence() {
	horizontalPixels, verticalPixels := r.HorizontalOffset().Pixels(), r.VerticalOffset().Pixels()
	cache := r.ScrollCacheExtent()
	horizontalCache, verticalCache := 0.0, 0.0
	if r.useCacheExtent {
		horizontalCache = cache.CalculateCacheOffset(r.ViewportDimension().Width)
		verticalCache = cache.CalculateCacheOffset(r.ViewportDimension().Height)
	}
	viewport := r.ViewportDimension()
	delegate := r.Delegate().(*TwoDimensionalChildBuilderDelegate)
	maxRow, maxColumn := delegate.MaxYIndex().OrElse(5), delegate.MaxXIndex().OrElse(5)
	leadingColumn := int(math.Max(math.Floor((horizontalPixels-horizontalCache)/200), 0))
	leadingRow := int(math.Max(math.Floor((verticalPixels-verticalCache)/200), 0))
	trailingColumn := int(math.Min(math.Ceil((horizontalPixels+viewport.Width+horizontalCache)/200), float64(maxColumn)))
	trailingRow := int(math.Min(math.Ceil((verticalPixels+viewport.Height+verticalCache)/200), float64(maxRow)))
	xLayoutOffset := float64(leadingColumn*200) - horizontalPixels
	for column := leadingColumn; column <= trailingColumn; column++ {
		yLayoutOffset := float64(leadingRow*200) - verticalPixels
		for row := leadingRow; row <= trailingRow; row++ {
			child := r.BuildOrObtainChildFor(NewChildVicinity(column, row))
			if !r.forgetToLayoutChild && child != nil {
				child.Layout(r.BoxConstraints().Tighten(rendering.TightenOpts{Width: foundation.Some(200.0), Height: foundation.Some(200.0)}))
			}
			if r.setLayoutOffset && child != nil {
				r.ParentDataOf(child).LayoutOffset = foundation.Some(ui.Offset{Dx: xLayoutOffset, Dy: yLayoutOffset})
			}
			yLayoutOffset += 200
		}
		xLayoutOffset += 200
	}
	if r.applyDimensions {
		r.VerticalOffset().ApplyContentDimensions(0, ui.ClampDouble(float64(200*(maxRow+1))-viewport.Height, 0, math.Inf(1)))
		r.HorizontalOffset().ApplyContentDimensions(0, ui.ClampDouble(float64(200*(maxColumn+1))-viewport.Width, 0, math.Inf(1)))
	}
}

// tableView is SimpleBuilderTableView.
type tableView struct {
	TwoDimensionalScrollView
	scrollCacheExtent                                                     foundation.Opt[rendering.ScrollCacheExtent]
	useCacheExtent, applyDimensions, forgetToLayoutChild, setLayoutOffset bool
}

type tableOpts struct {
	TwoDimensionalScrollViewOpts
	useCacheExtent, skipApplyDimensions, forgetToLayoutChild, skipLayoutOffset bool
}

func newTableView(delegate *TwoDimensionalChildBuilderDelegate, o tableOpts) *tableView {
	v := &tableView{scrollCacheExtent: o.ScrollCacheExtent, useCacheExtent: o.useCacheExtent, applyDimensions: !o.skipApplyDimensions,
		forgetToLayoutChild: o.forgetToLayoutChild, setLayoutOffset: !o.skipLayoutOffset}
	InitTwoDimensionalScrollView(&v.TwoDimensionalScrollView, v, delegate, o.TwoDimensionalScrollViewOpts)
	return v
}

func (v *tableView) BuildViewport(_ BuildContext, verticalOffset, horizontalOffset rendering.ViewportOffset) Widget {
	vd, hd := v.verticalDetails(), v.horizontalDetails()
	w := &tableViewport{useCacheExtent: v.useCacheExtent, applyDimensions: v.applyDimensions,
		forgetToLayoutChild: v.forgetToLayoutChild, setLayoutOffset: v.setLayoutOffset}
	w.TwoDimensionalViewport = NewTwoDimensionalViewportBase(verticalOffset, vd.Direction, horizontalOffset, hd.Direction,
		v.Delegate(), v.MainAxis(), TwoDimensionalViewportOpts{ScrollCacheExtent: v.scrollCacheExtent, ClipBehavior: v.ClipBehavior()})
	return w
}

func (v *tableView) verticalDetails() ScrollableDetails {
	return v.VerticalDetails()
}
func (v *tableView) horizontalDetails() ScrollableDetails {
	return v.HorizontalDetails()
}

func tableChild(vicinity ChildVicinity) Widget {
	return NewSizedBox(NewCenter(NewText(fmt.Sprintf("R%d:C%d", vicinity.XIndex, vicinity.YIndex))),
		SizedBoxOpts{Key: foundation.NewValueKey(vicinity), Width: foundation.Some(200.0), Height: foundation.Some(200.0)})
}

func tableDelegate(maxX, maxY int) *TwoDimensionalChildBuilderDelegate {
	return NewTwoDimensionalChildBuilderDelegate(func(_ BuildContext, v ChildVicinity) Widget { return tableChild(v) },
		TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(maxX), MaxYIndex: foundation.Some(maxY)})
}

func tableApp(delegate *TwoDimensionalChildBuilderDelegate, o tableOpts) Widget {
	if delegate == nil {
		delegate = tableDelegate(5, 5)
	}
	return ltr(NewAlign(newTableView(delegate, o)))
}

func tableViewportOf(ts *gluttertest.Tester) *renderTable {
	return ts.RenderObject(gluttertest.ByType[*tableViewport]()).(*renderTable)
}

func TestTwoDimensionalScrollViewAssertsAxisDirections(t *testing.T) {
	skipUnlessDebug(t)
	empty := func() *TwoDimensionalChildBuilderDelegate {
		return NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget { return nil })
	}
	tests := []struct {
		name string
		opts TwoDimensionalScrollViewOpts
	}{
		{"horizontal wrong", TwoDimensionalScrollViewOpts{HorizontalDetails: foundation.Some(NewScrollableDetailsVertical())}},
		{"vertical wrong", TwoDimensionalScrollViewOpts{VerticalDetails: foundation.Some(NewScrollableDetailsHorizontal())}},
		{"both wrong", TwoDimensionalScrollViewOpts{VerticalDetails: foundation.Some(NewScrollableDetailsHorizontal()),
			HorizontalDetails: foundation.Some(NewScrollableDetailsVertical())}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(tableApp(empty(), tableOpts{TwoDimensionalScrollViewOpts: tc.opts}))
			_ = requireErrorTag(t, takeFirst(t, ts), "TwoDimensionalScrollViewAxisInvalid")
		})
	}
}

func TestTwoDimensionalScrollViewControllersSetInitialPositions(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformLinux} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t, gluttertest.Opts{})
			vertical := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 100})
			defer vertical.Dispose()
			horizontal := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 50})
			defer horizontal.Dispose()
			ts.PumpWidget(tableApp(tableDelegate(99, 99), tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
				VerticalDetails:   foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Controller: vertical})),
				HorizontalDetails: foundation.Some(NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Controller: horizontal}))}}))
			ts.PumpAndSettle()
			check := func(step string, v, vmax, h, hmax float64) {
				t.Helper()
				if vertical.Position().Pixels() != v || vertical.Position().MaxScrollExtent() != vmax ||
					horizontal.Position().Pixels() != h || horizontal.Position().MaxScrollExtent() != hmax {
					t.Errorf("%s: vertical %v/%v, horizontal %v/%v", step, vertical.Position().Pixels(), vertical.Position().MaxScrollExtent(),
						horizontal.Position().Pixels(), horizontal.Position().MaxScrollExtent())
				}
			}
			check("initial", 100, 19400, 50, 19200)
			vertical.JumpTo(vertical.Position().MaxScrollExtent())
			horizontal.JumpTo(horizontal.Position().MaxScrollExtent())
			ts.Pump()
			check("max", 19400, 19400, 19200, 19200)
			// Out of bounds; account for the overscroll of some platforms.
			vertical.JumpTo(vertical.Position().MaxScrollExtent() + 100)
			horizontal.JumpTo(horizontal.Position().MaxScrollExtent() + 100)
			ts.PumpAndSettle()
			check("out of bounds", 19400, 19400, 19200, 19200)
		})
	}
}

func TestTwoDimensionalScrollableReceivesTheDetailsOfTheScrollView(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var captured BuildContext
	capture := NewTwoDimensionalChildBuilderDelegate(func(bc BuildContext, v ChildVicinity) Widget { captured = bc; return NewText(v.String()) })
	ts.PumpWidget(tableApp(capture, tableOpts{}))
	ts.PumpAndSettle()
	scrollable := TwoDimensionalScrollableOf(captured)
	w := scrollable.Widget()
	if w.VerticalDetails().Direction != painting.AxisDirectionDown || w.HorizontalDetails().Direction != painting.AxisDirectionRight ||
		w.DiagonalDragBehavior() != DiagonalDragBehaviorNone || w.DragStartBehavior() != gestures.DragStartBehaviorStart {
		t.Errorf("default details: %v %v %v %v", w.VerticalDetails().Direction, w.HorizontalDetails().Direction, w.DiagonalDragBehavior(), w.DragStartBehavior())
	}
	custom := NewTwoDimensionalChildBuilderDelegate(func(bc BuildContext, v ChildVicinity) Widget { captured = bc; return tableChild(v) })
	ts.PumpWidget(tableApp(custom, tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
		VerticalDetails:      foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Reverse: true})),
		HorizontalDetails:    foundation.Some(NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Reverse: true})),
		DiagonalDragBehavior: DiagonalDragBehaviorWeightedContinuous, DragStartBehavior: gestures.DragStartBehaviorDown}}))
	ts.PumpAndSettle()
	w = TwoDimensionalScrollableOf(captured).Widget()
	if w.VerticalDetails().Direction != painting.AxisDirectionUp || w.HorizontalDetails().Direction != painting.AxisDirectionLeft ||
		w.DiagonalDragBehavior() != DiagonalDragBehaviorWeightedContinuous || w.DragStartBehavior() != gestures.DragStartBehaviorDown {
		t.Errorf("custom details: %v %v %v %v", w.VerticalDetails().Direction, w.HorizontalDetails().Direction, w.DiagonalDragBehavior(), w.DragStartBehavior())
	}
}

func TestTwoDimensionalScrollableTranslucentHitTestBehavior(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	tapped := false
	key := foundation.NewValueKey("below")
	small := NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)})
	})
	ts.PumpWidget(ltr(NewStack([]Widget{
		NewPositionedFill(NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Key: key, Height: foundation.Some(300.0)}),
			GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { tapped = true }})),
		newTableView(small, tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{HitTestBehavior: rendering.HitTestBehaviorTranslucent}}),
	})))
	ts.PumpAndSettle()
	ts.TapAt(ts.Center(gluttertest.ByKey(key)))
	if !tapped {
		t.Error("the widget underneath was not tapped")
	}
}

func TestRenderTwoDimensionalViewportGetters(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	vertical, horizontal := rendering.NewViewportOffsetFixed(10), rendering.NewViewportOffsetFixed(20)
	delegate := tableDelegate(0, 0)
	r := &renderTable{}
	rendering.InitRenderTwoDimensionalViewport(&r.RenderTwoDimensionalViewport, r, horizontal, painting.AxisDirectionRight,
		vertical, painting.AxisDirectionDown, delegate, painting.AxisVertical, nil)
	defer r.Dispose()
	if r.ClipBehavior() != ui.ClipHardEdge || r.ScrollCacheExtent() != rendering.NewScrollCacheExtentPixels(rendering.RenderAbstractViewportDefaultCacheExtent) ||
		!r.IsRepaintBoundary() || !r.SizedByParent() {
		t.Error("defaults")
	}
	if r.HorizontalOffset().Pixels() != 20 || r.HorizontalAxisDirection() != painting.AxisDirectionRight ||
		r.VerticalOffset().Pixels() != 10 || r.VerticalAxisDirection() != painting.AxisDirectionDown ||
		r.Delegate() != rendering.TwoDimensionalChildDelegate(delegate) || r.MainAxis() != painting.AxisVertical {
		t.Error("getters")
	}
	// No size yet: it asserts.
	if foundation.DebugMode {
		requirePanic(t, func() { r.ViewportDimension() })
	}
	ts.PumpWidget(tableApp(nil, tableOpts{}))
	ts.PumpAndSettle()
	if got := tableViewportOf(ts).ViewportDimension(); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("viewportDimension = %v", got)
	}
}

func TestRenderTwoDimensionalViewportChildrenAreOrganizedByMainAxis(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	vicinityOf := func(b rendering.Box) ChildVicinity {
		return b.ParentData().(*TwoDimensionalViewportParentData).Vicinity
	}
	tests := []struct {
		axis                   painting.Axis
		second, last, beforeLs ChildVicinity
	}{
		{painting.AxisVertical, NewChildVicinity(1, 0), NewChildVicinity(4, 3), NewChildVicinity(3, 3)},
		{painting.AxisHorizontal, NewChildVicinity(0, 1), NewChildVicinity(4, 3), NewChildVicinity(4, 2)},
	}
	for _, tc := range tests {
		ts.PumpWidget(tableApp(tableDelegate(5, 5), tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{MainAxis: tc.axis}}))
		ts.PumpAndSettle()
		r := tableViewportOf(ts)
		if r.MainAxis() != tc.axis || vicinityOf(r.FirstChild()) != NewChildVicinity(0, 0) || vicinityOf(r.ChildAfter(r.FirstChild())) != tc.second ||
			r.ChildBefore(r.FirstChild()) != nil || vicinityOf(r.LastChild()) != tc.last || r.ChildAfter(r.LastChild()) != nil ||
			vicinityOf(r.ChildBefore(r.LastChild())) != tc.beforeLs {
			t.Errorf("main axis %v: first %v, second %v, last %v, before last %v", tc.axis, vicinityOf(r.FirstChild()),
				vicinityOf(r.ChildAfter(r.FirstChild())), vicinityOf(r.LastChild()), vicinityOf(r.ChildBefore(r.LastChild())))
		}
	}
}

func TestRenderTwoDimensionalViewportSetsUpParentData(t *testing.T) {
	// Also tests ComputeAbsolutePaintOffsetFor and ComputeChildPaintExtent.
	// Regression test for https://github.com/flutter/flutter/issues/128723
	ts := gluttertest.New(t, gluttertest.Opts{})
	type expectation struct {
		vicinity            ChildVicinity
		visible             bool
		paintOffset, layout ui.Offset
	}
	check := func(step string, parent func(b rendering.Box) *TwoDimensionalViewportParentData, r *renderTable, first, last *expectation) {
		t.Helper()
		for _, c := range []struct {
			b rendering.Box
			e *expectation
		}{{r.FirstChild(), first}, {r.LastChild(), last}} {
			if c.e == nil {
				continue
			}
			d := parent(c.b)
			if d.Vicinity != c.e.vicinity || d.IsVisible() != c.e.visible || d.PaintOffset != foundation.Some(c.e.paintOffset) ||
				d.LayoutOffset != foundation.Some(c.e.layout) {
				t.Errorf("%s: %v visible %v paint %v layout %v; want %+v", step, d.Vicinity, d.IsVisible(), d.PaintOffset, d.LayoutOffset, *c.e)
			}
		}
	}
	parent := func(b rendering.Box) *TwoDimensionalViewportParentData {
		return b.ParentData().(*TwoDimensionalViewportParentData)
	}
	reverse := func(v, h bool) tableOpts {
		return tableOpts{useCacheExtent: true, TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
			VerticalDetails:   foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Reverse: v})),
			HorizontalDetails: foundation.Some(NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Reverse: h}))}}
	}
	// The last child is in the cache extent, and is not visible.
	tests := []struct {
		name        string
		v, h        bool
		first, last expectation
	}{
		{"normal axes", false, false,
			expectation{NewChildVicinity(0, 0), true, ui.Offset{}, ui.Offset{}},
			expectation{NewChildVicinity(5, 5), false, ui.Offset{Dx: 1000, Dy: 1000}, ui.Offset{Dx: 1000, Dy: 1000}}},
		{"vertical reverse", true, false,
			expectation{NewChildVicinity(0, 0), true, ui.Offset{Dy: 400}, ui.Offset{}},
			expectation{NewChildVicinity(5, 5), false, ui.Offset{Dx: 1000, Dy: -600}, ui.Offset{Dx: 1000, Dy: 1000}}},
		{"horizontal reverse", false, true,
			expectation{NewChildVicinity(0, 0), true, ui.Offset{Dx: 600}, ui.Offset{}},
			expectation{NewChildVicinity(5, 5), false, ui.Offset{Dx: -400, Dy: 1000}, ui.Offset{Dx: 1000, Dy: 1000}}},
		{"both reverse", true, true,
			expectation{NewChildVicinity(0, 0), true, ui.Offset{Dx: 600, Dy: 400}, ui.Offset{}},
			expectation{NewChildVicinity(5, 5), false, ui.Offset{Dx: -400, Dy: -600}, ui.Offset{Dx: 1000, Dy: 1000}}},
	}
	for _, tc := range tests {
		ts.PumpWidget(tableApp(tableDelegate(5, 5), reverse(tc.v, tc.h)))
		ts.PumpAndSettle()
		check(tc.name, parent, tableViewportOf(ts), &tc.first, &tc.last)
	}
	// Scroll to test a partially visible child.
	vertical, horizontal := NewScrollController(), NewScrollController()
	defer vertical.Dispose()
	defer horizontal.Dispose()
	ts.PumpWidget(tableApp(tableDelegate(5, 5), tableOpts{useCacheExtent: true, TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
		VerticalDetails:   foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Controller: vertical})),
		HorizontalDetails: foundation.Some(NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Controller: horizontal}))}}))
	ts.PumpAndSettle()
	vertical.JumpTo(50)
	horizontal.JumpTo(50)
	ts.Pump()
	check("partially visible", parent, tableViewportOf(ts), &expectation{NewChildVicinity(0, 0), true, ui.Offset{Dx: -50, Dy: -50}, ui.Offset{Dx: -50, Dy: -50}}, nil)
}

func TestRenderTwoDimensionalViewportBoundedAxes(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("asserts that both axes are bounded", func(t *testing.T) {
		for _, name := range []string{"column", "row"} {
			ts := gluttertest.New(t, gluttertest.Opts{})
			child := Widget(newTableView(tableDelegate(5, 5), tableOpts{}))
			if name == "column" {
				ts.PumpWidget(ltr(NewColumn([]Widget{child})))
			} else {
				ts.PumpWidget(ltr(NewRow([]Widget{child})))
			}
			_ = requireErrorTag(t, takeFirst(t, ts), "ViewportUnbounded")
		}
	})
	t.Run("computeDryLayout asserts that the axes are bounded", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(tableApp(tableDelegate(0, 0), tableOpts{}))
		r := tableViewportOf(ts)
		requireTag(t, "ViewportUnbounded", func() { r.ComputeDryLayout(rendering.NewBoxConstraints()) })
	})
}

func TestRenderTwoDimensionalViewportResizes(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(tableApp(tableDelegate(0, 0), tableOpts{}))
	ts.PumpAndSettle()
	if got := tableViewportOf(ts).ViewportDimension(); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("viewportDimension = %v", got)
	}
	ts.SetViewSize(ui.Size{Width: 300, Height: 300})
	ts.PumpWidget(tableApp(tableDelegate(0, 0), tableOpts{}))
	ts.PumpAndSettle()
	if got := tableViewportOf(ts).ViewportDimension(); got != (ui.Size{Width: 300, Height: 300}) {
		t.Errorf("viewportDimension = %v", got)
	}
}

func TestRenderTwoDimensionalViewportRebuildsWhenTheDelegateChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	firstKey, newKey := foundation.NewValueKey("first"), foundation.NewValueKey("new")
	first := NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget {
		return NewSizedBoxSquare(foundation.Some(200.0), nil, WidgetOpts{Key: firstKey})
	}, TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(0), MaxYIndex: foundation.Some(0), AddRepaintBoundaries: foundation.Some(false)})
	ts.PumpWidget(tableApp(first, tableOpts{}))
	if tableViewportOf(ts).FirstChild() != ts.RenderObject(gluttertest.ByKey(firstKey)).(rendering.Box) {
		t.Fatal("the first child is not the child of the first delegate")
	}
	newDelegate := NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Key: newKey, Width: foundation.Some(300.0), Height: foundation.Some(300.0)})
	}, TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(0), MaxYIndex: foundation.Some(0), AddRepaintBoundaries: foundation.Some(false)})
	ts.PumpWidget(tableApp(newDelegate, tableOpts{}))
	if ts.Count(gluttertest.ByKey(firstKey)) != 0 || ts.Count(gluttertest.ByKey(newKey)) != 1 ||
		tableViewportOf(ts).FirstChild() != ts.RenderObject(gluttertest.ByKey(newKey)).(rendering.Box) {
		t.Error("the delegate did not replace the child")
	}
}

func TestRenderTwoDimensionalViewportHitTestChildren(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		t.Run(fmt.Sprintf("reversed %v", reversed), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var taps []ChildVicinity
			keyOf := func(v ChildVicinity) foundation.Key { return foundation.NewValueKey(v) }
			delegate := NewTwoDimensionalChildBuilderDelegate(func(_ BuildContext, v ChildVicinity) Widget {
				return NewSizedBoxSquare(foundation.Some(200.0), NewCenter(NewGestureDetector(NewSizedBoxSquare(foundation.Some(56.0), nil),
					GestureDetectorOpts{Key: keyOf(v), Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { taps = append(taps, v) }})))
			}, TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(19), MaxYIndex: foundation.Some(19)})
			ts.PumpWidget(tableApp(delegate, tableOpts{useCacheExtent: true, TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
				VerticalDetails:   foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Reverse: reversed})),
				HorizontalDetails: foundation.Some(NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Reverse: reversed}))}}))
			ts.PumpAndSettle()
			tapped := func(v ChildVicinity) bool {
				for _, t := range taps {
					if t == v {
						return true
					}
				}
				return false
			}
			at := ui.Offset{Dx: 100, Dy: 100}
			if reversed {
				at = ui.Offset{Dx: 700, Dy: 500}
			}
			ts.TapAt(at)
			ts.Pump()
			if !tapped(NewChildVicinity(0, 0)) {
				t.Error("the first child was not tapped")
			}
			ts.Tap(gluttertest.ByKey(keyOf(NewChildVicinity(2, 2))))
			ts.Pump()
			if !tapped(NewChildVicinity(2, 2)) {
				t.Error("child 2:2 was not tapped")
			}
			// The children in the cache extent are not tappable.
			ts.Tap(gluttertest.ByKey(keyOf(NewChildVicinity(5, 5))))
			ts.Pump()
			if tapped(NewChildVicinity(5, 5)) {
				t.Error("child 5:5 was tapped out of bounds")
			}
		})
	}
}

func TestRenderTwoDimensionalViewportGetChildFor(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(tableApp(tableDelegate(5, 5), tableOpts{}))
	ts.PumpAndSettle()
	r := tableViewportOf(ts)
	if r.GetChildFor(NewChildVicinity(0, 0)) == nil || r.GetChildFor(NewChildVicinity(0, 0)) != r.FirstChild() {
		t.Error("0:0 is not the first child")
	}
	if r.GetChildFor(NewChildVicinity(10, 10)) != nil {
		t.Error("10:10 has a child")
	}
}

func TestRenderTwoDimensionalViewportAssertions(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("the vicinity is valid when children are asked to build", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(tableApp(tableDelegate(5, 5), tableOpts{}))
		ts.PumpAndSettle()
		requireTag(t, "ChildVicinityInvalid", func() { tableViewportOf(ts).BuildOrObtainChildFor(rendering.ChildVicinityInvalid) })
	})
	tests := []struct {
		name string
		opts tableOpts
		tag  string
	}{
		{"content dimensions have been applied", tableOpts{skipApplyDimensions: true}, "TwoDimensionalViewportContentDimensions"},
		{"the layoutOffset has been set by the subclass", tableOpts{skipLayoutOffset: true}, "TwoDimensionalViewportLayoutOffsetMissing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(tableApp(tableDelegate(5, 5), tc.opts))
			_ = requireErrorTag(t, takeFirst(t, ts), tc.tag)
		})
	}
	t.Run("the children have a size after layoutChildSequence", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(tableApp(tableDelegate(5, 5), tableOpts{forgetToLayoutChild: true}))
		if errs := ts.TakeErrors(); len(errs) == 0 {
			t.Error("no error for children that were not laid out")
		}
	})
}

func TestRenderTwoDimensionalViewportDoesNotRebuildAChildThatCanBeReused(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var built []ChildVicinity
	controller := NewScrollController()
	defer controller.Dispose()
	delegate := NewTwoDimensionalChildBuilderDelegate(func(_ BuildContext, v ChildVicinity) Widget {
		built = append(built, v)
		return NewSizedBoxSquare(foundation.Some(200.0), nil)
	}, TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(5), MaxYIndex: foundation.Some(5)})
	ts.PumpWidget(tableApp(delegate, tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
		VerticalDetails: foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Controller: controller}))}}))
	if controller.Position().Pixels() != 0 || len(built) != 20 || built[0] != NewChildVicinity(0, 0) {
		t.Fatalf("built %d children, first %v", len(built), built[0])
	}
	built = nil
	controller.JumpTo(1) // Move slightly to trigger another layout.
	ts.Pump()
	if controller.Position().Pixels() != 1 || len(built) != 5 {
		t.Errorf("built %d children, want the next row of 5", len(built))
	}
	for _, v := range built {
		if v == NewChildVicinity(0, 0) {
			t.Error("a child of the first layout was rebuilt")
		}
	}
}
