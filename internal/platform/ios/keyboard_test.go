package ios

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// recorder collects what the responder sends.
type recorder struct {
	events  []ui.KeyData
	handled bool
}

func (r *recorder) send(d ui.KeyData) bool { r.events = append(r.events, d); return r.handled }

func newRecorded(handled bool) (*keyResponder, *recorder) {
	rec := &recorder{handled: handled}
	return newKeyResponder(rec.send), rec
}

const (
	hidA          uint32 = 0x04
	hidEnter      uint32 = 0x28
	hidArrowUp    uint32 = 0x52
	hidF1         uint32 = 0x3a
	hidLeftShift  uint32 = 0xe1
	hidRightShift uint32 = 0xe5
)

func TestKeyDownAndUp(t *testing.T) {
	r, rec := newRecorded(true)
	down := pressEvent{phase: pressBegan, keyCode: hidA, characters: "a", ignoring: "a", timestamp: time.Second}
	if !r.handle(down) {
		t.Error("the framework handled the key but the press was not handled")
	}
	if len(rec.events) != 1 {
		t.Fatalf("%d events, want 1", len(rec.events))
	}
	e := rec.events[0]
	if e.Type != ui.KeyEventTypeDown || e.Physical != 0x70004 || e.Logical != 0x61 || e.Character != "a" || e.Synthesized || e.DeviceType != ui.KeyEventDeviceTypeKeyboard || e.TimeStamp != time.Second {
		t.Errorf("down = %+v", e)
	}
	// A second press of the held key is nothing new.
	rec.events = nil
	if !r.handle(down) || len(rec.events) != 0 {
		t.Errorf("a press of a held key gave %d events", len(rec.events))
	}
	up := down
	up.phase = pressEnded
	r.handle(up)
	if len(rec.events) != 1 || rec.events[0].Type != ui.KeyEventTypeUp || rec.events[0].Logical != 0x61 || rec.events[0].Character != "" {
		t.Errorf("up = %+v", rec.events)
	}
	// An up of a key that was never down is handled and sends nothing.
	rec.events = nil
	if !r.handle(up) || len(rec.events) != 0 {
		t.Errorf("an up of a key that was not down gave %d events", len(rec.events))
	}
}

func TestLogicalKeys(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   pressEvent
		want uint64
		char string
	}{
		{"a letter is the lower case of its character", pressEvent{keyCode: hidA, characters: "A", ignoring: "A"}, 0x61, "A"},
		{"a Latin-1 capital is lowered", pressEvent{keyCode: 0x34, characters: "É", ignoring: "É"}, 0xe9, "É"},
		{"Enter is in the table", pressEvent{keyCode: hidEnter, characters: "\r", ignoring: "\r"}, 0x10000000d, "\r"},
		{"an arrow is a special key", pressEvent{keyCode: hidArrowUp, characters: "UIKeyInputUpArrow", ignoring: "UIKeyInputUpArrow"}, 0x100000304, ""},
		{"a function key has no characters", pressEvent{keyCode: hidF1, characters: "x", ignoring: "x"}, 0x100000801, ""},
		{"a key with no character is in the iOS plane", pressEvent{keyCode: 0x01, characters: "", ignoring: ""}, 0x1300000001, ""},
		{"a control character is not a label", pressEvent{keyCode: 0x2c, characters: "\x01", ignoring: "\x01"}, 0x130000002c, "\x01"},
		{"an emoji is its two UTF-16 units", pressEvent{keyCode: 0x2d, characters: "😀", ignoring: "😀"}, 0xd83dde00, "😀"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, rec := newRecorded(false)
			tc.ev.phase = pressBegan
			r.handle(tc.ev)
			if len(rec.events) == 0 {
				t.Fatal("no event")
			}
			e := rec.events[len(rec.events)-1]
			if e.Logical != tc.want || e.Character != tc.char {
				t.Errorf("logical 0x%x character %q, want 0x%x %q", e.Logical, e.Character, tc.want, tc.char)
			}
		})
	}
}

