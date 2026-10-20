package soft

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

// star is a star with n points around (cx, cy).
func star(cx, cy, outer, inner float64, n int) *ui.Path {
	var pts []ui.Offset
	for i := 0; i < 2*n; i++ {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		a := -math.Pi/2 + float64(i)*math.Pi/float64(n)
		pts = append(pts, ui.Offset{Dx: cx + r*math.Cos(a), Dy: cy + r*math.Sin(a)})
	}
	p := ui.NewPath()
	p.AddPolygon(pts, true)
	return p
}

func circle(cx, cy, r float64) *ui.Path {
	p := ui.NewPath()
	p.AddOval(ui.NewRectFromLTRB(cx-r, cy-r, cx+r, cy+r))
	return p
}

// TestPathCombineGoldens draws the result of every operation of PathCombine on two circles
// and on two stars: the two inputs as thin outlines, the result filled. One image of
// five cells for each pair, in the order difference, intersect, union, xor, reverse
// difference.
func TestPathCombineGoldens(t *testing.T) {
	ops := []ui.PathOperation{ui.PathOperationDifference, ui.PathOperationIntersect, ui.PathOperationUnion,
		ui.PathOperationXor, ui.PathOperationReverseDifference}
	pairs := []struct {
		name string
		a, b *ui.Path
	}{
		{"circles", circle(46, 56, 34), circle(80, 66, 34)},
		{"stars", star(46, 58, 40, 17, 5), star(78, 68, 40, 17, 6)},
		{"ring-and-circle", func() *ui.Path {
			p := circle(55, 60, 42)
			p.AddPath(circle(55, 60, 22), ui.Offset{})
			p.SetFillType(ui.PathFillTypeEvenOdd)
			return p
		}(), circle(88, 72, 30)},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			img := renderOf(t, 5*128, 128, func(c *ui.Canvas) {
				white(c)
				for i, op := range ops {
					c.Save()
					c.Translate(float64(i)*128, 0)
					c.ClipRect(ui.NewRectFromLTRB(0, 0, 128, 128))
					c.DrawPath(ui.PathCombine(op, pair.a, pair.b), paint(0xFF2E7D6B))
					c.DrawPath(pair.a, stroked(0xFFB3261E, 1, ui.StrokeCapButt, ui.StrokeJoinMiter))
					c.DrawPath(pair.b, stroked(0xFF1A5FB4, 1, ui.StrokeCapButt, ui.StrokeJoinMiter))
					c.Restore()
				}
			})
			goldens.Check(t, "pathcombine-"+pair.name, img)
		})
	}
}
