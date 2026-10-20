package input

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// Flutter's button bits (gestures/constants.dart).
const (
	buttonPrimary   = 0x01 // kPrimaryButton: the left button
	buttonSecondary = 0x02 // kSecondaryButton: the right button
	buttonMiddle    = 0x04 // kMiddleMouseButton
	buttonBack      = 0x08 // kBackMouseButton
	buttonForward   = 0x10 // kForwardMouseButton
)

// buttonBit maps a Linux button code (linux/input-event-codes.h) to Flutter's
// button bit, and 0 for a button Flutter has no bit for.
func buttonBit(code uint32) int {
	switch code {
	case 0x110: // BTN_LEFT
		return buttonPrimary
	case 0x111: // BTN_RIGHT
		return buttonSecondary
	case 0x112: // BTN_MIDDLE
		return buttonMiddle
	case 0x113, 0x116: // BTN_SIDE, BTN_BACK
		return buttonBack
	case 0x114, 0x115: // BTN_EXTRA, BTN_FORWARD
		return buttonForward
	}
	return 0
}

// scrollStep is how far one notch of a mouse wheel scrolls, in logical pixels
// (kScrollOffsetMultiplier of Flutter's Linux embedder).
const scrollStep = 53.0

// The device ids of the pointers (Flutter tells pointers apart by device).
const (
	// MouseDevice is the device id of the mouse.
	//
	// Flutter: no counterpart
	MouseDevice = 0
	// TrackpadDevice is the device id of the pan/zoom gestures of a trackpad.
	//
	// Flutter: no counterpart
	TrackpadDevice = 1
	// TouchDeviceBase is added to the id of a finger to make its device id.
	//
	// Flutter: no counterpart
	TouchDeviceBase = 1000
	// StylusDeviceBase is added to the id of a tablet tool to make its device id.
	//
	// Flutter: no counterpart
	StylusDeviceBase = 2000
)

// panKind says what started the pan/zoom gesture that is in progress.
type panKind int

const (
	panNone panKind = iota
	panScroll
	panPinch
	panSwipe
)

// Axis is a scroll axis.
//
// Flutter: no counterpart
type Axis int

// The scroll axes.
const (
	// AxisVertical is the vertical axis; positive values scroll down.
	//
	// Flutter: no counterpart
	AxisVertical Axis = iota
	// AxisHorizontal is the horizontal axis; positive values scroll right.
	//
	// Flutter: no counterpart
	AxisHorizontal
)

// AxisSource is what produced scroll events.
//
// Flutter: no counterpart
type AxisSource int

// The sources of scroll events, as Wayland's wl_pointer.axis_source has them.
const (
	// AxisSourceWheel is a physical wheel: a notch is a step.
	//
	// Flutter: no counterpart
	AxisSourceWheel AxisSource = iota
	// AxisSourceFinger is a finger on a touchpad.
	//
	// Flutter: no counterpart
	AxisSourceFinger
	// AxisSourceContinuous is a continuous coordinate space, in pixels.
	//
	// Flutter: no counterpart
	AxisSourceContinuous
	// AxisSourceWheelTilt is a wheel tilted sideways.
	//
	// Flutter: no counterpart
	AxisSourceWheelTilt
)

// axisFrame is what one scroll axis collected since the last frame.
type axisFrame struct {
	value    float64 // wl_pointer.axis, in logical pixels
	v120     int32   // wl_pointer.axis_value120
	discrete int32   // wl_pointer.axis_discrete (compositors before version 8)
	seen     bool    // an axis, value120 or discrete event arrived
	stopped  bool    // wl_pointer.axis_stop arrived
}

// PointerBatcher turns the events of a pointer, which the compositor groups in
// frames (Wayland's wl_pointer.frame; X11 sends one event at a time), into
// ui.PointerData. Enter, leave, motion, buttons and gestures are
// converted when they arrive; the scroll axes are held until frame, because only
// then it is known what the event was (a wheel, or a touchpad). The data
// belongs to the window the pointer is in. It works in physical pixels.
//
// A PointerBatcher is used on the UI thread only.
//
// Flutter: no counterpart
type PointerBatcher struct {
	now func() time.Duration
	// Scale is the number of physical pixels per logical pixel; the positions the
	// batcher gets are logical.
	Scale float64

	view    ui.ViewID
	inside  bool
	x, y    float64 // the position, physical
	buttons int

	source AxisSource
	// hasSource is whether axis_source arrived in this frame.
	hasSource bool
	axes      [2]axisFrame // AxisVertical, AxisHorizontal
	// inverted is whether the physical direction of an axis is the opposite of
	// what the events say (natural scrolling). The events already have the user's
	// direction, so it changes nothing here; it is recorded for whoever needs the
	// physical direction.
	inverted [2]bool

	pan        panKind
	panX, panY float64 // the translation of the gesture so far

	out []ui.PointerData
}

// NewPointerBatcher makes a batcher whose events carry the times that now gives.
//
// Flutter: no counterpart
func NewPointerBatcher(now func() time.Duration) *PointerBatcher {
	return &PointerBatcher{now: now, Scale: 1}
}

