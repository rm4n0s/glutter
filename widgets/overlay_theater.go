package widgets

import (
	"iter"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// theater is the widget of the render object that paints and lays out the entries
// of an overlay.
//
// Flutter: widgets/overlay.dart _Theater
type theater struct {
	MultiChildRenderObjectWidget
	skipCount           int
	clipBehavior        ui.Clip
	alwaysSizeToContent bool
}

func newTheater(skipCount int, clipBehavior ui.Clip, alwaysSizeToContent bool, children []Widget) *theater {
	return &theater{NewMultiChildRenderObjectWidgetBase(children), skipCount, clipBehavior, alwaysSizeToContent}
}

func (w *theater) CreateElement() Element {
	e := &theaterElement{}
	InitMultiChildRenderObjectElement(&e.MultiChildRenderObjectElement, e, w)
	return e
}

func (w *theater) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := &renderTheater{skipCount: w.skipCount, textDirection: DirectionalityOf(bc), clipBehavior: w.clipBehavior,
		alwaysSizeToContent: w.alwaysSizeToContent, clipRectLayer: rendering.NewLayerHandle[*rendering.ClipRectLayer](nil),
		entries: map[rendering.Box]*OverlayEntry{}}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func (w *theater) UpdateRenderObject(bc BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderTheater)
	r.setSkipCount(w.skipCount)
	r.setTextDirection(DirectionalityOf(bc))
	r.setClipBehavior(w.clipBehavior)
	r.setAlwaysSizeToContent(w.alwaysSizeToContent)
}

// theaterElement is the element of a theater: it tells the render object which
// overlay entry made each child.
//
// Flutter: widgets/overlay.dart _TheaterElement
type theaterElement struct {
	MultiChildRenderObjectElement
}

func (e *theaterElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.MultiChildRenderObjectElement.InsertRenderObjectChild(child, slot)
	entryWidget := e.widget.(*theater).Children()[slot.(IndexedSlot).Index].(*overlayEntryWidget)
	e.renderObject.(*renderTheater).entries[child.(rendering.Box)] = entryWidget.entry
}

// DebugVisitOnstageChildren visits the children that are on stage.
//
// Flutter: widgets/overlay.dart _TheaterElement.debugVisitOnstageChildren
func (e *theaterElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	children := e.Children()
	skip := e.widget.(*theater).skipCount
	if foundation.DebugMode && len(children) < skip {
		panic(errors.New("TheaterSkipCountInvalid", "the skip count is greater than the number of children"))
	}
	for _, c := range children[skip:] {
		visitor(c)
	}
}

// theaterHost is what the theater layout code needs from a render object that
// lays out and paints its children like the theater does: the theater itself, a
// deferred layout box and the layout builder of an overlay child.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin
type theaterHost interface {
	rendering.Box
	// theaterOf is the theater that the object lays out for.
	theaterOf() *renderTheater
	// childrenInPaintOrder are the children from the farthest to the closest to
	// the user. The sequence allows the children to change while it is read.
	childrenInPaintOrder() iter.Seq[rendering.Box]
	// childrenInHitTestOrder are the children from the closest to the farthest to
	// the user. The sequence is lazy, as hit testing stops at the first hit.
	childrenInHitTestOrder() iter.Seq[rendering.Box]
}

func theaterStackData(child rendering.Box) *rendering.StackParentData {
	return child.ParentData().(*rendering.StackParentData)
}

// theaterSetupParentData gives the child StackParentData.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.setupParentData
func theaterSetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*rendering.StackParentData); !ok {
		child.AsRenderObject().SetParentData(&rendering.StackParentData{})
	}
}

// theaterDistanceToActualBaseline is the topmost baseline of the children.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.computeDistanceToActualBaseline
func theaterDistanceToActualBaseline(host theaterHost, baseline ui.TextBaseline) foundation.Opt[float64] {
	baselineOffset := rendering.NoBaseline
	for child := range host.childrenInPaintOrder() {
		baselineOffset = baselineOffset.MinOf(rendering.NewBaselineOffset(child.GetDistanceToActualBaseline(baseline)).Add(theaterStackData(child).Offset.Dy))
	}
	return baselineOffset.Offset
}

