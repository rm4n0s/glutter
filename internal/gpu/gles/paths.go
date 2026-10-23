package gles

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// General paths (SPEC §9.1): the software rasterizer turns the shape into an
// 8-bit coverage mask, which is cached in the coverage atlas by the shape, the
// transform's linear part and its fractional translation, and the paint's stroke.
// The shape is cut into tiles of at most gpu.MaxAtlasTile pixels, anchored at the
// origin of the transformed shape's space, so that a shape drawn again at another
// whole-pixel offset, or only partly visible, finds its tiles. The GPU composites
// each tile with the paint's color.

// pathMask draws an operation's shape from the coverage atlas, with the paint's
// color. It is for the shapes the analytic pipeline cannot draw: paths, arcs, and
// the strokes and shapes that have no distance field here.
func (s *shapeContext) pathMask(op ui.DisplayOp, p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	sh, ok := gpu.ShapeOf(op)
	if !ok {
		return false, s.b.unsupported("operation " + typeName(op))
	}
	scale := s.scale()
	if !(scale > 0) {
		return true, nil
	}
	// The shape's bounds, widened by what its stroke and its edge can reach.
	tiles, err := s.b.res.masks.Tiles(op, sh, &s.c.Transform, gpu.MaskMargin(sh, scale), s.outer)
	if err != nil {
		return false, s.maskError(err)
	}
	for _, t := range tiles {
		var in [16]float32
		in[0], in[1], in[2], in[3] = float32(t.X0), float32(t.Y0), float32(t.X1), float32(t.Y1)
		in[4], in[5] = float32(t.AX), float32(t.AY)
		for i := 0; i < 4; i++ {
			in[8+i] = info.color[i]
		}
		copy(in[12:16], s.clip[:])
		key, ok := s.key(pipeMask, info, s.b.res.pages[t.Page], image.Rect(t.X0, t.Y0, t.X1, t.Y1))
		if !ok {
			continue
		}
		s.b.frame.add(key, in[:], 1)
	}
	return true, nil
}

// maskError reports an atlas that is full as an unsupported feature when frames
// skip them; otherwise, and for other errors, it returns the error as it is.
func (s *shapeContext) maskError(err error) error {
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == gpu.TagAtlasFull && s.b.skip {
		return s.b.unsupported("coverage atlas full")
	}
	return err
}
