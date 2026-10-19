package physics

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// BouncingScrollSimulation is a simulation of scrolling with the physics of
// iOS and macOS: friction while the content is inside its extents, and a spring
// that pulls it back when it goes beyond them (overscroll).
//
// Deviation: Flutter has it in widgets/scroll_simulation.dart; SPEC §11.1 lists
// it with the physics (SPEC §12.10).
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation
type BouncingScrollSimulation struct {
	SimulationBase
	leadingExtent, trailingExtent float64
	spring                        SpringDescription

	friction   *FrictionSimulation
	springSim  Simulation
	springTime float64
	timeOffset float64
}

// BouncingScrollSimulationMaxSpringTransferVelocity is the largest velocity, in
// pixels per second, that the friction part hands over to the spring part when
// it reaches an extent.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.maxSpringTransferVelocity
const BouncingScrollSimulationMaxSpringTransferVelocity = 5000.0

// BouncingScrollSimulationOpts are the optional parameters of
// NewBouncingScrollSimulation.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.new
type BouncingScrollSimulationOpts struct {
	// ConstantDeceleration is added to the friction; 0 by default.
	ConstantDeceleration float64
	// Tolerance decides when the simulation is done; the default tolerance if
	// not set.
	Tolerance foundation.Opt[Tolerance]
}

// NewBouncingScrollSimulation creates the simulation of a scroll at position
// with the velocity, between leadingExtent and trailingExtent, with the spring
// for the overscroll. A leading extent past the trailing one panics with the
// tag ScrollExtentsInvalid in debug builds.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.new
func NewBouncingScrollSimulation(position, velocity, leadingExtent, trailingExtent float64, spring SpringDescription, opts ...BouncingScrollSimulationOpts) *BouncingScrollSimulation {
	o := firstOpts(opts)
	if foundation.DebugMode && leadingExtent > trailingExtent {
		panic(errors.New("ScrollExtentsInvalid", "the leading extent must not be past the trailing extent",
			"leadingExtent", leadingExtent, "trailingExtent", trailingExtent))
	}
	b := &BouncingScrollSimulation{
		SimulationBase: NewSimulationBase(o.Tolerance),
		leadingExtent:  leadingExtent,
		trailingExtent: trailingExtent,
		spring:         spring,
	}
	switch {
	case position < leadingExtent:
		b.springSim = b.underscrollSimulation(position, velocity)
		b.springTime = math.Inf(-1)
	case position > trailingExtent:
		b.springSim = b.overscrollSimulation(position, velocity)
		b.springTime = math.Inf(-1)
	default:
		// Taken from UIScrollView.decelerationRate (.normal = 0.998)
		// 0.998^1000 = ~0.135
		b.friction = NewFrictionSimulation(0.135, position, velocity, FrictionSimulationOpts{ConstantDeceleration: o.ConstantDeceleration})
		finalX := b.friction.FinalX()
		switch {
		case velocity > 0 && finalX > trailingExtent:
			b.springTime = b.friction.TimeAtX(trailingExtent)
			b.springSim = b.overscrollSimulation(trailingExtent,
				math.Min(b.friction.Dx(b.springTime), BouncingScrollSimulationMaxSpringTransferVelocity))
		case velocity < 0 && finalX < leadingExtent:
			b.springTime = b.friction.TimeAtX(leadingExtent)
			b.springSim = b.underscrollSimulation(leadingExtent,
				math.Min(b.friction.Dx(b.springTime), BouncingScrollSimulationMaxSpringTransferVelocity))
		default:
			b.springTime = math.Inf(1)
		}
	}
	return b
}

// LeadingExtent is where the scrollable content starts.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.leadingExtent
func (b *BouncingScrollSimulation) LeadingExtent() float64 { return b.leadingExtent }

// TrailingExtent is where the scrollable content ends.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.trailingExtent
func (b *BouncingScrollSimulation) TrailingExtent() float64 { return b.trailingExtent }

// Spring is the spring that pulls an overscroll back.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.spring
func (b *BouncingScrollSimulation) Spring() SpringDescription { return b.spring }

func (b *BouncingScrollSimulation) underscrollSimulation(x, dx float64) Simulation {
	return NewScrollSpringSimulation(b.spring, x, b.leadingExtent, dx)
}

func (b *BouncingScrollSimulation) overscrollSimulation(x, dx float64) Simulation {
	return NewScrollSpringSimulation(b.spring, x, b.trailingExtent, dx)
}

// simulation picks the part that is running at the time, and the time offset
// that part counts from.
func (b *BouncingScrollSimulation) simulation(time float64) Simulation {
	var sim Simulation
	if time > b.springTime {
		b.timeOffset = 0
		if !math.IsInf(b.springTime, 0) && !math.IsNaN(b.springTime) {
			b.timeOffset = b.springTime
		}
		sim = b.springSim
	} else {
		b.timeOffset = 0
		sim = b.friction
	}
	sim.SetTolerance(b.Tolerance())
	return sim
}

// X is the scroll position at the time.
func (b *BouncingScrollSimulation) X(time float64) float64 {
	return b.simulation(time).X(time - b.timeOffset)
}

// Dx is the scroll velocity at the time.
func (b *BouncingScrollSimulation) Dx(time float64) float64 {
	return b.simulation(time).Dx(time - b.timeOffset)
}

// IsDone is whether the scroll has settled.
func (b *BouncingScrollSimulation) IsDone(time float64) bool {
	return b.simulation(time).IsDone(time - b.timeOffset)
}

