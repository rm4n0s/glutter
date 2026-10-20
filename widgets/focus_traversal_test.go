package widgets_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/focus_traversal_test.dart. The
// cases that need EditableText (the arrow keys in a text field) wait for step
// 11.4.

type gkey = *GlobalKey[State]

func newKeys(n int) []gkey {
	keys := make([]gkey, n)
	for i := range keys {
		keys[i] = NewGlobalKey[State]()
	}
	return keys
}

// sizedBoxKey is SizedBox(key: key, width: 100, height: 100).
func sizedBoxKey(key foundation.Key, size ...float64) Widget {
	w, h := 100.0, 100.0
	if len(size) == 2 {
		w, h = size[0], size[1]
	}
	return NewSizedBox(nil, SizedBoxOpts{Key: key, Width: foundation.Some(w), Height: foundation.Some(h)})
}

func ptr[T any](v T) *T { return &v }

func ptrStr(p *bool) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprint(*p)
}

func elemOf(ts *gluttertest.Tester, k foundation.Key) BuildContext {
	return ctxOf(ts, gluttertest.ByKey(k))
}

// SkipAllButFirstAndLastPolicy is used to test removal of nodes while sorting.
type skipAllButFirstAndLastPolicy struct {
	FocusTraversalPolicyBase
}

func newSkipAllButFirstAndLastPolicy() *skipAllButFirstAndLastPolicy {
	p := &skipAllButFirstAndLastPolicy{}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{Directional: true})
	return p
}

func (p *skipAllButFirstAndLastPolicy) SortDescendants(descendants []*FocusNode, currentNode *FocusNode) []*FocusNode {
	first, last := descendants[0], descendants[len(descendants)-1]
	out := []*FocusNode{first}
	if currentNode != first && currentNode != last {
		out = append(out, currentNode)
	}
	return append(out, last)
}

func TestWidgetOrderTraversalPolicy(t *testing.T) {
	t.Run("Find the initial focus if there is none yet", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(6)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocus(sizedBoxKey(k[3]), FocusOpts{Key: k[2]}),
				NewFocus(sizedBoxKey(k[5]), FocusOpts{Key: k[4]}),
			}), FocusScopeOpts{Key: k[1]}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()}))
		firstFocusNode, secondFocusNode := FocusOf(elemOf(ts, k[3])), FocusOf(elemOf(ts, k[5]))
		scope := firstFocusNode.EnclosingScope()
		secondFocusNode.NextFocus()
		ts.Pump()
		if !firstFocusNode.HasFocus() || secondFocusNode.HasFocus() || !scope.HasFocus() {
			t.Errorf("focus = %v %v %v", firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
		}
	})

	t.Run("Find the initial focus if there is none yet and traversing backwards", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(6)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocus(sizedBoxKey(k[3]), FocusOpts{Key: k[2]}),
				NewFocus(sizedBoxKey(k[5]), FocusOpts{Key: k[4]}),
			}), FocusScopeOpts{Key: k[1]}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()}))
		firstFocusNode, secondFocusNode := FocusOf(elemOf(ts, k[3])), FocusOf(elemOf(ts, k[5]))
		scope := firstFocusNode.EnclosingScope()
		if firstFocusNode.HasFocus() || secondFocusNode.HasFocus() {
			t.Fatal("no node should have the focus")
		}
		secondFocusNode.PreviousFocus()
		ts.Pump()
		if firstFocusNode.HasFocus() || !secondFocusNode.HasFocus() || !scope.HasFocus() {
			t.Errorf("focus = %v %v %v", firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
		}
	})

	// The two "focus traversal should work" cases.
	for _, tc := range []struct {
		name    string
		padding bool
	}{{"case 1", true}, {"case 2", false}} {
		t.Run("focus traversal should work "+tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			outer1, outer2 := NewFocusNode(FocusNodeOpts{DebugLabel: "outer1", SkipTraversal: true}), NewFocusNode(FocusNodeOpts{DebugLabel: "outer2", SkipTraversal: true})
			inner1, inner2 := NewFocusNode(FocusNodeOpts{DebugLabel: "inner1"}), NewFocusNode(FocusNodeOpts{DebugLabel: "inner2"})
			defer func() {
				for _, n := range []*FocusNode{outer1, outer2, inner1, inner2} {
					n.Dispose()
				}
			}()
			var second Widget = NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: inner2})
			if tc.padding {
				// Adds a padding to ensure both Focus widgets have different sizes.
				second = NewPadding(painting.NewEdgeInsetsAll(5), second)
			}
			ts.PumpWidget(ltr(NewFocusTraversalGroup(NewRow([]Widget{
				NewFocusScope(NewFocus(NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: inner1}), FocusOpts{FocusNode: outer1})),
				NewFocusScope(NewFocus(second, FocusOpts{FocusNode: outer2})),
			}))))
			fm := ts.Binding.FocusManager()
			if fm.PrimaryFocus() != nil {
				t.Fatal("there should be no primary focus")
			}
			inner1.RequestFocus()
			ts.Pump()
			if fm.PrimaryFocus() != inner1 {
				t.Fatalf("primary focus = %v, want inner1", fm.PrimaryFocus())
			}
			outer2.NextFocus()
			ts.Pump()
			if fm.PrimaryFocus() != inner2 {
				t.Errorf("primary focus = %v, want inner2", fm.PrimaryFocus())
			}
		})
	}

	t.Run("Move focus to next node", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(7)
		var focus1, focus2, focus3, focus5 *bool
		set := func(p **bool) func(bool) { return func(v bool) { *p = &v } }
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocusScope(NewColumn([]Widget{
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[4]}), FocusOpts{DebugLabel: "key3", Key: k[3], OnFocusChange: set(&focus3)}),
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[6]}), FocusOpts{DebugLabel: "key5", Key: k[5], OnFocusChange: set(&focus5)}),
				}), FocusScopeOpts{DebugLabel: "key2", Key: k[2], OnFocusChange: set(&focus2)}),
			}), FocusScopeOpts{DebugLabel: "key1", Key: k[1], OnFocusChange: set(&focus1)}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()}))
		firstChild, secondChild := elemOf(ts, k[4]), elemOf(ts, k[6])
		firstFocusNode, secondFocusNode := FocusOf(firstChild), FocusOf(secondChild)
		scope := firstFocusNode.EnclosingScope()
		reset := func() { focus1, focus2, focus3, focus5 = nil, nil, nil, nil }
		check := func(step string, f1, f2, f3, f5 *bool, first, second bool) {
			t.Helper()
			eq := func(a, b *bool) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
			if !eq(focus1, f1) || !eq(focus2, f2) || !eq(focus3, f3) || !eq(focus5, f5) {
				t.Errorf("%s: callbacks = %v %v %v %v", step, ptrStr(focus1), ptrStr(focus2), ptrStr(focus3), ptrStr(focus5))
			}
			if firstFocusNode.HasFocus() != first || secondFocusNode.HasFocus() != second || !scope.HasFocus() {
				t.Errorf("%s: focus = %v %v %v, want %v %v true", step, firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus(), first, second)
			}
		}
		yes, no := ptr(true), ptr(false)
		firstFocusNode.RequestFocus()
		ts.Pump()
		check("first requested", yes, yes, yes, nil, true, false)
		reset()
		FocusOf(firstChild).NextFocus()
		ts.Pump()
		check("next", nil, nil, no, yes, false, true)
		reset()
		FocusOf(firstChild).NextFocus()
		ts.Pump()
		check("next again", nil, nil, yes, no, true, false)
		reset()
		// Tests that can still move back to original node.
		FocusOf(firstChild).PreviousFocus()
		ts.Pump()
		check("previous", nil, nil, no, yes, false, true)
	})

	t.Run("Move focus to previous node", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(7)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocusScope(NewColumn([]Widget{
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[4]}), FocusOpts{Key: k[3]}),
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[6]}), FocusOpts{Key: k[5]}),
				}), FocusScopeOpts{Key: k[2]}),
			}), FocusScopeOpts{Key: k[1]}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()}))
		firstChild, secondChild := elemOf(ts, k[4]), elemOf(ts, k[6])
		firstFocusNode, secondFocusNode := FocusOf(firstChild), FocusOf(secondChild)
		scope := firstFocusNode.EnclosingScope()
		secondFocusNode.RequestFocus()
		ts.Pump()
		step := func(name string, act func(), first, second bool) {
			t.Helper()
			if act != nil {
				act()
				ts.Pump()
			}
			if firstFocusNode.HasFocus() != first || secondFocusNode.HasFocus() != second || !scope.HasFocus() {
				t.Errorf("%s: focus = %v %v %v", name, firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
			}
		}
		step("second requested", nil, false, true)
		step("previous", func() { FocusOf(firstChild).PreviousFocus() }, true, false)
		step("previous again", func() { FocusOf(firstChild).PreviousFocus() }, false, true)
		// Tests that can still move back to original node.
		step("next", func() { FocusOf(firstChild).NextFocus() }, true, false)
	})

	t.Run("Move focus to next/previous node while skipping nodes in policy", func(t *testing.T) {
		ts := gluttertest.New(t)
		nodes := make([]*FocusNode, 7)
		children := make([]Widget, len(nodes))
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: "Node " + string(rune('0'+i))})
			children[i] = NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: nodes[i]})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		ts.PumpWidget(NewFocusTraversalGroup(NewColumn(children), FocusTraversalGroupOpts{Policy: newSkipAllButFirstAndLastPolicy()}))
		nodes[2].RequestFocus()
		ts.Pump()
		if !nodes[2].HasPrimaryFocus() {
			t.Fatal("node 2 should have the primary focus")
		}
		ts.Binding.FocusManager().PrimaryFocus().NextFocus()
		ts.Pump()
		if !nodes[6].HasPrimaryFocus() {
			t.Fatal("node 6 should have the primary focus")
		}
		ts.Binding.FocusManager().PrimaryFocus().PreviousFocus()
		ts.Pump()
		if !nodes[0].HasPrimaryFocus() {
			t.Error("node 0 should have the primary focus")
		}
	})

	t.Run("Find the initial focus when a route is pushed or popped", func(t *testing.T) {
		testFindInitialFocusOnRoutePushPop(t, NewWidgetOrderTraversalPolicy(), false)
	})

	t.Run("Custom requestFocusCallback gets called on the next/previous focus", func(t *testing.T) {
		testCustomRequestFocusCallback(t, func(cb TraversalRequestFocusCallback) FocusTraversalPolicy {
			return NewWidgetOrderTraversalPolicy(FocusTraversalPolicyOpts{RequestFocusCallback: cb})
		}, false)
	})
}

