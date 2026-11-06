package evdev

import (
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
)

// This file is translated to Go and changed from src/joystick/linux/SDL_sysjoystick.c of SDL 3 (zlib license,
// third_party/sdl/LICENSE): HandleHat, AxisCorrect, PollAllValues and HandleInputEvents.

// inputEvent is an event the kernel sent, with its time on the monotonic clock.
type inputEvent struct {
	Time  time.Duration
	Type  uint16
	Code  uint16
	Value int32
}

// source is where the decoder asks for the state of the device when it has to find it out again: at the start, and after the
// kernel dropped events.
type source interface {
	// absValue returns the value of the axis, and whether it could be read.
	absValue(code int) (int32, bool)
	// keyState returns the bits of the keys that are down, and whether they could be read.
	keyState() (bits, bool)
}

// hatPositions is the position of a hat from the states of its two axes (0 low, 1 centre, 2 high), as the mask of the
// directions: indexed by the y axis and then the x axis.
var hatPositions = [3][3]uint8{
	{gamepaddb.HatLeft | gamepaddb.HatUp, gamepaddb.HatUp, gamepaddb.HatRight | gamepaddb.HatUp},
	{gamepaddb.HatLeft, gamepaddb.HatCentered, gamepaddb.HatRight},
	{gamepaddb.HatLeft | gamepaddb.HatDown, gamepaddb.HatDown, gamepaddb.HatRight | gamepaddb.HatDown},
}

// decoder turns the events of one device into the reports of the platform interface. It is used by one goroutine.
type decoder struct {
	id platform.PadID
	l  *layout
	// hatAxis is the state of the two axes of each hat: 0 low, 1 centre, 2 high.
	hatAxis [4][2]int
	// correct is the adaptive range of each digital hat, which widens as the device reports values beyond it.
	correct    [4]hatCorrect
	recovering bool
	out        []platform.PadEvent
}

func newDecoder(id platform.PadID, l *layout) *decoder {
	d := &decoder{id: id, l: l, correct: l.hatCorrect}
	for i := range d.hatAxis {
		d.hatAxis[i] = [2]int{1, 1}
	}
	return d
}

func (d *decoder) emit(t time.Duration, kind platform.PadEventKind, index, value int) {
	d.out = append(d.out, platform.PadEvent{Pad: d.id, Time: t, Kind: kind, Index: index, Value: value})
}

// handleHat takes in the value of one axis of a hat. Axes that are really analog and were taken for digital are kept from
// twitching by dead zones; real digital ones only ever send the minimum, 0 and the maximum.
func (d *decoder) handleHat(t time.Duration, hat, axis, value int) {
	c := &d.correct[hat]
	switch {
	case value < 0:
		switch {
		case value <= c.min[axis]:
			c.min[axis] = value
			value = 0
		case !c.useDeadzones || value < c.min[axis]/3:
			value = 0
		default:
			value = 1
		}
	case value > 0:
		switch {
		case value >= c.max[axis]:
			c.max[axis] = value
			value = 2
		case !c.useDeadzones || value > c.max[axis]/3:
			value = 2
		default:
			value = 1
		}
	default:
		value = 1
	}
	if value != d.hatAxis[hat][axis] {
		d.hatAxis[hat][axis] = value
		d.emit(t, platform.PadHat, d.l.hatIndex[hat], int(hatPositions[d.hatAxis[hat][1]][d.hatAxis[hat][0]]))
	}
}

// pollAll reports the state of every input again, from the device itself.
func (d *decoder) pollAll(t time.Duration, src source) {
	for code := absX; code < absMax; code++ {
		if d.l.absMap[code] < 0 {
			continue
		}
		if v, ok := src.absValue(code); ok {
			d.emit(t, platform.PadAxis, int(d.l.absMap[code]), d.l.axisValue(code, int(v)))
		}
	}
	for code := absHat0X; code <= absHat3Y; code++ {
		hat := (code - absHat0X) / 2
		if d.l.hatIndex[hat] < 0 {
			continue
		}
		if v, ok := src.absValue(code); ok {
			d.handleHat(t, hat, (code-absHat0X)%2, int(v))
		}
	}
	if keys, ok := src.keyState(); ok {
		for _, code := range d.l.keyCodes {
			down := 0
			if keys.has(code) {
				down = 1
			}
			d.emit(t, platform.PadButton, int(d.l.keyMap[code]), down)
		}
	}
}

// handle takes in one event of the kernel. now is the time to give the reports that come from asking the device again.
func (d *decoder) handle(ev inputEvent, now time.Duration, src source) {
	// After SYN_DROPPED the rest of the packet, up to the next SYN_REPORT, is to be ignored.
	if d.recovering && (ev.Type != evSyn || ev.Code != synReport) {
		return
	}
	code := int(ev.Code)
	switch ev.Type {
	case evKey:
		if code <= keyMax && d.l.keyMap[code] >= 0 {
			down := 0
			if ev.Value != 0 {
				down = 1
			}
			d.emit(ev.Time, platform.PadButton, int(d.l.keyMap[code]), down)
		}
	case evAbs:
		if code >= absHat0X && code <= absHat3Y && d.l.hatIndex[(code-absHat0X)/2] >= 0 {
			d.handleHat(ev.Time, (code-absHat0X)/2, code%2, int(ev.Value))
			return
		}
		if code < absCnt && d.l.absMap[code] >= 0 {
			d.emit(ev.Time, platform.PadAxis, int(d.l.absMap[code]), d.l.axisValue(code, int(ev.Value)))
		}
	case evSyn:
		switch code {
		case synDropped:
			d.recovering = true
		case synReport:
			if d.recovering {
				d.recovering = false
				d.pollAll(now, src)
			}
		}
	}
}

// take returns the reports made since the last call.
func (d *decoder) take() []platform.PadEvent {
	out := d.out
	d.out = nil
	return out
}
