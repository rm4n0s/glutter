package widgets_test

import (
	"github.com/rm4n0s/glutter/services"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/focus_scope_test.dart.
// Not ported: the cases of the deprecated onKey (RawKeyEvent), the ones that use
// a FocusNode subclass (TestExternalFocusNode), and the semantics tester cases
// that wait for the semantics tester.

// testFocus is TestFocus: a widget that shows its name, in capitals when its
// node has the focus, and requests the focus when it is tapped.
type testFocus struct {
	StatefulWidget
	debugLabel string
	name       string
	autofocus  bool
	parentNode *FocusNode
}

type testFocusOpts struct {
	debugLabel string
	name       string
	autofocus  bool
	parentNode *FocusNode
}

func newTestFocus(key foundation.Key, opts ...testFocusOpts) *testFocus {
	var o testFocusOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.name == "" {
		o.name = "a"
	}
	return &testFocus{NewStatefulWidgetBase(WidgetOpts{Key: key}), o.debugLabel, o.name, o.autofocus, o.parentNode}
}

func (w *testFocus) CreateState() State { return &testFocusState{} }

type testFocusState struct {
	StateOf[*testFocus]
	focusNode *FocusNode
	label     string
	built     bool
}

func (s *testFocusState) currentLabel() string {
	if s.focusNode.HasFocus() {
		return upper(s.Widget().name) + " FOCUSED"
	}
	return s.Widget().name
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func (s *testFocusState) Dispose() { s.focusNode.Dispose() }

func (s *testFocusState) InitState() {
	s.focusNode = NewFocusNode(FocusNodeOpts{DebugLabel: s.Widget().debugLabel})
	s.label = s.currentLabel()
	s.focusNode.AddListener(func() { s.SetState(func() { s.label = s.currentLabel() }) })
}

func (s *testFocusState) Build(bc BuildContext) Widget {
	s.built = true
	w := s.Widget()
	return NewGestureDetector(
		// The test text engine has no fonts, so a text is 0 pixels wide and cannot be
		// hit: the SizedBox gives it a size, as the Ahem font does in Flutter.
		NewFocus(NewSizedBox(NewText(s.label, TextOpts{TextDirection: ui.TextDirectionLTR}),
			SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)}), FocusOpts{
			Autofocus: w.autofocus, FocusNode: s.focusNode, ParentNode: w.parentNode, DebugLabel: w.debugLabel}),
		GestureDetectorOpts{OnTap: func() { FocusScopeOf(bc).RequestFocus(s.focusNode) }})
}

func tfKey() *GlobalKey[*testFocusState] { return NewGlobalKey[*testFocusState]() }

func tfState(k *GlobalKey[*testFocusState]) *testFocusState {
	s, ok := k.CurrentState()
	if !ok {
		panic("no state")
	}
	return s
}

func hasText(t *testing.T, ts *gluttertest.Tester, text string) {
	t.Helper()
	if n := ts.Count(gluttertest.ByText(text)); n != 1 {
		t.Errorf("found %d widgets with the text %q, want 1", n, text)
	}
}

func TestFocusScopeCanFocus(t *testing.T) {
	ts := gluttertest.New(t)
	key := tfKey()
	ts.PumpWidget(newTestFocus(key))
	if tfState(key).focusNode.HasFocus() {
		t.Fatal("the node should not have the focus yet")
	}
	FocusScopeOf(key.CurrentContext()).RequestFocus(tfState(key).focusNode)
	ts.PumpAndSettle()
	if !tfState(key).focusNode.HasFocus() {
		t.Error("the node should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
}

func TestFocusScopeCanUnfocus(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{newTestFocus(keyA), newTestFocus(keyB, testFocusOpts{name: "b"})}))
	if tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("nothing should have the focus yet")
	}
	hasText(t, ts, "a")
	hasText(t, ts, "b")
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Error("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
	// Setting the focus to B unfocuses A.
	FocusScopeOf(keyB.CurrentContext()).RequestFocus(tfState(keyB).focusNode)
	ts.PumpAndSettle()
	if tfState(keyA).focusNode.HasFocus() || !tfState(keyB).focusNode.HasFocus() {
		t.Error("B should have the focus")
	}
	hasText(t, ts, "a")
	hasText(t, ts, "B FOCUSED")
}

func TestFocusScopeAutofocusWorks(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{newTestFocus(keyA), newTestFocus(keyB, testFocusOpts{name: "b", autofocus: true})}))
	ts.Pump()
	if tfState(keyA).focusNode.HasFocus() || !tfState(keyB).focusNode.HasFocus() {
		t.Error("B should have the focus")
	}
	hasText(t, ts, "a")
	hasText(t, ts, "B FOCUSED")
}

func TestFocusScopeCanHaveMultipleFocusedChildrenAndTheyUpdateAccordingly(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{autofocus: true}), newTestFocus(keyB, testFocusOpts{name: "b"})}))
	// Autofocus is delayed one frame.
	ts.Pump()
	steps := []struct {
		tap            string
		aFocus, bFocus bool
		textA, textB   string
	}{
		{"", true, false, "A FOCUSED", "b"},
		{"A FOCUSED", true, false, "A FOCUSED", "b"},
		{"b", false, true, "a", "B FOCUSED"},
		{"a", true, false, "A FOCUSED", "b"},
	}
	for _, s := range steps {
		if s.tap != "" {
			ts.Tap(gluttertest.ByText(s.tap))
			ts.Pump()
		}
		if tfState(keyA).focusNode.HasFocus() != s.aFocus || tfState(keyB).focusNode.HasFocus() != s.bFocus {
			t.Errorf("after tapping %q: A %v, B %v", s.tap, tfState(keyA).focusNode.HasFocus(), tfState(keyB).focusNode.HasFocus())
		}
		hasText(t, ts, s.textA)
		hasText(t, ts, s.textB)
	}
}

