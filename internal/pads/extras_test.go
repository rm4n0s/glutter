package pads

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

func extraPad(r *rig) (platform.PadID, *Pad) {
	info := standard("DualShock")
	info.Touchpads, info.Sensors, info.SensorRate, info.Rumble, info.LED = 1, true, 250, true, true
	id := r.connect(info)
	return id, r.m.Pads()[0]
}

func TestTouchesReachTheListenersAndAreKept(t *testing.T) {
	r := newRig(t, false)
	id, pad := extraPad(r)
	var got []Touch
	remove := pad.AddTouchListener(func(tc Touch) { got = append(got, tc) })
	if pad.Touchpads() != 1 {
		t.Errorf("touchpads = %d", pad.Touchpads())
	}

	r.svc.SendTouch(platform.PadTouch{Pad: id, Time: time.Second, Finger: 0, Down: true, X: 0.25, Y: 0.5, Pressure: 1})
	r.svc.SendTouch(platform.PadTouch{Pad: id, Time: time.Second, Finger: 1, Down: true, X: 0.75, Y: 0.5, Pressure: 1})
	r.run()
	if len(got) != 2 || got[0].Pad != pad || got[0].X != 0.25 || got[1].Finger != 1 {
		t.Fatalf("touches = %+v", got)
	}
	if fingers := pad.Touches(); len(fingers) != 2 {
		t.Errorf("Touches() = %+v", fingers)
	}
	r.svc.SendTouch(platform.PadTouch{Pad: id, Time: 2 * time.Second, Finger: 0, X: 0.25, Y: 0.5})
	r.run()
	if len(got) != 3 || got[2].Down || len(pad.Touches()) != 1 {
		t.Errorf("after the finger left: %+v, fingers %d", got, len(pad.Touches()))
	}

	// A finger that is not one of the two, or a touchpad that is not there, is ignored.
	r.svc.SendTouch(platform.PadTouch{Pad: id, Finger: 2, Down: true})
	r.svc.SendTouch(platform.PadTouch{Pad: id, Touchpad: 5, Down: true})
	r.run()
	if len(got) != 3 {
		t.Errorf("touches that cannot be were reported: %+v", got[3:])
	}

	remove()
	r.svc.SendTouch(platform.PadTouch{Pad: id, Finger: 0, Down: true, X: 1})
	r.run()
	if len(got) != 3 {
		t.Error("a listener that was removed was called")
	}
}

func TestAPadThatIsRemovedLiftsItsFingers(t *testing.T) {
	r := newRig(t, false)
	id, pad := extraPad(r)
	r.svc.SendTouch(platform.PadTouch{Pad: id, Finger: 1, Down: true, X: 0.5, Y: 0.5, Pressure: 1})
	r.run()
	var got []Touch
	pad.AddTouchListener(func(tc Touch) { got = append(got, tc) })
	r.svc.Disconnect(id)
	r.run()
	if len(got) != 1 || got[0].Down || got[0].Finger != 1 || got[0].X != 0.5 {
		t.Errorf("touches = %+v", got)
	}
	// A touch for a pad that is gone is dropped.
	r.svc.SendTouch(platform.PadTouch{Pad: id, Down: true})
	r.run()
}

func TestBatteryIsKeptAndChanges(t *testing.T) {
	r := newRig(t, false)
	id, pad := extraPad(r)
	if pad.Battery() != -1 || pad.Power().State != platform.PadPowerUnknown {
		t.Errorf("power at first = %+v, battery %d", pad.Power(), pad.Battery())
	}
	r.svc.SendBattery(platform.PadBattery{Pad: id, State: platform.PadCharging, Percent: 55})
	r.run()
	if pad.Battery() != 55 || pad.Power() != (Power{platform.PadCharging, 55}) {
		t.Errorf("power = %+v, battery %d", pad.Power(), pad.Battery())
	}
}

func TestSensorsReportOnlyWhileTheyAreOn(t *testing.T) {
	r := newRig(t, false)
	id, pad := extraPad(r)
	if !pad.HasSensors() || pad.SensorRate() != 250 || pad.SensorsEnabled() {
		t.Fatalf("sensors: has %v, rate %v, enabled %v", pad.HasSensors(), pad.SensorRate(), pad.SensorsEnabled())
	}
	var got []SensorReading
	pad.AddSensorListener(func(s SensorReading) { got = append(got, s) })

	// A reading that arrives while the sensors are off is dropped.
	r.svc.SendSensor(platform.PadSensor{Pad: id, Kind: platform.PadGyroscope, Values: [3]float32{1, 2, 3}})
	r.run()
	if len(got) != 0 {
		t.Fatal("a reading arrived while the sensors were off")
	}
	if err := pad.EnableSensors(true); err != nil {
		t.Fatal(err)
	}
	if !r.svc.SensorsOn(id) || !pad.SensorsEnabled() {
		t.Error("the sensors were not turned on")
	}
	r.svc.SendSensor(platform.PadSensor{Pad: id, Time: time.Second, SensorTime: time.Millisecond, Kind: platform.PadGyroscope, Values: [3]float32{1, 2, 3}})
	r.svc.SendSensor(platform.PadSensor{Pad: id, Time: time.Second, Kind: platform.PadAccelerometer, Values: [3]float32{0, 9.8, 0}})
	r.run()
	if len(got) != 2 || got[0].Values != [3]float32{1, 2, 3} || got[0].SensorTime != time.Millisecond || got[1].Kind != platform.PadAccelerometer {
		t.Fatalf("readings = %+v", got)
	}
	if v, ok := pad.LatestSensor(platform.PadAccelerometer); !ok || v.Values[1] != 9.8 {
		t.Errorf("latest accel = %+v, %v", v, ok)
	}
	if err := pad.EnableSensors(false); err != nil {
		t.Fatal(err)
	}
	if _, ok := pad.LatestSensor(platform.PadGyroscope); ok {
		t.Error("a reading is kept after the sensors went off")
	}
	r.svc.SendSensor(platform.PadSensor{Pad: id, Kind: platform.PadGyroscope})
	r.run()
	if len(got) != 2 {
		t.Error("a reading arrived after the sensors went off")
	}
}

func TestSensorsOfAPadThatHasNone(t *testing.T) {
	r := newRig(t, false)
	r.connect(standard("Plain"))
	err := r.m.Pads()[0].EnableSensors(true)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("Pad.EnableSensors." + platform.TagGamepadSensorsUnavailable) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestAStandardPadHasTheButtonsItSays(t *testing.T) {
	r := newRig(t, false)
	info := standard("Pad")
	info.ButtonMask = 1<<gamepaddb.South | 1<<gamepaddb.Touchpad
	r.connect(info)
	pad := r.m.Pads()[0]
	if !pad.HasButton(gamepaddb.South) || !pad.HasButton(gamepaddb.Touchpad) || pad.HasButton(gamepaddb.Paddle1) {
		t.Error("the buttons of the pad are not those of its mask")
	}
	// Without a mask a standard pad has every button.
	r2 := newRig(t, false)
	r2.connect(standard("Other"))
	if !r2.m.Pads()[0].HasButton(gamepaddb.Paddle1) {
		t.Error("a standard pad without a mask lacks a button")
	}
}
