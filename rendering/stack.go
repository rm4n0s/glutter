package rendering

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// StackParentData is the parent data of the children of a RenderStack: which
// edges of the stack the child is attached to, besides its offset.
//
// Flutter: rendering/stack.dart StackParentData
type StackParentData struct {
	ContainerBoxParentData[Box]
	// Top is the distance that the child's top edge is inset from the top of
	// the stack. None is null.
	//
	// Flutter: rendering/stack.dart StackParentData.top
	Top foundation.Opt[float64]
	// Right is the distance that the child's right edge is inset from the right
	// of the stack.
	//
	// Flutter: rendering/stack.dart StackParentData.right
	Right foundation.Opt[float64]
	// Bottom is the distance that the child's bottom edge is inset from the
	// bottom of the stack.
	//
	// Flutter: rendering/stack.dart StackParentData.bottom
	Bottom foundation.Opt[float64]
	// Left is the distance that the child's left edge is inset from the left of
	// the stack.
	//
	// Flutter: rendering/stack.dart StackParentData.left
	Left foundation.Opt[float64]
	// Width is the child's width, ignored if both Left and Right are set.
	//
	// Flutter: rendering/stack.dart StackParentData.width
	Width foundation.Opt[float64]
	// Height is the child's height, ignored if both Top and Bottom are set.
	//
	// Flutter: rendering/stack.dart StackParentData.height
	Height foundation.Opt[float64]
}

// Rect is the edges as a RelativeRect. It panics if an edge is not set (Flutter
// throws a null check error).
//
// Flutter: rendering/stack.dart StackParentData.rect
func (d *StackParentData) Rect() RelativeRect {
	return RelativeRect{Left: d.Left.Value(), Top: d.Top.Value(), Right: d.Right.Value(), Bottom: d.Bottom.Value()}
}

// SetRect sets the four edges.
//
// Flutter: rendering/stack.dart StackParentData.rect
func (d *StackParentData) SetRect(v RelativeRect) {
	d.Top, d.Right = foundation.Some(v.Top), foundation.Some(v.Right)
	d.Bottom, d.Left = foundation.Some(v.Bottom), foundation.Some(v.Left)
}

// IsPositioned reports whether any of the edges, the width or the height is set.
//
// Flutter: rendering/stack.dart StackParentData.isPositioned
func (d *StackParentData) IsPositioned() bool {
	return d.Top.IsSome() || d.Right.IsSome() || d.Bottom.IsSome() || d.Left.IsSome() || d.Width.IsSome() || d.Height.IsSome()
}

// PositionedChildConstraints are the tight constraints for a positioned child
// in a stack of the size: they fix the dimensions that the edges or the size
// determine, and leave the others unconstrained.
//
// Flutter: rendering/stack.dart StackParentData.positionedChildConstraints
func (d *StackParentData) PositionedChildConstraints(stackSize ui.Size) BoxConstraints {
	width, height := d.Width, d.Height
	if l, ok := d.Left.Get(); ok {
		if r, ok := d.Right.Get(); ok {
			width = foundation.Some(stackSize.Width - r - l)
		}
	}
	if t, ok := d.Top.Get(); ok {
		if b, ok := d.Bottom.Get(); ok {
			height = foundation.Some(stackSize.Height - b - t)
		}
	}
	var tf TightForOpts
	if w, ok := width.Get(); ok {
		tf.Width = foundation.Some(math.Max(0, w))
	}
	if h, ok := height.Get(); ok {
		tf.Height = foundation.Some(math.Max(0, h))
	}
	return NewBoxConstraintsTightFor(tf)
}

