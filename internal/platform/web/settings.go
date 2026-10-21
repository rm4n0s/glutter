package web

import (
	"strings"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// Prefs are the preferences of the user that the browser tells through media queries
// and navigator.languages (SPEC §23.6).
//
// Flutter: no counterpart
type Prefs struct {
	// Dark is whether prefers-color-scheme is dark.
	Dark bool
	// ReducedMotion is whether prefers-reduced-motion is reduce.
	ReducedMotion bool
	// HighContrast is whether prefers-contrast is more.
	HighContrast bool
	// Languages is navigator.languages, most preferred first.
	Languages []string
}

// SettingsOf turns the preferences into the platform's settings: the brightness, reduced
// motion (which turns animations off), high contrast, the locales, and a text scale of 1.
// Without a usable language the locale is en-US, as in Flutter web.
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher.configuration
func SettingsOf(p Prefs) platform.Settings {
	s := platform.Settings{
		PlatformBrightness: ui.BrightnessLight,
		TextScaleFactor:    1,
		DisableAnimations:  p.ReducedMotion,
		HighContrast:       p.HighContrast,
	}
	if p.Dark {
		s.PlatformBrightness = ui.BrightnessDark
	}
	for _, tag := range p.Languages {
		if l, ok := ParseLocale(tag); ok {
			s.Locales = append(s.Locales, l)
		}
	}
	if len(s.Locales) == 0 {
		s.Locales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	return s
}

// ParseLocale reads a BCP 47 language tag such as "en", "en-US", "zh-Hant-TW" or "es-419"
// into a locale: the language in lower case, an optional script of four letters in title
// case, and an optional region (two letters in upper case, or three digits). It is false
// for a tag whose language is not two or three letters. Extensions and private-use
// subtags after the region are ignored.
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart parseBrowserLanguages
func ParseLocale(tag string) (ui.Locale, bool) {
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 || !isLetters(parts[0], 2, 3) {
		return ui.Locale{}, false
	}
	l := ui.Locale{LanguageCode: strings.ToLower(parts[0])}
	rest := parts[1:]
	if len(rest) > 0 && isLetters(rest[0], 4, 4) {
		l.ScriptCode = strings.ToUpper(rest[0][:1]) + strings.ToLower(rest[0][1:])
		rest = rest[1:]
	}
	if len(rest) > 0 {
		switch {
		case isLetters(rest[0], 2, 2):
			l.CountryCode = strings.ToUpper(rest[0])
		case isDigits(rest[0], 3):
			l.CountryCode = rest[0]
		}
	}
	return l, true
}

// isLetters says whether s is made of ASCII letters only, min to max of them.
func isLetters(s string, min, max int) bool {
	if len(s) < min || len(s) > max {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// isDigits says whether s is exactly n ASCII digits.
func isDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
