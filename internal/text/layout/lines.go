package layout

import (
	"math"
	"slices"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/rm4n0s/glutter/ui"
)

// placedGlyph is a glyph with its position relative to the start of its run.
type placedGlyph struct {
	id font.GID
	// cluster is the index of the first rune of the glyph's cluster (the position of the
	// ellipsis for the glyphs of an ellipsis).
	cluster int
	// advance is the advance of the glyph and xOffset its offset from the pen position,
	// before justification.
	advance, xOffset float64
	// x is the position of the glyph from the start of its run, and y the offset of the
	// glyph below the baseline.
	x, y float64
	// pen is where the glyph's advance starts, from the start of its run, and width the
	// advance with the justification added.
	pen, width float64
}

// placedRun is a run, or the part of one that is on a line, with its glyphs placed.
type placedRun struct {
	run *run
	// start and end are the runes of the line that the run holds.
	start, end int
	glyphs     []placedGlyph
	// x is the position of the start of the run on the line, width its advance.
	x, width float64
	// ellipsis marks the run that holds the ellipsis of a truncated line.
	ellipsis bool
}

// line is a laid-out line.
type line struct {
	// start and end are the runes of the line; visEnd is end without the white space
	// that hangs at the end.
	start, end, visEnd int
	hardBreak          bool
	// runs are in visual order.
	runs []placedRun

	// width is the width without the white space at the end, widthWithSpaces with it.
	width, widthWithSpaces float64
	// left is the x of the start of the line after alignment.
	left float64

	ascent, descent, ascentU float64
	height                   float64
	top, baseline            float64
	// justify is the extra advance added to each space of a justified line.
	justify float64

	// geo holds the boxes of the runes of the line, computed on demand.
	geo []runeBox
}

// lineRange is a line before its glyphs are placed.
type lineRange struct {
	start, end, visEnd int
	hard               bool
}

// Layout implements ui.ParagraphLayout.
func (p *paragraph) Layout(c ui.ParagraphConstraints) {
	width := c.Width
	if math.IsNaN(width) || width < 0 {
		width = 0
	}
	maxLines, hasMax := p.style.MaxLines()
	// With an ellipsis and no limit on the lines, the first line that does not fit is not
	// wrapped: it is ellipsized and the text after it dropped (TextPainter.ellipsis).
	keepWords := p.style.Ellipsis() != "" && !hasMax
	ranges, overflow := p.breakLines(width, keepWords)
	p.exceededLines = overflow
	if hasMax && len(ranges) > maxLines {
		p.exceededLines = true
		ranges = ranges[:maxLines]
	}

	p.lines = p.lines[:0]
	for i, lr := range ranges {
		l := p.buildLine(lr)
		if i == len(ranges)-1 && p.exceededLines && p.style.Ellipsis() != "" {
			p.ellipsize(l, width)
		}
		p.lines = append(p.lines, l)
	}

	p.longestLine = 0
	for _, l := range p.lines {
		p.longestLine = max(p.longestLine, l.width)
	}
	p.width = width
	if math.IsInf(width, 1) {
		p.width = p.longestLine
	}
	p.verticalMetrics()
	p.align()
	p.laidOut = true
}

// breakLines breaks the text into lines of at most width, at the break opportunities of
// UAX #14, and inside a word that is wider than the width (at cluster boundaries).
func (p *paragraph) breakLines(width float64, keepWords bool) (_ []lineRange, overflow bool) {
	n := len(p.runes)
	const eps = 1e-6
	var out []lineRange
	lineStart := 0
	used := 0.0 // the advance of the line so far, with the white space of its last segment
	emit := func(end int, hard bool) {
		visEnd := end
		for visEnd > lineStart && isHangingSpace(p.runes[visEnd-1]) {
			visEnd--
		}
		out = append(out, lineRange{lineStart, end, visEnd, hard})
		lineStart = end
		used = 0
	}
	lastHard := false
	for _, s := range p.segments {
		start := s.start
		for {
			full := p.span(start, s.end, used)
			vis := p.span(start, s.visEnd, used)
			if used+vis <= width+eps {
				used += full
				break
			}
			if keepWords {
				// The line holds the words up to this one and ends the text; the
				// ellipsis cuts the line to the width afterwards.
				emit(s.end, false)
				return out, true
			}
			if lineStart < start {
				// The segment does not fit after the words before it: it starts a line.
				emit(start, false)
				continue
			}
			// The word alone is wider than the line: put as much of it as fits.
			cut := start
			w := 0.0
			for k := start + 1; k <= s.visEnd; k++ {
				if !p.boundary[k] {
					continue
				}
				w = p.span(start, k, used)
				if w > width+eps && cut > start {
					break
				}
				cut = k
			}
			if cut >= s.visEnd {
				// Only the white space at the end is left over, which hangs.
				used += full
				break
			}
			if cut == start {
				cut = p.nextBoundary(start)
			}
			emit(cut, false)
			start = cut
		}
		lastHard = s.mandatory
		if s.mandatory {
			emit(s.end, true)
		}
	}
	if lineStart < n || len(out) == 0 || lastHard {
		emit(n, false)
	}
	return out, false
}

