package uia

import (
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// ValueKind says what a Value holds.
//
// Flutter: no counterpart
type ValueKind int

// The kinds of Value.
const (
	// ValueEmpty is VT_EMPTY: the property has no value from this provider, and UI Automation uses its
	// default.
	//
	// Flutter: no counterpart
	ValueEmpty ValueKind = iota
	// ValueBool is VT_BOOL.
	//
	// Flutter: no counterpart
	ValueBool
	// ValueInt is VT_I4.
	//
	// Flutter: no counterpart
	ValueInt
	// ValueFloat is VT_R8.
	//
	// Flutter: no counterpart
	ValueFloat
	// ValueString is VT_BSTR.
	//
	// Flutter: no counterpart
	ValueString
)

// Value is the value of a property in plain Go: the COM glue turns it into a VARIANT.
//
// Flutter: no counterpart
type Value struct {
	Kind ValueKind
	B    bool
	I    int32
	F    float64
	S    string
}

// Bool makes a boolean Value.
//
// Flutter: no counterpart
func Bool(b bool) Value { return Value{Kind: ValueBool, B: b} }

// Int makes an integer Value.
//
// Flutter: no counterpart
func Int(i int) Value { return Value{Kind: ValueInt, I: int32(i)} }

// Float makes a floating-point Value.
//
// Flutter: no counterpart
func Float(f float64) Value { return Value{Kind: ValueFloat, F: f} }

// String makes a string Value.
//
// Flutter: no counterpart
func String(s string) Value { return Value{Kind: ValueString, S: s} }

// frameworkID is the FrameworkId property of every element.
const frameworkID = "Glutter"

// className is the ClassName property of the elements below the root.
const className = "GlutterSemanticsNode"

// ControlTypeOf is the control type of a node. The SemanticsRole decides when there is one; otherwise
// the flags do, in this order: text field, slider, link, header, image, then the buttons and the
// checkable nodes; what has text is a Text; the rest is a Group. The root of a view is a Pane.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::GetUIAControlType
func ControlTypeOf(n *Node) ControlType {
	if n.ID == RootID {
		return ControlTypePane
	}
	if t, ok := roleControlTypes[n.Role]; ok {
		return t
	}
	f := n.Flags
	switch {
	case f.IsTextField:
		return ControlTypeEdit
	case f.IsSlider:
		return ControlTypeSlider
	case f.IsLink:
		return ControlTypeHyperlink
	case f.IsHeader:
		return ControlTypeText
	case f.IsImage:
		return ControlTypeImage
	case isRadio(n):
		return ControlTypeRadioButton
	case f.IsChecked != ui.CheckedStateNone:
		return ControlTypeCheckBox
	case f.IsButton:
		return ControlTypeButton
	case f.IsToggled != ui.TristateNone:
		return ControlTypeButton
	case n.Label != "" || n.Value != "" || n.Hint != "":
		return ControlTypeText
	}
	return ControlTypeGroup
}

// roleControlTypes maps a SemanticsRole to a control type, as the browsers' ARIA roles are mapped
// (ax_platform_node_win.cc). A SemanticsRole that is not here (none) is decided by the flags.
var roleControlTypes = map[ui.SemanticsRole]ControlType{
	ui.SemanticsRoleTab:              ControlTypeTabItem,
	ui.SemanticsRoleTabBar:           ControlTypeTab,
	ui.SemanticsRoleTabPanel:         ControlTypePane,
	ui.SemanticsRoleDialog:           ControlTypeWindow,
	ui.SemanticsRoleAlertDialog:      ControlTypeWindow,
	ui.SemanticsRoleTable:            ControlTypeTable,
	ui.SemanticsRoleCell:             ControlTypeDataItem,
	ui.SemanticsRoleRow:              ControlTypeDataItem,
	ui.SemanticsRoleColumnHeader:     ControlTypeHeaderItem,
	ui.SemanticsRoleDragHandle:       ControlTypeButton,
	ui.SemanticsRoleSpinButton:       ControlTypeSpinner,
	ui.SemanticsRoleComboBox:         ControlTypeComboBox,
	ui.SemanticsRoleMenuBar:          ControlTypeMenuBar,
	ui.SemanticsRoleMenu:             ControlTypeMenu,
	ui.SemanticsRoleMenuItem:         ControlTypeMenuItem,
	ui.SemanticsRoleMenuItemCheckbox: ControlTypeMenuItem,
	ui.SemanticsRoleMenuItemRadio:    ControlTypeMenuItem,
	ui.SemanticsRoleAlert:            ControlTypeText,
	ui.SemanticsRoleStatus:           ControlTypeStatusBar,
	ui.SemanticsRoleList:             ControlTypeList,
	ui.SemanticsRoleListItem:         ControlTypeListItem,
	ui.SemanticsRoleComplementary:    ControlTypeGroup,
	ui.SemanticsRoleContentInfo:      ControlTypeGroup,
	ui.SemanticsRoleMain:             ControlTypeGroup,
	ui.SemanticsRoleNavigation:       ControlTypeGroup,
	ui.SemanticsRoleRegion:           ControlTypeGroup,
	ui.SemanticsRoleForm:             ControlTypeGroup,
	ui.SemanticsRoleLoadingSpinner:   ControlTypeProgressBar,
	ui.SemanticsRoleProgressBar:      ControlTypeProgressBar,
	ui.SemanticsRoleRadioGroup:       ControlTypeGroup,
	ui.SemanticsRoleTooltip:          ControlTypeToolTip,
	ui.SemanticsRoleHotKey:           ControlTypeText,
}

// isRadio says whether the node is a radio button: a checkable node in a group where one is checked.
func isRadio(n *Node) bool {
	return n.Flags.IsChecked != ui.CheckedStateNone && n.Flags.IsInMutuallyExclusiveGroup ||
		n.Role == ui.SemanticsRoleMenuItemRadio
}

// textOf is the text of a text field: its value.
func textOf(n *Node) string { return n.Value }

// Name is the name of the node (the Name property): its label; the root has the title of the window
// when it has no label; a node with no label has its value, unless the value is shown by a pattern (a
// text field's text, a slider's value), and then its hint.
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetName
func (m *Model) Name(n *Node) string {
	if n == nil {
		return ""
	}
	if n.Label != "" {
		return n.Label
	}
	if n.ID == RootID {
		return m.Title(n.View)
	}
	if n.Value != "" && !n.Flags.IsTextField && !n.Flags.IsSlider && !hasRange(n) {
		return n.Value
	}
	return n.Hint
}

// HelpText is the tooltip of the node; a node that has a label shows its hint as help.
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetDescription
func HelpText(n *Node) string {
	if n.Tooltip != "" {
		return n.Tooltip
	}
	if n.Label != "" {
		return n.Hint
	}
	return ""
}

// AutomationID is the identifier the app gave the node, or else the node's number.
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetStringAttribute
func AutomationID(n *Node) string {
	if n.Identifier != "" {
		return n.Identifier
	}
	return strconv.Itoa(n.ID)
}

// isControlElement says whether the node is a control: it has something to say or to do. A container
// that has none of that is left out of the control view, as Chromium leaves out its ignored nodes.
func isControlElement(n *Node) bool {
	if n.ID == RootID {
		return true
	}
	if n.Label != "" || n.Value != "" || n.Hint != "" || n.Tooltip != "" || n.Role != ui.SemanticsRoleNone {
		return true
	}
	if n.Flags != ui.SemanticsFlagsNone && n.Flags != (ui.SemanticsFlags{HasImplicitScrolling: true}) {
		return true
	}
	return n.Actions&^ui.SemanticsActionShowOnScreen != 0
}

// isContentElement says whether the node is content: a control that shows text, an image or a field.
func isContentElement(n *Node) bool {
	return n.ID != RootID && isControlElement(n) && (n.Label != "" || n.Value != "" || n.Hint != "" || n.Flags.IsTextField || n.Flags.IsImage)
}

// toggleState is the Toggle pattern's state: checked and toggled nodes have one.
func toggleState(n *Node) (state int, ok bool) {
	switch n.Flags.IsChecked {
	case ui.CheckedStateIsTrue:
		return ToggleOn, true
	case ui.CheckedStateIsFalse:
		return ToggleOff, true
	case ui.CheckedStateMixed:
		return ToggleIndeterminate, true
	}
	switch n.Flags.IsToggled {
	case ui.TristateIsTrue:
		return ToggleOn, true
	case ui.TristateIsFalse:
		return ToggleOff, true
	}
	return 0, false
}

// expandState is the ExpandCollapse pattern's state.
func expandState(n *Node) (state int, ok bool) {
	switch n.Flags.IsExpanded {
	case ui.TristateIsTrue:
		return ExpandCollapseExpanded, true
	case ui.TristateIsFalse:
		return ExpandCollapseCollapsed, true
	}
	switch {
	case n.Actions&ui.SemanticsActionCollapse != 0:
		return ExpandCollapseExpanded, true
	case n.Actions&ui.SemanticsActionExpand != 0:
		return ExpandCollapseCollapsed, true
	}
	return 0, false
}

// rangeValue is the value of a slider, a spin button or a progress bar.
type rangeValue struct {
	current, min, max, small, large float64
}

// hasRange says whether the node has the RangeValue pattern.
func hasRange(n *Node) bool {
	_, ok := rangeOf(n)
	return ok
}

// rangeOf is the numeric value of a node that has the RangeValue pattern: a slider, a spin button or a
// progress bar. The framework gives the value as the text that is said ("50%", "3"): a percentage goes
// from 0 to 100; another number goes between the minimum and the maximum that the node says, or from 0
// to 100 when it says none. The small change is a hundredth of the range, at most 1; the large one is
// ten times that.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::get_Value
func rangeOf(n *Node) (rangeValue, bool) {
	switch {
	case n.Flags.IsSlider, n.Role == ui.SemanticsRoleSpinButton:
	case n.Role == ui.SemanticsRoleProgressBar && strings.TrimSpace(n.Value) != "":
	default:
		return rangeValue{}, false
	}
	v := rangeValue{min: 0, max: 100}
	text := strings.TrimSpace(n.Value)
	percent := strings.HasSuffix(text, "%")
	if f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(text, "%")), 64); err == nil {
		v.current = f
	}
	if !percent {
		if f, err := strconv.ParseFloat(strings.TrimSpace(n.MinValue), 64); err == nil {
			v.min = f
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(n.MaxValue), 64); err == nil {
			v.max = f
		}
	}
	v.small = min(1, (v.max-v.min)/100)
	if v.small <= 0 {
		v.small = 1
	}
	v.large = min(v.small*10, max(v.max-v.min, v.small))
	return v, true
}

