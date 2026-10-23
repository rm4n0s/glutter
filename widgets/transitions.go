package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// AnimatedWidgetInterface is what an animated widget implements: the embedded
// AnimatedWidget (which gives Listenable and CreateState) and Build. It is the
// type of the widget that AnimatedWidget's state builds.
//
// Flutter: widgets/transitions.dart AnimatedWidget
type AnimatedWidgetInterface interface {
	Widget
	// Listenable is the listenable that the widget rebuilds with.
	Listenable() foundation.Listenable
	// Build describes the part of the user interface represented by the widget.
	Build(bc BuildContext) Widget
}

// AnimatedWidget is the embedded base of a widget that rebuilds when a
// Listenable (usually an Animation) notifies. The widget type embeds it and
// implements Build; the base listens, and calls SetState for it.
//
//	type Spin struct {
//		widgets.AnimatedWidget
//	}
//
//	func NewSpin(turns animation.Animation[float64]) *Spin {
//		return &Spin{widgets.NewAnimatedWidgetBase(turns)}
//	}
//
//	func (w *Spin) Build(bc widgets.BuildContext) widgets.Widget { /* uses turns.Value() */ }
//
// Flutter: widgets/transitions.dart AnimatedWidget
type AnimatedWidget struct {
	StatefulWidget
	listenable foundation.Listenable
}

// NewAnimatedWidgetBase creates the embedded base of an animated widget.
//
// Flutter: widgets/transitions.dart AnimatedWidget.new
func NewAnimatedWidgetBase(listenable foundation.Listenable, opts ...WidgetOpts) AnimatedWidget {
	return AnimatedWidget{NewStatefulWidgetBase(firstOpts(opts)), listenable}
}

// Listenable is the listenable that the widget rebuilds with.
//
// Flutter: widgets/transitions.dart AnimatedWidget.listenable
func (w *AnimatedWidget) Listenable() foundation.Listenable { return w.listenable }

// CreateState creates the state.
//
// Flutter: widgets/transitions.dart AnimatedWidget.createState
func (w *AnimatedWidget) CreateState() State { return &animatedWidgetState{} }

type animatedWidgetState struct {
	StateOf[AnimatedWidgetInterface]
	handle foundation.ListenerHandle
}

func (s *animatedWidgetState) InitState() {
	s.handle = s.Widget().Listenable().AddListener(s.handleChange)
}

func (s *animatedWidgetState) DidUpdateWidget(old AnimatedWidgetInterface) {
	if old.Listenable() != s.Widget().Listenable() {
		old.Listenable().RemoveListener(s.handle)
		s.handle = s.Widget().Listenable().AddListener(s.handleChange)
	}
}

func (s *animatedWidgetState) Dispose() { s.Widget().Listenable().RemoveListener(s.handle) }

func (s *animatedWidgetState) handleChange() {
	if !s.Mounted() {
		return
	}
	// The animation's state is our build state, and it changed already.
	s.SetState(func() {})
}

func (s *animatedWidgetState) Build(bc BuildContext) Widget { return s.Widget().Build(bc) }

// SlideTransitionOpts are the optional parameters of NewSlideTransition.
//
// Flutter: widgets/transitions.dart SlideTransition.new
type SlideTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TransformHitTests is whether the hit tests are transformed too; the
	// default is true.
	TransformHitTests foundation.Opt[bool]
	// TextDirection, when it is right to left, flips the horizontal offset.
	TextDirection ui.TextDirection
}

// SlideTransition animates the position of a widget relative to its normal
// position: the offset is a fraction of the widget's size.
//
// Flutter: widgets/transitions.dart SlideTransition
type SlideTransition struct {
	AnimatedWidget
	position          animation.Animation[ui.Offset]
	transformHitTests bool
	textDirection     ui.TextDirection
	child             Widget
}