// This moves a focus node first into a focus scope that is added to its parent,
// and then out of that focus scope again.
func TestFocusScopeCanMoveFocusInAndOutOfFocusScope(t *testing.T) {
	ts := gluttertest.New(t)
	parentFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope Node"}})
	childFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Child Scope Node"}})
	defer parentFocusScope.Dispose()
	defer childFocusScope.Dispose()
	key := tfKey()
	parentScope := func(children ...Widget) Widget {
		return NewFocusScope(NewColumn(children), FocusScopeOpts{DebugLabel: "Parent Scope", Node: parentFocusScope})
	}
	// Initially creates the focus inside of the parent FocusScope.
	ts.PumpWidget(NewFocusScope(NewColumn([]Widget{newTestFocus(key, testFocusOpts{debugLabel: "Child"})}),
		FocusScopeOpts{DebugLabel: "Parent Scope", Node: parentFocusScope, Autofocus: true}))
	if tfState(key).focusNode.HasFocus() {
		t.Fatal("the node should not have the focus yet")
	}
	hasText(t, ts, "a")
	FocusScopeOf(key.CurrentContext()).RequestFocus(tfState(key).focusNode)
	ts.PumpAndSettle()
	if !tfState(key).focusNode.HasFocus() {
		t.Fatal("the node should have the focus")
	}
	hasText(t, ts, "A FOCUSED")

	if foundation.DebugMode {
		const wantParent = `FocusScopeNode#00000(Parent Scope Node [IN FOCUS PATH])
 │ context: FocusScope
 │ IN FOCUS PATH
 │ focusedChildren: FocusNode#00000(Child [PRIMARY FOCUS])
 │
 └─Child 1: FocusNode#00000(Child [PRIMARY FOCUS])
     context: Focus
     PRIMARY FOCUS
`
		if got := hashRE.ReplaceAllString(foundation.ToStringDeep(parentFocusScope.Node()), "#00000"); got != wantParent {
			t.Errorf("the parent scope =\n%s\nwant\n%s", got, wantParent)
		}
	}

	// Adds the child focus scope to the focus tree.
	childAttachment := childFocusScope.Attach(key.CurrentContext())
	parentFocusScope.SetFirstFocus(childFocusScope)
	ts.PumpAndSettle()
	if !childFocusScope.IsFirstFocus() {
		t.Fatal("the child scope should be the first focus")
	}
	// Adds the child focus scope with no focusable child in it to the tree.
	ts.PumpWidget(parentScope(
		newTestFocus(key, testFocusOpts{debugLabel: "Child"}),
		NewFocusScope(NewContainer(nil), FocusScopeOpts{DebugLabel: "Child Scope", Node: childFocusScope})))
	if tfState(key).focusNode.HasFocus() {
		t.Error("the node should have lost the focus")
	}
	hasText(t, ts, "a")
	// Moves the existing focus node into the child focus scope.
	ts.PumpWidget(parentScope(
		NewFocusScope(newTestFocus(key, testFocusOpts{debugLabel: "Child"}), FocusScopeOpts{DebugLabel: "Child Scope", Node: childFocusScope})))
	ts.PumpAndSettle()
	if tfState(key).focusNode.HasFocus() {
		t.Error("the node should not have the focus in the child scope")
	}
	hasText(t, ts, "a")
	// Removes the child focus scope.
	ts.PumpWidget(parentScope(newTestFocus(key, testFocusOpts{debugLabel: "Child"})))
	ts.PumpAndSettle()
	if tfState(key).focusNode.HasFocus() {
		t.Error("the node should not have the focus")
	}
	hasText(t, ts, "a")
	// The child must be detached because it was attached to call SetFirstFocus
	// before adding it to the widget.
	childAttachment.Detach()
}

func TestFocusScopeSettingFirstFocusRequestsFocusForTheScopeProperly(t *testing.T) {
	ts := gluttertest.New(t)
	parentFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope Node"}})
	childFocusScope1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Child Scope Node 1"}})
	childFocusScope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Child Scope Node 2"}})
	defer parentFocusScope.Dispose()
	defer childFocusScope1.Dispose()
	defer childFocusScope2.Dispose()
	keyA, keyB, keyC := tfKey(), tfKey(), tfKey()
	ts.PumpWidget(NewFocusScope(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{
			newTestFocus(keyA, testFocusOpts{autofocus: true, debugLabel: "Child A"}),
			newTestFocus(keyB, testFocusOpts{name: "b", debugLabel: "Child B"}),
		}), FocusScopeOpts{DebugLabel: "Child Scope 1", Node: childFocusScope1}),
		NewFocusScope(newTestFocus(keyC, testFocusOpts{name: "c", debugLabel: "Child C"}),
			FocusScopeOpts{DebugLabel: "Child Scope 2", Node: childFocusScope2}),
	}), FocusScopeOpts{DebugLabel: "Parent Scope", Node: parentFocusScope}))
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")

	parentFocusScope.SetFirstFocus(childFocusScope2)
	ts.PumpAndSettle()
	if tfState(keyA).focusNode.HasFocus() {
		t.Error("A should have lost the focus")
	}
	hasText(t, ts, "a")

	parentFocusScope.SetFirstFocus(childFocusScope1)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() {
		t.Error("A should have the focus again")
	}
	hasText(t, ts, "A FOCUSED")

	tfState(keyB).focusNode.RequestFocus()
	ts.PumpAndSettle()
	if !tfState(keyB).focusNode.HasFocus() || !parentFocusScope.IsFirstFocus() || !childFocusScope1.IsFirstFocus() {
		t.Error("B should have the focus")
	}
	hasText(t, ts, "B FOCUSED")

	parentFocusScope.SetFirstFocus(childFocusScope2)
	ts.PumpAndSettle()
	if tfState(keyB).focusNode.HasFocus() || !parentFocusScope.IsFirstFocus() || childFocusScope1.IsFirstFocus() || !childFocusScope2.IsFirstFocus() {
		t.Error("B should have lost the focus")
	}
	hasText(t, ts, "b")

	tfState(keyC).focusNode.RequestFocus()
	ts.PumpAndSettle()
	if tfState(keyB).focusNode.HasFocus() || !tfState(keyC).focusNode.HasFocus() || !parentFocusScope.IsFirstFocus() ||
		childFocusScope1.IsFirstFocus() || !childFocusScope2.IsFirstFocus() {
		t.Error("C should have the focus")
	}
	hasText(t, ts, "b")
	hasText(t, ts, "C FOCUSED")

	childFocusScope1.RequestFocus()
	ts.PumpAndSettle()
	if !tfState(keyB).focusNode.HasFocus() || tfState(keyC).focusNode.HasFocus() || !parentFocusScope.IsFirstFocus() ||
		!childFocusScope1.IsFirstFocus() || childFocusScope2.IsFirstFocus() {
		t.Error("B should have the focus again")
	}
	hasText(t, ts, "B FOCUSED")
	hasText(t, ts, "c")
}

func TestFocusScopeRemovingFocusedWidgetMovesFocusToNextWidget(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{newTestFocus(keyA), newTestFocus(keyB, testFocusOpts{name: "b"})}))
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
	ts.PumpWidget(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{name: "b"})}))
	ts.Pump()
	if tfState(keyB).focusNode.HasFocus() {
		t.Error("B should not have the focus")
	}
	hasText(t, ts, "b")
}

func TestFocusScopeAddingANewFocusScopeAttachesTheChildToItsParent(t *testing.T) {
	ts := gluttertest.New(t)
	keyA := tfKey()
	parentFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope Node"}})
	childFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Child Scope Node"}})
	defer parentFocusScope.Dispose()
	defer childFocusScope.Dispose()
	ts.PumpWidget(NewFocusScope(newTestFocus(keyA, testFocusOpts{debugLabel: "Child"}), FocusScopeOpts{Node: childFocusScope}))
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	if FocusScopeOf(keyA.CurrentContext()) != childFocusScope || FocusOf(keyA.CurrentContext(), FocusOfOpts{ScopeOK: true}) != childFocusScope.Node() {
		t.Fatal("the nearest scope should be the child scope")
	}
	ts.Binding.FocusManager().RootScope().SetFirstFocus(FocusScopeOf(keyA.CurrentContext()))
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || !childFocusScope.IsFirstFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	ts.PumpWidget(NewFocusScope(
		NewFocusScope(newTestFocus(keyA, testFocusOpts{debugLabel: "Child"}), FocusScopeOpts{Node: childFocusScope}),
		FocusScopeOpts{Node: parentFocusScope}))
	ts.Pump()
	// The node keeps its focus when moved to the new scope.
	if !childFocusScope.IsFirstFocus() || !tfState(keyA).focusNode.HasFocus() {
		t.Error("A should keep its focus")
	}
	hasText(t, ts, "A FOCUSED")
}

