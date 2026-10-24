// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// spellCheckSuggestionsToolbarDefaultHeight is the default height of the
	// SpellCheckSuggestionsToolbar, which assumes there are the maximum number of
	// spell check suggestions available, 3. The size was eyeballed on a Pixel 4
	// emulator running Android API 31.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart _kDefaultToolbarHeight
	spellCheckSuggestionsToolbarDefaultHeight = 193.0
	// spellCheckSuggestionsToolbarMaxSuggestions is the number of suggestions that
	// the toolbar shows.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart _kMaxSuggestions
	spellCheckSuggestionsToolbarMaxSuggestions = 3
	// spellCheckSuggestionsToolbarButtonHeight is the height of a button, which the
	// toolbar takes off its default height for each suggestion that is missing.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.build
	spellCheckSuggestionsToolbarButtonHeight = 48.0
	// spellCheckSuggestionsToolbarWidth is the width of the toolbar. It was
	// eyeballed on a Pixel 4 emulator running Android API 31.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart _SpellCheckSuggestionsToolbarContainer.build
	spellCheckSuggestionsToolbarWidth = 165.0
	// spellCheckSuggestionsToolbarElevation is the elevation of the toolbar. It was
	// eyeballed on a Pixel 4 emulator running Android API 31.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart _SpellCheckSuggestionsToolbarContainer.build
	spellCheckSuggestionsToolbarElevation = 2.0
	// spellCheckSuggestionsToolbarSizeDuration is the duration of the change of the
	// size of the toolbar. It was eyeballed on a Pixel 2 emulator running Android
	// API 28 for the Material TextSelectionToolbar.
	//
	// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.build
	spellCheckSuggestionsToolbarSizeDuration = 140 * time.Millisecond
)

// SpellCheckSuggestionsToolbar is the Material-style toolbar that shows the
// spell check suggestions for a misspelled word, and a delete button, as the
// Android toolbar of the system does.
//
// Deviation: it is also the toolbar of iOS and macOS, as there is no Cupertino
// library (SPEC §12.10).
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar
type SpellCheckSuggestionsToolbar struct {
	widgets.StatelessWidget
	anchor      ui.Offset
	buttonItems []widgets.ContextMenuButtonItem
}

// NewSpellCheckSuggestionsToolbar creates the toolbar below the anchor, with a
// button for each item. In debug builds it panics with the tag
// SpellCheckSuggestionsToolbarTooManyButtons when there are more than four items
// (three suggestions and the delete button).
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.new
func NewSpellCheckSuggestionsToolbar(anchor ui.Offset, buttonItems []widgets.ContextMenuButtonItem, opts ...widgets.WidgetOpts) *SpellCheckSuggestionsToolbar {
	if foundation.DebugMode && len(buttonItems) > spellCheckSuggestionsToolbarMaxSuggestions+1 {
		panic(errors.New(tagSpellCheckSuggestionsToolbarTooManyButtons, "a SpellCheckSuggestionsToolbar has at most three suggestions and a delete button",
			"buttonItems", len(buttonItems)))
	}
	return &SpellCheckSuggestionsToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOpts(opts)), anchor: anchor, buttonItems: buttonItems}
}

// NewSpellCheckSuggestionsToolbarEditableText creates a toolbar for the misspelled
// word at the cursor of the field: the buttons of its suggestions and the delete
// button, below the secondary anchor of its context menu (or the primary one).
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.editableText
func NewSpellCheckSuggestionsToolbarEditableText(editableTextState *widgets.EditableTextState, opts ...widgets.WidgetOpts) *SpellCheckSuggestionsToolbar {
	buttonItems, _ := SpellCheckSuggestionsToolbarBuildButtonItems(editableTextState)
	return NewSpellCheckSuggestionsToolbar(SpellCheckSuggestionsToolbarGetToolbarAnchor(editableTextState.ContextMenuAnchors()), buttonItems, opts...)
}

// Anchor is the point that the toolbar is placed below.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.anchor
func (w *SpellCheckSuggestionsToolbar) Anchor() ui.Offset { return w.anchor }

// ButtonItems are the items of the buttons.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.buttonItems
func (w *SpellCheckSuggestionsToolbar) ButtonItems() []widgets.ContextMenuButtonItem {
	return w.buttonItems
}

