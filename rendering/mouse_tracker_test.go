package rendering

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// testAnnotationTarget is the TestAnnotationTarget of mouse_tracker_test_utils.dart:
// a hit test target that is also a MouseTrackerAnnotation.
type testAnnotationTarget struct {
	onEnter services.PointerEnterEventListener
	onHover services.PointerHoverEventListener
	onExit  services.PointerExitEventListener
	// cursor is the cursor of the region; nil defers. hold makes it keep the cursor of
	// a pressed pointer, and invalid makes it a region that left the tree.
	cursor        services.MouseCursor
	hold, invalid bool
}

func (t *testAnnotationTarget) OnEnter() services.PointerEnterEventListener { return t.onEnter }
func (t *testAnnotationTarget) OnExit() services.PointerExitEventListener   { return t.onExit }
func (t *testAnnotationTarget) Cursor() services.MouseCursor {
	if t.cursor == nil {
		return services.MouseCursorDefer
	}
	return t.cursor
}
func (t *testAnnotationTarget) ValidForMouseTracker() bool { return !t.invalid }

// HoldCursorWhilePressed is what a region with HoldCursorWhilePressed answers.
func (t *testAnnotationTarget) HoldCursorWhilePressed() bool { return t.hold }
func (t *testAnnotationTarget) HandleEvent(event gestures.PointerEvent, _ gestures.HitTestEntry) {
	if h, ok := event.(*gestures.PointerHoverEvent); ok && t.onHover != nil {
		t.onHover(h)
	}
}

type testEntry struct {
	target    *testAnnotationTarget
	transform *ui.Matrix4
}

// trackerHarness plays the part of the RendererBinding for a MouseTracker: it
// hit tests with the entries of find, sends every event to the tracker, and
// sends hover events to the targets that were hit.
type trackerHarness struct {
	tracker *MouseTracker
	find    func(position ui.Offset) []testEntry
}

func newTrackerHarness() *trackerHarness {
	h := &trackerHarness{}
	h.tracker = NewMouseTracker(h.hitTest, services.MouseCursorUncontrolled)
	return h
}

func (h *trackerHarness) hitTest(position ui.Offset, _ ui.ViewID) *gestures.HitTestResult {
	result := gestures.NewHitTestResult()
	if h.find == nil {
		return result
	}
	for _, e := range h.find(position) {
		m := e.transform
		if m == nil {
			m = ui.NewMatrix4Identity()
		}
		result.PushTransform(m)
		result.Add(gestures.NewHitTestEntry(e.target))
		result.PopTransform()
	}
	return result
}

func (h *trackerHarness) send(event gestures.PointerEvent) {
	h.tracker.UpdateWithEvent(event, nil)
	if _, ok := event.(*gestures.PointerHoverEvent); ok {
		for _, entry := range h.hitTest(event.Position(), 0).Path() {
			entry.Target().HandleEvent(event.Transformed(entry.Transform()), entry)
		}
	}
}

func evAt(change string, p ui.Offset, device int, kind ui.PointerDeviceKind) gestures.PointerEvent {
	o := gestures.PointerEventOpts{Position: p, Device: device, Kind: kind}
	switch change {
	case "add":
		return gestures.NewPointerAddedEvent(o)
	case "hover":
		return gestures.NewPointerHoverEvent(o)
	case "remove":
		return gestures.NewPointerRemovedEvent(o)
	case "down":
		return gestures.NewPointerDownEvent(o)
	case "move":
		return gestures.NewPointerMoveEvent(o)
	case "up":
		return gestures.NewPointerUpEvent(o)
	}
	panic(change)
}

func mouseEv(change string, p ui.Offset) gestures.PointerEvent {
	return evAt(change, p, 0, ui.PointerDeviceKindMouse)
}

// critical is the part of an event the tests compare.
type critical struct {
	Type          string
	Position      ui.Offset
	LocalPosition ui.Offset
	Device        int
}

func crit(e gestures.PointerEvent) critical {
	return critical{foundation.RuntimeTypeName(e), e.Position(), e.LocalPosition(), e.Device()}
}

