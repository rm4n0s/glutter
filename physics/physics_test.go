// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package physics

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
)

// Ported from packages/flutter/test/physics/.

func TestNearEqual(t *testing.T) {
	inf, ninf := math.Inf(1), math.Inf(-1)
	tests := []struct {
		name    string
		a, b    float64
		epsilon float64
		want    bool
	}{
		{"5 and 6 within 2", 5, 6, 2, true},
		{"6 and 5 within 2", 6, 5, 2, true},
		{"5 and 6 within 0.5", 5, 6, 0.5, false},
		{"6 and 5 within 0.5", 6, 5, 0.5, false},
		{"infinity", inf, inf, 0.1, true},
		{"negative infinity", ninf, ninf, 0.1, true},
		{"opposite infinities", inf, ninf, 0.1, false},
		{"0.1 and 0.11 within 0.001", 0.1, 0.11, 0.001, false},
		{"0.1 and 0.11 within 0.1", 0.1, 0.11, 0.1, true},
		{"equal", 0.1, 0.1, 0.0000001, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NearEqual(tt.a, tt.b, tt.epsilon); got != tt.want {
				t.Errorf("NearEqual(%v, %v, %v) = %v, want %v", tt.a, tt.b, tt.epsilon, got, tt.want)
			}
		})
	}
}

func TestNearEqualOpt(t *testing.T) {
	some, none := foundation.Some[float64], foundation.None[float64]
	tests := []struct {
		name string
		a, b foundation.Opt[float64]
		want bool
	}{
		{"value and none", some(5), none(), false},
		{"none and value", none(), some(5), false},
		{"none and none", none(), none(), true},
		{"near values", some(5), some(6), true},
		{"far values", some(5), some(9), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NearEqualOpt(tt.a, tt.b, 2); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNearZeroAndNegativeEpsilon(t *testing.T) {
	if !NearZero(0.0005, 0.001) || NearZero(0.5, 0.001) {
		t.Error("NearZero")
	}
	assertRoute(t, recoverError(t, func() { NearEqual(1, 1, -1) }), "NearEqual->panicEpsilon.EpsilonNegative")
}

func TestToleranceDefaults(t *testing.T) {
	tests := []struct {
		name string
		got  Tolerance
		want Tolerance
	}{
		{"default", NewTolerance(), Tolerance{0.001, 0.001, 0.001}},
		{"package value", ToleranceDefault, Tolerance{0.001, 0.001, 0.001}},
		{"velocity only", NewTolerance(ToleranceOpts{Velocity: foundation.Some(1.0)}), Tolerance{0.001, 0.001, 1}},
		{"zero is not default", NewTolerance(ToleranceOpts{Distance: foundation.Some(0.0)}), Tolerance{0, 0.001, 0.001}},
		{"all", NewTolerance(ToleranceOpts{Distance: foundation.Some(0.01), Time: foundation.Some(0.02), Velocity: foundation.Some(0.03)}), Tolerance{0.01, 0.02, 0.03}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, tt.got, tt.want)
		}
	}
	got := ToleranceDefault.String()
	if got != "Tolerance(distance: ±0.001, time: ±0.001, velocity: ±0.001)" || strings.Contains(got, "\n") {
		t.Errorf("String() = %q: one line", got)
	}
}

func TestFrictionSimulation(t *testing.T) {
	// "test_friction" of newton_test.dart.
	friction := NewFrictionSimulation(0.3, 100, 400)
	friction.SetTolerance(NewTolerance(ToleranceOpts{Velocity: foundation.Some(1.0)}))
	if friction.IsDone(0) {
		t.Error("IsDone(0)")
	}
	check(t, "x(0)", friction.X(0), 100, 0)
	check(t, "dx(0)", friction.Dx(0), 400, 0)
	if x := friction.X(1); !(x > 330 && x < 335) {
		t.Errorf("x(1) = %v, want 330..335", x)
	}
	check(t, "dx(1)", friction.Dx(1), 120, 1e-9)
	check(t, "dx(2)", friction.Dx(2), 36, 1e-9)
	check(t, "dx(3)", friction.Dx(3), 10.8, defaultEpsilon*1e3)
	if friction.Dx(4) >= 3.5 {
		t.Errorf("dx(4) = %v, want < 3.5", friction.Dx(4))
	}
	if !friction.IsDone(5) {
		t.Error("IsDone(5)")
	}
	if x := friction.X(5); !(x > 431 && x < 432) {
		t.Errorf("x(5) = %v, want 431..432", x)
	}
}

func TestFrictionSimulationThrough(t *testing.T) {
	tests := []struct {
		name                      string
		startPosition, startVeloc float64
		forward                   bool
	}{
		{"positive velocities", 10, 600, true},
		{"negative velocities", 1000, -500, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrictionSimulation(0.025, tt.startPosition, tt.startVeloc)
			endPosition, endVelocity := f.X(1), f.Dx(1)
			if tt.forward && !(endPosition > tt.startPosition && endVelocity < tt.startVeloc) ||
				!tt.forward && !(endPosition < tt.startPosition && endVelocity > tt.startVeloc) {
				t.Fatalf("end %v, %v from %v, %v", endPosition, endVelocity, tt.startPosition, tt.startVeloc)
			}
			// The simulation that goes through those values ends up at endPosition
			// and endVelocity, which implies that it computed the right drag.
			through := NewFrictionSimulationThrough(tt.startPosition, endPosition, tt.startVeloc, endVelocity)
			if through.IsDone(0) {
				t.Error("IsDone(0)")
			}
			check(t, "x(0)", through.X(0), tt.startPosition, 1e-9)
			check(t, "dx(0)", through.Dx(0), tt.startVeloc, 0)
			if !through.IsDone(1 + foundation.PrecisionErrorTolerance) {
				t.Error("IsDone(1 + epsilon)")
			}
			check(t, "x(1)", through.X(1), endPosition, 1e-8)
			check(t, "dx(1)", through.Dx(1), endVelocity, 1e-8)
		})
	}
}

func TestFrictionSimulationThroughInvalid(t *testing.T) {
	tests := []struct {
		name           string
		sp, ep, sv, ev float64
	}{
		{"opposite velocities", 0, 10, 5, -1},
		{"end faster than start", 0, 10, 1, 5},
		{"end position the wrong way", 10, 0, 5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertRoute(t, recoverError(t, func() { NewFrictionSimulationThrough(tt.sp, tt.ep, tt.sv, tt.ev) }),
				"NewFrictionSimulationThrough.FrictionThroughInvalid")
		})
	}
}

