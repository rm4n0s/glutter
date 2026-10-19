package widgets

import (
	"slices"
	"testing"
)

// scopeWidget is _CustomBuildScopeWidget of framework_test.dart: a widget whose
// element builds in a build scope of its own and walks its child only when it
// is mounted.
type scopeWidget struct {
	WidgetBase
	scope *BuildScope
	child Widget
}

type scopeElement struct {
	ElementBase
	child Element
}

func (w *scopeWidget) CreateElement() Element {
	e := &scopeElement{}
	InitElement(&e.ElementBase, e, w)
	return e
}

func (e *scopeElement) BuildScope() *BuildScope {
	if s := e.widget.(*scopeWidget).scope; s != nil {
		return s
	}
	return e.ElementBase.BuildScope()
}

func (e *scopeElement) Mount(parent Element, slot any) {
	e.ElementBase.Mount(parent, slot)
	e.Rebuild(RebuildOpts{Force: true})
	e.child = e.UpdateChild(e.child, e.widget.(*scopeWidget).child, e.slot)
}

func (e *scopeElement) VisitChildren(v ElementVisitor) {
	if e.child != nil {
		v(e.child)
	}
}

func TestBuildScopeSegregatesDirtyElements(t *testing.T) {
	scope := NewBuildScope()
	tr := newTree(t)
	tr.pump(newBuilder(func(BuildContext, func(func())) Widget {
		return &scopeWidget{scope: scope, child: newLeaf()}
	}))
	root := elementOfType[*builder](t, tr)
	scopeEl := elementOfType[*scopeWidget](t, tr)
	leafEl := elementOfType[*leaf](t, tr)
	dirty := func() [3]bool { return [3]bool{root.Dirty(), scopeEl.Dirty(), leafEl.Dirty()} }
	check := func(step string, want [3]bool) {
		t.Helper()
		if got := dirty(); got != want {
			t.Errorf("%s: dirty (root, scope, leaf) = %v, want %v", step, got, want)
		}
	}
	check("after the first build", [3]bool{})

	root.MarkNeedsBuild()
	tr.pumpFrame()
	check("root marked, frame pumped", [3]bool{})

	// The frame builds the root scope only, so the element of the other scope
	// stays dirty.
	scopeEl.MarkNeedsBuild()
	tr.pumpFrame()
	check("scope element marked, frame pumped", [3]bool{false, true, false})

	tr.owner.BuildScope(scopeEl, nil)
	check("scope built", [3]bool{})

	leafEl.MarkNeedsBuild()
	tr.pumpFrame()
	check("leaf marked, frame pumped", [3]bool{false, false, true})

	tr.owner.BuildScope(scopeEl, nil)
	check("scope built again", [3]bool{})
}

func TestBuildScopeScheduleRebuild(t *testing.T) {
	var scheduled int
	scope := NewBuildScope(BuildScopeOpts{ScheduleRebuild: func() { scheduled++ }})
	tr := newTree(t)
	tr.pump(newMulti(&scopeWidget{scope: scope, child: newLeaf()}))
	leafEl := elementOfType[*leaf](t, tr)
	leafEl.MarkNeedsBuild()
	spyEl := elementOfType[*scopeWidget](t, tr)
	spyEl.MarkNeedsBuild()
	if scheduled != 1 {
		t.Errorf("ScheduleRebuild called %d times, want once for the batch", scheduled)
	}
	tr.owner.BuildScope(spyEl, nil)
	leafEl.MarkNeedsBuild()
	if scheduled != 2 {
		t.Errorf("ScheduleRebuild called %d times after the scope was built, want 2", scheduled)
	}
}

