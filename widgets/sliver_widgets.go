package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// SliverOpacityOpts are the optional parameters of NewSliverOpacity.
//
// Flutter: widgets/sliver.dart SliverOpacity.new
type SliverOpacityOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AlwaysIncludeSemantics is whether the semantics of the sliver are included
	// even when it is invisible (opacity 0).
	AlwaysIncludeSemantics bool
}

// SliverOpacity makes its sliver child partially transparent.
//
// Flutter: widgets/sliver.dart SliverOpacity
type SliverOpacity struct {
	SingleChildRenderObjectWidget
	opacity                float64
	alwaysIncludeSemantics bool
}

// NewSliverOpacity creates a widget that paints its sliver with the opacity, from
// 0 to 1. In debug builds it panics with the tag OpacityOutOfRange when the
// opacity is outside of that range.
//
// Flutter: widgets/sliver.dart SliverOpacity.new
func NewSliverOpacity(opacity float64, sliver Widget, opts ...SliverOpacityOpts) *SliverOpacity {
	o := firstOpts(opts)
	if !(opacity >= 0 && opacity <= 1) {
		panic(errors.New("OpacityOutOfRange", "the opacity must be between 0.0 and 1.0", "opacity", opacity))
	}
	return &SliverOpacity{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: o.Key}), opacity, o.AlwaysIncludeSemantics}
}

// Opacity is the fraction to scale the sliver's alpha value by.
//
// Flutter: widgets/sliver.dart SliverOpacity.opacity
func (w *SliverOpacity) Opacity() float64 { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics are included when invisible.
//
// Flutter: widgets/sliver.dart SliverOpacity.alwaysIncludeSemantics
func (w *SliverOpacity) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverOpacity.createRenderObject
func (w *SliverOpacity) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverOpacity(nil, rendering.RenderSliverOpacityOpts{
		Opacity: foundation.Some(w.opacity), AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/sliver.dart SliverOpacity.updateRenderObject
func (w *SliverOpacity) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSliverOpacity)
	r.SetOpacity(w.opacity)
	r.SetAlwaysIncludeSemantics(w.alwaysIncludeSemantics)
}

// SliverIgnorePointerOpts are the optional parameters of NewSliverIgnorePointer.
//
// Flutter: widgets/sliver.dart SliverIgnorePointer.new
type SliverIgnorePointerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Ignoring is whether this sliver is ignored during hit testing; the default
	// is true.
	Ignoring foundation.Opt[bool]
}

// SliverIgnorePointer is a sliver that is invisible during hit testing.
//
// Deviation: the deprecated ignoringSemantics parameter is not ported (SPEC
// §12.10).
//
// Flutter: widgets/sliver.dart SliverIgnorePointer
type SliverIgnorePointer struct {
	SingleChildRenderObjectWidget
	ignoring bool
}

// NewSliverIgnorePointer creates a widget that is invisible to hit testing while
// it is ignoring.
//
// Flutter: widgets/sliver.dart SliverIgnorePointer.new
func NewSliverIgnorePointer(sliver Widget, opts ...SliverIgnorePointerOpts) *SliverIgnorePointer {
	o := firstOpts(opts)
	return &SliverIgnorePointer{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: o.Key}), o.Ignoring.OrElse(true)}
}

// Ignoring is whether this sliver is ignored during hit testing.
//
// Flutter: widgets/sliver.dart SliverIgnorePointer.ignoring
func (w *SliverIgnorePointer) Ignoring() bool { return w.ignoring }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverIgnorePointer.createRenderObject
func (w *SliverIgnorePointer) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverIgnorePointer(nil, rendering.RenderSliverIgnorePointerOpts{Ignoring: foundation.Some(w.ignoring)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/sliver.dart SliverIgnorePointer.updateRenderObject
func (w *SliverIgnorePointer) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverIgnorePointer).SetIgnoring(w.ignoring)
}

// SliverOffstageOpts are the optional parameters of NewSliverOffstage.
//
// Flutter: widgets/sliver.dart SliverOffstage.new
type SliverOffstageOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Offstage is whether the sliver is hidden; the default is true.
	Offstage foundation.Opt[bool]
}

// SliverOffstage lays out its sliver child as if it was in the tree, but without
// painting anything, without making it available for hit testing, and without
// taking any room in the parent.
//
// Flutter: widgets/sliver.dart SliverOffstage
type SliverOffstage struct {
	SingleChildRenderObjectWidget
	offstage bool
}

// NewSliverOffstage creates a widget that hides its sliver while it is offstage.
//
// Flutter: widgets/sliver.dart SliverOffstage.new
func NewSliverOffstage(sliver Widget, opts ...SliverOffstageOpts) *SliverOffstage {
	o := firstOpts(opts)
	return &SliverOffstage{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: o.Key}), o.Offstage.OrElse(true)}
}

// IsOffstage is whether the sliver is hidden.
//
// Flutter: widgets/sliver.dart SliverOffstage.offstage
func (w *SliverOffstage) IsOffstage() bool { return w.offstage }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverOffstage.createRenderObject
func (w *SliverOffstage) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverOffstage(nil, rendering.RenderSliverOffstageOpts{Offstage: foundation.Some(w.offstage)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/sliver.dart SliverOffstage.updateRenderObject
func (w *SliverOffstage) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverOffstage).SetOffstage(w.offstage)
}

