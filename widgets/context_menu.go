package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// ContextMenuButtonType is a button that can appear in a context menu by
// default.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType
type ContextMenuButtonType int

// The values of ContextMenuButtonType follow Flutter's declaration order, after
// the zero value, which is the default (a custom button).
const (
	// ContextMenuButtonTypeUnspecified is the zero value: the default, a custom
	// button.
	//
	// Flutter: no counterpart
	ContextMenuButtonTypeUnspecified ContextMenuButtonType = iota
	// ContextMenuButtonTypeCut is a button that cuts the current text selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.cut
	ContextMenuButtonTypeCut
	// ContextMenuButtonTypeCopy is a button that copies the current text selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.copy
	ContextMenuButtonTypeCopy
	// ContextMenuButtonTypePaste is a button that pastes the clipboard contents
	// into the focused text field.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.paste
	ContextMenuButtonTypePaste
	// ContextMenuButtonTypeSelectAll is a button that selects all the contents of
	// the focused text field.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.selectAll
	ContextMenuButtonTypeSelectAll
	// ContextMenuButtonTypeDelete is a button that deletes the current text
	// selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.delete
	ContextMenuButtonTypeDelete
	// ContextMenuButtonTypeLookUp is a button that looks up the current text
	// selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.lookUp
	ContextMenuButtonTypeLookUp
	// ContextMenuButtonTypeSearchWeb is a button that launches a web search for
	// the current text selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.searchWeb
	ContextMenuButtonTypeSearchWeb
	// ContextMenuButtonTypeShare is a button that displays the share screen for
	// the current text selection.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.share
	ContextMenuButtonTypeShare
	// ContextMenuButtonTypeLiveTextInput is a button for starting Live Text input.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.liveTextInput
	ContextMenuButtonTypeLiveTextInput
	// ContextMenuButtonTypeCustom is anything other than the default button
	// types.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType.custom
	ContextMenuButtonTypeCustom
)

var contextMenuButtonTypeNames = []string{"custom", "cut", "copy", "paste", "selectAll", "delete", "lookUp", "searchWeb", "share", "liveTextInput", "custom"}

// Name is the name of the button type, as Dart's enum name.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType
func (t ContextMenuButtonType) Name() string {
	if t < 0 || int(t) >= len(contextMenuButtonTypeNames) {
		return "custom"
	}
	return contextMenuButtonTypeNames[t]
}

// String is the name with the type, as Dart prints an enum value.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonType
func (t ContextMenuButtonType) String() string { return "ContextMenuButtonType." + t.Name() }

// ContextMenuButtonItem is the type and callback for a context menu button.
//
// Deviation: Dart compares the callbacks with ==; Go cannot compare functions,
// so Equal compares the type and the label and whether there is a callback.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem
type ContextMenuButtonItem struct {
	// OnPressed is the callback to be called when the button is pressed; nil
	// disables the button.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.onPressed
	OnPressed foundation.VoidCallback
	// Type is the type of button this represents; the zero value is a custom one.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.type
	Type ContextMenuButtonType
	// Label is the label to display on the button. If Type is not custom and the
	// label is not provided, the default label for that type for the platform is
	// looked up.
	//
	// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.label
	Label foundation.Opt[string]
}

// NewContextMenuButtonItem creates the item of a button.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.new
func NewContextMenuButtonItem(onPressed foundation.VoidCallback, opts ...ContextMenuButtonItemOpts) ContextMenuButtonItem {
	o := firstOpts(opts)
	return ContextMenuButtonItem{OnPressed: onPressed, Type: o.Type, Label: o.Label}
}

// ContextMenuButtonItemOpts are the optional parameters of NewContextMenuButtonItem.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.new
type ContextMenuButtonItemOpts struct {
	// Type is the type of button; the default is custom.
	Type ContextMenuButtonType
	// Label is the label to display on the button.
	Label foundation.Opt[string]
}

// ContextMenuButtonItemCopy are the values CopyWith replaces.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.copyWith
type ContextMenuButtonItemCopy struct {
	OnPressed foundation.VoidCallback
	Type      ContextMenuButtonType
	Label     foundation.Opt[string]
}

// CopyWith creates a new item with the provided parameters overridden.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.copyWith
func (i ContextMenuButtonItem) CopyWith(c ContextMenuButtonItemCopy) ContextMenuButtonItem {
	if c.OnPressed != nil {
		i.OnPressed = c.OnPressed
	}
	if c.Type != ContextMenuButtonTypeUnspecified {
		i.Type = c.Type
	}
	if c.Label.IsSome() {
		i.Label = c.Label
	}
	return i
}

// Equal reports whether the items have the same type and label and both have a
// callback or both have none.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.==
func (i ContextMenuButtonItem) Equal(o ContextMenuButtonItem) bool {
	return i.Type == o.Type && i.Label == o.Label && (i.OnPressed == nil) == (o.OnPressed == nil)
}

