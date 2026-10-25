//go:build windows

package windows

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the glue of the pointer input of a window (SPEC §25.6): it reads the messages of the
// mouse, the wheel, touch and pen, and gives them to the plain-Go translators (mouse.go, wheel.go,
// pointer.go), then sends what they made to the handler as one batch; it applies the cursor.

// initInput makes the translators of the window and turns off the visual feedback of touch and pen
// (the contact ring and the barrel visualization): the app shows its own.
func (w *window) initInput() {
	w.mouse = newMouseTranslator(w.p.pointers, w.id)
	w.touch = newPointerTranslator(w.p.pointers, w.id)
	win32.SetWindowFeedbackSetting(w.hwnd, win32.FeedbackTouchContactVisualization, false)
	win32.SetWindowFeedbackSetting(w.hwnd, win32.FeedbackPenBarrelVisualization, false)
	w.initText()
}

// flushPointers sends the data the translators made as one batch.
func (w *window) flushPointers() {
	data := w.p.pointers.take()
	if len(data) == 0 || w.p.handler == nil {
		return
	}
	w.p.handler.OnPointer(data)
}

// applyMouseEffects makes the calls to Windows that a mouse message asked for. The scripted events
// do not ask for WM_MOUSELEAVE: the pointer is not really over the window, and Windows would
// send the message at once.
func (w *window) applyMouseEffects(fx mouseEffects) {
	if fx.trackLeave && !w.synthetic {
		win32.TrackMouseEvent(w.hwnd, win32.TMELeave)
	}
	if fx.setCapture {
		win32.SetCapture(w.hwnd)
	}
	if fx.releaseCapture && win32.GetCapture() == w.hwnd {
		win32.ReleaseCapture()
	}
}

// onMouse handles the classic mouse messages. It reports whether the message was handled and what
// the window procedure answers.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_MOUSEMOVE, WM_*BUTTON*, WM_MOUSELEAVE)
func (w *window) onMouse(msg uint32, wparam, lparam uintptr) (uintptr, bool) {
	if w.p.handler == nil {
		return 0, false
	}
	if !w.synthetic && isTouchOrPenMessage(win32.GetMessageExtraInfo()) {
		return 0, true // touch and pen come as pointer messages
	}
	var fx mouseEffects
	result := uintptr(0)
	switch msg {
	case win32.WMMouseMove:
		if !w.synthetic {
			// The message says which of Shift and Control are down: a window that did not get their
			// key messages learns it here (Flutter's OnPointerMove does the same).
			w.p.keys.syncModifiers(wparam&win32.MKShift != 0, wparam&win32.MKControl != 0)
		}
		fx = w.mouse.move(wparam, lparam)
	case win32.WMLButtonDown, win32.WMRButtonDown, win32.WMMButtonDown, win32.WMXButtonDown:
		fx = w.mouse.buttonDown(msg, wparam, lparam)
		if msg == win32.WMXButtonDown {
			result = 1 // an application that processes WM_XBUTTONDOWN returns TRUE
		}
	case win32.WMLButtonUp, win32.WMRButtonUp, win32.WMMButtonUp, win32.WMXButtonUp:
		fx = w.mouse.buttonUp(msg, wparam, lparam)
		if msg == win32.WMXButtonUp {
			result = 1
		}
	case win32.WMMouseLeave:
		fx = w.mouse.leave()
	default:
		return 0, false
	}
	w.applyMouseEffects(fx)
	w.flushPointers()
	return result, true
}

// onCaptureChanged handles WM_CAPTURECHANGED (lParam is the window that has the capture now).
//
// Flutter: no counterpart
func (w *window) onCaptureChanged(lparam uintptr) {
	w.mouse.captureChanged(lparam == w.hwnd)
	w.flushPointers()
}

