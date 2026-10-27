package gpu

import "github.com/rm4n0s/glutter/ui"

// PaintOf returns the paint of a drawing operation; nil for the operations without one.
//
// Flutter: no counterpart
func PaintOf(op ui.DisplayOp) *ui.Paint {
	switch v := op.(type) {
	case ui.OpDrawPath:
		return &v.Paint
	case ui.OpDrawRect:
		return &v.Paint
	case ui.OpDrawRRect:
		return &v.Paint
	case ui.OpDrawRSuperellipse:
		return &v.Paint
	case ui.OpDrawDRRect:
		return &v.Paint
	case ui.OpDrawOval:
		return &v.Paint
	case ui.OpDrawCircle:
		return &v.Paint
	case ui.OpDrawArc:
		return &v.Paint
	case ui.OpDrawLine:
		return &v.Paint
	case ui.OpDrawPoints:
		return &v.Paint
	case ui.OpDrawImage:
		return &v.Paint
	case ui.OpDrawImageRect:
		return &v.Paint
	case ui.OpDrawImageNine:
		return &v.Paint
	case ui.OpDrawAtlas:
		return &v.Paint
	case ui.OpDrawVertices:
		return &v.Paint
	case ui.OpDrawPaint:
		return &v.Paint
	}
	return nil
}

// CopyPaintOf copies the paint of a drawing operation into dst and says whether the operation
// has one. It is PaintOf for a caller that does not keep the paint: the copy lives in dst, where
// PaintOf makes the operation escape to the heap for every call, which a frame of thousands of
// glyphs cannot afford.
//
// Flutter: no counterpart
func CopyPaintOf(op ui.DisplayOp, dst *ui.Paint) bool {
	switch v := op.(type) {
	case ui.OpDrawPath:
		*dst = v.Paint
	case ui.OpDrawRect:
		*dst = v.Paint
	case ui.OpDrawRRect:
		*dst = v.Paint
	case ui.OpDrawRSuperellipse:
		*dst = v.Paint
	case ui.OpDrawDRRect:
		*dst = v.Paint
	case ui.OpDrawOval:
		*dst = v.Paint
	case ui.OpDrawCircle:
		*dst = v.Paint
	case ui.OpDrawArc:
		*dst = v.Paint
	case ui.OpDrawLine:
		*dst = v.Paint
	case ui.OpDrawPoints:
		*dst = v.Paint
	case ui.OpDrawImage:
		*dst = v.Paint
	case ui.OpDrawImageRect:
		*dst = v.Paint
	case ui.OpDrawImageNine:
		*dst = v.Paint
	case ui.OpDrawAtlas:
		*dst = v.Paint
	case ui.OpDrawVertices:
		*dst = v.Paint
	case ui.OpDrawPaint:
		*dst = v.Paint
	default:
		return false
	}
	return true
}

// PaintFilters says whether the paint of an operation has an image filter and a mask filter;
// both are false for an operation without a paint.
//
// Flutter: no counterpart
func PaintFilters(op ui.DisplayOp) (imageFilter, maskFilter bool) {
	var p ui.Paint
	if !CopyPaintOf(op, &p) {
		return false, false
	}
	return p.ImageFilter() != nil, p.MaskFilter() != nil
}
