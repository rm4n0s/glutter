package widgets

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// FocusOpts are the optional parameters of NewFocus.
//
// Flutter: widgets/focus_scope.dart Focus.new
type FocusOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FocusNode is the node the widget uses; without one the widget makes and
	// disposes its own. A node that is given is not disposed by the widget.
	FocusNode *FocusNode
	// ParentNode is the node that becomes the parent of the widget's node; the
	// default is the node of the nearest Focus or FocusScope above, or the root
	// scope.
	ParentNode *FocusNode
	// Autofocus asks for the focus when the widget is built, if nothing in the
	// scope has it.
	Autofocus bool
	// OnFocusChange is called with whether the node (or a descendant) has the
	// focus whenever that changes. It is not called when the widget leaves the
	// tree with the focus: to hear that, give the widget a FocusNode you own and
	// listen to it (see FocusNode, SPEC §11.7).
	OnFocusChange func(hasFocus bool)
	// OnKeyEvent is called for a key event when the node or a descendant has the
	// focus; the default is the one of FocusNode.
	OnKeyEvent FocusOnKeyEventCallback
	// CanRequestFocus is whether the node can take the focus; the default is the
	// one of FocusNode, or true.
	CanRequestFocus foundation.Opt[bool]
	// SkipTraversal is whether focus traversal skips the node; the default is the
	// one of FocusNode, or false.
	SkipTraversal foundation.Opt[bool]
	// DescendantsAreFocusable is whether the descendants can take the focus; the
	// default is the one of FocusNode, or true.
	DescendantsAreFocusable foundation.Opt[bool]
	// DescendantsAreTraversable is whether focus traversal visits the
	// descendants; the default is the one of FocusNode, or true.
	DescendantsAreTraversable foundation.Opt[bool]
	// ExcludeSemantics leaves out the Semantics node that tells the system the
	// widget is focusable and whether it has the focus (Flutter's
	// includeSemantics is true by default).
	ExcludeSemantics bool
	// DebugLabel names the node in debug output; the default is the one of
	// FocusNode.
	DebugLabel string
}

// Focus manages the focus of a widget subtree: it owns a FocusNode (or uses the
// one given), puts it in the focus tree, and handles key events for it. The
// nearest Focus above a context is found with FocusOf.
//
// Flutter: widgets/focus_scope.dart Focus
type Focus struct {
	StatefulWidget
	child Widget
	o     FocusOpts
	// external is Flutter's _usingExternalFocus: the widget follows the node
	// instead of configuring it.
	external bool
	// scope is whether the widget is a FocusScope.
	scope bool
}

// NewFocus creates a Focus widget.
//
// Flutter: widgets/focus_scope.dart Focus.new
func NewFocus(child Widget, opts ...FocusOpts) *Focus {
	o := firstOpts(opts)
	return &Focus{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: o}
}

// FocusWithExternalFocusNodeOpts are the optional parameters of
// NewFocusWithExternalFocusNode.
//
// Flutter: widgets/focus_scope.dart Focus.withExternalFocusNode
type FocusWithExternalFocusNodeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ParentNode is the node that becomes the parent of the node.
	ParentNode *FocusNode
	// Autofocus asks for the focus when the widget is built.
	Autofocus bool
	// OnFocusChange is called when the focus of the node changes.
	OnFocusChange func(hasFocus bool)
	// ExcludeSemantics leaves out the Semantics node.
	ExcludeSemantics bool
}

// NewFocusWithExternalFocusNode creates a Focus widget that follows the node
// given: its properties (key handler, whether it can be focused, whether it is
// skipped) are the node's, and the widget never changes them.
//
// Flutter: widgets/focus_scope.dart Focus.withExternalFocusNode
func NewFocusWithExternalFocusNode(focusNode *FocusNode, child Widget, opts ...FocusWithExternalFocusNodeOpts) *Focus {
	o := firstOpts(opts)
	return &Focus{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, external: true,
		o: FocusOpts{FocusNode: focusNode, ParentNode: o.ParentNode, Autofocus: o.Autofocus, OnFocusChange: o.OnFocusChange,
			ExcludeSemantics: o.ExcludeSemantics},
	}
}

