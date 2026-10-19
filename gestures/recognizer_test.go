package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func evDown(pointer int, p ui.Offset) *PointerDownEvent {
	return NewPointerDownEvent(PointerEventOpts{Pointer: pointer, Position: p})
}
func evMove(pointer int, p ui.Offset) *PointerMoveEvent {
	return NewPointerMoveEvent(PointerEventOpts{Pointer: pointer, Position: p})
}
func evUp(pointer int, p ui.Offset) *PointerUpEvent {
	return NewPointerUpEvent(PointerEventOpts{Pointer: pointer, Position: p})
}

// testGestureRecognizer is recognizer_test.dart's TestGestureRecognizer.
type testGestureRecognizer struct{ GestureRecognizerBase }

func (r *testGestureRecognizer) DebugDescription() string                    { return "debugDescription content" }
func (r *testGestureRecognizer) AddPointer(*PointerDownEvent)                {}
func (r *testGestureRecognizer) AcceptGesture(int)                           {}
func (r *testGestureRecognizer) RejectGesture(int)                           {}
func (r *testGestureRecognizer) AddPointerPanZoom(*PointerPanZoomStartEvent) {}

// indefiniteGestureRecognizer never resolves, so the arena stays open.
type indefiniteGestureRecognizer struct {
	GestureRecognizerBase
	b *GestureBinding
}

func newIndefinite(b *GestureBinding) *indefiniteGestureRecognizer {
	r := &indefiniteGestureRecognizer{b: b}
	r.Init(b, r, GestureRecognizerOpts{})
	return r
}
func (r *indefiniteGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.b.GestureArena.Add(event.Pointer(), r)
}
func (r *indefiniteGestureRecognizer) AcceptGesture(int)        {}
func (r *indefiniteGestureRecognizer) RejectGesture(int)        {}
func (r *indefiniteGestureRecognizer) DebugDescription() string { return "Unresolving" }

// testPrimaryPointerRecognizer resolves the arena when it sees an event of the
// type in resolveOn.
type testPrimaryPointerRecognizer struct {
	PrimaryPointerGestureRecognizer
	resolution  GestureDisposition
	resolveOn   func(PointerEvent) bool
	resolutions *[]string
}

func newTestPrimaryPointer(b *GestureBinding, resolution GestureDisposition, resolveOn func(PointerEvent) bool,
	resolutions *[]string, pre, post SlopTolerance) *testPrimaryPointerRecognizer {
	r := &testPrimaryPointerRecognizer{resolution: resolution, resolveOn: resolveOn, resolutions: resolutions}
	r.InitPrimaryPointer(b, r, PrimaryPointerOpts{PreAcceptSlopTolerance: pre, PostAcceptSlopTolerance: post})
	return r
}
func (r *testPrimaryPointerRecognizer) AcceptGesture(p int) {
	r.PrimaryPointerGestureRecognizer.AcceptGesture(p)
	*r.resolutions = append(*r.resolutions, "accepted")
}
func (r *testPrimaryPointerRecognizer) RejectGesture(p int) {
	r.PrimaryPointerGestureRecognizer.RejectGesture(p)
	*r.resolutions = append(*r.resolutions, "rejected")
}
func (r *testPrimaryPointerRecognizer) HandlePrimaryPointer(event PointerEvent) {
	if r.resolveOn(event) {
		r.Resolve(r.resolution)
	}
}
func (r *testPrimaryPointerRecognizer) DebugDescription() string { return "TestPrimaryPointer" }

func isUp(e PointerEvent) bool   { _, ok := e.(*PointerUpEvent); return ok }
func isMove(e PointerEvent) bool { _, ok := e.(*PointerMoveEvent); return ok }

func TestOffsetPair(t *testing.T) {
	o1 := OffsetPair{Local: ui.Offset{Dx: 10, Dy: 20}, Global: ui.Offset{Dx: 30, Dy: 40}}
	o2 := OffsetPair{Local: ui.Offset{Dx: 50, Dy: 60}, Global: ui.Offset{Dx: 70, Dy: 80}}
	if got := o2.Add(o1); got.Local != (ui.Offset{Dx: 60, Dy: 80}) || got.Global != (ui.Offset{Dx: 100, Dy: 120}) {
		t.Errorf("sum = %+v", got)
	}
	if got := o2.Sub(o1); got.Local != (ui.Offset{Dx: 40, Dy: 40}) || got.Global != (ui.Offset{Dx: 40, Dy: 40}) {
		t.Errorf("difference = %+v", got)
	}
}

