package gles

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Vertices (Canvas.drawVertices): the triangles are drawn as they are, in device
// pixels, without anti-aliasing: a pixel is covered when its center is inside. The
// color of a pixel is the paint's, blended with the interpolated vertex colors by
// the call's blend mode, the paint being the source and the vertices the
// destination. A paint with a shader gives the shader's color at the pixel, or
// at the interpolated texture coordinates when the vertices have some.

// vertices draws the triangles of the operation.
func (s *shapeContext) vertices(op ui.OpDrawVertices) (bool, error) {
	v := op.Vertices
	atTexture := len(v.TextureCoordinates()) > 0 && op.Paint.Shader() != nil
	toDevice := s.t
	if atTexture {
		toDevice = affine2D{A: 1, D: 1}
	}
	info, status, err := s.paintWith(&op.Paint, true, toDevice)
	if status != paintDraw {
		return status == paintNothing, err
	}
	colors := v.Colors()
	tex := v.TextureCoordinates()
	mode := 0
	if len(colors) > 0 {
		var ok bool
		if mode, ok = blendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("vertices blend mode " + op.BlendMode.String())
		}
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
	lo, hi := ui.Offset{Dx: math.Inf(1), Dy: math.Inf(1)}, ui.Offset{Dx: math.Inf(-1), Dy: math.Inf(-1)}
	for _, t := range tris {
		for _, i := range t {
			p := s.c.Transform.TransformPoint(pos[i])
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
		return true, nil
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	bounds := image.Rect(clamp(math.Floor(lo.Dx)), clamp(math.Floor(lo.Dy)), clamp(math.Ceil(hi.Dx)), clamp(math.Ceil(hi.Dy)))
	key, ok := s.key(pipeVertices, info, nil, bounds)
	if !ok {
		return true, nil
	}
	key.paint, key.mode, key.hasColors, key.atTexture, key.clipRect = info.color, int32(mode), len(colors) > 0, atTexture, s.clip
	s.b.frame.add(key, floats, len(tris)*3)
	return true, nil
}
