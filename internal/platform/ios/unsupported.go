package ios

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// This file has what the iOS platform does not have in step 35.2 and answers with an error, in plain Go (no OS tag), so that a test of
// each route runs on every host. The steps that follow replace them: the pasteboard comes with step 35.6.

// unsupportedWindow is embedded by the window: the calls of platform.Window that iOS has no counterpart for.
type unsupportedWindow struct{}

// StartDrag fails with IOSDragUnsupported: dragging out of the app is not in 1.15 (SPEC §30.9).
func (unsupportedWindow) StartDrag(ui.DragRequest) error {
	return errors.New(tagIOSDragUnsupported, "dragging out of an iOS app is not supported")
}

// pasteboard is the clipboard until step 35.6: every call answers with IOSClipboardUnavailable.
type pasteboard struct{}

// SetText fails with IOSClipboardUnavailable.
func (pasteboard) SetText(_ context.Context, _ string, done func(err error)) {
	done(errors.New(tagIOSClipboardUnavailable, "the pasteboard of iOS is not connected yet"))
}

// GetText fails with IOSClipboardUnavailable.
func (pasteboard) GetText(_ context.Context, done func(text string, ok bool, err error)) {
	done("", false, errors.New(tagIOSClipboardUnavailable, "the pasteboard of iOS is not connected yet"))
}

// SetContent fails with IOSClipboardUnavailable.
func (pasteboard) SetContent(_ context.Context, _ ui.ClipboardKind, _ ui.ClipboardContent, done func(err error)) {
	done(errors.New(tagIOSClipboardUnavailable, "the pasteboard of iOS is not connected yet"))
}

// GetContent fails with IOSClipboardUnavailable.
func (pasteboard) GetContent(_ context.Context, _ ui.ClipboardKind, _ string, done func(content ui.ClipboardContent, err error)) {
	done(ui.ClipboardContent{}, errors.New(tagIOSClipboardUnavailable, "the pasteboard of iOS is not connected yet"))
}

const (
	tagIOSDragUnsupported      = "IOSDragUnsupported"
	tagIOSClipboardUnavailable = "IOSClipboardUnavailable"
)
