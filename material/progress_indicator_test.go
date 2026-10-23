// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/progress_indicator_test.dart. The
// tests record what the painter of the indicator draws instead of using the
// `paints` matcher.

// paintedOps is what the painter of the CustomPaint inside the finder draws, at
// the size of that CustomPaint.
func paintedOps(t *testing.T, ts *gluttertest.Tester, indicator gluttertest.Finder) []ui.DisplayOp {
	t.Helper()
	paintFinder := gluttertest.Last(gluttertest.Descendant(indicator, gluttertest.ByType[*widgets.CustomPaint]()))
	cp := gluttertest.WidgetAs[*widgets.CustomPaint](ts, paintFinder)
	recorder := ui.NewPictureRecorder()
	cp.Painter().Paint(ui.NewCanvas(recorder), ts.Size(paintFinder))
	return recorder.EndRecording().DisplayList().Ops()
}

// pumpIndicator shows the widget in the top left of a Material app 800 by 600.
func pumpIndicator(t *testing.T, theme *ThemeData, child widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: widgets.NewAlign(child, widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}))
	return ts
}

func rectsOf(ops []ui.DisplayOp) (rects []ui.Rect, colors []ui.Color) {
	for _, op := range ops {
		if r, ok := op.(ui.OpDrawRect); ok {
			rects = append(rects, r.Rect)
			colors = append(colors, r.Paint.Color())
		}
	}
	return
}

func TestLinearProgressIndicatorDeterminate(t *testing.T) {
	material3 := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), VisualDensity: foundation.Some(VisualDensityStandard())})
	material2 := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false), VisualDensity: foundation.Some(VisualDensityStandard())})
	tests := []struct {
		name       string
		theme      *ThemeData
		opts       LinearProgressIndicatorOpts
		trackColor func(*ColorScheme) ui.Color
		wantTrack  ui.Rect
		wantValue  ui.Rect
	}{
		{"M3 2023 a quarter", material3, LinearProgressIndicatorOpts{Value: foundation.Some(0.25)},
			func(c *ColorScheme) ui.Color { return c.SecondaryContainer() }, ui.NewRectFromLTRB(0, 0, 200, 4), ui.NewRectFromLTRB(0, 0, 50, 4)},
		{"M2 half", material2, LinearProgressIndicatorOpts{Value: foundation.Some(0.5)},
			func(c *ColorScheme) ui.Color { return c.Background() }, ui.NewRectFromLTRB(0, 0, 200, 4), ui.NewRectFromLTRB(0, 0, 100, 4)},
		{"the value is clamped", material3, LinearProgressIndicatorOpts{Value: foundation.Some(1.5)},
			func(c *ColorScheme) ui.Color { return c.SecondaryContainer() }, ui.NewRectFromLTRB(0, 0, 200, 4), ui.NewRectFromLTRB(0, 0, 200, 4)},
		{"min height and color", material3, LinearProgressIndicatorOpts{Value: foundation.Some(0.5), MinHeight: foundation.Some(10.0), BackgroundColor: foundation.Some(ui.NewColor(0xFF00FF00))},
			func(*ColorScheme) ui.Color { return ui.NewColor(0xFF00FF00) }, ui.NewRectFromLTRB(0, 0, 200, 10), ui.NewRectFromLTRB(0, 0, 100, 10)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpIndicator(t, tc.theme, widgets.NewSizedBox(NewLinearProgressIndicator(tc.opts), widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
			if size := ts.Size(gluttertest.ByType[*LinearProgressIndicator]()); size.Width != 200 || size.Height != tc.wantValue.Bottom {
				t.Errorf("size %v", size)
			}
			rects, colors := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()))
			scheme := tc.theme.ColorScheme
			// The track is painted over the whole width, with the value over it.
			wantRects := []ui.Rect{tc.wantTrack, tc.wantValue}
			wantColors := []ui.Color{tc.trackColor(scheme), scheme.Primary()}
			if len(rects) != len(wantRects) {
				t.Fatalf("rects %v, want %v", rects, wantRects)
			}
			for i := range rects {
				if rects[i] != wantRects[i] || colors[i] != wantColors[i] {
					t.Errorf("rect %d is %v in %v, want %v in %v", i, rects[i], colors[i], wantRects[i], wantColors[i])
				}
			}
		})
	}
}

func TestLinearProgressIndicatorRightToLeft(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewDirectionality(ui.TextDirectionRTL,
		widgets.NewAlign(widgets.NewSizedBox(NewLinearProgressIndicator(LinearProgressIndicatorOpts{Value: foundation.Some(0.25)}), widgets.SizedBoxOpts{Width: foundation.Some(200.0)}),
			widgets.AlignOpts{Alignment: painting.AlignmentTopLeft}))}))
	rects, _ := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()))
	// The value starts at the right.
	if len(rects) != 2 || rects[1] != ui.NewRectFromLTRB(150, 0, 200, 4) || rects[0] != ui.NewRectFromLTRB(0, 0, 200, 4) {
		t.Errorf("rects %v", rects)
	}
}

