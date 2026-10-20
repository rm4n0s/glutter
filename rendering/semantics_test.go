package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// Tests ported from flutter/test/rendering/semantics_and_children_test.dart,
// simple_semantics_test.dart and semantics_layout_sized_test.dart, and the
// render-object tests of flutter/test/semantics/semantics_test.dart.

func TestOpacityChildrenAndSemantics(t *testing.T) {
	box := rendering.NewRenderOpacity(newSemNode(nil))
	for _, tc := range []struct {
		opacity float64
		want    int
	}{{1, 1}, {0.5, 1}, {0.25, 1}, {0.125, 1}, {0, 0}, {0.125, 1}, {0, 0}} {
		box.SetOpacity(tc.opacity)
		if got := countSemanticsChildren(box); got != tc.want {
			t.Errorf("opacity %v: %d semantics children, want %d", tc.opacity, got, tc.want)
		}
	}
	box.SetAlwaysIncludeSemantics(true)
	if got := countSemanticsChildren(box); got != 1 {
		t.Errorf("alwaysIncludeSemantics: %d semantics children, want 1", got)
	}
}

// tickerProvider makes the tickers of the harness's scheduler (TestVSync).
type tickerProvider struct{ h *renderingtest.Harness }

func (p tickerProvider) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	return scheduler.NewTicker(p.h.Scheduler, onTick)
}

func TestAnimatedOpacityChildrenAndSemantics(t *testing.T) {
	h := newHarness(t)
	controller := animation.NewAnimationController(tickerProvider{h})
	defer controller.Dispose()
	p := rendering.NewRenderParagraph(painting.NewTextSpan(painting.TextSpanOpts{}), ui.TextDirectionLTR, rendering.RenderParagraphOpts{TextEngine: h.TextEngine})
	box := rendering.NewRenderAnimatedOpacity(controller, p)
	check := func(step string, want int) {
		t.Helper()
		if got := countSemanticsChildren(box); got != want {
			t.Errorf("%s: %d semantics children, want %d", step, got, want)
		}
	}
	check("controller defaults to 0", 0)
	controller.SetValue(0.2) // has no effect, box isn't subscribed yet
	check("0.2 before attach", 0)
	controller.SetValue(1.0)      // ditto
	check("1.0 before attach", 0) // alpha is still 0
	layout(t, h, box)             // this causes the box to attach, which makes it subscribe
	check("attached", 1)
	for _, tc := range []struct {
		value float64
		want  int
	}{{1, 1}, {0.5, 1}, {0.25, 1}, {0.125, 1}, {0, 0}, {0.125, 1}, {0, 0}} {
		controller.SetValue(tc.value)
		check("value", tc.want)
	}
	box.SetAlwaysIncludeSemantics(true)
	check("alwaysIncludeSemantics", 1)
}

func noIndex(children ...rendering.Box) *rendering.RenderIndexedStack {
	s := rendering.NewRenderIndexedStack(children)
	s.SetIndex(foundation.None[int]())
	return s
}

func annotated(child rendering.Box, text string, container bool) *rendering.RenderSemanticsAnnotations {
	return rendering.NewRenderSemanticsAnnotations(label(text), child,
		rendering.SemanticsAnnotationsOpts{Container: container, TextDirection: ui.TextDirectionLTR})
}

func labeledNode(child rendering.Box) *semNode {
	n := newSemNode(child)
	n.label = "Test Node"
	return n
}

func TestChildrenForSemanticsOfProxyBoxes(t *testing.T) {
	child := func() rendering.Box { return newSemNode(nil) }
	tests := []struct {
		name string
		box  rendering.RenderObject
		want int
	}{
		{"offstage", rendering.NewRenderOffstage(child()), 0},
		{"onstage", rendering.NewRenderOffstage(child(), rendering.RenderOffstageOpts{Offstage: foundation.Some(false)}), 1},
		{"exclude semantics", rendering.NewRenderExcludeSemantics(child()), 0},
		{"include semantics", rendering.NewRenderExcludeSemantics(child(), rendering.RenderExcludeSemanticsOpts{Excluding: foundation.Some(false)}), 1},
		{"annotations excluding", rendering.NewRenderSemanticsAnnotations(semantics.NewSemanticsProperties(semantics.SemanticsProperties{}), child(),
			rendering.SemanticsAnnotationsOpts{ExcludeSemantics: true}), 0},
		{"indexed stack", rendering.NewRenderIndexedStack([]rendering.Box{child(), child()}, rendering.RenderIndexedStackOpts{Index: foundation.Some(1)}), 1},
		{"indexed stack without index", noIndex(child(), child()), 0},
		{"ignore pointer", rendering.NewRenderIgnorePointer(child()), 1},
	}
	for _, tc := range tests {
		if got := countSemanticsChildren(tc.box); got != tc.want {
			t.Errorf("%s: %d semantics children, want %d", tc.name, got, tc.want)
		}
	}
}

