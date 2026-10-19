//go:build !glutter_release && !glutter_profile

package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// requireTag fails unless call panics with an error that has the tag.
func requireTag(t *testing.T, tag string, call func()) {
	t.Helper()
	v := panicOf(call)
	if v == nil {
		t.Fatalf("expected a panic with tag %s", tag)
	}
	err, ok := v.(error)
	if !ok {
		t.Fatalf("panic value is %T (%v), want an error", v, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
}

// bareObject implements nothing but the base: its abstract methods panic.
type bareObject struct{ rendering.RenderObjectBase }

func newBare() *bareObject {
	o := &bareObject{}
	rendering.InitRenderObject(&o.RenderObjectBase, o)
	return o
}

// plainBox is a RenderBox that implements nothing.
type plainBox struct{ rendering.RenderBox }

func newPlain() *plainBox {
	b := &plainBox{}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

// scriptBox is a box whose layout, paint and others are scripted by the test.
type scriptBox struct {
	rendering.RenderBox
	sizedByParent bool
	onLayout      func(b *scriptBox)
	onResize      func(b *scriptBox)
	onPaint       func(b *scriptBox, ctx *rendering.PaintingContext, off ui.Offset)
	onDry         func(b *scriptBox, c rendering.BoxConstraints) ui.Size
	onBaseline    func(b *scriptBox, bl ui.TextBaseline) foundation.Opt[float64]
	onDryBaseline func(b *scriptBox, c rendering.BoxConstraints) foundation.Opt[float64]
	intrinsic     float64
	boundary      bool
}

func newScript() *scriptBox {
	b := &scriptBox{}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *scriptBox) SizedByParent() bool     { return b.sizedByParent }
func (b *scriptBox) IsRepaintBoundary() bool { return b.boundary }
func (b *scriptBox) PerformLayout() {
	if b.onLayout != nil {
		b.onLayout(b)
	} else if !b.sizedByParent {
		b.SetSize(b.BoxConstraints().Smallest())
	}
}
func (b *scriptBox) PerformResize() {
	if b.onResize != nil {
		b.onResize(b)
		return
	}
	b.RenderBox.PerformResize()
}
func (b *scriptBox) Paint(ctx *rendering.PaintingContext, off ui.Offset) {
	if b.onPaint != nil {
		b.onPaint(b, ctx, off)
	}
}
func (b *scriptBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	if b.onDry != nil {
		return b.onDry(b, c)
	}
	return c.Smallest()
}
func (b *scriptBox) ComputeDistanceToActualBaseline(bl ui.TextBaseline) foundation.Opt[float64] {
	if b.onBaseline != nil {
		return b.onBaseline(b, bl)
	}
	return b.RenderBox.ComputeDistanceToActualBaseline(bl)
}
func (b *scriptBox) ComputeDryBaseline(c rendering.BoxConstraints, bl ui.TextBaseline) foundation.Opt[float64] {
	if b.onDryBaseline != nil {
		return b.onDryBaseline(b, c)
	}
	return b.RenderBox.ComputeDryBaseline(c, bl)
}
func (b *scriptBox) ComputeMinIntrinsicWidth(float64) float64  { return b.intrinsic }
func (b *scriptBox) ComputeMaxIntrinsicWidth(float64) float64  { return b.intrinsic }
func (b *scriptBox) ComputeMinIntrinsicHeight(float64) float64 { return b.intrinsic }
func (b *scriptBox) ComputeMaxIntrinsicHeight(float64) float64 { return b.intrinsic }

var tight100 = rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})

func TestErrorsBox(t *testing.T) {
	checkHarness := func(t *testing.T) *renderingtest.Harness {
		dbg := rendering.NewDebugOptions()
		dbg.CheckIntrinsicSizes = true
		return renderingtest.New(t, renderingtest.Opts{Debug: dbg})
	}
	laidOut := func(t *testing.T) *scriptBox {
		b := newScript()
		b.Layout(tight100)
		return b
	}
	tests := []struct {
		tag  string
		call func(t *testing.T)
	}{
		{"BoxNotLaidOut", func(t *testing.T) { newTBox(1, 1).Size() }},
		{"BoxPerformLayoutMissing", func(t *testing.T) { newPlain().PerformLayout() }},
		{"BoxHitTestNoSize", func(t *testing.T) { newTBox(1, 1).HitTest(rendering.NewBoxHitTestResult(), ui.OffsetZero) }},
		{"BoxSizeSetterInvalid", func(t *testing.T) { newTBox(1, 1).SetSize(ui.Size{Width: 1, Height: 1}) }},
		{"BoxSizeNotSet", func(t *testing.T) {
			b := newScript()
			b.onLayout = func(*scriptBox) {}
			b.Layout(tight100)
		}},
		{"BoxSizeInfinite", func(t *testing.T) {
			b := newScript()
			b.onLayout = func(b *scriptBox) { b.SetSize(ui.Size{Width: math.Inf(1), Height: 1}) }
			b.Layout(rendering.NewBoxConstraints())
		}},
		{"BoxSizeViolatesConstraints", func(t *testing.T) {
			b := newScript()
			b.onLayout = func(b *scriptBox) { b.SetSize(ui.Size{Width: 500, Height: 500}) }
			b.Layout(tight100)
		}},
		{"BoxSizeAccessInvalid", func(t *testing.T) {
			child := newScript()
			parent := newScript()
			parent.AdoptChild(child)
			parent.onLayout = func(p *scriptBox) {
				child.Layout(tight100)
				_ = child.Size()
				p.SetSize(p.BoxConstraints().Smallest())
			}
			parent.Layout(tight100)
		}},
		{"BoxSizeAccessedInDryLayout", func(t *testing.T) {
			b := laidOut(t)
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return b.Size() }
			b.GetDryLayout(tight100)
		}},
		{"DryLayoutUnsupported", func(t *testing.T) { newPlain().GetDryLayout(tight100) }},
		{"DryLayoutReentrant", func(t *testing.T) {
			b := laidOut(t)
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size {
				return b.GetDryLayout(rendering.NewBoxConstraintsTight(ui.Size{Width: 3, Height: 3}))
			}
			b.GetDryLayout(tight100)
		}},
		{"IntrinsicArgumentNegative", func(t *testing.T) { newScript().GetMinIntrinsicWidth(-1) }},
		{"IntrinsicsDuringResize", func(t *testing.T) {
			b := newScript()
			b.sizedByParent = true
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return c.Smallest() }
			b.onResize = func(b *scriptBox) {
				b.GetMinIntrinsicWidth(1)
				b.RenderBox.PerformResize()
			}
			b.Layout(tight100)
		}},
		{"BoxIntrinsicsInvalid", func(t *testing.T) {
			h := checkHarness(t)
			b := newScript()
			b.sizedByParent = true
			b.intrinsic = -1
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return c.Smallest() }
			h.SetRoot(b)
			h.Pump()
		}},
		{"BoxDryLayoutMismatch", func(t *testing.T) {
			h := checkHarness(t)
			b := newScript()
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return ui.Size{Width: 1, Height: 1} }
			h.SetRoot(b)
			h.Pump()
		}},
		{"BoxDryBaselineMismatch", func(t *testing.T) {
			h := checkHarness(t)
			b := newScript()
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return c.Smallest() }
			b.onBaseline = func(b *scriptBox, _ ui.TextBaseline) foundation.Opt[float64] { return foundation.Some(10.0) }
			b.onDryBaseline = func(b *scriptBox, c rendering.BoxConstraints) foundation.Opt[float64] { return foundation.Some(20.0) }
			h.SetRoot(b)
			h.Pump()
		}},
		{"BoxDryBaselineMismatch", func(t *testing.T) { // a baseline on one side only
			h := checkHarness(t)
			b := newScript()
			b.onDry = func(b *scriptBox, c rendering.BoxConstraints) ui.Size { return c.Smallest() }
			b.onBaseline = func(b *scriptBox, _ ui.TextBaseline) foundation.Opt[float64] { return foundation.Some(10.0) }
			b.onDryBaseline = func(b *scriptBox, c rendering.BoxConstraints) foundation.Opt[float64] {
				return foundation.None[float64]()
			}
			h.SetRoot(b)
			h.Pump()
		}},
		{"BaselineWrongCall", func(t *testing.T) { laidOut(t).GetDistanceToActualBaseline(ui.TextBaselineAlphabetic) }},
		{"BaselineNeedsLayout", func(t *testing.T) { newScript().GetDistanceToBaseline(ui.TextBaselineAlphabetic) }},
		{"BaselineReentrant", func(t *testing.T) {
			b := laidOut(t)
			b.onBaseline = func(b *scriptBox, bl ui.TextBaseline) foundation.Opt[float64] { return b.GetDistanceToBaseline(bl) }
			b.GetDistanceToBaseline(ui.TextBaselineAlphabetic)
		}},
		{"BaselineWrongPhase", func(t *testing.T) {
			h := newHarness(t)
			b := newScript()
			h.SetRoot(b)
			h.Pump()
			b.GetDistanceToBaseline(ui.TextBaselineAlphabetic)
		}},
		{"BaselineNeedsLayout", func(t *testing.T) { // a container default
			row := newTRow(newTBox(1, 1))
			rendering.DefaultComputeDistanceToFirstActualBaseline(&row.ContainerRenderObject, ui.TextBaselineAlphabetic)
		}},
		{"ChildBaselineInvalid", func(t *testing.T) {
			rendering.ChildLayoutHelper.GetBaseline(newScript(), tight100, ui.TextBaselineAlphabetic)
		}},
		{"ApplyPaintTransformParentData", func(t *testing.T) {
			child := newTBox(1, 1)
			pad := newTPadding(10, child)
			child.SetParentData(&rendering.ParentDataBase{})
			pad.ApplyPaintTransform(child, ui.NewMatrix4Identity())
		}},
		{"RenderObjectNotChild", func(t *testing.T) { newTBox(1, 1).ApplyPaintTransform(newTBox(1, 1), ui.NewMatrix4Identity()) }},
		{"ContainerBoxParentDataMissing", func(t *testing.T) {
			child := newTBox(1, 1)
			row := newTRow(child)
			child.SetParentData(&rendering.ParentDataBase{})
			rendering.DefaultPaint(&row.ContainerRenderObject, rendering.NewPaintingContext(rendering.NewContainerLayer(), ui.Rect{}), ui.OffsetZero)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { requireTag(t, tc.tag, func() { tc.call(t) }) })
	}
}

