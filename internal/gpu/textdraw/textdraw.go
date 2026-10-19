// Package textdraw turns the drawing of a paragraph into ordinary drawing operations, for
// the renderers: each glyph is a path, drawn with its origin on a quarter of a pixel
// horizontally and on a pixel vertically, so that the coverage of a glyph at a size can be
// cached for the four subpixel positions (SPEC §9.1).
//
// Flutter: no counterpart
package textdraw

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// subpixels is how many horizontal positions a pixel is divided into for glyphs.
const subpixels = 4

// Expand calls emit with each operation that draws the paragraph at offset, under the
// transform ctm, with the transform to draw it under. It returns false, emitting nothing,
// if the paragraph cannot draw itself.
func Expand(op ui.OpDrawParagraph, ctm *ui.Matrix4, emit func(ui.DisplayOp, *ui.Matrix4)) bool {
	if op.Paragraph == nil {
		return true
	}
	painter, ok := op.Paragraph.Layouter().(ui.ParagraphPainter)
	if !ok {
		return false
	}
	axisAligned := isAxisAligned(ctm)
	painter.PaintParagraph(func(d ui.ParagraphDraw) {
		m := ctm.Clone()
		m.Translate(float32(op.Offset.Dx+d.Offset.Dx), float32(op.Offset.Dy+d.Offset.Dy), 0)
		if d.Matrix != nil {
			m.MultiplyBy(d.Matrix)
		}
		if d.Snap && axisAligned {
			s := m.Storage()
			s[12] = float32(math.Round(float64(s[12])*subpixels) / subpixels)
			s[13] = float32(math.Round(float64(s[13])))
		}
		emit(d.Op, m)
	})
	return true
}

// isAxisAligned says whether the transform only scales and translates (or flips and
// swaps nothing): the text then lies along the pixel grid.
func isAxisAligned(m *ui.Matrix4) bool {
	s := m.Storage()
	return s[1] == 0 && s[2] == 0 && s[3] == 0 && s[4] == 0 && s[6] == 0 && s[7] == 0 && s[8] == 0 && s[9] == 0 &&
		s[11] == 0 && s[15] == 1
}