// String lists the set values, then the offset.
//
// Flutter: rendering/stack.dart StackParentData.toString
func (d *StackParentData) String() string {
	var values []string
	for _, v := range []struct {
		name  string
		value foundation.Opt[float64]
	}{{"top", d.Top}, {"right", d.Right}, {"bottom", d.Bottom}, {"left", d.Left}, {"width", d.Width}, {"height", d.Height}} {
		if f, ok := v.value.Get(); ok {
			values = append(values, v.name+"="+foundation.DebugFormatDouble(foundation.Some(f)))
		}
	}
	if len(values) == 0 {
		values = append(values, "not positioned")
	}
	values = append(values, d.BoxParentData.String())
	return strings.Join(values, "; ")
}

// StackFit is how the non-positioned children of a stack are sized. The zero
// value is StackFitUnspecified: loose.
//
// Flutter: rendering/stack.dart StackFit
type StackFit int

// The values of StackFit follow Flutter's declaration order.
const (
	// StackFitUnspecified is the zero value: the default (loose).
	//
	// Flutter: no counterpart
	StackFitUnspecified StackFit = iota
	// StackFitLoose gives the non-positioned children loose constraints: from
	// zero to the stack's size.
	//
	// Flutter: rendering/stack.dart StackFit.loose
	StackFitLoose
	// StackFitExpand gives the non-positioned children the biggest size.
	//
	// Flutter: rendering/stack.dart StackFit.expand
	StackFitExpand
	// StackFitPassthrough passes the stack's constraints to the non-positioned
	// children unmodified.
	//
	// Flutter: rendering/stack.dart StackFit.passthrough
	StackFitPassthrough
)

// Name returns the Dart enum name, such as "loose".
//
// Flutter: rendering/stack.dart StackFit.name
func (f StackFit) Name() string { return enumName(int(f), "loose", "expand", "passthrough") }

// String returns the name Dart prints, such as "StackFit.loose".
//
// Flutter: rendering/stack.dart StackFit.toString
func (f StackFit) String() string { return "StackFit." + f.Name() }

func (f StackFit) orLoose() StackFit {
	if f == StackFitUnspecified {
		return StackFitLoose
	}
	return f
}

func (f StackFit) nonPositionedConstraints(constraints BoxConstraints) BoxConstraints {
	switch f {
	case StackFitExpand:
		return NewBoxConstraintsTight(constraints.Biggest())
	case StackFitPassthrough:
		return constraints
	}
	return constraints.Loosen()
}

func stackParentData(child Box) *StackParentData { return child.ParentData().(*StackParentData) }

// RenderStack implements the stack layout algorithm: the children are painted in
// order, positioned relative to the edges of the box or aligned in it.
//
// Flutter: rendering/stack.dart RenderStack
type RenderStack struct {
	RenderBox
	ContainerRenderObject[Box]
	alignment         painting.AlignmentGeometry
	textDirection     ui.TextDirection
	fit               StackFit
	clipBehavior      ui.Clip
	resolvedAlignment foundation.Opt[painting.Alignment]
	hasVisualOverflow bool
	clipRectLayer     *LayerHandle[*ClipRectLayer]
}

// RenderStackOpts are the optional parameters of NewRenderStack.
//
// Flutter: rendering/stack.dart RenderStack.new
type RenderStackOpts struct {
	// Alignment defaults to painting.AlignmentDirectionalTopStart.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment; unspecified is null.
	TextDirection ui.TextDirection
	// Fit defaults to StackFitLoose.
	Fit StackFit
	// ClipBehavior defaults to ui.ClipHardEdge.
	ClipBehavior ui.Clip
}

// stackSelf is what a subclass of RenderStack can override.
type stackSelf interface {
	PaintStack(context *PaintingContext, offset ui.Offset)
}

// InitRenderStack sets the outer object of a stack, for a type that embeds
// RenderStack.
//
// Flutter: rendering/stack.dart RenderStack.new
func InitRenderStack(s *RenderStack, self Box, children []Box, o RenderStackOpts) {
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentDirectionalTopStart
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	s.alignment, s.textDirection, s.fit, s.clipBehavior = o.Alignment, o.TextDirection, o.Fit.orLoose(), o.ClipBehavior
	s.clipRectLayer = NewLayerHandle[*ClipRectLayer](nil)
	InitRenderBox(&s.RenderBox, self)
	InitContainerRenderObject(&s.ContainerRenderObject, self)
	s.AddAll(children)
}