// theaterLayoutChild lays out a child with the constraints that a non-positioned
// child gets, or as a positioned child of the stack algorithm.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.layoutChild
func theaterLayoutChild(host theaterHost, child rendering.Box, nonPositionedChildConstraints rendering.BoxConstraints) {
	data := theaterStackData(child)
	alignment := host.theaterOf().resolvedAlignment()
	if !data.IsPositioned() {
		child.Layout(nonPositionedChildConstraints, rendering.LayoutOpts{ParentUsesSize: true})
		data.Offset = ui.Offset{}
		return
	}
	if _, deferred := child.(*renderDeferredLayoutBox); deferred && foundation.DebugMode {
		panic(errors.New("OverlayDeferredChildPositioned", "all deferred layout boxes of an OverlayPortal must be non-positioned children"))
	}
	rendering.LayoutPositionedStackChild(child, data, host.Size(), alignment)
}

// theaterHitTestChildren hits the children in hit test order.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.hitTestChildren
func theaterHitTestChildren(host theaterHost, result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := range host.childrenInHitTestOrder() {
		offset := theaterStackData(child).Offset
		c := child
		if result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
			return c.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}

// theaterPaint paints the children in paint order.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.paint
func theaterPaint(host theaterHost, context *rendering.PaintingContext, offset ui.Offset) {
	for child := range host.childrenInPaintOrder() {
		context.PaintChild(child, theaterStackData(child).Offset.Add(offset))
	}
}

// renderTheater is a box that sizes itself to its parent (or to the entry that
// can size the overlay), implements the stack layout algorithm and renders its
// children: the entries that are on stage after the first skipCount ones, and the
// overlay children of the OverlayPortals that are in them.
//
// Flutter: widgets/overlay.dart _RenderTheater
type renderTheater struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	textDirection       ui.TextDirection
	skipCount           int
	clipBehavior        ui.Clip
	alwaysSizeToContent bool
	alignmentCache      foundation.Opt[painting.Alignment]
	clipRectLayer       *rendering.LayerHandle[*rendering.ClipRectLayer]
	// entries is the overlay entry that created each child.
	entries map[rendering.Box]*OverlayEntry
	// outstandingDeferredChildUpdateCalls, when positive, makes MarkNeedsLayout do
	// nothing: adding or removing a deferred child does not change the layout of
	// the other children or of the overlay.
	outstandingDeferredChildUpdateCalls int
	layingOutSizeDeterminingChild       bool
}

func (r *renderTheater) theaterOf() *renderTheater { return r }

func (r *renderTheater) SetupParentData(child rendering.RenderObject) { theaterSetupParentData(child) }

// portalChildren are the overlay children of the OverlayPortals in the entry that
// made the child, in paint order, or the reverse.
//
// Flutter: widgets/overlay.dart _TheaterParentData.paintOrderIterator
func (r *renderTheater) portalChildren(child rendering.Box, reversed bool) iter.Seq[*renderDeferredLayoutBox] {
	entry := r.entries[child]
	if entry == nil || entry.notifier == nil {
		return func(func(*renderDeferredLayoutBox) bool) {}
	}
	state := entry.notifier.Value()
	if state == nil {
		return func(func(*renderDeferredLayoutBox) bool) {}
	}
	return state.childIterable(reversed)
}

// Attach attaches the overlay children too, which are not in the child list.
func (r *renderTheater) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	for child := r.FirstChild(); child != nil; child = r.data(child).NextSibling {
		for deferred := range r.portalChildren(child, false) {
			deferred.Attach(owner)
		}
	}
}

// Detach detaches the overlay children too.
func (r *renderTheater) Detach() {
	r.RenderBox.Detach()
	for child := r.FirstChild(); child != nil; child = r.data(child).NextSibling {
		for deferred := range r.portalChildren(child, false) {
			deferred.Detach()
		}
	}
}

