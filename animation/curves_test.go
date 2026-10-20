// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/animation/curves_test.dart.

func TestCurvesToString(t *testing.T) {
	for name, c := range map[string]Curve{
		"linear":            Curves.Linear,
		"sawtooth":          NewSawTooth(3),
		"interval":          NewInterval(0.25, 0.75),
		"interval with":     NewInterval(0.25, 0.75, Curves.Ease),
		"split":             NewSplit(0.25, SplitOpts{BeginCurve: Curves.Ease}),
		"flipped ease":      Curves.Ease.Flipped(),
		"elastic in":        Curves.ElasticIn,
		"elastic out":       Curves.ElasticOut,
		"elastic in out":    Curves.ElasticInOut,
		"ease in back":      Curves.EaseInBack,
		"ease out back":     Curves.EaseOutBack,
		"ease in out back":  Curves.EaseInOutBack,
		"decelerate":        Curves.Decelerate,
		"threshold":         NewThreshold(0.5),
		"three point cubic": Curves.FastEaseInToSlowEaseOut,
	} {
		oneLine(t, name, c)
	}
	if got, want := NewInterval(0.25, 0.75, Curves.Ease).String(), "Interval(0.25⋯0.75)➩Cubic(0.25, 0.10, 0.25, 1.00)"; got != want {
		t.Errorf("Interval.String = %q, want %q", got, want)
	}
}

func TestCurveFlipped(t *testing.T) {
	ease := Curves.Ease
	flipped := ease.Flipped()
	if v := flipped.Transform(0); v >= 0.001 {
		t.Errorf("flipped(0) = %v", v)
	}
	if flipped.Transform(0.5) >= ease.Transform(0.5) {
		t.Errorf("flipped(0.5) = %v, want less than ease(0.5) = %v", flipped.Transform(0.5), ease.Transform(0.5))
	}
	if v := flipped.Transform(1); v <= 0.999 {
		t.Errorf("flipped(1) = %v", v)
	}
}

