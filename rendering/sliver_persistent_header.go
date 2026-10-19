package rendering

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// trimEdges are the edges that persistentHeaderTrim cuts a rectangle to; an edge
// that is nil is left as it is.
type trimEdges struct {
	top, right, bottom, left *float64
}

func trimEdge(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

// persistentHeaderTrim is the original rectangle cut to the given edges; none
// stays none.
//
// Flutter: rendering/sliver_persistent_header.dart _trim
func persistentHeaderTrim(original foundation.Opt[ui.Rect], e trimEdges) foundation.Opt[ui.Rect] {
	r, ok := original.Get()
	if !ok {
		return original
	}
	return foundation.Some(r.Intersect(ui.NewRectFromLTRB(
		trimEdge(e.left, math.Inf(-1)), trimEdge(e.top, math.Inf(-1)), trimEdge(e.right, math.Inf(1)), trimEdge(e.bottom, math.Inf(1)))))
}

// OverScrollHeaderStretchConfiguration is the configuration of a header that
// stretches when it is overscrolled.
//
// Deviation: OnStretchTrigger is a plain function, not an AsyncCallback that
// returns a Future (SPEC §12.10).
//
// Flutter: rendering/sliver_persistent_header.dart OverScrollHeaderStretchConfiguration
type OverScrollHeaderStretchConfiguration struct {
	// StretchTriggerOffset is the overscroll distance, in logical pixels, at
	// which OnStretchTrigger is called; the default is 100.
	//
	// Flutter: rendering/sliver_persistent_header.dart OverScrollHeaderStretchConfiguration.stretchTriggerOffset
	StretchTriggerOffset float64
	// OnStretchTrigger is called when the overscroll reaches the trigger offset;
	// nil for none.
	//
	// Flutter: rendering/sliver_persistent_header.dart OverScrollHeaderStretchConfiguration.onStretchTrigger
	OnStretchTrigger func()
}

// NewOverScrollHeaderStretchConfiguration creates the configuration with the
// default trigger offset of 100 and no callback.
//
// Flutter: rendering/sliver_persistent_header.dart OverScrollHeaderStretchConfiguration.new
func NewOverScrollHeaderStretchConfiguration() *OverScrollHeaderStretchConfiguration {
	return &OverScrollHeaderStretchConfiguration{StretchTriggerOffset: 100}
}

// PersistentHeaderShowOnScreenConfiguration is how a header reacts to a request
// to show a descendant of it on screen: the range of extents it may take.
//
// Flutter: rendering/sliver_persistent_header.dart PersistentHeaderShowOnScreenConfiguration
type PersistentHeaderShowOnScreenConfiguration struct {
	// MinShowOnScreenExtent is the smallest extent the header may take; the
	// default is minus infinity.
	//
	// Flutter: rendering/sliver_persistent_header.dart PersistentHeaderShowOnScreenConfiguration.minShowOnScreenExtent
	MinShowOnScreenExtent float64
	// MaxShowOnScreenExtent is the largest extent the header may take; the
	// default is infinity.
	//
	// Flutter: rendering/sliver_persistent_header.dart PersistentHeaderShowOnScreenConfiguration.maxShowOnScreenExtent
	MaxShowOnScreenExtent float64
}

// PersistentHeaderShowOnScreenConfigurationOpts are the optional parameters of
// NewPersistentHeaderShowOnScreenConfiguration.
//
// Flutter: rendering/sliver_persistent_header.dart PersistentHeaderShowOnScreenConfiguration.new
type PersistentHeaderShowOnScreenConfigurationOpts struct {
	// MinShowOnScreenExtent defaults to minus infinity.
	MinShowOnScreenExtent foundation.Opt[float64]
	// MaxShowOnScreenExtent defaults to infinity.
	MaxShowOnScreenExtent foundation.Opt[float64]
}

// NewPersistentHeaderShowOnScreenConfiguration creates the configuration. In
// debug builds a minimum above the maximum panics with the tag
// PersistentHeaderShowOnScreenInvalid.
//
// Flutter: rendering/sliver_persistent_header.dart PersistentHeaderShowOnScreenConfiguration.new
func NewPersistentHeaderShowOnScreenConfiguration(opts ...PersistentHeaderShowOnScreenConfigurationOpts) *PersistentHeaderShowOnScreenConfiguration {
	o := firstOpts(opts)
	c := &PersistentHeaderShowOnScreenConfiguration{
		MinShowOnScreenExtent: o.MinShowOnScreenExtent.OrElse(math.Inf(-1)),
		MaxShowOnScreenExtent: o.MaxShowOnScreenExtent.OrElse(math.Inf(1)),
	}
	if foundation.DebugMode && !(c.MinShowOnScreenExtent <= c.MaxShowOnScreenExtent) {
		panic(errors.New("PersistentHeaderShowOnScreenInvalid", "the minimum show on screen extent must not be above the maximum",
			"min", c.MinShowOnScreenExtent, "max", c.MaxShowOnScreenExtent))
	}
	return c
}

// persistentHeaderHooks are the methods that the concrete headers supply to
// RenderSliverPersistentHeader.
type persistentHeaderHooks interface {
	MaxExtent() float64
	MinExtent() float64
}

// RenderSliverPersistentHeader is the base of the slivers that have a header
// box child whose extent changes with the scroll offset. The concrete type embeds
// one of the variants, and implements MaxExtent, MinExtent and, to build the
// child lazily, UpdateChild.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader
type RenderSliverPersistentHeader struct {
	RenderSliver
	RenderObjectWithChild[Box]

	stretchConfiguration *OverScrollHeaderStretchConfiguration
	lastStretchOffset    float64
	needsUpdateChild     bool
	lastShrinkOffset     float64
	lastOverlapsContent  bool
}

// InitRenderSliverPersistentHeader sets the outer object, which must have MaxExtent
// and MinExtent methods, the child (nil for none) and the stretch configuration
// (nil for none).
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.new
func InitRenderSliverPersistentHeader(h *RenderSliverPersistentHeader, self Sliver, child Box, stretchConfiguration *OverScrollHeaderStretchConfiguration) {
	InitRenderSliver(&h.RenderSliver, self)
	InitRenderObjectWithChild(&h.RenderObjectWithChild, self)
	h.stretchConfiguration = stretchConfiguration
	h.needsUpdateChild = true
	if !isNil(child) {
		h.SetChild(child)
	}
}

func (h *RenderSliverPersistentHeader) hooks() persistentHeaderHooks {
	return h.self.(persistentHeaderHooks)
}

// StretchConfiguration is how the header stretches when it is overscrolled; nil
// if it does not.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.stretchConfiguration
func (h *RenderSliverPersistentHeader) StretchConfiguration() *OverScrollHeaderStretchConfiguration {
	return h.stretchConfiguration
}

// SetStretchConfiguration changes the stretch configuration.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.stretchConfiguration
func (h *RenderSliverPersistentHeader) SetStretchConfiguration(value *OverScrollHeaderStretchConfiguration) {
	h.stretchConfiguration = value
}

// ChildExtent is the extent of the child along the main axis; zero without a
// child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.childExtent
func (h *RenderSliverPersistentHeader) ChildExtent() float64 {
	child := h.Child()
	if isNil(child) {
		return 0
	}
	if h.SliverConstraints().Axis() == painting.AxisVertical {
		return child.Size().Height
	}
	return child.Size().Width
}

// LastShrinkOffset is the shrink offset that the child was last updated for.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.lastShrinkOffset
func (h *RenderSliverPersistentHeader) LastShrinkOffset() float64 { return h.lastShrinkOffset }

// LastOverlapsContent is whether the child overlapped the content when it was
// last updated.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.lastOverlapsContent
func (h *RenderSliverPersistentHeader) LastOverlapsContent() bool { return h.lastOverlapsContent }

// UpdateChild updates the child for the shrink offset: the default does nothing.
// A concrete header that builds its child lazily defines its own.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.updateChild
func (h *RenderSliverPersistentHeader) UpdateChild(shrinkOffset float64, overlapsContent bool) {}

// MarkNeedsLayout also asks for the child to be updated.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.markNeedsLayout
func (h *RenderSliverPersistentHeader) MarkNeedsLayout() {
	h.needsUpdateChild = true
	h.RenderSliver.MarkNeedsLayout()
}

// LayoutChild updates the child (if the shrink offset or the overlap changed, or
// the sliver needs layout) and lays it out, between the minimum extent and the
// maximum minus the shrink offset. In debug builds a minimum above the maximum
// panics with the tag SliverPersistentHeaderExtentsInvalid.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.layoutChild
func (h *RenderSliverPersistentHeader) LayoutChild(scrollOffset, maxExtent float64, overlapsContent ...bool) {
	overlaps := len(overlapsContent) > 0 && overlapsContent[0]
	shrinkOffset := math.Min(scrollOffset, maxExtent)
	if h.needsUpdateChild || h.lastShrinkOffset != shrinkOffset || h.lastOverlapsContent != overlaps {
		h.InvokeLayoutCallback(func(Constraints) {
			h.self.(interface{ UpdateChild(float64, bool) }).UpdateChild(shrinkOffset, overlaps)
		})
		h.lastShrinkOffset = shrinkOffset
		h.lastOverlapsContent = overlaps
		h.needsUpdateChild = false
	}
	minExtent := h.hooks().MinExtent()
	if foundation.DebugMode && !(minExtent <= maxExtent) {
		panic(errors.New("SliverPersistentHeaderExtentsInvalid", "The maxExtent for this "+foundation.RuntimeTypeName(h.self)+" is less than its minExtent.",
			"maxExtent", maxExtent, "minExtent", minExtent))
	}
	c := h.SliverConstraints()
	stretchOffset := 0.0
	if h.stretchConfiguration != nil && c.ScrollOffset == 0 {
		stretchOffset += math.Abs(c.Overlap)
	}

	if child := h.Child(); !isNil(child) {
		child.Layout(c.AsBoxConstraints(AsBoxConstraintsOpts{
			MaxExtent: foundation.Some(math.Max(minExtent, maxExtent-shrinkOffset) + stretchOffset),
		}), LayoutOpts{ParentUsesSize: true})
	}

	if sc := h.stretchConfiguration; sc != nil && sc.OnStretchTrigger != nil &&
		stretchOffset >= sc.StretchTriggerOffset && h.lastStretchOffset <= sc.StretchTriggerOffset {
		sc.OnStretchTrigger()
	}
	h.lastStretchOffset = stretchOffset
}

// HitTestChildren hit tests the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.hitTestChildren
func (h *RenderSliverPersistentHeader) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	if child := h.Child(); !isNil(child) {
		return SliverHitTestBoxChild(h.debugSelf(), NewBoxHitTestResultWrap(result.HitTestResult), child, mainAxisPosition, crossAxisPosition)
	}
	return false
}

