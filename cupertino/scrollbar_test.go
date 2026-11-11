package cupertino_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/scrollbar_test.dart. They read what the
// scrollbar paints from its painter instead of with the `paints` matcher. Not ported: the horizontal cases
// (the same code with the axes swapped), the reverse cases, the cases of the visibility of the thumb (they
// test RawScrollbar, which has its tests in package widgets), and the case of a drag with a pointer scroll.

const (
	scrollbarTimeToFade        = 1200 * time.Millisecond
	scrollbarFadeDuration      = 250 * time.Millisecond
	scrollbarResizeDuration    = 100 * time.Millisecond
	scrollbarLongPressDuration = 100 * time.Millisecond
)

var scrollbarColor = cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x59000000), ui.NewColor(0x80FFFFFF))

// thumb is the rounded rectangle that the scrollbar paints, with its color.
func thumb(t *testing.T, ts *gluttertest.Tester) (ui.RRect, ui.Color) {
	t.Helper()
	state := gluttertest.StateOf[*widgets.RawScrollbarState](ts, gluttertest.ByType[*widgets.RawScrollbar]())
	recorder := ui.NewPictureRecorder()
	state.ScrollbarPainter().Paint(ui.NewCanvas(recorder), ts.Size(gluttertest.ByType[*widgets.RawScrollbar]()))
	var rrect ui.RRect
	var color ui.Color
	n := 0
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		if op, ok := op.(ui.OpDrawRRect); ok {
			rrect, color = op.RRect, op.Paint.Color()
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the scrollbar painted %d rounded rectangles, want 1", n)
	}
	return rrect, color
}

func scrollbarApp(controller widgets.ScrollController, brightness ui.Brightness, opts cupertino.CupertinoScrollbarOpts, width, height float64) widgets.Widget {
	data := widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, PlatformBrightness: brightness}
	var scrollbar widgets.Widget = cupertino.NewCupertinoScrollbar(widgets.NewSingleChildScrollView(
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)})), opts)
	if controller != nil {
		scrollbar = widgets.NewPrimaryScrollController(controller, scrollbar)
	}
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(data, scrollbar))
}

func TestCupertinoScrollbarNeverGoesAwayUntilFingerLift(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(scrollbarApp(nil, ui.BrightnessLight, cupertino.CupertinoScrollbarOpts{}, 4000, 4000))
	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SingleChildScrollView]()))
	gesture.MoveBy(ui.Offset{Dy: -10})
	ts.Pump()
	ts.Pump(500 * time.Millisecond) // The scrollbar is fully showing.
	if _, color := thumb(t, ts); color != scrollbarColor.Color {
		t.Errorf("the thumb is %v, want %v", color, scrollbarColor.Color)
	}
	ts.Pump(3 * time.Second)
	ts.Pump(3 * time.Second) // Still there.
	if _, color := thumb(t, ts); color != scrollbarColor.Color {
		t.Errorf("after 6 seconds the thumb is %v, want %v", color, scrollbarColor.Color)
	}
	gesture.Up()
	ts.Pump(scrollbarTimeToFade)
	ts.Pump(scrollbarFadeDuration / 2)
	// The opacity is going down now.
	if _, color := thumb(t, ts); color != scrollbarColor.Color.WithAlpha(69) {
		t.Errorf("the thumb fades to %v, want %v", color, scrollbarColor.Color.WithAlpha(69))
	}
}

func TestCupertinoScrollbarDarkMode(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(scrollbarApp(nil, ui.BrightnessLight, cupertino.CupertinoScrollbarOpts{}, 4000, 4000))
	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SingleChildScrollView]()))
	gesture.MoveBy(ui.Offset{Dy: 10})
	ts.Pump()
	ts.PumpAndSettle()
	if _, color := thumb(t, ts); color != scrollbarColor.Color {
		t.Errorf("the thumb is %v, want %v", color, scrollbarColor.Color)
	}
	ts.PumpWidget(scrollbarApp(nil, ui.BrightnessDark, cupertino.CupertinoScrollbarOpts{}, 4000, 4000))
	ts.Pump()
	if _, color := thumb(t, ts); color != scrollbarColor.DarkColor {
		t.Errorf("the dark thumb is %v, want %v", color, scrollbarColor.DarkColor)
	}
	gesture.Up()
	ts.PumpAndSettle()
}

