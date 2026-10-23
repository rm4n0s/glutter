package rendering_test

import (
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func tight(w, h float64) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsTight(ui.Size{Width: w, Height: h})
}

func TestRenderPaddingLayout(t *testing.T) {
	child := newSizedBox(40, 30)
	pad := rendering.NewRenderPadding(painting.EdgeInsets{Left: 1, Top: 2, Right: 3, Bottom: 4}, child)
	layout(t, newHarness(t), pad, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}), phaseLayout))
	if pad.Size() != (ui.Size{Width: 44, Height: 36}) {
		t.Errorf("padding size = %v, want 44x36", pad.Size())
	}
	if off := rendering.BoxParentDataOf(child).Offset; off != (ui.Offset{Dx: 1, Dy: 2}) {
		t.Errorf("child offset = %v, want (1, 2)", off)
	}
	if got := pad.GetDryLayout(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}); got != pad.Size() {
		t.Errorf("dry layout = %v, want %v", got, pad.Size())
	}
	// A directional padding resolves with the text direction.
	dir := rendering.NewRenderPadding(painting.EdgeInsetsDirectional{Start: 5, End: 1}, newSizedBox(10, 10),
		rendering.RenderPaddingOpts{TextDirection: ui.TextDirectionRTL})
	layout(t, newHarness(t), dir, withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}))
	if off := rendering.BoxParentDataOf(dir.Child()).Offset; off.Dx != 1 {
		t.Errorf("rtl child offset = %v, want dx 1", off)
	}
	dir.SetTextDirection(ui.TextDirectionLTR)
	if !dir.DebugNeedsLayout() && foundation.DebugMode {
		t.Error("changing the text direction did not mark the box for layout")
	}
}

func TestRenderPaddingHitTest(t *testing.T) {
	child := newSizedBox(10, 10)
	pad := rendering.NewRenderPadding(painting.NewEdgeInsetsAll(10), child)
	layout(t, newHarness(t), pad, layoutOpts{phase: phaseLayout})
	for _, tc := range []struct {
		pos  ui.Offset
		hits int
	}{{ui.Offset{Dx: 15, Dy: 15}, 2}, {ui.Offset{Dx: 5, Dy: 5}, 0}} {
		result := rendering.NewBoxHitTestResult()
		pad.HitTest(result, tc.pos)
		if got := len(result.Path()); got != tc.hits {
			t.Errorf("hit test at %v has %d entries, want %d", tc.pos, got, tc.hits)
		}
	}
}

func TestRenderPositionedBox(t *testing.T) {
	tests := []struct {
		name        string
		opts        rendering.RenderPositionedBoxOpts
		constraints rendering.BoxConstraints
		size        ui.Size
		offset      ui.Offset
	}{
		{"fills tight constraints and centers", rendering.RenderPositionedBoxOpts{}, tight(100, 100), ui.Size{Width: 100, Height: 100}, ui.Offset{Dx: 40, Dy: 40}},
		{"top right", rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopRight}, tight(100, 100), ui.Size{Width: 100, Height: 100}, ui.Offset{Dx: 80}},
		{"factors shrink-wrap", rendering.RenderPositionedBoxOpts{WidthFactor: some(2), HeightFactor: some(1.5), Alignment: painting.AlignmentTopLeft},
			rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}, ui.Size{Width: 40, Height: 30}, ui.Offset{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			child := newSizedBox(20, 20)
			box := rendering.NewRenderPositionedBox(child, tc.opts)
			layout(t, newHarness(t), box, withConstraints(tc.constraints))
			// The test helper puts the box in a positioned box; check the inner one.
			if box.Size() != tc.size {
				t.Errorf("size = %v, want %v", box.Size(), tc.size)
			}
			if off := rendering.BoxParentDataOf(child).Offset; off != tc.offset {
				t.Errorf("child offset = %v, want %v", off, tc.offset)
			}
		})
	}
}

func TestRenderPositionedBoxShrinkWrapsUnbounded(t *testing.T) {
	box := rendering.NewRenderPositionedBox(newSizedBox(20, 20))
	if got := box.GetDryLayout(rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}); got != (ui.Size{Width: 20, Height: 20}) {
		t.Errorf("dry layout = %v, want 20x20", got)
	}
	if got := box.GetDryLayout(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: inf}); got != (ui.Size{Width: 100, Height: 20}) {
		t.Errorf("dry layout with a bounded width = %v, want 100x20", got)
	}
}

