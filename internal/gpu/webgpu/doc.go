// Package webgpu is the WebGPU backend of the explicit renderer (SPEC §9.5): an
// explicit.Device over navigator.gpu, through syscall/js, so that internal/gpu/explicit draws on
// the web the way it draws on Vulkan.
//
// The browser's GPU API is asynchronous, and the UI goroutine never waits for a promise once the
// event listeners are installed (SPEC §23.3). So the adapter and the device are awaited by
// Acquire, which the engine calls while it starts, before the platform installs its listeners;
// what comes after is a callback that posts to the UI goroutine: a read-back that finishes later
// (ReadTexture), the loss of the device, and the new device that follows a loss.
//
// The decisions that need no browser are plain Go, tested in the normal run and under Node:
// the names the API uses for formats, usages, blend factors and vertex formats (mapping.go),
// the bind group layouts of a pipeline (layout.go), the supply of devices (supply.go), the loss
// of a device (loss.go) and the rows of a read-back (readback.go). The files that touch the
// browser carry the js tag; elsewhere Acquire fails with WebGPUUnavailable (unavailable.go).
//
// Flutter: Impeller's Metal and Vulkan backends are its counterparts; Flutter web uses CanvasKit
// or Skwasm (lib/web_ui/lib/src/engine/).
package webgpu
