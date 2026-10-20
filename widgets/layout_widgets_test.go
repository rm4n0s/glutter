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

// Ported from packages/flutter/test/widgets/: padding (basic_test.dart),
// align_test.dart, center_test.dart, sized_box_test.dart, basic_test.dart
// (ConstrainedBox, LimitedBox, OverflowBox, FractionallySizedBox),
// fitted_box_test.dart, aspect_ratio_test.dart, offstage_test.dart,
// unconstrained_box_test.dart, rotated_box_test.dart. The tests check the size of
// the widget and the position of the box inside it; the render objects have their
// own tests (steps 6.x).

var anyBox = gluttertest.ByType[*box](gluttertest.FindOpts{IncludeOffstage: true})

// pumpLoose shows the widget at the top left of the view, with loose constraints.
func pumpLoose(ts *gluttertest.Tester, w Widget, dir ui.TextDirection) {
	ts.PumpWidget(NewDirectionality(dir, NewAlign(w, AlignOpts{Alignment: painting.AlignmentTopLeft})))
}

func TestLayoutWidgets(t *testing.T) {
	tests := []struct {
		name      string
		w         Widget
		dir       ui.TextDirection
		wantSize  ui.Size
		wantChild ui.Offset // top left of the box
		wantBox   ui.Size   // size of the box
	}{
		{"padding all", NewPadding(painting.NewEdgeInsetsAll(10), newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 120, Height: 70}, ui.Offset{Dx: 10, Dy: 10}, ui.Size{Width: 100, Height: 50}},
		{"padding LTRB", NewPadding(painting.NewEdgeInsetsFromLTRB(1, 2, 3, 4), newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 104, Height: 56}, ui.Offset{Dx: 1, Dy: 2}, ui.Size{Width: 100, Height: 50}},
		{"padding directional LTR", NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(10, 0, 0, 0), newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 110, Height: 50}, ui.Offset{Dx: 10}, ui.Size{Width: 100, Height: 50}},
		{"padding directional RTL", NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(10, 0, 0, 0), newBox(100, 50)), ui.TextDirectionRTL,
			ui.Size{Width: 110, Height: 50}, ui.Offset{Dx: 0}, ui.Size{Width: 100, Height: 50}},
		{"align center", NewAlign(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 600}, ui.Offset{Dx: 350, Dy: 275}, ui.Size{Width: 100, Height: 50}},
		{"align bottom right", NewAlign(newBox(100, 50), AlignOpts{Alignment: painting.AlignmentBottomRight}), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 600}, ui.Offset{Dx: 700, Dy: 550}, ui.Size{Width: 100, Height: 50}},
		{"align directional start in RTL", NewAlign(newBox(100, 50), AlignOpts{Alignment: painting.AlignmentDirectionalTopStart}), ui.TextDirectionRTL,
			ui.Size{Width: 800, Height: 600}, ui.Offset{Dx: 700, Dy: 0}, ui.Size{Width: 100, Height: 50}},
		{"align with factors", NewAlign(newBox(100, 50), AlignOpts{WidthFactor: foundation.Some(2.0), HeightFactor: foundation.Some(3.0)}), ui.TextDirectionLTR,
			ui.Size{Width: 200, Height: 150}, ui.Offset{Dx: 50, Dy: 50}, ui.Size{Width: 100, Height: 50}},
		{"center", NewCenter(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 600}, ui.Offset{Dx: 350, Dy: 275}, ui.Size{Width: 100, Height: 50}},
		{"center with factors", NewCenter(newBox(100, 50), CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}), ui.TextDirectionLTR,
			ui.Size{Width: 100, Height: 50}, ui.Offset{}, ui.Size{Width: 100, Height: 50}},
		{"sized box", NewSizedBox(newBox(100, 50), SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(40.0)}), ui.TextDirectionLTR,
			ui.Size{Width: 30, Height: 40}, ui.Offset{}, ui.Size{Width: 30, Height: 40}},
		{"sized box with a width only", NewSizedBox(newBox(100, 50), SizedBoxOpts{Width: foundation.Some(30.0)}), ui.TextDirectionLTR,
			ui.Size{Width: 30, Height: 50}, ui.Offset{}, ui.Size{Width: 30, Height: 50}},
		{"sized box expand", NewSizedBoxExpand(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 600}, ui.Offset{}, ui.Size{Width: 800, Height: 600}},
		{"sized box shrink", NewSizedBoxShrink(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{}, ui.Offset{}, ui.Size{}},
		{"sized box from size", NewSizedBoxFromSize(ui.Size{Width: 20, Height: 10}, newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 20, Height: 10}, ui.Offset{}, ui.Size{Width: 20, Height: 10}},
		{"sized box square", NewSizedBoxSquare(foundation.Some(25.0), newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 25, Height: 25}, ui.Offset{}, ui.Size{Width: 25, Height: 25}},
		{"constrained box min width", NewConstrainedBox(rendering.BoxConstraints{MinWidth: 200, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}, newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 200, Height: 50}, ui.Offset{}, ui.Size{Width: 200, Height: 50}},
		{"limited box in an unconstrained box", NewUnconstrainedBox(NewLimitedBox(newBox(100, 50),
			LimitedBoxOpts{MaxWidth: foundation.Some(30.0), MaxHeight: foundation.Some(40.0)})), ui.TextDirectionLTR,
			ui.Size{Width: 30, Height: 40}, ui.Offset{}, ui.Size{Width: 30, Height: 40}},
		{"unconstrained box", NewUnconstrainedBox(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 100, Height: 50}, ui.Offset{}, ui.Size{Width: 100, Height: 50}},
		{"overflow box", NewOverflowBox(newBox(100, 50), OverflowBoxOpts{MinWidth: foundation.Some(300.0), MaxWidth: foundation.Some(300.0)}), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 600}, ui.Offset{Dx: 250, Dy: 275}, ui.Size{Width: 300, Height: 50}},
		{"sized overflow box", NewSizedOverflowBox(ui.Size{Width: 40, Height: 20}, newBox(100, 50), SizedOverflowBoxOpts{Alignment: painting.AlignmentTopLeft}), ui.TextDirectionLTR,
			ui.Size{Width: 40, Height: 20}, ui.Offset{}, ui.Size{Width: 100, Height: 50}},
		{"fractionally sized box", NewFractionallySizedBox(newBox(100, 50), FractionallySizedBoxOpts{WidthFactor: foundation.Some(0.5), HeightFactor: foundation.Some(0.25)}), ui.TextDirectionLTR,
			ui.Size{Width: 400, Height: 150}, ui.Offset{}, ui.Size{Width: 400, Height: 150}},
		{"aspect ratio", NewAspectRatio(2, newBox(10, 10)), ui.TextDirectionLTR,
			ui.Size{Width: 800, Height: 400}, ui.Offset{}, ui.Size{Width: 800, Height: 400}},
		{"rotated box", NewRotatedBox(1, newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 50, Height: 100}, ui.Offset{Dx: 50}, ui.Size{Width: 100, Height: 50}},
		{"fractional translation", NewFractionalTranslation(ui.Offset{Dx: 0.5, Dy: 0.25}, newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{Width: 100, Height: 50}, ui.Offset{Dx: 50, Dy: 12.5}, ui.Size{Width: 100, Height: 50}},
		{"offstage", NewOffstage(newBox(100, 50)), ui.TextDirectionLTR,
			ui.Size{}, ui.Offset{}, ui.Size{Width: 100, Height: 50}},
		{"onstage", NewOffstage(newBox(100, 50), OffstageOpts{Offstage: foundation.Some(false)}), ui.TextDirectionLTR,
			ui.Size{Width: 100, Height: 50}, ui.Offset{}, ui.Size{Width: 100, Height: 50}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, tt.w, tt.dir)
			if got := ts.Size(gluttertest.ByWidget(tt.w)); got != tt.wantSize {
				t.Errorf("size of the widget = %v, want %v", got, tt.wantSize)
			}
			if got := ts.Size(anyBox); got != tt.wantBox {
				t.Errorf("size of the box = %v, want %v", got, tt.wantBox)
			}
			if tt.name != "rotated box" {
				if got := ts.TopLeft(anyBox); got != tt.wantChild {
					t.Errorf("position of the box = %v, want %v", got, tt.wantChild)
				}
			}
		})
	}
}

