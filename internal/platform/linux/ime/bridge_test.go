//go:build linux

package ime

import (
	"context"
	stdErrors "errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/linux/ime/imetest"
	"github.com/rm4n0s/glutter/ui"
)

type harness struct {
	t       *testing.T
	p       *imetest.Poster
	im      *imetest.IM
	b       *Bridge
	opens   int
	openErr error
	got     []ui.TextInputEvent
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, p: imetest.NewPoster(), im: &imetest.IM{Fail: map[string]error{}}}
	h.b = New(h.p, func(_ context.Context, events func(dbus.IMEEvent)) (dbus.InputMethod, error) {
		h.opens++
		if h.openErr != nil {
			return nil, h.openErr
		}
		return h.im.Open(context.Background(), events)
	}, func(ev ui.TextInputEvent) { h.got = append(h.got, ev) })
	t.Cleanup(h.b.Close)
	return h
}

func (h *harness) waitCalls(n int) []string { h.t.Helper(); return h.im.WaitCalls(h.t, n) }

func enabled(text string, anchor, cursor int) ui.TextInputState {
	return ui.TextInputState{Enabled: true, View: 3, Text: text, SelectionAnchor: anchor, SelectionCursor: cursor}
}

func TestStateIsSentOnceAndFocusComesLast(t *testing.T) {
	h := newHarness(t)
	if h.b.Active() {
		t.Fatal("active before a text field has the focus")
	}
	st := enabled("héllo", 3, 3)
	st.Purpose = ui.TextInputPurposeEmail
	h.b.SetState(st, Rect{10, 20, 2, 16})
	if !h.b.Active() {
		t.Fatal("not active with a text field")
	}
	want := []string{`SetContent(6,false)`, `SetSurroundingText("héllo",2,2)`, `SetCursorRect(10,20,2,16)`, `FocusIn()`}
	if got := h.waitCalls(len(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}

	// The same state again sends nothing; a moved caret sends the rectangle.
	h.b.SetState(st, Rect{10, 20, 2, 16})
	h.b.SetState(st, Rect{30, 20, 2, 16})
	want = append(want, `SetCursorRect(30,20,2,16)`)
	if got := h.waitCalls(len(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}

	// New text and a new selection, then the field loses the focus.
	h.b.SetState(enabled("héllo!", 6, 6), Rect{30, 20, 2, 16})
	h.b.SetState(ui.TextInputState{}, Rect{})
	want = append(want, `SetContent(0,false)`, `SetSurroundingText("héllo!",5,5)`, `FocusOut()`)
	if got := h.waitCalls(len(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}
	if h.b.Active() {
		t.Error("active after the field lost the focus")
	}
	if h.opens != 1 {
		t.Errorf("%d connections, want 1", h.opens)
	}
}

func TestContentFollowsThePurposeAndTheHiddenHint(t *testing.T) {
	tests := []struct {
		name  string
		state func(*ui.TextInputState)
		want  string
	}{
		{"password", func(s *ui.TextInputState) { s.Purpose = ui.TextInputPurposePassword }, "SetContent(8,false)"},
		{"pin", func(s *ui.TextInputState) { s.Purpose = ui.TextInputPurposePIN }, "SetContent(9,false)"},
		{"hidden text", func(s *ui.TextInputState) { s.Hints = ui.TextInputHintHiddenText }, "SetContent(0,true)"},
		{"date is free-form", func(s *ui.TextInputState) { s.Purpose = ui.TextInputPurposeDate }, "SetContent(0,false)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			st := enabled("", 0, 0)
			tt.state(&st)
			h.b.SetState(st, Rect{})
			if got := h.waitCalls(1)[0]; got != tt.want {
				t.Errorf("first call = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestKeysGoThroughTheInputMethodInOrder(t *testing.T) {
	h := newHarness(t)
	h.im.Handled = func(k dbus.IMEKey) bool { return k.Keysym == 'n' } // consumes n only
	h.b.SetState(enabled("", 0, 0), Rect{})

	var results []string
	for _, k := range []dbus.IMEKey{
		{Keysym: 'n', Keycode: 49},
		{Keysym: 'x', Keycode: 45, State: 4},
		{Keysym: 'n', Keycode: 49, Release: true},
	} {
		key := k
		h.b.ProcessKey(key, func(handled bool) {
			results = append(results, fmt.Sprintf("%c:%v", rune(key.Keysym), handled))
		})
	}
	// The state job posts nothing; the three keys post their answers in order.
	for i := 0; i < 3; i++ {
		h.p.Run(t)
	}
	if want := []string{"n:true", "x:false", "n:true"}; !reflect.DeepEqual(results, want) {
		t.Errorf("results = %v, want %v", results, want)
	}
	calls := h.im.Log()
	wantKeys := []string{"ProcessKey(n,49,0,false)", "ProcessKey(x,45,4,false)", "ProcessKey(n,49,0,true)"}
	if got := calls[len(calls)-3:]; !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("key calls = %v, want %v", got, wantKeys)
	}
}

func TestKeyThatFailsOrTimesOutIsNotConsumed(t *testing.T) {
	h := newHarness(t)
	h.im.Fail["ProcessKey"] = stdErrors.New("bus gone")
	h.b.SetState(enabled("", 0, 0), Rect{})
	var handled []bool
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'a'}, func(v bool) { handled = append(handled, v) })
	h.p.Run(t)
	if !reflect.DeepEqual(handled, []bool{false}) {
		t.Errorf("handled = %v", handled)
	}
}

func TestKeyWithoutConnectionIsNotConsumed(t *testing.T) {
	h := newHarness(t)
	h.openErr = stdErrors.New("nobody home")
	h.b.SetState(enabled("", 0, 0), Rect{})
	var handled []bool
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'a'}, func(v bool) { handled = append(handled, v) })
	h.p.Run(t) // the answer for the key
	h.p.Run(t) // the bridge learns there is no input method
	if !reflect.DeepEqual(handled, []bool{false}) {
		t.Errorf("handled = %v", handled)
	}
	if h.b.Active() {
		t.Error("active without an input method")
	}
	// Later keys are answered at once, without a new attempt.
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'b'}, func(v bool) { handled = append(handled, v) })
	h.p.Run(t)
	if !reflect.DeepEqual(handled, []bool{false, false}) || h.opens != 1 {
		t.Errorf("handled = %v after %d connections", handled, h.opens)
	}
}

