package widgets_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	. "github.com/rm4n0s/glutter/widgets"
)

// macActionSpy is actionSpy for the intent types of the macOS bindings too.
func macActionSpy() (*actionSpy, ActionMap) {
	s := &actionSpy{}
	m := s.actions()
	for _, intent := range []Intent{
		ExpandSelectionToLineBreakIntent{}, ExpandSelectionToDocumentBoundaryIntent{}, TransposeCharactersIntent{},
		ScrollToDocumentBoundaryIntent{}, ScrollIntent{}, ExtendSelectionToNextWordBoundaryOrCaretLocationIntent{},
		ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent{},
	} {
		m[reflect.TypeOf(intent)] = callbackIntentAction(func(i Intent) any {
			s.last = i
			return nil
		})
	}
	return s, m
}

// TestMacTextShortcuts is the macOS cases of default_text_editing_shortcuts_test.dart, and the other
// bindings of _macShortcuts: Alt for the word, Command for the line and the document, Control for the
// Emacs keys.
//
// Flutter: packages/flutter/test/widgets/default_text_editing_shortcuts_test.dart
func TestMacTextShortcuts(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformMacOS, foundation.TargetPlatformIOS} {
		t.Run(platform.String(), func(t *testing.T) {
			onPlatform(t, platform)
			ts := gluttertest.New(t)
			node := NewFocusNode()
			defer node.Dispose()
			spy, actions := macActionSpy()
			ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewActions(actions, NewFocus(box100(nil), FocusOpts{FocusNode: node, Autofocus: true}))}))
			node.RequestFocus()
			ts.Pump()
			type keyCase struct {
				key  services.LogicalKeyboardKey
				opts SingleActivatorOpts
				want Intent
			}
			keyboard := services.SelectionChangedCauseKeyboard
			left, right := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight
			up, down := services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown
			bs, del := services.LogicalKeyboardKeyBackspace, services.LogicalKeyboardKeyDelete
			home, end := services.LogicalKeyboardKeyHome, services.LogicalKeyboardKeyEnd
			for _, c := range []keyCase{
				{bs, SingleActivatorOpts{}, NewDeleteCharacterIntent(false)},
				{bs, SingleActivatorOpts{Shift: true}, NewDeleteCharacterIntent(false)},
				{bs, SingleActivatorOpts{Alt: true}, NewDeleteToNextWordBoundaryIntent(false)},
				{bs, SingleActivatorOpts{Meta: true}, NewDeleteToLineBreakIntent(false)},
				{del, SingleActivatorOpts{}, NewDeleteCharacterIntent(true)},
				{del, SingleActivatorOpts{Alt: true, Shift: true}, NewDeleteToNextWordBoundaryIntent(true)},
				{del, SingleActivatorOpts{Meta: true}, NewDeleteToLineBreakIntent(true)},
				{left, SingleActivatorOpts{}, NewExtendSelectionByCharacterIntent(false, true)},
				{right, SingleActivatorOpts{Shift: true}, NewExtendSelectionByCharacterIntent(true, false)},
				{up, SingleActivatorOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)},
				{down, SingleActivatorOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)},
				{left, SingleActivatorOpts{Alt: true}, NewExtendSelectionToNextWordBoundaryIntent(false, true)},
				{right, SingleActivatorOpts{Alt: true}, NewExtendSelectionToNextWordBoundaryIntent(true, true)},
				{up, SingleActivatorOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(false, true)},
				{down, SingleActivatorOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(true, true)},
				{left, SingleActivatorOpts{Alt: true, Shift: true}, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false)},
				{right, SingleActivatorOpts{Alt: true, Shift: true}, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true)},
				{up, SingleActivatorOpts{Alt: true, Shift: true}, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)},
				{down, SingleActivatorOpts{Alt: true, Shift: true}, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)},
				{left, SingleActivatorOpts{Meta: true}, NewExtendSelectionToLineBreakIntent(false, true)},
				{right, SingleActivatorOpts{Meta: true}, NewExtendSelectionToLineBreakIntent(true, true)},
				{up, SingleActivatorOpts{Meta: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)},
				{down, SingleActivatorOpts{Meta: true}, NewExtendSelectionToDocumentBoundaryIntent(true, true)},
				{left, SingleActivatorOpts{Meta: true, Shift: true}, NewExpandSelectionToLineBreakIntent(false)},
				{right, SingleActivatorOpts{Meta: true, Shift: true}, NewExpandSelectionToLineBreakIntent(true)},
				{up, SingleActivatorOpts{Meta: true, Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(false)},
				{down, SingleActivatorOpts{Meta: true, Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(true)},
				{services.LogicalKeyboardKeyKeyT, SingleActivatorOpts{Control: true}, TransposeCharactersIntent{}},
				{home, SingleActivatorOpts{}, NewScrollToDocumentBoundaryIntent(false)},
				{end, SingleActivatorOpts{}, NewScrollToDocumentBoundaryIntent(true)},
				{home, SingleActivatorOpts{Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(false)},
				{end, SingleActivatorOpts{Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(true)},
				{services.LogicalKeyboardKeyPageUp, SingleActivatorOpts{}, ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}},
				{services.LogicalKeyboardKeyPageDown, SingleActivatorOpts{}, ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}},
				{services.LogicalKeyboardKeyPageUp, SingleActivatorOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)},
				{services.LogicalKeyboardKeyKeyX, SingleActivatorOpts{Meta: true}, NewCopySelectionTextIntentCut(keyboard)},
				{services.LogicalKeyboardKeyKeyC, SingleActivatorOpts{Meta: true}, NewCopySelectionTextIntentCopy()},
				{services.LogicalKeyboardKeyKeyV, SingleActivatorOpts{Meta: true}, PasteTextIntent{Cause: keyboard}},
				{services.LogicalKeyboardKeyKeyA, SingleActivatorOpts{Meta: true}, SelectAllTextIntent{Cause: keyboard}},
				{services.LogicalKeyboardKeyKeyZ, SingleActivatorOpts{Meta: true}, UndoTextIntent{Cause: keyboard}},
				{services.LogicalKeyboardKeyKeyZ, SingleActivatorOpts{Meta: true, Shift: true}, RedoTextIntent{Cause: keyboard}},
				{services.LogicalKeyboardKeyKeyE, SingleActivatorOpts{Control: true}, NewExtendSelectionToLineBreakIntent(true, true)},
				{services.LogicalKeyboardKeyKeyA, SingleActivatorOpts{Control: true}, NewExtendSelectionToLineBreakIntent(false, true)},
				{services.LogicalKeyboardKeyKeyF, SingleActivatorOpts{Control: true}, NewExtendSelectionByCharacterIntent(true, true)},
				{services.LogicalKeyboardKeyKeyB, SingleActivatorOpts{Control: true}, NewExtendSelectionByCharacterIntent(false, true)},
				{services.LogicalKeyboardKeyKeyN, SingleActivatorOpts{Control: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)},
				{services.LogicalKeyboardKeyKeyP, SingleActivatorOpts{Control: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)},
				{services.LogicalKeyboardKeySpace, SingleActivatorOpts{}, DoNothingAndStopPropagationTextIntent{}},
				{services.LogicalKeyboardKeyEnter, SingleActivatorOpts{}, DoNothingAndStopPropagationTextIntent{}},
			} {
				spy.last = nil
				sendKeyCombination(ts, NewSingleActivator(c.key, c.opts))
				if spy.last != c.want {
					t.Errorf("%v %+v: intent = %#v, want %#v", c.key, c.opts, spy.last, c.want)
				}
			}
			// Control + Z is a shortcut of the other platforms: here it is no text editing.
			spy.last = nil
			sendKeyCombination(ts, NewSingleActivator(services.LogicalKeyboardKeyKeyZ, SingleActivatorOpts{Control: true}))
			if spy.last != nil {
				t.Errorf("Control+Z: intent = %#v, want none", spy.last)
			}
		})
	}
}

