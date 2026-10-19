package widgets

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// TraversalDirection is a direction in which the focus can move.
//
// Flutter: widgets/focus_traversal.dart TraversalDirection
type TraversalDirection int

// The values of TraversalDirection follow Flutter's declaration order.
const (
	// TraversalDirectionUnspecified is the zero value: no direction.
	//
	// Flutter: no counterpart
	TraversalDirectionUnspecified TraversalDirection = iota
	// TraversalDirectionUp moves the focus up.
	//
	// Flutter: widgets/focus_traversal.dart TraversalDirection.up
	TraversalDirectionUp
	// TraversalDirectionRight moves the focus to the right.
	//
	// Flutter: widgets/focus_traversal.dart TraversalDirection.right
	TraversalDirectionRight
	// TraversalDirectionDown moves the focus down.
	//
	// Flutter: widgets/focus_traversal.dart TraversalDirection.down
	TraversalDirectionDown
	// TraversalDirectionLeft moves the focus to the left.
	//
	// Flutter: widgets/focus_traversal.dart TraversalDirection.left
	TraversalDirectionLeft
)

// String is the name of the direction, like Dart's enum toString.
//
// Flutter: widgets/focus_traversal.dart TraversalDirection
func (d TraversalDirection) String() string {
	switch d {
	case TraversalDirectionUp:
		return "TraversalDirection.up"
	case TraversalDirectionRight:
		return "TraversalDirection.right"
	case TraversalDirectionDown:
		return "TraversalDirection.down"
	case TraversalDirectionLeft:
		return "TraversalDirection.left"
	}
	return "TraversalDirection.unspecified"
}

// TraversalEdgeBehavior is what the focus does when traversal reaches the edge of
// a scope.
//
// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior
type TraversalEdgeBehavior int

// The values of TraversalEdgeBehavior follow Flutter's declaration order.
const (
	// TraversalEdgeBehaviorUnspecified is the zero value: the default of the
	// scope (closed loop for next and previous focus, stop for directional focus).
	//
	// Flutter: no counterpart
	TraversalEdgeBehaviorUnspecified TraversalEdgeBehavior = iota
	// TraversalEdgeBehaviorClosedLoop wraps the focus around inside the scope.
	//
	// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior.closedLoop
	TraversalEdgeBehaviorClosedLoop
	// TraversalEdgeBehaviorLeaveFlutterView lets the focus leave the view: the
	// focus is removed from the scope and traversal ends.
	//
	// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior.leaveFlutterView
	TraversalEdgeBehaviorLeaveFlutterView
	// TraversalEdgeBehaviorParentScope moves the focus on in the parent scope.
	//
	// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior.parentScope
	TraversalEdgeBehaviorParentScope
	// TraversalEdgeBehaviorStop keeps the focus where it is.
	//
	// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior.stop
	TraversalEdgeBehaviorStop
)

// String is the name of the behavior, like Dart's enum toString.
//
// Flutter: widgets/focus_traversal.dart TraversalEdgeBehavior
func (b TraversalEdgeBehavior) String() string {
	switch b {
	case TraversalEdgeBehaviorClosedLoop:
		return "TraversalEdgeBehavior.closedLoop"
	case TraversalEdgeBehaviorLeaveFlutterView:
		return "TraversalEdgeBehavior.leaveFlutterView"
	case TraversalEdgeBehaviorParentScope:
		return "TraversalEdgeBehavior.parentScope"
	case TraversalEdgeBehaviorStop:
		return "TraversalEdgeBehavior.stop"
	}
	return "TraversalEdgeBehavior.unspecified"
}

// TraversalRequestFocusOpts are the optional parameters of a
// TraversalRequestFocusCallback.
//
// Flutter: widgets/focus_traversal.dart TraversalRequestFocusCallback
type TraversalRequestFocusOpts struct {
	// AlignmentPolicy is how the node is aligned in the scrollables around it; the
	// default is ScrollPositionAlignmentPolicyExplicit.
	AlignmentPolicy ScrollPositionAlignmentPolicy
	// Alignment is where in the viewport the node ends up; the default is 1.
	Alignment foundation.Opt[float64]
	// Duration is the length of the scroll animation; the default is zero.
	Duration time.Duration
	// Curve is the curve of the scroll animation; the default is Curves.ease.
	Curve animation.Curve
}

// TraversalRequestFocusCallback is the function a traversal policy calls to give
// the focus to a node.
//
// Flutter: widgets/focus_traversal.dart TraversalRequestFocusCallback
type TraversalRequestFocusCallback func(node *FocusNode, opts TraversalRequestFocusOpts)

// DefaultTraversalRequestFocusCallback requests the focus for the node and
// scrolls it into view.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.defaultTraversalRequestFocusCallback
func DefaultTraversalRequestFocusCallback(node *FocusNode, opts TraversalRequestFocusOpts) {
	node.RequestFocus()
	ScrollableEnsureVisible(node.Context(), EnsureVisibleOpts{
		Alignment:       opts.Alignment.OrElse(1),
		AlignmentPolicy: opts.AlignmentPolicy,
		Duration:        opts.Duration,
		Curve:           opts.Curve,
	})
}

// FocusTraversalPolicy defines how the focus moves between the nodes of a
// FocusTraversalGroup: in what order Next and Previous visit them, and which node
// is next in a direction. Implement it by embedding FocusTraversalPolicyBase
// (InitFocusTraversalPolicyBase) and writing SortDescendants.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy
type FocusTraversalPolicy interface {
	foundation.Diagnosticable
	// RequestFocusCallback is the function that gives the focus to a node.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.requestFocusCallback
	RequestFocusCallback() TraversalRequestFocusCallback
	// FindFirstFocus is the node that gets the focus first when focus moves into
	// the scope of the node: the focused child of the scope, else the first of the
	// sorted descendants, else the node itself.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.findFirstFocus
	FindFirstFocus(currentNode *FocusNode, ignoreCurrentFocus ...bool) *FocusNode
	// FindLastFocus is like FindFirstFocus but picks the last node.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.findLastFocus
	FindLastFocus(currentNode *FocusNode, ignoreCurrentFocus ...bool) *FocusNode
	// FindFirstFocusInDirection is the node that gets the focus first when the
	// focus moves into the scope in the direction.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.findFirstFocusInDirection
	FindFirstFocusInDirection(currentNode *FocusNode, direction TraversalDirection) *FocusNode
	// InvalidateScopeData drops what the policy remembers about the scope.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.invalidateScopeData
	InvalidateScopeData(node *FocusScopeNode)
	// ChangedScope is called when a node moved from a scope to another.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.changedScope
	ChangedScope(node *FocusNode, oldScope *FocusScopeNode)
	// Next moves the focus to the next node and returns whether it moved.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.next
	Next(currentNode *FocusNode) bool
	// Previous moves the focus to the previous node and returns whether it moved.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.previous
	Previous(currentNode *FocusNode) bool
	// InDirection moves the focus to the closest node in the direction and
	// returns whether it moved.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.inDirection
	InDirection(currentNode *FocusNode, direction TraversalDirection) bool
	// SortDescendants returns the nodes in traversal order; currentNode is the
	// node being traversed from.
	//
	// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.sortDescendants
	SortDescendants(descendants []*FocusNode, currentNode *FocusNode) []*FocusNode
}

// FocusTraversalPolicyBaseOpts are the optional parameters of
// InitFocusTraversalPolicyBase.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.new
type FocusTraversalPolicyBaseOpts struct {
	// RequestFocusCallback replaces DefaultTraversalRequestFocusCallback.
	RequestFocusCallback TraversalRequestFocusCallback
	// Directional gives the policy the directional traversal of
	// DirectionalFocusTraversalPolicyMixin: FindFirstFocusInDirection, InDirection,
	// InvalidateScopeData and ChangedScope. Without it InDirection moves nothing.
	Directional bool
}

// FocusTraversalPolicyBase implements FocusTraversalPolicy but SortDescendants:
// embed it, call InitFocusTraversalPolicyBase with the outer value, and write
// SortDescendants.
//
// Deviation: Flutter's DirectionalFocusTraversalPolicyMixin is the Directional
// option of the base (SPEC §12.10).
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy
type FocusTraversalPolicyBase struct {
	self                 FocusTraversalPolicy
	requestFocusCallback TraversalRequestFocusCallback
	directional          bool
	policyData           map[*FocusScopeNode]*directionalPolicyData
}

