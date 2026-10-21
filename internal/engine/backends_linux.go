//go:build linux

package engine

import (
	"context"
	"image"
	"log/slog"
	"os"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/vulkan"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/internal/platform/linux/fontconfig"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/platform/linux/x11"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/ui"
)

// This file chooses what Run uses on Linux: Wayland or X11 for the platform, OpenGL
// ES for the renderer, fontconfig for the fonts (SPEC §6.3 rules 1 and 5).

// connectDisplay connects to the display server: Wayland or X11, as linux.Select
// chooses.
func connectDisplay(cfg *Config, getenv func(string) string, log *slog.Logger) (connection, error) {
	conn, _, err := linux.Select(linux.Selection{
		Backend: cfg.Backend,
		Getenv:  getenv,
		Wayland: cfg.waylandConnector(),
		X11:     cfg.x11Connector(),
		Logger:  log,
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// processStart is when the process started, if the system says.
func processStart() (time.Time, bool) { return linux.ProcessStart() }

// rendererCandidates are the renderers of Linux in the order auto tries them: vulkan, then gles
// (SPEC §9.6). A Vulkan whose only device is a CPU (lavapipe) does not count: it is used only
// when asked for by name, so auto then uses GLES.
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{{Name: "vulkan", New: newVulkanRenderer}, {Name: "gles", New: newGLESRenderer}}
}

// newVulkanRenderer makes the explicit renderer over a Vulkan device, which it opens at the
// first frame with the handle of the first window. A device of type CPU (lavapipe) is
// used only when Vulkan was asked for by name (SPEC §9.4).
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
		return vulkan.Open(vulkan.Options{Logger: log, Getenv: getenv, AllowCPU: cfg.rendererNamed, Surface: h})
	}, opts)
}

// waylandConnector returns the function that connects the Wayland backend.
func (c *Config) waylandConnector() func() (linux.Connection, error) {
	if c.wayland != nil {
		return asLinux(c.wayland)
	}
	return func() (linux.Connection, error) {
		p, err := wayland.New("", wayland.Options{ShmFill: c.ShmFill})
		if err != nil {
			return nil, err
		}
		return p, nil
	}
}

// x11Connector returns the function that connects the X11 backend.
func (c *Config) x11Connector() func() (linux.Connection, error) {
	if c.x11 != nil {
		return asLinux(c.x11)
	}
	return func() (linux.Connection, error) {
		if c.ShmFill {
			return nil, errors.New("ShmFillUnsupported", "the shared-memory mode is for Wayland only")
		}
		p, err := x11.New("")
		if err != nil {
			return nil, err
		}
		return p, nil
	}
}

// asLinux turns a connector of the engine into one of linux.Selection.
func asLinux(fn func() (connection, error)) func() (linux.Connection, error) {
	return func() (linux.Connection, error) {
		c, err := fn()
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}

// SystemFontMatcher returns the font matcher of the machine: libfontconfig, so that
// the family names, the user's UI font, and the emoji, CJK and per-language
// preferences follow the fontconfig settings, as Flutter's Linux engine does. It
// returns nil, and logs why, when fontconfig is not there; the text.Collection then
// scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(ctx context.Context, log *slog.Logger) text.Matcher {
	if log == nil {
		log = slog.Default()
	}
	m, err := openFontconfig(ctx)
	if err != nil {
		log.Info("fontconfig is not available; scanning the font directories", "error", err)
		return nil
	}
	return m
}

// fontconfigMatcher is a text.Matcher over libfontconfig.
type fontconfigMatcher struct {
	cfg *fontconfig.Config
}

// openFontconfig loads libfontconfig and the user's configuration. It fails like
// fontconfig.Open: with FontconfigLoadFailed when the library is not there.
func openFontconfig(ctx context.Context) (*fontconfigMatcher, error) {
	cfg, err := fontconfig.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &fontconfigMatcher{cfg: cfg}, nil
}

// Match implements text.Matcher.
func (m *fontconfigMatcher) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]text.FontFile, error) {
	files, err := m.cfg.Match(ctx, family, weight, italic, lang)
	if err != nil {
		return nil, err
	}
	out := make([]text.FontFile, len(files))
	for i, f := range files {
		out[i] = text.FontFile(f)
	}
	return out, nil
}

// Close releases the configuration. The matcher must not be used after.
func (m *fontconfigMatcher) Close() { m.cfg.Close() }

// glesRenderer is the OpenGL ES renderer of the engine: it draws the scene of the
// frame over developmentColor, into the surface of any window, with one GL context
// for all of them.
type glesRenderer struct {
	log      *slog.Logger
	skip     bool
	cache    bool // the raster cache is on
	damage   bool // frames tell the compositor what changed
	renderer *gles.Renderer
	surfaces map[gpu.SurfaceHandle]gpu.Surface
	reported int // the number of skipped commands the log last said
	// announced is whether the log said that the renderer is ready.
	announced bool
}

// Describe names the renderer and its GPU: "gles: llvmpipe (LLVM 19.1.7, 256 bits)".
func (r *glesRenderer) Describe() string {
	if r.renderer != nil {
		if name := r.renderer.DeviceName(); name != "" {
			return "gles: " + name
		}
	}
	return "gles"
}

func newGLESRenderer(cfg *Config, log *slog.Logger) renderer {
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	return &glesRenderer{
		log: log, skip: cfg.SkipUnsupported, surfaces: map[gpu.SurfaceHandle]gpu.Surface{},
		cache:  !cfg.DisableRasterCache && getenv(RasterCacheEnv) != "0",
		damage: !cfg.DisableDamage && getenv(DamageEnv) != "0",
	}
}

// rasterCacheAfter is how many frames in a row a retained layer must be in the scene
// to be drawn from a texture: three, as Flutter's raster cache does.
const rasterCacheAfter = 3

func (r *glesRenderer) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if r.renderer == nil {
		opts := gles.RendererOpts{
			Background: ui.NewColor(developmentColor), HasBackground: true, SkipUnsupported: r.skip,
			Damage: r.damage,
		}
		if r.cache {
			opts.RasterCacheAfter = rasterCacheAfter
		}
		r.renderer = gles.NewRenderer(opts)
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
			r.log.Info("OpenGL ES renderer ready", "device", r.renderer.DeviceName())
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
