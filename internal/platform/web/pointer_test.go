package web

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// These tests port pointer_binding_test.dart of Flutter web (the cases of the pointer adapter, the
// button sanitizer and the pointer data converter) to the plain Go PointerInput.

// op is what happens to the input.
type op int

const (
	opDown op = iota
	opMove
	opUp
	opLeave
	opCancel
)

func mouseEv(button, buttons int, x, y float64) PointerEvent {
	return PointerEvent{PointerType: "mouse", PointerID: 1, Button: button, Buttons: buttons, X: x, Y: y}
}

func touchEv(id int, buttons int, x, y float64) PointerEvent {
	return PointerEvent{PointerType: "touch", PointerID: id, Buttons: buttons, X: x, Y: y}
}

// noButtonChange is PointerEvent.button for an event that changes no button.
const noButtonChange = -1

type pstep struct {
	op   op
	ev   PointerEvent
	want []string
}

// fmtData describes pointer data in one line: the change (with * for synthesized), the device, the
// pointer identifier, the position, the delta and the buttons. short leaves the device, the
// identifier and the delta out.
func fmtData(d ui.PointerData, short bool) string {
	name := strings.ToLower(strings.TrimPrefix(d.Change.String(), "PointerChange"))
	if d.Synthesized {
		name += "*"
	}
	if short {
		return fmt.Sprintf("%s @%g,%g b%d", name, d.PhysicalX, d.PhysicalY, d.Buttons)
	}
	return fmt.Sprintf("%s dev%d id%d @%g,%g d%g,%g b%d", name, d.Device, d.PointerIdentifier, d.PhysicalX, d.PhysicalY, d.PhysicalDeltaX, d.PhysicalDeltaY, d.Buttons)
}

func run(in *PointerInput, o op, e PointerEvent, dpr float64) []ui.PointerData {
	switch o {
	case opDown:
		return in.Down(e, dpr)
	case opMove:
		return in.Move(e, dpr)
	case opUp:
		return in.Up(e, dpr)
	case opLeave:
		return in.Leave(e, dpr)
	}
	return in.Cancel(e, dpr)
}

