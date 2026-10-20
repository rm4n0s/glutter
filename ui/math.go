// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"math"
	"strconv"

	"github.com/rm4n0s/glutter/internal/dartfmt"
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

func boolString(b bool) string { return strconv.FormatBool(b) }
func itoa(i int) string        { return strconv.Itoa(i) }

// toStringAsFixed1 formats x as Dart's toStringAsFixed(1) does.
func toStringAsFixed1(x float64) string { return dartfmt.ToStringAsFixed(x, 1) }

// toStringAsFixed formats x with the given number of decimals as Dart's
// toStringAsFixed does.
func toStringAsFixed(x float64, digits int) string { return dartfmt.ToStringAsFixed(x, digits) }

// dartDouble formats x as Dart's double.toString does.
func dartDouble(x float64) string { return dartfmt.Double(x) }