// nextBoundary is the first cluster boundary after i.
func (p *paragraph) nextBoundary(i int) int {
	for k := i + 1; k < len(p.boundary); k++ {
		if p.boundary[k] {
			return k
		}
	}
	return len(p.runes)
}

// buildLine puts the runs of a line range in visual order and places their glyphs.
func (p *paragraph) buildLine(lr lineRange) *line {
	l := &line{start: lr.start, end: lr.end, visEnd: lr.visEnd, hardBreak: lr.hard}
	tabs := p.lineTabs(lr.start, lr.end)
	for _, r := range p.runs {
		lo, hi := max(lr.start, r.start), min(lr.end, r.end)
		if lo >= hi {
			continue
		}
		l.runs = append(l.runs, p.cut(r, lo, hi, lr.start, tabs))
	}
	l.width = p.span(lr.start, lr.visEnd, 0)
	l.widthWithSpaces = p.span(lr.start, lr.end, 0)
	return l
}

// cut makes the part of a run that holds the runes [lo, hi). tabs, when it is not nil, has the advance
// of the tabs of the line that starts at the rune lineStart (see lineTabs); a tab glyph takes it.
func (p *paragraph) cut(r *run, lo, hi, lineStart int, tabs []float64) placedRun {
	pr := placedRun{run: r, start: lo, end: hi}
	for _, g := range r.glyphs {
		if g.cluster < lo || g.cluster >= hi {
			continue
		}
		advance := g.advance
		if tabs != nil && r.ph == nil && p.runes[g.cluster] == '\t' {
			advance = tabs[g.cluster-lineStart]
		}
		pr.glyphs = append(pr.glyphs, placedGlyph{id: g.id, cluster: g.cluster, advance: advance, xOffset: g.xOffset, y: -g.yOffset})
	}
	pr.width = 0
	for _, g := range pr.glyphs {
		pr.width += g.advance
	}
	return pr
}

// ellipsize shortens the last line so that the ellipsis fits after its text.
func (p *paragraph) ellipsize(l *line, width float64) {
	ellipsis := []rune(p.style.Ellipsis())
	si := 0
	if len(p.runes) > 0 {
		si = p.styleAt[min(max(l.visEnd-1, l.start), len(p.runes)-1)]
	}
	er := p.shapeEllipsis(ellipsis, si)
	ew := 0.0
	for _, g := range er.glyphs {
		ew += g.advance
	}
	// The text that stays is the longest start of the line that fits with the ellipsis
	// (all of it when it does).
	cut := l.visEnd
	for cut > l.start && (!p.boundary[cut] || p.span(l.start, cut, 0)+ew > width+1e-6) {
		cut--
	}
	*l = *p.buildLine(lineRange{start: l.start, end: cut, visEnd: cut, hard: false})
	pr := p.cut(er, 0, len(ellipsis), 0, nil)
	pr.ellipsis = true
	for i := range pr.glyphs {
		pr.glyphs[i].cluster = cut
	}
	l.runs = append(l.runs, pr)
	l.width += ew
	l.widthWithSpaces = l.width
}

// shapeEllipsis shapes the ellipsis text in a style.
func (p *paragraph) shapeEllipsis(text []rune, si int) *run {
	st := p.styles[si]
	dir := di.DirectionLTR
	if p.style.Direction() == ui.TextDirectionRTL {
		dir = di.DirectionRTL
	}
	fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
	in := shaping.Input{Text: text, RunStart: 0, RunEnd: len(text), Direction: dir, Face: fm.ResolveFace(text[0]),
		Script: language.LookupScript(text[0]), Language: language.NewLanguage(st.Locale.LanguageTag())}
	r := p.shapeRun(in, si, fm)
	r.rtl = dir == di.DirectionRTL
	// The ellipsis is at the paragraph's level, after the text.
	r.level = 0
	if r.rtl {
		r.level = 1
	}
	return r
}

