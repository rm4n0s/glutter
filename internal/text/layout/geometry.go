package layout

import (
	"math"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// runeBox is where one rune is on its line, in paragraph coordinates.
type runeBox struct {
	x0, x1 float64
	rtl    bool
	// run is the placed run, by index in the line's runs; -1 if the rune has no glyph.
	run int
}

// geometry returns the boxes of the runes of a line, computing them on the first call
// after a layout.
func (p *paragraph) geometry(l *line) []runeBox {
	if l.geo != nil {
		return l.geo
	}
	geo := make([]runeBox, l.end-l.start)
	for i := range geo {
		geo[i].run = -1
	}
	for ri := range l.runs {
		pr := &l.runs[ri]
		if pr.ellipsis {
			continue
		}
		base := l.left + pr.x
		for i := 0; i < len(pr.glyphs); {
			g0 := pr.glyphs[i]
			x0, x1 := g0.pen, g0.pen+g0.width
			j := i + 1
			for j < len(pr.glyphs) && pr.glyphs[j].cluster == g0.cluster {
				x1 = pr.glyphs[j].pen + pr.glyphs[j].width
				j++
			}
			count := max(1, p.clusterRunes(pr.run, g0.cluster))
			count = min(count, l.end-g0.cluster)
			w := (x1 - x0) / float64(count)
			for k := 0; k < count; k++ {
				idx := g0.cluster + k - l.start
				if idx < 0 || idx >= len(geo) {
					continue
				}
				b := runeBox{rtl: pr.run.rtl, run: ri}
				if pr.run.rtl {
					b.x1 = base + x1 - float64(k)*w
					b.x0 = b.x1 - w
				} else {
					b.x0 = base + x0 + float64(k)*w
					b.x1 = b.x0 + w
				}
				geo[idx] = b
			}
			i = j
		}
	}
	l.geo = geo
	return geo
}

// clusterRunes is the number of runes in the cluster of the run that starts at rune c.
func (p *paragraph) clusterRunes(r *run, c int) int {
	for _, g := range r.glyphs {
		if g.cluster == c {
			return g.runes
		}
	}
	return 1
}

// graphemeBox is the extent of a grapheme cluster on a line.
type graphemeBox struct {
	start, end int // runes
	x0, x1     float64
	rtl        bool
}

// graphemesOf returns the grapheme clusters of a line, left to right. The line break at the
// end of a line is not one of them.
func (p *paragraph) graphemesOf(l *line) []graphemeBox {
	geo := p.geometry(l)
	end := l.end
	for end > l.start && isLineBreakRune(p.runes[end-1]) {
		end--
	}
	var out []graphemeBox
	gi := sort.SearchInts(p.graphemes, l.start)
	for ; gi < len(p.graphemes)-1 && p.graphemes[gi] < end; gi++ {
		s, e := p.graphemes[gi], min(p.graphemes[gi+1], end)
		if s < l.start {
			continue
		}
		g := graphemeBox{start: s, end: e, x0: math.Inf(1), x1: math.Inf(-1)}
		for r := s; r < e; r++ {
			b := geo[r-l.start]
			if b.run < 0 {
				continue
			}
			g.x0, g.x1 = min(g.x0, b.x0), max(g.x1, b.x1)
			g.rtl = b.rtl
		}
		if math.IsInf(g.x0, 1) {
			continue
		}
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].x0 < out[j].x0 })
	return out
}

// lineAt returns the index of the line that holds the y coordinate: the first line above
// the paragraph, the last one below it.
func (p *paragraph) lineAt(y float64) int {
	for i, l := range p.lines {
		if y < l.top+l.height {
			return i
		}
	}
	return len(p.lines) - 1
}

// runeIndex returns the index of the rune that holds the byte offset, rounding a byte in
// the middle of a rune down, and a negative offset to 0 and one past the end to the end.
func (p *paragraph) runeIndex(off int) int {
	if off <= 0 {
		return 0
	}
	if off >= len(p.content.Text) {
		return len(p.runes)
	}
	i := sort.SearchInts(p.byteOffset, off)
	if i < len(p.byteOffset) && p.byteOffset[i] == off {
		return i
	}
	return i - 1
}

