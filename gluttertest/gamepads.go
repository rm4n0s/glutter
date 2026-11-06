package gluttertest

import (
	"time"

	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// PadSpec describes a pad that a test connects with Tester.ConnectPad.
//
// Without Axes, Buttons, Hats and Mapping the pad is a standard pad: it reports the buttons and axes of the standard layout
// as they are (TestPad.Press, TestPad.SetAxis). With any of them it is a raw pad, like the ones the platforms report: it has
// that many buttons, axes and hats, the pad manager finds its mapping from GUID (or takes Mapping), and the test drives the
// raw inputs (TestPad.RawButton and the like).
//
// Flutter: no counterpart
type PadSpec struct {
	// Name is the name of the pad; "Test Pad" when empty.
	Name string
	// GUID is the 32 hexadecimal digits that name the model of the pad, as a mapping line does; when empty it is made from Name.
	GUID string
	// Axes, Buttons and Hats make the pad a raw pad with that many raw inputs.
	Axes, Buttons, Hats int
	// Mapping is the mapping text ("a:b0,b:b1,...") the platform reports for a raw pad, which is used when the table of mappings
	// has none for GUID.
	Mapping string
	// Rumble, RumbleTriggers and LED say what the pad can do.
	Rumble, RumbleTriggers, LED bool
	// Touchpads is the number of touchpads of the pad; TestPad.Touch moves fingers on them.
	Touchpads int
}

// PadRumble is a request that the app made to a test pad to rumble or to set its light.
//
// Flutter: no counterpart
type PadRumble struct {
	// Triggers says that the request was for the motors of the triggers.
	Triggers bool
	// LED says that the request was for the light, which Color holds as red, green and blue.
	LED   bool
	Color [3]uint8
	// Low and High (Left and Right for the triggers) are the strengths, from 0 to 65535.
	Low, High uint16
	Duration  time.Duration
}

// TestPad is a pad that a test connected. Its methods report to the app as the platform would: on the UI thread, in order, and
// the app's listeners and key handlers have run when they return.
//
// Flutter: no counterpart
type TestPad struct {
	t    *Tester
	id   platform.PadID
	raw  bool
	spec PadSpec
}

// ConnectPad connects a pad, the way a platform reports a pad that is plugged in, and runs what that posts.
//
// Flutter: no counterpart
func (t *Tester) ConnectPad(spec ...PadSpec) *TestPad {
	t.adopt()
	t.tb.Helper()
	var s PadSpec
	if len(spec) > 0 {
		s = spec[0]
	}
	if s.Name == "" {
		s.Name = "Test Pad"
	}
	guid := gamepaddb.ParseGUID(s.GUID)
	if s.GUID == "" {
		guid = gamepaddb.NewGUIDForName(s.Name)
	}
	raw := s.Axes > 0 || s.Buttons > 0 || s.Hats > 0 || s.Mapping != ""
	info := platform.PadInfo{
		Name: s.Name, GUID: guid, Axes: s.Axes, Buttons: s.Buttons, Hats: s.Hats, Mapping: s.Mapping, Standard: !raw,
		Rumble: s.Rumble, RumbleTriggers: s.RumbleTriggers, LED: s.LED, Battery: -1, Touchpads: s.Touchpads,
	}
	p := &TestPad{t: t, raw: raw, spec: s}
	p.id = t.padsvc.Connect(info)
	t.host.RunPosted()
	return p
}

// padKey gives the framework the key event of a pad.
func (t *Tester) padKey(k pads.Key) {
	typ := ui.KeyEventTypeUp
	if k.Down {
		typ = ui.KeyEventTypeDown
	}
	dev := ui.KeyEventDeviceTypeGamepad
	if k.DirectionalPad {
		dev = ui.KeyEventDeviceTypeDirectionalPad
	}
	t.d.DispatchKeyData(ui.KeyData{TimeStamp: t.Elapsed(), Type: typ, DeviceType: dev, Physical: k.Physical, Logical: k.Logical})
}

// ID is the number the platform gave the pad.
//
// Flutter: no counterpart
func (p *TestPad) ID() gamepad.PadID { return gamepad.PadID(p.id) }

// Pad returns the pad as the app sees it, or nil when the pad has no mapping (it is not a gamepad) or is disconnected.
//
// Flutter: no counterpart
func (p *TestPad) Pad() *gamepad.Pad {
	for _, q := range gamepad.ForBinding(p.t.Binding).Pads() {
		if q.ID() == p.ID() {
			return q
		}
	}
	return nil
}

func (p *TestPad) send(events ...platform.PadEvent) {
	p.t.adopt()
	for i := range events {
		events[i].Pad = p.id
		events[i].Time = p.t.Elapsed()
	}
	p.t.padsvc.Send(events...)
	p.t.host.RunPosted()
}

// Press presses the button of the standard layout. On a raw pad it presses the raw button that the pad's mapping gives the
// button, and fails the test when there is none.
//
// Flutter: no counterpart
func (p *TestPad) Press(b gamepad.Button) { p.t.tb.Helper(); p.setButton(b, true) }

// Release lets go of the button of the standard layout.
//
// Flutter: no counterpart
func (p *TestPad) Release(b gamepad.Button) { p.t.tb.Helper(); p.setButton(b, false) }

// Tap presses the button and lets go of it.
//
// Flutter: no counterpart
func (p *TestPad) Tap(b gamepad.Button) {
	p.t.tb.Helper()
	p.setButton(b, true)
	p.setButton(b, false)
}

func (p *TestPad) setButton(b gamepad.Button, down bool) {
	p.t.tb.Helper()
	if p.raw {
		p.t.tb.Fatalf("TestPad.Press and Release are for standard pads; this one is a raw pad: use RawButton")
		return
	}
	v := 0
	if down {
		v = 1
	}
	p.send(platform.PadEvent{Kind: platform.PadButton, Index: int(b), Value: v})
}

// SetAxis moves an axis of the standard layout: from −1 to 1, from 0 to 1 for a trigger. It is for standard pads.
//
// Flutter: no counterpart
func (p *TestPad) SetAxis(a gamepad.Axis, v float64) {
	p.t.tb.Helper()
	if p.raw {
		p.t.tb.Fatalf("TestPad.SetAxis is for standard pads; this one is a raw pad: use RawAxis")
		return
	}
	n := int(v * gamepaddb.AxisMax)
	if v <= -1 {
		n = gamepaddb.AxisMin
	}
	p.send(platform.PadEvent{Kind: platform.PadAxis, Index: int(a), Value: n})
}

// RawButton presses or releases the raw button i of a raw pad.
//
// Flutter: no counterpart
func (p *TestPad) RawButton(i int, down bool) {
	v := 0
	if down {
		v = 1
	}
	p.send(platform.PadEvent{Kind: platform.PadButton, Index: i, Value: v})
}

// RawAxis moves the raw axis i of a raw pad to v, from −32768 to 32767.
//
// Flutter: no counterpart
func (p *TestPad) RawAxis(i, v int) {
	p.send(platform.PadEvent{Kind: platform.PadAxis, Index: i, Value: v})
}

// RawHat sets the raw hat i of a raw pad to a mask of 1 (up), 2 (right), 4 (down) and 8 (left).
//
// Flutter: no counterpart
func (p *TestPad) RawHat(i int, mask uint8) {
	p.send(platform.PadEvent{Kind: platform.PadHat, Index: i, Value: int(mask)})
}

// Touch puts a finger on a touchpad of the pad, moves it or lifts it: x and y are from 0 to 1, from the top left corner.
//
// Flutter: no counterpart
func (p *TestPad) Touch(touchpad, finger int, down bool, x, y float64) {
	p.t.adopt()
	var pressure float32
	if down {
		pressure = 1
	}
	p.t.padsvc.SendTouch(platform.PadTouch{Pad: p.id, Time: p.t.Elapsed(), Touchpad: touchpad, Finger: finger, Down: down, X: float32(x), Y: float32(y), Pressure: pressure})
	p.t.host.RunPosted()
}

// SetBattery makes the pad report its battery: the charge from 0 to 1, and whether it is being charged.
//
// Flutter: no counterpart
func (p *TestPad) SetBattery(level float64, charging bool) {
	p.t.adopt()
	state := platform.PadOnBattery
	if charging {
		state = platform.PadCharging
	}
	p.t.padsvc.SendBattery(platform.PadBattery{Pad: p.id, State: state, Percent: int(level*100 + 0.5)})
	p.t.host.RunPosted()
}

// Disconnect unplugs the pad.
//
// Flutter: no counterpart
func (p *TestPad) Disconnect() {
	p.t.adopt()
	p.t.padsvc.Disconnect(p.id)
	p.t.host.RunPosted()
}

// Rumbles returns what the app asked the pad to do with its motors and its light, in order.
//
// Flutter: no counterpart
func (p *TestPad) Rumbles() []PadRumble {
	var out []PadRumble
	for _, r := range p.t.padsvc.Rumbles() {
		if r.Pad == p.id {
			out = append(out, PadRumble{Triggers: r.Triggers, LED: r.LED, Color: r.Color, Low: r.Low, High: r.High, Duration: r.Duration})
		}
	}
	return out
}
