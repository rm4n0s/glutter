package rendering

import (
	"github.com/rm4n0s/glutter/animation"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ChildVicinity is the location of a child in the two-dimensional space of a
// RenderTwoDimensionalViewport: the index along each axis. It orders by x, then y.
//
// Deviation: Flutter declares it, and the classes below, in the widgets library
// (widgets/two_dimensional_viewport.dart), where it can reach the private
// methods of the viewport. Go has no library-private members, so the render
// object lives in rendering and widgets re-exports the types (SPEC §12.10).
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity
type ChildVicinity struct {
	// XIndex is the index along the x axis.
	//
	// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.xIndex
	XIndex int
	// YIndex is the index along the y axis.
	//
	// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.yIndex
	YIndex int
}

// ChildVicinityInvalid is the vicinity of no child.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.invalid
var ChildVicinityInvalid = ChildVicinity{XIndex: -1, YIndex: -1}

// NewChildVicinity creates a vicinity. In debug builds it panics with the tag
// ChildVicinityInvalid when an index is less than -1.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.new
func NewChildVicinity(xIndex, yIndex int) ChildVicinity {
	if foundation.DebugMode && (xIndex < -1 || yIndex < -1) {
		panic(errors.New(tagChildVicinityInvalid, "the indices of a ChildVicinity must not be less than -1", "xIndex", xIndex, "yIndex", yIndex))
	}
	return ChildVicinity{XIndex: xIndex, YIndex: yIndex}
}

// CompareTo orders by the x index, then by the y index.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.compareTo
func (v ChildVicinity) CompareTo(other ChildVicinity) int {
	if v.XIndex == other.XIndex {
		return v.YIndex - other.YIndex
	}
	return v.XIndex - other.XIndex
}

// String is "(xIndex: 1, yIndex: 2)".
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.toString
func (v ChildVicinity) String() string {
	return "(xIndex: " + strconv.Itoa(v.XIndex) + ", yIndex: " + strconv.Itoa(v.YIndex) + ")"
}

// TwoDimensionalChildManager creates, reuses and removes the children of a
// RenderTwoDimensionalViewport while it lays itself out.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalChildManager
type TwoDimensionalChildManager interface {
	// StartLayout is called before the viewport lays its children out.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalChildManager._startLayout
	StartLayout()
	// BuildChild builds the child at the vicinity, if the delegate has one.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalChildManager._buildChild
	BuildChild(vicinity ChildVicinity)
	// ReuseChild keeps the child at the vicinity as it is.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalChildManager._reuseChild
	ReuseChild(vicinity ChildVicinity)
	// EndLayout is called after the viewport laid its children out: the children
	// that were not built or reused are unmounted.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalChildManager._endLayout
	EndLayout()
}

// TwoDimensionalChildDelegate is what the render object needs of the delegate
// that supplies the children of a two-dimensional viewport: it notifies its
// listeners when the children change.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildDelegate
type TwoDimensionalChildDelegate interface {
	foundation.Listenable
	// ShouldRebuild is whether the viewport must build its children again when
	// this delegate replaces the old one, which has the same type.
	//
	// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildDelegate.shouldRebuild
	ShouldRebuild(oldDelegate TwoDimensionalChildDelegate) bool
}

// TwoDimensionalViewportParentData is the parent data of the children of a
// RenderTwoDimensionalViewport: where the child is laid out, where it paints and
// how much of it shows.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData
type TwoDimensionalViewportParentData struct {
	ParentDataBase
	KeepAliveParentDataMixin
	// LayoutOffset is the position of the child in the scrollable space, set by
	// LayoutChildSequence of the viewport.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.layoutOffset
	LayoutOffset foundation.Opt[ui.Offset]
	// Vicinity is the vicinity of the child.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.vicinity
	Vicinity ChildVicinity
	// PaintOffset is the position of the child in the viewport, set after the
	// layout.
	//
	// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.paintOffset
	PaintOffset foundation.Opt[ui.Offset]

	paintExtent     foundation.Opt[ui.Size]
	previousSibling Box
	nextSibling     Box
}

// IsVisible is whether any part of the child is in the viewport. In debug builds
// it panics with the tag ParentDataPaintExtentUnknown before the layout of the
// viewport sets the paint extent.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.isVisible
func (d *TwoDimensionalViewportParentData) IsVisible() bool {
	extent, ok := d.paintExtent.Get()
	if !ok {
		if foundation.DebugMode {
			panic(errors.New(tagParentDataPaintExtentUnknown, "the paint extent of the child has not been determined yet; the paint extent, and therefore the visibility, of a child of a RenderTwoDimensionalViewport is computed after LayoutChildSequence"))
		}
		return false
	}
	return extent != (ui.Size{}) || extent.Height != 0 || extent.Width != 0
}

// PaintExtent is the part of the child that is visible, once known.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData._paintExtent
func (d *TwoDimensionalViewportParentData) PaintExtent() foundation.Opt[ui.Size] {
	return d.paintExtent
}

// KeptAlive is whether the viewport keeps the child alive while it is out of view.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.keptAlive
func (d *TwoDimensionalViewportParentData) KeptAlive() bool {
	if !d.KeepAlive {
		return false
	}
	extent, ok := d.paintExtent.Get()
	if !ok {
		return d.KeepAlive
	}
	visible := extent != (ui.Size{}) || extent.Height != 0 || extent.Width != 0
	return !visible
}

// String describes the data.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData.toString
func (d *TwoDimensionalViewportParentData) String() string {
	opt := func(o foundation.Opt[ui.Offset]) string {
		if v, ok := o.Get(); ok {
			return v.String()
		}
		return "null"
	}
	s := "vicinity=" + d.Vicinity.String() + "; layoutOffset=" + opt(d.LayoutOffset) + "; paintOffset=" + opt(d.PaintOffset) + "; "
	if extent, ok := d.paintExtent.Get(); !ok {
		s += "not visible; "
	} else {
		if !d.IsVisible() {
			s += "not "
		}
		s += "visible - paintExtent=" + extent.String() + "; "
	}
	if d.KeepAlive {
		s += "keepAlive; "
	}
	return s
}

// KeepAliveData is the keep alive mixin.
func (d *TwoDimensionalViewportParentData) KeepAliveData() *KeepAliveParentDataMixin {
	return &d.KeepAliveParentDataMixin
}

// vicinityBoxes is a map from vicinities to boxes that iterates in insertion
// order, as Dart's map does.
type vicinityBoxes struct {
	items map[ChildVicinity]Box
	order []ChildVicinity
}

func (m *vicinityBoxes) get(v ChildVicinity) (Box, bool) {
	b, ok := m.items[v]
	return b, ok
}

func (m *vicinityBoxes) has(v ChildVicinity) bool { _, ok := m.items[v]; return ok }

func (m *vicinityBoxes) set(v ChildVicinity, b Box) {
	if m.items == nil {
		m.items = map[ChildVicinity]Box{}
	}
	if _, ok := m.items[v]; !ok {
		m.order = append(m.order, v)
	}
	m.items[v] = b
}

func (m *vicinityBoxes) remove(v ChildVicinity) {
	if _, ok := m.items[v]; !ok {
		return
	}
	delete(m.items, v)
	for i, x := range m.order {
		if x == v {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

func (m *vicinityBoxes) clear() { m.items, m.order = nil, nil }

func (m *vicinityBoxes) values() []Box {
	out := make([]Box, 0, len(m.order))
	for _, v := range m.order {
		out = append(out, m.items[v])
	}
	return out
}

func (m *vicinityBoxes) len() int { return len(m.items) }

func (m *vicinityBoxes) containsValue(b Box) bool {
	for _, x := range m.items {
		if x == b {
			return true
		}
	}
	return false
}

// TwoDimensionalViewportSelf is what RenderTwoDimensionalViewport needs of the
// type that embeds it.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.layoutChildSequence
type TwoDimensionalViewportSelf interface {
	Box
	// LayoutChildSequence lays the children out: it builds each child that
	// should be in the viewport with BuildOrObtainChildFor, lays it out, and sets
	// the LayoutOffset of its parent data. It must also give both offsets their
	// content dimensions.
	//
	// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.layoutChildSequence
	LayoutChildSequence()
}

// RenderTwoDimensionalViewportOpts are the optional parameters of
// InitRenderTwoDimensionalViewport.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.new
type RenderTwoDimensionalViewportOpts struct {
	// ScrollCacheExtent is how far beyond the viewport children are built; the
	// default is 250 pixels.
	ScrollCacheExtent foundation.Opt[ScrollCacheExtent]
	// ClipBehavior is how the viewport clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// RenderTwoDimensionalViewport is the base of the render objects that show a
// window onto a two-dimensional space of children, which scrolls along both
// axes. The concrete type embeds it, calls InitRenderTwoDimensionalViewport and
// implements LayoutChildSequence.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport
type RenderTwoDimensionalViewport struct {
	RenderBox
	self TwoDimensionalViewportSelf

	horizontalOffset        ViewportOffset
	horizontalHandle        foundation.ListenerHandle
	horizontalAxisDirection painting.AxisDirection
	verticalOffset          ViewportOffset
	verticalHandle          foundation.ListenerHandle
	verticalAxisDirection   painting.AxisDirection
	delegate                TwoDimensionalChildDelegate
	delegateHandle          foundation.ListenerHandle
	mainAxis                painting.Axis
	scrollCacheExtent       ScrollCacheExtent
	clipBehavior            ui.Clip
	childManager            TwoDimensionalChildManager

	children                vicinityBoxes
	activeChildrenForLayout vicinityBoxes
	keepAliveBucket         vicinityBoxes
	debugDanglingKeepAlives []Box
	debugOrphans            []Box
	hasVisualOverflow       bool
	clipRectLayer           *LayerHandle[*ClipRectLayer]
	currentChildVicinities  []ChildVicinity
	firstChild, lastChild   Box
	didResize               bool
	needsDelegateRebuild    bool
}

// InitRenderTwoDimensionalViewport sets the outer object and the properties. In
// debug builds it panics with the tag TwoDimensionalViewportAxisInvalid when a
// direction is along the wrong axis.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.new
func InitRenderTwoDimensionalViewport(r *RenderTwoDimensionalViewport, self TwoDimensionalViewportSelf,
	horizontalOffset ViewportOffset, horizontalAxisDirection painting.AxisDirection,
	verticalOffset ViewportOffset, verticalAxisDirection painting.AxisDirection,
	delegate TwoDimensionalChildDelegate, mainAxis painting.Axis, childManager TwoDimensionalChildManager,
	opts ...RenderTwoDimensionalViewportOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if verticalAxisDirection != painting.AxisDirectionDown && verticalAxisDirection != painting.AxisDirectionUp {
			panic(errors.New(tagTwoDimensionalViewportAxisInvalid, "TwoDimensionalViewport.verticalAxisDirection is not Axis.vertical"))
		}
		if horizontalAxisDirection != painting.AxisDirectionLeft && horizontalAxisDirection != painting.AxisDirectionRight {
			panic(errors.New(tagTwoDimensionalViewportAxisInvalid, "TwoDimensionalViewport.horizontalAxisDirection is not Axis.horizontal"))
		}
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	r.self = self
	r.childManager = childManager
	r.horizontalOffset, r.horizontalAxisDirection = horizontalOffset, horizontalAxisDirection
	r.verticalOffset, r.verticalAxisDirection = verticalOffset, verticalAxisDirection
	r.delegate, r.mainAxis = delegate, mainAxis
	r.scrollCacheExtent = o.ScrollCacheExtent.OrElse(NewScrollCacheExtentPixels(RenderAbstractViewportDefaultCacheExtent))
	r.clipBehavior = o.ClipBehavior
	r.clipRectLayer = NewLayerHandle[*ClipRectLayer](nil)
	r.didResize = true
	r.needsDelegateRebuild = true
	InitRenderBox(&r.RenderBox, self)
}

// HorizontalOffset is the offset of the horizontal axis.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.horizontalOffset
func (r *RenderTwoDimensionalViewport) HorizontalOffset() ViewportOffset { return r.horizontalOffset }

// SetHorizontalOffset changes the offset and marks the viewport as needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.horizontalOffset
func (r *RenderTwoDimensionalViewport) SetHorizontalOffset(value ViewportOffset) {
	if r.horizontalOffset == value {
		return
	}
	if r.Attached() {
		r.horizontalOffset.RemoveListener(r.horizontalHandle)
	}
	r.horizontalOffset = value
	if r.Attached() {
		r.horizontalHandle = r.horizontalOffset.AddListener(r.MarkNeedsLayout)
	}
	r.MarkNeedsLayout()
}

// HorizontalAxisDirection is the direction in which the horizontal offset grows.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.horizontalAxisDirection
func (r *RenderTwoDimensionalViewport) HorizontalAxisDirection() painting.AxisDirection {
	return r.horizontalAxisDirection
}

// SetHorizontalAxisDirection changes the direction and marks the viewport as
// needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.horizontalAxisDirection
func (r *RenderTwoDimensionalViewport) SetHorizontalAxisDirection(value painting.AxisDirection) {
	if r.horizontalAxisDirection == value {
		return
	}
	r.horizontalAxisDirection = value
	r.MarkNeedsLayout()
}

// VerticalOffset is the offset of the vertical axis.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.verticalOffset
func (r *RenderTwoDimensionalViewport) VerticalOffset() ViewportOffset { return r.verticalOffset }

// SetVerticalOffset changes the offset and marks the viewport as needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.verticalOffset
func (r *RenderTwoDimensionalViewport) SetVerticalOffset(value ViewportOffset) {
	if r.verticalOffset == value {
		return
	}
	if r.Attached() {
		r.verticalOffset.RemoveListener(r.verticalHandle)
	}
	r.verticalOffset = value
	if r.Attached() {
		r.verticalHandle = r.verticalOffset.AddListener(r.MarkNeedsLayout)
	}
	r.MarkNeedsLayout()
}

// VerticalAxisDirection is the direction in which the vertical offset grows.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.verticalAxisDirection
func (r *RenderTwoDimensionalViewport) VerticalAxisDirection() painting.AxisDirection {
	return r.verticalAxisDirection
}

// SetVerticalAxisDirection changes the direction and marks the viewport as
// needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.verticalAxisDirection
func (r *RenderTwoDimensionalViewport) SetVerticalAxisDirection(value painting.AxisDirection) {
	if r.verticalAxisDirection == value {
		return
	}
	r.verticalAxisDirection = value
	r.MarkNeedsLayout()
}

// Delegate supplies the children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.delegate
func (r *RenderTwoDimensionalViewport) Delegate() TwoDimensionalChildDelegate { return r.delegate }

// SetDelegate changes the delegate. The children are built again when its type
// changed or it says so.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.delegate
func (r *RenderTwoDimensionalViewport) SetDelegate(value TwoDimensionalChildDelegate) {
	if r.delegate == value {
		return
	}
	if r.Attached() {
		r.delegate.RemoveListener(r.delegateHandle)
	}
	oldDelegate := r.delegate
	r.delegate = value
	if r.Attached() {
		r.delegateHandle = r.delegate.AddListener(r.handleDelegateNotification)
	}
	if reflect.TypeOf(r.delegate) != reflect.TypeOf(oldDelegate) || r.delegate.ShouldRebuild(oldDelegate) {
		r.handleDelegateNotification()
	}
}

// MainAxis is the axis of the scrolling that the children are ordered by.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.mainAxis
func (r *RenderTwoDimensionalViewport) MainAxis() painting.Axis { return r.mainAxis }

// SetMainAxis changes the main axis and marks the viewport as needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.mainAxis
func (r *RenderTwoDimensionalViewport) SetMainAxis(value painting.Axis) {
	if r.mainAxis == value {
		return
	}
	r.mainAxis = value
	// Child order needs to be resorted, which happens in PerformLayout.
	r.MarkNeedsLayout()
}

// ScrollCacheExtent is how far beyond the viewport children are built.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.scrollCacheExtent
func (r *RenderTwoDimensionalViewport) ScrollCacheExtent() ScrollCacheExtent {
	return r.scrollCacheExtent
}

// SetScrollCacheExtent changes the cache extent and marks the viewport as
// needing layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.scrollCacheExtent
func (r *RenderTwoDimensionalViewport) SetScrollCacheExtent(value ScrollCacheExtent) {
	if r.scrollCacheExtent == value {
		return
	}
	r.scrollCacheExtent = value
	r.MarkNeedsLayout()
}

// ClipBehavior is how the viewport clips its children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.clipBehavior
func (r *RenderTwoDimensionalViewport) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes the clip behavior.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.clipBehavior
func (r *RenderTwoDimensionalViewport) SetClipBehavior(value ui.Clip) {
	if r.clipBehavior == value {
		return
	}
	r.clipBehavior = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// IsRepaintBoundary is true.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.isRepaintBoundary
func (r *RenderTwoDimensionalViewport) IsRepaintBoundary() bool { return true }

// SizedByParent is true.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.sizedByParent
func (r *RenderTwoDimensionalViewport) SizedByParent() bool { return true }

// FirstChild is the first child in paint order, as of the last layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.firstChild
func (r *RenderTwoDimensionalViewport) FirstChild() Box { return r.firstChild }

// LastChild is the last child in paint order, as of the last layout.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.lastChild
func (r *RenderTwoDimensionalViewport) LastChild() Box { return r.lastChild }

// ChildBefore is the child before the given one in paint order.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.childBefore
func (r *RenderTwoDimensionalViewport) ChildBefore(child Box) Box {
	return r.ParentDataOf(child).previousSibling
}

// ChildAfter is the child after the given one in paint order.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.childAfter
func (r *RenderTwoDimensionalViewport) ChildAfter(child Box) Box {
	return r.ParentDataOf(child).nextSibling
}

func (r *RenderTwoDimensionalViewport) handleDelegateNotification() {
	r.MarkNeedsLayoutWithDelegateRebuild()
}

// SetupParentData gives the child a TwoDimensionalViewportParentData if it has
// none.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.setupParentData
func (r *RenderTwoDimensionalViewport) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*TwoDimensionalViewportParentData); !ok {
		child.AsRenderObject().SetParentData(&TwoDimensionalViewportParentData{})
	}
}

// ParentDataOf is the parent data of the child, which must be one of the children
// of the viewport.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.parentDataOf
func (r *RenderTwoDimensionalViewport) ParentDataOf(child Box) *TwoDimensionalViewportParentData {
	if foundation.DebugMode {
		known := r.children.containsValue(child) || r.keepAliveBucket.containsValue(child)
		for _, o := range r.debugOrphans {
			if o == child {
				known = true
			}
		}
		if !known {
			panic(errors.New(tagTwoDimensionalViewportChildUnknown, "the child is not a child of the viewport",
				"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	return child.ParentData().(*TwoDimensionalViewportParentData)
}

// GetChildFor is the child at the vicinity, or nil.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.getChildFor
func (r *RenderTwoDimensionalViewport) GetChildFor(vicinity ChildVicinity) Box {
	b, _ := r.children.get(vicinity)
	return b
}

// Attach listens to the offsets and the delegate, and attaches the children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.attach
func (r *RenderTwoDimensionalViewport) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.horizontalHandle = r.horizontalOffset.AddListener(r.MarkNeedsLayout)
	r.verticalHandle = r.verticalOffset.AddListener(r.MarkNeedsLayout)
	r.delegateHandle = r.delegate.AddListener(r.handleDelegateNotification)
	for _, child := range r.children.values() {
		child.Attach(owner)
	}
	for _, child := range r.keepAliveBucket.values() {
		child.Attach(owner)
	}
}

// Detach stops listening and detaches the children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.detach
func (r *RenderTwoDimensionalViewport) Detach() {
	r.RenderBox.Detach()
	r.horizontalOffset.RemoveListener(r.horizontalHandle)
	r.verticalOffset.RemoveListener(r.verticalHandle)
	r.delegate.RemoveListener(r.delegateHandle)
	for _, child := range r.children.values() {
		child.Detach()
	}
	for _, child := range r.keepAliveBucket.values() {
		child.Detach()
	}
}

// RedepthChildren updates the depth of the children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.redepthChildren
func (r *RenderTwoDimensionalViewport) RedepthChildren() {
	for _, child := range r.children.values() {
		child.RedepthChildren()
	}
	for _, child := range r.keepAliveBucket.values() {
		r.RedepthChild(child)
	}
}

// VisitChildren visits the children in paint order, then the kept alive ones.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.visitChildren
func (r *RenderTwoDimensionalViewport) VisitChildren(visitor RenderObjectVisitor) {
	for child := r.firstChild; !isNil(child); child = r.ParentDataOf(child).nextSibling {
		visitor(child)
	}
	for _, child := range r.keepAliveBucket.values() {
		visitor(child)
	}
}

// VisitChildrenForSemantics visits the children in paint order, not the kept
// alive ones.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.visitChildrenForSemantics
func (r *RenderTwoDimensionalViewport) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	for child := r.firstChild; !isNil(child); child = r.ParentDataOf(child).nextSibling {
		visitor(child)
	}
}

// DebugDescribeChildren describes the children by their vicinity.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.debugDescribeChildren
func (r *RenderTwoDimensionalViewport) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var out []foundation.DiagnosticsNode
	for _, v := range r.children.order {
		out = append(out, foundation.ToDiagnosticsNode(r.children.items[v], foundation.DiagnosticsNodeOpts{Name: foundation.Some(v.String())}))
	}
	return out
}

// ComputeDryLayout is the biggest size. In debug builds it panics with the tag
// BoxConstraintsUnbounded when an axis is unbounded.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeDryLayout
func (r *RenderTwoDimensionalViewport) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		DebugCheckHasBoundedAxis(painting.AxisVertical, constraints)
		DebugCheckHasBoundedAxis(painting.AxisHorizontal, constraints)
	}
	return constraints.Biggest()
}

// HitTestChildren hit tests the visible children.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.hitTestChildren
func (r *RenderTwoDimensionalViewport) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	for _, child := range r.children.values() {
		data := r.ParentDataOf(child)
		if !data.IsVisible() {
			// Can't hit a child that is not visible.
			continue
		}
		c := child
		paintOffset := data.PaintOffset.Value()
		isHit := result.AddWithPaintOffset(&paintOffset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
			return c.HitTest(result, transformed)
		})
		if isHit {
			return true
		}
	}
	return false
}

// ViewportDimension is the size of the viewport.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.viewportDimension
func (r *RenderTwoDimensionalViewport) ViewportDimension() ui.Size { return r.Size() }

// PerformResize sizes the viewport and gives the offsets their viewport
// dimension.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.performResize
func (r *RenderTwoDimensionalViewport) PerformResize() {
	var oldSize foundation.Opt[ui.Size]
	if r.HasSize() {
		oldSize = foundation.Some(r.Size())
	}
	r.RenderBox.PerformResize()
	// Ignoring the return value since there is a layout either way (PerformLayout
	// is next).
	r.horizontalOffset.ApplyViewportDimension(r.Size().Width)
	r.verticalOffset.ApplyViewportDimension(r.Size().Height)
	if old, ok := oldSize.Get(); !ok || old != r.Size() {
		// Specs can depend on viewport size.
		r.didResize = true
	}
}

// GetOffsetToReveal is the offset along the axis that puts the target at the
// alignment in the viewport, and where the target then is.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.getOffsetToReveal
func (r *RenderTwoDimensionalViewport) GetOffsetToReveal(target RenderObject, alignment float64, opts ...GetOffsetToRevealOpts) RevealedOffset {
	o := firstOpts(opts)
	// If an axis has not been specified, use the main axis.
	axis := o.Axis
	if axis == painting.AxisUnspecified {
		axis = r.mainAxis
	}
	var offset float64
	var axisDirection painting.AxisDirection
	if axis == painting.AxisVertical {
		offset, axisDirection = r.verticalOffset.Pixels(), r.verticalAxisDirection
	} else {
		offset, axisDirection = r.horizontalOffset.Pixels(), r.horizontalAxisDirection
	}
	rect := o.Rect.OrElse(target.PaintBounds())
	// child will be the last render object before the viewport when walking up
	// from target.
	child := target
	for child.Parent() != RenderObject(r.self) {
		child = child.Parent()
	}
	box := child.(Box)
	rectLocal := target.GetTransformTo(child).TransformRect(rect)
	leadingScrollOffset := offset
	// The scroll offset of rect within child.
	switch axisDirection {
	case painting.AxisDirectionUp:
		leadingScrollOffset += box.Size().Height - rectLocal.Bottom
	case painting.AxisDirectionLeft:
		leadingScrollOffset += box.Size().Width - rectLocal.Right
	case painting.AxisDirectionRight:
		leadingScrollOffset += rectLocal.Left
	case painting.AxisDirectionDown:
		leadingScrollOffset += rectLocal.Top
	}
	// The scroll offset in the viewport to rect.
	paintOffset := r.ParentDataOf(box).PaintOffset.Value()
	viewportDimension := r.ViewportDimension()
	switch axisDirection {
	case painting.AxisDirectionUp:
		leadingScrollOffset += viewportDimension.Height - paintOffset.Dy - box.Size().Height
	case painting.AxisDirectionLeft:
		leadingScrollOffset += viewportDimension.Width - paintOffset.Dx - box.Size().Width
	case painting.AxisDirectionRight:
		leadingScrollOffset += paintOffset.Dx
	case painting.AxisDirectionDown:
		leadingScrollOffset += paintOffset.Dy
	}
	// This step assumes the layout of the viewport is up to date, i.e., if the
	// position is changed after the last PerformLayout, the new scroll position
	// will not be accounted for.
	targetRect := target.GetTransformTo(r.self).TransformRect(rect)
	var mainAxisExtentDifference, currentPixels float64
	if axis == painting.AxisHorizontal {
		mainAxisExtentDifference = viewportDimension.Width - rectLocal.Width()
		currentPixels = r.horizontalOffset.Pixels()
	} else {
		mainAxisExtentDifference = viewportDimension.Height - rectLocal.Height()
		currentPixels = r.verticalOffset.Pixels()
	}
	targetOffset := leadingScrollOffset - mainAxisExtentDifference*alignment
	offsetDifference := currentPixels - targetOffset
	switch axisDirection {
	case painting.AxisDirectionUp:
		targetRect = targetRect.Translate(0, -offsetDifference)
	case painting.AxisDirectionDown:
		targetRect = targetRect.Translate(0, offsetDifference)
	case painting.AxisDirectionLeft:
		targetRect = targetRect.Translate(-offsetDifference, 0)
	case painting.AxisDirectionRight:
		targetRect = targetRect.Translate(offsetDifference, 0)
	}
	return RevealedOffset{Offset: targetOffset, Rect: targetRect}
}

// ShowOnScreen scrolls the descendant into view along the axes that allow
// implicit scrolling, then asks the parent to show the rectangle.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.showOnScreen
func (r *RenderTwoDimensionalViewport) ShowOnScreen(opts ShowOnScreenOpts) {
	// It is possible for one and not both axes to allow for implicit scrolling,
	// so the handling is split between the options.
	allowHorizontal := r.horizontalOffset.AllowImplicitScrolling()
	allowVertical := r.verticalOffset.AllowImplicitScrolling()
	var axisDirection foundation.Opt[painting.AxisDirection]
	switch {
	case allowHorizontal && allowVertical:
		// Both allow implicit scrolling.
	case !allowHorizontal && allowVertical:
		axisDirection = foundation.Some(r.verticalAxisDirection)
	case allowHorizontal && !allowVertical:
		axisDirection = foundation.Some(r.horizontalAxisDirection)
	default:
		// Neither axis allows for implicit scrolling.
		r.RenderBox.ShowOnScreen(opts)
		return
	}
	newRect := RenderTwoDimensionalViewportShowInViewport(RenderTwoDimensionalViewportShowInViewportOpts{
		Descendant: opts.Descendant, Viewport: r, AxisDirection: axisDirection, Rect: opts.Rect, Duration: opts.Duration, Curve: opts.Curve,
	})
	r.RenderBox.ShowOnScreen(ShowOnScreenOpts{Rect: newRect, Duration: opts.Duration, Curve: opts.Curve})
}

// RenderTwoDimensionalViewportShowInViewportOpts are the parameters of
// RenderTwoDimensionalViewportShowInViewport.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.showInViewport
type RenderTwoDimensionalViewportShowInViewportOpts struct {
	// Descendant is the object to show.
	Descendant RenderObject
	// Viewport is the viewport to scroll.
	Viewport *RenderTwoDimensionalViewport
	// AxisDirection is the direction to reveal along; none means both axes.
	AxisDirection foundation.Opt[painting.AxisDirection]
	// Rect is the part of the descendant to show, in its coordinates.
	Rect foundation.Opt[ui.Rect]
	// Duration is how long the scrolling takes; zero jumps.
	Duration time.Duration
	// Curve is the curve of the animation.
	Curve animation.Curve
}

// RenderTwoDimensionalViewportShowInViewport scrolls the viewport so that the
// descendant shows, along one axis or both, and returns the rectangle of the
// descendant after that, for the parent to show.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.showInViewport
func RenderTwoDimensionalViewportShowInViewport(o RenderTwoDimensionalViewportShowInViewportOpts) foundation.Opt[ui.Rect] {
	if isNil(o.Descendant) {
		return o.Rect
	}
	showAxis := func(rect foundation.Opt[ui.Rect], axis painting.Axis) foundation.Opt[ui.Rect] {
		return showInViewportForAxis(o.Descendant, rect, o.Viewport, axis, o.Duration, o.Curve)
	}
	if direction, ok := o.AxisDirection.Get(); ok {
		if direction == painting.AxisDirectionLeft || direction == painting.AxisDirectionRight {
			return showAxis(o.Rect, painting.AxisHorizontal)
		}
		return showAxis(o.Rect, painting.AxisVertical)
	}
	// Update rect after revealing in one axis before revealing in the next.
	rect := o.Rect
	if shown := showAxis(rect, painting.AxisHorizontal); shown.IsSome() {
		rect = shown
	}
	// Only the rect after both have been revealed is returned.
	rect = showAxis(rect, painting.AxisVertical)
	if rect.IsNone() {
		// The descendant is between the leading and trailing edge and hence
		// already fully shown on screen.
		transform := o.Descendant.GetTransformTo(o.Viewport.Parent())
		return foundation.Some(transform.TransformRect(o.Descendant.PaintBounds()))
	}
	return rect
}

func showInViewportForAxis(descendant RenderObject, rect foundation.Opt[ui.Rect], viewport *RenderTwoDimensionalViewport,
	axis painting.Axis, duration time.Duration, curve animation.Curve) foundation.Opt[ui.Rect] {
	offset := viewport.horizontalOffset
	if axis == painting.AxisVertical {
		offset = viewport.verticalOffset
	}
	leadingEdgeOffset := viewport.GetOffsetToReveal(descendant, 0, GetOffsetToRevealOpts{Rect: rect, Axis: axis})
	trailingEdgeOffset := viewport.GetOffsetToReveal(descendant, 1, GetOffsetToRevealOpts{Rect: rect, Axis: axis})
	targetOffset, ok := RevealedOffsetClampOffset(leadingEdgeOffset, trailingEdgeOffset, offset.Pixels()).Get()
	if !ok {
		// Already visible in this axis.
		return foundation.None[ui.Rect]()
	}
	if duration == 0 {
		offset.JumpTo(targetOffset.Offset)
	} else {
		offset.AnimateTo(targetOffset.Offset, duration, curve)
	}
	return foundation.Some(targetOffset.Rect)
}

// DidResize is whether the viewport changed size since the last layout, which
// the specs of the children can depend on.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.didResize
func (r *RenderTwoDimensionalViewport) DidResize() bool { return r.didResize }

// NeedsDelegateRebuild is whether the children must be built again by the
// delegate, as opposed to reused.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.needsDelegateRebuild
func (r *RenderTwoDimensionalViewport) NeedsDelegateRebuild() bool { return r.needsDelegateRebuild }

// MarkNeedsLayoutWithDelegateRebuild marks the viewport as needing layout, and
// the children as needing to be built again by the delegate.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.markNeedsLayout
func (r *RenderTwoDimensionalViewport) MarkNeedsLayoutWithDelegateRebuild() {
	r.needsDelegateRebuild = true
	r.MarkNeedsLayout()
}

// PerformLayout lays the children out through LayoutChildSequence, then gives
// the children their paint data in paint order.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.performLayout
func (r *RenderTwoDimensionalViewport) PerformLayout() {
	r.firstChild, r.lastChild = nil, nil
	r.activeChildrenForLayout.clear()
	r.childManager.StartLayout()

	// The subclass lays the children out.
	r.self.LayoutChildSequence()

	if foundation.DebugMode {
		r.debugCheckContentDimensions()
	}
	r.didResize = false
	r.needsDelegateRebuild = false
	r.cacheKeepAlives()
	r.InvokeLayoutCallback(func(Constraints) {
		r.childManager.EndLayout()
		if foundation.DebugMode {
			if len(r.debugOrphans) > 0 {
				panic(errors.New(tagTwoDimensionalViewportOrphans, "children are left that no slot refers to", "object", foundation.DescribeIdentity(r.self)))
			}
			if len(r.debugDanglingKeepAlives) > 0 {
				panic(errors.New(tagTwoDimensionalViewportDanglingKeepAlives, "kept alive children are left that no slot refers to", "object", foundation.DescribeIdentity(r.self)))
			}
			// Ensure nothing is kept alive that should not be any longer.
			for _, child := range r.keepAliveBucket.values() {
				if !r.ParentDataOf(child).KeepAlive {
					panic(errors.New(tagTwoDimensionalViewportKeepAliveStale, "a child is kept alive that does not ask for it", "object", foundation.DescribeIdentity(r.self)))
				}
			}
		}
		// Organize the children in paint order and complete the parent data after
		// the unused children are disposed of by the child manager.
		r.reifyChildren()
	})
}

func (r *RenderTwoDimensionalViewport) cacheKeepAlives() {
	for _, child := range r.children.values() {
		active := false
		for _, a := range r.activeChildrenForLayout.items {
			if a == child {
				active = true
				break
			}
		}
		if active {
			continue
		}
		data := r.ParentDataOf(child)
		if data.KeepAlive {
			r.keepAliveBucket.set(data.Vicinity, child)
			// Let the child manager know that this is kept.
			r.childManager.ReuseChild(data.Vicinity)
		}
	}
}

func (r *RenderTwoDimensionalViewport) sortByYIndex() {
	sort.SliceStable(r.currentChildVicinities, func(i, j int) bool {
		a, b := r.currentChildVicinities[i], r.currentChildVicinities[j]
		if a.YIndex != b.YIndex {
			return a.YIndex < b.YIndex
		}
		return a.XIndex < b.XIndex
	})
}

func (r *RenderTwoDimensionalViewport) sortByXIndex() {
	sort.SliceStable(r.currentChildVicinities, func(i, j int) bool {
		return r.currentChildVicinities[i].CompareTo(r.currentChildVicinities[j]) < 0
	})
}

// reifyChildren ensures all the children have a layout offset, sets the paint
// extent and paint offset, and arranges the children in paint order.
func (r *RenderTwoDimensionalViewport) reifyChildren() {
	var previousChild Box
	if r.mainAxis == painting.AxisVertical {
		// Row major traversal. This seems backwards, but the vertical axis is the
		// typical default axis for scrolling, while row major ordering is the
		// typical default for matrices.
		r.sortByYIndex()
	} else {
		// Column major traversal.
		r.sortByXIndex()
	}
	for _, vicinity := range r.currentChildVicinities {
		if completed := r.completeChildParentData(vicinity, previousChild); !isNil(completed) {
			previousChild = completed
		}
	}
	r.lastChild = previousChild
	if !isNil(r.lastChild) {
		r.ParentDataOf(r.lastChild).nextSibling = nil
	}
	// Reset for the next layout pass.
	r.currentChildVicinities = r.currentChildVicinities[:0]
}

func (r *RenderTwoDimensionalViewport) completeChildParentData(vicinity ChildVicinity, previousChild Box) Box {
	// It is possible and valid for a vicinity to be skipped. For example, a table
	// can have merged cells, spanning multiple indices, but only represented by
	// one render box and vicinity.
	child, ok := r.children.get(vicinity)
	if !ok {
		return nil
	}
	r.UpdateChildPaintData(child)
	if isNil(previousChild) {
		// firstChild is only set once.
		r.firstChild = child
	} else {
		r.ParentDataOf(previousChild).nextSibling = child
		r.ParentDataOf(child).previousSibling = previousChild
	}
	return child
}

type hasContentDimensions interface{ HasContentDimensions() bool }

func (r *RenderTwoDimensionalViewport) debugCheckContentDimensions() {
	const hint = "Subclasses should call ApplyContentDimensions on the verticalOffset and horizontalOffset to set the min and max scroll offset. " +
		"If the contents exceed one or both sides of the viewportDimension, ensure the viewportDimension height or width is subtracted in that axis for the correct extent."
	if p, ok := r.verticalOffset.(hasContentDimensions); ok && !p.HasContentDimensions() {
		panic(errors.New(tagTwoDimensionalViewportContentDimensions, "the verticalOffset was not given content dimensions during LayoutChildSequence", "hint", hint))
	}
	if p, ok := r.horizontalOffset.(hasContentDimensions); ok && !p.HasContentDimensions() {
		panic(errors.New(tagTwoDimensionalViewportContentDimensions, "the horizontalOffset was not given content dimensions during LayoutChildSequence", "hint", hint))
	}
}

// BuildOrObtainChildFor builds the child at the vicinity, or reuses the one that
// exists, and returns it; nil if the delegate has no child there. Call it from
// LayoutChildSequence.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.buildOrObtainChildFor
func (r *RenderTwoDimensionalViewport) BuildOrObtainChildFor(vicinity ChildVicinity) Box {
	if foundation.DebugMode {
		if vicinity == ChildVicinityInvalid {
			panic(errors.New(tagChildVicinityInvalid, "the vicinity is invalid", "object", foundation.DescribeIdentity(r.self)))
		}
		// This should only be called during layout.
		if !r.DebugDoingThisLayout() {
			panic(errors.New(tagTwoDimensionalViewportBuildOutsideLayout, "BuildOrObtainChildFor can only be called during LayoutChildSequence",
				"object", foundation.DescribeIdentity(r.self)))
		}
	}
	if r.needsDelegateRebuild || (!r.children.has(vicinity) && !r.keepAliveBucket.has(vicinity)) {
		r.InvokeLayoutCallback(func(Constraints) { r.childManager.BuildChild(vicinity) })
	} else {
		r.keepAliveBucket.remove(vicinity)
		r.childManager.ReuseChild(vicinity)
	}
	child, ok := r.children.get(vicinity)
	if !ok {
		// There is no child for this vicinity, we may have reached the end of the
		// children in one or both of the x/y indices.
		return nil
	}
	r.activeChildrenForLayout.set(vicinity, child)
	r.ParentDataOf(child).Vicinity = vicinity
	r.currentChildVicinities = append(r.currentChildVicinities, vicinity)
	return child
}

// UpdateChildPaintData sets the paint extent and the paint offset of the child
// from its layout offset. In debug builds it panics with the tag
// TwoDimensionalViewportLayoutOffsetMissing when the child has none.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.updateChildPaintData
func (r *RenderTwoDimensionalViewport) UpdateChildPaintData(child Box) {
	data := r.ParentDataOf(child)
	layoutOffset, ok := data.LayoutOffset.Get()
	if !ok {
		panic(errors.New(tagTwoDimensionalViewportLayoutOffsetMissing,
			"the child with ChildVicinity"+data.Vicinity.String()+" was not provided a layoutOffset; this should be set during LayoutChildSequence, representing the position of the child",
			"object", foundation.DescribeIdentity(r.self)))
	}
	// Set the paint extent (and visibility).
	data.paintExtent = foundation.Some(r.ComputeChildPaintExtent(layoutOffset, child.Size()))
	// Set the paint offset.
	data.PaintOffset = foundation.Some(r.ComputeAbsolutePaintOffsetFor(child, layoutOffset))
	// If the child is partially visible, or not visible at all, there is visual
	// overflow.
	extent := data.paintExtent.Value()
	r.hasVisualOverflow = r.hasVisualOverflow || layoutOffset != (ui.Offset{Dx: extent.Width, Dy: extent.Height}) || !data.IsVisible()
}

// ComputeChildPaintExtent is the part of the child that is in the viewport, when
// the child is at the layout offset; zero if none is.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeChildPaintExtent
func (r *RenderTwoDimensionalViewport) ComputeChildPaintExtent(layoutOffset ui.Offset, childSize ui.Size) ui.Size {
	if childSize == (ui.Size{}) || childSize.Height == 0 || childSize.Width == 0 {
		return ui.Size{}
	}
	viewport := r.ViewportDimension()
	// Horizontal extent.
	var width float64
	switch {
	case layoutOffset.Dx < 0:
		// The child is positioned beyond the leading edge of the viewport.
		if layoutOffset.Dx+childSize.Width <= 0 {
			// The child does not extend into the viewable area, it is not visible.
			return ui.Size{}
		}
		// If the child is positioned starting at -50, then the paint extent is the
		// width + (-50).
		width = layoutOffset.Dx + childSize.Width
	case layoutOffset.Dx >= viewport.Width:
		// The child is positioned after the trailing edge of the viewport, also not
		// visible.
		return ui.Size{}
	default:
		// The child is positioned within the viewport bounds, but may extend
		// beyond it.
		if layoutOffset.Dx+childSize.Width > viewport.Width {
			width = viewport.Width - layoutOffset.Dx
		} else {
			width = childSize.Width
		}
	}
	// Vertical extent.
	var height float64
	switch {
	case layoutOffset.Dy < 0:
		if layoutOffset.Dy+childSize.Height <= 0 {
			return ui.Size{}
		}
		height = layoutOffset.Dy + childSize.Height
	case layoutOffset.Dy >= viewport.Height:
		return ui.Size{}
	default:
		if layoutOffset.Dy+childSize.Height > viewport.Height {
			height = viewport.Height - layoutOffset.Dy
		} else {
			height = childSize.Height
		}
	}
	return ui.Size{Width: width, Height: height}
}

// ComputeAbsolutePaintOffsetFor is the position of the child in the viewport,
// which depends on the axis directions, when the child is at the layout offset.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeAbsolutePaintOffsetFor
func (r *RenderTwoDimensionalViewport) ComputeAbsolutePaintOffsetFor(child Box, layoutOffset ui.Offset) ui.Offset {
	viewport := r.ViewportDimension()
	var xOffset, yOffset float64
	switch r.horizontalAxisDirection {
	case painting.AxisDirectionRight:
		xOffset = layoutOffset.Dx
	case painting.AxisDirectionLeft:
		xOffset = viewport.Width - (layoutOffset.Dx + child.Size().Width)
	default:
		panic(errors.New(tagTwoDimensionalViewportAxisInvalid, "the horizontal axis direction is not horizontal"))
	}
	switch r.verticalAxisDirection {
	case painting.AxisDirectionUp:
		yOffset = viewport.Height - (layoutOffset.Dy + child.Size().Height)
	case painting.AxisDirectionDown:
		yOffset = layoutOffset.Dy
	default:
		panic(errors.New(tagTwoDimensionalViewportAxisInvalid, "the vertical axis direction is not vertical"))
	}
	return ui.Offset{Dx: xOffset, Dy: yOffset}
}

// Paint paints the visible children, clipped if some of them overflow.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.paint
func (r *RenderTwoDimensionalViewport) Paint(context *PaintingContext, offset ui.Offset) {
	if r.children.len() == 0 {
		return
	}
	if r.hasVisualOverflow && r.clipBehavior != ui.ClipNone {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.ViewportDimension()), r.paintChildren,
			PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	} else {
		r.clipRectLayer.SetLayer(nil)
		r.paintChildren(context, offset)
	}
}

