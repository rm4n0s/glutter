//go:build js && wasm

package web

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"syscall/js"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The motion sensors of the web (SPEC §28.4, step 31.7), designed after the Emscripten sensor driver of SDL 3 (src/sensor/
// emscripten/SDL_emscriptensensor.c; zlib license, third_party/sdl/LICENSE), translated to Go and changed: SDL reads the browser's
// events with Emscripten's own calls and a fixed list of one accelerometer and one gyroscope, and here one listener for the
// devicemotion events of the window feeds every open sensor (the readings of MotionReading), the page has no sensors before it is
// served from a secure context, and where the browser asks the user (iOS Safari: DeviceMotionEvent.requestPermission, which only a
// gesture may call) the first Open waits for the first gesture and the answer.

// webSensors is the sensors service of the page.
type webSensors struct {
	log *slog.Logger

	mu        sync.Mutex
	devices   map[*sensorDevice]struct{}
	listening bool
	closed    bool

	// made says that the callbacks exist; motionFn gets the events and gestureFn the gestures that may ask for the permission.
	made       bool
	motionFn   js.Func
	gestureFn  js.Func
	gestureOn  bool
	permission permissionState
	waiters    []chan struct{}
}

type permissionState uint8

const (
	// permissionUnknown: nothing was asked; where the browser has no question this is the same as granted.
	permissionUnknown permissionState = iota
	permissionGranted
	permissionDenied
)

func newWebSensors(log *slog.Logger) *webSensors {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &webSensors{log: log, devices: map[*sensorDevice]struct{}{}}
}

var _ platform.Sensors = (*webSensors)(nil)

// sensorService makes the sensors service on the first call.
func (p *Platform) sensorService() platform.Sensors {
	if p.sensors == nil {
		p.sensors = newWebSensors(p.log)
	}
	return p.sensors
}

// supported says whether the page can have motion events: the browser has them and the page is a secure context.
func supported() (ok bool, reason string) {
	g := global()
	if !isObject(g.Get("DeviceMotionEvent")) {
		return false, "the browser has no DeviceMotionEvent"
	}
	if secure := g.Get("isSecureContext"); secure.Type() == js.TypeBoolean && !secure.Bool() {
		return false, "motion sensors need a secure context: the page must be served over https or from localhost"
	}
	return true, ""
}

// Available says whether the browser has motion events. A browser cannot tell before the first event whether the device has the
// sensor, so it answers yes wherever the events exist.
//
// Flutter: no counterpart
func (s *webSensors) Available(platform.SensorKind) bool {
	ok, _ := supported()
	return ok
}

// needsPermission says that the browser asks the user for the sensors (and must be asked at a gesture).
func needsPermission() bool {
	return global().Get("DeviceMotionEvent").Get("requestPermission").Type() == js.TypeFunction
}

// makeCallbacks makes the functions that the browser calls, once. mu is held.
func (s *webSensors) makeCallbacks() {
	if s.made {
		return
	}
	s.made = true
	s.motionFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			s.onMotion(args[0])
		}
		return nil
	})
	s.gestureFn = js.FuncOf(func(js.Value, []js.Value) any {
		s.askPermission()
		return nil
	})
}

// numberOf reads a number of a JavaScript object, and says whether it was one (null is not).
func numberOf(v js.Value) (float64, bool) {
	if v.Type() != js.TypeNumber {
		return 0, false
	}
	return v.Float(), true
}

// readMotion turns a DeviceMotionEvent into a MotionEvent.
func readMotion(ev js.Value) MotionEvent {
	var e MotionEvent
	e.TimeStamp = number(ev.Get("timeStamp"), 0)
	e.Interval = number(ev.Get("interval"), 0)
	if a := ev.Get("accelerationIncludingGravity"); isObject(a) {
		x, okx := numberOf(a.Get("x"))
		y, oky := numberOf(a.Get("y"))
		z, okz := numberOf(a.Get("z"))
		if okx && oky && okz {
			e.HasAcceleration, e.Acceleration = true, [3]float64{x, y, z}
		}
	}
	if r := ev.Get("rotationRate"); isObject(r) {
		a, oka := numberOf(r.Get("alpha"))
		b, okb := numberOf(r.Get("beta"))
		g, okg := numberOf(r.Get("gamma"))
		if oka && okb && okg {
			e.HasRotation, e.Alpha, e.Beta, e.Gamma = true, a, b, g
		}
	}
	return e
}