func TestPointerInputSequences(t *testing.T) {
	tests := []struct {
		name  string
		dpr   float64
		short bool
		steps []pstep
	}{
		{
			name: "does create an add event if got a pointerdown", dpr: 1,
			steps: []pstep{{opDown, mouseEv(0, 1, 5, 6), []string{"add* dev-1 id1 @5,6 d0,0 b1", "down dev-1 id1 @5,6 d0,0 b1"}}},
		},
		{
			name: "creates an add event if the first pointer activity is a hover", dpr: 1,
			steps: []pstep{{opMove, mouseEv(noButtonChange, 0, 5, 6), []string{"add* dev-1 id0 @5,6 d0,0 b0", "hover dev-1 id0 @5,6 d0,0 b0"}}},
		},
		{
			name: "sends a pointermove instead of the second pointerdown in a row", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(0, 1, 10, 10), []string{"add* @10,10 b1", "down @10,10 b1"}},
				{opDown, mouseEv(0, 1, 20, 20), []string{"move @20,20 b1"}},
			},
		},
		{
			name: "does calculate delta and pointer identifier correctly", dpr: 2,
			steps: []pstep{
				{opMove, mouseEv(noButtonChange, 0, 10, 10), []string{"add* dev-1 id0 @20,20 d0,0 b0", "hover dev-1 id0 @20,20 d0,0 b0"}},
				{opMove, mouseEv(noButtonChange, 0, 20, 20), []string{"hover dev-1 id0 @40,40 d20,20 b0"}},
				{opDown, mouseEv(0, 1, 20, 20), []string{"down dev-1 id1 @40,40 d0,0 b1"}},
				{opMove, mouseEv(noButtonChange, 1, 40, 30), []string{"move dev-1 id1 @80,60 d40,20 b1"}},
				{opUp, mouseEv(0, 0, 40, 30), []string{"up dev-1 id1 @80,60 d0,0 b0"}},
				{opMove, mouseEv(noButtonChange, 0, 20, 10), []string{"hover dev-1 id1 @40,20 d-40,-40 b0"}},
				{opDown, mouseEv(0, 1, 20, 10), []string{"down dev-1 id2 @40,20 d0,0 b1"}},
			},
		},
		{
			name: "correctly converts buttons of down, move, leave and up events", dpr: 1, short: true,
			steps: []pstep{
				{opMove, mouseEv(noButtonChange, 0, 10, 11), []string{"add* @10,11 b0", "hover @10,11 b0"}},
				{opDown, mouseEv(0, 1, 10, 11), []string{"down @10,11 b1"}},
				{opMove, mouseEv(noButtonChange, 1, 20, 21), []string{"move @20,21 b1"}},
				{opUp, mouseEv(0, 0, 20, 21), []string{"up @20,21 b0"}},
				// Drag with the secondary button.
				{opDown, mouseEv(2, 2, 20, 21), []string{"down @20,21 b2"}},
				{opMove, mouseEv(noButtonChange, 2, 30, 31), []string{"move @30,31 b2"}},
				{opUp, mouseEv(2, 0, 30, 31), []string{"up @30,31 b0"}},
				// Drag with the middle button.
				{opDown, mouseEv(1, 4, 30, 31), []string{"down @30,31 b4"}},
				{opMove, mouseEv(noButtonChange, 4, 40, 41), []string{"move @40,41 b4"}},
				{opUp, mouseEv(1, 0, 40, 41), []string{"up @40,41 b0"}},
				// A leave with a button down is already handled by the moves.
				{opLeave, mouseEv(noButtonChange, 1, 1000, 2000), nil},
				{opLeave, mouseEv(noButtonChange, 0, 1000, 2000), []string{"hover @1000,2000 b0"}},
			},
		},
		{
			name: "correctly handles button changes during a down sequence", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(0, 1, 0, 0), []string{"add* @0,0 b1", "down @0,0 b1"}},
				{opMove, mouseEv(1, 5, 0, 0), []string{"move @0,0 b5"}}, // press the middle button
				{opMove, mouseEv(0, 4, 0, 0), []string{"move @0,0 b4"}}, // release the left button
				{opUp, mouseEv(1, 0, 0, 0), []string{"up @0,0 b0"}},     // release the middle button
			},
		},
		{
			name: "synthesizes a pointerup event when pointermove comes before the up", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 11), []string{"add* @10,11 b2", "down @10,11 b2"}},
				{opMove, mouseEv(noButtonChange, 2, 20, 21), []string{"move @20,21 b2"}},
				{opMove, mouseEv(noButtonChange, 2, 20, 21), []string{"move @20,21 b2"}},
				{opUp, mouseEv(2, 0, 20, 21), []string{"up @20,21 b0"}},
			},
		},
		{
			// Press the right button and hold it, popping up the context menu; click the left button;
			// release the right one.
			name: "correctly handles uncontinuous button changes during a down sequence", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 0, 0), []string{"add* @0,0 b2", "down @0,0 b2"}},
				{opMove, mouseEv(noButtonChange, 3, 0, 0), []string{"move @0,0 b3"}},
				{opMove, mouseEv(0, 2, 0, 0), []string{"move @0,0 b2"}},
				{opUp, mouseEv(2, 0, 0, 0), []string{"up @0,0 b0"}},
			},
		},
		{
			// The right button pops up the context menu, a left click closes it (the browser sends no
			// up), and the user moves the mouse.
			name: "correctly handles missing right mouse button up when followed by move", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 0, 0), []string{"add* @0,0 b2", "down @0,0 b2"}},
				{opMove, mouseEv(noButtonChange, 0, 0, 0), []string{"up @0,0 b0", "hover @0,0 b0"}},
			},
		},
		{
			// The user clicks the right button and moves the mouse quickly: the browser sends a move
			// with button -1 and buttons 2.
			name: "handles RMB click when the browser sends it as a move", dpr: 1, short: true,
			steps: []pstep{
				{opMove, mouseEv(-1, 2, 10, 10), []string{"add* @10,10 b0", "hover @10,10 b0"}},
			},
		},
		{
			name: "correctly handles hover after RMB click", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 10), []string{"add* @10,10 b2", "down @10,10 b2"}},
				// The right button was released without a pointerup, and the hover is somewhere else.
				{opMove, mouseEv(noButtonChange, 0, 20, 20), []string{"move* @20,20 b2", "up @20,20 b0", "hover @20,20 b0"}},
			},
		},
		{
			// The context menu stops the browser's move events, so the left click can be somewhere else.
			name: "correctly handles LMB click after RMB click", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 10), []string{"add* @10,10 b2", "down @10,10 b2"}},
				{opDown, mouseEv(0, 3, 20, 20), []string{"move @20,20 b3"}},
				{opUp, mouseEv(0, 0, 20, 20), []string{"up @20,20 b0"}},
			},
		},
		{
			name: "correctly handles two consecutive RMB clicks with no up in between", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 10), []string{"add* @10,10 b2", "down @10,10 b2"}},
				{opDown, mouseEv(2, 0, 20, 20), []string{"move* @20,20 b2", "up @20,20 b0", "down @20,20 b2"}},
				{opUp, mouseEv(2, 0, 20, 20), []string{"up @20,20 b0"}},
			},
		},
		{
			name: "correctly handles two consecutive RMB clicks with up in between", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 10), []string{"add* @10,10 b2", "down @10,10 b2"}},
				{opUp, mouseEv(2, 0, 10, 10), []string{"up @10,10 b0"}},
				{opDown, mouseEv(2, 0, 20, 20), []string{"hover* @20,20 b0", "down @20,20 b2"}},
				{opUp, mouseEv(2, 0, 20, 20), []string{"up @20,20 b0"}},
			},
		},
		{
			name: "correctly handles two consecutive RMB clicks in two different locations", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(2, 2, 10, 10), []string{"add* @10,10 b2", "down @10,10 b2"}},
				{opUp, mouseEv(2, 0, 10, 10), []string{"up @10,10 b0"}},
				{opDown, mouseEv(2, 2, 20, 20), []string{"hover* @20,20 b0", "down @20,20 b2"}},
			},
		},
		{
			// LMB:      down-------------------up
			// RMB:                 down------------------up
			// Flutter:  down-------move-------move-------up
			name: "handles overlapping left/right down and up events", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(0, 1, 5, 100), []string{"add* @5,100 b1", "down @5,100 b1"}},
				{opDown, mouseEv(2, 3, 20, 100), []string{"move @20,100 b3"}},
				{opUp, mouseEv(0, 2, 30, 100), []string{"move @30,100 b2"}},
				{opUp, mouseEv(2, 0, 30, 100), []string{"up @30,100 b0"}},
			},
		},
		{
			name: "correctly detects an up event outside of the view", dpr: 1, short: true,
			steps: []pstep{
				{opDown, mouseEv(0, 1, 10, 10), []string{"add* @10,10 b1", "down @10,10 b1"}},
				{opMove, mouseEv(noButtonChange, 1, 12, 10), []string{"move @12,10 b1"}},
				{opMove, mouseEv(noButtonChange, 1, 900, 1900), []string{"move @900,1900 b1"}},
				{opUp, mouseEv(0, 0, 1000, 2000), []string{"move* @1000,2000 b1", "up @1000,2000 b0"}},
			},
		},
		{
			// Repeated stylus touches use different pointer IDs, but the stylus is one device.
			name: "handles stylus touches", dpr: 1, short: true,
			steps: []pstep{
				{opDown, PointerEvent{PointerType: "pen", PointerID: 100, Button: 0, Buttons: 1, X: 5, Y: 100}, []string{"add* @5,100 b1", "down @5,100 b1"}},
				{opUp, PointerEvent{PointerType: "pen", PointerID: 100, X: 5, Y: 100}, []string{"up @5,100 b0"}},
				{opDown, PointerEvent{PointerType: "pen", PointerID: 101, Button: 0, Buttons: 1, X: 5, Y: 150}, []string{"hover* @5,150 b0", "down @5,150 b1"}},
				{opUp, PointerEvent{PointerType: "pen", PointerID: 101, X: 5, Y: 150}, []string{"up @5,150 b0"}},
			},
		},
		{
			// Wacom pens simulated with mouse events send a random pointer ID with each event.
			name: "handles random pointer id on up events", dpr: 1, short: true,
			steps: []pstep{
				{opDown, PointerEvent{PointerType: "mouse", PointerID: 12, Button: 0, Buttons: 1, X: 10, Y: 10}, []string{"add* @10,10 b1", "down @10,10 b1"}},
				{opUp, PointerEvent{PointerType: "mouse", PointerID: 41, Button: 0, Buttons: 0, X: 10, Y: 10}, []string{"up @10,10 b0"}},
			},
		},
		{
			name: "does not synthesize pointer up if from a different device", dpr: 1,
			steps: []pstep{
				{opDown, touchEv(1, 1, 100, 101), []string{"add* dev1 id1 @100,101 d0,0 b1", "down dev1 id1 @100,101 d0,0 b1"}},
				{opDown, touchEv(2, 1, 200, 202), []string{"add* dev2 id2 @200,202 d0,0 b1", "down dev2 id2 @200,202 d0,0 b1"}},
			},
		},
		{
			name: "ignores pointer up or pointer cancel events for an unknown device", dpr: 1,
			steps: []pstep{
				{opUp, touchEv(23, 0, 200, 202), nil},
				{opCancel, touchEv(24, 0, 200, 202), nil},
			},
		},
		{
			name: "correctly parses a cancel event", dpr: 1, short: true,
			steps: []pstep{
				{opDown, touchEv(2, 1, 100, 101), []string{"add* @100,101 b1", "down @100,101 b1"}},
				{opDown, touchEv(3, 1, 200, 201), []string{"add* @200,201 b1", "down @200,201 b1"}},
				// A cancel is at the place the pointer was, whatever coordinates it has, and ends the touch.
				{opCancel, touchEv(3, 0, 300, 302), []string{"cancel @200,201 b0", "remove* @200,201 b0"}},
			},
		},
		{
			name: "treats each pointer separately", dpr: 1,
			steps: []pstep{
				{opDown, touchEv(2, 1, 100, 101), []string{"add* dev2 id1 @100,101 d0,0 b1", "down dev2 id1 @100,101 d0,0 b1"}},
				{opDown, touchEv(3, 1, 200, 201), []string{"add* dev3 id2 @200,201 d0,0 b1", "down dev3 id2 @200,201 d0,0 b1"}},
				{opMove, touchEv(3, 1, 300, 302), []string{"move dev3 id2 @300,302 d100,101 b1"}},
				{opMove, touchEv(2, 1, 400, 402), []string{"move dev2 id1 @400,402 d300,301 b1"}},
				{opUp, touchEv(3, 0, 300, 302), []string{"up dev3 id2 @300,302 d0,0 b0", "remove* dev3 id2 @300,302 d0,0 b0"}},
				{opUp, touchEv(2, 0, 400, 402), []string{"up dev2 id1 @400,402 d0,0 b0", "remove* dev2 id1 @400,402 d0,0 b0"}},
				// The pointer IDs of the browser are used again.
				{opDown, touchEv(3, 1, 500, 501), []string{"add* dev3 id3 @500,501 d0,0 b1", "down dev3 id3 @500,501 d0,0 b1"}},
				{opDown, touchEv(2, 1, 600, 601), []string{"add* dev2 id4 @600,601 d0,0 b1", "down dev2 id4 @600,601 d0,0 b1"}},
			},
		},
		{
			name: "ignores pointerId on coalesced events", dpr: 1,
			steps: []pstep{
				{opDown, touchEv(52, 1, 100, 101), []string{"add* dev52 id1 @100,101 d0,0 b1", "down dev52 id1 @100,101 d0,0 b1"}},
				{opMove, PointerEvent{PointerType: "touch", PointerID: 52, Buttons: 1, Coalesced: []PointerEvent{
					{PointerType: "touch", PointerID: 0, Buttons: 1, X: 301, Y: 302},
					{PointerType: "touch", PointerID: 0, Buttons: 1, X: 401, Y: 402},
				}}, []string{"move dev52 id1 @301,302 d201,201 b1", "move dev52 id1 @401,402 d100,100 b1"}},
				{opUp, touchEv(52, 0, 401, 402), []string{"up dev52 id1 @401,402 d0,0 b0", "remove* dev52 id1 @401,402 d0,0 b0"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := NewPointerInput(0, OSLinux)
			for i, s := range tt.steps {
				got := run(in, s.op, s.ev, tt.dpr)
				var lines []string
				for _, d := range got {
					lines = append(lines, fmtData(d, tt.short))
				}
				if strings.Join(lines, "\n") != strings.Join(s.want, "\n") {
					t.Fatalf("step %d: got\n  %s\nwant\n  %s", i, strings.Join(lines, "\n  "), strings.Join(s.want, "\n  "))
				}
			}
		})
	}
}

