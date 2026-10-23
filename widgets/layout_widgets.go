package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// PaddingOpts are the optional parameters of NewPadding.
//
// Flutter: widgets/basic.dart Padding.new
type PaddingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// Padding insets its child by the given padding.
//
// Flutter: widgets/basic.dart Padding
type Padding struct {
	SingleChildRenderObjectWidget
	padding painting.EdgeInsetsGeometry
}

// NewPadding creates the widget.
//
// Flutter: widgets/basic.dart Padding.new
func NewPadding(padding painting.EdgeInsetsGeometry, child Widget, opts ...PaddingOpts) *Padding {
	o := firstOpts(opts)
	w := &Padding{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), padding: padding}
	return w
}

// Padding is the amount of space by which to inset the child.
//
// Flutter: widgets/basic.dart Padding.padding
func (w *Padding) Padding() painting.EdgeInsetsGeometry { return w.padding }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Padding.createRenderObject
func (w *Padding) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderPadding(w.padding, nil, rendering.RenderPaddingOpts{TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Padding.updateRenderObject
func (w *Padding) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPadding)
	r.SetPadding(w.padding)
	r.SetTextDirection(maybeTextDirection(bc))
}

// AlignOpts are the optional parameters of NewAlign.
//
// Flutter: widgets/basic.dart Align.new
type AlignOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// WidthFactor sets its width to the child's width times this factor when it is given.
	WidthFactor foundation.Opt[float64]
	// HeightFactor sets its height to the child's height times this factor when it is given.
	HeightFactor foundation.Opt[float64]
}

// Align aligns its child within itself and optionally sizes itself based on the child's size.
//
// Flutter: widgets/basic.dart Align
type Align struct {
	SingleChildRenderObjectWidget
	alignment    painting.AlignmentGeometry
	widthFactor  foundation.Opt[float64]
	heightFactor foundation.Opt[float64]
}

// NewAlign creates the widget.
//
// Flutter: widgets/basic.dart Align.new
func NewAlign(child Widget, opts ...AlignOpts) *Align {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	w := &Align{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), alignment: o.Alignment, widthFactor: o.WidthFactor, heightFactor: o.HeightFactor}
	if foundation.DebugMode {
		assertFactor("Align", "WidthFactor", o.WidthFactor)
		assertFactor("Align", "HeightFactor", o.HeightFactor)
	}
	return w
}

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart Align.alignment
func (w *Align) Alignment() painting.AlignmentGeometry { return w.alignment }

// WidthFactor sets its width to the child's width times this factor when it is given.
//
// Flutter: widgets/basic.dart Align.widthFactor
func (w *Align) WidthFactor() foundation.Opt[float64] { return w.widthFactor }

// HeightFactor sets its height to the child's height times this factor when it is given.
//
// Flutter: widgets/basic.dart Align.heightFactor
func (w *Align) HeightFactor() foundation.Opt[float64] { return w.heightFactor }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Align.createRenderObject
func (w *Align) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderPositionedBox(nil, rendering.RenderPositionedBoxOpts{Alignment: w.alignment, WidthFactor: w.widthFactor,
		HeightFactor: w.heightFactor, TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Align.updateRenderObject
func (w *Align) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPositionedBox)
	r.SetAlignment(w.alignment)
	r.SetWidthFactor(w.widthFactor)
	r.SetHeightFactor(w.heightFactor)
	r.SetTextDirection(maybeTextDirection(bc))
}

// ConstrainedBoxOpts are the optional parameters of NewConstrainedBox.
//
// Flutter: widgets/basic.dart ConstrainedBox.new
type ConstrainedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// ConstrainedBox imposes additional constraints on its child.
//
// Flutter: widgets/basic.dart ConstrainedBox
type ConstrainedBox struct {
	SingleChildRenderObjectWidget
	constraints rendering.BoxConstraints
}

