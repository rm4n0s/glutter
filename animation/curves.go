// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ParametricCurve is a mapping of the unit interval to type T. Transform panics
// with the tag CurveParameterOutOfRange in debug builds when t is outside 0 to 1.
//
// Flutter: animation/curves.dart ParametricCurve
type ParametricCurve[T any] interface {
	// Transform returns the value of the curve at t.
	//
	// Flutter: animation/curves.dart ParametricCurve.transform
	Transform(t float64) T
}

// Curve is a map of the unit interval to the unit interval, with Transform(0)
// near 0 and Transform(1) near 1; it is used to ease animations.
//
// Flutter: animation/curves.dart Curve
type Curve interface {
	ParametricCurve[float64]
	// Flipped is the curve mirrored on both axes.
	//
	// Flutter: animation/curves.dart Curve.flipped
	Flipped() Curve
}

func assertCurveParameter(t float64) {
	if foundation.DebugMode && !(t >= 0 && t <= 1) {
		panic(errors.New("CurveParameterOutOfRange", fmt.Sprintf("parametric value %v is outside of [0, 1] range", t), "t", t))
	}
}

// curveInternal is a curve that gives its own values between 0 and 1.
type curveInternal interface{ transformInternal(t float64) float64 }

func transformCurve(c curveInternal, t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	assertCurveParameter(t)
	return c.transformInternal(t)
}

// CurveFunc is a Curve made of a function that gives the values for t between
// 0 and 1 (exclusive): Transform returns 0 and 1 for them without calling it.
//
// Flutter: animation/curves.dart Curve (a subclass that overrides transformInternal)
type CurveFunc func(t float64) float64

func (f CurveFunc) transformInternal(t float64) float64 { return f(t) }

