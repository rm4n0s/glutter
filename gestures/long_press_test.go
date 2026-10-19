package gestures

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func newLoggedLongPress(g *gestureTester, l *log, opts ...LongPressGestureRecognizerOpts) (*LongPressGestureRecognizer, *ui.Offset) {
	var dragUpdate ui.Offset
	gesture := NewLongPressGestureRecognizer(g.binding, opts...)
	gesture.OnLongPressDown = func(LongPressDownDetails) { l.add("down") }
	gesture.OnLongPressCancel = func() { l.add("cancel") }
	gesture.OnLongPress = func() { l.add("start") }
	gesture.OnLongPressMoveUpdate = func(d LongPressMoveUpdateDetails) {
		l.add("move")
		dragUpdate = d.GlobalPosition
	}
	gesture.OnLongPressUp = func() { l.add("end") }
	return gesture, &dragUpdate
}

type longPressEvents struct {
	down, down2 *PointerDownEvent
	up, up2     *PointerUpEvent
	move        *PointerMoveEvent
}

func newLongPressEvents() longPressEvents {
	return longPressEvents{
		down: evDown(5, off(10, 10)), up: evUp(5, off(11, 9)), move: evMove(5, off(100, 200)),
		down2: evDown(6, off(10, 10)), up2: evUp(6, off(11, 9)),
	}
}

func TestLongPress(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration foundation.Opt[time.Duration]
		noAdd    bool // the steps add the pointer themselves
		steps    func(e longPressEvents, gesture *LongPressGestureRecognizer) []step
	}{
		{"should recognize long press", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), want(), route(e.down), want("down"), elapse(ms(300)), want("down"),
				elapse(ms(700)), want("down", "start"), do(gesture.Dispose), want("down", "start")}
		}},
		{"with altered duration", foundation.Some(ms(100)), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), want(), route(e.down), want("down"), elapse(ms(50)), want("down"),
				elapse(ms(50)), want("down", "start"), do(gesture.Dispose), want("down", "start")}
		}},
		{"up cancels long press", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), route(e.down), want("down"), elapse(ms(300)), want("down"),
				route(e.up), want("down", "cancel"), elapse(time.Second), do(gesture.Dispose), want("down", "cancel")}
		}},
		{"moving before accept cancels", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), route(e.down), want("down"), elapse(ms(300)), want("down"),
				route(e.move), want("down", "cancel"), elapse(time.Second), route(e.up), elapse(ms(300)),
				want("down", "cancel"), do(gesture.Dispose), want("down", "cancel")}
		}},
		{"moving after accept is ok", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), route(e.down), want("down"), elapse(time.Second), want("down", "start"),
				route(e.move), want("down", "start", "move"), route(e.up), want("down", "start", "move", "end"),
				elapse(ms(300)), want("down", "start", "move", "end"), do(gesture.Dispose), want("down", "start", "move", "end")}
		}},
		{"long press up", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), route(e.down), want("down"), elapse(ms(300)), want("down"),
				elapse(ms(700)), want("down", "start"), route(e.up), want("down", "start", "end"),
				do(gesture.Dispose), want("down", "start", "end")}
		}},
		{"not with more than one button", foundation.None[time.Duration](), true, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{
				do(func() {
					gesture.AddPointer(NewPointerDownEvent(PointerEventOpts{Pointer: 5, Kind: ui.PointerDeviceKindMouse,
						Buttons: foundation.Some(SecondaryMouseButton | TertiaryButton), Position: off(10, 10)}))
				}),
				closeArena(5), want(), route(e.down), want(), elapse(time.Second), want(), route(e.up), want(),
				do(gesture.Dispose), want()}
		}},
		{"cancel when buttons change before acceptance", foundation.None[time.Duration](), false, func(e longPressEvents, gesture *LongPressGestureRecognizer) []step {
			return []step{closeArena(5), route(e.down), want("down"), elapse(ms(300)), want("down"),
				route(NewPointerMoveEvent(PointerEventOpts{Pointer: 5, Kind: ui.PointerDeviceKindMouse,
					Buttons: foundation.Some(TertiaryButton), Position: off(10, 10)})),
				want("down", "cancel"), elapse(ms(700)), want("down", "cancel"), route(e.up), want("down", "cancel"),
				do(gesture.Dispose), want("down", "cancel")}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			var opts []LongPressGestureRecognizerOpts
			if d, ok := tc.duration.Get(); ok {
				opts = append(opts, LongPressGestureRecognizerOpts{Duration: foundation.Some(d)})
			}
			gesture, _ := newLoggedLongPress(g, &l, opts...)
			e := newLongPressEvents()
			if !tc.noAdd {
				gesture.AddPointer(e.down)
			}
			run(g, &l, tc.steps(e, gesture)...)
		})
	}
}

