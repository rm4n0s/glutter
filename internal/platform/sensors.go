package platform

import (
	"context"
	"time"
)

// SensorKind is the kind of a motion sensor of the device (SPEC §28.4).
//
// Flutter: no counterpart
type SensorKind uint8

const (
	// SensorAccelerometer measures the acceleration of the device with gravity included, in m/s², along the axes of the device in
	// its natural orientation: x to the right, y up and z out of the screen. A device that lies flat on a table, face up, reads
	// about 9.8 on z.
	//
	// Flutter: no counterpart
	SensorAccelerometer SensorKind = iota
	// SensorGyroscope measures the rate of rotation about the same axes, in rad/s, counterclockwise as seen from the positive
	// end of the axis (the right-hand rule).
	//
	// Flutter: no counterpart
	SensorGyroscope
)

// SensorReading is one reading of a sensor.
//
// Flutter: no counterpart
type SensorReading struct {
	// Time is when the reading was made, on a monotonic clock of the platform (nanoseconds from an arbitrary start): only the
	// difference of two readings of the same sensor means something.
	Time time.Duration
	// Values are x, y and z, in the unit and the axes of the kind.
	Values [3]float64
}

// SensorHandler is what a sensor calls with the readings it made. It may be called on any thread.
//
// Flutter: no counterpart
type SensorHandler interface {
	// SensorReadings gets the readings made since the last call, oldest first. The slice is valid only during the call.
	SensorReadings(readings []SensorReading)
}

// SensorOpenOptions are what an app asks of a sensor when it opens it.
//
// Flutter: no counterpart
type SensorOpenOptions struct {
	// Rate is the number of readings a second that the app would like; 0 leaves it to the backend (60, or the fastest that the
	// sensor can do if that is slower). A backend that cannot choose (the web's, which gets an event whenever the browser has
	// one) ignores it; SensorDevice.Rate says what the device does.
	Rate float64
}

// SensorDevice is a sensor that is open. The handler it was opened with is called until Close returns.
//
// Flutter: no counterpart
type SensorDevice interface {
	// Rate is the number of readings a second that the sensor was set to, or 0 when the backend does not know it.
	Rate() float64
	// Close ends the sensor. When it returns the handler is not running and is never called again.
	Close() error
}

// Sensors is the motion sensors service of a platform (SPEC §28.4).
//
// Flutter: no counterpart
type Sensors interface {
	// Available says whether the device has a sensor of the kind (the web cannot know before the first reading and says yes
	// where the browser has the events).
	Available(kind SensorKind) bool
	// Open opens a sensor and starts it. It fails with SensorUnavailable when the device has no such sensor, SensorPermissionDenied
	// when the user refused it, and SensorOpenFailed for the rest. Where the system asks the user, the call waits for the answer
	// until ctx ends.
	Open(ctx context.Context, kind SensorKind, opts SensorOpenOptions, h SensorHandler) (SensorDevice, error)
}

// SensorsHost is implemented by the host of the dispatcher (the engine) and found through ui.PlatformDispatcher.HostAs: it gives
// access to the sensors service of the platform, or nil when the platform has none.
//
// Flutter: no counterpart
type SensorsHost interface {
	// Sensors returns the sensors service, or nil.
	Sensors() Sensors
	// AddReleaseHook registers a function that the host calls when the app ends, so that a sensor that is still open is closed
	// before the process leaves. The function runs on the UI thread.
	AddReleaseHook(fn func())
}

// The tags of the errors a Sensors service returns.
const (
	// TagSensorUnavailable is the tag of the error of a sensor that the device does not have, or of a platform that has no sensors
	// service.
	//
	// Flutter: no counterpart
	TagSensorUnavailable = "SensorUnavailable"
	// TagSensorPermissionDenied is the tag of the error of a sensor that the user did not allow.
	//
	// Flutter: no counterpart
	TagSensorPermissionDenied = "SensorPermissionDenied"
	// TagSensorOpenFailed is the tag of the error of a sensor that cannot be opened.
	//
	// Flutter: no counterpart
	TagSensorOpenFailed = "SensorOpenFailed"
)
