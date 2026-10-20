package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// FlowPaintingContext is the context in which a FlowDelegate paints its
// children.
//
// Flutter: rendering/flow.dart FlowPaintingContext
type FlowPaintingContext interface {
	// Size is the size of the container in which the children are painted.
	//
	// Flutter: rendering/flow.dart FlowPaintingContext.size
	Size() ui.Size
	// ChildCount is the number of children available to paint.
	//
	// Flutter: rendering/flow.dart FlowPaintingContext.childCount
	ChildCount() int
	// GetChildSize is the size of the i-th child, or None if there is no such
	// child.
	//
	// Flutter: rendering/flow.dart FlowPaintingContext.getChildSize
	GetChildSize(i int) foundation.Opt[ui.Size]
	// PaintChildAt paints the i-th child with the transform (nil for the
	// identity) and the opacity. Each child can be painted only once.
	//
	// Flutter: rendering/flow.dart FlowPaintingContext.paintChild
	PaintChildAt(i int, opts ...FlowPaintChildOpts)
}

// FlowPaintChildOpts are the optional parameters of FlowPaintingContext.PaintChildAt.
//
// Flutter: rendering/flow.dart FlowPaintingContext.paintChild
type FlowPaintChildOpts struct {
	// Transform is the transform to apply to the child; nil is the identity.
	Transform *ui.Matrix4
	// Opacity defaults to 1 when None; an opacity of 0 skips painting the
	// child but still records it as painted.
	Opacity foundation.Opt[float64]
}

// FlowDelegate controls the layout and paint of the children of a RenderFlow.
// Embed FlowDelegateBase for the defaults.
//
// Flutter: rendering/flow.dart FlowDelegate
type FlowDelegate interface {
	foundation.Listenable
	// GetSize is the size of the container given the constraints.
	//
	// Flutter: rendering/flow.dart FlowDelegate.getSize
	GetSize(constraints BoxConstraints) ui.Size
	// GetConstraintsForChild are the constraints for the i-th child.
	//
	// Flutter: rendering/flow.dart FlowDelegate.getConstraintsForChild
	GetConstraintsForChild(i int, constraints BoxConstraints) BoxConstraints
	// PaintChildren paints the children by calling context.PaintChildAt.
	//
	// Flutter: rendering/flow.dart FlowDelegate.paintChildren
	PaintChildren(context FlowPaintingContext)
	// ShouldRelayout says whether the layout of a flow with the old delegate is
	// out of date.
	//
	// Flutter: rendering/flow.dart FlowDelegate.shouldRelayout
	ShouldRelayout(oldDelegate FlowDelegate) bool
	// ShouldRepaint says whether the painting of a flow with the old delegate is
	// out of date.
	//
	// Flutter: rendering/flow.dart FlowDelegate.shouldRepaint
	ShouldRepaint(oldDelegate FlowDelegate) bool
}

// FlowDelegateBase has the listenable and the default methods of a FlowDelegate.
//
// Flutter: rendering/flow.dart FlowDelegate
type FlowDelegateBase struct {
	repaint foundation.Listenable
}

// NewFlowDelegateBase creates the base; the flow repaints when the repaint
// listenable (nil for none) notifies.
//
// Flutter: rendering/flow.dart FlowDelegate.new
func NewFlowDelegateBase(repaint foundation.Listenable) FlowDelegateBase {
	return FlowDelegateBase{repaint: repaint}
}

// AddListener adds the listener to the repaint listenable.
//
// Flutter: rendering/flow.dart FlowDelegate._repaint
func (d FlowDelegateBase) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if d.repaint == nil {
		return foundation.ListenerHandle{}
	}
	return d.repaint.AddListener(listener)
}

// RemoveListener removes the listener from the repaint listenable.
//
// Flutter: rendering/flow.dart FlowDelegate._repaint
func (d FlowDelegateBase) RemoveListener(handle foundation.ListenerHandle) {
	if d.repaint != nil {
		d.repaint.RemoveListener(handle)
	}
}

// GetSize is the biggest size the constraints allow.
//
// Flutter: rendering/flow.dart FlowDelegate.getSize
func (FlowDelegateBase) GetSize(constraints BoxConstraints) ui.Size { return constraints.Biggest() }

// GetConstraintsForChild passes the constraints through.
//
// Flutter: rendering/flow.dart FlowDelegate.getConstraintsForChild
func (FlowDelegateBase) GetConstraintsForChild(_ int, constraints BoxConstraints) BoxConstraints {
	return constraints
}

// ShouldRelayout is false.
//
// Flutter: rendering/flow.dart FlowDelegate.shouldRelayout
func (FlowDelegateBase) ShouldRelayout(FlowDelegate) bool { return false }

// FlowParentData is the parent data of the children of a RenderFlow: the
// transform that the delegate painted the child with.
//
// Flutter: rendering/flow.dart FlowParentData
type FlowParentData struct {
	ContainerBoxParentData[Box]
	transform *ui.Matrix4
}