func wantEvents(t *testing.T, got []gestures.PointerEvent, want ...gestures.PointerEvent) {
	t.Helper()
	var g, w []critical
	for _, e := range got {
		g = append(g, crit(e))
	}
	for _, e := range want {
		w = append(w, crit(e))
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("events = %+v, want %+v", g, w)
	}
}

func recordingTarget(events *[]gestures.PointerEvent) *testAnnotationTarget {
	return &testAnnotationTarget{
		onEnter: func(e *gestures.PointerEnterEvent) { *events = append(*events, e) },
		onHover: func(e *gestures.PointerHoverEvent) { *events = append(*events, e) },
		onExit:  func(e *gestures.PointerExitEvent) { *events = append(*events, e) },
	}
}

func enterAt(p ui.Offset, device int, m *ui.Matrix4) gestures.PointerEvent {
	return gestures.NewPointerEnterEvent(gestures.PointerEventOpts{Position: p, Device: device}).Transformed(m)
}
func exitAt(p ui.Offset, device int, m *ui.Matrix4) gestures.PointerEvent {
	return gestures.NewPointerExitEvent(gestures.PointerEventOpts{Position: p, Device: device}).Transformed(m)
}
func hoverAt(p ui.Offset, device int, m *ui.Matrix4) gestures.PointerEvent {
	return gestures.NewPointerHoverEvent(gestures.PointerEventOpts{Position: p, Device: device}).Transformed(m)
}

func off(x, y float64) ui.Offset { return ui.Offset{Dx: x, Dy: y} }

func oneAnnotation(h *trackerHarness, events *[]gestures.PointerEvent) {
	target := recordingTarget(events)
	h.find = func(ui.Offset) []testEntry { return []testEntry{{target: target}} }
}

func TestMouseTrackerDetectsEnterHoverExit(t *testing.T) {
	for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindStylus} {
		t.Run(kind.Name(), func(t *testing.T) {
			h := newTrackerHarness()
			var events []gestures.PointerEvent
			oneAnnotation(h, &events)
			var listenerLogs []bool
			h.tracker.AddListener(func() { listenerLogs = append(listenerLogs, h.tracker.MouseIsConnected()) })
			if h.tracker.MouseIsConnected() {
				t.Fatal("connected at first")
			}
			h.send(evAt("add", ui.Offset{}, 0, kind))
			wantEvents(t, events, enterAt(ui.Offset{}, 0, nil))
			if !reflect.DeepEqual(listenerLogs, []bool{true}) {
				t.Errorf("listener logs = %v", listenerLogs)
			}
			events, listenerLogs = nil, nil
			h.send(evAt("hover", off(1, 101), 0, kind))
			wantEvents(t, events, hoverAt(off(1, 101), 0, nil))
			if !h.tracker.MouseIsConnected() || len(listenerLogs) != 0 {
				t.Errorf("connected = %v, listener logs = %v", h.tracker.MouseIsConnected(), listenerLogs)
			}
			events = nil
			h.send(evAt("remove", off(1, 101), 0, kind))
			wantEvents(t, events, exitAt(off(1, 101), 0, nil))
			if !reflect.DeepEqual(listenerLogs, []bool{false}) {
				t.Errorf("listener logs = %v", listenerLogs)
			}
			events, listenerLogs = nil, nil
			h.send(evAt("add", off(0, 301), 0, kind))
			wantEvents(t, events, enterAt(off(0, 301), 0, nil))
			if !reflect.DeepEqual(listenerLogs, []bool{true}) {
				t.Errorf("listener logs = %v", listenerLogs)
			}
		})
	}
}

func TestMouseTrackerRemovedFirstEvent(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	oneAnnotation(h, &events)
	h.send(mouseEv("remove", ui.Offset{})) // must not crash
	if len(events) != 0 || h.tracker.MouseIsConnected() {
		t.Errorf("events = %v, connected = %v", events, h.tracker.MouseIsConnected())
	}
}

