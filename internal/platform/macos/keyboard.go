package macos

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the keyboard of the macOS backend that has no AppKit call in it (SPEC §29.6): the conversion
// of a key event, as AppKit gives it, to the events of the framework. It is the embedder key responder of
// Flutter's macOS embedder, ported to plain Go: the state of the pressed keys, the synchronization of the
// modifier keys with the flags of each event, the CapsLock key (which macOS reports once for two taps), and
// the logical key of a key. The tables are in keymap_gen.go (generated from Flutter's key data); the layout
// of the keyboard is in layout.go; the AppKit side is in keyboard_darwin.go.

// The planes of a logical key, and the mask of its value: the logical key of a character is the character
// in the Unicode plane (lower case), and a key that has no character is the key code in the macOS plane.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/KeyCodeMap.g.mm kValueMask,
// kUnicodePlane, kMacosPlane
const (
	valueMask    uint64 = 0xffffffff
	unicodePlane uint64 = 0x0
	macosPlane   uint64 = 0x01400000000
)

// The physical and the logical key of CapsLock, which is handled apart.
//
// Flutter: KeyCodeMap.g.mm kCapsLockPhysicalKey, kCapsLockLogicalKey
const (
	capsLockPhysicalKey uint64 = 0x70039
	capsLockLogicalKey  uint64 = 0x100000104
)

// The bits of NSEvent.modifierFlags that tell which modifier key is down. They are not in Apple's
// documentation: Flutter found them by experiment, and it is the only way to tell the left key from the
// right one.
//
// Flutter: KeyCodeMap_Internal.h kModifierFlagControlLeft and the others
const (
	flagCapsLock     uint64 = 1 << 16
	flagControlLeft  uint64 = 0x1
	flagShiftLeft    uint64 = 0x2
	flagShiftRight   uint64 = 0x4
	flagMetaLeft     uint64 = 0x8
	flagMetaRight    uint64 = 0x10
	flagAltLeft      uint64 = 0x20
	flagAltRight     uint64 = 0x40
	flagControlRight uint64 = 0x2000
)

// keyCodeToModifierFlag is the flag of the modifier key of a key code (CapsLock is not here).
//
// Flutter: KeyCodeMap.g.mm keyCodeToModifierFlag
var keyCodeToModifierFlag = map[uint16]uint64{
	0x38: flagShiftLeft,
	0x3c: flagShiftRight,
	0x3b: flagControlLeft,
	0x3e: flagControlRight,
	0x3a: flagAltLeft,
	0x3d: flagAltRight,
	0x37: flagMetaLeft,
	0x36: flagMetaRight,
}

// modifierFlagToKeyCode is keyCodeToModifierFlag the other way round.
//
// Flutter: KeyCodeMap.g.mm modifierFlagToKeyCode
var modifierFlagToKeyCode = func() map[uint64]uint16 {
	m := make(map[uint64]uint16, len(keyCodeToModifierFlag))
	for code, flag := range keyCodeToModifierFlag {
		m[flag] = code
	}
	return m
}()

// modifierFlagOfInterestMask is every flag that the responder synchronizes: the flags of the modifier keys
// and CapsLock.
//
// Flutter: FlutterEmbedderKeyResponder.mm computeModifierFlagOfInterestMask
var modifierFlagOfInterestMask = func() uint64 {
	mask := flagCapsLock
	for _, flag := range keyCodeToModifierFlag {
		mask |= flag
	}
	return mask
}()

// The kinds of key event that AppKit gives the view.
const (
	keyKindDown = iota
	keyKindUp
	keyKindFlagsChanged
)

// keyEvent is what the responder reads of an NSEvent of a key.
type keyEvent struct {
	kind    int
	keyCode uint16
	// modifierFlags is NSEvent.modifierFlags.
	modifierFlags uint64
	timestamp     time.Duration
	isARepeat     bool
	// characters is NSEvent.characters, and ignoringModifiers is NSEvent.charactersIgnoringModifiers (both
	// are empty for a flags-changed event).
	characters, ignoringModifiers string
}

// keyResponder turns key events into the events of the framework, and keeps the state of the keys. It is
// used on the UI thread only.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/FlutterEmbedderKeyResponder.mm
// FlutterEmbedderKeyResponder
type keyResponder struct {
	// send gives an event to the framework and returns whether it handled it. The answer of a synthesized
	// event is not used.
	send func(ui.KeyData) bool
	// layout is the logical key that the layout of the keyboard gives a key code, for the typewriter keys
	// (layout.go); the map is replaced when the layout changes.
	layout map[uint16]uint64
	// pressing has the pressed keys: the physical key and the logical key of its down event.
	pressing map[uint64]uint64
	// lastFlags are the modifier flags of the last event, masked with modifierFlagOfInterestMask. They are
	// kept in step with pressing.
	lastFlags uint64
}

// newKeyResponder returns a responder that gives its events to send.
func newKeyResponder(send func(ui.KeyData) bool) *keyResponder {
	return &keyResponder{send: send, pressing: map[uint64]uint64{}}
}

