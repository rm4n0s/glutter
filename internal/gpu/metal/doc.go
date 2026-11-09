// Package metal is the Metal backend of the explicit renderer (D33, SPEC §29.5): the fourth explicit.Device, after
// vulkan, webgpu and d3d12, for macOS. Every call into Metal is made through purego and the Objective-C runtime,
// with the helpers of internal/platform/darwin/objcx; there is no cgo.
//
// The shaders are the MSL that `glutter-gen shaders` and `gpushaders` make from the WGSL with naga
// (explicit.Shader.MSL, Metal Shading Language 2.4). The generator puts every resource at the index of its class
// in Metal's argument tables that the binding map gives (explicit.ShaderBinding.MetalIndex: the place of the
// binding among the bindings of the module of its class, in the order of group and binding), and the vertex
// buffers above the shader's buffers (explicit.MetalVertexBufferBase + slot). The library is compiled when the
// first pipeline needs it, with newLibraryWithSource:options:error:, which needs no Xcode on the user's Mac, and the
// pipelines are kept between runs as an MTLBinaryArchive in the shader cache directory (SPEC §9.7).
//
// The device is Open, which returns a Device: the default Metal device (the low-power one when
// GLUTTER_METAL_DEVICE=lowpower), one command queue, buffers and textures in private memory (shared for buffers on
// a device with unified memory), samplers, render and compute pipelines, bind groups that hold the resources by
// binding number and set them in the argument tables of the stages that read them, and an encoder that records a
// frame as one command buffer. The CPU never touches a buffer or a texture: a write is copied into staging memory
// (shared on unified memory, managed with didModifyRange: otherwise) and a blit puts it in place before the next
// pass, a read is a blit into a readback buffer that is delivered when the frame has finished, so that the order of
// the commands is the order of the calls. A frame that has reads waits for the GPU in Submit, as Direct3D 12's
// does; the others are handled at the next Begin or Submit, or by Settle.
//
// The swap chain (NewSwapchain) is a CAMetalLayer that covers the layer of the window's view: BGRA8Unorm, at most
// three drawables, display sync on, framebufferOnly unless Options.Readback asks for the contents, and a drawable
// that is asked for only when a frame is drawn. During a live resize the present is made in the window's
// transaction.
//
//   - errors.go, mapping.go, dirty.go, bindings.go and structs.go are plain Go, tested on any host: the tags of the
//     errors, the mapping of formats, filters, blends, vertex formats and actions to Metal's numbers, the merge of
//     the dirty ranges of a managed buffer, the binding tables of a pipeline, and the Go mirrors of Metal's
//     structures.
//   - objc_darwin.go, device_darwin.go, resources_darwin.go, pipeline_darwin.go, bindgroup_darwin.go,
//     encoder_darwin.go and swapchain_darwin.go call Metal. Their tests run on a Mac: the layout of the structures
//     and every constant against the SDK's headers, every MSL compiled by Metal's compiler with its reflection
//     compared with the binding map, the error routes, the swap chain, and, with the glutter_display tag and
//     MTL_DEBUG_LAYER=1, the scenes that every backend runs (devicetest) and the parity test, which draws every proof
//     scene through the explicit renderer and compares it with the software renderer.
//
// Flutter: no counterpart. Flutter's macOS embedder draws with Impeller on Metal.
package metal
