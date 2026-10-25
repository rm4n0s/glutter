//go:build windows

package windows

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// unavailableClipboard is the clipboard until step 28.6: every call fails with ClipboardUnavailable.
type unavailableClipboard struct{}

var _ platform.Clipboard = unavailableClipboard{}

// SetText fails with ClipboardUnavailable.
//
// Flutter: no counterpart
func (unavailableClipboard) SetText(_ context.Context, _ string, done func(err error)) {
	if done != nil {
		done(errors.New(tagClipboardUnavailable, "the clipboard of Windows comes with step 28.6"))
	}
}

// GetText fails with ClipboardUnavailable.
//
// Flutter: no counterpart
func (unavailableClipboard) GetText(_ context.Context, done func(text string, ok bool, err error)) {
	if done != nil {
		done("", false, errors.New(tagClipboardUnavailable, "the clipboard of Windows comes with step 28.6"))
	}
}

// SetContent fails with ClipboardUnavailable.
//
// Flutter: no counterpart
func (unavailableClipboard) SetContent(_ context.Context, _ ui.ClipboardKind, _ ui.ClipboardContent, done func(err error)) {
	if done != nil {
		done(errors.New(tagClipboardUnavailable, "the clipboard of Windows comes with step 28.6"))
	}
}

// GetContent fails with ClipboardUnavailable.
//
// Flutter: no counterpart
func (unavailableClipboard) GetContent(_ context.Context, _ ui.ClipboardKind, _ string, done func(content ui.ClipboardContent, err error)) {
	if done != nil {
		done(ui.ClipboardContent{}, errors.New(tagClipboardUnavailable, "the clipboard of Windows comes with step 28.6"))
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardUnavailable = "ClipboardUnavailable"
)
