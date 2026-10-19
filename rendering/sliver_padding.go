package rendering

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// edgeInsetsPaddingHooks is what a concrete RenderSliverEdgeInsetsPadding gives
// to the base.
type edgeInsetsPaddingHooks interface {
	ResolvedPadding() foundation.Opt[painting.EdgeInsets]
}

// RenderSliverEdgeInsetsPadding is the base of the slivers that inset a sliver
// child by edge insets. The concrete type embeds it, calls
// InitRenderSliverEdgeInsetsPadding and implements ResolvedPadding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding
type RenderSliverEdgeInsetsPadding struct {
	RenderSliver
	RenderObjectWithChild[Sliver]
}

// InitRenderSliverEdgeInsetsPadding sets the outer object, which must have a
// ResolvedPadding method.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding
func InitRenderSliverEdgeInsetsPadding(p *RenderSliverEdgeInsetsPadding, self Sliver) {
	InitRenderSliver(&p.RenderSliver, self)
	InitRenderObjectWithChild(&p.RenderObjectWithChild, self)
}

func (p *RenderSliverEdgeInsetsPadding) resolvedPadding() painting.EdgeInsets {
	return p.self.(edgeInsetsPaddingHooks).ResolvedPadding().Value()
}

func (p *RenderSliverEdgeInsetsPadding) growthAxisDirection() painting.AxisDirection {
	c := p.SliverConstraints()
	return ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection)
}

// BeforePadding is the padding before the child along the main axis, in the
// growth direction.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.beforePadding
func (p *RenderSliverEdgeInsetsPadding) BeforePadding() float64 {
	r := p.resolvedPadding()
	switch p.growthAxisDirection() {
	case painting.AxisDirectionUp:
		return r.Bottom
	case painting.AxisDirectionRight:
		return r.Left
	case painting.AxisDirectionLeft:
		return r.Right
	}
	return r.Top
}

// AfterPadding is the padding after the child along the main axis, in the growth
// direction.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.afterPadding
func (p *RenderSliverEdgeInsetsPadding) AfterPadding() float64 {
	r := p.resolvedPadding()
	switch p.growthAxisDirection() {
	case painting.AxisDirectionUp:
		return r.Top
	case painting.AxisDirectionRight:
		return r.Right
	case painting.AxisDirectionLeft:
		return r.Left
	}
	return r.Bottom
}

// MainAxisPadding is the padding along the main axis.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.mainAxisPadding
func (p *RenderSliverEdgeInsetsPadding) MainAxisPadding() float64 {
	return p.resolvedPadding().Along(p.SliverConstraints().Axis())
}

// CrossAxisPadding is the padding along the cross axis.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.crossAxisPadding
func (p *RenderSliverEdgeInsetsPadding) CrossAxisPadding() float64 {
	r := p.resolvedPadding()
	if p.SliverConstraints().Axis() == painting.AxisHorizontal {
		return r.Vertical()
	}
	return r.Horizontal()
}

// SetupParentData gives the child a SliverPhysicalParentData if it has none.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.setupParentData
func (p *RenderSliverEdgeInsetsPadding) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(sliverPhysicalHolder); !ok {
		child.AsRenderObject().SetParentData(&SliverPhysicalParentData{})
	}
}

