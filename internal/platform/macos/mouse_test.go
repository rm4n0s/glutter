package macos

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// summary is the part of a pointer data that the tests compare: the change, the signal, the device and
// the buttons.
type summary struct {
	change  ui.PointerChange
	signal  ui.PointerSignalKind
	device  int
	buttons int
	synth   bool
}

func summaries(ds []ui.PointerData) []summary {
	out := make([]summary, len(ds))
	for i, d := range ds {
		out[i] = summary{d.Change, d.SignalKind, d.Device, d.Buttons, d.Synthesized}
	}
	return out
}

func at(x, y float64, ms int) pointerInput {
	return pointerInput{view: 1, x: x, y: y, time: time.Duration(ms) * time.Millisecond, scale: 2}
}

var (
	add    = ui.PointerChangeAdd
	hover  = ui.PointerChangeHover
	down   = ui.PointerChangeDown
	move   = ui.PointerChangeMove
	up     = ui.PointerChangeUp
	remove = ui.PointerChangeRemove
	pzs    = ui.PointerChangePanZoomStart
	pzu    = ui.PointerChangePanZoomUpdate
	pze    = ui.PointerChangePanZoomEnd
)

func TestMouseSequences(t *testing.T) {
	tests := []struct {
		name string
		run  func(m *pointerMachine)
		want []summary
	}{
		{"a hover with no enter first gets an add", func(m *pointerMachine) { m.moved(at(5, 5, 0)) },
			[]summary{{change: add}, {change: hover}}},
		{"enter, then a hover", func(m *pointerMachine) { m.entered(at(5, 5, 0)); m.moved(at(6, 5, 1)) },
			[]summary{{change: add}, {change: hover}}},
		{"a second enter is dropped", func(m *pointerMachine) { m.entered(at(5, 5, 0)); m.entered(at(5, 5, 1)) },
			[]summary{{change: add}}},
		{"a click: add, down, up", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.buttonUp(at(5, 5, 2), buttonPrimary)
		}, []summary{{change: add}, {change: down, buttons: buttonPrimary}, {change: up}}},
		{"a click with no enter: the add does not carry the buttons", func(m *pointerMachine) {
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.buttonUp(at(5, 5, 2), buttonPrimary)
		}, []summary{{change: add}, {change: down, buttons: buttonPrimary}, {change: up}}},
		{"a drag moves with the button down", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.moved(at(9, 5, 2))
			m.buttonUp(at(9, 5, 3), buttonPrimary)
		}, []summary{{change: add}, {change: down, buttons: buttonPrimary}, {change: move, buttons: buttonPrimary}, {change: up}}},
		{"two buttons: the second is a move, the last up is the up", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.buttonDown(at(5, 5, 2), buttonSecondary)
			m.buttonUp(at(5, 5, 3), buttonPrimary)
			m.buttonUp(at(5, 5, 4), buttonSecondary)
		}, []summary{{change: add}, {change: down, buttons: 1}, {change: move, buttons: 3}, {change: move, buttons: 2}, {change: up}}},
		{"leaving with a button down waits for the last button", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.exited(at(-3, 5, 2)) // Cocoa keeps sending drags after the exit
			m.moved(at(-9, 5, 3))
			m.buttonUp(at(-9, 5, 4), buttonPrimary)
		}, []summary{{change: add}, {change: down, buttons: 1}, {change: move, buttons: 1}, {change: up}, {change: remove}}},
		{"coming back while dragging cancels the delayed exit", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonPrimary)
			m.exited(at(-3, 5, 2))
			m.entered(at(2, 5, 3))
			m.buttonUp(at(2, 5, 4), buttonPrimary)
		}, []summary{{change: add}, {change: down, buttons: 1}, {change: up}}},
		{"exit with no button removes at once", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.exited(at(-3, 5, 1))
		}, []summary{{change: add}, {change: remove}}},
		{"a middle button and the back and forward buttons", func(m *pointerMachine) {
			m.entered(at(5, 5, 0))
			m.buttonDown(at(5, 5, 1), buttonMiddle|buttonBack|buttonForward)
		}, []summary{{change: add}, {change: down, buttons: 0x1c}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newPointerMachine()
			tt.run(m)
			if got := summaries(m.take()); !slices.Equal(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestMouseDataDeltasAndIdentifiers(t *testing.T) {
	m := newPointerMachine()
	m.entered(at(10, 10, 0))
	m.moved(at(14, 13, 1))
	m.buttonDown(at(14, 13, 2), buttonPrimary)
	m.buttonUp(at(14, 13, 3), buttonPrimary)
	m.buttonDown(at(14, 13, 4), buttonPrimary)
	d := m.take()
	if d[1].PhysicalDeltaX != 4 || d[1].PhysicalDeltaY != 3 {
		t.Errorf("the hover delta = %v, %v", d[1].PhysicalDeltaX, d[1].PhysicalDeltaY)
	}
	if d[0].PhysicalDeltaX != 0 {
		t.Error("an add has a delta")
	}
	if d[2].PointerIdentifier == 0 || d[2].PointerIdentifier == d[4].PointerIdentifier {
		t.Errorf("each down has an identifier of its own: %d and %d", d[2].PointerIdentifier, d[4].PointerIdentifier)
	}
	if d[2].PointerIdentifier != d[3].PointerIdentifier {
		t.Error("the up has another identifier than its down")
	}
	for _, x := range d {
		if x.Kind != ui.PointerDeviceKindMouse || x.Device != mouseDevice || x.ViewID != 1 {
			t.Errorf("data = %+v", x)
		}
	}
	if m.mouseButtons() != buttonPrimary {
		t.Errorf("mouseButtons = %d", m.mouseButtons())
	}
}

func TestMouseMovingToAnotherWindowLeavesTheFirst(t *testing.T) {
	m := newPointerMachine()
	m.entered(pointerInput{view: 1, x: 5, y: 5, scale: 1})
	m.buttonDown(pointerInput{view: 1, x: 5, y: 5, scale: 1}, buttonPrimary)
	m.take()
	m.entered(pointerInput{view: 2, x: 7, y: 7, scale: 1})
	got := m.take()
	want := []summary{{change: up, synth: true}, {change: remove}, {change: add}}
	if !slices.Equal(summaries(got), want) {
		t.Fatalf("got %+v, want %+v", summaries(got), want)
	}
	if got[0].ViewID != 1 || got[1].ViewID != 1 || got[2].ViewID != 2 {
		t.Errorf("views = %d %d %d", got[0].ViewID, got[1].ViewID, got[2].ViewID)
	}
}

func TestWheelSignalsAreScaledAndSwappedWithShift(t *testing.T) {
	tests := []struct {
		name           string
		in             pointerInput
		wantDX, wantDY float64
	}{
		// scale 2, a line is 40 points: a notch of 1 line down is dy = -1 in AppKit (scrollingDeltaY is
		// positive for content moving down) and a positive scroll delta of 80 pixels for Flutter.
		{"a notch down", pointerInput{dy: -1, scale: 2}, 0, 80},
		{"a notch up", pointerInput{dy: 1, scale: 2}, 0, -80},
		{"a notch right", pointerInput{dx: -1, scale: 2}, 80, 0},
		{"shift swaps the axes back", pointerInput{dy: -1, scale: 2, shift: true}, 80, 0},
		{"smooth deltas are pixels already", pointerInput{dy: -3, scale: 2, precise: true}, 0, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newPointerMachine()
			tt.in.view = 1
			m.scrollWheel(tt.in)
			d := m.take()
			if len(d) != 2 || d[0].Change != add {
				t.Fatalf("data = %+v", summaries(d))
			}
			s := d[1]
			if s.SignalKind != ui.PointerSignalKindScroll || s.Change != hover {
				t.Errorf("signal = %v change %v", s.SignalKind, s.Change)
			}
			if s.ScrollDeltaX != tt.wantDX || s.ScrollDeltaY != tt.wantDY {
				t.Errorf("delta = %v, %v; want %v, %v", s.ScrollDeltaX, s.ScrollDeltaY, tt.wantDX, tt.wantDY)
			}
		})
	}
}

func TestWheelOverADraggedMouseIsAMove(t *testing.T) {
	m := newPointerMachine()
	m.entered(at(5, 5, 0))
	m.buttonDown(at(5, 5, 1), buttonPrimary)
	m.take()
	m.scrollWheel(pointerInput{view: 1, x: 5, y: 5, dy: -1, scale: 1})
	d := m.take()
	if len(d) != 1 || d[0].Change != move || d[0].SignalKind != ui.PointerSignalKindScroll {
		t.Errorf("data = %+v", summaries(d))
	}
}

func scroll(phase, momentum evPhase, dx, dy float64, ms int) pointerInput {
	in := at(20, 30, ms)
	in.precise, in.phase, in.momentum, in.dx, in.dy = true, phase, momentum, dx, dy
	return in
}

func TestTrackpadScrollIsAPanZoomAndMomentumIsDropped(t *testing.T) {
	m := newPointerMachine()
	m.scrollWheel(scroll(phaseBegan, phaseNone, 0, 0, 0))
	m.scrollWheel(scroll(phaseChanged, phaseNone, 3, -2, 10))
	m.scrollWheel(scroll(phaseChanged, phaseNone, 1, -1, 20))
	m.scrollWheel(scroll(phaseEnded, phaseNone, 0, 0, 30))
	// The fling: momentum events with no phase of their own are skipped.
	m.scrollWheel(scroll(phaseNone, phaseBegan, 5, 5, 40))
	m.scrollWheel(scroll(phaseNone, phaseChanged, 5, 5, 50))
	m.scrollWheel(scroll(phaseNone, phaseEnded, 0, 0, 60))
	d := m.take()
	got := summaries(d)
	want := []summary{{change: add, device: panZoomDevice}, {change: pzs, device: panZoomDevice}, {change: pzu, device: panZoomDevice},
		{change: pzu, device: panZoomDevice}, {change: pze, device: panZoomDevice}}
	if !slices.Equal(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	for _, x := range d {
		if x.Kind != ui.PointerDeviceKindTrackpad {
			t.Errorf("kind = %v", x.Kind)
		}
	}
	// The pan is accumulated, in physical pixels (scale 2), with the delta since the last update.
	if d[2].PanX != 6 || d[2].PanY != -4 || d[2].PanDeltaX != 6 || d[2].PanDeltaY != -4 {
		t.Errorf("first update pan = %v,%v delta %v,%v", d[2].PanX, d[2].PanY, d[2].PanDeltaX, d[2].PanDeltaY)
	}
	if d[3].PanX != 8 || d[3].PanY != -6 || d[3].PanDeltaX != 2 || d[3].PanDeltaY != -2 {
		t.Errorf("second update pan = %v,%v delta %v,%v", d[3].PanX, d[3].PanY, d[3].PanDeltaX, d[3].PanDeltaY)
	}
	if d[2].Scale != 1 || d[2].Rotation != 0 {
		t.Errorf("a scroll has scale %v rotation %v", d[2].Scale, d[2].Rotation)
	}
	if d[1].PointerIdentifier == 0 || d[1].PointerIdentifier != d[2].PointerIdentifier || d[2].PointerIdentifier != d[4].PointerIdentifier {
		t.Errorf("the gesture's identifiers: %d %d %d", d[1].PointerIdentifier, d[2].PointerIdentifier, d[4].PointerIdentifier)
	}
	// A new gesture starts from zero.
	m.scrollWheel(scroll(phaseBegan, phaseNone, 0, 0, 100))
	m.scrollWheel(scroll(phaseChanged, phaseNone, 1, 0, 110))
	d = m.take()
	if last := d[len(d)-1]; last.PanX != 2 || last.PanDeltaX != 2 {
		t.Errorf("the second gesture's pan = %v delta %v", last.PanX, last.PanDeltaX)
	}
}

func TestPinchAndRotation(t *testing.T) {
	m := newPointerMachine()
	pinch := func(phase evPhase, mag float64) pointerInput {
		in := at(20, 30, 0)
		in.phase, in.magnification = phase, mag
		return in
	}
	rot := func(phase evPhase, deg float64) pointerInput {
		in := at(20, 30, 0)
		in.phase, in.rotation = phase, deg
		return in
	}
	m.magnify(pinch(phaseBegan, 0))
	m.magnify(pinch(phaseChanged, 1))
	m.rotate(rot(phaseBegan, 0))
	m.rotate(rot(phaseChanged, 90))
	m.magnify(pinch(phaseEnded, 0)) // the pinch ends but the rotation goes on: no end yet
	m.rotate(rot(phaseChanged, 90))
	m.rotate(rot(phaseEnded, 0))
	d := m.take()
	got := summaries(d)
	pz := func(c ui.PointerChange) summary { return summary{change: c, device: panZoomDevice} }
	want := []summary{pz(add), pz(pzs), pz(pzu), pz(pzu), pz(pzu), pz(pze)}
	if !slices.Equal(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if math.Abs(d[2].Scale-2) > 1e-9 { // 2^1
		t.Errorf("scale after a magnification of 1 = %v, want 2", d[2].Scale)
	}
	if math.Abs(d[3].Rotation-(-math.Pi/2)) > 1e-9 { // clockwise on screen is positive: AppKit's is the other way
		t.Errorf("rotation of 90 degrees = %v, want -pi/2", d[3].Rotation)
	}
	if math.Abs(d[4].Rotation-(-math.Pi)) > 1e-9 || math.Abs(d[4].Scale-2) > 1e-9 {
		t.Errorf("the gesture goes on: rotation %v scale %v", d[4].Rotation, d[4].Scale)
	}
	// The next gesture starts clean.
	m.magnify(pinch(phaseBegan, 0))
	m.magnify(pinch(phaseChanged, 0))
	d = m.take()
	if last := d[len(d)-1]; last.Scale != 1 || last.Rotation != 0 {
		t.Errorf("a new gesture has scale %v rotation %v", last.Scale, last.Rotation)
	}
}

func TestACancelWithNoGestureIsIgnored(t *testing.T) {
	m := newPointerMachine()
	m.scrollWheel(scroll(phaseCancelled, phaseNone, 0, 0, 0))
	if d := m.take(); len(d) != 0 {
		t.Errorf("data = %+v", summaries(d))
	}
	// A gesture that began and was cancelled ends once.
	m.scrollWheel(scroll(phaseBegan, phaseNone, 0, 0, 1))
	m.scrollWheel(scroll(phaseCancelled, phaseNone, 0, 0, 2))
	got := summaries(m.take())
	want := []summary{{change: add, device: panZoomDevice}, {change: pzs, device: panZoomDevice}, {change: pze, device: panZoomDevice}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestATouchAfterMomentumIsAnInertiaCancel(t *testing.T) {
	flick := func() *pointerMachine {
		m := newPointerMachine()
		m.scrollWheel(scroll(phaseBegan, phaseNone, 0, 0, 0))
		m.scrollWheel(scroll(phaseEnded, phaseNone, 0, 0, 10))
		m.scrollWheel(scroll(phaseNone, phaseBegan, 3, 3, 20))
		m.scrollWheel(scroll(phaseNone, phaseChanged, 3, 3, 100))
		m.take()
		return m
	}
	tests := []struct {
		name   string
		touch  int // ms
		cancel bool
	}{
		{"right after the last momentum event", 120, true},
		{"just inside the window", 149, true},
		{"after the window", 150, false},
		{"long after", 1000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := flick()
			m.touchesBegan(at(20, 30, tt.touch))
			d := m.take()
			if !tt.cancel {
				if len(d) != 0 {
					t.Errorf("data = %+v", summaries(d))
				}
				return
			}
			if len(d) != 1 || d[0].SignalKind != ui.PointerSignalKindScrollInertiaCancel || d[0].Device != panZoomDevice || d[0].Kind != ui.PointerDeviceKindTrackpad {
				t.Fatalf("data = %+v", summaries(d))
			}
			m.touchesBegan(at(20, 30, tt.touch+1)) // only one cancel for a flick
			if d := m.take(); len(d) != 0 {
				t.Error("a second touch cancelled again")
			}
		})
	}
	t.Run("a touch without momentum does nothing", func(t *testing.T) {
		m := newPointerMachine()
		m.touchesBegan(at(1, 1, 5))
		if d := m.take(); len(d) != 0 {
			t.Errorf("data = %+v", summaries(d))
		}
	})
	t.Run("a new scroll forgets the momentum", func(t *testing.T) {
		m := flick()
		m.scrollWheel(scroll(phaseBegan, phaseNone, 0, 0, 105))
		m.take()
		m.touchesBegan(at(20, 30, 110))
		if d := m.take(); len(d) != 0 {
			t.Errorf("data = %+v", summaries(d))
		}
	})
	t.Run("the first momentum event alone does not count", func(t *testing.T) {
		// Flutter waits for the first change of the momentum (a workaround for low power mode).
		m := newPointerMachine()
		m.scrollWheel(scroll(phaseNone, phaseBegan, 3, 3, 20))
		m.touchesBegan(at(20, 30, 30))
		if d := m.take(); len(d) != 0 {
			t.Errorf("data = %+v", summaries(d))
		}
	})
}

func TestStylusProximityAndContact(t *testing.T) {
	m := newPointerMachine()
	in := at(10, 10, 0)
	in.detail = pointerDetail{pressure: 0.5, pressureMax: 1, tilt: 0.3, orientation: 1}
	const dev = stylusDeviceBase + 2
	m.stylusEnter(in, ui.PointerDeviceKindStylus, dev)
	m.stylusMove(in, ui.PointerDeviceKindStylus, dev, 0)
	m.stylusMove(in, ui.PointerDeviceKindStylus, dev, buttonPrimary)
	in.x = 12
	m.stylusMove(in, ui.PointerDeviceKindStylus, dev, buttonPrimary)
	m.stylusMove(in, ui.PointerDeviceKindStylus, dev, 0)
	m.stylusLeave(in, ui.PointerDeviceKindStylus, dev)
	d := m.take()
	want := []summary{{change: add, device: dev}, {change: hover, device: dev}, {change: down, device: dev, buttons: 1},
		{change: move, device: dev, buttons: 1}, {change: up, device: dev}, {change: remove, device: dev}}
	if got := summaries(d); !slices.Equal(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if d[2].Pressure != 0.5 || d[2].PressureMax != 1 || d[2].Tilt != 0.3 || d[2].Orientation != 1 || d[2].Kind != ui.PointerDeviceKindStylus {
		t.Errorf("the contact = %+v", d[2])
	}
	if d[0].Pressure != 0 || d[4].Pressure != 0 || d[5].Pressure != 0 {
		t.Error("add, up and remove carry pressure")
	}
	// The eraser end is its own kind, and a pen that leaves while down is lifted first.
	m.stylusEnter(in, ui.PointerDeviceKindInvertedStylus, dev)
	m.stylusMove(in, ui.PointerDeviceKindInvertedStylus, dev, buttonPrimary)
	m.stylusLeave(in, ui.PointerDeviceKindInvertedStylus, dev)
	got := summaries(m.take())
	want = []summary{{change: add, device: dev}, {change: down, device: dev, buttons: 1}, {change: up, device: dev, synth: true}, {change: remove, device: dev}}
	if !slices.Equal(got, want) {
		t.Errorf("eraser: got %+v, want %+v", got, want)
	}
}

func TestCancelViewLiftsAndRemovesEveryPointer(t *testing.T) {
	m := newPointerMachine()
	m.entered(at(5, 5, 0))
	m.buttonDown(at(5, 5, 1), buttonPrimary)
	m.stylusEnter(at(6, 6, 2), ui.PointerDeviceKindStylus, stylusDeviceBase)
	m.take()
	m.cancelView(1, 10*time.Millisecond)
	got := summaries(m.take())
	want := []summary{{change: up, synth: true}, {change: remove}, {change: remove, device: stylusDeviceBase}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	m.cancelView(1, 20*time.Millisecond)
	if d := m.take(); len(d) != 0 {
		t.Error("a second cancel made data")
	}
}