func TestReparentingElementToAnotherBuildScope(t *testing.T) {
	// A dirty element that moves to a subtree of another build scope is not
	// rebuilt by the frame of the root scope.
	scope := NewBuildScope()
	tr := newTree(t)
	moving := newLeaf()
	inScope := &scopeWidget{scope: scope, child: newLeaf()}
	tr.pump(newMulti(inScope, moving))
	multiEl := elementOfType[*multi](t, tr).(*multiElement)
	scopeEl := multiEl.children[0].(*scopeElement)
	el := multiEl.children[1]
	if scopeEl.Dirty() || el.Dirty() {
		t.Fatal("dirty after the first build")
	}

	el.MarkNeedsBuild()
	multiEl.ForgetChild(el)
	multiEl.DeactivateChild(el)
	tr.owner.inactive.remove(el)
	el.AsElement().activateWithParent(scopeEl, nil)
	scopeEl.child.AsElement().parent = nil // the scope element drops its old child
	scopeEl.child = el
	tr.pumpFrame()
	if scopeEl.Dirty() {
		t.Error("the scope element is dirty")
	}
	if !el.Dirty() {
		t.Error("the moved element was rebuilt by the root scope")
	}
	if el.BuildScope() != scope {
		t.Error("the moved element did not take the build scope of its new parent")
	}
	tr.owner.BuildScope(scopeEl, nil)
	if el.Dirty() {
		t.Error("the moved element was not rebuilt by its own scope")
	}
}

func TestDirtyElementsBuildParentsFirst(t *testing.T) {
	var order []string
	named := func(name string, child Widget) *spy {
		w := newSpy(child)
		w.onBuild = func(BuildContext) { order = append(order, name) }
		return w
	}
	tr := newTree(t)
	tr.pump(named("a", named("b", named("c", nil))))
	order = nil

	// Mark them from the deepest to the shallowest; they are built the other
	// way around, and the parents do not rebuild the children (the widgets are
	// the same objects).
	states := map[string]*spyState{}
	for _, e := range findByType[*spy](tr) {
		states[foundationName(e)] = e.(*StatefulElement).State().(*spyState)
	}
	for _, name := range []string{"c", "b", "a"} {
		states[name].SetState(nil)
	}
	tr.pumpFrame()
	if want := []string{"a", "b", "c"}; !slices.Equal(order, want) {
		t.Errorf("build order = %v, want %v", order, want)
	}
}

// foundationName finds out which spy a state belongs to by its depth.
func foundationName(e Element) string { return string(rune('a' + (e.Depth() - 2))) }

func TestDirtyElementIsRebuiltOnce(t *testing.T) {
	var builds int
	tr := newTree(t)
	w := newSpy(nil)
	w.onBuild = func(BuildContext) { builds++ }
	tr.pump(w)
	builds = 0
	s := stateOf[*spyState](t, tr, elementOfType[*spy](t, tr))
	s.SetState(nil)
	s.SetState(nil)
	s.SetState(nil)
	tr.pumpFrame()
	if builds != 1 {
		t.Errorf("built %d times for three SetState calls, want 1", builds)
	}
	tr.pumpFrame()
	if builds != 1 {
		t.Errorf("built %d times by a frame with nothing dirty", builds)
	}
}

func TestOnBuildScheduled(t *testing.T) {
	tr := newTree(t)
	tr.pump(newMulti(newSpy(nil), newSpy(nil)))
	scheduledAfterMount := tr.scheduled
	states := findByType[*spy](tr)
	a := states[0].(*StatefulElement).State().(*spyState)
	b := states[1].(*StatefulElement).State().(*spyState)
	a.SetState(nil)
	b.SetState(nil)
	if got := tr.scheduled - scheduledAfterMount; got != 1 {
		t.Errorf("OnBuildScheduled called %d times for two dirty elements, want 1", got)
	}
	tr.pumpFrame()
	a.SetState(nil)
	if got := tr.scheduled - scheduledAfterMount; got != 2 {
		t.Errorf("OnBuildScheduled called %d times after another frame, want 2", got)
	}
}

func TestScheduleBuildForErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("not dirty", func(t *testing.T) {
		tr := newTree(t)
		tr.pump(newWrapper(newLeaf()))
		e := elementOfType[*wrapper](t, tr)
		requireTag(t, "ScheduleBuildFor.ScheduleBuildForNotDirty", func() { tr.owner.ScheduleBuildFor(e) })
	})
	t.Run("already in the dirty list", func(t *testing.T) {
		// 'Calling scheduleBuildFor on an Element already in dirty list throws'
		tr := newTree(t)
		tr.pump(newWrapper(newLeaf()))
		e := elementOfType[*wrapper](t, tr)
		e.MarkNeedsBuild()
		requireTag(t, "ScheduleBuildFor.ScheduleBuildForInappropriately", func() { tr.owner.ScheduleBuildFor(e) })
	})
}

