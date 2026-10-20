// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// RenderProxySliver is a render sliver that has the geometry of its single
// sliver child, lays it out with the same constraints, and paints and hit tests
// it as is. Embed it in the slivers that change one of these, and call
// InitRenderProxySliver in the constructor.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver
type RenderProxySliver struct {
	RenderSliver
	RenderObjectWithChild[Sliver]
}

// InitRenderProxySliver sets the outer object and the child (nil for none).
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.new
func InitRenderProxySliver(p *RenderProxySliver, self Sliver, child Sliver) {
	InitRenderSliver(&p.RenderSliver, self)
	InitRenderObjectWithChild(&p.RenderObjectWithChild, self)
	if !isNil(child) {
		p.SetChild(child)
	}
}

func (p *RenderProxySliver) hasChild() bool { return !isNil(p.Child()) }

func (p *RenderProxySliver) requireChild() Sliver {
	child := p.Child()
	if foundation.DebugMode && isNil(child) {
		panic(errors.New("ProxySliverChildMissing", "a proxy sliver was laid out without a child",
			"object", foundation.DescribeIdentity(p.self)))
	}
	return child
}

// SemanticBounds are the semantic bounds of the child, if there is one.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.semanticBounds
func (p *RenderProxySliver) SemanticBounds() ui.Rect {
	if p.hasChild() {
		return p.Child().SemanticBounds()
	}
	return p.RenderSliver.SemanticBounds()
}

// SetupParentData gives the child a SliverPhysicalParentData if it has none.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.setupParentData
func (p *RenderProxySliver) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(sliverPhysicalHolder); !ok {
		child.AsRenderObject().SetParentData(&SliverPhysicalParentData{})
	}
}

// PerformLayout lays the child out with the constraints and takes its geometry.
// In debug builds it panics with the tag ProxySliverChildMissing without a child.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.performLayout
func (p *RenderProxySliver) PerformLayout() {
	child := p.requireChild()
	child.Layout(p.SliverConstraints(), LayoutOpts{ParentUsesSize: true})
	p.SetGeometry(child.Geometry())
}

// Paint paints the child.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.paint
func (p *RenderProxySliver) Paint(context *PaintingContext, offset ui.Offset) {
	if p.hasChild() {
		context.PaintChild(p.Child(), offset)
	}
}

// HitTestChildren hit tests the child, if its hit test extent reaches.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.hitTestChildren
func (p *RenderProxySliver) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	child := p.Child()
	return !isNil(child) && child.Geometry().HitTestExtent > 0 && child.HitTest(result, mainAxisPosition, crossAxisPosition)
}

// ChildMainAxisPosition is zero.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.childMainAxisPosition
func (p *RenderProxySliver) ChildMainAxisPosition(RenderObject) float64 { return 0 }

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/proxy_sliver.dart RenderProxySliver.applyPaintTransform
func (p *RenderProxySliver) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// RenderSliverOpacity makes its sliver child partially transparent.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity
type RenderSliverOpacity struct {
	RenderProxySliver
	opacity                float64
	alwaysIncludeSemantics bool
	alpha                  int
}

// RenderSliverOpacityOpts are the optional parameters of NewRenderSliverOpacity.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.new
type RenderSliverOpacityOpts struct {
	// Opacity defaults to 1.
	Opacity foundation.Opt[float64]
	// AlwaysIncludeSemantics keeps the semantics of the child when it is not
	// visible.
	AlwaysIncludeSemantics bool
}

// NewRenderSliverOpacity creates the sliver; the child may be nil. An opacity
// outside 0..1 panics with OpacityOutOfRange in debug builds.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.new
func NewRenderSliverOpacity(sliver Sliver, opts ...RenderSliverOpacityOpts) *RenderSliverOpacity {
	o := firstOpts(opts)
	opacity := o.Opacity.OrElse(1)
	assertOpacity(opacity)
	r := &RenderSliverOpacity{opacity: opacity, alwaysIncludeSemantics: o.AlwaysIncludeSemantics, alpha: ui.ColorGetAlphaFromOpacity(opacity)}
	InitRenderProxySliver(&r.RenderProxySliver, r, sliver)
	return r
}

