package ui

import (
	"fmt"
	"time"
)

// PointerChange is how the pointer has changed since the last report. The zero
// value, PointerChangeUnspecified, is never produced by a platform backend.
//
// Flutter: ui/pointer.dart PointerChange
type PointerChange int

// The values of PointerChange follow Flutter's declaration order.
const (
	// PointerChangeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	PointerChangeUnspecified PointerChange = iota
	// PointerChangeCancel means the input from the pointer is no longer
	// directed towards this receiver.
	//
	// Flutter: ui/pointer.dart PointerChange.cancel
	PointerChangeCancel
	// PointerChangeAdd means the device has started tracking the pointer.
	//
	// Flutter: ui/pointer.dart PointerChange.add
	PointerChangeAdd
	// PointerChangeRemove means the device is no longer tracking the pointer.
	//
	// Flutter: ui/pointer.dart PointerChange.remove
	PointerChangeRemove
	// PointerChangeHover means the pointer moved while it was not in contact
	// with the device.
	//
	// Flutter: ui/pointer.dart PointerChange.hover
	PointerChangeHover
	// PointerChangeDown means the pointer has made contact with the device.
	//
	// Flutter: ui/pointer.dart PointerChange.down
	PointerChangeDown
	// PointerChangeMove means the pointer moved while in contact with the
	// device.
	//
	// Flutter: ui/pointer.dart PointerChange.move
	PointerChangeMove
	// PointerChangeUp means the pointer stopped making contact with the device.
	//
	// Flutter: ui/pointer.dart PointerChange.up
	PointerChangeUp
	// PointerChangePanZoomStart means a pan/zoom gesture started on a
	// trackpad.
	//
	// Flutter: ui/pointer.dart PointerChange.panZoomStart
	PointerChangePanZoomStart
	// PointerChangePanZoomUpdate means a pan/zoom gesture on a trackpad
	// changed.
	//
	// Flutter: ui/pointer.dart PointerChange.panZoomUpdate
	PointerChangePanZoomUpdate
	// PointerChangePanZoomEnd means a pan/zoom gesture on a trackpad ended.
	//
	// Flutter: ui/pointer.dart PointerChange.panZoomEnd
	PointerChangePanZoomEnd
)

var pointerChangeNames = [...]string{
	"PointerChangeUnspecified",
	"PointerChangeCancel",
	"PointerChangeAdd",
	"PointerChangeRemove",
	"PointerChangeHover",
	"PointerChangeDown",
	"PointerChangeMove",
	"PointerChangeUp",
	"PointerChangePanZoomStart",
	"PointerChangePanZoomUpdate",
	"PointerChangePanZoomEnd",
}

// String returns the name of the change, such as "PointerChangeDown".
//
// Flutter: ui/pointer.dart PointerChange
func (c PointerChange) String() string {
	return enumName(pointerChangeNames[:], int(c), "PointerChange")
}

// PointerDeviceKind is the kind of pointer device. The zero value,
// PointerDeviceKindUnspecified, is never produced by a platform backend.
//
// Flutter: ui/pointer.dart PointerDeviceKind
type PointerDeviceKind int

// The values of PointerDeviceKind follow Flutter's declaration order.
const (
	// PointerDeviceKindUnspecified is the zero value.
	//
	// Flutter: no counterpart
	PointerDeviceKindUnspecified PointerDeviceKind = iota
	// PointerDeviceKindTouch is a touch screen contact.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.touch
	PointerDeviceKindTouch
	// PointerDeviceKindMouse is a mouse.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.mouse
	PointerDeviceKindMouse
	// PointerDeviceKindStylus is a stylus.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.stylus
	PointerDeviceKindStylus
	// PointerDeviceKindInvertedStylus is the eraser end of a stylus.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.invertedStylus
	PointerDeviceKindInvertedStylus
	// PointerDeviceKindTrackpad is a trackpad.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.trackpad
	PointerDeviceKindTrackpad
	// PointerDeviceKindUnknown is a device of an unknown kind.
	//
	// Flutter: ui/pointer.dart PointerDeviceKind.unknown
	PointerDeviceKindUnknown
)

