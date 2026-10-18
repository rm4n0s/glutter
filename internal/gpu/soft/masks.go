package soft

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// OpMask returns the coverage of the shape that a drawing operation makes, drawn
// under the transform m, inside the area of the surface: the same coverage Render
// paints with, anti-aliased or not as the operation's paint says. It works for
// the operations that draw a shape from an outline: paths, rectangles, rounded
// rectangles, superellipses, DRRects, ovals, circles, arcs, lines and points. It
// returns nil when nothing is covered, and false for any other operation. The
// operation's mask filter and blend mode do not apply.
//
// This is what the GPU renderer rasterizes into its coverage atlas (SPEC §9.2).
//
// Flutter: no counterpart (SkMask)
func OpMask(op ui.DisplayOp, m *ui.Matrix4, area image.Rectangle) (*Mask, bool) {
	var (
		polys [][]ui.Offset
		rule  = FillNonZero
		p     *ui.Paint
	)
	switch v := op.(type) {
	case ui.OpDrawPath:
		p = &v.Paint
		polys, rule = pathPolys(v.Path, p, m, false)
	case ui.OpDrawRect:
		p = &v.Paint
		polys, rule = pathPolys(rectPath(v.Rect), p, m, false)
	case ui.OpDrawRRect:
		p = &v.Paint
		polys, rule = pathPolys(rrectPath(v.RRect), p, m, false)
	case ui.OpDrawRSuperellipse:
		p = &v.Paint
		polys, rule = pathPolys(rsuperellipsePath(v.RSuperellipse), p, m, false)
	case ui.OpDrawDRRect:
		p = &v.Paint
		polys, rule = pathPolys(drrectPath(v.Outer, v.Inner), p, m, false)
	case ui.OpDrawOval:
		p = &v.Paint
		polys, rule = pathPolys(ovalPath(v.Rect), p, m, false)
	case ui.OpDrawCircle:
		p = &v.Paint
		polys, rule = pathPolys(circlePath(v.Center, v.Radius), p, m, false)
	case ui.OpDrawArc:
		p = &v.Paint
		polys, rule = pathPolys(arcPath(v.Rect, v.StartAngle, v.SweepAngle, v.UseCenter, v.Paint.Style() != ui.PaintingStyleStroke), p, m, false)
	case ui.OpDrawLine:
		p = &v.Paint
		line := ui.NewPath()
		line.MoveTo(v.P1.Dx, v.P1.Dy)
		line.LineTo(v.P2.Dx, v.P2.Dy)
		polys, rule = pathPolys(line, p, m, true)
	case ui.OpDrawPoints:
		if len(v.Points) == 0 {
			return nil, true
		}
		p, polys = pointsPolys(v, m)
	default:
		return nil, false
	}
	mask := Rasterize(polys, rule, area)
	if !p.IsAntiAlias() {
		mask = threshold(mask)
	}
	return mask, true
}

// ClipMask returns the coverage of a clip shape: the path, filled by its fill
// type under the transform m, inside the area of the surface. Without
// antiAlias the coverage is all or nothing, sampled at the pixel centers. It
// returns nil when nothing is covered.
//
// Flutter: no counterpart (SkClipStack)
func ClipMask(path *ui.Path, m *ui.Matrix4, antiAlias bool, area image.Rectangle) *Mask {
	mask := Rasterize(flatPolys(path, m), fillRuleOf(path), area)
	if !antiAlias {
		mask = threshold(mask)
	}
	return mask
}
