//go:build js

package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/canvas2d"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/webgpu"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/ui"
)

// This file is what Run uses on the web (GOOS=js): the page's canvas for the platform, WebGPU
// through the explicit renderer, WebGL 2 through the gles renderer and then the software renderer
// into a 2D canvas for the renderer, and only the bundled fonts (SPEC §9.6, §23).

// connectDisplay makes the web platform: it finds the host element of the page. The error
// of the platform comes back unchanged (BackendUnavailable outside a page,
// HostElementNotFound, WebConfigInvalid).
//
// Before the platform exists, and so before it installs its event listeners, it also gets the
// browser's WebGPU device when the renderer can be WebGPU (auto, or asked for by name): the one
// place where the UI goroutine waits for promises (SPEC §23.3). Why there is no device is kept for
// the renderer, which returns it from its first frame, so that auto then falls to WebGL 2 and a
// renderer asked for by name ends Run with it.
func connectDisplay(cfg *Config, getenv func(string) string, log *slog.Logger) (connection, error) {
	tuneGC()
	want := cfg.Renderer
	if want == "" {
		want = getenv(RendererEnv)
	}
	if want == "" || want == rendererAuto || want == "webgpu" {
		prov, err := webgpu.Acquire(context.Background(), webgpu.Options{
			Logger:        log,
			Post:          func(fn func()) { cfg.postToUI(fn) },
			OnDeviceReady: func(h gpu.SurfaceHandle) { cfg.redrawWindow(h) },
		})
		if err != nil {
			cfg.webgpuOpen = func(gpu.SurfaceHandle) (explicit.Device, error) { return nil, err }
		} else {
			cfg.webgpuOpen, cfg.webgpuClose = prov.Open, prov.Close
		}
	}
	p, err := web.New(web.Options{Logger: log})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// postToUI runs fn on the UI thread once Run has made the platform; before that there is no other
// thread to run it on.
func (c *Config) postToUI(fn func()) {
	if c.post != nil {
		c.post(fn)
		return
	}
	fn()
}

// redrawWindow asks for a frame of the window, when Run has set the hook.
func (c *Config) redrawWindow(h gpu.SurfaceHandle) {
	if c.redraw != nil {
		c.redraw(h)
	}
}

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is the web's list in the order auto tries them (SPEC §9.6): webgl, the gles
// renderer on WebGL 2, then soft, drawing into a 2D canvas. The chooser puts a new canvas in
// place before each try after the first, because a canvas keeps the first kind of context it was
// asked for.
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{
		{Name: "webgpu", New: newWebGPURenderer, SwitchOn: webgpuLastResort}, {Name: "webgl", New: newWebGLRenderer}, {Name: "soft", New: newCanvasRenderer},
	}
}

// newWebGLRenderer makes the gles renderer over WebGL 2. It has no damage to report: the browser
// presents the whole canvas.
func newWebGLRenderer(cfg *Config, log *slog.Logger) renderer {
	return newGLRenderer("webgl", gles.NewRenderer, false, cfg, log)
}

// canvasRenderer is the engine's renderer for the web's last resort: soft draws the frame
// and putImageData puts it into the window's canvas.
type canvasRenderer struct {
	log       *slog.Logger
	renderer  *canvas2d.Renderer
	surfaces  map[gpu.SurfaceHandle]gpu.Surface
	announced bool
}

func newCanvasRenderer(_ *Config, log *slog.Logger) renderer {
	return &canvasRenderer{
		log:      log,
		renderer: canvas2d.New(canvas2d.Options{Background: ui.NewColor(developmentColor)}),
		surfaces: map[gpu.SurfaceHandle]gpu.Surface{},
	}
}

// Describe names the renderer for the log.
func (r *canvasRenderer) Describe() string { return "soft: 2D canvas (putImageData)" }

func (r *canvasRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	surface, ok := r.surfaces[h]
	if !ok {
		var err error
		if surface, err = r.renderer.Attach(h); err != nil {
			return nil, err
		}
		r.surfaces[h] = surface
		if !r.announced {
			r.announced = true
			r.log.Info("2D canvas renderer ready")
		}
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	if capture {
		return r.renderer.RenderCapture(surface, scene, size, 1)
	}
	return nil, r.renderer.Render(surface, scene, size, 1)
}

// Invalidate does nothing: every frame is drawn whole.
func (r *canvasRenderer) Invalidate(gpu.SurfaceHandle) {}

func (r *canvasRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
	}
}

func (r *canvasRenderer) Close() {
	for h := range r.surfaces {
		r.Release(h)
	}
	_ = r.renderer.Close() // closing the renderer only marks it; there is nothing to report
}

// bundledOnly is the font matcher of the web: there are no system fonts, so a family that
// is not bundled or loaded by the app matches nothing, and a character with no font draws as
// a missing-glyph box (SPEC §23.6).
type bundledOnly struct{}

// Match returns no system font.
func (bundledOnly) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// SystemFontMatcher returns the matcher that has no system fonts: the text.Collection then
// uses the bundled fonts and the fonts the app loads, and never scans a directory
// (SPEC §23.6).
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return bundledOnly{} }

// Web GC settings (SPEC §23.1, "Decided in step 21.7"). A WebAssembly program has one thread, so the garbage
// collector's work is taken from the frames; a build of a screen allocates megabytes (a page of the gallery: 5.7 MB
// in 57,000 objects), and with the default of collecting when the heap has doubled, a click on the gallery's
// navigation spent a quarter of its time in the collector. The collector runs when the heap has grown to five times
// what was live after the last one, unless the program set GOGC itself, and a soft limit keeps that from using
// more memory than a page should.
const (
	webGCPercent   = 400
	webMemoryLimit = 768 << 20
)

// tuneGC sets the web's GC percent and memory limit, except where the page's query gave GOGC or GOMEMLIMIT.
func tuneGC() {
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(webGCPercent)
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(webMemoryLimit)
	}
}
