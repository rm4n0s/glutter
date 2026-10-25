package windows

import (
	"sort"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go converter of the keyboard (SPEC §25.6): a port of Flutter's
// KeyboardKeyEmbedderHandler. A key message becomes the key data of the framework, with the
// synthesized events that keep the pressed keys and the lock states true. It calls nothing of
// Windows: keyHost stands for the few calls it needs, so a test runs it on any host.

// The planes of a key id: the low 32 bits are the value and the bits above them the plane.
//
// Flutter: shell/platform/windows/flutter_key_map.g.cc KeyboardKeyEmbedderHandler::valueMask
const (
	keyValueMask    uint64 = 0x000ffffffff
	keyUnicodePlane uint64 = 0x00000000000
	keyWindowsPlane uint64 = 0x01600000000
)

// The scan codes and virtual keys of the keys that the converter names.
//
// Flutter: shell/platform/windows/keyboard_utils.h kScanCodeShiftLeft and the others
const (
	scanShiftLeft    = 0x2a
	scanShiftRight   = 0x36
	scanControlLeft  = 0x1d
	scanControlRight = 0xe01d
)

// layoutGoal is a key whose logical key must not depend on the layout, so that shortcuts such as
// Control+C work on every layout (Flutter's layout_goals.json).
//
// Flutter: dev/tools/gen_keycodes/data/layout_goals.json
type layoutGoal struct {
	// scanCode is the Windows scan code of the key, in Chromium's form.
	scanCode uint32
	// logical is the id of the logical key on the US layout.
	logical uint64
	// mandatory goals fall back to logical on a layout that has no Latin character on the key.
	mandatory bool
}

// keyHost is what the converter and the manager need from Windows. The window code implements it
// with the calls of user32; the tests with a table.
//
// Flutter: no counterpart (the GetKeyStateHandler and MapVirtualKeyToScanCode of KeyboardKeyEmbedderHandler, and the window delegate of KeyboardManager)
type keyHost interface {
	// KeyState is GetKeyState: the high bit says the key is down, the low bit that a lock key is on.
	KeyState(vk uint32) int16
	// ScanCode is MapVirtualKey from a virtual key to its scan code, with the high byte 0xE0 for an
	// extended key when extended is set.
	ScanCode(vk uint32, extended bool) uint32
	// VirtualKeyOfScan is MapVirtualKey(MAPVK_VSC_TO_VK_EX) of a scan code, which tells the left key
	// from the right one.
	VirtualKeyOfScan(scan uint32) uint32
	// Layout is the handle of the active keyboard layout.
	Layout() uintptr
	// CharOfKey is the character that the key with the scan code makes on the active layout with no
	// modifier (MapVirtualKey(MAPVK_VK_TO_CHAR) of its virtual key), 0 for none; the mark of a dead
	// key may be set.
	CharOfKey(scan uint32) uint32
}

// criticalKey is a key whose state is checked after every event.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.h KeyboardKeyEmbedderHandler::CriticalKey
type criticalKey struct {
	// physical and logical are the ids last seen for the virtual key, to synthesize its events.
	physical, logical uint64
	// checkPressed says to ensure the pressed state of the key, checkToggled the toggled state of a
	// lock key (which implies checkPressed).
	checkPressed, checkToggled bool
	// toggledOn is whether the lock key is on as far as the framework knows.
	toggledOn bool
}

const (
	stateMaskToggled = 0x01
	stateMaskPressed = 0x80
)

// keyEmbedder turns key messages into key data. It keeps the state of the keyboard, so it is used
// on one goroutine, in the order of the messages.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler
type keyEmbedder struct {
	host keyHost
	now  func() time.Duration
	// pressing maps the pressed physical keys to their logical keys (pressingRecords_).
	pressing map[uint64]uint64
	// critical are the keys whose states are synchronized, in the order of their virtual keys
	// (a std::map in Flutter, so the order of the synthesized events is the same).
	critical     map[uint32]*criticalKey
	criticalKeys []uint32
	// goals are the layout goals by scan code, and layoutHandle and layoutChars the characters of
	// the keys on the layout they were read for.
	goals        map[uint32]layoutGoal
	layoutHandle uintptr
	layoutChars  map[uint32]uint32
	// out collects the key data of the message being handled.
	out []ui.KeyData
}

// newKeyEmbedder makes an embedder with no key pressed. It reads the lock states from the host.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::KeyboardKeyEmbedderHandler
func newKeyEmbedder(host keyHost, now func() time.Duration) *keyEmbedder {
	e := &keyEmbedder{
		host: host, now: now,
		pressing: map[uint64]uint64{},
		critical: map[uint32]*criticalKey{},
		goals:    map[uint32]layoutGoal{},
	}
	for _, g := range layoutGoals {
		e.goals[g.scanCode] = g
	}
	e.initCriticalKeys()
	return e
}

// initCriticalKeys assigns the keys whose states are checked after every event: the sides of Shift,
// Control, Alt and Windows, and the lock keys.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::InitCriticalKeys
func (e *keyEmbedder) initCriticalKeys() {
	create := func(vk uint32, extended, checkPressed, checkToggled bool) {
		scan := e.host.ScanCode(vk, extended)
		k := &criticalKey{
			physical:     e.physicalKey(scan, extended),
			logical:      e.logicalKey(vk, extended, scan),
			checkPressed: checkPressed || checkToggled,
			checkToggled: checkToggled,
		}
		if checkToggled {
			k.toggledOn = e.host.KeyState(vk)&stateMaskToggled != 0
		}
		e.critical[vk] = k
		e.criticalKeys = append(e.criticalKeys, vk)
	}
	create(win32.VKLShift, false, true, false)
	create(win32.VKRShift, false, true, false)
	create(win32.VKLControl, false, true, false)
	create(win32.VKRControl, true, true, false)
	create(win32.VKLMenu, false, true, false)
	create(win32.VKRMenu, true, true, false)
	create(win32.VKLWin, true, true, false)
	create(win32.VKRWin, true, true, false)
	create(win32.VKCapital, false, true, true)
	create(win32.VKScroll, false, true, true)
	create(win32.VKNumLock, true, true, true)
	sort.Slice(e.criticalKeys, func(i, j int) bool { return e.criticalKeys[i] < e.criticalKeys[j] })
}

// applyPlane puts the value of an id in a plane.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::ApplyPlaneToId
func applyPlane(id, plane uint64) uint64 { return id&keyValueMask | plane }

// isEasciiPrintable reports whether a virtual-key code is a printable character of ASCII or
// extended ASCII.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc isEasciiPrintable
func isEasciiPrintable(code uint32) bool {
	return (code <= 0x7f && code >= 0x20) || (code <= 0xff && code >= 0x80)
}

// asciiLower turns upper-case letters of ASCII and extended ASCII into lower case, and returns
// anything else as it is. It does not depend on the locale.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc toLower
func asciiLower(n uint64) uint64 {
	const lowerA, upperA, upperZ = 0x61, 0x41, 0x5a
	const lowerAGrave, upperAGrave, upperThorn, division = 0xe0, 0xc0, 0xde, 0xf7
	if n >= upperA && n <= upperZ {
		return n - upperA + lowerA
	}
	if n >= upperAGrave && n <= upperThorn && n != division {
		return n - upperAGrave + lowerAGrave
	}
	return n
}

// normalizeScanCode turns the scan code of Windows into the one of Chromium's specification: the
// extended bit is the prefix 0xE000 (PageUp is 0xe049).
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc normalizeScancode
func normalizeScanCode(scan uint32, extended bool) uint32 {
	n := scan & 0xff
	if extended {
		n |= 0xe000
	}
	return n
}

// physicalKey is the physical key of a scan code: the table, else the scan code in the Windows
// plane. It is never 0.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::GetPhysicalKey
func (e *keyEmbedder) physicalKey(scan uint32, extended bool) uint64 {
	if k, ok := scanCodeToPhysical[normalizeScanCode(scan, extended)]; ok {
		return k
	}
	return applyPlane(uint64(scan), keyWindowsPlane)
}

// vkProcessKey is the logical key id of a key that an input method took (VK_PROCESSKEY): the
// converter sends nothing for it.
const vkProcessKey = uint64(win32.VKProcessKey)

// logicalKey is the logical key of a virtual key, as Flutter's embedder makes it: some key codes
// are 0 or ambiguous (the numpad), so the scan code decides first; then the table of virtual
// keys; then the character of a printable key code in lower case; else the Windows plane. A key of
// the layout goals that the active layout gives a character outside Latin is the key of the US
// layout (SPEC §25.6). It is never 0.
//
// Deviation: Flutter's embedder has no layout goals on Windows (SPEC §12.10).
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::GetLogicalKey
func (e *keyEmbedder) logicalKey(vk uint32, extended bool, scan uint32) uint64 {
	if vk == win32.VKProcessKey {
		return vkProcessKey
	}
	norm := normalizeScanCode(scan, extended)
	if k, ok := scanCodeToLogical[norm]; ok {
		return k
	}
	if g, ok := e.goals[norm]; ok && g.mandatory && !e.latinOnKey(norm) {
		return g.logical
	}
	if k, ok := virtualKeyToLogical[vk]; ok {
		return k
	}
	if isEasciiPrintable(vk) {
		return applyPlane(asciiLower(uint64(vk)), keyUnicodePlane)
	}
	return applyPlane(asciiLower(uint64(vk)), keyWindowsPlane)
}

// latinLimit is where the Latin characters end: a character counts as Latin when it is ASCII (the
// rule of the Linux backends, SPEC §8.5).
const latinLimit = 0x80

// latinOnKey reports whether the active layout makes an ASCII character with the key at its first
// level. The characters are read once for each layout (WM_INPUTLANGCHANGE and the layout handle
// say when it changes).
func (e *keyEmbedder) latinOnKey(scan uint32) bool {
	if h := e.host.Layout(); h != e.layoutHandle || e.layoutChars == nil {
		e.layoutHandle, e.layoutChars = h, map[uint32]uint32{}
	}
	c, ok := e.layoutChars[scan]
	if !ok {
		c = undeadChar(e.host.CharOfKey(scan))
		e.layoutChars[scan] = c
	}
	return c > 0 && c < latinLimit
}

// layoutChanged forgets the characters of the keys: the layout is another one.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_INPUTLANGCHANGE)
func (e *keyEmbedder) layoutChanged() {
	e.layoutChars = nil
}

// keyMessage is a key message made ready for the converter.
type keyMessage struct {
	// key is the virtual key with the side of a modifier resolved, scan the scan code (8 bits) and
	// extended the extended flag.
	key      uint32
	scan     uint32
	extended bool
	// down is whether the message is a key down (WM_KEYDOWN, WM_SYSKEYDOWN), wasDown bit 30 of the
	// lParam: the key was down before the message (an auto-repeat).
	down, wasDown bool
	// text is the text the key makes, "" for none.
	text string
}

// hook converts one key message and returns the key data in order: the events synthesized before
// the main event, the main event, and those after it. main is the index of the main event, or -1
// when the message makes none (a key that an input method took, an up of a key that was not
// down): then the framework has nothing to answer and the message counts as handled. When no data
// was made at all, the data is the empty event of Flutter (both ids 0).
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::KeyboardHook
func (e *keyEmbedder) hook(m keyMessage) (data []ui.KeyData, main int) {
	e.out = nil
	main = e.hookImpl(m)
	if len(e.out) == 0 {
		e.out = append(e.out, ui.KeyData{TimeStamp: e.now(), Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard})
	}
	data = e.out
	e.out = nil
	return data, main
}

// hookImpl is the core algorithm of hook. It returns the index of the main event in e.out, -1 for none.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::KeyboardHookImpl
func (e *keyEmbedder) hookImpl(m keyMessage) int {
	physical := e.physicalKey(m.scan, m.extended)
	logical := e.logicalKey(m.key, m.extended, m.scan)

	lastRecord, had := e.pressing[physical]
	// The logical key for the current tap sequence: down, repeats, up keep one.
	sequenceLogical := logical
	if had {
		sequenceLogical = lastRecord
	}
	if sequenceLogical == vkProcessKey {
		// VK_PROCESSKEY means an input method used the press: it is handled and not sent. The key up
		// of such a press has the key's own logical key, and is filtered the same way.
		return -1
	}

	eventKeyCanBeRepeat := true
	e.updateLastSeenCriticalKey(m.key, physical, sequenceLogical)
	// Toggled states can only change on a down event: when the recorded state is not the true one,
	// synthesize (an up if the key is recorded pressed, then) a down.
	e.synchronizeCriticalToggledStates(m.key, m.down, &eventKeyCanBeRepeat)
	e.synchronizeCriticalPressedStates(m.key, physical, m.down, eventKeyCanBeRepeat)

	// The records may have changed.
	lastRecord, had = e.pressing[physical]

	var typ ui.KeyEventType
	var text string
	var eventualLogical, resultLogical uint64
	if m.down {
		if had && !m.wasDown {
			// A down without the up before it: the window lost the focus before it got the key up.
			// The framework releases the key before it processes the down.
			e.sendSynthesizedUp(physical, lastRecord)
			lastRecord, had = e.pressing[physical]
		}
		text = m.text
		if had {
			// A repeat.
			typ = ui.KeyEventTypeRepeat
			eventualLogical, resultLogical = lastRecord, lastRecord
		} else {
			typ = ui.KeyEventTypeDown
			eventualLogical, resultLogical = logical, logical
		}
	} else {
		if !had {
			// The key was released before: a missed event after a loss of focus, or two keyboards
			// pressing the same physical key. The up is ignored.
			return -1
		}
		typ = ui.KeyEventTypeUp
		eventualLogical, resultLogical = 0, lastRecord
	}

	if eventualLogical != 0 {
		e.pressing[physical] = eventualLogical
	} else {
		delete(e.pressing, physical)
	}

	e.out = append(e.out, ui.KeyData{
		TimeStamp: e.now(), Type: typ, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: physical, Logical: resultLogical, Character: text,
	})
	main := len(e.out) - 1

	// After the event: a CapsLock down event with GetKeyState saying it is not pressed gets a
	// synthesized up now.
	e.synchronizeCriticalPressedStates(m.key, physical, m.down, eventKeyCanBeRepeat)
	return main
}

// updateLastSeenCriticalKey keeps the ids last seen for a critical virtual key.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::UpdateLastSeenCriticalKey
func (e *keyEmbedder) updateLastSeenCriticalKey(vk uint32, physical, logical uint64) {
	if k, ok := e.critical[vk]; ok {
		k.physical, k.logical = physical, logical
	}
}

// synchronizeCriticalToggledStates checks the toggled state of the lock keys and synthesizes the
// events when the framework's state is not the true one.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SynchronizeCriticalToggledStates
func (e *keyEmbedder) synchronizeCriticalToggledStates(eventVK uint32, isEventDown bool, eventKeyCanBeRepeat *bool) {
	// NowState -> PreEventState -> TrueState, with the synchronization before the event.
	for _, vk := range e.criticalKeys {
		k := e.critical[vk]
		if k.physical == 0 || !k.checkToggled {
			continue
		}
		_, targetPressed := e.pressing[k.physical]
		// The lock keys go through four phases: down (pressed, toggled), up (released, toggled),
		// down (pressed, untoggled) and up.
		trueToggled := e.host.KeyState(vk)&stateMaskToggled != 0
		preEventToggled := trueToggled
		// A down event of this key that is not a repeat toggles the state.
		if vk == eventVK && !targetPressed && isEventDown {
			preEventToggled = !preEventToggled
		}
		if k.toggledOn != preEventToggled {
			if targetPressed {
				e.synthesize(ui.KeyEventTypeUp, k.physical, k.logical)
			}
			// Synchronizing the toggle state always ends with the key pressed.
			e.pressing[k.physical] = k.logical
			e.synthesize(ui.KeyEventTypeDown, k.physical, k.logical)
			*eventKeyCanBeRepeat = false
		}
		k.toggledOn = trueToggled
	}
}

// synchronizeCriticalPressedStates checks the pressed state of the critical keys and synthesizes
// the events that make the framework's records the true ones, taking the event into account.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SynchronizeCriticalPressedStates
func (e *keyEmbedder) synchronizeCriticalPressedStates(eventVK uint32, eventPhysical uint64, isEventDown, eventKeyCanBeRepeat bool) {
	for _, vk := range e.criticalKeys {
		k := e.critical[vk]
		if k.physical == 0 || !k.checkPressed {
			continue
		}
		truePressed := e.host.KeyState(vk)&stateMaskPressed != 0
		_, nowPressed := e.pressing[k.physical]
		preEventPressed := truePressed
		if isEventDown {
			// For a down event the key is the event's when the virtual keys are the same, because the
			// virtual key stands for functionality. Normally nothing is synthesized, since the event
			// itself can be a down or a repeat; but not when the toggled state was just synchronized.
			if vk == eventVK {
				if eventKeyCanBeRepeat {
					continue
				}
				preEventPressed = false
			}
		} else if eventPhysical == k.physical {
			// For an up event the key is the event's when the physical keys are the same. The key
			// should have been pressed, but no down is synthesized if it was not (the abrupt up is
			// skipped later), and no up either.
			continue
		}
		if nowPressed != preEventPressed {
			if nowPressed {
				delete(e.pressing, k.physical)
				e.synthesize(ui.KeyEventTypeUp, k.physical, k.logical)
			} else {
				e.pressing[k.physical] = k.logical
				e.synthesize(ui.KeyEventTypeDown, k.physical, k.logical)
			}
		}
	}
}

// synthesize appends a synthesized event with no character.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SynthesizeSimpleEvent
func (e *keyEmbedder) synthesize(typ ui.KeyEventType, physical, logical uint64) {
	e.out = append(e.out, ui.KeyData{
		TimeStamp: e.now(), Type: typ, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: physical, Logical: logical, Synthesized: true,
	})
}

// sendSynthesizedDown appends a synthesized down and records the key as pressed.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SendSynthesizeDownEvent
func (e *keyEmbedder) sendSynthesizedDown(physical, logical uint64) {
	e.synthesize(ui.KeyEventTypeDown, physical, logical)
	e.pressing[physical] = logical
}

// sendSynthesizedUp appends a synthesized up and records the key as released.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SendSynthesizeUpEvent
func (e *keyEmbedder) sendSynthesizedUp(physical, logical uint64) {
	e.synthesize(ui.KeyEventTypeUp, physical, logical)
	delete(e.pressing, physical)
}

// syncModifiers makes the pressed state of Shift and Control match the modifiers that a mouse
// message reports, synthesizing the key events that are missing. The message of a mouse move has
// the state of the modifiers at the time of the move, which a window can miss while it is not the
// one that gets the keys.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SyncModifiersIfNeeded
func (e *keyEmbedder) syncModifiers(shift, control bool) []ui.KeyData {
	e.out = nil
	e.synthesizeIfNeeded(scanCodeToPhysical[scanShiftLeft], scanCodeToPhysical[scanShiftRight],
		virtualKeyToLogical[win32.VKLShift], shift)
	e.synthesizeIfNeeded(scanCodeToPhysical[scanControlLeft], scanCodeToPhysical[scanControlRight],
		virtualKeyToLogical[win32.VKLControl], control)
	data := e.out
	e.out = nil
	return data
}

// synthesizeIfNeeded presses or releases a modifier whose two sides are not in the state wanted.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::SynthesizeIfNeeded
func (e *keyEmbedder) synthesizeIfNeeded(physicalLeft, physicalRight, logicalLeft uint64, pressed bool) {
	knownLeft, leftPressed := e.pressing[physicalLeft]
	knownRight, rightPressed := e.pressing[physicalRight]
	already := leftPressed || rightPressed
	if pressed && !already {
		e.sendSynthesizedDown(physicalLeft, logicalLeft)
	}
	if !pressed && already && leftPressed {
		e.sendSynthesizedUp(physicalLeft, knownLeft)
	}
	if !pressed && already && rightPressed {
		e.sendSynthesizedUp(physicalRight, knownRight)
	}
}

// releaseAll releases every key that is down, with synthesized up events, in the order of the
// physical keys. A window that loses the keyboard focus does not get the key ups of the keys held
// down, and the framework must not think they are still held (SPEC §25.6).
//
// Flutter: no counterpart (the embedder finds the loss when the next down comes without bit 30 of lParam)
func (e *keyEmbedder) releaseAll() []ui.KeyData {
	e.out = nil
	physical := make([]uint64, 0, len(e.pressing))
	for p := range e.pressing {
		physical = append(physical, p)
	}
	sort.Slice(physical, func(i, j int) bool { return physical[i] < physical[j] })
	for _, p := range physical {
		e.sendSynthesizedUp(p, e.pressing[p])
	}
	data := e.out
	e.out = nil
	return data
}

// syncLocks reads the state of the lock keys and the modifiers again and synthesizes the events
// that make the framework's records true: the key presses that happened while the window had no
// focus are not seen otherwise until the next key (SPEC §25.6).
//
// Flutter: no counterpart (the embedder does it with the next key event)
func (e *keyEmbedder) syncLocks() []ui.KeyData {
	e.out = nil
	canRepeat := true
	e.synchronizeCriticalToggledStates(0, false, &canRepeat)
	e.synchronizeCriticalPressedStates(0, 0, false, true)
	data := e.out
	e.out = nil
	return data
}

// pressedState returns a copy of the keys that are pressed: the ids of the physical keys mapped to
// the ids of their logical keys.
//
// Flutter: shell/platform/windows/keyboard_key_embedder_handler.cc KeyboardKeyEmbedderHandler::GetPressedState
func (e *keyEmbedder) pressedState() map[uint64]uint64 {
	state := make(map[uint64]uint64, len(e.pressing))
	for p, l := range e.pressing {
		state[p] = l
	}
	return state
}
