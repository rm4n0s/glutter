package ui

import (
	"strings"

	"github.com/rm4n0s/errors"
)

// SemanticsAction is an action that an assistive technology can ask a node to
// perform. The value of each action is its bit in a bit field of actions.
//
// Flutter: ui/semantics.dart SemanticsAction
type SemanticsAction int

// The semantics actions; each is one bit of the bit field the engine uses.
const (
	// SemanticsActionTap is a request to tap the node.
	//
	// Flutter: ui/semantics.dart SemanticsAction.tap
	SemanticsActionTap SemanticsAction = 1 << iota
	// SemanticsActionLongPress is a request to long press the node.
	//
	// Flutter: ui/semantics.dart SemanticsAction.longPress
	SemanticsActionLongPress
	// SemanticsActionScrollLeft scrolls the content so that it moves left.
	//
	// Flutter: ui/semantics.dart SemanticsAction.scrollLeft
	SemanticsActionScrollLeft
	// SemanticsActionScrollRight scrolls the content so that it moves right.
	//
	// Flutter: ui/semantics.dart SemanticsAction.scrollRight
	SemanticsActionScrollRight
	// SemanticsActionScrollUp scrolls the content so that it moves up.
	//
	// Flutter: ui/semantics.dart SemanticsAction.scrollUp
	SemanticsActionScrollUp
	// SemanticsActionScrollDown scrolls the content so that it moves down.
	//
	// Flutter: ui/semantics.dart SemanticsAction.scrollDown
	SemanticsActionScrollDown
	// SemanticsActionIncrease increases the value of a slider or similar.
	//
	// Flutter: ui/semantics.dart SemanticsAction.increase
	SemanticsActionIncrease
	// SemanticsActionDecrease decreases the value of a slider or similar.
	//
	// Flutter: ui/semantics.dart SemanticsAction.decrease
	SemanticsActionDecrease
	// SemanticsActionShowOnScreen scrolls the node into view.
	//
	// Flutter: ui/semantics.dart SemanticsAction.showOnScreen
	SemanticsActionShowOnScreen
	// SemanticsActionMoveCursorForwardByCharacter moves the cursor one
	// character forward.
	//
	// Flutter: ui/semantics.dart SemanticsAction.moveCursorForwardByCharacter
	SemanticsActionMoveCursorForwardByCharacter
	// SemanticsActionMoveCursorBackwardByCharacter moves the cursor one
	// character backward.
	//
	// Flutter: ui/semantics.dart SemanticsAction.moveCursorBackwardByCharacter
	SemanticsActionMoveCursorBackwardByCharacter
	// SemanticsActionSetSelection sets the text selection.
	//
	// Flutter: ui/semantics.dart SemanticsAction.setSelection
	SemanticsActionSetSelection
	// SemanticsActionCopy copies the selection to the clipboard.
	//
	// Flutter: ui/semantics.dart SemanticsAction.copy
	SemanticsActionCopy
	// SemanticsActionCut cuts the selection to the clipboard.
	//
	// Flutter: ui/semantics.dart SemanticsAction.cut
	SemanticsActionCut
	// SemanticsActionPaste pastes the clipboard.
	//
	// Flutter: ui/semantics.dart SemanticsAction.paste
	SemanticsActionPaste
	// SemanticsActionDidGainAccessibilityFocus says that the node got the
	// accessibility focus.
	//
	// Flutter: ui/semantics.dart SemanticsAction.didGainAccessibilityFocus
	SemanticsActionDidGainAccessibilityFocus
	// SemanticsActionDidLoseAccessibilityFocus says that the node lost the
	// accessibility focus.
	//
	// Flutter: ui/semantics.dart SemanticsAction.didLoseAccessibilityFocus
	SemanticsActionDidLoseAccessibilityFocus
	// SemanticsActionCustomAction is a custom action; the argument is its ID.
	//
	// Flutter: ui/semantics.dart SemanticsAction.customAction
	SemanticsActionCustomAction
	// SemanticsActionDismiss dismisses the node, for example a dialog.
	//
	// Flutter: ui/semantics.dart SemanticsAction.dismiss
	SemanticsActionDismiss
	// SemanticsActionMoveCursorForwardByWord moves the cursor one word forward.
	//
	// Flutter: ui/semantics.dart SemanticsAction.moveCursorForwardByWord
	SemanticsActionMoveCursorForwardByWord
	// SemanticsActionMoveCursorBackwardByWord moves the cursor one word
	// backward.
	//
	// Flutter: ui/semantics.dart SemanticsAction.moveCursorBackwardByWord
	SemanticsActionMoveCursorBackwardByWord
	// SemanticsActionSetText replaces the text of the field.
	//
	// Flutter: ui/semantics.dart SemanticsAction.setText
	SemanticsActionSetText
	// SemanticsActionFocus gives the node the input focus.
	//
	// Flutter: ui/semantics.dart SemanticsAction.focus
	SemanticsActionFocus
	// SemanticsActionScrollToOffset scrolls to an offset; the argument is the
	// offset.
	//
	// Flutter: ui/semantics.dart SemanticsAction.scrollToOffset
	SemanticsActionScrollToOffset
	// SemanticsActionExpand expands a collapsed node.
	//
	// Flutter: ui/semantics.dart SemanticsAction.expand
	SemanticsActionExpand
	// SemanticsActionCollapse collapses an expanded node.
	//
	// Flutter: ui/semantics.dart SemanticsAction.collapse
	SemanticsActionCollapse
)