var pointerDeviceKindNames = [...]string{
	"PointerDeviceKindUnspecified",
	"PointerDeviceKindTouch",
	"PointerDeviceKindMouse",
	"PointerDeviceKindStylus",
	"PointerDeviceKindInvertedStylus",
	"PointerDeviceKindTrackpad",
	"PointerDeviceKindUnknown",
}

// String returns the name of the kind, such as "PointerDeviceKindMouse".
//
// Flutter: ui/pointer.dart PointerDeviceKind
func (k PointerDeviceKind) String() string {
	return enumName(pointerDeviceKindNames[:], int(k), "PointerDeviceKind")
}

// PointerSignalKind is the kind of pointer signal event. The zero value,
// PointerSignalKindUnspecified, is Flutter's null signalKind: the data is not
// a signal.
//
// Flutter: ui/pointer.dart PointerSignalKind
type PointerSignalKind int

// The values of PointerSignalKind follow Flutter's declaration order.
const (
	// PointerSignalKindUnspecified is the zero value. It stands for the null
	// signalKind of PointerData.
	//
	// Flutter: ui/pointer.dart PointerData.signalKind (null)
	PointerSignalKindUnspecified PointerSignalKind = iota
	// PointerSignalKindNone is the type of a pointer signal event that is not
	// a signal.
	//
	// Flutter: ui/pointer.dart PointerSignalKind.none
	PointerSignalKindNone
	// PointerSignalKindScroll is a scroll signal.
	//
	// Flutter: ui/pointer.dart PointerSignalKind.scroll
	PointerSignalKindScroll
	// PointerSignalKindScrollInertiaCancel is a signal that cancels a scroll
	// that was still moving by inertia.
	//
	// Flutter: ui/pointer.dart PointerSignalKind.scrollInertiaCancel
	PointerSignalKindScrollInertiaCancel
	// PointerSignalKindScale is a scale signal.
	//
	// Flutter: ui/pointer.dart PointerSignalKind.scale
	PointerSignalKindScale
	// PointerSignalKindUnknown is a signal of an unknown kind.
	//
	// Flutter: ui/pointer.dart PointerSignalKind.unknown
	PointerSignalKindUnknown
)

var pointerSignalKindNames = [...]string{
	"PointerSignalKindUnspecified",
	"PointerSignalKindNone",
	"PointerSignalKindScroll",
	"PointerSignalKindScrollInertiaCancel",
	"PointerSignalKindScale",
	"PointerSignalKindUnknown",
}

// String returns the name of the kind, such as "PointerSignalKindScroll".
//
// Flutter: ui/pointer.dart PointerSignalKind
func (k PointerSignalKind) String() string {
	return enumName(pointerSignalKindNames[:], int(k), "PointerSignalKind")
}

