package text

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/opts"
)

// BundledPrefix starts the Path of a FontFile that is a bundled font (AddBundled)
// rather than a file on disk.
//
// Flutter: no counterpart
const BundledPrefix = "bundled:"

// LoadedPrefix starts the Path of a FontFile that is a font the app loaded at run
// time (AddFont), followed by the number of the font in the collection.
//
// Flutter: no counterpart
const LoadedPrefix = "loaded:"

// FontFile is one font of a fallback list: a file on disk (or a bundled font) and
// the face in it. Its fields are those of the fontconfig package's File.
//
// Flutter: no counterpart
type FontFile struct {
	// Path is the font file, BundledPrefix followed by the family of a bundled
	// font, or LoadedPrefix followed by the number of a loaded font.
	Path string
	// Index is the face in a font collection (the low 16 bits). The high 16 bits
	// that fontconfig sets for a named instance of a variable font are ignored:
	// the caller sets the variable axes (the weight, for one) with Variation.
	Index int
	// Family is the family name of the font, or "" when unknown.
	Family string
}

// IsBundled reports whether the font is a bundled font.
//
// Flutter: no counterpart
func (f FontFile) IsBundled() bool { return strings.HasPrefix(f.Path, BundledPrefix) }

// IsLoaded reports whether the font is one that the app loaded with AddFont.
//
// Flutter: no counterpart
func (f FontFile) IsLoaded() bool { return strings.HasPrefix(f.Path, LoadedPrefix) }

// Variation is the value of one variable-font axis.
//
// Flutter: dart:ui FontVariation
type Variation struct {
	// Tag is the four-letter axis tag, such as "wght".
	Tag string
	// Value is the value in design units.
	Value float32
}

// Matcher says which font files fit a family and style, best first. The fontconfig
// package implements it on Linux; without it the Collection scans the font
// directories itself.
//
// Flutter: no counterpart
type Matcher interface {
	// Match returns the fallback order for the family (which can be a generic name
	// such as "sans-serif"), the OpenType weight (400 regular), the slant and the
	// BCP 47 language tag ("" for the default).
	Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]FontFile, error)
}

// Aliaser is a Matcher that also knows which other family names a family stands
// for, such as Liberation Sans for Arial. A Collection uses it when the Matcher
// implements it: a family that is not generic gets the system fonts of its own family
// and of these names, and no other (Collection.Match).
//
// Flutter: no counterpart
type Aliaser interface {
	// Aliases returns the other names of the family, most preferred first; none for a
	// family that the configuration says nothing about.
	Aliases(ctx context.Context, family string) ([]string, error)
}

// genericFamilies are the family names that stand for a class of fonts instead of one family,
// as fontconfig and CSS use them. The Matcher gives the whole list of fonts for each.
var genericFamilies = map[string]bool{
	"sans-serif": true, "serif": true, "monospace": true, "cursive": true, "fantasy": true,
	"emoji": true, "math": true,
}

// IsGenericFamily says whether the family name is a generic one (sans-serif, serif,
// monospace, cursive, fantasy, emoji or math), which stands for the system's fonts of that kind.
//
// Flutter: no counterpart
func IsGenericFamily(family string) bool {
	return genericFamilies[strings.ToLower(strings.TrimSpace(family))]
}

// Fallbacker is a Matcher that can also find a font for one character that the fonts of
// Match do not have. A Collection uses it when the Matcher implements it.
//
// Flutter: no counterpart
type Fallbacker interface {
	// Fallback returns a font with a glyph for r, or false when there is none.
	Fallback(ctx context.Context, family string, weight int, italic bool, lang string, r rune) (FontFile, bool, error)
}

