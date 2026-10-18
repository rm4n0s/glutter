package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// vertex is a corner of a triangle in device coordinates, with its color
// unpremultiplied.
type vertex struct {
	p ui.Offset
	c [4]float32
}

// drawVertices draws the triangles of the vertices. The corners' colors are
// interpolated across each triangle. Triangles are not anti-aliased: a pixel is
// covered when its center is inside, and where it is on an edge shared by two
// triangles exactly one of them gets it.
//
// A paint shader (gradients and images), and vertex colors combined with the
// paint's color by a blend mode other than Dst or Src, come with step 2.8.
func (r *renderer) drawVertices(op ui.OpDrawVertices, st *state) error {
	if err := checkPaint(&op.Paint); err != nil {
		return err
	}
	v := op.Vertices
	colors := v.Colors()
	useColors := len(colors) > 0
	if useColors && op.BlendMode != ui.BlendModeDst && op.BlendMode != ui.BlendModeSrc {
		return notImplemented("vertices blend mode", op.BlendMode)
	}
	if op.BlendMode == ui.BlendModeSrc {
		useColors = false // the paint wins
	}
	paintColor := op.Paint.Color()
	pos := v.Positions()
	corners := make([]vertex, len(pos))
	m := st.m
	for i, p := range pos {
		corners[i].p = m.TransformPoint(p)
		c := paintColor
		if useColors {
			c = colors[i]
		}
		corners[i].c = [4]float32{float32(c.Red()) / 255, float32(c.Green()) / 255, float32(c.Blue()) / 255, float32(c.Alpha()) / 255}
	}
	index := func(i int) int { return i }
	n := len(corners)
	if idx := v.Indices(); len(idx) > 0 {
		n = len(idx)
		index = func(i int) int { return int(idx[i]) }
	}
	var tris [][3]int
	switch v.Mode() {
	case ui.VertexModeTriangleStrip:
		for i := 0; i+2 < n; i++ {
			tris = append(tris, [3]int{index(i), index(i + 1), index(i + 2)})
		}
	case ui.VertexModeTriangleFan:
		for i := 1; i+1 < n; i++ {
			tris = append(tris, [3]int{index(0), index(i), index(i + 1)})
		}
	default:
		for i := 0; i+2 < n; i += 3 {
			tris = append(tris, [3]int{index(i), index(i + 1), index(i + 2)})
		}
	}
	for _, t := range tris {
		r.triangle(corners[t[0]], corners[t[1]], corners[t[2]], st)
	}
	return nil
}

// triangle paints one triangle.
func (r *renderer) triangle(a, b, c vertex, st *state) {
	area := edge(a.p, b.p, c.p)
	if area == 0 || math.IsNaN(area) || math.IsInf(area, 0) {
		return
	}
	if area < 0 { // wind them all one way, so that the tie rule is consistent
		b, c = c, b
		area = -area
	}
	minX := math.Min(a.p.Dx, math.Min(b.p.Dx, c.p.Dx))
	maxX := math.Max(a.p.Dx, math.Max(b.p.Dx, c.p.Dx))
	minY := math.Min(a.p.Dy, math.Min(b.p.Dy, c.p.Dy))
	maxY := math.Max(a.p.Dy, math.Max(b.p.Dy, c.p.Dy))
	box := image.Rect(int(math.Floor(math.Max(minX, -maxCoord))), int(math.Floor(math.Max(minY, -maxCoord))),
		int(math.Ceil(math.Min(maxX, maxCoord))), int(math.Ceil(math.Min(maxY, maxCoord)))).Intersect(st.clip.limit(r.img.Rect))
	if box.Empty() {
		return
	}
	mask := &Mask{Rect: box, Cov: make([]float32, box.Dx()*box.Dy())}
	any := false
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			p := ui.Offset{Dx: float64(x) + 0.5, Dy: float64(y) + 0.5}
			if covers(b.p, c.p, p) && covers(c.p, a.p, p) && covers(a.p, b.p, p) {
				mask.Cov[(y-box.Min.Y)*box.Dx()+(x-box.Min.X)] = 1
				any = true
			}
		}
	}
	if !any {
		return
	}
	r.composite(mask, st.clip, func(x, y int) [4]float32 {
		p := ui.Offset{Dx: float64(x) + 0.5, Dy: float64(y) + 0.5}
		wa, wb, wc := float32(edge(b.p, c.p, p)/area), float32(edge(c.p, a.p, p)/area), float32(edge(a.p, b.p, p)/area)
		var px [4]float32
		for k := 0; k < 4; k++ {
			px[k] = wa*a.c[k] + wb*b.c[k] + wc*c.c[k]
		}
		return [4]float32{px[0] * px[3], px[1] * px[3], px[2] * px[3], px[3]}
	})
}

// edge is the cross product (b-a) x (p-a): twice the signed area of a, b, p.
func edge(a, b, p ui.Offset) float64 {
	return (b.Dx-a.Dx)*(p.Dy-a.Dy) - (b.Dy-a.Dy)*(p.Dx-a.Dx)
}

// covers says whether the point p is on the inner side of the edge a to b, where
// a point exactly on the edge is inside only on a top or left edge.
func covers(a, b, p ui.Offset) bool {
	e := edge(a, b, p)
	if e != 0 {
		return e > 0
	}
	dx, dy := b.Dx-a.Dx, b.Dy-a.Dy
	return dy < 0 || (dy == 0 && dx > 0)
}
