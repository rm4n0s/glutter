package gestures

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

var consecutiveTapDelay = DoubleTapTimeout / 2

type tapAndDragFixture struct {
	g      *gestureTester
	events log
	r      *BaseTapAndDragGestureRecognizer
	e      map[string]PointerEvent
}

func newTapAndDragFixture(t *testing.T) *tapAndDragFixture {
	f := &tapAndDragFixture{g: newGestureTester(t)}
	mouse := func(p int, e PointerEventOpts) PointerEventOpts {
		e.Pointer, e.Kind = p, ui.PointerDeviceKindMouse
		return e
	}
	f.e = map[string]PointerEvent{
		"down1": evDown(1, off(10, 10)), "up1": evUp(1, off(11, 9)), "cancel1": NewPointerCancelEvent(PointerEventOpts{Pointer: 1}),
		"down2": evDown(2, off(12, 12)), "up2": evUp(2, off(13, 11)),
		"down3": evDown(3, off(12, 12)), "up3": evUp(3, off(13, 11)),
		"down4": evDown(4, off(130, 130)), "up4": evUp(4, off(131, 129)),
		"down5": evDown(5, off(10, 10)), "move5": evMove(5, off(25, 25)), "up5": evUp(5, off(25, 25)),
		"down6": NewPointerDownEvent(mouse(6, PointerEventOpts{Position: off(10, 10)})),
		"move6": NewPointerMoveEvent(mouse(6, PointerEventOpts{Position: off(15, 15), Delta: off(5, 5)})),
		"up6":   NewPointerUpEvent(mouse(6, PointerEventOpts{Position: off(15, 15)})),
	}
	return f
}

func (f *tapAndDragFixture) setUp(horizontal bool, eager bool) {
	var base *BaseTapAndDragGestureRecognizer
	prefix := "pan"
	opts := TapAndDragGestureRecognizerOpts{NoEagerVictoryOnDrag: !eager}
	if horizontal {
		r := NewTapAndHorizontalDragGestureRecognizer(f.g.binding, opts)
		base = &r.BaseTapAndDragGestureRecognizer
		prefix = "horizontaldrag"
	} else {
		r := NewTapAndPanGestureRecognizer(f.g.binding, opts)
		base = &r.BaseTapAndDragGestureRecognizer
	}
	base.DragStartBehavior = DragStartBehaviorDown
	base.MaxConsecutiveTap = foundation.Some(3)
	base.OnTapDown = func(d TapDragDownDetails) { f.events.add(fmt.Sprintf("down#%d", d.ConsecutiveTapCount)) }
	base.OnTapUp = func(d TapDragUpDetails) { f.events.add(fmt.Sprintf("up#%d", d.ConsecutiveTapCount)) }
	base.OnDragStart = func(d TapDragStartDetails) { f.events.add(fmt.Sprintf("%sstart#%d", prefix, d.ConsecutiveTapCount)) }
	base.OnDragUpdate = func(d TapDragUpdateDetails) { f.events.add(fmt.Sprintf("%supdate#%d", prefix, d.ConsecutiveTapCount)) }
	base.OnDragEnd = func(d TapDragEndDetails) { f.events.add(fmt.Sprintf("%send#%d", prefix, d.ConsecutiveTapCount)) }
	base.OnCancel = func() { f.events.add("cancel") }
	f.r = base
}

func (f *tapAndDragFixture) down(name string) {
	f.r.AddPointer(f.e["down"+name].(*PointerDownEvent))
}

// tap is the sequence of a tap: add the pointer, close the arena, route down and up, sweep.
func (f *tapAndDragFixture) tap(n int) {
	name := fmt.Sprint(n)
	f.down(name)
	f.g.closeArena(n)
	f.g.route(f.e["down"+name])
	f.g.route(f.e["up"+name])
	f.g.binding.GestureArena.Sweep(n)
}

func TestTapAndDragConsecutiveTaps(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	f.tap(1)
	f.events.check(t, "down#1", "up#1")
	f.events.clear()
	f.g.elapse(consecutiveTapDelay)
	f.tap(2)
	f.events.check(t, "down#2", "up#2")
	f.events.clear()
	f.g.elapse(consecutiveTapDelay)
	f.tap(3)
	f.events.check(t, "down#3", "up#3")
}

