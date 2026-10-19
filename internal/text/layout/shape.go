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

	// ph is set for the run of a placeholder, which has a box instead of glyphs.
	ph *placeholder
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

// shapeRuns splits the text into runs and shapes them. The text is split by the levels of the
// bidi algorithm, then by script (go-text), then by style, then by the font that has the glyphs.
func (p *paragraph) shapeRuns() {
	n := len(p.runes)
	if n == 0 {
		return
	}
	baseRTL := p.style.Direction() == ui.TextDirectionRTL
	p.levels = resolveLevels(p.runes, baseRTL)
	lang := language.NewLanguage(p.baseLocale.LanguageTag())
	var seg shaping.Segmenter

	// One piece per run of equal level, split by script.
	for lo := 0; lo < n; {
		hi := lo
		for hi < n && p.levels[hi] == p.levels[lo] {
			hi++
		}
		dir := di.DirectionLTR
		if p.levels[lo]%2 == 1 {
			dir = di.DirectionRTL
		}
		whole := shaping.Input{Text: p.shapeRunes, RunStart: lo, RunEnd: hi, Direction: dir, Size: shapeSize * 64, Language: lang}
		for _, piece := range seg.Split(whole, sentinelFontmap{}) {
			// The segmenter resolves bidi too, with the first strong character; the level
			// run is one direction already.
			piece.Direction = dir
			p.splitPiece(piece)
		}
		lo = hi
	}
}

// splitPiece splits a piece of one level and script by style, placeholders and font, and
// shapes the parts.
func (p *paragraph) splitPiece(piece shaping.Input) {
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
		// A placeholder is a run of its own; the text between placeholders is shaped.
		from := start
		for k := start; k <= end; k++ {
			ph, isPh := p.phAt[k]
			if k < end && !isPh {
				continue
			}
			if k > from {
				chunk := sub
				chunk.RunStart, chunk.RunEnd = from, k
				for _, in := range shaping.SplitByFace(chunk, fm) {
					p.runs = append(p.runs, p.shapeRun(in, si))
				}
			}
			if k < end {
				p.runs = append(p.runs, p.placeholderRun(k, si, ph, sub))
				from = k + 1
			}
		}
		start = end
	}
}

// shapeRun shapes one input and converts the output to logical pixels.
func (p *paragraph) shapeRun(in shaping.Input, si int) *run {
	st := &p.styles[si]
	r := &run{start: in.RunStart, end: in.RunEnd, rtl: in.Direction.Progression() == di.TowardTopLeft, style: si, face: in.Face, size: st.FontSize}
	if in.RunStart < len(p.levels) {
		r.level = int(p.levels[in.RunStart])
	}
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
	applySpacing(r, in.Text, st.LetterSpacing, st.WordSpacing)
	return r
}

// applySpacing adds the letter spacing after every cluster and the word spacing after every
// space (CSS and Flutter add them after the glyph; in a right-to-left run the glyph moves
// to the end of the added room, so that the room is on its left).
func applySpacing(r *run, text []rune, letter, word float64) {
	if letter == 0 && word == 0 {
		return
	}
	for i := 0; i < len(r.glyphs); {
		c := r.glyphs[i].cluster
		extra := letter
		if c >= 0 && c < len(text) && isWordSpace(text[c]) {
			extra += word
		}
		if extra != 0 {
			r.glyphs[i].advance += extra
			if r.rtl {
				r.glyphs[i].xOffset += extra
			}
		}
		for i < len(r.glyphs) && r.glyphs[i].cluster == c {
			i++
		}
	}
}

// isWordSpace says whether word spacing applies to the rune: the white space between words.
func isWordSpace(r rune) bool { return !isLineBreakRune(r) && unicode.IsSpace(r) }

// placeholder is a box in the text for an inline widget.
type placeholder struct {
	spec ui.PlaceholderSpec
	// index is the order in which it was added, and rune its place in the text.
	index, rune int
	// width, height and baselineOffset are scaled.
	width, height, baselineOffset float64
	// floating is true for the alignments that depend on the line: top, bottom and middle.
	floating bool
}

// placeholderRun makes the run of a placeholder: one glyph of the size of the box.
func (p *paragraph) placeholderRun(at, si, index int, piece shaping.Input) *run {
	spec := p.content.Placeholders[index]
	ph := &placeholder{spec: spec, index: index, rune: at,
		width: spec.Width * spec.Scale, height: spec.Height * spec.Scale, baselineOffset: spec.BaselineOffset * spec.Scale}
	p.placeholders[index] = ph
	st := &p.styles[si]
	r := &run{start: at, end: at + 1, style: si, size: st.FontSize, ph: ph}
	r.rtl = piece.Direction.Progression() == di.TowardTopLeft
	r.level = int(p.levels[at])
	r.glyphs = []glyph{{cluster: at, runes: 1, advance: ph.width}}
	_, descent, gap := fontMetrics(p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations).ResolveFace(' '), st.FontSize)
	switch spec.Alignment {
	case ui.PlaceholderAlignmentAboveBaseline:
		r.ascentC, r.descentC = ph.height, 0
	case ui.PlaceholderAlignmentBelowBaseline:
		r.ascentC, r.descentC = 0, ph.height
	case ui.PlaceholderAlignmentTop, ui.PlaceholderAlignmentBottom, ui.PlaceholderAlignmentMiddle:
		ph.floating = true
	default:
		offset := ph.baselineOffset
		if spec.Baseline == ui.TextBaselineIdeographic {
			// The ideographic baseline of the text is below the alphabetic one.
			offset -= descent + gap/2
		}
		r.ascentC, r.descentC = offset, ph.height-offset
	}
	r.ascentU = r.ascentC
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
