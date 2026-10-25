//go:build windows

package windows

import (
	"unicode/utf16"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ime"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the glue of the text input of a window (SPEC §25.7): the input method messages and
// the characters that no key made go to the plain-Go model (internal/platform/windows/ime), and the
// events it makes go to the handler at once. The scripted events type and compose through the same
// window procedure.

// scriptedIMM is the IMM interface of the platform: it asks Windows for everything, except that
// while a scripted composition (the compose: and composing: steps) is going on, the composition
// string, the result string and the caret in it are the script's. The messages that tell the window
// about the composition are real window messages; only the strings that a real input method would
// have put into the input context come from here.
//
// Flutter: no counterpart
type scriptedIMM struct {
	ime.IMM
	on     bool
	comp   []uint16
	result []uint16
	cursor int32
}

var _ ime.IMM = (*scriptedIMM)(nil)

func (s *scriptedIMM) String(ctx uintptr, index uint32) ([]uint16, bool) {
	if s.on {
		switch index {
		case win32.GCSCompStr:
			return s.comp, true
		case win32.GCSResultStr:
			return s.result, true
		}
	}
	return s.IMM.String(ctx, index)
}

func (s *scriptedIMM) Value(ctx uintptr, index uint32) int32 {
	if s.on && index == win32.GCSCursorPos {
		return s.cursor
	}
	return s.IMM.Value(ctx, index)
}

// initText makes the text input of the window: the input method is off for the window until a text
// field has the focus.
func (w *window) initText() {
	w.text = ime.New(w.p.imm, w.hwnd, w.deliverText)
}

// deliverText gives the framework an event of the text input at once, after the input that waits
// for its frame (the same order as the keys).
func (w *window) deliverText(ev ui.TextInputEvent) {
	h := w.p.handler
	if h == nil || w.closed {
		return
	}
	ev.View = w.id
	if s, ok := h.(platform.SyncInputHandler); ok {
		s.OnTextInputSync(ev)
		return
	}
	h.OnTextInput(ev)
}

// SetTextInput tells the input method about the focused text field: the input method is on while a
// field has the focus, and its windows follow the caret rectangle (SPEC §25.7).
//
// Flutter: shell/platform/windows/text_input_plugin.cc TextInputPlugin::HandleMethodCall
func (w *window) SetTextInput(state ui.TextInputState) {
	if w.closed || w.text == nil {
		return
	}
	w.text.SetState(state, w.scale())
}

// onIME handles the messages of the input method. It returns what the window procedure answers and
// whether the message is handled, and the lParam to pass to DefWindowProc when it is not.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_IME_SETCONTEXT, WM_IME_STARTCOMPOSITION, WM_IME_COMPOSITION, WM_IME_ENDCOMPOSITION)
func (w *window) onIME(msg uint32, wparam, lparam uintptr) (result uintptr, handled bool, out uintptr) {
	if w.text == nil {
		return 0, false, lparam
	}
	switch msg {
	case win32.WMIMESetContext:
		return 0, false, w.text.HandleSetContext(wparam, lparam)
	case win32.WMIMEStartComposition:
		w.text.HandleStartComposition()
		return 0, true, lparam
	case win32.WMIMEComposition:
		return 0, w.text.HandleComposition(lparam), lparam
	case win32.WMIMEEndComposition:
		w.text.HandleEndComposition()
		return 0, true, lparam
	}
	return 0, false, lparam
}

// runDevText does one scripted text step: the characters arrive as WM_CHAR messages at the first
// window, one for each UTF-16 unit, as a keyboard layout or an input method sends them.
//
// Flutter: no counterpart
func (p *Platform) runDevText(ev DevEvent) {
	w := p.devWindow()
	if w == nil {
		p.log.Warn("a scripted event found no window", "step", ev.Kind)
		return
	}
	for _, u := range utf16.Encode([]rune(ev.Text)) {
		win32.SendMessage(w.hwnd, win32.WMChar, uintptr(u), 1)
	}
}

// composeFlags are the flags of a WM_IME_COMPOSITION that carries a composition string, as the
// Microsoft input methods send them.
const composeFlags = win32.GCSCompStr | win32.GCSCompAttr | win32.GCSCompClause | win32.GCSCursorPos | win32.GCSDeltaStart

// runDevCompose does one scripted composition: the messages of an input method that starts a
// composition (unless one is going on), composes each piece in turn, and (compose:) commits the last
// piece and ends, or (composing:) leaves the composition going. The caret is at the end of each
// piece.
//
// Flutter: no counterpart
func (p *Platform) runDevCompose(ev DevEvent) {
	w := p.devWindow()
	if w == nil || w.text == nil {
		p.log.Warn("a scripted event found no window", "step", ev.Kind)
		return
	}
	s := p.script
	s.on = true
	defer func() { s.on, s.comp, s.result = false, nil, nil }()
	if !w.text.Composing() {
		win32.SendMessage(w.hwnd, win32.WMIMEStartComposition, 0, 0)
	}
	for _, piece := range ev.Steps {
		s.comp = utf16.Encode([]rune(piece))
		s.cursor = int32(len(s.comp))
		win32.SendMessage(w.hwnd, win32.WMIMEComposition, 0, composeFlags)
	}
	if ev.Kind != "compose" {
		return
	}
	s.comp = nil
	s.result = utf16.Encode([]rune(ev.Steps[len(ev.Steps)-1]))
	win32.SendMessage(w.hwnd, win32.WMIMEComposition, 0, win32.GCSResultStr)
	win32.SendMessage(w.hwnd, win32.WMIMEEndComposition, 0, 0)
}
