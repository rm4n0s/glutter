package widgets_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/shortcuts_test.dart.
// Not ported: the hashCode tests (activators are comparable values in Go, and the
// order-independence of LogicalKeySet equality is tested through map keys), the
// key simulator transit mode variants (there is one key path), the cases that
// need the semantics tester, and the cases that need EditableText (step 11.4).

type testIntent2 struct{ IntentBase }

// callbackIntentAction is TestAction of shortcuts_test.dart: a CallbackAction<Intent>.
func callbackIntentAction(onInvoke func(Intent) any) Action {
	return NewCallbackAction[Intent](func(i Intent) any { return onInvoke(i) })
}

// activatorTester is activatorTester of shortcuts_test.dart: a Shortcuts widget
// with one or two activators over a focused box, whose intents run the callbacks.
func activatorTester(activator ShortcutActivator, onInvoke func(Intent), second ...any) Widget {
	actions := ActionMap{IntentTypeOf[testIntentV](): callbackIntentAction(func(i Intent) any {
		onInvoke(i)
		return true
	})}
	shortcuts := ShortcutMap{{activator, testIntentV{}}}
	if len(second) == 2 {
		activator2, onInvoke2 := second[0].(ShortcutActivator), second[1].(func(Intent))
		actions[IntentTypeOf[testIntent2]()] = callbackIntentAction(func(i Intent) any {
			onInvoke2(i)
			return nil
		})
		shortcuts = append(shortcuts, ShortcutEntry{activator2, testIntent2{}})
	}
	return NewActions(actions, NewShortcuts(shortcuts,
		NewFocus(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), FocusOpts{Autofocus: true})),
		ActionsOpts{Key: NewGlobalKey[State]()})
}

type keyStep struct {
	kind string // "down", "up" or "repeat"
	key  services.LogicalKeyboardKey
	want int // the number of invocations expected after the step; -1 for no check
}

func sendKey(ts *gluttertest.Tester, kind string, key services.LogicalKeyboardKey) {
	switch kind {
	case "down":
		ts.SendKeyDownEvent(key)
	case "up":
		ts.SendKeyUpEvent(key)
	case "repeat":
		ts.SendKeyRepeatEvent(key)
	}
}

// runKeySteps sends the steps and checks the number of invocations after the ones
// that say so. The counter resets between sequences.
func runKeySteps(t *testing.T, ts *gluttertest.Tester, invoked *int, steps []keyStep) {
	t.Helper()
	for i, s := range steps {
		sendKey(ts, s.kind, s.key)
		if s.want >= 0 && *invoked != s.want {
			t.Fatalf("step %d (%s %v): invoked = %d, want %d", i, s.kind, s.key, *invoked, s.want)
		}
	}
}

const (
	lCtrl  = services.LogicalKeyboardKeyControlLeft
	rCtrl  = services.LogicalKeyboardKeyControlRight
	lShift = services.LogicalKeyboardKeyShiftLeft
	keyA   = services.LogicalKeyboardKeyKeyA
	keyB   = services.LogicalKeyboardKeyKeyB
	keyC   = services.LogicalKeyboardKeyKeyC
	keyD   = services.LogicalKeyboardKeyKeyD
)

func down(k services.LogicalKeyboardKey, want int) keyStep   { return keyStep{"down", k, want} }
func up(k services.LogicalKeyboardKey) keyStep               { return keyStep{"up", k, -1} }
func repeat(k services.LogicalKeyboardKey, want int) keyStep { return keyStep{"repeat", k, want} }

func requireNoKeysPressed(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	if keys := ts.Binding.Keyboard().LogicalKeysPressed(); len(keys) != 0 {
		t.Errorf("logical keys pressed = %v, want none", keys)
	}
}

func TestLogicalKeySet(t *testing.T) {
	t.Run("passes parameters correctly", func(t *testing.T) {
		wantKeys := func(keys ...services.LogicalKeyboardKey) services.KeySet[services.LogicalKeyboardKey] {
			return services.NewKeySet(keys...)
		}
		for _, tc := range []struct {
			name string
			set  LogicalKeySet
			want services.KeySet[services.LogicalKeyboardKey]
		}{
			{"one", NewLogicalKeySet(keyA), wantKeys(keyA)},
			{"two", NewLogicalKeySet(keyA, keyB), wantKeys(keyA, keyB)},
			{"three", NewLogicalKeySet(keyA, keyB, keyC), wantKeys(keyA, keyB, keyC)},
			{"four", NewLogicalKeySet(keyA, keyB, keyC, keyD), wantKeys(keyA, keyB, keyC, keyD)},
			{"fromSet", NewLogicalKeySetFromSet(wantKeys(keyA, keyB, keyC, keyD)), wantKeys(keyA, keyB, keyC, keyD)},
		} {
			got := tc.set.Keys()
			if len(got) != len(tc.want) {
				t.Errorf("%s: keys = %v, want %v", tc.name, got, tc.want)
				continue
			}
			for k := range tc.want {
				if !got.Contains(k) {
					t.Errorf("%s: keys = %v, want %v", tc.name, got, tc.want)
				}
			}
		}
	})

	t.Run("works as a map key", func(t *testing.T) {
		set1 := NewLogicalKeySet(keyA)
		set2 := NewLogicalKeySet(keyA, keyB, keyC, keyD)
		set3 := NewLogicalKeySet(keyD, keyC, keyB, keyA)
		set4 := NewLogicalKeySetFromSet(services.NewKeySet(keyD, keyC, keyB, keyA))
		m := map[ShortcutActivator]string{set1: "one"}
		if set2 != set3 || set2 != set4 {
			t.Error("sets with the same keys should be equal")
		}
		if _, ok := m[set1]; !ok {
			t.Error("the map should contain set1")
		}
		if _, ok := m[NewLogicalKeySet(keyA)]; !ok {
			t.Error("the map should contain an equal set")
		}
	})

	t.Run("handles two keys", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewLogicalKeySet(keyC, services.LogicalKeyboardKeyControl), func(Intent) { invoked++ }))
		ts.Pump()
		for _, steps := range [][]keyStep{
			// LCtrl -> KeyC: Accept
			{down(lCtrl, 0), down(keyC, 1), up(keyC), up(lCtrl)},
			// KeyC -> LCtrl: Accept
			{down(keyC, 0), down(lCtrl, 1), up(keyC), up(lCtrl)},
			// RCtrl -> KeyC: Accept
			{down(rCtrl, 0), down(keyC, 1), up(keyC), up(rCtrl)},
			// LCtrl -> LShift -> KeyC: Reject
			{down(lCtrl, -1), down(lShift, -1), down(keyC, 0), up(keyC), up(lCtrl), up(lShift)},
			// LCtrl -> KeyA -> KeyC: Reject
			{down(lCtrl, -1), down(keyA, -1), down(keyC, 0), up(keyC), up(lCtrl), up(keyA)},
		} {
			invoked = 0
			runKeySteps(t, ts, &invoked, steps)
		}
		requireNoKeysPressed(t, ts)
	})

	t.Run("accepts works as expected", func(t *testing.T) {
		ts, events := keyEventCollector(t)
		set := NewLogicalKeySet(keyA, services.LogicalKeyboardKeyControl)
		accepts := func() bool { return set.Accepts((*events)[len(*events)-1], ts.Binding.Keyboard()) }
		ts.SendKeyDownEvent(lCtrl)
		ts.SendKeyDownEvent(keyA)
		requireBool(t, "down", accepts(), true)
		ts.SendKeyRepeatEvent(keyA)
		requireBool(t, "repeat", accepts(), true)
		ts.SendKeyUpEvent(keyA)
		requireBool(t, "up", accepts(), false)
		ts.SendKeyUpEvent(lCtrl)
		requireBool(t, "ctrl up", accepts(), false)
	})

	t.Run("diagnostics work", func(t *testing.T) {
		skipUnlessDebug(t)
		builder := foundation.NewDiagnosticPropertiesBuilder()
		NewLogicalKeySet(keyA, keyB).DebugFillProperties(builder)
		got := describeProperties(builder)
		if len(got) != 1 || got[0] != "keys: Key A + Key B" {
			t.Errorf("description = %q", got)
		}
	})
}

