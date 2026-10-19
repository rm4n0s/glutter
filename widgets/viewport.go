package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// ViewportOpts are the optional parameters of NewViewport.
//
// Flutter: widgets/viewport.dart Viewport.new
type ViewportOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AxisDirection is the direction in which the offset increases; the default
	// is AxisDirectionDown.
	AxisDirection painting.AxisDirection
	// CrossAxisDirection is the direction of the cross axis; the default follows
	// the text direction (down or up) or is down (left or right).
	CrossAxisDirection painting.AxisDirection
	// Anchor is the relative position of the zero scroll offset, from 0 (the
	// leading edge) to 1 (the trailing edge).
	Anchor float64
	// Center is the key of the sliver at the zero scroll offset; the default is
	// the first sliver.
	Center foundation.Key
	// ScrollCacheExtent is how far beyond the viewport the slivers build; the
	// default is 250 pixels.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// PaintOrder is the order in which the slivers paint; the default is the
	// first sliver on top.
	PaintOrder rendering.SliverPaintOrder
	// ClipBehavior is how the viewport clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// Viewport is the widget of a viewport that shows part of a list of slivers,
// the part that its offset (the scroll position) says. Its offset is usually the
// position of a Scrollable.
//
// Deviation: Flutter's has the deprecated cacheExtent and cacheExtentStyle; only
// ScrollCacheExtent is ported (SPEC §12.10).
//
// Flutter: widgets/viewport.dart Viewport
type Viewport struct {
	MultiChildRenderObjectWidget
	axisDirection      painting.AxisDirection
	crossAxisDirection painting.AxisDirection
	anchor             float64
	offset             rendering.ViewportOffset
	center             foundation.Key
	scrollCacheExtent  foundation.Opt[rendering.ScrollCacheExtent]
	paintOrder         rendering.SliverPaintOrder
	clipBehavior       ui.Clip
}