// TestPointerInputKinds: the kind of device, the tilt and the pressure of the data.
func TestPointerInputKinds(t *testing.T) {
	in := NewPointerInput(3, OSLinux)
	tests := []struct {
		name string
		ev   PointerEvent
		kind ui.PointerDeviceKind
		dev  int
	}{
		{"mouse", PointerEvent{PointerType: "mouse", PointerID: 9, Buttons: 1, Pressure: 0.5}, ui.PointerDeviceKindMouse, -1},
		{"pen", PointerEvent{PointerType: "pen", PointerID: 9, Buttons: 1, Pressure: 0.5, TiltX: 20, TiltY: -45}, ui.PointerDeviceKindStylus, -4},
		{"touch", PointerEvent{PointerType: "touch", PointerID: 9, Buttons: 1, Pressure: 0.5}, ui.PointerDeviceKindTouch, 9},
		{"something else is unknown, told apart by its pointer ID", PointerEvent{PointerType: "", PointerID: 8, Buttons: 1}, ui.PointerDeviceKindUnknown, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := in.Down(tt.ev, 1)
			d := data[len(data)-1]
			if d.Kind != tt.kind || d.Device != tt.dev || d.ViewID != 3 || d.PressureMax != 1 || d.Pressure != tt.ev.Pressure {
				t.Errorf("got kind %v device %d view %d pressure %v/%v, want kind %v device %d view 3 pressure %v/1",
					d.Kind, d.Device, d.ViewID, d.Pressure, d.PressureMax, tt.kind, tt.dev, tt.ev.Pressure)
			}
			wantTilt := math.Pi * -45 / 180
			if tt.name == "pen" && math.Abs(d.Tilt-wantTilt) > 1e-12 {
				t.Errorf("tilt = %v, want the larger deflection %v", d.Tilt, wantTilt)
			}
			in.Up(PointerEvent{PointerType: tt.ev.PointerType, PointerID: tt.ev.PointerID}, 1)
		})
	}
}