// CreateElement creates the element, which hides the child from the visitors of
// the onstage children while it is offstage.
//
// Flutter: widgets/sliver.dart SliverOffstage.createElement
func (w *SliverOffstage) CreateElement() Element {
	e := &sliverOffstageElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// sliverOffstageElement is the element of a SliverOffstage.
//
// Flutter: widgets/sliver.dart _SliverOffstageElement
type sliverOffstageElement struct {
	SingleChildRenderObjectElement
}

// DebugVisitOnstageChildren visits the child unless the widget is offstage.
func (e *sliverOffstageElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	if !e.widget.(*SliverOffstage).offstage {
		e.VisitChildren(visitor)
	}
}

// SliverPaddingOpts are the optional parameters of NewSliverPadding.
//
// Flutter: widgets/basic.dart SliverPadding.new
type SliverPaddingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverPadding insets another sliver by the given padding.
//
// Flutter: widgets/basic.dart SliverPadding
type SliverPadding struct {
	SingleChildRenderObjectWidget
	padding painting.EdgeInsetsGeometry
}

// NewSliverPadding creates a widget that insets its sliver.
//
// Flutter: widgets/basic.dart SliverPadding.new
func NewSliverPadding(padding painting.EdgeInsetsGeometry, sliver Widget, opts ...SliverPaddingOpts) *SliverPadding {
	return &SliverPadding{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: firstOpts(opts).Key}), padding}
}

// Padding is the amount of space by which to inset the child sliver.
//
// Flutter: widgets/basic.dart SliverPadding.padding
func (w *SliverPadding) Padding() painting.EdgeInsetsGeometry { return w.padding }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart SliverPadding.createRenderObject
func (w *SliverPadding) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverPadding(w.padding, nil, rendering.RenderSliverPaddingOpts{TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart SliverPadding.updateRenderObject
func (w *SliverPadding) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSliverPadding)
	r.SetPadding(w.padding)
	r.SetTextDirection(maybeTextDirection(bc))
}

// SliverToBoxAdapterOpts are the optional parameters of NewSliverToBoxAdapter.
//
// Flutter: widgets/basic.dart SliverToBoxAdapter.new
type SliverToBoxAdapterOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverToBoxAdapter is a sliver that contains a single box widget.
//
// Flutter: widgets/basic.dart SliverToBoxAdapter
type SliverToBoxAdapter struct {
	SingleChildRenderObjectWidget
}

// NewSliverToBoxAdapter creates a sliver that contains the box widget; the child
// may be nil.
//
// Flutter: widgets/basic.dart SliverToBoxAdapter.new
func NewSliverToBoxAdapter(child Widget, opts ...SliverToBoxAdapterOpts) *SliverToBoxAdapter {
	return &SliverToBoxAdapter{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: firstOpts(opts).Key})}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart SliverToBoxAdapter.createRenderObject
func (w *SliverToBoxAdapter) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverToBoxAdapter(nil)
}

// SliverConstrainedCrossAxisOpts are the optional parameters of
// NewSliverConstrainedCrossAxis.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis.new
type SliverConstrainedCrossAxisOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverConstrainedCrossAxis constrains the cross axis extent of its sliver
// child. It is used in a SliverCrossAxisGroup.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis
type SliverConstrainedCrossAxis struct {
	StatelessWidget
	maxExtent float64
	sliver    Widget
}

// NewSliverConstrainedCrossAxis creates a widget that limits the cross axis
// extent of the sliver to maxExtent.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis.new
func NewSliverConstrainedCrossAxis(maxExtent float64, sliver Widget, opts ...SliverConstrainedCrossAxisOpts) *SliverConstrainedCrossAxis {
	return &SliverConstrainedCrossAxis{NewStatelessWidgetBase(WidgetOpts{Key: firstOpts(opts).Key}), maxExtent, sliver}
}

// MaxExtent is the maximum extent in the cross axis.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis.maxExtent
func (w *SliverConstrainedCrossAxis) MaxExtent() float64 { return w.maxExtent }

// Sliver is the sliver below.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis.sliver
func (w *SliverConstrainedCrossAxis) Sliver() Widget { return w.sliver }

// Build wraps the constrained sliver so that it has no flex.
//
// Flutter: widgets/sliver.dart SliverConstrainedCrossAxis.build
func (w *SliverConstrainedCrossAxis) Build(BuildContext) Widget {
	return &sliverZeroFlexParentDataWidget{
		ParentDataWidgetOf: NewParentDataWidgetBase[*rendering.SliverPhysicalContainerParentData](&sliverConstrainedCrossAxis{
			SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(w.sliver), maxExtent: w.maxExtent})}
}

// sliverZeroFlexParentDataWidget sets the cross axis flex to zero.
//
// Flutter: widgets/sliver.dart _SliverZeroFlexParentDataWidget
type sliverZeroFlexParentDataWidget struct {
	ParentDataWidgetOf[*rendering.SliverPhysicalContainerParentData]
}

func (w *sliverZeroFlexParentDataWidget) ApplyParentData(renderObject rendering.RenderObject) {
	parentData := rendering.SliverPhysicalParentDataOf(renderObject)
	if parentData.CrossAxisFlex != foundation.Some(0) {
		parentData.CrossAxisFlex = foundation.Some(0)
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

func (w *sliverZeroFlexParentDataWidget) DebugTypicalAncestorWidgetClass() string {
	return "SliverCrossAxisGroup"
}

// sliverConstrainedCrossAxis is the render object widget of
// SliverConstrainedCrossAxis.
//
// Flutter: widgets/sliver.dart _SliverConstrainedCrossAxis
type sliverConstrainedCrossAxis struct {
	SingleChildRenderObjectWidget
	maxExtent float64
}

func (w *sliverConstrainedCrossAxis) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverConstrainedCrossAxis(w.maxExtent, nil)
}

func (w *sliverConstrainedCrossAxis) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverConstrainedCrossAxis).SetMaxExtent(w.maxExtent)
}

// SliverCrossAxisExpandedOpts are the optional parameters of
// NewSliverCrossAxisExpanded.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded.new
type SliverCrossAxisExpandedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverCrossAxisExpanded sets the cross axis flex of a sliver child of a
// SliverCrossAxisGroup.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded
type SliverCrossAxisExpanded struct {
	ParentDataWidgetOf[*rendering.SliverPhysicalContainerParentData]
	flex int
}

// NewSliverCrossAxisExpanded creates a widget that gives the sliver the flex
// factor, which must be greater than zero.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded.new
func NewSliverCrossAxisExpanded(flex int, sliver Widget, opts ...SliverCrossAxisExpandedOpts) *SliverCrossAxisExpanded {
	if foundation.DebugMode && flex <= 0 {
		panic(errors.New("SliverCrossAxisExpandedFlexInvalid", "the flex of a SliverCrossAxisExpanded must be greater than zero", "flex", flex))
	}
	return &SliverCrossAxisExpanded{NewParentDataWidgetBase[*rendering.SliverPhysicalContainerParentData](sliver, WidgetOpts{Key: firstOpts(opts).Key}), flex}
}

// Flex is the flex factor of the sliver.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded.flex
func (w *SliverCrossAxisExpanded) Flex() int { return w.flex }

// ApplyParentData writes the flex factor to the parent data.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded.applyParentData
func (w *SliverCrossAxisExpanded) ApplyParentData(renderObject rendering.RenderObject) {
	parentData := rendering.SliverPhysicalParentDataOf(renderObject)
	if parentData.CrossAxisFlex != foundation.Some(w.flex) {
		parentData.CrossAxisFlex = foundation.Some(w.flex)
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

// DebugTypicalAncestorWidgetClass names SliverCrossAxisGroup.
//
// Flutter: widgets/sliver.dart SliverCrossAxisExpanded.debugTypicalAncestorWidgetClass
func (w *SliverCrossAxisExpanded) DebugTypicalAncestorWidgetClass() string {
	return "SliverCrossAxisGroup"
}

// SliverCrossAxisGroupOpts are the optional parameters of NewSliverCrossAxisGroup.
//
// Flutter: widgets/sliver.dart SliverCrossAxisGroup.new
type SliverCrossAxisGroupOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverCrossAxisGroup places its sliver children next to each other in the
// cross axis.
//
// Flutter: widgets/sliver.dart SliverCrossAxisGroup
type SliverCrossAxisGroup struct {
	MultiChildRenderObjectWidget
}

// NewSliverCrossAxisGroup creates a widget that lays the slivers out in the
// cross axis.
//
// Flutter: widgets/sliver.dart SliverCrossAxisGroup.new
func NewSliverCrossAxisGroup(slivers []Widget, opts ...SliverCrossAxisGroupOpts) *SliverCrossAxisGroup {
	return &SliverCrossAxisGroup{NewMultiChildRenderObjectWidgetBase(slivers, WidgetOpts{Key: firstOpts(opts).Key})}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverCrossAxisGroup.createRenderObject
func (w *SliverCrossAxisGroup) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverCrossAxisGroup()
}

// SliverMainAxisGroupOpts are the optional parameters of NewSliverMainAxisGroup.
//
// Flutter: widgets/sliver.dart SliverMainAxisGroup.new
type SliverMainAxisGroupOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverMainAxisGroup places its sliver children one after the other in the main
// axis, as if they were one sliver.
//
// Flutter: widgets/sliver.dart SliverMainAxisGroup
type SliverMainAxisGroup struct {
	MultiChildRenderObjectWidget
}

// NewSliverMainAxisGroup creates a widget that lays the slivers out in the main
// axis.
//
// Flutter: widgets/sliver.dart SliverMainAxisGroup.new
func NewSliverMainAxisGroup(slivers []Widget, opts ...SliverMainAxisGroupOpts) *SliverMainAxisGroup {
	return &SliverMainAxisGroup{NewMultiChildRenderObjectWidgetBase(slivers, WidgetOpts{Key: firstOpts(opts).Key})}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverMainAxisGroup.createRenderObject
func (w *SliverMainAxisGroup) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverMainAxisGroup()
}

// CreateElement creates the element, which visits only the visible slivers when
// asked for the onstage children.
//
// Flutter: widgets/sliver.dart SliverMainAxisGroup.createElement
func (w *SliverMainAxisGroup) CreateElement() Element {
	e := &sliverMainAxisGroupElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	DebugChildrenHaveDuplicateKeys(w, w.Children())
	e.forgotten = map[Element]struct{}{}
	return e
}

// sliverMainAxisGroupElement is the element of a SliverMainAxisGroup.
//
// Flutter: widgets/sliver.dart _SliverMainAxisGroupElement
type sliverMainAxisGroupElement struct {
	MultiChildRenderObjectElement
}

// DebugVisitOnstageChildren visits the slivers that are visible.
func (e *sliverMainAxisGroupElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	for _, child := range e.Children() {
		if child.RenderObject().(rendering.Sliver).Geometry().Visible {
			visitor(child)
		}
	}
}

// SliverFillViewportOpts are the optional parameters of NewSliverFillViewport.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.new
type SliverFillViewportOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ViewportFraction is the fraction of the viewport that each child occupies;
	// the default is 1.
	ViewportFraction foundation.Opt[float64]
	// PadEnds is whether to add padding to both ends of the list so that the
	// first and last children can be centered; the default is true.
	PadEnds foundation.Opt[bool]
	// AllowImplicitScrolling is whether the children of the sliver keep their
	// semantics when they are out of view; the default is true.
	AllowImplicitScrolling foundation.Opt[bool]
}

// SliverFillViewport is a sliver that contains multiple box children that each
// fill the viewport.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport
type SliverFillViewport struct {
	StatelessWidget
	delegate               SliverChildDelegate
	viewportFraction       float64
	padEnds                bool
	allowImplicitScrolling bool
}

// NewSliverFillViewport creates a sliver whose children, that the delegate
// supplies, each fill the viewport. In debug builds it panics with the tag
// ViewportFractionInvalid when the fraction is not positive.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.new
func NewSliverFillViewport(delegate SliverChildDelegate, opts ...SliverFillViewportOpts) *SliverFillViewport {
	o := firstOpts(opts)
	fraction := o.ViewportFraction.OrElse(1)
	if foundation.DebugMode && !(fraction > 0) {
		panic(errors.New("ViewportFractionInvalid", "the viewport fraction must be greater than zero", "viewportFraction", fraction))
	}
	return &SliverFillViewport{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), delegate, fraction, o.PadEnds.OrElse(true), o.AllowImplicitScrolling.OrElse(true)}
}

