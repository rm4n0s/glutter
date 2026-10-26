package widgets

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

// defaultTreeRowExtent is the height of a row when the app gives no extent builder.
const defaultTreeRowExtent = 40.0

// TreeSliverNode is a node of the tree that a TreeSliver shows: a content, and the child nodes that
// it holds. The sliver tells each active node its depth and its parent.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode
type TreeSliverNode[T any] struct {
	content     T
	children    []*TreeSliverNode[T]
	hasChildren bool
	expanded    bool
	depth       foundation.Opt[int]
	parent      *TreeSliverNode[T]
	// changed tells the TreeSliver that shows the node that its children changed.
	changed func()
	// expansionChanged tells the TreeSliver that shows the node that the app set its expansion.
	expansionChanged func(*TreeSliverNode[T])
	// expandedByApp is whether the app gave the expansion with SetExpanded since a tree last took the node in;
	// a TreeSliver with a NodeKey then does not replace it with the expansion of the old node with the same key.
	expandedByApp bool
}

// TreeSliverNodeOpts are the optional parameters of NewTreeSliverNode.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.new
type TreeSliverNodeOpts[T any] struct {
	// Children are the nodes below this one; none for a leaf.
	Children []*TreeSliverNode[T]
	// Expanded is whether the children are shown. A node without children is never expanded, unless
	// HasChildren is set.
	Expanded bool
	// HasChildren makes the node one that has children that are not read yet (a lazy node, such as a folder
	// whose entries are listed when it opens): it can be toggled with no children, and the default node
	// builder shows its arrow. Toggling it calls OnNodeToggle, where the app adds the children (SetChildren),
	// which the node then shows, with the animation of the sliver when they are there at once.
	//
	// Deviation: Flutter's node has children or has none (SPEC §12.10).
	HasChildren bool
}

// NewTreeSliverNode creates a node that holds the content.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.new
func NewTreeSliverNode[T any](content T, opts ...TreeSliverNodeOpts[T]) *TreeSliverNode[T] {
	o := firstOpts(opts)
	return &TreeSliverNode[T]{content: content, children: o.Children, hasChildren: o.HasChildren,
		expanded: (len(o.Children) > 0 || o.HasChildren) && o.Expanded}
}

// Content is the content the node holds.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.content
func (n *TreeSliverNode[T]) Content() T { return n.content }

// Children are the nodes below this one.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) Children() []*TreeSliverNode[T] { return n.children }

// AddChild adds a node below this one, after the others. Dart's children is a list that the app changes; here
// the node has methods (AddChild, InsertChild, RemoveChild, SetChildren), and the TreeSliver that shows the node
// is told, so it shows the change at once, as Flutter's does when the list is changed and the tree is built again.
//
// Deviation: Flutter's children is mutable (SPEC §12.10).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) AddChild(child *TreeSliverNode[T]) {
	n.children = append(n.children, child)
	n.childrenChanged()
}

// InsertChild adds a node below this one at the index (0 to the number of children; the end for a larger one).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) InsertChild(index int, child *TreeSliverNode[T]) {
	n.children = slices.Insert(n.children, max(0, min(index, len(n.children))), child)
	n.childrenChanged()
}

// RemoveChild removes the node from the children of this one, and reports whether it was one. The removed node
// has no parent and no depth, and its subtree leaves the tree with it, whatever its state. A node that held its
// last child is collapsed (a lazy node is not).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) RemoveChild(child *TreeSliverNode[T]) bool {
	i := slices.Index(n.children, child)
	if i < 0 {
		return false
	}
	n.children = slices.Delete(slices.Clone(n.children), i, i+1)
	if child.parent == n {
		child.parent = nil
		child.depth = foundation.None[int]()
	}
	n.childrenChanged()
	return true
}

// SetChildren replaces the children of this one. The old children that are not among the new ones have no
// parent and no depth any more. A node left with no children is collapsed (a lazy node is not).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) SetChildren(children []*TreeSliverNode[T]) {
	for _, old := range n.children {
		if old.parent == n && !slices.Contains(children, old) {
			old.parent = nil
			old.depth = foundation.None[int]()
		}
	}
	n.children = slices.Clone(children)
	n.childrenChanged()
}

// HasChildren is whether the node has children, or is a lazy node whose children are not read yet.
//
// Deviation: Flutter's node has children or has none (SPEC §12.10).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.children
func (n *TreeSliverNode[T]) HasChildren() bool { return n.hasChildren || len(n.children) > 0 }

// childrenChanged keeps the expansion right after the children changed and tells the TreeSliver that shows the
// node, which makes the active nodes again (and with them the parent and the depth of every node).
func (n *TreeSliverNode[T]) childrenChanged() {
	if len(n.children) == 0 && !n.hasChildren {
		n.expanded = false
	}
	if n.changed != nil {
		n.changed()
	}
}

