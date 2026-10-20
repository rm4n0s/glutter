// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
)

// DurationsCollection is the type of Durations.
//
// Flutter: material/motion.dart Durations
type DurationsCollection struct {
	// Short1 is the short1 duration in the Material specification: 50 ms.
	Short1 time.Duration
	// Short2 is the short2 duration: 100 ms.
	Short2 time.Duration
	// Short3 is the short3 duration: 150 ms.
	Short3 time.Duration
	// Short4 is the short4 duration: 200 ms.
	Short4 time.Duration
	// Medium1 is the medium1 duration: 250 ms.
	Medium1 time.Duration
	// Medium2 is the medium2 duration: 300 ms.
	Medium2 time.Duration
	// Medium3 is the medium3 duration: 350 ms.
	Medium3 time.Duration
	// Medium4 is the medium4 duration: 400 ms.
	Medium4 time.Duration
	// Long1 is the long1 duration: 450 ms.
	Long1 time.Duration
	// Long2 is the long2 duration: 500 ms.
	Long2 time.Duration
	// Long3 is the long3 duration: 550 ms.
	Long3 time.Duration
	// Long4 is the long4 duration: 600 ms.
	Long4 time.Duration
	// Extralong1 is the extralong1 duration: 700 ms.
	Extralong1 time.Duration
	// Extralong2 is the extralong2 duration: 800 ms.
	Extralong2 time.Duration
	// Extralong3 is the extralong3 duration: 900 ms.
	Extralong3 time.Duration
	// Extralong4 is the extralong4 duration: 1000 ms.
	Extralong4 time.Duration
}

// Durations are the duration tokens of the Material specification.
//
// Flutter: material/motion.dart Durations
var Durations = DurationsCollection{
	Short1: 50 * time.Millisecond, Short2: 100 * time.Millisecond, Short3: 150 * time.Millisecond, Short4: 200 * time.Millisecond,
	Medium1: 250 * time.Millisecond, Medium2: 300 * time.Millisecond, Medium3: 350 * time.Millisecond, Medium4: 400 * time.Millisecond,
	Long1: 450 * time.Millisecond, Long2: 500 * time.Millisecond, Long3: 550 * time.Millisecond, Long4: 600 * time.Millisecond,
	Extralong1: 700 * time.Millisecond, Extralong2: 800 * time.Millisecond, Extralong3: 900 * time.Millisecond, Extralong4: 1000 * time.Millisecond,
}

// EasingCollection is the type of Easing.
//
// Flutter: material/motion.dart Easing
type EasingCollection struct {
	// EmphasizedAccelerate is the emphasizedAccelerate easing curve.
	EmphasizedAccelerate animation.Curve
	// EmphasizedDecelerate is the emphasizedDecelerate easing curve.
	EmphasizedDecelerate animation.Curve
	// Linear is the linear easing curve.
	Linear animation.Curve
	// Standard is the standard easing curve.
	Standard animation.Curve
	// StandardAccelerate is the standardAccelerate easing curve.
	StandardAccelerate animation.Curve
	// StandardDecelerate is the standardDecelerate easing curve.
	StandardDecelerate animation.Curve
	// LegacyDecelerate is the legacyDecelerate easing curve.
	LegacyDecelerate animation.Curve
	// LegacyAccelerate is the legacyAccelerate easing curve.
	LegacyAccelerate animation.Curve
	// Legacy is the legacy easing curve.
	Legacy animation.Curve
}

// Easing are the easing curve tokens of the Material specification.
//
// Flutter: material/motion.dart Easing
var Easing = EasingCollection{
	EmphasizedAccelerate: animation.NewCubic(0.3, 0.0, 0.8, 0.15),
	EmphasizedDecelerate: animation.NewCubic(0.05, 0.7, 0.1, 1.0),
	Linear:               animation.NewCubic(0.0, 0.0, 1.0, 1.0),
	Standard:             animation.NewCubic(0.2, 0.0, 0.0, 1.0),
	StandardAccelerate:   animation.NewCubic(0.3, 0.0, 1.0, 1.0),
	StandardDecelerate:   animation.NewCubic(0.0, 0.0, 0.0, 1.0),
	LegacyDecelerate:     animation.NewCubic(0.0, 0.0, 0.2, 1.0),
	LegacyAccelerate:     animation.NewCubic(0.4, 0.0, 1.0, 1.0),
	Legacy:               animation.NewCubic(0.4, 0.0, 0.2, 1.0),
}