// NewSlideTransition creates the widget.
//
// Flutter: widgets/transitions.dart SlideTransition.new
func NewSlideTransition(position animation.Animation[ui.Offset], child Widget, opts ...SlideTransitionOpts) *SlideTransition {
	o := firstOpts(opts)
	return &SlideTransition{AnimatedWidget: NewAnimatedWidgetBase(position, WidgetOpts{Key: o.Key}), position: position,
		transformHitTests: o.TransformHitTests.OrElse(true), textDirection: o.TextDirection, child: child}
}

// Position is the animation of the offset, as a fraction of the size.
//
// Flutter: widgets/transitions.dart SlideTransition.position
func (w *SlideTransition) Position() animation.Animation[ui.Offset] { return w.position }

// TextDirection is the direction that flips the horizontal offset when it is
// right to left.
//
// Flutter: widgets/transitions.dart SlideTransition.textDirection
func (w *SlideTransition) TextDirection() ui.TextDirection { return w.textDirection }

// TransformHitTests is whether the hit tests are transformed.
//
// Flutter: widgets/transitions.dart SlideTransition.transformHitTests
func (w *SlideTransition) TransformHitTests() bool { return w.transformHitTests }

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart SlideTransition.child
func (w *SlideTransition) Child() Widget { return w.child }

// Build translates the child by the current offset.
//
// Flutter: widgets/transitions.dart SlideTransition.build
func (w *SlideTransition) Build(BuildContext) Widget {
	offset := w.position.Value()
	if w.textDirection == ui.TextDirectionRTL {
		offset = ui.Offset{Dx: -offset.Dx, Dy: offset.Dy}
	}
	return NewFractionalTranslation(offset, w.child, FractionalTranslationOpts{TransformHitTests: foundation.Some(w.transformHitTests)})
}

// TransformCallback computes a matrix from the animation's value.
//
// Flutter: widgets/transitions.dart TransformCallback
type TransformCallback func(animationValue float64) *ui.Matrix4

// MatrixTransitionOpts are the optional parameters of NewMatrixTransition.
//
// Flutter: widgets/transitions.dart MatrixTransition.new
type MatrixTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is the origin of the transform, relative to the size of the
	// child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// FilterQuality is the filter quality of the child while the animation runs.
	FilterQuality foundation.Opt[ui.FilterQuality]
}

// MatrixTransition animates the matrix of a Transform with a callback.
//
// Flutter: widgets/transitions.dart MatrixTransition
type MatrixTransition struct {
	AnimatedWidget
	animation     animation.Animation[float64]
	onTransform   TransformCallback
	alignment     painting.AlignmentGeometry
	filterQuality foundation.Opt[ui.FilterQuality]
	child         Widget
}

// NewMatrixTransition creates the widget.
//
// Flutter: widgets/transitions.dart MatrixTransition.new
func NewMatrixTransition(animation animation.Animation[float64], onTransform TransformCallback, child Widget, opts ...MatrixTransitionOpts) *MatrixTransition {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	return &MatrixTransition{AnimatedWidget: NewAnimatedWidgetBase(animation, WidgetOpts{Key: o.Key}), animation: animation,
		onTransform: onTransform, alignment: o.Alignment, filterQuality: o.FilterQuality, child: child}
}

// Animation is the animation that drives the transform.
//
// Flutter: widgets/transitions.dart MatrixTransition.animation
func (w *MatrixTransition) Animation() animation.Animation[float64] { return w.animation }

// OnTransform is the callback that computes the matrix.
//
// Flutter: widgets/transitions.dart MatrixTransition.onTransform
func (w *MatrixTransition) OnTransform() TransformCallback { return w.onTransform }

// Alignment is the origin of the transform.
//
// Flutter: widgets/transitions.dart MatrixTransition.alignment
func (w *MatrixTransition) Alignment() painting.AlignmentGeometry { return w.alignment }