func testCustomRequestFocusCallback(t *testing.T, policy func(TraversalRequestFocusCallback) FocusTraversalPolicy, withDirectionality bool) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	testNode1 := NewFocusNode(FocusNodeOpts{DebugLabel: "Focus Node"})
	defer testNode1.Dispose()
	called := false
	var w Widget = NewFocusTraversalGroup(
		NewFocusScope(NewFocus(NewContainer(nil), FocusOpts{Key: key1, FocusNode: testNode1}), FocusScopeOpts{DebugLabel: "key1"}),
		FocusTraversalGroupOpts{Policy: policy(func(*FocusNode, TraversalRequestFocusOpts) { called = true })})
	if withDirectionality {
		w = ltr(w)
	}
	ts.PumpWidget(w)
	scope := FocusScopeOf(elemOf(ts, key1))
	scope.NextFocus()
	ts.Pump()
	if !called {
		t.Fatal("the callback was not called on nextFocus")
	}
	called = false
	scope.PreviousFocus()
	ts.Pump()
	if !called {
		t.Error("the callback was not called on previousFocus")
	}
}

func TestNestedNavigatorDoesNotTrapFocus(t *testing.T) {
	ts := gluttertest.New(t)
	node1, node2, node3 := NewFocusNode(), NewFocusNode(), NewFocusNode()
	defer node1.Dispose()
	defer node2.Dispose()
	defer node3.Dispose()
	nested := newPage("", "", func(p *testPage) Route {
		return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
			return NewFocus(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}), FocusOpts{FocusNode: node2})
		}, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: p}}})
	})
	ts.PumpWidget(ltr(NewFocusTraversalGroup(
		NewFocusScope(NewColumn([]Widget{
			NewFocus(sizedBoxKey(nil), FocusOpts{FocusNode: node1}),
			NewSizedBox(NewNavigator(NavigatorOpts{
				Pages:     []Page{nested},
				OnPopPage: func(Route, any) bool { return false },
			}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
			NewFocus(sizedBoxKey(nil), FocusOpts{FocusNode: node3}),
		})),
		FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()})))
	node1.RequestFocus()
	ts.Pump()
	check := func(step string, a, b, c bool) {
		t.Helper()
		if node1.HasFocus() != a || node2.HasFocus() != b || node3.HasFocus() != c {
			t.Errorf("%s: focus = %v %v %v, want %v %v %v", step, node1.HasFocus(), node2.HasFocus(), node3.HasFocus(), a, b, c)
		}
	}
	check("start", true, false, false)
	for _, s := range []struct {
		name    string
		act     func()
		a, b, c bool
	}{
		{"node1 next", func() { node1.NextFocus() }, false, true, false},
		{"node2 next", func() { node2.NextFocus() }, false, false, true},
		{"node3 next", func() { node3.NextFocus() }, true, false, false},
		{"node1 previous", func() { node1.PreviousFocus() }, false, false, true},
		{"node3 previous", func() { node3.PreviousFocus() }, false, true, false},
		{"node2 previous", func() { node2.PreviousFocus() }, true, false, false},
	} {
		s.act()
		ts.Pump()
		check(s.name, s.a, s.b, s.c)
	}
}

func TestReadingOrderTraversalPolicy(t *testing.T) {
	t.Run("Find the initial focus if there is none yet", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(6)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocus(sizedBoxKey(k[3]), FocusOpts{Key: k[2]}),
				NewFocus(sizedBoxKey(k[5]), FocusOpts{Key: k[4]}),
			}), FocusScopeOpts{Key: k[1]}),
			FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()}))
		firstFocusNode, secondFocusNode := FocusOf(elemOf(ts, k[3])), FocusOf(elemOf(ts, k[5]))
		scope := firstFocusNode.EnclosingScope()
		secondFocusNode.NextFocus()
		ts.Pump()
		if !firstFocusNode.HasFocus() || secondFocusNode.HasFocus() || !scope.HasFocus() {
			t.Errorf("focus = %v %v %v", firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
		}
	})

	t.Run("Move reading focus to next node", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(7)
		var focus1, focus2, focus3, focus5 *bool
		set := func(p **bool) func(bool) { return func(v bool) { *p = &v } }
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocusScope(NewRow([]Widget{
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[4]}), FocusOpts{DebugLabel: "key3", Key: k[3], OnFocusChange: set(&focus3)}),
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[6]}), FocusOpts{DebugLabel: "key5", Key: k[5], OnFocusChange: set(&focus5)}),
				}), FocusScopeOpts{DebugLabel: "key2", Key: k[2], OnFocusChange: set(&focus2)}),
			}), FocusScopeOpts{DebugLabel: "key1", Key: k[1], OnFocusChange: set(&focus1)}),
			FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()})))
		firstChild, secondChild := elemOf(ts, k[4]), elemOf(ts, k[6])
		firstFocusNode, secondFocusNode := FocusOf(firstChild), FocusOf(secondChild)
		scope := firstFocusNode.EnclosingScope()
		clear := func() { focus1, focus2, focus3, focus5 = nil, nil, nil, nil }
		check := func(step string, f1, f2, f3, f5 *bool, first, second bool) {
			t.Helper()
			eq := func(a, b *bool) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
			if !eq(focus1, f1) || !eq(focus2, f2) || !eq(focus3, f3) || !eq(focus5, f5) {
				t.Errorf("%s: callbacks = %v %v %v %v", step, ptrStr(focus1), ptrStr(focus2), ptrStr(focus3), ptrStr(focus5))
			}
			if firstFocusNode.HasFocus() != first || secondFocusNode.HasFocus() != second || !scope.HasFocus() {
				t.Errorf("%s: focus = %v %v %v", step, firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
			}
		}
		yes, no := ptr(true), ptr(false)
		firstFocusNode.RequestFocus()
		ts.Pump()
		check("first requested", yes, yes, yes, nil, true, false)
		clear()
		FocusOf(firstChild).NextFocus()
		ts.Pump()
		check("next", nil, nil, no, yes, false, true)
		clear()
		FocusOf(firstChild).NextFocus()
		ts.Pump()
		check("next again", nil, nil, yes, no, true, false)
		clear()
		// Tests that can still move back to original node.
		FocusOf(firstChild).PreviousFocus()
		ts.Pump()
		check("previous", nil, nil, no, yes, false, true)
	})

	t.Run("Requesting nextFocus on node focuses its descendant", func(t *testing.T) {
		for _, canRequestFocus := range []bool{true, false} {
			ts := gluttertest.New(t)
			node1, node2 := NewFocusNode(), NewFocusNode()
			ts.PumpWidget(ltr(NewFocusTraversalGroup(
				NewFocusScope(NewFocus(NewFocus(NewContainer(nil), FocusOpts{FocusNode: node2}),
					FocusOpts{FocusNode: node1, CanRequestFocus: foundation.Some(canRequestFocus)})),
				FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()})))
			didFindNode := node1.NextFocus()
			ts.Pump()
			if !didFindNode {
				t.Errorf("canRequestFocus %v: no node was found", canRequestFocus)
			}
			if node1.HasPrimaryFocus() != canRequestFocus || node2.HasPrimaryFocus() == canRequestFocus {
				t.Errorf("canRequestFocus %v: primary focus = %v %v", canRequestFocus, node1.HasPrimaryFocus(), node2.HasPrimaryFocus())
			}
			node1.Dispose()
			node2.Dispose()
		}
	})

	t.Run("Move reading focus to previous node", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(7)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocusScope(NewColumn([]Widget{
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[4]}), FocusOpts{Key: k[3]}),
					NewFocus(NewContainer(nil, ContainerOpts{Key: k[6]}), FocusOpts{Key: k[5]}),
				}), FocusScopeOpts{Key: k[2]}),
			}), FocusScopeOpts{Key: k[1]}),
			FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()}))
		firstChild, secondChild := elemOf(ts, k[4]), elemOf(ts, k[6])
		firstFocusNode, secondFocusNode := FocusOf(firstChild), FocusOf(secondChild)
		scope := firstFocusNode.EnclosingScope()
		secondFocusNode.RequestFocus()
		ts.Pump()
		step := func(name string, act func(), first, second bool) {
			t.Helper()
			if act != nil {
				act()
				ts.Pump()
			}
			if firstFocusNode.HasFocus() != first || secondFocusNode.HasFocus() != second || !scope.HasFocus() {
				t.Errorf("%s: focus = %v %v %v", name, firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
			}
		}
		step("second requested", nil, false, true)
		step("previous", func() { FocusOf(firstChild).PreviousFocus() }, true, false)
		step("previous again", func() { FocusOf(firstChild).PreviousFocus() }, false, true)
		step("next", func() { FocusOf(firstChild).NextFocus() }, true, false)
	})

	t.Run("Focus order is correct in the presence of different directionalities", func(t *testing.T) {
		ts := gluttertest.New(t)
		const nodeCount = 10
		scopeNode := NewFocusScopeNode()
		defer scopeNode.Dispose()
		nodes := make([]*FocusNode, nodeCount)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		box := func(i int) Widget {
			return NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: nodes[i]})
		}
		dir := func(d ui.TextDirection, child Widget) Widget { return NewDirectionality(d, child) }
		buildTest := func(top ui.TextDirection) Widget {
			return dir(top, NewFocusTraversalGroup(
				NewFocusScope(NewColumn([]Widget{
					dir(ui.TextDirectionLTR, NewRow([]Widget{box(0), box(1), box(2)})),
					dir(ui.TextDirectionLTR, NewRow([]Widget{
						dir(ui.TextDirectionRTL, box(3)),
						dir(ui.TextDirectionRTL, box(4)),
						dir(ui.TextDirectionLTR, box(5)),
					})),
					NewRow([]Widget{
						dir(ui.TextDirectionLTR, box(6)),
						dir(ui.TextDirectionRTL, box(7)),
						dir(ui.TextDirectionRTL, box(8)),
						dir(ui.TextDirectionLTR, box(9)),
					}),
				}), FocusScopeOpts{Node: scopeNode}),
				FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()}))
		}
		traverse := func() []int {
			var order []int
			for i := 0; i < nodeCount; i++ {
				nodes[0].NextFocus()
				ts.Pump()
				order = append(order, slices.Index(nodes, ts.Binding.FocusManager().PrimaryFocus()))
			}
			return order
		}
		ts.PumpWidget(buildTest(ui.TextDirectionRTL))
		// The last four *are* correct: the Row is sensitive to the directionality
		// too, so it swaps the positions of 7 and 8.
		if got, want := traverse(), []int{0, 1, 2, 4, 3, 5, 6, 7, 8, 9}; !slices.Equal(got, want) {
			t.Errorf("rtl order = %v, want %v", got, want)
		}
		ts.PumpWidget(buildTest(ui.TextDirectionLTR))
		if got, want := traverse(), []int{0, 1, 2, 4, 3, 5, 6, 8, 7, 9}; !slices.Equal(got, want) {
			t.Errorf("ltr order = %v, want %v", got, want)
		}
	})

	t.Run("Focus order is reading order regardless of widget order, even when overlapping", func(t *testing.T) {
		ts := gluttertest.New(t)
		const nodeCount = 10
		nodes := make([]*FocusNode, nodeCount)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		traverse := func() []int {
			var order []int
			for i := 0; i < nodeCount; i++ {
				nodes[0].NextFocus()
				ts.Pump()
				order = append(order, slices.Index(nodes, ts.Binding.FocusManager().PrimaryFocus()))
			}
			return order
		}
		stack := func(alignment painting.AlignmentGeometry, child func(i int) Widget) Widget {
			children := make([]Widget, nodeCount)
			for i := range children {
				children[i] = child(i)
			}
			return NewDirectionality(ui.TextDirectionRTL, NewFocusTraversalGroup(NewStack(children, StackOpts{Alignment: alignment}),
				FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy()}))
		}
		sized := func(i int) Widget {
			return NewFocus(sizedBoxKey(nil, 10*float64(i+1), 10*float64(i+1)), FocusOpts{FocusNode: nodes[i]})
		}
		descending := []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
		// Boxes that all have the same upper left origin corner.
		ts.PumpWidget(stack(painting.AlignmentTopLeft, sized))
		if got := traverse(); !slices.Equal(got, descending) {
			t.Errorf("top left order = %v, want %v", got, descending)
		}
		// Concentric boxes.
		ts.PumpWidget(stack(painting.AlignmentCenter, sized))
		if got := traverse(); !slices.Equal(got, descending) {
			t.Errorf("concentric order = %v, want %v", got, descending)
		}
		// Stacked (vertically) and centered (horizontally, on each other) widgets, not overlapping.
		ts.PumpWidget(stack(painting.AlignmentCenter, func(i int) Widget {
			return NewPositioned(sized(i), PositionedOpts{Top: foundation.Some(5.0 * float64(i) * float64(i+1)), Left: foundation.Some(5.0 * float64(9-i))})
		}))
		if got, want := traverse(), []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}; !slices.Equal(got, want) {
			t.Errorf("stacked order = %v, want %v", got, want)
		}
	})

	t.Run("Custom requestFocusCallback gets called on the next/previous focus", func(t *testing.T) {
		testCustomRequestFocusCallback(t, func(cb TraversalRequestFocusCallback) FocusTraversalPolicy {
			return NewReadingOrderTraversalPolicy(FocusTraversalPolicyOpts{RequestFocusCallback: cb})
		}, true)
	})
}

