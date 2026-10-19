package widgets

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// keyedLeaves makes a leaf for each name; a name that is empty has no key.
func keyedLeaves(names []string) []Widget {
	ws := make([]Widget, len(names))
	for i, n := range names {
		if n == "" {
			ws[i] = newLeaf()
		} else {
			ws[i] = newLeaf(WidgetOpts{Key: foundation.NewKey(n)})
		}
	}
	return ws
}

func keyName(w Widget) string {
	if k, ok := w.Key().(foundation.ValueKey[string]); ok {
		return k.Value()
	}
	return ""
}

func TestUpdateChildren(t *testing.T) {
	tests := []struct {
		name     string
		old, new []string
	}{
		{"append", []string{"a", "b"}, []string{"a", "b", "c"}},
		{"prepend", []string{"a", "b"}, []string{"x", "a", "b"}},
		{"insert in the middle", []string{"a", "b"}, []string{"a", "x", "b"}},
		{"remove from the middle", []string{"a", "b", "c"}, []string{"a", "c"}},
		{"remove the first", []string{"a", "b", "c"}, []string{"b", "c"}},
		{"remove the last", []string{"a", "b", "c"}, []string{"a", "b"}},
		{"reverse", []string{"a", "b", "c", "d"}, []string{"d", "c", "b", "a"}},
		{"swap the ends", []string{"a", "b", "c"}, []string{"c", "b", "a"}},
		{"rotate", []string{"a", "b", "c", "d"}, []string{"b", "c", "d", "a"}},
		{"replace all", []string{"a", "b"}, []string{"c", "d"}},
		{"from empty", nil, []string{"a", "b"}},
		{"to empty", []string{"a", "b"}, nil},
		{"unkeyed grow", []string{"", ""}, []string{"", "", ""}},
		{"unkeyed shrink", []string{"", "", ""}, []string{""}},
		{"keyed among unkeyed", []string{"a", "", "b"}, []string{"b", "", "a"}},
		{"keyed moves past unkeyed", []string{"", "a", ""}, []string{"a", "", ""}},
		{"same", []string{"a", "", "b"}, []string{"a", "", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := newTree(t)
			tr.pump(newMulti(keyedLeaves(tc.old)...))
			parent := elementOfType[*multi](t, tr).(*multiElement)
			before := map[string]Element{}
			for i, c := range parent.children {
				if tc.old[i] != "" {
					before[tc.old[i]] = c
				}
			}
			oldElements := slices.Clone(parent.children)

			tr.pump(newMulti(keyedLeaves(tc.new)...))

			if got := len(parent.children); got != len(tc.new) {
				t.Fatalf("%d children, want %d", got, len(tc.new))
			}
			for i, c := range parent.children {
				if got := keyName(c.AsElement().widget); got != tc.new[i] {
					t.Errorf("child %d has key %q, want %q", i, got, tc.new[i])
				}
				if tc.new[i] != "" {
					if prev, ok := before[tc.new[i]]; ok && prev != c {
						t.Errorf("child %d (%q) is not the element it was before", i, tc.new[i])
					}
					if _, ok := before[tc.new[i]]; !ok && slices.Contains(oldElements, c) {
						t.Errorf("child %d (%q) reuses an element of another key", i, tc.new[i])
					}
				}
				if !c.AsElement().DebugIsActiveForTest() {
					t.Errorf("child %d is not active", i)
				}
			}
			// Every old element that is not a child any more is gone for good
			// after the frame is finalized.
			for _, e := range oldElements {
				if !slices.Contains(parent.children, e) && e.AsElement().lifecycle != lifecycleDefunct {
					t.Errorf("a removed child is %v, want defunct", e.AsElement().lifecycle)
				}
			}
		})
	}
}

func TestUpdateChildrenSlots(t *testing.T) {
	tr := newTree(t)
	tr.pump(newMulti(keyedLeaves([]string{"a", "b", "c"})...))
	parent := elementOfType[*multi](t, tr).(*multiElement)
	tr.pump(newMulti(keyedLeaves([]string{"c", "b", "a"})...))
	var prev Element
	for i, c := range parent.children {
		want := IndexedSlot{Index: i, Value: prev}
		if got := c.Slot(); got != any(want) {
			t.Errorf("slot of child %d = %v, want %v", i, got, want)
		}
		prev = c
	}
}

