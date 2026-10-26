//go:build linux && !android

package fontconfig

import (
	"context"
	"slices"
	"strings"
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
	// defaults are the family names that the configuration puts after a family that
	// it knows nothing about (the default sans-serif and what it stands for); nil until
	// the first call of Aliases.
	defaults []string
	// charsets are the character sets (FcCharSet *) of the fonts that Match listed, by file and face. Each
	// is a reference that the Config holds until Close, so it outlives the font set of the Match.
	charsets map[faceKey]uintptr
}

// faceKey is a font file and the face in it (the low 16 bits of File.Index).
type faceKey struct {
	path  string
	index int
}

// probeFamily is a family name that no font and no configuration rule mentions.
const probeFamily = "glutter-probe-family-with-no-font"

// Open loads libfontconfig and the user's fontconfig configuration, which reads
// the configuration files and scans the font directories. It fails with
// FontconfigLoadFailed when the library is not there, with FontconfigInitFailed
// when it cannot load a configuration, and with FontconfigCanceled when ctx is
// done. Close the Config when done.
//
// Flutter: no counterpart
func Open(ctx context.Context) (*Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontconfigCanceled, err)
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
		return nil, errors.New(tagFontconfigInitFailed, "fontconfig could not load a configuration")
	}
	return &Config{lib: lib, config: config, scratch: dl.Calloc(1, 16), charsets: map[faceKey]uintptr{}}, nil
}

