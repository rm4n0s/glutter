// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"slices"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// DeviceOrientation is an orientation of the device's screen.
//
// Flutter: services/system_chrome.dart DeviceOrientation
type DeviceOrientation int

// The values of DeviceOrientation follow Flutter's declaration order.
const (
	// DeviceOrientationUnspecified is the zero value.
	//
	// Flutter: no counterpart
	DeviceOrientationUnspecified DeviceOrientation = iota
	// DeviceOrientationPortraitUp is upright portrait.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.portraitUp
	DeviceOrientationPortraitUp
	// DeviceOrientationLandscapeLeft is landscape, turned to the left.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.landscapeLeft
	DeviceOrientationLandscapeLeft
	// DeviceOrientationPortraitDown is upside-down portrait.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.portraitDown
	DeviceOrientationPortraitDown
	// DeviceOrientationLandscapeRight is landscape, turned to the right.
	//
	// Flutter: services/system_chrome.dart DeviceOrientation.landscapeRight
	DeviceOrientationLandscapeRight
)

// String is the name of the orientation, like Dart's enum toString.
//
// Flutter: services/system_chrome.dart DeviceOrientation
func (o DeviceOrientation) String() string {
	switch o {
	case DeviceOrientationPortraitUp:
		return "DeviceOrientation.portraitUp"
	case DeviceOrientationLandscapeLeft:
		return "DeviceOrientation.landscapeLeft"
	case DeviceOrientationPortraitDown:
		return "DeviceOrientation.portraitDown"
	case DeviceOrientationLandscapeRight:
		return "DeviceOrientation.landscapeRight"
	}
	return "DeviceOrientation.unspecified"
}

// SystemUiOverlay is a bar of the system's interface.
//
// Flutter: services/system_chrome.dart SystemUiOverlay
type SystemUiOverlay int

// The values of SystemUiOverlay follow Flutter's declaration order.
const (
	// SystemUiOverlayUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemUiOverlayUnspecified SystemUiOverlay = iota
	// SystemUiOverlayTop is the status bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlay.top
	SystemUiOverlayTop
	// SystemUiOverlayBottom is the navigation bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlay.bottom
	SystemUiOverlayBottom
)

// String is the name of the overlay, like Dart's enum toString.
//
// Flutter: services/system_chrome.dart SystemUiOverlay
func (o SystemUiOverlay) String() string {
	switch o {
	case SystemUiOverlayTop:
		return "SystemUiOverlay.top"
	case SystemUiOverlayBottom:
		return "SystemUiOverlay.bottom"
	}
	return "SystemUiOverlay.unspecified"
}

// SystemUiMode is how much of the system's interface (the status bar and the
// navigation bar) the app shows.
//
// Flutter: services/system_chrome.dart SystemUiMode
type SystemUiMode int

// The values of SystemUiMode follow Flutter's declaration order.
const (
	// SystemUiModeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemUiModeUnspecified SystemUiMode = iota
	// SystemUiModeLeanBack hides the bars until the user taps the screen.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.leanBack
	SystemUiModeLeanBack
	// SystemUiModeImmersive hides the bars until the user swipes from an edge.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.immersive
	SystemUiModeImmersive
	// SystemUiModeImmersiveSticky is immersive, and the bars hide again by
	// themselves.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.immersiveSticky
	SystemUiModeImmersiveSticky
	// SystemUiModeEdgeToEdge draws behind transparent bars.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.edgeToEdge
	SystemUiModeEdgeToEdge
	// SystemUiModeManual shows the overlays that are listed.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.manual
	SystemUiModeManual
)

// String is the name of the mode, like Dart's enum toString.
//
// Flutter: services/system_chrome.dart SystemUiMode
func (m SystemUiMode) String() string {
	switch m {
	case SystemUiModeLeanBack:
		return "SystemUiMode.leanBack"
	case SystemUiModeImmersive:
		return "SystemUiMode.immersive"
	case SystemUiModeImmersiveSticky:
		return "SystemUiMode.immersiveSticky"
	case SystemUiModeEdgeToEdge:
		return "SystemUiMode.edgeToEdge"
	case SystemUiModeManual:
		return "SystemUiMode.manual"
	}
	return "SystemUiMode.unspecified"
}