// testFindInitialFocusOnRoutePushPop is the 'Find the initial focus when a route is
// pushed or popped' case of the WidgetOrderTraversalPolicy and
// OrderedTraversalPolicy groups; the ordered one wraps the buttons in a
// FocusTraversalOrder.
func testFindInitialFocusOnRoutePushPop(t *testing.T, policy FocusTraversalPolicy, ordered bool) {
	ts := gluttertest.New(t)
	k := newKeys(3)
	testNode1, testNode2 := NewFocusNode(FocusNodeOpts{DebugLabel: "First Focus Node"}), NewFocusNode(FocusNodeOpts{DebugLabel: "Second Focus Node"})
	defer testNode1.Dispose()
	defer testNode2.Dispose()
	button := func(key foundation.Key, node *FocusNode, label string, onPressed func()) Widget {
		var w Widget = NewFocus(NewGestureDetector(navText(label), GestureDetectorOpts{OnTap: onPressed}), FocusOpts{Key: key, FocusNode: node, Autofocus: true})
		if ordered {
			w = NewFocusTraversalOrder(NewNumericFocusOrder(0), w)
		}
		return w
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewFocusTraversalGroup(
		NewCenter(NewBuilder(func(bc BuildContext) Widget {
			return button(k[1], testNode1, "Go Forward", func() {
				NavigatorOf(bc).Push(NewPageRouteBuilder(func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
					return NewCenter(button(k[2], testNode2, "Go Back", func() { NavigatorOf(bc).Pop() }))
				}))
			})
		})),
		FocusTraversalGroupOpts{Policy: policy})}))
	firstFocusNode := FocusOf(ctxOf(ts, gluttertest.ByText("Go Forward")))
	scope := firstFocusNode.EnclosingScope()
	ts.Pump()
	if !firstFocusNode.HasFocus() || !scope.HasFocus() {
		t.Fatal("the first node should have the focus")
	}
	ts.Tap(gluttertest.ByText("Go Forward"))
	ts.PumpAndSettle()
	secondFocusNode := FocusOf(ctxOf(ts, gluttertest.ByText("Go Back")))
	if firstFocusNode.HasFocus() || !secondFocusNode.HasFocus() {
		t.Fatal("the second node should have the focus")
	}
	ts.Tap(gluttertest.ByText("Go Back"))
	ts.PumpAndSettle()
	if !firstFocusNode.HasFocus() || !scope.HasFocus() {
		t.Error("the first node should have the focus again")
	}
}

func TestOrderedTraversalPolicy(t *testing.T) {
	newNodes := func(n int, label func(i int) string) ([]*FocusNode, func()) {
		nodes := make([]*FocusNode, n)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: label(i)})
		}
		return nodes, func() {
			for _, node := range nodes {
				node.Dispose()
			}
		}
	}
	nodeLabel := func(i int) string { return fmt.Sprintf("Node %d", i) }

	t.Run("Find the initial focus if there is none yet", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(3)
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewFocusTraversalOrder(NewNumericFocusOrder(2), NewFocus(sizedBoxKey(k[1]))),
				NewFocusTraversalOrder(NewNumericFocusOrder(1), NewFocus(sizedBoxKey(k[2]))),
			})),
			FocusTraversalGroupOpts{Policy: NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{Secondary: NewReadingOrderTraversalPolicy()})}))
		firstFocusNode, secondFocusNode := FocusOf(elemOf(ts, k[1])), FocusOf(elemOf(ts, k[2]))
		scope := firstFocusNode.EnclosingScope()
		secondFocusNode.NextFocus()
		ts.Pump()
		if firstFocusNode.HasFocus() || !secondFocusNode.HasFocus() || !scope.HasFocus() {
			t.Errorf("focus = %v %v %v", firstFocusNode.HasFocus(), secondFocusNode.HasFocus(), scope.HasFocus())
		}
	})

	t.Run("Fall back to the secondary sort if no FocusTraversalOrder exists", func(t *testing.T) {
		ts := gluttertest.New(t)
		const nodeCount = 10
		nodes, dispose := newNodes(nodeCount, nodeLabel)
		defer dispose()
		children := make([]Widget, nodeCount)
		for i := range children {
			children[i] = NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: nodes[i]})
		}
		ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewFocusTraversalGroup(
			NewFocusScope(NewRow(children)),
			FocusTraversalGroupOpts{Policy: NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{Secondary: NewWidgetOrderTraversalPolicy()})})))
		// Because it should be using widget order, this shouldn't be affected by
		// the directionality.
		for i := 0; i < nodeCount; i++ {
			nodes[0].NextFocus()
			ts.Pump()
			if !nodes[i].HasPrimaryFocus() {
				t.Errorf("node %d doesn't have focus, but should", i)
			}
		}
		// Now check backwards.
		for i := nodeCount - 1; i > 0; i-- {
			nodes[0].PreviousFocus()
			ts.Pump()
			if !nodes[i-1].HasPrimaryFocus() {
				t.Errorf("node %d doesn't have focus, but should", i-1)
			}
		}
	})

	for _, tc := range []struct {
		name  string
		label func(i int) string
		order func(i, nodeCount int) FocusOrder
	}{
		{"numerical", nodeLabel, func(i, n int) FocusOrder { return NewNumericFocusOrder(float64(n - i)) }},
		// Generates ['J' ... 'A'].
		{"lexical", func(i int) string { return "Node " + string(rune('A'+10-i-1)) },
			func(i, n int) FocusOrder { return NewLexicalFocusOrder(string(rune('A' + n - i - 1))) }},
	} {
		t.Run("Move focus to next/previous node using "+tc.name+" order", func(t *testing.T) {
			ts := gluttertest.New(t)
			const nodeCount = 10
			nodes, dispose := newNodes(nodeCount, tc.label)
			defer dispose()
			children := make([]Widget, nodeCount)
			for i := range children {
				children[i] = NewFocusTraversalOrder(tc.order(i, nodeCount), NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: nodes[i]}))
			}
			ts.PumpWidget(ltr(NewFocusTraversalGroup(NewFocusScope(NewRow(children)),
				FocusTraversalGroupOpts{Policy: NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{Secondary: NewWidgetOrderTraversalPolicy()})})))
			// The orders are assigned to be backwards from normal, so should go backwards.
			for i := nodeCount - 1; i >= 0; i-- {
				nodes[0].NextFocus()
				ts.Pump()
				if !nodes[i].HasPrimaryFocus() {
					t.Errorf("node %d doesn't have focus, but should", i)
				}
			}
			// Now check backwards.
			for i := 1; i < nodeCount; i++ {
				nodes[0].PreviousFocus()
				ts.Pump()
				if !nodes[i].HasPrimaryFocus() {
					t.Errorf("node %d doesn't have focus, but should", i)
				}
			}
		})
	}

	t.Run("Focus order is correct in the presence of FocusTraversalPolicyGroups", func(t *testing.T) {
		ts := gluttertest.New(t)
		const nodeCount = 10
		scopeNode := NewFocusScopeNode()
		defer scopeNode.Dispose()
		nodes, dispose := newNodes(nodeCount, nodeLabel)
		defer dispose()
		ordered := func() FocusTraversalPolicy {
			return NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{Secondary: NewWidgetOrderTraversalPolicy()})
		}
		leaf := func(order FocusOrder, i int) Widget {
			return NewFocusTraversalOrder(order, NewFocus(sizedBoxKey(nil, 10, 10), FocusOpts{FocusNode: nodes[i]}))
		}
		num := func(v float64) FocusOrder { return NewNumericFocusOrder(v) }
		lex := func(v string) FocusOrder { return NewLexicalFocusOrder(v) }
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewFocusTraversalGroup(NewRow([]Widget{
				NewFocusTraversalOrder(num(0), NewFocusTraversalGroup(
					NewRow([]Widget{leaf(num(9), 9), leaf(num(8), 8), leaf(num(7), 7)}),
					FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})),
				NewFocusTraversalOrder(num(1), NewFocusTraversalGroup(
					NewRow([]Widget{leaf(num(4), 4), leaf(num(5), 5), leaf(num(6), 6)}),
					FocusTraversalGroupOpts{Policy: ordered()})),
				NewFocusTraversalOrder(num(2), NewFocusTraversalGroup(
					NewRow([]Widget{leaf(lex("D"), 3), leaf(lex("C"), 2), leaf(lex("B"), 1), leaf(lex("A"), 0)}),
					FocusTraversalGroupOpts{Policy: ordered()})),
			}), FocusTraversalGroupOpts{Policy: ordered()}), FocusScopeOpts{Node: scopeNode}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		var order []int
		for i := 0; i < nodeCount; i++ {
			nodes[0].NextFocus()
			ts.Pump()
			order = append(order, slices.Index(nodes, ts.Binding.FocusManager().PrimaryFocus()))
		}
		if want := []int{9, 8, 7, 4, 5, 6, 0, 1, 2, 3}; !slices.Equal(order, want) {
			t.Errorf("order = %v, want %v", order, want)
		}
	})

	t.Run("Find the initial focus when a route is pushed or popped", func(t *testing.T) {
		testFindInitialFocusOnRoutePushPop(t, NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{Secondary: NewWidgetOrderTraversalPolicy()}), true)
	})

	t.Run("Custom requestFocusCallback gets called on the next/previous focus", func(t *testing.T) {
		testCustomRequestFocusCallback(t, func(cb TraversalRequestFocusCallback) FocusTraversalPolicy {
			return NewOrderedTraversalPolicy(OrderedTraversalPolicyOpts{RequestFocusCallback: cb})
		}, true)
	})
}

