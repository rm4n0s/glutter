//go:build android

package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/nativewindow"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/platform/android"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/androidfonts"
	"github.com/rm4n0s/glutter/ui"
)

// This file is what Run uses on Android (GOOS=android): the platform of the activity's view, Vulkan,
// OpenGL ES and then the software renderer into the window, and the system font matcher of the
// device's fonts.xml (SPEC §6.3, §24).

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

// rendererCandidates is the list of Android in the order auto tries them (SPEC §24.4): vulkan, then
// gles, then soft, drawing into the window's CPU buffers.
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{
		{Name: "vulkan", New: newVulkanRenderer}, {Name: "gles", New: newGLESRenderer}, {Name: "soft", New: newNativeWindowRenderer},
	}
}

// newVulkanRenderer makes the explicit renderer over Vulkan on the ANativeWindow. `auto` chooses it
// only on Android 10 or later and on a driver that is not on the deny list; Vulkan asked for by name
// is tried whatever the device (SPEC §24.4). The frame is drawn whole into its own texture and blitted
// into the swapchain image (rotated by the surface's pre-transform), so the damage only tells the
// compositor what changed.
func newVulkanRenderer(cfg *Config, log *slog.Logger) renderer {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	opts := explicit.RendererOpts{
		Background: ui.NewColor(developmentColor), HasBackground: true, Log: log,
		Damage: !cfg.DisableDamage && getenv(DamageEnv) != "0", SkipUnsupported: cfg.SkipUnsupported,
	}
	if !cfg.DisableRasterCache && getenv(RasterCacheEnv) != "0" {
		opts.RasterCacheAfter = rasterCacheAfter
	}
	return explicit.NewRenderer(func(h gpu.SurfaceHandle) (explicit.Device, error) {
		if err := checkAndroidVulkanSDK(android.SDKInt(), cfg.rendererNamed); err != nil {
			return nil, err
		}
		return vulkan.Open(vulkan.Options{
			Logger: log, Getenv: getenv, AllowCPU: cfg.rendererNamed, DenyKnownBad: !cfg.rendererNamed, Surface: h,
		})
	}, opts)
}

// newGLESRenderer makes the OpenGL ES renderer of Android, over EGL on the ANativeWindow. It
// reports no damage: the buffers that Android swaps are not preserved, so every frame is drawn
// whole.
func newGLESRenderer(cfg *Config, log *slog.Logger) renderer {
	return newGLRenderer("gles", gles.NewRenderer, false, cfg, log)
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

// SystemFontMatcher returns the font matcher of the device: the families, aliases and fallback chain of
// /system/etc/fonts.xml read as Skia's Android font manager reads them (SPEC §24.9). When the files
// cannot be read, it logs why and returns nil, and the text.Collection scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(_ context.Context, log *slog.Logger) text.Matcher {
	opts := androidfonts.DefaultOptions()
	opts.Logger = log
	m, err := androidfonts.New(opts)
	if err != nil {
		if log != nil {
			log.Warn("the system fonts of Android could not be read; the font directories are scanned", "err", err)
		}
		return nil
	}
	return m
}