// NewConstrainedBox creates the widget.
//
// Flutter: widgets/basic.dart ConstrainedBox.new
func NewConstrainedBox(constraints rendering.BoxConstraints, child Widget, opts ...ConstrainedBoxOpts) *ConstrainedBox {
	o := firstOpts(opts)
	w := &ConstrainedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), constraints: constraints}
	if foundation.DebugMode {
		constraints.DebugAssertIsValid()
	}
	return w
}

// Constraints are the additional constraints to impose on the child.
//
// Flutter: widgets/basic.dart ConstrainedBox.constraints
func (w *ConstrainedBox) Constraints() rendering.BoxConstraints { return w.constraints }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ConstrainedBox.createRenderObject
func (w *ConstrainedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedBox(w.constraints, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ConstrainedBox.updateRenderObject
func (w *ConstrainedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(w.constraints)
}

// LimitedBoxOpts are the optional parameters of NewLimitedBox.
//
// Flutter: widgets/basic.dart LimitedBox.new
type LimitedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MaxWidth is the maximum width limit to apply in the absence of a maxWidth constraint; the default is infinity.
	MaxWidth foundation.Opt[float64]
	// MaxHeight is the maximum height limit to apply in the absence of a maxHeight constraint; the default is infinity.
	MaxHeight foundation.Opt[float64]
}

// LimitedBox limits its size only when it is unconstrained.
//
// Flutter: widgets/basic.dart LimitedBox
type LimitedBox struct {
	SingleChildRenderObjectWidget
	maxWidth  foundation.Opt[float64]
	maxHeight foundation.Opt[float64]
}

// NewLimitedBox creates the widget.
//
// Flutter: widgets/basic.dart LimitedBox.new
func NewLimitedBox(child Widget, opts ...LimitedBoxOpts) *LimitedBox {
	o := firstOpts(opts)
	if o.MaxWidth.IsNone() {
		o.MaxWidth = foundation.Some(math.Inf(1))
	}
	if o.MaxHeight.IsNone() {
		o.MaxHeight = foundation.Some(math.Inf(1))
	}
	w := &LimitedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), maxWidth: o.MaxWidth, maxHeight: o.MaxHeight}
	if foundation.DebugMode {
		assertNonNegative("LimitedBox", "MaxWidth", o.MaxWidth.OrElse(0))
		assertNonNegative("LimitedBox", "MaxHeight", o.MaxHeight.OrElse(0))
	}
	return w
}

// MaxWidth is the maximum width limit to apply in the absence of a maxWidth constraint; the default is infinity.
//
// Flutter: widgets/basic.dart LimitedBox.maxWidth
func (w *LimitedBox) MaxWidth() foundation.Opt[float64] { return w.maxWidth }

// MaxHeight is the maximum height limit to apply in the absence of a maxHeight constraint; the default is infinity.
//
// Flutter: widgets/basic.dart LimitedBox.maxHeight
func (w *LimitedBox) MaxHeight() foundation.Opt[float64] { return w.maxHeight }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart LimitedBox.createRenderObject
func (w *LimitedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderLimitedBox(nil, rendering.RenderLimitedBoxOpts{MaxWidth: w.maxWidth, MaxHeight: w.maxHeight})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart LimitedBox.updateRenderObject
func (w *LimitedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderLimitedBox)
	r.SetMaxWidth(w.maxWidth.OrElse(math.Inf(1)))
	r.SetMaxHeight(w.maxHeight.OrElse(math.Inf(1)))
}

// OverflowBoxOpts are the optional parameters of NewOverflowBox.
//
// Flutter: widgets/basic.dart OverflowBox.new
type OverflowBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// MinWidth is the minimum width constraint to give the child; null uses the parent's.
	MinWidth foundation.Opt[float64]
	// MaxWidth is the maximum width constraint to give the child; null uses the parent's.
	MaxWidth foundation.Opt[float64]
	// MinHeight is the minimum height constraint to give the child; null uses the parent's.
	MinHeight foundation.Opt[float64]
	// MaxHeight is the maximum height constraint to give the child; null uses the parent's.
	MaxHeight foundation.Opt[float64]
	// Fit is how the box sizes itself: OverflowBoxFitMax (the default) or OverflowBoxFitDeferToChild.
	Fit rendering.OverflowBoxFit
}

