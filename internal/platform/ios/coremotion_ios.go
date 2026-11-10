//go:build ios

package ios

import (
	"context"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// coreMotion is platform.Sensors over CMMotionManager (SPEC §30.9): the accelerometer and the gyroscope, each a stream that Core Motion
// delivers on an operation queue of its own and the hub of coremotion.go gives to every sensor that is open. Apple asks for one manager
// in the whole app, which this is.
//
// Flutter: no counterpart
type coreMotion struct {
	once    sync.Once
	manager objc.ID
	acc     motionStream
	gyro    motionStream
}

var _ platform.Sensors = (*coreMotion)(nil)

// motionStream is one kind of data of the manager: its hub, the queue that Core Motion calls the handler on and the block that is running.
type motionStream struct {
	kind  platform.SensorKind
	hub   *motionHub
	queue objc.ID
	// block is the handler of the stream that Core Motion has (copied; released when the stream stops).
	block objc.Block
	mu    sync.Mutex
}

// motionVector is a CMAcceleration or a CMRotationRate: three doubles.
type motionVector struct{ X, Y, Z float64 }

// load makes the manager and the hubs at the first use.
func (c *coreMotion) load() {
	c.once.Do(func() {
		if _, err := dl.OpenFramework("CoreMotion"); err != nil {
			return
		}
		c.manager = class("CMMotionManager").Send(sel("alloc")).Send(sel("init"))
		c.acc.kind, c.gyro.kind = platform.SensorAccelerometer, platform.SensorGyroscope
		for _, s := range []*motionStream{&c.acc, &c.gyro} {
			s.queue = class("NSOperationQueue").Send(sel("alloc")).Send(sel("init"))
			s.queue.Send(sel("setMaxConcurrentOperationCount:"), int64(1))
			s.hub = &motionHub{start: func(rate float64) { c.startStream(s, rate) }, stop: func() { c.stopStream(s) }}
		}
	})
}

// available says whether the device has the sensor: the Simulator has neither.
func (c *coreMotion) available(kind platform.SensorKind) bool {
	c.load()
	if c.manager == 0 {
		return false
	}
	if kind == platform.SensorGyroscope {
		return send[bool](c.manager, "isGyroAvailable")
	}
	return send[bool](c.manager, "isAccelerometerAvailable")
}

// Available says whether the device has a sensor of the kind.
//
// Flutter: no counterpart
func (c *coreMotion) Available(kind platform.SensorKind) bool {
	var ok bool
	objcx.WithPool(func() { ok = c.available(kind) })
	return ok
}

// Open opens the sensor and starts it at the rate (60 readings a second when the app says none, at most 100). It fails with
// SensorUnavailable when the device has no such sensor (the Simulator has none), which is how Core Motion says it. The first reading comes
// from Core Motion's queue, never from the UI thread.
//
// Flutter: no counterpart
func (c *coreMotion) Open(ctx context.Context, kind platform.SensorKind, opts platform.SensorOpenOptions, h platform.SensorHandler) (platform.SensorDevice, error) {
	if kind != platform.SensorAccelerometer && kind != platform.SensorGyroscope {
		return nil, errors.New(platform.TagSensorUnavailable, "iOS has no such sensor", "kind", int(kind))
	}
	if !c.Available(kind) {
		return nil, errors.New(platform.TagSensorUnavailable, "this device has no such motion sensor", "kind", int(kind))
	}
	if ctx.Err() != nil {
		return nil, errors.NewErr(platform.TagSensorOpenFailed, ctx.Err()).SetMetadata("kind", int(kind))
	}
	s := &c.acc
	if kind == platform.SensorGyroscope {
		s = &c.gyro
	}
	return s.hub.open(opts.Rate, h), nil
}

// startStream starts (or restarts, for a new rate) the data of the stream.
func (c *coreMotion) startStream(s *motionStream, rate float64) {
	objcx.WithPool(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		gyro := s.kind == platform.SensorGyroscope
		if s.block != 0 {
			stopSelector := "stopAccelerometerUpdates"
			if gyro {
				stopSelector = "stopGyroUpdates"
			}
			c.manager.Send(sel(stopSelector))
			s.block.Release()
			s.block = 0
		}
		intervalSelector, startSelector, dataKey := "setAccelerometerUpdateInterval:", "startAccelerometerUpdatesToQueue:withHandler:", "acceleration"
		if gyro {
			intervalSelector, startSelector, dataKey = "setGyroUpdateInterval:", "startGyroUpdatesToQueue:withHandler:", "rotationRate"
		}
		c.manager.Send(sel(intervalSelector), 1/rate)
		blk := objcx.NewBlock(func(_ objc.Block, data, _ objc.ID) {
			if data == 0 {
				return
			}
			v := send[motionVector](data, dataKey)
			values := [3]float64{v.X, v.Y, v.Z}
			if !gyro {
				values = accelerometerValues(values)
			}
			s.hub.deliver([]platform.SensorReading{{Time: bootTime(send[float64](data, "timestamp")), Values: values}})
		})
		s.block = blk.Copy()
		blk.Release()
		c.manager.Send(sel(startSelector), s.queue, s.block)
	})
}

// stopStream stops the data of the stream and frees its block.
func (c *coreMotion) stopStream(s *motionStream) {
	objcx.WithPool(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.block == 0 {
			return
		}
		if s.kind == platform.SensorGyroscope {
			c.manager.Send(sel("stopGyroUpdates"))
		} else {
			c.manager.Send(sel("stopAccelerometerUpdates"))
		}
		s.block.Release()
		s.block = 0
	})
}

// close stops both streams and releases what the manager made.
func (c *coreMotion) close() {
	if c.manager == 0 {
		return
	}
	for _, s := range []*motionStream{&c.acc, &c.gyro} {
		c.stopStream(s)
		objcx.Release(s.queue)
		s.queue = 0
	}
	objcx.Release(c.manager)
	c.manager = 0
}
