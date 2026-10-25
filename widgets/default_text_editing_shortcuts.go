package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
)

// DefaultTextEditingShortcuts is a widget that provides the default key bindings
// of the text editing intents (a Shortcuts widget), for the text fields below it.
// Actions for the intents are provided by the text fields.
//
// Deviation: only the Linux and Windows bindings are ported (SPEC §12.10); every
// target platform but Windows uses the Linux ones.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts
type DefaultTextEditingShortcuts struct {
	StatelessWidget
	child Widget
}

// NewDefaultTextEditingShortcuts creates the widget.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts.new
func NewDefaultTextEditingShortcuts(child Widget, opts ...WidgetOpts) *DefaultTextEditingShortcuts {
	return &DefaultTextEditingShortcuts{StatelessWidget: NewStatelessWidgetBase(opts...), child: child}
}

// Child is the widget below this widget in the tree.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts.child
func (w *DefaultTextEditingShortcuts) Child() Widget { return w.child }

// Build returns the Shortcuts widget with the bindings of the platform.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts.build
func (w *DefaultTextEditingShortcuts) Build(BuildContext) Widget {
	return NewShortcuts(DefaultTextEditingShortcutMap(), w.child,
		ShortcutsOpts{DebugLabel: "<Default Text Editing Shortcuts>"})
}

// DefaultTextEditingShortcutMap returns the default text editing bindings of the
// current platform: the Windows bindings when the target platform is Windows, the
// Linux bindings on every other (see DefaultTextEditingShortcuts).
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts._shortcuts
func DefaultTextEditingShortcutMap() ShortcutMap {
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformWindows {
		return windowsShortcutMap()
	}
	m := make(ShortcutMap, 0, 160)
	m = append(m, commonTextShortcuts()...)
	m = append(m, clipboardTextShortcuts()...)
	m = append(m, linuxNumpadTextShortcuts()...)
	m = append(m, linuxTextShortcuts()...)
	return m
}

type shortcutOpts = SingleActivatorOpts

func textShortcut(key services.LogicalKeyboardKey, opts shortcutOpts, intent Intent) ShortcutEntry {
	return ShortcutEntry{Activator: NewSingleActivator(key, opts), Intent: intent}
}

