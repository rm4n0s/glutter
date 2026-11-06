package pads

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// This file holds what some pads have besides buttons and axes: a touchpad, motion sensors and a battery that tells its state
// (SPEC §28.2). The platform backends whose pads have them report them through platform.PadExtrasHandler, which the manager
// implements.

// Touch is a finger on a touchpad of a pad.
//
// Flutter: no counterpart
type Touch struct {
	Pad *Pad
	// Time is when the pad reported the finger, on the system's monotonic clock.
	Time time.Duration
	// Touchpad and Finger number the touchpad and the finger, from 0. A PlayStation pad has one touchpad and two fingers.
	Touchpad, Finger int
	// Down says that the finger is on the touchpad. X and Y are from 0 to 1, from the top left corner, and where the finger
	// left when it is not down; Pressure is 1 for a finger that is down and 0 for one that is not.
	Down           bool
	X, Y, Pressure float32
}

// SensorReading is a reading of a motion sensor of a pad (SPEC §28.4, which makes the type public in step 31.7).
//
// Flutter: no counterpart
type SensorReading struct {
	Pad *Pad
	// Time is when the report that held the reading arrived, and SensorTime when the sensor made it, on its own clock.
	Time, SensorTime time.Duration
	Kind             platform.PadSensorKind
	// Values are the three components: rad/s for the gyroscope, m/s² (gravity included) for the accelerometer.
	Values [3]float32
}

// Power is what a pad says about its battery.
//
// Flutter: no counterpart
type Power struct {
	State platform.PadPowerState
	// Percent is from 0 to 100, or -1 when it is not known.
	Percent int
}

type touchListener struct {
	id int
	fn func(Touch)
}

type sensorListener struct {
	id int
	fn func(SensorReading)
}

// padExtras is the state of the extras of a pad.
type padExtras struct {
	power           Power
	fingers         [4]Touch
	touchListeners  []touchListener
	sensorListeners []sensorListener
	nextID          int
	sensorsOn       bool
	latest          [2]SensorReading
	hasLatest       [2]bool
}

func (e *padExtras) init(info platform.PadInfo) {
	e.power = Power{Percent: info.Battery}
	if info.Battery < 0 {
		e.power.Percent = -1
	}
}

// Power returns the state of the pad's battery.
//
// Flutter: no counterpart
func (p *Pad) Power() Power { return p.extras.power }

func (p *Pad) setPower(b platform.PadBattery) {
	p.extras.power = Power{State: b.State, Percent: b.Percent}
}

// Touchpads is the number of touchpads of the pad.
//
// Flutter: no counterpart
func (p *Pad) Touchpads() int { return p.info.Touchpads }

// Touches returns the fingers that are on the touchpads of the pad now.
//
// Flutter: no counterpart
func (p *Pad) Touches() []Touch {
	var out []Touch
	for _, f := range p.extras.fingers {
		if f.Down {
			out = append(out, f)
		}
	}
	return out
}

// AddTouchListener adds a function that is called, on the UI thread, when a finger goes down, moves or goes up, and returns the
// function that removes it. A listener may add and remove listeners.
//
// Flutter: no counterpart
func (p *Pad) AddTouchListener(fn func(Touch)) (remove func()) {
	p.extras.nextID++
	id := p.extras.nextID
	p.extras.touchListeners = append(p.extras.touchListeners, touchListener{id, fn})
	return func() {
		ls := p.extras.touchListeners
		for i, l := range ls {
			if l.id == id {
				p.extras.touchListeners = append(ls[:i:i], ls[i+1:]...)
				return
			}
		}
	}
}

func (p *Pad) touch(t platform.PadTouch) {
	i := t.Touchpad*2 + t.Finger
	if t.Touchpad < 0 || t.Finger < 0 || t.Finger > 1 || i >= len(p.extras.fingers) {
		return
	}
	ev := Touch{Pad: p, Time: t.Time, Touchpad: t.Touchpad, Finger: t.Finger, Down: t.Down, X: t.X, Y: t.Y, Pressure: t.Pressure}
	p.extras.fingers[i] = ev
	for _, l := range append([]touchListener(nil), p.extras.touchListeners...) {
		l.fn(ev)
	}
}

// releaseTouches lifts the fingers that are down, before the pad is removed.
func (p *Pad) releaseTouches(t time.Duration) {
	for i := range p.extras.fingers {
		f := p.extras.fingers[i]
		if !f.Down {
			continue
		}
		p.touch(platform.PadTouch{Pad: p.info.ID, Time: t, Touchpad: f.Touchpad, Finger: f.Finger, X: f.X, Y: f.Y})
	}
}

// HasSensors says whether the pad has a gyroscope and an accelerometer.
//
// Flutter: no counterpart
func (p *Pad) HasSensors() bool { return p.info.Sensors }

// SensorRate is how many readings of each sensor the pad makes in a second, or 0 when it does not say.
//
// Flutter: no counterpart
func (p *Pad) SensorRate() float32 { return p.info.SensorRate }

// SensorsEnabled says whether the sensors are on.
//
// Flutter: no counterpart
func (p *Pad) SensorsEnabled() bool { return p.extras.sensorsOn }

// EnableSensors turns the sensors of the pad on or off: a pad reports its sensors only while they are on, which saves its
// battery. It fails with GamepadSensorsUnavailable when the pad has none or the platform cannot turn them on.
//
// Flutter: no counterpart
func (p *Pad) EnableSensors(on bool) error {
	ctl, ok := p.m.plat.(platform.PadSensorControl)
	if !p.info.Sensors || !ok {
		return errors.New(platform.TagGamepadSensorsUnavailable, "the pad has no motion sensors that can be turned on", "pad", int64(p.info.ID))
	}
	if err := ctl.SetPadSensors(p.info.ID, on); err != nil {
		return err
	}
	p.extras.sensorsOn = on
	if !on {
		p.extras.hasLatest = [2]bool{}
	}
	return nil
}

// LatestSensor returns the last reading of the sensor of the kind, if there is one.
//
// Flutter: no counterpart
func (p *Pad) LatestSensor(kind platform.PadSensorKind) (SensorReading, bool) {
	if int(kind) >= len(p.extras.latest) || !p.extras.hasLatest[kind] {
		return SensorReading{}, false
	}
	return p.extras.latest[kind], true
}

// AddSensorListener adds a function that is called, on the UI thread, with every reading of the sensors, and returns the
// function that removes it. A listener may add and remove listeners.
//
// Flutter: no counterpart
func (p *Pad) AddSensorListener(fn func(SensorReading)) (remove func()) {
	p.extras.nextID++
	id := p.extras.nextID
	p.extras.sensorListeners = append(p.extras.sensorListeners, sensorListener{id, fn})
	return func() {
		ls := p.extras.sensorListeners
		for i, l := range ls {
			if l.id == id {
				p.extras.sensorListeners = append(ls[:i:i], ls[i+1:]...)
				return
			}
		}
	}
}

func (p *Pad) sensor(s platform.PadSensor) {
	if !p.extras.sensorsOn || int(s.Kind) >= len(p.extras.latest) {
		return
	}
	r := SensorReading{Pad: p, Time: s.Time, SensorTime: s.SensorTime, Kind: s.Kind, Values: s.Values}
	p.extras.latest[s.Kind], p.extras.hasLatest[s.Kind] = r, true
	for _, l := range append([]sensorListener(nil), p.extras.sensorListeners...) {
		l.fn(r)
	}
}
