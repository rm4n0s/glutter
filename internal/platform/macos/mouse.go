package macos

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the state machine that turns what the mouse, the trackpad and a tablet say into the pointer
// data that the engine takes (SPEC §29.6). It is plain Go, so that its rules are tested on every host; the
// glue that reads an NSEvent and calls it is in input_darwin.go.

// Flutter's button bits (gestures/constants.dart), as the embedder and the framework use them.
const (
	buttonPrimary   = 0x01 // the left button
	buttonSecondary = 0x02 // the right button, and a Control-click
	buttonMiddle    = 0x04
	buttonBack      = 0x08
	buttonForward   = 0x10
)

// The device ids: the mouse and the trackpad's pan and zoom have one each, since the device cannot be told
// (FlutterViewController.mm kMousePointerDeviceId and kPointerPanZoomDeviceId); a stylus has its own.
const (
	mouseDevice      = 0
	panZoomDevice    = 1
	stylusDeviceBase = 100
)

// scrollPixelsPerLine is what a line of a notched wheel is worth in points; Chromium's value, since the
// system's own (10) is noticeably slower than other apps (FlutterViewController.mm).
const scrollPixelsPerLine = 40.0

// inertiaCancelWindow is how soon after the last momentum scroll a touch of the trackpad stops it
// (kTrackpadTouchInertiaCancelWindowMs, in seconds, which covers the delays of macOS Ventura).
const inertiaCancelWindow = 50 * time.Millisecond

// evPhase is NSEventPhase: the values are powers of two, so that phases can be merged.
type evPhase uint

// The values of NSEventPhase.
const (
	phaseNone       evPhase = 0
	phaseBegan      evPhase = 1 << 0
	phaseStationary evPhase = 1 << 1
	phaseChanged    evPhase = 1 << 2
	phaseEnded      evPhase = 1 << 3
	phaseCancelled  evPhase = 1 << 4
	phaseMayBegin   evPhase = 1 << 5
)

// gestureKind is the event that makes a pan/zoom gesture: the scroll of two fingers, a pinch or a rotation.
type gestureKind int

const (
	gesturePan gestureKind = iota
	gestureMagnify
	gestureRotate
)

// pointerInput is what an event says: where it is in the view (physical pixels, top-left origin), when it
// happened, and what the scroll, the pinch or the rotation carry.
type pointerInput struct {
	view ui.ViewID
	x, y float64
	// time is when the event happened, as a duration since the platform started.
	time time.Duration
	// shift says that Shift was down (macOS swaps the axes of a wheel then).
	shift bool
	// scale is the backing scale of the view, which turns points into physical pixels.
	scale float64
	// dx and dy are scrollingDeltaX and scrollingDeltaY in points; precise says they come from a
	// trackpad or a smooth wheel (hasPreciseScrollingDeltas).
	dx, dy  float64
	precise bool
	// phase and momentum are the phase and the momentum phase of a scroll; for a pinch or a rotation
	// only phase is used.
	phase, momentum evPhase
	// magnification is the change of a pinch (a fraction of the size), rotation the change of a rotation in
	// degrees, counter-clockwise.
	magnification, rotation float64
	// detail is the pressure, tilt and orientation of a stylus; zero for a mouse.
	detail pointerDetail
}

// pointerDetail is what a tablet says about the contact besides its place and buttons.
type pointerDetail struct {
	// pressure is from 0 to 1, with pressureMax 1 for a stylus and both 0 for a mouse.
	pressure, pressureMax float64
	// tilt is the angle from the normal of the surface in radians, orientation the direction of the lean.
	tilt, orientation float64
}

// pointerKey tells the pointers apart: the kind of device and its id.
type pointerKey struct {
	kind   ui.PointerDeviceKind
	device int
}

// pointerState is what the platform knows about one pointer.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterViewController.mm MouseState
type pointerState struct {
	kind   ui.PointerDeviceKind
	device int
	view   ui.ViewID
	// buttons are the buttons that are down; down says the framework has had a down for the pointer
	// (flutter_state_is_down) and added an add (flutter_state_is_added).
	buttons     int
	down, added bool
	x, y        float64
	// identifier identifies the pointer in the framework while it is in contact; it is renewed at each down.
	identifier int
	detail     pointerDetail
}

