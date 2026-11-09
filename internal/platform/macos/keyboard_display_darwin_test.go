//go:build darwin && !ios && glutter_display

package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestKeyboardReadsTheLayoutOfTheSystem(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			k := s.p.kb
			if k.layoutData == 0 {
				t.Error("no layout data: the Unicode layout of the current input source was not read")
			}
			// Whatever the layout is, every letter and digit is on some key.
			have := map[uint64]bool{}
			for _, logical := range k.resp.layout {
				have[logical] = true
			}
			for _, g := range layoutGoals {
				if g.mandatory && !have[g.logical] {
					t.Errorf("no key makes the logical key %#x", g.logical)
				}
			}
			// On the keys of the US layout's letters the layout of the system makes its own characters.
			if c := k.clueFor(0x00, false); c.character == 0 {
				t.Errorf("the key A has no character in the layout of the system: %+v", c)
			}
		})
	})
}

func TestScriptedKeysMakeKeyEvents(t *testing.T) {
	h := &recorder{keyHandled: true}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takeKeys()
		s.onUI(func() { w.p.runDevKey(DevEvent{Kind: "key", Key: "KeyC", Mods: "meta"}) })
		got := h.takeKeys()
		type want struct {
			typ      ui.KeyEventType
			physical uint64
		}
		wants := []want{
			{ui.KeyEventTypeDown, 0x700e3}, // MetaLeft
			{ui.KeyEventTypeDown, 0x70006}, // KeyC
			{ui.KeyEventTypeUp, 0x70006},
			{ui.KeyEventTypeUp, 0x700e3},
		}
		if len(got) != len(wants) {
			t.Fatalf("%d events: %+v, want %d", len(got), got, len(wants))
		}
		for i, wt := range wants {
			if got[i].Type != wt.typ || got[i].Physical != wt.physical || got[i].Synthesized {
				t.Errorf("event %d = %+v, want a %v of %#x that is not synthesized", i, got[i], wt.typ, wt.physical)
			}
			if got[i].DeviceType != ui.KeyEventDeviceTypeKeyboard {
				t.Errorf("event %d has device type %v", i, got[i].DeviceType)
			}
		}
		if got[0].Logical != 0x200000106 {
			t.Errorf("MetaLeft's logical key = %#x, want 0x200000106", got[0].Logical)
		}
		if got[1].Character != "c" {
			// Command does not change the character of the key.
			t.Errorf("Command+C has the character %q, want c", got[1].Character)
		}
		s.onUI(func() { w.p.runDevKey(DevEvent{Kind: "key", Key: "ArrowLeft", Mods: "shift"}) })
		got = h.takeKeys()
		if len(got) != 4 || got[1].Physical != 0x70050 || got[1].Logical != 0x100000302 || got[1].Character != "" {
			t.Errorf("Shift+ArrowLeft = %+v, want ShiftLeft, ArrowLeft (no character), ArrowLeft up, ShiftLeft up", got)
		}
		// An uppercase letter has the character that Shift makes and the logical key of the lower case letter.
		s.onUI(func() { w.p.runDevKey(DevEvent{Kind: "key", Key: "KeyA", Mods: "shift"}) })
		got = h.takeKeys()
		if len(got) != 4 || got[1].Character != "A" {
			t.Errorf("Shift+A = %+v, want the character A", got)
		}
		s.onUI(w.Close)
	})
}

func TestKeyEventsAreSynchronousWhenTheHandlerCanAnswer(t *testing.T) {
	for _, handled := range []bool{true, false} {
		h := &recorder{keyHandled: handled}
		runApp(t, Options{}, h, func(s *session) {
			w := inputWindow(s, h)
			h.takeKeys()
			var answer bool
			s.onUI(func() {
				ev := w.makeKeyEvent(eventTypeKeyDown, 0x00, devBaseFlags, "a", "a")
				// What the view does with a key: the responder's answer is the framework's.
				answer = w.p.kb.resp.handle(w.keyEventOf(ev, keyKindDown))
			})
			if answer != handled {
				t.Errorf("handler answers %v, the responder says %v", handled, answer)
			}
			s.onUI(w.Close)
		})
	}
}

func TestPerformKeyEquivalentTakesTheKeyWhenTheViewIsFirstResponder(t *testing.T) {
	h := &recorder{keyHandled: true}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takeKeys()
		s.onUI(func() {
			ev := w.makeKeyEvent(eventTypeKeyDown, 0x08, devBaseFlags|flagMetaLeft|flagCommand, "c", "c")
			if !w.performKeyEquivalent(w.view, ev) {
				t.Error("the view did not take the key equivalent")
			}
		})
		if got := h.takeKeys(); len(got) == 0 {
			t.Error("the key equivalent made no key event")
		}
		// The event that is being redispatched is left to the menus.
		s.onUI(func() {
			ev := w.makeKeyEvent(eventTypeKeyDown, 0x08, devBaseFlags, "c", "c")
			w.p.kb.dispatching = ev
			defer func() { w.p.kb.dispatching = 0 }()
			w.performKeyEquivalent(w.view, ev)
		})
		if got := h.takeKeys(); len(got) != 0 {
			t.Errorf("a redispatched key equivalent made the events %+v", got)
		}
		s.onUI(w.Close)
	})
}

func TestKeysThatAreDownGoUpWhenTheWindowLosesTheKey(t *testing.T) {
	h := &recorder{keyHandled: true}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takeKeys()
		s.onUI(func() {
			w.devKeyEvent(eventTypeKeyDown, 0x00, devBaseFlags, "a", "a")
			w.keyChanged(false)
		})
		got := h.takeKeys()
		if len(got) != 2 || got[0].Type != ui.KeyEventTypeDown || got[1].Type != ui.KeyEventTypeUp || !got[1].Synthesized || got[1].Physical != got[0].Physical {
			t.Errorf("events = %+v, want the down of A and its synthesized up", got)
		}
		s.onUI(w.Close)
	})
}

func TestKeyUpsOfCommandKeysReachTheResponder(t *testing.T) {
	h := &recorder{keyHandled: true}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takeKeys()
		s.onUI(func() {
			flags := devBaseFlags | flagCommand | flagMetaLeft
			w.devKeyEvent(eventTypeFlagsChanged, 0x37, flags, "", "")
			w.devKeyEvent(eventTypeKeyDown, 0x08, flags, "c", "c")
			// AppKit does not send the key up of C to the view while Command is down: the monitor does.
			up := w.makeKeyEvent(eventTypeKeyUp, 0x08, flags, "c", "c")
			w.p.kb.monitoredKeyUp(up)
			// A key up without Command is the view's own.
			w.p.kb.monitoredKeyUp(w.makeKeyEvent(eventTypeKeyUp, 0x00, devBaseFlags, "a", "a"))
		})
		got := h.takeKeys()
		if len(got) != 3 || got[2].Type != ui.KeyEventTypeUp || got[2].Physical != 0x70006 {
			t.Errorf("events = %+v, want the Command down, the C down and the C up", got)
		}
		s.onUI(w.Close)
	})
}
