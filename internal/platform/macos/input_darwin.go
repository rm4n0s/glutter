//go:build darwin && !ios

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the pointer input of the view (SPEC §29.6): the methods of GlutterView that AppKit calls for
// the mouse, the trackpad and a tablet read the NSEvent into a pointerInput and hand it to the state machine
// of mouse.go, and the data that came out goes to the handler.

// The bits of NSEvent.modifierFlags that the pointer input reads.
const (
	modShift   uint64 = 1 << 17
	modControl uint64 = 1 << 18
)

// NSEventSubtypeTabletPoint is the subtype of a mouse event that a tablet made.
const eventSubtypeTabletPoint = 1

// The NSTrackingAreaOptions of the view's tracking area: enter and exit, moves, only while the window is the
// key window, also during a drag, and the whole visible rectangle (Flutter's default tracking mode).
const (
	trackingMouseEnteredAndExited  uint64 = 1 << 0
	trackingMouseMoved             uint64 = 1 << 1
	trackingActiveInKeyWindow      uint64 = 1 << 5
	trackingInVisibleRect          uint64 = 1 << 9
	trackingEnabledDuringMouseDrag uint64 = 1 << 10
	touchTypeMaskIndirect          uint64 = 1 << 1
)

// viewEvent registers a method of the view that takes an NSEvent and calls fn on the window of the view,
// under the platform's guard; the data that the call made goes to the handler afterwards.
func viewEvent(name string, fn func(w *window, ev objc.ID)) objcx.Method {
	isDown := strings.Contains(name, "ouseDown:")
	return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, ev objc.ID) {
		w := windowOf(self)
		if w == nil || w.closed {
			return
		}
		if w.blocked() { // a window with a modal window over it takes no input; a press brings the modal window forward
			if isDown {
				w.p.guard(w.redirectToModal)
			}
			return
		}
		w.p.guard(func() {
			w.p.kb.syncModifiers(w, ev)
			fn(w, ev)
			w.p.flushPointers()
		})
	}}
}

// viewMethods are the pointer methods of GlutterView.
func viewMethods() []objcx.Method {
	return []objcx.Method{
		viewEvent("mouseEntered:", func(w *window, ev objc.ID) { w.p.ptr.entered(w.inputOf(ev)) }),
		viewEvent("mouseExited:", func(w *window, ev objc.ID) { w.p.ptr.exited(w.inputOf(ev)) }),
		viewEvent("mouseMoved:", func(w *window, ev objc.ID) { w.moveEvent(ev, 0) }),
		viewEvent("mouseDown:", func(w *window, ev objc.ID) { w.downEvent(ev, buttonPrimary) }),
		viewEvent("mouseUp:", func(w *window, ev objc.ID) { w.upEvent(ev, buttonPrimary) }),
		viewEvent("mouseDragged:", func(w *window, ev objc.ID) { w.moveEvent(ev, buttonPrimary) }),
		viewEvent("rightMouseDown:", func(w *window, ev objc.ID) { w.downEvent(ev, buttonSecondary) }),
		viewEvent("rightMouseUp:", func(w *window, ev objc.ID) { w.upEvent(ev, buttonSecondary) }),
		viewEvent("rightMouseDragged:", func(w *window, ev objc.ID) { w.moveEvent(ev, buttonSecondary) }),
		viewEvent("otherMouseDown:", func(w *window, ev objc.ID) { w.downEvent(ev, otherButtonBit(ev)) }),
		viewEvent("otherMouseUp:", func(w *window, ev objc.ID) { w.upEvent(ev, otherButtonBit(ev)) }),
		viewEvent("otherMouseDragged:", func(w *window, ev objc.ID) { w.moveEvent(ev, otherButtonBit(ev)) }),
		viewEvent("scrollWheel:", func(w *window, ev objc.ID) { w.p.ptr.scrollWheel(w.scrollInputOf(ev)) }),
		viewEvent("magnifyWithEvent:", func(w *window, ev objc.ID) {
			in := w.inputOf(ev)
			in.phase = evPhase(send[uint64](ev, "phase"))
			in.magnification = send[float64](ev, "magnification")
			w.p.ptr.magnify(in)
		}),
		viewEvent("rotateWithEvent:", func(w *window, ev objc.ID) {
			in := w.inputOf(ev)
			in.phase = evPhase(send[uint64](ev, "phase"))
			in.rotation = float64(send[float32](ev, "rotation"))
			w.p.ptr.rotate(in)
		}),
		viewEvent("touchesBeganWithEvent:", func(w *window, ev objc.ID) { w.p.ptr.touchesBegan(w.inputOf(ev)) }),
		viewEvent("swipeWithEvent:", func(*window, objc.ID) {}), // scrollWheel: handles it
		viewEvent("tabletProximity:", func(w *window, ev objc.ID) { w.proximityEvent(ev) }),
		{Name: "resetCursorRects", Fn: func(self objc.ID, _ objc.SEL) {
			if w := windowOf(self); w != nil && !w.closed && w.nsCursor != 0 {
				self.Send(sel("addCursorRect:cursor:"), send[nsRect](self, "bounds"), w.nsCursor)
			}
		}},
	}
}

