package gestures

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func newLoggedMultiTap(g *gestureTester, l *log, opts MultiTapGestureRecognizerOpts) *MultiTapGestureRecognizer {
	opts.LongTapDelay = LongPressTimeout
	tap := NewMultiTapGestureRecognizer(g.binding, opts)
	tap.OnTapDown = func(p int, _ TapDownDetails) { l.add(fmt.Sprintf("tap-down %d", p)) }
	tap.OnTapUp = func(p int, _ TapUpDetails) { l.add(fmt.Sprintf("tap-up %d", p)) }
	tap.OnTap = func(p int) { l.add(fmt.Sprintf("tap %d", p)) }
	tap.OnLongTapDown = func(p int, _ TapDownDetails) { l.add(fmt.Sprintf("long-tap-down %d", p)) }
	tap.OnTapCancel = func(p int) { l.add(fmt.Sprintf("tap-cancel %d", p)) }
	return tap
}

func TestMultiTapRecognizesPointers(t *testing.T) {
	g := newGestureTester(t)
	var l log
	tap := newLoggedMultiTap(g, &l, MultiTapGestureRecognizerOpts{})

	pointer5 := newTestPointer(5)
	down5 := pointer5.down(ui.Offset{Dx: 10, Dy: 10})
	tap.AddPointer(down5)
	g.closeArena(5)
	l.check(t, "tap-down 5")
	l.clear()
	g.route(down5)
	l.check(t)

	pointer6 := newTestPointer(6)
	down6 := pointer6.down(ui.Offset{Dx: 15, Dy: 15})
	tap.AddPointer(down6)
	g.closeArena(6)
	l.check(t, "tap-down 6")
	l.clear()
	g.route(down6)
	l.check(t)

	g.route(pointer5.move(ui.Offset{Dx: 11, Dy: 12}))
	l.check(t)
	g.route(pointer6.move(ui.Offset{Dx: 14, Dy: 13}))
	l.check(t)
	g.route(pointer5.up())
	l.check(t, "tap-up 5", "tap 5")
	l.clear()

	g.elapse(LongPressTimeout + PressTimeout)
	l.check(t, "long-tap-down 6")
	l.clear()

	g.route(pointer6.move(ui.Offset{Dx: 40, Dy: 30})) // more than the touch slop from 15,15
	l.check(t, "tap-cancel 6")
	l.clear()
	g.route(pointer6.up())
	l.check(t)
	tap.Dispose()
}

func TestMultiTapFiltersByDeviceKind(t *testing.T) {
	g := newGestureTester(t)
	var l log
	tap := newLoggedMultiTap(g, &l, MultiTapGestureRecognizerOpts{SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch}})

	touch5 := newTestPointer(5)
	down5 := touch5.down(ui.Offset{Dx: 10, Dy: 10})
	tap.AddPointer(down5)
	g.closeArena(5)
	l.check(t, "tap-down 5")
	l.clear()
	g.route(down5)
	l.check(t)

	mouse6 := newTestPointer(6)
	mouse6.kind = ui.PointerDeviceKindMouse
	down6 := mouse6.down(ui.Offset{Dx: 20, Dy: 20})
	tap.AddPointer(down6)
	g.closeArena(6)
	l.check(t)

	touch7 := newTestPointer(7)
	down7 := touch7.down(ui.Offset{Dx: 15, Dy: 15})
	tap.AddPointer(down7)
	g.closeArena(7)
	l.check(t, "tap-down 7")
	l.clear()
	g.route(down7)
	l.check(t)

	g.route(touch5.move(ui.Offset{Dx: 11, Dy: 12}))
	l.check(t)
	g.route(mouse6.move(ui.Offset{Dx: 21, Dy: 18}))
	g.route(mouse6.move(ui.Offset{Dx: 50, Dy: 40}))
	l.check(t)
	g.route(touch7.move(ui.Offset{Dx: 14, Dy: 13}))
	l.check(t)
	g.route(touch5.up())
	l.check(t, "tap-up 5", "tap 5")
	l.clear()
	g.route(mouse6.up())
	l.check(t)
	g.elapse(LongPressTimeout + PressTimeout)
	l.check(t, "long-tap-down 7")
	l.clear()
	g.route(touch7.move(ui.Offset{Dx: 40, Dy: 30}))
	l.check(t, "tap-cancel 7")
	tap.Dispose()
}