func TestFocusScopeSettingParentNodeDeterminesFocusTreeHierarchy(t *testing.T) {
	ts := gluttertest.New(t)
	topNode, parentNode, childNode, insertedNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Top"}), NewFocusNode(FocusNodeOpts{DebugLabel: "Parent"}),
		NewFocusNode(FocusNodeOpts{DebugLabel: "Child"}), NewFocusNode(FocusNodeOpts{DebugLabel: "Inserted"})
	defer topNode.Dispose()
	defer parentNode.Dispose()
	defer childNode.Dispose()
	defer insertedNode.Dispose()
	child := func(inner Widget) Widget { return inner }
	_ = child
	ts.PumpWidget(NewFocusScope(NewFocusWithExternalFocusNode(topNode, NewColumn([]Widget{
		NewFocusWithExternalFocusNode(parentNode, NewSizedBox(nil)),
		NewFocusWithExternalFocusNode(childNode, NewSizedBox(nil), FocusWithExternalFocusNodeOpts{ParentNode: parentNode, Autofocus: true}),
	}))))
	ts.Pump()
	if !childNode.HasPrimaryFocus() || !parentNode.HasFocus() || !topNode.HasFocus() {
		t.Fatal("the child should have the focus")
	}
	// Inserting a Focus in between does not reparent the child.
	ts.PumpWidget(NewFocusScope(NewFocusWithExternalFocusNode(topNode, NewColumn([]Widget{
		NewFocusWithExternalFocusNode(parentNode, NewSizedBox(nil)),
		NewFocusWithExternalFocusNode(
			insertedNode,
			NewFocusWithExternalFocusNode(childNode, NewSizedBox(nil), FocusWithExternalFocusNodeOpts{ParentNode: parentNode, Autofocus: true})),
	}))))
	ts.Pump()
	if !childNode.HasPrimaryFocus() || !parentNode.HasFocus() || !topNode.HasFocus() || insertedNode.HasFocus() {
		t.Error("the child should still be below the parent node")
	}
}

func TestFocusScopeSettingParentNodeDeterminesFocusScopeTreeHierarchy(t *testing.T) {
	ts := gluttertest.New(t)
	topNode, parentNode, childNode, insertedNode := NewFocusScopeNode(), NewFocusScopeNode(), NewFocusScopeNode(), NewFocusScopeNode()
	defer topNode.Dispose()
	defer parentNode.Dispose()
	defer childNode.Dispose()
	defer insertedNode.Dispose()
	ts.PumpWidget(NewFocusScopeWithExternalFocusNode(topNode, NewColumn([]Widget{
		NewFocusScopeWithExternalFocusNode(parentNode, NewSizedBox(nil)),
		NewFocusScopeWithExternalFocusNode(childNode, NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true}),
			FocusScopeWithExternalFocusNodeOpts{ParentNode: parentNode.Node()}),
	})))
	ts.Pump()
	if !childNode.HasFocus() || !parentNode.HasFocus() || !topNode.HasFocus() {
		t.Fatal("the child scope should have the focus")
	}
	ts.PumpWidget(NewFocusScopeWithExternalFocusNode(topNode, NewColumn([]Widget{
		NewFocusScopeWithExternalFocusNode(parentNode, NewSizedBox(nil)),
		NewFocusScopeWithExternalFocusNode(
			insertedNode,
			NewFocusScopeWithExternalFocusNode(childNode, NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true}),
				FocusScopeWithExternalFocusNodeOpts{ParentNode: parentNode.Node()})),
	})))
	ts.Pump()
	if !childNode.HasFocus() || !parentNode.HasFocus() || !topNode.HasFocus() || insertedNode.HasFocus() {
		t.Error("the child scope should still be below the parent scope")
	}
}

// Arguably this is not the correct behavior, but it is what happens now.
func TestFocusScopeRemovingFocusedWidgetDoesNotMoveFocusToNextWidgetWithinFocusScope(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	parentFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope"}})
	defer parentFocusScope.Dispose()
	ts.PumpWidget(NewFocusScope(NewColumn([]Widget{
		newTestFocus(keyA, testFocusOpts{debugLabel: "Widget A"}),
		newTestFocus(keyB, testFocusOpts{debugLabel: "Widget B", name: "b"}),
	}), FocusScopeOpts{DebugLabel: "Parent Scope", Node: parentFocusScope, Autofocus: true}))
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	scope := FocusScopeOf(keyA.CurrentContext())
	ts.Binding.FocusManager().RootScope().SetFirstFocus(scope)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
	ts.PumpWidget(NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{name: "b"})}), FocusScopeOpts{Node: parentFocusScope}))
	ts.Pump()
	if tfState(keyB).focusNode.HasFocus() {
		t.Error("B should not have the focus")
	}
	hasText(t, ts, "b")
}

func TestFocusScopeRemovingAFocusScopeRemovesItsNodeFromTheTree(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	scopeKeyA, scopeKeyB := NewGlobalKey[State](), NewGlobalKey[State]()
	parentFocusScope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope"}})
	defer parentFocusScope.Dispose()
	// This checks both FocusScopes that have their own nodes and the ones that use
	// external nodes.
	ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{debugLabel: "Child A"})}), FocusScopeOpts{Key: scopeKeyA, Node: parentFocusScope}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b"})}), FocusScopeOpts{Key: scopeKeyB}),
	})))
	FocusScopeOf(keyB.CurrentContext()).RequestFocus(tfState(keyB).focusNode)
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	aScope := FocusScopeOf(keyA.CurrentContext())
	bScope := FocusScopeOf(keyB.CurrentContext())
	root := ts.Binding.FocusManager().RootScope()
	root.SetFirstFocus(bScope)
	root.SetFirstFocus(aScope)
	ts.PumpAndSettle()
	if !FocusScopeOf(keyA.CurrentContext()).IsFirstFocus() || !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
	if n := len(root.Descendants()); n != 7 {
		t.Errorf("the root scope has %d descendants, want 7", n)
	}
	ts.PumpWidget(NewContainer(nil))
	if n := len(root.Descendants()); n != 2 {
		t.Errorf("the root scope has %d descendants after the removal, want 2", n)
	}
	if slices.Contains(root.Descendants(), aScope.Node()) || slices.Contains(root.Descendants(), bScope.Node()) {
		t.Error("the scopes should be gone")
	}
}

