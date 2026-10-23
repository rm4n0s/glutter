package android

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// unavailableClipboard is the clipboard until step 25.8 brings ClipboardManager: every call
// fails with ClipboardUnavailable, through the done function, after the call returned.
//
// Flutter: services/clipboard.dart Clipboard
type unavailableClipboard struct {
	post func(func())
}

var _ platform.Clipboard = unavailableClipboard{}

// SetText fails with ClipboardUnavailable.
func (c unavailableClipboard) SetText(_ context.Context, _ string, done func(error)) {
	err := errors.New(tagClipboardUnavailable, "the clipboard of Android is not there yet (step 25.8)")
	c.post(func() {
		if done != nil {
			done(err)
		}
	})
}

// GetText fails with ClipboardUnavailable.
func (c unavailableClipboard) GetText(_ context.Context, done func(string, bool, error)) {
	err := errors.New(tagClipboardUnavailable, "the clipboard of Android is not there yet (step 25.8)")
	c.post(func() {
		if done != nil {
			done("", false, err)
		}
	})
}

// SetContent fails with ClipboardUnavailable.
func (c unavailableClipboard) SetContent(_ context.Context, _ ui.ClipboardKind, _ ui.ClipboardContent, done func(error)) {
	err := errors.New(tagClipboardUnavailable, "the clipboard of Android is not there yet (step 25.8)")
	c.post(func() {
		if done != nil {
			done(err)
		}
	})
}

// GetContent fails with ClipboardUnavailable.
func (c unavailableClipboard) GetContent(_ context.Context, _ ui.ClipboardKind, _ string, done func(ui.ClipboardContent, error)) {
	err := errors.New(tagClipboardUnavailable, "the clipboard of Android is not there yet (step 25.8)")
	c.post(func() {
		if done != nil {
			done(ui.ClipboardContent{}, err)
		}
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardUnavailable = "ClipboardUnavailable"
)
