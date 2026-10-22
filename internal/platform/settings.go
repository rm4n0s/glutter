package platform

import "github.com/rm4n0s/glutter/ui"

// Settings is the system-wide configuration the app follows.
//
// Flutter: ui/platform_dispatcher.dart PlatformConfiguration
type Settings struct {
	// PlatformBrightness is the system color scheme.
	PlatformBrightness ui.Brightness
	// AccentColor is the system accent color. It is meaningful only when
	// HasAccentColor is true.
	AccentColor ui.Color
	// HasAccentColor is whether the system has an accent color.
	HasAccentColor bool
	// TextScaleFactor is the system text scale; 0 means 1.
	TextScaleFactor float64
	// DisableAnimations is whether the user asked for reduced motion.
	DisableAnimations bool
	// HighContrast is whether the user asked for high contrast.
	HighContrast bool
	// Locales lists the user's locales, most preferred first.
	Locales []ui.Locale
	// CursorSize is the size of the mouse cursor in pixels; 0 means the
	// platform default.
	CursorSize int
	// CursorTheme is the name of the XCursor theme; empty means the
	// platform default.
	CursorTheme string
	// FontAntialiasing is how the system wants text smoothed.
	FontAntialiasing FontAntialiasing
	// FontHinting is how strongly the system wants glyph outlines fitted to
	// the pixel grid.
	FontHinting FontHinting
	// UIFontFamily and UIFontSize are the family and the size, in points, of the
	// desktop's interface font (GNOME's font-name); empty and 0 when the platform
	// does not say.
	UIFontFamily string
	UIFontSize   float64
	// MonospaceFontFamily and MonospaceFontSize are the family and the size, in
	// points, of the desktop's monospace font (GNOME's monospace-font-name); empty and
	// 0 when the platform does not say.
	MonospaceFontFamily string
	MonospaceFontSize   float64
}

// FontAntialiasing is how the system wants text smoothed. The zero value,
// FontAntialiasingUnspecified, means the platform does not say.
//
// Flutter: no counterpart
type FontAntialiasing int

// The values of FontAntialiasing.
const (
	// FontAntialiasingUnspecified is the zero value.
	//
	// Flutter: no counterpart
	FontAntialiasingUnspecified FontAntialiasing = iota
	// FontAntialiasingNone draws glyphs without smoothing.
	//
	// Flutter: no counterpart
	FontAntialiasingNone
	// FontAntialiasingGrayscale smooths glyphs with gray levels.
	//
	// Flutter: no counterpart
	FontAntialiasingGrayscale
	// FontAntialiasingRGBA smooths glyphs per color channel (sub-pixel).
	//
	// Flutter: no counterpart
	FontAntialiasingRGBA
)

// FontHinting is how strongly the system wants glyph outlines fitted to the
// pixel grid. The zero value, FontHintingUnspecified, means the platform does
// not say.
//
// Flutter: no counterpart
type FontHinting int

// The values of FontHinting.
const (
	// FontHintingUnspecified is the zero value.
	//
	// Flutter: no counterpart
	FontHintingUnspecified FontHinting = iota
	// FontHintingNone does no hinting.
	//
	// Flutter: no counterpart
	FontHintingNone
	// FontHintingSlight hints vertically only.
	//
	// Flutter: no counterpart
	FontHintingSlight
	// FontHintingMedium hints moderately.
	//
	// Flutter: no counterpart
	FontHintingMedium
	// FontHintingFull hints fully.
	//
	// Flutter: no counterpart
	FontHintingFull
)
