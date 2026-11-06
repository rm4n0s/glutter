package pads

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// State is the buttons and axes of a pad now.
//
// Flutter: no counterpart
type State struct {
	buttons uint32
	// Axes are the raw values of the axes, from gamepaddb.AxisMin to gamepaddb.AxisMax (the triggers from 0).
	Axes [gamepaddb.AxisCount]int16
}

// Down says whether the button is down.
//
// Flutter: no counterpart
func (s State) Down(b gamepaddb.Button) bool {
	return b < gamepaddb.ButtonCount && s.buttons&(1<<b) != 0
}

// EventKind says what an Event reports.
//
// Flutter: no counterpart
type EventKind uint8

const (
	// ButtonDown is a button that went down.
	//
	// Flutter: no counterpart
	ButtonDown EventKind = iota
	// ButtonUp is a button that went up.
	//
	// Flutter: no counterpart
	ButtonUp
	// AxisMotion is an axis that has a new value.
	//
	// Flutter: no counterpart
	AxisMotion
)

// Event is a change of the buttons or axes of a pad.
//
// Flutter: no counterpart
type Event struct {
	Pad  *Pad
	Time time.Duration
	Kind EventKind
	// Button is the button of a ButtonDown or ButtonUp.
	Button gamepaddb.Button
	// Axis and Value are the axis of an AxisMotion and its new raw value.
	Axis  gamepaddb.Axis
	Value int16
}

// Pad is a connected gamepad that has a mapping to the standard layout.
//
// Flutter: no counterpart
type Pad struct {
	m    *Manager
	info platform.PadInfo

	name        string
	kind        gamepaddb.Kind
	face        gamepaddb.FaceStyle
	mapping     *gamepaddb.Mapping
	mappingText string
	dev         *gamepaddb.Device
	hasButton   uint32
	hasAxis     uint8
	connected   bool

	state     State
	listeners []padListener
	nextID    int
	triggers  [2]bool
	// Wrapper is a place for the public type that wraps the pad, so that the same pad is the same value for the app.
	Wrapper any
}

type padListener struct {
	id int
	fn func(Event)
}

// setMapping gives the pad the bindings of the mapping. It reports whether the mapping can be used.
func (p *Pad) setMapping(mp *gamepaddb.Mapping) bool {
	dev, err := gamepaddb.NewDevice(p.info.Axes, p.info.Buttons, p.info.Hats, mp, p)
	if err != nil {
		p.m.log.Warn("the mapping of a pad cannot be used", "name", p.info.Name, "err", err)
		return false
	}
	if p.dev != nil {
		// The buttons and axes that are held go back to rest before the new bindings read the pad.
		p.dev.Recenter(0)
	}
	p.dev = dev
	p.mapping, p.mappingText = mp, mp.Text
	p.name = mp.Name
	if p.name == "" || p.name == "*" {
		p.name = p.info.Name
	}
	guid := gamepaddb.GUID(p.info.GUID)
	p.kind = mp.Kind()
	if p.kind == gamepaddb.KindUnknown {
		p.kind = gamepaddb.KindOfGUID(guid, p.info.Name)
	}
	p.face = mp.FaceStyle()
	if p.face == gamepaddb.FaceUnknown {
		p.face = gamepaddb.FaceStyleOf(p.kind)
	}
	p.hasButton, p.hasAxis = 0, 0
	for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
		if dev.HasButton(b) {
			p.hasButton |= 1 << b
		}
	}
	for a := gamepaddb.Axis(0); a < gamepaddb.AxisCount; a++ {
		if dev.HasAxis(a) {
			p.hasAxis |= 1 << a
		}
	}
	return true
}

// ID is the platform's number of the pad.
//
// Flutter: no counterpart
func (p *Pad) ID() platform.PadID { return p.info.ID }

// Name is the name of the pad.
//
// Flutter: no counterpart
func (p *Pad) Name() string { return p.name }

// Path says where the pad is on the system, or "".
//
// Flutter: no counterpart
func (p *Pad) Path() string { return p.info.Path }

// GUID identifies the model of the pad.
//
// Flutter: no counterpart
func (p *Pad) GUID() gamepaddb.GUID { return gamepaddb.GUID(p.info.GUID) }

// Kind is the model of the pad.
//
// Flutter: no counterpart
func (p *Pad) Kind() gamepaddb.Kind { return p.kind }

// FaceStyle is how the face buttons of the pad are labelled.
//
// Flutter: no counterpart
func (p *Pad) FaceStyle() gamepaddb.FaceStyle { return p.face }