// InitFocusTraversalPolicyBase sets the outer value of the base.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.new
func InitFocusTraversalPolicyBase(b *FocusTraversalPolicyBase, self FocusTraversalPolicy, opts ...FocusTraversalPolicyBaseOpts) {
	o := firstOpts(opts)
	b.self = self
	b.requestFocusCallback = o.RequestFocusCallback
	if b.requestFocusCallback == nil {
		b.requestFocusCallback = DefaultTraversalRequestFocusCallback
	}
	b.directional = o.Directional
	b.policyData = map[*FocusScopeNode]*directionalPolicyData{}
}

// RequestFocusCallback is the function that gives the focus to a node.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.requestFocusCallback
func (b *FocusTraversalPolicyBase) RequestFocusCallback() TraversalRequestFocusCallback {
	return b.requestFocusCallback
}

// DebugFillProperties adds nothing.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy
func (b *FocusTraversalPolicyBase) DebugFillProperties(*foundation.DiagnosticPropertiesBuilder) {}

// String describes the policy by its type and identity, as Dart's toString does.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy
func (b *FocusTraversalPolicyBase) String() string { return foundation.DescribeIdentity(b.self) }

// requestTabTraversalFocus requests the focus for the node, going down into the
// scopes first.
func (b *FocusTraversalPolicyBase) requestTabTraversalFocus(node *FocusNode, opts TraversalRequestFocusOpts, forward bool) bool {
	if scope := node.AsScope(); scope != nil {
		if scope.FocusedChild() != nil {
			// The focused child may be a scope without a first focus: the first
			// focus is picked in the next iteration.
			return b.requestTabTraversalFocus(scope.FocusedChild(), opts, forward)
		}
		sortedChildren := sortAllDescendants(scope, node)
		if len(sortedChildren) > 0 {
			if forward {
				b.requestTabTraversalFocus(sortedChildren[0], opts, forward)
			} else {
				b.requestTabTraversalFocus(sortedChildren[len(sortedChildren)-1], opts, forward)
			}
			// Whatever it returned, a first focus has been picked.
			return true
		}
	}
	nodeHadPrimaryFocus := node.HasPrimaryFocus()
	b.self.RequestFocusCallback()(node, opts)
	return !nodeHadPrimaryFocus
}

// FindFirstFocus is the node that gets the focus first in the scope of the node.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.findFirstFocus
func (b *FocusTraversalPolicyBase) FindFirstFocus(currentNode *FocusNode, ignoreCurrentFocus ...bool) *FocusNode {
	return b.findInitialFocus(currentNode, false, len(ignoreCurrentFocus) > 0 && ignoreCurrentFocus[0])
}

// FindLastFocus is the node that gets the focus last in the scope of the node.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.findLastFocus
func (b *FocusTraversalPolicyBase) FindLastFocus(currentNode *FocusNode, ignoreCurrentFocus ...bool) *FocusNode {
	return b.findInitialFocus(currentNode, true, len(ignoreCurrentFocus) > 0 && ignoreCurrentFocus[0])
}

func (b *FocusTraversalPolicyBase) findInitialFocus(currentNode *FocusNode, fromEnd, ignoreCurrentFocus bool) *FocusNode {
	scope := currentNode.NearestScope()
	candidate := scope.FocusedChild()
	if ignoreCurrentFocus || (candidate == nil && len(scope.Descendants()) > 0) {
		var sorted []*FocusNode
		for _, node := range sortAllDescendants(scope, currentNode) {
			if canRequestTraversalFocus(node) {
				sorted = append(sorted, node)
			}
		}
		switch {
		case len(sorted) == 0:
			candidate = nil
		case fromEnd:
			candidate = sorted[len(sorted)-1]
		default:
			candidate = sorted[0]
		}
	}
	// If there is still no candidate, the current node is the fallback.
	if candidate == nil {
		candidate = currentNode
	}
	return candidate
}

// Next moves the focus to the next node in traversal order.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.next
func (b *FocusTraversalPolicyBase) Next(currentNode *FocusNode) bool {
	return b.moveFocus(currentNode, true)
}

// Previous moves the focus to the previous node in traversal order.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.previous
func (b *FocusTraversalPolicyBase) Previous(currentNode *FocusNode) bool {
	return b.moveFocus(currentNode, false)
}

func canRequestTraversalFocus(node *FocusNode) bool {
	return node.CanRequestFocus() && !node.SkipTraversal()
}

func descendantsWithoutExpandingScope(node *FocusNode) []*FocusNode {
	var result []*FocusNode
	for _, child := range node.children {
		result = append(result, child)
		if child.AsScope() == nil {
			result = append(result, descendantsWithoutExpandingScope(child)...)
		}
	}
	return result
}

// focusTraversalGroupInfo temporarily holds the information about a
// FocusTraversalGroup while its contents are sorted.
//
// Flutter: widgets/focus_traversal.dart _FocusTraversalGroupInfo
type focusTraversalGroupInfo struct {
	groupNode *FocusNode
	policy    FocusTraversalPolicy
	members   []*FocusNode
}

func newFocusTraversalGroupInfo(group *FocusNode, defaultPolicy FocusTraversalPolicy) *focusTraversalGroupInfo {
	info := &focusTraversalGroupInfo{groupNode: group}
	switch {
	case group != nil && group.group != nil && group.group.policy != nil:
		info.policy = group.group.policy
	case defaultPolicy != nil:
		info.policy = defaultPolicy
	default:
		info.policy = NewReadingOrderTraversalPolicy()
	}
	return info
}

func findGroups(scope *FocusScopeNode, scopeGroupNode *FocusNode, currentNode *FocusNode) map[*FocusNode]*focusTraversalGroupInfo {
	var defaultPolicy FocusTraversalPolicy
	if scopeGroupNode != nil {
		defaultPolicy = scopeGroupNode.group.policy
	} else {
		defaultPolicy = NewReadingOrderTraversalPolicy()
	}
	groups := map[*FocusNode]*focusTraversalGroupInfo{}
	info := func(group *FocusNode) *focusTraversalGroupInfo {
		i, ok := groups[group]
		if !ok {
			i = newFocusTraversalGroupInfo(group, defaultPolicy)
			groups[group] = i
		}
		return i
	}
	for _, node := range descendantsWithoutExpandingScope(&scope.FocusNode) {
		groupNode := focusTraversalGroupNodeOf(node)
		// Group nodes are added to their parent's group, or to the nil group if
		// no parent is found. This creates the hierarchy of group nodes and sorts
		// the entire group along with the other members of the parent group.
		if node == groupNode {
			// To find the parent of the group node, the parent of the Focus node
			// that FocusTraversalGroup builds is skipped: the lookup returns the
			// node it was called on if it matches the type.
			parentGroup := focusTraversalGroupNodeOf(groupNode.parent)
			i := info(parentGroup)
			i.members = append(i.members, groupNode)
			continue
		}
		// Non-focusable and non-traversable nodes are skipped as
		// FocusScopeNode.TraversalDescendants does. The current focused node is
		// kept so that the caller can find the next traversable node from it.
		if node == currentNode || (node.CanRequestFocus() && !node.SkipTraversal()) {
			i := info(groupNode)
			i.members = append(i.members, node)
		}
	}
	return groups
}

// sortAllDescendants sorts all the descendants of the scope, taking into account
// the FocusTraversalGroup each is in, and leaves out the nodes that cannot be
// traversed.
func sortAllDescendants(scope *FocusScopeNode, currentNode *FocusNode) []*FocusNode {
	scopeGroupNode := focusTraversalGroupNodeOf(&scope.FocusNode)
	// Builds the sorting data structure, separating descendants into groups.
	groups := findGroups(scope, scopeGroupNode, currentNode)
	// Sorts the member lists with the policy of each group.
	for _, info := range groups {
		sorted := slices.Clone(info.policy.SortDescendants(info.members, currentNode))
		info.members = sorted
	}
	// Traverses the group tree, adding the children of members in the order they
	// appear in the member lists.
	var sortedDescendants []*FocusNode
	var visitGroups func(info *focusTraversalGroupInfo)
	visitGroups = func(info *focusTraversalGroupInfo) {
		for _, node := range info.members {
			if inner, ok := groups[node]; ok {
				// This is a policy group focus node: it is replaced by the members of
				// the corresponding policy group.
				visitGroups(inner)
			} else {
				sortedDescendants = append(sortedDescendants, node)
			}
		}
	}
	// Visits the children of the scope, if any.
	if len(groups) > 0 {
		if info, ok := groups[scopeGroupNode]; ok {
			visitGroups(info)
		}
	}
	// Removes the FocusTraversalGroup nodes themselves, which are not focusable:
	// they were needed above to find their members during sorting.
	return slices.DeleteFunc(sortedDescendants, func(node *FocusNode) bool {
		return node != currentNode && !canRequestTraversalFocus(node)
	})
}

