//go:build darwin && !ios && glutter_display

package macos

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// setAppAppearance forces the appearance of the app to Aqua or Dark Aqua, as the user switching the
// system's appearance does for an app that follows it.
func setAppAppearance(s *session, dark bool) {
	s.onUI(func() {
		name := appearanceNameAqua
		if dark {
			name = appearanceNameDarkAqua
		}
		ns := objcx.NewString(name)
		s.p.app.Send(sel("setAppearance:"), class("NSAppearance").Send(sel("appearanceNamed:"), ns))
		objcx.Release(ns)
	})
}

func TestSettingsAreReadFromTheSystem(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			got := s.p.Settings()
			if got.TextScaleFactor != 1 || len(got.Locales) == 0 || got.Locales[0].LanguageCode == "" {
				t.Errorf("settings = %+v", got)
			}
			if !got.HasAccentColor || got.AccentColor.Alpha() != 255 {
				t.Errorf("the accent color = %v (has %v), want an opaque one", got.AccentColor, got.HasAccentColor)
			}
			if got.DoubleClickTime <= 0 {
				t.Errorf("DoubleClickTime = %v", got.DoubleClickTime)
			}
			if got.UIFontFamily != systemUIFont {
				t.Errorf("UIFontFamily = %q", got.UIFontFamily)
			}
		})
	})
}

func TestAppearanceSwitchReportsTheBrightness(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		setAppAppearance(s, false)
		s.waitFor("light", 5*time.Second, func() bool { return s.p.Settings().PlatformBrightness == ui.BrightnessLight })
		before := len(h.settingsHeard())
		setAppAppearance(s, true)
		if !s.waitFor("a dark OnSettings", 5*time.Second, func() bool {
			got := h.settingsHeard()
			return len(got) > before && got[len(got)-1].PlatformBrightness == ui.BrightnessDark
		}) {
			return
		}
		setAppAppearance(s, false)
		s.waitFor("a light OnSettings", 5*time.Second, func() bool {
			got := h.settingsHeard()
			return got[len(got)-1].PlatformBrightness == ui.BrightnessLight
		})
		// The same settings are not reported twice in a row.
		got := h.settingsHeard()
		for i := 1; i < len(got); i++ {
			if got[i].PlatformBrightness == got[i-1].PlatformBrightness && got[i].AccentColor == got[i-1].AccentColor {
				t.Errorf("OnSettings %d repeats the one before: %+v", i, got[i])
			}
		}
	})
}
