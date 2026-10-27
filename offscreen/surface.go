package offscreen

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// maxSurfaceSize is the largest width or height of a Surface, in pixels (the limit of the
// software renderer).
const maxSurfaceSize = 1 << 15

// Surface holds the pixels of one view across frames: Draw redraws only what changed
// since the last frame over the pixels of the frame before, and tells which rectangles it
// redrew. The pixels are those of a drawing of the whole scene. A Surface is used by one
// goroutine at a time; draw detached scenes (Detacher) when that is not the UI thread.
//
// Flutter: no counterpart
type Surface struct {
	img     *image.RGBA
	tracker *gpu.DamageTracker
	// last is the scene the pixels show, nil when they show none.
	last *ui.Scene
}

// NewSurface returns a surface of width x height pixels, transparent until the first
// Draw. It panics with the tag SurfaceSizeInvalid for a size that is not between 1 and
// 32768.
//
// Flutter: no counterpart
func NewSurface(width, height int) *Surface {
	if !validSize(width, height) {
		panic(errors.New(tagSurfaceSizeInvalid, "The surface size is not valid.", "width", width, "height", height))
	}
	return &Surface{img: image.NewRGBA(image.Rect(0, 0, width, height)), tracker: gpu.NewDamageTracker()}
}

func validSize(width, height int) bool {
	return width > 0 && height > 0 && width <= maxSurfaceSize && height <= maxSurfaceSize
}

// Resize makes the surface width x height pixels. The pixels are cleared, and the next
// Draw draws the whole scene. A surface of the same size is cleared too. It panics with
// the tag SurfaceSizeInvalid for a size that is not valid.
//
// Flutter: no counterpart
func (s *Surface) Resize(width, height int) {
	if !validSize(width, height) {
		panic(errors.New(tagSurfaceSizeInvalid, "The surface size is not valid.", "width", width, "height", height))
	}
	s.img = image.NewRGBA(image.Rect(0, 0, width, height))
	s.tracker.Invalidate()
	s.last = nil
}

// Width is the width of the surface in pixels.
//
// Flutter: no counterpart
func (s *Surface) Width() int { return s.img.Rect.Dx() }

// Height is the height of the surface in pixels.
//
// Flutter: no counterpart
func (s *Surface) Height() int { return s.img.Rect.Dy() }

// Image returns the pixels of the last frame, in premultiplied RGBA. The image is the
// surface's own: it is valid until the next Draw or Resize, and the caller does not change
// it.
//
// Flutter: no counterpart
func (s *Surface) Image() *image.RGBA { return s.img }

// Draw makes the pixels of the surface those of the scene, and returns the rectangles it
// redrew, in pixels.
//
//   - The first frame, the first after Resize or after an error, and a scene with something
//     the damage tracker cannot bound (an image filter, a backdrop filter) redraw
//     everything: the damage is the whole surface.
//   - Otherwise each damaged rectangle is cleared and the scene is drawn into it, clipped
//     to it, so the pixels equal those of a drawing of the whole scene. The damage is
//     empty for a scene that changes nothing, and nothing is drawn. The scene the surface
//     already shows, given again, changes nothing: this differs from the damage tracker of
//     a window (SPEC §9.1), which treats the same scene again as a request to draw all of
//     it, since the pixels of a surface are never lost.
//
// After an error from the software renderer (a layer or an operation it cannot draw, an
// image that was disposed) the pixels are not defined, and the next Draw redraws
// everything.
//
// Flutter: no counterpart
func (s *Surface) Draw(scene *ui.Scene) ([]image.Rectangle, error) {
	if scene == nil {
		return nil, errors.New(tagSurfaceSceneNil, "Draw needs a scene.")
	}
	if scene == s.last {
		return nil, nil
	}
	b := s.img.Rect
	damage := s.tracker.Update(scene, b.Dx(), b.Dy())
	rects := damage.Rects
	if damage.Full {
		rects = []image.Rectangle{b}
	}
	for _, r := range rects {
		clearRect(s.img, r)
		if err := soft.RenderInto(s.img, scene, r); err != nil {
			s.tracker.Invalidate()
			s.last = nil
			return nil, err
		}
	}
	s.last = scene
	return rects, nil
}

// clearRect makes the pixels of the rectangle transparent.
func clearRect(img *image.RGBA, r image.Rectangle) {
	r = r.Intersect(img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := img.PixOffset(r.Min.X, y)
		clear(img.Pix[i : i+4*r.Dx()])
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSurfaceSceneNil    = "SurfaceSceneNil"
	tagSurfaceSizeInvalid = "SurfaceSizeInvalid"
)