var semanticsActionNames = []string{
	"tap", "longPress", "scrollLeft", "scrollRight", "scrollUp", "scrollDown", "increase", "decrease", "showOnScreen",
	"moveCursorForwardByCharacter", "moveCursorBackwardByCharacter", "setSelection", "copy", "cut", "paste",
	"didGainAccessibilityFocus", "didLoseAccessibilityFocus", "customAction", "dismiss", "moveCursorForwardByWord",
	"moveCursorBackwardByWord", "setText", "focus", "scrollToOffset", "expand", "collapse",
}

// SemanticsActionValues are all the actions, in the order of their bits.
//
// Flutter: ui/semantics.dart SemanticsAction.values
func SemanticsActionValues() []SemanticsAction {
	out := make([]SemanticsAction, len(semanticsActionNames))
	for i := range out {
		out[i] = 1 << i
	}
	return out
}

// Name returns the name of the action, such as "tap".
//
// Flutter: ui/semantics.dart SemanticsAction.name
func (a SemanticsAction) Name() string {
	for i, n := range semanticsActionNames {
		if a == 1<<i {
			return n
		}
	}
	return "SemanticsAction(" + itoa(int(a)) + ")"
}

// String is "SemanticsAction.tap".
//
// Flutter: ui/semantics.dart SemanticsAction.toString
func (a SemanticsAction) String() string { return "SemanticsAction." + a.Name() }

// Tristate is a boolean that can also be unset.
//
// Flutter: ui/semantics.dart Tristate
type Tristate int

// The values of Tristate. The zero value is TristateNone.
const (
	// TristateNone is the state of a property that does not apply.
	//
	// Flutter: ui/semantics.dart Tristate.none
	TristateNone Tristate = iota
	// TristateIsTrue is true.
	//
	// Flutter: ui/semantics.dart Tristate.isTrue
	TristateIsTrue
	// TristateIsFalse is false.
	//
	// Flutter: ui/semantics.dart Tristate.isFalse
	TristateIsFalse
)

// Name returns the name of the value, such as "isTrue".
//
// Flutter: ui/semantics.dart Tristate.name
func (t Tristate) Name() string {
	return enumName([]string{"none", "isTrue", "isFalse"}, int(t), "Tristate")
}

// String is "Tristate.isTrue".
//
// Flutter: ui/semantics.dart Tristate.toString
func (t Tristate) String() string { return "Tristate." + t.Name() }

// ToBoolOrNull is the value as a pointer to a bool; nil for TristateNone.
//
// Flutter: ui/semantics.dart Tristate.toBoolOrNull
func (t Tristate) ToBoolOrNull() *bool {
	switch t {
	case TristateIsTrue:
		v := true
		return &v
	case TristateIsFalse:
		v := false
		return &v
	}
	return nil
}

// TristateFromBool is TristateIsTrue or TristateIsFalse.
//
// Flutter: semantics/semantics.dart _tristateFromBoolOrNull
func TristateFromBool(v bool) Tristate {
	if v {
		return TristateIsTrue
	}
	return TristateIsFalse
}

// CheckedState is the state of a checkable node.
//
// Flutter: ui/semantics.dart CheckedState
type CheckedState int

// The values of CheckedState. The zero value is CheckedStateNone.
const (
	// CheckedStateNone is the state of a node that is not checkable.
	//
	// Flutter: ui/semantics.dart CheckedState.none
	CheckedStateNone CheckedState = iota
	// CheckedStateIsTrue is checked.
	//
	// Flutter: ui/semantics.dart CheckedState.isTrue
	CheckedStateIsTrue
	// CheckedStateIsFalse is not checked.
	//
	// Flutter: ui/semantics.dart CheckedState.isFalse
	CheckedStateIsFalse
	// CheckedStateMixed is neither checked nor not checked.
	//
	// Flutter: ui/semantics.dart CheckedState.mixed
	CheckedStateMixed
)

