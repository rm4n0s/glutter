package gles

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

func TestSoftware(t *testing.T) {
	tests := []struct {
		renderer string
		want     bool
	}{
		// Rasterizers on the CPU.
		{"llvmpipe (LLVM 19.1.7, 256 bits)", true},
		{"llvmpipe (LLVM 15.0.7, 128 bits)", true},
		{"softpipe", true},
		{"Mesa DRI swrast", true},
		{"swrast", true},
		{"Software Rasterizer", true},
		{"Apple Software Renderer Software Rasterizer", true},
		{"Google SwiftShader", true},
		{"ANGLE (Google, Vulkan 1.3.0 (SwiftShader Device (Subzero) (0x0000C0DE)), SwiftShader driver)", true},
		{"LLVMPIPE", true},
		// Hardware, including drivers with LLVM or a virtual machine's GPU in the name.
		{"AMD Radeon 8060S Graphics (radeonsi, gfx1151, LLVM 19.1.7, DRM 3.64, 6.14.0)", false},
		{"Mesa Intel(R) UHD Graphics 620 (KBL GT2)", false},
		{"NVIDIA GeForce RTX 3080/PCIe/SSE2", false},
		{"virgl (NVIDIA GeForce RTX 3080)", false},
		{"Adreno (TM) 650", false},
		{"Mali-G78", false},
		{"Apple M1", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.renderer, func(t *testing.T) {
			if got := Software(tt.renderer); got != tt.want {
				t.Errorf("Software(%q) = %v, want %v", tt.renderer, got, tt.want)
			}
		})
	}
}

func TestRejectSoftwareClosesTheContextAndFailsTheFirstAttach(t *testing.T) {
	tests := []struct {
		name     string
		renderer string
		reject   bool
		wantErr  bool
	}{
		{"llvmpipe in auto", "llvmpipe (LLVM 19.1.7, 256 bits)", true, true},
		{"llvmpipe by name", "llvmpipe (LLVM 19.1.7, 256 bits)", false, false},
		{"a GPU in auto", "AMD Radeon 8060S Graphics (radeonsi)", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.0 Mesa", renderer: tt.renderer}
			r := newRenderer(func(h gpu.SurfaceHandle) (*Context, *Surface, error) {
				return openWith(egl, fakeGLLoader(gl), h)
			}, RendererOpts{RejectSoftware: tt.reject})
			s, err := r.Attach(waylandHandle)
			if !tt.wantErr {
				if err != nil || s == nil {
					t.Fatalf("Attach = %v, want a surface", err)
				}
				if err := r.Close(); err != nil {
					t.Fatal(err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagSoftwareRenderer || !appErr.HasRoute("Renderer.Attach.SoftwareRenderer") {
				t.Errorf("tag %s, route %s", appErr.Tag, appErr.Route())
			}
			if appErr.Metadata["renderer"] != tt.renderer {
				t.Errorf("metadata %v, want the GL_RENDERER string that decided", appErr.Metadata)
			}
			// The context is gone: surface, context and display, and what the device made.
			for _, want := range []string{"destroySurface", "destroyContext", "terminate"} {
				if !contains(egl.calls, want) {
					t.Errorf("the rejection did not call %s: %v", want, egl.calls)
				}
			}
			if gl.liveCount("program") != 0 {
				t.Errorf("%d programs leaked", gl.liveCount("program"))
			}
			if r.DeviceName() != "" {
				t.Errorf("the renderer keeps the device %q after the rejection", r.DeviceName())
			}
			if err := r.Close(); err != nil {
				t.Errorf("Close after the rejection: %v", err)
			}
		})
	}
}
