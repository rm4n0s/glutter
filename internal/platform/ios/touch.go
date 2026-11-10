package ios

import (
	"math"
	"sort"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the touch input (SPEC §30.4): the conversion of the UITouch objects of a UIEvent, as the view
// reads them, to the raw pointer data that the framework's converter (package pointerconv) completes with identifiers, deltas, and the
// add before a down. It ports what FlutterViewController's dispatchTouches does. It has no OS tag, so tables of touches test it on every host.

// touchPhase is the phase of a touch (UITouchPhase).
type touchPhase int

// The phases of a touch, with UIKit's numbers.
const (
	touchBegan touchPhase = iota
	touchMoved
	touchStationary
	touchEnded
	touchCancelled
)

// touchKind is what made a touch (UITouchType).
type touchKind int

// The kinds of touch, with UIKit's numbers: a finger, the Apple Pencil, and the indirect pointer of a trackpad or mouse that arrives
// as a touch (a click).
const (
	touchDirect touchKind = iota
	touchIndirect
	touchPencil
	touchIndirectPointer
)

// touchSample is what the conversion reads of one UITouch.
type touchSample struct {
	// ID identifies the touch while it lasts: the address of the UITouch, which UIKit keeps for the life of the touch.
	ID    uintptr
	Phase touchPhase
	Kind  touchKind
	// X and Y are the place in the view, in points.
	X, Y float64
	// Force and MaxForce are UITouch.force and maximumPossibleForce; a device with no force (MaxForce 0) gives a contact the pressure 1.
	Force, MaxForce float64
	// Altitude is the angle of the Pencil to the surface (0 is flat, pi/2 is upright) and Azimuth its direction in the view, both in
	// radians; they are zero for a finger.
	Altitude, Azimuth float64
	// Radius and Tolerance are the major radius of the contact in points and its tolerance.
	Radius, Tolerance float64
	// Time is the timestamp of the touch as a duration since the start of the platform.
	Time time.Duration
}

// touchTracker gives each touch that lasts a device number of its own (the converter keys its states by device), the smallest that no
// touch of the moment has, so that the numbers stay small and a number is reused after a touch has ended.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm _touchMapping
type touchTracker struct {
	devices map[uintptr]int
}

func newTouchTracker() *touchTracker { return &touchTracker{devices: map[uintptr]int{}} }

// device is the number of a touch, given when it begins.
func (t *touchTracker) device(id uintptr) int {
	if d, ok := t.devices[id]; ok {
		return d
	}
	used := map[int]bool{}
	for _, d := range t.devices {
		used[d] = true
	}
	d := 1
	for used[d] {
		d++
	}
	t.devices[id] = d
	return d
}

// Active is how many touches have begun and not ended.
func (t *touchTracker) Active() int { return len(t.devices) }

// convert turns the touches of one UIEvent (and the coalesced touches of each, oldest first: callers give them in order) into raw
// pointer data for a view at the scale: a began touch is a down, a moved one a move, an ended one an up followed by the remove of its
// device, a cancelled one a cancel and a remove, and a stationary one nothing. A move or an up of a touch that did not begin is
// dropped, as is any data of a touch that has gone.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm dispatchTouches:pointerDataChangeOverride:event:
func (t *touchTracker) convert(view ui.ViewID, samples []touchSample, scale float64) []ui.PointerData {
	if scale <= 0 {
		scale = 1
	}
	var out []ui.PointerData
	for _, s := range samples {
		var change ui.PointerChange
		switch s.Phase {
		case touchBegan:
			change = ui.PointerChangeDown
		case touchMoved:
			change = ui.PointerChangeMove
		case touchEnded:
			change = ui.PointerChangeUp
		case touchCancelled:
			change = ui.PointerChangeCancel
		default:
			continue
		}
		if _, known := t.devices[s.ID]; !known && s.Phase != touchBegan {
			continue
		}
		device := t.device(s.ID)
		d := ui.PointerData{
			ViewID: view, TimeStamp: s.Time, Change: change, Kind: touchKindOf(s.Kind), Device: device,
			PhysicalX: s.X * scale, PhysicalY: s.Y * scale, Scale: 1,
		}
		if s.Kind == touchIndirectPointer {
			d.Buttons = 1 // a click of the primary button
		}
		if s.Kind == touchPencil {
			d.Tilt = math.Pi/2 - s.Altitude
			d.Orientation = s.Azimuth
			d.Buttons = 0
		}
		d.Pressure, d.PressureMin, d.PressureMax = pressureOf(s)
		d.RadiusMajor, d.RadiusMinor = s.Radius*scale, s.Radius*scale
		d.RadiusMin, d.RadiusMax = math.Max(0, s.Radius-s.Tolerance)*scale, (s.Radius+s.Tolerance)*scale
		out = append(out, d)
		if change == ui.PointerChangeUp || change == ui.PointerChangeCancel {
			rm := d
			rm.Change = ui.PointerChangeRemove
			rm.Pressure = 0
			out = append(out, rm)
			delete(t.devices, s.ID)
		}
	}
	return out
}

// touchKindOf is the device kind of a touch.
func touchKindOf(k touchKind) ui.PointerDeviceKind {
	switch k {
	case touchPencil:
		return ui.PointerDeviceKindStylus
	case touchIndirectPointer:
		return ui.PointerDeviceKindMouse
	}
	return ui.PointerDeviceKindTouch
}

// pressureOf is the pressure of a touch and its range: the force of a device that has it, and 1 in a range of 0 to 1 for a device that
// has not (an iPhone without 3D Touch or Haptic Touch reports no force), as the Android embedding does for a touch without pressure.
func pressureOf(s touchSample) (pressure, min, max float64) {
	if s.MaxForce <= 0 {
		return 1, 0, 1
	}
	return s.Force, 0, s.MaxForce
}

// sortSamples orders the touches of an event by time then by identity, so that the same event gives the same data on every run (UIKit's
// set of touches has no order).
func sortSamples(s []touchSample) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Time != s[j].Time {
			return s[i].Time < s[j].Time
		}
		return s[i].ID < s[j].ID
	})
}