// visualOrder reorders the runs of a line from logical to visual order (rule L2 of
// UAX #9: from the highest level down to the lowest odd one, reverse every sequence
// of runs at that level or higher; the level of a right-to-left paragraph, 1, counts).
func visualOrder(runs []placedRun, rtlParagraph bool) {
	if len(runs) < 2 {
		return
	}
	highest, lowestOdd := 0, math.MaxInt
	if rtlParagraph {
		// The paragraph itself is at level 1.
		lowestOdd = 1
	}
	for _, r := range runs {
		highest = max(highest, r.run.level)
		if r.run.level%2 == 1 {
			lowestOdd = min(lowestOdd, r.run.level)
		}
	}
	for level := highest; level >= lowestOdd && level > 0; level-- {
		for i := 0; i < len(runs); {
			if runs[i].run.level < level {
				i++
				continue
			}
			j := i
			for j < len(runs) && runs[j].run.level >= level {
				j++
			}
			slices.Reverse(runs[i:j])
			i = j
		}
	}
}

// verticalMetrics gives each line its ascent, descent, height and position from the
// fonts of its runs, the style heights and the strut.
func (p *paragraph) verticalMetrics() {
	hb := p.style.HeightBehavior()
	strut := p.strutMetrics()
	top := 0.0
	for i, l := range p.lines {
		first, last := i == 0, i == len(p.lines)-1
		var ascent, descent, ascentU, descentU float64
		runs := 0
		for _, pr := range l.runs {
			r := pr.run
			a, d, au, du := r.ascentC, r.descentC, r.ascentU, r.descent+r.gap/2
			ascent, descent = max(ascent, a), max(descent, d)
			ascentU, descentU = max(ascentU, au), max(descentU, du)
			runs++
		}
		if runs == 0 {
			a, d, au, du := p.emptyLineMetrics(l)
			ascent, descent, ascentU, descentU = a, d, au, du
		}
		if strut.enabled {
			if strut.force {
				ascent, descent, ascentU, descentU = strut.ascent, strut.descent, strut.ascentU, strut.descentU
			} else {
				ascent, descent = max(ascent, strut.ascent), max(descent, strut.descent)
				ascentU, descentU = max(ascentU, strut.ascentU), max(descentU, strut.descentU)
			}
		}
		if first && !hb.ApplyHeightToFirstAscent {
			ascent = ascentU
		}
		if last && !hb.ApplyHeightToLastDescent {
			descent = descentU
		}
		// Placeholders aligned to the top, bottom or middle of the line make it taller when
		// they are taller than the text.
		for _, pr := range l.runs {
			if pr.run.ph == nil || !pr.run.ph.floating {
				continue
			}
			h := pr.run.ph.height
			switch pr.run.ph.spec.Alignment {
			case ui.PlaceholderAlignmentTop:
				descent = max(descent, h-ascent)
			case ui.PlaceholderAlignmentBottom:
				ascent = max(ascent, h-descent)
			default:
				if extra := h - (ascent + descent); extra > 0 {
					ascent, descent = ascent+extra/2, descent+extra/2
				}
			}
		}
		l.ascent, l.descent, l.ascentU = ascent, descent, ascentU
		l.height = math.Round(ascent + descent)
		l.top = top
		l.baseline = top + ascent
		top += l.height
	}
	p.height = top
}

// emptyLineMetrics are the metrics of a line with no runs (an empty paragraph, or the
// empty line after a final newline), from the style of the text before it.
func (p *paragraph) emptyLineMetrics(l *line) (ascent, descent, ascentU, descentU float64) {
	si := 0
	if len(p.runes) > 0 {
		si = p.styleAt[min(max(l.start-1, 0), len(p.runes)-1)]
	}
	st := p.styles[si]
	fm := p.eng.fontsFor(st.FontFamilies, st.FontWeight, st.FontStyle == ui.FontStyleItalic, st.Locale, st.FontVariations)
	a, d, gap := p.eng.fontMetrics(fm.ResolveFace(' '), st.FontSize)
	even := st.LeadingDistribution == ui.TextLeadingDistributionEven
	ascent, descent = correctedMetrics(a, d, gap, st.FontSize, st.Height, st.HasHeight, even)
	return ascent, descent, a + gap/2, d + gap/2
}

// strutMetricsT is the strut resolved against its font.
type strutMetricsT struct {
	enabled, force                     bool
	ascent, descent, ascentU, descentU float64
}

