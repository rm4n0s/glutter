package rendering

import (
	"math"
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// layerAs is the layer as the concrete type L, or the zero value (nil) if it is
// another type or none.
//
// Flutter: no counterpart (the cast `layer as ClipRectLayer?`)
func layerAs[L ContainerLayerLike](l ContainerLayerLike) L {
	v, _ := l.(L)
	return v
}

// setObjectLayer sets the layer of the object; a nil pointer of a concrete
// layer type lets the layer go.
//
// Flutter: no counterpart (a Dart null is a Go typed nil)
func setObjectLayer[L ContainerLayerLike](b *RenderObjectBase, l L) {
	var zero L
	if any(l) == any(zero) {
		b.SetLayer(nil)
		return
	}
	b.SetLayer(l)
}

// RenderProxyBox is a render box that has the size of its single child, lays it
// out with the same constraints, and paints and hit tests it as is. Embed it in
// the render boxes that change one of these, and call InitRenderProxyBox in the
// constructor.
//
// Flutter: rendering/proxy_box.dart RenderProxyBox, RenderProxyBoxMixin
type RenderProxyBox struct {
	RenderBox
	RenderObjectWithChild[Box]
}

// InitRenderProxyBox sets the outer object of a proxy box and its child (nil
// for none).
//
// Flutter: rendering/proxy_box.dart RenderProxyBox.new
func InitRenderProxyBox(p *RenderProxyBox, self Box, child Box) {
	InitRenderBox(&p.RenderBox, self)
	InitRenderObjectWithChild(&p.RenderObjectWithChild, self)
	if !isNil(child) {
		p.SetChild(child)
	}
}

// NewRenderProxyBox creates a proxy box with the child (nil for none).
//
// Flutter: rendering/proxy_box.dart RenderProxyBox.new
func NewRenderProxyBox(child Box) *RenderProxyBox {
	p := &RenderProxyBox{}
	InitRenderProxyBox(p, p, child)
	return p
}

// hasChild reports whether the box has a child.
func (p *RenderProxyBox) hasChild() bool { return !isNil(p.Child()) }

// SetupParentData gives the child a ParentData if it has none.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.setupParentData
func (p *RenderProxyBox) SetupParentData(child RenderObject) {
	if child.ParentData() == nil {
		child.AsRenderObject().SetParentData(&ParentDataBase{})
	}
}

// ComputeMinIntrinsicWidth is the child's, or 0.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeMinIntrinsicWidth
func (p *RenderProxyBox) ComputeMinIntrinsicWidth(height float64) float64 {
	if p.hasChild() {
		return p.Child().GetMinIntrinsicWidth(height)
	}
	return 0
}

// ComputeMaxIntrinsicWidth is the child's, or 0.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeMaxIntrinsicWidth
func (p *RenderProxyBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	if p.hasChild() {
		return p.Child().GetMaxIntrinsicWidth(height)
	}
	return 0
}

// ComputeMinIntrinsicHeight is the child's, or 0.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeMinIntrinsicHeight
func (p *RenderProxyBox) ComputeMinIntrinsicHeight(width float64) float64 {
	if p.hasChild() {
		return p.Child().GetMinIntrinsicHeight(width)
	}
	return 0
}

// ComputeMaxIntrinsicHeight is the child's, or 0.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeMaxIntrinsicHeight
func (p *RenderProxyBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	if p.hasChild() {
		return p.Child().GetMaxIntrinsicHeight(width)
	}
	return 0
}

// ComputeDistanceToActualBaseline is the child's, or the default.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeDistanceToActualBaseline
func (p *RenderProxyBox) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if p.hasChild() {
		if v := p.Child().GetDistanceToActualBaseline(baseline); v.IsSome() {
			return v
		}
	}
	return p.RenderBox.ComputeDistanceToActualBaseline(baseline)
}

// ComputeDryBaseline is the child's, or the default.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeDryBaseline
func (p *RenderProxyBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if p.hasChild() {
		if v := p.Child().GetDryBaseline(constraints, baseline); v.IsSome() {
			return v
		}
	}
	return p.RenderBox.ComputeDryBaseline(constraints, baseline)
}

// ComputeDryLayout is the child's dry layout, or the size for no child.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeDryLayout
func (p *RenderProxyBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if p.hasChild() {
		return p.Child().GetDryLayout(constraints)
	}
	return p.noChildSize(constraints)
}

type noChildSizer interface {
	ComputeSizeForNoChild(constraints BoxConstraints) ui.Size
}

func (p *RenderProxyBox) noChildSize(constraints BoxConstraints) ui.Size {
	return p.self.(noChildSizer).ComputeSizeForNoChild(constraints)
}

// PerformLayout lays out the child with the constraints and takes its size.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.performLayout
func (p *RenderProxyBox) PerformLayout() {
	if p.hasChild() {
		c := p.Child()
		c.Layout(p.BoxConstraints(), LayoutOpts{ParentUsesSize: true})
		p.SetSize(c.Size())
		return
	}
	p.SetSize(p.noChildSize(p.BoxConstraints()))
}

// ComputeSizeForNoChild is the size of the box without a child: the smallest
// size.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.computeSizeForNoChild
func (p *RenderProxyBox) ComputeSizeForNoChild(constraints BoxConstraints) ui.Size {
	return constraints.Smallest()
}

// HitTestChildren hit tests the child.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.hitTestChildren
func (p *RenderProxyBox) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return p.hasChild() && p.Child().HitTest(result, position)
}

// ApplyPaintTransform does nothing: the child is painted at the same offset.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.applyPaintTransform
func (p *RenderProxyBox) ApplyPaintTransform(RenderObject, *ui.Matrix4) {}

// Paint paints the child.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxMixin.paint
func (p *RenderProxyBox) Paint(context *PaintingContext, offset ui.Offset) {
	if p.hasChild() {
		context.PaintChild(p.Child(), offset)
	}
}

// HitTestBehavior is how a box behaves during hit testing.
//
// Flutter: rendering/proxy_box.dart HitTestBehavior
type HitTestBehavior int

