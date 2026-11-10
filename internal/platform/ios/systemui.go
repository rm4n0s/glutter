package ios

import (
	"slices"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the system UI and the services of step 35.6 (SPEC §30.3, §30.6): what the app's calls mean in
// UIKit's numbers. The calls to UIKit are in systemui_ios.go and haptics_ios.go; the tests run on every host.

// OrientationAll is every interface orientation (UIInterfaceOrientationMaskAll).
//
// Flutter: no counterpart
const OrientationAll = OrientationPortrait | OrientationPortraitUpsideDown | OrientationLandscape

// orientationMaskOf is the mask of the orientations that the app prefers, as FlutterPlatformPlugin does it: no orientation at all
// lets every one through, and the mask is not ok when the list has none that iOS knows (the call then does nothing).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm setSystemChromePreferredOrientations
func orientationMaskOf(orientations []ui.DeviceOrientation) (mask Orientation, ok bool) {
	if len(orientations) == 0 {
		return OrientationAll, true
	}
	for _, o := range orientations {
		switch o {
		case ui.DeviceOrientationPortraitUp:
			mask |= OrientationPortrait
		case ui.DeviceOrientationPortraitDown:
			mask |= OrientationPortraitUpsideDown
		case ui.DeviceOrientationLandscapeLeft:
			mask |= OrientationLandscapeLeft
		case ui.DeviceOrientationLandscapeRight:
			mask |= OrientationLandscapeRight
		}
	}
	return mask, mask != 0
}

// The values of UIStatusBarStyle that the app uses.
const (
	statusBarDefault      int64 = 0
	statusBarLightContent int64 = 1
	statusBarDarkContent  int64 = 3
)

// statusBarStyleOf is the style of the status bar for the brightness of the app's overlay style: a dark status bar has light content
// and a light one has dark content. The style is not ok when the app does not say (the status bar keeps what it has).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm setSystemChromeSystemUIOverlayStyle
func statusBarStyleOf(statusBarBrightness ui.Brightness) (style int64, ok bool) {
	switch statusBarBrightness {
	case ui.BrightnessDark:
		return statusBarLightContent, true
	case ui.BrightnessLight:
		return statusBarDarkContent, true
	}
	return statusBarDefault, false
}

// chrome is what the app asked of the system's interface around it, which the view controller answers UIKit with.
type chrome struct {
	// orientations is the mask of the supported orientations; 0 until the app asks (the view controller then says every one).
	orientations Orientation
	// style is the style of the status bar, and styleSet says the app asked for one (otherwise the system's default follows the
	// interface style).
	style    int64
	styleSet bool
	// hideStatusBar and hideHomeIndicator are the system UI mode: the status bar is hidden, and the home indicator hides itself.
	hideStatusBar, hideHomeIndicator bool
}

// withMode is the chrome after SetEnabledSystemUIMode. Only the top overlay (the status bar) and the bottom one (the home indicator)
// exist on iOS: edge to edge shows both, the modes that hide the bars hide both, and the manual mode shows the ones that are listed.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm setSystemChromeEnabledSystemUIMode, setSystemChromeEnabledSystemUIOverlays
func (c chrome) withMode(mode ui.SystemUIMode, overlays []ui.SystemUIOverlay) chrome {
	switch mode {
	case ui.SystemUIModeManual:
		c.hideStatusBar = !slices.Contains(overlays, ui.SystemUIOverlayTop)
		c.hideHomeIndicator = !slices.Contains(overlays, ui.SystemUIOverlayBottom)
	case ui.SystemUIModeLeanBack, ui.SystemUIModeImmersive, ui.SystemUIModeImmersiveSticky:
		c.hideStatusBar, c.hideHomeIndicator = true, true
	default:
		c.hideStatusBar, c.hideHomeIndicator = false, false
	}
	return c
}

// withStyle is the chrome after SetSystemUIOverlayStyle: the status bar's brightness is the only part of the style that iOS has.
func (c chrome) withStyle(style ui.SystemUIOverlayStyle) chrome {
	if s, ok := statusBarStyleOf(style.StatusBarBrightness); ok {
		c.style, c.styleSet = s, true
	}
	return c
}

// withOrientations is the chrome after SetPreferredOrientations; ok is false when the call does nothing.
func (c chrome) withOrientations(orientations []ui.DeviceOrientation) (chrome, bool) {
	mask, ok := orientationMaskOf(orientations)
	if !ok {
		return c, false
	}
	c.orientations = mask
	return c, true
}

// supported is the mask that the view controller answers supportedInterfaceOrientations with.
func (c chrome) supported() Orientation {
	if c.orientations == 0 {
		return OrientationAll
	}
	return c.orientations
}

// hapticKind is how a haptic feedback is made.
type hapticKind int

// The ways to give a haptic feedback: a generator of impacts, the selection generator, or the vibration of the system sound.
const (
	hapticNone hapticKind = iota
	hapticImpact
	hapticSelection
	hapticVibrate
)

// The values of UIImpactFeedbackStyle.
const (
	impactLight  int64 = 0
	impactMedium int64 = 1
	impactHeavy  int64 = 2
)

// haptic is a haptic feedback in UIKit's terms: the way, and the style of an impact.
type haptic struct {
	kind  hapticKind
	style int64
}

// hapticOf is the feedback for the kind of the app's call.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm vibrateHapticFeedback
func hapticOf(k ui.HapticFeedbackKind) haptic {
	switch k {
	case ui.HapticFeedbackVibrate:
		return haptic{kind: hapticVibrate}
	case ui.HapticFeedbackLightImpact:
		return haptic{hapticImpact, impactLight}
	case ui.HapticFeedbackMediumImpact:
		return haptic{hapticImpact, impactMedium}
	case ui.HapticFeedbackHeavyImpact:
		return haptic{hapticImpact, impactHeavy}
	case ui.HapticFeedbackSelectionClick:
		return haptic{kind: hapticSelection}
	}
	return haptic{}
}

// The system sounds of AudioServices that the app plays: the click of the keyboard, the wheels of time (the tick), and the
// vibration.
const (
	soundKeyPressClick uint32 = 1306
	soundWheelsOfTime  uint32 = 1157
	soundVibrate       uint32 = 4095
)

// soundOf is the system sound for the kind. Like Flutter's, the alert has none on iOS, and the sound is not ok then.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm playSystemSound
func soundOf(k ui.SystemSoundKind) (id uint32, ok bool) {
	switch k {
	case ui.SystemSoundClick:
		return soundKeyPressClick, true
	case ui.SystemSoundTick:
		return soundWheelsOfTime, true
	}
	return 0, false
}
