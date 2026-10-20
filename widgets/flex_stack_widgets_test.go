// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/: row_test.dart, column_test.dart,
// flex_test.dart, expanded_test.dart (parent data updates), spacer (flex_test),
// stack_test.dart, positioned_test (stack_test), indexed_stack_test.dart,
// wrap_test.dart, list_body_test.dart and table_test.dart. The tests check the
// position of each box; the algorithms have their own tests (steps 6.x).

// rectsOf are the global rects of the boxes in the tree, in tree order.
func rectsOf(ts *gluttertest.Tester) []ui.Rect {
	var out []ui.Rect
	for _, e := range ts.Elements(anyBox) {
		f := gluttertest.ByElementPredicate("this one", func(x Element) bool { return x == e })
		tl := ts.TopLeft(f)
		sz := ts.Size(f)
		out = append(out, ui.NewRectFromLTWH(tl.Dx, tl.Dy, sz.Width, sz.Height))
	}
	return out
}

func rectsEqual(got, want []ui.Rect) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		for _, p := range [][2]float64{{got[i].Left, want[i].Left}, {got[i].Top, want[i].Top}, {got[i].Right, want[i].Right}, {got[i].Bottom, want[i].Bottom}} {
			if math.Abs(p[0]-p[1]) > 1e-3 {
				return false
			}
		}
	}
	return true
}

func rect(l, t, w, h float64) ui.Rect { return ui.NewRectFromLTWH(l, t, w, h) }

func boxes3() []Widget { return []Widget{newBox(100, 50), newBox(100, 50), newBox(100, 50)} }

func TestRowMainAxisAlignment(t *testing.T) {
	tests := []struct {
		name  string
		opts  RowOpts
		dir   ui.TextDirection
		wantX []float64
	}{
		{"start", RowOpts{}, ui.TextDirectionLTR, []float64{0, 100, 200}},
		{"end", RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd}, ui.TextDirectionLTR, []float64{500, 600, 700}},
		{"center", RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}, ui.TextDirectionLTR, []float64{250, 350, 450}},
		{"space between", RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween}, ui.TextDirectionLTR, []float64{0, 350, 700}},
		{"space around", RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceAround}, ui.TextDirectionLTR, []float64{500.0 / 6, 350, 500.0/6 + 2*(100+500.0/3)}},
		{"space evenly", RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceEvenly}, ui.TextDirectionLTR, []float64{125, 350, 575}},
		{"start in RTL", RowOpts{}, ui.TextDirectionRTL, []float64{700, 600, 500}},
		{"spacing", RowOpts{Spacing: 10}, ui.TextDirectionLTR, []float64{0, 110, 220}},
		{"min size", RowOpts{MainAxisSize: rendering.MainAxisSizeMin, MainAxisAlignment: rendering.MainAxisAlignmentEnd}, ui.TextDirectionLTR, []float64{0, 100, 200}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewRow(boxes3(), tt.opts), tt.dir)
			var want []ui.Rect
			for _, x := range tt.wantX {
				want = append(want, rect(x, 0, 100, 50))
			}
			if got := rectsOf(ts); !rectsEqual(got, want) {
				t.Errorf("boxes = %v, want %v", got, want)
			}
		})
	}
}

