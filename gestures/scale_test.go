package gestures

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestScaleRecognizes(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding)
	tap := NewTapGestureRecognizer(g.binding)
	var (
		didStart, didEnd, didTap bool
		focal, delta             foundation.Opt[ui.Offset]
		sc, hs, vs               foundation.Opt[float64]
		count                    foundation.Opt[int]
	)
	scale.OnStart = func(d ScaleStartDetails) {
		didStart = true
		focal, count = foundation.Some(d.FocalPoint), foundation.Some(d.PointerCount)
	}
	scale.OnUpdate = func(d ScaleUpdateDetails) {
		sc, hs, vs = foundation.Some(d.Scale), foundation.Some(d.HorizontalScale), foundation.Some(d.VerticalScale)
		focal, delta, count = foundation.Some(d.FocalPoint), foundation.Some(d.FocalPointDelta), foundation.Some(d.PointerCount)
	}
	scale.OnEnd = func(ScaleEndDetails) { didEnd = true }
	tap.OnTap = func() { didTap = true }
	reset := func() {
		focal, delta, sc, hs, vs, count = foundation.None[ui.Offset](), foundation.None[ui.Offset](),
			foundation.None[float64](), foundation.None[float64](), foundation.None[float64](), foundation.None[int]()
	}
	reset()
	expectNothing := func(where string) {
		t.Helper()
		if didStart || sc.IsSome() || focal.IsSome() || delta.IsSome() || count.IsSome() || didEnd || didTap {
			t.Fatalf("%s: start %v, scale %v, focal %v, delta %v, count %v, end %v, tap %v", where, didStart, sc, focal, delta, count, didEnd, didTap)
		}
	}
	checkDelta := func(want ui.Offset, eps float64) {
		t.Helper()
		if d := delta.Value(); math.Abs(d.Dx-want.Dx) > eps || math.Abs(d.Dy-want.Dy) > eps {
			t.Errorf("focal point delta = %v, want %v", d, want)
		}
	}
	checkFocal := func(want ui.Offset) {
		t.Helper()
		if focal != foundation.Some(want) {
			t.Errorf("focal point = %v, want %v", focal, want)
		}
	}
	checkScale := func(want float64) {
		t.Helper()
		if sc != foundation.Some(want) {
			t.Errorf("scale = %v, want %v", sc, want)
		}
	}
	checkCount := func(want int) {
		t.Helper()
		if count != foundation.Some(want) {
			t.Errorf("pointer count = %v, want %d", count, want)
		}
	}
	started := func() {
		t.Helper()
		if !didStart {
			t.Error("scale did not start")
		}
		didStart = false
	}
	ended := func() {
		t.Helper()
		if !didEnd {
			t.Error("scale did not end")
		}
		didEnd = false
	}

	pointer1 := newTestPointer()
	down := pointer1.down(ui.Offset{})
	scale.AddPointer(down)
	tap.AddPointer(down)
	g.closeArena(1)
	expectNothing("after closing the arena")
	g.route(down)
	expectNothing("after the down event")

	g.route(pointer1.move(off(20, 30)))
	started()
	checkFocal(off(20, 30))
	checkScale(1)
	checkDelta(off(20, 30), 0)
	checkCount(1)
	if didEnd || didTap {
		t.Fatal("ended or tapped")
	}
	reset()

	pointer2 := newTestPointer(2)
	down2 := pointer2.down(off(10, 20))
	scale.AddPointer(down2)
	tap.AddPointer(down2)
	g.closeArena(2)
	g.route(down2)
	ended()
	if sc.IsSome() || focal.IsSome() || delta.IsSome() || count.IsSome() || didStart {
		t.Fatal("update after the second down")
	}
	g.route(pointer2.move(off(0, 10)))
	started()
	checkFocal(off(10, 20))
	checkScale(2)
	if hs != foundation.Some(2.0) || vs != foundation.Some(2.0) {
		t.Errorf("horizontal %v, vertical %v", hs, vs)
	}
	checkDelta(off(-5, -5), 0)
	checkCount(2)
	g.route(pointer2.move(off(15, 25)))
	checkFocal(off(17.5, 27.5))
	checkScale(0.5)
	checkDelta(off(7.5, 7.5), 0)
	g.route(pointer2.move(off(0, 20)))
	if hs != foundation.Some(2.0) || vs != foundation.Some(1.0) {
		t.Errorf("horizontal %v, vertical %v", hs, vs)
	}
	g.route(pointer2.move(off(10, 10)))
	if hs != foundation.Some(1.0) || vs != foundation.Some(2.0) {
		t.Errorf("horizontal %v, vertical %v", hs, vs)
	}
	checkDelta(off(5, -5), 0)
	g.route(pointer2.move(off(15, 25)))
	reset()

	pointer3 := newTestPointer(3)
	down3 := pointer3.down(off(25, 35))
	scale.AddPointer(down3)
	tap.AddPointer(down3)
	g.closeArena(3)
	g.route(down3)
	ended()
	if sc.IsSome() || didStart {
		t.Fatal("update after the third down")
	}
	g.route(pointer3.move(off(55, 65)))
	started()
	checkFocal(off(30, 40))
	checkScale(5)
	checkDelta(off(10, 10), 0)
	checkCount(3)
	reset()

	g.route(pointer1.move(off(25, 35)))
	g.route(pointer2.move(off(20, 30)))
	g.route(pointer3.move(off(15, 25)))
	if didStart {
		t.Fatal("started again")
	}
	checkFocal(off(20, 30))
	checkScale(1)
	checkDelta(off(-13.3, -13.3), 0.1)
	checkCount(3)
	reset()

	g.route(pointer1.up())
	if didStart || focal.IsSome() || sc.IsSome() || delta.IsSome() {
		t.Fatal("update at the up event")
	}
	ended()
	g.route(pointer3.move(off(10, 20)))
	started()
	checkFocal(off(15, 25))
	checkScale(2)
	checkDelta(off(-2.5, -2.5), 0)
	checkCount(2)
	g.route(pointer3.move(off(30, 40)))
	checkFocal(off(25, 35))
	checkScale(2)
	checkDelta(off(10, 10), 0)
	g.route(pointer3.move(off(10, 20)))
	checkFocal(off(15, 25))
	checkDelta(off(-10, -10), 0)
	reset()

	g.route(pointer2.up())
	ended()
	g.route(pointer3.move(ui.Offset{}))
	started()
	checkFocal(ui.Offset{})
	checkScale(1)
	checkDelta(off(-10, -20), 0)
	checkCount(1)
	reset()
	g.route(pointer3.up())
	ended()
	if didTap {
		t.Error("tapped")
	}
	scale.Dispose()
	tap.Dispose()
}

