package gestures

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// Velocity is a velocity in two dimensions, in logical pixels per second.
//
// Flutter: gestures/velocity_tracker.dart Velocity
type Velocity struct {
	// PixelsPerSecond is the number of pixels per second in each direction.
	//
	// Flutter: gestures/velocity_tracker.dart Velocity.pixelsPerSecond
	PixelsPerSecond ui.Offset
}

// VelocityZero is the zero velocity.
//
// Flutter: gestures/velocity_tracker.dart Velocity.zero
var VelocityZero = Velocity{}

// NewVelocity creates a velocity.
//
// Flutter: gestures/velocity_tracker.dart Velocity.new
func NewVelocity(pixelsPerSecond ui.Offset) Velocity {
	return Velocity{PixelsPerSecond: pixelsPerSecond}
}

// Neg returns the velocity in the opposite direction.
//
// Flutter: gestures/velocity_tracker.dart Velocity.operator-
func (v Velocity) Neg() Velocity { return Velocity{v.PixelsPerSecond.Neg()} }

// Sub returns the velocity minus other.
//
// Flutter: gestures/velocity_tracker.dart Velocity.operator-
func (v Velocity) Sub(other Velocity) Velocity {
	return Velocity{v.PixelsPerSecond.Sub(other.PixelsPerSecond)}
}

// Add returns the sum of the velocities.
//
// Flutter: gestures/velocity_tracker.dart Velocity.operator+
func (v Velocity) Add(other Velocity) Velocity {
	return Velocity{v.PixelsPerSecond.Add(other.PixelsPerSecond)}
}

// ClampMagnitude returns the velocity with its speed limited to the range
// [minValue, maxValue], in the same direction. The bounds must be zero or more
// and maxValue not less than minValue, or a debug build panics with the tag
// VelocityBoundsInvalid.
//
// Flutter: gestures/velocity_tracker.dart Velocity.clampMagnitude
func (v Velocity) ClampMagnitude(minValue, maxValue float64) Velocity {
	if foundation.DebugMode && (minValue < 0 || maxValue < 0 || maxValue < minValue) {
		panic(errors.New("VelocityBoundsInvalid", "the bounds must be zero or more and the maximum not below the minimum",
			"min", minValue, "max", maxValue))
	}
	valueSquared := v.PixelsPerSecond.DistanceSquared()
	if valueSquared > maxValue*maxValue {
		return Velocity{v.PixelsPerSecond.Div(v.PixelsPerSecond.Distance()).Mul(maxValue)}
	}
	if valueSquared < minValue*minValue {
		return Velocity{v.PixelsPerSecond.Div(v.PixelsPerSecond.Distance()).Mul(minValue)}
	}
	return v
}

// String describes the velocity.
//
// Flutter: gestures/velocity_tracker.dart Velocity.toString
func (v Velocity) String() string {
	return fmt.Sprintf("Velocity(%s, %s)", dartfmt.ToStringAsFixed(v.PixelsPerSecond.Dx, 1),
		dartfmt.ToStringAsFixed(v.PixelsPerSecond.Dy, 1))
}

// VelocityEstimate is an estimate of a velocity from the positions of a
// pointer, with how reliable it is.
//
// Flutter: gestures/velocity_tracker.dart VelocityEstimate
type VelocityEstimate struct {
	// PixelsPerSecond is the number of pixels per second in each direction.
	PixelsPerSecond ui.Offset
	// Confidence is the confidence of the estimate, from 0 to 1.
	Confidence float64
	// Duration is the time the samples span.
	Duration time.Duration
	// Offset is the distance the samples span.
	Offset ui.Offset
}

// String describes the estimate.
//
// Flutter: gestures/velocity_tracker.dart VelocityEstimate.toString
func (e VelocityEstimate) String() string {
	return fmt.Sprintf("VelocityEstimate(%s, %s; offset: %s, duration: %s, confidence: %s)",
		dartfmt.ToStringAsFixed(e.PixelsPerSecond.Dx, 1), dartfmt.ToStringAsFixed(e.PixelsPerSecond.Dy, 1),
		e.Offset, dartfmt.Duration(e.Duration), dartfmt.ToStringAsFixed(e.Confidence, 1))
}

type pointAtTime struct {
	point ui.Offset
	time  time.Duration
}

func (p pointAtTime) String() string {
	return fmt.Sprintf("_PointAtTime(%s at %s)", p.point, dartfmt.Duration(p.time))
}

// velocityTrackerMode is how a VelocityTracker estimates.
type velocityTrackerMode int

const (
	// modeLSQ fits a polynomial to the samples of the last 100 ms.
	modeLSQ velocityTrackerMode = iota
	// modeIOS gives the velocity of an iOS scroll view fling.
	modeIOS
	// modeMacOS gives the velocity of a macOS scroll view fling.
	modeMacOS
)

