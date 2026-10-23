//go:build js && wasm

package gles

import (
	"syscall/js"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// TestOpenWebFailures runs openWeb against a document of Go's making: Node has no page, so the
// document's querySelector answers what each case needs.
func TestOpenWebFailures(t *testing.T) {
	noContext := func() any {
		return map[string]any{"getContext": js.FuncOf(func(js.Value, []js.Value) any { return nil })}
	}
	tests := []struct {
		name   string
		kind   gpu.SurfaceKind
		canvas func() any // what querySelector answers; nil answers null
		tag    string
		key    string
		value  string
	}{
		{"a handle of another kind", gpu.SurfaceKindWayland, nil, "WebGLSurfaceKindUnsupported", "kind", string(gpu.SurfaceKindWayland)},
		{"canvas not found", gpu.SurfaceKindWeb, func() any { return nil }, "WebGLCanvasNotFound", "view", "3"},
		{"no WebGL 2", gpu.SurfaceKindWeb, noContext, "WebGL2Unavailable", "view", "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global := js.Global()
			old := global.Get("document")
			query := js.FuncOf(func(js.Value, []js.Value) any {
				if tt.canvas == nil {
					return nil
				}
				return tt.canvas()
			})
			global.Set("document", map[string]any{"querySelector": query})
			defer func() {
				global.Set("document", old)
				query.Release()
			}()

			_, _, err := openWeb(gpu.SurfaceHandle{Kind: tt.kind, Window: 3}, nil)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("openWeb."+tt.tag) {
				t.Errorf("got %s via %s, want %s via openWeb", appErr.Tag, appErr.Route(), tt.tag)
			}
			if appErr.Metadata[tt.key] != tt.value {
				t.Errorf("metadata = %v, want %s = %s", appErr.Metadata, tt.key, tt.value)
			}
			if tt.tag == "WebGL2Unavailable" && appErr.Metadata["reason"] == "" {
				t.Errorf("no reason in %v", appErr.Metadata)
			}
		})
	}
}