func TestLongPressWithTapAndDrag(t *testing.T) {
	t.Run("both tap down and long press", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		e := newLongPressEvents()
		gesture, _ := newLoggedLongPress(g, &l)
		tap := NewTapGestureRecognizer(g.binding)
		tap.OnTapDown = func(TapDownDetails) { l.add("tap_down") }
		tap.AddPointer(e.down)
		gesture.AddPointer(e.down)
		run(g, &l, closeArena(5), want(), route(e.down), want("down"), elapse(ms(300)), want("down", "tap_down"),
			elapse(ms(700)), want("down", "tap_down", "start"), do(tap.Dispose), do(gesture.Dispose),
			want("down", "tap_down", "start"))
	})
	t.Run("drag start delayed by microtask", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		e := newLongPressEvents()
		gesture, _ := newLoggedLongPress(g, &l)
		drag := NewHorizontalDragGestureRecognizer(g.binding)
		dangerousStack := false
		drag.OnStart = func(DragStartDetails) {
			if dangerousStack {
				t.Error("the drag started inside Dispose")
			}
			l.add("drag_start")
		}
		drag.AddPointer(e.down)
		gesture.AddPointer(e.down)
		run(g, &l, closeArena(5), want(), route(e.down), want("down"), elapse(ms(300)), want("down"),
			do(func() { dangerousStack = true; gesture.Dispose(); dangerousStack = false }),
			want("down", "cancel"), flushMicrotasks(), want("down", "cancel", "drag_start"), do(drag.Dispose))
	})
}

func TestLongPressDragUpdatePosition(t *testing.T) {
	g := newGestureTester(t)
	var l log
	e := newLongPressEvents()
	gesture, dragUpdate := newLoggedLongPress(g, &l)
	gesture.AddPointer(e.down)
	run(g, &l, closeArena(5), route(e.down), elapse(time.Second), route(e.move), want("down", "start", "move"))
	if *dragUpdate != off(100, 200) {
		t.Errorf("drag update = %v", *dragUpdate)
	}
	run(g, &l, route(e.up), elapse(ms(300)), want("down", "start", "move", "end"), do(gesture.Dispose))
}

func TestLongPressNonAllowedPointerDoesNotReset(t *testing.T) {
	g := newGestureTester(t)
	var l log
	e := newLongPressEvents()
	gesture, _ := newLoggedLongPress(g, &l, LongPressGestureRecognizerOpts{SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch}})
	gesture.AddPointer(e.down)
	run(g, &l, closeArena(5), route(e.down), elapse(ms(500)), want("down", "start"),
		do(func() {
			gesture.AddPointer(NewPointerDownEvent(PointerEventOpts{Pointer: 101, Kind: ui.PointerDeviceKindMouse, Position: off(10, 10)}))
		}),
		want("down", "start"), route(evMove(5, off(15, 15))), want("down", "start", "move"), do(gesture.Dispose))
}

