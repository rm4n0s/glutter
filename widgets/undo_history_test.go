package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/undo_history_test.dart.
// Not ported: the tests of UndoManager.client and of the iOS undo manager channel
// (SetUndoState, handlePlatformUndo), which Glutter does not have (SPEC §12.10).

var allPlatforms = []foundation.TargetPlatform{
	foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS,
	foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows,
}

// eachPlatform runs the test for every target platform, as TargetPlatformVariant.all().
func eachPlatform(t *testing.T, test func(t *testing.T)) {
	if !foundation.DebugMode {
		// The target platform cannot be overridden in release builds: the tests
		// run on the host's platform.
		t.Run("host", test)
		return
	}
	for _, p := range allPlatforms {
		t.Run(p.String(), func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
			defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
			test(t)
		})
	}
}

// sendUndoRedo presses Control+Z, with Shift to redo: the shortcut of the
// Linux bindings, which every platform uses.
func sendUndoRedo(ts *gluttertest.Tester, redo bool) {
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyControl)
	if redo {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyShift)
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyKeyZ)
	if redo {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyShift)
	}
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyControl)
	ts.Pump()
}

type undoFixture struct {
	ts         *gluttertest.Tester
	value      *foundation.ValueNotifier[int]
	controller *UndoHistoryController
	focus      *FocusNode
}

func newUndoFixture(t *testing.T, withFocusChild bool, trigger func(f *undoFixture, v int)) *undoFixture {
	f := &undoFixture{ts: gluttertest.New(t), value: foundation.NewValueNotifier(0), controller: NewUndoHistoryController(),
		focus: NewFocusNode(FocusNodeOpts{DebugLabel: "UndoHistory Node"})}
	t.Cleanup(func() { f.focus.Dispose() })
	var child Widget = NewContainer(nil)
	if withFocusChild {
		child = NewFocus(NewContainer(nil), FocusOpts{FocusNode: f.focus})
	}
	if trigger == nil {
		trigger = func(f *undoFixture, v int) { f.value.SetValue(v) }
	}
	f.ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewUndoHistory[int](f.value, func(v int) { trigger(f, v) }, f.focus, child,
		UndoHistoryOpts[int]{Controller: f.controller})}))
	return f
}

func (f *undoFixture) check(t *testing.T, step string, value int, canUndo, canRedo bool) {
	t.Helper()
	if f.value.Value() != value || f.controller.Value().CanUndo != canUndo || f.controller.Value().CanRedo != canRedo {
		t.Errorf("%s: value %d, canUndo %v, canRedo %v; want %d, %v, %v", step,
			f.value.Value(), f.controller.Value().CanUndo, f.controller.Value().CanRedo, value, canUndo, canRedo)
	}
}

// undoRedoSequence is the sequence of the two tests that undo and redo, once
// with the controller and once with the keyboard.
func undoRedoSequence(t *testing.T, f *undoFixture, undo, redo func()) {
	f.ts.Pump(500 * time.Millisecond)
	f.check(t, "start", 0, false, false)
	undo()
	f.check(t, "undo before focus", 0, false, false)
	redo()
	f.check(t, "redo before focus", 0, false, false)
	f.focus.RequestFocus()
	f.ts.Pump()
	f.check(t, "focused", 0, false, false)
	undo()
	redo()
	f.check(t, "nothing to undo", 0, false, false)
	f.value.SetValue(1)
	f.ts.Pump(500 * time.Millisecond)
	f.check(t, "1", 1, true, false)
	undo()
	f.check(t, "undo to 0", 0, false, true)
	redo()
	f.check(t, "redo to 1", 1, true, false)
	f.value.SetValue(2)
	f.ts.Pump(500 * time.Millisecond)
	f.check(t, "2", 2, true, false)
	undo()
	f.check(t, "undo to 1", 1, true, true)
	undo()
	f.check(t, "undo to 0 again", 0, false, true)
	redo()
	f.check(t, "redo to 1 again", 1, true, true)
	redo()
	f.check(t, "redo to 2", 2, true, false)
	undo()
	f.check(t, "undo to 1 again", 1, true, true)
	f.value.SetValue(3)
	f.ts.Pump(500 * time.Millisecond)
	f.check(t, "3 clears the redo", 3, true, false)
}

func TestUndoHistoryAllowsUndoAndRedoFromTheController(t *testing.T) {
	eachPlatform(t, func(t *testing.T) {
		f := newUndoFixture(t, false, nil)
		undoRedoSequence(t, f, f.controller.Undo, f.controller.Redo)
	})
}

func TestUndoHistoryAllowsUndoAndRedoFromTheKeyboard(t *testing.T) {
	eachPlatform(t, func(t *testing.T) {
		f := newUndoFixture(t, true, nil)
		undoRedoSequence(t, f, func() { sendUndoRedo(f.ts, false) }, func() { sendUndoRedo(f.ts, true) })
	})
}

func TestUndoHistoryDuplicateChangesDoNotAffectTheHistory(t *testing.T) {
	eachPlatform(t, func(t *testing.T) {
		f := newUndoFixture(t, false, nil)
		f.focus.RequestFocus()
		f.ts.Pump(500 * time.Millisecond)
		f.value.SetValue(1)
		f.ts.Pump(500 * time.Millisecond)
		f.check(t, "1", 1, true, false)
		f.controller.Undo()
		f.check(t, "undo", 0, false, true)
		f.controller.Redo()
		f.check(t, "redo", 1, true, false)
		f.value.SetValue(1)
		f.ts.Pump(500 * time.Millisecond)
		f.check(t, "duplicate", 1, true, false)
		f.controller.Undo()
		f.check(t, "undo again", 0, false, true)
	})
}

func TestUndoHistoryIgnoresValueChangesPushedDuringOnTriggered(t *testing.T) {
	skipUnlessDebug(t)
	valueToUse := func(v int) int { return v }
	key := NewGlobalKey[*UndoHistoryState[int]]()
	ts := gluttertest.New(t)
	value := foundation.NewValueNotifier(0)
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "UndoHistory Node"})
	defer focus.Dispose()
	controller := NewUndoHistoryController()
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewUndoHistory[int](value, func(v int) { value.SetValue(valueToUse(v)) }, focus, NewContainer(nil),
		UndoHistoryOpts[int]{Key: key, Controller: controller})}))
	ts.Pump(500 * time.Millisecond)
	focus.RequestFocus()
	ts.Pump()
	value.SetValue(1)
	ts.Pump(500 * time.Millisecond)
	valueToUse = func(int) int { return 3 }
	state, _ := key.CurrentState()
	requireTag(t, "UndoHistoryValueNotSet", state.Undo)
}

func TestUndoHistoryThrottlesChangesIntoOneEntry(t *testing.T) {
	f := newUndoFixture(t, false, nil)
	f.focus.RequestFocus()
	f.ts.Pump(500 * time.Millisecond)
	for i := 1; i <= 3; i++ {
		f.value.SetValue(i)
		f.ts.Pump(100 * time.Millisecond)
	}
	f.ts.Pump(500 * time.Millisecond)
	f.check(t, "throttled", 3, true, false)
	f.controller.Undo()
	f.check(t, "undo goes back to the start", 0, false, true)
}

func TestUndoHistoryValueString(t *testing.T) {
	got := fmt.Sprint(UndoHistoryValue{CanUndo: true})
	if got != "UndoHistoryValue(canUndo: true, canRedo: false)" {
		t.Errorf("String = %q", got)
	}
}