// hasScroll says whether the node scrolls on the axis (vertical or horizontal), as its actions say.
func hasScroll(n *Node, vertical bool) bool {
	if vertical {
		return n.Actions&(ui.SemanticsActionScrollUp|ui.SemanticsActionScrollDown) != 0
	}
	return n.Actions&(ui.SemanticsActionScrollLeft|ui.SemanticsActionScrollRight) != 0
}

// NoScroll is UIA_ScrollPatternNoScroll: the scroll percentage of an axis on which the node does not
// scroll.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::get_HorizontalScrollPercent
const NoScroll = -1.0

// scrollPercent is how far the node is scrolled, from 0 to 100, on the axis; NoScroll when it does not
// scroll there. A node that does not say its extents is at 0.
func scrollPercent(n *Node, vertical bool) float64 {
	if !hasScroll(n, vertical) {
		return NoScroll
	}
	span := n.ScrollExtentMax - n.ScrollExtentMin
	if math.IsNaN(span) || math.IsNaN(n.ScrollPosition) || span <= 0 {
		return 0
	}
	return min(100, max(0, (n.ScrollPosition-n.ScrollExtentMin)/span*100))
}

// scrollViewSize is the part of the content that the node shows, in percent: the size of the viewport
// over the size of the viewport plus the extent it scrolls. 100 when the node does not scroll on the
// axis.
func scrollViewSize(n *Node, vertical bool) float64 {
	if !hasScroll(n, vertical) {
		return 100
	}
	viewport := n.Rect.Width()
	if vertical {
		viewport = n.Rect.Height()
	}
	span := n.ScrollExtentMax - n.ScrollExtentMin
	if math.IsNaN(span) || span < 0 || viewport <= 0 {
		return 100
	}
	return viewport / (viewport + span) * 100
}

