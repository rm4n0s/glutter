package gestures

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/pointer_router_test.dart. The tests
// 'Exceptions do not stop pointer routing' and 'Exceptions include router, route
// & event' come with step 7.3, which adds the error reporting.

func TestRouterShouldRoutePointers(t *testing.T) {
	callbackRan := false
	callback := func(PointerEvent) { callbackRan = true }
	pointer2, pointer3 := newTestPointer(2), newTestPointer(3)
	router := NewPointerRouter()
	handle := router.AddRoute(3, callback, nil)

	router.Route(pointer2.down(ui.Offset{}))
	if callbackRan {
		t.Error("the route of pointer 3 ran for pointer 2")
	}
	router.Route(pointer3.down(ui.Offset{}))
	if !callbackRan {
		t.Error("the route of pointer 3 did not run for pointer 3")
	}
	callbackRan = false
	router.RemoveRoute(3, handle)
	router.Route(pointer3.up())
	if callbackRan {
		t.Error("the route ran after it was removed")
	}
}

func TestRouterSupportsReentrantCancellation(t *testing.T) {
	callbackRan := false
	router := NewPointerRouter()
	var handle PointerRouteHandle
	router.AddRoute(2, func(PointerEvent) { router.RemoveRoute(2, handle) }, nil)
	handle = router.AddRoute(2, func(PointerEvent) { callbackRan = true }, nil)
	router.Route(newTestPointer(2).down(ui.Offset{}))
	if callbackRan {
		t.Error("a route that was removed by an earlier route ran")
	}
}

func TestRouterSupportsGlobalCallbacks(t *testing.T) {
	firstRan, secondRan := false, false
	router := NewPointerRouter()
	router.AddGlobalRoute(func(PointerEvent) {
		firstRan = true
		router.AddGlobalRoute(func(PointerEvent) { secondRan = true }, nil)
	}, nil)
	router.Route(newTestPointer(2).down(ui.Offset{}))
	if !firstRan || secondRan {
		t.Errorf("first ran %v, second ran %v; want true, false", firstRan, secondRan)
	}
}

func TestRouterSupportsReentrantGlobalCancellation(t *testing.T) {
	callbackRan := false
	router := NewPointerRouter()
	var handle PointerRouteHandle
	router.AddGlobalRoute(func(PointerEvent) { router.RemoveGlobalRoute(handle) }, nil)
	handle = router.AddGlobalRoute(func(PointerEvent) { callbackRan = true }, nil)
	router.Route(newTestPointer(2).down(ui.Offset{}))
	if callbackRan {
		t.Error("a global route that was removed by an earlier route ran")
	}
}

func TestRouterPerPointerCannotAddGlobalRoutesReentrantly(t *testing.T) {
	callbackRan, perPointerRan := false, false
	router := NewPointerRouter()
	router.AddRoute(2, func(PointerEvent) {
		perPointerRan = true
		router.AddGlobalRoute(func(PointerEvent) { callbackRan = true }, nil)
	}, nil)
	router.Route(newTestPointer(2).down(ui.Offset{}))
	if !perPointerRan || callbackRan {
		t.Errorf("per-pointer ran %v, new global ran %v; want true, false", perPointerRan, callbackRan)
	}
}

func TestRouterPerPointerBeforeGlobal(t *testing.T) {
	var log []string
	add := func(s string) PointerRoute { return func(PointerEvent) { log = append(log, s) } }
	router := NewPointerRouter()
	router.AddGlobalRoute(add("global 1"), nil)
	router.AddRoute(2, add("per-pointer 1"), nil)
	router.AddGlobalRoute(add("global 2"), nil)
	router.AddRoute(2, add("per-pointer 2"), nil)
	router.Route(newTestPointer(2).down(ui.Offset{}))
	want := []string{"per-pointer 1", "per-pointer 2", "global 1", "global 2"}
	if !reflect.DeepEqual(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
}

func TestRouterShouldTransformEvents(t *testing.T) {
	var events, globalEvents []PointerEvent
	router := NewPointerRouter()
	transform := ui.NewMatrix4Identity()
	transform.Scale(0.5, 0.5, 1)
	transform = transform.Multiply(ui.NewMatrix4TranslationValues(-10, -30, 0))
	router.AddRoute(1, func(e PointerEvent) { events = append(events, e) }, transform)
	router.AddGlobalRoute(func(e PointerEvent) { globalEvents = append(globalEvents, e) }, transform)

	pointer1 := newTestPointer()
	first := ui.Offset{Dx: 16, Dy: 36}
	second := ui.Offset{Dx: 20, Dy: 40}
	steps := []struct {
		name          string
		event         PointerEvent
		position      ui.Offset
		delta         ui.Offset
		localPosition ui.Offset
		localDelta    ui.Offset
	}{
		{"down", pointer1.down(first), first, ui.Offset{}, ui.Offset{Dx: 3, Dy: 3}, ui.Offset{}},
		{"move", nil, second, second.Sub(first), ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 2, Dy: 2}},
	}
	for _, s := range steps {
		if s.event == nil {
			s.event = pointer1.move(second)
		}
		events, globalEvents = nil, nil
		router.Route(s.event)
		for kind, got := range map[string][]PointerEvent{"route": events, "global route": globalEvents} {
			if len(got) != 1 {
				t.Fatalf("%s: %s got %d events, want 1", s.name, kind, len(got))
			}
			e := got[0]
			if e.Transform() != transform {
				t.Errorf("%s: %s transform = %v", s.name, kind, e.Transform())
			}
			if e.Position() != s.position || e.Delta() != s.delta || e.LocalPosition() != s.localPosition || e.LocalDelta() != s.localDelta {
				t.Errorf("%s: %s got position %v delta %v local %v %v; want %v %v %v %v", s.name, kind,
					e.Position(), e.Delta(), e.LocalPosition(), e.LocalDelta(),
					s.position, s.delta, s.localPosition, s.localDelta)
			}
		}
	}
}

func TestRouterRoutesCancelToThePointer(t *testing.T) {
	var got []PointerEvent
	router := NewPointerRouter()
	router.AddRoute(5, func(e PointerEvent) { got = append(got, e) }, nil)
	p := newTestPointer(5)
	router.Route(p.down(ui.Offset{Dx: 1, Dy: 1}))
	router.Route(p.cancel())
	checkTypes(t, got, "PointerDownEvent", "PointerCancelEvent")
}