func TestBoundedFrictionSimulation(t *testing.T) {
	friction := NewBoundedFrictionSimulation(0.3, 100, 400, 50, 150)
	friction.SetTolerance(NewTolerance(ToleranceOpts{Velocity: foundation.Some(1.0)}))
	if friction.IsDone(0) {
		t.Error("IsDone(0)")
	}
	check(t, "x(0)", friction.X(0), 100, 0)
	check(t, "dx(0)", friction.Dx(0), 400, 0)
	check(t, "x(1)", friction.X(1), 150, 0)
	if !friction.IsDone(1) {
		t.Error("IsDone(1)")
	}
	assertRoute(t, recoverError(t, func() { NewBoundedFrictionSimulation(0.3, 200, 400, 50, 150) }),
		"NewBoundedFrictionSimulation.FrictionPositionOutOfBounds")
}

func TestFrictionSimulationQueries(t *testing.T) {
	tests := []struct {
		name         string
		drag, x0, v  float64
		deceleration float64
		finalX       float64
		finalEps     float64
		xs           map[float64]float64 // time -> position (±1)
		never        []float64           // positions the particle never reaches
	}{
		{"positive velocity", 0.135, 100, 100, 0, 149, 1, map[float64]float64{0.1: 110, 0.5: 131, 2: 149}, []float64{-1, 200}},
		{"negative velocity", 0.135, 100, -100, 0, 50, 1, map[float64]float64{0.1: 91, 0.5: 68, 2: 51}, []float64{101, 40}},
		{"constant deceleration", 0.135, 100, -100, 100, 80, 1, map[float64]float64{0.1: 91, 0.5: 80, 2: 80}, []float64{101, 40}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrictionSimulation(tt.drag, tt.x0, tt.v, FrictionSimulationOpts{ConstantDeceleration: tt.deceleration})
			check(t, "x(0)", f.X(0), 100, 1e-9)
			check(t, "dx(0)", f.Dx(0), tt.v, 1e-9)
			check(t, "finalX", f.FinalX(), tt.finalX, tt.finalEps)
			for time, want := range tt.xs {
				check(t, "x", f.X(time), want, 1)
			}
			if got := f.TimeAtX(100); got != 0 {
				t.Errorf("TimeAtX(start) = %v, want 0", got)
			}
			for _, time := range []float64{0.1, 0.2, 0.3} {
				if tt.deceleration == 0 && time == 0.3 {
					continue
				}
				check(t, "timeAtX(x(time))", f.TimeAtX(f.X(time)), time, 1e-6)
			}
			if tt.deceleration == 0 {
				for _, time := range []float64{0.5, 2} {
					check(t, "timeAtX(x(time))", f.TimeAtX(f.X(time)), time, 1e-6)
				}
			}
			for _, x := range tt.never {
				if got := f.TimeAtX(x); !math.IsInf(got, 1) {
					t.Errorf("TimeAtX(%v) = %v, want infinity", x, got)
				}
			}
		})
	}
	if got := NewFrictionSimulation(0.135, 5, 0).TimeAtX(7); !math.IsInf(got, 1) {
		t.Errorf("a particle at rest never gets there: %v", got)
	}
}