// Name returns the name of the value, such as "isTrue".
//
// Flutter: ui/semantics.dart CheckedState.name
func (c CheckedState) Name() string {
	return enumName([]string{"none", "isTrue", "isFalse", "mixed"}, int(c), "CheckedState")
}

// String is "CheckedState.isTrue".
//
// Flutter: ui/semantics.dart CheckedState.toString
func (c CheckedState) String() string { return "CheckedState." + c.Name() }

// SemanticsFlags are the boolean properties of a semantics node. The zero value
// is SemanticsFlagsNone.
//
// Flutter: ui/semantics.dart SemanticsFlags
type SemanticsFlags struct {
	// IsChecked is the checked state of a checkbox or radio button.
	IsChecked CheckedState
	// IsSelected is whether the node is selected.
	IsSelected Tristate
	// IsEnabled is whether the node is enabled.
	IsEnabled Tristate
	// IsToggled is whether a toggle is on.
	IsToggled Tristate
	// IsExpanded is whether an expandable node is expanded.
	IsExpanded Tristate
	// IsRequired is whether a form field is required.
	IsRequired Tristate
	// IsFocused is whether the node has the input focus; none if it cannot.
	IsFocused Tristate
	// IsButton is whether the node is a button.
	IsButton bool
	// IsTextField is whether the node is a text field.
	IsTextField bool
	// IsInMutuallyExclusiveGroup is whether the node is in a radio group.
	IsInMutuallyExclusiveGroup bool
	// IsHeader is whether the node is a header.
	IsHeader bool
	// IsObscured is whether the text is obscured, as in a password field.
	IsObscured bool
	// ScopesRoute is whether the node is the root of a route.
	ScopesRoute bool
	// NamesRoute is whether the node names the route.
	NamesRoute bool
	// IsHidden is whether the node is hidden from the user.
	IsHidden bool
	// IsImage is whether the node is an image.
	IsImage bool
	// IsLiveRegion is whether changes of the node are announced.
	IsLiveRegion bool
	// HasImplicitScrolling is whether the node scrolls when a child gets the
	// accessibility focus.
	HasImplicitScrolling bool
	// IsMultiline is whether the text field has several lines.
	IsMultiline bool
	// IsReadOnly is whether the text field is read only.
	IsReadOnly bool
	// IsLink is whether the node is a link.
	IsLink bool
	// IsSlider is whether the node is a slider.
	IsSlider bool
	// IsKeyboardKey is whether the node is a key of an on screen keyboard.
	IsKeyboardKey bool
	// IsAccessibilityFocusBlocked is whether the node cannot get the
	// accessibility focus.
	IsAccessibilityFocusBlocked bool
}

// SemanticsFlagsNone is the flags of a node with no flag set.
//
// Flutter: ui/semantics.dart SemanticsFlags.none
var SemanticsFlagsNone = SemanticsFlags{}

func mergeTristate(a, b Tristate) Tristate {
	if a == TristateIsTrue || b == TristateIsTrue {
		return TristateIsTrue
	}
	if a == TristateIsFalse || b == TristateIsFalse {
		return TristateIsFalse
	}
	return TristateNone
}

func mergeChecked(a, b CheckedState) CheckedState {
	for _, v := range []CheckedState{CheckedStateMixed, CheckedStateIsTrue, CheckedStateIsFalse} {
		if a == v || b == v {
			return v
		}
	}
	return CheckedStateNone
}

// Merge combines the flags of a node with those of a node merged into it: a
// flag is set if it is set in either, and a state that is true or mixed in
// either stays so.
//
// Flutter: ui/semantics.dart SemanticsFlags.merge
func (f SemanticsFlags) Merge(o SemanticsFlags) SemanticsFlags {
	return SemanticsFlags{
		IsChecked: mergeChecked(f.IsChecked, o.IsChecked), IsSelected: mergeTristate(f.IsSelected, o.IsSelected),
		IsEnabled: mergeTristate(f.IsEnabled, o.IsEnabled), IsToggled: mergeTristate(f.IsToggled, o.IsToggled),
		IsExpanded: mergeTristate(f.IsExpanded, o.IsExpanded), IsRequired: mergeTristate(f.IsRequired, o.IsRequired),
		IsFocused: mergeTristate(f.IsFocused, o.IsFocused), IsButton: f.IsButton || o.IsButton,
		IsTextField: f.IsTextField || o.IsTextField, IsInMutuallyExclusiveGroup: f.IsInMutuallyExclusiveGroup || o.IsInMutuallyExclusiveGroup,
		IsHeader: f.IsHeader || o.IsHeader, IsObscured: f.IsObscured || o.IsObscured, ScopesRoute: f.ScopesRoute || o.ScopesRoute,
		NamesRoute: f.NamesRoute || o.NamesRoute, IsHidden: f.IsHidden || o.IsHidden, IsImage: f.IsImage || o.IsImage,
		IsLiveRegion: f.IsLiveRegion || o.IsLiveRegion, HasImplicitScrolling: f.HasImplicitScrolling || o.HasImplicitScrolling,
		IsMultiline: f.IsMultiline || o.IsMultiline, IsReadOnly: f.IsReadOnly || o.IsReadOnly, IsLink: f.IsLink || o.IsLink,
		IsSlider: f.IsSlider || o.IsSlider, IsKeyboardKey: f.IsKeyboardKey || o.IsKeyboardKey,
		IsAccessibilityFocusBlocked: f.IsAccessibilityFocusBlocked || o.IsAccessibilityFocusBlocked,
	}
}

