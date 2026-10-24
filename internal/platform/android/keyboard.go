package android

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The constants of android.view.KeyEvent, KeyCharacterMap and InputDevice that the
// conversion reads.
const (
	// keyActionDown and keyActionUp are KeyEvent.ACTION_DOWN and ACTION_UP.
	keyActionDown int32 = 0
	keyActionUp   int32 = 1

	// The meta state bits of KeyEvent that KeyboardMap uses.
	metaAltOn      int32 = 0x2
	metaShiftOn    int32 = 0x1
	metaCtrlOn     int32 = 0x1000
	metaCapsLockOn int32 = 0x100000

	// combiningAccent and combiningAccentMask are KeyCharacterMap.COMBINING_ACCENT and
	// COMBINING_ACCENT_MASK, the bit that getUnicodeChar sets for a dead key and the bits
	// of the accent.
	combiningAccent     uint32 = 0x80000000
	combiningAccentMask uint32 = 0x7fffffff

	// virtualKeyboardDevice is KeyCharacterMap.VIRTUAL_KEYBOARD, the device id of the soft
	// keyboards.
	virtualKeyboardDevice int32 = -1

	// The sources of InputDevice that tell the device type.
	sourceKeyboard int32 = 0x101
	sourceDpad     int32 = 0x201
	sourceGamepad  int32 = 0x401
	sourceJoystick int32 = 0x1000010
	sourceHDMI     int32 = 0x2000001
)

// keyEvent holds the values that the Java side copies out of a KeyEvent, so that the
// conversion is plain Go. Fields that Flutter's embedding does not read are not copied.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.handleEvent
type keyEvent struct {
	// Action is KeyEvent.getAction: ACTION_DOWN (0) or ACTION_UP (1). Other actions
	// (ACTION_MULTIPLE) are ignored.
	Action int32
	// KeyCode is KeyEvent.getKeyCode and ScanCode is KeyEvent.getScanCode.
	KeyCode, ScanCode int32
	// MetaState is KeyEvent.getMetaState.
	MetaState int32
	// RepeatCount is KeyEvent.getRepeatCount.
	RepeatCount int32
	// EventTime is KeyEvent.getEventTime, in milliseconds.
	EventTime int64
	// UnicodeChar is KeyEvent.getUnicodeChar() as it is: a dead key has the bit
	// KeyCharacterMap.COMBINING_ACCENT set.
	UnicodeChar int32
	// DeviceID is KeyEvent.getDeviceId (KeyCharacterMap.VIRTUAL_KEYBOARD, -1, for soft
	// keyboards) and Source is KeyEvent.getSource.
	DeviceID, Source int32
}

// keyPair is a physical key and a logical key.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardMap.java KeyboardMap.KeyPair
type keyPair struct {
	physical, logical uint64
}

// pressingGoal says how to synchronize a pressing modifier (Shift, Control, Alt): the
// embedder synthesizes events until the pressing state of keys matches the bits of the
// meta state under mask.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardMap.java KeyboardMap.PressingGoal
type pressingGoal struct {
	mask int32
	keys []keyPair
}

// togglingGoal says how to synchronize a toggling modifier (CapsLock) with the bit of
// the meta state under mask. The current state is in the embedder, not here, so that the
// goals stay read-only.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardMap.java KeyboardMap.TogglingGoal
type togglingGoal struct {
	mask              int32
	physical, logical uint64
}

// keyEventType is what an event is: Flutter's KeyData.Type.
type keyEventType int

const (
	keyEventUnknown keyEventType = iota
	keyEventDown
	keyEventUp
	keyEventRepeat
)

// eventType maps the action and the repeat count of an event to a type.
//
// Deviation: Flutter throws an AssertionError for an action other than down and up; the
// embedder here gives keyEventUnknown, which the callers take as an up (no synthesized
// events for the key) and the event is ignored (SPEC §12.10).
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.getEventType
func eventType(ev keyEvent) keyEventType {
	switch ev.Action {
	case keyActionDown:
		if ev.RepeatCount > 0 {
			return keyEventRepeat
		}
		return keyEventDown
	case keyActionUp:
		return keyEventUp
	}
	return keyEventUnknown
}

// characterCombiner applies a dead key to the next character.
//
// Deviation: Flutter works on Java chars (16 bits) and truncates a code point above
// U+FFFF; this one keeps the whole code point (SPEC §12.10).
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager.CharacterCombiner
type characterCombiner struct {
	combining rune
}