// keyEventCollector pumps a focused widget that collects the key events it gets.
func keyEventCollector(t *testing.T) (*gluttertest.Tester, *[]services.KeyEvent) {
	t.Helper()
	ts := gluttertest.New(t)
	events := &[]services.KeyEvent{}
	ts.PumpWidget(NewFocus(NewSizedBoxShrink(nil), FocusOpts{Autofocus: true,
		OnKeyEvent: func(_ *FocusNode, e services.KeyEvent) KeyEventResult {
			*events = append(*events, e)
			return KeyEventResultIgnored
		}}))
	return ts, events
}

func requireBool(t *testing.T, what string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

func TestSingleActivator(t *testing.T) {
	t.Run("handles Ctrl-C", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewSingleActivator(keyC, SingleActivatorOpts{Control: true}), func(Intent) { invoked++ }))
		ts.Pump()
		for _, steps := range [][]keyStep{
			// LCtrl -> KeyC: Accept
			{down(lCtrl, 0), down(keyC, 1), up(keyC), up(lCtrl)},
			// KeyC -> LCtrl: Reject
			{down(keyC, 0), down(lCtrl, 0), up(keyC), up(lCtrl)},
			// LShift -> LCtrl -> KeyC: Reject
			{down(lShift, -1), down(lCtrl, -1), down(keyC, 0), up(keyC), up(lCtrl), up(lShift)},
			// With Ctrl-C pressed, KeyA -> Release KeyA: Reject
			{down(lCtrl, -1), down(keyC, -1), {"reset", 0, 0}, down(keyA, 0), up(keyA), up(keyC), up(lCtrl)},
			// LCtrl -> KeyA -> KeyC: Accept
			{down(lCtrl, -1), down(keyA, 0), down(keyC, 1), up(keyA), up(keyC), up(lCtrl)},
			// RCtrl -> KeyC: Accept
			{down(rCtrl, 0), down(keyC, 1), up(keyC), up(rCtrl)},
			// LCtrl -> RCtrl -> KeyC: Accept
			{down(lCtrl, -1), down(rCtrl, 0), down(keyC, 1), up(keyC), up(lCtrl), up(rCtrl)},
			// While holding Ctrl-C, press KeyA: Reject
			{down(lCtrl, 0), down(keyC, 1), down(keyA, 1), up(keyC), up(keyA), up(lCtrl)},
		} {
			invoked = 0
			for i, s := range steps {
				if s.kind == "reset" {
					invoked = 0
					continue
				}
				sendKey(ts, s.kind, s.key)
				if s.want >= 0 && invoked != s.want {
					t.Fatalf("sequence %v step %d (%s %v): invoked = %d, want %d", steps, i, s.kind, s.key, invoked, s.want)
				}
			}
		}
		requireNoKeysPressed(t, ts)
	})

	t.Run("handles repeated events", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewSingleActivator(keyC, SingleActivatorOpts{Control: true}), func(Intent) { invoked++ }))
		ts.Pump()
		runKeySteps(t, ts, &invoked, []keyStep{down(lCtrl, 0), down(keyC, 1), repeat(keyC, 2), up(keyC), up(lCtrl)})
		requireBool(t, "invoked twice", invoked == 2, true)
		requireNoKeysPressed(t, ts)
	})

	t.Run("rejects repeated events if requested", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewSingleActivator(keyC, SingleActivatorOpts{Control: true, ExcludeRepeats: true}), func(Intent) { invoked++ }))
		ts.Pump()
		runKeySteps(t, ts, &invoked, []keyStep{down(lCtrl, 0), down(keyC, 1), repeat(keyC, 1), up(keyC), up(lCtrl)})
		requireBool(t, "invoked once", invoked == 1, true)
		requireNoKeysPressed(t, ts)
	})

	t.Run("handles Shift-Ctrl-C", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewSingleActivator(keyC, SingleActivatorOpts{Shift: true, Control: true}), func(Intent) { invoked++ }))
		ts.Pump()
		for _, steps := range [][]keyStep{
			// LShift -> LCtrl -> KeyC: Accept
			{down(lShift, -1), down(lCtrl, 0), down(keyC, 1), up(keyC), up(lCtrl), up(lShift)},
			// LCtrl -> LShift -> KeyC: Accept
			{down(lCtrl, -1), down(lShift, 0), down(keyC, 1), up(keyC), up(lCtrl), up(lShift)},
			// LCtrl -> KeyC -> LShift: Reject
			{down(lCtrl, -1), down(keyC, -1), down(lShift, 0), up(keyC), up(lCtrl), up(lShift)},
		} {
			invoked = 0
			runKeySteps(t, ts, &invoked, steps)
		}
		requireNoKeysPressed(t, ts)
	})

	t.Run("accepts works as expected", func(t *testing.T) {
		ts, events := keyEventCollector(t)
		activator := NewSingleActivator(keyA, SingleActivatorOpts{Control: true})
		noRepeat := NewSingleActivator(keyA, SingleActivatorOpts{Control: true, ExcludeRepeats: true})
		accepts := func(a ShortcutActivator) bool { return a.Accepts((*events)[len(*events)-1], ts.Binding.Keyboard()) }
		ts.SendKeyDownEvent(lCtrl)
		ts.SendKeyDownEvent(keyA)
		requireBool(t, "down", accepts(activator), true)
		ts.SendKeyRepeatEvent(keyA)
		requireBool(t, "repeat", accepts(activator), true)
		ts.SendKeyUpEvent(keyA)
		requireBool(t, "up", accepts(activator), false)
		ts.SendKeyUpEvent(lCtrl)
		requireBool(t, "ctrl up", accepts(activator), false)

		ts.SendKeyDownEvent(lCtrl)
		ts.SendKeyDownEvent(keyA)
		requireBool(t, "no repeat: down", accepts(noRepeat), true)
		ts.SendKeyRepeatEvent(keyA)
		requireBool(t, "no repeat: repeat", accepts(noRepeat), false)
		ts.SendKeyUpEvent(keyA)
		requireBool(t, "no repeat: up", accepts(noRepeat), false)
		ts.SendKeyUpEvent(lCtrl)
		requireBool(t, "no repeat: ctrl up", accepts(noRepeat), false)
	})

	for _, tc := range []struct {
		name                     string
		numLock                  LockState
		whenLocked, whenUnlocked bool
	}{
		{"locked", LockStateLocked, true, false},
		{"unlocked", LockStateUnlocked, false, true},
		{"ignored", LockStateIgnored, true, true},
	} {
		t.Run("numLock works as expected when set to LockState."+tc.name, func(t *testing.T) {
			ts, events := keyEventCollector(t)
			activator := NewSingleActivator(services.LogicalKeyboardKeyNumpad4, SingleActivatorOpts{NumLock: tc.numLock})
			accepts := func() bool { return activator.Accepts((*events)[len(*events)-1], ts.Binding.Keyboard()) }
			numLockOn := func() bool {
				return ts.Binding.Keyboard().LockModesEnabled().Contains(services.KeyboardLockModeNumLock)
			}
			// Lock NumLock.
			ts.SendKeyEvent(services.LogicalKeyboardKeyNumLock)
			requireBool(t, "num lock on", numLockOn(), true)
			ts.SendKeyDownEvent(services.LogicalKeyboardKeyNumpad4)
			requireBool(t, "locked", accepts(), tc.whenLocked)
			ts.SendKeyUpEvent(services.LogicalKeyboardKeyNumpad4)
			// Unlock NumLock.
			ts.SendKeyEvent(services.LogicalKeyboardKeyNumLock)
			requireBool(t, "num lock on", numLockOn(), false)
			ts.SendKeyDownEvent(services.LogicalKeyboardKeyNumpad4)
			requireBool(t, "unlocked", accepts(), tc.whenUnlocked)
			ts.SendKeyUpEvent(services.LogicalKeyboardKeyNumpad4)
		})
	}

	t.Run("diagnostics", func(t *testing.T) {
		skipUnlessDebug(t)
		for _, tc := range []struct {
			name      string
			activator SingleActivator
			want      []string
		}{
			{"single key", NewSingleActivator(keyA), []string{"keys: Key A"}},
			{"no repeats", NewSingleActivator(keyA, SingleActivatorOpts{ExcludeRepeats: true}), []string{"keys: Key A", "excluding repeats"}},
			{"combination", NewSingleActivator(keyA, SingleActivatorOpts{Control: true, Shift: true, Alt: true, Meta: true}),
				[]string{"keys: Control + Alt + Meta + Shift + Key A"}},
		} {
			builder := foundation.NewDiagnosticPropertiesBuilder()
			tc.activator.DebugFillProperties(builder)
			if got := describeProperties(builder); !slicesEqual(got, tc.want) {
				t.Errorf("%s: description = %q, want %q", tc.name, got, tc.want)
			}
		}
	})
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// testShortcutManager is TestShortcutManager: it records the keys it gets.
type testShortcutManager struct {
	ShortcutManagerBase
	keys *[]services.LogicalKeyboardKey
}