// pointerMachine keeps the state of the mouse, the trackpad's gestures and the stylus of the app's windows
// and makes the data in the order the framework expects (add, hover, down, move, up, remove; the pan and
// zoom with its start, updates and end).
//
// Deviation: Flutter's embedder hands the engine a FlutterPointerEvent and the engine's
// PointerDataPacketConverter works out the deltas, the identifiers and the synthesized events; here the
// platform makes the final data, like the Linux and Windows backends. The rules of
// FlutterViewController are kept: the phase comes from the buttons and the "down" state, an add is
// synthesized before the first event of a pointer, the exit of a pointer with a button down is delayed
// until the last button is released, a pan/zoom starts once however many gestures are active and ends
// when the last of them ends, the momentum of a scroll is left to the framework, and a touch after
// momentum is a scroll inertia cancel.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterViewController.mm
type pointerMachine struct {
	states map[pointerKey]*pointerState
	// lastIdentifier is the last identifier given; they start at 1.
	lastIdentifier int
	out            []ui.PointerData

	// The state of the pan/zoom gesture of the trackpad (one for the app: there is one trackpad).
	panStarted                     bool
	panPhase, scalePhase, rotPhase evPhase
	panX, panY, zoom, rotation     float64
	lastPanX, lastPanY             float64
	panView                        ui.ViewID
	panIdentifier                  int
	lastMomentum                   time.Duration
	haveMomentum                   bool
	pendingExit                    bool
	lastMouseX, lastMouseY         float64
	mouseView                      ui.ViewID
}

func newPointerMachine() *pointerMachine {
	return &pointerMachine{states: map[pointerKey]*pointerState{}}
}

// take returns the data made since the last call, in order.
func (m *pointerMachine) take() []ui.PointerData {
	out := m.out
	m.out = nil
	return out
}

func (m *pointerMachine) mouse(view ui.ViewID) *pointerState {
	key := pointerKey{ui.PointerDeviceKindMouse, mouseDevice}
	st, ok := m.states[key]
	if ok && st.view != view {
		// The mouse went to another window without the first being told: it leaves there.
		m.removeState(st, st.x, st.y, 0)
		ok = false
	}
	if !ok {
		st = &pointerState{kind: ui.PointerDeviceKindMouse, device: mouseDevice, view: view}
		m.states[key] = st
	}
	return st
}

// phaseOf is the change for the buttons and the "down" state of the mouse.
//
// Flutter: FlutterViewController.mm dispatchMouseEvent:
func phaseOf(st *pointerState) ui.PointerChange {
	if st.buttons == 0 {
		if st.down {
			return ui.PointerChangeUp
		}
		return ui.PointerChangeHover
	}
	if st.down {
		return ui.PointerChangeMove
	}
	return ui.PointerChangeDown
}

// entered is mouseEntered:. An enter after a delayed exit (the pointer came back while a button was
// down) just cancels the exit.
//
// Flutter: FlutterViewController.mm mouseEntered:
func (m *pointerMachine) entered(in pointerInput) {
	st := m.mouse(in.view)
	if m.pendingExit {
		m.pendingExit = false
		return
	}
	m.send(st, ui.PointerChangeAdd, in, nil)
}

// exited is mouseExited:. With a button down, the remove waits for the last button to be released.
//
// Flutter: FlutterViewController.mm mouseExited:
func (m *pointerMachine) exited(in pointerInput) {
	st, ok := m.states[pointerKey{ui.PointerDeviceKindMouse, mouseDevice}]
	if !ok || st.view != in.view {
		return
	}
	if st.buttons != 0 {
		m.pendingExit = true
		return
	}
	m.send(st, ui.PointerChangeRemove, in, nil)
}

// buttonDown is a mouseDown:, rightMouseDown: or otherMouseDown:.
//
// Flutter: FlutterViewController.mm mouseDown:
func (m *pointerMachine) buttonDown(in pointerInput, bit int) {
	st := m.mouse(in.view)
	st.buttons |= bit
	m.dispatchMouse(st, in)
}

// buttonUp is a mouseUp:, rightMouseUp: or otherMouseUp:.
func (m *pointerMachine) buttonUp(in pointerInput, bit int) {
	st := m.mouse(in.view)
	st.buttons &^= bit
	m.dispatchMouse(st, in)
}

// moved is mouseMoved: and the mouseDragged: of any button.
func (m *pointerMachine) moved(in pointerInput) {
	m.dispatchMouse(m.mouse(in.view), in)
}

// dispatchMouse sends the change that the buttons and the "down" state ask for.
func (m *pointerMachine) dispatchMouse(st *pointerState, in pointerInput) {
	m.send(st, phaseOf(st), in, nil)
}

