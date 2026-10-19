package gestures

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func nearOffset(a, b ui.Offset, eps float64) bool {
	return math.Abs(a.Dx-b.Dx) <= eps && math.Abs(a.Dy-b.Dy) <= eps
}

func TestDragRecognizesPan(t *testing.T) {
	g := newGestureTester(t)
	pan := NewPanGestureRecognizer(g.binding)
	tap := NewTapGestureRecognizer(g.binding)
	tap.OnTap = func() {}
	var started, ended, tapped bool
	var delta foundation.Opt[ui.Offset]
	pan.OnStart = func(DragStartDetails) { started = true }
	pan.OnUpdate = func(d DragUpdateDetails) { delta = foundation.Some(d.Delta) }
	pan.OnEnd = func(DragEndDetails) { ended = true }
	tap.OnTap = func() { tapped = true }

	pointer := newTestPointer(5)
	down := pointer.down(off(10, 10))
	pan.AddPointer(down)
	tap.AddPointer(down)
	g.closeArena(5)
	g.route(down)
	if started || delta.IsSome() || ended || tapped {
		t.Fatal("something happened at the down event")
	}
	g.route(pointer.move(off(20, 20))) // 14 in total
	if started {
		t.Fatal("started at 14 < 18")
	}
	g.route(pointer.move(off(20, 30))) // 22 in total
	if !started {
		t.Fatal("did not start at 22 > 18")
	}
	started = false
	g.route(pointer.move(off(20, 25)))
	if started || delta != foundation.Some(off(0, -5)) || ended || tapped {
		t.Fatalf("started = %v, delta = %v, ended = %v, tapped = %v", started, delta, ended, tapped)
	}
	delta = foundation.None[ui.Offset]()
	g.route(pointer.up())
	if !ended || delta.IsSome() || tapped {
		t.Errorf("ended = %v, delta = %v, tapped = %v", ended, delta, tapped)
	}
}

func TestDragStartPosition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		behavior  DragStartBehavior
		wantStart ui.Offset
	}{
		{"the most recent point is reported to onStart by default", DragStartBehaviorUnspecified, off(30, 0)},
		{"the most recent point is reported with a start configuration", DragStartBehaviorStart, off(30, 0)},
		{"the initial down point is reported with a down configuration", DragStartBehaviorDown, off(10, 10)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: tc.behavior})
			competing := NewVerticalDragGestureRecognizer(g.binding)
			competing.OnStart = func(DragStartDetails) {}
			var at ui.Offset
			drag.OnStart = func(d DragStartDetails) { at = d.GlobalPosition }
			pointer := newTestPointer(5)
			down := pointer.down(off(10, 10))
			drag.AddPointer(down)
			competing.AddPointer(down)
			g.closeArena(5)
			g.route(down)
			g.route(pointer.move(off(30, 0)))
			if at != tc.wantStart {
				t.Errorf("position at onStart = %v, want %v", at, tc.wantStart)
			}
		})
	}
}

func TestDragRecognizesDragWithDownBehavior(t *testing.T) {
	g := newGestureTester(t)
	drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorDown})
	var started, ended bool
	var primaryDelta foundation.Opt[float64]
	drag.OnStart = func(DragStartDetails) { started = true }
	drag.OnUpdate = func(d DragUpdateDetails) { primaryDelta = d.PrimaryDelta }
	drag.OnEnd = func(DragEndDetails) { ended = true }
	pointer := newTestPointer(5)
	down := pointer.down(off(10, 10))
	drag.AddPointer(down)
	g.closeArena(5)
	if started || primaryDelta.IsSome() || ended {
		t.Fatal("something happened before the down event")
	}
	g.route(down)
	if !started || primaryDelta.IsSome() || ended {
		t.Fatalf("after down: started = %v, delta = %v, ended = %v", started, primaryDelta, ended)
	}
	started = false
	g.route(pointer.move(off(20, 25)))
	if started || primaryDelta != foundation.Some(10.0) || ended {
		t.Fatalf("after the move: started = %v, delta = %v", started, primaryDelta)
	}
	primaryDelta = foundation.None[float64]()
	g.route(pointer.move(off(20, 25)))
	if primaryDelta != foundation.Some(0.0) {
		t.Fatalf("after the second move: delta = %v", primaryDelta)
	}
	primaryDelta = foundation.None[float64]()
	g.route(pointer.up())
	if primaryDelta.IsSome() || !ended {
		t.Errorf("after up: delta = %v, ended = %v", primaryDelta, ended)
	}
}