func newTestShortcutManager(keys *[]services.LogicalKeyboardKey, shortcuts ShortcutMap) *testShortcutManager {
	m := &testShortcutManager{keys: keys}
	InitShortcutManagerBase(&m.ShortcutManagerBase, m, ShortcutManagerOpts{Shortcuts: shortcuts})
	return m
}

func (m *testShortcutManager) HandleKeypress(bc BuildContext, event services.KeyEvent) KeyEventResult {
	switch event.(type) {
	case *services.KeyDownEvent, *services.KeyRepeatEvent:
		*m.keys = append(*m.keys, event.LogicalKey())
	}
	return m.ShortcutManagerBase.HandleKeypress(bc, event)
}

func box100(key foundation.Key) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Key: key, Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
}

// focusedBox is Focus(autofocus: true, child: SizedBox(100, 100)), which the Dart
// tests use as the focused child; where they use a TestTextField, this stands in
// for it (the shortcuts see the same focus and the same result of the key).
func focusedBox() Widget { return NewFocus(box100(nil), FocusOpts{Autofocus: true}) }

func shiftShortcuts() ShortcutMap {
	return ShortcutMap{{NewLogicalKeySet(services.LogicalKeyboardKeyShift), testIntentV{}}}
}

func TestShortcuts(t *testing.T) {
	t.Run("default constructed Shortcuts has empty shortcuts", func(t *testing.T) {
		ts := gluttertest.New(t)
		w := NewShortcuts(ShortcutMap{}, NewSizedBoxShrink(nil))
		ts.PumpWidget(w)
		if got := w.Shortcuts(); got == nil || len(got) != 0 {
			t.Errorf("shortcuts = %v, want empty", got)
		}
	})

	t.Run("default constructed Shortcuts.manager has empty shortcuts", func(t *testing.T) {
		ts := gluttertest.New(t)
		manager := NewShortcutManager()
		defer manager.Dispose()
		if len(manager.Shortcuts()) != 0 {
			t.Errorf("manager shortcuts = %v, want empty", manager.Shortcuts())
		}
		w := NewShortcutsManager(manager, NewSizedBoxShrink(nil))
		ts.PumpWidget(w)
		if len(w.Shortcuts()) != 0 {
			t.Errorf("shortcuts = %v, want empty", w.Shortcuts())
		}
	})

	t.Run("Shortcuts.manager passes on shortcuts", func(t *testing.T) {
		ts := gluttertest.New(t)
		shortcuts := shiftShortcuts()
		manager := NewShortcutManager(ShortcutManagerOpts{Shortcuts: shortcuts})
		defer manager.Dispose()
		w := NewShortcutsManager(manager, NewSizedBoxShrink(nil))
		ts.PumpWidget(w)
		if len(manager.Shortcuts()) != 1 || len(w.Shortcuts()) != 1 || w.Shortcuts()[0] != shortcuts[0] {
			t.Errorf("manager shortcuts = %v, widget shortcuts = %v, want %v", manager.Shortcuts(), w.Shortcuts(), shortcuts)
		}
	})

	invokingAction := func(invoked *bool) Action {
		return newTestAction(func(Intent) any {
			*invoked = true
			return true
		})
	}

	t.Run("ShortcutManager handles shortcuts", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, shiftShortcuts())
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(NewActions(testActions(invokingAction(&invoked)), NewShortcutsManager(manager, focusedBox())))
		ts.Pump()
		ts.SendKeyDownEvent(lShift)
		if !invoked || !slices.Equal(pressed, []services.LogicalKeyboardKey{lShift}) {
			t.Errorf("invoked = %v, pressed = %v", invoked, pressed)
		}
	})

	t.Run("ShortcutManager ignores key presses with no primary focus", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, shiftShortcuts())
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(NewActions(testActions(invokingAction(&invoked)), NewShortcutsManager(manager, box100(nil))))
		ts.Pump()
		if ts.Binding.FocusManager().PrimaryFocus() != nil {
			t.Fatal("there should be no primary focus")
		}
		ts.SendKeyDownEvent(lShift)
		if invoked || len(pressed) != 0 {
			t.Errorf("invoked = %v, pressed = %v; want false, none", invoked, pressed)
		}
	})

	t.Run("Shortcuts passes to the next Shortcuts widget if it doesn't map the key", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, shiftShortcuts())
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(NewShortcutsManager(manager, NewActions(testActions(invokingAction(&invoked)),
			NewShortcuts(ShortcutMap{{NewLogicalKeySet(keyA), DoNothingIntent{}}}, focusedBox()))))
		ts.Pump()
		ts.SendKeyDownEvent(lShift)
		if !invoked || !slices.Equal(pressed, []services.LogicalKeyboardKey{lShift}) {
			t.Errorf("invoked = %v, pressed = %v", invoked, pressed)
		}
	})

	t.Run("Shortcuts can disable a shortcut with DoNothingIntent", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, shiftShortcuts())
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewShortcutsManager(manager, NewActions(testActions(invokingAction(&invoked)),
			NewShortcuts(ShortcutMap{{NewLogicalKeySet(services.LogicalKeyboardKeyShift), DoNothingIntent{}}}, focusedBox())))}))
		ts.Pump()
		ts.SendKeyDownEvent(lShift)
		if invoked || len(pressed) != 0 {
			t.Errorf("invoked = %v, pressed = %v; want false, none", invoked, pressed)
		}
	})

	// The Dart cases below use a TestTextField; a focused box stands in for it.
	keyAShortcuts := ShortcutMap{{NewLogicalKeySet(keyA), testIntentV{}}}

	t.Run("Shortcuts that aren't bound to an action don't absorb keys meant for text fields", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, keyAShortcuts)
		defer manager.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewShortcutsManager(manager, focusedBox())}))
		ts.Pump()
		handled := ts.SendKeyEvent(keyA)
		if handled || !slices.Equal(pressed, []services.LogicalKeyboardKey{keyA}) {
			t.Errorf("handled = %v, pressed = %v; want false, [A]", handled, pressed)
		}
	})

	t.Run("Shortcuts that are bound to an action do override text fields", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, keyAShortcuts)
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewShortcutsManager(manager,
			NewActions(testActions(invokingAction(&invoked)), focusedBox()))}))
		ts.Pump()
		handled := ts.SendKeyEvent(keyA)
		if !handled || !invoked || !slices.Equal(pressed, []services.LogicalKeyboardKey{keyA}) {
			t.Errorf("handled = %v, invoked = %v, pressed = %v; want true, true, [A]", handled, invoked, pressed)
		}
	})

	t.Run("Shortcuts can override intents that apply to text fields", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, keyAShortcuts)
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewShortcutsManager(manager,
			NewActions(testActions(invokingAction(&invoked)),
				NewActions(testActions(NewDoNothingAction(DoNothingActionOpts{DoesNotConsumeKey: true})), focusedBox())))}))
		ts.Pump()
		handled := ts.SendKeyEvent(keyA)
		if handled || invoked {
			t.Errorf("handled = %v, invoked = %v; want false, false", handled, invoked)
		}
	})

	t.Run("Shortcuts can override intents that apply to text fields with DoNothingAndStopPropagationIntent", func(t *testing.T) {
		ts := gluttertest.New(t)
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, keyAShortcuts)
		defer manager.Dispose()
		invoked := false
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewShortcutsManager(manager,
			NewActions(testActions(invokingAction(&invoked)),
				NewShortcuts(ShortcutMap{{NewLogicalKeySet(keyA), DoNothingAndStopPropagationIntent{}}}, focusedBox())))}))
		ts.Pump()
		handled := ts.SendKeyEvent(keyA)
		if handled || invoked {
			t.Errorf("handled = %v, invoked = %v; want false, false", handled, invoked)
		}
	})

	t.Run("Shortcuts pass the debug label to the focus node", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewShortcuts(ShortcutMap{}, NewSizedBoxShrink(nil), ShortcutsOpts{DebugLabel: "<My Shortcuts Debug Label>"}))
		node := FocusOf(ts.Element(gluttertest.ByType[*SizedBox]()))
		if node.DebugLabel() != "Shortcuts: <My Shortcuts Debug Label>" {
			t.Errorf("label = %q", node.DebugLabel())
		}
	})

	t.Run("Shortcuts support multiple intents", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
		orange, blue := ui.NewColor(0xFFFF8800), ui.NewColor(0xFF0000FF)
		value := true
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
			shortcuts: ShortcutMap{
				{NewLogicalKeySet(services.LogicalKeyboardKeySpace), PrioritizedIntents{OrderedIntents: []Intent{
					ActivateIntent{}, ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage},
				}}},
				{NewLogicalKeySet(services.LogicalKeyboardKeyTab), NextFocusIntent{}},
				{NewLogicalKeySet(services.LogicalKeyboardKeyPageUp), ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}},
			},
			home: NewCenter(NewListView([]Widget{
				NewStatefulBuilder(func(bc BuildContext, setState StateSetter) Widget {
					// A focusable widget that handles ActivateIntent.
					color := blue
					if value {
						color = orange
					}
					return NewActions(ActionMap{IntentTypeOf[ActivateIntent](): NewCallbackAction[ActivateIntent](func(ActivateIntent) any {
						setState(func() { value = !value })
						return nil
					})}, NewFocus(NewContainer(nil, ContainerOpts{Color: foundation.Some(color), Width: foundation.Some(48.0), Height: foundation.Some(48.0)})))
				}),
				NewContainer(nil, ContainerOpts{Color: foundation.Some(blue), Height: foundation.Some(1000.0)}),
			}, ListViewOpts{Primary: foundation.Some(true)})),
		}))
		ts.PumpAndSettle()
		if got := ts.Binding.FocusManager().PrimaryFocus().ToStringShort(); !strings.Contains(got, "Focus Scope [PRIMARY FOCUS]") {
			t.Errorf("primary focus = %q, want the route scope", got)
		}
		controller := PrimaryScrollControllerOf(ts.Element(gluttertest.ByType[*ListView]()))
		pixels := func() float64 { return controller.Position().Pixels() }
		if pixels() != 0 || !value {
			t.Fatalf("pixels = %v, value = %v; want 0, true", pixels(), value)
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.PumpAndSettle()
		// The ScrollView scrolls.
		if pixels() != 448 || !value {
			t.Fatalf("pixels = %v, value = %v; want 448, true", pixels(), value)
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeyPageUp)
		ts.PumpAndSettle()
		ts.SendKeyEvent(services.LogicalKeyboardKeyTab)
		ts.PumpAndSettle()
		// Focus is now on the focusable widget.
		if got := ts.Binding.FocusManager().PrimaryFocus().ToStringShort(); !strings.HasPrefix(got, "FocusNode#") || !strings.HasSuffix(got, "([PRIMARY FOCUS])") {
			t.Errorf("primary focus = %q", got)
		}
		if !value || pixels() != 0 {
			t.Fatalf("pixels = %v, value = %v; want 0, true", pixels(), value)
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.PumpAndSettle()
		// The focusable widget is toggled, the scroll view does not scroll.
		if value || pixels() != 0 {
			t.Fatalf("pixels = %v, value = %v; want 0, false", pixels(), value)
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.PumpAndSettle()
		if !value || pixels() != 0 {
			t.Errorf("pixels = %v, value = %v; want 0, true", pixels(), value)
		}
	})

	t.Run("Shortcuts support activators that return nil in triggers", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(dumbLogicalActivator{keyC}, func(Intent) { invoked++ },
			ShortcutActivator(NewSingleActivator(keyC, SingleActivatorOpts{Control: true})), func(Intent) { invoked += 10 }))
		ts.Pump()
		for _, steps := range [][]keyStep{
			// Press KeyC: accepted by DumbLogicalActivator.
			{down(keyC, 1), up(keyC)},
			// Press ControlLeft + KeyC: accepted by SingleActivator.
			{down(lCtrl, 0), down(keyC, 10), up(keyC), up(lCtrl)},
			// Press ControlLeft + ShiftLeft + KeyC: accepted by DumbLogicalActivator.
			{down(lShift, -1), down(lCtrl, 0), down(keyC, 1), up(keyC), up(lCtrl), up(lShift)},
		} {
			invoked = 0
			runKeySteps(t, ts, &invoked, steps)
		}
	})
}

