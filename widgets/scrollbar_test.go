// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from the widget tests of packages/flutter/test/widgets/scrollbar_test.dart.
// They scroll a Scrollable of one 1000 or 4000 pixel sliver instead of a
// SingleChildScrollView (step 9.3), and read what the scrollbar paints from its
// painter instead of with the `paints` matcher.

const (
	scrollbarTimeToFade  = 600 * time.Millisecond
	scrollbarFadeTime    = 300 * time.Millisecond
	defaultThumbColor    = uint32(0x66BCBCBC)
	scrollbarViewportExt = 600.0
)

type scrollbarApp struct {
	controller ScrollController
	scrollbar  RawScrollbarOpts
	width      float64
	height     float64
	noPrimary  bool
	scrollable ScrollableOpts
}

func (a scrollbarApp) widget() Widget {
	if a.width == 0 {
		a.width = 1000
	}
	if a.height == 0 {
		a.height = 1000
	}
	a.scrollable.Controller = a.controller
	scrollable := testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(a.width), Height: foundation.Some(a.height)})}, listOpts{ScrollableOpts: a.scrollable})
	var scrollbar Widget = NewRawScrollbar(scrollable, a.scrollbar)
	if !a.noPrimary && a.controller != nil {
		scrollbar = NewPrimaryScrollController(a.controller, scrollbar)
	}
	return ltr(NewMediaQuery(MediaQueryData{}, scrollbar))
}

// thumb is what the scrollbar paints: the track, the thumb and its color.
func thumb(t *testing.T, ts *gluttertest.Tester) (track, thumb ui.Rect, color ui.Color) {
	t.Helper()
	state := gluttertest.StateOf[*RawScrollbarState](ts, gluttertest.ByType[*RawScrollbar]())
	painted := paintScrollbar(state.ScrollbarPainter(), ts.Size(gluttertest.ByType[*RawScrollbar]()))
	if len(painted.rects) != 2 {
		t.Fatalf("the scrollbar painted %d rectangles, want the track and the thumb", len(painted.rects))
	}
	return painted.rects[0], painted.rects[1], painted.colors[1]
}

func expectThumb(t *testing.T, ts *gluttertest.Tester, wantTrack, wantThumb ui.Rect, wantColor ui.Color) {
	t.Helper()
	track, thumb, color := thumb(t, ts)
	if track != wantTrack || thumb != wantThumb || color != wantColor {
		t.Errorf("track %v thumb %v color %v, want track %v thumb %v color %v", track, thumb, color, wantTrack, wantThumb, wantColor)
	}
}

func ltrb(l, t, r, b float64) ui.Rect { return ui.NewRectFromLTRB(l, t, r, b) }

func TestRawScrollbarTappingTheTrackPagesTheScrollView(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller}}.widget())
	ts.PumpAndSettle()
	color := ui.NewColor(defaultThumbColor)
	track := ltrb(794, 0, 800, 600)
	if controller.Offset() != 0 {
		t.Fatalf("offset = %v", controller.Offset())
	}
	expectThumb(t, ts, track, ltrb(794, 0, 800, 360), color)

	ts.TapAt(ui.Offset{Dx: 796, Dy: 550}) // on the track area below the thumb
	ts.PumpAndSettle()
	if controller.Offset() != 400 {
		t.Errorf("offset = %v, want 400", controller.Offset())
	}
	expectThumb(t, ts, track, ltrb(794, 240, 800, 600), color)

	ts.TapAt(ui.Offset{Dx: 796, Dy: 50}) // on the track area above the thumb
	ts.PumpAndSettle()
	if controller.Offset() != 0 {
		t.Errorf("offset = %v, want 0", controller.Offset())
	}
	expectThumb(t, ts, track, ltrb(794, 0, 800, 360), color)
}

func TestRawScrollbarNeverGoesAwayUntilFingerLift(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000}.widget())
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	g.MoveBy(ui.Offset{Dy: -20})
	ts.Pump()
	ts.Pump(500 * time.Millisecond) // the scrollbar is fully showing
	track, color := ltrb(794, 0, 800, 600), ui.NewColor(defaultThumbColor)
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color)

	ts.Pump(3 * time.Second)
	ts.Pump(3 * time.Second) // still there
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color)

	g.Up()
	ts.Pump(scrollbarTimeToFade)
	ts.Pump(scrollbarFadeTime / 2) // the opacity is going down now
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), ui.NewColor(0x4fbcbcbc))
}

