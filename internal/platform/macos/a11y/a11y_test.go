package a11y

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// node makes a node update with the fields a test sets; the rest are the "no value" values of the framework.
func node(id int, f func(n *ui.SemanticsNodeUpdate), children ...int) ui.SemanticsNodeUpdate {
	n := ui.SemanticsNodeUpdate{
		ID: id, TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
		TraversalParent: -1, ChildrenInTraversalOrder: children, ChildrenInHitTestOrder: children,
		ScrollPosition: math.NaN(), ScrollExtentMax: math.NaN(), ScrollExtentMin: math.NaN(),
		Transform: identityMatrix(), HitTestTransform: identityMatrix(),
	}
	if f != nil {
		f(&n)
	}
	return n
}

func identityMatrix() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func update(nodes ...ui.SemanticsNodeUpdate) *ui.SemanticsUpdate {
	return &ui.SemanticsUpdate{Nodes: nodes}
}

func TestRole(t *testing.T) {
	tests := []struct {
		name string
		f    func(n *ui.SemanticsNodeUpdate)
		id   int
		role string
		sub  string
	}{
		{"root", nil, RootID, RoleGroup, ""},
		{"button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true }, 1, RoleButton, ""},
		{"check box", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsTrue }, 1, RoleCheckBox, ""},
		{"radio button", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsFalse
			n.Flags.IsInMutuallyExclusiveGroup = true
		}, 1, RoleRadioButton, ""},
		{"switch", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsTrue }, 1, RoleCheckBox, SubroleSwitch},
		{"text field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, 1, RoleTextField, ""},
		{"text area", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsMultiline = true }, 1, RoleTextArea, ""},
		{"password", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsObscured = true }, 1, RoleTextField, SubroleSecureField},
		{"slider", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true }, 1, RoleSlider, ""},
		{"link", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLink = true }, 1, RoleLink, ""},
		{"header", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHeader = true }, 1, RoleHeading, ""},
		{"image", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsImage = true }, 1, RoleImage, ""},
		{"text", func(n *ui.SemanticsNodeUpdate) { n.Label = "hello" }, 1, RoleStaticText, ""},
		{"container", nil, 1, RoleGroup, ""},
		{"tab", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTab }, 1, RoleRadioButton, ""},
		{"tab bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTabBar }, 1, RoleTabGroup, ""},
		{"dialog", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleDialog }, 1, RoleGroup, SubroleDialog},
		{"list", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleList }, 1, RoleList, ""},
		{"menu item", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenuItem }, 1, RoleMenuItem, ""},
		{"progress bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleProgressBar }, 1, RoleProgress, ""},
		{"spin button", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleSpinButton }, 1, RoleIncrementor, ""},
		{"combo box", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleComboBox }, 1, RolePopUp, ""},
		{"status", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleStatus }, 1, RoleGroup, SubroleStatus},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := &Node{SemanticsNodeUpdate: node(tc.id, tc.f)}
			if got := Role(n); got != tc.role {
				t.Errorf("role = %q, want %q", got, tc.role)
			}
			if got := Subrole(n); got != tc.sub {
				t.Errorf("subrole = %q, want %q", got, tc.sub)
			}
		})
	}
}