// Mapping is the mapping text of the pad ("a:b0,b:b1,..."), or "" for a pad that reports the standard layout.
//
// Flutter: no counterpart
func (p *Pad) Mapping() string {
	if p.mapping == nil {
		return ""
	}
	return p.mapping.Text
}

// Connected says whether the pad is still there.
//
// Flutter: no counterpart
func (p *Pad) Connected() bool { return p.connected }

// HasButton says whether the pad has the button.
//
// Flutter: no counterpart
func (p *Pad) HasButton(b gamepaddb.Button) bool {
	return b < gamepaddb.ButtonCount && p.hasButton&(1<<b) != 0
}

// HasAxis says whether the pad has the axis.
//
// Flutter: no counterpart
func (p *Pad) HasAxis(a gamepaddb.Axis) bool { return a < gamepaddb.AxisCount && p.hasAxis&(1<<a) != 0 }

// CanRumble says whether the pad has motors, as its platform reported.
//
// Flutter: no counterpart
func (p *Pad) CanRumble() bool { return p.info.Rumble }

// CanRumbleTriggers says whether the pad has motors in its triggers.
//
// Flutter: no counterpart
func (p *Pad) CanRumbleTriggers() bool { return p.info.RumbleTriggers }

// HasLED says whether the pad has a light that can be set.
//
// Flutter: no counterpart
func (p *Pad) HasLED() bool { return p.info.LED }

// Battery is the charge of the pad from 0 to 100, or -1 when it is not known.
//
// Flutter: no counterpart
func (p *Pad) Battery() int { return p.info.Battery }

// State returns the buttons and axes of the pad now.
//
// Flutter: no counterpart
func (p *Pad) State() State { return p.state }

// AddListener adds a function that is called, on the UI thread, with every change of a button or axis of the pad, and returns
// the function that removes it. A listener may add and remove listeners.
//
// Flutter: no counterpart
func (p *Pad) AddListener(fn func(Event)) (remove func()) {
	p.nextID++
	id := p.nextID
	p.listeners = append(p.listeners, padListener{id, fn})
	return func() {
		for i, l := range p.listeners {
			if l.id == id {
				p.listeners = append(p.listeners[:i:i], p.listeners[i+1:]...)
				return
			}
		}
	}
}

// Rumble turns the pad's motors for d; see platform.Gamepads.Rumble.
//
// Flutter: no counterpart
func (p *Pad) Rumble(low, high uint16, d time.Duration) error {
	return p.m.plat.Rumble(p.info.ID, low, high, d)
}

// RumbleTriggers turns the motors of the pad's triggers for d.
//
// Flutter: no counterpart
func (p *Pad) RumbleTriggers(left, right uint16, d time.Duration) error {
	return p.m.plat.RumbleTriggers(p.info.ID, left, right, d)
}

// SetLED sets the colour of the pad's light.
//
// Flutter: no counterpart
func (p *Pad) SetLED(r, g, b uint8) error { return p.m.plat.SetLED(p.info.ID, r, g, b) }

// report takes in a raw report of the platform.
func (p *Pad) report(e platform.PadEvent) {
	t := int64(e.Time)
	if p.info.Standard {
		switch e.Kind {
		case platform.PadButton:
			p.ButtonChanged(t, gamepaddb.Button(e.Index), e.Value != 0)
		case platform.PadAxis:
			p.AxisChanged(t, gamepaddb.Axis(e.Index), int16(max(gamepaddb.AxisMin, min(gamepaddb.AxisMax, e.Value))))
		}
		return
	}
	switch e.Kind {
	case platform.PadAxis:
		p.dev.RawAxis(t, e.Index, e.Value)
	case platform.PadButton:
		p.dev.RawButton(t, e.Index, e.Value != 0)
	case platform.PadHat:
		p.dev.RawHat(t, e.Index, uint8(e.Value))
	}
}

// remove ends the pad: whatever is held is let go (and the keys that it holds are released) before the listeners stop.
func (p *Pad) remove(t time.Duration) {
	if !p.connected {
		return
	}
	if p.dev != nil {
		p.dev.Recenter(int64(t))
	} else {
		for b := gamepaddb.Button(0); b < gamepaddb.ButtonCount; b++ {
			p.ButtonChanged(int64(t), b, false)
		}
		for a := gamepaddb.Axis(0); a < gamepaddb.AxisCount; a++ {
			p.AxisChanged(int64(t), a, 0)
		}
	}
	p.connected = false
	p.listeners = nil
}

