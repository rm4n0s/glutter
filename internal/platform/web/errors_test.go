package web

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
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

func TestClipboardFailsThroughPost(t *testing.T) {
	var posted []func()
	c := clipboard{post: func(fn func()) { posted = append(posted, fn) }}
	var errs []error
	c.SetText(context.Background(), "x", func(err error) { errs = append(errs, err) })
	c.GetText(context.Background(), func(_ string, ok bool, err error) {
		if ok {
			t.Error("GetText said it read text")
		}
		errs = append(errs, err)
	})
	c.SetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardContent{}, func(err error) { errs = append(errs, err) })
	c.GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { errs = append(errs, err) })
	c.SetText(context.Background(), "y", nil) // no callback: nothing to post
	if len(errs) != 0 || len(posted) != 4 {
		t.Fatalf("answered before the call returned (%d answers) or posted %d, want 0 and 4", len(errs), len(posted))
	}
	for _, fn := range posted {
		fn()
	}
	for i, err := range errs {
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "WebClipboardUnavailable" {
			t.Errorf("answer %d = %v, want WebClipboardUnavailable", i, err)
		}
	}
}
