package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ListBodyParentData is the parent data of the children of a RenderListBody.
//
// Flutter: rendering/list_body.dart ListBodyParentData
type ListBodyParentData struct {
	ContainerBoxParentData[Box]
}

// RenderListBody arranges its children in a line, without clipping or resizing
// them: the main axis is unbounded and the children fill the cross axis.
//
// Flutter: rendering/list_body.dart RenderListBody
type RenderListBody struct {
	RenderBox
	ContainerRenderObject[Box]
	axisDirection painting.AxisDirection
}

// RenderListBodyOpts are the optional parameters of NewRenderListBody.
//
// Flutter: rendering/list_body.dart RenderListBody.new
type RenderListBodyOpts struct {
	// AxisDirection defaults to painting.AxisDirectionDown.
	AxisDirection painting.AxisDirection
}

// NewRenderListBody creates a list body with the children (nil for none).
//
// Flutter: rendering/list_body.dart RenderListBody.new
func NewRenderListBody(children []Box, opts ...RenderListBodyOpts) *RenderListBody {
	o := firstOpts(opts)
	if o.AxisDirection == painting.AxisDirectionUnspecified {
		o.AxisDirection = painting.AxisDirectionDown
	}
	r := &RenderListBody{}
	InitRenderListBody(r, r, o)
	r.AddAll(children)
	return r
}

// InitRenderListBody sets up a RenderListBody that a render object outside
// rendering subclasses: self is the outer object.
//
// Flutter: rendering/list_body.dart RenderListBody.new
func InitRenderListBody(r *RenderListBody, self Box, opts ...RenderListBodyOpts) {
	o := firstOpts(opts)
	if o.AxisDirection == painting.AxisDirectionUnspecified {
		o.AxisDirection = painting.AxisDirectionDown
	}
	r.axisDirection = o.AxisDirection
	InitRenderBox(&r.RenderBox, self)
	InitContainerRenderObject(&r.ContainerRenderObject, self)
}

// SetupParentData gives the child a ListBodyParentData if it has none.
//
// Flutter: rendering/list_body.dart RenderListBody.setupParentData
func (r *RenderListBody) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*ListBodyParentData); !ok {
		child.AsRenderObject().SetParentData(&ListBodyParentData{})
	}
}

func listBodyParentData(child Box) *ListBodyParentData {
	return child.ParentData().(*ListBodyParentData)
}

// AxisDirection is the direction in which the children are laid out.
//
// Flutter: rendering/list_body.dart RenderListBody.axisDirection
func (r *RenderListBody) AxisDirection() painting.AxisDirection { return r.axisDirection }

// SetAxisDirection marks the list body as needing layout when the value changes.
//
// Flutter: rendering/list_body.dart RenderListBody.axisDirection
func (r *RenderListBody) SetAxisDirection(value painting.AxisDirection) {
	if value == painting.AxisDirectionUnspecified {
		value = painting.AxisDirectionDown
	}
	if r.axisDirection == value {
		return
	}
	r.axisDirection = value
	r.MarkNeedsLayout()
}

// MainAxis is the axis of the axis direction.
//
// Flutter: rendering/list_body.dart RenderListBody.mainAxis
func (r *RenderListBody) MainAxis() painting.Axis {
	return painting.AxisDirectionToAxis(r.axisDirection)
}

func (r *RenderListBody) isHorizontal() bool { return r.MainAxis() == painting.AxisHorizontal }

