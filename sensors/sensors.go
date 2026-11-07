// Package sensors gives an app the motion sensors of the device (SPEC §28.4): the accelerometer and the gyroscope of a phone or of a
// browser on one, and the sensors of a gamepad (package gamepad). Flutter has no such package in its framework (the sensors_plus
// plugin is the closest); the names here are Glutter's.
//
// A Sensor is opened with System.Open, keeps its latest reading, and calls its listeners with the readings that arrived since the
// last frame, once a frame, on the UI thread. The axes are SDL's: x to the right, y up and z out of the screen, for a device in its
// natural orientation.
package sensors

import (
	"context"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/widgets"
)

// Kind is the kind of a motion sensor.
//
// Flutter: no counterpart
type Kind uint8

const (
	// Accelerometer measures the acceleration of the device with gravity included, in m/s². A device that lies flat on a table,
	// face up, reads about 9.8 on z.
	//
	// Flutter: no counterpart
	Accelerometer Kind = iota
	// Gyroscope measures the rate of rotation about the three axes, in rad/s, counterclockwise as seen from the positive end of
	// an axis.
	//
	// Flutter: no counterpart
	Gyroscope
)

// String returns "accelerometer" or "gyroscope".
//
// Flutter: no counterpart
func (k Kind) String() string {
	switch k {
	case Accelerometer:
		return "accelerometer"
	case Gyroscope:
		return "gyroscope"
	}
	return "unknown"
}

// Reading is one reading of a sensor.
//
// Flutter: no counterpart
type Reading struct {
	// Timestamp is when the reading was made, on a monotonic clock whose start is not defined: only the difference of two
	// readings of the same sensor means something.
	Timestamp time.Duration
	// Values are x, y and z, in the unit of the kind (m/s² or rad/s).
	Values [3]float64
}

// Opts are the options of Open.
//
// Flutter: no counterpart
type Opts struct {
	// Rate is the number of readings a second that the app would like. 0 leaves it to the platform (60 a second, or the fastest
	// that the sensor can do if that is slower); a browser sends events at its own rate whatever is asked.
	Rate float64
}

// maxPending is the most readings that a sensor keeps for its listeners while no frame runs (an app that is hidden, or a program
// that does not draw): older readings are dropped, so that the memory does not grow.
const maxPending = 4096

// System is the motion sensors of the app. Get it with Of. It is made when it is first asked for, and closed with the app.
//
// Flutter: no counterpart
type System struct {
	plat   platform.Sensors
	post   func(func())
	sched  *scheduler.SchedulerBinding
	report foundation.ErrorReporter

	mu      sync.Mutex
	sensors map[*Sensor]struct{}
	closed  bool
}

type systemKey struct{}

// Of returns the sensors of the app (the ones of the binding). An app on a platform that has no sensors service gets a System whose
// calls fail with SensorUnavailable.
//
// Flutter: no counterpart
func Of(bc widgets.BuildContext) *System { return ForBinding(widgets.WidgetsBindingOf(bc)) }

// ForBinding returns the sensors of the binding, for code that has no BuildContext.
//
// Flutter: no counterpart
func ForBinding(b *widgets.WidgetsBinding) *System {
	return b.Extension(systemKey{}, func() any {
		var host platform.SensorsHost
		var plat platform.Sensors
		if b.PlatformDispatcher().HostAs(&host) {
			plat = host.Sensors()
		}
		s := &System{plat: plat, post: b.Post, sched: b.Scheduler(), report: b.ErrorReporter(), sensors: map[*Sensor]struct{}{}}
		if host != nil && plat != nil {
			host.AddReleaseHook(s.Close)
		}
		return s
	}).(*System)
}

// Available says whether the platform has a sensors service. It does not say that the device has the sensor that the app wants:
// Has does, and Open fails when it has not.
//
// Flutter: no counterpart
func (s *System) Available() bool { return s.plat != nil }

// Has says whether the device has a sensor of the kind. A browser cannot know before it sends the first reading, and answers yes
// where it has the events.
//
// Flutter: no counterpart
func (s *System) Has(kind Kind) bool { return s.plat != nil && s.plat.Available(platformKind(kind)) }