// PointerData is information about the state of a pointer. All positions and
// deltas are in physical pixels of the view.
//
// Flutter's defaults are the zero values, except that Change defaults to
// cancel and Kind to touch; platform backends always set both explicitly.
//
// Flutter: ui/pointer.dart PointerData
type PointerData struct {
	// ViewID is the ID of the view this event is for.
	ViewID ViewID
	// EmbedderID is the ID of the pointer event as assigned by the embedder.
	EmbedderID int
	// TimeStamp is the time at which this event happened, since an arbitrary
	// fixed point in the past.
	TimeStamp time.Duration
	// Change is how the pointer has changed since the last report.
	Change PointerChange
	// Kind is the kind of input device for which the event was generated.
	Kind PointerDeviceKind
	// SignalKind is the kind of signal, or PointerSignalKindUnspecified when
	// the data is not a signal.
	SignalKind PointerSignalKind
	// Device is the unique identifier of the device.
	Device int
	// PointerIdentifier identifies the pointer within its device while it is
	// in contact.
	PointerIdentifier int
	// PhysicalX is the x coordinate of the pointer, in physical pixels.
	PhysicalX float64
	// PhysicalY is the y coordinate of the pointer, in physical pixels.
	PhysicalY float64
	// PhysicalDeltaX is the distance in physical pixels that the pointer moved
	// along x since the last report.
	PhysicalDeltaX float64
	// PhysicalDeltaY is the distance in physical pixels that the pointer moved
	// along y since the last report.
	PhysicalDeltaY float64
	// Buttons is a bit field of the buttons that are currently pressed.
	Buttons int
	// Obscured is whether the input is obscured by another window.
	Obscured bool
	// Synthesized is whether the event was created by the engine to keep the
	// state consistent, rather than by a device.
	Synthesized bool
	// Pressure is the pressure of the touch, normalized by the range between
	// PressureMin and PressureMax.
	Pressure float64
	// PressureMin is the minimum value that Pressure can return.
	PressureMin float64
	// PressureMax is the maximum value that Pressure can return.
	PressureMax float64
	// Distance is the distance of the detected object from the input surface.
	Distance float64
	// DistanceMax is the maximum value that Distance can return.
	DistanceMax float64
	// Size is the area of the screen being pressed, scaled to a value between
	// 0 and 1.
	Size float64
	// RadiusMajor is the radius of the contact ellipse along the major axis,
	// in logical pixels.
	RadiusMajor float64
	// RadiusMinor is the radius of the contact ellipse along the minor axis,
	// in logical pixels.
	RadiusMinor float64
	// RadiusMin is the minimum value that RadiusMajor and RadiusMinor can
	// return.
	RadiusMin float64
	// RadiusMax is the maximum value that RadiusMajor and RadiusMinor can
	// return.
	RadiusMax float64
	// Orientation is the orientation angle of the detected object, in radians.
	Orientation float64
	// Tilt is the tilt angle of the detected object, in radians.
	Tilt float64
	// PlatformData is opaque platform-specific data associated with the event.
	PlatformData int
	// ScrollDeltaX is the amount to scroll in the x direction, in physical
	// pixels. Only set for scroll signals.
	ScrollDeltaX float64
	// ScrollDeltaY is the amount to scroll in the y direction, in physical
	// pixels. Only set for scroll signals.
	ScrollDeltaY float64
	// PanX is the total horizontal translation of a pan/zoom gesture.
	PanX float64
	// PanY is the total vertical translation of a pan/zoom gesture.
	PanY float64
	// PanDeltaX is the horizontal translation since the last pan/zoom report.
	PanDeltaX float64
	// PanDeltaY is the vertical translation since the last pan/zoom report.
	PanDeltaY float64
	// Scale is the scale of a pan/zoom gesture, 1 meaning no scaling.
	Scale float64
	// Rotation is the rotation of a pan/zoom gesture, in radians.
	Rotation float64
	// OnRespond, when not nil, lets the framework answer the event that
	// produced this data. See Respond.
	OnRespond func(allowPlatformDefault bool)
}

// Respond tells the platform whether it may run its default behavior for the
// event that produced this data (for example, scrolling the page on the web).
// It does nothing when OnRespond is nil.
//
// Flutter: ui/pointer.dart PointerData.respond
func (d PointerData) Respond(allowPlatformDefault bool) {
	if d.OnRespond != nil {
		d.OnRespond(allowPlatformDefault)
	}
}

// String returns a short description of the data.
//
// Flutter: ui/pointer.dart PointerData.toString
func (d PointerData) String() string {
	return fmt.Sprintf("PointerData(viewId: %d, x: %v, y: %v)", d.ViewID, d.PhysicalX, d.PhysicalY)
}

// PointerDataPacket is a sequence of reports about the state of pointers.
//
// Flutter: ui/pointer.dart PointerDataPacket
type PointerDataPacket struct {
	// Data is the pointer reports in this packet, in order.
	Data []PointerData
}