// scrollWheel is scrollWheel:. A precise scroll (a trackpad) is a pan/zoom gesture, made from the phases,
// and its momentum is dropped: the framework's own ballistic scroll takes over. A notched wheel is a scroll
// signal in pixels, with the axes swapped back when Shift is down.
//
// Flutter: FlutterViewController.mm dispatchGestureEvent: and dispatchMouseEvent:phase:
func (m *pointerMachine) scrollWheel(in pointerInput) {
	if !in.precise {
		// A notched wheel has no phases: it is a signal.
		m.wheelSignal(in)
		return
	}
	m.gesture(gesturePan, in)
}

// magnify is magnifyWithEvent: and rotate is rotateWithEvent: (a pinch and a rotation of the trackpad).
func (m *pointerMachine) magnify(in pointerInput) { m.gesture(gestureMagnify, in) }
func (m *pointerMachine) rotate(in pointerInput)  { m.gesture(gestureRotate, in) }

// gesture is dispatchGestureEvent: a phase of a gesture of the trackpad.
func (m *pointerMachine) gesture(kind gestureKind, in pointerInput) {
	switch {
	case in.phase == phaseBegan || in.phase == phaseMayBegin:
		m.panZoom(kind, ui.PointerChangePanZoomStart, in)
	case in.phase == phaseChanged:
		m.panZoom(kind, ui.PointerChangePanZoomUpdate, in)
	case in.phase == phaseEnded || in.phase == phaseCancelled:
		m.panZoom(kind, ui.PointerChangePanZoomEnd, in)
	case in.phase == phaseNone && in.momentum == phaseNone:
		// A scroll with no phases at all (a mouse with a smooth wheel, or a gesture that began before
		// the app looked): the wheel's signal.
		if kind == gesturePan {
			m.wheelSignal(in)
		}
	default:
		// The momentum of a scroll. Waiting for its first change is Flutter's workaround for a touchesBegan:
		// that arrives between the start of the momentum and its first change in low power mode.
		if in.momentum == phaseChanged {
			m.lastMomentum, m.haveMomentum = in.time, true
		}
	}
}

// panZoom is dispatchMouseEvent:phase: for the three phases of a pan/zoom.
func (m *pointerMachine) panZoom(kind gestureKind, change ui.PointerChange, in pointerInput) {
	if change == ui.PointerChangePanZoomStart || change == ui.PointerChangePanZoomEnd {
		switch kind {
		case gesturePan:
			m.panPhase = in.phase
		case gestureMagnify:
			m.scalePhase = in.phase
		case gestureRotate:
			m.rotPhase = in.phase
		}
	}
	if change == ui.PointerChangePanZoomStart {
		if kind == gesturePan {
			m.haveMomentum = false // no scroll inertia cancel afterwards
		}
		if m.panStarted {
			return // already started by another kind of gesture
		}
		m.panStarted = true
	}
	if change == ui.PointerChangePanZoomEnd {
		if !m.panStarted {
			return // a cancel that comes at the wrong time, with no gesture to cancel
		}
		active := phaseBegan | phaseChanged
		if (m.panPhase|m.scalePhase|m.rotPhase)&active != 0 {
			return // another kind of gesture is still active
		}
	}
	// The trackpad has a device of its own, and an add of its own comes first.
	pz := m.panZoomState(in.view)
	if !pz.added {
		m.send(pz, ui.PointerChangeAdd, in, nil)
	}
	switch change {
	case ui.PointerChangePanZoomStart:
		m.lastIdentifier++
		m.panIdentifier = m.lastIdentifier
		m.send(pz, change, in, func(d *ui.PointerData) { d.Scale = 1 })
	case ui.PointerChangePanZoomUpdate:
		switch kind {
		case gesturePan:
			m.panX += in.dx * in.scale
			m.panY += in.dy * in.scale
		case gestureMagnify:
			m.zoom += in.magnification
		case gestureRotate:
			m.rotation += in.rotation * (-math.Pi / 180)
		}
		m.send(pz, change, in, func(d *ui.PointerData) {
			d.PanX, d.PanY = m.panX, m.panY
			d.PanDeltaX, d.PanDeltaY = m.panX-m.lastPanX, m.panY-m.lastPanY
			// The scale is normalized to the range 0 to infinity.
			d.Scale = math.Pow(2, m.zoom)
			d.Rotation = m.rotation
		})
		m.lastPanX, m.lastPanY = m.panX, m.panY
	case ui.PointerChangePanZoomEnd:
		m.send(pz, change, in, func(d *ui.PointerData) { d.Scale = 1 })
		m.resetGesture()
	}
}

