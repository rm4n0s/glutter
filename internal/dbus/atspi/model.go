package atspi

import (
	"math"
	"slices"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// rootID is the ID of the root node of the semantics tree of a view.
const rootID = 0

// NodeRef names a node of a view.
type NodeRef struct {
	View ui.ViewID
	ID   int
}

// Node is a node of the mirrored tree: what the framework last said about it.
type Node struct {
	ui.SemanticsNodeUpdate
	// View is the view of the node.
	View ui.ViewID
	// Parent is the ID of the parent, or -1 for the root.
	Parent int
}

// Ref names the node.
func (n *Node) Ref() NodeRef { return NodeRef{n.View, n.ID} }

// viewTree is the semantics tree of one view, and what the window says.
type viewTree struct {
	nodes         map[int]*Node
	customActions map[int]ui.SemanticsCustomActionUpdate
	title         string
	active        bool
}

// Model is the mirror of the semantics trees of the views of the app. It does not
// touch the bus: Apply says what changed, and the bridge turns that into events.
// A Model is not safe for concurrent use; the bridge locks it.
type Model struct {
	views map[ui.ViewID]*viewTree
}

// NewModel creates an empty model.
func NewModel() *Model { return &Model{views: map[ui.ViewID]*viewTree{}} }

// ChangeKind says what a Change is.
type ChangeKind int

// The kinds of change.
const (
	// ChangeNodeAdded: a node appeared and has to be exported.
	ChangeNodeAdded ChangeKind = iota + 1
	// ChangeNodeRemoved: a node went away and has to be unexported.
	ChangeNodeRemoved
	// ChangeChildAdded: Child was added to the children of Parent, at Index. A Parent
	// with View -1 is the application.
	ChangeChildAdded
	// ChangeChildRemoved: Child was removed from the children of Parent, which had it at Index.
	ChangeChildRemoved
	// ChangeState: the state State of Ref is now On.
	ChangeState
	// ChangeProperty: the property Property ("accessible-name" or "accessible-description")
	// of Ref changed to Value.
	ChangeProperty
	// ChangeWindow: the window of View became active (On) or inactive.
	ChangeWindow
	// ChangeNodeUpdated: the node that exists was updated; its interfaces may have changed.
	ChangeNodeUpdated
	// ChangeText: Value was inserted (Property "insert") into the text of Ref at the
	// character Index, or deleted (Property "delete") from there; Length is the number
	// of characters.
	ChangeText
	// ChangeCaret: the caret of the text of Ref is now at the character Index.
	ChangeCaret
	// ChangeTextSelection: the selected part of the text of Ref changed.
	ChangeTextSelection
	// ChangeValue: the numeric value of Ref is now Number.
	ChangeValue
	// ChangeVisibleData: the position or the size of Ref changed.
	ChangeVisibleData
	// ChangeSelection: the selected children of Ref changed.
	ChangeSelection
)

// Change is one change that Apply or the window setters made.
type Change struct {
	Kind     ChangeKind
	Ref      NodeRef
	Parent   NodeRef
	Child    NodeRef
	Index    int
	State    State
	On       bool
	Property string
	Value    string
	Length   int
	Number   float64
}

// appRef names the application root in a Change.
var appRef = NodeRef{View: -1, ID: rootID}

// IsApp says whether the ref is the application root.
func (r NodeRef) IsApp() bool { return r.View == -1 }

// Apply takes an update of the tree of a view and returns what changed. Nodes that
// the update leaves unreachable from the root are removed, as the engine does.
func (m *Model) Apply(view ui.ViewID, update *ui.SemanticsUpdate) []Change {
	var changes []Change
	vt := m.views[view]
	if vt == nil {
		vt = &viewTree{nodes: map[int]*Node{}, customActions: map[int]ui.SemanticsCustomActionUpdate{}}
		m.views[view] = vt
	}
	hadRoot := vt.nodes[rootID] != nil
	for _, a := range update.CustomActions {
		vt.customActions[a.ID] = a
	}
	type before struct {
		children []int
		states   StateSet
		label    string
		hint     string
		text     string
		caret    int
		base     int
		value    float64
		hasValue bool
		extents  Extents
		selected ui.Tristate
		existed  bool
	}
	old := map[int]before{}
	for _, nu := range update.Nodes {
		if _, seen := old[nu.ID]; seen {
			continue
		}
		if prev := vt.nodes[nu.ID]; prev != nil {
			b := before{children: slices.Clone(prev.ChildrenInTraversalOrder), states: statesOf(prev, nu.ID == rootID, vt.active),
				label: prev.Label, hint: prev.Hint, text: textOf(prev), extents: m.Extents(prev), selected: prev.Flags.IsSelected, existed: true}
			b.caret, b.base = caretOf(prev)
			if v, ok := valueOf(prev); ok {
				b.value, b.hasValue = v.current, true
			}
			old[nu.ID] = b
		} else {
			old[nu.ID] = before{}
		}
	}
	for _, nu := range update.Nodes {
		vt.nodes[nu.ID] = &Node{SemanticsNodeUpdate: nu, View: view, Parent: -1}
	}
	// The parents follow from the children lists.
	for _, n := range vt.nodes {
		n.Parent = -1
	}
	for _, n := range vt.nodes {
		for _, c := range n.ChildrenInTraversalOrder {
			if child := vt.nodes[c]; child != nil {
				child.Parent = n.ID
			}
		}
	}
	// What the root does not reach is gone.
	reach := map[int]bool{}
	var walk func(id int)
	walk = func(id int) {
		n := vt.nodes[id]
		if n == nil || reach[id] {
			return
		}
		reach[id] = true
		for _, c := range n.ChildrenInTraversalOrder {
			walk(c)
		}
	}
	walk(rootID)
	var removed []int
	for id := range vt.nodes {
		if !reach[id] {
			removed = append(removed, id)
		}
	}
	sort.Ints(removed)

	// Nodes first, so that the events find their objects.
	ids := make([]int, 0, len(old))
	for id := range old {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	selectionChanged := map[int]bool{}
	if vt.nodes[rootID] != nil && !hadRoot {
		changes = append(changes, Change{Kind: ChangeChildAdded, Parent: appRef, Child: NodeRef{view, rootID}, Index: m.frameIndex(view)})
	}
	for _, id := range ids {
		if !reach[id] {
			continue
		}
		if !old[id].existed {
			changes = append(changes, Change{Kind: ChangeNodeAdded, Ref: NodeRef{view, id}})
		}
	}
	for _, id := range ids {
		n, b := vt.nodes[id], old[id]
		if !reach[id] || !b.existed {
			continue
		}
		changes = append(changes, childChanges(n, b.children)...)
		now := statesOf(n, id == rootID, vt.active)
		for s := State(0); s <= stateLast; s++ {
			if now.Has(s) != b.states.Has(s) && s.Name() != "" {
				changes = append(changes, Change{Kind: ChangeState, Ref: n.Ref(), State: s, On: now.Has(s)})
			}
		}
		if n.Label != b.label {
			changes = append(changes, Change{Kind: ChangeProperty, Ref: n.Ref(), Property: "accessible-name", Value: n.Label})
		}
		if n.Hint != b.hint {
			changes = append(changes, Change{Kind: ChangeProperty, Ref: n.Ref(), Property: "accessible-description", Value: n.Hint})
		}
		changes = append(changes, m.contentChanges(n, b.text, b.caret, b.base, b.value, b.hasValue)...)
		if e := m.Extents(n); e != b.extents {
			changes = append(changes, Change{Kind: ChangeVisibleData, Ref: n.Ref()})
		}
		if n.Flags.IsSelected != b.selected && n.Parent >= 0 {
			if parent := vt.nodes[n.Parent]; parent != nil && hasSelection(m, parent) && !selectionChanged[n.Parent] {
				selectionChanged[n.Parent] = true
				changes = append(changes, Change{Kind: ChangeNodeUpdated, Ref: parent.Ref()}, Change{Kind: ChangeSelection, Ref: parent.Ref()})
			}
		}
		changes = append(changes, Change{Kind: ChangeNodeUpdated, Ref: n.Ref()})
	}
	for _, id := range removed {
		if old[id].existed || vt.nodes[id] != nil {
			changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
		}
		delete(vt.nodes, id)
	}
	return changes
}

// contentChanges are the changes to the text, the caret and the value of a node that was
// updated.
func (m *Model) contentChanges(n *Node, oldText string, oldCaret, oldBase int, oldValue float64, hadValue bool) []Change {
	var out []Change
	if hasText(n) {
		now := textOf(n)
		if now != oldText {
			start, deleted, inserted := textDiff([]rune(oldText), []rune(now))
			if len(deleted) > 0 {
				out = append(out, Change{Kind: ChangeText, Ref: n.Ref(), Property: "delete", Index: start, Length: len(deleted), Value: string(deleted)})
			}
			if len(inserted) > 0 {
				out = append(out, Change{Kind: ChangeText, Ref: n.Ref(), Property: "insert", Index: start, Length: len(inserted), Value: string(inserted)})
			}
		}
		caret, base := caretOf(n)
		if caret != oldCaret && caret >= 0 {
			out = append(out, Change{Kind: ChangeCaret, Ref: n.Ref(), Index: caret})
		}
		if caret != oldCaret || base != oldBase {
			if (caret != base) || (oldCaret != oldBase) {
				out = append(out, Change{Kind: ChangeTextSelection, Ref: n.Ref()})
			}
		}
	}
	if v, ok := valueOf(n); ok && (!hadValue || v.current != oldValue) {
		out = append(out, Change{Kind: ChangeValue, Ref: n.Ref(), Number: v.current})
	}
	return out
}

// textDiff finds the part of the text that changed: the characters that were deleted
// from the old text and those that were inserted in their place, from the character
// start. What the two texts have at the start and at the end is left out.
func textDiff(old, now []rune) (start int, deleted, inserted []rune) {
	for start < len(old) && start < len(now) && old[start] == now[start] {
		start++
	}
	endOld, endNow := len(old), len(now)
	for endOld > start && endNow > start && old[endOld-1] == now[endNow-1] {
		endOld--
		endNow--
	}
	return start, old[start:endOld], now[start:endNow]
}

// childChanges is the difference between the children that a node had and has.
func childChanges(n *Node, was []int) []Change {
	var out []Change
	now := n.ChildrenInTraversalOrder
	for i, c := range was {
		if !slices.Contains(now, c) {
			out = append(out, Change{Kind: ChangeChildRemoved, Parent: n.Ref(), Child: NodeRef{n.View, c}, Index: i})
		}
	}
	for i, c := range now {
		if !slices.Contains(was, c) {
			out = append(out, Change{Kind: ChangeChildAdded, Parent: n.Ref(), Child: NodeRef{n.View, c}, Index: i})
		}
	}
	return out
}

// frameIndex is the index that the frame of the view has among the frames, which are in
// the order of the view IDs.
func (m *Model) frameIndex(view ui.ViewID) int {
	i := 0
	for _, v := range m.Views() {
		if v == view {
			return i
		}
		i++
	}
	return i
}

// Views are the views that have a tree, in the order of their IDs.
func (m *Model) Views() []ui.ViewID {
	var views []ui.ViewID
	for v, vt := range m.views {
		if vt.nodes[rootID] != nil {
			views = append(views, v)
		}
	}
	slices.Sort(views)
	return views
}

// RemoveView forgets the tree of a view that closed and says what that changed.
func (m *Model) RemoveView(view ui.ViewID) []Change {
	vt := m.views[view]
	if vt == nil {
		return nil
	}
	var changes []Change
	if vt.nodes[rootID] != nil {
		changes = append(changes, Change{Kind: ChangeChildRemoved, Parent: appRef, Child: NodeRef{view, rootID}, Index: m.frameIndex(view)})
	}
	ids := make([]int, 0, len(vt.nodes))
	for id := range vt.nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
	}
	delete(m.views, view)
	return changes
}

// SetWindow sets the title and whether the window of the view has the focus. The title
// is the name of the frame. It says what that changed.
func (m *Model) SetWindow(view ui.ViewID, title string, active bool) []Change {
	vt := m.views[view]
	if vt == nil {
		vt = &viewTree{nodes: map[int]*Node{}, customActions: map[int]ui.SemanticsCustomActionUpdate{}}
		m.views[view] = vt
	}
	var changes []Change
	root := NodeRef{view, rootID}
	hasRoot := vt.nodes[rootID] != nil
	if title != vt.title {
		vt.title = title
		if hasRoot {
			changes = append(changes, Change{Kind: ChangeProperty, Ref: root, Property: "accessible-name", Value: title})
		}
	}
	if active != vt.active {
		vt.active = active
		if hasRoot {
			changes = append(changes, Change{Kind: ChangeState, Ref: root, State: StateActive, On: active},
				Change{Kind: ChangeWindow, Ref: root, On: active})
		}
	}
	return changes
}

// Node returns a node, or nil.
func (m *Model) Node(ref NodeRef) *Node {
	if vt := m.views[ref.View]; vt != nil {
		return vt.nodes[ref.ID]
	}
	return nil
}

// Name is the name of the node: its label; the frame of a window has the title of the
// window when it has no label.
func (m *Model) Name(n *Node) string {
	if n.ID == rootID && n.Label == "" {
		return m.views[n.View].title
	}
	return n.Label
}

// Role is the role of the node.
func (m *Model) Role(n *Node) Role { return roleOf(n, n.ID == rootID) }

// States are the states of the node.
func (m *Model) States(n *Node) StateSet {
	return statesOf(n, n.ID == rootID, m.views[n.View].active)
}

// Children are the refs of the children of the node, in traversal order; those that the
// tree does not have are left out.
func (m *Model) Children(n *Node) []NodeRef {
	vt := m.views[n.View]
	out := make([]NodeRef, 0, len(n.ChildrenInTraversalOrder))
	for _, c := range n.ChildrenInTraversalOrder {
		if vt.nodes[c] != nil {
			out = append(out, NodeRef{n.View, c})
		}
	}
	return out
}

// affine is a 2D transform: x' = a*x + c*y + tx, y' = b*x + d*y + ty.
type affine struct{ a, b, c, d, tx, ty float64 }

var identity = affine{a: 1, d: 1}

// fromMatrix takes the 2D part of a 4x4 matrix in column-major order.
func fromMatrix(m [16]float32) affine {
	return affine{float64(m[0]), float64(m[1]), float64(m[4]), float64(m[5]), float64(m[12]), float64(m[13])}
}

// then is the transform that applies o first and then t.
func (t affine) then(o affine) affine {
	return affine{
		a: t.a*o.a + t.c*o.b, b: t.b*o.a + t.d*o.b,
		c: t.a*o.c + t.c*o.d, d: t.b*o.c + t.d*o.d,
		tx: t.a*o.tx + t.c*o.ty + t.tx, ty: t.b*o.tx + t.d*o.ty + t.ty,
	}
}

func (t affine) apply(x, y float64) (float64, float64) {
	return t.a*x + t.c*y + t.tx, t.b*x + t.d*y + t.ty
}

// Extents is a rectangle in pixels of the window.
type Extents struct{ X, Y, Width, Height int32 }

// Contains says whether the point is in the rectangle.
func (e Extents) Contains(x, y int32) bool {
	return x >= e.X && x < e.X+e.Width && y >= e.Y && y < e.Y+e.Height
}

// Extents is the rectangle of the node in the pixels of its window: the rectangle of the
// node in its own coordinates, moved by the transforms of the node and of its
// ancestors up to the root (whose transform has the device pixel ratio).
func (m *Model) Extents(n *Node) Extents {
	vt := m.views[n.View]
	t := identity
	for cur, hops := n, 0; cur != nil && hops < 10000; hops++ {
		t = fromMatrix(cur.Transform).then(t)
		if cur.Parent < 0 {
			break
		}
		cur = vt.nodes[cur.Parent]
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range [][2]float64{{n.Rect.Left, n.Rect.Top}, {n.Rect.Right, n.Rect.Top}, {n.Rect.Left, n.Rect.Bottom}, {n.Rect.Right, n.Rect.Bottom}} {
		x, y := t.apply(p[0], p[1])
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	if math.IsInf(minX, 0) || math.IsNaN(minX) {
		return Extents{}
	}
	return Extents{X: int32(math.Round(minX)), Y: int32(math.Round(minY)),
		Width: int32(math.Round(maxX - minX)), Height: int32(math.Round(maxY - minY))}
}

// HitTest is the deepest node under the point (in the pixels of the window) that is under
// the node, or nil. A hidden node and its children are not hit; the last children are
// tried first, as they are on top.
func (m *Model) HitTest(n *Node, x, y int32) *Node {
	if n.Flags.IsHidden || !m.Extents(n).Contains(x, y) {
		return nil
	}
	vt := m.views[n.View]
	for i := len(n.ChildrenInTraversalOrder) - 1; i >= 0; i-- {
		if c := vt.nodes[n.ChildrenInTraversalOrder[i]]; c != nil {
			if hit := m.HitTest(c, x, y); hit != nil {
				return hit
			}
		}
	}
	return n
}