// SpellCheckSuggestionsToolbarBuildButtonItems builds the items of the buttons for
// the misspelled word at the cursor of the field: a button for each of the first
// three suggestions, which replaces the word, and the delete button. It says false,
// with no items, when the cursor is not in a misspelled word.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.buildButtonItems
func SpellCheckSuggestionsToolbarBuildButtonItems(editableTextState *widgets.EditableTextState) ([]widgets.ContextMenuButtonItem, bool) {
	// Determine if composing region is misspelled.
	spanAtCursorIndex, ok := editableTextState.FindSuggestionSpanAtCursorIndex(editableTextState.TextEditingValue().Selection.BaseOffset)
	if !ok {
		return nil, false
	}

	var buttonItems []widgets.ContextMenuButtonItem

	// Build suggestion buttons.
	for i, suggestion := range spanAtCursorIndex.Suggestions {
		if i >= spellCheckSuggestionsToolbarMaxSuggestions {
			break
		}
		buttonItems = append(buttonItems, widgets.NewContextMenuButtonItem(func() {
			if !editableTextState.Mounted() {
				return
			}
			spellCheckSuggestionsToolbarReplaceText(editableTextState, suggestion, spanAtCursorIndex.Range)
		}, widgets.ContextMenuButtonItemOpts{Label: foundation.Some(suggestion)}))
	}

	// Build delete button.
	buttonItems = append(buttonItems, widgets.NewContextMenuButtonItem(func() {
		if !editableTextState.Mounted() {
			return
		}
		value, _ := editableTextState.CurrentTextEditingValue()
		spellCheckSuggestionsToolbarReplaceText(editableTextState, "", value.Composing)
	}, widgets.ContextMenuButtonItemOpts{Type: widgets.ContextMenuButtonTypeDelete}))

	return buttonItems, true
}

// spellCheckSuggestionsToolbarReplaceText replaces the range of the field's text,
// puts the cursor after the replacement and hides the toolbar. The field must be
// editable and not obscured (the panic SpellCheckSuggestionsToolbarReplaceNotAllowed
// in debug builds).
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar._replaceText
func spellCheckSuggestionsToolbarReplaceText(editableTextState *widgets.EditableTextState, text string, replacementRange ui.TextRange) {
	// Replacement cannot be performed if the text is read only or obscured.
	if foundation.DebugMode && (editableTextState.Widget().ReadOnly() || editableTextState.Widget().ObscureText()) {
		panic(errors.New(tagSpellCheckSuggestionsToolbarReplaceNotAllowed, "the text of a read-only or obscured field cannot be replaced"))
	}

	newValue := editableTextState.TextEditingValue().Replaced(replacementRange, text)
	editableTextState.UserUpdateTextEditingValue(newValue, services.SelectionChangedCauseToolbar)

	// Schedule a call to bringIntoView() after renderEditable updates.
	widgets.WidgetsBindingOf(editableTextState.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if editableTextState.Mounted() {
			editableTextState.BringIntoView(editableTextState.TextEditingValue().Selection.Extent())
		}
	})
	editableTextState.HideToolbar(true)
}

// SpellCheckSuggestionsToolbarGetToolbarAnchor is the anchor the toolbar is
// placed at for the anchors of a context menu: the secondary one, or the primary
// one when there is no secondary one.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.getToolbarAnchor
func SpellCheckSuggestionsToolbarGetToolbarAnchor(anchors widgets.TextSelectionToolbarAnchors) ui.Offset {
	// Since this will be positioned below the anchor point, use the secondary
	// anchor by default.
	return anchors.SecondaryAnchor.OrElse(anchors.PrimaryAnchor)
}