// dumbLogicalActivator is DumbLogicalActivator: an activator that accepts the down
// and repeat events of a key. It is poorly designed on purpose: its triggers are
// nil, and it checks the key in Accepts.
type dumbLogicalActivator struct{ key services.LogicalKeyboardKey }

func (a dumbLogicalActivator) Triggers() []services.LogicalKeyboardKey { return nil }

func (a dumbLogicalActivator) Accepts(event services.KeyEvent, _ *services.HardwareKeyboard) bool {
	switch event.(type) {
	case *services.KeyDownEvent, *services.KeyRepeatEvent:
		return event.LogicalKey() == a.key
	}
	return false
}

func (a dumbLogicalActivator) DebugDescribeKeys() string { return a.key.KeyLabel() }

func TestShortcutsDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	describe := func(w *Shortcuts) []string {
		builder := foundation.NewDiagnosticPropertiesBuilder()
		w.DebugFillProperties(builder)
		return describeProperties(builder)
	}
	t.Run("work", func(t *testing.T) {
		got := describe(NewShortcuts(ShortcutMap{
			{NewLogicalKeySet(services.LogicalKeyboardKeyShift, keyA), ActivateIntent{}},
			{NewLogicalKeySet(services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyArrowRight), DirectionalFocusIntent{Direction: TraversalDirectionRight}},
		}, NewSizedBoxShrink(nil)))
		if len(got) != 1 || !strings.HasPrefix(got[0], "shortcuts: {{Shift + Key A}: ActivateIntent#") ||
			!strings.Contains(got[0], "{Shift + Arrow Right}: DirectionalFocusIntent#") || !strings.HasSuffix(got[0], "(direction: right)}") {
			t.Errorf("description = %q", got)
		}
	})
	t.Run("work when debugLabel specified", func(t *testing.T) {
		got := describe(NewShortcuts(ShortcutMap{{NewLogicalKeySet(keyA, keyB), ActivateIntent{}}}, NewSizedBoxShrink(nil),
			ShortcutsOpts{DebugLabel: "<Debug Label>"}))
		if !slices.Equal(got, []string{"shortcuts: <Debug Label>"}) {
			t.Errorf("description = %q", got)
		}
	})
	t.Run("work when manager not specified", func(t *testing.T) {
		got := describe(NewShortcuts(ShortcutMap{{NewLogicalKeySet(keyA, keyB), ActivateIntent{}}}, NewSizedBoxShrink(nil)))
		if len(got) != 1 || !strings.HasPrefix(got[0], "shortcuts: {{Key A + Key B}: ActivateIntent#") {
			t.Errorf("description = %q", got)
		}
	})
	t.Run("work when manager specified", func(t *testing.T) {
		var pressed []services.LogicalKeyboardKey
		manager := newTestShortcutManager(&pressed, ShortcutMap{{NewLogicalKeySet(keyA, keyB), ActivateIntent{}}})
		defer manager.Dispose()
		got := describe(NewShortcutsManager(manager, NewSizedBoxShrink(nil)))
		if len(got) != 2 || !strings.HasPrefix(got[0], "manager: ") || !strings.Contains(got[0], "shortcuts: {") ||
			!strings.HasPrefix(got[1], "shortcuts: {{Key A + Key B}: ActivateIntent#") {
			t.Errorf("description = %q", got)
		}
	})
}