// The values of HitTestBehavior follow Flutter's declaration order.
const (
	// HitTestBehaviorUnspecified is the zero value: the widget's default.
	//
	// Flutter: no counterpart
	HitTestBehaviorUnspecified HitTestBehavior = iota
	// HitTestBehaviorDeferToChild targets only the boxes whose children are hit.
	//
	// Flutter: rendering/proxy_box.dart HitTestBehavior.deferToChild
	HitTestBehaviorDeferToChild
	// HitTestBehaviorOpaque targets the box, so the boxes behind it are not hit.
	//
	// Flutter: rendering/proxy_box.dart HitTestBehavior.opaque
	HitTestBehaviorOpaque
	// HitTestBehaviorTranslucent targets the box and lets the boxes behind it be
	// hit as well.
	//
	// Flutter: rendering/proxy_box.dart HitTestBehavior.translucent
	HitTestBehaviorTranslucent
)

// Name returns the Dart enum name, such as "opaque".
//
// Flutter: rendering/proxy_box.dart HitTestBehavior.name
func (b HitTestBehavior) Name() string {
	switch b {
	case HitTestBehaviorDeferToChild:
		return "deferToChild"
	case HitTestBehaviorOpaque:
		return "opaque"
	case HitTestBehaviorTranslucent:
		return "translucent"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "HitTestBehavior.opaque".
//
// Flutter: rendering/proxy_box.dart HitTestBehavior.toString
func (b HitTestBehavior) String() string { return "HitTestBehavior." + b.Name() }

// orDefer is the behavior, or deferToChild for the zero value.
func (b HitTestBehavior) orDefer() HitTestBehavior {
	if b == HitTestBehaviorUnspecified {
		return HitTestBehaviorDeferToChild
	}
	return b
}

// RenderProxyBoxWithHitTestBehavior is a proxy box that can be targeted by hit
// testing in more ways than its child.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxWithHitTestBehavior
type RenderProxyBoxWithHitTestBehavior struct {
	RenderProxyBox
	// Behavior is how the box behaves during hit testing; the zero value is
	// HitTestBehaviorDeferToChild.
	Behavior HitTestBehavior
}

// InitRenderProxyBoxWithHitTestBehavior sets the outer object, the behavior
// (the zero value is deferToChild) and the child of the box.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxWithHitTestBehavior.new
func InitRenderProxyBoxWithHitTestBehavior(p *RenderProxyBoxWithHitTestBehavior, self Box, behavior HitTestBehavior, child Box) {
	InitRenderProxyBox(&p.RenderProxyBox, self, child)
	p.Behavior = behavior.orDefer()
}

// HitTest hits the box if the position is inside it and a child is hit, or the
// box is opaque; a translucent box also adds itself when it is not hit.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxWithHitTestBehavior.hitTest
func (p *RenderProxyBoxWithHitTestBehavior) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	hitTarget := false
	if p.Size().Contains(position) {
		self := p.self.(Box)
		hitTarget = self.HitTestChildren(result, position) || self.HitTestSelf(position)
		if hitTarget || p.Behavior == HitTestBehaviorTranslucent {
			result.Add(NewBoxHitTestEntry(self, position))
		}
	}
	return hitTarget
}

// HitTestSelf is true if the box is opaque.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxWithHitTestBehavior.hitTestSelf
func (p *RenderProxyBoxWithHitTestBehavior) HitTestSelf(ui.Offset) bool {
	return p.Behavior == HitTestBehaviorOpaque
}

// DebugFillProperties adds the behavior.
//
// Flutter: rendering/proxy_box.dart RenderProxyBoxWithHitTestBehavior.debugFillProperties
func (p *RenderProxyBoxWithHitTestBehavior) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	p.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("behavior", foundation.Some(p.Behavior), foundation.EnumPropertyOpts{DefaultValue: HitTestBehaviorDeferToChild}))
}

// RenderConstrainedBox imposes additional constraints on its child.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox
type RenderConstrainedBox struct {
	RenderProxyBox
	additionalConstraints BoxConstraints
}

// NewRenderConstrainedBox creates a box that enforces the additional constraints
// on the child (nil for none).
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.new
func NewRenderConstrainedBox(additionalConstraints BoxConstraints, child Box) *RenderConstrainedBox {
	if foundation.DebugMode {
		additionalConstraints.DebugAssertIsValid()
	}
	r := &RenderConstrainedBox{additionalConstraints: additionalConstraints}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// AdditionalConstraints are the constraints that are enforced on the child.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.additionalConstraints
func (r *RenderConstrainedBox) AdditionalConstraints() BoxConstraints {
	return r.additionalConstraints
}

// SetAdditionalConstraints changes the constraints and marks the box as needing
// layout.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.additionalConstraints
func (r *RenderConstrainedBox) SetAdditionalConstraints(value BoxConstraints) {
	if foundation.DebugMode {
		value.DebugAssertIsValid()
	}
	if r.additionalConstraints == value {
		return
	}
	r.additionalConstraints = value
	r.MarkNeedsLayout()
}

// ComputeMinIntrinsicWidth is the minimum width of the additional constraints
// if they are tight, otherwise the child's width constrained by them.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeMinIntrinsicWidth
func (r *RenderConstrainedBox) ComputeMinIntrinsicWidth(height float64) float64 {
	c := r.additionalConstraints
	if c.HasBoundedWidth() && c.HasTightWidth() {
		return c.MinWidth
	}
	width := r.RenderProxyBox.ComputeMinIntrinsicWidth(height)
	if !c.HasInfiniteWidth() {
		return c.ConstrainWidth(width)
	}
	return width
}

// ComputeMaxIntrinsicWidth is like ComputeMinIntrinsicWidth for the maximum.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeMaxIntrinsicWidth
func (r *RenderConstrainedBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	c := r.additionalConstraints
	if c.HasBoundedWidth() && c.HasTightWidth() {
		return c.MinWidth
	}
	width := r.RenderProxyBox.ComputeMaxIntrinsicWidth(height)
	if !c.HasInfiniteWidth() {
		return c.ConstrainWidth(width)
	}
	return width
}

// ComputeMinIntrinsicHeight is the minimum height of the additional constraints
// if they are tight, otherwise the child's height constrained by them.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeMinIntrinsicHeight
func (r *RenderConstrainedBox) ComputeMinIntrinsicHeight(width float64) float64 {
	c := r.additionalConstraints
	if c.HasBoundedHeight() && c.HasTightHeight() {
		return c.MinHeight
	}
	height := r.RenderProxyBox.ComputeMinIntrinsicHeight(width)
	if !c.HasInfiniteHeight() {
		return c.ConstrainHeight(height)
	}
	return height
}

// ComputeMaxIntrinsicHeight is like ComputeMinIntrinsicHeight for the maximum.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeMaxIntrinsicHeight
func (r *RenderConstrainedBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	c := r.additionalConstraints
	if c.HasBoundedHeight() && c.HasTightHeight() {
		return c.MinHeight
	}
	height := r.RenderProxyBox.ComputeMaxIntrinsicHeight(width)
	if !c.HasInfiniteHeight() {
		return c.ConstrainHeight(height)
	}
	return height
}

// ComputeDryBaseline is the child's baseline for the enforced constraints.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeDryBaseline
func (r *RenderConstrainedBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.hasChild() {
		return r.Child().GetDryBaseline(r.additionalConstraints.Enforce(constraints), baseline)
	}
	return foundation.None[float64]()
}

// PerformLayout lays out the child with the additional constraints enforced.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.performLayout
func (r *RenderConstrainedBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if r.hasChild() {
		c := r.Child()
		c.Layout(r.additionalConstraints.Enforce(constraints), LayoutOpts{ParentUsesSize: true})
		r.SetSize(c.Size())
		return
	}
	r.SetSize(r.additionalConstraints.Enforce(constraints).Constrain(ui.Size{}))
}

// ComputeDryLayout is PerformLayout without the layout.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.computeDryLayout
func (r *RenderConstrainedBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.hasChild() {
		return r.Child().GetDryLayout(r.additionalConstraints.Enforce(constraints))
	}
	return r.additionalConstraints.Enforce(constraints).Constrain(ui.Size{})
}

// DebugPaintSize also paints a gray box over a box that has no (or an empty)
// child.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.debugPaintSize
func (r *RenderConstrainedBox) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	r.RenderProxyBox.DebugPaintSize(context, offset)
	if foundation.DebugMode && (!r.hasChild() || r.Child().Size().IsEmpty()) {
		paint := ui.NewPaint()
		paint.SetColor(ui.NewColor(0x90909090))
		context.Canvas().DrawRect(offset.And(r.Size()), paint)
	}
}

