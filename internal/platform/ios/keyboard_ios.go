//go:build ios

package ios

import (
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// This file is the soft keyboard's insets (SPEC §30.3): the observer of UIKit's keyboard notifications, and the inset that each window
// gets from them, animated with the keyboard's own duration (the plain-Go part is in inset.go).

// keyboardNotification is the name of the notification that UIKit posts when the keyboard's frame is about to change: it comes, goes,
// changes size or moves with an orientation. The frame that the keyboard ends at, in screen coordinates, and the time that it takes are in
// its user info.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterKeyboardInsetManager.mm (UIKeyboardWillChangeFrameNotification)
const keyboardNotification = "UIKeyboardWillChangeFrameNotification"

// registerKeyboardObserver registers the class of the observer, which the notification centre calls with the notification.
func registerKeyboardObserver() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "KeyboardObserver", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "keyboardWillChangeFrame:", Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				if p, ok := objcx.LookupAs[*Platform](objcx.HandleOf(self)); ok && !p.closed {
					p.guard(func() { p.keyboardFrameChanged(note) })
				}
			}},
		},
	})
}

// observeKeyboard starts listening to the keyboard's frame.
func (p *Platform) observeKeyboard() {
	p.observer = p.s.keyboard.New(p.handle)
	name := objcx.NewString(keyboardNotification)
	defer objcx.Release(name)
	class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("addObserver:selector:name:object:"), p.observer, sel("keyboardWillChangeFrame:"), name, objc.ID(0))
}

// stopObservingKeyboard stops listening and releases the observer.
func (p *Platform) stopObservingKeyboard() {
	if p.observer == 0 {
		return
	}
	class("NSNotificationCenter").Send(sel("defaultCenter")).Send(sel("removeObserver:"), p.observer)
	objcx.Release(p.observer)
	p.observer = 0
}

// keyboardFrameChanged reads the frame that the keyboard ends at and the duration of its animation, and gives each window its inset.
func (p *Platform) keyboardFrameChanged(note objc.ID) {
	info := send[objc.ID](note, "userInfo")
	if info == 0 {
		return
	}
	get := func(key string) objc.ID {
		k := objcx.NewString(key)
		defer objcx.Release(k)
		return info.Send(sel("objectForKey:"), k)
	}
	frameValue := get("UIKeyboardFrameEndUserInfoKey")
	if frameValue == 0 {
		return
	}
	frame := send[darwin.CGRect](frameValue, "CGRectValue")
	var duration time.Duration
	if d := get("UIKeyboardAnimationDurationUserInfoKey"); d != 0 {
		duration = time.Duration(objcx.Float(d) * float64(time.Second))
	}
	for _, w := range p.sortedWindows() {
		w.setKeyboardFrame(frame, duration)
	}
}

// setKeyboardFrame takes the frame of the keyboard, in screen coordinates: how much of the window's view it covers is the inset that the
// window animates to.
func (w *window) setKeyboardFrame(frame darwin.CGRect, duration time.Duration) {
	if w.closed || w.view == 0 || w.uiw == 0 {
		return
	}
	screen := send[objc.ID](class("UIScreen").Send(sel("mainScreen")), "coordinateSpace")
	r := send[darwin.CGRect](w.view, "convertRect:fromCoordinateSpace:", frame, screen)
	bounds := send[darwin.CGRect](w.view, "bounds")
	overlap := keyboardOverlap(r.Origin.Y, r.Origin.Y+r.Size.Height, bounds.Origin.Y, bounds.Origin.Y+bounds.Size.Height)
	w.animateInset(overlap*w.screenScale(), duration)
}

// animateInset moves the inset from where it is to the target over the duration, a frame of 16 ms at a time; none when there is no
// duration or the window is not shown.
func (w *window) animateInset(target float64, duration time.Duration) {
	w.insetTo = target
	if !w.shown {
		duration = 0
	}
	w.insetAnim.begin(w.inset, target, w.now(), duration)
	if w.insetTimer != 0 {
		w.p.timers.Cancel(w.insetTimer)
		w.insetTimer = 0
		w.p.rearm()
	}
	w.insetTick()
}

// insetTick applies the animation's value to the inset, tells the framework, and arms the next step while it runs.
func (w *window) insetTick() {
	w.insetTimer = 0
	if w.closed {
		return
	}
	v, running := w.insetAnim.value(w.now())
	if v != w.inset {
		w.inset = v
		w.reportMetrics(false)
	}
	if running {
		w.insetTimer = w.p.addTimer(time.Now().Add(devFrame), w.insetTick)
	}
}
