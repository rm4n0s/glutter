package ime

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/internal/platform/textedit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// IMM is what the text input asks of Windows: the input context of a window and its composition,
// the windows of the input method, the association of the window with its context and the system
// caret that some input methods read. All calls belong to the UI thread. The real implementation is
// System; the tests use a fake.
//
// Flutter: shell/platform/windows/text_input_manager.h TextInputManager (the Win32 calls it makes)
type IMM interface {
	// Context returns the input context of the window, 0 when the input method is off for it.
	// Release gives it back.
	Context(hwnd uintptr) uintptr
	Release(hwnd, ctx uintptr)
	// String reads the composition string (win32.GCSCompStr) or the result string
	// (win32.GCSResultStr) as UTF-16 units, and reports whether the call worked.
	String(ctx uintptr, index uint32) ([]uint16, bool)
	// Value reads a number of the composition, such as win32.GCSCursorPos; negative when the call
	// failed.
	Value(ctx uintptr, index uint32) int32
	// SetCompositionWindow and SetCandidateWindow place the windows of the input method.
	SetCompositionWindow(ctx uintptr, form *win32.CompositionForm) bool
	SetCandidateWindow(ctx uintptr, form *win32.CandidateForm) bool
	// Notify is ImmNotifyIME; SetCompositionString clears the composition string.
	Notify(ctx uintptr, action, index, value uint32) bool
	SetCompositionString(ctx uintptr, comp, read []uint16) bool
	// AssociateContext and AssociateContextEx turn the input method off and on for the window.
	AssociateContext(hwnd, ctx uintptr) uintptr
	AssociateContextEx(hwnd, ctx uintptr, flags uint32) bool
	// The system caret, and the window that has the keyboard focus.
	CreateCaret(hwnd uintptr, w, h int32) bool
	DestroyCaret() bool
	SetCaretPos(x, y int32) bool
	Focus() uintptr
}

// Input is the text input of one window: it keeps the framework's text field as the editor of the
// platform knows it, and turns what the input method of Windows does into events for the field
// (SPEC §25.7). It belongs to the UI thread.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager
type Input struct {
	imm     IMM
	hwnd    uintptr
	deliver func(ui.TextInputEvent)

	// editor holds the framework's text, selection and preedit, and finds the event that makes the
	// framework's field what the input method says (SPEC §23.5).
	editor textedit.TextEditor
	// enabled says a text field has the focus; imeOn says the window has its input context.
	enabled, imeOn bool
	// imeActive is Flutter's ime_active_: the system caret exists and the windows of the input
	// method are placed.
	imeActive bool
	// composing says the input method started a composition and has not ended it; preedit is the
	// composing text the framework shows now.
	composing bool
	preedit   string
	// caret is the caret rectangle in the client pixels of the window.
	caret win32.Rect
	// high is a high surrogate of a WM_CHAR that waits for its low surrogate.
	high uint16
}

// New makes the text input of a window. The input method is off for the window until a text field
// has the focus. deliver gets the events for the framework.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::SetWindowHandle
func New(imm IMM, hwnd uintptr, deliver func(ui.TextInputEvent)) *Input {
	in := &Input{imm: imm, hwnd: hwnd, deliver: deliver, imeOn: true}
	in.setIME(false)
	return in
}

// Enabled says whether a text field has the focus.
//
// Flutter: no counterpart
func (in *Input) Enabled() bool { return in.enabled }

// IMEOn says whether the window has its input context: the input method is on for it.
//
// Flutter: no counterpart
func (in *Input) IMEOn() bool { return in.imeOn }

// Composing says whether the input method is composing text.
//
// Flutter: shell/platform/windows/text_input_model.h TextInputModel::composing
func (in *Input) Composing() bool { return in.composing }

// Preedit is the composing text the framework shows.
//
// Flutter: no counterpart
func (in *Input) Preedit() string { return in.preedit }

// CaretRect is the caret rectangle in client pixels, as the windows of the input method follow it.
//
// Flutter: no counterpart
func (in *Input) CaretRect() win32.Rect { return in.caret }