// DebugFillProperties adds the additional constraints.
//
// Flutter: rendering/proxy_box.dart RenderConstrainedBox.debugFillProperties
func (r *RenderConstrainedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("additionalConstraints", r.additionalConstraints))
}

// RenderLimitedBox limits its size only when it is unconstrained.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox
type RenderLimitedBox struct {
	RenderProxyBox
	maxWidth, maxHeight float64
}

// RenderLimitedBoxOpts are the optional parameters of NewRenderLimitedBox.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.new
type RenderLimitedBoxOpts struct {
	// MaxWidth defaults to infinity.
	MaxWidth foundation.Opt[float64]
	// MaxHeight defaults to infinity.
	MaxHeight foundation.Opt[float64]
}

// NewRenderLimitedBox creates a box that limits its child to the maximum size
// when the constraints are unbounded. Negative limits panic with
// LimitedBoxNegative in debug builds.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.new
func NewRenderLimitedBox(child Box, opts ...RenderLimitedBoxOpts) *RenderLimitedBox {
	o := firstOpts(opts)
	r := &RenderLimitedBox{maxWidth: o.MaxWidth.OrElse(math.Inf(1)), maxHeight: o.MaxHeight.OrElse(math.Inf(1))}
	r.assertLimits()
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

func (r *RenderLimitedBox) assertLimits() {
	if foundation.DebugMode && (r.maxWidth < 0 || r.maxHeight < 0) {
		panic(errors.New("LimitedBoxNegative", "the limits of a RenderLimitedBox must not be negative",
			"maxWidth", r.maxWidth, "maxHeight", r.maxHeight))
	}
}

// MaxWidth is the maximum width when the width is unconstrained.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.maxWidth
func (r *RenderLimitedBox) MaxWidth() float64 { return r.maxWidth }

// SetMaxWidth changes the limit and marks the box as needing layout.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.maxWidth
func (r *RenderLimitedBox) SetMaxWidth(value float64) {
	old := r.maxWidth
	r.maxWidth = value
	r.assertLimits()
	if old == value {
		return
	}
	r.MarkNeedsLayout()
}

// MaxHeight is the maximum height when the height is unconstrained.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.maxHeight
func (r *RenderLimitedBox) MaxHeight() float64 { return r.maxHeight }

// SetMaxHeight changes the limit and marks the box as needing layout.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.maxHeight
func (r *RenderLimitedBox) SetMaxHeight(value float64) {
	old := r.maxHeight
	r.maxHeight = value
	r.assertLimits()
	if old == value {
		return
	}
	r.MarkNeedsLayout()
}

func (r *RenderLimitedBox) limitConstraints(c BoxConstraints) BoxConstraints {
	maxW, maxH := c.MaxWidth, c.MaxHeight
	if !c.HasBoundedWidth() {
		maxW = c.ConstrainWidth(r.maxWidth)
	}
	if !c.HasBoundedHeight() {
		maxH = c.ConstrainHeight(r.maxHeight)
	}
	return BoxConstraints{MinWidth: c.MinWidth, MaxWidth: maxW, MinHeight: c.MinHeight, MaxHeight: maxH}
}

func (r *RenderLimitedBox) computeSize(constraints BoxConstraints, layoutChild ChildLayouter) ui.Size {
	if r.hasChild() {
		return constraints.Constrain(layoutChild(r.Child(), r.limitConstraints(constraints)))
	}
	return r.limitConstraints(constraints).Constrain(ui.Size{})
}

// ComputeDryLayout is the size for the constraints.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.computeDryLayout
func (r *RenderLimitedBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.computeSize(constraints, ChildLayoutHelper.DryLayoutChild)
}

// PerformLayout lays out the child with the limited constraints.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.performLayout
func (r *RenderLimitedBox) PerformLayout() {
	r.SetSize(r.computeSize(r.BoxConstraints(), ChildLayoutHelper.LayoutChild))
}

// DebugFillProperties adds the limits.
//
// Flutter: rendering/proxy_box.dart RenderLimitedBox.debugFillProperties
func (r *RenderLimitedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("maxWidth", foundation.Some(r.maxWidth), foundation.NumberPropertyOpts{DefaultValue: math.Inf(1)}))
	b.Add(foundation.NewDoubleProperty("maxHeight", foundation.Some(r.maxHeight), foundation.NumberPropertyOpts{DefaultValue: math.Inf(1)}))
}