// CollectionOpts are the optional parameters of NewCollection.
//
// Flutter: no counterpart
type CollectionOpts struct {
	// Matcher decides the fallback order. When nil, the Collection scans the system
	// font directories with go-text's fontscan, and keeps its index in CacheDir.
	Matcher Matcher
	// CacheDir is where the fontscan index is kept. It defaults to
	// $XDG_CACHE_HOME/glutter/fonts (~/.cache/glutter/fonts).
	CacheDir string
	// Logger receives the non-fatal problems fontscan finds (fonts it cannot read).
	// It defaults to a logger that discards them.
	Logger interface {
		Printf(format string, args ...any)
	}
}

// Collection finds and loads fonts: the bundled ones, and the system fonts in the
// order the Matcher (or the font scan) gives. It caches the loaded faces. It is
// safe for concurrent use, but a font.Face it returns is not: each variation set
// has its own Face, which one goroutine at a time may use.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection
type Collection struct {
	opts CollectionOpts

	mu      sync.Mutex
	bundled map[string][]byte // normalized family → font data
	// loaded are the fonts of AddFont, by normalized family, in the order they were added.
	loaded     map[string][]*loadedFont
	loadedByID map[string]*loadedFont // FontFile.Path → font
	generation uint64                 // grows with every change of the loaded or bundled fonts
	aliasCache map[string][]string    // normalized family → the Aliaser's answer
	fonts      map[fontKey]*font.Font
	faces      map[faceKey]*font.Face
	scan       *scanMatcher // created on first use when opts.Matcher is nil
}

// fontKey identifies a parsed font: the file and the face in it.
type fontKey struct {
	path  string
	index int
}

// faceKey identifies a face: a font and its variation settings.
type faceKey struct {
	fontKey
	variations string
}

// NewCollection makes an empty font collection. More than one options value
// panics with MultipleOptions.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection::FontCollection
func NewCollection(opts ...CollectionOpts) *Collection {
	return &Collection{
		opts:       oneOpts(opts),
		bundled:    map[string][]byte{},
		loaded:     map[string][]*loadedFont{},
		loadedByID: map[string]*loadedFont{},
		aliasCache: map[string][]string{},
		fonts:      map[fontKey]*font.Font{},
		faces:      map[faceKey]*font.Face{},
	}
}

// normalizeFamily is how family names are compared: case and blanks do not count.
func normalizeFamily(family string) string { return font.NormalizeFamily(family) }