func TestDragRejectsUnsupportedDeviceKind(t *testing.T) {
	g := newGestureTester(t)
	drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{
		DragStartBehavior: DragStartBehaviorDown, SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch},
	})
	var l log
	drag.OnStart = func(DragStartDetails) { l.add("start") }
	drag.OnUpdate = func(DragUpdateDetails) { l.add("update") }
	drag.OnEnd = func(DragEndDetails) { l.add("end") }
	pointer := newTestPointer(5)
	pointer.kind = ui.PointerDeviceKindMouse
	down := pointer.down(off(10, 10))
	drag.AddPointer(down)
	run(g, &l, closeArena(5), route(down), route(pointer.move(off(20, 25))), route(pointer.move(off(20, 25))), route(pointer.up()), want())
}

func TestDragVelocity(t *testing.T) {
	t.Run("clamps the max velocity", func(t *testing.T) {
		g := newGestureTester(t)
		drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorDown})
		var velocity Velocity
		var primary foundation.Opt[float64]
		drag.OnEnd = func(d DragEndDetails) { velocity, primary = d.Velocity, d.PrimaryVelocity }
		pointer := newTestPointer(5)
		down := pointer.downAt(off(10, 25), ms(10))
		drag.AddPointer(down)
		g.closeArena(5)
		g.route(down)
		g.route(pointer.moveAt(off(20, 25), ms(10)))
		for i := 1; i <= 10; i++ {
			g.route(pointer.moveAt(off(20+10*float64(i), 25), ms(10+i)))
		}
		g.route(pointer.upAt(ms(20)))
		if dx := velocity.PixelsPerSecond.Dx; dx < 0.99*MaxFlingVelocity || dx > MaxFlingVelocity {
			t.Errorf("velocity dx = %v, want it in [%v, %v]", dx, 0.99*MaxFlingVelocity, MaxFlingVelocity)
		}
		if math.Abs(velocity.PixelsPerSecond.Dy) > 1e-9 || primary != foundation.Some(velocity.PixelsPerSecond.Dx) {
			t.Errorf("velocity = %v, primary = %v", velocity, primary)
		}
	})

	dragToEnd := func(g *gestureTester, r interface {
		GestureRecognizer
		setOnEnd(GestureDragEndCallback)
	}, v ui.Offset) DragEndDetails {
		var actual DragEndDetails
		r.setOnEnd(func(d DragEndDetails) { actual = d })
		pointer := newTestPointer(1)
		down := pointer.down(ui.Offset{})
		r.AddPointer(down)
		g.closeArena(1)
		g.route(down)
		g.route(pointer.moveAt(v.Mul(0.025), ms(25)))
		g.route(pointer.moveAt(v.Mul(0.050), ms(50)))
		g.route(pointer.upAt(ms(50)))
		return actual
	}
	t.Run("clamps the max pan velocity in 2D, isotropically", func(t *testing.T) {
		g := newGestureTester(t)
		pan := &panEnd{NewPanGestureRecognizer(g.binding)}
		for _, tc := range []struct{ in, want ui.Offset }{
			{off(400, 400), off(400, 400)}, {off(2000, -2000), off(2000, -2000)},
			{off(-8000, -8000), off(-5656.9, -5656.9)}, {off(-8000, 6000), off(-6400, 4800)},
			{off(-9000, 0), off(-8000, 0)}, {off(-9000, -1000), off(-7951.1, -883.5)},
			{off(-1000, 9000), off(-883.5, 7951.1)}, {off(0, 9000), off(0, 8000)},
		} {
			actual := dragToEnd(g, pan, tc.in)
			if !nearOffset(actual.Velocity.PixelsPerSecond, tc.want, 0.1) || actual.PrimaryVelocity.IsSome() {
				t.Errorf("drag %v: velocity = %v (primary %v), want %v", tc.in, actual.Velocity.PixelsPerSecond, actual.PrimaryVelocity, tc.want)
			}
		}
	})
	t.Run("clamps the max vertical-drag velocity vertically", func(t *testing.T) {
		g := newGestureTester(t)
		r := &verticalEnd{NewVerticalDragGestureRecognizer(g.binding)}
		for _, tc := range []struct {
			in   ui.Offset
			want float64
		}{{off(500, 400), 400}, {off(3000, -2000), -2000}, {off(-9000, -9000), -8000}, {off(-9000, 0), 0},
			{off(-9000, 1000), 1000}, {off(-1000, -9000), -8000}, {off(0, -9000), -8000}} {
			actual := dragToEnd(g, r, tc.in)
			pv, _ := actual.PrimaryVelocity.Get()
			if math.Abs(pv-tc.want) > 0.1 || actual.Velocity.PixelsPerSecond.Dx != 0 || actual.Velocity.PixelsPerSecond.Dy != pv {
				t.Errorf("drag %v: primary = %v, velocity = %v, want %v", tc.in, pv, actual.Velocity.PixelsPerSecond, tc.want)
			}
		}
	})
	t.Run("clamps the max horizontal-drag velocity horizontally", func(t *testing.T) {
		g := newGestureTester(t)
		r := &horizontalEnd{NewHorizontalDragGestureRecognizer(g.binding)}
		for _, tc := range []struct {
			in   ui.Offset
			want float64
		}{{off(500, 400), 500}, {off(3000, -2000), 3000}, {off(-9000, -9000), -8000}, {off(-9000, 0), -8000},
			{off(-9000, 1000), -8000}, {off(-1000, -9000), -1000}, {off(0, -9000), 0}} {
			actual := dragToEnd(g, r, tc.in)
			pv, _ := actual.PrimaryVelocity.Get()
			if math.Abs(pv-tc.want) > 0.1 || actual.Velocity.PixelsPerSecond.Dx != pv || actual.Velocity.PixelsPerSecond.Dy != 0 {
				t.Errorf("drag %v: primary = %v, velocity = %v, want %v", tc.in, pv, actual.Velocity.PixelsPerSecond, tc.want)
			}
		}
	})
	t.Run("ignores synthesized pointer events for velocity tracking", func(t *testing.T) {
		g := newGestureTester(t)
		drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorDown})
		var velocity Velocity
		drag.OnEnd = func(d DragEndDetails) { velocity = d.Velocity }
		pointer := newTestPointer(1)
		down := pointer.downAt(off(10, 25), ms(10))
		drag.AddPointer(down)
		g.closeArena(1)
		g.route(down)
		for i := 1; i <= 4; i++ {
			g.route(pointer.moveAt(off(10+10*float64(i), 25), ms(10+10*i)))
		}
		g.route(NewPointerMoveEvent(PointerEventOpts{Pointer: 1, Position: off(51, 26), TimeStamp: ms(60), Synthesized: true}))
		g.route(pointer.upAt(ms(70)))
		if math.Abs(velocity.PixelsPerSecond.Dx-1000) > 1e-6 || math.Abs(velocity.PixelsPerSecond.Dy) > 1e-6 {
			t.Errorf("velocity = %v, want (1000, 0)", velocity)
		}
	})
	t.Run("quick flicks have velocity", func(t *testing.T) {
		g := newGestureTester(t)
		drag := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorDown})
		var velocity Velocity
		drag.OnEnd = func(d DragEndDetails) { velocity = d.Velocity }
		pointer := newTestPointer(1)
		down := pointer.downAt(off(10, 25), ms(10))
		drag.AddPointer(down)
		g.closeArena(1)
		g.route(down)
		g.route(pointer.moveAt(off(20, 25), ms(20)))
		g.route(pointer.moveAt(off(30, 25), ms(30)))
		g.route(pointer.upAt(ms(40)))
		if math.Abs(velocity.PixelsPerSecond.Dx-1000) > 1e-6 || math.Abs(velocity.PixelsPerSecond.Dy) > 1e-6 {
			t.Errorf("velocity = %v, want (1000, 0)", velocity)
		}
	})
}

