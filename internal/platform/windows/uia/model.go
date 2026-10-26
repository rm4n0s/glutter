package uia

import (
	"math"
	"slices"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go model of the UI Automation bridge (SPEC §25.11): the mirror of the
// semantics tree of each view, built from ui.SemanticsUpdate, and the changes between two trees that
// become UI Automation events. It does not touch Windows. It is the Windows counterpart of
// internal/dbus/atspi's Model: that one speaks the vocabulary of AT-SPI2 (states, property names,
// characters), so it was not moved here (the change would have altered its behavior; see "Decided in
// step 28.13" in SPEC §25.11).
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
// Flutter: shell/platform/common/flutter_platform_node_delegate.cc FlutterPlatformNodeDelegate
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
	nodes         map[int]*Node
	customActions map[int]ui.SemanticsCustomActionUpdate
	title         string
	active        bool
	// focus is the ID of the node that has the keyboard focus, or -1.
	focus int
}

// Model is the mirror of the semantics trees of the views of the app. Apply says what changed, and the
// bridge turns that into events. A Model is not safe for concurrent use; the bridge locks it.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge
type Model struct {
	views map[ui.ViewID]*viewTree
	// pendingText is the text that SetValue wants a field to take once the framework offers the
	// action (it does so only for a field that has the focus).
	pendingText map[NodeRef]string
}

// NewModel creates an empty model.
//
// Flutter: no counterpart
func NewModel() *Model {
	return &Model{views: map[ui.ViewID]*viewTree{}, pendingText: map[NodeRef]string{}}
}

// ChangeKind says what a Change is.
//
// Flutter: no counterpart
type ChangeKind int

// The kinds of change.
const (
	// ChangeNodeAdded: a node appeared, and its provider has to be made.
	//
	// Flutter: no counterpart
	ChangeNodeAdded ChangeKind = iota + 1
	// ChangeNodeRemoved: a node went away, and its provider has to be disconnected.
	//
	// Flutter: no counterpart
	ChangeNodeRemoved
	// ChangeStructure: the children of Ref changed; Structure is the StructureChangeType to raise.
	//
	// Flutter: shell/windows accessibility_bridge_windows.cc kChildrenChanged
	ChangeStructure
	// ChangeProperty: the property Property of Ref changed from Old to New.
	//
	// Flutter: no counterpart
	ChangeProperty
	// ChangeFocus: Ref has the keyboard focus (the window is active).
	//
	// Flutter: shell/windows accessibility_bridge_windows.cc kFocus
	ChangeFocus
	// ChangeLiveRegion: the live region Ref changed.
	//
	// Flutter: shell/windows accessibility_bridge_windows.cc kLiveRegionChanged
	ChangeLiveRegion
	// ChangeTextChanged: the text of the field Ref changed.
	//
	// Flutter: shell/windows accessibility_bridge_windows.cc kTextChanged
	ChangeTextChanged
	// ChangeTextSelection: the caret or the selection of the field Ref changed.
	//
	// Flutter: shell/windows accessibility_bridge_windows.cc kDocumentSelectionChanged
	ChangeTextSelection
	// ChangeWindowOpened: the first tree of the view arrived, so the window of the view has content.
	//
	// Flutter: no counterpart
	ChangeWindowOpened
	// ChangeAction: the action Action has to be performed now (SetValue waited for it).
	//
	// Flutter: no counterpart
	ChangeAction
)

// Change is one change that Apply or the window setters made.
//
// Flutter: no counterpart
type Change struct {
	Kind      ChangeKind
	Ref       NodeRef
	Structure int
	Property  PropertyID
	Old, New  Value
	Action    ui.SemanticsActionEvent
}