func TestLinearProgressIndicator2024(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), VisualDensity: foundation.Some(VisualDensityStandard())})
	ts := pumpIndicator(t, theme, widgets.NewSizedBox(NewLinearProgressIndicator(LinearProgressIndicatorOpts{Value: foundation.Some(0.5), Year2023: foundation.Some(false)}),
		widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
	var rrects []ui.RRect
	var circles []ui.OpDrawCircle
	for _, op := range paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()) {
		switch op := op.(type) {
		case ui.OpDrawRRect:
			rrects = append(rrects, op.RRect)
		case ui.OpDrawCircle:
			circles = append(circles, op)
		}
	}
	if len(rrects) != 2 {
		t.Fatalf("%d rounded rectangles, want the track and the value", len(rrects))
	}
	// The gap of 4 is 0.02 of the width: the track starts at 0.5 + 0.02 * 0.5 / 0.01 clamped to a gap.
	if got := rrects[0].Left; math.Abs(got-(0.5*200+4)) > 1e-9 {
		t.Errorf("the track starts at %v, want 104", got)
	}
	if rrects[1].Right != 100 || rrects[1].Bottom != 4 {
		t.Errorf("value %v", rrects[1])
	}
	if len(circles) != 1 || circles[0].Radius != 2 || circles[0].Center != (ui.Offset{Dx: 198, Dy: 2}) || circles[0].Paint.Color() != theme.ColorScheme.Primary() {
		t.Errorf("stop indicator %+v", circles)
	}
}

func TestLinearProgressIndicatorIndeterminateAnimates(t *testing.T) {
	ts := pumpIndicator(t, NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true)}), widgets.NewSizedBox(NewLinearProgressIndicator(),
		widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
	seen := map[ui.Rect]bool{}
	for i := 0; i < 8; i++ {
		ts.Pump(200 * time.Millisecond)
		rects, _ := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()))
		for _, r := range rects {
			seen[r] = true
		}
	}
	if len(seen) < 8 {
		t.Errorf("the indicator did not move: %d different rectangles", len(seen))
	}
	// At the start no line has begun: only the track is painted.
	ts = pumpIndicator(t, NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true)}), widgets.NewSizedBox(NewLinearProgressIndicator(),
		widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
	rects, _ := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()))
	if len(rects) != 1 || rects[0] != ui.NewRectFromLTRB(0, 0, 200, 4) {
		t.Errorf("rectangles at the start: %v", rects)
	}
}

func TestLinearProgressIndicatorSemantics(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		opts LinearProgressIndicatorOpts
		want string
	}{
		{"determinate", LinearProgressIndicatorOpts{Value: foundation.Some(0.5), SemanticsLabel: "Loading"}, "Loading"},
		{"indeterminate", LinearProgressIndicatorOpts{SemanticsLabel: "Waiting"}, "Waiting"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpIndicator(t, standardTheme(true), NewLinearProgressIndicator(tc.opts))
			handle := ts.EnsureSemantics()
			defer handle.Dispose()
			ts.Pump()
			if !ts.Exists(gluttertest.BySemanticsLabel(tc.want)) {
				t.Errorf("no semantics label %q", tc.want)
			}
		})
	}
}

func TestCircularProgressIndicatorDeterminate(t *testing.T) {
	tests := []struct {
		name      string
		opts      CircularProgressIndicatorOpts
		material3 bool
		size      float64
		stroke    float64
		arcs      int
		sweep     float64
	}{
		{"M3 2023 half", CircularProgressIndicatorOpts{Value: foundation.Some(0.5)}, true, 36, 4, 1, 0.5 * (2*math.Pi - 0.001)},
		{"M2 full", CircularProgressIndicatorOpts{Value: foundation.Some(1.0)}, false, 36, 4, 1, 2*math.Pi - 0.001},
		{"M3 2024 gets a track, a padding and 40 pixels", CircularProgressIndicatorOpts{Value: foundation.Some(0.25), Year2023: foundation.Some(false)}, true, 48, 4, 2, 0.25 * (2*math.Pi - 0.001)},
		{"stroke width", CircularProgressIndicatorOpts{Value: foundation.Some(0.25), StrokeWidth: foundation.Some(10.0)}, true, 36, 10, 1, 0.25 * (2*math.Pi - 0.001)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(tc.material3)})
			ts := pumpIndicator(t, theme, NewCircularProgressIndicator(tc.opts))
			if size := ts.Size(gluttertest.ByType[*CircularProgressIndicator]()); size.Width != tc.size || size.Height != tc.size {
				t.Errorf("size %v, want %v", size, tc.size)
			}
			var arcs []ui.OpDrawArc
			for _, op := range paintedOps(t, ts, gluttertest.ByType[*CircularProgressIndicator]()) {
				if a, ok := op.(ui.OpDrawArc); ok {
					arcs = append(arcs, a)
				}
			}
			if len(arcs) != tc.arcs {
				t.Fatalf("%d arcs, want %d", len(arcs), tc.arcs)
			}
			value := arcs[len(arcs)-1]
			if math.Abs(value.SweepAngle-tc.sweep) > 1e-9 || value.StartAngle != -math.Pi/2 || value.Paint.Color() != theme.ColorScheme.Primary() || value.Paint.StrokeWidth() != tc.stroke {
				t.Errorf("value arc %+v, sweep want %v", value, tc.sweep)
			}
		})
	}
}