// IsExpanded is whether the children are shown (or being shown).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.isExpanded
func (n *TreeSliverNode[T]) IsExpanded() bool { return n.expanded }

// SetExpanded sets whether the children are shown, for a node that is shown or not. A node with no children
// that is not a lazy node (HasChildren) stays collapsed, as NewTreeSliverNode makes it. Setting the value the
// node has changes nothing. When the node is shown by a TreeSliver, the tree makes its active nodes again at
// once, with no animation of the rows (so Parent and Depth are right for the rows that come or go); the arrow
// of the default node builder still turns, as it has an animation of its own. The tree's controller
// (ExpandNode, CollapseNode, ToggleNode) animates and calls OnNodeToggle, which SetExpanded does not. A node
// that is hidden keeps the value until its ancestors are expanded. Call it on the UI thread.
//
// A TreeSliver that has a NodeKey keeps the expansion of the old nodes for new nodes with the same key; a
// node whose expansion the app gave with SetExpanded (even with the value the node had) before the tree took it
// in keeps the app's value instead, so an app that builds its nodes again can set the expansion from its model.
//
// Deviation: Flutter's TreeSliverNode._expanded is private, and Dart apps build the nodes again to change the
// expansion; Go apps keep the state of their tree in a model (SPEC §12.10).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode._expanded
func (n *TreeSliverNode[T]) SetExpanded(expanded bool) {
	n.expandedByApp = true
	expanded = expanded && n.HasChildren()
	if n.expanded == expanded {
		return
	}
	n.expanded = expanded
	if n.expansionChanged != nil {
		n.expansionChanged(n)
	}
}

// Depth is how deep the node is in the tree, 0 for a root node; none until the TreeSliver that has the node
// has made it active.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.depth
func (n *TreeSliverNode[T]) Depth() foundation.Opt[int] { return n.depth }

// Parent is the node that holds this one, nil for a root node (or before the node was active).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.parent
func (n *TreeSliverNode[T]) Parent() *TreeSliverNode[T] { return n.parent }

// String describes the node.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNode.toString
func (n *TreeSliverNode[T]) String() string {
	depth := "null"
	if d, ok := n.depth.Get(); ok {
		depth = fmt.Sprint(d)
		if d == 0 {
			depth = "root"
		}
	}
	kind := "leaf"
	if len(n.children) > 0 {
		kind = fmt.Sprintf("parent, expanded: %v", n.expanded)
	}
	return fmt.Sprintf("TreeSliverNode: %v, depth: %s, %s", n.content, depth, kind)
}

// TreeSliverNodeBuilder builds the widget of a node. The animation style is the one of the tree's toggle
// animation, for the widgets that show whether the node is expanded.
//
// Deviation: Flutter's TreeSliverNode<Object?> is the node of any tree; Go has no covariance, so the builder is
// generic in the content (SPEC §12.10).
//
// Flutter: widgets/sliver_tree.dart TreeSliverNodeBuilder
type TreeSliverNodeBuilder[T any] func(bc BuildContext, node *TreeSliverNode[T], animationStyle animation.AnimationStyle) Widget

// TreeSliverRowExtentBuilder gives the height of the row of a node.
//
// Flutter: widgets/sliver_tree.dart TreeSliverRowExtentBuilder
type TreeSliverRowExtentBuilder[T any] func(node *TreeSliverNode[T], dimensions rendering.SliverLayoutDimensions) float64

// TreeSliverNodeCallback is called with a node, for example when it is toggled.
//
// Flutter: widgets/sliver_tree.dart TreeSliverNodeCallback
type TreeSliverNodeCallback[T any] func(node *TreeSliverNode[T])

// treeSliverStateMixin is what a TreeSliverController needs of the state of its tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliverStateMixin
type treeSliverStateMixin[T any] interface {
	isExpanded(node *TreeSliverNode[T]) bool
	isActive(node *TreeSliverNode[T]) bool
	toggleNode(node *TreeSliverNode[T])
	collapseAll()
	expandAll()
	getNodeFor(content T) *TreeSliverNode[T]
	getActiveIndexFor(node *TreeSliverNode[T]) foundation.Opt[int]
}

// TreeSliverController expands and collapses the nodes of a TreeSliver, and answers questions about them.
// Give it to a TreeSliver in TreeSliverOpts.Controller, or find the one of the tree above a context with
// TreeSliverControllerOf. A controller can belong to one tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController
type TreeSliverController[T any] struct {
	state treeSliverStateMixin[T]
}

// NewTreeSliverController creates a controller that is not attached to a tree yet.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.new
func NewTreeSliverController[T any]() *TreeSliverController[T] { return &TreeSliverController[T]{} }

