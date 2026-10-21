package web

import (
	"sort"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// These tests port keyboard_converter_test.dart of Flutter web (the cases of KeyboardConverter) to
// the plain Go KeyConverter.

// Modifier flags of the key helpers.
const (
	kAlt   = 0x1
	kCtrl  = 0x2
	kShift = 0x4
	kMeta  = 0x8
)

func keyDown(code, key string, mods, loc int) KeyEvent {
	return KeyEvent{Down: true, Code: code, Key: key, Location: loc, Alt: mods&kAlt != 0, Ctrl: mods&kCtrl != 0, Shift: mods&kShift != 0, Meta: mods&kMeta != 0}
}

func keyUp(code, key string, mods, loc int) KeyEvent {
	e := keyDown(code, key, mods, loc)
	e.Down = false
	return e
}

func keyRepeat(code, key string, mods, loc int) KeyEvent {
	e := keyDown(code, key, mods, loc)
	e.Repeat = true
	return e
}

func at(e KeyEvent, ms float64) KeyEvent {
	e.TimeStamp = ms
	return e
}

// fakeClock stands in for Dart's FakeAsync: the converter's delayed events run when time is elapsed.
type fakeClock struct {
	now    time.Duration
	timers []*fakeTimer
}

type fakeTimer struct {
	at   time.Duration
	fn   func()
	dead bool
}

func (c *fakeClock) after(d time.Duration, fn func()) func() {
	t := &fakeTimer{at: c.now + d, fn: fn}
	c.timers = append(c.timers, t)
	return func() { t.dead = true }
}

func (c *fakeClock) elapse(d time.Duration) {
	end := c.now + d
	for {
		var next *fakeTimer
		sort.SliceStable(c.timers, func(i, j int) bool { return c.timers[i].at < c.timers[j].at })
		for _, t := range c.timers {
			if !t.dead && t.at <= end {
				next = t
				break
			}
		}
		if next == nil {
			break
		}
		next.dead = true
		c.now = next.at
		next.fn()
	}
	c.now = end
}

// harness is a converter that records the key data it gives the framework.
type harness struct {
	conv  *KeyConverter
	clock *fakeClock
	data  []ui.KeyData
	// handle decides what the framework answers.
	handle func(ui.KeyData) bool
}

func newHarness(os OperatingSystem, handle func(ui.KeyData) bool) *harness {
	h := &harness{clock: &fakeClock{}, handle: handle}
	h.conv = NewKeyConverter(func(d ui.KeyData) bool {
		h.data = append(h.data, d)
		return h.handle(d)
	}, os, h.clock.after)
	return h
}

func always(ui.KeyData) bool { return true }

func onlyDown(d ui.KeyData) bool { return d.Type == ui.KeyEventTypeDown }

func (h *harness) last() ui.KeyData { return h.data[len(h.data)-1] }

func (h *harness) clear() { h.data = nil }

// want is the expected key data; the zero timestamp is not checked.
type want struct {
	ts          time.Duration
	typ         ui.KeyEventType
	physical    uint64
	logical     uint64
	character   string
	synthesized bool
}

func (h *harness) expect(t *testing.T, got ui.KeyData, w want) {
	t.Helper()
	if got.Type != w.typ || got.Physical != w.physical || got.Logical != w.logical || got.Character != w.character || got.Synthesized != w.synthesized {
		t.Errorf("got %v (synthesized %v), want type %v physical %#x logical %#x character %q synthesized %v",
			got, got.Synthesized, w.typ, w.physical, w.logical, w.character, w.synthesized)
	}
	if got.DeviceType != ui.KeyEventDeviceTypeKeyboard {
		t.Errorf("device type = %v, want keyboard", got.DeviceType)
	}
	if w.ts != 0 && got.TimeStamp != w.ts {
		t.Errorf("timestamp = %v, want %v", got.TimeStamp, w.ts)
	}
}

func phys(code string) uint64 { return webCodeToPhysicalKey[code] }

func loc(key string, location int) uint64 { return webKeyLocationLogicalKey[key][location] }

const (
	logA = 0x61
	logE = 0x65
	logL = 0x6c
	logU = 0x75
)

// step is an event, whether the converter's result must prevent the default action (when checked)
// and the key data expected.
type step struct {
	event   KeyEvent
	prevent *bool
	want    []want
}

func yes() *bool { y := true; return &y }
func no() *bool  { n := false; return &n }

func TestKeyConverterSequences(t *testing.T) {
	shiftL := func(typ ui.KeyEventType) want {
		return want{typ: typ, physical: phys("ShiftLeft"), logical: loc("Shift", locationLeft)}
	}
	keyA := func(typ ui.KeyEventType, char string) want {
		return want{typ: typ, physical: phys("KeyA"), logical: logA, character: char}
	}
	tests := []struct {
		name   string
		os     OperatingSystem
		handle func(ui.KeyData) bool
		steps  []step
	}{
		{
			name: "single key press, repeat and release", os: OSLinux, handle: onlyDown,
			steps: []step{
				{at(keyDown("KeyA", "a", 0, 0), 1), yes(), []want{{ts: time.Millisecond, typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"}}},
				{at(keyRepeat("KeyA", "a", 0, 0), 1.5), no(), []want{{ts: 1500 * time.Microsecond, typ: ui.KeyEventTypeRepeat, physical: phys("KeyA"), logical: logA, character: "a"}}},
				{at(keyRepeat("KeyA", "a", 0, 0), 1500), no(), []want{{ts: 1500 * time.Millisecond, typ: ui.KeyEventTypeRepeat, physical: phys("KeyA"), logical: logA, character: "a"}}},
				{at(keyUp("KeyA", "a", 0, 0), 2000.5), no(), []want{{ts: 2*time.Second + 500*time.Microsecond, typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA}}},
			},
		},
		{
			name: "special cases: AltGr of en-in on Windows gives the layout's letter", os: OSWindows, handle: onlyDown,
			steps: []step{
				{at(keyDown("KeyL", "l̥", kCtrl|kAlt, 0), 1), nil, []want{{ts: time.Millisecond, typ: ui.KeyEventTypeDown, physical: phys("KeyL"), logical: logL, character: "l̥"}}},
			},
		},
		{
			name: "release a modifier during a repeated sequence", os: OSLinux, handle: onlyDown,
			steps: []step{
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), yes(), []want{shiftL(ui.KeyEventTypeDown)}},
				{keyDown("KeyA", "A", kShift, 0), yes(), []want{keyA(ui.KeyEventTypeDown, "A")}},
				{keyRepeat("KeyA", "A", kShift, 0), no(), []want{keyA(ui.KeyEventTypeRepeat, "A")}},
				{keyUp("ShiftLeft", "Shift", 0, locationLeft), no(), []want{shiftL(ui.KeyEventTypeUp)}},
				{keyRepeat("KeyA", "a", 0, 0), no(), []want{keyA(ui.KeyEventTypeRepeat, "a")}},
				{keyRepeat("KeyA", "a", 0, 0), no(), []want{keyA(ui.KeyEventTypeRepeat, "a")}},
				{keyUp("KeyA", "a", 0, 0), no(), []want{keyA(ui.KeyEventTypeUp, "")}},
			},
		},
		{
			name: "distinguish between left and right modifiers", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), nil, []want{shiftL(ui.KeyEventTypeDown)}},
				{keyDown("ShiftRight", "Shift", kShift, locationRight), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("ShiftRight"), logical: loc("Shift", locationRight)}}},
				{keyUp("ShiftLeft", "Shift", kShift, locationLeft), nil, []want{shiftL(ui.KeyEventTypeUp)}},
				{keyUp("ShiftRight", "Shift", 0, locationRight), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("ShiftRight"), logical: loc("Shift", locationRight)}}},
			},
		},
		{
			name: "treat modifiers at standard locations as if at left", os: OSLinux, handle: always,
			steps: func() []step {
				var s []step
				for _, m := range []struct {
					key  string
					mods int
				}{{"Shift", kShift}, {"Control", kCtrl}, {"Alt", kAlt}, {"Meta", kMeta}} {
					logical := loc(m.key, locationLeft)
					s = append(s,
						step{keyDown("", m.key, m.mods, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: 0x1700000000, logical: logical}}},
						step{keyUp("", m.key, m.mods, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: 0x1700000000, logical: logical}}},
					)
				}
				return s
			}(),
		},
		{
			name: "distinguish between normal and numpad digits", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("Digit1", "1", 0, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("Digit1"), logical: 0x31, character: "1"}}},
				{keyDown("Numpad1", "1", 0, locationNumpad), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("Numpad1"), logical: loc("1", locationNumpad), character: "1"}}},
				{keyUp("Digit1", "1", 0, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("Digit1"), logical: 0x31}}},
				{keyUp("Numpad1", "1", 0, locationNumpad), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("Numpad1"), logical: loc("1", locationNumpad)}}},
			},
		},
		{
			name: "dead keys are distinguishable", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("AltLeft", "Alt", kAlt, locationLeft), nil, nil},
				{keyDown("KeyE", "Dead", kAlt, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("KeyE"), logical: 0x1740070008}}},
				{keyUp("KeyE", "Dead", kAlt, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("KeyE"), logical: 0x1740070008}}},
				{keyDown("KeyU", "Dead", kAlt, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("KeyU"), logical: 0x1740070018}}},
				{keyUp("KeyU", "Dead", kAlt, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("KeyU"), logical: 0x1740070018}}},
				{keyDown("ShiftLeft", "Shift", kAlt|kShift, locationLeft), nil, nil},
				{keyDown("KeyE", "Dead", kAlt|kShift, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("KeyE"), logical: 0x1760070008}}},
				{keyUp("AltLeft", "Alt", kShift, locationLeft), nil, nil},
				{keyUp("KeyE", "e", kShift, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("KeyE"), logical: 0x1760070008}}},
				{keyUp("ShiftLeft", "Shift", 0, locationLeft), nil, nil},
			},
		},
		{
			name: "duplicate down is preceded with a synthesized up", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), yes(), []want{shiftL(ui.KeyEventTypeDown)}},
				// The up of ShiftLeft is missed.
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), yes(), []want{
					{typ: ui.KeyEventTypeUp, physical: phys("ShiftLeft"), logical: loc("Shift", locationLeft), synthesized: true},
					shiftL(ui.KeyEventTypeDown),
				}},
				{keyUp("ShiftLeft", "Shift", 0, locationLeft), yes(), []want{shiftL(ui.KeyEventTypeUp)}},
			},
		},
		{
			name: "duplicate down is preceded with a synthesized up using the registered logical key", os: OSLinux, handle: always,
			steps: []step{
				// BracketLeft on a French keyboard.
				{keyDown("BracketLeft", "Dead", 0, 0), yes(), []want{{typ: ui.KeyEventTypeDown, physical: phys("BracketLeft"), logical: phys("BracketLeft") + 0x1700000000}}},
				// The up of BracketLeft is missed.
				{keyDown("BracketLeft", "Process", 0, 0), yes(), []want{
					{typ: ui.KeyEventTypeUp, physical: phys("BracketLeft"), logical: phys("BracketLeft") + 0x1700000000, synthesized: true},
					{typ: ui.KeyEventTypeDown, physical: phys("BracketLeft"), logical: webKeyToLogicalKey["Process"]},
				}},
			},
		},
		{
			name: "duplicate ups are skipped", os: OSLinux, handle: always,
			steps: []step{
				// The down of ShiftRight is missed because of a loss of focus: an empty key data, and
				// the browser's default is prevented.
				{keyUp("ShiftRight", "Shift", 0, locationRight), yes(), []want{{typ: ui.KeyEventTypeDown}}},
			},
		},
		{
			name: "conflict from multiple keyboards does not crash", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("KeyA", "a", 0, 0), nil, nil}, {keyDown("KeyA", "a", 0, 0), nil, nil},
				{keyUp("KeyA", "a", 0, 0), nil, nil}, {keyUp("KeyA", "a", 0, 0), nil, nil},
				{keyDown("KeyA", "a", 0, 0), nil, nil}, {keyDown("KeyA", "u", 0, 0), nil, nil},
				{keyUp("KeyA", "u", 0, 0), nil, nil}, {keyUp("KeyA", "a", 0, 0), nil, nil},
				// The states are reset after everything is released.
				{keyDown("KeyA", "a", 0, 0), nil, []want{keyA(ui.KeyEventTypeDown, "a")}},
				{keyDown("KeyU", "u", 0, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("KeyU"), logical: logU, character: "u"}}},
			},
		},
		{
			name: "CapsLock behaves normally on Linux", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("CapsLock", "CapsLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("CapsLock"), logical: webKeyToLogicalKey["CapsLock"]}}},
				{keyUp("CapsLock", "CapsLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("CapsLock"), logical: webKeyToLogicalKey["CapsLock"]}}},
			},
		},
		{
			name: "lock flags of other keys", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("ScrollLock", "ScrollLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("ScrollLock"), logical: webKeyToLogicalKey["ScrollLock"]}}},
				{keyUp("ScrollLock", "ScrollLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("ScrollLock"), logical: webKeyToLogicalKey["ScrollLock"]}}},
				{keyDown("ScrollLock", "ScrollLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("ScrollLock"), logical: webKeyToLogicalKey["ScrollLock"]}}},
				{keyUp("ScrollLock", "ScrollLock", 0, 0), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("ScrollLock"), logical: webKeyToLogicalKey["ScrollLock"]}}},
			},
		},
		{
			name: "deduce a modifier key up from the modifier field", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("ShiftRight", "Shift", kShift, locationRight), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("ShiftRight"), logical: loc("Shift", locationRight)}}},
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), nil, []want{shiftL(ui.KeyEventTypeDown)}},
				// The releases of the shift keys are omitted.
				{keyDown("KeyA", "a", 0, 0), nil, []want{
					{typ: ui.KeyEventTypeUp, physical: phys("ShiftLeft"), logical: loc("Shift", locationLeft), synthesized: true},
					{typ: ui.KeyEventTypeUp, physical: phys("ShiftRight"), logical: loc("Shift", locationRight), synthesized: true},
					keyA(ui.KeyEventTypeDown, "a"),
				}},
			},
		},
		{
			// Regression test for flutter/flutter#99297: on Linux Chrome, the MetaLeft down has
			// metaKey true and the up has it false, which the definition of metaKey forbids.
			name: "ignore the inconsistent modifier flag of the current modifier", os: OSLinux, handle: always,
			steps: []step{
				{keyDown("ShiftLeft", "Shift", kShift, locationLeft), nil, []want{shiftL(ui.KeyEventTypeDown)}},
				{keyDown("MetaLeft", "Meta", kShift, locationLeft), nil, []want{{typ: ui.KeyEventTypeDown, physical: phys("MetaLeft"), logical: loc("Meta", locationLeft)}}},
				{keyUp("MetaLeft", "Meta", kShift|kMeta, locationLeft), nil, []want{{typ: ui.KeyEventTypeUp, physical: phys("MetaLeft"), logical: loc("Meta", locationLeft)}}},
				{keyUp("ShiftLeft", "Shift", 0, locationLeft), nil, []want{shiftL(ui.KeyEventTypeUp)}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(tt.os, tt.handle)
			for i, s := range tt.steps {
				h.clear()
				prevent := h.conv.Handle(s.event)
				if s.prevent != nil && prevent != *s.prevent {
					t.Errorf("step %d: preventDefault = %v, want %v", i, prevent, *s.prevent)
				}
				if s.want == nil {
					continue
				}
				if len(h.data) != len(s.want) {
					t.Fatalf("step %d: %d key data %v, want %d", i, len(h.data), h.data, len(s.want))
				}
				for j, w := range s.want {
					h.expect(t, h.data[j], w)
				}
			}
		})
	}
}

