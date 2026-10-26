package layout

import (
	"context"
	"slices"
	"strconv"
	"unicode"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/ui"
)

// shapeSize is the size, in pixels, at which every run is shaped. HarfBuzz scales
// by whole pixels, so the runs are shaped at this size and the metrics are scaled to
// the size of the style; the advances are exact to 1/64 of a pixel at this size.
const shapeSize = 128

// defaultFamily is the family used when a style names none, and the last resort
// after the families a style names.
const defaultFamily = "sans-serif"

// EngineOpts are the optional parameters of NewEngine.
//
// Flutter: no counterpart
type EngineOpts struct {
	// OnFontError, when not nil, receives the problems with finding or loading a
	// font; the layout goes on with the next font.
	OnFontError func(error)
}

// Engine is the text engine: it makes paragraphs from the fonts of a
// text.Collection. It implements ui.ParagraphFactory. An Engine and its paragraphs
// are for the UI thread only (SPEC §13).
//
// Flutter: engine/src/flutter/lib/ui/text/paragraph_builder.cc ParagraphBuilder
type Engine struct {
	ctx   context.Context
	fonts *text.Collection
	opts  EngineOpts

	byStyle map[string]*styleFonts
	keyBuf  []byte // the key of the style that fontsFor looks up
	// recent are the styles that fontsFor found last, so that the spans of a line that repeat a few styles
	// find them without building a key.
	recent     [recentStyles]recentStyle
	recentNext int
	// generation is the generation of the font collection that byStyle was made for.
	generation uint64

	// shapes caches the glyphs of the runs that paragraphs shape (shapecache.go).
	shapes *shapeCache
	// analyses caches what is found out about a text before it is shaped (analysis.go).
	analyses    *lruCache[string, *analysis]
	analysisBuf []byte
	// noAnalysisCache turns that cache off: the tests compare with it.
	noAnalysisCache bool
	// noFastAnalysis turns off the shortcuts for text of one direction: the tests compare them with the general path.
	noFastAnalysis bool
	// faceSplits is the buffer that splitByFace returns.
	faceSplits []shaping.Input
	// segmenter splits the levels of an analysis by script; it keeps its buffers.
	segmenter shaping.Segmenter

	// paths and pictures cache the outlines and the color glyphs that paragraphs draw.
	paths    *lruCache[glyphKey, *ui.Path]
	pictures *lruCache[colorKey, *ui.Picture]
}

var (
	_ ui.FontLoadingEngine = (*Engine)(nil)
	_ ui.FontQueryEngine   = (*Engine)(nil)
)

// ResolveFont says which face the family resolves to: the first font of the family that
// loads, in the order of text.Collection.Match (the loaded fonts, the closest to the weight and slant
// first, then the bundled one, then the system's). It is false when the family has none, so a
// family that nothing has does not resolve to the system's default font.
//
// Flutter: no counterpart
func (e *Engine) ResolveFont(family string, weight ui.FontWeight, style ui.FontStyle) (ui.ResolvedFont, bool) {
	w := int(weight)
	if w == 0 {
		w = int(ui.FontWeightW400)
	}
	files, err := e.fonts.Match(e.ctx, family, w, style == ui.FontStyleItalic, "")
	if err != nil {
		e.fontError(err)
		return ui.ResolvedFont{}, false
	}
	for _, file := range files {
		face, err := e.fonts.Face(e.ctx, file)
		if err != nil {
			e.fontError(err)
			continue
		}
		desc := face.Describe()
		out := ui.ResolvedFont{Family: file.Family, Weight: int(desc.Aspect.Weight), Italic: desc.Aspect.Style == font.StyleItalic}
		if out.Family == "" {
			out.Family = desc.Family
		}
		switch {
		case file.IsLoaded():
			out.Origin = ui.FontOriginLoaded
		case file.IsBundled():
			out.Origin = ui.FontOriginBundled
		default:
			out.Origin, out.Path, out.Index = ui.FontOriginSystem, file.Path, file.Index&0xffff
		}
		return out, true
	}
	return ui.ResolvedFont{}, false
}

// InstalledFontFamilies lists the families of the system's, bundled and loaded fonts, sorted
// without regard to case. When the system's fonts cannot be listed, the others are.
//
// Flutter: no counterpart
func (e *Engine) InstalledFontFamilies() []string {
	families, err := e.fonts.Families(e.ctx)
	if err != nil {
		e.fontError(err)
	}
	return families
}

// LoadFontFromList adds a font to the engine's collection under a family name (the family inside the font
// when family is empty), so that styles can name it. What the engine resolved before is
// forgotten, so a family that was missing now finds the font. It fails like text.Collection.AddFont. It
// is for the UI thread.
//
// Flutter: dart:ui loadFontFromList
func (e *Engine) LoadFontFromList(list []byte, family string) error {
	return e.fonts.AddFont(family, list)
}

