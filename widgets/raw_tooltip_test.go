package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/raw_tooltip_test.dart.

const tooltipText = "TIP"

type tooltipModeOpts struct {
	touchDelay          foundation.Opt[time.Duration]
	onTriggered         foundation.VoidCallback
	disableTapToDismiss bool
	ignorePointer       bool
}

// below24 is the positionDelegate of the Dart tests: a box 24 pixels from the target.
func below24(c TooltipPositionContext) ui.Offset {
	return painting.PositionDependentBox(painting.PositionDependentBoxOpts{
		Size: c.OverlaySize, ChildSize: c.TooltipSize, Target: c.Target, PreferBelow: c.PreferBelow(), VerticalOffset: 24,
	})
}

// setWidgetForTooltipMode is setWidgetForTooltipMode of the Dart test.
func setWidgetForTooltipMode(ts *gluttertest.Tester, mode TooltipTriggerMode, o tooltipModeOpts) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewCenter(NewRawTooltip(foundation.Some(tooltipText),
		func(BuildContext, animation.Animation[float64]) Widget { return ltrText(tooltipText) },
		NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
		RawTooltipOpts{TriggerMode: mode, TouchDelay: o.touchDelay, OnTriggered: o.onTriggered,
			DisableTapToDismiss: o.disableTapToDismiss, IgnorePointer: o.ignorePointer, PositionDelegate: below24}))}))
}

func tooltipShown(ts *gluttertest.Tester) bool { return ts.Exists(gluttertest.ByText(tooltipText)) }

func testGestureTap(ts *gluttertest.Tester, f gluttertest.Finder) {
	ts.Tap(f)
	ts.Pump(10 * time.Millisecond)
}

func testGestureLongPress(ts *gluttertest.Tester, f gluttertest.Finder) {
	g := ts.StartGesture(ts.Center(f))
	ts.Pump()
	ts.Pump(gestures.LongPressTimeout)
	g.Up()
	ts.Pump()
}

func TestRawTooltipStaysAfterLongPress(t *testing.T) {
	ts := gluttertest.New(t)
	setWidgetForTooltipMode(ts, TooltipTriggerModeLongPress, tooltipModeOpts{ignorePointer: true})
	tooltip := gluttertest.ByType[*RawTooltip]()
	g := ts.StartGesture(ts.Center(tooltip))
	ts.Pump(gestures.LongPressTimeout)
	ts.Pump(10 * time.Millisecond)
	if !tooltipShown(ts) {
		t.Fatal("long press did not reveal the tooltip")
	}
	g.Up()
	// A tap hides the tooltip, since it is not a long press.
	ts.Tap(tooltip)
	ts.Pump()
	ts.PumpAndSettle()
	if tooltipShown(ts) {
		t.Fatal("tap did not hide the tooltip")
	}
	g = ts.StartGesture(ts.Center(tooltip))
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if tooltipShown(ts) {
		t.Fatal("tooltip shown before the long press timeout")
	}
	ts.Pump(gestures.LongPressTimeout)
	ts.Pump(10 * time.Millisecond)
	if !tooltipShown(ts) {
		t.Fatal("second long press did not reveal the tooltip")
	}
	// Keep holding the long press: the tooltip stays.
	ts.Pump(gestures.LongPressTimeout)
	if !tooltipShown(ts) {
		t.Fatal("tooltip went away while held")
	}
	g.Up()
}

func TestRawTooltipDismissedAfterLongPressAndTouchDelay(t *testing.T) {
	ts := gluttertest.New(t)
	touchDelay := 3 * time.Second
	setWidgetForTooltipMode(ts, TooltipTriggerModeLongPress, tooltipModeOpts{touchDelay: foundation.Some(touchDelay)})
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*RawTooltip]()))
	ts.Pump(gestures.LongPressTimeout)
	ts.Pump(10 * time.Millisecond)
	if !tooltipShown(ts) {
		t.Fatal("long press did not reveal the tooltip")
	}
	g.Up()
	ts.Pump(touchDelay)
	ts.Pump(10 * time.Millisecond)
	if tooltipShown(ts) {
		t.Fatal("tooltip stayed after the touch delay")
	}
}

func TestRawTooltipDismissedAfterTapAndTouchDelay(t *testing.T) {
	ts := gluttertest.New(t)
	touchDelay := 3 * time.Second
	setWidgetForTooltipMode(ts, TooltipTriggerModeTap, tooltipModeOpts{touchDelay: foundation.Some(touchDelay)})
	if tooltipShown(ts) {
		t.Fatal("tooltip shown at start")
	}
	testGestureTap(ts, gluttertest.ByType[*RawTooltip]())
	if !tooltipShown(ts) {
		t.Fatal("tap did not reveal the tooltip")
	}
	ts.Pump(touchDelay)
	ts.Pump(10 * time.Millisecond)
	if tooltipShown(ts) {
		t.Fatal("tooltip stayed after the touch delay")
	}
}

func TestRawTooltipTapToDismiss(t *testing.T) {
	tests := []struct {
		name        string
		disableTap  bool
		wantVisible bool
	}{
		{"dismissed immediately", false, false},
		{"not dismissed when disabled", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			setWidgetForTooltipMode(ts, TooltipTriggerModeTap, tooltipModeOpts{disableTapToDismiss: tc.disableTap})
			testGestureTap(ts, gluttertest.ByType[*RawTooltip]())
			if !tooltipShown(ts) {
				t.Fatal("tap did not reveal the tooltip")
			}
			testGestureTap(ts, gluttertest.ByText(tooltipText))
			if got := tooltipShown(ts); got != tc.wantVisible {
				t.Fatalf("tooltip visible = %v, want %v", got, tc.wantVisible)
			}
		})
	}
}