func TestCupertinoScrollbarThumbCanBeDraggedWithLongPress(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := widgets.NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp(controller, ui.BrightnessLight, cupertino.CupertinoScrollbarOpts{}, 4000, 4000))
	if controller.Offset() != 0 {
		t.Fatalf("offset = %v, want 0", controller.Offset())
	}
	const scrollAmount = 10.0
	scroll := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SingleChildScrollView]()))
	scroll.MoveBy(ui.Offset{Dy: -scrollAmount}) // Scroll down by swiping up.
	ts.Pump()
	ts.Pump(500 * time.Millisecond)
	if _, color := thumb(t, ts); color != scrollbarColor.Color || controller.Offset() != scrollAmount {
		t.Fatalf("the thumb is %v and the offset %v, want %v and %v", color, controller.Offset(), scrollbarColor.Color, scrollAmount)
	}
	scroll.Up()
	ts.Pump()

	haptics := 0
	ts.Binding.HapticFeedback().OnPerform = func(ui.HapticFeedbackKind) { haptics++ }
	// A long press on the thumb makes a vibration after the thumb has resized.
	drag := ts.StartGesture(ui.Offset{Dx: 796, Dy: 50})
	ts.Pump(scrollbarLongPressDuration)
	if haptics != 0 {
		t.Errorf("%d haptic feedbacks before the thumb resized, want 0", haptics)
	}
	ts.Pump(scrollbarResizeDuration)
	ts.Pump(time.Millisecond) // Allows the haptic feedback some slack.
	if haptics != 1 {
		t.Errorf("%d haptic feedbacks after the thumb resized, want 1", haptics)
	}
	// Drag the thumb down to scroll down.
	drag.MoveBy(ui.Offset{Dy: scrollAmount})
	ts.Pump(100 * time.Millisecond)
	drag.Up()
	ts.PumpAndSettle()
	// The view has scrolled more than it would have by a swipe of the same distance, and the thumb is still shown.
	if controller.Offset() <= scrollAmount*2 {
		t.Errorf("the offset is %v, want more than %v", controller.Offset(), scrollAmount*2)
	}
	if _, color := thumb(t, ts); color != scrollbarColor.Color {
		t.Errorf("the thumb is %v, want %v", color, scrollbarColor.Color)
	}
	ts.Pump(scrollbarTimeToFade)
	ts.Pump(scrollbarFadeDuration)
}

func TestCupertinoScrollbarChangesThicknessAndRadiusWhenDragged(t *testing.T) {
	useAndroid(t)
	const (
		thickness              = 20.0
		thicknessWhileDragging = 40.0
		radius                 = 10.0
		radiusWhileDragging    = 20.0
		inset                  = 3.0
		scaleFactor            = 2.0
		screenWidth            = 800.0
		screenHeight           = 600.0
	)
	ts := gluttertest.New(t)
	controller := widgets.NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(scrollbarApp(controller, ui.BrightnessLight, cupertino.CupertinoScrollbarOpts{
		Thickness: foundation.Some(thickness), ThicknessWhileDragging: foundation.Some(thicknessWhileDragging),
		Radius: foundation.Some(ui.NewRadiusCircular(radius)), RadiusWhileDragging: foundation.Some(ui.NewRadiusCircular(radiusWhileDragging)),
	}, screenWidth*scaleFactor, screenHeight*scaleFactor))
	// Scroll a bit to show the thumb, then undo the scroll to put the thumb at the top.
	scroll := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SingleChildScrollView]()))
	scroll.MoveBy(ui.Offset{Dy: -10})
	ts.Pump()
	ts.Pump(500 * time.Millisecond)
	scroll.MoveBy(ui.Offset{Dy: 10})
	ts.Pump()
	scroll.Up()
	ts.Pump()
	// A long press on the thumb makes it grow.
	drag := ts.StartGesture(ui.Offset{Dx: 780, Dy: 50})
	ts.Pump(scrollbarLongPressDuration)
	want := func(thickness, radius float64) ui.RRect {
		return ui.NewRRectFromRectAndRadius(ui.NewRectFromLTWH(screenWidth-inset-thickness, inset, thickness, (screenHeight-2*inset)/scaleFactor), ui.NewRadiusCircular(radius))
	}
	check := func(when string, thickness, radius float64) {
		t.Helper()
		got, _ := thumb(t, ts)
		if exp := want(thickness, radius); got != exp {
			t.Errorf("%s: the thumb is %v, want %v", when, got, exp)
		}
	}
	check("at the start", thickness, radius)
	ts.Pump(scrollbarResizeDuration / 2)
	check("halfway", (thickness+thicknessWhileDragging)/2, (radius+radiusWhileDragging)/2)
	ts.Pump(scrollbarResizeDuration / 2)
	check("at the end", thicknessWhileDragging, radiusWhileDragging)
	drag.Up()
	ts.PumpAndSettle()
	ts.Pump(scrollbarTimeToFade)
	ts.Pump(scrollbarFadeDuration)
}

func TestCupertinoScrollbarDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(widgets.MediaQueryData{},
		widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoScrollbar(widgets.NewSingleChildScrollView(widgets.NewSizedBox(nil))))))))
	if size := ts.Size(gluttertest.ByType[*cupertino.CupertinoScrollbar]()); size != (ui.Size{}) {
		t.Errorf("size = %v, want zero", size)
	}
}

func TestCupertinoScrollbarThicknessMustBeFinite(t *testing.T) {
	skipUnlessDebug(t)
	inf := foundation.Some(math.Inf(1))
	for name, opts := range map[string]cupertino.CupertinoScrollbarOpts{
		"thickness":                {Thickness: inf},
		"thickness while dragging": {ThicknessWhileDragging: inf},
	} {
		t.Run(name, func(t *testing.T) {
			err := recoverError(func() { cupertino.NewCupertinoScrollbar(widgets.NewSizedBox(nil), opts) })
			checkErrorTag(t, err, "NewCupertinoScrollbar.CupertinoScrollbarThicknessInfinite")
		})
	}
}
