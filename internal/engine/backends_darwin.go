//go:build darwin && !ios

package engine

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on macOS (GOOS=darwin, not iOS): the AppKit platform, the renderers
// (Metal, and the software renderer into IOSurfaces, SPEC §9.6, §29.5) and the system font matcher
// (Core Text, SPEC §29.8). Until step 33.2 it has none of them, so Run fails with BackendUnavailable;
// step 33.2 adds the platform and the soft renderer, 33.6 the fonts and 33.8 Metal.

// connectDisplay fails: this build has no platform backend yet (step 33.2 adds AppKit).
func connectDisplay(*Config, func(string) string, *slog.Logger) (connection, error) {
	return nil, errors.New(tagBackendUnavailable, "this build has no macOS platform backend yet")
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT, a path of the
// working directory or an absolute one.
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is empty: Run fails before it needs a renderer (steps 33.2 and 33.8 add them).
func rendererCandidates() []rendererCandidate { return nil }

// SystemFontMatcher returns nil: the text.Collection scans the font directories (step 33.6 adds the
// Core Text matcher).
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
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
