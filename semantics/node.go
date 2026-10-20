// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package semantics

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

const maxFrameworkAccessibilityIdentifier = 1<<16 - 1

// idCounter hands out the IDs of the nodes. Like Flutter's static counter it is
// shared by the process (SPEC §12.10).
var idCounter struct {
	mu   sync.Mutex
	last int
}

func generateNewID() int {
	idCounter.mu.Lock()
	defer idCounter.mu.Unlock()
	idCounter.last = (idCounter.last + 1) % maxFrameworkAccessibilityIdentifier
	return idCounter.last
}

// DebugResetSemanticsIDCounter makes the next node get the ID 1.
//
// Flutter: semantics/semantics.dart debugResetSemanticsIdCounter
func DebugResetSemanticsIDCounter() {
	idCounter.mu.Lock()
	defer idCounter.mu.Unlock()
	idCounter.last = 0
}

// emptyConfig holds the defaults of a node; it is never changed.
var emptyConfig = NewSemanticsConfiguration()

// SemanticsNode is a node of the semantics tree, which the engine hands to an
// assistive technology. The semantics owner updates the nodes, and the render
// objects fill them from their configurations.
//
// Flutter: semantics/semantics.dart SemanticsNode
type SemanticsNode struct {
	// Key identifies the node among its siblings, for the tests.
	//
	// Flutter: semantics/semantics.dart SemanticsNode.key
	Key foundation.Key
	// ParentSemanticsClipRect is the clip of the semantics of the parent in the
	// coordinates of this node.
	//
	// Flutter: semantics/semantics.dart SemanticsNode.parentSemanticsClipRect
	ParentSemanticsClipRect foundation.Opt[ui.Rect]
	// ParentPaintClipRect is the clip of the painting of the parent in the
	// coordinates of this node.
	//
	// Flutter: semantics/semantics.dart SemanticsNode.parentPaintClipRect
	ParentPaintClipRect foundation.Opt[ui.Rect]
	// IndexInParent is the index of the node among the children that scroll.
	//
	// Flutter: semantics/semantics.dart SemanticsNode.indexInParent
	IndexInParent foundation.Opt[int]
	// Tags are the tags of the node.
	//
	// Flutter: semantics/semantics.dart SemanticsNode.tags
	Tags map[SemanticsTag]struct{}

	id                              int
	showOnScreen                    VoidCallback
	transform                       *ui.Matrix4
	traversalChildTransform         *ui.Matrix4
	rect                            ui.Rect
	isMergedIntoParent              bool
	areUserActionsBlocked           bool
	mergeAllDescendantsIntoThisNode bool
	children                        []*SemanticsNode
	debugPreviousSnapshot           []*SemanticsNode
	dead                            bool
	owner                           *SemanticsOwner
	parent                          *SemanticsNode
	traversalParent                 *SemanticsNode
	depth                           int
	locale                          foundation.Opt[ui.Locale]
	dirty                           bool

	actions                map[ui.SemanticsAction]SemanticsActionHandler
	customSemanticsActions map[CustomSemanticsAction]VoidCallback
	actionsAsBits          ui.SemanticsAction
	flags                  ui.SemanticsFlags
	identifier             string
	traversalParentID      any
	traversalChildID       any
	attributedLabel        AttributedString
	attributedValue        AttributedString
	attributedIncreased    AttributedString
	attributedDecreased    AttributedString
	attributedHint         AttributedString
	tooltip                string
	hintOverrides          *SemanticsHintOverrides
	textDirection          ui.TextDirection
	sortKey                SemanticsSortKey
	textSelection          foundation.Opt[services.TextSelection]
	isMultiline            bool
	scrollChildCount       foundation.Opt[int]
	scrollIndex            foundation.Opt[int]
	scrollPosition         foundation.Opt[float64]
	scrollExtentMax        foundation.Opt[float64]
	scrollExtentMin        foundation.Opt[float64]
	platformViewID         foundation.Opt[int]
	maxValueLength         foundation.Opt[int]
	currentValueLength     foundation.Opt[int]
	headingLevel           int
	linkURL                foundation.Opt[string]
	role                   ui.SemanticsRole
	controlsNodes          []string
	minValue               foundation.Opt[string]
	maxValue               foundation.Opt[string]
	validationResult       ui.SemanticsValidationResult
	hitTestBehavior        ui.SemanticsHitTestBehavior
	inputType              ui.SemanticsInputType
}

// SemanticsNodeOpts are the optional parameters of NewSemanticsNode.
//
// Flutter: semantics/semantics.dart SemanticsNode.new
type SemanticsNodeOpts struct {
	// Key identifies the node among its siblings.
	Key foundation.Key
	// ShowOnScreen is called when an assistive technology asks to show the node
	// and no handler of the configuration does it.
	ShowOnScreen VoidCallback
}

func newNode(id int, opts SemanticsNodeOpts) *SemanticsNode {
	n := &SemanticsNode{Key: opts.Key, id: id, showOnScreen: opts.ShowOnScreen}
	n.actions = emptyConfig.actions
	n.customSemanticsActions = emptyConfig.customSemanticsActions
	n.hitTestBehavior = ui.SemanticsHitTestBehaviorDefer
	return n
}

// NewSemanticsNode creates a node with a new ID.
//
// Flutter: semantics/semantics.dart SemanticsNode.new
func NewSemanticsNode(opts ...SemanticsNodeOpts) *SemanticsNode {
	var o SemanticsNodeOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return newNode(generateNewID(), o)
}

// NewSemanticsNodeRoot creates the root node, which has the ID 0, and attaches
// it to the owner.
//
// Flutter: semantics/semantics.dart SemanticsNode.root
func NewSemanticsNodeRoot(owner *SemanticsOwner, opts ...SemanticsNodeOpts) *SemanticsNode {
	var o SemanticsNodeOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	n := newNode(0, o)
	n.Attach(owner)
	return n
}

// ID is the ID of the node; the engine addresses the node with it.
//
// Flutter: semantics/semantics.dart SemanticsNode.id
func (n *SemanticsNode) ID() int { return n.id }

// Transform is the transform from the coordinates of the node to those of its
// parent; nil is the identity.
//
// Flutter: semantics/semantics.dart SemanticsNode.transform
func (n *SemanticsNode) Transform() *ui.Matrix4 { return n.transform }

// SetTransform sets the transform; the identity is stored as nil.
//
// Flutter: semantics/semantics.dart SemanticsNode.transform
func (n *SemanticsNode) SetTransform(value *ui.Matrix4) {
	if !transformEquals(n.transform, value) {
		if value == nil || value.IsIdentity() {
			n.transform = nil
		} else {
			n.transform = value
		}
		n.markDirty()
	}
}

func transformEquals(a, b *ui.Matrix4) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil:
		return b.IsIdentity()
	case b == nil:
		return a.IsIdentity()
	}
	return a.Equals(b)
}

// traversalTransform is the transform to the parent in traversal order.
func (n *SemanticsNode) traversalTransform() *ui.Matrix4 {
	if n.traversalChildTransform != nil {
		return n.traversalChildTransform
	}
	return n.transform
}

// Rect is the rectangle of the node in its own coordinates.
//
// Flutter: semantics/semantics.dart SemanticsNode.rect
func (n *SemanticsNode) Rect() ui.Rect { return n.rect }

// SetRect sets the rectangle. In debug builds a rectangle that is not finite
// panics with the tag SemanticsRectNotFinite.
//
// Flutter: semantics/semantics.dart SemanticsNode.rect
func (n *SemanticsNode) SetRect(value ui.Rect) {
	if foundation.DebugMode && !value.IsFinite() {
		panic(errors.New("SemanticsRectNotFinite", "a semantics node was given a rectangle that is not finite",
			"node", n.ToStringShort(), "rect", value.String()))
	}
	if n.rect != value {
		n.rect = value
		n.markDirty()
	}
}

// IsInvisible is whether the node is not on screen: its rectangle is empty or
// its transform zero, and it is not merged into its parent.
//
// Flutter: semantics/semantics.dart SemanticsNode.isInvisible
func (n *SemanticsNode) IsInvisible() bool {
	return !n.isMergedIntoParent && (n.rect.IsEmpty() || (n.transform != nil && n.transform.IsZero()))
}

