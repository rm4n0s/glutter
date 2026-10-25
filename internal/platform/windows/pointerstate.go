package windows

import (
	"sort"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// Flutter's button bits (gestures/constants.dart), as the embedder and the framework use them.
const (
	buttonPrimary   = 0x01 // kPrimaryButton: the left button, a touch, the tip of a pen
	buttonSecondary = 0x02 // kSecondaryButton: the right button, the barrel button of a pen
	buttonMiddle    = 0x04 // kMiddleMouseButton, the eraser button of a pen
	buttonBack      = 0x08 // kBackMouseButton
	buttonForward   = 0x10 // kForwardMouseButton
)

// The device ids of the pointers (Flutter tells pointers apart by device): the mouse is 0, a
// touch contact is touchDeviceBase plus its generated id (SPEC §8.5, the same numbers as Linux),
// and a pen is stylusDeviceBase plus its generated id.
const (
	mouseDevice      = 0
	touchDeviceBase  = 1000
	stylusDeviceBase = 2000
)

// pointerKey tells the pointers apart, as Flutter's embedder does: the kind of device and the
// device id, so that a mouse and a pen with the same id do not clash.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetOrCreatePointerState
type pointerKey struct {
	kind   ui.PointerDeviceKind
	device int
}

// pointerDetail is what a report says about the contact besides its place and buttons.
type pointerDetail struct {
	// pressure is from 0 to pressureMax (Flutter's PointerData.pressure); both are 0 for a mouse.
	pressure, pressureMax float64
	// tilt is the angle from the normal of the surface in radians, orientation the rotation of the
	// pen or the angle of the touch ellipse in radians.
	tilt, orientation float64
	// radiusMajor and radiusMinor are the half axes of the touch ellipse in physical pixels.
	radiusMajor, radiusMinor float64
}

// pointerSample is one report of a pointer, in physical pixels of the client area.
type pointerSample struct {
	kind    ui.PointerDeviceKind
	device  int
	x, y    float64
	buttons int
	detail  pointerDetail
}

// pointerState is what the platform knows about one pointer.
//
// Flutter: shell/platform/windows/flutter_windows_view.h FlutterWindowsView::PointerState
type pointerState struct {
	kind   ui.PointerDeviceKind
	device int
	view   ui.ViewID
	// buttons are the buttons that are down; down is Flutter's flutter_state_is_down (the
	// framework has had a down for this pointer) and added its flutter_state_is_added.
	buttons     int
	down, added bool
	x, y        float64
	// identifier identifies the pointer in the framework while it is in contact; it is renewed at
	// each down.
	identifier int
	detail     pointerDetail
}

// pointerTracker turns what the mouse, touch and pen messages say into ui.PointerData: it keeps
// the state of each pointer and sends the add, hover, down, move, up and remove events in the order
// the framework expects (SPEC §25.6, the rules of the Wayland pointer: an add, then hovers; the
// first button down is a down, the last up an up, in between moves with the new buttons; leaving
// with buttons down releases them before the remove). It has no dependency on Windows, so it
// is tested in the normal run. It is used on the UI thread only.
//
// Deviation: Flutter's embedder hands the engine a FlutterPointerEvent and the engine's
// PointerDataPacketConverter works out the deltas, the identifiers and the synthesized events;
// here the platform makes the final data, like the Linux backends. The rules of
// FlutterWindowsView are kept: the phase comes from the buttons and the "down" state, and an add
// is synthesized before any other event of an unknown pointer. Three details differ: a move that
// ends the contact (no buttons any more) clears the "down" state, which the C++ leaves set; a
// pointer that leaves while it is down is lifted first; and a leave for a pointer that was never
// added is ignored (the C++ adds it to remove it).
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView
type pointerTracker struct {
	now    func() time.Duration
	states map[pointerKey]*pointerState
	// lastIdentifier is the last identifier given; they start at 1.
	lastIdentifier int
	out            []ui.PointerData
}

// newPointerTracker makes a tracker whose events carry the times that now gives.
func newPointerTracker(now func() time.Duration) *pointerTracker {
	return &pointerTracker{now: now, states: map[pointerKey]*pointerState{}}
}

// take returns the data made since the last call, in order.
func (t *pointerTracker) take() []ui.PointerData {
	out := t.out
	t.out = nil
	return out
}

// stateFor finds or makes the state of the pointer for a view. A pointer that is in another view
// (the mouse moved from one window to the next before the first window was told) is lifted and
// removed there first.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetOrCreatePointerState
func (t *pointerTracker) stateFor(view ui.ViewID, kind ui.PointerDeviceKind, device int) *pointerState {
	key := pointerKey{kind, device}
	st, ok := t.states[key]
	if ok && st.view != view {
		t.removeState(st, st.x, st.y)
		ok = false
	}
	if !ok {
		st = &pointerState{kind: kind, device: device, view: view}
		t.states[key] = st
	}
	return st
}

// phase is the change for the buttons and the "down" state of the pointer.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::SetEventPhaseFromCursorButtonState
func phase(st *pointerState) ui.PointerChange {
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

// move reports a pointer that moved or whose buttons changed: the buttons are the ones the report
// says are down.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::OnPointerMove
func (t *pointerTracker) move(view ui.ViewID, s pointerSample) {
	st := t.stateFor(view, s.kind, s.device)
	st.buttons = s.buttons
	st.detail = s.detail
	ph := phase(st)
	t.send(st, ph, s.x, s.y, nil)
	switch ph {
	case ui.PointerChangeDown:
		st.down = true
	case ui.PointerChangeUp:
		st.down = false
	}
}

// down reports buttons that were pressed. Buttons that are 0 do nothing.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::OnPointerDown
func (t *pointerTracker) down(view ui.ViewID, s pointerSample) {
	if s.buttons == 0 {
		return
	}
	st := t.stateFor(view, s.kind, s.device)
	st.buttons |= s.buttons
	st.detail = s.detail
	t.send(st, phase(st), s.x, s.y, nil)
	st.down = true
}

// up reports buttons that were released; 0 means all of them. Nothing is sent when no button was
// down.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::OnPointerUp
func (t *pointerTracker) up(view ui.ViewID, s pointerSample) {
	st := t.stateFor(view, s.kind, s.device)
	released := s.buttons
	if released == 0 {
		released = st.buttons
	}
	if released == 0 {
		return
	}
	st.buttons &^= released
	st.detail = s.detail
	ph := phase(st)
	t.send(st, ph, s.x, s.y, nil)
	if ph == ui.PointerChangeUp {
		st.down = false
	}
}

// enter reports a pointer that came near or into the window without touching it (a pen in range):
// an add at its place. A pointer that is added already does nothing.
func (t *pointerTracker) enter(view ui.ViewID, s pointerSample) {
	st := t.stateFor(view, s.kind, s.device)
	st.detail = s.detail
	t.send(st, ui.PointerChangeAdd, s.x, s.y, nil)
}

// leave reports a pointer that left the window or went out of range: when it is down, it is lifted
// first (synthesized); then it is removed and forgotten. A pointer that is not known (or is in
// another view) is ignored.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::OnPointerLeave
func (t *pointerTracker) leave(view ui.ViewID, kind ui.PointerDeviceKind, device int, x, y float64) {
	st, ok := t.states[pointerKey{kind, device}]
	if !ok || st.view != view {
		return
	}
	t.removeState(st, x, y)
}

// removeState lifts the pointer, at its last place, when it is down, and removes it at the place
// given.
func (t *pointerTracker) removeState(st *pointerState, x, y float64) {
	if st.added {
		if st.down {
			st.buttons = 0
			d := t.data(st, ui.PointerChangeUp, st.x, st.y)
			d.Synthesized = true
			t.out = append(t.out, d)
			st.down = false
		}
		t.out = append(t.out, t.data(st, ui.PointerChangeRemove, x, y))
	}
	delete(t.states, pointerKey{st.kind, st.device})
}

// cancel reports a pointer whose contact was taken away (the capture was lost): when it is down, a
// cancel; then it is removed. A pointer that is not known is ignored.
func (t *pointerTracker) cancel(view ui.ViewID, kind ui.PointerDeviceKind, device int) {
	st, ok := t.states[pointerKey{kind, device}]
	if !ok || st.view != view {
		return
	}
	if st.added && st.down {
		st.buttons = 0
		t.out = append(t.out, t.data(st, ui.PointerChangeCancel, st.x, st.y))
		st.down = false
	}
	t.removeState(st, st.x, st.y)
}

// releaseButtons lifts a pointer whose buttons were taken away without a message for each (the
// capture went to another window): a synthesized up, and the pointer stays where it is.
func (t *pointerTracker) releaseButtons(kind ui.PointerDeviceKind, device int) {
	st, ok := t.states[pointerKey{kind, device}]
	if !ok || !st.down {
		return
	}
	st.buttons = 0
	d := t.data(st, ui.PointerChangeUp, st.x, st.y)
	d.Synthesized = true
	t.out = append(t.out, d)
	st.down = false
}

// cancelView cancels and removes every pointer of the view, in the order of their kinds and
// devices (a window that goes away).
func (t *pointerTracker) cancelView(view ui.ViewID) {
	var keys []pointerKey
	for k, st := range t.states {
		if st.view == view {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		return keys[i].device < keys[j].device
	})
	for _, k := range keys {
		t.cancel(view, k.kind, k.device)
	}
}

// scroll reports a scroll signal at a place, with the deltas in physical pixels (positive scrolls
// down and to the right). The signal is a hover, or a move while a button is down.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::SendScroll
func (t *pointerTracker) scroll(view ui.ViewID, kind ui.PointerDeviceKind, device int, x, y, dx, dy float64) {
	st := t.stateFor(view, kind, device)
	change := ui.PointerChangeHover
	if st.down {
		change = ui.PointerChangeMove
	}
	t.send(st, change, x, y, func(d *ui.PointerData) {
		d.SignalKind = ui.PointerSignalKindScroll
		d.ScrollDeltaX, d.ScrollDeltaY = dx, dy
		d.PhysicalDeltaX, d.PhysicalDeltaY = 0, 0
	})
	// A signal does not move the pointer.
	if st.added {
		st.x, st.y = x, y
	}
}

// send makes the data of a change, first the add that an unknown pointer needs; an add for a
// pointer that has one is dropped.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::SendPointerEventWithData
func (t *pointerTracker) send(st *pointerState, change ui.PointerChange, x, y float64, edit func(*ui.PointerData)) {
	if change == ui.PointerChangeAdd && st.added {
		return
	}
	if !st.added && change != ui.PointerChangeAdd {
		d := t.data(st, ui.PointerChangeAdd, x, y)
		d.Buttons = 0
		t.out = append(t.out, d)
		st.added, st.x, st.y = true, x, y
	}
	if change == ui.PointerChangeDown {
		t.lastIdentifier++
		st.identifier = t.lastIdentifier
	}
	d := t.data(st, change, x, y)
	if edit != nil {
		edit(&d)
	}
	t.out = append(t.out, d)
	st.x, st.y = x, y
	if change == ui.PointerChangeAdd {
		st.added = true
	}
}

// data makes the data of a change at a place from the state of the pointer.
func (t *pointerTracker) data(st *pointerState, change ui.PointerChange, x, y float64) ui.PointerData {
	d := ui.PointerData{
		ViewID:            st.view,
		TimeStamp:         t.now(),
		Change:            change,
		Kind:              st.kind,
		Device:            st.device,
		PointerIdentifier: st.identifier,
		PhysicalX:         x,
		PhysicalY:         y,
		Buttons:           st.buttons,
		Pressure:          st.detail.pressure,
		PressureMax:       st.detail.pressureMax,
		Tilt:              st.detail.tilt,
		Orientation:       st.detail.orientation,
		RadiusMajor:       st.detail.radiusMajor,
		RadiusMinor:       st.detail.radiusMinor,
	}
	if st.added && change != ui.PointerChangeAdd && change != ui.PointerChangeRemove {
		d.PhysicalDeltaX, d.PhysicalDeltaY = x-st.x, y-st.y
	}
	switch change {
	case ui.PointerChangeAdd, ui.PointerChangeUp, ui.PointerChangeRemove, ui.PointerChangeCancel:
		d.Pressure = 0 // nothing presses at these
	}
	return d
}

// mouseDown reports whether a button of the mouse is down.
func (t *pointerTracker) mouseDown() bool {
	st, ok := t.states[pointerKey{ui.PointerDeviceKindMouse, mouseDevice}]
	return ok && st.buttons != 0
}

// mouseButtons are the buttons of the mouse that are down, as the bits of ui.PointerData.Buttons.
func (t *pointerTracker) mouseButtons() int {
	if st, ok := t.states[pointerKey{ui.PointerDeviceKindMouse, mouseDevice}]; ok {
		return st.buttons
	}
	return 0
}