// OverflowBox imposes different constraints on its child than it gets from its parent, possibly allowing the child to overflow the parent.
//
// Flutter: widgets/basic.dart OverflowBox
type OverflowBox struct {
	SingleChildRenderObjectWidget
	alignment painting.AlignmentGeometry
	minWidth  foundation.Opt[float64]
	maxWidth  foundation.Opt[float64]
	minHeight foundation.Opt[float64]
	maxHeight foundation.Opt[float64]
	fit       rendering.OverflowBoxFit
}

// NewOverflowBox creates the widget.
//
// Flutter: widgets/basic.dart OverflowBox.new
func NewOverflowBox(child Widget, opts ...OverflowBoxOpts) *OverflowBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.Fit == rendering.OverflowBoxFitUnspecified {
		o.Fit = rendering.OverflowBoxFitMax
	}
	w := &OverflowBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), alignment: o.Alignment, minWidth: o.MinWidth, maxWidth: o.MaxWidth, minHeight: o.MinHeight, maxHeight: o.MaxHeight, fit: o.Fit}
	return w
}

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart OverflowBox.alignment
func (w *OverflowBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// MinWidth is the minimum width constraint to give the child; null uses the parent's.
//
// Flutter: widgets/basic.dart OverflowBox.minWidth
func (w *OverflowBox) MinWidth() foundation.Opt[float64] { return w.minWidth }

// MaxWidth is the maximum width constraint to give the child; null uses the parent's.
//
// Flutter: widgets/basic.dart OverflowBox.maxWidth
func (w *OverflowBox) MaxWidth() foundation.Opt[float64] { return w.maxWidth }

// MinHeight is the minimum height constraint to give the child; null uses the parent's.
//
// Flutter: widgets/basic.dart OverflowBox.minHeight
func (w *OverflowBox) MinHeight() foundation.Opt[float64] { return w.minHeight }

// MaxHeight is the maximum height constraint to give the child; null uses the parent's.
//
// Flutter: widgets/basic.dart OverflowBox.maxHeight
func (w *OverflowBox) MaxHeight() foundation.Opt[float64] { return w.maxHeight }

// Fit is how the box sizes itself: OverflowBoxFitMax (the default) or OverflowBoxFitDeferToChild.
//
// Flutter: widgets/basic.dart OverflowBox.fit
func (w *OverflowBox) Fit() rendering.OverflowBoxFit { return w.fit }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart OverflowBox.createRenderObject
func (w *OverflowBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedOverflowBox(nil, rendering.RenderConstrainedOverflowBoxOpts{Alignment: w.alignment,
		MinWidth: w.minWidth, MaxWidth: w.maxWidth, MinHeight: w.minHeight, MaxHeight: w.maxHeight, Fit: w.fit,
		TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart OverflowBox.updateRenderObject
func (w *OverflowBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderConstrainedOverflowBox)
	r.SetAlignment(w.alignment)
	r.SetMinWidth(w.minWidth)
	r.SetMaxWidth(w.maxWidth)
	r.SetMinHeight(w.minHeight)
	r.SetMaxHeight(w.maxHeight)
	r.SetFit(w.fit)
	r.SetTextDirection(maybeTextDirection(bc))
}

// SizedOverflowBoxOpts are the optional parameters of NewSizedOverflowBox.
//
// Flutter: widgets/basic.dart SizedOverflowBox.new
type SizedOverflowBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
}

// SizedOverflowBox is a widget of a given size that lets its child overflow.
//
// Flutter: widgets/basic.dart SizedOverflowBox
type SizedOverflowBox struct {
	SingleChildRenderObjectWidget
	size      ui.Size
	alignment painting.AlignmentGeometry
}

// NewSizedOverflowBox creates the widget.
//
// Flutter: widgets/basic.dart SizedOverflowBox.new
func NewSizedOverflowBox(size ui.Size, child Widget, opts ...SizedOverflowBoxOpts) *SizedOverflowBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	w := &SizedOverflowBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), size: size, alignment: o.Alignment}
	return w
}