func TestScaleRejectsUnallowedDeviceKinds(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding, ScaleGestureRecognizerOpts{SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindTouch}})
	started, updated := false, false
	scale.OnStart = func(ScaleStartDetails) { started = true }
	scale.OnUpdate = func(ScaleUpdateDetails) { updated = true }
	pointer := newTestPointer(1)
	pointer.kind = ui.PointerDeviceKindMouse
	down := pointer.down(ui.Offset{})
	scale.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	g.route(pointer.move(off(20, 30)))
	if started || updated {
		t.Error("scaled with a mouse")
	}
	scale.Dispose()
}

func TestScaleCompetesWithDrag(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding)
	drag := NewHorizontalDragGestureRecognizer(g.binding)
	var l log
	scale.OnStart = func(ScaleStartDetails) { l.add("scale-start") }
	scale.OnUpdate = func(ScaleUpdateDetails) { l.add("scale-update") }
	scale.OnEnd = func(ScaleEndDetails) { l.add("scale-end") }
	drag.OnStart = func(DragStartDetails) { l.add("drag-start") }
	drag.OnEnd = func(DragEndDetails) { l.add("drag-end") }

	pointer1 := newTestPointer()
	down := pointer1.down(off(10, 10))
	scale.AddPointer(down)
	drag.AddPointer(down)
	g.closeArena(1)
	l.check(t)
	g.route(down)
	l.check(t)
	g.route(pointer1.move(off(10, 50))) // 40 exceeds 18 * 2
	wantClear := func(w ...string) { t.Helper(); l.check(t, w...); l.clear() }
	wantClear("scale-start", "scale-update")

	pointer2 := newTestPointer(2)
	down2 := pointer2.down(off(10, 20))
	scale.AddPointer(down2)
	drag.AddPointer(down2)
	g.closeArena(2)
	l.check(t)
	g.route(down2)
	wantClear("scale-end")
	g.route(pointer2.move(off(30, 20)))
	wantClear("scale-start", "scale-update")
	g.route(pointer1.up())
	wantClear("scale-end")
	g.route(pointer2.up())
	wantClear()

	pointer3 := newTestPointer(3)
	down3 := pointer3.down(off(30, 30))
	scale.AddPointer(down3)
	drag.AddPointer(down3)
	g.closeArena(3)
	g.route(down3)
	l.check(t)
	g.route(pointer3.move(off(100, 30)))
	wantClear("scale-start", "scale-update")
	g.route(pointer3.up())
	wantClear("scale-end")
	scale.Dispose()
	drag.Dispose()
}

