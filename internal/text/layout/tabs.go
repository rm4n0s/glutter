package layout

import (
	"math"
	"slices"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/math/fixed"
)

// tabEpsilon is how close to a tab stop a position is when it counts as being on the stop, so
// that the sums of floating point advances of a whole number of tab widths do not give a tab
// of almost no width.
const tabEpsilon = 1e-6

// setupTabs finds the width of a tab stop when the paragraph style has a tab size and the text
// has a tab. It is zero otherwise, and then no function of this file changes anything.
func (p *paragraph) setupTabs() {
	p.tabW = 0
	size := p.style.TabSize()
	if size <= 0 || !slices.Contains(p.runes, '\t') {
		return
	}
	if w := p.spaceAdvance(); w > 0 {
		p.tabW = size * w
	}
}

// spaceAdvance is the advance of U+0020 in the font of the paragraph style, at its size: the
// unit of the tab size. It is zero when the style has no font or no size.
func (p *paragraph) spaceAdvance() float64 {
	st := &p.base
	if st.FontSize <= 0 {
		return 0
	}
	fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
	face := fm.ResolveFace(' ')
	if face == nil {
		return 0
	}
	in := shaping.Input{Text: []rune{' '}, RunStart: 0, RunEnd: 1, Direction: di.DirectionLTR, Face: face, Size: fixed.Int26_6(shapeSize * 64),
		Script: language.Latin, Language: language.NewLanguage(st.Locale.LanguageTag())}
	out := fm.shaper.Shape(in)
	total := 0.0
	for _, g := range out.Glyphs {
		total += float64(g.Advance)
	}
	return total * st.FontSize / shapeSize / 64
}

// isTab reports whether the rune is a tab that has a stop.
func (p *paragraph) isTab(i int) bool { return p.tabW > 0 && p.runes[i] == '\t' }

// tabAdvance is the distance from the position x, measured from the start of the line, to the next
// tab stop; a whole tab width when x is on a stop.
func (p *paragraph) tabAdvance(x float64) float64 {
	r := math.Mod(x, p.tabW)
	if r < tabEpsilon || p.tabW-r < tabEpsilon {
		r = 0
	}
	return p.tabW - r
}

// tabSpacing is the letter spacing, and the word spacing, that the tab's style adds after it.
func (p *paragraph) tabSpacing(i int) float64 {
	st := &p.styles[p.styleAt[i]]
	return st.LetterSpacing + st.WordSpacing
}

// advAt is the advance of the rune i when it is x from the start of the line.
func (p *paragraph) advAt(i int, x float64) float64 {
	if p.isTab(i) {
		return p.tabAdvance(x) + p.tabSpacing(i)
	}
	return p.adv[i]
}

// span is the advance of the runes [a, b) when a is x0 from the start of the line.
func (p *paragraph) span(a, b int, x0 float64) float64 {
	if p.tabW == 0 {
		return p.sum(a, b)
	}
	x := x0
	for i := a; i < b; i++ {
		x += p.advAt(i, x)
	}
	return x - x0
}

// lineTabs gives the advance of each tab of the line [start, end), indexed by the rune minus start,
// or nil when the line has no tab with a stop.
func (p *paragraph) lineTabs(start, end int) []float64 {
	if p.tabW == 0 {
		return nil
	}
	var out []float64
	x := 0.0
	for i := start; i < end; i++ {
		a := p.advAt(i, x)
		if p.isTab(i) {
			if out == nil {
				out = make([]float64, end-start)
			}
			out[i-start] = a
		}
		x += a
	}
	return out
}

// drawsNothing says whether the glyph is a tab, which leaves the room it advances empty.
func (p *paragraph) drawsNothing(pr *placedRun, g *placedGlyph) bool {
	return p.tabW > 0 && !pr.ellipsis && g.cluster >= 0 && g.cluster < len(p.runes) && p.runes[g.cluster] == '\t'
}