// Close releases the configuration. The Config must not be used after.
//
// Flutter: no counterpart
func (c *Config) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config != 0 {
		for _, cs := range c.charsets {
			c.lib.charSetDestroy(cs)
		}
		clear(c.charsets)
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
		return nil, errors.NewErr(tagFontconfigCanceled, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == 0 {
		return nil, errors.New(tagFontconfigClosed, "the fontconfig configuration is closed")
	}
	pat := c.lib.patternCreate()
	if pat == 0 {
		return nil, errors.New(tagFontconfigPatternFailed, "fontconfig could not create a pattern")
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
		return nil, errors.New(tagFontconfigMatchFailed, "fontconfig could not sort the fonts",
			"family", family, "weight", weight, "italic", italic, "lang", lang)
	}
	defer c.lib.fontSetDestroy(set)
	return c.files(set), nil
}

// Aliases returns the other family names that the user's configuration says the family
// stands for, in order: after FcConfigSubstitute the pattern lists the family and then, for
// Arial, Liberation Sans and its metric-compatible names, or for a family with a rule of the
// user, the names the rules add. The names that the configuration adds to every family (the
// default sans-serif and the fonts it stands for) are left out, so a family that nothing mentions
// has none. It fails like Match.
//
// Flutter: no counterpart (SkFontMgr_fontconfig::onMatchFamilyStyle accepts the strongly bound families)
func (c *Config) Aliases(ctx context.Context, family string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontconfigCanceled, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == 0 {
		return nil, errors.New(tagFontconfigClosed, "the fontconfig configuration is closed")
	}
	if c.defaults == nil {
		d, err := c.substitutedFamilies(probeFamily)
		if err != nil {
			return nil, err
		}
		c.defaults = append([]string{}, d...)
	}
	names, err := c.substitutedFamilies(family)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range names {
		if strings.EqualFold(n, family) || containsFold(c.defaults, n) || containsFold(out, n) {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

// Families lists the family names of the installed fonts, sorted without regard to case and
// without duplicates (FcFontList with the family as the only object). It fails with
// FontconfigListFailed when fontconfig cannot list the fonts, with FontconfigPatternFailed when it
// cannot build the query, with FontconfigClosed after Close, and with FontconfigCanceled when ctx is done.
//
// Flutter: no counterpart
func (c *Config) Families(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(tagFontconfigCanceled, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == 0 {
		return nil, errors.New(tagFontconfigClosed, "the fontconfig configuration is closed")
	}
	pat := c.lib.patternCreate()
	if pat == 0 {
		return nil, errors.New(tagFontconfigPatternFailed, "fontconfig could not create a pattern")
	}
	defer c.lib.patternDestroy(pat)
	objects := c.lib.objectSetCreate()
	if objects == 0 {
		return nil, errors.New(tagFontconfigPatternFailed, "fontconfig could not create an object set")
	}
	defer c.lib.objectSetDestroy(objects)
	name := dl.CString(objFamily)
	defer dl.Free(name)
	if c.lib.objectSetAdd(objects, name) == 0 {
		return nil, errors.New(tagFontconfigPatternFailed, "fontconfig could not add to the object set", "object", objFamily)
	}
	set := c.lib.fontList(c.config, pat, objects)
	if set == 0 {
		return nil, errors.New(tagFontconfigListFailed, "fontconfig could not list the fonts")
	}
	defer c.lib.fontSetDestroy(set)
	fs := (*fcFontSet)(dl.Pointer(set))
	if fs.nfont <= 0 || fs.fonts == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range unsafe.Slice((*uintptr)(dl.Pointer(fs.fonts)), int(fs.nfont)) {
		family, ok := c.getString(p, objFamily)
		if !ok || family == "" || seen[strings.ToLower(family)] {
			continue
		}
		seen[strings.ToLower(family)] = true
		out = append(out, family)
	}
	slices.SortFunc(out, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return out, nil
}

func containsFold(list []string, s string) bool {
	for _, e := range list {
		if strings.EqualFold(e, s) {
			return true
		}
	}
	return false
}

// substitutedFamilies is the list of the family names of a pattern for the family after the
// configuration's substitution (without the default substitution). The caller holds c.mu.
func (c *Config) substitutedFamilies(family string) ([]string, error) {
	pat := c.lib.patternCreate()
	if pat == 0 {
		return nil, errors.New(tagFontconfigPatternFailed, "fontconfig could not create a pattern")
	}
	defer c.lib.patternDestroy(pat)
	if err := c.addString(pat, objFamily, family); err != nil {
		return nil, err
	}
	c.lib.configSubstitute(c.config, pat, fcMatchPattern)
	var names []string
	obj := dl.CString(objFamily)
	defer dl.Free(obj)
	for n := int32(0); c.lib.patternGetString(pat, obj, n, c.scratch) == fcResultMatch; n++ {
		names = append(names, dl.GoString(*(*uintptr)(dl.Pointer(c.scratch))))
	}
	return names, nil
}

func (c *Config) addString(pat uintptr, object, value string) error {
	obj, val := dl.CString(object), dl.CString(value)
	defer dl.Free(obj)
	defer dl.Free(val)
	if c.lib.patternAddString(pat, obj, val) == 0 {
		return errors.New(tagFontconfigPatternFailed, "fontconfig could not add to the pattern",
			"object", object, "value", value)
	}
	return nil
}

func (c *Config) addInt(pat uintptr, object string, value int32) error {
	obj := dl.CString(object)
	defer dl.Free(obj)
	if c.lib.patternAddInteger(pat, obj, value) == 0 {
		return errors.New(tagFontconfigPatternFailed, "fontconfig could not add to the pattern",
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
		c.keepCharSet(pat, faceKey{f.Path, f.Index & 0xffff})
		out = append(out, f)
	}
	return out
}

// keepCharSet remembers the character set of a font pattern, which Covers answers from: a reference
// of the set, since the pattern goes with the font set of the Match. A font whose set is known is not
// read again.
func (c *Config) keepCharSet(pat uintptr, key faceKey) {
	if _, known := c.charsets[key]; known {
		return
	}
	obj := dl.CString(objCharSet)
	defer dl.Free(obj)
	if c.lib.patternGetCharSet(pat, obj, 0, c.scratch) != fcResultMatch {
		return
	}
	cs := *(*uintptr)(dl.Pointer(c.scratch))
	if cs = c.lib.charSetCopy(cs); cs != 0 {
		c.charsets[key] = cs
	}
}

// Covers says whether the font that an earlier Match listed (the file and the face in it) has a glyph for
// r, from the character set that fontconfig keeps for it: no font file is opened. known is false for a
// font that no Match listed, and after Close.
//
// Flutter: no counterpart
func (c *Config) Covers(path string, index int, r rune) (has, known bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == 0 || r < 0 {
		return false, false
	}
	cs, ok := c.charsets[faceKey{path, index & 0xffff}]
	if !ok {
		return false, false
	}
	return c.lib.charSetHasChar(cs, uint32(r)) != 0, true
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontconfigCanceled      = "FontconfigCanceled"
	tagFontconfigClosed        = "FontconfigClosed"
	tagFontconfigInitFailed    = "FontconfigInitFailed"
	tagFontconfigListFailed    = "FontconfigListFailed"
	tagFontconfigMatchFailed   = "FontconfigMatchFailed"
	tagFontconfigPatternFailed = "FontconfigPatternFailed"
)