func TestErrorsRenderObject(t *testing.T) {
	tests := []struct {
		tag  string
		call func(t *testing.T)
	}{
		{"RenderObjectAbstractMethod", func(t *testing.T) { newBare().PerformLayout() }},
		{"RenderObjectNoConstraints", func(t *testing.T) { newTObject(false).Constraints() }},
		{"RenderObjectNoParent", func(t *testing.T) { newTObject(false).MarkParentNeedsLayout() }},
		{"RenderObjectAttached", func(t *testing.T) {
			o := newTObject(false)
			owner := rendering.NewPipelineOwner()
			o.Attach(owner)
			o.Attach(owner)
		}},
		{"RenderObjectNotAttached", func(t *testing.T) { newTObject(false).Detach() }},
		{"RenderObjectHasParent", func(t *testing.T) {
			c := newTObject(false)
			newTObject(false).Add(c)
			newTObject(false).Add(c)
		}},
		{"RenderObjectCycle", func(t *testing.T) {
			a, b := newTObject(false), newTObject(false)
			a.Add(b)
			b.Add(a)
		}},
		{"RenderObjectDisposedTwice", func(t *testing.T) {
			o := newTObject(false)
			o.Dispose()
			o.Dispose()
		}},
		{"RenderObjectDisposedMutated", func(t *testing.T) {
			o := newTObject(false)
			o.Dispose()
			o.MarkNeedsPaint()
		}},
		{"RenderObjectCompositingBitsDirty", func(t *testing.T) {
			o := newTObject(false)
			o.MarkNeedsCompositingBitsUpdate()
			o.NeedsCompositing()
		}},
		{"RenderObjectOwnerMismatch", func(t *testing.T) {
			a, b := newTObject(false), newTObject(false)
			a.Attach(rendering.NewPipelineOwner())
			a.RedepthChild(b)
		}},
		{"RenderObjectBoundaryLayer", func(t *testing.T) {
			o := newTObject(false)
			o.repaintBoundary = true
			o.SetLayer(rendering.NewOffsetLayer())
		}},
		{"RenderObjectsNotInSameTree", func(t *testing.T) {
			owner := rendering.NewPipelineOwner()
			a, b := newTObject(false), newTObject(false)
			a.Attach(owner)
			b.Attach(owner)
			a.GetTransformTo(b)
		}},
		{"GetTransformToDetached", func(t *testing.T) { newTObject(false).GetTransformTo(nil) }},
		{"ScheduleInitialLayoutInvalid", func(t *testing.T) { newTObject(false).ScheduleInitialLayout() }},
		{"ScheduleInitialPaintInvalid", func(t *testing.T) { newTObject(false).ScheduleInitialPaint(rendering.NewOffsetLayer()) }},
		{"ReplaceRootLayerInvalid", func(t *testing.T) { newTObject(false).ReplaceRootLayer(rendering.NewOffsetLayer()) }},
		{"InvokeLayoutCallbackInvalid", func(t *testing.T) {
			o := newTObject(false)
			o.InvokeLayoutCallback(func(rendering.Constraints) {})
		}},
		{"RunLayoutCallbackInvalid", func(t *testing.T) {
			var m rendering.LayoutCallbackMixin
			rendering.InitLayoutCallbackMixin(&m, newTObject(false), func() {})
			m.RunLayoutCallback()
		}},
		{"UpdateCompositedLayerNotBoundary", func(t *testing.T) { newTObject(false).UpdateCompositedLayer(nil) }},
		{"UpdateLayerPropertiesInvalid", func(t *testing.T) { rendering.UpdateLayerProperties(newTObject(false)) }},
		{"RepaintNotNeeded", func(t *testing.T) {
			o := newTObject(true)
			o.repaintBoundary = true
			rendering.RepaintCompositedChild(o, true)
			o.Layout(rendering.NewBoxConstraintsTightForFinite())
			rendering.RepaintCompositedChild(o, true)
			rendering.RepaintCompositedChild(o, true)
		}},
		{"RepaintNotBoundary", func(t *testing.T) { rendering.RepaintCompositedChild(newTObject(true), true) }},
		{"RepaintLayerDetached", func(t *testing.T) {
			o := newTObject(true)
			o.repaintBoundary = true
			rendering.RepaintCompositedChild(o, true)
			o.Layout(rendering.NewBoxConstraintsTightForFinite())
			o.MarkNeedsPaint()
			rendering.RepaintCompositedChild(o, false)
		}},
		{"UpdateCompositedLayerNewInstance", func(t *testing.T) {
			o := newLayerObject(func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike { return rendering.NewOffsetLayer() })
			rendering.RepaintCompositedChild(o, true)
			o.Layout(rendering.NewBoxConstraintsTightForFinite())
			o.MarkNeedsPaint()
			rendering.RepaintCompositedChild(o, true)
		}},
		{"UpdateCompositedLayerOffsetChanged", func(t *testing.T) {
			o := newLayerObject(func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
				if old == nil {
					return rendering.NewOffsetLayer()
				}
				old.AsOffsetLayer().SetOffset(ui.Offset{Dx: 5})
				return old
			})
			rendering.RepaintCompositedChild(o, true)
			o.Layout(rendering.NewBoxConstraintsTightForFinite())
			o.MarkNeedsPaint()
			rendering.RepaintCompositedChild(o, true)
		}},
		{"MarkNeedsPaintDuringPaint", func(t *testing.T) {
			h := newHarness(t)
			b := newScript()
			b.onPaint = func(b *scriptBox, _ *rendering.PaintingContext, _ ui.Offset) { b.MarkNeedsPaint() }
			h.SetRoot(b)
			h.Pump()
		}},
		{"PaintMarkedDirty", func(t *testing.T) {
			h := newHarness(t)
			b := newScript()
			b.onPaint = func(b *scriptBox, _ *rendering.PaintingContext, _ ui.Offset) { b.MarkNeedsLayout() }
			h.SetRoot(b)
			h.Pump()
		}},
		{"PaintReentrant", func(t *testing.T) {
			h := newHarness(t)
			b := newScript()
			b.onPaint = func(b *scriptBox, ctx *rendering.PaintingContext, off ui.Offset) { ctx.PaintChild(b, off) }
			h.SetRoot(b)
			h.Pump()
		}},
		{"PaintBeforeCompositingBits", func(t *testing.T) {
			h := newHarness(t)
			child := newTBox(1, 1)
			parent := newTProxy(child)
			h.SetRoot(parent)
			h.Pump()
			child.MarkNeedsCompositingBitsUpdate()
			parent.MarkNeedsPaint()
			h.Owner.FlushPaint()
		}},
		{"LayoutReentrant", func(t *testing.T) {
			b := newScript()
			b.onLayout = func(b *scriptBox) { b.Layout(rendering.NewBoxConstraintsTight(ui.Size{Width: 5, Height: 5})) }
			b.Layout(tight100)
		}},
		{"RenderObjectMutatedDuringLayout", func(t *testing.T) {
			h := newHarness(t)
			other := newTObject(false)
			other.Attach(h.Owner)
			b := newScript()
			b.onLayout = func(b *scriptBox) {
				other.Add(newTObject(false))
				b.SetSize(b.BoxConstraints().Smallest())
			}
			h.SetRoot(b)
			h.Pump()
		}},
		{"PaintingContextRecording", func(t *testing.T) {
			ctx := rendering.NewPaintingContext(rendering.NewContainerLayer(), ui.Rect{})
			ctx.Canvas()
			ctx.AppendLayer(rendering.NewContainerLayer())
		}},
		{"ContainerParentDataMissing", func(t *testing.T) {
			o := newTObject(false)
			c := newTBox(1, 1)
			c.SetParentData(&rendering.ParentDataBase{})
			wrong := &wrongData{}
			wrong.host = newWrongContainer()
			wrong.host.Add(c)
			_ = o
		}},
		{"ContainerInsertInvalid", func(t *testing.T) {
			row := newTRow()
			row.Insert(row, nil)
		}},
		{"ContainerMoveInvalid", func(t *testing.T) {
			row := newTRow()
			row.Move(newTBox(1, 1), nil)
		}},
		{"ParentDataSiblingsSet", func(t *testing.T) {
			d := &rendering.ContainerBoxParentData[rendering.RenderObject]{}
			d.NextSibling = newTObject(false)
			d.Detach()
		}},
		{"RenderDebugVarsSet", func(t *testing.T) {
			o := rendering.NewDebugOptions()
			o.PaintSizeEnabled = true
			o.DebugAssertAllRenderVarsUnset("a flag is set", false)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { requireTag(t, tc.tag, func() { tc.call(t) }) })
	}
}