// Size is the size this widget requests; it is the parent that decides, within the constraints.
//
// Flutter: widgets/basic.dart SizedOverflowBox.size
func (w *SizedOverflowBox) Size() ui.Size { return w.size }

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart SizedOverflowBox.alignment
func (w *SizedOverflowBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart SizedOverflowBox.createRenderObject
func (w *SizedOverflowBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSizedOverflowBox(w.size, nil, rendering.RenderSizedOverflowBoxOpts{Alignment: w.alignment, TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart SizedOverflowBox.updateRenderObject
func (w *SizedOverflowBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSizedOverflowBox)
	r.SetAlignment(w.alignment)
	r.SetRequestedSize(w.size)
	r.SetTextDirection(maybeTextDirection(bc))
}

// FractionallySizedBoxOpts are the optional parameters of NewFractionallySizedBox.
//
// Flutter: widgets/basic.dart FractionallySizedBox.new
type FractionallySizedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// WidthFactor is the fraction of the available width to give the child; null passes the parent's width.
	WidthFactor foundation.Opt[float64]
	// HeightFactor is the fraction of the available height to give the child; null passes the parent's height.
	HeightFactor foundation.Opt[float64]
}

// FractionallySizedBox sizes its child to a fraction of the total available space.
//
// Flutter: widgets/basic.dart FractionallySizedBox
type FractionallySizedBox struct {
	SingleChildRenderObjectWidget
	alignment    painting.AlignmentGeometry
	widthFactor  foundation.Opt[float64]
	heightFactor foundation.Opt[float64]
}

// NewFractionallySizedBox creates the widget.
//
// Flutter: widgets/basic.dart FractionallySizedBox.new
func NewFractionallySizedBox(child Widget, opts ...FractionallySizedBoxOpts) *FractionallySizedBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	w := &FractionallySizedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), alignment: o.Alignment, widthFactor: o.WidthFactor, heightFactor: o.HeightFactor}
	if foundation.DebugMode {
		assertFactor("FractionallySizedBox", "WidthFactor", o.WidthFactor)
		assertFactor("FractionallySizedBox", "HeightFactor", o.HeightFactor)
	}
	return w
}

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart FractionallySizedBox.alignment
func (w *FractionallySizedBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// WidthFactor is the fraction of the available width to give the child; null passes the parent's width.
//
// Flutter: widgets/basic.dart FractionallySizedBox.widthFactor
func (w *FractionallySizedBox) WidthFactor() foundation.Opt[float64] { return w.widthFactor }

// HeightFactor is the fraction of the available height to give the child; null passes the parent's height.
//
// Flutter: widgets/basic.dart FractionallySizedBox.heightFactor
func (w *FractionallySizedBox) HeightFactor() foundation.Opt[float64] { return w.heightFactor }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart FractionallySizedBox.createRenderObject
func (w *FractionallySizedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderFractionallySizedOverflowBox(nil, rendering.RenderFractionallySizedOverflowBoxOpts{Alignment: w.alignment,
		WidthFactor: w.widthFactor, HeightFactor: w.heightFactor, TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart FractionallySizedBox.updateRenderObject
func (w *FractionallySizedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderFractionallySizedOverflowBox)
	r.SetAlignment(w.alignment)
	r.SetWidthFactor(w.widthFactor)
	r.SetHeightFactor(w.heightFactor)
	r.SetTextDirection(maybeTextDirection(bc))
}

// FittedBoxOpts are the optional parameters of NewFittedBox.
//
// Flutter: widgets/basic.dart FittedBox.new
type FittedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Fit is how to inscribe the child into the space allocated during layout; the default is BoxFitContain.
	Fit painting.BoxFit
	// Alignment is how to align the child within its parent's bounds; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// ClipBehavior is the content will be clipped (or not) according to this option; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// FittedBox scales and positions its child within itself according to fit.
//
// Flutter: widgets/basic.dart FittedBox
type FittedBox struct {
	SingleChildRenderObjectWidget
	fit          painting.BoxFit
	alignment    painting.AlignmentGeometry
	clipBehavior ui.Clip
}

// NewFittedBox creates the widget.
//
// Flutter: widgets/basic.dart FittedBox.new
func NewFittedBox(child Widget, opts ...FittedBoxOpts) *FittedBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.Fit == painting.BoxFitUnspecified {
		o.Fit = painting.BoxFitContain
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	w := &FittedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), fit: o.Fit, alignment: o.Alignment, clipBehavior: o.ClipBehavior}
	return w
}

// Fit is how to inscribe the child into the space allocated during layout; the default is BoxFitContain.
//
// Flutter: widgets/basic.dart FittedBox.fit
func (w *FittedBox) Fit() painting.BoxFit { return w.fit }

// Alignment is how to align the child within its parent's bounds; the default is Alignment.center.
//
// Flutter: widgets/basic.dart FittedBox.alignment
func (w *FittedBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// ClipBehavior is the content will be clipped (or not) according to this option; the default is ClipHardEdge.
//
// Flutter: widgets/basic.dart FittedBox.clipBehavior
func (w *FittedBox) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart FittedBox.createRenderObject
func (w *FittedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderFittedBox(nil, rendering.RenderFittedBoxOpts{Fit: w.fit, Alignment: w.alignment,
		TextDirection: maybeTextDirection(bc), ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart FittedBox.updateRenderObject
func (w *FittedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderFittedBox)
	r.SetFit(w.fit)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(maybeTextDirection(bc))
	r.SetClipBehavior(w.clipBehavior)
}

// AspectRatioOpts are the optional parameters of NewAspectRatio.
//
// Flutter: widgets/basic.dart AspectRatio.new
type AspectRatioOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// AspectRatio sizes its child to a specific aspect ratio.
//
// Flutter: widgets/basic.dart AspectRatio
type AspectRatio struct {
	SingleChildRenderObjectWidget
	aspectRatio float64
}

// NewAspectRatio creates the widget.
//
// Flutter: widgets/basic.dart AspectRatio.new
func NewAspectRatio(aspectRatio float64, child Widget, opts ...AspectRatioOpts) *AspectRatio {
	o := firstOpts(opts)
	w := &AspectRatio{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), aspectRatio: aspectRatio}
	if foundation.DebugMode && !(aspectRatio > 0) {
		panic(errors.New(tagAspectRatioInvalid, "the aspect ratio must be above zero", "aspectRatio", aspectRatio))
	}
	return w
}

// AspectRatio is the aspect ratio to attempt to use: the width divided by the height.
//
// Flutter: widgets/basic.dart AspectRatio.aspectRatio
func (w *AspectRatio) AspectRatio() float64 { return w.aspectRatio }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart AspectRatio.createRenderObject
func (w *AspectRatio) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderAspectRatio(w.aspectRatio, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart AspectRatio.updateRenderObject
func (w *AspectRatio) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderAspectRatio).SetAspectRatio(w.aspectRatio)
}

// BaselineOpts are the optional parameters of NewBaseline.
//
// Flutter: widgets/basic.dart Baseline.new
type BaselineOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// Baseline positions its child according to the child's baseline.
//
// Flutter: widgets/basic.dart Baseline
type Baseline struct {
	SingleChildRenderObjectWidget
	baseline     float64
	baselineType ui.TextBaseline
}

// NewBaseline creates the widget.
//
// Flutter: widgets/basic.dart Baseline.new
func NewBaseline(baseline float64, baselineType ui.TextBaseline, child Widget, opts ...BaselineOpts) *Baseline {
	o := firstOpts(opts)
	w := &Baseline{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), baseline: baseline, baselineType: baselineType}
	return w
}

// Baseline is the number of logical pixels from the top of this box at which to position the child's baseline.
//
// Flutter: widgets/basic.dart Baseline.baseline
func (w *Baseline) Baseline() float64 { return w.baseline }

// BaselineType is the type of baseline to use for positioning the child.
//
// Flutter: widgets/basic.dart Baseline.baselineType
func (w *Baseline) BaselineType() ui.TextBaseline { return w.baselineType }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Baseline.createRenderObject
func (w *Baseline) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderBaseline(w.baseline, w.baselineType, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Baseline.updateRenderObject
func (w *Baseline) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderBaseline)
	r.SetBaseline(w.baseline)
	r.SetBaselineType(w.baselineType)
}