func platformKind(k Kind) platform.SensorKind {
	if k == Gyroscope {
		return platform.SensorGyroscope
	}
	return platform.SensorAccelerometer
}

// Open opens a sensor and starts it. It fails with SensorUnavailable when the platform has no sensors service or the device has no
// such sensor, SensorPermissionDenied when the user refuses it, and SensorOpenFailed for the rest (also when ctx ends first). Where
// the system asks the user (a browser on some systems) the call waits for the answer, until ctx ends: call it from a goroutine that
// is not the UI one. At most one Opts may be given.
//
// Flutter: no counterpart
func (s *System) Open(ctx context.Context, kind Kind, opts ...Opts) (*Sensor, error) {
	if len(opts) > 1 {
		panic(errors.New(tagMultipleOptions, "Open takes at most one Opts", "count", len(opts)))
	}
	var o Opts
	if len(opts) == 1 {
		o = opts[0]
	}
	if s.plat == nil {
		return nil, errors.New(platform.TagSensorUnavailable, "the platform has no motion sensors service", "kind", kind.String())
	}
	sensor := newSensor(s, kind)
	plat, err := s.plat.Open(ctx, platformKind(kind), platform.SensorOpenOptions{Rate: o.Rate}, sensor)
	if err != nil {
		return nil, err
	}
	sensor.plat = plat
	sensor.rate = plat.Rate()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		if err := plat.Close(); err != nil {
			s.fail(err)
		}
		return nil, errors.New(platform.TagSensorUnavailable, "the sensors of the app were closed", "kind", kind.String())
	}
	s.sensors[sensor] = struct{}{}
	s.mu.Unlock()
	return sensor, nil
}

// fail reports an error that nobody can return: the close of a sensor when the app ends.
func (s *System) fail(err error) {
	if s.report != nil {
		s.report(err)
	}
}

// Close closes every sensor that the app left open. The engine does it when the app ends.
//
// Flutter: no counterpart
func (s *System) Close() {
	s.mu.Lock()
	s.closed = true
	list := make([]*Sensor, 0, len(s.sensors))
	for x := range s.sensors {
		list = append(list, x)
	}
	s.mu.Unlock()
	for _, x := range list {
		if err := x.Close(); err != nil {
			s.fail(err)
		}
	}
}

// Sensor is an open sensor. It is a platform.SensorHandler: the backend calls it on its own thread.
//
// Flutter: no counterpart
type Sensor struct {
	sys  *System
	kind Kind
	plat platform.SensorDevice
	rate float64
	// onClose is what Close does besides ending the platform's sensor: a sensor made by a Feed has no platform sensor.
	onClose func() error

	mu        sync.Mutex
	pending   []Reading
	spare     []Reading
	scheduled bool
	latest    Reading
	has       bool
	dropped   int
	closed    bool
	listeners []listener
	nextID    int
}

type listener struct {
	id int
	fn func(readings []Reading)
}

var _ platform.SensorHandler = (*Sensor)(nil)

func newSensor(sys *System, kind Kind) *Sensor { return &Sensor{sys: sys, kind: kind} }

// Kind says what the sensor measures.
//
// Flutter: no counterpart
func (s *Sensor) Kind() Kind { return s.kind }

// Rate is the number of readings a second that the sensor was set to, or 0 when the platform does not say (a browser).
//
// Flutter: no counterpart
func (s *Sensor) Rate() float64 { return s.rate }

// Latest returns the last reading and whether there is one yet. It may be called from any goroutine.
//
// Flutter: no counterpart
func (s *Sensor) Latest() (Reading, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest, s.has
}

// Dropped is how many readings were thrown away because no frame ran to deliver them (see maxPending).
//
// Flutter: no counterpart
func (s *Sensor) Dropped() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

