// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// This file has one type for each event class of Flutter's events.dart. The
// types differ in the fields their constructor takes and copyWith copies; the
// eventSpec of each says which.

var addedSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	kind:     ui.PointerDeviceKindTouch,
	pressure: 0,
}

// PointerAddedEvent is an event: the device has started tracking the pointer. For example, the pointer might be a touch that is about to start; for a mouse it means the mouse is in the window.
//
// Flutter: gestures/events.dart PointerAddedEvent
type PointerAddedEvent struct{ eventData }

// NewPointerAddedEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerAddedEvent.new
func NewPointerAddedEvent(opts ...PointerEventOpts) *PointerAddedEvent {
	return &PointerAddedEvent{newEventData("PointerAddedEvent", &addedSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerAddedEvent.transformed
func (e *PointerAddedEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerAddedEvent.copyWith
func (e *PointerAddedEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&addedSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerAddedEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerAddedEvent.toString
func (e *PointerAddedEvent) String() string { return foundation.DiagnosticableString(e) }

var removedSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fObscured | fPressureMin | fPressureMax | fDistanceMax | fRadiusMin | fRadiusMax,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fObscured | fPressureMin | fPressureMax | fDistanceMax | fRadiusMin | fRadiusMax,
	kind:     ui.PointerDeviceKindTouch,
	pressure: 0,
}

// PointerRemovedEvent is an event: the device is no longer tracking the pointer. For example, the pointer might be a touch that ended or a mouse that left the window.
//
// Flutter: gestures/events.dart PointerRemovedEvent
type PointerRemovedEvent struct{ eventData }

// NewPointerRemovedEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerRemovedEvent.new
func NewPointerRemovedEvent(opts ...PointerEventOpts) *PointerRemovedEvent {
	return &PointerRemovedEvent{newEventData("PointerRemovedEvent", &removedSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerRemovedEvent.transformed
func (e *PointerRemovedEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerRemovedEvent.copyWith
func (e *PointerRemovedEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&removedSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerRemovedEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerRemovedEvent.toString
func (e *PointerRemovedEvent) String() string { return foundation.DiagnosticableString(e) }

var hoverSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized | fPointer,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized,
	kind:     ui.PointerDeviceKindTouch,
	pressure: 0,
}

// PointerHoverEvent is an event: the pointer has moved with respect to the device while the pointer is not in contact with the device.
//
// Flutter: gestures/events.dart PointerHoverEvent
type PointerHoverEvent struct{ eventData }

// NewPointerHoverEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerHoverEvent.new
func NewPointerHoverEvent(opts ...PointerEventOpts) *PointerHoverEvent {
	return &PointerHoverEvent{newEventData("PointerHoverEvent", &hoverSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerHoverEvent.transformed
func (e *PointerHoverEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerHoverEvent.copyWith
func (e *PointerHoverEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&hoverSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerHoverEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerHoverEvent.toString
func (e *PointerHoverEvent) String() string { return foundation.DiagnosticableString(e) }

var enterSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized | fPointer | fDown,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized,
	kind:       ui.PointerDeviceKindTouch,
	pressure:   0,
	noTrackpad: true,
}

// PointerEnterEvent is an event: the pointer has moved with respect to the device while the pointer is or is not in contact with the device, and entered a target. It is made by the mouse tracker; a trackpad cannot make one.
//
// Flutter: gestures/events.dart PointerEnterEvent
type PointerEnterEvent struct{ eventData }

// NewPointerEnterEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerEnterEvent.new
func NewPointerEnterEvent(opts ...PointerEventOpts) *PointerEnterEvent {
	return &PointerEnterEvent{newEventData("PointerEnterEvent", &enterSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerEnterEvent.transformed
func (e *PointerEnterEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerEnterEvent.copyWith
func (e *PointerEnterEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&enterSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerEnterEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerEnterEvent.toString
func (e *PointerEnterEvent) String() string { return foundation.DiagnosticableString(e) }

var exitSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized | fPointer | fDown,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fDelta | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fSynthesized,
	kind:       ui.PointerDeviceKindTouch,
	pressure:   0,
	noTrackpad: true,
}

// PointerExitEvent is an event: the pointer has moved with respect to the device while the pointer is or is not in contact with the device, and exited a target. It is made by the mouse tracker; a trackpad cannot make one.
//
// Flutter: gestures/events.dart PointerExitEvent
type PointerExitEvent struct{ eventData }

// NewPointerExitEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerExitEvent.new
func NewPointerExitEvent(opts ...PointerEventOpts) *PointerExitEvent {
	return &PointerExitEvent{newEventData("PointerExitEvent", &exitSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerExitEvent.transformed
func (e *PointerExitEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerExitEvent.copyWith
func (e *PointerExitEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&exitSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerExitEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerExitEvent.toString
func (e *PointerExitEvent) String() string { return foundation.DiagnosticableString(e) }

var downSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	kind:       ui.PointerDeviceKindTouch,
	buttons:    PrimaryButton,
	pressure:   1,
	down:       true,
	noTrackpad: true,
}

// PointerDownEvent is an event: the pointer has made contact with the device. Its buttons default to PrimaryButton.
//
// Flutter: gestures/events.dart PointerDownEvent
type PointerDownEvent struct{ eventData }

// NewPointerDownEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerDownEvent.new
func NewPointerDownEvent(opts ...PointerEventOpts) *PointerDownEvent {
	return &PointerDownEvent{newEventData("PointerDownEvent", &downSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerDownEvent.transformed
func (e *PointerDownEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerDownEvent.copyWith
func (e *PointerDownEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&downSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerDownEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerDownEvent.toString
func (e *PointerDownEvent) String() string { return foundation.DiagnosticableString(e) }

var moveSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fDelta | fSynthesized | fPlatformData,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt | fDelta | fSynthesized,
	kind:       ui.PointerDeviceKindTouch,
	buttons:    PrimaryButton,
	pressure:   1,
	down:       true,
	noTrackpad: true,
}

// PointerMoveEvent is an event: the pointer has moved with respect to the device while the pointer is in contact with the device. Its buttons default to PrimaryButton.
//
// Flutter: gestures/events.dart PointerMoveEvent
type PointerMoveEvent struct{ eventData }

// NewPointerMoveEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerMoveEvent.new
func NewPointerMoveEvent(opts ...PointerEventOpts) *PointerMoveEvent {
	return &PointerMoveEvent{newEventData("PointerMoveEvent", &moveSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerMoveEvent.transformed
func (e *PointerMoveEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerMoveEvent.copyWith
func (e *PointerMoveEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&moveSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerMoveEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerMoveEvent.toString
func (e *PointerMoveEvent) String() string { return foundation.DiagnosticableString(e) }

var upSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressure | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	kind:       ui.PointerDeviceKindTouch,
	pressure:   0,
	noTrackpad: true,
}

// PointerUpEvent is an event: the pointer has stopped making contact with the device. The pressure defaults to 0, and a device may report another.
//
// Flutter: gestures/events.dart PointerUpEvent
type PointerUpEvent struct{ eventData }

// NewPointerUpEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerUpEvent.new
func NewPointerUpEvent(opts ...PointerEventOpts) *PointerUpEvent {
	return &PointerUpEvent{newEventData("PointerUpEvent", &upSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerUpEvent.transformed
func (e *PointerUpEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerUpEvent.copyWith
func (e *PointerUpEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&upSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerUpEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerUpEvent.toString
func (e *PointerUpEvent) String() string { return foundation.DiagnosticableString(e) }

var cancelSpec = eventSpec{
	accepts:    fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	copies:     fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fPointer | fButtons | fObscured | fPressureMin | fPressureMax | fDistance | fDistanceMax | fSize | fRadiusMajor | fRadiusMinor | fRadiusMin | fRadiusMax | fOrientation | fTilt,
	kind:       ui.PointerDeviceKindTouch,
	pressure:   0,
	noTrackpad: true,
}

// PointerCancelEvent is an event: the input from the pointer is no longer directed towards this receiver.
//
// Flutter: gestures/events.dart PointerCancelEvent
type PointerCancelEvent struct{ eventData }

// NewPointerCancelEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerCancelEvent.new
func NewPointerCancelEvent(opts ...PointerEventOpts) *PointerCancelEvent {
	return &PointerCancelEvent{newEventData("PointerCancelEvent", &cancelSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerCancelEvent.transformed
func (e *PointerCancelEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerCancelEvent.copyWith
func (e *PointerCancelEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&cancelSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerCancelEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerCancelEvent.toString
func (e *PointerCancelEvent) String() string { return foundation.DiagnosticableString(e) }

var scrollSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fScrollDelta | fOnRespond,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fScrollDelta | fOnRespond,
	kind:     ui.PointerDeviceKindMouse,
	pressure: 1,
}

// PointerScrollEvent is an event: the pointer issued a scroll signal, such as a mouse wheel turning. ScrollDelta is how far to scroll, in logical pixels.
//
// Flutter: gestures/events.dart PointerScrollEvent
type PointerScrollEvent struct{ eventData }

// NewPointerScrollEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerScrollEvent.new
func NewPointerScrollEvent(opts ...PointerEventOpts) *PointerScrollEvent {
	return &PointerScrollEvent{newEventData("PointerScrollEvent", &scrollSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerScrollEvent.transformed
func (e *PointerScrollEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerScrollEvent.copyWith
func (e *PointerScrollEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&scrollSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerScrollEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerScrollEvent.toString
func (e *PointerScrollEvent) String() string { return foundation.DiagnosticableString(e) }

func (e *PointerScrollEvent) isSignal() {}

var scrollInertiaCancelSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID,
	kind:     ui.PointerDeviceKindMouse,
	pressure: 1,
}

// PointerScrollInertiaCancelEvent is an event: the pointer issued a signal that stops the inertia of a scroll, such as fingers touching a trackpad while it still scrolls.
//
// Flutter: gestures/events.dart PointerScrollInertiaCancelEvent
type PointerScrollInertiaCancelEvent struct{ eventData }

// NewPointerScrollInertiaCancelEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerScrollInertiaCancelEvent.new
func NewPointerScrollInertiaCancelEvent(opts ...PointerEventOpts) *PointerScrollInertiaCancelEvent {
	return &PointerScrollInertiaCancelEvent{newEventData("PointerScrollInertiaCancelEvent", &scrollInertiaCancelSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerScrollInertiaCancelEvent.transformed
func (e *PointerScrollInertiaCancelEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerScrollInertiaCancelEvent.copyWith
func (e *PointerScrollInertiaCancelEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&scrollInertiaCancelSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerScrollInertiaCancelEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerScrollInertiaCancelEvent.toString
func (e *PointerScrollInertiaCancelEvent) String() string { return foundation.DiagnosticableString(e) }

func (e *PointerScrollInertiaCancelEvent) isSignal() {}

var scaleSpec = eventSpec{
	accepts:  fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fScale,
	copies:   fViewID | fTimeStamp | fKind | fDevice | fPosition | fEmbedderID | fScale,
	kind:     ui.PointerDeviceKindMouse,
	pressure: 1,
}

// PointerScaleEvent is an event: the pointer issued a scale signal, such as a pinch of a trackpad. Scale is the factor.
//
// Flutter: gestures/events.dart PointerScaleEvent
type PointerScaleEvent struct{ eventData }

// NewPointerScaleEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerScaleEvent.new
func NewPointerScaleEvent(opts ...PointerEventOpts) *PointerScaleEvent {
	return &PointerScaleEvent{newEventData("PointerScaleEvent", &scaleSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerScaleEvent.transformed
func (e *PointerScaleEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerScaleEvent.copyWith
func (e *PointerScaleEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&scaleSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerScaleEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerScaleEvent.toString
func (e *PointerScaleEvent) String() string { return foundation.DiagnosticableString(e) }

func (e *PointerScaleEvent) isSignal() {}

var panZoomStartSpec = eventSpec{
	accepts:   fViewID | fTimeStamp | fDevice | fPointer | fPosition | fEmbedderID | fSynthesized,
	copies:    fViewID | fTimeStamp | fDevice | fPosition | fEmbedderID,
	kind:      ui.PointerDeviceKindTrackpad,
	pressure:  1,
	forceKind: true,
}

// PointerPanZoomStartEvent is an event: a pan/zoom gesture of a trackpad has started.
//
// Flutter: gestures/events.dart PointerPanZoomStartEvent
type PointerPanZoomStartEvent struct{ eventData }

// NewPointerPanZoomStartEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerPanZoomStartEvent.new
func NewPointerPanZoomStartEvent(opts ...PointerEventOpts) *PointerPanZoomStartEvent {
	return &PointerPanZoomStartEvent{newEventData("PointerPanZoomStartEvent", &panZoomStartSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerPanZoomStartEvent.transformed
func (e *PointerPanZoomStartEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerPanZoomStartEvent.copyWith
func (e *PointerPanZoomStartEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&panZoomStartSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerPanZoomStartEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerPanZoomStartEvent.toString
func (e *PointerPanZoomStartEvent) String() string { return foundation.DiagnosticableString(e) }

var panZoomUpdateSpec = eventSpec{
	accepts:   fViewID | fTimeStamp | fDevice | fPointer | fPosition | fEmbedderID | fSynthesized | fPan | fPanDelta | fScale | fRotation,
	copies:    fViewID | fTimeStamp | fDevice | fPosition | fEmbedderID | fPan | fPanDelta | fScale | fRotation,
	kind:      ui.PointerDeviceKindTrackpad,
	pressure:  1,
	forceKind: true,
}

// PointerPanZoomUpdateEvent is an event: a pan/zoom gesture of a trackpad has changed. Pan, Scale and Rotation are the totals since it started, and PanDelta is the change of Pan since the last update.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent
type PointerPanZoomUpdateEvent struct{ eventData }

// NewPointerPanZoomUpdateEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.new
func NewPointerPanZoomUpdateEvent(opts ...PointerEventOpts) *PointerPanZoomUpdateEvent {
	return &PointerPanZoomUpdateEvent{newEventData("PointerPanZoomUpdateEvent", &panZoomUpdateSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.transformed
func (e *PointerPanZoomUpdateEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.copyWith
func (e *PointerPanZoomUpdateEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&panZoomUpdateSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerPanZoomUpdateEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerPanZoomUpdateEvent.toString
func (e *PointerPanZoomUpdateEvent) String() string { return foundation.DiagnosticableString(e) }

var panZoomEndSpec = eventSpec{
	accepts:   fViewID | fTimeStamp | fDevice | fPointer | fPosition | fEmbedderID | fSynthesized,
	copies:    fViewID | fTimeStamp | fDevice | fPosition | fEmbedderID,
	kind:      ui.PointerDeviceKindTrackpad,
	pressure:  1,
	forceKind: true,
}

// PointerPanZoomEndEvent is an event: a pan/zoom gesture of a trackpad has ended.
//
// Flutter: gestures/events.dart PointerPanZoomEndEvent
type PointerPanZoomEndEvent struct{ eventData }

// NewPointerPanZoomEndEvent creates the event. It reads the fields of PointerEventOpts that the
// Flutter constructor has.
//
// Flutter: gestures/events.dart PointerPanZoomEndEvent.new
func NewPointerPanZoomEndEvent(opts ...PointerEventOpts) *PointerPanZoomEndEvent {
	return &PointerPanZoomEndEvent{newEventData("PointerPanZoomEndEvent", &panZoomEndSpec, oneOpts(opts))}
}

// Transformed returns the event in the coordinates that transform leads to.
//
// Flutter: gestures/events.dart PointerPanZoomEndEvent.transformed
func (e *PointerPanZoomEndEvent) Transformed(transform *ui.Matrix4) PointerEvent {
	return transformEvent(e, transform)
}

// CopyWith returns a copy with the fields replaced.
//
// Flutter: gestures/events.dart PointerPanZoomEndEvent.copyWith
func (e *PointerPanZoomEndEvent) CopyWith(c PointerEventCopyOpts) PointerEvent {
	return copyEvent(&panZoomEndSpec, e, c, func(o PointerEventOpts) PointerEvent { return NewPointerPanZoomEndEvent(o) })
}

// String describes the event on one line.
//
// Flutter: gestures/events.dart PointerPanZoomEndEvent.toString
func (e *PointerPanZoomEndEvent) String() string { return foundation.DiagnosticableString(e) }
