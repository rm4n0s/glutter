// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package physics

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
)

// SpringDescription is the mass, stiffness and damping of a spring.
//
// Flutter: physics/spring_simulation.dart SpringDescription
type SpringDescription struct {
	// Mass is the mass of the spring (m).
	//
	// Flutter: physics/spring_simulation.dart SpringDescription.mass
	Mass float64
	// Stiffness is the spring constant (k): the force per unit of stretch.
	//
	// Flutter: physics/spring_simulation.dart SpringDescription.stiffness
	Stiffness float64
	// Damping is the coefficient of the force that slows the spring (c).
	//
	// Flutter: physics/spring_simulation.dart SpringDescription.damping
	Damping float64
}

// NewSpringDescription creates a spring description from its three values.
//
// Flutter: physics/spring_simulation.dart SpringDescription.new
func NewSpringDescription(mass, stiffness, damping float64) SpringDescription {
	return SpringDescription{Mass: mass, Stiffness: stiffness, Damping: damping}
}

// SpringDescriptionWithDampingRatioOpts are the optional parameters of
// NewSpringDescriptionWithDampingRatio.
//
// Flutter: physics/spring_simulation.dart SpringDescription.withDampingRatio
type SpringDescriptionWithDampingRatioOpts struct {
	// Ratio is the damping ratio: 1 is critically damped (the default), below
	// 1 is underdamped and above 1 is overdamped.
	Ratio foundation.Opt[float64]
}

// NewSpringDescriptionWithDampingRatio creates a spring description from the
// mass, the stiffness and a damping ratio; the damping is ratio * 2 *
// sqrt(mass * stiffness).
//
// Flutter: physics/spring_simulation.dart SpringDescription.withDampingRatio
func NewSpringDescriptionWithDampingRatio(mass, stiffness float64, opts ...SpringDescriptionWithDampingRatioOpts) SpringDescription {
	o := firstOpts(opts)
	return SpringDescription{Mass: mass, Stiffness: stiffness, Damping: o.Ratio.OrElse(1) * 2 * math.Sqrt(mass*stiffness)}
}

// SpringDescriptionWithDurationAndBounceOpts are the optional parameters of
// NewSpringDescriptionWithDurationAndBounce.
//
// Flutter: physics/spring_simulation.dart SpringDescription.withDurationAndBounce
type SpringDescriptionWithDurationAndBounceOpts struct {
	// Duration is the perceptual duration of the spring, the time it takes to
	// do one oscillation without damping; 500 ms by default. It must be longer
	// than a millisecond, or the constructor panics with the tag
	// SpringDurationInvalid in debug builds.
	Duration foundation.Opt[time.Duration]
	// Bounce is how bouncy the spring is: 0 is critically damped (no bounce),
	// positive values bounce, negative values are overdamped. 0 by default.
	Bounce float64
}

// NewSpringDescriptionWithDurationAndBounce creates a spring description from
// how long it takes and how much it bounces, with a mass of 1.
//
// Flutter: physics/spring_simulation.dart SpringDescription.withDurationAndBounce
func NewSpringDescriptionWithDurationAndBounce(opts ...SpringDescriptionWithDurationAndBounceOpts) SpringDescription {
	o := firstOpts(opts)
	duration := o.Duration.OrElse(500 * time.Millisecond)
	if foundation.DebugMode && duration.Milliseconds() <= 0 {
		panic(errors.New("SpringDurationInvalid", "Duration must be positive", "duration", duration.String()))
	}
	durationInSeconds := float64(duration.Milliseconds()) / 1000
	const mass = 1.0
	stiffness := (4 * math.Pi * math.Pi * mass) / (durationInSeconds * durationInSeconds)
	var dampingRatio float64
	if o.Bounce > 0 {
		dampingRatio = 1 - o.Bounce
	} else {
		dampingRatio = 1 / (o.Bounce + 1)
	}
	return SpringDescription{Mass: mass, Stiffness: stiffness, Damping: dampingRatio * 2 * math.Sqrt(mass*stiffness)}
}

// Duration is the perceptual duration of the spring, rounded to milliseconds.
//
// Flutter: physics/spring_simulation.dart SpringDescription.duration
func (s SpringDescription) Duration() time.Duration {
	durationInSeconds := math.Sqrt((4 * math.Pi * math.Pi * s.Mass) / s.Stiffness)
	return time.Duration(math.Round(durationInSeconds*1000)) * time.Millisecond
}

// Bounce is how bouncy the spring is: positive when underdamped, 0 when
// critically damped and negative when overdamped.
//
// Flutter: physics/spring_simulation.dart SpringDescription.bounce
func (s SpringDescription) Bounce() float64 {
	dampingRatio := s.Damping / (2 * math.Sqrt(s.Mass*s.Stiffness))
	if dampingRatio < 1 {
		return 1 - dampingRatio
	}
	return 1/dampingRatio - 1
}

