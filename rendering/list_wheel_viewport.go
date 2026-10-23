package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ListWheelChildManager creates and removes the children of a
// RenderListWheelViewport, lazily, while the viewport lays itself out.
//
// Flutter: rendering/list_wheel_viewport.dart ListWheelChildManager
type ListWheelChildManager interface {
	// ChildCount is the number of children, or none if the list is not bounded.
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelChildManager.childCount
	ChildCount() foundation.Opt[int]
	// ChildExistsAt is whether there is a child at the index.
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelChildManager.childExistsAt
	ChildExistsAt(index int) bool
	// CreateChild creates the child at the index and inserts it after the given
	// child (nil for the start).
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelChildManager.createChild
	CreateChild(index int, after Box)
	// RemoveChild removes the child.
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelChildManager.removeChild
	RemoveChild(child Box)
}

// ListWheelParentData is the parent data of the children of a
// RenderListWheelViewport: the index, and the transform that the child was last
// painted with.
//
// Flutter: rendering/list_wheel_viewport.dart ListWheelParentData
type ListWheelParentData struct {
	ContainerBoxParentData[Box]
	// Index is the index of the child in the list, or none before it is set.
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelParentData.index
	Index foundation.Opt[int]
	// Transform is the transform the child was last painted with, or nil.
	//
	// Flutter: rendering/list_wheel_viewport.dart ListWheelParentData.transform
	Transform *ui.Matrix4
}

// Detach panics in debug builds if the sibling links are still set.
//
// Flutter: rendering/object.dart ContainerParentDataMixin.detach
func (d *ListWheelParentData) Detach() { d.checkDetach() }

// RenderListWheelViewport renders a vertical list of children laid out on the
// surface of a cylinder, like the wheel of a slot machine.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport
type RenderListWheelViewport struct {
	RenderBox
	ContainerRenderObject[Box]
	childManager                  ListWheelChildManager
	offset                        ViewportOffset
	offsetHandle                  foundation.ListenerHandle
	diameterRatio                 float64
	perspective                   float64
	offAxisFraction               float64
	useMagnifier                  bool
	magnification                 float64
	overAndUnderCenterOpacity     float64
	itemExtent                    float64
	squeeze                       float64
	renderChildrenOutsideViewport bool
	clipBehavior                  ui.Clip
	clipRectLayer                 *LayerHandle[*ClipRectLayer]
	childOpacityLayerHandler      *LayerHandle[*OpacityLayer]
}

// The defaults and messages of RenderListWheelViewport.
const (
	// ListWheelDefaultDiameterRatio is the default ratio of the diameter of the
	// cylinder to the main axis extent of the viewport.
	//
	// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.defaultDiameterRatio
	ListWheelDefaultDiameterRatio = 2.0
	// ListWheelDefaultPerspective is the default perspective.
	//
	// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.defaultPerspective
	ListWheelDefaultPerspective = 0.003
)

const (
	listWheelDiameterRatioZeroMessage = "you can't set a diameterRatio of 0 or of a negative number; it would imply a cylinder of 0 in diameter in which case nothing will be drawn"
	listWheelPerspectiveTooHigh       = "a perspective too high will be clipped in the z-axis and therefore not renderable; value must be between 0 and 0.01"
	listWheelClipConflict             = "cannot renderChildrenOutsideViewport and clip since children rendered outside will be clipped anyway"
)

// RenderListWheelViewportOpts are the optional parameters of
// NewRenderListWheelViewport.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.new
type RenderListWheelViewportOpts struct {
	// DiameterRatio is the ratio of the diameter of the cylinder to the viewport
	// extent; the default is 2.
	DiameterRatio foundation.Opt[float64]
	// Perspective is how much the cylinder is distorted by the perspective; the
	// default is 0.003.
	Perspective foundation.Opt[float64]
	// OffAxisFraction is how far the cylinder is off the center of the viewport
	// horizontally.
	OffAxisFraction float64
	// UseMagnifier is whether to magnify the item in the center.
	UseMagnifier bool
	// Magnification is how much the center item is magnified; the default is 1.
	Magnification foundation.Opt[float64]
	// OverAndUnderCenterOpacity is the opacity of the items outside of the center
	// magnifier; the default is 1.
	OverAndUnderCenterOpacity foundation.Opt[float64]
	// Squeeze is how tightly the children are packed on the cylinder; the default
	// is 1.
	Squeeze foundation.Opt[float64]
	// RenderChildrenOutsideViewport is whether to build and paint children outside
	// of the viewport.
	RenderChildrenOutsideViewport bool
	// ClipBehavior is how the viewport clips; the default is ClipNone.
	ClipBehavior ui.Clip
	// Children are the initial children.
	Children []Box
}