// AlwaysNeedsCompositing is true if there is a child and the opacity is not 0.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.alwaysNeedsCompositing
func (r *RenderSliverOpacity) AlwaysNeedsCompositing() bool { return r.hasChild() && r.alpha > 0 }

// Opacity is the fraction to scale the child's alpha value by.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.opacity
func (r *RenderSliverOpacity) Opacity() float64 { return r.opacity }

// SetOpacity changes the opacity and marks the sliver as needing paint.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.opacity
func (r *RenderSliverOpacity) SetOpacity(value float64) {
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
	r.MarkNeedsPaint()
	if wasVisible != (r.alpha != 0) && !r.alwaysIncludeSemantics {
		r.MarkNeedsSemanticsUpdate()
	}
}

// AlwaysIncludeSemantics is whether the semantics of the child are kept when the
// child is not visible.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.alwaysIncludeSemantics
func (r *RenderSliverOpacity) AlwaysIncludeSemantics() bool { return r.alwaysIncludeSemantics }

// SetAlwaysIncludeSemantics changes the flag.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.alwaysIncludeSemantics
func (r *RenderSliverOpacity) SetAlwaysIncludeSemantics(value bool) {
	if value == r.alwaysIncludeSemantics {
		return
	}
	r.alwaysIncludeSemantics = value
	r.MarkNeedsSemanticsUpdate()
}

// Paint paints the child in an opacity layer.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.paint
func (r *RenderSliverOpacity) Paint(context *PaintingContext, offset ui.Offset) {
	child := r.Child()
	if isNil(child) || !child.Geometry().Visible {
		return
	}
	if r.alpha == 0 {
		// No need to keep the layer. We'll create a new one if necessary.
		r.SetLayer(nil)
		return
	}
	layer := context.PushOpacity(offset, r.alpha, r.RenderProxySliver.Paint, layerAs[*OpacityLayer](r.Layer()))
	if foundation.DebugMode {
		layer.DebugCreator = r.DebugCreator
	}
	setObjectLayer(&r.RenderObjectBase, layer)
}

// VisitChildrenForSemantics visits the child only if it is visible or
// AlwaysIncludeSemantics.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.visitChildrenForSemantics
func (r *RenderSliverOpacity) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.hasChild() && (r.alpha != 0 || r.alwaysIncludeSemantics) {
		visitor(r.Child())
	}
}

// DebugFillProperties adds the opacity and the semantics flag.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOpacity.debugFillProperties
func (r *RenderSliverOpacity) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxySliver.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("opacity", foundation.Some(r.opacity)))
	b.Add(foundation.NewFlagProperty("alwaysIncludeSemantics", foundation.Some(r.alwaysIncludeSemantics),
		foundation.FlagPropertyOpts{IfTrue: "alwaysIncludeSemantics"}))
}

// RenderSliverIgnorePointer ignores pointer events in its sliver child while
// ignoring is true.
//
// Deviation: the deprecated ignoringSemantics parameter is not ported, so the
// semantics are blocked exactly when the pointer is ignored (SPEC §12.10).
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer
type RenderSliverIgnorePointer struct {
	RenderProxySliver
	ignoring bool
}

// RenderSliverIgnorePointerOpts are the optional parameters of
// NewRenderSliverIgnorePointer.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.new
type RenderSliverIgnorePointerOpts struct {
	// Ignoring defaults to true.
	Ignoring foundation.Opt[bool]
}

// NewRenderSliverIgnorePointer creates the sliver; the child may be nil.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.new
func NewRenderSliverIgnorePointer(sliver Sliver, opts ...RenderSliverIgnorePointerOpts) *RenderSliverIgnorePointer {
	r := &RenderSliverIgnorePointer{ignoring: firstOpts(opts).Ignoring.OrElse(true)}
	InitRenderProxySliver(&r.RenderProxySliver, r, sliver)
	return r
}

// Ignoring is whether pointer events are ignored.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.ignoring
func (r *RenderSliverIgnorePointer) Ignoring() bool { return r.ignoring }

