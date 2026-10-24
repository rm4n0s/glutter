package android

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The layout of the arrays that GlutterView.sendMotion copies a MotionEvent into (Native.motionEvent).
const (
	motionMetaLen    = 8  // actionMasked, actionIndex, source, buttonState, eventTime, pointerCount, historySize, generic
	motionValuesEach = 11 // x, y, pressure, size, toolMajor, toolMinor, orientation, tilt, distance, hscroll, vscroll
)

// decodeMotion rebuilds the motion event from the arrays the Java side sent: meta (8 values), ids
// (the pointer id and the tool type of each pointer), vals (11 values for each pointer, first of the
// event, then of each historical sample, oldest first) and times (the time of each historical
// sample). It reports false when the arrays do not fit each other, an event of a Java side that is
// not this one, which is dropped.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/AndroidTouchProcessor.java (the values
// that its addPointerForIndex reads from the MotionEvent)
func decodeMotion(meta []int64, ids []int32, vals []float32, times []int64, pressureMin, pressureMax float32) (ev motionEvent, generic, ok bool) {
	if len(meta) != motionMetaLen {
		return motionEvent{}, false, false
	}
	n, history := int(meta[5]), int(meta[6])
	if n < 0 || history < 0 || len(ids) != 2*n || len(times) != history || len(vals) != motionValuesEach*n*(1+history) {
		return motionEvent{}, false, false
	}
	pointers := func(sample int) []motionPointer {
		out := make([]motionPointer, n)
		for p := range out {
			v := vals[motionValuesEach*(n*sample+p):]
			out[p] = motionPointer{
				ID: int(ids[2*p]), ToolType: int(ids[2*p+1]),
				X: float64(v[0]), Y: float64(v[1]), Pressure: float64(v[2]), Size: float64(v[3]),
				ToolMajor: float64(v[4]), ToolMinor: float64(v[5]), Orientation: float64(v[6]), Tilt: float64(v[7]),
				Distance: float64(v[8]), HScroll: float64(v[9]), VScroll: float64(v[10]),
			}
		}
		return out
	}
	ev = motionEvent{
		ActionMasked: int(meta[0]), ActionIndex: int(meta[1]), Source: int(meta[2]), ButtonState: int(meta[3]),
		EventTime: meta[4], PressureMin: float64(pressureMin), PressureMax: float64(pressureMax),
		Pointers: pointers(0),
	}
	for h := 0; h < history; h++ {
		ev.History = append(ev.History, motionSample{Time: times[h], Pointers: pointers(1 + h)})
	}
	return ev, meta[7] != 0, true
}

// motion queues a motion event for the UI thread. Events that come while no UI thread runs wait in
// the queue; the platform drops what is older than its first frame by itself (nothing is on screen
// to touch).
func (b *bridge) motion(ev motionEvent, generic bool, hScroll, vScroll float64) {
	b.post(func(p *Platform) { p.onMotion(ev, generic, hScroll, vScroll) })
}

// onMotion converts a motion event as Flutter's embedding does (touchProcessor, then the engine's
// pointer data packet converter) and gives it to the handler.
func (p *Platform) onMotion(ev motionEvent, generic bool, hScroll, vScroll float64) {
	if p.handler == nil || p.win == nil {
		return
	}
	var raw []ui.PointerData
	if generic {
		var handled bool
		if raw, handled = p.touch.OnGenericMotionEvent(ev, hScroll, vScroll); !handled {
			return
		}
	} else {
		raw = p.touch.OnTouchEvent(ev)
	}
	if data := p.conv.Convert(raw); len(data) > 0 {
		p.handler.OnPointer(data)
	}
}

// key queues a key event for the UI thread; id comes back with the answer.
func (b *bridge) key(id int, ev keyEvent) {
	b.post(func(p *Platform) { p.onKey(id, ev) })
}

// onKey converts a key event as Flutter's embedding does (keyboardManager), gives each key data to the
// framework at once (platform.SyncInputHandler) and tells the Java side whether the framework handled
// the event: one that it did not is dispatched again to the rest of the view hierarchy and the system
// (SPEC §24.5). A handler that cannot answer at once gets the data as it comes, and the event is
// taken.
func (p *Platform) onKey(id int, ev keyEvent) {
	if p.handler == nil {
		p.host.KeyHandled(id, false)
		return
	}
	sync, canSync := p.handler.(platform.SyncInputHandler)
	handled := p.keys.Handle(ev, func(d ui.KeyData) bool {
		if canSync {
			return sync.OnKeySync(d)
		}
		p.handler.OnKey(d)
		return true
	})
	p.host.KeyHandled(id, handled)
}
