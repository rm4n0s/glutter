//go:build android

package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on Android until the platform exists (step 25.3): Run fails, there
// are no renderers (steps 25.3 to 25.5) and no system font matcher (step 25.10).

// connectDisplay fails: the Android platform comes in step 25.3.
func connectDisplay(*Config, func(string) string, *slog.Logger) (connection, error) {
	return nil, errors.New(tagBackendUnavailable, "the Android platform is not there yet (step 25.3)")
}

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