// countingAnnotations is TestRender of simple_semantics_test.dart.
type countingAnnotations struct {
	rendering.RenderProxyBox
	rendering.SemanticsAnnotations
	calls int
}

func newCountingAnnotations(properties *semantics.SemanticsProperties) *countingAnnotations {
	r := &countingAnnotations{}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	rendering.InitSemanticsAnnotations(&r.SemanticsAnnotations, r, properties, rendering.SemanticsAnnotationsOpts{TextDirection: ui.TextDirectionLTR})
	return r
}

func (r *countingAnnotations) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.SemanticsAnnotations.DescribeSemanticsConfiguration(config)
	r.calls++
}

func label(s string) *semantics.SemanticsProperties {
	return semantics.NewSemanticsProperties(semantics.SemanticsProperties{Label: foundation.Some(s)})
}

func TestOnlySendSemanticsUpdateIfSemanticsChanged(t *testing.T) {
	h, owner := withSemantics(t)
	testRender := newCountingAnnotations(label("hello"))
	tree := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 20, Height: 20}), testRender)
	updates := 0
	owner.AddListener(func() { updates++ })

	layout(t, h, tree, atPhase(layoutOpts{}, phaseComposite))
	if updates != 1 {
		t.Errorf("initial render: %d semantics updates, want 1", updates)
	}
	if testRender.calls == 0 {
		t.Error("the object was not asked for its semantics")
	}

	testRender.calls, updates = 0, 0
	// Request a semantics update although nothing changed.
	testRender.MarkNeedsSemanticsUpdate()
	h.Pump()
	if updates != 0 {
		t.Errorf("unchanged semantics: %d updates, want 0", updates)
	}
	if testRender.calls == 0 {
		t.Error("the object was not asked for its semantics again")
	}

	testRender.calls, updates = 0, 0
	testRender.SetProperties(label("bye"))
	h.Pump()
	if updates != 1 {
		t.Errorf("changed semantics: %d updates, want 1", updates)
	}
	if testRender.calls == 0 {
		t.Error("the object was not asked for its semantics after the change")
	}
}

func TestSemanticsSizeUpdatesWhenLayoutSizeChanges(t *testing.T) {
	h, _ := withSemantics(t)
	// The parent does not use the size of the child, so the child is a relayout
	// boundary; it is also a semantic boundary.
	child := newSemNode(nil)
	child.boundary = true
	parent := newSemNode(child)
	layout(t, h, parent, atPhase(layoutOpts{}, phaseComposite))
	if child.Size() != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("size = %v", child.Size())
	}
	if got, want := debugRect(t, child), ui.NewRectFromLTWH(0, 0, 100, 100); got != want {
		t.Errorf("rect = %v, want %v", got, want)
	}
	child.targetSize = ui.Size{Width: 200, Height: 200}
	child.MarkNeedsLayout()
	h.Pump()
	if child.Size() != (ui.Size{Width: 200, Height: 200}) {
		t.Fatalf("size = %v", child.Size())
	}
	if got, want := debugRect(t, child), ui.NewRectFromLTWH(0, 0, 200, 200); got != want {
		t.Errorf("rect = %v, want %v", got, want)
	}
}

func TestSemanticsBlockSemanticsCornerCase(t *testing.T) {
	h, _ := withSemantics(t)
	child := labeledNode(nil)
	middle1 := labeledNode(child)
	middle1.boundary, middle1.explicit = true, true
	middle2 := labeledNode(middle1)
	parent := newSemMulti(middle2)
	parent.explicit = true
	layout(t, h, parent, atPhase(layoutOpts{}, phaseComposite))
	if got, want := debugRect(t, child), ui.NewRectFromLTWH(0, 0, 100, 100); got != want {
		t.Errorf("rect = %v, want %v", got, want)
	}
	middle1.MarkNeedsLayout()
	middle2.MarkNeedsLayout()
	child.MarkNeedsLayout()

	child2 := newSemNode(nil)
	child2.boundary, child2.blocking, child2.label = true, true, "Test Node2"
	parent.Add(child2)
	h.Pump()

	middle1.MarkNeedsLayout()
	child.MarkNeedsLayout()
	h.Pump() // does not crash
}