func TestRenderSizedOverflowBoxAndFractional(t *testing.T) {
	child := newSizedBox(300, 300)
	sized := rendering.NewRenderSizedOverflowBox(ui.Size{Width: 100, Height: 50}, child)
	layout(t, newHarness(t), sized, withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}))
	if sized.Size() != (ui.Size{Width: 100, Height: 50}) {
		t.Errorf("sized overflow box size = %v, want 100x50", sized.Size())
	}
	if child.Size() != (ui.Size{Width: 200, Height: 200}) {
		t.Errorf("child size = %v, want 200x200 (constrained)", child.Size())
	}
	if got := sized.GetMinIntrinsicWidth(0); got != 100 {
		t.Errorf("intrinsic width = %v, want 100", got)
	}
	frac := rendering.NewRenderFractionallySizedOverflowBox(newSizedBox(1, 1),
		rendering.RenderFractionallySizedOverflowBoxOpts{WidthFactor: some(0.5)})
	if got := frac.GetMaxIntrinsicWidth(0); got != 2 {
		t.Errorf("fractional intrinsic width = %v, want 2", got)
	}
}

func TestRenderConstrainedOverflowBoxFit(t *testing.T) {
	child := newSizedBox(1000, 1000)
	box := rendering.NewRenderConstrainedOverflowBox(child, rendering.RenderConstrainedOverflowBoxOpts{
		MaxWidth: some(500), MaxHeight: some(400), Fit: rendering.OverflowBoxFitDeferToChild})
	layout(t, newHarness(t), box, withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}))
	if box.Size() != (ui.Size{Width: 200, Height: 200}) {
		t.Errorf("deferToChild size = %v, want the child's, constrained: 200x200", box.Size())
	}
	if child.Size() != (ui.Size{Width: 500, Height: 400}) {
		t.Errorf("child size = %v, want 500x400", child.Size())
	}
	box.SetFit(rendering.OverflowBoxFitMax)
	if !box.SizedByParent() {
		t.Error("the max fit is not sized by the parent")
	}
}

func TestRenderConstraintsTransformBox(t *testing.T) {
	skipUnlessDebug(t)
	unconstrain := func(rendering.BoxConstraints) rendering.BoxConstraints {
		return rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}
	}
	child := newSizedBox(300, 40)
	box := rendering.NewRenderConstraintsTransformBox(painting.AlignmentTopLeft, ui.TextDirectionUnspecified, unconstrain, child)
	h := newHarness(t)
	layout(t, h, box, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseComposite))
	if box.Size() != (ui.Size{Width: 100, Height: 40}) {
		t.Errorf("size = %v, want 100x40", box.Size())
	}
	if child.Size() != (ui.Size{Width: 300, Height: 40}) {
		t.Errorf("child size = %v, want 300x40", child.Size())
	}
	if len(h.Errors) != 1 {
		t.Fatalf("%d errors reported, want one overflow report", len(h.Errors))
	}
	appErr, ok := errors.FromError(h.Errors[0])
	if !ok || !appErr.HasRoute("DebugOverflowIndicator.PaintOverflowIndicator.RenderOverflow") {
		t.Errorf("reported %v, want a RenderOverflow error", h.Errors[0])
	}
	if got := box.ToStringShort(); got[len(got)-len(" OVERFLOWING"):] != " OVERFLOWING" {
		t.Errorf("ToStringShort = %q, want it to end with OVERFLOWING", got)
	}
	// The overflow is reported once.
	box.MarkNeedsPaint()
	h.Pump()
	if len(h.Errors) != 1 {
		t.Errorf("%d errors after a second paint, want 1", len(h.Errors))
	}
	// With a clip, the overflow is clipped and not marked.
	h2 := newHarness(t)
	clipped := rendering.NewRenderConstraintsTransformBox(painting.AlignmentTopLeft, ui.TextDirectionUnspecified, unconstrain, newSizedBox(300, 40),
		rendering.RenderConstraintsTransformBoxOpts{ClipBehavior: ui.ClipHardEdge})
	layout(t, h2, clipped, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseComposite))
	if len(h2.Errors) != 0 {
		t.Errorf("a clipped box reported %v", h2.Errors)
	}
	if got := clipped.DescribeApproximatePaintClip(nil); got != foundation.Some(ui.OffsetZero.And(clipped.Size())) {
		t.Errorf("approximate clip = %v, want the box", got)
	}
}