// ApplyPaintTransform translates by the position of the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.applyPaintTransform
func (h *RenderSliverPersistentHeader) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverApplyPaintTransformForBoxChild(h.debugSelf(), child.(Box), transform)
}

// Paint paints the child at its position, if the sliver is visible.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.paint
func (h *RenderSliverPersistentHeader) Paint(context *PaintingContext, offset ui.Offset) {
	child := h.Child()
	if isNil(child) || !h.Geometry().Visible {
		return
	}
	c, g := h.SliverConstraints(), h.Geometry()
	position := h.debugSelf().ChildMainAxisPosition(child)
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		offset = offset.Add(ui.Offset{Dy: g.PaintExtent - position - h.ChildExtent()})
	case painting.AxisDirectionLeft:
		offset = offset.Add(ui.Offset{Dx: g.PaintExtent - position - h.ChildExtent()})
	case painting.AxisDirectionRight:
		offset = offset.Add(ui.Offset{Dx: position})
	default:
		offset = offset.Add(ui.Offset{Dy: position})
	}
	context.PaintChild(child, offset)
}

// DescribeSemanticsConfiguration excludes the header from scrolling.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.describeSemanticsConfiguration
func (h *RenderSliverPersistentHeader) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	h.RenderSliver.DescribeSemanticsConfiguration(config)
	config.AddTagForChildren(RenderViewportExcludeFromScrolling)
}