func TestLongPressConsistentButtons(t *testing.T) {
	moveR := func() *PointerMoveEvent {
		return NewPointerMoveEvent(PointerEventOpts{Pointer: 5, Buttons: foundation.Some(SecondaryButton), Position: off(10, 10)})
	}
	newGesture := func(g *gestureTester, l *log) *LongPressGestureRecognizer {
		gesture := NewLongPressGestureRecognizer(g.binding)
		gesture.OnLongPressDown = func(LongPressDownDetails) { l.add("down") }
		gesture.OnLongPressCancel = func() { l.add("cancel") }
		gesture.OnLongPressStart = func(LongPressStartDetails) { l.add("start") }
		gesture.OnLongPressEnd = func(LongPressEndDetails) { l.add("end") }
		return gesture
	}
	for _, tc := range []struct {
		name      string
		first     func(e longPressEvents) []step
		nextPress bool // a second long press follows and must work
	}{
		{"cancel when buttons change before acceptance", func(e longPressEvents) []step {
			return []step{route(e.down), elapse(ms(300)), route(moveR()), want("down", "cancel"), elapse(ms(700)), route(e.up), want("down", "cancel")}
		}, false},
		{"buttons change before acceptance does not prevent the next long press", func(e longPressEvents) []step {
			return []step{route(e.down), want("down"), elapse(ms(300)), route(moveR()), want("down", "cancel"), elapse(ms(700)), route(e.up)}
		}, true},
		{"not cancel when buttons change after acceptance", func(e longPressEvents) []step {
			return []step{route(e.down), elapse(time.Second), wantClear("down", "start"), route(moveR()), want(), route(e.up), want("end")}
		}, false},
		{"buttons change after acceptance does not prevent the next long press", func(e longPressEvents) []step {
			return []step{route(e.down), elapse(time.Second), route(moveR()), route(e.up)}
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			e := newLongPressEvents()
			gesture := newGesture(g, &l)
			gesture.AddPointer(e.down)
			closeArena(5)(g, &l)
			run(g, &l, tc.first(e)...)
			if tc.nextPress {
				l.clear()
				gesture.AddPointer(e.down2)
				run(g, &l, closeArena(6), route(e.down2), want("down"), elapse(time.Second), wantClear("down", "start"),
					route(e.up2), want("end"))
			}
			gesture.Dispose()
		})
	}
}

func TestLongPressFiltersByDeviceKind(t *testing.T) {
	g := newGestureTester(t)
	mouse := NewLongPressGestureRecognizer(g.binding, LongPressGestureRecognizerOpts{SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindMouse}})
	down := false
	mouse.OnLongPress = func() { down = true }
	mouseDown := NewPointerDownEvent(PointerEventOpts{Pointer: 5, Position: off(10, 10), Kind: ui.PointerDeviceKindMouse})
	touchDown := NewPointerDownEvent(PointerEventOpts{Pointer: 5, Position: off(10, 10)})
	mouse.AddPointer(touchDown)
	g.closeArena(5)
	g.route(touchDown)
	g.elapse(2 * time.Second)
	if down {
		t.Fatal("touch long pressed")
	}
	mouse.AddPointer(mouseDown)
	g.closeArena(5)
	g.route(mouseDown)
	if down {
		t.Fatal("mouse long pressed at once")
	}
	g.elapse(2 * time.Second)
	if !down {
		t.Error("mouse did not long press")
	}
	mouse.Dispose()
}

func TestLongPressDoesNotCompeteAcrossButtons(t *testing.T) {
	setup := func(g *gestureTester, l *log) (tapPrimary, tapSecondary *TapGestureRecognizer, longPress *LongPressGestureRecognizer) {
		tapPrimary = NewTapGestureRecognizer(g.binding)
		tapPrimary.OnTapDown = func(TapDownDetails) { l.add("tapPrimary") }
		tapSecondary = NewTapGestureRecognizer(g.binding)
		tapSecondary.OnSecondaryTapDown = func(TapDownDetails) { l.add("tapSecondary") }
		longPress = NewLongPressGestureRecognizer(g.binding)
		longPress.OnLongPressStart = func(LongPressStartDetails) { l.add("longPress") }
		return
	}
	t.Run("a primary long press does not compete with a secondary tap", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		_, tapSecondary, longPress := setup(g, &l)
		down3 := NewPointerDownEvent(PointerEventOpts{Pointer: 7, Position: off(30, 30), Buttons: foundation.Some(SecondaryButton)})
		longPress.AddPointer(down3)
		tapSecondary.AddPointer(down3)
		run(g, &l, closeArena(7), route(down3), want("tapSecondary"))
	})
	t.Run("a primary long press competes with a primary tap", func(t *testing.T) {
		g := newGestureTester(t)
		var l log
		e := newLongPressEvents()
		tapPrimary, _, longPress := setup(g, &l)
		longPress.AddPointer(e.down)
		tapPrimary.AddPointer(e.down)
		run(g, &l, closeArena(5), route(e.down), want(), route(e.up), want("tapPrimary"))
	})
}

