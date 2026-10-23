package semantics

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsOwner manages a semantics tree: it knows the nodes, collects the
// ones that changed, and sends the update to the engine. It is a Listenable
// that notifies after each update.
//
// Flutter: semantics/semantics.dart SemanticsOwner
type SemanticsOwner struct {
	foundation.ChangeNotifier
	onSemanticsUpdate SemanticsUpdateCallback
	onSendEvent       func(map[string]any)

	dirtyNodes           map[*SemanticsNode]struct{}
	nodes                map[int]*SemanticsNode
	detachedNodes        map[*SemanticsNode]struct{}
	traversalParentNodes map[any]*SemanticsNode
	traversalChildNodes  map[any]*nodeSet
}

// SemanticsOwnerOpts are the optional parameters of NewSemanticsOwner.
//
// Flutter: semantics/semantics.dart SemanticsOwner.new
type SemanticsOwnerOpts struct {
	// OnSendEvent receives the events that nodes send to the platform, as the
	// maps that Flutter sends on its accessibility channel.
	OnSendEvent func(event map[string]any)
}

// NewSemanticsOwner creates an owner that gives its updates to the callback.
//
// Flutter: semantics/semantics.dart SemanticsOwner.new
func NewSemanticsOwner(onSemanticsUpdate SemanticsUpdateCallback, opts ...SemanticsOwnerOpts) *SemanticsOwner {
	var o SemanticsOwnerOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return &SemanticsOwner{
		onSemanticsUpdate: onSemanticsUpdate, onSendEvent: o.OnSendEvent,
		dirtyNodes: map[*SemanticsNode]struct{}{}, nodes: map[int]*SemanticsNode{},
		detachedNodes: map[*SemanticsNode]struct{}{}, traversalParentNodes: map[any]*SemanticsNode{},
		traversalChildNodes: map[any]*nodeSet{},
	}
}

// RootSemanticsNode is the root of the tree, the node with the ID 0, or nil.
//
// Flutter: semantics/semantics.dart SemanticsOwner.rootSemanticsNode
func (o *SemanticsOwner) RootSemanticsNode() *SemanticsNode { return o.nodes[0] }

// GetSemanticsNode is the node with the ID, or nil.
//
// Flutter: semantics/semantics.dart SemanticsOwner.getSemanticsNode
func (o *SemanticsOwner) GetSemanticsNode(id int) *SemanticsNode { return o.nodes[id] }

// Dispose forgets the nodes.
//
// Flutter: semantics/semantics.dart SemanticsOwner.dispose
func (o *SemanticsOwner) Dispose() {
	clear(o.dirtyNodes)
	clear(o.nodes)
	clear(o.detachedNodes)
	clear(o.traversalChildNodes)
	clear(o.traversalParentNodes)
	o.ChangeNotifier.Dispose()
}

// markDirtyNode is how a node enters the list of dirty nodes; the order of
// entry does not matter, the update sorts by depth.
func (o *SemanticsOwner) dirtyList() []*SemanticsNode {
	out := make([]*SemanticsNode, 0, len(o.dirtyNodes))
	for n := range o.dirtyNodes {
		out = append(out, n)
	}
	slices.SortStableFunc(out, func(a, b *SemanticsNode) int { return a.id - b.id })
	return out
}

// SendSemanticsUpdate sends the changes since the last update to the callback
// and notifies the listeners. In debug builds it panics with the tag
// SemanticsInvisibleNodes if the tree has an invisible node.
//
// Flutter: semantics/semantics.dart SemanticsOwner.sendSemanticsUpdate
func (o *SemanticsOwner) SendSemanticsUpdate() {
	if foundation.DebugMode {
		o.checkInvisibleNodes()
	}
	if len(o.dirtyNodes) == 0 {
		return
	}
	customActionIDs := map[int]struct{}{}
	var visited []*SemanticsNode
	for len(o.dirtyNodes) > 0 {
		var local []*SemanticsNode
		for _, n := range o.dirtyList() {
			if _, detached := o.detachedNodes[n]; !detached {
				local = append(local, n)
			}
		}
		clear(o.dirtyNodes)
		clear(o.detachedNodes)
		slices.SortStableFunc(local, func(a, b *SemanticsNode) int { return a.depth - b.depth })
		visited = append(visited, local...)
		for _, node := range local {
			if node.IsPartOfNodeMerging() {
				if node.parent != nil && node.parent.IsPartOfNodeMerging() {
					node.parent.markDirty()
					node.dirty = false
				}
			}
			for key, old := range o.traversalParentNodes {
				if old == node {
					delete(o.traversalParentNodes, key)
				}
			}
			for key, set := range o.traversalChildNodes {
				set.remove(node)
				if set.len() == 0 {
					delete(o.traversalChildNodes, key)
				}
			}
			isParent, isChild := node.isTraversalParent(), node.isTraversalChild()
			if isParent {
				if foundation.DebugMode {
					if existing, ok := o.traversalParentNodes[node.traversalParentID]; ok && existing != node {
						panic(errors.New(tagSemanticsTraversalParentIdentifierDuplicated,
							"The traversalParentIdentifier must be unique. No two semantics nodes can share the same traversalParentIdentifier."))
					}
				}
				o.traversalParentNodes[node.traversalParentID] = node
			} else if isChild {
				set := o.traversalChildNodes[node.traversalChildID]
				if set == nil {
					set = &nodeSet{}
					o.traversalChildNodes[node.traversalChildID] = set
				}
				set.add(node)
			}
			if node.isTraversalChild() {
				if parentNode := o.traversalParentNodes[node.traversalChildID]; parentNode != nil && !slices.Contains(visited, parentNode) {
					parentNode.markDirty()
				}
			}
		}
	}
	slices.SortStableFunc(visited, func(a, b *SemanticsNode) int { return a.depth - b.depth })
	builder := ui.NewSemanticsUpdateBuilder()
	for _, node := range visited {
		if node.dirty && node.Attached() {
			node.addToUpdate(builder, customActionIDs)
		}
	}
	clear(o.dirtyNodes)
	ids := make([]int, 0, len(customActionIDs))
	for id := range customActionIDs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		action, _ := CustomSemanticsActionGetAction(id)
		override := -1
		if action.Action() != 0 {
			override = int(action.Action())
		}
		builder.UpdateCustomAction(ui.SemanticsCustomActionUpdate{ID: id, Label: action.Label(), Hint: action.Hint(), OverrideID: override})
	}
	o.onSemanticsUpdate(builder.Build())
	o.NotifyListeners()
}

