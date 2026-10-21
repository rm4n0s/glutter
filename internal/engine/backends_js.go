//go:build js

package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

// This file is what Run uses on the web (GOOS=js): no platform backend yet, so Run fails.

// connectDisplay fails: this build has no platform backend.
func connectDisplay(*Config, func(string) string, *slog.Logger) (connection, error) {
	return nil, errors.New("BackendUnavailable", "this build has no platform backend for this operating system", "goos", "js")
}

// processStart says it does not know when the process started.
func processStart() (time.Time, bool) { return time.Time{}, false }

// rendererCandidates is empty: Run fails before it needs a renderer.
func rendererCandidates() []rendererCandidate { return nil }

// SystemFontMatcher returns nil: the text.Collection scans the font directories.
//
// Flutter: no counterpart
func SystemFontMatcher(context.Context, *slog.Logger) text.Matcher { return nil }