// Delegate supplies the children.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.delegate
func (w *SliverFillViewport) Delegate() SliverChildDelegate { return w.delegate }

// ViewportFraction is the fraction of the viewport that each child occupies.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.viewportFraction
func (w *SliverFillViewport) ViewportFraction() float64 { return w.viewportFraction }

// PadEnds is whether the ends of the list are padded.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.padEnds
func (w *SliverFillViewport) PadEnds() bool { return w.padEnds }

// Build pads the ends if asked, and builds the sliver.
//
// Flutter: widgets/sliver_fill.dart SliverFillViewport.build
func (w *SliverFillViewport) Build(BuildContext) Widget {
	fraction := 0.0
	if w.padEnds {
		fraction = ui.ClampDouble(1-w.viewportFraction, 0, 1) / 2
	}
	return &sliverFractionalPadding{
		SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(&sliverFillViewportRenderObjectWidget{
			SliverMultiBoxAdaptorWidget: NewSliverMultiBoxAdaptorWidgetBase(w.delegate), viewportFraction: w.viewportFraction,
			allowImplicitScrolling: w.allowImplicitScrolling}),
		viewportFraction: fraction}
}

// sliverFillViewportRenderObjectWidget is the sliver of a SliverFillViewport.
//
// Flutter: widgets/sliver_fill.dart _SliverFillViewportRenderObjectWidget
type sliverFillViewportRenderObjectWidget struct {
	SliverMultiBoxAdaptorWidget
	viewportFraction       float64
	allowImplicitScrolling bool
}

func (w *sliverFillViewportRenderObjectWidget) CreateElement() Element {
	return NewSliverMultiBoxAdaptorElement(w, false)
}

func (w *sliverFillViewportRenderObjectWidget) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverFillViewport(bc.(*SliverMultiBoxAdaptorElement), rendering.RenderSliverFillViewportOpts{
		ViewportFraction: foundation.Some(w.viewportFraction), AllowImplicitScrolling: foundation.Some(w.allowImplicitScrolling)})
}

func (w *sliverFillViewportRenderObjectWidget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSliverFillViewport)
	r.SetViewportFraction(w.viewportFraction)
	r.SetAllowImplicitScrolling(w.allowImplicitScrolling)
}

// sliverFractionalPadding pads a sliver by a fraction of the viewport.
//
// Flutter: widgets/sliver_fill.dart _SliverFractionalPadding
type sliverFractionalPadding struct {
	SingleChildRenderObjectWidget
	viewportFraction float64
}

func (w *sliverFractionalPadding) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderSliverFractionalPadding{viewportFraction: w.viewportFraction}
	rendering.InitRenderSliverEdgeInsetsPadding(&r.RenderSliverEdgeInsetsPadding, r)
	return r
}

func (w *sliverFractionalPadding) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderSliverFractionalPadding).setViewportFraction(w.viewportFraction)
}

// renderSliverFractionalPadding is the render object of a
// sliverFractionalPadding.
//
// Flutter: widgets/sliver_fill.dart _RenderSliverFractionalPadding
type renderSliverFractionalPadding struct {
	rendering.RenderSliverEdgeInsetsPadding
	lastResolvedConstraints rendering.SliverConstraints
	hasResolved             bool
	viewportFraction        float64
	resolvedPadding         foundation.Opt[painting.EdgeInsets]
}