const (
	assumePointerMoveStoppedMilliseconds = 40
	velocityHistorySize                  = 20
	velocityHorizonMilliseconds          = 100
	velocityMinSampleSize                = 3
	iosSampleSize                        = 20
)

// VelocityTracker computes the velocity of a pointer from its positions. A
// recognizer feeds it the positions and asks for the velocity when the pointer
// is lifted.
//
// The three kinds of tracker of Flutter are one type here, chosen by the
// constructor: NewVelocityTrackerWithKind fits a polynomial to the last
// samples; NewIOSScrollViewFlingVelocityTracker and
// NewMacOSScrollViewFlingVelocityTracker estimate the fling velocity of an iOS
// or macOS scroll view.
//
// Deviation: Flutter's trackers are classes that extend VelocityTracker; Go's
// is one type with a mode (SPEC §12.10).
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker
type VelocityTracker struct {
	kind  ui.PointerDeviceKind
	mode  velocityTrackerMode
	clock SamplingClock

	// The time of the last sample, on the clock; valid if started.
	lastSample time.Time
	started    bool

	// samples is a circular buffer; the current sample is at index.
	samples      [velocityHistorySize]*pointAtTime
	index        int
	touchSamples [iosSampleSize]*pointAtTime
}

// NewVelocityTrackerWithKind creates a tracker for pointers of the kind. The
// clock tells how long ago the last position was added: with no recent sample,
// the velocity is zero.
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker.withKind
func NewVelocityTrackerWithKind(clock SamplingClock, kind ui.PointerDeviceKind) *VelocityTracker {
	return &VelocityTracker{kind: kind, clock: clock}
}

// NewIOSScrollViewFlingVelocityTracker creates a tracker that estimates the
// velocity the way an iOS scroll view does.
//
// Flutter: gestures/velocity_tracker.dart IOSScrollViewFlingVelocityTracker.new
func NewIOSScrollViewFlingVelocityTracker(clock SamplingClock, kind ui.PointerDeviceKind) *VelocityTracker {
	return &VelocityTracker{kind: kind, clock: clock, mode: modeIOS}
}

// NewMacOSScrollViewFlingVelocityTracker creates a tracker that estimates the
// velocity the way a macOS scroll view does.
//
// Flutter: gestures/velocity_tracker.dart MacOSScrollViewFlingVelocityTracker.new
func NewMacOSScrollViewFlingVelocityTracker(clock SamplingClock, kind ui.PointerDeviceKind) *VelocityTracker {
	return &VelocityTracker{kind: kind, clock: clock, mode: modeMacOS}
}

// Kind is the kind of pointer the tracker is for.
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker.kind
func (t *VelocityTracker) Kind() ui.PointerDeviceKind { return t.kind }

func (t *VelocityTracker) elapsedMilliseconds() int64 {
	if !t.started {
		return 0
	}
	return t.clock.Now().Sub(t.lastSample).Milliseconds()
}

func (t *VelocityTracker) markSample() {
	t.started = true
	t.lastSample = t.clock.Now()
}

// AddPosition adds a position at a time. The times must not go back.
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker.addPosition
func (t *VelocityTracker) AddPosition(tm time.Duration, position ui.Offset) {
	t.markSample()
	if t.mode == modeLSQ {
		t.index++
		if t.index == velocityHistorySize {
			t.index = 0
		}
		t.samples[t.index] = &pointAtTime{position, tm}
		return
	}
	if foundation.DebugMode {
		if prev := t.touchSamples[t.index]; prev != nil && prev.time > tm {
			panic(errors.New("VelocityPositionPrecedesPredecessor",
				"the position being added has a smaller timestamp than its predecessor",
				"position", position.String(), "time", dartfmt.Duration(tm), "previous", prev.String()))
		}
	}
	t.index = (t.index + 1) % iosSampleSize
	t.touchSamples[t.index] = &pointAtTime{position, tm}
}

