package layout

import (
	"math"
	"slices"
	"unicode"

	"github.com/rm4n0s/glutter/ui"
)

// wordSeg is a word segment of the text (UAX #29).
type wordSeg struct {
	start, end int
	// isWord is false for white space, punctuation and symbols.
	isWord bool
}

// wordBreaks segments the text into words, on the first call.
func (p *paragraph) wordBreaks() []wordSeg {
	if p.words != nil {
		return p.words
	}
	p.words = []wordSeg{}
	it := p.seg.WordIterator()
	for it.Next() {
		w := it.Word()
		seg := wordSeg{start: w.Offset, end: w.Offset + len(w.Text)}
		for _, r := range w.Text {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
				seg.isWord = true
				break
			}
		}
		p.words = append(p.words, seg)
	}
	return p.words
}

// tightExtent is the vertical extent of the glyphs of a line: the font's ascent and descent
// of its runs, without the line height.
func (p *paragraph) tightExtent(l *line) (top, bottom float64) {
	asc, desc := 0.0, 0.0
	for _, pr := range l.runs {
		if pr.run.ph != nil {
			continue
		}
		asc, desc = math.Max(asc, pr.run.ascent), math.Max(desc, pr.run.descent)
	}
	if asc == 0 && desc == 0 {
		asc, desc = l.ascentU, l.height-l.ascentU
	}
	return l.baseline - asc, l.baseline + desc
}

// placeholderRect is the vertical extent of the box of a placeholder on its line.
func (p *paragraph) placeholderRect(l *line, ph *placeholder, r *run) (top, bottom float64) {
	if !ph.floating {
		top = l.baseline - r.ascentC
		return top, top + ph.height
	}
	switch ph.spec.Alignment {
	case ui.PlaceholderAlignmentTop:
		top = l.baseline - l.ascent
	case ui.PlaceholderAlignmentBottom:
		top = l.baseline + l.descent - ph.height
	default:
		top = l.baseline - l.ascent + (l.ascent+l.descent-ph.height)/2
	}
	return top, top + ph.height
}

// boxVertical is the top and bottom of the box of a run on a line in a height style.
func (p *paragraph) boxVertical(li int, pr *placedRun, hs ui.BoxHeightStyle) (top, bottom float64) {
	l := p.lines[li]
	tightTop, tightBottom := p.tightExtent(l)
	if pr.run.ph != nil {
		top, bottom = p.placeholderRect(l, pr.run.ph, pr.run)
		if hs == ui.BoxHeightStyleUnspecified || hs == ui.BoxHeightStyleTight {
			return top, bottom
		}
		defer func() { top, bottom = math.Min(top, tightTop), math.Max(bottom, tightBottom) }()
	} else {
		top, bottom = l.baseline-pr.run.ascent, l.baseline+pr.run.descent
	}
	switch hs {
	case ui.BoxHeightStyleMax:
		return l.baseline - l.ascent, l.baseline + l.descent
	case ui.BoxHeightStyleStrut:
		if st := p.strutMetrics(); st.enabled {
			return l.baseline - st.ascent, l.baseline + st.descent
		}
		return top, bottom
	case ui.BoxHeightStyleIncludeLineSpacingMiddle, ui.BoxHeightStyleIncludeLineSpacingTop, ui.BoxHeightStyleIncludeLineSpacingBottom:
		// The space between this line's glyphs and the next line's goes to the box of
		// one of them; none above the first line or below the last.
		above, below := 0.0, 0.0
		if li > 0 {
			_, prevBottom := p.tightExtent(p.lines[li-1])
			above = math.Max(tightTop-prevBottom, 0)
		}
		if li < len(p.lines)-1 {
			nextTop, _ := p.tightExtent(p.lines[li+1])
			below = math.Max(nextTop-tightBottom, 0)
		}
		switch hs {
		case ui.BoxHeightStyleIncludeLineSpacingMiddle:
			return top - above/2, bottom + below/2
		case ui.BoxHeightStyleIncludeLineSpacingTop:
			return top - above, bottom
		default:
			return top, bottom + below
		}
	}
	return top, bottom
}

// onlyLineBreaks reports whether the runes [lo, hi) are all line breaks.
func (p *paragraph) onlyLineBreaks(lo, hi int) bool {
	for r := lo; r < hi; r++ {
		if !isLineBreakRune(p.runes[r]) {
			return false
		}
	}
	return true
}

// lineEnd is the x where the text of the line ends, and whether it is right to left.
func (p *paragraph) lineEnd(l *line) (x float64, rtl bool) {
	gs := p.graphemesOf(l)
	if len(gs) == 0 {
		return l.left, p.style.Direction() == ui.TextDirectionRTL
	}
	// The graphemes are in visual order: the end of the line is on the side of the
	// last one in the logical order of the paragraph direction.
	if p.style.Direction() == ui.TextDirectionRTL {
		return gs[0].x0, true
	}
	return gs[len(gs)-1].x1, false
}

// lineBoxVertical is the top and bottom of a box that has the height of the line.
func (p *paragraph) lineBoxVertical(li int, hs ui.BoxHeightStyle) (top, bottom float64) {
	l := p.lines[li]
	if hs == ui.BoxHeightStyleStrut {
		if st := p.strutMetrics(); st.enabled {
			return l.baseline - st.ascent, l.baseline + st.descent
		}
	}
	return l.baseline - l.ascent, l.baseline + l.descent
}

