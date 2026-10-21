package webgpu

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// TestErrorRoutes checks that every failure has its own tag and that the route of the error names
// the function that made it, which is what the tests of the callers match.
func TestErrorRoutes(t *testing.T) {
	cause := errors.New("Cause", "what the browser threw")
	tests := []struct {
		name  string
		err   error
		tag   string
		route string
	}{
		{"unavailable", errUnavailable("no navigator.gpu"), "WebGPUUnavailable", "errUnavailable.WebGPUUnavailable"},
		{"device failed", errDeviceFailed("requestDevice rejected"), "WebGPUDeviceFailed", "errDeviceFailed.WebGPUDeviceFailed"},
		{"shader invalid", errShaderInvalid("blit:3:1: expected"), "ShaderModuleInvalid", "errShaderInvalid.ShaderModuleInvalid"},
		{"device lost", errDeviceLost("destroyed", "Device was destroyed."), "GPUDeviceLost", "errDeviceLost.GPUDeviceLost"},
		{"canvas not found", errCanvasNotFound(3), "WebGPUCanvasNotFound", "errCanvasNotFound.WebGPUCanvasNotFound"},
		{"surface kind", errSurfaceKind("wayland"), "WebGPUSurfaceKindUnsupported", "errSurfaceKind.WebGPUSurfaceKindUnsupported"},
		{"context unavailable", errContextUnavailable(3), "WebGPUContextUnavailable", "errContextUnavailable.WebGPUContextUnavailable"},
		{"configure failed", errConfigureFailed(cause), "WebGPUConfigureFailed", "errConfigureFailed.WebGPUConfigureFailed"},
		{"call failed", errCallFailed("createTexture", cause), "WebGPUCallFailed", "errCallFailed.WebGPUCallFailed"},
		{"frame open", errFrameOpen(), "WebGPUFrameOpen", "errFrameOpen.WebGPUFrameOpen"},
		{"frame not open", errFrameNotOpen(), "WebGPUFrameNotOpen", "errFrameNotOpen.WebGPUFrameNotOpen"},
		{"pass not ended", errPassNotEnded(), "WebGPUPassNotEnded", "errPassNotEnded.WebGPUPassNotEnded"},
		{"foreign resource", errForeign("buffer"), "WebGPUForeignResource", "errForeign.WebGPUForeignResource"},
		{"texture invalid", errTextureInvalid(0, 4, 0), "WebGPUTextureInvalid", "errTextureInvalid.WebGPUTextureInvalid"},
		{"pipeline invalid", errPipelineInvalid("a pipeline needs a shader"), "WebGPUPipelineInvalid", "errPipelineInvalid.WebGPUPipelineInvalid"},
		{"bind group invalid", errBindGroupInvalid("the pipeline has no such bind group", "group", 5), "WebGPUBindGroupInvalid", "errBindGroupInvalid.WebGPUBindGroupInvalid"},
		{"write out of range", errWriteOutOfRange(8, 16, 12), "WebGPUWriteOutOfRange", "errWriteOutOfRange.WebGPUWriteOutOfRange"},
		{"level invalid", errLevelInvalid(3, 1), "WebGPULevelInvalid", "errLevelInvalid.WebGPULevelInvalid"},
		{"read failed", errReadFailed("Invalid buffer"), "WebGPUReadFailed", "errReadFailed.WebGPUReadFailed"},
		{"device pending", devicePending(), explicit.GPUDevicePending, "devicePending.GPUDevicePending"},
	}
	seen := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", tt.err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
			if seen[tt.tag] {
				t.Errorf("the tag %s is used for two failures", tt.tag)
			}
			seen[tt.tag] = true
		})
	}
	if !explicit.IsDevicePending(devicePending()) || explicit.IsDevicePending(errUnavailable("x")) {
		t.Error("IsDevicePending")
	}
}

// TestAcquireWithoutWebGPU: where there is no browser (Linux), or the browser has no navigator.gpu
// (Node, which runs this test under scripts/test-wasm.sh), Acquire fails with WebGPUUnavailable, and
// a renderer over no provider fails the same way when it opens its device.
func TestAcquireWithoutWebGPU(t *testing.T) {
	p, err := Acquire(context.Background(), Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WebGPUUnavailable" || !appErr.HasRoute("Acquire->") || !appErr.HasRoute("errUnavailable.WebGPUUnavailable") {
		t.Fatalf("Acquire = %v, %v; want WebGPUUnavailable via Acquire->...->errUnavailable", p, err)
	}

	_, err = (*Provider)(nil).Open(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1})
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != "WebGPUUnavailable" || !appErr.HasRoute("Open") {
		t.Fatalf("Open on no provider = %v, want WebGPUUnavailable via Open", err)
	}
}
