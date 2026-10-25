package windows

import (
	"math"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of touch and pen (SPEC §25.6): the values that WM_POINTER*
// messages carry (POINTER_INFO, POINTER_TOUCH_INFO, POINTER_PEN_INFO) decoded into plain numbers,
// and their conversion into the calls of the pointer tracker. The messages themselves cannot be
// made under Wine (SPEC §25.13), so a test feeds this code recorded values.

// penPressureMax is the range of the pressure of a pen (0 to 1024).
//
// Flutter: shell/platform/windows/flutter_windows_view.h kMaxPenPressure
const penPressureMax = 1024

// maxContactIDs is the number of ids the touch and pen contacts get, 0 to 127.
//
// Flutter: shell/platform/windows/flutter_window.cc kMaxTouchDeviceId
const maxContactIDs = 128

// pointerContact is one contact of a pointer message in plain values: the POINTER_INFO, and for a
// pen the POINTER_PEN_INFO or for a touch the POINTER_TOUCH_INFO, with the place already in the
// client area of the window in physical pixels.
type pointerContact struct {
	ID    uint32
	Type  uint32
	Frame uint32
	Flags uint32
	X, Y  int32

	// The pen: pressure from 0 to 1024, rotation in degrees (0 to 359), the tilt along x and y in
	// degrees (-90 to 90), and the flags (PEN_FLAG_*).
	PenFlags uint32
	Pressure uint32
	Rotation uint32
	TiltX    int32
	TiltY    int32

	// The touch: the contact rectangle in physical pixels, the orientation of the ellipse in degrees
	// (0 to 359) and the pressure from 0 to 1024 (0 when the screen has none).
	ContactWidth, ContactHeight int32
	Orientation                 uint32
	TouchPressure               uint32
}

// contactOfInfo decodes POINTER_INFO and the information of its kind (pen or touch, nil for the
// other) into a contact. The place is the screen place that clientOf turns into the client
// area's; a nil clientOf takes it as it is.
func contactOfInfo(info win32.PointerInfo, pen *win32.PointerPenInfo, touch *win32.PointerTouchInfo, clientOf func(win32.Point) win32.Point) pointerContact {
	pt := info.PixelLocation
	if clientOf != nil {
		pt = clientOf(pt)
	}
	c := pointerContact{ID: info.PointerID, Type: info.PointerType, Frame: info.FrameID, Flags: info.PointerFlags, X: pt.X, Y: pt.Y}
	if pen != nil {
		c.PenFlags, c.Pressure, c.Rotation, c.TiltX, c.TiltY = pen.PenFlags, pen.Pressure, pen.Rotation, pen.TiltX, pen.TiltY
	}
	if touch != nil {
		c.ContactWidth = touch.Contact.Right - touch.Contact.Left
		c.ContactHeight = touch.Contact.Bottom - touch.Contact.Top
		c.Orientation, c.TouchPressure = touch.Orientation, touch.Pressure
	}
	return c
}

// isTouchOrPen says whether the contact is one that the pointer messages are used for: the mouse
// and the touchpad go to DefWindowProc, which makes the classic messages (SPEC §25.6).
func (c pointerContact) isTouchOrPen() bool {
	return c.Type == win32.PTTouch || c.Type == win32.PTPen
}

// penButtons turns the flags of a pen into Flutter's button bits: nothing while it does not touch
// (a hover has no buttons), else the tip as the primary button, the barrel button as the secondary
// one and the eraser button as the middle one.
//
// Flutter: shell/platform/windows/flutter_window.cc ConvertWinStylusFlagsToFlutterButtons
func penButtons(penFlags, pointerFlags uint32) int {
	if pointerFlags&win32.PointerFlagInContact == 0 {
		return 0
	}
	b := buttonPrimary
	if penFlags&win32.PenFlagBarrel != 0 {
		b |= buttonSecondary
	}
	if penFlags&win32.PenFlagEraser != 0 {
		b |= buttonMiddle
	}
	return b
}

// touchButtons is the primary button while the finger touches the screen, and nothing else.
//
// Flutter: shell/platform/windows/flutter_window.cc ConvertWinPointerFlagsToFlutterButtons
func touchButtons(pointerFlags uint32) int {
	if pointerFlags&win32.PointerFlagInContact == 0 {
		return 0
	}
	return buttonPrimary
}

// tiltAngles turns the tilt of a pen along the x and y axes in degrees from the normal of the
// surface into the angle from the normal in radians (0 to pi/2), Flutter's PointerData.tilt.
func tiltAngles(xDeg, yDeg float64) float64 {
	tx, ty := math.Tan(xDeg*math.Pi/180), math.Tan(yDeg*math.Pi/180)
	return math.Min(math.Atan(math.Hypot(tx, ty)), math.Pi/2)
}

// degrees turns degrees into radians.
func degrees(d float64) float64 { return d * math.Pi / 180 }

// sampleOf makes the report of a contact: its kind, device, place, buttons and details.
//
// Deviation: Flutter hands the pressure of the pen and its rotation on as they are, in a
// FlutterPointerEvent with a pressure range of 0 to 1024. The data here has the pressure from 0 to
// 1 (PressureMax 1, as the other platforms have it), the tilt from tiltX and tiltY, and the
// rotation as the orientation in radians. A touch reports the pressure that the screen gives, or 1
// while a finger is down when it gives none, and the ellipse of the contact.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_POINTER*)
func sampleOf(c pointerContact, device int, kind ui.PointerDeviceKind) pointerSample {
	s := pointerSample{kind: kind, device: device, x: float64(c.X), y: float64(c.Y)}
	switch c.Type {
	case win32.PTPen:
		s.buttons = penButtons(c.PenFlags, c.Flags)
		s.detail = pointerDetail{
			pressureMax: 1,
			tilt:        tiltAngles(float64(c.TiltX), float64(c.TiltY)),
			orientation: degrees(float64(c.Rotation)),
		}
		if s.buttons != 0 { // a pen in the air presses nothing, whatever the driver says
			s.detail.pressure = math.Min(float64(c.Pressure)/penPressureMax, 1)
		}
	case win32.PTTouch:
		s.buttons = touchButtons(c.Flags)
		d := pointerDetail{pressureMax: 1, orientation: degrees(float64(c.Orientation))}
		if s.buttons != 0 {
			d.pressure = 1
			if c.TouchPressure != 0 {
				d.pressure = math.Min(float64(c.TouchPressure)/penPressureMax, 1)
			}
		}
		major, minor := float64(c.ContactWidth), float64(c.ContactHeight)
		if minor > major {
			major, minor = minor, major
		}
		d.radiusMajor, d.radiusMinor = major/2, minor/2
		s.detail = d
	}
	return s
}

// kindOf is the kind of device of a contact: a touch, a stylus, or the eraser end of one.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_POINTER*, the device kind)
func kindOf(c pointerContact) ui.PointerDeviceKind {
	switch c.Type {
	case win32.PTTouch:
		return ui.PointerDeviceKindTouch
	case win32.PTPen:
		if c.PenFlags&win32.PenFlagInverted != 0 {
			return ui.PointerDeviceKindInvertedStylus
		}
		return ui.PointerDeviceKindStylus
	}
	return ui.PointerDeviceKindMouse
}

// messageOfFlags is the pointer message that the flags of a contact stand for, for the contacts of
// a frame that the message of another contact brought.
func messageOfFlags(flags uint32) uint32 {
	switch {
	case flags&win32.PointerFlagDown != 0:
		return win32.WMPointerDown
	case flags&win32.PointerFlagUp != 0:
		return win32.WMPointerUp
	}
	return win32.WMPointerUpdate
}

// contactRecord is what the translator remembers about a contact from its first message to its end.
type contactRecord struct {
	kind   ui.PointerDeviceKind
	device int
	// x and y are its last place, for the messages that have none.
	x, y float64
}

// pendingMessage is a message that the translator did its work for already: the contact came
// with another contact's message in the frame.
type pendingMessage struct {
	frame uint32
	msg   uint32
}

// pointerTranslator turns the pointer messages of touch and pen into calls of the tracker (SPEC
// §25.6). The contacts of one frame are delivered together, when the first of their messages
// arrives, and become one batch; the other messages of the frame do nothing. Each contact has an id
// from the sequential generator for as long as it lasts.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_POINTERDOWN, WM_POINTERUPDATE, WM_POINTERUP, WM_POINTERLEAVE)
type pointerTranslator struct {
	tracker *pointerTracker
	view    ui.ViewID
	ids     *sequentialIDGenerator
	active  map[uint32]*contactRecord
	pending map[uint32]pendingMessage
}

// newPointerTranslator makes the translator of touch and pen of a view.
func newPointerTranslator(tracker *pointerTracker, view ui.ViewID) *pointerTranslator {
	return &pointerTranslator{
		tracker: tracker, view: view, ids: newSequentialIDGenerator(0, maxContactIDs),
		active: map[uint32]*contactRecord{}, pending: map[uint32]pendingMessage{},
	}
}

// handle handles the pointer message msg of the pointer own, with the contacts of its frame (own
// among them; just own when the frame cannot be read). It reports whether the message was for a
// touch or a pen, which the window procedure then does not pass to DefWindowProc.
func (tr *pointerTranslator) handle(msg uint32, own pointerContact, frame []pointerContact) bool {
	if !own.isTouchOrPen() {
		return false
	}
	if p, ok := tr.pending[own.ID]; ok {
		delete(tr.pending, own.ID)
		if p.frame == own.Frame && p.msg == msg {
			return true // delivered with the contact that came first
		}
	}
	tr.contact(msg, own)
	for _, c := range frame {
		if c.ID == own.ID || !c.isTouchOrPen() {
			continue
		}
		m := messageOfFlags(c.Flags)
		tr.pending[c.ID] = pendingMessage{frame: c.Frame, msg: m}
		tr.contact(m, c)
	}
	return true
}

// contact handles one contact with the message that made it.
func (tr *pointerTranslator) contact(msg uint32, c pointerContact) {
	rec := tr.active[c.ID]
	if rec == nil {
		// The kind and the device of a contact stay what they were at its first message.
		kind := kindOf(c)
		base := touchDeviceBase
		if c.Type == win32.PTPen {
			base = stylusDeviceBase
		}
		rec = &contactRecord{kind: kind, device: base + int(tr.ids.generatedID(c.ID))}
		tr.active[c.ID] = rec
	}
	if msg != win32.WMPointerLeave && msg != win32.WMPointerCaptureChanged {
		rec.x, rec.y = float64(c.X), float64(c.Y)
	}
	s := sampleOf(c, rec.device, rec.kind)
	switch msg {
	case win32.WMPointerEnter:
		tr.tracker.enter(tr.view, s)
	case win32.WMPointerDown:
		tr.tracker.down(tr.view, s)
	case win32.WMPointerUpdate:
		tr.tracker.move(tr.view, s)
	case win32.WMPointerUp:
		s.buttons = 0 // the up ends every button: the contact is over (the pen stays in range)
		tr.tracker.up(tr.view, s)
	case win32.WMPointerLeave:
		tr.tracker.leave(tr.view, rec.kind, rec.device, rec.x, rec.y)
		tr.forget(c.ID)
	case win32.WMPointerCaptureChanged:
		tr.tracker.cancel(tr.view, rec.kind, rec.device)
		tr.forget(c.ID)
	}
}

// forget ends a contact: its id can be given again.
func (tr *pointerTranslator) forget(id uint32) {
	delete(tr.active, id)
	delete(tr.pending, id)
	tr.ids.releaseNumber(id)
}

// cancelAll cancels every contact (the window goes away, or loses everything).
func (tr *pointerTranslator) cancelAll() {
	tr.tracker.cancelView(tr.view)
	for id := range tr.active {
		tr.forget(id)
	}
}

// handleGone handles the end of a contact whose information the system no longer gives (the leave
// or the capture change of a pointer that is gone): it reports whether the contact was known.
func (tr *pointerTranslator) handleGone(msg uint32, id uint32) bool {
	rec, ok := tr.active[id]
	if !ok {
		return false
	}
	switch msg {
	case win32.WMPointerLeave:
		tr.tracker.leave(tr.view, rec.kind, rec.device, rec.x, rec.y)
	case win32.WMPointerCaptureChanged:
		tr.tracker.cancel(tr.view, rec.kind, rec.device)
	}
	tr.forget(id)
	return true
}
