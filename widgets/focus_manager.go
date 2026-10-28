package widgets

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// KeyEventResult is what a key event handler says about the event. The zero
// value, KeyEventResultUnspecified, counts as KeyEventResultIgnored.
//
// Flutter: widgets/focus_manager.dart KeyEventResult
type KeyEventResult int

// The values of KeyEventResult follow Flutter's declaration order.
const (
	// KeyEventResultUnspecified is the zero value; it is treated as ignored.
	//
	// Flutter: no counterpart
	KeyEventResultUnspecified KeyEventResult = iota
	// KeyEventResultHandled says the handler used the event: nothing else sees it.
	//
	// Flutter: widgets/focus_manager.dart KeyEventResult.handled
	KeyEventResultHandled
	// KeyEventResultIgnored says the handler did not use the event: the next
	// handler sees it.
	//
	// Flutter: widgets/focus_manager.dart KeyEventResult.ignored
	KeyEventResultIgnored
	// KeyEventResultSkipRemainingHandlers says the handler did not use the event
	// but no other focus node sees it: it goes on to the platform.
	//
	// Flutter: widgets/focus_manager.dart KeyEventResult.skipRemainingHandlers
	KeyEventResultSkipRemainingHandlers
)

// String is the name of the result, like Dart's enum toString.
//
// Flutter: widgets/focus_manager.dart KeyEventResult
func (r KeyEventResult) String() string {
	switch r {
	case KeyEventResultHandled:
		return "KeyEventResult.handled"
	case KeyEventResultIgnored:
		return "KeyEventResult.ignored"
	case KeyEventResultSkipRemainingHandlers:
		return "KeyEventResult.skipRemainingHandlers"
	}
	return "KeyEventResult.unspecified"
}

// CombineKeyEventResults combines the results of several handlers: handled if any
// handled, else skipRemainingHandlers if any said so, else ignored.
//
// Flutter: widgets/focus_manager.dart combineKeyEventResults
func CombineKeyEventResults(results []KeyEventResult) KeyEventResult {
	hasSkipRemainingHandlers := false
	for _, result := range results {
		switch result {
		case KeyEventResultHandled:
			return KeyEventResultHandled
		case KeyEventResultSkipRemainingHandlers:
			hasSkipRemainingHandlers = true
		}
	}
	if hasSkipRemainingHandlers {
		return KeyEventResultSkipRemainingHandlers
	}
	return KeyEventResultIgnored
}

// FocusOnKeyEventCallback is the type of the function that a focus node calls
// for a key event, when it or a descendant has the focus.
//
// Deviation: Flutter's deprecated FocusOnKeyCallback (RawKeyEvent) is not ported
// (SPEC §12.10).
//
// Flutter: widgets/focus_manager.dart FocusOnKeyEventCallback
type FocusOnKeyEventCallback func(node *FocusNode, event services.KeyEvent) KeyEventResult

// OnKeyEventCallback is the type of the early and late key event handlers of the
// FocusManager.
//
// Flutter: widgets/focus_manager.dart OnKeyEventCallback
type OnKeyEventCallback func(event services.KeyEvent) KeyEventResult

// FocusHandlerHandle identifies a handler or a listener that was added to a
// FocusManager, to remove it (Go functions are not comparable).
//
// Flutter: no counterpart
type FocusHandlerHandle struct{ id uint64 }

// UnfocusDisposition is where the focus goes when a node is unfocused.
//
// Flutter: widgets/focus_manager.dart UnfocusDisposition
type UnfocusDisposition int

// The values of UnfocusDisposition follow Flutter's declaration order.
const (
	// UnfocusDispositionUnspecified is the zero value: the default, scope.
	//
	// Flutter: no counterpart
	UnfocusDispositionUnspecified UnfocusDisposition = iota
	// UnfocusDispositionScope focuses the nearest focusable enclosing scope.
	//
	// Flutter: widgets/focus_manager.dart UnfocusDisposition.scope
	UnfocusDispositionScope
	// UnfocusDispositionPreviouslyFocusedChild focuses the previously focused
	// child of the nearest scope, or the scope itself if there is none.
	//
	// Flutter: widgets/focus_manager.dart UnfocusDisposition.previouslyFocusedChild
	UnfocusDispositionPreviouslyFocusedChild
)

// autofocus is a pending autofocus request.
//
// Flutter: widgets/focus_manager.dart _Autofocus
type autofocus struct {
	scope         *FocusScopeNode
	autofocusNode *FocusNode
}

// applyIfValid applies the request if the node is still attached to the scope
// and the scope has no focused child.
//
// Flutter: widgets/focus_manager.dart _Autofocus.applyIfValid
func (a autofocus) applyIfValid(m *FocusManager) {
	shouldApply := (a.scope.parent != nil || a.scope == m.rootScope) &&
		a.scope.manager == m &&
		a.scope.FocusedChild() == nil &&
		slices.Contains(a.autofocusNode.Ancestors(), &a.scope.FocusNode)
	if shouldApply {
		m.focusDebug(func() string { return "Applying autofocus: " + a.autofocusNode.String() }, nil)
		a.autofocusNode.doRequestFocus(true)
	} else {
		m.focusDebug(func() string { return "Autofocus request discarded for node: " + a.autofocusNode.String() + "." }, nil)
	}
}

// FocusAttachment lets a widget attach a FocusNode to the focus tree and
// maintain the node's place in it: Focus and FocusScope do. The node's
// Attach makes one.
//
// Flutter: widgets/focus_manager.dart FocusAttachment
type FocusAttachment struct {
	node *FocusNode
}

// IsAttached is whether the node is attached to this attachment.
//
// Flutter: widgets/focus_manager.dart FocusAttachment.isAttached
func (a *FocusAttachment) IsAttached() bool { return a.node.attachment == a }

// Detach removes the node from the focus tree (unfocusing it first if it had the
// focus), so that it stops being notified.
//
// Flutter: widgets/focus_manager.dart FocusAttachment.detach
func (a *FocusAttachment) Detach() {
	n := a.node
	if n.manager != nil {
		n.manager.focusDebug(func() string { return "Detaching node:" }, func() []any {
			return []any{n, "With enclosing scope " + describeScopeNode(n.EnclosingScope())}
		})
	}
	if !a.IsAttached() {
		return
	}
	n.leaveTree()
}

// leaveTree takes the node out of the focus tree: it gives up the focus and does
// not send notifications any more.
func (n *FocusNode) leaveTree() {
	if n.HasPrimaryFocus() || (n.manager != nil && n.manager.markedForFocus == n) {
		n.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
	}
	// The node is no longer in the tree, so it should not send notifications.
	if n.manager != nil {
		n.manager.markDetached(n)
	}
	if n.parent != nil {
		n.parent.removeChild(n, true)
	}
	n.attachment = nil
}

// Reparent makes the node a child of the parent given, or of the nearest Focus
// or FocusScope above its context, or of the root scope. It does nothing for a
// detached attachment.
//
// Flutter: widgets/focus_manager.dart FocusAttachment.reparent
func (a *FocusAttachment) Reparent(parent ...*FocusNode) {
	if !a.IsAttached() {
		return
	}
	n := a.node
	var p *FocusNode
	if len(parent) > 0 {
		p = parent[0]
	}
	if p == nil {
		p = FocusMaybeOf(n.ctx, FocusOfOpts{ScopeOK: true})
	}
	if p == nil {
		p = &n.ctx.Owner().FocusManager().rootScope.FocusNode
	}
	p.reparent(n)
}

// FocusNodeOpts are the optional parameters of NewFocusNode.
//
// Flutter: widgets/focus_manager.dart FocusNode.new
type FocusNodeOpts struct {
	// DebugLabel names the node in debug output.
	DebugLabel string
	// OnKeyEvent is called for a key event when the node or a descendant has the
	// focus.
	OnKeyEvent FocusOnKeyEventCallback
	// SkipTraversal makes focus traversal skip the node.
	SkipTraversal bool
	// CannotRequestFocus makes the node unable to take the focus (Flutter's
	// canRequestFocus is true by default).
	CannotRequestFocus bool
	// DescendantsAreNotFocusable makes the descendants unable to take the focus
	// (Flutter's descendantsAreFocusable is true by default).
	DescendantsAreNotFocusable bool
	// DescendantsAreNotTraversable makes focus traversal skip the descendants
	// (Flutter's descendantsAreTraversable is true by default).
	DescendantsAreNotTraversable bool
}

// FocusNode is an object that can be used by a stateful widget to obtain the
// keyboard focus and to handle keyboard events. A node is part of the focus tree
// once a widget attaches it with Attach; the FocusManager keeps one primary
// focus in the tree.
//
// A FocusScopeNode is a FocusNode too: it embeds one, and AsScope turns a node
// back into its scope.
//
// Deviation: Flutter's FocusNode can be subclassed to override its getters; a
// node here is a plain struct (SPEC §12.10). A node is created with NewFocusNode,
// is disposed with Dispose, and the focus tree is reached through a
// FocusManager, never a global.
//
// Flutter: widgets/focus_manager.dart FocusNode
type FocusNode struct {
	foundation.ChangeNotifier

	debugLabel                string
	onKeyEvent                FocusOnKeyEventCallback
	skipTraversal             bool
	canRequestFocus           bool
	descendantsAreFocusable   bool
	descendantsAreTraversable bool
	ctx                       BuildContext

	manager          *FocusManager
	ancestors        []*FocusNode
	ancestorsValid   bool
	descendants      []*FocusNode
	descendantsValid bool
	hasKeyboardToken bool
	parent           *FocusNode
	children         []*FocusNode
	attachment       *FocusAttachment

	enclosingScopeCache        *FocusScopeNode
	requestFocusWhenReparented bool

	// scope is the scope this node is the base of, or nil.
	scope *FocusScopeNode
	// group is the traversal group data, when a FocusTraversalGroup made the node.
	group *focusTraversalGroupData
}

