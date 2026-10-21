package web

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The device IDs of Flutter web: it assumes at most one mouse, one trackpad and one stylus, so
// these pointers have fixed IDs; a touch has the pointerId of the browser.
const (
	mouseDeviceID    = -1
	trackpadDeviceID = -2
	stylusDeviceID   = -4
)

// The buttons of PointerEvent.buttons that Flutter names.
const (
	primaryMouseButton   = 0x1
	secondaryMouseButton = 0x2
	middleMouseButton    = 0x4
)

// buttonsMask cuts off the buttons that do not fit in 30 bits.
const buttonsMask = 0x3fffffff

// ConvertButtonToButtons turns PointerEvent.button, the button that changed (0 left, 1 middle,
// 2 right, 3 back, 4 forward, ...) into a bit of PointerEvent.buttons (0x1 left, 0x4 middle, 0x2
// right, 0x8 back, 0x10 forward, ...).
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart convertButtonToButtons
func ConvertButtonToButtons(button int) int {
	switch button {
	case 0:
		return primaryMouseButton
	case 1:
		return middleMouseButton
	case 2:
		return secondaryMouseButton
	}
	if button < 0 {
		return 0
	}
	return 1 << button
}

// sanitized is what the button sanitizer decides for an event: the change to report and the
// buttons that are down.
type sanitized struct {
	change  ui.PointerChange
	buttons int
}

// ButtonSanitizer keeps the buttons of one pointer and turns the browser's events, which are
// not always consistent (a move that says a button went down, a right button that never comes
// up because the context menu opened), into changes the framework can follow.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer
type ButtonSanitizer struct {
	pressedButtons int
}

func htmlButtonsToFlutterButtons(buttons int) int { return buttons & buttonsMask }

func inferDownFlutterButtons(button, buttons int) int {
	if buttons == 0 && button > -1 {
		// In some cases the browser sends buttons 0 in a down event: infer it from button.
		buttons = ConvertButtonToButtons(button)
	}
	return htmlButtonsToFlutterButtons(buttons)
}

// Down sanitizes a pointerdown. A pointer that is already down gets a move with the new buttons.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeDownEvent
func (s *ButtonSanitizer) Down(button, buttons int) sanitized {
	if s.pressedButtons != 0 {
		return s.Move(buttons)
	}
	s.pressedButtons = inferDownFlutterButtons(button, buttons)
	return sanitized{ui.PointerChangeDown, s.pressedButtons}
}

// Move sanitizes a pointermove. A move that says buttons are down for a pointer that is not
// (the user pressed the right button and moved quickly) is a hover.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeMoveEvent
func (s *ButtonSanitizer) Move(buttons int) sanitized {
	pressed := htmlButtonsToFlutterButtons(buttons)
	if s.pressedButtons == 0 && pressed != 0 {
		return sanitized{ui.PointerChangeHover, s.pressedButtons}
	}
	s.pressedButtons = pressed
	change := ui.PointerChangeMove
	if s.pressedButtons == 0 {
		change = ui.PointerChangeHover
	}
	return sanitized{change, s.pressedButtons}
}

// MissingRightClickUp reports the up that never came when the right button was clicked and
// released while the context menu was shown, if the buttons say so.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeMissingRightClickUp
func (s *ButtonSanitizer) MissingRightClickUp(buttons int) (sanitized, bool) {
	pressed := htmlButtonsToFlutterButtons(buttons)
	if s.pressedButtons != 0 && pressed == 0 {
		s.pressedButtons = 0
		return sanitized{ui.PointerChangeUp, 0}, true
	}
	return sanitized{}, false
}

// Leave sanitizes a pointerleave: a hover, when no button is down (a pointer that is down is
// already handled by the moves).
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeLeaveEvent
func (s *ButtonSanitizer) Leave(buttons int) (sanitized, bool) {
	if htmlButtonsToFlutterButtons(buttons) == 0 {
		s.pressedButtons = 0
		return sanitized{ui.PointerChangeHover, 0}, true
	}
	return sanitized{}, false
}

