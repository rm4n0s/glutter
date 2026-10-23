//go:build !(js && wasm)

package webgpu

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Provider hands out the device of the browser's WebGPU. Off the web there is none: Acquire
// fails and Open on a nil provider says so.
//
// Flutter: no counterpart
type Provider struct{}

// Acquire fails with WebGPUUnavailable: only a browser has WebGPU.
//
// Flutter: no counterpart
func Acquire(context.Context, Options) (*Provider, error) {
	return nil, errors.New(TagUnavailable, "this browser has no usable WebGPU", "reason", "this program does not run in a browser")
}

// Open fails with WebGPUUnavailable.
//
// Flutter: no counterpart
func (p *Provider) Open(gpu.SurfaceHandle) (explicit.Device, error) {
	return nil, errors.New(TagUnavailable, "this browser has no usable WebGPU", "reason", "this program does not run in a browser")
}

// Close does nothing: there is no device.
//
// Flutter: no counterpart
func (p *Provider) Close() {}
