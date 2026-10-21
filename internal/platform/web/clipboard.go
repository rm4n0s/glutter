package web

import (
	"context"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// clipboard is the clipboard of the web platform until step 21.3: every call fails
// with WebClipboardUnavailable. Like the real ones it answers through Post, never
// before the call has returned.
//
// Flutter: no counterpart
type clipboard struct{ post func(func()) }

var _ platform.Clipboard = clipboard{}

func (c clipboard) fail(report func(error)) {
	if report != nil {
		c.post(func() { report(errClipboardUnavailable()) })
	}
}

// SetText fails.
//
// Flutter: no counterpart
func (c clipboard) SetText(_ context.Context, _ string, done func(error)) { c.fail(done) }

// GetText fails.
//
// Flutter: no counterpart
func (c clipboard) GetText(_ context.Context, done func(string, bool, error)) {
	if done != nil {
		c.post(func() { done("", false, errClipboardUnavailable()) })
	}
}

// SetContent fails.
//
// Flutter: no counterpart
func (c clipboard) SetContent(_ context.Context, _ ui.ClipboardKind, _ ui.ClipboardContent, done func(error)) {
	c.fail(done)
}

// GetContent fails.
//
// Flutter: no counterpart
func (c clipboard) GetContent(_ context.Context, _ ui.ClipboardKind, _ string, done func(ui.ClipboardContent, error)) {
	if done != nil {
		c.post(func() { done(ui.ClipboardContent{}, errClipboardUnavailable()) })
	}
}
