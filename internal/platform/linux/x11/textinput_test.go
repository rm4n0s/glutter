//go:build linux

package x11

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/linux/ime/imetest"
	"github.com/rm4n0s/glutter/ui"
)

// textRecorder is the recorder that also keeps the text-input events.
type textRecorder struct {
	*recorder
	texts []ui.TextInputEvent
}

func (r *textRecorder) OnTextInput(e ui.TextInputEvent) { r.texts = append(r.texts, e) }

type imeEnv struct {
	in     *inputState
	rec    *textRecorder
	im     *imetest.IM
	poster *imetest.Poster
	origin [2]int
	w      *window
}

// newIMEInput builds the input state with a text input whose input method is
// the fake one. The window w has the keyboard focus.
func newIMEInput(t *testing.T, layout string) *imeEnv {
	t.Helper()
	in, rec := newInput(t, layout)
	e := &imeEnv{in: in, rec: &textRecorder{recorder: rec}, im: &imetest.IM{}, poster: imetest.NewPoster(), origin: [2]int{100, 50}}
	in.p.handler = e.rec
	in.p.input = in
	in.p.text = newTextInput(in.p, e.poster, e.im.Open)
	in.p.text.origin = func(*window) (int, int) { return e.origin[0], e.origin[1] }
	t.Cleanup(in.p.text.close)
	e.w = &window{p: in.p, id: 4}
	in.onFocus(e.w, &xiEnterEvent{detail: 0}, true)
	return e
}

func field(text string, caret ui.Rect) ui.TextInputState {
	return ui.TextInputState{Enabled: true, View: 4, Text: text, SelectionAnchor: len(text), SelectionCursor: len(text), CaretRect: caret}
}

func TestKeysGoStraightToTheAppWithoutATextField(t *testing.T) {
	e := newIMEInput(t, "us")
	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	if got := e.rec.takeKeys(); len(got) != 1 {
		t.Errorf("keys = %v, want the key at once", got)
	}
	if calls := e.im.Log(); len(calls) != 0 {
		t.Errorf("the input method had calls %v without a text field", calls)
	}
}

func TestTextFieldWaitsForKeyboardFocus(t *testing.T) {
	e := newIMEInput(t, "us")
	e.in.onFocus(e.w, &xiEnterEvent{detail: 0}, false) // the window loses the focus
	e.w.p.setTextInput(e.w, field("ab", ui.Rect{Left: 5, Top: 6, Right: 7, Bottom: 22}))
	e.poster.Idle(t)
	if calls := e.im.Log(); len(calls) != 0 {
		t.Fatalf("calls = %v before the window has the focus", calls)
	}
	if e.in.p.text.bridge.Active() {
		t.Error("active without the keyboard focus")
	}

	e.in.onFocus(e.w, &xiEnterEvent{detail: 0}, true)
	want := []string{`SetContent(0,false)`, `SetSurroundingText("ab",2,2)`, `SetCursorRect(105,56,2,16)`, `FocusIn()`}
	if got := e.im.WaitCalls(t, len(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}

	// The focus goes away: the input method is told, and the keys are plain.
	e.in.onFocus(e.w, &xiEnterEvent{detail: 0}, false)
	if got := e.im.WaitCalls(t, len(want)+1); got[len(want)] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut last", got)
	}
}

func TestKeyConsumedByTheInputMethodNeverReachesTheApp(t *testing.T) {
	e := newIMEInput(t, "us")
	e.im.Handled = func(k dbus.IMEKey) bool { return k.Keysym == 'a' }
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)

	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), true)  // consumed
	e.in.onKey(keyEvent(xE, 0, xiModifierState{}), true)  // passed on
	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), false) // its key-down never reached the app
	e.in.onKey(keyEvent(xE, 0, xiModifierState{}), false)
	if got := e.rec.takeKeys(); len(got) != 0 {
		t.Fatalf("keys = %v before the answers", got)
	}
	for i := 0; i < 4; i++ {
		e.poster.Run(t)
	}
	got := e.rec.takeKeys()
	if len(got) != 2 || got[0].Type != ui.KeyEventTypeDown || got[1].Type != ui.KeyEventTypeUp ||
		got[0].Logical != 'e' || got[1].Logical != 'e' || got[0].Character != "e" {
		t.Errorf("keys = %+v, want the down and the up of e", got)
	}
	calls := e.im.Log()[4:]
	want := []string{"ProcessKey(a,30,0,false)", "ProcessKey(e,18,0,false)", "ProcessKey(a,30,0,true)", "ProcessKey(e,18,0,true)"}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("key calls = %v, want %v", calls, want)
	}
}

