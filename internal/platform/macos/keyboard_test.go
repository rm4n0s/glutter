package macos

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The key codes and the keys that the tests use (Flutter's FlutterEmbedderKeyResponderTest.mm).
const (
	codeKeyA       = 0x00
	codeKeyW       = 0x0d
	codeShiftLeft  = 0x38
	codeShiftRight = 0x3c
	codeAltRight   = 0x3d
	codeCapsLock   = 0x39
	codeNumpad1    = 0x53
	codeF1         = 0x7a

	physKeyA       = 0x70004
	physKeyW       = 0x7001a
	physShiftLeft  = 0x700e1
	physShiftRight = 0x700e5
	physAltRight   = 0x700e6
	physNumpad1    = 0x70059
	physF1         = 0x7003a
	logKeyA        = 0x61
	logKeyW        = 0x77
	logShiftLeft   = 0x200000102
	logShiftRight  = 0x200000103
	logAltRight    = 0x200000105
	logNumpad1     = 0x200000231
	logF1          = 0x100000801
)

// sentKey is an event that the responder gave to the framework.
type sentKey struct {
	ui.KeyData
}

// keyRecorder is the send function of a responder that records the events and answers handled.
type keyRecorder struct {
	events  []ui.KeyData
	handled bool
}

func newKeyRecorder() (*keyResponder, *keyRecorder) {
	rec := &keyRecorder{handled: true}
	r := newKeyResponder(func(d ui.KeyData) bool {
		rec.events = append(rec.events, d)
		return rec.handled
	})
	return r, rec
}

func (rec *keyRecorder) take() []ui.KeyData {
	out := rec.events
	rec.events = nil
	return out
}

func evKey(kind int, flags uint64, chars, ignoring string, repeat bool, code uint16) keyEvent {
	return keyEvent{kind: kind, keyCode: code, modifierFlags: flags, characters: chars, ignoringModifiers: ignoring, isARepeat: repeat}
}

// kd is the shorthand of an expected event.
func kd(t ui.KeyEventType, phys, log uint64, char string, synth bool) ui.KeyData {
	return ui.KeyData{Type: t, DeviceType: ui.KeyEventDeviceTypeKeyboard, Physical: phys, Logical: log, Character: char, Synthesized: synth}
}

// noTime clears the time stamps, which the tests that compare whole events do not set.
func noTime(in []ui.KeyData) []ui.KeyData {
	out := make([]ui.KeyData, len(in))
	for i, d := range in {
		d.TimeStamp = 0
		out[i] = d
	}
	return out
}

func TestKeyResponderEvents(t *testing.T) {
	const (
		down, up, rep      = ui.KeyEventTypeDown, ui.KeyEventTypeUp, ui.KeyEventTypeRepeat
		kDown, kUp, kFlags = keyKindDown, keyKindUp, keyKindFlagsChanged
	)
	type step struct {
		ev   keyEvent
		want []ui.KeyData
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"press, hold and release A on a US keyboard", []step{
			{evKey(kDown, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{kd(down, physKeyA, logKeyA, "a", false)}},
			{evKey(kDown, 0x100, "a", "a", true, codeKeyA), []ui.KeyData{kd(rep, physKeyA, logKeyA, "a", false)}},
			{evKey(kUp, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{kd(up, physKeyA, logKeyA, "", false)}},
		}},
		{"the logical key does not depend on the modifiers (option W makes a sum sign)", []step{
			{evKey(kDown, 0x80140, "∑", "w", false, codeKeyW), []ui.KeyData{
				kd(down, physAltRight, logAltRight, "", true),
				kd(down, physKeyW, logKeyW, "∑", false),
			}},
			{evKey(kUp, 0x80140, "∑", "w", false, codeKeyW), []ui.KeyData{kd(up, physKeyW, logKeyW, "", false)}},
		}},
		{"a key with more than one character has a key of the macOS plane", []step{
			{evKey(kDown, 0, "àn", "àn", false, codeKeyA), []ui.KeyData{kd(down, physKeyA, 0x01400000000, "àn", false)}},
		}},
		{"a down of a key that is down gives a synthesized up first", []step{
			{evKey(kDown, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{kd(down, physKeyA, logKeyA, "a", false)}},
			{evKey(kDown, 0x100, "à", "à", false, codeKeyA), []ui.KeyData{
				kd(up, physKeyA, logKeyA, "", true),
				kd(down, physKeyA, 0xe0, "à", false),
			}},
			{evKey(kUp, 0x100, "à", "à", false, codeKeyA), []ui.KeyData{kd(up, physKeyA, 0xe0, "", false)}},
		}},
		{"an up of a key that is not down is not sent", []step{
			{evKey(kUp, 0x100, "a", "a", false, codeKeyA), nil},
		}},
		{"a repeat of a key that was never down is a down", []step{
			{evKey(kDown, 0x100, "a", "a", true, codeKeyA), []ui.KeyData{kd(down, physKeyA, logKeyA, "a", false)}},
			{evKey(kUp, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{kd(up, physKeyA, logKeyA, "", false)}},
		}},
		{"a modifier toggled during the tap of a key: the repeat keeps the logical key of the down", []step{
			{evKey(kFlags, 0x20104, "", "", false, codeShiftRight), []ui.KeyData{kd(down, physShiftRight, logShiftRight, "", false)}},
			{evKey(kDown, 0x20104, "A", "A", false, codeKeyA), []ui.KeyData{kd(down, physKeyA, logKeyA, "A", false)}},
			{evKey(kDown, 0x20104, "A", "A", true, codeKeyA), []ui.KeyData{kd(rep, physKeyA, logKeyA, "A", false)}},
			{evKey(kFlags, 0x100, "", "", false, codeShiftRight), []ui.KeyData{kd(up, physShiftRight, logShiftRight, "", false)}},
			{evKey(kDown, 0x100, "a", "a", true, codeKeyA), []ui.KeyData{kd(rep, physKeyA, logKeyA, "a", false)}},
			{evKey(kUp, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{kd(up, physKeyA, logKeyA, "", false)}},
		}},
		{"the flags that mark the numpad and the function keys are not modifiers", []step{
			{evKey(kDown, 0x200100, "1", "1", false, codeNumpad1), []ui.KeyData{kd(down, physNumpad1, logNumpad1, "1", false)}},
			{evKey(kUp, 0x200100, "1", "1", false, codeNumpad1), []ui.KeyData{kd(up, physNumpad1, logNumpad1, "", false)}},
			{evKey(kDown, 0x800100, "", "", false, codeF1), []ui.KeyData{kd(down, physF1, logF1, "", false)}},
			{evKey(kUp, 0x800100, "", "", false, codeF1), []ui.KeyData{kd(up, physF1, logF1, "", false)}},
		}},
		{"the left and the right modifier keys are told apart", []step{
			{evKey(kFlags, 0x20102, "", "", false, codeShiftLeft), []ui.KeyData{kd(down, physShiftLeft, logShiftLeft, "", false)}},
			{evKey(kFlags, 0x20106, "", "", false, codeShiftRight), []ui.KeyData{kd(down, physShiftRight, logShiftRight, "", false)}},
			{evKey(kFlags, 0x20104, "", "", false, codeShiftLeft), []ui.KeyData{kd(up, physShiftLeft, logShiftLeft, "", false)}},
			{evKey(kFlags, 0x100, "", "", false, codeShiftRight), []ui.KeyData{kd(up, physShiftRight, logShiftRight, "", false)}},
			{evKey(kFlags, 0x80140, "", "", false, codeAltRight), []ui.KeyData{kd(down, physAltRight, logAltRight, "", false)}},
			{evKey(kFlags, 0x100, "", "", false, codeAltRight), []ui.KeyData{kd(up, physAltRight, logAltRight, "", false)}},
		}},
		{"a modifier event that changes nothing is not sent", []step{
			{evKey(kFlags, 0x100, "", "", false, codeShiftLeft), nil},
		}},
		{"a modifier that went down unseen is synthesized before the key", []step{
			{evKey(kDown, 0x20102, "A", "A", false, codeKeyA), []ui.KeyData{
				kd(down, physShiftLeft, logShiftLeft, "", true),
				kd(down, physKeyA, logKeyA, "A", false),
			}},
			{evKey(kUp, 0x100, "a", "a", false, codeKeyA), []ui.KeyData{
				kd(up, physShiftLeft, logShiftLeft, "", true),
				kd(up, physKeyA, logKeyA, "", false),
			}},
		}},
		{"a tap of CapsLock reported as a down is a down and a synthesized up", []step{
			{evKey(kFlags, 0x10100, "", "", false, codeCapsLock), []ui.KeyData{
				kd(down, 0x70039, 0x100000104, "", false),
				kd(up, 0x70039, 0x100000104, "", true),
			}},
			{evKey(kFlags, 0x100, "", "", false, codeCapsLock), []ui.KeyData{
				kd(down, 0x70039, 0x100000104, "", false),
				kd(up, 0x70039, 0x100000104, "", true),
			}},
		}},
		{"CapsLock that changed unseen is synthesized before a key", []step{
			{evKey(kDown, 0x10100, "A", "a", false, codeKeyA), []ui.KeyData{
				kd(down, 0x70039, 0x100000104, "", true),
				kd(up, 0x70039, 0x100000104, "", true),
				kd(down, physKeyA, logKeyA, "A", false),
			}},
		}},
		{"a CapsLock event that is not a change is not sent", []step{
			{evKey(kFlags, 0x100, "", "", false, codeCapsLock), nil},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, rec := newKeyRecorder()
			for i, s := range tt.steps {
				r.handle(s.ev)
				got := noTime(rec.take())
				if len(got) == 0 {
					got = nil
				}
				if !reflect.DeepEqual(got, s.want) {
					t.Fatalf("step %d: events = %+v, want %+v", i, got, s.want)
				}
			}
		})
	}
}

func TestKeyResponderEventsAreAnsweredByTheFramework(t *testing.T) {
	tests := []struct {
		name    string
		handled bool
		ev      keyEvent
		want    bool
	}{
		{"a key the framework handled", true, evKey(keyKindDown, 0x100, "a", "a", false, codeKeyA), true},
		{"a key the framework did not handle goes on to the text input", false, evKey(keyKindDown, 0x100, "a", "a", false, codeKeyA), false},
		{"an up of a key that was never down counts as handled", false, evKey(keyKindUp, 0x100, "a", "a", false, codeKeyA), true},
		{"a modifier flags event that changes nothing counts as handled", false, evKey(keyKindFlagsChanged, 0x100, "", "", false, codeShiftLeft), true},
		{"a CapsLock tap is answered with its down", false, evKey(keyKindFlagsChanged, 0x10100, "", "", false, codeCapsLock), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, rec := newKeyRecorder()
			rec.handled = tt.handled
			if got := r.handle(tt.ev); got != tt.want {
				t.Errorf("handle = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyResponderTimestamp(t *testing.T) {
	r, rec := newKeyRecorder()
	ev := evKey(keyKindFlagsChanged, 0x20104, "", "", false, codeShiftRight)
	ev.timestamp = 123 * time.Second
	r.handle(ev)
	if len(rec.events) != 1 || rec.events[0].TimeStamp != 123*time.Second {
		t.Fatalf("events = %+v, want one at 123 s", rec.events)
	}
}

func TestKeyResponderPressedKeys(t *testing.T) {
	r, _ := newKeyRecorder()
	r.handle(evKey(keyKindDown, 0x100, "a", "a", false, codeKeyA))
	r.handle(evKey(keyKindFlagsChanged, 0x20104, "", "", false, codeShiftRight))
	want := map[uint64]uint64{physKeyA: logKeyA, physShiftRight: logShiftRight}
	if got := r.pressedKeys(); !reflect.DeepEqual(got, want) {
		t.Errorf("pressedKeys = %v, want %v", got, want)
	}
	r.handle(evKey(keyKindUp, 0x20104, "a", "a", false, codeKeyA))
	delete(want, physKeyA)
	if got := r.pressedKeys(); !reflect.DeepEqual(got, want) {
		t.Errorf("after the up: pressedKeys = %v, want %v", got, want)
	}
}

func TestKeyResponderSyncModifiers(t *testing.T) {
	r, rec := newKeyRecorder()
	r.syncModifiers(0x20102|flagMetaLeft, time.Second)
	got := noTime(rec.take())
	want := []ui.KeyData{
		kd(ui.KeyEventTypeDown, physShiftLeft, logShiftLeft, "", true),
		kd(ui.KeyEventTypeDown, 0x700e3, 0x200000106, "", true),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	// The same flags again: nothing changed.
	r.syncModifiers(0x20102|flagMetaLeft, 2*time.Second)
	if len(rec.events) != 0 {
		t.Errorf("events = %+v, want none", rec.events)
	}
	r.syncModifiers(0x100, 3*time.Second)
	if got := noTime(rec.take()); len(got) != 2 || got[0].Type != ui.KeyEventTypeUp || got[1].Type != ui.KeyEventTypeUp {
		t.Errorf("events = %+v, want two ups", got)
	}
}

func TestKeyResponderReleaseAll(t *testing.T) {
	r, rec := newKeyRecorder()
	r.handle(evKey(keyKindFlagsChanged, 0x20104, "", "", false, codeShiftRight))
	r.handle(evKey(keyKindDown, 0x20104, "A", "A", false, codeKeyA))
	rec.take()
	r.releaseAll(5 * time.Second)
	got := rec.take()
	if len(got) != 2 || got[0].Physical != physKeyA || got[1].Physical != physShiftRight {
		t.Fatalf("events = %+v, want the ups of A and of ShiftRight, in the order of the keys", got)
	}
	for _, d := range got {
		if d.Type != ui.KeyEventTypeUp || !d.Synthesized || d.TimeStamp != 5*time.Second {
			t.Errorf("event %+v: want a synthesized up at 5 s", d)
		}
	}
	if len(r.pressedKeys()) != 0 {
		t.Errorf("pressedKeys = %v, want none", r.pressedKeys())
	}
	// The flags were forgotten too: the same flags are a change again.
	r.handle(evKey(keyKindDown, 0x20104, "A", "A", false, codeKeyA))
	if got := rec.take(); len(got) != 2 || !got[0].Synthesized || got[0].Physical != physShiftRight {
		t.Errorf("after the release: events = %+v, want the synthesized down of ShiftRight, then A", got)
	}
}

func TestKeyResponderUsesTheLayout(t *testing.T) {
	r, rec := newKeyRecorder()
	// A Russian layout: the key of the letter Q makes a Cyrillic letter, and the layout puts Q there.
	r.layout = map[uint16]uint64{0x0c: 0x71}
	r.handle(evKey(keyKindDown, 0x100, "й", "й", false, 0x0c))
	if len(rec.events) != 1 || rec.events[0].Logical != 0x71 || rec.events[0].Character != "й" {
		t.Fatalf("events = %+v, want logical 0x71 and the character й", rec.events)
	}
}

func TestLogicalKeyForEvent(t *testing.T) {
	tests := []struct {
		name string
		ev   keyEvent
		want uint64
	}{
		{"a key of the table does not look at the characters", keyEvent{keyCode: 0x24, ignoringModifiers: "\r"}, 0x10000000d},
		{"a letter is the lower case letter", keyEvent{keyCode: codeKeyA, ignoringModifiers: "A"}, 0x61},
		{"a Latin-1 letter is lower cased", keyEvent{keyCode: 0x27, ignoringModifiers: "À"}, 0xe0},
		// Flutter excludes the division sign (0xf7) from the range of the upper-case letters, where it never
		// is, and not the multiplication sign (0xd7), which is: the port does the same.
		{"the multiplication sign is lower cased as Flutter does", keyEvent{keyCode: 0x27, ignoringModifiers: "×"}, 0xf7},
		{"a character outside the BMP is one character", keyEvent{keyCode: 0x27, ignoringModifiers: "😀"}, 0x1f600},
		{"two characters make a key of the macOS plane", keyEvent{keyCode: 0x27, ignoringModifiers: "ab"}, 0x01400000027},
		{"no characters make a key of the macOS plane", keyEvent{keyCode: 0x27}, 0x01400000027},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logicalKeyForEvent(tt.ev); got != tt.want {
				t.Errorf("logicalKeyForEvent = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"nothing", "", ""},
		{"a letter", "a", "a"},
		{"a function key has no text", "", ""},
		{"the last of the function keys range", "", ""},
		{"the Apple logo is printable", "", ""},
		{"text that follows a function key's character is dropped with it", "x", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eventString(tt.in); got != tt.want {
				t.Errorf("eventString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct{ in, want uint64 }{
		{'A', 'a'}, {'Z', 'z'}, {'a', 'a'}, {'0', '0'}, {0xc0, 0xe0}, {0xde, 0xfe}, {0xd7, 0xf7}, {0xdf, 0xdf}, {0x410, 0x410},
	}
	for _, tt := range tests {
		if got := toLower(tt.in); got != tt.want {
			t.Errorf("toLower(%#x) = %#x, want %#x", tt.in, got, tt.want)
		}
	}
}

func TestModifierTables(t *testing.T) {
	if len(keyCodeToModifierFlag) != 8 || len(modifierFlagToKeyCode) != 8 {
		t.Fatalf("tables have %d and %d entries, want 8", len(keyCodeToModifierFlag), len(modifierFlagToKeyCode))
	}
	for code, flag := range keyCodeToModifierFlag {
		if modifierFlagToKeyCode[flag] != code {
			t.Errorf("key code %#x: its flag %#x leads to %#x", code, flag, modifierFlagToKeyCode[flag])
		}
	}
	if want := uint64(0x10000 | 0x1 | 0x2 | 0x4 | 0x8 | 0x10 | 0x20 | 0x40 | 0x2000); modifierFlagOfInterestMask != want {
		t.Errorf("mask = %#x, want %#x", modifierFlagOfInterestMask, want)
	}
	if keyCodeToPhysicalKey[codeCapsLock] != capsLockPhysicalKey || keyCodeToLogicalKey[codeCapsLock] != capsLockLogicalKey {
		t.Errorf("the table's CapsLock is %#x and %#x, want %#x and %#x",
			keyCodeToPhysicalKey[codeCapsLock], keyCodeToLogicalKey[codeCapsLock], capsLockPhysicalKey, capsLockLogicalKey)
	}
}

func TestUnknownKeyCodeIsOnTheMacOSPlane(t *testing.T) {
	if got := physicalKeyForCode(0x7f7f); got != macosPlane|0x7f7f {
		t.Errorf("physicalKeyForCode = %#x, want %#x", got, macosPlane|0x7f7f)
	}
	if got := logicalKeyForModifier(0x7f7f, 0x70001); got != macosPlane|0x70001 {
		t.Errorf("logicalKeyForModifier = %#x, want %#x", got, macosPlane|0x70001)
	}
}