// layerObject is a repaint boundary with a scripted UpdateCompositedLayer.
type layerObject struct {
	rendering.RenderObjectBase
	update func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike
}

func newLayerObject(update func(old rendering.OffsetLayerLike) rendering.OffsetLayerLike) *layerObject {
	o := &layerObject{update: update}
	rendering.InitRenderObject(&o.RenderObjectBase, o)
	return o
}
func (o *layerObject) IsRepaintBoundary() bool { return true }
func (o *layerObject) PaintBounds() ui.Rect    { return ui.Rect{} }
func (o *layerObject) PerformLayout()          {}
func (o *layerObject) UpdateCompositedLayer(old rendering.OffsetLayerLike) rendering.OffsetLayerLike {
	return o.update(old)
}

// wrongData and wrongContainer: a container whose children's parent data does
// not embed ContainerParentData.
type wrongData struct{ host *wrongContainer }

type wrongContainer struct {
	rendering.RenderObjectBase
	rendering.ContainerRenderObject[rendering.RenderObject]
}

func newWrongContainer() *wrongContainer {
	c := &wrongContainer{}
	rendering.InitRenderObject(&c.RenderObjectBase, c)
	rendering.InitContainerRenderObject(&c.ContainerRenderObject, c)
	return c
}
func (c *wrongContainer) SetupParentData(child rendering.RenderObject) {
	child.AsRenderObject().SetParentData(&rendering.ParentDataBase{})
}
func (c *wrongContainer) PerformLayout()       {}
func (c *wrongContainer) PaintBounds() ui.Rect { return ui.Rect{} }

