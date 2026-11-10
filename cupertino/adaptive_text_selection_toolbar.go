package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoAdaptiveTextSelectionToolbar is a text selection toolbar that fits the platform: the one of iOS (with
// paging) on iOS, Android and Fuchsia, and the one of macOS on macOS, Linux and Windows, with a button for each of
// the items or the given children, at the anchors.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar
type CupertinoAdaptiveTextSelectionToolbar struct {
	widgets.StatelessWidget
	anchors     widgets.TextSelectionToolbarAnchors
	children    []widgets.Widget
	buttonItems []widgets.ContextMenuButtonItem
}

// NewCupertinoAdaptiveTextSelectionToolbar creates a toolbar with the given children, which are the buttons.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.new
func NewCupertinoAdaptiveTextSelectionToolbar(anchors widgets.TextSelectionToolbarAnchors, children []widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoAdaptiveTextSelectionToolbar {
	return &CupertinoAdaptiveTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOf(opts)), anchors: anchors, children: children}
}

// NewCupertinoAdaptiveTextSelectionToolbarButtonItems creates a toolbar with the buttons of the given items.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.buttonItems
func NewCupertinoAdaptiveTextSelectionToolbarButtonItems(buttonItems []widgets.ContextMenuButtonItem, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *CupertinoAdaptiveTextSelectionToolbar {
	return &CupertinoAdaptiveTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(firstOf(opts)), anchors: anchors, buttonItems: buttonItems}
}

// NewCupertinoAdaptiveTextSelectionToolbarEditable creates a toolbar with the default buttons of an editable field.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.editable
func NewCupertinoAdaptiveTextSelectionToolbarEditable(clipboardStatus widgets.ClipboardStatus, callbacks widgets.EditableButtonItemsCallbacks, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *CupertinoAdaptiveTextSelectionToolbar {
	return NewCupertinoAdaptiveTextSelectionToolbarButtonItems(widgets.GetEditableButtonItems(clipboardStatus, callbacks), anchors, opts...)
}

// NewCupertinoAdaptiveTextSelectionToolbarEditableText creates a toolbar with the default buttons and anchors of an
// EditableText.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.editableText
func NewCupertinoAdaptiveTextSelectionToolbarEditableText(editableTextState *widgets.EditableTextState, opts ...widgets.WidgetOpts) *CupertinoAdaptiveTextSelectionToolbar {
	return NewCupertinoAdaptiveTextSelectionToolbarButtonItems(editableTextState.ContextMenuButtonItems(), editableTextState.ContextMenuAnchors(), opts...)
}

// NewCupertinoAdaptiveTextSelectionToolbarSelectable creates a toolbar with the default buttons of a selectable text.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.selectable
func NewCupertinoAdaptiveTextSelectionToolbarSelectable(callbacks widgets.SelectableButtonItemsCallbacks, selectionGeometry rendering.SelectionGeometry, anchors widgets.TextSelectionToolbarAnchors, opts ...widgets.WidgetOpts) *CupertinoAdaptiveTextSelectionToolbar {
	return NewCupertinoAdaptiveTextSelectionToolbarButtonItems(widgets.GetSelectableButtonItems(selectionGeometry, callbacks), anchors, opts...)
}

// Anchors are where the toolbar is placed.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.anchors
func (w *CupertinoAdaptiveTextSelectionToolbar) Anchors() widgets.TextSelectionToolbarAnchors {
	return w.anchors
}

// Children are the buttons, when they are given as widgets.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.children
func (w *CupertinoAdaptiveTextSelectionToolbar) Children() []widgets.Widget { return w.children }

// ButtonItems are the items that the buttons are made from; none when the toolbar has children.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.buttonItems
func (w *CupertinoAdaptiveTextSelectionToolbar) ButtonItems() []widgets.ContextMenuButtonItem {
	return w.buttonItems
}

// CupertinoAdaptiveTextSelectionToolbarGetAdaptiveButtons returns the buttons of the items for the platform: the
// buttons of the iOS toolbar on iOS, Android and Fuchsia, and the ones of the desktop toolbar elsewhere.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.getAdaptiveButtons
func CupertinoAdaptiveTextSelectionToolbarGetAdaptiveButtons(_ widgets.BuildContext, buttonItems []widgets.ContextMenuButtonItem) []widgets.Widget {
	buttons := make([]widgets.Widget, 0, len(buttonItems))
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
		for _, item := range buttonItems {
			buttons = append(buttons, NewCupertinoTextSelectionToolbarButtonButtonItem(item))
		}
	default:
		for _, item := range buttonItems {
			buttons = append(buttons, NewCupertinoDesktopTextSelectionToolbarButtonButtonItem(item))
		}
	}
	return buttons
}

// Build builds the toolbar of the platform, or nothing when there is no button.
//
// Flutter: cupertino/adaptive_text_selection_toolbar.dart CupertinoAdaptiveTextSelectionToolbar.build
func (w *CupertinoAdaptiveTextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	children := w.children
	if children == nil {
		children = CupertinoAdaptiveTextSelectionToolbarGetAdaptiveButtons(bc, w.buttonItems)
	}
	if len(children) == 0 {
		return widgets.NewSizedBoxShrink(nil)
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia:
		return NewCupertinoTextSelectionToolbar(w.anchors.PrimaryAnchor, w.anchors.SecondaryAnchor.OrElse(w.anchors.PrimaryAnchor), children)
	}
	return NewCupertinoDesktopTextSelectionToolbar(w.anchors.PrimaryAnchor, children)
}
