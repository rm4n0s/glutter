package wayland

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
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

// The device ids of the pointers. Touch points use touchDeviceBase plus their id.
const (
	mouseDevice     = 0
	trackpadDevice  = 1
	touchDeviceBase = 1000
)

// panKind says what started the pan/zoom gesture that is in progress.
type panKind int

const (
	panNone panKind = iota
	panScroll
	panPinch
	panSwipe
)

// axisFrame is what one scroll axis collected since the last frame.
type axisFrame struct {
	value    float64 // wl_pointer.axis, in logical pixels
	v120     int32   // wl_pointer.axis_value120
	discrete int32   // wl_pointer.axis_discrete (compositors before version 8)
	seen     bool    // an axis, value120 or discrete event arrived
	stopped  bool    // wl_pointer.axis_stop arrived
}

// pointerBatcher turns the events of wl_pointer, which the compositor groups in
// frames, into ui.PointerData. Enter, leave, motion, buttons and gestures are
// converted when they arrive; the scroll axes are held until frame, because only
// then it is known what the event was (a wheel, or a touchpad). The data
// belongs to the window the pointer is in. It works in physical pixels.
//
// A pointerBatcher is used on the UI thread only.
type pointerBatcher struct {
	now   func() time.Duration
	scale float64 // physical pixels per logical pixel

	view    ui.ViewID
	inside  bool
	x, y    float64 // the position, physical
	buttons int

	source wl.WlPointerAxisSource
	// hasSource is whether axis_source arrived in this frame.
	hasSource bool
	axes      [2]axisFrame // wl.WlPointerAxisVertical, wl.WlPointerAxisHorizontal
	// inverted is whether the physical direction of an axis is the opposite of
	// what the events say (natural scrolling). The events already have the user's
	// direction, so it changes nothing here; it is recorded for whoever needs the
	// physical direction.
	inverted [2]bool

	pan        panKind
	panX, panY float64 // the translation of the gesture so far

	out []ui.PointerData
}

func newPointerBatcher(now func() time.Duration) *pointerBatcher {
	return &pointerBatcher{now: now, scale: 1}
}

// base returns the data every event of the pointer has.
func (b *pointerBatcher) base(change ui.PointerChange) ui.PointerData {
	return ui.PointerData{
		ViewID:    b.view,
		TimeStamp: b.now(),
		Change:    change,
		Kind:      ui.PointerDeviceKindMouse,
		Device:    mouseDevice,
		PhysicalX: b.x,
		PhysicalY: b.y,
		Buttons:   b.buttons,
	}
}

func (b *pointerBatcher) emit(d ui.PointerData) { b.out = append(b.out, d) }

// enter reports that the pointer entered the window view at the position, in
// logical pixels.
func (b *pointerBatcher) enter(view ui.ViewID, x, y float64) {
	if b.inside {
		b.leave()
	}
	b.view, b.inside = view, true
	b.x, b.y = x*b.scale, y*b.scale
	b.buttons = 0
	b.emit(b.base(ui.PointerChangeAdd))
}