// base returns the data every event of the pointer has.
func (b *PointerBatcher) base(change ui.PointerChange) ui.PointerData {
	return ui.PointerData{
		ViewID:    b.view,
		TimeStamp: b.now(),
		Change:    change,
		Kind:      ui.PointerDeviceKindMouse,
		Device:    MouseDevice,
		PhysicalX: b.x,
		PhysicalY: b.y,
		Buttons:   b.buttons,
	}
}

func (b *PointerBatcher) emit(d ui.PointerData) { b.out = append(b.out, d) }

// Enter reports that the pointer entered the window view at the position, in
// logical pixels.
//
// Flutter: no counterpart
func (b *PointerBatcher) Enter(view ui.ViewID, x, y float64) {
	if b.inside {
		b.Leave()
	}
	b.view, b.inside = view, true
	b.x, b.y = x*b.Scale, y*b.Scale
	b.buttons = 0
	b.emit(b.base(ui.PointerChangeAdd))
}

// Leave reports that the pointer left its window. Buttons that are still down
// are released first (synthesized), and the pointer is removed.
//
// Flutter: no counterpart
func (b *PointerBatcher) Leave() {
	if !b.inside {
		return
	}
	b.endPan()
	if b.buttons != 0 {
		b.buttons = 0
		up := b.base(ui.PointerChangeUp)
		up.Synthesized = true
		b.emit(up)
	}
	b.emit(b.base(ui.PointerChangeRemove))
	b.inside = false
	b.axes = [2]axisFrame{}
	b.hasSource = false
}

// Motion reports the pointer at a new position, in logical pixels.
//
// Flutter: no counterpart
func (b *PointerBatcher) Motion(x, y float64) {
	if !b.inside {
		return
	}
	nx, ny := x*b.Scale, y*b.Scale
	change := ui.PointerChangeHover
	if b.buttons != 0 {
		change = ui.PointerChangeMove
	}
	d := b.base(change)
	d.PhysicalX, d.PhysicalY = nx, ny
	d.PhysicalDeltaX, d.PhysicalDeltaY = nx-b.x, ny-b.y
	b.x, b.y = nx, ny
	b.emit(d)
}

// Button reports a button press or release, by its Linux button code. The first
// button down is a down event, the last one up an up event; the buttons in
// between change the button state with a move, as Flutter's embedders do.
//
// Flutter: no counterpart
func (b *PointerBatcher) Button(code uint32, pressed bool) {
	bit := buttonBit(code)
	if !b.inside || bit == 0 {
		return
	}
	was := b.buttons
	if pressed {
		b.buttons |= bit
	} else {
		b.buttons &^= bit
	}
	if b.buttons == was {
		return
	}
	switch {
	case pressed && was == 0:
		b.emit(b.base(ui.PointerChangeDown))
	case !pressed && b.buttons == 0:
		b.emit(b.base(ui.PointerChangeUp))
	default:
		b.emit(b.base(ui.PointerChangeMove))
	}
}

// AxisSource says what the scroll events of this frame come from.
//
// Flutter: no counterpart
func (b *PointerBatcher) AxisSource(src AxisSource) {
	b.source, b.hasSource = src, true
}

func axisIndex(axis Axis) int {
	if axis == AxisHorizontal {
		return 1
	}
	return 0
}

// Axis adds scroll in logical pixels on an axis to this frame.
//
// Flutter: no counterpart
func (b *PointerBatcher) Axis(axis Axis, value float64) {
	a := &b.axes[axisIndex(axis)]
	a.value += value
	a.seen = true
}

// AxisValue120 adds high-resolution wheel steps (120 is one notch) on an axis to
// this frame.
//
// Flutter: no counterpart
func (b *PointerBatcher) AxisValue120(axis Axis, v int32) {
	a := &b.axes[axisIndex(axis)]
	a.v120 += v
	a.seen = true
}

// AxisDiscrete adds whole wheel notches on an axis to this frame, for
// compositors that have no high-resolution steps.
//
// Flutter: no counterpart
func (b *PointerBatcher) AxisDiscrete(axis Axis, n int32) {
	a := &b.axes[axisIndex(axis)]
	a.discrete += n
	a.seen = true
}

// AxisStop says the scrolling on an axis has stopped: for a touchpad, the
// fingers lifted.
//
// Flutter: no counterpart
func (b *PointerBatcher) AxisStop(axis Axis) {
	a := &b.axes[axisIndex(axis)]
	a.stopped = true
	a.seen = true
}

// AxisDirection records whether the physical direction of an axis is the
// opposite of what its events say (natural scrolling).
//
// Flutter: no counterpart
func (b *PointerBatcher) AxisDirection(axis Axis, inverted bool) {
	b.inverted[axisIndex(axis)] = inverted
}

// Frame ends a group of events: it turns the collected scroll events into data
// and returns everything since the last frame, in order.
//
// Flutter: no counterpart
func (b *PointerBatcher) Frame() []ui.PointerData {
	b.flushAxes()
	return b.Take()
}

