package uia

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/ui"
)

func TestPatternSelection(t *testing.T) {
	tests := []struct {
		name string
		edit func(n *ui.SemanticsNodeUpdate)
		want []PatternID
	}{
		{"a plain label has none", func(n *ui.SemanticsNodeUpdate) { n.Label = "x" }, nil},
		{"a button with tap", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true; n.Actions = ui.SemanticsActionTap }, []PatternID{PatternInvoke}},
		{"a tap on a disabled button is still invoke", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsButton = true
			n.Flags.IsEnabled = ui.TristateIsFalse
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternInvoke}},
		{"a check box is toggle", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsFalse
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternToggle}},
		{"a switch is toggle", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsToggled = ui.TristateIsTrue
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternToggle}},
		{"a radio button is a selection item", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Flags.IsInMutuallyExclusiveGroup = true
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternSelectionItem}},
		{"a selected list tile is a selection item and can be invoked", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsSelected = ui.TristateIsFalse
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternInvoke, PatternSelectionItem}},
		{"a text field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, []PatternID{PatternValue, PatternText}},
		{"a text field you can tap is not invoked", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsTextField = true
			n.Actions = ui.SemanticsActionTap
		}, []PatternID{PatternValue, PatternText}},
		{"a slider", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "5%" }, []PatternID{PatternRangeValue}},
		{"a progress bar with a value", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleProgressBar; n.Value = "30%" }, []PatternID{PatternRangeValue}},
		{"a progress bar with none", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleProgressBar }, nil},
		{"a spin button", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleSpinButton; n.Value = "3" }, []PatternID{PatternRangeValue}},
		{"a scrolling list", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionScrollDown }, []PatternID{PatternScroll}},
		{"a horizontal scroll", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionScrollLeft }, []PatternID{PatternScroll}},
		{"an expanded tree item", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsExpanded = ui.TristateIsTrue }, []PatternID{PatternExpandCollapse}},
		{"an expand action", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionExpand }, []PatternID{PatternExpandCollapse}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n := nodeWith(t, tt.edit)
			got := m.Patterns(n).List()
			if !slices.Equal(got, tt.want) {
				t.Errorf("patterns = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectionContainerPattern(t *testing.T) {
	a := semtree.N(1, "a", 0, 0, 1, 1)
	a.Flags.IsSelected = ui.TristateIsTrue
	b := semtree.N(2, "b", 0, 0, 1, 1)
	b.Flags.IsSelected = ui.TristateIsFalse
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1, 2), a, b))
	if !m.Patterns(m.Root(1)).Has(PatternSelection) {
		t.Fatal("a node whose children have the selected flag is a selection container")
	}
	if got := m.SelectedChildren(m.Root(1)); !slices.Equal(got, []NodeRef{{1, 1}}) {
		t.Errorf("selected = %v, want node 1", got)
	}
	if c := m.SelectionContainer(m.Node(NodeRef{1, 2})); c == nil || c.ID != 0 {
		t.Errorf("container = %v, want the root", c)
	}
	// Select taps the child that is not selected, and a selected one is already so.
	b.Actions = ui.SemanticsActionTap
	m.Apply(1, semtree.Update(b))
	if ev, st := m.Select(NodeRef{1, 2}); st != StatusOK || len(ev) != 1 || ev[0].Type != ui.SemanticsActionTap {
		t.Errorf("Select = %v, %v, want a tap", ev, st)
	}
	if ev, st := m.Select(NodeRef{1, 1}); st != StatusOK || len(ev) != 0 {
		t.Errorf("Select of a selected node = %v, %v, want nothing to do", ev, st)
	}
	if ev, st := m.RemoveFromSelection(NodeRef{1, 2}); st != StatusOK || len(ev) != 0 {
		t.Errorf("RemoveFromSelection of a node that is not selected = %v, %v", ev, st)
	}
}