// buildToolbarButtons builds the buttons of the items.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar._buildToolbarButtons
func (w *SpellCheckSuggestionsToolbar) buildToolbarButtons(bc widgets.BuildContext) []widgets.Widget {
	buttons := make([]widgets.Widget, 0, len(w.buttonItems))
	for _, buttonItem := range w.buttonItems {
		var style *painting.TextStyle
		if buttonItem.Type == widgets.ContextMenuButtonTypeDelete {
			style = painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(Colors.Blue.Primary)})
		}
		var button widgets.Widget = NewTextSelectionToolbarTextButton(
			painting.NewEdgeInsetsFromLTRB(20, 0, 0, 0),
			widgets.NewText(AdaptiveTextSelectionToolbarGetButtonLabel(bc, buttonItem), widgets.TextOpts{Style: style}),
			TextSelectionToolbarTextButtonOpts{OnPressed: buttonItem.OnPressed, Alignment: painting.AlignmentCenterLeft})

		if buttonItem.Type != widgets.ContextMenuButtonTypeDelete {
			buttons = append(buttons, button)
			continue
		}
		buttons = append(buttons, widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{
				Border: painting.Border{Top: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(Colors.Grey.Primary)})},
			}), button))
	}
	return buttons
}

// Build puts the buttons in a card below the anchor, kept inside the screen
// padding, the status bar and the soft keyboard. With no items it builds nothing.
//
// Flutter: material/spell_check_suggestions_toolbar.dart SpellCheckSuggestionsToolbar.build
func (w *SpellCheckSuggestionsToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	if len(w.buttonItems) == 0 {
		return widgets.NewSizedBoxShrink(nil)
	}

	// Adjust toolbar height if needed.
	spellCheckSuggestionsToolbarHeight := spellCheckSuggestionsToolbarDefaultHeight -
		(spellCheckSuggestionsToolbarButtonHeight * float64(spellCheckSuggestionsToolbarMaxSuggestions+1-len(w.buttonItems)))
	// Incorporate the padding distance between the content and toolbar.
	mediaQueryData := widgets.MediaQueryOf(bc)
	softKeyboardViewInsetsBottom := mediaQueryData.ViewInsets.Bottom
	paddingAbove := mediaQueryData.Padding.Top + textSelectionToolbarScreenPadding
	// Makes up for the Padding.
	localAdjustment := ui.Offset{Dx: textSelectionToolbarScreenPadding, Dy: paddingAbove}

	return widgets.NewPadding(
		painting.NewEdgeInsetsFromLTRB(textSelectionToolbarScreenPadding, paddingAbove, textSelectionToolbarScreenPadding,
			textSelectionToolbarScreenPadding+softKeyboardViewInsetsBottom),
		widgets.NewCustomSingleChildLayout(
			NewSpellCheckSuggestionsToolbarLayoutDelegate(w.anchor.Sub(localAdjustment)),
			widgets.NewAnimatedSize(spellCheckSuggestionsToolbarSizeDuration,
				newSpellCheckSuggestionsToolbarContainer(spellCheckSuggestionsToolbarHeight, w.buildToolbarButtons(bc)))))
}

// spellCheckSuggestionsToolbarContainer is the card with the buttons.
//
// Flutter: material/spell_check_suggestions_toolbar.dart _SpellCheckSuggestionsToolbarContainer
type spellCheckSuggestionsToolbarContainer struct {
	widgets.StatelessWidget
	height   float64
	children []widgets.Widget
}

func newSpellCheckSuggestionsToolbarContainer(height float64, children []widgets.Widget) *spellCheckSuggestionsToolbarContainer {
	return &spellCheckSuggestionsToolbarContainer{StatelessWidget: widgets.NewStatelessWidgetBase(), height: height, children: children}
}

func (w *spellCheckSuggestionsToolbarContainer) Build(widgets.BuildContext) widgets.Widget {
	return NewMaterial(
		widgets.NewSizedBox(
			widgets.NewColumn(w.children, widgets.ColumnOpts{
				MainAxisSize:       rendering.MainAxisSizeMin,
				CrossAxisAlignment: rendering.CrossAxisAlignmentStretch,
			}),
			widgets.SizedBoxOpts{Width: foundation.Some(spellCheckSuggestionsToolbarWidth), Height: foundation.Some(w.height)}),
		MaterialOpts{Elevation: spellCheckSuggestionsToolbarElevation, Type: MaterialTypeCard})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSpellCheckSuggestionsToolbarReplaceNotAllowed = "SpellCheckSuggestionsToolbarReplaceNotAllowed"
	tagSpellCheckSuggestionsToolbarTooManyButtons    = "SpellCheckSuggestionsToolbarTooManyButtons"
)