// SystemUiChangeCallback is called when the system shows (true) or hides (false)
// its bars because of the user. It is called on the UI thread.
//
// Deviation: Flutter's callback returns a Future<void> (SPEC §12.10).
//
// Flutter: services/binding.dart SystemUiChangeCallback
type SystemUiChangeCallback func(systemOverlaysAreVisible bool)

// SystemUiOverlayStyle specifies a preference for the style of the system
// overlays (the status bar and the navigation bar). It is a value that
// widgets such as AppBar put in an AnnotatedRegion, and that
// SystemChrome.SetSystemUIOverlayStyle sends to the platform. Only Android acts
// on it; the desktop and the web ignore it, as Flutter does there.
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

// toUI converts the style to the engine's, where a nil pointer or an unspecified brightness is
// a value the app does not say.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle._toMap
func (s SystemUiOverlayStyle) toUI() ui.SystemUIOverlayStyle {
	color := func(o foundation.Opt[ui.Color]) *ui.Color {
		if c, ok := o.Get(); ok {
			return &c
		}
		return nil
	}
	flag := func(o foundation.Opt[bool]) *bool {
		if b, ok := o.Get(); ok {
			return &b
		}
		return nil
	}
	return ui.SystemUIOverlayStyle{
		SystemNavigationBarColor:            color(s.SystemNavigationBarColor),
		SystemNavigationBarDividerColor:     color(s.SystemNavigationBarDividerColor),
		SystemNavigationBarIconBrightness:   s.SystemNavigationBarIconBrightness,
		SystemNavigationBarContrastEnforced: flag(s.SystemNavigationBarContrastEnforced),
		StatusBarColor:                      color(s.StatusBarColor),
		StatusBarBrightness:                 s.StatusBarBrightness,
		StatusBarIconBrightness:             s.StatusBarIconBrightness,
		SystemStatusBarContrastEnforced:     flag(s.SystemStatusBarContrastEnforced),
	}
}

// SystemChromeBackend is what SystemChrome asks of the engine. *ui.PlatformDispatcher is one;
// it forwards the calls to the engine when the engine can control the system's interface
// (Android) and ignores them otherwise.
//
// Flutter: no counterpart (the flutter/platform channel of SystemChannels.platform)
type SystemChromeBackend interface {
	// SetPreferredOrientations says which orientations the app can be in.
	SetPreferredOrientations(orientations []ui.DeviceOrientation)
	// SetEnabledSystemUIMode sets how much of the system's interface shows.
	SetEnabledSystemUIMode(mode ui.SystemUIMode, overlays []ui.SystemUIOverlay)
	// SetSystemUIOverlayStyle sets the look of the bars.
	SetSystemUIOverlayStyle(style ui.SystemUIOverlayStyle)
	// SetSystemUIChangeCallback sets the function that hears of the bars changing; nil removes it.
	SetSystemUIChangeCallback(fn func(visible bool))
	// Post runs fn on the UI thread after the work that is running now (Dart's scheduleMicrotask).
	Post(fn func())
}

// SystemChrome controls specific aspects of the operating system's graphical interface and how
// it interacts with the application: the orientations the app can have, the status bar and the
// navigation bar. The binding makes one (WidgetsBinding.SystemChrome). A nil SystemChrome, and
// one with no backend, do nothing.
//
// Deviation: Flutter's SystemChrome is a class of static methods that call the flutter/platform
// channel; this one is an instance with a backend, and the calls are not asynchronous
// (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome
type SystemChrome struct {
	backend SystemChromeBackend

	pending    SystemUiOverlayStyle
	hasPending bool
	latest     SystemUiOverlayStyle
	hasLatest  bool

	mode    SystemUiMode
	overlay []SystemUiOverlay
}

// NewSystemChrome creates a SystemChrome that sends its calls to the backend, which may be nil.
//
// Flutter: services/system_chrome.dart SystemChrome
func NewSystemChrome(backend SystemChromeBackend) *SystemChrome {
	return &SystemChrome{backend: backend}
}

func (c *SystemChrome) ready() bool { return c != nil && c.backend != nil }