func TestMouseTrackerMultipleDevices(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	oneAnnotation(h, &events)
	dev := func(change string, p ui.Offset, device int) gestures.PointerEvent {
		return evAt(change, p, device, ui.PointerDeviceKindMouse)
	}
	h.send(dev("add", ui.Offset{}, 0))
	h.send(dev("hover", off(0, 1), 0))
	wantEvents(t, events, enterAt(ui.Offset{}, 0, nil), hoverAt(off(0, 1), 0, nil))
	events = nil
	h.send(dev("add", off(0, 401), 1))
	h.send(dev("hover", off(1, 401), 1))
	wantEvents(t, events, enterAt(off(0, 401), 1, nil), hoverAt(off(1, 401), 1, nil))
	events = nil
	h.send(dev("hover", off(0, 101), 0))
	wantEvents(t, events, hoverAt(off(0, 101), 0, nil))
	events = nil
	h.send(dev("hover", off(1, 501), 1))
	wantEvents(t, events, hoverAt(off(1, 501), 1, nil))
	events = nil
	h.send(dev("remove", off(0, 101), 0))
	wantEvents(t, events, exitAt(off(0, 101), 0, nil))
	if !h.tracker.MouseIsConnected() {
		t.Error("the other mouse is still connected")
	}
	events = nil
	h.send(dev("remove", off(1, 501), 1))
	wantEvents(t, events, exitAt(off(1, 501), 1, nil))
	if h.tracker.MouseIsConnected() {
		t.Error("still connected")
	}
}

func TestMouseTrackerIgnoresNonHoverEvents(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	oneAnnotation(h, &events)
	h.send(mouseEv("add", off(0, 101)))
	h.send(mouseEv("down", off(0, 101)))
	wantEvents(t, events, enterAt(off(0, 101), 0, nil))
	events = nil
	h.send(mouseEv("move", off(0, 201)))
	h.send(mouseEv("up", off(0, 301)))
	wantEvents(t, events)
}

func TestMouseTrackerIgnoresTouch(t *testing.T) {
	h := newTrackerHarness()
	h.find = func(ui.Offset) []testEntry { return nil }
	h.send(evAt("add", off(0, 100), 0, ui.PointerDeviceKindTouch))
	if h.tracker.MouseIsConnected() {
		t.Error("a touch pointer is a mouse")
	}
}

func TestMouseTrackerAnnotationAppearsOrDisappearsOnPointer(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	target := recordingTarget(&events)
	translate := ui.NewMatrix4TranslationValues(10, 20, 0)
	inRegion := false
	h.find = func(ui.Offset) []testEntry {
		if inRegion {
			return []testEntry{{target, translate}}
		}
		return nil
	}
	h.send(mouseEv("add", off(0, 100)))
	wantEvents(t, events)
	if !h.tracker.MouseIsConnected() {
		t.Fatal("not connected")
	}
	inRegion = true
	h.tracker.UpdateAllDevices() // the check after a frame
	wantEvents(t, events, enterAt(off(0, 100), 0, translate))
	events = nil
	inRegion = false
	h.tracker.UpdateAllDevices()
	wantEvents(t, events, exitAt(off(0, 100), 0, translate))
}

func TestMouseTrackerPointerAddedOrRemovedOnAnnotation(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	target := recordingTarget(&events)
	translate := ui.NewMatrix4TranslationValues(10, 20, 0)
	h.find = func(ui.Offset) []testEntry { return []testEntry{{target, translate}} }
	h.send(mouseEv("add", off(0, 100)))
	wantEvents(t, events, enterAt(off(0, 100), 0, translate))
	events = nil
	h.send(mouseEv("remove", off(0, 100)))
	wantEvents(t, events, exitAt(off(0, 100), 0, translate))
}

func TestMouseTrackerPointerMovesInOrOutOfAnnotation(t *testing.T) {
	h := newTrackerHarness()
	var events []gestures.PointerEvent
	target := recordingTarget(&events)
	translate := ui.NewMatrix4TranslationValues(10, 20, 0)
	h.find = func(p ui.Offset) []testEntry {
		if p.Dx < 100 {
			return []testEntry{{target, translate}}
		}
		return nil
	}
	h.send(mouseEv("add", off(200, 100)))
	events = nil
	h.send(mouseEv("hover", off(0, 100)))
	wantEvents(t, events, enterAt(off(0, 100), 0, translate), hoverAt(off(0, 100), 0, translate))
	events = nil
	h.send(mouseEv("hover", off(200, 100)))
	wantEvents(t, events, exitAt(off(200, 100), 0, translate))
}