func (r *renderSliverFractionalPadding) ResolvedPadding() foundation.Opt[painting.EdgeInsets] {
	return r.resolvedPadding
}

func (r *renderSliverFractionalPadding) setViewportFraction(value float64) {
	if r.viewportFraction == value {
		return
	}
	r.viewportFraction = value
	r.resolvedPadding = foundation.None[painting.EdgeInsets]()
	r.MarkNeedsLayout()
}

func (r *renderSliverFractionalPadding) resolve() {
	constraints := r.SliverConstraints()
	if r.resolvedPadding.IsSome() && r.hasResolved && r.lastResolvedConstraints == constraints {
		return
	}
	paddingValue := constraints.ViewportMainAxisExtent * r.viewportFraction
	r.lastResolvedConstraints, r.hasResolved = constraints, true
	if constraints.Axis() == painting.AxisHorizontal {
		r.resolvedPadding = foundation.Some(painting.NewEdgeInsetsSymmetric(0, paddingValue))
	} else {
		r.resolvedPadding = foundation.Some(painting.NewEdgeInsetsSymmetric(paddingValue, 0))
	}
}

// PerformLayout resolves the padding, then lays out.
func (r *renderSliverFractionalPadding) PerformLayout() {
	r.resolve()
	r.RenderSliverEdgeInsetsPadding.PerformLayout()
}

// SliverFillRemainingOpts are the optional parameters of NewSliverFillRemaining.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.new
type SliverFillRemainingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// HasScrollBody is whether the child has a scrollable body; the default is
	// true.
	HasScrollBody foundation.Opt[bool]
	// FillOverscroll is whether the child is stretched to fill the overscroll
	// area; it is only used when HasScrollBody is false.
	FillOverscroll bool
}

// SliverFillRemaining is a sliver that contains a single box child that fills the
// remaining space in the viewport.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining
type SliverFillRemaining struct {
	StatelessWidget
	child          Widget
	hasScrollBody  bool
	fillOverscroll bool
}

// NewSliverFillRemaining creates a sliver that fills the remaining space with
// the child, which may be nil.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.new
func NewSliverFillRemaining(child Widget, opts ...SliverFillRemainingOpts) *SliverFillRemaining {
	o := firstOpts(opts)
	return &SliverFillRemaining{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, o.HasScrollBody.OrElse(true), o.FillOverscroll}
}

// Child is the widget below this one.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.child
func (w *SliverFillRemaining) Child() Widget { return w.child }

// HasScrollBody is whether the child has a scrollable body.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.hasScrollBody
func (w *SliverFillRemaining) HasScrollBody() bool { return w.hasScrollBody }

// FillOverscroll is whether the child fills the overscroll area.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.fillOverscroll
func (w *SliverFillRemaining) FillOverscroll() bool { return w.fillOverscroll }

// Build builds the sliver that matches the mode.
//
// Flutter: widgets/sliver_fill.dart SliverFillRemaining.build
func (w *SliverFillRemaining) Build(BuildContext) Widget {
	switch {
	case w.hasScrollBody:
		return &sliverFillRemainingWithScrollable{NewSingleChildRenderObjectWidgetBase(w.child)}
	case !w.fillOverscroll:
		return &sliverFillRemainingWithoutScrollable{NewSingleChildRenderObjectWidgetBase(w.child)}
	}
	return &sliverFillRemainingAndOverscroll{NewSingleChildRenderObjectWidgetBase(w.child)}
}

type sliverFillRemainingWithScrollable struct{ SingleChildRenderObjectWidget }

func (w *sliverFillRemainingWithScrollable) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverFillRemainingWithScrollable(nil)
}

type sliverFillRemainingWithoutScrollable struct{ SingleChildRenderObjectWidget }

func (w *sliverFillRemainingWithoutScrollable) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverFillRemaining(nil)
}

type sliverFillRemainingAndOverscroll struct{ SingleChildRenderObjectWidget }

func (w *sliverFillRemainingAndOverscroll) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverFillRemainingAndOverscroll(nil)
}

// SliverPersistentHeaderDelegate builds the child of a SliverPersistentHeader and
// says how big it can be.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate
type SliverPersistentHeaderDelegate interface {
	// Build builds the header for the offset the sliver has been shrunk by and
	// whether it overlaps the content that follows.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.build
	Build(bc BuildContext, shrinkOffset float64, overlapsContent bool) Widget
	// MinExtent is the smallest size the header can reach, which a pinned header
	// keeps.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.minExtent
	MinExtent() float64
	// MaxExtent is the size of the header when it is not shrunk.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.maxExtent
	MaxExtent() float64
	// Vsync is the ticker provider of the snap animation of a floating header, or
	// nil.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.vsync
	Vsync() scheduler.TickerProvider
	// SnapConfiguration is how a floating header snaps, or nil.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.snapConfiguration
	SnapConfiguration() *rendering.FloatingHeaderSnapConfiguration
	// StretchConfiguration is how the header stretches in an overscroll, or nil.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.stretchConfiguration
	StretchConfiguration() *rendering.OverScrollHeaderStretchConfiguration
	// ShowOnScreenConfiguration is how the header shows on screen, or nil.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.showOnScreenConfiguration
	ShowOnScreenConfiguration() *rendering.PersistentHeaderShowOnScreenConfiguration
	// ShouldRebuild is whether the header must be built again, as this delegate
	// replaces the old one.
	//
	// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.shouldRebuild
	ShouldRebuild(oldDelegate SliverPersistentHeaderDelegate) bool
}

