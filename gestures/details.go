// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PositionedGestureDetails is the data of a gesture that happened at a
// position.
//
// Flutter: gestures/gesture_details.dart PositionedGestureDetails
type PositionedGestureDetails interface {
	// GlobalPosition is the position in the global coordinates of the view.
	//
	// Flutter: gestures/gesture_details.dart PositionedGestureDetails.globalPosition
	GetGlobalPosition() ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	//
	// Flutter: gestures/gesture_details.dart PositionedGestureDetails.localPosition
	GetLocalPosition() ui.Offset
}

func localOrGlobal(local foundation.Opt[ui.Offset], global ui.Offset) ui.Offset {
	return local.OrElse(global)
}

// DragDownOpts are the optional parameters of NewDragDownDetails.
//
// Flutter: gestures/drag_details.dart DragDownDetails.new
type DragDownOpts struct {
	// GlobalPosition is where the pointer touched the screen, in global
	// coordinates.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
}

// DragDownDetails is the data of the pointer that touched the screen, which
// might start a drag.
//
// Flutter: gestures/drag_details.dart DragDownDetails
type DragDownDetails struct {
	// GlobalPosition is the global position of the pointer when it touched the
	// screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
}

// NewDragDownDetails creates the details.
//
// Flutter: gestures/drag_details.dart DragDownDetails.new
func NewDragDownDetails(opts ...DragDownOpts) DragDownDetails {
	o := oneOpts(opts)
	return DragDownDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition)}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/drag_details.dart DragDownDetails.globalPosition