func TestRequests(t *testing.T) {
	button := semtree.N(1, "Press", 0, 0, 1, 1)
	button.Flags.IsButton = true
	button.Actions = ui.SemanticsActionTap
	off := semtree.N(2, "Off", 0, 0, 1, 1)
	off.Flags.IsButton = true
	off.Flags.IsEnabled = ui.TristateIsFalse
	off.Actions = ui.SemanticsActionTap
	check := semtree.N(3, "Agree", 0, 0, 1, 1)
	check.Flags.IsChecked = ui.CheckedStateIsFalse
	check.Actions = ui.SemanticsActionTap
	label := semtree.N(4, "Label", 0, 0, 1, 1)
	tree := semtree.N(5, "Branch", 0, 0, 1, 1)
	tree.Flags.IsExpanded = ui.TristateIsFalse
	tree.Actions = ui.SemanticsActionExpand
	field := semtree.N(6, "Name", 0, 0, 1, 1)
	field.Flags.IsTextField = true
	field.Actions = ui.SemanticsActionFocus
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1, 2, 3, 4, 5, 6), button, off, check, label, tree, field))
	tap := func(id int) []ui.SemanticsActionEvent {
		return []ui.SemanticsActionEvent{{ViewID: 1, NodeID: id, Type: ui.SemanticsActionTap}}
	}
	tests := []struct {
		name   string
		do     func() ([]ui.SemanticsActionEvent, Status)
		want   []ui.SemanticsActionEvent
		status Status
	}{
		{"invoke", func() ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 1}) }, tap(1), StatusOK},
		{"invoke a disabled button", func() ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 2}) }, nil, StatusDisabled},
		{"invoke a label", func() ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 4}) }, nil, StatusNotSupported},
		{"invoke a node that is gone", func() ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{1, 99}) }, nil, StatusGone},
		{"invoke in a view that is gone", func() ([]ui.SemanticsActionEvent, Status) { return m.Invoke(NodeRef{7, 1}) }, nil, StatusGone},
		{"toggle", func() ([]ui.SemanticsActionEvent, Status) { return m.Toggle(NodeRef{1, 3}) }, tap(3), StatusOK},
		{"toggle a button", func() ([]ui.SemanticsActionEvent, Status) { return m.Toggle(NodeRef{1, 1}) }, nil, StatusNotSupported},
		{"expand", func() ([]ui.SemanticsActionEvent, Status) { return m.Expand(NodeRef{1, 5}) }, []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 5, Type: ui.SemanticsActionExpand}}, StatusOK},
		{"collapse what cannot collapse", func() ([]ui.SemanticsActionEvent, Status) { return m.Collapse(NodeRef{1, 5}) }, nil, StatusNotSupported},
		{"focus", func() ([]ui.SemanticsActionEvent, Status) { return m.Focus(NodeRef{1, 6}) }, []ui.SemanticsActionEvent{{ViewID: 1, NodeID: 6, Type: ui.SemanticsActionFocus}}, StatusOK},
		{"focus what cannot take it", func() ([]ui.SemanticsActionEvent, Status) { return m.Focus(NodeRef{1, 4}) }, nil, StatusNotSupported},
		{"show on screen without the action", func() ([]ui.SemanticsActionEvent, Status) { return m.ShowOnScreen(NodeRef{1, 4}) }, nil, StatusNotSupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, st := tt.do()
			if st != tt.status || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, %v; want %+v, %v", got, st, tt.want, tt.status)
			}
		})
	}
}

