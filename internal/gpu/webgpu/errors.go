package webgpu

import (
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

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
	TagDeviceLost = "GPUDeviceLost"
)

// errUnavailable is the error for a browser without WebGPU.
func errUnavailable(reason string) error {
	return errors.New(TagUnavailable, "this browser has no usable WebGPU", "reason", reason)
}

// errDeviceFailed is the error for an adapter that gave no device.
func errDeviceFailed(reason string) error {
	return errors.New(TagDeviceFailed, "the WebGPU adapter did not give a device", "reason", reason)
}

// errShaderInvalid is the error for WGSL that the browser's compiler rejects; messages are its
// messages. A shader of Glutter that fails is an internal bug.
func errShaderInvalid(messages string) error {
	return errors.New(TagShaderInvalid, "the browser rejected a WGSL module of Glutter", "messages", messages)
}

// errDeviceLost is the error for calls on a lost device.
func errDeviceLost(reason, message string) error {
	return errors.New(TagDeviceLost, "the WebGPU device was lost", "reason", reason, "message", message)
}

// errCanvasNotFound is the error for a view whose canvas is not in the page.
func errCanvasNotFound(view uintptr) error {
	return errors.New("WebGPUCanvasNotFound", "there is no canvas for the view in the page", "view", strconv.FormatUint(uint64(view), 10))
}

// errSurfaceKind is the error for a handle that is not a canvas of the web.
func errSurfaceKind(kind string) error {
	return errors.New("WebGPUSurfaceKindUnsupported", "WebGPU draws into the canvas of a web page only", "kind", kind)
}

// errContextUnavailable is the error for a canvas that gives no webgpu context: it has a context
// of another kind already (a canvas keeps the first kind it was asked for, SPEC §9.6), or the
// browser has none.
func errContextUnavailable(view uintptr) error {
	return errors.New("WebGPUContextUnavailable", "the canvas gave no webgpu context", "view", strconv.FormatUint(uint64(view), 10))
}

// errConfigureFailed is the error for a canvas context that cannot be configured.
func errConfigureFailed(cause error) error {
	return errors.NewErr("WebGPUConfigureFailed", cause)
}

// errCallFailed is the error for a call into the browser's API that threw.
func errCallFailed(call string, cause error) error {
	return errors.NewErr("WebGPUCallFailed", cause).SetMetadata("call", call)
}

// errFrameOpen is the error for Begin before the frame in progress was submitted.
func errFrameOpen() error {
	return errors.New("WebGPUFrameOpen", "Begin was called before the frame in progress was submitted")
}

// errFrameNotOpen is the error for Submit on a frame that is not the one in progress.
func errFrameNotOpen() error {
	return errors.New("WebGPUFrameNotOpen", "Submit was called on a frame that is not the one in progress")
}

// errPassNotEnded is the error (a panic, a programmer error) for BeginPass before the pass ended.
func errPassNotEnded() error {
	return errors.New("WebGPUPassNotEnded", "BeginPass was called before the previous pass ended")
}

// errForeign is the error (a panic, a programmer error) for a resource this device did not make.
func errForeign(what string) error {
	return errors.New("WebGPUForeignResource", "the resource was not made by this WebGPU device", "resource", what)
}

// errTextureInvalid is the error for a texture that has no size or no format.
func errTextureInvalid(width, height, format int) error {
	return errors.New("WebGPUTextureInvalid", "a texture needs a size and a format", "width", width, "height", height, "format", format)
}

// errPipelineInvalid is the error for a pipeline that has no shader, or names an entry point the
// shader lacks or a shader that is not one of the renderer's.
func errPipelineInvalid(message string, md ...any) error {
	return errors.New("WebGPUPipelineInvalid", message, md...)
}

// errBindGroupInvalid is the error for a bind group of a group or a binding that the pipeline's
// shader does not have.
func errBindGroupInvalid(message string, md ...any) error {
	return errors.New("WebGPUBindGroupInvalid", message, md...)
}

// errWriteOutOfRange is the error (a panic, a programmer error) for a write outside its buffer.
func errWriteOutOfRange(offset, bytes, size int) error {
	return errors.New("WebGPUWriteOutOfRange", "the write is outside the buffer", "offset", offset, "bytes", bytes, "size", size)
}

// errLevelInvalid is the error (a panic, a programmer error) for a mip level the texture lacks.
func errLevelInvalid(level, levels int) error {
	return errors.New("WebGPULevelInvalid", "the texture has no such mip level", "level", level, "levels", levels)
}

// errReadFailed is the error for a read-back that the browser could not map.
func errReadFailed(message string) error {
	return errors.New("WebGPUReadFailed", "the browser could not map the buffer of a read-back", "message", message)
}

// devicePending is the error of an open function that has no device yet (explicit.GPUDevicePending).
func devicePending() error {
	return errors.New(explicit.GPUDevicePending, "the browser has not given the new WebGPU device yet")
}