// NewRenderListWheelViewport creates the viewport. In debug builds it panics with
// the tag ListWheelViewportInvalid when a value is out of its range.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.new
func NewRenderListWheelViewport(childManager ListWheelChildManager, offset ViewportOffset, itemExtent float64, opts ...RenderListWheelViewportOpts) *RenderListWheelViewport {
	o := firstOpts(opts)
	r := &RenderListWheelViewport{
		childManager: childManager, offset: offset, diameterRatio: o.DiameterRatio.OrElse(ListWheelDefaultDiameterRatio),
		perspective: o.Perspective.OrElse(ListWheelDefaultPerspective), offAxisFraction: o.OffAxisFraction, useMagnifier: o.UseMagnifier,
		magnification: o.Magnification.OrElse(1), overAndUnderCenterOpacity: o.OverAndUnderCenterOpacity.OrElse(1), itemExtent: itemExtent,
		squeeze: o.Squeeze.OrElse(1), renderChildrenOutsideViewport: o.RenderChildrenOutsideViewport, clipBehavior: o.ClipBehavior,
		clipRectLayer: NewLayerHandle[*ClipRectLayer](nil), childOpacityLayerHandler: NewLayerHandle[*OpacityLayer](nil),
	}
	if foundation.DebugMode {
		switch {
		case !(r.diameterRatio > 0):
			panic(errors.New(tagListWheelViewportInvalid, listWheelDiameterRatioZeroMessage, "diameterRatio", r.diameterRatio))
		case !(r.perspective > 0) || !(r.perspective <= 0.01):
			panic(errors.New(tagListWheelViewportInvalid, listWheelPerspectiveTooHigh, "perspective", r.perspective))
		case !(r.magnification > 0):
			panic(errors.New(tagListWheelViewportInvalid, "the magnification must be greater than zero", "magnification", r.magnification))
		case !(r.overAndUnderCenterOpacity >= 0 && r.overAndUnderCenterOpacity <= 1):
			panic(errors.New(tagListWheelViewportInvalid, "the opacity must be between 0 and 1", "overAndUnderCenterOpacity", r.overAndUnderCenterOpacity))
		case !(r.squeeze > 0):
			panic(errors.New(tagListWheelViewportInvalid, "the squeeze must be greater than zero", "squeeze", r.squeeze))
		case !(r.itemExtent > 0):
			panic(errors.New(tagListWheelViewportInvalid, "the item extent must be greater than zero", "itemExtent", r.itemExtent))
		case r.renderChildrenOutsideViewport && r.clipBehavior != ui.ClipNone:
			panic(errors.New(tagListWheelViewportInvalid, listWheelClipConflict))
		}
	}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(o.Children)
	return r
}

// ChildManager creates and removes the children.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.childManager
func (r *RenderListWheelViewport) ChildManager() ListWheelChildManager { return r.childManager }

// Offset is the offset of the viewport.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.offset
func (r *RenderListWheelViewport) Offset() ViewportOffset { return r.offset }

// SetOffset changes the offset and marks the viewport as needing layout.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.offset
func (r *RenderListWheelViewport) SetOffset(value ViewportOffset) {
	if value == r.offset {
		return
	}
	if r.Attached() {
		r.offset.RemoveListener(r.offsetHandle)
	}
	r.offset = value
	if r.Attached() {
		r.offsetHandle = r.offset.AddListener(r.hasScrolled)
	}
	r.MarkNeedsLayout()
}

// DiameterRatio is the ratio of the diameter of the cylinder to the viewport
// extent.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.diameterRatio
func (r *RenderListWheelViewport) DiameterRatio() float64 { return r.diameterRatio }