// apply takes the value of KeyEvent.getUnicodeChar and returns the character of the
// event, 0 for none. A dead key returns its accent and is remembered; the next regular
// character is combined with it (the accent alone when they do not combine, as Flutter
// does), and a dead key after a dead key combines with it.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager.CharacterCombiner.applyCombiningCharacterToBaseCharacter
func (c *characterCombiner) apply(unicodeChar int32) rune {
	bits := uint32(unicodeChar)
	if bits&combiningAccent != 0 {
		plain := rune(bits & combiningAccentMask)
		if c.combining != 0 {
			c.combining = deadChar(c.combining, plain)
		} else {
			c.combining = plain
		}
		return plain
	}
	complex := rune(unicodeChar)
	if c.combining != 0 {
		if combined := deadChar(c.combining, complex); combined > 0 {
			complex = combined
		}
		c.combining = 0
	}
	return complex
}

// keyEmbedder turns KeyEvents into key data, with the synthesized events that keep the
// pressed keys and the lock state true. It keeps the state of the keyboard, so it must
// be used on one goroutine, in the order of the events.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder
type keyEmbedder struct {
	// pressing maps the pressed physical keys to their logical keys (pressingRecords).
	pressing map[uint64]uint64
	// enabled is the enabling state of each of togglingGoals.
	enabled  []bool
	combiner characterCombiner
	// out collects the key data of the event being handled.
	out []ui.KeyData
}

// newKeyEmbedder makes an embedder with no key pressed and no lock enabled.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.KeyEmbedderResponder
func newKeyEmbedder() *keyEmbedder {
	return &keyEmbedder{
		pressing: map[uint64]uint64{},
		enabled:  make([]bool, len(togglingGoals)),
	}
}

// keyOfPlane puts the value of a key in a plane. The value is masked in case the key is a
// negative number.
func keyOfPlane(key int32, plane uint64) uint64 {
	return plane | (uint64(int64(key)) & keyValueMask)
}

// physicalKey is the physical key of an event. It is never 0.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.getPhysicalKey
func physicalKey(ev keyEvent) uint64 {
	// Scancode 0 can occur during emulation using `adb shell input keyevent`: make a
	// physical key from the key code so that keys can be told apart.
	if ev.ScanCode == 0 {
		return keyOfPlane(ev.KeyCode, keyAndroidPlane)
	}
	if ev.ScanCode > 0 {
		if k, ok := scanCodeToPhysical[uint32(ev.ScanCode)]; ok {
			return k
		}
	}
	return keyOfPlane(ev.ScanCode, keyAndroidPlane)
}

// logicalKey is the logical key of an event. It is never 0.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.getLogicalKey
func logicalKey(ev keyEvent) uint64 {
	if ev.KeyCode > 0 {
		if k, ok := keyCodeToLogical[uint32(ev.KeyCode)]; ok {
			return k
		}
	}
	return keyOfPlane(ev.KeyCode, keyAndroidPlane)
}

// deviceType maps InputDevice's source to the type of device.
func deviceType(source int32) ui.KeyEventDeviceType {
	switch source {
	case sourceDpad:
		return ui.KeyEventDeviceTypeDirectionalPad
	case sourceGamepad:
		return ui.KeyEventDeviceTypeGamepad
	case sourceJoystick:
		return ui.KeyEventDeviceTypeJoystick
	case sourceHDMI:
		return ui.KeyEventDeviceTypeHDMI
	}
	// sourceKeyboard and every other source.
	return ui.KeyEventDeviceTypeKeyboard
}

// updatePressingState marks a key pressed (logical is not 0) or released (logical is 0).
//
// Deviation: Flutter throws an AssertionError when a pressed key is marked pressed again
// or a released key released again; the invariant holds by construction, so this one
// just sets or removes the record.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.updatePressingState
func (k *keyEmbedder) updatePressingState(physical, logical uint64) {
	if logical != 0 {
		k.pressing[physical] = logical
		return
	}
	delete(k.pressing, physical)
}

// synthesize appends a synthesized event.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.synthesizeEvent
func (k *keyEmbedder) synthesize(isDown bool, logical, physical uint64, timestamp int64) {
	typ := ui.KeyEventTypeUp
	if isDown {
		typ = ui.KeyEventTypeDown
	}
	if physical != 0 && logical != 0 {
		if isDown {
			k.updatePressingState(physical, logical)
		} else {
			k.updatePressingState(physical, 0)
		}
	}
	k.out = append(k.out, ui.KeyData{
		TimeStamp:   time.Duration(timestamp) * time.Millisecond,
		Type:        typ,
		DeviceType:  ui.KeyEventDeviceTypeKeyboard,
		Physical:    physical,
		Logical:     logical,
		Synthesized: true,
	})
}

// postEvent is an event to synthesize after the main event.
type postEvent struct {
	logical, physical uint64
}