func TestLabelHelpAndValue(t *testing.T) {
	m := NewModel()
	m.Apply(0, update(node(0, nil, 1)))
	m.SetWindow(0, "My window", true)
	root := m.Root(0)
	if got := LabelOf(m, root); got != "My window" {
		t.Errorf("the root is named %q", got)
	}
	n := func(f func(n *ui.SemanticsNodeUpdate)) *Node { return &Node{SemanticsNodeUpdate: node(1, f)} }
	tests := []struct {
		name  string
		node  *Node
		label string
		help  string
		kind  ValueKind
		text  string
		num   float64
	}{
		{"label and hint", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Save"; u.Hint = "Saves the file"; u.Flags.IsButton = true }), "Save", "Saves the file", ValueNone, "", 0},
		{"tooltip wins as help", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Save"; u.Hint = "h"; u.Tooltip = "tip" }), "Save", "tip", ValueNone, "", 0},
		{"hint only is the label", n(func(u *ui.SemanticsNodeUpdate) { u.Hint = "Search" }), "Search", "", ValueNone, "", 0},
		{"text field: the value is the value", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Name"; u.Value = "Ada"; u.Flags.IsTextField = true }), "Name", "", ValueString, "Ada", 0},
		{"a value with a label of its own", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Volume"; u.Value = "loud" }), "Volume", "", ValueString, "loud", 0},
		{"value only is the label", n(func(u *ui.SemanticsNodeUpdate) { u.Value = "42" }), "42", "", ValueNone, "", 0},
		{"check box on", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Opt"; u.Flags.IsChecked = ui.CheckedStateIsTrue }), "Opt", "", ValueNumber, "", 1},
		{"check box mixed", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Opt"; u.Flags.IsChecked = ui.CheckedStateMixed }), "Opt", "", ValueNumber, "", 2},
		{"switch off", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Wi-Fi"; u.Flags.IsToggled = ui.TristateIsFalse }), "Wi-Fi", "", ValueNumber, "", 0},
		{"slider", n(func(u *ui.SemanticsNodeUpdate) { u.Label = "Vol"; u.Value = "30%"; u.Flags.IsSlider = true }), "Vol", "", ValueNumber, "", 30},
		{"slider with limits", n(func(u *ui.SemanticsNodeUpdate) {
			u.Label = "Temp"
			u.Value = "5"
			u.Flags.IsSlider = true
			u.MinValue, u.MaxValue = "0", "10"
		}), "Temp", "", ValueNumber, "", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := LabelOf(m, tc.node); got != tc.label {
				t.Errorf("label = %q, want %q", got, tc.label)
			}
			if got := Help(tc.node); got != tc.help {
				t.Errorf("help = %q, want %q", got, tc.help)
			}
			kind, text, num := Value(tc.node)
			if kind != tc.kind || text != tc.text || num != tc.num {
				t.Errorf("value = %v %q %v, want %v %q %v", kind, text, num, tc.kind, tc.text, tc.num)
			}
		})
	}
}

func TestRangeOf(t *testing.T) {
	n := &Node{SemanticsNodeUpdate: node(1, func(u *ui.SemanticsNodeUpdate) {
		u.Flags.IsSlider = true
		u.Value = "7"
		u.MinValue, u.MaxValue = "2", "12"
	})}
	r, ok := RangeOf(n)
	if !ok || r != (Range{Current: 7, Min: 2, Max: 12}) {
		t.Errorf("range = %+v, %v", r, ok)
	}
	pct := &Node{SemanticsNodeUpdate: node(1, func(u *ui.SemanticsNodeUpdate) {
		u.Role = ui.SemanticsRoleProgressBar
		u.Value = "75%"
		u.MinValue, u.MaxValue = "2", "12" // ignored for a percentage
	})}
	if r, ok := RangeOf(pct); !ok || r != (Range{Current: 75, Min: 0, Max: 100}) {
		t.Errorf("percentage = %+v, %v", r, ok)
	}
	if _, ok := RangeOf(&Node{SemanticsNodeUpdate: node(1, nil)}); ok {
		t.Error("a container has a range")
	}
	if _, ok := RangeOf(&Node{SemanticsNodeUpdate: node(1, func(u *ui.SemanticsNodeUpdate) { u.Role = ui.SemanticsRoleProgressBar })}); ok {
		t.Error("an indeterminate progress bar has a range")
	}
}

func TestStates(t *testing.T) {
	n := func(f func(u *ui.SemanticsNodeUpdate)) *Node { return &Node{SemanticsNodeUpdate: node(1, f)} }
	if !Enabled(n(nil)) || !Enabled(n(func(u *ui.SemanticsNodeUpdate) { u.Flags.IsEnabled = ui.TristateIsTrue })) {
		t.Error("a node with no enabled state, or an enabled one, is not enabled")
	}
	if Enabled(n(func(u *ui.SemanticsNodeUpdate) { u.Flags.IsEnabled = ui.TristateIsFalse })) {
		t.Error("a disabled node is enabled")
	}
	if !Selected(n(func(u *ui.SemanticsNodeUpdate) { u.Flags.IsSelected = ui.TristateIsTrue })) || Selected(n(nil)) {
		t.Error("selected")
	}
	for name, tc := range map[string]struct {
		f    func(u *ui.SemanticsNodeUpdate)
		want int
	}{
		"expanded":      {func(u *ui.SemanticsNodeUpdate) { u.Flags.IsExpanded = ui.TristateIsTrue }, 1},
		"collapsed":     {func(u *ui.SemanticsNodeUpdate) { u.Flags.IsExpanded = ui.TristateIsFalse }, 0},
		"can collapse":  {func(u *ui.SemanticsNodeUpdate) { u.Actions = ui.SemanticsActionCollapse }, 1},
		"can expand":    {func(u *ui.SemanticsNodeUpdate) { u.Actions = ui.SemanticsActionExpand }, 0},
		"cannot expand": {nil, -1},
	} {
		if got := ExpandedState(n(tc.f)); got != tc.want {
			t.Errorf("%s: %d, want %d", name, got, tc.want)
		}
	}
}

func TestActions(t *testing.T) {
	n := &Node{SemanticsNodeUpdate: node(1, func(u *ui.SemanticsNodeUpdate) {
		u.Actions = ui.SemanticsActionTap | ui.SemanticsActionIncrease | ui.SemanticsActionLongPress | ui.SemanticsActionScrollUp | ui.SemanticsActionFocus
	})}
	want := []string{ActionPress, ActionIncrement, ActionShowMenu, ActionScrollDown}
	if got := ActionNames(n); !reflect.DeepEqual(got, want) {
		t.Errorf("actions = %v, want %v", got, want)
	}
	for name, action := range map[string]ui.SemanticsAction{
		ActionPress: ui.SemanticsActionTap, ActionIncrement: ui.SemanticsActionIncrease,
		ActionShowMenu: ui.SemanticsActionLongPress, ActionScrollDown: ui.SemanticsActionScrollUp,
	} {
		if got, ok := ActionFor(n, name); !ok || got != action {
			t.Errorf("ActionFor(%s) = %v, %v, want %v", name, got, ok, action)
		}
	}
	if _, ok := ActionFor(n, ActionDecrement); ok {
		t.Error("a node that cannot decrease performs the decrement")
	}
	if _, ok := ActionFor(n, "AXBogus"); ok {
		t.Error("an unknown action is performed")
	}
	root := &Node{SemanticsNodeUpdate: node(RootID, func(u *ui.SemanticsNodeUpdate) { u.Actions = ui.SemanticsActionTap })}
	if len(ActionNames(root)) != 0 {
		t.Error("the root has actions")
	}
}

func TestTextRanges(t *testing.T) {
	text := "a😀bc" // the emoji is 4 bytes and 2 UTF-16 units
	n := &Node{SemanticsNodeUpdate: node(1, func(u *ui.SemanticsNodeUpdate) {
		u.Flags.IsTextField = true
		u.Value = text
		u.TextSelectionBase, u.TextSelectionExtent = 1, 5 // the emoji
	})}
	loc, length, ok := SelectedRange(n)
	if !ok || loc != 1 || length != 2 {
		t.Errorf("range = %d, %d, %v, want 1, 2", loc, length, ok)
	}
	// The caret, and a selection made backwards.
	n.TextSelectionBase, n.TextSelectionExtent = 5, 5
	if loc, length, _ := SelectedRange(n); loc != 3 || length != 0 {
		t.Errorf("caret = %d, %d, want 3, 0", loc, length)
	}
	n.TextSelectionBase, n.TextSelectionExtent = 6, 1
	if loc, length, _ := SelectedRange(n); loc != 1 || length != 3 {
		t.Errorf("backwards = %d, %d, want 1, 3", loc, length)
	}
	n.TextSelectionBase = -1
	if _, _, ok := SelectedRange(n); ok {
		t.Error("a field with no selection has a range")
	}
	if NumberOfCharacters(n) != 5 {
		t.Errorf("characters = %d, want 5", NumberOfCharacters(n))
	}
	args := SelectionArguments(text, 1, 2)
	if args["base"] != 1 || args["extent"] != 5 {
		t.Errorf("arguments = %v, want base 1 and extent 5 in bytes", args)
	}
	for _, tc := range []struct{ bytes, units int }{{0, 0}, {1, 1}, {5, 3}, {6, 4}, {7, 5}, {99, 5}} {
		if got := UTF16Offset(text, tc.bytes); got != tc.units {
			t.Errorf("UTF16Offset(%d) = %d, want %d", tc.bytes, got, tc.units)
		}
	}
	for _, tc := range []struct{ units, bytes int }{{0, 0}, {1, 1}, {2, 1}, {3, 5}, {99, 7}, {-1, 0}} {
		if got := ByteOffset(text, tc.units); got != tc.bytes {
			t.Errorf("ByteOffset(%d) = %d, want %d", tc.units, got, tc.bytes)
		}
	}
}

func kinds(cs []Change) []ChangeKind {
	out := make([]ChangeKind, len(cs))
	for i, c := range cs {
		out[i] = c.Kind
	}
	return out
}

func find(cs []Change, kind ChangeKind, id int) (Change, bool) {
	for _, c := range cs {
		if c.Kind == kind && c.Ref.ID == id {
			return c, true
		}
	}
	return Change{}, false
}

func TestTreeUpdatesGiveTheRightChanges(t *testing.T) {
	m := NewModel()
	m.SetWindow(7, "Window", true)
	// The first tree: every node is new, the window opens.
	cs := m.Apply(7, update(node(0, nil, 1, 2), node(1, func(u *ui.SemanticsNodeUpdate) { u.Label = "A"; u.Flags.IsButton = true }),
		node(2, func(u *ui.SemanticsNodeUpdate) { u.Label = "B" })))
	for _, id := range []int{0, 1, 2} {
		if _, ok := find(cs, ChangeNodeAdded, id); !ok {
			t.Errorf("node %d was not reported as added: %v", id, kinds(cs))
		}
	}
	if _, ok := find(cs, ChangeWindowOpened, 0); !ok {
		t.Error("the window did not open")
	}
	if got := m.Children(m.Root(7)); len(got) != 2 {
		t.Errorf("the root has %d children", len(got))
	}
	// A label changes: one update with the label field, no structure change.
	cs = m.Apply(7, update(node(1, func(u *ui.SemanticsNodeUpdate) { u.Label = "A2"; u.Flags.IsButton = true })))
	c, ok := find(cs, ChangeUpdated, 1)
	if !ok || c.Fields != FieldLabel {
		t.Errorf("label change = %+v %v (all %v)", c, ok, kinds(cs))
	}
	if _, ok := find(cs, ChangeChildren, 0); ok {
		t.Error("a label change changed the children")
	}
	// The value of a check box and a state flag.
	cs = m.Apply(7, update(node(2, func(u *ui.SemanticsNodeUpdate) {
		u.Label = "B"
		u.Flags.IsChecked = ui.CheckedStateIsTrue
		u.Flags.IsEnabled = ui.TristateIsFalse
	})))
	c, _ = find(cs, ChangeUpdated, 2)
	if c.Fields&FieldValue == 0 || c.Fields&FieldEnabled == 0 || c.Fields&FieldRole == 0 {
		t.Errorf("check box change fields = %b", c.Fields)
	}
	// Insert a node: a parent update with a new child.
	cs = m.Apply(7, update(node(0, nil, 1, 3, 2), node(3, func(u *ui.SemanticsNodeUpdate) { u.Label = "C" })))
	if _, ok := find(cs, ChangeNodeAdded, 3); !ok {
		t.Errorf("an inserted node was not added: %v", kinds(cs))
	}
	if _, ok := find(cs, ChangeChildren, 0); !ok {
		t.Errorf("the parent of an inserted node did not change its children: %v", kinds(cs))
	}
	// Move a node: the order changes.
	cs = m.Apply(7, update(node(0, nil, 3, 1, 2)))
	if _, ok := find(cs, ChangeChildren, 0); !ok || len(cs) != 1 {
		t.Errorf("a move: %v", kinds(cs))
	}
	// Remove a node: the parent no longer has it, the node goes.
	cs = m.Apply(7, update(node(0, nil, 3, 2)))
	if _, ok := find(cs, ChangeNodeRemoved, 1); !ok {
		t.Errorf("a removed node was not reported: %v", kinds(cs))
	}
	if _, ok := find(cs, ChangeChildren, 0); !ok {
		t.Errorf("the parent of a removed node did not change its children: %v", kinds(cs))
	}
	if m.Node(NodeRef{View: 7, ID: 1}) != nil {
		t.Error("the removed node is still in the model")
	}
	// A node that was cut off (its parent drops it and its child with it): both go.
	m.Apply(7, update(node(0, nil, 3, 2, 4), node(4, nil, 5), node(5, nil)))
	cs = m.Apply(7, update(node(0, nil, 3, 2)))
	for _, id := range []int{4, 5} {
		if _, ok := find(cs, ChangeNodeRemoved, id); !ok {
			t.Errorf("node %d of a dropped subtree stayed", id)
		}
	}
	// A view that closes drops everything.
	cs = m.RemoveView(7)
	if len(cs) != 3 { // 0, 2, 3
		t.Errorf("RemoveView = %v", kinds(cs))
	}
	if len(m.Views()) != 0 {
		t.Error("the view is still there")
	}
}

func TestFocusFollowsTheWindowAndTheTree(t *testing.T) {
	m := NewModel()
	m.Apply(1, update(node(0, nil, 1, 2), node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsButton = true }),
		node(2, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsButton = true })))
	// A window that is not active reports no focus, and has no focused node.
	if m.Focused(1) != nil {
		t.Error("a window that is not active has a focused node")
	}
	cs := m.Apply(1, update(node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsButton = true; u.Flags.IsFocused = ui.TristateIsTrue })))
	if _, ok := find(cs, ChangeFocus, 1); ok {
		t.Error("the focus of a window that is not active was announced")
	}
	// Activating the window announces the node that has the focus.
	cs = m.SetWindow(1, "w", true)
	if _, ok := find(cs, ChangeFocus, 1); !ok {
		t.Errorf("activation: %v", kinds(cs))
	}
	if f := m.Focused(1); f == nil || f.ID != 1 {
		t.Errorf("focused = %v", f)
	}
	// The focus moves.
	cs = m.Apply(1, update(node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsButton = true }),
		node(2, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsButton = true; u.Flags.IsFocused = ui.TristateIsTrue })))
	if _, ok := find(cs, ChangeFocus, 2); !ok {
		t.Errorf("a focus move: %v", kinds(cs))
	}
	// The title names the root.
	cs = m.SetWindow(1, "Other", true)
	if c, ok := find(cs, ChangeUpdated, 0); !ok || c.Fields != FieldLabel {
		t.Errorf("a new title: %v", kinds(cs))
	}
}