// onCancelMode handles WM_CANCELMODE (a modal dialog or menu opened): the capture is given up, which
// WM_CAPTURECHANGED then reports.
//
// Flutter: no counterpart
func (w *window) onCancelMode() {
	if win32.GetCapture() == w.hwnd {
		win32.ReleaseCapture()
	}
}

// wheelLines is the number of lines a notch of the wheel scrolls, from the system's setting.
func wheelLines() uint32 {
	if lines, ok := win32.SystemParametersInfoUint(win32.SPIGetWheelScrollLines); ok {
		return lines
	}
	return defaultWheelLines
}

// onWheel handles WM_MOUSEWHEEL and WM_MOUSEHWHEEL: the place is in the lParam, in screen
// coordinates, and the scroll is a signal of the mouse.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnScroll
func (w *window) onWheel(msg uint32, wparam, lparam uintptr) {
	if w.p.handler == nil {
		return
	}
	pt, ok := win32.ScreenToClient(w.hwnd, win32.Point{X: signed16(lparam), Y: signed16(lparam >> 16)})
	if !ok {
		return
	}
	dx, dy := wheelScroll(msg == win32.WMMouseHWheel, wparam, wheelLines(), float64(w.clientW), float64(w.clientH))
	w.p.pointers.scroll(w.id, ui.PointerDeviceKindMouse, mouseDevice, float64(pt.X), float64(pt.Y), dx, dy)
	w.flushPointers()
}

// onPointer handles the pointer messages of touch and pen: the contacts of the frame are read, and
// become one batch. It reports whether the message was handled, which keeps it from DefWindowProc
// (which would make mouse messages and gestures of it). A mouse or touchpad pointer is not handled:
// DefWindowProc turns it into the classic messages.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_POINTERDOWN, WM_POINTERUPDATE, WM_POINTERUP, WM_POINTERLEAVE)
func (w *window) onPointer(msg uint32, wparam uintptr) bool {
	if w.p.handler == nil {
		return false
	}
	id := uint32(wparam & 0xFFFF) // GET_POINTERID_WPARAM
	info, ok := win32.GetPointerInfo(id)
	if !ok {
		// The end of a contact can come when the system has forgotten it.
		if msg == win32.WMPointerLeave || msg == win32.WMPointerCaptureChanged {
			handled := w.touch.handleGone(msg, id)
			w.flushPointers()
			return handled
		}
		return false
	}
	if info.PointerType != win32.PTTouch && info.PointerType != win32.PTPen {
		return false
	}
	clientOf := func(pt win32.Point) win32.Point {
		if c, ok := win32.ScreenToClient(w.hwnd, pt); ok {
			return c
		}
		return pt
	}
	read := func(info win32.PointerInfo) pointerContact {
		switch info.PointerType {
		case win32.PTPen:
			if pen, ok := win32.GetPointerPenInfo(info.PointerID); ok {
				return contactOfInfo(info, &pen, nil, clientOf)
			}
		case win32.PTTouch:
			if touch, ok := win32.GetPointerTouchInfo(info.PointerID); ok {
				return contactOfInfo(info, nil, &touch, clientOf)
			}
		}
		return contactOfInfo(info, nil, nil, clientOf)
	}
	own := read(info)
	var frame []pointerContact
	if infos, ok := win32.GetPointerFrameInfo(id); ok {
		for _, i := range infos {
			if i.PointerID != id {
				frame = append(frame, read(i))
			}
		}
	}
	handled := w.touch.handle(msg, own, frame)
	w.flushPointers()
	return handled
}

// SetCursor sets the cursor over the window. It is shown at once when the pointer is in the
// client area, and again by every WM_SETCURSOR.
//
// Flutter: shell/platform/windows/flutter_windows_engine.cc FlutterWindowsEngine::UpdateFlutterCursor
func (w *window) SetCursor(cursor platform.MouseCursor) {
	if w.closed {
		return
	}
	w.cursor = cursor
	if w.pointerOverClient() {
		w.showCursor()
	}
}