// String describes the spring, such as
// "SpringDescription(mass: 1.0, stiffness: -2.0, damping: 3.0)".
//
// Flutter: physics/spring_simulation.dart SpringDescription.toString
func (s SpringDescription) String() string {
	return "SpringDescription(mass: " + dartfmt.ToStringAsFixed(s.Mass, 1) + ", stiffness: " +
		dartfmt.ToStringAsFixed(s.Stiffness, 1) + ", damping: " + dartfmt.ToStringAsFixed(s.Damping, 1) + ")"
}

// SpringType is how a spring settles, which its damping decides. The zero
// value is SpringTypeUnspecified.
//
// Flutter: physics/spring_simulation.dart SpringType
type SpringType int

// The values of SpringType follow Flutter's declaration order.
const (
	// SpringTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SpringTypeUnspecified SpringType = iota
	// SpringTypeCriticallyDamped is a spring that reaches its end as fast as
	// possible without overshooting.
	//
	// Flutter: physics/spring_simulation.dart SpringType.criticallyDamped
	SpringTypeCriticallyDamped
	// SpringTypeUnderDamped is a spring that overshoots its end and oscillates
	// around it.
	//
	// Flutter: physics/spring_simulation.dart SpringType.underDamped
	SpringTypeUnderDamped
	// SpringTypeOverDamped is a spring that reaches its end slowly, without
	// overshooting.
	//
	// Flutter: physics/spring_simulation.dart SpringType.overDamped
	SpringTypeOverDamped
)

var springTypeNames = [...]string{"unspecified", "criticallyDamped", "underDamped", "overDamped"}

// Name returns the name Dart's Enum.name gives the value, such as
// "underDamped".
func (t SpringType) Name() string {
	if t >= 0 && int(t) < len(springTypeNames) {
		return springTypeNames[t]
	}
	return "SpringType(" + strconv.Itoa(int(t)) + ")"
}

// String returns the name Dart prints for the value, such as
// "SpringType.underDamped".
func (t SpringType) String() string {
	if t >= 0 && int(t) < len(springTypeNames) {
		return "SpringType." + springTypeNames[t]
	}
	return t.Name()
}

// springSolution is the closed form of a spring: its position and velocity as
// functions of time, for the distance from the end.
type springSolution interface {
	x(time float64) float64
	dx(time float64) float64
	kind() SpringType
}

func newSpringSolution(spring SpringDescription, initialPosition, initialVelocity float64) springSolution {
	switch disc := spring.Damping*spring.Damping - 4*spring.Mass*spring.Stiffness; {
	case disc > 0:
		return newOverdamped(spring, initialPosition, initialVelocity)
	case disc < 0:
		return newUnderdamped(spring, initialPosition, initialVelocity)
	}
	return newCritical(spring, initialPosition, initialVelocity)
}

type criticalSolution struct{ r, c1, c2 float64 }

func newCritical(spring SpringDescription, distance, velocity float64) criticalSolution {
	r := -spring.Damping / (2 * spring.Mass)
	return criticalSolution{r: r, c1: distance, c2: velocity - r*distance}
}

func (s criticalSolution) x(time float64) float64 { return (s.c1 + s.c2*time) * math.Exp(s.r*time) }
func (s criticalSolution) dx(time float64) float64 {
	power := math.Exp(s.r * time)
	return s.r*(s.c1+s.c2*time)*power + s.c2*power
}
func (criticalSolution) kind() SpringType { return SpringTypeCriticallyDamped }

type overdampedSolution struct{ r1, r2, c1, c2 float64 }

func newOverdamped(spring SpringDescription, distance, velocity float64) overdampedSolution {
	cmk := spring.Damping*spring.Damping - 4*spring.Mass*spring.Stiffness
	r1 := (-spring.Damping - math.Sqrt(cmk)) / (2 * spring.Mass)
	r2 := (-spring.Damping + math.Sqrt(cmk)) / (2 * spring.Mass)
	c2 := (velocity - r1*distance) / (r2 - r1)
	return overdampedSolution{r1: r1, r2: r2, c1: distance - c2, c2: c2}
}

func (s overdampedSolution) x(time float64) float64 {
	return s.c1*math.Exp(s.r1*time) + s.c2*math.Exp(s.r2*time)
}
func (s overdampedSolution) dx(time float64) float64 {
	return s.c1*s.r1*math.Exp(s.r1*time) + s.c2*s.r2*math.Exp(s.r2*time)
}
func (overdampedSolution) kind() SpringType { return SpringTypeOverDamped }

type underdampedSolution struct{ w, r, c1, c2 float64 }