func TestGravitySimulation(t *testing.T) {
	// "test_gravity" of newton_test.dart.
	g := NewGravitySimulation(200, 100, 600, 0)
	if g.IsDone(0) {
		t.Error("IsDone(0)")
	}
	check(t, "x(0)", g.X(0), 100, 0)
	check(t, "dx(0)", g.Dx(0), 0, 0)
	xs := map[float64]float64{0.25: 106.25, 0.5: 125, 0.75: 156.25, 1: 200, 1.25: 256.25, 1.5: 325, 1.75: 406.25, 2.5: 725}
	dxs := map[float64]float64{0.25: 50, 0.5: 100, 0.75: 150, 1: 200, 1.25: 250, 1.5: 300, 1.75: 350, 2.5: 500}
	for time, want := range xs {
		check(t, "x", g.X(time), want, 0)
	}
	for time, want := range dxs {
		check(t, "dx", g.Dx(time), want, 0)
	}
	if !g.IsDone(2.5) {
		t.Error("IsDone(2.5)")
	}
}

func TestGravitySimulationFlutterCases(t *testing.T) {
	// "Gravity simulation 1" and "2" of gravity_simulation_test.dart.
	check(t, "x(10)", NewGravitySimulation(9.81, 10, 0, 0).X(10), 50*9.81+10, defaultEpsilon*1e3)
	g := NewGravitySimulation(-10, 0, 6, 10)
	steps := []struct {
		time, x, dx float64
		done        bool
	}{
		{0, 0, 10, false}, {1, 5, 0, false}, {2, 0, -10, false}, {3, -15, -20, true},
	}
	for _, s := range steps {
		check(t, "x", g.X(s.time), s.x, 0)
		check(t, "dx", g.Dx(s.time), s.dx, 0)
		if g.IsDone(s.time) != s.done {
			t.Errorf("IsDone(%v) = %v, want %v", s.time, !s.done, s.done)
		}
	}
	if g.IsDone(0.2) {
		t.Error("IsDone(0.2)")
	}
	assertRoute(t, recoverError(t, func() { NewGravitySimulation(1, 2, -3, 4) }), "NewGravitySimulation.GravityEndDistanceNegative")
}