// Transform returns t for 0 and 1 and the function's value between.
//
// Flutter: animation/curves.dart Curve.transform
func (f CurveFunc) Transform(t float64) float64 { return transformCurve(f, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (f CurveFunc) Flipped() Curve { return FlippedCurve{f} }

type linearCurve struct{}

func (linearCurve) transformInternal(t float64) float64 { return t }
func (c linearCurve) Transform(t float64) float64       { return transformCurve(c, t) }
func (c linearCurve) Flipped() Curve                    { return FlippedCurve{c} }
func (linearCurve) String() string                      { return "Linear" }

// SawTooth is a sawtooth curve that repeats a given number of times over the
// unit interval; its value goes from 0 to 1 each time.
//
// Flutter: animation/curves.dart SawTooth
type SawTooth struct{ Count int }

// NewSawTooth creates the curve.
//
// Flutter: animation/curves.dart SawTooth.new
func NewSawTooth(count int) SawTooth { return SawTooth{count} }

func (c SawTooth) transformInternal(t float64) float64 {
	t *= float64(c.Count)
	return t - math.Trunc(t)
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart SawTooth.transformInternal
func (c SawTooth) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c SawTooth) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart SawTooth.toString
func (c SawTooth) String() string { return fmt.Sprintf("SawTooth(%d)", c.Count) }

// Interval is a curve that is 0.0 until Begin, then follows another curve until
// End, then 1.0.
//
// Flutter: animation/curves.dart Interval
type Interval struct {
	// Begin is the largest value for which the curve is 0.
	Begin float64
	// End is the smallest value for which the curve is 1.
	End float64
	// Curve is the curve between Begin and End; the default is Curves.Linear.
	Curve Curve
}

// NewInterval creates the curve; the curve (nil for linear) is applied between
// begin and end.
//
// Flutter: animation/curves.dart Interval.new
func NewInterval(begin, end float64, curve ...Curve) Interval {
	i := Interval{Begin: begin, End: end, Curve: Curves.Linear}
	if len(curve) > 0 && curve[0] != nil {
		i.Curve = curve[0]
	}
	return i
}

func (c Interval) transformInternal(t float64) float64 {
	if foundation.DebugMode && !(c.Begin >= 0 && c.Begin <= 1 && c.End >= 0 && c.End <= 1 && c.End >= c.Begin) {
		panic(errors.New("IntervalInvalid", "the begin and end of an Interval must be in 0 to 1, with the end not before the begin", "begin", c.Begin, "end", c.End))
	}
	t = ui.ClampDouble((t-c.Begin)/(c.End-c.Begin), 0, 1)
	if t == 0 || t == 1 {
		return t
	}
	return c.Curve.Transform(t)
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart Interval.transformInternal
func (c Interval) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c Interval) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart Interval.toString
func (c Interval) String() string {
	if _, linear := c.Curve.(linearCurve); !linear {
		return fmt.Sprintf("Interval(%v⋯%v)➩%v", c.Begin, c.End, c.Curve)
	}
	return fmt.Sprintf("Interval(%v⋯%v)", c.Begin, c.End)
}

// Split is a curve that progresses according to BeginCurve until Split, then
// according to EndCurve.
//
// Flutter: animation/curves.dart Split
type Split struct {
	// Split is the progress at which the curve changes, from 0 to 1.
	Split float64
	// BeginCurve is the curve before the split; the default is Curves.Linear.
	BeginCurve Curve
	// EndCurve is the curve after the split; the default is Curves.EaseOutCubic.
	EndCurve Curve
}

// SplitOpts are the optional parameters of NewSplit.
//
// Flutter: animation/curves.dart Split.new
type SplitOpts struct {
	// BeginCurve is the curve before the split; the default is Curves.Linear.
	BeginCurve Curve
	// EndCurve is the curve after the split; the default is Curves.EaseOutCubic.
	EndCurve Curve
}

// NewSplit creates the curve.
//
// Flutter: animation/curves.dart Split.new
func NewSplit(split float64, opts ...SplitOpts) Split {
	o := firstOpts(opts)
	if o.BeginCurve == nil {
		o.BeginCurve = Curves.Linear
	}
	if o.EndCurve == nil {
		o.EndCurve = Curves.EaseOutCubic
	}
	return Split{split, o.BeginCurve, o.EndCurve}
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart Split.transform
func (c Split) Transform(t float64) float64 {
	assertCurveParameter(t)
	if foundation.DebugMode && !(c.Split >= 0 && c.Split <= 1) {
		panic(errors.New("SplitInvalid", "the split of a Split curve must be in 0 to 1", "split", c.Split))
	}
	if t == 0 || t == 1 {
		return t
	}
	if t == c.Split {
		return c.Split
	}
	if t < c.Split {
		return ui.LerpDouble(0, c.Split, c.BeginCurve.Transform(t/c.Split))
	}
	return ui.LerpDouble(c.Split, 1, c.EndCurve.Transform((t-c.Split)/(1-c.Split)))
}

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c Split) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart Split.toString
func (c Split) String() string {
	return fmt.Sprintf("Split(%v, %v, %v)", c.Split, c.BeginCurve, c.EndCurve)
}

// Threshold is a curve that is 0 until the threshold, then jumps to 1.
//
// Flutter: animation/curves.dart Threshold
type Threshold struct{ Threshold float64 }

// NewThreshold creates the curve.
//
// Flutter: animation/curves.dart Threshold.new
func NewThreshold(threshold float64) Threshold { return Threshold{threshold} }

func (c Threshold) transformInternal(t float64) float64 {
	if foundation.DebugMode && !(c.Threshold >= 0 && c.Threshold <= 1) {
		panic(errors.New("ThresholdInvalid", "the threshold must be in 0 to 1", "threshold", c.Threshold))
	}
	if t < c.Threshold {
		return 0
	}
	return 1
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart Threshold.transformInternal
func (c Threshold) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c Threshold) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart Threshold
func (c Threshold) String() string { return fmt.Sprintf("Threshold(%v)", c.Threshold) }

// Cubic is a cubic polynomial mapping of the unit interval: a Bézier curve from
// (0, 0) to (1, 1) with the control points (A, B) and (C, D).
//
// Flutter: animation/curves.dart Cubic
type Cubic struct{ A, B, C, D float64 }

// NewCubic creates the curve.
//
// Flutter: animation/curves.dart Cubic.new
func NewCubic(a, b, c, d float64) Cubic { return Cubic{a, b, c, d} }

const cubicErrorBound = 0.001

func evaluateCubic(a, b, m float64) float64 {
	return 3*a*(1-m)*(1-m)*m + 3*b*(1-m)*m*m + m*m*m
}

func (c Cubic) transformInternal(t float64) float64 {
	if math.IsNaN(t) {
		panic(errors.New("CubicNaN", "the parameter t of a Cubic must not be NaN"))
	}
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	start, end := 0.0, 1.0
	for {
		midpoint := (start + end) / 2
		estimate := evaluateCubic(c.A, c.C, midpoint)
		if math.Abs(t-estimate) < cubicErrorBound {
			return evaluateCubic(c.B, c.D, midpoint)
		}
		if estimate < t {
			start = midpoint
		} else {
			end = midpoint
		}
	}
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart Cubic.transformInternal
func (c Cubic) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c Cubic) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart Cubic.toString
func (c Cubic) String() string {
	return fmt.Sprintf("Cubic(%.2f, %.2f, %.2f, %.2f)", c.A, c.B, c.C, c.D)
}

// ThreePointCubic is a curve made of two cubic Bézier curves that meet at a
// midpoint.
//
// Flutter: animation/curves.dart ThreePointCubic
type ThreePointCubic struct {
	// A1 is the first control point of the first curve.
	A1 ui.Offset
	// B1 is the second control point of the first curve.
	B1 ui.Offset
	// Midpoint is the end point of the first curve and the start of the second.
	Midpoint ui.Offset
	// A2 is the first control point of the second curve.
	A2 ui.Offset
	// B2 is the second control point of the second curve.
	B2 ui.Offset
}

// NewThreePointCubic creates the curve.
//
// Flutter: animation/curves.dart ThreePointCubic.new
func NewThreePointCubic(a1, b1, midpoint, a2, b2 ui.Offset) ThreePointCubic {
	return ThreePointCubic{a1, b1, midpoint, a2, b2}
}

func (c ThreePointCubic) transformInternal(t float64) float64 {
	first := t < c.Midpoint.Dx
	scaleX, scaleY := 1.0-c.Midpoint.Dx, 1.0-c.Midpoint.Dy
	if first {
		scaleX, scaleY = c.Midpoint.Dx, c.Midpoint.Dy
	}
	from := 0.0
	if !first {
		from = c.Midpoint.Dx
	}
	scaledT := (t - from) / scaleX
	if first {
		return NewCubic(c.A1.Dx/scaleX, c.A1.Dy/scaleY, c.B1.Dx/scaleX, c.B1.Dy/scaleY).Transform(scaledT) * scaleY
	}
	return NewCubic((c.A2.Dx-c.Midpoint.Dx)/scaleX, (c.A2.Dy-c.Midpoint.Dy)/scaleY, (c.B2.Dx-c.Midpoint.Dx)/scaleX, (c.B2.Dy-c.Midpoint.Dy)/scaleY).
		Transform(scaledT)*scaleY + c.Midpoint.Dy
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart ThreePointCubic.transformInternal
func (c ThreePointCubic) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c ThreePointCubic) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart ThreePointCubic.toString
func (c ThreePointCubic) String() string {
	return fmt.Sprintf("ThreePointCubic(%v, %v, %v, %v, %v) ", c.A1, c.B1, c.Midpoint, c.A2, c.B2)
}

// Curve2DSample is a sample of a Curve2D: a value at a parameter t.
//
// Flutter: animation/curves.dart Curve2DSample
type Curve2DSample struct {
	// T is the parameter of the sample.
	T float64
	// Value is the point of the curve at T.
	Value ui.Offset
}

// String describes the sample.
//
// Flutter: animation/curves.dart Curve2DSample.toString
func (s Curve2DSample) String() string {
	return fmt.Sprintf("[(%.2f, %.2f), %.2f]", s.Value.Dx, s.Value.Dy, s.T)
}

// Curve2D is a curve of the unit interval to 2D points.
//
// Flutter: animation/curves.dart Curve2D
type Curve2D interface {
	ParametricCurve[ui.Offset]
}

// samplingSeeder is a Curve2D with its own seed for the sampling.
type samplingSeeder interface{ SamplingSeed() int }

// Curve2DGenerateSamplesOpts are the optional parameters of
// Curve2DGenerateSamples.
//
// Flutter: animation/curves.dart Curve2D.generateSamples
type Curve2DGenerateSamplesOpts struct {
	// Start is the beginning of the range; the default is 0.
	Start float64
	// End is the end of the range; the default is 1.
	End float64
	// Tolerance is how flat a part of the curve must be to stop dividing it; the
	// default is 1e-10.
	Tolerance float64
}

// Curve2DGenerateSamples samples the curve so that a line between the samples is
// within the tolerance of it. It panics with the tag Curve2DRangeInvalid when
// the end is not after the start.
//
// Deviation: the points to sample are chosen with a seeded pseudo random number
// generator like Flutter's, but not the same one, so the samples differ slightly
// while the curve they describe is the same within the tolerance (SPEC §12.10).
//
// Flutter: animation/curves.dart Curve2D.generateSamples
func Curve2DGenerateSamples(c Curve2D, opts ...Curve2DGenerateSamplesOpts) []Curve2DSample {
	o := firstOpts(opts)
	if o.End == 0 && o.Start == 0 {
		o.End = 1
	}
	if o.Tolerance == 0 {
		o.Tolerance = 1e-10
	}
	if !(o.End > o.Start) {
		panic(errors.New("Curve2DRangeInvalid", "the end of the range must be after its start", "start", o.Start, "end", o.End))
	}
	seed := 0
	if s, ok := c.(samplingSeeder); ok {
		seed = s.SamplingSeed()
	}
	rng := rand.New(rand.NewPCG(uint64(int64(seed)), uint64(int64(seed))^0x9e3779b97f4a7c15))
	isFlat := func(p, q, r ui.Offset) bool {
		pr, qr := p.Sub(r), q.Sub(r)
		z := pr.Dx*qr.Dy - qr.Dx*pr.Dy
		return z*z < o.Tolerance
	}
	first := Curve2DSample{o.Start, c.Transform(o.Start)}
	last := Curve2DSample{o.End, c.Transform(o.End)}
	samples := []Curve2DSample{first}
	var sample func(p, q Curve2DSample, forceSubdivide bool)
	sample = func(p, q Curve2DSample, forceSubdivide bool) {
		t := p.T + (0.45+0.1*rng.Float64())*(q.T-p.T)
		r := Curve2DSample{t, c.Transform(t)}
		if !forceSubdivide && isFlat(p.Value, q.Value, r.Value) {
			samples = append(samples, q)
		} else {
			sample(p, r, false)
			sample(r, q, false)
		}
	}
	sample(first, last, math.Abs(first.Value.Dx-last.Value.Dx) < o.Tolerance && math.Abs(first.Value.Dy-last.Value.Dy) < o.Tolerance)
	return samples
}

// Curve2DFindInverse finds the parameter t where the curve's x is the given x,
// by bisection.
//
// Flutter: animation/curves.dart Curve2D.findInverse
func Curve2DFindInverse(c Curve2D, x float64) float64 {
	start, end := 0.0, 1.0
	mid := 0.0
	offsetToOrigin := func(pos float64) float64 { return x - c.Transform(pos).Dx }
	const errorLimit = 1e-6
	count := 100
	startValue := offsetToOrigin(start)
	for (end-start)/2 > errorLimit && count > 0 {
		mid = (end + start) / 2
		value := offsetToOrigin(mid)
		if math.Signbit(value) == math.Signbit(startValue) {
			start = mid
		} else {
			end = mid
		}
		count--
	}
	return mid
}

// CatmullRomSpline is a 2D spline that passes smoothly through the control
// points, using a centripetal Catmull-Rom parametrization.
//
// Flutter: animation/curves.dart CatmullRomSpline
type CatmullRomSpline struct {
	cubicSegments [][]ui.Offset
	controlPoints []ui.Offset
	startHandle   foundation.Opt[ui.Offset]
	endHandle     foundation.Opt[ui.Offset]
	tension       float64
}

// CatmullRomSplineOpts are the optional parameters of NewCatmullRomSpline.
//
// Flutter: animation/curves.dart CatmullRomSpline.new
type CatmullRomSplineOpts struct {
	// Tension is how tightly the curve bends, from 0 (the default) to 1.
	Tension float64
	// StartHandle is the point before the first control point; the default
	// extends the first segment backwards.
	StartHandle foundation.Opt[ui.Offset]
	// EndHandle is the point after the last control point; the default extends
	// the last segment forwards.
	EndHandle foundation.Opt[ui.Offset]
}

func assertSplineArguments(controlPoints []ui.Offset, tension float64) {
	if !foundation.DebugMode {
		return
	}
	if tension > 1 || tension < 0 {
		panic(errors.New("CatmullRomSplineInvalid", fmt.Sprintf("tension %v must be from 0 to 1", tension), "tension", tension))
	}
	if len(controlPoints) <= 3 {
		panic(errors.New("CatmullRomSplineInvalid", "there must be at least four control points to create a CatmullRomSpline", "count", len(controlPoints)))
	}
}

// NewCatmullRomSpline creates the spline; the segments are computed when it is
// first used. It panics with the tag CatmullRomSplineInvalid in debug builds
// when the tension is not from 0 to 1 or there are fewer than four points.
//
// Flutter: animation/curves.dart CatmullRomSpline.new
func NewCatmullRomSpline(controlPoints []ui.Offset, opts ...CatmullRomSplineOpts) *CatmullRomSpline {
	o := firstOpts(opts)
	assertSplineArguments(controlPoints, o.Tension)
	return &CatmullRomSpline{controlPoints: controlPoints, startHandle: o.StartHandle, endHandle: o.EndHandle, tension: o.Tension}
}

// NewCatmullRomSplinePrecompute creates the spline and computes its segments at
// once.
//
// Flutter: animation/curves.dart CatmullRomSpline.precompute
func NewCatmullRomSplinePrecompute(controlPoints []ui.Offset, opts ...CatmullRomSplineOpts) *CatmullRomSpline {
	o := firstOpts(opts)
	assertSplineArguments(controlPoints, o.Tension)
	return &CatmullRomSpline{cubicSegments: computeSplineSegments(controlPoints, o.Tension, o.StartHandle, o.EndHandle)}
}

func computeSplineSegments(controlPoints []ui.Offset, tension float64, startHandle, endHandle foundation.Opt[ui.Offset]) [][]ui.Offset {
	if foundation.DebugMode {
		if h, ok := startHandle.Get(); ok && !h.IsFinite() {
			panic(errors.New("CatmullRomSplineInvalid", fmt.Sprintf("the provided startHandle of CatmullRomSpline must be finite; it was %v", h)))
		}
		if h, ok := endHandle.Get(); ok && !h.IsFinite() {
			panic(errors.New("CatmullRomSplineInvalid", fmt.Sprintf("the provided endHandle of CatmullRomSpline must be finite; it was %v", h)))
		}
		for i, p := range controlPoints {
			if !p.IsFinite() {
				panic(errors.New("CatmullRomSplineInvalid", fmt.Sprintf("the provided CatmullRomSpline control point at index %d is not finite; it was %v", i, p)))
			}
		}
	}
	start := startHandle.OrElse(controlPoints[0].Mul(2).Sub(controlPoints[1]))
	end := endHandle.OrElse(controlPoints[len(controlPoints)-1].Mul(2).Sub(controlPoints[len(controlPoints)-2]))
	all := append(append([]ui.Offset{start}, controlPoints...), end)
	const alpha = 0.5
	reverseTension := 1 - tension
	var result [][]ui.Offset
	for i := 0; i < len(all)-3; i++ {
		curve := all[i : i+4]
		d10, d21, d32 := curve[1].Sub(curve[0]), curve[2].Sub(curve[1]), curve[3].Sub(curve[2])
		t01, t12, t23 := math.Pow(d10.Distance(), alpha), math.Pow(d21.Distance(), alpha), math.Pow(d32.Distance(), alpha)
		m1 := d21.Add(d10.Div(t01).Sub(curve[2].Sub(curve[0]).Div(t01 + t12)).Mul(t12)).Mul(reverseTension)
		m2 := d21.Add(d32.Div(t23).Sub(curve[3].Sub(curve[1]).Div(t12 + t23)).Mul(t12)).Mul(reverseTension)
		sumM12 := m1.Add(m2)
		result = append(result, []ui.Offset{
			d21.Mul(-2).Add(sumM12),
			d21.Mul(3).Sub(m1).Sub(sumM12),
			m1,
			curve[1],
		})
	}
	return result
}

func (s *CatmullRomSpline) initializeIfNeeded() {
	if len(s.cubicSegments) == 0 {
		s.cubicSegments = computeSplineSegments(s.controlPoints, s.tension, s.startHandle, s.endHandle)
	}
}

// SamplingSeed is the seed that Curve2DGenerateSamples uses: it comes from the
// first segment, so the samples are the same for the same spline.
//
// Flutter: animation/curves.dart CatmullRomSpline.samplingSeed
func (s *CatmullRomSpline) SamplingSeed() int {
	s.initializeIfNeeded()
	seed := s.cubicSegments[0][1]
	return int(math.Round((seed.Dx + seed.Dy) * 10000))
}

// Transform returns the point of the spline at t.
//
// Flutter: animation/curves.dart CatmullRomSpline.transformInternal
func (s *CatmullRomSpline) Transform(t float64) ui.Offset {
	assertCurveParameter(t)
	s.initializeIfNeeded()
	length := float64(len(s.cubicSegments))
	var localT float64
	var index int
	if t < 1 {
		position := t * length
		localT = math.Mod(position, 1)
		index = int(math.Floor(position))
	} else {
		localT = 1
		index = len(s.cubicSegments) - 1
	}
	c := s.cubicSegments[index]
	localT2 := localT * localT
	return c[0].Mul(localT2 * localT).Add(c[1].Mul(localT2)).Add(c[2].Mul(localT)).Add(c[3])
}

// CatmullRomCurve is a curve from a Catmull-Rom spline through the control
// points, which begins at (0, 0) and ends at (1, 1); each x of the control
// points must be between 0 and 1 and increase.
//
// Flutter: animation/curves.dart CatmullRomCurve
type CatmullRomCurve struct {
	// ControlPoints are the points that the curve passes through, between (0, 0)
	// and (1, 1).
	ControlPoints []ui.Offset
	// Tension is how tightly the curve bends, from 0 to 1.
	Tension float64

	precomputed []Curve2DSample
}

func assertCatmullRomPoints(points []ui.Offset, tension float64) {
	if !foundation.DebugMode {
		return
	}
	var reasons []string
	if !CatmullRomCurveValidateControlPoints(points, tension, &reasons) {
		panic(errors.New("CatmullRomCurveInvalid", fmt.Sprintf("control points %v could not be validated:\n  %s", points, strings.Join(reasons, "\n  ")),
			"reasons", strings.Join(reasons, "; ")))
	}
}

// NewCatmullRomCurve creates the curve; its samples are computed when it is
// first used. In debug builds it panics with the tag CatmullRomCurveInvalid when
// the control points are not valid (see CatmullRomCurveValidateControlPoints).
//
// Flutter: animation/curves.dart CatmullRomCurve.new
func NewCatmullRomCurve(controlPoints []ui.Offset, tension ...float64) *CatmullRomCurve {
	c := &CatmullRomCurve{ControlPoints: controlPoints}
	if len(tension) > 0 {
		c.Tension = tension[0]
	}
	assertCatmullRomPoints(controlPoints, c.Tension)
	return c
}

// NewCatmullRomCurvePrecompute creates the curve and computes its samples at
// once.
//
// Flutter: animation/curves.dart CatmullRomCurve.precompute
func NewCatmullRomCurvePrecompute(controlPoints []ui.Offset, tension ...float64) *CatmullRomCurve {
	c := NewCatmullRomCurve(controlPoints, tension...)
	c.precomputed = catmullRomSamples(controlPoints, c.Tension)
	return c
}

func catmullRomSamples(controlPoints []ui.Offset, tension float64) []Curve2DSample {
	points := append(append([]ui.Offset{{}}, controlPoints...), ui.Offset{Dx: 1, Dy: 1})
	return Curve2DGenerateSamples(NewCatmullRomSplinePrecompute(points, CatmullRomSplineOpts{Tension: tension}),
		Curve2DGenerateSamplesOpts{Tolerance: 1e-12})
}

// CatmullRomCurveValidateControlPoints reports whether the control points make
// a valid curve: at least two, with x strictly between 0 and 1 and increasing,
// and a curve with a single y for each x. When reasons is not nil, it gets a
// text for each problem found.
//
// Flutter: animation/curves.dart CatmullRomCurve.validateControlPoints
func CatmullRomCurveValidateControlPoints(controlPoints []ui.Offset, tension float64, reasons *[]string) bool {
	add := func(s string) {
		if reasons != nil {
			*reasons = append(*reasons, s)
		}
	}
	if len(controlPoints) < 2 {
		add("There must be at least two points supplied to create a valid curve.")
		return false
	}
	points := append(append([]ui.Offset{{}}, controlPoints...), ui.Offset{Dx: 1, Dy: 1})
	startHandle := points[0].Mul(2).Sub(points[1])
	endHandle := points[len(points)-1].Mul(2).Sub(points[len(points)-2])
	points = append(append([]ui.Offset{startHandle}, points...), endHandle)
	lastX := math.Inf(-1)
	for i, p := range points {
		if i > 1 && i < len(points)-2 && (p.Dx <= 0 || p.Dx >= 1) {
			add(fmt.Sprintf("Control points must have X values between 0.0 and 1.0, exclusive. Point %d has an x value (%v) which is outside the range.", i, p.Dx))
			return false
		}
		if p.Dx <= lastX {
			add(fmt.Sprintf("Each X coordinate must be greater than the preceding X coordinate (i.e. must be monotonically increasing in X). Point %d has an x value of %v, which is not greater than %v", i, p.Dx, lastX))
			return false
		}
		lastX = p.Dx
	}
	success := true
	lastX = math.Inf(-1)
	const tolerance = 1e-3
	test := NewCatmullRomSpline(points, CatmullRomSplineOpts{Tension: tension})
	start, end := Curve2DFindInverse(test, 0), Curve2DFindInverse(test, 1)
	samples := Curve2DGenerateSamples(test, Curve2DGenerateSamplesOpts{Start: start, End: end})
	if math.Abs(samples[0].Value.Dy) > tolerance || math.Abs(1-samples[len(samples)-1].Value.Dy) > tolerance {
		success = false
		add(fmt.Sprintf("The curve has more than one Y value at X = %v. Try moving some control points further away from this value of X, or increasing the tension.", samples[0].Value.Dx))
		if reasons == nil {
			return false
		}
	}
	for _, s := range samples {
		x := s.Value.Dx
		if s.T >= start && s.T <= end && (x < -1e-3 || x > 1+1e-3) {
			success = false
			add(fmt.Sprintf("The resulting curve has an X value (%v) which is outside the range [0.0, 1.0], inclusive.", x))
			if reasons == nil {
				return false
			}
		}
		if x < lastX {
			success = false
			add(fmt.Sprintf("The curve has more than one Y value at x = %v. Try moving some control points further apart in X, or increasing the tension.", x))
			if reasons == nil {
				return false
			}
		}
		lastX = x
	}
	return success
}

func (c *CatmullRomCurve) transformInternal(t float64) float64 {
	if len(c.precomputed) == 0 {
		c.precomputed = catmullRomSamples(c.ControlPoints, c.Tension)
	}
	start, end := 0, len(c.precomputed)-1
	startValue, endValue := c.precomputed[start].Value, c.precomputed[end].Value
	for end-start > 1 {
		mid := (end + start) / 2
		value := c.precomputed[mid].Value
		if t >= value.Dx {
			start, startValue = mid, value
		} else {
			end, endValue = mid, value
		}
	}
	t2 := (t - startValue.Dx) / (endValue.Dx - startValue.Dx)
	return ui.LerpDouble(startValue.Dy, endValue.Dy, t2)
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart CatmullRomCurve.transformInternal
func (c *CatmullRomCurve) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c *CatmullRomCurve) Flipped() Curve { return FlippedCurve{c} }

// FlippedCurve is a curve that is the reversed inversion of another: 1 - curve(1
// - t). It makes an ease out from an ease in.
//
// Flutter: animation/curves.dart FlippedCurve
type FlippedCurve struct{ Curve Curve }

// NewFlippedCurve creates the curve.
//
// Flutter: animation/curves.dart FlippedCurve.new
func NewFlippedCurve(curve Curve) FlippedCurve { return FlippedCurve{curve} }

func (c FlippedCurve) transformInternal(t float64) float64 { return 1 - c.Curve.Transform(1-t) }

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart FlippedCurve.transformInternal
func (c FlippedCurve) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve that was flipped.
//
// Flutter: animation/curves.dart Curve.flipped
func (c FlippedCurve) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart FlippedCurve.toString
func (c FlippedCurve) String() string { return fmt.Sprintf("FlippedCurve(%v)", c.Curve) }

type decelerateCurve struct{}

func (decelerateCurve) transformInternal(t float64) float64 { t = 1 - t; return 1 - t*t }
func (c decelerateCurve) Transform(t float64) float64       { return transformCurve(c, t) }
func (c decelerateCurve) Flipped() Curve                    { return FlippedCurve{c} }
func (decelerateCurve) String() string                      { return "Decelerate" }

func bounce(t float64) float64 {
	switch {
	case t < 1/2.75:
		return 7.5625 * t * t
	case t < 2/2.75:
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	case t < 2.5/2.75:
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	}
	t -= 2.625 / 2.75
	return 7.5625*t*t + 0.984375
}

type bounceInCurve struct{}

func (bounceInCurve) transformInternal(t float64) float64 { return 1 - bounce(1-t) }
func (c bounceInCurve) Transform(t float64) float64       { return transformCurve(c, t) }
func (c bounceInCurve) Flipped() Curve                    { return FlippedCurve{c} }
func (bounceInCurve) String() string                      { return "BounceIn" }

type bounceOutCurve struct{}

func (bounceOutCurve) transformInternal(t float64) float64 { return bounce(t) }
func (c bounceOutCurve) Transform(t float64) float64       { return transformCurve(c, t) }
func (c bounceOutCurve) Flipped() Curve                    { return FlippedCurve{c} }
func (bounceOutCurve) String() string                      { return "BounceOut" }

type bounceInOutCurve struct{}

func (bounceInOutCurve) transformInternal(t float64) float64 {
	if t < 0.5 {
		return (1 - bounce(1-t*2)) * 0.5
	}
	return bounce(t*2-1)*0.5 + 0.5
}
func (c bounceInOutCurve) Transform(t float64) float64 { return transformCurve(c, t) }
func (c bounceInOutCurve) Flipped() Curve              { return FlippedCurve{c} }
func (bounceInOutCurve) String() string                { return "BounceInOut" }

// ElasticInCurve is an oscillating curve that grows in magnitude.
//
// Flutter: animation/curves.dart ElasticInCurve
type ElasticInCurve struct{ Period float64 }

// NewElasticInCurve creates the curve; the period (the default is 0.4) is the
// duration of each oscillation.
//
// Flutter: animation/curves.dart ElasticInCurve.new
func NewElasticInCurve(period ...float64) ElasticInCurve { return ElasticInCurve{periodOf(period)} }

func periodOf(p []float64) float64 {
	if len(p) > 0 {
		return p[0]
	}
	return 0.4
}

func (c ElasticInCurve) transformInternal(t float64) float64 {
	s := c.Period / 4
	t--
	return -math.Pow(2, 10*t) * math.Sin((t-s)*(math.Pi*2)/c.Period)
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart ElasticInCurve.transformInternal
func (c ElasticInCurve) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c ElasticInCurve) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart ElasticInCurve.toString
func (c ElasticInCurve) String() string { return fmt.Sprintf("ElasticInCurve(%v)", c.Period) }

// ElasticOutCurve is an oscillating curve that shrinks in magnitude while
// overshooting its bounds.
//
// Flutter: animation/curves.dart ElasticOutCurve
type ElasticOutCurve struct{ Period float64 }

// NewElasticOutCurve creates the curve; the period (the default is 0.4) is the
// duration of each oscillation.
//
// Flutter: animation/curves.dart ElasticOutCurve.new
func NewElasticOutCurve(period ...float64) ElasticOutCurve { return ElasticOutCurve{periodOf(period)} }

func (c ElasticOutCurve) transformInternal(t float64) float64 {
	s := c.Period / 4
	return math.Pow(2, -10*t)*math.Sin((t-s)*(math.Pi*2)/c.Period) + 1
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart ElasticOutCurve.transformInternal
func (c ElasticOutCurve) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c ElasticOutCurve) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart ElasticOutCurve.toString
func (c ElasticOutCurve) String() string { return fmt.Sprintf("ElasticOutCurve(%v)", c.Period) }

// ElasticInOutCurve is an oscillating curve that grows and then shrinks in
// magnitude.
//
// Flutter: animation/curves.dart ElasticInOutCurve
type ElasticInOutCurve struct{ Period float64 }

// NewElasticInOutCurve creates the curve; the period (the default is 0.4) is
// the duration of each oscillation.
//
// Flutter: animation/curves.dart ElasticInOutCurve.new
func NewElasticInOutCurve(period ...float64) ElasticInOutCurve {
	return ElasticInOutCurve{periodOf(period)}
}

func (c ElasticInOutCurve) transformInternal(t float64) float64 {
	s := c.Period / 4
	t = 2*t - 1
	if t < 0 {
		return -0.5 * math.Pow(2, 10*t) * math.Sin((t-s)*(math.Pi*2)/c.Period)
	}
	return math.Pow(2, -10*t)*math.Sin((t-s)*(math.Pi*2)/c.Period)*0.5 + 1
}

// Transform returns the value of the curve at t.
//
// Flutter: animation/curves.dart ElasticInOutCurve.transformInternal
func (c ElasticInOutCurve) Transform(t float64) float64 { return transformCurve(c, t) }

// Flipped is the curve mirrored on both axes.
//
// Flutter: animation/curves.dart Curve.flipped
func (c ElasticInOutCurve) Flipped() Curve { return FlippedCurve{c} }

// String describes the curve.
//
// Flutter: animation/curves.dart ElasticInOutCurve.toString
func (c ElasticInOutCurve) String() string { return fmt.Sprintf("ElasticInOutCurve(%v)", c.Period) }

// CurvesCollection holds the standard curves. The one value of it is Curves; it
// is read-only.
//
// Flutter: animation/curves.dart Curves
type CurvesCollection struct {
	// Linear is a linear animation curve: the identity.
	Linear Curve
	// Decelerate is a curve that is shaped like a quadratic ease out.
	Decelerate Curve
	// FastLinearToSlowEaseIn is a cubic curve that starts fast and ends slowly.
	FastLinearToSlowEaseIn Cubic
	// FastEaseInToSlowEaseOut is a curve of two cubics, fast in and slow out.
	FastEaseInToSlowEaseOut ThreePointCubic
	// Ease is a cubic animation curve that speeds up quickly and ends slowly.
	Ease Cubic
	// EaseIn is a cubic animation curve that starts slowly and ends quickly.
	EaseIn Cubic
	// EaseInToLinear is a cubic that starts slowly and ends linearly.
	EaseInToLinear Cubic
	// EaseInSine is a cubic that starts slowly (sinusoidal).
	EaseInSine Cubic
	// EaseInQuad is a cubic that starts slowly (quadratic).
	EaseInQuad Cubic
	// EaseInCubic is a cubic that starts slowly (cubic).
	EaseInCubic Cubic
	// EaseInQuart is a cubic that starts slowly (quartic).
	EaseInQuart Cubic
	// EaseInQuint is a cubic that starts slowly (quintic).
	EaseInQuint Cubic
	// EaseInExpo is a cubic that starts slowly (exponential).
	EaseInExpo Cubic
	// EaseInCirc is a cubic that starts slowly (circular).
	EaseInCirc Cubic
	// EaseInBack is a cubic that starts by going backwards.
	EaseInBack Cubic
	// EaseOut is a cubic that starts quickly and ends slowly.
	EaseOut Cubic
	// LinearToEaseOut is a cubic that starts linearly and ends slowly.
	LinearToEaseOut Cubic
	// EaseOutSine is a cubic that ends slowly (sinusoidal).
	EaseOutSine Cubic
	// EaseOutQuad is a cubic that ends slowly (quadratic).
	EaseOutQuad Cubic
	// EaseOutCubic is a cubic that ends slowly (cubic).
	EaseOutCubic Cubic
	// EaseOutQuart is a cubic that ends slowly (quartic).
	EaseOutQuart Cubic
	// EaseOutQuint is a cubic that ends slowly (quintic).
	EaseOutQuint Cubic
	// EaseOutExpo is a cubic that ends slowly (exponential).
	EaseOutExpo Cubic
	// EaseOutCirc is a cubic that ends slowly (circular).
	EaseOutCirc Cubic
	// EaseOutBack is a cubic that overshoots its end and comes back.
	EaseOutBack Cubic
	// EaseInOut is a cubic that starts and ends slowly.
	EaseInOut Cubic
	// EaseInOutSine is a cubic that starts and ends slowly (sinusoidal).
	EaseInOutSine Cubic
	// EaseInOutQuad is a cubic that starts and ends slowly (quadratic).
	EaseInOutQuad Cubic
	// EaseInOutCubic is a cubic that starts and ends slowly (cubic).
	EaseInOutCubic Cubic
	// EaseInOutCubicEmphasized is a curve of two cubics with an emphasized
	// middle.
	EaseInOutCubicEmphasized ThreePointCubic
	// EaseInOutQuart is a cubic that starts and ends slowly (quartic).
	EaseInOutQuart Cubic
	// EaseInOutQuint is a cubic that starts and ends slowly (quintic).
	EaseInOutQuint Cubic
	// EaseInOutExpo is a cubic that starts and ends slowly (exponential).
	EaseInOutExpo Cubic
	// EaseInOutCirc is a cubic that starts and ends slowly (circular).
	EaseInOutCirc Cubic
	// EaseInOutBack is a cubic that goes back at the start and overshoots at the
	// end.
	EaseInOutBack Cubic
	// FastOutSlowIn is the standard Material motion curve: fast out, slow in.
	FastOutSlowIn Cubic
	// SlowMiddle is a cubic that is slow in the middle.
	SlowMiddle Cubic
	// BounceIn is an oscillating curve that grows in magnitude.
	BounceIn Curve
	// BounceOut is an oscillating curve that shrinks in magnitude.
	BounceOut Curve
	// BounceInOut is an oscillating curve that first grows and then shrinks.
	BounceInOut Curve
	// ElasticIn is an oscillating curve that grows in magnitude.
	ElasticIn ElasticInCurve
	// ElasticOut is an oscillating curve that shrinks in magnitude.
	ElasticOut ElasticOutCurve
	// ElasticInOut is an oscillating curve that grows and then shrinks.
	ElasticInOut ElasticInOutCurve
}

// Curves is the collection of the standard curves.
//
// Flutter: animation/curves.dart Curves
var Curves = CurvesCollection{
	Linear:                  linearCurve{},
	Decelerate:              decelerateCurve{},
	FastLinearToSlowEaseIn:  Cubic{0.18, 1.0, 0.04, 1.0},
	FastEaseInToSlowEaseOut: ThreePointCubic{ui.Offset{Dx: 0.056, Dy: 0.024}, ui.Offset{Dx: 0.108, Dy: 0.3085}, ui.Offset{Dx: 0.198, Dy: 0.541}, ui.Offset{Dx: 0.3655, Dy: 1.0}, ui.Offset{Dx: 0.5465, Dy: 0.989}},
	Ease:                    Cubic{0.25, 0.1, 0.25, 1.0},
	EaseIn:                  Cubic{0.42, 0.0, 1.0, 1.0},
	EaseInToLinear:          Cubic{0.67, 0.03, 0.65, 0.09},
	EaseInSine:              Cubic{0.47, 0.0, 0.745, 0.715},
	EaseInQuad:              Cubic{0.55, 0.085, 0.68, 0.53},
	EaseInCubic:             Cubic{0.55, 0.055, 0.675, 0.19},
	EaseInQuart:             Cubic{0.895, 0.03, 0.685, 0.22},
	EaseInQuint:             Cubic{0.755, 0.05, 0.855, 0.06},
	EaseInExpo:              Cubic{0.95, 0.05, 0.795, 0.035},
	EaseInCirc:              Cubic{0.6, 0.04, 0.98, 0.335},
	EaseInBack:              Cubic{0.6, -0.28, 0.735, 0.045},
	EaseOut:                 Cubic{0.0, 0.0, 0.58, 1.0},
	LinearToEaseOut:         Cubic{0.35, 0.91, 0.33, 0.97},
	EaseOutSine:             Cubic{0.39, 0.575, 0.565, 1.0},
	EaseOutQuad:             Cubic{0.25, 0.46, 0.45, 0.94},
	EaseOutCubic:            Cubic{0.215, 0.61, 0.355, 1.0},
	EaseOutQuart:            Cubic{0.165, 0.84, 0.44, 1.0},
	EaseOutQuint:            Cubic{0.23, 1.0, 0.32, 1.0},
	EaseOutExpo:             Cubic{0.19, 1.0, 0.22, 1.0},
	EaseOutCirc:             Cubic{0.075, 0.82, 0.165, 1.0},
	EaseOutBack:             Cubic{0.175, 0.885, 0.32, 1.275},
	EaseInOut:               Cubic{0.42, 0.0, 0.58, 1.0},
	EaseInOutSine:           Cubic{0.445, 0.05, 0.55, 0.95},
	EaseInOutQuad:           Cubic{0.455, 0.03, 0.515, 0.955},
	EaseInOutCubic:          Cubic{0.645, 0.045, 0.355, 1.0},
	EaseInOutCubicEmphasized: ThreePointCubic{ui.Offset{Dx: 0.05, Dy: 0}, ui.Offset{Dx: 0.133333, Dy: 0.06}, ui.Offset{Dx: 0.166666, Dy: 0.4},
		ui.Offset{Dx: 0.208333, Dy: 0.82}, ui.Offset{Dx: 0.25, Dy: 1}},
	EaseInOutQuart: Cubic{0.77, 0.0, 0.175, 1.0},
	EaseInOutQuint: Cubic{0.86, 0.0, 0.07, 1.0},
	EaseInOutExpo:  Cubic{1.0, 0.0, 0.0, 1.0},
	EaseInOutCirc:  Cubic{0.785, 0.135, 0.15, 0.86},
	EaseInOutBack:  Cubic{0.68, -0.55, 0.265, 1.55},
	FastOutSlowIn:  Cubic{0.4, 0.0, 0.2, 1.0},
	SlowMiddle:     Cubic{0.15, 0.85, 0.85, 0.15},
	BounceIn:       bounceInCurve{},
	BounceOut:      bounceOutCurve{},
	BounceInOut:    bounceInOutCurve{},
	ElasticIn:      ElasticInCurve{0.4},
	ElasticOut:     ElasticOutCurve{0.4},
	ElasticInOut:   ElasticInOutCurve{0.4},
}
