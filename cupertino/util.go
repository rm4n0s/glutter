package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/painting"
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