// HasConflictingFlags reports whether both have the same flag set, or a value
// for the same state; two such nodes cannot be merged into one.
//
// Flutter: ui/semantics.dart SemanticsFlags.hasConflictingFlags
func (f SemanticsFlags) HasConflictingFlags(o SemanticsFlags) bool {
	both := func(a, b Tristate) bool { return a != TristateNone && b != TristateNone }
	return (f.IsChecked != CheckedStateNone && o.IsChecked != CheckedStateNone) ||
		both(f.IsSelected, o.IsSelected) || both(f.IsEnabled, o.IsEnabled) || both(f.IsToggled, o.IsToggled) ||
		both(f.IsExpanded, o.IsExpanded) || both(f.IsRequired, o.IsRequired) || both(f.IsFocused, o.IsFocused) ||
		f.IsButton && o.IsButton || f.IsTextField && o.IsTextField ||
		f.IsInMutuallyExclusiveGroup && o.IsInMutuallyExclusiveGroup || f.IsHeader && o.IsHeader ||
		f.IsObscured && o.IsObscured || f.ScopesRoute && o.ScopesRoute || f.NamesRoute && o.NamesRoute ||
		f.IsHidden && o.IsHidden || f.IsImage && o.IsImage || f.IsLiveRegion && o.IsLiveRegion ||
		f.HasImplicitScrolling && o.HasImplicitScrolling || f.IsMultiline && o.IsMultiline ||
		f.IsReadOnly && o.IsReadOnly || f.IsLink && o.IsLink || f.IsSlider && o.IsSlider ||
		f.IsKeyboardKey && o.IsKeyboardKey || f.IsAccessibilityFocusBlocked && o.IsAccessibilityFocusBlocked
}

// ToStrings names the flags that are set, in the order of the engine's bit
// field (hasCheckedState, isChecked, isSelected, isButton, ...).
//
// Flutter: ui/semantics.dart SemanticsFlags.toStrings
func (f SemanticsFlags) ToStrings() []string {
	var out []string
	add := func(on bool, name string) {
		if on {
			out = append(out, name)
		}
	}
	add(f.IsChecked != CheckedStateNone, "hasCheckedState")
	add(f.IsChecked == CheckedStateIsTrue, "isChecked")
	add(f.IsSelected == TristateIsTrue, "isSelected")
	add(f.IsButton, "isButton")
	add(f.IsTextField, "isTextField")
	add(f.IsFocused == TristateIsTrue, "isFocused")
	add(f.IsEnabled != TristateNone, "hasEnabledState")
	add(f.IsEnabled == TristateIsTrue, "isEnabled")
	add(f.IsInMutuallyExclusiveGroup, "isInMutuallyExclusiveGroup")
	add(f.IsHeader, "isHeader")
	add(f.IsObscured, "isObscured")
	add(f.ScopesRoute, "scopesRoute")
	add(f.NamesRoute, "namesRoute")
	add(f.IsHidden, "isHidden")
	add(f.IsImage, "isImage")
	add(f.IsLiveRegion, "isLiveRegion")
	add(f.IsToggled != TristateNone, "hasToggledState")
	add(f.IsToggled == TristateIsTrue, "isToggled")
	add(f.HasImplicitScrolling, "hasImplicitScrolling")
	add(f.IsMultiline, "isMultiline")
	add(f.IsReadOnly, "isReadOnly")
	add(f.IsFocused != TristateNone, "isFocusable")
	add(f.IsLink, "isLink")
	add(f.IsSlider, "isSlider")
	add(f.IsKeyboardKey, "isKeyboardKey")
	add(f.IsChecked == CheckedStateMixed, "isCheckStateMixed")
	add(f.IsExpanded != TristateNone, "hasExpandedState")
	add(f.IsExpanded == TristateIsTrue, "isExpanded")
	add(f.IsSelected != TristateNone, "hasSelectedState")
	add(f.IsRequired != TristateNone, "hasRequiredState")
	add(f.IsRequired == TristateIsTrue, "isRequired")
	add(f.IsAccessibilityFocusBlocked, "isAccessibilityFocusBlocked")
	return out
}

