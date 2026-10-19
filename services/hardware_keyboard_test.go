package services

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/ui"
)

// These tests are ported from packages/flutter/test/services/hardware_keyboard_test.dart.
// The key simulator of flutter_test is the key data that the engine would send;
// the cases that need the raw key events (RawKeyboard, the transit modes and the
// KeyMessage handler) or the Focus widgets are not ported: Glutter has no raw
// key events, and the focus handlers come with step 10.5.

// simulate sends the key data of an event to the keyboard and returns whether it
// was handled.
func simulate(k *HardwareKeyboard, typ ui.KeyEventType, physical PhysicalKeyboardKey, logical LogicalKeyboardKey) bool {
	return k.HandleKeyData(ui.KeyData{Type: typ, Physical: uint64(physical), Logical: uint64(logical)})
}

func expectState(t *testing.T, k *HardwareKeyboard, want map[PhysicalKeyboardKey]LogicalKeyboardKey) {
	t.Helper()
	wantPhysical, wantLogical := KeySet[PhysicalKeyboardKey]{}, KeySet[LogicalKeyboardKey]{}
	for p, l := range want {
		wantPhysical[p] = struct{}{}
		wantLogical[l] = struct{}{}
	}
	if got := k.PhysicalKeysPressed(); !got.Equal(wantPhysical) {
		t.Errorf("physical keys pressed = %v, want %v", got, wantPhysical)
	}
	if got := k.LogicalKeysPressed(); !got.Equal(wantLogical) {
		t.Errorf("logical keys pressed = %v, want %v", got, wantLogical)
	}
}

func TestHardwareKeyboardRecordsPressedKeysAndEnabledLocks(t *testing.T) {
	k := NewHardwareKeyboard()
	const (
		numLock = PhysicalKeyboardKeyNumLock
		numpad1 = PhysicalKeyboardKeyNumpad1
	)
	steps := []struct {
		name     string
		typ      ui.KeyEventType
		physical PhysicalKeyboardKey
		logical  LogicalKeyboardKey
		pressed  map[PhysicalKeyboardKey]LogicalKeyboardKey
		locks    KeySet[KeyboardLockMode]
	}{
		{"num lock down", ui.KeyEventTypeDown, numLock, LogicalKeyboardKeyNumLock,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numLock: LogicalKeyboardKeyNumLock}, NewKeySet(KeyboardLockModeNumLock)},
		{"numpad 1 down", ui.KeyEventTypeDown, numpad1, LogicalKeyboardKeyNumpad1,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numLock: LogicalKeyboardKeyNumLock, numpad1: LogicalKeyboardKeyNumpad1}, NewKeySet(KeyboardLockModeNumLock)},
		{"numpad 1 repeat", ui.KeyEventTypeRepeat, numpad1, LogicalKeyboardKeyNumpad1,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numLock: LogicalKeyboardKeyNumLock, numpad1: LogicalKeyboardKeyNumpad1}, NewKeySet(KeyboardLockModeNumLock)},
		{"num lock up", ui.KeyEventTypeUp, numLock, LogicalKeyboardKeyNumLock,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numpad1: LogicalKeyboardKeyNumpad1}, NewKeySet(KeyboardLockModeNumLock)},
		{"num lock down again", ui.KeyEventTypeDown, numLock, LogicalKeyboardKeyNumLock,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numLock: LogicalKeyboardKeyNumLock, numpad1: LogicalKeyboardKeyNumpad1}, KeySet[KeyboardLockMode]{}},
		{"numpad 1 up", ui.KeyEventTypeUp, numpad1, LogicalKeyboardKeyNumpad1,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{numLock: LogicalKeyboardKeyNumLock}, KeySet[KeyboardLockMode]{}},
		{"num lock up again", ui.KeyEventTypeUp, numLock, LogicalKeyboardKeyNumLock,
			map[PhysicalKeyboardKey]LogicalKeyboardKey{}, KeySet[KeyboardLockMode]{}},
	}
	for _, s := range steps {
		simulate(k, s.typ, s.physical, s.logical)
		expectState(t, k, s.pressed)
		if got := k.LockModesEnabled(); !got.Equal(s.locks) {
			t.Fatalf("%s: lock modes = %v, want %v", s.name, got, s.locks)
		}
	}
}

