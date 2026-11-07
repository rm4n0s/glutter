package fake

import (
	"context"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Sensors is the scripted motion sensors service of the fake platform: a test says which sensors the device has, opens them
// through the app, and plays the part of the sensor thread by sending readings to an open sensor (SPEC §28.8). Nothing runs by
// itself, so a test decides when the readings arrive.
//
// Flutter: no counterpart
type Sensors struct {
	mu      sync.Mutex
	missing map[platform.SensorKind]bool
	denied  map[platform.SensorKind]bool
	openErr error
	opened  []*SensorDevice
}

// NewSensors makes the service with an accelerometer and a gyroscope.
//
// Flutter: no counterpart
func NewSensors() *Sensors {
	return &Sensors{missing: map[platform.SensorKind]bool{}, denied: map[platform.SensorKind]bool{}}
}

// Sensors returns the scripted sensors service, which Services gives the engine.
//
// Flutter: no counterpart
func (p *Platform) Sensors() *Sensors {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sensors == nil {
		p.sensors = NewSensors()
	}
	return p.sensors
}

// SetAvailable says whether the device has a sensor of the kind (both are there at the start).
//
// Flutter: no counterpart
func (s *Sensors) SetAvailable(kind platform.SensorKind, available bool) {
	s.mu.Lock()
	s.missing[kind] = !available
	s.mu.Unlock()
}

// SetDenied makes the user refuse the sensor of the kind: opening it fails with SensorPermissionDenied.
//
// Flutter: no counterpart
func (s *Sensors) SetDenied(kind platform.SensorKind, denied bool) {
	s.mu.Lock()
	s.denied[kind] = denied
	s.mu.Unlock()
}

// FailOpen makes the next opens fail with err (nil makes them work again).
//
// Flutter: no counterpart
func (s *Sensors) FailOpen(err error) {
	s.mu.Lock()
	s.openErr = err
	s.mu.Unlock()
}

// Available implements platform.Sensors.
func (s *Sensors) Available(kind platform.SensorKind) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.missing[kind]
}

// Opened returns the sensors that are open now, in the order they were opened.
//
// Flutter: no counterpart
func (s *Sensors) Opened() []*SensorDevice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*SensorDevice(nil), s.opened...)
}

// Open implements platform.Sensors.
func (s *Sensors) Open(ctx context.Context, kind platform.SensorKind, opts platform.SensorOpenOptions, h platform.SensorHandler) (platform.SensorDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.missing[kind] {
		return nil, errors.New(platform.TagSensorUnavailable, "the fake device has no such sensor", "kind", int(kind))
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagSensorOpenFailed, err).SetMetadata("kind", int(kind))
	}
	if s.denied[kind] {
		return nil, errors.New(platform.TagSensorPermissionDenied, "the fake user refused the sensor", "kind", int(kind))
	}
	if s.openErr != nil {
		return nil, s.openErr
	}
	rate := opts.Rate
	if rate == 0 {
		rate = 60
	}
	d := &SensorDevice{s: s, kind: kind, h: h, opts: opts, rate: rate}
	s.opened = append(s.opened, d)
	return d, nil
}

// SensorDevice is a sensor that the app opened on the fake service.
//
// Flutter: no counterpart
type SensorDevice struct {
	s    *Sensors
	kind platform.SensorKind
	h    platform.SensorHandler
	opts platform.SensorOpenOptions
	rate float64

	mu     sync.Mutex
	closed bool
}

var _ platform.SensorDevice = (*SensorDevice)(nil)

// Kind is the kind of the sensor.
//
// Flutter: no counterpart
func (d *SensorDevice) Kind() platform.SensorKind { return d.kind }

// Options are what the app asked for when it opened the sensor.
//
// Flutter: no counterpart
func (d *SensorDevice) Options() platform.SensorOpenOptions { return d.opts }

// Rate implements platform.SensorDevice: what the app asked for, or 60.
func (d *SensorDevice) Rate() float64 { return d.rate }

// Closed says whether the app closed the sensor.
//
// Flutter: no counterpart
func (d *SensorDevice) Closed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

// Send makes readings arrive, on the caller's goroutine, as the sensor thread of a backend would. A sensor that was closed gets
// nothing. Values are given for the sensor's kind.
//
// Flutter: no counterpart
func (d *SensorDevice) Send(readings ...platform.SensorReading) {
	d.mu.Lock()
	closed := d.closed
	d.mu.Unlock()
	if closed || len(readings) == 0 {
		return
	}
	d.h.SensorReadings(readings)
}

// SendValues makes one reading arrive with the time at which the call was made, counted from the start of the fake clock that the
// caller passes.
//
// Flutter: no counterpart
func (d *SensorDevice) SendValues(at time.Duration, x, y, z float64) {
	d.Send(platform.SensorReading{Time: at, Values: [3]float64{x, y, z}})
}

// Close implements platform.SensorDevice.
func (d *SensorDevice) Close() error {
	d.mu.Lock()
	d.closed = true
	d.mu.Unlock()
	d.s.mu.Lock()
	for i, o := range d.s.opened {
		if o == d {
			d.s.opened = append(d.s.opened[:i:i], d.s.opened[i+1:]...)
			break
		}
	}
	d.s.mu.Unlock()
	return nil
}