// SetDiameterRatio changes the ratio.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.diameterRatio
func (r *RenderListWheelViewport) SetDiameterRatio(value float64) {
	if foundation.DebugMode && !(value > 0) {
		panic(errors.New(tagListWheelViewportInvalid, listWheelDiameterRatioZeroMessage, "diameterRatio", value))
	}
	if value == r.diameterRatio {
		return
	}
	r.diameterRatio = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// Perspective is how much the cylinder is distorted by the perspective.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.perspective
func (r *RenderListWheelViewport) Perspective() float64 { return r.perspective }

// SetPerspective changes the perspective.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.perspective
func (r *RenderListWheelViewport) SetPerspective(value float64) {
	if foundation.DebugMode && (!(value > 0) || !(value <= 0.01)) {
		panic(errors.New(tagListWheelViewportInvalid, listWheelPerspectiveTooHigh, "perspective", value))
	}
	if value == r.perspective {
		return
	}
	r.perspective = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// OffAxisFraction is how far the cylinder is off the center of the viewport.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.offAxisFraction
func (r *RenderListWheelViewport) OffAxisFraction() float64 { return r.offAxisFraction }

// SetOffAxisFraction changes the fraction.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.offAxisFraction
func (r *RenderListWheelViewport) SetOffAxisFraction(value float64) {
	if value == r.offAxisFraction {
		return
	}
	r.offAxisFraction = value
	r.MarkNeedsPaint()
}

// UseMagnifier is whether the item in the center is magnified.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.useMagnifier
func (r *RenderListWheelViewport) UseMagnifier() bool { return r.useMagnifier }

// SetUseMagnifier changes whether the center is magnified.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.useMagnifier
func (r *RenderListWheelViewport) SetUseMagnifier(value bool) {
	if value == r.useMagnifier {
		return
	}
	r.useMagnifier = value
	r.MarkNeedsPaint()
}

// Magnification is how much the center item is magnified.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.magnification
func (r *RenderListWheelViewport) Magnification() float64 { return r.magnification }

// SetMagnification changes the magnification.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.magnification
func (r *RenderListWheelViewport) SetMagnification(value float64) {
	if foundation.DebugMode && !(value > 0) {
		panic(errors.New(tagListWheelViewportInvalid, "the magnification must be greater than zero", "magnification", value))
	}
	if value == r.magnification {
		return
	}
	r.magnification = value
	r.MarkNeedsPaint()
}

// OverAndUnderCenterOpacity is the opacity of the items outside of the center.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.overAndUnderCenterOpacity
func (r *RenderListWheelViewport) OverAndUnderCenterOpacity() float64 {
	return r.overAndUnderCenterOpacity
}

// SetOverAndUnderCenterOpacity changes the opacity.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.overAndUnderCenterOpacity
func (r *RenderListWheelViewport) SetOverAndUnderCenterOpacity(value float64) {
	if foundation.DebugMode && !(value >= 0 && value <= 1) {
		panic(errors.New(tagListWheelViewportInvalid, "the opacity must be between 0 and 1", "overAndUnderCenterOpacity", value))
	}
	if value == r.overAndUnderCenterOpacity {
		return
	}
	r.overAndUnderCenterOpacity = value
	r.MarkNeedsPaint()
}

// ItemExtent is the extent of every child along the main axis.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.itemExtent
func (r *RenderListWheelViewport) ItemExtent() float64 { return r.itemExtent }

// SetItemExtent changes the extent and marks the viewport as needing layout.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.itemExtent
func (r *RenderListWheelViewport) SetItemExtent(value float64) {
	if foundation.DebugMode && !(value > 0) {
		panic(errors.New(tagListWheelViewportInvalid, "the item extent must be greater than zero", "itemExtent", value))
	}
	if value == r.itemExtent {
		return
	}
	r.itemExtent = value
	r.MarkNeedsLayout()
}

// Squeeze is how tightly the children are packed on the cylinder.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.squeeze
func (r *RenderListWheelViewport) Squeeze() float64 { return r.squeeze }

// SetSqueeze changes the squeeze and marks the viewport as needing layout.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.squeeze
func (r *RenderListWheelViewport) SetSqueeze(value float64) {
	if foundation.DebugMode && !(value > 0) {
		panic(errors.New(tagListWheelViewportInvalid, "the squeeze must be greater than zero", "squeeze", value))
	}
	if value == r.squeeze {
		return
	}
	r.squeeze = value
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

// RenderChildrenOutsideViewport is whether children outside of the viewport are
// built and painted.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.renderChildrenOutsideViewport
func (r *RenderListWheelViewport) RenderChildrenOutsideViewport() bool {
	return r.renderChildrenOutsideViewport
}

// SetRenderChildrenOutsideViewport changes whether children outside of the
// viewport are built and painted.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.renderChildrenOutsideViewport
func (r *RenderListWheelViewport) SetRenderChildrenOutsideViewport(value bool) {
	if foundation.DebugMode && r.renderChildrenOutsideViewport && r.clipBehavior != ui.ClipNone {
		panic(errors.New(tagListWheelViewportInvalid, listWheelClipConflict))
	}
	if value == r.renderChildrenOutsideViewport {
		return
	}
	r.renderChildrenOutsideViewport = value
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

// ClipBehavior is how the viewport clips.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.clipBehavior
func (r *RenderListWheelViewport) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes how the viewport clips.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.clipBehavior
func (r *RenderListWheelViewport) SetClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *RenderListWheelViewport) hasScrolled() {
	r.MarkNeedsLayout()
	r.MarkNeedsSemanticsUpdate()
}

// SetupParentData gives the child a ListWheelParentData if it has none.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.setupParentData
func (r *RenderListWheelViewport) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*ListWheelParentData); !ok {
		child.AsRenderObject().SetParentData(&ListWheelParentData{})
	}
}

// Attach listens to the offset.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.attach
func (r *RenderListWheelViewport) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.offsetHandle = r.offset.AddListener(r.hasScrolled)
}

