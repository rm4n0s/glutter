// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from the painter tests of packages/flutter/test/widgets/scrollbar_test.dart.

var scrollbarColor = ui.NewColor(0xFF123456)

const (
	scrollbarThick = 2.5
	minThumbExtent = 18.0
)

type paintedScrollbar struct {
	rects  []ui.Rect
	colors []ui.Color // the color of each rectangle
	rrects []ui.RRect
}

// last is the last rectangle that was drawn: the thumb.
func (p *paintedScrollbar) lastRect(t *testing.T) ui.Rect {
	t.Helper()
	if len(p.rects) == 0 {
		t.Fatal("no rectangle was drawn")
	}
	r := p.rects[len(p.rects)-1]
	p.rects, p.colors = p.rects[:len(p.rects)-1], p.colors[:len(p.colors)-1]
	return r
}

func (p *paintedScrollbar) lastRRect(t *testing.T) ui.RRect {
	t.Helper()
	if len(p.rrects) == 0 {
		t.Fatal("no rounded rectangle was drawn")
	}
	r := p.rrects[len(p.rrects)-1]
	p.rrects = p.rrects[:len(p.rrects)-1]
	return r
}

func paintScrollbar(painter *ScrollbarPainter, size ui.Size) *paintedScrollbar {
	recorder := ui.NewPictureRecorder()
	canvas := ui.NewCanvas(recorder)
	painter.Paint(canvas, size)
	out := &paintedScrollbar{}
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		switch op := op.(type) {
		case ui.OpDrawRect:
			out.rects = append(out.rects, op.Rect)
			out.colors = append(out.colors, op.Paint.Color())
		case ui.OpDrawRRect:
			out.rrects = append(out.rrects, op.RRect)
		}
	}
	return out
}

type painterOpts struct {
	textDirection        ui.TextDirection
	padding              painting.EdgeInsetsGeometry
	mainAxisMargin       float64
	crossAxisMargin      float64
	radius, trackRadius  foundation.Opt[ui.Radius]
	minLength            foundation.Opt[float64]
	minOverscrollLength  foundation.Opt[float64]
	scrollbarOrientation ScrollbarOrientation
}

func buildPainter(metrics ScrollMetrics, o painterOpts) *ScrollbarPainter {
	if o.textDirection == ui.TextDirectionUnspecified {
		o.textDirection = ui.TextDirectionLTR
	}
	minLength := o.minLength.OrElse(minThumbExtent)
	p := NewScrollbarPainter(scrollbarColor, animation.AlwaysComplete, ScrollbarPainterOpts{
		TextDirection: o.textDirection, Thickness: foundation.Some(scrollbarThick), Padding: o.padding,
		MainAxisMargin: o.mainAxisMargin, CrossAxisMargin: o.crossAxisMargin, Radius: o.radius, TrackRadius: o.trackRadius,
		MinLength: foundation.Some(minLength), MinOverscrollLength: foundation.Some(o.minOverscrollLength.OrElse(minLength)),
		ScrollbarOrientation: o.scrollbarOrientation})
	p.Update(metrics, metrics.AxisDirection())
	return p
}

func defaultScrollMetrics() ScrollMetrics {
	return NewFixedScrollMetrics(foundation.Some(0.0), foundation.Some(0.0), foundation.Some(0.0), foundation.Some(100.0),
		painting.AxisDirectionDown, 3.0)
}

func with(m ScrollMetrics, c ScrollMetricsChanges) ScrollMetrics { return m.CopyWith(c) }

func TestScrollbarPainterIsNotSmallerThanMinLength(t *testing.T) {
	const minLen = 3.5
	size := ui.Size{Width: 600, Height: 10}
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(100000.0), ViewportDimension: foundation.Some(size.Height)})
	painter := buildPainter(metrics, painterOpts{minLength: foundation.Some(minLen), minOverscrollLength: foundation.Some(minLen)})

	rect := paintScrollbar(painter, size).lastRect(t) // when overscroll
	if rect.Top != 0 || rect.Left != size.Width-scrollbarThick || rect.Width() != scrollbarThick || rect.Height() < minLen {
		t.Errorf("rect = %v", rect)
	}
	painter.Update(with(metrics, ScrollMetricsChanges{Pixels: foundation.Some(1.0)}), metrics.AxisDirection()) // when scroll normally
	rect = paintScrollbar(painter, size).lastRect(t)
	if rect.Left != size.Width-scrollbarThick || rect.Width() != scrollbarThick || rect.Height() < minLen {
		t.Errorf("rect = %v", rect)
	}
}

