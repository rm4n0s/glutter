package gles

import (
	"github.com/rm4n0s/glutter/ui"
)

// Vertices (Canvas.drawVertices): the triangles are drawn as they are, in device
// pixels, without anti-aliasing: a pixel is covered when its center is inside. The
// color of a pixel is the paint's, blended with the interpolated vertex colors by
// the call's blend mode, the paint being the source and the vertices the
// destination.

// vertices draws the triangles of the operation.
func (s *shapeContext) vertices(op ui.OpDrawVertices) (bool, error) {
	v := op.Vertices
	info, status, err := s.paint(&op.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	colors := v.Colors()
	mode := 0
	if len(colors) > 0 {
		var ok bool
		if mode, ok = blendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("vertices blend mode " + op.BlendMode.String())
		}
	}
	if len(v.TextureCoordinates()) > 0 && op.Paint.Shader() != nil {
		return false, s.b.unsupported("paint shader")
	}
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
		return true, nil
	}
	floats := make([]float32, 0, len(tris)*3*itemFloats[pipeVertices])
	for _, t := range tris {
		for _, i := range t {
			p := s.c.Transform.TransformPoint(pos[i])
			var c ui.Color
			if len(colors) > 0 {
				c = colors[i]
			}
			floats = append(floats, float32(p.Dx), float32(p.Dy), c.R, c.G, c.B, c.A)
		}
	}
	s.b.frame.add(callKey{
		pipe: pipeVertices, blend: info.blend, scissor: s.scissor, clip: s.mask,
		paint: info.color, mode: int32(mode), hasColors: len(colors) > 0, clipRect: s.clip,
	}, floats, len(tris)*3)
	return true, nil
}
