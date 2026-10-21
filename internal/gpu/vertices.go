package gpu

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// VertexFloats is the size of a vertex that VertexTriangles makes, in floats: the position
// in pixels (x, y), the color not premultiplied (r, g, b, a), and the texture coordinates.
//
// Flutter: no counterpart
const VertexFloats = 8

// VertexTriangles turns the vertices of Canvas.drawVertices into the triangles that are
// drawn, three vertices of VertexFloats floats each, with the positions under the transform.
// bounds are the pixels the triangles touch; ok is false when there is nothing to draw.
//
// Flutter: no counterpart
func VertexTriangles(v *ui.Vertices, m *ui.Matrix4) (floats []float32, bounds image.Rectangle, ok bool) {
	colors := v.Colors()
	tex := v.TextureCoordinates()
	pos := v.Positions()
	index := func(i int) int { return i }
	n := len(pos)
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
	if len(tris) == 0 {
		return nil, image.Rectangle{}, false
	}
	floats = make([]float32, 0, len(tris)*3*VertexFloats)
	lo, hi := ui.Offset{Dx: math.Inf(1), Dy: math.Inf(1)}, ui.Offset{Dx: math.Inf(-1), Dy: math.Inf(-1)}
	for _, t := range tris {
		for _, i := range t {
			p := m.TransformPoint(pos[i])
			lo.Dx, lo.Dy = math.Min(lo.Dx, p.Dx), math.Min(lo.Dy, p.Dy)
			hi.Dx, hi.Dy = math.Max(hi.Dx, p.Dx), math.Max(hi.Dy, p.Dy)
			var c ui.Color
			if len(colors) > 0 {
				c = colors[i]
			}
			var uv ui.Offset
			if len(tex) > 0 {
				uv = tex[i]
			}
			floats = append(floats, float32(p.Dx), float32(p.Dy), c.R, c.G, c.B, c.A, float32(uv.Dx), float32(uv.Dy))
		}
	}
	if !(lo.Dx <= hi.Dx && lo.Dy <= hi.Dy) {
		return nil, image.Rectangle{}, false
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	return floats, image.Rect(clamp(math.Floor(lo.Dx)), clamp(math.Floor(lo.Dy)), clamp(math.Ceil(hi.Dx)), clamp(math.Ceil(hi.Dy))), true
}