func (r *renderTheater) RedepthChildren() {
	r.VisitChildren(func(child rendering.RenderObject) { r.RedepthChild(child) })
}

func (r *renderTheater) resolvedAlignment() painting.Alignment {
	if a, ok := r.alignmentCache.Get(); ok {
		return a
	}
	a := painting.AlignmentDirectionalTopStart.Resolve(r.textDirection)
	r.alignmentCache = foundation.Some(a)
	return a
}

func (r *renderTheater) setTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.alignmentCache = foundation.None[painting.Alignment]()
	r.MarkNeedsLayout()
}

func (r *renderTheater) setSkipCount(value int) {
	if r.skipCount != value {
		r.skipCount = value
		r.MarkNeedsLayout()
	}
}

func (r *renderTheater) setClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *renderTheater) setAlwaysSizeToContent(value bool) {
	if r.alwaysSizeToContent != value {
		r.alwaysSizeToContent = value
		r.MarkNeedsLayout()
	}
}

// addDeferredChild makes the box a child of the theater, which paints and hit
// tests it but does not lay it out.
//
// Flutter: widgets/overlay.dart _RenderTheater._addDeferredChild
func (r *renderTheater) addDeferredChild(child *renderDeferredLayoutBox) {
	r.outstandingDeferredChildUpdateCalls++
	r.AdoptChild(child)
	// The overlay still needs repainting when a deferred child is added. Usually
	// MarkNeedsLayout implies MarkNeedsPaint, but here MarkNeedsLayout is skipped.
	r.MarkNeedsPaint()
	r.outstandingDeferredChildUpdateCalls--
	// After adding the child to the render tree, make sure it is laid out in the
	// same frame: marking the surrogate adds the child to the dirty list (see
	// renderLayoutSurrogateProxyBox.PerformLayout).
	child.layoutSurrogate.MarkNeedsLayout()
}

// removeDeferredChild removes the box from the theater.
//
// Flutter: widgets/overlay.dart _RenderTheater._removeDeferredChild
func (r *renderTheater) removeDeferredChild(child *renderDeferredLayoutBox) {
	r.outstandingDeferredChildUpdateCalls++
	r.DropChild(child)
	r.MarkNeedsPaint()
	r.outstandingDeferredChildUpdateCalls--
}

// MarkNeedsLayout does nothing while a deferred child is added or removed.
func (r *renderTheater) MarkNeedsLayout() {
	if r.outstandingDeferredChildUpdateCalls == 0 {
		r.RenderBox.MarkNeedsLayout()
	}
}

func (r *renderTheater) data(child rendering.Box) *rendering.StackParentData {
	return child.ParentData().(*rendering.StackParentData)
}

func (r *renderTheater) firstOnstageChild() rendering.Box {
	if r.skipCount == r.ChildCount() {
		return nil
	}
	child := r.FirstChild()
	for toSkip := r.skipCount; toSkip > 0; toSkip-- {
		child = r.data(child).NextSibling
	}
	return child
}

func (r *renderTheater) lastOnstageChild() rendering.Box {
	if r.skipCount == r.ChildCount() {
		return nil
	}
	return r.LastChild()
}

// childrenInPaintOrder is the children on stage, each followed by the overlay
// children of the portals in it, from the farthest to the closest to the user.
func (r *renderTheater) childrenInPaintOrder() iter.Seq[rendering.Box] {
	return func(yield func(rendering.Box) bool) {
		for child := r.firstOnstageChild(); child != nil; {
			if !yield(child) {
				return
			}
			for deferred := range r.portalChildren(child, false) {
				if !yield(deferred) {
					return
				}
			}
			child = r.data(child).NextSibling
		}
	}
}

