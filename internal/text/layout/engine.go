package layout

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-text/typesetting/font"
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
}

var _ ui.ParagraphFactory = (*Engine)(nil)

// NewEngine makes a text engine over a font collection. ctx bounds the loading of
// font files, which happens while a paragraph is built. More than one options value
// panics with MultipleOptions.
//
// Flutter: no counterpart
func NewEngine(ctx context.Context, fonts *text.Collection, opts ...EngineOpts) *Engine {
	return &Engine{ctx: ctx, fonts: fonts, opts: oneOpts(opts), byStyle: map[string]*styleFonts{}}
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
	faces  map[rune]*font.Face
	first  *font.Face // the first face that loads: the one for a rune no face has
}

// fontsFor returns the font finder for a style.
func (e *Engine) fontsFor(families []string, weight ui.FontWeight, italic bool, locale ui.Locale, vars []ui.FontVariation) *styleFonts {
	w := int(weight)
	if w == 0 {
		w = int(ui.FontWeightW400)
	}
	lang := locale.LanguageTag()
	var key strings.Builder
	fmt.Fprintf(&key, "%q|%d|%v|%s", families, w, italic, lang)
	tvars := []text.Variation{{Tag: "wght", Value: float32(w)}}
	if italic {
		tvars = append(tvars, text.Variation{Tag: "ital", Value: 1})
	}
	for _, v := range vars {
		tvars = append(tvars, text.Variation{Tag: v.Axis, Value: float32(v.Value)})
		fmt.Fprintf(&key, "|%s=%v", v.Axis, v.Value)
	}
	if f := e.byStyle[key.String()]; f != nil {
		return f
	}
	f := &styleFonts{e: e, families: families, weight: w, italic: italic, lang: lang, variations: tvars, faces: map[rune]*font.Face{}}
	e.byStyle[key.String()] = f
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

// face loads the face of a candidate, reporting a failure and skipping it.
func (f *styleFonts) face(file text.FontFile) *font.Face {
	face, err := f.e.fonts.Face(f.e.ctx, file, f.variations...)
	if err != nil {
		f.e.fontError(err)
		return nil
	}
	return face
}

// ResolveFace returns the first face of the style that has a glyph for r; when none
// has, the first face that loads (which draws its missing-glyph box), or nil when
// there is no font at all.
func (f *styleFonts) ResolveFace(r rune) *font.Face {
	if face, ok := f.faces[r]; ok {
		return face
	}
	var found *font.Face
	for _, file := range f.candidates() {
		face := f.face(file)
		if face == nil {
			continue
		}
		if f.first == nil {
			f.first = face
		}
		if _, ok := face.NominalGlyph(r); ok {
			found = face
			break
		}
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
		found = f.first
	}
	f.faces[r] = found
	return found
}
