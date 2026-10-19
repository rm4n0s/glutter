package services

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// KeyboardLockMode is a lock mode of the keyboard: a mode that a lock key (Num
// Lock, Scroll Lock, Caps Lock) switches on and off. The zero value,
// KeyboardLockModeUnspecified, is no lock mode.
//
// Flutter: services/hardware_keyboard.dart KeyboardLockMode
type KeyboardLockMode int

// The values of KeyboardLockMode follow Flutter's declaration order.
const (
	// KeyboardLockModeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	KeyboardLockModeUnspecified KeyboardLockMode = iota
	// KeyboardLockModeNumLock is the Num Lock mode, switched by the Num Lock key.
	//
	// Flutter: services/hardware_keyboard.dart KeyboardLockMode.numLock
	KeyboardLockModeNumLock
	// KeyboardLockModeScrollLock is the Scroll Lock mode, switched by the Scroll
	// Lock key.
	//
	// Flutter: services/hardware_keyboard.dart KeyboardLockMode.scrollLock
	KeyboardLockModeScrollLock
	// KeyboardLockModeCapsLock is the Caps Lock mode, switched by the Caps Lock
	// key.
	//
	// Flutter: services/hardware_keyboard.dart KeyboardLockMode.capsLock
	KeyboardLockModeCapsLock
)

// LogicalKey is the key that switches the lock mode.
//
// Flutter: services/hardware_keyboard.dart KeyboardLockMode.logicalKey
func (m KeyboardLockMode) LogicalKey() LogicalKeyboardKey {
	switch m {
	case KeyboardLockModeNumLock:
		return LogicalKeyboardKeyNumLock
	case KeyboardLockModeScrollLock:
		return LogicalKeyboardKeyScrollLock
	case KeyboardLockModeCapsLock:
		return LogicalKeyboardKeyCapsLock
	}
	return 0
}

// String is the name of the lock mode.
//
// Flutter: services/hardware_keyboard.dart KeyboardLockMode
func (m KeyboardLockMode) String() string {
	switch m {
	case KeyboardLockModeNumLock:
		return "KeyboardLockMode.numLock"
	case KeyboardLockModeScrollLock:
		return "KeyboardLockMode.scrollLock"
	case KeyboardLockModeCapsLock:
		return "KeyboardLockMode.capsLock"
	}
	return "KeyboardLockMode.unspecified"
}

// FindKeyboardLockModeByLogicalKey returns the lock mode that the key switches,
// and false when it switches none.
//
// Flutter: services/hardware_keyboard.dart KeyboardLockMode.findLockByLogicalKey
func FindKeyboardLockModeByLogicalKey(key LogicalKeyboardKey) (KeyboardLockMode, bool) {
	switch key {
	case LogicalKeyboardKeyNumLock:
		return KeyboardLockModeNumLock, true
	case LogicalKeyboardKeyScrollLock:
		return KeyboardLockModeScrollLock, true
	case LogicalKeyboardKeyCapsLock:
		return KeyboardLockModeCapsLock, true
	}
	return KeyboardLockModeUnspecified, false
}

// KeyEvent is an event that a key of a keyboard sent: a KeyDownEvent, a
// KeyRepeatEvent or a KeyUpEvent. Compare the events by their type.
//
// Flutter: services/hardware_keyboard.dart KeyEvent
type KeyEvent interface {
	foundation.Diagnosticable
	// PhysicalKey is the key that was pressed or released, independent of the
	// layout.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.physicalKey
	PhysicalKey() PhysicalKeyboardKey
	// LogicalKey is the key interpreted with the layout and the modifiers.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.logicalKey
	LogicalKey() LogicalKeyboardKey
	// Character is the text that the key produced, or an empty string. A release
	// has none.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.character
	Character() string
	// TimeStamp is the time at which the event was generated, since an arbitrary
	// fixed point in the past.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.timeStamp
	TimeStamp() time.Duration
	// DeviceType is the device that produced the event.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.deviceType
	DeviceType() ui.KeyEventDeviceType
	// Synthesized is whether the engine made the event to keep the pressed keys
	// consistent, and not the user.
	//
	// Flutter: services/hardware_keyboard.dart KeyEvent.synthesized
	Synthesized() bool
}