func (r *RenderTwoDimensionalViewport) paintChildren(context *PaintingContext, offset ui.Offset) {
	for child := r.firstChild; !isNil(child); {
		data := r.ParentDataOf(child)
		if data.IsVisible() {
			context.PaintChild(child, offset.Add(data.PaintOffset.Value()))
		}
		child = data.nextSibling
	}
}

// InsertChild is called by the element to add a child at the slot.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport._insertChild
func (r *RenderTwoDimensionalViewport) InsertChild(child Box, slot ChildVicinity) {
	if foundation.DebugMode {
		if existing, ok := r.children.get(slot); ok {
			r.debugOrphans = append(r.debugOrphans, existing)
		}
		if r.keepAliveBucket.containsValue(child) {
			panic(errors.New(tagTwoDimensionalViewportInsertKeptAlive, "a child that is kept alive was inserted again", "object", foundation.DescribeIdentity(r.self)))
		}
	}
	r.children.set(slot, child)
	r.AdoptChild(child)
}

// MoveChild is called by the element to move a child between slots.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport._moveChild
func (r *RenderTwoDimensionalViewport) MoveChild(child Box, from, to ChildVicinity) {
	data := r.ParentDataOf(child)
	if !data.KeptAlive() {
		if existing, ok := r.children.get(from); ok && existing == child {
			r.children.remove(from)
		}
		if foundation.DebugMode {
			if existing, ok := r.children.get(to); ok {
				r.debugOrphans = append(r.debugOrphans, existing)
			}
			r.debugOrphans = removeBox(r.debugOrphans, child)
		}
		r.children.set(to, child)
		return
	}
	// If the child in the bucket is not the current child, someone has already
	// moved and replaced the current child, and it can not be removed.
	if existing, ok := r.keepAliveBucket.get(data.Vicinity); ok && existing == child {
		r.keepAliveBucket.remove(data.Vicinity)
	}
	if foundation.DebugMode {
		r.debugDanglingKeepAlives = removeBox(r.debugDanglingKeepAlives, child)
		// If there is an existing child in the new slot, that child will be moved to
		// another index. In other cases the existing child should have been removed
		// by RemoveChild. Thus, it is ok to overwrite it.
		if existing, ok := r.keepAliveBucket.get(data.Vicinity); ok {
			r.debugDanglingKeepAlives = append(r.debugDanglingKeepAlives, existing)
		}
	}
	r.keepAliveBucket.set(data.Vicinity, child)
}

