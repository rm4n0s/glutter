package uia

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/ui"
)

// kinds lists the kinds of the changes, in order.
func kinds(changes []Change) []ChangeKind {
	out := make([]ChangeKind, len(changes))
	for i, c := range changes {
		out[i] = c.Kind
	}
	return out
}

// find returns the changes of a kind.
func find(changes []Change, kind ChangeKind) []Change {
	var out []Change
	for _, c := range changes {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

func TestApplyAddsRemovesMovesAndReorders(t *testing.T) {
	root := func(kids ...int) ui.SemanticsNodeUpdate { return semtree.N(0, "", 0, 0, 100, 100, kids...) }
	leaf := func(id int) ui.SemanticsNodeUpdate { return semtree.N(id, "n", 0, 0, 10, 10) }
	tests := []struct {
		name      string
		first     []ui.SemanticsNodeUpdate
		second    []ui.SemanticsNodeUpdate
		wantKinds []ChangeKind
		wantNodes []int // the IDs the model has after the second update
		wantStruc int   // the structure change of the root, 0 when none
	}{
		{
			name:      "a child is added",
			first:     []ui.SemanticsNodeUpdate{root(1), leaf(1)},
			second:    []ui.SemanticsNodeUpdate{root(1, 2), leaf(2)},
			wantKinds: []ChangeKind{ChangeNodeAdded, ChangeStructure},
			wantNodes: []int{0, 1, 2},
			wantStruc: StructureChildrenInvalidated,
		},
		{
			name:      "a child is removed",
			first:     []ui.SemanticsNodeUpdate{root(1, 2), leaf(1), leaf(2)},
			second:    []ui.SemanticsNodeUpdate{root(1)},
			wantKinds: []ChangeKind{ChangeStructure, ChangeNodeRemoved},
			wantNodes: []int{0, 1},
			wantStruc: StructureChildrenInvalidated,
		},
		{
			name:      "the children change places",
			first:     []ui.SemanticsNodeUpdate{root(1, 2, 3), leaf(1), leaf(2), leaf(3)},
			second:    []ui.SemanticsNodeUpdate{root(3, 1, 2)},
			wantKinds: []ChangeKind{ChangeStructure},
			wantNodes: []int{0, 1, 2, 3},
			wantStruc: StructureChildrenReordered,
		},
		{
			name:      "a node moves to another parent",
			first:     []ui.SemanticsNodeUpdate{root(1, 2), semtree.N(1, "a", 0, 0, 10, 10, 3), leaf(2), leaf(3)},
			second:    []ui.SemanticsNodeUpdate{semtree.N(1, "a", 0, 0, 10, 10), semtree.N(2, "n", 0, 0, 10, 10, 3)},
			wantKinds: []ChangeKind{ChangeStructure, ChangeStructure},
			wantNodes: []int{0, 1, 2, 3},
		},
		{
			name:      "a node the root does not reach is gone",
			first:     []ui.SemanticsNodeUpdate{root(1), leaf(1)},
			second:    []ui.SemanticsNodeUpdate{root(), leaf(1), leaf(9)},
			wantKinds: []ChangeKind{ChangeStructure, ChangeNodeRemoved, ChangeNodeRemoved},
			wantNodes: []int{0},
			wantStruc: StructureChildrenInvalidated,
		},
		{
			name:      "an update that changes nothing says nothing",
			first:     []ui.SemanticsNodeUpdate{root(1), leaf(1)},
			second:    []ui.SemanticsNodeUpdate{root(1), leaf(1)},
			wantKinds: []ChangeKind{},
			wantNodes: []int{0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.Apply(1, semtree.Update(tt.first...))
			got := m.Apply(1, semtree.Update(tt.second...))
			if !slices.Equal(kinds(got), tt.wantKinds) {
				t.Errorf("kinds = %v, want %v", kinds(got), tt.wantKinds)
			}
			var ids []int
			for id := 0; id < 10; id++ {
				if m.Node(NodeRef{1, id}) != nil {
					ids = append(ids, id)
				}
			}
			if !slices.Equal(ids, tt.wantNodes) {
				t.Errorf("nodes = %v, want %v", ids, tt.wantNodes)
			}
			if tt.wantStruc != 0 {
				s := find(got, ChangeStructure)
				if len(s) == 0 || s[0].Structure != tt.wantStruc {
					t.Errorf("structure change = %+v, want kind %d", s, tt.wantStruc)
				}
			}
		})
	}
}

func TestApplyKeepsOneTreePerView(t *testing.T) {
	m := NewModel()
	first := m.Apply(1, semtree.Update(semtree.N(0, "one", 0, 0, 10, 10, 1), semtree.N(1, "a", 0, 0, 5, 5)))
	second := m.Apply(2, semtree.Update(semtree.N(0, "two", 0, 0, 10, 10)))
	if len(find(first, ChangeWindowOpened)) != 1 || len(find(second, ChangeWindowOpened)) != 1 {
		t.Errorf("each view's first tree opens its window: %v, %v", kinds(first), kinds(second))
	}
	if got := m.Views(); !slices.Equal(got, []ui.ViewID{1, 2}) {
		t.Errorf("views = %v, want [1 2]", got)
	}
	if n := m.Node(NodeRef{2, 1}); n != nil {
		t.Error("view 2 has the node 1 of view 1")
	}
	removed := m.RemoveView(1)
	if got := len(find(removed, ChangeNodeRemoved)); got != 2 {
		t.Errorf("removing view 1 removes %d nodes, want 2", got)
	}
	if m.Node(NodeRef{1, 0}) != nil || m.Node(NodeRef{2, 0}) == nil {
		t.Error("RemoveView took the wrong view")
	}
}

func TestPropertyChangesAreReported(t *testing.T) {
	m := NewModel()
	btn := semtree.N(1, "Press", 0, 0, 10, 10)
	btn.Flags.IsButton = true
	btn.Actions = ui.SemanticsActionTap
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1), btn))
	tests := []struct {
		name string
		edit func(n *ui.SemanticsNodeUpdate)
		prop PropertyID
		want Value
	}{
		{"the label", func(n *ui.SemanticsNodeUpdate) { n.Label = "Go" }, PropName, String("Go")},
		{"the tooltip", func(n *ui.SemanticsNodeUpdate) { n.Tooltip = "Tip" }, PropHelpText, String("Tip")},
		{"disabled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsEnabled = ui.TristateIsFalse }, PropIsEnabled, Bool(false)},
		{"hidden", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHidden = true }, PropIsOffscreen, Bool(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1), btn))
			edited := btn
			tt.edit(&edited)
			changes := find(m.Apply(1, semtree.Update(edited)), ChangeProperty)
			var got *Change
			for i := range changes {
				if changes[i].Property == tt.prop {
					got = &changes[i]
				}
			}
			if got == nil {
				t.Fatalf("no change of property %d in %+v", tt.prop, changes)
			}
			if got.New != tt.want {
				t.Errorf("new value = %+v, want %+v", got.New, tt.want)
			}
		})
	}
}