// pressedKeys returns the pressed keys, as the framework's "getKeyboardState" answers: physical key to the
// logical key of its down event.
//
// Flutter: FlutterEmbedderKeyResponder.mm getPressedState
func (r *keyResponder) pressedKeys() map[uint64]uint64 {
	out := make(map[uint64]uint64, len(r.pressing))
	for k, v := range r.pressing {
		out[k] = v
	}
	return out
}

// reset forgets every pressed key and the modifier flags, without events: a window that closes, or an app
// that lost the keyboard, keeps nothing.
func (r *keyResponder) reset() {
	r.pressing = map[uint64]uint64{}
	r.lastFlags = 0
}

// releaseAll sends a synthesized release for every pressed key, in the order of the keys, and forgets
// them: the window lost the keyboard focus, and the key ups will go to another window.
//
// Flutter: no counterpart (the framework's HardwareKeyboard.syncKeyboardState does it from its side)
func (r *keyResponder) releaseAll(now time.Duration) {
	keys := make([]uint64, 0, len(r.pressing))
	for k := range r.pressing {
		keys = append(keys, k)
	}
	sortUint64(keys)
	for _, physical := range keys {
		r.sendSynth(ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeUp, Physical: physical, Logical: r.pressing[physical]})
	}
	r.reset()
}

// handle processes one event of the system and says whether the framework handled it: it did when
// the framework said so, and an event that changed nothing (a key up of a key that was never down) counts
// as handled, so that it is not given to the text input.
//
// Flutter: FlutterEmbedderKeyResponder.mm handleEvent, FlutterKeyboardManager.mm performProcessEvent
func (r *keyResponder) handle(ev keyEvent) (handled bool) {
	switch ev.kind {
	case keyKindDown:
		handled = r.handleDown(ev)
	case keyKindUp:
		handled = r.handleUp(ev)
	case keyKindFlagsChanged:
		handled = r.handleFlags(ev)
	}
	return handled
}

// syncModifiers gives the framework the modifier keys that went down or up without an event of the
// window (the user changed them while another window was in front): the events are synthesized.
//
// Flutter: FlutterEmbedderKeyResponder.mm syncModifiersIfNeeded
func (r *keyResponder) syncModifiers(flags uint64, now time.Duration) {
	r.synchronize(flags, 0, now)
}

func (r *keyResponder) synchronize(current, ignoring uint64, now time.Duration) {
	updating := modifierFlagOfInterestMask &^ ignoring
	currentOfInterest := current & updating
	lastOfInterest := r.lastFlags & updating
	diff := currentOfInterest ^ lastOfInterest
	if diff&flagCapsLock != 0 {
		r.capsLockTap(now, true)
		diff &^= flagCapsLock
	}
	for diff != 0 {
		flag := diff & -diff // the lowest set bit
		diff &^= flag
		code, ok := modifierFlagToKeyCode[flag]
		if !ok {
			continue
		}
		r.modifierEvent(currentOfInterest&flag != 0, now, code, true)
	}
	r.lastFlags = (r.lastFlags &^ updating) | currentOfInterest
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

// capsLockTap sends a CapsLock down and a CapsLock up. macOS reports a tap of CapsLock as a down or as an
// up, alternately, and the responder never learns when the key is released, so the up is always
// synthesized; the down is too when the tap is only noticed from the flags.
//
// Flutter: FlutterEmbedderKeyResponder.mm sendCapsLockTapWithTimestamp
func (r *keyResponder) capsLockTap(now time.Duration, synthesizeDown bool) (handled bool) {
	d := ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeDown, Physical: capsLockPhysicalKey, Logical: capsLockLogicalKey}
	handled = true
	if synthesizeDown {
		r.sendSynth(d)
	} else {
		handled = r.sendPrimary(d)
	}
	d.Type = ui.KeyEventTypeUp
	r.sendSynth(d)
	return handled
}

// modifierEvent sends the event of a modifier key.
//
// Flutter: FlutterEmbedderKeyResponder.mm sendModifierEventOfType
func (r *keyResponder) modifierEvent(down bool, now time.Duration, code uint16, synthesized bool) (handled bool) {
	physical := physicalKeyForCode(code)
	logical := logicalKeyForModifier(code, physical)
	d := ui.KeyData{TimeStamp: now, Type: ui.KeyEventTypeUp, Physical: physical, Logical: logical}
	if down {
		d.Type = ui.KeyEventTypeDown
	}
	if down {
		r.pressing[physical] = logical
	} else {
		delete(r.pressing, physical)
	}
	if synthesized {
		r.sendSynth(d)
		return true
	}
	return r.sendPrimary(d)
}

