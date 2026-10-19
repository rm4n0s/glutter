package widgets

import (
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// The equality functions of the targets of the implicit animations, which are
// Dart's `==` for each type.

// isNilValue is whether v is a nil interface, or a nil pointer, map, slice or
// function inside one.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	switch r := reflect.ValueOf(v); r.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface:
		return r.IsNil()
	}
	return false
}

func floatEqual(a, b float64) bool { return a == b }

func alignmentGeometryEqual(a, b painting.AlignmentGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func edgeInsetsGeometryEqual(a, b painting.EdgeInsetsGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func decorationEqual(a, b painting.Decoration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case *painting.BoxDecoration:
		y, ok := b.(*painting.BoxDecoration)
		return ok && x.Equal(y)
	case *painting.ShapeDecoration:
		y, ok := b.(*painting.ShapeDecoration)
		return ok && x.Equal(y)
	case *painting.FlutterLogoDecoration:
		y, ok := b.(*painting.FlutterLogoDecoration)
		return ok && x.Equal(y)
	}
	return foundation.ValuesEqual(a, b)
}

func matrixEqual(a, b *ui.Matrix4) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equals(b)
}

func textStyleEqual(a, b *painting.TextStyle) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func comparableEqual[T comparable](a, b T) bool { return a == b }

func floatTween(v float64) *animation.Tween[float64] { return animation.NewTween(v, v) }

// optNonNil is some(v) for a value that is not a nil interface or pointer.
func optNonNil[T any](v T) foundation.Opt[T] {
	if isNilValue(v) {
		return foundation.None[T]()
	}
	return foundation.Some(v)
}

// AnimatedContainerOpts are the optional parameters of NewAnimatedContainer.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.new
type AnimatedContainerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animations; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Alignment aligns the child within the container.
	Alignment painting.AlignmentGeometry
	// Padding is the empty space to inscribe inside the decoration.
	Padding painting.EdgeInsetsGeometry
	// Color is the background color; it is a shorthand for a BoxDecoration with
	// the color, so it cannot be given together with Decoration.
	Color foundation.Opt[ui.Color]
	// Decoration is the decoration to paint behind the child.
	Decoration painting.Decoration
	// ForegroundDecoration is the decoration to paint in front of the child.
	ForegroundDecoration painting.Decoration
	// Width is a tight width for the container, merged into Constraints.
	Width foundation.Opt[float64]
	// Height is a tight height for the container, merged into Constraints.
	Height foundation.Opt[float64]
	// Constraints are additional constraints to apply to the child.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Margin is the empty space to surround the decoration and the child.
	Margin painting.EdgeInsetsGeometry
	// Transform is the transformation matrix to apply before painting the
	// container.
	Transform *ui.Matrix4
	// TransformAlignment is the alignment of the origin of the transform.
	TransformAlignment painting.AlignmentGeometry
	// ClipBehavior is whether to clip the contents to the decoration's shape;
	// the default is ClipNone.
	ClipBehavior ui.Clip
}

// AnimatedContainer is an animated version of Container that gradually changes
// its values over a period of time: when it is rebuilt with a new alignment,
// padding, decoration, constraints, margin or transform, it animates from the
// old value to the new one.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer
type AnimatedContainer struct {
	ImplicitlyAnimatedWidget
	child                Widget
	alignment            painting.AlignmentGeometry
	padding              painting.EdgeInsetsGeometry
	decoration           painting.Decoration
	foregroundDecoration painting.Decoration
	constraints          foundation.Opt[rendering.BoxConstraints]
	margin               painting.EdgeInsetsGeometry
	transform            *ui.Matrix4
	transformAlignment   painting.AlignmentGeometry
	clipBehavior         ui.Clip
}

// NewAnimatedContainer creates the widget. In debug builds it panics with the
// tags of NewContainer for a negative margin or padding, a color together with a
// decoration, and an invalid decoration or constraints.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.new
func NewAnimatedContainer(duration time.Duration, child Widget, opts ...AnimatedContainerOpts) *AnimatedContainer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if foundation.DebugMode {
		if o.Margin != nil && !o.Margin.IsNonNegative() {
			panic(errors.New("ContainerNegativePadding", "the margin of an AnimatedContainer must not be negative"))
		}
		if o.Padding != nil && !o.Padding.IsNonNegative() {
			panic(errors.New("ContainerNegativePadding", "the padding of an AnimatedContainer must not be negative"))
		}
		if o.Decoration != nil {
			o.Decoration.DebugAssertIsValid()
		}
		if c, ok := o.Constraints.Get(); ok {
			c.DebugAssertIsValid()
		}
		if o.Color.IsSome() && o.Decoration != nil {
			panic(errors.New("ContainerColorAndDecoration", "cannot provide both a color and a decoration; the color argument is just a shorthand for a decoration with the color; to use both a color and other decoration properties, set the color in the BoxDecoration instead"))
		}
	}
	decoration := o.Decoration
	if decoration == nil {
		if c, ok := o.Color.Get(); ok {
			decoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(c)})
		}
	}
	constraints := o.Constraints
	if o.Width.IsSome() || o.Height.IsSome() {
		if c, ok := constraints.Get(); ok {
			constraints = foundation.Some(c.Tighten(rendering.TightenOpts{Width: o.Width, Height: o.Height}))
		} else {
			constraints = foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: o.Width, Height: o.Height}))
		}
	}
	return &AnimatedContainer{
		ImplicitlyAnimatedWidget: NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child:                    child, alignment: o.Alignment, padding: o.Padding, decoration: decoration,
		foregroundDecoration: o.ForegroundDecoration, constraints: constraints, margin: o.Margin, transform: o.Transform,
		transformAlignment: o.TransformAlignment, clipBehavior: o.ClipBehavior,
	}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.child
func (w *AnimatedContainer) Child() Widget { return w.child }

