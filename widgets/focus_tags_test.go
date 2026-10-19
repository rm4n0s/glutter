package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// Route tests of the failure tags of the focus system (SPEC §14): each case checks
// the tag, not only that something failed.

func TestFocusFailureTags(t *testing.T) {
	skipUnlessDebug(t)

	t.Run("FocusNotFound", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewContainer(nil))
		// The View is above every focus scope.
		viewContext := ts.Element(gluttertest.ByType[*View]()).(BuildContext)
		requireTag(t, "FocusNotFound", func() { FocusOf(viewContext) })
	})

	t.Run("FocusScopeFound", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[State]()
		ts.PumpWidget(NewContainer(nil, ContainerOpts{Key: key}))
		// The nearest node is the scope of the View, not a Focus.
		requireTag(t, "FocusScopeFound", func() { FocusOf(key.CurrentContext()) })
	})

	t.Run("FocusTraversalPolicyMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewContainer(nil))
		viewContext := ts.Element(gluttertest.ByType[*View]()).(BuildContext)
		requireTag(t, "FocusTraversalPolicyMissing", func() { FocusTraversalGroupOf(viewContext) })
	})

	t.Run("FocusTraversalOrderNotFound", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := NewGlobalKey[State]()
		ts.PumpWidget(NewContainer(nil, ContainerOpts{Key: key}))
		requireTag(t, "FocusTraversalOrderNotFound", func() { FocusTraversalOrderOf(key.CurrentContext()) })
	})

	t.Run("FocusOrderIncomparable", func(t *testing.T) {
		requireTag(t, "FocusOrderIncomparable", func() {
			CompareFocusOrders(NewNumericFocusOrder(1), NewLexicalFocusOrder("a"))
		})
	})

	t.Run("FocusTraversalNodeNotSorted", func(t *testing.T) {
		ts := gluttertest.New(t)
		nodes := make([]*FocusNode, 4)
		children := make([]Widget, len(nodes))
		for i := range nodes {
			nodes[i] = NewFocusNode()
			defer nodes[i].Dispose()
			children[i] = NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: nodes[i]})
		}
		ts.PumpWidget(NewFocusTraversalGroup(NewColumn(children), FocusTraversalGroupOpts{Policy: newDropsCurrentPolicy()}))
		nodes[1].RequestFocus()
		ts.Pump()
		requireTag(t, "FocusTraversalNodeNotSorted", func() { nodes[1].NextFocus() })
	})

	t.Run("FocusChangesDuringBuild", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewContainer(nil))
		ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
			WidgetsBindingOf(bc).FocusManager().ApplyFocusChangesIfNeeded()
			return NewContainer(nil)
		}))
		_ = requireErrorTag(t, takeFirst(t, ts), "FocusChangesDuringBuild")
	})

	t.Run("FocusNodeContextMissing", func(t *testing.T) {
		node := NewFocusNode()
		defer node.Dispose()
		requireTag(t, "FocusNodeContextMissing", func() { node.Rect() })
	})

	t.Run("FocusRequestedOutsideScope", func(t *testing.T) {
		ts := gluttertest.New(t)
		scope1, scope2 := NewFocusScopeNode(), NewFocusScopeNode()
		node := NewFocusNode()
		defer scope1.Dispose()
		defer scope2.Dispose()
		defer node.Dispose()
		ts.PumpWidget(NewColumn([]Widget{
			NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope1}),
			NewFocusScope(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: node}), FocusScopeOpts{Node: scope2}),
		}))
		requireTag(t, "FocusRequestedOutsideScope", func() { scope1.RequestFocus(node) })
	})

	t.Run("FocusNodeReparentSelf", func(t *testing.T) {
		ts := gluttertest.New(t)
		ctx := focusContext(ts)
		a := attach(t, ctx, NewFocusNode())
		requireTag(t, "FocusNodeReparentSelf", func() { a.attachment.Reparent(a.node) })
	})

	t.Run("FocusNodeParentLoop", func(t *testing.T) {
		ts := gluttertest.New(t)
		ctx := focusContext(ts)
		parent, child := attach(t, ctx, NewFocusNode()), attach(t, ctx, NewFocusNode())
		parent.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
		child.attachment.Reparent(parent.node)
		requireTag(t, "FocusNodeParentLoop", func() { parent.attachment.Reparent(child.node) })
	})

	t.Run("FocusNodeRootReparented", func(t *testing.T) {
		ts := gluttertest.New(t)
		ctx := focusContext(ts)
		a := attach(t, ctx, NewFocusNode())
		root := ts.Binding.FocusManager().RootScope()
		a.attachment.Reparent(root.Node())
		// Asking a node to take the root scope as its child reparents the root.
		requireTag(t, "FocusNodeRootReparented", func() { a.node.RequestFocus(root.Node()) })
	})

	t.Run("FocusScopeSelfReference", func(t *testing.T) {
		ts := gluttertest.New(t)
		scope := NewFocusScopeNode()
		defer scope.Dispose()
		ts.PumpWidget(NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope}))
		requireTag(t, "FocusScopeSelfReference", func() { scope.SetFirstFocus(scope) })
	})

	t.Run("FocusScopeNotChild", func(t *testing.T) {
		ts := gluttertest.New(t)
		scope1, scope2 := NewFocusScopeNode(), NewFocusScopeNode()
		defer scope1.Dispose()
		defer scope2.Dispose()
		ts.PumpWidget(NewColumn([]Widget{
			NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope1}),
			NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope2}),
		}))
		requireTag(t, "FocusScopeNotChild", func() { scope1.SetFirstFocus(scope2) })
	})
}

// dropsCurrentPolicy is a policy whose SortDescendants removes the current node,
// which breaks traversal.
type dropsCurrentPolicy struct{ FocusTraversalPolicyBase }

func newDropsCurrentPolicy() *dropsCurrentPolicy {
	p := &dropsCurrentPolicy{}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{})
	return p
}

func (p *dropsCurrentPolicy) SortDescendants(descendants []*FocusNode, current *FocusNode) []*FocusNode {
	var out []*FocusNode
	for _, n := range descendants {
		if n != current {
			out = append(out, n)
		}
	}
	return out
}