// ButtonChanged implements gamepaddb.Sink.
func (p *Pad) ButtonChanged(t int64, b gamepaddb.Button, down bool) {
	if b >= gamepaddb.ButtonCount || p.state.Down(b) == down {
		return
	}
	if down {
		p.state.buttons |= 1 << b
	} else {
		p.state.buttons &^= 1 << b
	}
	kind := ButtonUp
	if down {
		kind = ButtonDown
	}
	p.emit(Event{Pad: p, Time: time.Duration(t), Kind: kind, Button: b})
	if k, ok := buttonKeys[b]; ok {
		p.key(time.Duration(t), k, down)
	}
}

// AxisChanged implements gamepaddb.Sink.
func (p *Pad) AxisChanged(t int64, a gamepaddb.Axis, v int16) {
	if a >= gamepaddb.AxisCount || p.state.Axes[a] == v {
		return
	}
	p.state.Axes[a] = v
	p.emit(Event{Pad: p, Time: time.Duration(t), Kind: AxisMotion, Axis: a, Value: v})
	if a == gamepaddb.LeftTrigger || a == gamepaddb.RightTrigger {
		i := int(a - gamepaddb.LeftTrigger)
		down := v >= triggerThreshold
		if down != p.triggers[i] {
			p.triggers[i] = down
			p.key(time.Duration(t), triggerKeys[i], down)
		}
	}
}

// triggerThreshold is the value from which a trigger counts as a pressed key: half its travel.
const triggerThreshold = 16384

func (p *Pad) emit(e Event) {
	for _, l := range append([]padListener(nil), p.listeners...) {
		l.fn(e)
	}
}

func (p *Pad) key(t time.Duration, k keyIDs, down bool) {
	if !p.m.keys {
		return
	}
	p.m.opts.Keys(Key{Time: t, Down: down, Physical: k.physical, Logical: k.logical, DirectionalPad: k.dpad})
}

// keyIDs are the IDs of a key that a button of a pad makes.
type keyIDs struct {
	physical, logical uint64
	dpad              bool
}

// The IDs are those of services.PhysicalKeyboardKey and services.LogicalKeyboardKey (package gamepad's tests check them).
var buttonKeys = map[gamepaddb.Button]keyIDs{
	gamepaddb.South:         {0x0005ff11, 0x200000311, false}, // GameButtonA
	gamepaddb.East:          {0x0005ff12, 0x200000312, false}, // GameButtonB
	gamepaddb.West:          {0x0005ff1d, 0x20000031d, false}, // GameButtonX
	gamepaddb.North:         {0x0005ff1e, 0x20000031e, false}, // GameButtonY
	gamepaddb.LeftShoulder:  {0x0005ff14, 0x200000314, false}, // GameButtonLeft1
	gamepaddb.RightShoulder: {0x0005ff17, 0x200000317, false}, // GameButtonRight1
	gamepaddb.Start:         {0x0005ff1a, 0x20000031a, false}, // GameButtonStart
	gamepaddb.Back:          {0x0005ff19, 0x200000319, false}, // GameButtonSelect
	gamepaddb.Guide:         {0x0005ff16, 0x200000316, false}, // GameButtonMode
	gamepaddb.LeftStick:     {0x0005ff1b, 0x20000031b, false}, // GameButtonThumbLeft
	gamepaddb.RightStick:    {0x0005ff1c, 0x20000031c, false}, // GameButtonThumbRight
	gamepaddb.DPadUp:        {0x00070052, 0x100000304, true},  // ArrowUp
	gamepaddb.DPadDown:      {0x00070051, 0x100000301, true},  // ArrowDown
	gamepaddb.DPadLeft:      {0x00070050, 0x100000302, true},  // ArrowLeft
	gamepaddb.DPadRight:     {0x0007004f, 0x100000303, true},  // ArrowRight
}

// triggerKeys are the keys of the left and right trigger.
var triggerKeys = [2]keyIDs{
	{0x0005ff15, 0x200000315, false}, // GameButtonLeft2
	{0x0005ff18, 0x200000318, false}, // GameButtonRight2
}

// KeyIDs returns the physical and logical key IDs that the button makes, and whether it makes a key at all.
//
// Flutter: no counterpart
func KeyIDs(b gamepaddb.Button) (physical, logical uint64, ok bool) {
	k, ok := buttonKeys[b]
	return k.physical, k.logical, ok
}

// TriggerKeyIDs returns the physical and logical key IDs of the left (0) or right (1) trigger.
//
// Flutter: no counterpart
func TriggerKeyIDs(i int) (physical, logical uint64) {
	return triggerKeys[i].physical, triggerKeys[i].logical
}