func TestErrorsPipelineOwnerAndBinding(t *testing.T) {
	tests := []struct {
		tag  string
		call func(t *testing.T)
	}{
		{"PipelineOwnerAttached", func(t *testing.T) {
			h := newHarness(t)
			h.Owner.Attach(nil)
		}},
		{"PipelineOwnerNotAttached", func(t *testing.T) { rendering.NewPipelineOwner().Detach() }},
		{"PipelineOwnerHasParent", func(t *testing.T) {
			a, b, c := rendering.NewPipelineOwner(), rendering.NewPipelineOwner(), rendering.NewPipelineOwner()
			a.AdoptChild(c)
			b.AdoptChild(c)
		}},
		{"PipelineOwnerNotChild", func(t *testing.T) { rendering.NewPipelineOwner().DropChild(rendering.NewPipelineOwner()) }},
		{"PipelineOwnerChildListLocked", func(t *testing.T) {
			parent, child := rendering.NewPipelineOwner(), rendering.NewPipelineOwner()
			parent.AdoptChild(child)
			node := newScript()
			node.onLayout = func(b *scriptBox) {
				parent.AdoptChild(rendering.NewPipelineOwner())
				b.SetSize(b.BoxConstraints().Smallest())
			}
			child.SetRootNode(node)
			node.ScheduleInitialLayout()
			parent.FlushLayout()
		}},
		{"PipelineOwnerDisposeBusy", func(t *testing.T) {
			o := rendering.NewPipelineOwner()
			o.SetRootNode(newTObject(false))
			o.Dispose()
		}},
		{"RootNodeNotAllowed", func(t *testing.T) {
			h := newHarness(t)
			h.Binding.RootPipelineOwner().SetRootNode(newTObject(false))
		}},
		{"AllowFirstFrameUnbalanced", func(t *testing.T) { newHarness(t).Binding.AllowFirstFrame() }},
		{"RenderViewAlreadyAdded", func(t *testing.T) {
			h := newHarness(t)
			h.Binding.AddRenderView(rendering.NewRenderView(h.View))
		}},
		{"RenderViewNotAdded", func(t *testing.T) {
			h := newHarness(t)
			h.Binding.RemoveRenderView(rendering.NewRenderView(h.View))
		}},
		{"RenderViewNoConfiguration", func(t *testing.T) {
			h := newHarness(t)
			rendering.NewRenderView(h.View).Configuration()
		}},
		{"RenderViewNotPrepared", func(t *testing.T) {
			h := newHarness(t)
			rendering.NewRenderView(h.View).PerformLayout()
		}},
		{"RenderViewResize", func(t *testing.T) { rendering.NewRenderView(newHarness(t).View).PerformResize() }},
		{"PrepareInitialFrameInvalid", func(t *testing.T) { rendering.NewRenderView(newHarness(t).View).PrepareInitialFrame() }},
		{"CompositeFrameNotPrepared", func(t *testing.T) { _ = rendering.NewRenderView(newHarness(t).View).CompositeFrame() }},
		{"ViewportUnboundedMainAxis", func(t *testing.T) {
			rendering.DebugCheckHasBoundedAxis(painting.AxisVertical, rendering.BoxConstraints{MaxWidth: 10, MaxHeight: math.Inf(1)})
		}},
		{"ViewportUnboundedMainAxis", func(t *testing.T) {
			rendering.DebugCheckHasBoundedAxis(painting.AxisHorizontal, rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: 10})
		}},
		{"ViewportUnboundedCrossAxis", func(t *testing.T) {
			rendering.DebugCheckHasBoundedAxis(painting.AxisVertical, rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: 10})
		}},
		{"ViewportUnboundedCrossAxis", func(t *testing.T) {
			rendering.DebugCheckHasBoundedAxis(painting.AxisHorizontal, rendering.BoxConstraints{MaxWidth: 10, MaxHeight: math.Inf(1)})
		}},
		{"OutOfBandOptionsInvalid", func(t *testing.T) {
			rendering.NewBoxHitTestResult().AddWithOutOfBandPosition(rendering.OutOfBandOpts{}, func(*rendering.BoxHitTestResult) bool { return true })
		}},
		{"OutOfBandTransformNotInvertible", func(t *testing.T) {
			rendering.NewBoxHitTestResult().AddWithOutOfBandPosition(
				rendering.OutOfBandOpts{PaintTransform: ui.NewMatrix4ForceToPoint(ui.OffsetZero)},
				func(*rendering.BoxHitTestResult) bool { return true })
		}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { requireTag(t, tc.tag, func() { tc.call(t) }) })
	}
}