func TestKeyEventCanTellWhichKeysArePressed(t *testing.T) {
	k := NewHardwareKeyboard()
	check := func(step string, physical PhysicalKeyboardKey, logical LogicalKeyboardKey, want bool) {
		t.Helper()
		if got := k.IsPhysicalKeyPressed(physical); got != want {
			t.Errorf("%s: IsPhysicalKeyPressed(%v) = %v", step, physical, got)
		}
		if got := k.IsLogicalKeyPressed(logical); got != want {
			t.Errorf("%s: IsLogicalKeyPressed(%v) = %v", step, logical, got)
		}
	}
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock)
	check("num lock down", PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock, true)
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1)
	simulate(k, ui.KeyEventTypeRepeat, PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1)
	check("numpad 1 repeat", PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1, true)
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock)
	check("num lock up", PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1, true)
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock)
	check("num lock down again", PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock, true)
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1)
	check("numpad 1 up", PhysicalKeyboardKeyNumpad1, LogicalKeyboardKeyNumpad1, false)
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock)
	check("num lock up again", PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock, false)
}

func TestModifierShortcuts(t *testing.T) {
	k := NewHardwareKeyboard()
	if k.IsControlPressed() || k.IsShiftPressed() || k.IsAltPressed() || k.IsMetaPressed() {
		t.Fatal("no modifier should be pressed")
	}
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyControlRight, LogicalKeyboardKeyControlRight)
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyShiftLeft, LogicalKeyboardKeyShiftLeft)
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyAltLeft, LogicalKeyboardKeyAltLeft)
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyMetaRight, LogicalKeyboardKeyMetaRight)
	if !k.IsControlPressed() || !k.IsShiftPressed() || !k.IsAltPressed() || !k.IsMetaPressed() {
		t.Error("every modifier should be pressed")
	}
	if l, ok := k.LookUpLayout(PhysicalKeyboardKeyShiftLeft); !ok || l != LogicalKeyboardKeyShiftLeft {
		t.Errorf("LookUpLayout = %v, %v", l, ok)
	}
	if _, ok := k.LookUpLayout(PhysicalKeyboardKeyKeyA); ok {
		t.Error("an unpressed key has no layout entry")
	}
}

func TestDispatchEventsToAllHandlers(t *testing.T) {
	k := NewHardwareKeyboard()
	var logs []int
	// The framework's own handler, which the binding would add first.
	k.AddHandler(func(KeyEvent) bool { logs = append(logs, 1); return false })
	check := func(typ ui.KeyEventType, wantHandled bool, wantLogs ...int) {
		t.Helper()
		logs = nil
		if got := simulate(k, typ, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA); got != wantHandled {
			t.Errorf("handled = %v, want %v", got, wantHandled)
		}
		if !slices.Equal(logs, wantLogs) {
			t.Errorf("logs = %v, want %v", logs, wantLogs)
		}
	}
	// Only the framework handler.
	check(ui.KeyEventTypeDown, false, 1)

	// Adds a handler.
	handler2Result, handler3Result := false, false
	handler2 := func(KeyEvent) bool { logs = append(logs, 2); return handler2Result }
	handler3 := func(KeyEvent) bool { logs = append(logs, 3); return handler3Result }
	first2 := k.AddHandler(handler2)
	check(ui.KeyEventTypeUp, false, 1, 2)
	handler2Result = true
	check(ui.KeyEventTypeDown, true, 1, 2)

	// Adds another handler.
	handler2Result = false
	k.AddHandler(handler3)
	check(ui.KeyEventTypeUp, false, 1, 2, 3)
	handler2Result = true
	check(ui.KeyEventTypeDown, true, 1, 2, 3)
	handler3Result = true
	check(ui.KeyEventTypeUp, true, 1, 2, 3)

	// Adds handler2 again.
	k.AddHandler(handler2)
	handler3Result, handler2Result = false, false
	check(ui.KeyEventTypeDown, false, 1, 2, 3, 2)
	handler2Result = true
	check(ui.KeyEventTypeUp, true, 1, 2, 3, 2)

	// Removes the first handler2 only.
	k.RemoveHandler(first2)
	check(ui.KeyEventTypeDown, true, 1, 3, 2)
}