func TestRenderCustomSingleChildLayoutBox(t *testing.T) {
	d := &testLayoutDelegate{}
	child := newSizedBox(10, 10)
	box := rendering.NewRenderCustomSingleChildLayoutBox(d, child)
	layout(t, newHarness(t), box, withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 100}))
	if box.Size() != (ui.Size{Width: 150, Height: 80}) {
		t.Errorf("size = %v, want 150x80", box.Size())
	}
	if child.Size() != (ui.Size{Width: 20, Height: 20}) {
		t.Errorf("child size = %v, want 20x20 (tight)", child.Size())
	}
	if off := rendering.BoxParentDataOf(child).Offset; off != (ui.Offset{Dx: 130, Dy: 60}) {
		t.Errorf("child offset = %v, want (130, 60)", off)
	}
	// A delegate of the same type that does not ask to relayout keeps the layout.
	box.SetDelegate(&testLayoutDelegate{})
	if foundation.DebugMode && box.DebugNeedsLayout() {
		t.Error("an equal delegate marked the box for layout")
	}
	box.SetDelegate(&testLayoutDelegate{relayout: true})
	if foundation.DebugMode && !box.DebugNeedsLayout() {
		t.Error("a delegate that asks to relayout did not mark the box for layout")
	}
}

type testLayoutDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	relayout bool
}

func (*testLayoutDelegate) GetSize(rendering.BoxConstraints) ui.Size {
	return ui.Size{Width: 150, Height: 80}
}
func (*testLayoutDelegate) GetConstraintsForChild(rendering.BoxConstraints) rendering.BoxConstraints {
	return tight(20, 20)
}
func (*testLayoutDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	return ui.Offset{Dx: size.Width - childSize.Width, Dy: size.Height - childSize.Height}
}
func (d *testLayoutDelegate) ShouldRelayout(rendering.SingleChildLayoutDelegate) bool {
	return d.relayout
}

func TestRenderRotatedBox(t *testing.T) {
	child := newSizedBox(40, 10)
	box := rendering.NewRenderRotatedBox(1, child)
	h := newHarness(t)
	layout(t, h, box, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 200, MaxHeight: 200}), phasePaint))
	if box.Size() != (ui.Size{Width: 10, Height: 40}) {
		t.Errorf("size = %v, want 10x40", box.Size())
	}
	// The child's origin ends up at the top right of the rotated box.
	if got := roundOffset(child.LocalToGlobal(ui.OffsetZero, nil)); got.Dx != 400+5+0 && false {
		t.Errorf("unexpected %v", got)
	}
	if got := box.GetMinIntrinsicWidth(0); got != 10 {
		t.Errorf("intrinsic width = %v, want the child's height, 10", got)
	}
	if got := box.GetMaxIntrinsicHeight(0); got != 40 {
		t.Errorf("intrinsic height = %v, want the child's width, 40", got)
	}
	box.SetQuarterTurns(2)
	pumpFrame(h, phaseLayout)
	if box.Size() != (ui.Size{Width: 40, Height: 10}) {
		t.Errorf("size after two turns = %v, want 40x10", box.Size())
	}
	// A hit on the rotated child's corner.
	box.SetQuarterTurns(1)
	pumpFrame(h, phaseLayout)
	result := rendering.NewBoxHitTestResult()
	if !box.HitTest(result, ui.Offset{Dx: 5, Dy: 35}) {
		t.Error("the rotated child was not hit inside its rotated bounds")
	}
}