func slashKey(ts *gluttertest.Tester, kind string) {
	switch kind {
	case "down":
		ts.SendKeyDownEvent(services.LogicalKeyboardKeySlash, gluttertest.KeyOpts{Character: "?"})
	case "repeat":
		ts.SendKeyRepeatEvent(services.LogicalKeyboardKeySlash, gluttertest.KeyOpts{Character: "?"})
	case "up":
		ts.SendKeyUpEvent(services.LogicalKeyboardKeySlash)
	}
}

func TestCharacterActivator(t *testing.T) {
	t.Run("is triggered on events with correct character", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewCharacterActivator("?"), func(Intent) { invoked++ }))
		ts.Pump()
		// Press Shift + /
		ts.SendKeyDownEvent(lShift)
		slashKey(ts, "down")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "up")
		ts.SendKeyUpEvent(lShift)
		requireInvoked(t, invoked, 1)
	})

	t.Run("handles repeated events", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewCharacterActivator("?"), func(Intent) { invoked++ }))
		ts.Pump()
		ts.SendKeyDownEvent(lShift)
		slashKey(ts, "down")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "repeat")
		requireInvoked(t, invoked, 2)
		slashKey(ts, "up")
		ts.SendKeyUpEvent(lShift)
		requireInvoked(t, invoked, 2)
	})

	t.Run("rejects repeated events if requested", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewCharacterActivator("?", CharacterActivatorOpts{ExcludeRepeats: true}), func(Intent) { invoked++ }))
		ts.Pump()
		ts.SendKeyDownEvent(lShift)
		slashKey(ts, "down")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "repeat")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "up")
		ts.SendKeyUpEvent(lShift)
		requireInvoked(t, invoked, 1)
	})

	t.Run("handles Alt, Ctrl and Meta", func(t *testing.T) {
		ts := gluttertest.New(t)
		invoked := 0
		ts.PumpWidget(activatorTester(NewCharacterActivator("?", CharacterActivatorOpts{Alt: true, Meta: true, Control: true}), func(Intent) { invoked++ }))
		ts.Pump()
		// Press Shift + /
		ts.SendKeyDownEvent(lShift)
		slashKey(ts, "down")
		slashKey(ts, "up")
		requireInvoked(t, invoked, 0)
		// Press Left Alt + Ctrl + Meta + Shift + /
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyAltLeft)
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyMetaLeft)
		ts.SendKeyDownEvent(lCtrl)
		requireInvoked(t, invoked, 0)
		slashKey(ts, "down")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "up")
		ts.SendKeyUpEvent(lShift)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyMetaLeft)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyAltLeft)
		ts.SendKeyUpEvent(lCtrl)
		requireInvoked(t, invoked, 1)
		invoked = 0
		// Press Right Alt + Ctrl + Meta + Shift + /
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyShiftRight)
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyAltRight)
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyMetaRight)
		ts.SendKeyDownEvent(rCtrl)
		requireInvoked(t, invoked, 0)
		slashKey(ts, "down")
		requireInvoked(t, invoked, 1)
		slashKey(ts, "up")
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyShiftRight)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyMetaRight)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyAltRight)
		ts.SendKeyUpEvent(rCtrl)
		requireInvoked(t, invoked, 1)
	})

	t.Run("accepts works as expected", func(t *testing.T) {
		ts, events := keyEventCollector(t)
		activator := NewCharacterActivator("a")
		noRepeat := NewCharacterActivator("a", CharacterActivatorOpts{ExcludeRepeats: true})
		accepts := func(a ShortcutActivator) bool { return a.Accepts((*events)[len(*events)-1], ts.Binding.Keyboard()) }
		a := gluttertest.KeyOpts{Character: "a"}
		ts.SendKeyDownEvent(keyA, a)
		requireBool(t, "down", accepts(activator), true)
		ts.SendKeyRepeatEvent(keyA, a)
		requireBool(t, "repeat", accepts(activator), true)
		ts.SendKeyUpEvent(keyA)
		requireBool(t, "up", accepts(activator), false)

		ts.SendKeyDownEvent(keyA, a)
		requireBool(t, "no repeat: down", accepts(noRepeat), true)
		ts.SendKeyRepeatEvent(keyA, a)
		requireBool(t, "no repeat: repeat", accepts(noRepeat), false)
		ts.SendKeyUpEvent(keyA)
		requireBool(t, "no repeat: up", accepts(noRepeat), false)
	})

	t.Run("diagnostics", func(t *testing.T) {
		skipUnlessDebug(t)
		for _, tc := range []struct {
			name      string
			activator CharacterActivator
			want      []string
		}{
			{"single key", NewCharacterActivator("A"), []string{"character: 'A'"}},
			{"no repeats", NewCharacterActivator("A", CharacterActivatorOpts{ExcludeRepeats: true}), []string{"character: 'A'", "excluding repeats"}},
			{"combination", NewCharacterActivator("A", CharacterActivatorOpts{Control: true, Meta: true}), []string{"character: Control + Meta + 'A'"}},
		} {
			builder := foundation.NewDiagnosticPropertiesBuilder()
			tc.activator.DebugFillProperties(builder)
			if got := describeProperties(builder); !slices.Equal(got, tc.want) {
				t.Errorf("%s: description = %q, want %q", tc.name, got, tc.want)
			}
		}
	})
}