// Apply takes an update of the tree of a view and returns what changed. Nodes that the update leaves
// unreachable from the root are removed, as the engine does.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::CommitUpdates
func (m *Model) Apply(view ui.ViewID, update *ui.SemanticsUpdate) []Change {
	var changes []Change
	vt := m.viewTree(view)
	hadRoot := vt.nodes[RootID] != nil
	for _, a := range update.CustomActions {
		vt.customActions[a.ID] = a
	}
	type before struct {
		existed  bool
		children []int
		props    map[PropertyID]Value
		text     string
		caret    int
		base     int
		live     bool
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
		b := before{existed: true, children: slices.Clone(prev.ChildrenInTraversalOrder), props: m.watched(prev), text: textOf(prev), live: prev.Flags.IsLiveRegion}
		b.caret, b.base = caretOf(prev)
		old[nu.ID] = b
	}
	oldFocus := vt.focus
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
	walk(RootID)
	var removed []int
	for id := range vt.nodes {
		if !reach[id] {
			removed = append(removed, id)
		}
	}
	sort.Ints(removed)
	vt.focus = m.findFocus(vt)

	ids := make([]int, 0, len(old))
	for id := range old {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	// The nodes first, so that the events find their providers.
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
		if kind, ok := structureChange(b.children, n.ChildrenInTraversalOrder); ok {
			changes = append(changes, Change{Kind: ChangeStructure, Ref: n.Ref(), Structure: kind})
		}
		now := m.watched(n)
		for _, p := range watchedProperties {
			if o, nw := b.props[p], now[p]; o != nw {
				changes = append(changes, Change{Kind: ChangeProperty, Ref: n.Ref(), Property: p, Old: o, New: nw})
			}
		}
		if n.Flags.IsLiveRegion && (n.Label != "" || n.Value != "") && (b.props[PropName] != now[PropName] || b.props[PropValueValue] != now[PropValueValue] || !b.live) {
			changes = append(changes, Change{Kind: ChangeLiveRegion, Ref: n.Ref()})
		}
		if n.Flags.IsTextField {
			if textOf(n) != b.text {
				changes = append(changes, Change{Kind: ChangeTextChanged, Ref: n.Ref()})
			}
			if caret, base := caretOf(n); caret != b.caret || base != b.base {
				changes = append(changes, Change{Kind: ChangeTextSelection, Ref: n.Ref()})
			}
		}
	}
	// A node that is new and is a live region speaks as it appears.
	for _, id := range ids {
		if n := vt.nodes[id]; reach[id] && !old[id].existed && n.Flags.IsLiveRegion && (n.Label != "" || n.Value != "") {
			changes = append(changes, Change{Kind: ChangeLiveRegion, Ref: n.Ref()})
		}
	}
	for _, id := range removed {
		if old[id].existed || vt.nodes[id] != nil {
			changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
		}
		delete(vt.nodes, id)
		delete(m.pendingText, NodeRef{view, id})
	}
	if vt.focus != oldFocus && vt.focus >= 0 && vt.active {
		changes = append(changes, Change{Kind: ChangeFocus, Ref: NodeRef{view, vt.focus}})
	}
	// SetValue waited for a field to get the action.
	for ref, text := range m.pendingText {
		if ref.View != view {
			continue
		}
		if n := vt.nodes[ref.ID]; n != nil && n.Actions&ui.SemanticsActionSetText != 0 {
			delete(m.pendingText, ref)
			changes = append(changes, Change{Kind: ChangeAction, Ref: ref,
				Action: ui.SemanticsActionEvent{ViewID: view, NodeID: ref.ID, Type: ui.SemanticsActionSetText, Arguments: text}})
		}
	}
	return changes
}

// viewTree returns the tree of the view, made if it is new.
func (m *Model) viewTree(view ui.ViewID) *viewTree {
	vt := m.views[view]
	if vt == nil {
		vt = &viewTree{nodes: map[int]*Node{}, customActions: map[int]ui.SemanticsCustomActionUpdate{}, focus: -1}
		m.views[view] = vt
	}
	return vt
}

// findFocus is the node that has the keyboard focus: the first node in traversal order whose focus flag
// is true.
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