// visibleEnd is the rune after the last visible line.
func (p *paragraph) visibleEnd() int {
	if len(p.lines) == 0 {
		return 0
	}
	return p.lines[len(p.lines)-1].end
}

// GetPositionForOffset implements ui.ParagraphLayout.
func (p *paragraph) GetPositionForOffset(o ui.Offset) ui.TextPosition {
	if len(p.lines) == 0 || len(p.runes) == 0 {
		return ui.NewTextPosition(0)
	}
	li := p.lineAt(o.Dy)
	l := p.lines[li]
	gs := p.graphemesOf(l)
	if len(gs) == 0 {
		return p.positionAt(li, l.start)
	}
	first, last := gs[0], gs[len(gs)-1]
	var rune int
	switch {
	case o.Dx <= first.x0:
		rune = first.start
		if first.rtl {
			rune = first.end
		}
	case o.Dx >= last.x1:
		rune = last.end
		if last.rtl {
			rune = last.start
		}
	default:
		best, bestDist := gs[0], math.Inf(1)
		for _, g := range gs {
			d := 0.0
			if o.Dx < g.x0 {
				d = g.x0 - o.Dx
			} else if o.Dx > g.x1 {
				d = o.Dx - g.x1
			}
			if d < bestDist {
				best, bestDist = g, d
			}
		}
		left := o.Dx < (best.x0+best.x1)/2
		switch {
		case left != best.rtl:
			rune = best.start
		default:
			rune = best.end
		}
	}
	return p.positionAt(li, rune)
}

// positionAt makes the position of a rune index on a line: at the end of a line that is not
// the last it is upstream, as the end of that line.
func (p *paragraph) positionAt(li, rune int) ui.TextPosition {
	pos := ui.NewTextPosition(p.byteOf(rune))
	l := p.lines[li]
	if li < len(p.lines)-1 {
		contentEnd := l.end
		for l.hardBreak && contentEnd > l.start && isLineBreakRune(p.runes[contentEnd-1]) {
			contentEnd--
		}
		// A hard break ends the line before its line break, which also is the end
		// of the line (unless the line is empty).
		if (rune == l.end && !l.hardBreak) || (l.hardBreak && rune == contentEnd && rune > l.start) {
			pos.Affinity = ui.TextAffinityUpstream
		}
	}
	return pos
}

func (p *paragraph) glyphInfoFor(li int, g graphemeBox) ui.GlyphInfo {
	l := p.lines[li]
	dir := ui.TextDirectionLTR
	if g.rtl {
		dir = ui.TextDirectionRTL
	}
	return ui.GlyphInfo{
		GraphemeClusterLayoutBounds: ui.Rect{Left: g.x0, Top: l.top, Right: g.x1, Bottom: l.top + l.height},
		GraphemeClusterRange:        ui.TextRange{Start: p.byteOf(g.start), End: p.byteOf(g.end)},
		WritingDirection:            dir,
	}
}

// GetClosestGlyphInfoForOffset implements ui.ParagraphLayout.
func (p *paragraph) GetClosestGlyphInfoForOffset(o ui.Offset) (ui.GlyphInfo, bool) {
	if len(p.lines) == 0 || len(p.runes) == 0 {
		return ui.GlyphInfo{}, false
	}
	li := p.lineAt(o.Dy)
	gs := p.graphemesOf(p.lines[li])
	if len(gs) == 0 {
		return ui.GlyphInfo{}, false
	}
	best, bestDist := gs[0], math.Inf(1)
	for _, g := range gs {
		d := 0.0
		if o.Dx < g.x0 {
			d = g.x0 - o.Dx
		} else if o.Dx > g.x1 {
			d = o.Dx - g.x1
		}
		if d < bestDist {
			best, bestDist = g, d
		}
	}
	return p.glyphInfoFor(li, best), true
}

