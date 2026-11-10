package ios

import (
	"sort"
	"time"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the hardware keyboard of the iOS backend that has no UIKit call in it (SPEC §30.4): the conversion of a UIPress with a
// UIKey, as the view gives it, to the events of the framework. It is the embedder key responder of Flutter's iOS embedder, ported to plain
// Go: the state of the pressed keys, the synchronization of the modifier keys with the flags of each press, the CapsLock key, and the
// logical key of a key. The tables are in keymap_gen.go (generated from Flutter's key data). The UIKit side is in input_ios.go.

// The planes of a logical key and the mask of its value: the logical key of a character is the character in the Unicode plane (lower
// case), and a key that has no character is the key code in the iOS plane.
//
// Flutter: shell/platform/darwin/ios/framework/Source/KeyCodeMap.g.mm kValueMask, kUnicodePlane, kIosPlane
const (
	valueMask    uint64 = 0xffffffff
	unicodePlane uint64 = 0x0
	iosPlane     uint64 = 0x01300000000
)

// The physical and the logical key of CapsLock, which is handled apart.
//
// Flutter: KeyCodeMap.g.mm kCapsLockPhysicalKey, kCapsLockLogicalKey
const (
	capsLockPhysicalKey uint64 = 0x70039
	capsLockLogicalKey  uint64 = 0x100000104
	capsLockKeyCode     uint32 = 0x39
)

// The bits of UIKey.modifierFlags (UIKeyModifierFlags) and the sided bits that Flutter keeps next to them: UIKit has no left and right,
// so the responder works them out from the presses of the modifier keys.
//
// Flutter: KeyCodeMap_Internal.h ModifierFlag
const (
	flagControlLeft  uint64 = 0x1
	flagShiftLeft    uint64 = 0x2
	flagShiftRight   uint64 = 0x4
	flagMetaLeft     uint64 = 0x8
	flagMetaRight    uint64 = 0x10
	flagAltLeft      uint64 = 0x20
	flagAltRight     uint64 = 0x40
	flagControlRight uint64 = 0x2000
	flagCapsLock     uint64 = 0x010000
	flagShiftAny     uint64 = 0x020000
	flagControlAny   uint64 = 0x040000
	flagAltAny       uint64 = 0x080000
	flagMetaAny      uint64 = 0x100000
)

const (
	flagAnyMask   = flagShiftAny | flagControlAny | flagAltAny | flagMetaAny
	flagSidedMask = flagControlLeft | flagShiftLeft | flagShiftRight | flagMetaLeft | flagMetaRight | flagAltLeft | flagAltRight | flagControlRight
)

// keyCodeToModifierFlag is the sided flag of the modifier key of a key code (UIKeyboardHIDUsage).
//
// Flutter: KeyCodeMap.g.mm keyCodeToModifierFlag
var keyCodeToModifierFlag = map[uint32]uint64{
	0xe1: flagShiftLeft, 0xe5: flagShiftRight, 0xe0: flagControlLeft, 0xe4: flagControlRight,
	0xe2: flagAltLeft, 0xe6: flagAltRight, 0xe3: flagMetaLeft, 0xe7: flagMetaRight,
}

// modifierFlagToKeyCode is keyCodeToModifierFlag the other way round.
//
// Flutter: KeyCodeMap.g.mm modifierFlagToKeyCode
var modifierFlagToKeyCode = func() map[uint64]uint32 {
	m := make(map[uint64]uint32, len(keyCodeToModifierFlag))
	for code, flag := range keyCodeToModifierFlag {
		m[flag] = code
	}
	return m
}()

// modifierFlagOfInterestMask is every flag that the responder synchronizes.
//
// Flutter: FlutterEmbedderKeyResponder.mm computeModifierFlagOfInterestMask
var modifierFlagOfInterestMask = func() uint64 {
	mask := flagCapsLock | flagShiftAny | flagControlAny | flagAltAny | flagMetaAny
	for _, flag := range keyCodeToModifierFlag {
		mask |= flag
	}
	return mask
}()

// The key codes of the modifier keys that fixSidedFlags tells apart.
const (
	keyLeftControl  uint32 = 0xe0
	keyLeftShift    uint32 = 0xe1
	keyLeftAlt      uint32 = 0xe2
	keyLeftGUI      uint32 = 0xe3
	keyRightControl uint32 = 0xe4
	keyRightShift   uint32 = 0xe5
	keyRightAlt     uint32 = 0xe6
	keyRightGUI     uint32 = 0xe7
)

// specialKeyMapping is the logical key of a key whose UIKey.charactersIgnoringModifiers is one of UIKit's special strings.
//
// Flutter: KeyCodeMap.g.mm specialKeyMapping
var specialKeyMapping = map[string]uint64{
	"UIKeyInputEscape": 0x10000001b,
	"UIKeyInputF1":     0x100000801, "UIKeyInputF2": 0x100000802, "UIKeyInputF3": 0x100000803, "UIKeyInputF4": 0x100000804,
	"UIKeyInputF5": 0x100000805, "UIKeyInputF6": 0x100000806, "UIKeyInputF7": 0x100000807, "UIKeyInputF8": 0x100000808,
	"UIKeyInputF9": 0x100000809, "UIKeyInputF10": 0x10000080a, "UIKeyInputF11": 0x10000080b, "UIKeyInputF12": 0x10000080c,
	"UIKeyInputUpArrow": 0x100000304, "UIKeyInputDownArrow": 0x100000301, "UIKeyInputLeftArrow": 0x100000302, "UIKeyInputRightArrow": 0x100000303,
	"UIKeyInputHome": 0x100000306, "UIKeyInputEnd": 0x10000000d, "UIKeyInputPageUp": 0x100000308, "UIKeyInputPageDown": 0x100000307,
}

// pressPhase is the phase of a UIPress that the responder handles: the others (changed, stationary, cancelled) are not given to it.
type pressPhase int

// The phases of a press, with UIKit's numbers.
const (
	pressBegan pressPhase = iota
	pressChanged
	pressStationary
	pressEnded
	pressCancelled
)

// pressEvent is what the responder reads of a UIPress with a UIKey.
type pressEvent struct {
	phase   pressPhase
	keyCode uint32
	// modifierFlags is UIKey.modifierFlags.
	modifierFlags uint64
	timestamp     time.Duration
	// characters is UIKey.characters, and ignoring is UIKey.charactersIgnoringModifiers.
	characters, ignoring string
}

// keyResponder turns presses into the events of the framework, and keeps the state of the keys. It is used on the UI thread only.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterEmbedderKeyResponder.mm FlutterEmbedderKeyResponder
type keyResponder struct {
	// send gives an event to the framework and returns whether it handled it. The answer of a synthesized event is not used.
	send func(ui.KeyData) bool
	// pressing has the pressed keys: the physical key and the logical key of its down event.
	pressing map[uint64]uint64
	// lastFlags are the modifier flags of the last press, masked with modifierFlagOfInterestMask, in step with pressing.
	lastFlags uint64
}

// newKeyResponder returns a responder that gives its events to send.
func newKeyResponder(send func(ui.KeyData) bool) *keyResponder {
	return &keyResponder{send: send, pressing: map[uint64]uint64{}}
}

// pressedKeys returns the pressed keys, as the framework's "getKeyboardState" answers: physical key to the logical key of its down event.
//
// Flutter: FlutterEmbedderKeyResponder.mm getPressedState
func (r *keyResponder) pressedKeys() map[uint64]uint64 {
	out := make(map[uint64]uint64, len(r.pressing))
	for k, v := range r.pressing {
		out[k] = v
	}
	return out
}

// releaseAll sends a synthesized release for every pressed key, in the order of the keys, and forgets them: the view lost the keyboard
// focus (another app, the app switcher), and the key ups will go elsewhere.
//
// Flutter: no counterpart (the framework's HardwareKeyboard.syncKeyboardState does it from its side)
func (r *keyResponder) releaseAll(now time.Duration) {
	keys := make([]uint64, 0, len(r.pressing))
	for k := range r.pressing {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, physical := range keys {
		r.sendSynth(ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeUp, Physical: physical, Logical: r.pressing[physical]})
	}
	r.pressing = map[uint64]uint64{}
	r.lastFlags = 0
}

// handle processes one press and says whether the framework handled it: a press of a key that is down already, and the end of a key
// that was never down, count as handled, so that they go no further.
//
// Flutter: FlutterEmbedderKeyResponder.mm handlePress:callback:
func (r *keyResponder) handle(ev pressEvent) bool {
	switch ev.phase {
	case pressBegan:
		return r.handleBegin(ev)
	case pressEnded:
		return r.handleEnd(ev)
	}
	return true
}

func (r *keyResponder) handleBegin(ev pressEvent) bool {
	physical := physicalKeyForCode(ev.keyCode)
	special, isSpecial := specialKeyMapping[ev.ignoring]
	logical := logicalKeyForEvent(ev, special, isSpecial)
	r.synchronize(ev)
	if _, down := r.pressing[physical]; down {
		return true // UIKit gives no repeat of a key that is held: a second press of it is nothing new
	}
	r.pressing[physical] = logical
	d := ui.KeyData{TimeStamp: ev.timestamp, Type: ui.KeyEventTypeDown, Physical: physical, Logical: logical}
	if !isSpecial {
		d.Character = eventCharacters(ev.characters, ev.keyCode)
	}
	return r.sendPrimary(d)
}

func (r *keyResponder) handleEnd(ev pressEvent) bool {
	r.synchronize(ev)
	physical := physicalKeyForCode(ev.keyCode)
	logical, down := r.pressing[physical]
	if !down {
		return true
	}
	delete(r.pressing, physical)
	return r.sendPrimary(ui.KeyData{TimeStamp: ev.timestamp, Type: ui.KeyEventTypeUp, Physical: physical, Logical: logical})
}

// synchronize gives the framework the modifier keys that went down or up without a press of the view (the user changed them while
// another view had the keyboard): the events are synthesized.
//
// Flutter: FlutterEmbedderKeyResponder.mm synchronizeModifiers:
func (r *keyResponder) synchronize(ev pressEvent) {
	last := r.lastFlags & modifierFlagOfInterestMask
	current := r.adjustModifiers(ev) & modifierFlagOfInterestMask
	diff := current ^ last
	if diff&flagCapsLock != 0 {
		if ev.keyCode != capsLockKeyCode {
			r.capsLockTap(ev.timestamp)
		}
		diff &^= flagCapsLock
	}
	for diff != 0 {
		flag := diff & -diff // the lowest set bit
		diff &^= flag
		if flag&flagAnyMask != 0 {
			continue
		}
		code, ok := modifierFlagToKeyCode[flag]
		if !ok || code == ev.keyCode {
			continue
		}
		r.modifierEvent(current&flag != 0, ev.timestamp, code)
	}
	r.lastFlags = (r.lastFlags &^ modifierFlagOfInterestMask) | current
}

// adjustModifiers is the modifier flags of the press with the sided flags that UIKit does not have: those of the last press are kept, and
// the key of the press is applied, because UIKit leaves the flag of a key that is being released in the flags of its press.
//
// Flutter: FlutterEmbedderKeyResponder.mm adjustModifiers:
func (r *keyResponder) adjustModifiers(ev pressEvent) uint64 {
	down := ev.phase == pressBegan || ev.phase == pressChanged || ev.phase == pressStationary
	flags := ev.modifierFlags | (r.lastFlags & flagSidedMask)
	for _, m := range []struct {
		any, left, right  uint64
		leftKey, rightKey uint32
	}{
		{flagShiftAny, flagShiftLeft, flagShiftRight, keyLeftShift, keyRightShift},
		{flagControlAny, flagControlLeft, flagControlRight, keyLeftControl, keyRightControl},
		{flagAltAny, flagAltLeft, flagAltRight, keyLeftAlt, keyRightAlt},
		{flagMetaAny, flagMetaLeft, flagMetaRight, keyLeftGUI, keyRightGUI},
	} {
		flags = fixSidedFlags(m.any, m.left, m.right, m.leftKey, m.rightKey, ev.keyCode, down, flags)
	}
	return flags
}

// fixSidedFlags keeps the left and right flags of one kind of modifier in step with the any flag and with the key of the press.
//
// Flutter: FlutterEmbedderKeyResponder.mm fixSidedFlags:withLeftFlag:withRightFlag:withLeftKey:withRightKey:withKeyCode:keyDown:forFlags:
func fixSidedFlags(anyFlag, leftSide, rightSide uint64, leftKey, rightKey, keyCode uint32, keyDown bool, flags uint64) uint64 {
	n := flags
	if keyDown {
		switch keyCode {
		case leftKey:
			n |= leftSide | anyFlag
		case rightKey:
			n |= rightSide | anyFlag
		}
	} else {
		switch keyCode {
		case leftKey:
			n &^= leftSide
			if n&rightSide == 0 {
				n &^= anyFlag
			}
		case rightKey:
			n &^= rightSide
			if n&leftSide == 0 {
				n &^= anyFlag
			}
		}
	}
	if n&anyFlag == 0 {
		n &^= leftSide | rightSide
	}
	return n
}

func (r *keyResponder) sendSynth(d ui.KeyData) {
	d.DeviceType = ui.KeyEventDeviceTypeKeyboard
	d.Synthesized = true
	r.send(d)
}

func (r *keyResponder) sendPrimary(d ui.KeyData) bool {
	d.DeviceType = ui.KeyEventDeviceTypeKeyboard
	return r.send(d)
}

// capsLockTap sends a synthesized CapsLock down and CapsLock up.
//
// Flutter: FlutterEmbedderKeyResponder.mm synthesizeCapsLockTapWithTimestamp:
func (r *keyResponder) capsLockTap(now time.Duration) {
	d := ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeDown, Physical: capsLockPhysicalKey, Logical: capsLockLogicalKey}
	r.sendSynth(d)
	d.Type = ui.KeyEventTypeUp
	r.sendSynth(d)
}