func TestModifierSynchronization(t *testing.T) {
	r, rec := newRecorded(true)
	// The left Shift goes down: its own press, and the flags say Shift.
	r.handle(pressEvent{phase: pressBegan, keyCode: hidLeftShift, modifierFlags: flagShiftAny})
	if len(rec.events) != 1 || rec.events[0].Physical != 0x700e1 || rec.events[0].Synthesized {
		t.Fatalf("shift down = %+v", rec.events)
	}
	// A letter with Shift still down: nothing to synchronize.
	rec.events = nil
	r.handle(pressEvent{phase: pressBegan, keyCode: hidA, characters: "A", ignoring: "a", modifierFlags: flagShiftAny})
	if len(rec.events) != 1 || rec.events[0].Logical != 0x61 || rec.events[0].Character != "A" {
		t.Fatalf("letter = %+v", rec.events)
	}
	// The user released Shift while another view had the keyboard: the next press brings a synthesized Shift up before the letter.
	rec.events = nil
	r.handle(pressEvent{phase: pressBegan, keyCode: 0x05, characters: "b", ignoring: "b", modifierFlags: 0})
	if len(rec.events) != 2 || rec.events[0].Type != ui.KeyEventTypeUp || rec.events[0].Physical != 0x700e1 || !rec.events[0].Synthesized || rec.events[1].Logical != 0x62 {
		t.Fatalf("after the missed release = %+v", rec.events)
	}
	// A CapsLock that came on without a press is a synthesized tap.
	rec.events = nil
	r.handle(pressEvent{phase: pressBegan, keyCode: 0x06, characters: "c", ignoring: "c", modifierFlags: flagCapsLock})
	if len(rec.events) != 3 || rec.events[0].Physical != capsLockPhysicalKey || rec.events[0].Type != ui.KeyEventTypeDown || rec.events[1].Type != ui.KeyEventTypeUp {
		t.Fatalf("caps lock = %+v", rec.events)
	}
}

func TestFixSidedFlags(t *testing.T) {
	// Right Shift goes down while the left one is down: both sided flags stay, and the any flag.
	got := fixSidedFlags(flagShiftAny, flagShiftLeft, flagShiftRight, keyLeftShift, keyRightShift, keyRightShift, true, flagShiftAny|flagShiftLeft)
	if got != flagShiftAny|flagShiftLeft|flagShiftRight {
		t.Errorf("both down = 0x%x", got)
	}
	// The left one goes up while the right one is down: the any flag stays.
	got = fixSidedFlags(flagShiftAny, flagShiftLeft, flagShiftRight, keyLeftShift, keyRightShift, keyLeftShift, false, flagShiftAny|flagShiftLeft|flagShiftRight)
	if got != flagShiftAny|flagShiftRight {
		t.Errorf("left up = 0x%x", got)
	}
	// The right one goes up too: no flag is left.
	got = fixSidedFlags(flagShiftAny, flagShiftLeft, flagShiftRight, keyLeftShift, keyRightShift, keyRightShift, false, flagShiftAny|flagShiftRight)
	if got != 0 {
		t.Errorf("both up = 0x%x", got)
	}
}

func TestReleaseAll(t *testing.T) {
	r, rec := newRecorded(true)
	r.handle(pressEvent{phase: pressBegan, keyCode: hidA, characters: "a", ignoring: "a"})
	r.handle(pressEvent{phase: pressBegan, keyCode: 0x05, characters: "b", ignoring: "b"})
	rec.events = nil
	if len(r.pressedKeys()) != 2 {
		t.Fatalf("%d keys pressed, want 2", len(r.pressedKeys()))
	}
	r.releaseAll(time.Minute)
	if len(rec.events) != 2 || rec.events[0].Physical != 0x70004 || rec.events[1].Physical != 0x70005 || !rec.events[0].Synthesized || rec.events[0].Type != ui.KeyEventTypeUp {
		t.Errorf("releases = %+v", rec.events)
	}
	if len(r.pressedKeys()) != 0 {
		t.Error("keys are still pressed")
	}
}
