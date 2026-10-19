package gestures

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/gestures/gesture_binding_test.dart, and from
// gesture_binding_resample_event_test.dart. The tests of the platform view hit
// test (ui.HitTestRequest) are not ported: Glutter has no platform views. The
// 'Error handling' test is in error_test.go.

// recordingTarget is a hit test target that records the events it gets, as the
// overridden handleEvent of Flutter's TestGestureFlutterBinding does.
type recordingTarget struct {
	events  *[]PointerEvent
	onEvent func(PointerEvent)
}

func (r recordingTarget) HandleEvent(event PointerEvent, _ HitTestEntry) {
	*r.events = append(*r.events, event)
	if r.onEvent != nil {
		r.onEvent(event)
	}
}

type bindingTester struct {
	t       *testing.T
	host    *fakeHost
	d       *ui.PlatformDispatcher
	sched   *scheduler.SchedulerBinding
	binding *GestureBinding
	events  []PointerEvent
	onEvent func(PointerEvent)
}

func newBindingTester(t *testing.T, dpr float64) *bindingTester {
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: dpr})
	sched := scheduler.NewSchedulerBinding(d)
	bt := &bindingTester{t: t, host: host, d: d, sched: sched, binding: NewGestureBinding(d, sched)}
	// The target is in front of the binding, so it sees every event first.
	bt.binding.OnHitTestInView = func(result *HitTestResult, _ ui.Offset, _ ui.ViewID) {
		result.Add(NewHitTestEntry(recordingTarget{events: &bt.events, onEvent: func(e PointerEvent) {
			if bt.onEvent != nil {
				bt.onEvent(e)
			}
		}}))
	}
	return bt
}

func (bt *bindingTester) send(data ...ui.PointerData) {
	bt.d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: data})
	bt.host.flush()
}

func typeNames(events []PointerEvent) []string {
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = foundation.RuntimeTypeName(e)
	}
	return names
}

func checkTypes(t *testing.T, got []PointerEvent, want ...string) {
	t.Helper()
	names := typeNames(got)
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("event %d is %s, want %s (all: %v)", i, names[i], want[i], names)
		}
	}
}

func TestBindingPointerEvents(t *testing.T) {
	tests := []struct {
		name   string
		data   []ui.PointerData
		events []string
	}{
		{"tap", []ui.PointerData{{Change: ui.PointerChangeDown}, {Change: ui.PointerChangeUp}},
			[]string{"PointerDownEvent", "PointerUpEvent"}},
		{"move", []ui.PointerData{{Change: ui.PointerChangeDown}, {Change: ui.PointerChangeMove}, {Change: ui.PointerChangeUp}},
			[]string{"PointerDownEvent", "PointerMoveEvent", "PointerUpEvent"}},
		{"cancel", []ui.PointerData{{Change: ui.PointerChangeDown}, {}},
			[]string{"PointerDownEvent", "PointerCancelEvent"}},
		{"pan/zoom", []ui.PointerData{{Change: ui.PointerChangePanZoomStart}, {Change: ui.PointerChangePanZoomUpdate}, {Change: ui.PointerChangePanZoomEnd}},
			[]string{"PointerPanZoomStartEvent", "PointerPanZoomUpdateEvent", "PointerPanZoomEndEvent"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bt := newBindingTester(t, 1)
			bt.send(tc.data...)
			checkTypes(t, bt.events, tc.events...)
		})
	}
}

func TestBindingPointerHoverEvents(t *testing.T) {
	bt := newBindingTester(t, 1)
	var routed []PointerEvent
	bt.binding.PointerRouter.AddGlobalRoute(func(e PointerEvent) { routed = append(routed, e) }, nil)
	bt.send(
		ui.PointerData{Change: ui.PointerChangeAdd},
		ui.PointerData{Change: ui.PointerChangeHover},
		ui.PointerData{Change: ui.PointerChangeHover},
		ui.PointerData{Change: ui.PointerChangeRemove},
		ui.PointerData{Change: ui.PointerChangeAdd},
		ui.PointerData{Change: ui.PointerChangeHover},
	)
	checkTypes(t, bt.events, "PointerHoverEvent", "PointerHoverEvent", "PointerHoverEvent")
	checkTypes(t, routed, "PointerAddedEvent", "PointerHoverEvent", "PointerHoverEvent",
		"PointerRemovedEvent", "PointerAddedEvent", "PointerHoverEvent")
}

func TestBindingCanCancelPointers(t *testing.T) {
	bt := newBindingTester(t, 1)
	bt.onEvent = func(e PointerEvent) {
		if _, ok := e.(*PointerDownEvent); ok {
			bt.binding.CancelPointer(e.Pointer())
		}
	}
	bt.send(ui.PointerData{Change: ui.PointerChangeDown}, ui.PointerData{Change: ui.PointerChangeUp})
	checkTypes(t, bt.events, "PointerDownEvent", "PointerCancelEvent")
}