// AddListener adds a function that is called, on the UI thread, once a frame, with the readings that arrived since the last call,
// oldest first. The slice is valid only during the call. It returns the function that removes the listener. A sensor with no
// listener schedules no frames.
//
// Flutter: no counterpart
func (s *Sensor) AddListener(fn func(readings []Reading)) (remove func()) {
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.listeners = append(s.listeners, listener{id: id, fn: fn})
	again := len(s.pending) > 0 && !s.scheduled && !s.closed
	if again {
		s.scheduled = true
	}
	s.mu.Unlock()
	if again {
		s.sys.post(s.scheduleDelivery)
	}
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, l := range s.listeners {
			if l.id == id {
				s.listeners = append(s.listeners[:i:i], s.listeners[i+1:]...)
				return
			}
		}
	}
}

// SensorReadings implements platform.SensorHandler. It runs on the backend's thread: it keeps the readings and asks the UI thread
// for a frame to deliver them in.
func (s *Sensor) SensorReadings(in []platform.SensorReading) {
	if len(in) == 0 {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	for _, r := range in {
		s.pending = append(s.pending, Reading{Timestamp: r.Time, Values: r.Values})
	}
	if over := len(s.pending) - maxPending; over > 0 {
		s.pending = append(s.pending[:0], s.pending[over:]...)
		s.dropped += over
	}
	s.latest, s.has = s.pending[len(s.pending)-1], true
	schedule := !s.scheduled && len(s.listeners) > 0
	if schedule {
		s.scheduled = true
	}
	s.mu.Unlock()
	if schedule {
		s.sys.post(s.scheduleDelivery)
	}
}

// scheduleDelivery runs on the UI thread: the readings are given to the listeners in the next frame.
func (s *Sensor) scheduleDelivery() { s.sys.sched.ScheduleFrameCallback(s.deliver) }

// deliver runs in a frame, on the UI thread.
func (s *Sensor) deliver(time.Duration) {
	s.mu.Lock()
	s.scheduled = false
	if s.closed || len(s.pending) == 0 {
		s.mu.Unlock()
		return
	}
	batch := s.pending
	s.pending = s.spare[:0]
	listeners := append([]listener(nil), s.listeners...)
	s.mu.Unlock()
	for _, l := range listeners {
		l.fn(batch)
	}
	s.mu.Lock()
	s.spare = batch[:0]
	s.mu.Unlock()
}

// Close ends the sensor: no listener is called after it returns. It may be called more than once.
//
// Flutter: no counterpart
func (s *Sensor) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.pending = nil
	s.listeners = nil
	s.mu.Unlock()
	s.sys.mu.Lock()
	delete(s.sys.sensors, s)
	s.sys.mu.Unlock()
	if s.plat != nil {
		if err := s.plat.Close(); err != nil {
			return err
		}
	}
	if s.onClose != nil {
		return s.onClose()
	}
	return nil
}

// Feed gives readings to a Sensor that the platform's sensors service does not make: the sensors of a gamepad (package gamepad) are
// made this way. An app that wants a Sensor of its own (a recording that it plays back, a test) may make one too.
//
// Flutter: no counterpart
type Feed struct {
	sensor *Sensor
}

// NewFeed makes a Feed whose Sensor is of the kind, with the rate that the source makes (0 when it does not say). onClose, which may
// be nil, runs when the sensor is closed.
//
// Flutter: no counterpart
func NewFeed(b *widgets.WidgetsBinding, kind Kind, rate float64, onClose func() error) *Feed {
	sensor := newSensor(ForBinding(b), kind)
	sensor.rate = rate
	sensor.onClose = onClose
	sys := sensor.sys
	sys.mu.Lock()
	sys.sensors[sensor] = struct{}{}
	sys.mu.Unlock()
	return &Feed{sensor: sensor}
}

// Sensor is the sensor that the feed's readings reach.
//
// Flutter: no counterpart
func (f *Feed) Sensor() *Sensor { return f.sensor }

// Put gives readings to the sensor, from any goroutine. A closed sensor gets nothing.
//
// Flutter: no counterpart
func (f *Feed) Put(readings ...Reading) {
	in := make([]platform.SensorReading, len(readings))
	for i, r := range readings {
		in[i] = platform.SensorReading{Time: r.Timestamp, Values: r.Values}
	}
	f.sensor.SensorReadings(in)
}

// The tags of the errors that this package makes (rm4n0s/errors, R5).
const (
	tagMultipleOptions = "SensorsMultipleOptions"
)
