package gamepaddb

// This file is translated to Go and changed from src/joystick/SDL_joystick.c (the layer that takes in the raw reports of a pad)
// and src/joystick/SDL_gamepad.c (the binding of the inputs of a pad to the buttons and axes of the layout) of SDL 3 (zlib
// license, third_party/sdl/LICENSE).

// Sink receives the buttons and axes of the standard layout that a Device makes out of the raw reports of a pad. It is told of
// changes only: a value that is the same as the last one is not sent. t is the time of the report, as the backend gave it.
//
// Flutter: no counterpart
type Sink interface {
	// ButtonChanged tells that the button went down or up.
	ButtonChanged(t int64, b Button, down bool)
	// AxisChanged tells that the axis has a new value, from AxisMin to AxisMax (triggers from 0).
	AxisChanged(t int64, a Axis, v int16)
}

// axisInfo is what the raw layer keeps of an axis of the pad, to find the value it rests at.
type axisInfo struct {
	value, initial, zero                               int
	hasInitial, hasSecond, sentInitial, sendingInitial bool
}

// Device turns the raw buttons, axes and hats of one pad into the buttons and axes of the layout, by the pad's mapping. It is
// the two layers of SDL in one: the layer that finds the rest value of an axis and drops the first jitters, and the layer that
// binds the inputs to the outputs. It is not safe for use from several goroutines.
//
// Flutter: no counterpart
type Device struct {
	sink     Sink
	bindings []Binding
	axes     []axisInfo
	buttons  []bool
	hats     []uint8
	// lastMatch is, for each raw axis, the binding that last matched it (-1 for none).
	lastMatch []int

	outButtons [ButtonCount]bool
	outAxes    [AxisCount]int16
}

// NewDevice makes the device of a pad that has the given numbers of raw axes, buttons and hats, and the mapping that says what
// they are (nil makes a device with no bindings). The first report of an axis that a trigger is bound to sets its rest value.
//
// Flutter: no counterpart
func NewDevice(axes, buttons, hats int, m *Mapping, sink Sink) (*Device, error) {
	d := &Device{
		sink:      sink,
		axes:      make([]axisInfo, axes),
		buttons:   make([]bool, buttons),
		hats:      make([]uint8, hats),
		lastMatch: make([]int, axes),
	}
	for i := range d.lastMatch {
		d.lastMatch[i] = -1
	}
	if m != nil {
		b, err := ParseBindings(m.Text)
		if err != nil {
			return nil, err
		}
		d.bindings = b
	}
	// Set the zero point for triggers: an axis that gives a trigger rests at the end of its range.
	for _, b := range d.bindings {
		if b.In == bindAxis && b.Out == bindAxis && (b.OutAxis == LeftTrigger || b.OutAxis == RightTrigger) && b.InIndex < axes {
			a := &d.axes[b.InIndex]
			a.value, a.zero = b.InMin, b.InMin
		}
	}
	return d, nil
}

// Bindings returns the bindings of the device.
//
// Flutter: no counterpart
func (d *Device) Bindings() []Binding { return d.bindings }

// HasButton says whether the mapping binds the button.
//
// Flutter: no counterpart
func (d *Device) HasButton(b Button) bool {
	for _, x := range d.bindings {
		if x.Out == bindButton && x.OutButton == b {
			return true
		}
	}
	return false
}

// HasAxis says whether the mapping binds the axis.
//
// Flutter: no counterpart
func (d *Device) HasAxis(a Axis) bool {
	for _, x := range d.bindings {
		if x.Out == bindAxis && x.OutAxis == a {
			return true
		}
	}
	return false
}

// Button returns whether the button of the layout is down.
//
// Flutter: no counterpart
func (d *Device) Button(b Button) bool { return b < ButtonCount && d.outButtons[b] }

// Axis returns the value of the axis of the layout.
//
// Flutter: no counterpart
func (d *Device) Axis(a Axis) int16 {
	if a >= AxisCount {
		return 0
	}
	return d.outAxes[a]
}

// RawAxis takes in a report of the axis i of the pad. A value that equals the last one is dropped, and so are the reports of an
// axis that has not moved yet beyond jitter from the value it was first read at (some pads report a value off the centre
// before anything is touched); the first real movement is then sent after the value the axis rests at.
//
// Flutter: no counterpart
func (d *Device) RawAxis(t int64, i int, value int) {
	if i < 0 || i >= len(d.axes) {
		return
	}
	info := &d.axes[i]
	if !info.hasInitial ||
		(!info.hasSecond && (info.initial <= -32767 || info.initial == 32767) && abs(value) < AxisMax/4) {
		info.initial, info.value, info.zero = value, value, value
		info.hasInitial = true
	} else if value == info.value && !info.sendingInitial {
		return
	} else {
		info.hasSecond = true
	}
	if !info.sentInitial {
		// Make sure we don't send motion until there's real activity on this axis.
		const maxAllowedJitter = AxisMax / 80
		if abs(value-info.value) <= maxAllowedJitter {
			return
		}
		info.sentInitial = true
		info.sendingInitial = true
		d.RawAxis(t, i, info.initial)
		info.sendingInitial = false
	}
	info.value = value
	d.bindAxis(t, i, value)
}