func (b *FocusTraversalPolicyBase) moveFocus(currentNode *FocusNode, forward bool) bool {
	nearestScope := currentNode.NearestScope()
	b.self.InvalidateScopeData(nearestScope)
	focusedChild := nearestScope.FocusedChild()
	alignment := ScrollPositionAlignmentPolicyKeepVisibleAtStart
	if forward {
		alignment = ScrollPositionAlignmentPolicyKeepVisibleAtEnd
	}
	if focusedChild == nil {
		var firstFocus *FocusNode
		if forward {
			firstFocus = b.self.FindFirstFocus(currentNode)
		} else {
			firstFocus = b.self.FindLastFocus(currentNode)
		}
		if firstFocus != nil {
			return b.requestTabTraversalFocus(firstFocus, TraversalRequestFocusOpts{AlignmentPolicy: alignment}, forward)
		}
	}
	if focusedChild == nil {
		focusedChild = &nearestScope.FocusNode
	}
	sortedNodes := sortAllDescendants(nearestScope, focusedChild)
	if len(sortedNodes) == 0 || !slices.Contains(sortedNodes, focusedChild) {
		panic(errors.New("FocusTraversalNodeNotSorted",
			"the focused node is not among the sorted nodes of its scope; a SortDescendants that removes the current node breaks traversal",
			"node", focusedChild.String()))
	}
	rootScope := (*FocusScopeNode)(nil)
	if m := currentNode.manager; m != nil {
		rootScope = m.rootScope
	}
	atEnd := (forward && focusedChild == sortedNodes[len(sortedNodes)-1]) || (!forward && focusedChild == sortedNodes[0])
	if atEnd {
		edge := sortedNodes[0]
		edgeAlignment := ScrollPositionAlignmentPolicyKeepVisibleAtEnd
		if !forward {
			edge = sortedNodes[len(sortedNodes)-1]
			edgeAlignment = ScrollPositionAlignmentPolicyKeepVisibleAtStart
		}
		switch nearestScope.traversalEdgeBehavior {
		case TraversalEdgeBehaviorLeaveFlutterView:
			focusedChild.Unfocus()
			return false
		case TraversalEdgeBehaviorParentScope:
			parentScope := nearestScope.EnclosingScope()
			if parentScope != nil && parentScope != rootScope {
				focusedChild.Unfocus()
				if forward {
					parentScope.NextFocus()
				} else {
					parentScope.PreviousFocus()
				}
				// Verifies that the focus really has changed.
				return focusedChild.EnclosingScope().FocusedChild() != focusedChild
			}
			// No valid parent scope: falls back to the closed loop behavior.
			return b.requestTabTraversalFocus(edge, TraversalRequestFocusOpts{AlignmentPolicy: edgeAlignment}, forward)
		case TraversalEdgeBehaviorStop:
			return false
		default: // closed loop
			return b.requestTabTraversalFocus(edge, TraversalRequestFocusOpts{AlignmentPolicy: edgeAlignment}, forward)
		}
	}
	ordered := sortedNodes
	if !forward {
		ordered = slices.Clone(sortedNodes)
		slices.Reverse(ordered)
	}
	var previousNode *FocusNode
	for _, node := range ordered {
		if previousNode == focusedChild {
			return b.requestTabTraversalFocus(node, TraversalRequestFocusOpts{AlignmentPolicy: alignment}, forward)
		}
		previousNode = node
	}
	return false
}

// directionalPolicyEntry is a step of the traversal history of a scope.
//
// Flutter: widgets/focus_traversal.dart _DirectionalPolicyDataEntry
type directionalPolicyEntry struct {
	direction TraversalDirection
	node      *FocusNode
}

// directionalPolicyData is the traversal history of a scope, kept to avoid
// hysteresis when the direction changes.
//
// Flutter: widgets/focus_traversal.dart _DirectionalPolicyData
type directionalPolicyData struct {
	history []directionalPolicyEntry
}

// InvalidateScopeData drops the traversal history of the scope.
//
// Flutter: widgets/focus_traversal.dart DirectionalFocusTraversalPolicyMixin.invalidateScopeData
func (b *FocusTraversalPolicyBase) InvalidateScopeData(node *FocusScopeNode) {
	if b.directional {
		delete(b.policyData, node)
	}
}

// ChangedScope forgets the node in the history of the scope it left.
//
// Flutter: widgets/focus_traversal.dart DirectionalFocusTraversalPolicyMixin.changedScope
func (b *FocusTraversalPolicyBase) ChangedScope(node *FocusNode, oldScope *FocusScopeNode) {
	if !b.directional || oldScope == nil {
		return
	}
	if data := b.policyData[oldScope]; data != nil {
		data.history = slices.DeleteFunc(data.history, func(e directionalPolicyEntry) bool { return e.node == node })
	}
}

// FindFirstFocusInDirection is the node at the opposite edge of the scope from
// the direction.
//
// Flutter: widgets/focus_traversal.dart DirectionalFocusTraversalPolicyMixin.findFirstFocusInDirection
func (b *FocusTraversalPolicyBase) FindFirstFocusInDirection(currentNode *FocusNode, direction TraversalDirection) *FocusNode {
	if !b.directional {
		return nil
	}
	sorted := slices.Clone(currentNode.NearestScope().TraversalDescendants())
	var vertical, first bool
	switch direction {
	case TraversalDirectionUp:
		vertical, first = true, false // starts with the bottom-most node
	case TraversalDirectionDown:
		vertical, first = true, true // starts with the topmost node
	case TraversalDirectionLeft:
		vertical, first = false, false // starts with the rightmost node
	case TraversalDirectionRight:
		vertical, first = false, true // starts with the leftmost node
	}
	slices.SortStableFunc(sorted, func(a, c *FocusNode) int {
		switch {
		case vertical && first:
			return cmp.Compare(a.Rect().Top, c.Rect().Top)
		case vertical:
			return cmp.Compare(c.Rect().Bottom, a.Rect().Bottom)
		case first:
			return cmp.Compare(a.Rect().Left, c.Rect().Left)
		}
		return cmp.Compare(c.Rect().Right, a.Rect().Right)
	})
	if len(sorted) == 0 {
		return nil
	}
	return sorted[0]
}

func verticalCompare(target, a, c ui.Offset) int {
	return cmp.Compare(math.Abs(a.Dy-target.Dy), math.Abs(c.Dy-target.Dy))
}

func horizontalCompare(target, a, c ui.Offset) int {
	return cmp.Compare(math.Abs(a.Dx-target.Dx), math.Abs(c.Dx-target.Dx))
}

// sortByDistancePreferVertical sorts the ones closest to the target vertically
// first, and the ones closest horizontally if two are at the same vertical
// distance.
func sortByDistancePreferVertical(target ui.Offset, nodes []*FocusNode) []*FocusNode {
	sorted := slices.Clone(nodes)
	slices.SortStableFunc(sorted, func(nodeA, nodeB *FocusNode) int {
		a, c := nodeA.Rect().Center(), nodeB.Rect().Center()
		if v := verticalCompare(target, a, c); v != 0 {
			return v
		}
		return horizontalCompare(target, a, c)
	})
	return sorted
}

// sortByDistancePreferHorizontal sorts the ones closest horizontally first, and
// the ones closest vertically if two are at the same horizontal distance.
func sortByDistancePreferHorizontal(target ui.Offset, nodes []*FocusNode) []*FocusNode {
	sorted := slices.Clone(nodes)
	slices.SortStableFunc(sorted, func(nodeA, nodeB *FocusNode) int {
		a, c := nodeA.Rect().Center(), nodeB.Rect().Center()
		if h := horizontalCompare(target, a, c); h != 0 {
			return h
		}
		return verticalCompare(target, a, c)
	})
	return sorted
}