// Alignment is how the child is aligned within the container.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.alignment
func (w *AnimatedContainer) Alignment() painting.AlignmentGeometry { return w.alignment }

// Padding is the empty space inside the decoration.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.padding
func (w *AnimatedContainer) Padding() painting.EdgeInsetsGeometry { return w.padding }

// Decoration is the decoration to paint behind the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.decoration
func (w *AnimatedContainer) Decoration() painting.Decoration { return w.decoration }

// ForegroundDecoration is the decoration to paint in front of the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.foregroundDecoration
func (w *AnimatedContainer) ForegroundDecoration() painting.Decoration { return w.foregroundDecoration }

// Constraints are the additional constraints to apply to the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.constraints
func (w *AnimatedContainer) Constraints() foundation.Opt[rendering.BoxConstraints] {
	return w.constraints
}

// Margin is the empty space to surround the decoration and the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.margin
func (w *AnimatedContainer) Margin() painting.EdgeInsetsGeometry { return w.margin }

// Transform is the transformation matrix to apply before painting the
// container.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.transform
func (w *AnimatedContainer) Transform() *ui.Matrix4 { return w.transform }

// TransformAlignment is the alignment of the origin of the transform.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.transformAlignment
func (w *AnimatedContainer) TransformAlignment() painting.AlignmentGeometry {
	return w.transformAlignment
}

// ClipBehavior is whether the contents are clipped to the decoration's shape.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.clipBehavior
func (w *AnimatedContainer) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedContainer.createState
func (w *AnimatedContainer) CreateState() State { return &animatedContainerState{} }

type animatedContainerState struct {
	AnimatedWidgetBaseState[*AnimatedContainer]
	alignment            *animation.Tween[painting.AlignmentGeometry]
	padding              *animation.Tween[painting.EdgeInsetsGeometry]
	decoration           *animation.Tween[painting.Decoration]
	foregroundDecoration *animation.Tween[painting.Decoration]
	constraints          *animation.Tween[rendering.BoxConstraints]
	margin               *animation.Tween[painting.EdgeInsetsGeometry]
	transform            *animation.Tween[*ui.Matrix4]
	transformAlignment   *animation.Tween[painting.AlignmentGeometry]
}

func (s *animatedContainerState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	alignment := func(x painting.AlignmentGeometry) *animation.Tween[painting.AlignmentGeometry] {
		return rendering.NewAlignmentGeometryTween(x, x)
	}
	insets := func(x painting.EdgeInsetsGeometry) *animation.Tween[painting.EdgeInsetsGeometry] {
		return NewEdgeInsetsGeometryTween(x, x)
	}
	decoration := func(x painting.Decoration) *animation.Tween[painting.Decoration] { return NewDecorationTween(x, x) }
	VisitTween(v, &s.alignment, optNonNil(w.alignment), alignmentGeometryEqual, alignment)
	VisitTween(v, &s.padding, optNonNil(w.padding), edgeInsetsGeometryEqual, insets)
	VisitTween(v, &s.decoration, optNonNil(w.decoration), decorationEqual, decoration)
	VisitTween(v, &s.foregroundDecoration, optNonNil(w.foregroundDecoration), decorationEqual, decoration)
	VisitTween(v, &s.constraints, w.constraints, comparableEqual[rendering.BoxConstraints],
		func(x rendering.BoxConstraints) *animation.Tween[rendering.BoxConstraints] {
			return NewBoxConstraintsTween(x, x)
		})
	VisitTween(v, &s.margin, optNonNil(w.margin), edgeInsetsGeometryEqual, insets)
	VisitTween(v, &s.transform, optNonNil(w.transform), matrixEqual,
		func(x *ui.Matrix4) *animation.Tween[*ui.Matrix4] { return NewMatrix4Tween(x, x) })
	VisitTween(v, &s.transformAlignment, optNonNil(w.transformAlignment), alignmentGeometryEqual, alignment)
}

func (s *animatedContainerState) Build(BuildContext) Widget {
	a := s.Animation()
	pick := func(t *animation.Tween[painting.AlignmentGeometry]) painting.AlignmentGeometry {
		if t == nil {
			return nil
		}
		return t.Evaluate(a)
	}
	var o ContainerOpts
	o.Alignment = pick(s.alignment)
	if s.padding != nil {
		o.Padding = s.padding.Evaluate(a)
	}
	if s.decoration != nil {
		o.Decoration = s.decoration.Evaluate(a)
	}
	if s.foregroundDecoration != nil {
		o.ForegroundDecoration = s.foregroundDecoration.Evaluate(a)
	}
	o.Constraints = TweenOf(s.constraints, a)
	if s.margin != nil {
		o.Margin = s.margin.Evaluate(a)
	}
	if s.transform != nil {
		o.Transform = s.transform.Evaluate(a)
	}
	o.TransformAlignment = pick(s.transformAlignment)
	o.ClipBehavior = s.Widget().clipBehavior
	return NewContainer(s.Widget().child, o)
}

// AnimatedPaddingOpts are the optional parameters of NewAnimatedPadding.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding.new
type AnimatedPaddingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
}

// AnimatedPadding is the animated version of Padding: it animates the insets
// when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding
type AnimatedPadding struct {
	ImplicitlyAnimatedWidget
	padding painting.EdgeInsetsGeometry
	child   Widget
}

// NewAnimatedPadding creates the widget. In debug builds it panics with the tag
// ContainerNegativePadding when the padding is negative.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding.new
func NewAnimatedPadding(padding painting.EdgeInsetsGeometry, duration time.Duration, child Widget, opts ...AnimatedPaddingOpts) *AnimatedPadding {
	o := firstOpts(opts)
	if foundation.DebugMode && !padding.IsNonNegative() {
		panic(errors.New("ContainerNegativePadding", "the padding of an AnimatedPadding must not be negative"))
	}
	return &AnimatedPadding{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts(o)), padding, child}
}