// setIME turns the input method on or off for the window: off is ImmAssociateContext(hwnd, NULL),
// on gives the window its default context back.
func (in *Input) setIME(on bool) {
	if on == in.imeOn {
		return
	}
	in.imeOn = on
	if on {
		in.imm.AssociateContextEx(in.hwnd, 0, win32.IACEDefault)
		return
	}
	in.imm.AssociateContext(in.hwnd, 0)
}

// SetState takes what the framework says about the focused text field (platform.Window.SetTextInput).
// A field that gets the focus turns the input method on and puts its windows at the caret; the
// caret rectangle (logical pixels of the view, scale pixels each) moves them when it changes; no
// field turns the input method off and ends a composition that was going on. A change of the text
// or the selection leaves the input method's composition alone: the next message of the input method
// is applied to the new text (Flutter does the same).
//
// Flutter: shell/platform/windows/text_input_plugin.cc TextInputPlugin::HandleMethodCall (TextInput.setClient, TextInput.setEditingState, TextInput.clearClient, TextInput.setMarkedTextRect)
func (in *Input) SetState(st ui.TextInputState, scale float64) {
	if !st.Enabled {
		in.disable()
		return
	}
	if !in.enabled {
		in.enabled = true
		in.setIME(true)
		in.createIMEWindow()
	}
	in.editor.SetState(st)
	if scale <= 0 {
		scale = 1
	}
	rect := CaretRect(st.CaretRect, scale)
	if rect != in.caret {
		in.caret = rect
		in.moveIMEWindow()
	}
}

// disable handles the loss of the text field: a composition is cancelled in the input method, and
// the input method is turned off.
func (in *Input) disable() {
	if !in.enabled {
		return
	}
	if in.composing {
		in.abortComposing()
	}
	in.destroyIMEWindow()
	in.enabled = false
	in.composing = false
	in.preedit = ""
	in.high = 0
	in.editor.Reset()
	in.setIME(false)
}

// CaretRect is the caret rectangle of the framework (logical pixels of the view) in client pixels:
// each edge times the scale, cut to a whole number as the conversion to a LONG does in Flutter.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnCursorRectUpdated
func CaretRect(r ui.Rect, scale float64) win32.Rect {
	return win32.Rect{
		Left: int32(r.Left * scale), Top: int32(r.Top * scale),
		Right: int32(r.Right * scale), Bottom: int32(r.Bottom * scale),
	}
}

// FocusChanged puts the windows of the input method at the caret again: the window may have got the
// keyboard focus after the field asked for them (they are only placed while the window has it).
//
// Flutter: no counterpart
func (in *Input) FocusChanged() { in.moveIMEWindow() }

// createIMEWindow makes the system caret that some input methods read for their place and puts the
// windows of the input method at the caret.
//
// Deviation: Flutter makes it when the input method sends WM_IME_SETCONTEXT or starts a composition
// and removes it when the composition ends; here it lives as long as a text field has the focus, so
// that the first character of a composition already opens its windows at the caret (SPEC §12.10).
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::CreateImeWindow
func (in *Input) createIMEWindow() {
	if !in.imeActive {
		in.imm.CreateCaret(in.hwnd, 1, 1)
	}
	in.imeActive = true
	in.moveIMEWindow()
}

// destroyIMEWindow removes the system caret.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::DestroyImeWindow
func (in *Input) destroyIMEWindow() {
	if in.imeActive {
		in.imm.DestroyCaret()
	}
	in.imeActive = false
}

// moveIMEWindow sets the places of the composition window (at the caret's top left) and of the
// candidate window (next to the caret's bottom left, keeping off the caret's rectangle), and moves
// the system caret to its bottom left. Nothing happens while the window has no keyboard focus, or
// no input context.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::MoveImeWindow
func (in *Input) moveIMEWindow() {
	if !in.imeActive || !in.imeOn || in.imm.Focus() != in.hwnd {
		return
	}
	ctx := in.imm.Context(in.hwnd)
	if ctx == 0 {
		return
	}
	defer in.imm.Release(in.hwnd, ctx)
	r := in.caret
	in.imm.SetCaretPos(r.Left, r.Bottom)
	in.imm.SetCompositionWindow(ctx, &win32.CompositionForm{
		Style: win32.CFSPoint, CurrentPos: win32.Point{X: r.Left, Y: r.Top},
	})
	in.imm.SetCandidateWindow(ctx, &win32.CandidateForm{
		Index: 0, Style: win32.CFSExclude, CurrentPos: win32.Point{X: r.Left, Y: r.Bottom}, Area: r,
	})
}