func TestUpdateChildrenWithGivenSlots(t *testing.T) {
	tr := newTree(t)
	tr.pump(newMulti(keyedLeaves([]string{"a", "b"})...))
	parent := elementOfType[*multi](t, tr).(*multiElement)
	slots := []any{"x", "y", "z"}
	parent.children = parent.UpdateChildren(parent.children, keyedLeaves([]string{"a", "b", "c"}), UpdateChildrenOpts{Slots: slots})
	for i, c := range parent.children {
		if got := c.Slot(); got != slots[i] {
			t.Errorf("slot of child %d = %v, want %v", i, got, slots[i])
		}
	}
}

func TestUpdateChildrenDifferentTypeSameKey(t *testing.T) {
	tr := newTree(t)
	k := foundation.NewKey("a")
	tr.pump(newMulti(newLeaf(WidgetOpts{Key: k})))
	parent := elementOfType[*multi](t, tr).(*multiElement)
	old := parent.children[0]
	tr.pump(newMulti(&otherLeaf{NewWidgetBase(WidgetOpts{Key: k})}))
	if parent.children[0] == old {
		t.Error("an element was reused for a widget of another type")
	}
	if old.AsElement().lifecycle != lifecycleDefunct {
		t.Error("the replaced element was not unmounted")
	}
}

func TestUpdateChildrenForgottenChildren(t *testing.T) {
	tr := newTree(t)
	tr.pump(newMulti(keyedLeaves([]string{"a", "b", "c"})...))
	parent := elementOfType[*multi](t, tr).(*multiElement)
	forgotten := parent.children[1]
	// The forgotten child moved to another parent: it must not be updated or
	// deactivated here, and the new list gets a new element in its place.
	parent.children = parent.UpdateChildren(parent.children, keyedLeaves([]string{"a", "b", "c"}),
		UpdateChildrenOpts{ForgottenChildren: map[Element]struct{}{forgotten: {}}})
	if parent.children[1] == forgotten {
		t.Error("the forgotten child was reused")
	}
	if !forgotten.AsElement().DebugIsActiveForTest() {
		t.Errorf("the forgotten child is %v, want active", forgotten.AsElement().lifecycle)
	}
}

func TestUpdateChildNilCases(t *testing.T) {
	tr := newTree(t)
	tr.pump(newWrapper(newLeaf()))
	parent := elementOfType[*wrapper](t, tr).(*StatelessElement)
	if got := parent.UpdateChild(nil, nil, nil); got != nil {
		t.Errorf("UpdateChild(nil, nil) = %v, want nil", got)
	}
	child := parent.child
	if got := parent.UpdateChild(child, nil, nil); got != nil {
		t.Errorf("UpdateChild(child, nil) = %v, want nil", got)
	}
	if child.AsElement().lifecycle != lifecycleInactive {
		t.Errorf("the removed child is %v, want inactive", child.AsElement().lifecycle)
	}
	if child.AsElement().Parent() != nil {
		t.Error("the removed child still has a parent")
	}
}

func TestUpdateChildSameWidgetKeepsElement(t *testing.T) {
	var builds int
	tr := newTree(t)
	inner := newSpy(nil)
	inner.onBuild = func(BuildContext) { builds++ }
	tr.pump(newWrapper(inner))
	tr.pump(newWrapper(inner)) // the same widget again: nothing below is rebuilt
	if builds != 1 {
		t.Errorf("inner built %d times, want 1", builds)
	}
}

func TestElementDepth(t *testing.T) {
	skipUnlessDebug(t)
	e := CreateElement(newWrapper(newLeaf()))
	requireTag(t, "ElementDepthBeforeMount", func() { e.Depth() })
	tr := newTree(t)
	tr.pump(newWrapper(newWrapper(newLeaf())))
	for i, e := range []Element{
		findByType[*wrapper](tr)[0],
		tr.find(func(w Widget) bool { _, ok := w.(*leaf); return ok })[0],
	} {
		if got, want := e.Depth(), 2+i*2; got != want {
			t.Errorf("depth %d = %d, want %d", i, got, want)
		}
	}
}

func TestVisitChildElementsDuringBuild(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	w := newBuilder(func(bc BuildContext, _ func(func())) Widget {
		bc.VisitChildElements(func(Element) {})
		return newLeaf()
	})
	requireTag(t, "VisitChildElementsDuringBuild", func() { tr.pump(w) })
}

func TestFindRenderObjectInactive(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newSpy(nil))
	e := elementOfType[*spy](t, tr)
	if ro := e.FindRenderObject(); ro != nil {
		t.Errorf("an element tree without render objects found %v", ro)
	}
	tr.pump(newLeaf())
	requireTag(t, "FindRenderObjectInactive", func() { e.FindRenderObject() })
}