// AddBundled registers a font that is part of the program under a family name, such
// as Roboto or MaterialIcons. A request for that family finds it before any system
// font. data is the content of a TrueType or OpenType file (a collection is read at
// index 0). It fails with FontBundledInvalid when the data is not a font.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection::RegisterFont
func (c *Collection) AddBundled(family string, data []byte) error {
	if _, err := parseFont(bytes.NewReader(data), 0); err != nil {
		return errors.NewErr("FontBundledInvalid", err).SetMetadata("family", family)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bundled[normalizeFamily(family)] = data
	c.generation++
	return nil
}

// loadedFont is a font added with AddFont and what Match needs to rank it.
type loadedFont struct {
	path   string // LoadedPrefix + the number
	family string // as the app gave it
	data   []byte
	sum    [sha256.Size]byte
	// weight is the OpenType weight of the font (OS/2), italic whether it is slanted.
	weight int
	italic bool
	// wghtMin and wghtMax are the range of the font's wght axis; both are 0 when it has none.
	wghtMin, wghtMax int
	// italAxis is whether the font has an ital axis.
	italAxis bool
}

// distance says how far the font is from the wanted style, smaller is closer: a
// different slant counts more than any difference in weight, and a font that is lighter
// or heavier than wanted, on the side that the CSS font matching tries last, comes after
// one that is as far on the other side. A variable font is at distance 0 for every weight of its wght axis.
func (f *loadedFont) distance(weight int, italic bool) int {
	d := 0
	if !f.italAxis && f.italic != italic {
		d += 100000
	}
	lo, hi := f.weight, f.weight
	if f.wghtMax > 0 {
		lo, hi = f.wghtMin, f.wghtMax
	}
	switch {
	case weight < lo:
		d += 2 * (lo - weight)
		if weight <= 500 { // wanted a light weight: a heavier font is the worse side
			d++
		}
	case weight > hi:
		d += 2 * (weight - hi)
		if weight > 500 {
			d++
		}
	}
	return d
}

// AddFont registers a font that the app loads at run time under a family name. A family can
// hold several fonts (regular, bold, italic...): Match puts the closest to the requested
// weight and slant first, then the others of the family. The weight and the slant come from the
// font (its OS/2 table; a variable font counts as every weight of its wght axis). When family is empty, the
// family name of the font is used. data is a TrueType or OpenType file (a collection is read at index 0); the
// Collection keeps a copy. The same bytes under the same family are added once. The font wins over a bundled font of the same
// family. It fails with FontDataInvalid when data is not a font, and with FontFamilyMissing when family
// is empty and the font names none.
//
// Flutter: dart:ui loadFontFromList
func (c *Collection) AddFont(family string, data []byte) error {
	desc, axes, err := describeFont(data)
	if err != nil {
		return errors.NewErr("FontDataInvalid", err).SetMetadata("family", family, "bytes", len(data))
	}
	if family == "" {
		family = desc.Family
	}
	if strings.TrimSpace(family) == "" {
		return errors.New("FontFamilyMissing", "the font has no family name and the call gave none", "bytes", len(data))
	}
	lf := &loadedFont{family: family, sum: sha256.Sum256(data), weight: int(desc.Aspect.Weight), italic: desc.Aspect.Style == font.StyleItalic}
	if lf.weight == 0 {
		lf.weight = 400
	}
	for _, a := range axes {
		switch a.Tag.String() {
		case "wght":
			lf.wghtMin, lf.wghtMax = int(a.Minimum), int(a.Maximum)
		case "ital":
			lf.italAxis = true
		}
	}
	key := normalizeFamily(family)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, other := range c.loaded[key] {
		if other.sum == lf.sum {
			return nil
		}
	}
	lf.data = bytes.Clone(data)
	lf.path = LoadedPrefix + strconv.Itoa(len(c.loadedByID)+1)
	c.loaded[key] = append(c.loaded[key], lf)
	c.loadedByID[lf.path] = lf
	c.generation++
	return nil
}

// Generation is a number that grows whenever the loaded or the bundled fonts change. A
// cache of what Match and Face returned is valid while it stays the same.
//
// Flutter: no counterpart
func (c *Collection) Generation() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// describeFont parses the font at index 0 of data: its family and aspect, and the axes of its fvar table.
func describeFont(data []byte) (font.Description, []tables.VariationAxisRecord, error) {
	loaders, err := ot.NewLoaders(bytes.NewReader(data))
	if err != nil {
		return font.Description{}, nil, err
	}
	if len(loaders) == 0 {
		return font.Description{}, nil, io.ErrUnexpectedEOF // a file with no font in it
	}
	ft, err := font.NewFont(loaders[0])
	if err != nil {
		return font.Description{}, nil, err
	}
	var axes []tables.VariationAxisRecord
	if raw, err := loaders[0].RawTable(ot.MustNewTag("fvar")); err == nil {
		if fvar, _, err := tables.ParseFvar(raw); err == nil {
			axes = fvar.Axis
		}
	}
	return ft.Describe(), axes, nil
}

// Match returns the fonts for the family, best first: the fonts the app loaded under
// that family (the closest to the weight and slant first), the bundled font of that
// family if there is one, then the system fonts in the order of the Matcher or of
// the font scan (only TrueType and OpenType files, which go-text reads). For a generic
// family (IsGenericFamily) that is the Matcher's whole list, which goes on with the
// fonts that follow in the user's configuration. For any other family it is only the
// system fonts that belong to it: those of the family itself, and those of the names that
// the Matcher says it stands for (Aliaser); a family that no installed font has gives none, so the
// caller can go on to the next family instead of getting the system's default font. weight is an OpenType weight (400 regular, 700 bold), lang a BCP 47
// tag or "". It fails with FontMatchCanceled when ctx is done, with FontScanFailed
// when the fonts cannot be scanned; the Matcher's own error is returned unchanged.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection::GetMinikinFontCollectionForFamilies
func (c *Collection) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]FontFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontMatchCanceled", err).SetMetadata("family", family)
	}
	var out []FontFile
	c.mu.Lock()
	key := normalizeFamily(family)
	_, bundled := c.bundled[key]
	loaded := slices.Clone(c.loaded[key])
	c.mu.Unlock()
	slices.SortStableFunc(loaded, func(a, b *loadedFont) int { return a.distance(weight, italic) - b.distance(weight, italic) })
	for _, lf := range loaded {
		out = append(out, FontFile{Path: lf.path, Family: lf.family})
	}
	if bundled {
		out = append(out, FontFile{Path: BundledPrefix + family, Family: family})
	}
	matcher, err := c.matcher()
	if err != nil {
		return nil, err
	}
	files, err := matcher.Match(ctx, family, weight, italic, lang)
	if err != nil {
		return nil, err
	}
	belongs := func(FontFile) bool { return true }
	if !IsGenericFamily(family) {
		names := map[string]bool{normalizeFamily(family): true}
		if al, ok := matcher.(Aliaser); ok {
			aliases, err := c.aliases(ctx, al, family)
			if err != nil {
				return nil, err
			}
			for _, a := range aliases {
				names[normalizeFamily(a)] = true
			}
		}
		belongs = func(f FontFile) bool { return names[normalizeFamily(f.Family)] }
	}
	for _, f := range files {
		if supportedFont(f.Path) && belongs(f) {
			out = append(out, f)
		}
	}
	return out, nil
}

