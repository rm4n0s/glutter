package atspi

import (
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func mat(s, dx, dy float32) [16]float32 {
	return [16]float32{s, 0, 0, 0, 0, s, 0, 0, 0, 0, 1, 0, dx, dy, 0, 1}
}

func nu(id int, children []int, edit func(n *ui.SemanticsNodeUpdate)) ui.SemanticsNodeUpdate {
	n := ui.SemanticsNodeUpdate{ID: id, Rect: ui.Rect{Right: 100, Bottom: 50}, Transform: mat(1, 0, 0), ChildrenInTraversalOrder: children}
	if edit != nil {
		edit(&n)
	}
	return n
}

func kinds(changes []Change) []ChangeKind {
	var out []ChangeKind
	for _, c := range changes {
		out = append(out, c.Kind)
	}
	return out
}

func TestModelApply(t *testing.T) {
	m := NewModel()
	changes := m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, []int{1, 2}, nil), nu(1, nil, nil), nu(2, []int{3}, nil), nu(3, nil, nil)}})
	want := []Change{
		{Kind: ChangeChildAdded, Parent: appRef, Child: NodeRef{0, 0}, Index: 0},
		{Kind: ChangeNodeAdded, Ref: NodeRef{0, 0}}, {Kind: ChangeNodeAdded, Ref: NodeRef{0, 1}},
		{Kind: ChangeNodeAdded, Ref: NodeRef{0, 2}}, {Kind: ChangeNodeAdded, Ref: NodeRef{0, 3}},
	}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("first update:\n got %+v\nwant %+v", changes, want)
	}
	if n := m.Node(NodeRef{0, 3}); n == nil || n.Parent != 2 {
		t.Errorf("node 3 = %+v", n)
	}
	if got := m.Children(m.Node(NodeRef{0, 0})); !reflect.DeepEqual(got, []NodeRef{{0, 1}, {0, 2}}) {
		t.Errorf("children = %v", got)
	}
	if m.Node(NodeRef{0, 99}) != nil || m.Node(NodeRef{5, 0}) != nil {
		t.Error("a node that is not there")
	}

	// Moving node 3 under the root removes it from node 2; the order of the children changes.
	changes = m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, []int{2, 1, 3}, nil), nu(2, nil, nil)}})
	if got := kinds(changes); !reflect.DeepEqual(got, []ChangeKind{ChangeChildAdded, ChangeChildRemoved}) {
		t.Errorf("reparent kinds = %v: %+v", got, changes)
	}
	if n := m.Node(NodeRef{0, 3}); n.Parent != 0 {
		t.Errorf("parent of 3 = %d", n.Parent)
	}

	// A node that no parent has any more is gone, and so is what is under it.
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(2, []int{4}, nil), nu(4, nil, nil)}})
	changes = m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, []int{1, 3}, nil)}})
	var removed []int
	for _, c := range changes {
		if c.Kind == ChangeNodeRemoved {
			removed = append(removed, c.Ref.ID)
		}
	}
	if !reflect.DeepEqual(removed, []int{2, 4}) {
		t.Errorf("removed %v, want [2 4]: %+v", removed, changes)
	}
	if m.Node(NodeRef{0, 4}) != nil {
		t.Error("node 4 is still there")
	}
}

func TestModelStatesAndNames(t *testing.T) {
	m := NewModel()
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, []int{1}, nil), nu(1, nil, func(n *ui.SemanticsNodeUpdate) { n.Label = "a" })}})
	changes := m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(1, nil, func(n *ui.SemanticsNodeUpdate) {
		n.Label, n.Hint = "b", "h"
		n.Flags.IsSelected = ui.TristateIsTrue
	})}})
	want := []Change{
		{Kind: ChangeState, Ref: NodeRef{0, 1}, State: StateSelectable, On: true},
		{Kind: ChangeState, Ref: NodeRef{0, 1}, State: StateSelected, On: true},
		{Kind: ChangeProperty, Ref: NodeRef{0, 1}, Property: "accessible-name", Value: "b"},
		{Kind: ChangeProperty, Ref: NodeRef{0, 1}, Property: "accessible-description", Value: "h"},
	}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("changes:\n got %+v\nwant %+v", changes, want)
	}
	if changes := m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(1, nil, func(n *ui.SemanticsNodeUpdate) {
		n.Label, n.Hint = "b", "h"
		n.Flags.IsSelected = ui.TristateIsTrue
	})}}); len(changes) != 0 {
		t.Errorf("the same update changed %+v", changes)
	}
}