// Up sanitizes a pointerup. It reports false for a pointer that was not down (a pointerout can
// have released it), and a move, not an up, while buttons are still down.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeUpEvent
func (s *ButtonSanitizer) Up(buttons int) (sanitized, bool) {
	if s.pressedButtons == 0 {
		return sanitized{}, false
	}
	s.pressedButtons = htmlButtonsToFlutterButtons(buttons)
	if s.pressedButtons == 0 {
		return sanitized{ui.PointerChangeUp, 0}, true
	}
	// Some buttons are still down: a move updates the position instead of an up.
	return sanitized{ui.PointerChangeMove, s.pressedButtons}, true
}

// Cancel sanitizes a pointercancel.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _ButtonSanitizer.sanitizeCancelEvent
func (s *ButtonSanitizer) Cancel() sanitized {
	s.pressedButtons = 0
	return sanitized{ui.PointerChangeCancel, 0}
}

// PointerParams are the parameters of PointerConverter.Convert: what Flutter web reads from a
// browser event, in physical pixels.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_converter.dart PointerDataConverter.convert
type PointerParams struct {
	// View is the view the pointer is over.
	View ui.ViewID
	// TimeStamp is the time of the event.
	TimeStamp time.Duration
	// Change is the change; the zero value is taken for a cancel, like Flutter's default.
	Change ui.PointerChange
	// Kind is the kind of device; the zero value is taken for a touch.
	Kind ui.PointerDeviceKind
	// SignalKind is the signal, PointerSignalKindUnspecified or None for none.
	SignalKind ui.PointerSignalKind
	// Device is the ID of the device (the browser's pointerId for a touch).
	Device int
	// PhysicalX and PhysicalY are the position.
	PhysicalX, PhysicalY float64
	// Buttons are the buttons that are down.
	Buttons int
	// Pressure and PressureMax are the pressure.
	Pressure, PressureMax float64
	// Tilt is the tilt of a stylus, in radians.
	Tilt float64
	// ScrollDeltaX and ScrollDeltaY are the scroll of a scroll signal.
	ScrollDeltaX, ScrollDeltaY float64
	// Scale is the scale of a scale signal; the zero value is 1.
	Scale float64
	// OnRespond is the answer to a scroll signal.
	OnRespond func(allowPlatformDefault bool)
}

type pointerDevice struct {
	pointer int
	x, y    float64
}

// PointerConverter turns the pointer events of the browser into the data the framework
// needs: it works out the position change and a pointer identifier for each pointer, and
// keeps the order the framework checks by synthesizing the add before a hover or a down, the
// hover before a down at another place, the move before an up at another place, and the remove
// after the end of a touch.
//
//	hover -> down -> move -> up          becomes   add(synthesized) -> hover -> down -> move -> up
//	down -> move -> up                   becomes   add(synthesized) -> down -> move -> up
//
// Deviation: Flutter keeps the state of the pointers in a static, because the framework's is
// global; here it is a field of the converter, which the platform has one of (CLAUDE.md: no
// global mutable state).
//
// Flutter: lib/web_ui/lib/src/engine/pointer_converter.dart PointerDataConverter
type PointerConverter struct {
	pointers map[int]*pointerDevice
	count    int
	// activeButtons is the buttons of the last down, hover or move: a down followed by an up
	// gets a move first, whose buttons must be right whatever the up says.
	activeButtons int
}

// NewPointerConverter makes a converter that has seen no pointer.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_converter.dart PointerDataConverter.PointerDataConverter
func NewPointerConverter() *PointerConverter {
	return &PointerConverter{pointers: map[int]*pointerDevice{}}
}

func (c *PointerConverter) ensure(device int, x, y float64) *pointerDevice {
	d, ok := c.pointers[device]
	if !ok {
		d = &pointerDevice{x: x, y: y}
		c.pointers[device] = d
	}
	return d
}

// complete makes the data of the event itself and moves the device to its position.
func (c *PointerConverter) complete(p PointerParams, onRespond func(bool)) ui.PointerData {
	d := c.pointers[p.Device]
	dx, dy := p.PhysicalX-d.x, p.PhysicalY-d.y
	d.x, d.y = p.PhysicalX, p.PhysicalY
	return c.data(p, d.pointer, dx, dy, false, onRespond)
}