// trace records the onFocusChange calls of nodes, as the lists of bool? of the
// Dart tests: one character per node, 'n' for null, 'T' for true and 'F' for false.
type trace []byte

func newTrace(n int) trace {
	t := make(trace, n)
	t.clear()
	return t
}

func (t trace) clear() {
	for i := range t {
		t[i] = 'n'
	}
}

func (t trace) callback(i int) func(bool) {
	return func(v bool) {
		if v {
			t[i] = 'T'
		} else {
			t[i] = 'F'
		}
	}
}

func (t trace) expect(tb testing.TB, step, want string) {
	tb.Helper()
	if string(t) != want {
		tb.Errorf("%s: focus changes = %s, want %s", step, t, want)
	}
}

func TestDirectionalFocusTraversal(t *testing.T) {
	t.Run("Move focus in all directions", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(4)
		tr := newTrace(4)
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewRow([]Widget{
					NewFocus(sizedBoxKey(k[0]), FocusOpts{DebugLabel: "upperLeft", OnFocusChange: tr.callback(0)}),
					NewFocus(sizedBoxKey(k[1]), FocusOpts{DebugLabel: "upperRight", OnFocusChange: tr.callback(1)}),
				}),
				NewRow([]Widget{
					NewFocus(sizedBoxKey(k[2]), FocusOpts{DebugLabel: "lowerLeft", OnFocusChange: tr.callback(2)}),
					NewFocus(sizedBoxKey(k[3]), FocusOpts{DebugLabel: "lowerRight", OnFocusChange: tr.callback(3)}),
				}),
			}), FocusScopeOpts{DebugLabel: "Scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		var fn [4]*FocusNode
		for i := range fn {
			fn[i] = FocusOf(elemOf(ts, k[i]))
		}
		scope := fn[0].EnclosingScope()
		check := func(step, want string, has [4]bool) {
			t.Helper()
			tr.expect(t, step, want)
			for i := range fn {
				if fn[i].HasFocus() != has[i] {
					t.Errorf("%s: node %d has focus = %v", step, i, fn[i].HasFocus())
				}
			}
			if !scope.HasFocus() {
				t.Errorf("%s: the scope should have the focus", step)
			}
			tr.clear()
		}
		fn[0].RequestFocus()
		ts.Pump()
		check("requested", "Tnnn", [4]bool{true})
		for _, s := range []struct {
			dir  TraversalDirection
			want string
			has  [4]bool
		}{
			{TraversalDirectionRight, "FTnn", [4]bool{false, true}},
			{TraversalDirectionDown, "nFnT", [4]bool{false, false, false, true}},
			{TraversalDirectionLeft, "nnTF", [4]bool{false, false, true}},
			{TraversalDirectionUp, "TnFn", [4]bool{true}},
		} {
			if !scope.FocusInDirection(s.dir) {
				t.Errorf("FocusInDirection(%v) = false", s.dir)
			}
			ts.Pump()
			check(s.dir.String(), s.want, s.has)
		}
	})

	t.Run("Directional focus avoids hysteresis", func(t *testing.T) {
		ts := gluttertest.New(t)
		tr := newTrace(6)
		nodes := make([]*FocusNode, 6)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		makeFocus := func(i int) Widget {
			return NewFocus(sizedBoxKey(nil), FocusOpts{DebugLabel: fmt.Sprintf("[%d]", i), FocusNode: nodes[i], OnFocusChange: tr.callback(i)})
		}
		center := RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}
		// Layout is:
		//          [0]
		//       [1]   [2]
		//    [3]   [4]   [5]
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewRow([]Widget{makeFocus(0)}, center),
				NewRow([]Widget{makeFocus(1), makeFocus(2)}, center),
				NewRow([]Widget{makeFocus(3), makeFocus(4), makeFocus(5)}, center),
			}), FocusScopeOpts{DebugLabel: "Scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		scope := nodes[0].EnclosingScope()
		nodes[4].RequestFocus()
		// Test to make sure that the same path is followed backwards and forwards.
		ts.Pump()
		tr.expect(t, "requested", "nnnnTn")
		tr.clear()
		for _, s := range []struct {
			dir  TraversalDirection
			want string
		}{
			{TraversalDirectionUp, "nnTnFn"},
			{TraversalDirectionUp, "TnFnnn"},
			{TraversalDirectionDown, "FnTnnn"},
			{TraversalDirectionDown, "nnFnTn"},
			// Make sure that moving in a different axis clears the history.
			{TraversalDirectionLeft, "nnnTFn"},
			{TraversalDirectionUp, "nTnFnn"},
			{TraversalDirectionUp, "TFnnnn"},
			{TraversalDirectionDown, "FTnnnn"},
			{TraversalDirectionDown, "nFnTnn"},
		} {
			if !scope.FocusInDirection(s.dir) {
				t.Errorf("FocusInDirection(%v) = false", s.dir)
			}
			ts.Pump()
			tr.expect(t, s.dir.String(), s.want)
			tr.clear()
		}
	})

	t.Run("Directional prefers the closest node even on irregular grids", func(t *testing.T) {
		ts := gluttertest.New(t)
		const cols, rows = 3, 3
		tr := newTrace(rows * cols)
		nodes := make([]*FocusNode, rows*cols)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		makeFocus := func(row, col int) Widget {
			index := row*rows + col
			// Make some of the items a different size to test the code that
			// checks for the closest node.
			width, height := 100.0, 100.0
			if index == 3 {
				width = 150
			}
			if index == 1 {
				height = 150
			}
			return NewFocus(NewContainer(NewText(fmt.Sprintf("[%d, %d]", row, col)), ContainerOpts{Width: foundation.Some(width), Height: foundation.Some(height)}),
				FocusOpts{FocusNode: nodes[index], OnFocusChange: tr.callback(index)})
		}
		// Layout is:
		//           [0, 1]
		//    [0, 0] [    ] [0, 2]
		//    [  1,  0 ] [1, 1] [1, 2]
		//    [2, 0] [2, 1] [2, 2]
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewRow([]Widget{makeFocus(0, 0), makeFocus(0, 1), makeFocus(0, 2)},
					RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter, CrossAxisAlignment: rendering.CrossAxisAlignmentEnd}),
				NewRow([]Widget{makeFocus(1, 0), makeFocus(1, 1), makeFocus(1, 2)}, RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}),
				NewRow([]Widget{makeFocus(2, 0), makeFocus(2, 1), makeFocus(2, 2)}, RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}),
			}), FocusScopeOpts{DebugLabel: "Scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		scope := nodes[0].EnclosingScope()
		// Go down the center column and make sure that the focus stays in that
		// column, even though the second row is irregular.
		nodes[1].RequestFocus()
		ts.Pump()
		tr.expect(t, "requested", "nTnnnnnnn")
		tr.clear()
		for _, s := range []struct {
			dir   TraversalDirection
			found bool
			want  string
		}{
			{TraversalDirectionDown, true, "nFnnTnnnn"},
			{TraversalDirectionDown, true, "nnnnFnnTn"},
			{TraversalDirectionDown, false, "nnnnnnnnn"},
			// Go back up the right column and make sure that the focus stays in
			// that column, even though the second row is irregular.
			{TraversalDirectionRight, true, "nnnnnnnFT"},
			{TraversalDirectionUp, true, "nnnnnTnnF"},
			{TraversalDirectionUp, true, "nnTnnFnnn"},
			{TraversalDirectionUp, false, "nnnnnnnnn"},
			// Go left on the top row and make sure that the focus stays in that
			// row, even though the second column is irregular.
			{TraversalDirectionLeft, true, "nTFnnnnnn"},
			{TraversalDirectionLeft, true, "TFnnnnnnn"},
			{TraversalDirectionLeft, false, "nnnnnnnnn"},
		} {
			if got := scope.FocusInDirection(s.dir); got != s.found {
				t.Errorf("FocusInDirection(%v) = %v, want %v", s.dir, got, s.found)
			}
			ts.Pump()
			tr.expect(t, s.dir.String(), s.want)
			tr.clear()
		}
	})

	t.Run("Closest vertical is picked when only out of band items are considered", func(t *testing.T) {
		ts := gluttertest.New(t)
		const rows = 4
		tr := newTrace(rows)
		nodes := make([]*FocusNode, rows)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		makeFocus := func(row int) Widget {
			end := 110.0
			if row == 0 {
				end = 0
			}
			width := 100.0
			if row == 1 {
				width = 150
			}
			return NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, end, 0),
				NewFocus(NewContainer(NewText(fmt.Sprintf("[%d]", row)), ContainerOpts{Width: foundation.Some(width), Height: foundation.Some(100.0)}),
					FocusOpts{FocusNode: nodes[row], OnFocusChange: tr.callback(row)}))
		}
		// Layout is:
		//           [0]
		//    [  1]
		//     [ 2]
		//     [ 3]
		//
		// The important feature is that nothing is in the vertical band defined
		// by widget [0]. We want it to traverse to 1, 2, 3 in order, even though
		// the center of [2] is horizontally closer to the vertical axis of [0]'s
		// center than [1]'s.
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{makeFocus(0), makeFocus(1), makeFocus(2), makeFocus(3)},
				ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentEnd}), FocusScopeOpts{DebugLabel: "Scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		scope := nodes[0].EnclosingScope()
		nodes[0].RequestFocus()
		ts.Pump()
		tr.expect(t, "requested", "Tnnn")
		tr.clear()
		for _, s := range []struct {
			found bool
			want  string
		}{{true, "FTnn"}, {true, "nFTn"}, {true, "nnFT"}, {false, "nnnn"}} {
			if got := scope.FocusInDirection(TraversalDirectionDown); got != s.found {
				t.Errorf("FocusInDirection(down) = %v, want %v", got, s.found)
			}
			ts.Pump()
			tr.expect(t, "down", s.want)
			tr.clear()
		}
	})

	t.Run("Closest horizontal is picked when only out of band items are considered", func(t *testing.T) {
		ts := gluttertest.New(t)
		const cols = 4
		tr := newTrace(cols)
		nodes := make([]*FocusNode, cols)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
		}
		defer func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}()
		makeFocus := func(col int) Widget {
			top := 110.0
			if col == 0 {
				top = 0
			}
			height := 100.0
			if col == 1 {
				height = 150
			}
			return NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, top, 0, 0),
				NewFocus(NewContainer(NewText(fmt.Sprintf("[%d]", col)), ContainerOpts{Width: foundation.Some(100.0), Height: foundation.Some(height)}),
					FocusOpts{FocusNode: nodes[col], OnFocusChange: tr.callback(col)}))
		}
		// Layout is:
		//    [0]
		//        [ ][2][3]
		//        [1]
		// ([ ] is part of [1], [1] is just taller than [2] and [3]).
		//
		// The important feature is that nothing is in the horizontal band
		// defined by widget [0]. We want it to traverse to 1, 2, 3 in order,
		// even though the center of [2] is vertically closer to the horizontal
		// axis of [0]'s center than [1]'s.
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewRow([]Widget{makeFocus(0), makeFocus(1), makeFocus(2), makeFocus(3)},
				RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}), FocusScopeOpts{DebugLabel: "Scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		scope := nodes[0].EnclosingScope()
		nodes[0].RequestFocus()
		ts.Pump()
		tr.expect(t, "requested", "Tnnn")
		tr.clear()
		for _, s := range []struct {
			found bool
			want  string
		}{{true, "FTnn"}, {true, "nFTn"}, {true, "nnFT"}, {false, "nnnn"}} {
			if got := scope.FocusInDirection(TraversalDirectionRight); got != s.found {
				t.Errorf("FocusInDirection(right) = %v, want %v", got, s.found)
			}
			ts.Pump()
			tr.expect(t, "right", s.want)
			tr.clear()
		}
	})

	t.Run("Can find first focus in all directions", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(3)
		ts.PumpWidget(ltr(NewFocusTraversalGroup(
			NewFocusScope(NewColumn([]Widget{
				NewRow([]Widget{
					NewFocus(sizedBoxKey(k[0]), FocusOpts{DebugLabel: "upperLeft"}),
					NewFocus(sizedBoxKey(k[1]), FocusOpts{DebugLabel: "upperRight"}),
				}),
				NewRow([]Widget{
					NewFocus(sizedBoxKey(k[2]), FocusOpts{DebugLabel: "lowerLeft"}),
					NewFocus(sizedBoxKey(nil), FocusOpts{DebugLabel: "lowerRight"}),
				}),
			}), FocusScopeOpts{DebugLabel: "scope"}),
			FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()})))
		upperLeft, upperRight, lowerLeft := FocusOf(elemOf(ts, k[0])), FocusOf(elemOf(ts, k[1])), FocusOf(elemOf(ts, k[2]))
		scope := upperLeft.EnclosingScope()
		ts.Pump()
		policy := FocusTraversalGroupOf(k[0].CurrentContext())
		for _, s := range []struct {
			dir  TraversalDirection
			want *FocusNode
		}{
			{TraversalDirectionUp, lowerLeft},
			{TraversalDirectionDown, upperLeft},
			{TraversalDirectionLeft, upperRight},
			{TraversalDirectionRight, upperLeft},
		} {
			if got := policy.FindFirstFocusInDirection(scope.Node(), s.dir); got != s.want {
				t.Errorf("FindFirstFocusInDirection(%v) = %v, want %v", s.dir, got, s.want)
			}
		}
	})

	t.Run("Can find focus when policy data dirty", func(t *testing.T) {
		ts := gluttertest.New(t)
		focusTop, focusCenter, focusBottom := NewFocusNode(FocusNodeOpts{DebugLabel: "top"}), NewFocusNode(FocusNodeOpts{DebugLabel: "center"}), NewFocusNode(FocusNodeOpts{DebugLabel: "bottom"})
		defer focusTop.Dispose()
		defer focusCenter.Dispose()
		defer focusBottom.Dispose()
		policy := NewReadingOrderTraversalPolicy()
		box := func(n *FocusNode) Widget { return NewFocus(sizedBoxKey(nil), FocusOpts{FocusNode: n}) }
		build := func(children ...Widget) Widget {
			return NewFocusTraversalGroup(NewFocusScope(NewColumn(children), FocusScopeOpts{DebugLabel: "Scope"}), FocusTraversalGroupOpts{Policy: policy})
		}
		ts.PumpWidget(build(box(focusTop), box(focusCenter), box(focusBottom)))
		focusTop.RequestFocus()
		scope := focusTop.EnclosingScope()
		scope.FocusInDirection(TraversalDirectionDown)
		scope.FocusInDirection(TraversalDirectionDown)
		ts.Pump()
		if !focusBottom.HasFocus() {
			t.Fatal("the bottom node should have the focus")
		}
		// Remove center focus node.
		ts.PumpWidget(build(box(focusTop), box(focusBottom)))
		if !focusBottom.HasFocus() {
			t.Fatal("the bottom node should still have the focus")
		}
		scope.FocusInDirection(TraversalDirectionUp)
		ts.Pump()
		if focusCenter.HasFocus() || !focusTop.HasFocus() {
			t.Errorf("focus = center %v, top %v", focusCenter.HasFocus(), focusTop.HasFocus())
		}
	})
}