func TestClampedSimulation(t *testing.T) {
	// "Clamped simulation 1" and "2" of clamped_simulation_test.dart.
	one := NewClampedSimulation(NewGravitySimulation(9.81, 10, 0, 0), ClampedSimulationOpts{
		XMin: foundation.Some(20.0), XMax: foundation.Some(100.0), DxMin: foundation.Some(7.0), DxMax: foundation.Some(11.0)})
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"x(0)", one.X(0), 20}, {"dx(0)", one.Dx(0), 7}, {"x(100)", one.X(100), 100}, {"dx(100)", one.Dx(100), 11},
	} {
		check(t, c.name, c.got, c.want, 0)
	}
	two := NewClampedSimulation(NewGravitySimulation(-10, 0, 6, 10), ClampedSimulationOpts{
		XMin: foundation.Some(0.0), XMax: foundation.Some(2.5), DxMin: foundation.Some(-1.0), DxMax: foundation.Some(1.0)})
	steps := []struct {
		time, x, dx float64
		done        bool
	}{
		{0, 0, 1, false}, {1, 2.5, 0, false}, {2, 0, -1, false}, {3, 0, -1, true},
	}
	for _, s := range steps {
		check(t, "x", two.X(s.time), s.x, 0)
		check(t, "dx", two.Dx(s.time), s.dx, 0)
		if two.IsDone(s.time) != s.done {
			t.Errorf("IsDone(%v) = %v", s.time, !s.done)
		}
	}
	if two.IsDone(0.2) {
		t.Error("IsDone(0.2)")
	}
	// A clamped simulation with no limits changes nothing.
	free := NewClampedSimulation(NewGravitySimulation(10, 0, 100, 0))
	check(t, "free x", free.X(2), 20, 0)
	assertRoute(t, recoverError(t, func() {
		NewClampedSimulation(&testSimulation{}, ClampedSimulationOpts{XMin: foundation.Some(1.0), XMax: foundation.Some(0.0)})
	}), "NewClampedSimulation.ClampedRangeInvalid")
	assertRoute(t, recoverError(t, func() {
		NewClampedSimulation(&testSimulation{}, ClampedSimulationOpts{DxMin: foundation.Some(1.0), DxMax: foundation.Some(0.0)})
	}), "NewClampedSimulation.ClampedRangeInvalid")
	if one.Simulation() == nil {
		t.Error("Simulation()")
	}
}

func TestSpringTypes(t *testing.T) {
	tests := []struct {
		name string
		desc SpringDescription
		want SpringType
	}{
		{"critically damped", NewSpringDescriptionWithDampingRatio(1, 100), SpringTypeCriticallyDamped},
		{"under damped", NewSpringDescriptionWithDampingRatio(1, 100, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(0.75)}), SpringTypeUnderDamped},
		{"over damped", NewSpringDescriptionWithDampingRatio(1, 100, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(1.25)}), SpringTypeOverDamped},
		{"critical without a ratio", NewSpringDescription(1, 100, 20), SpringTypeCriticallyDamped},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpringSimulation(tt.desc, 0, 300, 0)
			if s.Type() != tt.want {
				t.Errorf("Type() = %v, want %v", s.Type(), tt.want)
			}
		})
	}
}

func TestCriticalSpring(t *testing.T) {
	crit := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100), 0, 500, 0)
	crit.SetTolerance(NewTolerance(ToleranceOpts{Distance: foundation.Some(0.01), Velocity: foundation.Some(0.01)}))
	if crit.Type() != SpringTypeCriticallyDamped || crit.IsDone(0) {
		t.Fatal("type or IsDone(0)")
	}
	check(t, "x(0)", crit.X(0), 0, 0)
	check(t, "dx(0)", crit.Dx(0), 0, 0)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"x(0.25)", crit.X(0.25), 356}, {"x(0.5)", crit.X(0.5), 479}, {"x(0.75)", crit.X(0.75), 497},
		{"dx(0.25)", crit.Dx(0.25), 1026}, {"dx(0.5)", crit.Dx(0.5), 168}, {"dx(0.75)", crit.Dx(0.75), 20},
	} {
		if math.Floor(c.got) != c.want {
			t.Errorf("floor(%s) = %v, want %v", c.name, math.Floor(c.got), c.want)
		}
	}
	if x := crit.X(1.5); !(x > 499 && x < 501) {
		t.Errorf("x(1.5) = %v", x)
	}
	if crit.Dx(1.5) >= 0.1 {
		t.Errorf("dx(1.5) = %v", crit.Dx(1.5))
	}
	if !crit.IsDone(1.6) {
		t.Error("IsDone(1.6)")
	}
}