func (c *TreeSliverController[T]) attached() treeSliverStateMixin[T] {
	if c.state == nil {
		panic(errors.New(tagTreeSliverControllerNotAttached, "the TreeSliverController is not attached to a TreeSliver"))
	}
	return c.state
}

// IsExpanded is whether the node is expanded; a node that is not in the tree is not.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.isExpanded
func (c *TreeSliverController[T]) IsExpanded(node *TreeSliverNode[T]) bool {
	return c.attached().isExpanded(node)
}

// IsActive is whether the node is shown now: its parents are expanded (or animating).
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.isActive
func (c *TreeSliverController[T]) IsActive(node *TreeSliverNode[T]) bool {
	return c.attached().isActive(node)
}

// GetNodeFor finds the node that holds the content, looking through the tree one level after the other; nil
// when there is none.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.getNodeFor
func (c *TreeSliverController[T]) GetNodeFor(content T) *TreeSliverNode[T] {
	return c.attached().getNodeFor(content)
}

// ToggleNode expands the node, or collapses it. The node must be active (TreeSliverNodeNotActive).
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.toggleNode
func (c *TreeSliverController[T]) ToggleNode(node *TreeSliverNode[T]) { c.attached().toggleNode(node) }

// ExpandNode expands the node when it is collapsed.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.expandNode
func (c *TreeSliverController[T]) ExpandNode(node *TreeSliverNode[T]) {
	state := c.attached()
	if !node.expanded {
		state.toggleNode(node)
	}
}

// CollapseNode collapses the node when it is expanded.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.collapseNode
func (c *TreeSliverController[T]) CollapseNode(node *TreeSliverNode[T]) {
	state := c.attached()
	if node.expanded {
		state.toggleNode(node)
	}
}

// ExpandAll expands every node of the tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.expandAll
func (c *TreeSliverController[T]) ExpandAll() { c.attached().expandAll() }

// CollapseAll collapses every node of the tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.collapseAll
func (c *TreeSliverController[T]) CollapseAll() { c.attached().collapseAll() }

// GetActiveIndexFor is the index of the node in the list of the nodes that are shown; none when it is not shown.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.getActiveIndexFor
func (c *TreeSliverController[T]) GetActiveIndexFor(node *TreeSliverNode[T]) foundation.Opt[int] {
	return c.attached().getActiveIndexFor(node)
}

// TreeSliverControllerOf is the controller of the closest TreeSliver of content type T above the context. It
// panics with TreeSliverControllerNotFound when there is none; use a Builder to get a context below the
// TreeSliver when the context is the one of the widget that builds it.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.of
func TreeSliverControllerOf[T any](bc BuildContext) *TreeSliverController[T] {
	if c, ok := TreeSliverControllerMaybeOf[T](bc); ok {
		return c
	}
	panic(errors.New(tagTreeSliverControllerNotFound,
		"TreeSliverControllerOf was called with a context that does not contain a TreeSliver of that content type: "+
			"use a Builder to get a context under the TreeSliver, or split the build function into widgets",
		"context", bc.DescribeElement("The context used was")))
}

// TreeSliverControllerMaybeOf is the controller of the closest TreeSliver of content type T above the context, or
// false.
//
// Flutter: widgets/sliver_tree.dart TreeSliverController.maybeOf
func TreeSliverControllerMaybeOf[T any](bc BuildContext) (*TreeSliverController[T], bool) {
	st, ok := FindAncestorStateOfType[*treeSliverState[T]](bc)
	if !ok {
		return nil, false
	}
	return st.controller(), true
}

// TreeSliverDefaultAnimationCurve is the curve of the toggle animation: linear.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.defaultAnimationCurve
var TreeSliverDefaultAnimationCurve animation.Curve = animation.Curves.Linear

// TreeSliverDefaultAnimationDuration is the duration of the toggle animation: 150 ms.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.defaultAnimationDuration
const TreeSliverDefaultAnimationDuration = 150 * time.Millisecond

// TreeSliverDefaultToggleAnimationStyle is the style of the toggle animation when the tree has none.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.defaultToggleAnimationStyle
var TreeSliverDefaultToggleAnimationStyle = animation.AnimationStyle{
	Curve: TreeSliverDefaultAnimationCurve, Duration: foundation.Some(TreeSliverDefaultAnimationDuration),
}