func TestHandlersAddedDuringDispatchRunNextTime(t *testing.T) {
	k := NewHardwareKeyboard()
	var logs []string
	var late KeyHandlerHandle
	var self KeyHandlerHandle
	self = k.AddHandler(func(KeyEvent) bool {
		logs = append(logs, "first")
		late = k.AddHandler(func(KeyEvent) bool { logs = append(logs, "late"); return false })
		k.RemoveHandler(self)
		return false
	})
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	if !slices.Equal(logs, []string{"first"}) {
		t.Fatalf("logs = %v, want only the first handler during the dispatch", logs)
	}
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	if !slices.Equal(logs, []string{"first", "late"}) {
		t.Errorf("logs = %v, want the late handler on the next event and the first one gone", logs)
	}
	k.RemoveHandler(late)
}

func TestKeyEventsCarryTheirData(t *testing.T) {
	down := NewKeyDownEvent(PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA, time.Second, KeyDownEventOpts{Character: "a", Synthesized: true})
	if down.Character() != "a" || !down.Synthesized() || down.TimeStamp() != time.Second || down.DeviceType() != ui.KeyEventDeviceTypeKeyboard {
		t.Errorf("down event = %v", down)
	}
	up := NewKeyUpEvent(PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA, 0, KeyUpEventOpts{DeviceType: ui.KeyEventDeviceTypeGamepad})
	if up.Character() != "" || up.DeviceType() != ui.KeyEventDeviceTypeGamepad {
		t.Errorf("up event = %v", up)
	}
	repeat := NewKeyRepeatEvent(PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA, 0, KeyRepeatEventOpts{Character: "a"})
	if repeat.Synthesized() || repeat.Character() != "a" {
		t.Errorf("repeat event = %v", repeat)
	}
	if got := down.String(); !strings.HasPrefix(got, "KeyDownEvent(") || !strings.Contains(got, "synthesized") {
		t.Errorf("String = %q", got)
	}
}

func TestKeyEventFromData(t *testing.T) {
	// A key that Flutter knows gets its constant; an unknown one keeps its id.
	e, err := KeyEventFromData(ui.KeyData{Type: ui.KeyEventTypeDown, Physical: 0x70004, Logical: 0x61, Character: "a", TimeStamp: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(*KeyDownEvent); !ok || e.PhysicalKey() != PhysicalKeyboardKeyKeyA || e.LogicalKey() != LogicalKeyboardKeyKeyA || e.Character() != "a" {
		t.Errorf("event = %v", e)
	}
	e, err = KeyEventFromData(ui.KeyData{Type: ui.KeyEventTypeRepeat, Physical: 0x12345, Logical: 0x1500012345})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(*KeyRepeatEvent); !ok || e.PhysicalKey() != 0x12345 || e.LogicalKey() != 0x1500012345 {
		t.Errorf("event = %v", e)
	}
	if e, _ := KeyEventFromData(ui.KeyData{Type: ui.KeyEventTypeUp, Physical: 1, Logical: 1, Synthesized: true}); !e.Synthesized() {
		t.Error("the up event lost its synthesized flag")
	}
}

func TestKeyDataFailures(t *testing.T) {
	_, err := KeyEventFromData(ui.KeyData{Physical: 1, Logical: 1})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("KeyEventFromData.KeyDataTypeInvalid") {
		t.Fatalf("error = %v, want KeyDataTypeInvalid", err)
	}
	// With an error reporter, HandleKeyData reports it and returns false.
	var reported []error
	k := NewHardwareKeyboard(HardwareKeyboardOpts{ErrorReporter: func(err error) { reported = append(reported, err) }})
	if k.HandleKeyData(ui.KeyData{Physical: 1, Logical: 1}) || len(reported) != 1 {
		t.Errorf("reported %d errors", len(reported))
	}
	// Without one, it panics.
	k = NewHardwareKeyboard()
	func() {
		defer func() {
			appErr, _ := errors.FromError(asError(recover()))
			if appErr == nil || appErr.Tag != "KeyDataTypeInvalid" {
				t.Errorf("recovered %v, want KeyDataTypeInvalid", appErr)
			}
		}()
		k.HandleKeyData(ui.KeyData{Physical: 1, Logical: 1})
	}()
}

func asError(v any) error {
	err, _ := v.(error)
	return err
}

func TestEmptyKeyDataYieldsNoEvent(t *testing.T) {
	k := NewHardwareKeyboard()
	var events []KeyEvent
	k.AddHandler(func(e KeyEvent) bool { events = append(events, e); return true })
	if k.HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown, Character: "a"}) {
		t.Error("empty key data was handled")
	}
	if len(events) != 0 {
		t.Fatalf("%d events, want none", len(events))
	}
	if !k.HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 0x22, Physical: 0x70034, Character: `"`}) || len(events) != 1 {
		t.Errorf("%d events, want one", len(events))
	}
}

