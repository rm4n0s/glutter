package rendering_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// hitTestViewport hit tests a viewport of 800x600 at the position and returns the
// targets of the hit, from the innermost.
func hitTargets(root rendering.Box, position ui.Offset) []gestures.HitTestTarget {
	result := rendering.NewBoxHitTestResult()
	root.HitTest(result, position)
	var targets []gestures.HitTestTarget
	for _, e := range result.Path() {
		targets = append(targets, e.Target())
	}
	return targets
}

func TestViewportHitTestsSlivers(t *testing.T) {
	list, boxes := listOf(10, 100)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(150), rendering.RenderViewportOpts{Children: []rendering.Sliver{list}})
	layout(t, newHarness(t), root)
	// the item 2 spans 200 to 300 in the list, and the list is scrolled by 150
	targets := hitTargets(root, off(100, 100))
	if len(targets) == 0 || targets[0] != gestures.HitTestTarget(boxes[2]) {
		t.Errorf("hit targets = %v, want to start with item 2", targets)
	}
	if got := hitTargets(root, off(100, 599)); len(got) == 0 || got[0] != gestures.HitTestTarget(boxes[7]) {
		t.Errorf("hit targets at the bottom = %v, want to start with item 7", got)
	}
}

func TestRenderSliverIgnorePointer(t *testing.T) {
	skipUnlessDebug(t)
	list, boxes := listOf(3, 100)
	ignore := rendering.NewRenderSliverIgnorePointer(list)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{ignore}})
	h := newHarness(t)
	layout(t, h, root)
	if !ignore.Ignoring() {
		t.Fatal("ignoring should default to true")
	}
	if got := hitTargets(root, off(100, 50)); len(got) != 0 {
		t.Errorf("hit targets while ignoring = %v, want none", got)
	}
	ignore.SetIgnoring(false)
	if got := hitTargets(root, off(100, 50)); len(got) == 0 || got[0] != gestures.HitTestTarget(boxes[0]) {
		t.Errorf("hit targets while not ignoring = %v, want to start with item 0", got)
	}
	config := semantics.NewSemanticsConfiguration()
	ignore.SetIgnoring(true)
	ignore.DescribeSemanticsConfiguration(config)
	if !config.IsBlockingUserActions {
		t.Error("an ignoring sliver does not block the user actions")
	}
	if got := deepInfo(ignore); got == "" || !strings.Contains(got, "ignoring: true") {
		t.Errorf("description lacks the ignoring flag:\n%s", got)
	}
}

func TestRenderSliverOffstage(t *testing.T) {
	skipUnlessDebug(t)
	list, boxes := listOf(3, 100)
	offstage := rendering.NewRenderSliverOffstage(list, rendering.RenderSliverOffstageOpts{Offstage: foundation.Some(true)})
	after, afterBoxes := listOf(3, 100)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{offstage, after}})
	h := newHarness(t)
	layout(t, h, root)
	if g := offstage.Geometry(); g != rendering.SliverGeometryZero {
		t.Errorf("geometry = %v, want zero", g)
	}
	if !boxes[0].HasSize() {
		t.Error("the child of an offstage sliver is laid out")
	}
	if got := afterBoxes[0].LocalToGlobal(ui.Offset{}, root).Dy; got != 0 {
		t.Errorf("the sliver after an offstage one starts at %v, want 0", got)
	}
	if got := hitTargets(root, off(100, 50)); len(got) == 0 || got[0] != gestures.HitTestTarget(afterBoxes[0]) {
		t.Errorf("hit targets = %v, want to start with the item of the next sliver", got)
	}
	offstage.SetOffstage(false)
	pumpFrame(h, phaseLayout)
	if g := offstage.Geometry(); g.ScrollExtent != 300 {
		t.Errorf("geometry = %v, want a scroll extent of 300", g)
	}
	if got := afterBoxes[0].LocalToGlobal(ui.Offset{}, root).Dy; got != 300 {
		t.Errorf("the sliver after a shown one starts at %v, want 300", got)
	}
	if got := hitTargets(root, off(100, 50)); len(got) == 0 || got[0] != gestures.HitTestTarget(boxes[0]) {
		t.Errorf("hit targets = %v, want to start with the item of the list", got)
	}
	children := offstage.DebugDescribeChildren()
	if len(children) != 1 || children[0].Style() != foundation.DiagnosticsTreeStyleSparse {
		t.Errorf("children of a shown sliver = %v", children)
	}
	offstage.SetOffstage(true)
	if children = offstage.DebugDescribeChildren(); children[0].Style() != foundation.DiagnosticsTreeStyleOffstage {
		t.Errorf("the child of an offstage sliver is described as %v", children[0].Style())
	}
}

func TestRenderSliverConstrainedCrossAxis(t *testing.T) {
	list, boxes := listOf(2, 100)
	constrained := rendering.NewRenderSliverConstrainedCrossAxis(60, list)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{constrained}})
	h := newHarness(t)
	layout(t, h, root)
	if got := boxes[0].Size().Width; got != 60 {
		t.Errorf("item width = %v, want 60", got)
	}
	if got := constrained.Geometry().CrossAxisExtent; got != foundation.Some(60.0) {
		t.Errorf("geometry cross axis extent = %v, want 60", got)
	}
	constrained.SetMaxExtent(5000)
	pumpFrame(h, phaseLayout)
	if got := boxes[0].Size().Width; got != 800 {
		t.Errorf("item width = %v, want the viewport width (800)", got)
	}
	if constrained.MaxExtent() != 5000 {
		t.Errorf("MaxExtent = %v", constrained.MaxExtent())
	}
}

func TestRenderSliverOpacityBehavior(t *testing.T) {
	list, _ := listOf(2, 100)
	opacity := rendering.NewRenderSliverOpacity(list, rendering.RenderSliverOpacityOpts{Opacity: foundation.Some(0.5)})
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{opacity}})
	h := newHarness(t)
	layout(t, h, root, atPhase(layoutOpts{}, phaseComposite))
	if opacity.Opacity() != 0.5 || opacity.AlwaysIncludeSemantics() {
		t.Errorf("opacity %v, alwaysIncludeSemantics %v", opacity.Opacity(), opacity.AlwaysIncludeSemantics())
	}
	var visited int
	count := func(rendering.RenderObject) { visited++ }
	opacity.VisitChildrenForSemantics(count)
	if visited != 1 {
		t.Errorf("visited %d children for semantics, want 1", visited)
	}
	opacity.SetOpacity(0)
	pumpFrame(h, phaseComposite)
	if opacity.NeedsCompositing() {
		t.Error("a transparent sliver needs compositing")
	}
	visited = 0
	opacity.VisitChildrenForSemantics(count)
	if visited != 0 {
		t.Errorf("visited %d children for semantics of a transparent sliver, want 0", visited)
	}
	opacity.SetAlwaysIncludeSemantics(true)
	opacity.VisitChildrenForSemantics(count)
	if visited != 1 {
		t.Errorf("visited %d children for semantics with alwaysIncludeSemantics, want 1", visited)
	}
	skipUnlessDebug(t)
	requireTag(t, "OpacityOutOfRange", func() { opacity.SetOpacity(2) })
}