func TestOverdampedSpring(t *testing.T) {
	over := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(1.25)}), 0, 500, 0)
	over.SetTolerance(NewTolerance(ToleranceOpts{Distance: foundation.Some(0.01), Velocity: foundation.Some(0.01)}))
	if over.Type() != SpringTypeOverDamped || over.IsDone(0) {
		t.Fatal("type or IsDone(0)")
	}
	check(t, "x(0)", over.X(0), 0, 0)
	check(t, "dx(0)", over.Dx(0), 0, defaultEpsilon*1e3)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"x(0.5)", over.X(0.5), 445}, {"x(1)", over.X(1), 495}, {"x(1.5)", over.X(1.5), 499},
		{"dx(0.5)", over.Dx(0.5), 273}, {"dx(1)", over.Dx(1), 22}, {"dx(1.5)", over.Dx(1.5), 1},
	} {
		if math.Floor(c.got) != c.want {
			t.Errorf("floor(%s) = %v, want %v", c.name, math.Floor(c.got), c.want)
		}
	}
	if !over.IsDone(3) {
		t.Error("IsDone(3)")
	}
}

func TestUnderdampedSpring(t *testing.T) {
	under := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(0.25)}), 0, 300, 0)
	if under.Type() != SpringTypeUnderDamped || under.IsDone(0) {
		t.Fatal("type or IsDone(0)")
	}
	check(t, "x(0)", under.X(0), 0, defaultEpsilon*1e3)
	check(t, "dx(0)", under.Dx(0), 0, defaultEpsilon*1e3)
	// Overshot with negative velocity.
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"x(1)", under.X(1), 325}, {"dx(1)", under.Dx(1), -65}, {"dx(6)", under.Dx(6), 0}, {"x(6)", under.X(6), 299},
	} {
		if math.Floor(c.got) != c.want {
			t.Errorf("floor(%s) = %v, want %v", c.name, math.Floor(c.got), c.want)
		}
	}
	if !under.IsDone(6) {
		t.Error("IsDone(6)")
	}
}

func TestSpringSnapToEnd(t *testing.T) {
	desc := NewSpringDescriptionWithDampingRatio(1, 400)
	tol := foundation.Some(NewTolerance(ToleranceOpts{Distance: foundation.Some(0.1), Velocity: foundation.Some(0.1)}))
	const time = 0.4
	regular := NewSpringSimulation(desc, 0, 1, 0, SpringSimulationOpts{Tolerance: tol})
	if !(regular.X(time) < 1) || !(regular.Dx(time) > 0) {
		t.Errorf("regular: x %v dx %v", regular.X(time), regular.Dx(time))
	}
	snapping := NewSpringSimulation(desc, 0, 1, 0, SpringSimulationOpts{SnapToEnd: true, Tolerance: tol})
	// Must be exactly equal.
	if snapping.X(time) != 1 || snapping.Dx(time) != 0 {
		t.Errorf("snapping: x %v dx %v", snapping.X(time), snapping.Dx(time))
	}
}

func TestSpringContinuousNearCriticalDamping(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/163858
	const time, stiffness, mass = 0.4, 0.4, 0.4
	for _, ratio := range []float64{1, 1 + 1e-3, 1 - 1e-3} {
		s := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(mass, stiffness, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(ratio)}), 0, 1, 0)
		check(t, "x", s.X(time), 0.06155, 0.01)
		check(t, "dx", s.Dx(time), 0.2681, 0.01)
	}
}

