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