// TreeSliverOpts are the optional parameters of NewTreeSliver.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.new
type TreeSliverOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TreeNodeBuilder builds the widget of a node; the default is TreeSliverDefaultTreeNodeBuilder.
	TreeNodeBuilder TreeSliverNodeBuilder[T]
	// TreeRowExtentBuilder gives the height of a row; the default is 40.
	TreeRowExtentBuilder TreeSliverRowExtentBuilder[T]
	// Controller expands and collapses the nodes; the tree makes its own when there is none.
	Controller *TreeSliverController[T]
	// OnNodeToggle is called when a node is toggled, after its state changed.
	OnNodeToggle TreeSliverNodeCallback[T]
	// NodeKey gives the key of the content of a node: GetNodeFor and IsExpanded find the node whose key is
	// the key of the content, and a tree that is built again with new node objects keeps the expansion of the
	// nodes whose keys were in the tree before. The keys must be comparable. Without it, contents are compared
	// with reflect.DeepEqual and the new nodes keep their own state.
	//
	// Deviation: Flutter compares the contents with ==, and its nodes are kept by the app (SPEC §12.10).
	NodeKey func(content T) any
	// ToggleAnimationStyle is the curve and the duration of the animation of a toggle; AnimationStyleNoAnimation, or
	// a zero duration, skips it. The default is TreeSliverDefaultToggleAnimationStyle.
	ToggleAnimationStyle foundation.Opt[animation.AnimationStyle]
	// Indentation is how far each level is indented; the default is TreeSliverIndentationTypeStandard.
	Indentation foundation.Opt[rendering.TreeSliverIndentationType]
	// AddAutomaticKeepAlives is whether to wrap each row in an AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each row in a RepaintBoundary; the default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each row in an IndexedSemantics; the default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// SemanticIndexCallback gives the semantic index of a row; the default is its index.
	SemanticIndexCallback SemanticIndexCallback
	// SemanticIndexOffset is added to the semantic index of every row.
	SemanticIndexOffset int
	// FindChildIndexCallback finds the index of the row that has a key.
	FindChildIndexCallback ChildIndexGetter
}

// TreeSliver is a sliver that shows a tree of nodes as a list of rows, each indented by its depth. A parent
// node expands to show its children and collapses to hide them, with an animation; give it to a
// CustomScrollView. A TreeSliverController does the toggling from code, and TreeSliverWrapChildToToggleNode
// makes a widget that toggles its node when it is tapped.
//
// Flutter: widgets/sliver_tree.dart TreeSliver
type TreeSliver[T any] struct {
	StatefulWidget
	tree []*TreeSliverNode[T]
	o    TreeSliverOpts[T]
}

// NewTreeSliver creates a sliver that shows the nodes of the tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.new
func NewTreeSliver[T any](tree []*TreeSliverNode[T], opts ...TreeSliverOpts[T]) *TreeSliver[T] {
	o := firstOpts(opts)
	if o.TreeNodeBuilder == nil {
		o.TreeNodeBuilder = TreeSliverDefaultTreeNodeBuilder[T]
	}
	if o.TreeRowExtentBuilder == nil {
		o.TreeRowExtentBuilder = TreeSliverDefaultTreeRowExtentBuilder[T]
	}
	if o.SemanticIndexCallback == nil {
		o.SemanticIndexCallback = func(_ Widget, localIndex int) foundation.Opt[int] { return foundation.Some(localIndex) }
	}
	return &TreeSliver[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), tree: tree, o: o}
}

// Tree is the root nodes of the tree.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.tree
func (w *TreeSliver[T]) Tree() []*TreeSliverNode[T] { return w.tree }

// Controller is the controller that was given, if any.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.controller
func (w *TreeSliver[T]) Controller() *TreeSliverController[T] { return w.o.Controller }

// Indentation is how far each level is indented.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.indentation
func (w *TreeSliver[T]) Indentation() rendering.TreeSliverIndentationType {
	return w.o.Indentation.OrElse(rendering.TreeSliverIndentationTypeStandard)
}

// ToggleAnimationStyle is the style of the toggle animation, if one was given.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.toggleAnimationStyle
func (w *TreeSliver[T]) ToggleAnimationStyle() foundation.Opt[animation.AnimationStyle] {
	return w.o.ToggleAnimationStyle
}

// TreeSliverWrapChildToToggleNode wraps the widget so that a tap on it toggles the node.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.wrapChildToToggleNode
func TreeSliverWrapChildToToggleNode[T any](node *TreeSliverNode[T], child Widget) Widget {
	return NewBuilder(func(bc BuildContext) Widget {
		return NewGestureDetector(child, GestureDetectorOpts{OnTap: func() {
			TreeSliverControllerOf[T](bc).ToggleNode(node)
		}})
	})
}

// TreeSliverDefaultTreeRowExtentBuilder gives every row a height of 40.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.defaultTreeRowExtentBuilder
func TreeSliverDefaultTreeRowExtentBuilder[T any](*TreeSliverNode[T], rendering.SliverLayoutDimensions) float64 {
	return defaultTreeRowExtent
}