// RenderFlow implements the flow layout algorithm: a delegate positions the
// children by painting them with transforms.
//
// Flutter: rendering/flow.dart RenderFlow
type RenderFlow struct {
	RenderBox
	ContainerRenderObject[Box]
	delegate        FlowDelegate
	delegateHandle  foundation.ListenerHandle
	clipBehavior    ui.Clip
	clipRectLayer   *LayerHandle[*ClipRectLayer]
	randomAccess    []Box
	lastPaintOrder  []int
	paintingContext *PaintingContext
	paintingOffset  ui.Offset
}

// RenderFlowOpts are the optional parameters of NewRenderFlow.
//
// Flutter: rendering/flow.dart RenderFlow.new
type RenderFlowOpts struct {
	// ClipBehavior defaults to ui.ClipHardEdge.
	ClipBehavior ui.Clip
}

// NewRenderFlow creates a flow with the children (nil for none).
//
// Flutter: rendering/flow.dart RenderFlow.new
func NewRenderFlow(delegate FlowDelegate, children []Box, opts ...RenderFlowOpts) *RenderFlow {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	r := &RenderFlow{delegate: delegate, clipBehavior: o.ClipBehavior, clipRectLayer: NewLayerHandle[*ClipRectLayer](nil)}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// SetupParentData gives the child a FlowParentData, or resets the transform of
// the one it has.
//
// Flutter: rendering/flow.dart RenderFlow.setupParentData
func (r *RenderFlow) SetupParentData(child RenderObject) {
	if d, ok := child.ParentData().(*FlowParentData); ok {
		d.transform = nil
		return
	}
	child.AsRenderObject().SetParentData(&FlowParentData{})
}

func flowParentData(child Box) *FlowParentData { return child.ParentData().(*FlowParentData) }

// Delegate controls the layout and paint of the children.
//
// Flutter: rendering/flow.dart RenderFlow.delegate
func (r *RenderFlow) Delegate() FlowDelegate { return r.delegate }

// SetDelegate changes the delegate. The flow needs layout if the new delegate is
// of another type or says so, and needs paint if it says so.
//
// Flutter: rendering/flow.dart RenderFlow.delegate
func (r *RenderFlow) SetDelegate(newDelegate FlowDelegate) {
	if any(r.delegate) == any(newDelegate) {
		return
	}
	old := r.delegate
	r.delegate = newDelegate
	if foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(old) || newDelegate.ShouldRelayout(old) {
		r.MarkNeedsLayout()
	} else if newDelegate.ShouldRepaint(old) {
		r.MarkNeedsPaint()
	}
	if r.Attached() {
		old.RemoveListener(r.delegateHandle)
		r.delegateHandle = newDelegate.AddListener(r.MarkNeedsPaint)
	}
}

// ClipBehavior is how the content that overflows is clipped.
//
// Flutter: rendering/flow.dart RenderFlow.clipBehavior
func (r *RenderFlow) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior marks the flow as needing paint when the value changes.
//
// Flutter: rendering/flow.dart RenderFlow.clipBehavior
func (r *RenderFlow) SetClipBehavior(value ui.Clip) {
	if value == ui.ClipUnspecified {
		value = ui.ClipHardEdge
	}
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// Attach starts listening to the delegate's repaint listenable.
//
// Flutter: rendering/flow.dart RenderFlow.attach
func (r *RenderFlow) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.delegateHandle = r.delegate.AddListener(r.MarkNeedsPaint)
}

// Detach stops listening to the delegate's repaint listenable.
//
// Flutter: rendering/flow.dart RenderFlow.detach
func (r *RenderFlow) Detach() {
	r.delegate.RemoveListener(r.delegateHandle)
	r.delegateHandle = foundation.ListenerHandle{}
	r.RenderBox.Detach()
}

func (r *RenderFlow) getSize(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		constraints.DebugAssertIsValid()
	}
	return constraints.Constrain(r.delegate.GetSize(constraints))
}

// IsRepaintBoundary is true.
//
// Flutter: rendering/flow.dart RenderFlow.isRepaintBoundary
func (r *RenderFlow) IsRepaintBoundary() bool { return true }

// ComputeMinIntrinsicWidth is the delegate's width for the height, or 0 if it
// is not finite.
//
// Flutter: rendering/flow.dart RenderFlow.computeMinIntrinsicWidth
func (r *RenderFlow) ComputeMinIntrinsicWidth(height float64) float64 {
	return finiteOrZero(r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Height: foundation.Some(height)})).Width)
}

// ComputeMaxIntrinsicWidth is the delegate's width for the height, or 0 if it
// is not finite.
//
// Flutter: rendering/flow.dart RenderFlow.computeMaxIntrinsicWidth
func (r *RenderFlow) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.ComputeMinIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the delegate's height for the width, or 0 if it
// is not finite.
//
// Flutter: rendering/flow.dart RenderFlow.computeMinIntrinsicHeight
func (r *RenderFlow) ComputeMinIntrinsicHeight(width float64) float64 {
	return finiteOrZero(r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)})).Height)
}