func TestConvertButtonToButtons(t *testing.T) {
	tests := []struct{ button, want int }{{0, 1}, {1, 4}, {2, 2}, {3, 8}, {4, 16}, {5, 32}, {-1, 0}}
	for _, tt := range tests {
		if got := ConvertButtonToButtons(tt.button); got != tt.want {
			t.Errorf("ConvertButtonToButtons(%d) = %d, want %d", tt.button, got, tt.want)
		}
	}
	// A down that reports no buttons (some browsers) gets them from button.
	s := &ButtonSanitizer{}
	if d := s.Down(2, 0); d.change != ui.PointerChangeDown || d.buttons != secondaryMouseButton {
		t.Errorf("a down with buttons 0 and button 2 = %v, want buttons 2", d)
	}
}

// chrome gives a wheel event that a page creates a wheelDelta of zero, which the detection of a
// trackpad reads (the tests of Flutter web create such events); w makes the event so.
func w(e WheelEvent) WheelEvent {
	if !e.HasWheelDelta {
		e.HasWheelDelta = true
	}
	return e
}

func fmtWheel(d ui.PointerData) string {
	name := strings.ToLower(strings.TrimPrefix(d.Change.String(), "PointerChange"))
	if d.Synthesized {
		name += "*"
	}
	sig := strings.ToLower(strings.TrimPrefix(d.SignalKind.String(), "PointerSignalKind"))
	kind := strings.ToLower(strings.TrimPrefix(d.Kind.String(), "PointerDeviceKind"))
	return fmt.Sprintf("%s %s %s dev%d id%d @%g,%g d%g,%g scroll%g,%g b%d", name, sig, kind, d.Device, d.PointerIdentifier,
		d.PhysicalX, d.PhysicalY, d.PhysicalDeltaX, d.PhysicalDeltaY, d.ScrollDeltaX, d.ScrollDeltaY, d.Buttons)
}

