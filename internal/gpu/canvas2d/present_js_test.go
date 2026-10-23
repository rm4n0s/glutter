//go:build js && wasm

package canvas2d

import (
	"syscall/js"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// TestFindFailures runs canvasPresenter.find against a document of Go's making: Node has no
// page, so the document's querySelector answers what each case needs.
func TestFindFailures(t *testing.T) {
	tests := []struct {
		name   string
		canvas func() any // what querySelector answers; nil answers null
		tag    string
	}{
		{"canvas not found", func() any { return nil }, "CanvasNotFound"},
		{"no 2D context", func() any {
			return map[string]any{"getContext": js.FuncOf(func(js.Value, []js.Value) any { return nil })}
		}, "Canvas2DUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global := js.Global()
			old := global.Get("document")
			query := js.FuncOf(func(js.Value, []js.Value) any { return tt.canvas() })
			global.Set("document", map[string]any{"querySelector": query})
			defer func() {
				global.Set("document", old)
				query.Release()
			}()

			p := &canvasPresenter{targets: map[uintptr]*target{}}
			_, err := p.find(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 3})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("canvasPresenter.find."+tt.tag) {
				t.Errorf("got %s via %s, want %s via canvasPresenter.find", appErr.Tag, appErr.Route(), tt.tag)
			}
			if appErr.Metadata["view"] != "3" {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}
