package macos

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestSettingsOf(t *testing.T) {
	en := []ui.Locale{ui.NewLocale("en", "US")}
	tests := []struct {
		name string
		in   settingsInput
		want func(platform.Settings) bool
	}{
		{"defaults", settingsInput{}, func(s platform.Settings) bool {
			return s.PlatformBrightness == ui.BrightnessLight && s.TextScaleFactor == 1 && !s.HasAccentColor &&
				len(s.Locales) == 1 && s.Locales[0] == en[0] && s.CaretBlinkTime == 0 && s.DoubleClickTime == 0 &&
				s.AlwaysUse24HourFormat && s.UIFontFamily == systemUIFont
		}},
		{"dark", settingsInput{Dark: true}, func(s platform.Settings) bool { return s.PlatformBrightness == ui.BrightnessDark }},
		{"accent blue", settingsInput{HasAccent: true, AccentRed: 0, AccentGreen: 0.478, AccentBlue: 1}, func(s platform.Settings) bool {
			return s.HasAccentColor && s.AccentColor == ui.NewColor(0xFF007AFF)
		}},
		{"accent clamps", settingsInput{HasAccent: true, AccentRed: 1.5, AccentGreen: -1, AccentBlue: 0.5}, func(s platform.Settings) bool {
			return s.AccentColor == ui.NewColor(0xFFFF0080)
		}},
		{"accessibility", settingsInput{ReduceMotion: true, IncreaseContrast: true}, func(s platform.Settings) bool {
			return s.DisableAnimations && s.HighContrast
		}},
		{"reduce transparency and differentiate without color", settingsInput{ReduceTransparency: true, DifferentiateWithoutColor: true}, func(s platform.Settings) bool {
			return s.ReduceTransparency && s.DifferentiateWithoutColor && !s.OnOffSwitchLabels && !s.HighContrast
		}},
		{"nothing on", settingsInput{}, func(s platform.Settings) bool { return !s.ReduceTransparency && !s.DifferentiateWithoutColor }},
		{"12 hour", settingsInput{TimeFormat: "h a"}, func(s platform.Settings) bool { return !s.AlwaysUse24HourFormat }},
		{"24 hour", settingsInput{TimeFormat: "HH"}, func(s platform.Settings) bool { return s.AlwaysUse24HourFormat }},
		{"locales", settingsInput{Languages: []string{"el-GR", "zh-Hans-CN", "", "C", "sr-Latn-RS"}}, func(s platform.Settings) bool {
			return len(s.Locales) == 3 && s.Locales[0] == ui.NewLocale("el", "GR") &&
				s.Locales[1] == (ui.Locale{LanguageCode: "zh", ScriptCode: "Hans", CountryCode: "CN"}) &&
				s.Locales[2] == (ui.Locale{LanguageCode: "sr", ScriptCode: "Latn", CountryCode: "RS"})
		}},
		{"no valid language", settingsInput{Languages: []string{"", "C"}}, func(s platform.Settings) bool {
			return len(s.Locales) == 1 && s.Locales[0] == en[0]
		}},
		{"blink on", settingsInput{HasBlink: true, BlinkOnMillis: 560, BlinkOffMillis: 280}, func(s platform.Settings) bool {
			return s.CaretBlinkTime == 560*time.Millisecond
		}},
		{"blink never", settingsInput{HasBlink: true}, func(s platform.Settings) bool { return s.CaretBlinkTime == ui.CaretBlinkNever }},
		{"blink only off set", settingsInput{HasBlink: true, BlinkOffMillis: 300}, func(s platform.Settings) bool { return s.CaretBlinkTime == 0 }},
		{"blink capped", settingsInput{HasBlink: true, BlinkOnMillis: 10_000_000}, func(s platform.Settings) bool {
			return s.CaretBlinkTime == time.Minute
		}},
		{"double click", settingsInput{DoubleClickSeconds: 0.5}, func(s platform.Settings) bool { return s.DoubleClickTime == 500*time.Millisecond }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := settingsOf(tt.in); !tt.want(got) {
				t.Errorf("settingsOf(%+v) = %+v", tt.in, got)
			}
		})
	}
}