func verticalCompareClosestEdge(target ui.Offset, a, c ui.Rect) int {
	// Finds which edge is closest to the target for each.
	aCoord := a.Bottom
	if math.Abs(a.Top-target.Dy) < math.Abs(a.Bottom-target.Dy) {
		aCoord = a.Top
	}
	cCoord := c.Bottom
	if math.Abs(c.Top-target.Dy) < math.Abs(c.Bottom-target.Dy) {
		cCoord = c.Top
	}
	return cmp.Compare(math.Abs(aCoord-target.Dy), math.Abs(cCoord-target.Dy))
}

func horizontalCompareClosestEdge(target ui.Offset, a, c ui.Rect) int {
	aCoord := a.Right
	if math.Abs(a.Left-target.Dx) < math.Abs(a.Right-target.Dx) {
		aCoord = a.Left
	}
	cCoord := c.Right
	if math.Abs(c.Left-target.Dx) < math.Abs(c.Right-target.Dx) {
		cCoord = c.Left
	}
	return cmp.Compare(math.Abs(aCoord-target.Dx), math.Abs(cCoord-target.Dx))
}

// sortClosestEdgesByDistancePreferHorizontal sorts the ones with edges closest
// horizontally first, and the closest vertically among those at the same
// horizontal distance.
func sortClosestEdgesByDistancePreferHorizontal(target ui.Offset, nodes []*FocusNode) []*FocusNode {
	sorted := slices.Clone(nodes)
	slices.SortStableFunc(sorted, func(nodeA, nodeB *FocusNode) int {
		if h := horizontalCompareClosestEdge(target, nodeA.Rect(), nodeB.Rect()); h != 0 {
			return h
		}
		return verticalCompare(target, nodeA.Rect().Center(), nodeB.Rect().Center())
	})
	return sorted
}

// sortClosestEdgesByDistancePreferVertical sorts the ones with edges closest
// vertically first, and the closest horizontally among those at the same
// vertical distance.
func sortClosestEdgesByDistancePreferVertical(target ui.Offset, nodes []*FocusNode) []*FocusNode {
	sorted := slices.Clone(nodes)
	slices.SortStableFunc(sorted, func(nodeA, nodeB *FocusNode) int {
		if v := verticalCompareClosestEdge(target, nodeA.Rect(), nodeB.Rect()); v != 0 {
			return v
		}
		return horizontalCompare(target, nodeA.Rect().Center(), nodeB.Rect().Center())
	})
	return sorted
}

// sortAndFilterHorizontally sorts the nodes from left to right and removes the
// ones that are to the right of the left side of the target when going left, or
// to the left of the right side of the target when going right.
func sortAndFilterHorizontally(direction TraversalDirection, target ui.Rect, nodes []*FocusNode, forward bool) []*FocusNode {
	var sorted []*FocusNode
	for _, node := range nodes {
		r := node.Rect()
		if r == target {
			continue
		}
		var ok bool
		switch direction {
		case TraversalDirectionLeft:
			ok = (forward && r.Center().Dx <= target.Left) || (!forward && r.Center().Dx >= target.Left)
		case TraversalDirectionRight:
			ok = (forward && r.Center().Dx >= target.Right) || (!forward && r.Center().Dx <= target.Right)
		default:
			panic(errors.New("TraversalDirectionInvalid", "invalid direction for a horizontal sort", "direction", direction.String()))
		}
		if ok {
			sorted = append(sorted, node)
		}
	}
	slices.SortStableFunc(sorted, func(a, c *FocusNode) int { return cmp.Compare(a.Rect().Center().Dx, c.Rect().Center().Dx) })
	return sorted
}

// sortAndFilterVertically sorts the nodes from top to bottom and removes the ones
// that are below the top of the target when going up, or above the bottom of the
// target when going down.
func sortAndFilterVertically(direction TraversalDirection, target ui.Rect, nodes []*FocusNode, forward bool) []*FocusNode {
	var sorted []*FocusNode
	for _, node := range nodes {
		r := node.Rect()
		if r == target {
			continue
		}
		var ok bool
		switch direction {
		case TraversalDirectionUp:
			ok = (forward && r.Center().Dy <= target.Top) || (!forward && r.Center().Dy >= target.Top)
		case TraversalDirectionDown:
			ok = (forward && r.Center().Dy >= target.Bottom) || (!forward && r.Center().Dy <= target.Bottom)
		default:
			panic(errors.New("TraversalDirectionInvalid", "invalid direction for a vertical sort", "direction", direction.String()))
		}
		if ok {
			sorted = append(sorted, node)
		}
	}
	slices.SortStableFunc(sorted, func(a, c *FocusNode) int { return cmp.Compare(a.Rect().Center().Dy, c.Rect().Center().Dy) })
	return sorted
}

func filterByScrollable(focusedChild *FocusNode, eligible []*FocusNode, axis painting.Axis) []*FocusNode {
	focusedScrollable := ScrollableMaybeOf(focusedChild.Context(), ScrollableOfOpts{Axis: foundation.Some(axis)})
	if focusedScrollable == nil {
		return eligible
	}
	var filtered []*FocusNode
	for _, node := range eligible {
		if ScrollableMaybeOf(node.Context(), ScrollableOfOpts{Axis: foundation.Some(axis)}) == focusedScrollable {
			filtered = append(filtered, node)
		}
	}
	if len(filtered) > 0 {
		return filtered
	}
	return eligible
}

func (b *FocusTraversalPolicyBase) findNextFocusInDirection(focusedChild *FocusNode, traversalDescendants []*FocusNode, direction TraversalDirection, forward bool) *FocusNode {
	infinity := math.Inf(1)
	switch direction {
	case TraversalDirectionDown, TraversalDirectionUp:
		eligible := sortAndFilterVertically(direction, focusedChild.Rect(), traversalDescendants, forward)
		if len(eligible) == 0 {
			break
		}
		eligible = filterByScrollable(focusedChild, eligible, painting.AxisVertical)
		if direction == TraversalDirectionUp {
			eligible = slices.Clone(eligible)
			slices.Reverse(eligible)
		}
		// Finds any nodes that intersect the band of the focused child.
		rect := focusedChild.Rect()
		band := ui.NewRectFromLTRB(rect.Left, -infinity, rect.Right, infinity)
		var inBand []*FocusNode
		for _, node := range eligible {
			if !node.Rect().Intersect(band).IsEmpty() {
				inBand = append(inBand, node)
			}
		}
		if len(inBand) > 0 {
			sorted := sortByDistancePreferVertical(rect.Center(), inBand)
			if forward {
				return sorted[0]
			}
			return sorted[len(sorted)-1]
		}
		// Only out-of-band targets are eligible: the one closest to the center line
		// horizontally is picked, and among those at the same horizontal distance
		// the closest one vertically.
		sorted := sortClosestEdgesByDistancePreferHorizontal(rect.Center(), eligible)
		if forward {
			return sorted[0]
		}
		return sorted[len(sorted)-1]
	case TraversalDirectionRight, TraversalDirectionLeft:
		eligible := sortAndFilterHorizontally(direction, focusedChild.Rect(), traversalDescendants, forward)
		if len(eligible) == 0 {
			break
		}
		eligible = filterByScrollable(focusedChild, eligible, painting.AxisHorizontal)
		if direction == TraversalDirectionLeft {
			eligible = slices.Clone(eligible)
			slices.Reverse(eligible)
		}
		rect := focusedChild.Rect()
		band := ui.NewRectFromLTRB(-infinity, rect.Top, infinity, rect.Bottom)
		var inBand []*FocusNode
		for _, node := range eligible {
			if !node.Rect().Intersect(band).IsEmpty() {
				inBand = append(inBand, node)
			}
		}
		if len(inBand) > 0 {
			sorted := sortByDistancePreferHorizontal(rect.Center(), inBand)
			if forward {
				return sorted[0]
			}
			return sorted[len(sorted)-1]
		}
		sorted := sortClosestEdgesByDistancePreferVertical(rect.Center(), eligible)
		if forward {
			return sorted[0]
		}
		return sorted[len(sorted)-1]
	}
	return nil
}

