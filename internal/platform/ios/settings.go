package ios

import (
	"strings"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the system settings of iOS (SPEC §30.7): what the platform reads from UIKit is gathered in a
// settingsInput, and settingsOf turns it into platform.Settings. The reading and the notifications that say it changed are
// settings_ios.go.

// settingsInput is everything the platform reads from iOS for the settings.
type settingsInput struct {
	// Dark is whether the main screen's trait collection has the dark interface style.
	Dark bool
	// Category is UIApplication.preferredContentSizeCategory: the Dynamic Type size.
	Category string
	// BoldText, ReduceMotion, IncreaseContrast (UIAccessibilityDarkerSystemColorsEnabled or the high contrast trait) and
	// ReduceTransparency are the switches of Settings > Accessibility > Display & Text Size and Motion.
	BoldText, ReduceMotion, IncreaseContrast, ReduceTransparency bool
	// OnOffLabels is Settings > Accessibility > On/Off Labels.
	OnOffLabels bool
	// DifferentiateWithoutColor is Settings > Accessibility > Display & Text Size > Differentiate Without Color
	// (UIAccessibilityShouldDifferentiateWithoutColor).
	DifferentiateWithoutColor bool
	// VoiceOver and SwitchControl say that the assistive technology is running; SpeakScreen that Speak Screen is on.
	VoiceOver, SwitchControl, SpeakScreen bool
	// TimeFormat is the date format of the template "j" in the current locale (hour only), such as "h a" or "HH".
	TimeFormat string
	// Languages are NSLocale.preferredLanguages: BCP 47 tags, most preferred first.
	Languages []string
}

// The fonts that the settings name (SPEC §30.7): the system font of UIKit at the size of body text.
const (
	systemUIFont     = ".AppleSystemUIFont"
	systemUIFontSize = 17
)

// settingsOf turns what was read into the settings the engine takes:
//   - the brightness is dark in the dark interface style;
//   - the text scale is the Dynamic Type size (textScaleOf);
//   - bold text, reduced motion and increased contrast are the switches of Accessibility;
//   - accessible navigation is VoiceOver or Switch Control running;
//   - the locales are the preferred languages, en-US when there are none;
//   - the 24-hour clock is a time format of the current locale with no am/pm marker.
//
// Reduce transparency, differentiate without color and on/off labels are the switches of the same names (AccessibilityFeatures
// and MediaQuery have them).
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm onUserSettingsChanged, AccessibilityFeatures.swift flags
func settingsOf(in settingsInput) platform.Settings {
	s := platform.Settings{
		PlatformBrightness:    ui.BrightnessLight,
		TextScaleFactor:       textScaleOf(in.Category),
		AlwaysUse24HourFormat: is24HourFormat(in.TimeFormat),
		BoldText:              in.BoldText,
		DisableAnimations:     in.ReduceMotion,
		HighContrast:          in.IncreaseContrast,
		OnOffSwitchLabels:     in.OnOffLabels,
		ReduceTransparency:    in.ReduceTransparency, DifferentiateWithoutColor: in.DifferentiateWithoutColor,
		AccessibleNavigation: in.VoiceOver || in.SwitchControl,
		FontAntialiasing:     platform.FontAntialiasingGrayscale,
		UIFontFamily:         systemUIFont, UIFontSize: systemUIFontSize,
	}
	if in.Dark {
		s.PlatformBrightness = ui.BrightnessDark
	}
	for _, tag := range in.Languages {
		if l, ok := ui.ParseLocale(tag); ok {
			s.Locales = append(s.Locales, l)
		}
	}
	if len(s.Locales) == 0 {
		s.Locales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	return s
}

// bodySizes are the sizes of body text, in points, for each Dynamic Type category, after Apple's typography guidelines; the large size
// is the default and has the scale 1.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm textScaleFactor
var bodySizes = map[string]float64{
	"UICTContentSizeCategoryXS":                14,
	"UICTContentSizeCategoryS":                 15,
	"UICTContentSizeCategoryM":                 16,
	"UICTContentSizeCategoryL":                 17,
	"UICTContentSizeCategoryXL":                19,
	"UICTContentSizeCategoryXXL":               21,
	"UICTContentSizeCategoryXXXL":              23,
	"UICTContentSizeCategoryAccessibilityM":    28,
	"UICTContentSizeCategoryAccessibilityL":    33,
	"UICTContentSizeCategoryAccessibilityXL":   40,
	"UICTContentSizeCategoryAccessibilityXXL":  47,
	"UICTContentSizeCategoryAccessibilityXXXL": 53,
}

// defaultBodySize is the body size of the large category, the one with the scale 1.
const defaultBodySize = 17

// textScaleOf is the text scale for the Dynamic Type category: the body size of the category over the body size of the default one. A
// category that is not known (or none) is 1.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm textScaleFactor
func textScaleOf(category string) float64 {
	if size, ok := bodySizes[category]; ok {
		return size / defaultBodySize
	}
	return 1
}

// is24HourFormat says that a date format of the template "j" has no am/pm marker: the hour is the 24-hour one.
//
// Flutter: shell/platform/darwin/common/framework/Source/FlutterHourFormat.mm FlutterHourFormat isAlwaysUse24HourFormat
func is24HourFormat(format string) bool { return !strings.Contains(format, "a") }