func TestMouseTrackerToleratesUnlistenedEvents(t *testing.T) {
	h := newTrackerHarness()
	a2 := &testAnnotationTarget{onExit: func(*gestures.PointerExitEvent) {}}
	h.find = func(ui.Offset) []testEntry { return []testEntry{{target: a2}} }
	h.send(mouseEv("add", off(0, 101)))
	h.send(mouseEv("hover", off(1, 101))) // must not crash
}

func TestMouseTrackerCallbackOrder(t *testing.T) {
	t.Run("between parents and children", func(t *testing.T) {
		h := newTrackerHarness()
		var logs []string
		mk := func(name string) *testAnnotationTarget {
			return &testAnnotationTarget{
				onEnter: func(*gestures.PointerEnterEvent) { logs = append(logs, "enter"+name) },
				onExit:  func(*gestures.PointerExitEvent) { logs = append(logs, "exit"+name) },
				onHover: func(*gestures.PointerHoverEvent) { logs = append(logs, "hover"+name) },
			}
		}
		a, b := mk("A"), mk("B")
		isInB := false
		h.find = func(ui.Offset) []testEntry {
			if isInB {
				return []testEntry{{target: b}, {target: a}}
			}
			return nil
		}
		h.send(mouseEv("add", off(0, 1)))
		isInB = true
		h.send(mouseEv("hover", off(0, 10)))
		if want := []string{"enterA", "enterB", "hoverB", "hoverA"}; !reflect.DeepEqual(logs, want) {
			t.Errorf("logs = %v, want %v", logs, want)
		}
		logs = nil
		isInB = false
		h.send(mouseEv("hover", off(0, 20)))
		if want := []string{"exitB", "exitA"}; !reflect.DeepEqual(logs, want) {
			t.Errorf("logs = %v, want %v", logs, want)
		}
	})
	t.Run("between disjoint siblings", func(t *testing.T) {
		h := newTrackerHarness()
		var logs []string
		mk := func(name string) *testAnnotationTarget {
			return &testAnnotationTarget{
				onEnter: func(*gestures.PointerEnterEvent) { logs = append(logs, "enter"+name) },
				onExit:  func(*gestures.PointerExitEvent) { logs = append(logs, "exit"+name) },
				onHover: func(*gestures.PointerHoverEvent) { logs = append(logs, "hover"+name) },
			}
		}
		a, b := mk("A"), mk("B")
		isInA, isInB := true, false
		h.find = func(ui.Offset) []testEntry {
			switch {
			case isInA:
				return []testEntry{{target: a}}
			case isInB:
				return []testEntry{{target: b}}
			}
			return nil
		}
		h.send(mouseEv("add", off(0, 1)))
		if want := []string{"enterA"}; !reflect.DeepEqual(logs, want) {
			t.Errorf("logs = %v, want %v", logs, want)
		}
		logs = nil
		isInA, isInB = false, true
		h.send(mouseEv("hover", off(0, 10)))
		if want := []string{"exitA", "enterB", "hoverB"}; !reflect.DeepEqual(logs, want) {
			t.Errorf("logs = %v, want %v", logs, want)
		}
		logs = nil
		isInA, isInB = true, false
		h.send(mouseEv("hover", off(0, 1)))
		if want := []string{"exitB", "enterA", "hoverA"}; !reflect.DeepEqual(logs, want) {
			t.Errorf("logs = %v, want %v", logs, want)
		}
	})
}

func TestMouseTrackerInvalidAnnotationIsSkipped(t *testing.T) {
	h := newTrackerHarness()
	entered := false
	invalid := services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOpts{
		OnEnter: func(*gestures.PointerEnterEvent) { entered = true }, Invalid: true,
	})
	h.tracker = NewMouseTracker(func(ui.Offset, ui.ViewID) *gestures.HitTestResult {
		r := gestures.NewHitTestResult()
		r.Add(gestures.NewHitTestEntry(annotationTarget{invalid}))
		return r
	}, services.MouseCursorUncontrolled)
	h.send(mouseEv("add", ui.Offset{}))
	if entered {
		t.Error("an invalid annotation was notified")
	}
}

// annotationTarget makes a services annotation a hit test target.
type annotationTarget struct {
	*services.MouseTrackerAnnotationData
}

func (annotationTarget) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}
