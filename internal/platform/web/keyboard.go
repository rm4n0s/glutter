package web

import (
	"hash/fnv"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/ui"
)

// The locations of KeyboardEvent.location.
const (
	locationStandard = 0
	locationLeft     = 1
	locationRight    = 2
	locationNumpad   = 3
)

// webKeyIDPlane is the plane of the logical and physical key ids that Flutter web makes up
// for a key it has no id for.
const webKeyIDPlane uint64 = 0x1700000000

// The bits that mark a dead key's logical key with the modifiers that were down, because
// the key is told apart by its physical key and its modifiers.
const (
	deadKeyCtrl  uint64 = 0x10000000
	deadKeyShift uint64 = 0x20000000
	deadKeyAlt   uint64 = 0x40000000
	deadKeyMeta  uint64 = 0x80000000
)

// keyGuardDuration is how long a key that went down while a meta key was down on macOS or
// iOS waits for its repeat before it is taken to be released (the browser sends no keyup
// for the shortcuts of the system).
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._kKeydownCancelDurationMac
const keyGuardDuration = 2000 * time.Millisecond

// webLayoutGoal is a key whose logical key should not depend on the layout, so that shortcuts
// such as Control+C work on every layout (Flutter's layout_goals.json), by KeyboardEvent.code.
type webLayoutGoal struct {
	// logical is the logical key on the US layout.
	logical uint64
	// mandatory goals fall back to logical on a layout whose key is not a Latin letter or
	// digit; the others keep the layout's own character.
	mandatory bool
}

// KeyEvent is what the converter reads from a DOM KeyboardEvent. The platform copies the
// fields out of the event, so that the conversion is plain Go.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart FlutterHtmlKeyboardEvent
type KeyEvent struct {
	// Down is true for keydown and false for keyup.
	Down bool
	// Code is KeyboardEvent.code, and Key is KeyboardEvent.key.
	Code, Key string
	// Incomplete is true when the event has no key or no code, as the events of Chrome's
	// autofill: the converter ignores it.
	Incomplete bool
	// Repeat is KeyboardEvent.repeat.
	Repeat bool
	// Location is KeyboardEvent.location: 0 standard, 1 left, 2 right, 3 numpad.
	Location int
	// TimeStamp is KeyboardEvent.timeStamp, in milliseconds.
	TimeStamp float64
	// Alt, Ctrl, Shift and Meta are the modifier flags of the event.
	Alt, Ctrl, Shift, Meta bool
}

// KeyScheduler runs fn after d on the thread of the converter, and returns the function that
// cancels it.
//
// Flutter: no counterpart (dart:async Future.delayed)
type KeyScheduler func(d time.Duration, fn func()) (cancel func())

// KeyConverter turns the keyboard events of the browser into the key data of the framework:
// the physical key from KeyboardEvent.code, the logical key from KeyboardEvent.key through the
// generated tables, the type (down, repeat or up) from what is known to be pressed, and the
// events it synthesizes to keep that record true. It must be used on one goroutine.
//
// Deviation: Flutter asks the web_locale_keymap package for the logical key of the keys that
// are characters; this package is not part of the Flutter sources the port reads (reference/),
// so the keys that are characters get the lower case of the character, or, on a layout whose
// key is not a Latin letter or digit, the key of the layout goal (SPEC §23.4, §12.10). Keys
// Flutter makes up an id for (a hash of the name) use a stable FNV hash, not Dart's hashCode.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter
type KeyConverter struct {
	perform DispatchKeyData
	after   KeyScheduler
	darwin  bool
	// dispatch is perform wrapped with the tracking of Handle; nil outside Handle.
	dispatch DispatchKeyData
	disposed bool
	// pressing maps the pressed physical keys to their logical keys, in the order they went
	// down.
	pressing []pressedKey
	guards   map[uint64]func()
}

type pressedKey struct{ physical, logical uint64 }

// DispatchKeyData gives key data to the framework and reports whether it handled it.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart DispatchKeyData
type DispatchKeyData func(ui.KeyData) bool

// NewKeyConverter makes a converter that gives its key data to perform. The operating system
// decides how CapsLock and the key guards work (macOS and iOS); after runs the converter's
// delayed events and may be nil where there are none.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.KeyboardConverter
func NewKeyConverter(perform DispatchKeyData, os OperatingSystem, after KeyScheduler) *KeyConverter {
	if after == nil {
		after = func(time.Duration, func()) func() { return func() {} }
	}
	return &KeyConverter{
		perform: perform, after: after, darwin: os == OSMacOS || os == OSIOS,
		guards: map[uint64]func(){},
	}
}