// GetVelocityEstimate returns the estimate of the velocity, or None if there is
// no sample. If the last sample is older than 40 ms the estimate is zero.
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker.getVelocityEstimate
func (t *VelocityTracker) GetVelocityEstimate() foundation.Opt[VelocityEstimate] {
	if t.mode != modeLSQ {
		return foundation.Some(t.flingEstimate())
	}
	// Has user recently moved since last sample?
	if t.elapsedMilliseconds() > assumePointerMoveStoppedMilliseconds {
		return foundation.Some(VelocityEstimate{Confidence: 1})
	}

	var x, y, w, tm []float64
	sampleCount := 0
	index := t.index

	newest := t.samples[index]
	if newest == nil {
		return foundation.None[VelocityEstimate]()
	}
	previous, oldest := newest, newest

	// Starting with the most recent PointAtTime sample, iterate backwards while
	// the samples represent continuous motion.
	for {
		sample := t.samples[index]
		if sample == nil {
			break
		}
		age := float64((newest.time - sample.time).Microseconds()) / 1000
		delta := math.Abs(float64((sample.time - previous.time).Microseconds())) / 1000
		previous = sample
		if age > velocityHorizonMilliseconds || delta > assumePointerMoveStoppedMilliseconds {
			break
		}
		oldest = sample
		x = append(x, sample.point.Dx)
		y = append(y, sample.point.Dy)
		w = append(w, 1.0)
		tm = append(tm, -age)
		if index == 0 {
			index = velocityHistorySize
		}
		index--
		sampleCount++
		if sampleCount >= velocityHistorySize {
			break
		}
	}

	if sampleCount >= velocityMinSampleSize {
		xFit := NewLeastSquaresSolver(tm, x, w).Solve(2)
		var yFit *PolynomialFit
		if xFit != nil {
			yFit = NewLeastSquaresSolver(tm, y, w).Solve(2)
		}
		if xFit != nil && yFit != nil {
			return foundation.Some(VelocityEstimate{
				// convert from pixels/ms to pixels/s
				PixelsPerSecond: ui.Offset{Dx: xFit.Coefficients[1] * 1000, Dy: yFit.Coefficients[1] * 1000},
				Confidence:      xFit.Confidence * yFit.Confidence,
				Duration:        newest.time - oldest.time,
				Offset:          newest.point.Sub(oldest.point),
			})
		}
	}

	// We're unable to make a velocity estimate but we did have at least one
	// valid pointer position.
	return foundation.Some(VelocityEstimate{
		Confidence: 1,
		Duration:   newest.time - oldest.time,
		Offset:     newest.point.Sub(oldest.point),
	})
}

// previousVelocityAt computes the velocity using 2 adjacent points in history.
// When index is 0, it uses the latest point recorded and the point recorded
// immediately before it. The smaller index is, the earlier in history the points
// used are.
func (t *VelocityTracker) previousVelocityAt(index int) ui.Offset {
	endIndex := mod(t.index+index, iosSampleSize)
	startIndex := mod(t.index+index-1, iosSampleSize)
	end, start := t.touchSamples[endIndex], t.touchSamples[startIndex]
	if end == nil || start == nil {
		return ui.Offset{}
	}
	dt := (end.time - start.time).Microseconds()
	if foundation.DebugMode && dt < 0 {
		panic(errors.New("VelocityTimesNotIncreasing", "the samples go back in time", "dt", dt))
	}
	if dt > 0 {
		// Convert dt to milliseconds to preserve floating point precision.
		return end.point.Sub(start.point).Mul(1000).Div(float64(dt) / 1000)
	}
	return ui.Offset{}
}

func mod(a, n int) int { return ((a % n) + n) % n }

func (t *VelocityTracker) flingEstimate() VelocityEstimate {
	// Has user recently moved since last sample?
	if t.elapsedMilliseconds() > assumePointerMoveStoppedMilliseconds {
		return VelocityEstimate{Confidence: 1}
	}

	// The velocity estimated using this expression is an approximation of the
	// scroll velocity of the scroll view at the moment the user touch was
	// released, not the final velocity the pan gesture recognizer installed on
	// the scroll view would report. Typically in a scroll view the velocity
	// values are different between the two, because the scroll view usually
	// slows down when the touch is released.
	var estimated ui.Offset
	if t.mode == modeIOS {
		estimated = t.previousVelocityAt(-2).Mul(0.6).Add(t.previousVelocityAt(-1).Mul(0.35)).Add(t.previousVelocityAt(0).Mul(0.05))
	} else {
		estimated = t.previousVelocityAt(-2).Mul(0.15).Add(t.previousVelocityAt(-1).Mul(0.65)).Add(t.previousVelocityAt(0).Mul(0.2))
	}

	newest := t.touchSamples[t.index]
	var oldest *pointAtTime
	for i := 1; i <= iosSampleSize; i++ {
		oldest = t.touchSamples[(t.index+i)%iosSampleSize]
		if oldest != nil {
			break
		}
	}
	if oldest == nil || newest == nil {
		if foundation.DebugMode {
			panic(errors.New("VelocityNoSamples", "there must be at least 1 point in the samples"))
		}
		return VelocityEstimate{}
	}
	return VelocityEstimate{
		PixelsPerSecond: estimated,
		Confidence:      1,
		Duration:        newest.time - oldest.time,
		Offset:          newest.point.Sub(oldest.point),
	}
}

// GetVelocity returns the velocity, which is zero if no estimate can be made.
//
// Flutter: gestures/velocity_tracker.dart VelocityTracker.getVelocity
func (t *VelocityTracker) GetVelocity() Velocity {
	estimate, ok := t.GetVelocityEstimate().Get()
	if !ok || estimate.PixelsPerSecond == (ui.Offset{}) {
		return VelocityZero
	}
	return Velocity{estimate.PixelsPerSecond}
}