// NewRenderStack creates a stack with the children (nil for none).
//
// Flutter: rendering/stack.dart RenderStack.new
func NewRenderStack(children []Box, opts ...RenderStackOpts) *RenderStack {
	s := &RenderStack{}
	InitRenderStack(s, s, children, firstOpts(opts))
	return s
}

// SetupParentData gives the child a StackParentData if it has none.
//
// Flutter: rendering/stack.dart RenderStack.setupParentData
func (s *RenderStack) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*StackParentData); !ok {
		child.AsRenderObject().SetParentData(&StackParentData{})
	}
}

// getResolvedAlignment is the alignment resolved with the text direction.
func (s *RenderStack) getResolvedAlignment() painting.Alignment {
	if !s.resolvedAlignment.IsSome() {
		s.resolvedAlignment = foundation.Some(s.alignment.Resolve(s.textDirection))
	}
	return s.resolvedAlignment.Value()
}

func (s *RenderStack) markNeedResolution() {
	s.resolvedAlignment = foundation.None[painting.Alignment]()
	s.MarkNeedsLayout()
}

// Alignment is how the non-positioned and partially positioned children are
// aligned.
//
// Flutter: rendering/stack.dart RenderStack.alignment
func (s *RenderStack) Alignment() painting.AlignmentGeometry { return s.alignment }

// SetAlignment marks the stack as needing layout when the value changes.
//
// Flutter: rendering/stack.dart RenderStack.alignment
func (s *RenderStack) SetAlignment(value painting.AlignmentGeometry) {
	if s.alignment == value {
		return
	}
	s.alignment = value
	s.markNeedResolution()
}

// TextDirection resolves a directional alignment. Unspecified is null.
//
// Flutter: rendering/stack.dart RenderStack.textDirection
func (s *RenderStack) TextDirection() ui.TextDirection { return s.textDirection }

// SetTextDirection marks the stack as needing layout when the value changes.
//
// Flutter: rendering/stack.dart RenderStack.textDirection
func (s *RenderStack) SetTextDirection(value ui.TextDirection) {
	if s.textDirection == value {
		return
	}
	s.textDirection = value
	s.markNeedResolution()
}

// Fit is how to size the non-positioned children.
//
// Flutter: rendering/stack.dart RenderStack.fit
func (s *RenderStack) Fit() StackFit { return s.fit }

// SetFit marks the stack as needing layout when the value changes.
//
// Flutter: rendering/stack.dart RenderStack.fit
func (s *RenderStack) SetFit(value StackFit) {
	value = value.orLoose()
	if s.fit != value {
		s.fit = value
		s.MarkNeedsLayout()
	}
}

// ClipBehavior is how the content that overflows the stack is clipped.
//
// Flutter: rendering/stack.dart RenderStack.clipBehavior
func (s *RenderStack) ClipBehavior() ui.Clip { return s.clipBehavior }

// SetClipBehavior marks the stack as needing paint when the value changes.
//
// Flutter: rendering/stack.dart RenderStack.clipBehavior
func (s *RenderStack) SetClipBehavior(value ui.Clip) {
	if value == ui.ClipUnspecified {
		value = ui.ClipHardEdge
	}
	if value != s.clipBehavior {
		s.clipBehavior = value
		s.MarkNeedsPaint()
		s.MarkNeedsSemanticsUpdate()
	}
}

// StackIntrinsicDimension is the largest value that the getter gives among the
// non-positioned children, from the first child on.
//
// Flutter: rendering/stack.dart RenderStack.getIntrinsicDimension
func StackIntrinsicDimension(firstChild Box, mainChildSizeGetter func(child Box) float64) float64 {
	extent := 0.0
	for child := firstChild; !isNil(child); {
		data := stackParentData(child)
		if !data.IsPositioned() {
			extent = math.Max(extent, mainChildSizeGetter(child))
		}
		child = data.NextSibling
	}
	return extent
}