func requireInvoked(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("invoked = %d, want %d", got, want)
	}
}

func TestCallbackShortcuts(t *testing.T) {
	focused := func() Widget { return NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true}) }
	count := func(n *int) func() { return func() { *n++ } }

	t.Run("trigger on key events", func(t *testing.T) {
		ts := gluttertest.New(t)
		var invokedA, invokedB int
		ts.PumpWidget(NewCallbackShortcuts([]CallbackShortcutEntry{
			{NewSingleActivator(keyA), count(&invokedA)},
			{NewSingleActivator(keyB), count(&invokedB)},
		}, focused()))
		ts.Pump()
		ts.SendKeyDownEvent(keyA)
		ts.Pump()
		if invokedA != 1 || invokedB != 0 {
			t.Fatalf("A = %d, B = %d; want 1, 0", invokedA, invokedB)
		}
		ts.SendKeyUpEvent(keyA)
		if invokedA != 1 || invokedB != 0 {
			t.Fatalf("A = %d, B = %d; want 1, 0", invokedA, invokedB)
		}
		invokedA, invokedB = 0, 0
		ts.SendKeyDownEvent(keyB)
		if invokedA != 0 || invokedB != 1 {
			t.Fatalf("A = %d, B = %d; want 0, 1", invokedA, invokedB)
		}
		ts.SendKeyUpEvent(keyB)
		if invokedA != 0 || invokedB != 1 {
			t.Errorf("A = %d, B = %d; want 0, 1", invokedA, invokedB)
		}
	})

	t.Run("nested CallbackShortcuts stop propagation", func(t *testing.T) {
		ts := gluttertest.New(t)
		var outer, inner int
		ts.PumpWidget(NewCallbackShortcuts([]CallbackShortcutEntry{{NewSingleActivator(keyA), count(&outer)}},
			NewCallbackShortcuts([]CallbackShortcutEntry{{NewSingleActivator(keyA), count(&inner)}}, focused())))
		ts.Pump()
		ts.SendKeyDownEvent(keyA)
		if outer != 0 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
		ts.SendKeyUpEvent(keyA)
		if outer != 0 || inner != 1 {
			t.Errorf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
	})

	t.Run("non-overlapping nested CallbackShortcuts fire appropriately", func(t *testing.T) {
		ts := gluttertest.New(t)
		var outer, inner int
		ts.PumpWidget(NewCallbackShortcuts([]CallbackShortcutEntry{{NewCharacterActivator("b"), count(&outer)}},
			NewCallbackShortcuts([]CallbackShortcutEntry{{NewCharacterActivator("a"), count(&inner)}}, focused())))
		ts.Pump()
		ts.SendKeyDownEvent(keyA, gluttertest.KeyOpts{Character: "a"})
		if outer != 0 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
		ts.SendKeyDownEvent(keyB, gluttertest.KeyOpts{Character: "b"})
		if outer != 1 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 1, 1", outer, inner)
		}
		ts.SendKeyUpEvent(keyA)
		ts.SendKeyUpEvent(keyB)
		if outer != 1 || inner != 1 {
			t.Errorf("outer = %d, inner = %d; want 1, 1", outer, inner)
		}
	})

	t.Run("works correctly with Shortcuts too", func(t *testing.T) {
		ts := gluttertest.New(t)
		var callbackA, callbackB, actionA, actionB int
		ts.PumpWidget(NewActions(ActionMap{
			IntentTypeOf[testIntentV](): callbackIntentAction(func(Intent) any { actionA++; return true }),
			IntentTypeOf[testIntent2](): callbackIntentAction(func(Intent) any { actionB++; return true }),
		}, NewCallbackShortcuts([]CallbackShortcutEntry{{NewCharacterActivator("b"), count(&callbackB)}},
			NewShortcuts(ShortcutMap{
				{NewLogicalKeySet(keyA), testIntentV{}},
				{NewLogicalKeySet(keyB), testIntent2{}},
			}, NewCallbackShortcuts([]CallbackShortcutEntry{{NewCharacterActivator("a"), count(&callbackA)}}, focused())))))
		ts.Pump()
		ts.SendKeyDownEvent(keyA, gluttertest.KeyOpts{Character: "a"})
		if callbackA != 1 || callbackB != 0 || actionA != 0 || actionB != 0 {
			t.Fatalf("callbacks %d/%d, actions %d/%d; want 1/0, 0/0", callbackA, callbackB, actionA, actionB)
		}
		ts.SendKeyUpEvent(keyA)
		callbackA, callbackB, actionA, actionB = 0, 0, 0, 0
		ts.SendKeyDownEvent(keyB, gluttertest.KeyOpts{Character: "b"})
		if callbackA != 0 || callbackB != 0 || actionA != 0 || actionB != 1 {
			t.Errorf("callbacks %d/%d, actions %d/%d; want 0/0, 0/1", callbackA, callbackB, actionA, actionB)
		}
		ts.SendKeyUpEvent(keyB)
	})
}