// IsMergedIntoParent is whether the node is part of the node of an ancestor.
//
// Flutter: semantics/semantics.dart SemanticsNode.isMergedIntoParent
func (n *SemanticsNode) IsMergedIntoParent() bool { return n.isMergedIntoParent }

// SetIsMergedIntoParent sets whether the node is merged into its parent.
//
// Flutter: semantics/semantics.dart SemanticsNode.isMergedIntoParent
func (n *SemanticsNode) SetIsMergedIntoParent(value bool) {
	if n.isMergedIntoParent == value {
		return
	}
	n.isMergedIntoParent = value
	if n.parent != nil {
		n.parent.markDirty()
	}
}

// AreUserActionsBlocked is whether the user actions of the node are blocked.
//
// Flutter: semantics/semantics.dart SemanticsNode.areUserActionsBlocked
func (n *SemanticsNode) AreUserActionsBlocked() bool { return n.areUserActionsBlocked }

// SetAreUserActionsBlocked sets whether the user actions are blocked.
//
// Flutter: semantics/semantics.dart SemanticsNode.areUserActionsBlocked
func (n *SemanticsNode) SetAreUserActionsBlocked(value bool) {
	if n.areUserActionsBlocked == value {
		return
	}
	n.areUserActionsBlocked = value
	n.markDirty()
}

// IsPartOfNodeMerging is whether the node merges its descendants or is merged
// into its parent.
//
// Flutter: semantics/semantics.dart SemanticsNode.isPartOfNodeMerging
func (n *SemanticsNode) IsPartOfNodeMerging() bool {
	return n.mergeAllDescendantsIntoThisNode || n.isMergedIntoParent
}

// MergeAllDescendantsIntoThisNode is whether the node merges its descendants.
//
// Flutter: semantics/semantics.dart SemanticsNode.mergeAllDescendantsIntoThisNode
func (n *SemanticsNode) MergeAllDescendantsIntoThisNode() bool {
	return n.mergeAllDescendantsIntoThisNode
}

// replaceChildren sets the children, adopting and dropping nodes. In debug
// builds it panics with the tag SemanticsChildrenMutated if the list is the
// same slice as before with other nodes in it.
func (n *SemanticsNode) replaceChildren(newChildren []*SemanticsNode) {
	if foundation.DebugMode {
		seen := map[*SemanticsNode]bool{}
		for _, c := range newChildren {
			if c == n {
				panic(errors.New("SemanticsNodeOwnChild", "a semantics node can not be its own child"))
			}
			if seen[c] {
				panic(errors.New("SemanticsChildDuplicated", "a semantics node was added as a child twice", "node", c.ToStringShort()))
			}
			seen[c] = true
		}
	}
	for _, c := range n.children {
		c.dead = true
	}
	for _, c := range newChildren {
		c.dead = false
	}
	sawChange := false
	for _, c := range n.children {
		if c.dead {
			if c.parent == n {
				n.dropChild(c)
			}
			sawChange = true
		}
	}
	for _, c := range newChildren {
		if c.parent != n {
			if c.parent != nil {
				c.parent.dropChild(c)
			}
			if foundation.DebugMode && c.Attached() {
				panic(errors.New("SemanticsChildAttached", "a child that is added to a node must not be attached", "node", c.ToStringShort()))
			}
			n.adoptChild(c)
			sawChange = true
		}
	}
	if foundation.DebugMode {
		// Go has no list identity: the new list is the old one if both share
		// the first element in memory.
		if len(newChildren) > 0 && len(n.children) > 0 && &newChildren[0] == &n.children[0] {
			var details []string
			if len(newChildren) != len(n.debugPreviousSnapshot) {
				details = append(details, "The list's length has changed from "+itoa(len(n.debugPreviousSnapshot))+" to "+itoa(len(newChildren))+".")
			} else {
				for i := range newChildren {
					if newChildren[i] != n.debugPreviousSnapshot[i] {
						if len(details) > 0 {
							details = append(details, "")
						}
						details = append(details, "Child node at position "+itoa(i)+" was replaced:",
							"Previous child: "+n.debugPreviousSnapshot[i].String(), "New child: "+newChildren[i].String())
					}
				}
			}
			if len(details) > 0 {
				panic(errors.New("SemanticsChildrenMutated",
					"Failed to replace child semantics nodes because the list of `SemanticsNode`s was mutated.\n"+
						"Instead of mutating the existing list, create a new list containing the desired `SemanticsNode`s.\n"+
						"Error details:\n"+strings.Join(details, "\n")))
			}
		}
		n.debugPreviousSnapshot = slices.Clone(newChildren)
		ancestor := n
		seen := map[*SemanticsNode]bool{n: true}
		for ancestor.parent != nil {
			ancestor = ancestor.parent
			if seen[ancestor] {
				// The adopted children made a loop of parents.
				panic(errors.New("SemanticsCycle", "a semantics node was added as a child of its own descendant"))
			}
			seen[ancestor] = true
		}
		for _, c := range newChildren {
			if c == ancestor {
				panic(errors.New("SemanticsCycle", "a semantics node was added as a child of its own descendant"))
			}
		}
	}
	if !sawChange && n.children != nil {
		for i := range n.children {
			if n.children[i].id != newChildren[i].id {
				sawChange = true
				break
			}
		}
	}
	n.children = newChildren
	if sawChange {
		n.markDirty()
	}
}

// HasChildren is whether the node has children.
//
// Flutter: semantics/semantics.dart SemanticsNode.hasChildren
func (n *SemanticsNode) HasChildren() bool { return len(n.children) > 0 }

// ChildrenCount is the number of children.
//
// Flutter: semantics/semantics.dart SemanticsNode.childrenCount
func (n *SemanticsNode) ChildrenCount() int { return len(n.children) }

// ChildrenCountInTraversalOrder is the number of children in traversal order,
// which differs when traversal identifiers move nodes.
//
// Flutter: semantics/semantics.dart SemanticsNode.childrenCountInTraversalOrder
func (n *SemanticsNode) ChildrenCountInTraversalOrder() int { return len(n.childrenInTraversalOrder()) }

// VisitChildren calls the visitor on the children until it returns false.
//
// Flutter: semantics/semantics.dart SemanticsNode.visitChildren
func (n *SemanticsNode) VisitChildren(visitor SemanticsNodeVisitor) {
	for _, c := range n.children {
		if !visitor(c) {
			return
		}
	}
}

// visitDescendants visits all the descendants depth first until the visitor
// returns false; it returns false if it was stopped.
func (n *SemanticsNode) visitDescendants(visitor SemanticsNodeVisitor) bool {
	for _, c := range n.children {
		if !visitor(c) || !c.visitDescendants(visitor) {
			return false
		}
	}
	return true
}

// Owner is the owner of the node, or nil if it is not attached.
//
// Flutter: semantics/semantics.dart SemanticsNode.owner
func (n *SemanticsNode) Owner() *SemanticsOwner { return n.owner }

// Attached is whether the node is attached to an owner.
//
// Flutter: semantics/semantics.dart SemanticsNode.attached
func (n *SemanticsNode) Attached() bool { return n.owner != nil }

// Parent is the parent node, or nil.
//
// Flutter: semantics/semantics.dart SemanticsNode.parent
func (n *SemanticsNode) Parent() *SemanticsNode { return n.parent }

// TraversalParent is the traversal parent, or the parent.
//
// Flutter: semantics/semantics.dart SemanticsNode.traversalParent
func (n *SemanticsNode) TraversalParent() *SemanticsNode {
	if n.traversalParent != nil {
		return n.traversalParent
	}
	return n.parent
}

func (n *SemanticsNode) setTraversalParent(value *SemanticsNode) {
	if n.traversalParent == value {
		return
	}
	n.traversalParent = value
	n.markDirty()
}

// Depth is the distance from the root.
//
// Flutter: semantics/semantics.dart SemanticsNode.depth
func (n *SemanticsNode) Depth() int { return n.depth }

