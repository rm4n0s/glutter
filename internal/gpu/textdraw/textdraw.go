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
// if the paragraph cannot draw itself. The transform that emit receives is valid only until
// emit returns: a caller that keeps it copies it.
//
// Flutter: no counterpart
func Expand(op ui.OpDrawParagraph, ctm *ui.Matrix4, emit func(ui.DisplayOp, *ui.Matrix4)) bool {
	return expandIn(op, ctm, nil, emit)
}

// RowPainter is a paragraph painter that can leave out the lines that lie outside a range of
// rows. The layout of the text engine is one.
//
// Flutter: no counterpart
type RowPainter interface {
	// PaintParagraphRows is PaintParagraph for the lines that may reach the rows from top to
	// bottom, in the coordinates of the paragraph; it may emit more.
	PaintParagraphRows(top, bottom float64, emit func(ui.ParagraphDraw))
}

// BoundsPainter is a paragraph painter that knows a rectangle that holds everything it draws,
// without drawing it. The layout of the text engine is one.
//
// Flutter: no counterpart
type BoundsPainter interface {
	// PaintBounds is a rectangle, in the coordinates of the paragraph, that holds everything
	// PaintParagraph emits (it may be larger).
	PaintBounds() ui.Rect
}

// Bounds returns a rectangle that holds everything the paragraph draws, in the coordinates of the
// paragraph's operation (its offset applied), and false when the paragraph cannot tell without
// being expanded: a caller then expands it.
//
// Flutter: no counterpart
func Bounds(op ui.OpDrawParagraph) (ui.Rect, bool) {
	if op.Paragraph == nil {
		return ui.Rect{}, true
	}
	bp, ok := op.Paragraph.Layouter().(BoundsPainter)
	if !ok {
		return ui.Rect{}, false
	}
	return bp.PaintBounds().Shift(op.Offset), true
}

// ExpandVisible is Expand for a caller that draws only the part of the paragraph that lies in
// visible, a rectangle of the surface in the coordinates that ctm maps to: the lines of a
// paragraph that lie wholly outside it are not expanded, so that a screenful of a long text
// costs what the screenful costs. It may emit operations outside visible (the caller's clip
// removes them), and all of those inside.
//
// Flutter: no counterpart
func ExpandVisible(op ui.OpDrawParagraph, ctm *ui.Matrix4, visible ui.Rect, emit func(ui.DisplayOp, *ui.Matrix4)) bool {
	return expandIn(op, ctm, &visible, emit)
}

func expandIn(op ui.OpDrawParagraph, ctm *ui.Matrix4, visible *ui.Rect, emit func(ui.DisplayOp, *ui.Matrix4)) bool {
	if op.Paragraph == nil {
		return true
	}
	layouter := op.Paragraph.Layouter()
	painter, ok := layouter.(ui.ParagraphPainter)
	if !ok {
		return false
	}
	axisAligned := isAxisAligned(ctm)
	// One matrix serves every piece: emit copies what it keeps, so the glyphs of a screen of
	// text do not allocate a matrix each.
	var m ui.Matrix4
	each := func(d ui.ParagraphDraw) {
		m = *ctm
		m.Translate(float32(op.Offset.Dx+d.Offset.Dx), float32(op.Offset.Dy+d.Offset.Dy), 0)
		if d.Matrix != nil {
			m.MultiplyBy(d.Matrix)
		}
		if d.Snap && axisAligned {
			s := m.Storage()
			s[12] = float32(math.Round(float64(s[12])*subpixels) / subpixels)
			s[13] = float32(math.Round(float64(s[13])))
		}
		emit(d.Op, &m)
	}
	if rp, ok := layouter.(RowPainter); ok && visible != nil {
		if top, bottom, ok := rows(ctm, op.Offset, *visible); ok {
			rp.PaintParagraphRows(top, bottom, each)
			return true
		}
	}
	painter.PaintParagraph(each)
	return true
}

// rows maps the visible rectangle to the paragraph's coordinates and returns the rows it
// covers. It says false for a transform that cannot be inverted or has a perspective, and
// for a rectangle that is not finite.
func rows(ctm *ui.Matrix4, offset ui.Offset, visible ui.Rect) (top, bottom float64, ok bool) {
	s := ctm.Storage()
	if s[3] != 0 || s[7] != 0 || s[15] != 1 || !visible.IsFinite() {
		return 0, 0, false
	}
	inv := *ctm
	if inv.Invert() == 0 {
		return 0, 0, false
	}
	local := inv.TransformRect(visible)
	if !local.IsFinite() {
		return 0, 0, false
	}
	return local.Top - offset.Dy, local.Bottom - offset.Dy, true
}

// isAxisAligned says whether the transform only scales and translates (or flips and
// swaps nothing): the text then lies along the pixel grid.
func isAxisAligned(m *ui.Matrix4) bool {
	s := m.Storage()
	return s[1] == 0 && s[2] == 0 && s[3] == 0 && s[4] == 0 && s[6] == 0 && s[7] == 0 && s[8] == 0 && s[9] == 0 &&
		s[11] == 0 && s[15] == 1
}
