package animation

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/glutter/foundation"
)

// AnimationStyle is used to override the default parameters of an animation:
// its curves and durations. A field that is not set (nil, or none) means the
// widget's default.
//
// Flutter: animation/animation_style.dart AnimationStyle
type AnimationStyle struct {
	// Curve is the curve of the animation in the forward direction.
	//
	// Flutter: animation/animation_style.dart AnimationStyle.curve
	Curve Curve
	// Duration is the length of the animation in the forward direction.
	//
	// Flutter: animation/animation_style.dart AnimationStyle.duration
	Duration foundation.Opt[time.Duration]
	// ReverseCurve is the curve of the animation in the reverse direction.
	//
	// Flutter: animation/animation_style.dart AnimationStyle.reverseCurve
	ReverseCurve Curve
	// ReverseDuration is the length of the animation in the reverse direction.
	//
	// Flutter: animation/animation_style.dart AnimationStyle.reverseDuration
	ReverseDuration foundation.Opt[time.Duration]
}

// AnimationStyleNoAnimation is an AnimationStyle with the durations set to zero,
// which disables an animation.
//
// Flutter: animation/animation_style.dart AnimationStyle.noAnimation
var AnimationStyleNoAnimation = AnimationStyle{
	Duration:        foundation.Some(time.Duration(0)),
	ReverseDuration: foundation.Some(time.Duration(0)),
}

// NewAnimationStyle creates an AnimationStyle; the fields of the options are the
// ones of the style.
//
// Flutter: animation/animation_style.dart AnimationStyle.new
func NewAnimationStyle(opts ...AnimationStyle) AnimationStyle { return firstOpts(opts) }

// CopyWith is a copy of the style with the fields that are set in the argument
// replaced.
//
// Flutter: animation/animation_style.dart AnimationStyle.copyWith
func (s AnimationStyle) CopyWith(o AnimationStyle) AnimationStyle {
	if o.Curve != nil {
		s.Curve = o.Curve
	}
	if o.Duration.IsSome() {
		s.Duration = o.Duration
	}
	if o.ReverseCurve != nil {
		s.ReverseCurve = o.ReverseCurve
	}
	if o.ReverseDuration.IsSome() {
		s.ReverseDuration = o.ReverseDuration
	}
	return s
}

// Merge is a copy of the style with the fields that are set in the other style
// replaced.
//
// Flutter: animation/animation_style.dart AnimationStyle.merge
func (s AnimationStyle) Merge(other *AnimationStyle) AnimationStyle {
	if other == nil {
		return s
	}
	return s.CopyWith(*other)
}

// curveEqual compares two curves; curves that can't be compared (functions) are
// equal only when they are both nil.
func curveEqual(a, b Curve) (equal bool) {
	defer func() {
		if recover() != nil {
			equal = false
		}
	}()
	return a == b
}

// Equals is whether the two styles have the same curves and durations.
//
// Flutter: animation/animation_style.dart AnimationStyle.==
func (s AnimationStyle) Equals(o AnimationStyle) bool {
	return curveEqual(s.Curve, o.Curve) && s.Duration == o.Duration &&
		curveEqual(s.ReverseCurve, o.ReverseCurve) && s.ReverseDuration == o.ReverseDuration
}

// String describes the style.
//
// Flutter: animation/animation_style.dart AnimationStyle.debugFillProperties
func (s AnimationStyle) String() string {
	var parts []string
	if s.Curve != nil {
		parts = append(parts, fmt.Sprintf("curve: %v", s.Curve))
	}
	if d, ok := s.Duration.Get(); ok {
		parts = append(parts, fmt.Sprintf("duration: %v", d))
	}
	if s.ReverseCurve != nil {
		parts = append(parts, fmt.Sprintf("reverseCurve: %v", s.ReverseCurve))
	}
	if d, ok := s.ReverseDuration.Get(); ok {
		parts = append(parts, fmt.Sprintf("reverseDuration: %v", d))
	}
	out := "AnimationStyle("
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out + ")"
}

// LerpAnimationStyle linearly interpolates between two styles. The curves are
// interpolated by averaging their values, and the durations linearly. A nil
// style has no fields set.
//
// Flutter: animation/animation_style.dart AnimationStyle.lerp
func LerpAnimationStyle(a, b *AnimationStyle, t float64) *AnimationStyle {
	if a == b {
		return a
	}
	var sa, sb AnimationStyle
	if a != nil {
		sa = *a
	}
	if b != nil {
		sb = *b
	}
	return &AnimationStyle{
		Curve:           lerpStyleCurve(sa.Curve, sb.Curve, t),
		Duration:        lerpStyleDuration(sa.Duration, sb.Duration, t),
		ReverseCurve:    lerpStyleCurve(sa.ReverseCurve, sb.ReverseCurve, t),
		ReverseDuration: lerpStyleDuration(sa.ReverseDuration, sb.ReverseDuration, t),
	}
}

func lerpStyleCurve(a, b Curve, t float64) Curve {
	if curveEqual(a, b) || t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	return newLerpedCurve(a, b, t)
}

func lerpStyleDuration(a, b foundation.Opt[time.Duration], t float64) foundation.Opt[time.Duration] {
	if a == b || t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	us := math.Round(float64(a.OrElse(0).Microseconds())*(1-t) + float64(b.OrElse(0).Microseconds())*t)
	return foundation.Some(time.Duration(us) * time.Microsecond)
}

type lerpedCurve struct {
	first, second Curve
	t             float64
}

func newLerpedCurve(a, b Curve, t float64) *lerpedCurve {
	if a == nil {
		a = Curves.Linear
	}
	if b == nil {
		b = Curves.Linear
	}
	return &lerpedCurve{a, b, t}
}

func (c *lerpedCurve) Transform(t float64) float64 {
	return c.first.Transform(t)*(1-c.t) + c.second.Transform(t)*c.t
}

func (c *lerpedCurve) Flipped() Curve { return FlippedCurve{c} }

func (c *lerpedCurve) String() string {
	return fmt.Sprintf("_LerpedCurve(%v, %v, t: %v)", c.first, c.second, c.t)
}