func TestModelWindow(t *testing.T) {
	m := NewModel()
	// The title and the focus can arrive before the tree.
	if got := m.SetWindow(0, "Title", true); len(got) != 0 {
		t.Errorf("changes without a tree: %+v", got)
	}
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, nil, nil)}})
	root := m.Node(NodeRef{0, 0})
	if got := m.Name(root); got != "Title" {
		t.Errorf("the frame is named %q", got)
	}
	if !m.States(root).Has(StateActive) {
		t.Error("the frame is not active")
	}
	if got := m.Role(root); got != RoleFrame {
		t.Errorf("role = %v", got)
	}
	changes := m.SetWindow(0, "New", false)
	want := []Change{
		{Kind: ChangeProperty, Ref: NodeRef{0, 0}, Property: "accessible-name", Value: "New"},
		{Kind: ChangeState, Ref: NodeRef{0, 0}, State: StateActive, On: false},
		{Kind: ChangeWindow, Ref: NodeRef{0, 0}, On: false},
	}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("changes = %+v", changes)
	}
	// A label of the root beats the title.
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, nil, func(n *ui.SemanticsNodeUpdate) { n.Label = "Own" })}})
	if got := m.Name(m.Node(NodeRef{0, 0})); got != "Own" {
		t.Errorf("name = %q", got)
	}
}

func TestModelRemoveView(t *testing.T) {
	m := NewModel()
	if got := m.RemoveView(4); got != nil {
		t.Errorf("removing a view that is not there changed %+v", got)
	}
	m.Apply(4, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, []int{1}, nil), nu(1, nil, nil)}})
	m.Apply(2, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(0, nil, nil)}})
	if got := m.Views(); !reflect.DeepEqual(got, []ui.ViewID{2, 4}) {
		t.Errorf("views = %v", got)
	}
	changes := m.RemoveView(4)
	want := []Change{
		{Kind: ChangeChildRemoved, Parent: appRef, Child: NodeRef{4, 0}, Index: 1},
		{Kind: ChangeNodeRemoved, Ref: NodeRef{4, 0}}, {Kind: ChangeNodeRemoved, Ref: NodeRef{4, 1}},
	}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("changes = %+v", changes)
	}
	if got := m.Views(); !reflect.DeepEqual(got, []ui.ViewID{2}) {
		t.Errorf("views = %v", got)
	}
}

func TestModelExtents(t *testing.T) {
	m := NewModel()
	rotate := [16]float32{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} // 90 degrees
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		nu(0, []int{1}, func(n *ui.SemanticsNodeUpdate) { n.Transform = mat(2, 5, 7); n.Rect = ui.Rect{Right: 500, Bottom: 500} }),
		nu(1, []int{2}, func(n *ui.SemanticsNodeUpdate) {
			n.Transform = mat(1, 10, 20)
			n.Rect = ui.Rect{Right: 200, Bottom: 100}
		}),
		nu(2, nil, func(n *ui.SemanticsNodeUpdate) { n.Transform = rotate; n.Rect = ui.Rect{Right: 30, Bottom: 10} }),
	}})
	tests := []struct {
		id   int
		want Extents
	}{
		{0, Extents{5, 7, 1000, 1000}},
		{1, Extents{25, 47, 400, 200}},
		// The child is turned a quarter: 30x10 becomes 10x30 at x in [-10, 0] from its origin.
		{2, Extents{5, 47, 20, 60}},
	}
	for _, tt := range tests {
		if got := m.Extents(m.Node(NodeRef{0, tt.id})); got != tt.want {
			t.Errorf("node %d: extents = %+v, want %+v", tt.id, got, tt.want)
		}
	}
	if !tests[1].want.Contains(25, 47) || tests[1].want.Contains(425, 47) || tests[1].want.Contains(24, 47) {
		t.Error("Contains is wrong at the edges")
	}
	// A rectangle that is not a number has no extents.
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{nu(2, nil, func(n *ui.SemanticsNodeUpdate) { n.Rect = ui.Rect{Left: math.NaN()} })}})
	if got := m.Extents(m.Node(NodeRef{0, 2})); got != (Extents{}) {
		t.Errorf("extents of a NaN rectangle = %+v", got)
	}
}

func TestModelHitTest(t *testing.T) {
	m := NewModel()
	m.Apply(0, &ui.SemanticsUpdate{Nodes: []ui.SemanticsNodeUpdate{
		nu(0, []int{1, 2, 3}, func(n *ui.SemanticsNodeUpdate) { n.Rect = ui.Rect{Right: 300, Bottom: 100} }),
		nu(1, nil, nil), // 0..100
		nu(2, nil, func(n *ui.SemanticsNodeUpdate) { n.Transform = mat(1, 50, 0) }), // 50..150, over 1
		nu(3, nil, func(n *ui.SemanticsNodeUpdate) { n.Transform = mat(1, 200, 0); n.Flags.IsHidden = true }),
	}})
	root := m.Node(NodeRef{0, 0})
	tests := []struct {
		x, y int32
		want int // -1 for nobody
	}{
		{10, 10, 1}, {60, 10, 2}, {120, 10, 2}, {160, 10, 0}, {250, 10, 0}, {10, 80, 0}, {400, 10, -1}, {10, -5, -1},
	}
	for _, tt := range tests {
		got := -1
		if hit := m.HitTest(root, tt.x, tt.y); hit != nil {
			got = hit.ID
		}
		if got != tt.want {
			t.Errorf("(%d, %d): hit %d, want %d", tt.x, tt.y, got, tt.want)
		}
	}
}