// moveFocus sends the arrow key of the direction, which the default shortcuts of
// the app map to a DirectionalFocusIntent.
func moveFocus(ts *gluttertest.Tester, dir TraversalDirection) {
	key := map[TraversalDirection]services.LogicalKeyboardKey{
		TraversalDirectionLeft: services.LogicalKeyboardKeyArrowLeft, TraversalDirectionRight: services.LogicalKeyboardKeyArrowRight,
		TraversalDirectionUp: services.LogicalKeyboardKeyArrowUp, TraversalDirectionDown: services.LogicalKeyboardKeyArrowDown,
	}[dir]
	ts.SendKeyEvent(key)
	ts.Pump()
}

func TestDirectionalFocusTraversalInScrollables(t *testing.T) {
	items := 11
	newItems := func() (nodes []*FocusNode, dispose func()) {
		nodes = make([]*FocusNode, items)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Item %d", i+1)})
		}
		return nodes, func() {
			for _, n := range nodes {
				n.Dispose()
			}
		}
	}
	for _, horizontal := range []bool{false, true} {
		name := "vertical"
		if horizontal {
			name = "horizontal"
		}
		t.Run("Focus traversal inside a "+name+" scrollable scrolls to stay visible", func(t *testing.T) {
			ts := gluttertest.New(t)
			nodes, dispose := newItems()
			defer dispose()
			first, last := NewFocusNode(FocusNodeOpts{DebugLabel: "first"}), NewFocusNode(FocusNodeOpts{DebugLabel: "last"})
			defer first.Dispose()
			defer last.Dispose()
			controller := NewScrollController()
			defer controller.Dispose()
			box := func(n *FocusNode) Widget {
				if horizontal {
					return NewFocus(NewContainer(nil, ContainerOpts{Width: foundation.Some(100.0)}), FocusOpts{FocusNode: n})
				}
				return NewFocus(NewContainer(nil, ContainerOpts{Height: foundation.Some(100.0)}), FocusOpts{FocusNode: n})
			}
			children := make([]Widget, items)
			for i := range children {
				children[i] = box(nodes[i])
			}
			forward, back := TraversalDirectionDown, TraversalDirectionUp
			// How many items fit before the scrollable has to scroll: the view is
			// 600 high (minus the header and the footer) and 800 wide (minus the
			// side nodes).
			visible, firstScroll := 3, 4
			list := NewListView(children, ListViewOpts{Controller: controller})
			var home Widget = NewColumn([]Widget{box(first), NewExpanded(list), box(last)})
			if horizontal {
				forward, back = TraversalDirectionRight, TraversalDirectionLeft
				visible, firstScroll = 5, 6
				list = NewListView(children, ListViewOpts{Controller: controller, ScrollDirection: painting.AxisHorizontal})
				home = NewRow([]Widget{box(first), NewExpanded(list), box(last)})
			}
			ts.PumpWidget(navTestApp(navTestAppOpts{home: home}))
			offset := func() float64 { return controller.Offset() }
			wantOffset := func(step string, want float64) {
				t.Helper()
				if offset() != want {
					t.Errorf("%s: offset = %v, want %v", step, offset(), want)
				}
			}

			// Start at the top.
			wantOffset("start", 0)
			moveFocus(ts, forward)
			if !first.HasPrimaryFocus() {
				t.Fatal("the first node should have the primary focus")
			}
			wantOffset("first", 0)
			// Enter the list.
			moveFocus(ts, forward)
			if !nodes[0].HasPrimaryFocus() {
				t.Fatal("item 1 should have the primary focus")
			}
			wantOffset("enter", 0)
			// Go down until we hit the bottom of the visible area.
			for i := 1; i <= visible; i++ {
				moveFocus(ts, forward)
				wantOffset(fmt.Sprintf("focusing item %d", i), 0)
			}
			// Now keep going down, and the scrollable should scroll automatically.
			lowest := 0.0
			for i := firstScroll; i <= 10; i++ {
				moveFocus(ts, forward)
				lowest = 100.0 * float64(i-5)
				if !horizontal {
					lowest += 200
				}
				wantOffset(fmt.Sprintf("focusing item %d", i), lowest)
			}
			// Now go one more, and see that the footer gets focused.
			moveFocus(ts, forward)
			if !last.HasPrimaryFocus() {
				t.Fatal("the last node should have the primary focus")
			}
			wantOffset("last", lowest)
			moveFocus(ts, back)
			if !nodes[10].HasPrimaryFocus() {
				t.Fatal("item 11 should have the primary focus")
			}
			wantOffset("back to item 11", lowest)
			// Now reverse directions and go back to the top.
			// These should not cause a scroll.
			noScrollBack := 8
			if horizontal {
				noScrollBack = 7
			}
			for i := 10; i >= noScrollBack; i-- {
				moveFocus(ts, back)
				wantOffset(fmt.Sprintf("focusing item %d", i), lowest)
			}
			// These should all cause a scroll.
			for i := noScrollBack - 1; i >= 1; i-- {
				moveFocus(ts, back)
				wantOffset(fmt.Sprintf("focusing item %d", i), 100.0*float64(i-1))
			}
			// Back at the top.
			if !nodes[0].HasPrimaryFocus() {
				t.Fatal("item 1 should have the primary focus")
			}
			wantOffset("top", 0)
			// Now we jump to the header.
			moveFocus(ts, back)
			if !first.HasPrimaryFocus() {
				t.Error("the first node should have the primary focus")
			}
			wantOffset("header", 0)
		})
	}
}