// Detach stops listening to the offset.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.detach
func (r *RenderListWheelViewport) Detach() {
	r.offset.RemoveListener(r.offsetHandle)
	r.RenderBox.Detach()
}

// IsRepaintBoundary is true.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.isRepaintBoundary
func (r *RenderListWheelViewport) IsRepaintBoundary() bool { return true }

func (r *RenderListWheelViewport) viewportExtent() float64 { return r.Size().Height }

func (r *RenderListWheelViewport) minEstimatedScrollExtent() float64 {
	if r.childManager.ChildCount().IsNone() {
		return math.Inf(-1)
	}
	return 0
}

func (r *RenderListWheelViewport) maxEstimatedScrollExtent() float64 {
	count, ok := r.childManager.ChildCount().Get()
	if !ok {
		return math.Inf(1)
	}
	return math.Max(0, float64(count-1)*r.itemExtent)
}

func (r *RenderListWheelViewport) topScrollMarginExtent() float64 {
	return -r.Size().Height/2 + r.itemExtent/2
}

func (r *RenderListWheelViewport) untransformedPaintingCoordinateY(layoutCoordinateY float64) float64 {
	return layoutCoordinateY - r.topScrollMarginExtent() - r.offset.Pixels()
}

func (r *RenderListWheelViewport) maxVisibleRadian() float64 {
	if r.diameterRatio < 1 {
		return math.Pi / 2
	}
	return math.Asin(1 / r.diameterRatio)
}

func (r *RenderListWheelViewport) intrinsicCrossAxis(childSize func(child Box) float64) float64 {
	extent := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		extent = math.Max(extent, childSize(child))
	}
	return extent
}

// ComputeMinIntrinsicWidth is the largest child.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.computeMinIntrinsicWidth
func (r *RenderListWheelViewport) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.intrinsicCrossAxis(func(child Box) float64 { return child.GetMinIntrinsicWidth(height) })
}

// ComputeMaxIntrinsicWidth is the largest child.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.computeMaxIntrinsicWidth
func (r *RenderListWheelViewport) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.intrinsicCrossAxis(func(child Box) float64 { return child.GetMaxIntrinsicWidth(height) })
}

// ComputeMinIntrinsicHeight is the extent of all the children.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.computeMinIntrinsicHeight
func (r *RenderListWheelViewport) ComputeMinIntrinsicHeight(float64) float64 {
	count, ok := r.childManager.ChildCount().Get()
	if !ok {
		return 0
	}
	return float64(count) * r.itemExtent
}

// ComputeMaxIntrinsicHeight is the extent of all the children.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.computeMaxIntrinsicHeight
func (r *RenderListWheelViewport) ComputeMaxIntrinsicHeight(float64) float64 {
	count, ok := r.childManager.ChildCount().Get()
	if !ok {
		return 0
	}
	return float64(count) * r.itemExtent
}

// SizedByParent is true.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.sizedByParent
func (r *RenderListWheelViewport) SizedByParent() bool { return true }

// ComputeDryLayout is the biggest size.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.computeDryLayout
func (r *RenderListWheelViewport) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// IndexOf is the index of the child in the list.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.indexOf
func (r *RenderListWheelViewport) IndexOf(child Box) int {
	data := child.ParentData().(*ListWheelParentData)
	if foundation.DebugMode && data.Index.IsNone() {
		panic(errors.New(tagListWheelChildIndexMissing, "a child of a list wheel viewport has no index"))
	}
	return data.Index.Value()
}