func TestSemanticsSizeUpdatesToZeroSize(t *testing.T) {
	h, _ := withSemantics(t)
	child := newSemNode(nil)
	child.boundary = true
	parent := newSemNode(child)
	parent.boundary = true
	layout(t, h, parent, atPhase(layoutOpts{}, phaseComposite))
	if got, want := debugRect(t, child), ui.NewRectFromLTWH(0, 0, 100, 100); got != want {
		t.Errorf("rect = %v, want %v", got, want)
	}
	child.targetSize = ui.Size{}
	child.MarkNeedsLayout()
	h.Pump()
}

func TestSemanticsUpdatesWhenChildChangesInLastChildParent(t *testing.T) {
	h, _ := withSemantics(t)
	childOfChild := newSemNode(nil)
	originalChild := newSemNode(childOfChild)
	originalChild.boundary, originalChild.explicit = true, true
	parent := newSemMulti(originalChild)
	parent.lastChildOnly, parent.explicit = true, true
	layout(t, h, parent, atPhase(layoutOpts{}, phaseComposite))
	parentSemantics := parent.DebugSemantics()
	if got := parentSemantics.ChildrenCount(); got != 1 {
		t.Fatalf("initial children = %d, want 1", got)
	}

	newChild := newSemNode(nil)
	newChild.boundary = true
	parent.Add(newChild)
	h.Pump()

	// This adds originalChild to the dirty list, but it must not be updated.
	childOfChild.MarkNeedsLayout()
	h.Pump()

	parent.Remove(newChild)
	h.Pump()
	if got := parentSemantics.ChildrenCount(); got != 1 {
		t.Errorf("children after removal = %d, want 1", got)
	}
}

func TestSkipUpdateForInvisibleChildDroppedFromTree(t *testing.T) {
	h, _ := withSemantics(t)
	child := newSemNode(nil)
	child.boundary = true
	child.targetSize = ui.Size{}
	parent := newSemNode(child)
	parent.usesSize, parent.boundary, parent.explicit = true, true, true
	grandParent := newSemMulti(parent)
	grandParent.lastChildOnly, grandParent.explicit = true, true
	layout(t, h, grandParent, atPhase(layoutOpts{}, phaseComposite))
	if got := grandParent.DebugSemantics().ChildrenCount(); got != 1 {
		t.Fatalf("children = %d, want 1", got)
	}
	parent.MarkNeedsLayout()
	child.MarkNeedsLayout()
	// The new last child makes the original parent and child drop out of the
	// semantics tree without their parent data being cleared.
	newChild := newSemNode(nil)
	newChild.boundary = true
	grandParent.Add(newChild)
	h.Pump()
}

func TestMarkNeedsSemanticsUpdateAsksAllObjectsBetweenBoundaries(t *testing.T) {
	h, _ := withSemantics(t)
	deepest := newSemNode(nil)
	deepest.scrollUp, deepest.boundary = true, true
	scrollRight := newSemNode(deepest)
	scrollRight.scrollRight = true
	middle := newSemNode(scrollRight)
	middle.scrollLeft = true
	longPress := newSemNode(middle)
	longPress.longPress = true
	root := newSemNode(longPress)
	root.tap, root.boundary = true, true
	layout(t, h, root, atPhase(layoutOpts{}, phaseComposite))

	actions := func() ui.SemanticsAction { return root.DebugSemantics().GetSemanticsData().Actions }
	want := ui.SemanticsActionTap | ui.SemanticsActionLongPress | ui.SemanticsActionScrollLeft | ui.SemanticsActionScrollRight
	if got := actions(); got != want {
		t.Errorf("actions = %v, want %v", got, want)
	}
	middle.scrollLeft, middle.scrollDown = false, true
	middle.MarkNeedsSemanticsUpdate()
	h.Pump()
	want = ui.SemanticsActionTap | ui.SemanticsActionLongPress | ui.SemanticsActionScrollDown | ui.SemanticsActionScrollRight
	if got := actions(); got != want {
		t.Errorf("actions after the change = %v, want %v", got, want)
	}
}