// keyEventBase holds the fields that every key event has.
type keyEventBase struct {
	physicalKey PhysicalKeyboardKey
	logicalKey  LogicalKeyboardKey
	character   string
	timeStamp   time.Duration
	deviceType  ui.KeyEventDeviceType
	synthesized bool
}

func (e *keyEventBase) PhysicalKey() PhysicalKeyboardKey { return e.physicalKey }
func (e *keyEventBase) LogicalKey() LogicalKeyboardKey   { return e.logicalKey }
func (e *keyEventBase) Character() string                { return e.character }
func (e *keyEventBase) TimeStamp() time.Duration         { return e.timeStamp }
func (e *keyEventBase) Synthesized() bool                { return e.synthesized }
func (e *keyEventBase) DeviceType() ui.KeyEventDeviceType {
	if e.deviceType == ui.KeyEventDeviceTypeUnspecified {
		return ui.KeyEventDeviceTypeKeyboard
	}
	return e.deviceType
}

func (e *keyEventBase) describe(kind string) string {
	text := fmt.Sprintf("%s(physicalKey: %v, logicalKey: %v", kind, e.physicalKey, e.logicalKey)
	if e.character != "" {
		text += fmt.Sprintf(", character: %q", e.character)
	}
	text += fmt.Sprintf(", timeStamp: %v", e.timeStamp)
	if e.synthesized {
		text += ", synthesized"
	}
	return text + ")"
}

// DebugFillProperties adds the keys, the character, the time stamp and the
// synthesized flag.
//
// Flutter: services/hardware_keyboard.dart KeyEvent.debugFillProperties
func (e *keyEventBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("physicalKey", e.physicalKey))
	p.Add(foundation.NewDiagnosticsProperty("logicalKey", e.logicalKey))
	if e.character == "" {
		p.Add(foundation.NewStringProperty("character", foundation.None[string]()))
	} else {
		p.Add(foundation.NewStringProperty("character", foundation.Some(e.character)))
	}
	p.Add(foundation.NewDiagnosticsProperty("timeStamp", e.timeStamp))
	p.Add(foundation.NewFlagProperty("synthesized", foundation.Some(e.synthesized), foundation.FlagPropertyOpts{IfTrue: "synthesized"}))
}

// KeyDownEvent is the event of a key that was pressed.
//
// Flutter: services/hardware_keyboard.dart KeyDownEvent
type KeyDownEvent struct{ keyEventBase }

// String describes the event for logs and test failures.
//
// Flutter: services/hardware_keyboard.dart KeyEvent.toString
func (e *KeyDownEvent) String() string { return e.describe("KeyDownEvent") }

// KeyDownEventOpts are the optional parameters of NewKeyDownEvent.
//
// Flutter: services/hardware_keyboard.dart KeyDownEvent.new
type KeyDownEventOpts struct {
	// Character is the text that the key produced.
	Character string
	// Synthesized is whether the engine made the event.
	Synthesized bool
	// DeviceType is the device of the event; the default is a keyboard.
	DeviceType ui.KeyEventDeviceType
}

// NewKeyDownEvent creates the event of a press.
//
// Flutter: services/hardware_keyboard.dart KeyDownEvent.new
func NewKeyDownEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyDownEventOpts) *KeyDownEvent {
	o := firstKeyOpts(opts)
	return &KeyDownEvent{keyEventBase{physicalKey, logicalKey, o.Character, timeStamp, o.DeviceType, o.Synthesized}}
}

// KeyUpEvent is the event of a key that was released.
//
// Flutter: services/hardware_keyboard.dart KeyUpEvent
type KeyUpEvent struct{ keyEventBase }

// String describes the event for logs and test failures.
//
// Flutter: services/hardware_keyboard.dart KeyEvent.toString
func (e *KeyUpEvent) String() string { return e.describe("KeyUpEvent") }

// KeyUpEventOpts are the optional parameters of NewKeyUpEvent.
//
// Flutter: services/hardware_keyboard.dart KeyUpEvent.new
type KeyUpEventOpts struct {
	// Synthesized is whether the engine made the event.
	Synthesized bool
	// DeviceType is the device of the event; the default is a keyboard.
	DeviceType ui.KeyEventDeviceType
}