// SetIgnoring changes the flag and marks the sliver as needing a semantics
// update.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.ignoring
func (r *RenderSliverIgnorePointer) SetIgnoring(value bool) {
	if value == r.ignoring {
		return
	}
	r.ignoring = value
	r.MarkNeedsSemanticsUpdate()
}

// HitTest is false while ignoring.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.hitTest
func (r *RenderSliverIgnorePointer) HitTest(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	return !r.ignoring && r.RenderProxySliver.HitTest(result, mainAxisPosition, crossAxisPosition)
}

// DescribeSemanticsConfiguration blocks user actions while ignoring.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.describeSemanticsConfiguration
func (r *RenderSliverIgnorePointer) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxySliver.DescribeSemanticsConfiguration(config)
	config.IsBlockingUserActions = r.ignoring
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverIgnorePointer.debugFillProperties
func (r *RenderSliverIgnorePointer) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxySliver.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("ignoring", r.ignoring))
}

// RenderSliverOffstage lays its sliver child out as if it were visible, but does
// not paint it, does not hit test it and takes no space.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage
type RenderSliverOffstage struct {
	RenderProxySliver
	offstage bool
}

// RenderSliverOffstageOpts are the optional parameters of NewRenderSliverOffstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.new
type RenderSliverOffstageOpts struct {
	// Offstage defaults to true.
	Offstage foundation.Opt[bool]
}

// NewRenderSliverOffstage creates the sliver; the child may be nil.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.new
func NewRenderSliverOffstage(sliver Sliver, opts ...RenderSliverOffstageOpts) *RenderSliverOffstage {
	r := &RenderSliverOffstage{offstage: firstOpts(opts).Offstage.OrElse(true)}
	InitRenderProxySliver(&r.RenderProxySliver, r, sliver)
	return r
}

// Offstage is whether the child is hidden.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.offstage
func (r *RenderSliverOffstage) Offstage() bool { return r.offstage }

// SetOffstage changes the flag and marks the sliver as needing layout.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.offstage
func (r *RenderSliverOffstage) SetOffstage(value bool) {
	if value == r.offstage {
		return
	}
	r.offstage = value
	r.MarkNeedsLayoutForSizedByParentChange()
}

// PerformLayout lays the child out, and takes no space while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.performLayout
func (r *RenderSliverOffstage) PerformLayout() {
	child := r.requireChild()
	child.Layout(r.SliverConstraints(), LayoutOpts{ParentUsesSize: true})
	if r.offstage {
		r.SetGeometry(SliverGeometryZero)
	} else {
		r.SetGeometry(child.Geometry())
	}
}

// HitTest is false while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.hitTest
func (r *RenderSliverOffstage) HitTest(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	return !r.offstage && r.RenderProxySliver.HitTest(result, mainAxisPosition, crossAxisPosition)
}

// HitTestChildren is false while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.hitTestChildren
func (r *RenderSliverOffstage) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	return !r.offstage && r.RenderProxySliver.HitTestChildren(result, mainAxisPosition, crossAxisPosition)
}

// Paint paints nothing while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.paint
func (r *RenderSliverOffstage) Paint(context *PaintingContext, offset ui.Offset) {
	if r.offstage {
		return
	}
	context.PaintChild(r.Child(), offset)
}

// VisitChildrenForSemantics visits nothing while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.visitChildrenForSemantics
func (r *RenderSliverOffstage) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.offstage {
		return
	}
	r.RenderProxySliver.VisitChildrenForSemantics(visitor)
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.debugFillProperties
func (r *RenderSliverOffstage) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxySliver.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("offstage", r.offstage))
}

// DebugDescribeChildren describes the child, as offstage while offstage.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverOffstage.debugDescribeChildren
func (r *RenderSliverOffstage) DebugDescribeChildren() []foundation.DiagnosticsNode {
	if !r.hasChild() {
		return nil
	}
	style := foundation.DiagnosticsTreeStyleSparse
	if r.offstage {
		style = foundation.DiagnosticsTreeStyleOffstage
	}
	return []foundation.DiagnosticsNode{foundation.ToDiagnosticsNode(r.Child(),
		foundation.DiagnosticsNodeOpts{Name: foundation.Some("child"), Style: style})}
}