func TestMergeSemanticsMergesDescendants(t *testing.T) {
	h, _ := withSemantics(t)
	a := annotated(newSemNode(nil), "a", false)
	b := annotated(newSemNode(nil), "b", false)
	column := newSemMulti(a, b)
	column.explicit = false
	merge := rendering.NewRenderMergeSemantics(column)
	layout(t, h, merge, atPhase(layoutOpts{}, phaseComposite))
	node := merge.DebugSemantics()
	if node == nil {
		t.Fatal("the merging object has no node")
	}
	if got := node.GetSemanticsData().Label(); got == "" {
		t.Error("the merged label is empty")
	}
	if a.DebugSemantics() != nil && a.DebugSemantics() != node && !a.DebugSemantics().IsMergedIntoParent() {
		t.Error("the child node is not merged into the parent")
	}
}

func TestExcludeAndBlockSemantics(t *testing.T) {
	h, _ := withSemantics(t)
	under := annotated(newSemNode(nil), "under", true)
	over := annotated(newSemNode(nil), "over", true)
	stack := newSemMulti(under, rendering.NewRenderBlockSemantics(over))
	stack.explicit = true
	layout(t, h, stack, atPhase(layoutOpts{}, phaseComposite))
	if under.DebugSemantics() != nil && under.DebugSemantics().Parent() != nil {
		t.Error("the blocked object still has a node in the tree")
	}
	if over.DebugSemantics() == nil {
		t.Error("the blocking object has no node")
	}
	if got := stack.DebugSemantics().ChildrenCount(); got != 1 {
		t.Errorf("children = %d, want 1", got)
	}

	exclude := rendering.NewRenderExcludeSemantics(annotated(newSemNode(nil), "hidden", true))
	root := newSemMulti(exclude)
	root.explicit = true
	layout(t, h, root, atPhase(layoutOpts{}, phaseComposite))
	if got := root.DebugSemantics().ChildrenCount(); got != 0 {
		t.Errorf("children of the excluded subtree = %d, want 0", got)
	}
}

func TestIndexedSemanticsSetsIndexInParent(t *testing.T) {
	h, _ := withSemantics(t)
	inner := annotated(newSemNode(nil), "item", true)
	indexed := rendering.NewRenderIndexedSemantics(3, inner)
	layout(t, h, indexed, atPhase(layoutOpts{}, phaseComposite))
	node := indexed.DebugSemantics()
	if node == nil {
		t.Fatal("no node")
	}
	if got := node.IndexInParent; got != foundation.Some(3) {
		t.Errorf("indexInParent = %v", got)
	}
}

func TestSemanticsAnnotationsPropertiesReachTheNode(t *testing.T) {
	h, _ := withSemantics(t)
	taps := 0
	props := semantics.NewSemanticsProperties(semantics.SemanticsProperties{
		Label: foundation.Some("Press"), Button: foundation.Some(true), Enabled: foundation.Some(true),
		OnTap: func() { taps++ },
	})
	box := rendering.NewRenderSemanticsAnnotations(props, newSemNode(nil), rendering.SemanticsAnnotationsOpts{Container: true, TextDirection: ui.TextDirectionLTR})
	layout(t, h, box, atPhase(layoutOpts{}, phaseComposite))
	node := box.DebugSemantics()
	data := node.GetSemanticsData()
	if data.Label() != "Press" || !data.FlagsCollection.IsButton || !data.HasAction(ui.SemanticsActionTap) {
		t.Errorf("data = %s", data)
	}
	h.Owner.SemanticsOwner().PerformAction(node.ID(), ui.SemanticsActionTap, nil)
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
	// The handler is looked up when it runs: a new properties object with another
	// handler does not need a semantics update.
	other := 0
	box.SetProperties(semantics.NewSemanticsProperties(semantics.SemanticsProperties{
		Label: foundation.Some("Press"), Button: foundation.Some(true), Enabled: foundation.Some(true),
		OnTap: func() { other++ },
	}))
	h.Pump()
	h.Owner.SemanticsOwner().PerformAction(node.ID(), ui.SemanticsActionTap, nil)
	if other != 1 || taps != 1 {
		t.Errorf("after replacing the handler: taps %d, other %d", taps, other)
	}
}

