package soft

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// The shapes are drawn as paths: the rasterizer gives exact coverage for the
// straight edges, and the curves are flattened to within flattenTolerance of a
// pixel.

func rectPath(r ui.Rect) *ui.Path {
	p := ui.NewPath()
	p.AddRect(r)
	return p
}

func rrectPath(r ui.RRect) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(r)
	return p
}

func rsuperellipsePath(r ui.RSuperellipse) *ui.Path {
	p := ui.NewPath()
	p.AddRSuperellipse(r)
	return p
}

func ovalPath(r ui.Rect) *ui.Path {
	p := ui.NewPath()
	p.AddOval(r)
	return p
}

func circlePath(center ui.Offset, radius float64) *ui.Path {
	return ovalPath(ui.NewRectFromCircle(center, radius))
}

// drrectPath is the region between two rounded rectangles: both outlines, filled
// with the even-odd rule.
func drrectPath(outer, inner ui.RRect) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(outer)
	p.AddRRect(inner)
	p.SetFillType(ui.PathFillTypeEvenOdd)
	return p
}

// arcPath is the path Skia draws for drawArc (SkPathPriv::CreateDrawArcPath).
// With useCenter it goes from the center of the oval to the arc and back, which
// makes a pie slice; without it, a filled arc is closed by a chord and a stroked
// one is left open. A filled arc of a full turn or more is the oval.
func arcPath(oval ui.Rect, start, sweep float64, useCenter, fill bool) *ui.Path {
	if fill && math.Abs(sweep) >= 2*math.Pi {
		return ovalPath(oval)
	}
	p := ui.NewPath()
	if useCenter {
		p.MoveTo((oval.Left+oval.Right)/2, (oval.Top+oval.Bottom)/2)
	}
	forceMoveTo := !useCenter
	for math.Abs(sweep) > 2*math.Pi {
		step := math.Copysign(2*math.Pi, sweep)
		p.ArcTo(oval, start, step, forceMoveTo)
		forceMoveTo = false
		start += step
		sweep -= step
	}
	p.ArcTo(oval, start, sweep, forceMoveTo)
	if useCenter {
		p.Close()
	}
	return p
}

// drawPoints draws points, lines or a polygon with the paint's stroke width and
// cap; the paint's style is ignored. A point is a round dot with a round cap and
// a square otherwise.
func (r *renderer) drawPoints(op ui.OpDrawPoints, st *state) error {
	ps, err := r.setup(&op.Paint, affineOf(st.m))
	if err != nil {
		return err
	}
	if st.clip.empty() || len(op.Points) == 0 {
		return nil
	}
	var contours []ui.FlatContour
	switch op.Mode {
	case ui.PointModeLines:
		for i := 0; i+1 < len(op.Points); i += 2 {
			contours = append(contours, ui.FlatContour{Points: []ui.Offset{op.Points[i], op.Points[i+1]}})
		}
	case ui.PointModePolygon:
		contours = []ui.FlatContour{{Points: op.Points}}
	default:
		for _, pt := range op.Points {
			contours = append(contours, ui.FlatContour{Points: []ui.Offset{pt}})
		}
	}
	p := &op.Paint
	if op.Mode == ui.PointModePoints && p.StrokeCap() != ui.StrokeCapRound {
		// A butt cap draws nothing for a point, but a point is still drawn.
		q := *p
		q.SetStrokeCap(ui.StrokeCapSquare)
		p = &q
	}
	r.fillPolys(strokePolys(contours, p, st.m), FillNonZero, p, ps, st)
	return nil
}