// TreeSliverDefaultTreeNodeBuilder builds a row with an arrow that turns when the node expands, which toggles
// the node when tapped, and the text of the node's content.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.defaultTreeNodeBuilder
func TreeSliverDefaultTreeNodeBuilder[T any](bc BuildContext, node *TreeSliverNode[T], toggleAnimationStyle animation.AnimationStyle) Widget {
	duration := toggleAnimationStyle.Duration.OrElse(TreeSliverDefaultAnimationDuration)
	curve := toggleAnimationStyle.Curve
	if curve == nil {
		curve = TreeSliverDefaultAnimationCurve
	}
	index := TreeSliverControllerOf[T](bc).GetActiveIndexFor(node).Value()
	var arrow Widget
	if node.HasChildren() {
		turns := 0.0
		if node.expanded {
			turns = 0.25
		}
		// A unicode right-facing arrow, as a glyph of the Icon font.
		arrow = NewAnimatedRotation(turns, duration, NewIcon(NewIconData(0x25BA), IconOpts{Size: foundation.Some(14.0)}),
			AnimatedRotationOpts{Key: foundation.NewValueKey(index), Curve: curve})
	}
	return NewPadding(painting.NewEdgeInsetsAll(8.0), NewRow([]Widget{
		TreeSliverWrapChildToToggleNode(node, NewSizedBoxSquare(foundation.Some(30.0), arrow)),
		NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(8.0)}),
		NewText(fmt.Sprint(node.content)),
	}))
}

// CreateState creates the state.
//
// Flutter: widgets/sliver_tree.dart TreeSliver.createState
func (w *TreeSliver[T]) CreateState() State { return &treeSliverState[T]{} }

// treeAnimationRecord is the animation of the children of a node that expands or collapses.
type treeAnimationRecord struct {
	controller *animation.AnimationController
	animation  *animation.CurvedAnimation
	// key keeps track of the life of the animation in the render object, since the indices can change at any time.
	key *foundation.UniqueKey
}

type treeSliverState[T any] struct {
	StateOf[*TreeSliver[T]]
	TickerProvider
	treeController *TreeSliverController[T]
	activeNodes    []*TreeSliverNode[T]

	currentAnimationForParent map[*TreeSliverNode[T]]*treeAnimationRecord
	activeAnimations          map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation

	// notify is what the nodes call when their children change.
	notify func()
	// notifyExpansion is what the nodes call when the app sets their expansion.
	notifyExpansion func(*TreeSliverNode[T])
	// toggling is whether toggleNode is running: it makes the active nodes again by itself.
	toggling bool
}

var _ treeSliverStateMixin[any] = (*treeSliverState[any])(nil)

func (s *treeSliverState[T]) controller() *TreeSliverController[T] { return s.treeController }

// childrenChanged is called by a node whose children changed: the active nodes are made again, and the
// animations of nodes that left the tree or lost their children end. Flutter's tree shows the change when it
// is built again.
func (s *treeSliverState[T]) childrenChanged() {
	if s.toggling || !s.Mounted() {
		return
	}
	s.SetState(func() {
		s.unpackActiveNodes()
		s.dropStaleAnimations()
		s.updateActiveAnimations()
	})
}

// expansionChanged is called by a node whose expansion the app set: when the node is shown, its animation (if
// it has one) ends, and the active nodes are made again at once, with no animation. A node that is not shown
// has nothing to change until its ancestors open, and the active nodes are made again then.
func (s *treeSliverState[T]) expansionChanged(node *TreeSliverNode[T]) {
	if s.toggling || !s.Mounted() || !s.isActive(node) {
		return
	}
	s.SetState(func() {
		if record := s.currentAnimationForParent[node]; record != nil {
			record.animation.Dispose()
			record.controller.Dispose()
			delete(s.currentAnimationForParent, node)
		}
		s.unpackActiveNodes()
		s.dropStaleAnimations()
		s.updateActiveAnimations()
	})
}

// dropStaleAnimations ends the animations of the nodes that are not in the tree any more, and of those that
// have no children to animate.
func (s *treeSliverState[T]) dropStaleAnimations() {
	for node, record := range s.currentAnimationForParent {
		if len(node.children) > 0 && slices.Contains(s.activeNodes, node) {
			continue
		}
		record.animation.Dispose()
		record.controller.Dispose()
		delete(s.currentAnimationForParent, node)
	}
}

func (s *treeSliverState[T]) shouldUnpackNode(node *TreeSliverNode[T]) bool {
	if len(node.children) == 0 {
		return false // no children to unpack
	}
	if s.currentAnimationForParent[node] != nil {
		// Expanding or collapsing, the children are still active.
		return true
	}
	return node.expanded
}

func (s *treeSliverState[T]) unpackActiveNodes() {
	s.activeNodes = s.activeNodes[:0]
	s.unpack(0, s.Widget().tree, nil)
}

func (s *treeSliverState[T]) unpack(depth int, nodes []*TreeSliverNode[T], parent *TreeSliverNode[T]) {
	for _, node := range nodes {
		node.depth = foundation.Some(depth)
		node.parent = parent
		node.changed = s.notify
		node.expansionChanged = s.notifyExpansion
		s.activeNodes = append(s.activeNodes, node)
		if s.shouldUnpackNode(node) {
			s.unpack(depth+1, node.children, node)
		}
	}
}

