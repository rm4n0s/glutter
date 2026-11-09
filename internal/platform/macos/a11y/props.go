package a11y

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/ui"
)

// This file maps a node of the semantics tree to what NSAccessibility says about an element: the role and the
// subrole, the label, the help, the value, the state, the actions and the selected range of a text field. The
// strings are the values of the NSAccessibility constants (they are the Accessibility API's AX names), so the
// tables are plain Go.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa

// The roles of NSAccessibility that the mapping uses.
const (
	RoleGroup       = "AXGroup"
	RoleButton      = "AXButton"
	RoleCheckBox    = "AXCheckBox"
	RoleRadioButton = "AXRadioButton"
	RoleSlider      = "AXSlider"
	RoleTextField   = "AXTextField"
	RoleTextArea    = "AXTextArea"
	RoleStaticText  = "AXStaticText"
	RoleLink        = "AXLink"
	RoleImage       = "AXImage"
	RoleList        = "AXList"
	RoleRow         = "AXRow"
	RoleCell        = "AXCell"
	RoleColumn      = "AXColumn"
	RoleTable       = "AXTable"
	RoleMenu        = "AXMenu"
	RoleMenuItem    = "AXMenuItem"
	RoleMenuBar     = "AXMenuBar"
	RoleTabGroup    = "AXTabGroup"
	RoleProgress    = "AXProgressIndicator"
	RolePopUp       = "AXPopUpButton"
	RoleComboBox    = "AXComboBox"
	RoleIncrementor = "AXIncrementor"
	RoleHeading     = "AXHeading"
	RoleHelpTag     = "AXHelpTag"
	RoleUnknown     = "AXUnknown"
)

// The subroles that the mapping uses.
const (
	SubroleSwitch      = "AXSwitch"
	SubroleToggle      = "AXToggle"
	SubroleDialog      = "AXDialog"
	SubroleSearchField = "AXSearchField"
	SubroleSecureField = "AXSecureTextField"
	SubroleTabButton   = "AXTabButton"
	SubroleStatus      = "AXApplicationStatus"
	SubroleAlert       = "AXApplicationAlert"
)

// roleOfSemantics maps a SemanticsRole to a role, as Chromium maps the ARIA roles (ax_platform_node_mac.mm). A
// SemanticsRole that is not here is decided by the flags.
var roleOfSemantics = map[ui.SemanticsRole]string{
	ui.SemanticsRoleTab:              RoleRadioButton, // a tab is a radio button of the tab group
	ui.SemanticsRoleTabBar:           RoleTabGroup,
	ui.SemanticsRoleTabPanel:         RoleGroup,
	ui.SemanticsRoleDialog:           RoleGroup,
	ui.SemanticsRoleAlertDialog:      RoleGroup,
	ui.SemanticsRoleTable:            RoleTable,
	ui.SemanticsRoleCell:             RoleCell,
	ui.SemanticsRoleRow:              RoleRow,
	ui.SemanticsRoleColumnHeader:     RoleCell,
	ui.SemanticsRoleDragHandle:       RoleButton,
	ui.SemanticsRoleSpinButton:       RoleIncrementor,
	ui.SemanticsRoleComboBox:         RolePopUp,
	ui.SemanticsRoleMenuBar:          RoleMenuBar,
	ui.SemanticsRoleMenu:             RoleMenu,
	ui.SemanticsRoleMenuItem:         RoleMenuItem,
	ui.SemanticsRoleMenuItemCheckbox: RoleMenuItem,
	ui.SemanticsRoleMenuItemRadio:    RoleMenuItem,
	ui.SemanticsRoleAlert:            RoleGroup,
	ui.SemanticsRoleStatus:           RoleGroup,
	ui.SemanticsRoleList:             RoleList,
	ui.SemanticsRoleListItem:         RoleGroup,
	ui.SemanticsRoleComplementary:    RoleGroup,
	ui.SemanticsRoleContentInfo:      RoleGroup,
	ui.SemanticsRoleMain:             RoleGroup,
	ui.SemanticsRoleNavigation:       RoleGroup,
	ui.SemanticsRoleRegion:           RoleGroup,
	ui.SemanticsRoleForm:             RoleGroup,
	ui.SemanticsRoleLoadingSpinner:   RoleProgress,
	ui.SemanticsRoleProgressBar:      RoleProgress,
	ui.SemanticsRoleRadioGroup:       RoleGroup,
	ui.SemanticsRoleTooltip:          RoleHelpTag,
	ui.SemanticsRoleHotKey:           RoleStaticText,
}

