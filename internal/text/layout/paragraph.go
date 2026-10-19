package layout

import (
	"math"

	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/segmenter"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

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

// tagOf makes the OpenType tag of a four-letter feature name.
func tagOf(name string) ot.Tag { return ot.MustNewTag(name) }

// segment is a stretch of the text between two line break opportunities.
type segment struct {
	start, end int
	// visEnd is end without the white space at the end.
	visEnd int
	// mandatory is true when a line must end after the segment.
	mandatory bool
}

// paragraph is the text engine's ui.ParagraphLayout.
type paragraph struct {
	eng   *Engine
	style *ui.ParagraphStyle
	// baseLocale is the locale of the paragraph style, for the segmentation.
	baseLocale ui.Locale
	base       ui.ResolvedTextStyle
	strut      ui.ResolvedStrut

	runes      []rune
	shapeRunes []rune
	// byteOffset[i] is the byte offset of rune i; it has len(runes)+1 entries.
	byteOffset []int
	styles     []ui.ResolvedTextStyle
	// styleAt is the index of the style of each rune.
	styleAt []int

	runs []*run
	// adv is the advance of each rune and boundary says whether a cluster starts at the
	// rune (len(runes)+1 entries: the last is true).
	adv      []float64
	boundary []bool
	segments []segment

	minIntrinsic, maxIntrinsic float64

	// Results of the last layout.
	width         float64 // the width the paragraph was laid out with
	longestLine   float64
	height        float64
	lines         []*line
	exceededLines bool
	laidOut       bool
}

var _ ui.ParagraphLayout = (*paragraph)(nil)

// BuildParagraph shapes the content (ui.ParagraphFactory). The result is laid out
// at once with an infinite width, so that the intrinsic widths are known; Layout
// lays it out for a width.
func (e *Engine) BuildParagraph(c *ui.ParagraphContent) ui.ParagraphLayout {
	p := &paragraph{eng: e, style: c.Style, base: c.Style.BaseTextStyle(), strut: c.Style.Strut()}
	if c.Style != nil && c.Style.Options().Locale != nil {
		p.baseLocale = *c.Style.Options().Locale
	}
	p.byteOffset = make([]int, 0, len(c.Text)+1)
	for i, r := range c.Text {
		p.byteOffset = append(p.byteOffset, i)
		p.runes = append(p.runes, r)
	}
	p.byteOffset = append(p.byteOffset, len(c.Text))
	p.shapeRunes = shapeText(p.runes)
	p.assignStyles(c)
	p.shapeRuns()
	p.measure()
	p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	return p
}

// assignStyles maps the byte ranges of the content's runs to the rune indices.
func (p *paragraph) assignStyles(c *ui.ParagraphContent) {
	p.styleAt = make([]int, len(p.runes))
	if len(c.Runs) == 0 {
		p.styles = []ui.ResolvedTextStyle{p.base}
		return
	}
	ri := 0
	for i := range p.runes {
		for ri < len(c.Runs)-1 && p.byteOffset[i] >= c.Runs[ri].End {
			ri++
		}
		p.styleAt[i] = ri
	}
	p.styles = make([]ui.ResolvedTextStyle, len(c.Runs))
	for i, r := range c.Runs {
		p.styles[i] = r.Style
	}
}

// measure computes the advance of each rune, the cluster boundaries, the break
// segments and the intrinsic widths: everything that does not depend on the width.
func (p *paragraph) measure() {
	n := len(p.runes)
	p.adv = make([]float64, n)
	p.boundary = make([]bool, n+1)
	p.boundary[n] = true
	for _, r := range p.runs {
		for i := 0; i < len(r.glyphs); {
			g := r.glyphs[i]
			total, j := 0.0, i
			for j < len(r.glyphs) && r.glyphs[j].cluster == g.cluster {
				total += r.glyphs[j].advance
				j++
			}
			count := max(g.runes, 1)
			if g.cluster+count > r.end {
				count = r.end - g.cluster
			}
			for k := g.cluster; k < g.cluster+count; k++ {
				p.adv[k] = total / float64(count)
			}
			p.boundary[g.cluster] = true
			i = j
		}
		// A run starts and ends at a cluster boundary.
		p.boundary[r.start] = true
	}

	var seg segmenter.Segmenter
	seg.Init(p.runes)
	it := seg.LineIterator()
	for it.Next() {
		l := it.Line()
		s := segment{start: l.Offset, end: l.Offset + len(l.Text)}
		// UAX #14 also ends the text with a mandatory break (LB3); only a line break
		// character ends a line.
		s.mandatory = l.IsMandatoryBreak && s.end > s.start && isLineBreakRune(p.runes[s.end-1])
		s.visEnd = s.end
		for s.visEnd > s.start && isHangingSpace(p.runes[s.visEnd-1]) {
			s.visEnd--
		}
		p.segments = append(p.segments, s)
	}

	// The intrinsic widths: the widest unbreakable word, and the widest line when only
	// the mandatory breaks end lines.
	p.minIntrinsic, p.maxIntrinsic = 0, 0
	lineWidth := 0.0
	for _, s := range p.segments {
		p.minIntrinsic = max(p.minIntrinsic, p.sum(s.start, s.visEnd))
		lineWidth += p.sum(s.start, s.end)
		if s.mandatory {
			p.maxIntrinsic = max(p.maxIntrinsic, lineWidth)
			lineWidth = 0
		}
	}
	p.maxIntrinsic = max(p.maxIntrinsic, lineWidth)
}

// sum is the advance of the runes [a, b).
func (p *paragraph) sum(a, b int) float64 {
	w := 0.0
	for i := a; i < b; i++ {
		w += p.adv[i]
	}
	return w
}

// byteOf is the UTF-8 byte offset of rune i.
func (p *paragraph) byteOf(i int) int { return p.byteOffset[i] }

// Width implements ui.ParagraphLayout.
func (p *paragraph) Width() float64 { return p.width }

// Height implements ui.ParagraphLayout.
func (p *paragraph) Height() float64 { return p.height }

// LongestLine implements ui.ParagraphLayout.
func (p *paragraph) LongestLine() float64 { return p.longestLine }

// MinIntrinsicWidth implements ui.ParagraphLayout.
func (p *paragraph) MinIntrinsicWidth() float64 { return p.minIntrinsic }

// MaxIntrinsicWidth implements ui.ParagraphLayout.
func (p *paragraph) MaxIntrinsicWidth() float64 { return p.maxIntrinsic }

// AlphabeticBaseline implements ui.ParagraphLayout: the baseline of the first line.
func (p *paragraph) AlphabeticBaseline() float64 {
	if len(p.lines) == 0 {
		return 0
	}
	return p.lines[0].baseline
}

// IdeographicBaseline implements ui.ParagraphLayout: the bottom of the first line.
func (p *paragraph) IdeographicBaseline() float64 {
	if len(p.lines) == 0 {
		return 0
	}
	return p.lines[0].baseline + p.lines[0].descent
}

// DidExceedMaxLines implements ui.ParagraphLayout.
func (p *paragraph) DidExceedMaxLines() bool { return p.exceededLines }

// ComputeLineMetrics implements ui.ParagraphLayout.
func (p *paragraph) ComputeLineMetrics() []ui.LineMetrics {
	out := make([]ui.LineMetrics, len(p.lines))
	for i, l := range p.lines {
		out[i] = ui.LineMetrics{
			HardBreak:      l.hardBreak,
			Ascent:         l.ascent,
			Descent:        l.descent,
			UnscaledAscent: l.ascentU,
			Height:         l.height,
			Width:          l.width,
			Left:           l.left,
			Baseline:       l.baseline,
			LineNumber:     i,
		}
	}
	return out
}
