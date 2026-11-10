package ios

import (
	"math"
	"time"
)

// This file is the plain-Go part of the keyboard's insets (SPEC §30.3): how much of the view the keyboard covers, and the animation of the
// inset while the keyboard comes or goes, so that the app moves with it as it does in Flutter's iOS embedding.

// keyboardOverlap is how many points of the bottom of the view a keyboard covers: the keyboard's frame, in the view's coordinates, has its
// top at keyboardTop, and the view ends at viewBottom. A keyboard below the view (hidden) or with no height covers nothing, and none covers more than
// the view.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterKeyboardInsetManager.mm (the intersection of the keyboard frame with the view)
func keyboardOverlap(keyboardTop, keyboardBottom, viewTop, viewBottom float64) float64 {
	if keyboardBottom <= keyboardTop {
		return 0
	}
	top := math.Max(keyboardTop, viewTop)
	bottom := math.Min(keyboardBottom, viewBottom)
	if bottom <= top || keyboardBottom < viewBottom {
		// A keyboard that does not reach the bottom of the view (a floating keyboard on an iPad, or a hardware keyboard's shortcut bar) is
		// not an inset: the view is not moved for it.
		return 0
	}
	return viewBottom - top
}

// insetAnimation animates the bottom inset from a value to another over the duration of the keyboard's own animation, with an ease-out
// curve. It belongs to the UI thread.
//
// Flutter: FlutterKeyboardInsetManager.mm (the animation of the inset with the keyboard's duration)
type insetAnimation struct {
	from, to float64
	start    time.Duration
	duration time.Duration
	active   bool
}

// begin starts an animation at the time now (since the platform started) from the value that is shown now.
func (a *insetAnimation) begin(from, to float64, now, duration time.Duration) {
	*a = insetAnimation{from: from, to: to, start: now, duration: duration, active: duration > 0 && from != to}
	if !a.active {
		a.from = to
	}
}

// value is the inset at the time now, and whether the animation went on (false once it has ended: the value is the target).
func (a *insetAnimation) value(now time.Duration) (v float64, running bool) {
	if !a.active {
		return a.to, false
	}
	t := float64(now-a.start) / float64(a.duration)
	if t >= 1 {
		a.active = false
		return a.to, false
	}
	if t < 0 {
		t = 0
	}
	eased := 1 - (1-t)*(1-t) // ease out: the keyboard slows as it arrives
	return a.from + (a.to-a.from)*eased, true
}