// DebugFillProperties adds the maximum extent and the position of the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPersistentHeader.debugFillProperties
func (h *RenderSliverPersistentHeader) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	h.RenderSliver.DebugFillProperties(b)
	b.Add(foundation.NewDoublePropertyLazy("maxExtent", func() foundation.Opt[float64] { return foundation.Some(h.hooks().MaxExtent()) }))
	b.Add(foundation.NewDoublePropertyLazy("child position", func() foundation.Opt[float64] {
		return foundation.Some(h.debugSelf().ChildMainAxisPosition(h.Child()))
	}))
}

// RenderSliverScrollingPersistentHeader is a header that scrolls off the screen
// with the content.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverScrollingPersistentHeader
type RenderSliverScrollingPersistentHeader struct {
	RenderSliverPersistentHeader
	childPosition foundation.Opt[float64]
}

// InitRenderSliverScrollingPersistentHeader sets the outer object, the child
// (nil for none) and the stretch configuration (nil for none).
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverScrollingPersistentHeader.new
func InitRenderSliverScrollingPersistentHeader(h *RenderSliverScrollingPersistentHeader, self Sliver, child Box, stretchConfiguration *OverScrollHeaderStretchConfiguration) {
	InitRenderSliverPersistentHeader(&h.RenderSliverPersistentHeader, self, child, stretchConfiguration)
}