// onMotion gives the event to every open sensor, as the callback of the page.
func (s *webSensors) onMotion(ev js.Value) {
	e := readMotion(ev)
	s.mu.Lock()
	list := make([]*sensorDevice, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	s.mu.Unlock()
	rate := MotionRate(e.Interval)
	for _, d := range list {
		if r, ok := MotionReading(d.kind, e); ok {
			d.deliver(r, rate)
		}
	}
}

// askPermission asks the browser for the sensors, now, and settles the open calls that wait. A browser that grants them without a
// gesture (Chrome) answers at once. One that wants a gesture (Safari on iOS) rejects a call that is not made in one: then the page
// waits for the first gesture and asks again, from its listener.
func (s *webSensors) askPermission() {
	s.mu.Lock()
	if s.permission != permissionUnknown || len(s.waiters) == 0 || s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	then := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 && args[0].Type() == js.TypeString && args[0].String() == "granted" {
			s.settle(permissionGranted)
		} else {
			s.settle(permissionDenied)
		}
		return nil
	})
	fail := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			s.log.Debug("sensors: the permission was not asked for; waiting for a gesture", "reason", describe(args[0]))
		}
		s.mu.Lock()
		if s.permission == permissionUnknown && len(s.waiters) > 0 && !s.closed {
			s.setGestureLocked(true)
		}
		s.mu.Unlock()
		return nil
	})
	if !try(func() {
		p := global().Get("DeviceMotionEvent").Call("requestPermission")
		p.Call("then", then, fail)
	}) {
		fail.Release()
		then.Release()
	}
	// The functions of a promise stay for as long as the page needs them; they are few (one for each question).
}

// settle records the user's answer and wakes the open calls that wait.
func (s *webSensors) settle(state permissionState) {
	s.mu.Lock()
	s.permission = state
	waiters := s.waiters
	s.waiters = nil
	s.setGestureLocked(false)
	s.mu.Unlock()
	for _, w := range waiters {
		close(w)
	}
}

// setGestureLocked starts or stops waiting for the gesture that may ask for the permission. mu is held.
func (s *webSensors) setGestureLocked(on bool) {
	if on == s.gestureOn {
		return
	}
	s.gestureOn = on
	doc := global().Get("document")
	for _, ev := range gestures {
		if on {
			doc.Call("addEventListener", ev, s.gestureFn, true)
		} else {
			doc.Call("removeEventListener", ev, s.gestureFn, true)
		}
	}
}

// Open opens a sensor and starts it. It fails with SensorUnavailable when the browser has no motion events or the page is not a
// secure context, SensorPermissionDenied when the user refuses (where the browser asks), and SensorOpenFailed when ctx ends first. On
// a browser that asks, the call waits for the answer (and, where the browser wants one, for the first gesture): an app opens the sensor from a goroutine of its own.
//
// Flutter: no counterpart
func (s *webSensors) Open(ctx context.Context, kind platform.SensorKind, opts platform.SensorOpenOptions, h platform.SensorHandler) (platform.SensorDevice, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(platform.TagSensorOpenFailed, err).SetMetadata("kind", int(kind))
	}
	if ok, reason := supported(); !ok {
		return nil, errors.New(platform.TagSensorUnavailable, reason, "kind", int(kind))
	}
	if needsPermission() {
		state, wait := s.permissionWait()
		if state == permissionDenied {
			return nil, errors.New(platform.TagSensorPermissionDenied, "the user did not allow the motion sensors", "kind", int(kind))
		}
		if state == permissionUnknown {
			select {
			case <-wait:
			case <-ctx.Done():
				return nil, errors.NewErr(platform.TagSensorOpenFailed, ctx.Err()).SetMetadata("kind", int(kind), "waiting for", "the user's answer")
			}
			s.mu.Lock()
			state = s.permission
			s.mu.Unlock()
			if state != permissionGranted {
				return nil, errors.New(platform.TagSensorPermissionDenied, "the user did not allow the motion sensors", "kind", int(kind))
			}
		}
	}
	d := &sensorDevice{s: s, kind: kind, h: h}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New(platform.TagSensorUnavailable, "the sensors of the page were closed", "kind", int(kind))
	}
	s.makeCallbacks()
	s.devices[d] = struct{}{}
	if !s.listening {
		s.listening = true
		global().Call("addEventListener", "devicemotion", s.motionFn)
	}
	return d, nil
}