// IntrinsicWidthOpts are the optional parameters of NewIntrinsicWidth.
//
// Flutter: widgets/basic.dart IntrinsicWidth.new
type IntrinsicWidthOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// StepWidth makes the width a multiple of this value when it is given.
	StepWidth foundation.Opt[float64]
	// StepHeight makes the height a multiple of this value when it is given.
	StepHeight foundation.Opt[float64]
}

// IntrinsicWidth sizes its child to the child's maximum intrinsic width.
//
// Flutter: widgets/basic.dart IntrinsicWidth
type IntrinsicWidth struct {
	SingleChildRenderObjectWidget
	stepWidth  foundation.Opt[float64]
	stepHeight foundation.Opt[float64]
}

// NewIntrinsicWidth creates the widget.
//
// Flutter: widgets/basic.dart IntrinsicWidth.new
func NewIntrinsicWidth(child Widget, opts ...IntrinsicWidthOpts) *IntrinsicWidth {
	o := firstOpts(opts)
	w := &IntrinsicWidth{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), stepWidth: o.StepWidth, stepHeight: o.StepHeight}
	if foundation.DebugMode {
		assertNonNegativeOpt("IntrinsicWidth", "StepWidth", o.StepWidth)
		assertNonNegativeOpt("IntrinsicWidth", "StepHeight", o.StepHeight)
	}
	return w
}