func TestFittedBoxScalesItsChild(t *testing.T) {
	tests := []struct {
		name  string
		fit   painting.BoxFit
		align painting.AlignmentGeometry
		want  ui.Offset // top left of the 100x50 box, scaled
		scale float64
	}{
		{"contain", painting.BoxFitContain, nil, ui.Offset{Dx: 0, Dy: 50}, 2},
		{"fill", painting.BoxFitFill, nil, ui.Offset{}, 0},
		{"none keeps the size, centered", painting.BoxFitNone, nil, ui.Offset{Dx: 50, Dy: 75}, 1},
		{"cover", painting.BoxFitCover, nil, ui.Offset{Dx: -100, Dy: 0}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewSizedBox(NewFittedBox(newBox(100, 50), FittedBoxOpts{Fit: tt.fit, Alignment: tt.align}),
				SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(200.0)}), ui.TextDirectionLTR)
			if got := ts.Size(gluttertest.ByType[*FittedBox]()); got != (ui.Size{Width: 200, Height: 200}) {
				t.Errorf("size = %v", got)
			}
			if tt.fit == painting.BoxFitFill {
				return
			}
			if got := ts.TopLeft(anyBox); got != tt.want {
				t.Errorf("top left of the child = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstraintsTransformBox(t *testing.T) {
	tests := []struct {
		name      string
		transform rendering.BoxConstraintsTransform
		want      ui.Size // size of the 300x300 box inside a 100x100 sized box
	}{
		{"unconstrained", ConstraintsTransformUnconstrained, ui.Size{Width: 300, Height: 300}},
		{"width unconstrained", ConstraintsTransformWidthUnconstrained, ui.Size{Width: 300, Height: 100}},
		{"height unconstrained", ConstraintsTransformHeightUnconstrained, ui.Size{Width: 100, Height: 300}},
		{"max unconstrained", ConstraintsTransformMaxUnconstrained, ui.Size{Width: 300, Height: 300}},
		{"max width unconstrained", ConstraintsTransformMaxWidthUnconstrained, ui.Size{Width: 300, Height: 100}},
		{"max height unconstrained", ConstraintsTransformMaxHeightUnconstrained, ui.Size{Width: 100, Height: 300}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewSizedBox(NewConstraintsTransformBox(tt.transform, newBox(300, 300)),
				SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), ui.TextDirectionLTR)
			if got := ts.Size(anyBox); got != tt.want {
				t.Errorf("size of the child = %v, want %v", got, tt.want)
			}
			// A child bigger than the box overflows, which the framework reports.
			for _, err := range ts.TakeErrors() {
				_ = requireErrorTag(t, err, "RenderOverflow")
			}
		})
	}
}