func (n *SemanticsNode) redepthChild(child *SemanticsNode) {
	if child.depth <= n.depth {
		child.depth = n.depth + 1
		child.redepthChildren()
	}
}

func (n *SemanticsNode) redepthChildren() {
	for _, c := range n.children {
		n.redepthChild(c)
	}
}

func (n *SemanticsNode) updateChildMergeFlagRecursively(child *SemanticsNode) {
	childShouldMerge := n.IsPartOfNodeMerging()
	if childShouldMerge == child.isMergedIntoParent {
		return
	}
	child.SetIsMergedIntoParent(childShouldMerge)
	if !child.mergeAllDescendantsIntoThisNode {
		child.updateChildrenMergeFlags()
	}
}

func (n *SemanticsNode) updateChildrenMergeFlags() {
	for _, c := range n.children {
		n.updateChildMergeFlagRecursively(c)
	}
}

func (n *SemanticsNode) adoptChild(child *SemanticsNode) {
	child.parent = n
	if n.Attached() {
		child.Attach(n.owner)
	}
	n.redepthChild(child)
	n.updateChildMergeFlagRecursively(child)
}

func (n *SemanticsNode) dropChild(child *SemanticsNode) {
	child.parent = nil
	if n.Attached() {
		child.Detach()
	}
}

// Attach makes the node, and its descendants, belong to the owner. A node
// whose ID is taken gets a new one.
//
// Flutter: semantics/semantics.dart SemanticsNode.attach
func (n *SemanticsNode) Attach(owner *SemanticsOwner) {
	if foundation.DebugMode && n.owner != nil {
		panic(errors.New("SemanticsNodeAttached", "a semantics node was attached twice", "node", n.ToStringShort()))
	}
	n.owner = owner
	for {
		if _, taken := owner.nodes[n.id]; !taken {
			break
		}
		n.id = generateNewID()
	}
	owner.nodes[n.id] = n
	delete(owner.detachedNodes, n)
	if n.dirty {
		n.dirty = false
		n.markDirty()
	}
	for _, c := range n.children {
		c.Attach(owner)
	}
}

// Detach removes the node, and its descendants, from the owner.
//
// Flutter: semantics/semantics.dart SemanticsNode.detach
func (n *SemanticsNode) Detach() {
	if foundation.DebugMode && n.owner == nil {
		panic(errors.New("SemanticsNodeDetached", "a semantics node that is not attached was detached", "node", n.ToStringShort()))
	}
	owner := n.owner
	delete(owner.nodes, n.id)
	owner.detachedNodes[n] = struct{}{}
	if id := n.traversalChildID; id != nil {
		if p := owner.traversalParentNodes[id]; p != nil {
			p.markDirty()
		}
	}
	for key, node := range owner.traversalParentNodes {
		if node == n {
			delete(owner.traversalParentNodes, key)
		}
	}
	for key, set := range owner.traversalChildNodes {
		set.remove(n)
		if set.len() == 0 {
			delete(owner.traversalChildNodes, key)
		}
	}
	n.owner = nil
	for _, c := range n.children {
		if c.parent == n {
			c.Detach()
		}
	}
	n.markDirty()
}

func (n *SemanticsNode) markDirty() {
	if n.dirty {
		return
	}
	n.dirty = true
	if n.Attached() {
		n.owner.dirtyNodes[n] = struct{}{}
	}
}

// DebugIsDirty is whether the node has changes that were not sent yet (debug
// builds only; None in release builds).
//
// Flutter: semantics/semantics.dart SemanticsNode.debugIsDirty
func (n *SemanticsNode) DebugIsDirty() foundation.Opt[bool] {
	if !foundation.DebugMode {
		return foundation.None[bool]()
	}
	return foundation.Some(n.dirty)
}

func (n *SemanticsNode) isDifferentFromCurrentSemanticAnnotation(c *SemanticsConfiguration) bool {
	return !n.attributedLabel.Equal(c.attributedLabel) || !n.attributedHint.Equal(c.attributedHint) ||
		!n.attributedValue.Equal(c.attributedValue) || !n.attributedIncreased.Equal(c.attributedIncreased) ||
		!n.attributedDecreased.Equal(c.attributedDecreased) || n.tooltip != c.tooltip || n.flags != c.flags ||
		n.textDirection != c.textDirection || n.sortKey != c.sortKey || !optEqual(n.textSelection, c.textSelection) ||
		!optEqual(n.scrollPosition, c.scrollPosition) || !optEqual(n.scrollExtentMax, c.scrollExtentMax) ||
		!optEqual(n.scrollExtentMin, c.scrollExtentMin) || n.actionsAsBits != c.actionsAsBits ||
		!optEqual(n.IndexInParent, c.indexInParent) || !optEqual(n.platformViewID, c.platformViewID) ||
		!optEqual(n.maxValueLength, c.maxValueLength) || !optEqual(n.currentValueLength, c.currentValueLen) ||
		n.mergeAllDescendantsIntoThisNode != c.isMergingSemanticsOfDescendants ||
		n.areUserActionsBlocked != c.IsBlockingUserActions || n.headingLevel != c.headingLevel ||
		!optEqual(n.linkURL, c.linkURL) || n.role != c.role || n.validationResult != c.validationResult ||
		n.hitTestBehavior != c.hitTestBehavior || n.traversalChildID != c.traversalChild ||
		n.traversalParentID != c.traversalParent || !optEqual(n.minValue, c.minValue) || !optEqual(n.maxValue, c.maxValue) ||
		!sameCustomActions(n.customSemanticsActions, c.customSemanticsActions)
}

// sameCustomActions compares the actions, not their handlers: the handlers are
// not part of what the engine is sent.
func sameCustomActions(a, b map[CustomSemanticsAction]VoidCallback) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func (n *SemanticsNode) effectiveActionsAsBits() ui.SemanticsAction {
	if n.areUserActionsBlocked {
		return n.actionsAsBits & unblockedUserActions
	}
	return n.actionsAsBits
}

// IsTagged is whether the node has the tag.
//
// Flutter: semantics/semantics.dart SemanticsNode.isTagged
func (n *SemanticsNode) IsTagged(tag SemanticsTag) bool {
	_, ok := n.Tags[tag]
	return ok
}

// FlagsCollection are the flags of the node.
//
// Flutter: semantics/semantics.dart SemanticsNode.flagsCollection
func (n *SemanticsNode) FlagsCollection() ui.SemanticsFlags { return n.flags }

// Identifier is the identifier for tests.
//
// Flutter: semantics/semantics.dart SemanticsNode.identifier
func (n *SemanticsNode) Identifier() string { return n.identifier }

// TraversalParentIdentifier is the traversal parent identifier.
//
// Flutter: semantics/semantics.dart SemanticsNode.traversalParentIdentifier
func (n *SemanticsNode) TraversalParentIdentifier() any { return n.traversalParentID }

// TraversalChildIdentifier is the traversal child identifier.
//
// Flutter: semantics/semantics.dart SemanticsNode.traversalChildIdentifier
func (n *SemanticsNode) TraversalChildIdentifier() any { return n.traversalChildID }

func (n *SemanticsNode) isTraversalParent() bool { return n.traversalParentID != nil }
func (n *SemanticsNode) isTraversalChild() bool  { return n.traversalChildID != nil }

// Label is the text of the label.
//
// Flutter: semantics/semantics.dart SemanticsNode.label
func (n *SemanticsNode) Label() string { return n.attributedLabel.String }

// AttributedLabel is the label with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsNode.attributedLabel
func (n *SemanticsNode) AttributedLabel() AttributedString { return n.attributedLabel }

// Value is the text of the value.
//
// Flutter: semantics/semantics.dart SemanticsNode.value
func (n *SemanticsNode) Value() string { return n.attributedValue.String }

// AttributedValue is the value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsNode.attributedValue
func (n *SemanticsNode) AttributedValue() AttributedString { return n.attributedValue }

