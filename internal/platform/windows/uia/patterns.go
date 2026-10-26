package uia

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// PatternSet is the set of patterns that a node supports.
//
// Flutter: no counterpart
type PatternSet uint16

// The patterns that the bridge implements, in the order GetPatternProvider's table has them.
var patternOrder = []PatternID{
	PatternInvoke, PatternToggle, PatternValue, PatternRangeValue, PatternScroll, PatternText,
	PatternExpandCollapse, PatternSelection, PatternSelectionItem,
}

func patternBit(p PatternID) PatternSet {
	for i, q := range patternOrder {
		if p == q {
			return 1 << i
		}
	}
	return 0
}

// Has says whether the set has the pattern.
//
// Flutter: no counterpart
func (s PatternSet) Has(p PatternID) bool { return s&patternBit(p) != 0 }

// List returns the patterns of the set.
//
// Flutter: no counterpart
func (s PatternSet) List() []PatternID {
	var out []PatternID
	for _, p := range patternOrder {
		if s.Has(p) {
			out = append(out, p)
		}
	}
	return out
}

// Patterns are the patterns a node supports; its flags and actions say which:
//
//	Invoke          the tap action, on a node that is not checkable, selectable, a text field or a slider
//	Toggle          a checked or toggled node that is not a radio button
//	SelectionItem   a radio button, or a node with a selected flag
//	Selection       a node with children that have a selected flag
//	Value           a text field
//	RangeValue      a slider, a spin button, a progress bar that says its value
//	Scroll          a node with scroll actions
//	Text            a text field
//	ExpandCollapse  a node with an expanded flag, or the expand or collapse action
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::GetPatternProviderFactory
func (m *Model) Patterns(n *Node) PatternSet {
	var s PatternSet
	if n == nil {
		return s
	}
	_, toggles := toggleState(n)
	radio := isRadio(n)
	selectable := n.Flags.IsSelected != ui.TristateNone || radio
	if toggles && !radio {
		s |= patternBit(PatternToggle)
	}
	if selectable {
		s |= patternBit(PatternSelectionItem)
	}
	if n.Actions&ui.SemanticsActionTap != 0 && !toggles && !radio && !n.Flags.IsTextField && !n.Flags.IsSlider {
		s |= patternBit(PatternInvoke)
	}
	if n.Flags.IsTextField {
		s |= patternBit(PatternValue) | patternBit(PatternText)
	}
	if hasRange(n) {
		s |= patternBit(PatternRangeValue)
	}
	if hasScroll(n, true) || hasScroll(n, false) {
		s |= patternBit(PatternScroll)
	}
	if _, ok := expandState(n); ok {
		s |= patternBit(PatternExpandCollapse)
	}
	if m.hasSelection(n) {
		s |= patternBit(PatternSelection)
	}
	return s
}

// hasSelection says whether the node has children that can be selected.
func (m *Model) hasSelection(n *Node) bool {
	for _, c := range m.Children(n) {
		if m.Node(c).Flags.IsSelected != ui.TristateNone {
			return true
		}
	}
	return false
}

// SelectedChildren are the children of the node that are selected, in order.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::GetSelection
func (m *Model) SelectedChildren(n *Node) []NodeRef {
	var out []NodeRef
	for _, c := range m.Children(n) {
		if m.Node(c).Flags.IsSelected == ui.TristateIsTrue {
			out = append(out, c)
		}
	}
	return out
}

// SelectionContainer is the nearest ancestor of the node that has the Selection pattern, or nil.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::get_SelectionContainer
func (m *Model) SelectionContainer(n *Node) *Node {
	for p := m.ParentOf(n); p != nil; p = m.ParentOf(p) {
		if m.hasSelection(p) {
			return p
		}
	}
	return nil
}

// Status is the outcome of a request that a client made to a pattern.
//
// Flutter: no counterpart
type Status int