// By "pinned" is meant kept in the tree by a GlobalKey.
func TestFocusScopeRemovingPinnedFocusedScopeDoesNotMoveFocusToFocusedWidgetWithinNextFocusScope(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	scopeKeyA, scopeKeyB := NewGlobalKey[State](), NewGlobalKey[State]()
	parentFocusScope1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope 1"}})
	parentFocusScope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope 2"}})
	defer parentFocusScope1.Dispose()
	defer parentFocusScope2.Dispose()
	ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{debugLabel: "Child A"})}), FocusScopeOpts{Key: scopeKeyA, Node: parentFocusScope1}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b"})}), FocusScopeOpts{Key: scopeKeyB, Node: parentFocusScope2}),
	})))
	FocusScopeOf(keyB.CurrentContext()).RequestFocus(tfState(keyB).focusNode)
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	bScope, aScope := FocusScopeOf(keyB.CurrentContext()), FocusScopeOf(keyA.CurrentContext())
	root := ts.Binding.FocusManager().RootScope()
	root.SetFirstFocus(bScope)
	root.SetFirstFocus(aScope)
	ts.PumpAndSettle()
	if !FocusScopeOf(keyA.CurrentContext()).IsFirstFocus() || !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b", autofocus: true})}),
			FocusScopeOpts{Key: scopeKeyB, Node: parentFocusScope2}),
	})))
	ts.Pump()
	if !tfState(keyB).focusNode.HasFocus() {
		t.Error("B should have the focus")
	}
	hasText(t, ts, "B FOCUSED")
}

func TestFocusScopeRemovingUnpinnedFocusedScopeDoesNotMoveFocusToFocusedWidgetWithinNextFocusScope(t *testing.T) {
	ts := gluttertest.New(t)
	keyA, keyB := tfKey(), tfKey()
	parentFocusScope1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope 1"}})
	parentFocusScope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Parent Scope 2"}})
	defer parentFocusScope1.Dispose()
	defer parentFocusScope2.Dispose()
	ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{debugLabel: "Child A"})}), FocusScopeOpts{Node: parentFocusScope1}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b"})}), FocusScopeOpts{Node: parentFocusScope2}),
	})))
	FocusScopeOf(keyB.CurrentContext()).RequestFocus(tfState(keyB).focusNode)
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	bScope, aScope := FocusScopeOf(keyB.CurrentContext()), FocusScopeOf(keyA.CurrentContext())
	root := ts.Binding.FocusManager().RootScope()
	root.SetFirstFocus(bScope)
	root.SetFirstFocus(aScope)
	ts.PumpAndSettle()
	if !FocusScopeOf(keyA.CurrentContext()).IsFirstFocus() || !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	ts.PumpWidget(NewFocusTraversalGroup(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b", autofocus: true})}),
			FocusScopeOpts{Node: parentFocusScope2}),
	})))
	ts.Pump()
	if !tfState(keyB).focusNode.HasFocus() {
		t.Error("B should have the focus")
	}
	hasText(t, ts, "B FOCUSED")
}

func TestFocusScopeMovingWidgetFromOneScopeToAnotherRetainsFocus(t *testing.T) {
	ts := gluttertest.New(t)
	parentFocusScope1, parentFocusScope2 := NewFocusScopeNode(), NewFocusScopeNode()
	defer parentFocusScope1.Dispose()
	defer parentFocusScope2.Dispose()
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA)}), FocusScopeOpts{Node: parentFocusScope1}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{name: "b"})}), FocusScopeOpts{Node: parentFocusScope2}),
	}))
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	aScope := FocusScopeOf(keyA.CurrentContext())
	ts.Binding.FocusManager().RootScope().SetFirstFocus(aScope)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
	ts.PumpWidget(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{name: "b"})}), FocusScopeOpts{Node: parentFocusScope1}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA)}), FocusScopeOpts{Node: parentFocusScope2}),
	}))
	ts.Pump()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Error("A should keep the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
}

func TestFocusScopeMovingFocusScopeNodesRetainsFocus(t *testing.T) {
	ts := gluttertest.New(t)
	parentFocusScope1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope 1"}})
	parentFocusScope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope 2"}})
	defer parentFocusScope1.Dispose()
	defer parentFocusScope2.Dispose()
	keyA, keyB := tfKey(), tfKey()
	ts.PumpWidget(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{debugLabel: "Child A"})}), FocusScopeOpts{Node: parentFocusScope1}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b"})}), FocusScopeOpts{Node: parentFocusScope2}),
	}))
	FocusScopeOf(keyA.CurrentContext()).RequestFocus(tfState(keyA).focusNode)
	aScope := FocusScopeOf(keyA.CurrentContext())
	ts.Binding.FocusManager().RootScope().SetFirstFocus(aScope)
	ts.PumpAndSettle()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Fatal("A should have the focus")
	}
	// This swaps the FocusScopeNodes that the FocusScopes have.
	ts.PumpWidget(NewColumn([]Widget{
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyA, testFocusOpts{debugLabel: "Child A"})}), FocusScopeOpts{Node: parentFocusScope2}),
		NewFocusScope(NewColumn([]Widget{newTestFocus(keyB, testFocusOpts{debugLabel: "Child B", name: "b"})}), FocusScopeOpts{Node: parentFocusScope1}),
	}))
	ts.Pump()
	if !tfState(keyA).focusNode.HasFocus() || tfState(keyB).focusNode.HasFocus() {
		t.Error("A should keep the focus")
	}
	hasText(t, ts, "A FOCUSED")
	hasText(t, ts, "b")
}

func TestFocusScopeCanFocusRootNode(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	ts.PumpWidget(NewFocus(NewContainer(nil), FocusOpts{Key: key1}))
	rootNode := FocusScopeOf(ts.Element(gluttertest.ByKey(key1)).(BuildContext))
	rootNode.RequestFocus()
	ts.Pump()
	if !rootNode.HasFocus() {
		t.Error("the root node should have the focus")
	}
	if got := ts.Binding.FocusManager().RootScope().Descendants()[1]; got != rootNode.Node() {
		t.Errorf("the second descendant of the root scope = %v, want the view scope", got)
	}
}

func TestFocusScopeCanAutofocusANode(t *testing.T) {
	ts := gluttertest.New(t)
	focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node"})
	defer focusNode.Dispose()
	ts.PumpWidget(NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNode}))
	ts.Pump()
	if focusNode.HasPrimaryFocus() {
		t.Fatal("the node should not have the focus yet")
	}
	ts.PumpWidget(NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNode, Autofocus: true}))
	ts.Pump()
	if !focusNode.HasPrimaryFocus() {
		t.Error("the node should have the focus")
	}
}

func TestFocusScopeWillNotAutofocusANodeIfOneIsAlreadyFocused(t *testing.T) {
	ts := gluttertest.New(t)
	focusNodeA, focusNodeB := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node A"}), NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node B"})
	defer focusNodeA.Dispose()
	defer focusNodeB.Dispose()
	ts.PumpWidget(NewColumn([]Widget{NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNodeA, Autofocus: true})}))
	ts.Pump()
	if !focusNodeA.HasPrimaryFocus() {
		t.Fatal("A should have the focus")
	}
	ts.PumpWidget(NewColumn([]Widget{
		NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNodeA}),
		NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNodeB, Autofocus: true}),
	}))
	ts.Pump()
	if focusNodeB.HasPrimaryFocus() || !focusNodeA.HasPrimaryFocus() {
		t.Error("A should keep the focus")
	}
}

// ctxOf is the context of the one element the finder finds.
func ctxOf(ts *gluttertest.Tester, f gluttertest.Finder) BuildContext {
	return ts.Element(f).(BuildContext)
}