// RenderAspectRatio attempts to size its child to a given aspect ratio.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio
type RenderAspectRatio struct {
	RenderProxyBox
	aspectRatio float64
}

// NewRenderAspectRatio creates a box with the aspect ratio (width over height).
// A ratio that is not positive and finite panics with AspectRatioInvalid in
// debug builds.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.new
func NewRenderAspectRatio(aspectRatio float64, child Box) *RenderAspectRatio {
	assertAspectRatio(aspectRatio)
	r := &RenderAspectRatio{aspectRatio: aspectRatio}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

func assertAspectRatio(v float64) {
	if foundation.DebugMode && (!(v > 0) || math.IsInf(v, 0)) {
		panic(errors.New("AspectRatioInvalid", "the aspect ratio must be positive and finite", "aspectRatio", v))
	}
}

// AspectRatio is the ratio of width to height.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.aspectRatio
func (r *RenderAspectRatio) AspectRatio() float64 { return r.aspectRatio }

// SetAspectRatio changes the ratio and marks the box as needing layout.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.aspectRatio
func (r *RenderAspectRatio) SetAspectRatio(value float64) {
	assertAspectRatio(value)
	if r.aspectRatio == value {
		return
	}
	r.aspectRatio = value
	r.MarkNeedsLayout()
}

// ComputeMinIntrinsicWidth is the height times the ratio, or the child's when
// the height is infinite.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeMinIntrinsicWidth
func (r *RenderAspectRatio) ComputeMinIntrinsicWidth(height float64) float64 {
	if !math.IsInf(height, 0) {
		return height * r.aspectRatio
	}
	return r.RenderProxyBox.ComputeMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the height times the ratio, or the child's when
// the height is infinite.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeMaxIntrinsicWidth
func (r *RenderAspectRatio) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !math.IsInf(height, 0) {
		return height * r.aspectRatio
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the width over the ratio, or the child's when the
// width is infinite.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeMinIntrinsicHeight
func (r *RenderAspectRatio) ComputeMinIntrinsicHeight(width float64) float64 {
	if !math.IsInf(width, 0) {
		return width / r.aspectRatio
	}
	return r.RenderProxyBox.ComputeMinIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is the width over the ratio, or the child's when the
// width is infinite.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeMaxIntrinsicHeight
func (r *RenderAspectRatio) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !math.IsInf(width, 0) {
		return width / r.aspectRatio
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicHeight(width)
}

func (r *RenderAspectRatio) applyAspectRatio(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		constraints.DebugAssertIsValid()
		if !constraints.HasBoundedWidth() && !constraints.HasBoundedHeight() {
			panic(errors.New("AspectRatioUnbounded", foundation.RuntimeTypeName(r.self)+
				" was given an aspect ratio but also both unbounded width and unbounded height constraints; "+
				"because both constraints were unbounded, this render object does not know how much size to consume",
				"object", foundation.DescribeIdentity(r.self), "aspectRatio", r.aspectRatio))
		}
	}
	if constraints.IsTight() {
		return constraints.Smallest()
	}
	width := constraints.MaxWidth
	var height float64
	if !math.IsInf(width, 0) {
		height = width / r.aspectRatio
	} else {
		height = constraints.MaxHeight
		width = height * r.aspectRatio
	}
	if width > constraints.MaxWidth {
		width = constraints.MaxWidth
		height = width / r.aspectRatio
	}
	if height > constraints.MaxHeight {
		height = constraints.MaxHeight
		width = height * r.aspectRatio
	}
	if width < constraints.MinWidth {
		width = constraints.MinWidth
		height = width / r.aspectRatio
	}
	if height < constraints.MinHeight {
		height = constraints.MinHeight
		width = height * r.aspectRatio
	}
	return constraints.Constrain(ui.Size{Width: width, Height: height})
}

// ComputeDryLayout is the size that fits the aspect ratio.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeDryLayout
func (r *RenderAspectRatio) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.applyAspectRatio(constraints)
}

// ComputeDryBaseline is the child's baseline for tight constraints of the size.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.computeDryBaseline
func (r *RenderAspectRatio) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	return r.RenderProxyBox.ComputeDryBaseline(NewBoxConstraintsTight(r.GetDryLayout(constraints)), baseline)
}

// PerformLayout sizes the box to the ratio and lays the child out tight in it.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.performLayout
func (r *RenderAspectRatio) PerformLayout() {
	r.SetSize(r.GetDryLayout(r.BoxConstraints()))
	if r.hasChild() {
		r.Child().Layout(NewBoxConstraintsTight(r.Size()))
	}
}

// DebugFillProperties adds the aspect ratio.
//
// Flutter: rendering/proxy_box.dart RenderAspectRatio.debugFillProperties
func (r *RenderAspectRatio) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("aspectRatio", foundation.Some(r.aspectRatio)))
}

func applyStep(input float64, step foundation.Opt[float64]) float64 {
	s, ok := step.Get()
	if !ok {
		return input
	}
	return math.Ceil(input/s) * s
}

// RenderIntrinsicWidth sizes its child to the child's maximum intrinsic width.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth
type RenderIntrinsicWidth struct {
	RenderProxyBox
	stepWidth, stepHeight foundation.Opt[float64]
}

// RenderIntrinsicWidthOpts are the optional parameters of NewRenderIntrinsicWidth.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.new
type RenderIntrinsicWidthOpts struct {
	// StepWidth rounds the width up to a multiple of it, if set.
	StepWidth foundation.Opt[float64]
	// StepHeight rounds the height up to a multiple of it, if set.
	StepHeight foundation.Opt[float64]
}