// synthesize makes the data of a change the browser did not report.
func (c *PointerConverter) synthesize(p PointerParams, change ui.PointerChange, buttons int, pressure float64) ui.PointerData {
	d := c.pointers[p.Device]
	dx, dy := p.PhysicalX-d.x, p.PhysicalY-d.y
	d.x, d.y = p.PhysicalX, p.PhysicalY
	p.Change, p.Buttons, p.Pressure, p.SignalKind = change, buttons, pressure, ui.PointerSignalKindNone
	return c.data(p, d.pointer, dx, dy, true, nil)
}

func (c *PointerConverter) data(p PointerParams, pointer int, dx, dy float64, synthesized bool, onRespond func(bool)) ui.PointerData {
	scale := p.Scale
	if scale == 0 {
		scale = 1
	}
	signal := p.SignalKind
	if signal == ui.PointerSignalKindUnspecified {
		signal = ui.PointerSignalKindNone
	}
	return ui.PointerData{
		ViewID: p.View, TimeStamp: p.TimeStamp, Change: p.Change, Kind: p.Kind, SignalKind: signal,
		Device: p.Device, PointerIdentifier: pointer,
		PhysicalX: p.PhysicalX, PhysicalY: p.PhysicalY, PhysicalDeltaX: dx, PhysicalDeltaY: dy,
		Buttons: p.Buttons, Synthesized: synthesized,
		Pressure: p.Pressure, PressureMax: p.PressureMax, Tilt: p.Tilt,
		ScrollDeltaX: p.ScrollDeltaX, ScrollDeltaY: p.ScrollDeltaY, Scale: scale,
		OnRespond: onRespond,
	}
}

func (c *PointerConverter) locationChanged(p PointerParams) bool {
	d := c.pointers[p.Device]
	return d.x != p.PhysicalX || d.y != p.PhysicalY
}

// Convert appends to result the data for one browser event: the data of the event, preceded
// and followed by what is synthesized to keep the sequence valid.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_converter.dart PointerDataConverter.convert
func (c *PointerConverter) Convert(result []ui.PointerData, p PointerParams) []ui.PointerData {
	if p.Change == ui.PointerChangeUnspecified {
		p.Change = ui.PointerChangeCancel
	}
	if p.Kind == ui.PointerDeviceKindUnspecified {
		p.Kind = ui.PointerDeviceKindTouch
	}
	isDown := p.Buttons != 0
	if p.SignalKind == ui.PointerSignalKindUnspecified || p.SignalKind == ui.PointerSignalKindNone {
		return c.convertChange(result, p, isDown)
	}
	switch p.SignalKind {
	case ui.PointerSignalKindScroll, ui.PointerSignalKindScrollInertiaCancel, ui.PointerSignalKindScale:
		_, alreadyAdded := c.pointers[p.Device]
		c.ensure(p.Device, p.PhysicalX, p.PhysicalY)
		if !alreadyAdded {
			result = append(result, c.synthesize(p, ui.PointerChangeAdd, p.Buttons, p.Pressure))
		}
		if c.locationChanged(p) {
			// Hover or move the pointer to the place of the signal first, so that nobody has
			// to care about the order the platform sends hover and scroll in.
			change := ui.PointerChangeHover
			if isDown {
				change = ui.PointerChangeMove
			}
			result = append(result, c.synthesize(p, change, p.Buttons, p.Pressure))
		}
		result = append(result, c.complete(p, p.OnRespond))
	}
	// An unknown signal is ignored.
	return result
}