// permissionWait is the state of the permission, and, while it is not known, the channel that closes when it is. The first call
// starts waiting for the gesture.
func (s *webSensors) permissionWait() (permissionState, <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.permission != permissionUnknown {
		return s.permission, nil
	}
	s.makeCallbacks()
	w := make(chan struct{})
	s.waiters = append(s.waiters, w)
	first := len(s.waiters) == 1
	if first {
		// Asked outside the lock, on the goroutine of the page's event loop: it answers by promise.
		go s.askPermission()
	}
	return permissionUnknown, w
}

// forget drops a sensor, and stops listening with the last one.
func (s *webSensors) forget(d *sensorDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, d)
	if len(s.devices) == 0 && s.listening {
		s.listening = false
		global().Call("removeEventListener", "devicemotion", s.motionFn)
	}
}

// Close closes the sensors and releases the callbacks.
func (s *webSensors) Close() {
	s.mu.Lock()
	s.closed = true
	list := make([]*sensorDevice, 0, len(s.devices))
	for d := range s.devices {
		list = append(list, d)
	}
	waiters := s.waiters
	s.waiters = nil
	s.mu.Unlock()
	for _, w := range waiters {
		close(w)
	}
	for _, d := range list {
		if err := d.Close(); err != nil {
			s.log.Debug("sensors: a sensor could not be closed", "err", err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setGestureLocked(false)
	if s.made {
		s.motionFn.Release()
		s.gestureFn.Release()
		s.made = false
	}
}

// sensorDevice is an open sensor of the page.
type sensorDevice struct {
	s    *webSensors
	kind platform.SensorKind
	h    platform.SensorHandler

	// mu is held while the handler runs, so that Close finds it idle.
	mu     sync.Mutex
	closed bool
	rate   float64
	one    [1]platform.SensorReading
}

var _ platform.SensorDevice = (*sensorDevice)(nil)

// Rate implements platform.SensorDevice: the rate that the browser's events show, 0 before the first.
func (d *sensorDevice) Rate() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rate
}

func (d *sensorDevice) deliver(r platform.SensorReading, rate float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	if rate > 0 {
		d.rate = rate
	}
	d.one[0] = r
	d.h.SensorReadings(d.one[:])
}

// Close implements platform.SensorDevice.
func (d *sensorDevice) Close() error {
	d.mu.Lock()
	already := d.closed
	d.closed = true
	d.mu.Unlock()
	if !already {
		d.s.forget(d)
	}
	return nil
}

// PerformHapticFeedback vibrates with navigator.vibrate for the length of the kind (HapticMillis), on the devices whose browser has it
// (phones'). A browser may refuse it before the user has touched the page; there is nothing to report then.
//
// Flutter: shell/platform/android/io/flutter/plugin/platform/PlatformPlugin.java PlatformPlugin.vibrateHapticFeedback
func (w *window) PerformHapticFeedback(kind ui.HapticFeedbackKind) {
	ms := HapticMillis(kind)
	nav := global().Get("navigator")
	if ms == 0 || nav.Get("vibrate").Type() != js.TypeFunction {
		return
	}
	try(func() { nav.Call("vibrate", ms) })
}
