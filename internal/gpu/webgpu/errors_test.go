package webgpu

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// TestAcquireWithoutWebGPU: where there is no browser (Linux), or the browser has no navigator.gpu
// (Node, which runs this test under scripts/test-wasm.sh), Acquire fails with WebGPUUnavailable, and
// a renderer over no provider fails the same way when it opens its device.
func TestAcquireWithoutWebGPU(t *testing.T) {
	p, err := Acquire(context.Background(), Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagUnavailable || !appErr.HasRoute("Acquire.WebGPUUnavailable") {
		t.Fatalf("Acquire = %v, %v; want WebGPUUnavailable via Acquire", p, err)
	}

	_, err = (*Provider)(nil).Open(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1})
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != TagUnavailable || !appErr.HasRoute("Provider.Open.WebGPUUnavailable") {
		t.Fatalf("Open on no provider = %v, want WebGPUUnavailable via Provider.Open", err)
	}
}
