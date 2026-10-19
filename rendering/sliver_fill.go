package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
)

// RenderSliverFillViewport is a sliver that contains multiple box children that
// each fill the viewport (or a fraction of it) in the main axis.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport
type RenderSliverFillViewport struct {
	RenderSliverFixedExtentBoxAdaptor
	viewportFraction       float64
	allowImplicitScrolling bool
}

// RenderSliverFillViewportOpts are the optional parameters of
// NewRenderSliverFillViewport.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.new
type RenderSliverFillViewportOpts struct {
	// ViewportFraction defaults to 1.
	ViewportFraction foundation.Opt[float64]
	// AllowImplicitScrolling defaults to true.
	AllowImplicitScrolling foundation.Opt[bool]
}

// NewRenderSliverFillViewport creates the sliver. In debug builds a fraction
// that is not positive panics with the tag SliverViewportFractionInvalid.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.new
func NewRenderSliverFillViewport(childManager RenderSliverBoxChildManager, opts ...RenderSliverFillViewportOpts) *RenderSliverFillViewport {
	o := firstOpts(opts)
	r := &RenderSliverFillViewport{
		viewportFraction:       o.ViewportFraction.OrElse(1),
		allowImplicitScrolling: o.AllowImplicitScrolling.OrElse(true),
	}
	assertViewportFraction(r.viewportFraction)
	InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, childManager)
	return r
}

func assertViewportFraction(v float64) {
	if foundation.DebugMode && !(v > 0) {
		panic(errors.New("SliverViewportFractionInvalid", "the viewport fraction must be positive", "viewportFraction", v))
	}
}

// ItemExtent is the main axis extent of the viewport times the fraction.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.itemExtent
func (r *RenderSliverFillViewport) ItemExtent() foundation.Opt[float64] {
	return foundation.Some(r.SliverConstraints().ViewportMainAxisExtent * r.viewportFraction)
}

// ItemExtentBuilder is nil.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.itemExtentBuilder
func (r *RenderSliverFillViewport) ItemExtentBuilder() ItemExtentBuilder { return nil }

// ViewportFraction is the fraction of the viewport that each child occupies.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.viewportFraction
func (r *RenderSliverFillViewport) ViewportFraction() float64 { return r.viewportFraction }

// SetViewportFraction changes the fraction and marks the sliver as needing layout.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.viewportFraction
func (r *RenderSliverFillViewport) SetViewportFraction(value float64) {
	if r.viewportFraction == value {
		return
	}
	r.viewportFraction = value
	r.MarkNeedsLayout()
}

// AllowImplicitScrolling is whether the semantics include the children that are
// not visible.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.allowImplicitScrolling
func (r *RenderSliverFillViewport) AllowImplicitScrolling() bool { return r.allowImplicitScrolling }

// SetAllowImplicitScrolling changes the flag and marks the sliver as needing a
// semantics update.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.allowImplicitScrolling
func (r *RenderSliverFillViewport) SetAllowImplicitScrolling(value bool) {
	if r.allowImplicitScrolling == value {
		return
	}
	r.allowImplicitScrolling = value
	r.MarkNeedsSemanticsUpdate()
}

// VisitChildrenForSemantics visits all the children if implicit scrolling is
// allowed, otherwise only those in the viewport.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillViewport.visitChildrenForSemantics
func (r *RenderSliverFillViewport) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.allowImplicitScrolling {
		r.RenderSliverFixedExtentBoxAdaptor.VisitChildrenForSemantics(visitor)
		return
	}
	c := r.SliverConstraints()
	visibleStart := c.ScrollOffset
	visibleEnd := visibleStart + c.ViewportMainAxisExtent
	itemExtent := r.ItemExtent().Value()
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		childStart := SliverMultiBoxAdaptorParentDataOf(child).LayoutOffset.Value()
		if childStart >= visibleEnd {
			break
		}
		if childStart+itemExtent > visibleStart {
			visitor(child)
		}
	}
}