// structureChange tells how the children of a node changed: the order only (reordered), or anything
// else (invalidated, which tells a client to ask again).
func structureChange(was, now []int) (kind int, changed bool) {
	if slices.Equal(was, now) {
		return 0, false
	}
	a, b := slices.Clone(was), slices.Clone(now)
	slices.Sort(a)
	slices.Sort(b)
	if slices.Equal(a, b) {
		return StructureChildrenReordered, true
	}
	return StructureChildrenInvalidated, true
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
	var changes []Change
	ids := make([]int, 0, len(vt.nodes))
	for id := range vt.nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		changes = append(changes, Change{Kind: ChangeNodeRemoved, Ref: NodeRef{view, id}})
		delete(m.pendingText, NodeRef{view, id})
	}
	delete(m.views, view)
	return changes
}

// SetWindow sets the title and whether the window of the view has the focus. The title names the root.
// It says what that changed: the focus event of the node that has the focus when the window is activated.
//
// Flutter: no counterpart
func (m *Model) SetWindow(view ui.ViewID, title string, active bool) []Change {
	vt := m.viewTree(view)
	var changes []Change
	hasRoot := vt.nodes[RootID] != nil
	if title != vt.title {
		old := m.Property(vt.nodes[RootID], PropName)
		vt.title = title
		if hasRoot {
			if now := m.Property(vt.nodes[RootID], PropName); now != old {
				changes = append(changes, Change{Kind: ChangeProperty, Ref: NodeRef{view, RootID}, Property: PropName, Old: old, New: now})
			}
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

// Focused returns the node of the view that has the keyboard focus, or nil. A window that is not
// active has none.
//
// Flutter: shell/platform/common/accessibility_bridge.cc AccessibilityBridge::GetFocusedNode
func (m *Model) Focused(view ui.ViewID) *Node {
	vt := m.views[view]
	if vt == nil || !vt.active || vt.focus < 0 {
		return nil
	}
	return vt.nodes[vt.focus]
}

// Children are the refs of the children of the node, in traversal order; those that the tree does not
// have are left out.
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

// Sibling returns the sibling of the node that comes next (delta 1) or before (delta -1) in its
// parent's children, or nil. The root has none.
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::Navigate
func (m *Model) Sibling(n *Node, delta int) *Node {
	if n.Parent < 0 {
		return nil
	}
	parent := m.views[n.View].nodes[n.Parent]
	if parent == nil {
		return nil
	}
	kids := m.Children(parent)
	for i, k := range kids {
		if k.ID == n.ID {
			if j := i + delta; j >= 0 && j < len(kids) {
				return m.Node(kids[j])
			}
			return nil
		}
	}
	return nil
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

// RuntimeID is the runtime ID of a node: UiaAppendRuntimeId, which UI Automation completes with the
// identity of the window, then the node's number plus one (so that no number is 0).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::GetRuntimeId
func RuntimeID(id int) []int32 { return []int32{AppendRuntimeID, int32(id) + 1} }

// RootRuntimeID is the runtime ID of the fragment root of a window.
//
// Flutter: third_party/accessibility/ax/platform/ax_fragment_root_win.cc AXFragmentRootWin::GetRuntimeId
func RootRuntimeID() []int32 { return []int32{AppendRuntimeID, 0} }

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

// Rect is a rectangle in pixels, as UI Automation's UiaRect has it.
//
// Flutter: no counterpart
type Rect struct{ X, Y, Width, Height float64 }

// Contains says whether the point is in the rectangle.
//
// Flutter: no counterpart
func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Extents is the rectangle of the node in the pixels of the client area of its window: the rectangle of
// the node in its own coordinates, moved by the transforms of the node and of its ancestors up to the
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

// ScreenRect is the rectangle of the node on the screen: Extents moved by the position of the client
// area of the window on the screen, in physical pixels (BoundingRectangle).
//
// Flutter: third_party/accessibility/ax/platform/ax_platform_node_win.cc AXPlatformNodeWin::get_BoundingRectangle
func (m *Model) ScreenRect(n *Node, originX, originY float64) Rect {
	r := m.Extents(n)
	r.X += originX
	r.Y += originY
	return r
}

// HitTest is the deepest node under the point (in the pixels of the client area) that is under the
// node, or nil. A hidden node and its children are not hit; the last children are tried first, as they
// are on top.
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