func (w *Focus) focus() *Focus { return w }

// focusWidget is a Focus or a FocusScope.
type focusWidget interface {
	Widget
	focus() *Focus
}

// Child is the widget below.
//
// Flutter: widgets/focus_scope.dart Focus.child
func (w *Focus) Child() Widget { return w.child }

// ParentNode is the node that becomes the parent of the widget's node, or nil.
//
// Flutter: widgets/focus_scope.dart Focus.parentNode
func (w *Focus) ParentNode() *FocusNode { return w.o.ParentNode }

// FocusNode is the node given to the widget, or nil.
//
// Flutter: widgets/focus_scope.dart Focus.focusNode
func (w *Focus) FocusNode() *FocusNode { return w.o.FocusNode }

// Autofocus is whether the widget asks for the focus when it is built.
//
// Flutter: widgets/focus_scope.dart Focus.autofocus
func (w *Focus) Autofocus() bool { return w.o.Autofocus }

// OnFocusChange is the function called when the focus changes.
//
// Flutter: widgets/focus_scope.dart Focus.onFocusChange
func (w *Focus) OnFocusChange() func(bool) { return w.o.OnFocusChange }

// OnKeyEvent is the key handler of the widget: its own, or the node's.
//
// Flutter: widgets/focus_scope.dart Focus.onKeyEvent
func (w *Focus) OnKeyEvent() FocusOnKeyEventCallback {
	if w.external {
		return w.o.FocusNode.OnKeyEvent()
	}
	if w.o.OnKeyEvent != nil {
		return w.o.OnKeyEvent
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.OnKeyEvent()
	}
	return nil
}

// CanRequestFocus is whether the node can take the focus: the widget's value,
// else the node's, else true.
//
// Flutter: widgets/focus_scope.dart Focus.canRequestFocus
func (w *Focus) CanRequestFocus() bool {
	if w.external {
		return w.o.FocusNode.CanRequestFocus()
	}
	if w.o.CanRequestFocus.IsSome() {
		return w.o.CanRequestFocus.Value()
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.CanRequestFocus()
	}
	return true
}

// SkipTraversal is whether focus traversal skips the node.
//
// Flutter: widgets/focus_scope.dart Focus.skipTraversal
func (w *Focus) SkipTraversal() bool {
	if w.external {
		return w.o.FocusNode.SkipTraversal()
	}
	if w.o.SkipTraversal.IsSome() {
		return w.o.SkipTraversal.Value()
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.SkipTraversal()
	}
	return false
}

// DescendantsAreFocusable is whether the descendants can take the focus.
//
// Flutter: widgets/focus_scope.dart Focus.descendantsAreFocusable
func (w *Focus) DescendantsAreFocusable() bool {
	if w.external {
		return w.o.FocusNode.DescendantsAreFocusable()
	}
	if w.o.DescendantsAreFocusable.IsSome() {
		return w.o.DescendantsAreFocusable.Value()
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.DescendantsAreFocusable()
	}
	return true
}

// DescendantsAreTraversable is whether focus traversal visits the descendants.
//
// Flutter: widgets/focus_scope.dart Focus.descendantsAreTraversable
func (w *Focus) DescendantsAreTraversable() bool {
	if w.external {
		return w.o.FocusNode.DescendantsAreTraversable()
	}
	if w.o.DescendantsAreTraversable.IsSome() {
		return w.o.DescendantsAreTraversable.Value()
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.DescendantsAreTraversable()
	}
	return true
}

// IncludeSemantics is whether the widget adds a Semantics node for the focus.
//
// Flutter: widgets/focus_scope.dart Focus.includeSemantics
func (w *Focus) IncludeSemantics() bool { return !w.o.ExcludeSemantics }

// DebugLabel is the name of the node: the widget's, else the node's.
//
// Flutter: widgets/focus_scope.dart Focus.debugLabel
func (w *Focus) DebugLabel() string {
	if w.external {
		return w.o.FocusNode.DebugLabel()
	}
	if w.o.DebugLabel != "" {
		return w.o.DebugLabel
	}
	if w.o.FocusNode != nil {
		return w.o.FocusNode.DebugLabel()
	}
	return ""
}