// NewKeyUpEvent creates the event of a release.
//
// Flutter: services/hardware_keyboard.dart KeyUpEvent.new
func NewKeyUpEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyUpEventOpts) *KeyUpEvent {
	o := firstKeyOpts(opts)
	return &KeyUpEvent{keyEventBase{physicalKey, logicalKey, "", timeStamp, o.DeviceType, o.Synthesized}}
}

// KeyRepeatEvent is the event of a key that is held down and repeats.
//
// Flutter: services/hardware_keyboard.dart KeyRepeatEvent
type KeyRepeatEvent struct{ keyEventBase }

// String describes the event for logs and test failures.
//
// Flutter: services/hardware_keyboard.dart KeyEvent.toString
func (e *KeyRepeatEvent) String() string { return e.describe("KeyRepeatEvent") }

// KeyRepeatEventOpts are the optional parameters of NewKeyRepeatEvent.
//
// Flutter: services/hardware_keyboard.dart KeyRepeatEvent.new
type KeyRepeatEventOpts struct {
	// Character is the text that the key produced.
	Character string
	// DeviceType is the device of the event; the default is a keyboard.
	DeviceType ui.KeyEventDeviceType
}

// NewKeyRepeatEvent creates the event of a repeat.
//
// Flutter: services/hardware_keyboard.dart KeyRepeatEvent.new
func NewKeyRepeatEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyRepeatEventOpts) *KeyRepeatEvent {
	o := firstKeyOpts(opts)
	return &KeyRepeatEvent{keyEventBase{physicalKey, logicalKey, o.Character, timeStamp, o.DeviceType, false}}
}

func firstKeyOpts[T any](opts []T) T {
	var zero T
	switch len(opts) {
	case 0:
		return zero
	case 1:
		return opts[0]
	}
	panic(errors.New("MultipleOptions", "pass at most one options value", "type", fmt.Sprintf("%T", zero)))
}

// KeyEventFromData makes the key event that the engine's key data describes.
//
// Flutter: services/hardware_keyboard.dart KeyEventManager._eventFromData
func KeyEventFromData(data ui.KeyData) (KeyEvent, error) {
	physical, ok := FindPhysicalKeyByCode(data.Physical)
	if !ok {
		physical = PhysicalKeyboardKey(data.Physical)
	}
	logical, ok := FindLogicalKeyByKeyID(data.Logical)
	if !ok {
		logical = LogicalKeyboardKey(data.Logical)
	}
	switch data.Type {
	case ui.KeyEventTypeDown:
		return NewKeyDownEvent(physical, logical, data.TimeStamp, KeyDownEventOpts{Character: data.Character, Synthesized: data.Synthesized, DeviceType: data.DeviceType}), nil
	case ui.KeyEventTypeUp:
		return NewKeyUpEvent(physical, logical, data.TimeStamp, KeyUpEventOpts{Synthesized: data.Synthesized, DeviceType: data.DeviceType}), nil
	case ui.KeyEventTypeRepeat:
		return NewKeyRepeatEvent(physical, logical, data.TimeStamp, KeyRepeatEventOpts{Character: data.Character, DeviceType: data.DeviceType}), nil
	}
	return nil, errors.New("KeyDataTypeInvalid", "the key data has no event type", "type", data.Type.Label())
}

// KeyEventCallback handles a key event and returns whether it did.
//
// Flutter: services/hardware_keyboard.dart KeyEventCallback
type KeyEventCallback func(event KeyEvent) bool

// KeyHandlerHandle identifies a handler that was added to a HardwareKeyboard,
// to remove it.
//
// Flutter: no counterpart
type KeyHandlerHandle struct{ id uint64 }

type keyHandlerEntry struct {
	id uint64
	fn KeyEventCallback
}

// HardwareKeyboardOpts are the optional parameters of NewHardwareKeyboard.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.new
type HardwareKeyboardOpts struct {
	// ErrorReporter receives the panics of the handlers; without it they are not
	// recovered.
	ErrorReporter foundation.ErrorReporter
	// DebugPrintKeyboardEvents says, for each event, whether the keyboard prints
	// what it does with it, and the irregular events as errors (in debug
	// builds). It is a function so that the flag can change while the app runs.
	DebugPrintKeyboardEvents func() bool
	// DebugPrint prints the debug messages; the default is to drop them.
	DebugPrint func(message string)
}

