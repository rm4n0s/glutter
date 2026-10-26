//go:build windows

package win32

import (
	"unsafe"
)

// The bindings of imm32.dll and the caret calls of user32.dll for the input methods of step 28.5
// (SPEC §25.7): the input context of a window, the composition string, the places of the composition
// and candidate windows, and the association of a window with its context. A function whose failure
// the caller can only skip answers 0 or false.

var imm32 = &dllTable{name: "imm32.dll"}

// ImmGetContext is the input context of a window (an HIMC), 0 when the window has none: the input
// method is off for it (ImmAssociateContext with 0). ImmReleaseContext gives it back.
//
// Flutter: shell/platform/windows/text_input_manager.cc ImmContext
func ImmGetContext(hwnd uintptr) uintptr {
	r, _, err := imm32.call("ImmGetContext", hwnd)
	if err != nil {
		return 0
	}
	return r
}

// ImmReleaseContext releases the context that ImmGetContext returned.
//
// Flutter: shell/platform/windows/text_input_manager.cc ImmContext
func ImmReleaseContext(hwnd, himc uintptr) {
	_, _, _ = imm32.call("ImmReleaseContext", hwnd, himc)
}

// ImmGetCompositionString reads a string of the composition (GCSCompStr or GCSResultStr) as UTF-16
// units, and reports whether the call worked. It is ImmGetCompositionStringW called twice, once for
// the length in bytes.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::GetString
func ImmGetCompositionString(himc uintptr, index uint32) ([]uint16, bool) {
	n := immGetCompositionString(himc, index, 0, 0)
	if n < 0 {
		return nil, false
	}
	units := make([]uint16, n/2)
	if len(units) > 0 {
		immGetCompositionString(himc, index, uintptr(unsafe.Pointer(&units[0])), uint32(n))
	}
	return units, true
}

// ImmGetCompositionValue reads a number of the composition, such as the cursor position
// (GCSCursorPos): the value of ImmGetCompositionStringW with no buffer. It is negative when the
// call failed.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::GetComposingCursorPosition
func ImmGetCompositionValue(himc uintptr, index uint32) int32 {
	return immGetCompositionString(himc, index, 0, 0)
}

// immGetCompositionString is ImmGetCompositionStringW; a pointer converted in its argument list
// stays alive for the call (t.call).
//
//go:uintptrescapes
func immGetCompositionString(himc uintptr, index uint32, buf uintptr, size uint32) int32 {
	r, _, err := imm32.call("ImmGetCompositionStringW", himc, uintptr(index), buf, uintptr(size))
	if err != nil {
		return -1
	}
	return int32(r)
}

// ImmSetCompositionString sets the composition string of an input context (SCSSetStr with a string
// and its reading), which makes the input method send the composition messages. It reports whether
// the call worked.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::AbortComposing
func ImmSetCompositionString(himc uintptr, index uint32, comp, read []uint16) bool {
	// The pointers stay unsafe.Pointer until the argument list, where call keeps them alive (SPEC §25.2).
	var pc, pr unsafe.Pointer
	if len(comp) > 0 {
		pc = unsafe.Pointer(&comp[0])
	}
	if len(read) > 0 {
		pr = unsafe.Pointer(&read[0])
	}
	r, _, err := imm32.call("ImmSetCompositionStringW", himc, uintptr(index), uintptr(pc), uintptr(len(comp)*2), uintptr(pr), uintptr(len(read)*2))
	return err == nil && r != 0
}

// ImmSetCompositionWindow sets where the input method draws the composition string.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func ImmSetCompositionWindow(himc uintptr, form *CompositionForm) bool {
	return boolCall(imm32, "ImmSetCompositionWindow", himc, uintptr(unsafe.Pointer(form)))
}

// ImmGetCompositionWindow reads where the input method draws the composition string.
//
// Flutter: no counterpart
func ImmGetCompositionWindow(himc uintptr) (CompositionForm, bool) {
	var f CompositionForm
	ok := boolCall(imm32, "ImmGetCompositionWindow", himc, uintptr(unsafe.Pointer(&f)))
	return f, ok
}

// ImmSetCandidateWindow sets where the input method opens the window of its candidates; the form's
// Index says which candidate list.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func ImmSetCandidateWindow(himc uintptr, form *CandidateForm) bool {
	return boolCall(imm32, "ImmSetCandidateWindow", himc, uintptr(unsafe.Pointer(form)))
}

// ImmGetCandidateWindow reads where the input method opens the window of the candidate list index.
//
// Flutter: no counterpart
func ImmGetCandidateWindow(himc uintptr, index uint32) (CandidateForm, bool) {
	var f CandidateForm
	ok := boolCall(imm32, "ImmGetCandidateWindow", himc, uintptr(index), uintptr(unsafe.Pointer(&f)))
	return f, ok
}

// ImmNotifyIME tells the input method to do something with the composition (NICompositionStr with
// CPSCancel or CPSComplete) or the candidates (NICloseCandidate).
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::AbortComposing
func ImmNotifyIME(himc uintptr, action, index, value uint32) bool {
	return boolCall(imm32, "ImmNotifyIME", himc, uintptr(action), uintptr(index), uintptr(value))
}

// ImmAssociateContext gives a window an input context and returns the one it had. With 0 the input
// method is off for the window.
//
// Flutter: no counterpart
func ImmAssociateContext(hwnd, himc uintptr) uintptr {
	r, _, err := imm32.call("ImmAssociateContext", hwnd, himc)
	if err != nil {
		return 0
	}
	return r
}

// ImmAssociateContextEx associates the window with an input context as the flags say; with 0 and
// IACEDefault the window gets the default context back, which turns the input method on again.
//
// Flutter: no counterpart
func ImmAssociateContextEx(hwnd, himc uintptr, flags uint32) bool {
	return boolCall(imm32, "ImmAssociateContextEx", hwnd, himc, uintptr(flags))
}

// ImmGetOpenStatus says whether the input method of the context is open (composing is possible).
//
// Flutter: no counterpart
func ImmGetOpenStatus(himc uintptr) bool { return boolCall(imm32, "ImmGetOpenStatus", himc) }

// ImmSetOpenStatus opens or closes the input method of the context.
//
// Flutter: no counterpart
func ImmSetOpenStatus(himc uintptr, open bool) bool {
	return boolCall(imm32, "ImmSetOpenStatus", himc, boolInt(open))
}

// CreateCaret makes a system caret for the window with a bitmap of the size (a null bitmap makes a
// solid one). The caret is hidden until ShowCaret; some input methods read its place to put their
// windows (SPEC §25.7).
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::CreateImeWindow
func CreateCaret(hwnd uintptr, w, h int32) bool {
	return boolCall(user32, "CreateCaret", hwnd, 0, uintptr(w), uintptr(h))
}

// DestroyCaret destroys the system caret of the calling thread.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::DestroyImeWindow
func DestroyCaret() bool { return boolCall(user32, "DestroyCaret") }

// SetCaretPos moves the system caret to client coordinates of its window.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func SetCaretPos(x, y int32) bool { return boolCall(user32, "SetCaretPos", uintptr(x), uintptr(y)) }

// GetCaretPos is the place of the system caret in client coordinates of its window.
//
// Flutter: no counterpart
func GetCaretPos() (Point, bool) {
	var p Point
	ok := boolCall(user32, "GetCaretPos", uintptr(unsafe.Pointer(&p)))
	return p, ok
}