func TestValueRangeToggleExpandAndScrollChangesAreReported(t *testing.T) {
	slider := semtree.N(1, "Level", 0, 0, 10, 10)
	slider.Flags.IsSlider = true
	slider.Value = "50%"
	check := semtree.N(2, "Agree", 0, 0, 10, 10)
	check.Flags.IsChecked = ui.CheckedStateIsFalse
	check.Actions = ui.SemanticsActionTap
	tree := semtree.N(3, "Node", 0, 0, 10, 10)
	tree.Flags.IsExpanded = ui.TristateIsFalse
	list := semtree.N(4, "List", 0, 0, 100, 100)
	list.Actions = ui.SemanticsActionScrollUp
	list.ScrollPosition, list.ScrollExtentMin, list.ScrollExtentMax = 0, 0, 100
	field := semtree.N(5, "Name", 0, 0, 10, 10)
	field.Flags.IsTextField = true
	field.Value = "ab"
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1, 2, 3, 4, 5), slider, check, tree, list, field))

	slider.Value = "80%"
	check.Flags.IsChecked = ui.CheckedStateIsTrue
	tree.Flags.IsExpanded = ui.TristateIsTrue
	list.ScrollPosition = 50
	field.Value = "abc"
	field.TextSelectionBase, field.TextSelectionExtent = 3, 3
	changes := m.Apply(1, semtree.Update(slider, check, tree, list, field))

	want := map[PropertyID]Value{
		PropRangeValueValue:                   Float(80),
		PropToggleToggleState:                 Int(ToggleOn),
		PropExpandCollapseExpandCollapseState: Int(ExpandCollapseExpanded),
		PropScrollVerticalScrollPercent:       Float(50),
		PropValueValue:                        String("abc"),
	}
	got := map[PropertyID]Value{}
	for _, c := range find(changes, ChangeProperty) {
		got[c.Property] = c.New
	}
	// The slider's text is its name for no one: Name does not change; the range value does.
	for p, v := range want {
		if got[p] != v {
			t.Errorf("property %d = %+v, want %+v", p, got[p], v)
		}
	}
	if len(find(changes, ChangeTextChanged)) != 1 || len(find(changes, ChangeTextSelection)) != 1 {
		t.Errorf("text events: changed %d, selection %d, want 1 and 1", len(find(changes, ChangeTextChanged)), len(find(changes, ChangeTextSelection)))
	}
}

