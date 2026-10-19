package widgets_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from
// packages/flutter/test/widgets/default_text_editing_shortcuts_test.dart. The
// Dart tests put an ActionSpy above an EditableText (step 11.4) and check the last
// intent; here the spy is above a focused box and the tests check the same
// intents. Not ported: the iOS and macOS cases (only the Linux bindings exist, so
// every platform uses them), the web ones, the Home and End cases of Android and
// Windows, which are the Linux ones, and the cases that check the text of an
// EditableText.

// textIntentTypes are the types of the intents the text editing shortcuts make.
var textIntentTypes = []Intent{
	DeleteCharacterIntent{}, DeleteToNextWordBoundaryIntent{}, DeleteToLineBreakIntent{},
	ExtendSelectionByCharacterIntent{}, ExtendSelectionToNextWordBoundaryIntent{}, ExtendSelectionToLineBreakIntent{},
	ExtendSelectionVerticallyToAdjacentLineIntent{}, ExtendSelectionVerticallyToAdjacentPageIntent{},
	ExtendSelectionToNextParagraphBoundaryIntent{}, ExtendSelectionToDocumentBoundaryIntent{},
	SelectAllTextIntent{}, CopySelectionTextIntent{}, PasteTextIntent{}, UndoTextIntent{}, RedoTextIntent{},
	DoNothingAndStopPropagationTextIntent{},
}

// actionSpy is ActionSpy: it records the last intent of the text editing types.
type actionSpy struct{ last Intent }

func (s *actionSpy) actions() ActionMap {
	m := ActionMap{}
	for _, intent := range textIntentTypes {
		m[reflect.TypeOf(intent)] = callbackIntentAction(func(i Intent) any {
			s.last = i
			return nil
		})
	}
	return m
}

// buildSpyAboveFocus is buildSpyAboveEditableText: a WidgetsApp, whose
// DefaultTextEditingShortcuts is above a spy above a focused box.
func buildSpyAboveFocus(spy *actionSpy, node *FocusNode) Widget {
	return testWidgetsApp(testWidgetsAppOpts{home: NewActions(spy.actions(), NewFocus(box100(nil), FocusOpts{FocusNode: node, Autofocus: true}))})
}

// sendKeyCombination sends the modifiers down, then the key, then everything up.
func sendKeyCombination(ts *gluttertest.Tester, a SingleActivator) {
	var modifiers []services.LogicalKeyboardKey
	if a.Control() {
		modifiers = append(modifiers, services.LogicalKeyboardKeyControlLeft)
	}
	if a.Shift() {
		modifiers = append(modifiers, services.LogicalKeyboardKeyShiftLeft)
	}
	if a.Alt() {
		modifiers = append(modifiers, services.LogicalKeyboardKeyAltLeft)
	}
	if a.Meta() {
		modifiers = append(modifiers, services.LogicalKeyboardKeyMetaLeft)
	}
	for _, m := range modifiers {
		ts.SendKeyDownEvent(m)
	}
	ts.SendKeyEvent(a.Trigger())
	for i := len(modifiers) - 1; i >= 0; i-- {
		ts.SendKeyUpEvent(modifiers[i])
	}
}

