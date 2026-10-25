package windows

import (
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the mouse (SPEC §25.6): the classic messages (WM_MOUSEMOVE,
// WM_{L,R,M,X}BUTTON{DOWN,UP}, WM_MOUSELEAVE, WM_CAPTURECHANGED) turned into the calls of the
// pointer tracker, and the answers the window procedure has to act on (take the capture, give it
// back, ask for WM_MOUSELEAVE). It takes the numbers of the messages and calls nothing of Windows,
// so a test runs it on any host.

// mouseEffects are the calls to Windows that a mouse message asks for.
type mouseEffects struct {
	// trackLeave asks for one WM_MOUSELEAVE (TrackMouseEvent with TME_LEAVE).
	trackLeave bool
	// setCapture takes the mouse capture, so that a drag goes on outside the window.
	setCapture bool
	// releaseCapture gives the capture back.
	releaseCapture bool
}

// mouseTranslator is the state of the classic mouse messages of one window.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_MOUSEMOVE, WM_MOUSELEAVE, WM_*BUTTON*)
type mouseTranslator struct {
	tracker *pointerTracker
	view    ui.ViewID
	// tracking says WM_MOUSELEAVE was asked for and has not come yet: the pointer is in the window.
	tracking bool
	// captured says this window took the capture because a button went down.
	captured bool
	// leaveDeferred says a WM_MOUSELEAVE came while a button was held with the capture: Windows
	// delays it until the capture is gone, but Wine does not. It is acted on at the last button up.
	leaveDeferred bool
	// x and y are the last place of the pointer, for WM_MOUSELEAVE, which has none.
	x, y float64
}

// newMouseTranslator makes the translator of the mouse of a view.
func newMouseTranslator(tracker *pointerTracker, view ui.ViewID) *mouseTranslator {
	return &mouseTranslator{tracker: tracker, view: view}
}

// signed16 reads a coordinate of an lParam as a signed 16-bit value (a captured pointer outside the
// window has negative ones).
func signed16(v uintptr) int32 { return int32(int16(v & 0xFFFF)) }

// packPoint makes the lParam of a mouse message from a place in physical pixels.
func packPoint(x, y int32) uintptr {
	return uintptr(uint16(int16(x))) | uintptr(uint16(int16(y)))<<16
}

// clientPoint reads the client position that the lParam of a mouse message carries.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (GET_X_LPARAM, GET_Y_LPARAM)
func clientPoint(lparam uintptr) (x, y float64) {
	return float64(signed16(lparam)), float64(signed16(lparam >> 16))
}

// mouseButtonsOfState turns the MK_* flags of the wParam of WM_MOUSEMOVE into Flutter's bits.
//
// Flutter: shell/platform/windows/flutter_window.cc ConvertWinMouseStateToFlutterButtons
func mouseButtonsOfState(wparam uintptr) int {
	var b int
	if wparam&win32.MKLButton != 0 {
		b |= buttonPrimary
	}
	if wparam&win32.MKRButton != 0 {
		b |= buttonSecondary
	}
	if wparam&win32.MKMButton != 0 {
		b |= buttonMiddle
	}
	if wparam&win32.MKXButton1 != 0 {
		b |= buttonBack
	}
	if wparam&win32.MKXButton2 != 0 {
		b |= buttonForward
	}
	return b
}

// mouseButtonOfMessage is the button that a WM_*BUTTON* message is about, as Flutter's bit; 0 for
// a message that is none, or an X button that is neither XBUTTON1 (back, 8) nor XBUTTON2 (forward,
// 16).
//
// Flutter: shell/platform/windows/flutter_window.cc ConvertWinButtonToFlutterButton
func mouseButtonOfMessage(msg uint32, wparam uintptr) int {
	switch msg {
	case win32.WMLButtonDown, win32.WMLButtonUp:
		return buttonPrimary
	case win32.WMRButtonDown, win32.WMRButtonUp:
		return buttonSecondary
	case win32.WMMButtonDown, win32.WMMButtonUp:
		return buttonMiddle
	case win32.WMXButtonDown, win32.WMXButtonUp:
		switch (wparam >> 16) & 0xFFFF {
		case win32.XButton1:
			return buttonBack
		case win32.XButton2:
			return buttonForward
		}
	}
	return 0
}

// isTouchOrPenMessage says whether the mouse message in the queue was made by Windows from touch or
// pen input, by the signature in the extra information of the message. A pointer message handled
// without DefWindowProc makes none; the check is for the ones that arrive anyway.
//
// Flutter: shell/platform/windows/flutter_window.cc GetFlutterPointerDeviceKind
func isTouchOrPenMessage(extraInfo uintptr) bool {
	const touchOrPenSignature, signatureMask = 0xFF515700, 0xFFFFFF00
	return extraInfo&signatureMask == touchOrPenSignature
}

func (m *mouseTranslator) sample(x, y float64, buttons int) pointerSample {
	m.x, m.y = x, y
	return pointerSample{kind: ui.PointerDeviceKindMouse, device: mouseDevice, x: x, y: y, buttons: buttons}
}

// leaveRequest asks for WM_MOUSELEAVE the first time the pointer is seen in the window, and says
// whether the call has to be made.
func (m *mouseTranslator) leaveRequest() bool {
	if m.tracking {
		return false
	}
	m.tracking = true
	return true
}

// move handles WM_MOUSEMOVE: the pointer is added the first time, and then it hovers, or moves
// with the buttons the message says are down. A move that says no button is down ends a contact
// the platform had not heard the end of (the up was lost).
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_MOUSEMOVE)
func (m *mouseTranslator) move(wparam, lparam uintptr) mouseEffects {
	var fx mouseEffects
	fx.trackLeave = m.leaveRequest()
	x, y := clientPoint(lparam)
	buttons := mouseButtonsOfState(wparam)
	m.tracker.move(m.view, m.sample(x, y, buttons))
	if buttons == 0 && m.captured {
		m.captured, fx.releaseCapture = false, true
	}
	return fx
}

// buttonDown handles WM_{L,R,M,X}BUTTONDOWN: the first button down takes the capture, so that a drag
// goes on outside the window. A button that Flutter has no bit for does nothing.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_LBUTTONDOWN)
func (m *mouseTranslator) buttonDown(msg uint32, wparam, lparam uintptr) mouseEffects {
	var fx mouseEffects
	bit := mouseButtonOfMessage(msg, wparam)
	if bit == 0 {
		return fx
	}
	fx.trackLeave = m.leaveRequest()
	x, y := clientPoint(lparam)
	if !m.captured {
		m.captured, fx.setCapture = true, true
	}
	m.tracker.down(m.view, m.sample(x, y, bit))
	return fx
}

// buttonUp handles WM_{L,R,M,X}BUTTONUP: the last button up gives the capture back.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_LBUTTONUP)
func (m *mouseTranslator) buttonUp(msg uint32, wparam, lparam uintptr) mouseEffects {
	var fx mouseEffects
	bit := mouseButtonOfMessage(msg, wparam)
	if bit == 0 {
		return fx
	}
	x, y := clientPoint(lparam)
	m.tracker.up(m.view, m.sample(x, y, bit))
	if !m.tracker.mouseDown() {
		if m.captured {
			m.captured, fx.releaseCapture = false, true
		}
		m.runDeferredLeave()
	}
	return fx
}

// runDeferredLeave does the WM_MOUSELEAVE that waited for the buttons to be up.
func (m *mouseTranslator) runDeferredLeave() {
	if !m.leaveDeferred {
		return
	}
	m.leaveDeferred, m.tracking = false, false
	m.tracker.leave(m.view, ui.PointerDeviceKindMouse, mouseDevice, m.x, m.y)
}

// leave handles WM_MOUSELEAVE: the pointer is removed, after the buttons that are down are lifted.
// Once the message came, the request is spent, and the next move asks again. While a button is
// held with the capture, the drag goes on outside the window and the leave waits for the last
// button up (Windows sends it then; Wine sends it at once, and the up would be lost if the
// capture were given back).
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_MOUSELEAVE)
func (m *mouseTranslator) leave() mouseEffects {
	var fx mouseEffects
	if m.captured && m.tracker.mouseDown() {
		m.leaveDeferred = true
		return fx
	}
	m.tracking = false
	m.tracker.leave(m.view, ui.PointerDeviceKindMouse, mouseDevice, m.x, m.y)
	if m.captured {
		m.captured, fx.releaseCapture = false, true
	}
	return fx
}

// captureChanged handles WM_CAPTURECHANGED: when another window (or nobody) has the capture now,
// the buttons that were down are released with a synthesized up, the pointer stays.
//
// Flutter: no counterpart
func (m *mouseTranslator) captureChanged(gainedByThisWindow bool) {
	if gainedByThisWindow {
		return
	}
	m.captured = false
	m.tracker.releaseButtons(ui.PointerDeviceKindMouse, mouseDevice)
	m.runDeferredLeave()
}