// TestKeyConverterIgnoresEventsWithoutKeyOrCode: Chrome's autofill sends events with no key or code.
func TestKeyConverterIgnoresEventsWithoutKeyOrCode(t *testing.T) {
	h := newHarness(OSLinux, always)
	if h.conv.Handle(KeyEvent{Down: true, Incomplete: true}) || h.conv.Handle(KeyEvent{Incomplete: true}) {
		t.Error("an incomplete event prevented the default")
	}
	if len(h.data) != 0 {
		t.Errorf("an incomplete event gave key data: %v", h.data)
	}
}

// TestKeyConverterCapsLockOnDarwin: on macOS and iOS both the down and the up of CapsLock are a
// down followed by an immediate synthesized up.
func TestKeyConverterCapsLockOnDarwin(t *testing.T) {
	for _, os := range []OperatingSystem{OSMacOS, OSIOS} {
		t.Run(os.TargetPlatform().String(), func(t *testing.T) {
			h := newHarness(os, always)
			caps := func(typ ui.KeyEventType, synthesized bool) want {
				return want{typ: typ, physical: phys("CapsLock"), logical: webKeyToLogicalKey["CapsLock"], synthesized: synthesized}
			}
			for _, e := range []KeyEvent{keyDown("CapsLock", "CapsLock", 0, 0), keyUp("CapsLock", "CapsLock", 0, 0)} {
				h.clear()
				if !h.conv.Handle(e) {
					t.Error("the default was not prevented")
				}
				if len(h.data) != 1 {
					t.Fatalf("%d key data, want 1", len(h.data))
				}
				h.expect(t, h.last(), caps(ui.KeyEventTypeDown, false))
				h.clear()
				h.clock.elapse(time.Microsecond)
				if len(h.data) != 1 {
					t.Fatalf("%d key data after the delay, want 1", len(h.data))
				}
				h.expect(t, h.last(), caps(ui.KeyEventTypeUp, true))
			}
			// Another down works, and the delayed events are cancelled by Dispose.
			h.clear()
			h.conv.Handle(keyDown("CapsLock", "CapsLock", 0, 0))
			h.clear()
			h.conv.Dispose()
			h.clock.elapse(10 * time.Second)
			if len(h.data) != 0 {
				t.Errorf("a delayed event ran after Dispose: %v", h.data)
			}
		})
	}
}

