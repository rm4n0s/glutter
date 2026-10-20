// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package physics

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// newtonsMethod finds an x with f(x) = target, starting at initialGuess, by
// iterating Newton's method with the derivative df.
func newtonsMethod(initialGuess, target float64, f, df func(float64) float64, iterations int) float64 {
	guess := initialGuess
	for i := 0; i < iterations; i++ {
		guess = guess - (f(guess)-target)/df(guess)
	}
	return guess
}

// FrictionSimulation is a simulation of a particle that slows down under
// friction (drag) and, optionally, a constant deceleration.
//
// Flutter: physics/friction_simulation.dart FrictionSimulation
type FrictionSimulation struct {
	SimulationBase
	drag, dragLog        float64
	x0, v                float64
	constantDeceleration float64
	// finalTime is the time at which the simulation should be stopped. It is
	// needed when constantDeceleration is not zero (on desktop); with pure
	// friction the velocity reaches zero only in the limit.
	finalTime float64
	// x is the position function that timeAtX and finalX use. It is the
	// simulation's own X, or the bounded one for a BoundedFrictionSimulation.
	xFunc func(float64) float64
}

// FrictionSimulationOpts are the optional parameters of NewFrictionSimulation.
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.new
type FrictionSimulationOpts struct {
	// Tolerance decides when the simulation is done; the default tolerance if
	// not set.
	Tolerance foundation.Opt[Tolerance]
	// ConstantDeceleration is a deceleration that is added to the friction, in
	// the opposite direction to the velocity; 0 by default.
	ConstantDeceleration float64
}

// NewFrictionSimulation creates a simulation of a particle at position with the
// velocity, slowed by drag, which is the fraction of the velocity that is left
// after one second (a value between 0 and 1).
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.new
func NewFrictionSimulation(drag, position, velocity float64, opts ...FrictionSimulationOpts) *FrictionSimulation {
	o := firstOpts(opts)
	f := &FrictionSimulation{
		SimulationBase:       NewSimulationBase(o.Tolerance),
		drag:                 drag,
		dragLog:              math.Log(drag),
		x0:                   position,
		v:                    velocity,
		constantDeceleration: o.ConstantDeceleration * sign(velocity),
		finalTime:            math.Inf(1), // needs to be infinity for the newtonsMethod call below
	}
	f.xFunc = f.X
	f.finalTime = newtonsMethod(0, 0, f.Dx,
		func(time float64) float64 { return f.v*math.Pow(f.drag, time)*f.dragLog - f.constantDeceleration }, 10)
	return f
}

// NewFrictionSimulationThrough creates a simulation whose position and velocity
// at some time pass through endPosition and endVelocity, having started from
// startPosition with startVelocity: the drag is found from the four values, and
// the simulation is done when the velocity reaches endVelocity. The velocities
// must have the same direction (or one is zero), the start velocity must be at
// least as fast as the end velocity, and the end position must lie in the
// direction of the start velocity; otherwise it panics with the tag
// FrictionThroughInvalid in debug builds.
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.through
func NewFrictionSimulationThrough(startPosition, endPosition, startVelocity, endVelocity float64) *FrictionSimulation {
	if foundation.DebugMode {
		ok := (startVelocity == 0 || endVelocity == 0 || sign(startVelocity) == sign(endVelocity)) &&
			math.Abs(startVelocity) >= math.Abs(endVelocity) &&
			sign(endPosition-startPosition) == sign(startVelocity)
		if !ok {
			panic(errors.New("FrictionThroughInvalid", "the start and end values do not describe a slowing particle",
				"startPosition", startPosition, "endPosition", endPosition,
				"startVelocity", startVelocity, "endVelocity", endVelocity))
		}
	}
	return NewFrictionSimulation(dragFor(startPosition, endPosition, startVelocity, endVelocity), startPosition, startVelocity,
		FrictionSimulationOpts{Tolerance: foundation.Some(NewTolerance(ToleranceOpts{Velocity: foundation.Some(math.Abs(endVelocity))}))})
}

// dragFor returns the drag value for a FrictionSimulation whose X and Dx pass
// through the given start and end position and velocity values.
//
// The total time to reach endVelocity is (log(endVelocity) / log(startVelocity))
// / log(drag), or (log(v1) - log(v0)) / log(D), given v = v0 * D^t per the Dx
// function. Solving for D given x(time) is trickier; the algebra gives
// x1 = x0 + (v0 * D^((log(v1) - log(v0)) / log(D))) / log(D) - v0 / log(D).
func dragFor(startPosition, endPosition, startVelocity, endVelocity float64) float64 {
	return math.Pow(math.E, (startVelocity-endVelocity)/(startPosition-endPosition))
}

// sign is Dart's num.sign: -1, 0 or 1 (NaN stays NaN).
func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return x
}

// X is the position at the time.
func (f *FrictionSimulation) X(time float64) float64 {
	if time > f.finalTime {
		return f.FinalX()
	}
	return f.x0 + f.v*math.Pow(f.drag, time)/f.dragLog - f.v/f.dragLog - (f.constantDeceleration/2)*time*time
}