func TestScrollbarPainterKeepsItsSizeWhileScrollingNormally(t *testing.T) {
	const viewportDimension, maxExtent = 23.0, 100.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(maxExtent), ViewportDimension: foundation.Some(viewportDimension)})
	size := ui.Size{Width: 600, Height: viewportDimension}
	painter := buildPainter(defaultScrollMetrics(), painterOpts{minLength: foundation.Some(0.0), minOverscrollLength: foundation.Some(0.0)})

	metricsList := []ScrollMetrics{with(starting, ScrollMetricsChanges{Pixels: foundation.Some(0.01)})}
	for i := 0; i < int(math.Round(maxExtent/viewportDimension)); i++ {
		m := with(starting, ScrollMetricsChanges{Pixels: foundation.Some(float64(i+1) * viewportDimension)})
		if !m.OutOfRange() {
			metricsList = append(metricsList, m)
		}
	}
	metricsList = append(metricsList, with(starting, ScrollMetricsChanges{Pixels: foundation.Some(maxExtent - 0.01)}))

	wantHeight := viewportDimension * viewportDimension / (viewportDimension + maxExtent)
	for _, m := range metricsList {
		painter.Update(m, m.AxisDirection())
		rect := paintScrollbar(painter, size).lastRect(t)
		if rect.Top < 0 || rect.Bottom > maxExtent || rect.Left != size.Width-scrollbarThick || rect.Width() != scrollbarThick ||
			math.Abs(rect.Height()-wantHeight) > 0.001 {
			t.Errorf("pixels %v: rect = %v (want height %v)", m.Pixels(), rect, wantHeight)
		}
	}
}

func TestScrollbarPainterMainAxisMarginIsRespected(t *testing.T) {
	const viewportDimension, maxExtent = 23.0, 100.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(maxExtent), ViewportDimension: foundation.Some(viewportDimension)})
	size := ui.Size{Width: 600, Height: viewportDimension}
	for _, margin := range []float64{-10, 1, viewportDimension/2 - 0.01} {
		painter := buildPainter(defaultScrollMetrics(), painterOpts{mainAxisMargin: margin, minLength: foundation.Some(0.0)})
		// The thumb of an overscroll this far has no length and a canvas does not
		// record an empty rectangle, so the test reads where the thumb is.
		painter.Update(with(starting, ScrollMetricsChanges{Pixels: foundation.Some(math.Inf(-1))}), starting.AxisDirection()) // overscroll to the top
		paintScrollbar(painter, size)
		if top := painter.ThumbOffset(); top != margin {
			t.Errorf("margin %v: top = %v", margin, top)
		}
		painter.Update(with(starting, ScrollMetricsChanges{Pixels: foundation.Some(math.Inf(1))}), starting.AxisDirection()) // overscroll down
		paintScrollbar(painter, size)
		if gap := size.Height - painter.ThumbOffset(); gap != margin {
			t.Errorf("margin %v: bottom gap = %v", margin, gap)
		}
	}
}

func TestScrollbarPainterCrossAxisMarginAndTextDirectionAreRespected(t *testing.T) {
	const viewportDimension, maxExtent, margin = 23.0, 100.0, 4.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(maxExtent), ViewportDimension: foundation.Some(viewportDimension)})
	size := ui.Size{Width: 600, Height: viewportDimension}
	directions := []painting.AxisDirection{painting.AxisDirectionUp, painting.AxisDirectionRight, painting.AxisDirectionDown, painting.AxisDirectionLeft}
	for _, textDirection := range []ui.TextDirection{ui.TextDirectionLTR, ui.TextDirectionRTL} {
		painter := buildPainter(starting, painterOpts{crossAxisMargin: margin, textDirection: textDirection})
		for _, direction := range directions {
			painter.Update(with(starting, ScrollMetricsChanges{AxisDirection: direction}), direction)
			rect := paintScrollbar(painter, size).lastRect(t)
			var got float64
			if painting.AxisDirectionToAxis(direction) == painting.AxisVertical {
				got = size.Width - rect.Right
				if textDirection == ui.TextDirectionRTL {
					got = rect.Left
				}
			} else {
				got = size.Height - rect.Bottom
			}
			if got != margin {
				t.Errorf("%v %v: margin = %v, want %v", textDirection, direction, got, margin)
			}
		}
	}
}