func TestBindingCancelPointerFlushesLater(t *testing.T) {
	bt := newBindingTester(t, 1)
	bt.send(ui.PointerData{Change: ui.PointerChangeDown})
	bt.binding.CancelPointer(0)
	if len(bt.events) != 1 {
		t.Fatalf("the cancel event was dispatched at once: %v", typeNames(bt.events))
	}
	bt.host.flush()
	checkTypes(t, bt.events, "PointerDownEvent", "PointerCancelEvent")
}

func TestBindingLockedHoldsEvents(t *testing.T) {
	bt := newBindingTester(t, 1)
	unlock := bt.sched.LockEvents()
	bt.send(ui.PointerData{Change: ui.PointerChangeDown}, ui.PointerData{Change: ui.PointerChangeUp})
	if len(bt.events) != 0 {
		t.Fatalf("events were dispatched while locked: %v", typeNames(bt.events))
	}
	unlock()
	checkTypes(t, bt.events, "PointerDownEvent", "PointerUpEvent")
}

func TestBindingGestureArenaAndRouting(t *testing.T) {
	// The binding closes the arena on a down event and sweeps it on the up
	// event, so the first member wins on the up event; both events are routed.
	bt := newBindingTester(t, 1)
	first, second := &testGestureArenaMember{t: t, key: 0}, &testGestureArenaMember{t: t, key: 0}
	var routed []PointerEvent
	bt.binding.PointerRouter.AddRoute(0, func(e PointerEvent) { routed = append(routed, e) }, nil)
	bt.binding.GestureArena.Add(0, first)
	bt.binding.GestureArena.Add(0, second)
	bt.send(ui.PointerData{Change: ui.PointerChangeDown})
	if first.acceptRan || first.rejectRan || second.acceptRan || second.rejectRan {
		t.Fatal("the arena resolved on the down event")
	}
	bt.send(ui.PointerData{Change: ui.PointerChangeUp})
	if !first.acceptRan || first.rejectRan || second.acceptRan || !second.rejectRan {
		t.Errorf("first accepted %v rejected %v, second accepted %v rejected %v; want the first to win",
			first.acceptRan, first.rejectRan, second.acceptRan, second.rejectRan)
	}
	checkTypes(t, routed, "PointerDownEvent", "PointerUpEvent")
}