// TestKeyConverterKeyGuards: on macOS and iOS the browser sends no keyup for the keys of a
// shortcut with the meta key, so a key that goes down while meta is down is released two seconds
// after its last repeat, unless the meta key is not involved.
func TestKeyConverterKeyGuards(t *testing.T) {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	t.Run("keys that go down while a meta key is down are guarded", func(t *testing.T) {
		for _, os := range []OperatingSystem{OSMacOS, OSIOS} {
			for _, meta := range []struct {
				code     string
				location int
			}{{"MetaLeft", locationLeft}, {"MetaRight", locationRight}} {
				h := newHarness(os, always)
				h.conv.Handle(at(keyDown(meta.code, "Meta", kMeta, meta.location), 100))
				h.clock.elapse(ms(100))
				h.conv.Handle(at(keyDown("KeyA", "a", kMeta, 0), 200))
				h.expect(t, h.last(), want{ts: ms(200), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
				h.clear()
				// The up of KeyA is omitted because of the shortcut.
				h.clock.elapse(ms(2500))
				h.expect(t, h.last(), want{ts: ms(2200), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA, synthesized: true})
				h.clear()
				h.conv.Handle(at(keyUp(meta.code, "Meta", 0, meta.location), 2700))
				h.expect(t, h.last(), want{ts: ms(2700), typ: ui.KeyEventTypeUp, physical: phys(meta.code), logical: loc("Meta", meta.location)})
				h.clock.elapse(ms(100))
				// The state of KeyA is cleared.
				h.conv.Handle(at(keyDown("KeyA", "a", 0, 0), 2800))
				h.expect(t, h.last(), want{ts: ms(2800), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
				h.clock.elapse(ms(100))
				h.conv.Handle(at(keyUp("KeyA", "a", 0, 0), 2900))
				h.expect(t, h.last(), want{ts: ms(2900), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA})
				h.conv.Dispose()
			}
		}
	})
	t.Run("keys that go down while no modifier is down are not guarded", func(t *testing.T) {
		for _, os := range []OperatingSystem{OSMacOS, OSIOS} {
			h := newHarness(os, always)
			h.conv.Handle(at(keyDown("KeyA", "a", 0, 0), 100))
			h.clock.elapse(ms(100))
			h.conv.Handle(at(keyDown("KeyE", "e", 0, 0), 200))
			h.expect(t, h.last(), want{ts: ms(200), typ: ui.KeyEventTypeDown, physical: phys("KeyE"), logical: logE, character: "e"})
			h.clock.elapse(ms(2500))
			if len(h.data) != 2 {
				t.Errorf("a key up was made up after the delay with no modifier involved: %v", h.data)
			}
			h.conv.Handle(at(keyUp("KeyE", "e", 0, 0), 2700))
			h.conv.Handle(at(keyUp("KeyA", "a", 0, 0), 2800))
			if len(h.data) != 4 || h.data[2].Type != ui.KeyEventTypeUp || h.data[3].Type != ui.KeyEventTypeUp {
				t.Errorf("the ups of the keys were not delivered: %v", h.data)
			}
		}
	})
	t.Run("keys that go down while another modifier is down are not guarded", func(t *testing.T) {
		for _, os := range []OperatingSystem{OSMacOS, OSIOS} {
			for _, m := range []struct {
				code, key string
				mask, loc int
			}{
				{"ShiftLeft", "Shift", kShift, locationLeft}, {"ShiftRight", "Shift", kShift, locationRight},
				{"ControlLeft", "Control", kCtrl, locationLeft}, {"ControlRight", "Control", kCtrl, locationRight},
				{"AltLeft", "Alt", kAlt, locationLeft}, {"AltRight", "Alt", kAlt, locationRight},
			} {
				h := newHarness(os, always)
				h.conv.Handle(at(keyDown(m.code, m.key, m.mask, m.loc), 100))
				h.clock.elapse(ms(100))
				h.expect(t, h.last(), want{ts: ms(100), typ: ui.KeyEventTypeDown, physical: phys(m.code), logical: loc(m.key, m.loc)})
				h.clear()
				h.conv.Handle(at(keyDown("KeyA", "a", m.mask, 0), 200))
				h.clock.elapse(ms(2500))
				if len(h.data) != 1 {
					t.Errorf("%s: a key up was made up: %v", m.code, h.data)
				}
				h.conv.Dispose()
			}
		}
	})
	t.Run("repeated down events refresh the guard", func(t *testing.T) {
		h := newHarness(OSMacOS, always)
		h.conv.Handle(at(keyDown("MetaLeft", "Meta", kMeta, locationLeft), 100))
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyDown("KeyA", "a", kMeta, 0), 200))
		h.clock.elapse(ms(400))
		h.conv.Handle(at(keyRepeat("KeyA", "a", kMeta, 0), 600))
		h.clock.elapse(ms(50))
		h.conv.Handle(at(keyRepeat("KeyA", "a", kMeta, 0), 650))
		h.clock.elapse(ms(50))
		h.conv.Handle(at(keyRepeat("KeyA", "a", kMeta, 0), 700))
		h.clock.elapse(ms(2000))
		h.expect(t, h.last(), want{ts: ms(2700), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA, synthesized: true})
		h.clear()
		h.conv.Handle(at(keyUp("MetaLeft", "Meta", 0, locationLeft), 3200))
		h.expect(t, h.last(), want{ts: ms(3200), typ: ui.KeyEventTypeUp, physical: phys("MetaLeft"), logical: loc("Meta", locationLeft)})
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyDown("KeyA", "a", 0, 0), 3300))
		h.expect(t, h.last(), want{ts: ms(3300), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyUp("KeyA", "a", 0, 0), 3400))
		h.expect(t, h.last(), want{ts: ms(3400), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA})
	})
	t.Run("guards are cleared by keyups", func(t *testing.T) {
		h := newHarness(OSMacOS, always)
		h.conv.Handle(at(keyDown("MetaLeft", "Meta", kMeta, locationLeft), 100))
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyDown("KeyA", "a", kCtrl, 0), 200))
		h.expect(t, h.last(), want{ts: ms(200), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
		h.clear()
		h.clock.elapse(ms(500))
		h.conv.Handle(at(keyUp("MetaLeft", "Meta", 0, locationLeft), 700))
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyUp("KeyA", "a", 0, 0), 800))
		h.expect(t, h.last(), want{ts: ms(800), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA})
		h.clear()
		h.clock.elapse(ms(2000))
		if len(h.data) != 0 {
			t.Errorf("a guard ran after the key went up: %v", h.data)
		}
		h.conv.Handle(at(keyDown("KeyA", "a", 0, 0), 2800))
		h.expect(t, h.last(), want{ts: ms(2800), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyUp("KeyA", "a", 0, 0), 2900))
		h.expect(t, h.last(), want{ts: ms(2900), typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA})
	})
	t.Run("keys are not guarded on Linux", func(t *testing.T) {
		h := newHarness(OSLinux, always)
		h.conv.Handle(at(keyDown("MetaLeft", "Meta", kMeta, locationLeft), 100))
		h.clock.elapse(ms(100))
		h.conv.Handle(at(keyDown("KeyA", "a", kMeta, 0), 200))
		h.expect(t, h.last(), want{ts: ms(200), typ: ui.KeyEventTypeDown, physical: phys("KeyA"), logical: logA, character: "a"})
		h.clear()
		h.clock.elapse(ms(2500))
		if len(h.data) != 0 {
			t.Errorf("a guard ran on Linux: %v", h.data)
		}
	})
}