func (o *SemanticsOwner) checkInvisibleNodes() {
	var invisible []*SemanticsNode
	var find SemanticsNodeVisitor
	find = func(node *SemanticsNode) bool {
		if node.rect.IsEmpty() {
			invisible = append(invisible, node)
		} else if !node.mergeAllDescendantsIntoThisNode {
			node.VisitChildren(find)
		}
		return true
	}
	root := o.RootSemanticsNode()
	if root != nil {
		if root.ChildrenCount() > 0 && root.rect.IsEmpty() {
			invisible = append(invisible, root)
		} else if !root.mergeAllDescendantsIntoThisNode {
			root.VisitChildren(find)
		}
	}
	if len(invisible) == 0 {
		return
	}
	msg := "Invisible SemanticsNodes should not be added to the tree.\n" +
		"The following invisible SemanticsNodes were added to the tree:\n"
	for _, n := range invisible {
		msg += n.String() + "\n"
		if n.parent != nil {
			msg += "which was added as a child of:\n  " + n.parent.String() + "\n"
		} else {
			msg += "which was added as the root SemanticsNode\n"
		}
	}
	msg += "An invisible SemanticsNode is one whose rect is not on screen hence not reachable for users, " +
		"and its semantic information is not merged into a visible parent.\n" +
		"An invisible SemanticsNode makes the accessibility experience confusing, " +
		"as it does not provide any visual indication when the user selects it via accessibility technologies.\n" +
		"Consider removing the above invisible SemanticsNodes if they were added by your " +
		"RenderObject.assembleSemanticsNode implementation, or filing a bug on GitHub:\n" +
		"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml"
	panic(errors.New(tagSemanticsInvisibleNodes, msg))
}

func (o *SemanticsOwner) getSemanticsActionHandlerForID(id int, action ui.SemanticsAction, args any) SemanticsActionHandler {
	result := o.nodes[id]
	if result == nil {
		return nil
	}
	if result.IsPartOfNodeMerging() && !result.canHandleAction(action, args) {
		var found *SemanticsNode
		result.visitDescendants(func(node *SemanticsNode) bool {
			if node.canHandleAction(action, args) {
				found = node
				return false
			}
			return true
		})
		result = found
	}
	if result == nil || !result.canHandleAction(action, args) {
		return nil
	}
	return result.actions[action]
}

// PerformAction performs the action on the node with the ID. A merged node
// passes it to the descendant that handles it. For showOnScreen, a node
// without a handler uses the callback it was created with.
//
// Flutter: semantics/semantics.dart SemanticsOwner.performAction
func (o *SemanticsOwner) PerformAction(id int, action ui.SemanticsAction, args any) {
	if handler := o.getSemanticsActionHandlerForID(id, action, args); handler != nil {
		handler(args)
		return
	}
	if action == ui.SemanticsActionShowOnScreen {
		if n := o.nodes[id]; n != nil && n.showOnScreen != nil {
			n.showOnScreen()
		}
	}
}

func (o *SemanticsOwner) getSemanticsActionHandlerForPosition(node *SemanticsNode, position ui.Offset, action ui.SemanticsAction, args any) SemanticsActionHandler {
	if node.transform != nil {
		inverse := node.transform.Clone()
		if inverse.Invert() == 0 {
			return nil
		}
		position = inverse.TransformPoint(position)
	}
	if !node.rect.Contains(position) {
		return nil
	}
	if node.mergeAllDescendantsIntoThisNode {
		if node.canHandleAction(action, args) {
			return node.actions[action]
		}
		var result *SemanticsNode
		node.visitDescendants(func(child *SemanticsNode) bool {
			if child.canHandleAction(action, args) {
				result = child
				return false
			}
			return true
		})
		if result != nil {
			return result.actions[action]
		}
		return nil
	}
	if node.HasChildren() {
		for i := len(node.children) - 1; i >= 0; i-- {
			if h := o.getSemanticsActionHandlerForPosition(node.children[i], position, action, args); h != nil {
				return h
			}
		}
	}
	return node.actions[action]
}

// PerformActionAt performs the action on the node at the position, in the
// coordinates of the root.
//
// Flutter: semantics/semantics.dart SemanticsOwner.performActionAt
func (o *SemanticsOwner) PerformActionAt(position ui.Offset, action ui.SemanticsAction, args any) {
	node := o.RootSemanticsNode()
	if node == nil {
		return
	}
	if h := o.getSemanticsActionHandlerForPosition(node, position, action, args); h != nil {
		h(args)
	}
}

// String is "SemanticsOwner#hash".
func (o *SemanticsOwner) String() string { return foundation.DescribeIdentity(o) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSemanticsInvisibleNodes                      = "SemanticsInvisibleNodes"
	tagSemanticsTraversalParentIdentifierDuplicated = "SemanticsTraversalParentIdentifierDuplicated"
)