func TestFocusFollowsTheFlagAndTheWindow(t *testing.T) {
	a := semtree.N(1, "a", 0, 0, 10, 10)
	a.Flags.IsFocused = ui.TristateIsFalse
	b := semtree.N(2, "b", 0, 0, 10, 10)
	b.Flags.IsFocused = ui.TristateIsFalse
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1, 2), a, b))
	if n := m.Focused(1); n != nil {
		t.Errorf("focused = %v before anything has the focus", n.ID)
	}
	b.Flags.IsFocused = ui.TristateIsTrue
	if changes := m.Apply(1, semtree.Update(b)); len(find(changes, ChangeFocus)) != 0 {
		t.Error("an inactive window raised a focus event")
	}
	if n := m.Focused(1); n != nil {
		t.Error("an inactive window has a focused node")
	}
	changes := m.SetWindow(1, "T", true)
	if f := find(changes, ChangeFocus); len(f) != 1 || f[0].Ref.ID != 2 {
		t.Errorf("activating the window: focus changes = %+v, want node 2", f)
	}
	if n := m.Focused(1); n == nil || n.ID != 2 {
		t.Errorf("focused = %v, want node 2", n)
	}
	a.Flags.IsFocused, b.Flags.IsFocused = ui.TristateIsTrue, ui.TristateIsFalse
	if f := find(m.Apply(1, semtree.Update(a, b)), ChangeFocus); len(f) != 1 || f[0].Ref.ID != 1 {
		t.Errorf("moving the focus: %+v, want node 1", f)
	}
}

func TestSetWindowRenamesTheRoot(t *testing.T) {
	m := NewModel()
	m.SetWindow(1, "First", false)
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10)))
	if got := m.Name(m.Root(1)); got != "First" {
		t.Errorf("name = %q, want the title", got)
	}
	changes := m.SetWindow(1, "Second", false)
	if len(changes) != 1 || changes[0].Kind != ChangeProperty || changes[0].Property != PropName || changes[0].New != String("Second") {
		t.Errorf("changes = %+v, want the name property", changes)
	}
	if again := m.SetWindow(1, "Second", false); len(again) != 0 {
		t.Errorf("the same title again changed %+v", again)
	}
}

func TestNavigationHelpers(t *testing.T) {
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1, 2, 3), semtree.N(1, "a", 0, 0, 1, 1), semtree.N(2, "b", 0, 0, 1, 1, 4), semtree.N(3, "c", 0, 0, 1, 1), semtree.N(4, "d", 0, 0, 1, 1)))
	n2 := m.Node(NodeRef{1, 2})
	tests := []struct {
		name string
		got  *Node
		want int // -1 for nil
	}{
		{"next of the first", m.Sibling(m.Node(NodeRef{1, 1}), 1), 2},
		{"previous of the first", m.Sibling(m.Node(NodeRef{1, 1}), -1), -1},
		{"next of the last", m.Sibling(m.Node(NodeRef{1, 3}), 1), -1},
		{"previous of the last", m.Sibling(m.Node(NodeRef{1, 3}), -1), 2},
		{"the root has no sibling", m.Sibling(m.Root(1), 1), -1},
		{"parent", m.ParentOf(n2), 0},
		{"the root has no parent", m.ParentOf(m.Root(1)), -1},
		{"parent of a grandchild", m.ParentOf(m.Node(NodeRef{1, 4})), 2},
	}
	for _, tt := range tests {
		id := -1
		if tt.got != nil {
			id = tt.got.ID
		}
		if id != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, id, tt.want)
		}
	}
}