// ScrollOffsetToIndex is the index of the child that the scroll offset is in.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.scrollOffsetToIndex
func (r *RenderListWheelViewport) ScrollOffsetToIndex(scrollOffset float64) int {
	return int(math.Floor(scrollOffset / r.itemExtent))
}

// IndexToScrollOffset is the scroll offset of the child at the index.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.indexToScrollOffset
func (r *RenderListWheelViewport) IndexToScrollOffset(index int) float64 {
	return float64(index) * r.itemExtent
}

func (r *RenderListWheelViewport) createChild(index int, after Box) {
	r.InvokeLayoutCallback(func(Constraints) { r.childManager.CreateChild(index, after) })
}

func (r *RenderListWheelViewport) destroyChild(child Box) {
	r.InvokeLayoutCallback(func(Constraints) { r.childManager.RemoveChild(child) })
}

func (r *RenderListWheelViewport) layoutChild(child Box, constraints BoxConstraints, index int) {
	child.Layout(constraints, LayoutOpts{ParentUsesSize: true})
	data := child.ParentData().(*ListWheelParentData)
	crossPosition := r.Size().Width/2 - child.Size().Width/2
	data.Offset = ui.Offset{Dx: crossPosition, Dy: r.IndexToScrollOffset(index)}
}

// PerformLayout builds and lays out the children that are visible, and tells the
// offset about the extents.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.performLayout
func (r *RenderListWheelViewport) PerformLayout() {
	r.offset.ApplyViewportDimension(r.viewportExtent())
	if r.childManager.ChildCount().IsSome() {
		r.offset.ApplyContentDimensions(r.minEstimatedScrollExtent(), r.maxEstimatedScrollExtent())
	}
	// The height, in pixel, that children will be visible and might be laid out
	// and painted.
	visibleHeight := r.Size().Height * r.squeeze
	// If renderChildrenOutsideViewport is true, we spawn extra children by
	// doubling the visibility range, those that are in the backside of the
	// cylinder won't be painted anyway.
	if r.renderChildrenOutsideViewport {
		visibleHeight *= 2
	}
	firstVisibleOffset := r.offset.Pixels() + r.itemExtent/2 - visibleHeight/2
	lastVisibleOffset := firstVisibleOffset + visibleHeight

	// The index range that we want to spawn children. We find indexes that
	// are in the interval [firstVisibleOffset, lastVisibleOffset).
	targetFirstIndex := r.ScrollOffsetToIndex(firstVisibleOffset)
	targetLastIndex := r.ScrollOffsetToIndex(lastVisibleOffset)
	// Because we exclude lastVisibleOffset, if there's a new child starting at
	// that offset, it is removed.
	if float64(targetLastIndex)*r.itemExtent == lastVisibleOffset {
		targetLastIndex--
	}
	// Validates the target index range.
	for !r.childManager.ChildExistsAt(targetFirstIndex) && targetFirstIndex <= targetLastIndex {
		targetFirstIndex++
	}
	for !r.childManager.ChildExistsAt(targetLastIndex) && targetFirstIndex <= targetLastIndex {
		targetLastIndex--
	}
	// If it turns out there's no children to layout, we remove old children and
	// return.
	if targetFirstIndex > targetLastIndex {
		for !isNil(r.FirstChild()) {
			r.destroyChild(r.FirstChild())
		}
		return
	}
	// Now there are 2 cases:
	//  - The target index range and our current index range have intersection:
	//    We shorten and extend our current child list so that the two lists
	//    match. Most of the time we are in this case.
	//  - The target list and our current child list have no intersection: We
	//    first remove all children and then add one child from the target list.
	//    Later the method will extend the child list to cover the target list.
	if r.ChildCount() > 0 && (r.IndexOf(r.FirstChild()) > targetLastIndex || r.IndexOf(r.LastChild()) < targetFirstIndex) {
		for !isNil(r.FirstChild()) {
			r.destroyChild(r.FirstChild())
		}
	}
	childConstraints := r.BoxConstraints().CopyWith(BoxConstraintsCopy{
		MinHeight: foundation.Some(r.itemExtent), MaxHeight: foundation.Some(r.itemExtent), MinWidth: foundation.Some(0.0)})
	// If there is no children, create one.
	if r.ChildCount() == 0 {
		r.createChild(targetFirstIndex, nil)
		r.layoutChild(r.FirstChild(), childConstraints, targetFirstIndex)
	}
	// Now there are at least one child.
	currentFirstIndex := r.IndexOf(r.FirstChild())
	currentLastIndex := r.IndexOf(r.LastChild())
	// Remove all unnecessary children by shortening the current child list, in
	// both directions.
	for currentFirstIndex < targetFirstIndex {
		r.destroyChild(r.FirstChild())
		currentFirstIndex++
	}
	for currentLastIndex > targetLastIndex {
		r.destroyChild(r.LastChild())
		currentLastIndex--
	}
	// Relayout all active children.
	index := currentFirstIndex
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		r.layoutChild(child, childConstraints, index)
		index++
	}
	// Spawning new children that are actually visible but not in child list yet.
	for currentFirstIndex > targetFirstIndex {
		r.createChild(currentFirstIndex-1, nil)
		currentFirstIndex--
		r.layoutChild(r.FirstChild(), childConstraints, currentFirstIndex)
	}
	for currentLastIndex < targetLastIndex {
		r.createChild(currentLastIndex+1, r.LastChild())
		currentLastIndex++
		r.layoutChild(r.LastChild(), childConstraints, currentLastIndex)
	}
	minScrollExtent := r.IndexToScrollOffset(targetFirstIndex)
	if r.childManager.ChildExistsAt(targetFirstIndex - 1) {
		minScrollExtent = r.minEstimatedScrollExtent()
	}
	maxScrollExtent := r.IndexToScrollOffset(targetLastIndex)
	if r.childManager.ChildExistsAt(targetLastIndex + 1) {
		maxScrollExtent = r.maxEstimatedScrollExtent()
	}
	r.offset.ApplyContentDimensions(minScrollExtent, maxScrollExtent)
}

