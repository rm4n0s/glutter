package android

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestParseSettings(t *testing.T) {
	tests := []struct {
		name string
		text string
		want systemSettings
	}{
		{"nothing", "", systemSettings{known: true, fontScale: 1}},
		{"dark and big text", "night=1;font=1.5", systemSettings{known: true, dark: true, fontScale: 1.5}},
		{"everything", "night=1;font=1.3;nonlinear=1;locales=en-US,el-GR,sr-Latn-RS,es-419,fr;h24=1;noanim=1;bold=1;contrast=1", systemSettings{
			known: true, dark: true, fontScale: 1.3, nonLinear: true, is24Hour: true, noAnimations: true, bold: true, highContrast: true,
			locales: []ui.Locale{
				{LanguageCode: "en", CountryCode: "US"}, {LanguageCode: "el", CountryCode: "GR"},
				{LanguageCode: "sr", ScriptCode: "Latn", CountryCode: "RS"}, {LanguageCode: "es", CountryCode: "419"}, {LanguageCode: "fr"},
			},
		}},
		{"an unknown key is ignored", "future=7;night=0", systemSettings{known: true, fontScale: 1}},
		{"underscores", "locales=pt_BR", systemSettings{known: true, fontScale: 1, locales: []ui.Locale{{LanguageCode: "pt", CountryCode: "BR"}}}},
	}
	for _, tt := range tests {
		got, err := parseSettings(tt.text)
		if err != nil || !sameSettings(got, tt.want) {
			t.Errorf("%s: parseSettings = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

func TestParseSettingsFailsWithAndroidSettingsInvalid(t *testing.T) {
	for name, text := range map[string]string{
		"no value":       "night",
		"a zero scale":   "font=0",
		"not a number":   "font=big",
		"a huge scale":   "font=1e9",
		"a negative one": "font=-1",
	} {
		_, err := parseSettings(text)
		if !hasRoute(err, "parseSettings."+tagAndroidSettingsInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestPlatformSettings(t *testing.T) {
	scale := func(f float64) float64 { return f * 2 }
	def := systemSettings{}.platformSettings(scale)
	if def.PlatformBrightness != ui.BrightnessLight || def.TextScaleFactor != 1 || len(def.Locales) != 1 || def.ScaleFontSize != nil {
		t.Errorf("the defaults are %+v", def)
	}
	s, _ := parseSettings("night=1;font=1.5;nonlinear=1;h24=1;noanim=1;bold=1;contrast=1;locales=el-GR")
	got := s.platformSettings(scale)
	want := platform.Settings{
		PlatformBrightness: ui.BrightnessDark, TextScaleFactor: 1.5, AlwaysUse24HourFormat: true, DisableAnimations: true,
		BoldText: true, HighContrast: true, Locales: []ui.Locale{{LanguageCode: "el", CountryCode: "GR"}},
	}
	if got.ScaleFontSize == nil || got.ScaleFontSize(10) != 20 {
		t.Error("a non-linear system did not give its scale")
	}
	got.ScaleFontSize = nil
	if got.PlatformBrightness != want.PlatformBrightness ||
		got.TextScaleFactor != want.TextScaleFactor || got.AlwaysUse24HourFormat != want.AlwaysUse24HourFormat ||
		got.DisableAnimations != want.DisableAnimations || got.BoldText != want.BoldText || got.HighContrast != want.HighContrast ||
		len(got.Locales) != 1 || got.Locales[0] != want.Locales[0] {
		t.Errorf("platformSettings = %+v, want %+v", got, want)
	}
	// A linear system gives no scale function.
	s, _ = parseSettings("font=1.5")
	if s.platformSettings(scale).ScaleFontSize != nil {
		t.Error("a linear system gave a scale function")
	}
}

// TestSettingsChangesAreReported checks that the settings of the Java side reach the engine, once for
// each change, and that a text that cannot be read is dropped.
func TestSettingsChangesAreReported(t *testing.T) {
	h := newHarness(t)
	h.host.fontScale = 2
	h.run()
	if s := h.p.Settings(); s.TextScaleFactor != 1 || s.PlatformBrightness != ui.BrightnessLight {
		t.Errorf("the settings before the Java side said: %+v", s)
	}
	h.b.settingsChanged("night=1;font=1.5;nonlinear=1")
	waitFor(t, "the settings", func() bool { return h.h.count("settings") == 1 })
	s := h.h.lastSettings()
	if s.PlatformBrightness != ui.BrightnessDark || s.TextScaleFactor != 1.5 || s.ScaleFontSize == nil || s.ScaleFontSize(10) != 20 {
		t.Errorf("the settings reported are %+v", s)
	}
	h.b.settingsChanged("night=1;font=1.5;nonlinear=1")
	h.b.settingsChanged("font=banana")
	h.sync(func() {})
	if n := h.h.count("settings"); n != 1 {
		t.Errorf("the same or an invalid text was reported (%d)", n)
	}
	h.b.settingsChanged("night=0;font=1.5;nonlinear=1")
	waitFor(t, "the second change", func() bool { return h.h.count("settings") == 2 })
}