// HardwareKeyboard is the state of the keyboard: the keys that are pressed, the
// lock modes that are on, and the handlers that see the events first. The
// binding makes one and feeds it the key events of the engine. All calls are on
// the UI thread.
//
// Deviation: Flutter reads HardwareKeyboard.instance; Glutter has no singletons,
// so the keyboard is reached through the binding (SPEC §12.10).
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard
type HardwareKeyboard struct {
	o                HardwareKeyboardOpts
	pressedKeys      map[PhysicalKeyboardKey]LogicalKeyboardKey
	lockModes        KeySet[KeyboardLockMode]
	handlers         []keyHandlerEntry
	duringDispatch   bool
	modifiedHandlers []keyHandlerEntry
	hasModified      bool
	nextHandlerID    uint64
}

// NewHardwareKeyboard creates a keyboard with no keys pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.new
func NewHardwareKeyboard(opts ...HardwareKeyboardOpts) *HardwareKeyboard {
	return &HardwareKeyboard{
		o:           firstKeyOpts(opts),
		pressedKeys: map[PhysicalKeyboardKey]LogicalKeyboardKey{},
		lockModes:   KeySet[KeyboardLockMode]{},
	}
}

// PhysicalKeysPressed is the set of the physical keys that are pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.physicalKeysPressed
func (k *HardwareKeyboard) PhysicalKeysPressed() KeySet[PhysicalKeyboardKey] {
	out := make(KeySet[PhysicalKeyboardKey], len(k.pressedKeys))
	for p := range k.pressedKeys {
		out[p] = struct{}{}
	}
	return out
}

// LogicalKeysPressed is the set of the logical keys that are pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.logicalKeysPressed
func (k *HardwareKeyboard) LogicalKeysPressed() KeySet[LogicalKeyboardKey] {
	out := make(KeySet[LogicalKeyboardKey], len(k.pressedKeys))
	for _, l := range k.pressedKeys {
		out[l] = struct{}{}
	}
	return out
}

// LookUpLayout is the logical key that a pressed physical key produces, and false
// when the key is not pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.lookUpLayout
func (k *HardwareKeyboard) LookUpLayout(physicalKey PhysicalKeyboardKey) (LogicalKeyboardKey, bool) {
	l, ok := k.pressedKeys[physicalKey]
	return l, ok
}

// LockModesEnabled is the set of the lock modes that are on.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.lockModesEnabled
func (k *HardwareKeyboard) LockModesEnabled() KeySet[KeyboardLockMode] { return k.lockModes.Clone() }

// IsLogicalKeyPressed is whether the logical key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isLogicalKeyPressed
func (k *HardwareKeyboard) IsLogicalKeyPressed(key LogicalKeyboardKey) bool {
	for _, l := range k.pressedKeys {
		if l == key {
			return true
		}
	}
	return false
}

// IsPhysicalKeyPressed is whether the physical key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isPhysicalKeyPressed
func (k *HardwareKeyboard) IsPhysicalKeyPressed(key PhysicalKeyboardKey) bool {
	_, ok := k.pressedKeys[key]
	return ok
}

// IsControlPressed is whether a control key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isControlPressed
func (k *HardwareKeyboard) IsControlPressed() bool {
	return k.IsLogicalKeyPressed(LogicalKeyboardKeyControlLeft) || k.IsLogicalKeyPressed(LogicalKeyboardKeyControlRight)
}

// IsShiftPressed is whether a shift key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isShiftPressed
func (k *HardwareKeyboard) IsShiftPressed() bool {
	return k.IsLogicalKeyPressed(LogicalKeyboardKeyShiftLeft) || k.IsLogicalKeyPressed(LogicalKeyboardKeyShiftRight)
}

// IsAltPressed is whether an alt key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isAltPressed
func (k *HardwareKeyboard) IsAltPressed() bool {
	return k.IsLogicalKeyPressed(LogicalKeyboardKeyAltLeft) || k.IsLogicalKeyPressed(LogicalKeyboardKeyAltRight)
}