func TestEventsBecomeTextInputEvents(t *testing.T) {
	h := newHarness(t)
	h.b.SetState(enabled("héllo wörld", 6, 6), Rect{})
	h.waitCalls(4)
	send := func(ev dbus.IMEEvent) {
		h.im.Events(ev)
		h.p.Run(t)
	}

	send(dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "にほん", Cursor: 6})
	send(dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "ab", Cursor: -1})
	send(dbus.IMEEvent{Kind: dbus.IMEEventPreedit})
	send(dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "日本"})
	// The cursor is after "héllo" (6 bytes): 2 characters before it are "lo",
	// and 1 after it is " ".
	send(dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -2, Count: 3})
	send(dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -1, Count: 1})
	send(dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: 0, Count: 2})

	want := []ui.TextInputEvent{
		{View: 3, Preedit: "にほん", PreeditCursorBegin: 6, PreeditCursorEnd: 6},
		{View: 3, Preedit: "ab", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
		{View: 3, PreeditCursorBegin: -1, PreeditCursorEnd: -1},
		{View: 3, Commit: "日本", PreeditCursorBegin: -1, PreeditCursorEnd: -1},
		{View: 3, DeleteBefore: 2, DeleteAfter: 1, PreeditCursorBegin: -1, PreeditCursorEnd: -1},
		{View: 3, DeleteBefore: 1, PreeditCursorBegin: -1, PreeditCursorEnd: -1},
		{View: 3, DeleteAfter: 2, PreeditCursorBegin: -1, PreeditCursorEnd: -1}, // " w"
	}
	if !reflect.DeepEqual(h.got, want) {
		for i := range want {
			if i >= len(h.got) || !reflect.DeepEqual(h.got[i], want[i]) {
				t.Errorf("event %d = %+v, want %+v", i, h.got[min(i, len(h.got)-1)], want[i])
			}
		}
	}
}

func TestDeleteOutsideTheTextIsDropped(t *testing.T) {
	h := newHarness(t)
	h.b.SetState(enabled("ab", 1, 1), Rect{})
	h.waitCalls(4)
	for _, ev := range []dbus.IMEEvent{
		{Kind: dbus.IMEEventDelete, Offset: -3, Count: 1}, // a gap before the cursor
		{Kind: dbus.IMEEventDelete, Offset: 2, Count: 1},  // a gap after it
		{Kind: dbus.IMEEventDelete, Offset: -1, Count: 0}, // nothing
		{Kind: dbus.IMEEventKind(99)},                     // not a kind
	} {
		h.im.Events(ev)
		h.p.Run(t)
	}
	if len(h.got) != 0 {
		t.Errorf("events = %+v, want none", h.got)
	}
	// A range longer than the text is cut to it.
	h.im.Events(dbus.IMEEvent{Kind: dbus.IMEEventDelete, Offset: -5, Count: 6})
	h.p.Run(t)
	if len(h.got) != 1 || h.got[0].DeleteBefore != 1 || h.got[0].DeleteAfter != 1 {
		t.Errorf("events = %+v, want 1 byte before and 1 after", h.got)
	}
}