// RemoveChild is called by the element to remove the child at the slot.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport._removeChild
func (r *RenderTwoDimensionalViewport) RemoveChild(child Box, slot ChildVicinity) {
	data := r.ParentDataOf(child)
	if !data.KeptAlive() {
		if existing, ok := r.children.get(slot); ok && existing == child {
			r.children.remove(slot)
		}
		if foundation.DebugMode {
			r.debugOrphans = removeBox(r.debugOrphans, child)
		}
		if existing, ok := r.keepAliveBucket.get(data.Vicinity); ok && existing == child {
			r.keepAliveBucket.remove(data.Vicinity)
		}
		r.DropChild(child)
		return
	}
	if foundation.DebugMode {
		r.debugDanglingKeepAlives = removeBox(r.debugDanglingKeepAlives, child)
	}
	r.keepAliveBucket.remove(data.Vicinity)
	r.DropChild(child)
}

func removeBox(list []Box, child Box) []Box {
	for i, b := range list {
		if b == child {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// ComputeMinIntrinsicWidth is zero: the intrinsic dimensions of a
// two-dimensional viewport would require instantiating every child, which defeats
// the point of a lazy viewport.
//
// Deviation: Flutter throws in debug builds when they are asked for outside of a
// check of the intrinsics; Glutter has no such check (SPEC §12.10).
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeMinIntrinsicWidth
func (r *RenderTwoDimensionalViewport) ComputeMinIntrinsicWidth(float64) float64 { return 0 }

// ComputeMaxIntrinsicWidth is zero, like ComputeMinIntrinsicWidth.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeMaxIntrinsicWidth
func (r *RenderTwoDimensionalViewport) ComputeMaxIntrinsicWidth(float64) float64 { return 0 }

// ComputeMinIntrinsicHeight is zero, like ComputeMinIntrinsicWidth.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeMinIntrinsicHeight
func (r *RenderTwoDimensionalViewport) ComputeMinIntrinsicHeight(float64) float64 { return 0 }

// ComputeMaxIntrinsicHeight is zero, like ComputeMinIntrinsicWidth.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.computeMaxIntrinsicHeight
func (r *RenderTwoDimensionalViewport) ComputeMaxIntrinsicHeight(float64) float64 { return 0 }

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.applyPaintTransform
func (r *RenderTwoDimensionalViewport) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	paintOffset := r.ParentDataOf(child.(Box)).PaintOffset.Value()
	transform.Translate(float32(paintOffset.Dx), float32(paintOffset.Dy), 0)
}

// Dispose lets the clip layer go.
//
// Flutter: widgets/two_dimensional_viewport.dart RenderTwoDimensionalViewport.dispose
func (r *RenderTwoDimensionalViewport) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// AsRenderTwoDimensionalViewport is the base that the concrete viewport embeds.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func (r *RenderTwoDimensionalViewport) AsRenderTwoDimensionalViewport() *RenderTwoDimensionalViewport {
	return r
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagChildVicinityInvalid                      = "ChildVicinityInvalid"
	tagParentDataPaintExtentUnknown              = "ParentDataPaintExtentUnknown"
	tagTwoDimensionalViewportAxisInvalid         = "TwoDimensionalViewportAxisInvalid"
	tagTwoDimensionalViewportBuildOutsideLayout  = "TwoDimensionalViewportBuildOutsideLayout"
	tagTwoDimensionalViewportChildUnknown        = "TwoDimensionalViewportChildUnknown"
	tagTwoDimensionalViewportContentDimensions   = "TwoDimensionalViewportContentDimensions"
	tagTwoDimensionalViewportDanglingKeepAlives  = "TwoDimensionalViewportDanglingKeepAlives"
	tagTwoDimensionalViewportInsertKeptAlive     = "TwoDimensionalViewportInsertKeptAlive"
	tagTwoDimensionalViewportKeepAliveStale      = "TwoDimensionalViewportKeepAliveStale"
	tagTwoDimensionalViewportLayoutOffsetMissing = "TwoDimensionalViewportLayoutOffsetMissing"
	tagTwoDimensionalViewportOrphans             = "TwoDimensionalViewportOrphans"
)