func TestSetValueWaitsForTheFieldToTakeTheFocus(t *testing.T) {
	field := semtree.N(1, "Name", 0, 0, 1, 1)
	field.Flags.IsTextField = true
	field.Flags.IsFocused = ui.TristateIsFalse
	field.Actions = ui.SemanticsActionFocus
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1), field))
	ref := NodeRef{1, 1}

	ev, st := m.SetValue(ref, "new")
	if st != StatusOK || len(ev) != 1 || ev[0].Type != ui.SemanticsActionFocus {
		t.Fatalf("SetValue on a field without the action = %v, %v, want the focus first", ev, st)
	}
	// An update that still lacks the action does nothing.
	if got := find(m.Apply(1, semtree.Update(field)), ChangeAction); len(got) != 0 {
		t.Fatalf("the action came before the field offered it: %+v", got)
	}
	// The field has the focus and offers the action: the text is set.
	field.Flags.IsFocused = ui.TristateIsTrue
	field.Actions = ui.SemanticsActionFocus | ui.SemanticsActionSetText
	got := find(m.Apply(1, semtree.Update(field)), ChangeAction)
	want := ui.SemanticsActionEvent{ViewID: 1, NodeID: 1, Type: ui.SemanticsActionSetText, Arguments: "new"}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Action, want) {
		t.Fatalf("actions = %+v, want %+v", got, want)
	}
	if again := find(m.Apply(1, semtree.Update(field)), ChangeAction); len(again) != 0 {
		t.Error("the text was set twice")
	}
	// With the action there, it goes at once.
	if ev, st := m.SetValue(ref, "now"); st != StatusOK || len(ev) != 1 || ev[0].Type != ui.SemanticsActionSetText || ev[0].Arguments != "now" {
		t.Errorf("SetValue = %v, %v", ev, st)
	}
	// A read-only field refuses.
	field.Flags.IsReadOnly = true
	m.Apply(1, semtree.Update(field))
	if _, st := m.SetValue(ref, "x"); st != StatusNotSupported {
		t.Errorf("SetValue on a read-only field = %v, want not supported", st)
	}
	// A pending text dies with its node.
	field.Flags.IsReadOnly = false
	field.Actions = ui.SemanticsActionFocus
	m.Apply(1, semtree.Update(field))
	m.SetValue(ref, "lost")
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10)))
	if len(m.pendingText) != 0 {
		t.Errorf("pending text = %v after the node went", m.pendingText)
	}
}

func TestSetRangeValueSteps(t *testing.T) {
	slider := semtree.N(1, "Level", 0, 0, 1, 1)
	slider.Flags.IsSlider = true
	slider.Value = "50%"
	slider.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
	locked := semtree.N(2, "Locked", 0, 0, 1, 1)
	locked.Flags.IsSlider = true
	locked.Value = "50%"
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1, 2), slider, locked))
	tests := []struct {
		name   string
		ref    NodeRef
		want   float64
		action ui.SemanticsAction
		status Status
	}{
		{"above", NodeRef{1, 1}, 75, ui.SemanticsActionIncrease, StatusOK},
		{"below", NodeRef{1, 1}, 25, ui.SemanticsActionDecrease, StatusOK},
		{"the same value", NodeRef{1, 1}, 50, 0, StatusOK},
		{"just above", NodeRef{1, 1}, 50.0001, ui.SemanticsActionIncrease, StatusOK},
		{"over the maximum", NodeRef{1, 1}, 101, 0, StatusInvalid},
		{"under the minimum", NodeRef{1, 1}, -1, 0, StatusInvalid},
		{"a slider that cannot move", NodeRef{1, 2}, 60, 0, StatusNotSupported},
		{"a node that is not a range", NodeRef{1, 0}, 60, 0, StatusNotSupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, st := m.SetRangeValue(tt.ref, tt.want)
			if st != tt.status {
				t.Fatalf("status = %v, want %v", st, tt.status)
			}
			if tt.action == 0 {
				if len(ev) != 0 {
					t.Errorf("events = %v, want none", ev)
				}
				return
			}
			if len(ev) != 1 || ev[0].Type != tt.action {
				t.Errorf("events = %v, want one %v", ev, tt.action)
			}
		})
	}
}