// String describes the item.
//
// Flutter: widgets/context_menu_button_item.dart ContextMenuButtonItem.toString
func (i ContextMenuButtonItem) String() string {
	label := "null"
	if l, ok := i.Label.Get(); ok {
		label = l
	}
	return "ContextMenuButtonItem " + i.Type.String() + ", " + label
}

// contextMenuState is the context menu that is shown in the app. There is one
// at a time, and the binding owns it.
type contextMenuState struct {
	shown   *ContextMenuController
	builder WidgetBuilder
	entry   *OverlayEntry
}

// ContextMenuController builds and manages a context menu at a given location.
//
// There can only ever be one context menu shown at a given time in the entire
// app. Calling Show on one instance of this class will hide any other shown
// instances.
//
// Deviation: Dart keeps the shown menu in static fields; here the binding of the
// context that shows it keeps it, and the static removeAny is
// ContextMenuControllerRemoveAny, which takes a context (SPEC §12.10).
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController
type ContextMenuController struct {
	onRemove foundation.VoidCallback
	binding  *WidgetsBinding
}

// ContextMenuControllerOpts are the optional parameters of NewContextMenuController.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.new
type ContextMenuControllerOpts struct {
	// OnRemove is called when this menu is removed.
	OnRemove foundation.VoidCallback
}

// NewContextMenuController creates a context menu that can be shown with Show.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.new
func NewContextMenuController(opts ...ContextMenuControllerOpts) *ContextMenuController {
	return &ContextMenuController{onRemove: firstOpts(opts).OnRemove}
}

// ContextMenuShowOpts are the optional parameters of Show.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.show
type ContextMenuShowOpts struct {
	// DebugRequiredFor is the widget that requires the overlay, named in the
	// error when there is none.
	DebugRequiredFor Widget
}

// Show shows the given context menu. Since there can only be one shown context
// menu at a time, calling this will also remove any other context menu that is
// visible.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.show
func (c *ContextMenuController) Show(bc BuildContext, contextMenuBuilder WidgetBuilder, opts ...ContextMenuShowOpts) {
	binding := WidgetsBindingOf(bc)
	state := &binding.contextMenu
	if c.IsShown() {
		// Update the currently-shown menu in-place by swapping the builder and
		// rebuilding the existing overlay entry.
		state.builder = contextMenuBuilder
		if state.entry != nil {
			state.entry.MarkNeedsBuild()
		}
		return
	}

	contextMenuRemoveAny(binding)
	overlayState := OverlayOf(bc, OverlayOfOpts{RootOverlay: true, DebugRequiredFor: firstOpts(opts).DebugRequiredFor})
	state.builder = contextMenuBuilder
	c.binding = binding
	state.entry = NewOverlayEntry(func(bc BuildContext) Widget {
		var navigatorContext BuildContext
		if n := NavigatorMaybeOf(bc); n != nil {
			navigatorContext = n.Context()
		}
		capturedThemes := CaptureThemes(bc, navigatorContext)
		return capturedThemes.Wrap(state.builder(bc))
	})
	state.shown = c
	overlayState.Insert(state.entry)
}

// ContextMenuControllerRemoveAny removes the currently shown context menu from
// the UI. It does nothing if no context menu is currently shown.
//
// If a menu is removed, and that menu provided an OnRemove callback when it was
// created, then that callback will be called.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.removeAny
func ContextMenuControllerRemoveAny(bc BuildContext) {
	contextMenuRemoveAny(WidgetsBindingOf(bc))
}

func contextMenuRemoveAny(binding *WidgetsBinding) {
	state := &binding.contextMenu
	if state.entry != nil {
		state.entry.Remove()
		state.entry.Dispose()
		state.entry = nil
	}
	state.builder = nil
	if state.shown != nil {
		shown := state.shown
		state.shown = nil
		if shown.onRemove != nil {
			shown.onRemove()
		}
	}
}

// IsShown reports whether this menu is currently being shown.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.isShown
func (c *ContextMenuController) IsShown() bool {
	return c.binding != nil && c.binding.contextMenu.shown == c
}

// MarkNeedsBuild causes the underlying OverlayEntry to rebuild during the next
// pipeline flush. It's necessary to call this function if the output of the
// builder has changed. It is an error if the context menu is not currently
// shown (the tag ContextMenuNotShown in debug builds).
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.markNeedsBuild
func (c *ContextMenuController) MarkNeedsBuild() {
	if foundation.DebugMode && !c.IsShown() {
		panic(errors.New(tagContextMenuNotShown, "the context menu must be shown to be marked as needing a build"))
	}
	if c.binding != nil && c.binding.contextMenu.entry != nil {
		c.binding.contextMenu.entry.MarkNeedsBuild()
	}
}

// Remove removes this menu from the UI. It does nothing if this instance is not
// currently shown: if another context menu is shown, that menu is not removed.
//
// If an OnRemove callback was given to this instance, it will be called.
//
// Flutter: widgets/context_menu_controller.dart ContextMenuController.remove
func (c *ContextMenuController) Remove() {
	if !c.IsShown() {
		return
	}
	contextMenuRemoveAny(c.binding)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagContextMenuNotShown = "ContextMenuNotShown"
)
