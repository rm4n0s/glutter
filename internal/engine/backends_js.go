//go:build js

package engine

import (
	"context"
	"image"
	"log/slog"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/canvas2d"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/ui"
)

// This file is what Run uses on the web (GOOS=js): the page's canvas for the platform, the
// software renderer into a 2D canvas for the renderer (the other web renderers come with
// steps 21.4 and 21.5), and only the bundled fonts (SPEC §9.6, §23).

// connectDisplay makes the web platform: it finds the host element of the page. The error
// of the platform comes back unchanged (BackendUnavailable outside a page,
// HostElementNotFound, WebConfigInvalid).
func connectDisplay(_ *Config, _ func(string) string, log *slog.Logger) (connection, error) {
	p, err := web.New(web.Options{Logger: log})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is the web's list in the order auto tries them (SPEC §9.6). For
// now it is soft, drawing into a 2D canvas.
func rendererCandidates() []rendererCandidate {
	// TODO(step 21.5, 21.4): webgpu, then webgl, before soft.
	return []rendererCandidate{{Name: "soft", New: newCanvasRenderer}}
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