func TestIntentForMacOSSelector(t *testing.T) {
	tests := []struct {
		selector string
		want     Intent
	}{
		{"deleteBackward:", NewDeleteCharacterIntent(false)},
		{"deleteWordBackward:", NewDeleteToNextWordBoundaryIntent(false)},
		{"deleteToBeginningOfLine:", NewDeleteToLineBreakIntent(false)},
		{"deleteForward:", NewDeleteCharacterIntent(true)},
		{"deleteWordForward:", NewDeleteToNextWordBoundaryIntent(true)},
		{"deleteToEndOfLine:", NewDeleteToLineBreakIntent(true)},
		{"moveLeft:", NewExtendSelectionByCharacterIntent(false, true)},
		{"moveRight:", NewExtendSelectionByCharacterIntent(true, true)},
		{"moveForward:", NewExtendSelectionByCharacterIntent(true, true)},
		{"moveBackward:", NewExtendSelectionByCharacterIntent(false, true)},
		{"moveUp:", NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)},
		{"moveDown:", NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)},
		{"moveLeftAndModifySelection:", NewExtendSelectionByCharacterIntent(false, false)},
		{"moveRightAndModifySelection:", NewExtendSelectionByCharacterIntent(true, false)},
		{"moveUpAndModifySelection:", NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)},
		{"moveDownAndModifySelection:", NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)},
		{"moveWordLeft:", NewExtendSelectionToNextWordBoundaryIntent(false, true)},
		{"moveWordRight:", NewExtendSelectionToNextWordBoundaryIntent(true, true)},
		{"moveToBeginningOfParagraph:", NewExtendSelectionToLineBreakIntent(false, true)},
		{"moveToEndOfParagraph:", NewExtendSelectionToLineBreakIntent(true, true)},
		{"moveWordLeftAndModifySelection:", NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false)},
		{"moveWordRightAndModifySelection:", NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true)},
		{"moveParagraphBackwardAndModifySelection:", NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)},
		{"moveParagraphForwardAndModifySelection:", NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)},
		{"moveToLeftEndOfLine:", NewExtendSelectionToLineBreakIntent(false, true)},
		{"moveToRightEndOfLine:", NewExtendSelectionToLineBreakIntent(true, true)},
		{"moveToBeginningOfDocument:", NewExtendSelectionToDocumentBoundaryIntent(false, true)},
		{"moveToEndOfDocument:", NewExtendSelectionToDocumentBoundaryIntent(true, true)},
		{"moveToLeftEndOfLineAndModifySelection:", NewExpandSelectionToLineBreakIntent(false)},
		{"moveToRightEndOfLineAndModifySelection:", NewExpandSelectionToLineBreakIntent(true)},
		{"moveToBeginningOfDocumentAndModifySelection:", NewExpandSelectionToDocumentBoundaryIntent(false)},
		{"moveToEndOfDocumentAndModifySelection:", NewExpandSelectionToDocumentBoundaryIntent(true)},
		{"transpose:", TransposeCharactersIntent{}},
		{"scrollToBeginningOfDocument:", NewScrollToDocumentBoundaryIntent(false)},
		{"scrollToEndOfDocument:", NewScrollToDocumentBoundaryIntent(true)},
		{"scrollPageUp:", ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}},
		{"scrollPageDown:", ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}},
		{"pageUpAndModifySelection:", NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)},
		{"pageDownAndModifySelection:", NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)},
		{"cancelOperation:", DismissIntent{}},
		{"insertTab:", NextFocusIntent{}},
		{"insertBacktab:", PreviousFocusIntent{}},
		{"noSuchSelector:", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			if got := IntentForMacOSSelector(tt.selector); got != tt.want {
				t.Errorf("IntentForMacOSSelector(%q) = %#v, want %#v", tt.selector, got, tt.want)
			}
		})
	}
}