// IncreasedValue is the text of the value after an increase.
//
// Flutter: semantics/semantics.dart SemanticsNode.increasedValue
func (n *SemanticsNode) IncreasedValue() string { return n.attributedIncreased.String }

// AttributedIncreasedValue is the increased value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsNode.attributedIncreasedValue
func (n *SemanticsNode) AttributedIncreasedValue() AttributedString { return n.attributedIncreased }

// DecreasedValue is the text of the value after a decrease.
//
// Flutter: semantics/semantics.dart SemanticsNode.decreasedValue
func (n *SemanticsNode) DecreasedValue() string { return n.attributedDecreased.String }

// AttributedDecreasedValue is the decreased value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsNode.attributedDecreasedValue
func (n *SemanticsNode) AttributedDecreasedValue() AttributedString { return n.attributedDecreased }

// Hint is the text of the hint.
//
// Flutter: semantics/semantics.dart SemanticsNode.hint
func (n *SemanticsNode) Hint() string { return n.attributedHint.String }

// AttributedHint is the hint with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsNode.attributedHint
func (n *SemanticsNode) AttributedHint() AttributedString { return n.attributedHint }

// Tooltip is the tooltip.
//
// Flutter: semantics/semantics.dart SemanticsNode.tooltip
func (n *SemanticsNode) Tooltip() string { return n.tooltip }

// HintOverrides are the hint overrides.
//
// Flutter: semantics/semantics.dart SemanticsNode.hintOverrides
func (n *SemanticsNode) HintOverrides() *SemanticsHintOverrides { return n.hintOverrides }

// TextDirection is the direction of the strings.
//
// Flutter: semantics/semantics.dart SemanticsNode.textDirection
func (n *SemanticsNode) TextDirection() ui.TextDirection { return n.textDirection }

// SortKey is the sort key.
//
// Flutter: semantics/semantics.dart SemanticsNode.sortKey
func (n *SemanticsNode) SortKey() SemanticsSortKey { return n.sortKey }

// TextSelection is the selection of a text field.
//
// Flutter: semantics/semantics.dart SemanticsNode.textSelection
func (n *SemanticsNode) TextSelection() foundation.Opt[services.TextSelection] {
	return n.textSelection
}

// IsMultiline is whether the text field has several lines.
//
// Flutter: semantics/semantics.dart SemanticsNode.isMultiline
func (n *SemanticsNode) IsMultiline() bool { return n.isMultiline }

// ScrollChildCount is the number of children of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsNode.scrollChildCount
func (n *SemanticsNode) ScrollChildCount() foundation.Opt[int] { return n.scrollChildCount }

// ScrollIndex is the index of the first visible child of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsNode.scrollIndex
func (n *SemanticsNode) ScrollIndex() foundation.Opt[int] { return n.scrollIndex }

// ScrollPosition is the scroll position.
//
// Flutter: semantics/semantics.dart SemanticsNode.scrollPosition
func (n *SemanticsNode) ScrollPosition() foundation.Opt[float64] { return n.scrollPosition }

// ScrollExtentMax is the highest scroll position.
//
// Flutter: semantics/semantics.dart SemanticsNode.scrollExtentMax
func (n *SemanticsNode) ScrollExtentMax() foundation.Opt[float64] { return n.scrollExtentMax }

// ScrollExtentMin is the lowest scroll position.
//
// Flutter: semantics/semantics.dart SemanticsNode.scrollExtentMin
func (n *SemanticsNode) ScrollExtentMin() foundation.Opt[float64] { return n.scrollExtentMin }

// PlatformViewID is the platform view the node stands for.
//
// Flutter: semantics/semantics.dart SemanticsNode.platformViewId
func (n *SemanticsNode) PlatformViewID() foundation.Opt[int] { return n.platformViewID }

// MaxValueLength is the longest value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsNode.maxValueLength
func (n *SemanticsNode) MaxValueLength() foundation.Opt[int] { return n.maxValueLength }

// CurrentValueLength is the length of the value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsNode.currentValueLength
func (n *SemanticsNode) CurrentValueLength() foundation.Opt[int] { return n.currentValueLength }

// HeadingLevel is the heading level, 0 to 6.
//
// Flutter: semantics/semantics.dart SemanticsNode.headingLevel
func (n *SemanticsNode) HeadingLevel() int { return n.headingLevel }

// LinkURL is the URL of a link.
//
// Flutter: semantics/semantics.dart SemanticsNode.linkUrl
func (n *SemanticsNode) LinkURL() foundation.Opt[string] { return n.linkURL }

// Role is the role.
//
// Flutter: semantics/semantics.dart SemanticsNode.role
func (n *SemanticsNode) Role() ui.SemanticsRole { return n.role }

// ControlsNodes are the identifiers of the nodes this one controls.
//
// Flutter: semantics/semantics.dart SemanticsNode.controlsNodes
func (n *SemanticsNode) ControlsNodes() []string { return n.controlsNodes }

// MinValue is the lowest value.
//
// Flutter: semantics/semantics.dart SemanticsNode.minValue
func (n *SemanticsNode) MinValue() foundation.Opt[string] { return n.minValue }

// MaxValue is the highest value.
//
// Flutter: semantics/semantics.dart SemanticsNode.maxValue
func (n *SemanticsNode) MaxValue() foundation.Opt[string] { return n.maxValue }

// ValidationResult is the validation result.
//
// Flutter: semantics/semantics.dart SemanticsNode.validationResult
func (n *SemanticsNode) ValidationResult() ui.SemanticsValidationResult { return n.validationResult }

// HitTestBehavior is the hit test behavior.
//
// Flutter: semantics/semantics.dart SemanticsNode.hitTestBehavior
func (n *SemanticsNode) HitTestBehavior() ui.SemanticsHitTestBehavior { return n.hitTestBehavior }

// InputType is the input type.
//
// Flutter: semantics/semantics.dart SemanticsNode.inputType
func (n *SemanticsNode) InputType() ui.SemanticsInputType { return n.inputType }

func (n *SemanticsNode) canPerformAction(action ui.SemanticsAction) bool {
	_, ok := n.actions[action]
	return ok
}

func (n *SemanticsNode) canPerformCustomAction(id int) bool {
	a, ok := CustomSemanticsActionGetAction(id)
	if !ok {
		return false
	}
	_, ok = n.customSemanticsActions[a]
	return ok
}

func (n *SemanticsNode) canHandleAction(action ui.SemanticsAction, args any) bool {
	if action == ui.SemanticsActionCustomAction {
		id, ok := args.(int)
		return ok && n.canPerformCustomAction(id)
	}
	return n.canPerformAction(action)
}