func TestDirectionalFocusTraversalWithNestedScrollables(t *testing.T) {
	// A grid of scrollables in a scrollable of the other axis: the focus moves
	// along the cross axis of the outer one without leaving the cell's column
	// (or row), and the outer scrollable scrolls to keep the focus visible. The
	// view is 800x600.
	for _, horizontalOuter := range []bool{false, true} {
		name := "horizontal scrollables inside a vertical scrollable handles vertical navigation correctly"
		if horizontalOuter {
			name = "vertical scrollables inside a horizontal scrollable handles horizontal navigation correctly"
		}
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t)
			const cell = 100.0
			count, perLine := 10, 5
			if horizontalOuter {
				perLine = 10
			}
			sticky := NewFocusNode(FocusNodeOpts{DebugLabel: "Sticky Button"})
			defer sticky.Dispose()
			grid := make([][]*FocusNode, count)
			for i := range grid {
				grid[i] = make([]*FocusNode, perLine)
				for j := range grid[i] {
					grid[i][j] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Button %d-%d", i, j)})
				}
			}
			defer func() {
				for _, line := range grid {
					for _, n := range line {
						n.Dispose()
					}
				}
			}()
			outerController := NewScrollController()
			defer outerController.Dispose()
			innerControllers := make([]ScrollController, count)
			for i := range innerControllers {
				innerControllers[i] = NewScrollController()
				defer innerControllers[i].Dispose()
			}
			innerAxis, outerAxis := painting.AxisHorizontal, painting.AxisVertical
			forward, back := TraversalDirectionDown, TraversalDirectionUp
			if horizontalOuter {
				innerAxis, outerAxis = painting.AxisVertical, painting.AxisHorizontal
				forward, back = TraversalDirectionRight, TraversalDirectionLeft
			}
			item := func(n *FocusNode) Widget {
				return NewFocus(NewContainer(nil, ContainerOpts{Width: foundation.Some(cell), Height: foundation.Some(cell)}), FocusOpts{FocusNode: n})
			}
			outer := NewListViewSeparated(func(bc BuildContext, line int) Widget {
				inner := NewListViewBuilder(func(bc BuildContext, i int) Widget { return item(grid[line][i]) },
					ListViewOpts{Controller: innerControllers[line], ScrollDirection: innerAxis, ItemCount: foundation.Some(perLine)})
				if horizontalOuter {
					return NewSizedBox(inner, SizedBoxOpts{Width: foundation.Some(cell)})
				}
				return NewSizedBox(inner, SizedBoxOpts{Height: foundation.Some(cell)})
			}, func(bc BuildContext, i int) Widget {
				if horizontalOuter {
					return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(32.0)})
				}
				return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(32.0)})
			}, count, ListViewOpts{Controller: outerController, ScrollDirection: outerAxis})
			stickyBox := NewFocus(NewContainer(nil, ContainerOpts{Height: foundation.Some(cell)}), FocusOpts{FocusNode: sticky})
			var home Widget = NewColumn([]Widget{stickyBox, NewExpanded(outer)})
			if horizontalOuter {
				stickyBox = NewFocus(NewContainer(nil, ContainerOpts{Width: foundation.Some(cell)}), FocusOpts{FocusNode: sticky})
				home = NewRow([]Widget{stickyBox, NewExpanded(outer)})
			}
			ts.PumpWidget(navTestApp(navTestAppOpts{home: home}))

			// Start by focusing the sticky button.
			sticky.RequestFocus()
			ts.Pump()
			if !sticky.HasPrimaryFocus() {
				t.Fatal("the sticky button should have the primary focus")
			}
			// Navigate to the first line - should focus one of its widgets.
			moveFocus(ts, forward)
			focused := -1
			for i, n := range grid[0] {
				if n.HasPrimaryFocus() {
					focused = i
					break
				}
			}
			if focused < 0 {
				t.Fatal("nothing in the first line is focused")
			}
			// Navigate through the lines.
			scrollFrom := 5
			if horizontalOuter {
				scrollFrom = 6
			}
			for line := 1; line < count; line++ {
				moveFocus(ts, forward)
				if !grid[line][focused].HasPrimaryFocus() {
					t.Fatalf("line %d: button %d should have the primary focus", line, focused)
				}
				if line >= scrollFrom && outerController.Offset() <= 0 {
					t.Errorf("line %d: the outer scrollable should have scrolled", line)
				}
			}
			// Navigate back - should go to the previous lines, not to the sticky button.
			for line := count - 2; line >= 0; line-- {
				moveFocus(ts, back)
				if !grid[line][focused].HasPrimaryFocus() {
					t.Fatalf("line %d: button %d should have the primary focus going back", line, focused)
				}
			}
			// Only now should we reach the sticky button.
			moveFocus(ts, back)
			if !sticky.HasPrimaryFocus() {
				t.Error("the sticky button should have the primary focus")
			}
		})
	}
}

func TestFocusTraversalDoesNotThrowWhenNoFocusableIsAvailableInAGroup(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewSizedBoxExpand(NewCenter(navText("title")))}))
	initialFocus := ts.Binding.FocusManager().PrimaryFocus()
	// The Tab key runs NextFocusAction, which moves the primary focus to the next.
	if primary := initialFocus; primary != nil {
		primary.NextFocus()
	}
	ts.Pump()
	if got := ts.Binding.FocusManager().PrimaryFocus(); got != initialFocus {
		t.Errorf("primary focus = %v, want %v", got, initialFocus)
	}
}

func TestCustomRequestFocusCallbackIsCalledOnFocusInDirection(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	testNode1 := NewFocusNode(FocusNodeOpts{DebugLabel: "Focus Node"})
	defer testNode1.Dispose()
	called := false
	ts.PumpWidget(NewFocusTraversalGroup(
		NewFocusScope(NewFocus(NewContainer(nil), FocusOpts{Key: key1, FocusNode: testNode1}), FocusScopeOpts{DebugLabel: "key1"}),
		FocusTraversalGroupOpts{Policy: NewReadingOrderTraversalPolicy(FocusTraversalPolicyOpts{
			RequestFocusCallback: func(*FocusNode, TraversalRequestFocusOpts) { called = true }})}))
	scope := FocusScopeOf(elemOf(ts, key1))
	for _, dir := range []TraversalDirection{TraversalDirectionUp, TraversalDirectionDown, TraversalDirectionLeft, TraversalDirectionRight} {
		called = false
		scope.FocusInDirection(dir)
		ts.Pump()
		if !called {
			t.Errorf("the callback was not called for %v", dir)
		}
	}
}

// focusTestButton is TestButton of button_tester.dart: a focusable, tappable widget
// that can not take the focus when it is disabled (onPressed is nil).
func focusTestButton(node *FocusNode, label string, onPressed func(), autofocus ...bool) Widget {
	return newTestButton(navText(label), testButtonOpts{focusNode: node, autofocus: len(autofocus) > 0 && autofocus[0], onPressed: onPressed})
}

func TestFocusTraversalGroup(t *testing.T) {
	t.Run("Descendants of FocusTraversalGroup aren't focusable if descendantsAreFocusable is false", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(3)
		focusNode := NewFocusNode()
		defer focusNode.Dispose()
		var gotFocus *bool
		ts.PumpWidget(NewFocusTraversalGroup(
			NewFocus(NewFocus(NewContainer(nil, ContainerOpts{Key: k[2]}), FocusOpts{Key: k[1], FocusNode: focusNode}),
				FocusOpts{OnFocusChange: func(f bool) { gotFocus = &f }}),
			FocusTraversalGroupOpts{DescendantsAreNotFocusable: true}))
		unfocusableNode, containerNode := FocusOf(elemOf(ts, k[1])), FocusOf(elemOf(ts, k[2]))
		for _, n := range []*FocusNode{unfocusableNode, containerNode} {
			n.RequestFocus()
			ts.Pump()
			if gotFocus != nil || containerNode.HasFocus() || unfocusableNode.HasFocus() {
				t.Fatal("no node should have the focus")
			}
		}
	})

	t.Run("Group applies correct policy if focus tree is different from widget tree", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(5)
		focusNode, parentFocusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "child"}), NewFocusNode(FocusNodeOpts{DebugLabel: "parent"})
		defer focusNode.Dispose()
		defer parentFocusNode.Dispose()
		ts.PumpWidget(NewColumn([]Widget{
			NewFocusTraversalGroup(
				NewFocus(
					// This makes focusNode be a child of parentFocusNode instead of
					// the surrounding Focus.
					NewFocusWithExternalFocusNode(focusNode, NewContainer(nil, ContainerOpts{Key: k[2]}),
						FocusWithExternalFocusNodeOpts{Key: k[1], ParentNode: parentFocusNode})),
				FocusTraversalGroupOpts{Policy: NewWidgetOrderTraversalPolicy()}),
			NewFocusTraversalGroup(
				NewFocusScope(NewFocusWithExternalFocusNode(parentFocusNode, NewContainer(nil, ContainerOpts{Key: k[4]}), FocusWithExternalFocusNodeOpts{Key: k[3]})),
				FocusTraversalGroupOpts{Policy: newSkipAllButFirstAndLastPolicy()}),
		}))
		if focusNode.Parent() != parentFocusNode {
			t.Errorf("parent = %v, want %v", focusNode.Parent(), parentFocusNode)
		}
		if _, ok := FocusTraversalGroupMaybeOf(k[2].CurrentContext()).(*skipAllButFirstAndLastPolicy); !ok {
			t.Errorf("MaybeOf policy = %T", FocusTraversalGroupMaybeOf(k[2].CurrentContext()))
		}
		if _, ok := FocusTraversalGroupOf(k[2].CurrentContext()).(*skipAllButFirstAndLastPolicy); !ok {
			t.Errorf("Of policy = %T", FocusTraversalGroupOf(k[2].CurrentContext()))
		}
	})

	t.Run("Group parentNode attaches the group to the given node instead of the enclosing scope", func(t *testing.T) {
		ts := gluttertest.New(t)
		parentNode := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "parent"}})
		childNode := NewFocusNode(FocusNodeOpts{DebugLabel: "child"})
		defer parentNode.Dispose()
		defer childNode.Dispose()
		root := ts.Binding.FocusManager().RootScope()
		ts.PumpWidget(NewFocusScopeWithExternalFocusNode(
			parentNode,
			// Without parentNode, this group would attach under the enclosing
			// FocusScope above. parentNode reparents it to the root scope.
			NewFocusScope(NewFocusTraversalGroup(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: childNode}),
				FocusTraversalGroupOpts{ParentNode: root.Node()}))))
		// The group's node (the parent of childNode) is attached to the root
		// scope, not to the enclosing parentNode.
		groupNode := childNode.Parent()
		if groupNode.Parent() != root.Node() {
			t.Errorf("the group is attached to %v, want the root scope", groupNode.Parent())
		}
		if slices.Contains(parentNode.Descendants(), childNode) {
			t.Error("the child is a descendant of parentNode")
		}
	})

	for _, tc := range []struct {
		name  string
		group FocusTraversalGroupOpts
		skip  bool
	}{
		{"Descendants of FocusTraversalGroup aren't traversable if descendantsAreTraversable is false", FocusTraversalGroupOpts{DescendantsAreNotTraversable: true}, false},
		{"FocusTraversalGroup with skipTraversal for all descendants set to true doesn't cause an exception", FocusTraversalGroupOpts{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			node1, node2 := NewFocusNode(), NewFocusNode()
			defer node1.Dispose()
			defer node2.Dispose()
			skip := foundation.Some(tc.skip)
			ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
				NewFocus(NewContainer(nil), FocusOpts{FocusNode: node1, SkipTraversal: skip}),
				NewFocus(NewContainer(nil), FocusOpts{FocusNode: node2, SkipTraversal: skip}),
			}), tc.group))
			node1.RequestFocus()
			ts.Pump()
			if !node1.HasPrimaryFocus() || node2.HasPrimaryFocus() {
				t.Fatal("node 1 should have the primary focus")
			}
			if ts.Binding.FocusManager().PrimaryFocus().NextFocus() {
				t.Error("NextFocus found a node")
			}
			ts.Pump()
			if !node1.HasPrimaryFocus() || node2.HasPrimaryFocus() {
				t.Error("node 1 should still have the primary focus")
			}
		})
	}

	t.Run("Nested FocusTraversalGroup with unfocusable children doesn't assert", func(t *testing.T) {
		ts := gluttertest.New(t)
		k := newKeys(3)
		focusNode := NewFocusNode()
		defer focusNode.Dispose()
		var gotFocus *bool
		ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
			NewFocus(NewContainer(nil), FocusOpts{Autofocus: true}),
			NewFocusTraversalGroup(
				NewFocus(NewFocus(NewContainer(nil, ContainerOpts{Key: k[2]}), FocusOpts{Key: k[1], FocusNode: focusNode}),
					FocusOpts{OnFocusChange: func(f bool) { gotFocus = &f }}),
				FocusTraversalGroupOpts{DescendantsAreNotFocusable: true}),
		})))
		unfocusableNode, containerNode := FocusOf(elemOf(ts, k[1])), FocusOf(elemOf(ts, k[2]))
		ts.Pump()
		ts.Binding.FocusManager().PrimaryFocus().NextFocus()
		if gotFocus != nil || containerNode.HasFocus() || unfocusableNode.HasFocus() {
			t.Fatal("no node should have the focus after nextFocus")
		}
		containerNode.RequestFocus()
		ts.Pump()
		if gotFocus != nil || containerNode.HasFocus() || unfocusableNode.HasFocus() {
			t.Error("no node should have the focus after the request")
		}
	})

	t.Run("Empty FocusTraversalGroup doesn't cause an exception", func(t *testing.T) {
		ts := gluttertest.New(t)
		focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node"})
		defer focusNode.Dispose()
		ts.PumpWidget(NewFocusTraversalGroup(NewDirectionality(ui.TextDirectionRTL, NewColumn([]Widget{
			NewFocusTraversalGroup(NewContainer(nil)),
			NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNode, Autofocus: true}),
		}))))
		ts.Pump()
		ts.Binding.FocusManager().PrimaryFocus().NextFocus()
		ts.Pump()
		if ts.Binding.FocusManager().PrimaryFocus() != focusNode {
			t.Errorf("primary focus = %v", ts.Binding.FocusManager().PrimaryFocus())
		}
	})

	t.Run("Focus traversal group doesn't introduce a Semantics node", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewFocusTraversalGroup(NewContainer(nil)))
		if n := ts.Count(gluttertest.Descendant(gluttertest.ByType[*FocusTraversalGroup](), gluttertest.ByType[*Semantics]())); n != 0 {
			t.Errorf("found %d Semantics widgets", n)
		}
	})

	t.Run("onFocusNodeCreated is called when the FocusTraversalGroup is built", func(t *testing.T) {
		ts := gluttertest.New(t)
		var node *FocusNode
		ts.PumpWidget(NewFocusTraversalGroup(NewSizedBox(nil), FocusTraversalGroupOpts{OnFocusNodeCreated: func(n *FocusNode) { node = n }}))
		ts.PumpAndSettle()
		if node == nil {
			t.Error("OnFocusNodeCreated was not called")
		}
	})
}