func assertFillRemainingExtentFinite(extent float64) {
	if foundation.DebugMode && (math.IsInf(extent, 0) || math.IsNaN(extent)) {
		panic(errors.New("SliverFillRemainingExtentInfinite", "The calculated extent for the child of SliverFillRemaining is not finite. "+
			"This can happen if the child is a scrollable, in which case, the hasScrollBody property of SliverFillRemaining should not be set to false.",
			"extent", extent))
	}
}

func assertPaintedChildSize(s Sliver, size float64) {
	if foundation.DebugMode && (math.IsInf(size, 0) || math.IsNaN(size) || size < 0) {
		panic(errors.New("SliverPaintExtentInvalid", "the painted size of the child is not finite and not negative",
			"object", foundation.DescribeIdentity(s), "paintedChildSize", size))
	}
}

// RenderSliverFillRemainingWithScrollable is a sliver that contains a single box
// child that fills the remaining space in the viewport, and that is a scrollable
// itself.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingWithScrollable
type RenderSliverFillRemainingWithScrollable struct {
	RenderSliverSingleBoxAdapter
}

// NewRenderSliverFillRemainingWithScrollable creates the sliver; the child may
// be nil.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingWithScrollable.new
func NewRenderSliverFillRemainingWithScrollable(child Box) *RenderSliverFillRemainingWithScrollable {
	r := &RenderSliverFillRemainingWithScrollable{}
	InitRenderSliverSingleBoxAdapter(&r.RenderSliverSingleBoxAdapter, r, child)
	return r
}

// PerformLayout lays the child out to fill the remaining paint extent.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingWithScrollable.performLayout
func (r *RenderSliverFillRemainingWithScrollable) PerformLayout() {
	c := r.SliverConstraints()
	extent := c.RemainingPaintExtent - math.Min(c.Overlap, 0)

	cacheExtent := r.CalculateCacheOffset(c, 0, c.ViewportMainAxisExtent)
	if r.hasChild() {
		maxExtent := extent
		// If the extent is 0 (e.g. the sliver is outside the viewport) but the
		// cache extent is not, lay the child out in the cache extent so it is
		// built.
		if extent == 0 && cacheExtent > 0 {
			maxExtent = cacheExtent
		}
		r.Child().Layout(c.AsBoxConstraints(AsBoxConstraintsOpts{MinExtent: extent, MaxExtent: foundation.Some(maxExtent)}))
	}

	paintedChildSize := r.CalculatePaintOffset(c, 0, extent)
	assertPaintedChildSize(r.self.(Sliver), paintedChildSize)

	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      c.ViewportMainAxisExtent,
		PaintExtent:       paintedChildSize,
		MaxPaintExtent:    paintedChildSize,
		HasVisualOverflow: extent > c.RemainingPaintExtent || c.ScrollOffset > 0,
		CacheExtent:       foundation.Some(cacheExtent),
	}))
	if r.hasChild() {
		r.SetChildParentData(r.Child(), c, r.Geometry())
	}
}

func (a *RenderSliverSingleBoxAdapter) childMaxIntrinsicExtent(c SliverConstraints) float64 {
	if c.Axis() == painting.AxisHorizontal {
		return a.Child().GetMaxIntrinsicWidth(c.CrossAxisExtent)
	}
	return a.Child().GetMaxIntrinsicHeight(c.CrossAxisExtent)
}

// RenderSliverFillRemaining is a sliver that contains a single box child that
// fills the remaining space in the viewport, without a scrollable inside.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemaining
type RenderSliverFillRemaining struct {
	RenderSliverSingleBoxAdapter
}

// NewRenderSliverFillRemaining creates the sliver; the child may be nil.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemaining.new
func NewRenderSliverFillRemaining(child Box) *RenderSliverFillRemaining {
	r := &RenderSliverFillRemaining{}
	InitRenderSliverSingleBoxAdapter(&r.RenderSliverSingleBoxAdapter, r, child)
	return r
}

