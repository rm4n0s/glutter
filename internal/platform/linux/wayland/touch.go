package wayland

import (
	"sort"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
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

// touchBatcher turns the events of wl_touch into ui.PointerData. Each finger is a
// device of its own (Flutter tells pointers apart by device): it is added when it
// touches down and removed when it lifts.
//
// A touchBatcher is used on the UI thread only.
type touchBatcher struct {
	now    func() time.Duration
	scale  float64
	points map[int32]*touchPoint
	out    []ui.PointerData
}

func newTouchBatcher(now func() time.Duration) *touchBatcher {
	return &touchBatcher{now: now, scale: 1, points: map[int32]*touchPoint{}}
}

func (t *touchBatcher) data(id int32, pt *touchPoint, change ui.PointerChange) ui.PointerData {
	d := ui.PointerData{
		ViewID:            pt.view,
		TimeStamp:         t.now(),
		Change:            change,
		Kind:              ui.PointerDeviceKindTouch,
		Device:            touchDeviceBase + int(id),
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

// down reports a finger touching the window view at the position, in logical
// pixels. A finger that is already down with the id is cancelled first.
func (t *touchBatcher) down(view ui.ViewID, id int32, x, y float64) {
	t.cancelPoint(id)
	pt := &touchPoint{view: view, x: x * t.scale, y: y * t.scale}
	t.points[id] = pt
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeAdd), t.data(id, pt, ui.PointerChangeDown))
}

func (t *touchBatcher) motion(id int32, x, y float64) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	nx, ny := x*t.scale, y*t.scale
	d := t.data(id, pt, ui.PointerChangeMove)
	d.PhysicalX, d.PhysicalY = nx, ny
	d.PhysicalDeltaX, d.PhysicalDeltaY = nx-pt.x, ny-pt.y
	pt.x, pt.y = nx, ny
	t.out = append(t.out, d)
}

func (t *touchBatcher) up(id int32) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	delete(t.points, id)
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeUp), t.data(id, pt, ui.PointerChangeRemove))
}

// shape sets the size of the contact ellipse (wl_touch.shape: its axes in
// logical pixels); it shows in the events that follow.
func (t *touchBatcher) shape(id int32, major, minor float64) {
	if pt, ok := t.points[id]; ok {
		pt.major, pt.minor = major, minor
	}
}

// orientation sets the angle of the contact ellipse, in degrees from the
// vertical (wl_touch.orientation); it shows in the events that follow, in
// radians.
func (t *touchBatcher) orientation(id int32, degrees float64) {
	if pt, ok := t.points[id]; ok {
		pt.orientation = degrees * 3.141592653589793 / 180
	}
}

// cancelPoint cancels one finger: its input no longer goes to the window.
func (t *touchBatcher) cancelPoint(id int32) {
	pt, ok := t.points[id]
	if !ok {
		return
	}
	delete(t.points, id)
	t.out = append(t.out, t.data(id, pt, ui.PointerChangeCancel), t.data(id, pt, ui.PointerChangeRemove))
}

// cancel cancels every finger, in the order of their ids.
func (t *touchBatcher) cancel() {
	for _, id := range t.ids(func(*touchPoint) bool { return true }) {
		t.cancelPoint(id)
	}
}

// cancelView cancels the fingers on a window that goes away.
func (t *touchBatcher) cancelView(view ui.ViewID) {
	for _, id := range t.ids(func(pt *touchPoint) bool { return pt.view == view }) {
		t.cancelPoint(id)
	}
}

func (t *touchBatcher) ids(match func(*touchPoint) bool) []int32 {
	var ids []int32
	for id, pt := range t.points {
		if match(pt) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// take returns the data since the last call.
func (t *touchBatcher) take() []ui.PointerData {
	out := t.out
	t.out = nil
	return out
}

// touch is the wl_touch of the seat.
type touch struct {
	p   *Platform
	obj *wl.WlTouch
	b   *touchBatcher
}

func newTouch(p *Platform) *touch {
	t := &touch{p: p, b: newTouchBatcher(p.now)}
	// TODO(step 3.7): t.b.scale = the scale of the output.
	t.obj = p.sess.seat.GetTouch()
	t.obj.SetEvents(&wl.WlTouchEvents{
		OnDown: func(_ uint32, _ uint32, surface *wl.WlSurface, id int32, x, y float64) {
			if w := p.windowOf(surface); w != nil {
				t.b.down(w.id, id, x, y)
			}
		},
		OnUp:          func(_ uint32, _ uint32, id int32) { t.b.up(id) },
		OnMotion:      func(_ uint32, id int32, x, y float64) { t.b.motion(id, x, y) },
		OnShape:       func(id int32, major, minor float64) { t.b.shape(id, major, minor) },
		OnOrientation: func(id int32, o float64) { t.b.orientation(id, o) },
		OnFrame:       t.flush,
		OnCancel: func() {
			t.b.cancel()
			t.flush()
		},
	})
	return t
}

// flush hands the collected touch data to the handler, as one batch.
func (t *touch) flush() {
	if batch := t.b.take(); len(batch) > 0 {
		t.p.deliverPointers(batch)
	}
}

// windowClosed cancels the fingers on a window that is destroyed.
func (t *touch) windowClosed(w *window) {
	t.b.cancelView(w.id)
	t.flush()
}

// close drops the touch device. Fingers that are down are cancelled.
func (t *touch) close() {
	t.b.cancel()
	t.flush()
	if t.p.sess.seatVersion >= 3 {
		t.obj.Release() // wl_touch.release is since version 3
	}
	t.obj = nil
}