// RenderSliverAnimatedOpacity makes its sliver child partially transparent,
// driven by an animation: the opacity layer is updated when the animation
// changes, and the child is not repainted.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverAnimatedOpacity
type RenderSliverAnimatedOpacity struct {
	RenderProxySliver
	opacity                 animation.Animation[float64]
	opacityHandle           foundation.ListenerHandle
	alpha                   foundation.Opt[int]
	currentlyIsRepaintBound foundation.Opt[bool]
	alwaysIncludeSemantics  bool
}

// RenderSliverAnimatedOpacityOpts are the optional parameters of
// NewRenderSliverAnimatedOpacity.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverAnimatedOpacity.new
type RenderSliverAnimatedOpacityOpts struct {
	// AlwaysIncludeSemantics keeps the semantics of the child when it is not
	// visible.
	AlwaysIncludeSemantics bool
}

// NewRenderSliverAnimatedOpacity creates the sliver; the child may be nil.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverAnimatedOpacity.new
func NewRenderSliverAnimatedOpacity(opacity animation.Animation[float64], sliver Sliver, opts ...RenderSliverAnimatedOpacityOpts) *RenderSliverAnimatedOpacity {
	r := &RenderSliverAnimatedOpacity{}
	InitRenderProxySliver(&r.RenderProxySliver, r, sliver)
	r.SetOpacity(opacity)
	r.alwaysIncludeSemantics = firstOpts(opts).AlwaysIncludeSemantics
	return r
}

// IsRepaintBoundary is true if there is a child and the alpha is not 0.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.isRepaintBoundary
func (r *RenderSliverAnimatedOpacity) IsRepaintBoundary() bool {
	return r.hasChild() && r.currentlyIsRepaintBound.OrElse(false)
}

// UpdateCompositedLayer sets the alpha of the opacity layer.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.updateCompositedLayer
func (r *RenderSliverAnimatedOpacity) UpdateCompositedLayer(oldLayer OffsetLayerLike) OffsetLayerLike {
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
func (r *RenderSliverAnimatedOpacity) Opacity() animation.Animation[float64] { return r.opacity }

// SetOpacity changes the animation. The sliver listens to it while it is attached.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.opacity
func (r *RenderSliverAnimatedOpacity) SetOpacity(value animation.Animation[float64]) {
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
func (r *RenderSliverAnimatedOpacity) AlwaysIncludeSemantics() bool { return r.alwaysIncludeSemantics }

// SetAlwaysIncludeSemantics changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.alwaysIncludeSemantics
func (r *RenderSliverAnimatedOpacity) SetAlwaysIncludeSemantics(value bool) {
	if value == r.alwaysIncludeSemantics {
		return
	}
	r.alwaysIncludeSemantics = value
	r.MarkNeedsSemanticsUpdate()
}

// Attach starts listening to the opacity animation.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.attach
func (r *RenderSliverAnimatedOpacity) Attach(owner *PipelineOwner) {
	r.RenderProxySliver.Attach(owner)
	r.opacityHandle = r.opacity.AddListener(r.updateOpacity)
	r.updateOpacity() // in case it changed while we weren't listening
}

// Detach stops listening to the opacity animation.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.detach
func (r *RenderSliverAnimatedOpacity) Detach() {
	r.opacity.RemoveListener(r.opacityHandle)
	r.opacityHandle = foundation.ListenerHandle{}
	r.RenderProxySliver.Detach()
}

func (r *RenderSliverAnimatedOpacity) updateOpacity() {
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
func (r *RenderSliverAnimatedOpacity) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.Parent() != RenderObject(r.self) {
		panic(errors.New("RenderObjectNotChild", "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
	}
	return r.opacity.Value() > 0
}

// Paint paints the child unless it is invisible.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.paint
func (r *RenderSliverAnimatedOpacity) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() || r.alpha.OrElse(0) == 0 {
		return
	}
	r.RenderProxySliver.Paint(context, offset)
}

// VisitChildrenForSemantics visits the child only if it is visible or
// AlwaysIncludeSemantics.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.visitChildrenForSemantics
func (r *RenderSliverAnimatedOpacity) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.hasChild() && (r.alpha.OrElse(0) != 0 || r.alwaysIncludeSemantics) {
		visitor(r.Child())
	}
}

// DebugFillProperties adds the opacity animation and the semantics flag.
//
// Flutter: rendering/proxy_box.dart RenderAnimatedOpacityMixin.debugFillProperties
func (r *RenderSliverAnimatedOpacity) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxySliver.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("opacity", r.opacity))
	b.Add(foundation.NewFlagProperty("alwaysIncludeSemantics", foundation.Some(r.alwaysIncludeSemantics),
		foundation.FlagPropertyOpts{IfTrue: "alwaysIncludeSemantics"}))
}

