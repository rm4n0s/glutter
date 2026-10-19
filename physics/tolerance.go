package physics

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
)

// toleranceEpsilonDefault is the default of every field of a Tolerance.
const toleranceEpsilonDefault = 1e-3

// Tolerance structure for specifying an acceptable margin of error for
// simulations: how close to the end a value must be, how close in time, and how
// slow, to count as done.
//
// Use NewTolerance to get the default 1e-3 for the fields you leave out; the
// zero value has all fields 0.
//
// Flutter: physics/tolerance.dart Tolerance
type Tolerance struct {
	// Distance is the magnitude of the maximum distance between two points for
	// them to be considered within tolerance. The units for the distance
	// tolerance must be the same as the units used for the distances that are
	// to be compared to this tolerance.
	//
	// Flutter: physics/tolerance.dart Tolerance.distance
	Distance float64
	// Time is the magnitude of the maximum duration between two times for them
	// to be considered within tolerance, in seconds.
	//
	// Flutter: physics/tolerance.dart Tolerance.time
	Time float64
	// Velocity is the magnitude of the maximum difference between two
	// velocities for them to be considered within tolerance. The units for the
	// velocity tolerance must be the same as the units used for the velocities
	// that are to be compared to this tolerance.
	//
	// Flutter: physics/tolerance.dart Tolerance.velocity
	Velocity float64
}

// ToleranceOpts are the optional parameters of NewTolerance. Each field that is
// not set is 1e-3.
//
// Flutter: physics/tolerance.dart Tolerance.new
type ToleranceOpts struct {
	// Distance is the tolerance of positions.
	Distance foundation.Opt[float64]
	// Time is the tolerance of times, in seconds.
	Time foundation.Opt[float64]
	// Velocity is the tolerance of velocities.
	Velocity foundation.Opt[float64]
}

// NewTolerance creates a tolerance; the fields that opts leave out are 1e-3.
//
// Flutter: physics/tolerance.dart Tolerance.new
func NewTolerance(opts ...ToleranceOpts) Tolerance {
	o := firstOpts(opts)
	return Tolerance{
		Distance: o.Distance.OrElse(toleranceEpsilonDefault),
		Time:     o.Time.OrElse(toleranceEpsilonDefault),
		Velocity: o.Velocity.OrElse(toleranceEpsilonDefault),
	}
}

// ToleranceDefault is a tolerance with every field 1e-3. It never changes.
//
// Flutter: physics/tolerance.dart Tolerance.defaultTolerance
var ToleranceDefault = NewTolerance()

// String describes the tolerance, such as
// "Tolerance(distance: ±0.001, time: ±0.001, velocity: ±0.001)".
//
// Flutter: physics/tolerance.dart Tolerance.toString
func (t Tolerance) String() string {
	return "Tolerance(distance: ±" + dartfmt.Double(t.Distance) + ", time: ±" + dartfmt.Double(t.Time) +
		", velocity: ±" + dartfmt.Double(t.Velocity) + ")"
}
