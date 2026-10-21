package engine

import (
	"image"
	"log/slog"
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/webgpu"
	"github.com/rm4n0s/glutter/ui"
)

// newWebGPURenderer makes the WebGPU renderer: the explicit renderer over the browser's WebGPU
// device, which connectDisplay got at startup. It has no damage to report: the browser presents the
// whole canvas. A capture (the proof screenshot) finishes in a later task, and reaches the engine through
// Config.captured; a snapshot is drawn by the software renderer (SPEC §9.3).
func newWebGPURenderer(cfg *Config, log *slog.Logger) renderer {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	opts := explicit.RendererOpts{
		Background: ui.NewColor(developmentColor), HasBackground: true, Log: log, SkipUnsupported: cfg.SkipUnsupported,
		OnCapture: func(img *image.RGBA, err error) {
			if cfg.captured != nil {
				cfg.captured(img, err)
			}
		},
	}
	if !cfg.DisableRasterCache && getenv(RasterCacheEnv) != "0" {
		opts.RasterCacheAfter = rasterCacheAfter
	}
	return &webgpuRenderer{Renderer: webgpu.NewRenderer(func(h gpu.SurfaceHandle) (explicit.Device, error) {
		if cfg.webgpuOpen == nil {
			return nil, errors.New("WebGPUUnavailable", "the engine did not start WebGPU")
		}
		return cfg.webgpuOpen(h)
	}, opts), log: log}
}

// webgpuRenderer is the engine's renderer over WebGPU: the explicit renderer, which names itself
// in the log when its first frame is presented.
type webgpuRenderer struct {
	*webgpu.Renderer
	log       *slog.Logger
	announced bool
}

// Draw draws a frame and, the first time, tells the log which adapter draws.
func (r *webgpuRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	img, err := r.Renderer.Draw(h, width, height, scene, capture)
	if err == nil && !r.announced && r.Describe() != "" {
		r.announced = true
		r.log.Info("WebGPU renderer ready", "device", r.Describe())
	}
	return img, err
}

// closeWebGPU destroys the device that the web's connectDisplay got, when Run ends before a renderer
// opened it (the platform could not start).
func (c *Config) closeWebGPU() {
	if c.webgpuClose != nil {
		c.webgpuClose()
	}
}
