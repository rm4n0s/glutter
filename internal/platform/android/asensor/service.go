//go:build linux

package asensor

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Options are what New is given.
type Options struct {
	// Package is the package name of the app, which the sensor manager of the NDK wants (it checks the app's permission to read
	// the sensors fast against it). Empty means the name of the process, which on Android is the package.
	Package string
	// Log gets what cannot be returned (nil discards it).
	Log *slog.Logger
}

// defaultRate is the readings a second that a sensor is set to when the app asks for none (SDL's choice too).
const defaultRate = 60

// maxEvents is how many events are taken from a queue with one call.
const maxEvents = 16

// Service is the sensors backend: it implements platform.Sensors.
//
// Flutter: no counterpart
type Service struct {
	pkg  string
	log  *slog.Logger
	load func() (*library, error)

	mu      sync.Mutex
	lib     *library
	mgr     uintptr
	loop    *looper
	devices map[*device]struct{}
	closed  bool
}

var _ platform.Sensors = (*Service)(nil)

// New makes a Service. libandroid is not loaded yet.
//
// Flutter: no counterpart
func New(opts Options) *Service {
	log := opts.Log
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	pkg := opts.Package
	if pkg == "" {
		pkg = processName()
	}
	return &Service{pkg: pkg, log: log, load: loadLibrary, devices: map[*device]struct{}{}}
}

// processName is the name of the process, from /proc/self/cmdline: the package of an Android app.
func processName() string {
	b, err := os.ReadFile("/proc/self/cmdline")
	if err != nil {
		return ""
	}
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// manager loads libandroid and gets the sensor manager on first use. It fails with SensorUnavailable when the library cannot be
// loaded or the manager is not there.
func (s *Service) manager() (*library, uintptr, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, 0, errors.New(platform.TagSensorUnavailable, "the sensors of the app were closed")
	}
	if s.lib == nil {
		lib, err := s.load()
		if err != nil {
			return nil, 0, err
		}
		pkg := dl.CString(s.pkg)
		mgr := lib.getInstanceForPackage(pkg)
		dl.Free(pkg)
		if mgr == 0 {
			return nil, 0, errors.New(platform.TagSensorUnavailable, "the system gave no sensor manager", "package", s.pkg)
		}
		s.lib, s.mgr = lib, mgr
	}
	return s.lib, s.mgr, nil
}

func typeOf(kind platform.SensorKind) int32 {
	if kind == platform.SensorGyroscope {
		return typeGyroscope
	}
	return typeAccelerometer
}

// Available says whether the device has a default sensor of the kind.
//
// Flutter: no counterpart
func (s *Service) Available(kind platform.SensorKind) bool {
	lib, mgr, err := s.manager()
	if err != nil {
		return false
	}
	return lib.getDefaultSensor(mgr, typeOf(kind)) != 0
}

// Open opens the default sensor of the kind and starts it. It fails with SensorUnavailable when libandroid is not there or the device
// has no such sensor, and SensorOpenFailed for the rest.
//
// Flutter: no counterpart
func (s *Service) Open(ctx context.Context, kind platform.SensorKind, opts platform.SensorOpenOptions, h platform.SensorHandler) (platform.SensorDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagSensorOpenFailed, err).SetMetadata("kind", int(kind))
	}
	lib, mgr, err := s.manager()
	if err != nil {
		return nil, err
	}
	sensor := lib.getDefaultSensor(mgr, typeOf(kind))
	if sensor == 0 {
		return nil, errors.New(platform.TagSensorUnavailable, "the device has no such sensor", "kind", int(kind))
	}
	loop, err := s.looper(lib)
	if err != nil {
		return nil, err
	}
	queue := lib.createEventQueue(mgr, loop.ptr, looperIdent, 0, 0)
	if queue == 0 {
		return nil, errors.New(platform.TagSensorOpenFailed, "the sensor manager made no event queue", "kind", int(kind))
	}
	if r := lib.enableSensor(queue, sensor); r < 0 {
		lib.destroyEventQueue(mgr, queue)
		return nil, errors.New(platform.TagSensorOpenFailed, "the sensor could not be enabled", "kind", int(kind), "result", int(r))
	}
	rate := opts.Rate
	if rate <= 0 {
		rate = defaultRate
	}
	usec := periodMicros(rate, lib.getMinDelay(sensor))
	lib.setEventRate(queue, sensor, usec)
	d := &device{svc: s, lib: lib, mgr: mgr, kind: kind, h: h, sensor: sensor, queue: queue, rate: 1e6 / float64(usec)}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		lib.disableSensor(queue, sensor)
		lib.destroyEventQueue(mgr, queue)
		return nil, errors.New(platform.TagSensorUnavailable, "the sensors of the app were closed")
	}
	s.devices[d] = struct{}{}
	s.mu.Unlock()
	return d, nil
}