// FilterQuality is the filter quality of the child while the animation runs.
//
// Flutter: widgets/transitions.dart MatrixTransition.filterQuality
func (w *MatrixTransition) FilterQuality() foundation.Opt[ui.FilterQuality] { return w.filterQuality }

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart MatrixTransition.child
func (w *MatrixTransition) Child() Widget { return w.child }

// Build transforms the child by the matrix of the current value.
//
// Flutter: widgets/transitions.dart MatrixTransition.build
func (w *MatrixTransition) Build(BuildContext) Widget {
	var quality foundation.Opt[ui.FilterQuality]
	if animation.IsAnimating(w.animation) {
		quality = w.filterQuality
	}
	return NewTransform(w.onTransform(w.animation.Value()), w.child, TransformOpts{Alignment: w.alignment, FilterQuality: quality})
}

// ScaleTransitionOpts are the optional parameters of NewScaleTransition.
//
// Flutter: widgets/transitions.dart ScaleTransition.new
type ScaleTransitionOpts = MatrixTransitionOpts

// ScaleTransition animates the scale of a transformed widget.
//
// Flutter: widgets/transitions.dart ScaleTransition
type ScaleTransition struct{ MatrixTransition }

// NewScaleTransition creates the widget; the scale is applied to x and y.
//
// Flutter: widgets/transitions.dart ScaleTransition.new
func NewScaleTransition(scale animation.Animation[float64], child Widget, opts ...ScaleTransitionOpts) *ScaleTransition {
	return &ScaleTransition{*NewMatrixTransition(scale, func(v float64) *ui.Matrix4 {
		return ui.NewMatrix4Diagonal3Values(float32(v), float32(v), 1)
	}, child, firstOpts(opts))}
}

// Scale is the animation of the scale.
//
// Flutter: widgets/transitions.dart ScaleTransition.scale
func (w *ScaleTransition) Scale() animation.Animation[float64] { return w.animation }

// RotationTransitionOpts are the optional parameters of NewRotationTransition.
//
// Flutter: widgets/transitions.dart RotationTransition.new
type RotationTransitionOpts = MatrixTransitionOpts

// RotationTransition animates the rotation of a widget, in turns.
//
// Flutter: widgets/transitions.dart RotationTransition
type RotationTransition struct{ MatrixTransition }

// NewRotationTransition creates the widget; one turn is a full rotation.
//
// Flutter: widgets/transitions.dart RotationTransition.new
func NewRotationTransition(turns animation.Animation[float64], child Widget, opts ...RotationTransitionOpts) *RotationTransition {
	return &RotationTransition{*NewMatrixTransition(turns, func(v float64) *ui.Matrix4 {
		return ui.NewMatrix4RotationZ(v * math.Pi * 2)
	}, child, firstOpts(opts))}
}

// Turns is the animation of the rotation, in turns.
//
// Flutter: widgets/transitions.dart RotationTransition.turns
func (w *RotationTransition) Turns() animation.Animation[float64] { return w.animation }

// FadeTransitionOpts are the optional parameters of NewFadeTransition.
//
// Flutter: widgets/transitions.dart FadeTransition.new
type FadeTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AlwaysIncludeSemantics is whether the semantics of the child are included
	// even when it is fully transparent.
	AlwaysIncludeSemantics bool
}

// FadeTransition animates the opacity of a widget; the child is not rebuilt or
// repainted while the opacity changes.
//
// Flutter: widgets/transitions.dart FadeTransition
type FadeTransition struct {
	SingleChildRenderObjectWidget
	opacity                animation.Animation[float64]
	alwaysIncludeSemantics bool
}

// NewFadeTransition creates the widget.
//
// Flutter: widgets/transitions.dart FadeTransition.new
func NewFadeTransition(opacity animation.Animation[float64], child Widget, opts ...FadeTransitionOpts) *FadeTransition {
	o := firstOpts(opts)
	return &FadeTransition{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), opacity, o.AlwaysIncludeSemantics}
}

