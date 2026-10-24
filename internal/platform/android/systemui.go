package android

import (
	"strconv"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// The operations of the activity's command call (GlutterActivity.command): what the framework
// asks of the system's interface and the feedback devices (SPEC §24.7). Each one is done on the
// main thread of Android.
const (
	// opHaptic: A is the ui.HapticFeedbackKind.
	opHaptic = 1
	// opSound: A is the ui.SystemSoundKind.
	opSound = 2
	// opOrientation: A is the ActivityInfo.SCREEN_ORIENTATION_* value.
	opOrientation = 3
	// opUIMode: A is the ui.SystemUIMode, B the bits of the overlays that stay (overlayTop,
	// overlayBottom).
	opUIMode = 4
	// opOverlayStyle: S is the style, see encodeOverlayStyle.
	opOverlayStyle = 5
	// opKeepScreenOn: A is 1 to keep the screen on, 0 to let it go idle.
	opKeepScreenOn = 6
	// opUIChangeListener: A is 1 when Go wants to hear of the bars appearing and disappearing.
	opUIChangeListener = 7
)

// The bits of an overlay set.
const (
	overlayTop    = 1
	overlayBottom = 2
)

// command is one call to the activity.
//
// Flutter: no counterpart
type command struct {
	Op         int
	A, B, C, D int64
	S          string
}

// The values of ActivityInfo.SCREEN_ORIENTATION_*.
const (
	screenOrientationUnspecified      = -1
	screenOrientationLandscape        = 0
	screenOrientationPortrait         = 1
	screenOrientationUser             = 2
	screenOrientationReverseLandscape = 8
	screenOrientationReversePortrait  = 9
	screenOrientationUserLandscape    = 11
	screenOrientationUserPortrait     = 12
	screenOrientationFullUser         = 13
)

// requestedOrientation turns the orientations of SystemChrome.setPreferredOrientations into the
// value for Activity.setRequestedOrientation, as Flutter's PlatformChannel does: Android cannot
// say some sets, and then the first orientation of the list wins.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/PlatformChannel.java PlatformChannel.decodeOrientations
func requestedOrientation(orientations []ui.DeviceOrientation) int {
	requested, first := 0, 0
	for _, o := range orientations {
		switch o {
		case ui.DeviceOrientationPortraitUp:
			requested |= 0x01
		case ui.DeviceOrientationPortraitDown:
			requested |= 0x04
		case ui.DeviceOrientationLandscapeLeft:
			requested |= 0x02
		case ui.DeviceOrientationLandscapeRight:
			requested |= 0x08
		}
		if first == 0 {
			first = requested
		}
	}
	switch requested {
	case 0x00:
		return screenOrientationUnspecified
	case 0x01:
		return screenOrientationPortrait
	case 0x02:
		return screenOrientationLandscape
	case 0x04:
		return screenOrientationReversePortrait
	case 0x05:
		return screenOrientationUserPortrait
	case 0x08:
		return screenOrientationReverseLandscape
	case 0x0a:
		return screenOrientationUserLandscape
	case 0x0b:
		return screenOrientationUser
	case 0x0f:
		return screenOrientationFullUser
	}
	// Android cannot describe the other sets: the first orientation of the list stands for them.
	switch first {
	case 0x01:
		return screenOrientationPortrait
	case 0x02:
		return screenOrientationLandscape
	case 0x04:
		return screenOrientationReversePortrait
	case 0x08:
		return screenOrientationReverseLandscape
	}
	return screenOrientationPortrait
}

// overlayBits is the set of overlays as the bits of opUIMode.
func overlayBits(overlays []ui.SystemUIOverlay) int64 {
	var bits int64
	for _, o := range overlays {
		switch o {
		case ui.SystemUIOverlayTop:
			bits |= overlayTop
		case ui.SystemUIOverlayBottom:
			bits |= overlayBottom
		}
	}
	return bits
}

// encodeOverlayStyle turns a style into the text of opOverlayStyle: seven fields separated by
// commas, an empty one for what the app does not say:
//
//	status bar color (ARGB), status bar icons (1 light, 2 dark), status bar contrast enforced (0, 1),
//	navigation bar color, divider color, navigation bar icons, navigation bar contrast enforced
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/PlatformChannel.java PlatformChannel.decodeSystemChromeStyle
func encodeOverlayStyle(s ui.SystemUIOverlayStyle) string {
	color := func(c *ui.Color) string {
		if c == nil {
			return ""
		}
		return strconv.FormatUint(uint64(c.ToARGB32()), 10)
	}
	flag := func(b *bool) string {
		switch {
		case b == nil:
			return ""
		case *b:
			return "1"
		}
		return "0"
	}
	icons := func(b ui.Brightness) string {
		switch b {
		case ui.BrightnessLight:
			return "1"
		case ui.BrightnessDark:
			return "2"
		}
		return ""
	}
	return strings.Join([]string{
		color(s.StatusBarColor), icons(s.StatusBarIconBrightness), flag(s.SystemStatusBarContrastEnforced),
		color(s.SystemNavigationBarColor), color(s.SystemNavigationBarDividerColor),
		icons(s.SystemNavigationBarIconBrightness), flag(s.SystemNavigationBarContrastEnforced),
	}, ",")
}