func dirOf(rtl bool) ui.TextDirection {
	if rtl {
		return ui.TextDirectionRTL
	}
	return ui.TextDirectionLTR
}

// GetBoxesForRange implements ui.ParagraphLayout.
func (p *paragraph) GetBoxesForRange(start, end int, hs ui.BoxHeightStyle, ws ui.BoxWidthStyle) []ui.TextBox {
	if start < 0 || start >= end || len(p.lines) == 0 {
		return nil
	}
	a, b := p.runeIndex(start), p.runeIndex(end)
	if b < len(p.runes) && p.byteOffset[b] < end {
		b++ // an end inside a rune takes the whole rune
	}
	b = min(b, p.visibleEnd())
	if a >= b {
		return nil
	}
	type lineBoxes struct {
		line  int
		boxes []ui.TextBox
	}
	var perLine []lineBoxes
	for li, l := range p.lines {
		lo, hi := max(a, l.start), min(b, l.end)
		if lo >= hi {
			continue
		}
		geo := p.geometry(l)
		var raw []ui.TextBox
		for r := lo; r < hi; r++ {
			g := geo[r-l.start]
			if g.run < 0 || isLineBreakRune(p.runes[r]) {
				continue
			}
			top, bottom := p.boxVertical(li, &l.runs[g.run], hs)
			raw = append(raw, ui.TextBox{Left: g.x0, Top: top, Right: g.x1, Bottom: bottom, Direction: dirOf(g.rtl)})
		}
		slices.SortStableFunc(raw, func(x, y ui.TextBox) int {
			switch {
			case x.Left < y.Left:
				return -1
			case x.Left > y.Left:
				return 1
			}
			return 0
		})
		// Neighbouring boxes of the same direction and height are one box.
		var merged []ui.TextBox
		for _, bx := range raw {
			if n := len(merged); n > 0 {
				m := &merged[n-1]
				if m.Direction == bx.Direction && m.Top == bx.Top && m.Bottom == bx.Bottom && math.Abs(m.Right-bx.Left) < 1e-3 {
					m.Right = bx.Right
					continue
				}
			}
			merged = append(merged, bx)
		}
		if len(merged) == 0 && l.hardBreak && lo == a && hi == b && hi == l.end && p.onlyLineBreaks(lo, hi) {
			// The line break alone has a box without width at the end of its line,
			// where the caret goes.
			x, rtl := p.lineEnd(l)
			top, bottom := p.lineBoxVertical(li, hs)
			merged = append(merged, ui.TextBox{Left: x, Top: top, Right: x, Bottom: bottom, Direction: dirOf(rtl)})
		}
		if len(merged) > 0 {
			perLine = append(perLine, lineBoxes{li, merged})
		}
	}
	if ws == ui.BoxWidthStyleMax {
		minLeft, maxRight := math.Inf(1), math.Inf(-1)
		for _, l := range p.lines {
			minLeft = math.Min(minLeft, l.left)
			maxRight = math.Max(maxRight, l.left+l.width)
		}
		paraDir := p.style.Direction()
		for i := range perLine {
			bs := perLine[i].boxes
			first, last := bs[0], bs[len(bs)-1]
			l := p.lines[perLine[i].line]
			// A box is extended only where the selection goes on past the edge of
			// the line: at the start of the line when the selection holds its first
			// character, and at the end when it holds the line break or goes on to
			// the next line. A word in the middle of a line, or the last line of the
			// text, keeps tight boxes.
			startsLine := max(a, l.start) == l.start
			endsLine := min(b, l.end) == l.end && (l.hardBreak || perLine[i].line < len(p.lines)-1)
			var out []ui.TextBox
			if startsLine && first.Left > minLeft+1e-3 {
				out = append(out, ui.TextBox{Left: minLeft, Top: first.Top, Right: first.Left, Bottom: first.Bottom, Direction: paraDir})
			}
			out = append(out, bs...)
			if endsLine && last.Right < maxRight-1e-3 {
				out = append(out, ui.TextBox{Left: last.Right, Top: last.Top, Right: maxRight, Bottom: last.Bottom, Direction: paraDir})
			}
			perLine[i].boxes = out
		}
	}
	var all []ui.TextBox
	for _, lb := range perLine {
		all = append(all, lb.boxes...)
	}
	return all
}

// GetBoxesForPlaceholders implements ui.ParagraphLayout. A placeholder that is not on a
// visible line (it was cut by the maximum number of lines) has no box.
func (p *paragraph) GetBoxesForPlaceholders() []ui.TextBox {
	var out []ui.TextBox
	for _, ph := range p.placeholders {
		if ph == nil {
			continue
		}
		for _, l := range p.lines {
			if ph.rune < l.start || ph.rune >= l.end {
				continue
			}
			geo := p.geometry(l)
			g := geo[ph.rune-l.start]
			if g.run < 0 {
				break
			}
			top, bottom := p.placeholderRect(l, ph, l.runs[g.run].run)
			out = append(out, ui.TextBox{Left: g.x0, Top: top, Right: g.x1, Bottom: bottom, Direction: dirOf(g.rtl)})
			break
		}
	}
	return out
}