// childrenInHitTestOrder is the children on stage, from the closest to the
// farthest to the user: the overlay children of the portals in an entry come
// before the entry.
func (r *renderTheater) childrenInHitTestOrder() iter.Seq[rendering.Box] {
	return func(yield func(rendering.Box) bool) {
		child := r.lastOnstageChild()
		childLeft := r.ChildCount() - r.skipCount
		for child != nil {
			for deferred := range r.portalChildren(child, true) {
				if !yield(deferred) {
					return
				}
			}
			if !yield(child) {
				return
			}
			previous := r.data(child).PreviousSibling
			childLeft--
			if childLeft <= 0 {
				return
			}
			child = previous
		}
	}
}

func (r *renderTheater) ComputeMinIntrinsicWidth(height float64) float64 {
	return rendering.StackIntrinsicDimension(r.firstOnstageChild(), func(c rendering.Box) float64 { return c.GetMinIntrinsicWidth(height) })
}

func (r *renderTheater) ComputeMaxIntrinsicWidth(height float64) float64 {
	return rendering.StackIntrinsicDimension(r.firstOnstageChild(), func(c rendering.Box) float64 { return c.GetMaxIntrinsicWidth(height) })
}

func (r *renderTheater) ComputeMinIntrinsicHeight(width float64) float64 {
	return rendering.StackIntrinsicDimension(r.firstOnstageChild(), func(c rendering.Box) float64 { return c.GetMinIntrinsicHeight(width) })
}

func (r *renderTheater) ComputeMaxIntrinsicHeight(width float64) float64 {
	return rendering.StackIntrinsicDimension(r.firstOnstageChild(), func(c rendering.Box) float64 { return c.GetMaxIntrinsicHeight(width) })
}

func (r *renderTheater) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return theaterDistanceToActualBaseline(r, baseline)
}

func (r *renderTheater) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	var size ui.Size
	if !r.alwaysSizeToContent && constraints.Biggest().IsFinite() {
		size = constraints.Biggest()
	} else {
		size = r.findSizeDeterminingChild().GetDryLayout(constraints)
	}
	nonPositionedChildConstraints := rendering.NewBoxConstraintsTight(size)
	alignment := r.resolvedAlignment()
	baselineOffset := rendering.NoBaseline
	for child := range r.childrenInPaintOrder() {
		baselineOffset = baselineOffset.MinOf(rendering.NewBaselineOffset(
			rendering.StackBaselineForChild(child, size, nonPositionedChildConstraints, alignment, baseline)))
	}
	return baselineOffset.Offset
}

func (r *renderTheater) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if !r.alwaysSizeToContent && constraints.Biggest().IsFinite() {
		return constraints.Biggest()
	}
	return r.findSizeDeterminingChild().GetDryLayout(constraints)
}

func (r *renderTheater) PerformLayout() {
	constraints := r.BoxConstraints()
	var sizeDeterminingChild rendering.Box
	if !r.alwaysSizeToContent && constraints.Biggest().IsFinite() {
		r.SetSize(constraints.Biggest())
	} else {
		sizeDeterminingChild = r.findSizeDeterminingChild()
		r.layingOutSizeDeterminingChild = true
		theaterLayoutChild(r, sizeDeterminingChild, constraints)
		r.layingOutSizeDeterminingChild = false
		r.SetSize(sizeDeterminingChild.Size())
	}
	// Equivalent to the BoxConstraints used by RenderStack for StackFit.expand.
	nonPositionedChildConstraints := rendering.NewBoxConstraintsTight(r.Size())
	for child := range r.childrenInPaintOrder() {
		if child != sizeDeterminingChild {
			theaterLayoutChild(r, child, nonPositionedChildConstraints)
		}
	}
}