// ComputeMinIntrinsicWidth is the largest of the non-positioned children.
//
// Flutter: rendering/stack.dart RenderStack.computeMinIntrinsicWidth
func (s *RenderStack) ComputeMinIntrinsicWidth(height float64) float64 {
	return StackIntrinsicDimension(s.FirstChild(), func(c Box) float64 { return c.GetMinIntrinsicWidth(height) })
}

// ComputeMaxIntrinsicWidth is the largest of the non-positioned children.
//
// Flutter: rendering/stack.dart RenderStack.computeMaxIntrinsicWidth
func (s *RenderStack) ComputeMaxIntrinsicWidth(height float64) float64 {
	return StackIntrinsicDimension(s.FirstChild(), func(c Box) float64 { return c.GetMaxIntrinsicWidth(height) })
}

// ComputeMinIntrinsicHeight is the largest of the non-positioned children.
//
// Flutter: rendering/stack.dart RenderStack.computeMinIntrinsicHeight
func (s *RenderStack) ComputeMinIntrinsicHeight(width float64) float64 {
	return StackIntrinsicDimension(s.FirstChild(), func(c Box) float64 { return c.GetMinIntrinsicHeight(width) })
}

// ComputeMaxIntrinsicHeight is the largest of the non-positioned children.
//
// Flutter: rendering/stack.dart RenderStack.computeMaxIntrinsicHeight
func (s *RenderStack) ComputeMaxIntrinsicHeight(width float64) float64 {
	return StackIntrinsicDimension(s.FirstChild(), func(c Box) float64 { return c.GetMaxIntrinsicHeight(width) })
}

// ComputeDistanceToActualBaseline is the highest baseline among the children.
//
// Flutter: rendering/stack.dart RenderStack.computeDistanceToActualBaseline
func (s *RenderStack) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return DefaultComputeDistanceToHighestActualBaseline(&s.ContainerRenderObject, baseline)
}

// LayoutPositionedStackChild lays out a positioned child of a stack of the size
// and sets its offset. It reports whether the child overflows the stack.
//
// Flutter: rendering/stack.dart RenderStack.layoutPositionedChild
func LayoutPositionedStackChild(child Box, data *StackParentData, size ui.Size, alignment painting.Alignment) bool {
	if foundation.DebugMode && !data.IsPositioned() {
		panic(errors.New("StackChildNotPositioned", "the child is not positioned", "child", foundation.DescribeIdentity(child)))
	}
	child.Layout(data.PositionedChildConstraints(size), LayoutOpts{ParentUsesSize: true})
	childSize := child.Size()
	var x, y float64
	switch {
	case data.Left.IsSome():
		x = data.Left.Value()
	case data.Right.IsSome():
		x = size.Width - data.Right.Value() - childSize.Width
	default:
		x = alignment.AlongOffset(sizeDiff(size, childSize)).Dx
	}
	switch {
	case data.Top.IsSome():
		y = data.Top.Value()
	case data.Bottom.IsSome():
		y = size.Height - data.Bottom.Value() - childSize.Height
	default:
		y = alignment.AlongOffset(sizeDiff(size, childSize)).Dy
	}
	data.Offset = ui.Offset{Dx: x, Dy: y}
	return x < 0 || x+childSize.Width > size.Width || y < 0 || y+childSize.Height > size.Height
}

// StackBaselineForChild is the baseline of the child, in the stack of the size,
// for a dry layout: the child is laid out with its positioned constraints, or
// with the constraints of a non-positioned child.
//
// Flutter: widgets/overlay.dart _RenderTheaterMixin.baselineForChild
func StackBaselineForChild(child Box, stackSize ui.Size, nonPositioned BoxConstraints, alignment painting.Alignment,
	baseline ui.TextBaseline) foundation.Opt[float64] {
	return stackBaselineForChild(child, stackSize, nonPositioned, alignment, baseline)
}

