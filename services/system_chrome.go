// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SystemUiOverlayStyle specifies a preference for the style of the system
// overlays (the status bar and the navigation bar). It is a value that
// widgets such as AppBar put in an AnnotatedRegion; Glutter's desktop
// platform layer does not act on it, and SystemChrome.setSystemUIOverlayStyle
// is not ported.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle
type SystemUiOverlayStyle struct {
	// SystemNavigationBarColor is the color of the system bottom navigation
	// bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.systemNavigationBarColor
	SystemNavigationBarColor foundation.Opt[ui.Color]
	// SystemNavigationBarDividerColor is the color of the divider between the
	// system bottom navigation bar and the content.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.systemNavigationBarDividerColor
	SystemNavigationBarDividerColor foundation.Opt[ui.Color]
	// SystemNavigationBarIconBrightness is the brightness of the system
	// navigation bar icons.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.systemNavigationBarIconBrightness
	SystemNavigationBarIconBrightness ui.Brightness
	// SystemNavigationBarContrastEnforced is whether the system should ensure
	// that the navigation bar has enough contrast.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.systemNavigationBarContrastEnforced
	SystemNavigationBarContrastEnforced foundation.Opt[bool]
	// StatusBarColor is the color of the top status bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.statusBarColor
	StatusBarColor foundation.Opt[ui.Color]
	// StatusBarBrightness is the brightness of the top status bar (iOS).
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.statusBarBrightness
	StatusBarBrightness ui.Brightness
	// StatusBarIconBrightness is the brightness of the top status bar icons
	// (Android).
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.statusBarIconBrightness
	StatusBarIconBrightness ui.Brightness
	// SystemStatusBarContrastEnforced is whether the system should ensure that
	// the status bar has enough contrast.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlayStyle.systemStatusBarContrastEnforced
	SystemStatusBarContrastEnforced foundation.Opt[bool]
}

// SystemUiOverlayStyleLight is a style for light system overlays: light icons
// on a dark background.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle.light
var SystemUiOverlayStyleLight = SystemUiOverlayStyle{
	SystemNavigationBarColor:          foundation.Some(ui.NewColor(0xFF000000)),
	SystemNavigationBarIconBrightness: ui.BrightnessLight,
	StatusBarIconBrightness:           ui.BrightnessLight,
	StatusBarBrightness:               ui.BrightnessDark,
}

// SystemUiOverlayStyleDark is a style for dark system overlays: dark icons on
// a light background.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle.dark
var SystemUiOverlayStyleDark = SystemUiOverlayStyle{
	SystemNavigationBarColor:          foundation.Some(ui.NewColor(0xFF000000)),
	SystemNavigationBarIconBrightness: ui.BrightnessLight,
	StatusBarIconBrightness:           ui.BrightnessDark,
	StatusBarBrightness:               ui.BrightnessLight,
}

// SystemUiOverlayStyleCopy holds the replacements for
// SystemUiOverlayStyle.CopyWith; an unset field keeps the value of the copied
// style.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle.copyWith
type SystemUiOverlayStyleCopy = SystemUiOverlayStyle

// CopyWith creates a copy of this style with the fields of c that are set
// (an Opt that is Some, a brightness that is not unspecified) replaced.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle.copyWith
func (s SystemUiOverlayStyle) CopyWith(c SystemUiOverlayStyleCopy) SystemUiOverlayStyle {
	pickB := func(n, old ui.Brightness) ui.Brightness {
		if n != ui.BrightnessUnspecified {
			return n
		}
		return old
	}
	return SystemUiOverlayStyle{
		SystemNavigationBarColor:            optOr(c.SystemNavigationBarColor, s.SystemNavigationBarColor),
		SystemNavigationBarDividerColor:     optOr(c.SystemNavigationBarDividerColor, s.SystemNavigationBarDividerColor),
		SystemNavigationBarIconBrightness:   pickB(c.SystemNavigationBarIconBrightness, s.SystemNavigationBarIconBrightness),
		SystemNavigationBarContrastEnforced: optOr(c.SystemNavigationBarContrastEnforced, s.SystemNavigationBarContrastEnforced),
		StatusBarColor:                      optOr(c.StatusBarColor, s.StatusBarColor),
		StatusBarBrightness:                 pickB(c.StatusBarBrightness, s.StatusBarBrightness),
		StatusBarIconBrightness:             pickB(c.StatusBarIconBrightness, s.StatusBarIconBrightness),
		SystemStatusBarContrastEnforced:     optOr(c.SystemStatusBarContrastEnforced, s.SystemStatusBarContrastEnforced),
	}
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SystemUiOverlayStyle) ToStringShort() string { return "SystemUiOverlayStyle" }

// String returns the description of the style.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (s SystemUiOverlayStyle) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties of the style.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle.debugFillProperties
func (s SystemUiOverlayStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewDiagnosticsPropertyOpt("systemNavigationBarColor", s.SystemNavigationBarColor))
	properties.Add(foundation.NewDiagnosticsPropertyOpt("systemNavigationBarDividerColor", s.SystemNavigationBarDividerColor))
	properties.Add(foundation.NewDiagnosticsProperty("systemNavigationBarIconBrightness", s.SystemNavigationBarIconBrightness))
	properties.Add(foundation.NewDiagnosticsPropertyOpt("systemNavigationBarContrastEnforced", s.SystemNavigationBarContrastEnforced))
	properties.Add(foundation.NewDiagnosticsPropertyOpt("statusBarColor", s.StatusBarColor))
	properties.Add(foundation.NewDiagnosticsProperty("statusBarBrightness", s.StatusBarBrightness))
	properties.Add(foundation.NewDiagnosticsProperty("statusBarIconBrightness", s.StatusBarIconBrightness))
	properties.Add(foundation.NewDiagnosticsPropertyOpt("systemStatusBarContrastEnforced", s.SystemStatusBarContrastEnforced))
}

// optOr returns n when it is set and old otherwise.
func optOr[T any](n, old foundation.Opt[T]) foundation.Opt[T] {
	if n.IsSome() {
		return n
	}
	return old
}
