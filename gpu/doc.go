// Package gpu is the public GPU API of Glutter (SPEC §27.2): buffers, textures, samplers, WGSL shaders,
// render and compute pipelines, command encoders and readback over the device of the explicit renderer.
//
// The names follow WebGPU, in Go form. SDL 3's GPU API is the other design reference (D32).
//
// An app gets the device of the renderer that draws its windows with DeviceOf, or with OnDevice, which also
// tells when the renderer makes a new device after a loss. Everything made from a device is used on the UI thread
// (SPEC §13). Commands are recorded with a CommandEncoder, finished into a CommandBuffer and sent with
// Queue.Submit. The shaders are made by glutter-gen gpushaders (SPEC §27.3).
//
// Flutter: no counterpart
package gpu