func stackBaselineForChild(child Box, stackSize ui.Size, nonPositioned BoxConstraints, alignment painting.Alignment,
	baseline ui.TextBaseline) foundation.Opt[float64] {
	data := stackParentData(child)
	childConstraints := nonPositioned
	if data.IsPositioned() {
		childConstraints = data.PositionedChildConstraints(stackSize)
	}
	baselineOffset, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	var y float64
	switch {
	case data.Top.IsSome():
		y = data.Top.Value()
	case data.Bottom.IsSome():
		y = stackSize.Height - data.Bottom.Value() - child.GetDryLayout(childConstraints).Height
	default:
		y = alignment.AlongOffset(sizeDiff(stackSize, child.GetDryLayout(childConstraints))).Dy
	}
	return foundation.Some(baselineOffset + y)
}

// ComputeDryBaseline is the highest baseline among the children for the
// constraints.
//
// Flutter: rendering/stack.dart RenderStack.computeDryBaseline
func (s *RenderStack) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	nonPositioned := s.fit.nonPositionedConstraints(constraints)
	alignment := s.getResolvedAlignment()
	size := s.GetDryLayout(constraints)
	minBaseline := NoBaseline
	for child := s.FirstChild(); !isNil(child); child = s.ChildAfter(child) {
		minBaseline = minBaseline.MinOf(NewBaselineOffset(stackBaselineForChild(child, size, nonPositioned, alignment, baseline)))
	}
	return minBaseline.Offset
}

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: rendering/stack.dart RenderStack.computeDryLayout
func (s *RenderStack) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return s.computeSize(constraints, ChildLayoutHelper.DryLayoutChild)
}

func (s *RenderStack) computeSize(constraints BoxConstraints, layoutChild ChildLayouter) ui.Size {
	if s.ChildCount() == 0 {
		if constraints.Biggest().IsFinite() {
			return constraints.Biggest()
		}
		return constraints.Smallest()
	}
	width, height := constraints.MinWidth, constraints.MinHeight
	nonPositioned := s.fit.nonPositionedConstraints(constraints)
	hasNonPositioned := false
	for child := s.FirstChild(); !isNil(child); {
		data := stackParentData(child)
		if !data.IsPositioned() {
			hasNonPositioned = true
			childSize := layoutChild(child, nonPositioned)
			width, height = math.Max(width, childSize.Width), math.Max(height, childSize.Height)
		}
		child = data.NextSibling
	}
	size := constraints.Biggest()
	if hasNonPositioned {
		size = ui.Size{Width: width, Height: height}
	}
	if foundation.DebugMode && !size.IsFinite() {
		panic(errors.New("StackUnboundedConstraints", "A Stack requires bounded constraints from its parent. "+
			"This error commonly occurs when a Stack is placed inside a widget like Column, ListView, or other widgets "+
			"that do not constrain their children. To fix this, wrap the Stack in a widget that provides finite height "+
			"and width constraints, such as a SizedBox or ConstrainedBox. Use Expanded only if the parent is a Flex "+
			"widget like Row or Column.", "object", foundation.DescribeIdentity(s.self), "constraints", constraints))
	}
	return size
}

// PerformLayout sizes the stack around its non-positioned children, then lays
// out the children.
//
// Flutter: rendering/stack.dart RenderStack.performLayout
func (s *RenderStack) PerformLayout() {
	constraints := s.BoxConstraints()
	s.hasVisualOverflow = false
	s.SetSize(s.computeSize(constraints, ChildLayoutHelper.LayoutChild))
	size := s.Size()
	resolved := s.getResolvedAlignment()
	for child := s.FirstChild(); !isNil(child); {
		data := stackParentData(child)
		if !data.IsPositioned() {
			data.Offset = resolved.AlongOffset(sizeDiff(size, child.Size()))
		} else if LayoutPositionedStackChild(child, data, size, resolved) {
			s.hasVisualOverflow = true
		}
		child = data.NextSibling
	}
}