func TestLongPressOfAnotherButtonDoesNotTrigger(t *testing.T) {
	for _, tc := range []struct {
		name    string
		buttons int
		primary bool
	}{
		{"a secondary long press should not trigger primary", SecondaryButton, false},
		{"a tertiary long press should not trigger primary or secondary", TertiaryButton, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			longPress := NewLongPressGestureRecognizer(g.binding)
			longPress.OnLongPressStart = func(LongPressStartDetails) { l.add("primaryStart") }
			longPress.OnLongPress = func() { l.add("primary") }
			longPress.OnLongPressMoveUpdate = func(LongPressMoveUpdateDetails) { l.add("primaryUpdate") }
			longPress.OnLongPressEnd = func(LongPressEndDetails) { l.add("primaryEnd") }
			longPress.OnLongPressUp = func() { l.add("primaryUp") }
			if tc.primary {
				longPress.OnSecondaryLongPressStart = func(LongPressStartDetails) { l.add("secondaryStart") }
				longPress.OnSecondaryLongPress = func() { l.add("secondary") }
				longPress.OnSecondaryLongPressMoveUpdate = func(LongPressMoveUpdateDetails) { l.add("secondaryUpdate") }
				longPress.OnSecondaryLongPressEnd = func(LongPressEndDetails) { l.add("secondaryEnd") }
				longPress.OnSecondaryLongPressUp = func() { l.add("secondaryUp") }
			}
			down := NewPointerDownEvent(PointerEventOpts{Pointer: 2, Buttons: foundation.Some(tc.buttons), Position: off(30, 30)})
			move := NewPointerMoveEvent(PointerEventOpts{Pointer: 2, Buttons: foundation.Some(tc.buttons), Position: off(100, 200)})
			longPress.AddPointer(down)
			run(g, &l, closeArena(2), route(down), elapse(ms(700)), route(move), route(evUp(2, off(100, 201))), want())
			longPress.Dispose()
		})
	}
}

func TestLongPressSwitchingButtonsStillSendsEnd(t *testing.T) {
	buttons := func(pointer int, b int) *PointerMoveEvent {
		return NewPointerMoveEvent(PointerEventOpts{Pointer: pointer, Position: off(100, 200), Buttons: foundation.Some(b)})
	}
	mark := func(l *log, s string) step { return func(*gestureTester, *log) { l.add(s) } }
	for _, tc := range []struct {
		name  string
		build func(l *log) (pointer int, steps []step)
		want  []string
	}{
		{"mid-stream", func(l *log) (int, []step) {
			up := NewPointerUpEvent(PointerEventOpts{Pointer: 8, Position: off(100, 200), Buttons: foundation.Some(SecondaryButton)})
			return 8, []step{closeArena(4), route(evDown(8, off(10, 10))), elapse(time.Second), mark(l, "two seconds later..."),
				route(buttons(8, PrimaryButton|SecondaryButton)), elapse(time.Second), mark(l, "two more seconds later..."),
				route(up), elapse(time.Second)}
		}, []string{"primaryStart", "two seconds later...", "two more seconds later...", "primaryEnd"}},
		{"alternative sequence", func(l *log) (int, []step) {
			return 9, []step{closeArena(4), route(evDown(9, off(10, 10))), elapse(time.Second), mark(l, "two seconds later..."),
				route(buttons(9, 3)), elapse(time.Second), mark(l, "two more seconds later..."),
				route(buttons(9, 2)), elapse(time.Second), mark(l, "two more seconds later still..."),
				route(evUp(9, off(100, 200))), elapse(time.Second)}
		}, []string{"primaryStart", "two seconds later...", "two more seconds later...", "two more seconds later still...", "primaryEnd"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			longPress := NewLongPressGestureRecognizer(g.binding)
			longPress.OnLongPressStart = func(LongPressStartDetails) { l.add("primaryStart") }
			longPress.OnLongPressEnd = func(LongPressEndDetails) { l.add("primaryEnd") }
			pointer, steps := tc.build(&l)
			longPress.AddPointer(evDown(pointer, off(10, 10)))
			run(g, &l, steps...)
			l.check(t, tc.want...)
			longPress.Dispose()
		})
	}
}
