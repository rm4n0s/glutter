//go:build windows

package engine

import (
	"context"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/d3d12"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gdiwindow"
	"github.com/rm4n0s/glutter/internal/platform/windows"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/winfonts"
)

// This file is what Run uses on Windows (GOOS=windows): the Win32 platform with its windows and
// message loop, the renderers (Direct3D 12 on hardware and on WARP, and the software renderer into the
// window with GDI, SPEC §9.6, §25.5), and the system font matcher (step 28.6).

// backendEnv is the environment variable that names the platform backend; Windows has one,
// "win32", which is also what an empty value means.
const backendEnv = "GLUTTER_BACKEND"

// connectDisplay makes the Win32 platform on the calling thread, which becomes the UI thread. The
// backend is empty or "win32" (Config.Backend or GLUTTER_BACKEND); wayland and x11 are
// BackendUnavailable on this OS and any other name is BackendUnknown (SPEC §25.3).
func connectDisplay(cfg *Config, getenv func(string) string, log *slog.Logger) (connection, error) {
	name := cfg.Backend
	if name == "" {
		name = getenv(backendEnv)
	}
	switch name {
	case "", "win32":
	case "wayland", "x11":
		return nil, errors.New(tagBackendUnavailable, "this build has no Wayland or X11 backend: the backend of Windows is win32", "value", name)
	default:
		return nil, errors.New(tagBackendUnknown, "the backend must be win32 on Windows", "value", name)
	}
	p, err := windows.New(windows.Options{Logger: log, Getenv: getenv})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT, a path of the
// working directory or an absolute one (a Unix path of the host is Z:\... under Wine).
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates are the renderers of Windows in the order auto tries them: d3d12 on a hardware
// adapter, d3d12 on WARP, then soft into the window with GDI (SPEC §9.6, §25.5). gles and vulkan are
// RendererUnavailable.
func rendererCandidates() []rendererCandidate {
	return windowsCandidates(newD3D12Renderer(false), newD3D12Renderer(true), newSoftWindowRenderer)
}

// newD3D12Renderer returns the constructor of the explicit renderer over a Direct3D 12 device, which it
// opens at the first frame (SPEC §9.8). warp is the candidate that uses WARP, Windows' own software
// adapter: it asks the device for the adapter "warp" (GLUTTER_D3D12_ADAPTER is not read for it, since the
// candidate is left out when that variable names an adapter). The hardware candidate never falls back to
// WARP inside the device (Options.AllowWarp is false): a device that fails, or a first frame that does not
// present, moves the chooser on to the WARP candidate, which is a try of its own and is logged.
func newD3D12Renderer(warp bool) func(cfg *Config, log *slog.Logger) renderer {
	return func(cfg *Config, log *slog.Logger) renderer {
		getenv := configGetenv(cfg)
		opts := explicitOptions(cfg, getenv, log)
		dev := d3d12.Options{Logger: log, Getenv: getenv}
		if c := newShaderCache(cfg, getenv, log); c != nil {
			dev.ShaderCache = c
		}
		if warp {
			dev.Getenv = func(k string) string {
				if k == d3d12.AdapterEnvVar {
					return "warp"
				}
				return getenv(k)
			}
		}
		return explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
			d, err := d3d12.Open(dev)
			if err != nil {
				return nil, err
			}
			return d, nil
		}, opts)
	}
}

// newSoftWindowRenderer makes the renderer that presents the soft frame with GDI.
func newSoftWindowRenderer(_ *Config, log *slog.Logger) renderer {
	return newGDIRenderer(gdiwindow.NewPresenter, log)
}

// bundledOnly is a font matcher with no system fonts: a family that is not bundled or loaded by the
// app matches nothing.
type bundledOnly struct{}

// Match returns no system font.
func (bundledOnly) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// SystemFontMatcher returns the font matcher of Windows (SPEC §25.8): the fonts that the registry
// and the font directories list, read with go-text, the generic families (sans-serif is Segoe UI,
// serif Times New Roman, monospace Consolas, cursive Segoe Script, system-ui Segoe UI Variable or
// Segoe UI), and the fallback chain by script and language. When no installed font can be read
// (a prefix of Wine with no fonts), it logs why and returns a matcher with no system fonts: a scan
// that fails would stop every paragraph.
//
// Flutter: no counterpart
func SystemFontMatcher(ctx context.Context, log *slog.Logger) text.Matcher {
	faces := windows.SystemFonts(ctx, log)
	m, err := winfonts.New(winfonts.Options{Faces: faces, Locale: windows.UserLocale()})
	if err != nil {
		if log != nil {
			log.Info("this machine has no font the app can read; only the bundled fonts and the fonts of the app are used", "err", err)
		}
		return bundledOnly{}
	}
	return m
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
	tagBackendUnknown     = "BackendUnknown"
)

// defaultShaderCacheDir is the directory of the shader cache when Config names none: glutter\shaders
// under %LOCALAPPDATA%, which is what os.UserCacheDir returns on Windows (SPEC §9.7). The variable is read
// through getenv, so a test can set it; XDG_CACHE_HOME and HOME, which Wine passes through from the host,
// are not read.
func defaultShaderCacheDir(getenv func(string) string) string {
	return shadercache.DirUnder(getenv, getenv("LocalAppData"))
}
