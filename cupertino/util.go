package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// firstOf is the first of the options values that a constructor takes as a variadic parameter, or the zero value.
func firstOf[T any](opts []T) T {
	var zero T
	if len(opts) > 0 {
		return opts[0]
	}
	return zero
}

// infinity is positive infinity, for the asserts of the finite thickness.
var infinity = math.Inf(1)

// absFloat is the absolute value of a number.
func absFloat(v float64) float64 { return math.Abs(v) }

// axisHorizontal is painting.AxisHorizontal.
const axisHorizontal = painting.AxisHorizontal

// millisecond is a time.Millisecond, for the durations of the animations.
const millisecond = time.Millisecond

// lerpColor is the color between a and b at t.
func lerpColor(a, b ui.Color, t float64) ui.Color { return *ui.ColorLerp(&a, &b, t) }

// lerpOptDouble is the number between a and b at t when both are set, and none otherwise (Flutter's lerpDouble
// with a null end gives null).
func lerpOptDouble(a, b foundation.Opt[float64], t float64) foundation.Opt[float64] {
	av, aok := a.Get()
	bv, bok := b.Get()
	if !aok || !bok {
		return foundation.None[float64]()
	}
	return foundation.Some(ui.LerpDouble(av, bv, t))
}

// rectFromOffsetAndSize is Dart's offset & size: the rectangle with the offset as its top left corner.
func rectFromOffsetAndSize(o ui.Offset, s ui.Size) ui.Rect {
	return ui.NewRectFromLTWH(o.Dx, o.Dy, s.Width, s.Height)
}

// ptrColor is the address of a copy of the color, for the options that take a *ui.Color.
func ptrColor(c ui.Color) *ui.Color { return &c }
