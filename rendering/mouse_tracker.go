package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// MouseTrackerHitTest hit tests a position of a view and returns the result.
//
// Flutter: rendering/mouse_tracker.dart MouseTrackerHitTest
type MouseTrackerHitTest func(offset ui.Offset, viewID ui.ViewID) *gestures.HitTestResult

// annotationMap maps the annotations under a pointer to their transforms and
// iterates in insertion order, like Dart's LinkedHashMap.
type annotationMap struct {
	keys       []services.MouseTrackerAnnotation
	transforms map[services.MouseTrackerAnnotation]*ui.Matrix4
}

func newAnnotationMap() *annotationMap {
	return &annotationMap{transforms: map[services.MouseTrackerAnnotation]*ui.Matrix4{}}
}

// set adds the annotation, or replaces its transform keeping its position.
func (m *annotationMap) set(a services.MouseTrackerAnnotation, t *ui.Matrix4) {
	if _, ok := m.transforms[a]; !ok {
		m.keys = append(m.keys, a)
	}
	m.transforms[a] = t
}

func (m *annotationMap) has(a services.MouseTrackerAnnotation) bool {
	_, ok := m.transforms[a]
	return ok
}

type mouseState struct {
	annotations *annotationMap
	latestEvent gestures.PointerEvent
}

func (s *mouseState) replaceAnnotations(v *annotationMap) *annotationMap {
	prev := s.annotations
	s.annotations = v
	return prev
}

func (s *mouseState) replaceLatestEvent(v gestures.PointerEvent) gestures.PointerEvent {
	if foundation.DebugMode && v.Device() != s.latestEvent.Device() {
		panic(errors.New("MouseTrackerDeviceChanged", "the device of a mouse state can not change",
			"was", s.latestEvent.Device(), "now", v.Device()))
	}
	prev := s.latestEvent
	s.latestEvent = v
	return prev
}

func (s *mouseState) device() int { return s.latestEvent.Device() }

type mouseTrackerUpdateDetails struct {
	lastAnnotations *annotationMap
	nextAnnotations *annotationMap
	previousEvent   gestures.PointerEvent
	triggeringEvent gestures.PointerEvent
}

func (d mouseTrackerUpdateDetails) device() int {
	if d.previousEvent != nil {
		return d.previousEvent.Device()
	}
	return d.triggeringEvent.Device()
}

func (d mouseTrackerUpdateDetails) latestEvent() gestures.PointerEvent {
	if d.triggeringEvent != nil {
		return d.triggeringEvent
	}
	return d.previousEvent
}

// MouseTracker tracks the annotations under each mouse pointer, and sends enter
// and exit events and the cursor changes to them. It notifies its listeners when
// a mouse is connected or disconnected.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker
type MouseTracker struct {
	foundation.ChangeNotifier
	hitTestInView           MouseTrackerHitTest
	mouseCursorManager      *services.MouseCursorManager
	mouseStates             map[int]*mouseState
	mouseOrder              []int
	debugDuringDeviceUpdate bool
}

// NewMouseTracker creates a tracker that hit tests with hitTestInView. The
// fallback cursor is the basic one.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker.new
func NewMouseTracker(hitTestInView MouseTrackerHitTest, fallbackCursor services.MouseCursor) *MouseTracker {
	return &MouseTracker{
		hitTestInView:      hitTestInView,
		mouseCursorManager: services.NewMouseCursorManager(fallbackCursor),
		mouseStates:        map[int]*mouseState{},
	}
}

// MouseIsConnected is whether any mouse is connected.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker.mouseIsConnected
func (t *MouseTracker) MouseIsConnected() bool { return len(t.mouseStates) > 0 }

func (t *MouseTracker) monitorMouseConnection(task func()) {
	was := t.MouseIsConnected()
	task()
	if was != t.MouseIsConnected() {
		t.NotifyListeners()
	}
}

func (t *MouseTracker) deviceUpdatePhase(task func()) {
	if foundation.DebugMode {
		if t.debugDuringDeviceUpdate {
			panic(errors.New("MouseTrackerReentered", "the mouse tracker was updated while it was updating"))
		}
		t.debugDuringDeviceUpdate = true
	}
	task()
	if foundation.DebugMode {
		t.debugDuringDeviceUpdate = false
	}
}

func shouldMarkStateDirty(state *mouseState, event gestures.PointerEvent) bool {
	if state == nil {
		return true
	}
	last := state.latestEvent
	if foundation.DebugMode {
		_, isAdded := event.(*gestures.PointerAddedEvent)
		_, lastRemoved := last.(*gestures.PointerRemovedEvent)
		if event.Device() != last.Device() || isAdded != lastRemoved {
			panic(errors.New("MouseTrackerEventOrder", "an added event must follow a removed event on the same device",
				"device", event.Device()))
		}
	}
	if _, ok := event.(gestures.PointerSignalEvent); ok {
		return false
	}
	_, lastAdded := last.(*gestures.PointerAddedEvent)
	_, removed := event.(*gestures.PointerRemovedEvent)
	return lastAdded || removed || last.Position() != event.Position()
}

func hitTestResultToAnnotations(result *gestures.HitTestResult) *annotationMap {
	m := newAnnotationMap()
	for _, entry := range result.Path() {
		if a, ok := entry.Target().(services.MouseTrackerAnnotation); ok {
			m.set(a, entry.Transform())
		}
	}
	return m
}