// modifierEvent sends the synthesized event of a modifier key and records it.
//
// Flutter: FlutterEmbedderKeyResponder.mm synthesizeModifierEventOfType:timestamp:keyCode:
func (r *keyResponder) modifierEvent(down bool, now time.Duration, code uint32) {
	physical := physicalKeyForCode(code)
	logical := logicalKeyForModifier(code, physical)
	if physical == 0 || logical == 0 {
		return
	}
	d := ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeUp, Physical: physical, Logical: logical}
	if down {
		d.Type = ui.KeyEventTypeDown
		r.pressing[physical] = logical
	} else {
		delete(r.pressing, physical)
	}
	r.sendSynth(d)
}

// physicalKeyForCode is the physical key of a key code: the USB usage, or a key of the iOS plane for a code that the table does not have.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetPhysicalKeyForKeyCode
func physicalKeyForCode(code uint32) uint64 {
	if k, ok := keyCodeToPhysicalKey[code]; ok {
		return k
	}
	return keyOfPlane(uint64(code), iosPlane)
}

// logicalKeyForModifier is the logical key of a modifier key.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetLogicalKeyForModifier
func logicalKeyForModifier(code uint32, hid uint64) uint64 {
	if k, ok := keyCodeToLogicalKey[code]; ok {
		return k
	}
	return keyOfPlane(hid, iosPlane)
}