func TestConverterExpand(t *testing.T) {
	const devicePixelRatio = 2.5
	dpr := func(ui.ViewID) foundation.Opt[float64] { return foundation.Some(devicePixelRatio) }

	t.Run("add and hover", func(t *testing.T) {
		events := PointerEventConverterExpand([]ui.PointerData{
			{Change: ui.PointerChangeAdd, Device: 24}, {Change: ui.PointerChangeHover, Device: 24},
			{Change: ui.PointerChangeRemove, Device: 24}, {Change: ui.PointerChangeAdd, Device: 24},
			{Change: ui.PointerChangeHover, Device: 24},
		}, dpr)
		checkTypes(t, events, "PointerAddedEvent", "PointerHoverEvent", "PointerRemovedEvent",
			"PointerAddedEvent", "PointerHoverEvent")
	})

	t.Run("malformed scrolling events", func(t *testing.T) {
		scroll := func(dx, dy float64) ui.PointerData {
			return ui.PointerData{SignalKind: ui.PointerSignalKindScroll, Device: 24, ScrollDeltaX: dx, ScrollDeltaY: dy}
		}
		inf, nan := math.Inf(1), math.NaN()
		malformed := []ui.PointerData{
			scroll(inf, 10), scroll(nan, 10), scroll(math.Inf(-1), 10),
			scroll(10, inf), scroll(10, nan), scroll(10, math.Inf(-1)),
		}
		if events := PointerEventConverterExpand(malformed, dpr); len(events) != 0 {
			t.Errorf("got %v for malformed scroll events, want none", typeNames(events))
		}
		valid := []ui.PointerData{scroll(10, 10)}
		events := PointerEventConverterExpand(valid, dpr)
		checkTypes(t, events, "PointerScrollEvent")
		if got, want := events[0].(*PointerScrollEvent).ScrollDelta(), (ui.Offset{Dx: 4, Dy: 4}); got != want {
			t.Errorf("scroll delta = %v, want %v", got, want)
		}
		// An invalid device pixel ratio drops the event.
		zero := func(ui.ViewID) foundation.Opt[float64] { return foundation.Some(0.0) }
		if events := PointerEventConverterExpand(valid, zero); len(events) != 0 {
			t.Errorf("got %v with a device pixel ratio of 0, want none", typeNames(events))
		}
	})

	t.Run("scroll events", func(t *testing.T) {
		events := PointerEventConverterExpand([]ui.PointerData{
			{Change: ui.PointerChangeAdd},
			{Change: ui.PointerChangeHover, SignalKind: ui.PointerSignalKindScroll},
		}, dpr)
		checkTypes(t, events, "PointerAddedEvent", "PointerScrollEvent")
	})

	t.Run("other signals", func(t *testing.T) {
		events := PointerEventConverterExpand([]ui.PointerData{
			{SignalKind: ui.PointerSignalKindScrollInertiaCancel},
			{SignalKind: ui.PointerSignalKindScale, Scale: 2},
			{SignalKind: ui.PointerSignalKindUnknown},
		}, dpr)
		checkTypes(t, events, "PointerScrollInertiaCancelEvent", "PointerScaleEvent")
		if got := events[1].(*PointerScaleEvent).Scale(); got != 2 {
			t.Errorf("scale = %v, want 2", got)
		}
	})

	t.Run("logical pixels", func(t *testing.T) {
		events := PointerEventConverterExpand([]ui.PointerData{{
			Change: ui.PointerChangeMove, PhysicalX: 25, PhysicalY: 50, PhysicalDeltaX: 5, PhysicalDeltaY: 10,
			RadiusMajor: 5, RadiusMinor: 2.5, RadiusMin: 1.25, RadiusMax: 10, Pressure: 0.5,
			PlatformData: 7, PointerIdentifier: 3,
		}}, dpr)
		checkTypes(t, events, "PointerMoveEvent")
		e := events[0]
		if e.Position() != (ui.Offset{Dx: 10, Dy: 20}) || e.Delta() != (ui.Offset{Dx: 2, Dy: 4}) {
			t.Errorf("position %v, delta %v", e.Position(), e.Delta())
		}
		if e.RadiusMajor() != 2 || e.RadiusMinor() != 1 || e.RadiusMin() != 0.5 || e.RadiusMax() != 4 {
			t.Errorf("radii %v %v %v %v", e.RadiusMajor(), e.RadiusMinor(), e.RadiusMin(), e.RadiusMax())
		}
		if e.Pressure() != 0.5 || e.PlatformData() != 7 || e.Pointer() != 3 {
			t.Errorf("pressure %v, platform data %d, pointer %d", e.Pressure(), e.PlatformData(), e.Pointer())
		}
	})

	t.Run("pan/zoom update", func(t *testing.T) {
		events := PointerEventConverterExpand([]ui.PointerData{{
			Change: ui.PointerChangePanZoomUpdate, PanX: 25, PanY: 50, PanDeltaX: 5, PanDeltaY: 10, Scale: 2, Rotation: 1,
		}}, dpr)
		checkTypes(t, events, "PointerPanZoomUpdateEvent")
		e := events[0].(*PointerPanZoomUpdateEvent)
		if e.Pan() != (ui.Offset{Dx: 10, Dy: 20}) || e.PanDelta() != (ui.Offset{Dx: 2, Dy: 4}) || e.Scale() != 2 || e.Rotation() != 1 {
			t.Errorf("pan %v, delta %v, scale %v, rotation %v", e.Pan(), e.PanDelta(), e.Scale(), e.Rotation())
		}
	})

	t.Run("a scroll responds through the data", func(t *testing.T) {
		var allowed bool
		events := PointerEventConverterExpand([]ui.PointerData{{
			SignalKind: ui.PointerSignalKindScroll, OnRespond: func(a bool) { allowed = a },
		}}, dpr)
		events[0].(PointerSignalEvent).Respond(true)
		if !allowed {
			t.Error("the data's OnRespond was not called")
		}
	})
}

