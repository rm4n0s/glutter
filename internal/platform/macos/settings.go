package macos

import (
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the system settings (SPEC §29.8): what the platform reads from
// AppKit is gathered in a settingsInput, and settingsOf turns it into platform.Settings. The reading and
// the notifications that say it changed are settings_darwin.go.

// settingsInput is everything the platform reads from macOS for the settings. A Has field says whether
// the value was there.
type settingsInput struct {
	// Dark is whether NSApp.effectiveAppearance is a dark one (the appearance that best matches
	// NSAppearanceNameAqua or NSAppearanceNameDarkAqua is the dark one).
	Dark bool
	// Accent is NSColor.controlAccentColor converted to sRGB, each channel in 0...1.
	AccentRed, AccentGreen, AccentBlue float64
	HasAccent                          bool
	// ReduceMotion is NSWorkspace.accessibilityDisplayShouldReduceMotion and IncreaseContrast is
	// accessibilityDisplayShouldIncreaseContrast.
	ReduceMotion, IncreaseContrast bool
	// DoubleClickSeconds is NSEvent.doubleClickInterval.
	DoubleClickSeconds float64
	// BlinkOnMillis and BlinkOffMillis are the user defaults NSTextInsertionPointBlinkPeriodOn and
	// NSTextInsertionPointBlinkPeriodOff, in milliseconds; HasBlink is false when neither is set.
	BlinkOnMillis, BlinkOffMillis int
	HasBlink                      bool
	// TimeFormat is the date format of the template "j" in the current locale (hour only), such as
	// "h a" or "HH".
	TimeFormat string
	// Languages are NSLocale.preferredLanguages: BCP 47 tags, most preferred first.
	Languages []string
}

// The fonts that the settings name (SPEC §29.8): the system font of AppKit at its default size, and the
// monospace font that Terminal and Xcode use.
const (
	systemUIFont       = ".AppleSystemUIFont"
	systemUIFontSize   = 13
	monospaceFont      = "Menlo"
	monospaceFontSize  = 11
	maxCaretBlinkMilli = 60 * 1000
)

// settingsOf turns what was read into the settings the engine takes:
//   - the brightness is dark in a dark appearance;
//   - the accent color is the system accent color (Appearance in System Settings), opaque;
//   - reduced motion is Reduce motion and high contrast is Increase contrast of Accessibility;
//   - the locales are the preferred languages, en-US when there are none;
//   - the 24-hour clock is a time format of the current locale with no am/pm marker;
//   - the caret blinks at the period that the user's defaults name, never when both are 0;
//   - the double-click time is the system's.
//
// The text scale is 1: macOS has no system text size setting (SPEC §29.8). Reduce transparency and
// Differentiate without color have no field in platform.Settings.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm FlutterEngine onSettingsChanged
func settingsOf(in settingsInput) platform.Settings {
	s := platform.Settings{
		PlatformBrightness:    ui.BrightnessLight,
		TextScaleFactor:       1,
		AlwaysUse24HourFormat: is24HourFormat(in.TimeFormat),
		DisableAnimations:     in.ReduceMotion,
		HighContrast:          in.IncreaseContrast,
		FontAntialiasing:      platform.FontAntialiasingGrayscale,
		UIFontFamily:          systemUIFont, UIFontSize: systemUIFontSize,
		MonospaceFontFamily: monospaceFont, MonospaceFontSize: monospaceFontSize,
	}
	if in.Dark {
		s.PlatformBrightness = ui.BrightnessDark
	}
	if in.HasAccent {
		s.AccentColor = ui.NewColor(0xFF000000 | channel(in.AccentRed)<<16 | channel(in.AccentGreen)<<8 | channel(in.AccentBlue))
		s.HasAccentColor = true
	}
	for _, tag := range in.Languages {
		if l, ok := ui.ParseLocale(tag); ok {
			s.Locales = append(s.Locales, l)
		}
	}
	if len(s.Locales) == 0 {
		s.Locales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	if in.HasBlink {
		switch on := min(in.BlinkOnMillis, maxCaretBlinkMilli); {
		case in.BlinkOnMillis <= 0 && in.BlinkOffMillis <= 0:
			s.CaretBlinkTime = ui.CaretBlinkNever
		case on > 0:
			s.CaretBlinkTime = time.Duration(on) * time.Millisecond
		}
	}
	if in.DoubleClickSeconds > 0 {
		s.DoubleClickTime = time.Duration(in.DoubleClickSeconds * float64(time.Second))
	}
	return s
}

// channel is a color channel of 0...1 as a byte value, rounded and clamped.
func channel(v float64) uint32 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint32(v*255 + 0.5)
}

// is24HourFormat says that a date format of the template "j" has no am/pm marker: the hour is the
// 24-hour one.
//
// Flutter: shell/platform/darwin/common/framework/Source/FlutterHourFormat.mm FlutterHourFormat isAlwaysUse24HourFormat
func is24HourFormat(format string) bool { return !strings.Contains(format, "a") }