func TestExcludeFocusTraversal(t *testing.T) {
	t.Run("Descendants aren't traversable", func(t *testing.T) {
		ts := gluttertest.New(t)
		nodes := make([]*FocusNode, 4)
		for i := range nodes {
			nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("node %d", i+1)})
			defer nodes[i].Dispose()
		}
		ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
			NewFocus(NewContainer(nil), FocusOpts{Autofocus: true, FocusNode: nodes[0]}),
			NewExcludeFocusTraversal(NewFocus(NewFocus(NewContainer(nil), FocusOpts{FocusNode: nodes[2]}), FocusOpts{FocusNode: nodes[1]})),
			NewFocus(NewContainer(nil), FocusOpts{FocusNode: nodes[3]}),
		})))
		ts.Pump()
		primary := func() []bool {
			out := make([]bool, len(nodes))
			for i, n := range nodes {
				out[i] = n.HasPrimaryFocus()
			}
			return out
		}
		if got, want := primary(), []bool{true, false, false, false}; !slices.Equal(got, want) {
			t.Fatalf("primary focus = %v, want %v", got, want)
		}
		nodes[0].NextFocus()
		ts.Pump()
		if got, want := primary(), []bool{false, false, false, true}; !slices.Equal(got, want) {
			t.Errorf("primary focus = %v, want %v", got, want)
		}
	})

	t.Run("Doesn't introduce a Semantics node", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewExcludeFocusTraversal(NewContainer(nil)))
		if n := ts.Count(gluttertest.Descendant(gluttertest.ByType[*ExcludeFocusTraversal](), gluttertest.ByType[*Semantics]())); n != 0 {
			t.Errorf("found %d Semantics widgets", n)
		}
	})
}

// nextFocus and previousFocus invoke the NextFocusIntent and the PreviousFocusIntent
// at the primary focus, and return what the action returned: whether the focus
// moved.
func nextFocus(ts *gluttertest.Tester) bool {
	result := ActionsInvoke(ts.Binding.FocusManager().PrimaryFocus().Context(), NextFocusIntent{}).(bool)
	ts.Pump()
	return result
}

func previousFocus(ts *gluttertest.Tester) bool {
	result := ActionsInvoke(ts.Binding.FocusManager().PrimaryFocus().Context(), PreviousFocusIntent{}).(bool)
	ts.Pump()
	return result
}

func TestDefaultRouteEdgeTraversalBehavior(t *testing.T) {
	ts := gluttertest.New(t)
	nodeA, nodeB := NewFocusNode(), NewFocusNode()
	defer nodeA.Dispose()
	defer nodeB.Dispose()
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewColumn([]Widget{
		focusTestButton(nodeA, "A", func() {}), focusTestButton(nodeB, "B", func() {}),
	})}))
	nodeA.RequestFocus()
	ts.Pump()
	check := func(step string, a, b bool) {
		t.Helper()
		if nodeA.HasFocus() != a || nodeB.HasFocus() != b {
			t.Errorf("%s: focus = %v %v, want %v %v", step, nodeA.HasFocus(), nodeB.HasFocus(), a, b)
		}
	}
	check("start", true, false)
	expect := func(step string, got, want bool) {
		t.Helper()
		if got != want {
			t.Errorf("%s: result = %v, want %v", step, got, want)
		}
	}
	// A -> B
	expect("A -> B", nextFocus(ts), true)
	check("A -> B", false, true)
	// A <- B
	expect("A <- B", previousFocus(ts), true)
	check("A <- B", true, false)
	// A -> B
	expect("A -> B again", nextFocus(ts), true)
	check("A -> B again", false, true)
	// B -> cycle back to A (on the web the focus would escape the app)
	expect("B -> A", nextFocus(ts), true)
	check("B -> A", true, false)
	// Start with A again, but wrap around in the opposite direction.
	nodeA.RequestFocus()
	ts.Pump()
	expect("A <- B (wrap)", previousFocus(ts), true)
	check("A <- B (wrap)", false, true)
}

func TestFocusScopeNodeTraversalEdgeBehaviorTakesEffectAfterUpdate(t *testing.T) {
	ts := gluttertest.New(t)
	scope := NewFocusScopeNode()
	defer scope.Dispose()
	if scope.TraversalEdgeBehavior() != TraversalEdgeBehaviorClosedLoop {
		t.Fatalf("default behavior = %v", scope.TraversalEdgeBehavior())
	}
	nodeA, nodeB := NewFocusNode(), NewFocusNode()
	defer nodeA.Dispose()
	defer nodeB.Dispose()
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewFocus(NewColumn([]Widget{
		focusTestButton(nodeA, "A", func() {}), focusTestButton(nodeB, "B", func() {}),
	}), FocusOpts{FocusNode: &scope.FocusNode})}))
	nodeA.RequestFocus()
	ts.Pump()
	check := func(step string, a, b bool) {
		t.Helper()
		if nodeA.HasFocus() != a || nodeB.HasFocus() != b {
			t.Errorf("%s: focus = %v %v, want %v %v", step, nodeA.HasFocus(), nodeB.HasFocus(), a, b)
		}
	}
	expect := func(step string, got, want bool) {
		t.Helper()
		if got != want {
			t.Errorf("%s: result = %v, want %v", step, got, want)
		}
	}
	check("start", true, false)
	expect("A -> B", nextFocus(ts), true)
	check("A -> B", false, true)
	// A <- B (wrap around)
	expect("B -> A", nextFocus(ts), true)
	check("B -> A", true, false)

	// Change the behavior and verify that the new behavior is in effect.
	scope.SetTraversalEdgeBehavior(TraversalEdgeBehaviorLeaveFlutterView)
	expect("A -> B", nextFocus(ts), true)
	check("A -> B (leave)", false, true)
	// B -> escape the view
	expect("B -> out", nextFocus(ts), false)
	check("B -> out", false, false)

	scope.SetTraversalEdgeBehavior(TraversalEdgeBehaviorStop)
	// B -> A, but stop at the edge
	nodeB.RequestFocus()
	ts.Pump()
	check("B requested", false, true)
	expect("B -> stop", nextFocus(ts), false)
	check("B -> stop", false, true)

	// Change the behavior back to closedLoop and verify it's in effect. Also,
	// this time traverse in the opposite direction.
	nodeA.RequestFocus()
	ts.Pump()
	check("A requested", true, false)
	scope.SetTraversalEdgeBehavior(TraversalEdgeBehaviorClosedLoop)
	expect("A <- B (wrap)", previousFocus(ts), true)
	check("A <- B (wrap)", false, true)
}