// debugCheckConstraints panics if the main axis is bounded or the cross axis is
// not.
//
// Flutter: rendering/list_body.dart RenderListBody._debugCheckConstraints
func (r *RenderListBody) debugCheckConstraints(constraints BoxConstraints) {
	if !foundation.DebugMode {
		return
	}
	mainBounded, crossBounded := constraints.HasBoundedHeight(), constraints.HasBoundedWidth()
	if r.isHorizontal() {
		mainBounded, crossBounded = constraints.HasBoundedWidth(), constraints.HasBoundedHeight()
	}
	if mainBounded {
		panic(errors.New(tagListBodyBoundedMainAxis, "RenderListBody must have unlimited space along its main axis. "+
			"RenderListBody does not clip or resize its children, so it must be placed in a parent that does not "+
			"constrain the main axis. You probably want to put the RenderListBody inside a RenderViewport with a "+
			"matching main axis.", "object", foundation.DescribeIdentity(r), "constraints", constraints))
	}
	if !crossBounded {
		panic(errors.New(tagListBodyUnboundedCrossAxis, "RenderListBody must have a bounded constraint for its cross axis. "+
			"RenderListBody forces its children to expand to fit the RenderListBody's container, so it must be placed in "+
			"a parent that constrains the cross axis to a finite dimension. If you are attempting to nest a "+
			"RenderListBody with one direction inside one of another direction, you will want to wrap the inner one "+
			"inside a box that fixes the dimension in that direction, for example, a RenderIntrinsicWidth or "+
			"RenderIntrinsicHeight object. This is relatively expensive, however.",
			"object", foundation.DescribeIdentity(r), "constraints", constraints))
	}
}

func (r *RenderListBody) crossTight(constraints BoxConstraints) BoxConstraints {
	if r.isHorizontal() {
		return NewBoxConstraintsTightFor(TightForOpts{Height: foundation.Some(constraints.MaxHeight)})
	}
	return NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(constraints.MaxWidth)})
}

// ComputeDryBaseline is the first baseline found along the main axis, or the
// highest one for a horizontal list.
//
// Flutter: rendering/list_body.dart RenderListBody.computeDryBaseline
func (r *RenderListBody) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	r.debugCheckConstraints(constraints)
	childConstraints := r.crossTight(constraints)
	if r.isHorizontal() {
		offset := NoBaseline
		for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
			offset = offset.MinOf(NewBaselineOffset(child.GetDryBaseline(childConstraints, baseline)))
		}
		return offset.Offset
	}
	child, next := r.FirstChild(), r.ChildAfter
	if r.axisDirection == painting.AxisDirectionUp {
		child, next = r.LastChild(), r.ChildBefore
	}
	mainAxisExtent := 0.0
	for ; !isNil(child); child = next(child) {
		if childBaseline, ok := child.GetDryBaseline(childConstraints, baseline).Get(); ok {
			return foundation.Some(childBaseline + mainAxisExtent)
		}
		mainAxisExtent += child.GetDryLayout(childConstraints).Height
	}
	return foundation.None[float64]()
}

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: rendering/list_body.dart RenderListBody.computeDryLayout
func (r *RenderListBody) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	r.debugCheckConstraints(constraints)
	inner := r.crossTight(constraints)
	mainAxisExtent := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		childSize := child.GetDryLayout(inner)
		if r.isHorizontal() {
			mainAxisExtent += childSize.Width
		} else {
			mainAxisExtent += childSize.Height
		}
	}
	if r.isHorizontal() {
		return constraints.Constrain(ui.Size{Width: mainAxisExtent, Height: constraints.MaxHeight})
	}
	return constraints.Constrain(ui.Size{Width: constraints.MaxWidth, Height: mainAxisExtent})
}

// PerformLayout lays the children out one after the other along the main axis,
// each as wide (or tall) as the list body.
//
// Flutter: rendering/list_body.dart RenderListBody.performLayout
func (r *RenderListBody) PerformLayout() {
	constraints := r.BoxConstraints()
	r.debugCheckConstraints(constraints)
	inner := r.crossTight(constraints)
	horizontal := r.isHorizontal()
	reversed := r.axisDirection == painting.AxisDirectionLeft || r.axisDirection == painting.AxisDirectionUp
	mainAxisExtent := 0.0
	extentOf := func(child Box) float64 {
		if horizontal {
			return child.Size().Width
		}
		return child.Size().Height
	}
	for child := r.FirstChild(); !isNil(child); {
		child.Layout(inner, LayoutOpts{ParentUsesSize: true})
		data := listBodyParentData(child)
		if !reversed {
			if horizontal {
				data.Offset = ui.Offset{Dx: mainAxisExtent}
			} else {
				data.Offset = ui.Offset{Dy: mainAxisExtent}
			}
		}
		mainAxisExtent += extentOf(child)
		child = data.NextSibling
	}
	if reversed {
		position := 0.0
		for child := r.FirstChild(); !isNil(child); {
			data := listBodyParentData(child)
			position += extentOf(child)
			if horizontal {
				data.Offset = ui.Offset{Dx: mainAxisExtent - position}
			} else {
				data.Offset = ui.Offset{Dy: mainAxisExtent - position}
			}
			child = data.NextSibling
		}
	}
	if horizontal {
		r.SetSize(constraints.Constrain(ui.Size{Width: mainAxisExtent, Height: constraints.MaxHeight}))
	} else {
		r.SetSize(constraints.Constrain(ui.Size{Width: constraints.MaxWidth, Height: mainAxisExtent}))
	}
}