// UpdateGeometry sets the geometry from the maximum extent and returns the
// position of the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverScrollingPersistentHeader.updateGeometry
func (h *RenderSliverScrollingPersistentHeader) UpdateGeometry() float64 {
	c := h.SliverConstraints()
	stretchOffset := 0.0
	if h.stretchConfiguration != nil {
		stretchOffset += math.Abs(c.Overlap)
	}
	maxExtent := h.hooks().MaxExtent()
	paintExtent := maxExtent - c.ScrollOffset
	cacheExtent := h.CalculateCacheOffset(c, 0, maxExtent)

	h.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		CacheExtent:       foundation.Some(cacheExtent),
		ScrollExtent:      maxExtent,
		PaintOrigin:       math.Min(c.Overlap, 0),
		PaintExtent:       ui.ClampDouble(paintExtent, 0, c.RemainingPaintExtent),
		MaxPaintExtent:    maxExtent + stretchOffset,
		HasVisualOverflow: true, // Conservatively say we do have overflow to avoid complexity.
	}))
	if stretchOffset > 0 {
		return 0
	}
	return math.Min(0, paintExtent-h.ChildExtent())
}

// PerformLayout lays the header out and sets the geometry.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverScrollingPersistentHeader.performLayout
func (h *RenderSliverScrollingPersistentHeader) PerformLayout() {
	h.LayoutChild(h.SliverConstraints().ScrollOffset, h.hooks().MaxExtent())
	h.childPosition = foundation.Some(h.UpdateGeometry())
}

// ChildMainAxisPosition is the position that the layout computed.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverScrollingPersistentHeader.childMainAxisPosition
func (h *RenderSliverScrollingPersistentHeader) ChildMainAxisPosition(RenderObject) float64 {
	return h.childPosition.Value()
}

// RenderSliverPinnedPersistentHeader is a header that is pinned to the start of
// the viewport, its extent shrinking as the content scrolls.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader
type RenderSliverPinnedPersistentHeader struct {
	RenderSliverPersistentHeader
	showOnScreenConfiguration *PersistentHeaderShowOnScreenConfiguration
}

// InitRenderSliverPinnedPersistentHeader sets the outer object, the child (nil
// for none), the stretch configuration (nil for none) and the show on screen
// configuration.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.new
func InitRenderSliverPinnedPersistentHeader(h *RenderSliverPinnedPersistentHeader, self Sliver, child Box, stretchConfiguration *OverScrollHeaderStretchConfiguration, showOnScreenConfiguration *PersistentHeaderShowOnScreenConfiguration) {
	InitRenderSliverPersistentHeader(&h.RenderSliverPersistentHeader, self, child, stretchConfiguration)
	h.showOnScreenConfiguration = showOnScreenConfiguration
}

// ShowOnScreenConfiguration is how the header reacts to a request to show a
// descendant on screen.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.showOnScreenConfiguration
func (h *RenderSliverPinnedPersistentHeader) ShowOnScreenConfiguration() *PersistentHeaderShowOnScreenConfiguration {
	return h.showOnScreenConfiguration
}

// SetShowOnScreenConfiguration changes the configuration.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.showOnScreenConfiguration
func (h *RenderSliverPinnedPersistentHeader) SetShowOnScreenConfiguration(value *PersistentHeaderShowOnScreenConfiguration) {
	h.showOnScreenConfiguration = value
}

// PerformLayout lays the header out and sets the geometry.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.performLayout
func (h *RenderSliverPinnedPersistentHeader) PerformLayout() {
	c := h.SliverConstraints()
	maxExtent := h.hooks().MaxExtent()
	overlapsContent := c.Overlap > 0
	h.LayoutChild(c.ScrollOffset, maxExtent, overlapsContent)
	effectiveRemainingPaintExtent := math.Max(0, c.RemainingPaintExtent-c.Overlap)
	layoutExtent := ui.ClampDouble(maxExtent-c.ScrollOffset, 0, effectiveRemainingPaintExtent)
	stretchOffset := 0.0
	if h.stretchConfiguration != nil {
		stretchOffset = math.Abs(c.Overlap)
	}
	cacheExtent := layoutExtent
	if layoutExtent > 0 {
		cacheExtent = -c.CacheOrigin + layoutExtent
	}
	h.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:               maxExtent,
		PaintOrigin:                c.Overlap,
		PaintExtent:                math.Min(h.ChildExtent(), effectiveRemainingPaintExtent),
		LayoutExtent:               foundation.Some(layoutExtent),
		MaxPaintExtent:             maxExtent + stretchOffset,
		MaxScrollObstructionExtent: h.hooks().MinExtent(),
		CacheExtent:                foundation.Some(cacheExtent),
		HasVisualOverflow:          true, // Conservatively say we do have overflow to avoid complexity.
	}))
}

// ChildMainAxisPosition is zero.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.childMainAxisPosition
func (h *RenderSliverPinnedPersistentHeader) ChildMainAxisPosition(RenderObject) float64 { return 0 }