// testCallbackRegistration is TestCallbackRegistration: it registers shortcuts
// with the ShortcutRegistry above it.
type testCallbackRegistration struct {
	StatefulWidget
	shortcuts          ShortcutMap
	onDependencyUpdate func(ShortcutMap)
	child              Widget
}

func newTestCallbackRegistration(shortcuts ShortcutMap, child Widget, onDependencyUpdate ...func(ShortcutMap)) *testCallbackRegistration {
	w := &testCallbackRegistration{StatefulWidget: NewStatefulWidgetBase(), shortcuts: shortcuts, child: child}
	if len(onDependencyUpdate) > 0 {
		w.onDependencyUpdate = onDependencyUpdate[0]
	}
	return w
}

func (w *testCallbackRegistration) CreateState() State { return &testCallbackRegistrationState{} }

type testCallbackRegistrationState struct {
	StateOf[*testCallbackRegistration]
	token *ShortcutRegistryEntry
}

func (s *testCallbackRegistrationState) DidChangeDependencies() {
	if s.token != nil {
		s.token.Dispose()
	}
	s.token = ShortcutRegistryOf(s.Context()).AddAll(s.Widget().shortcuts)
}

func (s *testCallbackRegistrationState) DidUpdateWidget(old *testCallbackRegistration) {
	if !slices.Equal(s.Widget().shortcuts, old.shortcuts) || s.token == nil {
		if s.token != nil {
			s.token.Dispose()
		}
		s.token = ShortcutRegistryOf(s.Context()).AddAll(s.Widget().shortcuts)
	}
	if s.Widget().onDependencyUpdate != nil {
		s.Widget().onDependencyUpdate(ShortcutRegistryOf(s.Context()).Shortcuts())
	}
}

func (s *testCallbackRegistrationState) Dispose() {
	if s.token != nil {
		s.token.Dispose()
	}
}

func (s *testCallbackRegistrationState) Build(BuildContext) Widget { return s.Widget().child }

func voidCallbackIntent(n *int) Intent { return VoidCallbackIntent{Callback: func() { *n++ }} }

func voidCallbackActions(child Widget) Widget {
	return NewActions(ActionMap{IntentTypeOf[VoidCallbackIntent](): NewVoidCallbackAction()}, child)
}

