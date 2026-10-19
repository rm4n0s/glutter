package widgets

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// flexChildren returns the render objects of the children of a render flex.
func flexChildren(flex *rendering.RenderFlex) []rendering.RenderObject {
	var out []rendering.RenderObject
	flex.VisitChildren(func(c rendering.RenderObject) { out = append(out, c) })
	return out
}

func columnRenderObject(t *testing.T, rt *renderTree) *rendering.RenderFlex {
	t.Helper()
	return renderElementOfType[*column](t, rt).RenderObject().(*rendering.RenderFlex)
}

func keyedBoxes(names ...string) []Widget {
	ws := make([]Widget, len(names))
	for i, n := range names {
		ws[i] = newBox(10, float64(10+i), WidgetOpts{Key: foundation.NewKey(n)})
	}
	return ws
}

func TestRenderObjectsAreCreatedOnceAndUpdated(t *testing.T) {
	var created int
	mk := func(w float64) Widget {
		b := newBox(w, 20)
		b.created = &created
		return newCenter(b)
	}
	rt := newRenderTree(t)
	rt.pump(mk(10))
	boxEl := renderElementOfType[*box](t, rt)
	ro := boxEl.RenderObject().(*rendering.RenderConstrainedBox)
	if created != 1 {
		t.Fatalf("created %d render objects, want 1", created)
	}
	if size, ok := boxEl.Size(); !ok || size != (ui.Size{Width: 10, Height: 20}) {
		t.Errorf("Size = %v, %v; want 10x20", size, ok)
	}

	rt.pump(mk(30))
	if created != 1 || boxEl.RenderObject() != rendering.RenderObject(ro) {
		t.Errorf("an update created a new render object (created %d)", created)
	}
	if size, _ := boxEl.Size(); size != (ui.Size{Width: 30, Height: 20}) {
		t.Errorf("Size after the update = %v, want 30x20", size)
	}
}

func TestRenderObjectTreeFollowsTheElementTree(t *testing.T) {
	rt := newRenderTree(t)
	rt.pump(newCenter(newColumn(keyedBoxes("a", "b", "c")...)))
	flex := columnRenderObject(t, rt)
	if got := len(flexChildren(flex)); got != 3 {
		t.Fatalf("flex has %d children, want 3", got)
	}
	if flex.Parent() == nil {
		t.Error("the column's render object has no parent")
	}
	if !flex.Attached() {
		t.Error("the column's render object is not attached")
	}
	// A component element in between keeps the render object in the right place.
	rt.pump(newCenter(newColumn(newWrapper(newBox(5, 5)), newBox(6, 6))))
	if got := len(flexChildren(columnRenderObject(t, rt))); got != 2 {
		t.Errorf("flex has %d children, want 2", got)
	}
}

func TestMultiChildMovesInsertsAndRemovesRenderObjects(t *testing.T) {
	order := func(rt *renderTree, el Element) []rendering.RenderObject {
		return flexChildren(el.RenderObject().(*rendering.RenderFlex))
	}
	tests := []struct {
		name     string
		old, new []string
	}{
		{"reverse", []string{"a", "b", "c"}, []string{"c", "b", "a"}},
		{"insert first", []string{"a", "b"}, []string{"x", "a", "b"}},
		{"insert middle", []string{"a", "b"}, []string{"a", "x", "b"}},
		{"insert last", []string{"a", "b"}, []string{"a", "b", "x"}},
		{"remove first", []string{"a", "b", "c"}, []string{"b", "c"}},
		{"remove middle", []string{"a", "b", "c"}, []string{"a", "c"}},
		{"remove last", []string{"a", "b", "c"}, []string{"a", "b"}},
		{"move and add", []string{"a", "b", "c", "d"}, []string{"d", "x", "a", "c"}},
		{"replace all", []string{"a", "b"}, []string{"x", "y"}},
		{"empty", []string{"a", "b"}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRenderTree(t)
			rt.pump(newColumn(keyedBoxes(tc.old...)...))
			col := renderElementOfType[*column](t, rt)
			before := map[string]rendering.RenderObject{}
			for i, name := range tc.old {
				before[name] = order(rt, col)[i]
			}
			rt.pump(newColumn(keyedBoxes(tc.new...)...))
			got := order(rt, col)
			if len(got) != len(tc.new) {
				t.Fatalf("%d render children, want %d", len(got), len(tc.new))
			}
			for i, name := range tc.new {
				if old, ok := before[name]; ok && got[i] != old {
					t.Errorf("render object %d (%q) is not the one it was before", i, name)
				}
			}
			// The element order and the render order agree.
			var fromElements []rendering.RenderObject
			col.VisitChildren(func(c Element) { fromElements = append(fromElements, c.RenderObject()) })
			if !slices.Equal(fromElements, got) {
				t.Errorf("the order of the render objects differs from the order of the elements")
			}
		})
	}
}