func TestSemanticsGestureHandlerPerformsScrolls(t *testing.T) {
	h, _ := withSemantics(t)
	var deltas []ui.Offset
	var taps int
	handler := rendering.NewRenderSemanticsGestureHandler(newSemNode(nil), rendering.RenderSemanticsGestureHandlerOpts{
		OnTap:                  func() { taps++ },
		OnHorizontalDragUpdate: func(d gestures.DragUpdateDetails) { deltas = append(deltas, d.Delta) },
		OnVerticalDragUpdate:   func(d gestures.DragUpdateDetails) { deltas = append(deltas, d.Delta) },
	})
	layout(t, h, handler, atPhase(layoutOpts{}, phaseComposite))
	node := handler.DebugSemantics()
	if node == nil {
		t.Fatal("no node")
	}
	owner := h.Owner.SemanticsOwner()
	for _, a := range []ui.SemanticsAction{ui.SemanticsActionTap, ui.SemanticsActionScrollLeft, ui.SemanticsActionScrollRight,
		ui.SemanticsActionScrollUp, ui.SemanticsActionScrollDown} {
		owner.PerformAction(node.ID(), a, nil)
	}
	size := handler.Size()
	want := []ui.Offset{{Dx: -size.Width * 0.8}, {Dx: size.Width * 0.8}, {Dy: -size.Height * 0.8}, {Dy: size.Height * 0.8}}
	if taps != 1 || len(deltas) != 4 {
		t.Fatalf("taps %d, deltas %v", taps, deltas)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta %d = %v, want %v", i, deltas[i], want[i])
		}
	}

	// validActions filters the actions.
	handler.SetValidActions(map[ui.SemanticsAction]struct{}{ui.SemanticsActionScrollLeft: {}})
	h.Pump()
	data := handler.DebugSemantics().GetSemanticsData()
	if data.HasAction(ui.SemanticsActionTap) || !data.HasAction(ui.SemanticsActionScrollLeft) || data.HasAction(ui.SemanticsActionScrollRight) {
		t.Errorf("actions = %v", data.Actions)
	}
}

func TestIgnoreAndAbsorbPointerBlockUserActions(t *testing.T) {
	for name, wrap := range map[string]func(rendering.Box) rendering.Box{
		"ignore": func(c rendering.Box) rendering.Box { return rendering.NewRenderIgnorePointer(c) },
		"absorb": func(c rendering.Box) rendering.Box { return rendering.NewRenderAbsorbPointer(c) },
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := withSemantics(t)
			inner := newSemNode(nil)
			inner.boundary, inner.tap, inner.label = true, true, "inner"
			layout(t, h, wrap(inner), atPhase(layoutOpts{}, phaseComposite))
			node := inner.DebugSemantics()
			if node == nil {
				t.Fatal("no node")
			}
			if !node.AreUserActionsBlocked() {
				t.Error("the user actions are not blocked")
			}
			if node.GetSemanticsData().HasAction(ui.SemanticsActionTap) {
				t.Error("the blocked node still offers the tap action")
			}
		})
	}
}

func TestSemanticsEnabledMessageInRenderViewDump(t *testing.T) {
	skipUnlessDebug(t)
	h, _ := withSemantics(t)
	h.Dispatcher.SetSemanticsEnabled(true)
	layout(t, h, newSemNode(nil), atPhase(layoutOpts{}, phaseComposite))
	if got := deepInfo(h.RenderView); !containsLine(got, "semantics enabled") {
		t.Errorf("the dump of the view lacks 'semantics enabled':\n%s", got)
	}
}