// plainLayer is a layer that does not implement AddToScene.
type plainLayer struct{ rendering.LayerBase }

func newPlainLayer() *plainLayer {
	l := &plainLayer{}
	rendering.InitLayer(&l.LayerBase, l)
	return l
}

func TestErrorsLayers(t *testing.T) {
	disposed := func() *concreteLayer {
		l := newConcreteLayer()
		l.Dispose()
		return l
	}
	tests := []struct {
		tag  string
		call func(t *testing.T)
	}{
		{"LayerAddToSceneMissing", func(t *testing.T) { newPlainLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerAlreadyAttached", func(t *testing.T) {
			l := newConcreteLayer()
			l.Attach(new(int))
			l.Attach(new(int))
		}},
		{"LayerNotAttached", func(t *testing.T) { newConcreteLayer().Detach() }},
		{"LayerAlwaysNeedsAddToScene", func(t *testing.T) { newAlwaysLayer().MarkNeedsAddToScene() }},
		{"LayerDisposedTwice", func(t *testing.T) { disposed().Dispose() }},
		{"LayerDisposedWithHandles", func(t *testing.T) {
			l := newConcreteLayer()
			rendering.NewLayerHandle[rendering.Layer](l)
			l.Dispose()
		}},
		{"LayerDisposedUsed", func(t *testing.T) { disposed().SetEngineLayer(nil) }},
		{"LayerDisposedUsed", func(t *testing.T) { disposed().MarkNeedsAddToScene() }},
		{"LayerHandleDisposed", func(t *testing.T) {
			rendering.NewLayerHandle[rendering.Layer](nil).SetLayer(disposed())
		}},
		{"LayerAppendInvalid", func(t *testing.T) {
			c := rendering.NewContainerLayer()
			c.Append(c)
		}},
		{"LayerAppendInvalid", func(t *testing.T) {
			p1, p2, c := rendering.NewContainerLayer(), rendering.NewContainerLayer(), newConcreteLayer()
			p1.Append(c)
			p2.Append(c)
		}},
		{"LayerCompositionCallbackRemoved", func(t *testing.T) {
			l := rendering.NewContainerLayer()
			remove := l.AddCompositionCallback(func(rendering.Layer) {})
			remove()
			remove()
		}},
		{"LayerMutationLocked", func(t *testing.T) {
			root := rendering.NewContainerLayer()
			root.AddCompositionCallback(func(l rendering.Layer) { l.AsLayer().Remove() })
			root.BuildScene(ui.NewSceneBuilder())
		}},
		{"LayerLinkLeaderUnknown", func(t *testing.T) { rendering.NewLeaderLayer(rendering.NewLayerLink()).Detach() }},
		{"FollowerLayerLeaderOwner", func(t *testing.T) {
			link := rendering.NewLayerLink()
			leader, follower := rendering.NewLeaderLayer(link), rendering.NewFollowerLayer(link)
			leader.Attach(new(int))
			follower.Attach(new(int))
			follower.AddToScene(ui.NewSceneBuilder())
		}},
		{"FollowerLayerNoCommonAncestor", func(t *testing.T) {
			link := rendering.NewLayerLink()
			owner := new(int)
			r1, r2 := rendering.NewContainerLayer(), rendering.NewContainerLayer()
			r1.Attach(owner)
			r2.Attach(owner)
			leader, follower := rendering.NewLeaderLayer(link), rendering.NewFollowerLayer(link)
			r1.Append(leader)
			r2.Append(follower)
			follower.AddToScene(ui.NewSceneBuilder())
		}},
		{"FollowerLayerLeaderAfterFollower", func(t *testing.T) {
			link := rendering.NewLayerLink()
			root := rendering.NewContainerLayer()
			root.Attach(new(int))
			leader, follower := rendering.NewLeaderLayer(link), rendering.NewFollowerLayer(link)
			root.Append(follower)
			root.Append(leader)
			follower.AddToScene(ui.NewSceneBuilder())
		}},
		{"ClipLayerBehaviorNone", func(t *testing.T) {
			rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipBehavior: ui.ClipNone})
		}},
		{"ClipLayerBehaviorNone", func(t *testing.T) { rendering.NewClipPathLayer().SetClipBehavior(ui.ClipNone) }},
		{"TransformLayerNotFinite", func(t *testing.T) {
			m := ui.NewMatrix4Identity()
			m.SetEntry(0, 0, float32(math.NaN()))
			rendering.NewTransformLayer().SetTransform(m)
		}},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewClipRectLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewClipRRectLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewClipRSuperellipseLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewClipPathLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewColorFilterLayer(nil).AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewColorFilterLayer(nil).SetColorFilter(nil) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewImageFilterLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewImageFilterLayer().SetImageFilter(nil) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewTransformLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewTransformLayer().SetTransform(nil) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewOpacityLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewShaderMaskLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewBackdropFilterLayer().AddToScene(ui.NewSceneBuilder()) }},
		{"LayerPropertyMissing", func(t *testing.T) { rendering.NewPictureLayer(ui.Rect{}).AddToScene(ui.NewSceneBuilder()) }},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) { requireTag(t, tc.tag, func() { tc.call(t) }) })
	}
}

func TestErrorsRouteHasTheTagOfTheFailure(t *testing.T) {
	// The route ends with the tag; a wrapped panic keeps it.
	v := panicOf(func() { newTObject(false).Constraints() })
	err, _ := v.(error)
	requireRoute(t, err, "RenderObjectNoConstraints")
	var _ = gestures.NewHitTestResult
}