func TestSpringDescriptionWithDurationAndBounce(t *testing.T) {
	t.Run("bounce 0.3", func(t *testing.T) {
		s := NewSpringDescriptionWithDurationAndBounce(SpringDescriptionWithDurationAndBounceOpts{Bounce: 0.3})
		check(t, "mass", s.Mass, 1, 0)
		check(t, "stiffness", s.Stiffness, 157.91, 0.01)
		check(t, "damping", s.Damping, 17.59, 0.01)
		check(t, "bounce", s.Bounce(), 0.3, 0.0001)
		if s.Duration() != 500*time.Millisecond {
			t.Errorf("duration = %v", s.Duration())
		}
	})
	t.Run("negative bounce", func(t *testing.T) {
		s := NewSpringDescriptionWithDurationAndBounce(SpringDescriptionWithDurationAndBounceOpts{Bounce: -0.3})
		check(t, "stiffness", s.Stiffness, 157.91, 0.01)
		check(t, "damping", s.Damping, 35.90, 0.01)
		check(t, "bounce", s.Bounce(), -0.3, 0.0001)
		if s.Duration() != 500*time.Millisecond {
			t.Errorf("duration = %v", s.Duration())
		}
	})
	t.Run("from mass and stiffness", func(t *testing.T) {
		s := NewSpringDescription(1, 157.91, 17.59)
		check(t, "bounce", s.Bounce(), 0.3, 0.001)
		if s.Duration() != 500*time.Millisecond {
			t.Errorf("duration = %v", s.Duration())
		}
	})
	t.Run("custom duration", func(t *testing.T) {
		s := NewSpringDescriptionWithDurationAndBounce(SpringDescriptionWithDurationAndBounceOpts{Duration: foundation.Some(100 * time.Millisecond)})
		check(t, "stiffness", s.Stiffness, 3947.84, 0.01)
		check(t, "damping", s.Damping, 125.66, 0.01)
		check(t, "bounce", s.Bounce(), 0, 0.001)
		if s.Duration() != 100*time.Millisecond {
			t.Errorf("duration = %v", s.Duration())
		}
	})
	for _, d := range []time.Duration{-time.Second, 0, 500 * time.Microsecond} {
		assertRoute(t, recoverError(t, func() {
			NewSpringDescriptionWithDurationAndBounce(SpringDescriptionWithDurationAndBounceOpts{Duration: foundation.Some(d), Bounce: 0.3})
		}), "NewSpringDescriptionWithDurationAndBounce.SpringDurationInvalid")
	}
}

func TestScrollSpringSimulationIsExactAtTheEnd(t *testing.T) {
	desc := NewSpringDescriptionWithDampingRatio(1, 400)
	tol := foundation.Some(NewTolerance(ToleranceOpts{Distance: foundation.Some(0.1), Velocity: foundation.Some(0.1)}))
	s := NewScrollSpringSimulation(desc, 0, 1, 0, SpringSimulationOpts{Tolerance: tol})
	if s.X(0.4) != 1 || !s.IsDone(0.4) {
		t.Errorf("x(0.4) = %v, done %v: exactly the end once done", s.X(0.4), s.IsDone(0.4))
	}
	if s.X(0.05) >= 1 || s.X(0.05) <= 0 {
		t.Errorf("x(0.05) = %v, want between 0 and 1", s.X(0.05))
	}
}

func TestKineticScroll(t *testing.T) {
	spring := NewSpringDescriptionWithDampingRatio(1, 50, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(0.5)})
	tol := NewTolerance(ToleranceOpts{Velocity: foundation.Some(0.5), Distance: foundation.Some(0.1)})
	for _, velocity := range []float64{800, -800} {
		scroll := NewBouncingScrollSimulation(100, velocity, 0, 300, spring)
		scroll.SetTolerance(tol)
		if scroll.IsDone(0) || scroll.IsDone(0.5) { // switch from friction to spring
			t.Errorf("velocity %v: done too early", velocity)
		}
		if !scroll.IsDone(3.5) {
			t.Errorf("velocity %v: not done at 3.5", velocity)
		}
	}
}

func TestScrollWithInfiniteEdgeEnds(t *testing.T) {
	spring := NewSpringDescriptionWithDampingRatio(1, 50, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(0.5)})
	scroll := NewBouncingScrollSimulation(100, 400, 0, math.Inf(1), spring)
	scroll.SetTolerance(NewTolerance(ToleranceOpts{Velocity: foundation.Some(1.0)}))
	if scroll.IsDone(0) {
		t.Error("IsDone(0)")
	}
	check(t, "x(0)", scroll.X(0), 100, 0)
	check(t, "dx(0)", scroll.Dx(0), 400, 0)
	check(t, "x(1)", scroll.X(1), 272, 1)
	check(t, "dx(1)", scroll.Dx(1), 54, 1)
	check(t, "dx(2)", scroll.Dx(2), 7, 1)
	if scroll.Dx(3) >= 1 {
		t.Errorf("dx(3) = %v", scroll.Dx(3))
	}
	if !scroll.IsDone(5) {
		t.Error("IsDone(5)")
	}
	check(t, "x(5)", scroll.X(5), 300, 1)
}