func TestLinuxNumpadShortcuts(t *testing.T) {
	n := func(i int) services.LogicalKeyboardKey {
		return []services.LogicalKeyboardKey{
			services.LogicalKeyboardKeyNumpad0, services.LogicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad2,
			services.LogicalKeyboardKeyNumpad3, services.LogicalKeyboardKeyNumpad4, services.LogicalKeyboardKeyNumpad5,
			services.LogicalKeyboardKeyNumpad6, services.LogicalKeyboardKeyNumpad7, services.LogicalKeyboardKeyNumpad8,
			services.LogicalKeyboardKeyNumpad9,
		}[i]
	}
	dec := services.LogicalKeyboardKeyNumpadDecimal
	shift, ctrl := SingleActivatorOpts{Shift: true}, SingleActivatorOpts{Shift: true, Control: true}
	type keyCase struct {
		key  services.LogicalKeyboardKey
		opts SingleActivatorOpts
		want Intent
	}
	lockedCases := []keyCase{
		{n(6), shift, NewExtendSelectionByCharacterIntent(true, false)},
		{n(4), shift, NewExtendSelectionByCharacterIntent(false, false)},
		{n(8), shift, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)},
		{n(2), shift, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)},
		{n(9), shift, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)},
		{n(3), shift, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)},
		{n(7), shift, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)},
		{n(1), shift, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)},
		{dec, shift, NewDeleteCharacterIntent(true)},
		{n(6), ctrl, NewExtendSelectionToNextWordBoundaryIntent(true, false)},
		{n(4), ctrl, NewExtendSelectionToNextWordBoundaryIntent(false, false)},
		{n(8), ctrl, NewExtendSelectionToNextParagraphBoundaryIntent(false, false)},
		{n(2), ctrl, NewExtendSelectionToNextParagraphBoundaryIntent(true, false)},
		{dec, ctrl, NewDeleteToNextWordBoundaryIntent(true)},
	}
	unlockedCases := []keyCase{
		{n(6), SingleActivatorOpts{}, NewExtendSelectionByCharacterIntent(true, true)},
		{n(4), SingleActivatorOpts{}, NewExtendSelectionByCharacterIntent(false, true)},
		{n(8), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)},
		{n(2), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)},
		{n(9), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)},
		{n(3), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)},
		{n(7), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)},
		{n(1), SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)},
		{dec, SingleActivatorOpts{}, NewDeleteCharacterIntent(true)},
		{n(6), SingleActivatorOpts{Control: true}, NewExtendSelectionToNextWordBoundaryIntent(true, true)},
		{n(4), SingleActivatorOpts{Control: true}, NewExtendSelectionToNextWordBoundaryIntent(false, true)},
		{n(8), SingleActivatorOpts{Control: true}, NewExtendSelectionToNextParagraphBoundaryIntent(false, true)},
		{n(2), SingleActivatorOpts{Control: true}, NewExtendSelectionToNextParagraphBoundaryIntent(true, true)},
		{dec, SingleActivatorOpts{Control: true}, NewDeleteToNextWordBoundaryIntent(true)},
	}
	for _, tc := range []struct {
		name  string
		lock  bool
		cases []keyCase
	}{
		{"are triggered when numlock is locked", true, lockedCases},
		{"are triggered when numlock is unlocked", false, unlockedCases},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			node := NewFocusNode()
			defer node.Dispose()
			var spy actionSpy
			ts.PumpWidget(buildSpyAboveFocus(&spy, node))
			node.RequestFocus()
			ts.Pump()
			if tc.lock {
				ts.SendKeyEvent(services.LogicalKeyboardKeyNumLock)
			}
			if got := ts.Binding.Keyboard().LockModesEnabled().Contains(services.KeyboardLockModeNumLock); got != tc.lock {
				t.Fatalf("num lock = %v, want %v", got, tc.lock)
			}
			for _, c := range tc.cases {
				spy.last = nil
				sendKeyCombination(ts, NewSingleActivator(c.key, c.opts))
				if spy.last != c.want {
					t.Errorf("%v %+v: intent = %#v, want %#v", c.key, c.opts, spy.last, c.want)
				}
			}
		})
	}

	t.Run("are not triggered with the opposite num lock state", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := NewFocusNode()
		defer node.Dispose()
		var spy actionSpy
		ts.PumpWidget(buildSpyAboveFocus(&spy, node))
		node.RequestFocus()
		ts.Pump()
		// Shift + numpad 4 is a shortcut only while num lock is on.
		sendKeyCombination(ts, NewSingleActivator(services.LogicalKeyboardKeyNumpad4, SingleActivatorOpts{Shift: true}))
		if spy.last != nil {
			t.Errorf("intent = %#v, want none", spy.last)
		}
	})
}

