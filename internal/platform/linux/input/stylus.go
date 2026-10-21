//go:build linux

package input

import (
	"math"
	"sort"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The Linux button codes of a stylus (linux/input-event-codes.h).
const (
	btnStylus  = 0x14b // BTN_STYLUS: the first barrel button
	btnStylus2 = 0x14c // BTN_STYLUS2: the second barrel button
)

// stylusButtonBit maps the button code of a tablet tool to Flutter's button bit:
// the barrel buttons are kPrimaryStylusButton (the secondary button) and
// kSecondaryStylusButton (the middle one), the others are those of a mouse.
func stylusButtonBit(code uint32) int {
	switch code {
	case btnStylus:
		return buttonSecondary
	case btnStylus2:
		return buttonMiddle
	}
	return buttonBit(code)
}

// stylusAxisMax is the largest value of the pressure and distance axes of
// zwp_tablet_tool_v2.
const stylusAxisMax = 65535.0

// stylusTool is the state of one tablet tool.
type stylusTool struct {
	kind ui.PointerDeviceKind
	view ui.ViewID
	// x and y are the position in logical pixels; pressure and distance are 0..1.
	x, y              float64
	lastX, lastY      float64 // the position of the last data
	pressure          float64
	distance          float64
	tiltX, tiltY      float64 // degrees
	hasTilt           bool
	rotation          float64 // degrees
	hasRotation       bool
	buttons           int // the barrel and mouse buttons that are down
	present, touching bool

	// The events of the frame that is being collected.
	in, out, down, up, changed bool
}

// StylusBatcher turns the events of the tablet tools (zwp_tablet_tool_v2) into
// ui.PointerData. The events of a tool are collected until its frame event, then
// they become one change: Add when the tool comes near the surface, Hover while it
// moves there, Down, Move and Up while it touches, and Remove when it leaves. A
// tool that leaves while it touches is lifted first.
//
// A StylusBatcher is used on the UI thread only.
//
// Flutter: no counterpart
type StylusBatcher struct {
	now func() time.Duration
	// Scale is the number of physical pixels per logical pixel; the positions the
	// batcher gets are logical.
	Scale float64
	tools map[int]*stylusTool
	out   []ui.PointerData
}

// NewStylusBatcher makes a batcher whose events carry the times that now gives.
//
// Flutter: no counterpart
func NewStylusBatcher(now func() time.Duration) *StylusBatcher {
	return &StylusBatcher{now: now, Scale: 1, tools: map[int]*stylusTool{}}
}

// AddTool announces a tool. The kind is stylus for a pen, a brush, a pencil or an
// airbrush, inverted stylus for an eraser, touch for a finger and mouse for a
// mouse or a lens.
//
// Flutter: no counterpart
func (b *StylusBatcher) AddTool(id int, kind ui.PointerDeviceKind) {
	b.tools[id] = &stylusTool{kind: kind}
}

// RemoveTool forgets a tool that went away. A tool that is near a surface is
// lifted and removed first.
//
// Flutter: no counterpart
func (b *StylusBatcher) RemoveTool(id int) {
	t, ok := b.tools[id]
	if !ok {
		return
	}
	if t.present {
		t.out = true
		b.Frame(id)
	}
	delete(b.tools, id)
}

// ProximityIn reports that the tool came near the window view.
//
// Flutter: no counterpart
func (b *StylusBatcher) ProximityIn(id int, view ui.ViewID) {
	if t, ok := b.tools[id]; ok {
		t.view, t.in = view, true
	}
}

// ProximityOut reports that the tool went away from the surface.
//
// Flutter: no counterpart
func (b *StylusBatcher) ProximityOut(id int) {
	if t, ok := b.tools[id]; ok {
		t.out = true
	}
}

// Down reports that the tool touches the surface.
//
// Flutter: no counterpart
func (b *StylusBatcher) Down(id int) {
	if t, ok := b.tools[id]; ok {
		t.down = true
	}
}

// Up reports that the tool no longer touches the surface.
//
// Flutter: no counterpart
func (b *StylusBatcher) Up(id int) {
	if t, ok := b.tools[id]; ok {
		t.up = true
	}
}

// Motion reports the position of the tool, in logical pixels.
//
// Flutter: no counterpart
func (b *StylusBatcher) Motion(id int, x, y float64) {
	if t, ok := b.tools[id]; ok && (t.x != x || t.y != y || !t.present) {
		t.x, t.y, t.changed = x, y, true
	}
}

// Pressure reports the pressure of the tool, from 0 to 65535.
//
// Flutter: no counterpart
func (b *StylusBatcher) Pressure(id int, raw uint32) {
	if t, ok := b.tools[id]; ok {
		t.pressure, t.changed = math.Min(float64(raw)/stylusAxisMax, 1), true
	}
}

// Distance reports the distance of the tool from the surface, from 0 to 65535.
//
// Flutter: no counterpart
func (b *StylusBatcher) Distance(id int, raw uint32) {
	if t, ok := b.tools[id]; ok {
		t.distance, t.changed = math.Min(float64(raw)/stylusAxisMax, 1), true
	}
}

// Tilt reports the tilt of the tool along the x and y axes, in degrees from the
// normal of the surface.
//
// Flutter: no counterpart
func (b *StylusBatcher) Tilt(id int, x, y float64) {
	if t, ok := b.tools[id]; ok {
		t.tiltX, t.tiltY, t.hasTilt, t.changed = x, y, true, true
	}
}

// Rotation reports the rotation of the tool around its axis, in degrees.
//
// Flutter: no counterpart
func (b *StylusBatcher) Rotation(id int, degrees float64) {
	if t, ok := b.tools[id]; ok {
		t.rotation, t.hasRotation, t.changed = degrees, true, true
	}
}

// Button reports that a button of the tool (a Linux button code) was pressed or
// released.
//
// Flutter: no counterpart
func (b *StylusBatcher) Button(id int, code uint32, pressed bool) {
	t, ok := b.tools[id]
	if !ok {
		return
	}
	bit := stylusButtonBit(code)
	if bit == 0 {
		return
	}
	if pressed {
		t.buttons |= bit
	} else {
		t.buttons &^= bit
	}
	t.changed = true
}

// tiltAngles returns the tilt from the normal of the surface and the azimuth of
// the tool, in radians: Flutter's PointerData.tilt (0 to pi/2) and orientation.
func tiltAngles(xDeg, yDeg float64) (tilt, azimuth float64) {
	tx, ty := math.Tan(xDeg*math.Pi/180), math.Tan(yDeg*math.Pi/180)
	tilt = math.Min(math.Atan(math.Hypot(tx, ty)), math.Pi/2)
	if tx == 0 && ty == 0 {
		return 0, 0
	}
	azimuth = math.Atan2(ty, tx)
	if azimuth < 0 {
		azimuth += 2 * math.Pi
	}
	return tilt, azimuth
}

func (b *StylusBatcher) data(id int, t *stylusTool, change ui.PointerChange) ui.PointerData {
	d := ui.PointerData{
		ViewID:            t.view,
		TimeStamp:         b.now(),
		Change:            change,
		Kind:              t.kind,
		Device:            StylusDeviceBase + id,
		PointerIdentifier: id,
		PhysicalX:         t.x * b.Scale,
		PhysicalY:         t.y * b.Scale,
		PressureMax:       1,
		DistanceMax:       1,
		Distance:          t.distance,
	}
	if t.touching {
		d.Pressure = t.pressure
	}
	if t.hasTilt {
		d.Tilt, d.Orientation = tiltAngles(t.tiltX, t.tiltY)
	} else if t.hasRotation {
		d.Orientation = t.rotation * math.Pi / 180
	}
	switch change {
	case ui.PointerChangeDown, ui.PointerChangeMove:
		d.Buttons = buttonPrimary | t.buttons
	case ui.PointerChangeHover:
		d.Buttons = t.buttons
	}
	return d
}

// emit adds the data of a change and makes the deltas from the last position the
// tool reported.
func (b *StylusBatcher) emit(id int, t *stylusTool, change ui.PointerChange, lastX, lastY float64) {
	d := b.data(id, t, change)
	if change == ui.PointerChangeHover || change == ui.PointerChangeMove {
		d.PhysicalDeltaX, d.PhysicalDeltaY = d.PhysicalX-lastX, d.PhysicalY-lastY
	}
	b.out = append(b.out, d)
}

// Frame ends the group of events of the tool and turns them into data.
//
// Flutter: no counterpart
func (b *StylusBatcher) Frame(id int) {
	t, ok := b.tools[id]
	if !ok {
		return
	}
	if t.in {
		t.present = true
		t.lastX, t.lastY = t.x, t.y
		b.emit(id, t, ui.PointerChangeAdd, 0, 0)
	}
	if t.present {
		switch {
		case t.down && !t.touching:
			t.touching = true
			b.emit(id, t, ui.PointerChangeDown, 0, 0)
		case t.up && t.touching:
			t.touching = false
			b.emit(id, t, ui.PointerChangeUp, 0, 0)
		case t.changed && !t.in:
			change := ui.PointerChangeHover
			if t.touching {
				change = ui.PointerChangeMove
			}
			b.emit(id, t, change, t.lastX*b.Scale, t.lastY*b.Scale)
		}
	}
	if t.out && t.present {
		if t.touching {
			t.touching = false
			b.emit(id, t, ui.PointerChangeUp, 0, 0)
		}
		b.emit(id, t, ui.PointerChangeRemove, 0, 0)
		t.present = false
	}
	t.lastX, t.lastY = t.x, t.y
	t.in, t.out, t.down, t.up, t.changed = false, false, false, false, false
}

// CancelView cancels the tools that touch a window that goes away and removes the
// ones that are near it.
//
// Flutter: no counterpart
func (b *StylusBatcher) CancelView(view ui.ViewID) {
	ids := make([]int, 0, len(b.tools))
	for id, t := range b.tools {
		if t.present && t.view == view {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		t := b.tools[id]
		if t.touching {
			t.touching = false
			b.out = append(b.out, b.data(id, t, ui.PointerChangeCancel))
		}
		b.out = append(b.out, b.data(id, t, ui.PointerChangeRemove))
		t.present = false
	}
}

// Take returns the data since the last call.
//
// Flutter: no counterpart
func (b *StylusBatcher) Take() []ui.PointerData {
	out := b.out
	b.out = nil
	return out
}
