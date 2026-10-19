package platform

import "github.com/rm4n0s/glutter/ui"

// Settings is the system-wide configuration the app follows.
//
// Flutter: ui/platform_dispatcher.dart PlatformConfiguration
type Settings struct {
	// PlatformBrightness is the system color scheme.
	PlatformBrightness ui.Brightness
	// AccentColor is the system accent color. It is meaningful only when
	// HasAccentColor is true.
	AccentColor ui.Color
	// HasAccentColor is whether the system has an accent color.
	HasAccentColor bool
	// TextScaleFactor is the system text scale; 0 means 1.
	TextScaleFactor float64
	// DisableAnimations is whether the user asked for reduced motion.
	DisableAnimations bool
	// HighContrast is whether the user asked for high contrast.
	HighContrast bool
	// Locales lists the user's locales, most preferred first.
	Locales []ui.Locale
}
