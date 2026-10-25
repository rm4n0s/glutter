package engine

import (
	"image"
	"log/slog"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gdiwindow"
	"github.com/rm4n0s/glutter/ui"
)

// gdiRenderer is the engine's renderer of last resort on Windows: soft draws the frame and the
// presenter puts it into the window with GDI (SPEC §25.5). It is plain Go over a presenter, so the
// tests run it with a fake one; the Windows build gives it gdiwindow.NewPresenter.
type gdiRenderer struct {
	log        *slog.Logger
	newPresent func() gdiwindow.Presenter
	renderer   *gdiwindow.Renderer
	surfaces   map[gpu.SurfaceHandle]gpu.Surface
	announced  bool
}

func newGDIRenderer(newPresent func() gdiwindow.Presenter, log *slog.Logger) *gdiRenderer {
	return &gdiRenderer{log: log, newPresent: newPresent, surfaces: map[gpu.SurfaceHandle]gpu.Surface{}}
}

// Describe names the renderer for the log.
func (r *gdiRenderer) Describe() string { return "soft: GDI into the window" }

func (r *gdiRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		r.renderer = gdiwindow.NewRenderer(r.newPresent(), gdiwindow.Options{Background: ui.NewColor(developmentColor)})
	}
	surface, ok := r.surfaces[h]
	if !ok {
		var err error
		if surface, err = r.renderer.Attach(h); err != nil {
			return nil, err
		}
		r.surfaces[h] = surface
		if !r.announced {
			r.announced = true
			r.log.Info("GDI software renderer ready")
		}
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	if capture {
		return r.renderer.RenderCapture(surface, scene, size, 1)
	}
	return nil, r.renderer.Render(surface, scene, size, 1)
}

// Invalidate makes the next frame of the window present whole.
func (r *gdiRenderer) Invalidate(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		r.renderer.Invalidate(surface)
	}
}

func (r *gdiRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
	}
}

func (r *gdiRenderer) Close() {
	for h := range r.surfaces {
		r.Release(h)
	}
	if r.renderer != nil {
		_ = r.renderer.Close() // closing the renderer only marks it; there is nothing to report
	}
}