func containsLine(s, line string) bool {
	for _, l := range splitLines(s) {
		if l == line || len(l) > len(line) && l[len(l)-len(line):] == line {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// painterWithSemantics is a CustomPainter with a semantics builder.
type painterWithSemantics struct {
	rendering.CustomPainterBase
	build rendering.SemanticsBuilderCallback
}

func (p *painterWithSemantics) Paint(*ui.Canvas, ui.Size)                  {}
func (p *painterWithSemantics) ShouldRepaint(rendering.CustomPainter) bool { return false }
func (p *painterWithSemantics) HitTest(ui.Offset) foundation.Opt[bool] {
	return foundation.None[bool]()
}
func (p *painterWithSemantics) SemanticsBuilder() rendering.SemanticsBuilderCallback { return p.build }

func TestCustomPaintSemantics(t *testing.T) {
	h, _ := withSemantics(t)
	keyA, keyB := foundation.NewValueKey("a"), foundation.NewValueKey("b")
	items := func(keys ...foundation.Key) rendering.SemanticsBuilderCallback {
		return func(size ui.Size) []rendering.CustomPainterSemantics {
			var out []rendering.CustomPainterSemantics
			for i, k := range keys {
				out = append(out, rendering.CustomPainterSemantics{
					Key: k, Rect: ui.NewRectFromLTWH(float64(i)*10, 0, 10, 10),
					Properties: semantics.NewSemanticsProperties(semantics.SemanticsProperties{
						Label: foundation.Some("item"), TextDirection: ui.TextDirectionLTR,
					}),
				})
			}
			return out
		}
	}
	painter := &painterWithSemantics{CustomPainterBase: rendering.NewCustomPainterBase(nil), build: items(keyA, keyB)}
	box := rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: painter, PreferredSize: ui.Size{Width: 50, Height: 50}})
	layout(t, h, box, atPhase(layoutOpts{}, phaseComposite))
	node := box.DebugSemantics()
	if node == nil || node.ChildrenCount() != 2 {
		t.Fatalf("node = %v", node)
	}
	var ids []int
	node.VisitChildren(func(c *semantics.SemanticsNode) bool { ids = append(ids, c.ID()); return true })

	// The same keys in the opposite order keep the nodes.
	box.SetPainter(&painterWithSemantics{CustomPainterBase: rendering.NewCustomPainterBase(nil), build: items(keyB, keyA)})
	h.Pump()
	var after []int
	box.DebugSemantics().VisitChildren(func(c *semantics.SemanticsNode) bool { after = append(after, c.ID()); return true })
	if len(after) != 2 || (after[0] != ids[0] || after[1] != ids[1]) && (after[0] != ids[1] || after[1] != ids[0]) {
		t.Errorf("node IDs %v changed to %v", ids, after)
	}
}

func TestTableSemantics(t *testing.T) {
	h, _ := withSemantics(t)
	cell := func(text string) rendering.Box {
		return annotated(newSemNode(nil), text, true)
	}
	table := rendering.NewRenderTable(ui.TextDirectionLTR)
	table.SetFlatChildren(2, []rendering.Box{cell("a"), cell("b"), cell("c"), cell("d")})
	layout(t, h, table, atPhase(layoutOpts{}, phaseComposite))
	node := table.DebugSemantics()
	if node == nil {
		t.Fatal("the table has no node")
	}
	if node.Role() != ui.SemanticsRoleTable {
		t.Errorf("role = %v, want table", node.Role())
	}
	rows := 0
	node.VisitChildren(func(row *semantics.SemanticsNode) bool {
		rows++
		if row.Role() != ui.SemanticsRoleRow || row.ChildrenCount() != 2 {
			t.Errorf("row %d: role %v with %d children", rows, row.Role(), row.ChildrenCount())
		}
		row.VisitChildren(func(c *semantics.SemanticsNode) bool {
			if c.Role() != ui.SemanticsRoleCell {
				t.Errorf("cell role = %v", c.Role())
			}
			return true
		})
		return true
	})
	if rows != 2 {
		t.Errorf("rows = %d, want 2", rows)
	}
}

func TestParagraphSemanticsLabel(t *testing.T) {
	h, _ := withSemantics(t)
	span := painting.NewTextSpan(painting.TextSpanOpts{
		Text: foundation.Some("Hello "),
		Children: []painting.InlineSpan{
			painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("w0rld"), SemanticsLabel: foundation.Some("world")}),
		},
	})
	p := rendering.NewRenderParagraph(span, ui.TextDirectionLTR, rendering.RenderParagraphOpts{TextEngine: h.TextEngine})
	layout(t, h, p, atPhase(layoutOpts{}, phaseComposite))
	node := p.DebugSemantics()
	if node == nil {
		t.Fatal("the paragraph has no node")
	}
	if got := node.Label(); got != "Hello world" {
		t.Errorf("label = %q, want %q", got, "Hello world")
	}
	if node.TextDirection() != ui.TextDirectionLTR {
		t.Errorf("direction = %v", node.TextDirection())
	}
}

func TestParagraphSemanticsOfRecognizerSpans(t *testing.T) {
	h, _ := withSemantics(t)
	tapped := false
	rec := gestures.NewTapGestureRecognizer(h.Gestures)
	rec.OnTap = func() { tapped = true }
	span := painting.NewTextSpan(painting.TextSpanOpts{
		Text: foundation.Some("Read "),
		Children: []painting.InlineSpan{
			painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("more"), Recognizer: rec}),
		},
	})
	p := rendering.NewRenderParagraph(span, ui.TextDirectionLTR, rendering.RenderParagraphOpts{TextEngine: h.TextEngine})
	layout(t, h, p, atPhase(layoutOpts{}, phaseComposite))
	node := p.DebugSemantics()
	if node == nil || node.ChildrenCount() != 2 {
		t.Fatalf("node = %v", node)
	}
	var link *semantics.SemanticsNode
	node.VisitChildren(func(c *semantics.SemanticsNode) bool {
		if c.GetSemanticsData().FlagsCollection.IsLink {
			link = c
		}
		return true
	})
	if link == nil || link.Label() != "more" {
		t.Fatalf("no link node for the recognizer span among the children")
	}
	h.Owner.SemanticsOwner().PerformAction(link.ID(), ui.SemanticsActionTap, nil)
	if !tapped {
		t.Error("the tap action did not call the recognizer's handler")
	}
}

