package gestures

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// TestRecognizersForgetTheArenasOfPastPointers checks that a recognizer that lives as long as the
// app holds no arena entry for each pointer it ever tracked, and that the arena manager holds no
// arena of a finished pointer. Every press has a new pointer id (step 29.12), so an entry that is
// kept is an entry, an arena and its members for each press (step 29.21).
func TestRecognizersForgetTheArenasOfPastPointers(t *testing.T) {
	tests := []struct {
		name string
		make func(g *gestureTester) (addPointer func(*PointerDownEvent), entries func() int)
	}{
		{"tap", func(g *gestureTester) (func(*PointerDownEvent), func() int) {
			r := NewTapGestureRecognizer(g.binding)
			r.OnTap = func() {}
			return r.AddPointer, func() int { return len(r.entries) }
		}},
		{"long press", func(g *gestureTester) (func(*PointerDownEvent), func() int) {
			r := NewLongPressGestureRecognizer(g.binding)
			r.OnLongPress = func() {}
			return r.AddPointer, func() int { return len(r.entries) }
		}},
		{"horizontal drag", func(g *gestureTester) (func(*PointerDownEvent), func() int) {
			r := NewHorizontalDragGestureRecognizer(g.binding)
			r.OnUpdate = func(DragUpdateDetails) {}
			return r.AddPointer, func() int { return len(r.entries) }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGestureTester(t)
			add, entries := tt.make(g)
			for pointer := 1; pointer <= 50; pointer++ {
				p := newTestPointer(pointer)
				down := p.down(off(10, 10))
				add(down)
				g.closeArena(pointer)
				g.route(down)
				g.route(p.up())
				g.binding.GestureArena.Sweep(pointer)
				g.elapse(0)
			}
			if n := entries(); n > 1 {
				t.Errorf("the recognizer holds %d arena entries after 50 pointers, want at most 1", n)
			}
			if n := len(g.binding.GestureArena.arenas); n != 0 {
				t.Errorf("the arena manager holds %d arenas after 50 pointers, want 0", n)
			}
		})
	}
}

// TestRecognizersRememberTheLastPointersOnly checks the bound of the kind and buttons that a
// recognizer keeps for its pointers (step 29.21).
func TestRecognizersRememberTheLastPointersOnly(t *testing.T) {
	tests := []struct {
		name string
		make func(g *gestureTester) (r GestureRecognizer, press func(p *testPointer) *PointerDownEvent)
	}{
		{"tap", func(g *gestureTester) (GestureRecognizer, func(*testPointer) *PointerDownEvent) {
			r := NewTapGestureRecognizer(g.binding)
			r.OnTap = func() {}
			return r, func(p *testPointer) *PointerDownEvent { return p.down(off(10, 10)) }
		}},
		{"double tap", func(g *gestureTester) (GestureRecognizer, func(*testPointer) *PointerDownEvent) {
			r := NewDoubleTapGestureRecognizer(g.binding)
			r.OnDoubleTap = func() {}
			return r, func(p *testPointer) *PointerDownEvent { return p.down(off(10, 10)) }
		}},
		{"scale", func(g *gestureTester) (GestureRecognizer, func(*testPointer) *PointerDownEvent) {
			r := NewScaleGestureRecognizer(g.binding)
			r.OnUpdate = func(ScaleUpdateDetails) {}
			return r, func(p *testPointer) *PointerDownEvent { return p.down(off(10, 10)) }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGestureTester(t)
			r, press := tt.make(g)
			const pointers = 1000
			for pointer := 1; pointer <= pointers; pointer++ {
				p := newTestPointer(pointer)
				down := press(p)
				r.Base().AddPointer(down)
				g.closeArena(pointer)
				g.route(down)
				g.route(p.up())
				g.binding.GestureArena.Sweep(pointer)
				g.elapse(0)
			}
			if n := len(r.Base().pointerToEventData); n > maxRememberedPointers {
				t.Errorf("the recognizer remembers %d pointers, want at most %d", n, maxRememberedPointers)
			}
			if k := r.Base().GetKindForPointer(pointers); k != ui.PointerDeviceKindTouch {
				t.Errorf("the kind of the last pointer = %v, want touch", k)
			}
			if k := r.Base().GetKindForPointer(pointers - maxRememberedPointers + 1); k != ui.PointerDeviceKindTouch {
				t.Errorf("the kind of the oldest pointer that is kept = %v, want touch", k)
			}
		})
	}
}