// NewFocusNode creates a focus node.
//
// Flutter: widgets/focus_manager.dart FocusNode.new
func NewFocusNode(opts ...FocusNodeOpts) *FocusNode {
	o := firstOpts(opts)
	n := &FocusNode{
		debugLabel:                o.DebugLabel,
		onKeyEvent:                o.OnKeyEvent,
		skipTraversal:             o.SkipTraversal,
		canRequestFocus:           !o.CannotRequestFocus,
		descendantsAreFocusable:   !o.DescendantsAreNotFocusable,
		descendantsAreTraversable: !o.DescendantsAreNotTraversable,
	}
	return n
}

// AsScope is the scope that the node is the base of, or nil for a node that is
// not a scope (Dart's "node is FocusScopeNode").
//
// Flutter: widgets/focus_manager.dart FocusScopeNode
func (n *FocusNode) AsScope() *FocusScopeNode { return n.scope }

// SkipTraversal is whether the node is skipped by focus traversal: it is when it
// or one of its ancestors says so.
//
// Flutter: widgets/focus_manager.dart FocusNode.skipTraversal
func (n *FocusNode) SkipTraversal() bool {
	if n.skipTraversal {
		return true
	}
	for _, ancestor := range n.Ancestors() {
		if !ancestor.descendantsAreTraversable {
			return true
		}
	}
	return false
}

// SetSkipTraversal sets whether focus traversal skips the node.
//
// Flutter: widgets/focus_manager.dart FocusNode.skipTraversal
func (n *FocusNode) SetSkipTraversal(value bool) {
	if value != n.skipTraversal {
		n.skipTraversal = value
		if n.manager != nil {
			n.manager.markPropertiesChanged(n)
		}
	}
}

// CanRequestFocus is whether the node can take the focus: it can when it says so
// and all its ancestors let their descendants be focusable.
//
// Flutter: widgets/focus_manager.dart FocusNode.canRequestFocus
func (n *FocusNode) CanRequestFocus() bool {
	if !n.canRequestFocus {
		return false
	}
	for _, ancestor := range n.Ancestors() {
		if !ancestor.DescendantsAreFocusable() {
			return false
		}
	}
	return true
}

// SetCanRequestFocus sets whether the node can take the focus. A node that has
// the focus and cannot any more loses it.
//
// Flutter: widgets/focus_manager.dart FocusNode.canRequestFocus
func (n *FocusNode) SetCanRequestFocus(value bool) {
	if value != n.canRequestFocus {
		// This is set first because Unfocus checks it to cull unfocusable,
		// previously focused children.
		n.canRequestFocus = value
		if n.HasFocus() && !value {
			n.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
		}
		if n.manager != nil {
			n.manager.markPropertiesChanged(n)
		}
	}
}

// DescendantsAreFocusable is whether the descendants of the node can take the
// focus. For a scope it also needs the scope itself to be focusable.
//
// Flutter: widgets/focus_manager.dart FocusNode.descendantsAreFocusable
func (n *FocusNode) DescendantsAreFocusable() bool {
	if n.scope != nil {
		return n.canRequestFocus && n.descendantsAreFocusable
	}
	return n.descendantsAreFocusable
}

// SetDescendantsAreFocusable sets whether the descendants can take the focus.
// The descendants that have the focus lose it.
//
// Flutter: widgets/focus_manager.dart FocusNode.descendantsAreFocusable
func (n *FocusNode) SetDescendantsAreFocusable(value bool) {
	if value == n.descendantsAreFocusable {
		return
	}
	// This is set before unfocusing, so that the scope does not focus any of the
	// children again.
	n.descendantsAreFocusable = value
	if !value && n.HasFocus() {
		n.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
	}
	if n.manager != nil {
		n.manager.markPropertiesChanged(n)
	}
}

// DescendantsAreTraversable is whether focus traversal visits the descendants.
//
// Flutter: widgets/focus_manager.dart FocusNode.descendantsAreTraversable
func (n *FocusNode) DescendantsAreTraversable() bool { return n.descendantsAreTraversable }

// SetDescendantsAreTraversable sets whether focus traversal visits the
// descendants.
//
// Flutter: widgets/focus_manager.dart FocusNode.descendantsAreTraversable
func (n *FocusNode) SetDescendantsAreTraversable(value bool) {
	if value != n.descendantsAreTraversable {
		n.descendantsAreTraversable = value
		if n.manager != nil {
			n.manager.markPropertiesChanged(n)
		}
	}
}

// Context is the context of the widget that attached the node, or nil.
//
// Flutter: widgets/focus_manager.dart FocusNode.context
func (n *FocusNode) Context() BuildContext { return n.ctx }

// OnKeyEvent is the function called for a key event when the node or a
// descendant has the focus.
//
// Flutter: widgets/focus_manager.dart FocusNode.onKeyEvent
func (n *FocusNode) OnKeyEvent() FocusOnKeyEventCallback { return n.onKeyEvent }

// SetOnKeyEvent sets the function called for a key event.
//
// Flutter: widgets/focus_manager.dart FocusNode.onKeyEvent
func (n *FocusNode) SetOnKeyEvent(f FocusOnKeyEventCallback) { n.onKeyEvent = f }

// Parent is the node above in the focus tree, or nil.
//
// Flutter: widgets/focus_manager.dart FocusNode.parent
func (n *FocusNode) Parent() *FocusNode { return n.parent }

// Children are the nodes below, in the order they were attached.
//
// Flutter: widgets/focus_manager.dart FocusNode.children
func (n *FocusNode) Children() []*FocusNode { return slices.Clone(n.children) }

// TraversalChildren are the children that focus traversal can visit.
//
// Flutter: widgets/focus_manager.dart FocusNode.traversalChildren
func (n *FocusNode) TraversalChildren() []*FocusNode {
	if n.scope != nil && !n.CanRequestFocus() {
		return nil
	}
	if !n.DescendantsAreFocusable() {
		return nil
	}
	var out []*FocusNode
	for _, child := range n.children {
		if !child.SkipTraversal() && child.CanRequestFocus() {
			out = append(out, child)
		}
	}
	return out
}

// DebugLabel is the name of the node in debug output.
//
// Flutter: widgets/focus_manager.dart FocusNode.debugLabel
func (n *FocusNode) DebugLabel() string { return n.debugLabel }

// SetDebugLabel sets the name of the node; release builds ignore it.
//
// Flutter: widgets/focus_manager.dart FocusNode.debugLabel
func (n *FocusNode) SetDebugLabel(label string) {
	if foundation.DebugMode {
		n.debugLabel = label
	}
}

// Descendants are all the nodes below, depth first, each node after its own
// descendants. A scope's descendants include the ones in nested scopes.
//
// Flutter: widgets/focus_manager.dart FocusNode.descendants
func (n *FocusNode) Descendants() []*FocusNode {
	if !n.descendantsValid {
		var result []*FocusNode
		for _, child := range n.children {
			result = append(result, child.Descendants()...)
			result = append(result, child)
		}
		n.descendants, n.descendantsValid = result, true
	}
	return n.descendants
}

// TraversalDescendants are the descendants that focus traversal can visit.
//
// Flutter: widgets/focus_manager.dart FocusNode.traversalDescendants
func (n *FocusNode) TraversalDescendants() []*FocusNode {
	if n.scope != nil && !n.CanRequestFocus() {
		return nil
	}
	if !n.DescendantsAreFocusable() {
		return nil
	}
	var out []*FocusNode
	for _, node := range n.Descendants() {
		if !node.SkipTraversal() && node.CanRequestFocus() {
			out = append(out, node)
		}
	}
	return out
}

// Ancestors are the nodes above, nearest first.
//
// Flutter: widgets/focus_manager.dart FocusNode.ancestors
func (n *FocusNode) Ancestors() []*FocusNode {
	if !n.ancestorsValid {
		var result []*FocusNode
		for p := n.parent; p != nil; p = p.parent {
			result = append(result, p)
		}
		n.ancestors, n.ancestorsValid = result, true
	}
	return n.ancestors
}

// HasFocus is whether the node or a descendant has the primary focus.
//
// Flutter: widgets/focus_manager.dart FocusNode.hasFocus
func (n *FocusNode) HasFocus() bool {
	if n.HasPrimaryFocus() {
		return true
	}
	if n.manager == nil || n.manager.primaryFocus == nil {
		return false
	}
	return slices.Contains(n.manager.primaryFocus.Ancestors(), n)
}

// HasPrimaryFocus is whether the node is the primary focus.
//
// Flutter: widgets/focus_manager.dart FocusNode.hasPrimaryFocus
func (n *FocusNode) HasPrimaryFocus() bool { return n.manager != nil && n.manager.primaryFocus == n }

