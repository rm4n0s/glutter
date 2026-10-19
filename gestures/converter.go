package gestures

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// synthesiseDownButtons adds PrimaryButton to buttons when a pointer of
// certain devices is down.
func synthesiseDownButtons(buttons int, kind ui.PointerDeviceKind) int {
	switch kind {
	case ui.PointerDeviceKindMouse, ui.PointerDeviceKindTrackpad:
		return buttons
	}
	// Touch, stylus, inverted stylus and unknown devices: the pointer is
	// down, so the buttons are never 0.
	if buttons == 0 {
		return PrimaryButton
	}
	return buttons
}

// DevicePixelRatioGetter returns the device pixel ratio of the view, or None if
// the view does not exist (any more).
//
// Flutter: gestures/converter.dart DevicePixelRatioGetter
type DevicePixelRatioGetter func(viewID ui.ViewID) foundation.Opt[float64]

// PointerEventConverterExpand converts the pointer data of the engine, which
// is in physical pixels, to pointer events in logical pixels. Data of a view
// that does not exist, data of an unknown kind of signal and a scroll with a
// delta that is not finite are dropped.
//
// Flutter: gestures/converter.dart PointerEventConverter.expand
func PointerEventConverterExpand(data []ui.PointerData, devicePixelRatioForView DevicePixelRatioGetter) []PointerEvent {
	events := make([]PointerEvent, 0, len(data))
	for i := range data {
		datum := &data[i]
		if datum.SignalKind == ui.PointerSignalKindUnknown {
			continue
		}
		dpr, ok := devicePixelRatioForView(datum.ViewID).Get()
		if !ok {
			continue // The view doesn't exist anymore.
		}
		if e := convertPointerData(datum, dpr); e != nil {
			events = append(events, e)
		}
	}
	return events
}

func convertPointerData(datum *ui.PointerData, dpr float64) PointerEvent {
	position := ui.Offset{Dx: datum.PhysicalX, Dy: datum.PhysicalY}.Div(dpr)
	delta := ui.Offset{Dx: datum.PhysicalDeltaX, Dy: datum.PhysicalDeltaY}.Div(dpr)
	radiusMinor := datum.RadiusMinor / dpr
	radiusMajor := datum.RadiusMajor / dpr
	radiusMin := datum.RadiusMin / dpr
	radiusMax := datum.RadiusMax / dpr
	kind := datum.Kind
	some := foundation.Some[float64]
	base := PointerEventOpts{
		ViewID:     datum.ViewID,
		TimeStamp:  datum.TimeStamp,
		Kind:       kind,
		Device:     datum.Device,
		Position:   position,
		EmbedderID: datum.EmbedderID,
	}
	switch datum.SignalKind {
	case ui.PointerSignalKindScroll:
		if math.IsInf(datum.ScrollDeltaX, 0) || math.IsNaN(datum.ScrollDeltaX) ||
			math.IsInf(datum.ScrollDeltaY, 0) || math.IsNaN(datum.ScrollDeltaY) || dpr <= 0 {
			return nil
		}
		o := base
		o.ScrollDelta = ui.Offset{Dx: datum.ScrollDeltaX, Dy: datum.ScrollDeltaY}.Div(dpr)
		o.OnRespond = datum.Respond
		return NewPointerScrollEvent(o)
	case ui.PointerSignalKindScrollInertiaCancel:
		return NewPointerScrollInertiaCancelEvent(base)
	case ui.PointerSignalKindScale:
		o := base
		o.Scale = some(datum.Scale)
		return NewPointerScaleEvent(o)
	}

	// The data is not a signal.
	full := base
	full.Delta = delta
	full.Obscured = datum.Obscured
	full.PressureMin = some(datum.PressureMin)
	full.PressureMax = some(datum.PressureMax)
	full.Distance = datum.Distance
	full.DistanceMax = datum.DistanceMax
	full.Size = datum.Size
	full.RadiusMajor = radiusMajor
	full.RadiusMinor = radiusMinor
	full.RadiusMin = radiusMin
	full.RadiusMax = radiusMax
	full.Orientation = datum.Orientation
	full.Tilt = datum.Tilt
	full.Synthesized = datum.Synthesized
	full.Pointer = datum.PointerIdentifier

	switch datum.Change {
	case ui.PointerChangeAdd:
		return NewPointerAddedEvent(full)
	case ui.PointerChangeHover:
		full.Buttons = foundation.Some(datum.Buttons)
		return NewPointerHoverEvent(full)
	case ui.PointerChangeDown:
		full.Buttons = foundation.Some(synthesiseDownButtons(datum.Buttons, kind))
		full.Pressure = some(datum.Pressure)
		return NewPointerDownEvent(full)
	case ui.PointerChangeMove:
		full.Buttons = foundation.Some(synthesiseDownButtons(datum.Buttons, kind))
		full.Pressure = some(datum.Pressure)
		full.PlatformData = datum.PlatformData
		return NewPointerMoveEvent(full)
	case ui.PointerChangeUp:
		full.Buttons = foundation.Some(datum.Buttons)
		full.Pressure = some(datum.Pressure)
		return NewPointerUpEvent(full)
	case ui.PointerChangeCancel, ui.PointerChangeUnspecified:
		// The zero value stands for the default of PointerData.change in
		// Flutter, which is cancel.
		full.Buttons = foundation.Some(datum.Buttons)
		return NewPointerCancelEvent(full)
	case ui.PointerChangeRemove:
		return NewPointerRemovedEvent(full)
	case ui.PointerChangePanZoomStart:
		return NewPointerPanZoomStartEvent(full)
	case ui.PointerChangePanZoomUpdate:
		full.Pan = ui.Offset{Dx: datum.PanX, Dy: datum.PanY}.Div(dpr)
		full.PanDelta = ui.Offset{Dx: datum.PanDeltaX, Dy: datum.PanDeltaY}.Div(dpr)
		full.Scale = some(datum.Scale)
		full.Rotation = datum.Rotation
		return NewPointerPanZoomUpdateEvent(full)
	case ui.PointerChangePanZoomEnd:
		return NewPointerPanZoomEndEvent(full)
	}
	return nil
}