// FocusOfOpts are the optional parameters of FocusOf and FocusMaybeOf.
//
// Flutter: widgets/focus_scope.dart Focus.of
type FocusOfOpts struct {
	// ScopeOK allows the node found to be a FocusScopeNode.
	ScopeOK bool
	// CreateDependency says whether the context depends on the node and is
	// rebuilt when its focus changes; the default is true.
	CreateDependency foundation.Opt[bool]
}

// FocusOf is the focus node of the nearest Focus above the context, and the
// context depends on it by default. It panics with the tag FocusNotFound when
// there is none, and with FocusScopeFound when the nearest one is a FocusScope
// and ScopeOK is not set.
//
// Flutter: widgets/focus_scope.dart Focus.of
func FocusOf(bc BuildContext, opts ...FocusOfOpts) *FocusNode {
	o := firstOpts(opts)
	if foundation.DebugMode {
		scopeOK := o.ScopeOK
		o.ScopeOK = true
		node := FocusMaybeOf(bc, o)
		if node == nil {
			panic(errors.New(tagFocusNotFound,
				"FocusOf() was called with a context that does not contain a Focus widget: no Focus widget ancestor could be found starting from the context that was passed to FocusOf(); this can happen because you are using a widget that looks for a Focus ancestor, and do not have a Focus widget descendant in the nearest FocusScope",
				"context", describeContext(bc)))
		}
		if !scopeOK && node.AsScope() != nil {
			panic(errors.New(tagFocusScopeFound,
				"FocusOf() was called with a context that does not contain a Focus between the given context and the nearest FocusScope widget: no Focus ancestor could be found starting from the context that was passed to FocusOf() to the point where it found the nearest FocusScope widget; this can happen because you are using a widget that looks for a Focus ancestor, and do not have a Focus widget ancestor in the current FocusScope",
				"context", describeContext(bc)))
		}
		return node
	}
	return FocusMaybeOf(bc, o)
}

// FocusMaybeOf is FocusOf that returns nil instead of panicking.
//
// Flutter: widgets/focus_scope.dart Focus.maybeOf
func FocusMaybeOf(bc BuildContext, opts ...FocusOfOpts) *FocusNode {
	o := firstOpts(opts)
	var scope *focusInheritedScope
	var ok bool
	if o.CreateDependency.OrElse(true) {
		scope, ok = DependOn[*focusInheritedScope](bc)
	} else {
		scope, ok = GetInherited[*focusInheritedScope](bc)
	}
	if !ok || scope.node == nil {
		return nil
	}
	if scope.node.AsScope() != nil && !o.ScopeOK {
		return nil
	}
	return scope.node
}

// FocusIsAt is whether the nearest Focus above the context has the focus.
//
// Flutter: widgets/focus_scope.dart Focus.isAt
func FocusIsAt(bc BuildContext) bool {
	n := FocusMaybeOf(bc)
	return n != nil && n.HasFocus()
}