// Dx is the velocity at the time.
func (f *FrictionSimulation) Dx(time float64) float64 {
	if time > f.finalTime {
		return 0
	}
	return f.v*math.Pow(f.drag, time) - f.constantDeceleration*time
}

// FinalX is the value X approaches: where the particle stops.
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.finalX
func (f *FrictionSimulation) FinalX() float64 {
	if f.constantDeceleration == 0 {
		return f.x0 - f.v/f.dragLog
	}
	return f.xFunc(f.finalTime)
}

// TimeAtX is the time at which the particle is at x, or positive infinity if it
// never is. Time 0 is the start.
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.timeAtX
func (f *FrictionSimulation) TimeAtX(x float64) float64 {
	if x == f.x0 {
		return 0
	}
	if f.v == 0 {
		return math.Inf(1)
	}
	final := f.FinalX()
	if f.v > 0 && (x < f.x0 || x > final) || f.v < 0 && (x > f.x0 || x < final) {
		return math.Inf(1)
	}
	return newtonsMethod(0, x, f.xFunc, f.Dx, 10)
}

// IsDone is whether the velocity is below the velocity tolerance.
func (f *FrictionSimulation) IsDone(time float64) bool {
	return math.Abs(f.Dx(time)) < f.Tolerance().Velocity
}

// String describes the simulation, such as
// "FrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0)".
//
// Flutter: physics/friction_simulation.dart FrictionSimulation.toString
func (f *FrictionSimulation) String() string {
	return "FrictionSimulation(cₓ: " + dartfmt.ToStringAsFixed(f.drag, 1) + ", x₀: " + dartfmt.ToStringAsFixed(f.x0, 1) +
		", dx₀: " + dartfmt.ToStringAsFixed(f.v, 1) + ")"
}

// BoundedFrictionSimulation is a FrictionSimulation whose position stays
// between two bounds, and which is done when it reaches one.
//
// Flutter: physics/friction_simulation.dart BoundedFrictionSimulation
type BoundedFrictionSimulation struct {
	*FrictionSimulation
	minX, maxX float64
}

// NewBoundedFrictionSimulation creates a friction simulation that cannot leave
// minX..maxX. A position outside the bounds panics with the tag
// FrictionPositionOutOfBounds in debug builds.
//
// Flutter: physics/friction_simulation.dart BoundedFrictionSimulation.new
func NewBoundedFrictionSimulation(drag, position, velocity, minX, maxX float64) *BoundedFrictionSimulation {
	if foundation.DebugMode && ui.ClampDouble(position, minX, maxX) != position {
		panic(errors.New("FrictionPositionOutOfBounds", "the start position must be between the bounds",
			"position", position, "minX", minX, "maxX", maxX))
	}
	b := &BoundedFrictionSimulation{FrictionSimulation: NewFrictionSimulation(drag, position, velocity), minX: minX, maxX: maxX}
	b.xFunc = b.X
	return b
}

// X is the position at the time, kept between the bounds.
func (b *BoundedFrictionSimulation) X(time float64) float64 {
	return ui.ClampDouble(b.FrictionSimulation.X(time), b.minX, b.maxX)
}

// FinalX is the position where the particle stops, kept between the bounds
// when there is a constant deceleration.
func (b *BoundedFrictionSimulation) FinalX() float64 {
	if b.constantDeceleration == 0 {
		return b.x0 - b.v/b.dragLog
	}
	return b.X(b.finalTime)
}

// TimeAtX is the time at which the particle is at x, or positive infinity.
func (b *BoundedFrictionSimulation) TimeAtX(x float64) float64 {
	if x == b.x0 {
		return 0
	}
	if b.v == 0 {
		return math.Inf(1)
	}
	final := b.FinalX()
	if b.v > 0 && (x < b.x0 || x > final) || b.v < 0 && (x > b.x0 || x < final) {
		return math.Inf(1)
	}
	return newtonsMethod(0, x, b.X, b.Dx, 10)
}

// IsDone is whether the velocity is below the tolerance, or the particle is
// at a bound.
func (b *BoundedFrictionSimulation) IsDone(time float64) bool {
	tol := b.Tolerance()
	return b.FrictionSimulation.IsDone(time) ||
		math.Abs(b.X(time)-b.minX) < tol.Distance ||
		math.Abs(b.X(time)-b.maxX) < tol.Distance
}

// String describes the simulation, such as
// "BoundedFrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0, x: -4.0..5.0)".
//
// Flutter: physics/friction_simulation.dart BoundedFrictionSimulation.toString
func (b *BoundedFrictionSimulation) String() string {
	return "BoundedFrictionSimulation(cₓ: " + dartfmt.ToStringAsFixed(b.drag, 1) + ", x₀: " + dartfmt.ToStringAsFixed(b.x0, 1) +
		", dx₀: " + dartfmt.ToStringAsFixed(b.v, 1) + ", x: " + dartfmt.ToStringAsFixed(b.minX, 1) + ".." +
		dartfmt.ToStringAsFixed(b.maxX, 1) + ")"
}
