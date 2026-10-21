package gpu

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// ShapeBounds returns the bounds of what a shape-drawing operation covers in its own
// coordinates, strokes included, and false for an operation that is not one of the
// shapes (rectangles, rounded rectangles, ovals, circles, arcs, paths, lines and
// points).
//
// Flutter: no counterpart
func ShapeBounds(op ui.DisplayOp) (ui.Rect, bool) {
	var r ui.Rect
	var p *ui.Paint
	switch v := op.(type) {
	case ui.OpDrawRect:
		r, p = v.Rect, &v.Paint
	case ui.OpDrawRRect:
		r, p = rectOfRRect(v.RRect), &v.Paint
	case ui.OpDrawOval:
		r, p = v.Rect, &v.Paint
	case ui.OpDrawCircle:
		r, p = ui.NewRectFromCircle(v.Center, v.Radius), &v.Paint
	case ui.OpDrawDRRect:
		r, p = rectOfRRect(v.Outer), &v.Paint
	case ui.OpDrawRSuperellipse:
		r, p = rectOfRRect(ui.RRect(v.RSuperellipse)), &v.Paint
	case ui.OpDrawArc:
		r, p = v.Rect, &v.Paint
	case ui.OpDrawPath:
		r, p = v.Path.GetBounds(), &v.Paint
	case ui.OpDrawLine:
		r = ui.Rect{Left: min(v.P1.Dx, v.P2.Dx), Top: min(v.P1.Dy, v.P2.Dy), Right: max(v.P1.Dx, v.P2.Dx), Bottom: max(v.P1.Dy, v.P2.Dy)}
		p = &v.Paint
	case ui.OpDrawPoints:
		if len(v.Points) == 0 {
			return ui.Rect{}, false
		}
		r = pointsBounds(v.Points)
		p = &v.Paint
	default:
		return ui.Rect{}, false
	}
	if p.Style() == ui.PaintingStyleStroke || isLineOrPoints(op) {
		// A stroke reaches half its width, or a miter's length, past the geometry.
		grow := max(p.StrokeWidth(), 0) / 2 * max(math.Sqrt2, p.StrokeMiterLimit())
		r = ui.Rect{Left: r.Left - grow, Top: r.Top - grow, Right: r.Right + grow, Bottom: r.Bottom + grow}
	}
	return r, r.IsFinite()
}

func rectOfRRect(r ui.RRect) ui.Rect {
	return ui.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
}

func isLineOrPoints(op ui.DisplayOp) bool {
	switch op.(type) {
	case ui.OpDrawLine, ui.OpDrawPoints:
		return true
	}
	return false
}

func pointsBounds(points []ui.Offset) ui.Rect {
	r := ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
	for _, pt := range points {
		r = ui.Rect{Left: min(r.Left, pt.Dx), Top: min(r.Top, pt.Dy), Right: max(r.Right, pt.Dx), Bottom: max(r.Bottom, pt.Dy)}
	}
	return r
}

// OpPaint returns the paint that a drawing operation is drawn with, or nil for an
// operation that has none (clips, transforms, pictures, paragraphs, shadows, colors).
//
// Flutter: no counterpart
func OpPaint(op ui.DisplayOp) *ui.Paint {
	switch v := op.(type) {
	case ui.OpSaveLayer:
		return &v.Paint
	case ui.OpDrawPaint:
		return &v.Paint
	case ui.OpDrawLine:
		return &v.Paint
	case ui.OpDrawRect:
		return &v.Paint
	case ui.OpDrawRRect:
		return &v.Paint
	case ui.OpDrawDRRect:
		return &v.Paint
	case ui.OpDrawRSuperellipse:
		return &v.Paint
	case ui.OpDrawOval:
		return &v.Paint
	case ui.OpDrawCircle:
		return &v.Paint
	case ui.OpDrawArc:
		return &v.Paint
	case ui.OpDrawPath:
		return &v.Paint
	case ui.OpDrawImage:
		return &v.Paint
	case ui.OpDrawImageRect:
		return &v.Paint
	case ui.OpDrawImageNine:
		return &v.Paint
	case ui.OpDrawPoints:
		return &v.Paint
	case ui.OpDrawVertices:
		return &v.Paint
	case ui.OpDrawAtlas:
		return &v.Paint
	}
	return nil
}

// The constants of Skia's shadows, which soft and gles draw (SPEC §9.1): the light is
// 600 pixels above the surface with a radius of 800, the ambient blur grows by half a
// pixel for every pixel of elevation up to 300.
const (
	shadowLightHeight = 600.0
	shadowLightRadius = 800.0
)

func blurSigma(radius float64) float64 {
	if radius > 0 {
		return 0.57735*radius + 0.5
	}
	return 0
}

// ShadowBounds returns the rectangle, in the path's coordinates, that Canvas.drawShadow
// can change: the ambient shadow (the shape blurred in place) and the spot shadow (the
// shape grown about its center, moved down, and blurred), both reaching three sigma.
// It returns false when the bounds cannot be known (a path that is not finite).
//
// Flutter: no counterpart
func ShadowBounds(op ui.OpDrawShadow) (ui.Rect, bool) {
	b := op.Path.GetBounds()
	if !b.IsFinite() || !(op.Elevation > 0) || op.Elevation >= shadowLightHeight {
		return b, b.IsFinite()
	}
	z := op.Elevation
	ambient := blurSigma(min(z*0.5, 300))
	spot := blurSigma(shadowLightRadius * z / (shadowLightHeight - z))
	scale := shadowLightHeight / (shadowLightHeight - z)
	dy := z * (shadowLightHeight + b.Height()/2) / (shadowLightHeight - z)
	cx, cy := (b.Left+b.Right)/2, (b.Top+b.Bottom)/2
	spotRect := ui.Rect{
		Left: cx + (b.Left-cx)*scale, Right: cx + (b.Right-cx)*scale,
		Top: cy + (b.Top-cy)*scale + dy, Bottom: cy + (b.Bottom-cy)*scale + dy,
	}
	grow := func(r ui.Rect, by float64) ui.Rect {
		return ui.Rect{Left: r.Left - by, Top: r.Top - by, Right: r.Right + by, Bottom: r.Bottom + by}
	}
	out := grow(b, 3*ambient).ExpandToInclude(grow(spotRect, 3*spot))
	return out, out.IsFinite()
}