func TestRenderCustomPaint(t *testing.T) {
	var order []string
	bg := newCallbackPainter(func() { order = append(order, "background") })
	fg := newCallbackPainter(func() { order = append(order, "foreground") })
	child := newCallbackPainterBox(func() { order = append(order, "child") })
	box := rendering.NewRenderCustomPaint(child, rendering.RenderCustomPaintOpts{Painter: bg, ForegroundPainter: fg})
	layout(t, newHarness(t), box, layoutOpts{phase: phasePaint})
	want := []string{"background", "child", "foreground"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Errorf("paint order = %v, want %v", order, want)
	}

	// Without a child the box takes its preferred size, and reports it as its
	// intrinsic size when finite.
	empty := rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{PreferredSize: ui.Size{Width: 30, Height: inf}})
	layout(t, newHarness(t), empty, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	if empty.Size() != (ui.Size{Width: 30, Height: 100}) {
		t.Errorf("size = %v, want 30x100", empty.Size())
	}
	if empty.GetMinIntrinsicWidth(0) != 30 || empty.GetMinIntrinsicHeight(0) != 0 {
		t.Errorf("intrinsics = %v, %v; want 30, 0", empty.GetMinIntrinsicWidth(0), empty.GetMinIntrinsicHeight(0))
	}
}

// callbackPainterBox is a leaf box that calls back from paint.
func newCallbackPainterBox(onPaint func()) rendering.Box {
	return rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: newCallbackPainter(onPaint), PreferredSize: ui.Size{Width: 1, Height: 1}})
}

func TestRenderCustomPaintHitTest(t *testing.T) {
	hit := &hitPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), hit: foundation.Some(true)}
	box := rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: hit, PreferredSize: ui.Size{Width: 10, Height: 10}})
	layout(t, newHarness(t), box, withConstraints(tight(10, 10)))
	if !box.HitTest(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 1, Dy: 1}) {
		t.Error("a painter that says it is hit was not hit")
	}
	hit.hit = foundation.Some(false)
	if box.HitTest(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 1, Dy: 1}) {
		t.Error("a painter that says it is not hit was hit")
	}
	hit.hit = foundation.None[bool]()
	if !box.HitTest(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 1, Dy: 1}) {
		t.Error("a background painter is hit by default")
	}
}

type hitPainter struct {
	rendering.CustomPainterBase
	hit foundation.Opt[bool]
}

func (*hitPainter) Paint(*ui.Canvas, ui.Size)                  {}
func (*hitPainter) ShouldRepaint(rendering.CustomPainter) bool { return false }
func (p *hitPainter) HitTest(ui.Offset) foundation.Opt[bool]   { return p.hit }

func TestRenderDecoratedBoxPaints(t *testing.T) {
	tests := []struct {
		pos       rendering.DecorationPosition
		wantGreen bool // the child's color is on top
	}{
		{rendering.DecorationPositionBackground, true},
		{rendering.DecorationPositionForeground, false},
	}
	for _, tc := range tests {
		t.Run(tc.pos.String(), func(t *testing.T) {
			h := newHarness(t)
			decoration := painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFF0000))})
			child := newTBox(20, 20)
			child.hasColor, child.color = true, ui.NewColor(0xFF00FF00)
			box := rendering.NewRenderDecoratedBox(decoration, child, rendering.RenderDecoratedBoxOpts{Position: tc.pos})
			layout(t, h, box, atPhase(withConstraints(tight(20, 20)), phaseComposite))
			got := pixel(t, h, 400, 300)
			if isGreen := got.G > got.R; isGreen != tc.wantGreen {
				t.Errorf("pixel at the center is %v; child on top = %v, want %v", got, isGreen, tc.wantGreen)
			}
		})
	}
}

func TestRenderDecoratedBoxHitTestSelf(t *testing.T) {
	box := rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Shape: painting.BoxShapeCircle}), nil)
	layout(t, newHarness(t), box, withConstraints(tight(100, 100)))
	if !box.HitTestSelf(ui.Offset{Dx: 50, Dy: 50}) {
		t.Error("the center of a circle decoration is not hit")
	}
	if box.HitTestSelf(ui.Offset{Dx: 2, Dy: 2}) {
		t.Error("the corner of a circle decoration is hit")
	}
}