func TestAncestorLookups(t *testing.T) {
	var found struct {
		stateful, root *builderState
		widget         *wrapper
		wOK, sOK, rOK  bool
	}
	tr := newTree(t)
	innerWrapper := newWrapper(nil)
	tr.pump(newBuilder(func(BuildContext, func(func())) Widget {
		return newBuilder(func(BuildContext, func(func())) Widget {
			innerWrapper.child = newBuilder(func(bc BuildContext, _ func(func())) Widget {
				found.widget, found.wOK = FindAncestorWidgetOfExactType[*wrapper](bc)
				found.stateful, found.sOK = FindAncestorStateOfType[*builderState](bc)
				found.root, found.rOK = FindRootAncestorStateOfType[*builderState](bc)
				return newLeaf()
			})
			return innerWrapper
		})
	}))
	if !found.wOK || found.widget != innerWrapper {
		t.Errorf("FindAncestorWidgetOfExactType = %v, %v", found.widget, found.wOK)
	}
	if !found.sOK || !found.rOK || found.stateful == found.root {
		t.Errorf("the closest and the root ancestor state must differ: %v %v", found.stateful, found.root)
	}
	if got := found.root.Widget(); got != tr.find(func(w Widget) bool { _, ok := w.(*builder); return ok })[0].AsElement().widget {
		t.Error("the root ancestor state is not the outermost builder")
	}
}

func TestAncestorLookupOnInactiveElement(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	var inDispose BuildContext
	w := newSpy(nil)
	w.onDispose = func(bc BuildContext) { inDispose = bc }
	tr.pump(newWrapper(w))
	tr.pump(newLeaf())
	requireTag(t, "AncestorLookupOnInactiveElement", func() { inDispose.VisitAncestorElements(func(Element) bool { return true }) })
	requireTag(t, "AncestorLookupOnInactiveElement", func() { FindAncestorWidgetOfExactType[*wrapper](inDispose) })
}

func TestVisitAncestorElementsStopsWhenToldTo(t *testing.T) {
	tr := newTree(t)
	var visited []string
	tr.pump(newWrapper(newWrapper(newBuilder(func(bc BuildContext, _ func(func())) Widget {
		bc.VisitAncestorElements(func(e Element) bool {
			visited = append(visited, foundation.RuntimeTypeName(e.AsElement().widget))
			return len(visited) < 2
		})
		return newLeaf()
	}))))
	if want := []string{"wrapper", "wrapper"}; !slices.Equal(visited, want) {
		t.Errorf("visited %v, want %v", visited, want)
	}
}

func TestDescribeMissingAncestor(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	var nodes []foundation.DiagnosticsNode
	tr.pump(newWrapper(newBuilder(func(bc BuildContext, _ func(func())) Widget {
		nodes = bc.DescribeMissingAncestor("Theme")
		return newLeaf()
	})))
	if len(nodes) != 2 {
		t.Fatalf("%d nodes, want 2", len(nodes))
	}
	if got := nodes[0].String(); !strings.Contains(got, "could not find a Theme ancestor was") {
		t.Errorf("first node = %q", got)
	}
	if got := nodes[1].String(); !strings.Contains(got, "ancestors of this widget were") {
		t.Errorf("second node = %q", got)
	}
}

func TestDescribeMissingAncestorOfRoot(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newLeaf()))
	e := elementOfType[*wrapper](t, tr)
	e.AsElement().parent = nil // the root of its own tree
	nodes := e.DescribeMissingAncestor("Theme")
	if len(nodes) != 2 || !strings.Contains(nodes[1].String(), "This widget is the root of the tree") {
		t.Errorf("nodes = %v", nodes)
	}
}

func TestElementDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newMulti(
		newLeaf(),
		newLeaf(WidgetOpts{Key: foundation.NewKey("k")}),
		newBuilder(func(BuildContext, func(func())) Widget { return newLeaf() }),
	)))
	got := normalizeHashes(foundation.ToStringDeep(elementOfType[*wrapper](t, tr)))
	for _, line := range []string{
		"wrapper#00000",
		"└multi#00000",
		" ├leaf#00000",
		" ├leaf#00000-[<'k'>]",
		" └builder#00000(state: builderState#00000)",
		"  └leaf#00000",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("the dump does not contain %q:\n%s", line, got)
		}
	}
	if strings.Contains(got, "dirty") {
		t.Errorf("a built tree has dirty elements:\n%s", got)
	}
}