// UpdateWith gives the node the properties of the configuration (the empty
// configuration if nil) and the children, which are in inverse paint order.
//
// Flutter: semantics/semantics.dart SemanticsNode.updateWith
func (n *SemanticsNode) UpdateWith(config *SemanticsConfiguration, childrenInInversePaintOrder []*SemanticsNode) {
	if config == nil {
		config = emptyConfig
	}
	if n.isDifferentFromCurrentSemanticAnnotation(config) {
		n.markDirty()
	}
	if foundation.DebugMode && config.platformViewID.IsSome() && len(childrenInInversePaintOrder) > 0 {
		panic(errors.New("SemanticsPlatformViewWithChildren", "SemanticsNodes with children must not specify a platformViewId."))
	}
	mergeChanged := n.mergeAllDescendantsIntoThisNode != config.isMergingSemanticsOfDescendants

	n.identifier = config.identifier
	n.traversalParentID = config.traversalParent
	n.traversalChildID = config.traversalChild
	n.attributedLabel = config.attributedLabel
	n.attributedValue = config.attributedValue
	n.attributedIncreased = config.attributedIncreased
	n.attributedDecreased = config.attributedDecreased
	n.attributedHint = config.attributedHint
	n.tooltip = config.tooltip
	n.hintOverrides = config.hintOverrides
	n.flags = config.flags
	n.textDirection = config.textDirection
	n.sortKey = config.sortKey
	n.actions = maps.Clone(config.actions)
	n.customSemanticsActions = maps.Clone(config.customSemanticsActions)
	n.actionsAsBits = config.actionsAsBits
	n.textSelection = config.textSelection
	n.isMultiline = config.IsMultiline()
	n.scrollPosition = config.scrollPosition
	n.scrollExtentMax = config.scrollExtentMax
	n.scrollExtentMin = config.scrollExtentMin
	n.mergeAllDescendantsIntoThisNode = config.isMergingSemanticsOfDescendants
	n.scrollChildCount = config.scrollChildCount
	n.scrollIndex = config.scrollIndex
	n.IndexInParent = config.indexInParent
	n.platformViewID = config.platformViewID
	n.maxValueLength = config.maxValueLength
	n.currentValueLength = config.currentValueLen
	n.areUserActionsBlocked = config.IsBlockingUserActions
	n.headingLevel = config.headingLevel
	n.linkURL = config.linkURL
	n.role = config.role
	n.controlsNodes = config.controlsNodes
	n.validationResult = config.validationResult
	n.hitTestBehavior = config.hitTestBehavior
	n.inputType = config.inputType
	n.locale = config.Locale
	n.minValue = config.minValue
	n.maxValue = config.maxValue
	n.replaceChildren(childrenInInversePaintOrder)

	if mergeChanged {
		n.updateChildrenMergeFlags()
	}
	if foundation.DebugMode {
		if n.canPerformAction(ui.SemanticsActionIncrease) && (n.Value() == "") != (n.IncreasedValue() == "") {
			panic(errors.New("SemanticsIncreaseWithoutValue",
				`A SemanticsNode with action "increase" needs to be annotated with either both "value" and "increasedValue" or neither`))
		}
		if n.canPerformAction(ui.SemanticsActionDecrease) && (n.Value() == "") != (n.DecreasedValue() == "") {
			panic(errors.New("SemanticsDecreaseWithoutValue",
				`A SemanticsNode with action "decrease" needs to be annotated with either both "value" and "decreasedValue" or neither`))
		}
	}
}

func overridingIDs(h *SemanticsHintOverrides, ids map[int]struct{}) {
	if h == nil {
		return
	}
	if h.OnTapHint != "" {
		ids[CustomSemanticsActionGetIdentifier(NewCustomSemanticsActionOverriding(h.OnTapHint, ui.SemanticsActionTap))] = struct{}{}
	}
	if h.OnLongPressHint != "" {
		ids[CustomSemanticsActionGetIdentifier(NewCustomSemanticsActionOverriding(h.OnLongPressHint, ui.SemanticsActionLongPress))] = struct{}{}
	}
}

// GetSemanticsData is the data of the node, with the information of the nodes
// that are merged into it.
//
// Flutter: semantics/semantics.dart SemanticsNode.getSemanticsData
func (n *SemanticsNode) GetSemanticsData() *SemanticsData {
	flags := n.flags
	actions := n.actionsAsBits
	identifier := n.identifier
	traversalParentID, traversalChildID := n.traversalParentID, n.traversalChildID
	label, value, increased, decreased, hint := n.attributedLabel, n.attributedValue, n.attributedIncreased, n.attributedDecreased, n.attributedHint
	tooltip := n.tooltip
	textDirection := n.textDirection
	var mergedTags map[SemanticsTag]struct{}
	if n.Tags != nil {
		mergedTags = maps.Clone(n.Tags)
	}
	textSelection := n.textSelection
	scrollChildCount, scrollIndex := n.scrollChildCount, n.scrollIndex
	scrollPosition, scrollMax, scrollMin := n.scrollPosition, n.scrollExtentMax, n.scrollExtentMin
	platformViewID, maxLen, curLen := n.platformViewID, n.maxValueLength, n.currentValueLength
	headingLevel := n.headingLevel
	linkURL := n.linkURL
	role := n.role
	controlsNodes := n.controlsNodes
	validation := n.validationResult
	hitTest := n.hitTestBehavior
	inputType := n.inputType
	minValue, maxValue := n.minValue, n.maxValue
	customIDs := map[int]struct{}{}
	for a := range n.customSemanticsActions {
		customIDs[CustomSemanticsActionGetIdentifier(a)] = struct{}{}
	}
	overridingIDs(n.hintOverrides, customIDs)

	if n.mergeAllDescendantsIntoThisNode {
		n.visitDescendants(func(node *SemanticsNode) bool {
			flags = flags.Merge(node.flags)
			actions |= node.effectiveActionsAsBits()
			if textDirection == ui.TextDirectionUnspecified {
				textDirection = node.textDirection
			}
			if textSelection.IsNone() {
				textSelection = node.textSelection
			}
			if scrollChildCount.IsNone() {
				scrollChildCount = node.scrollChildCount
			}
			if scrollIndex.IsNone() {
				scrollIndex = node.scrollIndex
			}
			if scrollPosition.IsNone() {
				scrollPosition = node.scrollPosition
			}
			if scrollMax.IsNone() {
				scrollMax = node.scrollExtentMax
			}
			if scrollMin.IsNone() {
				scrollMin = node.scrollExtentMin
			}
			if platformViewID.IsNone() {
				platformViewID = node.platformViewID
			}
			if maxLen.IsNone() {
				maxLen = node.maxValueLength
			}
			if curLen.IsNone() {
				curLen = node.currentValueLength
			}
			if linkURL.IsNone() {
				linkURL = node.linkURL
			}
			headingLevel = mergeHeadingLevels(node.headingLevel, headingLevel)
			if identifier == "" {
				identifier = node.identifier
			}
			if traversalParentID == nil {
				traversalParentID = node.traversalParentID
			}
			if traversalChildID == nil {
				traversalChildID = node.traversalChildID
			}
			if value.String == "" {
				value = node.attributedValue
			}
			if increased.String == "" {
				increased = node.attributedIncreased
			}
			if decreased.String == "" {
				decreased = node.attributedDecreased
			}
			if role == ui.SemanticsRoleNone {
				role = node.role
			}
			if inputType == ui.SemanticsInputTypeNone {
				inputType = node.inputType
			}
			if hitTest == ui.SemanticsHitTestBehaviorDefer {
				hitTest = node.hitTestBehavior
			}
			if tooltip == "" {
				tooltip = node.tooltip
			}
			if node.Tags != nil {
				if mergedTags == nil {
					mergedTags = map[SemanticsTag]struct{}{}
				}
				maps.Copy(mergedTags, node.Tags)
			}
			for a := range node.customSemanticsActions {
				customIDs[CustomSemanticsActionGetIdentifier(a)] = struct{}{}
			}
			overridingIDs(node.hintOverrides, customIDs)
			label = concatAttributedString(label, textDirection, node.attributedLabel, node.textDirection)
			hint = concatAttributedString(hint, textDirection, node.attributedHint, node.textDirection)
			if controlsNodes == nil {
				controlsNodes = node.controlsNodes
			} else if node.controlsNodes != nil {
				controlsNodes = unionStrings(controlsNodes, node.controlsNodes)
			}
			if minValue.IsNone() {
				minValue = node.minValue
			}
			if maxValue.IsNone() {
				maxValue = node.maxValue
			}
			switch validation {
			case ui.SemanticsValidationResultNone:
				validation = node.validationResult
			case ui.SemanticsValidationResultValid:
				if node.validationResult != ui.SemanticsValidationResultNone && node.validationResult != ui.SemanticsValidationResultValid {
					validation = node.validationResult
				}
			}
			return true
		})
	}
	ids := make([]int, 0, len(customIDs))
	for id := range customIDs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if n.areUserActionsBlocked {
		actions &= unblockedUserActions
	}
	return NewSemanticsData(SemanticsData{
		FlagsCollection: flags, Actions: actions, Identifier: identifier,
		TraversalParentIdentifier: traversalParentID, TraversalChildIdentifier: traversalChildID,
		AttributedLabel: label, AttributedValue: value, AttributedIncreasedValue: increased, AttributedDecreasedValue: decreased,
		AttributedHint: hint, Tooltip: tooltip, TextDirection: textDirection, Rect: n.rect, Transform: n.transform,
		Tags: mergedTags, TextSelection: textSelection, ScrollChildCount: scrollChildCount, ScrollIndex: scrollIndex,
		ScrollPosition: scrollPosition, ScrollExtentMax: scrollMax, ScrollExtentMin: scrollMin, PlatformViewID: platformViewID,
		MaxValueLength: maxLen, CurrentValueLength: curLen, CustomSemanticsActionIDs: ids, HeadingLevel: headingLevel,
		LinkURL: linkURL, Role: role, ControlsNodes: controlsNodes, ValidationResult: validation, HitTestBehavior: hitTest,
		InputType: inputType, Locale: n.locale, MinValue: minValue, MaxValue: maxValue,
	})
}