func TestColumnAlignmentsAndDirections(t *testing.T) {
	tests := []struct {
		name string
		opts ColumnOpts
		dir  ui.TextDirection
		want []ui.Rect
	}{
		{"start", ColumnOpts{}, ui.TextDirectionLTR, []ui.Rect{rect(0, 0, 100, 50), rect(0, 50, 100, 50), rect(0, 100, 100, 50)}},
		{"end", ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 450, 100, 50), rect(0, 500, 100, 50), rect(0, 550, 100, 50)}},
		{"up", ColumnOpts{VerticalDirection: painting.VerticalDirectionUp}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 550, 100, 50), rect(0, 500, 100, 50), rect(0, 450, 100, 50)}},
		{"cross start in LTR", ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart, MainAxisSize: rendering.MainAxisSizeMin}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(0, 50, 100, 50), rect(0, 100, 100, 50)}},
		{"cross end", ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentEnd, MainAxisSize: rendering.MainAxisSizeMin}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(0, 50, 100, 50), rect(0, 100, 100, 50)}},
		{"cross stretch", ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch, MainAxisSize: rendering.MainAxisSizeMin}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 800, 50), rect(0, 50, 800, 50), rect(0, 100, 800, 50)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewColumn(boxes3(), tt.opts), tt.dir)
			if got := rectsOf(ts); !rectsEqual(got, tt.want) {
				t.Errorf("boxes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlexibleChildren(t *testing.T) {
	tests := []struct {
		name     string
		children []Widget
		wantW    []float64 // widths of the three boxes
		wantX    []float64
	}{
		{"expanded 1:2", []Widget{NewExpanded(newBox(10, 50)), NewExpanded(newBox(10, 50), ExpandedOpts{Flex: foundation.Some(2)})}, []float64{800.0 / 3, 1600.0 / 3}, []float64{0, 800.0 / 3}},
		{"fixed box and expanded", []Widget{newBox(200, 50), NewExpanded(newBox(10, 50)), NewExpanded(newBox(10, 50))}, []float64{200, 300, 300}, []float64{0, 200, 500}},
		{"flexible loose keeps the size", []Widget{NewFlexible(newBox(100, 50)), NewFlexible(newBox(100, 50), FlexibleOpts{Fit: rendering.FlexFitTight})}, []float64{100, 400}, []float64{0, 100}},
		{"flex zero is not flexible", []Widget{NewExpanded(newBox(70, 50), ExpandedOpts{Flex: foundation.Some(0)}), NewExpanded(newBox(10, 50))}, []float64{70, 730}, []float64{0, 70}},
		{"spacer", []Widget{newBox(100, 50), NewSpacer(), newBox(100, 50)}, []float64{100, 100}, []float64{0, 700}},
		{"spacer with flex", []Widget{newBox(100, 50), NewSpacer(SpacerOpts{Flex: foundation.Some(3)}), newBox(100, 50), NewSpacer()}, []float64{100, 100}, []float64{0, 100 + 600*0.75}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewRow(tt.children), ui.TextDirectionLTR)
			got := rectsOf(ts)
			if len(got) != len(tt.wantW) {
				t.Fatalf("boxes = %v", got)
			}
			for i := range got {
				if math.Abs(got[i].Width()-tt.wantW[i]) > 1e-3 || math.Abs(got[i].Left-tt.wantX[i]) > 1e-3 {
					t.Errorf("box %d = %v, want x %v width %v", i, got[i], tt.wantX[i], tt.wantW[i])
				}
			}
		})
	}
}

func TestFlexibleParentDataUpdates(t *testing.T) {
	ts := gluttertest.New(t)
	build := func(flex int, fit rendering.FlexFit) Widget {
		return NewRow([]Widget{NewFlexible(newBox(10, 50), FlexibleOpts{Flex: foundation.Some(flex), Fit: fit}), newBox(100, 50)})
	}
	pumpLoose(ts, build(1, rendering.FlexFitTight), ui.TextDirectionLTR)
	if got := rectsOf(ts)[0]; got.Width() != 700 {
		t.Fatalf("width = %v, want 700", got.Width())
	}
	pumpLoose(ts, build(1, rendering.FlexFitLoose), ui.TextDirectionLTR)
	if got := rectsOf(ts)[0]; got.Width() != 10 {
		t.Errorf("width after the fit changed = %v, want 10", got.Width())
	}
	pumpLoose(ts, NewRow([]Widget{NewFlexible(newBox(10, 50), FlexibleOpts{Fit: rendering.FlexFitTight}),
		NewFlexible(newBox(10, 50), FlexibleOpts{Flex: foundation.Some(3), Fit: rendering.FlexFitTight})}), ui.TextDirectionLTR)
	if got := rectsOf(ts); got[0].Width() != 200 || got[1].Width() != 600 {
		t.Errorf("widths = %v", got)
	}
}