// TestSynthesizeModifiersIfNeeded ports the tests of pointer_binding_test.dart that check how the
// pointer events keep the record of the modifier keys: a down of the left key when neither side is
// known to be down, an up of each side that is, nothing when they agree, and AltGraph.
func TestSynthesizeModifiersIfNeeded(t *testing.T) {
	mods := []struct {
		name string
		flag int
	}{{"Alt", kAlt}, {"Control", kCtrl}, {"Meta", kMeta}, {"Shift", kShift}}
	call := func(h *harness, flags int) {
		h.conv.SynthesizeModifiersIfNeeded(flags&kAlt != 0, flags&kCtrl != 0, flags&kMeta != 0, flags&kShift != 0, 5)
	}
	for _, m := range mods {
		left, right := phys(m.name+"Left"), phys(m.name+"Right")
		t.Run(m.name+" down is synthesized for the left key when neither is pressed", func(t *testing.T) {
			h := newHarness(OSLinux, always)
			call(h, m.flag)
			if len(h.data) != 1 {
				t.Fatalf("%d key data, want 1", len(h.data))
			}
			h.expect(t, h.last(), want{typ: ui.KeyEventTypeDown, physical: left, logical: loc(m.name, locationLeft), synthesized: true})
			if !h.conv.KeyIsPressed(left) {
				t.Error("the left key is not known to be pressed")
			}
		})
		for _, side := range []struct {
			name            string
			code            string
			location        int
			physical, other uint64
		}{{"left", m.name + "Left", locationLeft, left, right}, {"right", m.name + "Right", locationRight, right, left}} {
			t.Run(m.name+" down is not synthesized when the "+side.name+" key is pressed", func(t *testing.T) {
				h := newHarness(OSLinux, always)
				h.conv.Handle(keyDown(side.code, m.name, m.flag, side.location))
				h.clear()
				call(h, m.flag)
				if len(h.data) != 0 {
					t.Errorf("key data was made up: %v", h.data)
				}
			})
			t.Run(m.name+" up is synthesized for the "+side.name+" key when it is pressed", func(t *testing.T) {
				h := newHarness(OSLinux, always)
				h.conv.Handle(keyDown(side.code, m.name, m.flag, side.location))
				h.clear()
				call(h, 0)
				if len(h.data) != 1 {
					t.Fatalf("%d key data, want 1", len(h.data))
				}
				h.expect(t, h.last(), want{typ: ui.KeyEventTypeUp, physical: side.physical, logical: loc(m.name, side.location), synthesized: true})
				if h.conv.KeyIsPressed(side.physical) {
					t.Error("the key is still known to be pressed")
				}
			})
		}
		t.Run(m.name+" up is not synthesized when neither is pressed", func(t *testing.T) {
			h := newHarness(OSLinux, always)
			call(h, 0)
			if len(h.data) != 0 {
				t.Errorf("key data was made up: %v", h.data)
			}
		})
	}
	t.Run("an up is synthesized for AltGraph", func(t *testing.T) {
		h := newHarness(OSLinux, always)
		h.conv.Handle(keyDown("AltRight", "AltGraph", 0, 0))
		if !h.conv.KeyIsPressed(phys("AltRight")) {
			t.Fatal("AltRight is not known to be pressed")
		}
		h.clear()
		call(h, 0)
		if len(h.data) != 1 {
			t.Fatalf("%d key data, want 1", len(h.data))
		}
		h.expect(t, h.last(), want{typ: ui.KeyEventTypeUp, physical: phys("AltRight"), logical: loc("AltGraph", 0), synthesized: true})
		if h.conv.KeyIsPressed(phys("AltRight")) {
			t.Error("AltRight is still known to be pressed")
		}
	})
}