// abortComposing cancels the composition in the input method, closes its candidate window and
// clears its composition string.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::AbortComposing
func (in *Input) abortComposing() {
	ctx := in.imm.Context(in.hwnd)
	if ctx == 0 {
		return
	}
	defer in.imm.Release(in.hwnd, ctx)
	in.imm.Notify(ctx, win32.NICompositionStr, win32.CPSCancel, 0)
	in.imm.Notify(ctx, win32.NICloseCandidate, 0, 0)
	in.imm.SetCompositionString(ctx, []uint16{0}, []uint16{0})
}

// HandleSetContext handles WM_IME_SETCONTEXT and returns the lParam for DefWindowProc: the flag that
// asks the input method to draw its own composition window is cleared, because the field draws the
// preedit. A context that becomes active places the windows of the input method.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_IME_SETCONTEXT), OnImeSetContext
func (in *Input) HandleSetContext(wparam, lparam uintptr) uintptr {
	if wparam != 0 && in.enabled {
		in.createIMEWindow()
	}
	return lparam &^ win32.ISCShowUICompositionWindow
}

// HandleStartComposition handles WM_IME_STARTCOMPOSITION: a composition begins. The window procedure
// answers 0 without DefWindowProc, so the system draws no composition window of its own.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnImeStartComposition, OnComposeBegin
func (in *Input) HandleStartComposition() {
	if in.enabled {
		in.createIMEWindow()
	}
	in.composing = true
}

// HandleComposition handles WM_IME_COMPOSITION and reports whether the window procedure answers 0
// without DefWindowProc: when the message carries the result string or the composition string
// (otherwise DefWindowProc would make WM_IME_CHAR and WM_CHAR messages of the result). The result
// string is a commit, the composition string is the preedit with the caret inside it; a message that
// carries both (Google Japanese Input and ATOK do) makes one event, in the order of the event: the
// commit, then the new preedit. A message with no flag cancels the composition.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnImeComposition
func (in *Input) HandleComposition(lparam uintptr) bool {
	in.moveIMEWindow()
	if lparam == 0 {
		in.change("", "", 0)
		return false
	}
	var commit, preedit string
	cursor := -1
	changed := false
	if lparam&win32.GCSResultStr != 0 {
		if units, ok := in.read(win32.GCSResultStr); ok {
			commit, changed = utf16ToString(units), true
		}
	}
	if lparam&win32.GCSCompStr != 0 {
		if units, ok := in.read(win32.GCSCompStr); ok {
			preedit, changed = utf16ToString(units), true
			cursor = in.cursor(lparam, units)
		}
	}
	if changed {
		in.change(commit, preedit, cursor)
	}
	return lparam&(win32.GCSResultStr|win32.GCSCompStr) != 0
}

// read reads a string of the composition of the window's input context: the units, and whether the
// call worked. A window with no context, or a text input that is off, reads nothing.
//
// Flutter: shell/platform/windows/text_input_manager.cc TextInputManager::GetString
func (in *Input) read(index uint32) ([]uint16, bool) {
	if !in.enabled || !in.imeActive {
		return nil, false
	}
	ctx := in.imm.Context(in.hwnd)
	if ctx == 0 {
		return nil, false
	}
	defer in.imm.Release(in.hwnd, ctx)
	return in.imm.String(ctx, index)
}

// cursor is the position of the caret in the composition string, in UTF-16 units: when the message
// does not say, or the position is outside the text, it is the end of the string.
//
// Flutter: shell/platform/windows/flutter_window.cc GetCursorPositionForComposition
func (in *Input) cursor(lparam uintptr, units []uint16) int {
	if lparam&win32.GCSCursorPos == 0 {
		return len(units)
	}
	ctx := in.imm.Context(in.hwnd)
	if ctx == 0 {
		return len(units)
	}
	defer in.imm.Release(in.hwnd, ctx)
	pos := int(in.imm.Value(ctx, win32.GCSCursorPos))
	if pos < 0 || pos > len(units) {
		return len(units)
	}
	return pos
}

