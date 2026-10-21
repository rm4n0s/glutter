package web

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestErrorRoutes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		tag  string
		fn   string
	}{
		{"host element", errHostElementNotFound("#missing"), "HostElementNotFound", "errHostElementNotFound"},
		{"second window", errWindowUnsupported("a second window was asked for"), "WindowUnsupported", "errWindowUnsupported"},
		{"no page", errNoPage(), "BackendUnavailable", "errNoPage"},
		{"drag", errDragUnsupported(), "DragUnsupported", "errDragUnsupported"},
		{"screen on", errScreenOnUnsupported(), "IdleInhibitUnavailable", "errScreenOnUnsupported"},
		{"clipboard", errClipboardUnavailable(), "WebClipboardUnavailable", "errClipboardUnavailable"},
		{"clipboard read", errClipboardReadFailed("NotAllowedError"), "WebClipboardReadFailed", "errClipboardReadFailed"},
		{"clipboard write", errClipboardWriteFailed("NotAllowedError"), "WebClipboardWriteFailed", "errClipboardWriteFailed"},
		{"clipboard format", errClipboardFormatUnsupported(), "WebClipboardFormatUnsupported", "errClipboardFormatUnsupported"},
		{"clipboard primary", errClipboardPrimaryUnavailable(), "ClipboardPrimaryUnavailable", "errClipboardPrimaryUnavailable"},
		{"clipboard timeout", errClipboardReadTimeout(context.DeadlineExceeded), "ClipboardReadTimeout", "errClipboardReadTimeout"},
		{"screenshot", errScreenshotPostFailed("status 500"), "ScreenshotWriteFailed", "errScreenshotPostFailed"},
		{"run twice", errAlreadyRunning(), "PlatformAlreadyRunning", "errAlreadyRunning"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tc.err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", tc.err)
			}
			if appErr.Tag != tc.tag || !appErr.HasRoute(tc.fn+"."+tc.tag) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tc.tag, tc.fn)
			}
		})
	}
}

func TestHostElementNotFoundNamesTheSelector(t *testing.T) {
	appErr, _ := errors.FromError(errHostElementNotFound("#missing"))
	if appErr.Metadata["selector"] != "#missing" {
		t.Errorf("metadata = %v, want the selector", appErr.Metadata)
	}
}