func TestScaleFirstRotation(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding)
	rotation := foundation.None[float64]()
	scale.OnUpdate = func(d ScaleUpdateDetails) { rotation = foundation.Some(d.Rotation) }
	pointer1 := newTestPointer()
	down := pointer1.down(ui.Offset{})
	scale.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	pointer2 := newTestPointer(2)
	down2 := pointer2.down(off(10, 10))
	scale.AddPointer(down2)
	g.closeArena(2)
	g.route(down2)
	if rotation.IsSome() {
		t.Fatal("rotated before moving")
	}
	g.route(pointer2.move(off(0, 10)))
	if rotation != foundation.Some(math.Pi/4) {
		t.Errorf("rotation = %v, want pi/4", rotation)
	}
	scale.Dispose()
}

func TestScalePointerCount(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding)
	var start, update, end int
	scale.OnStart = func(d ScaleStartDetails) { start = d.PointerCount }
	scale.OnUpdate = func(d ScaleUpdateDetails) { update = d.PointerCount }
	scale.OnEnd = func(d ScaleEndDetails) { end = d.PointerCount }
	check := func(what string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %d, want %d", what, got, want)
		}
	}
	pointer1 := newTestPointer()
	down := pointer1.down(ui.Offset{})
	scale.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	check("start", start, 1)
	g.route(pointer1.move(off(20, 30)))
	check("update", update, 1)
	pointer2 := newTestPointer(2)
	down2 := pointer2.down(off(10, 20))
	scale.AddPointer(down2)
	g.closeArena(2)
	g.route(down2)
	check("end", end, 2) // an additional pointer down triggers an end
	g.route(pointer2.move(off(0, 10)))
	check("start", start, 2)
	check("update", update, 2)
	g.route(pointer1.up())
	check("end", end, 1)
	g.route(pointer2.move(off(0, 10)))
	check("start", start, 1)
	check("update", update, 1)
	g.route(pointer2.up())
	check("end", end, 0)
	scale.Dispose()
}

type panZoomState struct {
	didStart, didEnd bool
	focal, delta     foundation.Opt[ui.Offset]
	sc, hs, vs       foundation.Opt[float64]
	count            foundation.Opt[int]
	endVelocity      foundation.Opt[float64]
}

func (s *panZoomState) reset() {
	*s = panZoomState{didStart: s.didStart, didEnd: s.didEnd, endVelocity: s.endVelocity}
}

func (s *panZoomState) attach(scale *ScaleGestureRecognizer) {
	scale.OnStart = func(d ScaleStartDetails) {
		s.didStart = true
		s.focal, s.count = foundation.Some(d.FocalPoint), foundation.Some(d.PointerCount)
	}
	scale.OnUpdate = func(d ScaleUpdateDetails) {
		s.sc, s.hs, s.vs = foundation.Some(d.Scale), foundation.Some(d.HorizontalScale), foundation.Some(d.VerticalScale)
		s.focal, s.delta, s.count = foundation.Some(d.FocalPoint), foundation.Some(d.FocalPointDelta), foundation.Some(d.PointerCount)
	}
	scale.OnEnd = func(d ScaleEndDetails) { s.didEnd = true; s.endVelocity = foundation.Some(d.ScaleVelocity) }
}