// modifierKeys are the modifier keys the converter keeps true, in the order they are
// checked, with the modifier flag of an event that says whether they are down.
var modifierKeys = [...]struct {
	name     string
	location int
	flag     func(KeyEvent) bool
}{
	{"Alt", locationLeft, func(e KeyEvent) bool { return e.Alt }},
	{"Alt", locationRight, func(e KeyEvent) bool { return e.Alt }},
	{"Control", locationLeft, func(e KeyEvent) bool { return e.Ctrl }},
	{"Control", locationRight, func(e KeyEvent) bool { return e.Ctrl }},
	{"Shift", locationLeft, func(e KeyEvent) bool { return e.Shift }},
	{"Shift", locationRight, func(e KeyEvent) bool { return e.Shift }},
	{"Meta", locationLeft, func(e KeyEvent) bool { return e.Meta }},
	{"Meta", locationRight, func(e KeyEvent) bool { return e.Meta }},
}

func locationKey(name string, location int) uint64 { return webKeyLocationLogicalKey[name][location] }

func physicalOf(code string) uint64 { return webCodeToPhysicalKey[code] }

// webHash is the stable hash that stands in for Dart's String.hashCode in the ids of keys
// that are made up: 30 bits, as Dart's small integers.
func webHash(s string) uint64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s)) // a hash.Hash never fails to write
	return uint64(h.Sum32() & 0x3fffffff)
}

// physicalCode is the physical key of KeyboardEvent.code: the USB usage, or an id made up
// for a code the tables do not have (and the plane itself for an empty code).
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._getPhysicalCode
func physicalCode(code string) uint64 {
	if code == "" {
		return webKeyIDPlane
	}
	if p, ok := webCodeToPhysicalKey[code]; ok {
		return p
	}
	return webHash(code) + webKeyIDPlane
}

// keyIsName reports whether KeyboardEvent.key is the name of a key ("Shift"), not a
// character ("S" or "ж"): a name has more than one UTF-16 code unit and its first two are
// ASCII. (A character can have more than one code unit: en-in maps KeyL to l with a ring.)
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._eventKeyIsKeyName
func keyIsName(key string) bool {
	u := utf16.Encode([]rune(key))
	return len(u) > 1 && u[0] < 0x7f && u[1] < 0x7f
}