// StepWidth makes the width a multiple of this value when it is given.
//
// Flutter: widgets/basic.dart IntrinsicWidth.stepWidth
func (w *IntrinsicWidth) StepWidth() foundation.Opt[float64] { return w.stepWidth }

// StepHeight makes the height a multiple of this value when it is given.
//
// Flutter: widgets/basic.dart IntrinsicWidth.stepHeight
func (w *IntrinsicWidth) StepHeight() foundation.Opt[float64] { return w.stepHeight }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart IntrinsicWidth.createRenderObject
func (w *IntrinsicWidth) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderIntrinsicWidth(nil, rendering.RenderIntrinsicWidthOpts{StepWidth: w.stepWidth, StepHeight: w.stepHeight})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart IntrinsicWidth.updateRenderObject
func (w *IntrinsicWidth) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderIntrinsicWidth)
	r.SetStepWidth(w.stepWidth)
	r.SetStepHeight(w.stepHeight)
}

// IntrinsicHeightOpts are the optional parameters of NewIntrinsicHeight.
//
// Flutter: widgets/basic.dart IntrinsicHeight.new
type IntrinsicHeightOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// IntrinsicHeight sizes its child to the child's intrinsic height.
//
// Flutter: widgets/basic.dart IntrinsicHeight
type IntrinsicHeight struct {
	SingleChildRenderObjectWidget
}

// NewIntrinsicHeight creates the widget.
//
// Flutter: widgets/basic.dart IntrinsicHeight.new
func NewIntrinsicHeight(child Widget, opts ...IntrinsicHeightOpts) *IntrinsicHeight {
	o := firstOpts(opts)
	w := &IntrinsicHeight{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o))}
	return w
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart IntrinsicHeight.createRenderObject
func (w *IntrinsicHeight) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderIntrinsicHeight(nil)
}

