package ios

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// This file is the plain-Go part of the gamepads of iOS (SPEC §30.9): what the extended gamepad profile of the Game Controller framework
// reports becomes the standard layout, and the set of the pads that are connected reports to the handler of package platform. The calls to
// GCController are in gamecontroller_ios.go.
//
// The design follows SDL 3's src/joystick/apple/SDL_mfijoystick.m, which is where the choice of the extended profile, the buttons that a
// controller may lack and the vertical axes came from; the code here is new and is changed.

// gcState is the state of an extended gamepad: each button as down or up, the triggers as 0 to 1 and the sticks as -1 to 1 with up positive
// (the Game Controller framework's own directions).
type gcState struct {
	A, B, X, Y                                  bool
	LeftShoulder, RightShoulder                 bool
	LeftThumbstickButton, RightThumbstickButton bool
	DPadUp, DPadDown, DPadLeft, DPadRight       bool
	Menu, Options, Home                         bool
	Paddle1, Paddle2, Paddle3, Paddle4          bool
	Touchpad, Share                             bool
	LeftTrigger, RightTrigger                   float64
	LeftX, LeftY, RightX, RightY                float64
}

// gcFeatures says which of the optional buttons the controller has (a controller of the extended profile always has the rest).
type gcFeatures struct {
	Options, Home, ThumbstickButtons, Paddles, Touchpad, Share bool
}

// buttonMaskOf is the mask of the buttons of the standard layout that the controller has: A, B, X, Y, the shoulders, the d-pad, the
// triggers' side of Start (the menu button), and the buttons that the features say.
func buttonMaskOf(f gcFeatures) uint32 {
	bit := func(b gamepaddb.Button) uint32 { return 1 << b }
	mask := bit(gamepaddb.South) | bit(gamepaddb.East) | bit(gamepaddb.West) | bit(gamepaddb.North) |
		bit(gamepaddb.LeftShoulder) | bit(gamepaddb.RightShoulder) |
		bit(gamepaddb.DPadUp) | bit(gamepaddb.DPadDown) | bit(gamepaddb.DPadLeft) | bit(gamepaddb.DPadRight) |
		bit(gamepaddb.Start)
	if f.Options {
		mask |= bit(gamepaddb.Back)
	}
	if f.Home {
		mask |= bit(gamepaddb.Guide)
	}
	if f.ThumbstickButtons {
		mask |= bit(gamepaddb.LeftStick) | bit(gamepaddb.RightStick)
	}
	if f.Paddles {
		mask |= bit(gamepaddb.Paddle1) | bit(gamepaddb.Paddle2) | bit(gamepaddb.Paddle3) | bit(gamepaddb.Paddle4)
	}
	if f.Touchpad {
		mask |= bit(gamepaddb.Touchpad)
	}
	if f.Share {
		mask |= bit(gamepaddb.Misc1)
	}
	return mask
}

// snapshotOf turns the state of an extended gamepad into the snapshot of the standard layout: the menu button is Start, the options button
// Back and the home button Guide; a stick's up is the layout's negative y (the layout has down and right positive); the triggers go from 0
// to 32767.
func snapshotOf(st gcState) padstd.Snapshot {
	var s padstd.Snapshot
	set := func(b gamepaddb.Button, down bool) { s.Buttons[b] = down }
	set(gamepaddb.South, st.A)
	set(gamepaddb.East, st.B)
	set(gamepaddb.West, st.X)
	set(gamepaddb.North, st.Y)
	set(gamepaddb.LeftShoulder, st.LeftShoulder)
	set(gamepaddb.RightShoulder, st.RightShoulder)
	set(gamepaddb.LeftStick, st.LeftThumbstickButton)
	set(gamepaddb.RightStick, st.RightThumbstickButton)
	set(gamepaddb.DPadUp, st.DPadUp)
	set(gamepaddb.DPadDown, st.DPadDown)
	set(gamepaddb.DPadLeft, st.DPadLeft)
	set(gamepaddb.DPadRight, st.DPadRight)
	set(gamepaddb.Start, st.Menu)
	set(gamepaddb.Back, st.Options)
	set(gamepaddb.Guide, st.Home)
	set(gamepaddb.Paddle1, st.Paddle1)
	set(gamepaddb.Paddle2, st.Paddle2)
	set(gamepaddb.Paddle3, st.Paddle3)
	set(gamepaddb.Paddle4, st.Paddle4)
	set(gamepaddb.Touchpad, st.Touchpad)
	set(gamepaddb.Misc1, st.Share)
	s.Axes[gamepaddb.LeftX] = padstd.Signed(st.LeftX)
	s.Axes[gamepaddb.LeftY] = padstd.Signed(-st.LeftY)
	s.Axes[gamepaddb.RightX] = padstd.Signed(st.RightX)
	s.Axes[gamepaddb.RightY] = padstd.Signed(-st.RightY)
	s.Axes[gamepaddb.LeftTrigger] = padstd.Unit(st.LeftTrigger)
	s.Axes[gamepaddb.RightTrigger] = padstd.Unit(st.RightTrigger)
	return s
}