// aliases asks the Matcher for the other names of a family once.
func (c *Collection) aliases(ctx context.Context, al Aliaser, family string) ([]string, error) {
	key := normalizeFamily(family)
	c.mu.Lock()
	cached, ok := c.aliasCache[key]
	c.mu.Unlock()
	if ok {
		return cached, nil
	}
	names, err := al.Aliases(ctx, family)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.aliasCache[key] = names
	c.mu.Unlock()
	return names, nil
}

// supportedFont says whether go-text can read the file, judging by its extension:
// TrueType and OpenType fonts and their collections. Fontconfig also lists Type 1
// and bitmap fonts (.pfb, .pcf, .bdf), which are skipped.
func supportedFont(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc", ".otc":
		return true
	}
	return false
}

// Fallback returns a system font that has a glyph for r, when the fonts of Match do not; it
// is false when there is none (or the Matcher cannot look for a character). It fails like Match.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection::GetFallbackFontForRune
func (c *Collection) Fallback(ctx context.Context, family string, weight int, italic bool, lang string, r rune) (FontFile, bool, error) {
	if err := ctx.Err(); err != nil {
		return FontFile{}, false, errors.NewErr("FontMatchCanceled", err).SetMetadata("family", family)
	}
	matcher, err := c.matcher()
	if err != nil {
		return FontFile{}, false, err
	}
	fb, ok := matcher.(Fallbacker)
	if !ok {
		return FontFile{}, false, nil
	}
	f, found, err := fb.Fallback(ctx, family, weight, italic, lang, r)
	if err != nil || !found || !supportedFont(f.Path) {
		return FontFile{}, false, err
	}
	return f, true, nil
}

