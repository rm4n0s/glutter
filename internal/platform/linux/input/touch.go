//go:build linux && !android

package input

import (
	"sort"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// touchPoint is a finger that is down.
type touchPoint struct {
	view        ui.ViewID
	x, y        float64 // physical
	major       float64 // the ellipse of the contact, logical pixels across (0 when unknown)
	minor       float64
	orientation float64 // radians
}

// TouchBatcher turns the events of a touch screen into ui.PointerData. Each finger is a
// device of its own (Flutter tells pointers apart by device): it is added when it
// touches down and removed when it lifts.
//
// A TouchBatcher is used on the UI thread only.
//
// Flutter: no counterpart
type TouchBatcher struct {
	now func() time.Duration
	// Scale is the number of physical pixels per logical pixel; the positions the
	// batcher gets are logical.
	Scale  float64
	points map[int32]*touchPoint
	out    []ui.PointerData
}

// NewTouchBatcher makes a batcher whose events carry the times that now gives.
//
// Flutter: no counterpart
func NewTouchBatcher(now func() time.Duration) *TouchBatcher {
	return &TouchBatcher{now: now, Scale: 1, points: map[int32]*touchPoint{}}
}

func (t *TouchBatcher) data(id int32, pt *touchPoint, change ui.PointerChange) ui.PointerData {
	d := ui.PointerData{
		ViewID:            pt.view,
		TimeStamp:         t.now(),
		Change:            change,
		Kind:              ui.PointerDeviceKindTouch,
		Device:            TouchDeviceBase + int(id),
		PointerIdentifier: int(id),
		PhysicalX:         pt.x,
		PhysicalY:         pt.y,
		Pressure:          1,
		PressureMax:       1,
		RadiusMajor:       pt.major / 2,
		RadiusMinor:       pt.minor / 2,
		Orientation:       pt.orientation,
	}
	if change == ui.PointerChangeDown || change == ui.PointerChangeMove {
		d.Buttons = buttonPrimary // a touch is a press of the primary button
	}
	if change == ui.PointerChangeAdd || change == ui.PointerChangeRemove || change == ui.PointerChangeUp || change == ui.PointerChangeCancel {
		d.Pressure = 0
	}
	return d
}

// Down reports a finger touching the window view at the position, in logical
// pixels. A finger that is already down with the id is cancelled first.
//
// Flutter: no counterpart
func (t *TouchBatcher) Down(view ui.ViewID, id int32, x, y float64) {
	t.cancelPoint(id)
	pt := &touchPoint{view: view, x: x * t.Scale, y: y * t.Scale}
	t.points[id] = pt
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeAdd), t.data(id, pt, ui.PointerChangeDown))
}

// Motion reports a finger that moved.
//
// Flutter: no counterpart
func (t *TouchBatcher) Motion(id int32, x, y float64) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	nx, ny := x*t.Scale, y*t.Scale
	d := t.data(id, pt, ui.PointerChangeMove)
	d.PhysicalX, d.PhysicalY = nx, ny
	d.PhysicalDeltaX, d.PhysicalDeltaY = nx-pt.x, ny-pt.y
	pt.x, pt.y = nx, ny
	t.out = append(t.out, d)
}

// Up reports a finger that lifted.
//
// Flutter: no counterpart
func (t *TouchBatcher) Up(id int32) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	delete(t.points, id)
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeUp), t.data(id, pt, ui.PointerChangeRemove))
}

// Shape sets the size of the contact ellipse (its axes in logical pixels); it
// shows in the events that follow.
//
// Flutter: no counterpart
func (t *TouchBatcher) Shape(id int32, major, minor float64) {
	if pt, ok := t.points[id]; ok {
		pt.major, pt.minor = major, minor
	}
}

// Orientation sets the angle of the contact ellipse, in degrees from the
// vertical; it shows in the events that follow, in radians.
//
// Flutter: no counterpart
func (t *TouchBatcher) Orientation(id int32, degrees float64) {
	if pt, ok := t.points[id]; ok {
		pt.orientation = degrees * 3.141592653589793 / 180
	}
}

// cancelPoint cancels one finger: its input no longer goes to the window.
func (t *TouchBatcher) cancelPoint(id int32) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	delete(t.points, id)
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeCancel), t.data(id, pt, ui.PointerChangeRemove))
}

// Cancel cancels every finger, in the order of their ids.
//
// Flutter: no counterpart
func (t *TouchBatcher) Cancel() {
	for _, id := range t.ids(func(*touchPoint) bool { return true }) {
		t.cancelPoint(id)
	}
}

// CancelView cancels the fingers on a window that goes away.
//
// Flutter: no counterpart
func (t *TouchBatcher) CancelView(view ui.ViewID) {
	for _, id := range t.ids(func(pt *touchPoint) bool { return pt.view == view }) {
		t.cancelPoint(id)
	}
}

func (t *TouchBatcher) ids(match func(*touchPoint) bool) []int32 {
	var ids []int32
	for id, pt := range t.points {
		if match(pt) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Take returns the data since the last call.
//
// Flutter: no counterpart
func (t *TouchBatcher) Take() []ui.PointerData {
	out := t.out
	t.out = nil
	return out
}