func (r *RenderListWheelViewport) shouldClipAtCurrentOffset() bool {
	highestUntransformedPaintY := r.untransformedPaintingCoordinateY(0)
	return highestUntransformedPaintY < 0 || r.Size().Height < highestUntransformedPaintY+r.maxEstimatedScrollExtent()+r.itemExtent
}

// Paint paints the children on the cylinder.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.paint
func (r *RenderListWheelViewport) Paint(context *PaintingContext, offset ui.Offset) {
	if r.ChildCount() > 0 {
		if r.shouldClipAtCurrentOffset() && r.clipBehavior != ui.ClipNone {
			r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), r.paintVisibleChildren,
				PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
		} else {
			r.clipRectLayer.SetLayer(nil)
			r.paintVisibleChildren(context, offset)
		}
	}
}

// Dispose lets the layers go.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.dispose
func (r *RenderListWheelViewport) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.childOpacityLayerHandler.SetLayer(nil)
	r.RenderBox.Dispose()
}

// paintCenter says which children paintAllChildren paints: all, the center, or
// the others.
type paintCenter int

const (
	paintCenterAll paintCenter = iota
	paintCenterOnly
	paintCenterOthers
)

func (r *RenderListWheelViewport) paintVisibleChildren(context *PaintingContext, offset ui.Offset) {
	// The never-opaque case is the fast path: paint everything.
	if r.overAndUnderCenterOpacity >= 1 {
		r.paintAllChildren(context, offset, paintCenterAll)
		return
	}
	// In order to reduce the number of opacity layers, we first paint all
	// off-center children in one opacity layer, then the center ones.
	r.childOpacityLayerHandler.SetLayer(context.PushOpacity(offset, int(math.Round(r.overAndUnderCenterOpacity*255)),
		func(context *PaintingContext, offset ui.Offset) {
			r.paintAllChildren(context, offset, paintCenterOthers)
		},
		r.childOpacityLayerHandler.Layer()))
	r.paintAllChildren(context, offset, paintCenterOnly)
}

func (r *RenderListWheelViewport) paintAllChildren(context *PaintingContext, offset ui.Offset, center paintCenter) {
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		data := child.ParentData().(*ListWheelParentData)
		r.paintTransformedChild(child, context, offset, data.Offset, center)
	}
}