// isRadio says whether the node is a radio button: a checkable node in a group where one is checked.
func isRadio(n *Node) bool {
	return n.Flags.IsChecked != ui.CheckedStateNone && n.Flags.IsInMutuallyExclusiveGroup ||
		n.Role == ui.SemanticsRoleMenuItemRadio
}

// Role is the NSAccessibility role of a node. The SemanticsRole decides when there is one; otherwise the flags
// do, in this order: text field, slider, link, header, image, then the radio buttons, the checkable nodes and the
// buttons; what has text is a static text; the rest is a group. The root of a view is a group.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilityRole
func Role(n *Node) string {
	if n.ID == RootID {
		return RoleGroup
	}
	if r, ok := roleOfSemantics[n.Role]; ok {
		return r
	}
	f := n.Flags
	switch {
	case f.IsTextField:
		if f.IsMultiline {
			return RoleTextArea
		}
		return RoleTextField
	case f.IsSlider:
		return RoleSlider
	case f.IsLink:
		return RoleLink
	case f.IsHeader:
		return RoleHeading
	case f.IsImage:
		return RoleImage
	case isRadio(n):
		return RoleRadioButton
	case f.IsChecked != ui.CheckedStateNone:
		return RoleCheckBox
	case f.IsButton:
		return RoleButton
	case f.IsToggled != ui.TristateNone:
		return RoleCheckBox // a switch is a check box with the switch subrole
	case n.Label != "" || n.Value != "" || n.Hint != "":
		return RoleStaticText
	}
	return RoleGroup
}

// Subrole is the subrole of a node, "" for none: a toggled node is a switch, a dialog is a dialog, and the status
// and the alert are the application's.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilitySubrole
func Subrole(n *Node) string {
	switch {
	case n.Flags.IsToggled != ui.TristateNone && n.Flags.IsChecked == ui.CheckedStateNone:
		return SubroleSwitch
	case n.Role == ui.SemanticsRoleDialog || n.Role == ui.SemanticsRoleAlertDialog:
		return SubroleDialog
	case n.Role == ui.SemanticsRoleStatus:
		return SubroleStatus
	case n.Role == ui.SemanticsRoleAlert:
		return SubroleAlert
	case n.Flags.IsTextField && n.Flags.IsObscured:
		return SubroleSecureField
	}
	return ""
}

// Label is the label of a node, which is what VoiceOver reads first (AXDescription, accessibilityLabel): its label;
// the root has the title of the window when it has no label; a node with no label has its value, unless the value
// is shown as the value (a text field's text, a slider's value), and then its hint.
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetName
func (m *Model) Label(n *Node) string {
	if n == nil {
		return ""
	}
	if n.Label != "" {
		return n.Label
	}
	if n.ID == RootID {
		return m.Title(n.View)
	}
	if n.Value != "" && !n.Flags.IsTextField && !n.Flags.IsSlider && Role(n) != RoleIncrementor && Role(n) != RoleProgress {
		return n.Value
	}
	return n.Hint
}

// Help is the help of a node (AXHelp): its tooltip; a node that has a label shows its hint as help.
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetDescription
func Help(n *Node) string {
	if n.Tooltip != "" {
		return n.Tooltip
	}
	if n.Label != "" {
		return n.Hint
	}
	return ""
}

// Identifier is the identifier the app gave the node, or else the node's number (AXIdentifier).
//
// Flutter: no counterpart
func Identifier(n *Node) string {
	if n.Identifier != "" {
		return n.Identifier
	}
	return strconv.Itoa(n.ID)
}

// ValueKind says what the value of an element is.
//
// Flutter: no counterpart
type ValueKind int

// The kinds of value of an element.
const (
	// ValueNone: the element has no value.
	ValueNone ValueKind = iota
	// ValueString: the value is the text of a text field or of a node that has a value of its own.
	ValueString
	// ValueNumber: the value is a number: the state of a check box or a switch (0, 1 or 2 for a mixed one), or the
	// position of a slider or a progress bar.
	ValueNumber
)