// Opacity is the animation of the opacity.
//
// Flutter: widgets/transitions.dart FadeTransition.opacity
func (w *FadeTransition) Opacity() animation.Animation[float64] { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics of the child are included
// even when it is fully transparent.
//
// Flutter: widgets/transitions.dart FadeTransition.alwaysIncludeSemantics
func (w *FadeTransition) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/transitions.dart FadeTransition.createRenderObject
func (w *FadeTransition) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderAnimatedOpacity(w.opacity, nil, rendering.RenderAnimatedOpacityOpts{AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/transitions.dart FadeTransition.updateRenderObject
func (w *FadeTransition) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderAnimatedOpacity)
	r.SetOpacity(w.opacity)
	r.SetAlwaysIncludeSemantics(w.alwaysIncludeSemantics)
}

// DelegatedTransitionBuilder builds the transition that a route delegates to the
// route below it: it returns nil when there is none.
//
// Flutter: widgets/transitions.dart DelegatedTransitionBuilder
type DelegatedTransitionBuilder func(bc BuildContext, animation, secondaryAnimation animation.Animation[float64], allowSnapshotting bool, child Widget) Widget

// SizeTransitionOpts are the optional parameters of NewSizeTransition.
//
// Flutter: widgets/transitions.dart SizeTransition.new
type SizeTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Axis is the axis the size animates on; the default is AxisVertical.
	Axis painting.Axis
	// AxisAlignment is the alignment on the axis, from -1 (start) to 1 (end).
	//
	// Deprecated: use Alignment, which controls both axes.
	AxisAlignment foundation.Opt[float64]
	// Alignment is how the child is aligned while the size animates; it
	// overrides AxisAlignment.
	Alignment painting.AlignmentGeometry
	// FixedCrossAxisSizeFactor is the size factor of the axis the animation
	// does not run on; without it the child decides that size.
	FixedCrossAxisSizeFactor foundation.Opt[float64]
}

// SizeTransition animates its own size and clips and aligns its child.
//
// Flutter: widgets/transitions.dart SizeTransition
type SizeTransition struct {
	AnimatedWidget
	axis                     painting.Axis
	axisAlignment            foundation.Opt[float64]
	alignment                painting.AlignmentGeometry
	fixedCrossAxisSizeFactor foundation.Opt[float64]
	child                    Widget
}

// NewSizeTransition creates the widget. In debug builds it panics with the tag
// SizeTransitionInvalid when the fixed factor is negative, or when both
// AxisAlignment and Alignment are given.
//
// Flutter: widgets/transitions.dart SizeTransition.new
func NewSizeTransition(sizeFactor animation.Animation[float64], child Widget, opts ...SizeTransitionOpts) *SizeTransition {
	o := firstOpts(opts)
	if o.Axis == painting.AxisUnspecified {
		o.Axis = painting.AxisVertical
	}
	if foundation.DebugMode {
		if v, ok := o.FixedCrossAxisSizeFactor.Get(); ok && !(v >= 0) {
			panic(errors.New(tagSizeTransitionInvalid, "the fixedCrossAxisSizeFactor of a SizeTransition must not be negative", "fixedCrossAxisSizeFactor", v))
		}
		if o.AxisAlignment.IsSome() && o.Alignment != nil {
			panic(errors.New(tagSizeTransitionInvalid, "cannot provide both axisAlignment and alignment as axisAlignment has been deprecated and superseded by alignment"))
		}
	}
	return &SizeTransition{AnimatedWidget: NewAnimatedWidgetBase(sizeFactor, WidgetOpts{Key: o.Key}), axis: o.Axis,
		axisAlignment: o.AxisAlignment, alignment: o.Alignment, fixedCrossAxisSizeFactor: o.FixedCrossAxisSizeFactor, child: child}
}

// Axis is the axis the size animates on.
//
// Flutter: widgets/transitions.dart SizeTransition.axis
func (w *SizeTransition) Axis() painting.Axis { return w.axis }