// ComputeMaxIntrinsicHeight is the delegate's height for the width, or 0 if it
// is not finite.
//
// Flutter: rendering/flow.dart RenderFlow.computeMaxIntrinsicHeight
func (r *RenderFlow) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.ComputeMinIntrinsicHeight(width)
}

// ComputeDryLayout is the size the delegate gives.
//
// Flutter: rendering/flow.dart RenderFlow.computeDryLayout
func (r *RenderFlow) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.getSize(constraints)
}

// PerformLayout sizes the flow and lays out the children with the constraints
// the delegate gives.
//
// Flutter: rendering/flow.dart RenderFlow.performLayout
func (r *RenderFlow) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(r.getSize(constraints))
	r.randomAccess = r.randomAccess[:0]
	i := 0
	for child := r.FirstChild(); !isNil(child); {
		r.randomAccess = append(r.randomAccess, child)
		child.Layout(r.delegate.GetConstraintsForChild(i, constraints), LayoutOpts{ParentUsesSize: true})
		data := flowParentData(child)
		data.Offset = ui.OffsetZero
		child = data.NextSibling
		i++
	}
}

// GetChildSize is the size of the i-th child, or None.
//
// Flutter: rendering/flow.dart RenderFlow.getChildSize
func (r *RenderFlow) GetChildSize(i int) foundation.Opt[ui.Size] {
	if i < 0 || i >= len(r.randomAccess) {
		return foundation.None[ui.Size]()
	}
	return foundation.Some(r.randomAccess[i].Size())
}

// PaintChildAt paints the i-th child with the transform and the opacity. In
// debug builds it panics with FlowChildPaintedTwice if the child was painted
// already during this paint.
//
// Flutter: rendering/flow.dart RenderFlow.paintChild
func (r *RenderFlow) PaintChildAt(i int, opts ...FlowPaintChildOpts) {
	o := firstOpts(opts)
	transform := o.Transform
	if transform == nil {
		transform = ui.NewMatrix4Identity()
	}
	opacity := o.Opacity.OrElse(1)
	child := r.randomAccess[i]
	data := flowParentData(child)
	if foundation.DebugMode && data.transform != nil {
		panic(errors.New("FlowChildPaintedTwice", "Cannot call paintChild twice for the same child. The flow delegate of type "+
			foundation.RuntimeTypeName(r.delegate)+" attempted to paint child "+itoa(i)+" multiple times, which is not permitted.",
			"object", foundation.DescribeIdentity(r), "child", i))
	}
	r.lastPaintOrder = append(r.lastPaintOrder, i)
	data.transform = transform
	if opacity == 0 {
		return
	}
	painter := func(context *PaintingContext, offset ui.Offset) { context.PaintChild(child, offset) }
	if opacity == 1 {
		r.paintingContext.PushTransform(r.NeedsCompositing(), r.paintingOffset, transform, painter, nil)
		return
	}
	r.paintingContext.PushOpacity(r.paintingOffset, ui.ColorGetAlphaFromOpacity(opacity), func(context *PaintingContext, offset ui.Offset) {
		context.PushTransform(r.NeedsCompositing(), offset, transform, painter, nil)
	}, nil)
}

func (r *RenderFlow) paintWithDelegate(context *PaintingContext, offset ui.Offset) {
	r.lastPaintOrder = r.lastPaintOrder[:0]
	r.paintingContext, r.paintingOffset = context, offset
	for _, child := range r.randomAccess {
		flowParentData(child).transform = nil
	}
	defer func() { r.paintingContext, r.paintingOffset = nil, ui.OffsetZero }()
	r.delegate.PaintChildren(r)
}

// Paint lets the delegate paint the children, clipped to the flow.
//
// Flutter: rendering/flow.dart RenderFlow.paint
func (r *RenderFlow) Paint(context *PaintingContext, offset ui.Offset) {
	r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
		r.paintWithDelegate, PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/flow.dart RenderFlow.dispose
func (r *RenderFlow) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// HitTestChildren hit tests the children that were painted, last painted first,
// through their paint transforms.
//
// Flutter: rendering/flow.dart RenderFlow.hitTestChildren
func (r *RenderFlow) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	children := GetChildrenAsList(&r.ContainerRenderObject)
	for i := len(r.lastPaintOrder) - 1; i >= 0; i-- {
		index := r.lastPaintOrder[i]
		if index >= len(children) {
			continue
		}
		child := children[index]
		transform := flowParentData(child).transform
		if transform == nil {
			continue
		}
		if result.AddWithPaintTransform(transform, position, func(result *BoxHitTestResult, position ui.Offset) bool {
			return child.HitTest(result, position)
		}) {
			return true
		}
	}
	return false
}

// ApplyPaintTransform applies the transform the child was painted with.
//
// Flutter: rendering/flow.dart RenderFlow.applyPaintTransform
func (r *RenderFlow) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if t := flowParentData(child.(Box)).transform; t != nil {
		transform.MultiplyBy(t)
	}
	r.RenderBox.ApplyPaintTransform(child, transform)
}
