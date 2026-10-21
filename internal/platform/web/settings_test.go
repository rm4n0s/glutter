package web

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestParseLocale(t *testing.T) {
	tests := []struct {
		tag  string
		want ui.Locale
		ok   bool
	}{
		{"en", ui.Locale{LanguageCode: "en"}, true},
		{"en-US", ui.Locale{LanguageCode: "en", CountryCode: "US"}, true},
		{"EN-us", ui.Locale{LanguageCode: "en", CountryCode: "US"}, true},
		{"en_GB", ui.Locale{LanguageCode: "en", CountryCode: "GB"}, true},
		{"zh-Hant-TW", ui.Locale{LanguageCode: "zh", ScriptCode: "Hant", CountryCode: "TW"}, true},
		{"sr-LATN", ui.Locale{LanguageCode: "sr", ScriptCode: "Latn"}, true},
		{"es-419", ui.Locale{LanguageCode: "es", CountryCode: "419"}, true},
		{"fil", ui.Locale{LanguageCode: "fil"}, true},
		{"de-DE-u-co-phonebk", ui.Locale{LanguageCode: "de", CountryCode: "DE"}, true},
		{"", ui.Locale{}, false},
		{"x", ui.Locale{}, false},
		{"english", ui.Locale{}, false},
		{"1a", ui.Locale{}, false},
		{"en-", ui.Locale{LanguageCode: "en"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			got, ok := ParseLocale(tc.tag)
			if ok != tc.ok || got != tc.want {
				t.Errorf("ParseLocale(%q) = %+v, %v, want %+v, %v", tc.tag, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestSettingsOf(t *testing.T) {
	tests := []struct {
		name  string
		prefs Prefs
		check func(t *testing.T, s settingsView)
	}{
		{"defaults", Prefs{}, func(t *testing.T, s settingsView) {
			if s.brightness != ui.BrightnessLight || s.scale != 1 || s.noAnim || s.contrast {
				t.Errorf("got %+v, want light, scale 1, animations on, normal contrast", s)
			}
			if !reflect.DeepEqual(s.locales, []ui.Locale{ui.NewLocale("en", "US")}) {
				t.Errorf("locales = %v, want en-US", s.locales)
			}
		}},
		{"dark, reduced motion, high contrast", Prefs{Dark: true, ReducedMotion: true, HighContrast: true}, func(t *testing.T, s settingsView) {
			if s.brightness != ui.BrightnessDark || !s.noAnim || !s.contrast {
				t.Errorf("got %+v, want dark, no animations, high contrast", s)
			}
		}},
		{"languages in order, bad ones skipped", Prefs{Languages: []string{"el-GR", "??", "en"}}, func(t *testing.T, s settingsView) {
			want := []ui.Locale{{LanguageCode: "el", CountryCode: "GR"}, {LanguageCode: "en"}}
			if !reflect.DeepEqual(s.locales, want) {
				t.Errorf("locales = %v, want %v", s.locales, want)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SettingsOf(tc.prefs)
			tc.check(t, settingsView{got.PlatformBrightness, got.TextScaleFactor, got.DisableAnimations, got.HighContrast, got.Locales})
		})
	}
}

// settingsView is the part of platform.Settings that SettingsOf sets.
type settingsView struct {
	brightness ui.Brightness
	scale      float64
	noAnim     bool
	contrast   bool
	locales    []ui.Locale
}
