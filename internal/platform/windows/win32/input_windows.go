//go:build windows

package win32

import (
	"unsafe"
)

// The bindings of user32.dll for the pointer input of step 28.3: the capture of the mouse, the
// request for WM_MOUSELEAVE, the pointer messages of touch and pen, the cursors and the lines of
// the wheel (SPEC §25.6). A function whose failure the caller can only skip returns a plain answer;
// one whose library or function is missing (an old Windows, Wine without the function) answers
// false or 0 as well.

// SetCapture directs the mouse input of the thread to the window until ReleaseCapture, and returns
// the window that had the capture, or 0.
//
// Flutter: no counterpart
func SetCapture(hwnd uintptr) uintptr {
	r, _, err := user32.call("SetCapture", hwnd)
	if err != nil {
		return 0
	}
	return r
}

// GetCapture is the window of the thread that has the mouse capture, or 0.
//
// Flutter: no counterpart
func GetCapture() uintptr {
	r, _, err := user32.call("GetCapture")
	if err != nil {
		return 0
	}
	return r
}

// TrackMouseEvent asks for one WM_MOUSELEAVE when the pointer leaves the client area of the
// window (TME_LEAVE), and reports whether the request was made.
//
// Flutter: no counterpart
func TrackMouseEvent(hwnd uintptr, flags uint32) bool {
	tme := TrackMouseEventData{Size: uint32(unsafe.Sizeof(TrackMouseEventData{})), Flags: flags, Track: hwnd}
	return boolCall(user32, "TrackMouseEvent", uintptr(unsafe.Pointer(&tme)))
}

// ScreenToClient turns a point of the screen into a point of the client area of a window.
//
// Flutter: no counterpart
func ScreenToClient(hwnd uintptr, pt Point) (Point, bool) {
	r, _, err := user32.call("ScreenToClient", hwnd, uintptr(unsafe.Pointer(&pt)))
	return pt, err == nil && r != 0
}

// SetCursor sets the cursor of the pointer and returns the one it had; 0 hides it.
//
// Flutter: no counterpart
func SetCursor(cursor uintptr) uintptr {
	r, _, err := user32.call("SetCursor", cursor)
	if err != nil {
		return 0
	}
	return r
}

// GetCursorInfo is the state of the pointer's cursor: its handle, whether it is shown, and its
// place on the screen.
//
// Flutter: no counterpart
func GetCursorInfo() (CursorInfo, bool) {
	ci := CursorInfo{Size: uint32(unsafe.Sizeof(CursorInfo{}))}
	r, _, err := user32.call("GetCursorInfo", uintptr(unsafe.Pointer(&ci)))
	return ci, err == nil && r != 0
}

// SetCursorPos moves the pointer to a place of the screen.
//
// Flutter: no counterpart
func SetCursorPos(x, y int32) bool {
	return boolCall(user32, "SetCursorPos", uintptr(x), uintptr(y))
}

// GetMessageExtraInfo is the extra information of the message that the thread is handling; the
// signature 0xFF515700 in its high 24 bits says that a mouse message was made from touch or pen.
//
// Flutter: no counterpart
func GetMessageExtraInfo() uintptr {
	r, _, err := user32.call("GetMessageExtraInfo")
	if err != nil {
		return 0
	}
	return r
}

// GetPointerInfo is the state of a pointer of the pointer messages.
//
// Flutter: shell/platform/windows/windows_proc_table.cc WindowsProcTable::GetPointerInfo
func GetPointerInfo(id uint32) (PointerInfo, bool) {
	var info PointerInfo
	r, _, err := user32.call("GetPointerInfo", uintptr(id), uintptr(unsafe.Pointer(&info)))
	return info, err == nil && r != 0
}

// GetPointerFrameInfo lists the pointers of the frame that the pointer belongs to (the contacts
// that moved together), the pointer included.
//
// Flutter: no counterpart
func GetPointerFrameInfo(id uint32) ([]PointerInfo, bool) {
	var count uint32
	r, _, err := user32.call("GetPointerFrameInfo", uintptr(id), uintptr(unsafe.Pointer(&count)), 0)
	if err != nil || r == 0 || count == 0 {
		return nil, false
	}
	infos := make([]PointerInfo, count)
	r, _, err = user32.call("GetPointerFrameInfo", uintptr(id), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&infos[0])))
	if err != nil || r == 0 {
		return nil, false
	}
	return infos[:count], true
}

// GetPointerTouchInfo is the touch information of a touch pointer.
//
// Flutter: no counterpart
func GetPointerTouchInfo(id uint32) (PointerTouchInfo, bool) {
	var info PointerTouchInfo
	r, _, err := user32.call("GetPointerTouchInfo", uintptr(id), uintptr(unsafe.Pointer(&info)))
	return info, err == nil && r != 0
}

// GetPointerPenInfo is the pen information of a pen pointer.
//
// Flutter: shell/platform/windows/windows_proc_table.cc WindowsProcTable::GetPointerPenInfo
func GetPointerPenInfo(id uint32) (PointerPenInfo, bool) {
	var info PointerPenInfo
	r, _, err := user32.call("GetPointerPenInfo", uintptr(id), uintptr(unsafe.Pointer(&info)))
	return info, err == nil && r != 0
}

// SetWindowFeedbackSetting turns one kind of visual feedback of touch and pen (the contact ring,
// the barrel visualization) on or off for the window.
//
// Flutter: no counterpart
func SetWindowFeedbackSetting(hwnd uintptr, feedback uint32, enabled bool) bool {
	value := uint32(boolInt(enabled))
	return boolCall(user32, "SetWindowFeedbackSetting", hwnd, uintptr(feedback), 0, unsafe.Sizeof(value), uintptr(unsafe.Pointer(&value)))
}

// SystemParametersInfoUint reads a system parameter that is one number (SPI_GET*): the lines that
// a notch of the wheel scrolls.
//
// Flutter: no counterpart
func SystemParametersInfoUint(action uint32) (uint32, bool) {
	var v uint32
	r, _, err := user32.call("SystemParametersInfoW", uintptr(action), 0, uintptr(unsafe.Pointer(&v)), 0)
	return v, err == nil && r != 0
}

// SendInput puts mouse input into the system's input queue as a user's device does, and returns how
// many inputs it inserted. The tests use it to click and move the pointer.
//
// Flutter: no counterpart
func SendInput(inputs []Input) uint32 {
	if len(inputs) == 0 {
		return 0
	}
	r, _, err := user32.call("SendInput", uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(Input{}))
	if err != nil {
		return 0
	}
	return uint32(r)
}

// WindowFromPoint is the window under a place of the screen, 0 when there is none.
//
// Flutter: no counterpart
func WindowFromPoint(pt Point) uintptr {
	// POINT is passed by value: its two 32-bit halves are one 64-bit argument.
	r, _, err := user32.call("WindowFromPoint", uintptr(uint32(pt.X))|uintptr(uint32(pt.Y))<<32)
	if err != nil {
		return 0
	}
	return r
}