// computeTraversalTransform is the transform from the child to the parent,
// through their common ancestor.
func computeTraversalTransform(parent, child *SemanticsNode) *ui.Matrix4 {
	traversal := ui.NewMatrix4Identity()
	var parentToCommon *ui.Matrix4
	from, to := child, parent
	for from != to {
		fromDepth, toDepth := from.depth, to.depth
		if fromDepth >= toDepth {
			if from.transform != nil {
				traversal.MultiplyBy(from.transform)
			}
			from = from.parent
		}
		if fromDepth <= toDepth {
			if parentToCommon == nil {
				parentToCommon = ui.NewMatrix4Identity()
			}
			if to.transform != nil {
				parentToCommon.MultiplyBy(to.transform)
			}
			to = to.parent
		}
	}
	if parentToCommon != nil {
		if parentToCommon.Invert() != 0 {
			traversal.MultiplyBy(parentToCommon)
		} else {
			traversal.SetZero()
		}
	}
	return traversal
}

func (n *SemanticsNode) childrenIDInTraversalOrder() []int {
	sorted := n.childrenInTraversalOrder()
	ids := make([]int, len(sorted))
	for i, c := range sorted {
		ids[i] = c.id
	}
	return ids
}

func (n *SemanticsNode) childrenInHitTestOrder() []*SemanticsNode {
	if n.children == nil {
		return nil
	}
	if n.isTraversalParent() {
		return n.children
	}
	var out []*SemanticsNode
	for _, c := range n.children {
		if c.isTraversalChild() && n.owner.traversalParentNodes[c.GetSemanticsData().TraversalChildIdentifier] == nil {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (n *SemanticsNode) childrenIDInHitTestOrder() []int {
	children := n.childrenInHitTestOrder()
	ids := make([]int, len(children))
	for i, c := range children {
		ids[len(children)-1-i] = c.id
	}
	return ids
}

func (n *SemanticsNode) addToUpdate(builder *ui.SemanticsUpdateBuilder, customActionIDs map[int]struct{}) {
	data := n.GetSemanticsData()
	if foundation.DebugMode {
		if err := checkSemanticsData(n); err != nil {
			panic(err)
		}
	}
	var inTraversal, inHitTest []int
	if !n.HasChildren() || n.mergeAllDescendantsIntoThisNode {
		if n.isTraversalParent() {
			if n.owner != nil {
				if set, ok := n.owner.traversalChildNodes[n.traversalParentID]; ok {
					for _, node := range set.list {
						if node.Attached() {
							inTraversal = append(inTraversal, node.id)
						}
					}
				}
			}
		}
	} else {
		inTraversal = n.childrenIDInTraversalOrder()
		inHitTest = n.childrenIDInHitTestOrder()
	}
	var customIDs []int
	for _, id := range data.CustomSemanticsActionIDs {
		customIDs = append(customIDs, id)
		customActionIDs[id] = struct{}{}
	}
	traversalParentID := -1
	if id := data.TraversalChildIdentifier; id != nil {
		if p := n.owner.traversalParentNodes[id]; p != nil {
			traversalParentID = p.id
		}
	}
	if id := n.traversalChildID; id != nil {
		n.setTraversalParent(n.owner.traversalParentNodes[id])
		if n.traversalParent != nil {
			n.traversalChildTransform = computeTraversalTransform(n.traversalParent, n)
		}
	}
	optInt := func(o foundation.Opt[int], def int) int { return o.OrElse(def) }
	nan := func(o foundation.Opt[float64]) float64 { return o.OrElse(math.NaN()) }
	identity := ui.NewMatrix4Identity()
	tt := n.traversalTransform()
	if tt == nil {
		tt = identity
	}
	ht := data.Transform
	if ht == nil {
		ht = identity
	}
	update := ui.SemanticsNodeUpdate{
		ID: n.id, Flags: data.FlagsCollection, Actions: data.Actions, Rect: data.Rect, Identifier: data.Identifier,
		Label: data.AttributedLabel.String, LabelAttributes: data.AttributedLabel.Attributes,
		Value: data.AttributedValue.String, ValueAttributes: data.AttributedValue.Attributes,
		IncreasedValue: data.AttributedIncreasedValue.String, IncreasedValueAttributes: data.AttributedIncreasedValue.Attributes,
		DecreasedValue: data.AttributedDecreasedValue.String, DecreasedValueAttributes: data.AttributedDecreasedValue.Attributes,
		Hint: data.AttributedHint.String, HintAttributes: data.AttributedHint.Attributes, Tooltip: data.Tooltip,
		TextDirection: data.TextDirection, TextSelectionBase: -1, TextSelectionExtent: -1,
		PlatformViewID: optInt(data.PlatformViewID, -1), MaxValueLength: optInt(data.MaxValueLength, -1),
		CurrentValueLength: optInt(data.CurrentValueLength, -1), ScrollChildren: optInt(data.ScrollChildCount, 0),
		ScrollIndex: optInt(data.ScrollIndex, 0), ScrollPosition: nan(data.ScrollPosition),
		ScrollExtentMax: nan(data.ScrollExtentMax), ScrollExtentMin: nan(data.ScrollExtentMin),
		Transform: *tt.Storage(), TraversalParent: traversalParentID, HitTestTransform: *ht.Storage(),
		ChildrenInTraversalOrder: inTraversal, ChildrenInHitTestOrder: inHitTest, AdditionalActions: customIDs,
		HeadingLevel: data.HeadingLevel, LinkURL: data.LinkURL.OrElse(""), Role: data.Role,
		ControlsNodes: data.ControlsNodes, ValidationResult: data.ValidationResult, HitTestBehavior: data.HitTestBehavior,
		InputType: data.InputType, MinValue: data.MinValue.OrElse(""), MaxValue: data.MaxValue.OrElse(""),
	}
	if sel, ok := data.TextSelection.Get(); ok {
		update.TextSelectionBase, update.TextSelectionExtent = sel.BaseOffset, sel.ExtentOffset
	}
	if l, ok := data.Locale.Get(); ok {
		update.Locale = &l
	}
	builder.UpdateNode(update)
	n.dirty = false
}

func (n *SemanticsNode) updateChildrenInTraversalOrder() []*SemanticsNode {
	var updated []*SemanticsNode
	for _, child := range n.children {
		if child.isTraversalChild() && !n.isTraversalParent() {
			tp := n.owner.traversalParentNodes[child.GetSemanticsData().TraversalChildIdentifier]
			tpID := -1
			if tp != nil {
				tpID = tp.id
			}
			for ; tp != nil; tp = tp.parent {
				if tp == child {
					panic(errors.New("SemanticsTraversalCycle", "the traversalParent cannot be the child of the traversalChild in hit-test order",
						"traversalParent", tpID, "traversalChild", child.id))
				}
			}
			continue
		}
		updated = append(updated, child)
	}
	if n.isTraversalParent() && n.owner != nil {
		if set := n.owner.traversalChildNodes[n.traversalParentID]; set != nil {
			for cur := n; cur.parent != nil; {
				cur = cur.parent
				if set.has(cur) {
					panic(errors.New("SemanticsTraversalCycle", "the traversalParent cannot be the child of the traversalChild in hit-test order",
						"traversalParent", n.id, "traversalChild", cur.id))
				}
			}
			for _, node := range set.list {
				if node.Attached() {
					updated = append(updated, node)
				}
			}
		}
	}
	return updated
}

func (n *SemanticsNode) childrenInTraversalOrder() []*SemanticsNode {
	var updated []*SemanticsNode
	if n.owner != nil {
		updated = n.updateChildrenInTraversalOrder()
	} else {
		updated = n.children
	}
	inherited := n.textDirection
	for a := n.parent; inherited == ui.TextDirectionUnspecified && a != nil; a = a.parent {
		inherited = a.textDirection
	}
	inDefault := updated
	if inherited != ui.TextDirectionUnspecified {
		inDefault = childrenInDefaultOrder(updated, inherited)
	}
	var everything, sortNodes []traversalSortNode
	var lastKey SemanticsSortKey
	for position, child := range inDefault {
		key := child.sortKey
		lastKey = nil
		if position > 0 {
			lastKey = inDefault[position-1].sortKey
		}
		compatible := position == 0 || sameKeyType(key, lastKey) && (key == nil || optEqual(key.Name(), lastKey.Name()))
		if !compatible && len(sortNodes) > 0 {
			if lastKey != nil {
				sortTraversalNodes(sortNodes)
			}
			everything = append(everything, sortNodes...)
			sortNodes = nil
		}
		sortNodes = append(sortNodes, traversalSortNode{node: child, sortKey: key, position: position})
	}
	if lastKey != nil {
		sortTraversalNodes(sortNodes)
	}
	everything = append(everything, sortNodes...)
	out := make([]*SemanticsNode, len(everything))
	for i, s := range everything {
		out[i] = s.node
	}
	return out
}

// SendEvent sends the event about the node to the platform; it does nothing
// if the node is not attached.
//
// Flutter: semantics/semantics.dart SemanticsNode.sendEvent
func (n *SemanticsNode) SendEvent(event SemanticsEvent) {
	if !n.Attached() || n.owner.onSendEvent == nil {
		return
	}
	n.owner.onSendEvent(event.ToMap(foundation.Some(n.id)))
}

func (n *SemanticsNode) debugIsActionBlocked(action ui.SemanticsAction) bool {
	return foundation.DebugMode && n.effectiveActionsAsBits()&action == 0
}

// ToStringShort is "SemanticsNode#id".
//
// Flutter: semantics/semantics.dart SemanticsNode.toStringShort
func (n *SemanticsNode) ToStringShort() string { return "SemanticsNode#" + itoa(n.id) }

// String describes the node on one line.
func (n *SemanticsNode) String() string { return foundation.DiagnosticableString(n) }

// DebugFillProperties adds the state of the node.
//
// Flutter: semantics/semantics.dart SemanticsNode.debugFillProperties
func (n *SemanticsNode) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	hideOwner := true
	if n.dirty {
		_, inDirty := n.dirtyIn()
		p.Add(foundation.NewFlagProperty("inDirtyNodes", foundation.Some(inDirty), foundation.FlagPropertyOpts{IfTrue: "dirty", IfFalse: "STALE"}))
		hideOwner = inDirty
	}
	var ownerValue any
	if n.owner != nil {
		ownerValue = n.owner
	}
	level := foundation.DiagnosticLevelInfo
	if hideOwner {
		level = foundation.DiagnosticLevelHidden
	}
	p.Add(foundation.NewDiagnosticsProperty("owner", ownerValue, foundation.DiagnosticsPropertyOpts{Level: level}))
	p.Add(foundation.NewFlagProperty("isMergedIntoParent", foundation.Some(n.isMergedIntoParent), foundation.FlagPropertyOpts{IfTrue: "merged up ⬆️"}))
	p.Add(foundation.NewFlagProperty("mergeAllDescendantsIntoThisNode", foundation.Some(n.mergeAllDescendantsIntoThisNode),
		foundation.FlagPropertyOpts{IfTrue: "merge boundary ⛔️"}))
	if l, ok := n.locale.Get(); ok {
		p.Add(foundation.NewStringProperty("locale", foundation.Some(l.String())))
	}
	var offset *ui.Offset
	if n.transform != nil {
		if o, ok := n.transform.AsTranslation(); ok {
			offset = &o
		}
	}
	if offset != nil {
		p.Add(foundation.NewDiagnosticsProperty("rect", n.rect.Shift(*offset), foundation.DiagnosticsPropertyOpts{ShowName: foundation.Some(false)}))
	} else {
		description := ""
		if n.transform != nil {
			if scale, ok := n.transform.AsScale(); ok {
				description = n.rect.String() + " scaled by " + strconv.FormatFloat(scale, 'f', 1, 64) + "x"
			} else if !n.transform.IsIdentity() {
				description = n.rect.String() + " with transform [" + describeTransformRows(n.transform) + "]"
			}
		}
		p.Add(foundation.NewDiagnosticsProperty("rect", n.rect, foundation.DiagnosticsPropertyOpts{Description: description, ShowName: foundation.Some(false)}))
	}
	var tagNames []string
	if n.Tags != nil {
		tagNames = setTags(n.Tags)
	}
	p.Add(foundation.NewIterableProperty("tags", tagNames, foundation.IterablePropertyOpts{DefaultValue: foundation.DefaultNull}))
	actions := []string{}
	for a := range n.actions {
		name := a.Name()
		if n.debugIsActionBlocked(a) {
			name += "🚫️"
		}
		actions = append(actions, name)
	}
	slices.Sort(actions)
	custom := []string{}
	for a := range n.customSemanticsActions {
		custom = append(custom, a.Label())
	}
	slices.Sort(custom)
	p.Add(foundation.NewIterableProperty("actions", actions, foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewIterableProperty("customActions", custom, foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewIterableProperty("flags", append([]string{}, n.flags.ToStrings()...), foundation.IterablePropertyOpts{NoIfEmpty: true}))
	p.Add(foundation.NewFlagProperty("isInvisible", foundation.Some(n.IsInvisible()), foundation.FlagPropertyOpts{IfTrue: "invisible"}))
	p.Add(foundation.NewFlagProperty("isHidden", foundation.Some(n.flags.IsHidden), foundation.FlagPropertyOpts{IfTrue: "HIDDEN"}))
	null := foundation.DefaultNull
	p.Add(foundation.NewStringProperty("identifier", foundation.Some(n.identifier), foundation.StringPropertyOpts{DefaultValue: ""}))
	p.Add(foundation.NewDiagnosticsProperty("traversalParentIdentifier", n.traversalParentID, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewDiagnosticsProperty("traversalChildIdentifier", n.traversalChildID, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	p.Add(NewAttributedStringProperty("label", &n.attributedLabel, false))
	p.Add(NewAttributedStringProperty("value", &n.attributedValue, false))
	p.Add(NewAttributedStringProperty("increasedValue", &n.attributedIncreased, false))
	p.Add(NewAttributedStringProperty("decreasedValue", &n.attributedDecreased, false))
	p.Add(NewAttributedStringProperty("hint", &n.attributedHint, false))
	p.Add(foundation.NewStringProperty("tooltip", foundation.Some(n.tooltip), foundation.StringPropertyOpts{DefaultValue: ""}))
	p.Add(foundation.NewEnumProperty("textDirection", optDirection(n.textDirection), foundation.EnumPropertyOpts{DefaultValue: null}))
	if n.role != ui.SemanticsRoleNone {
		p.Add(foundation.NewEnumProperty("role", foundation.Some(n.role)))
	}
	p.Add(foundation.NewDiagnosticsProperty("sortKey", n.sortKey, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	if sel, ok := n.textSelection.Get(); ok && sel.IsValid() {
		p.Add(foundation.NewMessageProperty("text selection", "["+itoa(sel.Start())+", "+itoa(sel.End())+"]"))
	}
	nullNum := foundation.NumberPropertyOpts{DefaultValue: null}
	p.Add(foundation.NewIntProperty("platformViewId", n.platformViewID, nullNum))
	p.Add(foundation.NewIntProperty("maxValueLength", n.maxValueLength, nullNum))
	p.Add(foundation.NewIntProperty("currentValueLength", n.currentValueLength, nullNum))
	p.Add(foundation.NewIntProperty("scrollChildren", n.scrollChildCount, nullNum))
	p.Add(foundation.NewIntProperty("scrollIndex", n.scrollIndex, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollExtentMin", n.scrollExtentMin, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollPosition", n.scrollPosition, nullNum))
	p.Add(foundation.NewDoubleProperty("scrollExtentMax", n.scrollExtentMax, nullNum))
	p.Add(foundation.NewIntProperty("indexInParent", n.IndexInParent, nullNum))
	p.Add(foundation.NewIntProperty("headingLevel", foundation.Some(n.headingLevel), foundation.NumberPropertyOpts{DefaultValue: 0}))
	if n.inputType != ui.SemanticsInputTypeNone {
		p.Add(foundation.NewEnumProperty("inputType", foundation.Some(n.inputType)))
	}
	if n.validationResult != ui.SemanticsValidationResultNone {
		p.Add(foundation.NewEnumProperty("validationResult", foundation.Some(n.validationResult),
			foundation.EnumPropertyOpts{DefaultValue: ui.SemanticsValidationResultNone}))
	}
	p.Add(foundation.NewStringProperty("minValue", n.minValue, foundation.StringPropertyOpts{DefaultValue: null}))
	p.Add(foundation.NewStringProperty("maxValue", n.maxValue, foundation.StringPropertyOpts{DefaultValue: null}))
}

func (n *SemanticsNode) dirtyIn() (struct{}, bool) {
	if n.owner == nil {
		return struct{}{}, false
	}
	_, ok := n.owner.dirtyNodes[n]
	return struct{}{}, ok
}

func describeTransformRows(m *ui.Matrix4) string {
	rows := painting.DebugDescribeTransform(m)
	parts := make([]string, 0, 4)
	for _, r := range rows {
		if len(r) > 4 {
			parts = append(parts, r[4:])
		}
	}
	return strings.Join(parts, "; ")
}

// DebugSemanticsDumpOrder is the order in which a dump lists the children.
//
// Flutter: semantics/semantics.dart DebugSemanticsDumpOrder
type DebugSemanticsDumpOrder int

// The values of DebugSemanticsDumpOrder. The zero value is
// DebugSemanticsDumpOrderTraversal, Flutter's default.
const (
	// DebugSemanticsDumpOrderTraversal lists the children in traversal order.
	//
	// Flutter: semantics/semantics.dart DebugSemanticsDumpOrder.traversalOrder
	DebugSemanticsDumpOrderTraversal DebugSemanticsDumpOrder = iota
	// DebugSemanticsDumpOrderInverseHitTest lists the children in inverse hit
	// test order.
	//
	// Flutter: semantics/semantics.dart DebugSemanticsDumpOrder.inverseHitTest
	DebugSemanticsDumpOrderInverseHitTest
)

// DebugDescribeChildren describes the children in traversal order.
//
// Flutter: semantics/semantics.dart SemanticsNode.debugDescribeChildren
func (n *SemanticsNode) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return n.DebugDescribeChildrenInOrder(DebugSemanticsDumpOrderTraversal)
}

// DebugDescribeChildrenInOrder describes the children in the order.
//
// Flutter: semantics/semantics.dart SemanticsNode.debugDescribeChildren
func (n *SemanticsNode) DebugDescribeChildrenInOrder(order DebugSemanticsDumpOrder) []foundation.DiagnosticsNode {
	children := n.DebugListChildrenInOrder(order)
	out := make([]foundation.DiagnosticsNode, len(children))
	for i, c := range children {
		out[i] = c.ToDiagnosticsNodeWithOrder(foundation.DiagnosticsNodeOpts{}, order)
	}
	return out
}

// DebugListChildrenInOrder lists the children in the order.
//
// Flutter: semantics/semantics.dart SemanticsNode.debugListChildrenInOrder
func (n *SemanticsNode) DebugListChildrenInOrder(order DebugSemanticsDumpOrder) []*SemanticsNode {
	if n.children == nil {
		return nil
	}
	if order == DebugSemanticsDumpOrderInverseHitTest {
		return n.childrenInHitTestOrder()
	}
	return n.childrenInTraversalOrder()
}

// ToDiagnosticsNode is the diagnostics node of the node, in traversal order.
//
// Flutter: semantics/semantics.dart SemanticsNode.toDiagnosticsNode
func (n *SemanticsNode) ToDiagnosticsNode(opts foundation.DiagnosticsNodeOpts) foundation.DiagnosticsNode {
	return n.ToDiagnosticsNodeWithOrder(opts, DebugSemanticsDumpOrderTraversal)
}

// ToDiagnosticsNodeWithOrder is the diagnostics node, listing the children in
// the order. The style defaults to sparse.
//
// Flutter: semantics/semantics.dart SemanticsNode.toDiagnosticsNode
func (n *SemanticsNode) ToDiagnosticsNodeWithOrder(opts foundation.DiagnosticsNodeOpts, order DebugSemanticsDumpOrder) foundation.DiagnosticsNode {
	if opts.Style == foundation.DiagnosticsTreeStyleUnspecified {
		opts.Style = foundation.DiagnosticsTreeStyleSparse
	}
	d := &semanticsDiagnosticsNode{value: n, order: order}
	foundation.InitDiagnosticsNodeBase(&d.DiagnosticsNodeBase, d, foundation.DiagnosticsNodeBaseOpts{Name: opts.Name, Style: opts.Style})
	return d
}

// SemanticsToStringDeepOpts are the optional parameters of
// SemanticsNode.ToStringDeep.
//
// Flutter: semantics/semantics.dart SemanticsNode.toStringDeep
type SemanticsToStringDeepOpts struct {
	foundation.ToStringDeepOpts
	// ChildOrder is the order of the children; the default is traversal order.
	ChildOrder DebugSemanticsDumpOrder
}

// ToStringDeep is a dump of the node and its descendants.
//
// Flutter: semantics/semantics.dart SemanticsNode.toStringDeep
func (n *SemanticsNode) ToStringDeep(opts ...SemanticsToStringDeepOpts) string {
	var o SemanticsToStringDeepOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return n.ToDiagnosticsNodeWithOrder(foundation.DiagnosticsNodeOpts{}, o.ChildOrder).ToStringDeep(o.ToStringDeepOpts)
}

// semanticsDiagnosticsNode describes a node, listing its children in a chosen
// order.
type semanticsDiagnosticsNode struct {
	foundation.DiagnosticsNodeBase
	value  *SemanticsNode
	order  DebugSemanticsDumpOrder
	cached *foundation.DiagnosticPropertiesBuilder
}

func (d *semanticsDiagnosticsNode) builder() *foundation.DiagnosticPropertiesBuilder {
	if d.cached == nil {
		d.cached = foundation.NewDiagnosticPropertiesBuilder()
		d.value.DebugFillProperties(d.cached)
	}
	return d.cached
}

func (d *semanticsDiagnosticsNode) Value() any { return d.value }

func (d *semanticsDiagnosticsNode) Style() foundation.DiagnosticsTreeStyle {
	if foundation.ReleaseMode {
		return foundation.DiagnosticsTreeStyleNone
	}
	if s := d.DiagnosticsNodeBase.Style(); s != foundation.DiagnosticsTreeStyleUnspecified {
		return s
	}
	return d.builder().DefaultDiagnosticsTreeStyle
}

func (d *semanticsDiagnosticsNode) EmptyBodyDescription() foundation.Opt[string] {
	if foundation.ReleaseMode || foundation.ProfileMode {
		return foundation.Some("")
	}
	return d.builder().EmptyBodyDescription
}

func (d *semanticsDiagnosticsNode) GetProperties() []foundation.DiagnosticsNode {
	if foundation.ReleaseMode || foundation.ProfileMode {
		return nil
	}
	return d.builder().Properties
}

func (d *semanticsDiagnosticsNode) GetChildren() []foundation.DiagnosticsNode {
	return d.value.DebugDescribeChildrenInOrder(d.order)
}

func (d *semanticsDiagnosticsNode) ToDescription(*foundation.TextTreeConfiguration) string {
	if !foundation.DebugMode {
		return ""
	}
	return d.value.ToStringShort()
}