// PerformLayout lays the child out with the padding taken off the constraints,
// and gives the sliver the geometry of the child plus the padding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.performLayout
func (p *RenderSliverEdgeInsetsPadding) PerformLayout() {
	c := p.SliverConstraints()
	paintOffset := func(from, to float64) float64 { return p.CalculatePaintOffset(c, from, to) }
	cacheOffset := func(from, to float64) float64 { return p.CalculateCacheOffset(c, from, to) }

	resolved := p.resolvedPadding()
	beforePadding, mainAxisPadding, crossAxisPadding := p.BeforePadding(), p.MainAxisPadding(), p.CrossAxisPadding()
	child := p.Child()
	if isNil(child) {
		paintExtent := paintOffset(0, mainAxisPadding)
		cacheExtent := cacheOffset(0, mainAxisPadding)
		p.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
			ScrollExtent:   mainAxisPadding,
			PaintExtent:    math.Min(paintExtent, c.RemainingPaintExtent),
			MaxPaintExtent: mainAxisPadding,
			CacheExtent:    foundation.Some(cacheExtent),
		}))
		return
	}
	beforePaddingPaintExtent := paintOffset(0, beforePadding)
	overlap := c.Overlap
	if overlap > 0 {
		overlap = math.Max(0, c.Overlap-beforePaddingPaintExtent)
	}
	child.Layout(c.CopyWith(SliverConstraintsCopy{
		ScrollOffset:          foundation.Some(math.Max(0, c.ScrollOffset-beforePadding)),
		CacheOrigin:           foundation.Some(math.Min(0, c.CacheOrigin+beforePadding)),
		Overlap:               foundation.Some(overlap),
		RemainingPaintExtent:  foundation.Some(c.RemainingPaintExtent - paintOffset(0, beforePadding)),
		RemainingCacheExtent:  foundation.Some(c.RemainingCacheExtent - cacheOffset(0, beforePadding)),
		CrossAxisExtent:       foundation.Some(math.Max(0, c.CrossAxisExtent-crossAxisPadding)),
		PrecedingScrollExtent: foundation.Some(beforePadding + c.PrecedingScrollExtent),
	}), LayoutOpts{ParentUsesSize: true})
	childGeometry := child.Geometry()
	if correction, ok := childGeometry.ScrollOffsetCorrection.Get(); ok {
		p.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(correction)}))
		return
	}

	scrollExtent := childGeometry.ScrollExtent
	beforePaddingCacheExtent := cacheOffset(0, beforePadding)
	afterPaddingCacheExtent := cacheOffset(beforePadding+scrollExtent, mainAxisPadding+scrollExtent)
	afterPaddingPaintExtent := paintOffset(beforePadding+scrollExtent, mainAxisPadding+scrollExtent)
	mainAxisPaddingCacheExtent := beforePaddingCacheExtent + afterPaddingCacheExtent
	mainAxisPaddingPaintExtent := beforePaddingPaintExtent + afterPaddingPaintExtent
	paintExtent := math.Min(
		beforePaddingPaintExtent+math.Max(childGeometry.PaintExtent, childGeometry.LayoutExtent+afterPaddingPaintExtent),
		c.RemainingPaintExtent)
	p.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		PaintOrigin:       childGeometry.PaintOrigin,
		ScrollExtent:      mainAxisPadding + scrollExtent,
		PaintExtent:       paintExtent,
		LayoutExtent:      foundation.Some(math.Min(mainAxisPaddingPaintExtent+childGeometry.LayoutExtent, paintExtent)),
		CacheExtent:       foundation.Some(math.Min(mainAxisPaddingCacheExtent+childGeometry.CacheExtent, c.RemainingCacheExtent)),
		MaxPaintExtent:    mainAxisPadding + childGeometry.MaxPaintExtent,
		HitTestExtent:     foundation.Some(math.Max(mainAxisPaddingPaintExtent+childGeometry.PaintExtent, beforePaddingPaintExtent+childGeometry.HitTestExtent)),
		HasVisualOverflow: childGeometry.HasVisualOverflow,
	}))

	var calculatedOffset float64
	switch p.growthAxisDirection() {
	case painting.AxisDirectionUp:
		calculatedOffset = paintOffset(resolved.Bottom+scrollExtent, resolved.Vertical()+scrollExtent)
	case painting.AxisDirectionLeft:
		calculatedOffset = paintOffset(resolved.Right+scrollExtent, resolved.Horizontal()+scrollExtent)
	case painting.AxisDirectionRight:
		calculatedOffset = paintOffset(0, resolved.Left)
	default:
		calculatedOffset = paintOffset(0, resolved.Top)
	}
	data := SliverPhysicalParentDataOf(child)
	if c.Axis() == painting.AxisHorizontal {
		data.PaintOffset = ui.Offset{Dx: calculatedOffset, Dy: resolved.Top}
	} else {
		data.PaintOffset = ui.Offset{Dx: resolved.Left, Dy: calculatedOffset}
	}
}

// HitTestChildren hit tests the child, if its hit test extent reaches.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.hitTestChildren
func (p *RenderSliverEdgeInsetsPadding) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	child := p.Child()
	if !isNil(child) && child.Geometry().HitTestExtent > 0 {
		offset := SliverPhysicalParentDataOf(child).PaintOffset
		return result.AddWithAxisOffset(&offset, p.self.(Sliver).ChildMainAxisPosition(child), p.self.(Sliver).ChildCrossAxisPosition(child),
			mainAxisPosition, crossAxisPosition, child.HitTest)
	}
	return false
}

// ChildMainAxisPosition is the paint extent of the padding before the child.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.childMainAxisPosition
func (p *RenderSliverEdgeInsetsPadding) ChildMainAxisPosition(RenderObject) float64 {
	return p.CalculatePaintOffset(p.SliverConstraints(), 0, p.BeforePadding())
}

// ChildCrossAxisPosition is the padding before the child in the cross axis.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.childCrossAxisPosition
func (p *RenderSliverEdgeInsetsPadding) ChildCrossAxisPosition(RenderObject) float64 {
	if p.SliverConstraints().Axis() == painting.AxisHorizontal {
		return p.resolvedPadding().Top
	}
	return p.resolvedPadding().Left
}