func TestGetRectOfSemanticsNodeInViewCoordinates(t *testing.T) {
	h, _ := withSemantics(t)
	box := annotated(newSemNode(nil), "x", true)
	layout(t, h, box, atPhase(layoutOpts{}, phaseComposite))
	node := box.DebugSemantics()
	r, ok := h.Binding.GetRectOfSemanticsNodeInViewCoordinates(h.View.ViewID(), node.ID())
	if !ok || r != ui.NewRectFromLTWH(0, 0, 800, 600) {
		t.Errorf("rect = %v, %v", r, ok)
	}
}

func TestDebugDumpSemanticsTree(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	layout(t, h, newSemNode(nil), atPhase(layoutOpts{}, phaseComposite))
	got := h.Binding.DebugDumpSemanticsTree(semantics.DebugSemanticsDumpOrderTraversal)
	if !containsLine(got, "Semantics not generated for RenderView#"+got[len("Semantics not generated for RenderView#"):len("Semantics not generated for RenderView#")+5]+".") {
		t.Errorf("dump = %q", got)
	}
	handle := h.Binding.SemanticsBinding().EnsureSemantics()
	defer handle.Dispose()
	h.Pump()
	if got := h.Binding.DebugDumpSemanticsTree(semantics.DebugSemanticsDumpOrderTraversal); !containsLine(got, "SemanticsNode#0") {
		t.Errorf("dump with semantics = %q", got)
	}
}

