package widgets

import (
	"math"

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