func TestEventsWithoutAFocusedFieldAreDropped(t *testing.T) {
	h := newHarness(t)
	h.b.SetState(enabled("", 0, 0), Rect{})
	h.waitCalls(4)
	h.b.SetState(ui.TextInputState{}, Rect{})
	h.waitCalls(5)
	h.im.Events(dbus.IMEEvent{Kind: dbus.IMEEventCommit, Text: "late"})
	h.p.Run(t)
	if len(h.got) != 0 {
		t.Errorf("events = %+v, want none", h.got)
	}
}

func TestMovingTheCaretEndsTheComposition(t *testing.T) {
	h := newHarness(t)
	h.b.SetState(enabled("abc", 3, 3), Rect{})
	h.waitCalls(4)
	h.im.Events(dbus.IMEEvent{Kind: dbus.IMEEventPreedit, Text: "x", Cursor: 1})
	h.p.Run(t)

	// A change that comes from the input method does not end it.
	st := enabled("abcx", 4, 4)
	st.CausedByInputMethod = true
	h.b.SetState(st, Rect{})
	// A change by the user does.
	h.b.SetState(enabled("abcx", 1, 1), Rect{})
	// With no composition, a change does not reset.
	h.b.SetState(enabled("abcx", 2, 2), Rect{})
	calls := h.waitCalls(8)
	resets := 0
	for _, c := range calls {
		if c == "Reset()" {
			resets++
		}
	}
	if resets != 1 {
		t.Errorf("%d resets in %v, want 1", resets, calls)
	}
}

func TestCloseDestroysTheContextAndAnswersPendingKeys(t *testing.T) {
	h := newHarness(t)
	h.im.Block = make(chan struct{})
	h.b.SetState(enabled("", 0, 0), Rect{})
	h.waitCalls(4)
	var handled []bool
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'a'}, func(v bool) { handled = append(handled, v) })
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'b'}, func(v bool) { handled = append(handled, v) })
	time.Sleep(20 * time.Millisecond) // the first key is waiting in the input method
	h.b.Close()
	h.b.Close() // twice is fine
	if h.im.Log()[len(h.im.Log())-1] != "Close()" {
		t.Errorf("calls = %v, want Close last", h.im.Log())
	}
	for len(handled) < 2 {
		h.p.Run(t)
	}
	if !reflect.DeepEqual(handled, []bool{false, false}) {
		t.Errorf("handled = %v", handled)
	}
	// After the close, new work does nothing.
	h.b.SetState(enabled("x", 0, 0), Rect{})
	h.b.ProcessKey(dbus.IMEKey{Keysym: 'c'}, func(v bool) { handled = append(handled, v) })
	h.p.Run(t)
	if len(handled) != 3 || handled[2] {
		t.Errorf("handled = %v", handled)
	}
}

func TestCloseBeforeAnyUse(t *testing.T) {
	h := newHarness(t)
	h.b.Close()
	if h.opens != 0 {
		t.Errorf("%d connections for a bridge that was never used", h.opens)
	}
}

func TestFailedCallsDoNotStopTheBridge(t *testing.T) {
	h := newHarness(t)
	h.im.Fail["SetCursorRect"] = stdErrors.New("gone")
	h.im.Fail["Reset"] = stdErrors.New("gone")
	h.b.SetState(enabled("a", 1, 1), Rect{1, 2, 3, 4})
	h.waitCalls(4)
	if h.im.Log()[3] != "FocusIn()" {
		t.Errorf("calls = %v, want the focus after the failed rectangle", h.im.Log())
	}
}

func TestRuneIndexAndBytes(t *testing.T) {
	if got := runeIndex("héllo", 3); got != 2 {
		t.Errorf("runeIndex = %d, want 2", got)
	}
	if got, want := runeIndex("ab", -1), 0; got != want {
		t.Errorf("runeIndex(-1) = %d", got)
	}
	if got, want := runeIndex("ab", 9), 2; got != want {
		t.Errorf("runeIndex(9) = %d", got)
	}
	if got := bytesBefore("aö", 1); got != 2 {
		t.Errorf("bytesBefore = %d, want 2", got)
	}
	if got := bytesAfter("öa", 5); got != 3 {
		t.Errorf("bytesAfter = %d, want 3", got)
	}
}