var oneToOne = WheelContext{DevicePixelRatio: 1, PhysicalWidth: 800, PhysicalHeight: 600}

func TestWheelSynthesizesAddHoverAndMove(t *testing.T) {
	in := NewPointerInput(0, OSLinux)
	step := func(e WheelEvent) []string {
		data, _ := in.Wheel(e, oneToOne)
		var out []string
		for _, d := range data {
			out = append(out, fmtWheel(d))
		}
		return out
	}
	check := func(got, want []string) {
		t.Helper()
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
		}
	}
	// An add is synthesized.
	check(step(w(WheelEvent{X: 10, Y: 10, DeltaX: 10, DeltaY: 10})), []string{
		"add* none mouse dev-1 id0 @10,10 d0,0 scroll10,10 b0",
		"hover scroll mouse dev-1 id0 @10,10 d0,0 scroll10,10 b0",
	})
	// A hover is synthesized.
	check(step(w(WheelEvent{X: 20, Y: 50, DeltaX: 10, DeltaY: 10})), []string{
		"hover* none mouse dev-1 id0 @20,50 d10,40 scroll10,10 b0",
		"hover scroll mouse dev-1 id0 @20,50 d0,0 scroll10,10 b0",
	})
	// No synthetic data for a down.
	down := in.Down(mouseEv(0, 1, 20, 50), 1)
	if len(down) != 1 || fmtData(down[0], false) != "down dev-1 id1 @20,50 d0,0 b1" {
		t.Fatalf("down = %v", down)
	}
	// A move is synthesized instead of the hover because the button is down.
	check(step(w(WheelEvent{X: 30, Y: 60, DeltaX: 10, DeltaY: 10, Buttons: 1})), []string{
		"move* none mouse dev-1 id1 @30,60 d10,10 scroll10,10 b1",
		"hover scroll mouse dev-1 id1 @30,60 d0,0 scroll10,10 b1",
	})
}