// The outcomes.
const (
	// StatusOK: the request was carried out, or posted to the framework.
	//
	// Flutter: no counterpart
	StatusOK Status = iota
	// StatusGone: the node is not in the tree (UIA_E_ELEMENTNOTAVAILABLE).
	//
	// Flutter: no counterpart
	StatusGone
	// StatusDisabled: the node is disabled (UIA_E_ELEMENTNOTENABLED).
	//
	// Flutter: no counterpart
	StatusDisabled
	// StatusNotSupported: the node cannot do that (UIA_E_NOTSUPPORTED).
	//
	// Flutter: no counterpart
	StatusNotSupported
	// StatusInvalid: an argument is out of range (E_INVALIDARG).
	//
	// Flutter: no counterpart
	StatusInvalid
)

// request finds the node of the request and checks that it is there and enabled.
func (m *Model) request(ref NodeRef) (*Node, Status) {
	n := m.Node(ref)
	switch {
	case n == nil:
		return nil, StatusGone
	case n.Flags.IsEnabled == ui.TristateIsFalse:
		return n, StatusDisabled
	}
	return n, StatusOK
}

// act is the request for one action of a node that has it.
func (m *Model) act(ref NodeRef, need PatternID, action ui.SemanticsAction) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !m.Patterns(n).Has(need) || n.Actions&action == 0 {
		return nil, StatusNotSupported
	}
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: action}}, StatusOK
}

// Invoke is IInvokeProvider::Invoke: the framework taps the node.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Invoke
func (m *Model) Invoke(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	return m.act(ref, PatternInvoke, ui.SemanticsActionTap)
}

// Toggle is IToggleProvider::Toggle: the framework taps the node.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Toggle
func (m *Model) Toggle(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	return m.act(ref, PatternToggle, ui.SemanticsActionTap)
}

// Expand is IExpandCollapseProvider::Expand.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Expand
func (m *Model) Expand(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	return m.act(ref, PatternExpandCollapse, ui.SemanticsActionExpand)
}

// Collapse is IExpandCollapseProvider::Collapse.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Collapse
func (m *Model) Collapse(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	return m.act(ref, PatternExpandCollapse, ui.SemanticsActionCollapse)
}

// Focus is IRawElementProviderFragment::SetFocus: the framework focuses the node.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::SetFocus
func (m *Model) Focus(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if n.Actions&ui.SemanticsActionFocus == 0 {
		return nil, StatusNotSupported
	}
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionFocus}}, StatusOK
}

// Select is ISelectionItemProvider::Select: the framework taps the node when it is not selected.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Select
func (m *Model) Select(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if isSelected(n) {
		return nil, StatusOK
	}
	return m.act(ref, PatternSelectionItem, ui.SemanticsActionTap)
}

// AddToSelection is ISelectionItemProvider::AddToSelection, as Select (the selection is single).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::AddToSelection
func (m *Model) AddToSelection(ref NodeRef) ([]ui.SemanticsActionEvent, Status) { return m.Select(ref) }

// RemoveFromSelection is ISelectionItemProvider::RemoveFromSelection: the framework taps a node that is
// selected.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::RemoveFromSelection
func (m *Model) RemoveFromSelection(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !isSelected(n) {
		return nil, StatusOK
	}
	return m.act(ref, PatternSelectionItem, ui.SemanticsActionTap)
}

// SetValue is IValueProvider::SetValue: the text of the field becomes text. The framework offers the
// action to set the text only while the field has the focus, so a field that has not got it is focused
// first and the text is set when the next update shows the action (Apply returns it as a ChangeAction).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::SetValue
func (m *Model) SetValue(ref NodeRef, text string) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !m.Patterns(n).Has(PatternValue) || n.Flags.IsReadOnly {
		return nil, StatusNotSupported
	}
	if n.Actions&ui.SemanticsActionSetText != 0 {
		delete(m.pendingText, ref)
		return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionSetText, Arguments: text}}, StatusOK
	}
	if n.Actions&ui.SemanticsActionFocus == 0 {
		return nil, StatusNotSupported
	}
	m.pendingText[ref] = text
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionFocus}}, StatusOK
}

// SetRangeValue is IRangeValueProvider::SetValue: the framework is asked to take the value one step
// nearer to the one asked for, as the AT-SPI2 bridge does (a node can only increase and decrease, and a
// second request before the framework rebuilds would repeat the first).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::SetValue
func (m *Model) SetRangeValue(ref NodeRef, want float64) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	r, ok := rangeOf(n)
	if !ok {
		return nil, StatusNotSupported
	}
	if math.IsNaN(want) || want < r.min || want > r.max {
		return nil, StatusInvalid
	}
	switch {
	case want == r.current:
		return nil, StatusOK
	case want > r.current:
		return m.act(ref, PatternRangeValue, ui.SemanticsActionIncrease)
	}
	return m.act(ref, PatternRangeValue, ui.SemanticsActionDecrease)
}