// ShowOnScreen asks the parent to show the part of the descendant that is inside
// the header.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverPinnedPersistentHeader.showOnScreen
func (h *RenderSliverPinnedPersistentHeader) ShowOnScreen(opts ShowOnScreenOpts) {
	localBounds := opts.Rect
	if !isNil(opts.Descendant) {
		r := opts.Rect.OrElse(opts.Descendant.PaintBounds())
		localBounds = foundation.Some(opts.Descendant.GetTransformTo(h.self).TransformRect(r))
	}
	c := h.SliverConstraints()
	zero, childExtent := 0.0, h.ChildExtent()
	var newRect foundation.Opt[ui.Rect]
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		newRect = persistentHeaderTrim(localBounds, trimEdges{bottom: &childExtent})
	case painting.AxisDirectionLeft:
		newRect = persistentHeaderTrim(localBounds, trimEdges{right: &childExtent})
	case painting.AxisDirectionRight:
		newRect = persistentHeaderTrim(localBounds, trimEdges{left: &zero})
	default:
		newRect = persistentHeaderTrim(localBounds, trimEdges{top: &zero})
	}
	h.RenderSliver.ShowOnScreen(ShowOnScreenOpts{Descendant: h.self, Rect: newRect, Duration: opts.Duration, Curve: opts.Curve})
}

// FloatingHeaderSnapConfiguration is how a floating header snaps into or out of
// view when the user stops scrolling.
//
// Flutter: rendering/sliver_persistent_header.dart FloatingHeaderSnapConfiguration
type FloatingHeaderSnapConfiguration struct {
	// Curve is the snap animation curve; the default is Curves.ease.
	//
	// Flutter: rendering/sliver_persistent_header.dart FloatingHeaderSnapConfiguration.curve
	Curve animation.Curve
	// Duration is the snap animation duration; the default is 300ms.
	//
	// Flutter: rendering/sliver_persistent_header.dart FloatingHeaderSnapConfiguration.duration
	Duration time.Duration
}

// FloatingHeaderSnapConfigurationOpts are the optional parameters of
// NewFloatingHeaderSnapConfiguration.
//
// Flutter: rendering/sliver_persistent_header.dart FloatingHeaderSnapConfiguration.new
type FloatingHeaderSnapConfigurationOpts struct {
	// Curve defaults to animation.Curves.Ease.
	Curve animation.Curve
	// Duration defaults to 300ms.
	Duration foundation.Opt[time.Duration]
}

// NewFloatingHeaderSnapConfiguration creates the configuration.
//
// Flutter: rendering/sliver_persistent_header.dart FloatingHeaderSnapConfiguration.new
func NewFloatingHeaderSnapConfiguration(opts ...FloatingHeaderSnapConfigurationOpts) *FloatingHeaderSnapConfiguration {
	o := firstOpts(opts)
	c := &FloatingHeaderSnapConfiguration{Curve: o.Curve, Duration: o.Duration.OrElse(300 * time.Millisecond)}
	if c.Curve == nil {
		c.Curve = animation.Curves.Ease
	}
	return c
}

// RenderSliverFloatingPersistentHeader is a header that reappears as soon as the
// user scrolls back, and optionally snaps into or out of view.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader
type RenderSliverFloatingPersistentHeader struct {
	RenderSliverPersistentHeader

	controller                 *animation.AnimationController
	animation                  animation.Animation[float64]
	lastActualScrollOffset     foundation.Opt[float64]
	effectiveScrollOffset      foundation.Opt[float64]
	lastStartedScrollDirection ScrollDirection
	childPosition              foundation.Opt[float64]
	vsync                      scheduler.TickerProvider
	snapConfiguration          *FloatingHeaderSnapConfiguration
	showOnScreenConfiguration  *PersistentHeaderShowOnScreenConfiguration
}

// InitRenderSliverFloatingPersistentHeader sets the outer object (whose
// UpdateGeometry is called, so that a variant can replace it), the child (nil for none), the
// vsync (nil for none), and the snap, stretch and show on screen configurations
// (nil for none).
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.new
func InitRenderSliverFloatingPersistentHeader(h *RenderSliverFloatingPersistentHeader, self Sliver, child Box, vsync scheduler.TickerProvider, snapConfiguration *FloatingHeaderSnapConfiguration, stretchConfiguration *OverScrollHeaderStretchConfiguration, showOnScreenConfiguration *PersistentHeaderShowOnScreenConfiguration) {
	InitRenderSliverPersistentHeader(&h.RenderSliverPersistentHeader, self, child, stretchConfiguration)
	h.vsync = vsync
	h.snapConfiguration = snapConfiguration
	h.showOnScreenConfiguration = showOnScreenConfiguration
}