// IsMetaPressed is whether a meta key is pressed.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.isMetaPressed
func (k *HardwareKeyboard) IsMetaPressed() bool {
	return k.IsLogicalKeyPressed(LogicalKeyboardKeyMetaLeft) || k.IsLogicalKeyPressed(LogicalKeyboardKeyMetaRight)
}

func (k *HardwareKeyboard) debugPrinting() bool {
	return foundation.DebugMode && k.o.DebugPrintKeyboardEvents != nil && k.o.DebugPrintKeyboardEvents() && k.o.DebugPrint != nil
}

// keyboardDebug prints a debug message when the option is on. The message is
// made by a function, so that nothing is formatted when it is off.
func (k *HardwareKeyboard) keyboardDebug(message func() string, details func() []string) {
	if !k.debugPrinting() {
		return
	}
	k.o.DebugPrint("KEYBOARD: " + message())
	if details != nil {
		for _, d := range details() {
			k.o.DebugPrint("    " + d)
		}
	}
}

// logEventIfIrregular prints a debug message if the event does not fit the
// current state.
func (k *HardwareKeyboard) logEventIfIrregular(event KeyEvent) {
	if !k.debugPrinting() {
		return
	}
	const common = "This is typically either due to https://github.com/flutter/flutter/issues/125975, " +
		"or a bug in the embedding's key event conciliation logic."
	physical := event.PhysicalKey()
	pressed, isPressed := k.pressedKeys[physical]
	switch event.(type) {
	case *KeyDownEvent:
		if isPressed {
			k.keyboardDebug(func() string {
				return fmt.Sprintf("ERROR: Received unexpected %T for key that is already pressed.\n%s\n    Event: %v\n    Pressed logical key: %v", event, common, event, pressed)
			}, nil)
		}
	case *KeyRepeatEvent, *KeyUpEvent:
		switch {
		case !isPressed:
			k.keyboardDebug(func() string {
				return fmt.Sprintf("ERROR: Received unexpected %T for key that is not pressed:\n%s\n    Event: %v", event, common, event)
			}, nil)
		case pressed != event.LogicalKey():
			k.keyboardDebug(func() string {
				return fmt.Sprintf("ERROR: Received unexpected %T for key with mismatched logical key:\n%s\n    Event: %v\n    Pressed logical key: %v", event, common, event, pressed)
			}, nil)
		}
	}
}

// AddHandler adds a handler that sees the events before the framework's own
// handlers do; the keyboard calls the handlers in the order they were added.
// Adding one during a dispatch takes effect after it. The handle removes it.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.addHandler
func (k *HardwareKeyboard) AddHandler(handler KeyEventCallback) KeyHandlerHandle {
	k.nextHandlerID++
	entry := keyHandlerEntry{k.nextHandlerID, handler}
	if k.duringDispatch {
		if !k.hasModified {
			k.modifiedHandlers = slices.Clone(k.handlers)
			k.hasModified = true
		}
		k.modifiedHandlers = append(k.modifiedHandlers, entry)
	} else {
		k.handlers = append(k.handlers, entry)
	}
	return KeyHandlerHandle{entry.id}
}

// RemoveHandler removes the handler of the handle; removing during a dispatch
// takes effect after it.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.removeHandler
func (k *HardwareKeyboard) RemoveHandler(handle KeyHandlerHandle) {
	drop := func(list []keyHandlerEntry) []keyHandlerEntry {
		return slices.DeleteFunc(list, func(e keyHandlerEntry) bool { return e.id == handle.id })
	}
	if k.duringDispatch {
		if !k.hasModified {
			k.modifiedHandlers = slices.Clone(k.handlers)
			k.hasModified = true
		}
		k.modifiedHandlers = drop(k.modifiedHandlers)
	} else {
		k.handlers = drop(k.handlers)
	}
}

func (k *HardwareKeyboard) dispatchKeyEvent(event KeyEvent) bool {
	if k.duringDispatch {
		panic(errors.New("KeyboardDispatchNested", "nested keyboard dispatching is not supported"))
	}
	k.duringDispatch = true
	// The deferred function also runs when a handler panics and the keyboard has
	// no error reporter, so that the keyboard stays usable.
	defer func() {
		k.duringDispatch = false
		if k.hasModified {
			k.handlers = k.modifiedHandlers
			k.modifiedHandlers = nil
			k.hasModified = false
		}
	}()
	handled := false
	for _, h := range slices.Clone(k.handlers) {
		foundation.Guard(k.o.ErrorReporter, foundation.CallbackPanicked, "while processing a key handler",
			func() []any { return []any{"library", "services library", "Event", fmt.Sprint(event)} },
			func() {
				if h.fn(event) {
					handled = true
				}
			})
	}
	return handled
}

