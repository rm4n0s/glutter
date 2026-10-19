package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

type mockHitTestTarget struct{}

func (mockHitTestTarget) HandleEvent(PointerEvent, HitTestEntry) {}

func TestDragUpEventForPendingPointerAfterWinningForAnother(t *testing.T) {
	g := newGestureTester(t)
	v := NewVerticalDragGestureRecognizer(g.binding)
	v.OnStart = func(DragStartDetails) {}
	h := NewHorizontalDragGestureRecognizer(g.binding)
	h.OnStart = func(DragStartDetails) {}
	down90, up90 := evDown(90, off(10, 10)), evUp(90, off(10, 10))
	down91, up91 := evDown(91, off(20, 20)), evUp(91, off(20, 20))
	v.AddPointer(down90)
	g.binding.GestureArena.Close(90)
	h.AddPointer(down91)
	v.AddPointer(down91)
	g.binding.GestureArena.Close(91)
	g.flush()
	// must not panic
	g.binding.HandleEvent(up90, NewHitTestEntry(mockHitTestTarget{}))
	g.binding.HandleEvent(up91, NewHitTestEntry(mockHitTestTarget{}))
}

func TestDragOnlyAcceptDragOnThreshold(t *testing.T) {
	for _, tc := range []struct {
		name      string
		threshold bool
		want      []string
	}{
		{"does not dispatch when the threshold has not been met", true, nil},
		{"dispatches when the threshold is not required", false, []string{"onStart", "onEnd"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			drag := NewVerticalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{OnlyAcceptDragOnThreshold: tc.threshold})
			drag.OnStart = func(DragStartDetails) { l.add("onStart") }
			drag.OnUpdate = func(DragUpdateDetails) { l.add("onUpdate") }
			drag.OnEnd = func(DragEndDetails) { l.add("onEnd") }
			down, up := evDown(6, off(10, 10)), evUp(6, off(10, 10))
			drag.AddPointer(down)
			g.closeArena(6)
			g.route(down)
			g.route(up)
			l.check(t, tc.want...)
			drag.Dispose()
		})
	}
}

func TestDragRecognizersOnDifferentButtonFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		buttons int
		want    string
	}{
		{"primary button works", PrimaryButton, "onStartPrimary"},
		{"secondary button works", SecondaryMouseButton, "onStartSecondary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			var l log
			primary := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{AllowedButtonsFilter: func(b int) bool { return b == PrimaryButton }})
			primary.OnStart = func(DragStartDetails) { l.add("onStartPrimary") }
			secondary := NewHorizontalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{AllowedButtonsFilter: func(b int) bool { return b == SecondaryButton }})
			secondary.OnStart = func(DragStartDetails) { l.add("onStartSecondary") }
			down := NewPointerDownEvent(PointerEventOpts{Pointer: 6, Position: off(10, 10), Buttons: foundation.Some(tc.buttons)})
			primary.AddPointer(down)
			secondary.AddPointer(down)
			g.closeArena(6)
			g.route(down)
			l.check(t, tc.want)
			primary.Dispose()
			secondary.Dispose()
		})
	}
}

func TestPanStartsAfterPanSlop(t *testing.T) {
	g := newGestureTester(t)
	started := false
	pan := NewPanGestureRecognizer(g.binding)
	pan.OnStart = func(DragStartDetails) { started = true }
	pointer := newTestPointer(1)
	down := pointer.down(off(50, 50))
	pan.AddPointer(down)
	newIndefinite(g.binding).AddPointer(down) // keeps the arena open
	g.closeArena(1)
	g.route(down)
	g.route(pointer.move(off(50, 50+PanSlop/2)))
	if started {
		t.Fatal("started before the pan slop")
	}
	g.route(pointer.move(off(50, 50+PanSlop+1)))
	if !started {
		t.Error("pan start was not called")
	}
	pan.Dispose()
}

// pickyVertical is a vertical drag recognizer that takes the pointers of one
// button only, as a subclass in Dart that overrides isPointerAllowed.
type pickyVertical struct {
	VerticalDragGestureRecognizer
	asked int
}

func (p *pickyVertical) IsPointerAllowed(event *PointerDownEvent) bool {
	p.asked++
	return event.Pointer() == 1 && p.VerticalDragGestureRecognizer.IsPointerAllowed(event)
}

func TestInitVerticalDragGestureRecognizerCallsTheOverrides(t *testing.T) {
	g := newGestureTester(t)
	p := &pickyVertical{}
	InitVerticalDragGestureRecognizer(&p.VerticalDragGestureRecognizer, p, g.binding)
	var l log
	p.OnStart = func(DragStartDetails) { l.add("start") }
	p.OnEnd = func(DragEndDetails) { l.add("end") }
	for _, pointer := range []int{1, 2} {
		down, up := evDown(pointer, off(10, 10)), evUp(pointer, off(10, 10))
		p.AddPointer(down)
		g.closeArena(pointer)
		g.route(down)
		g.route(up)
	}
	if p.asked != 2 {
		t.Errorf("the override was asked %d times, want 2", p.asked)
	}
	l.check(t, "start", "end") // the second pointer is not allowed
	p.Dispose()
}

type pickyHorizontal struct {
	HorizontalDragGestureRecognizer
	asked int
}

func (p *pickyHorizontal) IsPointerAllowed(event *PointerDownEvent) bool {
	p.asked++
	return p.HorizontalDragGestureRecognizer.IsPointerAllowed(event)
}

func TestInitHorizontalDragGestureRecognizerCallsTheOverrides(t *testing.T) {
	g := newGestureTester(t)
	p := &pickyHorizontal{}
	InitHorizontalDragGestureRecognizer(&p.HorizontalDragGestureRecognizer, p, g.binding)
	down := evDown(3, off(10, 10))
	p.AddPointer(down)
	if p.asked != 1 {
		t.Errorf("the override was asked %d times, want 1", p.asked)
	}
	g.closeArena(3)
	g.route(evUp(3, off(10, 10)))
	p.Dispose()
}
