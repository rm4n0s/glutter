package metal

// The tags of the errors of the Metal backend (rm4n0s/errors, SPEC §14.4, §29.14). The exported ones are
// compared with by other packages.
const (
	// TagMetalDeviceUnavailable is the tag of the error of Open when the machine has no Metal device (a
	// virtual machine without GPU passthrough), the framework cannot be loaded, or the device is not one
	// that the backend can use. The engine's chooser moves on to the next renderer (SPEC §29.5).
	//
	// Flutter: no counterpart
	TagMetalDeviceUnavailable = "MetalDeviceUnavailable"
	// TagMetalShaderCompileFailed is the tag of the error of a shader that the Metal compiler refuses; the
	// metadata are the "shader" and the compiler's "output".
	//
	// Flutter: no counterpart
	TagMetalShaderCompileFailed = "MetalShaderCompileFailed"
	// TagMetalPipelineFailed is the tag of the error of a pipeline state that Metal cannot make; the
	// metadata are the "shader", the entry points and Metal's "output".
	//
	// Flutter: no counterpart
	TagMetalPipelineFailed = "MetalPipelineFailed"
	// TagMetalDrawableTimeout is the tag of the error of a layer that gave no drawable within its timeout
	// (the window server holds all of them, as a window that is not shown does): the frame is not drawn
	// and the caller tries again with the next one (SPEC §29.5).
	//
	// Flutter: no counterpart
	TagMetalDrawableTimeout = "MetalDrawableTimeout"
	// TagMetalCallFailed is the tag of an error of a call that Metal answers with an error object or with
	// no object: a resource that cannot be made, a command buffer that failed; the metadata are the "call"
	// and Metal's "output".
	//
	// Flutter: no counterpart
	TagMetalCallFailed = "MetalCallFailed"
	// TagMetalSwapchainFailed is the tag of the error of a swap chain that cannot be made: the surface is
	// not the layer of a view, or the layer refused a setting.
	//
	// Flutter: no counterpart
	TagMetalSwapchainFailed = "MetalSwapchainFailed"
	// TagMetalFrameOpen is the tag of the error of Begin when the frame before it was not submitted.
	//
	// Flutter: no counterpart
	TagMetalFrameOpen = "MetalFrameOpen"
	// TagMetalBindGroupInvalid is the tag of the error of a bind group that the pipeline does not accept: a
	// group or binding the shader lacks, an entry without its resource, a texture that is not sampled, a
	// uniform offset that is not aligned.
	//
	// Flutter: no counterpart
	TagMetalBindGroupInvalid = "MetalBindGroupInvalid"
	// TagMetalForeignResource is the tag of the panic of a resource that another device made.
	//
	// Flutter: no counterpart
	TagMetalForeignResource = "MetalForeignResource"
	// TagMetalUsage is the tag of the panic of a call that breaks a rule of the encoder (a copy inside a
	// pass, a draw without a pipeline, a rectangle outside a texture).
	//
	// Flutter: no counterpart
	TagMetalUsage = "MetalUsage"

	tagMetalBindingsInvalid = "MetalBindingsInvalid"
	tagGPUDeviceLost        = "GPUDeviceLost"
	tagGPUOutOfMemory       = "GPUOutOfMemory"
)