func TestScaleFromPanZoomEvents(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding)
	drag := NewHorizontalDragGestureRecognizer(g.binding)
	var s panZoomState
	s.attach(scale)
	pointer := newTestPointer(2)
	pointer.kind = ui.PointerDeviceKindTrackpad
	start := pointer.panZoomStart(ui.Offset{})
	scale.AddPointerPanZoom(start)
	drag.AddPointerPanZoom(start)
	g.closeArena(2)
	g.route(start)
	if s.didStart || s.sc.IsSome() || s.didEnd {
		t.Fatal("something happened at the start")
	}
	g.route(pointer.panZoomUpdate(ui.Offset{}, off(20, 30), 1, 0))
	if !s.didStart || s.focal != foundation.Some(off(20, 30)) || s.sc != foundation.Some(1.0) ||
		s.delta != foundation.Some(off(20, 30)) || s.count != foundation.Some(2) || s.didEnd {
		t.Fatalf("after the first update: %+v", s)
	}
	s.didStart = false
	s.reset()
	g.route(pointer.panZoomUpdate(ui.Offset{}, off(20, 30), 2, 0))
	if s.focal != foundation.Some(off(20, 30)) || s.sc != foundation.Some(2.0) || s.hs != foundation.Some(2.0) ||
		s.vs != foundation.Some(2.0) || s.delta != foundation.Some(ui.Offset{}) || s.count != foundation.Some(2) {
		t.Fatalf("after the zoom: %+v", s)
	}
	s.reset()
	g.route(pointer.panZoomUpdate(ui.Offset{}, off(20, 30), 1, 0))
	if s.sc != foundation.Some(1.0) || s.hs != foundation.Some(1.0) || s.vs != foundation.Some(1.0) ||
		s.delta != foundation.Some(ui.Offset{}) {
		t.Fatalf("after zooming back: %+v", s)
	}
	s.reset()
	g.route(pointer.panZoomEnd(0))
	if s.didStart || s.focal.IsSome() || s.sc.IsSome() || !s.didEnd {
		t.Errorf("after the end: %+v", s)
	}
	scale.Dispose()
	drag.Dispose()
}

func TestScaleTrackpadScrollCausesScale(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding, ScaleGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorStart, TrackpadScrollCausesScale: true})
	var s panZoomState
	s.attach(scale)
	pointer := newTestPointer(2)
	pointer.kind = ui.PointerDeviceKindTrackpad
	start := pointer.panZoomStart(ui.Offset{})
	scale.AddPointerPanZoom(start)
	g.closeArena(2)
	g.route(start)
	if !s.didStart || s.sc.IsSome() || s.focal != foundation.Some(ui.Offset{}) || s.count != foundation.Some(2) {
		t.Fatalf("after the start: %+v", s)
	}
	s.didStart = false
	s.reset()
	for _, pan := range []ui.Offset{off(0, -200), off(200, -200)} {
		g.route(pointer.panZoomUpdate(ui.Offset{}, pan, 1, 0))
		if s.didStart || s.focal != foundation.Some(ui.Offset{}) || s.delta != foundation.Some(ui.Offset{}) || s.count != foundation.Some(2) {
			t.Fatalf("after %v: %+v", pan, s)
		}
		if sc, _ := s.sc.Get(); math.Abs(sc-math.E) > 1e-9 {
			t.Errorf("scale after %v = %v, want e", pan, sc)
		}
		s.reset()
	}
	g.route(pointer.panZoomEnd(0))
	if !s.didEnd || s.sc.IsSome() {
		t.Errorf("after the end: %+v", s)
	}
	scale.Dispose()
}

func TestScaleEndingVelocity(t *testing.T) {
	g := newGestureTester(t)
	scale := NewScaleGestureRecognizer(g.binding, ScaleGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorStart, TrackpadScrollCausesScale: true})
	var s panZoomState
	s.attach(scale)
	pointer := newTestPointer(2)
	pointer.kind = ui.PointerDeviceKindTrackpad
	start := pointer.panZoomStart(ui.Offset{})
	scale.AddPointerPanZoom(start)
	g.closeArena(2)
	g.route(start)
	if !s.didStart || s.focal != foundation.Some(ui.Offset{}) || s.didEnd {
		t.Fatalf("after the start: %+v", s)
	}
	for i := 0; i < 100; i++ {
		g.route(pointer.panZoomUpdate(ui.Offset{}, off(0, float64(i)*-10), 1, ms(i*25)))
	}
	g.route(pointer.panZoomEnd(ms(2500)))
	if !s.didEnd {
		t.Fatal("did not end")
	}
	if v, _ := s.endVelocity.Get(); math.Abs(v-281.41454098027765) > 1e-6 {
		t.Errorf("scale velocity = %v, want 281.41454098027765", v)
	}
	scale.Dispose()
}