func TestRawTooltipTriggerModes(t *testing.T) {
	tests := []struct {
		name          string
		mode          TooltipTriggerMode
		wantAfterTap  bool
		wantAfterLong bool
	}{
		{"tap", TooltipTriggerModeTap, true, true},
		{"longPress", TooltipTriggerModeLongPress, false, true},
		{"manual", TooltipTriggerModeManual, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			setWidgetForTooltipMode(ts, tc.mode, tooltipModeOpts{})
			tooltip := gluttertest.ByType[*RawTooltip]()
			testGestureTap(ts, tooltip)
			if got := tooltipShown(ts); got != tc.wantAfterTap {
				t.Fatalf("shown after tap = %v, want %v", got, tc.wantAfterTap)
			}
			ts.Pump(5 * time.Second)
			ts.PumpAndSettle()
			testGestureLongPress(ts, tooltip)
			if got := tooltipShown(ts); got != tc.wantAfterLong {
				t.Fatalf("shown after long press = %v, want %v", got, tc.wantAfterLong)
			}
		})
	}
}

func TestRawTooltipOnTriggeredIsCalledWhenTheTooltipTriggers(t *testing.T) {
	ts := gluttertest.New(t)
	called := false
	setWidgetForTooltipMode(ts, TooltipTriggerModeLongPress, tooltipModeOpts{onTriggered: func() { called = true }})
	testGestureLongPress(ts, gluttertest.ByType[*RawTooltip]())
	if !called {
		t.Fatal("OnTriggered was not called for the long press")
	}
	called = false
	setWidgetForTooltipMode(ts, TooltipTriggerModeTap, tooltipModeOpts{onTriggered: func() { called = true }})
	testGestureTap(ts, gluttertest.ByType[*RawTooltip]())
	if !called {
		t.Fatal("OnTriggered was not called for the tap")
	}
}

func TestRawTooltipEnsureTooltipVisibleOnAnUnmountedStateReturnsFalse(t *testing.T) {
	ts := gluttertest.New(t)
	setWidgetForTooltipMode(ts, TooltipTriggerModeLongPress, tooltipModeOpts{})
	state := gluttertest.StateOf[*RawTooltipState](ts, gluttertest.ByType[*RawTooltip]())
	if !state.EnsureTooltipVisible() {
		t.Fatal("EnsureTooltipVisible returned false on a mounted state")
	}
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCenter(NewSizedBox(nil))}))
	if state.EnsureTooltipVisible() {
		t.Fatal("EnsureTooltipVisible returned true on an unmounted state")
	}
}

func pumpHoverTooltip(ts *gluttertest.Tester) gluttertest.Finder {
	ts.Hover(ui.Offset{Dx: 1, Dy: 1})
	ts.Pump()
	ts.Hover(ui.Offset{})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewCenter(NewRawTooltip(foundation.Some(tooltipText),
		func(BuildContext, animation.Animation[float64]) Widget { return ltrText(tooltipText) },
		NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
		RawTooltipOpts{PositionDelegate: below24}))}))
	return gluttertest.ByType[*RawTooltip]()
}

func TestRawTooltipShowsAndHidesWhenHovered(t *testing.T) {
	ts := gluttertest.New(t)
	tooltip := pumpHoverTooltip(ts)
	ts.Hover(ui.Offset{})
	ts.Pump()
	ts.Hover(ts.Center(tooltip))
	ts.Pump()
	ts.Pump(0)
	if !tooltipShown(ts) {
		t.Fatal("hover did not show the tooltip")
	}
	// It stays while the mouse is over the widget.
	ts.Pump(24 * time.Hour)
	ts.PumpAndSettle()
	if !tooltipShown(ts) {
		t.Fatal("tooltip went away under the mouse")
	}
	ts.Hover(ui.Offset{})
	ts.Pump()
	ts.PumpAndSettle()
	if tooltipShown(ts) {
		t.Fatal("tooltip stayed after the mouse left")
	}
}

func TestRawTooltipCanBeDismissedByEscape(t *testing.T) {
	ts := gluttertest.New(t)
	tooltip := pumpHoverTooltip(ts)
	ts.Hover(ts.Center(tooltip))
	ts.Pump()
	ts.Pump(0)
	if !tooltipShown(ts) {
		t.Fatal("hover did not show the tooltip")
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	ts.PumpAndSettle()
	if tooltipShown(ts) {
		t.Fatal("Escape did not dismiss the tooltip")
	}
}

func TestRawTooltipDismissAllDismissesHoveredTooltips(t *testing.T) {
	ts := gluttertest.New(t)
	tooltip := pumpHoverTooltip(ts)
	ts.Hover(ts.Center(tooltip))
	ts.Pump()
	ts.Pump(0)
	ts.Pump(24 * time.Hour)
	ts.PumpAndSettle()
	if !tooltipShown(ts) {
		t.Fatal("hover did not show the tooltip")
	}
	bc := ts.Element(tooltip)
	if !RawTooltipDismissAll(bc) {
		t.Fatal("RawTooltipDismissAll returned false with an open tooltip")
	}
	ts.PumpAndSettle()
	if tooltipShown(ts) {
		t.Fatal("tooltip still shown after RawTooltipDismissAll")
	}
	if RawTooltipDismissAll(bc) {
		t.Fatal("RawTooltipDismissAll returned true with no open tooltip")
	}
}
