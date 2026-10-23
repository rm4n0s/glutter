package gestures

import (
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// The bit masks of PointerEvent.Buttons.
//
// Flutter: gestures/events.dart kPrimaryButton and the constants after it
const (
	// PrimaryButton is the bit of the primary button: the left mouse button,
	// the stylus tip or a touch contact.
	//
	// Flutter: gestures/events.dart kPrimaryButton
	PrimaryButton = 0x01
	// SecondaryButton is the bit of the secondary button: the right mouse
	// button or the stylus barrel button.
	//
	// Flutter: gestures/events.dart kSecondaryButton
	SecondaryButton = 0x02
	// TertiaryButton is the bit of the tertiary button: the middle mouse
	// button.
	//
	// Flutter: gestures/events.dart kTertiaryButton
	TertiaryButton = 0x04
	// PrimaryMouseButton is PrimaryButton.
	//
	// Flutter: gestures/events.dart kPrimaryMouseButton
	PrimaryMouseButton = PrimaryButton
	// SecondaryMouseButton is SecondaryButton.
	//
	// Flutter: gestures/events.dart kSecondaryMouseButton
	SecondaryMouseButton = SecondaryButton
	// PrimaryStylusButton is the bit of the stylus tip when it presses on the
	// screen.
	//
	// Flutter: gestures/events.dart kPrimaryStylusButton
	PrimaryStylusButton = SecondaryButton
	// StylusContact is PrimaryButton.
	//
	// Flutter: gestures/events.dart kStylusContact
	StylusContact = PrimaryButton
	// MiddleMouseButton is TertiaryButton.
	//
	// Flutter: gestures/events.dart kMiddleMouseButton
	MiddleMouseButton = TertiaryButton
	// SecondaryStylusButton is TertiaryButton.
	//
	// Flutter: gestures/events.dart kSecondaryStylusButton
	SecondaryStylusButton = TertiaryButton
	// BackMouseButton is the bit of the back button of a mouse.
	//
	// Flutter: gestures/events.dart kBackMouseButton
	BackMouseButton = 0x08
	// ForwardMouseButton is the bit of the forward button of a mouse.
	//
	// Flutter: gestures/events.dart kForwardMouseButton
	ForwardMouseButton = 0x10
	// TouchContact is PrimaryButton.
	//
	// Flutter: gestures/events.dart kTouchContact
	TouchContact = PrimaryButton

	// maxUnsignedSMI is the largest unsigned small integer of a 64-bit Dart VM.
	maxUnsignedSMI = 0x3FFFFFFFFFFFFFFF
)

// NthMouseButton returns the bit of the mouse button with the number (1 is the
// primary button).
//
// Flutter: gestures/events.dart nthMouseButton
func NthMouseButton(number int) int {
	return (PrimaryMouseButton << (number - 1)) & maxUnsignedSMI
}

// NthStylusButton returns the bit of the stylus button with the number (1 is
// PrimaryStylusButton).
//
// Flutter: gestures/events.dart nthStylusButton
func NthStylusButton(number int) int {
	return (PrimaryStylusButton << (number - 1)) & maxUnsignedSMI
}

// SmallestButton returns the lowest set bit of buttons: the smallest button.
//
// Flutter: gestures/events.dart smallestButton
func SmallestButton(buttons int) int { return buttons & -buttons }

// IsSingleButton reports whether exactly one button is set in buttons.
//
// Flutter: gestures/events.dart isSingleButton
func IsSingleButton(buttons int) bool {
	return buttons != 0 && SmallestButton(buttons) == buttons
}

// PointerEvent is the base of all pointer events: a report of where a pointer
// is and what it does. Pointer events are immutable. Each concrete event is a
// pointer to a struct named after Flutter's class (*PointerDownEvent, ...);
// use a type switch where Dart uses is.
//
// An event has a position in the coordinates of its view (Position) and one in
// the coordinates of the target it is delivered to (LocalPosition). Transformed
// makes the second kind from the first.
//
// Deviation: Flutter has a class per event and constructor parameters that
// differ per class; Glutter shares one options struct, PointerEventOpts, and a
// constructor ignores the fields its Flutter counterpart does not have (SPEC
// §12.10).
//
// Flutter: gestures/events.dart PointerEvent
type PointerEvent interface {
	foundation.Diagnosticable

	// ViewID is the ID of the view the event happened in.
	//
	// Flutter: gestures/events.dart PointerEvent.viewId
	ViewID() ui.ViewID
	// EmbedderID is the ID the embedder gave the pointer; it is 0 if it did
	// not.
	//
	// Flutter: gestures/events.dart PointerEvent.embedderId
	EmbedderID() int
	// TimeStamp is the time since the platform's start-up of the event.
	//
	// Flutter: gestures/events.dart PointerEvent.timeStamp
	TimeStamp() time.Duration
	// Pointer is the unique identifier of the pointer, from its down event to
	// its up or cancel event.
	//
	// Flutter: gestures/events.dart PointerEvent.pointer
	Pointer() int
	// Kind is the kind of device that made the event.
	//
	// Flutter: gestures/events.dart PointerEvent.kind
	Kind() ui.PointerDeviceKind
	// Device is the ID of the device; it is unique for a kind.
	//
	// Flutter: gestures/events.dart PointerEvent.device
	Device() int
	// Position is where the pointer is, in logical pixels of the view.
	//
	// Flutter: gestures/events.dart PointerEvent.position
	Position() ui.Offset
	// LocalPosition is Position in the coordinates of the target the event is
	// delivered to.
	//
	// Flutter: gestures/events.dart PointerEvent.localPosition
	LocalPosition() ui.Offset
	// Delta is how far the pointer moved since the last event, in the view's
	// coordinates.
	//
	// Flutter: gestures/events.dart PointerEvent.delta
	Delta() ui.Offset
	// LocalDelta is Delta in the coordinates of the target.
	//
	// Flutter: gestures/events.dart PointerEvent.localDelta
	LocalDelta() ui.Offset
	// Buttons is the set of buttons pressed, as the bits of PrimaryButton and
	// the constants beside it.
	//
	// Flutter: gestures/events.dart PointerEvent.buttons
	Buttons() int
	// Down reports whether the pointer is in contact with the device.
	//
	// Flutter: gestures/events.dart PointerEvent.down
	Down() bool
	// Obscured reports whether the input is obscured by another window.
	//
	// Flutter: gestures/events.dart PointerEvent.obscured
	Obscured() bool
	// Pressure is the pressure of the contact, between PressureMin and
	// PressureMax.
	//
	// Flutter: gestures/events.dart PointerEvent.pressure
	Pressure() float64
	// PressureMin is the minimum value Pressure can have.
	//
	// Flutter: gestures/events.dart PointerEvent.pressureMin
	PressureMin() float64
	// PressureMax is the maximum value Pressure can have.
	//
	// Flutter: gestures/events.dart PointerEvent.pressureMax
	PressureMax() float64
	// Distance is the distance of a hovering pointer from the screen.
	//
	// Flutter: gestures/events.dart PointerEvent.distance
	Distance() float64
	// DistanceMin is the minimum value Distance can have; it is always 0.
	//
	// Flutter: gestures/events.dart PointerEvent.distanceMin
	DistanceMin() float64
	// DistanceMax is the maximum value Distance can have.
	//
	// Flutter: gestures/events.dart PointerEvent.distanceMax
	DistanceMax() float64
	// Size is the area of the contact, normalized to the device.
	//
	// Flutter: gestures/events.dart PointerEvent.size
	Size() float64
	// RadiusMajor is the radius of the major axis of the contact ellipse.
	//
	// Flutter: gestures/events.dart PointerEvent.radiusMajor
	RadiusMajor() float64
	// RadiusMinor is the radius of the minor axis of the contact ellipse.
	//
	// Flutter: gestures/events.dart PointerEvent.radiusMinor
	RadiusMinor() float64
	// RadiusMin is the minimum value the radii can have.
	//
	// Flutter: gestures/events.dart PointerEvent.radiusMin
	RadiusMin() float64
	// RadiusMax is the maximum value the radii can have.
	//
	// Flutter: gestures/events.dart PointerEvent.radiusMax
	RadiusMax() float64
	// Orientation is the angle of the contact ellipse, in radians.
	//
	// Flutter: gestures/events.dart PointerEvent.orientation
	Orientation() float64
	// Tilt is the angle between the stylus and the surface normal, in radians.
	//
	// Flutter: gestures/events.dart PointerEvent.tilt
	Tilt() float64
	// PlatformData is opaque data from the platform.
	//
	// Flutter: gestures/events.dart PointerEvent.platformData
	PlatformData() int
	// Synthesized reports whether the event was made by the framework or the
	// engine, not by a device.
	//
	// Flutter: gestures/events.dart PointerEvent.synthesized
	Synthesized() bool
	// Transform is the matrix that took Position to LocalPosition; nil if the
	// event was not transformed.
	//
	// Flutter: gestures/events.dart PointerEvent.transform
	Transform() *ui.Matrix4
	// Original is the event this one was transformed from; nil if it is the
	// original.
	//
	// Flutter: gestures/events.dart PointerEvent.original
	Original() PointerEvent
	// Transformed returns the event with LocalPosition and LocalDelta in the
	// coordinates that transform leads to. Transforming a transformed event
	// starts from the original. A nil transform returns the original.
	//
	// Flutter: gestures/events.dart PointerEvent.transformed
	Transformed(transform *ui.Matrix4) PointerEvent
	// CopyWith returns an event of the same type with the given fields
	// replaced. As in Flutter, the copy has only the fields the event's
	// copyWith lists; the others get their defaults.
	//
	// Flutter: gestures/events.dart PointerEvent.copyWith
	CopyWith(c PointerEventCopyOpts) PointerEvent
	// String describes the event on one line.
	//
	// Flutter: gestures/events.dart PointerEvent.toString
	String() string

	data() *eventData
}

// PointerSignalEvent is an event that happens without the pointer being down:
// a scroll or a scale. Only one object may handle it, which it claims with
// PointerSignalResolver.
//
// Flutter: gestures/events.dart PointerSignalEvent
type PointerSignalEvent interface {
	PointerEvent
	// Respond tells the platform whether it may do its default action for the
	// event, after the framework has had its turn.
	//
	// Flutter: gestures/events.dart PointerSignalEvent.respond
	Respond(allowPlatformDefault bool)

	isSignal()
}

// PointerEventOpts are the optional parameters of the constructors of the
// events. A constructor reads the fields that the Flutter constructor of the
// same event has, and ignores the others.
//
// Flutter: gestures/events.dart PointerEvent.new
type PointerEventOpts struct {
	// ViewID is the view of the event; the default is 0.
	ViewID ui.ViewID
	// EmbedderID is the ID the embedder gave the pointer.
	EmbedderID int
	// TimeStamp is the time of the event.
	TimeStamp time.Duration
	// Pointer is the ID of the pointer.
	Pointer int
	// Kind is the kind of device; the default is touch (mouse for the signal
	// events, trackpad for the pan/zoom events).
	Kind ui.PointerDeviceKind
	// Device is the ID of the device.
	Device int
	// Position is where the pointer is.
	Position ui.Offset
	// Delta is how far the pointer moved.
	Delta ui.Offset
	// Buttons are the buttons pressed; the default is PrimaryButton for a
	// down and a move event, and 0 for the others.
	Buttons foundation.Opt[int]
	// Down is whether the pointer is in contact; only enter and exit events
	// take it.
	Down bool
	// Obscured is whether the input is obscured.
	Obscured bool
	// Pressure is the pressure; the default is 1 for a down and a move event,
	// and 0 for an up event.
	Pressure foundation.Opt[float64]
	// PressureMin is the least pressure; the default is 1.
	PressureMin foundation.Opt[float64]
	// PressureMax is the greatest pressure; the default is 1.
	PressureMax foundation.Opt[float64]
	// Distance is the distance from the screen.
	Distance float64
	// DistanceMax is the greatest distance.
	DistanceMax float64
	// Size is the area of the contact.
	Size float64
	// RadiusMajor is the major radius of the contact.
	RadiusMajor float64
	// RadiusMinor is the minor radius of the contact.
	RadiusMinor float64
	// RadiusMin is the least radius.
	RadiusMin float64
	// RadiusMax is the greatest radius.
	RadiusMax float64
	// Orientation is the angle of the contact.
	Orientation float64
	// Tilt is the tilt of the stylus.
	Tilt float64
	// PlatformData is opaque data from the platform.
	PlatformData int
	// Synthesized is whether the framework or the engine made the event.
	Synthesized bool
	// ScrollDelta is how far a scroll event scrolls, in logical pixels.
	ScrollDelta ui.Offset
	// Scale is the scale of a scale or a pan/zoom update event; the default
	// is 1.
	Scale foundation.Opt[float64]
	// Pan is the pan of a pan/zoom update event, from where it started.
	Pan ui.Offset
	// PanDelta is the change of Pan since the last pan/zoom update event.
	PanDelta ui.Offset
	// Rotation is the rotation of a pan/zoom update event, in radians.
	Rotation float64
	// OnRespond is called by a scroll event's Respond.
	OnRespond func(allowPlatformDefault bool)
}

// PointerEventCopyOpts are the replacements CopyWith takes. A field that is
// None keeps the event's own value.
//
// Flutter: gestures/events.dart PointerEvent.copyWith
type PointerEventCopyOpts struct {
	// ViewID replaces the view.
	ViewID foundation.Opt[ui.ViewID]
	// TimeStamp replaces the time.
	TimeStamp foundation.Opt[time.Duration]
	// Pointer replaces the pointer ID.
	Pointer foundation.Opt[int]
	// Kind replaces the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// Device replaces the device ID.
	Device foundation.Opt[int]
	// Position replaces the position.
	Position foundation.Opt[ui.Offset]
	// Delta replaces the delta.
	Delta foundation.Opt[ui.Offset]
	// Buttons replaces the buttons.
	Buttons foundation.Opt[int]
	// Obscured replaces the obscured flag.
	Obscured foundation.Opt[bool]
	// Pressure replaces the pressure.
	Pressure foundation.Opt[float64]
	// PressureMin replaces the least pressure.
	PressureMin foundation.Opt[float64]
	// PressureMax replaces the greatest pressure.
	PressureMax foundation.Opt[float64]
	// Distance replaces the distance.
	Distance foundation.Opt[float64]
	// DistanceMax replaces the greatest distance.
	DistanceMax foundation.Opt[float64]
	// Size replaces the size.
	Size foundation.Opt[float64]
	// RadiusMajor replaces the major radius.
	RadiusMajor foundation.Opt[float64]
	// RadiusMinor replaces the minor radius.
	RadiusMinor foundation.Opt[float64]
	// RadiusMin replaces the least radius.
	RadiusMin foundation.Opt[float64]
	// RadiusMax replaces the greatest radius.
	RadiusMax foundation.Opt[float64]
	// Orientation replaces the orientation.
	Orientation foundation.Opt[float64]
	// Tilt replaces the tilt.
	Tilt foundation.Opt[float64]
	// Synthesized replaces the synthesized flag.
	Synthesized foundation.Opt[bool]
	// EmbedderID replaces the embedder's ID.
	EmbedderID foundation.Opt[int]
	// OnRespond replaces the callback of a scroll event; the default is the
	// event's own Respond.
	OnRespond func(allowPlatformDefault bool)
}

// fieldSet is a set of the fields of PointerEventOpts.
type fieldSet uint32

const (
	fViewID fieldSet = 1 << iota
	fEmbedderID
	fTimeStamp
	fPointer
	fKind
	fDevice
	fPosition
	fDelta
	fButtons
	fDown
	fObscured
	fPressure
	fPressureMin
	fPressureMax
	fDistance
	fDistanceMax
	fSize
	fRadiusMajor
	fRadiusMinor
	fRadiusMin
	fRadiusMax
	fOrientation
	fTilt
	fPlatformData
	fSynthesized
	fScrollDelta
	fScale
	fPan
	fPanDelta
	fRotation
	fOnRespond
)

// masked returns the options with every field outside the set cleared.
func (o PointerEventOpts) masked(set fieldSet) PointerEventOpts {
	var r PointerEventOpts
	keep := func(f fieldSet) bool { return set&f != 0 }
	if keep(fViewID) {
		r.ViewID = o.ViewID
	}
	if keep(fEmbedderID) {
		r.EmbedderID = o.EmbedderID
	}
	if keep(fTimeStamp) {
		r.TimeStamp = o.TimeStamp
	}
	if keep(fPointer) {
		r.Pointer = o.Pointer
	}
	if keep(fKind) {
		r.Kind = o.Kind
	}
	if keep(fDevice) {
		r.Device = o.Device
	}
	if keep(fPosition) {
		r.Position = o.Position
	}
	if keep(fDelta) {
		r.Delta = o.Delta
	}
	if keep(fButtons) {
		r.Buttons = o.Buttons
	}
	if keep(fDown) {
		r.Down = o.Down
	}
	if keep(fObscured) {
		r.Obscured = o.Obscured
	}
	if keep(fPressure) {
		r.Pressure = o.Pressure
	}
	if keep(fPressureMin) {
		r.PressureMin = o.PressureMin
	}
	if keep(fPressureMax) {
		r.PressureMax = o.PressureMax
	}
	if keep(fDistance) {
		r.Distance = o.Distance
	}
	if keep(fDistanceMax) {
		r.DistanceMax = o.DistanceMax
	}
	if keep(fSize) {
		r.Size = o.Size
	}
	if keep(fRadiusMajor) {
		r.RadiusMajor = o.RadiusMajor
	}
	if keep(fRadiusMinor) {
		r.RadiusMinor = o.RadiusMinor
	}
	if keep(fRadiusMin) {
		r.RadiusMin = o.RadiusMin
	}
	if keep(fRadiusMax) {
		r.RadiusMax = o.RadiusMax
	}
	if keep(fOrientation) {
		r.Orientation = o.Orientation
	}
	if keep(fTilt) {
		r.Tilt = o.Tilt
	}
	if keep(fPlatformData) {
		r.PlatformData = o.PlatformData
	}
	if keep(fSynthesized) {
		r.Synthesized = o.Synthesized
	}
	if keep(fScrollDelta) {
		r.ScrollDelta = o.ScrollDelta
	}
	if keep(fScale) {
		r.Scale = o.Scale
	}
	if keep(fPan) {
		r.Pan = o.Pan
	}
	if keep(fPanDelta) {
		r.PanDelta = o.PanDelta
	}
	if keep(fRotation) {
		r.Rotation = o.Rotation
	}
	if keep(fOnRespond) {
		r.OnRespond = o.OnRespond
	}
	return r
}

// eventSpec describes what the constructor and copyWith of one event class
// take and fix.
type eventSpec struct {
	// accepts are the fields of the constructor.
	accepts fieldSet
	// copies are the fields copyWith passes to the constructor.
	copies fieldSet
	// kind is the kind when none is given.
	kind ui.PointerDeviceKind
	// forceKind makes kind the kind whatever is given.
	forceKind bool
	// noTrackpad makes a trackpad kind an error.
	noTrackpad bool
	// buttons is the default of Buttons.
	buttons int
	// pressure is the default of Pressure, or its fixed value if the
	// constructor does not take it.
	pressure float64
	// down is the fixed value of Down if the constructor does not take it.
	down bool
}

// eventData is the state of every event, embedded by the concrete types. A
// transformed event is a copy of its original with the transform and the local
// values set.
type eventData struct {
	viewID       ui.ViewID
	embedderID   int
	timeStamp    time.Duration
	pointer      int
	kind         ui.PointerDeviceKind
	device       int
	position     ui.Offset
	localPos     ui.Offset
	delta        ui.Offset
	localDelta   ui.Offset
	buttons      int
	down         bool
	obscured     bool
	pressure     float64
	pressureMin  float64
	pressureMax  float64
	distance     float64
	distanceMax  float64
	size         float64
	radiusMajor  float64
	radiusMinor  float64
	radiusMin    float64
	radiusMax    float64
	orientation  float64
	tilt         float64
	platformData int
	synthesized  bool
	transform    *ui.Matrix4
	original     PointerEvent

	scrollDelta ui.Offset
	scale       float64
	pan         ui.Offset
	localPan    ui.Offset
	panDelta    ui.Offset
	localPanDlt ui.Offset
	rotation    float64
	onRespond   func(allowPlatformDefault bool)
}

func newEventData(typeName string, s *eventSpec, o PointerEventOpts) eventData {
	o = o.masked(s.accepts)
	kind := o.Kind
	if kind == ui.PointerDeviceKindUnspecified || s.forceKind {
		kind = s.kind
	}
	if foundation.DebugMode && s.noTrackpad && kind == ui.PointerDeviceKindTrackpad {
		panic(errors.New(tagPointerEventKindTrackpad, "this pointer event cannot come from a trackpad",
			"event", typeName))
	}
	d := eventData{
		viewID:       o.ViewID,
		embedderID:   o.EmbedderID,
		timeStamp:    o.TimeStamp,
		pointer:      o.Pointer,
		kind:         kind,
		device:       o.Device,
		position:     o.Position,
		localPos:     o.Position,
		delta:        o.Delta,
		localDelta:   o.Delta,
		buttons:      o.Buttons.OrElse(s.buttons),
		down:         s.down,
		obscured:     o.Obscured,
		pressure:     s.pressure,
		pressureMin:  o.PressureMin.OrElse(1),
		pressureMax:  o.PressureMax.OrElse(1),
		distance:     o.Distance,
		distanceMax:  o.DistanceMax,
		size:         o.Size,
		radiusMajor:  o.RadiusMajor,
		radiusMinor:  o.RadiusMinor,
		radiusMin:    o.RadiusMin,
		radiusMax:    o.RadiusMax,
		orientation:  o.Orientation,
		tilt:         o.Tilt,
		platformData: o.PlatformData,
		synthesized:  o.Synthesized,
		scrollDelta:  o.ScrollDelta,
		scale:        o.Scale.OrElse(1),
		pan:          o.Pan,
		localPan:     o.Pan,
		panDelta:     o.PanDelta,
		localPanDlt:  o.PanDelta,
		rotation:     o.Rotation,
		onRespond:    o.OnRespond,
	}
	if s.accepts&fDown != 0 {
		d.down = o.Down
	}
	if s.accepts&fPressure != 0 {
		d.pressure = o.Pressure.OrElse(s.pressure)
	}
	return d
}

func (d *eventData) data() *eventData { return d }

func (d *eventData) ViewID() ui.ViewID          { return d.viewID }
func (d *eventData) EmbedderID() int            { return d.embedderID }
func (d *eventData) TimeStamp() time.Duration   { return d.timeStamp }
func (d *eventData) Pointer() int               { return d.pointer }
func (d *eventData) Kind() ui.PointerDeviceKind { return d.kind }
func (d *eventData) Device() int                { return d.device }
func (d *eventData) Position() ui.Offset        { return d.position }
func (d *eventData) LocalPosition() ui.Offset   { return d.localPos }
func (d *eventData) Delta() ui.Offset           { return d.delta }
func (d *eventData) LocalDelta() ui.Offset      { return d.localDelta }
func (d *eventData) Buttons() int               { return d.buttons }
func (d *eventData) Down() bool                 { return d.down }
func (d *eventData) Obscured() bool             { return d.obscured }
func (d *eventData) Pressure() float64          { return d.pressure }
func (d *eventData) PressureMin() float64       { return d.pressureMin }
func (d *eventData) PressureMax() float64       { return d.pressureMax }
func (d *eventData) Distance() float64          { return d.distance }
func (d *eventData) DistanceMin() float64       { return 0 }
func (d *eventData) DistanceMax() float64       { return d.distanceMax }
func (d *eventData) Size() float64              { return d.size }
func (d *eventData) RadiusMajor() float64       { return d.radiusMajor }
func (d *eventData) RadiusMinor() float64       { return d.radiusMinor }
func (d *eventData) RadiusMin() float64         { return d.radiusMin }
func (d *eventData) RadiusMax() float64         { return d.radiusMax }
func (d *eventData) Orientation() float64       { return d.orientation }
func (d *eventData) Tilt() float64              { return d.tilt }
func (d *eventData) PlatformData() int          { return d.platformData }
func (d *eventData) Synthesized() bool          { return d.synthesized }
func (d *eventData) Transform() *ui.Matrix4     { return d.transform }
func (d *eventData) Original() PointerEvent     { return d.original }

// Respond calls the callback the event was made with, if any.
//
// Flutter: gestures/events.dart PointerSignalEvent.respond
func (d *eventData) Respond(allowPlatformDefault bool) {
	if d.onRespond != nil {
		d.onRespond(allowPlatformDefault)
	}
}

// optsOf returns the options that make an event like e, for CopyWith.
func optsOf(e PointerEvent) PointerEventOpts {
	d := e.data()
	return PointerEventOpts{
		ViewID:       d.viewID,
		EmbedderID:   d.embedderID,
		TimeStamp:    d.timeStamp,
		Pointer:      d.pointer,
		Kind:         d.kind,
		Device:       d.device,
		Position:     d.position,
		Delta:        d.delta,
		Buttons:      foundation.Some(d.buttons),
		Down:         d.down,
		Obscured:     d.obscured,
		Pressure:     foundation.Some(d.pressure),
		PressureMin:  foundation.Some(d.pressureMin),
		PressureMax:  foundation.Some(d.pressureMax),
		Distance:     d.distance,
		DistanceMax:  d.distanceMax,
		Size:         d.size,
		RadiusMajor:  d.radiusMajor,
		RadiusMinor:  d.radiusMinor,
		RadiusMin:    d.radiusMin,
		RadiusMax:    d.radiusMax,
		Orientation:  d.orientation,
		Tilt:         d.tilt,
		PlatformData: d.platformData,
		Synthesized:  d.synthesized,
		ScrollDelta:  d.scrollDelta,
		Scale:        foundation.Some(d.scale),
		Pan:          d.pan,
		PanDelta:     d.panDelta,
		Rotation:     d.rotation,
		OnRespond:    d.onRespond,
	}
}

// copyEvent makes the copy CopyWith returns: a new event of the type that
// make builds, from the fields the type's copyWith lists, with the
// replacements applied, transformed as e is.
func copyEvent(s *eventSpec, e PointerEvent, c PointerEventCopyOpts, make func(PointerEventOpts) PointerEvent) PointerEvent {
	o := optsOf(e)
	if v, ok := c.ViewID.Get(); ok {
		o.ViewID = v
	}
	if v, ok := c.TimeStamp.Get(); ok {
		o.TimeStamp = v
	}
	if v, ok := c.Pointer.Get(); ok {
		o.Pointer = v
	}
	if v, ok := c.Kind.Get(); ok {
		o.Kind = v
	}
	if v, ok := c.Device.Get(); ok {
		o.Device = v
	}
	if v, ok := c.Position.Get(); ok {
		o.Position = v
	}
	if v, ok := c.Delta.Get(); ok {
		o.Delta = v
	}
	if v, ok := c.Buttons.Get(); ok {
		o.Buttons = foundation.Some(v)
	}
	if v, ok := c.Obscured.Get(); ok {
		o.Obscured = v
	}
	if v, ok := c.Pressure.Get(); ok {
		o.Pressure = foundation.Some(v)
	}
	if v, ok := c.PressureMin.Get(); ok {
		o.PressureMin = foundation.Some(v)
	}
	if v, ok := c.PressureMax.Get(); ok {
		o.PressureMax = foundation.Some(v)
	}
	if v, ok := c.Distance.Get(); ok {
		o.Distance = v
	}
	if v, ok := c.DistanceMax.Get(); ok {
		o.DistanceMax = v
	}
	if v, ok := c.Size.Get(); ok {
		o.Size = v
	}
	if v, ok := c.RadiusMajor.Get(); ok {
		o.RadiusMajor = v
	}
	if v, ok := c.RadiusMinor.Get(); ok {
		o.RadiusMinor = v
	}
	if v, ok := c.RadiusMin.Get(); ok {
		o.RadiusMin = v
	}
	if v, ok := c.RadiusMax.Get(); ok {
		o.RadiusMax = v
	}
	if v, ok := c.Orientation.Get(); ok {
		o.Orientation = v
	}
	if v, ok := c.Tilt.Get(); ok {
		o.Tilt = v
	}
	if v, ok := c.Synthesized.Get(); ok {
		o.Synthesized = v
	}
	if v, ok := c.EmbedderID.Get(); ok {
		o.EmbedderID = v
	}
	if sig, ok := e.(PointerSignalEvent); ok {
		o.OnRespond = sig.Respond
		if c.OnRespond != nil {
			o.OnRespond = c.OnRespond
		}
	}
	return make(o.masked(s.copies)).Transformed(e.Transform())
}

// transformEvent is the Transformed method of every event type.
func transformEvent[T any, P interface {
	*T
	PointerEvent
}](e P, m *ui.Matrix4) P {
	base := e
	if orig := e.Original(); orig != nil {
		base = orig.(P)
	}
	if m == nil {
		return base
	}
	c := P(new(T))
	*c = *base
	d := c.data()
	d.original = base
	d.transform = m
	d.localPos = TransformPosition(m, d.position)
	d.localDelta = TransformDeltaViaPositions(d.position, d.delta, m,
		TransformDeltaViaPositionsOpts{TransformedEndPosition: foundation.Some(d.localPos)})
	d.localPan = TransformPosition(m, d.pan)
	d.localPanDlt = TransformDeltaViaPositions(d.pan, d.panDelta, m,
		TransformDeltaViaPositionsOpts{TransformedEndPosition: foundation.Some(d.localPan)})
	return c
}

// TransformPosition returns position transformed by transform; a nil transform
// leaves it as it is.
//
// Flutter: gestures/events.dart PointerEvent.transformPosition
func TransformPosition(transform *ui.Matrix4, position ui.Offset) ui.Offset {
	if transform == nil {
		return position
	}
	v := transform.PerspectiveTransform(ui.Vector3{X: float32(position.Dx), Y: float32(position.Dy)})
	return ui.Offset{Dx: float64(v.X), Dy: float64(v.Y)}
}

// TransformDeltaViaPositionsOpts are the optional parameters of
// TransformDeltaViaPositions.
//
// Flutter: gestures/events.dart PointerEvent.transformDeltaViaPositions
type TransformDeltaViaPositionsOpts struct {
	// TransformedEndPosition is the end position already transformed; if it
	// is None the function transforms it.
	TransformedEndPosition foundation.Opt[ui.Offset]
}

// TransformDeltaViaPositions transforms a delta that ends at
// untransformedEndPosition. It transforms the start and the end of the delta
// and subtracts them, which is more precise than transforming the delta.
//
// Flutter: gestures/events.dart PointerEvent.transformDeltaViaPositions
func TransformDeltaViaPositions(untransformedEndPosition, untransformedDelta ui.Offset, transform *ui.Matrix4, opts ...TransformDeltaViaPositionsOpts) ui.Offset {
	if transform == nil {
		return untransformedDelta
	}
	var o TransformDeltaViaPositionsOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	end, ok := o.TransformedEndPosition.Get()
	if !ok {
		end = TransformPosition(transform, untransformedEndPosition)
	}
	start := TransformPosition(transform, untransformedEndPosition.Sub(untransformedDelta))
	return end.Sub(start)
}

// RemovePerspectiveTransform returns a copy of transform that keeps a point
// under the pointing device: its third row and third column are those of the
// identity. A paint transform must go through it before it is used for hit
// testing.
//
// Flutter: gestures/events.dart PointerEvent.removePerspectiveTransform
func RemovePerspectiveTransform(transform *ui.Matrix4) *ui.Matrix4 {
	m := transform.Clone()
	vec := [4]float32{0, 0, 1, 0}
	for i, v := range vec {
		m.SetEntry(i, 2, v)
	}
	for i, v := range vec {
		m.SetEntry(2, i, v)
	}
	return m
}

// DebugFillProperties adds the properties of the event.
//
// Flutter: gestures/events.dart _PointerEventDescription.debugFillProperties
func (d *eventData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	debug := foundation.DiagnosticLevelDebug
	num := func(name string, v float64, def any) {
		p.Add(foundation.NewDoubleProperty(name, foundation.Some(v),
			foundation.NumberPropertyOpts{DefaultValue: def, Level: debug}))
	}
	p.Add(foundation.NewDiagnosticsProperty("position", d.position))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.localPos,
		foundation.DiagnosticsPropertyOpts{DefaultValue: d.position, Level: debug}))
	p.Add(foundation.NewDiagnosticsProperty("delta", d.delta,
		foundation.DiagnosticsPropertyOpts{DefaultValue: ui.Offset{}, Level: debug}))
	p.Add(foundation.NewDiagnosticsProperty("localDelta", d.localDelta,
		foundation.DiagnosticsPropertyOpts{DefaultValue: d.delta, Level: debug}))
	p.Add(foundation.NewDiagnosticsProperty("timeStamp", d.timeStamp,
		foundation.DiagnosticsPropertyOpts{
			Description:  dartfmt.Duration(d.timeStamp),
			DefaultValue: time.Duration(0),
			Level:        debug,
		}))
	p.Add(foundation.NewIntProperty("pointer", foundation.Some(d.pointer),
		foundation.NumberPropertyOpts{Level: debug}))
	p.Add(foundation.NewEnumProperty("kind", foundation.Some(d.kind),
		foundation.EnumPropertyOpts{Level: debug}))
	p.Add(foundation.NewIntProperty("device", foundation.Some(d.device),
		foundation.NumberPropertyOpts{DefaultValue: 0, Level: debug}))
	p.Add(foundation.NewIntProperty("buttons", foundation.Some(d.buttons),
		foundation.NumberPropertyOpts{DefaultValue: 0, Level: debug}))
	p.Add(foundation.NewDiagnosticsProperty("down", d.down,
		foundation.DiagnosticsPropertyOpts{Level: debug}))
	num("pressure", d.pressure, 1.0)
	num("pressureMin", d.pressureMin, 1.0)
	num("pressureMax", d.pressureMax, 1.0)
	num("distance", d.distance, 0.0)
	num("distanceMin", 0, 0.0)
	num("distanceMax", d.distanceMax, 0.0)
	num("size", d.size, 0.0)
	num("radiusMajor", d.radiusMajor, 0.0)
	num("radiusMinor", d.radiusMinor, 0.0)
	num("radiusMin", d.radiusMin, 0.0)
	num("radiusMax", d.radiusMax, 0.0)
	num("orientation", d.orientation, 0.0)
	num("tilt", d.tilt, 0.0)
	p.Add(foundation.NewIntProperty("platformData", foundation.Some(d.platformData),
		foundation.NumberPropertyOpts{DefaultValue: 0, Level: debug}))
	p.Add(foundation.NewFlagProperty("obscured", foundation.Some(d.obscured),
		foundation.FlagPropertyOpts{IfTrue: "obscured", Level: debug}))
	p.Add(foundation.NewFlagProperty("synthesized", foundation.Some(d.synthesized),
		foundation.FlagPropertyOpts{IfTrue: "synthesized", Level: debug}))
	p.Add(foundation.NewIntProperty("embedderId", foundation.Some(d.embedderID),
		foundation.NumberPropertyOpts{DefaultValue: 0, Level: debug}))
	p.Add(foundation.NewIntProperty("viewId", foundation.Some(int(d.viewID)),
		foundation.NumberPropertyOpts{DefaultValue: 0, Level: debug}))
}

// ToStringFull describes the event with all its properties, down to the fine
// level.
//
// Flutter: gestures/events.dart _PointerEventDescription.toStringFull
func ToStringFull(e PointerEvent) string {
	return foundation.DiagnosticableString(e, foundation.DiagnosticLevelFine)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPointerEventKindTrackpad = "PointerEventKindTrackpad"
)