// Padding is the amount of space by which to inset the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding.padding
func (w *AnimatedPadding) Padding() painting.EdgeInsetsGeometry { return w.padding }

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding.child
func (w *AnimatedPadding) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedPadding.createState
func (w *AnimatedPadding) CreateState() State { return &animatedPaddingState{} }

type animatedPaddingState struct {
	AnimatedWidgetBaseState[*AnimatedPadding]
	padding *animation.Tween[painting.EdgeInsetsGeometry]
}

func (s *animatedPaddingState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.padding, optNonNil(s.Widget().padding), edgeInsetsGeometryEqual,
		func(x painting.EdgeInsetsGeometry) *animation.Tween[painting.EdgeInsetsGeometry] {
			return NewEdgeInsetsGeometryTween(x, x)
		})
}

func (s *animatedPaddingState) Build(BuildContext) Widget {
	padding := s.padding.Evaluate(s.Animation()).Clamp(painting.EdgeInsetsZero, painting.EdgeInsetsGeometryInfinity)
	return NewPadding(padding, s.Widget().child)
}

// AnimatedAlignOpts are the optional parameters of NewAnimatedAlign.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.new
type AnimatedAlignOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// HeightFactor sets the height to the child's height times this factor.
	HeightFactor foundation.Opt[float64]
	// WidthFactor sets the width to the child's width times this factor.
	WidthFactor foundation.Opt[float64]
}

// AnimatedAlign is the animated version of Align: it animates the alignment
// (and the size factors) when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign
type AnimatedAlign struct {
	ImplicitlyAnimatedWidget
	alignment    painting.AlignmentGeometry
	child        Widget
	heightFactor foundation.Opt[float64]
	widthFactor  foundation.Opt[float64]
}

// NewAnimatedAlign creates the widget. In debug builds it panics with the tag
// AlignFactorNegative when a factor is negative.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.new
func NewAnimatedAlign(alignment painting.AlignmentGeometry, duration time.Duration, child Widget, opts ...AnimatedAlignOpts) *AnimatedAlign {
	o := firstOpts(opts)
	assertFactors("AnimatedAlign", o.WidthFactor, o.HeightFactor)
	return &AnimatedAlign{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		alignment, child, o.HeightFactor, o.WidthFactor}
}

func assertFactors(widget string, width, height foundation.Opt[float64]) {
	if !foundation.DebugMode {
		return
	}
	if v, ok := width.Get(); ok && v < 0 {
		panic(errors.New("AlignFactorNegative", "the widthFactor of an "+widget+" must not be negative", "widthFactor", v))
	}
	if v, ok := height.Get(); ok && v < 0 {
		panic(errors.New("AlignFactorNegative", "the heightFactor of an "+widget+" must not be negative", "heightFactor", v))
	}
}

// Alignment is how to align the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.alignment
func (w *AnimatedAlign) Alignment() painting.AlignmentGeometry { return w.alignment }

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.child
func (w *AnimatedAlign) Child() Widget { return w.child }

// HeightFactor is the factor of the child's height.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.heightFactor
func (w *AnimatedAlign) HeightFactor() foundation.Opt[float64] { return w.heightFactor }

// WidthFactor is the factor of the child's width.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.widthFactor
func (w *AnimatedAlign) WidthFactor() foundation.Opt[float64] { return w.widthFactor }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedAlign.createState
func (w *AnimatedAlign) CreateState() State { return &animatedAlignState{} }

type animatedAlignState struct {
	AnimatedWidgetBaseState[*AnimatedAlign]
	alignment    *animation.Tween[painting.AlignmentGeometry]
	heightFactor *animation.Tween[float64]
	widthFactor  *animation.Tween[float64]
}

func (s *animatedAlignState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	VisitTween(v, &s.alignment, optNonNil(w.alignment), alignmentGeometryEqual,
		func(x painting.AlignmentGeometry) *animation.Tween[painting.AlignmentGeometry] {
			return rendering.NewAlignmentGeometryTween(x, x)
		})
	// Like Flutter, a factor that becomes none keeps its old tween.
	if w.heightFactor.IsSome() {
		VisitTween(v, &s.heightFactor, w.heightFactor, floatEqual, floatTween)
	}
	if w.widthFactor.IsSome() {
		VisitTween(v, &s.widthFactor, w.widthFactor, floatEqual, floatTween)
	}
}

func (s *animatedAlignState) Build(BuildContext) Widget {
	a := s.Animation()
	return NewAlign(s.Widget().child, AlignOpts{Alignment: s.alignment.Evaluate(a),
		HeightFactor: TweenOf(s.heightFactor, a), WidthFactor: TweenOf(s.widthFactor, a)})
}

// AnimatedPositionedOpts are the optional parameters of NewAnimatedPositioned.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.new
type AnimatedPositionedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Left is the offset of the child's left edge from the left of the stack.
	Left foundation.Opt[float64]
	// Top is the offset of the child's top edge from the top of the stack.
	Top foundation.Opt[float64]
	// Right is the offset of the child's right edge from the right of the stack.
	Right foundation.Opt[float64]
	// Bottom is the offset of the child's bottom edge from the bottom of the
	// stack.
	Bottom foundation.Opt[float64]
	// Width is the child's width.
	Width foundation.Opt[float64]
	// Height is the child's height.
	Height foundation.Opt[float64]
}

// AnimatedPositioned is the animated version of Positioned, which must be a
// descendant of a Stack: it animates the position and size when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned
type AnimatedPositioned struct {
	ImplicitlyAnimatedWidget
	child                                   Widget
	left, top, right, bottom, width, height foundation.Opt[float64]
}

// NewAnimatedPositioned creates the widget. In debug builds it panics with the
// tag PositionedOverconstrained when left, right and width, or top, bottom and
// height, are all given.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.new
func NewAnimatedPositioned(duration time.Duration, child Widget, opts ...AnimatedPositionedOpts) *AnimatedPositioned {
	o := firstOpts(opts)
	assertNotOverconstrained("AnimatedPositioned", o.Left, o.Right, o.Width, o.Top, o.Bottom, o.Height)
	return &AnimatedPositioned{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, o.Left, o.Top, o.Right, o.Bottom, o.Width, o.Height}
}