// SizeFactor is the animation of the fraction of the child's size.
//
// Flutter: widgets/transitions.dart SizeTransition.sizeFactor
func (w *SizeTransition) SizeFactor() animation.Animation[float64] {
	return w.listenable.(animation.Animation[float64])
}

// Alignment is how the child is aligned while the size animates, or nil.
//
// Flutter: widgets/transitions.dart SizeTransition.alignment
func (w *SizeTransition) Alignment() painting.AlignmentGeometry { return w.alignment }

// FixedCrossAxisSizeFactor is the size factor of the other axis.
//
// Flutter: widgets/transitions.dart SizeTransition.fixedCrossAxisSizeFactor
func (w *SizeTransition) FixedCrossAxisSizeFactor() foundation.Opt[float64] {
	return w.fixedCrossAxisSizeFactor
}

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart SizeTransition.child
func (w *SizeTransition) Child() Widget { return w.child }

// Build clips and aligns the child by the current size factor.
//
// Flutter: widgets/transitions.dart SizeTransition.build
func (w *SizeTransition) Build(BuildContext) Widget {
	alignment := w.alignment
	if alignment == nil {
		if w.axis == painting.AxisHorizontal {
			alignment = painting.AlignmentDirectional{Start: w.axisAlignment.OrElse(0), Y: -1}
		} else {
			alignment = painting.AlignmentDirectional{Start: -1, Y: w.axisAlignment.OrElse(0)}
		}
	}
	factor := foundation.Some(math.Max(w.SizeFactor().Value(), 0))
	var height, width foundation.Opt[float64]
	if w.axis == painting.AxisVertical {
		height, width = factor, w.fixedCrossAxisSizeFactor
	} else {
		height, width = w.fixedCrossAxisSizeFactor, factor
	}
	return NewClipRect(NewAlign(w.child, AlignOpts{Alignment: alignment, HeightFactor: height, WidthFactor: width}))
}

// NewRelativeRectTween creates a tween between two relative rectangles.
//
// Flutter: widgets/transitions.dart RelativeRectTween
func NewRelativeRectTween(begin, end rendering.RelativeRect) *animation.Tween[rendering.RelativeRect] {
	return animation.NewTweenFunc(begin, end, func(b, e rendering.RelativeRect, t float64) rendering.RelativeRect {
		return *rendering.RelativeRectLerp(&b, &e, t)
	}).SetName("RelativeRectTween")
}

// PositionedTransition animates the position of a widget relative to a Stack
// that is its parent.
//
// Flutter: widgets/transitions.dart PositionedTransition
type PositionedTransition struct {
	AnimatedWidget
	child Widget
}

// NewPositionedTransition creates the widget.
//
// Flutter: widgets/transitions.dart PositionedTransition.new
func NewPositionedTransition(rect animation.Animation[rendering.RelativeRect], child Widget, opts ...WidgetOpts) *PositionedTransition {
	return &PositionedTransition{NewAnimatedWidgetBase(rect, firstOpts(opts)), child}
}

// Rect is the animation of the rectangle.
//
// Flutter: widgets/transitions.dart PositionedTransition.rect
func (w *PositionedTransition) Rect() animation.Animation[rendering.RelativeRect] {
	return w.listenable.(animation.Animation[rendering.RelativeRect])
}

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart PositionedTransition.child
func (w *PositionedTransition) Child() Widget { return w.child }

// Build positions the child by the current rectangle.
//
// Flutter: widgets/transitions.dart PositionedTransition.build
func (w *PositionedTransition) Build(BuildContext) Widget {
	return NewPositionedFromRelativeRect(w.Rect().Value(), w.child)
}

// RelativePositionedTransition animates the position of a widget relative to
// a rectangle of the given size, in a Stack.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition
type RelativePositionedTransition struct {
	AnimatedWidget
	size  ui.Size
	child Widget
}