// SliverPersistentHeaderDelegateBase gives the optional methods of a
// SliverPersistentHeaderDelegate their default (nil); embed it in a delegate.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate
type SliverPersistentHeaderDelegateBase struct{}

// Vsync is nil.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.vsync
func (SliverPersistentHeaderDelegateBase) Vsync() scheduler.TickerProvider { return nil }

// SnapConfiguration is nil.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.snapConfiguration
func (SliverPersistentHeaderDelegateBase) SnapConfiguration() *rendering.FloatingHeaderSnapConfiguration {
	return nil
}

// StretchConfiguration is nil.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.stretchConfiguration
func (SliverPersistentHeaderDelegateBase) StretchConfiguration() *rendering.OverScrollHeaderStretchConfiguration {
	return nil
}

// ShowOnScreenConfiguration is nil.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeaderDelegate.showOnScreenConfiguration
func (SliverPersistentHeaderDelegateBase) ShowOnScreenConfiguration() *rendering.PersistentHeaderShowOnScreenConfiguration {
	return nil
}

// SliverPersistentHeaderOpts are the optional parameters of
// NewSliverPersistentHeader.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.new
type SliverPersistentHeaderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Pinned is whether the header stays at the start of the viewport when it
	// would scroll out of view.
	Pinned bool
	// Floating is whether the header appears as soon as the user scrolls back
	// towards it.
	Floating bool
}

// SliverPersistentHeader is a sliver whose size varies when the sliver is
// scrolled to the edge of the viewport opposite the sliver's
// GrowthDirection.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader
type SliverPersistentHeader struct {
	StatelessWidget
	delegate SliverPersistentHeaderDelegate
	pinned   bool
	floating bool
}

// NewSliverPersistentHeader creates a header that the delegate builds.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.new
func NewSliverPersistentHeader(delegate SliverPersistentHeaderDelegate, opts ...SliverPersistentHeaderOpts) *SliverPersistentHeader {
	o := firstOpts(opts)
	return &SliverPersistentHeader{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), delegate, o.Pinned, o.Floating}
}

// Delegate builds the header.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.delegate
func (w *SliverPersistentHeader) Delegate() SliverPersistentHeaderDelegate { return w.delegate }

// Pinned is whether the header is pinned.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.pinned
func (w *SliverPersistentHeader) Pinned() bool { return w.pinned }

// Floating is whether the header floats.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.floating
func (w *SliverPersistentHeader) Floating() bool { return w.floating }

// Build builds the header of the kind that the flags say.
//
// Flutter: widgets/sliver_persistent_header.dart SliverPersistentHeader.build
func (w *SliverPersistentHeader) Build(BuildContext) Widget {
	kind := persistentHeaderScrolling
	switch {
	case w.floating && w.pinned:
		kind = persistentHeaderFloatingPinned
	case w.pinned:
		kind = persistentHeaderPinned
	case w.floating:
		kind = persistentHeaderFloating
	}
	return &sliverPersistentHeaderRenderObjectWidget{RenderObjectWidget: NewRenderObjectWidgetBase(), delegate: w.delegate, kind: kind}
}

type persistentHeaderKind int

const (
	persistentHeaderScrolling persistentHeaderKind = iota
	persistentHeaderPinned
	persistentHeaderFloating
	persistentHeaderFloatingPinned
)

func (k persistentHeaderKind) floating() bool {
	return k == persistentHeaderFloating || k == persistentHeaderFloatingPinned
}

// floatingHeader listens to the scrolling of the scrollable, for the header
// renderer above it.
//
// Flutter: widgets/sliver_persistent_header.dart _FloatingHeader
type floatingHeader struct {
	StatefulWidget
	child Widget
}

func (w *floatingHeader) CreateState() State { return &floatingHeaderState{} }

type floatingHeaderState struct {
	StateOf[*floatingHeader]
	position ScrollPosition
	handle   foundation.ListenerHandle
}

func (s *floatingHeaderState) DidChangeDependencies() {
	if s.position != nil {
		s.position.IsScrollingNotifier().RemoveListener(s.handle)
	}
	s.position = nil
	if scrollable := ScrollableMaybeOf(s.Context()); scrollable != nil {
		s.position = scrollable.Position()
	}
	if s.position != nil {
		s.handle = s.position.IsScrollingNotifier().AddListener(s.isScrollingListener)
	}
}

func (s *floatingHeaderState) Dispose() {
	if s.position != nil {
		s.position.IsScrollingNotifier().RemoveListener(s.handle)
	}
}

func (s *floatingHeaderState) isScrollingListener() {
	header, _ := FindAncestorRenderObjectOfType[*renderSliverFloatingPersistentHeaderForWidgets](s.Context())
	var floating interface {
		UpdateScrollStartDirection(rendering.ScrollDirection)
		MaybeStopSnapAnimation(rendering.ScrollDirection)
		MaybeStartSnapAnimation(rendering.ScrollDirection)
	} = nil
	if header != nil {
		floating = header
	} else if pinned, ok := FindAncestorRenderObjectOfType[*renderSliverFloatingPinnedPersistentHeaderForWidgets](s.Context()); ok && pinned != nil {
		floating = pinned
	}
	direction := rendering.ScrollDirectionIdle
	if d, ok := s.position.(interface {
		UserScrollDirection() rendering.ScrollDirection
	}); ok {
		direction = d.UserScrollDirection()
	}
	if s.position.IsScrollingNotifier().Value() {
		if floating != nil {
			floating.UpdateScrollStartDirection(direction)
			floating.MaybeStopSnapAnimation(direction)
		}
	} else if floating != nil {
		floating.MaybeStartSnapAnimation(direction)
	}
}