func assertNotOverconstrained(widget string, a, b, size, c, d, size2 foundation.Opt[float64]) {
	if foundation.DebugMode && ((a.IsSome() && b.IsSome() && size.IsSome()) || (c.IsSome() && d.IsSome() && size2.IsSome())) {
		panic(errors.New("PositionedOverconstrained", "an "+widget+" can't have all three of its start, end and size (or top, bottom and size) set"))
	}
}

// NewAnimatedPositionedFromRect creates an AnimatedPositioned with the left,
// top, width and height of the rectangle.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.fromRect
func NewAnimatedPositionedFromRect(rect ui.Rect, duration time.Duration, child Widget, opts ...AnimatedPositionedOpts) *AnimatedPositioned {
	o := firstOpts(opts)
	o.Left, o.Top = foundation.Some(rect.Left), foundation.Some(rect.Top)
	o.Width, o.Height = foundation.Some(rect.Width()), foundation.Some(rect.Height())
	o.Right, o.Bottom = foundation.None[float64](), foundation.None[float64]()
	return NewAnimatedPositioned(duration, child, o)
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.child
func (w *AnimatedPositioned) Child() Widget { return w.child }

// Left is the offset of the child's left edge from the left of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.left
func (w *AnimatedPositioned) Left() foundation.Opt[float64] { return w.left }

// Top is the offset of the child's top edge from the top of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.top
func (w *AnimatedPositioned) Top() foundation.Opt[float64] { return w.top }

// Right is the offset of the child's right edge from the right of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.right
func (w *AnimatedPositioned) Right() foundation.Opt[float64] { return w.right }

// Bottom is the offset of the child's bottom edge from the bottom of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.bottom
func (w *AnimatedPositioned) Bottom() foundation.Opt[float64] { return w.bottom }

// Width is the child's width.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.width
func (w *AnimatedPositioned) Width() foundation.Opt[float64] { return w.width }

// Height is the child's height.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.height
func (w *AnimatedPositioned) Height() foundation.Opt[float64] { return w.height }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositioned.createState
func (w *AnimatedPositioned) CreateState() State { return &animatedPositionedState{} }

type animatedPositionedState struct {
	AnimatedWidgetBaseState[*AnimatedPositioned]
	left, top, right, bottom, width, height *animation.Tween[float64]
}

func (s *animatedPositionedState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	VisitTween(v, &s.left, w.left, floatEqual, floatTween)
	VisitTween(v, &s.top, w.top, floatEqual, floatTween)
	VisitTween(v, &s.right, w.right, floatEqual, floatTween)
	VisitTween(v, &s.bottom, w.bottom, floatEqual, floatTween)
	VisitTween(v, &s.width, w.width, floatEqual, floatTween)
	VisitTween(v, &s.height, w.height, floatEqual, floatTween)
}

func (s *animatedPositionedState) Build(BuildContext) Widget {
	a := s.Animation()
	return NewPositioned(s.Widget().child, PositionedOpts{Left: TweenOf(s.left, a), Top: TweenOf(s.top, a),
		Right: TweenOf(s.right, a), Bottom: TweenOf(s.bottom, a), Width: TweenOf(s.width, a), Height: TweenOf(s.height, a)})
}

// AnimatedPositionedDirectionalOpts are the optional parameters of
// NewAnimatedPositionedDirectional.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.new
type AnimatedPositionedDirectionalOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Start is the offset of the child's start edge from the start of the stack.
	Start foundation.Opt[float64]
	// Top is the offset of the child's top edge from the top of the stack.
	Top foundation.Opt[float64]
	// End is the offset of the child's end edge from the end of the stack.
	End foundation.Opt[float64]
	// Bottom is the offset of the child's bottom edge from the bottom of the
	// stack.
	Bottom foundation.Opt[float64]
	// Width is the child's width.
	Width foundation.Opt[float64]
	// Height is the child's height.
	Height foundation.Opt[float64]
}

// AnimatedPositionedDirectional is the animated version of PositionedDirectional:
// it animates the position and size when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional
type AnimatedPositionedDirectional struct {
	ImplicitlyAnimatedWidget
	child                                  Widget
	start, top, end, bottom, width, height foundation.Opt[float64]
}