// DebugFillProperties adds the axis direction.
//
// Flutter: rendering/list_body.dart RenderListBody.debugFillProperties
func (r *RenderListBody) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("axisDirection", foundation.Some(r.axisDirection)))
}

func (r *RenderListBody) intrinsicCrossAxis(childSize func(Box) float64) float64 {
	extent := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		extent = math.Max(extent, childSize(child))
	}
	return extent
}

func (r *RenderListBody) intrinsicMainAxis(childSize func(Box) float64) float64 {
	extent := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		extent += childSize(child)
	}
	return extent
}

// ComputeMinIntrinsicWidth is the sum (horizontal) or the maximum (vertical) of
// the children's.
//
// Flutter: rendering/list_body.dart RenderListBody.computeMinIntrinsicWidth
func (r *RenderListBody) ComputeMinIntrinsicWidth(height float64) float64 {
	get := func(c Box) float64 { return c.GetMinIntrinsicWidth(height) }
	if r.isHorizontal() {
		return r.intrinsicMainAxis(get)
	}
	return r.intrinsicCrossAxis(get)
}

// ComputeMaxIntrinsicWidth is the sum (horizontal) or the maximum (vertical) of
// the children's.
//
// Flutter: rendering/list_body.dart RenderListBody.computeMaxIntrinsicWidth
func (r *RenderListBody) ComputeMaxIntrinsicWidth(height float64) float64 {
	get := func(c Box) float64 { return c.GetMaxIntrinsicWidth(height) }
	if r.isHorizontal() {
		return r.intrinsicMainAxis(get)
	}
	return r.intrinsicCrossAxis(get)
}

// ComputeMinIntrinsicHeight is the sum (horizontal) or the maximum (vertical) of
// the children's.
//
// Flutter: rendering/list_body.dart RenderListBody.computeMinIntrinsicHeight
func (r *RenderListBody) ComputeMinIntrinsicHeight(width float64) float64 {
	get := func(c Box) float64 { return c.GetMinIntrinsicHeight(width) }
	if r.isHorizontal() {
		return r.intrinsicMainAxis(get)
	}
	return r.intrinsicCrossAxis(get)
}

// ComputeMaxIntrinsicHeight is the sum (horizontal) or the maximum (vertical) of
// the children's.
//
// Flutter: rendering/list_body.dart RenderListBody.computeMaxIntrinsicHeight
func (r *RenderListBody) ComputeMaxIntrinsicHeight(width float64) float64 {
	get := func(c Box) float64 { return c.GetMaxIntrinsicHeight(width) }
	if r.isHorizontal() {
		return r.intrinsicMainAxis(get)
	}
	return r.intrinsicCrossAxis(get)
}

// ComputeDistanceToActualBaseline is the first baseline among the children.
//
// Flutter: rendering/list_body.dart RenderListBody.computeDistanceToActualBaseline
func (r *RenderListBody) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return DefaultComputeDistanceToFirstActualBaseline(&r.ContainerRenderObject, baseline)
}

// Paint paints the children in order.
//
// Flutter: rendering/list_body.dart RenderListBody.paint
func (r *RenderListBody) Paint(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&r.ContainerRenderObject, context, offset)
}

// HitTestChildren hit tests the children from the front.
//
// Flutter: rendering/list_body.dart RenderListBody.hitTestChildren
func (r *RenderListBody) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagListBodyBoundedMainAxis    = "ListBodyBoundedMainAxis"
	tagListBodyUnboundedCrossAxis = "ListBodyUnboundedCrossAxis"
)
