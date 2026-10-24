// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import "github.com/rm4n0s/glutter/ui"

// HapticFeedbackBackend is what HapticFeedback asks of the engine. *ui.PlatformDispatcher is
// one; it forwards the call when the engine can vibrate (Android) and ignores it otherwise.
//
// Flutter: no counterpart (the flutter/platform channel of SystemChannels.platform)
type HapticFeedbackBackend interface {
	// PerformHapticFeedback gives the feedback.
	PerformHapticFeedback(kind ui.HapticFeedbackKind)
}

// HapticFeedback provides haptic feedback to the user: vibrations and the other tactile
// responses of the device. The binding makes one (WidgetsBinding.HapticFeedback). It does
// nothing where the device cannot give the feedback (the desktop and the web), and when it is
// nil or has no backend.
//
// Deviation: Flutter's HapticFeedback is a class of static methods that return a Future<void>;
// this one is an instance with a backend, and its methods return nothing. The iOS-only
// successNotification, warningNotification and errorNotification are not ported (SPEC §12.10).
//
// Flutter: services/haptic_feedback.dart HapticFeedback
type HapticFeedback struct {
	// OnPerform is called with the kind of each feedback that is given, whether or not the
	// device can give it (a test sets it to see the feedback).
	OnPerform func(ui.HapticFeedbackKind)

	backend HapticFeedbackBackend
}

// NewHapticFeedback creates a HapticFeedback that sends its calls to the backend, which may be
// nil.
//
// Flutter: services/haptic_feedback.dart HapticFeedback
func NewHapticFeedback(backend HapticFeedbackBackend) *HapticFeedback {
	return &HapticFeedback{backend: backend}
}

func (h *HapticFeedback) perform(kind ui.HapticFeedbackKind) {
	if h == nil {
		return
	}
	if h.OnPerform != nil {
		h.OnPerform(kind)
	}
	if h.backend != nil {
		h.backend.PerformHapticFeedback(kind)
	}
}

// Vibrate provides haptic feedback to the user for a short duration. On Android it vibrates for
// the platform's default length (on iOS it would be a heavy impact).
//
// Flutter: services/haptic_feedback.dart HapticFeedback.vibrate
func (h *HapticFeedback) Vibrate() { h.perform(ui.HapticFeedbackVibrate) }

// LightImpact provides a haptic feedback corresponding to a collision impact with a light mass.
//
// Flutter: services/haptic_feedback.dart HapticFeedback.lightImpact
func (h *HapticFeedback) LightImpact() { h.perform(ui.HapticFeedbackLightImpact) }

// MediumImpact provides a haptic feedback corresponding to a collision impact with a medium
// mass.
//
// Flutter: services/haptic_feedback.dart HapticFeedback.mediumImpact
func (h *HapticFeedback) MediumImpact() { h.perform(ui.HapticFeedbackMediumImpact) }

// HeavyImpact provides a haptic feedback corresponding to a collision impact with a heavy mass.
//
// Flutter: services/haptic_feedback.dart HapticFeedback.heavyImpact
func (h *HapticFeedback) HeavyImpact() { h.perform(ui.HapticFeedbackHeavyImpact) }

// SelectionClick provides a haptic feedback indication selection changing through discrete
// values.
//
// Flutter: services/haptic_feedback.dart HapticFeedback.selectionClick
func (h *HapticFeedback) SelectionClick() { h.perform(ui.HapticFeedbackSelectionClick) }
