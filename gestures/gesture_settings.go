// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"fmt"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// DeviceGestureSettings are the gesture thresholds of the device, in logical
// pixels. A recognizer reads them in place of its default constants.
//
// Flutter: gestures/gesture_settings.dart DeviceGestureSettings
type DeviceGestureSettings struct {
	// TouchSlop is how far a touch may move before it is a drag; None if the
	// device gives none.
	//
	// Flutter: gestures/gesture_settings.dart DeviceGestureSettings.touchSlop
	TouchSlop foundation.Opt[float64]
}

// NewDeviceGestureSettings creates the settings with the touch slop.
//
// Flutter: gestures/gesture_settings.dart DeviceGestureSettings.new
func NewDeviceGestureSettings(touchSlop foundation.Opt[float64]) DeviceGestureSettings {
	return DeviceGestureSettings{TouchSlop: touchSlop}
}

// NewDeviceGestureSettingsFromView creates the settings from the platform's
// settings of the view, which are in physical pixels.
//
// Flutter: gestures/gesture_settings.dart DeviceGestureSettings.fromView
func NewDeviceGestureSettingsFromView(view *ui.View) DeviceGestureSettings {
	physical := view.GestureSettings().PhysicalTouchSlop
	if physical == 0 {
		return DeviceGestureSettings{}
	}
	return DeviceGestureSettings{TouchSlop: foundation.Some(physical / view.DevicePixelRatio())}
}

// PanSlop is twice the touch slop; None if there is no touch slop.
//
// Flutter: gestures/gesture_settings.dart DeviceGestureSettings.panSlop
func (s DeviceGestureSettings) PanSlop() foundation.Opt[float64] {
	if v, ok := s.TouchSlop.Get(); ok {
		return foundation.Some(v * 2)
	}
	return foundation.None[float64]()
}

// String describes the settings.
//
// Flutter: gestures/gesture_settings.dart DeviceGestureSettings.toString
func (s DeviceGestureSettings) String() string {
	v := "null"
	if t, ok := s.TouchSlop.Get(); ok {
		v = dartfmt.Double(t)
	}
	return fmt.Sprintf("DeviceGestureSettings(touchSlop: %s)", v)
}

// ComputeHitSlop is how far a pointer of the kind may move and still hit: a
// mouse is precise, and the others use the device's touch slop, or TouchSlop if
// it gives none.
//
// Flutter: gestures/events.dart computeHitSlop
func ComputeHitSlop(kind ui.PointerDeviceKind, settings foundation.Opt[DeviceGestureSettings]) float64 {
	if kind == ui.PointerDeviceKindMouse {
		return PrecisePointerHitSlop
	}
	if s, ok := settings.Get(); ok {
		return s.TouchSlop.OrElse(TouchSlop)
	}
	return TouchSlop
}

// ComputePanSlop is how far a pointer of the kind may move before it pans: a
// mouse is precise, and the others use the device's pan slop, or PanSlop if it
// gives none.
//
// Flutter: gestures/events.dart computePanSlop
func ComputePanSlop(kind ui.PointerDeviceKind, settings foundation.Opt[DeviceGestureSettings]) float64 {
	if kind == ui.PointerDeviceKindMouse {
		return PrecisePointerPanSlop
	}
	if s, ok := settings.Get(); ok {
		return s.PanSlop().OrElse(PanSlop)
	}
	return PanSlop
}

// ComputeScaleSlop is how far a pointer of the kind may move before it scales.
//
// Flutter: gestures/events.dart computeScaleSlop
func ComputeScaleSlop(kind ui.PointerDeviceKind) float64 {
	if kind == ui.PointerDeviceKindMouse {
		return PrecisePointerScaleSlop
	}
	return ScaleSlop
}