func TestDefaultTextEditingShortcutsKeys(t *testing.T) {
	ts := gluttertest.New(t)
	node := NewFocusNode()
	defer node.Dispose()
	var spy actionSpy
	ts.PumpWidget(buildSpyAboveFocus(&spy, node))
	node.RequestFocus()
	ts.Pump()
	type keyCase struct {
		key  services.LogicalKeyboardKey
		opts SingleActivatorOpts
		want Intent
	}
	keyboard := services.SelectionChangedCauseKeyboard
	arrowLeft, arrowRight := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight
	for _, c := range []keyCase{
		{services.LogicalKeyboardKeyBackspace, SingleActivatorOpts{}, NewDeleteCharacterIntent(false)},
		{services.LogicalKeyboardKeyBackspace, SingleActivatorOpts{Shift: true}, NewDeleteCharacterIntent(false)},
		{services.LogicalKeyboardKeyBackspace, SingleActivatorOpts{Control: true}, NewDeleteToNextWordBoundaryIntent(false)},
		{services.LogicalKeyboardKeyBackspace, SingleActivatorOpts{Alt: true}, NewDeleteToLineBreakIntent(false)},
		{services.LogicalKeyboardKeyDelete, SingleActivatorOpts{}, NewDeleteCharacterIntent(true)},
		{services.LogicalKeyboardKeyDelete, SingleActivatorOpts{Control: true}, NewDeleteToNextWordBoundaryIntent(true)},
		{arrowLeft, SingleActivatorOpts{}, NewExtendSelectionByCharacterIntent(false, true)},
		{arrowRight, SingleActivatorOpts{Shift: true}, NewExtendSelectionByCharacterIntent(true, false)},
		{arrowLeft, SingleActivatorOpts{Control: true}, NewExtendSelectionToNextWordBoundaryIntent(false, true)},
		{arrowRight, SingleActivatorOpts{Control: true, Shift: true}, NewExtendSelectionToNextWordBoundaryIntent(true, false)},
		{arrowLeft, SingleActivatorOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(false, true)},
		{services.LogicalKeyboardKeyArrowUp, SingleActivatorOpts{Alt: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)},
		{services.LogicalKeyboardKeyPageDown, SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)},
		{services.LogicalKeyboardKeyHome, SingleActivatorOpts{}, NewExtendSelectionToLineBreakIntent(false, true)},
		{services.LogicalKeyboardKeyEnd, SingleActivatorOpts{Shift: true}, NewExtendSelectionToLineBreakIntent(true, false)},
		{services.LogicalKeyboardKeyHome, SingleActivatorOpts{Control: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)},
		{services.LogicalKeyboardKeyEnd, SingleActivatorOpts{Control: true, Shift: true}, NewExtendSelectionToDocumentBoundaryIntent(true, false)},
		{services.LogicalKeyboardKeyKeyA, SingleActivatorOpts{Control: true}, SelectAllTextIntent{Cause: keyboard}},
		{services.LogicalKeyboardKeyKeyC, SingleActivatorOpts{Control: true}, NewCopySelectionTextIntentCopy()},
		{services.LogicalKeyboardKeyKeyX, SingleActivatorOpts{Control: true}, NewCopySelectionTextIntentCut(keyboard)},
		{services.LogicalKeyboardKeyKeyV, SingleActivatorOpts{Control: true}, PasteTextIntent{Cause: keyboard}},
		{services.LogicalKeyboardKeyKeyZ, SingleActivatorOpts{Control: true}, UndoTextIntent{Cause: keyboard}},
		{services.LogicalKeyboardKeyKeyZ, SingleActivatorOpts{Control: true, Shift: true}, RedoTextIntent{Cause: keyboard}},
		{services.LogicalKeyboardKeyInsert, SingleActivatorOpts{Shift: true}, PasteTextIntent{Cause: keyboard}},
		{services.LogicalKeyboardKeyDelete, SingleActivatorOpts{Shift: true}, NewCopySelectionTextIntentCut(keyboard)},
		{services.LogicalKeyboardKeySpace, SingleActivatorOpts{}, DoNothingAndStopPropagationTextIntent{}},
		{services.LogicalKeyboardKeyEnter, SingleActivatorOpts{}, DoNothingAndStopPropagationTextIntent{}},
	} {
		spy.last = nil
		sendKeyCombination(ts, NewSingleActivator(c.key, c.opts))
		if spy.last != c.want {
			t.Errorf("%v %+v: intent = %#v, want %#v", c.key, c.opts, spy.last, c.want)
		}
	}
}

func TestTextEditingIntentConstructors(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "CaretMovementIntentInvalid", func() {
		NewExtendSelectionToLineBreakIntent(true, true, ExtendSelectionToLineBreakIntentOpts{CollapseAtReversal: true})
	})
}