// NewRenderIntrinsicWidth creates the box. A step that is not positive panics
// with IntrinsicStepInvalid in debug builds.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.new
func NewRenderIntrinsicWidth(child Box, opts ...RenderIntrinsicWidthOpts) *RenderIntrinsicWidth {
	o := firstOpts(opts)
	assertStep(o.StepWidth)
	assertStep(o.StepHeight)
	r := &RenderIntrinsicWidth{stepWidth: o.StepWidth, stepHeight: o.StepHeight}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

func assertStep(step foundation.Opt[float64]) {
	if v, ok := step.Get(); ok && foundation.DebugMode && !(v > 0) {
		panic(errors.New("IntrinsicStepInvalid", "the step of an intrinsic size must be positive", "step", v))
	}
}

// StepWidth is the step that the width is a multiple of.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.stepWidth
func (r *RenderIntrinsicWidth) StepWidth() foundation.Opt[float64] { return r.stepWidth }

// SetStepWidth changes the step and marks the box as needing layout.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.stepWidth
func (r *RenderIntrinsicWidth) SetStepWidth(value foundation.Opt[float64]) {
	assertStep(value)
	if value == r.stepWidth {
		return
	}
	r.stepWidth = value
	r.MarkNeedsLayout()
}

// StepHeight is the step that the height is a multiple of.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.stepHeight
func (r *RenderIntrinsicWidth) StepHeight() foundation.Opt[float64] { return r.stepHeight }

// SetStepHeight changes the step and marks the box as needing layout.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.stepHeight
func (r *RenderIntrinsicWidth) SetStepHeight(value foundation.Opt[float64]) {
	assertStep(value)
	if value == r.stepHeight {
		return
	}
	r.stepHeight = value
	r.MarkNeedsLayout()
}

// ComputeMinIntrinsicWidth is the maximum intrinsic width.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeMinIntrinsicWidth
func (r *RenderIntrinsicWidth) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.GetMaxIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the child's maximum intrinsic width rounded up to
// the step.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeMaxIntrinsicWidth
func (r *RenderIntrinsicWidth) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	return applyStep(r.Child().GetMaxIntrinsicWidth(height), r.stepWidth)
}

// ComputeMinIntrinsicHeight is the child's minimum intrinsic height at the
// intrinsic width, rounded up to the step.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeMinIntrinsicHeight
func (r *RenderIntrinsicWidth) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if math.IsInf(width, 0) {
		width = r.GetMaxIntrinsicWidth(math.Inf(1))
	}
	return applyStep(r.Child().GetMinIntrinsicHeight(width), r.stepHeight)
}

// ComputeMaxIntrinsicHeight is the child's maximum intrinsic height at the
// intrinsic width, rounded up to the step.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeMaxIntrinsicHeight
func (r *RenderIntrinsicWidth) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if math.IsInf(width, 0) {
		width = r.GetMaxIntrinsicWidth(math.Inf(1))
	}
	return applyStep(r.Child().GetMaxIntrinsicHeight(width), r.stepHeight)
}

func (r *RenderIntrinsicWidth) childConstraints(child Box, c BoxConstraints) BoxConstraints {
	var o TightenOpts
	if !c.HasTightWidth() {
		o.Width = foundation.Some(applyStep(child.GetMaxIntrinsicWidth(c.MaxHeight), r.stepWidth))
	}
	if r.stepHeight.IsSome() {
		o.Height = foundation.Some(applyStep(child.GetMaxIntrinsicHeight(c.MaxWidth), r.stepHeight))
	}
	return c.Tighten(o)
}

func (r *RenderIntrinsicWidth) computeSize(layoutChild ChildLayouter, c BoxConstraints) ui.Size {
	if !r.hasChild() {
		return c.Smallest()
	}
	return layoutChild(r.Child(), r.childConstraints(r.Child(), c))
}

// ComputeDryLayout is the size for the constraints.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeDryLayout
func (r *RenderIntrinsicWidth) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.computeSize(ChildLayoutHelper.DryLayoutChild, constraints)
}

// ComputeDryBaseline is the child's baseline for the intrinsic constraints.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.computeDryBaseline
func (r *RenderIntrinsicWidth) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	return r.Child().GetDryBaseline(r.childConstraints(r.Child(), constraints), baseline)
}

// PerformLayout lays out the child at its intrinsic width.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.performLayout
func (r *RenderIntrinsicWidth) PerformLayout() {
	r.SetSize(r.computeSize(ChildLayoutHelper.LayoutChild, r.BoxConstraints()))
}

// DebugFillProperties adds the steps.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicWidth.debugFillProperties
func (r *RenderIntrinsicWidth) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("stepWidth", r.stepWidth))
	b.Add(foundation.NewDoubleProperty("stepHeight", r.stepHeight))
}

// RenderIntrinsicHeight sizes its child to the child's intrinsic height.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight
type RenderIntrinsicHeight struct {
	RenderProxyBox
}

// NewRenderIntrinsicHeight creates the box.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.new
func NewRenderIntrinsicHeight(child Box) *RenderIntrinsicHeight {
	r := &RenderIntrinsicHeight{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// ComputeMinIntrinsicWidth is the child's minimum intrinsic width at its
// intrinsic height.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.computeMinIntrinsicWidth
func (r *RenderIntrinsicHeight) ComputeMinIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if math.IsInf(height, 0) {
		height = r.Child().GetMaxIntrinsicHeight(math.Inf(1))
	}
	return r.Child().GetMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the child's maximum intrinsic width at its
// intrinsic height.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.computeMaxIntrinsicWidth
func (r *RenderIntrinsicHeight) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if math.IsInf(height, 0) {
		height = r.Child().GetMaxIntrinsicHeight(math.Inf(1))
	}
	return r.Child().GetMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the maximum intrinsic height.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.computeMinIntrinsicHeight
func (r *RenderIntrinsicHeight) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.GetMaxIntrinsicHeight(width)
}

func (r *RenderIntrinsicHeight) childConstraints(child Box, c BoxConstraints) BoxConstraints {
	if c.HasTightHeight() {
		return c
	}
	return c.Tighten(TightenOpts{Height: foundation.Some(child.GetMaxIntrinsicHeight(c.MaxWidth))})
}

func (r *RenderIntrinsicHeight) computeSize(layoutChild ChildLayouter, c BoxConstraints) ui.Size {
	if !r.hasChild() {
		return c.Smallest()
	}
	return layoutChild(r.Child(), r.childConstraints(r.Child(), c))
}

// ComputeDryLayout is the size for the constraints.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.computeDryLayout
func (r *RenderIntrinsicHeight) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.computeSize(ChildLayoutHelper.DryLayoutChild, constraints)
}

// ComputeDryBaseline is the child's baseline for the intrinsic constraints.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.computeDryBaseline
func (r *RenderIntrinsicHeight) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	return r.Child().GetDryBaseline(r.childConstraints(r.Child(), constraints), baseline)
}

// PerformLayout lays out the child at its intrinsic height.
//
// Flutter: rendering/proxy_box.dart RenderIntrinsicHeight.performLayout
func (r *RenderIntrinsicHeight) PerformLayout() {
	r.SetSize(r.computeSize(ChildLayoutHelper.LayoutChild, r.BoxConstraints()))
}