func (r *RenderListWheelViewport) paintTransformedChild(child Box, context *PaintingContext, offset, layoutOffset ui.Offset, center paintCenter) {
	untransformedPaintingCoordinates := offset.Add(ui.Offset{Dx: layoutOffset.Dx, Dy: r.untransformedPaintingCoordinateY(layoutOffset.Dy)})
	// Get child's center as a fraction of the viewport's height.
	fractionalY := (untransformedPaintingCoordinates.Dy + r.itemExtent/2) / r.Size().Height
	// Angle of the item on the cylinder: 0 at the center, +/- pi/2 on the sides.
	angle := -(fractionalY - 0.5) * 2 * r.maxVisibleRadian() / r.squeeze
	// Don't paint the backside of the cylinder when diameterRatio < 1, instead
	// of letting the cylinder cover the other side.
	if angle > math.Pi/2 || angle < -math.Pi/2 || math.IsNaN(angle) {
		return
	}
	transform := ui.NewMatrix4CylindricalProjection(r.Size().Height*r.diameterRatio/2, angle, r.perspective, true)
	// Offset that helps painting everything in the center (e.g. angle = 0).
	offsetToCenter := ui.Offset{Dx: untransformedPaintingCoordinates.Dx, Dy: -r.topScrollMarginExtent()}
	if r.useMagnifier || r.overAndUnderCenterOpacity < 1 {
		r.paintChildWithMagnifier(context, offset, child, transform, offsetToCenter, untransformedPaintingCoordinates, center)
	} else {
		r.paintChildCylindrically(context, offset, child, transform, offsetToCenter)
	}
}

func (r *RenderListWheelViewport) paintChildWithMagnifier(context *PaintingContext, offset ui.Offset, child Box, cylindricalTransform *ui.Matrix4,
	offsetToCenter, untransformedPaintingCoordinates ui.Offset, center paintCenter) {
	magnifierTopLinePosition := r.Size().Height/2 - r.itemExtent*r.magnification/2
	magnifierBottomLinePosition := r.Size().Height/2 + r.itemExtent*r.magnification/2
	isAfterMagnifierTopLine := untransformedPaintingCoordinates.Dy >= magnifierTopLinePosition-r.itemExtent*r.magnification
	isBeforeMagnifierBottomLine := untransformedPaintingCoordinates.Dy <= magnifierBottomLinePosition
	centerRect := ui.NewRectFromLTWH(0, magnifierTopLinePosition, r.Size().Width, r.itemExtent*r.magnification)
	topHalfRect := ui.NewRectFromLTWH(0, 0, r.Size().Width, magnifierTopLinePosition)
	bottomHalfRect := ui.NewRectFromLTWH(0, magnifierBottomLinePosition, r.Size().Width, magnifierTopLinePosition)
	inCenter := isAfterMagnifierTopLine && isBeforeMagnifierBottomLine

	if (center == paintCenterAll || center == paintCenterOnly) && inCenter {
		context.PushClipRect(r.NeedsCompositing(), offset, centerRect, func(context *PaintingContext, offset ui.Offset) {
			context.PushTransform(r.NeedsCompositing(), offset, r.magnifyTransform(), func(context *PaintingContext, offset ui.Offset) {
				context.PaintChild(child, offset.Add(untransformedPaintingCoordinates))
			}, nil)
		})
	}
	if (center == paintCenterAll || center == paintCenterOthers) && inCenter {
		clip := bottomHalfRect
		if untransformedPaintingCoordinates.Dy <= magnifierTopLinePosition {
			clip = topHalfRect
		}
		context.PushClipRect(r.NeedsCompositing(), offset, clip, func(context *PaintingContext, offset ui.Offset) {
			r.paintChildCylindrically(context, offset, child, cylindricalTransform, offsetToCenter)
		})
	}
	// Paint the non-center children cylindrically.
	if (center == paintCenterAll || center == paintCenterOthers) && !inCenter {
		r.paintChildCylindrically(context, offset, child, cylindricalTransform, offsetToCenter)
	}
}

func (r *RenderListWheelViewport) paintChildCylindrically(context *PaintingContext, offset ui.Offset, child Box, cylindricalTransform *ui.Matrix4, offsetToCenter ui.Offset) {
	paintOriginOffset := offset.Add(offsetToCenter)
	// Paint child with the given transform at the origin.
	context.PushTransform(r.NeedsCompositing(), offset, r.centerOriginTransform(cylindricalTransform), func(context *PaintingContext, _ ui.Offset) {
		context.PaintChild(child, paintOriginOffset)
	}, nil)
	// Save the final transform that accounts both for the offset and cylindrical
	// transform.
	transform := r.centerOriginTransform(cylindricalTransform)
	transform.Translate(float32(paintOriginOffset.Dx), float32(paintOriginOffset.Dy), 0)
	child.ParentData().(*ListWheelParentData).Transform = transform
}