// TestKeyConverterReleaseAll: a loss of the focus releases the keys that are down with synthesized
// ups, in the order they went down.
func TestKeyConverterReleaseAll(t *testing.T) {
	h := newHarness(OSMacOS, always)
	h.conv.Handle(keyDown("ShiftLeft", "Shift", kShift, locationLeft))
	h.conv.Handle(keyDown("KeyA", "A", kShift, 0))
	h.clear()
	h.conv.ReleaseAll(9)
	if len(h.data) != 2 {
		t.Fatalf("%d key data, want 2", len(h.data))
	}
	h.expect(t, h.data[0], want{typ: ui.KeyEventTypeUp, physical: phys("ShiftLeft"), logical: loc("Shift", locationLeft), synthesized: true})
	h.expect(t, h.data[1], want{typ: ui.KeyEventTypeUp, physical: phys("KeyA"), logical: logA, synthesized: true})
	if h.conv.KeyIsPressed(phys("KeyA")) || h.conv.KeyIsPressed(phys("ShiftLeft")) {
		t.Error("keys are still known to be pressed")
	}
}

// TestCharacterLogicalKey: the logical key of a key that is a character is its lower case; on a
// layout whose key is not a Latin letter or digit, a mandatory layout goal gives its US key
// (Deviation: web_locale_keymap is not part of the sources).
func TestCharacterLogicalKey(t *testing.T) {
	tests := []struct {
		name      string
		code, key string
		want      uint64
	}{
		{"a letter", "KeyA", "a", 'a'},
		{"a capital letter", "KeyA", "A", 'a'},
		{"AZERTY: the key of Digit1 is &, the goal is the digit", "Digit1", "&", '1'},
		{"AZERTY: the shifted digit", "Digit1", "1", '1'},
		{"Cyrillic: the key of KeyA is the goal", "KeyA", "ф", 'a'},
		{"a punctuation key without a mandatory goal keeps the layout's character", "Comma", "ю", 'ю'},
		{"a Latin letter with a ring", "KeyL", "l̥", 'l'},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := characterLogicalKey(tt.code, tt.key); got != tt.want {
				t.Errorf("characterLogicalKey(%q, %q) = %#x, want %#x", tt.code, tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyIsName(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{{"Shift", true}, {"ArrowLeft", true}, {"a", false}, {"S", false}, {"ж", false}, {"l̥", false}, {"", false}, {"Dead", true}}
	for _, tt := range tests {
		if got := keyIsName(tt.key); got != tt.want {
			t.Errorf("keyIsName(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestPhysicalCodeOfUnknownKeys(t *testing.T) {
	if got := physicalCode(""); got != webKeyIDPlane {
		t.Errorf("empty code = %#x, want the plane", got)
	}
	a, b := physicalCode("NoSuchKey"), physicalCode("NoSuchKey")
	if a != b || a < webKeyIDPlane || a == physicalCode("OtherKey") {
		t.Errorf("made-up physical keys are not stable and distinct: %#x %#x", a, physicalCode("OtherKey"))
	}
}

func TestDurationOf(t *testing.T) {
	tests := []struct {
		ms   float64
		want time.Duration
	}{{0, 0}, {1, time.Millisecond}, {1.5, 1500 * time.Microsecond}, {2000.5, 2*time.Second + 500*time.Microsecond}}
	for _, tt := range tests {
		if got := durationOf(tt.ms); got != tt.want {
			t.Errorf("durationOf(%v) = %v, want %v", tt.ms, got, tt.want)
		}
	}
}
