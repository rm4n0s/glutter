package dbus

import (
	"context"
	"strconv"
	"strings"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

const (
	settingsInterface = "org.freedesktop.portal.Settings"

	// The namespaces and keys of SPEC §8.8.
	appearanceNamespace = "org.freedesktop.appearance"
	gnomeNamespace      = "org.gnome.desktop.interface"

	keyColorScheme      = "color-scheme"
	keyAccentColor      = "accent-color"
	keyContrast         = "contrast"
	keyTextScaling      = "text-scaling-factor"
	keyEnableAnimations = "enable-animations"
	keyCursorSize       = "cursor-size"
	keyCursorTheme      = "cursor-theme"
	keyFontAntialiasing = "font-antialiasing"
	keyFontHinting      = "font-hinting"

	// defaultCursorSize is the size of the XCursor theme when XCURSOR_SIZE is
	// not set.
	defaultCursorSize = 24
)

// FallbackSettings returns the settings to use when the Settings portal is
// missing: the environment variables XCURSOR_SIZE and XCURSOR_THEME, then the
// defaults (a light theme, no text scaling, animations on).
//
// getenv looks up an environment variable; pass os.Getenv.
//
// The locale is always read from the environment (LC_ALL, LC_MESSAGES, LANG): the
// portal has no locale. X11 adds XSETTINGS to the fallback in its own package.
//
// Flutter: no counterpart
func FallbackSettings(getenv func(string) string) platform.Settings {
	s := platform.Settings{
		PlatformBrightness: ui.BrightnessLight,
		TextScaleFactor:    1,
		CursorSize:         defaultCursorSize,
		CursorTheme:        getenv("XCURSOR_THEME"),
		Locales:            LocalesFromEnv(getenv),
	}
	if n, err := strconv.Atoi(getenv("XCURSOR_SIZE")); err == nil && n > 0 {
		s.CursorSize = n
	}
	return s
}

// LocalesFromEnv returns the locale that the environment names: the first of
// LC_ALL, LC_MESSAGES and LANG that is set and is not empty, "C" or "POSIX", as a
// language and a country ("de_DE.UTF-8@euro" is de DE). It returns nil when none
// names one, and the app then has the default locale.
//
// Flutter: no counterpart
func LocalesFromEnv(getenv func(string) string) []ui.Locale {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := getenv(name)
		if v == "" {
			continue
		}
		if l, ok := parsePOSIXLocale(v); ok {
			return []ui.Locale{l}
		}
		return nil // the first variable that is set decides, even for C
	}
	return nil
}

// parsePOSIXLocale reads language[_TERRITORY][.codeset][@modifier].
func parsePOSIXLocale(v string) (ui.Locale, bool) {
	v, _, _ = strings.Cut(v, "@")
	v, _, _ = strings.Cut(v, ".")
	lang, country, _ := strings.Cut(v, "_")
	if lang == "" || lang == "C" || lang == "POSIX" {
		return ui.Locale{}, false
	}
	for _, r := range lang {
		if r < 'a' || r > 'z' {
			return ui.Locale{}, false
		}
	}
	if country != "" {
		return ui.NewLocale(lang, strings.ToUpper(country)), true
	}
	return ui.NewLocale(lang), true
}

// ReadSettings reads the keys of SPEC §8.8 from the Settings portal and applies
// them over base. A key that the portal does not have keeps its value in base,
// and a value of the wrong type is ignored.
//
// Flutter: no counterpart
func ReadSettings(ctx context.Context, bus Bus, base platform.Settings) (platform.Settings, error) {
	reply, err := bus.Call(ctx, PortalDest, PortalPath, settingsInterface+".ReadAll",
		[]string{appearanceNamespace, gnomeNamespace})
	if err != nil {
		return platform.Settings{}, err
	}
	if len(reply) != 1 {
		return platform.Settings{}, errors.New("SettingsReplyInvalid", "ReadAll does not return one value", "values", len(reply))
	}
	all, ok := reply[0].(map[string]map[string]godbus.Variant)
	if !ok {
		return platform.Settings{}, errors.New("SettingsReplyInvalid", "ReadAll does not return a{sa{sv}}", "type", typeName(reply[0]))
	}
	s := base
	for namespace, keys := range all {
		for key, value := range keys {
			applySetting(&s, namespace, key, value)
		}
	}
	return s, nil
}

// WatchSettings returns the current settings and follows their changes until
// ctx is done. Each change is passed to onSettings, which stands for
// Handler.OnSettings: it runs on the UI thread, through post (Platform.Post).
//
// The settings come from the Settings portal over bus. When there is no bus
// (bus is nil), or the portal is missing, they are FallbackSettings(getenv) and
// nothing is followed; an app on a desktop without the portal simply keeps the
// settings it started with.
//
// Flutter: no counterpart
func WatchSettings(ctx context.Context, bus Bus, getenv func(string) string, post func(func()), onSettings func(platform.Settings)) platform.Settings {
	s, _ := WatchPortalSettings(ctx, bus, getenv, post, onSettings)
	return s
}

