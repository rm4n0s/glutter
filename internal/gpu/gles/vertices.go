package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
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
	mode := 0
	if len(colors) > 0 {
		var ok bool
		if mode, ok = blendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("vertices blend mode " + op.BlendMode.String())
		}
	}
	floats, bounds, ok := gpu.VertexTriangles(v, &s.c.Transform)
	if !ok {
		return true, nil
	}
	key, ok := s.key(pipeVertices, info, nil, bounds)
	if !ok {
		return true, nil
	}
	key.paint, key.mode, key.hasColors, key.atTexture, key.clipRect = info.color, int32(mode), len(colors) > 0, atTexture, s.clip
	s.b.frame.add(key, floats, len(floats)/gpu.VertexFloats)
	return true, nil
}