// HighlightMode is how focus is highlighted now (see FocusHighlightMode). A node
// that is not attached to a manager gives the default for the platform.
//
// Flutter: widgets/focus_manager.dart FocusNode.highlightMode
func (n *FocusNode) HighlightMode() FocusHighlightMode {
	if n.manager != nil {
		return n.manager.HighlightMode()
	}
	return defaultFocusHighlightModeForPlatform(nil)
}

// NearestScope is the node's own scope if it is one, else the nearest scope
// above it; nil for a node outside any scope.
//
// Flutter: widgets/focus_manager.dart FocusNode.nearestScope
func (n *FocusNode) NearestScope() *FocusScopeNode {
	if n.scope != nil {
		return n.scope
	}
	return n.EnclosingScope()
}

// EnclosingScope is the nearest scope above the node, or nil.
//
// Flutter: widgets/focus_manager.dart FocusNode.enclosingScope
func (n *FocusNode) EnclosingScope() *FocusScopeNode {
	if n.enclosingScopeCache == nil && n.parent != nil {
		n.enclosingScopeCache = n.parent.NearestScope()
	}
	return n.enclosingScopeCache
}

func (n *FocusNode) clearEnclosingScopeCache() {
	cached := n.enclosingScopeCache
	if cached == nil {
		return
	}
	n.enclosingScopeCache = nil
	for _, child := range n.children {
		if child.enclosingScopeCache == cached {
			child.clearEnclosingScopeCache()
		}
	}
}

// Size is the size of the node's render object, from its context.
//
// Flutter: widgets/focus_manager.dart FocusNode.size
func (n *FocusNode) Size() ui.Size { return n.Rect().Size() }

// Offset is the top left of the node's render object on the screen.
//
// Flutter: widgets/focus_manager.dart FocusNode.offset
func (n *FocusNode) Offset() ui.Offset { return n.Rect().TopLeft() }

// Rect is the bounds of the node's render object on the screen. It panics with
// the tag FocusNodeContextMissing when the node has no context yet.
//
// Flutter: widgets/focus_manager.dart FocusNode.rect
func (n *FocusNode) Rect() ui.Rect {
	if n.ctx == nil {
		panic(errors.New(tagFocusNodeContextMissing,
			"tried to get the bounds of a focus node that did not have its context set yet; the context needs to be set before evaluating traversal policies, which Attach does",
			"node", n.String()))
	}
	object := n.ctx.FindRenderObject()
	transform := object.GetTransformTo(nil)
	bounds := object.SemanticBounds()
	topLeft := transform.TransformPoint(bounds.TopLeft())
	bottomRight := transform.TransformPoint(bounds.BottomRight())
	return ui.NewRectFromLTRB(topLeft.Dx, topLeft.Dy, bottomRight.Dx, bottomRight.Dy)
}

// Unfocus removes the focus from the node or its descendants, if it has it. The
// disposition says where the focus goes; the default is the nearest scope.
//
// Flutter: widgets/focus_manager.dart FocusNode.unfocus
func (n *FocusNode) Unfocus(disposition ...UnfocusDisposition) {
	d := UnfocusDispositionScope
	if len(disposition) > 0 && disposition[0] != UnfocusDispositionUnspecified {
		d = disposition[0]
	}
	if !n.HasFocus() && (n.manager == nil || n.manager.markedForFocus != n) {
		return
	}
	scope := n.EnclosingScope()
	if scope == nil {
		// This is the root node, or a node that is not in the tree yet; neither
		// does anything when unfocused.
		return
	}
	switch d {
	case UnfocusDispositionScope:
		// If the scope cannot take the focus, its focused children are kept.
		if scope.CanRequestFocus() {
			// Clearing them prevents re-focusing the node that was just unfocused
			// if "next" is hit right away, and choosing the next-to-last focused
			// child if Unfocus is called more than once.
			scope.focusedChildren = nil
		}
		for !scope.CanRequestFocus() {
			next := scope.EnclosingScope()
			if next == nil && n.manager != nil {
				next = n.manager.rootScope
			}
			scope = next
		}
		scope.doRequestFocus(false)
	case UnfocusDispositionPreviouslyFocusedChild:
		// Selects the most recent focused child from the nearest focusable scope
		// and focuses it, or the scope itself if there is none.
		if scope.CanRequestFocus() {
			scope.removeFocusedChild(n)
		}
		for !scope.CanRequestFocus() {
			if outer := scope.EnclosingScope(); outer != nil {
				outer.removeFocusedChild(&scope.FocusNode)
			}
			next := scope.EnclosingScope()
			if next == nil && n.manager != nil {
				next = n.manager.rootScope
			}
			scope = next
		}
		scope.doRequestFocus(true)
	}
	if n.manager != nil {
		n.manager.focusDebug(func() string { return "Unfocused node:" }, func() []any {
			return []any{"primary focus was " + describeFocusNode(n), "next focus will be " + describeFocusNode(n.manager.markedForFocus)}
		})
	}
}

// ConsumeKeyboardToken returns true once after the node requested the focus,
// telling the caller that the keyboard may be shown for it.
//
// Flutter: widgets/focus_manager.dart FocusNode.consumeKeyboardToken
func (n *FocusNode) ConsumeKeyboardToken() bool {
	if !n.hasKeyboardToken {
		return false
	}
	n.hasKeyboardToken = false
	return true
}

// markNextFocus marks newFocus as the next focus: it becomes the primary focus
// when the manager next resolves the focus change.
func (n *FocusNode) markNextFocus(newFocus *FocusNode) {
	if n.manager != nil {
		n.manager.markNextFocus(n)
		return
	}
	// Without a manager the focus changes locally.
	newFocus.setAsFocusedChildForScope()
	newFocus.notify()
	if newFocus != n {
		n.notify()
	}
}

// removeChild removes the node and its children as children of this node.
func (n *FocusNode) removeChild(node *FocusNode, removeScopeFocus bool) {
	if removeScopeFocus {
		if nodeScope := node.EnclosingScope(); nodeScope != nil {
			nodeScope.removeFocusedChild(node)
			for _, descendant := range node.Descendants() {
				if descendant.EnclosingScope() == nodeScope {
					nodeScope.removeFocusedChild(descendant)
				}
			}
		}
	}
	node.parent = nil
	node.clearEnclosingScopeCache()
	n.children = slices.DeleteFunc(n.children, func(c *FocusNode) bool { return c == node })
	for _, ancestor := range n.Ancestors() {
		ancestor.descendantsValid = false
		ancestor.descendants = nil
	}
	n.descendantsValid = false
	n.descendants = nil
}

func (n *FocusNode) updateManager(manager *FocusManager) {
	n.manager = manager
	for _, descendant := range n.Descendants() {
		descendant.manager = manager
		descendant.ancestorsValid = false
		descendant.ancestors = nil
	}
}

// reparent makes the child a child of this node (FocusAttachment.Reparent does
// it).
func (n *FocusNode) reparent(child *FocusNode) {
	if child == n {
		panic(errors.New(tagFocusNodeReparentSelf, "tried to make a focus node into a parent of itself", "node", n.String()))
	}
	if child.parent == n {
		// The child is already a child of this parent.
		return
	}
	if n.manager != nil && child == &n.manager.rootScope.FocusNode {
		panic(errors.New(tagFocusNodeRootReparented, "reparenting the root focus node is not allowed"))
	}
	if slices.Contains(n.Ancestors(), child) {
		panic(errors.New(tagFocusNodeParentLoop, "the supplied child is already an ancestor of this node; loops are not allowed",
			"node", n.String(), "child", child.String()))
	}
	oldScope := child.EnclosingScope()
	hadFocus := child.HasFocus()
	if child.parent != nil {
		child.parent.removeChild(child, oldScope != n.NearestScope())
	}
	n.children = append(n.children, child)
	child.parent = n
	child.ancestorsValid = false
	child.ancestors = nil
	child.updateManager(n.manager)
	for _, ancestor := range child.Ancestors() {
		ancestor.descendantsValid = false
		ancestor.descendants = nil
	}
	if hadFocus && n.manager != nil && n.manager.primaryFocus != nil {
		// Updates the focus chain for the current focus without changing it.
		n.manager.primaryFocus.setAsFocusedChildForScope()
	}
	if oldScope != nil && child.ctx != nil && child.EnclosingScope() != oldScope {
		if policy := FocusTraversalGroupMaybeOf(child.ctx); policy != nil {
			policy.ChangedScope(child, oldScope)
		}
	}
	if child.requestFocusWhenReparented {
		child.doRequestFocus(true)
		child.requestFocusWhenReparented = false
	}
}

// AttachOpts are the optional parameters of FocusNode.Attach.
//
// Flutter: widgets/focus_manager.dart FocusNode.attach
type AttachOpts struct {
	// OnKeyEvent replaces the node's key event handler when it is not nil.
	OnKeyEvent FocusOnKeyEventCallback
}

// Attach gives the node its context and returns the attachment that the widget
// uses to put the node in the focus tree (Reparent) and take it out (Detach).
//
// Flutter: widgets/focus_manager.dart FocusNode.attach
func (n *FocusNode) Attach(bc BuildContext, opts ...AttachOpts) *FocusAttachment {
	o := firstOpts(opts)
	n.ctx = bc
	if o.OnKeyEvent != nil {
		n.onKeyEvent = o.OnKeyEvent
	}
	n.attachment = &FocusAttachment{node: n}
	return n.attachment
}