// IgnoreBaselineOpts are the optional parameters of NewIgnoreBaseline.
//
// Flutter: widgets/basic.dart IgnoreBaseline.new
type IgnoreBaselineOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// IgnoreBaseline makes its child's baselines invisible to its ancestors.
//
// Flutter: widgets/basic.dart IgnoreBaseline
type IgnoreBaseline struct {
	SingleChildRenderObjectWidget
}

// NewIgnoreBaseline creates the widget.
//
// Flutter: widgets/basic.dart IgnoreBaseline.new
func NewIgnoreBaseline(child Widget, opts ...IgnoreBaselineOpts) *IgnoreBaseline {
	o := firstOpts(opts)
	w := &IgnoreBaseline{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o))}
	return w
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart IgnoreBaseline.createRenderObject
func (w *IgnoreBaseline) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderIgnoreBaseline(nil)
}

// RotatedBoxOpts are the optional parameters of NewRotatedBox.
//
// Flutter: widgets/basic.dart RotatedBox.new
type RotatedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// RotatedBox rotates its child by a integral number of quarter turns.
//
// Flutter: widgets/basic.dart RotatedBox
type RotatedBox struct {
	SingleChildRenderObjectWidget
	quarterTurns int
}

// NewRotatedBox creates the widget.
//
// Flutter: widgets/basic.dart RotatedBox.new
func NewRotatedBox(quarterTurns int, child Widget, opts ...RotatedBoxOpts) *RotatedBox {
	o := firstOpts(opts)
	w := &RotatedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), quarterTurns: quarterTurns}
	return w
}

// QuarterTurns is the number of clockwise quarter turns the child should be rotated.
//
// Flutter: widgets/basic.dart RotatedBox.quarterTurns
func (w *RotatedBox) QuarterTurns() int { return w.quarterTurns }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart RotatedBox.createRenderObject
func (w *RotatedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderRotatedBox(w.quarterTurns, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart RotatedBox.updateRenderObject
func (w *RotatedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderRotatedBox).SetQuarterTurns(w.quarterTurns)
}

// FractionalTranslationOpts are the optional parameters of NewFractionalTranslation.
//
// Flutter: widgets/basic.dart FractionalTranslation.new
type FractionalTranslationOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TransformHitTests is whether to apply the translation when performing hit tests; the default is true.
	TransformHitTests foundation.Opt[bool]
}

// FractionalTranslation applies a translation expressed as a fraction of the box size before painting its child.
//
// Flutter: widgets/basic.dart FractionalTranslation
type FractionalTranslation struct {
	SingleChildRenderObjectWidget
	translation       ui.Offset
	transformHitTests foundation.Opt[bool]
}

// NewFractionalTranslation creates the widget.
//
// Flutter: widgets/basic.dart FractionalTranslation.new
func NewFractionalTranslation(translation ui.Offset, child Widget, opts ...FractionalTranslationOpts) *FractionalTranslation {
	o := firstOpts(opts)
	w := &FractionalTranslation{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), translation: translation, transformHitTests: o.TransformHitTests}
	return w
}

// Translation is the offset by which to translate the child, as a multiple of its size.
//
// Flutter: widgets/basic.dart FractionalTranslation.translation
func (w *FractionalTranslation) Translation() ui.Offset { return w.translation }

// TransformHitTests is whether to apply the translation when performing hit tests; the default is true.
//
// Flutter: widgets/basic.dart FractionalTranslation.transformHitTests
func (w *FractionalTranslation) TransformHitTests() foundation.Opt[bool] { return w.transformHitTests }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart FractionalTranslation.createRenderObject
func (w *FractionalTranslation) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderFractionalTranslation(w.translation, nil, rendering.RenderFractionalTranslationOpts{TransformHitTests: w.transformHitTests})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart FractionalTranslation.updateRenderObject
func (w *FractionalTranslation) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderFractionalTranslation)
	r.SetTranslation(w.translation)
	r.TransformHitTests = w.transformHitTests.OrElse(true)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAspectRatioInvalid = "AspectRatioInvalid"
)
