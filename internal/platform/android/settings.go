package android

import (
	"slices"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// systemSettings are the system's settings that the app follows, as the Java side reads them
// (SPEC §24.7). The zero value is what the platform assumes before the activity reported: light,
// the text scale 1 and the locale en-US.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/SettingsChannel.java SettingsChannel
type systemSettings struct {
	// known is whether the Java side reported the settings; the defaults stand in until it does.
	known bool
	// dark is the night mode of the configuration.
	dark bool
	// fontScale is Configuration.fontScale.
	fontScale float64
	// nonLinear is whether the system scales text non-linearly (Android 14 and later); fonts are
	// then scaled by asking the system.
	nonLinear bool
	// locales are the user's languages, the preferred first.
	locales []ui.Locale
	// is24Hour is DateFormat.is24HourFormat.
	is24Hour bool
	// noAnimations is whether the animator duration scale is 0.
	noAnimations bool
	// bold is whether the system's font weight adjustment makes text bold (API 31).
	bold bool
	// highContrast is the high text contrast setting.
	highContrast bool
}

// parseSettings reads the text of Native.settings: pairs "key=value" separated by ";", flags as 0
// and 1, the locales as BCP 47 tags separated by ",". A key that is not known is ignored, so that
// a newer Java side works with an older Go side. It fails with AndroidSettingsInvalid.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/SettingsChannel.java SettingsChannel.MessageBuilder
func parseSettings(text string) (systemSettings, error) {
	s := systemSettings{known: true, fontScale: 1}
	for _, pair := range strings.Split(text, ";") {
		if pair == "" {
			continue
		}
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return systemSettings{}, errors.New(tagAndroidSettingsInvalid, "a setting has no value", "pair", pair)
		}
		flag := value == "1"
		switch key {
		case "night":
			s.dark = flag
		case "font":
			f, err := strconv.ParseFloat(value, 64)
			if err != nil || f <= 0 || f > 100 {
				return systemSettings{}, errors.New(tagAndroidSettingsInvalid, "the font scale is not a number above zero", "value", value)
			}
			s.fontScale = f
		case "nonlinear":
			s.nonLinear = flag
		case "locales":
			s.locales = parseLocales(value)
		case "h24":
			s.is24Hour = flag
		case "noanim":
			s.noAnimations = flag
		case "bold":
			s.bold = flag
		case "contrast":
			s.highContrast = flag
		}
	}
	return s, nil
}

// sameSettings is whether two settings are the same.
func sameSettings(a, b systemSettings) bool {
	return a.known == b.known && a.dark == b.dark && a.fontScale == b.fontScale && a.nonLinear == b.nonLinear &&
		a.is24Hour == b.is24Hour && a.noAnimations == b.noAnimations && a.bold == b.bold && a.highContrast == b.highContrast &&
		slices.Equal(a.locales, b.locales)
}

// parseLocales reads BCP 47 tags separated by commas ("en-US,el-GR,sr-Latn-RS"): the language, the
// script of four letters and the region of two letters or three digits. A tag without a language
// is left out.
func parseLocales(text string) []ui.Locale {
	var out []ui.Locale
	for _, tag := range strings.Split(text, ",") {
		parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		l := ui.Locale{LanguageCode: strings.ToLower(parts[0])}
		for _, p := range parts[1:] {
			switch {
			case len(p) == 4 && l.ScriptCode == "" && l.CountryCode == "":
				l.ScriptCode = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
			case (len(p) == 2 || len(p) == 3) && l.CountryCode == "":
				l.CountryCode = strings.ToUpper(p)
			}
		}
		out = append(out, l)
	}
	return out
}

// platformSettings turns the settings into what the engine takes. scale is how a font size is
// scaled by the system when it is not linear.
func (s systemSettings) platformSettings(scale func(float64) float64) platform.Settings {
	out := platform.Settings{
		PlatformBrightness: ui.BrightnessLight,
		TextScaleFactor:    1,
		Locales:            []ui.Locale{{LanguageCode: "en", CountryCode: "US"}},
	}
	if !s.known {
		return out
	}
	if s.dark {
		out.PlatformBrightness = ui.BrightnessDark
	}
	out.TextScaleFactor = s.fontScale
	if s.nonLinear {
		out.ScaleFontSize = scale
	}
	if len(s.locales) > 0 {
		out.Locales = s.locales
	}
	out.AlwaysUse24HourFormat = s.is24Hour
	out.DisableAnimations = s.noAnimations
	out.BoldText = s.bold
	out.HighContrast = s.highContrast
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagAndroidSettingsInvalid = "AndroidSettingsInvalid"
