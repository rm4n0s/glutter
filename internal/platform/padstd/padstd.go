// Package padstd helps the backends whose pads already report the standard layout (XInput on Windows, the standard mapping of
// a browser): such a pad is a PadInfo with Standard set, and its raw buttons and axes are the Button and Axis of gamepaddb in
// the order of the layout. A backend that polls fills a Snapshot each time and a Tracker turns the changes into PadEvents
// (SPEC §28.2). It is plain Go with no OS tag.
//
// Flutter: no counterpart
package padstd

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Snapshot is the state of a pad of the standard layout: which buttons are down, and the axes (the sticks from −32768 to 32767
// with down and right positive, the triggers from 0 to 32767).
//
// Flutter: no counterpart
type Snapshot struct {
	Buttons [gamepaddb.ButtonCount]bool
	Axes    [gamepaddb.AxisCount]int16
}

// Tracker turns the snapshots of one pad into the events of what changed.
//
// Flutter: no counterpart
type Tracker struct {
	id   platform.PadID
	last Snapshot
}

// NewTracker makes the tracker of a pad that has just been connected: the state it starts from is everything up and every axis
// at rest.
//
// Flutter: no counterpart
func NewTracker(id platform.PadID) *Tracker { return &Tracker{id: id} }

// Update appends to out the events of what differs between s and the snapshot before it, buttons first and then axes, and
// remembers s. now is the time the state was read.
//
// Flutter: no counterpart
func (t *Tracker) Update(now time.Duration, s *Snapshot, out []platform.PadEvent) []platform.PadEvent {
	for b := range s.Buttons {
		if s.Buttons[b] != t.last.Buttons[b] {
			v := 0
			if s.Buttons[b] {
				v = 1
			}
			out = append(out, platform.PadEvent{Pad: t.id, Time: now, Kind: platform.PadButton, Index: b, Value: v})
		}
	}
	for a := range s.Axes {
		if s.Axes[a] != t.last.Axes[a] {
			out = append(out, platform.PadEvent{Pad: t.id, Time: now, Kind: platform.PadAxis, Index: a, Value: int(s.Axes[a])})
		}
	}
	t.last = *s
	return out
}

// Trigger8 scales a trigger of 8 bits (0 to 255) to the range of a trigger axis (0 to 32767).
//
// Flutter: no counterpart
func Trigger8(v uint8) int16 { return int16(int32(v) * 32767 / 255) }

// Unit scales a number from 0 to 1, as the browser gives the value of a trigger, to the range of a trigger axis, clamping it.
//
// Flutter: no counterpart
func Unit(v float64) int16 {
	switch {
	case v <= 0 || v != v:
		return 0
	case v >= 1:
		return gamepaddb.AxisMax
	}
	return int16(v * gamepaddb.AxisMax)
}

// Signed scales a number from −1 to 1, as the browser gives a stick, to the range of an axis, clamping it.
//
// Flutter: no counterpart
func Signed(v float64) int16 {
	switch {
	case v != v:
		return 0
	case v <= -1:
		return gamepaddb.AxisMin
	case v >= 1:
		return gamepaddb.AxisMax
	case v < 0:
		return int16(v * -gamepaddb.AxisMin)
	}
	return int16(v * gamepaddb.AxisMax)
}

// Flip turns an axis around: the vertical axes of XInput are positive upwards and those of the layout downwards.
//
// Flutter: no counterpart
func Flip(v int16) int16 {
	if v == gamepaddb.AxisMin {
		return gamepaddb.AxisMax
	}
	return -v
}