func TestCircularProgressIndicatorIndeterminate(t *testing.T) {
	ts := pumpIndicator(t, standardTheme(true), NewCircularProgressIndicator())
	arcAt := func() ui.OpDrawArc {
		for _, op := range paintedOps(t, ts, gluttertest.ByType[*CircularProgressIndicator]()) {
			if a, ok := op.(ui.OpDrawArc); ok {
				return a
			}
		}
		t.Fatal("no arc")
		return ui.OpDrawArc{}
	}
	first := arcAt()
	ts.Pump(300 * time.Millisecond)
	second := arcAt()
	if first.StartAngle == second.StartAngle && first.SweepAngle == second.SweepAngle {
		t.Error("the arc did not move")
	}
	if second.Paint.StrokeCap() != ui.StrokeCapSquare {
		t.Errorf("the indeterminate arc of 2023 has the cap %v, want square", second.Paint.StrokeCap())
	}
	// Setting a value stops the animation.
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewCircularProgressIndicator(CircularProgressIndicatorOpts{Value: foundation.Some(0.5)})}))
	ts.PumpAndSettle() // it would never settle if the indicator kept animating
}

func TestCircularProgressIndicatorController(t *testing.T) {
	// The indicator follows the controller that is given to it.
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	controller := animation.NewAnimationController(vsync{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{Duration: foundation.Some(time.Hour)})
	defer controller.Dispose()
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewCircularProgressIndicator(CircularProgressIndicatorOpts{Controller: controller})}))
	ts.Pump()
	first := paintedOps(t, ts, gluttertest.ByType[*CircularProgressIndicator]())
	controller.SetValue(0.001)
	ts.Pump()
	second := paintedOps(t, ts, gluttertest.ByType[*CircularProgressIndicator]())
	if len(first) == 0 || first[len(first)-1] == second[len(second)-1] {
		t.Error("the indicator does not follow the controller")
	}
}

func TestProgressIndicatorTheme(t *testing.T) {
	data := ProgressIndicatorThemeData{
		Color: foundation.Some(ui.NewColor(0xFFAA0000)), LinearTrackColor: foundation.Some(ui.NewColor(0xFF00AA00)), LinearMinHeight: foundation.Some(9.0),
		CircularTrackColor: foundation.Some(ui.NewColor(0xFF0000AA)), StrokeWidth: foundation.Some(7.0), StrokeCap: foundation.Some(ui.StrokeCapRound),
		Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(50.0), MinHeight: foundation.Some(50.0)})),
	}
	ts := pumpIndicator(t, standardTheme(true), NewProgressIndicatorTheme(data, widgets.NewColumn([]widgets.Widget{
		widgets.NewSizedBox(NewLinearProgressIndicator(LinearProgressIndicatorOpts{Value: foundation.Some(0.5)}), widgets.SizedBoxOpts{Width: foundation.Some(200.0)}),
		NewCircularProgressIndicator(CircularProgressIndicatorOpts{Value: foundation.Some(0.5)}),
	}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})))
	rects, colors := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]()))
	if len(rects) != 2 || colors[0] != data.LinearTrackColor.Value() || colors[1] != data.Color.Value() || rects[0].Bottom != 9 {
		t.Errorf("linear %v %v", rects, colors)
	}
	if size := ts.Size(gluttertest.ByType[*CircularProgressIndicator]()); size.Width != 50 {
		t.Errorf("circular size %v, the theme says 50", size)
	}
	var arcs []ui.OpDrawArc
	for _, op := range paintedOps(t, ts, gluttertest.ByType[*CircularProgressIndicator]()) {
		if a, ok := op.(ui.OpDrawArc); ok {
			arcs = append(arcs, a)
		}
	}
	if len(arcs) != 2 || arcs[0].Paint.Color() != data.CircularTrackColor.Value() || arcs[1].Paint.StrokeWidth() != 7 || arcs[1].Paint.StrokeCap() != ui.StrokeCapRound {
		t.Errorf("arcs %+v", arcs)
	}
	// The theme of ThemeData is used without an inherited one.
	themed := NewThemeData(ThemeDataOpts{ProgressIndicatorTheme: data})
	ts = pumpIndicator(t, themed, widgets.NewSizedBox(NewLinearProgressIndicator(LinearProgressIndicatorOpts{Value: foundation.Some(0.5)}), widgets.SizedBoxOpts{Width: foundation.Some(200.0)}))
	if _, colors := rectsOf(paintedOps(t, ts, gluttertest.ByType[*LinearProgressIndicator]())); len(colors) != 2 || colors[1] != data.Color.Value() {
		t.Errorf("the theme of ThemeData is not used: %v", colors)
	}
}