func TestOverUnderScrollSpring(t *testing.T) {
	spring := NewSpringDescriptionWithDampingRatio(1, 170, SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(1.1)})
	scroll := NewBouncingScrollSimulation(500, -7500, 0, 1000, spring)
	scroll.SetTolerance(NewTolerance(ToleranceOpts{Velocity: foundation.Some(45.0), Distance: foundation.Some(1.5)}))
	if scroll.IsDone(0) {
		t.Error("IsDone(0)")
	}
	check(t, "x(0)", scroll.X(0), 500, defaultEpsilon*1e3)
	check(t, "dx(0)", scroll.Dx(0), -7500, defaultEpsilon*1e3)
	// Expect to reach 0.0 at about t=.07, at which point the simulation will
	// switch from friction to the spring.
	if scroll.IsDone(0.065) {
		t.Error("IsDone(0.065)")
	}
	check(t, "x(0.065)", scroll.X(0.065), 42, 1)
	check(t, "dx(0.065)", scroll.Dx(0.065), -6584, 1)
	// We've overscrolled (0.1 > 0.07). Trigger the underscroll simulation, and
	// reverse direction.
	if scroll.IsDone(0.1) {
		t.Error("IsDone(0.1)")
	}
	check(t, "x(0.1)", scroll.X(0.1), -123, 1)
	check(t, "dx(0.1)", scroll.Dx(0.1), -2613, 1)
	// Headed back towards 0.0 and slowing down.
	if scroll.IsDone(0.5) {
		t.Error("IsDone(0.5)")
	}
	check(t, "x(0.5)", scroll.X(0.5), -15, 1)
	check(t, "dx(0.5)", scroll.Dx(0.5), 124, 1)
	// Now jump back to the beginning, because we can.
	if scroll.IsDone(0) {
		t.Error("IsDone(0) again")
	}
	check(t, "x(0) again", scroll.X(0), 500, defaultEpsilon*1e3)
	check(t, "dx(0) again", scroll.Dx(0), -7500, defaultEpsilon*1e3)
	if !scroll.IsDone(2) {
		t.Error("IsDone(2)")
	}
	check(t, "x(2)", scroll.X(2), 0, 0)
	check(t, "dx(2)", scroll.Dx(2), 0, 1)
}

func TestBouncingScrollFromOutsideTheExtents(t *testing.T) {
	spring := NewSpringDescriptionWithDampingRatio(1, 170)
	tests := []struct {
		name     string
		position float64
		end      float64
	}{
		{"before the leading extent", -100, 0},
		{"after the trailing extent", 1100, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewBouncingScrollSimulation(tt.position, 0, 0, 1000, spring)
			check(t, "x(0)", s.X(0), tt.position, 1e-9)
			if s.IsDone(0) {
				t.Error("IsDone(0)")
			}
			if !s.IsDone(5) {
				t.Error("IsDone(5)")
			}
			check(t, "x(5)", s.X(5), tt.end, 0)
		})
	}
	s := NewBouncingScrollSimulation(0, 0, 0, 10, spring)
	if s.LeadingExtent() != 0 || s.TrailingExtent() != 10 || s.Spring() != spring {
		t.Error("accessors")
	}
	assertRoute(t, recoverError(t, func() { NewBouncingScrollSimulation(0, 0, 10, 0, spring) }),
		"NewBouncingScrollSimulation.ScrollExtentsInvalid")
	if got := s.String(); got != "BouncingScrollSimulation(leadingExtent: 0.0, trailingExtent: 10.0)" {
		t.Errorf("String() = %q", got)
	}
}

func TestBouncingScrollConstantDeceleration(t *testing.T) {
	spring := NewSpringDescriptionWithDampingRatio(1, 170)
	s := NewBouncingScrollSimulation(100, 500, 0, 1000, spring, BouncingScrollSimulationOpts{ConstantDeceleration: 100})
	// With the constant deceleration the scroll stops sooner than friction alone.
	plain := NewBouncingScrollSimulation(100, 500, 0, 1000, spring)
	if !(s.X(10) < plain.X(10)) {
		t.Errorf("x(10) = %v, plain %v", s.X(10), plain.X(10))
	}
}

