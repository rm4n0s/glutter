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

// outerPixels returns the pixels that a rectangle of the surface touches, inside
// the surface of w x h pixels.
func outerPixels(r ui.Rect, w, h int) image.Rectangle {
	return pixelRect(r).Intersect(image.Rect(0, 0, w, h))
}

// pixelRect returns the pixels that a rectangle touches.
func pixelRect(r ui.Rect) image.Rectangle {
	if !r.IsFinite() {
		return image.Rectangle{}
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	return image.Rect(clamp(math.Floor(r.Left)), clamp(math.Floor(r.Top)), clamp(math.Ceil(r.Right)), clamp(math.Ceil(r.Bottom)))
}

// clipMask returns the mask of the shapes of the clip over the clip's bounds
// inside a surface of w x h pixels, making it if there is none cached. empty says
// that the shapes leave nothing.
func (r *resources) clipMask(c gpu.Clip, w, h int) (m *clipMask, empty bool, err error) {
	area := outerPixels(c.Bounds, w, h)
	if area.Empty() {
		return nil, true, nil
	}
	key := gpu.ClipKey(area, c.Shapes)
	if m, ok := r.clipMasks[key]; ok {
		m.lastUsed = r.frame
		return m, false, nil
	}
	r.clipMasksBuilt++
	bytes, empty := gpu.ClipCoverage(c, area, func(path *ui.Path, m *ui.Matrix4, aa bool, a image.Rectangle) (image.Rectangle, []float32) {
		mask := soft.ClipMask(path, m, aa, a)
		if mask == nil {
			return image.Rectangle{}, nil
		}
		return mask.Rect, mask.Cov
	})
	if empty {
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
