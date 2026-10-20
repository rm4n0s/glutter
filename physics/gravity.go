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
)

// GravitySimulation is a simulation of a particle with constant acceleration,
// such as a falling object.
//
// Flutter: physics/gravity_simulation.dart GravitySimulation
type GravitySimulation struct {
	SimulationBase
	x, v, a, end float64
}

// NewGravitySimulation creates a simulation of an object that starts at
// distance with the velocity, accelerated by acceleration, and is done when it
// is endDistance or more away from zero. An endDistance below zero panics with
// the tag GravityEndDistanceNegative in debug builds.
//
// Flutter: physics/gravity_simulation.dart GravitySimulation.new
func NewGravitySimulation(acceleration, distance, endDistance, velocity float64) *GravitySimulation {
	if foundation.DebugMode && endDistance < 0 {
		panic(errors.New("GravityEndDistanceNegative", "endDistance must not be negative", "endDistance", endDistance))
	}
	return &GravitySimulation{a: acceleration, x: distance, v: velocity, end: endDistance}
}

// X is the position at the time.
func (g *GravitySimulation) X(time float64) float64 { return g.x + g.v*time + 0.5*g.a*time*time }

// Dx is the velocity at the time.
func (g *GravitySimulation) Dx(time float64) float64 { return g.v + time*g.a }

// IsDone is whether the object is endDistance away from zero.
func (g *GravitySimulation) IsDone(time float64) bool { return math.Abs(g.X(time)) >= g.end }

// String describes the simulation, such as
// "GravitySimulation(g: 1.0, x₀: -2.0, dx₀: -4.0, xₘₐₓ: ±3.0)".
//
// Flutter: physics/gravity_simulation.dart GravitySimulation.toString
func (g *GravitySimulation) String() string {
	return "GravitySimulation(g: " + dartfmt.ToStringAsFixed(g.a, 1) + ", x₀: " + dartfmt.ToStringAsFixed(g.x, 1) +
		", dx₀: " + dartfmt.ToStringAsFixed(g.v, 1) + ", xₘₐₓ: ±" + dartfmt.ToStringAsFixed(g.end, 1) + ")"
}