func TestScrollbarPainterOrientationIsRespected(t *testing.T) {
	const viewportDimension, maxExtent = 23.0, 100.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(maxExtent), ViewportDimension: foundation.Some(viewportDimension)})
	size := ui.Size{Width: 600, Height: viewportDimension}
	tests := []struct {
		orientation ScrollbarOrientation
		axis        painting.AxisDirection
		want        ui.Rect
	}{
		{ScrollbarOrientationLeft, painting.AxisDirectionDown, ui.NewRectFromLTRB(0, 0, scrollbarThick, minThumbExtent)},
		{ScrollbarOrientationRight, painting.AxisDirectionDown, ui.NewRectFromLTRB(600-scrollbarThick, 0, 600, minThumbExtent)},
		{ScrollbarOrientationTop, painting.AxisDirectionRight, ui.NewRectFromLTRB(0, 0, minThumbExtent, scrollbarThick)},
		{ScrollbarOrientationBottom, painting.AxisDirectionRight, ui.NewRectFromLTRB(0, 23-scrollbarThick, minThumbExtent, 23)},
	}
	for _, tt := range tests {
		t.Run(tt.orientation.String(), func(t *testing.T) {
			painter := buildPainter(starting, painterOpts{scrollbarOrientation: tt.orientation})
			painter.Update(with(starting, ScrollMetricsChanges{AxisDirection: tt.axis}), tt.axis)
			if got := paintScrollbar(painter, size).lastRect(t); got != tt.want {
				t.Errorf("rect = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScrollbarPainterDefaultOrientations(t *testing.T) {
	const viewportDimension, maxExtent = 23.0, 100.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(maxExtent), ViewportDimension: foundation.Some(viewportDimension)})
	size := ui.Size{Width: 600, Height: viewportDimension}
	tests := []struct {
		name string
		dir  ui.TextDirection
		axis painting.AxisDirection
		want ui.Rect
	}{
		{"vertical ltr", ui.TextDirectionLTR, painting.AxisDirectionDown, ui.NewRectFromLTRB(600-scrollbarThick, 0, 600, minThumbExtent)},
		{"vertical rtl", ui.TextDirectionRTL, painting.AxisDirectionDown, ui.NewRectFromLTRB(0, 0, scrollbarThick, minThumbExtent)},
		{"horizontal", ui.TextDirectionLTR, painting.AxisDirectionRight, ui.NewRectFromLTRB(0, 23-scrollbarThick, minThumbExtent, 23)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			painter := buildPainter(starting, painterOpts{textDirection: tt.dir})
			painter.Update(with(starting, ScrollMetricsChanges{AxisDirection: tt.axis}), tt.axis)
			if got := paintScrollbar(painter, size).lastRect(t); got != tt.want {
				t.Errorf("rect = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScrollbarPainterPaddingWorksForAllScrollDirections(t *testing.T) {
	padding := painting.NewEdgeInsetsFromLTRB(1, 2, 3, 4)
	size := ui.Size{Width: 60, Height: 80}
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MinScrollExtent: foundation.Some(-100.0), MaxScrollExtent: foundation.Some(240.0)})
	inf := math.Inf(1)
	tests := []struct {
		name  string
		axis  painting.AxisDirection
		dim   float64
		first float64 // the pixels of the first overscroll
		check func(t *testing.T, first, second ui.Rect)
	}{
		{"down", painting.AxisDirectionDown, size.Height, -inf, func(t *testing.T, a, b ui.Rect) {
			expectEq(t, "top", a.Top, padding.Top)
			expectEq(t, "right", size.Width-a.Right, padding.Right)
			expectEq(t, "bottom", size.Height-b.Bottom, padding.Bottom)
			expectEq(t, "right", size.Width-b.Right, padding.Right)
		}},
		{"up", painting.AxisDirectionUp, size.Height, inf, func(t *testing.T, a, b ui.Rect) {
			expectEq(t, "top", a.Top, padding.Top)
			expectEq(t, "right", size.Width-a.Right, padding.Right)
			expectEq(t, "bottom", size.Height-b.Bottom, padding.Bottom)
			expectEq(t, "right", size.Width-b.Right, padding.Right)
		}},
		{"left", painting.AxisDirectionLeft, size.Width, -inf, func(t *testing.T, a, b ui.Rect) {
			expectEq(t, "bottom", size.Height-a.Bottom, padding.Bottom)
			expectEq(t, "right", size.Width-a.Right, padding.Right)
			expectEq(t, "bottom", size.Height-b.Bottom, padding.Bottom)
			expectEq(t, "left", b.Left, padding.Left)
		}},
		{"right", painting.AxisDirectionRight, size.Width, inf, func(t *testing.T, a, b ui.Rect) {
			expectEq(t, "bottom", size.Height-a.Bottom, padding.Bottom)
			expectEq(t, "right", size.Width-a.Right, padding.Right)
			expectEq(t, "bottom", size.Height-b.Bottom, padding.Bottom)
			expectEq(t, "left", b.Left, padding.Left)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			painter := buildPainter(with(metrics, ScrollMetricsChanges{AxisDirection: painting.AxisDirectionDown}), painterOpts{padding: padding})
			painter.Update(with(metrics, ScrollMetricsChanges{ViewportDimension: foundation.Some(tt.dim), Pixels: foundation.Some(tt.first), AxisDirection: tt.axis}), tt.axis)
			a := paintScrollbar(painter, size).lastRect(t)
			painter.Update(with(metrics, ScrollMetricsChanges{ViewportDimension: foundation.Some(tt.dim), Pixels: foundation.Some(-tt.first), AxisDirection: tt.axis}), tt.axis)
			b := paintScrollbar(painter, size).lastRect(t)
			tt.check(t, a, b)
		})
	}
}

func expectEq(t *testing.T, what string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestScrollbarPainterThumbResizesGraduallyOnOverscroll(t *testing.T) {
	padding := painting.NewEdgeInsetsFromLTRB(1, 2, 3, 4)
	size := ui.Size{Width: 60, Height: 300}
	scrollExtent := size.Height * 10
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(scrollExtent), ViewportDimension: foundation.Some(size.Height)})
	const minOverscrollLength = 8.0
	painter := buildPainter(metrics, painterOpts{padding: padding, minLength: foundation.Some(36.0), minOverscrollLength: foundation.Some(minOverscrollLength)})
	heightAt := func(pixels float64) float64 {
		painter.Update(with(metrics, ScrollMetricsChanges{Pixels: foundation.Some(pixels)}), painting.AxisDirectionDown)
		return paintScrollbar(painter, size).lastRect(t).Height()
	}
	full := heightAt(0) // no overscroll gives a full sized thumb
	if full <= minThumbExtent {
		t.Fatalf("full thumb = %v", full)
	}
	for _, pixels := range []float64{scrollExtent / 2, scrollExtent} { // the middle and the very end
		if got := heightAt(pixels); math.Abs(got-full) > 1e-6 {
			t.Errorf("at %v the thumb is %v, want %v", pixels, got, full)
		}
	}
	if got := heightAt(scrollExtent * 1.001); math.Abs(got-full) > 2 { // just past the end shrinks it slightly
		t.Errorf("just past the end the thumb is %v, want about %v", got, full)
	}
	if got := heightAt(math.Inf(1)); got != minOverscrollLength { // way past the end shrinks it to the minimum
		t.Errorf("way past the end the thumb is %v, want %v", got, minOverscrollLength)
	}
}

func TestScrollbarPainterScrollsTowardsTheRightDirection(t *testing.T) {
	size := ui.Size{Width: 60, Height: 80}
	const maxScrollExtent, minScrollExtent = 240.0, -100.0
	starting := with(defaultScrollMetrics(), ScrollMetricsChanges{MinScrollExtent: foundation.Some(minScrollExtent),
		MaxScrollExtent: foundation.Some(maxScrollExtent), ViewportDimension: foundation.Some(size.Height)})
	for _, minLength := range []float64{minThumbExtent, math.Inf(1)} {
		// Disregard the minimum lengths to keep the scroll direction correct, if needed.
		painter := buildPainter(starting, painterOpts{minLength: foundation.Some(minLength), minOverscrollLength: foundation.Some(minLength)})
		var previous *ui.Rect
		for i := 0; i < 9999; i++ {
			m := with(starting, ScrollMetricsChanges{Pixels: foundation.Some(minScrollExtent + float64(i)*size.Height/3)})
			if m.OutOfRange() {
				break
			}
			painter.Update(m, m.AxisDirection())
			rect := paintScrollbar(painter, size).lastRect(t)
			if previous != nil {
				if rect.Height() == size.Height { // the scrollbar is too large for the viewport
					if !(previous.Top <= rect.Top && previous.Bottom <= rect.Bottom) {
						t.Fatalf("minLength %v, step %d: %v after %v", minLength, i, rect, *previous)
					}
				} else if !(previous.Top < rect.Top && previous.Bottom < rect.Bottom) { // the scrollbar fits in the viewport
					t.Fatalf("minLength %v, step %d: %v after %v", minLength, i, rect, *previous)
				}
			}
			r := rect
			previous = &r
		}
	}
}

func TestScrollbarPainterRadiusAndTrackRadiusAreRespected(t *testing.T) {
	const minLen = 3.5
	size := ui.Size{Width: 600, Height: 10}
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(100000.0), ViewportDimension: foundation.Some(size.Height)})
	painter := buildPainter(metrics, painterOpts{trackRadius: foundation.Some(ui.NewRadiusCircular(2)), radius: foundation.Some(ui.NewRadiusCircular(3)),
		minLength: foundation.Some(minLen), minOverscrollLength: foundation.Some(minLen)})
	painted := paintScrollbar(painter, size)
	thumb := painted.lastRRect(t)
	for name, got := range map[string]ui.Radius{"bl": thumb.BlRadius(), "br": thumb.BrRadius(), "tl": thumb.TlRadius(), "tr": thumb.TrRadius()} {
		if got != ui.NewRadiusCircular(3) {
			t.Errorf("thumb %s radius = %v, want 3", name, got)
		}
	}
	track := painted.lastRRect(t)
	for name, got := range map[string]ui.Radius{"bl": track.BlRadius(), "br": track.BrRadius(), "tl": track.TlRadius(), "tr": track.TrRadius()} {
		if got != ui.NewRadiusCircular(2) {
			t.Errorf("track %s radius = %v, want 2", name, got)
		}
	}
}

func TestScrollbarPainterNeedsATextDirectionToPaint(t *testing.T) {
	skipUnlessDebug(t)
	painter := NewScrollbarPainter(scrollbarColor, animation.AlwaysComplete)
	size := ui.Size{Width: 60, Height: 80}
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(100000.0), ViewportDimension: foundation.Some(size.Height)})
	painter.Update(metrics, metrics.AxisDirection())
	requireTag(t, "ScrollbarPainterInvalid", func() { paintScrollbar(painter, size) })
	requireTag(t, "ScrollbarPainterInvalid", func() { painter.SetTextDirection(ui.TextDirectionUnspecified) })
}

