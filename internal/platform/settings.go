package platform

// Brightness describes the contrast of a theme or color palette. The zero
// value, BrightnessUnspecified, means the platform does not say.
//
// TODO(step 5.2): becomes ui.Brightness.
//
// Flutter: ui/platform_dispatcher.dart Brightness
type Brightness int

// The values of Brightness.
const (
	// BrightnessUnspecified is the zero value.
	//
	// Flutter: no counterpart
	BrightnessUnspecified Brightness = iota
	// BrightnessDark is a dark appearance.
	//
	// Flutter: ui/platform_dispatcher.dart Brightness.dark
	BrightnessDark
	// BrightnessLight is a light appearance.
	//
	// Flutter: ui/platform_dispatcher.dart Brightness.light
	BrightnessLight
)

// Settings is the system-wide configuration the app follows.
//
// Flutter: ui/platform_dispatcher.dart PlatformConfiguration
type Settings struct {
	// PlatformBrightness is the system color scheme.
	PlatformBrightness Brightness
	// AccentColor is the system accent color as 0xAARRGGBB. It is meaningful
	// only when HasAccentColor is true.
	//
	// TODO(step 2.2): becomes ui.Color.
	AccentColor uint32
	// HasAccentColor is whether the system has an accent color.
	HasAccentColor bool
	// TextScaleFactor is the system text scale; 0 means 1.
	TextScaleFactor float64
	// DisableAnimations is whether the user asked for reduced motion.
	DisableAnimations bool
	// HighContrast is whether the user asked for high contrast.
	HighContrast bool
	// Locales lists the user's locales as BCP 47 tags, most preferred first.
	//
	// TODO(step 4.2): becomes []ui.Locale.
	Locales []string
}