func TestShortcutRegistrar(t *testing.T) {
	focused := func() Widget { return NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true}) }

	checkAB := func(t *testing.T, ts *gluttertest.Tester, invokedA, invokedB *int) {
		t.Helper()
		ts.Pump()
		ts.SendKeyDownEvent(keyA)
		ts.Pump()
		if *invokedA != 1 || *invokedB != 0 {
			t.Fatalf("A = %d, B = %d; want 1, 0", *invokedA, *invokedB)
		}
		ts.SendKeyUpEvent(keyA)
		*invokedA, *invokedB = 0, 0
		ts.SendKeyDownEvent(keyB)
		if *invokedA != 0 || *invokedB != 1 {
			t.Fatalf("A = %d, B = %d; want 0, 1", *invokedA, *invokedB)
		}
		ts.SendKeyUpEvent(keyB)
	}

	t.Run("trigger ShortcutRegistrar on key events", func(t *testing.T) {
		ts := gluttertest.New(t)
		var a, b int
		ts.PumpWidget(NewShortcutRegistrar(newTestCallbackRegistration(ShortcutMap{
			{NewSingleActivator(keyA), voidCallbackIntent(&a)},
			{NewSingleActivator(keyB), voidCallbackIntent(&b)},
		}, voidCallbackActions(focused()))))
		checkAB(t, ts, &a, &b)
	})

	t.Run("WidgetsApp has a ShortcutRegistrar listening", func(t *testing.T) {
		ts := gluttertest.New(t)
		var a, b int
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: newTestCallbackRegistration(ShortcutMap{
			{NewSingleActivator(keyA), voidCallbackIntent(&a)},
			{NewSingleActivator(keyB), voidCallbackIntent(&b)},
		}, voidCallbackActions(focused()))}))
		checkAB(t, ts, &a, &b)
	})

	t.Run("nested ShortcutRegistrars stop propagation", func(t *testing.T) {
		ts := gluttertest.New(t)
		var outer, inner int
		ts.PumpWidget(NewShortcutRegistrar(newTestCallbackRegistration(
			ShortcutMap{{NewSingleActivator(keyA), voidCallbackIntent(&outer)}},
			NewShortcutRegistrar(newTestCallbackRegistration(
				ShortcutMap{{NewSingleActivator(keyA), voidCallbackIntent(&inner)}}, voidCallbackActions(focused()))))))
		ts.Pump()
		ts.SendKeyDownEvent(keyA)
		if outer != 0 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
		ts.SendKeyUpEvent(keyA)
		if outer != 0 || inner != 1 {
			t.Errorf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
	})

	t.Run("non-overlapping nested ShortcutRegistrars fire appropriately", func(t *testing.T) {
		ts := gluttertest.New(t)
		var outer, inner int
		ts.PumpWidget(NewShortcutRegistrar(newTestCallbackRegistration(
			ShortcutMap{{NewCharacterActivator("b"), voidCallbackIntent(&outer)}},
			NewShortcutRegistrar(newTestCallbackRegistration(
				ShortcutMap{{NewCharacterActivator("a"), voidCallbackIntent(&inner)}}, voidCallbackActions(focused()))))))
		ts.Pump()
		ts.SendKeyDownEvent(keyA, gluttertest.KeyOpts{Character: "a"})
		if outer != 0 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 0, 1", outer, inner)
		}
		ts.SendKeyDownEvent(keyB, gluttertest.KeyOpts{Character: "b"})
		if outer != 1 || inner != 1 {
			t.Fatalf("outer = %d, inner = %d; want 1, 1", outer, inner)
		}
		ts.SendKeyUpEvent(keyA)
		ts.SendKeyUpEvent(keyB)
		if outer != 1 || inner != 1 {
			t.Errorf("outer = %d, inner = %d; want 1, 1", outer, inner)
		}
	})

	t.Run("works correctly with Shortcuts too", func(t *testing.T) {
		ts := gluttertest.New(t)
		var callbackA, callbackB, actionA, actionB int
		ts.PumpWidget(NewActions(ActionMap{
			IntentTypeOf[testIntentV]():        callbackIntentAction(func(Intent) any { actionA++; return true }),
			IntentTypeOf[testIntent2]():        callbackIntentAction(func(Intent) any { actionB++; return true }),
			IntentTypeOf[VoidCallbackIntent](): NewVoidCallbackAction(),
		}, NewShortcutRegistrar(newTestCallbackRegistration(
			ShortcutMap{{NewCharacterActivator("b"), voidCallbackIntent(&callbackB)}},
			NewShortcuts(ShortcutMap{
				{NewSingleActivator(keyA), testIntentV{}},
				{NewSingleActivator(keyB), testIntent2{}},
			}, NewShortcutRegistrar(newTestCallbackRegistration(
				ShortcutMap{{NewCharacterActivator("a"), voidCallbackIntent(&callbackA)}}, focused())))))))
		ts.Pump()
		ts.SendKeyDownEvent(keyA, gluttertest.KeyOpts{Character: "a"})
		if callbackA != 1 || callbackB != 0 || actionA != 0 || actionB != 0 {
			t.Fatalf("callbacks %d/%d, actions %d/%d; want 1/0, 0/0", callbackA, callbackB, actionA, actionB)
		}
		ts.SendKeyUpEvent(keyA)
		callbackA, callbackB, actionA, actionB = 0, 0, 0, 0
		ts.SendKeyDownEvent(keyB, gluttertest.KeyOpts{Character: "b"})
		if callbackA != 0 || callbackB != 0 || actionA != 0 || actionB != 1 {
			t.Errorf("callbacks %d/%d, actions %d/%d; want 0/0, 0/1", callbackA, callbackB, actionA, actionB)
		}
		ts.SendKeyUpEvent(keyB)
	})

	t.Run("updating shortcuts triggers dependency rebuild", func(t *testing.T) {
		ts := gluttertest.New(t)
		var changed []ShortcutMap
		onUpdate := func(m ShortcutMap) { changed = append(changed, m) }
		selectAll := SelectAllTextIntent{Cause: services.SelectionChangedCauseKeyboard}
		both := ShortcutMap{{NewSingleActivator(keyA), selectAll}, {NewSingleActivator(keyB), ActivateIntent{}}}
		onlyA := ShortcutMap{{NewSingleActivator(keyA), selectAll}}
		build := func(shortcuts ShortcutMap) Widget {
			return NewShortcutRegistrar(newTestCallbackRegistration(shortcuts, voidCallbackActions(focused()), onUpdate))
		}
		ts.PumpWidget(build(both))
		ts.PumpWidget(build(onlyA))
		ts.PumpWidget(build(both))
		if len(changed) != 2 {
			t.Fatalf("shortcuts changed %d times, want 2", len(changed))
		}
		if !slices.Equal(changed[1], both) {
			t.Errorf("last shortcuts = %v, want %v", changed[1], both)
		}
	})

	t.Run("sets the debug label on the focus node", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewShortcutRegistrar(NewSizedBoxShrink(nil)))
		node := FocusOf(ts.Element(gluttertest.ByType[*SizedBox]()))
		if node.DebugLabel() != "Shortcuts: <Shortcut Registrar>" {
			t.Errorf("label = %q", node.DebugLabel())
		}
	})
}

func TestShortcutRegistryFailures(t *testing.T) {
	skipUnlessDebug(t)
	newRegistry := func(t *testing.T) *ShortcutRegistry {
		ts := gluttertest.New(t)
		r := NewShortcutRegistry(ts.Binding.Scheduler())
		t.Cleanup(r.Dispose)
		return r
	}
	t.Run("using a disposed token", func(t *testing.T) {
		registry := newRegistry(t)
		token := registry.AddAll(ShortcutMap{{NewSingleActivator(keyA), DoNothingIntent{}}})
		token.Dispose()
		requireTag(t, "ShortcutRegistryEntryDisposed", func() { token.ReplaceAll(ShortcutMap{}) })
	})
	t.Run("setting duplicate bindings", func(t *testing.T) {
		registry := newRegistry(t)
		token := registry.AddAll(ShortcutMap{{NewSingleActivator(keyA), DoNothingIntent{}}})
		requireTag(t, "ShortcutRegistryDuplicate", func() {
			registry.AddAll(ShortcutMap{{NewSingleActivator(keyA), ActivateIntent{}}}).Dispose()
		})
		token.Dispose()
	})
	t.Run("registering nothing", func(t *testing.T) {
		registry := newRegistry(t)
		requireTag(t, "ShortcutRegistryEmpty", func() { registry.AddAll(ShortcutMap{}) })
	})
	t.Run("no registrar above the context", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := containerKey()
		ts.PumpWidget(NewContainer(nil, ContainerOpts{Key: key}))
		requireTag(t, "ShortcutRegistrarNotFound", func() { ShortcutRegistryOf(key.CurrentContext()) })
		if ShortcutRegistryMaybeOf(key.CurrentContext()) != nil {
			t.Error("MaybeOf should return nil")
		}
	})
}

func TestShortcutsFailureTags(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "KeySetDuplicateKey", func() { NewLogicalKeySet(keyA, keyB, keyA) })
	requireTag(t, "KeySetEmpty", func() { NewLogicalKeySetFromSet(services.NewKeySet[services.LogicalKeyboardKey]()) })
	for _, modifier := range []services.LogicalKeyboardKey{
		services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyAltRight, services.LogicalKeyboardKeyMeta,
	} {
		requireTag(t, "SingleActivatorTriggerIsModifier", func() { NewSingleActivator(modifier) })
	}
}