func TestPrimaryPointerCleansUpAfterWinning(t *testing.T) {
	g := newGestureTester(t)
	var res []string
	indefinite := newIndefinite(g.binding)
	accepting := newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, SlopTolerance{}, SlopTolerance{})
	if accepting.State() != GestureRecognizerStateReady || accepting.PrimaryPointer().IsSome() || accepting.InitialPosition().IsSome() || len(res) != 0 {
		t.Fatal("the initial state is wrong")
	}
	down := evDown(5, ui.Offset{Dx: 10, Dy: 10})
	indefinite.AddPointer(down)
	accepting.AddPointer(down)
	if accepting.State() != GestureRecognizerStatePossible || accepting.PrimaryPointer().Value() != 5 {
		t.Fatalf("state = %v, primary = %v", accepting.State(), accepting.PrimaryPointer())
	}
	if p := accepting.InitialPosition().Value(); p.Global != down.Position() || p.Local != down.LocalPosition() {
		t.Errorf("initial position = %+v", p)
	}
	g.closeArena(5)
	g.flush()
	g.route(down)
	g.route(evUp(5, ui.Offset{Dx: 15, Dy: 15}))
	if accepting.State() != GestureRecognizerStateReady || accepting.PrimaryPointer().Value() != 5 || accepting.InitialPosition().IsSome() {
		t.Errorf("state = %v, initial = %v", accepting.State(), accepting.InitialPosition())
	}
	if len(res) != 1 || res[0] != "accepted" {
		t.Errorf("resolutions = %v", res)
	}
}

func TestPrimaryPointerCleansUpAfterLosing(t *testing.T) {
	g := newGestureTester(t)
	var res []string
	indefinite := newIndefinite(g.binding)
	rejecting := newTestPrimaryPointer(g.binding, GestureRejected, isMove, &res, SlopTolerance{}, SlopTolerance{})
	down := evDown(5, ui.Offset{Dx: 10, Dy: 10})
	indefinite.AddPointer(down)
	rejecting.AddPointer(down)
	g.closeArena(5)
	g.flush()
	g.route(down)
	g.route(evMove(5, ui.Offset{Dx: 15, Dy: 15}))
	if rejecting.State() != GestureRecognizerStateDefunct || !rejecting.InitialPosition().IsSome() || len(res) != 1 || res[0] != "rejected" {
		t.Errorf("state = %v, initial = %v, resolutions = %v", rejecting.State(), rejecting.InitialPosition(), res)
	}
	g.route(evUp(5, ui.Offset{Dx: 15, Dy: 15}))
	if rejecting.State() != GestureRecognizerStateReady || rejecting.InitialPosition().IsSome() || len(res) != 1 {
		t.Errorf("state = %v, initial = %v, resolutions = %v", rejecting.State(), rejecting.InitialPosition(), res)
	}
}

func TestPrimaryPointerWorksWhenRecycled(t *testing.T) {
	g := newGestureTester(t)
	var res []string
	indefinite := newIndefinite(g.binding)
	accepting := newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, NewSlopTolerance(15), NewSlopTolerance(1000))
	origin := ui.Offset{Dx: 10, Dy: 10}
	down := evDown(5, origin)
	indefinite.AddPointer(down)
	accepting.AddPointer(down)
	g.closeArena(5)
	g.flush()
	g.route(down)
	g.route(evUp(5, ui.Offset{Dx: 15, Dy: 15}))
	if len(res) != 1 || res[0] != "accepted" {
		t.Fatalf("resolutions = %v", res)
	}
	res = nil
	down2 := evDown(6, origin)
	indefinite.AddPointer(down2)
	accepting.AddPointer(down2)
	g.closeArena(6)
	g.flush()
	g.route(down2)
	g.route(evMove(6, ui.Offset{Dx: 100, Dy: 200}))
	if len(res) != 1 || res[0] != "rejected" {
		t.Fatalf("resolutions = %v", res)
	}
	g.route(evUp(6, ui.Offset{Dx: 100, Dy: 200}))
	if len(res) != 1 {
		t.Errorf("resolutions = %v", res)
	}
}

func TestPrimaryPointerPreAcceptSlopTolerance(t *testing.T) {
	g := newGestureTester(t)
	var res []string
	indefinite := newIndefinite(g.binding)
	none := SlopToleranceNone
	defaultSlop := newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, SlopTolerance{}, none)
	setSlop := newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, NewSlopTolerance(5), none)
	nullSlop := newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, none, none)
	if defaultSlop.PreAcceptSlopTolerance() != foundation.Some(TouchSlop) ||
		setSlop.PreAcceptSlopTolerance() != foundation.Some(5.0) || nullSlop.PreAcceptSlopTolerance().IsSome() {
		t.Fatal("the pre-accept slop tolerances are wrong")
	}

	run := func(r *testPrimaryPointerRecognizer, pointer int, from, to ui.Offset) {
		down := evDown(pointer, from)
		indefinite.AddPointer(down)
		r.AddPointer(down)
		g.closeArena(pointer)
		g.flush()
		g.route(down)
		g.route(evMove(pointer, to))
		g.route(evUp(pointer, to))
	}
	want := func(w string) {
		t.Helper()
		if len(res) != 1 || res[0] != w {
			t.Errorf("resolutions = %v, want [%s]", res, w)
		}
		res = nil
	}
	run(defaultSlop, 5, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 15, Dy: 15})
	want("accepted")
	defaultSlop.GestureSettings = foundation.Some(DeviceGestureSettings{TouchSlop: foundation.Some(5.0)})
	run(defaultSlop, 5, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 15, Dy: 15})
	want("rejected")
	run(setSlop, 5, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 15, Dy: 15})
	want("rejected")
	run(nullSlop, 6, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 100, Dy: 200})
	want("accepted")
}