func TestSynthesizedEventsAreDispatchedAtOnce(t *testing.T) {
	k := NewHardwareKeyboard()
	var logs []int
	k.AddHandler(func(KeyEvent) bool { logs = append(logs, 1); return false })
	k.AddHandler(func(KeyEvent) bool { logs = append(logs, 2); return false })
	if k.HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown, Logical: uint64(LogicalKeyboardKeyKeyA), Physical: uint64(PhysicalKeyboardKeyKeyA), Synthesized: true}) {
		t.Error("nothing handled the event")
	}
	if !slices.Equal(logs, []int{1, 2}) {
		t.Errorf("logs = %v", logs)
	}
	if !k.IsLogicalKeyPressed(LogicalKeyboardKeyKeyA) {
		t.Error("the synthesized press was not recorded")
	}
}

func TestExceptionsFromHandlersAreCaughtAndReported(t *testing.T) {
	var reported []error
	k := NewHardwareKeyboard(HardwareKeyboardOpts{ErrorReporter: func(err error) { reported = append(reported, err) }})
	throwing := k.AddHandler(func(KeyEvent) bool { panic(1) })
	var after int
	k.AddHandler(func(KeyEvent) bool { after++; return false })
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	if len(reported) != 1 {
		t.Fatalf("%d errors reported, want 1", len(reported))
	}
	appErr, ok := errors.FromError(reported[0])
	if !ok || !appErr.HasRoute("CallbackPanicked") || !strings.HasPrefix(appErr.Metadata["Event"].(string), "KeyDownEvent(") {
		t.Errorf("reported %v, want CallbackPanicked with the event", reported[0])
	}
	if appErr.Metadata["context"] != "while processing a key handler" {
		t.Errorf("context = %v", appErr.Metadata["context"])
	}
	// The other handler ran, and the exception does not stop recording the state.
	if after != 1 || !k.IsLogicalKeyPressed(LogicalKeyboardKeyKeyA) {
		t.Errorf("the next handler ran %d times, pressed: %v", after, k.IsLogicalKeyPressed(LogicalKeyboardKeyKeyA))
	}
	k.RemoveHandler(throwing)
	reported = nil
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	if len(reported) != 0 {
		t.Errorf("reported %v", reported)
	}
}

func TestHandlerPanicWithoutReporterKeepsTheKeyboardUsable(t *testing.T) {
	k := NewHardwareKeyboard()
	throwing := k.AddHandler(func(KeyEvent) bool { panic("boom") })
	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic was swallowed")
			}
		}()
		simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	}()
	k.RemoveHandler(throwing)
	simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
}

func TestNestedDispatchPanics(t *testing.T) {
	skipUnlessDebug(t)
	var reported []error
	k := NewHardwareKeyboard(HardwareKeyboardOpts{ErrorReporter: func(err error) { reported = append(reported, err) }})
	k.AddHandler(func(KeyEvent) bool {
		simulate(k, ui.KeyEventTypeUp, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
		return false
	})
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	if len(reported) != 1 {
		t.Fatalf("reported %v", reported)
	}
	if appErr, _ := errors.FromError(reported[0]); appErr == nil || !appErr.HasRoute("KeyboardDispatchNested") {
		t.Errorf("reported %v, want KeyboardDispatchNested", reported[0])
	}
}

func TestDebugPrintKeyboardEventsLogsKeyEvents(t *testing.T) {
	skipUnlessDebug(t)
	var messages []string
	k := NewHardwareKeyboard(HardwareKeyboardOpts{DebugPrintKeyboardEvents: func() bool { return true }, DebugPrint: func(m string) { messages = append(messages, m) }})
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA)
	all := strings.Join(messages, "\n")
	for _, want := range []string{
		"KEYBOARD: Key event received: ",
		"KEYBOARD: Pressed state before processing the event:",
		"KEYBOARD: Pressed state after processing the event:",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the log lacks %q:\n%s", want, all)
		}
	}
}

