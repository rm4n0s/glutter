package text

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/rm4n0s/errors"
)

// BundledPrefix starts the Path of a FontFile that is a bundled font (AddBundled)
// rather than a file on disk.
//
// Flutter: no counterpart
const BundledPrefix = "bundled:"

// FontFile is one font of a fallback list: a file on disk (or a bundled font) and
// the face in it. Its fields are those of the fontconfig package's File.
//
// Flutter: no counterpart
type FontFile struct {
	// Path is the font file, or BundledPrefix followed by the family of a bundled
	// font.
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
	fonts   map[fontKey]*font.Font
	faces   map[faceKey]*font.Face
	scan    *scanMatcher // created on first use when opts.Matcher is nil
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
		opts:    oneOpts(opts),
		bundled: map[string][]byte{},
		fonts:   map[fontKey]*font.Font{},
		faces:   map[faceKey]*font.Face{},
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
	return nil
}

// Match returns the fonts for the family, best first: the bundled font of that
// family if there is one, then the system fonts in the order of the Matcher or of
// the font scan (only TrueType and OpenType files, which go-text reads). weight is an OpenType weight (400 regular, 700 bold), lang a BCP 47
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
	_, bundled := c.bundled[normalizeFamily(family)]
	c.mu.Unlock()
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
	for _, f := range files {
		if supportedFont(f.Path) {
			out = append(out, f)
		}
	}
	return out, nil
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

// load parses the font of a bundled family or of a file.
func (c *Collection) load(k fontKey) (*font.Font, error) {
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

// oneOpts returns the one options value, or the zero value when there is none.
// More than one panics with the tag MultipleOptions (SPEC §12.1).
//
// TODO(step 5.1): use the shared opts.First helper.
func oneOpts[T any](opts []T) T {
	var zero T
	switch len(opts) {
	case 0:
		return zero
	case 1:
		return opts[0]
	}
	panic(errors.New("MultipleOptions", "more than one options value was passed"))
}