func (s *treeSliverState[T]) InitState() {
	s.notify = s.childrenChanged
	s.notifyExpansion = s.expansionChanged
	s.currentAnimationForParent = map[*TreeSliverNode[T]]*treeAnimationRecord{}
	s.activeAnimations = map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation{}
	s.unpackActiveNodes()
	w := s.Widget()
	s.keepExpansion(nil, w.tree) // the tree takes the nodes in: the marks of SetExpanded end
	if foundation.DebugMode && w.o.Controller != nil && w.o.Controller.state != nil {
		panic(errors.New(tagTreeSliverControllerAttached,
			"the provided TreeSliverController is already associated with another TreeSliver; a TreeSliverController can only be associated with one TreeSliver"))
	}
	s.treeController = w.o.Controller
	if s.treeController == nil {
		s.treeController = NewTreeSliverController[T]()
	}
	s.treeController.state = s
}

func (s *treeSliverState[T]) DidUpdateWidget(old *TreeSliver[T]) {
	w := s.Widget()
	switch {
	case old.o.Controller == nil && w.o.Controller != nil:
		// A controller was given: the one that the tree made is let go.
		s.treeController.state = nil
		s.treeController = w.o.Controller
		s.treeController.state = s
	case old.o.Controller != nil && w.o.Controller == nil:
		// The controller that had been given was taken away: the tree makes one.
		old.o.Controller.state = nil
		s.treeController = NewTreeSliverController[T]()
		s.treeController.state = s
	case old.o.Controller != w.o.Controller:
		s.treeController.state = nil
		s.treeController = w.o.Controller
		s.treeController.state = s
	}
	s.keepExpansion(old.tree, w.tree)
	s.unpackActiveNodes()
	s.dropStaleAnimations()
	s.updateActiveAnimations()
}

func (s *treeSliverState[T]) Dispose() {
	s.treeController.state = nil
	for _, record := range s.currentAnimationForParent {
		record.animation.Dispose()
		record.controller.Dispose()
	}
}

func (s *treeSliverState[T]) Build(BuildContext) Widget {
	w := s.Widget()
	toggleStyle := w.o.ToggleAnimationStyle.OrElse(TreeSliverDefaultToggleAnimationStyle)
	delegate := NewSliverChildBuilderDelegate(func(bc BuildContext, index int) Widget {
		node := s.activeNodes[index]
		child := w.o.TreeNodeBuilder(bc, node, toggleStyle)
		if w.o.AddRepaintBoundaries.OrElse(true) {
			child = NewRepaintBoundary(child)
		}
		if w.o.AddSemanticIndexes.OrElse(true) {
			if semanticIndex, ok := w.o.SemanticIndexCallback(child, index).Get(); ok {
				child = NewIndexedSemantics(semanticIndex+w.o.SemanticIndexOffset, child)
			}
		}
		return newTreeNodeParentDataWidget(node.depth.Value(), child)
	}, SliverChildBuilderDelegateOpts{
		FindChildIndexCallback: w.o.FindChildIndexCallback, ChildCount: foundation.Some(len(s.activeNodes)),
		AddAutomaticKeepAlives: w.o.AddAutomaticKeepAlives,
		AddRepaintBoundaries:   foundation.Some(false), // added above
		AddSemanticIndexes:     foundation.Some(false), // added above
	})
	return &sliverTree{
		SliverMultiBoxAdaptorWidget: NewSliverMultiBoxAdaptorWidgetBase(delegate),
		itemExtentBuilder: func(index int, dimensions rendering.SliverLayoutDimensions) foundation.Opt[float64] {
			return foundation.Some(w.o.TreeRowExtentBuilder(s.activeNodes[index], dimensions))
		},
		activeAnimations: s.activeAnimations,
		indentation:      w.Indentation().Value(),
	}
}

// contentEqual is Dart's == on two contents. Deviation: reflect.DeepEqual, since Go can not compare every T.
func contentEqual[T any](a, b T) bool { return reflect.DeepEqual(a, b) }

// sameContent compares two contents by the NodeKey of the tree, or by contentEqual without one.
func (s *treeSliverState[T]) sameContent(a, b T) bool {
	if key := s.Widget().o.NodeKey; key != nil {
		return key(a) == key(b)
	}
	return contentEqual(a, b)
}