func alignmentForDirection(direction TraversalDirection) ScrollPositionAlignmentPolicy {
	switch direction {
	case TraversalDirectionUp, TraversalDirectionLeft:
		return ScrollPositionAlignmentPolicyKeepVisibleAtStart
	}
	return ScrollPositionAlignmentPolicyKeepVisibleAtEnd
}

// popPolicyDataIfNeeded updates the policy data to keep the previously visited
// node so that hysteresis is avoided when the direction of the navigation
// changes. It returns true if the focus was requested on a previous node.
func (b *FocusTraversalPolicyBase) popPolicyDataIfNeeded(direction TraversalDirection, nearestScope *FocusScopeNode, focusedChild *FocusNode) bool {
	policyData := b.policyData[nearestScope]
	if policyData != nil && len(policyData.history) > 0 && policyData.history[0].direction != direction {
		if policyData.history[len(policyData.history)-1].node.parent == nil {
			// A node removed from the tree is not referenced any more and the data
			// of the scope is reset so that the focus is not requested on it. This
			// can happen in slivers where the rendered node has been unmounted.
			b.self.InvalidateScopeData(nearestScope)
			return false
		}
		popOrInvalidate := func(direction TraversalDirection) bool {
			last := policyData.history[len(policyData.history)-1]
			policyData.history = policyData.history[:len(policyData.history)-1]
			lastNode := last.node
			primary := PrimaryFocus(focusedChild.Context())
			if ScrollableMaybeOf(lastNode.Context()) != ScrollableMaybeOf(primary.Context()) {
				b.self.InvalidateScopeData(nearestScope)
				return false
			}
			b.self.RequestFocusCallback()(lastNode, TraversalRequestFocusOpts{AlignmentPolicy: alignmentForDirection(direction)})
			return true
		}
		switch direction {
		case TraversalDirectionDown, TraversalDirectionUp:
			switch policyData.history[0].direction {
			case TraversalDirectionLeft, TraversalDirectionRight:
				// Resets the policy data if the direction changed.
				b.self.InvalidateScopeData(nearestScope)
			case TraversalDirectionUp, TraversalDirectionDown:
				if popOrInvalidate(direction) {
					return true
				}
			}
		case TraversalDirectionLeft, TraversalDirectionRight:
			switch policyData.history[0].direction {
			case TraversalDirectionLeft, TraversalDirectionRight:
				if popOrInvalidate(direction) {
					return true
				}
			case TraversalDirectionUp, TraversalDirectionDown:
				b.self.InvalidateScopeData(nearestScope)
			}
		}
	}
	if policyData != nil && len(policyData.history) == 0 {
		b.self.InvalidateScopeData(nearestScope)
	}
	return false
}

func (b *FocusTraversalPolicyBase) pushPolicyData(direction TraversalDirection, nearestScope *FocusScopeNode, focusedChild *FocusNode) {
	entry := directionalPolicyEntry{node: focusedChild, direction: direction}
	if data := b.policyData[nearestScope]; data != nil {
		data.history = append(data.history, entry)
	} else {
		b.policyData[nearestScope] = &directionalPolicyData{history: []directionalPolicyEntry{entry}}
	}
}

func (b *FocusTraversalPolicyBase) requestTraversalFocusInDirection(currentNode, node *FocusNode, nearestScope *FocusScopeNode, direction TraversalDirection) bool {
	if scope := node.AsScope(); scope != nil {
		if scope.FocusedChild() != nil {
			return b.requestTraversalFocusInDirection(currentNode, scope.FocusedChild(), scope, direction)
		}
		firstNode := b.self.FindFirstFocusInDirection(node, direction)
		if firstNode == nil {
			firstNode = currentNode
		}
		b.self.RequestFocusCallback()(firstNode, TraversalRequestFocusOpts{AlignmentPolicy: alignmentForDirection(direction)})
		return true
	}
	nodeHadPrimaryFocus := node.HasPrimaryFocus()
	b.self.RequestFocusCallback()(node, TraversalRequestFocusOpts{AlignmentPolicy: alignmentForDirection(direction)})
	return !nodeHadPrimaryFocus
}

func (b *FocusTraversalPolicyBase) onEdgeForDirection(currentNode, focusedChild *FocusNode, direction TraversalDirection, scope *FocusScopeNode) bool {
	nearestScope := scope
	if nearestScope == nil {
		nearestScope = currentNode.NearestScope()
	}
	var found *FocusNode
	switch nearestScope.directionalTraversalEdgeBehavior {
	case TraversalEdgeBehaviorLeaveFlutterView:
		focusedChild.Unfocus()
		return false
	case TraversalEdgeBehaviorParentScope:
		parentScope := nearestScope.EnclosingScope()
		var rootScope *FocusScopeNode
		if m := currentNode.manager; m != nil {
			rootScope = m.rootScope
		}
		if parentScope != nil && parentScope != rootScope {
			b.self.InvalidateScopeData(nearestScope)
			nearestScope = parentScope
			b.self.InvalidateScopeData(nearestScope)
			found = b.findNextFocusInDirection(focusedChild, nearestScope.TraversalDescendants(), direction, true)
			if found == nil {
				return b.onEdgeForDirection(currentNode, focusedChild, direction, nearestScope)
			}
		} else {
			found = b.findNextFocusInDirection(focusedChild, nearestScope.TraversalDescendants(), direction, false)
		}
	case TraversalEdgeBehaviorStop:
		return false
	default: // closed loop
		found = b.findNextFocusInDirection(focusedChild, nearestScope.TraversalDescendants(), direction, false)
	}
	if found != nil {
		return b.requestTraversalFocusInDirection(currentNode, found, nearestScope, direction)
	}
	return false
}

// InDirection moves the focus to the closest node in the direction; at the edge
// of the scope it does what the scope's directional edge behavior says.
//
// Flutter: widgets/focus_traversal.dart DirectionalFocusTraversalPolicyMixin.inDirection
func (b *FocusTraversalPolicyBase) InDirection(currentNode *FocusNode, direction TraversalDirection) bool {
	if !b.directional {
		return false
	}
	nearestScope := currentNode.NearestScope()
	focusedChild := nearestScope.FocusedChild()
	if focusedChild == nil {
		firstFocus := b.self.FindFirstFocusInDirection(currentNode, direction)
		if firstFocus == nil {
			firstFocus = currentNode
		}
		b.self.RequestFocusCallback()(firstFocus, TraversalRequestFocusOpts{AlignmentPolicy: alignmentForDirection(direction)})
		return true
	}
	if b.popPolicyDataIfNeeded(direction, nearestScope, focusedChild) {
		return true
	}
	found := b.findNextFocusInDirection(focusedChild, nearestScope.TraversalDescendants(), direction, true)
	if found != nil {
		b.pushPolicyData(direction, nearestScope, focusedChild)
		return b.requestTraversalFocusInDirection(currentNode, found, nearestScope, direction)
	}
	return b.onEdgeForDirection(currentNode, focusedChild, direction, nil)
}

// FocusTraversalPolicyOpts are the optional parameters of the constructors of the
// traversal policies.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalPolicy.new
type FocusTraversalPolicyOpts struct {
	// RequestFocusCallback replaces DefaultTraversalRequestFocusCallback.
	RequestFocusCallback TraversalRequestFocusCallback
}

// WidgetOrderTraversalPolicy traverses the nodes in the order the widgets were
// built (the order of the focus tree).
//
// Flutter: widgets/focus_traversal.dart WidgetOrderTraversalPolicy
type WidgetOrderTraversalPolicy struct {
	FocusTraversalPolicyBase
}

// NewWidgetOrderTraversalPolicy creates the policy.
//
// Flutter: widgets/focus_traversal.dart WidgetOrderTraversalPolicy.new
func NewWidgetOrderTraversalPolicy(opts ...FocusTraversalPolicyOpts) *WidgetOrderTraversalPolicy {
	p := &WidgetOrderTraversalPolicy{}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{
		RequestFocusCallback: firstOpts(opts).RequestFocusCallback, Directional: true})
	return p
}

// SortDescendants keeps the order of the nodes.
//
// Flutter: widgets/focus_traversal.dart WidgetOrderTraversalPolicy.sortDescendants
func (p *WidgetOrderTraversalPolicy) SortDescendants(descendants []*FocusNode, _ *FocusNode) []*FocusNode {
	return descendants
}

