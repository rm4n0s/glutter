package ios

import (
	"sync"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

// This file is the plain-Go part of the motion sensors of iOS (SPEC §30.9): the conversion of what Core Motion reports to the units and axes
// of package platform, the rate, and the hub that gives one Core Motion stream to every sensor that the app opened of a kind. The calls to
// CMMotionManager are in coremotion_ios.go.
//
// The design follows SDL 3's src/sensor/coremotion/SDL_coremotionsensor.m, which is where the sign of the acceleration and the polling
// interval came from; the code here is new and is changed.

// standardGravity is the acceleration of gravity in m/s², the unit of the accelerometer in g.
const standardGravity = 9.80665

// accelerometerValues turns an acceleration of Core Motion, in g with the sign that makes a device that lies flat and face up read -1 on z,
// into m/s² with the sign of package platform (the same device reads about +9.8 on z).
//
// Flutter: no counterpart
func accelerometerValues(g [3]float64) [3]float64 {
	return [3]float64{-g[0] * standardGravity, -g[1] * standardGravity, -g[2] * standardGravity}
}

// motionRate is the rate of the readings a second that Core Motion is set to for the rate that an app asked for: 60 for none, and at most
// 100, which is what the accelerometer and the gyroscope of an iPhone do.
//
// Flutter: no counterpart
func motionRate(asked float64) float64 {
	switch {
	case asked <= 0:
		return 60
	case asked > 100:
		return 100
	case asked < 1:
		return 1
	}
	return asked
}

// motionDevice is a sensor that an app opened: its handler and the rate it asked for.
type motionDevice struct {
	hub  *motionHub
	h    platform.SensorHandler
	rate float64

	mu     sync.Mutex
	closed bool
}

var _ platform.SensorDevice = (*motionDevice)(nil)

// Rate is the rate that Core Motion is set to while the sensor is open.
func (d *motionDevice) Rate() float64 { return d.hub.currentRate() }

// Close ends the sensor: when it returns the handler is not running and is never called again. Closing twice does nothing.
func (d *motionDevice) Close() error {
	d.mu.Lock()
	d.closed = true
	d.mu.Unlock()
	d.hub.remove(d)
	return nil
}

// deliver gives the readings to the handler unless the sensor is closed; the lock makes Close wait for a handler that runs.
func (d *motionDevice) deliver(r []platform.SensorReading) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.closed {
		d.h.SensorReadings(r)
	}
}

// motionHub gives the readings of one Core Motion stream (the accelerometer, or the gyroscope) to every device that is open, and starts and
// stops the stream when the first device opens and the last one closes. The rate is the fastest that a device asked for.
type motionHub struct {
	// start starts the stream at the rate and stop stops it; the readings come back to deliver.
	start func(rate float64)
	stop  func()

	mu   sync.Mutex
	devs []*motionDevice
	rate float64
}

// open adds a sensor and starts the stream at the rate that now serves all of them.
func (h *motionHub) open(asked float64, handler platform.SensorHandler) *motionDevice {
	d := &motionDevice{hub: h, h: handler, rate: motionRate(asked)}
	h.mu.Lock()
	h.devs = append(h.devs, d)
	rate := h.maxRateLocked()
	changed := rate != h.rate
	h.rate = rate
	h.mu.Unlock()
	if changed {
		h.start(rate)
	}
	return d
}

// remove takes a sensor out: the stream stops with the last one, and slows down when the fastest one is gone.
func (h *motionHub) remove(d *motionDevice) {
	h.mu.Lock()
	idx := -1
	for i, x := range h.devs {
		if x == d {
			idx = i
		}
	}
	if idx < 0 {
		h.mu.Unlock()
		return
	}
	h.devs = append(h.devs[:idx], h.devs[idx+1:]...)
	rate := h.maxRateLocked()
	changed := rate != h.rate
	h.rate = rate
	h.mu.Unlock()
	switch {
	case !changed:
	case rate == 0:
		h.stop()
	default:
		h.start(rate)
	}
}

func (h *motionHub) maxRateLocked() float64 {
	var rate float64
	for _, d := range h.devs {
		rate = max(rate, d.rate)
	}
	return rate
}

// currentRate is the rate that the stream is set to.
func (h *motionHub) currentRate() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rate
}

// deliver gives the readings to every open device.
func (h *motionHub) deliver(r []platform.SensorReading) {
	h.mu.Lock()
	devs := append([]*motionDevice(nil), h.devs...)
	h.mu.Unlock()
	for _, d := range devs {
		d.deliver(r)
	}
}

// bootTime turns the timestamp of Core Motion (seconds since the device started) into the monotonic time of a reading.
//
// Flutter: no counterpart
func bootTime(seconds float64) time.Duration { return time.Duration(seconds * float64(time.Second)) }