// String lists the flags that are set.
//
// Flutter: ui/semantics.dart SemanticsFlags.toString
func (f SemanticsFlags) String() string {
	return "SemanticsFlags[" + strings.Join(f.ToStrings(), ", ") + "]"
}

// SemanticsRole is the role of a node, which tells an assistive technology
// what kind of control or landmark it is.
//
// Flutter: ui/semantics.dart SemanticsRole
type SemanticsRole int

// The values of SemanticsRole. The zero value is SemanticsRoleNone.
const (
	// SemanticsRoleNone is no role.
	//
	// Flutter: ui/semantics.dart SemanticsRole.none
	SemanticsRoleNone SemanticsRole = iota
	// SemanticsRoleTab is a tab in a tab bar.
	//
	// Flutter: ui/semantics.dart SemanticsRole.tab
	SemanticsRoleTab
	// SemanticsRoleTabBar is a bar of tabs.
	//
	// Flutter: ui/semantics.dart SemanticsRole.tabBar
	SemanticsRoleTabBar
	// SemanticsRoleTabPanel is the content of a tab.
	//
	// Flutter: ui/semantics.dart SemanticsRole.tabPanel
	SemanticsRoleTabPanel
	// SemanticsRoleDialog is a dialog.
	//
	// Flutter: ui/semantics.dart SemanticsRole.dialog
	SemanticsRoleDialog
	// SemanticsRoleAlertDialog is a dialog that needs an answer.
	//
	// Flutter: ui/semantics.dart SemanticsRole.alertDialog
	SemanticsRoleAlertDialog
	// SemanticsRoleTable is a table.
	//
	// Flutter: ui/semantics.dart SemanticsRole.table
	SemanticsRoleTable
	// SemanticsRoleCell is a cell of a table.
	//
	// Flutter: ui/semantics.dart SemanticsRole.cell
	SemanticsRoleCell
	// SemanticsRoleRow is a row of a table.
	//
	// Flutter: ui/semantics.dart SemanticsRole.row
	SemanticsRoleRow
	// SemanticsRoleColumnHeader is the header of a column of a table.
	//
	// Flutter: ui/semantics.dart SemanticsRole.columnHeader
	SemanticsRoleColumnHeader
	// SemanticsRoleDragHandle is a handle that is dragged.
	//
	// Flutter: ui/semantics.dart SemanticsRole.dragHandle
	SemanticsRoleDragHandle
	// SemanticsRoleSpinButton is a field with a value that steps up and down.
	//
	// Flutter: ui/semantics.dart SemanticsRole.spinButton
	SemanticsRoleSpinButton
	// SemanticsRoleComboBox is a text field with a list of choices.
	//
	// Flutter: ui/semantics.dart SemanticsRole.comboBox
	SemanticsRoleComboBox
	// SemanticsRoleMenuBar is a bar of menus.
	//
	// Flutter: ui/semantics.dart SemanticsRole.menuBar
	SemanticsRoleMenuBar
	// SemanticsRoleMenu is a menu.
	//
	// Flutter: ui/semantics.dart SemanticsRole.menu
	SemanticsRoleMenu
	// SemanticsRoleMenuItem is an item of a menu.
	//
	// Flutter: ui/semantics.dart SemanticsRole.menuItem
	SemanticsRoleMenuItem
	// SemanticsRoleMenuItemCheckbox is a checkbox in a menu.
	//
	// Flutter: ui/semantics.dart SemanticsRole.menuItemCheckbox
	SemanticsRoleMenuItemCheckbox
	// SemanticsRoleMenuItemRadio is a radio button in a menu.
	//
	// Flutter: ui/semantics.dart SemanticsRole.menuItemRadio
	SemanticsRoleMenuItemRadio
	// SemanticsRoleAlert is a message that is announced at once.
	//
	// Flutter: ui/semantics.dart SemanticsRole.alert
	SemanticsRoleAlert
	// SemanticsRoleStatus is a status message.
	//
	// Flutter: ui/semantics.dart SemanticsRole.status
	SemanticsRoleStatus
	// SemanticsRoleList is a list.
	//
	// Flutter: ui/semantics.dart SemanticsRole.list
	SemanticsRoleList
	// SemanticsRoleListItem is an item of a list.
	//
	// Flutter: ui/semantics.dart SemanticsRole.listItem
	SemanticsRoleListItem
	// SemanticsRoleComplementary is a landmark for content next to the main one.
	//
	// Flutter: ui/semantics.dart SemanticsRole.complementary
	SemanticsRoleComplementary
	// SemanticsRoleContentInfo is a landmark for the information about the page.
	//
	// Flutter: ui/semantics.dart SemanticsRole.contentInfo
	SemanticsRoleContentInfo
	// SemanticsRoleMain is the landmark of the main content.
	//
	// Flutter: ui/semantics.dart SemanticsRole.main
	SemanticsRoleMain
	// SemanticsRoleNavigation is a landmark for navigation.
	//
	// Flutter: ui/semantics.dart SemanticsRole.navigation
	SemanticsRoleNavigation
	// SemanticsRoleRegion is a landmark for a region with a label.
	//
	// Flutter: ui/semantics.dart SemanticsRole.region
	SemanticsRoleRegion
	// SemanticsRoleForm is a form.
	//
	// Flutter: ui/semantics.dart SemanticsRole.form
	SemanticsRoleForm
	// SemanticsRoleLoadingSpinner is a progress indicator without a value.
	//
	// Flutter: ui/semantics.dart SemanticsRole.loadingSpinner
	SemanticsRoleLoadingSpinner
	// SemanticsRoleProgressBar is a progress bar.
	//
	// Flutter: ui/semantics.dart SemanticsRole.progressBar
	SemanticsRoleProgressBar
	// SemanticsRoleRadioGroup is a group of radio buttons.
	//
	// Flutter: ui/semantics.dart SemanticsRole.radioGroup
	SemanticsRoleRadioGroup
	// SemanticsRoleTooltip is a tooltip.
	//
	// Flutter: ui/semantics.dart SemanticsRole.tooltip
	SemanticsRoleTooltip
	// SemanticsRoleHotKey is a keyboard shortcut.
	//
	// Flutter: ui/semantics.dart SemanticsRole.hotKey
	SemanticsRoleHotKey
)

