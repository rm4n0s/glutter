//go:build linux && !android

package wayland

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/linux/ime/imetest"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// startDBUSIME starts a platform on a compositor without text-input-v3 whose
// input method over D-Bus is a fake one. The window has the keyboard focus.
func startDBUSIME(t *testing.T) (*keyEnv, *imetest.IM) {
	t.Helper()
	t.Setenv("LC_ALL", "en_US.UTF-8")
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: keymapFor(t, "us")})
	if e.p.text != nil || e.p.dbusIME == nil {
		t.Fatalf("text = %v, dbusIME = %v: the fallback is expected without the manager", e.p.text, e.p.dbusIME)
	}
	im := &imetest.IM{}
	e.onLoop(func() bool {
		e.p.dbusIME.close()
		e.p.dbusIME = newDBUSIME(e.p, e.p.loop, im.Open)
		return true
	})
	e.srv.KeyboardEnter(e.surface)
	e.settle()
	return e, im
}

func (e *keyEnv) setField(s ui.TextInputState) {
	e.t.Helper()
	e.onLoop(func() bool { e.p.windows[ui.ImplicitViewID].SetTextInput(s); return true })
}

// flush waits for the answers of the input method to be handled.
func (e *keyEnv) flush() {
	e.t.Helper()
	for i := 0; i < 3; i++ {
		e.settle()
		time.Sleep(5 * time.Millisecond)
	}
}

func fieldState() ui.TextInputState {
	return ui.TextInputState{
		Enabled: true, Text: "ab", SelectionAnchor: 2, SelectionCursor: 2,
		CaretRect: ui.Rect{Left: 10.2, Top: 20, Right: 11.4, Bottom: 34.5},
	}
}

func TestTextInputOverDBUSOnlyWithoutTheManager(t *testing.T) {
	with := startKeyboardNoFocus(t, fakecomp.Config{Globals: fakecomp.TextInputGlobals()})
	if with.p.text == nil || with.p.dbusIME != nil {
		t.Errorf("text = %v, dbusIME = %v with the manager", with.p.text, with.p.dbusIME)
	}
	without := startKeyboardNoFocus(t, fakecomp.Config{})
	if without.p.text != nil || without.p.dbusIME == nil {
		t.Errorf("text = %v, dbusIME = %v without the manager", without.p.text, without.p.dbusIME)
	}
}

func TestDBUSIMEGetsTheStateWhileTheWindowHasTheFocus(t *testing.T) {
	e, im := startDBUSIME(t)
	e.srv.KeyboardLeave(e.surface)
	e.settle()
	e.setField(fieldState())
	e.flush()
	if calls := im.Log(); len(calls) != 0 {
		t.Fatalf("calls = %v before the window has the focus", calls)
	}

	e.srv.KeyboardEnter(e.surface)
	want := []string{`SetContent(0,false)`, `SetSurroundingText("ab",2,2)`, `SetCursorRect(10,20,1,14)`, `FocusIn()`}
	if got := im.WaitCalls(t, len(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}

	e.srv.KeyboardLeave(e.surface)
	if got := im.WaitCalls(t, len(want)+1); got[len(want)] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut last", got)
	}
}

func TestDBUSIMEConsumesKeys(t *testing.T) {
	e, im := startDBUSIME(t)
	im.Handled = func(k dbus.IMEKey) bool { return k.Keysym == 'a' }
	e.setField(fieldState())
	im.WaitCalls(t, 4)

	e.srv.KeyboardKey(keyA, true) // consumed
	e.srv.KeyboardKey(keyE, true) // passed on
	e.srv.KeyboardKey(keyA, false)
	e.srv.KeyboardKey(keyE, false)
	waitUntil(t, func() bool { e.settle(); return len(e.h.seen()) >= 2 }, "the key of the app never arrived")
	e.flush()
	checkEvents(t, e.events(),
		want{down, 0x70008, 'e', "e", false},
		want{up, 0x70008, 'e', "", false})
	got := im.Log()[4:]
	wantCalls := []string{"ProcessKey(a,30,0,false)", "ProcessKey(e,18,0,false)", "ProcessKey(a,30,0,true)", "ProcessKey(e,18,0,true)"}
	if !reflect.DeepEqual(got, wantCalls) {
		t.Errorf("key calls = %v, want %v", got, wantCalls)
	}
}

func TestDBUSIMERepeatsGoThroughTheInputMethod(t *testing.T) {
	e, im := startDBUSIME(t)
	e.setField(fieldState())
	im.WaitCalls(t, 4)
	e.srv.KeyboardKey(keyE, true)
	waitUntil(t, func() bool { e.settle(); return len(e.h.seen()) >= 1 }, "no key-down")
	e.advance(700 * time.Millisecond) // past the delay: the first repeat
	waitUntil(t, func() bool { e.settle(); return len(e.h.seen()) >= 2 }, "no repeat")
	if seen := e.h.seen(); seen[1].Type != ui.KeyEventTypeRepeat {
		t.Errorf("second event = %+v, want a repeat", seen[1])
	}
	calls := im.Log()[4:]
	if len(calls) < 2 || calls[0] != "ProcessKey(e,18,0,false)" || calls[1] != "ProcessKey(e,18,0,false)" {
		t.Errorf("key calls = %v, want the key and its repeat", calls)
	}
}

func TestDBUSIMEEventsReachTheHandler(t *testing.T) {
	e, im := startDBUSIME(t)
	e.setField(fieldState())
	im.WaitCalls(t, 4)
	im.Events(dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "你好"})
	waitUntil(t, func() bool { e.settle(); return len(e.h.textEvents()) >= 1 }, "no text event")
	want := []ui.TextInputEvent{{Commit: "你好", PreeditCursorBegin: -1, PreeditCursorEnd: -1}}
	if got := e.h.textEvents(); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

func TestDBUSIMEFieldOfAnotherWindowAndClose(t *testing.T) {
	e, im := startDBUSIME(t)
	e.setField(fieldState())
	im.WaitCalls(t, 4)
	e.onLoop(func() bool {
		other := &window{p: e.p, id: 77}
		e.p.dbusIME.set(other, ui.TextInputState{}) // not the window of the field
		return true
	})
	e.flush()
	if got := len(im.Log()); got != 4 {
		t.Errorf("calls = %v, want no more", im.Log())
	}
	e.setField(ui.TextInputState{})
	if got := im.WaitCalls(t, 5); got[4] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut", got)
	}

	// Closing the window of a field turns it off. The text and the caret are the
	// same as before, so only the content and the focus are sent again.
	e.setField(fieldState())
	im.WaitCalls(t, 7)
	e.onLoop(func() bool { e.p.windows[ui.ImplicitViewID].Close(); return true })
	if got := im.WaitCalls(t, 8); got[7] != "FocusOut()" {
		t.Errorf("calls = %v, want FocusOut after the window closed", got)
	}
}