// TestRecognizersKeepNoStateForFinishedPointers drags with many pointers, one after the other
// (every press has a new id, step 29.12), and checks that the recognizers that keep a velocity
// tracker, a tap tracker or a set of pointers for each pointer that is down keep none for the
// pointers that are up.
func TestRecognizersKeepNoStateForFinishedPointers(t *testing.T) {
	tests := []struct {
		name string
		make func(g *gestureTester) (r GestureRecognizer, state func() int)
	}{
		{"horizontal drag", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewHorizontalDragGestureRecognizer(g.binding)
			r.OnUpdate = func(DragUpdateDetails) {}
			r.OnEnd = func(DragEndDetails) {}
			return r, func() int { return len(r.velocityTrackers) + len(r.moveDeltaBeforeFrame) + r.TrackedPointerCount() }
		}},
		{"pan", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewPanGestureRecognizer(g.binding)
			r.OnUpdate = func(DragUpdateDetails) {}
			r.OnEnd = func(DragEndDetails) {}
			return r, func() int { return len(r.velocityTrackers) + len(r.moveDeltaBeforeFrame) + r.TrackedPointerCount() }
		}},
		{"scale", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewScaleGestureRecognizer(g.binding)
			r.OnUpdate = func(ScaleUpdateDetails) {}
			r.OnEnd = func(ScaleEndDetails) {}
			return r, func() int { return len(r.velocityTrackers) + len(r.pointerQueue) + r.TrackedPointerCount() }
		}},
		{"tap and pan", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewTapAndPanGestureRecognizer(g.binding)
			r.OnDragUpdate = func(TapDragUpdateDetails) {}
			r.OnTapDown = func(TapDragDownDetails) {}
			return r, func() int { return len(r.acceptedActivePointers) + r.TrackedPointerCount() }
		}},
		{"multi drag", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewImmediateMultiDragGestureRecognizer(g.binding)
			r.OnStart = func(ui.Offset) Drag { return nopDrag{} }
			return r, func() int { return len(r.pointers) + len(r.pointerRoutes) }
		}},
		{"double tap", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewDoubleTapGestureRecognizer(g.binding)
			r.OnDoubleTap = func() {}
			return r, func() int { return len(r.trackers) }
		}},
		{"multi tap", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewMultiTapGestureRecognizer(g.binding)
			r.OnTap = func(int) {}
			return r, func() int { return len(r.gestureMap) }
		}},
		{"serial tap", func(g *gestureTester) (GestureRecognizer, func() int) {
			r := NewSerialTapGestureRecognizer(g.binding)
			r.OnSerialTapDown = func(SerialTapDownDetails) {}
			return r, func() int { return len(r.gestureResolution) }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGestureTester(t)
			r, state := tt.make(g)
			for pointer := 1; pointer <= 60; pointer++ {
				p := newTestPointer(pointer)
				down := p.down(off(10, 10))
				r.Base().AddPointer(down)
				g.closeArena(pointer)
				g.route(down)
				g.route(p.move(off(60, 10)))
				g.route(p.move(off(110, 10)))
				g.route(p.up())
				g.binding.GestureArena.Sweep(pointer)
				g.elapse(time.Second)
			}
			if n := state(); n != 0 {
				t.Errorf("the recognizer keeps %d records for the pointers that are up, want 0", n)
			}
			if n := len(g.binding.GestureArena.arenas); n != 0 {
				t.Errorf("the arena manager holds %d arenas, want 0", n)
			}
		})
	}
}

// nopDrag is a Drag that does nothing.
type nopDrag struct{}

func (nopDrag) Update(DragUpdateDetails) {}
func (nopDrag) End(DragEndDetails)       {}
func (nopDrag) Cancel()                  {}