// NewRelativePositionedTransition creates the widget; the rectangle is
// relative to the given size, and none counts as the zero rectangle.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition.new
func NewRelativePositionedTransition(rect animation.Animation[foundation.Opt[ui.Rect]], size ui.Size, child Widget, opts ...WidgetOpts) *RelativePositionedTransition {
	return &RelativePositionedTransition{NewAnimatedWidgetBase(rect, firstOpts(opts)), size, child}
}

// Rect is the animation of the rectangle.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition.rect
func (w *RelativePositionedTransition) Rect() animation.Animation[foundation.Opt[ui.Rect]] {
	return w.listenable.(animation.Animation[foundation.Opt[ui.Rect]])
}

// Size is the size the rectangle is relative to.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition.size
func (w *RelativePositionedTransition) Size() ui.Size { return w.size }

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition.child
func (w *RelativePositionedTransition) Child() Widget { return w.child }

// Build positions the child by the current rectangle.
//
// Flutter: widgets/transitions.dart RelativePositionedTransition.build
func (w *RelativePositionedTransition) Build(BuildContext) Widget {
	offsets := rendering.NewRelativeRectFromSize(w.Rect().Value().OrElse(ui.Rect{}), w.size)
	return NewPositioned(w.child, PositionedOpts{Top: foundation.Some(offsets.Top), Right: foundation.Some(offsets.Right),
		Bottom: foundation.Some(offsets.Bottom), Left: foundation.Some(offsets.Left)})
}

// DecoratedBoxTransitionOpts are the optional parameters of
// NewDecoratedBoxTransition.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.new
type DecoratedBoxTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Position is where the decoration is painted relative to the child; the
	// default is background.
	Position rendering.DecorationPosition
}

// DecoratedBoxTransition animates the decoration of a DecoratedBox.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition
type DecoratedBoxTransition struct {
	AnimatedWidget
	position rendering.DecorationPosition
	child    Widget
}

// NewDecoratedBoxTransition creates the widget.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.new
func NewDecoratedBoxTransition(decoration animation.Animation[painting.Decoration], child Widget, opts ...DecoratedBoxTransitionOpts) *DecoratedBoxTransition {
	o := firstOpts(opts)
	if o.Position == rendering.DecorationPositionUnspecified {
		o.Position = rendering.DecorationPositionBackground
	}
	return &DecoratedBoxTransition{NewAnimatedWidgetBase(decoration, WidgetOpts{Key: o.Key}), o.Position, child}
}

// Decoration is the animation of the decoration.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.decoration
func (w *DecoratedBoxTransition) Decoration() animation.Animation[painting.Decoration] {
	return w.listenable.(animation.Animation[painting.Decoration])
}

// Position is where the decoration is painted.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.position
func (w *DecoratedBoxTransition) Position() rendering.DecorationPosition { return w.position }

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.child
func (w *DecoratedBoxTransition) Child() Widget { return w.child }

// Build decorates the child with the current decoration.
//
// Flutter: widgets/transitions.dart DecoratedBoxTransition.build
func (w *DecoratedBoxTransition) Build(BuildContext) Widget {
	return NewDecoratedBox(w.Decoration().Value(), w.child, DecoratedBoxOpts{Position: w.position})
}

// AlignTransitionOpts are the optional parameters of NewAlignTransition.
//
// Flutter: widgets/transitions.dart AlignTransition.new
type AlignTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// WidthFactor, if given, sizes the width as a multiple of the child's.
	WidthFactor foundation.Opt[float64]
	// HeightFactor, if given, sizes the height as a multiple of the child's.
	HeightFactor foundation.Opt[float64]
}

// AlignTransition animates the alignment of a child.
//
// Flutter: widgets/transitions.dart AlignTransition
type AlignTransition struct {
	AnimatedWidget
	widthFactor  foundation.Opt[float64]
	heightFactor foundation.Opt[float64]
	child        Widget
}