// periodMicros is the time between two readings, in microseconds, for the rate asked: not shorter than the least that the sensor
// takes (its minimum delay, 0 when it has none, and a negative number for a sensor that reports only when its value changes).
func periodMicros(rate float64, minDelay int32) int32 {
	usec := int32(1e6 / rate)
	if usec < 1 {
		usec = 1
	}
	if minDelay > 0 && usec < minDelay {
		usec = minDelay
	}
	return usec
}

// looper starts the thread that reads the events, on first use.
func (s *Service) looper(lib *library) (*looper, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loop != nil {
		return s.loop, nil
	}
	l := &looper{svc: s, lib: lib, ready: make(chan struct{}), done: make(chan struct{})}
	l.events = dl.Malloc(maxEvents * eventSize)
	go l.run()
	<-l.ready
	if l.ptr == 0 {
		dl.Free(l.events)
		return nil, errors.New(platform.TagSensorOpenFailed, "the sensor thread has no looper")
	}
	s.loop = l
	return l, nil
}

// Close closes every sensor that is open and ends the thread of the looper.
//
// Flutter: no counterpart
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	list := make([]*device, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	loop := s.loop
	s.loop = nil
	s.mu.Unlock()
	for _, d := range list {
		if err := d.Close(); err != nil {
			s.log.Debug("sensors: a sensor could not be closed", "err", err)
		}
	}
	if loop != nil {
		loop.stop()
	}
}

// looper is the thread that waits for the events of every queue and hands them to the sensors they belong to.
type looper struct {
	svc    *Service
	lib    *library
	ptr    uintptr
	events uintptr
	ready  chan struct{}
	done   chan struct{}
	ended  atomic.Bool
}

func (l *looper) run() {
	runtime.LockOSThread()
	defer close(l.done)
	l.ptr = l.lib.looperPrepare(looperPrepareAllowNonCallbacks)
	if l.ptr != 0 {
		l.lib.looperAcquire(l.ptr)
	}
	close(l.ready)
	if l.ptr == 0 {
		return
	}
	for !l.ended.Load() {
		if l.lib.looperPoll(-1, 0, 0, 0) == looperIdent {
			l.drain()
		}
	}
	l.lib.looperRelease(l.ptr)
}

// stop ends the thread and waits for it, then frees its buffer.
func (l *looper) stop() {
	l.ended.Store(true)
	l.lib.looperWake(l.ptr)
	<-l.done
	dl.Free(l.events)
}

// drain reads every queue.
func (l *looper) drain() {
	l.svc.mu.Lock()
	list := make([]*device, 0, len(l.svc.devices))
	for d := range l.svc.devices {
		list = append(list, d)
	}
	l.svc.mu.Unlock()
	for _, d := range list {
		d.drain(l.events)
	}
}

// device is an open sensor: an event queue with one sensor enabled.
type device struct {
	svc    *Service
	lib    *library
	mgr    uintptr
	kind   platform.SensorKind
	h      platform.SensorHandler
	sensor uintptr
	queue  uintptr
	rate   float64

	// mu is held while the queue is read and the handler runs, so that Close finds both idle.
	mu     sync.Mutex
	closed bool
	batch  []platform.SensorReading
}

var _ platform.SensorDevice = (*device)(nil)

// Rate implements platform.SensorDevice.
func (d *device) Rate() float64 { return d.rate }

// drain takes the events that are in the queue and hands them over as one batch for each call of getEvents.
func (d *device) drain(buf uintptr) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for !d.closed {
		n := d.lib.getEvents(d.queue, buf, maxEvents)
		if n <= 0 {
			return
		}
		d.batch = d.batch[:0]
		for i := 0; i < int(n); i++ {
			d.batch = append(d.batch, readEvent(buf+uintptr(i)*eventSize))
		}
		d.h.SensorReadings(d.batch)
		if n < maxEvents {
			return
		}
	}
}

// readEvent turns the ASensorEvent at the address into a reading. Android's sensors already measure in m/s² and rad/s on the axes
// of the device in its natural orientation (x right, y up, z out of the screen), which are the toolkit's.
func readEvent(addr uintptr) platform.SensorReading {
	ts := *(*int64)(dl.Pointer(addr + eventTimestamp))
	v := unsafe.Slice((*float32)(dl.Pointer(addr+eventData)), 3)
	return platform.SensorReading{Time: time.Duration(ts), Values: [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}}
}

// Close implements platform.SensorDevice: when it returns the handler is not running and is never called again.
func (d *device) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()
	d.svc.mu.Lock()
	delete(d.svc.devices, d)
	d.svc.mu.Unlock()
	if r := d.lib.disableSensor(d.queue, d.sensor); r < 0 {
		d.svc.log.Debug("sensors: a sensor could not be disabled", "result", int(r))
	}
	d.lib.destroyEventQueue(d.mgr, d.queue)
	return nil
}