func (c *PointerConverter) convertChange(result []ui.PointerData, p PointerParams, isDown bool) []ui.PointerData {
	switch p.Change {
	case ui.PointerChangeAdd:
		c.ensure(p.Device, p.PhysicalX, p.PhysicalY)
		result = append(result, c.complete(p, nil))
	case ui.PointerChangeHover:
		_, alreadyAdded := c.pointers[p.Device]
		c.ensure(p.Device, p.PhysicalX, p.PhysicalY)
		if !alreadyAdded {
			result = append(result, c.synthesize(p, ui.PointerChangeAdd, p.Buttons, p.Pressure))
		}
		result = append(result, c.complete(p, nil))
		c.activeButtons = p.Buttons
	case ui.PointerChangeDown:
		_, alreadyAdded := c.pointers[p.Device]
		d := c.ensure(p.Device, p.PhysicalX, p.PhysicalY)
		c.count++
		d.pointer = c.count
		if !alreadyAdded {
			result = append(result, c.synthesize(p, ui.PointerChangeAdd, p.Buttons, p.Pressure))
		}
		if c.locationChanged(p) {
			// Hover to the place of the down first.
			result = append(result, c.synthesize(p, ui.PointerChangeHover, 0, 0))
		}
		result = append(result, c.complete(p, nil))
		c.activeButtons = p.Buttons
	case ui.PointerChangeMove:
		if _, ok := c.pointers[p.Device]; !ok {
			c.ensure(p.Device, p.PhysicalX, p.PhysicalY)
		}
		result = append(result, c.complete(p, nil))
		c.activeButtons = p.Buttons
	case ui.PointerChangeUp, ui.PointerChangeCancel:
		d, ok := c.pointers[p.Device]
		if !ok {
			return result // an up for a pointer nobody added
		}
		// A cancel can have other coordinates (the window lost the focus, or the event has
		// none): the pointer is cancelled where it was.
		if p.Change == ui.PointerChangeCancel {
			p.PhysicalX, p.PhysicalY = d.x, d.y
		}
		if c.locationChanged(p) {
			// Move to the place of the up first.
			result = append(result, c.synthesize(p, ui.PointerChangeMove, c.activeButtons, p.Pressure))
		}
		result = append(result, c.complete(p, nil))
		if p.Kind == ui.PointerDeviceKindTouch {
			// The browser gives every touch a new device ID: remove it when it ends, so that
			// the state does not grow.
			result = append(result, c.synthesize(p, ui.PointerChangeRemove, 0, 0))
			delete(c.pointers, p.Device)
		}
	case ui.PointerChangeRemove:
		d, ok := c.pointers[p.Device]
		if !ok {
			return result
		}
		p.PhysicalX, p.PhysicalY = d.x, d.y
		result = append(result, c.complete(p, nil))
		delete(c.pointers, p.Device)
	}
	return result
}

// PointerEvent is what the pointer input reads from a DOM PointerEvent (or MouseEvent), copied
// out of the event so that the conversion is plain Go.
//
// Flutter: lib/web_ui/lib/src/engine/dom.dart DomPointerEvent
type PointerEvent struct {
	// PointerType is PointerEvent.pointerType: "mouse", "pen" or "touch".
	PointerType string
	// PointerID is PointerEvent.pointerId.
	PointerID int
	// Button and Buttons are PointerEvent.button and PointerEvent.buttons.
	Button, Buttons int
	// X and Y are the position in CSS pixels relative to the view.
	X, Y float64
	// Pressure, TiltX and TiltY are the pressure and the tilt of a stylus, in degrees.
	Pressure, TiltX, TiltY float64
	// TimeStamp is the time of the event, in milliseconds.
	TimeStamp float64
	// Alt, Ctrl, Meta and Shift are the modifier flags of the event.
	Alt, Ctrl, Meta, Shift bool
	// Coalesced are the events a pointermove stands for, from getCoalescedEvents; they have
	// no pointer ID of their own. Empty when the browser has none.
	Coalesced []PointerEvent
}

// WheelEvent is what the pointer input reads from a DOM WheelEvent.
//
// Flutter: lib/web_ui/lib/src/engine/dom.dart DomWheelEvent
type WheelEvent struct {
	// X and Y are the position in CSS pixels relative to the view.
	X, Y float64
	// DeltaX, DeltaY and DeltaMode are WheelEvent.deltaX, deltaY and deltaMode (0 pixels, 1
	// lines, 2 pages).
	DeltaX, DeltaY float64
	DeltaMode      int
	// WheelDeltaX and WheelDeltaY are the legacy wheelDeltaX and wheelDeltaY, when the browser
	// has them (HasWheelDelta).
	WheelDeltaX, WheelDeltaY float64
	HasWheelDelta            bool
	// Buttons is MouseEvent.buttons.
	Buttons int
	// Ctrl is whether the control key is down: a pinch on a trackpad arrives as a wheel with it.
	Ctrl bool
	// TimeStamp is the time of the event, in milliseconds.
	TimeStamp float64
}

