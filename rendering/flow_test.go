package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// testFlowDelegate paints the children translated by 20 pixels each, like
// the delegate in widgets/flow_test.dart.
type testFlowDelegate struct {
	rendering.FlowDelegateBase
	step       float64
	opacities  map[int]float64
	paintTwice bool
	relayout   bool
	repaint    bool
	paintCalls int
}

func newTestFlowDelegate(repaint foundation.Listenable) *testFlowDelegate {
	return &testFlowDelegate{FlowDelegateBase: rendering.NewFlowDelegateBase(repaint), step: 20}
}

func (d *testFlowDelegate) PaintChildren(ctx rendering.FlowPaintingContext) {
	d.paintCalls++
	for i := 0; i < ctx.ChildCount(); i++ {
		opts := rendering.FlowPaintChildOpts{Transform: ui.NewMatrix4TranslationValues(float32(d.step*float64(i)), 0, 0)}
		if o, ok := d.opacities[i]; ok {
			opts.Opacity = foundation.Some(o)
		}
		ctx.PaintChildAt(i, opts)
		if d.paintTwice {
			ctx.PaintChildAt(i)
		}
	}
}
func (d *testFlowDelegate) GetConstraintsForChild(_ int, c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.Loosen()
}
func (d *testFlowDelegate) ShouldRepaint(rendering.FlowDelegate) bool  { return d.repaint }
func (d *testFlowDelegate) ShouldRelayout(rendering.FlowDelegate) bool { return d.relayout }

func TestFlowPaintsAndHitTestsWithTransforms(t *testing.T) {
	boxes := []rendering.Box{newSizedBox(10, 10), newSizedBox(10, 10), newSizedBox(10, 10)}
	delegate := newTestFlowDelegate(nil)
	flow := rendering.NewRenderFlow(delegate, boxes)
	h := newHarness(t)
	layout(t, h, flow, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 50})), phaseComposite))
	if flow.Size() != (ui.Size{Width: 100, Height: 50}) {
		t.Errorf("flow size = %v", flow.Size())
	}
	if !flow.IsRepaintBoundary() {
		t.Error("a flow is not a repaint boundary")
	}
	for i, b := range boxes {
		if want := off(20*float64(i), 0); global(b) != want && b.LocalToGlobal(ui.OffsetZero, flow) != want {
			t.Errorf("box %d at %v relative to the flow, want %v", i, b.LocalToGlobal(ui.OffsetZero, flow), want)
		}
	}
	tests := []struct {
		at   ui.Offset
		want int
	}{{off(5, 5), 0}, {off(25, 5), 1}, {off(45, 5), 2}, {off(15, 5), -1}}
	for _, tc := range tests {
		res := rendering.NewBoxHitTestResult()
		flow.HitTest(res, tc.at)
		hit := -1
		for _, e := range res.Path() {
			for i, b := range boxes {
				if e.Target() == any(b) {
					hit = i
				}
			}
		}
		if hit != tc.want {
			t.Errorf("hit at %v = box %d, want %d", tc.at, hit, tc.want)
		}
	}
}

func TestFlowPaintChildTwiceAndOpacity(t *testing.T) {
	skipUnlessDebug(t)
	delegate := newTestFlowDelegate(nil)
	delegate.paintTwice = true
	flow := rendering.NewRenderFlow(delegate, []rendering.Box{newSizedBox(10, 10)})
	h := newHarness(t)
	layout(t, h, flow, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 50})), phaseCompositingBits))
	requireTag(t, "FlowChildPaintedTwice", func() { pumpFrame(h, phasePaint) })
}

func TestFlowOpacityZeroSkipsPaint(t *testing.T) {
	a, b := newTBox(10, 10), newTBox(10, 10)
	delegate := newTestFlowDelegate(nil)
	delegate.opacities = map[int]float64{0: 0, 1: 0.5}
	flow := rendering.NewRenderFlow(delegate, []rendering.Box{a, b})
	layout(t, newHarness(t), flow, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 50})), phaseComposite))
	if a.paints != 0 || b.paints != 1 {
		t.Errorf("paints: %d and %d, want 0 and 1", a.paints, b.paints)
	}
}

func TestFlowDelegateChanges(t *testing.T) {
	notifier := foundation.NewChangeNotifier()
	first := newTestFlowDelegate(notifier)
	box := newTBox(10, 10)
	flow := rendering.NewRenderFlow(first, []rendering.Box{box})
	h := newHarness(t)
	layout(t, h, flow, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 50})), phaseComposite))
	before := first.paintCalls
	notifier.NotifyListeners()
	if !flow.AsRenderObject().DebugNeedsPaint() && foundation.DebugMode {
		t.Error("the repaint listenable did not mark the flow as needing paint")
	}
	pumpFrame(h, phaseComposite)
	if first.paintCalls != before+1 {
		t.Errorf("paint calls = %d, want %d", first.paintCalls, before+1)
	}
	t.Run("same delegate does nothing", func(t *testing.T) {
		flow.SetDelegate(first)
		if foundation.DebugMode && (flow.AsRenderObject().DebugNeedsLayout() || flow.AsRenderObject().DebugNeedsPaint()) {
			t.Error("setting the same delegate marked the flow")
		}
	})
	t.Run("repaint only", func(t *testing.T) {
		next := newTestFlowDelegate(nil)
		next.repaint = true
		flow.SetDelegate(next)
		if foundation.DebugMode && (flow.AsRenderObject().DebugNeedsLayout() || !flow.AsRenderObject().DebugNeedsPaint()) {
			t.Error("the flow needs layout, or does not need paint")
		}
		pumpFrame(h, phaseComposite)
	})
	t.Run("relayout", func(t *testing.T) {
		next := newTestFlowDelegate(nil)
		next.relayout = true
		flow.SetDelegate(next)
		if foundation.DebugMode && !flow.AsRenderObject().DebugNeedsLayout() {
			t.Error("the flow does not need layout")
		}
	})
}

func TestFlowIntrinsicsAndClip(t *testing.T) {
	flow := rendering.NewRenderFlow(newTestFlowDelegate(nil), nil)
	if flow.GetMinIntrinsicWidth(40) != 0 || flow.GetMaxIntrinsicHeight(40) != 0 {
		t.Error("intrinsics of an unbounded flow are not 0")
	}
	if flow.ClipBehavior() != ui.ClipHardEdge {
		t.Errorf("default clip = %v, want hardEdge", flow.ClipBehavior())
	}
	for _, tc := range clipCases(ui.ClipHardEdge) {
		flow := rendering.NewRenderFlow(newTestFlowDelegate(nil), []rendering.Box{newTBox(200, 200)}, rendering.RenderFlowOpts{ClipBehavior: tc.clip})
		h := newHarness(t)
		layout(t, h, flow, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})), phaseComposite))
		root := rendering.NewContainerLayer()
		ctx := rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 800, 600))
		flow.Paint(ctx, ui.OffsetZero)
		ctx.StopRecordingIfNeeded()
		if tc.want == ui.ClipNone && countOps[ui.OpClipRect](recordOps(flow)) != 0 {
			t.Errorf("clip %v: a clip was applied", tc.clip)
		}
	}
	if got := flow.GetMinIntrinsicHeight(inf); got != 0 {
		t.Errorf("min intrinsic height = %v", got)
	}
}