func (r *RenderListWheelViewport) magnifyTransform() *ui.Matrix4 {
	result := ui.NewMatrix4Identity()
	result.Translate(float32(r.Size().Width*(-r.offAxisFraction+0.5)), float32(r.Size().Height/2), 0)
	result.Scale(float32(r.magnification), float32(r.magnification), float32(r.magnification))
	result.Translate(float32(-r.Size().Width*(-r.offAxisFraction+0.5)), float32(-r.Size().Height/2), 0)
	return result
}

func (r *RenderListWheelViewport) centerOriginTransform(originalMatrix *ui.Matrix4) *ui.Matrix4 {
	result := ui.NewMatrix4Identity()
	centerOriginTranslation := painting.AlignmentCenter.AlongSize(r.Size())
	result.Translate(float32(centerOriginTranslation.Dx*(-r.offAxisFraction*2+1)), float32(centerOriginTranslation.Dy), 0)
	result.MultiplyBy(originalMatrix)
	result.Translate(float32(-centerOriginTranslation.Dx*(-r.offAxisFraction*2+1)), float32(-centerOriginTranslation.Dy), 0)
	return result
}

// ApplyPaintTransform multiplies by the transform that the child was painted with.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.applyPaintTransform
func (r *RenderListWheelViewport) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if t := child.ParentData().(*ListWheelParentData).Transform; t != nil {
		transform.MultiplyBy(t)
	}
}

// DescribeApproximatePaintClip is the viewport, if the children overflow it.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.describeApproximatePaintClip
func (r *RenderListWheelViewport) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if r.shouldClipAtCurrentOffset() {
		return foundation.Some(ui.OffsetZero.And(r.Size()))
	}
	return foundation.None[ui.Rect]()
}

// HitTestChildren hit tests the children with the transforms that they were
// painted with, last first.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.hitTestChildren
func (r *RenderListWheelViewport) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); !isNil(child); {
		data := child.ParentData().(*ListWheelParentData)
		if data.Transform != nil {
			c := child
			isHit := result.AddWithPaintTransform(data.Transform, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
				return c.HitTest(result, transformed)
			})
			if isHit {
				return true
			}
		}
		child = data.PreviousSibling
	}
	return false
}

// GetOffsetToReveal is the scroll offset that puts the target at the center of
// the wheel; only the vertical axis is supported.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.getOffsetToReveal
func (r *RenderListWheelViewport) GetOffsetToReveal(target RenderObject, _ float64, opts ...GetOffsetToRevealOpts) RevealedOffset {
	var o GetOffsetToRevealOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	rect := o.Rect.OrElse(target.PaintBounds())
	child := target
	for child.Parent() != RenderObject(r) {
		child = child.Parent()
	}
	data := child.ParentData().(*ListWheelParentData)
	targetOffset := data.Offset.Dy // the so-called "centerPosition"
	transform := target.GetTransformTo(child)
	bounds := transform.TransformRect(rect)
	targetRect := bounds.Translate(0, (r.Size().Height-r.itemExtent)/2)
	return RevealedOffset{Offset: targetOffset, Rect: targetRect}
}

// ShowOnScreen scrolls the descendant to the center of the wheel, then asks the
// parent to show the rectangle.
//
// Flutter: rendering/list_wheel_viewport.dart RenderListWheelViewport.showOnScreen
func (r *RenderListWheelViewport) ShowOnScreen(opts ShowOnScreenOpts) {
	rect := opts.Rect
	if !isNil(opts.Descendant) {
		revealedOffset := r.GetOffsetToReveal(opts.Descendant, 0.5, GetOffsetToRevealOpts{Rect: rect})
		if opts.Duration == 0 {
			r.offset.JumpTo(revealedOffset.Offset)
		} else {
			r.offset.AnimateTo(revealedOffset.Offset, opts.Duration, opts.Curve)
		}
		rect = foundation.Some(revealedOffset.Rect)
	}
	r.RenderBox.ShowOnScreen(ShowOnScreenOpts{Rect: rect, Duration: opts.Duration, Curve: opts.Curve})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagListWheelChildIndexMissing = "ListWheelChildIndexMissing"
	tagListWheelViewportInvalid   = "ListWheelViewportInvalid"
)
