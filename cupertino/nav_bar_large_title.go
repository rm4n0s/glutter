package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// largeTitle lays out the large title of a navigation bar: at the bottom start corner of the space, and
// scaled up to 1.1 times when the bar is stretched by an overscroll.
//
// Flutter: cupertino/nav_bar.dart _LargeTitle
type largeTitle struct {
	widgets.SingleChildRenderObjectWidget
	height float64
}

func newLargeTitle(child widgets.Widget, height float64) *largeTitle {
	return &largeTitle{widgets.NewSingleChildRenderObjectWidgetBase(child, widgets.WidgetOpts{}), height}
}

func largeTitleAlignment(bc widgets.BuildContext) painting.Alignment {
	return painting.AlignmentDirectionalBottomStart.Resolve(widgets.DirectionalityOf(bc))
}

// CreateRenderObject creates the render object.
//
// Flutter: cupertino/nav_bar.dart _LargeTitle.createRenderObject
func (w *largeTitle) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderLargeTitle(largeTitleAlignment(bc), w.height)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: cupertino/nav_bar.dart _LargeTitle.updateRenderObject
func (w *largeTitle) UpdateRenderObject(bc widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderLargeTitle)
	r.setAlignment(largeTitleAlignment(bc))
	r.setHeight(w.height)
}

// renderLargeTitle is the render box of largeTitle.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle
type renderLargeTitle struct {
	rendering.RenderShiftedBox
	alignment painting.Alignment
	height    float64
	scale     float64
}

func newRenderLargeTitle(alignment painting.Alignment, height float64) *renderLargeTitle {
	r := &renderLargeTitle{alignment: alignment, height: height, scale: 1.0}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (r *renderLargeTitle) setAlignment(value painting.Alignment) {
	if r.alignment == value {
		return
	}
	r.alignment = value
	r.MarkNeedsLayout()
}

func (r *renderLargeTitle) setHeight(value float64) {
	if r.height == value {
		return
	}
	r.height = value
	r.MarkNeedsLayout()
}

func clampDouble(v, min, max float64) float64 {
	switch {
	case v < min:
		return min
	case v > max:
		return max
	}
	return v
}

// computeTitleScale is the scale of the title: it grows by 3% of the extra height of the bar, but the title
// stays within the width and grows at most 1.1 times.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle._computeTitleScale
func computeTitleScale(childSize ui.Size, constraints rendering.BoxConstraints, height float64) float64 {
	maxHeight := height - navBarBottomPadding
	scale := 1.0 + 0.03*(constraints.MaxHeight-maxHeight)/maxHeight
	maxScale := 1.1
	if childSize.Width != 0.0 {
		maxScale = clampDouble(constraints.MaxWidth/childSize.Width, 1.0, 1.1)
	}
	return clampDouble(scale, 1.0, maxScale)
}

// ComputeDistanceToActualBaseline is the baseline of the child, scaled.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.computeDistanceToActualBaseline
func (r *renderLargeTitle) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	distance, ok := child.GetDistanceToActualBaseline(baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(rendering.BoxParentDataOf(child).Offset.Dy + distance*r.scale)
}

// ComputeDryBaseline is the baseline of the child for the constraints, scaled and aligned.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.computeDryBaseline
func (r *renderLargeTitle) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	childConstraints := constraints.WidthConstraints().Loosen()
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(childConstraints)
	scale := computeTitleScale(childSize, constraints, r.height)
	scaledChildSize := childSize.Mul(scale)
	biggest := constraints.Biggest()
	return foundation.Some(result*scale + r.alignment.AlongOffset(ui.Offset{Dx: biggest.Width - scaledChildSize.Width, Dy: biggest.Height - scaledChildSize.Height}).Dy)
}

// PerformLayout takes the biggest size and puts the scaled child in the alignment.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.performLayout
func (r *renderLargeTitle) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(constraints.Biggest())
	child := r.Child()
	if child == nil {
		return
	}
	childConstraints := constraints.WidthConstraints().Loosen()
	child.Layout(childConstraints, rendering.LayoutOpts{ParentUsesSize: true})
	r.scale = computeTitleScale(child.Size(), constraints, r.height)
	size := r.Size()
	rendering.BoxParentDataOf(child).Offset = r.alignment.AlongOffset(ui.Offset{
		Dx: size.Width - child.Size().Width*r.scale, Dy: size.Height - child.Size().Height*r.scale})
}

// ApplyPaintTransform adds the offset of the child and its scale.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.applyPaintTransform
func (r *renderLargeTitle) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	r.RenderBox.ApplyPaintTransform(child, transform)
	transform.Scale(float32(r.scale), float32(r.scale), float32(r.scale))
}

// Paint paints the child scaled, from the point of its offset.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.paint
func (r *renderLargeTitle) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	child := r.Child()
	if child == nil {
		r.SetLayer(nil)
		return
	}
	childOffset := rendering.BoxParentDataOf(child).Offset
	old, _ := r.Layer().(*rendering.TransformLayer)
	layer := context.PushTransform(r.NeedsCompositing(), offset.Add(childOffset), ui.NewMatrix4Diagonal3Values(float32(r.scale), float32(r.scale), 1.0),
		func(context *rendering.PaintingContext, offset ui.Offset) { context.PaintChild(child, offset) }, old)
	if layer == nil {
		r.SetLayer(nil)
	} else {
		r.SetLayer(layer)
	}
}

// HitTestChildren hit tests the child through the inverse of its scale.
//
// Flutter: cupertino/nav_bar.dart _RenderLargeTitle.hitTestChildren
func (r *renderLargeTitle) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	child := r.Child()
	if child == nil {
		return false
	}
	childOffset := rendering.BoxParentDataOf(child).Offset
	transform := ui.NewMatrix4Identity()
	transform.Scale(float32(1.0/r.scale), float32(1.0/r.scale), 1.0)
	transform.Translate(float32(-childOffset.Dx), float32(-childOffset.Dy), 0)
	return result.AddWithRawTransform(transform, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
		return child.HitTest(result, transformed)
	})
}