func TestFocusScopeExternalNodeAttributesAreNotUpdatedByTheWidget(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	focusScopeNode := NewFocusScopeNode()
	defer focusScopeNode.Dispose()
	keyEventHandled := false
	focusScopeNode.SetOnKeyEvent(func(*FocusNode, services.KeyEvent) KeyEventResult { return KeyEventResultIgnored })
	focusScopeNode.SetDescendantsAreFocusable(false)
	focusScopeNode.SetDescendantsAreTraversable(false)
	focusScopeNode.SetSkipTraversal(false)
	focusScopeNode.SetCanRequestFocus(true)
	widget := func() *FocusScope {
		return NewFocusScopeWithExternalFocusNode(focusScopeNode, NewContainer(nil, ContainerOpts{Key: key1}))
	}
	check := func(w *FocusScope, descendantsFocusable, descendantsTraversable bool) {
		t.Helper()
		if focusScopeNode.DescendantsAreFocusable() != descendantsFocusable || focusScopeNode.DescendantsAreTraversable() != descendantsTraversable ||
			focusScopeNode.SkipTraversal() || !focusScopeNode.CanRequestFocus() {
			t.Errorf("the node was modified: %v %v %v %v", focusScopeNode.DescendantsAreFocusable(), focusScopeNode.DescendantsAreTraversable(),
				focusScopeNode.SkipTraversal(), focusScopeNode.CanRequestFocus())
		}
		if w.DescendantsAreFocusable() != focusScopeNode.DescendantsAreFocusable() || w.DescendantsAreTraversable() != focusScopeNode.DescendantsAreTraversable() ||
			w.SkipTraversal() != focusScopeNode.SkipTraversal() || w.CanRequestFocus() != focusScopeNode.CanRequestFocus() {
			t.Error("the widget does not follow the node")
		}
	}
	w := widget()
	ts.PumpWidget(w)
	check(w, false, false)
	FocusScopeOf(key1.CurrentContext()).RequestFocus()
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if keyEventHandled {
		t.Fatal("the key event was handled")
	}
	focusScopeNode.SetOnKeyEvent(func(*FocusNode, services.KeyEvent) KeyEventResult {
		keyEventHandled = true
		return KeyEventResultHandled
	})
	focusScopeNode.SetDescendantsAreFocusable(true)
	focusScopeNode.SetDescendantsAreTraversable(true)
	w = widget()
	ts.PumpWidget(w)
	check(w, true, true)
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if !keyEventHandled {
		t.Error("the key event was not handled")
	}
}

func TestFocusScopeDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{})
	focusScopeNode := NewFocusScopeNode()
	defer focusScopeNode.Dispose()
	key := foundation.NewValueKey("scope")
	ts.PumpWidget(ltr(NewCenter(NewFocusScope(NewText("X"), FocusScopeOpts{Key: key, Node: focusScopeNode}))))
	if got := ts.Size(gluttertest.ByKey(key)); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
	focusScopeNode.RequestFocus()
	ts.Pump()
}

func TestFocusOfStopsAtTheNearestFocusWidget(t *testing.T) {
	ts := gluttertest.New(t)
	keys := make([]*GlobalKey[State], 7)
	for i := range keys {
		keys[i] = NewGlobalKey[State]()
	}
	scopeNode := NewFocusScopeNode()
	defer scopeNode.Dispose()
	ts.PumpWidget(NewFocusScope(
		NewContainer(
			NewFocus(
				NewContainer(
					NewFocus(NewContainer(nil, ContainerOpts{Key: keys[6]}), FocusOpts{DebugLabel: "Key 5", Key: keys[5]}),
					ContainerOpts{Key: keys[4]}),
				FocusOpts{DebugLabel: "Key 3", Key: keys[3]}),
			ContainerOpts{Key: keys[2]}),
		FocusScopeOpts{Key: keys[1], Node: scopeNode, DebugLabel: "Key 1"}))
	el := func(i int) BuildContext { return ctxOf(ts, gluttertest.ByKey(keys[i])) }
	root := el(1).Owner().FocusManager().RootScope().Node()
	for i := 1; i <= 3; i++ {
		if FocusMaybeOf(el(i)) != nil {
			t.Errorf("FocusMaybeOf(element %d) is not nil", i)
		}
	}
	up := func(n *FocusNode, times int) *FocusNode {
		for ; times > 0; times-- {
			n = n.Parent()
		}
		return n
	}
	if up(FocusOf(el(4)), 4) != root || up(FocusOf(el(5)), 4) != root || up(FocusOf(el(6)), 5) != root {
		t.Error("the nodes are not at the expected depth below the root scope")
	}
}

func TestFocusCanTraverseFocusChildren(t *testing.T) {
	ts := gluttertest.New(t)
	keys := make([]*GlobalKey[State], 9)
	for i := range keys {
		keys[i] = NewGlobalKey[State]()
	}
	ts.PumpWidget(NewFocus(NewColumn([]Widget{
		NewFocus(NewFocus(NewContainer(nil), FocusOpts{Key: keys[3]}), FocusOpts{Key: keys[2]}),
		NewFocus(NewFocus(NewContainer(nil), FocusOpts{Key: keys[5]}), FocusOpts{Key: keys[4]}),
		NewFocus(NewColumn([]Widget{
			NewFocus(NewContainer(nil), FocusOpts{Key: keys[7]}),
			NewFocus(NewContainer(nil), FocusOpts{Key: keys[8]}),
		}), FocusOpts{Key: keys[6]}),
	}, ColumnOpts{Key: keys[1]})))
	ts.Pump()
	var gotKeys []foundation.Key
	visit := func(descendants []*FocusNode) {
		gotKeys = nil
		for _, n := range descendants {
			gotKeys = append(gotKeys, n.Context().Widget().Key())
		}
	}
	firstScope := ctxOf(ts, gluttertest.ByKey(keys[1]))
	visit(FocusOf(firstScope).Descendants())
	if len(gotKeys) != 7 {
		t.Fatalf("found %d nodes, want 7", len(gotKeys))
	}
	// Depth first.
	want := []foundation.Key{keys[3], keys[2], keys[5], keys[4], keys[7], keys[8], keys[6]}
	for i := range want {
		if gotKeys[i] != want[i] {
			t.Errorf("descendant %d has the key %v, want %v", i, gotKeys[i], want[i])
		}
	}
	// Just traverses a sub-tree.
	visit(FocusOf(ctxOf(ts, gluttertest.ByKey(keys[7]))).Descendants())
	if len(gotKeys) != 2 || gotKeys[0] != keys[7] || gotKeys[1] != keys[8] {
		t.Errorf("sub-tree keys = %v", gotKeys)
	}
}

func TestFocusCanSetFocus(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	gotFocus := false
	ts.PumpWidget(NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{OnFocusChange: func(f bool) { gotFocus = f }}))
	node := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	node.RequestFocus()
	ts.Pump()
	if !gotFocus || !node.HasFocus() {
		t.Error("the node should have the focus")
	}
}

