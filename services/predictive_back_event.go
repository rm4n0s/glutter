// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SwipeEdge is the edge of the screen from which a swipe starts in a back
// gesture. It is used in PredictiveBackEvent.
//
// Flutter: services/predictive_back_event.dart SwipeEdge
type SwipeEdge int

// The values of SwipeEdge follow Flutter's declaration order.
const (
	// SwipeEdgeUnspecified is the zero value: no edge is known. A PredictiveBackEvent
	// cannot have it.
	//
	// Flutter: no counterpart
	SwipeEdgeUnspecified SwipeEdge = iota
	// SwipeEdgeLeft means that the swipe gesture starts from the left edge of the screen.
	//
	// Flutter: services/predictive_back_event.dart SwipeEdge.left
	SwipeEdgeLeft
	// SwipeEdgeRight means that the swipe gesture starts from the right edge of the screen.
	//
	// Flutter: services/predictive_back_event.dart SwipeEdge.right
	SwipeEdgeRight
)

// String names the edge as Dart's enum toString does.
//
// Flutter: services/predictive_back_event.dart SwipeEdge
func (e SwipeEdge) String() string {
	switch e {
	case SwipeEdgeLeft:
		return "SwipeEdge.left"
	case SwipeEdgeRight:
		return "SwipeEdge.right"
	}
	return "SwipeEdge.unspecified"
}

// PredictiveBackEvent reports the progress of a back gesture on Android: it holds
// the touch point, the swipe edge and the progress that a predictive back
// animation follows. It is immutable.
//
// Deviation: Flutter's PredictiveBackEvent.fromMap reads the event from the map
// of a platform channel; NewPredictiveBackEventFromUI reads it from the
// ui.BackGestureEvent that the engine gives the dispatcher, and the failed
// assertions on the progress and the swipe edge panic (SPEC §12.10).
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent
type PredictiveBackEvent struct {
	touchOffset foundation.Opt[ui.Offset]
	progress    float64
	swipeEdge   SwipeEdge
}

// NewPredictiveBackEvent creates an event. The touchOffset is the global position
// of the touch point, or none for a back that a button makes. It panics with the
// tag PredictiveBackProgressOutOfRange when the progress is not in 0..1, and with
// PredictiveBackSwipeEdgeInvalid when the edge is not left or right.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent._
func NewPredictiveBackEvent(touchOffset foundation.Opt[ui.Offset], progress float64, swipeEdge SwipeEdge) PredictiveBackEvent {
	if !(progress >= 0.0 && progress <= 1.0) {
		panic(errors.New(tagPredictiveBackProgressOutOfRange, "the progress of a PredictiveBackEvent must be between 0.0 and 1.0",
			"progress", progress))
	}
	if swipeEdge != SwipeEdgeLeft && swipeEdge != SwipeEdgeRight {
		panic(errors.New(tagPredictiveBackSwipeEdgeInvalid, "the swipe edge of a PredictiveBackEvent must be left or right",
			"swipeEdge", int(swipeEdge)))
	}
	return PredictiveBackEvent{touchOffset: touchOffset, progress: progress, swipeEdge: swipeEdge}
}

// NewPredictiveBackEventFromUI creates an event from what the engine reports. It
// panics like NewPredictiveBackEvent.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.fromMap
func NewPredictiveBackEventFromUI(e ui.BackGestureEvent) PredictiveBackEvent {
	touch := foundation.None[ui.Offset]()
	if e.HasTouchOffset {
		touch = foundation.Some(ui.Offset{Dx: e.TouchX, Dy: e.TouchY})
	}
	var edge SwipeEdge
	switch e.SwipeEdge {
	case ui.BackSwipeEdgeLeft:
		edge = SwipeEdgeLeft
	case ui.BackSwipeEdgeRight:
		edge = SwipeEdgeRight
	}
	return NewPredictiveBackEvent(touch, e.Progress, edge)
}

// TouchOffset is the global position of the touch point, or none if the event is
// triggered by a button press (a back button on a device with hardware buttons).
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.touchOffset
func (e PredictiveBackEvent) TouchOffset() foundation.Opt[ui.Offset] { return e.touchOffset }

// Progress is a value between 0.0 and 1.0 that says how far along the back
// gesture is: 0.0 when the touch is at the starting edge, and about 1.0 at the
// opposite side of the screen. It is the fraction to seek the predictive back
// animation with. When the gesture is canceled, it continues to update, animating
// back to 0.0 until the cancellation animation completes.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.progress
func (e PredictiveBackEvent) Progress() float64 { return e.progress }

// SwipeEdge is the screen edge from which the swipe gesture starts.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.swipeEdge
func (e PredictiveBackEvent) SwipeEdge() SwipeEdge { return e.swipeEdge }

// IsButtonEvent reports whether the event was triggered by a press of the system
// back button; it is false for a predictive back gesture.
//
// Android's documentation says that the touch point of a button press is NaN, but
// in practice it is 0.0, hence the check for the zero offset.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.isButtonEvent
func (e PredictiveBackEvent) IsButtonEvent() bool {
	t, ok := e.touchOffset.Get()
	return !ok || (e.progress == 0.0 && t == ui.OffsetZero)
}

// Equal reports whether the events have the same touch offset, progress and swipe
// edge.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.==
func (e PredictiveBackEvent) Equal(other PredictiveBackEvent) bool {
	return e.touchOffset == other.touchOffset && e.progress == other.progress && e.swipeEdge == other.swipeEdge
}

// String describes the event, as Dart's toString does.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.toString
func (e PredictiveBackEvent) String() string {
	touch := "null"
	if t, ok := e.touchOffset.Get(); ok {
		touch = t.String()
	}
	progress := strconv.FormatFloat(e.progress, 'f', -1, 64)
	if !strings.Contains(progress, ".") {
		progress += ".0"
	}
	return "PredictiveBackEvent{touchOffset: " + touch + ", progress: " + progress + ", swipeEdge: " + e.swipeEdge.String() + "}"
}

const (
	tagPredictiveBackProgressOutOfRange = "PredictiveBackProgressOutOfRange"
	tagPredictiveBackSwipeEdgeInvalid   = "PredictiveBackSwipeEdgeInvalid"
)
