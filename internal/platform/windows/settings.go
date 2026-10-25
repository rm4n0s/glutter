package windows

import (
	"math"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the system settings (SPEC §25.8): what the platform reads from
// Windows is gathered in a settingsInput, and settingsOf turns it into platform.Settings. The
// reading is settings_windows.go. Both follow Flutter's SettingsPlugin (settings_plugin.cc) and the
// preferred languages of system_utils.cc.

// settingsInput is everything the platform reads from Windows for the settings. A Has field says
// whether the value was there: a Windows (or a Wine) without it has the default.
type settingsInput struct {
	// AppsUseLightTheme is the registry value of that name (0 is the dark mode of the apps).
	AppsUseLightTheme    uint32
	HasAppsUseLightTheme bool
	// Colorization is the accent color of the desktop as 0xAARRGGBB (DwmGetColorizationColor).
	Colorization    uint32
	HasColorization bool
	// HighContrastFlags are the flags of SPI_GETHIGHCONTRAST.
	HighContrastFlags uint32
	// TextScalePercent is the registry value TextScaleFactor, a percentage; HasTextScale is false
	// when the value is absent.
	TextScalePercent uint32
	HasTextScale     bool
	// ClientAreaAnimation is SPI_GETCLIENTAREAANIMATION; HasClientAreaAnimation is false when the
	// call failed.
	ClientAreaAnimation    bool
	HasClientAreaAnimation bool
	// Languages are the preferred UI languages as BCP 47 names, most preferred first.
	Languages []string
	// ShortTime is the short time pattern of the user's locale (LOCALE_SSHORTTIME), "HH:mm" or "h:mm tt".
	ShortTime string
	// CursorWidth is SM_CXCURSOR.
	CursorWidth int
	// FontSmoothing is SPI_GETFONTSMOOTHING; FontSmoothingType is SPI_GETFONTSMOOTHINGTYPE.
	FontSmoothing     bool
	FontSmoothingType uint32
	HasFontSmoothing  bool
	// MessageFont is the family of lfMessageFont and MessageFontHeight its lfHeight in pixels at
	// the system DPI of SystemDPI; empty when SPI_GETNONCLIENTMETRICS failed.
	MessageFont       string
	MessageFontHeight int32
	SystemDPI         uint32
	// CaretBlinkMillis is GetCaretBlinkTime (0xFFFFFFFF is INFINITE) and DoubleClickMillis
	// GetDoubleClickTime.
	CaretBlinkMillis  uint32
	DoubleClickMillis uint32
}

// The fonts the settings name when Windows does not say (SPEC §25.8).
const (
	defaultUIFont       = "Segoe UI"
	defaultUIFontSize   = 9
	defaultMonoFont     = "Consolas"
	defaultMonoFontSize = 11
	minTextScalePercent = 100
	maxTextScalePercent = 225
	infiniteCaretBlink  = 0xFFFFFFFF
	defaultCursorSizePx = 32
	pointsPerInch       = 72
	defaultSystemDPI    = 96
)

// settingsOf turns what was read into the settings the engine takes:
//   - the brightness is dark when AppsUseLightTheme is 0 (absent: light);
//   - the accent color is the colorization color, made opaque;
//   - high contrast is HCF_HIGHCONTRASTON;
//   - the text scale is the percentage of the Accessibility setting (100 to 225) divided by 100;
//   - reduced motion is "animation effects off" (SPI_GETCLIENTAREAANIMATION false);
//   - the locales are the preferred UI languages, en-US when there are none;
//   - the 24-hour clock is a short time pattern with an H;
//   - the cursor size, the font smoothing (ClearType is sub-pixel), the interface font and the
//     monospace font (Consolas 11 pt);
//   - the caret blink time and the double-click time.
//
// Flutter: shell/platform/windows/settings_plugin.cc SettingsPlugin::SendSettings
func settingsOf(in settingsInput) platform.Settings {
	s := platform.Settings{
		PlatformBrightness:  ui.BrightnessLight,
		TextScaleFactor:     textScale(in),
		HighContrast:        in.HighContrastFlags&win32.HCFHighContrastOn != 0,
		DisableAnimations:   in.HasClientAreaAnimation && !in.ClientAreaAnimation,
		CursorSize:          in.CursorWidth,
		MonospaceFontFamily: defaultMonoFont, MonospaceFontSize: defaultMonoFontSize,
		UIFontFamily: defaultUIFont, UIFontSize: defaultUIFontSize,
		AlwaysUse24HourFormat: strings.Contains(in.ShortTime, "H"),
	}
	if darkTitleBar(in.AppsUseLightTheme, in.HasAppsUseLightTheme) {
		s.PlatformBrightness = ui.BrightnessDark
	}
	if in.HasColorization {
		s.AccentColor = ui.NewColor(0xFF000000 | in.Colorization&0x00FFFFFF)
		s.HasAccentColor = true
	}
	if s.CursorSize <= 0 {
		s.CursorSize = defaultCursorSizePx
	}
	for _, tag := range in.Languages {
		if l, ok := ui.ParseLocale(tag); ok {
			s.Locales = append(s.Locales, l)
		}
	}
	if len(s.Locales) == 0 {
		s.Locales = []ui.Locale{ui.NewLocale("en", "US")}
	}
	if in.HasFontSmoothing {
		switch {
		case !in.FontSmoothing:
			s.FontAntialiasing = platform.FontAntialiasingNone
		case in.FontSmoothingType == win32.FEFontSmoothingClearType:
			s.FontAntialiasing = platform.FontAntialiasingRGBA
		default:
			s.FontAntialiasing = platform.FontAntialiasingGrayscale
		}
	}
	if in.MessageFont != "" {
		s.UIFontFamily = in.MessageFont
		if h := in.MessageFontHeight; h != 0 {
			dpi := in.SystemDPI
			if dpi == 0 {
				dpi = defaultSystemDPI
			}
			// A negative lfHeight is the height of the characters in pixels, a positive one the
			// height of the cell; the size in points is the same either way for the message font.
			s.UIFontSize = math.Round(math.Abs(float64(h)) * pointsPerInch / float64(dpi))
		}
	}
	switch in.CaretBlinkMillis {
	case 0: // the call failed
	case infiniteCaretBlink:
		s.CaretBlinkTime = ui.CaretBlinkNever
	default:
		s.CaretBlinkTime = time.Duration(in.CaretBlinkMillis) * time.Millisecond
	}
	if in.DoubleClickMillis > 0 {
		s.DoubleClickTime = time.Duration(in.DoubleClickMillis) * time.Millisecond
	}
	return s
}

// textScale is the text scale factor: the percentage of the registry value TextScaleFactor
// (Settings → Accessibility → Text size) divided by 100, within the 100 to 225 that Windows allows;
// 1 when the value is absent. The scale of the display (DPI) is not in it: it is the window's scale.
func textScale(in settingsInput) float64 {
	if !in.HasTextScale || in.TextScalePercent == 0 {
		return 1
	}
	p := min(max(in.TextScalePercent, minTextScalePercent), maxTextScalePercent)
	return float64(p) / 100
}