func TestWheelDeltaModes(t *testing.T) {
	tests := []struct {
		name         string
		os           OperatingSystem
		e            WheelEvent
		ctx          WheelContext
		wantX, wantY float64
	}{
		{"pixels are kept", OSLinux, WheelEvent{DeltaX: 10, DeltaY: 10}, WheelContext{DevicePixelRatio: 2.5}, 10, 10},
		{"pixels are multiplied by the pixel ratio on macOS", OSMacOS, WheelEvent{DeltaX: 10, DeltaY: 10}, WheelContext{DevicePixelRatio: 2.5}, 25, 25},
		{"lines are multiplied by the line height", OSLinux, WheelEvent{DeltaX: 1, DeltaY: 3, DeltaMode: 1}, WheelContext{DevicePixelRatio: 2, LineHeight: 4}, 4, 12},
		{"lines use 16 pixels when the browser's font size is unknown", OSLinux, WheelEvent{DeltaY: 2, DeltaMode: 1}, WheelContext{DevicePixelRatio: 1}, 0, 32},
		{"pages are multiplied by the size of the view", OSLinux, WheelEvent{DeltaX: 1, DeltaY: 2, DeltaMode: 2}, WheelContext{DevicePixelRatio: 2, PhysicalWidth: 800, PhysicalHeight: 600}, 800, 1200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := NewPointerInput(0, tt.os).Wheel(tt.e, tt.ctx)
			d := data[len(data)-1]
			if d.SignalKind != ui.PointerSignalKindScroll || d.ScrollDeltaX != tt.wantX || d.ScrollDeltaY != tt.wantY {
				t.Errorf("scroll = %v,%v (%v), want %v,%v", d.ScrollDeltaX, d.ScrollDeltaY, d.SignalKind, tt.wantX, tt.wantY)
			}
			// The position is in physical pixels.
		})
	}
	// converts scroll delta to physical pixels (macOS): the add is the synthesized data with the same deltas.
	in := NewPointerInput(0, OSMacOS)
	data, _ := in.Wheel(w(WheelEvent{X: 10, Y: 10, DeltaX: 10, DeltaY: 10}), WheelContext{DevicePixelRatio: 2.5})
	if len(data) != 2 || data[0].Change != ui.PointerChangeAdd || data[0].ScrollDeltaX != 25 || data[1].PhysicalX != 25 {
		t.Errorf("macOS data = %v", data)
	}
}

// TestWheelKindFollowsDeltaPrecision ports 'does set pointer device kind based on delta precision
// and wheelDelta': a delta that is a multiple of 120 is a mouse wheel, unless it is like the one
// before it or soon after a trackpad event; any other is a trackpad's if its wheelDelta says so.
func TestWheelKindFollowsDeltaPrecision(t *testing.T) {
	events := []struct {
		e    WheelEvent
		want string
	}{
		// Not in increments of 120 and with a matching wheelDelta: a trackpad.
		{WheelEvent{X: 10, Y: 10, DeltaX: 119, DeltaY: 119, WheelDeltaX: -357, WheelDeltaY: -357, HasWheelDelta: true, TimeStamp: 0}, "trackpad dev-2 scroll119,119"},
		// In increments of 120, but like the one before: a trackpad.
		{WheelEvent{X: 10, Y: 10, DeltaX: 120, DeltaY: 120, WheelDeltaX: -360, WheelDeltaY: -360, HasWheelDelta: true, TimeStamp: 10}, "trackpad dev-2 scroll120,120"},
		{WheelEvent{X: 10, Y: 10, DeltaX: 120, DeltaY: 120, WheelDeltaX: -360, WheelDeltaY: -360, HasWheelDelta: true, TimeStamp: 20}, "trackpad dev-2 scroll120,120"},
		{WheelEvent{X: 10, Y: 10, DeltaX: 119, DeltaY: 119, WheelDeltaX: -357, WheelDeltaY: -357, HasWheelDelta: true, TimeStamp: 1000}, "trackpad dev-2 scroll119,119"},
		// In increments of 120 and not like the one before, but soon after it: a trackpad.
		{WheelEvent{X: 10, Y: 10, DeltaX: -120, DeltaY: -120, WheelDeltaX: 360, WheelDeltaY: 360, HasWheelDelta: true, TimeStamp: 1010}, "trackpad dev-2 scroll-120,-120"},
		// In increments of 120, not like the one before, and long after it: a mouse.
		{WheelEvent{X: 10, Y: 10, DeltaX: 0, DeltaY: -120, WheelDeltaX: 0, WheelDeltaY: 360, HasWheelDelta: true, TimeStamp: 2000}, "mouse dev-1 scroll0,-120"},
		// Not in increments of 120 and a wheelDelta that does not match: a mouse (with macOS's acceleration).
		{WheelEvent{X: 10, Y: 10, DeltaX: 0, DeltaY: 40, WheelDeltaX: 0, WheelDeltaY: -360, HasWheelDelta: true, TimeStamp: 3000}, "mouse dev-1 scroll0,40"},
	}
	in := NewPointerInput(0, OSLinux)
	for i, ev := range events {
		data, _ := in.Wheel(ev.e, oneToOne)
		d := data[len(data)-1]
		kind := strings.ToLower(strings.TrimPrefix(d.Kind.String(), "PointerDeviceKind"))
		got := fmt.Sprintf("%s dev%d scroll%g,%g", kind, d.Device, d.ScrollDeltaX, d.ScrollDeltaY)
		if got != ev.want {
			t.Errorf("event %d: got %q, want %q", i, got, ev.want)
		}
	}
	// Firefox cannot tell a trackpad from a wheel: always a mouse.
	ff := NewPointerInput(0, OSLinux, PointerInputOptions{Firefox: true})
	data, _ := ff.Wheel(events[0].e, oneToOne)
	if k := data[len(data)-1].Kind; k != ui.PointerDeviceKindMouse {
		t.Errorf("Firefox wheel kind = %v, want mouse", k)
	}
}

