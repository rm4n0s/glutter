package gluttertest

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/sensors"
)

// TestSensors is the scripted motion sensors service of the test platform (SPEC §28.8): the device has an accelerometer and a
// gyroscope, and a test plays the part of the sensor thread by sending readings to the sensor that the app opened. The app's
// listeners get them in the next frame (Pump), as they would on a phone. Nothing arrives by itself.
//
// Flutter: no counterpart
type TestSensors struct {
	t *Tester
	s *fake.Sensors
}

// Sensors returns the scripted sensors service of the tester.
//
// Flutter: no counterpart
func (t *Tester) Sensors() *TestSensors { return &TestSensors{t: t, s: t.sensorsvc} }

func platformKind(k sensors.Kind) platform.SensorKind {
	if k == sensors.Gyroscope {
		return platform.SensorGyroscope
	}
	return platform.SensorAccelerometer
}

// SetAvailable says whether the device has a sensor of the kind (both are there at the start).
//
// Flutter: no counterpart
func (s *TestSensors) SetAvailable(kind sensors.Kind, available bool) {
	s.s.SetAvailable(platformKind(kind), available)
}

// SetDenied makes the user refuse the sensor of the kind: opening it fails with SensorPermissionDenied.
//
// Flutter: no counterpart
func (s *TestSensors) SetDenied(kind sensors.Kind, denied bool) {
	s.s.SetDenied(platformKind(kind), denied)
}

// FailOpen makes the next opens fail with err; nil makes them work again.
//
// Flutter: no counterpart
func (s *TestSensors) FailOpen(err error) { s.s.FailOpen(err) }

// Open returns the first open sensor of the kind that the app has, and fails the test when there is none.
//
// Flutter: no counterpart
func (s *TestSensors) Open(kind sensors.Kind) *TestSensor {
	s.t.tb.Helper()
	for _, d := range s.s.Opened() {
		if d.Kind() == platformKind(kind) {
			return &TestSensor{t: s.t, d: d}
		}
	}
	s.t.tb.Fatalf("the app has no %v sensor open", kind)
	return nil
}

// Count is how many sensors the app has open now.
//
// Flutter: no counterpart
func (s *TestSensors) Count() int { return len(s.s.Opened()) }

// TestSensor is a sensor that the app opened, seen from the side of the backend.
//
// Flutter: no counterpart
type TestSensor struct {
	t *Tester
	d *fake.SensorDevice
}

// Rate is what the app asked for, or 60.
//
// Flutter: no counterpart
func (s *TestSensor) Rate() float64 { return s.d.Rate() }

// RequestedRate is the rate that the app asked for when it opened the sensor, 0 for none.
//
// Flutter: no counterpart
func (s *TestSensor) RequestedRate() float64 { return s.d.Options().Rate }

// Closed says whether the app closed the sensor.
//
// Flutter: no counterpart
func (s *TestSensor) Closed() bool { return s.d.Closed() }

// Send makes readings arrive, as the backend's thread would send them. The app's listeners are called by the next Pump.
//
// Flutter: no counterpart
func (s *TestSensor) Send(readings ...sensors.Reading) {
	s.t.adopt()
	in := make([]platform.SensorReading, len(readings))
	for i, r := range readings {
		in[i] = platform.SensorReading{Time: r.Timestamp, Values: r.Values}
	}
	s.d.Send(in...)
	s.t.host.RunPosted()
}

// SendValues makes one reading arrive at the time given (counted from any start).
//
// Flutter: no counterpart
func (s *TestSensor) SendValues(at time.Duration, x, y, z float64) {
	s.Send(sensors.Reading{Timestamp: at, Values: [3]float64{x, y, z}})
}