// Dispose detaches the node (unfocusing it) and releases its listeners.
//
// Flutter: widgets/focus_manager.dart FocusNode.dispose
func (n *FocusNode) Dispose() {
	// Detaching also unfocuses and cleans up the manager's data structures.
	if n.attachment != nil {
		n.attachment.Detach()
	} else if n.parent != nil {
		// Deviation: a node that was made a child of a scope without a widget
		// attaching it (a route's scope node that is set as the first focus and is
		// disposed before its FocusScope is built) leaves the tree, so that the
		// manager does not notify it after it is disposed.
		n.leaveTree()
	}
	n.ChangeNotifier.Dispose()
}

// notify tells the listeners, if the node is in the tree.
func (n *FocusNode) notify() {
	if n.parent == nil {
		// No longer part of the tree, so it does not notify.
		return
	}
	if n.HasPrimaryFocus() {
		n.setAsFocusedChildForScope()
	}
	n.NotifyListeners()
}

// RequestFocus requests the primary focus for the node, or for a descendant node
// that is given (a node that is not in the tree is attached to this one first).
// The focus changes when the manager next resolves the changes, usually a
// microtask later. A scope hands the focus to its focused child, down the tree.
//
// Flutter: widgets/focus_manager.dart FocusNode.requestFocus
func (n *FocusNode) RequestFocus(node ...*FocusNode) {
	if len(node) > 0 && node[0] != nil {
		target := node[0]
		if target.parent == nil {
			n.reparent(target)
		}
		if !slices.Contains(target.Ancestors(), n) {
			panic(errors.New(tagFocusRequestedOutsideScope,
				"focus was requested for a node that is not a descendant of the scope from which it was requested",
				"node", target.String(), "scope", n.String()))
		}
		target.doRequestFocus(true)
		return
	}
	n.doRequestFocus(true)
}

// doRequestFocus is RequestFocus' work; a scope does it differently.
func (n *FocusNode) doRequestFocus(findFirstFocus bool) {
	if n.scope != nil {
		n.scope.doRequestFocus(findFirstFocus)
		return
	}
	n.doRequestFocusNode(findFirstFocus)
}

func (n *FocusNode) doRequestFocusNode(bool) {
	if !n.CanRequestFocus() {
		if n.manager != nil {
			n.manager.focusDebug(func() string {
				return "Node NOT requesting focus because canRequestFocus is false: " + n.String()
			}, nil)
		}
		return
	}
	// If the node is not in the tree, the focus request waits until the next
	// time it is reparented, so that newly added widgets can be focused.
	if n.parent == nil {
		n.requestFocusWhenReparented = true
		return
	}
	n.setAsFocusedChildForScope()
	if n.HasPrimaryFocus() && (n.manager.markedForFocus == nil || n.manager.markedForFocus == n) {
		return
	}
	n.hasKeyboardToken = true
	n.manager.focusDebug(func() string { return "Node requesting focus: " + n.String() }, nil)
	n.markNextFocus(n)
}

// setAsFocusedChildForScope makes the node the focused child of each scope above
// it, and each scope the focused child of the one above.
func (n *FocusNode) setAsFocusedChildForScope() {
	scopeFocus := n
	for _, ancestor := range n.Ancestors() {
		scope := ancestor.scope
		if scope == nil {
			continue
		}
		if m := n.manager; m != nil {
			focus := scopeFocus
			m.focusDebug(func() string { return fmt.Sprintf("Setting %v as focused child for scope:", focus) },
				func() []any { return []any{scope} })
		}
		// Removes it anywhere in the focused child history, and adds it to the end
		// of the list, which is the top of the queue: the end of the list is the
		// currently focused child.
		scope.removeFocusedChild(scopeFocus)
		scope.focusedChildren = append(scope.focusedChildren, scopeFocus)
		scopeFocus = &scope.FocusNode
	}
}

// NextFocus moves the focus to the next node in the focus traversal order, in
// the nearest FocusTraversalGroup. It returns whether the focus moved.
//
// Flutter: widgets/focus_manager.dart FocusNode.nextFocus
func (n *FocusNode) NextFocus() bool { return FocusTraversalGroupOf(n.ctx).Next(n) }

// PreviousFocus moves the focus to the previous node in the focus traversal
// order. It returns whether the focus moved.
//
// Flutter: widgets/focus_manager.dart FocusNode.previousFocus
func (n *FocusNode) PreviousFocus() bool { return FocusTraversalGroupOf(n.ctx).Previous(n) }

// FocusInDirection moves the focus to the closest node in the direction. It
// returns whether the focus moved.
//
// Flutter: widgets/focus_manager.dart FocusNode.focusInDirection
func (n *FocusNode) FocusInDirection(direction TraversalDirection) bool {
	return FocusTraversalGroupOf(n.ctx).InDirection(n, direction)
}

// DebugFillProperties adds the focus state of the node.
//
// Flutter: widgets/focus_manager.dart FocusNode.debugFillProperties
func (n *FocusNode) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if n.ctx != nil {
		p.Add(foundation.NewDiagnosticsProperty[BuildContext]("context", n.ctx))
	}
	p.Add(foundation.NewFlagProperty("descendantsAreFocusable", foundation.Some(n.DescendantsAreFocusable()),
		foundation.FlagPropertyOpts{IfFalse: "DESCENDANTS UNFOCUSABLE", DefaultValue: foundation.Some(true)}))
	p.Add(foundation.NewFlagProperty("descendantsAreTraversable", foundation.Some(n.DescendantsAreTraversable()),
		foundation.FlagPropertyOpts{IfFalse: "DESCENDANTS UNTRAVERSABLE", DefaultValue: foundation.Some(true)}))
	p.Add(foundation.NewFlagProperty("canRequestFocus", foundation.Some(n.CanRequestFocus()),
		foundation.FlagPropertyOpts{IfFalse: "NOT FOCUSABLE", DefaultValue: foundation.Some(true)}))
	p.Add(foundation.NewFlagProperty("hasFocus", foundation.Some(n.HasFocus() && !n.HasPrimaryFocus()),
		foundation.FlagPropertyOpts{IfTrue: "IN FOCUS PATH", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("hasPrimaryFocus", foundation.Some(n.HasPrimaryFocus()),
		foundation.FlagPropertyOpts{IfTrue: "PRIMARY FOCUS", DefaultValue: foundation.Some(false)}))
	if n.scope != nil {
		n.scope.debugFillScopeProperties(p)
	}
}

// DebugDescribeChildren describes the children, named "Child 1", "Child 2", ….
//
// Flutter: widgets/focus_manager.dart FocusNode.debugDescribeChildren
func (n *FocusNode) DebugDescribeChildren() []foundation.DiagnosticsNode {
	out := make([]foundation.DiagnosticsNode, len(n.children))
	for i, child := range n.children {
		out[i] = foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{
			Name: foundation.Some(fmt.Sprintf("Child %d", i+1)),
		})
	}
	return out
}

// ToStringShort is the identity of the node with its label and focus state.
//
// Flutter: widgets/focus_manager.dart FocusNode.toStringShort
func (n *FocusNode) ToStringShort() string {
	hasDebugLabel := n.debugLabel != ""
	var extra strings.Builder
	if hasDebugLabel {
		extra.WriteString(n.debugLabel)
	}
	hasFocus := n.HasFocus()
	if hasFocus && hasDebugLabel {
		extra.WriteString(" ")
	}
	if hasFocus && !n.HasPrimaryFocus() {
		extra.WriteString("[IN FOCUS PATH]")
	}
	if n.HasPrimaryFocus() {
		extra.WriteString("[PRIMARY FOCUS]")
	}
	typeName := "FocusNode"
	switch {
	case n.scope != nil:
		typeName = "FocusScopeNode"
	case n.group != nil:
		typeName = "_FocusTraversalGroupNode"
	}
	id := typeName + "#" + foundation.ShortHash(n)
	if extra.Len() > 0 {
		return id + "(" + extra.String() + ")"
	}
	return id
}

// String describes the node and its properties on one line.
//
// Flutter: widgets/focus_manager.dart FocusNode.toString
func (n *FocusNode) String() string { return foundation.DiagnosticableString(n) }

// FocusScopeNodeOpts are the optional parameters of NewFocusScopeNode.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.new
type FocusScopeNodeOpts struct {
	FocusNodeOpts
	// TraversalEdgeBehavior is what happens when next or previous focus reaches
	// the edge of the scope; the default is TraversalEdgeBehaviorClosedLoop.
	TraversalEdgeBehavior TraversalEdgeBehavior
	// DirectionalTraversalEdgeBehavior is what happens when directional focus
	// reaches the edge of the scope; the default is TraversalEdgeBehaviorStop.
	DirectionalTraversalEdgeBehavior TraversalEdgeBehavior
}

// FocusScopeNode is a node that groups the focus nodes below it and remembers
// which of them had the focus last. Focus traversal stays inside the scope
// (see TraversalEdgeBehavior), and a Navigator route has one.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode
type FocusScopeNode struct {
	FocusNode
	traversalEdgeBehavior            TraversalEdgeBehavior
	directionalTraversalEdgeBehavior TraversalEdgeBehavior
	// focusedChildren is a stack of the children that were set as the focused
	// child, the most recent last.
	focusedChildren []*FocusNode
}