func TestLiveRegionsAndTextSelectionAreAnnounced(t *testing.T) {
	m := NewModel()
	m.SetWindow(1, "w", true)
	cs := m.Apply(1, update(node(0, nil, 1, 2),
		node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsLiveRegion = true; u.Label = "Saved" }),
		node(2, func(u *ui.SemanticsNodeUpdate) {
			u.Flags.IsTextField = true
			u.Value = "abc"
			u.TextSelectionBase, u.TextSelectionExtent = 3, 3
		})))
	if _, ok := find(cs, ChangeAnnounce, 1); !ok {
		t.Errorf("a live region that appears speaks: %v", kinds(cs))
	}
	cs = m.Apply(1, update(node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsLiveRegion = true; u.Label = "Saved" })))
	if _, ok := find(cs, ChangeAnnounce, 1); ok {
		t.Error("a live region that did not change spoke")
	}
	cs = m.Apply(1, update(node(1, func(u *ui.SemanticsNodeUpdate) { u.Flags.IsLiveRegion = true; u.Label = "Failed" })))
	if _, ok := find(cs, ChangeAnnounce, 1); !ok {
		t.Error("a live region with a new label did not speak")
	}
	// The caret moves: a selection change; the text changes too: a value change.
	cs = m.Apply(1, update(node(2, func(u *ui.SemanticsNodeUpdate) {
		u.Flags.IsTextField = true
		u.Value = "abc"
		u.TextSelectionBase, u.TextSelectionExtent = 1, 1
	})))
	if _, ok := find(cs, ChangeTextSelection, 2); !ok {
		t.Errorf("a caret move: %v", kinds(cs))
	}
	cs = m.Apply(1, update(node(2, func(u *ui.SemanticsNodeUpdate) {
		u.Flags.IsTextField = true
		u.Value = "abcd"
		u.TextSelectionBase, u.TextSelectionExtent = 4, 4
	})))
	c, _ := find(cs, ChangeUpdated, 2)
	if c.Fields&FieldValue == 0 {
		t.Errorf("a text change did not change the value: %v", cs)
	}
}

