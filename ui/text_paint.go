package ui

// ParagraphDraw is one piece of the drawing of a paragraph: a shape, a glyph, a decoration
// line or a color glyph, which a renderer draws with the usual operations.
//
// Flutter: no counterpart
type ParagraphDraw struct {
	// Op is the drawing operation (OpDrawPath, OpDrawRect or OpDrawPicture), in the
	// coordinates of its own origin.
	Op DisplayOp
	// Offset is where the origin of Op is, from the top left corner of the paragraph.
	Offset Offset
	// Matrix, when not nil, transforms the coordinates of Op before Offset applies.
	Matrix *Matrix4
	// Snap says that the origin lands on the pixel grid the way text does: with its
	// horizontal position at a quarter of a pixel and its vertical one at a pixel, when the
	// transform allows it. Glyphs are snapped; the lines of decorations are not.
	Snap bool
}

// ParagraphPainter is a ParagraphLayout that can draw itself: the renderers draw a
// paragraph by drawing what it emits, in order.
//
// Flutter: no counterpart
type ParagraphPainter interface {
	// PaintParagraph calls emit for each piece of the drawing of the laid-out paragraph, in
	// drawing order: for each line its backgrounds, shadows, glyphs and decorations.
	PaintParagraph(emit func(ParagraphDraw))
}