// readingOrderSortData copies the rect out of a node, because it is read many
// times in the reading order algorithm and FocusNode.Rect transforms
// coordinates, and keeps the directionality of its context.
//
// Flutter: widgets/focus_traversal.dart _ReadingOrderSortData
type readingOrderSortData struct {
	directionality    ui.TextDirection
	hasDirectionality bool
	rect              ui.Rect
	node              *FocusNode
	ancestors         []*Directionality
	ancestorsKnown    bool
}

func newReadingOrderSortData(node *FocusNode) *readingOrderSortData {
	d := &readingOrderSortData{rect: node.Rect(), node: node}
	// Finds the directionality in force for the context without creating a
	// dependency.
	if w, ok := GetInherited[*Directionality](node.Context()); ok {
		d.directionality, d.hasDirectionality = w.TextDirection(), true
	}
	return d
}

// directionalAncestors are the Directionality widgets above the node, nearest
// first.
func (d *readingOrderSortData) directionalAncestors() []*Directionality {
	if d.ancestorsKnown {
		return d.ancestors
	}
	var result []*Directionality
	el := GetElementForInheritedWidget[*Directionality](d.node.Context())
	for el != nil {
		result = append(result, el.AsElement().widget.(*Directionality))
		parent := getAncestor(el.(BuildContext), 1)
		if parent == nil {
			break
		}
		el = GetElementForInheritedWidget[*Directionality](parent)
	}
	d.ancestors, d.ancestorsKnown = result, true
	return result
}

// getAncestor is the count-th ancestor of the context. BuildContext has no parent
// accessor, but it can be simulated with VisitAncestorElements; the lookup
// GetElementForInheritedWidget returns the element itself if it has the type, so
// the parent is needed.
//
// Flutter: widgets/focus_traversal.dart _getAncestor
func getAncestor(bc BuildContext, count int) BuildContext {
	var target BuildContext
	bc.VisitAncestorElements(func(ancestor Element) bool {
		count--
		if count == 0 {
			target = ancestor.(BuildContext)
			return false
		}
		return true
	})
	return target
}

func commonDirectionalityOf(list []*readingOrderSortData) (ui.TextDirection, bool) {
	var common map[*Directionality]bool
	for _, member := range list {
		set := map[*Directionality]bool{}
		for _, a := range member.directionalAncestors() {
			set[a] = true
		}
		if common == nil {
			common = set
			continue
		}
		for k := range common {
			if !set[k] {
				delete(common, k)
			}
		}
	}
	if len(common) == 0 {
		// With no common ancestor, the directionality of the first group is picked
		// arbitrarily, the equivalent of the "first strongly typed" item in a
		// bidirectional algorithm.
		return list[0].directionality, list[0].hasDirectionality
	}
	// Finds the closest common ancestor: the ancestors of the first member are in
	// order from nearest to furthest.
	for _, a := range list[0].directionalAncestors() {
		if common[a] {
			return a.TextDirection(), true
		}
	}
	return list[0].directionality, list[0].hasDirectionality
}

func sortSortDataWithDirectionality(list []*readingOrderSortData, directionality ui.TextDirection) {
	slices.SortStableFunc(list, func(a, c *readingOrderSortData) int {
		if directionality == ui.TextDirectionRTL {
			return cmp.Compare(c.rect.Right, a.rect.Right)
		}
		return cmp.Compare(a.rect.Left, c.rect.Left)
	})
}

// readingOrderDirectionalGroupData contains the group data while sorting in
// reading order with the ambient directionality.
//
// Flutter: widgets/focus_traversal.dart _ReadingOrderDirectionalGroupData
type readingOrderDirectionalGroupData struct {
	members []*readingOrderSortData
	rect    ui.Rect
	hasRect bool
}

func (g *readingOrderDirectionalGroupData) directionality() (ui.TextDirection, bool) {
	return g.members[0].directionality, g.members[0].hasDirectionality
}

func (g *readingOrderDirectionalGroupData) boundingRect() ui.Rect {
	if !g.hasRect {
		for _, m := range g.members {
			if !g.hasRect {
				g.rect, g.hasRect = m.rect, true
			}
			g.rect = g.rect.ExpandToInclude(m.rect)
		}
	}
	return g.rect
}

func sortGroupsWithDirectionality(list []*readingOrderDirectionalGroupData, directionality ui.TextDirection) {
	slices.SortStableFunc(list, func(a, c *readingOrderDirectionalGroupData) int {
		if directionality == ui.TextDirectionRTL {
			return cmp.Compare(c.boundingRect().Right, a.boundingRect().Right)
		}
		return cmp.Compare(a.boundingRect().Left, c.boundingRect().Left)
	})
}

// ReadingOrderTraversalPolicy traverses the nodes top to bottom in rows, then in
// the reading direction of each row (left to right for ltr), treating the nodes
// whose rects overlap vertically as one row.
//
// Flutter: widgets/focus_traversal.dart ReadingOrderTraversalPolicy
type ReadingOrderTraversalPolicy struct {
	FocusTraversalPolicyBase
}

// NewReadingOrderTraversalPolicy creates the policy.
//
// Flutter: widgets/focus_traversal.dart ReadingOrderTraversalPolicy.new
func NewReadingOrderTraversalPolicy(opts ...FocusTraversalPolicyOpts) *ReadingOrderTraversalPolicy {
	p := &ReadingOrderTraversalPolicy{}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{
		RequestFocusCallback: firstOpts(opts).RequestFocusCallback, Directional: true})
	return p
}

// ReadingOrderSort sorts the nodes in reading order, with the directionality of
// the context of each node.
//
// Flutter: widgets/focus_traversal.dart ReadingOrderTraversalPolicy.sort
func ReadingOrderSort(nodes []*FocusNode) []*FocusNode {
	if len(nodes) <= 1 {
		return nodes
	}
	unplaced := make([]*readingOrderSortData, len(nodes))
	for i, node := range nodes {
		unplaced[i] = newReadingOrderSortData(node)
	}
	var sortedList []*FocusNode
	// Picks the initial node as the one at the beginning of the band of the
	// topmost, or the topmost if there are no others in its band.
	current := pickNextInReadingOrder(unplaced)
	sortedList = append(sortedList, current.node)
	unplaced = slices.DeleteFunc(unplaced, func(d *readingOrderSortData) bool { return d == current })
	// Removing the previously picked node exposes a new band in which to choose
	// candidates.
	for len(unplaced) > 0 {
		current = pickNextInReadingOrder(unplaced)
		sortedList = append(sortedList, current.node)
		unplaced = slices.DeleteFunc(unplaced, func(d *readingOrderSortData) bool { return d == current })
	}
	return sortedList
}

// collectDirectionalityGroups collects the candidates, already sorted as if all
// had the directionality of the nearest Directionality ancestor, into groups of
// the same directionality.
func collectDirectionalityGroups(candidates []*readingOrderSortData) []*readingOrderDirectionalGroupData {
	currentDirection, currentKnown := candidates[0].directionality, candidates[0].hasDirectionality
	var currentGroup []*readingOrderSortData
	var result []*readingOrderDirectionalGroupData
	// Splits the candidates into runs of the same directionality.
	for _, candidate := range candidates {
		if candidate.directionality == currentDirection && candidate.hasDirectionality == currentKnown {
			currentGroup = append(currentGroup, candidate)
			continue
		}
		currentDirection, currentKnown = candidate.directionality, candidate.hasDirectionality
		result = append(result, &readingOrderDirectionalGroupData{members: currentGroup})
		currentGroup = []*readingOrderSortData{candidate}
	}
	if len(currentGroup) > 0 {
		result = append(result, &readingOrderDirectionalGroupData{members: currentGroup})
	}
	// Sorts each group separately; each has the same directionality.
	for _, bandGroup := range result {
		if len(bandGroup.members) == 1 {
			continue // no need to sort one node
		}
		d, _ := bandGroup.directionality()
		sortSortDataWithDirectionality(bandGroup.members, d)
	}
	return result
}