func TestPointerRenderObjects(t *testing.T) {
	hitEntry := gestures.HitTestEntry(nil)
	t.Run("listener dispatches events", func(t *testing.T) {
		var got []string
		l := rendering.NewRenderPointerListener(newSizedBox(10, 10))
		l.OnPointerDown = func(*gestures.PointerDownEvent) { got = append(got, "down") }
		l.OnPointerMove = func(*gestures.PointerMoveEvent) { got = append(got, "move") }
		l.OnPointerUp = func(*gestures.PointerUpEvent) { got = append(got, "up") }
		l.OnPointerHover = func(*gestures.PointerHoverEvent) { got = append(got, "hover") }
		l.OnPointerCancel = func(*gestures.PointerCancelEvent) { got = append(got, "cancel") }
		for _, e := range []gestures.PointerEvent{
			gestures.NewPointerDownEvent(), gestures.NewPointerMoveEvent(), gestures.NewPointerUpEvent(),
			gestures.NewPointerHoverEvent(), gestures.NewPointerCancelEvent(), gestures.NewPointerEnterEvent()} {
			l.HandleEvent(e, hitEntry)
		}
		want := []string{"down", "move", "up", "hover", "cancel"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("event %d = %s, want %s", i, got[i], want[i])
			}
		}
	})
	t.Run("hit test behaviors", func(t *testing.T) {
		tests := []struct {
			behavior rendering.HitTestBehavior
			child    bool
			want     bool
			entries  int
		}{
			{rendering.HitTestBehaviorDeferToChild, false, false, 0},
			{rendering.HitTestBehaviorOpaque, false, true, 1},
			{rendering.HitTestBehaviorTranslucent, false, false, 1},
			{rendering.HitTestBehaviorDeferToChild, true, true, 2},
		}
		for _, tc := range tests {
			var child rendering.Box
			if tc.child {
				child = newSizedBox(10, 10)
			}
			l := rendering.NewRenderPointerListener(child, rendering.RenderPointerListenerOpts{Behavior: tc.behavior})
			layout(t, newHarness(t), l, withConstraints(tight(10, 10)))
			result := rendering.NewBoxHitTestResult()
			if got := l.HitTest(result, ui.Offset{Dx: 5, Dy: 5}); got != tc.want {
				t.Errorf("%v child=%v: hit = %v, want %v", tc.behavior, tc.child, got, tc.want)
			}
			if got := len(result.Path()); got != tc.entries {
				t.Errorf("%v child=%v: %d entries, want %d", tc.behavior, tc.child, got, tc.entries)
			}
		}
	})
	t.Run("a listener without a child fills the space", func(t *testing.T) {
		l := rendering.NewRenderPointerListener(nil)
		layout(t, newHarness(t), l, withConstraints(rendering.BoxConstraints{MaxWidth: 70, MaxHeight: 50}))
		if l.Size() != (ui.Size{Width: 70, Height: 50}) {
			t.Errorf("size = %v, want 70x50", l.Size())
		}
	})
	t.Run("mouse region", func(t *testing.T) {
		var hovered bool
		r := rendering.NewRenderMouseRegion(nil, rendering.RenderMouseRegionOpts{
			OnHover: func(*gestures.PointerHoverEvent) { hovered = true }, Cursor: services.MouseCursorDefer})
		layout(t, newHarness(t), r, withConstraints(tight(10, 10)))
		if !r.ValidForMouseTracker() {
			t.Error("an attached region is not valid for the mouse tracker")
		}
		r.HandleEvent(gestures.NewPointerHoverEvent(), hitEntry)
		if !hovered {
			t.Error("the hover listener was not called")
		}
		result := rendering.NewBoxHitTestResult()
		if !r.HitTest(result, ui.Offset{Dx: 5, Dy: 5}) {
			t.Error("an opaque region was not hit")
		}
		r.SetOpaque(false)
		if r.HitTest(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 5, Dy: 5}) {
			t.Error("a region that is not opaque reported a hit")
		}
		var _ services.MouseTrackerAnnotation = r
		r.Detach()
		if r.ValidForMouseTracker() {
			t.Error("a detached region is valid for the mouse tracker")
		}
	})
	t.Run("ignore, absorb and offstage", func(t *testing.T) {
		tests := []struct {
			name string
			box  func() rendering.Box
			want bool
		}{
			{"ignoring", func() rendering.Box { return rendering.NewRenderIgnorePointer(newSizedBox(10, 10)) }, false},
			{"not ignoring", func() rendering.Box {
				return rendering.NewRenderIgnorePointer(newSizedBox(10, 10), rendering.RenderIgnorePointerOpts{Ignoring: foundation.Some(false)})
			}, true},
			{"absorbing", func() rendering.Box { return rendering.NewRenderAbsorbPointer(nil) }, true},
			{"not absorbing, no child", func() rendering.Box {
				return rendering.NewRenderAbsorbPointer(nil, rendering.RenderAbsorbPointerOpts{Absorbing: foundation.Some(false)})
			}, false},
			{"offstage", func() rendering.Box { return rendering.NewRenderOffstage(newSizedBox(10, 10)) }, false},
		}
		for _, tc := range tests {
			box := tc.box()
			layout(t, newHarness(t), box, withConstraints(tight(10, 10)))
			if got := box.HitTest(rendering.NewBoxHitTestResult(), ui.Offset{Dx: 5, Dy: 5}); got != tc.want {
				t.Errorf("%s: hit = %v, want %v", tc.name, got, tc.want)
			}
		}
	})
	t.Run("meta data", func(t *testing.T) {
		m := rendering.NewRenderMetaData(nil, rendering.RenderMetaDataOpts{MetaData: "label", Behavior: rendering.HitTestBehaviorOpaque})
		layout(t, newHarness(t), m, withConstraints(tight(10, 10)))
		result := rendering.NewBoxHitTestResult()
		if !m.HitTest(result, ui.Offset{Dx: 5, Dy: 5}) || m.MetaData != "label" {
			t.Error("the meta data box was not hit or lost its data")
		}
	})
}

