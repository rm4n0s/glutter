package widgets

import (
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
)

// macShortcutMap is _macShortcuts: the bindings of macOS (and of iOS, which has none of its own). They use
// other modifiers for the word and the line than the other platforms: Alt for the word, Command for the line
// and the document, and Control for the Emacs-like keys of the Cocoa text system.
//
// macOS document shortcuts: https://support.apple.com/en-us/HT201236.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts._macShortcuts
func macShortcutMap() ShortcutMap {
	var m ShortcutMap
	for _, shift := range []bool{true, false} {
		m = append(m,
			textShortcut(services.LogicalKeyboardKeyBackspace, shortcutOpts{Shift: shift}, NewDeleteCharacterIntent(false)),
			textShortcut(services.LogicalKeyboardKeyBackspace, shortcutOpts{Alt: true, Shift: shift}, NewDeleteToNextWordBoundaryIntent(false)),
			textShortcut(services.LogicalKeyboardKeyBackspace, shortcutOpts{Meta: true, Shift: shift}, NewDeleteToLineBreakIntent(false)),
			textShortcut(services.LogicalKeyboardKeyDelete, shortcutOpts{Shift: shift}, NewDeleteCharacterIntent(true)),
			textShortcut(services.LogicalKeyboardKeyDelete, shortcutOpts{Alt: true, Shift: shift}, NewDeleteToNextWordBoundaryIntent(true)),
			textShortcut(services.LogicalKeyboardKeyDelete, shortcutOpts{Meta: true, Shift: shift}, NewDeleteToLineBreakIntent(true)),
		)
	}
	keyboard := services.SelectionChangedCauseKeyboard
	left, right, up, down := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight, services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown
	return append(m,
		textShortcut(left, shortcutOpts{}, NewExtendSelectionByCharacterIntent(false, true)),
		textShortcut(right, shortcutOpts{}, NewExtendSelectionByCharacterIntent(true, true)),
		textShortcut(up, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		textShortcut(down, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),

		// Shift + Arrow: extend the selection.
		textShortcut(left, shortcutOpts{Shift: true}, NewExtendSelectionByCharacterIntent(false, false)),
		textShortcut(right, shortcutOpts{Shift: true}, NewExtendSelectionByCharacterIntent(true, false)),
		textShortcut(up, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		textShortcut(down, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),

		textShortcut(left, shortcutOpts{Alt: true}, NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		textShortcut(right, shortcutOpts{Alt: true}, NewExtendSelectionToNextWordBoundaryIntent(true, true)),
		textShortcut(up, shortcutOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(false, true)),
		textShortcut(down, shortcutOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(true, true)),

		textShortcut(left, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false)),
		textShortcut(right, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true)),
		textShortcut(up, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)),
		textShortcut(down, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)),

		textShortcut(left, shortcutOpts{Meta: true}, NewExtendSelectionToLineBreakIntent(false, true)),
		textShortcut(right, shortcutOpts{Meta: true}, NewExtendSelectionToLineBreakIntent(true, true)),
		textShortcut(up, shortcutOpts{Meta: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		textShortcut(down, shortcutOpts{Meta: true}, NewExtendSelectionToDocumentBoundaryIntent(true, true)),

		textShortcut(left, shortcutOpts{Shift: true, Meta: true}, NewExpandSelectionToLineBreakIntent(false)),
		textShortcut(right, shortcutOpts{Shift: true, Meta: true}, NewExpandSelectionToLineBreakIntent(true)),
		textShortcut(up, shortcutOpts{Shift: true, Meta: true}, NewExpandSelectionToDocumentBoundaryIntent(false)),
		textShortcut(down, shortcutOpts{Shift: true, Meta: true}, NewExpandSelectionToDocumentBoundaryIntent(true)),

		textShortcut(services.LogicalKeyboardKeyKeyT, shortcutOpts{Control: true}, TransposeCharactersIntent{}),

		textShortcut(services.LogicalKeyboardKeyHome, shortcutOpts{}, NewScrollToDocumentBoundaryIntent(false)),
		textShortcut(services.LogicalKeyboardKeyEnd, shortcutOpts{}, NewScrollToDocumentBoundaryIntent(true)),
		textShortcut(services.LogicalKeyboardKeyHome, shortcutOpts{Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(false)),
		textShortcut(services.LogicalKeyboardKeyEnd, shortcutOpts{Shift: true}, NewExpandSelectionToDocumentBoundaryIntent(true)),

		textShortcut(services.LogicalKeyboardKeyPageUp, shortcutOpts{}, ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}),
		textShortcut(services.LogicalKeyboardKeyPageDown, shortcutOpts{}, ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}),
		textShortcut(services.LogicalKeyboardKeyPageUp, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		textShortcut(services.LogicalKeyboardKeyPageDown, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),

		textShortcut(services.LogicalKeyboardKeyKeyX, shortcutOpts{Meta: true}, NewCopySelectionTextIntentCut(keyboard)),
		textShortcut(services.LogicalKeyboardKeyKeyC, shortcutOpts{Meta: true}, NewCopySelectionTextIntentCopy()),
		textShortcut(services.LogicalKeyboardKeyKeyV, shortcutOpts{Meta: true}, PasteTextIntent{Cause: keyboard}),
		textShortcut(services.LogicalKeyboardKeyKeyA, shortcutOpts{Meta: true}, SelectAllTextIntent{Cause: keyboard}),
		textShortcut(services.LogicalKeyboardKeyKeyZ, shortcutOpts{Meta: true}, UndoTextIntent{Cause: keyboard}),
		textShortcut(services.LogicalKeyboardKeyKeyZ, shortcutOpts{Shift: true, Meta: true}, RedoTextIntent{Cause: keyboard}),

		textShortcut(services.LogicalKeyboardKeyKeyE, shortcutOpts{Control: true}, NewExtendSelectionToLineBreakIntent(true, true)),
		textShortcut(services.LogicalKeyboardKeyKeyA, shortcutOpts{Control: true}, NewExtendSelectionToLineBreakIntent(false, true)),
		textShortcut(services.LogicalKeyboardKeyKeyF, shortcutOpts{Control: true}, NewExtendSelectionByCharacterIntent(true, true)),
		textShortcut(services.LogicalKeyboardKeyKeyB, shortcutOpts{Control: true}, NewExtendSelectionByCharacterIntent(false, true)),
		textShortcut(services.LogicalKeyboardKeyKeyN, shortcutOpts{Control: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		textShortcut(services.LogicalKeyboardKeyKeyP, shortcutOpts{Control: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),

		// These keys go to the input method when a field is focused, not to other Shortcuts.
		textShortcut(services.LogicalKeyboardKeySpace, shortcutOpts{}, DoNothingAndStopPropagationTextIntent{}),
		textShortcut(services.LogicalKeyboardKeyEnter, shortcutOpts{}, DoNothingAndStopPropagationTextIntent{}),
		// No effect on text editing on this platform: End, Home, Control + Shift? + End, Control + Shift? +
		// Home, and Control + Shift? + Z.
	)
}

// IntentForMacOSSelector maps the selector of NSStandardKeyBindingResponding (what the Cocoa text system
// asks of a text field for a key it does not insert: "moveLeft:", "deleteWordBackward:") to the intent of the
// framework. It returns nil for a selector that is not known.
//
// Flutter: widgets/default_text_editing_shortcuts.dart intentForMacOSSelector
func IntentForMacOSSelector(selectorName string) Intent {
	if f, ok := macSelectorIntents[selectorName]; ok {
		return f()
	}
	return nil
}

// macSelectorIntents makes the intent of each selector; a function, so that every call has its own intent.
//
// Deviation: the first six, the commands of the Edit menu that AppKit sends to the first responder (cut:, copy:,
// paste:, selectAll:, undo:, redo:), are Glutter's: Flutter's template menu reaches them some other way
// (SPEC §12.10).
var macSelectorIntents = map[string]func() Intent{
	"cut:":       func() Intent { return NewCopySelectionTextIntentCut(services.SelectionChangedCauseKeyboard) },
	"copy:":      func() Intent { return NewCopySelectionTextIntentCopy() },
	"paste:":     func() Intent { return PasteTextIntent{Cause: services.SelectionChangedCauseKeyboard} },
	"selectAll:": func() Intent { return SelectAllTextIntent{Cause: services.SelectionChangedCauseKeyboard} },
	"undo:":      func() Intent { return UndoTextIntent{Cause: services.SelectionChangedCauseKeyboard} },
	"redo:":      func() Intent { return RedoTextIntent{Cause: services.SelectionChangedCauseKeyboard} },

	"deleteBackward:":          func() Intent { return NewDeleteCharacterIntent(false) },
	"deleteWordBackward:":      func() Intent { return NewDeleteToNextWordBoundaryIntent(false) },
	"deleteToBeginningOfLine:": func() Intent { return NewDeleteToLineBreakIntent(false) },
	"deleteForward:":           func() Intent { return NewDeleteCharacterIntent(true) },
	"deleteWordForward:":       func() Intent { return NewDeleteToNextWordBoundaryIntent(true) },
	"deleteToEndOfLine:":       func() Intent { return NewDeleteToLineBreakIntent(true) },

	"moveLeft:":     func() Intent { return NewExtendSelectionByCharacterIntent(false, true) },
	"moveRight:":    func() Intent { return NewExtendSelectionByCharacterIntent(true, true) },
	"moveForward:":  func() Intent { return NewExtendSelectionByCharacterIntent(true, true) },
	"moveBackward:": func() Intent { return NewExtendSelectionByCharacterIntent(false, true) },

	"moveUp:":   func() Intent { return NewExtendSelectionVerticallyToAdjacentLineIntent(false, true) },
	"moveDown:": func() Intent { return NewExtendSelectionVerticallyToAdjacentLineIntent(true, true) },

	"moveLeftAndModifySelection:":  func() Intent { return NewExtendSelectionByCharacterIntent(false, false) },
	"moveRightAndModifySelection:": func() Intent { return NewExtendSelectionByCharacterIntent(true, false) },
	"moveUpAndModifySelection:":    func() Intent { return NewExtendSelectionVerticallyToAdjacentLineIntent(false, false) },
	"moveDownAndModifySelection:":  func() Intent { return NewExtendSelectionVerticallyToAdjacentLineIntent(true, false) },

	"moveWordLeft:":               func() Intent { return NewExtendSelectionToNextWordBoundaryIntent(false, true) },
	"moveWordRight:":              func() Intent { return NewExtendSelectionToNextWordBoundaryIntent(true, true) },
	"moveToBeginningOfParagraph:": func() Intent { return NewExtendSelectionToLineBreakIntent(false, true) },
	"moveToEndOfParagraph:":       func() Intent { return NewExtendSelectionToLineBreakIntent(true, true) },

	"moveWordLeftAndModifySelection:":  func() Intent { return NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false) },
	"moveWordRightAndModifySelection:": func() Intent { return NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true) },
	"moveParagraphBackwardAndModifySelection:": func() Intent {
		return NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)
	},
	"moveParagraphForwardAndModifySelection:": func() Intent {
		return NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)
	},

	"moveToLeftEndOfLine:":       func() Intent { return NewExtendSelectionToLineBreakIntent(false, true) },
	"moveToRightEndOfLine:":      func() Intent { return NewExtendSelectionToLineBreakIntent(true, true) },
	"moveToBeginningOfDocument:": func() Intent { return NewExtendSelectionToDocumentBoundaryIntent(false, true) },
	"moveToEndOfDocument:":       func() Intent { return NewExtendSelectionToDocumentBoundaryIntent(true, true) },

	"moveToLeftEndOfLineAndModifySelection:":       func() Intent { return NewExpandSelectionToLineBreakIntent(false) },
	"moveToRightEndOfLineAndModifySelection:":      func() Intent { return NewExpandSelectionToLineBreakIntent(true) },
	"moveToBeginningOfDocumentAndModifySelection:": func() Intent { return NewExpandSelectionToDocumentBoundaryIntent(false) },
	"moveToEndOfDocumentAndModifySelection:":       func() Intent { return NewExpandSelectionToDocumentBoundaryIntent(true) },

	"transpose:": func() Intent { return TransposeCharactersIntent{} },

	"scrollToBeginningOfDocument:": func() Intent { return NewScrollToDocumentBoundaryIntent(false) },
	"scrollToEndOfDocument:":       func() Intent { return NewScrollToDocumentBoundaryIntent(true) },

	"scrollPageUp:": func() Intent { return ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage} },
	"scrollPageDown:": func() Intent {
		return ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}
	},
	"pageUpAndModifySelection:": func() Intent {
		return NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)
	},
	"pageDownAndModifySelection:": func() Intent {
		return NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)
	},

	// Escape when there is no selection popup of the input method.
	"cancelOperation:": func() Intent { return DismissIntent{} },
	// Tab when there is no selection of the input method.
	"insertTab:":     func() Intent { return NextFocusIntent{} },
	"insertBacktab:": func() Intent { return PreviousFocusIntent{} },
}
