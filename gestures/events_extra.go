package gestures

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollDelta is how far the pointer scrolls, in logical pixels.
//
// Flutter: gestures/events.dart PointerScrollEvent.scrollDelta
func (e *PointerScrollEvent) ScrollDelta() ui.Offset { return e.scrollDelta }

// DebugFillProperties adds the properties of the event and its scroll delta.
//
// Flutter: gestures/events.dart PointerScrollEvent.debugFillProperties
func (e *PointerScrollEvent) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	e.eventData.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("scrollDelta", e.scrollDelta))
}

// Scale is the scale factor of the signal.
//
// Flutter: gestures/events.dart PointerScaleEvent.scale
func (e *PointerScaleEvent) Scale() float64 { return e.scale }

// Pan is how far the fingers moved since the gesture started, in the
// coordinates of the view.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.pan
func (e *PointerPanZoomUpdateEvent) Pan() ui.Offset { return e.pan }

// LocalPan is Pan in the coordinates of the target.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.localPan
func (e *PointerPanZoomUpdateEvent) LocalPan() ui.Offset { return e.localPan }

// PanDelta is the change of Pan since the last update.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.panDelta
func (e *PointerPanZoomUpdateEvent) PanDelta() ui.Offset { return e.panDelta }

// LocalPanDelta is PanDelta in the coordinates of the target.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.localPanDelta
func (e *PointerPanZoomUpdateEvent) LocalPanDelta() ui.Offset { return e.localPanDlt }

// Scale is the scale factor since the gesture started.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.scale
func (e *PointerPanZoomUpdateEvent) Scale() float64 { return e.scale }

// Rotation is the rotation since the gesture started, in radians.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.rotation
func (e *PointerPanZoomUpdateEvent) Rotation() float64 { return e.rotation }

// mouseEventOpts returns the options that make an enter or exit event from the
// hover or move event, the way fromMouseEvent does.
func mouseEventOpts(event PointerEvent) PointerEventOpts {
	return PointerEventOpts{
		ViewID:      event.ViewID(),
		TimeStamp:   event.TimeStamp(),
		Pointer:     event.Pointer(),
		Kind:        event.Kind(),
		Device:      event.Device(),
		Position:    event.Position(),
		Delta:       event.Delta(),
		Buttons:     foundation.Some(event.Buttons()),
		Obscured:    event.Obscured(),
		PressureMin: foundation.Some(event.PressureMin()),
		PressureMax: foundation.Some(event.PressureMax()),
		Distance:    event.Distance(),
		DistanceMax: event.DistanceMax(),
		Size:        event.Size(),
		RadiusMajor: event.RadiusMajor(),
		RadiusMinor: event.RadiusMinor(),
		RadiusMin:   event.RadiusMin(),
		RadiusMax:   event.RadiusMax(),
		Orientation: event.Orientation(),
		Tilt:        event.Tilt(),
		Down:        event.Down(),
		Synthesized: event.Synthesized(),
	}
}

// NewPointerEnterEventFromMouseEvent creates the enter event that matches a
// hover or move event, with the same transform.
//
// Flutter: gestures/events.dart PointerEnterEvent.fromMouseEvent
func NewPointerEnterEventFromMouseEvent(event PointerEvent) *PointerEnterEvent {
	return transformEvent(NewPointerEnterEvent(mouseEventOpts(event)), event.Transform())
}

// NewPointerExitEventFromMouseEvent creates the exit event that matches a
// hover or move event, with the same transform.
//
// Flutter: gestures/events.dart PointerExitEvent.fromMouseEvent
func NewPointerExitEventFromMouseEvent(event PointerEvent) *PointerExitEvent {
	return transformEvent(NewPointerExitEvent(mouseEventOpts(event)), event.Transform())
}
