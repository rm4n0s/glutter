package engine

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// TestTextureLayersOnARendererWithoutGPUTexturesAreLoggedOnce: the OpenGL ES, WebGL and software renderers draw nothing
// for a texture layer, and the engine says so once, at Info level (SPEC §27.4).
func TestTextureLayersOnARendererWithoutGPUTexturesAreLoggedOnce(t *testing.T) {
	tests := []struct {
		name     string
		renderer string
		texture  bool
		want     int
	}{
		{"GLES", "gles", true, 1},
		{"WebGL", "webgl", true, 1},
		{"the software renderer", "soft", true, 1},
		{"Vulkan", "vulkan", true, 0},
		{"WebGPU", "webgpu", true, 0},
		{"Direct3D 12", "d3d12", true, 0},
		{"GLES with no texture in the scene", "gles", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			var out bytes.Buffer
			e.h.log = slog.New(slog.NewTextHandler(&out, nil))
			e.h.gpu.renderer = func() (bool, string) { return true, tt.renderer }
			b := ui.NewSceneBuilder()
			if tt.texture {
				b.AddTexture(4, ui.AddTextureOpts{Width: 10, Height: 10})
			}
			e.size(0, 400, 300)
			e.renderAll(map[ui.ViewID]*ui.Scene{0: b.Build()})
			for range 3 {
				e.h.OnFrame(0, time.Millisecond)
			}
			if n := strings.Count(out.String(), "a texture layer draws nothing"); n != tt.want {
				t.Errorf("logged %d times, want %d:\n%s", n, tt.want, out.String())
			}
		})
	}
}
