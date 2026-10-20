// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package physics

import (
	"github.com/rm4n0s/glutter/foundation"
)

// Simulation is the base interface for a physics simulation that works in
// terms of a one dimensional position and velocity as functions of time, in
// seconds.
//
// A simulation of your own embeds SimulationBase for the tolerance, and
// implements X, Dx and IsDone.
//
// Flutter: physics/simulation.dart Simulation
type Simulation interface {
	// X is the position of the object in the simulation at the given time.
	//
	// Flutter: physics/simulation.dart Simulation.x
	X(time float64) float64
	// Dx is the velocity of the object in the simulation at the given time.
	//
	// Flutter: physics/simulation.dart Simulation.dx
	Dx(time float64) float64
	// IsDone reports whether the simulation is "done" at the given time.
	//
	// Flutter: physics/simulation.dart Simulation.isDone
	IsDone(time float64) bool
	// Tolerance is how close a position, velocity or time must be to count as
	// the end of the simulation.
	//
	// Flutter: physics/simulation.dart Simulation.tolerance
	Tolerance() Tolerance
	// SetTolerance changes the tolerance.
	//
	// Flutter: physics/simulation.dart Simulation.tolerance=
	SetTolerance(Tolerance)
}

// SimulationBase holds the tolerance of a Simulation. Embed it.
//
// Flutter: physics/simulation.dart Simulation
type SimulationBase struct {
	tolerance foundation.Opt[Tolerance]
}

// NewSimulationBase creates the base with a tolerance, or the default one for
// None.
//
// Flutter: physics/simulation.dart Simulation.new
func NewSimulationBase(tolerance foundation.Opt[Tolerance]) SimulationBase {
	return SimulationBase{tolerance: tolerance}
}

// Tolerance returns the tolerance, ToleranceDefault until it is set.
func (s *SimulationBase) Tolerance() Tolerance { return s.tolerance.OrElse(ToleranceDefault) }

// SetTolerance changes the tolerance.
func (s *SimulationBase) SetTolerance(t Tolerance) { s.tolerance = foundation.Some(t) }