// NewAnimatedPositionedDirectional creates the widget. In debug builds it panics
// with the tag PositionedOverconstrained when start, end and width, or top,
// bottom and height, are all given.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.new
func NewAnimatedPositionedDirectional(duration time.Duration, child Widget, opts ...AnimatedPositionedDirectionalOpts) *AnimatedPositionedDirectional {
	o := firstOpts(opts)
	assertNotOverconstrained("AnimatedPositionedDirectional", o.Start, o.End, o.Width, o.Top, o.Bottom, o.Height)
	return &AnimatedPositionedDirectional{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, o.Start, o.Top, o.End, o.Bottom, o.Width, o.Height}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.child
func (w *AnimatedPositionedDirectional) Child() Widget { return w.child }

// Start is the offset of the child's start edge from the start of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.start
func (w *AnimatedPositionedDirectional) Start() foundation.Opt[float64] { return w.start }

// Top is the offset of the child's top edge from the top of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.top
func (w *AnimatedPositionedDirectional) Top() foundation.Opt[float64] { return w.top }

// End is the offset of the child's end edge from the end of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.end
func (w *AnimatedPositionedDirectional) End() foundation.Opt[float64] { return w.end }

// Bottom is the offset of the child's bottom edge from the bottom of the stack.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.bottom
func (w *AnimatedPositionedDirectional) Bottom() foundation.Opt[float64] { return w.bottom }

// Width is the child's width.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.width
func (w *AnimatedPositionedDirectional) Width() foundation.Opt[float64] { return w.width }

// Height is the child's height.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.height
func (w *AnimatedPositionedDirectional) Height() foundation.Opt[float64] { return w.height }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedPositionedDirectional.createState
func (w *AnimatedPositionedDirectional) CreateState() State {
	return &animatedPositionedDirectionalState{}
}

type animatedPositionedDirectionalState struct {
	AnimatedWidgetBaseState[*AnimatedPositionedDirectional]
	start, top, end, bottom, width, height *animation.Tween[float64]
}

func (s *animatedPositionedDirectionalState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	VisitTween(v, &s.start, w.start, floatEqual, floatTween)
	VisitTween(v, &s.top, w.top, floatEqual, floatTween)
	VisitTween(v, &s.end, w.end, floatEqual, floatTween)
	VisitTween(v, &s.bottom, w.bottom, floatEqual, floatTween)
	VisitTween(v, &s.width, w.width, floatEqual, floatTween)
	VisitTween(v, &s.height, w.height, floatEqual, floatTween)
}

func (s *animatedPositionedDirectionalState) Build(bc BuildContext) Widget {
	DebugCheckHasDirectionality(bc)
	a := s.Animation()
	return NewPositionedWithDirection(DirectionalityOf(bc), s.Widget().child, PositionedDirectionalOpts{Start: TweenOf(s.start, a),
		Top: TweenOf(s.top, a), End: TweenOf(s.end, a), Bottom: TweenOf(s.bottom, a), Width: TweenOf(s.width, a), Height: TweenOf(s.height, a)})
}

// AnimatedScaleOpts are the optional parameters of NewAnimatedScale.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.new
type AnimatedScaleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Alignment is the alignment of the origin of the scale; the default is
	// Alignment.center.
	Alignment foundation.Opt[painting.Alignment]
	// FilterQuality is the filter quality of the child while it animates.
	FilterQuality foundation.Opt[ui.FilterQuality]
}

// AnimatedScale animates the scale of a widget when it changes.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale
type AnimatedScale struct {
	ImplicitlyAnimatedWidget
	child         Widget
	scale         float64
	alignment     painting.Alignment
	filterQuality foundation.Opt[ui.FilterQuality]
}

// NewAnimatedScale creates the widget.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.new
func NewAnimatedScale(scale float64, duration time.Duration, child Widget, opts ...AnimatedScaleOpts) *AnimatedScale {
	o := firstOpts(opts)
	return &AnimatedScale{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, scale, o.Alignment.OrElse(painting.AlignmentCenter), o.FilterQuality}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.child
func (w *AnimatedScale) Child() Widget { return w.child }

// Scale is the target scale.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.scale
func (w *AnimatedScale) Scale() float64 { return w.scale }

// Alignment is the alignment of the origin of the scale.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.alignment
func (w *AnimatedScale) Alignment() painting.Alignment { return w.alignment }

// FilterQuality is the filter quality of the child while it animates.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.filterQuality
func (w *AnimatedScale) FilterQuality() foundation.Opt[ui.FilterQuality] { return w.filterQuality }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedScale.createState
func (w *AnimatedScale) CreateState() State { return &animatedScaleState{} }

type animatedScaleState struct {
	ImplicitlyAnimatedWidgetState[*AnimatedScale]
	scale          *animation.Tween[float64]
	scaleAnimation animation.Animation[float64]
}

func (s *animatedScaleState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.scale, foundation.Some(s.Widget().scale), floatEqual, floatTween)
}

func (s *animatedScaleState) DidUpdateTweens() {
	s.scaleAnimation = animation.Drive(s.Animation(), animation.Animatable[float64](s.scale))
}

func (s *animatedScaleState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewScaleTransition(s.scaleAnimation, w.child, ScaleTransitionOpts{Alignment: w.alignment, FilterQuality: w.filterQuality})
}

// AnimatedRotationOpts are the optional parameters of NewAnimatedRotation.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.new
type AnimatedRotationOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Alignment is the alignment of the origin of the rotation; the default is
	// Alignment.center.
	Alignment foundation.Opt[painting.Alignment]
	// FilterQuality is the filter quality of the child while it animates.
	FilterQuality foundation.Opt[ui.FilterQuality]
}

// AnimatedRotation animates the rotation of a widget when it changes.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation
type AnimatedRotation struct {
	ImplicitlyAnimatedWidget
	child         Widget
	turns         float64
	alignment     painting.Alignment
	filterQuality foundation.Opt[ui.FilterQuality]
}

// NewAnimatedRotation creates the widget; one turn is a full rotation.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.new
func NewAnimatedRotation(turns float64, duration time.Duration, child Widget, opts ...AnimatedRotationOpts) *AnimatedRotation {
	o := firstOpts(opts)
	return &AnimatedRotation{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, turns, o.Alignment.OrElse(painting.AlignmentCenter), o.FilterQuality}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.child
func (w *AnimatedRotation) Child() Widget { return w.child }

// Turns is the target rotation, in turns.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.turns
func (w *AnimatedRotation) Turns() float64 { return w.turns }

// Alignment is the alignment of the origin of the rotation.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.alignment
func (w *AnimatedRotation) Alignment() painting.Alignment { return w.alignment }

// FilterQuality is the filter quality of the child while it animates.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.filterQuality
func (w *AnimatedRotation) FilterQuality() foundation.Opt[ui.FilterQuality] { return w.filterQuality }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedRotation.createState
func (w *AnimatedRotation) CreateState() State { return &animatedRotationState{} }

type animatedRotationState struct {
	ImplicitlyAnimatedWidgetState[*AnimatedRotation]
	turns          *animation.Tween[float64]
	turnsAnimation animation.Animation[float64]
}

func (s *animatedRotationState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.turns, foundation.Some(s.Widget().turns), floatEqual, floatTween)
}

func (s *animatedRotationState) DidUpdateTweens() {
	s.turnsAnimation = animation.Drive(s.Animation(), animation.Animatable[float64](s.turns))
}

