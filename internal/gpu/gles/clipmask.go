package gles

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// clipMask is the coverage of the clip shapes of a clip (everything that is not
// an axis-aligned rectangle), drawn into an 8-bit texture over a rectangle of the
// surface. A draw under the clip multiplies its coverage by the mask's, as the
// software renderer does (SPEC §9.1, "Clips": antialiased through coverage masks).
type clipMask struct {
	tex        *Texture
	x, y, w, h int // the rectangle of the surface the texture covers
	lastUsed   uint64
}

// clipKey hashes what a clip mask is made of: its area and its shapes, with the
// transform each was clipped under.
func clipKey(area image.Rectangle, shapes []gpu.ClipShape) uint64 {
	h := newHasher()
	h.i(area.Min.X)
	h.i(area.Min.Y)
	h.i(area.Max.X)
	h.i(area.Max.Y)
	for i := range shapes {
		s := &shapes[i]
		h.i(int(s.Kind))
		h.b(s.AntiAlias)
		h.b(s.Difference)
		for _, v := range s.Transform.Storage() {
			h.f64(float64(v))
		}
		switch s.Kind {
		case gpu.ClipShapeRect:
			h.f64(s.Rect.Left)
			h.f64(s.Rect.Top)
			h.f64(s.Rect.Right)
			h.f64(s.Rect.Bottom)
		case gpu.ClipShapeRRect:
			rr := s.RRect
			for _, v := range [...]float64{rr.Left, rr.Top, rr.Right, rr.Bottom, rr.TlRadiusX, rr.TlRadiusY, rr.TrRadiusX, rr.TrRadiusY, rr.BrRadiusX, rr.BrRadiusY, rr.BlRadiusX, rr.BlRadiusY} {
				h.f64(v)
			}
		case gpu.ClipShapeRSuperellipse:
			rr := s.RSuperellipse
			for _, v := range [...]float64{rr.Left, rr.Top, rr.Right, rr.Bottom, rr.TlRadiusX, rr.TlRadiusY, rr.TrRadiusX, rr.TrRadiusY, rr.BrRadiusX, rr.BrRadiusY, rr.BlRadiusX, rr.BlRadiusY} {
				h.f64(v)
			}
		case gpu.ClipShapePath:
			h.u64(s.Path.Fingerprint())
		}
	}
	return uint64(h)
}

// shapePath is the outline of a clip shape.
func shapePath(s *gpu.ClipShape) *ui.Path {
	p := ui.NewPath()
	switch s.Kind {
	case gpu.ClipShapeRect:
		p.AddRect(s.Rect)
	case gpu.ClipShapeRRect:
		p.AddRRect(s.RRect)
	case gpu.ClipShapeRSuperellipse:
		p.AddRSuperellipse(s.RSuperellipse)
	case gpu.ClipShapePath:
		return s.Path
	}
	return p
}

// outerPixels returns the pixels that a rectangle of the surface touches, inside
// the surface of w x h pixels.
func outerPixels(r ui.Rect, w, h int) image.Rectangle {
	if !r.IsFinite() {
		return image.Rectangle{}
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	return image.Rect(clamp(math.Floor(r.Left)), clamp(math.Floor(r.Top)), clamp(math.Ceil(r.Right)), clamp(math.Ceil(r.Bottom))).
		Intersect(image.Rect(0, 0, w, h))
}

// clipMask returns the mask of the shapes of the clip over the clip's bounds
// inside a surface of w x h pixels, making it if there is none cached. empty says
// that the shapes leave nothing.
func (r *resources) clipMask(c gpu.Clip, w, h int) (m *clipMask, empty bool, err error) {
	area := outerPixels(c.Bounds, w, h)
	if area.Empty() {
		return nil, true, nil
	}
	key := clipKey(area, c.Shapes)
	if m, ok := r.clipMasks[key]; ok {
		m.lastUsed = r.frame
		return m, false, nil
	}
	r.clipMasksBuilt++
	cov := make([]float32, area.Dx()*area.Dy())
	for i := range cov {
		cov[i] = 1
	}
	for i := range c.Shapes {
		s := &c.Shapes[i]
		mask := soft.ClipMask(shapePath(s), &s.Transform, s.AntiAlias, area)
		if s.Difference {
			// Cut the shape out: where it covers, the clip is left by the rest.
			if mask != nil {
				cut(cov, area, mask, true)
			}
			continue
		}
		if mask == nil {
			return nil, true, nil
		}
		cut(cov, area, mask, false)
	}
	bytes := make([]byte, len(cov))
	any := false
	for i, v := range cov {
		b := byte(math.Min(math.Max(float64(v)*255+0.5, 0), 255))
		bytes[i] = b
		any = any || b != 0
	}
	if !any {
		return nil, true, nil
	}
	tex, err := r.dev.AcquireTexture(area.Dx(), area.Dy(), TextureFormatR8)
	if err != nil {
		return nil, false, err
	}
	r.dev.SetTextureParams(tex, glNearest, glNearest, glClampToEdge, glClampToEdge)
	if err := r.dev.UploadTexture(tex, 0, 0, area.Dx(), area.Dy(), bytes); err != nil {
		r.dev.ReleaseTexture(tex)
		return nil, false, err
	}
	m = &clipMask{tex: tex, x: area.Min.X, y: area.Min.Y, w: area.Dx(), h: area.Dy(), lastUsed: r.frame}
	r.clipMasks[key] = m
	r.clipBytes += len(bytes)
	return m, false, nil
}

// cut multiplies the coverage of a clip, over the area, by a shape's mask, or by
// what the mask leaves when subtract is set.
func cut(cov []float32, area image.Rectangle, mask *soft.Mask, subtract bool) {
	w := area.Dx()
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			v := mask.At(x, y)
			if subtract {
				v = 1 - v
			}
			cov[(y-area.Min.Y)*w+(x-area.Min.X)] *= v
		}
	}
}

func (r *resources) dropClipMask(key uint64, m *clipMask) {
	r.dev.ReleaseTexture(m.tex)
	r.clipBytes -= m.w * m.h
	delete(r.clipMasks, key)
}

// dropOldestClipMask drops the least recently used mask that this frame does not
// use; false when there is none.
func (r *resources) dropOldestClipMask() bool {
	var oldestKey uint64
	var oldest *clipMask
	for k, m := range r.clipMasks {
		if m.lastUsed != r.frame && (oldest == nil || m.lastUsed < oldest.lastUsed) {
			oldestKey, oldest = k, m
		}
	}
	if oldest == nil {
		return false
	}
	r.dropClipMask(oldestKey, oldest)
	return true
}