// leave reports that the pointer left its window. Buttons that are still down
// are released first (synthesized), and the pointer is removed.
func (b *pointerBatcher) leave() {
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

// motion reports the pointer at a new position, in logical pixels.
func (b *pointerBatcher) motion(x, y float64) {
	if !b.inside {
		return
	}
	nx, ny := x*b.scale, y*b.scale
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

// button reports a button press or release. The first button down is a down
// event, the last one up an up event; the buttons in between change the button
// state with a move, as Flutter's embedders do.
func (b *pointerBatcher) button(code uint32, pressed bool) {
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

func (b *pointerBatcher) axisSource(src wl.WlPointerAxisSource) {
	b.source, b.hasSource = src, true
}

func axisIndex(axis wl.WlPointerAxis) int {
	if axis == wl.WlPointerAxisHorizontalScroll {
		return 1
	}
	return 0
}

func (b *pointerBatcher) axis(axis wl.WlPointerAxis, value float64) {
	a := &b.axes[axisIndex(axis)]
	a.value += value
	a.seen = true
}

func (b *pointerBatcher) axisValue120(axis wl.WlPointerAxis, v int32) {
	a := &b.axes[axisIndex(axis)]
	a.v120 += v
	a.seen = true
}

func (b *pointerBatcher) axisDiscrete(axis wl.WlPointerAxis, n int32) {
	a := &b.axes[axisIndex(axis)]
	a.discrete += n
	a.seen = true
}

func (b *pointerBatcher) axisStop(axis wl.WlPointerAxis) {
	a := &b.axes[axisIndex(axis)]
	a.stopped = true
	a.seen = true
}

func (b *pointerBatcher) axisDirection(axis wl.WlPointerAxis, dir wl.WlPointerAxisRelativeDirection) {
	b.inverted[axisIndex(axis)] = dir == wl.WlPointerAxisRelativeDirectionInverted
}

// frame ends a group of events: it turns the collected scroll events into data
// and returns everything since the last frame, in order.
func (b *pointerBatcher) frame() []ui.PointerData {
	b.flushAxes()
	return b.take()
}

// take returns the data converted since the last call, without ending the frame.
func (b *pointerBatcher) take() []ui.PointerData {
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
func (b *pointerBatcher) flushAxes() {
	axes, source, hasSource := b.axes, b.source, b.hasSource
	b.axes = [2]axisFrame{}
	b.hasSource = false
	if !b.inside || (!axes[0].seen && !axes[1].seen) {
		return
	}
	if hasSource && source == wl.WlPointerAxisSourceFinger {
		dx, dy := -axes[1].value*b.scale, -axes[0].value*b.scale
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
			delta[i] = float64(a.v120) / 120 * scrollStep * b.scale
		case a.discrete != 0:
			delta[i] = float64(a.discrete) * scrollStep * b.scale
		default:
			delta[i] = a.value * b.scale
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
func (b *pointerBatcher) panData(change ui.PointerChange) ui.PointerData {
	d := b.base(change)
	d.Kind = ui.PointerDeviceKindTrackpad
	d.Device = trackpadDevice
	d.Buttons = 0
	d.Scale = 1
	return d
}

// startPan begins a pan/zoom gesture; one that is still going is ended first.
func (b *pointerBatcher) startPan(kind panKind) {
	b.endPan()
	b.pan = kind
	b.panX, b.panY = 0, 0
	b.emit(b.panData(ui.PointerChangePanZoomStart))
}

// updatePan reports the gesture's translation since the last report (physical
// pixels), its scale and its rotation in radians.
func (b *pointerBatcher) updatePan(dx, dy, scale, rotation float64) {
	b.panX += dx
	b.panY += dy
	d := b.panData(ui.PointerChangePanZoomUpdate)
	d.PanX, d.PanY = b.panX, b.panY
	d.PanDeltaX, d.PanDeltaY = dx, dy
	d.Scale, d.Rotation = scale, rotation
	b.emit(d)
}

func (b *pointerBatcher) endPan() { b.endPanOf(b.pan) }

// endPanOf ends the gesture if it is of the kind.
func (b *pointerBatcher) endPanOf(kind panKind) {
	if b.pan == panNone || b.pan != kind {
		return
	}
	b.pan = panNone
	b.emit(b.panData(ui.PointerChangePanZoomEnd))
}

// pinchBegin, pinchUpdate and pinchEnd follow zwp_pointer_gesture_pinch_v1.
func (b *pointerBatcher) pinchBegin() {
	if b.inside {
		b.startPan(panPinch)
	}
}

// pinchUpdate gives the movement of the gesture's focus point in logical pixels,
// the scale since the start and the rotation since the start in degrees.
func (b *pointerBatcher) pinchUpdate(dx, dy, scale, rotationDegrees float64) {
	if b.pan == panPinch {
		b.updatePan(dx*b.scale, dy*b.scale, scale, rotationDegrees*math.Pi/180)
	}
}

func (b *pointerBatcher) pinchEnd() { b.endPanOf(panPinch) }

// swipeBegin, swipeUpdate and swipeEnd follow zwp_pointer_gesture_swipe_v1: a
// swipe is a pan without scale or rotation.
func (b *pointerBatcher) swipeBegin() {
	if b.inside {
		b.startPan(panSwipe)
	}
}

func (b *pointerBatcher) swipeUpdate(dx, dy float64) {
	if b.pan == panSwipe {
		b.updatePan(dx*b.scale, dy*b.scale, 1, 0)
	}
}

func (b *pointerBatcher) swipeEnd() { b.endPanOf(panSwipe) }