var semanticsRoleNames = []string{"none", "tab", "tabBar", "tabPanel", "dialog", "alertDialog", "table", "cell", "row",
	"columnHeader", "dragHandle", "spinButton", "comboBox", "menuBar", "menu", "menuItem", "menuItemCheckbox", "menuItemRadio",
	"alert", "status", "list", "listItem", "complementary", "contentInfo", "main", "navigation", "region", "form",
	"loadingSpinner", "progressBar", "radioGroup", "tooltip", "hotKey"}

// Name returns the name of the role, such as "tabBar".
//
// Flutter: ui/semantics.dart SemanticsRole.name
func (r SemanticsRole) Name() string { return enumName(semanticsRoleNames, int(r), "SemanticsRole") }

// String is "SemanticsRole.tabBar".
//
// Flutter: ui/semantics.dart SemanticsRole.toString
func (r SemanticsRole) String() string { return "SemanticsRole." + r.Name() }

// SemanticsValidationResult is the result of the validation of a field.
//
// Flutter: ui/semantics.dart SemanticsValidationResult
type SemanticsValidationResult int

// The values of SemanticsValidationResult. The zero value is
// SemanticsValidationResultNone.
const (
	// SemanticsValidationResultNone is no validation.
	//
	// Flutter: ui/semantics.dart SemanticsValidationResult.none
	SemanticsValidationResultNone SemanticsValidationResult = iota
	// SemanticsValidationResultValid is a value that passed the validation.
	//
	// Flutter: ui/semantics.dart SemanticsValidationResult.valid
	SemanticsValidationResultValid
	// SemanticsValidationResultInvalid is a value that failed the validation.
	//
	// Flutter: ui/semantics.dart SemanticsValidationResult.invalid
	SemanticsValidationResultInvalid
)

// Name returns the name of the value, such as "valid".
//
// Flutter: ui/semantics.dart SemanticsValidationResult.name
func (v SemanticsValidationResult) Name() string {
	return enumName([]string{"none", "valid", "invalid"}, int(v), "SemanticsValidationResult")
}

// String is "SemanticsValidationResult.valid".
//
// Flutter: ui/semantics.dart SemanticsValidationResult.toString
func (v SemanticsValidationResult) String() string { return "SemanticsValidationResult." + v.Name() }

// SemanticsInputType is the type of input of a text field.
//
// Flutter: ui/semantics.dart SemanticsInputType
type SemanticsInputType int

