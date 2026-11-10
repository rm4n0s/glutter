package ios

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestTextScaleOf(t *testing.T) {
	tests := []struct {
		category string
		want     float64
	}{
		{"UICTContentSizeCategoryXS", 14.0 / 17},
		{"UICTContentSizeCategoryS", 15.0 / 17},
		{"UICTContentSizeCategoryM", 16.0 / 17},
		{"UICTContentSizeCategoryL", 1},
		{"UICTContentSizeCategoryXL", 19.0 / 17},
		{"UICTContentSizeCategoryXXL", 21.0 / 17},
		{"UICTContentSizeCategoryXXXL", 23.0 / 17},
		{"UICTContentSizeCategoryAccessibilityM", 28.0 / 17},
		{"UICTContentSizeCategoryAccessibilityL", 33.0 / 17},
		{"UICTContentSizeCategoryAccessibilityXL", 40.0 / 17},
		{"UICTContentSizeCategoryAccessibilityXXL", 47.0 / 17},
		{"UICTContentSizeCategoryAccessibilityXXXL", 53.0 / 17},
		{"UICTContentSizeCategoryUnspecified", 1},
		{"", 1},
		{"something new", 1},
	}
	for _, tc := range tests {
		if got := textScaleOf(tc.category); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("textScaleOf(%q) = %v, want %v", tc.category, got, tc.want)
		}
	}
	// The scale grows with the size of the category.
	last := 0.0
	for _, c := range []string{"XS", "S", "M", "L", "XL", "XXL", "XXXL", "AccessibilityM", "AccessibilityL", "AccessibilityXL", "AccessibilityXXL", "AccessibilityXXXL"} {
		got := textScaleOf("UICTContentSizeCategory" + c)
		if got <= last {
			t.Errorf("the scale of %s (%v) does not grow past %v", c, got, last)
		}
		last = got
	}
}

func TestSettingsOf(t *testing.T) {
	t.Run("the defaults", func(t *testing.T) {
		s := settingsOf(settingsInput{TimeFormat: "h a"})
		if s.PlatformBrightness != ui.BrightnessLight || s.TextScaleFactor != 1 || s.AlwaysUse24HourFormat || s.BoldText || s.DisableAnimations ||
			s.HighContrast || s.AccessibleNavigation {
			t.Errorf("defaults: %+v", s)
		}
		if len(s.Locales) != 1 || s.Locales[0] != ui.NewLocale("en", "US") {
			t.Errorf("locales: %v", s.Locales)
		}
	})
	t.Run("everything on", func(t *testing.T) {
		s := settingsOf(settingsInput{Dark: true, Category: "UICTContentSizeCategoryAccessibilityXXXL", BoldText: true, ReduceMotion: true,
			IncreaseContrast: true, VoiceOver: true, TimeFormat: "HH", Languages: []string{"el-GR", "en-US"}})
		if s.PlatformBrightness != ui.BrightnessDark || math.Abs(s.TextScaleFactor-53.0/17) > 1e-9 || !s.AlwaysUse24HourFormat || !s.BoldText ||
			!s.DisableAnimations || !s.HighContrast || !s.AccessibleNavigation {
			t.Errorf("settings: %+v", s)
		}
		if len(s.Locales) != 2 || s.Locales[0] != ui.NewLocale("el", "GR") {
			t.Errorf("locales: %v", s.Locales)
		}
	})
	t.Run("reduce transparency, differentiate without color and on/off labels", func(t *testing.T) {
		s := settingsOf(settingsInput{ReduceTransparency: true, DifferentiateWithoutColor: true, OnOffLabels: true})
		if !s.ReduceTransparency || !s.DifferentiateWithoutColor || !s.OnOffSwitchLabels || s.HighContrast || s.BoldText {
			t.Errorf("settings: %+v", s)
		}
		if s := settingsOf(settingsInput{}); s.ReduceTransparency || s.DifferentiateWithoutColor || s.OnOffSwitchLabels {
			t.Errorf("nothing on gave %+v", s)
		}
	})
	t.Run("switch control is accessible navigation, speak screen is not", func(t *testing.T) {
		if !settingsOf(settingsInput{SwitchControl: true}).AccessibleNavigation {
			t.Error("Switch Control is not accessible navigation")
		}
		if settingsOf(settingsInput{SpeakScreen: true}).AccessibleNavigation {
			t.Error("Speak Screen is accessible navigation")
		}
	})
	t.Run("tags that are not locales are left out", func(t *testing.T) {
		s := settingsOf(settingsInput{Languages: []string{"", "fr-FR"}})
		if len(s.Locales) != 1 || s.Locales[0] != ui.NewLocale("fr", "FR") {
			t.Errorf("locales: %v", s.Locales)
		}
	})
}

func TestIs24HourFormat(t *testing.T) {
	for format, want := range map[string]bool{"HH": true, "H": true, "h a": false, "ah": false, "h 'o''clock' a": false} {
		if got := is24HourFormat(format); got != want {
			t.Errorf("is24HourFormat(%q) = %v, want %v", format, got, want)
		}
	}
}
