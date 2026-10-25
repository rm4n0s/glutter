//go:build windows

package windows

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestTheClipboardIsUnavailableUntilStep28_6(t *testing.T) {
	var c platform.Clipboard = unavailableClipboard{}
	ctx := context.Background()
	tests := []struct {
		name  string
		call  func(done func(error))
		route string
	}{
		{"SetText", func(done func(error)) { c.SetText(ctx, "x", done) }, "unavailableClipboard.SetText.ClipboardUnavailable"},
		{"GetText", func(done func(error)) {
			c.GetText(ctx, func(_ string, ok bool, err error) {
				if ok {
					t.Error("ok")
				}
				done(err)
			})
		}, "unavailableClipboard.GetText.ClipboardUnavailable"},
		{"SetContent", func(done func(error)) { c.SetContent(ctx, ui.ClipboardKindUnspecified, ui.ClipboardContent{}, done) }, "unavailableClipboard.SetContent.ClipboardUnavailable"},
		{"GetContent", func(done func(error)) {
			c.GetContent(ctx, ui.ClipboardKindUnspecified, "text/plain", func(_ ui.ClipboardContent, err error) { done(err) })
		}, "unavailableClipboard.GetContent.ClipboardUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got error
			tt.call(func(err error) { got = err })
			assertRoute(t, got, "ClipboardUnavailable", tt.route)
		})
	}
	// A nil done is allowed.
	c.SetText(ctx, "x", nil)
	c.GetText(ctx, nil)
}

func TestStartDragIsUnsupportedUntilStep28_12(t *testing.T) {
	err := (&window{}).StartDrag(ui.DragRequest{})
	assertRoute(t, err, "DragUnsupported", "window.StartDrag.DragUnsupported")
}

func TestTheCallbacksOfTheWindowsStayUnderTheLimit(t *testing.T) {
	_ = wndProcCallback()
	if n := win32.CallbackCount(); n > 1500 {
		t.Errorf("%d callbacks", n)
	}
}
