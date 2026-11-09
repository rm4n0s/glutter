package a11y

import (
	"math"
	"slices"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go model of the bridge (SPEC §29.10): the mirror of the semantics tree of each view, built
// from ui.SemanticsUpdate, and the changes between two trees that become NSAccessibility notifications. It does not
// touch AppKit. It is the macOS counterpart of internal/platform/windows/uia's Model, which speaks UI Automation's
// vocabulary and so was not shared.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge

// RootID is the ID of the root node of the semantics tree of a view.
const RootID = 0

// NodeRef names a node of a view.
//
// Flutter: no counterpart
type NodeRef struct {
	View ui.ViewID
	ID   int
}

// Node is a node of the mirrored tree: what the framework last said about it.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterPlatformNodeDelegateMac.mm FlutterPlatformNodeDelegateMac
type Node struct {
	ui.SemanticsNodeUpdate
	// View is the view of the node.
	View ui.ViewID
	// Parent is the ID of the parent, or -1 for the root.
	Parent int
}

// Ref names the node.
//
// Flutter: no counterpart
func (n *Node) Ref() NodeRef { return NodeRef{n.View, n.ID} }

// viewTree is the semantics tree of one view, and what the window says.
type viewTree struct {
	nodes  map[int]*Node
	title  string
	active bool
	// focus is the ID of the node that has the keyboard focus, or -1.
	focus int
}

// Model is the mirror of the semantics trees of the views of the app. Apply says what changed, and the bridge turns
// that into notifications. A Model is not safe for concurrent use: it belongs to the UI thread.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge
type Model struct {
	views map[ui.ViewID]*viewTree
}

// NewModel creates an empty model.
//
// Flutter: no counterpart
func NewModel() *Model { return &Model{views: map[ui.ViewID]*viewTree{}} }

// ChangeKind says what a Change is.
//
// Flutter: no counterpart
type ChangeKind int

// The kinds of change.
const (
	// ChangeNodeAdded: a node appeared, and its element has to be made.
	//
	// Flutter: no counterpart
	ChangeNodeAdded ChangeKind = iota + 1
	// ChangeNodeRemoved: a node went away, and its element has to be dropped.
	//
	// Flutter: no counterpart
	ChangeNodeRemoved
	// ChangeChildren: the children of Ref changed (a node was added, removed or moved below it): its element is told
	// its new children and a layout change is posted.
	//
	// Flutter: shell/platform/darwin/macos/framework/Source/AccessibilityBridgeMac.mm kLayoutChanged
	ChangeChildren
	// ChangeUpdated: the properties in Fields of the node changed.
	//
	// Flutter: shell/platform/darwin/macos/framework/Source/AccessibilityBridgeMac.mm kValueChanged
	ChangeUpdated
	// ChangeFocus: Ref has the keyboard focus (the window is active).
	//
	// Flutter: shell/platform/darwin/macos/framework/Source/AccessibilityBridgeMac.mm kFocusChanged
	ChangeFocus
	// ChangeAnnounce: the live region Ref spoke.
	//
	// Flutter: shell/platform/darwin/macos/framework/Source/AccessibilityBridgeMac.mm kLiveRegionChanged
	ChangeAnnounce
	// ChangeTextSelection: the caret or the selection of the field Ref changed.
	//
	// Flutter: shell/platform/darwin/macos/framework/Source/FlutterTextInputSemanticsObject.mm
	ChangeTextSelection
	// ChangeWindowOpened: the first tree of the view arrived: the window has content.
	//
	// Flutter: no counterpart
	ChangeWindowOpened
)

// Fields is a set of the properties of a node that Apply watches.
//
// Flutter: no counterpart
type Fields uint

// The watched properties.
const (
	FieldRole Fields = 1 << iota
	FieldLabel
	FieldHelp
	FieldValue
	FieldEnabled
	FieldSelected
	FieldExpanded
	FieldFrame
	FieldFlags
)

// Change is one change that Apply or the window setters made.
//
// Flutter: no counterpart
type Change struct {
	Kind   ChangeKind
	Ref    NodeRef
	Fields Fields
}

// snapshot is what Apply compares before and after an update, for one node.
type snapshot struct {
	children []int
	role     string
	label    string
	help     string
	value    string
	enabled  bool
	selected bool
	expanded int
	frame    Rect
	flags    ui.SemanticsFlags
	live     bool
	caret    int
	base     int
}

func (m *Model) snapshotOf(n *Node) snapshot {
	caret, base := Selection(n)
	return snapshot{
		children: slices.Clone(n.ChildrenInTraversalOrder),
		role:     Role(n), label: m.Label(n), help: Help(n), value: ValueText(n),
		enabled: Enabled(n), selected: Selected(n), expanded: ExpandedState(n),
		frame: m.Extents(n), flags: n.Flags, live: n.Flags.IsLiveRegion, caret: caret, base: base,
	}
}

// Apply takes an update of the tree of a view and returns what changed. Nodes that the update leaves unreachable from
// the root are removed, as the engine does.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::CommitUpdates
func (m *Model) Apply(view ui.ViewID, update *ui.SemanticsUpdate) []Change {
	var changes []Change
	vt := m.viewTree(view)
	hadRoot := vt.nodes[RootID] != nil
	type before struct {
		existed bool
		snap    snapshot
	}
	old := map[int]before{}
	for _, nu := range update.Nodes {
		if _, seen := old[nu.ID]; seen {
			continue
		}
		prev := vt.nodes[nu.ID]
		if prev == nil {
			old[nu.ID] = before{}
			continue
		}
		old[nu.ID] = before{existed: true, snap: m.snapshotOf(prev)}
	}
	// A node whose own state did not change can still move, when an ancestor's transform changes: the frames of all the
	// nodes that existed are compared, so the snapshots are taken for the whole tree.
	for id, n := range vt.nodes {
		if _, ok := old[id]; !ok {
			old[id] = before{existed: true, snap: m.snapshotOf(n)}
		}
	}
	oldFocus := vt.focus
	for _, nu := range update.Nodes {
		vt.nodes[nu.ID] = &Node{SemanticsNodeUpdate: nu, View: view, Parent: -1}
	}
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
	walk(RootID)
	var removed []int
	for id := range vt.nodes {
		if !reach[id] {
			removed = append(removed, id)
		}
	}
	sort.Ints(removed)
	vt.focus = m.findFocus(vt)

	ids := make([]int, 0, len(vt.nodes))
	for id := range vt.nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if reach[id] && !old[id].existed {
			changes = append(changes, Change{Kind: ChangeNodeAdded, Ref: NodeRef{view, id}})
		}
	}
	if vt.nodes[RootID] != nil && !hadRoot {
		changes = append(changes, Change{Kind: ChangeWindowOpened, Ref: NodeRef{view, RootID}})
	}
	for _, id := range ids {
		n, b := vt.nodes[id], old[id]
		if !reach[id] || !b.existed {
			continue
		}
		now := m.snapshotOf(n)
		if !slices.Equal(b.snap.children, now.children) {
			changes = append(changes, Change{Kind: ChangeChildren, Ref: n.Ref()})
		}
		var f Fields
		set := func(cond bool, field Fields) {
			if cond {
				f |= field
			}
		}
		set(b.snap.role != now.role, FieldRole)
		set(b.snap.label != now.label, FieldLabel)
		set(b.snap.help != now.help, FieldHelp)
		set(b.snap.value != now.value, FieldValue)
		set(b.snap.enabled != now.enabled, FieldEnabled)
		set(b.snap.selected != now.selected, FieldSelected)
		set(b.snap.expanded != now.expanded, FieldExpanded)
		set(b.snap.frame != now.frame, FieldFrame)
		set(b.snap.flags != now.flags, FieldFlags)
		if f != 0 {
			changes = append(changes, Change{Kind: ChangeUpdated, Ref: n.Ref(), Fields: f})
		}
		if n.Flags.IsLiveRegion && (n.Label != "" || n.Value != "") && (b.snap.label != now.label || b.snap.value != now.value || !b.snap.live) {
			changes = append(changes, Change{Kind: ChangeAnnounce, Ref: n.Ref()})
		}
		if n.Flags.IsTextField && (now.caret != b.snap.caret || now.base != b.snap.base) {
			changes = append(changes, Change{Kind: ChangeTextSelection, Ref: n.Ref()})
		}
	}
	for _, id := range ids {
		if n := vt.nodes[id]; reach[id] && !old[id].existed && n.Flags.IsLiveRegion && (n.Label != "" || n.Value != "") {
			changes = append(changes, Change{Kind: ChangeAnnounce, Ref: n.Ref()})
		}
	}
	for _, id := range removed {
		changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
		delete(vt.nodes, id)
	}
	if vt.focus != oldFocus && vt.focus >= 0 && vt.active {
		changes = append(changes, Change{Kind: ChangeFocus, Ref: NodeRef{view, vt.focus}})
	}
	return changes
}