// PerformLayout lays the child out to fill the remaining space, or its intrinsic
// size if that is more. In debug builds an extent that is not finite panics with
// the tag SliverFillRemainingExtentInfinite.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemaining.performLayout
func (r *RenderSliverFillRemaining) PerformLayout() {
	c := r.SliverConstraints()
	// The remaining space in the viewportMainAxisExtent. Can be <= 0 if we have
	// already scrolled past the extent of the viewport.
	extent := c.ViewportMainAxisExtent - c.PrecedingScrollExtent

	if r.hasChild() {
		childExtent := r.childMaxIntrinsicExtent(c)
		// If the childExtent is greater than the computed extent, we want to use
		// that instead of potentially cutting off the child. This allows
		// sliverFillRemaining to handle the case where its child has more height
		// than the remaining space.
		extent = math.Max(extent, childExtent)
		r.Child().Layout(c.AsBoxConstraints(AsBoxConstraintsOpts{MinExtent: extent, MaxExtent: foundation.Some(extent)}))
	}

	assertFillRemainingExtentFinite(extent)
	paintedChildSize := r.CalculatePaintOffset(c, 0, extent)
	assertPaintedChildSize(r.self.(Sliver), paintedChildSize)

	cacheExtent := r.CalculateCacheOffset(c, 0, extent)
	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      extent,
		PaintExtent:       paintedChildSize,
		MaxPaintExtent:    paintedChildSize,
		HasVisualOverflow: extent > c.RemainingPaintExtent || c.ScrollOffset > 0,
		CacheExtent:       foundation.Some(cacheExtent),
	}))
	if r.hasChild() {
		r.SetChildParentData(r.Child(), c, r.Geometry())
	}
}

// RenderSliverFillRemainingAndOverscroll is a sliver that contains a single box
// child that fills the remaining space in the viewport, and also fills the
// overscroll.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingAndOverscroll
type RenderSliverFillRemainingAndOverscroll struct {
	RenderSliverSingleBoxAdapter
}

// NewRenderSliverFillRemainingAndOverscroll creates the sliver; the child may be
// nil.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingAndOverscroll.new
func NewRenderSliverFillRemainingAndOverscroll(child Box) *RenderSliverFillRemainingAndOverscroll {
	r := &RenderSliverFillRemainingAndOverscroll{}
	InitRenderSliverSingleBoxAdapter(&r.RenderSliverSingleBoxAdapter, r, child)
	return r
}

// PerformLayout lays the child out to fill the remaining space and the
// overscroll. In debug builds an extent that is not finite panics with the tag
// SliverFillRemainingExtentInfinite.
//
// Flutter: rendering/sliver_fill.dart RenderSliverFillRemainingAndOverscroll.performLayout
func (r *RenderSliverFillRemainingAndOverscroll) PerformLayout() {
	c := r.SliverConstraints()
	// The remaining space in the viewportMainAxisExtent. Can be <= 0 if we have
	// already scrolled past the extent of the viewport.
	extent := c.ViewportMainAxisExtent - c.PrecedingScrollExtent
	maxExtent := c.RemainingPaintExtent - math.Min(c.Overlap, 0)

	if r.hasChild() {
		childExtent := r.childMaxIntrinsicExtent(c)
		// If the childExtent is greater than the computed extent, we want to use
		// that instead of potentially cutting off the child.
		extent = math.Max(extent, childExtent)
		// The extent could be larger than the maxExtent due to a larger child
		// size or overscrolling at the top of the scrollable (rather than at the
		// end where this sliver is).
		maxExtent = math.Max(extent, maxExtent)
		r.Child().Layout(c.AsBoxConstraints(AsBoxConstraintsOpts{MinExtent: extent, MaxExtent: foundation.Some(maxExtent)}))
	}

	assertFillRemainingExtentFinite(extent)
	paintedChildSize := r.CalculatePaintOffset(c, 0, extent)
	assertPaintedChildSize(r.self.(Sliver), paintedChildSize)

	cacheExtent := r.CalculateCacheOffset(c, 0, extent)
	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      extent,
		PaintExtent:       math.Min(maxExtent, c.RemainingPaintExtent),
		MaxPaintExtent:    maxExtent,
		HasVisualOverflow: extent > c.RemainingPaintExtent || c.ScrollOffset > 0,
		CacheExtent:       foundation.Some(cacheExtent),
	}))
	if r.hasChild() {
		r.SetChildParentData(r.Child(), c, r.Geometry())
	}
}