// DebugFillProperties adds the focus options of the widget.
//
// Flutter: widgets/focus_scope.dart Focus.debugFillProperties
func (w *Focus) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("debugLabel", optString(w.DebugLabel())))
	p.Add(foundation.NewFlagProperty("autofocus", foundation.Some(w.o.Autofocus), foundation.FlagPropertyOpts{
		IfTrue: "AUTOFOCUS", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("canRequestFocus", foundation.Some(w.CanRequestFocus()), foundation.FlagPropertyOpts{
		IfFalse: "NOT FOCUSABLE", DefaultValue: foundation.Some(false)}))
	p.Add(foundation.NewFlagProperty("descendantsAreFocusable", foundation.Some(w.DescendantsAreFocusable()), foundation.FlagPropertyOpts{
		IfFalse: "DESCENDANTS UNFOCUSABLE", DefaultValue: foundation.Some(true)}))
	p.Add(foundation.NewFlagProperty("descendantsAreTraversable", foundation.Some(w.DescendantsAreTraversable()), foundation.FlagPropertyOpts{
		IfFalse: "DESCENDANTS UNTRAVERSABLE", DefaultValue: foundation.Some(true)}))
	if w.o.FocusNode != nil {
		p.Add(foundation.NewDiagnosticsProperty("focusNode", w.o.FocusNode))
	}
}

func optString(s string) foundation.Opt[string] {
	if s == "" {
		return foundation.None[string]()
	}
	return foundation.Some(s)
}

// CreateState creates the state.
//
// Flutter: widgets/focus_scope.dart Focus.createState
func (w *Focus) CreateState() State { return &focusState{} }

// focusState is the state of Focus and FocusScope.
//
// Flutter: widgets/focus_scope.dart _FocusState
type focusState struct {
	StateOf[focusWidget]
	internalNode               *FocusNode
	hadPrimaryFocus            bool
	couldRequestFocus          bool
	descendantsWereFocusable   bool
	descendantsWereTraversable bool
	didAutofocus               bool
	attachment                 *FocusAttachment
	listener                   foundation.ListenerHandle
}

func (s *focusState) w() *Focus { return s.Widget().focus() }

func (s *focusState) focusNode() *FocusNode {
	if n := s.w().o.FocusNode; n != nil {
		return n
	}
	if s.internalNode == nil {
		s.internalNode = s.createNode()
	}
	return s.internalNode
}

func (s *focusState) createNode() *FocusNode {
	w := s.w()
	if w.scope {
		return &NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{
			DebugLabel:         w.DebugLabel(),
			CannotRequestFocus: !w.CanRequestFocus(),
			SkipTraversal:      w.SkipTraversal(),
		}}).FocusNode
	}
	return NewFocusNode(FocusNodeOpts{
		DebugLabel:                   w.DebugLabel(),
		CannotRequestFocus:           !w.CanRequestFocus(),
		DescendantsAreNotFocusable:   !w.DescendantsAreFocusable(),
		DescendantsAreNotTraversable: !w.DescendantsAreTraversable(),
		SkipTraversal:                w.SkipTraversal(),
	})
}

func (s *focusState) InitState() { s.initNode() }

func (s *focusState) initNode() {
	w := s.w()
	node := s.focusNode()
	if !w.external {
		node.SetDescendantsAreFocusable(w.DescendantsAreFocusable())
		node.SetDescendantsAreTraversable(w.DescendantsAreTraversable())
		node.SetSkipTraversal(w.SkipTraversal())
		if w.o.CanRequestFocus.IsSome() {
			node.SetCanRequestFocus(w.o.CanRequestFocus.Value())
		}
	}
	s.couldRequestFocus = node.CanRequestFocus()
	s.descendantsWereFocusable = node.DescendantsAreFocusable()
	s.descendantsWereTraversable = node.DescendantsAreTraversable()
	s.hadPrimaryFocus = node.HasPrimaryFocus()
	s.attachment = node.Attach(s.Context(), AttachOpts{OnKeyEvent: w.OnKeyEvent()})
	// The listener is added even when the internal node existed before, since a
	// node that is re-used should have removed its listener already.
	s.listener = node.AddListener(s.handleFocusChanged)
}

func (s *focusState) Dispose() {
	// Whoever owns the node, it is removed from the tree and not listened to.
	node := s.focusNode()
	node.RemoveListener(s.listener)
	s.attachment.Detach()
	// The lifetime of an external node is not managed here, only the internal
	// node's.
	if s.internalNode != nil {
		s.internalNode.Dispose()
	}
}

func (s *focusState) DidChangeDependencies() {
	if s.attachment != nil {
		s.attachment.Reparent()
	}
	s.handleAutofocus()
}

func (s *focusState) handleAutofocus() {
	if !s.didAutofocus && s.w().o.Autofocus {
		FocusScopeOf(s.Context()).Autofocus(s.focusNode())
		s.didAutofocus = true
	}
}

