//go:build !linux && !js && !windows

package engine

import (
	"context"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on an operating system that has no backend yet: no platform backend yet, so Run fails.

// connectDisplay fails: this build has no platform backend.
func connectDisplay(*Config, func(string) string, *slog.Logger) (connection, error) {
	return nil, errors.New(tagBackendUnavailable, "this build has no platform backend for this operating system")
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT.
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is empty: Run fails before it needs a renderer.
func rendererCandidates() []rendererCandidate { return nil }

// SystemFontMatcher returns nil: the text.Collection scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
)

// defaultShaderCacheDir is the directory of the shader cache when Config names none: the environment's
// (SPEC §9.7).
func defaultShaderCacheDir(getenv func(string) string) string { return shadercache.Dir(getenv) }