// NewViewport creates a viewport that shows the slivers at the offset. In debug
// builds it panics with the tag ViewportCenterInvalid when the center key is not
// the key of exactly one of the slivers.
//
// Flutter: widgets/viewport.dart Viewport.new
func NewViewport(offset rendering.ViewportOffset, slivers []Widget, opts ...ViewportOpts) *Viewport {
	o := firstOpts(opts)
	if o.AxisDirection == painting.AxisDirectionUnspecified {
		o.AxisDirection = painting.AxisDirectionDown
	}
	if o.PaintOrder == rendering.SliverPaintOrderUnspecified {
		o.PaintOrder = rendering.SliverPaintOrderFirstIsTop
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.Center != nil && foundation.DebugMode {
		count := 0
		for _, s := range slivers {
			if s.Key() == o.Center {
				count++
			}
		}
		if count != 1 {
			panic(errors.New("ViewportCenterInvalid", "the center key of a viewport must be the key of exactly one of its slivers",
				"center", o.Center.String(), "matches", count))
		}
	}
	return &Viewport{
		MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(slivers, WidgetOpts{Key: o.Key}),
		axisDirection:                o.AxisDirection, crossAxisDirection: o.CrossAxisDirection, anchor: o.Anchor, offset: offset,
		center: o.Center, scrollCacheExtent: o.ScrollCacheExtent, paintOrder: o.PaintOrder, clipBehavior: o.ClipBehavior,
	}
}

// AxisDirection is the direction in which the offset increases.
//
// Flutter: widgets/viewport.dart Viewport.axisDirection
func (w *Viewport) AxisDirection() painting.AxisDirection { return w.axisDirection }

// CrossAxisDirection is the direction of the cross axis, or unspecified for the
// default.
//
// Flutter: widgets/viewport.dart Viewport.crossAxisDirection
func (w *Viewport) CrossAxisDirection() painting.AxisDirection { return w.crossAxisDirection }

// Anchor is the relative position of the zero scroll offset.
//
// Flutter: widgets/viewport.dart Viewport.anchor
func (w *Viewport) Anchor() float64 { return w.anchor }

// Offset is the offset of the viewport.
//
// Flutter: widgets/viewport.dart Viewport.offset
func (w *Viewport) Offset() rendering.ViewportOffset { return w.offset }

// Center is the key of the sliver at the zero scroll offset, or nil.
//
// Flutter: widgets/viewport.dart Viewport.center
func (w *Viewport) Center() foundation.Key { return w.center }

// ScrollCacheExtent is how far beyond the viewport the slivers build.
//
// Flutter: widgets/viewport.dart Viewport.scrollCacheExtent
func (w *Viewport) ScrollCacheExtent() foundation.Opt[rendering.ScrollCacheExtent] {
	return w.scrollCacheExtent
}

// PaintOrder is the order in which the slivers paint.
//
// Flutter: widgets/viewport.dart Viewport.paintOrder
func (w *Viewport) PaintOrder() rendering.SliverPaintOrder { return w.paintOrder }

// ClipBehavior is how the viewport clips.
//
// Flutter: widgets/viewport.dart Viewport.clipBehavior
func (w *Viewport) ClipBehavior() ui.Clip { return w.clipBehavior }

// ViewportGetDefaultCrossAxisDirection is the cross axis direction of a viewport
// with the axis direction: the text direction for a vertical axis, down for a
// horizontal one. It panics with the tag DirectionalityMissing in debug builds
// when it needs the text direction and there is no Directionality.
//
// Flutter: widgets/viewport.dart Viewport.getDefaultCrossAxisDirection
func ViewportGetDefaultCrossAxisDirection(bc BuildContext, axisDirection painting.AxisDirection) painting.AxisDirection {
	switch axisDirection {
	case painting.AxisDirectionUp, painting.AxisDirectionDown:
		return painting.TextDirectionToAxisDirection(DirectionalityOf(bc))
	}
	return painting.AxisDirectionDown
}

func (w *Viewport) crossAxis(bc BuildContext) painting.AxisDirection {
	if w.crossAxisDirection != painting.AxisDirectionUnspecified {
		return w.crossAxisDirection
	}
	return ViewportGetDefaultCrossAxisDirection(bc, w.axisDirection)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/viewport.dart Viewport.createRenderObject
func (w *Viewport) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderViewport(w.crossAxis(bc), w.offset, rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{
			AxisDirection: w.axisDirection, ScrollCacheExtent: w.scrollCacheExtent, PaintOrder: w.paintOrder, ClipBehavior: w.clipBehavior},
		Anchor: w.anchor})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/viewport.dart Viewport.updateRenderObject
func (w *Viewport) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderViewport)
	r.SetAxisDirection(w.axisDirection)
	r.SetCrossAxisDirection(w.crossAxis(bc))
	r.SetAnchor(w.anchor)
	r.SetOffset(w.offset)
	r.SetScrollCacheExtent(w.scrollCacheExtent)
	r.SetPaintOrder(w.paintOrder)
	r.SetClipBehavior(w.clipBehavior)
}

// CreateElement creates the element.
//
// Flutter: widgets/viewport.dart Viewport.createElement
func (w *Viewport) CreateElement() Element {
	e := &viewportElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	DebugChildrenHaveDuplicateKeys(w, w.Children())
	e.forgotten = map[Element]struct{}{}
	return e
}

// viewportElement is the element of a Viewport: it tells the render viewport
// which child is the center, and counts the notifications that bubble through.
//
// Flutter: widgets/viewport.dart _ViewportElement
type viewportElement struct {
	MultiChildRenderObjectElement
	doingMountOrUpdate bool
	centerSlotIndex    int
	hasCenterSlot      bool
}

// OnNotification counts the notification as one that went through a viewport.
//
// Flutter: widgets/scroll_notification.dart ViewportElementMixin.onNotification
func (e *viewportElement) OnNotification(notification Notification) bool {
	return ViewportElementOnNotification(notification)
}