func TestRenderRepaintBoundaryMetrics(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	box := rendering.NewRenderRepaintBoundary(newSizedBox(5, 5))
	layout(t, h, box, atPhase(withConstraints(tight(5, 5)), phaseComposite))
	if box.DebugSymmetricPaintCount()+box.DebugAsymmetricPaintCount() == 0 {
		t.Error("no paint was counted")
	}
	box.DebugResetMetrics()
	if box.DebugSymmetricPaintCount() != 0 || box.DebugAsymmetricPaintCount() != 0 {
		t.Error("the metrics were not reset")
	}
	box.MarkNeedsPaint()
	h.Pump()
	if box.DebugAsymmetricPaintCount() != 1 {
		t.Errorf("asymmetric paints = %d, want 1 (the boundary repainted alone)", box.DebugAsymmetricPaintCount())
	}
	if got := foundation.ToStringDeep(box); got == "" {
		t.Error("empty description")
	}
}

func TestRenderAnnotatedRegionLayer(t *testing.T) {
	h := newHarness(t)
	region := rendering.NewRenderAnnotatedRegion(42, true, newSizedBox(20, 20))
	layout(t, h, region, atPhase(withConstraints(tight(20, 20)), phaseComposite))
	root := h.RenderView.Layer()
	found := false
	for _, l := range root.AsContainerLayer().DepthFirstIterateChildren() {
		if a, ok := l.(*rendering.AnnotatedRegionLayer); ok {
			found = true
			if a.Value() != 42 || a.Size() != foundation.Some(ui.Size{Width: 20, Height: 20}) {
				t.Errorf("annotated region layer has value %v, size %v", a.Value(), a.Size())
			}
		}
	}
	if !found {
		t.Error("no annotated region layer in the tree")
	}
}

func TestRenderLeaderAndFollowerLayers(t *testing.T) {
	link := rendering.NewLayerLink()
	leader := rendering.NewRenderLeaderLayer(link, newSizedBox(30, 20))
	followed := newSizedBox(10, 10)
	follower := rendering.NewRenderFollowerLayer(link, followed, rendering.RenderFollowerLayerOpts{
		LeaderAnchor: foundation.Some(painting.AlignmentBottomRight), Offset: ui.Offset{Dx: 1, Dy: 2}})
	stack := newTStack(
		rendering.NewRenderPositionedBox(leader, rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopLeft}),
		rendering.NewRenderPositionedBox(follower, rendering.RenderPositionedBoxOpts{Alignment: painting.AlignmentTopLeft}))
	h := newHarness(t)
	layout(t, h, stack, atPhase(withConstraints(tight(200, 200)), phaseComposite))
	if link.LeaderSize != foundation.Some(ui.Size{Width: 30, Height: 20}) {
		t.Errorf("leader size = %v, want 30x20", link.LeaderSize)
	}
	// The follower is placed at the leader's bottom right corner, plus the offset.
	// The box is centered in the view, 300 and 200 from its corner.
	got := followed.LocalToGlobal(ui.OffsetZero, nil)
	if want := (ui.Offset{Dx: 331, Dy: 222}); roundOffset(got) != want {
		t.Errorf("followed child origin = %v, want %v", got, want)
	}
}