// Detach disposes the snap animation controller; it is created again when
// needed.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.detach
func (h *RenderSliverFloatingPersistentHeader) Detach() {
	if h.controller != nil {
		h.controller.Dispose()
		h.controller = nil // lazily recreated if we're reattached.
	}
	h.RenderSliverPersistentHeader.Detach()
}

// Vsync is the provider of the tickers of the snap animations.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.vsync
func (h *RenderSliverFloatingPersistentHeader) Vsync() scheduler.TickerProvider { return h.vsync }

// SetVsync changes the ticker provider.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.vsync
func (h *RenderSliverFloatingPersistentHeader) SetVsync(value scheduler.TickerProvider) {
	if value == h.vsync {
		return
	}
	h.vsync = value
	if value == nil {
		if h.controller != nil {
			h.controller.Dispose()
		}
		h.controller = nil
	} else if h.controller != nil {
		h.controller.Resync(value)
	}
}

// SnapConfiguration is how the header snaps; nil if it does not.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.snapConfiguration
func (h *RenderSliverFloatingPersistentHeader) SnapConfiguration() *FloatingHeaderSnapConfiguration {
	return h.snapConfiguration
}

// SetSnapConfiguration changes the snap configuration.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.snapConfiguration
func (h *RenderSliverFloatingPersistentHeader) SetSnapConfiguration(value *FloatingHeaderSnapConfiguration) {
	h.snapConfiguration = value
}

// ShowOnScreenConfiguration is how the header reacts to a request to show a
// descendant on screen; nil for the default.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.showOnScreenConfiguration
func (h *RenderSliverFloatingPersistentHeader) ShowOnScreenConfiguration() *PersistentHeaderShowOnScreenConfiguration {
	return h.showOnScreenConfiguration
}

// SetShowOnScreenConfiguration changes the configuration.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.showOnScreenConfiguration
func (h *RenderSliverFloatingPersistentHeader) SetShowOnScreenConfiguration(value *PersistentHeaderShowOnScreenConfiguration) {
	h.showOnScreenConfiguration = value
}

// UpdateGeometry sets the geometry and returns the position of the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.updateGeometry
func (h *RenderSliverFloatingPersistentHeader) UpdateGeometry() float64 {
	c := h.SliverConstraints()
	stretchOffset := 0.0
	if h.stretchConfiguration != nil {
		stretchOffset += math.Abs(c.Overlap)
	}
	maxExtent := h.hooks().MaxExtent()
	paintExtent := maxExtent - h.effectiveScrollOffset.Value()
	layoutExtent := maxExtent - c.ScrollOffset
	h.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      maxExtent,
		PaintOrigin:       math.Min(c.Overlap, 0),
		PaintExtent:       ui.ClampDouble(paintExtent, 0, c.RemainingPaintExtent),
		LayoutExtent:      foundation.Some(ui.ClampDouble(layoutExtent, 0, c.RemainingPaintExtent)),
		MaxPaintExtent:    maxExtent + stretchOffset,
		HasVisualOverflow: true, // Conservatively say we do have overflow to avoid complexity.
	}))
	if stretchOffset > 0 {
		return 0
	}
	return math.Min(0, paintExtent-h.ChildExtent())
}

func (h *RenderSliverFloatingPersistentHeader) updateAnimation(duration time.Duration, endValue float64, curve animation.Curve) {
	if foundation.DebugMode && h.vsync == nil {
		panic(errors.New("SliverFloatingHeaderVsyncMissing", "vsync must not be null if the floating header changes size animatedly.",
			"object", foundation.DescribeIdentity(h.self)))
	}
	if h.controller == nil {
		h.controller = animation.NewAnimationController(h.vsync, animation.AnimationControllerOpts{Duration: foundation.Some(duration)})
		h.controller.AddListener(func() {
			if h.effectiveScrollOffset == foundation.Some(h.animation.Value()) {
				return
			}
			h.effectiveScrollOffset = foundation.Some(h.animation.Value())
			h.MarkNeedsLayout()
		})
	}
	h.animation = animation.Drive[float64](h.controller, animation.NewTween(h.effectiveScrollOffset.Value(), endValue).Chain(animation.NewCurveTween(curve)))
}

// UpdateScrollStartDirection records the direction in which the user started to
// scroll.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.updateScrollStartDirection
func (h *RenderSliverFloatingPersistentHeader) UpdateScrollStartDirection(direction ScrollDirection) {
	h.lastStartedScrollDirection = direction
}