func TestRuntimeIDs(t *testing.T) {
	if got := RuntimeID(7); !reflect.DeepEqual(got, []int32{AppendRuntimeID, 8}) {
		t.Errorf("RuntimeID(7) = %v", got)
	}
	if got := RootRuntimeID(); !reflect.DeepEqual(got, []int32{AppendRuntimeID, 0}) {
		t.Errorf("RootRuntimeID() = %v", got)
	}
	seen := map[int32]bool{}
	for id := range 50 {
		r := RuntimeID(id)
		if seen[r[1]] || r[1] == 0 {
			t.Fatalf("runtime id %d repeats or is the root's", id)
		}
		seen[r[1]] = true
	}
}

func TestRectanglesAtScalesAndOrigins(t *testing.T) {
	tests := []struct {
		name         string
		scale        float32
		originX      float64
		originY      float64
		wantExtents  Rect
		wantOnScreen Rect
	}{
		{"scale 1 at the origin", 1, 0, 0, Rect{20, 30, 100, 40}, Rect{20, 30, 100, 40}},
		{"scale 1.5 at the origin", 1.5, 0, 0, Rect{30, 45, 150, 60}, Rect{30, 45, 150, 60}},
		{"scale 1 away from the origin", 1, 200, 100, Rect{20, 30, 100, 40}, Rect{220, 130, 100, 40}},
		{"scale 1.5 away from the origin", 1.5, 200, 100, Rect{30, 45, 150, 60}, Rect{230, 145, 150, 60}},
		{"scale 1.5 and a negative origin (a second monitor on the left)", 1.5, -1920, 40, Rect{30, 45, 150, 60}, Rect{-1890, 85, 150, 60}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := semtree.N(0, "", 0, 0, 400, 300, 1)
			root.Transform = semtree.Scale(tt.scale)
			child := semtree.N(1, "c", 20, 30, 100, 40)
			m := NewModel()
			m.Apply(1, semtree.Update(root, child))
			n := m.Node(NodeRef{1, 1})
			if got := m.Extents(n); got != tt.wantExtents {
				t.Errorf("extents = %+v, want %+v", got, tt.wantExtents)
			}
			if got := m.ScreenRect(n, tt.originX, tt.originY); got != tt.wantOnScreen {
				t.Errorf("screen rect = %+v, want %+v", got, tt.wantOnScreen)
			}
		})
	}
}

func TestRectanglesFollowTheTransformsOfTheAncestors(t *testing.T) {
	root := semtree.N(0, "", 0, 0, 400, 300, 1)
	mid := semtree.N(1, "mid", 0, 0, 200, 200, 2)
	mid.Transform[12], mid.Transform[13] = 50, 60 // translated by (50, 60)
	leaf := semtree.N(2, "leaf", 10, 10, 20, 20)
	m := NewModel()
	m.Apply(1, semtree.Update(root, mid, leaf))
	if got, want := m.Extents(m.Node(NodeRef{1, 2})), (Rect{60, 70, 20, 20}); got != want {
		t.Errorf("extents = %+v, want %+v", got, want)
	}
}

func TestHitTestFindsTheDeepestNodeOnTop(t *testing.T) {
	hidden := semtree.N(4, "hidden", 0, 0, 50, 50)
	hidden.Flags.IsHidden = true
	m := NewModel()
	m.Apply(1, semtree.Update(
		semtree.N(0, "", 0, 0, 400, 300, 1, 2, 4),
		semtree.N(1, "under", 0, 0, 100, 100, 3),
		semtree.N(2, "over", 50, 50, 100, 100),
		semtree.N(3, "inner", 10, 10, 20, 20),
		hidden,
	))
	tests := []struct {
		x, y float64
		want int
	}{
		{15, 15, 3},     // inside the inner node
		{60, 60, 2},     // where the later sibling covers the earlier one
		{80, 20, 1},     // only the first sibling
		{300, 250, 0},   // only the root
		{999, 999, -1},  // outside
		{10.5, 10.5, 3}, // a fractional point
		{100, 100, 2},   // the right edge of the first is not in it
	}
	for _, tt := range tests {
		id := -1
		if hit := m.HitTest(m.Root(1), tt.x, tt.y); hit != nil {
			id = hit.ID
		}
		if id != tt.want {
			t.Errorf("HitTest(%v, %v) = %d, want %d", tt.x, tt.y, id, tt.want)
		}
	}
	// A hidden node is not hit: the point (5, 5) is in it, and the answer is the node under it.
	if hit := m.HitTest(m.Root(1), 5, 5); hit == nil || hit.ID != 1 {
		t.Errorf("HitTest(5, 5) = %v, want the node under the hidden one", hit)
	}
}