func (r *renderTheater) findSizeDeterminingChild() rendering.Box {
	for child := r.lastOnstageChild(); child != nil; child = r.data(child).PreviousSibling {
		if entry := r.entries[child]; entry != nil && entry.canSizeOverlay && !r.data(child).IsPositioned() {
			return child
		}
	}
	constraints := r.BoxConstraints()
	if r.alwaysSizeToContent {
		panic(errors.New("OverlaySizeToContentInvalid",
			"Overlay was asked to size itself to content but does not have a suitable child; when alwaysSizeToContent is true, the Overlay requires at least one non-positioned OverlayEntry with canSizeOverlay set to true to determine its size; try removing alwaysSizeToContent=true or provide a suitable child that can size the Overlay"))
	}
	panic(errors.New("OverlayUnbounded",
		"Overlay was given infinite constraints and cannot be sized by a suitable child; the constraints given to the overlay ("+constraints.String()+") would result in an illegal infinite size ("+constraints.Biggest().String()+
			"); to avoid that, the Overlay tried to size itself to one of its children, but no suitable non-positioned child that belongs to an OverlayEntry with canSizeOverlay set to true could be found; try wrapping the Overlay in a SizedBox to give it a finite size or use an OverlayEntry with canSizeOverlay set to true"))
}

func (r *renderTheater) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return theaterHitTestChildren(r, result, position)
}

func (r *renderTheater) paintChildren(context *rendering.PaintingContext, offset ui.Offset) {
	theaterPaint(r, context, offset)
}

func (r *renderTheater) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if r.clipBehavior != ui.ClipNone {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), r.paintChildren,
			rendering.PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	} else {
		r.clipRectLayer.SetLayer(nil)
		r.paintChildren(context, offset)
	}
}

func (r *renderTheater) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

func (r *renderTheater) RemoveObject(child rendering.RenderObject) {
	delete(r.entries, child.(rendering.Box))
	r.ContainerRenderObject.RemoveObject(child)
}

// VisitChildren visits each child followed by the overlay children of the portals
// in its entry.
func (r *renderTheater) VisitChildren(visitor rendering.RenderObjectVisitor) {
	for child := r.FirstChild(); child != nil; {
		visitor(child)
		for deferred := range r.portalChildren(child, false) {
			visitor(deferred)
		}
		child = r.data(child).NextSibling
	}
}

func (r *renderTheater) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	for child := r.firstOnstageChild(); child != nil; {
		visitor(child)
		for deferred := range r.portalChildren(child, false) {
			visitor(deferred)
		}
		child = r.data(child).NextSibling
	}
}

func (r *renderTheater) DescribeApproximatePaintClip(rendering.RenderObject) foundation.Opt[ui.Rect] {
	if r.clipBehavior == ui.ClipNone {
		return foundation.None[ui.Rect]()
	}
	return foundation.Some(ui.OffsetZero.And(r.Size()))
}

func (r *renderTheater) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var offstage, onstage []foundation.DiagnosticsNode
	count := 1
	isOnstage := false
	first := r.firstOnstageChild()
	for child := r.FirstChild(); child != nil; child = r.data(child).NextSibling {
		if child == first {
			isOnstage = true
			count = 1
		}
		if isOnstage {
			onstage = append(onstage, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some("onstage " + strconv.Itoa(count))}))
		} else {
			offstage = append(offstage, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{
				Name: foundation.Some("offstage " + strconv.Itoa(count)), Style: foundation.DiagnosticsTreeStyleOffstage}))
		}
		subcount := 1
		for deferred := range r.portalChildren(child, false) {
			name := strconv.Itoa(count) + " - " + strconv.Itoa(subcount)
			if isOnstage {
				onstage = append(onstage, foundation.ToDiagnosticsNode(deferred, foundation.DiagnosticsNodeOpts{Name: foundation.Some("onstage " + name)}))
			} else {
				offstage = append(offstage, foundation.ToDiagnosticsNode(deferred, foundation.DiagnosticsNodeOpts{
					Name: foundation.Some("offstage " + name), Style: foundation.DiagnosticsTreeStyleOffstage}))
			}
			subcount++
		}
		count++
	}
	out := onstage
	if len(offstage) > 0 {
		out = append(out, offstage...)
	} else {
		out = append(out, foundation.NewDiagnosticsNodeMessage("no offstage children", foundation.DiagnosticsNodeMessageOpts{Style: foundation.DiagnosticsTreeStyleOffstage}))
	}
	return out
}
