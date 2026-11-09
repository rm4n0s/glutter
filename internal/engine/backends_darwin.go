//go:build darwin && !ios

package engine

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/platform/macos"
	"github.com/rm4n0s/glutter/internal/text"
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
	p, err := macos.New(macos.Options{Logger: log, Getenv: getenv, Accessory: getenv(accessoryEnv) == "1"})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT, a path of the
// working directory or an absolute one.
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates are the renderers of macOS in the order auto tries them: soft, drawn into
// IOSurfaces that the window's layer shows, for now (step 33.8 puts Metal before it). gles, vulkan and
// metal are RendererUnavailable until then (SPEC §29.5).
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{{Name: "soft", New: newSoftWindowRenderer}}
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

// SystemFontMatcher returns nil: the text.Collection scans the font directories (step 33.6 adds the
// Core Text matcher).
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }

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