// inputOf reads the place, the time and the modifiers of an event into a pointerInput, in physical pixels
// of the view with the origin at its top left.
func (w *window) inputOf(ev objc.ID) pointerInput {
	loc := send[nsPoint](ev, "locationInWindow")
	return w.inputAt(ev, loc)
}

// inputAt is inputOf for an event whose place the caller knows in the window's coordinates (the events that
// the dev events make from a scroll have none of their own).
func (w *window) inputAt(ev objc.ID, locInWindow nsPoint) pointerInput {
	pt := objc.Send[nsPoint](w.view, sel("convertPoint:fromView:"), locInWindow, objc.ID(0))
	h := send[nsRect](w.view, "bounds").Size.Height
	scale := w.scale
	flags := send[uint64](ev, "modifierFlags")
	return pointerInput{
		view:  w.id,
		x:     pt.X * scale,
		y:     (h - pt.Y) * scale,
		time:  w.p.frameTime(send[float64](ev, "timestamp")),
		shift: flags&modShift != 0,
		scale: scale,
	}
}

// scrollInputOf adds what a scroll wheel event carries.
func (w *window) scrollInputOf(ev objc.ID) pointerInput {
	in := w.inputOf(ev)
	in.dx = send[float64](ev, "scrollingDeltaX")
	in.dy = send[float64](ev, "scrollingDeltaY")
	in.precise = send[bool](ev, "hasPreciseScrollingDeltas")
	in.phase = evPhase(send[uint64](ev, "phase"))
	in.momentum = evPhase(send[uint64](ev, "momentumPhase"))
	return in
}

// otherButtonBit is the button of an otherMouse event: Flutter's bit is 1 << buttonNumber, so the middle
// button (2) is 4, back (3) is 8 and forward (4) is 16.
func otherButtonBit(ev objc.ID) int {
	n := send[int64](ev, "buttonNumber")
	if n < 0 || n > 30 {
		return 0
	}
	return 1 << n
}

// isTablet says that a mouse event came from a tablet.
func isTablet(ev objc.ID) bool {
	return send[uint16](ev, "subtype") == eventSubtypeTabletPoint
}

// tabletInputOf adds the pressure and the lean of a tablet event.
func (w *window) tabletInputOf(ev objc.ID) pointerInput {
	in := w.inputOf(ev)
	tilt := send[nsPoint](ev, "tilt")
	in.detail = tabletDetail(float64(send[float32](ev, "pressure")), tilt.X, tilt.Y)
	return in
}

// stylusKind is the kind of device of the pen that is in proximity: the eraser end is its own kind.
func (w *window) stylusKind() ui.PointerDeviceKind {
	if w.p.penEraser {
		return ui.PointerDeviceKindInvertedStylus
	}
	return ui.PointerDeviceKindStylus
}

// downEvent is a button going down. A Control-click is the secondary button (Flutter's rule, which is also
// macOS's: the system makes a rightMouseDown of it for most views, and this makes it for the rest).
func (w *window) downEvent(ev objc.ID, bit int) {
	if isTablet(ev) {
		w.p.stylusButtons |= bit
		in := w.tabletInputOf(ev)
		w.p.ptr.stylusMove(in, w.stylusKind(), w.p.penDevice, w.p.stylusButtons)
		return
	}
	if bit == buttonPrimary && send[uint64](ev, "modifierFlags")&modControl != 0 {
		bit = buttonSecondary
		w.p.ctrlClick = true
	}
	w.p.ptr.buttonDown(w.inputOf(ev), bit)
}

