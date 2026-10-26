//go:build linux || js

package engine

import (
	"image"
	"log/slog"
	"os"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/ui"
)

// glesRenderer is the engine's renderer over internal/gpu/gles: OpenGL ES through EGL on Linux,
// and WebGL 2 on the web, where the context is the canvas's and the only difference is how the
// gles renderer is made. It draws the scene of the frame over developmentColor, into the
// surface of any window, with one GL context for all of them.
type glesRenderer struct {
	// name is what GLUTTER_RENDERER calls it: "gles" or "webgl".
	name string
	// newGL is gles.NewRenderer of the OS.
	newGL  func(...gles.RendererOpts) *gles.Renderer
	redraw func(gpu.SurfaceHandle) // Config.redraw: a context that came back wants a frame
	log    *slog.Logger
	skip   bool
	cache  bool // the raster cache is on
	damage bool // frames tell the compositor what changed
	// programs keeps the program binaries between runs (nil when the cache is off).
	programs gles.ProgramCache
	renderer *gles.Renderer
	surfaces map[gpu.SurfaceHandle]gpu.Surface
	reported int // the number of skipped commands the log last said
	// announced is whether the log said that the renderer is ready.
	announced bool
	// reattached is whether the first frame on a surface made for the existing context is still to
	// come: its log line says what the frame had to make again (SPEC §24.4).
	reattached bool
}

// StartupReport says how the start went (the context, the shader programs), for the log.
func (r *glesRenderer) StartupReport() gpu.StartupReport {
	if r.renderer == nil {
		return gpu.StartupReport{}
	}
	return r.renderer.StartupReport()
}

// Describe names the renderer and its GPU: "gles: llvmpipe (LLVM 19.1.7, 256 bits)", or
// "webgl: ANGLE (...)".
func (r *glesRenderer) Describe() string {
	if r.renderer != nil {
		if name := r.renderer.DeviceName(); name != "" {
			return r.name + ": " + name
		}
	}
	return r.name
}

// newGLRenderer makes the renderer of the gles package that the OS has: newGL is its NewRenderer,
// and damage says whether frames tell the display what changed (the web has no such call).
func newGLRenderer(name string, newGL func(...gles.RendererOpts) *gles.Renderer, damage bool, cfg *Config, log *slog.Logger) renderer {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	var programs gles.ProgramCache
	if c := newShaderCache(cfg, getenv, log); c != nil {
		programs = c
	}
	return &glesRenderer{
		programs: programs,
		name:     name, newGL: newGL, redraw: cfg.redraw,
		log: log, skip: cfg.SkipUnsupported, surfaces: map[gpu.SurfaceHandle]gpu.Surface{},
		cache:  !cfg.DisableRasterCache && getenv(RasterCacheEnv) != "0",
		damage: damage && !cfg.DisableDamage && getenv(DamageEnv) != "0",
	}
}

func (r *glesRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		opts := gles.RendererOpts{
			Background: ui.NewColor(developmentColor), HasBackground: true, SkipUnsupported: r.skip,
			Damage: r.damage,
		}
		if r.cache {
			opts.RasterCacheAfter = rasterCacheAfter
		}
		opts.ProgramCache = r.programs
		if r.redraw != nil {
			opts.OnContextRestored = func() { r.redraw(h) }
		}
		r.renderer = r.newGL(opts)
	}
	surface, ok := r.surfaces[h]
	if !ok {
		var err error
		if surface, err = r.renderer.Attach(h); err != nil {
			return nil, err
		}
		r.surfaces[h] = surface
		if r.announced {
			// A new surface for the context that already draws: what the app drew before is still in
			// the context's textures and atlases.
			r.reattached = true
			r.log.Info("OpenGL ES surface attached to the existing context", "device", r.renderer.DeviceName())
		}
		if !r.announced {
			r.announced = true
			label := "OpenGL ES"
			if r.name == "webgl" {
				label = "WebGL 2"
			}
			r.log.Info(label+" renderer ready", "device", r.renderer.DeviceName())
		}
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	var img *image.RGBA
	var err error
	if capture {
		img, err = r.renderer.RenderCapture(surface, scene, size, 1)
	} else {
		err = r.renderer.Render(surface, scene, size, 1)
	}
	if err != nil {
		return nil, err
	}
	if st := r.renderer.LastFrame(); r.reattached {
		r.reattached = false
		r.log.Info("OpenGL ES first frame on the new surface", "tiles_rasterized", st.TilesRasterized, "images_uploaded", st.ImagesUploaded, "clip_masks_built", st.ClipMasksBuilt)
	}
	if st := r.renderer.LastFrame(); st.Skipped > 0 && st.Skipped != r.reported {
		r.reported = st.Skipped
		r.log.Warn("the GPU renderer left out what it cannot draw yet", "skipped", st.Skipped, "drawn", st.Drawn, "features", st.Unsupported)
	}
	return img, nil
}

func (r *glesRenderer) Invalidate(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok && r.renderer != nil {
		r.renderer.Invalidate(surface)
	}
}

func (r *glesRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
		r.log.Info("OpenGL ES surface released; the context stays")
	}
}

func (r *glesRenderer) Close() {
	for h, surface := range r.surfaces {
		surface.Destroy()
		delete(r.surfaces, h)
	}
	if r.renderer != nil {
		_ = r.renderer.Close()
		r.renderer = nil
	}
}
