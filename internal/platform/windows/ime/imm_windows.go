//go:build windows

package ime

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// System is the IMM interface over imm32.dll and user32.dll.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager
type System struct{}

var _ IMM = System{}

// Context implements IMM with ImmGetContext.
//
// Flutter: shell/platform/windows/text_input_manager.cc ImmContext
func (System) Context(hwnd uintptr) uintptr { return win32.ImmGetContext(hwnd) }

// Release implements IMM with ImmReleaseContext.
//
// Flutter: shell/platform/windows/text_input_manager.cc ImmContext
func (System) Release(hwnd, ctx uintptr) { win32.ImmReleaseContext(hwnd, ctx) }

// String implements IMM with ImmGetCompositionStringW.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::GetString
func (System) String(ctx uintptr, index uint32) ([]uint16, bool) {
	return win32.ImmGetCompositionString(ctx, index)
}

// Value implements IMM with ImmGetCompositionStringW.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::GetComposingCursorPosition
func (System) Value(ctx uintptr, index uint32) int32 { return win32.ImmGetCompositionValue(ctx, index) }

// SetCompositionWindow implements IMM with ImmSetCompositionWindow.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func (System) SetCompositionWindow(ctx uintptr, form *win32.CompositionForm) bool {
	return win32.ImmSetCompositionWindow(ctx, form)
}

// SetCandidateWindow implements IMM with ImmSetCandidateWindow.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func (System) SetCandidateWindow(ctx uintptr, form *win32.CandidateForm) bool {
	return win32.ImmSetCandidateWindow(ctx, form)
}

// Notify implements IMM with ImmNotifyIME.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::AbortComposing
func (System) Notify(ctx uintptr, action, index, value uint32) bool {
	return win32.ImmNotifyIME(ctx, action, index, value)
}

// SetCompositionString implements IMM with ImmSetCompositionStringW and SCS_SETSTR.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::AbortComposing
func (System) SetCompositionString(ctx uintptr, comp, read []uint16) bool {
	return win32.ImmSetCompositionString(ctx, win32.SCSSetStr, comp, read)
}

// AssociateContext implements IMM with ImmAssociateContext.
//
// Flutter: no counterpart
func (System) AssociateContext(hwnd, ctx uintptr) uintptr {
	return win32.ImmAssociateContext(hwnd, ctx)
}

// AssociateContextEx implements IMM with ImmAssociateContextEx.
//
// Flutter: no counterpart
func (System) AssociateContextEx(hwnd, ctx uintptr, flags uint32) bool {
	return win32.ImmAssociateContextEx(hwnd, ctx, flags)
}

// CreateCaret implements IMM with CreateCaret.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::CreateImeWindow
func (System) CreateCaret(hwnd uintptr, w, h int32) bool { return win32.CreateCaret(hwnd, w, h) }

// DestroyCaret implements IMM with DestroyCaret.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::DestroyImeWindow
func (System) DestroyCaret() bool { return win32.DestroyCaret() }

// SetCaretPos implements IMM with SetCaretPos.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func (System) SetCaretPos(x, y int32) bool { return win32.SetCaretPos(x, y) }

// Focus implements IMM with GetFocus.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func (System) Focus() uintptr { return win32.GetFocus() }