// keepExpansion gives the new nodes that have the key of a node of the old tree the expansion of that node, when
// the tree has a NodeKey. The animations of old nodes that are not in the new tree end.
func (s *treeSliverState[T]) keepExpansion(old, tree []*TreeSliverNode[T]) {
	key := s.Widget().o.NodeKey
	if key == nil {
		return
	}
	before := map[any]*TreeSliverNode[T]{}
	var collect func(nodes []*TreeSliverNode[T])
	collect = func(nodes []*TreeSliverNode[T]) {
		for _, n := range nodes {
			before[key(n.content)] = n
			collect(n.children)
		}
	}
	collect(old)
	var apply func(nodes []*TreeSliverNode[T])
	apply = func(nodes []*TreeSliverNode[T]) {
		for _, n := range nodes {
			if was := before[key(n.content)]; was != nil && was != n && n.HasChildren() && !n.expandedByApp {
				n.expanded = was.expanded
			}
			n.expandedByApp = false // the tree has taken the node in
			apply(n.children)
		}
	}
	apply(tree)
}

func (s *treeSliverState[T]) isExpanded(node *TreeSliverNode[T]) bool {
	found := s.searchNode(node.content, s.Widget().tree)
	return found != nil && found.expanded
}

func (s *treeSliverState[T]) isActive(node *TreeSliverNode[T]) bool {
	return slices.Contains(s.activeNodes, node)
}

func (s *treeSliverState[T]) getNodeFor(content T) *TreeSliverNode[T] {
	return s.searchNode(content, s.Widget().tree)
}

// searchNode looks for the content one level after the other.
func (s *treeSliverState[T]) searchNode(content T, tree []*TreeSliverNode[T]) *TreeSliverNode[T] {
	for len(tree) > 0 {
		var nextDepth []*TreeSliverNode[T]
		for _, node := range tree {
			if s.sameContent(node.content, content) {
				return node
			}
			nextDepth = append(nextDepth, node.children...)
		}
		tree = nextDepth
	}
	return nil
}

func (s *treeSliverState[T]) getActiveIndexFor(node *TreeSliverNode[T]) foundation.Opt[int] {
	if i := slices.Index(s.activeNodes, node); i >= 0 {
		return foundation.Some(i)
	}
	return foundation.None[int]()
}

func (s *treeSliverState[T]) expandAll() {
	var activeNodesToExpand []*TreeSliverNode[T]
	s.setExpandedBelow(s.Widget().tree, true, &activeNodesToExpand)
	for _, node := range slices.Backward(activeNodesToExpand) {
		s.toggleNode(node)
	}
}

func (s *treeSliverState[T]) collapseAll() {
	var activeNodesToCollapse []*TreeSliverNode[T]
	s.setExpandedBelow(s.Widget().tree, false, &activeNodesToCollapse)
	for _, node := range slices.Backward(activeNodesToCollapse) {
		s.toggleNode(node)
	}
}

// setExpandedBelow sets every node below the tree to the state. The nodes that are active are collected, to be toggled
// when the hidden ones are done; the hidden ones are changed in place.
func (s *treeSliverState[T]) setExpandedBelow(tree []*TreeSliverNode[T], expanded bool, active *[]*TreeSliverNode[T]) {
	for _, node := range tree {
		if len(node.children) == 0 {
			continue
		}
		s.setExpandedBelow(node.children, expanded, active)
		if node.expanded == expanded {
			continue
		}
		if s.isActive(node) {
			*active = append(*active, node)
		} else {
			node.expanded = expanded
		}
	}
}

func (s *treeSliverState[T]) updateActiveAnimations() {
	// The indices of the animations change whenever nodes expand or collapse: they are made again each
	// time the active nodes change.
	clear(s.activeAnimations)
	for node, record := range s.currentAnimationForParent {
		if len(node.children) == 0 {
			continue
		}
		leadingChildIndex := slices.Index(s.activeNodes, node) + 1
		s.activeAnimations[record.key] = rendering.TreeSliverNodesAnimation{
			FromIndex: leadingChildIndex, ToIndex: leadingChildIndex + len(node.children) - 1, Value: record.animation.Value(),
		}
	}
}