func (s *focusState) Deactivate() {
	// The node's location in the tree is no longer valid. It is neither unfocused
	// nor removed because a widget that is moved to another part of the tree
	// (with a global key) should keep its focus state. So it is parked on the
	// root focus node until it is moved (DidChangeDependencies) or disposed.
	if s.attachment != nil {
		s.attachment.Reparent()
	}
	s.didAutofocus = false
}

func (s *focusState) DidUpdateWidget(old focusWidget) {
	w := s.w()
	oldW := old.focus()
	node := s.focusNode()
	if foundation.DebugMode && oldW.o.FocusNode == w.o.FocusNode && !w.external && oldW.DebugLabel() != w.DebugLabel() {
		node.SetDebugLabel(w.DebugLabel())
	}
	if oldW.o.FocusNode == w.o.FocusNode {
		if !w.external {
			node.SetOnKeyEvent(w.OnKeyEvent())
			node.SetSkipTraversal(w.SkipTraversal())
			if w.o.CanRequestFocus.IsSome() {
				node.SetCanRequestFocus(w.o.CanRequestFocus.Value())
			}
			node.SetDescendantsAreFocusable(w.DescendantsAreFocusable())
			node.SetDescendantsAreTraversable(w.DescendantsAreTraversable())
		}
	} else {
		s.attachment.Detach()
		if oldW.o.FocusNode != nil {
			oldW.o.FocusNode.RemoveListener(s.listener)
		}
		s.initNode()
	}
	if oldW.o.Autofocus != w.o.Autofocus {
		s.handleAutofocus()
	}
}

func (s *focusState) handleFocusChanged() {
	node := s.focusNode()
	hasPrimaryFocus := node.HasPrimaryFocus()
	canRequestFocus := node.CanRequestFocus()
	descendantsAreFocusable := node.DescendantsAreFocusable()
	descendantsAreTraversable := node.DescendantsAreTraversable()
	if cb := s.w().o.OnFocusChange; cb != nil {
		cb(node.HasFocus())
	}
	// The cached states that matter here are checked, and SetState is called
	// when they changed.
	if s.hadPrimaryFocus != hasPrimaryFocus {
		s.SetState(func() { s.hadPrimaryFocus = hasPrimaryFocus })
	}
	if s.couldRequestFocus != canRequestFocus {
		s.SetState(func() { s.couldRequestFocus = canRequestFocus })
	}
	if s.descendantsWereFocusable != descendantsAreFocusable {
		s.SetState(func() { s.descendantsWereFocusable = descendantsAreFocusable })
	}
	if s.descendantsWereTraversable != descendantsAreTraversable {
		s.SetState(func() { s.descendantsWereTraversable = descendantsAreTraversable })
	}
}

func (s *focusState) Build(bc BuildContext) Widget {
	w := s.w()
	node := s.focusNode()
	var parent *FocusNode
	if w.o.ParentNode != nil {
		parent = w.o.ParentNode
	}
	if parent != nil {
		s.attachment.Reparent(parent)
	} else {
		s.attachment.Reparent()
	}
	if w.scope {
		var result Widget = newFocusInheritedScope(node, w.child)
		if w.IncludeSemantics() {
			result = NewSemantics(result, SemanticsOpts{ExplicitChildNodes: true})
		}
		return result
	}
	child := w.child
	if w.IncludeSemantics() {
		opts := SemanticsOpts{Focusable: foundation.Some(s.couldRequestFocus)}
		// A focusable widget requests the focus when it receives an input focus
		// action from the semantics. Nothing is needed for losing the focus
		// because if the focus is lost, another node gains it.
		if foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS && s.couldRequestFocus {
			opts.OnFocus = func() { node.RequestFocus() }
		}
		if s.couldRequestFocus {
			opts.Focused = foundation.Some(s.hadPrimaryFocus)
		}
		child = NewSemantics(w.child, opts)
	}
	if foundation.DebugMode && WidgetsBindingOf(bc).debug.PaintFocusBoxes {
		child = newDebugFocusBorder(node, child)
	}
	return newFocusInheritedScope(node, child)
}

