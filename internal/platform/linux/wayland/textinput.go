//go:build linux && !android

package wayland

import (
	"math"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// maxSurroundingBytes is the most text that set_surrounding_text can carry: the
// protocol limits it to 4000 bytes because of the size of a Wayland message.
const maxSurroundingBytes = 4000

// textInput is the input method of the seat through zwp_text_input_v3 (SPEC
// §8.6, §8.11 "Input"). The compositor decides which surface has the text-input
// focus (it follows the keyboard focus) with enter and leave; the app enables
// the input method while a text field of that surface has the focus, and sends
// its state with commit. The changes the input method asks for come as
// preedit_string, commit_string and delete_surrounding_text, applied at done. It
// belongs to the UI thread.
//
// The protocol calls for a client that doesn't send the surrounding text to be
// treated as one without that support, so the surrounding text is always sent.
type textInput struct {
	p   *Platform
	mgr *wl.ZwpTextInputManagerV3
	obj *wl.ZwpTextInputV3

	// focus is the window that has the text-input focus (enter without leave), or
	// nil.
	focus *window
	// enabled is whether the compositor has the input method enabled for the
	// focus, and sent is the state the compositor has.
	enabled bool
	sent    ui.TextInputState
	// commits counts the commit requests, which is the serial of done.
	commits uint32

	// want is the last state the app asked for, and wantWindow its window.
	want       ui.TextInputState
	wantWindow *window

	// pending are the changes of the events since the last done.
	pending textInputPending
	// preedit is whether the app shows a preedit now.
	preedit bool
}

// textInputPending holds the double-buffered state of the events.
type textInputPending struct {
	preedit                   string
	preeditBegin, preeditEnd  int32
	commit                    string
	deleteBefore, deleteAfter uint32
}

// newTextInput binds the text input of the seat. It returns nil when the
// compositor has no zwp_text_input_manager_v3.
func newTextInput(p *Platform) *textInput {
	g, ok := p.sess.optionalPresent[wl.ZwpTextInputManagerV3Interface.Name]
	if !ok {
		return nil
	}
	t := &textInput{p: p}
	t.mgr = wl.BindZwpTextInputManagerV3(p.sess.registry, g.Name, bindVersion(g, 1))
	t.obj = t.mgr.GetTextInput(p.sess.seat)
	t.obj.SetEvents(&wl.ZwpTextInputV3Events{
		OnEnter:                 t.onEnter,
		OnLeave:                 t.onLeave,
		OnPreeditString:         t.onPreeditString,
		OnCommitString:          t.onCommitString,
		OnDeleteSurroundingText: t.onDeleteSurroundingText,
		OnDone:                  t.onDone,
	})
	return t
}

// close destroys the text input object.
func (t *textInput) close() {
	if t.obj != nil {
		t.obj.Destroy()
		t.obj = nil
	}
	// The manager has a destructor, which goes with the connection.
	t.focus = nil
}

// forget drops what the text input knows about a window that is closing.
func (t *textInput) forget(w *window) {
	if t.focus == w {
		t.focus = nil
		t.enabled = false
		t.sent = ui.TextInputState{}
		t.pending = textInputPending{}
		t.preedit = false
	}
	if t.wantWindow == w {
		t.wantWindow = nil
		t.want = ui.TextInputState{}
	}
}

// set is Window.SetTextInput: it records what the app wants and tells the
// compositor if the window has the text-input focus.
func (t *textInput) set(w *window, state ui.TextInputState) {
	if !state.Enabled && t.wantWindow != nil && t.wantWindow != w {
		return // another window's text field is the one that has the input method
	}
	t.want, t.wantWindow = state, w
	t.sync()
}

func (t *textInput) onEnter(surface *wl.WlSurface) {
	t.focus = t.p.windowOf(surface)
	// Entering resets the state of the compositor's side.
	t.enabled = false
	t.sent = ui.TextInputState{}
	t.pending = textInputPending{}
	t.sync()
}

func (t *textInput) onLeave(*wl.WlSurface) {
	t.focus = nil
	t.enabled = false
	t.sent = ui.TextInputState{}
	t.pending = textInputPending{}
	// The client should reset any preedit string previously set.
	if t.preedit {
		t.preedit = false
		t.deliver(ui.TextInputEvent{PreeditCursorBegin: -1, PreeditCursorEnd: -1})
	}
}

// sync makes the compositor's state the one the app wants, if the window of the
// text field has the text-input focus.
func (t *textInput) sync() {
	if t.obj == nil || t.focus == nil {
		return
	}
	want := t.want
	if !want.Enabled || t.wantWindow != t.focus {
		if t.enabled {
			t.obj.Disable()
			t.commit()
			t.enabled = false
			t.sent = ui.TextInputState{}
		}
		return
	}
	changed := false
	if !t.enabled {
		t.obj.Enable()
		t.enabled = true
		t.sent = ui.TextInputState{}
		changed = true
	}
	text, cursor, anchor := clipSurrounding(want.Text, want.SelectionCursor, want.SelectionAnchor)
	sentText, sentCursor, sentAnchor := clipSurrounding(t.sent.Text, t.sent.SelectionCursor, t.sent.SelectionAnchor)
	if changed || text != sentText || cursor != sentCursor || anchor != sentAnchor {
		t.obj.SetSurroundingText(text, int32(cursor), int32(anchor))
		if !want.CausedByInputMethod && !changed {
			// The text changed for another reason than the input method's own
			// request: it may want to drop its preedit.
			t.obj.SetTextChangeCause(wl.ZwpTextInputV3ChangeCauseOther)
		}
		changed = true
	}
	if t.sent.Purpose != want.Purpose || t.sent.Hints != want.Hints || !t.sent.Enabled {
		t.obj.SetContentType(contentHint(want.Hints), contentPurpose(want.Purpose))
		changed = true
	}
	if t.sent.CaretRect != want.CaretRect || !t.sent.Enabled {
		x, y, w, h := caretRectangle(want.CaretRect)
		t.obj.SetCursorRectangle(x, y, w, h)
		changed = true
	}
	t.sent = want
	if changed {
		t.commit()
	}
}

func (t *textInput) commit() {
	t.obj.Commit()
	t.commits++
}

func (t *textInput) onPreeditString(text string, begin, end int32) {
	t.pending.preedit, t.pending.preeditBegin, t.pending.preeditEnd = text, begin, end
}

func (t *textInput) onCommitString(text string) { t.pending.commit = text }

func (t *textInput) onDeleteSurroundingText(before, after uint32) {
	t.pending.deleteBefore, t.pending.deleteAfter = before, after
}

// onDone applies the changes of the events at once. A serial that is not the
// number of commits means the compositor has not seen the latest state yet; the
// changes are applied all the same, as the protocol says, and the state of the
// app is sent as it changes.
func (t *textInput) onDone(uint32) {
	pe := t.pending
	t.pending = textInputPending{}
	if t.focus == nil {
		return
	}
	event := ui.TextInputEvent{
		View:               t.focus.id,
		DeleteBefore:       int(pe.deleteBefore),
		DeleteAfter:        int(pe.deleteAfter),
		Commit:             pe.commit,
		Preedit:            pe.preedit,
		PreeditCursorBegin: int(pe.preeditBegin),
		PreeditCursorEnd:   int(pe.preeditEnd),
	}
	if event.Preedit == "" {
		event.PreeditCursorBegin, event.PreeditCursorEnd = -1, -1
	}
	hadPreedit := t.preedit
	t.preedit = event.Preedit != ""
	// An event that changes nothing is not worth a frame, unless it removes a
	// preedit that is shown.
	if event.Commit == "" && event.Preedit == "" && event.DeleteBefore == 0 && event.DeleteAfter == 0 && !hadPreedit {
		return
	}
	t.deliver(event)
}

func (t *textInput) deliver(e ui.TextInputEvent) {
	if h := t.p.handler; h != nil {
		h.OnTextInput(e)
	}
}

// clipSurrounding cuts the surrounding text to what a message can carry,
// around the cursor, at characters. The cursor and the anchor are offsets in the
// result.
func clipSurrounding(text string, cursor, anchor int) (string, int, int) {
	cursor, anchor = min(max(cursor, 0), len(text)), min(max(anchor, 0), len(text))
	if len(text) <= maxSurroundingBytes {
		return text, cursor, anchor
	}
	lo, hi := min(cursor, anchor), max(cursor, anchor)
	// Keep the selection when it fits, and put the rest of the room around it.
	room := maxSurroundingBytes - (hi - lo)
	start := lo - room/2
	if hi-lo > maxSurroundingBytes {
		start = cursor - maxSurroundingBytes/2
	}
	start = min(max(start, 0), len(text)-maxSurroundingBytes)
	end := start + maxSurroundingBytes
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	for end > start && end < len(text) && !utf8.RuneStart(text[end]) {
		end--
	}
	clip := func(off int) int { return min(max(off-start, 0), end-start) }
	return text[start:end], clip(cursor), clip(anchor)
}

// caretRectangle is the cursor rectangle in whole logical pixels, covering the
// caret.
func caretRectangle(r ui.Rect) (x, y, w, h int32) {
	if !r.IsFinite() {
		return 0, 0, 0, 0
	}
	left, top := math.Floor(r.Left), math.Floor(r.Top)
	right, bottom := math.Ceil(r.Right), math.Ceil(r.Bottom)
	return int32(left), int32(top), int32(max(right-left, 0)), int32(max(bottom-top, 0))
}

// contentPurpose maps a purpose of ui to the content purpose of the protocol.
func contentPurpose(p ui.TextInputPurpose) wl.ZwpTextInputV3ContentPurpose {
	switch p {
	case ui.TextInputPurposeAlpha:
		return wl.ZwpTextInputV3ContentPurposeAlpha
	case ui.TextInputPurposeDigits:
		return wl.ZwpTextInputV3ContentPurposeDigits
	case ui.TextInputPurposeNumber:
		return wl.ZwpTextInputV3ContentPurposeNumber
	case ui.TextInputPurposePhone:
		return wl.ZwpTextInputV3ContentPurposePhone
	case ui.TextInputPurposeURL:
		return wl.ZwpTextInputV3ContentPurposeUrl
	case ui.TextInputPurposeEmail:
		return wl.ZwpTextInputV3ContentPurposeEmail
	case ui.TextInputPurposeName:
		return wl.ZwpTextInputV3ContentPurposeName
	case ui.TextInputPurposePassword:
		return wl.ZwpTextInputV3ContentPurposePassword
	case ui.TextInputPurposePIN:
		return wl.ZwpTextInputV3ContentPurposePin
	case ui.TextInputPurposeDate:
		return wl.ZwpTextInputV3ContentPurposeDate
	case ui.TextInputPurposeTime:
		return wl.ZwpTextInputV3ContentPurposeTime
	case ui.TextInputPurposeDatetime:
		return wl.ZwpTextInputV3ContentPurposeDatetime
	}
	return wl.ZwpTextInputV3ContentPurposeNormal
}

// contentHint maps the hints of ui to the content hints of the protocol.
func contentHint(h ui.TextInputHints) wl.ZwpTextInputV3ContentHint {
	// The app draws the preedit in the text itself.
	out := wl.ZwpTextInputV3ContentHintPreeditShown
	for _, m := range []struct {
		from ui.TextInputHints
		to   wl.ZwpTextInputV3ContentHint
	}{
		{ui.TextInputHintCompletion, wl.ZwpTextInputV3ContentHintCompletion},
		{ui.TextInputHintSpellcheck, wl.ZwpTextInputV3ContentHintSpellcheck},
		{ui.TextInputHintAutoCapitalization, wl.ZwpTextInputV3ContentHintAutoCapitalization},
		{ui.TextInputHintLowercase, wl.ZwpTextInputV3ContentHintLowercase},
		{ui.TextInputHintUppercase, wl.ZwpTextInputV3ContentHintUppercase},
		{ui.TextInputHintTitlecase, wl.ZwpTextInputV3ContentHintTitlecase},
		{ui.TextInputHintHiddenText, wl.ZwpTextInputV3ContentHintHiddenText},
		{ui.TextInputHintSensitiveData, wl.ZwpTextInputV3ContentHintSensitiveData},
		{ui.TextInputHintLatin, wl.ZwpTextInputV3ContentHintLatin},
		{ui.TextInputHintMultiline, wl.ZwpTextInputV3ContentHintMultiline},
	} {
		if h.Has(m.from) {
			out |= m.to
		}
	}
	return out
}