// NewAlignTransition creates the widget.
//
// Flutter: widgets/transitions.dart AlignTransition.new
func NewAlignTransition(alignment animation.Animation[painting.AlignmentGeometry], child Widget, opts ...AlignTransitionOpts) *AlignTransition {
	o := firstOpts(opts)
	return &AlignTransition{NewAnimatedWidgetBase(alignment, WidgetOpts{Key: o.Key}), o.WidthFactor, o.HeightFactor, child}
}

// Alignment is the animation of the alignment.
//
// Flutter: widgets/transitions.dart AlignTransition.alignment
func (w *AlignTransition) Alignment() animation.Animation[painting.AlignmentGeometry] {
	return w.listenable.(animation.Animation[painting.AlignmentGeometry])
}

// WidthFactor is the factor of the child's width.
//
// Flutter: widgets/transitions.dart AlignTransition.widthFactor
func (w *AlignTransition) WidthFactor() foundation.Opt[float64] { return w.widthFactor }

// HeightFactor is the factor of the child's height.
//
// Flutter: widgets/transitions.dart AlignTransition.heightFactor
func (w *AlignTransition) HeightFactor() foundation.Opt[float64] { return w.heightFactor }

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart AlignTransition.child
func (w *AlignTransition) Child() Widget { return w.child }

// Build aligns the child by the current alignment.
//
// Flutter: widgets/transitions.dart AlignTransition.build
func (w *AlignTransition) Build(BuildContext) Widget {
	return NewAlign(w.child, AlignOpts{Alignment: w.Alignment().Value(), WidthFactor: w.widthFactor, HeightFactor: w.heightFactor})
}

// DefaultTextStyleTransitionOpts are the optional parameters of
// NewDefaultTextStyleTransition.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition.new
type DefaultTextStyleTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextAlign is how the text is aligned horizontally.
	TextAlign ui.TextAlign
	// SoftWrap is whether the text breaks at soft line breaks; the default is true.
	SoftWrap foundation.Opt[bool]
	// Overflow is how visual overflow is handled; the default is clip.
	Overflow painting.TextOverflow
	// MaxLines is the maximum number of lines.
	MaxLines foundation.Opt[int]
}

// DefaultTextStyleTransition animates the style of a DefaultTextStyle.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition
type DefaultTextStyleTransition struct {
	AnimatedWidget
	textAlign ui.TextAlign
	softWrap  bool
	overflow  painting.TextOverflow
	maxLines  foundation.Opt[int]
	child     Widget
}

// NewDefaultTextStyleTransition creates the widget.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition.new
func NewDefaultTextStyleTransition(style animation.Animation[*painting.TextStyle], child Widget, opts ...DefaultTextStyleTransitionOpts) *DefaultTextStyleTransition {
	o := firstOpts(opts)
	if o.Overflow == painting.TextOverflowUnspecified {
		o.Overflow = painting.TextOverflowClip
	}
	return &DefaultTextStyleTransition{NewAnimatedWidgetBase(style, WidgetOpts{Key: o.Key}), o.TextAlign, o.SoftWrap.OrElse(true), o.Overflow, o.MaxLines, child}
}

// Style is the animation of the style.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition.style
func (w *DefaultTextStyleTransition) Style() animation.Animation[*painting.TextStyle] {
	return w.listenable.(animation.Animation[*painting.TextStyle])
}

// Child is the widget below this one.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition.child
func (w *DefaultTextStyleTransition) Child() Widget { return w.child }

// Build sets the default text style of the child to the current style.
//
// Flutter: widgets/transitions.dart DefaultTextStyleTransition.build
func (w *DefaultTextStyleTransition) Build(BuildContext) Widget {
	return NewDefaultTextStyle(w.Style().Value(), w.child, DefaultTextStyleOpts{TextAlign: w.textAlign, SoftWrap: foundation.Some(w.softWrap),
		Overflow: w.overflow, MaxLines: w.maxLines})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSizeTransitionInvalid = "SizeTransitionInvalid"
)