// RawButton takes in a report of the button i of the pad.
//
// Flutter: no counterpart
func (d *Device) RawButton(t int64, i int, down bool) {
	if i < 0 || i >= len(d.buttons) || d.buttons[i] == down {
		return
	}
	d.buttons[i] = down
	for _, b := range d.bindings {
		if b.In == bindButton && b.InIndex == i {
			if b.Out == bindAxis {
				v := b.OutMin
				if down {
					v = b.OutMax
				}
				d.sendAxis(t, b.OutAxis, v)
			} else {
				d.sendButton(t, b.OutButton, down)
			}
			break
		}
	}
}

// RawHat takes in a report of the hat i of the pad: a mask of HatUp, HatRight, HatDown and HatLeft.
//
// Flutter: no counterpart
func (d *Device) RawHat(t int64, i int, value uint8) {
	if i < 0 || i >= len(d.hats) || d.hats[i] == value {
		return
	}
	changed := d.hats[i] ^ value
	d.hats[i] = value
	for _, b := range d.bindings {
		if b.In != bindHat || b.InIndex != i || changed&b.HatMask == 0 {
			continue
		}
		switch {
		case value&b.HatMask == 0:
			d.reset(t, b)
		case b.Out == bindAxis:
			d.sendAxis(t, b.OutAxis, b.OutMax)
		default:
			d.sendButton(t, b.OutButton, true)
		}
	}
}

// bindAxis finds the binding for the value of a raw axis and sends its output.
func (d *Device) bindAxis(t int64, axis, value int) {
	last := d.lastMatch[axis]
	match := -1
	for n, b := range d.bindings {
		if b.In != bindAxis || b.InIndex != axis {
			continue
		}
		if b.InMin < b.InMax {
			if value >= b.InMin && value <= b.InMax {
				match = n
				break
			}
		} else if value >= b.InMax && value <= b.InMin {
			match = n
			break
		}
	}
	if last >= 0 && (match < 0 || !sameOutput(d.bindings[last], d.bindings[match])) {
		// Clear the last output that this axis generated.
		d.reset(t, d.bindings[last])
	}
	if match >= 0 {
		b := d.bindings[match]
		if b.Out == bindAxis {
			if b.InMin != b.OutMin || b.InMax != b.OutMax {
				normalized := float32(value-b.InMin) / float32(b.InMax-b.InMin)
				value = b.OutMin + int(normalized*float32(b.OutMax-b.OutMin))
			}
			d.sendAxis(t, b.OutAxis, value)
		} else {
			threshold := b.InMin + (b.InMax-b.InMin)/2
			var down bool
			if b.InMax < b.InMin {
				down = value <= threshold
			} else {
				down = value >= threshold
			}
			d.sendButton(t, b.OutButton, down)
		}
	}
	d.lastMatch[axis] = match
}

func sameOutput(a, b Binding) bool {
	if a.Out != b.Out {
		return false
	}
	if a.Out == bindAxis {
		return a.OutAxis == b.OutAxis
	}
	return a.OutButton == b.OutButton
}

func (d *Device) reset(t int64, b Binding) {
	if b.Out == bindAxis {
		d.sendAxis(t, b.OutAxis, 0)
	} else {
		d.sendButton(t, b.OutButton, false)
	}
}

func (d *Device) sendButton(t int64, b Button, down bool) {
	if b >= ButtonCount || d.outButtons[b] == down {
		return
	}
	d.outButtons[b] = down
	d.sink.ButtonChanged(t, b, down)
}

func (d *Device) sendAxis(t int64, a Axis, v int) {
	if a >= AxisCount {
		return
	}
	v16 := int16(max(AxisMin, min(AxisMax, v)))
	if d.outAxes[a] == v16 {
		return
	}
	d.outAxes[a] = v16
	d.sink.AxisChanged(t, a, v16)
}

// Recenter lets go of every button and centres every axis that is not at rest, as a pad that is removed does: the raw layer
// does it by its own guess of the centre, and this one knows the idle position of the layout.
//
// Flutter: no counterpart
func (d *Device) Recenter(t int64) {
	for b := Button(0); b < ButtonCount; b++ {
		d.sendButton(t, b, false)
	}
	for a := Axis(0); a < AxisCount; a++ {
		d.sendAxis(t, a, 0)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
