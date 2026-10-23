//go:build android

package engine

import (
	"context"
	"image"
	"log/slog"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/nativewindow"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/ui"
)

// This file is what Run uses on Android (GOOS=android): the platform of the activity's view, the
// software renderer into the window for now (steps 25.4 and 25.5 add GLES and Vulkan), and no
// system font matcher until step 25.10 (SPEC §6.3, §24).

// connectDisplay makes the platform of the process. It must run on the UI thread, which is where
// the looper is prepared (SPEC §24.3). It fails with LooperUnavailable.
func connectDisplay(_ *Config, _ func(string) string, log *slog.Logger) (connection, error) {
	p, err := android.New(android.Options{Logger: log})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// screenshotPath gives the self-capture of Android a file in the app's files directory when the
// value is "1" (SPEC §24.13).
func screenshotPath(value string, getenv func(string) string) string {
	return androidScreenshotPath(value, getenv)
}

// rendererCandidates is the list of Android in the order auto tries them (SPEC §24.4): soft for
// now, drawing into the window's CPU buffers. Steps 25.4 and 25.5 put gles and vulkan before it.
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{{Name: "soft", New: newNativeWindowRenderer}}
}

// nativeWindowRenderer is the engine's renderer of last resort on Android: soft draws the frame
// and it is copied into the window's buffer.
type nativeWindowRenderer struct {
	log       *slog.Logger
	renderer  *nativewindow.Renderer
	surfaces  map[gpu.SurfaceHandle]gpu.Surface
	announced bool
}

func newNativeWindowRenderer(_ *Config, log *slog.Logger) renderer {
	return &nativeWindowRenderer{log: log, surfaces: map[gpu.SurfaceHandle]gpu.Surface{}}
}

// Describe names the renderer for the log.
func (r *nativeWindowRenderer) Describe() string { return "soft: ANativeWindow buffers" }

func (r *nativeWindowRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		presenter, err := nativewindow.NewPresenter()
		if err != nil {
			return nil, err
		}
		r.renderer = nativewindow.NewRenderer(presenter, nativewindow.Options{Background: ui.NewColor(developmentColor)})
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
			r.log.Info("ANativeWindow software renderer ready")
		}
	}
	size := ui.Size{Width: float64(width), Height: float64(height)}
	if capture {
		return r.renderer.RenderCapture(surface, scene, size, 1)
	}
	return nil, r.renderer.Render(surface, scene, size, 1)
}

// Invalidate does nothing: every frame is drawn whole.
func (r *nativeWindowRenderer) Invalidate(gpu.SurfaceHandle) {}

func (r *nativeWindowRenderer) Release(h gpu.SurfaceHandle) {
	if surface, ok := r.surfaces[h]; ok {
		surface.Destroy()
		delete(r.surfaces, h)
	}
}

func (r *nativeWindowRenderer) Close() {
	for h := range r.surfaces {
		r.Release(h)
	}
	if r.renderer != nil {
		_ = r.renderer.Close() // closing the renderer only marks it; there is nothing to report
	}
}

// SystemFontMatcher returns nil: the text.Collection scans the font directories until step 25.10.
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }
