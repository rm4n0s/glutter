package layout

import (
	"math"

	"github.com/go-text/typesetting/font"
	"github.com/rm4n0s/glutter/ui"
)

// DecorationSegment is one line of a text decoration: an underline, an overline or a line
// through a stretch of text of one style on one line, ready to draw. The positions are in
// paragraph coordinates.
//
// Flutter: engine/src/flutter/third_party/skia/modules/skparagraph/src/Decorations.cpp Decorations
type DecorationSegment struct {
	// Line is the line the segment is on.
	Line int
	// Kind is one of TextDecorationUnderline, TextDecorationOverline and TextDecorationLineThrough.
	Kind ui.TextDecoration
	// Style is how the line is drawn: solid, double, dotted, dashed or wavy.
	Style ui.TextDecorationStyle
	// Color is the color of the line.
	Color ui.Color
	// X0 and X1 are the horizontal extent of the line.
	X0, X1 float64
	// Y is the vertical center of the line (of the first line, for a double decoration).
	Y float64
	// Thickness is the thickness of the line.
	Thickness float64
	// FontSize is the size of the font of the text, for the shape of dotted, dashed and wavy lines.
	FontSize float64
}

// DoubleDecorationSpacing is the distance between the two lines of a double decoration.
//
// Flutter: engine/src/flutter/third_party/skia/modules/skparagraph/src/Decorations.cpp kDoubleDecorationSpacing
const DoubleDecorationSpacing = 3.0

// decorationMetrics gives the position of the three decoration lines relative to the
// baseline (positive is below) and the thickness of each, in logical pixels, for a face.
func decorationMetrics(face *font.Face, size, ascent float64) (under, strike, over, underThick, strikeThick float64) {
	upem := 1.0
	if face != nil {
		upem = float64(face.Upem())
	}
	scale := size / upem
	underThick, strikeThick = size/14, size/14
	under = size * 0.1
	strike = -size * 0.3
	if face != nil {
		if t := float64(face.LineMetric(font.UnderlineThickness)) * scale; t > 0 {
			underThick = t
		}
		if t := float64(face.LineMetric(font.StrikethroughThickness)) * scale; t > 0 {
			strikeThick = t
		}
		if pos := float64(face.LineMetric(font.UnderlinePosition)) * scale; pos != 0 {
			// The top of the underline is pos above the baseline.
			under = -pos + underThick/2
		}
		if pos := float64(face.LineMetric(font.StrikethroughPosition)) * scale; pos != 0 {
			strike = -pos + strikeThick/2
		}
	}
	// The overline sits at the top of the glyphs.
	over = -ascent + underThick/2
	return under, strike, over, underThick, strikeThick
}

// Decorations returns the decoration lines of the laid-out text, in line order. White space
// that hangs at the end of a line is not decorated. Neighbouring pieces of text with the same
// decoration form one segment.
//
// Flutter: engine/src/flutter/third_party/skia/modules/skparagraph/src/TextLine.cpp TextLine::paintDecorations
func (p *paragraph) Decorations() []DecorationSegment {
	var out []DecorationSegment
	for li, l := range p.lines {
		geo := p.geometry(l)
		first := len(out)
		for ri := range l.runs {
			pr := &l.runs[ri]
			if pr.run.ph != nil {
				continue
			}
			st := &p.styles[pr.run.style]
			if st.Decoration == ui.TextDecorationNone {
				continue
			}
			x0, x1 := math.Inf(1), math.Inf(-1)
			if pr.ellipsis {
				x0, x1 = l.left+pr.x, l.left+pr.x+pr.width
			}
			for r := pr.start; r < min(pr.end, l.visEnd) && !pr.ellipsis; r++ {
				if b := geo[r-l.start]; b.run == ri {
					x0, x1 = math.Min(x0, b.x0), math.Max(x1, b.x1)
				}
			}
			if x1 <= x0 {
				continue
			}
			color := st.Color
			if st.HasDecorationColor {
				color = st.DecorationColor
			}
			under, strike, over, underThick, strikeThick := decorationMetrics(pr.run.face, pr.run.size, pr.run.ascent)
			mult := st.DecorationThickness
			style := st.DecorationStyle
			for _, k := range []struct {
				kind  ui.TextDecoration
				y, th float64
			}{
				{ui.TextDecorationUnderline, under, underThick},
				{ui.TextDecorationOverline, over, underThick},
				{ui.TextDecorationLineThrough, strike, strikeThick},
			} {
				if st.Decoration&k.kind == 0 {
					continue
				}
				out = appendSegment(out, first, DecorationSegment{Line: li, Kind: k.kind, Style: style, Color: color, X0: x0, X1: x1,
					Y: l.baseline + k.y, Thickness: k.th * mult, FontSize: pr.run.size})
			}
		}
	}
	return out
}

// appendSegment adds a segment, joining it to a touching segment of the same look on the
// same line (from index first on).
func appendSegment(out []DecorationSegment, first int, s DecorationSegment) []DecorationSegment {
	for i := first; i < len(out); i++ {
		o := &out[i]
		if o.Kind == s.Kind && o.Style == s.Style && o.Color == s.Color && o.Y == s.Y && o.Thickness == s.Thickness &&
			o.FontSize == s.FontSize && (math.Abs(o.X1-s.X0) < 1e-3 || math.Abs(s.X1-o.X0) < 1e-3) {
			o.X0, o.X1 = math.Min(o.X0, s.X0), math.Max(o.X1, s.X1)
			return out
		}
	}
	return append(out, s)
}