func (m *Model) viewTree(view ui.ViewID) *viewTree {
	vt := m.views[view]
	if vt == nil {
		vt = &viewTree{nodes: map[int]*Node{}, focus: -1}
		m.views[view] = vt
	}
	return vt
}

// findFocus is the node that has the keyboard focus: the first node in traversal order whose focus flag is true.
func (m *Model) findFocus(vt *viewTree) int {
	found := -1
	seen := map[int]bool{}
	var walk func(id int)
	walk = func(id int) {
		n := vt.nodes[id]
		if n == nil || seen[id] || found >= 0 {
			return
		}
		seen[id] = true
		if n.Flags.IsFocused == ui.TristateIsTrue {
			found = id
			return
		}
		for _, c := range n.ChildrenInTraversalOrder {
			walk(c)
		}
	}
	walk(RootID)
	return found
}

// Views are the views that have a tree, in the order of their IDs.
//
// Flutter: no counterpart
func (m *Model) Views() []ui.ViewID {
	var views []ui.ViewID
	for v, vt := range m.views {
		if vt.nodes[RootID] != nil {
			views = append(views, v)
		}
	}
	slices.Sort(views)
	return views
}

// RemoveView forgets the tree of a view that closed and says what that changed.
//
// Flutter: no counterpart
func (m *Model) RemoveView(view ui.ViewID) []Change {
	vt := m.views[view]
	if vt == nil {
		return nil
	}
	ids := make([]int, 0, len(vt.nodes))
	for id := range vt.nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	changes := make([]Change, 0, len(ids))
	for _, id := range ids {
		changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
	}
	delete(m.views, view)
	return changes
}