// GetGlyphInfoAt implements ui.ParagraphLayout.
func (p *paragraph) GetGlyphInfoAt(off int) (ui.GlyphInfo, bool) {
	li, ok := p.GetLineNumberAt(off)
	if !ok {
		return ui.GlyphInfo{}, false
	}
	r := p.runeIndex(off)
	l := p.lines[li]
	for _, g := range p.graphemesOf(l) {
		if r >= g.start && r < g.end {
			return p.glyphInfoFor(li, g), true
		}
	}
	if l.hardBreak && isLineBreakRune(p.runes[r]) {
		// The line break is a glyph without width at the end of its line.
		gi := sort.SearchInts(p.graphemes, r+1) - 1
		x, rtl := p.lineEnd(l)
		return p.glyphInfoFor(li, graphemeBox{start: p.graphemes[gi], end: p.graphemes[gi+1], x0: x, x1: x, rtl: rtl}), true
	}
	return ui.GlyphInfo{}, false
}

// GetLineNumberAt implements ui.ParagraphLayout.
func (p *paragraph) GetLineNumberAt(off int) (int, bool) {
	if off < 0 || off >= len(p.content.Text) || len(p.lines) == 0 {
		return 0, false
	}
	r := p.runeIndex(off)
	if r >= p.visibleEnd() {
		return 0, false
	}
	for i, l := range p.lines {
		if r >= l.start && r < l.end {
			return i, true
		}
	}
	return 0, false
}

// NumberOfLines implements ui.ParagraphLayout.
func (p *paragraph) NumberOfLines() int { return len(p.lines) }

// GetLineMetricsAt implements ui.ParagraphLayout.
func (p *paragraph) GetLineMetricsAt(n int) (ui.LineMetrics, bool) {
	if n < 0 || n >= len(p.lines) {
		return ui.LineMetrics{}, false
	}
	return p.ComputeLineMetrics()[n], true
}

// GetLineBoundary implements ui.ParagraphLayout.
func (p *paragraph) GetLineBoundary(pos ui.TextPosition) ui.TextRange {
	if len(p.lines) == 0 {
		return ui.TextRange{}
	}
	r := p.runeIndex(pos.Offset)
	li := len(p.lines) - 1
	for i, l := range p.lines {
		if r < l.end || (r == l.end && i == len(p.lines)-1) {
			li = i
			break
		}
	}
	l := p.lines[li]
	// At the boundary between two lines, the affinity says which one.
	if li > 0 && r == l.start && pos.Affinity.IsUpstream() {
		li--
		l = p.lines[li]
	}
	// The newline that ends a line is not part of its range.
	end := l.end
	if l.hardBreak && end > l.start && p.runes[end-1] == '\n' {
		end--
	}
	return ui.TextRange{Start: p.byteOf(l.start), End: p.byteOf(end)}
}

// GetWordBoundary implements ui.ParagraphLayout.
func (p *paragraph) GetWordBoundary(pos ui.TextPosition) ui.TextRange {
	n := len(p.runes)
	if n == 0 {
		return ui.TextRange{}
	}
	r := p.runeIndex(pos.Offset)
	if pos.Affinity.IsUpstream() {
		r--
	}
	if r < 0 {
		return ui.TextRange{}
	}
	if r >= n {
		return ui.TextRange{Start: len(p.content.Text), End: len(p.content.Text)}
	}
	for _, w := range p.wordBreaks() {
		if r >= w.start && r < w.end {
			if w.isWord {
				return ui.TextRange{Start: p.byteOf(w.start), End: p.byteOf(w.end)}
			}
			break
		}
	}
	// Anything that is not a word is a word break on both sides: one grapheme cluster.
	gi := sort.SearchInts(p.graphemes, r+1) - 1
	return ui.TextRange{Start: p.byteOf(p.graphemes[gi]), End: p.byteOf(p.graphemes[gi+1])}
}