// NewEngine makes a text engine over a font collection. ctx bounds the loading of
// font files, which happens while a paragraph is built. More than one options value
// panics with MultipleOptions.
//
// Flutter: no counterpart
func NewEngine(ctx context.Context, fonts *text.Collection, opts ...EngineOpts) *Engine {
	return &Engine{ctx: ctx, fonts: fonts, opts: oneOpts(opts), byStyle: map[string]*styleFonts{}, shapes: newShapeCache(), analyses: newWeightedLRU(maxAnalyses, maxAnalysisBytes, weighAnalysis),
		paths: newLRU[glyphKey, *ui.Path](maxCachedGlyphs), pictures: newLRU[colorKey, *ui.Picture](maxCachedGlyphs)}
}

func (e *Engine) fontError(err error) {
	if e.opts.OnFontError != nil {
		e.opts.OnFontError(err)
	}
}

// styleFonts finds the faces for one combination of families, weight, slant, language
// and variations. It implements shaping.Fontmap.
type styleFonts struct {
	e          *Engine
	families   []string
	weight     int
	italic     bool
	lang       string
	variations []text.Variation

	// shaper shapes with the faces of this style. HarfBuzz keeps a font per parsed font and
	// reads the variation coordinates from the first face it sees, so faces with other
	// variations need a shaper of their own.
	shaper shaping.HarfbuzzShaper

	files  []text.FontFile
	loaded bool

	emoji      map[rune]*font.Face
	emojiFiles []text.FontFile
	faces      map[rune]*font.Face
	// ascii and asciiSet are faces for the runes below 128, without a map: most text is made of them.
	ascii    [128]*font.Face
	asciiSet [128]bool
	first    *font.Face // the first face that loads: the one for a rune no face has
	// firstTried is whether firstFace has looked for first.
	firstTried bool
}

// forgetText empties what the engine keeps of the text it has shaped: the shaped runs and the
// analyses of texts. The fonts it has resolved stay. The tests and the benchmarks use it to
// measure a build that finds nothing in the caches.
func (e *Engine) forgetText() {
	e.shapes.clear()
	e.analyses.clear()
}

// syncFonts forgets what the engine resolved or shaped when fonts were added to its collection: a
// style may find other faces now, and the runs that were shaped may have another face.
func (e *Engine) syncFonts() {
	if g := e.fonts.Generation(); g != e.generation {
		e.generation = g
		clear(e.byStyle)
		e.recent = [recentStyles]recentStyle{}
		e.shapes.clear()
	}
}

// recentStyles is how many styles fontsFor remembers by their values.
const recentStyles = 8

// recentStyle is the arguments of a call of fontsFor and what it found.
type recentStyle struct {
	families   []string
	weight     ui.FontWeight
	italic     bool
	lang       string
	variations []ui.FontVariation
	found      *styleFonts
}

func (r *recentStyle) matches(families []string, weight ui.FontWeight, italic bool, lang string, vars []ui.FontVariation) bool {
	return r.found != nil && r.weight == weight && r.italic == italic && r.lang == lang &&
		slices.Equal(r.families, families) && slices.Equal(r.variations, vars)
}

// fontsFor returns the font finder for a style. It is called for every span of every paragraph,
// so the lookup builds its key in a buffer that the Engine keeps, and allocates only for a style it has not seen.
func (e *Engine) fontsFor(families []string, weight ui.FontWeight, italic bool, locale ui.Locale, vars []ui.FontVariation) *styleFonts {
	w := int(weight)
	if w == 0 {
		w = int(ui.FontWeightW400)
	}
	e.syncFonts()
	lang := locale.LanguageTag()
	for i := range e.recent {
		if r := &e.recent[i]; r.matches(families, weight, italic, lang, vars) {
			return r.found
		}
	}
	f := e.lookupFonts(families, w, italic, lang, vars)
	e.recent[e.recentNext] = recentStyle{families: slices.Clone(families), weight: weight, italic: italic, lang: lang, variations: slices.Clone(vars), found: f}
	e.recentNext = (e.recentNext + 1) % recentStyles
	return f
}

// lookupFonts finds the font finder of a style in the engine's map, making it when it is new.
func (e *Engine) lookupFonts(families []string, w int, italic bool, lang string, vars []ui.FontVariation) *styleFonts {
	key := e.keyBuf[:0]
	for _, f := range families {
		key = strconv.AppendQuote(key, f)
		key = append(key, ',')
	}
	key = append(key, '|')
	key = strconv.AppendInt(key, int64(w), 10)
	if italic {
		key = append(key, "|i|"...)
	} else {
		key = append(key, "|u|"...)
	}
	key = append(key, lang...)
	for _, v := range vars {
		key = append(key, '|')
		key = append(key, v.Axis...)
		key = append(key, '=')
		key = strconv.AppendFloat(key, float64(v.Value), 'g', -1, 64)
	}
	e.keyBuf = key
	if f := e.byStyle[string(key)]; f != nil { // the conversion in an index expression does not allocate
		return f
	}
	tvars := []text.Variation{{Tag: "wght", Value: float32(w)}}
	if italic {
		tvars = append(tvars, text.Variation{Tag: "ital", Value: 1})
	}
	for _, v := range vars {
		tvars = append(tvars, text.Variation{Tag: v.Axis, Value: float32(v.Value)})
	}
	f := &styleFonts{e: e, families: families, weight: w, italic: italic, lang: lang, variations: tvars, faces: map[rune]*font.Face{}}
	e.byStyle[string(key)] = f
	return f
}