// SetPreferredOrientations specifies the set of orientations the application interface can be
// displayed in. An empty list lets the system choose.
//
// Flutter: services/system_chrome.dart SystemChrome.setPreferredOrientations
func (c *SystemChrome) SetPreferredOrientations(orientations []DeviceOrientation) {
	if !c.ready() {
		return
	}
	out := make([]ui.DeviceOrientation, 0, len(orientations))
	for _, o := range orientations {
		out = append(out, ui.DeviceOrientation(o))
	}
	c.backend.SetPreferredOrientations(out)
}

// SetEnabledSystemUIMode specifies the set of system overlays to have visible when the
// application is running. Overlays are the bars that stay for SystemUiModeManual; the other
// modes ignore them.
//
// Deviation: Flutter asserts in debug mode that SystemUiModeManual comes with a list; here no
// list shows no bars (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setEnabledSystemUIMode
func (c *SystemChrome) SetEnabledSystemUIMode(mode SystemUiMode, overlays []SystemUiOverlay) {
	if !c.ready() {
		return
	}
	c.mode = mode
	c.overlay = nil
	var out []ui.SystemUIOverlay
	if mode == SystemUiModeManual {
		c.overlay = slices.Clone(overlays)
		out = make([]ui.SystemUIOverlay, 0, len(overlays))
		for _, o := range overlays {
			out = append(out, ui.SystemUIOverlay(o))
		}
	}
	c.backend.SetEnabledSystemUIMode(ui.SystemUIMode(mode), out)
}

// SetSystemUIChangeCallback sets the function that is called when the system shows or hides its
// bars because of the user (for example, a swipe from the edge in an immersive mode). A nil
// callback removes it.
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIChangeCallback
func (c *SystemChrome) SetSystemUIChangeCallback(callback SystemUiChangeCallback) {
	if !c.ready() {
		return
	}
	c.backend.SetSystemUIChangeCallback(callback)
}

// RestoreSystemUIOverlays restores the system overlays to the last settings provided by
// SetEnabledSystemUIMode, for when the user's interaction with the system brought them back. It
// does nothing when the mode was never set.
//
// Deviation: Flutter asks the platform plugin to apply what it remembers; here the mode is
// remembered by SystemChrome and sent again (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.restoreSystemUIOverlays
func (c *SystemChrome) RestoreSystemUIOverlays() {
	if !c.ready() || c.mode == SystemUiModeUnspecified {
		return
	}
	c.SetEnabledSystemUIMode(c.mode, c.overlay)
}

// SetSystemUIOverlayStyle specifies the style of the system overlays that are visible. The call
// is debounced: the style is sent after the work that is running now, and only when it differs
// from the last one sent, so many calls in one frame (one per AnnotatedRegion) cost one.
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIOverlayStyle
func (c *SystemChrome) SetSystemUIOverlayStyle(style SystemUiOverlayStyle) {
	if !c.ready() {
		return
	}
	if c.hasPending {
		// The microtask has already been queued; just update the pending value.
		c.pending = style
		return
	}
	if c.hasLatest && style == c.latest {
		// Trivial success: no microtask has been queued and the given style is already in
		// effect.
		return
	}
	c.pending, c.hasPending = style, true
	c.backend.Post(func() {
		if !c.hasLatest || c.pending != c.latest {
			c.backend.SetSystemUIOverlayStyle(c.pending.toUI())
			c.latest, c.hasLatest = c.pending, true
		}
		c.pending, c.hasPending = SystemUiOverlayStyle{}, false
	})
}

// HandleAppLifecycleStateChanged tells SystemChrome of a change of the lifecycle. The binding
// calls it. When the app is detached, the record of the style sent to the host is cleared so
// that the style is sent again when the app is attached again.
//
// Flutter: services/system_chrome.dart SystemChrome.handleAppLifecycleStateChanged
func (c *SystemChrome) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	if !c.ready() || state != ui.AppLifecycleStateDetached {
		return
	}
	c.backend.Post(func() {
		c.latest, c.hasLatest = SystemUiOverlayStyle{}, false
	})
}

// LatestStyle is the style that was last sent to the host, and false when none was sent (or the
// app detached since).
//
// Flutter: services/system_chrome.dart SystemChrome.latestStyle
func (c *SystemChrome) LatestStyle() (SystemUiOverlayStyle, bool) {
	if c == nil {
		return SystemUiOverlayStyle{}, false
	}
	return c.latest, c.hasLatest
}
