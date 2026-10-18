package ui

import (
	"math"
	"strconv"
	"strings"
)

// ClampDouble clamps x to the range [min, max]. A NaN x gives max, as in
// Flutter.
//
// Flutter: ui/math.dart clampDouble
func ClampDouble(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	if math.IsNaN(x) {
		return max
	}
	return x
}

// LerpDouble linearly interpolates between a and b: a when t is 0, b when t is
// 1. Equal values, and two NaNs, give a exactly, whatever t is.
//
// Flutter: ui/lerp.dart lerpDouble
func LerpDouble(a, b, t float64) float64 {
	if a == b || (math.IsNaN(a) && math.IsNaN(b)) {
		return a
	}
	return a*(1.0-t) + b*t
}

// LerpDoubleOpt is LerpDouble for values that may be absent: a nil value counts
// as 0, and nil with nil gives nil.
//
// Flutter: ui/lerp.dart lerpDouble
func LerpDoubleOpt(a, b *float64, t float64) *float64 {
	if a == nil && b == nil {
		return nil
	}
	var av, bv float64
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	r := LerpDouble(av, bv, t)
	return &r
}

// lerpDouble is the interpolation the geometry classes use: unlike LerpDouble it
// keeps infinities and NaNs flowing through the arithmetic.
func lerpDouble(a, b, t float64) float64 {
	return a*(1.0-t) + b*t
}

// dartMod is Dart's % on doubles: the remainder, made non-negative by adding the
// magnitude of the divisor.
func dartMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		r += math.Abs(b)
	}
	return r
}

// toStringAsFixed1 formats x as Dart's toStringAsFixed(1) does: one decimal,
// rounding exact ties away from zero (Go's formatting rounds them to even).
func toStringAsFixed1(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}
	s := strconv.FormatFloat(x, 'f', 1, 64)
	exact := strconv.FormatFloat(math.Abs(x), 'f', 80, 64)
	dot := strings.IndexByte(exact, '.')
	if dot >= 0 && exact[dot+2] == '5' && strings.Trim(exact[dot+3:], "0") == "" {
		// An exact tie, such as 0.25: round away from zero.
		r := strconv.FormatFloat(math.Abs(x)+0.05, 'f', 1, 64)
		if math.Signbit(x) {
			r = "-" + r
		}
		return r
	}
	return s
}