func TestEdgeCasesForInDirection(t *testing.T) {
	ts := gluttertest.New(t)
	tr := newTrace(6)
	nodes := make([]*FocusNode, 6)
	for i := range nodes {
		nodes[i] = NewFocusNode(FocusNodeOpts{DebugLabel: fmt.Sprintf("Node %d", i)})
	}
	childScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Child Scope"}})
	defer func() {
		for _, n := range nodes {
			n.Dispose()
		}
		childScope.Dispose()
	}()
	makeFocus := func(i int) Widget {
		return NewFocus(sizedBoxKey(nil), FocusOpts{DebugLabel: fmt.Sprintf("[%d]", i), FocusNode: nodes[i], OnFocusChange: tr.callback(i)})
	}
	// Layout is:
	//          [0]
	// ---------Child FocusScope---------
	//          [1]
	//          [2]
	//              [3]
	// ---------Child FocusScope End---------
	//          [4] [5]
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewColumn([]Widget{
		NewRow([]Widget{makeFocus(0)}, RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}),
		NewFocusScope(NewColumn([]Widget{
			makeFocus(1),
			NewRow([]Widget{
				makeFocus(2),
				NewPadding(painting.NewEdgeInsetsFromLTRB(0, 100, 0, 0), makeFocus(3)),
			}, RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}),
		}, ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}), FocusScopeOpts{Node: childScope}),
		NewRow([]Widget{makeFocus(4), makeFocus(5)}),
	}, ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})}))
	resetTo := func(i int) {
		nodes[i].RequestFocus()
		ts.Pump()
		tr.clear()
	}
	for _, tc := range []struct {
		behavior   TraversalEdgeBehavior
		down, up   string
		downResult string
	}{
		// focus should not change
		{TraversalEdgeBehaviorStop, "nnnnnn", "nnnnnn", ""},
		// focus should change in a loop
		{TraversalEdgeBehaviorClosedLoop, "nTnFnn", "nFTnnn", ""},
		// focus can change to the parent scope
		{TraversalEdgeBehaviorParentScope, "nnnFnT", "TFnnnn", ""},
		// focus will be lost
		{TraversalEdgeBehaviorLeaveFlutterView, "nnnFnn", "nFnnnn", ""},
	} {
		childScope.SetDirectionalTraversalEdgeBehavior(tc.behavior)
		resetTo(3)
		moveFocus(ts, TraversalDirectionDown)
		tr.expect(t, tc.behavior.String()+" down", tc.down)
		tr.clear()
		resetTo(1)
		moveFocus(ts, TraversalDirectionUp)
		tr.expect(t, tc.behavior.String()+" up", tc.up)
		tr.clear()
	}
}

func TestWhenThereIsNoFocusedNodeTheFocusCanBeSetToTheFocusScopeNode(t *testing.T) {
	ts := gluttertest.New(t)
	scope, childScope, nodeA := NewFocusScopeNode(), NewFocusScopeNode(), NewFocusNode()
	defer scope.Dispose()
	defer childScope.Dispose()
	defer nodeA.Dispose()
	ts.PumpWidget(ltr(NewFocusScope(
		NewFocusScope(NewPadding(painting.NewEdgeInsetsFromLTRB(0, 10, 0, 0),
			NewFocus(NewText("A"), FocusOpts{FocusNode: nodeA})), FocusScopeOpts{Node: childScope}),
		FocusScopeOpts{Node: scope})))
	if !scope.FocusInDirection(TraversalDirectionDown) {
		t.Fatal("FocusInDirection(down) = false")
	}
	ts.Pump()
	if !childScope.HasFocus() || nodeA.HasFocus() {
		t.Errorf("focus = child scope %v, node A %v", childScope.HasFocus(), nodeA.HasFocus())
	}
}

func TestFocusScopeWithNoFocusableDescendantsIsSkippedByTab(t *testing.T) {
	ts := gluttertest.New(t)
	enabled1, enabled2 := NewFocusNode(), NewFocusNode()
	defer enabled1.Dispose()
	defer enabled2.Dispose()
	ts.PumpWidget(navTestApp(navTestAppOpts{home: NewCenter(NewColumn([]Widget{
		focusTestButton(enabled1, "Enabled Button 1", func() {}),
		NewFocusTraversalGroup(NewColumn([]Widget{
			focusTestButton(nil, "Disabled Button 1", nil),
			NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(16.0)}),
			focusTestButton(nil, "Disabled Button 2", nil),
		}, ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})),
		focusTestButton(enabled2, "Enabled Button 2", func() {}),
	}, ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}))}))
	nextFocus(ts)
	ts.PumpAndSettle()
	if !enabled1.HasPrimaryFocus() {
		t.Fatal("enabled button 1 should have the primary focus")
	}
	nextFocus(ts)
	ts.PumpAndSettle()
	if !enabled2.HasPrimaryFocus() {
		t.Error("enabled button 2 should have the primary focus")
	}
}

func TestFocusTraversalDoesNotBreakWhenNoFocusableIsAvailableOnAnApp(t *testing.T) {
	for name, home := range map[string]Widget{
		"container":   NewContainer(nil),
		"placeholder": NewPlaceholder(nil),
	} {
		t.Run(name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var events []services.KeyEvent
			ts.PumpWidget(navTestApp(navTestAppOpts{home: home}))
			handle := ts.Binding.Keyboard().AddHandler(func(event services.KeyEvent) bool {
				events = append(events, event)
				return true
			})
			defer ts.Binding.Keyboard().RemoveHandler(handle)
			ts.SendKeyEvent(services.LogicalKeyboardKeyArrowRight)
			if len(events) != 2 {
				t.Errorf("got %d key events, want 2", len(events))
			}
		})
	}
}

func TestFocusTraversalActionsAreInvokedWhenShortcutsAreUsed(t *testing.T) {
	ts := gluttertest.New(t)
	k := newKeys(4)
	upperLeft, upperRight, lowerLeft, lowerRight := k[0], k[1], k[2], k[3]
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: ltr(NewFocusScope(NewColumn([]Widget{
		NewRow([]Widget{
			NewFocus(sizedBoxKey(upperLeft), FocusOpts{Autofocus: true, DebugLabel: "upperLeft"}),
			NewFocus(sizedBoxKey(upperRight), FocusOpts{DebugLabel: "upperRight"}),
		}),
		NewRow([]Widget{
			NewFocus(sizedBoxKey(lowerLeft), FocusOpts{DebugLabel: "lowerLeft"}),
			NewFocus(sizedBoxKey(lowerRight), FocusOpts{DebugLabel: "lowerRight"}),
		}),
	}), FocusScopeOpts{DebugLabel: "scope"}))}))
	hasPrimary := func(k gkey) bool { return FocusOf(elemOf(ts, k)).HasPrimaryFocus() }
	requireFocus := func(step string, k gkey) {
		t.Helper()
		if !hasPrimary(k) {
			t.Fatalf("%s: the wrong node has the primary focus: %s", step, ts.Binding.FocusManager().PrimaryFocus().ToStringShort())
		}
	}
	shiftTab := func() {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyShift)
		ts.SendKeyEvent(services.LogicalKeyboardKeyTab)
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyShift)
	}
	requireFocus("start", upperLeft)
	for i, want := range []gkey{upperRight, lowerLeft, lowerRight, upperLeft} {
		ts.SendKeyEvent(services.LogicalKeyboardKeyTab)
		requireFocus("tab "+string(rune('1'+i)), want)
	}
	for i, want := range []gkey{lowerRight, lowerLeft, upperRight, upperLeft} {
		shiftTab()
		requireFocus("shift+tab "+string(rune('1'+i)), want)
	}
	// Traverse in a direction.
	for _, step := range []struct {
		key  services.LogicalKeyboardKey
		want gkey
	}{
		{services.LogicalKeyboardKeyArrowRight, upperRight}, {services.LogicalKeyboardKeyArrowDown, lowerRight},
		{services.LogicalKeyboardKeyArrowLeft, lowerLeft}, {services.LogicalKeyboardKeyArrowUp, upperLeft},
	} {
		ts.SendKeyEvent(step.key)
		requireFocus(step.key.String(), step.want)
	}
}

func TestFocusTraversalActionsWorkWhenCurrentFocusSkipsTraversal(t *testing.T) {
	ts := gluttertest.New(t)
	k := newKeys(3)
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: ltr(NewFocusScope(NewColumn([]Widget{
		NewRow([]Widget{
			NewFocus(sizedBoxKey(k[0]), FocusOpts{Autofocus: true, SkipTraversal: foundation.Some(true), DebugLabel: "1"}),
			NewFocus(sizedBoxKey(k[1]), FocusOpts{DebugLabel: "2"}),
			NewFocus(sizedBoxKey(k[2]), FocusOpts{DebugLabel: "3"}),
		}),
	}), FocusScopeOpts{DebugLabel: "scope"}))}))
	hasPrimary := func(i int) bool { return FocusOf(elemOf(ts, k[i])).HasPrimaryFocus() }
	if !hasPrimary(0) {
		t.Fatal("the first node should have the primary focus")
	}
	// The first node is skipped from then on, because it skips traversal.
	for i, want := range []int{1, 2, 1, 2} {
		ts.SendKeyEvent(services.LogicalKeyboardKeyTab)
		if !hasPrimary(want) {
			t.Fatalf("tab %d: node %d should have the primary focus", i+1, want+1)
		}
	}
}

func TestNextAndPreviousFocusActionsConvertTheInvokeResultToKeyEventResult(t *testing.T) {
	for name, action := range map[string]Action{
		"NextFocusAction":     NewNextFocusAction(),
		"PreviousFocusAction": NewPreviousFocusAction(),
	} {
		intent := Intent(NextFocusIntent{})
		if name == "PreviousFocusAction" {
			intent = PreviousFocusIntent{}
		}
		if got := action.ToKeyEventResult(intent, true); got != KeyEventResultHandled {
			t.Errorf("%s: true → %v, want handled", name, got)
		}
		if got := action.ToKeyEventResult(intent, false); got != KeyEventResultSkipRemainingHandlers {
			t.Errorf("%s: false → %v, want skipRemainingHandlers", name, got)
		}
	}
}

func TestRequestFocusActionCallsTheRequestFocusCallbackOfTheIntent(t *testing.T) {
	ts := gluttertest.New(t)
	nodeA := NewFocusNode()
	defer nodeA.Dispose()
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewSingleChildScrollView(
		newTestButton(navText("A"), testButtonOpts{focusNode: nodeA, onPressed: func() {}}))}))
	action := NewRequestFocusAction()
	action.Invoke(RequestFocusIntent{FocusNode: nodeA})
	ts.Pump()
	if !nodeA.HasFocus() {
		t.Fatal("the action should have given the focus to the node")
	}
	nodeA.Unfocus()
	ts.Pump()
	if nodeA.HasFocus() {
		t.Fatal("the node should have lost the focus")
	}
	called := false
	action.Invoke(RequestFocusIntent{FocusNode: nodeA, RequestFocusCallback: func(*FocusNode, TraversalRequestFocusOpts) { called = true }})
	ts.Pump()
	if !called {
		t.Error("the callback of the intent was not called")
	}
}