func pickNextInReadingOrder(candidates []*readingOrderSortData) *readingOrderSortData {
	// Finds the topmost node by sorting on the top of the rectangles.
	slices.SortStableFunc(candidates, func(a, c *readingOrderSortData) int { return cmp.Compare(a.rect.Top, c.rect.Top) })
	topmost := candidates[0]
	// Finds the candidates that are in the same horizontal band as the topmost.
	band := ui.NewRectFromLTRB(math.Inf(-1), topmost.rect.Top, math.Inf(1), topmost.rect.Bottom)
	var inBandOfTop []*readingOrderSortData
	for _, item := range candidates {
		if !item.rect.Intersect(band).IsEmpty() {
			inBandOfTop = append(inBandOfTop, item)
		}
	}
	// The topmost rect is in a band by itself.
	if len(inBandOfTop) <= 1 {
		return topmost
	}
	// The one at the beginning is picked, depending on the text direction in
	// force. The directionality of the nearest common Directionality ancestor of
	// all nodes is the base directionality for the ordering of the groups.
	nearestCommonDirectionality, _ := commonDirectionalityOf(inBandOfTop)
	// An initial sort with the common directionality gives a consistent geometric
	// ordering for grouping into directionality groups: rectangles can overlap and
	// give different results for different directionalities.
	sortSortDataWithDirectionality(inBandOfTop, nearestCommonDirectionality)
	// Collects the top band into internally sorted groups with shared
	// directionality.
	bandGroups := collectDirectionalityGroups(inBandOfTop)
	if len(bandGroups) == 1 {
		// Only one directionality group: its first member, already sorted.
		return bandGroups[0].members[0]
	}
	// Sorts the groups based on the common directionality and bounding boxes.
	sortGroupsWithDirectionality(bandGroups, nearestCommonDirectionality)
	return bandGroups[0].members[0]
}

// SortDescendants sorts the nodes into reading order.
//
// Flutter: widgets/focus_traversal.dart ReadingOrderTraversalPolicy.sortDescendants
func (p *ReadingOrderTraversalPolicy) SortDescendants(descendants []*FocusNode, _ *FocusNode) []*FocusNode {
	return ReadingOrderSort(descendants)
}

// FocusOrder is the order of a node, for OrderedTraversalPolicy; the orders that
// are compared must be of the same kind.
//
// Flutter: widgets/focus_traversal.dart FocusOrder
type FocusOrder interface {
	foundation.Diagnosticable
	// doCompare compares with an order of the same kind.
	doCompare(other FocusOrder) int
}

// CompareFocusOrders compares two orders; it panics with the tag
// FocusOrderIncomparable when they are not of the same kind.
//
// Flutter: widgets/focus_traversal.dart FocusOrder.compareTo
func CompareFocusOrders(a, b FocusOrder) int {
	if fmt.Sprintf("%T", a) != fmt.Sprintf("%T", b) {
		panic(errors.New("FocusOrderIncomparable",
			"the sorting algorithm must not compare incomparable keys, since they do not know how to order themselves relative to each other",
			"a", fmt.Sprint(a), "b", fmt.Sprint(b)))
	}
	return a.doCompare(b)
}

// NumericFocusOrder is a focus order given by a number.
//
// Flutter: widgets/focus_traversal.dart NumericFocusOrder
type NumericFocusOrder struct {
	// Order is the number; the lowest comes first.
	Order float64
}

// NewNumericFocusOrder creates the order.
//
// Flutter: widgets/focus_traversal.dart NumericFocusOrder.new
func NewNumericFocusOrder(order float64) *NumericFocusOrder { return &NumericFocusOrder{order} }

func (o *NumericFocusOrder) doCompare(other FocusOrder) int {
	return cmp.Compare(o.Order, other.(*NumericFocusOrder).Order)
}

// DebugFillProperties adds the order.
//
// Flutter: widgets/focus_traversal.dart NumericFocusOrder.debugFillProperties
func (o *NumericFocusOrder) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDoubleProperty("order", foundation.Some(o.Order)))
}

// LexicalFocusOrder is a focus order given by a string, compared lexically.
//
// Flutter: widgets/focus_traversal.dart LexicalFocusOrder
type LexicalFocusOrder struct {
	// Order is the string; the lowest comes first.
	Order string
}

// NewLexicalFocusOrder creates the order.
//
// Flutter: widgets/focus_traversal.dart LexicalFocusOrder.new
func NewLexicalFocusOrder(order string) *LexicalFocusOrder { return &LexicalFocusOrder{order} }

func (o *LexicalFocusOrder) doCompare(other FocusOrder) int {
	return cmp.Compare(o.Order, other.(*LexicalFocusOrder).Order)
}

// DebugFillProperties adds the order.
//
// Flutter: widgets/focus_traversal.dart LexicalFocusOrder.debugFillProperties
func (o *LexicalFocusOrder) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("order", foundation.Some(o.Order)))
}

// OrderedTraversalPolicyOpts are the optional parameters of
// NewOrderedTraversalPolicy.
//
// Flutter: widgets/focus_traversal.dart OrderedTraversalPolicy.new
type OrderedTraversalPolicyOpts struct {
	// Secondary orders the nodes that have no FocusTraversalOrder, and the nodes
	// first; the default is ReadingOrderTraversalPolicy.
	Secondary FocusTraversalPolicy
	// RequestFocusCallback replaces DefaultTraversalRequestFocusCallback.
	RequestFocusCallback TraversalRequestFocusCallback
}

// OrderedTraversalPolicy traverses the nodes that have a FocusTraversalOrder in
// that order, followed by the other nodes ordered by the secondary policy.
//
// Flutter: widgets/focus_traversal.dart OrderedTraversalPolicy
type OrderedTraversalPolicy struct {
	FocusTraversalPolicyBase
	secondary FocusTraversalPolicy
}

// NewOrderedTraversalPolicy creates the policy.
//
// Flutter: widgets/focus_traversal.dart OrderedTraversalPolicy.new
func NewOrderedTraversalPolicy(opts ...OrderedTraversalPolicyOpts) *OrderedTraversalPolicy {
	o := firstOpts(opts)
	p := &OrderedTraversalPolicy{secondary: o.Secondary}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{
		RequestFocusCallback: o.RequestFocusCallback, Directional: true})
	return p
}

// SortDescendants sorts the nodes with a FocusTraversalOrder by it, and puts them
// before the others.
//
// Flutter: widgets/focus_traversal.dart OrderedTraversalPolicy.sortDescendants
func (p *OrderedTraversalPolicy) SortDescendants(descendants []*FocusNode, currentNode *FocusNode) []*FocusNode {
	secondaryPolicy := p.secondary
	if secondaryPolicy == nil {
		secondaryPolicy = NewReadingOrderTraversalPolicy()
	}
	sortedDescendants := secondaryPolicy.SortDescendants(descendants, currentNode)
	type orderedInfo struct {
		node  *FocusNode
		order FocusOrder
	}
	var unordered []*FocusNode
	var ordered []orderedInfo
	for _, node := range sortedDescendants {
		if order := FocusTraversalOrderMaybeOf(node.Context()); order != nil {
			ordered = append(ordered, orderedInfo{node, order})
		} else {
			unordered = append(unordered, node)
		}
	}
	slices.SortStableFunc(ordered, func(a, c orderedInfo) int { return CompareFocusOrders(a.order, c.order) })
	out := make([]*FocusNode, 0, len(descendants))
	for _, info := range ordered {
		out = append(out, info.node)
	}
	return append(out, unordered...)
}

// FocusTraversalOrder is a widget that describes the order of its descendant
// focus nodes in an OrderedTraversalPolicy.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder
type FocusTraversalOrder struct {
	InheritedWidget
	order FocusOrder
}

// NewFocusTraversalOrder creates the widget.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.new
func NewFocusTraversalOrder(order FocusOrder, child Widget, opts ...WidgetOpts) *FocusTraversalOrder {
	return &FocusTraversalOrder{NewInheritedWidgetBase(child, opts...), order}
}

// Order is the order of the descendant nodes.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.order
func (w *FocusTraversalOrder) Order() FocusOrder { return w.order }

// UpdateShouldNotify is false: the order of traversal does not affect the display
// of anything, so nothing needs to rebuild.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.updateShouldNotify
func (w *FocusTraversalOrder) UpdateShouldNotify(Widget) bool { return false }

// DebugFillProperties adds the order.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.debugFillProperties
func (w *FocusTraversalOrder) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("order", w.order))
}