func TestTapAndDragResets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps func(f *tapAndDragFixture)
	}{
		{"times out in between taps", func(f *tapAndDragFixture) { f.g.elapse(ms(1000)); f.tap(2) }},
		{"taps are far apart", func(f *tapAndDragFixture) { f.g.elapse(ms(100)); f.tap(4) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTapAndDragFixture(t)
			f.setUp(false, true)
			f.tap(1)
			f.events.check(t, "down#1", "up#1")
			f.events.clear()
			tc.steps(f)
			f.events.check(t, "down#1", "up#1")
		})
	}
	t.Run("consecutiveTapCount reaches maxConsecutiveTap", func(t *testing.T) {
		f := newTapAndDragFixture(t)
		f.setUp(false, true)
		for i, n := range []int{1, 2, 3} {
			f.tap(n)
			f.events.check(t, fmt.Sprintf("down#%d", i+1), fmt.Sprintf("up#%d", i+1))
			f.events.clear()
		}
		f.tap(3)
		f.events.check(t, "down#1", "up#1")
	})
}

func TestTapAndDragRecognizesDrag(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	pointer := newTestPointer(5)
	down := pointer.down(off(10, 10))
	f.r.AddPointer(down)
	f.g.closeArena(5)
	f.g.route(down)
	f.g.route(pointer.move(off(40, 45)))
	f.g.route(pointer.up())
	f.g.binding.GestureArena.Sweep(5)
	f.events.check(t, "down#1", "panstart#1", "panupdate#1", "panend#1")
}

func TestTapAndDragConsecutiveTapsThenDrag(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	pointer := newTestPointer(5)
	for i := 0; i < 2; i++ {
		down := pointer.down(off(10, 10))
		f.r.AddPointer(down)
		f.g.closeArena(5)
		f.g.route(down)
		f.g.route(pointer.up())
		f.g.binding.GestureArena.Sweep(5)
		f.g.elapse(consecutiveTapDelay)
	}
	down := pointer.down(off(10, 10))
	f.r.AddPointer(down)
	f.g.closeArena(5)
	f.g.route(down)
	f.g.route(pointer.move(off(40, 45)))
	f.g.route(pointer.up())
	f.events.check(t, "down#1", "up#1", "down#2", "up#2", "down#3", "panstart#3", "panupdate#3", "panend#3")
}

func TestTapAndDragRejectsNonPrimaryPointers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps func(f *tapAndDragFixture)
	}{
		{"FIFO before acceptance", func(f *tapAndDragFixture) {
			f.down("1")
			f.down("2")
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.g.closeArena(2)
			f.g.route(f.e["down2"])
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
			f.g.route(f.e["up2"])
			f.g.binding.GestureArena.Sweep(2)
		}},
		{"FILO before acceptance", func(f *tapAndDragFixture) {
			f.down("1")
			f.down("2")
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.g.closeArena(2)
			f.g.route(f.e["down2"])
			f.g.route(f.e["up2"])
			f.g.binding.GestureArena.Sweep(2)
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
		}},
		{"FIFO after acceptance", func(f *tapAndDragFixture) {
			f.down("1")
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.down("2")
			f.g.closeArena(2)
			f.g.route(f.e["down2"])
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
			f.g.route(f.e["up2"])
			f.g.binding.GestureArena.Sweep(2)
		}},
		{"FILO after acceptance", func(f *tapAndDragFixture) {
			f.down("1")
			f.g.closeArena(1)
			f.g.route(f.e["down1"])
			f.down("2")
			f.g.closeArena(2)
			f.g.route(f.e["down2"])
			f.g.route(f.e["up2"])
			f.g.binding.GestureArena.Sweep(2)
			f.g.route(f.e["up1"])
			f.g.binding.GestureArena.Sweep(1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTapAndDragFixture(t)
			f.setUp(false, true)
			tc.steps(f)
			f.events.check(t, "down#1", "up#1")
		})
	}
}

func TestTapAndDragTapUpWhenAcceptingBeforeHandleEvent(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	f.down("1")
	f.g.closeArena(1)
	f.g.binding.GestureArena.Sweep(1)
	f.g.route(f.e["down1"])
	f.g.route(f.e["up1"])
	f.events.check(t, "down#1", "up#1")
}

func TestTapAndDragPastTapToleranceButNotDragMinimum(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	f.down("5")
	f.g.closeArena(5)
	f.g.route(f.e["down5"])
	f.g.route(f.e["move5"])
	f.g.route(f.e["up5"])
	f.g.binding.GestureArena.Sweep(5)
	f.events.check(t, "down#1", "panstart#1", "panend#1")
	// After the drag, the next tap starts a new series.
	f.events.clear()
	f.g.elapse(ms(1000))
	f.tap(1)
	f.events.check(t, "down#1", "up#1")
}

