package ui

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

// SystemUIMode is how much of the system's interface (the status bar and the navigation bar)
// the app shows.
//
// Flutter: services/system_chrome.dart SystemUiMode
type SystemUIMode int

// The values of SystemUIMode follow Flutter's declaration order.
const (
	// SystemUIModeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemUIModeUnspecified SystemUIMode = iota
	// SystemUIModeLeanBack hides the bars until the user taps the screen.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.leanBack
	SystemUIModeLeanBack
	// SystemUIModeImmersive hides the bars until the user swipes from an edge.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.immersive
	SystemUIModeImmersive
	// SystemUIModeImmersiveSticky is immersive, and the bars hide again by themselves.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.immersiveSticky
	SystemUIModeImmersiveSticky
	// SystemUIModeEdgeToEdge draws behind transparent bars.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.edgeToEdge
	SystemUIModeEdgeToEdge
	// SystemUIModeManual shows the overlays that are listed.
	//
	// Flutter: services/system_chrome.dart SystemUiMode.manual
	SystemUIModeManual
)

// SystemUIOverlay is a bar of the system's interface.
//
// Flutter: services/system_chrome.dart SystemUiOverlay
type SystemUIOverlay int

// The values of SystemUIOverlay follow Flutter's declaration order.
const (
	// SystemUIOverlayUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemUIOverlayUnspecified SystemUIOverlay = iota
	// SystemUIOverlayTop is the status bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlay.top
	SystemUIOverlayTop
	// SystemUIOverlayBottom is the navigation bar.
	//
	// Flutter: services/system_chrome.dart SystemUiOverlay.bottom
	SystemUIOverlayBottom
)

// SystemUIOverlayStyle is the look of the status bar and the navigation bar. A nil color or flag
// is one that the app does not say: the platform keeps what it has.
//
// Flutter: services/system_chrome.dart SystemUiOverlayStyle
type SystemUIOverlayStyle struct {
	// SystemNavigationBarColor is the color of the navigation bar.
	SystemNavigationBarColor *Color
	// SystemNavigationBarDividerColor is the color of the divider above it.
	SystemNavigationBarDividerColor *Color
	// SystemNavigationBarIconBrightness is the brightness of its icons.
	SystemNavigationBarIconBrightness Brightness
	// SystemNavigationBarContrastEnforced is whether the system makes sure it has contrast.
	SystemNavigationBarContrastEnforced *bool
	// StatusBarColor is the color of the status bar.
	StatusBarColor *Color
	// StatusBarBrightness is the brightness of the status bar (iOS).
	StatusBarBrightness Brightness
	// StatusBarIconBrightness is the brightness of the status bar's icons.
	StatusBarIconBrightness Brightness
	// SystemStatusBarContrastEnforced is whether the system makes sure it has contrast.
	SystemStatusBarContrastEnforced *bool
}

// HapticFeedbackKind is a kind of haptic feedback.
//
// Flutter: services/haptic_feedback.dart HapticFeedback
type HapticFeedbackKind int

// The values of HapticFeedbackKind.
const (
	// HapticFeedbackUnspecified is the zero value.
	//
	// Flutter: no counterpart
	HapticFeedbackUnspecified HapticFeedbackKind = iota
	// HapticFeedbackVibrate is a vibration of the platform's default length.
	//
	// Flutter: services/haptic_feedback.dart HapticFeedback.vibrate
	HapticFeedbackVibrate
	// HapticFeedbackLightImpact is a light impact.
	//
	// Flutter: services/haptic_feedback.dart HapticFeedback.lightImpact
	HapticFeedbackLightImpact
	// HapticFeedbackMediumImpact is a medium impact.
	//
	// Flutter: services/haptic_feedback.dart HapticFeedback.mediumImpact
	HapticFeedbackMediumImpact
	// HapticFeedbackHeavyImpact is a heavy impact.
	//
	// Flutter: services/haptic_feedback.dart HapticFeedback.heavyImpact
	HapticFeedbackHeavyImpact
	// HapticFeedbackSelectionClick is the tick of a selection that changes.
	//
	// Flutter: services/haptic_feedback.dart HapticFeedback.selectionClick
	HapticFeedbackSelectionClick
)

// SystemSoundKind is a sound of the system.
//
// Flutter: services/system_sound.dart SystemSoundType
type SystemSoundKind int