func (t *MouseTracker) findAnnotations(state *mouseState) *annotationMap {
	if _, ok := t.mouseStates[state.device()]; !ok {
		return newAnnotationMap()
	}
	return hitTestResultToAnnotations(t.hitTestInView(state.latestEvent.Position(), state.latestEvent.ViewID()))
}

func (t *MouseTracker) handleDeviceUpdate(d mouseTrackerUpdateDetails) {
	handleDeviceUpdateMouseEvents(d)
	cursors := make([]services.MouseCursor, 0, len(d.nextAnnotations.keys))
	for _, a := range d.nextAnnotations.keys {
		cursors = append(cursors, a.Cursor())
	}
	t.mouseCursorManager.HandleDeviceCursorUpdate(d.device(), d.triggeringEvent, cursors)
}

// UpdateWithEvent updates the annotations under the device of a mouse or stylus
// event. The hit test result is the one the event produced, or nil to hit test
// again. Other kinds of device and signal events are ignored.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker.updateWithEvent
func (t *MouseTracker) UpdateWithEvent(event gestures.PointerEvent, hitTestResult *gestures.HitTestResult) {
	if event.Kind() != ui.PointerDeviceKindMouse && event.Kind() != ui.PointerDeviceKindStylus {
		return
	}
	if _, ok := event.(gestures.PointerSignalEvent); ok {
		return
	}
	_, removedEvent := event.(*gestures.PointerRemovedEvent)
	var result *gestures.HitTestResult
	switch {
	case removedEvent:
		result = gestures.NewHitTestResult()
	case hitTestResult != nil:
		result = hitTestResult
	default:
		result = t.hitTestInView(event.Position(), event.ViewID())
	}
	device := event.Device()
	existing := t.mouseStates[device]
	if !shouldMarkStateDirty(existing, event) {
		return
	}
	t.monitorMouseConnection(func() {
		t.deviceUpdatePhase(func() {
			if existing == nil {
				if removedEvent {
					return
				}
				t.mouseStates[device] = &mouseState{annotations: newAnnotationMap(), latestEvent: event}
				t.mouseOrder = append(t.mouseOrder, device)
			} else {
				if removedEvent {
					delete(t.mouseStates, device)
					for i, d := range t.mouseOrder {
						if d == device {
							t.mouseOrder = append(t.mouseOrder[:i], t.mouseOrder[i+1:]...)
							break
						}
					}
				}
			}
			target := t.mouseStates[device]
			if target == nil {
				target = existing
			}
			last := target.replaceLatestEvent(event)
			next := newAnnotationMap()
			if !removedEvent {
				next = hitTestResultToAnnotations(result)
			}
			lastAnnotations := target.replaceAnnotations(next)
			t.handleDeviceUpdate(mouseTrackerUpdateDetails{
				lastAnnotations: lastAnnotations, nextAnnotations: next, previousEvent: last, triggeringEvent: event,
			})
		})
	})
}

// UpdateAllDevices hit tests again for every device, which sends the enter and
// exit events that a new frame caused. It is called after every frame.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker.updateAllDevices
func (t *MouseTracker) UpdateAllDevices() {
	t.deviceUpdatePhase(func() {
		for _, d := range append([]int(nil), t.mouseOrder...) {
			dirty := t.mouseStates[d]
			last := dirty.latestEvent
			next := t.findAnnotations(dirty)
			lastAnnotations := dirty.replaceAnnotations(next)
			t.handleDeviceUpdate(mouseTrackerUpdateDetails{
				lastAnnotations: lastAnnotations, nextAnnotations: next, previousEvent: last,
			})
		}
	})
}

// DebugDeviceActiveCursor is the cursor of the device; it is only available in
// debug mode.
//
// Flutter: rendering/mouse_tracker.dart MouseTracker.debugDeviceActiveCursor
func (t *MouseTracker) DebugDeviceActiveCursor(device int) (services.MouseCursor, bool) {
	return t.mouseCursorManager.DebugDeviceActiveCursor(device)
}

func handleDeviceUpdateMouseEvents(d mouseTrackerUpdateDetails) {
	latest := d.latestEvent()
	baseExit := gestures.NewPointerExitEventFromMouseEvent(latest)
	for _, a := range d.lastAnnotations.keys {
		if a.ValidForMouseTracker() && !d.nextAnnotations.has(a) {
			if f := a.OnExit(); f != nil {
				f(baseExit.Transformed(d.lastAnnotations.transforms[a]).(*gestures.PointerExitEvent))
			}
		}
	}
	var entering []services.MouseTrackerAnnotation
	for _, a := range d.nextAnnotations.keys {
		if !d.lastAnnotations.has(a) {
			entering = append(entering, a)
		}
	}
	baseEnter := gestures.NewPointerEnterEventFromMouseEvent(latest)
	for i := len(entering) - 1; i >= 0; i-- {
		a := entering[i]
		if a.ValidForMouseTracker() {
			if f := a.OnEnter(); f != nil {
				f(baseEnter.Transformed(d.nextAnnotations.transforms[a]).(*gestures.PointerEnterEvent))
			}
		}
	}
}