func TestTapAndDragBeatsTapWithMouse(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	taps := NewTapGestureRecognizer(f.g.binding)
	taps.OnTapDown = func(TapDownDetails) { f.events.add("tapdown") }
	taps.OnTapUp = func(TapUpDetails) { f.events.add("tapup") }
	taps.OnTapCancel = func() { f.events.add("tapscancel") }
	f.down("6")
	taps.AddPointer(f.e["down6"].(*PointerDownEvent))
	f.g.closeArena(6)
	f.g.route(f.e["down6"])
	f.g.route(f.e["move6"])
	f.g.route(f.e["up6"])
	f.g.binding.GestureArena.Sweep(6)
	f.events.check(t, "down#1", "panstart#1", "panupdate#1", "panend#1")
}

func TestTapAndDragFiresCancelForPointerCancel(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(false, true)
	f.down("1")
	f.g.closeArena(1)
	f.g.route(f.e["down1"])
	f.g.route(f.e["cancel1"])
	f.g.binding.GestureArena.Sweep(1)
	f.events.check(t, "down#1", "cancel")
	f.events.clear()
	f.g.elapse(ms(100))
	f.tap(2)
	f.events.check(t, "down#1", "up#1")
}

func TestTapAndDragWithoutDragEndCallback(t *testing.T) {
	f := newTapAndDragFixture(t)
	r := NewTapAndPanGestureRecognizer(f.g.binding)
	r.DragStartBehavior = DragStartBehaviorDown
	r.MaxConsecutiveTap = foundation.Some(3)
	r.OnTapDown = func(d TapDragDownDetails) { f.events.add(fmt.Sprintf("down#%d", d.ConsecutiveTapCount)) }
	f.r = &r.BaseTapAndDragGestureRecognizer
	f.down("5")
	f.g.closeArena(5)
	f.g.route(f.e["down5"])
	f.g.route(f.e["move5"])
	f.g.route(f.e["up5"])
	f.g.binding.GestureArena.Sweep(5)
	f.events.check(t, "down#1")
	f.events.clear()
	f.g.elapse(ms(1000))
	f.tap(1)
	f.events.check(t, "down#1")
}

func TestTapAndDragEndDetailsPositions(t *testing.T) {
	g := newGestureTester(t)
	r := NewTapAndHorizontalDragGestureRecognizer(g.binding)
	r.DragStartBehavior = DragStartBehaviorDown
	r.MaxConsecutiveTap = foundation.Some(3)
	var end TapDragEndDetails
	r.OnDragEnd = func(d TapDragEndDetails) { end = d }
	pointer := newTestPointer(5)
	down := pointer.down(off(10, 10))
	r.AddPointer(down)
	g.closeArena(5)
	g.route(down)
	g.route(pointer.move(off(50, 20)))
	g.route(pointer.move(off(90, 30)))
	g.route(pointer.move(off(120, 45)))
	g.route(pointer.up())
	if end.GlobalPosition != off(120, 45) {
		t.Errorf("global position = %v, want (120, 45)", end.GlobalPosition)
	}
}

func TestTapAndDragStateResetsAfterLosingArena(t *testing.T) {
	f := newTapAndDragFixture(t)
	f.setUp(true, false)
	horizontal := NewHorizontalDragGestureRecognizer(f.g.binding)
	horizontal.OnStart = func(DragStartDetails) { f.events.add("basichorizontalstart") }
	horizontal.OnUpdate = func(DragUpdateDetails) { f.events.add("basichorizontalupdate") }
	horizontal.OnEnd = func(DragEndDetails) { f.events.add("basichorizontalend") }
	horizontal.OnCancel = func() { f.events.add("basichorizontalcancel") }
	longPress := NewLongPressGestureRecognizer(f.g.binding)
	longPress.OnLongPressStart = func(LongPressStartDetails) { f.events.add("longpressstart") }
	longPress.OnLongPressMoveUpdate = func(LongPressMoveUpdateDetails) { f.events.add("longpressmoveupdate") }
	longPress.OnLongPressEnd = func(LongPressEndDetails) { f.events.add("longpressend") }
	longPress.OnLongPressCancel = func() { f.events.add("longpresscancel") }

	pointer := newTestPointer(5)
	downB := pointer.down(off(10, 10))
	f.r.AddPointer(downB)
	horizontal.AddPointer(downB)
	longPress.AddPointer(downB)
	f.g.closeArena(5)
	f.g.route(downB)
	f.g.route(pointer.move(off(28.1, 10)))
	f.g.route(pointer.up())
	f.events.check(t, "basichorizontalstart", "basichorizontalend")

	downC := pointer.down(off(10, 10))
	f.r.AddPointer(downC)
	horizontal.AddPointer(downC)
	longPress.AddPointer(downC)
	f.g.closeArena(5)
	f.g.route(downC)
	f.g.route(pointer.up()) // must not panic
}
