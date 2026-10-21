package canvas2d

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// Presenter shows a finished frame in the canvas of a surface. The js build's
// Presenter puts it there with putImageData; tests give one that records the frames.
//
// Flutter: no counterpart
type Presenter interface {
	// Present shows img, an opaque image, in the canvas that the handle names, and sets
	// the canvas's size to the image's if it differs. It must not keep img.
	Present(h gpu.SurfaceHandle, img *image.RGBA) error
	// Release forgets the canvas of the handle.
	Release(h gpu.SurfaceHandle)
}

// Options are the optional parameters of NewRenderer.
//
// Flutter: no counterpart
type Options struct {
	// Background is what shows where the scene draws nothing. A canvas of the page is
	// opaque, so only the color's red, green and blue count; the zero value is black.
	Background ui.Color
}

// Renderer is a gpu.Renderer that draws with soft and presents through a Presenter.
//
// Flutter: no counterpart
type Renderer struct {
	present Presenter
	bg      [3]uint8
	closed  bool
}

var _ gpu.Renderer = (*Renderer)(nil)

// NewRenderer makes a renderer that presents through p. More than one options value
// panics with MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(p Presenter, opts ...Options) *Renderer {
	var o Options
	switch len(opts) {
	case 0:
	case 1:
		o = opts[0]
	default:
		panic(errors.New("MultipleOptions", "NewRenderer takes at most one Options value", "count", len(opts)))
	}
	return &Renderer{present: p, bg: [3]uint8{uint8(o.Background.Red()), uint8(o.Background.Green()), uint8(o.Background.Blue())}}
}

// surface is the surface of one canvas.
type surface struct {
	r         *Renderer
	h         gpu.SurfaceHandle
	destroyed bool
}

// Destroy forgets the canvas. It is safe to call twice.
//
// Flutter: no counterpart
func (s *surface) Destroy() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	s.r.present.Release(s.h)
}

// Attach starts drawing into the canvas that the handle names, which must be of kind
// gpu.SurfaceKindWeb. It fails with SurfaceKindUnsupported for another kind and with
// RendererClosed after Close.
//
// Flutter: no counterpart
func (r *Renderer) Attach(h gpu.SurfaceHandle) (gpu.Surface, error) {
	if r.closed {
		return nil, errors.New("RendererClosed", "the renderer was closed")
	}
	if h.Kind != gpu.SurfaceKindWeb {
		return nil, errors.New("SurfaceKindUnsupported", "canvas2d draws into the canvas of a web page only", "kind", string(h.Kind))
	}
	return &surface{r: r, h: h}, nil
}

// Render draws the scene into the surface at the size in device pixels and presents it.
// A nil scene draws the background only. The scene's coordinates are already in device
// pixels, so dpr is not used.
//
// Flutter: no counterpart
func (r *Renderer) Render(s gpu.Surface, scene *ui.Scene, size ui.Size, dpr float32) error {
	_, err := r.RenderCapture(s, scene, size, dpr)
	return err
}

// RenderCapture is Render that also returns the frame it presented, top row first. The
// image belongs to the caller.
//
// Flutter: no counterpart
func (r *Renderer) RenderCapture(s gpu.Surface, scene *ui.Scene, size ui.Size, _ float32) (*image.RGBA, error) {
	surf, ok := s.(*surface)
	if !ok || surf.r != r || surf.destroyed {
		return nil, errors.New("SurfaceInvalid", "the surface was not attached by this renderer or was destroyed")
	}
	img, err := r.frame(scene, int(size.Width), int(size.Height))
	if err != nil {
		return nil, err
	}
	if err := r.present.Present(surf.h, img); err != nil {
		return nil, err
	}
	return img, nil
}

// frame draws the scene with soft and lays it over the background.
func (r *Renderer) frame(scene *ui.Scene, width, height int) (*image.RGBA, error) {
	if scene == nil {
		scene = ui.NewSceneBuilder().Build()
	}
	img, err := soft.Render(scene, width, height)
	if err != nil {
		return nil, err
	}
	overBackground(img, r.bg)
	return img, nil
}

// Snapshot draws the scene into a new image of the size, without a canvas.
//
// Flutter: no counterpart
func (r *Renderer) Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) {
	if scene == nil {
		scene = ui.NewSceneBuilder().Build()
	}
	return soft.Render(scene, int(size.Width), int(size.Height))
}

// Close stops the renderer; its surfaces are destroyed. It is safe to call twice.
//
// Flutter: no counterpart
func (r *Renderer) Close() error {
	r.closed = true
	return nil
}

// overBackground lays the premultiplied image over an opaque background color, in
// place: every pixel ends up opaque, which is what a canvas without alpha shows.
func overBackground(img *image.RGBA, bg [3]uint8) {
	b := img.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := img.Pix[img.PixOffset(b.Min.X, y):][:4*b.Dx()]
		for x := 0; x < len(row); x += 4 {
			a := uint32(row[x+3])
			if a == 255 {
				continue
			}
			inv := 255 - a
			for c := 0; c < 3; c++ {
				// Premultiplied source plus the background through the rest of the pixel.
				v := uint32(row[x+c]) + (uint32(bg[c])*inv+127)/255
				if v > 255 {
					v = 255
				}
				row[x+c] = uint8(v)
			}
			row[x+3] = 255
		}
	}
}