func TestFocusIsIgnoredWhenSetToNotFocusable(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	var gotFocus *bool
	ts.PumpWidget(NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{
		CanRequestFocus: foundation.Some(false), OnFocusChange: func(f bool) { gotFocus = &f }}))
	node := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	node.RequestFocus()
	ts.Pump()
	if gotFocus != nil || node.HasFocus() {
		t.Error("the node should not have the focus")
	}
}

func TestFocusIsLostWhenSetToNotFocusable(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	var gotFocus *bool
	onChange := func(f bool) { gotFocus = &f }
	ts.PumpWidget(NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{
		Autofocus: true, CanRequestFocus: foundation.Some(true), OnFocusChange: onChange}))
	node := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	node.RequestFocus()
	ts.Pump()
	if gotFocus == nil || !*gotFocus || !node.HasFocus() {
		t.Fatal("the node should have the focus")
	}
	gotFocus = nil
	ts.PumpWidget(NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{
		CanRequestFocus: foundation.Some(false), OnFocusChange: onChange}))
	node = FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	node.RequestFocus()
	ts.Pump()
	if gotFocus == nil || *gotFocus || node.HasFocus() {
		t.Errorf("the node should have lost the focus, got %v", gotFocus)
	}
}

func TestFocusChildOfUnfocusableFocusCanGetFocus(t *testing.T) {
	ts := gluttertest.New(t)
	key1, key2 := NewGlobalKey[State](), NewGlobalKey[State]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	var gotFocus *bool
	ts.PumpWidget(NewFocus(
		NewFocus(NewContainer(nil, ContainerOpts{Key: key2}), FocusOpts{Key: key1, FocusNode: focusNode}),
		FocusOpts{CanRequestFocus: foundation.Some(false), OnFocusChange: func(f bool) { gotFocus = &f }}))
	unfocusableNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	unfocusableNode.RequestFocus()
	ts.Pump()
	if gotFocus != nil || unfocusableNode.HasFocus() {
		t.Fatal("the unfocusable node should not have the focus")
	}
	focusableNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key2)))
	focusableNode.RequestFocus()
	ts.Pump()
	if gotFocus == nil || !*gotFocus || !unfocusableNode.HasFocus() {
		t.Error("the node should have the focus")
	}
}

func TestFocusNodesAreRemovedWhenAllFocusesAreRemoved(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	gotFocus := false
	ts.PumpWidget(NewFocusScope(NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{OnFocusChange: func(f bool) { gotFocus = f }})))
	node := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	node.RequestFocus()
	ts.Pump()
	if !gotFocus || !node.HasFocus() {
		t.Fatal("the node should have the focus")
	}
	ts.PumpWidget(NewContainer(nil))
	// Even with no other focusable widgets, there are the top level focus
	// traversal and view focus nodes.
	if n := len(ts.Binding.FocusManager().RootScope().Descendants()); n != 2 {
		t.Errorf("the root scope has %d descendants, want 2", n)
	}
}

func TestFocusWidgetsSetSemanticsInformationAboutFocus(t *testing.T) {
	ts := gluttertest.New(t)
	key := tfKey()
	ts.PumpWidget(newTestFocus(key))
	semanticsOf := func() *Semantics {
		return ts.Widget(gluttertest.Descendant(gluttertest.ByKey(key), gluttertest.ByType[*Semantics]())).(*Semantics)
	}
	focused := func() foundation.Opt[bool] { return semanticsOf().Properties().Focused }
	if tfState(key).focusNode.HasFocus() || focused() != foundation.Some(false) {
		t.Fatalf("the node should not have the focus: %v", focused())
	}
	FocusScopeOf(key.CurrentContext()).RequestFocus(tfState(key).focusNode)
	ts.PumpAndSettle()
	if !tfState(key).focusNode.HasFocus() || focused() != foundation.Some(true) {
		t.Fatalf("the node should have the focus: %v", focused())
	}
	tfState(key).focusNode.SetCanRequestFocus(false)
	ts.PumpAndSettle()
	if tfState(key).focusNode.HasFocus() || tfState(key).focusNode.CanRequestFocus() || focused().IsSome() {
		t.Errorf("the node should not be focusable: %v", focused())
	}
}

func TestFocusSettingCanRequestFocusOnFocusNodeCausesUpdate(t *testing.T) {
	ts := gluttertest.New(t)
	key := tfKey()
	ts.PumpWidget(newTestFocus(key))
	ts.PumpAndSettle()
	tfState(key).built = false
	tfState(key).focusNode.SetCanRequestFocus(false)
	ts.PumpAndSettle()
	if tfState(key).focusNode.CanRequestFocus() {
		t.Error("the node should not be focusable")
	}
}

func TestFocusCanRequestFocusCausesDescendantsOfScopeToBeSkipped(t *testing.T) {
	ts := gluttertest.New(t)
	scope1, scope2, focus1, focus2, container1 := NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()
	type allow struct{ scope1, scope2, focus1, focus2 bool }
	pumpTest := func(a allow) {
		ts.PumpWidget(NewFocusScope(
			NewFocusScope(
				NewFocus(
					NewFocus(NewContainer(nil, ContainerOpts{Key: container1}), FocusOpts{Key: focus2, CanRequestFocus: foundation.Some(a.focus2)}),
					FocusOpts{Key: focus1, CanRequestFocus: foundation.Some(a.focus1)}),
				FocusScopeOpts{Key: scope2, CanRequestFocus: foundation.Some(a.scope2)}),
			FocusScopeOpts{Key: scope1, CanRequestFocus: foundation.Some(a.scope1)}))
		ts.Pump()
	}
	all := allow{true, true, true, true}
	nodeAt := func() *FocusNode { return FocusOf(container1.CurrentContext()) }
	hasFocus := func() bool { return nodeAt().HasFocus() }
	want := func(step string, v bool) {
		t.Helper()
		if hasFocus() != v {
			t.Errorf("%s: has focus = %v, want %v", step, hasFocus(), v)
		}
	}

	// Checks the childless node (focus2).
	pumpTest(all)
	nodeAt().RequestFocus()
	ts.Pump()
	want("focus2 requested", true)
	a := all
	a.focus2 = false
	pumpTest(a)
	want("focus2 disallowed", false)
	nodeAt().RequestFocus()
	ts.Pump()
	want("focus2 disallowed, requested", false)
	pumpTest(all)
	nodeAt().RequestFocus()
	ts.Pump()
	want("focus2 allowed again", true)

	// Checks the node with a child (focus1): it does not affect the children.
	a = all
	a.focus1 = false
	pumpTest(a)
	want("focus1 disallowed", true)                 // focus2 has the focus
	FocusOf(focus2.CurrentContext()).RequestFocus() // tries to focus focus1
	ts.Pump()
	want("focus1 requested", true) // focus2 still has the focus
	nodeAt().RequestFocus()        // now tries to focus focus2
	ts.Pump()
	want("focus2 requested again", true)
	pumpTest(all)
	// Tries again, now that focus1 can be focused again.
	nodeAt().Unfocus()
	ts.Pump()
	want("unfocused", false)
	nodeAt().RequestFocus()
	ts.Pump()
	want("focus requested after the unfocus", true)

	// Checks the scope with only FocusNode children (scope2): it affects the children.
	a = all
	a.scope2 = false
	pumpTest(a)
	want("scope2 disallowed", false)
	FocusScopeOf(focus1.CurrentContext()).RequestFocus() // tries to focus scope2
	ts.Pump()
	want("scope2 requested", false)
	FocusOf(focus2.CurrentContext()).RequestFocus() // tries to focus focus1
	ts.Pump()
	want("focus1 requested", false)
	nodeAt().RequestFocus() // tries to focus focus2
	ts.Pump()
	want("focus2 requested", false)
	pumpTest(all)
	nodeAt().RequestFocus()
	ts.Pump()
	want("scope2 allowed again", true)

	// Checks the scope with FocusNode and FocusScope children (scope1): it affects the children.
	a = all
	a.scope1 = false
	pumpTest(a)
	want("scope1 disallowed", false)
	for _, k := range []*GlobalKey[State]{scope2, focus1} {
		FocusScopeOf(k.CurrentContext()).RequestFocus()
		ts.Pump()
		want("a scope requested", false)
	}
	FocusOf(focus2.CurrentContext()).RequestFocus()
	ts.Pump()
	want("focus1 requested", false)
	nodeAt().RequestFocus()
	ts.Pump()
	want("focus2 requested", false)
	pumpTest(all)
	nodeAt().RequestFocus()
	ts.Pump()
	want("scope1 allowed again", true)
}