func (s *floatingHeaderState) Build(BuildContext) Widget { return s.Widget().child }

// sliverPersistentHeaderRenderObjectWidget is the widget of the render object of
// a SliverPersistentHeader.
//
// Flutter: widgets/sliver_persistent_header.dart _SliverPersistentHeaderRenderObjectWidget
type sliverPersistentHeaderRenderObjectWidget struct {
	RenderObjectWidget
	delegate SliverPersistentHeaderDelegate
	kind     persistentHeaderKind
}

func (w *sliverPersistentHeaderRenderObjectWidget) CreateElement() Element {
	e := &sliverPersistentHeaderElement{floating: w.kind.floating()}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

func (w *sliverPersistentHeaderRenderObjectWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	d := w.delegate
	switch w.kind {
	case persistentHeaderPinned:
		r := &renderSliverPinnedPersistentHeaderForWidgets{}
		rendering.InitRenderSliverPinnedPersistentHeader(&r.RenderSliverPinnedPersistentHeader, r, nil, d.StretchConfiguration(), d.ShowOnScreenConfiguration())
		return r
	case persistentHeaderFloating:
		r := &renderSliverFloatingPersistentHeaderForWidgets{}
		rendering.InitRenderSliverFloatingPersistentHeader(&r.RenderSliverFloatingPersistentHeader, r, nil, d.Vsync(), d.SnapConfiguration(),
			d.StretchConfiguration(), d.ShowOnScreenConfiguration())
		return r
	case persistentHeaderFloatingPinned:
		r := &renderSliverFloatingPinnedPersistentHeaderForWidgets{}
		rendering.InitRenderSliverFloatingPinnedPersistentHeader(&r.RenderSliverFloatingPinnedPersistentHeader, r, nil, d.Vsync(), d.SnapConfiguration(),
			d.StretchConfiguration(), d.ShowOnScreenConfiguration())
		return r
	}
	r := &renderSliverScrollingPersistentHeaderForWidgets{}
	rendering.InitRenderSliverScrollingPersistentHeader(&r.RenderSliverScrollingPersistentHeader, r, nil, d.StretchConfiguration())
	return r
}

func (w *sliverPersistentHeaderRenderObjectWidget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	d := w.delegate
	switch r := renderObject.(type) {
	case *renderSliverScrollingPersistentHeaderForWidgets:
		r.SetStretchConfiguration(d.StretchConfiguration())
	case *renderSliverPinnedPersistentHeaderForWidgets:
		r.SetStretchConfiguration(d.StretchConfiguration())
		r.SetShowOnScreenConfiguration(d.ShowOnScreenConfiguration())
	case *renderSliverFloatingPersistentHeaderForWidgets:
		r.SetVsync(d.Vsync())
		r.SetSnapConfiguration(d.SnapConfiguration())
		r.SetStretchConfiguration(d.StretchConfiguration())
		r.SetShowOnScreenConfiguration(d.ShowOnScreenConfiguration())
	case *renderSliverFloatingPinnedPersistentHeaderForWidgets:
		r.SetVsync(d.Vsync())
		r.SetSnapConfiguration(d.SnapConfiguration())
		r.SetStretchConfiguration(d.StretchConfiguration())
		r.SetShowOnScreenConfiguration(d.ShowOnScreenConfiguration())
	}
}

// persistentHeaderForWidgets is what the render objects of the headers share: they
// ask the element to build the child.
//
// Flutter: widgets/sliver_persistent_header.dart _RenderSliverPersistentHeaderForWidgetsMixin
type persistentHeaderForWidgets interface {
	rendering.RenderObject
	setElement(e *sliverPersistentHeaderElement)
	lastShrinkOffset() float64
	lastOverlapsContent() bool
	triggerRebuild()
}

type headerWidgetsMixin struct {
	element *sliverPersistentHeaderElement
}

func (m *headerWidgetsMixin) setElement(e *sliverPersistentHeaderElement) { m.element = e }

// MinExtent is the minimum extent that the delegate says.
func (m *headerWidgetsMixin) MinExtent() float64 {
	return m.element.widget.(*sliverPersistentHeaderRenderObjectWidget).delegate.MinExtent()
}

// MaxExtent is the maximum extent that the delegate says.
func (m *headerWidgetsMixin) MaxExtent() float64 {
	return m.element.widget.(*sliverPersistentHeaderRenderObjectWidget).delegate.MaxExtent()
}

// UpdateChild builds the child in the element.
func (m *headerWidgetsMixin) UpdateChild(shrinkOffset float64, overlapsContent bool) {
	m.element.build(shrinkOffset, overlapsContent)
}

type renderSliverScrollingPersistentHeaderForWidgets struct {
	rendering.RenderSliverScrollingPersistentHeader
	headerWidgetsMixin
}

func (r *renderSliverScrollingPersistentHeaderForWidgets) lastShrinkOffset() float64 {
	return r.LastShrinkOffset()
}
func (r *renderSliverScrollingPersistentHeaderForWidgets) lastOverlapsContent() bool {
	return r.LastOverlapsContent()
}
func (r *renderSliverScrollingPersistentHeaderForWidgets) triggerRebuild() { r.MarkNeedsLayout() }

type renderSliverPinnedPersistentHeaderForWidgets struct {
	rendering.RenderSliverPinnedPersistentHeader
	headerWidgetsMixin
}

func (r *renderSliverPinnedPersistentHeaderForWidgets) lastShrinkOffset() float64 {
	return r.LastShrinkOffset()
}
func (r *renderSliverPinnedPersistentHeaderForWidgets) lastOverlapsContent() bool {
	return r.LastOverlapsContent()
}
func (r *renderSliverPinnedPersistentHeaderForWidgets) triggerRebuild() { r.MarkNeedsLayout() }

type renderSliverFloatingPersistentHeaderForWidgets struct {
	rendering.RenderSliverFloatingPersistentHeader
	headerWidgetsMixin
}

func (r *renderSliverFloatingPersistentHeaderForWidgets) lastShrinkOffset() float64 {
	return r.LastShrinkOffset()
}
func (r *renderSliverFloatingPersistentHeaderForWidgets) lastOverlapsContent() bool {
	return r.LastOverlapsContent()
}
func (r *renderSliverFloatingPersistentHeaderForWidgets) triggerRebuild() { r.MarkNeedsLayout() }

type renderSliverFloatingPinnedPersistentHeaderForWidgets struct {
	rendering.RenderSliverFloatingPinnedPersistentHeader
	headerWidgetsMixin
}

func (r *renderSliverFloatingPinnedPersistentHeaderForWidgets) lastShrinkOffset() float64 {
	return r.LastShrinkOffset()
}
func (r *renderSliverFloatingPinnedPersistentHeaderForWidgets) lastOverlapsContent() bool {
	return r.LastOverlapsContent()
}
func (r *renderSliverFloatingPinnedPersistentHeaderForWidgets) triggerRebuild() { r.MarkNeedsLayout() }

// sliverPersistentHeaderElement is the element of a persistent header: it builds
// the child during the layout of the render object.
//
// Flutter: widgets/sliver_persistent_header.dart _SliverPersistentHeaderElement
type sliverPersistentHeaderElement struct {
	RenderObjectElement
	floating bool
	child    Element
}

func (e *sliverPersistentHeaderElement) header() persistentHeaderForWidgets {
	return e.renderObject.(persistentHeaderForWidgets)
}

// Mount tells the render object about this element.
func (e *sliverPersistentHeaderElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	e.header().setElement(e)
}

// Unmount takes this element away from the render object.
func (e *sliverPersistentHeaderElement) Unmount() {
	e.header().setElement(nil)
	e.RenderObjectElement.Unmount()
}

// Update builds the child again when the delegate says it must.
func (e *sliverPersistentHeaderElement) Update(newWidget Widget) {
	oldDelegate := e.widget.(*sliverPersistentHeaderRenderObjectWidget).delegate
	e.RenderObjectElement.Update(newWidget)
	newDelegate := newWidget.(*sliverPersistentHeaderRenderObjectWidget).delegate
	if newDelegate != oldDelegate && (foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(oldDelegate) || newDelegate.ShouldRebuild(oldDelegate)) {
		header := e.header()
		e.updateChild(newDelegate, header.lastShrinkOffset(), header.lastOverlapsContent())
		header.triggerRebuild()
	}
}

// PerformRebuild asks the render object to lay out again.
func (e *sliverPersistentHeaderElement) PerformRebuild() {
	e.RenderObjectElement.PerformRebuild()
	e.header().triggerRebuild()
}

func (e *sliverPersistentHeaderElement) updateChild(delegate SliverPersistentHeaderDelegate, shrinkOffset float64, overlapsContent bool) {
	newWidget := delegate.Build(e, shrinkOffset, overlapsContent)
	if e.floating {
		newWidget = &floatingHeader{NewStatefulWidgetBase(), newWidget}
	}
	e.child = e.UpdateChild(e.child, newWidget, nil)
}

func (e *sliverPersistentHeaderElement) build(shrinkOffset float64, overlapsContent bool) {
	e.owner.BuildScope(e, func() {
		e.updateChild(e.widget.(*sliverPersistentHeaderRenderObjectWidget).delegate, shrinkOffset, overlapsContent)
	})
}

// ForgetChild drops the child.
func (e *sliverPersistentHeaderElement) ForgetChild(child Element) {
	e.child = nil
	e.RenderObjectElement.ForgetChild(child)
}

// InsertRenderObjectChild sets the child of the render object.
func (e *sliverPersistentHeaderElement) InsertRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(singleChildHolder).SetChildObject(child)
}

// MoveRenderObjectChild does nothing: the single child has no other slot.
func (e *sliverPersistentHeaderElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {}

// RemoveRenderObjectChild empties the render object.
func (e *sliverPersistentHeaderElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	e.renderObject.(singleChildHolder).SetChildObject(nil)
}

// VisitChildren visits the child.
func (e *sliverPersistentHeaderElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
}