func TestElementToStringShort(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newLeaf(WidgetOpts{Key: foundation.NewKey("k")}))
	e := tr.find(func(w Widget) bool { return true })[0]
	if got := normalizeHashes(e.ToStringShort()); got != "leaf#00000-[<'k'>]" {
		t.Errorf("ToStringShort = %q", got)
	}
	tr.pump(newMulti())
	if got := normalizeHashes(e.ToStringShort()); got != "leafElement#00000(DEFUNCT)" {
		t.Errorf("ToStringShort of an unmounted element = %q", got)
	}
}

func TestDebugGetCreatorChain(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newWrapper(newLeaf())))
	e := tr.find(func(w Widget) bool { _, ok := w.(*leaf); return ok })[0]
	got := normalizeHashes(e.AsElement().DebugGetCreatorChain(2))
	if want := "leaf#00000 \u2190 wrapper#00000 \u2190 \u22EF"; got != want {
		t.Errorf("chain = %q, want %q", got, want)
	}
	if n := len(e.AsElement().DebugGetDiagnosticChain()); n != 4 {
		t.Errorf("diagnostic chain has %d elements, want 4", n)
	}
}

func TestPanicInDeactivateLeavesElementInactive(t *testing.T) {
	// 'widget is not active if throw in deactivated'
	tr := newTree(t)
	inner := newLeaf()
	w := newSpy(inner)
	w.onDeactivate = func(BuildContext) { panic("kaboom") }
	tr.pump(w)
	el := tr.find(func(w Widget) bool { return w == Widget(inner) })[0]
	if v := panicOf(func() { tr.pump(newMulti()) }); v != "kaboom" {
		t.Fatalf("panic = %v, want kaboom", v)
	}
	if el.AsElement().DebugIsActiveForTest() {
		t.Error("the element of the failed subtree is still active")
	}
	if el.AsElement().lifecycle == lifecycleDefunct {
		t.Error("the element of the failed subtree is defunct")
	}
}

func TestPanicInDisposeStillUnmounts(t *testing.T) {
	// 'widget is unmounted if throw in dispose'
	tr := newTree(t)
	inner := newLeaf()
	w := newSpy(inner)
	w.onDispose = func(BuildContext) { panic("kaboom") }
	tr.pump(w)
	el := tr.find(func(w Widget) bool { return w == Widget(inner) })[0]
	if v := panicOf(func() { tr.pump(newMulti()) }); v != "kaboom" {
		t.Fatalf("panic = %v, want kaboom", v)
	}
	if el.AsElement().lifecycle != lifecycleDefunct {
		t.Errorf("the element is %v, want defunct", el.AsElement().lifecycle)
	}
}

func TestPanicInActivateLeavesElementInactive(t *testing.T) {
	// 'widget is not active if throw in activated'
	tr := newTree(t)
	inner := newLeaf()
	w := newSpy(inner)
	w.onActivate = func(BuildContext) { panic("kaboom") }
	first, second := newMulti(w), newMulti()
	tr.pump(newMulti(first, second))
	from := tr.find(func(x Widget) bool { return x == Widget(first) })[0].(*multiElement)
	to := tr.find(func(x Widget) bool { return x == Widget(second) })[0].(*multiElement)
	spyEl := from.children[0]
	child := tr.find(func(x Widget) bool { return x == Widget(inner) })[0]
	from.ForgetChild(spyEl)
	from.DeactivateChild(spyEl)
	tr.owner.inactive.remove(spyEl)
	v := panicOf(func() { spyEl.AsElement().activateWithParent(to, nil) })
	if v != "kaboom" {
		t.Fatalf("panic = %v, want kaboom", v)
	}
	if child.AsElement().DebugIsActiveForTest() || child.AsElement().lifecycle == lifecycleDefunct {
		t.Errorf("the child is %v, want inactive", child.AsElement().lifecycle)
	}
}

func TestInflateFailureCleansUp(t *testing.T) {
	tr := newTree(t)
	boom := newBuilder(func(BuildContext, func(func())) Widget { panic("kaboom") })
	tr.pump(newMulti())
	parent := elementOfType[*multi](t, tr).(*multiElement)
	v := panicOf(func() { parent.InflateWidget(boom, nil) })
	if v != "kaboom" {
		t.Fatalf("panic = %v, want kaboom", v)
	}
	// A widget that cannot be inflated panics before anything is mounted.
	requireTag(t, "CreateElement.WidgetKindUnknown", func() { parent.InflateWidget(&badWidget{}, nil) })
}