// showCursor makes the window's cursor the pointer's.
func (w *window) showCursor() {
	idc, visible := cursorResource(w.cursor)
	if !visible {
		win32.SetCursor(0)
		return
	}
	win32.SetCursor(win32.LoadCursor(0, idc))
}

// pointerOverClient reports whether the mouse pointer is over the client area of this window.
func (w *window) pointerOverClient() bool {
	pt, ok := win32.GetCursorPos()
	if !ok {
		return false
	}
	c, ok := win32.ScreenToClient(w.hwnd, pt)
	if !ok || c.X < 0 || c.Y < 0 || c.X >= w.clientW || c.Y >= w.clientH {
		return false
	}
	return win32.WindowFromPoint(pt) == w.hwnd
}

// onSetCursor handles WM_SETCURSOR: over the client area the window's cursor is set and the message
// is answered TRUE, so that DefWindowProc does not set the cursor of the class; any other part
// (the frame, the caption) is DefWindowProc's.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_SETCURSOR)
func (w *window) onSetCursor(lparam uintptr) bool {
	if uint32(lparam&0xFFFF) != win32.HTClient {
		return false
	}
	w.showCursor()
	return true
}

// devWindow is the window the scripted events go to: the first one that is shown.
func (p *Platform) devWindow() *window {
	for _, w := range p.sortedWindows() {
		if w.shown && !w.closed && !w.hidden {
			return w
		}
	}
	return nil
}

// sendMouse sends a window message of a mouse as the system would, to this window, with the place
// in logical pixels of the client area. The pointer is not over the window in fact, so
// WM_MOUSELEAVE is not asked for.
func (w *window) sendMouse(msg uint32, wparam uintptr, x, y float64) {
	scale := w.scale()
	w.synthetic = true
	defer func() { w.synthetic = false }()
	win32.SendMessage(w.hwnd, msg, wparam, packPoint(physical(x, scale), physical(y, scale)))
}

// runDevMouse does one scripted mouse step: move, tap (a move, the left button down and up) or
// wheel (a vertical and a horizontal message), on the first window.
//
// Flutter: no counterpart
func (p *Platform) runDevMouse(ev DevEvent) {
	w := p.devWindow()
	if w == nil {
		p.log.Warn("a scripted event found no window", "step", ev.Kind)
		return
	}
	switch ev.Kind {
	case "move":
		w.sendMouse(win32.WMMouseMove, 0, ev.X, ev.Y)
	case "tap":
		w.sendMouse(win32.WMMouseMove, 0, ev.X, ev.Y)
		w.sendMouse(win32.WMLButtonDown, win32.MKLButton, ev.X, ev.Y)
		w.sendMouse(win32.WMLButtonUp, 0, ev.X, ev.Y)
	case "wheel":
		w.sendWheel(ev)
	}
}

// sendWheel sends the wheel messages of a scripted wheel step. The scroll is in logical pixels, and
// the message gets the delta that scrolls that much: the wheel is positive away from the user,
// which scrolls up, so a scroll down is a negative delta.
func (w *window) sendWheel(ev DevEvent) {
	scale := w.scale()
	pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: physical(ev.X, scale), Y: physical(ev.Y, scale)})
	if !ok {
		return
	}
	lparam := packPoint(pt.X, pt.Y)
	lines := wheelLines()
	w.synthetic = true
	defer func() { w.synthetic = false }()
	if ev.DY != 0 {
		win32.SendMessage(w.hwnd, win32.WMMouseWheel, wheelParam(-ev.DY*scale, lines, float64(w.clientH)), lparam)
	}
	if ev.DX != 0 {
		win32.SendMessage(w.hwnd, win32.WMMouseHWheel, wheelParam(ev.DX*scale, lines, float64(w.clientW)), lparam)
	}
}

// pointerClock is the clock of the pointer data: the time since the platform started, as the
// frames have it.
func (p *Platform) pointerClock() time.Duration { return time.Since(p.start) }