// TestWheelScrollOrScaleByControl ports 'does choose scroll vs scale based on ctrlKey' on macOS:
// control with the wheel is a scale (a pinch), unless the control key is physically down.
func TestWheelScrollOrScaleByControl(t *testing.T) {
	keys := NewKeyConverter(func(ui.KeyData) bool { return true }, OSMacOS, nil)
	in := NewPointerInput(0, OSMacOS, PointerInputOptions{Keys: keys})
	data, _ := in.Wheel(w(WheelEvent{X: 10, Y: 10, DeltaY: 120}), oneToOne)
	if len(data) != 2 || data[1].SignalKind != ui.PointerSignalKindScroll || data[1].Kind != ui.PointerDeviceKindMouse || data[1].ScrollDeltaY != 120 {
		t.Errorf("without control: %v, want a scroll", data)
	}
	data, _ = in.Wheel(w(WheelEvent{X: 10, Y: 10, DeltaY: 100, Ctrl: true}), oneToOne)
	if len(data) != 1 || data[0].SignalKind != ui.PointerSignalKindScale || math.Abs(data[0].Scale-0.60653065971) > 1e-10 {
		t.Errorf("with control: %v, want a scale of exp(-100/200)", data)
	}
	keys.Handle(keyDown("ControlLeft", "Control", kCtrl, 0))
	data, _ = in.Wheel(w(WheelEvent{X: 10, Y: 10, DeltaY: 240, Ctrl: true}), oneToOne)
	if len(data) != 1 || data[0].SignalKind != ui.PointerSignalKindScroll || data[0].ScrollDeltaY != 240 {
		t.Errorf("with the control key down on macOS: %v, want a scroll", data)
	}
}

// TestWheelPreventDefault ports the tests of what the framework's answer does to the wheel
// event: the default action is prevented unless a scrollable at its boundary let the page scroll;
// once it did, it stays so; a full-page app in a frame lets the page scroll only when no widget
// took the scroll as well.
func TestWheelPreventDefault(t *testing.T) {
	tests := []struct {
		name            string
		answers         []bool // the framework's calls of Respond
		fullPageInFrame bool
		want            bool
	}{
		{"wheel event - preventDefault called", nil, false, true},
		{"the framework can stop preventDefault", []bool{true}, false, false},
		{"once allowPlatformDefault is set to true, it cannot be rolled back", []bool{false, true, false}, false, false},
		{"a widget took the scroll", []bool{false}, false, true},
		{"full-page iframe calls preventDefault when handled", []bool{false}, true, true},
		{"full-page iframe allows browser scroll when platform default allowed", []bool{true}, true, false},
		{"full-page iframe: allowed by one scrollable but handled by another", []bool{true, false}, true, true},
		{"custom-element iframe uses the original behavior when handled", []bool{false}, false, true},
		{"custom-element iframe allows browser scroll when platform default allowed", []bool{true}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, res := NewPointerInput(0, OSLinux).Wheel(w(WheelEvent{X: 10, Y: 10, DeltaX: 10}), oneToOne)
			// The framework answers while it dispatches the scroll.
			for _, d := range data {
				if d.SignalKind == ui.PointerSignalKindScroll {
					for _, a := range tt.answers {
						d.Respond(a)
					}
				}
			}
			if got := res.PreventDefault(tt.fullPageInFrame); got != tt.want {
				t.Errorf("PreventDefault = %v, want %v", got, tt.want)
			}
		})
	}
	// A scale (a pinch) has no answer: the page never zooms.
	data, res := NewPointerInput(0, OSLinux).Wheel(w(WheelEvent{X: 10, Y: 10, DeltaY: 10, Ctrl: true}), oneToOne)
	if data[len(data)-1].OnRespond != nil || !res.PreventDefault(false) {
		t.Error("a scale event can be answered or lets the page zoom")
	}
}