// HandleEndComposition handles WM_IME_ENDCOMPOSITION: what is still being composed is committed (an
// input method that ends with a composition string and no result), and the composition is over.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnImeEndComposition, text_input_plugin.cc TextInputPlugin::ComposeEndHook
func (in *Input) HandleEndComposition() {
	if in.preedit != "" {
		in.change(in.preedit, "", 0)
	}
	in.composing = false
}

// change applies a commit and a new preedit (with the caret at the UTF-16 offset cursor of the
// preedit) to the framework's field: the text of the field after the input method replaced the
// selection and the old preedit is worked out as the element of the web would hold it, and the
// editor finds the event.
//
// Flutter: shell/platform/windows/text_input_plugin.cc TextInputPlugin::ComposeChangeHook, ComposeCommitHook
func (in *Input) change(commit, preedit string, cursor int) {
	if !in.enabled {
		return
	}
	st := in.editor.State()
	a, b := min(st.SelectionAnchor, st.SelectionCursor), max(st.SelectionAnchor, st.SelectionCursor)
	a, b = clampByte(st.Text, a), clampByte(st.Text, b)
	prefix := st.Text[:a] + commit
	base := utf16Len(prefix)
	dom := textedit.EditingState{
		Text: prefix + preedit + st.Text[b:], ComposingBase: -1, ComposingExtent: -1,
	}
	n := utf16Len(preedit)
	if cursor < 0 || cursor > n {
		cursor = n
	}
	at := base + cursor
	dom.Base, dom.Extent = at, at
	if preedit != "" {
		dom.ComposingBase, dom.ComposingExtent = base, base+utf16Len(preedit)
	}
	in.preedit = preedit
	ev, ok := in.editor.Handle(dom)
	if !ok {
		return
	}
	in.send(ev)
}

// HandleChar handles a WM_CHAR that no key made (the character of an input method that composes
// nothing, a key that SendInput sent as a Unicode character, a script's text): a character is typed
// as a commit. The two messages of a surrogate pair make one character, a control character makes
// none, and nothing is typed while no text field has the focus.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnText, text_input_plugin.cc TextInputPlugin::TextHook
func (in *Input) HandleChar(unit uint16) {
	var r rune
	switch {
	case utf16.IsSurrogate(rune(unit)) && unit < 0xDC00: // a high surrogate waits for the low one
		in.high = unit
		return
	case utf16.IsSurrogate(rune(unit)):
		high := in.high
		in.high = 0
		if high == 0 {
			return
		}
		r = utf16.DecodeRune(rune(high), rune(unit))
	default:
		in.high = 0
		r = rune(unit)
	}
	if r < ' ' || r == 0x7F || !in.enabled {
		return
	}
	in.Commit(string(r))
}

// Commit types text into the field at the caret, in place of the selection (and of the preedit).
//
// Flutter: shell/platform/windows/text_input_plugin.cc TextInputPlugin::TextHook
func (in *Input) Commit(text string) {
	if text == "" || !in.enabled {
		return
	}
	in.change(text, "", 0)
	// A preedit that was being composed is gone: the event replaced it.
}

// send gives the framework an event.
func (in *Input) send(ev ui.TextInputEvent) {
	if in.deliver != nil {
		in.deliver(ev)
	}
}

// utf16ToString turns UTF-16 units into a string; a lone surrogate becomes U+FFFD.
func utf16ToString(units []uint16) string { return string(utf16.Decode(units)) }

// utf16Len is the length of s in UTF-16 units.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// clampByte moves a byte offset into s and back to the start of the character it is in.
func clampByte(s string, off int) int {
	off = min(max(off, 0), len(s))
	for off > 0 && off < len(s) && !utf8.RuneStart(s[off]) {
		off--
	}
	return off
}