func TestFlexMisuse(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("a row needs a text direction", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewAlign(NewRow(boxes3())))
		_ = requireErrorTag(t, takeFirst(t, ts), "FlexNeedsTextDirection")
	})
	t.Run("a column does not", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewAlign(NewColumn(boxes3())))
	})
	t.Run("baseline needs a text baseline", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewRow(boxes3(), RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentBaseline}))))
		_ = requireErrorTag(t, takeFirst(t, ts), "FlexNeedsTextBaseline")
	})
	t.Run("spacer flex", func(t *testing.T) {
		requireTag(t, "SpacerFlexInvalid", func() { NewSpacer(SpacerOpts{Flex: foundation.Some(0)}) })
	})
	t.Run("flexible outside a flex", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewFlexible(newBox(1, 1)))))
		_ = requireErrorTag(t, takeFirst(t, ts), "ParentDataWidgetIncorrectUse")
	})
}

func TestStackPositioned(t *testing.T) {
	big := func() Widget { return newBox(200, 100) }
	tests := []struct {
		name  string
		stack *Stack
		want  []ui.Rect // the big box, then the positioned one
	}{
		{"left top width height", NewStack([]Widget{big(), NewPositioned(newBox(1, 1), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(20.0), Width: foundation.Some(30.0), Height: foundation.Some(40.0)})}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(10, 20, 30, 40)}},
		{"right bottom", NewStack([]Widget{big(), NewPositioned(newBox(1, 1), PositionedOpts{Right: foundation.Some(10.0), Bottom: foundation.Some(20.0), Width: foundation.Some(30.0), Height: foundation.Some(40.0)})}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(160, 40, 30, 40)}},
		{"all four edges", NewStack([]Widget{big(), NewPositioned(newBox(1, 1), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(10.0), Right: foundation.Some(10.0), Bottom: foundation.Some(10.0)})}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(10, 10, 180, 80)}},
		{"fill", NewStack([]Widget{big(), NewPositionedFill(newBox(1, 1))}), []ui.Rect{rect(0, 0, 200, 100), rect(0, 0, 200, 100)}},
		{"fill with distances", NewStack([]Widget{big(), NewPositionedFill(newBox(1, 1), PositionedFillOpts{Left: 5, Top: 6, Right: 7, Bottom: 8})}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(5, 6, 188, 86)}},
		{"from rect", NewStack([]Widget{big(), NewPositionedFromRect(ui.NewRectFromLTWH(1, 2, 3, 4), newBox(1, 1))}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(1, 2, 3, 4)}},
		{"from relative rect", NewStack([]Widget{big(), NewPositionedFromRelativeRect(rendering.RelativeRect{Left: 1, Top: 2, Right: 3, Bottom: 4}, newBox(1, 1))}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(1, 2, 196, 94)}},
		{"directional in LTR", NewStack([]Widget{big(), NewPositionedDirectional(newBox(1, 1), PositionedDirectionalOpts{Start: foundation.Some(10.0), Top: foundation.Some(0.0), Width: foundation.Some(30.0), Height: foundation.Some(40.0)})}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(10, 0, 30, 40)}},
		{"alignment bottom right", NewStack([]Widget{big(), newBox(20, 10)}, StackOpts{Alignment: painting.AlignmentBottomRight}),
			[]ui.Rect{rect(0, 0, 200, 100), rect(180, 90, 20, 10)}},
		{"fit expand", NewStack([]Widget{big(), newBox(20, 10)}, StackOpts{Fit: rendering.StackFitExpand}),
			[]ui.Rect{rect(0, 0, 800, 600), rect(0, 0, 800, 600)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.stack, ui.TextDirectionLTR)
			if got := rectsOf(ts); !rectsEqual(got, tt.want) {
				t.Errorf("boxes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPositionedDirectionalInRTL(t *testing.T) {
	ts := gluttertest.New(t)
	pumpLoose(ts, NewStack([]Widget{newBox(200, 100), NewPositionedDirectional(newBox(1, 1),
		PositionedDirectionalOpts{Start: foundation.Some(10.0), Top: foundation.Some(0.0), Width: foundation.Some(30.0), Height: foundation.Some(40.0)})},
		StackOpts{TextDirection: ui.TextDirectionRTL}), ui.TextDirectionRTL)
	if got := rectsOf(ts)[1]; got != rect(160, 0, 30, 40) {
		t.Errorf("box = %v, want it 10 from the right edge", got)
	}
}

func TestPositionedUpdates(t *testing.T) {
	ts := gluttertest.New(t)
	build := func(left float64) Widget {
		return NewStack([]Widget{newBox(200, 100), NewPositioned(newBox(1, 1), PositionedOpts{Left: foundation.Some(left), Top: foundation.Some(0.0), Width: foundation.Some(10.0), Height: foundation.Some(10.0)})})
	}
	pumpLoose(ts, build(5), ui.TextDirectionLTR)
	pumpLoose(ts, build(50), ui.TextDirectionLTR)
	if got := rectsOf(ts)[1]; got != rect(50, 0, 10, 10) {
		t.Errorf("box = %v, want it moved to 50", got)
	}
}

func TestPositionedMisuse(t *testing.T) {
	skipUnlessDebug(t)
	f := foundation.Some(1.0)
	tests := []struct {
		name string
		do   func()
	}{
		{"left right width", func() { NewPositioned(nil, PositionedOpts{Left: f, Right: f, Width: f}) }},
		{"top bottom height", func() { NewPositioned(nil, PositionedOpts{Top: f, Bottom: f, Height: f}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, "PositionedOverconstrained", tt.do) })
	}
	t.Run("outside a stack", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewPositioned(newBox(1, 1), PositionedOpts{Left: f}))))
		_ = requireErrorTag(t, takeFirst(t, ts), "ParentDataWidgetIncorrectUse")
	})
	t.Run("a directional stack needs a direction", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewAlign(NewStack([]Widget{newBox(1, 1)})))
		_ = requireErrorTag(t, takeFirst(t, ts), "DirectionalityMissing")
	})
}