// Value is the value of a node: a check box has its state (0, 1, 2 for mixed), a switch 0 or 1, a slider, a spin
// button and a progress bar the number that its text says (a percentage is its number), a text field its text, and
// any other node with a value of its own and a label shows that value.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilityValue
func Value(n *Node) (kind ValueKind, text string, number float64) {
	switch n.Flags.IsChecked {
	case ui.CheckedStateIsTrue:
		return ValueNumber, "", 1
	case ui.CheckedStateIsFalse:
		return ValueNumber, "", 0
	case ui.CheckedStateMixed:
		return ValueNumber, "", 2
	}
	switch n.Flags.IsToggled {
	case ui.TristateIsTrue:
		return ValueNumber, "", 1
	case ui.TristateIsFalse:
		return ValueNumber, "", 0
	}
	if r, ok := RangeOf(n); ok {
		return ValueNumber, "", r.Current
	}
	if n.Flags.IsTextField {
		return ValueString, n.Value, 0
	}
	if n.Value != "" && n.Label != "" {
		return ValueString, n.Value, 0
	}
	return ValueNone, "", 0
}

// ValueText is the value of a node as text, for the comparison of two states of a node.
func ValueText(n *Node) string {
	kind, text, num := Value(n)
	switch kind {
	case ValueString:
		return text
	case ValueNumber:
		return strconv.FormatFloat(num, 'f', -1, 64)
	}
	return ""
}

// Range is the value of a slider, a spin button or a progress bar, and its limits.
//
// Flutter: no counterpart
type Range struct{ Current, Min, Max float64 }

// RangeOf is the numeric value of a node that has a range: a slider, a spin button or a progress bar. The
// framework gives the value as the text that is said ("50%", "3"): a percentage goes from 0 to 100; another number
// goes between the minimum and the maximum that the node says, or from 0 to 100 when it says none.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilityMinValue
func RangeOf(n *Node) (Range, bool) {
	switch {
	case n.Flags.IsSlider, n.Role == ui.SemanticsRoleSpinButton:
	case n.Role == ui.SemanticsRoleProgressBar && strings.TrimSpace(n.Value) != "":
	default:
		return Range{}, false
	}
	r := Range{Min: 0, Max: 100}
	text := strings.TrimSpace(n.Value)
	percent := strings.HasSuffix(text, "%")
	if f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(text, "%")), 64); err == nil {
		r.Current = f
	}
	if !percent {
		if f, err := strconv.ParseFloat(strings.TrimSpace(n.MinValue), 64); err == nil {
			r.Min = f
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(n.MaxValue), 64); err == nil {
			r.Max = f
		}
	}
	return r, true
}

// Enabled says whether the node is enabled: it is, unless its enabled state is false.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::isAccessibilityEnabled
func Enabled(n *Node) bool { return n.Flags.IsEnabled != ui.TristateIsFalse }

// Selected says whether the node is selected.
func Selected(n *Node) bool { return n.Flags.IsSelected == ui.TristateIsTrue }

// ExpandedState is 1 for an expanded node, 0 for a collapsed one and -1 for a node that cannot expand.
func ExpandedState(n *Node) int {
	switch n.Flags.IsExpanded {
	case ui.TristateIsTrue:
		return 1
	case ui.TristateIsFalse:
		return 0
	}
	switch {
	case n.Actions&ui.SemanticsActionCollapse != 0:
		return 1
	case n.Actions&ui.SemanticsActionExpand != 0:
		return 0
	}
	return -1
}

// Hidden says that the node is not on the screen for the assistive technologies.
func Hidden(n *Node) bool { return n.Flags.IsHidden }

// Focusable says whether the node can take the keyboard focus.
func Focusable(n *Node) bool {
	return n.Actions&ui.SemanticsActionFocus != 0 || n.Flags.IsFocused != ui.TristateNone || n.Flags.IsTextField
}

// The actions of NSAccessibility that the mapping uses.
const (
	ActionPress         = "AXPress"
	ActionIncrement     = "AXIncrement"
	ActionDecrement     = "AXDecrement"
	ActionShowMenu      = "AXShowMenu"
	ActionCancel        = "AXCancel"
	ActionScrollVisible = "AXScrollToVisible"
	ActionScrollUp      = "AXScrollUpByPage"
	ActionScrollDown    = "AXScrollDownByPage"
	ActionScrollLeft    = "AXScrollLeftByPage"
	ActionScrollRight   = "AXScrollRightByPage"
)

