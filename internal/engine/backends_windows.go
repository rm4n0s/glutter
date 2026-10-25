//go:build windows

package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on Windows (GOOS=windows) until the platform and the renderer exist:
// the Win32 platform comes with step 28.2, the system font matcher with 28.6 and the Direct3D 12
// renderer with 28.10 (SPEC §6.3, §25). Until then Run fails with BackendUnavailable, there are
// no renderer candidates and no system font matcher.

// connectDisplay fails: this build has no platform backend yet (TODO(step 28.2): the Win32 platform).
func connectDisplay(*Config, func(string) string, *slog.Logger) (connection, error) {
	return nil, errors.New(tagBackendUnavailable, "the Windows platform backend is not there yet")
}

// screenshotPath is the file of the self-capture: the value of GLUTTER_SCREENSHOT.
func screenshotPath(value string, _ func(string) string) string { return value }

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is empty: Run fails before it needs a renderer (TODO(step 28.10): Direct3D 12,
// then soft into the window).
func rendererCandidates() []rendererCandidate { return nil }

// SystemFontMatcher returns nil: the text.Collection scans the font directories (TODO(step 28.6): the
// matcher of the system fonts of Windows).
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendUnavailable = "BackendUnavailable"
)