// RenderIgnoreBaseline makes its child's baseline unavailable to the parent.
//
// Flutter: rendering/proxy_box.dart RenderIgnoreBaseline
type RenderIgnoreBaseline struct {
	RenderProxyBox
}

// NewRenderIgnoreBaseline creates the box.
//
// Flutter: rendering/proxy_box.dart RenderIgnoreBaseline.new
func NewRenderIgnoreBaseline(child Box) *RenderIgnoreBaseline {
	r := &RenderIgnoreBaseline{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// ComputeDistanceToActualBaseline is none.
//
// Flutter: rendering/proxy_box.dart RenderIgnoreBaseline.computeDistanceToActualBaseline
func (r *RenderIgnoreBaseline) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	return foundation.None[float64]()
}

// ComputeDryBaseline is none.
//
// Flutter: rendering/proxy_box.dart RenderIgnoreBaseline.computeDryBaseline
func (r *RenderIgnoreBaseline) ComputeDryBaseline(BoxConstraints, ui.TextBaseline) foundation.Opt[float64] {
	return foundation.None[float64]()
}

// RenderOpacity makes its child partially transparent.
//
// Flutter: rendering/proxy_box.dart RenderOpacity
type RenderOpacity struct {
	RenderProxyBox
	opacity                float64
	alpha                  int
	alwaysIncludeSemantics bool
}

// RenderOpacityOpts are the optional parameters of NewRenderOpacity.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.new
type RenderOpacityOpts struct {
	// Opacity defaults to 1.
	Opacity foundation.Opt[float64]
	// AlwaysIncludeSemantics keeps the semantics of the child when it is not
	// visible.
	AlwaysIncludeSemantics bool
}

// NewRenderOpacity creates the box. An opacity outside 0..1 panics with
// OpacityOutOfRange in debug builds.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.new
func NewRenderOpacity(child Box, opts ...RenderOpacityOpts) *RenderOpacity {
	o := firstOpts(opts)
	opacity := o.Opacity.OrElse(1)
	assertOpacity(opacity)
	r := &RenderOpacity{opacity: opacity, alwaysIncludeSemantics: o.AlwaysIncludeSemantics, alpha: ui.ColorGetAlphaFromOpacity(opacity)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

func assertOpacity(v float64) {
	if foundation.DebugMode && !(v >= 0 && v <= 1) {
		panic(errors.New("OpacityOutOfRange", "the opacity must be between 0 and 1", "opacity", v))
	}
}

// AlwaysNeedsCompositing is true if there is a child and the opacity is not 0.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.alwaysNeedsCompositing
func (r *RenderOpacity) AlwaysNeedsCompositing() bool { return r.hasChild() && r.alpha > 0 }

// IsRepaintBoundary is true when it AlwaysNeedsCompositing.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.isRepaintBoundary
func (r *RenderOpacity) IsRepaintBoundary() bool { return r.AlwaysNeedsCompositing() }

// Opacity is the fraction to scale the child's alpha value by.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.opacity
func (r *RenderOpacity) Opacity() float64 { return r.opacity }

// SetOpacity changes the opacity; the layer is updated, not repainted.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.opacity
func (r *RenderOpacity) SetOpacity(value float64) {
	assertOpacity(value)
	if r.opacity == value {
		return
	}
	didNeedCompositing := r.AlwaysNeedsCompositing()
	wasVisible := r.alpha != 0
	r.opacity = value
	r.alpha = ui.ColorGetAlphaFromOpacity(value)
	if didNeedCompositing != r.AlwaysNeedsCompositing() {
		r.MarkNeedsCompositingBitsUpdate()
	}
	r.MarkNeedsCompositedLayerUpdate()
	if wasVisible != (r.alpha != 0) && !r.alwaysIncludeSemantics {
		r.MarkNeedsSemanticsUpdate()
	}
}

// AlwaysIncludeSemantics is whether the semantics of the child are kept when the
// child is not visible.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.alwaysIncludeSemantics
func (r *RenderOpacity) AlwaysIncludeSemantics() bool { return r.alwaysIncludeSemantics }

// SetAlwaysIncludeSemantics changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.alwaysIncludeSemantics
func (r *RenderOpacity) SetAlwaysIncludeSemantics(value bool) {
	if value == r.alwaysIncludeSemantics {
		return
	}
	r.alwaysIncludeSemantics = value
	r.MarkNeedsSemanticsUpdate()
}

// VisitChildrenForSemantics visits the child only if it is visible or
// AlwaysIncludeSemantics.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.visitChildrenForSemantics
func (r *RenderOpacity) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.hasChild() && (r.alpha != 0 || r.alwaysIncludeSemantics) {
		visitor(r.Child())
	}
}

// PaintsChild is true if the opacity is not 0.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.paintsChild
func (r *RenderOpacity) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.Parent() != RenderObject(r.self) {
		panic(errors.New("RenderObjectNotChild", "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
	}
	return r.alpha > 0
}

// UpdateCompositedLayer sets the alpha of the opacity layer.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.updateCompositedLayer
func (r *RenderOpacity) UpdateCompositedLayer(oldLayer OffsetLayerLike) OffsetLayerLike {
	layer, _ := oldLayer.(*OpacityLayer)
	if layer == nil {
		layer = NewOpacityLayer()
	}
	layer.SetAlpha(r.alpha)
	return layer
}

// Paint paints the child unless it is invisible.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.paint
func (r *RenderOpacity) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() || r.alpha == 0 {
		return
	}
	r.RenderProxyBox.Paint(context, offset)
}

// DebugFillProperties adds the opacity and the semantics flag.
//
// Flutter: rendering/proxy_box.dart RenderOpacity.debugFillProperties
func (r *RenderOpacity) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("opacity", foundation.Some(r.opacity)))
	b.Add(foundation.NewFlagProperty("alwaysIncludeSemantics", foundation.Some(r.alwaysIncludeSemantics),
		foundation.FlagPropertyOpts{IfTrue: "alwaysIncludeSemantics"}))
}

// RenderAnimatedOpacity makes its child partially transparent, driven by an
// animation: the opacity layer is updated when the animation changes, and the
// child is not repainted.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacity
type RenderAnimatedOpacity struct {
	RenderProxyBox
	opacity                 animation.Animation[float64]
	opacityHandle           foundation.ListenerHandle
	alpha                   foundation.Opt[int]
	currentlyIsRepaintBound foundation.Opt[bool]
	alwaysIncludeSemantics  bool
}