// WheelContext is what the conversion of a wheel event needs to know about the view.
//
// Flutter: no counterpart
type WheelContext struct {
	// DevicePixelRatio is the number of physical pixels per CSS pixel.
	DevicePixelRatio float64
	// PhysicalWidth and PhysicalHeight are the size of the view in physical pixels, which a
	// page of scroll is.
	PhysicalWidth, PhysicalHeight float64
	// LineHeight is the height of a line of scroll in physical pixels; DefaultScrollLineHeight
	// when zero.
	LineHeight float64
}

// DefaultScrollLineHeight is the height of a line of scroll when the browser's font size
// cannot be read: Flutter takes a quarter of the default font size, which is 16 pixels, when
// it can, and this fallback is 16 pixels as in Flutter.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._computeDefaultScrollLineHeight
const DefaultScrollLineHeight = 16.0

// WheelResult is the answer of the framework to a wheel event, collected while the event is
// dispatched: whether the page may scroll (a scrollable is at its boundary) or a widget took the
// scroll. Once the page may scroll, a later answer does not take it back.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _BaseAdapter._lastWheelEventAllowedDefault
type WheelResult struct {
	// AllowedDefault is whether a scrollable at its boundary let the browser scroll.
	AllowedDefault bool
	// HandledByWidget is whether a widget took the scroll.
	HandledByWidget bool
}

// PreventDefault reports whether the wheel event's default action must be prevented: unless a
// scrollable let the page scroll, the page does not. A full-page app in a frame (an iframe)
// lets the page scroll, so that the scroll goes on to the parent page, only when the scrollables
// are at their boundary and no widget took the scroll.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._handleWheelEvent
func (r *WheelResult) PreventDefault(fullPageInFrame bool) bool {
	if fullPageInFrame {
		return !r.AllowedDefault || r.HandledByWidget
	}
	return !r.AllowedDefault
}

