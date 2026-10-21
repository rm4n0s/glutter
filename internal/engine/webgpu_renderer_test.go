package engine

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// TestWebGPURendererWithoutAStartupDeviceFailsAtTheFirstFrame: a renderer made while Run did not get
// a WebGPU device (the engine's tests, or a connectDisplay that was not run) says so with
// WebGPUUnavailable from its first frame, which is what makes auto go on to WebGL 2.
func TestWebGPURendererWithoutAStartupDeviceFailsAtTheFirstFrame(t *testing.T) {
	r := newWebGPURenderer(&Config{Getenv: func(string) string { return "" }}, quiet())
	defer r.Close()
	_, err := r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1}, 8, 8, nil, false)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WebGPUUnavailable" {
		t.Fatalf("Draw = %v, want WebGPUUnavailable", err)
	}
}

func TestCloseWebGPUIsSafeWithoutAStartupDevice(t *testing.T) {
	var c Config
	c.closeWebGPU() // nothing was started: nothing happens
	closed := 0
	c.webgpuClose = func() { closed++ }
	c.closeWebGPU()
	if closed != 1 {
		t.Errorf("closed %d times, want 1", closed)
	}
}