func TestConverterButtons(t *testing.T) {
	const devicePixelRatio = 2.5
	dpr := func(ui.ViewID) foundation.Opt[float64] { return foundation.Some(devicePixelRatio) }
	location := ui.Offset{Dx: 10, Dy: 10}.Mul(devicePixelRatio)
	at := func(change ui.PointerChange, kind ui.PointerDeviceKind, buttons int) ui.PointerData {
		return ui.PointerData{Change: change, Kind: kind, Buttons: buttons, PhysicalX: location.Dx, PhysicalY: location.Dy}
	}
	tests := []struct {
		name string
		kind ui.PointerDeviceKind
		// the buttons in the data of down and move, and the buttons expected.
		downIn, moveIn   int
		downOut, moveOut int
	}{
		{"touch with no button", ui.PointerDeviceKindTouch, 0, 0, PrimaryButton, PrimaryButton},
		{"touch with a button", ui.PointerDeviceKindTouch, SecondaryButton, SecondaryButton, SecondaryButton, SecondaryButton},
		{"stylus", ui.PointerDeviceKindStylus, 0, SecondaryStylusButton, PrimaryButton, SecondaryStylusButton},
		{"inverted stylus", ui.PointerDeviceKindInvertedStylus, 0, SecondaryStylusButton, PrimaryButton, SecondaryStylusButton},
		{"unknown", ui.PointerDeviceKindUnknown, 0, SecondaryButton, PrimaryButton, SecondaryButton},
		{"mouse", ui.PointerDeviceKindMouse, MiddleMouseButton, MiddleMouseButton | SecondaryMouseButton, MiddleMouseButton, MiddleMouseButton | SecondaryMouseButton},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			events := PointerEventConverterExpand([]ui.PointerData{
				at(ui.PointerChangeAdd, tc.kind, 0), at(ui.PointerChangeHover, tc.kind, 0),
				at(ui.PointerChangeDown, tc.kind, tc.downIn), at(ui.PointerChangeMove, tc.kind, tc.moveIn),
				at(ui.PointerChangeUp, tc.kind, 0),
			}, dpr)
			checkTypes(t, events, "PointerAddedEvent", "PointerHoverEvent", "PointerDownEvent", "PointerMoveEvent", "PointerUpEvent")
			for i, want := range []int{0, 0, tc.downOut, tc.moveOut, 0} {
				if got := events[i].Buttons(); got != want {
					t.Errorf("event %d (%s) has buttons %d, want %d", i, typeNames(events)[i], got, want)
				}
			}
		})
	}
}

func TestConverterProcessesViewIDs(t *testing.T) {
	const startID = 987654
	changes := []ui.PointerChange{
		ui.PointerChangeCancel, ui.PointerChangeAdd, ui.PointerChangeRemove, ui.PointerChangeHover,
		ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangePanZoomStart,
		ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomEnd,
	}
	var data []ui.PointerData
	var want []ui.ViewID
	for i, c := range changes {
		data = append(data, ui.PointerData{ViewID: startID + ui.ViewID(i), Change: c})
		want = append(want, startID+ui.ViewID(i))
	}
	var asked []ui.ViewID
	events := PointerEventConverterExpand(data, func(id ui.ViewID) foundation.Opt[float64] {
		asked = append(asked, id)
		return foundation.Some(float64(id) / 10)
	})
	if len(events) != 10 {
		t.Fatalf("got %d events, want 10", len(events))
	}
	for i := range want {
		if asked[i] != want[i] || events[i].ViewID() != want[i] {
			t.Errorf("event %d: asked for view %d, event has view %d; want %d", i, asked[i], events[i].ViewID(), want[i])
		}
	}
}

func TestConverterDropsUnknownViews(t *testing.T) {
	events := PointerEventConverterExpand([]ui.PointerData{{Change: ui.PointerChangeAdd}},
		func(ui.ViewID) foundation.Opt[float64] { return foundation.None[float64]() })
	if len(events) != 0 {
		t.Errorf("got %v for a view that does not exist, want none", typeNames(events))
	}
}

func TestBindingHitTestAndDispatch(t *testing.T) {
	bt := newBindingTester(t, 2)
	var positions []ui.Offset
	var transforms []*ui.Matrix4
	bt.binding.OnHitTestInView = func(result *HitTestResult, position ui.Offset, _ ui.ViewID) {
		result.PushOffset(ui.Offset{Dx: -10, Dy: -10})
		result.Add(NewHitTestEntry(recordingTarget{events: &bt.events, onEvent: func(e PointerEvent) {
			positions = append(positions, e.LocalPosition())
			transforms = append(transforms, e.Transform())
		}}))
		result.PopTransform()
	}
	bt.send(ui.PointerData{Change: ui.PointerChangeDown, PhysicalX: 60, PhysicalY: 80},
		ui.PointerData{Change: ui.PointerChangeMove, PhysicalX: 80, PhysicalY: 80},
		ui.PointerData{Change: ui.PointerChangeUp, PhysicalX: 80, PhysicalY: 80})
	// The device pixel ratio is 2, so the down event is at (30, 40) and the
	// target sees it relative to its own origin, which is at (10, 10).
	want := []ui.Offset{{Dx: 20, Dy: 30}, {Dx: 30, Dy: 30}, {Dx: 30, Dy: 30}}
	if len(positions) != 3 {
		t.Fatalf("the target got %d events, want 3", len(positions))
	}
	for i := range want {
		if positions[i] != want[i] {
			t.Errorf("local position %d = %v, want %v", i, positions[i], want[i])
		}
		if transforms[i] == nil {
			t.Errorf("event %d has no transform", i)
		}
	}
}

// sampleClock adapts the fake clock to the SamplingClock of the binding.
// fakeScheduler is the scheduler of the resampling test: it keeps the post-frame
// callback and the frame time the test sets.