// MaybeStartSnapAnimation starts the animation that snaps the header into or out
// of view, if it is configured to and is partly visible.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.maybeStartSnapAnimation
func (h *RenderSliverFloatingPersistentHeader) MaybeStartSnapAnimation(direction ScrollDirection) {
	snap := h.snapConfiguration
	if snap == nil {
		return
	}
	maxExtent := h.hooks().MaxExtent()
	if direction == ScrollDirectionForward && h.effectiveScrollOffset.Value() <= 0 {
		return
	}
	if direction == ScrollDirectionReverse && h.effectiveScrollOffset.Value() >= maxExtent {
		return
	}
	end := maxExtent
	if direction == ScrollDirectionForward {
		end = 0
	}
	h.updateAnimation(snap.Duration, end, snap.Curve)
	if h.controller != nil {
		h.controller.Forward(0)
	}
}

// MaybeStopSnapAnimation stops the snap animation.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.maybeStopSnapAnimation
func (h *RenderSliverFloatingPersistentHeader) MaybeStopSnapAnimation(ScrollDirection) {
	if h.controller != nil {
		h.controller.Stop()
	}
}

// PerformLayout lays the header out: it keeps the scroll offset at which the
// header is shown, which follows the scroll offset of the sliver when the user
// scrolls forward.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.performLayout
func (h *RenderSliverFloatingPersistentHeader) PerformLayout() {
	c := h.SliverConstraints()
	maxExtent := h.hooks().MaxExtent()
	if last, ok := h.lastActualScrollOffset.Get(); ok && // We've laid out at least once to get an initial position, and either
		(c.ScrollOffset < last || // we are scrolling back, so should reveal, or
			h.effectiveScrollOffset.Value() < maxExtent) { // some part of it is visible, so should shrink or reveal as appropriate.
		delta := last - c.ScrollOffset

		allowFloatingExpansion := c.UserScrollDirection == ScrollDirectionForward ||
			h.lastStartedScrollDirection == ScrollDirectionForward
		effective := h.effectiveScrollOffset.Value()
		if allowFloatingExpansion {
			if effective > maxExtent { // We're scrolled off-screen, but should reveal, so pretend we're just at the limit.
				effective = maxExtent
			}
		} else {
			if delta > 0 { // If we are trying to expand when allowFloatingExpansion is false, disallow it.
				delta = 0
			}
		}
		h.effectiveScrollOffset = foundation.Some(ui.ClampDouble(effective-delta, 0, c.ScrollOffset))
	} else {
		h.effectiveScrollOffset = foundation.Some(c.ScrollOffset)
	}
	overlapsContent := h.effectiveScrollOffset.Value() < c.ScrollOffset

	h.LayoutChild(h.effectiveScrollOffset.Value(), maxExtent, overlapsContent)
	h.childPosition = foundation.Some(h.self.(interface{ UpdateGeometry() float64 }).UpdateGeometry())
	h.lastActualScrollOffset = foundation.Some(c.ScrollOffset)
}

// ShowOnScreen shows the descendant by growing the header to the extent that its
// configuration allows, then asks the parent to show the child.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.showOnScreen
func (h *RenderSliverFloatingPersistentHeader) ShowOnScreen(opts ShowOnScreenOpts) {
	showOnScreen := h.showOnScreenConfiguration
	if showOnScreen == nil {
		h.RenderSliver.ShowOnScreen(opts)
		return
	}

	child := h.Child()
	if foundation.DebugMode && isNil(child) && !isNil(opts.Descendant) {
		panic(errors.New("SliverFloatingHeaderChildMissing", "a floating header without a child was asked to show a descendant",
			"object", foundation.DescribeIdentity(h.self)))
	}
	childBounds := opts.Rect
	if !isNil(opts.Descendant) {
		r := opts.Rect.OrElse(opts.Descendant.PaintBounds())
		childBounds = foundation.Some(opts.Descendant.GetTransformTo(child).TransformRect(r))
	}

	c := h.SliverConstraints()
	childExtent := h.ChildExtent()
	var targetExtent float64
	var targetRect foundation.Opt[ui.Rect]
	zero := 0.0
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		targetExtent = childExtent - childBounds.OrElse(ui.Rect{}).Top
		targetRect = persistentHeaderTrim(childBounds, trimEdges{bottom: &childExtent})
	case painting.AxisDirectionRight:
		targetExtent = childExtent
		if b, ok := childBounds.Get(); ok {
			targetExtent = b.Right
		}
		targetRect = persistentHeaderTrim(childBounds, trimEdges{left: &zero})
	case painting.AxisDirectionDown:
		targetExtent = childExtent
		if b, ok := childBounds.Get(); ok {
			targetExtent = b.Bottom
		}
		targetRect = persistentHeaderTrim(childBounds, trimEdges{top: &zero})
	default: // left
		targetExtent = childExtent - childBounds.OrElse(ui.Rect{}).Left
		targetRect = persistentHeaderTrim(childBounds, trimEdges{right: &childExtent})
	}

	// When the target is far from the edge of the screen, it is not a problem to
	// let the header grow, so clamp to what the configuration allows, and to the
	// extents that the header can have.
	maxExtent := h.hooks().MaxExtent()
	effectiveMaxExtent := math.Max(childExtent, maxExtent)
	targetExtent = ui.ClampDouble(ui.ClampDouble(targetExtent, showOnScreen.MinShowOnScreenExtent, showOnScreen.MaxShowOnScreenExtent), childExtent, effectiveMaxExtent)

	if targetExtent > childExtent && (h.controller == nil || h.controller.Status() != animation.AnimationStatusForward) {
		targetScrollOffset := maxExtent - targetExtent
		h.updateAnimation(opts.Duration, targetScrollOffset, opts.Curve)
		if h.controller != nil {
			h.controller.Forward(0)
		}
	}

	descendant := RenderObject(child)
	if isNil(opts.Descendant) {
		descendant = h.self
	}
	h.RenderSliver.ShowOnScreen(ShowOnScreenOpts{Descendant: descendant, Rect: targetRect, Duration: opts.Duration, Curve: opts.Curve})
}

