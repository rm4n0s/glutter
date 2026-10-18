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

// toStringAsFixed1 formats x as Dart's toStringAsFixed(1) does.
func toStringAsFixed1(x float64) string { return toStringAsFixed(x, 1) }

// toStringAsFixed formats x with the given number of decimals as Dart's
// toStringAsFixed does: exact ties are rounded away from zero, where Go's
// formatting rounds them to even.
func toStringAsFixed(x float64, digits int) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	}
	s := strconv.FormatFloat(x, 'f', digits, 64)
	exact := strconv.FormatFloat(math.Abs(x), 'f', 80, 64)
	dot := strings.IndexByte(exact, '.')
	if dot < 0 || exact[dot+1+digits] != '5' || strings.Trim(exact[dot+2+digits:], "0") != "" {
		return s
	}
	// An exact tie, such as 0.25 with one decimal: round away from zero by
	// adding one unit in the last place to the truncated digits.
	r := incrementDecimal(exact[:dot+1+digits])
	r = strings.TrimSuffix(r, ".")
	if math.Signbit(x) {
		r = "-" + r
	}
	return r
}

// incrementDecimal adds one to the last digit of a decimal string such as
// "12.3", carrying as needed.
func incrementDecimal(s string) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case '.':
			continue
		case '9':
			b[i] = '0'
		default:
			b[i]++
			return string(b)
		}
	}
	return "1" + string(b)
}

// dartDouble formats x as Dart's double.toString does: "1.0" for an integral
// value, the shortest digits that read back as x otherwise, and an exponent
// ("1e-5", "1e+21") below 1e-4 and from 1e21.
func dartDouble(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x == 0:
		if math.Signbit(x) {
			return "-0.0"
		}
		return "0.0"
	}
	if a := math.Abs(x); a < 1e-4 || a >= 1e21 {
		s := strconv.FormatFloat(x, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		sign := exp[0]
		exp = strings.TrimLeft(exp[1:], "0")
		if sign == '-' {
			exp = "-" + exp
		} else {
			exp = "+" + exp
		}
		return mant + "e" + exp
	}
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func boolString(b bool) string { return strconv.FormatBool(b) }
func itoa(i int) string        { return strconv.Itoa(i) }
