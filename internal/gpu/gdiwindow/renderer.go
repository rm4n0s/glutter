package gdiwindow

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// Presenter shows a finished frame in a window. The Windows build's Presenter blits it with GDI;
// tests give one that records the frames.
//
// Flutter: no counterpart
type Presenter interface {
	// Present shows img, an opaque image, in the window that the handle names, at 1:1. damage says
	// which part changed from the frame before, which is what the window already shows; when it is
	// Full, or the size of the image changed, all of the image is shown. It must not keep img.
	Present(h gpu.SurfaceHandle, img *image.RGBA, damage gpu.Damage) error
	// Release forgets what the presenter knows about the window of the handle.
	Release(h gpu.SurfaceHandle)
}

// Options are the optional parameters of NewRenderer.
//
// Flutter: no counterpart
type Options struct {
	// Background is what shows where the scene draws nothing. A window of Windows is opaque here,
	// so only the color's red, green and blue count; the zero value is black.
	Background ui.Color
}

// Renderer is a gpu.Renderer that draws with soft and presents through a Presenter.
//
// Flutter: shell/platform/windows/compositor_software.cc CompositorSoftware
type Renderer struct {
	present Presenter
	bg      [3]uint8
	closed  bool
}

var _ gpu.Renderer = (*Renderer)(nil)

// NewRenderer makes a renderer that presents through p. More than one options value panics with
// MultipleOptions.
//
// Flutter: no counterpart
func NewRenderer(p Presenter, opts ...Options) *Renderer {
	var o Options
	switch len(opts) {
	case 0:
	case 1:
		o = opts[0]
	default:
		panic(errors.New(tagMultipleOptions, "NewRenderer takes at most one Options value", "count", len(opts)))
	}
	return &Renderer{present: p, bg: [3]uint8{uint8(o.Background.Red()), uint8(o.Background.Green()), uint8(o.Background.Blue())}}
}

// surface is the surface of one window. It tracks what each frame changed from the one before.
type surface struct {
	r         *Renderer
	h         gpu.SurfaceHandle
	tracker   *gpu.DamageTracker
	destroyed bool
}

// Destroy forgets the window. It is safe to call twice.
//
// Flutter: no counterpart
func (s *surface) Destroy() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	s.r.present.Release(s.h)
}

// Invalidate says that the window lost its content, so that the next frame is presented whole.
//
// Flutter: no counterpart
func (s *surface) Invalidate() { s.tracker.Invalidate() }

// Attach starts drawing into the window that the handle names, which must be of kind
// gpu.SurfaceKindWin32. It fails with SurfaceKindUnsupported for another kind and with
// RendererClosed after Close.
//
// Flutter: no counterpart
func (r *Renderer) Attach(h gpu.SurfaceHandle) (gpu.Surface, error) {
	if r.closed {
		return nil, errors.New(tagRendererClosed, "the renderer was closed")
	}
	if h.Kind != gpu.SurfaceKindWin32 || h.Window == 0 {
		return nil, errors.New(tagSurfaceKindUnsupported, "gdiwindow draws into the window of a Win32 surface only", "kind", string(h.Kind))
	}
	return &surface{r: r, h: h, tracker: gpu.NewDamageTracker()}, nil
}

// Invalidate says that the window of the surface lost its content, so that the next frame is
// presented whole. A surface that is not this renderer's is ignored.
//
// Flutter: no counterpart
func (r *Renderer) Invalidate(s gpu.Surface) {
	if surf, ok := s.(*surface); ok && surf.r == r {
		surf.Invalidate()
	}
}

// Render draws the scene into the surface at the size in device pixels and presents it. A nil
// scene draws the background only. The scene's coordinates are already in device pixels, so dpr
// is not used.
//
// Flutter: no counterpart
func (r *Renderer) Render(s gpu.Surface, scene *ui.Scene, size ui.Size, dpr float32) error {
	_, err := r.RenderCapture(s, scene, size, dpr)
	return err
}

// RenderCapture is Render that also returns the frame it presented, top row first. The image
// belongs to the caller.
//
// Flutter: no counterpart
func (r *Renderer) RenderCapture(s gpu.Surface, scene *ui.Scene, size ui.Size, _ float32) (*image.RGBA, error) {
	surf, ok := s.(*surface)
	if !ok || surf.r != r || surf.destroyed {
		return nil, errors.New(tagSurfaceInvalid, "the surface was not attached by this renderer or was destroyed")
	}
	w, h := int(size.Width), int(size.Height)
	img, err := r.frame(scene, w, h)
	if err != nil {
		return nil, err
	}
	damage := surf.tracker.Update(scene, w, h)
	if err := r.present.Present(surf.h, img, damage); err != nil {
		surf.tracker.Invalidate() // the window may not show this frame
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

// Snapshot draws the scene into a new image of the size, without a window.
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

// overBackground lays the premultiplied image over an opaque background color, in place: every
// pixel ends up opaque, which is what an opaque window shows.
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMultipleOptions        = "MultipleOptions"
	tagRendererClosed         = "RendererClosed"
	tagSurfaceInvalid         = "SurfaceInvalid"
	tagSurfaceKindUnsupported = "SurfaceKindUnsupported"
)
