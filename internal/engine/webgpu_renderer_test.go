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
	if !ok || appErr.Tag != tagWebGPUUnavailable {
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

// TestWebGPULastResort: only the loss that the explicit renderer gives up on (a second one within
// ten seconds) and a new device that the browser refuses switch the page to WebGL 2.
func TestWebGPULastResort(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"a device lost twice", errors.New(tagGPUDeviceLost, "the WebGPU device was lost"), true},
		{"a new device that was refused", errors.New(tagWebGPUDeviceFailed, "the adapter did not give a device"), true},
		{"a call that threw", errors.New(tagWebGPUCallFailed, "a call threw"), false},
		{"no WebGPU at all", errors.New(tagWebGPUUnavailable, "no navigator.gpu"), false},
		{"a size that is not positive", errors.New(tagRendererSizeInvalid, "bad size"), false},
		{"not an application error", plainError{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := webgpuLastResort(tc.err); got != tc.want {
				t.Errorf("webgpuLastResort(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

type plainError struct{}

func (plainError) Error() string { return "deadline" }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRendererSizeInvalid = "RendererSizeInvalid"
	tagWebGPUCallFailed    = "WebGPUCallFailed"
	tagWebGPUDeviceFailed  = "WebGPUDeviceFailed"
	tagGPUDeviceLost       = "GPUDeviceLost"
)