func TestClampingScrollSimulation(t *testing.T) {
	tests := []struct {
		name     string
		position float64
		velocity float64
	}{
		{"fling forward", 0, 2000},
		{"fling backward", 5000, -3000},
		{"slow fling", 100, 50},
		{"fast fling", 0, 9000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewClampingScrollSimulation(tt.position, tt.velocity)
			check(t, "x(0)", s.X(0), tt.position, 1e-9)
			check(t, "dx(0)", s.Dx(0), tt.velocity, 1e-9)
			if s.IsDone(0) {
				t.Error("IsDone(0)")
			}
			// The velocity falls and the position moves in the direction of the fling.
			prevX, prevDx := s.X(0), math.Abs(s.Dx(0))
			for time := 0.05; time < 1.5; time += 0.05 {
				x, dx := s.X(time), math.Abs(s.Dx(time))
				if (x-prevX)*tt.velocity < 0 || dx > prevDx+1e-9 {
					t.Fatalf("t=%v: x %v (was %v), |dx| %v (was %v)", time, x, prevX, dx, prevDx)
				}
				prevX, prevDx = x, dx
			}
			// At the end it is done and has stopped.
			end := 100.0
			if !s.IsDone(end) {
				t.Fatal("IsDone(end)")
			}
			check(t, "dx(end)", s.Dx(end), 0, 1e-9)
			// The distance is what Android's formula gives: v * duration / rate.
			check(t, "distance", s.X(end)-tt.position, tt.velocity*s.duration/clampingDecelerationRate, 1e-6)
		})
	}
	s := NewClampingScrollSimulation(10, 1000, ClampingScrollSimulationOpts{Friction: foundation.Some(0.03)})
	slower := NewClampingScrollSimulation(10, 1000)
	if !(s.X(100)-10 < slower.X(100)-10) || s.Friction() != 0.03 || s.Position() != 10 || s.Velocity() != 1000 {
		t.Errorf("friction option: %v vs %v", s.X(100), slower.X(100))
	}
	// A fling of zero velocity does not move.
	still := NewClampingScrollSimulation(7, 0)
	check(t, "still x", still.X(1), 7, 0)
}

func TestSimulationToString(t *testing.T) {
	// "Simulation.toString" of to_string_test.dart.
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"clamped", NewClampedSimulation(&testSimulation{}, ClampedSimulationOpts{
			XMin: foundation.Some(-1.0), XMax: foundation.Some(2.0), DxMin: foundation.Some(-3.0), DxMax: foundation.Some(4.0)}).String(),
			"ClampedSimulation(simulation: testSimulation, x: -1.0..2.0, dx: -3.0..4.0)"},
		{"gravity", NewGravitySimulation(1, -2, 3, -4).String(), "GravitySimulation(g: 1.0, x₀: -2.0, dx₀: -4.0, xₘₐₓ: ±3.0)"},
		{"friction", NewFrictionSimulation(1, -2, 3).String(), "FrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0)"},
		{"bounded friction", NewBoundedFrictionSimulation(1, -2, 3, -4, 5).String(), "BoundedFrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0, x: -4.0..5.0)"},
		{"spring description", NewSpringDescription(1, -2, 3).String(), "SpringDescription(mass: 1.0, stiffness: -2.0, damping: 3.0)"},
		{"spring description with ratio", NewSpringDescriptionWithDampingRatio(1, 9).String(), "SpringDescription(mass: 1.0, stiffness: 9.0, damping: 6.0)"},
		{"spring", NewSpringSimulation(NewSpringDescription(1, 2, 3), 0, 1, 2).String(), "SpringSimulation(end: 1.0, SpringType.overDamped)"},
		{"spring type", SpringTypeUnderDamped.Name() + " " + SpringType(9).String(), "underDamped SpringType(9)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
	// A simulation without a String method is described by its type.
	if got := describe(&testSimulation{}); got != "testSimulation" {
		t.Errorf("describe = %q", got)
	}
	var s Simulation = &testSimulation{}
	if s.Tolerance() != ToleranceDefault {
		t.Errorf("default tolerance = %v", s.Tolerance())
	}
	s.SetTolerance(Tolerance{1, 2, 3})
	_ = s
}