func commonTextShortcuts() ShortcutMap {
	k := services.LogicalKeyboardKeyBackspace
	del := services.LogicalKeyboardKeyDelete
	left, right := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight
	up, down := services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown
	pgUp, pgDown := services.LogicalKeyboardKeyPageUp, services.LogicalKeyboardKeyPageDown
	var m ShortcutMap
	for _, shift := range []bool{true, false} {
		m = append(m,
			textShortcut(k, shortcutOpts{Shift: shift}, NewDeleteCharacterIntent(false)),
			textShortcut(k, shortcutOpts{Control: true, Shift: shift}, NewDeleteToNextWordBoundaryIntent(false)),
			textShortcut(k, shortcutOpts{Alt: true, Shift: shift}, NewDeleteToLineBreakIntent(false)),
			textShortcut(del, shortcutOpts{Control: true, Shift: shift}, NewDeleteToNextWordBoundaryIntent(true)),
			textShortcut(del, shortcutOpts{Alt: true, Shift: shift}, NewDeleteToLineBreakIntent(true)),
		)
	}
	m = append(m,
		textShortcut(del, shortcutOpts{}, NewDeleteCharacterIntent(true)),

		// Arrow: move the selection.
		textShortcut(left, shortcutOpts{}, NewExtendSelectionByCharacterIntent(false, true)),
		textShortcut(right, shortcutOpts{}, NewExtendSelectionByCharacterIntent(true, true)),
		textShortcut(up, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		textShortcut(down, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),

		// Shift + arrow: extend the selection.
		textShortcut(left, shortcutOpts{Shift: true}, NewExtendSelectionByCharacterIntent(false, false)),
		textShortcut(right, shortcutOpts{Shift: true}, NewExtendSelectionByCharacterIntent(true, false)),
		textShortcut(up, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		textShortcut(down, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),

		// Alt + arrow: the line break and the document boundary.
		textShortcut(left, shortcutOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(false, true)),
		textShortcut(right, shortcutOpts{Alt: true}, NewExtendSelectionToLineBreakIntent(true, true)),
		textShortcut(up, shortcutOpts{Alt: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		textShortcut(down, shortcutOpts{Alt: true}, NewExtendSelectionToDocumentBoundaryIntent(true, true)),

		textShortcut(left, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToLineBreakIntent(false, false)),
		textShortcut(right, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToLineBreakIntent(true, false)),
		textShortcut(up, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToDocumentBoundaryIntent(false, false)),
		textShortcut(down, shortcutOpts{Shift: true, Alt: true}, NewExtendSelectionToDocumentBoundaryIntent(true, false)),

		// Control + arrow: the word boundary.
		textShortcut(left, shortcutOpts{Control: true}, NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		textShortcut(right, shortcutOpts{Control: true}, NewExtendSelectionToNextWordBoundaryIntent(true, true)),

		textShortcut(left, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToNextWordBoundaryIntent(false, false)),
		textShortcut(right, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToNextWordBoundaryIntent(true, false)),

		textShortcut(up, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToNextParagraphBoundaryIntent(false, false)),
		textShortcut(down, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToNextParagraphBoundaryIntent(true, false)),

		// Page up and down.
		textShortcut(pgUp, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)),
		textShortcut(pgDown, shortcutOpts{}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)),
		textShortcut(pgUp, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		textShortcut(pgDown, shortcutOpts{Shift: true}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),
	)
	return m
}

func clipboardTextShortcuts() ShortcutMap {
	keyboard := services.SelectionChangedCauseKeyboard
	return ShortcutMap{
		textShortcut(services.LogicalKeyboardKeyKeyX, shortcutOpts{Control: true}, NewCopySelectionTextIntentCut(keyboard)),
		textShortcut(services.LogicalKeyboardKeyKeyC, shortcutOpts{Control: true}, NewCopySelectionTextIntentCopy()),
		textShortcut(services.LogicalKeyboardKeyKeyV, shortcutOpts{Control: true}, PasteTextIntent{Cause: keyboard}),

		textShortcut(services.LogicalKeyboardKeyDelete, shortcutOpts{Shift: true}, NewCopySelectionTextIntentCut(keyboard)),
		textShortcut(services.LogicalKeyboardKeyInsert, shortcutOpts{Control: true}, NewCopySelectionTextIntentCopy()),
		textShortcut(services.LogicalKeyboardKeyInsert, shortcutOpts{Shift: true}, PasteTextIntent{Cause: keyboard}),

		textShortcut(services.LogicalKeyboardKeyKeyA, shortcutOpts{Control: true}, SelectAllTextIntent{Cause: keyboard}),
		textShortcut(services.LogicalKeyboardKeyKeyZ, shortcutOpts{Control: true}, UndoTextIntent{Cause: keyboard}),
		textShortcut(services.LogicalKeyboardKeyKeyZ, shortcutOpts{Shift: true, Control: true}, RedoTextIntent{Cause: keyboard}),

		// Space and Enter are taken by the text field, not by the activation shortcuts.
		textShortcut(services.LogicalKeyboardKeySpace, shortcutOpts{}, DoNothingAndStopPropagationTextIntent{}),
		textShortcut(services.LogicalKeyboardKeyEnter, shortcutOpts{}, DoNothingAndStopPropagationTextIntent{}),
	}
}

func linuxNumpadTextShortcuts() ShortcutMap {
	const locked, unlocked = LockStateLocked, LockStateUnlocked
	key := func(n int) services.LogicalKeyboardKey {
		return [...]services.LogicalKeyboardKey{
			services.LogicalKeyboardKeyNumpad0, services.LogicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad2,
			services.LogicalKeyboardKeyNumpad3, services.LogicalKeyboardKeyNumpad4, services.LogicalKeyboardKeyNumpad5,
			services.LogicalKeyboardKeyNumpad6, services.LogicalKeyboardKeyNumpad7, services.LogicalKeyboardKeyNumpad8,
			services.LogicalKeyboardKeyNumpad9,
		}[n]
	}
	dec := services.LogicalKeyboardKeyNumpadDecimal
	return ShortcutMap{
		// Num Lock on, with Shift: the digit keys are the arrow keys.
		textShortcut(key(6), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionByCharacterIntent(true, false)),
		textShortcut(key(4), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionByCharacterIntent(false, false)),
		textShortcut(key(8), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		textShortcut(key(2), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),

		textShortcut(key(6), shortcutOpts{Shift: true, Control: true, NumLock: locked}, NewExtendSelectionToNextWordBoundaryIntent(true, false)),
		textShortcut(key(4), shortcutOpts{Shift: true, Control: true, NumLock: locked}, NewExtendSelectionToNextWordBoundaryIntent(false, false)),
		textShortcut(key(8), shortcutOpts{Shift: true, Control: true, NumLock: locked}, NewExtendSelectionToNextParagraphBoundaryIntent(false, false)),
		textShortcut(key(2), shortcutOpts{Shift: true, Control: true, NumLock: locked}, NewExtendSelectionToNextParagraphBoundaryIntent(true, false)),

		textShortcut(key(9), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		textShortcut(key(3), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),

		textShortcut(key(7), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		textShortcut(key(1), shortcutOpts{Shift: true, NumLock: locked}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),

		textShortcut(dec, shortcutOpts{Shift: true, NumLock: locked}, NewDeleteCharacterIntent(true)),
		textShortcut(dec, shortcutOpts{Shift: true, Control: true, NumLock: locked}, NewDeleteToNextWordBoundaryIntent(true)),

		// Num Lock off: the digit keys are the navigation keys.
		textShortcut(key(6), shortcutOpts{NumLock: unlocked}, NewExtendSelectionByCharacterIntent(true, true)),
		textShortcut(key(4), shortcutOpts{NumLock: unlocked}, NewExtendSelectionByCharacterIntent(false, true)),
		textShortcut(key(8), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		textShortcut(key(2), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),

		textShortcut(key(6), shortcutOpts{Control: true, NumLock: unlocked}, NewExtendSelectionToNextWordBoundaryIntent(true, true)),
		textShortcut(key(4), shortcutOpts{Control: true, NumLock: unlocked}, NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		textShortcut(key(8), shortcutOpts{Control: true, NumLock: unlocked}, NewExtendSelectionToNextParagraphBoundaryIntent(false, true)),
		textShortcut(key(2), shortcutOpts{Control: true, NumLock: unlocked}, NewExtendSelectionToNextParagraphBoundaryIntent(true, true)),

		textShortcut(key(9), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)),
		textShortcut(key(3), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)),

		textShortcut(key(7), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		textShortcut(key(1), shortcutOpts{NumLock: unlocked}, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),

		textShortcut(dec, shortcutOpts{NumLock: unlocked}, NewDeleteCharacterIntent(true)),
		textShortcut(dec, shortcutOpts{Control: true, NumLock: unlocked}, NewDeleteToNextWordBoundaryIntent(true)),
	}
}

func linuxTextShortcuts() ShortcutMap {
	home, end := services.LogicalKeyboardKeyHome, services.LogicalKeyboardKeyEnd
	return ShortcutMap{
		textShortcut(home, shortcutOpts{}, NewExtendSelectionToLineBreakIntent(false, true)),
		textShortcut(end, shortcutOpts{}, NewExtendSelectionToLineBreakIntent(true, true)),
		textShortcut(home, shortcutOpts{Shift: true}, NewExtendSelectionToLineBreakIntent(false, false)),
		textShortcut(end, shortcutOpts{Shift: true}, NewExtendSelectionToLineBreakIntent(true, false)),
		textShortcut(home, shortcutOpts{Control: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		textShortcut(end, shortcutOpts{Control: true}, NewExtendSelectionToDocumentBoundaryIntent(true, true)),
		textShortcut(home, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToDocumentBoundaryIntent(false, false)),
		textShortcut(end, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToDocumentBoundaryIntent(true, false)),
	}
}

// windowsShortcutMap is _windowsShortcuts: the common and the clipboard shortcuts, and Home and End,
// which continue at a soft wrap. Page Up and Page Down are in the common shortcuts.
//
// Flutter: widgets/default_text_editing_shortcuts.dart DefaultTextEditingShortcuts._windowsShortcuts
func windowsShortcutMap() ShortcutMap {
	home, end := services.LogicalKeyboardKeyHome, services.LogicalKeyboardKeyEnd
	wrap := ExtendSelectionToLineBreakIntentOpts{ContinuesAtWrap: true}
	m := make(ShortcutMap, 0, 100)
	m = append(m, commonTextShortcuts()...)
	m = append(m, clipboardTextShortcuts()...)
	return append(m,
		textShortcut(home, shortcutOpts{}, NewExtendSelectionToLineBreakIntent(false, true, wrap)),
		textShortcut(end, shortcutOpts{}, NewExtendSelectionToLineBreakIntent(true, true, wrap)),
		textShortcut(home, shortcutOpts{Shift: true}, NewExtendSelectionToLineBreakIntent(false, false, wrap)),
		textShortcut(end, shortcutOpts{Shift: true}, NewExtendSelectionToLineBreakIntent(true, false, wrap)),
		textShortcut(home, shortcutOpts{Control: true}, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		textShortcut(end, shortcutOpts{Control: true}, NewExtendSelectionToDocumentBoundaryIntent(true, true)),
		textShortcut(home, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToDocumentBoundaryIntent(false, false)),
		textShortcut(end, shortcutOpts{Shift: true, Control: true}, NewExtendSelectionToDocumentBoundaryIntent(true, false)),
	)
}