func TestRawScrollbarDoesNotFadeAwayWhileHovering(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000}.widget())
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	g.MoveBy(ui.Offset{Dy: -20})
	ts.Pump()
	ts.Pump(500 * time.Millisecond)
	track, color := ltrb(794, 0, 800, 600), ui.NewColor(defaultThumbColor)
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color)

	ts.Hover(ui.Offset{Dx: 790, Dy: 5}) // over the thumb, to keep the scrollbar from fading out
	g.Up()
	ts.Pump(3 * time.Second)
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color) // still there
}

func TestRawScrollbarFadesBackInWhenHoveringOverTheKnownTrackArea(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000}.widget())
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	g.MoveBy(ui.Offset{Dy: -20})
	ts.Pump()
	ts.Pump(500 * time.Millisecond)
	track, color := ltrb(794, 0, 800, 600), ui.NewColor(defaultThumbColor)
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color)
	g.Up()
	ts.Pump(scrollbarTimeToFade)
	ts.Pump(scrollbarFadeTime / 2)
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), ui.NewColor(0x4fbcbcbc)) // fading out

	// Hover over the scrollbar with a mouse to bring the opacity back up.
	ts.Hover(ui.Offset{Dx: 794, Dy: 5})
	ts.PumpAndSettle()
	expectThumb(t, ts, track, ltrb(794, 3, 800, 93), color)
}

func TestRawScrollbarShowsOnHoverWithoutScrollingFirst(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000}.widget())
	ts.Pump()
	// Even though nothing scrolled, the ScrollMetricsNotification informed the
	// hit testing of the scrollbar.
	ts.Hover(ui.Offset{Dx: 794, Dy: 5})
	ts.PumpAndSettle()
	expectThumb(t, ts, ltrb(794, 0, 800, 600), ltrb(794, 0, 800, 90), ui.NewColor(defaultThumbColor))
}

func TestRawScrollbarThumbCanBeDragged(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000,
		scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller}}.widget())
	ts.PumpAndSettle()
	track, color := ltrb(794, 0, 800, 600), ui.NewColor(defaultThumbColor)
	expectThumb(t, ts, track, ltrb(794, 0, 800, 90), color)

	// Drag the thumb down to scroll down.
	const scrollAmount = 10.0
	g := ts.StartGesture(ui.Offset{Dx: 797, Dy: 45})
	ts.PumpAndSettle()
	g.MoveBy(ui.Offset{Dy: scrollAmount})
	ts.PumpAndSettle()
	g.Up()
	ts.PumpAndSettle()
	// The view has scrolled more than it would have by a swipe gesture of the same distance.
	if controller.Offset() <= scrollAmount*2 {
		t.Errorf("offset = %v, want more than %v", controller.Offset(), scrollAmount*2)
	}
	expectThumb(t, ts, track, ltrb(794, 10, 800, 100), color)
}

func TestRawScrollbarThumbCannotBeDraggedIntoOverscrollIfThePhysicsDoNotAllow(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000,
		scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller}}.widget())
	ts.PumpAndSettle()
	g := ts.StartGesture(ui.Offset{Dx: 797, Dy: 45})
	ts.PumpAndSettle()
	g.MoveBy(ui.Offset{Dy: -10})
	ts.PumpAndSettle()
	// The physics should not have allowed us to enter overscroll.
	if controller.Offset() != 0 {
		t.Errorf("offset = %v, want 0", controller.Offset())
	}
	expectThumb(t, ts, ltrb(794, 0, 800, 600), ltrb(794, 0, 800, 90), ui.NewColor(defaultThumbColor))
	g.Up()
}

func TestRawScrollbarThumbCanBeDraggedInReverse(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000,
		scrollbar:  RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller},
		scrollable: ScrollableOpts{AxisDirection: painting.AxisDirectionUp}}.widget())
	ts.PumpAndSettle()
	// The thumb starts at the bottom of the track when the content is reversed.
	expectThumb(t, ts, ltrb(794, 0, 800, 600), ltrb(794, 510, 800, 600), ui.NewColor(defaultThumbColor))
	g := ts.StartGesture(ui.Offset{Dx: 797, Dy: 555})
	ts.PumpAndSettle()
	g.MoveBy(ui.Offset{Dy: -10})
	ts.PumpAndSettle()
	g.Up()
	ts.PumpAndSettle()
	if controller.Offset() <= 20 {
		t.Errorf("offset = %v, want more than 20", controller.Offset())
	}
	expectThumb(t, ts, ltrb(794, 0, 800, 600), ltrb(794, 500, 800, 590), ui.NewColor(defaultThumbColor))
}