type panEnd struct{ *PanGestureRecognizer }

func (r *panEnd) setOnEnd(f GestureDragEndCallback) { r.OnEnd = f }

type verticalEnd struct{ *VerticalDragGestureRecognizer }

func (r *verticalEnd) setOnEnd(f GestureDragEndCallback) { r.OnEnd = f }

type horizontalEnd struct {
	*HorizontalDragGestureRecognizer
}

func (r *horizontalEnd) setOnEnd(f GestureDragEndCallback) { r.OnEnd = f }

func TestDragConsistentButtons(t *testing.T) {
	newPair := func(g *gestureTester, l *log) (*PanGestureRecognizer, *TapGestureRecognizer) {
		tap := NewTapGestureRecognizer(g.binding)
		tap.OnTap = func() {} // needs a callback to enable competition
		pan := NewPanGestureRecognizer(g.binding)
		pan.OnStart = func(DragStartDetails) { l.add("start") }
		pan.OnDown = func(DragDownDetails) { l.add("down") }
		pan.OnUpdate = func(DragUpdateDetails) { l.add("update") }
		pan.OnCancel = func() { l.add("cancel") }
		pan.OnEnd = func(DragEndDetails) { l.add("end") }
		return pan, tap
	}
	mousePointer := func(n int) *testPointer {
		p := newTestPointer(n)
		p.kind = ui.PointerDeviceKindMouse
		return p
	}
	press := func(g *gestureTester, pan *PanGestureRecognizer, tap *TapGestureRecognizer, p *testPointer) *PointerDownEvent {
		down := p.down(off(10, 10))
		pan.AddPointer(down)
		tap.AddPointer(down)
		g.closeArena(p.pointer)
		g.route(down)
		return down
	}
	t.Run("button change before acceptance leads to an immediate cancel", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		pan, tap := newPair(g, &l)
		p := mousePointer(5)
		press(g, pan, tap, p)
		l.check(t, "down")
		g.route(p.moveButtons(off(30, 30), SecondaryButton))
		l.check(t, "down", "cancel")
		g.route(p.up())
	})
	t.Run("button change before acceptance does not prevent the next drag", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		pan, tap := newPair(g, &l)
		p := mousePointer(5)
		press(g, pan, tap, p)
		g.route(p.moveButtons(off(10, 10), SecondaryButton))
		g.route(p.up())
		l.check(t, "down", "cancel")
		l.clear()
		p2 := mousePointer(6)
		press(g, pan, tap, p2)
		l.check(t, "down")
		g.route(p2.move(off(30, 30)))
		l.check(t, "down", "start")
		g.route(p2.up())
		l.check(t, "down", "start", "end")
	})
	t.Run("button change after acceptance leads to an immediate end", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		pan, tap := newPair(g, &l)
		p := mousePointer(5)
		press(g, pan, tap, p)
		l.check(t, "down")
		g.route(p.move(off(30, 30)))
		l.check(t, "down", "start")
		g.route(p.moveButtons(off(30, 30), SecondaryButton))
		l.check(t, "down", "start", "end")
		g.route(p.move(off(50, 50)))
		l.check(t, "down", "start", "end")
		g.route(p.up())
	})
	t.Run("button change after acceptance does not prevent the next drag", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		pan, tap := newPair(g, &l)
		p := mousePointer(5)
		press(g, pan, tap, p)
		g.route(p.move(off(30, 30)))
		g.route(p.moveButtons(off(30, 31), SecondaryButton))
		g.route(p.up())
		l.check(t, "down", "start", "end")
		l.clear()
		p2 := mousePointer(6)
		press(g, pan, tap, p2)
		l.check(t, "down")
		g.route(p2.move(off(30, 30)))
		l.check(t, "down", "start")
		g.route(p2.up())
		l.check(t, "down", "start", "end")
	})
}

func TestDragOfAnotherButtonDoesNotTriggerPrimary(t *testing.T) {
	g := newGestureTester(t)
	var l log
	drag := NewHorizontalDragGestureRecognizer(g.binding)
	drag.OnDown = func(DragDownDetails) { l.add("primaryDown") }
	drag.OnStart = func(DragStartDetails) { l.add("primaryStart") }
	drag.OnUpdate = func(DragUpdateDetails) { l.add("primaryUpdate") }
	drag.OnEnd = func(DragEndDetails) { l.add("primaryEnd") }
	down := NewPointerDownEvent(PointerEventOpts{Pointer: 2, Position: off(30, 30), Buttons: foundation.Some(SecondaryButton)})
	drag.AddPointer(down)
	run(g, &l, closeArena(2), route(down),
		route(NewPointerMoveEvent(PointerEventOpts{Pointer: 2, Position: off(100, 200), Delta: off(70, 170), Buttons: foundation.Some(SecondaryButton)})),
		route(evUp(2, off(100, 201))), want())
	drag.Dispose()
}