func (e *viewportElement) renderViewport() *rendering.RenderViewport {
	return e.renderObject.(*rendering.RenderViewport)
}

// Mount mounts the children and finds the center.
//
// Flutter: widgets/viewport.dart _ViewportElement.mount
func (e *viewportElement) Mount(parent Element, newSlot any) {
	if foundation.DebugMode && e.doingMountOrUpdate {
		panic(errors.New("ViewportReentered", "the viewport element was mounted while it mounts or updates"))
	}
	e.doingMountOrUpdate = true
	e.MultiChildRenderObjectElement.Mount(parent, newSlot)
	e.updateCenter()
	e.doingMountOrUpdate = false
}

// Update updates the children and finds the center.
//
// Flutter: widgets/viewport.dart _ViewportElement.update
func (e *viewportElement) Update(newWidget Widget) {
	if foundation.DebugMode && e.doingMountOrUpdate {
		panic(errors.New("ViewportReentered", "the viewport element was updated while it mounts or updates"))
	}
	e.doingMountOrUpdate = true
	e.MultiChildRenderObjectElement.Update(newWidget)
	e.updateCenter()
	e.doingMountOrUpdate = false
}

func (e *viewportElement) updateCenter() {
	viewport := e.widget.(*Viewport)
	children := e.Children()
	r := e.renderViewport()
	switch {
	case viewport.center != nil:
		elementIndex := 0
		for _, child := range children {
			if child.AsElement().widget.Key() == viewport.center {
				r.SetCenter(child.RenderObject().(rendering.Sliver))
				break
			}
			elementIndex++
		}
		if foundation.DebugMode && elementIndex >= len(children) {
			panic(errors.New("ViewportCenterInvalid", "the center of the viewport is not among its children", "center", viewport.center.String()))
		}
		e.centerSlotIndex, e.hasCenterSlot = elementIndex, true
	case len(children) > 0:
		r.SetCenter(children[0].RenderObject().(rendering.Sliver))
		e.centerSlotIndex, e.hasCenterSlot = 0, true
	default:
		r.SetCenter(nil)
		e.hasCenterSlot = false
	}
}

// InsertRenderObjectChild inserts the child, and makes it the center if it is in
// the slot of the center.
//
// Flutter: widgets/viewport.dart _ViewportElement.insertRenderObjectChild
func (e *viewportElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.MultiChildRenderObjectElement.InsertRenderObjectChild(child, slot)
	// Once [mount]/[update] are done, the new child might have been inserted
	// at the slot of the center.
	if s, ok := slot.(IndexedSlot); ok && !e.doingMountOrUpdate && e.hasCenterSlot && s.Index == e.centerSlotIndex {
		e.renderViewport().SetCenter(child.(rendering.Sliver))
	}
}

// MoveRenderObjectChild moves the child. In debug builds it panics with the tag
// ViewportReentered when it is not during mount or update.
//
// Flutter: widgets/viewport.dart _ViewportElement.moveRenderObjectChild
func (e *viewportElement) MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any) {
	e.MultiChildRenderObjectElement.MoveRenderObjectChild(child, oldSlot, newSlot)
	if foundation.DebugMode && !e.doingMountOrUpdate {
		panic(errors.New("ViewportReentered", "a child of the viewport moved outside of a mount or update"))
	}
}

// RemoveRenderObjectChild removes the child, and clears the center if it was it.
//
// Flutter: widgets/viewport.dart _ViewportElement.removeRenderObjectChild
func (e *viewportElement) RemoveRenderObjectChild(child rendering.RenderObject, slot any) {
	e.MultiChildRenderObjectElement.RemoveRenderObjectChild(child, slot)
	if !e.doingMountOrUpdate && e.renderViewport().Center() == child.(rendering.Sliver) {
		e.renderViewport().SetCenter(nil)
	}
}

