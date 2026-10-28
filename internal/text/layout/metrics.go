package layout

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

var _ ui.FontMetricsEngine = (*Engine)(nil)

// FontMetrics measures the face that the style draws the digit "0" with, the way a paragraph of one
// line in the style does (ui.FontMetricsEngine): the ascent, descent and height are those of the
// line of a paragraph of "0" (the style's height and the paragraph defaults applied), the leading is
// the font's line gap at the size, and the advance is the width of the "0". It is for the UI thread.
//
// Flutter: no counterpart
func (e *Engine) FontMetrics(style *ui.ResolvedTextStyle) ui.FontMetrics {
	content := &ui.ParagraphContent{
		Style: ui.NewParagraphStyle(),
		Text:  "0",
		Runs:  []ui.TextRun{{Start: 0, End: 1, Style: style}},
	}
	p := e.BuildParagraph(content)
	p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	out := ui.FontMetrics{ZeroAdvance: p.MaxIntrinsicWidth()}
	if lines := p.ComputeLineMetrics(); len(lines) > 0 {
		out.Ascent, out.Descent, out.Height = lines[0].Ascent, lines[0].Descent, lines[0].Height
	}
	fm := e.fontsFor(style.FontFamilies, style.FontWeight, style.FontStyle == ui.FontStyleItalic, style.Locale, style.FontVariations)
	face := fm.ResolveFace('0')
	_, _, out.Leading = e.fontMetrics(face, style.FontSize)
	if face != nil {
		out.Monospaced = face.IsMonospace()
	}
	return out
}