func TestScrollbarPainterOrientationMustMatchTheAxis(t *testing.T) {
	skipUnlessDebug(t)
	size := ui.Size{Width: 60, Height: 80}
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(100000.0), ViewportDimension: foundation.Some(size.Height)})
	painter := buildPainter(metrics, painterOpts{scrollbarOrientation: ScrollbarOrientationTop}) // top is for a horizontal axis
	requireTag(t, "ScrollbarOrientationInvalid", func() { paintScrollbar(painter, size) })
}

func TestScrollbarPainterThumbScrollOffsetNeedsFiniteExtents(t *testing.T) {
	skipUnlessDebug(t)
	metrics := with(defaultScrollMetrics(), ScrollMetricsChanges{MaxScrollExtent: foundation.Some(math.Inf(1))})
	painter := buildPainter(metrics, painterOpts{})
	requireTag(t, "ScrollbarExtentsInfinite", func() { painter.GetThumbScrollOffset() })
}

func TestScrollbarPainterOptionsMustBeConsistent(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		o    ScrollbarPainterOpts
	}{
		{"a radius and a shape", ScrollbarPainterOpts{Radius: foundation.Some(ui.NewRadiusCircular(1)), Shape: painting.NewCircleBorder()}},
		{"a negative minimum length", ScrollbarPainterOpts{MinLength: foundation.Some(-1.0)}},
		{"a minimum overscroll length above the minimum length", ScrollbarPainterOpts{MinLength: foundation.Some(10.0), MinOverscrollLength: foundation.Some(20.0)}},
		{"a negative minimum overscroll length", ScrollbarPainterOpts{MinOverscrollLength: foundation.Some(-1.0)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTag(t, "ScrollbarPainterInvalid", func() { NewScrollbarPainter(scrollbarColor, animation.AlwaysComplete, tt.o) })
		})
	}
}

