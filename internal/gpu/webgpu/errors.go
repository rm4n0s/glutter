package webgpu

import "github.com/rm4n0s/glutter/internal/gpu/explicit"

// The tags of the errors of this package. Each failure has a tag of its own (CLAUDE.md), and
// the route of an error lists the calls it came through.
const (
	// TagUnavailable is the tag of a browser that has no navigator.gpu, or gives no adapter.
	TagUnavailable = "WebGPUUnavailable"
	// TagDeviceFailed is the tag of an adapter that does not give a device.
	TagDeviceFailed = "WebGPUDeviceFailed"
	// TagShaderInvalid is the tag of a WGSL module the browser rejects: an internal bug.
	TagShaderInvalid = "ShaderModuleInvalid"
	// TagDeviceLost is the tag of a device that was lost, which explicit.Renderer recovers from.
	TagDeviceLost = explicit.TagGPUDeviceLost
)
