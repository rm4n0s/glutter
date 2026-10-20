// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import "time"

// The thresholds of the gestures, modeled after Android's ViewConfiguration.
// Distances are in logical pixels.
const (
	// PressTimeout is how long a pointer must stay down before the press is
	// a tap-down (the pressed state of a button).
	//
	// Flutter: gestures/constants.dart kPressTimeout
	PressTimeout = 100 * time.Millisecond
	// HoverTapTimeout is the time within which a hover tap would happen.
	//
	// Flutter: gestures/constants.dart kHoverTapTimeout
	HoverTapTimeout = 150 * time.Millisecond
	// HoverTapSlop is the distance a hover tap may move.
	//
	// Flutter: gestures/constants.dart kHoverTapSlop
	HoverTapSlop = 20.0
	// LongPressTimeout is how long a pointer must stay down for a long press.
	//
	// Flutter: gestures/constants.dart kLongPressTimeout
	LongPressTimeout = 500 * time.Millisecond
	// DoubleTapTimeout is the longest time between two taps of a double tap.
	//
	// Flutter: gestures/constants.dart kDoubleTapTimeout
	DoubleTapTimeout = 300 * time.Millisecond
	// DoubleTapMinTime is the shortest time between two taps of a double
	// tap.
	//
	// Flutter: gestures/constants.dart kDoubleTapMinTime
	DoubleTapMinTime = 40 * time.Millisecond
	// DoubleTapTouchSlop is the distance between the taps of a double tap
	// that a touch may move.
	//
	// Flutter: gestures/constants.dart kDoubleTapTouchSlop
	DoubleTapTouchSlop = TouchSlop
	// DoubleTapSlop is the distance between the two taps of a double tap.
	//
	// Flutter: gestures/constants.dart kDoubleTapSlop
	DoubleTapSlop = 100.0
	// ZoomControlsTimeout is how long the zoom controls stay visible.
	//
	// Flutter: gestures/constants.dart kZoomControlsTimeout
	ZoomControlsTimeout = 3000 * time.Millisecond
	// TouchSlop is how far a touch may move before it is a pan or drag. The
	// value was derived by experiment.
	//
	// Flutter: gestures/constants.dart kTouchSlop
	TouchSlop = 18.0
	// PagingTouchSlop is how far a touch may move before it pages.
	//
	// Flutter: gestures/constants.dart kPagingTouchSlop
	PagingTouchSlop = TouchSlop * 2.0
	// PanSlop is how far a touch may move before it is a pan.
	//
	// Flutter: gestures/constants.dart kPanSlop
	PanSlop = TouchSlop * 2.0
	// ScaleSlop is how far a touch may move before it is a scale.
	//
	// Flutter: gestures/constants.dart kScaleSlop
	ScaleSlop = TouchSlop
	// WindowTouchSlop is how far a touch may move outside a window before it
	// dismisses it.
	//
	// Flutter: gestures/constants.dart kWindowTouchSlop
	WindowTouchSlop = 16.0
	// MinFlingVelocity is the slowest fling, in logical pixels per second.
	//
	// Flutter: gestures/constants.dart kMinFlingVelocity
	MinFlingVelocity = 50.0
	// MaxFlingVelocity is the fastest fling, in logical pixels per second.
	//
	// Flutter: gestures/constants.dart kMaxFlingVelocity
	MaxFlingVelocity = 8000.0
	// JumpTapTimeout is the time within which a jump tap would happen.
	//
	// Flutter: gestures/constants.dart kJumpTapTimeout
	JumpTapTimeout = 500 * time.Millisecond
	// PrecisePointerHitSlop is the slop of a precise pointer, such as a mouse.
	//
	// Flutter: gestures/constants.dart kPrecisePointerHitSlop
	PrecisePointerHitSlop = 1.0
	// PrecisePointerPanSlop is the pan slop of a precise pointer.
	//
	// Flutter: gestures/constants.dart kPrecisePointerPanSlop
	PrecisePointerPanSlop = PrecisePointerHitSlop * 2.0
	// PrecisePointerScaleSlop is the scale slop of a precise pointer.
	//
	// Flutter: gestures/constants.dart kPrecisePointerScaleSlop
	PrecisePointerScaleSlop = PrecisePointerHitSlop
)
