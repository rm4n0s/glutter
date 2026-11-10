package cupertino

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// spellCheckMaxSuggestions is the number of suggestions that the toolbar shows.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart _kMaxSuggestions
const spellCheckMaxSuggestions = 3

// CupertinoSpellCheckSuggestionsToolbar is the toolbar of iOS that shows the spell check suggestions for a
// misspelled word.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar
type CupertinoSpellCheckSuggestionsToolbar struct {
	widgets.StatelessWidget
	anchors     widgets.TextSelectionToolbarAnchors
	buttonItems []widgets.ContextMenuButtonItem
}

// NewCupertinoSpellCheckSuggestionsToolbar creates the toolbar with a button for each item. In debug builds it
// panics with the tag CupertinoSpellCheckSuggestionsToolbarTooManyButtons when there are more than three items.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.new
func NewCupertinoSpellCheckSuggestionsToolbar(anchors widgets.TextSelectionToolbarAnchors, buttonItems []widgets.ContextMenuButtonItem, opts ...widgets.WidgetOpts) *CupertinoSpellCheckSuggestionsToolbar {
	if foundation.DebugMode && len(buttonItems) > spellCheckMaxSuggestions {
		panic(errors.New(tagCupertinoSpellCheckSuggestionsToolbarTooManyButtons, "a CupertinoSpellCheckSuggestionsToolbar has at most three suggestions",
			"buttonItems", len(buttonItems)))
	}
	return &CupertinoSpellCheckSuggestionsToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOf(opts)), anchors: anchors, buttonItems: buttonItems}
}

// NewCupertinoSpellCheckSuggestionsToolbarEditableText creates a toolbar for the misspelled word at the cursor of
// the field: the buttons of its suggestions, at the anchors of its context menu.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.editableText
func NewCupertinoSpellCheckSuggestionsToolbarEditableText(editableTextState *widgets.EditableTextState, opts ...widgets.WidgetOpts) *CupertinoSpellCheckSuggestionsToolbar {
	buttonItems, _ := CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(editableTextState)
	return NewCupertinoSpellCheckSuggestionsToolbar(editableTextState.ContextMenuAnchors(), buttonItems, opts...)
}

// Anchors are where the toolbar is placed.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.anchors
func (w *CupertinoSpellCheckSuggestionsToolbar) Anchors() widgets.TextSelectionToolbarAnchors {
	return w.anchors
}

// ButtonItems are the items of the buttons.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.buttonItems
func (w *CupertinoSpellCheckSuggestionsToolbar) ButtonItems() []widgets.ContextMenuButtonItem {
	return w.buttonItems
}

// CupertinoSpellCheckSuggestionsToolbarBuildButtonItems builds the items of the buttons for the misspelled word at
// the cursor of the field: a button for each of the first three suggestions, which replaces the word, or one
// disabled button that says there are none. It says false, with no items, when the cursor is not in a misspelled
// word.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.buildButtonItems
func CupertinoSpellCheckSuggestionsToolbarBuildButtonItems(editableTextState *widgets.EditableTextState) ([]widgets.ContextMenuButtonItem, bool) {
	spanAtCursorIndex, ok := editableTextState.FindSuggestionSpanAtCursorIndex(editableTextState.TextEditingValue().Selection.BaseOffset)
	if !ok {
		return nil, false
	}
	if len(spanAtCursorIndex.Suggestions) == 0 {
		DebugCheckHasCupertinoLocalizations(editableTextState.Context())
		localizations := CupertinoLocalizationsOf(editableTextState.Context())
		return []widgets.ContextMenuButtonItem{widgets.NewContextMenuButtonItem(nil, widgets.ContextMenuButtonItemOpts{Label: foundation.Some(localizations.NoSpellCheckReplacementsLabel())})}, true
	}
	var buttonItems []widgets.ContextMenuButtonItem
	for i, suggestion := range spanAtCursorIndex.Suggestions {
		if i >= spellCheckMaxSuggestions {
			break
		}
		buttonItems = append(buttonItems, widgets.NewContextMenuButtonItem(func() {
			if !editableTextState.Mounted() {
				return
			}
			cupertinoSpellCheckReplaceText(editableTextState, suggestion, spanAtCursorIndex.Range)
		}, widgets.ContextMenuButtonItemOpts{Label: foundation.Some(suggestion)}))
	}
	return buttonItems, true
}

// cupertinoSpellCheckReplaceText replaces the range of the field's text, puts the cursor after the replacement and
// hides the toolbar. The field must be editable and not obscured (the panic
// CupertinoSpellCheckSuggestionsToolbarReplaceNotAllowed in debug builds).
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar._replaceText
func cupertinoSpellCheckReplaceText(editableTextState *widgets.EditableTextState, text string, replacementRange ui.TextRange) {
	if foundation.DebugMode && (editableTextState.Widget().ReadOnly() || editableTextState.Widget().ObscureText()) {
		panic(errors.New(tagCupertinoSpellCheckSuggestionsToolbarReplaceNotAllowed, "the text of a read-only or obscured field cannot be replaced"))
	}
	newValue := editableTextState.TextEditingValue().Replaced(replacementRange, text)
	editableTextState.UserUpdateTextEditingValue(newValue, services.SelectionChangedCauseToolbar)
	widgets.WidgetsBindingOf(editableTextState.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if editableTextState.Mounted() {
			editableTextState.BringIntoView(editableTextState.TextEditingValue().Selection.Extent())
		}
	})
	editableTextState.HideToolbar(true)
}

// Build builds the toolbar, or nothing when there is no button.
//
// Flutter: cupertino/spell_check_suggestions_toolbar.dart CupertinoSpellCheckSuggestionsToolbar.build
func (w *CupertinoSpellCheckSuggestionsToolbar) Build(widgets.BuildContext) widgets.Widget {
	if len(w.buttonItems) == 0 {
		return widgets.NewSizedBoxShrink(nil)
	}
	children := make([]widgets.Widget, len(w.buttonItems))
	for i, item := range w.buttonItems {
		children[i] = NewCupertinoTextSelectionToolbarButtonButtonItem(item)
	}
	return NewCupertinoTextSelectionToolbar(w.anchors.PrimaryAnchor, w.anchors.SecondaryAnchor.OrElse(w.anchors.PrimaryAnchor), children)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSpellCheckSuggestionsToolbarTooManyButtons    = "CupertinoSpellCheckSuggestionsToolbarTooManyButtons"
	tagCupertinoSpellCheckSuggestionsToolbarReplaceNotAllowed = "CupertinoSpellCheckSuggestionsToolbarReplaceNotAllowed"
)