func TestMarkNeedsBuildWhileLocked(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newLeaf()))
	e := elementOfType[*wrapper](t, tr)
	requireTag(t, "MarkNeedsBuildWhileLocked", func() { tr.owner.LockState(e.MarkNeedsBuild) })
	// The lock is released even when the callback panics.
	e.MarkNeedsBuild()
	if !e.Dirty() {
		t.Error("an element cannot be marked dirty after a lock was released")
	}
}

func TestMarkNeedsBuildOfDescendantsDuringBuild(t *testing.T) {
	// An element below the one being built may be marked dirty; it is built
	// later in the same pass.
	var childState *spyState
	var childBuilds int
	child := newSpy(nil)
	child.onBuild = func(BuildContext) { childBuilds++ }
	parent := newBuilder(func(BuildContext, func(func())) Widget {
		if childState != nil {
			childState.SetState(nil)
		}
		return child
	})
	tr := newTree(t)
	tr.pump(parent)
	childState = stateOf[*spyState](t, tr, elementOfType[*spy](t, tr))
	childBuilds = 0
	stateOf[*builderState](t, tr, elementOfType[*builder](t, tr)).SetState(nil)
	tr.pumpFrame()
	if childBuilds != 1 {
		t.Errorf("the child built %d times, want 1", childBuilds)
	}
}

func TestDebugBuilding(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	var during bool
	tr.pump(newBuilder(func(bc BuildContext, _ func(func())) Widget {
		during = bc.Owner().DebugBuilding() && bc.DebugDoingBuild()
		return newLeaf()
	}))
	if !during {
		t.Error("DebugBuilding and DebugDoingBuild are false during build")
	}
	if tr.owner.DebugBuilding() {
		t.Error("DebugBuilding is true after the build")
	}
	if elementOfType[*builder](t, tr).DebugDoingBuild() {
		t.Error("DebugDoingBuild is true after the build")
	}
}

func TestBuildReturnedNil(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	requireTag(t, "BuildReturnedNil", func() {
		tr.pump(newBuilder(func(BuildContext, func(func())) Widget { return nil }))
	})
}

func TestFinalizeTreeUnmountsInactiveElements(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newWrapper(newSpy(nil)))
	top := elementOfType[*wrapper](t, tr)
	e := elementOfType[*spy](t, tr)

	tr.owner.BuildScope(tr.root, func() { tr.root.Update(&rootWidget{child: newLeaf()}) })
	// Only the top of the removed subtree is on the list; the whole subtree is
	// inactive.
	if !tr.owner.DebugInactiveElementsContain(top) || tr.owner.DebugInactiveElementsContain(e) {
		t.Error("the inactive list must hold the top of the removed subtree only")
	}
	if e.AsElement().lifecycle != lifecycleInactive {
		t.Errorf("before the frame is finalized the element is %v", e.AsElement().lifecycle)
	}
	tr.owner.FinalizeTree()
	if e.AsElement().lifecycle != lifecycleDefunct || tr.owner.DebugInactiveElementsContain(top) {
		t.Errorf("after the frame is finalized the element is %v", e.AsElement().lifecycle)
	}
}

func TestFinalizeTreeUnmountsChildrenBeforeParents(t *testing.T) {
	var order []string
	named := func(name string, child Widget) *spy {
		w := newSpy(child)
		w.onDispose = func(BuildContext) { order = append(order, name) }
		return w
	}
	tr := newTree(t)
	tr.pump(named("outer", named("middle", named("inner", nil))))
	tr.pump(newLeaf())
	if want := []string{"inner", "middle", "outer"}; !slices.Equal(order, want) {
		t.Errorf("dispose order = %v, want %v", order, want)
	}
}