// NewFocusScopeNode creates a focus scope node.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.new
func NewFocusScopeNode(opts ...FocusScopeNodeOpts) *FocusScopeNode {
	o := firstOpts(opts)
	s := &FocusScopeNode{
		FocusNode:                        *NewFocusNode(o.FocusNodeOpts),
		traversalEdgeBehavior:            o.TraversalEdgeBehavior,
		directionalTraversalEdgeBehavior: o.DirectionalTraversalEdgeBehavior,
	}
	if s.traversalEdgeBehavior == TraversalEdgeBehaviorUnspecified {
		s.traversalEdgeBehavior = TraversalEdgeBehaviorClosedLoop
	}
	if s.directionalTraversalEdgeBehavior == TraversalEdgeBehaviorUnspecified {
		s.directionalTraversalEdgeBehavior = TraversalEdgeBehaviorStop
	}
	s.descendantsAreFocusable = true
	s.scope = s
	return s
}

// Node is the scope as a FocusNode, for the APIs that take any node.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode
func (s *FocusScopeNode) Node() *FocusNode { return &s.FocusNode }

// TraversalEdgeBehavior is what next and previous focus do at the edge of the
// scope.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.traversalEdgeBehavior
func (s *FocusScopeNode) TraversalEdgeBehavior() TraversalEdgeBehavior {
	return s.traversalEdgeBehavior
}

// SetTraversalEdgeBehavior sets what next and previous focus do at the edge.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.traversalEdgeBehavior
func (s *FocusScopeNode) SetTraversalEdgeBehavior(b TraversalEdgeBehavior) {
	s.traversalEdgeBehavior = b
}

// DirectionalTraversalEdgeBehavior is what directional focus does at the edge of
// the scope.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.directionalTraversalEdgeBehavior
func (s *FocusScopeNode) DirectionalTraversalEdgeBehavior() TraversalEdgeBehavior {
	return s.directionalTraversalEdgeBehavior
}

// SetDirectionalTraversalEdgeBehavior sets what directional focus does at the
// edge.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.directionalTraversalEdgeBehavior
func (s *FocusScopeNode) SetDirectionalTraversalEdgeBehavior(b TraversalEdgeBehavior) {
	s.directionalTraversalEdgeBehavior = b
}

// IsFirstFocus is whether the scope is the focused child of its own enclosing
// scope.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.isFirstFocus
func (s *FocusScopeNode) IsFirstFocus() bool {
	return s.EnclosingScope().FocusedChild() == &s.FocusNode
}

// FocusedChild is the child that has the focus, or would have it when the scope
// gets it; nil if there is none.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.focusedChild
func (s *FocusScopeNode) FocusedChild() *FocusNode {
	if s == nil || len(s.focusedChildren) == 0 {
		return nil
	}
	return s.focusedChildren[len(s.focusedChildren)-1]
}

func (s *FocusScopeNode) removeFocusedChild(node *FocusNode) {
	s.focusedChildren = slices.DeleteFunc(s.focusedChildren, func(c *FocusNode) bool { return c == node })
}

// SetFirstFocus makes the scope the one that takes the focus first when this
// scope gets it. A scope that is not in the tree is attached below this one.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.setFirstFocus
func (s *FocusScopeNode) SetFirstFocus(scope *FocusScopeNode) {
	if scope == s {
		panic(errors.New(tagFocusScopeSelfReference, "unexpected self-reference in SetFirstFocus", "scope", s.String()))
	}
	s.manager.focusDebugOpt(func() string { return "Setting scope as first focus in " + s.String() + " to node:" },
		func() []any { return []any{scope} })
	if scope.parent == nil {
		s.reparent(&scope.FocusNode)
	}
	if !slices.Contains(scope.Ancestors(), &s.FocusNode) {
		panic(errors.New(tagFocusScopeNotChild, "the scope must be a child of this scope to set it as first focus",
			"scope", scope.String(), "parent", s.String()))
	}
	if s.HasFocus() {
		scope.doRequestFocus(true)
	} else {
		scope.setAsFocusedChildForScope()
	}
}

// Autofocus requests the focus for the node when the scope has no focused child,
// when the manager next resolves focus changes (a node that is not in the tree is
// attached below the scope first).
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.autofocus
func (s *FocusScopeNode) Autofocus(node *FocusNode) {
	// The node is attached to the tree first, so that applying the request does
	// not add a detached node back.
	if node.parent == nil {
		s.reparent(node)
	}
	if s.manager != nil {
		s.manager.focusDebug(func() string { return fmt.Sprintf("Autofocus scheduled for %v: scope %v", node, s) }, nil)
		s.manager.pendingAutofocuses = append(s.manager.pendingAutofocuses, autofocus{scope: s, autofocusNode: node})
		s.manager.markNeedsUpdate()
	}
}

// RequestScopeFocus requests the focus for the scope itself, not for its focused
// child.
//
// Flutter: widgets/focus_manager.dart FocusScopeNode.requestScopeFocus
func (s *FocusScopeNode) RequestScopeFocus() { s.doRequestFocus(false) }

func (s *FocusScopeNode) doRequestFocus(findFirstFocus bool) {
	// A previously focused child can have become unfocusable: they are cleaned
	// out of the list.
	for len(s.focusedChildren) > 0 {
		last := s.focusedChildren[len(s.focusedChildren)-1]
		if last.CanRequestFocus() && last.EnclosingScope() != nil {
			break
		}
		s.focusedChildren = s.focusedChildren[:len(s.focusedChildren)-1]
	}
	focusedChild := s.FocusedChild()
	// If findFirstFocus is false, the request is to make this scope the focus
	// instead of looking for the first focus of this scope and its descendants.
	if !findFirstFocus || focusedChild == nil {
		if s.CanRequestFocus() {
			s.setAsFocusedChildForScope()
			s.markNextFocus(&s.FocusNode)
		}
		return
	}
	focusedChild.doRequestFocus(true)
}

func (s *FocusScopeNode) debugFillScopeProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if len(s.focusedChildren) == 0 {
		return
	}
	childList := make([]string, 0, len(s.focusedChildren))
	for i := len(s.focusedChildren) - 1; i >= 0; i-- {
		childList = append(childList, s.focusedChildren[i].ToStringShort())
	}
	p.Add(foundation.NewIterableProperty("focusedChildren", childList))
	p.Add(foundation.NewDiagnosticsProperty("traversalEdgeBehavior", s.traversalEdgeBehavior,
		foundation.DiagnosticsPropertyOpts{DefaultValue: TraversalEdgeBehaviorClosedLoop}))
}

// FocusHighlightMode is how focus is shown: whether a focused widget should look
// focused when it has the focus after a pointer or a keyboard interaction.
//
// Flutter: widgets/focus_manager.dart FocusHighlightMode
type FocusHighlightMode int

// The values of FocusHighlightMode follow Flutter's declaration order.
const (
	// FocusHighlightModeUnspecified is the zero value: no mode is known.
	//
	// Flutter: no counterpart
	FocusHighlightModeUnspecified FocusHighlightMode = iota
	// FocusHighlightModeTouch is for touch interaction: widgets that take the
	// focus do not show it, except for text fields.
	//
	// Flutter: widgets/focus_manager.dart FocusHighlightMode.touch
	FocusHighlightModeTouch
	// FocusHighlightModeTraditional is for mouse and keyboard interaction:
	// the focus is shown.
	//
	// Flutter: widgets/focus_manager.dart FocusHighlightMode.traditional
	FocusHighlightModeTraditional
)

// String is the name of the mode, like Dart's enum toString.
//
// Flutter: widgets/focus_manager.dart FocusHighlightMode
func (m FocusHighlightMode) String() string {
	switch m {
	case FocusHighlightModeTouch:
		return "FocusHighlightMode.touch"
	case FocusHighlightModeTraditional:
		return "FocusHighlightMode.traditional"
	}
	return "FocusHighlightMode.unspecified"
}

// FocusHighlightStrategy is how the FocusManager chooses the highlight mode.
//
// Flutter: widgets/focus_manager.dart FocusHighlightStrategy
type FocusHighlightStrategy int

// The values of FocusHighlightStrategy follow Flutter's declaration order.
const (
	// FocusHighlightStrategyUnspecified is the zero value: automatic.
	//
	// Flutter: no counterpart
	FocusHighlightStrategyUnspecified FocusHighlightStrategy = iota
	// FocusHighlightStrategyAutomatic follows the last interaction: touch after a
	// touch, traditional after a key.
	//
	// Flutter: widgets/focus_manager.dart FocusHighlightStrategy.automatic
	FocusHighlightStrategyAutomatic
	// FocusHighlightStrategyAlwaysTouch is always touch.
	//
	// Flutter: widgets/focus_manager.dart FocusHighlightStrategy.alwaysTouch
	FocusHighlightStrategyAlwaysTouch
	// FocusHighlightStrategyAlwaysTraditional is always traditional.
	//
	// Flutter: widgets/focus_manager.dart FocusHighlightStrategy.alwaysTraditional
	FocusHighlightStrategyAlwaysTraditional
)