func (s *animatedRotationState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewRotationTransition(s.turnsAnimation, w.child, RotationTransitionOpts{Alignment: w.alignment, FilterQuality: w.filterQuality})
}

// AnimatedSlideOpts are the optional parameters of NewAnimatedSlide.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide.new
type AnimatedSlideOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
}

// AnimatedSlide animates the position of a widget relative to its normal
// position, as a fraction of its size, when the offset changes.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide
type AnimatedSlide struct {
	ImplicitlyAnimatedWidget
	child  Widget
	offset ui.Offset
}

// NewAnimatedSlide creates the widget.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide.new
func NewAnimatedSlide(offset ui.Offset, duration time.Duration, child Widget, opts ...AnimatedSlideOpts) *AnimatedSlide {
	o := firstOpts(opts)
	return &AnimatedSlide{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts(o)), child, offset}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide.child
func (w *AnimatedSlide) Child() Widget { return w.child }

// Offset is the target offset, as a fraction of the size of the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide.offset
func (w *AnimatedSlide) Offset() ui.Offset { return w.offset }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedSlide.createState
func (w *AnimatedSlide) CreateState() State { return &animatedSlideState{} }

type animatedSlideState struct {
	ImplicitlyAnimatedWidgetState[*AnimatedSlide]
	offset          *animation.Tween[ui.Offset]
	offsetAnimation animation.Animation[ui.Offset]
}

func (s *animatedSlideState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.offset, foundation.Some(s.Widget().offset), comparableEqual[ui.Offset],
		func(x ui.Offset) *animation.Tween[ui.Offset] { return animation.NewTween(x, x) })
}

func (s *animatedSlideState) DidUpdateTweens() {
	s.offsetAnimation = animation.Drive(s.Animation(), animation.Animatable[ui.Offset](s.offset))
}

func (s *animatedSlideState) Build(BuildContext) Widget {
	return NewSlideTransition(s.offsetAnimation, s.Widget().child)
}

// AnimatedOpacityOpts are the optional parameters of NewAnimatedOpacity.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.new
type AnimatedOpacityOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// AlwaysIncludeSemantics is whether the semantics of the child are included
	// even when it is fully transparent.
	AlwaysIncludeSemantics bool
}

// AnimatedOpacity is the animated version of Opacity: it animates the opacity of
// its child when it changes.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity
type AnimatedOpacity struct {
	ImplicitlyAnimatedWidget
	child                  Widget
	opacity                float64
	alwaysIncludeSemantics bool
}

