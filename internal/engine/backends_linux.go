//go:build linux && !android

package engine

import (
	"context"
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

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT.
func screenshotPath(value string, _ func(string) string) string { return value }

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
	var pipelines vulkan.PipelineCache
	if c := newShaderCache(cfg, getenv, log); c != nil {
		pipelines = c
	}
	return explicit.NewRenderer(func(h gpu.SurfaceHandle) (explicit.Device, error) {
		return vulkan.Open(vulkan.Options{Logger: log, Getenv: getenv, AllowCPU: cfg.rendererNamed, Surface: h, PipelineCache: pipelines})
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
			return nil, errors.New(tagShmFillUnsupported, "the shared-memory mode is for Wayland only")
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
		log.Info("fontconfig is not available; scanning the font directories", "err", err)
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

// Aliases implements text.Aliaser.
func (m *fontconfigMatcher) Aliases(ctx context.Context, family string) ([]string, error) {
	return m.cfg.Aliases(ctx, family)
}

// Families implements text.Lister.
func (m *fontconfigMatcher) Families(ctx context.Context) ([]string, error) {
	return m.cfg.Families(ctx)
}

// Close releases the configuration. The matcher must not be used after.
func (m *fontconfigMatcher) Close() { m.cfg.Close() }

// newGLESRenderer makes the OpenGL ES renderer of Linux, over EGL.
func newGLESRenderer(cfg *Config, log *slog.Logger) renderer {
	return newGLRenderer("gles", gles.NewRenderer, true, cfg, log)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShmFillUnsupported = "ShmFillUnsupported"
)
