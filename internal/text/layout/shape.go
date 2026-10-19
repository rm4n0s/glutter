package layout

import (
	"unicode"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/math/fixed"
)

// glyph is one shaped glyph, in logical pixels.
type glyph struct {
	id font.GID
	// cluster is the index of the first rune of the cluster the glyph belongs to.
	cluster int
	// runes is the number of runes in the cluster.
	runes int
	// advance is the advance of this glyph.
	advance float64
	// xOffset and yOffset move the glyph from its pen position (y grows upward).
	xOffset, yOffset float64
}

// run is a stretch of text with one style, one font, one direction and one script,
// shaped.
type run struct {
	// start and end are the rune range, end exclusive.
	start, end int
	rtl        bool
	// level is the embedding level for reordering.
	level int
	// style is the index of the style in the paragraph's styles.
	style int
	face  *font.Face
	size  float64
	// glyphs are in visual order: for a right-to-left run the last rune comes first.
	glyphs []glyph

	// The font's metrics at the size, positive: ascent above the baseline, descent
	// below it and the gap between lines.
	ascent, descent, gap float64
	// The corrected ascent and descent of the run with the line height of the style
	// applied (and half the gap added to each side), and without the height.
	ascentC, descentC float64
	ascentU           float64
}

// isLineBreakRune reports whether the rune ends a line when it is in the text.
func isLineBreakRune(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// isHangingSpace reports whether the rune is white space that stays at the end of a
// line without counting for its width (not the no-break spaces).
func isHangingSpace(r rune) bool {
	switch r {
	case 0xA0, 0x2007, 0x202F:
		return false
	}
	return unicode.IsSpace(r)
}

// shapeText is the text the shaper sees: the characters that end lines are replaced
// by a zero width space, so that they take no room and draw nothing.
func shapeText(runes []rune) []rune {
	out := runes
	for i, r := range runes {
		if isLineBreakRune(r) {
			if &out[0] == &runes[0] {
				out = append([]rune(nil), runes...)
			}
			out[i] = 0x200B
		}
	}
	return out
}

// sentinelFace stands in for a font in the first segmentation, which only splits by
// direction and script.
var sentinelFace = &font.Face{}

type sentinelFontmap struct{}

func (sentinelFontmap) ResolveFace(rune) *font.Face { return sentinelFace }

// correctedMetrics computes the ascent and descent of a line with the font's
// metrics and a style's height, as skparagraph's Run::calculateMetrics does: half of
// the font's gap goes on each side, then the height is applied, proportionally or
// evenly.
func correctedMetrics(ascent, descent, gap, size, height float64, hasHeight bool, even bool) (a, d float64) {
	a, d = ascent+gap/2, descent+gap/2
	if !hasHeight || height == 0 {
		return a, d
	}
	runHeight := height * size
	intrinsic := a + d
	if intrinsic <= 0 {
		return a, d
	}
	if even {
		extra := runHeight - intrinsic
		return a + extra/2, d + extra/2
	}
	m := runHeight / intrinsic
	return a * m, d * m
}

// fontMetrics returns the ascent, descent and gap of the face at the size.
func fontMetrics(face *font.Face, size float64) (ascent, descent, gap float64) {
	if face == nil {
		return 0.93 * size, 0.24 * size, 0
	}
	ext, ok := face.FontHExtents()
	if !ok || (ext.Ascender == 0 && ext.Descender == 0) {
		return 0.93 * size, 0.24 * size, 0
	}
	upem := float64(face.Upem())
	return float64(ext.Ascender) / upem * size, -float64(ext.Descender) / upem * size, float64(ext.LineGap) / upem * size
}

// shapeRuns splits the text into runs and shapes them.
func (p *paragraph) shapeRuns() {
	n := len(p.runes)
	if n == 0 {
		return
	}
	dir := di.DirectionLTR
	if p.style.Direction() == ui.TextDirectionRTL {
		dir = di.DirectionRTL
	}
	lang := language.NewLanguage(p.baseLocale.LanguageTag())
	whole := shaping.Input{Text: p.shapeRunes, RunStart: 0, RunEnd: n, Direction: dir, Size: shapeSize * 64, Language: lang}
	var seg shaping.Segmenter
	pieces := append([]shaping.Input(nil), seg.Split(whole, sentinelFontmap{})...)

	for _, piece := range pieces {
		// Split the piece at the style boundaries, then by face.
		start := piece.RunStart
		for start < piece.RunEnd {
			si := p.styleAt[start]
			end := start
			for end < piece.RunEnd && p.styleAt[end] == si {
				end++
			}
			sub := piece
			sub.RunStart, sub.RunEnd = start, end
			sub.Face = nil
			st := &p.styles[si]
			if st.Locale.LanguageCode != "" {
				sub.Language = language.NewLanguage(st.Locale.LanguageTag())
			}
			fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
			for _, in := range shaping.SplitByFace(sub, fm) {
				p.runs = append(p.runs, p.shapeRun(in, si))
			}
			start = end
		}
	}
}

// shapeRun shapes one input and converts the output to logical pixels.
func (p *paragraph) shapeRun(in shaping.Input, si int) *run {
	st := &p.styles[si]
	r := &run{start: in.RunStart, end: in.RunEnd, rtl: in.Direction.Progression() == di.TowardTopLeft, style: si, face: in.Face, size: st.FontSize}
	r.level = p.levelOf(r.rtl)
	r.ascent, r.descent, r.gap = fontMetrics(in.Face, r.size)
	even := st.LeadingDistribution == ui.TextLeadingDistributionEven
	r.ascentC, r.descentC = correctedMetrics(r.ascent, r.descent, r.gap, r.size, st.Height, st.HasHeight, even)
	r.ascentU = r.ascent + r.gap/2
	if in.Face == nil || st.FontSize <= 0 {
		r.glyphs = p.missingGlyphs(r)
		return r
	}
	in.Size = fixed.Int26_6(shapeSize * 64)
	for _, f := range st.FontFeatures {
		in.FontFeatures = append(in.FontFeatures, shaping.FontFeature{Tag: tagOf(f.Feature), Value: uint32(f.Value)})
	}
	fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
	out := fm.shaper.Shape(in)
	scale := r.size / shapeSize / 64
	r.glyphs = make([]glyph, len(out.Glyphs))
	for i, g := range out.Glyphs {
		r.glyphs[i] = glyph{
			id:      g.GlyphID,
			cluster: g.TextIndex(),
			runes:   g.RunesCount(),
			advance: float64(g.Advance) * scale,
			xOffset: float64(g.XOffset) * scale,
			yOffset: float64(g.YOffset) * scale,
		}
	}
	return r
}

// missingGlyphs gives a run with no font the glyphs of nothing: one cluster per
// rune with no advance, so that the text keeps its positions.
func (p *paragraph) missingGlyphs(r *run) []glyph {
	gs := make([]glyph, 0, r.end-r.start)
	for i := r.start; i < r.end; i++ {
		gs = append(gs, glyph{id: 0, cluster: i, runes: 1})
	}
	if r.rtl {
		for i, j := 0, len(gs)-1; i < j; i, j = i+1, j-1 {
			gs[i], gs[j] = gs[j], gs[i]
		}
	}
	return gs
}

// levelOf is the embedding level of a run from its direction: the levels 0 and 1 of a
// left-to-right paragraph, 1 and 2 of a right-to-left one.
func (p *paragraph) levelOf(rtl bool) int {
	switch {
	case p.style.Direction() != ui.TextDirectionRTL && !rtl:
		return 0
	case p.style.Direction() != ui.TextDirectionRTL && rtl:
		return 1
	case rtl:
		return 1
	}
	return 2
}