func isASCIILetterOrDigit(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

// characterLogicalKey is the logical key of a key that is a character: the lower case of the
// character; where the layout puts something else than a Latin letter or digit on a key that
// has a mandatory layout goal, the goal's key.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart (web_locale_keymap LocaleKeymap.getLogicalKey)
func characterLogicalKey(code, key string) uint64 {
	var r rune
	for _, c := range key {
		r = c
		break
	}
	if goal, ok := webLayoutGoals[code]; ok && goal.mandatory && !isASCIILetterOrDigit(r) {
		return goal.logical
	}
	return uint64(unicode.ToLower(r))
}

// durationOf converts a timestamp in milliseconds to a duration by splitting it into whole
// milliseconds and microseconds.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart _eventTimeStampToDuration
func durationOf(ms float64) time.Duration {
	whole := int64(ms)
	micro := int64((ms - float64(whole)) * 1000)
	return time.Duration(whole)*time.Millisecond + time.Duration(micro)*time.Microsecond
}

func modifierMask(e KeyEvent) uint64 {
	var m uint64
	if e.Alt {
		m += deadKeyAlt
	}
	if e.Ctrl {
		m += deadKeyCtrl
	}
	if e.Shift {
		m += deadKeyShift
	}
	if e.Meta {
		m += deadKeyMeta
	}
	return m
}

// logicalKey is the logical key of an event: a mapped key (ArrowLeft, Escape), a key with
// locations (Shift, the numpad), a key that is a character, a dead key, or one made up.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._handleEvent
func logicalKey(e KeyEvent, physical uint64, isCharacter bool) uint64 {
	if id, ok := webKeyToLogicalKey[e.Key]; ok {
		return id
	}
	if ids, ok := webKeyLocationLogicalKey[e.Key]; ok {
		loc := e.Location
		if loc < 0 || loc >= len(ids) || ids[loc] == 0 {
			loc = locationStandard
		}
		if ids[loc] != 0 {
			return ids[loc]
		}
	}
	if isCharacter {
		return characterLogicalKey(e.Code, e.Key)
	}
	if e.Key == "Dead" {
		return physical + modifierMask(e) + webKeyIDPlane
	}
	return webHash(e.Key) + webKeyIDPlane
}

// Handle converts a keydown or keyup event, gives the key data to the framework, and reports
// whether the event's default action must be prevented: the framework handled the key, or the
// event was an up of a key that was not down. Key data that updates the state of the
// converter (a synthesized up of a modifier the event says is up) is given first, and what
// the framework answers to it is dropped; exactly one key data is made from the event itself.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.handleEvent
func (c *KeyConverter) Handle(e KeyEvent) (preventDefault bool) {
	// Autofill on Chrome sends keyboard events whose key and code are missing.
	if e.Incomplete {
		return false
	}
	sentAny := false
	c.dispatch = func(d ui.KeyData) bool {
		sentAny = true
		return c.perform(d)
	}
	defer func() { c.dispatch = nil }()
	preventDefault = c.handle(e)
	if !sentAny {
		c.dispatch(ui.KeyData{Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard})
	}
	return preventDefault
}

func (c *KeyConverter) handle(e KeyEvent) (preventDefault bool) {
	ts := durationOf(e.TimeStamp)
	physical := physicalCode(e.Code)
	isCharacter := !keyIsName(e.Key)
	var cached uint64
	var haveCached bool
	logical := func() uint64 {
		if !haveCached {
			cached, haveCached = logicalKey(e, physical, isCharacter), true
		}
		return cached
	}

	capsLockOnDarwin := c.darwin && e.Code == "CapsLock"
	isPhysicalDown := e.Down || capsLockOnDarwin
	var typ ui.KeyEventType
	switch {
	case capsLockOnDarwin:
		// On macOS both the keydown and the keyup of CapsLock are a key down for Flutter,
		// followed by an immediate synthesized up.
		l := logical()
		c.schedule(0, func() { c.remove(physical) }, ui.KeyData{
			TimeStamp: ts, Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard,
			Physical: physical, Logical: l, Synthesized: true,
		})
		typ = ui.KeyEventTypeDown
	case isPhysicalDown:
		if last, pressed := c.record(physical); pressed {
			if e.Repeat {
				typ = ui.KeyEventTypeRepeat
			} else {
				// A key that is down went down again without repeating: another keyboard
				// pressed the same physical key, or the up was lost with the focus, or the
				// up of a system shortcut was skipped. The framework gets an up first.
				c.dispatch(ui.KeyData{
					TimeStamp: ts, Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard,
					Physical: physical, Logical: last, Synthesized: true,
				})
				c.remove(physical)
				typ = ui.KeyEventTypeDown
			}
		} else {
			typ = ui.KeyEventTypeDown
		}
	default:
		if _, pressed := c.record(physical); !pressed {
			// The key was released before: more than one keyboard pressed the same key.
			return true
		}
		typ = ui.KeyEventTypeUp
	}

	last, hadLast := c.record(physical)
	var next uint64
	hasNext := true
	switch typ {
	case ui.KeyEventTypeDown:
		next = logical()
	case ui.KeyEventTypeUp:
		hasNext = false
	default: // repeat
		next = last
	}
	if hasNext {
		c.set(physical, next)
	} else {
		c.remove(physical)
	}

	// Synchronize the modifiers: a modifier that is down in the records but not in the event's
	// flags is released (a down cannot be made up: the physical key is not known). The key of
	// the event itself is the ground truth.
	for _, m := range modifierKeys {
		testee := locationKey(m.name, m.location)
		if logical() == testee {
			continue
		}
		if m.flag(e) {
			continue
		}
		kept := c.pressing[:0:0]
		for _, p := range c.pressing {
			if p.logical != testee {
				kept = append(kept, p)
				continue
			}
			c.dispatch(ui.KeyData{
				TimeStamp: ts, Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard,
				Physical: p.physical, Logical: testee, Synthesized: true,
			})
		}
		c.pressing = kept
	}

	if isCharacter {
		if hasNext {
			c.startGuard(physical, logical(), ts)
		} else {
			c.stopGuard(physical)
		}
	}

	character := ""
	if isCharacter && typ != ui.KeyEventTypeUp {
		character = e.Key
	}
	use := last
	if !hadLast {
		use = logical()
	}
	handled := c.dispatch(ui.KeyData{
		TimeStamp: ts, Type: typ, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: physical, Logical: use, Character: character,
	})
	return handled
}

// schedule runs update and then gives the framework the key data, after d, unless the
// converter was disposed or the returned cancel ran. The framework's answer is dropped.
func (c *KeyConverter) schedule(d time.Duration, update func(), data ui.KeyData) (cancel func()) {
	canceled := false
	stop := c.after(d, func() {
		if canceled || c.disposed {
			return
		}
		update()
		c.perform(data)
	})
	return func() {
		canceled = true
		stop()
	}
}

// startGuard guards a key that went down or repeated while a meta key is down on macOS or iOS:
// the browser sends no keyup for the keys of a system shortcut, so without a repeat within
// keyGuardDuration the key is taken to be released.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._startGuardingKey
func (c *KeyConverter) startGuard(physical, logical uint64, now time.Duration) {
	if !c.darwin || !c.metaPressed() {
		return
	}
	cancel := c.schedule(keyGuardDuration, func() { c.remove(physical) }, ui.KeyData{
		TimeStamp: now + keyGuardDuration, Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: physical, Logical: logical, Synthesized: true,
	})
	if old, ok := c.guards[physical]; ok {
		old()
	}
	c.guards[physical] = cancel
}

// stopGuard ends the guard of a key that went up.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter._stopGuardingKey
func (c *KeyConverter) stopGuard(physical uint64) {
	if cancel, ok := c.guards[physical]; ok {
		delete(c.guards, physical)
		cancel()
	}
}

func (c *KeyConverter) metaPressed() bool {
	return c.KeyIsPressed(physicalOf("MetaLeft")) || c.KeyIsPressed(physicalOf("MetaRight"))
}

func (c *KeyConverter) record(physical uint64) (logical uint64, ok bool) {
	for _, p := range c.pressing {
		if p.physical == physical {
			return p.logical, true
		}
	}
	return 0, false
}

func (c *KeyConverter) set(physical, logical uint64) {
	for i, p := range c.pressing {
		if p.physical == physical {
			c.pressing[i].logical = logical
			return
		}
	}
	c.pressing = append(c.pressing, pressedKey{physical, logical})
}

func (c *KeyConverter) remove(physical uint64) {
	for i, p := range c.pressing {
		if p.physical == physical {
			c.pressing = append(c.pressing[:i], c.pressing[i+1:]...)
			return
		}
	}
}

// SynthesizeModifiersIfNeeded gives the framework the modifier key events that make its
// record of the modifiers agree with the flags of a pointer event (or any event that has them),
// only where they differ: a down of the left key when neither side is known to be down, an up
// of each side that is. It is called before the pointer events are converted.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.synthesizeModifiersIfNeeded
func (c *KeyConverter) SynthesizeModifiersIfNeeded(alt, ctrl, meta, shift bool, timeStamp float64) {
	c.synthesizeModifier("Alt", alt, timeStamp)
	c.synthesizeModifier("Control", ctrl, timeStamp)
	c.synthesizeModifier("Meta", meta, timeStamp)
	c.synthesizeModifier("Shift", shift, timeStamp)
}

func (c *KeyConverter) synthesizeModifier(name string, pressed bool, timeStamp float64) {
	physicalLeft, physicalRight := physicalOf(name+"Left"), physicalOf(name+"Right")
	leftLogical, leftPressed := c.record(physicalLeft)
	rightLogical, rightPressed := c.record(physicalRight)
	already := leftPressed || rightPressed
	if pressed && !already {
		c.synthesizeKey(timeStamp, ui.KeyEventTypeDown, physicalLeft, locationKey(name, locationLeft))
		return
	}
	if !pressed && leftPressed {
		c.synthesizeKey(timeStamp, ui.KeyEventTypeUp, physicalLeft, leftLogical)
	}
	if !pressed && rightPressed {
		c.synthesizeKey(timeStamp, ui.KeyEventTypeUp, physicalRight, rightLogical)
	}
}

func (c *KeyConverter) synthesizeKey(timeStamp float64, typ ui.KeyEventType, physical, logical uint64) {
	c.perform(ui.KeyData{
		TimeStamp: durationOf(timeStamp), Type: typ, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: physical, Logical: logical, Synthesized: true,
	})
	if typ == ui.KeyEventTypeDown {
		c.set(physical, logical)
	} else {
		c.remove(physical)
	}
}

// KeyIsPressed reports whether the physical key is known to be down.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.keyIsPressed
func (c *KeyConverter) KeyIsPressed(physical uint64) bool {
	_, ok := c.record(physical)
	return ok
}

// ClearPressedKeys forgets which keys are down.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.clearPressedKeys
func (c *KeyConverter) ClearPressedKeys() { c.pressing = nil }

// ReleaseAll gives the framework a synthesized up for every key that is down, in the order
// they went down, and forgets them: the page lost the focus, and the keyups will go elsewhere.
//
// Deviation: Flutter web keeps the record across a loss of focus and repairs it at the next
// event (SPEC §12.10).
//
// Flutter: no counterpart
func (c *KeyConverter) ReleaseAll(timeStamp float64) {
	pressed := c.pressing
	c.pressing = nil
	for _, p := range pressed {
		c.perform(ui.KeyData{
			TimeStamp: durationOf(timeStamp), Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard,
			Physical: p.physical, Logical: p.logical, Synthesized: true,
		})
	}
	for p, cancel := range c.guards {
		delete(c.guards, p)
		cancel()
	}
}

// Dispose cancels what the converter scheduled and forgets the pressed keys.
//
// Flutter: lib/web_ui/lib/src/engine/keyboard_binding.dart KeyboardConverter.dispose
func (c *KeyConverter) Dispose() {
	c.disposed = true
	c.ClearPressedKeys()
}