// HitTestChildren hit tests the children from the front.
//
// Flutter: rendering/stack.dart RenderStack.hitTestChildren
func (s *RenderStack) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&s.ContainerRenderObject, result, position)
}

// PaintStack paints the children. A subclass overrides it to paint differently.
//
// Flutter: rendering/stack.dart RenderStack.paintStack
func (s *RenderStack) PaintStack(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&s.ContainerRenderObject, context, offset)
}

func (s *RenderStack) paintStackSelf(context *PaintingContext, offset ui.Offset) {
	s.self.(stackSelf).PaintStack(context, offset)
}

// Paint paints the children, clipped to the stack if one overflows it.
//
// Flutter: rendering/stack.dart RenderStack.paint
func (s *RenderStack) Paint(context *PaintingContext, offset ui.Offset) {
	if s.clipBehavior != ui.ClipNone && s.hasVisualOverflow {
		s.clipRectLayer.SetLayer(context.PushClipRect(s.NeedsCompositing(), offset, ui.OffsetZero.And(s.Size()),
			s.paintStackSelf, PushClipRectOpts{ClipBehavior: s.clipBehavior, OldLayer: s.clipRectLayer.Layer()}))
		return
	}
	s.clipRectLayer.SetLayer(nil)
	s.paintStackSelf(context, offset)
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/stack.dart RenderStack.dispose
func (s *RenderStack) Dispose() {
	s.clipRectLayer.SetLayer(nil)
	s.RenderBox.Dispose()
}

// DescribeApproximatePaintClip is the stack's box when a child overflows and
// is clipped.
//
// Flutter: rendering/stack.dart RenderStack.describeApproximatePaintClip
func (s *RenderStack) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if s.clipBehavior != ui.ClipNone && s.hasVisualOverflow {
		return foundation.Some(ui.OffsetZero.And(s.Size()))
	}
	return foundation.None[ui.Rect]()
}

// DebugFillProperties adds the alignment, the text direction, the fit and the
// clip behavior.
//
// Flutter: rendering/stack.dart RenderStack.debugFillProperties
func (s *RenderStack) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	s.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("alignment", s.alignment))
	b.Add(foundation.NewEnumProperty("textDirection", optEnum(s.textDirection, ui.TextDirectionUnspecified)))
	b.Add(foundation.NewEnumProperty("fit", foundation.Some(s.fit)))
	b.Add(foundation.NewEnumProperty("clipBehavior", foundation.Some(s.clipBehavior), foundation.EnumPropertyOpts{DefaultValue: ui.ClipHardEdge}))
}

// RenderIndexedStack shows a single child from a list of children.
//
// Flutter: rendering/stack.dart RenderIndexedStack
type RenderIndexedStack struct {
	RenderStack
	index foundation.Opt[int]
}

// RenderIndexedStackOpts are the optional parameters of NewRenderIndexedStack.
//
// Flutter: rendering/stack.dart RenderIndexedStack.new
type RenderIndexedStackOpts struct {
	RenderStackOpts
	// Index is the index of the child to show. None is the default, 0; use
	// SetIndex(None) to show no child (Flutter's null).
	Index foundation.Opt[int]
}

// NewRenderIndexedStack creates an indexed stack with the children (nil for
// none).
//
// Flutter: rendering/stack.dart RenderIndexedStack.new
func NewRenderIndexedStack(children []Box, opts ...RenderIndexedStackOpts) *RenderIndexedStack {
	o := firstOpts(opts)
	s := &RenderIndexedStack{index: o.Index}
	if !o.Index.IsSome() {
		s.index = foundation.Some(0)
	}
	InitRenderStack(&s.RenderStack, s, children, o.RenderStackOpts)
	return s
}