// strutMetrics computes the strut of the paragraph style.
func (p *paragraph) strutMetrics() strutMetricsT {
	s := p.strut
	if !s.Enabled {
		return strutMetricsT{}
	}
	families := s.FontFamilies
	if len(families) == 0 {
		families = p.base.FontFamilies
	}
	fm := p.eng.fontsFor(families, s.FontWeight, s.FontStyle == ui.FontStyleItalic, p.baseLocale, nil)
	a, d, gap := p.eng.fontMetrics(fm.ResolveFace(' '), s.FontSize)
	if s.Leading >= 0 {
		gap = s.Leading * s.FontSize
	}
	even := s.LeadingDistribution == ui.TextLeadingDistributionEven
	asc, desc := correctedMetrics(a, d, gap, s.FontSize, s.Height, s.HasHeight, even)
	return strutMetricsT{enabled: true, force: s.Force, ascent: asc, descent: desc, ascentU: a + gap/2, descentU: d + gap/2}
}

// align places the lines horizontally: it gives the glyphs their positions, justifies
// and sets the left edge of each line.
func (p *paragraph) align() {
	avail := p.width
	rtl := p.style.Direction() == ui.TextDirectionRTL
	align := p.style.Align()
	switch align {
	case ui.TextAlignStart:
		align = ui.TextAlignLeft
		if rtl {
			align = ui.TextAlignRight
		}
	case ui.TextAlignEnd:
		align = ui.TextAlignRight
		if rtl {
			align = ui.TextAlignLeft
		}
	}
	for i, l := range p.lines {
		visualOrder(l.runs, rtl)
		l.justify = 0
		lineAlign := align
		if align == ui.TextAlignJustify {
			lineAlign = ui.TextAlignLeft
			if rtl {
				lineAlign = ui.TextAlignRight
			}
			last := i == len(p.lines)-1
			if spaces := p.countSpaces(l); !last && !l.hardBreak && spaces > 0 && avail > l.width && !math.IsInf(avail, 1) {
				l.justify = (avail - l.width) / float64(spaces)
				l.width = avail
			}
		}
		switch lineAlign {
		case ui.TextAlignRight:
			l.left = avail - l.width
		case ui.TextAlignCenter:
			l.left = (avail - l.width) / 2
		default:
			l.left = 0
		}
		// The white space at the end hangs: after the text, or before it in a right-to-left
		// paragraph.
		trailing := l.widthWithSpaces - p.span(l.start, l.visEnd, 0)
		if l.justify != 0 {
			trailing = 0
		}
		x := 0.0
		if rtl && trailing > 0 && !p.hasEllipsis(l) {
			x = -trailing
		}
		for k := range l.runs {
			pr := &l.runs[k]
			pr.x = x
			x += p.placeRun(pr, l.justify, l.visEnd)
		}
		if l.justify == 0 {
			l.widthWithSpaces = l.width + trailing
		}
	}
}

func (p *paragraph) hasEllipsis(l *line) bool {
	return len(l.runs) > 0 && slices.ContainsFunc(l.runs, func(r placedRun) bool { return r.ellipsis })
}

// countSpaces counts the spaces of a line that justification widens.
func (p *paragraph) countSpaces(l *line) int {
	n := 0
	for i := l.start; i < l.visEnd; i++ {
		if isJustifySpace(p.runes[i]) {
			n++
		}
	}
	return n
}

// isJustifySpace says whether justification widens the rune.
func isJustifySpace(r rune) bool { return r == ' ' || r == 0xA0 || r == '\t' || r == 0x3000 }

// placeRun positions the glyphs of a placed run from its own start, adding the
// justification to each space before visEnd, and returns the advance of the run.
func (p *paragraph) placeRun(pr *placedRun, justify float64, visEnd int) float64 {
	pen := 0.0
	for i := range pr.glyphs {
		g := &pr.glyphs[i]
		g.pen = pen
		g.x = pen + g.xOffset
		g.width = g.advance
		if !pr.ellipsis && justify != 0 && g.cluster < visEnd && isJustifySpace(p.runes[g.cluster]) {
			g.width += justify
		}
		pen += g.width
	}
	pr.width = pen
	return pen
}

// ensureLaidOut lays the paragraph out with an infinite width when nothing has laid it out yet,
// so that a paragraph that is asked before it is laid out answers as one that has been.
func (p *paragraph) ensureLaidOut() {
	if !p.laidOut {
		p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	}
}
