//go:build darwin && !ios

package macos

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// clipboard is the pasteboard until step 33.5 reads and writes it: every call fails with
// ClipboardUnavailable (SPEC §29.7).
type clipboard struct{}

func (*clipboard) fail(done func(error)) {
	if done != nil {
		done(errors.New(TagClipboardUnavailable, "the pasteboard of macOS is not connected yet"))
	}
}

// SetText fails with ClipboardUnavailable.
func (c *clipboard) SetText(_ context.Context, _ string, done func(err error)) { c.fail(done) }

// GetText fails with ClipboardUnavailable.
func (*clipboard) GetText(_ context.Context, done func(text string, ok bool, err error)) {
	if done != nil {
		done("", false, errors.New(TagClipboardUnavailable, "the pasteboard of macOS is not connected yet"))
	}
}

// SetContent fails with ClipboardUnavailable.
func (c *clipboard) SetContent(_ context.Context, _ ui.ClipboardKind, _ ui.ClipboardContent, done func(err error)) {
	c.fail(done)
}

// GetContent fails with ClipboardUnavailable.
func (*clipboard) GetContent(_ context.Context, _ ui.ClipboardKind, _ string, done func(content ui.ClipboardContent, err error)) {
	if done != nil {
		done(ui.ClipboardContent{}, errors.New(TagClipboardUnavailable, "the pasteboard of macOS is not connected yet"))
	}
}