func TestInputMethodSeesTheKeysymAndModifiers(t *testing.T) {
	e := newIMEInput(t, "us")
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	shift := xiModifierState{base: 1, effective: 1}
	e.in.onKey(keyEvent(xA, 0, shift), true)
	e.poster.Run(t)
	if got := e.im.Log()[4]; got != "ProcessKey(A,30,1,false)" {
		t.Errorf("call = %s, want the keysym A with the Shift bit", got)
	}
	if got := e.rec.takeKeys(); len(got) != 1 || got[0].Character != "A" {
		t.Errorf("keys = %+v", got)
	}
}

func TestRepeatOfAConsumedKeyGoesToTheInputMethodAgain(t *testing.T) {
	e := newIMEInput(t, "us")
	e.im.Handled = func(dbus.IMEKey) bool { return true }
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	e.in.onKey(keyEvent(xA, xiKeyRepeat, xiModifierState{}), true)
	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), false)
	for i := 0; i < 3; i++ {
		e.poster.Run(t)
	}
	if got := e.rec.takeKeys(); len(got) != 0 {
		t.Errorf("keys = %+v, want none", got)
	}
}

func TestEventsOfTheInputMethodReachTheHandler(t *testing.T) {
	e := newIMEInput(t, "us")
	e.w.p.setTextInput(e.w, field("ab", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	e.im.Events(dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ni", Cursor: 1})
	e.poster.Run(t)
	e.im.Events(dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "你"})
	e.poster.Run(t)
	want := []ui.TextInputEvent{
		{View: 4, Preedit: "ni", PreeditCursorBegin: 1, PreeditCursorEnd: 1},
		{View: 4, Commit: "你", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
	}
	if !reflect.DeepEqual(e.rec.texts, want) {
		t.Errorf("events = %+v, want %+v", e.rec.texts, want)
	}
}

func TestOtherWindowCannotTurnOffTheField(t *testing.T) {
	e := newIMEInput(t, "us")
	other := &window{p: e.in.p, id: 5}
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	e.w.p.setTextInput(other, ui.TextInputState{}) // the other window has no field
	e.poster.Idle(t)
	if !e.in.p.text.bridge.Active() {
		t.Error("another window turned the text input off")
	}
	e.w.p.setTextInput(e.w, ui.TextInputState{})
	if got := e.im.WaitCalls(t, 5); got[4] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut", got)
	}
	if e.in.p.text.bridge.Active() {
		t.Error("active after the field was turned off")
	}
}

func TestClosingTheWindowTurnsOffTheField(t *testing.T) {
	e := newIMEInput(t, "us")
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	e.in.p.text.windowClosed(e.w)
	if got := e.im.WaitCalls(t, 5); got[4] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut", got)
	}
	e.in.p.text.windowClosed(e.w) // a second time does nothing
}

func TestCaretFollowsTheWindow(t *testing.T) {
	e := newIMEInput(t, "us")
	e.w.p.setTextInput(e.w, field("", ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 12}))
	e.im.WaitCalls(t, 4)
	e.origin = [2]int{200, 80}
	e.in.p.textInputMoved(e.w)
	if got := e.im.WaitCalls(t, 5); got[4] != "SetCursorRect(201,82,2,10)" {
		t.Errorf("calls = %v, want the caret at the new origin", got)
	}
	e.in.p.textInputMoved(&window{}) // another window: nothing
	e.poster.Idle(t)
}

func TestAnswerAfterTheLayoutIsGoneIsDropped(t *testing.T) {
	e := newIMEInput(t, "us")
	e.w.p.setTextInput(e.w, field("", ui.Rect{}))
	e.im.WaitCalls(t, 4)
	e.in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	e.in.layout = nil // the platform is closing
	e.poster.Run(t)
	if got := e.rec.takeKeys(); len(got) != 0 {
		t.Errorf("keys = %+v, want none", got)
	}
}

func TestWithoutATextInputNothingHappens(t *testing.T) {
	in, _ := newInput(t, "us")
	w := &window{p: in.p, id: 1}
	in.p.setTextInput(w, field("", ui.Rect{})) // p.text is nil
	in.p.textInputMoved(w)
}