func TestRefreshProgressIndicator(t *testing.T) {
	ts := pumpIndicator(t, standardTheme(true), NewRefreshProgressIndicator(RefreshProgressIndicatorOpts{Value: foundation.Some(1.0)}))
	if size := ts.Size(gluttertest.ByType[*RefreshProgressIndicator]()); size.Width != 49 || size.Height != 49 {
		t.Errorf("size %v, want the disc of 41 and a margin of 4", size)
	}
	var arcs, paths int
	for _, op := range paintedOps(t, ts, gluttertest.ByType[*RefreshProgressIndicator]()) {
		switch op.(type) {
		case ui.OpDrawArc:
			arcs++
		case ui.OpDrawPath:
			paths++
		}
	}
	if arcs != 1 || paths != 1 {
		t.Errorf("%d arcs and %d arrow heads, want one of each", arcs, paths)
	}
	// An indeterminate indicator has no arrow head yet and spins.
	ts = pumpIndicator(t, standardTheme(true), NewRefreshProgressIndicator())
	for _, op := range paintedOps(t, ts, gluttertest.ByType[*RefreshProgressIndicator]()) {
		if _, ok := op.(ui.OpDrawPath); ok {
			t.Error("an indeterminate refresh indicator paints an arrow head")
		}
	}
}

func TestProgressIndicatorErrors(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	controller := animation.NewAnimationController(vsync{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{})
	defer controller.Dispose()
	tests := []struct {
		name string
		f    func()
		tag  string
	}{
		{"linear min height", func() { NewLinearProgressIndicator(LinearProgressIndicatorOpts{MinHeight: foundation.Some(0.0)}) }, "LinearProgressIndicatorMinHeightNotPositive"},
		{"linear value and controller", func() {
			NewLinearProgressIndicator(LinearProgressIndicatorOpts{Value: foundation.Some(0.5), Controller: controller})
		}, "NewLinearProgressIndicator.LinearProgressIndicatorValueAndController"},
		{"circular value and controller", func() {
			NewCircularProgressIndicator(CircularProgressIndicatorOpts{Value: foundation.Some(0.5), Controller: controller})
		}, "newCircularProgressIndicator.CircularProgressIndicatorValueAndController"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { wantRoute(t, recoverError(tc.f), tc.tag) })
	}
}

func TestProgressIndicatorThemeData(t *testing.T) {
	a := ProgressIndicatorThemeData{Color: foundation.Some(ui.NewColor(0xFF000000)), StrokeWidth: foundation.Some(2.0), LinearMinHeight: foundation.Some(4.0)}
	b := ProgressIndicatorThemeData{Color: foundation.Some(ui.NewColor(0xFFFFFFFF)), StrokeWidth: foundation.Some(6.0), LinearMinHeight: foundation.Some(8.0)}
	got := ProgressIndicatorThemeDataLerp(a, b, 0.5)
	if got.StrokeWidth != foundation.Some(4.0) || got.LinearMinHeight != foundation.Some(6.0) {
		t.Errorf("lerp %v %v", got.StrokeWidth, got.LinearMinHeight)
	}
	if want := lerpColor(a.Color.Value(), b.Color.Value(), 0.5); got.Color != foundation.Some(want) {
		t.Errorf("color %v, want %v", got.Color, want)
	}
	if !a.Equal(a) || a.Equal(b) {
		t.Error("Equal")
	}
	if c := a.CopyWith(ProgressIndicatorThemeDataCopy{StrokeWidth: foundation.Some(9.0)}); c.StrokeWidth != foundation.Some(9.0) || c.Color != a.Color {
		t.Errorf("CopyWith %v", c)
	}
}
