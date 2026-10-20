package atspi

import (
	"strconv"
	"strings"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/glutter/ui"
)

const (
	ifaceValue     = "org.a11y.atspi.Value"
	ifaceSelection = "org.a11y.atspi.Selection"
)

// numericValue is the value of a slider, a progress bar or a spin button as the Value
// interface shows it.
type numericValue struct {
	current, min, max, step float64
}

// valueOf is the numeric value of a node that has the Value interface: a slider, a spin
// button or a progress bar. The framework gives the value as the text that is said ("50%",
// "3"): a percentage goes from 0 to 100; another number goes between the minimum and
// the maximum that the node says, or from 0 to 100 when it says none. The step is a
// hundredth of the range, at most 1.
func valueOf(n *Node) (numericValue, bool) {
	switch {
	case n.Flags.IsSlider, n.Role == ui.SemanticsRoleSpinButton, n.Role == ui.SemanticsRoleProgressBar:
	default:
		return numericValue{}, false
	}
	v := numericValue{min: 0, max: 100}
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
	v.step = min(1, (v.max-v.min)/100)
	if v.step <= 0 {
		v.step = 1
	}
	return v, true
}

// hasSelection says whether the node has the Selection interface: it has children that can
// be selected (they say whether they are, with the flag IsSelected).
func hasSelection(m *Model, n *Node) bool {
	for _, c := range m.Children(n) {
		if m.Node(c).Flags.IsSelected != ui.TristateNone {
			return true
		}
	}
	return false
}

// selectionObject is what is exported under org.a11y.atspi.Selection.
type selectionObject struct{ o *object }

// kids returns the node and the children that can be selected, in order.
func (s selectionObject) kids() (*Node, []*Node, *godbus.Error) {
	n, err := s.o.node()
	if err != nil {
		return nil, nil, err
	}
	var kids []*Node
	for _, c := range s.o.b.model.Children(n) {
		kids = append(kids, s.o.b.model.Node(c))
	}
	return n, kids, nil
}

func (s selectionObject) selected(kids []*Node) []*Node {
	var out []*Node
	for _, k := range kids {
		if k.Flags.IsSelected == ui.TristateIsTrue {
			out = append(out, k)
		}
	}
	return out
}

// toggle asks the framework to tap the child, which selects or deselects it.
func (s selectionObject) toggle(k *Node) bool {
	if k.Actions&ui.SemanticsActionTap == 0 {
		return false
	}
	s.o.b.perform(ui.SemanticsActionEvent{ViewID: k.View, NodeID: k.ID, Type: ui.SemanticsActionTap})
	return true
}

// GetSelectedChild is Selection.GetSelectedChild.
func (s selectionObject) GetSelectedChild(selectedIndex int32) (objRef, *godbus.Error) {
	s.o.b.mu.Lock()
	defer s.o.b.mu.Unlock()
	_, kids, err := s.kids()
	if err != nil {
		return objRef{}, err
	}
	sel := s.selected(kids)
	if selectedIndex < 0 || int(selectedIndex) >= len(sel) {
		return objRef{}, invalidArgs("no such selected child")
	}
	return s.o.objRef(sel[selectedIndex].Ref()), nil
}

// SelectChild is Selection.SelectChild: it taps the child that is not selected.
func (s selectionObject) SelectChild(childIndex int32) (bool, *godbus.Error) {
	s.o.b.mu.Lock()
	defer s.o.b.mu.Unlock()
	_, kids, err := s.kids()
	if err != nil {
		return false, err
	}
	if childIndex < 0 || int(childIndex) >= len(kids) {
		return false, invalidArgs("no such child")
	}
	if kids[childIndex].Flags.IsSelected == ui.TristateIsTrue {
		return true, nil
	}
	return s.toggle(kids[childIndex]), nil
}

// DeselectChild is Selection.DeselectChild: it taps the child that is selected.
func (s selectionObject) DeselectChild(childIndex int32) (bool, *godbus.Error) {
	s.o.b.mu.Lock()
	defer s.o.b.mu.Unlock()
	_, kids, err := s.kids()
	if err != nil {
		return false, err
	}
	if childIndex < 0 || int(childIndex) >= len(kids) {
		return false, invalidArgs("no such child")
	}
	if kids[childIndex].Flags.IsSelected != ui.TristateIsTrue {
		return true, nil
	}
	return s.toggle(kids[childIndex]), nil
}

// DeselectSelectedChild is Selection.DeselectSelectedChild.
func (s selectionObject) DeselectSelectedChild(selectedIndex int32) (bool, *godbus.Error) {
	s.o.b.mu.Lock()
	defer s.o.b.mu.Unlock()
	_, kids, err := s.kids()
	if err != nil {
		return false, err
	}
	sel := s.selected(kids)
	if selectedIndex < 0 || int(selectedIndex) >= len(sel) {
		return false, invalidArgs("no such selected child")
	}
	return s.toggle(sel[selectedIndex]), nil
}

// IsChildSelected is Selection.IsChildSelected.
func (s selectionObject) IsChildSelected(childIndex int32) (bool, *godbus.Error) {
	s.o.b.mu.Lock()
	defer s.o.b.mu.Unlock()
	_, kids, err := s.kids()
	if err != nil {
		return false, err
	}
	if childIndex < 0 || int(childIndex) >= len(kids) {
		return false, invalidArgs("no such child")
	}
	return kids[childIndex].Flags.IsSelected == ui.TristateIsTrue, nil
}

// SelectAll is Selection.SelectAll: not supported.
func (s selectionObject) SelectAll() (bool, *godbus.Error) { return false, nil }

// ClearSelection is Selection.ClearSelection: not supported.
func (s selectionObject) ClearSelection() (bool, *godbus.Error) { return false, nil }
