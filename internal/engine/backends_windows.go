//go:build windows

package engine

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gdiwindow"
	"github.com/rm4n0s/glutter/internal/platform/windows"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on Windows (GOOS=windows): the Win32 platform with its windows and
// message loop, and the software renderer into the window with GDI (SPEC §6.3, §25). The Direct3D
// 12 renderer comes with step 28.10, the system font matcher with 28.6.

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

// rendererCandidates are the renderers of Windows in the order auto tries them: soft into the
// window with GDI until step 28.10 adds Direct3D 12, which asked for by name is RendererUnavailable
// (SPEC §25.5).
func rendererCandidates() []rendererCandidate {
	return []rendererCandidate{{Name: "soft", New: newSoftWindowRenderer}}
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

// SystemFontMatcher returns nil when the machine has a font directory (%WINDIR%\Fonts), so that the
// text.Collection scans it, and a matcher with no system fonts when it has none: the prefix of Wine
// has no such directory, and a scan that fails would stop every paragraph. The matcher of the system
// fonts of Windows comes with step 28.6 (TODO(step 28.6): the families, the aliases and the fallback
// chain of Windows).
//
// Flutter: no counterpart
func SystemFontMatcher(_ context.Context, log *slog.Logger) text.Matcher {
	if root := os.Getenv("SYSTEMROOT"); root != "" { // the variable that go-text's font scan reads
		if st, err := os.Stat(filepath.Join(root, "Fonts")); err == nil && st.IsDir() {
			return nil
		}
	}
	if log != nil {
		log.Info("this machine has no font directory; only the bundled fonts and the fonts of the app are used")
	}
	return bundledOnly{}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
	tagBackendUnknown     = "BackendUnknown"
)