// FocusTraversalOrderOf is the order of the nearest FocusTraversalOrder above the
// context. It panics with the tag FocusTraversalOrderNotFound when there is none.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.of
func FocusTraversalOrderOf(bc BuildContext) FocusOrder {
	marker, ok := GetInherited[*FocusTraversalOrder](bc)
	if !ok {
		panic(errors.New("FocusTraversalOrderNotFound",
			"FocusTraversalOrderOf() was called with a context that does not contain a FocusTraversalOrder widget; no FocusTraversalOrder widget ancestor could be found starting from the context that was passed to FocusTraversalOrderOf()",
			"context", describeContext(bc)))
	}
	return marker.order
}

// FocusTraversalOrderMaybeOf is FocusTraversalOrderOf that returns nil instead of
// panicking.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalOrder.maybeOf
func FocusTraversalOrderMaybeOf(bc BuildContext) FocusOrder {
	if marker, ok := GetInherited[*FocusTraversalOrder](bc); ok {
		return marker.order
	}
	return nil
}

// focusTraversalGroupData is what a FocusTraversalGroup's node holds.
//
// Flutter: widgets/focus_traversal.dart _FocusTraversalGroupNode
type focusTraversalGroupData struct {
	policy FocusTraversalPolicy
}

// focusTraversalGroupNodeOf is the nearest node of a FocusTraversalGroup, from the
// node up, or nil.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup._getGroupNode
func focusTraversalGroupNodeOf(node *FocusNode) *FocusNode {
	for node != nil && node.parent != nil {
		if node.ctx == nil {
			return nil
		}
		if node.group != nil {
			return node
		}
		node = node.parent
	}
	return nil
}

// FocusTraversalGroupOpts are the optional parameters of NewFocusTraversalGroup.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.new
type FocusTraversalGroupOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Policy is the policy of the group; the default is
	// ReadingOrderTraversalPolicy.
	Policy FocusTraversalPolicy
	// DescendantsAreNotFocusable makes the descendants unable to take the focus
	// (Flutter's descendantsAreFocusable is true by default).
	DescendantsAreNotFocusable bool
	// DescendantsAreNotTraversable makes traversal skip the descendants (Flutter's
	// descendantsAreTraversable is true by default).
	DescendantsAreNotTraversable bool
	// OnFocusNodeCreated is called with the node the group makes.
	OnFocusNodeCreated func(*FocusNode)
	// ParentNode is the node that becomes the parent of the group's node.
	ParentNode *FocusNode
}

// FocusTraversalGroup establishes a policy for the focus traversal of the
// widgets below it, and groups them: traversal visits all the nodes of a group
// before it goes on to the next node.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup
type FocusTraversalGroup struct {
	StatefulWidget
	policy FocusTraversalPolicy
	o      FocusTraversalGroupOpts
	child  Widget
}

// NewFocusTraversalGroup creates the widget.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.new
func NewFocusTraversalGroup(child Widget, opts ...FocusTraversalGroupOpts) *FocusTraversalGroup {
	o := firstOpts(opts)
	policy := o.Policy
	if policy == nil {
		policy = NewReadingOrderTraversalPolicy()
	}
	return &FocusTraversalGroup{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), policy, o, child}
}

// Policy is the policy of the group.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.policy
func (w *FocusTraversalGroup) Policy() FocusTraversalPolicy { return w.policy }

// Child is the widget below.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.child
func (w *FocusTraversalGroup) Child() Widget { return w.child }

// FocusTraversalGroupMaybeOfNode is the policy of the group the node is in, or
// nil.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.maybeOfNode
func FocusTraversalGroupMaybeOfNode(node *FocusNode) FocusTraversalPolicy {
	if g := focusTraversalGroupNodeOf(node); g != nil {
		return g.group.policy
	}
	return nil
}

// FocusTraversalGroupOf is the policy of the nearest FocusTraversalGroup above
// the context. It panics with the tag FocusTraversalPolicyMissing when there is
// none.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.of
func FocusTraversalGroupOf(bc BuildContext) FocusTraversalPolicy {
	policy := FocusTraversalGroupMaybeOf(bc)
	if policy == nil {
		panic(errors.New("FocusTraversalPolicyMissing",
			"unable to find a Focus or FocusScope widget in the given context, or the FocusNode from the widget that was found is not associated with a FocusTraversalPolicy; this can happen if there is not a FocusTraversalGroup that defines the policy, or if the context comes from a widget that is above the WidgetsApp, which introduces an implicit default policy",
			"context", describeContext(bc)))
	}
	return policy
}

// FocusTraversalGroupMaybeOf is FocusTraversalGroupOf that returns nil instead of
// panicking.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.maybeOf
func FocusTraversalGroupMaybeOf(bc BuildContext) FocusTraversalPolicy {
	if bc == nil {
		return nil
	}
	node := FocusMaybeOf(bc, FocusOfOpts{ScopeOK: true, CreateDependency: foundation.Some(false)})
	if node == nil {
		return nil
	}
	return FocusTraversalGroupMaybeOfNode(node)
}

// DebugFillProperties adds the policy.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.debugFillProperties
func (w *FocusTraversalGroup) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("policy", w.policy))
}

// CreateState creates the state.
//
// Flutter: widgets/focus_traversal.dart FocusTraversalGroup.createState
func (w *FocusTraversalGroup) CreateState() State { return &focusTraversalGroupState{} }

type focusTraversalGroupState struct {
	StateOf[*FocusTraversalGroup]
	// focusNode collects the children of the group, and gives the traversal
	// algorithm a context to sort with.
	focusNode *FocusNode
}

func (s *focusTraversalGroupState) InitState() {
	s.focusNode = NewFocusNode(FocusNodeOpts{DebugLabel: "FocusTraversalGroup"})
	s.focusNode.group = &focusTraversalGroupData{policy: s.Widget().policy}
	if cb := s.Widget().o.OnFocusNodeCreated; cb != nil {
		cb(s.focusNode)
	}
}

func (s *focusTraversalGroupState) Dispose() { s.focusNode.Dispose() }

func (s *focusTraversalGroupState) DidUpdateWidget(old *FocusTraversalGroup) {
	if old.policy != s.Widget().policy {
		s.focusNode.group.policy = s.Widget().policy
	}
}

func (s *focusTraversalGroupState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewFocus(w.child, FocusOpts{
		FocusNode:                 s.focusNode,
		ParentNode:                w.o.ParentNode,
		CanRequestFocus:           foundation.Some(false),
		SkipTraversal:             foundation.Some(true),
		ExcludeSemantics:          true,
		DescendantsAreFocusable:   foundation.Some(!w.o.DescendantsAreNotFocusable),
		DescendantsAreTraversable: foundation.Some(!w.o.DescendantsAreNotTraversable),
	})
}

// ExcludeFocusTraversalOpts are the optional parameters of
// NewExcludeFocusTraversal.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal.new
type ExcludeFocusTraversalOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NotExcluding turns the exclusion off (Flutter's excluding is true by
	// default).
	NotExcluding bool
}

// ExcludeFocusTraversal excludes its subtree from focus traversal: its nodes can
// still take the focus, but traversal skips them.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal
type ExcludeFocusTraversal struct {
	StatelessWidget
	excluding bool
	child     Widget
}

// NewExcludeFocusTraversal creates the widget.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal.new
func NewExcludeFocusTraversal(child Widget, opts ...ExcludeFocusTraversalOpts) *ExcludeFocusTraversal {
	o := firstOpts(opts)
	return &ExcludeFocusTraversal{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), !o.NotExcluding, child}
}

// Excluding is whether the subtree is excluded.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal.excluding
func (w *ExcludeFocusTraversal) Excluding() bool { return w.excluding }

// Child is the widget below.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal.child
func (w *ExcludeFocusTraversal) Child() Widget { return w.child }

// Build makes a Focus that is skipped by traversal and has descendants that are
// not traversable.
//
// Flutter: widgets/focus_traversal.dart ExcludeFocusTraversal.build
func (w *ExcludeFocusTraversal) Build(BuildContext) Widget {
	return NewFocus(w.child, FocusOpts{
		CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true), ExcludeSemantics: true,
		DescendantsAreTraversable: foundation.Some(!w.excluding),
	})
}
