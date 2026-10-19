package widgets

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// firstDirection is the text direction the widget was given, or the one of the
// closest Directionality when it was given none.
func firstDirection(given ui.TextDirection, bc BuildContext) ui.TextDirection {
	if given != ui.TextDirectionUnspecified {
		return given
	}
	return maybeTextDirection(bc)
}

// ConstraintsTransformUnconstrained removes every constraint.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.unconstrained
func ConstraintsTransformUnconstrained(rendering.BoxConstraints) rendering.BoxConstraints {
	return rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}
}

// ConstraintsTransformWidthUnconstrained removes the constraints on the width.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.widthUnconstrained
func ConstraintsTransformWidthUnconstrained(c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.CopyWith(rendering.BoxConstraintsCopy{MinWidth: foundation.Some(0.0), MaxWidth: foundation.Some(math.Inf(1))})
}

// ConstraintsTransformHeightUnconstrained removes the constraints on the
// height.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.heightUnconstrained
func ConstraintsTransformHeightUnconstrained(c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.CopyWith(rendering.BoxConstraintsCopy{MinHeight: foundation.Some(0.0), MaxHeight: foundation.Some(math.Inf(1))})
}

// ConstraintsTransformMaxWidthUnconstrained removes the maximum width.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.maxWidthUnconstrained
func ConstraintsTransformMaxWidthUnconstrained(c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.CopyWith(rendering.BoxConstraintsCopy{MaxWidth: foundation.Some(math.Inf(1))})
}

// ConstraintsTransformMaxHeightUnconstrained removes the maximum height.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.maxHeightUnconstrained
func ConstraintsTransformMaxHeightUnconstrained(c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.CopyWith(rendering.BoxConstraintsCopy{MaxHeight: foundation.Some(math.Inf(1))})
}

// ConstraintsTransformMaxUnconstrained removes the maximum width and height.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.maxUnconstrained
func ConstraintsTransformMaxUnconstrained(c rendering.BoxConstraints) rendering.BoxConstraints {
	return c.CopyWith(rendering.BoxConstraintsCopy{MaxWidth: foundation.Some(math.Inf(1)), MaxHeight: foundation.Some(math.Inf(1))})
}

// CenterOpts are the optional parameters of NewCenter.
//
// Flutter: widgets/basic.dart Center.new
type CenterOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// WidthFactor sets the width to the child's width times this factor when it
	// is given.
	WidthFactor foundation.Opt[float64]
	// HeightFactor sets the height to the child's height times this factor when it
	// is given.
	HeightFactor foundation.Opt[float64]
}

// Center is an Align that aligns its child in the center.
//
// Flutter: widgets/basic.dart Center
type Center struct{ Align }

// NewCenter creates a widget that centers its child.
//
// Flutter: widgets/basic.dart Center.new
func NewCenter(child Widget, opts ...CenterOpts) *Center {
	o := firstOpts(opts)
	return &Center{*NewAlign(child, AlignOpts{Key: o.Key, WidthFactor: o.WidthFactor, HeightFactor: o.HeightFactor})}
}

// SizedBoxOpts are the optional parameters of NewSizedBox.
//
// Flutter: widgets/basic.dart SizedBox.new
type SizedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Width is the width the box forces on its child; unset leaves it free.
	Width foundation.Opt[float64]
	// Height is the height the box forces on its child; unset leaves it free.
	Height foundation.Opt[float64]
}

// SizedBox is a box with a specified size; with a child it forces the child to
// have a specific width and/or height.
//
// Flutter: widgets/basic.dart SizedBox
type SizedBox struct {
	SingleChildRenderObjectWidget
	width, height foundation.Opt[float64]
}

// NewSizedBox creates a fixed size box; the width and height can be unset to
// leave them free. The child may be nil.
//
// Flutter: widgets/basic.dart SizedBox.new
func NewSizedBox(child Widget, opts ...SizedBoxOpts) *SizedBox {
	o := firstOpts(opts)
	return &SizedBox{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.Width, o.Height}
}

