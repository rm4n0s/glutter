//go:build ios

package ios

import (
	"sort"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// inputMethods are the methods of the view that take the input (SPEC §30.4): the touches of a finger, the Pencil and the pointer's
// clicks, the gestures of the iPad pointer (hover, scroll, pinch, rotation) and the presses of a hardware keyboard. Each finds its window by
// the handle of the view and runs under the platform's guard.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm touchesBegan:withEvent:, pressesBegan:withEvent:
func inputMethods() []objcx.Method {
	touch := func(name string) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, touches, event objc.ID) {
			if w := windowOf(self); w != nil {
				w.p.guard(func() { w.touches(touches, event) })
			}
		}}
	}
	press := func(name string) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, sel objc.SEL, presses, event objc.ID) {
			w := windowOf(self)
			if w == nil || !w.p.guardBool(func() bool { return w.presses(presses) }) {
				self.SendSuper(sel, presses, event) // what the framework did not take goes on to UIKit: shortcuts, the system's own keys
			}
		}}
	}
	gesture := func(name string, fn func(*window, objc.ID)) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, recognizer objc.ID) {
			if w := windowOf(self); w != nil {
				w.p.guard(func() { fn(w, recognizer) })
			}
		}}
	}
	return []objcx.Method{
		touch("touchesBegan:withEvent:"), touch("touchesMoved:withEvent:"), touch("touchesEnded:withEvent:"), touch("touchesCancelled:withEvent:"),
		press("pressesBegan:withEvent:"), press("pressesChanged:withEvent:"), press("pressesEnded:withEvent:"), press("pressesCancelled:withEvent:"),
		{Name: "canBecomeFirstResponder", Fn: func(objc.ID, objc.SEL) bool { return true }},
		gesture("hover:", (*window).hover), gesture("scroll:", (*window).scroll), gesture("pinch:", (*window).pinch), gesture("rotate:", (*window).rotate),
	}
}

// guardBool is guard for a callback that answers: the answer is false when the callback panicked.
func (p *Platform) guardBool(fn func() bool) (ok bool) {
	p.guard(func() { ok = fn() })
	return ok
}

// addRecognizers gives the view the gesture recognizers of the iPad pointer: hover, the scroll of a mouse wheel or a trackpad's two-finger
// pan (which no finger makes: it takes no touch type), and the pinch and the rotation of a trackpad (which only the indirect pointer makes).
// The view is their target.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm addInternalPlugins, viewDidLoad (the recognizers)
func (w *window) addRecognizers() {
	add := func(name, action string, configure func(rec objc.ID)) {
		rec := class(name).Send(sel("alloc")).Send(sel("initWithTarget:action:"), w.view, sel(action))
		if rec == 0 {
			return
		}
		configure(rec)
		w.view.Send(sel("addGestureRecognizer:"), rec)
		objcx.Release(rec) // the view keeps it
	}
	indirect := func() objc.ID { return objcx.NewArray(objcx.NewNumberInt(int64(touchIndirectPointer))) }
	add("UIHoverGestureRecognizer", "hover:", func(rec objc.ID) {})
	add("UIPanGestureRecognizer", "scroll:", func(rec objc.ID) {
		rec.Send(sel("setAllowedScrollTypesMask:"), int64(3)) // continuous and discrete
		rec.Send(sel("setAllowedTouchTypes:"), objcx.NewArray())
		rec.Send(sel("setMaximumNumberOfTouches:"), uint64(0))
	})
	add("UIPinchGestureRecognizer", "pinch:", func(rec objc.ID) {
		rec.Send(sel("setAllowedTouchTypes:"), indirect())
		rec.Send(sel("setCancelsTouchesInView:"), false)
	})
	add("UIRotationGestureRecognizer", "rotate:", func(rec objc.ID) {
		rec.Send(sel("setAllowedTouchTypes:"), indirect())
		rec.Send(sel("setCancelsTouchesInView:"), false)
	})
}

// now is the time of an input event that carries none: the time since the platform started.
func (w *window) now() time.Duration { return w.p.frameTime(w.p.s.mediaTime()) }

// touches converts the touches of one UIEvent: each moved touch brings the coalesced touches that UIKit made between two frames, oldest
// first, so that a fast stroke keeps its shape.
//
// Flutter: FlutterViewController.mm dispatchTouches:pointerDataChangeOverride:event:
func (w *window) touches(set, event objc.ID) {
	if w.p.handler == nil || w.closed {
		return
	}
	all := set.Send(sel("allObjects"))
	var samples []touchSample
	for i := 0; i < objcx.ArrayCount(all); i++ {
		t := objcx.ArrayAt(all, i)
		first := w.sampleOf(t, t)
		if first.Phase == touchMoved && event != 0 {
			if co := event.Send(sel("coalescedTouchesForTouch:"), t); co != 0 && objcx.ArrayCount(co) > 0 {
				for j := 0; j < objcx.ArrayCount(co); j++ {
					samples = append(samples, w.sampleOf(objcx.ArrayAt(co, j), t))
				}
				continue
			}
		}
		samples = append(samples, first)
	}
	sortSamples(samples)
	w.dispatchTouches(samples)
}