// resetGesture forgets the accumulated gesture (MouseState::GestureReset).
func (m *pointerMachine) resetGesture() {
	m.panStarted = false
	m.panPhase, m.scalePhase, m.rotPhase = phaseNone, phaseNone, phaseNone
	m.panX, m.panY, m.lastPanX, m.lastPanY, m.zoom, m.rotation = 0, 0, 0, 0, 0, 0
}

func (m *pointerMachine) panZoomState(view ui.ViewID) *pointerState {
	key := pointerKey{ui.PointerDeviceKindTrackpad, panZoomDevice}
	st, ok := m.states[key]
	if ok && st.view != view {
		m.removeState(st, st.x, st.y, 0)
		ok = false
	}
	if !ok {
		st = &pointerState{kind: ui.PointerDeviceKindTrackpad, device: panZoomDevice, view: view}
		m.states[key] = st
	}
	return st
}

// wheelSignal is a scroll signal of a notched wheel: pixels, and the axes swapped back when Shift is down
// (macOS flips them, and the other platforms do not).
func (m *pointerMachine) wheelSignal(in pointerInput) {
	st := m.mouse(in.view)
	per := scrollPixelsPerLine
	if in.precise {
		per = 1
	}
	dx, dy := -in.dx*per*in.scale, -in.dy*per*in.scale
	if in.shift {
		dx, dy = dy, dx
	}
	change := phaseOf(st)
	m.send(st, change, in, func(d *ui.PointerData) {
		d.SignalKind = ui.PointerSignalKindScroll
		d.ScrollDeltaX, d.ScrollDeltaY = dx, dy
		d.PhysicalDeltaX, d.PhysicalDeltaY = 0, 0
	})
	// A signal does not move the pointer.
	if st.added {
		st.x, st.y = in.x, in.y
	}
}

// touchesBegan is touchesBeganWithEvent: a touch of the trackpad right after the momentum of a scroll stops
// it: the framework gets a scroll inertia cancel.
//
// Flutter: FlutterViewController.mm touchesBeganWithEvent:
func (m *pointerMachine) touchesBegan(in pointerInput) {
	if !m.haveMomentum || in.time-m.lastMomentum >= inertiaCancelWindow {
		return
	}
	pz := m.panZoomState(in.view)
	d := m.data(pz, ui.PointerChangeHover, in)
	d.SignalKind = ui.PointerSignalKindScrollInertiaCancel
	d.Buttons = 0
	m.out = append(m.out, d)
	m.haveMomentum = false
}

// stylus reports a stylus in proximity or in contact. enter adds it, leave removes it; the buttons are the
// ones that are down (the tip is the primary button).
func (m *pointerMachine) stylusState(view ui.ViewID, kind ui.PointerDeviceKind, device int) *pointerState {
	key := pointerKey{kind, device}
	st, ok := m.states[key]
	if ok && st.view != view {
		m.removeState(st, st.x, st.y, 0)
		ok = false
	}
	if !ok {
		st = &pointerState{kind: kind, device: device, view: view}
		m.states[key] = st
	}
	return st
}

// stylusEnter is a pen coming into range of the tablet: an add at its place.
func (m *pointerMachine) stylusEnter(in pointerInput, kind ui.PointerDeviceKind, device int) {
	st := m.stylusState(in.view, kind, device)
	st.detail = in.detail
	m.send(st, ui.PointerChangeAdd, in, nil)
}

// stylusLeave is a pen going out of range: lifted first if it was down, then removed.
func (m *pointerMachine) stylusLeave(in pointerInput, kind ui.PointerDeviceKind, device int) {
	st, ok := m.states[pointerKey{kind, device}]
	if !ok {
		return
	}
	m.removeState(st, in.x, in.y, in.time)
}

// stylusMove is a pen that moved or whose buttons changed: the buttons are the ones that are down.
func (m *pointerMachine) stylusMove(in pointerInput, kind ui.PointerDeviceKind, device, buttons int) {
	st := m.stylusState(in.view, kind, device)
	st.buttons = buttons
	st.detail = in.detail
	m.send(st, phaseOf(st), in, nil)
}