// actionTable maps the actions of a node to the names of NSAccessibility, in the order VoiceOver lists them.
var actionTable = []struct {
	name   string
	action ui.SemanticsAction
}{
	{ActionPress, ui.SemanticsActionTap},
	{ActionIncrement, ui.SemanticsActionIncrease},
	{ActionDecrement, ui.SemanticsActionDecrease},
	{ActionShowMenu, ui.SemanticsActionLongPress},
	{ActionCancel, ui.SemanticsActionDismiss},
	{ActionScrollVisible, ui.SemanticsActionShowOnScreen},
	{ActionScrollUp, ui.SemanticsActionScrollDown}, // the content moves up when the view scrolls down: Flutter's names are the content's
	{ActionScrollDown, ui.SemanticsActionScrollUp},
	{ActionScrollLeft, ui.SemanticsActionScrollRight},
	{ActionScrollRight, ui.SemanticsActionScrollLeft},
}

// ActionNames are the NSAccessibility actions that the node supports, in a fixed order: a node that can be tapped
// can be pressed, and so on (the table above). The root has none.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilityActionNames
func ActionNames(n *Node) []string {
	if n.ID == RootID {
		return nil
	}
	var out []string
	for _, e := range actionTable {
		if n.Actions&e.action != 0 {
			out = append(out, e.name)
		}
	}
	return out
}

// ActionFor is the semantics action that an NSAccessibility action asks for, and whether the node supports it.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_mac.mm AXPlatformNodeCocoa::accessibilityPerformAction
func ActionFor(n *Node, name string) (ui.SemanticsAction, bool) {
	for _, e := range actionTable {
		if e.name == name && n.Actions&e.action != 0 {
			return e.action, true
		}
	}
	return 0, false
}

// UTF16Offset is the offset in UTF-16 units of the byte offset in the UTF-8 text.
func UTF16Offset(s string, byteOffset int) int {
	n := 0
	for i := 0; i < len(s) && i < byteOffset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// ByteOffset is the byte offset in the UTF-8 text of the offset in UTF-16 units; one past the end is the end, and
// an offset in the middle of a surrogate pair is the start of the pair.
func ByteOffset(s string, utf16Offset int) int {
	if utf16Offset <= 0 {
		return 0
	}
	units := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		w := 1
		if r >= 0x10000 {
			w = 2
		}
		if units+w > utf16Offset {
			return i
		}
		units += w
		i += size
		if units == utf16Offset {
			return i
		}
	}
	return len(s)
}

// Selection is the caret (the extent) and the base of the selection of a text field in UTF-16 units; both are -1 when
// the node is not a text field or has no selection.
func Selection(n *Node) (caret, base int) {
	if !n.Flags.IsTextField || n.TextSelectionBase < 0 || n.TextSelectionExtent < 0 {
		return -1, -1
	}
	return UTF16Offset(n.Value, n.TextSelectionExtent), UTF16Offset(n.Value, n.TextSelectionBase)
}

// SelectedRange is the selected range of a text field as NSAccessibility has it: the location and the length in
// UTF-16 units; the caret is a range of length 0. ok is false when the field has no selection.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterTextInputSemanticsObject.mm accessibilitySelectedTextRange
func SelectedRange(n *Node) (location, length int, ok bool) {
	caret, base := Selection(n)
	if caret < 0 {
		return 0, 0, false
	}
	return min(caret, base), int(math.Abs(float64(caret - base))), true
}

// SelectionArguments is the argument of the set selection action for the range of NSAccessibility (UTF-16 units):
// the offsets in the bytes of the text, as the framework has them.
func SelectionArguments(text string, location, length int) map[string]int {
	return map[string]int{
		"base":   ByteOffset(text, location),
		"extent": ByteOffset(text, location+length),
	}
}

// NumberOfCharacters is the length of the text of a field in UTF-16 units.
func NumberOfCharacters(n *Node) int { return UTF16Offset(n.Value, len(n.Value)) }