// RenderAnimatedOpacityOpts are the optional parameters of
// NewRenderAnimatedOpacity.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacity.new
type RenderAnimatedOpacityOpts struct {
	// AlwaysIncludeSemantics keeps the semantics of the child when it is not
	// visible.
	AlwaysIncludeSemantics bool
}

// NewRenderAnimatedOpacity creates the box. The opacity animation drives the
// alpha of the child once the box is attached; before that, the alpha is read
// once, when the box is created.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacity.new
func NewRenderAnimatedOpacity(opacity animation.Animation[float64], child Box, opts ...RenderAnimatedOpacityOpts) *RenderAnimatedOpacity {
	o := firstOpts(opts)
	r := &RenderAnimatedOpacity{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	r.SetOpacity(opacity)
	r.alwaysIncludeSemantics = o.AlwaysIncludeSemantics
	return r
}

// IsRepaintBoundary is true if there is a child and the alpha is not 0.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.isRepaintBoundary
func (r *RenderAnimatedOpacity) IsRepaintBoundary() bool {
	return r.hasChild() && r.currentlyIsRepaintBound.OrElse(false)
}

// UpdateCompositedLayer sets the alpha of the opacity layer.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.updateCompositedLayer
func (r *RenderAnimatedOpacity) UpdateCompositedLayer(oldLayer OffsetLayerLike) OffsetLayerLike {
	layer, _ := oldLayer.(*OpacityLayer)
	if layer == nil {
		layer = NewOpacityLayer()
	}
	layer.SetAlpha(r.alpha.OrElse(0))
	return layer
}

// Opacity is the animation that drives the opacity of the child.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.opacity
func (r *RenderAnimatedOpacity) Opacity() animation.Animation[float64] { return r.opacity }

// SetOpacity changes the animation. The box listens to it while it is attached.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.opacity
func (r *RenderAnimatedOpacity) SetOpacity(value animation.Animation[float64]) {
	if r.opacity == value {
		return
	}
	if r.Attached() && r.opacity != nil {
		r.opacity.RemoveListener(r.opacityHandle)
	}
	r.opacity = value
	if r.Attached() {
		r.opacityHandle = r.opacity.AddListener(r.updateOpacity)
	}
	r.updateOpacity()
}

// AlwaysIncludeSemantics is whether the semantics of the child are kept when the
// child is not visible.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.alwaysIncludeSemantics
func (r *RenderAnimatedOpacity) AlwaysIncludeSemantics() bool { return r.alwaysIncludeSemantics }

// SetAlwaysIncludeSemantics changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.alwaysIncludeSemantics
func (r *RenderAnimatedOpacity) SetAlwaysIncludeSemantics(value bool) {
	if value == r.alwaysIncludeSemantics {
		return
	}
	r.alwaysIncludeSemantics = value
	r.MarkNeedsSemanticsUpdate()
}

// Attach starts listening to the opacity animation.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.attach
func (r *RenderAnimatedOpacity) Attach(owner *PipelineOwner) {
	r.RenderProxyBox.Attach(owner)
	r.opacityHandle = r.opacity.AddListener(r.updateOpacity)
	r.updateOpacity() // in case it changed while we weren't listening
}

// Detach stops listening to the opacity animation.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.detach
func (r *RenderAnimatedOpacity) Detach() {
	r.opacity.RemoveListener(r.opacityHandle)
	r.opacityHandle = foundation.ListenerHandle{}
	r.RenderProxyBox.Detach()
}

func (r *RenderAnimatedOpacity) updateOpacity() {
	oldAlpha, hadAlpha := r.alpha.Get()
	newAlpha := ui.ColorGetAlphaFromOpacity(r.opacity.Value())
	r.alpha = foundation.Some(newAlpha)
	if !hadAlpha || oldAlpha != newAlpha {
		wasRepaintBoundary := r.currentlyIsRepaintBound
		r.currentlyIsRepaintBound = foundation.Some(newAlpha > 0)
		if r.hasChild() && wasRepaintBoundary != r.currentlyIsRepaintBound {
			r.MarkNeedsCompositingBitsUpdate()
		}
		r.MarkNeedsCompositedLayerUpdate()
		if oldAlpha == 0 || newAlpha == 0 {
			r.MarkNeedsSemanticsUpdate()
		}
	}
}

// PaintsChild is true if the opacity is not 0.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.paintsChild
func (r *RenderAnimatedOpacity) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.Parent() != RenderObject(r.self) {
		panic(errors.New("RenderObjectNotChild", "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
	}
	return r.opacity.Value() > 0
}

// Paint paints the child unless it is invisible.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.paint
func (r *RenderAnimatedOpacity) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() || r.alpha.OrElse(0) == 0 {
		return
	}
	r.RenderProxyBox.Paint(context, offset)
}

// VisitChildrenForSemantics visits the child only if it is visible or
// AlwaysIncludeSemantics.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.visitChildrenForSemantics
func (r *RenderAnimatedOpacity) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.hasChild() && (r.alpha.OrElse(0) != 0 || r.alwaysIncludeSemantics) {
		visitor(r.Child())
	}
}

// DebugFillProperties adds the opacity animation and the semantics flag.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.debugFillProperties
func (r *RenderAnimatedOpacity) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("opacity", r.opacity))
	b.Add(foundation.NewFlagProperty("alwaysIncludeSemantics", foundation.Some(r.alwaysIncludeSemantics),
		foundation.FlagPropertyOpts{IfTrue: "alwaysIncludeSemantics"}))
}

// ShaderCallback creates the shader for the bounds.
//
// Flutter: rendering/proxy_box.dart ShaderCallback
type ShaderCallback func(bounds ui.Rect) ui.Shader

// RenderShaderMask applies a mask generated by a shader to its child.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask
type RenderShaderMask struct {
	RenderProxyBox
	shaderCallback ShaderCallback
	blendMode      ui.BlendMode
}

// RenderShaderMaskOpts are the optional parameters of NewRenderShaderMask.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.new
type RenderShaderMaskOpts struct {
	// BlendMode defaults to BlendModeModulate.
	BlendMode ui.BlendMode
}