func TestIndexedStack(t *testing.T) {
	ts := gluttertest.New(t)
	var visible [3]bool
	child := func(i int) Widget {
		return builderOf(func(bc BuildContext) Widget { visible[i] = VisibilityOf(bc); return newBox(10*float64(i+1), 10) })
	}
	pumpLoose(ts, NewIndexedStack([]Widget{child(0), child(1), child(2)}, IndexedStackOpts{Index: foundation.Some(1)}), ui.TextDirectionLTR)
	if visible != [3]bool{false, true, false} {
		t.Errorf("visible = %v, want only the second", visible)
	}
	if got := ts.Size(gluttertest.ByType[*IndexedStack]()); got != (ui.Size{Width: 30, Height: 10}) {
		t.Errorf("size = %v, want the one of the biggest child", got)
	}
	indexed := ts.Elements(gluttertest.ByElementPredicate("a render indexed stack", func(e Element) bool {
		_, ok := e.RenderObject().(*rendering.RenderIndexedStack)
		return ok
	}))
	if len(indexed) == 0 {
		t.Fatal("no RenderIndexedStack")
	}
	ro := indexed[len(indexed)-1].RenderObject().(*rendering.RenderIndexedStack)
	if i, _ := ro.Index().Get(); i != 1 {
		t.Errorf("index = %v", ro.Index())
	}
	// Only the shown child is onstage.
	onstage := 0
	ts.Elements(gluttertest.ByType[*IndexedStack]())[0].DebugVisitOnstageChildren(func(Element) { onstage++ })
	if onstage != 1 {
		t.Errorf("%d onstage children of the IndexedStack, want 1", onstage)
	}
	pumpLoose(ts, NewIndexedStack([]Widget{child(0), child(1), child(2)}, IndexedStackOpts{NoIndex: true}), ui.TextDirectionLTR)
	if visible != [3]bool{} {
		t.Errorf("visible = %v, want none", visible)
	}
}