// String is the name of the strategy, like Dart's enum toString.
//
// Flutter: widgets/focus_manager.dart FocusHighlightStrategy
func (s FocusHighlightStrategy) String() string {
	switch s {
	case FocusHighlightStrategyAlwaysTouch:
		return "FocusHighlightStrategy.alwaysTouch"
	case FocusHighlightStrategyAlwaysTraditional:
		return "FocusHighlightStrategy.alwaysTraditional"
	case FocusHighlightStrategyAutomatic:
		return "FocusHighlightStrategy.automatic"
	}
	return "FocusHighlightStrategy.unspecified"
}

// FocusManager keeps the focus tree of an app and the primary focus in it. The
// binding makes one (WidgetsBinding.FocusManager); a build owner reaches it with
// BuildOwner.FocusManager.
//
// Deviation: Flutter reads FocusManager.instance; Glutter has no singletons, so
// the manager is reached through the binding or the build owner (SPEC §12.10).
//
// Flutter: widgets/focus_manager.dart FocusManager
type FocusManager struct {
	foundation.ChangeNotifier

	binding   *WidgetsBinding
	rootScope *FocusScopeNode

	primaryFocus *FocusNode
	// dirtyNodes need to notify their listeners at the next update, in the order
	// they were marked.
	dirtyNodes []*FocusNode

	lifecycleObserver *focusLifecycleObserver
	// suspendedNode was focused before the app lifecycle changed; it is restored as
	// the primary focus when the app resumes.
	suspendedNode *FocusNode
	// markedForFocus has requested the primary focus but has not been given it.
	markedForFocus     *FocusNode
	pendingAutofocuses []autofocus
	// haveScheduledUpdate is whether an update is pending.
	haveScheduledUpdate bool

	highlight *highlightModeManager
}

// NewFocusManager creates a manager with the root scope, and listens to the
// lifecycle of the app of the binding on the platforms where focus is suspended
// when the app is inactive. The binding makes its own; this is for a test of the
// manager alone.
//
// Flutter: widgets/focus_manager.dart FocusManager.new
func NewFocusManager(binding *WidgetsBinding) *FocusManager {
	m := &FocusManager{binding: binding}
	m.highlight = newHighlightModeManager(m)
	m.rootScope = NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Root Focus Scope"}})
	m.rootScope.manager = m
	if m.respondToLifecycleChange() && binding != nil {
		m.lifecycleObserver = &focusLifecycleObserver{m: m}
		binding.AddObserver(m.lifecycleObserver)
	}
	return m
}

func (m *FocusManager) respondToLifecycleChange() bool {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
		return false
	}
	return true
}

// focusLifecycleObserver passes the lifecycle changes to the manager.
//
// Flutter: widgets/focus_manager.dart _AppLifecycleListener
type focusLifecycleObserver struct {
	WidgetsBindingObserverBase
	m *FocusManager
}

func (o *focusLifecycleObserver) DidChangeAppLifecycleState(state ui.AppLifecycleState) {
	o.m.appLifecycleChange(state)
}

// RegisterGlobalHandlers listens to the key events and the pointer events, so
// that the focus system handles keys and follows the interaction for the
// highlight mode. The binding calls it.
//
// Flutter: widgets/focus_manager.dart FocusManager.registerGlobalHandlers
func (m *FocusManager) RegisterGlobalHandlers() { m.highlight.registerGlobalHandlers() }

// Dispose stops listening and disposes the root scope.
//
// Flutter: widgets/focus_manager.dart FocusManager.dispose
func (m *FocusManager) Dispose() {
	if m.lifecycleObserver != nil && m.binding != nil {
		m.binding.RemoveObserver(m.lifecycleObserver)
		m.lifecycleObserver = nil
	}
	m.highlight.dispose()
	m.rootScope.Dispose()
	m.ChangeNotifier.Dispose()
}

// RootScope is the scope at the root of the focus tree.
//
// Flutter: widgets/focus_manager.dart FocusManager.rootScope
func (m *FocusManager) RootScope() *FocusScopeNode { return m.rootScope }

// PrimaryFocus is the node that has the primary focus, or nil.
//
// Flutter: widgets/focus_manager.dart FocusManager.primaryFocus
func (m *FocusManager) PrimaryFocus() *FocusNode { return m.primaryFocus }

// HighlightStrategy is how the highlight mode is chosen.
//
// Flutter: widgets/focus_manager.dart FocusManager.highlightStrategy
func (m *FocusManager) HighlightStrategy() FocusHighlightStrategy { return m.highlight.strategy }

// SetHighlightStrategy sets how the highlight mode is chosen.
//
// Flutter: widgets/focus_manager.dart FocusManager.highlightStrategy
func (m *FocusManager) SetHighlightStrategy(s FocusHighlightStrategy) {
	if s == FocusHighlightStrategyUnspecified {
		s = FocusHighlightStrategyAutomatic
	}
	if m.highlight.strategy == s {
		return
	}
	m.highlight.strategy = s
	m.highlight.updateMode()
}

// HighlightMode is how focus is highlighted now.
//
// Flutter: widgets/focus_manager.dart FocusManager.highlightMode
func (m *FocusManager) HighlightMode() FocusHighlightMode { return m.highlight.currentMode() }

// AddHighlightModeListener adds a function that is called when the highlight
// mode changes.
//
// Flutter: widgets/focus_manager.dart FocusManager.addHighlightModeListener
func (m *FocusManager) AddHighlightModeListener(listener func(FocusHighlightMode)) FocusHandlerHandle {
	return m.highlight.addListener(listener)
}

// RemoveHighlightModeListener removes a listener.
//
// Flutter: widgets/focus_manager.dart FocusManager.removeHighlightModeListener
func (m *FocusManager) RemoveHighlightModeListener(handle FocusHandlerHandle) {
	m.highlight.listeners = removeHandler(m.highlight.listeners, handle)
}

// AddEarlyKeyEventHandler adds a handler that sees key events before the focus
// nodes do. If it handles the event, no node sees it. It sees every key event,
// also when no node has the primary focus (Flutter ignores the key then).
//
// Flutter: widgets/focus_manager.dart FocusManager.addEarlyKeyEventHandler
func (m *FocusManager) AddEarlyKeyEventHandler(handler OnKeyEventCallback) FocusHandlerHandle {
	return m.highlight.addKeyHandler(&m.highlight.earlyHandlers, handler)
}

// RemoveEarlyKeyEventHandler removes an early handler.
//
// Flutter: widgets/focus_manager.dart FocusManager.removeEarlyKeyEventHandler
func (m *FocusManager) RemoveEarlyKeyEventHandler(handle FocusHandlerHandle) {
	m.highlight.earlyHandlers = removeHandler(m.highlight.earlyHandlers, handle)
}

// AddLateKeyEventHandler adds a handler that sees the key events that no focus
// node handled. It also sees the key events that arrive when no node has the
// primary focus, unless an early handler stopped them (Flutter ignores the key
// then).
//
// Flutter: widgets/focus_manager.dart FocusManager.addLateKeyEventHandler
func (m *FocusManager) AddLateKeyEventHandler(handler OnKeyEventCallback) FocusHandlerHandle {
	return m.highlight.addKeyHandler(&m.highlight.lateHandlers, handler)
}

// RemoveLateKeyEventHandler removes a late handler.
//
// Flutter: widgets/focus_manager.dart FocusManager.removeLateKeyEventHandler
func (m *FocusManager) RemoveLateKeyEventHandler(handle FocusHandlerHandle) {
	m.highlight.lateHandlers = removeHandler(m.highlight.lateHandlers, handle)
}

func (m *FocusManager) appLifecycleChange(state ui.AppLifecycleState) {
	if state == ui.AppLifecycleStateResumed {
		if m.primaryFocus != &m.rootScope.FocusNode {
			m.focusDebug(func() string {
				return "focus changed while app was paused, ignoring " + describeFocusNode(m.suspendedNode)
			}, nil)
			m.suspendedNode = nil
		} else if m.suspendedNode != nil {
			// The focus that was suspended when the app went inactive is restored
			// only if nothing else has requested the focus in the meantime: for
			// example, activating a different window makes its view request it.
			if m.markedForFocus == nil {
				m.focusDebug(func() string { return "requesting focus for " + describeFocusNode(m.suspendedNode) }, nil)
				m.suspendedNode.RequestFocus()
				m.suspendedNode = nil
			} else {
				m.suspendedNode = nil
			}
		}
	} else if m.primaryFocus != &m.rootScope.FocusNode {
		m.focusDebug(func() string { return "suspending " + describeFocusNode(m.primaryFocus) }, nil)
		m.markedForFocus = &m.rootScope.FocusNode
		m.suspendedNode = m.primaryFocus
		m.ApplyFocusChangesIfNeeded()
	}
}

func (m *FocusManager) markDetached(node *FocusNode) {
	// The node has been removed from the tree, so it no longer needs to be
	// notified of changes.
	m.focusDebug(func() string { return "Node was detached: " + node.String() }, nil)
	if m.primaryFocus == node {
		m.primaryFocus = nil
	}
	if m.suspendedNode == node {
		m.suspendedNode = nil
	}
	m.dirtyNodes = slices.DeleteFunc(m.dirtyNodes, func(n *FocusNode) bool { return n == node })
}

func (m *FocusManager) markPropertiesChanged(node *FocusNode) {
	m.markNeedsUpdate()
	m.focusDebug(func() string { return "Properties changed for node " + node.String() + "." }, nil)
	m.addDirty(node)
}

