package web

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The motion sensors and the haptics of the web (SPEC §28.4, step 31.7). The calls into the browser are in sensors_js.go; this file
// has what needs no browser, so that a test runs it anywhere: how a DeviceMotionEvent becomes the readings of the toolkit's sensors
// (units and axes), and the lengths of the vibrations of navigator.vibrate.

// MotionEvent is what the platform reads from a DeviceMotionEvent (https://w3c.github.io/deviceorientation/): the acceleration with
// gravity included in m/s² on the axes of the device (x to the right, y toward the top of the screen, z out of the screen), the
// rate of rotation in degrees a second about the axes z (alpha), x (beta) and y (gamma), the time of the event in milliseconds and
// the interval between two events in milliseconds. A browser that has no such sensor sends events whose members are null; HasAcceleration and
// HasRotation say that the three numbers are there.
//
// Flutter: no counterpart
type MotionEvent struct {
	HasAcceleration bool
	Acceleration    [3]float64 // x, y, z
	HasRotation     bool
	Alpha           float64
	Beta            float64
	Gamma           float64
	// TimeStamp is Event.timeStamp, a time in milliseconds on the page's clock, and Interval is DeviceMotionEvent.interval.
	TimeStamp, Interval float64
}

// degreesToRadians turns the browser's rate of rotation into rad/s.
const degreesToRadians = math.Pi / 180

// MotionReading is the reading of the sensor of the kind that the event holds, in the unit and on the axes of the toolkit
// (m/s² and rad/s; x to the right, y up, z out of the screen, which are the browser's axes for the acceleration, and for the
// rotation x is beta, y is gamma and z is alpha). It is false when the event has none for the kind, or has a number that is not
// finite.
//
// Flutter: no counterpart
func MotionReading(kind platform.SensorKind, e MotionEvent) (platform.SensorReading, bool) {
	var v [3]float64
	switch kind {
	case platform.SensorAccelerometer:
		if !e.HasAcceleration {
			return platform.SensorReading{}, false
		}
		v = e.Acceleration
	case platform.SensorGyroscope:
		if !e.HasRotation {
			return platform.SensorReading{}, false
		}
		v = [3]float64{e.Beta * degreesToRadians, e.Gamma * degreesToRadians, e.Alpha * degreesToRadians}
	default:
		return platform.SensorReading{}, false
	}
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return platform.SensorReading{}, false
		}
	}
	return platform.SensorReading{Time: time.Duration(e.TimeStamp * float64(time.Millisecond)), Values: v}, true
}

// MotionRate is the number of events a second that an interval in milliseconds means, or 0 when the browser gives none.
//
// Flutter: no counterpart
func MotionRate(intervalMillis float64) float64 {
	if intervalMillis > 0 && !math.IsInf(intervalMillis, 0) && !math.IsNaN(intervalMillis) {
		return 1000 / intervalMillis
	}
	return 0
}

// HapticMillis is the length of the vibration of navigator.vibrate that a kind of haptic feedback is, in milliseconds, or 0 for a
// kind that makes none. The lengths are Glutter's: a phone's browser has only a motor that is on or off, so the lengths tell the
// kinds apart (the browser rounds them to what the motor can do).
//
// Deviation: Flutter asks the platform for its own feedback (a click of the screen on Android, an impact generator on iOS); the web
// has the length alone (SPEC §12.10).
//
// Flutter: services/haptic_feedback.dart HapticFeedback
func HapticMillis(kind ui.HapticFeedbackKind) int {
	switch kind {
	case ui.HapticFeedbackVibrate:
		return 50
	case ui.HapticFeedbackLightImpact:
		return 10
	case ui.HapticFeedbackMediumImpact:
		return 20
	case ui.HapticFeedbackHeavyImpact:
		return 40
	case ui.HapticFeedbackSelectionClick:
		return 5
	}
	return 0
}