// NewAnimatedOpacity creates the widget. In debug builds it panics with the tag
// OpacityOutOfRange when the opacity is not between 0 and 1.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.new
func NewAnimatedOpacity(opacity float64, duration time.Duration, child Widget, opts ...AnimatedOpacityOpts) *AnimatedOpacity {
	o := firstOpts(opts)
	if foundation.DebugMode && !(opacity >= 0 && opacity <= 1) {
		panic(errors.New("OpacityOutOfRange", "the opacity must be between 0.0 and 1.0", "opacity", opacity))
	}
	return &AnimatedOpacity{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, opacity, o.AlwaysIncludeSemantics}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.child
func (w *AnimatedOpacity) Child() Widget { return w.child }

// Opacity is the target opacity, from 0 (transparent) to 1 (opaque).
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.opacity
func (w *AnimatedOpacity) Opacity() float64 { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics of the child are included
// even when it is fully transparent.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.alwaysIncludeSemantics
func (w *AnimatedOpacity) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedOpacity.createState
func (w *AnimatedOpacity) CreateState() State { return &animatedOpacityState{} }

type animatedOpacityState struct {
	ImplicitlyAnimatedWidgetState[*AnimatedOpacity]
	opacity          *animation.Tween[float64]
	opacityAnimation animation.Animation[float64]
}

func (s *animatedOpacityState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.opacity, foundation.Some(s.Widget().opacity), floatEqual, floatTween)
}

func (s *animatedOpacityState) DidUpdateTweens() {
	s.opacityAnimation = animation.Drive(s.Animation(), animation.Animatable[float64](s.opacity))
}

func (s *animatedOpacityState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewFadeTransition(s.opacityAnimation, w.child, FadeTransitionOpts{AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}

// AnimatedDefaultTextStyleOpts are the optional parameters of
// NewAnimatedDefaultTextStyle.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.new
type AnimatedDefaultTextStyleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// TextAlign is how the text is aligned horizontally.
	TextAlign ui.TextAlign
	// SoftWrap is whether the text wraps at the box width; the default is true.
	SoftWrap foundation.Opt[bool]
	// Overflow is how visual overflow is handled; the default is
	// TextOverflowClip.
	Overflow painting.TextOverflow
	// MaxLines is the maximum number of lines; it must be greater than zero.
	MaxLines foundation.Opt[int]
	// TextWidthBasis is how the width of the text is measured; the default is
	// TextWidthBasisParent.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first ascent and
	// the last descent.
	TextHeightBehavior *ui.TextHeightBehavior
}

// AnimatedDefaultTextStyle animates the default text style of its subtree when
// the style changes.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle
type AnimatedDefaultTextStyle struct {
	ImplicitlyAnimatedWidget
	child Widget
	style *painting.TextStyle
	opts  AnimatedDefaultTextStyleOpts
}

// NewAnimatedDefaultTextStyle creates the widget. In debug builds it panics
// with the tag TextMaxLinesInvalid when MaxLines is not greater than zero.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.new
func NewAnimatedDefaultTextStyle(style *painting.TextStyle, duration time.Duration, child Widget, opts ...AnimatedDefaultTextStyleOpts) *AnimatedDefaultTextStyle {
	o := firstOpts(opts)
	assertMaxLines(o.MaxLines)
	if o.Overflow == painting.TextOverflowUnspecified {
		o.Overflow = painting.TextOverflowClip
	}
	if o.TextWidthBasis == painting.TextWidthBasisUnspecified {
		o.TextWidthBasis = painting.TextWidthBasisParent
	}
	return &AnimatedDefaultTextStyle{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, style, o}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.child
func (w *AnimatedDefaultTextStyle) Child() Widget { return w.child }

// Style is the target text style.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.style
func (w *AnimatedDefaultTextStyle) Style() *painting.TextStyle { return w.style }

// TextAlign is how the text is aligned horizontally.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.textAlign
func (w *AnimatedDefaultTextStyle) TextAlign() ui.TextAlign { return w.opts.TextAlign }

// SoftWrap is whether the text wraps at the box width.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.softWrap
func (w *AnimatedDefaultTextStyle) SoftWrap() bool { return w.opts.SoftWrap.OrElse(true) }

// Overflow is how visual overflow is handled.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.overflow
func (w *AnimatedDefaultTextStyle) Overflow() painting.TextOverflow { return w.opts.Overflow }

// MaxLines is the maximum number of lines.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.maxLines
func (w *AnimatedDefaultTextStyle) MaxLines() foundation.Opt[int] { return w.opts.MaxLines }

// TextWidthBasis is how the width of the text is measured.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.textWidthBasis
func (w *AnimatedDefaultTextStyle) TextWidthBasis() painting.TextWidthBasis {
	return w.opts.TextWidthBasis
}

// TextHeightBehavior says how the line height applies to the first ascent and
// the last descent.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.textHeightBehavior
func (w *AnimatedDefaultTextStyle) TextHeightBehavior() *ui.TextHeightBehavior {
	return w.opts.TextHeightBehavior
}

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedDefaultTextStyle.createState
func (w *AnimatedDefaultTextStyle) CreateState() State { return &animatedDefaultTextStyleState{} }

type animatedDefaultTextStyleState struct {
	AnimatedWidgetBaseState[*AnimatedDefaultTextStyle]
	style *animation.Tween[*painting.TextStyle]
}

func (s *animatedDefaultTextStyleState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.style, optNonNil(s.Widget().style), textStyleEqual,
		func(x *painting.TextStyle) *animation.Tween[*painting.TextStyle] { return NewTextStyleTween(x, x) })
}

func (s *animatedDefaultTextStyleState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewDefaultTextStyle(s.style.Evaluate(s.Animation()), w.child, DefaultTextStyleOpts{TextAlign: w.opts.TextAlign,
		SoftWrap: w.opts.SoftWrap, Overflow: w.opts.Overflow, MaxLines: w.opts.MaxLines, TextWidthBasis: w.opts.TextWidthBasis,
		TextHeightBehavior: w.opts.TextHeightBehavior})
}

// AnimatedPhysicalModelOpts are the optional parameters of
// NewAnimatedPhysicalModel.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.new
type AnimatedPhysicalModelOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Shape is the type of shape; the default is BoxShapeRectangle.
	Shape painting.BoxShape
	// ClipBehavior is whether the content is clipped and how; the default is
	// ClipNone.
	ClipBehavior ui.Clip
	// BorderRadius is the border radius of the rounded corners when the shape is
	// a rectangle.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Elevation is the z-coordinate at which to place this physical object; it
	// must not be negative.
	Elevation float64
	// AnimateColor is whether to animate the color; the default is true.
	AnimateColor foundation.Opt[bool]
	// AnimateShadowColor is whether to animate the shadow color; the default is
	// true.
	AnimateShadowColor foundation.Opt[bool]
}

// AnimatedPhysicalModel is the animated version of PhysicalModel: it animates
// the elevation, the colors and the border radius when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel
type AnimatedPhysicalModel struct {
	ImplicitlyAnimatedWidget
	child       Widget
	color       ui.Color
	shadowColor ui.Color
	opts        AnimatedPhysicalModelOpts
}

// NewAnimatedPhysicalModel creates the widget. In debug builds it panics with the
// tag PhysicalModelElevationNegative when the elevation is negative.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.new
func NewAnimatedPhysicalModel(color, shadowColor ui.Color, duration time.Duration, child Widget, opts ...AnimatedPhysicalModelOpts) *AnimatedPhysicalModel {
	o := firstOpts(opts)
	if foundation.DebugMode && !(o.Elevation >= 0) {
		panic(errors.New("PhysicalModelElevationNegative", "the elevation of an AnimatedPhysicalModel must not be negative", "elevation", o.Elevation))
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if o.Shape == painting.BoxShapeUnspecified {
		o.Shape = painting.BoxShapeRectangle
	}
	return &AnimatedPhysicalModel{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, color, shadowColor, o}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.child
func (w *AnimatedPhysicalModel) Child() Widget { return w.child }

// Color is the target background color.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.color
func (w *AnimatedPhysicalModel) Color() ui.Color { return w.color }

// ShadowColor is the target shadow color.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.shadowColor
func (w *AnimatedPhysicalModel) ShadowColor() ui.Color { return w.shadowColor }

// Elevation is the target elevation.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.elevation
func (w *AnimatedPhysicalModel) Elevation() float64 { return w.opts.Elevation }

// Shape is the type of shape.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.shape
func (w *AnimatedPhysicalModel) Shape() painting.BoxShape { return w.opts.Shape }

// ClipBehavior is whether the content is clipped and how.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.clipBehavior
func (w *AnimatedPhysicalModel) ClipBehavior() ui.Clip { return w.opts.ClipBehavior }

// BorderRadius is the target border radius.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.borderRadius
func (w *AnimatedPhysicalModel) BorderRadius() foundation.Opt[painting.BorderRadius] {
	return w.opts.BorderRadius
}

// AnimateColor is whether the color is animated.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.animateColor
func (w *AnimatedPhysicalModel) AnimateColor() bool { return w.opts.AnimateColor.OrElse(true) }

// AnimateShadowColor is whether the shadow color is animated.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.animateShadowColor
func (w *AnimatedPhysicalModel) AnimateShadowColor() bool {
	return w.opts.AnimateShadowColor.OrElse(true)
}

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedPhysicalModel.createState
func (w *AnimatedPhysicalModel) CreateState() State { return &animatedPhysicalModelState{} }

type animatedPhysicalModelState struct {
	AnimatedWidgetBaseState[*AnimatedPhysicalModel]
	borderRadius *animation.Tween[foundation.Opt[painting.BorderRadius]]
	elevation    *animation.Tween[float64]
	color        *animation.Tween[foundation.Opt[ui.Color]]
	shadowColor  *animation.Tween[foundation.Opt[ui.Color]]
}

func (s *animatedPhysicalModelState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	colorTween := func(x foundation.Opt[ui.Color]) *animation.Tween[foundation.Opt[ui.Color]] {
		return animation.NewColorTween(x, x)
	}
	radius := w.opts.BorderRadius.OrElse(painting.BorderRadiusZero)
	VisitTween(v, &s.borderRadius, foundation.Some(foundation.Some(radius)), comparableEqual[foundation.Opt[painting.BorderRadius]],
		func(x foundation.Opt[painting.BorderRadius]) *animation.Tween[foundation.Opt[painting.BorderRadius]] {
			return NewBorderRadiusTween(x, x)
		})
	VisitTween(v, &s.elevation, foundation.Some(w.opts.Elevation), floatEqual, floatTween)
	VisitTween(v, &s.color, foundation.Some(foundation.Some(w.color)), comparableEqual[foundation.Opt[ui.Color]], colorTween)
	VisitTween(v, &s.shadowColor, foundation.Some(foundation.Some(w.shadowColor)), comparableEqual[foundation.Opt[ui.Color]], colorTween)
}

func (s *animatedPhysicalModelState) Build(BuildContext) Widget {
	w := s.Widget()
	a := s.Animation()
	color, shadow := w.color, w.shadowColor
	if w.AnimateColor() {
		color = s.color.Evaluate(a).Value()
	}
	if w.AnimateShadowColor() {
		shadow = s.shadowColor.Evaluate(a).Value()
	}
	return NewPhysicalModel(color, w.child, PhysicalModelOpts{Shape: w.opts.Shape, ClipBehavior: w.opts.ClipBehavior,
		BorderRadius: s.borderRadius.Evaluate(a), Elevation: s.elevation.Evaluate(a), ShadowColor: foundation.Some(shadow)})
}

// AnimatedFractionallySizedBoxOpts are the optional parameters of
// NewAnimatedFractionallySizedBox.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.new
type AnimatedFractionallySizedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// HeightFactor is the fraction of the available height to give the child.
	HeightFactor foundation.Opt[float64]
	// WidthFactor is the fraction of the available width to give the child.
	WidthFactor foundation.Opt[float64]
}

// AnimatedFractionallySizedBox is the animated version of FractionallySizedBox:
// it animates the factors and the alignment when they change.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox
type AnimatedFractionallySizedBox struct {
	ImplicitlyAnimatedWidget
	child        Widget
	alignment    painting.AlignmentGeometry
	heightFactor foundation.Opt[float64]
	widthFactor  foundation.Opt[float64]
}

// NewAnimatedFractionallySizedBox creates the widget. In debug builds it panics
// with the tag AlignFactorNegative when a factor is negative.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.new
func NewAnimatedFractionallySizedBox(duration time.Duration, child Widget, opts ...AnimatedFractionallySizedBoxOpts) *AnimatedFractionallySizedBox {
	o := firstOpts(opts)
	assertFactors("AnimatedFractionallySizedBox", o.WidthFactor, o.HeightFactor)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	return &AnimatedFractionallySizedBox{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		child, o.Alignment, o.HeightFactor, o.WidthFactor}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.child
func (w *AnimatedFractionallySizedBox) Child() Widget { return w.child }

// Alignment is how to align the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.alignment
func (w *AnimatedFractionallySizedBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// HeightFactor is the fraction of the available height to give the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.heightFactor
func (w *AnimatedFractionallySizedBox) HeightFactor() foundation.Opt[float64] { return w.heightFactor }

// WidthFactor is the fraction of the available width to give the child.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.widthFactor
func (w *AnimatedFractionallySizedBox) WidthFactor() foundation.Opt[float64] { return w.widthFactor }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart AnimatedFractionallySizedBox.createState
func (w *AnimatedFractionallySizedBox) CreateState() State {
	return &animatedFractionallySizedBoxState{}
}

type animatedFractionallySizedBoxState struct {
	AnimatedWidgetBaseState[*AnimatedFractionallySizedBox]
	alignment    *animation.Tween[painting.AlignmentGeometry]
	heightFactor *animation.Tween[float64]
	widthFactor  *animation.Tween[float64]
}

func (s *animatedFractionallySizedBoxState) ForEachTween(v *TweenVisitor) {
	w := s.Widget()
	VisitTween(v, &s.alignment, optNonNil(w.alignment), alignmentGeometryEqual,
		func(x painting.AlignmentGeometry) *animation.Tween[painting.AlignmentGeometry] {
			return rendering.NewAlignmentGeometryTween(x, x)
		})
	if w.heightFactor.IsSome() {
		VisitTween(v, &s.heightFactor, w.heightFactor, floatEqual, floatTween)
	}
	if w.widthFactor.IsSome() {
		VisitTween(v, &s.widthFactor, w.widthFactor, floatEqual, floatTween)
	}
}

func (s *animatedFractionallySizedBoxState) Build(BuildContext) Widget {
	a := s.Animation()
	return NewFractionallySizedBox(s.Widget().child, FractionallySizedBoxOpts{Alignment: s.alignment.Evaluate(a),
		HeightFactor: TweenOf(s.heightFactor, a), WidthFactor: TweenOf(s.widthFactor, a)})
}