func (m *FocusManager) addDirty(node *FocusNode) {
	if !slices.Contains(m.dirtyNodes, node) {
		m.dirtyNodes = append(m.dirtyNodes, node)
	}
}

func (m *FocusManager) markNextFocus(node *FocusNode) {
	if m.primaryFocus == node {
		// The caller asked for the current focus to be the next focus: pretend
		// that did not happen.
		m.markedForFocus = nil
	} else {
		m.markedForFocus = node
		m.markNeedsUpdate()
	}
}

// markNeedsUpdate schedules an update of the focus state in a microtask.
func (m *FocusManager) markNeedsUpdate() {
	m.focusDebug(func() string {
		return "Scheduling update, current focus is " + describeFocusNode(m.primaryFocus) + ", next focus will be " + describeFocusNode(m.markedForFocus)
	}, nil)
	if m.haveScheduledUpdate {
		return
	}
	m.haveScheduledUpdate = true
	if m.binding != nil {
		m.binding.Post(m.ApplyFocusChangesIfNeeded)
	}
}

// ApplyFocusChangesIfNeeded resolves the requests for the focus that were made
// since the last time: the autofocus requests, the node marked for focus, and
// the nodes whose state changed are notified. It runs in a microtask after a
// change; it may be called to resolve the changes at once, but not during the
// build phase.
//
// Flutter: widgets/focus_manager.dart FocusManager.applyFocusChangesIfNeeded
func (m *FocusManager) ApplyFocusChangesIfNeeded() {
	if m.binding != nil && foundation.DebugMode && m.binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
		panic(errors.New(tagFocusChangesDuringBuild, "ApplyFocusChangesIfNeeded() should not be called during the build phase"))
	}
	m.haveScheduledUpdate = false
	previousFocus := m.primaryFocus
	for _, a := range slices.Clone(m.pendingAutofocuses) {
		a.applyIfValid(m)
	}
	m.pendingAutofocuses = nil
	if m.primaryFocus == nil && m.markedForFocus == nil {
		// If there is no current focus, and nobody has asked to focus yet, then
		// revert to the root scope.
		m.markedForFocus = &m.rootScope.FocusNode
	}
	m.focusDebug(func() string {
		return "Refreshing focus state. Next focus will be " + describeFocusNode(m.markedForFocus)
	}, nil)
	// A node has requested to be the next focus, and is not already the primary
	// focus.
	if m.markedForFocus != nil && m.markedForFocus != m.primaryFocus {
		var previousPath []*FocusNode
		if previousFocus != nil {
			previousPath = previousFocus.Ancestors()
		}
		nextPath := m.markedForFocus.Ancestors()
		// Notifies the nodes that are newly focused and the ones that no longer are.
		for _, n := range nextPath {
			if !slices.Contains(previousPath, n) {
				m.addDirty(n)
			}
		}
		for _, n := range previousPath {
			if !slices.Contains(nextPath, n) {
				m.addDirty(n)
			}
		}
		m.primaryFocus = m.markedForFocus
		m.markedForFocus = nil
	}
	if previousFocus != m.primaryFocus {
		m.focusDebug(func() string {
			return "Updating focus from " + describeFocusNode(previousFocus) + " to " + describeFocusNode(m.primaryFocus)
		}, nil)
		if previousFocus != nil {
			m.addDirty(previousFocus)
		}
		if m.primaryFocus != nil {
			m.addDirty(m.primaryFocus)
		}
	}
	dirty := m.dirtyNodes
	m.dirtyNodes = nil
	for _, n := range dirty {
		n.notify()
	}
	m.focusDebug(func() string { return fmt.Sprintf("Notified %d dirty nodes:", len(dirty)) }, func() []any {
		out := make([]any, len(dirty))
		for i, n := range dirty {
			out[i] = n
		}
		return out
	})
	if previousFocus != m.primaryFocus {
		m.NotifyListeners()
	}
	if m.binding != nil && foundation.DebugMode && m.binding.debug.FocusChanges {
		m.binding.debug.DebugPrint(m.DebugDescribe())
	}
}

// DebugDescribe describes the focus tree, as DebugDumpFocusTree prints it.
//
// Flutter: widgets/focus_manager.dart debugDescribeFocusTree
func (m *FocusManager) DebugDescribe() string {
	if !foundation.DebugMode {
		return ""
	}
	return foundation.ToStringDeep(m)
}

// DebugFillProperties adds the primary focus and the next focus.
//
// Flutter: widgets/focus_manager.dart FocusManager.debugFillProperties
func (m *FocusManager) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("haveScheduledUpdate", foundation.Some(m.haveScheduledUpdate),
		foundation.FlagPropertyOpts{IfTrue: "UPDATE SCHEDULED"}))
	if m.primaryFocus != nil {
		p.Add(foundation.NewDiagnosticsProperty("primaryFocus", m.primaryFocus))
	}
	if m.markedForFocus != nil {
		p.Add(foundation.NewDiagnosticsProperty("nextFocus", m.markedForFocus))
	}
	if m.primaryFocus != nil && m.primaryFocus.ctx != nil {
		if e, ok := m.primaryFocus.ctx.(Element); ok {
			p.Add(foundation.NewDiagnosticsProperty("primaryFocusCreator", e.AsElement().DebugGetCreatorChain(20)))
		}
	}
}

// DebugDescribeChildren is the root scope, named "rootScope".
//
// Flutter: widgets/focus_manager.dart FocusManager.debugDescribeChildren
func (m *FocusManager) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return []foundation.DiagnosticsNode{
		foundation.ToDiagnosticsNode(&m.rootScope.FocusNode, foundation.DiagnosticsNodeOpts{Name: foundation.Some("rootScope")}),
	}
}

// focusDebug prints a message about focus changes when the debug flag asks for
// it; messageFunc and detailsFunc are called only then.
//
// Flutter: widgets/focus_manager.dart _focusDebug
func (m *FocusManager) focusDebug(messageFunc func() string, detailsFunc func() []any) {
	if m == nil || !foundation.DebugMode || m.binding == nil || !m.binding.debug.FocusChanges {
		return
	}
	m.binding.debug.DebugPrint("FOCUS: " + messageFunc())
	if detailsFunc != nil {
		for _, detail := range detailsFunc() {
			m.binding.debug.DebugPrint(fmt.Sprintf("    %v", detail))
		}
	}
}

// focusDebugOpt is focusDebug for a manager that can be nil.
func (m *FocusManager) focusDebugOpt(messageFunc func() string, detailsFunc func() []any) {
	m.focusDebug(messageFunc, detailsFunc)
}

// removeHandler removes the entry with the handle.
func removeHandler[T interface{ handlerID() uint64 }](list []T, handle FocusHandlerHandle) []T {
	return slices.DeleteFunc(list, func(e T) bool { return e.handlerID() == handle.id })
}

type keyHandlerRegistration struct {
	id uint64
	fn OnKeyEventCallback
}

func (r keyHandlerRegistration) handlerID() uint64 { return r.id }

type highlightListenerRegistration struct {
	id uint64
	fn func(FocusHighlightMode)
}

func (r highlightListenerRegistration) handlerID() uint64 { return r.id }

// trilean is a bool that can be unknown.
type trilean int

const (
	triUnknown trilean = iota
	triFalse
	triTrue
)

// highlightModeManager detects and manages the highlight mode transitions. The
// FocusManager owns one.
//
// Flutter: widgets/focus_manager.dart _HighlightModeManager
type highlightModeManager struct {
	m *FocusManager
	// lastInteractionRequiresTraditionalHighlights is unknown until an
	// interaction has occurred, and the default for the platform applies.
	lastInteractionRequiresTraditionalHighlights trilean
	mode                                         FocusHighlightMode
	strategy                                     FocusHighlightStrategy
	earlyHandlers, lateHandlers                  []keyHandlerRegistration
	listeners                                    []highlightListenerRegistration
	nextID                                       uint64
	keyHandlerRegistered                         bool
	pointerRoute                                 gestures.PointerRouteHandle
}

func newHighlightModeManager(m *FocusManager) *highlightModeManager {
	return &highlightModeManager{m: m, strategy: FocusHighlightStrategyAutomatic}
}

func (h *highlightModeManager) currentMode() FocusHighlightMode {
	if h.mode != FocusHighlightModeUnspecified {
		return h.mode
	}
	return defaultFocusHighlightModeForPlatform(h.m.binding)
}

func (h *highlightModeManager) addKeyHandler(list *[]keyHandlerRegistration, fn OnKeyEventCallback) FocusHandlerHandle {
	h.nextID++
	*list = append(*list, keyHandlerRegistration{h.nextID, fn})
	return FocusHandlerHandle{h.nextID}
}

func (h *highlightModeManager) addListener(fn func(FocusHighlightMode)) FocusHandlerHandle {
	h.nextID++
	h.listeners = append(h.listeners, highlightListenerRegistration{h.nextID, fn})
	return FocusHandlerHandle{h.nextID}
}

func (h *highlightModeManager) registerGlobalHandlers() {
	b := h.m.binding
	if b == nil {
		return
	}
	b.Keyboard().SetKeyMessageHandler(h.handleKeyMessage)
	h.keyHandlerRegistered = true
	h.pointerRoute = b.Gestures().PointerRouter.AddGlobalRoute(h.handlePointerEvent, nil)
}