// upEvent is a button going up.
func (w *window) upEvent(ev objc.ID, bit int) {
	if isTablet(ev) {
		w.p.stylusButtons &^= bit
		in := w.tabletInputOf(ev)
		w.p.ptr.stylusMove(in, w.stylusKind(), w.p.penDevice, w.p.stylusButtons)
		return
	}
	if bit == buttonPrimary && w.p.ctrlClick {
		bit = buttonSecondary
		w.p.ctrlClick = false
	}
	w.p.ptr.buttonUp(w.inputOf(ev), bit)
}

// moveEvent is a move or a drag. The button of a drag is the button that the event names; a move has none.
func (w *window) moveEvent(ev objc.ID, bit int) {
	if isTablet(ev) {
		in := w.tabletInputOf(ev)
		w.p.ptr.stylusMove(in, w.stylusKind(), w.p.penDevice, w.p.stylusButtons)
		return
	}
	w.p.ptr.moved(w.inputOf(ev))
}

// proximityEvent is the pen of a tablet coming into range or going out of it.
func (w *window) proximityEvent(ev objc.ID) {
	entering := send[bool](ev, "isEnteringProximity")
	in := w.inputOf(ev)
	if entering {
		w.p.penEraser = pointingDevice(send[uint64](ev, "pointingDeviceType")) == pointingEraser
		w.p.penDevice = stylusDeviceBase + int(send[uint64](ev, "deviceID")&0xff)
		w.p.stylusButtons = 0
		w.p.ptr.stylusEnter(in, w.stylusKind(), w.p.penDevice)
		return
	}
	w.p.ptr.stylusLeave(in, w.stylusKind(), w.p.penDevice)
	w.p.stylusButtons = 0
}

// flushPointers hands the data that the state machine made to the handler.
func (p *Platform) flushPointers() {
	data := p.ptr.take()
	if len(data) > 0 && p.handler != nil {
		p.handler.OnPointer(data)
	}
}

// SetCursor shows the cursor over the window (SPEC §29.6): the NSCursor of Flutter's table, or a blank
// one for none. The window keeps it in a cursor rect of the whole view, so that AppKit puts it back when the
// pointer comes in again, and it is set now.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMouseCursorPlugin.mm displayCursorObject:
func (w *window) SetCursor(c platform.MouseCursor) {
	if w.closed {
		return
	}
	cursor := w.p.cursorObject(c)
	if cursor == w.nsCursor {
		return
	}
	w.nsCursor = cursor
	objcx.WithPool(func() {
		w.ns.Send(sel("invalidateCursorRectsForView:"), w.view)
		if cursor != 0 {
			cursor.Send(sel("set"))
		}
	})
}

// cursorObject is the NSCursor for the mouse cursor, kept for the life of the platform. The blank cursor of
// MouseCursorNone is a transparent image of one pixel, as Flutter makes it.
func (p *Platform) cursorObject(c platform.MouseCursor) objc.ID {
	selector, visible := cursorSelector(c)
	if !visible {
		if p.blankCursor == 0 {
			img := class("NSImage").Send(sel("alloc")).Send(sel("initWithSize:"), nsSize{Width: 1, Height: 1})
			cur := class("NSCursor").Send(sel("alloc")).Send(sel("initWithImage:hotSpot:"), img, nsPoint{})
			objcx.Release(img)
			p.blankCursor = cur
		}
		return p.blankCursor
	}
	if cur, ok := p.cursors[selector]; ok {
		return cur
	}
	cur := objcx.Retain(class("NSCursor").Send(sel(selector)))
	if p.cursors == nil {
		p.cursors = map[string]objc.ID{}
	}
	p.cursors[selector] = cur
	return cur
}

// configureInput gives the view its tracking area and lets it take the touches of the trackpad.
func (w *window) configureInput() {
	opts := trackingMouseEnteredAndExited | trackingMouseMoved | trackingActiveInKeyWindow | trackingInVisibleRect | trackingEnabledDuringMouseDrag
	area := class("NSTrackingArea").Send(sel("alloc")).Send(sel("initWithRect:options:owner:userInfo:"), nsRect{}, opts, w.view, objc.ID(0))
	w.view.Send(sel("addTrackingArea:"), area)
	objcx.Release(area)
	w.view.Send(sel("setAllowedTouchTypes:"), touchTypeMaskIndirect)
}
