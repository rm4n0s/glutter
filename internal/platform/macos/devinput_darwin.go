//go:build darwin && !ios

package macos

import (
	"math"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// The dev events that make pointer input (SPEC §29.6) build real NSEvents and hand them to the methods of the
// first window's view, as AppKit would: nothing is posted to the system, so a script never moves the user's
// pointer or reaches another app.

// The NSEventType values of the events that are made.
const (
	eventTypeLeftMouseDown uint64 = 1
	eventTypeLeftMouseUp   uint64 = 2
	eventTypeMouseMoved    uint64 = 5
	eventTypeMouseEntered  uint64 = 8
)

// The CGEvent fields of a scroll wheel event that the scripts set.
const (
	cgScrollWheelEventIsContinuous  = 88
	cgScrollWheelEventScrollPhase   = 99
	cgScrollWheelEventMomentumPhase = 123
	cgScrollEventUnitPixel          = 0
	cgScrollEventUnitLine           = 1
	cgScrollPhaseBegan              = 1
	cgScrollPhaseChanged            = 2
	cgScrollPhaseEnded              = 4
)

// cgEvents are the CoreGraphics functions that make a scroll event.
type cgEvents struct {
	createScroll func(source uintptr, units uint32, count uint32, w1, w2, w3 int32) uintptr
	setInt       func(ev uintptr, field uint32, value int64)
}

var cgEventFuncs = sync.OnceValues(func() (*cgEvents, error) {
	lib, err := dl.OpenFramework("CoreGraphics")
	if err != nil {
		return nil, err
	}
	f := new(cgEvents)
	if err := lib.Bind(&f.createScroll, "CGEventCreateScrollWheelEvent2"); err != nil {
		return nil, err
	}
	if err := lib.Bind(&f.setInt, "CGEventSetIntegerValueField"); err != nil {
		return nil, err
	}
	return f, nil
})

// devWindow is the window that the dev events act on: the first one that was made and is still there.
func (p *Platform) devWindow() *window {
	ws := p.sortedWindows()
	if len(ws) == 0 {
		return nil
	}
	return ws[0]
}

// runDevPointer runs a step that makes pointer input; it reports whether the step was one of those.
func (p *Platform) runDevPointer(ev DevEvent) bool {
	switch ev.Kind {
	case "move", "tap", "wheel", "panbegin", "panstep", "panend":
	default:
		return false
	}
	w := p.devWindow()
	if w == nil || w.closed {
		return true
	}
	p.guard(func() {
		switch ev.Kind {
		case "move":
			w.devMove(ev.X, ev.Y)
		case "tap":
			w.devMove(ev.X, ev.Y)
			w.devButton(ev.X, ev.Y, eventTypeLeftMouseDown)
			w.devButton(ev.X, ev.Y, eventTypeLeftMouseUp)
		case "wheel":
			w.devWheel(ev.X, ev.Y, ev.DX, ev.DY)
		case "panbegin":
			w.devPan(ev.X, ev.Y, 0, 0, cgScrollPhaseBegan)
		case "panstep":
			w.devPan(ev.X, ev.Y, ev.DX, ev.DY, cgScrollPhaseChanged)
		case "panend":
			w.devPan(ev.X, ev.Y, 0, 0, cgScrollPhaseEnded)
		}
		p.flushPointers()
	})
	return true
}

// windowPoint is the place in the window's coordinates (origin at the bottom left) of a place in logical
// pixels from the top left of the view.
func (w *window) windowPoint(x, y float64) nsPoint {
	h := send[nsRect](w.view, "bounds").Size.Height
	return objc.Send[nsPoint](w.view, sel("convertPoint:toView:"), nsPoint{X: x, Y: h - y}, objc.ID(0))
}

// devMove makes the mouse enter the window if it was not there and move to the place.
func (w *window) devMove(x, y float64) {
	loc := w.windowPoint(x, y)
	num := send[int64](w.ns, "windowNumber")
	now := w.p.s.mediaTime()
	if !w.devInside {
		w.devInside = true
		ev := class("NSEvent").Send(sel("enterExitEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:trackingNumber:userData:"),
			eventTypeMouseEntered, loc, uint64(0), now, num, objc.ID(0), int64(0), int64(0), objc.ID(0))
		w.view.Send(sel("mouseEntered:"), ev)
	}
	ev := class("NSEvent").Send(sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
		eventTypeMouseMoved, loc, uint64(0), now, num, objc.ID(0), int64(0), int64(0), float32(0))
	w.view.Send(sel("mouseMoved:"), ev)
}

// devButton makes a press or a release of the left button.
func (w *window) devButton(x, y float64, typ uint64) {
	loc := w.windowPoint(x, y)
	pressure := float32(1)
	selector := "mouseDown:"
	if typ == eventTypeLeftMouseUp {
		pressure, selector = 0, "mouseUp:"
	}
	ev := class("NSEvent").Send(sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
		typ, loc, uint64(0), w.p.s.mediaTime(), send[int64](w.ns, "windowNumber"), objc.ID(0), int64(0), int64(1), pressure)
	w.view.Send(sel(selector), ev)
}

// scrollEvent makes the NSEvent of a scroll wheel through a CGEvent, which is the only way to make one.
func scrollEvent(units uint32, dy, dx int32, continuous bool, phase int64) (objc.ID, error) {
	f, err := cgEventFuncs()
	if err != nil {
		return 0, err
	}
	cg := f.createScroll(0, units, 2, dy, dx, 0)
	if cg == 0 {
		return 0, nil
	}
	defer cf.Release(cf.Ref(cg))
	if continuous {
		f.setInt(cg, cgScrollWheelEventIsContinuous, 1)
	}
	if phase != 0 {
		f.setInt(cg, cgScrollWheelEventScrollPhase, phase)
	}
	return class("NSEvent").Send(sel("eventWithCGEvent:"), objc.ID(cg)), nil
}

// devWheel makes a notched wheel turn: the scroll in logical pixels is lines of 40, at least one.
func (w *window) devWheel(x, y, dx, dy float64) {
	lines := func(v float64) int32 {
		if v == 0 {
			return 0
		}
		n := int32(math.Round(v / scrollPixelsPerLine))
		if n == 0 {
			n = int32(math.Copysign(1, v))
		}
		return n
	}
	w.devMove(x, y)
	// AppKit's wheel is positive for content that moves down, the opposite of the scripts' delta.
	ev, err := scrollEvent(cgScrollEventUnitLine, -lines(dy), -lines(dx), false, 0)
	if err != nil || ev == 0 {
		w.p.log.Warn("a scroll event could not be made", "err", err)
		return
	}
	w.p.ptr.scrollWheel(w.scrollInputAt(ev, w.windowPoint(x, y)))
}

// devPan makes a phase of a two-finger scroll of the trackpad: the delta is in points, positive for content
// that moves down and to the right.
func (w *window) devPan(x, y, dx, dy float64, phase int64) {
	w.devMove(x, y)
	ev, err := scrollEvent(cgScrollEventUnitPixel, int32(math.Round(dy)), int32(math.Round(dx)), true, phase)
	if err != nil || ev == 0 {
		w.p.log.Warn("a scroll event could not be made", "err", err)
		return
	}
	w.p.ptr.scrollWheel(w.scrollInputAt(ev, w.windowPoint(x, y)))
}

// scrollInputAt is scrollInputOf for an event that has no place of its own.
func (w *window) scrollInputAt(ev objc.ID, loc nsPoint) pointerInput {
	in := w.inputAt(ev, loc)
	in.dx = send[float64](ev, "scrollingDeltaX")
	in.dy = send[float64](ev, "scrollingDeltaY")
	in.precise = send[bool](ev, "hasPreciseScrollingDeltas")
	in.phase = evPhase(send[uint64](ev, "phase"))
	in.momentum = evPhase(send[uint64](ev, "momentumPhase"))
	in.time = w.p.frameTime(w.p.s.mediaTime())
	return in
}