func TestFocusSkipTraversalWorksAsExpected(t *testing.T) {
	ts := gluttertest.New(t)
	scope1, scope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "scope1"}}), NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "scope2"}})
	focus1, focus2 := NewFocusNode(FocusNodeOpts{DebugLabel: "focus1"}), NewFocusNode(FocusNodeOpts{DebugLabel: "focus2"})
	defer scope1.Dispose()
	defer scope2.Dispose()
	defer focus1.Dispose()
	defer focus2.Dispose()
	type skip struct{ scope1, scope2, focus1, focus2 bool }
	pumpTest := func(s skip) {
		ts.PumpWidget(NewFocusScope(
			NewFocusScope(
				NewFocus(
					NewFocus(NewContainer(nil), FocusOpts{FocusNode: focus2, SkipTraversal: foundation.Some(s.focus2)}),
					FocusOpts{FocusNode: focus1, SkipTraversal: foundation.Some(s.focus1)}),
				FocusScopeOpts{Node: scope2, SkipTraversal: foundation.Some(s.scope2)}),
			FocusScopeOpts{Node: scope1, SkipTraversal: foundation.Some(s.scope1)}))
		ts.Pump()
	}
	want := func(step string, ns ...*FocusNode) {
		t.Helper()
		if got := scope1.TraversalDescendants(); !slices.Equal(got, ns) {
			t.Errorf("%s: traversal descendants = %v, want %v", step, got, ns)
		}
	}
	pumpTest(skip{})
	want("nothing skipped", focus2, focus1, scope2.Node())
	// The childless node (focus2).
	pumpTest(skip{focus2: true})
	want("focus2 skipped", focus1, scope2.Node())
	// The node with a child (focus1): it does not affect the children.
	pumpTest(skip{focus1: true})
	want("focus1 skipped", focus2, scope2.Node())
	// The scope with only FocusNode children (scope2): it affects the children.
	pumpTest(skip{scope2: true})
	want("scope2 skipped", focus2, focus1)
	// The scope with both kinds of children (scope1).
	pumpTest(skip{scope1: true})
	want("scope1 skipped", focus2, focus1, scope2.Node())
}

func TestFocusDescendantsAreFocusableWorksAsExpected(t *testing.T) {
	ts := gluttertest.New(t)
	key1, key2 := NewGlobalKey[State](), NewGlobalKey[State]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	var gotFocus *bool
	ts.PumpWidget(NewFocus(
		NewFocus(
			NewFocus(NewContainer(nil, ContainerOpts{Key: key2}), FocusOpts{Key: key1, FocusNode: focusNode}),
			FocusOpts{OnFocusChange: func(f bool) { gotFocus = &f }}),
		FocusOpts{DescendantsAreFocusable: foundation.Some(false)}))
	unfocusableNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	containerNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key2)))
	for _, n := range nodes(unfocusableNode, containerNode) {
		n.RequestFocus()
		ts.Pump()
		if gotFocus != nil || containerNode.HasFocus() || unfocusableNode.HasFocus() {
			t.Fatal("no node should have the focus")
		}
	}
}

func TestFocusDescendantsAreTraversableWorksAsExpected(t *testing.T) {
	ts := gluttertest.New(t)
	scopeNode := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "scope"}})
	node1, node2, node3 := NewFocusNode(FocusNodeOpts{DebugLabel: "node 1"}), NewFocusNode(FocusNodeOpts{DebugLabel: "node 2"}), NewFocusNode(FocusNodeOpts{DebugLabel: "node 3"})
	defer scopeNode.Dispose()
	defer node1.Dispose()
	defer node2.Dispose()
	defer node3.Dispose()
	ts.PumpWidget(NewFocusScope(NewColumn([]Widget{
		NewFocus(NewContainer(nil), FocusOpts{FocusNode: node1}),
		NewFocus(NewFocus(NewContainer(nil), FocusOpts{FocusNode: node3}), FocusOpts{FocusNode: node2, DescendantsAreTraversable: foundation.Some(false)}),
	}), FocusScopeOpts{Node: scopeNode}))
	ts.Pump()
	if got := scopeNode.TraversalDescendants(); !slices.Equal(got, nodes(node1, node2)) {
		t.Errorf("traversal descendants = %v", got)
	}
	if got := node2.TraversalDescendants(); len(got) != 0 {
		t.Errorf("traversal descendants of node2 = %v, want none", got)
	}
}

func TestFocusUpdatesTheOnKeyEventHandlerWhenTheWidgetUpdates(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	keyEventHandled := false
	handle := func(*FocusNode, services.KeyEvent) KeyEventResult {
		keyEventHandled = true
		return KeyEventResultHandled
	}
	ignore := func(*FocusNode, services.KeyEvent) KeyEventResult { return KeyEventResultIgnored }
	widget := func(h FocusOnKeyEventCallback) *Focus {
		return NewFocus(NewContainer(nil, ContainerOpts{Key: key1}), FocusOpts{
			OnKeyEvent: h, FocusNode: focusNode, SkipTraversal: foundation.Some(true), CanRequestFocus: foundation.Some(true)})
	}
	focusNode.SetOnKeyEvent(nil)
	w := widget(ignore) // this one does nothing
	ts.PumpWidget(w)
	if focusNode.OnKeyEvent() == nil || w.OnKeyEvent() == nil || w.DescendantsAreFocusable() != focusNode.DescendantsAreFocusable() ||
		w.SkipTraversal() != focusNode.SkipTraversal() || w.CanRequestFocus() != focusNode.CanRequestFocus() {
		t.Fatal("the node does not follow the widget")
	}
	FocusOf(key1.CurrentContext()).RequestFocus()
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if keyEventHandled {
		t.Fatal("the event was handled by the ignoring handler")
	}
	w = widget(handle)
	ts.PumpWidget(w)
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if !keyEventHandled {
		t.Error("the new handler did not get the event")
	}
}