// ParsePixels reads a CSS length in pixels, such as the computed font size "16px".
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._computeDefaultScrollLineHeight
func ParsePixels(s string) (float64, bool) {
	if !strings.Contains(s, "px") {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, "px", ""), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (r *WheelResult) respond(allowPlatformDefault bool) {
	if allowPlatformDefault {
		r.AllowedDefault = true
	} else {
		r.HandledByWidget = true
	}
}

// PointerInput converts the pointer events of one view. It keeps the buttons of each pointer,
// the state of the converter and what it needs to tell a trackpad from a wheel. It must be
// used on one goroutine.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter
type PointerInput struct {
	view ui.ViewID
	os   OperatingSystem
	// firefox is whether the browser is Firefox, whose wheel events cannot be told from a
	// trackpad's.
	firefox bool
	conv    *PointerConverter
	keys    *KeyConverter

	sanitizers map[int]*ButtonSanitizer
	// downTouches are the touches that went down and have not been released.
	downTouches map[int]bool

	lastWheel            WheelEvent
	hasLastWheel         bool
	lastWheelWasTrackpad bool
}

// PointerInputOptions are the optional parameters of NewPointerInput.
//
// Flutter: no counterpart
type PointerInputOptions struct {
	// Keys is the key converter whose record of the modifier keys is kept in step with the
	// modifier flags of the pointer events (and which tells whether control is down on macOS).
	Keys *KeyConverter
	// Firefox is whether the browser is Firefox.
	Firefox bool
}

// NewPointerInput makes the input of a view, on an operating system.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart PointerBinding.PointerBinding
func NewPointerInput(view ui.ViewID, os OperatingSystem, opts ...PointerInputOptions) *PointerInput {
	var o PointerInputOptions
	if len(opts) > 1 {
		panic("web: NewPointerInput takes at most one PointerInputOptions")
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	return &PointerInput{
		view: view, os: os, firefox: o.Firefox, conv: NewPointerConverter(), keys: o.Keys,
		sanitizers: map[int]*ButtonSanitizer{}, downTouches: map[int]bool{},
	}
}

func (in *PointerInput) sanitizer(device int) *ButtonSanitizer {
	s, ok := in.sanitizers[device]
	if !ok {
		s = &ButtonSanitizer{}
		in.sanitizers[device] = s
	}
	return s
}

func pointerKind(pointerType string) ui.PointerDeviceKind {
	switch pointerType {
	case "mouse":
		return ui.PointerDeviceKindMouse
	case "pen":
		return ui.PointerDeviceKindStylus
	case "touch":
		return ui.PointerDeviceKindTouch
	}
	return ui.PointerDeviceKindUnknown
}

// deviceOf is the ID of the device of an event: every mouse event has the mouse's, including
// the wheel, which can come before any other event and has no pointerId; the stylus has one ID.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter._getPointerId
func deviceOf(e PointerEvent) int {
	switch pointerKind(e.PointerType) {
	case ui.PointerDeviceKindMouse:
		return mouseDeviceID
	case ui.PointerDeviceKindStylus:
		return stylusDeviceID
	}
	return e.PointerID
}

// highestTilt is the larger of the two tilts of a stylus, from -90 to 90 degrees, in radians.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter._computeHighestTilt
func highestTilt(e PointerEvent) float64 {
	t := e.TiltY
	if math.Abs(e.TiltX) > math.Abs(e.TiltY) {
		t = e.TiltX
	}
	return t / 180.0 * math.Pi
}

func (in *PointerInput) checkModifiers(e PointerEvent) {
	if in.keys != nil {
		in.keys.SynthesizeModifiersIfNeeded(e.Alt, e.Ctrl, e.Meta, e.Shift, e.TimeStamp)
	}
}

// convert converts the event with the details to pointer data. pointerID is the device when
// the event is a coalesced one, which has no pointerId of its own.
func (in *PointerInput) convert(data []ui.PointerData, e PointerEvent, d sanitized, device int, dpr float64) []ui.PointerData {
	return in.conv.Convert(data, PointerParams{
		View: in.view, TimeStamp: durationOf(e.TimeStamp), Change: d.change, Kind: pointerKind(e.PointerType),
		SignalKind: ui.PointerSignalKindNone, Device: device,
		PhysicalX: e.X * dpr, PhysicalY: e.Y * dpr, Buttons: d.buttons,
		Pressure: e.Pressure, PressureMax: 1, Tilt: highestTilt(e),
	})
}

// Down converts a pointerdown, with the up of a right click that never came before it.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter.setup (pointerdown)
func (in *PointerInput) Down(e PointerEvent, dpr float64) []ui.PointerData {
	in.checkModifiers(e)
	device := deviceOf(e)
	var data []ui.PointerData
	s := in.sanitizer(device)
	if up, ok := s.MissingRightClickUp(e.Buttons); ok {
		data = in.convert(data, e, up, device, dpr)
	}
	data = in.convert(data, e, s.Down(e.Button, e.Buttons), device, dpr)
	if pointerKind(e.PointerType) == ui.PointerDeviceKindTouch {
		in.downTouches[device] = true
	}
	return data
}

// Move converts a pointermove, as many pointer data as the browser coalesced into it.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter.setup (pointermove)
func (in *PointerInput) Move(e PointerEvent, dpr float64) []ui.PointerData {
	in.checkModifiers(e)
	device := deviceOf(e)
	s := in.sanitizer(device)
	var data []ui.PointerData
	events := e.Coalesced
	if len(events) == 0 {
		// Some events do not coalesce, and some browsers cannot.
		events = []PointerEvent{e}
	}
	for _, ev := range events {
		if up, ok := s.MissingRightClickUp(ev.Buttons); ok {
			data = in.convert(data, ev, up, device, dpr)
		}
		data = in.convert(data, ev, s.Move(ev.Buttons), device, dpr)
	}
	return data
}

// Leave converts a pointerleave.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter.setup (pointerleave)
func (in *PointerInput) Leave(e PointerEvent, dpr float64) []ui.PointerData {
	device := deviceOf(e)
	if d, ok := in.sanitizer(device).Leave(e.Buttons); ok {
		return in.convert(nil, e, d, device, dpr)
	}
	return nil
}

// Up converts a pointerup; the data is empty for a pointer that was not down.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter.setup (pointerup)
func (in *PointerInput) Up(e PointerEvent, dpr float64) []ui.PointerData {
	in.checkModifiers(e)
	device := deviceOf(e)
	s, ok := in.sanitizers[device]
	if !ok {
		return nil
	}
	details, has := s.Up(e.Buttons)
	in.removeIfUnhoverable(e)
	if !has {
		return nil
	}
	return in.convert(nil, e, details, device, dpr)
}

// Cancel converts a pointercancel, which the browser sends when the pointer can no longer
// produce events (the device was deactivated).
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter.setup (pointercancel)
func (in *PointerInput) Cancel(e PointerEvent, dpr float64) []ui.PointerData {
	device := deviceOf(e)
	s, ok := in.sanitizers[device]
	if !ok {
		return nil
	}
	details := s.Cancel()
	in.removeIfUnhoverable(e)
	return in.convert(nil, e, details, device, dpr)
}

// removeIfUnhoverable forgets a touch that ended: it cannot hover.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter._removePointerIfUnhoverable
func (in *PointerInput) removeIfUnhoverable(e PointerEvent) {
	if e.PointerType == "touch" {
		device := deviceOf(e)
		delete(in.sanitizers, device)
		delete(in.downTouches, device)
	}
}

// CancelAbandonedTouches cancels the touches that the browser stopped reporting in the middle of
// a gesture: iOS WebKit sends neither pointerup nor pointercancel for a touch it promotes to a
// native gesture (dragging the caret in a text field), which would leave the framework with a
// pointer that never lifts. touchesOnSurface are the identifiers of the touches the browser
// still lists; any touch believed to be down that is not among them is cancelled. Only iOS
// does it.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _PointerAdapter._cancelAbandonedTouches
func (in *PointerInput) CancelAbandonedTouches(touchesOnSurface []int, timeStamp float64) []ui.PointerData {
	if in.os != OSIOS || len(in.downTouches) == 0 {
		return nil
	}
	onSurface := map[int]bool{}
	for _, id := range touchesOnSurface {
		onSurface[id] = true
	}
	var stale []int
	for device := range in.downTouches {
		if !onSurface[device] {
			stale = append(stale, device)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	sortInts(stale)
	var data []ui.PointerData
	for _, device := range stale {
		delete(in.sanitizers, device)
		// A cancel is reported where the pointer was last, so no position is given.
		data = in.conv.Convert(data, PointerParams{
			View: in.view, TimeStamp: durationOf(timeStamp), Change: ui.PointerChangeCancel,
			SignalKind: ui.PointerSignalKindNone, Device: device, PressureMax: 1,
		})
		delete(in.downTouches, device)
	}
	return data
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

// isAcceleratedMouseWheelDelta reports whether the delta of a wheel has the acceleration curve
// that macOS applies: its legacy wheelDelta keeps the original value, which for the events
// without the curve is three times the delta with the opposite sign.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._isAcceleratedMouseWheelDelta
func isAcceleratedMouseWheelDelta(delta, wheelDelta float64, has bool) bool {
	if !has {
		return false
	}
	// Allow 1 pixel of error for the browsers that truncate.
	return math.Abs(wheelDelta-(-3*delta)) > 1
}

// isTrackpad reports whether a wheel event comes from a trackpad, with the heuristics of
// Flutter web, which rely on non-standard details of the browsers: a delta that is a multiple
// of 120 in both directions is a mouse wheel, unless it is like the previous one or comes within
// 50 ms of a trackpad event.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._isTrackpadEvent
func (in *PointerInput) isTrackpad(e WheelEvent) bool {
	if in.firefox {
		// Firefox restricts wheelDelta: a trackpad cannot be told from a wheel.
		return false
	}
	if isAcceleratedMouseWheelDelta(e.DeltaX, e.WheelDeltaX, e.HasWheelDelta) ||
		isAcceleratedMouseWheelDelta(e.DeltaY, e.WheelDeltaY, e.HasWheelDelta) {
		return false
	}
	wx, wy := 1.0, 1.0
	if e.HasWheelDelta {
		wx, wy = e.WheelDeltaX, e.WheelDeltaY
	}
	if (math.Mod(e.DeltaX, 120) == 0 && math.Mod(e.DeltaY, 120) == 0) || (math.Mod(wx, 120) == 0 && math.Mod(wy, 120) == 0) {
		// Blink and WebKit use 120 for one turn of a mouse wheel: both deltas divisible by it
		// are probably a mouse. A trackpad event can have exactly 120 by chance, so check it is
		// not like the previous one before calling it a mouse event.
		var dxc, dyc float64
		if in.hasLastWheel {
			dxc, dyc = math.Abs(e.DeltaX-in.lastWheel.DeltaX), math.Abs(e.DeltaY-in.lastWheel.DeltaY)
		} else {
			dxc, dyc = math.Abs(e.DeltaX), math.Abs(e.DeltaY)
		}
		if !in.hasLastWheel || (dxc == 0 && dyc == 0) || !(dxc < 20 && dyc < 20) {
			// With a large delta from the previous event, a trackpad event within 50 ms before
			// it makes this one a trackpad's too (an unlucky 120 in a rapid movement).
			if in.hasLastWheel && e.TimeStamp-in.lastWheel.TimeStamp < 50 && in.lastWheelWasTrackpad {
				return true
			}
			return false
		}
	}
	return true
}

// Wheel converts a wheel event: the deltas in lines or pages become pixels (a line is a quarter
// of the font size, a page the size of the view, and on macOS Safari and Firefox report pixels
// in CSS pixels where Chrome reports device pixels); a wheel with control down is a scale (a
// pinch), anything else a scroll. The result collects the framework's answers, which come while
// the data is dispatched.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._convertWheelEventToPointerData
func (in *PointerInput) Wheel(e WheelEvent, ctx WheelContext) ([]ui.PointerData, *WheelResult) {
	const (
		deltaPixel = 0
		deltaLine  = 1
		deltaPage  = 2
	)
	kind, device := ui.PointerDeviceKindMouse, mouseDeviceID
	if in.isTrackpad(e) {
		kind, device = ui.PointerDeviceKindTrackpad, trackpadDeviceID
	}
	dx, dy := e.DeltaX, e.DeltaY
	switch e.DeltaMode {
	case deltaLine:
		lh := ctx.LineHeight
		if lh == 0 {
			lh = DefaultScrollLineHeight
		}
		dx, dy = dx*lh, dy*lh
	case deltaPage:
		dx, dy = dx*ctx.PhysicalWidth, dy*ctx.PhysicalHeight
	case deltaPixel:
		if in.os == OSMacOS {
			dx, dy = dx*ctx.DevicePixelRatio, dy*ctx.DevicePixelRatio
		}
	}
	ignoreCtrl := false
	if in.os == OSMacOS && in.keys != nil {
		ignoreCtrl = in.keys.KeyIsPressed(physicalOf("ControlLeft")) || in.keys.KeyIsPressed(physicalOf("ControlRight"))
	}
	res := &WheelResult{}
	p := PointerParams{
		View: in.view, TimeStamp: durationOf(e.TimeStamp), Change: ui.PointerChangeHover, Kind: kind,
		Device: device, PhysicalX: e.X * ctx.DevicePixelRatio, PhysicalY: e.Y * ctx.DevicePixelRatio,
		Buttons: e.Buttons, Pressure: 1, PressureMax: 1,
	}
	if e.Ctrl && !ignoreCtrl {
		p.SignalKind = ui.PointerSignalKindScale
		p.Scale = math.Exp(-dy / 200)
	} else {
		p.SignalKind = ui.PointerSignalKindScroll
		p.ScrollDeltaX, p.ScrollDeltaY = dx, dy
		p.OnRespond = res.respond
	}
	data := in.conv.Convert(nil, p)
	in.lastWheel, in.hasLastWheel = e, true
	in.lastWheelWasTrackpad = kind == ui.PointerDeviceKindTrackpad
	return data, res
}