// matcher returns the Matcher of the options, or the font scan.
func (c *Collection) matcher() (Matcher, error) {
	if c.opts.Matcher != nil {
		return c.opts.Matcher, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scan == nil {
		s, err := newScanMatcher(c.opts.CacheDir, c.opts.Logger)
		if err != nil {
			return nil, err
		}
		c.scan = s
	}
	return c.scan, nil
}

// Face returns the face of the font with the variable axes set. The same file,
// index and variations give the same Face. It fails with FontLoadFailed when the
// file cannot be read or is not a font, or has no face at the index, and with
// FontLoadCanceled when ctx is done. A variation whose tag is not four letters fails
// with FontVariationInvalid.
//
// Flutter: engine/src/flutter/third_party/txt/src/txt/font_collection.cc FontCollection::CreateMinikinFontCollection
func (c *Collection) Face(ctx context.Context, f FontFile, variations ...Variation) (*font.Face, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("FontLoadCanceled", err).SetMetadata("path", f.Path)
	}
	for _, v := range variations {
		if len(v.Tag) != 4 {
			return nil, errors.New("FontVariationInvalid", "a variation axis tag has four letters", "tag", v.Tag)
		}
	}
	fk := fontKey{f.Path, f.Index & 0xffff}
	key := faceKey{fk, variationsKey(variations)}
	c.mu.Lock()
	defer c.mu.Unlock()
	if face := c.faces[key]; face != nil {
		return face, nil
	}
	ft := c.fonts[fk]
	if ft == nil {
		var err error
		if ft, err = c.load(fk); err != nil {
			return nil, err
		}
		c.fonts[fk] = ft
	}
	face := font.NewFace(ft)
	if len(variations) > 0 {
		vs := make([]font.Variation, len(variations))
		for i, v := range variations {
			vs[i] = font.Variation{Tag: ot.NewTag(v.Tag[0], v.Tag[1], v.Tag[2], v.Tag[3]), Value: v.Value}
		}
		face.SetVariations(vs)
	}
	c.faces[key] = face
	return face, nil
}

// load parses the font of a loaded font, of a bundled family or of a file.
func (c *Collection) load(k fontKey) (*font.Font, error) {
	if lf, ok := c.loadedByID[k.path]; ok {
		ft, err := parseFont(bytes.NewReader(lf.data), k.index)
		if err != nil {
			return nil, errors.NewErr("FontLoadFailed", err).SetMetadata("path", k.path, "index", k.index)
		}
		return ft, nil
	}
	if family, ok := strings.CutPrefix(k.path, BundledPrefix); ok {
		data, found := c.bundled[normalizeFamily(family)]
		if !found {
			return nil, errors.New("FontLoadFailed", "there is no bundled font with that family", "path", k.path)
		}
		ft, err := parseFont(bytes.NewReader(data), k.index)
		if err != nil {
			return nil, errors.NewErr("FontLoadFailed", err).SetMetadata("path", k.path, "index", k.index)
		}
		return ft, nil
	}
	file, err := os.Open(k.path)
	if err != nil {
		return nil, errors.NewErr("FontLoadFailed", err).SetMetadata("path", k.path, "index", k.index)
	}
	defer func() { _ = file.Close() }() // the font is read; a failed close of a read-only file loses nothing
	ft, err := parseFont(file, k.index)
	if err != nil {
		return nil, errors.NewErr("FontLoadFailed", err).SetMetadata("path", k.path, "index", k.index)
	}
	return ft, nil
}

// parseFont reads the font at index of a font file or collection.
func parseFont(r font.Resource, index int) (*font.Font, error) {
	loaders, err := ot.NewLoaders(r)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(loaders) {
		return nil, fmt.Errorf("the font file has %d faces, there is none at index %d", len(loaders), index)
	}
	return font.NewFont(loaders[index])
}

// variationsKey makes the part of a face's cache key that the variations give. The
// order of the variations does not matter, and a later value of the same axis wins.
func variationsKey(vs []Variation) string {
	if len(vs) == 0 {
		return ""
	}
	last := map[string]float32{}
	for _, v := range vs {
		last[v.Tag] = v.Value
	}
	tags := make([]string, 0, len(last))
	for tag := range last {
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	var b strings.Builder
	for _, tag := range tags {
		fmt.Fprintf(&b, "%s=%g;", tag, last[tag])
	}
	return b.String()
}

// oneOpts returns the one options value of a trailing `opts ...XOpts`
// parameter (SPEC §12.1). It wraps opts.First, because a parameter named opts
// hides the package.
func oneOpts[T any](o []T) T { return opts.First(o) }