// FocusScopeOpts are the optional parameters of NewFocusScope.
//
// Flutter: widgets/focus_scope.dart FocusScope.new
type FocusScopeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Node is the scope node the widget uses; without one the widget makes and
	// disposes its own.
	Node *FocusScopeNode
	// ParentNode is the node that becomes the parent of the scope's node.
	ParentNode *FocusNode
	// Autofocus asks for the focus when the widget is built.
	Autofocus bool
	// OnFocusChange is called with whether the scope has the focus.
	OnFocusChange func(hasFocus bool)
	// CanRequestFocus is whether the scope can take the focus; the default is the
	// one of Node, or true.
	CanRequestFocus foundation.Opt[bool]
	// SkipTraversal is whether focus traversal skips the scope.
	SkipTraversal foundation.Opt[bool]
	// OnKeyEvent is called for a key event when a node of the scope has the focus.
	OnKeyEvent FocusOnKeyEventCallback
	// DebugLabel names the node in debug output.
	DebugLabel string
	// ExcludeSemantics leaves out the Semantics node.
	ExcludeSemantics bool
	// DescendantsAreFocusable is whether the nodes in the scope can take the
	// focus.
	DescendantsAreFocusable foundation.Opt[bool]
	// DescendantsAreTraversable is whether focus traversal visits the nodes in
	// the scope.
	DescendantsAreTraversable foundation.Opt[bool]
}

// FocusScope creates a FocusScopeNode that groups the focus nodes below it: they
// have the focus in turn, the scope remembers which one had it last, and focus
// traversal stays in the scope. FocusScopeOf finds the nearest one.
//
// Flutter: widgets/focus_scope.dart FocusScope
type FocusScope struct {
	Focus
}

// NewFocusScope creates a FocusScope widget.
//
// Flutter: widgets/focus_scope.dart FocusScope.new
func NewFocusScope(child Widget, opts ...FocusScopeOpts) *FocusScope {
	o := firstOpts(opts)
	f := FocusOpts{
		Key: o.Key, ParentNode: o.ParentNode, Autofocus: o.Autofocus, OnFocusChange: o.OnFocusChange,
		OnKeyEvent: o.OnKeyEvent, CanRequestFocus: o.CanRequestFocus, SkipTraversal: o.SkipTraversal,
		DescendantsAreFocusable: o.DescendantsAreFocusable, DescendantsAreTraversable: o.DescendantsAreTraversable,
		ExcludeSemantics: o.ExcludeSemantics, DebugLabel: o.DebugLabel,
	}
	if o.Node != nil {
		f.FocusNode = o.Node.Node()
	}
	return &FocusScope{Focus{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: f, scope: true}}
}

// FocusScopeWithExternalFocusNodeOpts are the optional parameters of
// NewFocusScopeWithExternalFocusNode.
//
// Flutter: widgets/focus_scope.dart FocusScope.withExternalFocusNode
type FocusScopeWithExternalFocusNodeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ParentNode is the node that becomes the parent of the scope's node.
	ParentNode *FocusNode
	// Autofocus asks for the focus when the widget is built.
	Autofocus bool
	// ExcludeSemantics leaves out the Semantics node.
	ExcludeSemantics bool
	// OnFocusChange is called with whether the scope has the focus.
	OnFocusChange func(hasFocus bool)
}

// NewFocusScopeWithExternalFocusNode creates a FocusScope widget that follows
// the scope node given, never changing its properties.
//
// Flutter: widgets/focus_scope.dart FocusScope.withExternalFocusNode
func NewFocusScopeWithExternalFocusNode(focusScopeNode *FocusScopeNode, child Widget, opts ...FocusScopeWithExternalFocusNodeOpts) *FocusScope {
	o := firstOpts(opts)
	return &FocusScope{Focus{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, external: true, scope: true,
		o: FocusOpts{FocusNode: focusScopeNode.Node(), ParentNode: o.ParentNode, Autofocus: o.Autofocus,
			OnFocusChange: o.OnFocusChange, ExcludeSemantics: o.ExcludeSemantics},
	}}
}