// Index is the index of the child to show, or None for none.
//
// Flutter: rendering/stack.dart RenderIndexedStack.index
func (s *RenderIndexedStack) Index() foundation.Opt[int] { return s.index }

// SetIndex marks the stack as needing layout when the value changes.
//
// Flutter: rendering/stack.dart RenderIndexedStack.index
func (s *RenderIndexedStack) SetIndex(value foundation.Opt[int]) {
	if s.index != value {
		s.index = value
		s.MarkNeedsLayout()
	}
}

func (s *RenderIndexedStack) childAtIndex() Box {
	index, ok := s.index.Get()
	if !ok {
		return nil
	}
	child := s.FirstChild()
	for i := 0; i < index && !isNil(child); i++ {
		child = s.ChildAfter(child)
	}
	return child
}

// VisitChildrenForSemantics visits only the displayed child.
//
// Flutter: rendering/stack.dart RenderIndexedStack.visitChildrenForSemantics
func (s *RenderIndexedStack) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if displayed := s.childAtIndex(); !isNil(displayed) {
		visitor(displayed)
	}
}

// ComputeDistanceToActualBaseline is the baseline of the displayed child.
//
// Flutter: rendering/stack.dart RenderIndexedStack.computeDistanceToActualBaseline
func (s *RenderIndexedStack) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	displayed := s.childAtIndex()
	if isNil(displayed) {
		return foundation.None[float64]()
	}
	return NewBaselineOffset(displayed.GetDistanceToActualBaseline(baseline)).Add(stackParentData(displayed).Offset.Dy).Offset
}

// ComputeDryBaseline is the baseline of the displayed child.
//
// Flutter: rendering/stack.dart RenderIndexedStack.computeDryBaseline
func (s *RenderIndexedStack) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	displayed := s.childAtIndex()
	if isNil(displayed) {
		return foundation.None[float64]()
	}
	return stackBaselineForChild(displayed, s.GetDryLayout(constraints), s.fit.nonPositionedConstraints(constraints),
		s.getResolvedAlignment(), baseline)
}

// HitTestChildren hit tests the displayed child only.
//
// Flutter: rendering/stack.dart RenderIndexedStack.hitTestChildren
func (s *RenderIndexedStack) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	displayed := s.childAtIndex()
	if isNil(displayed) {
		return false
	}
	offset := stackParentData(displayed).Offset
	return result.AddWithPaintOffset(&offset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
		return displayed.HitTest(result, transformed)
	})
}

// PaintStack paints the displayed child only.
//
// Flutter: rendering/stack.dart RenderIndexedStack.paintStack
func (s *RenderIndexedStack) PaintStack(context *PaintingContext, offset ui.Offset) {
	displayed := s.childAtIndex()
	if isNil(displayed) {
		return
	}
	context.PaintChild(displayed, stackParentData(displayed).Offset.Add(offset))
}

// DebugFillProperties adds the index.
//
// Flutter: rendering/stack.dart RenderIndexedStack.debugFillProperties
func (s *RenderIndexedStack) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	s.RenderStack.DebugFillProperties(b)
	b.Add(foundation.NewIntProperty("index", s.index))
}

// DebugDescribeChildren describes the children, the hidden ones with the
// offstage style.
//
// Flutter: rendering/stack.dart RenderIndexedStack.debugDescribeChildren
func (s *RenderIndexedStack) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var out []foundation.DiagnosticsNode
	i := 0
	for child := s.FirstChild(); !isNil(child); child = stackParentData(child).NextSibling {
		opts := foundation.DiagnosticsNodeOpts{Name: foundation.Some("child " + itoa(i+1))}
		if idx, ok := s.index.Get(); !ok || i != idx {
			opts.Style = foundation.DiagnosticsTreeStyleOffstage
		}
		out = append(out, foundation.ToDiagnosticsNode(child, opts))
		i++
	}
	return out
}
