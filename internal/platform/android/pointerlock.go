package android

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The pointer lock of the activity (SPEC §27.6) is the pointer capture of Android 8: the view asks
// for it (View.requestPointerCapture), the system hides the cursor and sends the motion of the
// mouse, unaccelerated, to View.onCapturedPointerEvent as the deltas in the x and y of the event
// (the source SOURCE_MOUSE_RELATIVE). The capture is the system's to take back: the user's Back,
// the loss of the focus and another app's capture end it
// (View.onPointerCaptureChange(false)). The Java side (GlutterView) forwards the three.

// opPointerCapture is the command of the activity that asks for the pointer capture (A is 1) or gives
// it up (A is 0); it is done on the main thread of Android.
const opPointerCapture = 27

// pointerLock is the state of the lock of the window.
type pointerLock struct {
	// x and y are the place of the pointer in physical pixels, where it stays.
	x, y float64
	// vx and vy are where the converter believes the mouse is relative to the held place: it works out
	// the deltas from positions and drops a hover that does not move, so the motion is added to them.
	vx, vy float64
	// id is the pointer ID of the mouse.
	id int
}

var _ platform.PointerLocker = (*window)(nil)

// LockPointer asks the system for the pointer capture. It fails with PointerLockDenied when the
// window has not the focus, which the system requires. The system answers later: a capture that it
// does not give, or takes back, reaches the handler as OnPointerLockEnded.
//
// Flutter: no counterpart
func (w *window) LockPointer() error {
	p := w.p
	if p.lock != nil {
		return nil
	}
	state, _, focused := p.b.snapshotAll()
	if !focused {
		return errors.New(ui.TagPointerLockDenied, "the window has no focus, which the system requires to capture the pointer")
	}
	x, y := p.mouseX, p.mouseY
	if !p.mouseSeen {
		x, y = float64(state.width)/2, float64(state.height)/2
	}
	p.lock = &pointerLock{x: x, y: y, id: p.mouseID}
	p.host.Command(command{Op: opPointerCapture, A: 1})
	p.lockedMotion(actionHoverMove, 0, 0, 0, time.Now().UnixMilli())
	return nil
}

// UnlockPointer gives the capture back. The handler is not told.
//
// Flutter: no counterpart
func (w *window) UnlockPointer() {
	p := w.p
	if p.lock == nil {
		return
	}
	p.endLock()
	p.host.Command(command{Op: opPointerCapture, A: 0})
}

// endLock forgets the lock and puts the converter's idea of the mouse back at the held place, where
// the pointer is for the framework and where the cursor reappears.
func (p *Platform) endLock() {
	l := p.lock
	p.lock = nil
	if l == nil || (l.vx == 0 && l.vy == 0) {
		return
	}
	hover := ui.PointerData{
		ViewID: ui.ImplicitViewID, Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindMouse,
		SignalKind: ui.PointerSignalKindNone, Device: uniquePointerIDByType(motionPointer{ID: l.id, ToolType: toolTypeMouse}),
		PhysicalX: l.x, PhysicalY: l.y, Scale: 1,
	}
	p.conv.Convert([]ui.PointerData{hover})
}

// pointerCaptureChanged handles View.onPointerCaptureChange: the system gave the capture or took it
// away. A capture that comes after the app unlocked is given back; one that goes while the app has
// the lock ends it.
func (p *Platform) pointerCaptureChanged(captured bool) {
	switch {
	case captured && p.lock == nil:
		p.host.Command(command{Op: opPointerCapture, A: 0})
	case !captured && p.lock != nil:
		p.endLock()
		if h, ok := p.handler.(platform.PointerLockHandler); ok && p.win != nil {
			h.OnPointerLockEnded(p.win.id)
		}
	}
}

// lockedMotion handles one sample of a captured pointer event: the action (ACTION_MOVE, ACTION_DOWN,
// ACTION_UP), the motion dx, dy in pixels and the buttons. The pointer stays at the held place and the
// motion is its delta. Other actions (ACTION_BUTTON_PRESS and the like) make nothing.
func (p *Platform) lockedMotion(action int, dx, dy float64, buttons int, eventTime int64) {
	l := p.lock
	if l == nil || p.handler == nil || p.win == nil {
		return
	}
	switch action {
	case actionDown, actionUp:
		dx, dy = 0, 0
	case actionMove, actionHoverMove:
		action = actionHoverMove
		if buttons != 0 {
			action = actionMove
		}
	default:
		return
	}
	pressure := 0.0
	if buttons != 0 || action == actionDown {
		pressure = 1
	}
	l.vx, l.vy = l.vx+dx, l.vy+dy
	ev := motionEvent{
		ActionMasked: action, Source: sourceMouse, ButtonState: buttons, EventTime: eventTime,
		Pointers: []motionPointer{{ID: l.id, ToolType: toolTypeMouse, X: l.x + l.vx, Y: l.y + l.vy, Pressure: pressure}},
	}
	data := p.convertMotion(ev, action == actionHoverMove, 0, 0)
	for i := range data {
		d := &data[i]
		d.PhysicalX, d.PhysicalY = l.x, l.y
		d.PhysicalDeltaX, d.PhysicalDeltaY = 0, 0
		if !d.Synthesized && (d.Change == ui.PointerChangeHover || d.Change == ui.PointerChangeMove) {
			d.PhysicalDeltaX, d.PhysicalDeltaY = dx, dy
		}
	}
	if len(data) > 0 {
		p.handler.OnPointer(data)
	}
}

// capturedMotion queues a sample of a captured pointer event for the UI thread.
func (b *bridge) capturedMotion(action int, dx, dy float64, buttons int, eventTime int64) {
	b.post(func(p *Platform) { p.lockedMotion(action, dx, dy, buttons, eventTime) })
}

// pointerCapture queues the system's answer about the capture for the UI thread.
func (b *bridge) pointerCapture(captured bool) {
	b.post(func(p *Platform) { p.pointerCaptureChanged(captured) })
}
