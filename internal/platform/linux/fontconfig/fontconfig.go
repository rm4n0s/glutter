package fontconfig

import (
	"context"
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// File is one font that fontconfig matched.
//
// Flutter: no counterpart
type File struct {
	// Path is the font file.
	Path string
	// Index is fontconfig's FC_INDEX: the face in a collection in the low 16 bits,
	// and for a variable font, 1 + the named instance in the high 16 bits (0 means
	// the default instance).
	Index int
	// Family is the first family name of the font, or "".
	Family string
}

// fcFontSet mirrors struct _FcFontSet of fontconfig.h, which fontconfig
// allocates and the Go code reads (never writes).
type fcFontSet struct {
	nfont int32
	sfont int32
	fonts uintptr // FcPattern **
}

// Config is a loaded fontconfig configuration with the system fonts. It is safe
// for concurrent use.
//
// Flutter: no counterpart
type Config struct {
	lib    *library
	mu     sync.Mutex
	config uintptr // FcConfig *
	// scratch is C memory for what the library writes through a pointer: a char
	// pointer or an int.
	scratch uintptr
}

// Open loads libfontconfig and the user's fontconfig configuration, which reads
// the configuration files and scans the font directories. It fails with
// FontconfigLoadFailed when the library is not there, with FontconfigInitFailed
// when it cannot load a configuration, and with FontconfigCanceled when ctx is
// done. Close the Config when done.
//
// Flutter: no counterpart
func Open(ctx context.Context) (*Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontconfigCanceled", err)
	}
	lib, err := loadLibrary()
	if err != nil {
		return nil, err
	}
	return newConfig(lib)
}

func newConfig(lib *library) (*Config, error) {
	config := lib.initLoadConfigAndFonts()
	if config == 0 {
		return nil, errors.New("FontconfigInitFailed", "fontconfig could not load a configuration")
	}
	return &Config{lib: lib, config: config, scratch: dl.Calloc(1, 16)}, nil
}

// Close releases the configuration. The Config must not be used after.
//
// Flutter: no counterpart
func (c *Config) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config != 0 {
		c.lib.configDestroy(c.config)
		dl.Free(c.scratch)
		c.config, c.scratch = 0, 0
	}
}

// openTypeWeights maps an OpenType weight (100 to 900) to fontconfig's, the table
// of FcWeightFromOpenType.
var openTypeWeights = [...]struct{ ot, fc int }{
	{0, 0}, {100, 0}, {200, 40}, {300, 50}, {350, 55}, {380, 75}, {400, 80},
	{500, 100}, {600, 180}, {700, 200}, {800, 205}, {900, 210}, {1000, 210},
}

// weightToFontconfig converts an OpenType weight to a FC_WEIGHT value by linear
// interpolation, as FcWeightFromOpenType does.
func weightToFontconfig(ot int) int {
	t := openTypeWeights[:]
	if ot <= t[0].ot {
		return t[0].fc
	}
	for i := 1; i < len(t); i++ {
		if ot <= t[i].ot {
			a, b := t[i-1], t[i]
			return a.fc + (b.fc-a.fc)*(ot-a.ot)/(b.ot-a.ot)
		}
	}
	return t[len(t)-1].fc
}

// Match returns the font files for the family in the order of fontconfig's
// fallback: the best match first, then the fonts the family and language
// substitution rules and the user's preferences put next. family may be a
// generic name such as "sans-serif" or "emoji"; weight is an OpenType weight
// (400 regular, 700 bold); lang is a BCP 47 tag such as "ja" or "" for the
// default. It fails with FontconfigMatchFailed when fontconfig cannot sort the
// fonts, with FontconfigPatternFailed when it cannot build the query, with
// FontconfigClosed after Close, and with FontconfigCanceled when ctx is done.
//
// Flutter: no counterpart (SkFontMgr_fontconfig::onMatchFamilyStyle)
func (c *Config) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontconfigCanceled", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == 0 {
		return nil, errors.New("FontconfigClosed", "the fontconfig configuration is closed")
	}
	pat := c.lib.patternCreate()
	if pat == 0 {
		return nil, errors.New("FontconfigPatternFailed", "fontconfig could not create a pattern")
	}
	defer c.lib.patternDestroy(pat)

	slant := int32(0) // FC_SLANT_ROMAN
	if italic {
		slant = 100 // FC_SLANT_ITALIC
	}
	if err := c.addString(pat, objFamily, family); err != nil {
		return nil, err
	}
	if lang != "" {
		if err := c.addString(pat, objLang, lang); err != nil {
			return nil, err
		}
	}
	if err := c.addInt(pat, objWeight, int32(weightToFontconfig(weight))); err != nil {
		return nil, err
	}
	if err := c.addInt(pat, objSlant, slant); err != nil {
		return nil, err
	}
	// Substitution applies the user's configuration (aliases, per-language rules,
	// emoji and CJK preferences); the default substitution fills in what is left.
	c.lib.configSubstitute(c.config, pat, fcMatchPattern)
	c.lib.defaultSubstitute(pat)

	set := c.lib.fontSort(c.config, pat, fcTrue, 0, c.scratch)
	if set == 0 {
		return nil, errors.New("FontconfigMatchFailed", "fontconfig could not sort the fonts",
			"family", family, "weight", weight, "italic", italic, "lang", lang)
	}
	defer c.lib.fontSetDestroy(set)
	return c.files(set), nil
}

func (c *Config) addString(pat uintptr, object, value string) error {
	obj, val := dl.CString(object), dl.CString(value)
	defer dl.Free(obj)
	defer dl.Free(val)
	if c.lib.patternAddString(pat, obj, val) == 0 {
		return errors.New("FontconfigPatternFailed", "fontconfig could not add to the pattern",
			"object", object, "value", value)
	}
	return nil
}

func (c *Config) addInt(pat uintptr, object string, value int32) error {
	obj := dl.CString(object)
	defer dl.Free(obj)
	if c.lib.patternAddInteger(pat, obj, value) == 0 {
		return errors.New("FontconfigPatternFailed", "fontconfig could not add to the pattern",
			"object", object, "value", value)
	}
	return nil
}

// files reads the patterns of a sorted font set. A pattern with no file is
// skipped.
func (c *Config) files(set uintptr) []File {
	fs := (*fcFontSet)(dl.Pointer(set))
	if fs.nfont <= 0 || fs.fonts == 0 {
		return nil
	}
	patterns := unsafe.Slice((*uintptr)(dl.Pointer(fs.fonts)), int(fs.nfont))
	out := make([]File, 0, len(patterns))
	for _, pat := range patterns {
		path, ok := c.getString(pat, objFile)
		if !ok {
			continue
		}
		f := File{Path: path}
		if idx, ok := c.getInt(pat, objIndex); ok {
			f.Index = int(idx)
		}
		f.Family, _ = c.getString(pat, objFamily)
		out = append(out, f)
	}
	return out
}

func (c *Config) getString(pat uintptr, object string) (string, bool) {
	obj := dl.CString(object)
	defer dl.Free(obj)
	if c.lib.patternGetString(pat, obj, 0, c.scratch) != fcResultMatch {
		return "", false
	}
	return dl.GoString(*(*uintptr)(dl.Pointer(c.scratch))), true
}

func (c *Config) getInt(pat uintptr, object string) (int32, bool) {
	obj := dl.CString(object)
	defer dl.Free(obj)
	if c.lib.patternGetInteger(pat, obj, 0, c.scratch) != fcResultMatch {
		return 0, false
	}
	return *(*int32)(dl.Pointer(c.scratch)), true
}