// synchronizePressingKey synthesizes the events that make the pressed state of the keys
// of a pressing goal match the meta state, taking the event into account. The events for
// before the main event are synthesized now, those for after it are appended to post.
//
// Like Flutter, it only uses the unsided mask (META_SHIFT_ON), because ChromeOS gives the
// right modifiers the unsided and the left bits.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.synchronizePressingKey
func (k *keyEmbedder) synchronizePressingKey(goal pressingGoal, truePressed bool, eventLogical, eventPhysical uint64, ev keyEvent, post *[]postEvent) {
	// The states: now (from the records), before the event (derived), after the event
	// (the true state). The goal is a pre-event state that satisfies the true state after
	// the event with as few synthesized events as possible.
	n := len(goal.keys)
	nowStates := make([]bool, n)
	preEventStates := make([]bool, n)
	preEventSet := make([]bool, n)
	postEventAnyPressed := false
	for i, key := range goal.keys {
		_, nowStates[i] = k.pressing[key.physical]
		if key.logical != eventLogical {
			postEventAnyPressed = postEventAnyPressed || nowStates[i]
			continue
		}
		switch eventType(ev) {
		case keyEventDown:
			preEventStates[i], preEventSet[i] = false, true
			postEventAnyPressed = true
			if !truePressed {
				*post = append(*post, postEvent{key.logical, eventPhysical})
			}
		case keyEventRepeat:
			// Do not synthesize a down here, or there would be a down and a repeat, both
			// with the character.
			if !truePressed {
				*post = append(*post, postEvent{key.logical, eventPhysical})
			}
			preEventStates[i], preEventSet[i] = nowStates[i], true
			postEventAnyPressed = true
		default:
			// An up: the previous state should be pressed, but no down is synthesized if it
			// is not (the abrupt up is skipped later), and no up either.
			preEventStates[i], preEventSet[i] = nowStates[i], true
		}
	}

	// Soft keyboards (Gboard) often keep meta bits without physical key events: trust the
	// meta state and do not synthesize physical modifiers, or they would be stuck.
	isVirtualKeyboard := ev.DeviceID == virtualKeyboardDevice

	// Fill the rest of the pre-event states to match the true state.
	if truePressed {
		// At least one key is pressed.
		for i := range goal.keys {
			if preEventSet[i] {
				continue
			}
			if postEventAnyPressed || isVirtualKeyboard {
				preEventStates[i] = nowStates[i]
			} else {
				preEventStates[i] = true
				postEventAnyPressed = true
			}
		}
		if !postEventAnyPressed && !isVirtualKeyboard {
			preEventStates[0] = true
		}
	}
	// With the true state released, the states that are not set stay false.

	// Synthesize the events for the differences.
	for i, key := range goal.keys {
		if nowStates[i] != preEventStates[i] {
			k.synthesize(preEventStates[i], key.logical, key.physical, ev.EventTime)
		}
	}
}

// synchronizeTogglingKey synthesizes the events that make the lock state match the meta
// state. Only CapsLock is synchronized, and only during events of other keys, because
// ChromeOS sets the CapsLock bit of its own events as if it were a pressing key.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.synchronizeTogglingKey
func (k *keyEmbedder) synchronizeTogglingKey(idx int, trueEnabled bool, eventLogical uint64, ev keyEvent) {
	goal := togglingGoals[idx]
	if goal.logical == eventLogical {
		// No synthesis for the key's own events, whose meta states are weird on ChromeOS.
		return
	}
	if k.enabled[idx] == trueEnabled {
		return
	}
	_, pressed := k.pressing[goal.physical]
	firstIsDown := !pressed
	if firstIsDown {
		k.enabled[idx] = !k.enabled[idx]
	}
	k.synthesize(firstIsDown, goal.logical, goal.physical, ev.EventTime)
	if !firstIsDown {
		k.enabled[idx] = !k.enabled[idx]
	}
	k.synthesize(!firstIsDown, goal.logical, goal.physical, ev.EventTime)
}

// HandleEvent converts one KeyEvent and returns the key data to send to the framework,
// in order: the events synthesized before the main event, the main event (the one that is
// not Synthesized), and the events synthesized after it. When the event gives no main
// event (an event without codes, an abrupt up, an action that is neither down nor up),
// the last data is the empty event of Flutter: a synthesized down with physical and
// logical key 0.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.handleEvent
func (k *keyEmbedder) HandleEvent(ev keyEvent) []ui.KeyData {
	out, _ := k.handle(ev)
	return out
}

// handle is HandleEvent that also returns the index of the main event in the data, or -1
// when there is none.
func (k *keyEmbedder) handle(ev keyEvent) (data []ui.KeyData, main int) {
	k.out = nil
	main = k.handleEventImpl(ev)
	if main < 0 {
		k.synthesize(true, 0, 0, 0)
	}
	data = k.out
	k.out = nil
	return data, main
}