// NewRenderShaderMask creates the box.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.new
func NewRenderShaderMask(shaderCallback ShaderCallback, child Box, opts ...RenderShaderMaskOpts) *RenderShaderMask {
	o := firstOpts(opts)
	if o.BlendMode == ui.BlendModeUnspecified {
		o.BlendMode = ui.BlendModeModulate
	}
	r := &RenderShaderMask{shaderCallback: shaderCallback, blendMode: o.BlendMode}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// SetShaderCallback changes the callback and marks the box as needing paint.
//
// Deviation: Dart compares the callbacks and skips an equal one; Go functions
// can not be compared, so the box is always repainted (SPEC §12.10).
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.shaderCallback
func (r *RenderShaderMask) SetShaderCallback(value ShaderCallback) {
	r.shaderCallback = value
	r.MarkNeedsPaint()
}

// BlendMode is how the shader is blended with the child.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.blendMode
func (r *RenderShaderMask) BlendMode() ui.BlendMode { return r.blendMode }

// SetBlendMode changes the blend mode and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.blendMode
func (r *RenderShaderMask) SetBlendMode(value ui.BlendMode) {
	if r.blendMode == value {
		return
	}
	r.blendMode = value
	r.MarkNeedsPaint()
}

// AlwaysNeedsCompositing is true if there is a child.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.alwaysNeedsCompositing
func (r *RenderShaderMask) AlwaysNeedsCompositing() bool { return r.hasChild() }

// Paint pushes a shader mask layer around the child.
//
// Flutter: rendering/proxy_box.dart RenderShaderMask.paint
func (r *RenderShaderMask) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	layer := layerAs[*ShaderMaskLayer](r.Layer())
	if layer == nil {
		layer = NewShaderMaskLayer()
		r.SetLayer(layer)
	}
	layer.SetShader(r.shaderCallback(ui.OffsetZero.And(r.Size())))
	layer.SetMaskRect(offset.And(r.Size()))
	layer.SetBlendMode(r.blendMode)
	context.PushLayer(layer, r.RenderProxyBox.Paint, offset, foundation.None[ui.Rect]())
}

// RenderBackdropFilter applies a filter to the existing painted content and
// then paints the child.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter
type RenderBackdropFilter struct {
	RenderProxyBox
	filter      ui.ImageFilter
	blendMode   ui.BlendMode
	enabled     bool
	backdropKey foundation.Opt[BackdropKey]
}

// RenderBackdropFilterOpts are the optional parameters of NewRenderBackdropFilter.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.new
type RenderBackdropFilterOpts struct {
	// BlendMode defaults to BlendModeSrcOver.
	BlendMode ui.BlendMode
	// Enabled defaults to true.
	Enabled foundation.Opt[bool]
	// BackdropKey shares the backdrop with other filters that have the same key.
	BackdropKey foundation.Opt[BackdropKey]
}

// NewRenderBackdropFilter creates the box.
//
// Deviation: Flutter takes an ImageFilterConfig (a filter that is created from
// the bounds); this takes the filter (SPEC §12.10).
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.new
func NewRenderBackdropFilter(filter ui.ImageFilter, child Box, opts ...RenderBackdropFilterOpts) *RenderBackdropFilter {
	o := firstOpts(opts)
	if o.BlendMode == ui.BlendModeUnspecified {
		o.BlendMode = ui.BlendModeSrcOver
	}
	r := &RenderBackdropFilter{filter: filter, blendMode: o.BlendMode, enabled: o.Enabled.OrElse(true), backdropKey: o.BackdropKey}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Enabled is whether the filter is applied.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.enabled
func (r *RenderBackdropFilter) Enabled() bool { return r.enabled }

// SetEnabled changes the flag and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.enabled
func (r *RenderBackdropFilter) SetEnabled(value bool) {
	if r.enabled == value {
		return
	}
	r.enabled = value
	r.MarkNeedsPaint()
}

// Filter is the image filter that is applied to the backdrop.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.filter
func (r *RenderBackdropFilter) Filter() ui.ImageFilter { return r.filter }

// SetFilter changes the filter and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.filter
func (r *RenderBackdropFilter) SetFilter(value ui.ImageFilter) {
	if r.filter == value {
		return
	}
	r.filter = value
	r.MarkNeedsPaint()
}

// BlendMode is the blend mode of the filtered backdrop.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.blendMode
func (r *RenderBackdropFilter) BlendMode() ui.BlendMode { return r.blendMode }

// SetBlendMode changes the blend mode and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.blendMode
func (r *RenderBackdropFilter) SetBlendMode(value ui.BlendMode) {
	if r.blendMode == value {
		return
	}
	r.blendMode = value
	r.MarkNeedsPaint()
}

// BackdropKey is the key that shares the backdrop between filters.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.backdropKey
func (r *RenderBackdropFilter) BackdropKey() foundation.Opt[BackdropKey] { return r.backdropKey }

// SetBackdropKey changes the key and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.backdropKey
func (r *RenderBackdropFilter) SetBackdropKey(value foundation.Opt[BackdropKey]) {
	if value == r.backdropKey {
		return
	}
	r.backdropKey = value
	r.MarkNeedsPaint()
}

// AlwaysNeedsCompositing is true if there is a child.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.alwaysNeedsCompositing
func (r *RenderBackdropFilter) AlwaysNeedsCompositing() bool { return r.hasChild() }

// Paint pushes a backdrop filter layer around the child.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.paint
func (r *RenderBackdropFilter) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.enabled {
		r.RenderProxyBox.Paint(context, offset)
		return
	}
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	layer := layerAs[*BackdropFilterLayer](r.Layer())
	if layer == nil {
		layer = NewBackdropFilterLayer()
		r.SetLayer(layer)
	}
	layer.SetFilter(r.filter)
	layer.SetBlendMode(r.blendMode)
	layer.SetBackdropKey(r.backdropKey)
	context.PushLayer(layer, r.RenderProxyBox.Paint, offset, foundation.None[ui.Rect]())
}

// DebugFillProperties adds the filter, the blend mode and the flag.
//
// Flutter: rendering/proxy_box.dart RenderBackdropFilter.debugFillProperties
func (r *RenderBackdropFilter) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("filter", r.filter))
	b.Add(foundation.NewEnumProperty("blendMode", foundation.Some(r.blendMode)))
	b.Add(foundation.NewFlagProperty("enabled", foundation.Some(r.enabled), foundation.FlagPropertyOpts{IfTrue: "enabled"}))
}

func itoa(n int) string { return strconv.Itoa(n) }
