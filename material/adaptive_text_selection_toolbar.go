// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"strings"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/widgets"
)

// AdaptiveTextSelectionToolbar is a text selection toolbar that fits the
// platform: it has a button for each of the items, or the given children, at the
// anchors.
//
// Deviation: Glutter has no Cupertino library and no Material toolbar of the
// mobile platforms (TextSelectionToolbar with its overflow menu), so every
// platform gets the toolbar of the desktop (SPEC §12.10).
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar
type AdaptiveTextSelectionToolbar struct {
	widgets.StatelessWidget
	buttonItems []widgets.ContextMenuButtonItem
	children    []widgets.Widget
	anchors     widgets.TextSelectionToolbarAnchors
}

// NewAdaptiveTextSelectionToolbar creates a toolbar with the given children,
// which are the buttons.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.new
func NewAdaptiveTextSelectionToolbar(children []widgets.Widget, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return &AdaptiveTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOpts(opts)), children: children, anchors: anchors}
}

// NewAdaptiveTextSelectionToolbarButtonItems creates a toolbar with the buttons
// of the given items.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.buttonItems
func NewAdaptiveTextSelectionToolbarButtonItems(buttonItems []widgets.ContextMenuButtonItem, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return &AdaptiveTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOpts(opts)), buttonItems: buttonItems, anchors: anchors}
}

// NewAdaptiveTextSelectionToolbarEditable creates a toolbar with the default
// buttons of an editable field.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.editable
func NewAdaptiveTextSelectionToolbarEditable(clipboardStatus widgets.ClipboardStatus, callbacks widgets.EditableButtonItemsCallbacks, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return NewAdaptiveTextSelectionToolbarButtonItems(widgets.GetEditableButtonItems(clipboardStatus, callbacks), anchors, opts...)
}

// NewAdaptiveTextSelectionToolbarEditableText creates a toolbar with the default
// buttons and anchors of an EditableText.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.editableText
func NewAdaptiveTextSelectionToolbarEditableText(editableTextState *widgets.EditableTextState, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return NewAdaptiveTextSelectionToolbarButtonItems(editableTextState.ContextMenuButtonItems(), editableTextState.ContextMenuAnchors(), opts...)
}

// NewAdaptiveTextSelectionToolbarSelectable creates a toolbar with the default
// buttons of a selectable text.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.selectable
func NewAdaptiveTextSelectionToolbarSelectable(callbacks widgets.SelectableButtonItemsCallbacks, selectionGeometry rendering.SelectionGeometry, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return NewAdaptiveTextSelectionToolbarButtonItems(widgets.GetSelectableButtonItems(selectionGeometry, callbacks), anchors, opts...)
}

// NewAdaptiveTextSelectionToolbarSelectableRegion creates a toolbar with the
// default buttons and anchors of a SelectableRegion.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.selectableRegion
func NewAdaptiveTextSelectionToolbarSelectableRegion(selectableRegionState *widgets.SelectableRegionState, opts ...widgets.WidgetOpts) *AdaptiveTextSelectionToolbar {
	return NewAdaptiveTextSelectionToolbarButtonItems(selectableRegionState.ContextMenuButtonItems(), selectableRegionState.ContextMenuAnchors(), opts...)
}

// ButtonItems are the items that the buttons are made from; none when the
// toolbar has children.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.buttonItems
func (w *AdaptiveTextSelectionToolbar) ButtonItems() []widgets.ContextMenuButtonItem {
	return w.buttonItems
}

// Children are the buttons, when they are given as widgets.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.children
func (w *AdaptiveTextSelectionToolbar) Children() []widgets.Widget { return w.children }

// Anchors are where the toolbar is placed.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.anchors
func (w *AdaptiveTextSelectionToolbar) Anchors() widgets.TextSelectionToolbarAnchors {
	return w.anchors
}

// AdaptiveTextSelectionToolbarGetButtonLabel returns the label of the button of
// an item: its own, or the one of its type in the localizations.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.getButtonLabel
func AdaptiveTextSelectionToolbarGetButtonLabel(bc widgets.BuildContext, buttonItem widgets.ContextMenuButtonItem) string {
	if label, ok := buttonItem.Label.Get(); ok {
		return label
	}
	DebugCheckHasMaterialLocalizations(bc)
	l := MaterialLocalizationsOf(bc)
	switch buttonItem.Type {
	case widgets.ContextMenuButtonTypeCut:
		return l.CutButtonLabel()
	case widgets.ContextMenuButtonTypeCopy:
		return l.CopyButtonLabel()
	case widgets.ContextMenuButtonTypePaste:
		return l.PasteButtonLabel()
	case widgets.ContextMenuButtonTypeSelectAll:
		return l.SelectAllButtonLabel()
	case widgets.ContextMenuButtonTypeDelete:
		return strings.ToUpper(l.DeleteButtonTooltip())
	case widgets.ContextMenuButtonTypeLookUp:
		return l.LookUpButtonLabel()
	case widgets.ContextMenuButtonTypeSearchWeb:
		return l.SearchWebButtonLabel()
	case widgets.ContextMenuButtonTypeShare:
		return l.ShareButtonLabel()
	case widgets.ContextMenuButtonTypeLiveTextInput:
		return l.ScanTextButtonLabel()
	}
	return ""
}

// AdaptiveTextSelectionToolbarGetAdaptiveButtons returns the buttons of the
// items for the platform.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.getAdaptiveButtons
func AdaptiveTextSelectionToolbarGetAdaptiveButtons(bc widgets.BuildContext, buttonItems []widgets.ContextMenuButtonItem) []widgets.Widget {
	buttons := make([]widgets.Widget, 0, len(buttonItems))
	for _, item := range buttonItems {
		buttons = append(buttons, NewDesktopTextSelectionToolbarButtonText(bc, item.OnPressed, AdaptiveTextSelectionToolbarGetButtonLabel(bc, item)))
	}
	return buttons
}

// Build builds the toolbar of the platform, or nothing when there is no button.
//
// Flutter: material/adaptive_text_selection_toolbar.dart AdaptiveTextSelectionToolbar.build
func (w *AdaptiveTextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	children := w.children
	if children == nil {
		children = AdaptiveTextSelectionToolbarGetAdaptiveButtons(bc, w.buttonItems)
	}
	if len(children) == 0 {
		return widgets.NewSizedBoxShrink(nil)
	}
	return NewDesktopTextSelectionToolbar(w.anchors.PrimaryAnchor, children)
}