// Take returns the data converted since the last call, without ending the frame.
//
// Flutter: no counterpart
func (b *PointerBatcher) Take() []ui.PointerData {
	out := b.out
	b.out = nil
	return out
}

// flushAxes converts the scroll axes of the frame. A touchpad (finger source)
// makes a pan/zoom gesture: it starts with the first scroll event and ends with
// axis_stop, and the pan follows the fingers, so it is the opposite of the axis
// values. Anything else is a scroll signal: a wheel scrolls scrollStep logical
// pixels per notch (the high-resolution value120 is a 120th of a notch), and a
// continuous source scrolls by its pixels.
func (b *PointerBatcher) flushAxes() {
	axes, source, hasSource := b.axes, b.source, b.hasSource
	b.axes = [2]axisFrame{}
	b.hasSource = false
	if !b.inside || (!axes[0].seen && !axes[1].seen) {
		return
	}
	if hasSource && source == AxisSourceFinger {
		dx, dy := -axes[1].value*b.Scale, -axes[0].value*b.Scale
		if (dx != 0 || dy != 0) && b.pan == panNone {
			b.startPan(panScroll)
		}
		if dx != 0 || dy != 0 {
			b.updatePan(dx, dy, 1, 0)
		}
		if axes[0].stopped || axes[1].stopped {
			b.endPanOf(panScroll)
		}
		return
	}
	var delta [2]float64
	for i, a := range axes {
		switch {
		case a.v120 != 0:
			delta[i] = float64(a.v120) / 120 * scrollStep * b.Scale
		case a.discrete != 0:
			delta[i] = float64(a.discrete) * scrollStep * b.Scale
		default:
			delta[i] = a.value * b.Scale
		}
	}
	if delta[0] == 0 && delta[1] == 0 {
		return
	}
	d := b.base(ui.PointerChangeHover)
	d.SignalKind = ui.PointerSignalKindScroll
	d.ScrollDeltaX, d.ScrollDeltaY = delta[1], delta[0]
	b.emit(d)
}

// panData is the data of a pan/zoom event of the trackpad.
func (b *PointerBatcher) panData(change ui.PointerChange) ui.PointerData {
	d := b.base(change)
	d.Kind = ui.PointerDeviceKindTrackpad
	d.Device = TrackpadDevice
	d.Buttons = 0
	d.Scale = 1
	return d
}

// startPan begins a pan/zoom gesture; one that is still going is ended first.
func (b *PointerBatcher) startPan(kind panKind) {
	b.endPan()
	b.pan = kind
	b.panX, b.panY = 0, 0
	b.emit(b.panData(ui.PointerChangePanZoomStart))
}

// updatePan reports the gesture's translation since the last report (physical
// pixels), its scale and its rotation in radians.
func (b *PointerBatcher) updatePan(dx, dy, scale, rotation float64) {
	b.panX += dx
	b.panY += dy
	d := b.panData(ui.PointerChangePanZoomUpdate)
	d.PanX, d.PanY = b.panX, b.panY
	d.PanDeltaX, d.PanDeltaY = dx, dy
	d.Scale, d.Rotation = scale, rotation
	b.emit(d)
}

func (b *PointerBatcher) endPan() { b.endPanOf(b.pan) }

// endPanOf ends the gesture if it is of the kind.
func (b *PointerBatcher) endPanOf(kind panKind) {
	if b.pan == panNone || b.pan != kind {
		return
	}
	b.pan = panNone
	b.emit(b.panData(ui.PointerChangePanZoomEnd))
}

// PinchBegin starts a touchpad pinch gesture; PinchUpdate and PinchEnd follow
// it.
//
// Flutter: no counterpart
func (b *PointerBatcher) PinchBegin() {
	if b.inside {
		b.startPan(panPinch)
	}
}

// PinchUpdate gives the movement of the gesture's focus point in logical pixels,
// the scale since the start and the rotation since the start in degrees.
//
// Flutter: no counterpart
func (b *PointerBatcher) PinchUpdate(dx, dy, scale, rotationDegrees float64) {
	if b.pan == panPinch {
		b.updatePan(dx*b.Scale, dy*b.Scale, scale, rotationDegrees*math.Pi/180)
	}
}

// PinchEnd ends the pinch gesture.
//
// Flutter: no counterpart
func (b *PointerBatcher) PinchEnd() { b.endPanOf(panPinch) }

// SwipeBegin starts a touchpad swipe, which is a pan without scale or
// rotation; SwipeUpdate and SwipeEnd follow it.
//
// Flutter: no counterpart
func (b *PointerBatcher) SwipeBegin() {
	if b.inside {
		b.startPan(panSwipe)
	}
}

// SwipeUpdate gives the movement of the swipe in logical pixels.
//
// Flutter: no counterpart
func (b *PointerBatcher) SwipeUpdate(dx, dy float64) {
	if b.pan == panSwipe {
		b.updatePan(dx*b.Scale, dy*b.Scale, 1, 0)
	}
}

// SwipeEnd ends the swipe gesture.
//
// Flutter: no counterpart
func (b *PointerBatcher) SwipeEnd() { b.endPanOf(panSwipe) }
