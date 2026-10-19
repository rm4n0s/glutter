package rendering_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// keepAliveManager marks the children whose index is in keep as keep alive when
// the adaptor adopts them.
type keepAliveManager struct {
	*testSliverBoxChildManager
	keep map[int]bool
}

func (m *keepAliveManager) DidAdoptChild(child rendering.Box) {
	m.testSliverBoxChildManager.DidAdoptChild(child)
	data := rendering.SliverMultiBoxAdaptorParentDataOf(child)
	data.KeepAlive = m.keep[data.Index.Value()]
}

func newKeepAliveList(keep ...int) (*rendering.RenderSliverList, *keepAliveManager, []*RenderSizedBox) {
	boxes := sizedBoxes(10, 300, 100)
	inner := newTestChildManager(asBoxes(boxes)...)
	manager := &keepAliveManager{testSliverBoxChildManager: inner, keep: map[int]bool{}}
	for _, k := range keep {
		manager.keep[k] = true
	}
	list := rendering.NewRenderSliverList(manager)
	inner.insert, inner.remove = list.Insert, list.Remove
	return list, manager, boxes
}

func TestSliverListKeepsChildrenAlive(t *testing.T) {
	list, _, boxes := newKeepAliveList(0)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), list)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	if !boxes[0].Attached() || !list.PaintsChild(boxes[0]) {
		t.Fatal("item 0 should be built and painted")
	}
	if rendering.SliverMultiBoxAdaptorParentDataOf(boxes[0]).KeptAlive() {
		t.Error("item 0 is kept alive while it is visible")
	}
	root.SetOffset(rendering.NewViewportOffsetFixed(500))
	pumpFrame(h, phaseLayout)
	// item 1 is not kept alive, so it is gone; item 0 is kept alive
	if boxes[1].Attached() {
		t.Error("item 1 should have been removed")
	}
	if !boxes[0].Attached() {
		t.Fatal("item 0 should be kept alive")
	}
	data := rendering.SliverMultiBoxAdaptorParentDataOf(boxes[0])
	if !data.KeptAlive() || !data.KeepAlive {
		t.Errorf("item 0 keptAlive %v keepAlive %v, want both true", data.KeptAlive(), data.KeepAlive)
	}
	if list.PaintsChild(boxes[0]) {
		t.Error("a kept alive item that is not live is painted")
	}
	// the kept alive child is visited, in the semantics it is not
	var visited, semantic int
	list.VisitChildren(func(rendering.RenderObject) { visited++ })
	list.VisitChildrenForSemantics(func(rendering.RenderObject) { semantic++ })
	if visited != semantic+1 {
		t.Errorf("visited %d children and %d for semantics, want one more", visited, semantic)
	}
	m := ui.NewMatrix4Identity()
	list.ApplyPaintTransform(boxes[0], m)
	if *m.Storage() != [16]float32{} {
		t.Errorf("the paint transform of a kept alive child = %v, want zero", m)
	}
	// it comes back when scrolling back
	root.SetOffset(rendering.NewViewportOffsetFixed(0))
	pumpFrame(h, phaseLayout)
	if !boxes[0].Attached() || !list.PaintsChild(boxes[0]) || data.KeptAlive() {
		t.Errorf("item 0 attached %v, painted %v, kept alive %v; want true, true, false", boxes[0].Attached(), list.PaintsChild(boxes[0]), data.KeptAlive())
	}
	// when the child stops asking, the adaptor lets it go
	root.SetOffset(rendering.NewViewportOffsetFixed(500))
	pumpFrame(h, phaseLayout)
	data.KeepAlive = false
	list.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	if boxes[0].Attached() {
		t.Error("item 0 should be removed once it does not ask to be kept alive")
	}
}

func TestSliverListKeepAliveChildrenInDescription(t *testing.T) {
	skipUnlessDebug(t)
	list, _, boxes := newKeepAliveList(0)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), list)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	root.SetOffset(rendering.NewViewportOffsetFixed(500))
	pumpFrame(h, phaseLayout)
	got := deepInfo(list)
	for _, want := range []string{
		"currently live children: 4 to 6",
		"child with index 0 (kept alive but not laid out)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("description lacks %q:\n%s", want, got)
		}
	}
	_ = boxes
}

func TestSliverListDetachesKeptAliveChildren(t *testing.T) {
	list, _, boxes := newKeepAliveList(0)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), list)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	root.SetOffset(rendering.NewViewportOffsetFixed(500))
	pumpFrame(h, phaseLayout)
	if !boxes[0].Attached() {
		t.Fatal("item 0 should be kept alive")
	}
	h.SetRoot(nil)
	if boxes[0].Attached() {
		t.Error("a kept alive child stays attached after the sliver is detached")
	}
}

func TestSliverListKeepAliveFailures(t *testing.T) {
	skipUnlessDebug(t)
	scrolledAway := func(t *testing.T, keep ...int) (*rendering.RenderSliverList, *keepAliveManager, []*RenderSizedBox) {
		list, manager, boxes := newKeepAliveList(keep...)
		root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), list)
		h := newHarness(t)
		layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
		root.SetOffset(rendering.NewViewportOffsetFixed(800))
		pumpFrame(h, phaseLayout)
		return list, manager, boxes
	}
	t.Run("inserting a kept alive child", func(t *testing.T) {
		list, _, boxes := scrolledAway(t, 0)
		requireTag(t, "SliverInsertKeptAlive", func() { list.Insert(boxes[0], nil) })
	})
	t.Run("dangling kept alive children", func(t *testing.T) {
		list, manager, boxes := scrolledAway(t, 0, 1)
		// the child of the index 1 moves to the index 0, where another one is kept
		manager.currentlyUpdatingIndex = foundation.Some(0)
		list.Move(boxes[1], nil)
		requireTag(t, "SliverDanglingKeepAlives", func() { list.SetDebugChildIntegrityEnabled(true) })
	})
	t.Run("children out of order", func(t *testing.T) {
		list, manager, boxes := newKeepAliveList()
		manager.currentlyUpdatingIndex = foundation.Some(5)
		list.Insert(boxes[5], nil)
		manager.currentlyUpdatingIndex = foundation.Some(3)
		requireTag(t, "SliverChildOrderInvalid", func() { list.Insert(boxes[3], boxes[5]) })
		// the same check runs when the integrity checks are turned on again
		list.SetDebugChildIntegrityEnabled(false)
		requireTag(t, "SliverChildOrderInvalid", func() { list.SetDebugChildIntegrityEnabled(true) })
	})
	t.Run("children that are not contiguous", func(t *testing.T) {
		list, manager, boxes := newKeepAliveList()
		list.SetDebugChildIntegrityEnabled(false)
		manager.currentlyUpdatingIndex = foundation.Some(1)
		list.Insert(boxes[1], nil)
		manager.currentlyUpdatingIndex = foundation.Some(3)
		list.Insert(boxes[3], boxes[1])
		requireTag(t, "SliverChildListInvalid", func() { list.DebugAssertChildListIsNonEmptyAndContiguous() })
	})
}