// String describes the simulation.
//
// Flutter: widgets/scroll_simulation.dart BouncingScrollSimulation.toString
func (b *BouncingScrollSimulation) String() string {
	return "BouncingScrollSimulation(leadingExtent: " + dartfmt.Double(b.leadingExtent) + ", trailingExtent: " + dartfmt.Double(b.trailingExtent) + ")"
}

// The constants of ClampingScrollSimulation follow Android's OverScroller (the
// class SplineOverScroller, from its method fling): times in seconds,
// positions in logical pixels, velocities in logical pixels per second.
var (
	// clampingDecelerationRate is DECELERATION_RATE.
	clampingDecelerationRate = math.Log(0.78) / math.Log(0.9)
)

const (
	// clampingInflexion is INFLEXION.
	clampingInflexion = 0.35
	// clampingPhysicalCoeff is mPhysicalCoeff: 0.84 times Earth gravity, in
	// logical pixels per second squared.
	clampingPhysicalCoeff = 9.80665 * // g, in meters per second^2
		39.37 * // 1 meter / 1 inch
		160.0 * // 1 inch / 1 logical pixel
		0.84 // "look and feel tuning"
)

// ClampingScrollSimulation is a simulation of scrolling with the physics of
// Android: a fling that slows down along a curve and stops at its end. It does
// not know about the extents of the content; the scroll position clamps.
//
// It is based on SplineOverScroller in Android's OverScroller.java.
//
// Deviation: Flutter has it in widgets/scroll_simulation.dart; SPEC §11.1 lists
// it with the physics (SPEC §12.10).
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation
type ClampingScrollSimulation struct {
	SimulationBase
	position, velocity, friction float64
	duration, distance           float64
}

// ClampingScrollSimulationOpts are the optional parameters of
// NewClampingScrollSimulation.
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation.new
type ClampingScrollSimulationOpts struct {
	// Friction is the fling friction (mFlingFriction); 0.015 by default.
	Friction foundation.Opt[float64]
	// Tolerance decides when the simulation is done; the default tolerance if
	// not set.
	Tolerance foundation.Opt[Tolerance]
}

// NewClampingScrollSimulation creates the simulation of a fling that starts at
// position with the velocity.
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation.new
func NewClampingScrollSimulation(position, velocity float64, opts ...ClampingScrollSimulationOpts) *ClampingScrollSimulation {
	o := firstOpts(opts)
	c := &ClampingScrollSimulation{
		SimulationBase: NewSimulationBase(o.Tolerance),
		position:       position,
		velocity:       velocity,
		friction:       o.Friction.OrElse(0.015),
	}
	c.duration = c.flingDuration()
	c.distance = c.flingDistance()
	return c
}

// Position is where the fling starts.
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation.position
func (c *ClampingScrollSimulation) Position() float64 { return c.position }

// Velocity is the velocity at the start.
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation.velocity
func (c *ClampingScrollSimulation) Velocity() float64 { return c.velocity }

// Friction is the fling friction.
//
// Flutter: widgets/scroll_simulation.dart ClampingScrollSimulation.friction
func (c *ClampingScrollSimulation) Friction() float64 { return c.friction }

// flingDuration is getSplineFlingDuration().
func (c *ClampingScrollSimulation) flingDuration() float64 {
	// See getSplineDeceleration(). That function's value is
	// math.log(velocity.abs() / referenceVelocity).
	referenceVelocity := c.friction * clampingPhysicalCoeff / clampingInflexion
	// This is the value getSplineFlingDuration() would return, but in seconds.
	androidDuration := math.Pow(math.Abs(c.velocity)/referenceVelocity, 1/(clampingDecelerationRate-1))
	// We finish a bit sooner than Android, in order to travel the same total
	// distance.
	return clampingDecelerationRate * clampingInflexion * androidDuration
}

// flingDistance is getSplineFlingDistance(): the same value with the sign of the
// velocity, in logical pixels.
func (c *ClampingScrollSimulation) flingDistance() float64 {
	distance := c.velocity * c.duration / clampingDecelerationRate
	if foundation.DebugMode {
		// This is the more complicated calculation that getSplineFlingDistance()
		// actually performs, which boils down to the much simpler formula above.
		referenceVelocity := c.friction * clampingPhysicalCoeff / clampingInflexion
		logVelocity := math.Log(math.Abs(c.velocity) / referenceVelocity)
		distanceAgain := c.friction * clampingPhysicalCoeff *
			math.Exp(logVelocity*clampingDecelerationRate/(clampingDecelerationRate-1))
		if !(math.Abs(math.Abs(distance)-distanceAgain) < c.Tolerance().Distance) {
			panic(errors.New("ClampingFlingDistanceInconsistent", "the fling distance does not match Android's calculation",
				"distance", distance, "distanceAgain", distanceAgain))
		}
	}
	return distance
}

// X is the position at the time.
func (c *ClampingScrollSimulation) X(time float64) float64 {
	t := ui.ClampDouble(time/c.duration, 0, 1)
	return c.position + c.distance*(1-math.Pow(1-t, clampingDecelerationRate))
}

// Dx is the velocity at the time.
func (c *ClampingScrollSimulation) Dx(time float64) float64 {
	t := ui.ClampDouble(time/c.duration, 0, 1)
	return c.velocity * math.Pow(1-t, clampingDecelerationRate-1)
}

// IsDone is whether the fling has run its duration.
func (c *ClampingScrollSimulation) IsDone(time float64) bool { return time >= c.duration }