func TestUnmountDisposesRenderObject(t *testing.T) {
	var unmounted []rendering.RenderObject
	w := newSized(10, 10, nil)
	w.unmounted = &unmounted
	rt := newRenderTree(t)
	rt.pump(newColumn(w))
	ro := renderElementOfType[*sized](t, rt).RenderObject()
	rt.pump(newColumn())
	if len(unmounted) != 1 || unmounted[0] != ro {
		t.Fatalf("DidUnmountRenderObject got %v, want the render object", unmounted)
	}
	if foundation.DebugMode && !ro.AsRenderObject().DebugDisposed() {
		t.Error("the render object was not disposed")
	}
	if ro.Attached() {
		t.Error("the render object is still attached")
	}
}

func TestRenderObjectElementDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	rt.pump(newColumn(newBox(10, 10)))
	got := normalizeHashes(foundation.ToStringDeep(renderElementOfType[*column](t, rt)))
	for _, want := range []string{"column(renderObject: RenderFlex#00000)", "└box(renderObject: RenderConstrainedBox#00000", "renderObject: RenderConstrainedBox#00000"} {
		if !strings.Contains(got, want) {
			t.Errorf("the dump does not contain %q:\n%s", want, got)
		}
	}
	// A render object element that was never mounted has no render object.
	b := foundation.NewDiagnosticPropertiesBuilder()
	NewLeafRenderObjectElement(newBox(1, 1)).DebugFillProperties(b)
	found := false
	for _, p := range b.Properties {
		if p.Name().OrElse("") == "renderObject" {
			found = true
		}
	}
	if !found {
		t.Error("an element without a render object has no renderObject property")
	}
}

func TestDuplicateKeysInChildren(t *testing.T) {
	skipUnlessDebug(t)
	dup := func() []Widget { return keyedBoxes("a", "a") }
	t.Run("on creation", func(t *testing.T) {
		requireTag(t, "ChildrenHaveDuplicateKeys", func() { NewMultiChildRenderObjectElement(newColumn(dup()...)) })
	})
	t.Run("on update", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newColumn(keyedBoxes("a", "b")...))
		requireTag(t, "ChildrenHaveDuplicateKeys", func() { rt.pump(newColumn(dup()...)) })
	})
	t.Run("items", func(t *testing.T) {
		requireTag(t, "ItemsHaveDuplicateKeys", func() { DebugItemsHaveDuplicateKeys(dup()) })
		if DebugItemsHaveDuplicateKeys(keyedBoxes("a", "b")) {
			t.Error("unique keys are reported as duplicate")
		}
	})
	t.Run("different keys and unkeyed are fine", func(t *testing.T) {
		if DebugChildrenHaveDuplicateKeys(newColumn(), append(keyedBoxes("a", "b"), newBox(1, 1), newBox(1, 1))) {
			t.Error("reported duplicates")
		}
	})
}

func TestMultiChildNeedsRenderObject(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	rt.pump(newColumn(newWrapper(newLeaf())))
	_ = requireErrorTag(t, rt.takeError(), "MultiChildNeedsRenderObject")
}

func TestRenderObjectAncestorMissing(t *testing.T) {
	skipUnlessDebug(t)
	// The component-only harness root has no render object to attach to.
	tr := newTree(t)
	tr.pump(newBox(10, 10))
	appErr := requireErrorTag(t, tr.takeError(), "RenderObjectAncestorMissing")
	if !strings.Contains(appErr.Message, "cannot find an ancestor render object") {
		t.Errorf("message = %q", appErr.Message)
	}
}