func TestUnconstrainedBoxConstrainedAxis(t *testing.T) {
	tests := []struct {
		name string
		axis painting.Axis
		want ui.Size // the 300x300 box in a 100x100 sized box
	}{
		{"none", painting.AxisUnspecified, ui.Size{Width: 300, Height: 300}},
		{"horizontal keeps the width", painting.AxisHorizontal, ui.Size{Width: 100, Height: 300}},
		{"vertical keeps the height", painting.AxisVertical, ui.Size{Width: 300, Height: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			pumpLoose(ts, NewSizedBox(NewUnconstrainedBox(newBox(300, 300), UnconstrainedBoxOpts{ConstrainedAxis: tt.axis}),
				SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), ui.TextDirectionLTR)
			if got := ts.Size(anyBox); got != tt.want {
				t.Errorf("size of the child = %v, want %v", got, tt.want)
			}
			// A child bigger than the box overflows, which the framework reports.
			for _, err := range ts.TakeErrors() {
				_ = requireErrorTag(t, err, "RenderOverflow")
			}
		})
	}
}

func TestIntrinsicWidgets(t *testing.T) {
	ts := gluttertest.New(t)
	// A column's children are as wide as the widest one with IntrinsicWidth.
	pumpLoose(ts, NewIntrinsicWidth(NewColumn([]Widget{newBox(30, 10), newBox(80, 10)},
		ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch, MainAxisSize: rendering.MainAxisSizeMin})), ui.TextDirectionLTR)
	if got := ts.Size(gluttertest.ByType[*IntrinsicWidth]()); got != (ui.Size{Width: 80, Height: 20}) {
		t.Errorf("IntrinsicWidth size = %v", got)
	}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewIntrinsicHeight(NewRow([]Widget{newBox(10, 30), newBox(10, 80)},
		RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch, MainAxisSize: rendering.MainAxisSizeMin})), AlignOpts{Alignment: painting.AlignmentTopLeft})))
	if got := ts.Size(gluttertest.ByType[*IntrinsicHeight]()); got != (ui.Size{Width: 20, Height: 80}) {
		t.Errorf("IntrinsicHeight size = %v", got)
	}
}