func (s *treeSliverState[T]) toggleNode(node *TreeSliverNode[T]) {
	if foundation.DebugMode && !slices.Contains(s.activeNodes, node) {
		panic(errors.New(tagTreeSliverNodeNotActive, "only the nodes that are shown can be toggled", "node", node.String()))
	}
	if !node.HasChildren() {
		return // no state to change
	}
	w := s.Widget()
	s.SetState(func() {
		node.expanded = !node.expanded
		s.toggling = true
		defer func() { s.toggling = false }()
		if w.o.OnNodeToggle != nil {
			// The app adds the children of a lazy node here.
			w.o.OnNodeToggle(node)
		}
		if len(node.children) == 0 {
			// A lazy node whose children are not there (yet): there is nothing to animate. They show when
			// the app sets them.
			if record := s.currentAnimationForParent[node]; record != nil {
				record.animation.Dispose()
				record.controller.Dispose()
				delete(s.currentAnimationForParent, node)
			}
			s.unpackActiveNodes()
			s.updateActiveAnimations()
			return
		}
		if record := s.currentAnimationForParent[node]; record != nil {
			// Let go of the old animation if the node was already animating.
			record.animation.Dispose()
		}

		// With no animation, or a duration of zero, the active nodes are updated at once. A tree whose children
		// are no longer active after a zero duration animation would be wrong.
		if style, ok := w.o.ToggleAnimationStyle.Get(); ok && style.Duration == foundation.Some(time.Duration(0)) {
			s.unpackActiveNodes()
			return
		}

		var controller *animation.AnimationController
		if record := s.currentAnimationForParent[node]; record != nil {
			controller = record.controller
		} else {
			value := 1.0
			if node.expanded {
				value = 0.0
			}
			controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
				Value:    foundation.Some(value),
				Duration: foundation.Some(w.o.ToggleAnimationStyle.OrElse(TreeSliverDefaultToggleAnimationStyle).Duration.OrElse(TreeSliverDefaultAnimationDuration)),
			})
			controller.AddStatusListener(func(status animation.AnimationStatus) {
				if status != animation.AnimationStatusDismissed && status != animation.AnimationStatusCompleted {
					return
				}
				record := s.currentAnimationForParent[node]
				record.animation.Dispose()
				record.controller.Dispose()
				delete(s.currentAnimationForParent, node)
				s.updateActiveAnimations()
				// A node that collapsed: the nodes that left the tree are removed from the active ones.
				if !node.expanded {
					s.unpackActiveNodes()
				}
			})
			controller.AddListener(func() {
				s.SetState(s.updateActiveAnimations)
			})
		}

		if status := controller.Status(); status == animation.AnimationStatusForward || status == animation.AnimationStatusReverse {
			controller.Stop() // an animation in progress is interrupted
		}

		curve := w.o.ToggleAnimationStyle.OrElse(TreeSliverDefaultToggleAnimationStyle).Curve
		if curve == nil {
			curve = TreeSliverDefaultAnimationCurve
		}
		s.currentAnimationForParent[node] = &treeAnimationRecord{
			controller: controller, animation: animation.NewCurvedAnimation(controller, curve), key: foundation.NewUniqueKey(),
		}
		if node.expanded {
			s.unpackActiveNodes()
			controller.Forward()
		} else {
			controller.Reverse()
		}
	})
}

// treeNodeParentDataWidget gives a row its depth.
//
// Flutter: widgets/sliver_tree.dart _TreeNodeParentDataWidget
type treeNodeParentDataWidget struct {
	ParentDataWidgetOf[*rendering.TreeSliverNodeParentData]
	depth int
}

func newTreeNodeParentDataWidget(depth int, child Widget) *treeNodeParentDataWidget {
	if foundation.DebugMode && depth < 0 {
		panic(errors.New(tagTreeSliverDepthNegative, "the depth of a tree node must not be negative", "depth", depth))
	}
	return &treeNodeParentDataWidget{NewParentDataWidgetBase[*rendering.TreeSliverNodeParentData](child), depth}
}

func (w *treeNodeParentDataWidget) ApplyParentData(renderObject rendering.RenderObject) {
	pd := renderObject.ParentData().(*rendering.TreeSliverNodeParentData)
	if pd.Depth != w.depth {
		pd.Depth = w.depth
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

func (*treeNodeParentDataWidget) DebugTypicalAncestorWidgetClass() string { return "sliverTree" }

// sliverTree is the sliver of a TreeSliver: a varied extent list that makes a RenderTreeSliver.
//
// Flutter: widgets/sliver_tree.dart _SliverTree
type sliverTree struct {
	SliverMultiBoxAdaptorWidget
	itemExtentBuilder rendering.ItemExtentBuilder
	activeAnimations  map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation
	indentation       float64
}

func (w *sliverTree) CreateElement() Element { return NewSliverMultiBoxAdaptorElement(w, false) }

func (w *sliverTree) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderTreeSliver(bc.(rendering.RenderSliverBoxChildManager), w.itemExtentBuilder, w.activeAnimations, w.indentation)
}

func (w *sliverTree) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	r := ro.(*rendering.RenderTreeSliver)
	r.SetItemExtentBuilder(w.itemExtentBuilder)
	r.SetActiveAnimations(w.activeAnimations)
	r.SetIndentation(w.indentation)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTreeSliverControllerAttached    = "TreeSliverControllerAttached"
	tagTreeSliverControllerNotAttached = "TreeSliverControllerNotAttached"
	tagTreeSliverControllerNotFound    = "TreeSliverControllerNotFound"
	tagTreeSliverDepthNegative         = "TreeSliverDepthNegative"
	tagTreeSliverNodeNotActive         = "TreeSliverNodeNotActive"
)
