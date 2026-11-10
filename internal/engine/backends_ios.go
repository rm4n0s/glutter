//go:build ios

package engine

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/metal"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/platform/darwin"
	"github.com/rm4n0s/glutter/internal/platform/ios"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/darwinfonts"
)

// This file is what Run uses on iOS (GOOS=ios): the UIKit platform, the software renderer into IOSurfaces that the view's layer shows
// (step 35.2; Metal comes with step 35.3, SPEC §9.6, §30.5) and the system font matcher (Core Text, SPEC §30.8).

// connectDisplay makes the UIKit platform. It runs on the main thread, which becomes the UI thread when Run starts UIApplicationMain;
// iOS has one backend, so Config.Backend and GLUTTER_BACKEND mean nothing.
func connectDisplay(cfg *Config, getenv func(string) string, log *slog.Logger) (connection, error) {
	p, err := ios.New(ios.Options{Logger: log, Getenv: getenv})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT, which `glutter-ios shot` makes a path in the app's
// container (the sandbox of an iOS app can write nowhere else).
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates are the renderers of iOS in the order auto tries them: metal, then soft, drawn into IOSurfaces that the window's
// layer shows. gles and vulkan are RendererUnavailable (SPEC §30.5).
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{
		{Name: "metal", New: newMetalRenderer},
		{Name: "soft", New: newSoftWindowRenderer},
	}
}

// newMetalRenderer makes the explicit renderer over a Metal device, which it opens at the first frame (SPEC §9.8, §30.5): the default
// device of the phone or of the Simulator, with the binary archive of its pipelines in the shader cache directory. A device without
// Metal fails the first frame with MetalDeviceUnavailable, which moves auto on to soft.
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

// SystemFontMatcher returns the font matcher of iOS: the fonts that Core Text lists, read with go-text, the system UI font (San
// Francisco) by the file that Core Text names, the generic families and the font that Core Text chooses for a character that the
// font of the request lacks. When Core Text or its fonts cannot be read it logs why and returns nil, so that the text.Collection
// scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(ctx context.Context, log *slog.Logger) text.Matcher {
	opts, err := darwin.FontOptions(ctx, log)
	if err == nil {
		var m *darwinfonts.Matcher
		if m, err = darwinfonts.New(opts); err == nil {
			return m
		}
	}
	if log != nil {
		log.Info("the fonts of this device could not be read through Core Text; the font directories are scanned instead", "err", err)
	}
	return nil
}

// defaultShaderCacheDir is the directory of the shader cache when Config names none: Library/Caches/glutter/shaders of the app's
// container, the cache directory of an iOS app (SPEC §9.7, §30.5). It is empty when the cache is off or there is no HOME.
func defaultShaderCacheDir(getenv func(string) string) string {
	home := getenv("HOME")
	if home == "" {
		return ""
	}
	return shadercache.DirUnder(getenv, filepath.Join(home, "Library", "Caches"))
}
