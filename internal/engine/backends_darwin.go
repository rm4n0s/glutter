//go:build darwin && !ios

package engine

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/platform/macos"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/darwinfonts"
)

// This file is what Run uses on macOS (GOOS=darwin, not iOS): the AppKit platform, the renderers (the
// software renderer into IOSurfaces from step 33.2, Metal from step 33.8, SPEC §9.6, §29.5) and the
// system font matcher (Core Text from step 33.6, SPEC §29.8).

// backendEnv is the environment variable that names the platform backend; macOS has one, "appkit",
// which is also what an empty value means.
const backendEnv = "GLUTTER_BACKEND"

// accessoryEnv makes the app an accessory app (no Dock icon, no menu bar, never activated): the display
// tests set it so that they never take the focus from the user's apps (SPEC §29.13).
const accessoryEnv = "GLUTTER_MACOS_ACCESSORY"

// connectDisplay makes the AppKit platform on the calling thread, which must be the main thread and
// becomes the UI thread. The backend is empty or "appkit" (Config.Backend or GLUTTER_BACKEND); wayland and
// x11 are BackendUnavailable on this OS and any other name is BackendUnknown (SPEC §29.3).
func connectDisplay(cfg *Config, getenv func(string) string, log *slog.Logger) (connection, error) {
	name := cfg.Backend
	if name == "" {
		name = getenv(backendEnv)
	}
	switch name {
	case "", "appkit":
	case "wayland", "x11":
		return nil, errors.New(tagBackendUnavailable, "this build has no Wayland or X11 backend: the backend of macOS is appkit", "value", name)
	default:
		return nil, errors.New(tagBackendUnknown, "the backend must be appkit on macOS", "value", name)
	}
	p, err := macos.New(macos.Options{Logger: log, Getenv: getenv, Accessory: getenv(accessoryEnv) == "1", QuitOnLastWindowClosed: cfg.QuitOnLastWindowClosed})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT, a path of the
// working directory or an absolute one.
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return macos.ProcessStart() }

// rendererCandidates are the renderers of macOS in the order auto tries them: metal, then soft, drawn into
// IOSurfaces that the window's layer shows. gles and vulkan are RendererUnavailable (SPEC §29.5).
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{
		{Name: "metal", New: newMetalRenderer},
		{Name: "soft", New: newSoftWindowRenderer},
	}
}

// newMetalRenderer makes the explicit renderer over a Metal device, which it opens at the first frame (SPEC
// §9.8, §29.5). The device is the default one, or the low-power one with GLUTTER_METAL_DEVICE=lowpower; the
// binary archive of its pipelines is in the shader cache directory. A Mac without a Metal device fails the
// first frame with MetalDeviceUnavailable, which moves auto on to soft.
func newMetalRenderer(cfg *Config, log *slog.Logger) renderer {
	getenv := configGetenv(cfg)
	opts := explicitOptions(cfg, getenv, log)
	dev := metal.Options{Logger: log, Getenv: getenv, Readback: getenv(ScreenshotEnv) != ""}
	if c := newShaderCache(cfg, getenv, log); c != nil {
		dev.ShaderCacheDir = c.Dir()
	}
	return explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := metal.Open(dev)
		if err != nil {
			return nil, err
		}
		return d, nil
	}, opts)
}

// newSoftWindowRenderer makes the software renderer over the IOSurfaces of the windows.
func newSoftWindowRenderer(cfg *Config, log *slog.Logger) renderer {
	return newSHMRenderer(func(h gpu.SurfaceHandle) (gpu.PixelTarget, bool) {
		if cfg.pixelTarget == nil {
			return nil, false
		}
		return cfg.pixelTarget(h)
	}, log)
}

// SystemFontMatcher returns the font matcher of macOS (SPEC §29.8): the fonts that Core Text lists, read
// with go-text, the system UI font (San Francisco) by the file that Core Text names, the generic
// families (sans-serif is San Francisco, serif Times, monospace Menlo) and the font that Core Text
// chooses for a character that the font of the request lacks. When Core Text or its fonts cannot be
// read it logs why and returns nil, so that the text.Collection scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(ctx context.Context, log *slog.Logger) text.Matcher {
	opts, err := macos.FontOptions(ctx, log)
	if err == nil {
		var m *darwinfonts.Matcher
		if m, err = darwinfonts.New(opts); err == nil {
			return m
		}
	}
	if log != nil {
		log.Info("the fonts of this Mac could not be read through Core Text; the font directories are scanned instead", "err", err)
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
	tagBackendUnknown     = "BackendUnknown"
)

// defaultShaderCacheDir is the directory of the shader cache when Config names none: glutter/shaders
// under ~/Library/Caches, the user's cache directory on macOS (SPEC §9.7, §29.5). It is empty when the
// cache is off or there is no HOME.
func defaultShaderCacheDir(getenv func(string) string) string {
	home := getenv("HOME")
	if home == "" {
		return ""
	}
	return shadercache.DirUnder(getenv, filepath.Join(home, "Library", "Caches"))
}
