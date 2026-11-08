package engine

import (
	"image"
	"log/slog"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/shmwindow"
	"github.com/rm4n0s/glutter/ui"
)

// shmRenderer is the software renderer of the Linux desktop (SPEC §9.6): soft draws the frame
// into the buffers that the window gives (wl_shm on Wayland), and only the damaged rectangles
// are drawn and presented. It is plain Go over a function that finds the gpu.PixelTarget of a
// window, so the tests run it with a fake one; Run gives it the windows of the handler.
type shmRenderer struct {
	log       *slog.Logger
	targets   func(gpu.SurfaceHandle) (gpu.PixelTarget, bool)
	renderer  *shmwindow.Renderer
	surfaces  map[gpu.SurfaceHandle]gpu.Surface
	kind      gpu.SurfaceKind
	announced bool
	// target is the first window's target, which may say how it sends its pixels (pixelPather).
	target gpu.PixelTarget
}

// pixelPather is a gpu.PixelTarget that can say how its pixels reach the display server: "MIT-SHM"
// or "XPutImage" on X11. The answer is known once the first buffer was asked for.
type pixelPather interface{ PixelPath() string }

func newSHMRenderer(targets func(gpu.SurfaceHandle) (gpu.PixelTarget, bool), log *slog.Logger) *shmRenderer {
	return &shmRenderer{log: log, targets: targets, surfaces: map[gpu.SurfaceHandle]gpu.Surface{}}
}

// Describe names the renderer for the log.
func (r *shmRenderer) Describe() string {
	switch r.kind {
	case gpu.SurfaceKindWayland:
		return "soft: shared memory (wl_shm)"
	case gpu.SurfaceKindX11:
		if p, ok := r.target.(pixelPather); ok {
			if path := p.PixelPath(); path != "" {
				return "soft: " + path + " (X11)"
			}
		}
		return "soft: X11 pixels"
	}
	return "soft: shared memory"
}

func (r *shmRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		r.renderer = shmwindow.NewRenderer(r.targets, shmwindow.Options{Background: ui.NewColor(developmentColor)})
	}
	surface, ok := r.surfaces[h]
	if !ok {
		var err error
		if surface, err = r.renderer.Attach(h); err != nil {
			return nil, err
		}
		r.surfaces[h] = surface
		r.kind = h.Kind
		if r.target == nil {
			r.target, _ = r.targets(h)
		}
		if !r.announced {
			r.announced = true
			r.log.Info("software renderer ready", "presents", r.Describe())
		}
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	return r.renderer.RenderCapture(surface, scene, size, 1, capture)
}

// Invalidate makes the next frame of the window draw and present whole.
func (r *shmRenderer) Invalidate(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		r.renderer.Invalidate(surface)
	}
}

func (r *shmRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
	}
}

func (r *shmRenderer) Close() {
	for h := range r.surfaces {
		r.Release(h)
	}
	if r.renderer != nil {
		_ = r.renderer.Close() // closing the renderer only marks it; there is nothing to report
	}
}