// handleEventImpl is the core algorithm. It returns the index of the main event in k.out,
// or -1 when no main event was made.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.handleEventImpl
func (k *keyEmbedder) handleEventImpl(ev keyEvent) int {
	// Events with no codes at all cannot be recognized.
	if ev.ScanCode == 0 && ev.KeyCode == 0 {
		return -1
	}
	physical := physicalKey(ev)
	logical := logicalKey(ev)

	var post []postEvent
	for _, goal := range pressingGoals {
		k.synchronizePressingKey(goal, ev.MetaState&goal.mask != 0, logical, physical, ev, &post)
	}
	for i, goal := range togglingGoals {
		k.synchronizeTogglingKey(i, ev.MetaState&goal.mask != 0, logical, ev)
	}

	var isDown bool
	switch ev.Action {
	case keyActionDown:
		isDown = true
	case keyActionUp:
		isDown = false
	default:
		return -1
	}

	var typ ui.KeyEventType
	var character string
	last, pressed := k.pressing[physical]
	if isDown {
		switch {
		case !pressed:
			typ = ui.KeyEventTypeDown
		case ev.RepeatCount > 0:
			typ = ui.KeyEventTypeRepeat
		default:
			// A key went down that is already down: release it first.
			k.synthesize(false, last, physical, ev.EventTime)
			typ = ui.KeyEventTypeDown
		}
		if c := k.combiner.apply(ev.UnicodeChar); validRune(c) {
			character = string(c)
		}
	} else {
		if !pressed {
			// Ignore abrupt up events.
			return -1
		}
		typ = ui.KeyEventTypeUp
	}

	if typ != ui.KeyEventTypeRepeat {
		if isDown {
			k.updatePressingState(physical, logical)
		} else {
			k.updatePressingState(physical, 0)
		}
	}
	if typ == ui.KeyEventTypeDown {
		for i, goal := range togglingGoals {
			if goal.logical == logical {
				k.enabled[i] = !k.enabled[i]
			}
		}
	}

	k.out = append(k.out, ui.KeyData{
		TimeStamp:  time.Duration(ev.EventTime) * time.Millisecond,
		Type:       typ,
		DeviceType: deviceType(ev.Source),
		Physical:   physical,
		Logical:    logical,
		Character:  character,
	})
	main := len(k.out) - 1
	for _, p := range post {
		k.synthesize(false, p.logical, p.physical, ev.EventTime)
	}
	return main
}

// PressedState returns a copy of the pressed keys: the ids of the physical keys mapped to
// the ids of their logical keys.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyEmbedderResponder.java KeyEmbedderResponder.getPressedState
func (k *keyEmbedder) PressedState() map[uint64]uint64 {
	state := make(map[uint64]uint64, len(k.pressing))
	for p, l := range k.pressing {
		state[p] = l
	}
	return state
}

// keyboardManager gives each KeyEvent to the framework through its one responder, the
// embedder. Flutter's second responder, the legacy key event channel, is not ported:
// Glutter has no such channel.
//
// Flutter answers the system at once that it handled every key and, when the framework
// says later that it did not, dispatches the event again; the framework of Glutter
// answers at once (SyncInputHandler.OnKeySync), so Handle returns the answer and the
// Java side (the text input and the redispatch, with the memory of the events it
// redispatched) does the rest.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager
type keyboardManager struct {
	embedder *keyEmbedder
}

// newKeyboardManager makes a manager with no key pressed.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager.KeyboardManager
func newKeyboardManager() *keyboardManager {
	return &keyboardManager{embedder: newKeyEmbedder()}
}

// Handle converts the event and calls send for each key data in order; send reports
// whether the framework handled it. handled is the answer for the main event, as the
// callback of Flutter's responder: true when the framework handled it, and true when
// the event made no main event (nothing was left to handle). The answers for the
// synthesized events are ignored. When handled is false, the Java side gives the event
// to the text input and then dispatches it again to the rest of the view hierarchy and
// the system.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager.handleEvent
func (m *keyboardManager) Handle(ev keyEvent, send func(ui.KeyData) bool) (handled bool) {
	data, main := m.embedder.handle(ev)
	handled = main < 0
	for i, d := range data {
		answered := send != nil && send(d)
		if i == main {
			handled = answered
		}
	}
	return handled
}

// PressedState returns a copy of the keys that are pressed, physical key id to logical
// key id.
//
// Flutter: engine/src/flutter/shell/platform/android/io/flutter/embedding/android/KeyboardManager.java KeyboardManager.getKeyboardState
func (m *keyboardManager) PressedState() map[uint64]uint64 {
	return m.embedder.PressedState()
}