func (d DragDownDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/drag_details.dart DragDownDetails.localPosition
func (d DragDownDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/drag_details.dart DragDownDetails.debugFillProperties
func (d DragDownDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
}

// String describes the details.
//
// Flutter: gestures/drag_details.dart DragDownDetails.toString
func (d DragDownDetails) String() string { return foundation.DiagnosticableString(d) }

// GestureDragDownCallback is called when a pointer that might drag touches the
// screen.
//
// Flutter: gestures/drag_details.dart GestureDragDownCallback
type GestureDragDownCallback func(details DragDownDetails)

// DragStartOpts are the optional parameters of NewDragStartDetails.
//
// Flutter: gestures/drag_details.dart DragStartDetails.new
type DragStartOpts struct {
	// GlobalPosition is where the drag started, in global coordinates.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// SourceTimeStamp is the time of the event that started the drag.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Kind is the kind of device that started the drag.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// DragStartDetails is the data of a drag that started.
//
// Flutter: gestures/drag_details.dart DragStartDetails
type DragStartDetails struct {
	// GlobalPosition is the global position of the pointer when the drag
	// started.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// SourceTimeStamp is the time of the event that started the drag, if the
	// source gives one.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Kind is the kind of device that started the drag.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// NewDragStartDetails creates the details.
//
// Flutter: gestures/drag_details.dart DragStartDetails.new
func NewDragStartDetails(opts ...DragStartOpts) DragStartDetails {
	o := oneOpts(opts)
	return DragStartDetails{
		GlobalPosition:  o.GlobalPosition,
		LocalPosition:   localOrGlobal(o.LocalPosition, o.GlobalPosition),
		SourceTimeStamp: o.SourceTimeStamp,
		Kind:            o.Kind,
	}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/drag_details.dart DragStartDetails.globalPosition
func (d DragStartDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/drag_details.dart DragStartDetails.localPosition
func (d DragStartDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/drag_details.dart DragStartDetails.debugFillProperties
func (d DragStartDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// String describes the details.
//
// Flutter: gestures/drag_details.dart DragStartDetails.toString
func (d DragStartDetails) String() string { return foundation.DiagnosticableString(d) }

// GestureDragStartCallback is called when a drag starts.
//
// Flutter: gestures/drag_details.dart GestureDragStartCallback
type GestureDragStartCallback func(details DragStartDetails)

// DragUpdateOpts are the optional parameters of NewDragUpdateDetails.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.new
type DragUpdateOpts struct {
	// LocalPosition is the position in the coordinates of the target; the
	// default is the global position.
	LocalPosition foundation.Opt[ui.Offset]
	// SourceTimeStamp is the time of the event that moved the pointer.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Delta is how far the pointer moved since the last update.
	Delta ui.Offset
	// PrimaryDelta is the amount the pointer moved along the primary axis of
	// the drag. If given, the delta must be along that axis only, or a debug
	// build panics with the tag DragUpdatePrimaryDeltaInvalid.
	PrimaryDelta foundation.Opt[float64]
	// Kind is the kind of device that moved the pointer.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// DragUpdateDetails is the data of a pointer that moved while it dragged.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails
type DragUpdateDetails struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// SourceTimeStamp is the time of the event that moved the pointer, if the
	// source gives one.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Delta is how far the pointer moved since the last update.
	Delta ui.Offset
	// PrimaryDelta is the amount the pointer moved along the primary axis of
	// the drag; None for a drag that has no primary axis.
	PrimaryDelta foundation.Opt[float64]
	// Kind is the kind of device that moved the pointer.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// NewDragUpdateDetails creates the details for a pointer at the global
// position.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.new
func NewDragUpdateDetails(globalPosition ui.Offset, opts ...DragUpdateOpts) DragUpdateDetails {
	o := oneOpts(opts)
	if pd, ok := o.PrimaryDelta.Get(); ok && foundation.DebugMode {
		if !((pd == o.Delta.Dx && o.Delta.Dy == 0) || (pd == o.Delta.Dy && o.Delta.Dx == 0)) {
			panic(errors.New("DragUpdatePrimaryDeltaInvalid", "the primary delta must be the delta along one axis",
				"primaryDelta", pd, "delta", o.Delta.String()))
		}
	}
	return DragUpdateDetails{
		GlobalPosition:  globalPosition,
		LocalPosition:   localOrGlobal(o.LocalPosition, globalPosition),
		SourceTimeStamp: o.SourceTimeStamp,
		Delta:           o.Delta,
		PrimaryDelta:    o.PrimaryDelta,
		Kind:            o.Kind,
	}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.globalPosition
func (d DragUpdateDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.localPosition
func (d DragUpdateDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.debugFillProperties
func (d DragUpdateDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
	p.Add(foundation.NewDiagnosticsProperty("delta", d.Delta))
	p.Add(foundation.NewDoubleProperty("primaryDelta", d.PrimaryDelta))
}

// String describes the details.
//
// Flutter: gestures/drag_details.dart DragUpdateDetails.toString
func (d DragUpdateDetails) String() string { return foundation.DiagnosticableString(d) }

// GestureDragUpdateCallback is called when a pointer moves while it drags.
//
// Flutter: gestures/drag_details.dart GestureDragUpdateCallback
type GestureDragUpdateCallback func(details DragUpdateDetails)

// DragEndOpts are the optional parameters of NewDragEndDetails.
//
// Flutter: gestures/drag_details.dart DragEndDetails.new
type DragEndOpts struct {
	// GlobalPosition is where the pointer was lifted, in global coordinates.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// Velocity is the velocity of the pointer when it was lifted.
	Velocity Velocity
	// PrimaryVelocity is the velocity along the primary axis of the drag. If
	// given, the velocity must be along that axis only, or a debug build
	// panics with the tag DragEndPrimaryVelocityInvalid.
	PrimaryVelocity foundation.Opt[float64]
}

// DragEndDetails is the data of a drag that ended.
//
// Flutter: gestures/drag_details.dart DragEndDetails
type DragEndDetails struct {
	// GlobalPosition is the global position where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Velocity is the velocity the pointer had when it was lifted.
	Velocity Velocity
	// PrimaryVelocity is the velocity along the primary axis; None for a drag
	// that has no primary axis.
	PrimaryVelocity foundation.Opt[float64]
}

// NewDragEndDetails creates the details.
//
// Flutter: gestures/drag_details.dart DragEndDetails.new
func NewDragEndDetails(opts ...DragEndOpts) DragEndDetails {
	o := oneOpts(opts)
	if pv, ok := o.PrimaryVelocity.Get(); ok && foundation.DebugMode {
		v := o.Velocity.PixelsPerSecond
		if !((pv == v.Dx && v.Dy == 0) || (pv == v.Dy && v.Dx == 0)) {
			panic(errors.New("DragEndPrimaryVelocityInvalid", "the primary velocity must be the velocity along one axis",
				"primaryVelocity", pv, "velocity", o.Velocity.String()))
		}
	}
	return DragEndDetails{
		GlobalPosition:  o.GlobalPosition,
		LocalPosition:   localOrGlobal(o.LocalPosition, o.GlobalPosition),
		Velocity:        o.Velocity,
		PrimaryVelocity: o.PrimaryVelocity,
	}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/drag_details.dart DragEndDetails.globalPosition
func (d DragEndDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/drag_details.dart DragEndDetails.localPosition
func (d DragEndDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/drag_details.dart DragEndDetails.debugFillProperties
func (d DragEndDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	p.Add(foundation.NewDoubleProperty("primaryVelocity", d.PrimaryVelocity))
}

// String describes the details.
//
// Flutter: gestures/drag_details.dart DragEndDetails.toString
func (d DragEndDetails) String() string { return foundation.DiagnosticableString(d) }

// GestureDragEndCallback is called when a drag ends.
//
// Flutter: gestures/drag_details.dart GestureDragEndCallback
type GestureDragEndCallback func(details DragEndDetails)

// GestureDragCancelCallback is called when the pointer of a drag is cancelled.
//
// Flutter: gestures/drag_details.dart GestureDragCancelCallback
type GestureDragCancelCallback func()