// NewSizedBoxExpand creates a box that will become as large as its parent
// allows.
//
// Flutter: widgets/basic.dart SizedBox.expand
func NewSizedBoxExpand(child Widget, opts ...WidgetOpts) *SizedBox {
	return NewSizedBox(child, SizedBoxOpts{Key: firstOpts(opts).Key, Width: foundation.Some(math.Inf(1)), Height: foundation.Some(math.Inf(1))})
}

// NewSizedBoxShrink creates a box that will become as small as its parent
// allows.
//
// Flutter: widgets/basic.dart SizedBox.shrink
func NewSizedBoxShrink(child Widget, opts ...WidgetOpts) *SizedBox {
	return NewSizedBox(child, SizedBoxOpts{Key: firstOpts(opts).Key, Width: foundation.Some(0.0), Height: foundation.Some(0.0)})
}

// NewSizedBoxFromSize creates a box with the given size.
//
// Flutter: widgets/basic.dart SizedBox.fromSize
func NewSizedBoxFromSize(size ui.Size, child Widget, opts ...WidgetOpts) *SizedBox {
	return NewSizedBox(child, SizedBoxOpts{Key: firstOpts(opts).Key, Width: foundation.Some(size.Width), Height: foundation.Some(size.Height)})
}

// NewSizedBoxSquare creates a box with the same width and height.
//
// Flutter: widgets/basic.dart SizedBox.square
func NewSizedBoxSquare(dimension foundation.Opt[float64], child Widget, opts ...WidgetOpts) *SizedBox {
	return NewSizedBox(child, SizedBoxOpts{Key: firstOpts(opts).Key, Width: dimension, Height: dimension})
}

// Width is the width the box forces on its child, if any.
//
// Flutter: widgets/basic.dart SizedBox.width
func (w *SizedBox) Width() foundation.Opt[float64] { return w.width }

// Height is the height the box forces on its child, if any.
//
// Flutter: widgets/basic.dart SizedBox.height
func (w *SizedBox) Height() foundation.Opt[float64] { return w.height }

func (w *SizedBox) additionalConstraints() rendering.BoxConstraints {
	return rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: w.width, Height: w.height})
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart SizedBox.createRenderObject
func (w *SizedBox) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedBox(w.additionalConstraints(), nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart SizedBox.updateRenderObject
func (w *SizedBox) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(w.additionalConstraints())
}

// OffstageOpts are the optional parameters of NewOffstage.
//
// Flutter: widgets/basic.dart Offstage.new
type OffstageOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Offstage is whether the child is hidden; the default is true.
	Offstage foundation.Opt[bool]
}

// Offstage lays out its child as if it was in the tree, but without painting
// anything, without making the child available for hit testing, and without
// taking any room in the parent.
//
// Flutter: widgets/basic.dart Offstage
type Offstage struct {
	SingleChildRenderObjectWidget
	offstage bool
}

// NewOffstage creates a widget that visually hides its child.
//
// Flutter: widgets/basic.dart Offstage.new
func NewOffstage(child Widget, opts ...OffstageOpts) *Offstage {
	o := firstOpts(opts)
	return &Offstage{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o.Offstage.OrElse(true)}
}

// IsOffstage is whether the child is hidden. It is the interface that
// gluttertest's finders use to skip the offstage subtrees.
//
// Flutter: widgets/basic.dart Offstage.offstage
func (w *Offstage) IsOffstage() bool { return w.offstage }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Offstage.createRenderObject
func (w *Offstage) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderOffstage(nil, rendering.RenderOffstageOpts{Offstage: foundation.Some(w.offstage)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Offstage.updateRenderObject
func (w *Offstage) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderOffstage).SetOffstage(w.offstage)
}

// CreateElement creates the element, which hides the child from the visitors of
// the onstage children while it is offstage.
//
// Flutter: widgets/basic.dart Offstage.createElement
func (w *Offstage) CreateElement() Element {
	e := &offstageElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// offstageElement is the element of an Offstage.
//
// Flutter: widgets/basic.dart _OffstageElement
type offstageElement struct {
	SingleChildRenderObjectElement
}

// DebugVisitOnstageChildren visits the child unless the widget is offstage.
func (e *offstageElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	if !e.widget.(*Offstage).offstage {
		e.VisitChildren(visitor)
	}
}