// Scroll is IScrollProvider::Scroll: a vertical increment asks the framework for the scroll up action
// (the content moves up, which shows what is further down) and a decrement for scroll down; a
// horizontal increment is scroll left and a decrement scroll right, as TalkBack's forward and backward
// are in Flutter's Android embedder. A large amount and a small one are the same single action.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Scroll
func (m *Model) Scroll(ref NodeRef, horizontal, vertical int) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !m.Patterns(n).Has(PatternScroll) {
		return nil, StatusNotSupported
	}
	if horizontal < ScrollLargeDecrement || horizontal > ScrollSmallIncrement || vertical < ScrollLargeDecrement || vertical > ScrollSmallIncrement {
		return nil, StatusInvalid
	}
	var events []ui.SemanticsActionEvent
	step := func(amount int, vert bool, inc, dec ui.SemanticsAction) Status {
		if amount == ScrollNoAmount {
			return StatusOK
		}
		if !hasScroll(n, vert) {
			return StatusNotSupported
		}
		act := inc
		if amount == ScrollLargeDecrement || amount == ScrollSmallDecrement {
			act = dec
		}
		if n.Actions&act == 0 {
			return StatusOK // already at that end
		}
		events = append(events, ui.SemanticsActionEvent{ViewID: n.View, NodeID: n.ID, Type: act})
		return StatusOK
	}
	if st := step(vertical, true, ui.SemanticsActionScrollUp, ui.SemanticsActionScrollDown); st != StatusOK {
		return nil, st
	}
	if st := step(horizontal, false, ui.SemanticsActionScrollLeft, ui.SemanticsActionScrollRight); st != StatusOK {
		return nil, st
	}
	return events, StatusOK
}

// SetScrollPercent is IScrollProvider::SetScrollPercent: the framework scrolls to the offset (the
// scroll to offset action, which the framework offers on a node that allows implicit scrolling). A
// percentage of NoScroll (-1) leaves an axis alone.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::SetScrollPercent
func (m *Model) SetScrollPercent(ref NodeRef, horizontal, vertical float64) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if !m.Patterns(n).Has(PatternScroll) || n.Actions&ui.SemanticsActionScrollToOffset == 0 {
		return nil, StatusNotSupported
	}
	valid := func(p float64, vert bool) bool {
		if p == NoScroll {
			return true
		}
		return !math.IsNaN(p) && p >= 0 && p <= 100 && hasScroll(n, vert)
	}
	if !valid(horizontal, false) || !valid(vertical, true) {
		return nil, StatusInvalid
	}
	target := func(p float64, current float64) float64 {
		if p == NoScroll || math.IsNaN(n.ScrollExtentMax) || math.IsNaN(n.ScrollExtentMin) {
			return current
		}
		return n.ScrollExtentMin + p/100*(n.ScrollExtentMax-n.ScrollExtentMin)
	}
	pos := n.ScrollPosition
	if math.IsNaN(pos) {
		pos = 0
	}
	var dx, dy float64
	if hasScroll(n, false) {
		dx = target(horizontal, pos)
	}
	if hasScroll(n, true) {
		dy = target(vertical, pos)
	}
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionScrollToOffset, Arguments: []float64{dx, dy}}}, StatusOK
}

// ShowOnScreen is ITextRangeProvider::ScrollIntoView and IScrollItemProvider::ScrollIntoView: the
// framework shows the node on the screen.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::ScrollIntoView
func (m *Model) ShowOnScreen(ref NodeRef) ([]ui.SemanticsActionEvent, Status) {
	n, st := m.request(ref)
	if st != StatusOK {
		return nil, st
	}
	if n.Actions&ui.SemanticsActionShowOnScreen == 0 {
		return nil, StatusNotSupported
	}
	return []ui.SemanticsActionEvent{{ViewID: n.View, NodeID: n.ID, Type: ui.SemanticsActionShowOnScreen}}, StatusOK
}
