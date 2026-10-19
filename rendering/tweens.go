package rendering

import (
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
)

// NewFractionalOffsetTween creates a tween between two fractional offsets. A
// value that is none counts as the center when the other is some, as
// FractionalOffset.lerp does with null.
//
// Flutter: rendering/tweens.dart FractionalOffsetTween
func NewFractionalOffsetTween(begin, end foundation.Opt[painting.FractionalOffset]) *animation.Tween[foundation.Opt[painting.FractionalOffset]] {
	opt := func(f foundation.Opt[painting.FractionalOffset]) *painting.FractionalOffset {
		if v, ok := f.Get(); ok {
			return &v
		}
		return nil
	}
	return animation.NewTweenFunc(begin, end, func(b, e foundation.Opt[painting.FractionalOffset], t float64) foundation.Opt[painting.FractionalOffset] {
		if r := painting.FractionalOffsetLerp(opt(b), opt(e), t); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[painting.FractionalOffset]()
	}).SetName("FractionalOffsetTween").SetNullable(func(o foundation.Opt[painting.FractionalOffset]) bool { return o.IsNone() })
}

// NewAlignmentTween creates a tween between two alignments.
//
// Flutter: rendering/tweens.dart AlignmentTween
func NewAlignmentTween(begin, end painting.Alignment) *animation.Tween[painting.Alignment] {
	return animation.NewTweenFunc(begin, end, func(b, e painting.Alignment, t float64) painting.Alignment {
		return *painting.AlignmentLerp(&b, &e, t)
	}).SetName("AlignmentTween")
}

// NewAlignmentGeometryTween creates a tween between two alignment geometries,
// either of which may be nil (which counts as the center).
//
// Flutter: rendering/tweens.dart AlignmentGeometryTween
func NewAlignmentGeometryTween(begin, end painting.AlignmentGeometry) *animation.Tween[painting.AlignmentGeometry] {
	return animation.NewTweenFunc(begin, end, painting.AlignmentGeometryLerp).SetName("AlignmentGeometryTween")
}