func TestSemanticsErrorRoutesInRendering(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("SemanticsUpdateCallbackMissing", func(t *testing.T) {
		h := newHarness(t)
		owner := rendering.NewPipelineOwner()
		h.Binding.RootPipelineOwner().AdoptChild(owner)
		requireTag(t, "SemanticsUpdateCallbackMissing", func() {
			handle := h.Binding.SemanticsBinding().EnsureSemantics()
			handle.Dispose()
		})
	})
	t.Run("SemanticsOwnerMissing", func(t *testing.T) {
		h := newHarness(t)
		requireTag(t, "SemanticsOwnerMissing", h.RenderView.ScheduleInitialSemantics)
	})
	t.Run("SemanticsInitialNotRoot", func(t *testing.T) {
		h, _ := withSemantics(t)
		child := newSemNode(nil)
		h.SetRoot(child)
		requireTag(t, "SemanticsInitialNotRoot", child.ScheduleInitialSemantics)
	})
	t.Run("SemanticsExplicitNodesWithDelegate", func(t *testing.T) {
		h, _ := withSemantics(t)
		box := &badDelegateBox{}
		rendering.InitRenderBox(&box.RenderBox, box)
		requireTag(t, "SemanticsExplicitNodesWithDelegate", func() { h.SetRoot(box); h.Pump() })
	})
	t.Run("SemanticsScopesRouteNeedsExplicitChildNodes", func(t *testing.T) {
		h, _ := withSemantics(t)
		props := semantics.NewSemanticsProperties(semantics.SemanticsProperties{ScopesRoute: foundation.Some(true)})
		requireTag(t, "SemanticsScopesRouteNeedsExplicitChildNodes", func() {
			h.SetRoot(rendering.NewRenderSemanticsAnnotations(props, newSemNode(nil)))
			h.Pump()
		})
	})
	t.Run("SemanticsToggledAndChecked", func(t *testing.T) {
		h, _ := withSemantics(t)
		props := semantics.NewSemanticsProperties(semantics.SemanticsProperties{Toggled: foundation.Some(true), Checked: foundation.Some(true)})
		requireTag(t, "SemanticsToggledAndChecked", func() {
			h.SetRoot(rendering.NewRenderSemanticsAnnotations(props, newSemNode(nil)))
			h.Pump()
		})
	})
	t.Run("SemanticsLookupFailed", func(t *testing.T) {
		h := newHarness(t)
		requireTag(t, "SemanticsLookupFailed", func() { h.Binding.GetRectOfSemanticsNodeInViewCoordinates(h.View.ViewID(), 3) })
		requireTag(t, "SemanticsLookupFailed", func() { h.Binding.GetRectOfSemanticsNodeInViewCoordinates(ui.ViewID(99), 3) })
		hh, _ := withSemantics(t)
		requireTag(t, "SemanticsLookupFailed", func() { hh.Binding.GetRectOfSemanticsNodeInViewCoordinates(hh.View.ViewID(), 99) })
	})
	t.Run("CustomPaintSemanticsNoChild", func(t *testing.T) {
		h, _ := withSemantics(t)
		painter := &painterWithSemantics{CustomPainterBase: rendering.NewCustomPainterBase(nil)}
		box := rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: painter})
		h.SetRoot(box)
		h.Pump()
		requireTag(t, "CustomPaintSemanticsNoChild", func() {
			box.AssembleSemanticsNode(semantics.NewSemanticsNode(), semantics.NewSemanticsConfiguration(), []*semantics.SemanticsNode{semantics.NewSemanticsNode()})
		})
	})
	t.Run("CustomPainterSemanticsKeyDuplicated", func(t *testing.T) {
		h, _ := withSemantics(t)
		key := foundation.NewValueKey("same")
		painter := &painterWithSemantics{CustomPainterBase: rendering.NewCustomPainterBase(nil)}
		painter.build = func(ui.Size) []rendering.CustomPainterSemantics {
			item := rendering.CustomPainterSemantics{Key: key, Rect: ui.NewRectFromLTWH(0, 0, 5, 5),
				Properties: semantics.NewSemanticsProperties(semantics.SemanticsProperties{})}
			return []rendering.CustomPainterSemantics{item, item}
		}
		box := rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: painter, PreferredSize: ui.Size{Width: 50, Height: 50}})
		requireTag(t, "CustomPainterSemanticsKeyDuplicated", func() { h.SetRoot(box); h.Pump() })
	})
	t.Run("ParagraphRecognizerUnsupported", func(t *testing.T) {
		h, _ := withSemantics(t)
		rec := gestures.NewScaleGestureRecognizer(h.Gestures)
		span := painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("pinch"), Recognizer: rec})
		p := rendering.NewRenderParagraph(span, ui.TextDirectionLTR, rendering.RenderParagraphOpts{TextEngine: h.TextEngine})
		requireTag(t, "ParagraphRecognizerUnsupported", func() { h.SetRoot(p); h.Pump() })
	})
	t.Run("SemanticsDuringSemantics", func(t *testing.T) {
		h, _ := withSemantics(t)
		box := &markDuringSemantics{}
		box.hook = h.RenderView.ScheduleInitialSemantics
		rendering.InitRenderBox(&box.RenderBox, box)
		h.SetRoot(box)
		requireTag(t, "SemanticsDuringSemantics", h.Pump)
	})
	t.Run("SemanticsMarkedWhileFlushing", func(t *testing.T) {
		h, _ := withSemantics(t)
		box := &markDuringSemantics{}
		rendering.InitRenderBox(&box.RenderBox, box)
		h.SetRoot(box)
		requireTag(t, "SemanticsMarkedWhileFlushing", h.Pump)
	})
}

// badDelegateBox has explicit child nodes and a child configurations delegate.
type badDelegateBox struct{ rendering.RenderBox }

func (b *badDelegateBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size { return c.Biggest() }
func (b *badDelegateBox) PerformLayout()                                      { b.SetSize(b.BoxConstraints().Biggest()) }
func (b *badDelegateBox) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	config.ExplicitChildNodes = true
	config.SetChildConfigurationsDelegate(func([]*semantics.SemanticsConfiguration) *semantics.ChildSemanticsConfigurationsResult {
		return semantics.NewChildSemanticsConfigurationsResultBuilder().Build()
	})
}

// markDuringSemantics marks itself as needing a semantics update while the
// semantics are flushed.
type markDuringSemantics struct {
	rendering.RenderBox
	hook func()
}

func (b *markDuringSemantics) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Biggest()
}
func (b *markDuringSemantics) PerformLayout() { b.SetSize(b.BoxConstraints().Biggest()) }
func (b *markDuringSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	config.SetIsSemanticBoundary(true)
}

func (b *markDuringSemantics) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if b.hook != nil {
		b.hook()
	} else {
		b.MarkNeedsSemanticsUpdate()
	}
	node.UpdateWith(config, children)
}
