package windows

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The cases of settings_plugin_unittests.cc that concern what is sent (the brightness, the text scale,
// the 24-hour clock, the high contrast) and of system_utils_unittests.cc (the languages), as tables.

func TestSettingsOf(t *testing.T) {
	tests := []struct {
		name  string
		in    settingsInput
		check func(t *testing.T, s platform.Settings)
	}{
		{"nothing read gives the defaults", settingsInput{}, func(t *testing.T, s platform.Settings) {
			if s.PlatformBrightness != ui.BrightnessLight || s.TextScaleFactor != 1 || s.HasAccentColor || s.HighContrast || s.DisableAnimations {
				t.Errorf("settings = %+v", s)
			}
			if !reflect.DeepEqual(s.Locales, []ui.Locale{ui.NewLocale("en", "US")}) {
				t.Errorf("locales = %v, want en-US", s.Locales)
			}
			if s.UIFontFamily != "Segoe UI" || s.UIFontSize != 9 || s.MonospaceFontFamily != "Consolas" || s.MonospaceFontSize != 11 {
				t.Errorf("fonts = %q %v, %q %v", s.UIFontFamily, s.UIFontSize, s.MonospaceFontFamily, s.MonospaceFontSize)
			}
			if s.CursorSize != 32 || s.CaretBlinkTime != 0 || s.DoubleClickTime != 0 || s.AlwaysUse24HourFormat {
				t.Errorf("cursor %d, blink %v, double click %v, 24 hours %v", s.CursorSize, s.CaretBlinkTime, s.DoubleClickTime, s.AlwaysUse24HourFormat)
			}
		}},
		{"AppsUseLightTheme 0 is dark", settingsInput{HasAppsUseLightTheme: true}, func(t *testing.T, s platform.Settings) {
			if s.PlatformBrightness != ui.BrightnessDark {
				t.Errorf("brightness = %v, want dark", s.PlatformBrightness)
			}
		}},
		{"AppsUseLightTheme 1 is light", settingsInput{HasAppsUseLightTheme: true, AppsUseLightTheme: 1}, func(t *testing.T, s platform.Settings) {
			if s.PlatformBrightness != ui.BrightnessLight {
				t.Errorf("brightness = %v, want light", s.PlatformBrightness)
			}
		}},
		{"AppsUseLightTheme absent is light", settingsInput{AppsUseLightTheme: 0}, func(t *testing.T, s platform.Settings) {
			if s.PlatformBrightness != ui.BrightnessLight {
				t.Errorf("brightness = %v, want light", s.PlatformBrightness)
			}
		}},
		{"the accent color is opaque", settingsInput{HasColorization: true, Colorization: 0xC40078D7}, func(t *testing.T, s platform.Settings) {
			if !s.HasAccentColor || s.AccentColor != ui.NewColor(0xFF0078D7) {
				t.Errorf("accent = %v, %v; want 0xFF0078D7", s.AccentColor, s.HasAccentColor)
			}
		}},
		{"high contrast", settingsInput{HighContrastFlags: 0x1 | 0x4}, func(t *testing.T, s platform.Settings) {
			if !s.HighContrast {
				t.Error("high contrast is off")
			}
		}},
		{"the high contrast flags without the on flag", settingsInput{HighContrastFlags: 0x4}, func(t *testing.T, s platform.Settings) {
			if s.HighContrast {
				t.Error("high contrast is on")
			}
		}},
		{"animation effects off", settingsInput{HasClientAreaAnimation: true}, func(t *testing.T, s platform.Settings) {
			if !s.DisableAnimations {
				t.Error("animations are not disabled")
			}
		}},
		{"animation effects on", settingsInput{HasClientAreaAnimation: true, ClientAreaAnimation: true}, func(t *testing.T, s platform.Settings) {
			if s.DisableAnimations {
				t.Error("animations are disabled")
			}
		}},
		{"a failed read of the animation setting leaves them on", settingsInput{}, func(t *testing.T, s platform.Settings) {
			if s.DisableAnimations {
				t.Error("animations are disabled")
			}
		}},
		{"a 24-hour pattern", settingsInput{ShortTime: "HH:mm"}, func(t *testing.T, s platform.Settings) {
			if !s.AlwaysUse24HourFormat {
				t.Error("24-hour format is off")
			}
		}},
		{"a 24-hour pattern without a leading zero", settingsInput{ShortTime: "H:mm"}, func(t *testing.T, s platform.Settings) {
			if !s.AlwaysUse24HourFormat {
				t.Error("24-hour format is off")
			}
		}},
		{"a 12-hour pattern", settingsInput{ShortTime: "h:mm tt"}, func(t *testing.T, s platform.Settings) {
			if s.AlwaysUse24HourFormat {
				t.Error("24-hour format is on")
			}
		}},
		{"a 12-hour pattern with a leading zero", settingsInput{ShortTime: "hh:mm tt"}, func(t *testing.T, s platform.Settings) {
			if s.AlwaysUse24HourFormat {
				t.Error("24-hour format is on")
			}
		}},
		{"the cursor size", settingsInput{CursorWidth: 48}, func(t *testing.T, s platform.Settings) {
			if s.CursorSize != 48 {
				t.Errorf("cursor size = %d", s.CursorSize)
			}
		}},
		{"ClearType is sub-pixel", settingsInput{HasFontSmoothing: true, FontSmoothing: true, FontSmoothingType: 2}, func(t *testing.T, s platform.Settings) {
			if s.FontAntialiasing != platform.FontAntialiasingRGBA {
				t.Errorf("antialiasing = %v", s.FontAntialiasing)
			}
		}},
		{"standard smoothing is gray", settingsInput{HasFontSmoothing: true, FontSmoothing: true, FontSmoothingType: 1}, func(t *testing.T, s platform.Settings) {
			if s.FontAntialiasing != platform.FontAntialiasingGrayscale {
				t.Errorf("antialiasing = %v", s.FontAntialiasing)
			}
		}},
		{"smoothing off", settingsInput{HasFontSmoothing: true}, func(t *testing.T, s platform.Settings) {
			if s.FontAntialiasing != platform.FontAntialiasingNone {
				t.Errorf("antialiasing = %v", s.FontAntialiasing)
			}
		}},
		{"smoothing unknown", settingsInput{}, func(t *testing.T, s platform.Settings) {
			if s.FontAntialiasing != platform.FontAntialiasingUnspecified {
				t.Errorf("antialiasing = %v", s.FontAntialiasing)
			}
		}},
		{"the caret blink time", settingsInput{CaretBlinkMillis: 530, DoubleClickMillis: 500}, func(t *testing.T, s platform.Settings) {
			if s.CaretBlinkTime != 530*time.Millisecond || s.DoubleClickTime != 500*time.Millisecond {
				t.Errorf("blink %v, double click %v", s.CaretBlinkTime, s.DoubleClickTime)
			}
		}},
		{"a caret blink time of INFINITE is no blinking", settingsInput{CaretBlinkMillis: 0xFFFFFFFF}, func(t *testing.T, s platform.Settings) {
			if s.CaretBlinkTime != ui.CaretBlinkNever {
				t.Errorf("blink = %v, want never", s.CaretBlinkTime)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.check(t, settingsOf(tt.in)) })
	}
}

func TestTheTextScale(t *testing.T) {
	tests := []struct {
		name    string
		percent uint32
		has     bool
		want    float64
	}{
		{"absent", 0, false, 1},
		{"100", 100, true, 1},
		{"125", 125, true, 1.25},
		{"225 is the largest", 225, true, 2.25},
		{"above 225 is cut", 400, true, 2.25},
		{"below 100 is cut", 50, true, 1},
		{"a zero value is 1", 0, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := settingsOf(settingsInput{TextScalePercent: tt.percent, HasTextScale: tt.has})
			if s.TextScaleFactor != tt.want {
				t.Errorf("text scale = %v, want %v", s.TextScaleFactor, tt.want)
			}
		})
	}
}

func TestTheLanguagesOfTheUser(t *testing.T) {
	loc := func(l, s, c string) ui.Locale { return ui.Locale{LanguageCode: l, ScriptCode: s, CountryCode: c} }
	tests := []struct {
		name  string
		langs []string
		want  []ui.Locale
	}{
		{"a language and a region", []string{"en-US"}, []ui.Locale{loc("en", "", "US")}},
		{"several, most preferred first", []string{"el-GR", "en-US", "de"}, []ui.Locale{loc("el", "", "GR"), loc("en", "", "US"), loc("de", "", "")}},
		{"a script tag", []string{"zh-Hans-CN", "sr-Latn-RS"}, []ui.Locale{loc("zh", "Hans", "CN"), loc("sr", "Latn", "RS")}},
		{"a script and no region", []string{"zh-Hant"}, []ui.Locale{loc("zh", "Hant", "")}},
		{"a numeric region", []string{"es-419"}, []ui.Locale{loc("es", "", "419")}},
		{"lower case region", []string{"pt-br"}, []ui.Locale{loc("pt", "", "BR")}},
		{"a tag that is no language is left out", []string{"", "x", "fr-FR"}, []ui.Locale{loc("fr", "", "FR")}},
		{"none gives en-US", nil, []ui.Locale{loc("en", "", "US")}},
		{"only tags that are no language give en-US", []string{"-", ""}, []ui.Locale{loc("en", "", "US")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := settingsOf(settingsInput{Languages: tt.langs}).Locales; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("locales = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTheInterfaceFont(t *testing.T) {
	tests := []struct {
		name   string
		family string
		height int32
		dpi    uint32
		size   float64
	}{
		{"Segoe UI 9 pt at 96 DPI", "Segoe UI", -12, 96, 9},
		{"the same at 144 DPI", "Segoe UI", -18, 144, 9},
		{"a cell height", "Tahoma", 12, 96, 9},
		{"Yu Gothic UI 9 pt on a Japanese system", "Yu Gothic UI", -12, 96, 9},
		{"11 pt", "Segoe UI", -15, 96, 11},
		{"no DPI is 96", "Segoe UI", -12, 0, 9},
		{"a font with no height keeps the default size", "Segoe UI", 0, 96, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := settingsOf(settingsInput{MessageFont: tt.family, MessageFontHeight: tt.height, SystemDPI: tt.dpi})
			if s.UIFontFamily != tt.family || s.UIFontSize != tt.size {
				t.Errorf("font = %q %v, want %q %v", s.UIFontFamily, s.UIFontSize, tt.family, tt.size)
			}
		})
	}
}
