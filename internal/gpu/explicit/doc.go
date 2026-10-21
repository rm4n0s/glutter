// Package explicit is the renderer for the explicit GPU APIs, Vulkan and WebGPU
// (D19): the Device interface that both implement, the design limits they share
// (SPEC §9.3), and the WGSL shaders with the SPIR-V that glutter-gen makes from
// them (D20).
//
// Flutter: Impeller's Vulkan and Metal backends
// (engine/src/flutter/impeller/renderer/backend).
package explicit

//go:generate go run ../../../cmd/glutter-gen shaders -dir .