// ChildMainAxisPosition is the position that the layout computed.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.childMainAxisPosition
func (h *RenderSliverFloatingPersistentHeader) ChildMainAxisPosition(RenderObject) float64 {
	return h.childPosition.OrElse(0)
}

// DebugFillProperties adds the effective scroll offset.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPersistentHeader.debugFillProperties
func (h *RenderSliverFloatingPersistentHeader) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	h.RenderSliverPersistentHeader.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("effective scroll offset", h.effectiveScrollOffset))
}

// InitRenderSliverFloatingPinnedPersistentHeader sets the outer object, the child
// (nil for none), the vsync (nil for none), and the snap, stretch and show on
// screen configurations (nil for none).
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPinnedPersistentHeader.new
func InitRenderSliverFloatingPinnedPersistentHeader(h *RenderSliverFloatingPinnedPersistentHeader, self Sliver, child Box, vsync scheduler.TickerProvider, snapConfiguration *FloatingHeaderSnapConfiguration, stretchConfiguration *OverScrollHeaderStretchConfiguration, showOnScreenConfiguration *PersistentHeaderShowOnScreenConfiguration) {
	InitRenderSliverFloatingPersistentHeader(&h.RenderSliverFloatingPersistentHeader, self, child, vsync, snapConfiguration, stretchConfiguration, showOnScreenConfiguration)
}

// RenderSliverFloatingPinnedPersistentHeader is a header that floats like
// RenderSliverFloatingPersistentHeader, but shrinks to its minimum extent
// instead of leaving the viewport.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPinnedPersistentHeader
type RenderSliverFloatingPinnedPersistentHeader struct {
	RenderSliverFloatingPersistentHeader
}

// UpdateGeometry sets the geometry, with the minimum extent as the least paint
// extent, and returns zero.
//
// Flutter: rendering/sliver_persistent_header.dart RenderSliverFloatingPinnedPersistentHeader.updateGeometry
func (h *RenderSliverFloatingPinnedPersistentHeader) UpdateGeometry() float64 {
	c := h.SliverConstraints()
	minExtent := h.hooks().MinExtent()
	minAllowedExtent := c.RemainingPaintExtent
	if c.RemainingPaintExtent > minExtent {
		minAllowedExtent = minExtent
	}
	maxExtent := h.hooks().MaxExtent()
	paintExtent := maxExtent - h.effectiveScrollOffset.Value()
	clampedPaintExtent := ui.ClampDouble(paintExtent, minAllowedExtent, c.RemainingPaintExtent)
	layoutExtent := maxExtent - c.ScrollOffset
	stretchOffset := 0.0
	if h.stretchConfiguration != nil {
		stretchOffset = math.Abs(c.Overlap)
	}
	h.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:               maxExtent,
		PaintOrigin:                math.Min(c.Overlap, 0),
		PaintExtent:                clampedPaintExtent,
		LayoutExtent:               foundation.Some(ui.ClampDouble(layoutExtent, 0, clampedPaintExtent)),
		MaxPaintExtent:             maxExtent + stretchOffset,
		MaxScrollObstructionExtent: minExtent,
		HasVisualOverflow:          true, // Conservatively say we do have overflow to avoid complexity.
	}))
	return 0
}