func TestIrregularKeyEventsAreProcessed(t *testing.T) {
	skipUnlessDebug(t)
	var logs []string
	k := NewHardwareKeyboard(HardwareKeyboardOpts{DebugPrintKeyboardEvents: func() bool { return true }, DebugPrint: func(m string) { logs = append(logs, m) }})
	expectIsRegular := func(handled bool, isRegular bool) {
		t.Helper()
		// All simulated events return false: they are dispatched to the handlers.
		if handled {
			t.Error("an event was handled")
		}
		hasError := strings.Contains(strings.Join(logs, "\n"), "ERROR")
		if hasError == isRegular {
			t.Errorf("error logged: %v, but the event is regular: %v\n%s", hasError, isRegular, strings.Join(logs, "\n"))
		}
		logs = nil
	}
	a, b := LogicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyB
	pa := PhysicalKeyboardKeyKeyA

	// 1. Presses key A.
	expectIsRegular(simulate(k, ui.KeyEventTypeDown, pa, a), true)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{pa: a})
	// Presses it again with a mismatched logical key, which affects the state.
	expectIsRegular(simulate(k, ui.KeyEventTypeDown, pa, b), false)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{pa: b})
	// 2. Releases key A.
	expectIsRegular(simulate(k, ui.KeyEventTypeUp, pa, b), true)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{})
	// Releases it again.
	expectIsRegular(simulate(k, ui.KeyEventTypeUp, pa, a), false)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{})
	// 3. Sends a repeat for key A, which affects the state.
	expectIsRegular(simulate(k, ui.KeyEventTypeRepeat, pa, a), false)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{pa: a})
	// 4. Sends a down event with a mismatched logical key.
	expectIsRegular(simulate(k, ui.KeyEventTypeDown, pa, b), false)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{pa: b})
	// 5. Sends a key up event with a mismatched logical key.
	expectIsRegular(simulate(k, ui.KeyEventTypeUp, pa, a), false)
	expectState(t, k, map[PhysicalKeyboardKey]LogicalKeyboardKey{})
}

func TestSetPressedKeysAndClearState(t *testing.T) {
	k := NewHardwareKeyboard()
	k.SetPressedKeys(map[PhysicalKeyboardKey]LogicalKeyboardKey{PhysicalKeyboardKeyShiftLeft: LogicalKeyboardKeyShiftLeft})
	if !k.IsShiftPressed() {
		t.Fatal("shift should be pressed")
	}
	simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyCapsLock, LogicalKeyboardKeyCapsLock)
	k.AddHandler(func(KeyEvent) bool { return true })
	k.ClearState()
	if len(k.PhysicalKeysPressed()) != 0 || len(k.LockModesEnabled()) != 0 {
		t.Error("the state was not cleared")
	}
	if simulate(k, ui.KeyEventTypeDown, PhysicalKeyboardKeyKeyA, LogicalKeyboardKeyKeyA) {
		t.Error("the handlers were not cleared")
	}
}

func TestKeyboardLockModes(t *testing.T) {
	tests := []struct {
		mode KeyboardLockMode
		key  LogicalKeyboardKey
	}{
		{KeyboardLockModeNumLock, LogicalKeyboardKeyNumLock},
		{KeyboardLockModeScrollLock, LogicalKeyboardKeyScrollLock},
		{KeyboardLockModeCapsLock, LogicalKeyboardKeyCapsLock},
	}
	for _, tc := range tests {
		if tc.mode.LogicalKey() != tc.key {
			t.Errorf("%v.LogicalKey() = %v", tc.mode, tc.mode.LogicalKey())
		}
		if got, ok := FindKeyboardLockModeByLogicalKey(tc.key); !ok || got != tc.mode {
			t.Errorf("FindKeyboardLockModeByLogicalKey(%v) = %v, %v", tc.key, got, ok)
		}
	}
	if _, ok := FindKeyboardLockModeByLogicalKey(LogicalKeyboardKeyKeyA); ok {
		t.Error("A switches no lock mode")
	}
	if KeyboardLockModeUnspecified.LogicalKey() != 0 || KeyboardLockModeCapsLock.String() != "KeyboardLockMode.capsLock" {
		t.Error("lock mode names are wrong")
	}
}

func TestMultipleOptionsPanic(t *testing.T) {
	defer func() {
		appErr, _ := errors.FromError(asError(recover()))
		if appErr == nil || !appErr.HasRoute("MultipleOptions") {
			t.Errorf("recovered %v, want MultipleOptions", appErr)
		}
	}()
	NewHardwareKeyboard(HardwareKeyboardOpts{}, HardwareKeyboardOpts{})
}