// watchedProperties are the properties whose change raises a property-changed event.
var watchedProperties = []PropertyID{
	PropName, PropHelpText, PropIsEnabled, PropIsOffscreen, PropToggleToggleState,
	PropExpandCollapseExpandCollapseState, PropValueValue, PropRangeValueValue,
	PropSelectionItemIsSelected, PropScrollHorizontalScrollPercent, PropScrollVerticalScrollPercent,
}

// watched are the values of the watched properties of the node.
func (m *Model) watched(n *Node) map[PropertyID]Value {
	out := make(map[PropertyID]Value, len(watchedProperties))
	for _, p := range watchedProperties {
		out[p] = m.Property(n, p)
	}
	return out
}

// Property is the value of a property of the node, as GetPropertyValue answers it. A property that the
// node has no value for is ValueEmpty, which UI Automation fills in with its default.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::GetPropertyValue
func (m *Model) Property(n *Node, id PropertyID) Value {
	if n == nil {
		return Value{}
	}
	patterns := m.Patterns(n)
	switch id {
	case PropControlType:
		return Int(int(ControlTypeOf(n)))
	case PropName:
		return String(m.Name(n))
	case PropHelpText:
		return String(HelpText(n))
	case PropAutomationId:
		return String(AutomationID(n))
	case PropClassName:
		if n.ID == RootID {
			return Value{}
		}
		return String(className)
	case PropFrameworkId:
		return String(frameworkID)
	case PropIsEnabled:
		return Bool(n.Flags.IsEnabled != ui.TristateIsFalse)
	case PropIsKeyboardFocusable:
		return Bool(n.Flags.IsFocused != ui.TristateNone || n.Actions&ui.SemanticsActionFocus != 0)
	case PropHasKeyboardFocus:
		return Bool(n.Flags.IsFocused == ui.TristateIsTrue && m.Active(n.View))
	case PropIsOffscreen:
		return Bool(n.Flags.IsHidden)
	case PropIsControlElement:
		return Bool(isControlElement(n))
	case PropIsContentElement:
		return Bool(isContentElement(n))
	case PropIsPassword:
		return Bool(n.Flags.IsObscured)
	case PropLiveSetting:
		if n.Flags.IsLiveRegion {
			return Int(LivePolite)
		}
		return Int(LiveOff)
	case PropHeadingLevel:
		if n.HeadingLevel > 0 {
			return Int(n.HeadingLevel)
		}
	case PropToggleToggleState:
		if s, ok := toggleState(n); ok && patterns.Has(PatternToggle) {
			return Int(s)
		}
	case PropExpandCollapseExpandCollapseState:
		if s, ok := expandState(n); ok && patterns.Has(PatternExpandCollapse) {
			return Int(s)
		}
	case PropValueValue:
		if patterns.Has(PatternValue) {
			if n.Flags.IsObscured {
				return String("")
			}
			return String(textOf(n))
		}
	case PropValueIsReadOnly:
		if patterns.Has(PatternValue) {
			return Bool(n.Flags.IsReadOnly)
		}
	case PropRangeValueValue, PropRangeValueMinimum, PropRangeValueMaximum, PropRangeValueSmallChange, PropRangeValueLargeChange:
		if r, ok := rangeOf(n); ok {
			switch id {
			case PropRangeValueValue:
				return Float(r.current)
			case PropRangeValueMinimum:
				return Float(r.min)
			case PropRangeValueMaximum:
				return Float(r.max)
			case PropRangeValueSmallChange:
				return Float(r.small)
			default:
				return Float(r.large)
			}
		}
	case PropRangeValueIsReadOnly:
		if _, ok := rangeOf(n); ok {
			return Bool(n.Actions&(ui.SemanticsActionIncrease|ui.SemanticsActionDecrease) == 0)
		}
	case PropSelectionItemIsSelected:
		if patterns.Has(PatternSelectionItem) {
			return Bool(isSelected(n))
		}
	case PropScrollHorizontalScrollPercent, PropScrollVerticalScrollPercent, PropScrollHorizontalViewSize, PropScrollVerticalViewSize,
		PropScrollHorizontallyScrollable, PropScrollVerticallyScrollable:
		if patterns.Has(PatternScroll) {
			switch id {
			case PropScrollHorizontalScrollPercent:
				return Float(scrollPercent(n, false))
			case PropScrollVerticalScrollPercent:
				return Float(scrollPercent(n, true))
			case PropScrollHorizontalViewSize:
				return Float(scrollViewSize(n, false))
			case PropScrollVerticalViewSize:
				return Float(scrollViewSize(n, true))
			case PropScrollHorizontallyScrollable:
				return Bool(hasScroll(n, false))
			default:
				return Bool(hasScroll(n, true))
			}
		}
	case PropSelectionCanSelectMultiple, PropSelectionIsSelectionRequired:
		if patterns.Has(PatternSelection) {
			return Bool(false)
		}
	}
	return Value{}
}

// isSelected says whether the node is selected: it is checked (a radio button) or its selected flag is
// true.
func isSelected(n *Node) bool {
	if n.Flags.IsSelected != ui.TristateNone {
		return n.Flags.IsSelected == ui.TristateIsTrue
	}
	return n.Flags.IsChecked == ui.CheckedStateIsTrue
}