// removeState lifts the pointer, at its last place, when it is down, and removes it at the place given.
func (m *pointerMachine) removeState(st *pointerState, x, y float64, at time.Duration) {
	in := pointerInput{view: st.view, x: x, y: y, time: at}
	if st.added {
		if st.down {
			st.buttons = 0
			d := m.data(st, ui.PointerChangeUp, pointerInput{view: st.view, x: st.x, y: st.y, time: at})
			d.Synthesized = true
			m.out = append(m.out, d)
			st.down = false
		}
		m.out = append(m.out, m.data(st, ui.PointerChangeRemove, in))
	}
	delete(m.states, pointerKey{st.kind, st.device})
	if st.kind == ui.PointerDeviceKindMouse {
		m.pendingExit = false
	}
}

// cancelView lifts and removes every pointer of the view (a window that goes away).
func (m *pointerMachine) cancelView(view ui.ViewID, at time.Duration) {
	for _, st := range m.sortedStates() {
		if st.view == view {
			m.removeState(st, st.x, st.y, at)
		}
	}
	m.resetGesture()
}

func (m *pointerMachine) sortedStates() []*pointerState {
	var out []*pointerState
	for _, st := range m.states {
		out = append(out, st)
	}
	// By kind and device, so that the order does not depend on the map.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && lessState(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func lessState(a, b *pointerState) bool {
	if a.kind != b.kind {
		return a.kind < b.kind
	}
	return a.device < b.device
}

// send makes the data of a change, first the add that an unknown pointer needs; an add for a pointer that
// has one is dropped (the system sometimes delivers an enter after the events that follow it).
//
// Flutter: FlutterViewController.mm dispatchMouseEvent:phase:
func (m *pointerMachine) send(st *pointerState, change ui.PointerChange, in pointerInput, edit func(*ui.PointerData)) {
	if change == ui.PointerChangeAdd && st.added {
		return
	}
	if !st.added && change != ui.PointerChangeAdd {
		d := m.data(st, ui.PointerChangeAdd, in)
		d.Buttons = 0 // a click that made the add does not say its buttons in it
		m.out = append(m.out, d)
		st.added, st.x, st.y = true, in.x, in.y
	}
	if change == ui.PointerChangeDown {
		m.lastIdentifier++
		st.identifier = m.lastIdentifier
	}
	if change == ui.PointerChangePanZoomStart {
		st.identifier = m.panIdentifier
	}
	d := m.data(st, change, in)
	if edit != nil {
		edit(&d)
	}
	m.out = append(m.out, d)
	st.x, st.y = in.x, in.y
	switch change {
	case ui.PointerChangeAdd:
		st.added = true
	case ui.PointerChangeDown:
		st.down = true
	case ui.PointerChangeUp:
		st.down = false
		if m.pendingExit && st.kind == ui.PointerDeviceKindMouse {
			// Cocoa kept sending drags after the exit; the remove that was waiting goes out now.
			m.pendingExit = false
			m.removeState(st, in.x, in.y, in.time)
		}
	}
}

// data makes the data of a change at a place from the state of the pointer.
func (m *pointerMachine) data(st *pointerState, change ui.PointerChange, in pointerInput) ui.PointerData {
	d := ui.PointerData{
		ViewID:            st.view,
		TimeStamp:         in.time,
		Change:            change,
		Kind:              st.kind,
		Device:            st.device,
		PointerIdentifier: st.identifier,
		PhysicalX:         in.x,
		PhysicalY:         in.y,
		Buttons:           st.buttons,
		Pressure:          st.detail.pressure,
		PressureMax:       st.detail.pressureMax,
		Tilt:              st.detail.tilt,
		Orientation:       st.detail.orientation,
	}
	if st.added && change != ui.PointerChangeAdd && change != ui.PointerChangeRemove && st.kind != ui.PointerDeviceKindTrackpad {
		d.PhysicalDeltaX, d.PhysicalDeltaY = in.x-st.x, in.y-st.y
	}
	switch change {
	case ui.PointerChangeAdd, ui.PointerChangeUp, ui.PointerChangeRemove, ui.PointerChangeCancel:
		d.Pressure = 0 // nothing presses at these
	}
	if st.kind == ui.PointerDeviceKindTrackpad {
		d.Buttons = 0
	}
	return d
}

// mouseButtons are the buttons of the mouse that are down.
func (m *pointerMachine) mouseButtons() int {
	if st, ok := m.states[pointerKey{ui.PointerDeviceKindMouse, mouseDevice}]; ok {
		return st.buttons
	}
	return 0
}