func TestBaselineWidget(t *testing.T) {
	ts := gluttertest.New(t)
	// The box has no baseline, so the Baseline child is placed at the top.
	pumpLoose(ts, NewBaseline(30, ui.TextBaselineAlphabetic, newBox(10, 10)), ui.TextDirectionLTR)
	if got := ts.Size(gluttertest.ByType[*Baseline]()); got.Width != 10 {
		t.Errorf("Baseline size = %v", got)
	}
}

func TestWidgetsUpdateTheirRenderObjects(t *testing.T) {
	// Each case pumps a widget, then another of the same type with other values,
	// and checks that the render object was updated, not replaced.
	tests := []struct {
		name  string
		a, b  Widget
		check func(t *testing.T, ro rendering.RenderObject)
	}{
		{"padding", NewPadding(painting.NewEdgeInsetsAll(1), newBox(1, 1)), NewPadding(painting.NewEdgeInsetsAll(5), newBox(1, 1)),
			func(t *testing.T, ro rendering.RenderObject) {
				if p := ro.(*rendering.RenderPadding).Padding(); p != painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(5)) {
					t.Errorf("padding = %v", p)
				}
			}},
		{"limited box", NewLimitedBox(newBox(1, 1)), NewLimitedBox(newBox(1, 1), LimitedBoxOpts{MaxWidth: foundation.Some(7.0)}),
			func(t *testing.T, ro rendering.RenderObject) {
				if w := ro.(*rendering.RenderLimitedBox).MaxWidth(); w != 7 {
					t.Errorf("max width = %v", w)
				}
			}},
		{"aspect ratio", NewAspectRatio(1, newBox(1, 1)), NewAspectRatio(3, newBox(1, 1)),
			func(t *testing.T, ro rendering.RenderObject) {
				if r := ro.(*rendering.RenderAspectRatio).AspectRatio(); r != 3 {
					t.Errorf("aspect ratio = %v", r)
				}
			}},
		{"rotated box", NewRotatedBox(1, newBox(1, 1)), NewRotatedBox(2, newBox(1, 1)),
			func(t *testing.T, ro rendering.RenderObject) {
				if r := ro.(*rendering.RenderRotatedBox).QuarterTurns(); r != 2 {
					t.Errorf("quarter turns = %v", r)
				}
			}},
		{"offstage", NewOffstage(newBox(1, 1)), NewOffstage(newBox(1, 1), OffstageOpts{Offstage: foundation.Some(false)}),
			func(t *testing.T, ro rendering.RenderObject) {
				if ro.(*rendering.RenderOffstage).Offstage() {
					t.Errorf("still offstage")
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(tt.a)))
			var ro rendering.RenderObject
			for _, e := range ts.Elements(anyBox) {
				ro = e.AsElement().Parent().RenderObject()
			}
			if ro == nil {
				t.Fatal("no render object found")
			}
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(tt.b)))
			for _, e := range ts.Elements(anyBox) {
				if again := e.AsElement().Parent().RenderObject(); again != ro {
					t.Fatalf("the render object was replaced")
				}
			}
			tt.check(t, ro)
		})
	}
}

func TestWidgetAssertions(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		do   func()
		tag  string
	}{
		{"align factor", func() { NewAlign(nil, AlignOpts{WidthFactor: foundation.Some(-1.0)}) }, "FactorNegative"},
		{"fractionally sized factor", func() { NewFractionallySizedBox(nil, FractionallySizedBoxOpts{HeightFactor: foundation.Some(-1.0)}) }, "FactorNegative"},
		{"limited box", func() { NewLimitedBox(nil, LimitedBoxOpts{MaxWidth: foundation.Some(-1.0)}) }, "ValueNegative"},
		{"intrinsic width", func() { NewIntrinsicWidth(nil, IntrinsicWidthOpts{StepWidth: foundation.Some(-1.0)}) }, "ValueNegative"},
		{"aspect ratio", func() { NewAspectRatio(0, nil) }, "AspectRatioInvalid"},
		{"opacity", func() { NewOpacity(1.5, nil) }, "OpacityOutOfRange"},
		{"constrained box", func() { NewConstrainedBox(rendering.BoxConstraints{MinWidth: 5, MaxWidth: 1}, nil) }, "BoxConstraints"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.do) })
	}
}
