package explicit

import (
	"fmt"
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// pathMask draws an operation's shape from the coverage atlas, with the paint's color. It is
// for the shapes the analytic pipeline cannot draw: paths, arcs, and the strokes and shapes
// that have no distance field (ported from gles/paths.go).
func (s *shapeContext) pathMask(op ui.DisplayOp, p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p, true)
	if status != paintDraw {
		return status == paintNothing, err
	}
	sh, ok := gpu.ShapeOf(op)
	if !ok {
		return false, s.b.unsupported("operation " + typeName(op))
	}
	scale := s.scaleOf()
	if !(scale > 0) {
		return true, nil
	}
	tiles, err := s.b.res.masks.Tiles(op, sh, &s.c.Transform, gpu.MaskMargin(sh, scale), s.outer)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == "AtlasFull" && s.b.skip {
			return false, s.b.unsupported("coverage atlas full")
		}
		return false, err
	}
	for _, t := range tiles {
		var in [16]float32
		in[0], in[1], in[2], in[3] = float32(t.X0), float32(t.Y0), float32(t.X1), float32(t.Y1)
		in[4], in[5] = float32(t.AX), float32(t.AY)
		copy(in[8:12], info.color[:])
		copy(in[12:16], s.clip[:])
		key, ok := s.key(pipeMask, info, image.Rect(t.X0, t.Y0, t.X1, t.Y1))
		if !ok {
			continue
		}
		key.page = t.Page
		s.b.add(key, in[:])
	}
	return true, nil
}

// scaleOf is the factor that lengths in the shape's units grow by.
func (s *shapeContext) scaleOf() float64 {
	return (&gpu.ShapeEncoder{T: s.t}).Scale()
}

// typeName is the name of the dynamic type of an operation, for the list of unsupported
// features.
func typeName(op ui.DisplayOp) string { return fmt.Sprintf("%T", op) }

// imageEncoder returns the image encoder of the command for an image whose texture has the
// entry, which adds each instance it makes to the frame.
func (s *shapeContext) imageEncoder(info paintInfo, entry *imageEntry, p *ui.Paint) *gpu.ImageEncoder {
	return &gpu.ImageEncoder{
		T: s.t, Outer: s.outer, Clip: s.clip,
		Paint:   gpu.ShapePaint{Color: info.color, Antialias: info.aa},
		Quality: p.FilterQuality(), Levels: entry.levels,
		Emit: func(in *gpu.ImageInstance, covers image.Rectangle) {
			if key, ok := s.key(pipeImage, info, covers); ok {
				key.img = entry
				s.b.add(key, in[:])
			}
		},
	}
}

// image checks the image and the paint, and returns what an image draw needs; done says that
// the command needs no more work, and ok whether it counts as drawn.
func (s *shapeContext) image(img *ui.Image, p *ui.Paint) (info paintInfo, entry *imageEntry, done, ok bool, err error) {
	if img == nil || img.IsDisposed() {
		return info, nil, true, false, errors.New("GPUImageDisposed", "the image was disposed, or is nil, and cannot be drawn")
	}
	// The paint's shader does not apply to images, as in Skia.
	info, status, err := s.paintWith(p, false, s.t)
	if status != paintDraw {
		return info, nil, true, status == paintNothing, err
	}
	entry, err = s.b.res.imageTexture(img)
	if err != nil {
		if appErr, isErr := errors.FromError(err); isErr && appErr.Tag == "GPUImageTooLarge" && s.b.skip {
			return info, nil, true, false, s.b.unsupported("image larger than the textures of the device")
		}
		return info, nil, true, false, err
	}
	return info, entry, false, true, nil
}

// imageRect draws the part src of the image into dst, as Canvas.drawImageRect.
func (s *shapeContext) imageRect(img *ui.Image, src, dst ui.Rect, p *ui.Paint) (bool, error) {
	info, entry, done, ok, err := s.image(img, p)
	if done {
		return ok, err
	}
	s.imageEncoder(info, entry, p).Rect(img.Width(), img.Height(), src, dst)
	return true, nil
}

// imageNine draws an image as a nine-patch: the corners keep their size, the edges stretch
// along one axis and the center along both.
func (s *shapeContext) imageNine(op ui.OpDrawImageNine) (bool, error) {
	info, entry, done, ok, err := s.image(op.Image, &op.Paint)
	if done {
		return ok, err
	}
	s.imageEncoder(info, entry, &op.Paint).Nine(op.Image.Width(), op.Image.Height(), op.Center, op.Dst)
	return true, nil
}

// imageAtlas draws the parts of an atlas image, each moved by its transform. With colors, each
// sprite is the source and its color the destination of the blend mode, so that srcIn with a
// transparent white scales the sprite's alpha.
func (s *shapeContext) imageAtlas(op ui.OpDrawAtlas) (bool, error) {
	mode := 0
	if len(op.Colors) > 0 {
		var ok bool
		if mode, ok = gpu.BlendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("atlas blend mode " + op.BlendMode.String())
		}
	}
	info, entry, done, ok, err := s.image(op.Atlas, &op.Paint)
	if done {
		return ok, err
	}
	s.imageEncoder(info, entry, &op.Paint).Atlas(op.Atlas.Width(), op.Atlas.Height(), op, mode)
	return true, nil
}

// vertices draws the triangles of Canvas.drawVertices as they are, in device pixels, without
// anti-aliasing: a pixel is covered when its center is inside. The color of a pixel is the paint's,
// blended with the interpolated vertex colors by the call's blend mode, the paint being the source
// and the vertices the destination. A paint with a shader gives the shader's color at the pixel, or
// at the interpolated texture coordinates when the vertices have some.
func (s *shapeContext) vertices(op ui.OpDrawVertices) (bool, error) {
	v := op.Vertices
	atTexture := len(v.TextureCoordinates()) > 0 && op.Paint.Shader() != nil
	toDevice := s.t
	if atTexture {
		toDevice = gpu.Affine2D{A: 1, D: 1}
	}
	info, status, err := s.paintWith(&op.Paint, true, toDevice)
	if status != paintDraw {
		return status == paintNothing, err
	}
	colors := v.Colors()
	mode := 0
	if len(colors) > 0 {
		var ok bool
		if mode, ok = gpu.BlendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("vertices blend mode " + op.BlendMode.String())
		}
	}
	floats, bounds, ok := gpu.VertexTriangles(v, &s.c.Transform)
	if !ok {
		return true, nil
	}
	key, ok := s.key(pipeVertices, info, bounds)
	if !ok {
		return true, nil
	}
	key.vert = vertexParams{paint: info.color, clip: s.clip, mode: int32(mode), colors: len(colors) > 0, atTexture: atTexture}
	s.b.addItems(key, floats, len(floats)/gpu.VertexFloats)
	return true, nil
}
