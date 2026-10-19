package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// NewBoxConstraintsTween creates a tween between two box constraints.
//
// Flutter: widgets/implicit_animations.dart BoxConstraintsTween
func NewBoxConstraintsTween(begin, end rendering.BoxConstraints) *animation.Tween[rendering.BoxConstraints] {
	return animation.NewTweenFunc(begin, end, func(b, e rendering.BoxConstraints, t float64) rendering.BoxConstraints {
		return *rendering.BoxConstraintsLerp(&b, &e, t)
	}).SetName("BoxConstraintsTween")
}

// NewDecorationTween creates a tween between two decorations (either may be
// nil).
//
// Flutter: widgets/implicit_animations.dart DecorationTween
func NewDecorationTween(begin, end painting.Decoration) *animation.Tween[painting.Decoration] {
	return animation.NewTweenFunc(begin, end, painting.DecorationLerp).SetName("DecorationTween")
}

// NewEdgeInsetsTween creates a tween between two EdgeInsets.
//
// Flutter: widgets/implicit_animations.dart EdgeInsetsTween
func NewEdgeInsetsTween(begin, end painting.EdgeInsets) *animation.Tween[painting.EdgeInsets] {
	return animation.NewTweenFunc(begin, end, func(b, e painting.EdgeInsets, t float64) painting.EdgeInsets {
		return *painting.EdgeInsetsLerp(&b, &e, t)
	}).SetName("EdgeInsetsTween")
}

// NewEdgeInsetsGeometryTween creates a tween between two EdgeInsetsGeometry
// values, which can be of different kinds (either may be nil).
//
// Flutter: widgets/implicit_animations.dart EdgeInsetsGeometryTween
func NewEdgeInsetsGeometryTween(begin, end painting.EdgeInsetsGeometry) *animation.Tween[painting.EdgeInsetsGeometry] {
	return animation.NewTweenFunc(begin, end, painting.EdgeInsetsGeometryLerp).SetName("EdgeInsetsGeometryTween")
}

// NewBorderRadiusTween creates a tween between two border radii. A value that
// is none counts as zero when the other is some, as BorderRadius.lerp does with
// null; between two nones the value stays none.
//
// Flutter: widgets/implicit_animations.dart BorderRadiusTween
func NewBorderRadiusTween(begin, end foundation.Opt[painting.BorderRadius]) *animation.Tween[foundation.Opt[painting.BorderRadius]] {
	ptr := func(o foundation.Opt[painting.BorderRadius]) *painting.BorderRadius {
		if v, ok := o.Get(); ok {
			return &v
		}
		return nil
	}
	return animation.NewTweenFunc(begin, end, func(b, e foundation.Opt[painting.BorderRadius], t float64) foundation.Opt[painting.BorderRadius] {
		if r := painting.BorderRadiusLerp(ptr(b), ptr(e), t); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[painting.BorderRadius]()
	}).SetName("BorderRadiusTween").SetNullable(func(o foundation.Opt[painting.BorderRadius]) bool { return o.IsNone() })
}

// NewBorderTween creates a tween between two borders. A value that is none
// counts as no border when the other is some; between two nones the value stays
// none.
//
// Flutter: widgets/implicit_animations.dart BorderTween
func NewBorderTween(begin, end foundation.Opt[painting.Border]) *animation.Tween[foundation.Opt[painting.Border]] {
	ptr := func(o foundation.Opt[painting.Border]) *painting.Border {
		if v, ok := o.Get(); ok {
			return &v
		}
		return nil
	}
	return animation.NewTweenFunc(begin, end, func(b, e foundation.Opt[painting.Border], t float64) foundation.Opt[painting.Border] {
		if r := painting.BorderLerp(ptr(b), ptr(e), t); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[painting.Border]()
	}).SetName("BorderTween").SetNullable(func(o foundation.Opt[painting.Border]) bool { return o.IsNone() })
}

// NewMatrix4Tween creates a tween between two matrices: it interpolates the
// translation, the rotation (as a quaternion) and the scale they decompose to.
// Both matrices are needed; a nil one panics with the tag Matrix4TweenNull when
// the tween is used in debug builds.
//
// Flutter: widgets/implicit_animations.dart Matrix4Tween
func NewMatrix4Tween(begin, end *ui.Matrix4) *animation.Tween[*ui.Matrix4] {
	return animation.NewTweenFunc(begin, end, func(b, e *ui.Matrix4, t float64) *ui.Matrix4 {
		if b == nil || e == nil {
			panic(errors.New("Matrix4TweenNull", "a Matrix4Tween needs both a begin and an end matrix"))
		}
		bt, br, bs := b.Decompose()
		et, er, es := e.Decompose()
		lerp3 := func(x, y ui.Vector3) ui.Vector3 {
			f := func(a, b float32) float32 { return float32(float64(a)*(1-t) + float64(b)*t) }
			return ui.Vector3{X: f(x.X, y.X), Y: f(x.Y, y.Y), Z: f(x.Z, y.Z)}
		}
		// TODO(alangardner) in Flutter: implement lerp for constant rotation.
		rotation := br.Scaled(1 - t).Add(er.Scaled(t)).Normalized()
		return ui.NewMatrix4Compose(lerp3(bt, et), rotation, lerp3(bs, es))
	}).SetName("Matrix4Tween")
}

// NewTextStyleTween creates a tween between two text styles.
//
// Flutter: widgets/implicit_animations.dart TextStyleTween
func NewTextStyleTween(begin, end *painting.TextStyle) *animation.Tween[*painting.TextStyle] {
	return animation.NewTweenFunc(begin, end, painting.TextStyleLerp).SetName("TextStyleTween")
}