func TestFocusExternalNodeAttributesAreNotUpdatedByTheWidget(t *testing.T) {
	ts := gluttertest.New(t)
	key1 := NewGlobalKey[State]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	keyEventHandled := false
	focusNode.SetOnKeyEvent(func(*FocusNode, services.KeyEvent) KeyEventResult { return KeyEventResultIgnored })
	focusNode.SetDescendantsAreFocusable(false)
	focusNode.SetDescendantsAreTraversable(false)
	focusNode.SetSkipTraversal(false)
	focusNode.SetCanRequestFocus(true)
	widget := func() *Focus {
		return NewFocusWithExternalFocusNode(focusNode, NewContainer(nil, ContainerOpts{Key: key1}))
	}
	check := func(w *Focus, focusable, traversable bool) {
		t.Helper()
		if focusNode.DescendantsAreFocusable() != focusable || focusNode.DescendantsAreTraversable() != traversable || focusNode.SkipTraversal() || !focusNode.CanRequestFocus() {
			t.Errorf("the node was modified")
		}
		if w.DescendantsAreFocusable() != focusNode.DescendantsAreFocusable() || w.DescendantsAreTraversable() != focusNode.DescendantsAreTraversable() ||
			w.SkipTraversal() != focusNode.SkipTraversal() || w.CanRequestFocus() != focusNode.CanRequestFocus() {
			t.Error("the widget does not follow the node")
		}
	}
	w := widget()
	ts.PumpWidget(w)
	check(w, false, false)
	FocusOf(key1.CurrentContext()).RequestFocus()
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if keyEventHandled {
		t.Fatal("the event was handled")
	}
	focusNode.SetOnKeyEvent(func(*FocusNode, services.KeyEvent) KeyEventResult {
		keyEventHandled = true
		return KeyEventResultHandled
	})
	focusNode.SetDescendantsAreFocusable(true)
	focusNode.SetDescendantsAreTraversable(true)
	w = widget()
	ts.PumpWidget(w)
	check(w, true, true)
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if !keyEventHandled {
		t.Error("the event was not handled")
	}
}

func TestFocusDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{})
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	key := foundation.NewValueKey("focus")
	ts.PumpWidget(ltr(NewCenter(NewFocus(NewText("X"), FocusOpts{Key: key, FocusNode: focusNode}))))
	if got := ts.Size(gluttertest.ByKey(key)); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
	focusNode.RequestFocus()
	ts.Pump()
}

func TestFocusDoesNotIntroduceASemanticsNodeWhenIncludeSemanticsIsFalse(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewFocus(NewContainer(nil), FocusOpts{ExcludeSemantics: true}))
	if n := ts.Count(gluttertest.Descendant(gluttertest.ByType[*Focus](), gluttertest.ByType[*Semantics]())); n != 0 {
		t.Errorf("found %d Semantics widgets", n)
	}
}

func TestExcludeFocusDescendantsAreNotFocusable(t *testing.T) {
	ts := gluttertest.New(t)
	key1, key2 := NewGlobalKey[State](), NewGlobalKey[State]()
	focusNode := NewFocusNode()
	defer focusNode.Dispose()
	var gotFocus *bool
	ts.PumpWidget(NewExcludeFocus(NewFocus(
		NewFocus(NewContainer(nil, ContainerOpts{Key: key2}), FocusOpts{Key: key1, FocusNode: focusNode}),
		FocusOpts{OnFocusChange: func(f bool) { gotFocus = &f }})))
	unfocusableNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key1)))
	containerNode := FocusOf(ctxOf(ts, gluttertest.ByKey(key2)))
	for _, n := range nodes(unfocusableNode, containerNode) {
		n.RequestFocus()
		ts.Pump()
		if gotFocus != nil || containerNode.HasFocus() || unfocusableNode.HasFocus() {
			t.Fatal("no node should have the focus")
		}
	}
}

// Regression test for https://github.com/flutter/flutter/issues/61700.
func TestExcludeFocusDoesNotTransferFocusToAnotherDescendant(t *testing.T) {
	ts := gluttertest.New(t)
	parentFocusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "group"})
	focusNode1, focusNode2 := NewFocusNode(FocusNodeOpts{DebugLabel: "node 1"}), NewFocusNode(FocusNodeOpts{DebugLabel: "node 2"})
	defer parentFocusNode.Dispose()
	defer focusNode1.Dispose()
	defer focusNode2.Dispose()
	tree := func(excluding bool) Widget {
		return NewExcludeFocus(NewFocus(NewColumn([]Widget{
			NewFocus(NewContainer(nil), FocusOpts{Autofocus: true, FocusNode: focusNode1}),
			NewFocus(NewContainer(nil), FocusOpts{FocusNode: focusNode2}),
		}), FocusOpts{FocusNode: parentFocusNode}), ExcludeFocusOpts{NotExcluding: !excluding})
	}
	ts.PumpWidget(tree(false))
	ts.Pump()
	if !parentFocusNode.HasFocus() || !focusNode1.HasPrimaryFocus() || focusNode2.HasFocus() {
		t.Fatal("focusNode1 should have the focus")
	}
	// Moves the focus to the second node to create some focus history for the scope.
	focusNode2.RequestFocus()
	ts.Pump()
	if !parentFocusNode.HasFocus() || focusNode1.HasFocus() || !focusNode2.HasPrimaryFocus() {
		t.Fatal("focusNode2 should have the focus")
	}
	// Turns off the focus for the subtree.
	ts.PumpWidget(tree(true))
	ts.Pump()
	if focusNode1.HasFocus() || focusNode2.HasFocus() || parentFocusNode.HasFocus() || !parentFocusNode.EnclosingScope().HasPrimaryFocus() {
		t.Error("the enclosing scope should have the primary focus")
	}
}

func TestExcludeFocusDoesNotIntroduceASemanticsNode(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewExcludeFocus(NewContainer(nil)))
	if n := ts.Count(gluttertest.Descendant(gluttertest.ByType[*ExcludeFocus](), gluttertest.ByType[*Semantics]())); n != 0 {
		t.Errorf("found %d Semantics widgets", n)
	}
}

// Regression test for https://github.com/flutter/flutter/issues/92693.
func TestSettingParentFocusScopeCanRequestFocusToFalseDoesNotSetDescendantCanRequestFocusToFalse(t *testing.T) {
	ts := gluttertest.New(t)
	childFocusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "node 1"})
	defer childFocusNode.Dispose()
	tree := func(parentCanRequestFocus bool) Widget {
		return NewFocusScope(
			NewColumn([]Widget{NewFocus(NewContainer(nil), FocusOpts{FocusNode: childFocusNode})}),
			FocusScopeOpts{CanRequestFocus: foundation.Some(parentCanRequestFocus)})
	}
	for _, step := range []struct {
		parent bool
		want   bool
	}{{true, true}, {false, false}, {true, true}} {
		ts.PumpWidget(tree(step.parent))
		if got := childFocusNode.CanRequestFocus(); got != step.want {
			t.Errorf("parent canRequestFocus %v: child canRequestFocus = %v, want %v", step.parent, got, step.want)
		}
	}
}

func TestFocusPassesChangesInAttributeValuesToItsFocusNode(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewFocus(NewContainer(nil)))
}