// The values of SemanticsInputType. The zero value is SemanticsInputTypeNone.
const (
	// SemanticsInputTypeNone is not a text field.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.none
	SemanticsInputTypeNone SemanticsInputType = iota
	// SemanticsInputTypeText is plain text.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.text
	SemanticsInputTypeText
	// SemanticsInputTypeURL is a URL.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.url
	SemanticsInputTypeURL
	// SemanticsInputTypePhone is a phone number.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.phone
	SemanticsInputTypePhone
	// SemanticsInputTypeSearch is a search query.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.search
	SemanticsInputTypeSearch
	// SemanticsInputTypeEmail is an email address.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.email
	SemanticsInputTypeEmail
	// SemanticsInputTypeMultiline is several lines of text.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.multiline
	SemanticsInputTypeMultiline
	// SemanticsInputTypeNumber is a number.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.number
	SemanticsInputTypeNumber
	// SemanticsInputTypeDatetime is a date and time.
	//
	// Flutter: ui/semantics.dart SemanticsInputType.datetime
	SemanticsInputTypeDatetime
)

// Name returns the name of the value, such as "email".
//
// Flutter: ui/semantics.dart SemanticsInputType.name
func (t SemanticsInputType) Name() string {
	return enumName([]string{"none", "text", "url", "phone", "search", "email", "multiline", "number", "datetime"}, int(t), "SemanticsInputType")
}

// String is "SemanticsInputType.email".
//
// Flutter: ui/semantics.dart SemanticsInputType.toString
func (t SemanticsInputType) String() string { return "SemanticsInputType." + t.Name() }

// SemanticsHitTestBehavior says how a node takes part in hit testing for
// assistive technology.
//
// Flutter: ui/semantics.dart SemanticsHitTestBehavior
type SemanticsHitTestBehavior int

// The values of SemanticsHitTestBehavior. The zero value is
// SemanticsHitTestBehaviorDefer.
const (
	// SemanticsHitTestBehaviorDefer leaves the decision to the children.
	//
	// Flutter: ui/semantics.dart SemanticsHitTestBehavior.defer
	SemanticsHitTestBehaviorDefer SemanticsHitTestBehavior = iota
	// SemanticsHitTestBehaviorOpaque is hit, and hides what is behind it.
	//
	// Flutter: ui/semantics.dart SemanticsHitTestBehavior.opaque
	SemanticsHitTestBehaviorOpaque
	// SemanticsHitTestBehaviorTransparent is hit, and lets what is behind it
	// be hit too.
	//
	// Flutter: ui/semantics.dart SemanticsHitTestBehavior.transparent
	SemanticsHitTestBehaviorTransparent
)

// Name returns the name of the value, such as "opaque".
//
// Flutter: ui/semantics.dart SemanticsHitTestBehavior.name
func (b SemanticsHitTestBehavior) Name() string {
	return enumName([]string{"defer", "opaque", "transparent"}, int(b), "SemanticsHitTestBehavior")
}

// String is "SemanticsHitTestBehavior.opaque".
//
// Flutter: ui/semantics.dart SemanticsHitTestBehavior.toString
func (b SemanticsHitTestBehavior) String() string { return "SemanticsHitTestBehavior." + b.Name() }

// SemanticsActionEvent is a request of an assistive technology to perform an
// action on a node.
//
// Flutter: ui/semantics.dart SemanticsActionEvent
type SemanticsActionEvent struct {
	// ViewID is the view of the node.
	ViewID ViewID
	// NodeID is the ID of the node.
	NodeID int
	// Type is the action.
	Type SemanticsAction
	// Arguments are the arguments of the action: a bool for the cursor moves,
	// an int for a custom action, a string for setText, a map with "base" and
	// "extent" for setSelection, a []float64 for scrollToOffset; nil for the
	// others.
	Arguments any
}

