package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The tags of the errors of package gpu (SPEC §27.11). Each has a route test. An error that a backend
// makes (GPUOutOfMemory, VulkanDeviceLost, GPUShaderCompileFailed, ...) comes back untouched; IsDeviceLost
// says whether an error is a loss of the device, whatever backend it came from.
const (
	// TagGPUUnavailable is the tag of the error of DeviceOf when the renderer of the app has no GPU device to
	// give: the app runs on the OpenGL ES, WebGL or software renderer, or has no engine behind it. The
	// metadata "renderer" says why.
	//
	// Flutter: no counterpart
	TagGPUUnavailable = "GPUUnavailable"
	// TagGPUDeviceNotReady is the tag of the error of DeviceOf before the renderer has opened its device,
	// which it does when it draws its first frame (the choice of a device depends on the window), and
	// between the loss of a device and the frame that opens the next one. OnDevice calls its function when
	// there is one.
	//
	// Flutter: no counterpart
	TagGPUDeviceNotReady = "GPUDeviceNotReady"
	// TagGPUDeviceLost is the tag of the error of a call on a device that was lost, and of the error that
	// OnLost functions get. The device and everything made with it stay unusable: DeviceOf gives the new device
	// when the renderer has made it.
	//
	// Flutter: no counterpart
	TagGPUDeviceLost = "GPUDeviceLost"
	// TagGPUInvalidDescriptor is the tag of the error of a constructor whose description cannot be made: a size
	// of 0, a format that a usage does not allow, an entry point that the shader lacks. The metadata "field" names
	// the field of the description and "value" the value it had.
	//
	// Flutter: no counterpart
	TagGPUInvalidDescriptor = "GPUInvalidDescriptor"
	// TagGPULimitExceeded is the tag of the error of a constructor whose description is larger than a limit
	// (Device.Limits). The metadata "field", "value" and "max" say which.
	//
	// Flutter: no counterpart
	TagGPULimitExceeded = "GPULimitExceeded"
	// TagGPUShaderCompileFailed is the tag of the error of a pipeline whose shader the driver refuses (the
	// backends make it; the messages are in its metadata).
	//
	// Flutter: no counterpart
	TagGPUShaderCompileFailed = "GPUShaderCompileFailed"
	// TagGPUOutOfMemory is the tag of the error of a call that needs more memory than the GPU or the system has
	// (the backends make it).
	//
	// Flutter: no counterpart
	TagGPUOutOfMemory = "GPUOutOfMemory"
	// TagGPUReadFailed is the tag of the error that Buffer.Read gives to its function when the bytes could not be
	// read back; the error of the backend is its cause.
	//
	// Flutter: no counterpart
	TagGPUReadFailed = "GPUReadFailed"
	// TagGPUWrongThread is the tag of the programmer error of a call from a goroutine that is not the UI thread.
	// Debug builds panic with it (SPEC §13).
	//
	// Flutter: no counterpart
	TagGPUWrongThread = "GPUWrongThread"
	// TagGPUEncodingInvalid is the tag of the programmer error of a misuse while encoding: a pass that was not
	// ended, a draw without a pipeline, a bind group of another layout, a draw past the end of a buffer, a command
	// buffer submitted twice. Debug builds panic with it; release builds skip the checks. The metadata "call" and
	// "what" say which call and which rule.
	//
	// Flutter: no counterpart
	TagGPUEncodingInvalid = "GPUEncodingInvalid"
	// TagGPUResourceReleased is the programmer error of using a resource after Release. Debug builds panic with
	// it; the metadata "kind" and "label" say which resource.
	//
	// Flutter: no counterpart
	TagGPUResourceReleased = "GPUResourceReleased"
	// TagGPUResourceLeaked is the tag of the warning that debug builds log, for each resource that was not
	// released, when the app ends. The attributes "kind", "label" and "created" say what it was and which call made
	// it.
	//
	// Flutter: no counterpart
	TagGPUResourceLeaked = "GPUResourceLeaked"
)

// IsDeviceLost says whether err is the loss of a GPU device: GPUDeviceLost, or the loss as a backend reports
// it (VulkanDeviceLost).
//
// Flutter: no counterpart
func IsDeviceLost(err error) bool {
	appErr, ok := errors.FromError(err)
	return ok && (appErr.Tag == TagGPUDeviceLost || appErr.Tag == explicit.TagVulkanDeviceLost)
}