func TestThresholdHasAThreshold(t *testing.T) {
	step := NewThreshold(0.25)
	for _, tt := range []struct{ in, want float64 }{{0, 0}, {0.24, 0}, {0.25, 1}, {0.26, 1}, {1, 1}} {
		if got := step.Transform(tt.in); got != tt.want {
			t.Errorf("Threshold(0.25)(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCurveIsContinuous(t *testing.T) {
	const delta = 0.005
	curves := map[string]Curve{
		"linear": Curves.Linear, "decelerate": Curves.Decelerate, "fastOutSlowIn": Curves.FastOutSlowIn,
		"slowMiddle": Curves.SlowMiddle, "bounceIn": Curves.BounceIn, "bounceOut": Curves.BounceOut,
		"bounceInOut": Curves.BounceInOut, "elasticOut": Curves.ElasticOut, "elasticInOut": Curves.ElasticInOut,
		"ease": Curves.Ease, "easeIn": Curves.EaseIn, "easeInSine": Curves.EaseInSine, "easeInQuad": Curves.EaseInQuad,
		"easeInCubic": Curves.EaseInCubic, "easeInQuart": Curves.EaseInQuart, "easeInQuint": Curves.EaseInQuint,
		"easeInExpo": Curves.EaseInExpo, "easeInCirc": Curves.EaseInCirc, "easeOut": Curves.EaseOut,
		"easeOutSine": Curves.EaseOutSine, "easeOutQuad": Curves.EaseOutQuad, "easeOutCubic": Curves.EaseOutCubic,
		"easeInOutCubicEmphasized": Curves.EaseInOutCubicEmphasized, "easeOutQuart": Curves.EaseOutQuart,
		"easeOutQuint": Curves.EaseOutQuint, "easeOutExpo": Curves.EaseOutExpo, "easeOutCirc": Curves.EaseOutCirc,
		"easeInOut": Curves.EaseInOut, "easeInOutSine": Curves.EaseInOutSine, "easeInOutQuad": Curves.EaseInOutQuad,
		"easeInOutCubic": Curves.EaseInOutCubic, "easeInOutQuart": Curves.EaseInOutQuart,
		"easeInOutQuint": Curves.EaseInOutQuint, "easeInOutCirc": Curves.EaseInOutCirc,
	}
	// Curves.EaseInOutExpo is discontinuous at its midpoint, so not included.
	for name, c := range curves {
		for x := 0.0; x < 1.0-delta; x += delta {
			if dy := math.Abs(c.Transform(x) - c.Transform(x+delta)); dy >= delta*20 {
				t.Errorf("%s discontinuous at %v", name, x)
				break
			}
		}
	}
}

func estimateBounds(c Curve) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for i := 0; i <= 10; i++ {
		v := c.Transform(float64(i) / 10)
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	return lo, hi
}

func TestBounceStaysInBounds(t *testing.T) {
	for name, c := range map[string]Curve{"in": Curves.BounceIn, "out": Curves.BounceOut, "inOut": Curves.BounceInOut} {
		if lo, hi := estimateBounds(c); lo < 0 || hi > 1 {
			t.Errorf("bounce %s bounds = [%v, %v]", name, lo, hi)
		}
	}
}

func TestElasticAndBackOvershoot(t *testing.T) {
	tests := []struct {
		name         string
		curve        Curve
		loBelowZero  bool
		hiAboveOne   bool
		hiAtMostOne  bool
		loAtLeastZer bool
	}{
		{"elasticIn", Curves.ElasticIn, true, false, true, false},
		{"elasticOut", Curves.ElasticOut, false, true, false, true},
		{"elasticInOut", Curves.ElasticInOut, true, true, false, false},
		{"easeInBack", Curves.EaseInBack, true, false, true, false},
		{"easeOutBack", Curves.EaseOutBack, false, true, false, true},
		{"easeInOutBack", Curves.EaseInOutBack, true, true, false, false},
	}
	for _, tt := range tests {
		lo, hi := estimateBounds(tt.curve)
		if tt.loBelowZero && !(lo < 0) || tt.loAtLeastZer && !(lo >= 0) || tt.hiAboveOne && !(hi > 1) || tt.hiAtMostOne && !(hi <= 1) {
			t.Errorf("%s bounds = [%v, %v]", tt.name, lo, hi)
		}
	}
}

func TestDecelerateDoesSo(t *testing.T) {
	lo, hi := estimateBounds(Curves.Decelerate)
	if lo < 0 || hi > 1 {
		t.Errorf("bounds = [%v, %v]", lo, hi)
	}
	d1 := Curves.Decelerate.Transform(0.2) - Curves.Decelerate.Transform(0)
	d2 := Curves.Decelerate.Transform(1) - Curves.Decelerate.Transform(0.8)
	if !(d2 < d1) {
		t.Errorf("d2 = %v, want less than d1 = %v", d2, d1)
	}
}

func TestThreePointCubicInterpolatesMidpoint(t *testing.T) {
	c := NewThreePointCubic(ui.Offset{Dx: 0.05, Dy: 0}, ui.Offset{Dx: 0.133333, Dy: 0.06}, ui.Offset{Dx: 0.166666, Dy: 0.4},
		ui.Offset{Dx: 0.208333, Dy: 0.82}, ui.Offset{Dx: 0.25, Dy: 1})
	if got := c.Transform(0.166666); got != 0.4 {
		t.Errorf("Transform(midpoint) = %v, want 0.4", got)
	}
}

func TestCubicTransformInternalClampsOutsideTheUnitInterval(t *testing.T) {
	if got := Curves.EaseOut.transformInternal(2); got != 1 {
		t.Errorf("transformInternal(2) = %v, want 1", got)
	}
	if got := Curves.EaseOut.transformInternal(-1); got != 0 {
		t.Errorf("transformInternal(-1) = %v, want 0", got)
	}
}

func TestCubicTransformInternalPanicsForNaN(t *testing.T) {
	// The tag is checked in every build mode: it is not an assertion.
	if tag := panicTag(t, func() { Curves.EaseOut.transformInternal(math.NaN()) }); tag != "CubicNaN" {
		t.Errorf("panic tag = %q, want CubicNaN", tag)
	}
}

func TestInvalidTransformParameterShouldAssert(t *testing.T) {
	skipUnlessDebug(t)
	curves := map[string]Curve{
		"SawTooth": NewSawTooth(2), "Interval": NewInterval(0, 1), "Split": NewSplit(0), "Threshold": NewThreshold(0.5),
		"ElasticIn": NewElasticInCurve(), "ElasticOut": NewElasticOutCurve(), "Cubic": NewCubic(0.42, 0, 0.58, 1),
		"Emphasized": Curves.EaseInOutCubicEmphasized, "Decelerate": Curves.Decelerate, "BounceIn": Curves.BounceIn,
		"BounceOut": Curves.BounceOut, "BounceInOut": Curves.BounceInOut, "Linear": Curves.Linear,
		"Flipped": Curves.Ease.Flipped(), "ElasticInOut": Curves.ElasticInOut,
	}
	for name, c := range curves {
		for _, in := range []float64{-0.0001, 1.0001} {
			if tag := panicTag(t, func() { c.Transform(in) }); tag != "CurveParameterOutOfRange" {
				t.Errorf("%s.Transform(%v) panic tag = %q, want CurveParameterOutOfRange", name, in, tag)
			}
		}
	}
}

func TestCurveTransformEndpoints(t *testing.T) {
	for name, c := range map[string]Curve{
		"SawTooth": NewSawTooth(2), "Interval": NewInterval(0, 1), "Split": NewSplit(0.5), "Threshold": NewThreshold(0.5),
		"ElasticIn": NewElasticInCurve(), "ElasticOut": NewElasticOutCurve(), "ElasticInOut": NewElasticInOutCurve(),
		"Linear": Curves.Linear, "EaseInOutExpo": Curves.EaseInOutExpo, "Emphasized": Curves.EaseInOutCubicEmphasized,
		"FlippedExpo": NewFlippedCurve(Curves.EaseInOutExpo), "Decelerate": Curves.Decelerate,
		"BounceIn": Curves.BounceIn, "BounceOut": Curves.BounceOut, "BounceInOut": Curves.BounceInOut,
	} {
		if c.Transform(0) != 0 || c.Transform(1) != 1 {
			t.Errorf("%s: Transform(0) = %v, Transform(1) = %v", name, c.Transform(0), c.Transform(1))
		}
	}
}

func TestSplitInterpolatesValuesProperly(t *testing.T) {
	c := NewSplit(0.3)
	for _, tt := range []struct{ in, want, eps float64 }{
		{0, 0, 0}, {0.1, 0.1, 1e-12}, {0.25, 0.25, 1e-12}, {0.3, 0.3, 0}, {0.5, 0.760461, 1e-6}, {0.75, 0.962055, 1e-6}, {1, 1, 0},
	} {
		near(t, "Split(0.3)", c.Transform(tt.in), tt.want, tt.eps)
	}
}

var catmullPoints = []ui.Offset{
	{}, {Dx: 0.01, Dy: 0.25}, {Dx: 0.2, Dy: 0.25}, {Dx: 0.33, Dy: 0.25}, {Dx: 0.5, Dy: 1}, {Dx: 0.66, Dy: 0.75}, {Dx: 1, Dy: 1},
}

func checkCatmullRomSpline(t *testing.T, c *CatmullRomSpline) {
	t.Helper()
	const tol = 1e-6
	for _, tt := range []struct{ at, dx, dy float64 }{
		{0, 0, 0}, {0.25, 0.0966945, 0.2626806}, {0.5, 0.33, 0.25}, {0.75, 0.570260, 0.883085}, {1, 1, 1},
	} {
		p := c.Transform(tt.at)
		near(t, "spline dx", p.Dx, tt.dx, tol)
		near(t, "spline dy", p.Dy, tt.dy, tol)
	}
}

func TestCatmullRomSplineInterpolatesValuesProperly(t *testing.T) {
	opts := CatmullRomSplineOpts{StartHandle: foundation.Some(ui.Offset{Dy: -0.3}), EndHandle: foundation.Some(ui.Offset{Dx: 1.3, Dy: 1.3})}
	checkCatmullRomSpline(t, NewCatmullRomSpline(catmullPoints, opts))
	checkCatmullRomSpline(t, NewCatmullRomSplinePrecompute(catmullPoints, opts))
}

func TestCatmullRomSplineEnforcesContract(t *testing.T) {
	skipUnlessDebug(t)
	zero := ui.Offset{}
	inf := ui.Offset{Dx: math.Inf(1)}
	infY := ui.Offset{Dy: math.Inf(1)}
	four := []ui.Offset{zero, zero, zero, zero}
	withFirst := func(o ui.Offset) []ui.Offset { return []ui.Offset{o, zero, zero, zero} }
	for _, precompute := range []bool{false, true} {
		build := func(points []ui.Offset, opts ...CatmullRomSplineOpts) func() {
			return func() {
				if precompute {
					NewCatmullRomSplinePrecompute(points, opts...)
				} else {
					Curve2DGenerateSamples(NewCatmullRomSpline(points, opts...))
				}
			}
		}
		for name, fn := range map[string]func(){
			"empty":        build(nil),
			"one":          build([]ui.Offset{zero}),
			"two":          build([]ui.Offset{zero, zero}),
			"three":        build([]ui.Offset{zero, zero, zero}),
			"tension -1":   build(four, CatmullRomSplineOpts{Tension: -1}),
			"tension 2":    build(four, CatmullRomSplineOpts{Tension: 2}),
			"infinite x":   build(withFirst(inf)),
			"infinite y":   build(withFirst(infY)),
			"start handle": build(four, CatmullRomSplineOpts{StartHandle: foundation.Some(infY)}),
			"end handle":   build(four, CatmullRomSplineOpts{EndHandle: foundation.Some(infY)}),
		} {
			if tag := panicTag(t, fn); tag != "CatmullRomSplineInvalid" {
				t.Errorf("precompute=%v %s: panic tag = %q, want CatmullRomSplineInvalid", precompute, name, tag)
			}
		}
	}
}

var catmullCurvePoints = []ui.Offset{{Dx: 0.2, Dy: 0.25}, {Dx: 0.33, Dy: 0.25}, {Dx: 0.5, Dy: 1}, {Dx: 0.8, Dy: 0.75}}

func checkCatmullRomCurve(t *testing.T, c *CatmullRomCurve) {
	t.Helper()
	for _, tt := range []struct{ at, want float64 }{
		{0, 0}, {0.01, 0.012874734350170863}, {0.2, 0.24989646045277542}, {0.33, 0.250037698527661},
		{0.5, 0.9999057323235939}, {0.6, 0.9357294964536621}, {0.8, 0.7500423402378034}, {1, 1},
	} {
		// Deviation: the samples come from another pseudo random generator than
		// Dart's, so the curve differs from Flutter's by up to the sampling
		// tolerance (1e-4) instead of 1e-6.
		near(t, "curve", c.Transform(tt.at), tt.want, 2e-4)
	}
}

func TestCatmullRomCurveInterpolatesGivenPointsCorrectly(t *testing.T) {
	checkCatmullRomCurve(t, NewCatmullRomCurve(catmullCurvePoints))
	checkCatmullRomCurve(t, NewCatmullRomCurvePrecompute(catmullCurvePoints))
}

func TestCatmullRomCurveEnforcesContract(t *testing.T) {
	skipUnlessDebug(t)
	zero := ui.Offset{}
	invalid := map[string][]ui.Offset{
		"empty":                      nil,
		"one":                        {zero},
		"two zeros":                  {zero, zero},
		"not monotonic":              {{Dx: 0.2, Dy: 0.25}, {Dx: 0.01, Dy: 0.25}},
		"x out of range":             {{Dx: 0.2, Dy: 0.25}, {Dx: 1.01, Dy: 0.25}},
		"multi-valued at 0":          {{Dx: 0.05, Dy: 0.5}, {Dx: 0.5, Dy: 0.5}, {Dx: 0.75, Dy: 0.75}},
		"multi-valued at 1":          {{Dx: 0.25, Dy: 0.25}, {Dx: 0.5, Dy: 0.5}, {Dx: 0.95, Dy: 0.51}},
		"multi-valued in the middle": {{Dx: 0.5, Dy: 0.05}, {Dx: 0.5, Dy: 0.95}},
	}
	for name, points := range invalid {
		if len(points) > 2 || name == "not monotonic" || name == "x out of range" || name == "multi-valued in the middle" {
			var reasons []string
			if CatmullRomCurveValidateControlPoints(points, 0, &reasons) {
				t.Errorf("%s: control points should not be valid", name)
			}
			if len(reasons) == 0 {
				t.Errorf("%s: no reasons given", name)
			}
		}
		if tag := panicTag(t, func() { NewCatmullRomCurve(points) }); tag != "CatmullRomCurveInvalid" {
			t.Errorf("%s: panic tag = %q, want CatmullRomCurveInvalid", name, tag)
		}
		if tag := panicTag(t, func() { NewCatmullRomCurvePrecompute(points) }); tag != "CatmullRomCurveInvalid" {
			t.Errorf("%s precomputed: panic tag = %q, want CatmullRomCurveInvalid", name, tag)
		}
	}
	var reasons []string
	if !CatmullRomCurveValidateControlPoints(catmullCurvePoints, 0, &reasons) || len(reasons) != 0 {
		t.Errorf("the valid points were rejected: %s", strings.Join(reasons, "; "))
	}
}

func TestCurve2DGenerateSamplesRange(t *testing.T) {
	skipUnlessDebug(t)
	s := NewCatmullRomSpline(catmullPoints)
	if tag := panicTag(t, func() { Curve2DGenerateSamples(s, Curve2DGenerateSamplesOpts{Start: 0.5, End: 0.5}) }); tag != "Curve2DRangeInvalid" {
		t.Errorf("panic tag = %q, want Curve2DRangeInvalid", tag)
	}
}

func TestCurveFuncAndFlipped(t *testing.T) {
	calls := 0
	c := CurveFunc(func(t float64) float64 { calls++; return t * t })
	if c.Transform(0) != 0 || c.Transform(1) != 1 || calls != 0 {
		t.Errorf("the endpoints must not call the function (calls = %d)", calls)
	}
	near(t, "CurveFunc(0.5)", c.Transform(0.5), 0.25, 1e-12)
	near(t, "flipped(0.5)", c.Flipped().Transform(0.5), 0.75, 1e-12)
	if _, ok := c.Flipped().Flipped().(FlippedCurve); !ok {
		t.Errorf("flipping a flipped curve gives %T", c.Flipped().Flipped())
	}
}

func TestIntervalAndSawToothValues(t *testing.T) {
	iv := NewInterval(0.25, 0.75)
	for _, tt := range []struct{ in, want float64 }{{0.1, 0}, {0.25, 0}, {0.5, 0.5}, {0.75, 1}, {0.9, 1}} {
		near(t, "interval", iv.Transform(tt.in), tt.want, 1e-12)
	}
	near(t, "sawtooth", NewSawTooth(4).Transform(0.3), 0.2, 1e-12)
	skipUnlessDebug(t)
	if tag := panicTag(t, func() { NewInterval(0.6, 0.4).Transform(0.5) }); tag != "IntervalInvalid" {
		t.Errorf("panic tag = %q, want IntervalInvalid", tag)
	}
	if tag := panicTag(t, func() { NewThreshold(2).Transform(0.5) }); tag != "ThresholdInvalid" {
		t.Errorf("panic tag = %q, want ThresholdInvalid", tag)
	}
	if tag := panicTag(t, func() { NewSplit(2).Transform(0.5) }); tag != "SplitInvalid" {
		t.Errorf("panic tag = %q, want SplitInvalid", tag)
	}
}