func TestIndexedStackIndexOutOfRange(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewIndexedStack([]Widget{newBox(1, 1)}, IndexedStackOpts{Index: foundation.Some(3)}))))
	_ = requireErrorTag(t, takeFirst(t, ts), "IndexedStackIndexInvalid")
}

func TestWrap(t *testing.T) {
	five := func() []Widget {
		return []Widget{newBox(100, 50), newBox(100, 50), newBox(100, 50), newBox(100, 50), newBox(100, 50)}
	}
	tests := []struct {
		name string
		opts WrapOpts
		dir  ui.TextDirection
		want []ui.Rect
	}{
		{"runs of two", WrapOpts{}, ui.TextDirectionLTR, []ui.Rect{rect(0, 0, 100, 50), rect(100, 0, 100, 50), rect(0, 50, 100, 50), rect(100, 50, 100, 50), rect(0, 100, 100, 50)}},
		{"spacing and run spacing", WrapOpts{Spacing: 10, RunSpacing: 5}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(110, 0, 100, 50), rect(0, 55, 100, 50), rect(110, 55, 100, 50), rect(0, 110, 100, 50)}},
		{"RTL", WrapOpts{}, ui.TextDirectionRTL, []ui.Rect{rect(150, 0, 100, 50), rect(50, 0, 100, 50), rect(150, 50, 100, 50), rect(50, 50, 100, 50), rect(150, 100, 100, 50)}},
		{"vertical", WrapOpts{Direction: painting.AxisVertical}, ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(0, 50, 100, 50), rect(0, 100, 100, 50), rect(0, 150, 100, 50), rect(0, 200, 100, 50)}},
		{"center alignment", WrapOpts{Alignment: rendering.WrapAlignmentCenter}, ui.TextDirectionLTR,
			[]ui.Rect{rect(25, 0, 100, 50), rect(125, 0, 100, 50), rect(25, 50, 100, 50), rect(125, 50, 100, 50), rect(75, 100, 100, 50)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(tt.dir, NewAlign(NewSizedBox(NewWrap(five(), tt.opts),
				SizedBoxOpts{Width: foundation.Some(250.0), Height: foundation.Some(600.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})))
			got := rectsOf(ts)
			if tt.dir == ui.TextDirectionRTL {
				// The sized box is at the left of the view: shift is not needed.
				_ = got
			}
			if !rectsEqual(got, tt.want) {
				t.Errorf("boxes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListBody(t *testing.T) {
	tests := []struct {
		name string
		w    *ListBody
		dir  ui.TextDirection
		want []ui.Rect
	}{
		{"vertical", NewListBody([]Widget{newBox(10, 50), newBox(10, 30)}), ui.TextDirectionLTR, []ui.Rect{rect(0, 0, 800, 50), rect(0, 50, 800, 30)}},
		{"vertical reversed", NewListBody([]Widget{newBox(10, 50), newBox(10, 30)}, ListBodyOpts{Reverse: true}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 30, 800, 50), rect(0, 0, 800, 30)}},
		{"horizontal LTR", NewListBody([]Widget{newBox(50, 10), newBox(30, 10)}, ListBodyOpts{MainAxis: painting.AxisHorizontal}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 50, 600), rect(50, 0, 30, 600)}},
		{"horizontal RTL", NewListBody([]Widget{newBox(50, 10), newBox(30, 10)}, ListBodyOpts{MainAxis: painting.AxisHorizontal}), ui.TextDirectionRTL,
			[]ui.Rect{rect(30, 0, 50, 600), rect(0, 0, 30, 600)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			axis := painting.AxisHorizontal // the axis whose constraints the unconstrained box keeps
			if tt.w.MainAxis() == painting.AxisHorizontal {
				axis = painting.AxisVertical
			}
			pumpLoose(ts, NewUnconstrainedBox(tt.w, UnconstrainedBoxOpts{ConstrainedAxis: axis}), tt.dir)
			if got := rectsOf(ts); !rectsEqual(got, tt.want) {
				t.Errorf("boxes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable(t *testing.T) {
	cell := func(h float64) Widget { return newBox(10, h) }
	tests := []struct {
		name  string
		table *Table
		dir   ui.TextDirection
		want  []ui.Rect
	}{
		{"fixed columns", NewTable([]TableRow{NewTableRow([]Widget{cell(50), cell(50)}), NewTableRow([]Widget{cell(30), cell(30)})},
			TableOpts{ColumnWidths: map[int]rendering.TableColumnWidth{0: rendering.NewFixedColumnWidth(100), 1: rendering.NewFixedColumnWidth(200)}}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(100, 0, 200, 50), rect(0, 50, 100, 30), rect(100, 50, 200, 30)}},
		{"default flex columns", NewTable([]TableRow{NewTableRow([]Widget{cell(50), cell(50)})}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 400, 50), rect(400, 0, 400, 50)}},
		{"RTL", NewTable([]TableRow{NewTableRow([]Widget{cell(50), cell(50)})}, TableOpts{DefaultColumnWidth: rendering.NewFixedColumnWidth(100)}), ui.TextDirectionRTL,
			[]ui.Rect{rect(100, 0, 100, 50), rect(0, 0, 100, 50)}},
		{"cell alignment", NewTable([]TableRow{NewTableRow([]Widget{cell(50), NewTableCell(newAlignedProbe(), TableCellOpts{VerticalAlignment: rendering.TableCellVerticalAlignmentBottom})})},
			TableOpts{DefaultColumnWidth: rendering.NewFixedColumnWidth(100)}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(100, 30, 100, 20)}},
		{"middle alignment by default", NewTable([]TableRow{NewTableRow([]Widget{cell(50), newAlignedProbe()})},
			TableOpts{DefaultColumnWidth: rendering.NewFixedColumnWidth(100), DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentMiddle}), ui.TextDirectionLTR,
			[]ui.Rect{rect(0, 0, 100, 50), rect(100, 15, 100, 20)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.table, tt.dir)
			if got := rectsOf(ts); !rectsEqual(got, tt.want) {
				t.Errorf("boxes = %v, want %v", got, tt.want)
			}
		})
	}
}

// newAlignedProbe is a box that is 20 tall whatever the column width is.
func newAlignedProbe() Widget { return newBox(10, 20) }

func TestTableRowsAreKeptByKey(t *testing.T) {
	ts := gluttertest.New(t)
	ka, kb := foundation.NewKey("a"), foundation.NewKey("b")
	build := func(order ...foundation.Key) Widget {
		var rows []TableRow
		for _, k := range order {
			h := 20.0
			if k == kb {
				h = 40
			}
			rows = append(rows, NewTableRow([]Widget{newBox(10, h)}, TableRowOpts{Key: k}))
		}
		return NewTable(rows)
	}
	pumpLoose(ts, build(ka, kb), ui.TextDirectionLTR)
	first := ts.Elements(anyBox)
	pumpLoose(ts, build(kb, ka), ui.TextDirectionLTR)
	now := ts.Elements(anyBox)
	if len(now) != 2 || now[0] != first[1] || now[1] != first[0] {
		t.Errorf("the elements of the rows were not kept when the rows moved")
	}
	if got := rectsOf(ts); !rectsEqual(got, []ui.Rect{rect(0, 0, 800, 40), rect(0, 40, 800, 20)}) {
		t.Errorf("boxes = %v", got)
	}
	pumpLoose(ts, build(ka), ui.TextDirectionLTR)
	if got := rectsOf(ts); len(got) != 1 {
		t.Errorf("boxes after a row was removed = %v", got)
	}
}

func TestTableMisuse(t *testing.T) {
	skipUnlessDebug(t)
	one := func() []Widget { return []Widget{newBox(1, 1)} }
	ka := foundation.NewKey("a")
	tests := []struct {
		name string
		do   func()
		tag  string
	}{
		{"irregular rows", func() { NewTable([]TableRow{NewTableRow(one()), NewTableRow(append(one(), newBox(1, 1)))}) }, "TableRowsIrregular"},
		{"empty row", func() { NewTable([]TableRow{NewTableRow(nil)}) }, "TableRowEmpty"},
		{"baseline without a text baseline", func() {
			NewTable(nil, TableOpts{DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentBaseline})
		}, "TableBaselineMissing"},
		{"duplicate row keys", func() {
			NewTable([]TableRow{NewTableRow(one(), TableRowOpts{Key: ka}), NewTableRow(one(), TableRowOpts{Key: ka})})
		}, "TableRowKeysDuplicate"},
		{"duplicate cell keys", func() {
			NewTable([]TableRow{NewTableRow([]Widget{newBox(1, 1, WidgetOpts{Key: ka}), newBox(1, 1, WidgetOpts{Key: ka})})})
		}, "ChildrenHaveDuplicateKeys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.do) })
	}
}

func TestTableRowString(t *testing.T) {
	tests := []struct {
		name string
		row  TableRow
		want string
	}{
		{"no children", NewTableRow(nil), "TableRow(no children)"},
		{"key", NewTableRow(nil, TableRowOpts{Key: foundation.NewKey("k")}), "TableRow([<'k'>], no children)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.row.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// diagonalFlow paints child i at (10i, 10i).
type diagonalFlow struct {
	rendering.FlowDelegateBase
	size ui.Size
}

func (d *diagonalFlow) GetSize(c rendering.BoxConstraints) ui.Size { return c.Constrain(d.size) }
func (d *diagonalFlow) GetConstraintsForChild(_ int, c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.Loosen()
}
func (d *diagonalFlow) PaintChildren(ctx rendering.FlowPaintingContext) {
	for i := 0; i < ctx.ChildCount(); i++ {
		ctx.PaintChildAt(i, rendering.FlowPaintChildOpts{Transform: ui.NewMatrix4TranslationValues(float32(10*i), float32(10*i), 0)})
	}
}
func (d *diagonalFlow) ShouldRelayout(rendering.FlowDelegate) bool { return true }
func (d *diagonalFlow) ShouldRepaint(rendering.FlowDelegate) bool  { return true }

func TestFlow(t *testing.T) {
	ts := gluttertest.New(t)
	d := &diagonalFlow{FlowDelegateBase: rendering.NewFlowDelegateBase(nil), size: ui.Size{Width: 100, Height: 80}}
	w := NewFlow(d, []Widget{newBox(20, 20), newBox(20, 20, WidgetOpts{Key: foundation.NewKey("k")}), newBox(30, 10)})
	pumpLoose(ts, w, ui.TextDirectionLTR)
	if got := ts.Size(gluttertest.ByWidget(w)); got != d.size {
		t.Errorf("size = %v, want the one of the delegate", got)
	}
	// The children are laid out where the flow puts them and painted by the delegate.
	if got := rectsOf(ts); !rectsEqual(got, []ui.Rect{rect(0, 0, 20, 20), rect(10, 10, 20, 20), rect(20, 20, 30, 10)}) {
		t.Errorf("boxes = %v", got)
	}
	// Flow wraps its children in repaint boundaries keyed like RepaintBoundary.wrapAll.
	if n := ts.Count(gluttertest.ByType[*RepaintBoundary]()); n != 3 {
		t.Errorf("%d repaint boundaries, want one per child", n)
	}
	ro := ts.RenderObject(gluttertest.ByWidget(w)).(*rendering.RenderFlow)
	d2 := &diagonalFlow{FlowDelegateBase: rendering.NewFlowDelegateBase(nil), size: ui.Size{Width: 50, Height: 50}}
	pumpLoose(ts, NewFlow(d2, []Widget{newBox(20, 20), newBox(20, 20, WidgetOpts{Key: foundation.NewKey("k")}), newBox(30, 10)}), ui.TextDirectionLTR)
	if ts.RenderObject(gluttertest.ByType[*Flow]()) != rendering.RenderObject(ro) {
		t.Errorf("the render object was replaced")
	}
	if got := ts.Size(gluttertest.ByType[*Flow]()); got != d2.size {
		t.Errorf("size after the delegate changed = %v", got)
	}
}