func (h *highlightModeManager) dispose() {
	b := h.m.binding
	if b != nil && h.keyHandlerRegistered {
		b.Gestures().PointerRouter.RemoveGlobalRoute(h.pointerRoute)
		b.Keyboard().SetKeyMessageHandler(nil)
		h.keyHandlerRegistered = false
	}
	h.listeners = nil
}

func (h *highlightModeManager) notifyListeners() {
	if len(h.listeners) == 0 {
		return
	}
	local := slices.Clone(h.listeners)
	for _, l := range local {
		if !slices.ContainsFunc(h.listeners, func(e highlightListenerRegistration) bool { return e.id == l.id }) {
			continue
		}
		var report foundation.ErrorReporter
		if h.m.binding != nil {
			report = h.m.binding.report
		}
		foundation.Guard(report, foundation.CallbackPanicked, "while dispatching notifications for the highlight mode manager",
			func() []any { return []any{"library", "widgets library"} },
			func() { l.fn(h.currentMode()) })
	}
}

func (h *highlightModeManager) handlePointerEvent(event gestures.PointerEvent) {
	switch event.Kind() {
	case ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus:
		if h.lastInteractionRequiresTraditionalHighlights != triTrue {
			h.lastInteractionRequiresTraditionalHighlights = triTrue
			h.updateMode()
		}
	}
}

// handleKeyMessage hands the key event to the early handlers, the focus nodes
// from the primary focus up to the root, and the late handlers, in that order.
// It returns whether the event was handled.
//
// Deviation: when nothing has the primary focus, Flutter ignores the key. Here
// the early handlers and then the late handlers still see it (the walk up the
// focus chain has no node to start from), so that an app can claim a key at any
// time, also in the few frames between a focused widget leaving the tree and
// the focus moving on (SPEC §12.10). An early handler that returns
// SkipRemainingHandlers ends the event unhandled, so the late handlers do not
// see it then.
func (h *highlightModeManager) handleKeyMessage(event services.KeyEvent) bool {
	// The highlight mode is updated first, since things responding to the keys
	// might look at it, and it should be accurate.
	if h.lastInteractionRequiresTraditionalHighlights != triFalse {
		h.lastInteractionRequiresTraditionalHighlights = triFalse
		h.updateMode()
	}
	m := h.m
	m.focusDebug(func() string { return fmt.Sprintf("Received key event %v", event) }, nil)
	if m.primaryFocus == nil {
		m.focusDebug(func() string {
			return fmt.Sprintf("No primary focus for key event, only the key event handlers see it: %v", event)
		}, nil)
	}
	handled := false
	stopped := false
	// If any of the early handlers handles the key, the event is done.
	if len(h.earlyHandlers) > 0 {
		var results []KeyEventResult
		for _, cb := range slices.Clone(h.earlyHandlers) {
			results = append(results, cb.fn(event))
		}
		switch CombineKeyEventResults(results) {
		case KeyEventResultHandled:
			m.focusDebug(func() string { return fmt.Sprintf("Key event %v handled by early key event callback.", event) }, nil)
			handled = true
		case KeyEventResultSkipRemainingHandlers:
			m.focusDebug(func() string {
				return fmt.Sprintf("Key event %v propagation stopped by early key event callback.", event)
			}, nil)
			stopped = true
		}
	}
	if handled {
		return true
	}
	if m.primaryFocus == nil && stopped {
		return false
	}
	// Walks the current focus from the leaf to the root, calling each node's
	// OnKeyEvent on the way up; if one handled the event or wants to stop the
	// propagation, it stops.
	var chain []*FocusNode
	if m.primaryFocus != nil {
		chain = append([]*FocusNode{m.primaryFocus}, m.primaryFocus.Ancestors()...)
	}
	for _, node := range chain {
		var results []KeyEventResult
		if node.onKeyEvent != nil {
			results = append(results, node.onKeyEvent(node, event))
		}
		result := CombineKeyEventResults(results)
		if result == KeyEventResultIgnored {
			continue
		}
		switch result {
		case KeyEventResultHandled:
			m.focusDebug(func() string { return fmt.Sprintf("Node %v handled key event %v.", node, event) }, nil)
			handled = true
		case KeyEventResultSkipRemainingHandlers:
			m.focusDebug(func() string { return fmt.Sprintf("Node %v stopped key event propagation: %v.", node, event) }, nil)
			handled = false
		}
		// Only KeyEventResultIgnored continues the loop: all the others stop the
		// propagation.
		break
	}
	// The late handlers may want to handle the event.
	if !handled && len(h.lateHandlers) > 0 {
		var results []KeyEventResult
		for _, cb := range slices.Clone(h.lateHandlers) {
			results = append(results, cb.fn(event))
		}
		switch CombineKeyEventResults(results) {
		case KeyEventResultHandled:
			m.focusDebug(func() string { return fmt.Sprintf("Key event %v handled by late key event callback.", event) }, nil)
			handled = true
		case KeyEventResultSkipRemainingHandlers:
			m.focusDebug(func() string {
				return fmt.Sprintf("Key event %v propagation stopped by late key event callback.", event)
			}, nil)
		}
	}
	if !handled {
		m.focusDebug(func() string { return fmt.Sprintf("Key event not handled by focus system: %v.", event) }, nil)
	}
	return handled
}

// updateMode is called whenever the state that the highlight mode depends on
// changes.
func (h *highlightModeManager) updateMode() {
	var newMode FocusHighlightMode
	switch h.strategy {
	case FocusHighlightStrategyAlwaysTouch:
		newMode = FocusHighlightModeTouch
	case FocusHighlightStrategyAlwaysTraditional:
		newMode = FocusHighlightModeTraditional
	default:
		if h.lastInteractionRequiresTraditionalHighlights == triUnknown {
			// Without information about the last interaction the default for the
			// platform applies, which is found if mode is not set.
			return
		}
		if h.lastInteractionRequiresTraditionalHighlights == triTrue {
			newMode = FocusHighlightModeTouch
		} else {
			newMode = FocusHighlightModeTraditional
		}
	}
	// The mode is compared through the getter, since the field can be unset, and
	// clients look at the getter.
	oldMode := h.currentMode()
	h.mode = newMode
	if h.currentMode() != oldMode {
		h.notifyListeners()
	}
}

// defaultFocusHighlightModeForPlatform assumes that on a mobile platform with no
// mouse connected the first interaction is touch, and traditional mouse and
// keyboard everywhere else. It only affects the initial value.
func defaultFocusHighlightModeForPlatform(b *WidgetsBinding) FocusHighlightMode {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
		if b != nil && b.Renderer().MouseTracker().MouseIsConnected() {
			return FocusHighlightModeTraditional
		}
		return FocusHighlightModeTouch
	}
	return FocusHighlightModeTraditional
}

// FocusManager is the focus manager of the binding.
//
// Flutter: widgets/binding.dart WidgetsBinding.focusManager
func (b *WidgetsBinding) FocusManager() *FocusManager { return b.focusManager }

// FocusManager is the focus manager of the binding the owner belongs to, or nil.
//
// Flutter: widgets/framework.dart BuildOwner.focusManager
func (o *BuildOwner) FocusManager() *FocusManager {
	if o == nil || o.binding == nil {
		return nil
	}
	return o.binding.focusManager
}

// PrimaryFocus is the node that has the primary focus in the app of the context,
// or nil.
//
// Flutter: widgets/focus_manager.dart primaryFocus
func PrimaryFocus(bc BuildContext) *FocusNode {
	if m := bc.Owner().FocusManager(); m != nil {
		return m.PrimaryFocus()
	}
	return nil
}

// DebugDescribeFocusTree describes the focus tree of the app of the context.
//
// Flutter: widgets/focus_manager.dart debugDescribeFocusTree
func DebugDescribeFocusTree(bc BuildContext) string {
	if m := bc.Owner().FocusManager(); m != nil {
		return m.DebugDescribe()
	}
	return ""
}

// DebugDumpFocusTree prints the focus tree of the app of the context.
//
// Flutter: widgets/focus_manager.dart debugDumpFocusTree
func DebugDumpFocusTree(bc BuildContext) {
	if foundation.DebugMode {
		if b := MaybeWidgetsBindingOf(bc); b != nil {
			b.debug.DebugPrint(DebugDescribeFocusTree(bc))
		}
	}
}

// describeFocusNode describes the node the way Dart's string interpolation does,
// with "null" for no node.
func describeFocusNode(n *FocusNode) string {
	if n == nil {
		return "null"
	}
	return n.String()
}

func describeScopeNode(s *FocusScopeNode) string {
	if s == nil {
		return "null"
	}
	return s.String()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFocusChangesDuringBuild    = "FocusChangesDuringBuild"
	tagFocusNodeContextMissing    = "FocusNodeContextMissing"
	tagFocusNodeParentLoop        = "FocusNodeParentLoop"
	tagFocusNodeReparentSelf      = "FocusNodeReparentSelf"
	tagFocusNodeRootReparented    = "FocusNodeRootReparented"
	tagFocusRequestedOutsideScope = "FocusRequestedOutsideScope"
	tagFocusScopeNotChild         = "FocusScopeNotChild"
	tagFocusScopeSelfReference    = "FocusScopeSelfReference"
)