func TestRawScrollbarThumbVisibilityNeedsAScrollPosition(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController() // never attached to the scrollable
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{noPrimary: true, scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller,
		ThumbColor: foundation.Some(ui.NewColor(0x11111111))}}.widget())
	ts.PumpAndSettle()
	_ = requireErrorTag(t, takeFirst(t, ts), "ScrollbarNoScrollPosition")
}

func TestRawScrollbarInteractiveScrollbarsNeedAValidScrollController(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	primary, own := NewScrollController(), NewScrollController()
	defer primary.Dispose()
	defer own.Dispose()
	scrollable := testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1000.0), Height: foundation.Some(1000.0)})}, listOpts{ScrollableOpts: ScrollableOpts{Controller: own}})
	ts.PumpWidget(ltr(NewMediaQuery(MediaQueryData{}, NewPrimaryScrollController(primary, NewRawScrollbar(scrollable)))))
	ts.PumpAndSettle()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("the scrollbar is not visible and cannot be interacted with, but %v", errs)
	}
	// Scroll to trigger the scrollbar to come into view.
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	g.MoveBy(ui.Offset{Dy: -20})
	_ = requireErrorTag(t, takeFirst(t, ts), "ScrollbarNoScrollPosition")
}

func TestRawScrollbarsNeedASingleScrollPosition(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	column := func() Widget {
		return NewSizedBox(NewRawScrollbar(testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(4000.0)})},
			listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}}), RawScrollbarOpts{Controller: controller}),
			SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(600.0)})
	}
	ts.PumpWidget(ltr(NewMediaQuery(MediaQueryData{}, NewPrimaryScrollController(controller, NewRow([]Widget{column(), column()})))))
	ts.PumpAndSettle()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("the scrollbar is not visible and cannot be interacted with, but %v", errs)
	}
	g := ts.StartGesture(ts.Center(gluttertest.First(gluttertest.ByType[*Scrollable]())))
	g.MoveBy(ui.Offset{Dy: -20})
	_ = requireErrorTag(t, takeFirst(t, ts), "ScrollbarMultipleScrollPositions")
}

func TestRawScrollbarNeedsAController(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(scrollbarApp{noPrimary: true, scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true)}}.widget())
	ts.PumpAndSettle()
	_ = requireErrorTag(t, takeFirst(t, ts), "ScrollbarControllerMissing")
}

func TestRawScrollbarOptionsMustBeConsistent(t *testing.T) {
	skipUnlessDebug(t)
	child := NewSizedBox(nil)
	tests := []struct {
		name string
		o    RawScrollbarOpts
	}{
		{"a track without a thumb", RawScrollbarOpts{ThumbVisibility: foundation.Some(false), TrackVisibility: foundation.Some(true)}},
		{"a negative minimum thumb length", RawScrollbarOpts{MinThumbLength: foundation.Some(-1.0)}},
		{"an overscroll length above the thumb length", RawScrollbarOpts{MinThumbLength: foundation.Some(10.0), MinOverscrollLength: foundation.Some(11.0)}},
		{"a negative overscroll length", RawScrollbarOpts{MinOverscrollLength: foundation.Some(-1.0)}},
		{"a radius and a shape", RawScrollbarOpts{Radius: foundation.Some(ui.NewRadiusCircular(1)), Shape: painting.NewCircleBorder()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTag(t, "RawScrollbarInvalid", func() { NewRawScrollbar(child, tt.o) })
		})
	}
}

func TestRawScrollbarThumbDragCannotStartTwice(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp{controller: controller, width: 4000, height: 4000,
		scrollbar: RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller}}.widget())
	ts.PumpAndSettle()
	state := gluttertest.StateOf[*RawScrollbarState](ts, gluttertest.ByType[*RawScrollbar]())
	state.HandleThumbPress() // the press that comes before the start of a drag
	state.HandleThumbPressStart(ui.Offset{Dx: 797, Dy: 45})
	requireTag(t, "RawScrollbarDragInvalid", func() { state.HandleThumbPressStart(ui.Offset{Dx: 797, Dy: 45}) })
	state.HandleThumbPressEnd(ui.Offset{Dx: 797, Dy: 45}, gestures.Velocity{})
}