// padSpec is what the framework says of a controller that connected.
type padSpec struct {
	Name     string
	Features gcFeatures
	Rumble   bool
	LED      bool
	// Battery is the charge from 0 to 1, or a negative number when it is not known.
	Battery float64
}

// gcPad is a controller that is connected.
type gcPad struct {
	id       platform.PadID
	spec     padSpec
	tracker  *padstd.Tracker
	reported bool
}

// padSet is the set of the controllers that are connected and the handler that hears of them. A controller is known by a key that the
// platform gives it (the address of its object). It belongs to the UI thread, where the Game Controller framework posts its
// notifications and calls its handlers.
type padSet struct {
	h    platform.PadHandler
	next platform.PadID
	pads map[uintptr]*gcPad
	// order is the keys in the order the controllers connected.
	order []uintptr
}

func newPadSet() *padSet { return &padSet{pads: map[uintptr]*gcPad{}} }

// info is the description of a pad for the handler: the extended profile reports the standard layout, so the pad is Standard with the
// mask of the buttons it has, and its GUID is made from its name.
func (s *padSet) info(p *gcPad) platform.PadInfo {
	battery := -1
	if p.spec.Battery >= 0 {
		battery = int(p.spec.Battery*100 + 0.5)
	}
	return platform.PadInfo{
		ID: p.id, Name: p.spec.Name, GUID: gamepaddb.NewGUIDForName(p.spec.Name),
		Axes: int(gamepaddb.AxisCount), Buttons: int(gamepaddb.ButtonCount),
		Standard: true, ButtonMask: buttonMaskOf(p.spec.Features),
		Rumble: p.spec.Rumble, RumbleTriggers: false, LED: p.spec.LED, Battery: battery,
	}
}

// start gives the handler the pads that are connected already, and the ones that come after.
func (s *padSet) start(h platform.PadHandler) {
	s.h = h
	for _, key := range s.order {
		s.report(s.pads[key])
	}
}

// stop ends the reports: the pads that are connected stay known, and are reported again by the next start.
func (s *padSet) stop() {
	s.h = nil
	for _, p := range s.pads {
		p.reported = false
	}
}

func (s *padSet) report(p *gcPad) {
	if s.h == nil || p.reported {
		return
	}
	p.reported = true
	s.h.PadAdded(s.info(p))
}

// add is a controller that connected. A controller that is known already is left as it is. It returns the pad's ID, and ok false for a
// controller that was there.
func (s *padSet) add(key uintptr, spec padSpec) (platform.PadID, bool) {
	if p, ok := s.pads[key]; ok {
		return p.id, false
	}
	s.next++
	p := &gcPad{id: s.next, spec: spec, tracker: padstd.NewTracker(s.next)}
	s.pads[key] = p
	s.order = append(s.order, key)
	s.report(p)
	return p.id, true
}

// remove is a controller that went.
func (s *padSet) remove(key uintptr) {
	p, ok := s.pads[key]
	if !ok {
		return
	}
	delete(s.pads, key)
	for i, k := range s.order {
		if k == key {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	if s.h != nil && p.reported {
		s.h.PadRemoved(p.id)
	}
}

// update gives the handler the events of what changed in the state of the controller since the last time. Nothing is reported for a
// controller that is not known or for a state that did not change.
func (s *padSet) update(key uintptr, st gcState, now time.Duration) {
	p, ok := s.pads[key]
	if !ok || s.h == nil || !p.reported {
		return
	}
	snap := snapshotOf(st)
	if events := p.tracker.Update(now, &snap, nil); len(events) > 0 {
		s.h.PadEvents(events)
	}
}

// id is the ID of the pad of the controller, and false when it is not known.
func (s *padSet) id(key uintptr) (platform.PadID, bool) {
	p, ok := s.pads[key]
	if !ok {
		return 0, false
	}
	return p.id, true
}

// keyOf is the key of the controller that has the pad ID, and false when there is none.
func (s *padSet) keyOf(id platform.PadID) (uintptr, bool) {
	for key, p := range s.pads {
		if p.id == id {
			return key, true
		}
	}
	return 0, false
}

// rumbleParams are the intensity (0 to 1) and the sharpness (0 to 1) of a continuous haptic event for the strengths of the two motors:
// the intensity is the stronger of the two, and the sharpness the share of the high-frequency motor in the sum, as the handles of a
// controller feel (SDL does the same with its two amplitudes).
func rumbleParams(low, high uint16) (intensity, sharpness float32) {
	if low == 0 && high == 0 {
		return 0, 0
	}
	intensity = float32(max(low, high)) / 65535
	sharpness = float32(high) / float32(int(low)+int(high))
	return intensity, sharpness
}