// WatchPortalSettings is WatchSettings that also says whether the settings came from
// the portal. When they did not, the caller can look for other sources, such as
// XSETTINGS on X11.
//
// Flutter: no counterpart
func WatchPortalSettings(ctx context.Context, bus Bus, getenv func(string) string, post func(func()), onSettings func(platform.Settings)) (settings platform.Settings, fromPortal bool) {
	base := FallbackSettings(getenv)
	if bus == nil {
		return base, false
	}
	watchCtx, cancel := context.WithCancel(ctx)
	// Subscribe before reading, so that a change in between is not lost.
	sigs, err := bus.Subscribe(watchCtx, SignalMatch{Sender: PortalDest, Path: PortalPath, Interface: settingsInterface, Member: "SettingChanged"})
	if err != nil {
		cancel()
		return base, false
	}
	current, err := ReadSettings(watchCtx, bus, base)
	if err != nil {
		cancel()
		return base, false
	}
	initial := current
	go func() {
		defer cancel()
		for sig := range sigs {
			namespace, key, value, ok := parseSettingChanged(sig)
			if !ok || !applySetting(&current, namespace, key, value) {
				continue
			}
			changed := current
			post(func() { onSettings(changed) })
		}
	}()
	return initial, true
}

func parseSettingChanged(sig *godbus.Signal) (namespace, key string, value any, ok bool) {
	if len(sig.Body) != 3 {
		return "", "", nil, false
	}
	namespace, ok1 := sig.Body[0].(string)
	key, ok2 := sig.Body[1].(string)
	if !ok1 || !ok2 {
		return "", "", nil, false
	}
	return namespace, key, sig.Body[2], true
}

// applySetting sets the field of s that the key stands for, and reports
// whether the field changed. Keys that SPEC §8.8 does not use, and values of an
// unexpected type, change nothing.
func applySetting(s *platform.Settings, namespace, key string, value any) bool {
	v := unwrapVariant(value)
	switch namespace {
	case appearanceNamespace:
		switch key {
		case keyColorScheme:
			n, ok := v.(uint32)
			if !ok {
				return false
			}
			// 0 is "no preference", which Flutter's platform brightness
			// reports as light.
			b := ui.BrightnessLight
			if n == 1 {
				b = ui.BrightnessDark
			}
			return set(&s.PlatformBrightness, b)
		case keyAccentColor:
			c, ok := parseAccent(v)
			if !ok {
				// Out-of-range components mean "no accent color".
				return set(&s.HasAccentColor, false)
			}
			hasChanged := set(&s.HasAccentColor, true)
			return set(&s.AccentColor, c) || hasChanged
		case keyContrast:
			n, ok := v.(uint32)
			if !ok {
				return false
			}
			return set(&s.HighContrast, n == 1)
		}
	case gnomeNamespace:
		switch key {
		case keyTextScaling:
			f, ok := v.(float64)
			if !ok || f <= 0 {
				return false
			}
			return set(&s.TextScaleFactor, f)
		case keyEnableAnimations:
			b, ok := v.(bool)
			if !ok {
				return false
			}
			return set(&s.DisableAnimations, !b)
		case keyCursorSize:
			n, ok := v.(int32)
			if !ok || n <= 0 {
				return false
			}
			return set(&s.CursorSize, int(n))
		case keyCursorTheme:
			t, ok := v.(string)
			if !ok {
				return false
			}
			return set(&s.CursorTheme, t)
		case keyFontAntialiasing:
			name, ok := v.(string)
			if !ok {
				return false
			}
			return set(&s.FontAntialiasing, fontAntialiasing(name))
		case keyFontHinting:
			name, ok := v.(string)
			if !ok {
				return false
			}
			return set(&s.FontHinting, fontHinting(name))
		}
	}
	return false
}

// set stores v in *p and reports whether that changed it.
func set[T comparable](p *T, v T) bool {
	if *p == v {
		return false
	}
	*p = v
	return true
}

// parseAccent reads the (ddd) value of accent-color. Components outside 0..1
// mean that the user has no accent color.
func parseAccent(v any) (ui.Color, bool) {
	parts, ok := v.([]any)
	if !ok || len(parts) != 3 {
		return ui.Color{}, false
	}
	var c [3]float64
	for i, p := range parts {
		f, ok := p.(float64)
		if !ok || f < 0 || f > 1 {
			return ui.Color{}, false
		}
		c[i] = f
	}
	return ui.NewColorFrom(1, c[0], c[1], c[2]), true
}

func fontAntialiasing(name string) platform.FontAntialiasing {
	switch name {
	case "none":
		return platform.FontAntialiasingNone
	case "grayscale":
		return platform.FontAntialiasingGrayscale
	case "rgba":
		return platform.FontAntialiasingRGBA
	}
	return platform.FontAntialiasingUnspecified
}

func fontHinting(name string) platform.FontHinting {
	switch name {
	case "none":
		return platform.FontHintingNone
	case "slight":
		return platform.FontHintingSlight
	case "medium":
		return platform.FontHintingMedium
	case "full":
		return platform.FontHintingFull
	}
	return platform.FontHintingUnspecified
}