// The values of SystemSoundKind.
const (
	// SystemSoundUnspecified is the zero value.
	//
	// Flutter: no counterpart
	SystemSoundUnspecified SystemSoundKind = iota
	// SystemSoundClick is the sound of a click.
	//
	// Flutter: services/system_sound.dart SystemSoundType.click
	SystemSoundClick
	// SystemSoundTick is the sound of a tick.
	//
	// Flutter: services/system_sound.dart SystemSoundType.tick
	SystemSoundTick
	// SystemSoundAlert is the sound of an alert.
	//
	// Flutter: services/system_sound.dart SystemSoundType.alert
	SystemSoundAlert
)

// SystemChromeHost is what the engine implements to control the system's interface around the
// app, to give haptic feedback and to play the sounds of the system. The dispatcher uses it when
// the host has it; a host without it (the desktop and the web) does nothing, as Flutter does
// there.
//
// Flutter: no counterpart (the engine's side of the flutter/platform channel)
type SystemChromeHost interface {
	// SetPreferredOrientations says which orientations the app can be in; an empty list lets
	// the system choose.
	SetPreferredOrientations(orientations []DeviceOrientation)
	// SetEnabledSystemUIMode sets how much of the system's interface shows; overlays are the
	// bars that stay for SystemUIModeManual.
	SetEnabledSystemUIMode(mode SystemUIMode, overlays []SystemUIOverlay)
	// SetSystemUIOverlayStyle sets the look of the bars.
	SetSystemUIOverlayStyle(style SystemUIOverlayStyle)
	// SetSystemUIChangeCallback sets the function that is called, on the UI thread, when the
	// system shows (true) or hides (false) its bars because of the user; nil removes it.
	SetSystemUIChangeCallback(fn func(visible bool))
	// PerformHapticFeedback gives the feedback.
	PerformHapticFeedback(kind HapticFeedbackKind)
	// PlaySystemSound plays the sound.
	PlaySystemSound(kind SystemSoundKind)
}

// SetPreferredOrientations asks the engine to limit the orientations of the app. Nothing happens
// when the host cannot.
//
// Deviation: Flutter sends it as the SystemChrome.setPreferredOrientations call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setPreferredOrientations
func (d *PlatformDispatcher) SetPreferredOrientations(orientations []DeviceOrientation) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.SetPreferredOrientations(orientations)
	}
}

// SetEnabledSystemUIMode asks the engine to show or hide the system's bars. Nothing happens when
// the host cannot.
//
// Deviation: Flutter sends it as the SystemChrome.setEnabledSystemUIMode call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setEnabledSystemUIMode
func (d *PlatformDispatcher) SetEnabledSystemUIMode(mode SystemUIMode, overlays []SystemUIOverlay) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.SetEnabledSystemUIMode(mode, overlays)
	}
}

// SetSystemUIOverlayStyle asks the engine to set the look of the system's bars. Nothing happens
// when the host cannot.
//
// Deviation: Flutter sends it as the SystemChrome.setSystemUIOverlayStyle call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIOverlayStyle
func (d *PlatformDispatcher) SetSystemUIOverlayStyle(style SystemUIOverlayStyle) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.SetSystemUIOverlayStyle(style)
	}
}

// SetSystemUIChangeCallback sets the function that hears of the system showing or hiding its bars.
// Nothing happens when the host cannot.
//
// Deviation: Flutter sends it as the SystemChrome.setSystemUIChangeListener call of the
// flutter/platform channel and receives the changes as method calls (SPEC §12.10).
//
// Flutter: services/system_chrome.dart SystemChrome.setSystemUIChangeCallback
func (d *PlatformDispatcher) SetSystemUIChangeCallback(fn func(visible bool)) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.SetSystemUIChangeCallback(fn)
	}
}

// PerformHapticFeedback asks the engine for haptic feedback. Nothing happens when the host cannot.
//
// Deviation: Flutter sends it as the HapticFeedback.vibrate call of the flutter/platform channel
// (SPEC §12.10).
//
// Flutter: services/haptic_feedback.dart HapticFeedback
func (d *PlatformDispatcher) PerformHapticFeedback(kind HapticFeedbackKind) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.PerformHapticFeedback(kind)
	}
}

// PlaySystemSound asks the engine to play a sound of the system. Nothing happens when the host
// cannot.
//
// Deviation: Flutter sends it as the SystemSound.play call of the flutter/platform channel
// (SPEC §12.10).
//
// Flutter: services/system_sound.dart SystemSound.play
func (d *PlatformDispatcher) PlaySystemSound(kind SystemSoundKind) {
	if h, ok := d.host.(SystemChromeHost); ok {
		h.PlaySystemSound(kind)
	}
}
