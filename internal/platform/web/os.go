package web

import (
	"strings"

	"github.com/rm4n0s/glutter/foundation"
)

// OperatingSystem is the operating system a browser runs on, as Flutter web detects it.
//
// Flutter: lib/web_ui/lib/ui_web/src/ui_web/browser_detection.dart OperatingSystem
type OperatingSystem int

// The values of OperatingSystem. OSUnknown is the zero value.
const (
	// OSUnknown is an operating system that none of the checks recognizes.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.unknown
	OSUnknown OperatingSystem = iota
	// OSAndroid is Android.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.android
	OSAndroid
	// OSIOS is iOS and iPadOS.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.iOs
	OSIOS
	// OSLinux is Linux.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.linux
	OSLinux
	// OSMacOS is macOS.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.macOs
	OSMacOS
	// OSWindows is Windows.
	//
	// Flutter: ui_web/browser_detection.dart OperatingSystem.windows
	OSWindows
)

// DetectOS finds the operating system from navigator.platform, navigator.userAgent and
// navigator.maxTouchPoints, with the checks of Flutter web in the same order: a Mac
// with more than two touch points is an iPad that asks for the desktop site; the user
// agent decides Android, because Android reports its platform as "Linux armv8l".
//
// Flutter: lib/web_ui/lib/ui_web/src/ui_web/browser_detection.dart BrowserDetection.detectOperatingSystem
func DetectOS(navigatorPlatform, userAgent string, maxTouchPoints int) OperatingSystem {
	lower := strings.ToLower(navigatorPlatform)
	switch {
	case strings.HasPrefix(navigatorPlatform, "Mac"):
		if maxTouchPoints > 2 {
			return OSIOS
		}
		return OSMacOS
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") || strings.Contains(lower, "ipod"):
		return OSIOS
	case strings.Contains(userAgent, "Android"):
		return OSAndroid
	case strings.HasPrefix(navigatorPlatform, "Linux"):
		return OSLinux
	case strings.HasPrefix(navigatorPlatform, "Win"):
		return OSWindows
	}
	return OSUnknown
}

// TargetPlatform is the platform the framework behaves like on this operating system.
// An operating system that is not known becomes Android, as in Flutter web.
//
// Flutter: packages/flutter/lib/src/foundation/_platform_web.dart _operatingSystemToTargetPlatform
func (o OperatingSystem) TargetPlatform() foundation.TargetPlatform {
	switch o {
	case OSIOS:
		return foundation.TargetPlatformIOS
	case OSLinux:
		return foundation.TargetPlatformLinux
	case OSMacOS:
		return foundation.TargetPlatformMacOS
	case OSWindows:
		return foundation.TargetPlatformWindows
	}
	return foundation.TargetPlatformAndroid
}

// IsFirefox says whether the user agent is Firefox's, whose wheel events are told from a
// trackpad's differently.
//
// Flutter: lib/web_ui/lib/ui_web/src/ui_web/browser_detection.dart BrowserDetection.browserEngine
func IsFirefox(userAgent string) bool { return strings.Contains(userAgent, "Firefox") }