// ChildScrollOffset is the padding before the child.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.childScrollOffset
func (p *RenderSliverEdgeInsetsPadding) ChildScrollOffset(RenderObject) foundation.Opt[float64] {
	return foundation.Some(p.BeforePadding())
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.applyPaintTransform
func (p *RenderSliverEdgeInsetsPadding) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// Paint paints the child at its paint offset if it is visible.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.paint
func (p *RenderSliverEdgeInsetsPadding) Paint(context *PaintingContext, offset ui.Offset) {
	child := p.Child()
	if !isNil(child) && child.Geometry().Visible {
		context.PaintChild(child, offset.Add(SliverPhysicalParentDataOf(child).PaintOffset))
	}
}

// DebugPaint also paints the padding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverEdgeInsetsPadding.debugPaint
func (p *RenderSliverEdgeInsetsPadding) DebugPaint(context *PaintingContext, offset ui.Offset) {
	p.RenderSliver.DebugPaint(context, offset)
	if !foundation.DebugMode || !p.DebugOptions().PaintSizeEnabled {
		return
	}
	outer := offset.And(p.GetAbsoluteSize())
	inner := foundation.None[ui.Rect]()
	if child := p.Child(); !isNil(child) {
		inner = foundation.Some((offset.Add(SliverPhysicalParentDataOf(child).PaintOffset)).And(child.GetAbsoluteSize()))
	}
	DebugPaintPadding(context.Canvas(), outer, inner, 0)
}

// RenderSliverPadding insets a sliver child by the given padding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding
type RenderSliverPadding struct {
	RenderSliverEdgeInsetsPadding
	padding         painting.EdgeInsetsGeometry
	textDirection   ui.TextDirection
	resolvedPadding foundation.Opt[painting.EdgeInsets]
}

// RenderSliverPaddingOpts are the optional parameters of NewRenderSliverPadding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.new
type RenderSliverPaddingOpts struct {
	// TextDirection resolves a directional padding.
	TextDirection ui.TextDirection
}

// NewRenderSliverPadding creates the sliver; the child may be nil. A negative
// padding panics with PaddingNegative in debug builds.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.new
func NewRenderSliverPadding(padding painting.EdgeInsetsGeometry, child Sliver, opts ...RenderSliverPaddingOpts) *RenderSliverPadding {
	assertPaddingNonNegative(padding)
	r := &RenderSliverPadding{padding: padding, textDirection: firstOpts(opts).TextDirection}
	InitRenderSliverEdgeInsetsPadding(&r.RenderSliverEdgeInsetsPadding, r)
	if !isNil(child) {
		r.SetChild(child)
	}
	return r
}

// ResolvedPadding is the padding resolved with the text direction; none until the
// first layout.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.resolvedPadding
func (r *RenderSliverPadding) ResolvedPadding() foundation.Opt[painting.EdgeInsets] {
	return r.resolvedPadding
}

func (r *RenderSliverPadding) resolve() {
	if r.resolvedPadding.IsSome() {
		return
	}
	r.resolvedPadding = foundation.Some(r.padding.Resolve(r.textDirection))
	assertPaddingNonNegative(r.resolvedPadding.Value())
}

func (r *RenderSliverPadding) markNeedsResolution() {
	r.resolvedPadding = foundation.None[painting.EdgeInsets]()
	r.MarkNeedsLayout()
}

// Padding is the amount to pad the child in each dimension.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.padding
func (r *RenderSliverPadding) Padding() painting.EdgeInsetsGeometry { return r.padding }

// SetPadding changes the padding and marks the sliver as needing layout.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.padding
func (r *RenderSliverPadding) SetPadding(value painting.EdgeInsetsGeometry) {
	assertPaddingNonNegative(value)
	if r.padding.Equal(value) {
		return
	}
	r.padding = value
	r.markNeedsResolution()
}

// TextDirection resolves a directional padding.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.textDirection
func (r *RenderSliverPadding) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and marks the sliver as needing layout.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.textDirection
func (r *RenderSliverPadding) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.markNeedsResolution()
}

// PerformLayout resolves the padding, then lays out.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.performLayout
func (r *RenderSliverPadding) PerformLayout() {
	r.resolve()
	r.RenderSliverEdgeInsetsPadding.PerformLayout()
}

// DebugFillProperties adds the padding and the text direction.
//
// Flutter: rendering/sliver_padding.dart RenderSliverPadding.debugFillProperties
func (r *RenderSliverPadding) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderSliverEdgeInsetsPadding.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("padding", r.padding))
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.textDirection), foundation.EnumPropertyOpts{DefaultValue: ui.TextDirectionUnspecified}))
}