// sampleOf reads a UITouch; id is the touch that names it (a coalesced touch is named by the touch it was made for).
func (w *window) sampleOf(t, id objc.ID) touchSample {
	loc := send[darwin.CGPoint](t, "locationInView:", w.view)
	return touchSample{
		ID: uintptr(id), Phase: touchPhase(send[int64](t, "phase")), Kind: touchKind(send[int64](t, "type")),
		X: loc.X, Y: loc.Y,
		Force: send[float64](t, "force"), MaxForce: send[float64](t, "maximumPossibleForce"),
		Altitude: send[float64](t, "altitudeAngle"), Azimuth: send[float64](t, "azimuthAngleInView:", w.view),
		Radius: send[float64](t, "majorRadius"), Tolerance: send[float64](t, "majorRadiusTolerance"),
		Time: w.p.frameTime(send[float64](t, "timestamp")),
	}
}

// dispatchTouches converts the samples and gives the framework the pointer data.
func (w *window) dispatchTouches(samples []touchSample) {
	w.dispatch(w.p.touch.convert(w.id, samples, w.screenScale()))
}

// dispatch gives pointer data to the converter and then to the handler.
func (w *window) dispatch(data []ui.PointerData) {
	if len(data) == 0 || w.p.handler == nil {
		return
	}
	if out := w.p.conv.Convert(data); len(out) > 0 {
		w.p.handler.OnPointer(out)
	}
}

// location is the place of a recognizer in the view, in points.
func (w *window) location(rec objc.ID) (x, y float64) {
	p := send[darwin.CGPoint](rec, "locationInView:", w.view)
	return p.X, p.Y
}

// hover is the hover of the pointer.
//
// Flutter: FlutterViewController.mm hoverEvent:
func (w *window) hover(rec objc.ID) {
	x, y := w.location(rec)
	w.dispatch(hoverData(w.id, gesturePhase(send[int64](rec, "state")), x, y, w.screenScale(), w.now()))
}

// scroll is the scroll of the wheel or the two-finger pan of the trackpad.
//
// Flutter: FlutterViewController.mm scrollEvent:
func (w *window) scroll(rec objc.ID) {
	x, y := w.location(rec)
	tr := send[darwin.CGPoint](rec, "translationInView:", w.view)
	w.dispatch(w.scrollSession.data(w.id, gesturePhase(send[int64](rec, "state")), x, y, tr.X, tr.Y, 1, 0, w.screenScale(), w.now()))
}

// pinch is the pinch of the trackpad.
//
// Flutter: FlutterViewController.mm pinchEvent:
func (w *window) pinch(rec objc.ID) {
	x, y := w.location(rec)
	w.dispatch(w.pinchSession.data(w.id, gesturePhase(send[int64](rec, "state")), x, y, 0, 0, send[float64](rec, "scale"), 0, w.screenScale(), w.now()))
}

// rotate is the rotation of the trackpad.
//
// Flutter: FlutterViewController.mm rotationEvent:
func (w *window) rotate(rec objc.ID) {
	x, y := w.location(rec)
	w.dispatch(w.rotateSession.data(w.id, gesturePhase(send[int64](rec, "state")), x, y, 0, 0, 1, send[float64](rec, "rotation"), w.screenScale(), w.now()))
}

// presses converts the presses of a hardware keyboard and says whether the framework took them all (a press that is not a key is not
// taken). A press that was cancelled is an end.
//
// Flutter: FlutterViewController.mm pressesBegan:withEvent:
func (w *window) presses(set objc.ID) bool {
	if w.p.handler == nil || w.closed {
		return false
	}
	all := set.Send(sel("allObjects"))
	events := make([]pressEvent, 0, objcx.ArrayCount(all))
	for i := 0; i < objcx.ArrayCount(all); i++ {
		press := objcx.ArrayAt(all, i)
		key := send[objc.ID](press, "key")
		if key == 0 {
			return false
		}
		ev := pressEvent{
			phase: pressPhase(send[int64](press, "phase")), keyCode: uint32(send[int64](key, "keyCode")),
			modifierFlags: uint64(send[int64](key, "modifierFlags")), timestamp: w.p.frameTime(send[float64](press, "timestamp")),
			characters: nsString(send[objc.ID](key, "characters")), ignoring: nsString(send[objc.ID](key, "charactersIgnoringModifiers")),
		}
		if ev.phase == pressCancelled {
			ev.phase = pressEnded
		}
		events = append(events, ev)
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].timestamp < events[j].timestamp })
	handled := true
	for _, ev := range events {
		if !w.p.keys.handle(ev) {
			handled = false
		}
	}
	return handled
}

// nsString is the Go string of an NSString, empty for nil.
func nsString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	return objcx.String(id)
}

// sendKey gives a key data to the framework at once (platform.SyncInputHandler) and says whether it handled it; a handler that cannot answer
// at once gets the data as it comes, and the key is taken.
func (p *Platform) sendKey(d ui.KeyData) bool {
	h := p.handler
	if h == nil {
		return false
	}
	if sync, ok := h.(platform.SyncInputHandler); ok {
		return sync.OnKeySync(d)
	}
	h.OnKey(d)
	return true
}