func TestFramesFollowTheTransforms(t *testing.T) {
	m := NewModel()
	scale := func(s float32) [16]float32 { mat := identityMatrix(); mat[0], mat[5] = s, s; return mat }
	translate := func(x, y float32) [16]float32 { mat := identityMatrix(); mat[12], mat[13] = x, y; return mat }
	m.Apply(1, update(
		node(0, func(u *ui.SemanticsNodeUpdate) { u.Rect = ui.NewRectFromLTWH(0, 0, 100, 100); u.Transform = scale(2) }, 1),
		node(1, func(u *ui.SemanticsNodeUpdate) {
			u.Rect = ui.NewRectFromLTWH(0, 0, 20, 10)
			u.Transform = translate(10, 30)
		})))
	child := m.Node(NodeRef{View: 1, ID: 1})
	if got, want := m.Extents(child), (Rect{X: 20, Y: 60, Width: 40, Height: 20}); got != want {
		t.Errorf("extents = %+v, want %+v", got, want)
	}
	if hit := m.HitTest(m.Root(1), 30, 70); hit == nil || hit.ID != 1 {
		t.Errorf("a hit inside the child = %v", hit)
	}
	if hit := m.HitTest(m.Root(1), 150, 150); hit == nil || hit.ID != 0 {
		t.Errorf("a hit outside the child = %v", hit)
	}
	if hit := m.HitTest(m.Root(1), 250, 250); hit != nil {
		t.Errorf("a hit outside the root = %v", hit)
	}
	// The root moves: the child's frame changes with it, though the child did not.
	cs := m.Apply(1, update(node(0, func(u *ui.SemanticsNodeUpdate) { u.Rect = ui.NewRectFromLTWH(0, 0, 100, 100); u.Transform = scale(3) }, 1)))
	if c, ok := find(cs, ChangeUpdated, 1); !ok || c.Fields&FieldFrame == 0 {
		t.Errorf("the child did not move with its parent: %v", cs)
	}
	// A hidden node and its children are not hit.
	m.Apply(1, update(node(1, func(u *ui.SemanticsNodeUpdate) {
		u.Rect = ui.NewRectFromLTWH(0, 0, 20, 10)
		u.Transform = translate(10, 30)
		u.Flags.IsHidden = true
	})))
	if hit := m.HitTest(m.Root(1), 40, 100); hit == nil || hit.ID != 0 {
		t.Errorf("a hit on a hidden child = %v", hit)
	}
	if !slices.Equal([]int{0, 1}, []int{0, 1}) {
		t.Fatal("unreachable")
	}
}