func (k *HardwareKeyboard) debugPressedKeysDetails() []string {
	if len(k.pressedKeys) == 0 {
		return []string{"Empty"}
	}
	physical := make([]PhysicalKeyboardKey, 0, len(k.pressedKeys))
	for p := range k.pressedKeys {
		physical = append(physical, p)
	}
	sort.Slice(physical, func(i, j int) bool { return physical[i] < physical[j] })
	out := make([]string, len(physical))
	for i, p := range physical {
		out[i] = fmt.Sprintf("%v: %v", p, k.pressedKeys[p])
	}
	return out
}

// HandleKeyEvent records the event in the state of the keyboard, then gives it
// to the handlers. It returns whether one of them handled it.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.handleKeyEvent
func (k *HardwareKeyboard) HandleKeyEvent(event KeyEvent) bool {
	k.keyboardDebug(func() string { return fmt.Sprintf("Key event received: %v", event) }, nil)
	k.keyboardDebug(func() string { return "Pressed state before processing the event:" }, k.debugPressedKeysDetails)
	k.logEventIfIrregular(event)
	physical, logical := event.PhysicalKey(), event.LogicalKey()
	switch event.(type) {
	case *KeyDownEvent:
		k.pressedKeys[physical] = logical
		if mode, ok := FindKeyboardLockModeByLogicalKey(logical); ok {
			if k.lockModes.Contains(mode) {
				delete(k.lockModes, mode)
			} else {
				k.lockModes[mode] = struct{}{}
			}
		}
	case *KeyUpEvent:
		delete(k.pressedKeys, physical)
	case *KeyRepeatEvent:
		// Updates the logical key in case it has changed.
		k.pressedKeys[physical] = logical
	}
	k.keyboardDebug(func() string { return "Pressed state after processing the event:" }, k.debugPressedKeysDetails)
	return k.dispatchKeyEvent(event)
}

// HandleKeyData makes the event of the engine's key data and handles it. The
// engine sends key data with no physical and no logical key to say that it
// sends key data at all; Flutter ignores it, and so does this: it returns false.
//
// Deviation: Flutter's KeyEventManager holds the key data until a raw key event
// arrives, because the embedders send both; Glutter's platforms send key data
// only, so each event is handled at once (SPEC §12.10).
//
// Flutter: services/hardware_keyboard.dart KeyEventManager.handleKeyData
func (k *HardwareKeyboard) HandleKeyData(data ui.KeyData) bool {
	if data.Physical == 0 && data.Logical == 0 {
		return false
	}
	event, err := KeyEventFromData(data)
	if err != nil {
		if k.o.ErrorReporter != nil {
			k.o.ErrorReporter(err)
			return false
		}
		panic(err)
	}
	return k.HandleKeyEvent(event)
}

// SetPressedKeys replaces the pressed keys with the state that the platform
// reports, for example after the window got the focus.
//
// Deviation: Flutter's syncKeyboardState asks the embedder through a platform
// channel; the Glutter platform layer synthesizes the key events on focus
// changes instead (SPEC §8.5), and this lets a host that knows the state set it.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.syncKeyboardState
func (k *HardwareKeyboard) SetPressedKeys(state map[PhysicalKeyboardKey]LogicalKeyboardKey) {
	for p, l := range state {
		k.pressedKeys[p] = l
	}
}

// ClearState forgets the pressed keys, the lock modes and the handlers.
//
// Flutter: services/hardware_keyboard.dart HardwareKeyboard.clearState
func (k *HardwareKeyboard) ClearState() {
	clear(k.pressedKeys)
	clear(k.lockModes)
	k.handlers = nil
	if k.hasModified {
		panic(errors.New("KeyboardClearedDuringDispatch", "the keyboard was cleared during a dispatch"))
	}
}