func TestScrollbarPainterShouldRepaint(t *testing.T) {
	create := func(change func(o *ScrollbarPainterOpts, color *ui.Color, fade *animation.Animation[float64])) *ScrollbarPainter {
		o := ScrollbarPainterOpts{TextDirection: ui.TextDirectionLTR, Thickness: foundation.Some(scrollbarThick),
			MinLength: foundation.Some(minThumbExtent), ScrollbarOrientation: ScrollbarOrientationTop, Padding: painting.EdgeInsetsZero}
		color := ui.NewColor(0xFF000000)
		fade := animation.AlwaysComplete
		if change != nil {
			change(&o, &color, &fade)
		}
		return NewScrollbarPainter(color, fade, o)
	}
	painter := create(nil)
	if painter.ShouldRepaint(create(nil)) {
		t.Error("an identical painter should not repaint")
	}
	tests := []struct {
		name   string
		change func(o *ScrollbarPainterOpts, color *ui.Color, fade *animation.Animation[float64])
	}{
		{"color", func(_ *ScrollbarPainterOpts, c *ui.Color, _ *animation.Animation[float64]) {
			*c = ui.NewColor(0xFFFFFFFF)
		}},
		{"fade animation", func(_ *ScrollbarPainterOpts, _ *ui.Color, f *animation.Animation[float64]) {
			*f = animation.AlwaysDismissed
		}},
		{"track color", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.TrackColor = ui.NewColor(0xFFFFFFFF)
		}},
		{"track border color", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.TrackBorderColor = ui.NewColor(0xFFFFFFFF)
		}},
		{"text direction", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.TextDirection = ui.TextDirectionRTL
		}},
		{"thickness", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.Thickness = foundation.Some(scrollbarThick + 1)
		}},
		{"padding", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.Padding = painting.NewEdgeInsetsAll(1)
		}},
		{"main axis margin", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) { o.MainAxisMargin = 1 }},
		{"cross axis margin", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) { o.CrossAxisMargin = 1 }},
		{"radius", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.Radius = foundation.Some(ui.NewRadiusCircular(1))
		}},
		{"track radius", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.TrackRadius = foundation.Some(ui.NewRadiusCircular(1))
		}},
		{"shape", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.Shape = painting.NewCircleBorder(painting.CircleBorderOpts{Side: painting.BorderSide{Width: 2}})
		}},
		{"min length", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.MinLength = foundation.Some(minThumbExtent + 1)
		}},
		{"min overscroll length", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.MinOverscrollLength = foundation.Some(1.0)
		}},
		{"orientation", func(o *ScrollbarPainterOpts, _ *ui.Color, _ *animation.Animation[float64]) {
			o.ScrollbarOrientation = ScrollbarOrientationBottom
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !painter.ShouldRepaint(create(tt.change)) {
				t.Errorf("a different %s should repaint", tt.name)
			}
		})
	}
}
