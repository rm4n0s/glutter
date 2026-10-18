package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// vertex is a corner of a triangle in device coordinates, with its color
// unpremultiplied and its texture coordinates.
type vertex struct {
	p   ui.Offset
	c   rgba
	tex ui.Offset
}

// drawVertices draws the triangles of the vertices. The color of a pixel is the
// paint's (its shader, evaluated at the interpolated texture coordinates when
// there are some, or its color) blended with the interpolated vertex colors by
// the blend mode of the call, the paint being the source and the vertices the
// destination; the paint's color filters and own blend mode then apply.
// Triangles are not anti-aliased: a pixel is covered when its center is inside,
// and where it is on an edge shared by two triangles exactly one of them gets it.
func (r *renderer) drawVertices(op ui.OpDrawVertices, st *state) error {
	p := &op.Paint
	if err := checkEffects(p); err != nil {
		return err
	}
	v := op.Vertices
	colors, tex := v.Colors(), v.TextureCoordinates()
	var paintAt func(x, y int, t ui.Offset) rgba
	if sh := p.Shader(); sh != nil && len(tex) > 0 {
		ev, _, err := r.shaderEval(sh, affine{a: 1, d: 1})
		if err != nil {
			return err
		}
		a := float32(p.Color().Alpha()) / 255
		paintAt = func(_, _ int, t ui.Offset) rgba {
			c := ev(t)
			return rgba{c[0] * a, c[1] * a, c[2] * a, c[3] * a}
		}
	} else {
		src, _, err := r.baseSource(p, affineOf(st.m))
		if err != nil {
			return err
		}
		paintAt = func(x, y int, _ ui.Offset) rgba { return src(x, y) }
	}
	filter := filtersOf(p)
	useColors := len(colors) > 0
	mode := op.BlendMode
	shade := func(x, y int, vc rgba, t ui.Offset) rgba {
		c := paintAt(x, y, t)
		if useColors {
			c = blend(mode, c, premul(vc))
		}
		if filter != nil {
			c = filter(c)
		}
		return c
	}

	pos := v.Positions()
	corners := make([]vertex, len(pos))
	for i, pt := range pos {
		corners[i].p = st.m.TransformPoint(pt)
		if useColors {
			c := colors[i]
			corners[i].c = rgba{float32(c.Red()) / 255, float32(c.Green()) / 255, float32(c.Blue()) / 255, float32(c.Alpha()) / 255}
		}
		if len(tex) > 0 {
			corners[i].tex = tex[i]
		}
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
		r.triangle(corners[t[0]], corners[t[1]], corners[t[2]], st, p.BlendMode(), shade)
	}
	return nil
}

// triangle paints one triangle with the colors shade gives.
func (r *renderer) triangle(a, b, c vertex, st *state, mode ui.BlendMode, shade func(x, y int, vc rgba, t ui.Offset) rgba) {
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
	r.composite(mask, st.clip, paintSetup{mode: mode, src: func(x, y int) rgba {
		p := ui.Offset{Dx: float64(x) + 0.5, Dy: float64(y) + 0.5}
		wa, wb, wc := edge(b.p, c.p, p)/area, edge(c.p, a.p, p)/area, edge(a.p, b.p, p)/area
		var vc rgba
		for k := range vc {
			vc[k] = float32(wa)*a.c[k] + float32(wb)*b.c[k] + float32(wc)*c.c[k]
		}
		t := ui.Offset{Dx: wa*a.tex.Dx + wb*b.tex.Dx + wc*c.tex.Dx, Dy: wa*a.tex.Dy + wb*b.tex.Dy + wc*c.tex.Dy}
		return shade(x, y, vc, t)
	}})
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
