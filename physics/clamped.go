package physics

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// ClampedSimulation is a simulation that restricts the position and velocity of
// another simulation to a range.
//
// Flutter: physics/clamped_simulation.dart ClampedSimulation
type ClampedSimulation struct {
	SimulationBase
	simulation Simulation
	xMin, xMax float64
	dxMin      float64
	dxMax      float64
}

// ClampedSimulationOpts are the optional parameters of NewClampedSimulation. A
// bound that is not set is infinite.
//
// Flutter: physics/clamped_simulation.dart ClampedSimulation.new
type ClampedSimulationOpts struct {
	// XMin is the lowest position; negative infinity by default.
	XMin foundation.Opt[float64]
	// XMax is the highest position; infinity by default.
	XMax foundation.Opt[float64]
	// DxMin is the lowest velocity; negative infinity by default.
	DxMin foundation.Opt[float64]
	// DxMax is the highest velocity; infinity by default.
	DxMax foundation.Opt[float64]
}

// NewClampedSimulation creates a simulation that gives the values of
// simulation, clamped to the ranges. A range whose maximum is below its minimum
// panics with the tag ClampedRangeInvalid in debug builds.
//
// Flutter: physics/clamped_simulation.dart ClampedSimulation.new
func NewClampedSimulation(simulation Simulation, opts ...ClampedSimulationOpts) *ClampedSimulation {
	o := firstOpts(opts)
	c := &ClampedSimulation{
		simulation: simulation,
		xMin:       o.XMin.OrElse(math.Inf(-1)),
		xMax:       o.XMax.OrElse(math.Inf(1)),
		dxMin:      o.DxMin.OrElse(math.Inf(-1)),
		dxMax:      o.DxMax.OrElse(math.Inf(1)),
	}
	if foundation.DebugMode && (c.xMax < c.xMin || c.dxMax < c.dxMin) {
		panic(errors.New(tagClampedRangeInvalid, "the maximum of a range must not be below its minimum",
			"xMin", c.xMin, "xMax", c.xMax, "dxMin", c.dxMin, "dxMax", c.dxMax))
	}
	return c
}

// Simulation is the simulation being clamped.
//
// Flutter: physics/clamped_simulation.dart ClampedSimulation.simulation
func (c *ClampedSimulation) Simulation() Simulation { return c.simulation }

// X is the position of the simulation, clamped.
func (c *ClampedSimulation) X(time float64) float64 {
	return ui.ClampDouble(c.simulation.X(time), c.xMin, c.xMax)
}

// Dx is the velocity of the simulation, clamped.
func (c *ClampedSimulation) Dx(time float64) float64 {
	return ui.ClampDouble(c.simulation.Dx(time), c.dxMin, c.dxMax)
}

// IsDone is whether the simulation is done.
func (c *ClampedSimulation) IsDone(time float64) bool { return c.simulation.IsDone(time) }

// String describes the simulation, such as
// "ClampedSimulation(simulation: Foo, x: -1.0..2.0, dx: -3.0..4.0)".
//
// Flutter: physics/clamped_simulation.dart ClampedSimulation.toString
func (c *ClampedSimulation) String() string {
	return foundation.RuntimeTypeName(c) + "(simulation: " + describe(c.simulation) + ", x: " +
		dartfmt.ToStringAsFixed(c.xMin, 1) + ".." + dartfmt.ToStringAsFixed(c.xMax, 1) + ", dx: " +
		dartfmt.ToStringAsFixed(c.dxMin, 1) + ".." + dartfmt.ToStringAsFixed(c.dxMax, 1) + ")"
}

// describe is Dart's toString of a simulation: its String method, or its type.
func describe(s Simulation) string {
	if str, ok := s.(interface{ String() string }); ok {
		return str.String()
	}
	return foundation.RuntimeTypeName(s)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClampedRangeInvalid = "ClampedRangeInvalid"
)