// logicalKeyForEvent is the logical key of a key down: a special key first, then the table of key codes, then the character that the key
// makes without the modifiers (lower case; a character outside the BMP is its two UTF-16 units, as Flutter has it), and last a key of the
// iOS plane.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetLogicalKeyForEvent
func logicalKeyForEvent(ev pressEvent, special uint64, isSpecial bool) uint64 {
	if isSpecial {
		return special
	}
	if k, ok := keyCodeToLogicalKey[ev.keyCode]; ok {
		return k
	}
	if label := eventCharacters(ev.ignoring, ev.keyCode); label != "" {
		units := utf16.Encode([]rune(label))
		if len(units) <= 2 && !isControlCharacter(units) && !isUnprintableKey(units) {
			code := uint64(units[0])
			if len(units) == 2 {
				code = code<<16 | uint64(units[1])
			}
			return keyOfPlane(toLower(code), unicodePlane)
		}
	}
	return keyOfPlane(uint64(ev.keyCode), iosPlane)
}

// isControlCharacter says whether a label of one UTF-16 unit is a control character.
//
// Flutter: FlutterEmbedderKeyResponder.mm IsControlCharacter
func isControlCharacter(units []uint16) bool {
	if len(units) > 1 {
		return false
	}
	u := units[0]
	return u <= 0x1f || (u >= 0x7f && u <= 0x9f)
}

// isUnprintableKey says whether a label of one UTF-16 unit is in the private area of the function keys (0xF700 to 0xF8FF).
//
// Flutter: FlutterEmbedderKeyResponder.mm IsUnprintableKey
func isUnprintableKey(units []uint16) bool {
	if len(units) > 1 {
		return false
	}
	return units[0] >= 0xf700 && units[0] <= 0xf8ff
}

// eventCharacters is the characters of a press for the framework: none for a key with no characters and for a function key.
//
// Flutter: FlutterEmbedderKeyResponder.mm getEventCharacters
func eventCharacters(characters string, keyCode uint32) string {
	if characters == "" || functionKeyCodes[keyCode] {
		return ""
	}
	return characters
}

func keyOfPlane(base, plane uint64) uint64 { return plane | (base & valueMask) }

// toLower makes the ASCII and Latin-1 upper-case letters lower case and leaves the rest, whatever the locale is.
//
// Flutter: FlutterEmbedderKeyResponder.mm toLower
func toLower(n uint64) uint64 {
	switch {
	case n >= 0x41 && n <= 0x5a:
		return n - 0x41 + 0x61
	case n >= 0xc0 && n <= 0xde && n != 0xf7:
		return n - 0xc0 + 0xe0
	}
	return n
}