// TestCancelAbandonedTouches ports the tests of the safety net for the touches that iOS WebKit
// abandons without a pointerup or a pointercancel.
func TestCancelAbandonedTouches(t *testing.T) {
	down := func(in *PointerInput, ids ...int) {
		for _, id := range ids {
			in.Down(touchEv(id, 1, float64(id)*100, float64(id)*100+1), 1)
		}
	}
	str := func(data []ui.PointerData) string {
		var out []string
		for _, d := range data {
			out = append(out, fmtData(d, true)+fmt.Sprintf(" dev%d", d.Device))
		}
		return strings.Join(out, ";")
	}
	t.Run("cancels a touch the browser abandons without pointerup", func(t *testing.T) {
		in := NewPointerInput(0, OSIOS)
		down(in, 2)
		// The finger lifts, but the browser only reports touchend; touches lists touch 3, whose
		// pointerdown has not been dispatched yet, but not touch 2.
		got := str(in.CancelAbandonedTouches([]int{3}, 5))
		if want := "cancel @200,201 b0 dev2;remove* @200,201 b0 dev2"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("does not cancel a touch that was released normally", func(t *testing.T) {
		in := NewPointerInput(0, OSIOS)
		down(in, 2)
		in.Up(touchEv(2, 0, 200, 201), 1)
		if got := in.CancelAbandonedTouches(nil, 5); got != nil {
			t.Errorf("got %v, want nothing", got)
		}
	})
	t.Run("cancels an abandoned touch whose touchend never arrives", func(t *testing.T) {
		in := NewPointerInput(0, OSIOS)
		down(in, 2, 3)
		in.Up(touchEv(3, 0, 300, 301), 1)
		got := in.CancelAbandonedTouches(nil, 5)
		if len(got) == 0 || got[0].Change != ui.PointerChangeCancel || got[0].Device != 2 {
			t.Errorf("got %v, want a cancel of device 2", got)
		}
	})
	t.Run("cancels an abandoned touch while other fingers are still down", func(t *testing.T) {
		in := NewPointerInput(0, OSIOS)
		down(in, 2, 3, 4)
		in.Up(touchEv(3, 0, 300, 301), 1)
		got := in.CancelAbandonedTouches([]int{4}, 5)
		if len(got) != 2 || got[0].Change != ui.PointerChangeCancel || got[0].Device != 2 || got[1].Change != ui.PointerChangeRemove || got[1].Device != 2 {
			t.Errorf("got %v, want the cancel and the remove of device 2 only", got)
		}
	})
	t.Run("cancels an abandoned touch on touchcancel", func(t *testing.T) {
		in := NewPointerInput(0, OSIOS)
		down(in, 2)
		got := in.CancelAbandonedTouches(nil, 5)
		if len(got) == 0 || got[0].Change != ui.PointerChangeCancel || got[0].Device != 2 {
			t.Errorf("got %v, want a cancel of device 2", got)
		}
	})
	t.Run("does not cancel abandoned touches on non-iOS browsers", func(t *testing.T) {
		for _, os := range []OperatingSystem{OSLinux, OSAndroid, OSMacOS, OSWindows} {
			in := NewPointerInput(0, os)
			down(in, 2)
			if got := in.CancelAbandonedTouches(nil, 5); got != nil {
				t.Errorf("%v: got %v, want nothing", os, got)
			}
		}
	})
}

// TestPointerInputSynthesizesModifiers: a pointer event brings the record of the modifier keys in
// step with its flags before the pointer data is made.
func TestPointerInputSynthesizesModifiers(t *testing.T) {
	var keys []ui.KeyData
	conv := NewKeyConverter(func(d ui.KeyData) bool { keys = append(keys, d); return true }, OSLinux, nil)
	in := NewPointerInput(0, OSLinux, PointerInputOptions{Keys: conv})
	e := mouseEv(0, 1, 1, 1)
	e.Shift = true
	in.Down(e, 1)
	if len(keys) != 1 || keys[0].Type != ui.KeyEventTypeDown || keys[0].Physical != phys("ShiftLeft") || !keys[0].Synthesized {
		t.Fatalf("keys = %v, want a synthesized down of ShiftLeft", keys)
	}
	up := mouseEv(0, 0, 1, 1)
	in.Up(up, 1)
	if len(keys) != 2 || keys[1].Type != ui.KeyEventTypeUp || keys[1].Physical != phys("ShiftLeft") {
		t.Fatalf("keys = %v, want a synthesized up of ShiftLeft", keys)
	}
	// A leave does not check the modifiers.
	e.Shift = true
	in.Leave(mouseEv(noButtonChange, 0, 5, 5), 1)
	in.Leave(e, 1)
	if len(keys) != 2 {
		t.Errorf("a leave synthesized keys: %v", keys[2:])
	}
}

func TestParsePixels(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{{"16px", 16, true}, {"12.5px", 12.5, true}, {"1em", 0, false}, {"", 0, false}, {"pxpx", 0, false}}
	for _, tt := range tests {
		if got, ok := ParsePixels(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("ParsePixels(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestEventTimeStamp(t *testing.T) {
	in := NewPointerInput(0, OSLinux)
	e := mouseEv(0, 1, 1, 1)
	e.TimeStamp = 12.5
	data := in.Down(e, 1)
	if got := data[len(data)-1].TimeStamp; got != 12*time.Millisecond+500*time.Microsecond {
		t.Errorf("timestamp = %v, want 12.5ms", got)
	}
}