func (w *FocusScope) focus() *Focus { return &w.Focus }

// CreateState creates the state.
//
// Flutter: widgets/focus_scope.dart FocusScope.createState
func (w *FocusScope) CreateState() State { return &focusState{} }

// FocusScopeOf is the nearest FocusScopeNode above the context, or the root
// scope of the focus tree when there is none; the context depends on it by
// default.
//
// Flutter: widgets/focus_scope.dart FocusScope.of
func FocusScopeOf(bc BuildContext, createDependency ...bool) *FocusScopeNode {
	dep := true
	if len(createDependency) > 0 {
		dep = createDependency[0]
	}
	node := FocusMaybeOf(bc, FocusOfOpts{ScopeOK: true, CreateDependency: foundation.Some(dep)})
	if node != nil {
		if scope := node.NearestScope(); scope != nil {
			return scope
		}
	}
	return bc.Owner().FocusManager().RootScope()
}

// debugFocusBorder paints a border that shows the state of a node.
//
// Flutter: widgets/focus_scope.dart _DebugFocusBorder
type debugFocusBorder struct {
	StatelessWidget
	node  *FocusNode
	child Widget
}

func newDebugFocusBorder(node *FocusNode, child Widget) *debugFocusBorder {
	return &debugFocusBorder{NewStatelessWidgetBase(), node, child}
}

func (w *debugFocusBorder) borderColor() ui.Color {
	switch {
	case w.node.HasPrimaryFocus():
		return ui.NewColor(0xF000FF00)
	case w.node.HasFocus():
		return ui.NewColor(0xF00000FF)
	case !w.node.CanRequestFocus():
		return ui.NewColor(0xF0FF0000)
	case w.node.SkipTraversal():
		return ui.NewColor(0xF0FFFF00)
	}
	return ui.NewColor(0xF000FFFF)
}

func (w *debugFocusBorder) Build(BuildContext) Widget {
	return NewListenableBuilder(w.node, func(BuildContext, Widget) Widget {
		return NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Border: painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(w.borderColor()), Width: foundation.Some(3.0)}),
		}), w.child, DecoratedBoxOpts{Position: rendering.DecorationPositionForeground})
	}, nil)
}

// focusInheritedScope is the inherited widget of Focus and FocusScope: the
// dependents are rebuilt when the node notifies.
//
// Flutter: widgets/focus_scope.dart _FocusInheritedScope
type focusInheritedScope struct {
	InheritedNotifier
	node *FocusNode
}

func newFocusInheritedScope(node *FocusNode, child Widget) *focusInheritedScope {
	return &focusInheritedScope{NewInheritedNotifierBase(node, child), node}
}

// ExcludeFocusOpts are the optional parameters of NewExcludeFocus.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus.new
type ExcludeFocusOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NotExcluding turns the exclusion off (Flutter's excluding is true by
	// default).
	NotExcluding bool
}

// ExcludeFocus is a widget that excludes its subtree from the focus system: no
// node below it can take the focus.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus
type ExcludeFocus struct {
	StatelessWidget
	excluding bool
	child     Widget
}

// NewExcludeFocus creates the widget.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus.new
func NewExcludeFocus(child Widget, opts ...ExcludeFocusOpts) *ExcludeFocus {
	o := firstOpts(opts)
	return &ExcludeFocus{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), !o.NotExcluding, child}
}

// Excluding is whether the subtree is excluded.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus.excluding
func (w *ExcludeFocus) Excluding() bool { return w.excluding }

// Child is the widget below.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus.child
func (w *ExcludeFocus) Child() Widget { return w.child }

// Build makes a Focus that cannot be focused and does not let its descendants be.
//
// Flutter: widgets/focus_scope.dart ExcludeFocus.build
func (w *ExcludeFocus) Build(BuildContext) Widget {
	return NewFocus(w.child, FocusOpts{
		CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true), ExcludeSemantics: true,
		DescendantsAreFocusable: foundation.Some(!w.excluding),
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFocusNotFound   = "FocusNotFound"
	tagFocusScopeFound = "FocusScopeFound"
)