func newUnderdamped(spring SpringDescription, distance, velocity float64) underdampedSolution {
	w := math.Sqrt(4*spring.Mass*spring.Stiffness-spring.Damping*spring.Damping) / (2 * spring.Mass)
	r := -(spring.Damping / 2 / spring.Mass)
	return underdampedSolution{w: w, r: r, c1: distance, c2: (velocity - r*distance) / w}
}

func (s underdampedSolution) x(time float64) float64 {
	return math.Exp(s.r*time) * (s.c1*math.Cos(s.w*time) + s.c2*math.Sin(s.w*time))
}
func (s underdampedSolution) dx(time float64) float64 {
	power := math.Exp(s.r * time)
	cosine := math.Cos(s.w * time)
	sine := math.Sin(s.w * time)
	return power*(s.c2*s.w*cosine-s.c1*s.w*sine) + s.r*power*(s.c2*sine+s.c1*cosine)
}
func (underdampedSolution) kind() SpringType { return SpringTypeUnderDamped }

// SpringSimulation is a spring that pulls a particle to its end position: the
// particle starts at start with a velocity and settles at end.
//
// Flutter: physics/spring_simulation.dart SpringSimulation
type SpringSimulation struct {
	SimulationBase
	endPosition float64
	solution    springSolution
	snapToEnd   bool
}

// SpringSimulationOpts are the optional parameters of NewSpringSimulation.
//
// Flutter: physics/spring_simulation.dart SpringSimulation.new
type SpringSimulationOpts struct {
	// SnapToEnd makes X and Dx return exactly the end position and zero once
	// the simulation is done, instead of values within the tolerance.
	SnapToEnd bool
	// Tolerance decides when the simulation is done; the default tolerance if
	// not set.
	Tolerance foundation.Opt[Tolerance]
}

// NewSpringSimulation creates a spring simulation of a particle that starts at
// start with the velocity and is pulled to end by the spring.
//
// Flutter: physics/spring_simulation.dart SpringSimulation.new
func NewSpringSimulation(spring SpringDescription, start, end, velocity float64, opts ...SpringSimulationOpts) *SpringSimulation {
	o := firstOpts(opts)
	return &SpringSimulation{
		SimulationBase: NewSimulationBase(o.Tolerance),
		endPosition:    end,
		solution:       newSpringSolution(spring, start-end, velocity),
		snapToEnd:      o.SnapToEnd,
	}
}

// Type is how the spring settles.
//
// Flutter: physics/spring_simulation.dart SpringSimulation.type
func (s *SpringSimulation) Type() SpringType { return s.solution.kind() }

// X is the position at the time.
func (s *SpringSimulation) X(time float64) float64 {
	if s.snapToEnd && s.IsDone(time) {
		return s.endPosition
	}
	return s.endPosition + s.solution.x(time)
}

// Dx is the velocity at the time.
func (s *SpringSimulation) Dx(time float64) float64 {
	if s.snapToEnd && s.IsDone(time) {
		return 0
	}
	return s.solution.dx(time)
}

// IsDone is whether the particle is at its end, at rest, within the tolerance.
func (s *SpringSimulation) IsDone(time float64) bool {
	tol := s.Tolerance()
	return NearZero(s.solution.x(time), tol.Distance) && NearZero(s.solution.dx(time), tol.Velocity)
}

// String describes the simulation, such as
// "SpringSimulation(end: 1.0, SpringType.overDamped)".
//
// Flutter: physics/spring_simulation.dart SpringSimulation.toString
func (s *SpringSimulation) String() string {
	return "SpringSimulation(end: " + dartfmt.ToStringAsFixed(s.endPosition, 1) + ", " + s.Type().String() + ")"
}

// ScrollSpringSimulation is a SpringSimulation that, when it is done, is exactly
// at its end position. Scrolling uses it so that a scroll position settles on
// the edge.
//
// Flutter: physics/spring_simulation.dart ScrollSpringSimulation
type ScrollSpringSimulation struct {
	*SpringSimulation
}

// NewScrollSpringSimulation creates a scroll spring simulation. The options are
// those of NewSpringSimulation, and SnapToEnd has no effect.
//
// Flutter: physics/spring_simulation.dart ScrollSpringSimulation.new
func NewScrollSpringSimulation(spring SpringDescription, start, end, velocity float64, opts ...SpringSimulationOpts) *ScrollSpringSimulation {
	return &ScrollSpringSimulation{NewSpringSimulation(spring, start, end, velocity, opts...)}
}

// X is the position at the time, exactly the end once done.
func (s *ScrollSpringSimulation) X(time float64) float64 {
	if s.IsDone(time) {
		return s.endPosition
	}
	return s.SpringSimulation.X(time)
}
