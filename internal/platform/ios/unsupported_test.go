package ios

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func TestStartDragIsUnsupported(t *testing.T) {
	err := unsupportedWindow{}.StartDrag(ui.DragRequest{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the error is a %T: %v", err, err)
	}
	if !appErr.HasRoute("unsupportedWindow.StartDrag." + tagIOSDragUnsupported) {
		t.Errorf("route = %s", appErr.Route())
	}
}

func TestPasteboardIsUnavailable(t *testing.T) {
	ctx := context.Background()
	var p pasteboard
	for name, call := range map[string]func(done func(error)){
		"SetText":    func(done func(error)) { p.SetText(ctx, "x", done) },
		"GetText":    func(done func(error)) { p.GetText(ctx, func(_ string, _ bool, err error) { done(err) }) },
		"SetContent": func(done func(error)) { p.SetContent(ctx, 0, ui.ClipboardContent{}, done) },
		"GetContent": func(done func(error)) { p.GetContent(ctx, 0, "", func(_ ui.ClipboardContent, err error) { done(err) }) },
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			call(func(err error) {
				called = true
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("the error is a %T: %v", err, err)
				}
				if appErr.Tag != tagIOSClipboardUnavailable || !appErr.HasRoute("pasteboard."+name+"."+tagIOSClipboardUnavailable) {
					t.Errorf("tag %s, route = %s", appErr.Tag, appErr.Route())
				}
			})
			if !called {
				t.Error("done was not called")
			}
		})
	}
}