// DebugVisitOnstageChildren visits the children whose slivers are visible.
//
// Flutter: widgets/viewport.dart _ViewportElement.debugVisitOnstageChildren
func (e *viewportElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	for _, child := range e.Children() {
		if child.RenderObject().(rendering.Sliver).Geometry().Visible {
			visitor(child)
		}
	}
}

// ShrinkWrappingViewportOpts are the optional parameters of
// NewShrinkWrappingViewport.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.new
type ShrinkWrappingViewportOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AxisDirection is the direction in which the offset increases; the default
	// is AxisDirectionDown.
	AxisDirection painting.AxisDirection
	// CrossAxisDirection is the direction of the cross axis; see Viewport.
	CrossAxisDirection painting.AxisDirection
	// PaintOrder is the order in which the slivers paint; the default is the
	// first sliver on top.
	PaintOrder rendering.SliverPaintOrder
	// ClipBehavior is how the viewport clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// ScrollCacheExtent is how far beyond the viewport the slivers build.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
}

// ShrinkWrappingViewport is a viewport whose extent along the main axis is the
// extent of its content, up to the constraints. It is slower than Viewport,
// because it has to lay out all the content to know its size.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport
type ShrinkWrappingViewport struct {
	MultiChildRenderObjectWidget
	axisDirection      painting.AxisDirection
	crossAxisDirection painting.AxisDirection
	offset             rendering.ViewportOffset
	paintOrder         rendering.SliverPaintOrder
	clipBehavior       ui.Clip
	scrollCacheExtent  foundation.Opt[rendering.ScrollCacheExtent]
}

// NewShrinkWrappingViewport creates a viewport that shows the slivers at the
// offset and shrinks to them.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.new
func NewShrinkWrappingViewport(offset rendering.ViewportOffset, slivers []Widget, opts ...ShrinkWrappingViewportOpts) *ShrinkWrappingViewport {
	o := firstOpts(opts)
	if o.AxisDirection == painting.AxisDirectionUnspecified {
		o.AxisDirection = painting.AxisDirectionDown
	}
	if o.PaintOrder == rendering.SliverPaintOrderUnspecified {
		o.PaintOrder = rendering.SliverPaintOrderFirstIsTop
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &ShrinkWrappingViewport{
		MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(slivers, WidgetOpts{Key: o.Key}),
		axisDirection:                o.AxisDirection, crossAxisDirection: o.CrossAxisDirection, offset: offset, paintOrder: o.PaintOrder,
		clipBehavior: o.ClipBehavior, scrollCacheExtent: o.ScrollCacheExtent,
	}
}

// AxisDirection is the direction in which the offset increases.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.axisDirection
func (w *ShrinkWrappingViewport) AxisDirection() painting.AxisDirection { return w.axisDirection }

// Offset is the offset of the viewport.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.offset
func (w *ShrinkWrappingViewport) Offset() rendering.ViewportOffset { return w.offset }

func (w *ShrinkWrappingViewport) crossAxis(bc BuildContext) painting.AxisDirection {
	if w.crossAxisDirection != painting.AxisDirectionUnspecified {
		return w.crossAxisDirection
	}
	return ViewportGetDefaultCrossAxisDirection(bc, w.axisDirection)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.createRenderObject
func (w *ShrinkWrappingViewport) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderShrinkWrappingViewport(w.crossAxis(bc), w.offset, rendering.RenderShrinkWrappingViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{
			AxisDirection: w.axisDirection, ScrollCacheExtent: w.scrollCacheExtent, PaintOrder: w.paintOrder, ClipBehavior: w.clipBehavior}})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/viewport.dart ShrinkWrappingViewport.updateRenderObject
func (w *ShrinkWrappingViewport) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderShrinkWrappingViewport)
	r.SetAxisDirection(w.axisDirection)
	r.SetCrossAxisDirection(w.crossAxis(bc))
	r.SetOffset(w.offset)
	r.SetPaintOrder(w.paintOrder)
	r.SetClipBehavior(w.clipBehavior)
	r.SetScrollCacheExtent(w.scrollCacheExtent)
}