func TestRenderTreeRootCannotAttachToAnAncestor(t *testing.T) {
	skipUnlessDebug(t)
	rt := newRenderTree(t)
	rt.pump(newSized(10, 10, nil))
	root := rt.stack
	inner := &viewRootWidget{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(nil), view: root.RenderView}
	requireTag(t, "RenderTreeRootHasAncestor", func() { rt.pump(newSized(10, 10, inner)) })
}

func TestLeafAndSingleChildRenderObjectElementsRefuseWrongOperations(t *testing.T) {
	leaf := NewLeafRenderObjectElement(newBox(1, 1))
	requireTag(t, "LeafRenderObjectHasNoChildren", func() { leaf.InsertRenderObjectChild(nil, nil) })
	requireTag(t, "LeafRenderObjectHasNoChildren", func() { leaf.MoveRenderObjectChild(nil, nil, nil) })
	requireTag(t, "LeafRenderObjectHasNoChildren", func() { leaf.RemoveRenderObjectChild(nil, nil) })
	requireTag(t, "LeafRenderObjectHasNoChildren", func() { leaf.ForgetChild(nil) })
	single := NewSingleChildRenderObjectElement(newSized(1, 1, nil))
	requireTag(t, "SingleChildRenderObjectMove", func() { single.MoveRenderObjectChild(nil, nil, nil) })
}

func TestFindAncestorRenderObject(t *testing.T) {
	rt := newRenderTree(t)
	var found *rendering.RenderFlex
	var foundBox *rendering.RenderConstrainedBox
	var okFlex, okBox bool
	rt.pump(newColumn(newWrapper(newBuilder(func(bc BuildContext, _ func(func())) Widget {
		found, okFlex = FindAncestorRenderObjectOfType[*rendering.RenderFlex](bc)
		foundBox, okBox = FindAncestorRenderObjectOfType[*rendering.RenderConstrainedBox](bc)
		return newBox(1, 1)
	}))))
	if !okFlex || found != columnRenderObject(t, rt) {
		t.Errorf("FindAncestorRenderObjectOfType[RenderFlex] = %v, %v", found, okFlex)
	}
	if okBox {
		t.Errorf("found a RenderConstrainedBox ancestor: %v", foundBox)
	}
}

func TestSizeErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("of an inactive element", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newBox(1, 1))
		e := renderElementOfType[*box](t, rt)
		rt.pump(newColumn())
		requireTag(t, "SizeOfInactiveElement", func() { e.Size() })
	})
	t.Run("during build", func(t *testing.T) {
		rt := newRenderTree(t)
		w := newBuilder(func(bc BuildContext, _ func(func())) Widget {
			bc.Size()
			return newBox(1, 1)
		})
		requireTag(t, "SizeDuringBuild", func() { rt.pump(w) })
	})
	t.Run("without a render object", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newWrapper(newLeaf()))
		requireTag(t, "SizeWithoutRenderObject", func() { renderElementOfType[*wrapper](t, rt).Size() })
	})
	t.Run("of a render object that is not a box", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newSized(1, 1, nil))
		// The view root's render object is the render view, which is not a box.
		requireTag(t, "SizeOfNonBox", func() { rt.root.Size() })
	})
	t.Run("before layout", func(t *testing.T) {
		rt := newRenderTree(t)
		rw := &viewRootWidget{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(newBox(1, 1)), view: rt.stack.RenderView}
		rt.root = rw.CreateElement().(*viewRootElement)
		rt.root.AssignOwner(rt.owner)
		rt.owner.BuildScope(rt.root, func() { rt.root.Mount(nil, nil) })
		rt.owner.FinalizeTree()
		requireTag(t, "SizeBeforeLayout", func() { renderElementOfType[*box](t, rt).Size() })
	})
	t.Run("after being marked dirty", func(t *testing.T) {
		rt := newRenderTree(t)
		rt.pump(newBox(1, 1))
		e := renderElementOfType[*box](t, rt)
		e.RenderObject().MarkNeedsLayout()
		requireTag(t, "SizeOfDirtyLayout", func() { e.Size() })
	})
}