// RenderSliverConstrainedCrossAxis limits the cross axis extent of its sliver
// child to a maximum.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverConstrainedCrossAxis
type RenderSliverConstrainedCrossAxis struct {
	RenderProxySliver
	maxExtent float64
}

// NewRenderSliverConstrainedCrossAxis creates the sliver; the child may be nil.
// In debug builds a negative maximum panics with the tag SliverMaxExtentNegative.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverConstrainedCrossAxis.new
func NewRenderSliverConstrainedCrossAxis(maxExtent float64, sliver Sliver) *RenderSliverConstrainedCrossAxis {
	assertSliverMaxExtent(maxExtent)
	r := &RenderSliverConstrainedCrossAxis{maxExtent: maxExtent}
	InitRenderProxySliver(&r.RenderProxySliver, r, sliver)
	return r
}

func assertSliverMaxExtent(v float64) {
	if foundation.DebugMode && !(v >= 0) {
		panic(errors.New("SliverMaxExtentNegative", "the maximum extent must not be negative", "maxExtent", v))
	}
}

// MaxExtent is the maximum cross axis extent.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverConstrainedCrossAxis.maxExtent
func (r *RenderSliverConstrainedCrossAxis) MaxExtent() float64 { return r.maxExtent }

// SetMaxExtent changes the maximum and marks the sliver as needing layout.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverConstrainedCrossAxis.maxExtent
func (r *RenderSliverConstrainedCrossAxis) SetMaxExtent(value float64) {
	if r.maxExtent == value {
		return
	}
	r.maxExtent = value
	r.MarkNeedsLayout()
}

// PerformLayout lays the child out with the limited cross axis extent.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverConstrainedCrossAxis.performLayout
func (r *RenderSliverConstrainedCrossAxis) PerformLayout() {
	child := r.requireChild()
	assertSliverMaxExtent(r.maxExtent)
	c := r.SliverConstraints()
	extent := math.Min(r.maxExtent, c.CrossAxisExtent)
	child.Layout(c.CopyWith(SliverConstraintsCopy{CrossAxisExtent: foundation.Some(extent)}), LayoutOpts{ParentUsesSize: true})
	r.SetGeometry(child.Geometry().CopyWith(SliverGeometryCopy{CrossAxisExtent: foundation.Some(extent)}))
}

// RenderSliverSemanticsAnnotations adds annotations to the semantics node of its
// sliver child's subtree.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverSemanticsAnnotations
type RenderSliverSemanticsAnnotations struct {
	RenderProxySliver
	SemanticsAnnotations
}

// NewRenderSliverSemanticsAnnotations creates the sliver; the child may be nil.
//
// Flutter: rendering/proxy_sliver.dart RenderSliverSemanticsAnnotations.new
func NewRenderSliverSemanticsAnnotations(child Sliver, properties *semantics.SemanticsProperties, opts ...SemanticsAnnotationsOpts) *RenderSliverSemanticsAnnotations {
	r := &RenderSliverSemanticsAnnotations{}
	InitRenderProxySliver(&r.RenderProxySliver, r, child)
	InitSemanticsAnnotations(&r.SemanticsAnnotations, r, properties, opts...)
	return r
}