func TestScrollRequests(t *testing.T) {
	list := semtree.N(1, "List", 0, 0, 100, 100)
	list.Actions = ui.SemanticsActionScrollUp | ui.SemanticsActionScrollDown | ui.SemanticsActionScrollToOffset
	list.ScrollPosition, list.ScrollExtentMin, list.ScrollExtentMax = 40, 0, 200
	wide := semtree.N(2, "Wide", 0, 0, 100, 100)
	wide.Actions = ui.SemanticsActionScrollLeft
	wide.ScrollPosition, wide.ScrollExtentMin, wide.ScrollExtentMax = 0, 0, 200
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10, 1, 2), list, wide))
	act := func(id int, a ui.SemanticsAction) ui.SemanticsActionEvent {
		return ui.SemanticsActionEvent{ViewID: 1, NodeID: id, Type: a}
	}
	tests := []struct {
		name   string
		ref    NodeRef
		h, v   int
		want   []ui.SemanticsActionEvent
		status Status
	}{
		{"down by a small amount", NodeRef{1, 1}, ScrollNoAmount, ScrollSmallIncrement, []ui.SemanticsActionEvent{act(1, ui.SemanticsActionScrollUp)}, StatusOK},
		{"down by a large amount", NodeRef{1, 1}, ScrollNoAmount, ScrollLargeIncrement, []ui.SemanticsActionEvent{act(1, ui.SemanticsActionScrollUp)}, StatusOK},
		{"up", NodeRef{1, 1}, ScrollNoAmount, ScrollSmallDecrement, []ui.SemanticsActionEvent{act(1, ui.SemanticsActionScrollDown)}, StatusOK},
		{"no amount", NodeRef{1, 1}, ScrollNoAmount, ScrollNoAmount, nil, StatusOK},
		{"right", NodeRef{1, 2}, ScrollSmallIncrement, ScrollNoAmount, []ui.SemanticsActionEvent{act(2, ui.SemanticsActionScrollLeft)}, StatusOK},
		{"back left, where the node cannot go", NodeRef{1, 2}, ScrollSmallDecrement, ScrollNoAmount, nil, StatusOK},
		{"the axis the node does not scroll", NodeRef{1, 1}, ScrollSmallIncrement, ScrollNoAmount, nil, StatusNotSupported},
		{"an amount out of range", NodeRef{1, 1}, 9, ScrollNoAmount, nil, StatusInvalid},
		{"not a scroller", NodeRef{1, 0}, ScrollNoAmount, ScrollSmallIncrement, nil, StatusNotSupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, st := m.Scroll(tt.ref, tt.h, tt.v)
			if st != tt.status || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, %v; want %+v, %v", got, st, tt.want, tt.status)
			}
		})
	}
	percent := []struct {
		name   string
		ref    NodeRef
		h, v   float64
		want   []float64
		status Status
	}{
		{"to the middle", NodeRef{1, 1}, NoScroll, 50, []float64{0, 100}, StatusOK},
		{"to the end", NodeRef{1, 1}, NoScroll, 100, []float64{0, 200}, StatusOK},
		{"a fraction", NodeRef{1, 1}, NoScroll, 37.5, []float64{0, 75}, StatusOK},
		{"over 100", NodeRef{1, 1}, NoScroll, 101, nil, StatusInvalid},
		{"an axis it does not scroll", NodeRef{1, 1}, 10, NoScroll, nil, StatusInvalid},
		{"no scroll to offset action", NodeRef{1, 2}, 10, NoScroll, nil, StatusNotSupported},
	}
	for _, tt := range percent {
		t.Run("percent "+tt.name, func(t *testing.T) {
			got, st := m.SetScrollPercent(tt.ref, tt.h, tt.v)
			if st != tt.status {
				t.Fatalf("status = %v, want %v", st, tt.status)
			}
			if tt.want == nil {
				return
			}
			if len(got) != 1 || got[0].Type != ui.SemanticsActionScrollToOffset || !reflect.DeepEqual(got[0].Arguments, tt.want) {
				t.Errorf("events = %+v, want an offset of %v", got, tt.want)
			}
		})
	}
}
