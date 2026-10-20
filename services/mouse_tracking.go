// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
)

// PointerEnterEventListener is called when a mouse pointer enters a region.
//
// Flutter: services/mouse_tracking.dart PointerEnterEventListener
type PointerEnterEventListener func(event *gestures.PointerEnterEvent)

// PointerExitEventListener is called when a mouse pointer leaves a region.
//
// Flutter: services/mouse_tracking.dart PointerExitEventListener
type PointerExitEventListener func(event *gestures.PointerExitEvent)

// PointerHoverEventListener is called when a mouse pointer moves over a region
// without a button down.
//
// Flutter: services/mouse_tracking.dart PointerHoverEventListener
type PointerHoverEventListener func(event *gestures.PointerHoverEvent)

// MouseTrackerAnnotation is what a hit-test target provides to be notified when
// the mouse enters or leaves it. The identity of the value is what the tracker
// compares, so use a pointer.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotation
type MouseTrackerAnnotation interface {
	// OnEnter is called when a mouse pointer, with or without buttons pressed,
	// enters the region; nil for none.
	OnEnter() PointerEnterEventListener
	// OnExit is called when a mouse pointer leaves the region; nil for none.
	OnExit() PointerExitEventListener
	// Cursor is the cursor over the region.
	Cursor() MouseCursor
	// ValidForMouseTracker is false when the annotation is not valid: its
	// callbacks are not called. It is false for a detached object.
	ValidForMouseTracker() bool
}

// MouseTrackerAnnotationOpts are the optional parameters of
// NewMouseTrackerAnnotation.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotation.new
type MouseTrackerAnnotationOpts struct {
	// OnEnter is called when a mouse pointer enters the region.
	OnEnter PointerEnterEventListener
	// OnExit is called when a mouse pointer leaves the region.
	OnExit PointerExitEventListener
	// Cursor is the cursor over the region; the default is MouseCursorDefer.
	Cursor MouseCursor
	// Invalid makes ValidForMouseTracker false. Flutter's default is true.
	Invalid bool
}

// MouseTrackerAnnotationData is the plain MouseTrackerAnnotation.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotation
type MouseTrackerAnnotationData struct {
	onEnter PointerEnterEventListener
	onExit  PointerExitEventListener
	cursor  MouseCursor
	valid   bool
}

// NewMouseTrackerAnnotation creates an annotation.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotation.new
func NewMouseTrackerAnnotation(opts ...MouseTrackerAnnotationOpts) *MouseTrackerAnnotationData {
	var o MouseTrackerAnnotationOpts
	if len(opts) > 1 {
		panic("services: more than one MouseTrackerAnnotationOpts")
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Cursor == nil {
		o.Cursor = MouseCursorDefer
	}
	return &MouseTrackerAnnotationData{onEnter: o.OnEnter, onExit: o.OnExit, cursor: o.Cursor, valid: !o.Invalid}
}

// OnEnter is the enter callback.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotationData.onEnter
func (a *MouseTrackerAnnotationData) OnEnter() PointerEnterEventListener { return a.onEnter }

// OnExit is the exit callback.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotationData.onExit
func (a *MouseTrackerAnnotationData) OnExit() PointerExitEventListener { return a.onExit }

// Cursor is the cursor of the region.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotationData.cursor
func (a *MouseTrackerAnnotationData) Cursor() MouseCursor { return a.cursor }

// ValidForMouseTracker is whether the callbacks are called.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotationData.validForMouseTracker
func (a *MouseTrackerAnnotationData) ValidForMouseTracker() bool { return a.valid }

// DebugFillProperties adds the callbacks and the cursor.
//
// Flutter: services/mouse_tracking.dart MouseTrackerAnnotation.debugFillProperties
func (a *MouseTrackerAnnotationData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagsSummary("callbacks", []foundation.FlagEntry{
		{Description: "enter", Present: a.onEnter != nil},
		{Description: "exit", Present: a.onExit != nil},
	}, foundation.FlagsSummaryOpts{IfEmpty: "<none>"}))
	p.Add(foundation.NewDiagnosticsProperty("cursor", a.cursor.DebugDescription(), foundation.DiagnosticsPropertyOpts{DefaultValue: MouseCursorDefer.DebugDescription()}))
}