// SemanticsNodeUpdate is what SemanticsUpdateBuilder.UpdateNode records about a
// node. The IDs of the children are in the order the engine uses them.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.updateNode
type SemanticsNodeUpdate struct {
	// ID is the ID of the node.
	ID int
	// Flags are the flags of the node.
	Flags SemanticsFlags
	// Actions are the supported actions as a bit field.
	Actions SemanticsAction
	// Rect is the rectangle of the node in its own coordinates.
	Rect Rect
	// Identifier is the identifier for tests.
	Identifier string
	// Label and LabelAttributes are the label.
	Label           string
	LabelAttributes []StringAttribute
	// Value and ValueAttributes are the value.
	Value           string
	ValueAttributes []StringAttribute
	// IncreasedValue and IncreasedValueAttributes are the value after an
	// increase.
	IncreasedValue           string
	IncreasedValueAttributes []StringAttribute
	// DecreasedValue and DecreasedValueAttributes are the value after a
	// decrease.
	DecreasedValue           string
	DecreasedValueAttributes []StringAttribute
	// Hint and HintAttributes are the hint.
	Hint           string
	HintAttributes []StringAttribute
	// Tooltip is the tooltip.
	Tooltip string
	// TextDirection is the direction of the strings; unspecified if there is
	// none.
	TextDirection TextDirection
	// TextSelectionBase and TextSelectionExtent are the selection, or -1.
	TextSelectionBase, TextSelectionExtent int
	// PlatformViewID is the platform view, or -1.
	PlatformViewID int
	// MaxValueLength and CurrentValueLength are the text field lengths, or -1.
	MaxValueLength, CurrentValueLength int
	// ScrollChildren, ScrollIndex, ScrollPosition, ScrollExtentMax and
	// ScrollExtentMin describe a scrollable; the positions are NaN if unset.
	ScrollChildren, ScrollIndex                      int
	ScrollPosition, ScrollExtentMax, ScrollExtentMin float64
	// Transform maps the node to its parent in traversal order (column major).
	Transform [16]float32
	// TraversalParent is the ID of the traversal parent, or -1.
	TraversalParent int
	// HitTestTransform maps the node to its parent for hit testing.
	HitTestTransform [16]float32
	// ChildrenInTraversalOrder and ChildrenInHitTestOrder are the IDs of the
	// children.
	ChildrenInTraversalOrder, ChildrenInHitTestOrder []int
	// AdditionalActions are the IDs of the custom actions.
	AdditionalActions []int
	// HeadingLevel is 0 to 6.
	HeadingLevel int
	// LinkURL is the URL of a link, or "".
	LinkURL string
	// Role is the role.
	Role SemanticsRole
	// ControlsNodes are the identifiers of the nodes that this one controls;
	// nil if none.
	ControlsNodes []string
	// ValidationResult is the validation result.
	ValidationResult SemanticsValidationResult
	// HitTestBehavior is the hit test behavior.
	HitTestBehavior SemanticsHitTestBehavior
	// InputType is the input type.
	InputType SemanticsInputType
	// Locale is the locale of the strings, or nil.
	Locale *Locale
	// MinValue and MaxValue are the limits of a value, or "".
	MinValue, MaxValue string
}

// SemanticsCustomActionUpdate is what SemanticsUpdateBuilder.UpdateCustomAction
// records about a custom action.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.updateCustomAction
type SemanticsCustomActionUpdate struct {
	// ID is the ID of the action.
	ID int
	// Label is the label of the action; empty for an overriding action.
	Label string
	// Hint is the hint of an overriding action.
	Hint string
	// OverrideID is the bit of the action that this one overrides, or -1.
	OverrideID int
}

// SemanticsUpdate is a set of changes to the semantics tree, which the
// framework sends to the engine.
//
// Flutter: ui/semantics.dart SemanticsUpdate
type SemanticsUpdate struct {
	// Nodes are the nodes that changed, parents before children.
	Nodes []SemanticsNodeUpdate
	// CustomActions are the custom actions that the nodes use.
	CustomActions []SemanticsCustomActionUpdate
}

// SemanticsUpdateBuilder builds a SemanticsUpdate.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder
type SemanticsUpdateBuilder struct {
	update SemanticsUpdate
}

// NewSemanticsUpdateBuilder creates an empty builder.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.new
func NewSemanticsUpdateBuilder() *SemanticsUpdateBuilder { return &SemanticsUpdateBuilder{} }

// UpdateNode adds the new state of a node. A negative ID panics with the tag
// SemanticsUpdateInvalid.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.updateNode
func (b *SemanticsUpdateBuilder) UpdateNode(n SemanticsNodeUpdate) {
	if n.ID < 0 {
		panic(errors.New(tagSemanticsUpdateInvalid, "the ID of a node must not be negative", "id", n.ID))
	}
	b.update.Nodes = append(b.update.Nodes, n)
}

// UpdateCustomAction adds the definition of a custom action.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.updateCustomAction
func (b *SemanticsUpdateBuilder) UpdateCustomAction(a SemanticsCustomActionUpdate) {
	b.update.CustomActions = append(b.update.CustomActions, a)
}

// Build returns the update. The builder can not be used again.
//
// Flutter: ui/semantics.dart SemanticsUpdateBuilder.build
func (b *SemanticsUpdateBuilder) Build() *SemanticsUpdate {
	u := b.update
	b.update = SemanticsUpdate{}
	return &u
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSemanticsUpdateInvalid = "SemanticsUpdateInvalid"
)