func (r *keyResponder) handleDown(ev keyEvent) bool {
	physical := physicalKeyForCode(ev.keyCode)
	logical, ok := r.layout[ev.keyCode]
	if !ok {
		logical = logicalKeyForEvent(ev)
	}
	r.synchronize(ev.modifierFlags, 0, ev.timestamp)

	pressedLogical, pressed := r.pressing[physical]
	if pressed && !ev.isARepeat {
		// The key is down already: the focus came from another view amid the tap, or an input method forged
		// the event with the key code of a key that is still down.
		r.sendSynth(ui.KeyData{TimeStamp: ev.timestamp, Type: ui.KeyEventTypeUp, Physical: physical, Logical: pressedLogical})
		pressed = false
	}
	d := ui.KeyData{TimeStamp: ev.timestamp, Physical: physical, Character: eventString(ev.characters)}
	if pressed {
		d.Type, d.Logical = ui.KeyEventTypeRepeat, pressedLogical
	} else {
		d.Type, d.Logical = ui.KeyEventTypeDown, logical
		r.pressing[physical] = logical
	}
	return r.sendPrimary(d)
}

func (r *keyResponder) handleUp(ev keyEvent) bool {
	r.synchronize(ev.modifierFlags, 0, ev.timestamp)
	physical := physicalKeyForCode(ev.keyCode)
	pressedLogical, pressed := r.pressing[physical]
	if !pressed {
		// The down went to another view.
		return true
	}
	delete(r.pressing, physical)
	return r.sendPrimary(ui.KeyData{TimeStamp: ev.timestamp, Type: ui.KeyEventTypeUp, Physical: physical, Logical: pressedLogical})
}

func (r *keyResponder) handleFlags(ev keyEvent) bool {
	target := keyCodeToModifierFlag[ev.keyCode]
	targetKey := physicalKeyForCode(ev.keyCode)
	if targetKey == capsLockPhysicalKey {
		return r.handleCapsLock(ev)
	}
	r.synchronize(ev.modifierFlags, target, ev.timestamp)
	_, lastPressed := r.pressing[targetKey]
	shouldBePressed := ev.modifierFlags&target != 0
	if lastPressed == shouldBePressed {
		return true
	}
	r.lastFlags ^= target
	return r.modifierEvent(shouldBePressed, ev.timestamp, ev.keyCode, false)
}

func (r *keyResponder) handleCapsLock(ev keyEvent) bool {
	r.synchronize(ev.modifierFlags, flagCapsLock, ev.timestamp)
	if r.lastFlags&flagCapsLock != ev.modifierFlags&flagCapsLock {
		handled := r.capsLockTap(ev.timestamp, false)
		r.lastFlags ^= flagCapsLock
		return handled
	}
	return true
}

// physicalKeyForCode is the physical key of a key code: the USB usage, or a key of the macOS plane for a code
// that the table does not have.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetPhysicalKeyForKeyCode
func physicalKeyForCode(code uint16) uint64 {
	if k, ok := keyCodeToPhysicalKey[code]; ok {
		return k
	}
	return keyOfPlane(uint64(code), macosPlane)
}

// logicalKeyForModifier is the logical key of a modifier key.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetLogicalKeyForModifier
func logicalKeyForModifier(code uint16, hid uint64) uint64 {
	if k, ok := keyCodeToLogicalKey[code]; ok {
		return k
	}
	return keyOfPlane(hid, macosPlane)
}

// logicalKeyForEvent is the logical key of a key down: the table of key codes first (the keys that have
// no character, or whose logical key must not depend on the layout), then the character the key makes
// without the modifiers, in lower case, and last a key of the macOS plane.
//
// Flutter: FlutterEmbedderKeyResponder.mm GetLogicalKeyForEvent
func logicalKeyForEvent(ev keyEvent) uint64 {
	if k, ok := keyCodeToLogicalKey[ev.keyCode]; ok {
		return k
	}
	// A key whose label is one character (in code points) takes it as its logical key; one with more
	// characters has no meaningful value.
	var character rune
	n := 0
	for _, c := range ev.ignoringModifiers {
		character = c
		n++
	}
	if n == 1 && character != 0 {
		return keyOfPlane(toLower(uint64(character)), unicodePlane)
	}
	return keyOfPlane(uint64(ev.keyCode), macosPlane)
}

func keyOfPlane(base, plane uint64) uint64 { return plane | (base & valueMask) }

// toLower makes the ASCII and Latin-1 upper-case letters lower case and leaves the rest, whatever the
// locale is. Like Flutter's, it also turns the multiplication sign (0xd7) into the division sign: Flutter
// leaves out 0xf7, which is not in the range, and not 0xd7, which is.
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

// eventString is the character of a key down for the framework: none when the key has no text, and none
// when the first character is one of the function keys that AppKit gives a code from the private use area
// (0xF700-0xF7FF); the printable private characters, such as the Apple logo (0xF8FF), stay.
//
// Flutter: FlutterEmbedderKeyResponder.mm getEventString
func eventString(characters string) string {
	for _, c := range characters {
		if c >= 0xf700 && c <= 0xf7ff {
			return ""
		}
		break
	}
	return characters
}

// sortUint64 sorts a small slice in place (insertion sort: the pressed keys are a handful).
func sortUint64(a []uint64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
