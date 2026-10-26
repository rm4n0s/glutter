package androidfonts

import (
	"io"
	"log/slog"
	"os"
	"path"
)

// The locations of the font configuration on a device (SkFontMgr_android_parser.cpp:
// LMP_SYSTEM_FONTS_FILE, PRODUCT_CUSTOMIZATION_FILE, and the folder of the fonts, which Skia
// builds from $ANDROID_ROOT).
const (
	// SystemFontsXML is the font configuration of Android 5 and later.
	SystemFontsXML = "/system/etc/fonts.xml"
	// SystemFontDir is the folder of the system fonts.
	SystemFontDir = "/system/fonts"
)

// customizationFiles are the fonts_customization.xml of the device builders: /oem/etc in Android 8
// and 9, /product/etc from Android 10 (Skia: PRODUCT_CUSTOMIZATION_FILE) and /vendor/etc.
var customizationFiles = []string{
	"/oem/etc/fonts_customization.xml",
	"/product/etc/fonts_customization.xml",
	"/vendor/etc/fonts_customization.xml",
}

// Options are the parameters of New.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android.h SkFontMgr_Android_CustomFonts)
type Options struct {
	// FontsXML are the font configuration files, read in order. Their families are added one
	// after the other, so a later file extends the earlier ones (an alias can name a family of an
	// earlier file). A file that is missing is skipped; New fails when no file gives a family.
	FontsXML []string
	// Customization are the fonts_customization.xml files of the device builder, read after
	// FontsXML. A file that is missing or malformed is skipped (the problem is logged). A relative font
	// file name in /<partition>/etc/fonts_customization.xml is in /<partition>/fonts; in any
	// other path it is in FontDir.
	Customization []string
	// FontDir is where the relative font file names of FontsXML are. It defaults to
	// /system/fonts.
	FontDir string
	// ReadFile reads a configuration file. It defaults to os.ReadFile.
	ReadFile func(path string) ([]byte, error)
	// FileExists says whether a font file exists. A font whose file does not exist is not in its
	// family, and a family with no font is left out, as Skia does with a file it cannot open. It
	// defaults to a check with os.Stat that the path is a regular file.
	FileExists func(path string) bool
	// HasGlyph says whether the face at index of a font file has a glyph for r. It is how Fallback
	// finds a font for a character. It defaults to the cmap of the font, read with go-text and
	// kept per file and index.
	HasGlyph func(path string, index int, r rune) bool
	// covers is HasGlyph that also says whether it could tell: the default one knows a font it cannot read.
	covers func(path string, index int, r rune) (has, known bool)
	// Locale is the BCP 47 tag of the language used when a request names none (the user's
	// locale). It is "" by default: the fonts then come in the order of the configuration.
	Locale string
	// Logger receives the problems of the configuration that are not fatal: a file that cannot
	// be read, an unknown element, a value that is not valid. It defaults to a logger that
	// discards them.
	Logger *slog.Logger
}

// DefaultOptions returns the options for a device: /system/etc/fonts.xml (the configuration of
// Android 5 and later; the older system_fonts.xml and fallback_fonts.xml are not read, Glutter
// runs on Android 8 and later), the fonts_customization.xml of /oem, /product and /vendor,
// and /system/fonts.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android_parser.cpp SkFontMgr_Android_Parser::GetSystemFontFamilies)
func DefaultOptions() Options {
	return Options{
		FontsXML:      []string{SystemFontsXML},
		Customization: append([]string(nil), customizationFiles...),
		FontDir:       SystemFontDir,
	}
}

// withDefaults fills what the options leave out.
func (o Options) withDefaults() Options {
	if o.FontDir == "" {
		o.FontDir = SystemFontDir
	}
	if o.ReadFile == nil {
		o.ReadFile = os.ReadFile
	}
	if o.FileExists == nil {
		o.FileExists = regularFileExists
	}
	if o.HasGlyph == nil {
		checker := newGlyphChecker()
		o.HasGlyph, o.covers = checker.hasGlyph, checker.covers
	}
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return o
}

// regularFileExists is the default Options.FileExists.
func regularFileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// customizationDir is the folder of the font files that a customization file names:
// /<partition>/fonts for /<partition>/etc/fonts_customization.xml, else fallback.
func customizationDir(file, fallback string) string {
	etc := path.Dir(file)
	if path.Base(etc) == "etc" {
		if partition := path.Dir(etc); partition != "/" && partition != "." {
			return path.Join(partition, "fonts")
		}
	}
	return fallback
}
