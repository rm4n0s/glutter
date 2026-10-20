package wayland

import (
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// pointer is the wl_pointer of the seat: it feeds the compositor's events to a
// pointerBatcher and sends what the batcher makes, one batch per wl_pointer.frame,
// to the handler. It also follows the touchpad gestures, and sets the cursor.
type pointer struct {
	p   *Platform
	obj *wl.WlPointer
	b   *input.PointerBatcher

	// focus is the window the pointer is in, and serial the serial of its enter
	// event, which set_cursor needs.
	focus  *window
	serial uint32
	// framed is whether the compositor sends wl_pointer.frame (version 5). Without
	// it every event is a batch of its own.
	framed bool

	shape *wl.WpCursorShapeDeviceV1 // nil when the compositor has no cursor shapes
	pinch *wl.ZwpPointerGesturePinchV1
	swipe *wl.ZwpPointerGestureSwipeV1
}

func newPointer(p *Platform) *pointer {
	pt := &pointer{p: p, b: input.NewPointerBatcher(p.now)}
	pt.obj = p.sess.seat.GetPointer()
	pt.framed = pt.obj.ProxyVersion() >= 5
	pt.obj.SetEvents(&wl.WlPointerEvents{
		OnEnter:        pt.onEnter,
		OnLeave:        pt.onLeave,
		OnMotion:       func(_ uint32, x, y float64) { pt.b.Motion(x, y); pt.eventDone() },
		OnButton:       pt.onButton,
		OnAxis:         func(_ uint32, axis wl.WlPointerAxis, v float64) { pt.b.Axis(axisOf(axis), v); pt.eventDone() },
		OnAxisSource:   func(src wl.WlPointerAxisSource) { pt.b.AxisSource(sourceOf(src)) },
		OnAxisStop:     func(_ uint32, axis wl.WlPointerAxis) { pt.b.AxisStop(axisOf(axis)); pt.eventDone() },
		OnAxisDiscrete: func(axis wl.WlPointerAxis, n int32) { pt.b.AxisDiscrete(axisOf(axis), n); pt.eventDone() },
		OnAxisValue120: func(axis wl.WlPointerAxis, v int32) { pt.b.AxisValue120(axisOf(axis), v); pt.eventDone() },
		OnAxisRelativeDirection: func(axis wl.WlPointerAxis, d wl.WlPointerAxisRelativeDirection) {
			pt.b.AxisDirection(axisOf(axis), d == wl.WlPointerAxisRelativeDirectionInverted)
		},
		OnFrame: pt.flush,
	})
	if m := p.sess.cursorShape; m != nil {
		pt.shape = m.GetPointer(pt.obj)
	}
	if g := p.sess.gestures; g != nil {
		pt.setUpGestures(g)
	}
	return pt
}

func (pt *pointer) onEnter(serial uint32, surface *wl.WlSurface, x, y float64) {
	w := pt.p.windowOf(surface)
	if w == nil {
		return // a surface that is not ours, such as the cursor's
	}
	pt.focus, pt.serial = w, serial
	pt.b.Scale = w.scaleFactor()
	pt.b.Enter(w.id, x, y)
	// The cursor of the previous window or app is gone once the pointer enters:
	// the client sets its own.
	pt.p.cursors.apply(pt, w.cursor)
	pt.eventDone()
}

func (pt *pointer) onLeave(_ uint32, surface *wl.WlSurface) {
	if pt.focus == nil || pt.focus.surface != surface {
		return
	}
	pt.b.Leave()
	pt.focus = nil
	pt.eventDone()
}

func (pt *pointer) onButton(serial uint32, _ uint32, button uint32, state wl.WlPointerButtonState) {
	pt.p.inputSerial = serial
	pt.p.pointerSerial = serial
	pt.b.Button(button, state == wl.WlPointerButtonStatePressed)
	pt.eventDone()
}

// eventDone sends the data at once when the compositor does not send frames.
func (pt *pointer) eventDone() {
	if !pt.framed {
		pt.flush()
	}
}

// flush ends a frame: the batch goes to the handler.
func (pt *pointer) flush() {
	if batch := pt.b.Frame(); len(batch) > 0 {
		pt.p.deliverPointers(batch)
	}
}

// flushGesture sends what a touchpad gesture produced. The gestures are not part
// of the pointer frames, so they go out at once.
func (pt *pointer) flushGesture() {
	if batch := pt.b.Take(); len(batch) > 0 {
		pt.p.deliverPointers(batch)
	}
}

func (pt *pointer) setUpGestures(g *wl.ZwpPointerGesturesV1) {
	pt.pinch = g.GetPinchGesture(pt.obj)
	pt.pinch.SetEvents(&wl.ZwpPointerGesturePinchV1Events{
		OnBegin: func(uint32, uint32, *wl.WlSurface, uint32) { pt.b.PinchBegin(); pt.flushGesture() },
		OnUpdate: func(_ uint32, dx, dy, scale, rotation float64) {
			pt.b.PinchUpdate(dx, dy, scale, rotation)
			pt.flushGesture()
		},
		OnEnd: func(uint32, uint32, int32) { pt.b.PinchEnd(); pt.flushGesture() },
	})
	pt.swipe = g.GetSwipeGesture(pt.obj)
	pt.swipe.SetEvents(&wl.ZwpPointerGestureSwipeV1Events{
		OnBegin:  func(uint32, uint32, *wl.WlSurface, uint32) { pt.b.SwipeBegin(); pt.flushGesture() },
		OnUpdate: func(_ uint32, dx, dy float64) { pt.b.SwipeUpdate(dx, dy); pt.flushGesture() },
		OnEnd:    func(uint32, uint32, int32) { pt.b.SwipeEnd(); pt.flushGesture() },
	})
}

// windowClosed makes the pointer leave a window that is destroyed.
func (pt *pointer) windowClosed(w *window) {
	if pt.focus == w {
		pt.b.Leave()
		pt.focus = nil
		pt.flush()
	}
}

// close drops the pointer. If it is in a window, it leaves first.
func (pt *pointer) close() {
	pt.b.Leave()
	pt.focus = nil
	pt.flush()
	if pt.pinch != nil {
		pt.pinch.Destroy()
	}
	if pt.swipe != nil {
		pt.swipe.Destroy()
	}
	if pt.shape != nil {
		pt.shape.Destroy()
	}
	if pt.p.sess.seatVersion >= 3 {
		pt.obj.Release() // wl_pointer.release is since version 3
	}
	pt.obj = nil
}

// deliverPointers hands a batch to the handler.
func (p *Platform) deliverPointers(batch []ui.PointerData) {
	if h := p.handler; h != nil {
		h.OnPointer(batch)
	}
}

// axisOf converts a wl_pointer axis to the shared one.
func axisOf(a wl.WlPointerAxis) input.Axis {
	if a == wl.WlPointerAxisHorizontalScroll {
		return input.AxisHorizontal
	}
	return input.AxisVertical
}

// sourceOf converts a wl_pointer axis source to the shared one.
func sourceOf(s wl.WlPointerAxisSource) input.AxisSource {
	switch s {
	case wl.WlPointerAxisSourceFinger:
		return input.AxisSourceFinger
	case wl.WlPointerAxisSourceContinuous:
		return input.AxisSourceContinuous
	case wl.WlPointerAxisSourceWheelTilt:
		return input.AxisSourceWheelTilt
	}
	return input.AxisSourceWheel
}