// SetWindow sets the title and whether the window of the view has the focus. The title names the root. It says
// what that changed: the focus of the node that has the focus when the window is activated, the new name of the root.
//
// Flutter: no counterpart
func (m *Model) SetWindow(view ui.ViewID, title string, active bool) []Change {
	vt := m.viewTree(view)
	var changes []Change
	if title != vt.title {
		vt.title = title
		if vt.nodes[RootID] != nil {
			changes = append(changes, Change{Kind: ChangeUpdated, Ref: NodeRef{view, RootID}, Fields: FieldLabel})
		}
	}
	if active != vt.active {
		vt.active = active
		if active && vt.focus >= 0 {
			changes = append(changes, Change{Kind: ChangeFocus, Ref: NodeRef{view, vt.focus}})
		}
	}
	return changes
}

// Active says whether the window of the view has the focus.
//
// Flutter: no counterpart
func (m *Model) Active(view ui.ViewID) bool {
	vt := m.views[view]
	return vt != nil && vt.active
}

// Title is the title of the window of the view.
//
// Flutter: no counterpart
func (m *Model) Title(view ui.ViewID) string {
	if vt := m.views[view]; vt != nil {
		return vt.title
	}
	return ""
}

// Node returns a node, or nil.
//
// Flutter: no counterpart
func (m *Model) Node(ref NodeRef) *Node {
	if vt := m.views[ref.View]; vt != nil {
		return vt.nodes[ref.ID]
	}
	return nil
}

// Root returns the root node of the view, or nil while the view has no tree.
//
// Flutter: no counterpart
func (m *Model) Root(view ui.ViewID) *Node { return m.Node(NodeRef{view, RootID}) }

// Focused returns the node of the view that has the keyboard focus, or nil. A window that is not active has none.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::GetFocusedNode
func (m *Model) Focused(view ui.ViewID) *Node {
	vt := m.views[view]
	if vt == nil || !vt.active || vt.focus < 0 {
		return nil
	}
	return vt.nodes[vt.focus]
}

// Children are the refs of the children of the node, in traversal order; those that the tree does not have are
// left out.
//
// Flutter: no counterpart
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

// ParentOf returns the parent of the node, or nil for the root.
//
// Flutter: no counterpart
func (m *Model) ParentOf(n *Node) *Node {
	if n.Parent < 0 {
		return nil
	}
	return m.views[n.View].nodes[n.Parent]
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

// Rect is a rectangle with its top left corner and its size.
//
// Flutter: no counterpart
type Rect struct{ X, Y, Width, Height float64 }

// Contains says whether the point is in the rectangle.
//
// Flutter: no counterpart
func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Extents is the rectangle of the node in the pixels of the client area of its window, from the top left: the
// rectangle of the node in its own coordinates, moved by the transforms of the node and of its ancestors up to the
// root (whose transform has the device pixel ratio).
//
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate::GetBoundsRect
func (m *Model) Extents(n *Node) Rect {
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
		return Rect{}
	}
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// HitTest is the deepest node under the point (in the pixels of the client area) that is under the node, or nil. A
// hidden node and its children are not hit; the last children are tried first, as they are on top.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::GetHitTestedNode
func (m *Model) HitTest(n *Node, x, y float64) *Node {
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