// candidates lists the font files for the style: those of each family in turn, then
// the system's default family.
func (f *styleFonts) candidates() []text.FontFile {
	if f.loaded {
		return f.files
	}
	f.loaded = true
	seen := map[text.FontFile]bool{}
	families := append(append([]string(nil), f.families...), defaultFamily)
	for _, family := range families {
		files, err := f.e.fonts.Match(f.e.ctx, family, f.weight, f.italic, f.lang)
		if err != nil {
			f.e.fontError(err)
			continue
		}
		for _, file := range files {
			key := text.FontFile{Path: file.Path, Index: file.Index & 0xffff}
			if !seen[key] {
				seen[key] = true
				f.files = append(f.files, key)
			}
		}
	}
	return f.files
}

// emojiFace returns the face of the "emoji" family (the color emoji font, as fontconfig
// prefers it) that has a glyph for r, or nil.
func (f *styleFonts) emojiFace(r rune) *font.Face {
	if f.emoji == nil {
		f.emoji = map[rune]*font.Face{}
		files, err := f.e.fonts.Match(f.e.ctx, "emoji", f.weight, f.italic, f.lang)
		if err != nil {
			f.e.fontError(err)
		}
		f.emojiFiles = files
	}
	if face, ok := f.emoji[r]; ok {
		return face
	}
	found := f.firstWith(f.emojiFiles, r)
	f.emoji[r] = found
	return found
}

// face loads the face of a candidate, reporting a failure and skipping it.
func (f *styleFonts) face(file text.FontFile) *font.Face {
	face, err := f.e.fonts.Face(f.e.ctx, file, f.variations...)
	if err != nil {
		f.e.fontError(err)
		return nil
	}
	return face
}

// firstWith returns the face of the first file that has a glyph for r, or nil. A file whose character map
// says it has none is not parsed (text.Collection.Covers): only a file that may have the glyph is
// loaded, and the face is then asked, since the map and the face can differ.
func (f *styleFonts) firstWith(files []text.FontFile, r rune) *font.Face {
	for _, file := range files {
		has, err := f.e.fonts.Covers(f.e.ctx, file, r)
		if err != nil {
			f.e.fontError(err)
			continue
		}
		if !has {
			continue
		}
		face := f.face(file)
		if face == nil {
			continue
		}
		if _, ok := face.NominalGlyph(r); ok {
			return face
		}
	}
	return nil
}

// firstFace returns the first face of the style that loads: the one that draws a character no face has
// (its missing-glyph box) and the one that stands for the characters that choose no font. It is nil when
// no file of the style loads. Only the files before the first that loads are parsed.
func (f *styleFonts) firstFace() *font.Face {
	if f.first != nil || f.firstTried {
		return f.first
	}
	f.firstTried = true
	for _, file := range f.candidates() {
		if face := f.face(file); face != nil {
			f.first = face
			break
		}
	}
	return f.first
}

// choosesNoFont says whether the character is one that has no glyph of its own, so that searching the
// fonts for it would only load them: a control character (general category Cc, the tab among them) or a
// default-ignorable code point (a zero-width space, a joiner, a variation selector, the byte order mark).
// HarfBuzz hides those from the glyphs, and go-text's face splitting does not let them change the font of a run
// (shaping.ignoreFaceChange).
func choosesNoFont(r rune) bool {
	return unicode.Is(unicode.Cc, r) || harfbuzz.IsDefaultIgnorable(r)
}

// ResolveFace returns the first face of the style that has a glyph for r; when none
// has, the first face that loads (which draws its missing-glyph box), or nil when
// there is no font at all. A control character or a default-ignorable code point resolves to
// that first face with no search. A face is parsed only when it may have the glyph: the fonts that
// cannot show r are ruled out by their character maps first, so a style of many fonts costs only the
// fonts that can help.
func (f *styleFonts) ResolveFace(r rune) *font.Face {
	if uint32(r) < 128 && f.asciiSet[r] {
		return f.ascii[r]
	}
	if face, ok := f.faces[r]; ok {
		return face
	}
	var found *font.Face
	switch {
	case choosesNoFont(r):
		found = f.firstFace()
	default:
		if text.IsEmoji(r) {
			found = f.emojiFace(r)
		}
		if found == nil {
			found = f.firstWith(f.candidates(), r)
		}
		if found == nil {
			// None of the families has the character: the system's fallback for it.
			for _, family := range append(append([]string(nil), f.families...), defaultFamily) {
				file, ok, err := f.e.fonts.Fallback(f.e.ctx, family, f.weight, f.italic, f.lang, r)
				if err != nil {
					f.e.fontError(err)
					continue
				}
				if !ok {
					continue
				}
				if face := f.face(file); face != nil {
					found = face
					break
				}
			}
		}
		if found == nil {
			found = f.firstFace()
		}
	}
	f.faces[r] = found
	if uint32(r) < 128 {
		f.ascii[r], f.asciiSet[r] = found, true
	}
	return found
}
